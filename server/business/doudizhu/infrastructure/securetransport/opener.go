package securetransport

import (
	"context"
	"fmt"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application"
	"github.com/oublie6/awesome-zero-platform/server/foundation/revealkeys"
	"github.com/oublie6/awesome-zero-platform/server/foundation/secureenvelope"
)

type CoreOpener interface {
	Open(context.Context, secureenvelope.Envelope, []byte) ([]byte, error)
}

type KeyAuthorizer interface {
	AuthorizeBound(context.Context, revealkeys.BoundContext) error
}

type Opener struct {
	core CoreOpener
	keys KeyAuthorizer
}

// New keeps construction source-compatible while failing closed at Open when
// no lifecycle authorizer is supplied. Production wiring must pass the registry.
func New(core CoreOpener, authorizer ...KeyAuthorizer) (*Opener, error) {
	if core == nil || len(authorizer) > 1 {
		return nil, fmt.Errorf("secure-envelope opener configuration is invalid")
	}
	var keys KeyAuthorizer
	if len(authorizer) == 1 {
		keys = authorizer[0]
	}
	return &Opener{core: core, keys: keys}, nil
}

func (o *Opener) Open(ctx context.Context, envelope application.SecureEnvelope, aad []byte, key application.RevealKeyContext) ([]byte, error) {
	if o.keys == nil {
		return nil, fmt.Errorf("reveal key lifecycle authorizer is required")
	}
	if err := o.keys.AuthorizeBound(ctx, revealkeys.BoundContext{
		KeyID:           key.KeyID,
		PublicKeySHA256: key.PublicKeySHA256,
		BoundAt:         key.BoundAt,
		UseAt:           key.UseAt,
	}); err != nil {
		return nil, fmt.Errorf("reveal key context rejected: %w", err)
	}
	return o.core.Open(ctx, secureenvelope.Envelope{
		Version:         envelope.Version,
		KeyID:           envelope.KeyID,
		Suite:           envelope.Suite,
		EncapsulatedKey: envelope.EncapsulatedKey,
		Ciphertext:      envelope.Ciphertext,
	}, aad)
}
