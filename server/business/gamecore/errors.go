package gamecore

import "errors"

var (
	ErrInvalidArgument       = errors.New("gamecore: invalid argument")
	ErrUnsupportedVersion    = errors.New("gamecore: unsupported version")
	ErrVerificationFailed    = errors.New("gamecore: verification failed")
	ErrDuplicateRegistration = errors.New("gamecore: duplicate registration")
	ErrModuleNotFound        = errors.New("gamecore: module not found")
	ErrInstanceNotFound      = errors.New("gamecore: live instance not found")
	ErrInstanceExists        = errors.New("gamecore: live instance already exists")
	ErrFinalizationPending   = errors.New("gamecore: finalization pending")
	ErrNotFinalizing         = errors.New("gamecore: no finalization pending")
	ErrArchiveFailed         = errors.New("gamecore: final archive failed")
)
