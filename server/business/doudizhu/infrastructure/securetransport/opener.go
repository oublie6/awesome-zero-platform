package securetransport

import (
	"context"
	"fmt"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application"
	"github.com/oublie6/awesome-zero-platform/server/foundation/secureenvelope"
)

type CoreOpener interface {
	Open(context.Context, secureenvelope.Envelope, []byte) ([]byte, error)
}

type Opener struct{ core CoreOpener }

func New(core CoreOpener) (*Opener, error) {
	if core == nil {
		return nil, fmt.Errorf("secure-envelope opener is required")
	}
	return &Opener{core: core}, nil
}

func (o *Opener) Open(ctx context.Context, envelope application.SecureEnvelope, aad []byte) ([]byte, error) {
	return o.core.Open(ctx, secureenvelope.Envelope{
		Version:         envelope.Version,
		KeyID:           envelope.KeyID,
		Suite:           envelope.Suite,
		EncapsulatedKey: envelope.EncapsulatedKey,
		Ciphertext:      envelope.Ciphertext,
	}, aad)
}
