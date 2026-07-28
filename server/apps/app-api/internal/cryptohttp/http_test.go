package cryptohttp

import (
	"crypto/ed25519"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/foundation/revealkeys"
)

func TestCurrentManifestEndpointUsesETagAndBoundedCaching(t *testing.T) {
	registry := httpRegistry(t)
	h := handler{keys: registry}
	request := httptest.NewRequest(http.MethodGet, keyPathPrefix+"current", nil)
	response := httptest.NewRecorder()
	h.current(response, request)
	if response.Code != http.StatusOK || response.Header().Get("ETag") == "" || response.Header().Get("Cache-Control") != "public, max-age=300, must-revalidate" {
		t.Fatalf("status=%d headers=%v body=%s", response.Code, response.Header(), response.Body.String())
	}
	var manifest revealkeys.Manifest
	if err := json.Unmarshal(response.Body.Bytes(), &manifest); err != nil || manifest.KeyID != "active" || manifest.Signature == "" {
		t.Fatalf("manifest=%#v error=%v", manifest, err)
	}
	conditional := httptest.NewRequest(http.MethodGet, keyPathPrefix+"current", nil)
	conditional.Header.Set("If-None-Match", response.Header().Get("ETag"))
	notModified := httptest.NewRecorder()
	h.current(notModified, conditional)
	if notModified.Code != http.StatusNotModified || notModified.Body.Len() != 0 {
		t.Fatalf("status=%d body=%s", notModified.Code, notModified.Body.String())
	}
}

func TestHistoricalLookupAndUnavailableErrors(t *testing.T) {
	h := handler{keys: httpRegistry(t)}
	request := httptest.NewRequest(http.MethodGet, keyPathPrefix+"retiring", nil)
	response := httptest.NewRecorder()
	h.byID(response, request)
	if response.Code != http.StatusOK || response.Header().Get("Cache-Control") != "public, max-age=60, must-revalidate" {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	missing := httptest.NewRecorder()
	h.byID(missing, httptest.NewRequest(http.MethodGet, keyPathPrefix+"missing", nil))
	if missing.Code != http.StatusNotFound || missing.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("status=%d body=%s", missing.Code, missing.Body.String())
	}
}

func TestUnconfiguredEndpointFailsClosed(t *testing.T) {
	response := httptest.NewRecorder()
	handler{}.current(response, httptest.NewRequest(http.MethodGet, keyPathPrefix+"current", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d", response.Code)
	}
}

func httpRegistry(t *testing.T) *revealkeys.Registry {
	t.Helper()
	now := time.Date(2026, 7, 28, 4, 0, 0, 0, time.UTC)
	seed := make([]byte, ed25519.SeedSize)
	signing := ed25519.NewKeyFromSeed(seed)
	active := revealkeys.Record{ManifestVersion: 2, KeyID: "active", PublicKey: httpBytes(1), PrivateKey: httpBytes(2), NotBefore: now.Add(-time.Hour), ExpiresAt: now.Add(24 * time.Hour), Status: revealkeys.StatusActive}
	retiring := revealkeys.Record{ManifestVersion: 1, KeyID: "retiring", PublicKey: httpBytes(3), PrivateKey: httpBytes(4), NotBefore: now.Add(-24 * time.Hour), ExpiresAt: now.Add(24 * time.Hour), Status: revealkeys.StatusRetiring, RetiringAt: now.Add(-time.Hour), RetireAfter: now.Add(time.Hour)}
	registry, err := revealkeys.New([]revealkeys.Record{retiring, active}, "active", "root", signing, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	return registry
}

func httpBytes(value byte) []byte {
	result := make([]byte, 32)
	for index := range result {
		result[index] = value
	}
	return result
}
