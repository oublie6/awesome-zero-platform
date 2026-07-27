package secureenvelope

import "errors"

var (
	// ErrInvalidEnvelope reports a malformed or unsupported envelope before decryption.
	ErrInvalidEnvelope = errors.New("secure envelope is invalid")
	// ErrKeyUnavailable reports that the requested private key version is unavailable.
	ErrKeyUnavailable = errors.New("secure envelope key is unavailable")
	// ErrOpenFailed intentionally hides cryptographic failure details from callers.
	ErrOpenFailed = errors.New("secure envelope could not be opened")
)
