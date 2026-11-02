package bootstrap

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/apps/app-api/internal/config"
	"github.com/oublie6/awesome-zero-platform/server/apps/app-api/internal/doudizhuapi"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application/lifecycle"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/infrastructure/mysqlstore"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/infrastructure/protection"
	ddzruntime "github.com/oublie6/awesome-zero-platform/server/business/doudizhu/infrastructure/runtime"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/infrastructure/securetransport"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/infrastructure/textnormalization"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore/infrastructure/mysqlarchive"
	"github.com/oublie6/awesome-zero-platform/server/foundation/database"
	"github.com/oublie6/awesome-zero-platform/server/foundation/revealkeys"
	"github.com/oublie6/awesome-zero-platform/server/foundation/secureenvelope"
	"github.com/oublie6/awesome-zero-platform/server/platform/authn"
	"github.com/oublie6/awesome-zero-platform/server/platform/realtime"
	"github.com/zeromicro/go-zero/rest"
)

func doudizhuModule(
	cfg config.Config,
	mysql database.Handle,
	keys *revealkeys.Registry,
	server *rest.Server,
	authentication *authn.Service,
	hub *realtime.Hub,
) moduleSpec {
	return moduleSpec{
		name:    "doudizhu",
		enabled: cfg.Doudizhu.Enabled,
		start: func() (context.CancelFunc, error) {
			components, err := initializeDoudizhu(cfg, mysql, keys)
			if err != nil {
				return nil, err
			}
			if components.dispatcher == nil || components.cancel == nil {
				return nil, fmt.Errorf("enabled Doudizhu module is incomplete")
			}
			doudizhuapi.Register(server, authentication, components.dispatcher)
			if hub != nil {
				if _, err := doudizhuapi.RegisterRealtime(hub, components.dispatcher, components.audience); err != nil {
					components.cancel()
					return nil, fmt.Errorf("register Doudizhu realtime handlers: %w", err)
				}
			}
			return components.cancel, nil
		},
	}
}

type doudizhuComponents struct {
	dispatcher *doudizhuapi.Dispatcher
	audience   doudizhuapi.HandAudience
	cancel     context.CancelFunc
}

func initializeDoudizhu(cfg config.Config, mysql database.Handle, keys *revealkeys.Registry) (doudizhuComponents, error) {
	if !cfg.Doudizhu.Enabled {
		return doudizhuComponents{}, nil
	}
	if mysql == nil || mysql.DB() == nil || keys == nil {
		return doudizhuComponents{}, fmt.Errorf("Doudizhu dependencies are unavailable")
	}
	store, err := mysqlstore.New(mysql)
	if err != nil {
		return doudizhuComponents{}, err
	}
	clock := ddzruntime.Clock{}
	ids := ddzruntime.UUIDv7Generator{}
	archive, err := mysqlarchive.New(mysql.DB(), clock)
	if err != nil {
		return doudizhuComponents{}, err
	}
	directory, err := gamecore.NewLiveDirectory(archive)
	if err != nil {
		return doudizhuComponents{}, err
	}
	contributionKey, err := hex.DecodeString(cfg.Doudizhu.ContributionKeyHex)
	if err != nil {
		return doudizhuComponents{}, err
	}
	defer clear(contributionKey)
	// A bounded set of prior encryption roots allows existing acknowledged
	// hands and contribution records to remain decryptable during rotation.
	allRoots := map[string][]byte{cfg.Doudizhu.ContributionKeyID: contributionKey}
	if cfg.Doudizhu.PreviousKeysJSON != "" {
		var previous map[string]string
		if err := json.Unmarshal([]byte(cfg.Doudizhu.PreviousKeysJSON), &previous); err != nil {
			return doudizhuComponents{}, fmt.Errorf("decode previous Doudizhu keyring: %w", err)
		}
		if len(previous) > 7 {
			return doudizhuComponents{}, fmt.Errorf("too many previous Doudizhu encryption keys")
		}
		for id, encoded := range previous {
			if id == "" || len(id) > 128 || id == cfg.Doudizhu.ContributionKeyID {
				return doudizhuComponents{}, fmt.Errorf("invalid previous Doudizhu key ID")
			}
			old, err := hex.DecodeString(encoded)
			if err != nil || len(old) != 32 {
				return doudizhuComponents{}, fmt.Errorf("invalid previous Doudizhu 256-bit key")
			}
			allRoots[id] = old
		}
	}
	defer func() {
		for _, key := range allRoots {
			clear(key)
		}
	}()
	seedStore, err := mysqlstore.NewProtectedSeedStoreKeyring(mysql.DB(), cfg.Doudizhu.ContributionKeyID, allRoots)
	if err != nil {
		return doudizhuComponents{}, err
	}
	seeds, err := ddzruntime.NewPersistentSeedVault(nil, seedStore)
	if err != nil {
		return doudizhuComponents{}, err
	}
	journal, err := mysqlstore.NewLiveJournal(mysql.DB())
	if err != nil {
		return doudizhuComponents{}, err
	}
	if err := journal.ConfigureCheckpointKeyring(cfg.Doudizhu.ContributionKeyID, allRoots); err != nil {
		return doudizhuComponents{}, fmt.Errorf("configure protected live checkpoints: %w", err)
	}
	liveHands, err := ddzruntime.NewDurableLiveHandCoordinator(seeds, directory, journal)
	if err != nil {
		return doudizhuComponents{}, err
	}
	if err := liveHands.RecoverActive(context.Background(), store.LoadHand); err != nil {
		return doudizhuComponents{}, fmt.Errorf("recover durable Doudizhu live hands: %w", err)
	}
	plan := domain.BeaconPlan{Provider: cfg.Doudizhu.BeaconProvider, Round: cfg.Doudizhu.BeaconRound}
	setups, err := ddzruntime.NewSeededHandSetupProvider(seeds, keys, clock, plan)
	if err != nil {
		return doudizhuComponents{}, err
	}
	var proofs ddzruntime.BeaconProofVerifier
	var drand *ddzruntime.DrandQuicknet
	if cfg.Doudizhu.BeaconProvider == ddzruntime.DrandQuicknetProvider {
		if cfg.Doudizhu.BeaconRound != "auto" {
			return doudizhuComponents{}, fmt.Errorf("drand Quicknet must pin an auto-selected future round")
		}
		drand, err = ddzruntime.NewDrandQuicknet(cfg.Doudizhu.BeaconPublicKeyHex)
		if err != nil {
			return doudizhuComponents{}, err
		}
		proofs = drand
	} else {
		// Only legacy or deliberately isolated test deployments may use HMAC.
		// It is never considered an independently verifiable public beacon.
		proofs, err = ddzruntime.NewHMACBeaconProofVerifier([]byte(cfg.Doudizhu.BeaconProofSecret))
		if err != nil {
			return doudizhuComponents{}, err
		}
	}
	beacons, err := ddzruntime.NewBeaconVerifier(proofs)
	if err != nil {
		return doudizhuComponents{}, err
	}
	coreOpener, err := secureenvelope.NewOpener(keys)
	if err != nil {
		return doudizhuComponents{}, err
	}
	opener, err := securetransport.New(coreOpener, keys)
	if err != nil {
		return doudizhuComponents{}, err
	}
	keyring, err := protection.NewStaticKeyring(cfg.Doudizhu.ContributionKeyID, allRoots)
	if err != nil {
		return doudizhuComponents{}, err
	}
	protector, err := protection.New(keyring)
	if err != nil {
		return doudizhuComponents{}, err
	}
	service, err := application.NewService(store, clock, ids, setups, beacons, liveHands, opener, protector, textnormalization.NFKC{}, application.DefaultConfig())
	if err != nil {
		return doudizhuComponents{}, err
	}
	evidence, err := application.NewFinalEvidenceService(store, archive, application.DoudizhuFinalRecordVerifier{})
	if err != nil {
		return doudizhuComponents{}, err
	}
	lifecycleConfig := lifecycle.DefaultConfig()
	lifecycleConfig.BiddingTimeout = cfg.Doudizhu.BiddingTimeout
	lifecycleConfig.PlayingTimeout = cfg.Doudizhu.PlayingTimeout
	lifecycleConfig.SweepInterval = cfg.Doudizhu.SweepInterval
	lifecycleConfig.CommandTTL = cfg.Doudizhu.CommandTTL
	supervisor, err := lifecycle.New(store, service, clock, ids, lifecycleConfig)
	if err != nil {
		return doudizhuComponents{}, err
	}
	if err := supervisor.ReconcileRecovered(context.Background(), liveHands.RecoveredPhases()); err != nil {
		return doudizhuComponents{}, fmt.Errorf("reconcile recovered Doudizhu lifecycle: %w", err)
	}
	apiConfig := doudizhuapi.DefaultConfig()
	apiConfig.CommandTTL = cfg.Doudizhu.CommandTTL
	apiConfig.ReplayTTL = cfg.Doudizhu.ReplayTTL
	apiConfig.ReplayEntries = cfg.Doudizhu.ReplayEntries
	dispatcher, err := doudizhuapi.NewDispatcher(service, evidence, supervisor, clock, apiConfig)
	if err != nil {
		return doudizhuComponents{}, err
	}
	if err := dispatcher.ConfigureRevealTickets(contributionKey); err != nil {
		return doudizhuComponents{}, fmt.Errorf("configure bound reveal AAD: %w", err)
	}
	var autoDealer *fairAutoDealer
	if drand != nil {
		autoDealer, err = newFairAutoDealer(store, service, supervisor, drand, ids, clock, cfg.Doudizhu.CommandTTL)
		if err != nil {
			return doudizhuComponents{}, err
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	if autoDealer != nil {
		go func() {
			ticker := time.NewTicker(cfg.Doudizhu.SweepInterval)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					scanCtx, done := context.WithTimeout(ctx, 10*time.Second)
					_, processErr := autoDealer.ProcessOnce(scanCtx)
					done()
					if processErr != nil && ctx.Err() == nil {
						log.Printf("fair drand auto-deal retry after verification/transaction error: %v", processErr)
					}
				}
			}
		}()
	}
	go func() {
		for ctx.Err() == nil {
			if err := supervisor.Run(ctx); err != nil && ctx.Err() == nil {
				time.Sleep(cfg.Doudizhu.SweepInterval)
			}
		}
	}()
	return doudizhuComponents{dispatcher: dispatcher, audience: store, cancel: cancel}, nil
}
