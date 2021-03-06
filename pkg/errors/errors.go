package errors

import (
	"fmt"

	"github.com/pkg/errors"
)

// Error is error interface for deepalert to handle related variables
type Error struct {
	cause  error
	Values map[string]interface{} `json:"values"`
}

func newError(cause error) *Error {
	return &Error{
		cause:  cause,
		Values: make(map[string]interface{}),
	}
}

// New creates a new error with message
func New(msg string) *Error {
	return newError(errors.New(msg))
}

// Error returns error message for error interface
func (x *Error) Error() string {
	return x.cause.Error()
}

// Unwrap returns *fundamental of github.com/pkg/errors
func (x *Error) Unwrap() error {
	return x.cause
}

// With adds key and value related to the error event
func (x *Error) With(key string, value interface{}) *Error {
	x.Values[key] = value
	return x
}

func (x *Error) StackTrace() string {
	return fmt.Sprintf("%+v", x.cause)
}

// With adds key and value related to the error event
func With(cause error, key string, value interface{}) *Error {
	if err, ok := cause.(*Error); ok {
		return err.With(key, value)
	} else {
		err := newError(errors.WithStack(cause))
		return err.With(key, value)
	}
}

// Wrap creates a new Error and add message
func Wrap(cause error, msg string) *Error {
	if err, ok := cause.(*Error); ok {
		err.cause = errors.Wrap(err.cause, msg)
		return err
	} else {
		// err := errors.WithStack(cause)
		return newError(errors.Wrap(cause, msg))
	}
}
