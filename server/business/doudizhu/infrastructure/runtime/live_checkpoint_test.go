package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/livehand"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/randomizedsetup"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/infrastructure/mysqlstore"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

func newRecoveryCheckpointGame(t *testing.T) *livehand.Game {
	t.Helper()
	var vector struct {
		HandID           string `json:"handId"`
		ServerSeed       string `json:"serverSeed"`
		ServerCommitment string `json:"serverCommitment"`
		Contributions    []struct {
			Seat       uint8  `json:"seat"`
			Digest     string `json:"digest"`
			Commitment string `json:"commitment"`
		} `json:"contributions"`
		BeaconProvider        string `json:"beaconProvider"`
		BeaconRound           string `json:"beaconRound"`
		BeaconDigest          string `json:"beaconDigest"`
		BeaconProofRef        string `json:"beaconProofRef"`
		RevealKeyID           string `json:"revealKeyId"`
		RevealPublicKeySHA256 string `json:"revealPublicKeySha256"`
	}
	data, err := os.ReadFile("../../domain/carddeck/testdata/golden-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &vector); err != nil {
		t.Fatal(err)
	}
	decode := func(hexString string) gamecore.Digest {
		t.Helper()
		var out gamecore.Digest
		b, err := hex.DecodeString(hexString)
		if err != nil || len(b) != len(out) {
			t.Fatalf("invalid golden digest: %v", err)
		}
		copy(out[:], b)
		return out
	}
	seats := [3]domain.HandSeat{
		{Seat: 1, AccountID: "player-1"}, {Seat: 2, AccountID: "player-2"}, {Seat: 3, AccountID: "player-3"},
	}
	commitment := decode(vector.ServerCommitment)
	hand, _, err := domain.NewHand(domain.HandID(vector.HandID), "checkpoint-room", seats,
		domain.ServerCommitment(commitment), vector.RevealKeyID,
		domain.BeaconPlan{Provider: vector.BeaconProvider, Round: vector.BeaconRound},
		domain.RevealKeyBinding{PublicKeySHA256: domain.RevealPublicKeyHash(decode(vector.RevealPublicKeySHA256)), BoundAt: time.Unix(0, 0).UTC()})
	if err != nil {
		t.Fatal(err)
	}
	material := gamecore.FairnessMaterial{
		Descriptor: randomizedsetup.Descriptor(), InstanceID: gamecore.InstanceID(vector.HandID),
		ServerSeed:       gamecore.Seed(decode(vector.ServerSeed)),
		ServerCommitment: commitment, Participants: make([]gamecore.ParticipantFairness, 3),
		Beacon: gamecore.BeaconEvidence{Provider: vector.BeaconProvider, Round: vector.BeaconRound,
			Digest: decode(vector.BeaconDigest), ProofRef: vector.BeaconProofRef},
		RevealKey: gamecore.RevealKeyAudit{KeyID: vector.RevealKeyID, PublicKeySHA256: decode(vector.RevealPublicKeySHA256)},
	}
	for i, v := range vector.Contributions {
		digest := decode(v.Digest)
		committed := domain.ComputeClientCommitment(domain.HandID(vector.HandID), domain.Seat(v.Seat),
			domain.ContributionDigest(digest))
		if committed != domain.Commitment(decode(v.Commitment)) {
			t.Fatal("golden commitment mismatch")
		}
		material.Participants[i] = gamecore.ParticipantFairness{
			Position: v.Seat, Contribution: digest, Commitment: decode(v.Commitment)}
		if _, err := hand.SubmitCommit(seats[i].AccountID, committed, hand.Snapshot().Version); err != nil {
			t.Fatal(err)
		}
	}
	for i, v := range vector.Contributions {
		digest := decode(v.Digest)
		if _, err := hand.SubmitReveal(seats[i].AccountID, domain.ContributionDigest(digest), "checkpoint-contribution", hand.Snapshot().Version); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := hand.LockPublicBeacon(domain.BeaconValue{
		Provider: vector.BeaconProvider, Round: vector.BeaconRound,
		Digest: domain.BeaconDigest(decode(vector.BeaconDigest)), ProofRef: vector.BeaconProofRef,
	}, hand.Snapshot().Version); err != nil {
		t.Fatal(err)
	}
	// Binding the digest is part of the protected genesis and must match
	// the fairness material used to generate the deterministic deck.
	snapshot := hand.Snapshot()
	module := randomizedsetup.NewModule()
	artifact, err := module.GenerateSetup(material)
	if err != nil {
		t.Fatal(err)
	}
	if err := module.VerifySetup(material, artifact); err != nil {
		t.Fatal(err)
	}
	game, err := livehand.New(snapshot, material, artifact)
	if err != nil {
		t.Fatal(err)
	}
	return game
}

func TestRecoveryCheckpointRequiresExactCommittedStateAndPrivateViews(t *testing.T) {
	original := newRecoveryCheckpointGame(t)
	genesis := newRecoveryCheckpointGame(t)
	start, err := original.View(gamecore.ViewRequest{PublicOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	var pub livehand.PublicView
	if err := json.Unmarshal(start.Payload, &pub); err != nil {
		t.Fatal(err)
	}
	var first uint8 = pub.Bidding.FirstBidder
	command := gamecore.Command{ActorPosition: first, ExpectedVersion: 1,
		Payload: []byte(`{"v":"doudizhu-live-bid-command-v1","score":3}`)}
	outcome, err := original.Apply(command)
	if err != nil || outcome.Version != 2 {
		t.Fatalf("bid %v %+v", err, outcome)
	}
	pubHash, privHashes, err := checkpointViewDigests(original)
	if err != nil {
		t.Fatal(err)
	}
	checkpoint := recoveryCheckpoint{Version: liveRecoveryCheckpointV1,
		HandID: domain.HandID(original.InstanceID()), StateVersion: 2,
		PublicDigest: pubHash, PrivateDigests: privHashes,
		Receipts: []checkpointReceipt{{CommandID: "bid-id", Actor: domain.AccountID("player-" + string(rune('0'+first))),
			Command: command, Outcome: outcome}}}
	raw, err := json.Marshal(checkpoint)
	if err != nil {
		t.Fatal(err)
	}
	digestInput, err := json.Marshal(outcome)
	if err != nil {
		t.Fatal(err)
	}
	record := mysqlstore.LiveAction{HandID: checkpoint.HandID, Version: 2, Epoch: 2,
		CommandID: "bid-id", Actor: checkpoint.Receipts[0].Actor,
		ActorPosition: first, ExpectedVersion: 1, Payload: command.Payload, ResultDigest: sha256.Sum256(digestInput)}
	recovered, err := restoreRecoveryCheckpoint(genesis, mysqlstore.LiveCheckpoint{Version: 2, Payload: raw}, []mysqlstore.LiveAction{record})
	if err != nil || recovered["bid-id"].outcome.Version != 2 {
		t.Fatalf("valid checkpoint rejected: %v %#v", err, recovered)
	}
	if after, _, digestErr := checkpointViewDigests(genesis); digestErr != nil || after != pubHash {
		t.Fatal("restored public state drift")
	}
	tampered := checkpoint
	tampered.PrivateDigests[0][0] ^= 1
	bad, _ := json.Marshal(tampered)
	if _, err := restoreRecoveryCheckpoint(newRecoveryCheckpointGame(t), mysqlstore.LiveCheckpoint{Version: 2, Payload: bad}, []mysqlstore.LiveAction{record}); !errors.Is(err, gamecore.ErrVerificationFailed) {
		t.Fatalf("forged private state was accepted: %v", err)
	}
	forged := record
	forged.ResultDigest[0] ^= 1
	if _, err := restoreRecoveryCheckpoint(newRecoveryCheckpointGame(t), mysqlstore.LiveCheckpoint{Version: 2, Payload: raw}, []mysqlstore.LiveAction{forged}); !errors.Is(err, gamecore.ErrVerificationFailed) {
		t.Fatalf("forged committed result was accepted: %v", err)
	}
}

func TestThreePlayerCheckpointRestoresFullGameAndFinalEvidence(t *testing.T) {
	live := newRecoveryCheckpointGame(t)
	restored := newRecoveryCheckpointGame(t)
	var journal []mysqlstore.LiveAction
	var receipts []checkpointReceipt
	var sealedCheckpoint []byte
	var completed gamecore.CommandOutcome

	for {
		publicView, err := live.View(gamecore.ViewRequest{PublicOnly: true})
		if err != nil {
			t.Fatal(err)
		}
		var state livehand.PublicView
		if err := json.Unmarshal(publicView.Payload, &state); err != nil {
			t.Fatal(err)
		}
		var seat uint8
		var payload []byte
		switch state.Phase {
		case livehand.PhaseBidding:
			seat = state.Bidding.FirstBidder
			payload, err = json.Marshal(livehand.BidCommand{Version: livehand.BidCommandVersion, Score: 3})
		case livehand.PhasePlaying:
			seat = state.PlayingSeat
			if seat == state.LandlordSeat {
				view, viewErr := live.View(gamecore.ViewRequest{ViewerPosition: seat})
				if viewErr != nil {
					t.Fatal(viewErr)
				}
				var own livehand.PrivateView
				if err := json.Unmarshal(view.Payload, &own); err != nil {
					t.Fatal(err)
				}
				if len(own.Cards) == 0 {
					t.Fatal("landlord ran out of cards before final action")
				}
				payload, err = json.Marshal(livehand.PlayCommand{Version: livehand.PlayCommandVersion, Cards: []string{own.Cards[0]}})
			} else {
				payload, err = json.Marshal(livehand.PassCommand{Version: livehand.PassCommandVersion})
			}
		default:
			t.Fatalf("unexpected pre-terminal phase %s", state.Phase)
		}
		if err != nil {
			t.Fatal(err)
		}
		if seat < 1 || seat > 3 {
			t.Fatalf("invalid active seat %d", seat)
		}
		command := gamecore.Command{ActorPosition: seat, ExpectedVersion: uint64(len(journal) + 1), Payload: payload}
		outcome, err := live.Apply(command)
		if err != nil {
			t.Fatalf("action %d: %v", len(journal), err)
		}
		id := fmt.Sprintf("three-player-command-%d", len(journal)+1)
		actor := domain.AccountID(fmt.Sprintf("player-%d", seat))
		encoded, err := json.Marshal(outcome)
		if err != nil {
			t.Fatal(err)
		}
		journal = append(journal, mysqlstore.LiveAction{
			HandID: domain.HandID(live.InstanceID()), Version: outcome.Version, Epoch: 3,
			CommandID: id, Actor: actor, ActorPosition: seat,
			ExpectedVersion: command.ExpectedVersion, Payload: payload,
			ResultDigest: sha256.Sum256(encoded),
		})
		receipts = append(receipts, checkpointReceipt{CommandID: id, Actor: actor, Command: command, Outcome: outcome})
		if outcome.Version == 33 {
			pubHash, privHashes, err := checkpointViewDigests(live)
			if err != nil {
				t.Fatal(err)
			}
			state := recoveryCheckpoint{Version: liveRecoveryCheckpointV1,
				HandID: domain.HandID(live.InstanceID()), StateVersion: outcome.Version,
				PublicDigest: pubHash, PrivateDigests: privHashes,
				Receipts: append([]checkpointReceipt(nil), receipts...)}
			sealedCheckpoint, err = json.Marshal(state)
			if err != nil {
				t.Fatal(err)
			}
		}
		if outcome.Terminal {
			completed = outcome
			break
		}
		if len(journal) > 80 {
			t.Fatal("three-player game never reached settlement")
		}
	}
	if completed.Version != 60 || len(completed.FinalPayload) == 0 || len(sealedCheckpoint) == 0 {
		t.Fatalf("game/backup incomplete: version=%d actions=%d checkpoint=%d", completed.Version, len(journal), len(sealedCheckpoint))
	}
	recoveredReceipts, err := restoreRecoveryCheckpoint(restored,
		mysqlstore.LiveCheckpoint{Version: 33, Payload: sealedCheckpoint}, journal)
	if err != nil || len(recoveredReceipts) != 32 {
		t.Fatalf("checkpoint restore error %v, receipts=%d", err, len(recoveredReceipts))
	}
	for _, entry := range journal[32:] {
		outcome, err := restored.Apply(gamecore.Command{ActorPosition: entry.ActorPosition,
			ExpectedVersion: entry.ExpectedVersion, Payload: entry.Payload})
		if err != nil {
			t.Fatalf("suffix replay version %d: %v", entry.Version, err)
		}
		encoded, err := json.Marshal(outcome)
		if err != nil || sha256.Sum256(encoded) != entry.ResultDigest {
			t.Fatalf("suffix result divergence at version %d: %v", entry.Version, err)
		}
		if entry.Version == completed.Version && string(outcome.FinalPayload) != string(completed.FinalPayload) {
			t.Fatal("checkpoint recovery changed authoritative terminal evidence")
		}
	}
	if restored.Phase() != livehand.PhaseCompleted {
		t.Fatal("replayed terminal hand was not completed")
	}
}
