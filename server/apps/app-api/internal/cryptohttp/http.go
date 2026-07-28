package cryptohttp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/oublie6/awesome-zero-platform/server/apps/app-api/internal/svc"
	"github.com/oublie6/awesome-zero-platform/server/foundation/revealkeys"
	"github.com/zeromicro/go-zero/rest"
)

const keyPathPrefix = "/api/v1/crypto/reveal-keys/"

type manifestProvider interface {
	Current(context.Context) (revealkeys.Manifest, error)
	Lookup(context.Context, string) (revealkeys.Manifest, error)
}

type handler struct{ keys manifestProvider }

func Register(server *rest.Server, serviceContext *svc.ServiceContext) {
	h := handler{}
	if serviceContext != nil {
		h.keys = serviceContext.RevealKeys
	}
	server.AddRoutes([]rest.Route{
		{Method: http.MethodGet, Path: keyPathPrefix + "current", Handler: h.current},
		{Method: http.MethodGet, Path: keyPathPrefix + ":keyId", Handler: h.byID},
	})
}

func (h handler) current(w http.ResponseWriter, r *http.Request) {
	if h.keys == nil {
		writeError(w, http.StatusServiceUnavailable, "REVEAL_KEYS_NOT_CONFIGURED", "reveal key publication is unavailable")
		return
	}
	manifest, err := h.keys.Current(r.Context())
	if err != nil {
		writeLookupError(w, err)
		return
	}
	writeManifest(w, r, manifest, "public, max-age=300, must-revalidate")
}

func (h handler) byID(w http.ResponseWriter, r *http.Request) {
	if h.keys == nil {
		writeError(w, http.StatusServiceUnavailable, "REVEAL_KEYS_NOT_CONFIGURED", "reveal key publication is unavailable")
		return
	}
	keyID := strings.TrimPrefix(r.URL.Path, keyPathPrefix)
	if keyID == "" || keyID == "current" || strings.Contains(keyID, "/") {
		writeError(w, http.StatusBadRequest, "REVEAL_KEY_ID_INVALID", "reveal key id is invalid")
		return
	}
	manifest, err := h.keys.Lookup(r.Context(), keyID)
	if err != nil {
		writeLookupError(w, err)
		return
	}
	writeManifest(w, r, manifest, "public, max-age=60, must-revalidate")
}

func writeManifest(w http.ResponseWriter, r *http.Request, manifest revealkeys.Manifest, cacheControl string) {
	encoded, err := json.Marshal(manifest)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "REVEAL_KEY_RESPONSE_FAILED", "reveal key response failed")
		return
	}
	digest := sha256.Sum256(encoded)
	etag := `"` + hex.EncodeToString(digest[:]) + `"`
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", cacheControl)
	w.Header().Set("ETag", etag)
	w.Header().Set("Vary", "Accept-Encoding")
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(encoded)
}

func writeLookupError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, revealkeys.ErrKeyUnavailable):
		writeError(w, http.StatusNotFound, "REVEAL_KEY_NOT_FOUND", "reveal key is unavailable")
	case errors.Is(err, revealkeys.ErrKeyExpired), errors.Is(err, revealkeys.ErrKeyRevoked):
		writeError(w, http.StatusGone, "REVEAL_KEY_RETIRED", "reveal key is no longer available")
	default:
		writeError(w, http.StatusServiceUnavailable, "REVEAL_KEY_LOOKUP_FAILED", "reveal key lookup failed")
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"code": code, "message": message})
}
