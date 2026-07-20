package identity

type ErrorCode string

const (
	CodeAccountNotFound     ErrorCode = "ACCOUNT_NOT_FOUND"
	CodeIdentityConflict    ErrorCode = "IDENTITY_CONFLICT"
	CodeInvalidAccountState ErrorCode = "INVALID_ACCOUNT_STATE"
	CodeInvalidCredentials  ErrorCode = "INVALID_CREDENTIALS"
	CodePersistenceFailure  ErrorCode = "PERSISTENCE_FAILURE"
)

type Error struct {
	Code    ErrorCode
	Message string
	Cause   error
}

var (
	ErrAccountNotFound     = &Error{Code: CodeAccountNotFound, Message: "account not found"}
	ErrIdentityConflict    = &Error{Code: CodeIdentityConflict, Message: "identity already exists"}
	ErrInvalidAccountState = &Error{Code: CodeInvalidAccountState, Message: "account state does not allow this operation"}
	ErrInvalidCredentials  = &Error{Code: CodeInvalidCredentials, Message: "invalid credentials"}
	ErrPersistence         = &Error{Code: CodePersistenceFailure, Message: "identity persistence failed"}
)

func (e *Error) Error() string {
	if e == nil {
		return ""
	}

	return e.Message
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.Cause
}

func (e *Error) Is(target error) bool {
	candidate, ok := target.(*Error)
	if !ok {
		return false
	}

	return e.Code == candidate.Code
}

func wrapIdentityError(template *Error, cause error) error {
	if cause == nil {
		return template
	}

	return &Error{
		Code:    template.Code,
		Message: template.Message,
		Cause:   cause,
	}
}
