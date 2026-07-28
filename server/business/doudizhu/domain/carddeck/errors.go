package carddeck

import "errors"

var (
	ErrInvalidArgument    = errors.New("carddeck: invalid argument")
	ErrUnsupportedVersion = errors.New("carddeck: unsupported version")
	ErrVerificationFailed = errors.New("carddeck: verification failed")
)
