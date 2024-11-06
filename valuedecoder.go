package urlquery

import (
	"reflect"
	"strconv"
	"time"
)

// ValueDecoder are things that can decode string into a particular type
// The DecodesType function returns whether the value decoder can
// handle a particular type.
type ValueDecoder interface {
	DecodesType(reflect.Type) bool
	Decode(value string) (reflect.Value, error)
}

var builtinDecoders = []ValueDecoder{TimeDecoder{}}

type TimeDecoder struct{}

func (TimeDecoder) Decode(value string) (reflect.Value, error) {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return reflect.Value{}, err
	}
	return reflect.ValueOf(t), nil
}

func (TimeDecoder) DecodesType(t reflect.Type) bool {
	return reflect.TypeOf(time.Time{}).ConvertibleTo(t)
}

// Any type that implements Decodable will be automatically
// decoded using the DecodeValue function
type Decodable interface {
	UnmarshalQueryParam(value string) error
}

func isDecodable(typ reflect.Type) bool {
	var d *Decodable
	if typ.Implements(reflect.TypeOf(d).Elem()) {
		if typ.Kind() == reflect.Pointer {
			// Apparently pointers to things that implement something also count
			// as implementing the thing. Because we can only instantiate
			// one type automatically, we will just wait until
			// the parser instantiates the pointer and then we will parse.
			// For example, if we have a type that's a map and it implements Decodable
			// but we're parsing a *map[string]string then we will let the parser handle the pointer
			return !typ.Elem().Implements(reflect.TypeOf(d).Elem())
		} else if typ.Kind() == reflect.Map {
			return true
		}
		return false
	}
	return reflect.PointerTo(typ).Implements(reflect.TypeOf(d).Elem())
}

// Decode any type T if either T or *T implements Decodable
// Returns the decoded value, nil, and true if successful
// Returns the empty reflect.Value, error from Parse(), and true if unsuccessful
// Returns the empty reflect.Value, nil, and false if not decodable
func maybeDecodeableDecode(typ reflect.Type, value string) (reflect.Value, error, bool) {
	var d Decodable
	var rv reflect.Value
	var mustIndirect bool
	if typ.Implements(reflect.TypeOf(&d).Elem()) {
		// Almost always Parse is going to be a pointer receiver
		// but it might also be a map
		if typ.Kind() == reflect.Pointer {
			rv = reflect.New(typ.Elem())
		} else if typ.Kind() == reflect.Map {
			rv = reflect.MakeMap(typ)
		} else {
			return reflect.Value{}, nil, false
		}
		d = rv.Interface().(Decodable)
	} else if reflect.PointerTo(typ).Implements(reflect.TypeOf(&d).Elem()) {
		// We also allow a pointer to the type to implement Decodable
		rv = reflect.New(typ)
		d = rv.Interface().(Decodable)
		mustIndirect = true
	} else {
		return reflect.Value{}, nil, false
	}
	if err := d.UnmarshalQueryParam(value); err != nil {
		return reflect.Value{}, ErrTranslated{err: err}, true
	}
	if mustIndirect {
		return rv.Elem(), nil, true
	}
	return rv, nil, true
}

// A valueDecode is a converter from string to go basic structure
type valueDecode func(string) (reflect.Value, error)

// convert from string to bool
func boolDecode(value string) (reflect.Value, error) {
	b, err := strconv.ParseBool(value)
	if err != nil {
		err = ErrTranslated{err: err}
		return reflect.Value{}, err
	}
	return reflect.ValueOf(b), nil
}

// convert from string to int(8-64)
func baseIntDecode(value string, bitSize int) (rv reflect.Value, err error) {
	v, err := strconv.ParseInt(value, 10, bitSize)
	if err != nil {
		err = ErrTranslated{err: err}
		return
	}
	switch bitSize {
	case 64:
		rv = reflect.ValueOf(v)
	case 32:
		rv = reflect.ValueOf(int32(v))
	case 16:
		rv = reflect.ValueOf(int16(v))
	case 8:
		rv = reflect.ValueOf(int8(v))
	case 0:
		rv = reflect.ValueOf(int(v))
	default:
		err = ErrUnsupportedBitSize{bitSize: bitSize}
	}
	return
}

// convert from string to int(8-64)
func intDecode(bitSize int) valueDecode {
	return func(value string) (reflect.Value, error) {
		return baseIntDecode(value, bitSize)
	}
}

// convert from string to uint(8-64)
func baseUintDecode(value string, bitSize int) (rv reflect.Value, err error) {
	v, err := strconv.ParseUint(value, 10, bitSize)
	if err != nil {
		err = ErrTranslated{err: err}
		return
	}
	switch bitSize {
	case 64:
		rv = reflect.ValueOf(v)
	case 32:
		rv = reflect.ValueOf(uint32(v))
	case 16:
		rv = reflect.ValueOf(uint16(v))
	case 8:
		rv = reflect.ValueOf(uint8(v))
	case 0:
		rv = reflect.ValueOf(uint(v))
	default:
		err = ErrUnsupportedBitSize{bitSize: bitSize}
	}
	return
}

// convert from string to uint(8-64)
func uintDecode(bitSize int) valueDecode {
	return func(value string) (reflect.Value, error) {
		return baseUintDecode(value, bitSize)
	}
}

// convert from string to uintPrt
func uintPrtDecode(value string) (rv reflect.Value, err error) {
	v, err := strconv.ParseUint(value, 10, 0)
	if err != nil {
		err = ErrTranslated{err: err}
		return
	}
	return reflect.ValueOf(uintptr(v)), nil
}

// convert from string to float,double
func baseFloatDecode(value string, bitSize int) (rv reflect.Value, err error) {
	v, err := strconv.ParseFloat(value, bitSize)
	if err != nil {
		err = ErrTranslated{err: err}
		return
	}
	switch bitSize {
	case 64:
		rv = reflect.ValueOf(v)
	case 32:
		rv = reflect.ValueOf(float32(v))
	default:
		err = ErrUnsupportedBitSize{bitSize: bitSize}
	}
	return
}

// convert from string to float
func floatDecode(bitSize int) valueDecode {
	return func(value string) (reflect.Value, error) {
		return baseFloatDecode(value, bitSize)
	}
}

// convert from string to string
func stringDecode(value string) (reflect.Value, error) {
	return reflect.ValueOf(value), nil
}

// get decode func for specified reflect kind
func getDecodeFunc(kind reflect.Kind) valueDecode {
	switch kind {
	case reflect.Bool:
		return boolDecode
	case reflect.Int:
		return intDecode(0)
	case reflect.Int8:
		return intDecode(8)
	case reflect.Int16:
		return intDecode(16)
	case reflect.Int32:
		return intDecode(32)
	case reflect.Int64:
		return intDecode(64)
	case reflect.Uint:
		return uintDecode(0)
	case reflect.Uint8:
		return uintDecode(8)
	case reflect.Uint16:
		return uintDecode(16)
	case reflect.Uint32:
		return uintDecode(32)
	case reflect.Uint64:
		return uintDecode(64)
	case reflect.Uintptr:
		return uintPrtDecode
	case reflect.Float32:
		return floatDecode(32)
	case reflect.Float64:
		return floatDecode(64)
	case reflect.String:
		return stringDecode
	default:
		return nil
	}
}
