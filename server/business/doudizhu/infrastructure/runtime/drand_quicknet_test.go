package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cloudflare/circl/sign/bls"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

func TestDrandQuicknetValidatesIndependentBLSSignatureAndRejectsTampering(t *testing.T) {
	private, err := bls.KeyGen[bls.G2]([]byte(strings.Repeat("k", 32)), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	pub, err := private.PublicKey().MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	provider, err := NewDrandQuicknet(hex.EncodeToString(pub))
	if err != nil {
		t.Fatal(err)
	}
	plan := domain.BeaconPlan{Provider: DrandQuicknetProvider, Round: "1234567"}
	var round [8]byte
	binary.BigEndian.PutUint64(round[:], 1234567)
	msg := sha256.Sum256(round[:])
	sig := bls.Sign(private, msg[:])
	if len(sig) != 48 {
		t.Fatalf("unexpected G1 signature size %d", len(sig))
	}
	digest := sha256.Sum256(sig)
	value := domain.BeaconValue{Provider: plan.Provider, Round: plan.Round,
		Digest: domain.BeaconDigest(digest), ProofRef: drandQuicknetProofPrefix + hex.EncodeToString(sig)}
	if err := provider.Verify(context.Background(), plan, value); err != nil {
		t.Fatal(err)
	}
	changed := value
	changed.Round = "1234568"
	if !errors.Is(provider.Verify(context.Background(), plan, changed), ErrUnverifiedBeacon) {
		t.Fatal("accepted changed round")
	}
	changed = value
	changed.Digest[0] ^= 0x7f
	if !errors.Is(provider.Verify(context.Background(), plan, changed), ErrUnverifiedBeacon) {
		t.Fatal("accepted changed digest")
	}
	changed = value
	changed.ProofRef = drandQuicknetProofPrefix + strings.Repeat("f", 96)
	if !errors.Is(provider.Verify(context.Background(), plan, changed), ErrUnverifiedBeacon) {
		t.Fatal("accepted forged BLS signature")
	}
	if _, err := NewDrandQuicknet("invalid"); !errors.Is(err, ErrUnverifiedBeacon) {
		t.Fatal("accepted unpinned root")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/1234567") {
			t.Errorf("incorrect fixed round URL %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"round": 1234567, "signature": hex.EncodeToString(sig)})
	}))
	defer server.Close()
	// Keep host pinned in production; test response transport without changing
	// the requested destination or the production verifier.
	provider.client.Transport = redirectTransport{url: server.URL, base: http.DefaultTransport}
	fetched, err := provider.Fetch(context.Background(), plan)
	if err != nil || fetched != value {
		t.Fatalf("fetch=%+v err=%v", fetched, err)
	}
}

type redirectTransport struct {
	url  string
	base http.RoundTripper
}

func (rt redirectTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	copy := r.Clone(r.Context())
	copy.URL.Scheme = "http"
	copy.URL.Host = strings.TrimPrefix(rt.url, "http://")
	return rt.base.RoundTrip(copy)
}
func TestQuicknetRoundIsAlwaysFutureAndLockedBeforeCommit(t *testing.T) {
	now := time.Unix(DrandQuicknetGenesisUnix+3000, 0).UTC()
	round, err := DrandFutureRound(now)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := quicknetRound(round)
	if err != nil {
		t.Fatal(err)
	}
	emitted := DrandQuicknetGenesisUnix + int64(parsed-1)*3
	if emitted <= now.Unix()+60 {
		t.Fatalf("round %d not strictly future at %d", parsed, now.Unix())
	}
	for _, value := range []string{"", "0", "01", "a", "-1", "99999999999999999999999"} {
		if _, err := quicknetRound(value); err == nil {
			t.Fatalf("invalid quicknet round accepted: %s", value)
		}
	}
}

func TestPublishedQuicknetRound1000AgainstPinnedPublicKey(t *testing.T) {
	// Captured independent public-chain verification vector; unlike an
	// internally generated key, it also detects a drand/CIRCL scheme mismatch.
	const publicKey = "83cf0f2896adee7eb8b5f01fcad3912212c437e0073e911fb90022d3e760183c8c4b450b6a0a6c3ac6a5776a2d1064510d1fec758c921cc22b0e17e63aaf4bcb5ed66304de9cf809bd274ca73bab4af5a6e9c76a4bc09e76eae8991ef5ece45a"
	const signature = "b44679b9a59af2ec876b1a6b1ad52ea9b1615fc3982b19576350f93447cb1125e342b73a8dd2bacbe47e4b6b63ed5e39"
	p, err := NewDrandQuicknet(publicKey)
	if err != nil {
		t.Fatal(err)
	}
	sig, err := hex.DecodeString(signature)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(sig)
	plan := domain.BeaconPlan{Provider: DrandQuicknetProvider, Round: "1000"}
	value := domain.BeaconValue{Provider: plan.Provider, Round: plan.Round,
		Digest: domain.BeaconDigest(digest), ProofRef: drandQuicknetProofPrefix + signature}
	if err := p.Verify(context.Background(), plan, value); err != nil {
		t.Fatalf("published drand Quicknet signature rejected: %v", err)
	}
}
