package customerrors

import "errors"

var (
	ErrDataNotFound           = errors.New("Data Not Found")
	ErrInvalidPassword        = errors.New("Invalid Password")
	ErrTokenExpired           = errors.New("Token Expired")
	ErrUnauthorized           = errors.New("Unauthorized")
	ErrBadRequest             = errors.New("Bad Request")
	ErrCooldownActive         = errors.New("Please Wait 5 Minutes Before Sending Again")
	ErrInvalidCredentials     = errors.New("Invalid Username / Email")
	ErrAccountInactive        = errors.New("Account Not Verified")
	ErrAccountAlreadyVerified = errors.New("Account Already Verified")
	ErrFailHash               = errors.New("Failed to hash Password")
)

type ValidationError struct {
	Detail string
}

func (e *ValidationError) Error() string {
	return "Validation Failed"
}

func NewValidationError(detail string) error {
	return &ValidationError{
		Detail: detail,
	}
}

type ConflictError struct {
	Detail string
}

func (e *ConflictError) Error() string {
	return "Data Conflict"
}

func NewConflictError(detail string) error {
	return &ConflictError{
		Detail: detail,
	}
}

type NotFoundError struct {
	Detail string
}

func (e *NotFoundError) Error() string {
	return "Data Not Found"
}

func NewNotFoundError(detail string) error {
	return &NotFoundError{
		Detail: detail,
	}
}
