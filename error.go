package urlquery

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
)

func ParamNameFromError(err error) string {
	if err == nil {
		return ""
	}
	var errKey ErrInvalidParamKey
	if errors.As(err, &errKey) {
		return errKey.key
	}
	var errValue ErrInvalidParamValue
	if errors.As(err, &errValue) {
		return errValue.key
	}
	var errMissing ErrMissingRequiredParam
	if errors.As(err, &errMissing) {
		return errMissing.key
	}
	var errKeyType ErrInvalidMapKeyType
	if errors.As(err, &errKeyType) {
		return errKeyType.key
	}
	var errValueType ErrInvalidMapValueType
	if errors.As(err, &errValueType) {
		return errValueType.key
	}
	return ""
}

func IsInvalidParamError(err error) bool {
	if err == nil {
		return false
	}
	var errKey ErrInvalidParamKey
	if errors.As(err, &errKey) {
		return true
	}
	var errValue ErrInvalidParamValue
	return errors.As(err, &errValue)
}

func IsMissingParamError(err error) bool {
	if err == nil {
		return false
	}
	var errMissing ErrMissingRequiredParam
	return errors.As(err, &errMissing)
}

func IsInvalidDestinationValueError(err error) bool {
	if err == nil {
		return false
	}
	var errKeyType ErrInvalidMapKeyType
	if errors.As(err, &errKeyType) {
		return true
	}
	var errValueType ErrInvalidMapValueType
	if errors.As(err, &errValueType) {
		return true
	}
	var errUnhandledType ErrUnhandledType
	if errors.As(err, &errUnhandledType) {
		return true
	}
	return false
}

type ErrInvalidParamKey struct {
	key string
	err error
}

func (e ErrInvalidParamKey) Error() string {
	return fmt.Sprintf("failed to parse param key %q: %v", e.key, e.err)
}

func (e ErrInvalidParamKey) Unwrap() error {
	return e.err
}

type ErrMissingRequiredParam struct {
	key string
}

func (e ErrMissingRequiredParam) Error() string {
	return fmt.Sprintf("missing required param %q", e.key)
}

// An ErrInvalidParamValue includes name of the param being decoded.
type ErrInvalidParamValue struct {
	val string
	key string
	err error
}

func (e ErrInvalidParamValue) Error() string {
	return fmt.Sprintf("failed to parse param value %q for key %q: %v", e.val, e.key, e.err)
}

func (e ErrInvalidParamValue) Unwrap() error {
	return e.err
}

// An ErrUnhandledType is a customized error
type ErrUnhandledType struct {
	typ reflect.Type
}

func (e ErrUnhandledType) Error() string {
	return "failed to unhandled type(" + e.typ.String() + ")"
}

// An ErrInvalidUnmarshalError is a customized error
type ErrInvalidUnmarshalError struct{}

func (e ErrInvalidUnmarshalError) Error() string {
	return "failed to unmarshal(non-pointer)"
}

// An ErrUnsupportedBitSize is a customized error
type ErrUnsupportedBitSize struct {
	bitSize int
}

func (e ErrUnsupportedBitSize) Error() string {
	return "failed to handle unsupported bitSize(" + strconv.Itoa(e.bitSize) + ")"
}

// An ErrTranslated is a customized error type
type ErrTranslated struct {
	err error
}

func (e ErrTranslated) Error() string {
	return "failed to translate:" + e.err.Error()
}

func (e ErrTranslated) Unwrap() error {
	return e.err
}

// An ErrInvalidMapKeyType is a customized error
type ErrInvalidMapKeyType struct {
	key string
	typ reflect.Type
}

func (e ErrInvalidMapKeyType) Error() string {
	return fmt.Sprintf("failed to handle map key type(%s) for key %q", e.typ.String(), e.key)
}

// An ErrInvalidMapValueType is a customized error
type ErrInvalidMapValueType struct {
	key string
	typ reflect.Type
}

func (e ErrInvalidMapValueType) Error() string {
	return fmt.Sprintf("failed to handle map value type(%s) for key %q", e.typ.String(), e.key)
}
