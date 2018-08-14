package errors

import (
	"fmt"
	"reflect"

	"github.com/CentrifugeInc/centrifuge-protobufs/gen/go/errors"
	"github.com/go-errors/errors"
)

// errpb is the type alias for errorspb.Error
type errpb errorspb.Error

// Error implements the error interface
// message format: [code]message: [sub errors if any]
func (err *errpb) Error() string {
	if err.Errors == nil || len(err.Errors) == 0 {
		return fmt.Sprintf("[%d]%s", err.Code, err.Message)
	}

	return fmt.Sprintf("[%d]%s: %v", err.Code, err.Message, err.Errors)
}

// NewPBError constructs and returns a new error with code and message passed
func NewPBError(code Code, message string, errors map[string]string) error {
	return &errpb{
		Code:    int32(code),
		Message: message,
		Errors:  errors,
	}
}

// P2PError represents p2p error type
type P2PError struct {
	err *errorspb.Error
}

// FromError constructs and returns errorspb.Error
// if bool true, conversion to P2PError successful
// else failed and returns unknown P2PError
func FromError(err error) (*P2PError, bool) {
	if err == nil {
		return &P2PError{err: &errorspb.Error{Code: int32(Ok)}}, true
	}

	errpb, ok := err.(*errpb)
	if !ok {
		return &P2PError{err: &errorspb.Error{Code: int32(Unknown), Message: "unknown error"}}, false
	}

	return &P2PError{err: (*errorspb.Error)(errpb)}, true
}

// Code returns the error code
func (p2pErr *P2PError) Code() Code {
	if p2pErr == nil || p2pErr.err == nil {
		return Ok
	}

	return getCode(p2pErr.err.Code)
}

// Message returns error message
func (p2pErr *P2PError) Message() string {
	if p2pErr == nil || p2pErr.err == nil {
		return ""
	}

	return p2pErr.err.Message
}

// Errors returns map errors passed
func (p2pErr *P2PError) Errors() map[string]string {
	if p2pErr == nil || p2pErr.err == nil {
		return nil
	}

	return p2pErr.err.Errors
}

func GenerateNilParameterError(param interface{}) error {
	return errors.Errorf("NIL %v provided", reflect.TypeOf(param))
}

func New(message string) error {
	return errors.New(message)
}
