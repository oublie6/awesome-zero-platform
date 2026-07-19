package apperrors

import (
	"errors"
	"fmt"
	"net/http"
)

const (
	CodeOK              = "OK"
	CodeParamInvalid    = "PARAM_INVALID"
	CodeUnauthorized    = "UNAUTHORIZED"
	CodeForbidden       = "FORBIDDEN"
	CodeNotFound        = "NOT_FOUND"
	CodeConflict        = "CONFLICT"
	CodeRequestTooLarge = "REQUEST_TOO_LARGE"
	CodeInternal        = "INTERNAL_ERROR"
)

type Error struct {
	code    string
	message string
	status  int
	cause   error
}

func New(code, message string, status int) *Error {
	return &Error{
		code:    code,
		message: message,
		status:  status,
	}
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}

	if e.cause != nil {
		return e.cause.Error()
	}

	return e.message
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.cause
}

func (e *Error) Code() string {
	return e.code
}

func (e *Error) Message() string {
	return e.message
}

func (e *Error) Status() int {
	return e.status
}

func (e *Error) Cause() error {
	return e.cause
}

func (e *Error) WithCause(cause error) *Error {
	clone := *e
	clone.cause = cause
	return &clone
}

func InvalidParameter(message string) *Error {
	return New(CodeParamInvalid, message, http.StatusBadRequest)
}

func Unauthorized(message string) *Error {
	return New(CodeUnauthorized, message, http.StatusUnauthorized)
}

func Forbidden(message string) *Error {
	return New(CodeForbidden, message, http.StatusForbidden)
}

func NotFound(message string) *Error {
	return New(CodeNotFound, message, http.StatusNotFound)
}

func Conflict(message string) *Error {
	return New(CodeConflict, message, http.StatusConflict)
}

func RequestTooLarge() *Error {
	return New(CodeRequestTooLarge, "request body too large", http.StatusRequestEntityTooLarge)
}

func Internal(cause error) *Error {
	return New(CodeInternal, "internal server error", http.StatusInternalServerError).WithCause(cause)
}

func As(err error) (*Error, bool) {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr, true
	}

	return nil, false
}

func StatusCode(err error) int {
	if appErr, ok := As(err); ok {
		return appErr.Status()
	}

	return http.StatusInternalServerError
}

func SafeMessage(err error) string {
	if appErr, ok := As(err); ok {
		return appErr.Message()
	}

	return "internal server error"
}

func StableCode(err error) string {
	if appErr, ok := As(err); ok {
		return appErr.Code()
	}

	return CodeInternal
}

func Cause(err error) error {
	if appErr, ok := As(err); ok && appErr.Cause() != nil {
		return appErr.Cause()
	}

	return err
}

func String(err error) string {
	if err == nil {
		return ""
	}

	if appErr, ok := As(err); ok {
		if appErr.Cause() != nil {
			return fmt.Sprintf("%s: %v", appErr.Code(), appErr.Cause())
		}

		return appErr.Code()
	}

	return err.Error()
}
