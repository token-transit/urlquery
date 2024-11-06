package urlquery

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestErrUnhandledType_Error(t *testing.T) {
	err := ErrUnhandledType{typ: reflect.TypeOf("s")}
	if err.Error() != "failed to unhandled type(string)" {
		t.Error(err.Error())
	}
}

func TestIsMissingParamError(t *testing.T) {
	testCases := []struct {
		err    error
		result bool
	}{{
		err:    nil,
		result: false,
	}, {
		err:    ErrInvalidParamKey{key: "foo"},
		result: false,
	}, {
		err:    ErrInvalidParamValue{key: "foo", val: "bar"},
		result: false,
	}, {
		err:    ErrMissingRequiredParam{key: "foo"},
		result: true,
	}}
	for i, tc := range testCases {
		if res := IsMissingParamError(tc.err); res != tc.result {
			t.Errorf("testCases[%d]: unexpected IsMissingParamError for %v: expected %t, got %t", i, tc.err, tc.result, res)
		}
	}
}

func TestIsInvalidParamError(t *testing.T) {
	testCases := []struct {
		err    error
		result bool
	}{{
		err:    nil,
		result: false,
	}, {
		err:    ErrInvalidParamKey{key: "foo"},
		result: true,
	}, {
		err:    ErrInvalidParamValue{key: "foo", val: "bar"},
		result: true,
	}, {
		err:    ErrMissingRequiredParam{key: "foo"},
		result: false,
	}}
	for i, tc := range testCases {
		if res := IsInvalidParamError(tc.err); res != tc.result {
			t.Errorf("testCases[%d]: unexpected IsMissingParamError result for %v: expected %t, got %t", i, tc.err, tc.result, res)
		}
	}
}

func TestIsInvalidDestinationValueError(t *testing.T) {
	testCases := []struct {
		err    error
		result bool
	}{{
		err:    nil,
		result: false,
	}, {
		err:    ErrInvalidParamKey{key: "foo"},
		result: false,
	}, {
		err:    ErrInvalidParamValue{key: "foo", val: "bar"},
		result: false,
	}, {
		err:    ErrMissingRequiredParam{key: "foo"},
		result: false,
	}, {
		err:    ErrInvalidMapKeyType{key: "foo", typ: reflect.TypeOf("")},
		result: true,
	}, {
		err:    ErrInvalidMapValueType{key: "foo", typ: reflect.TypeOf("")},
		result: true,
	}, {
		err:    ErrUnhandledType{typ: reflect.TypeOf("")},
		result: true,
	}, {
		err:    ErrAmbiguousParseForParam{key: "foo"},
		result: true,
	}}
	for i, tc := range testCases {
		if res := IsInvalidDestinationValueError(tc.err); res != tc.result {
			t.Errorf("testCases[%d]: unexpected IsInvalidDestinationValueError for %v: expected %t, got %t", i, tc.err, tc.result, res)
		}
	}
}

func TestParamNameFromError(t *testing.T) {
	testCases := []struct {
		err       error
		paramName string
	}{{
		err:       nil,
		paramName: "",
	}, {
		err:       ErrInvalidParamKey{key: "foo"},
		paramName: "foo",
	}, {
		err:       ErrInvalidParamValue{key: "foo", val: "bar"},
		paramName: "foo",
	}, {
		err:       ErrInvalidUnmarshalError{},
		paramName: "",
	}, {
		err:       ErrMissingRequiredParam{key: "foo"},
		paramName: "foo",
	}, {
		err:       ErrInvalidMapKeyType{key: "foo", typ: reflect.TypeOf(int64(2))},
		paramName: "foo",
	}, {
		err:       ErrInvalidMapValueType{key: "foo", typ: reflect.TypeOf("")},
		paramName: "foo",
	}}
	for i, tc := range testCases {
		if paramName := ParamNameFromError(tc.err); paramName != tc.paramName {
			t.Errorf("testCases[%d]: expected param name %q, got %q", i, tc.paramName, paramName)
		}
	}
}

func TestErrInvalidParamKey(t *testing.T) {
	err := ErrInvalidParamKey{key: "foo"}
	if errStr := err.Error(); !strings.Contains(errStr, "foo") {
		t.Error("expected invalid param error to contain name of param")
	}
}

func TestErrInvalidParamValue(t *testing.T) {
	err := ErrInvalidParamValue{key: "foo", val: "bar"}
	if errStr := err.Error(); !strings.Contains(errStr, "foo") || !strings.Contains(errStr, "bar") {
		t.Error("expected invalid param error to contain name of param and name of value")
	}
}

func TestErrMissingRequiredParam(t *testing.T) {
	err := ErrMissingRequiredParam{key: "foo"}
	if errStr := err.Error(); !strings.Contains(errStr, "foo") {
		t.Error("expected missing param to contain name of missing param")
	}
}

func TestErrTranslated_Error(t *testing.T) {
	err1 := errors.New("new")
	err := ErrTranslated{err: err1}
	if err.Error() != "failed to translate:new" {
		t.Error(err.Error())
	}
}

func TestErrUnsupportedBitSize_Error(t *testing.T) {
	err := ErrUnsupportedBitSize{bitSize: 32}
	if err.Error() != "failed to handle unsupported bitSize(32)" {
		t.Error(err.Error())
	}
}

func TestErrInvalidMapKeyType_Error(t *testing.T) {
	f := 3.14
	err := ErrInvalidMapKeyType{typ: reflect.TypeOf(f)}
	if !strings.HasPrefix(err.Error(), "failed to handle map key type") {
		t.Errorf("expected error to start with %q, instead got %v", "failed to handle map key type", err)
	}
}

func TestErrInvalidUnmarshalError_Error(t *testing.T) {
	err := ErrInvalidUnmarshalError{}
	if err.Error() != "failed to unmarshal(non-pointer)" {
		t.Error(err.Error())
	}
}

func TestErrInvalidMapValueType_Error(t *testing.T) {
	i := uint(2)
	err := ErrInvalidMapValueType{typ: reflect.TypeOf(i)}
	if !strings.HasPrefix(err.Error(), "failed to handle map value type") {
		t.Errorf("expected error to start with %q, instead got %v", "failed to handle map value type", err)
	}
}
