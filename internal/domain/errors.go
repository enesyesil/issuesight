package domain

import (
	"errors"
	"fmt"
)

// Sentinel errors - use errors.Is() to check these.
var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("forbidden")
	ErrInvalidInput  = errors.New("invalid input")
	ErrQuotaExceeded = errors.New("quota exceeded")
	ErrGitHubAPI = errors.New("github api error")
	ErrLLMAPI    = errors.New("llm api error")
)

// AppError wraps an error with HTTP status code and user-friendly message.
type AppError struct {
	Err        error  // underlying error
	Message    string // user-facing message
	StatusCode int    // HTTP status code
}

func (e *AppError) Error() string {
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// Error constructors

func NotFound(resource string) *AppError {
	return &AppError{
		Err:        ErrNotFound,
		Message:    fmt.Sprintf("%s not found", resource),
		StatusCode: 404,
	}
}

func AlreadyExists(resource string) *AppError {
	return &AppError{
		Err:        ErrAlreadyExists,
		Message:    fmt.Sprintf("%s already exists", resource),
		StatusCode: 409,
	}
}

func Unauthorized(msg string) *AppError {
	return &AppError{
		Err:        ErrUnauthorized,
		Message:    msg,
		StatusCode: 401,
	}
}

func Forbidden(msg string) *AppError {
	return &AppError{
		Err:        ErrForbidden,
		Message:    msg,
		StatusCode: 403,
	}
}

func BadRequest(msg string) *AppError {
	return &AppError{
		Err:        ErrInvalidInput,
		Message:    msg,
		StatusCode: 400,
	}
}

func QuotaExceeded(msg string) *AppError {
	return &AppError{
		Err:        ErrQuotaExceeded,
		Message:    msg,
		StatusCode: 429,
	}
}

func InternalError(msg string) *AppError {
	return &AppError{
		Err:        errors.New(msg),
		Message:    "internal server error",
		StatusCode: 500,
	}
}

// Wrap creates an AppError from any error with a status code.
func Wrap(err error, msg string, statusCode int) *AppError {
	return &AppError{
		Err:        err,
		Message:    msg,
		StatusCode: statusCode,
	}
}
