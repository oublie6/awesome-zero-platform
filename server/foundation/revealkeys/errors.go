package revealkeys

import "errors"

var (
	ErrInvalidConfig    = errors.New("invalid reveal key configuration")
	ErrKeyUnavailable   = errors.New("reveal key unavailable")
	ErrKeyNotCurrent    = errors.New("reveal key is not current")
	ErrKeyRevoked       = errors.New("reveal key revoked")
	ErrKeyExpired       = errors.New("reveal key expired")
	ErrKeyHashMismatch  = errors.New("reveal key hash mismatch")
	ErrInvalidManifest  = errors.New("invalid reveal key manifest")
	ErrSignatureInvalid = errors.New("reveal key manifest signature invalid")
	ErrManifestRollback = errors.New("reveal key manifest version rollback")
)
