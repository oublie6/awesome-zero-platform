package bootstrap

import (
	"context"
	"encoding/hex"
	"fmt"
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
)

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
	seeds, err := ddzruntime.NewSeedVault(nil)
	if err != nil {
		return doudizhuComponents{}, err
	}
	liveHands, err := ddzruntime.NewLiveHandCoordinator(seeds, directory)
	if err != nil {
		return doudizhuComponents{}, err
	}
	plan := domain.BeaconPlan{Provider: cfg.Doudizhu.BeaconProvider, Round: cfg.Doudizhu.BeaconRound}
	setups, err := ddzruntime.NewSeededHandSetupProvider(seeds, keys, clock, plan)
	if err != nil {
		return doudizhuComponents{}, err
	}
	proofs, err := ddzruntime.NewHMACBeaconProofVerifier([]byte(cfg.Doudizhu.BeaconProofSecret))
	if err != nil {
		return doudizhuComponents{}, err
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
	contributionKey, err := hex.DecodeString(cfg.Doudizhu.ContributionKeyHex)
	if err != nil {
		return doudizhuComponents{}, err
	}
	keyring, err := protection.NewStaticKeyring(cfg.Doudizhu.ContributionKeyID, map[string][]byte{cfg.Doudizhu.ContributionKeyID: contributionKey})
	clear(contributionKey)
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
	apiConfig := doudizhuapi.DefaultConfig()
	apiConfig.CommandTTL = cfg.Doudizhu.CommandTTL
	apiConfig.ReplayTTL = cfg.Doudizhu.ReplayTTL
	apiConfig.ReplayEntries = cfg.Doudizhu.ReplayEntries
	dispatcher, err := doudizhuapi.NewDispatcher(service, evidence, supervisor, clock, apiConfig)
	if err != nil {
		return doudizhuComponents{}, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for ctx.Err() == nil {
			if err := supervisor.Run(ctx); err != nil && ctx.Err() == nil {
				time.Sleep(cfg.Doudizhu.SweepInterval)
			}
		}
	}()
	return doudizhuComponents{dispatcher: dispatcher, audience: store, cancel: cancel}, nil
}
