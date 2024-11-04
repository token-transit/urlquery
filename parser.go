package urlquery

import (
	"bytes"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

// parser from URL Query string to go structure
type parser struct {
	container      map[string]string
	err            error
	opts           options
	mutex          sync.Mutex
	queryEncoder   QueryEncoder
	decodeFuncMap  map[reflect.Kind]valueDecode
	customDecoders []ValueDecoder
}

var anyType reflect.Type = reflect.TypeOf(new(interface{})).Elem()

// NewParser make a new parser object
// do some option initialization
func NewParser(opts ...Option) *parser {
	p := &parser{}
	for _, option := range opts {
		option(&p.opts)
	}
	p.decodeFuncMap = make(map[reflect.Kind]valueDecode)
	return p
}

// handle string data to a map structure for the next parsing
func (p *parser) init(data []byte) (err error) {
	arr := bytes.Split(data, []byte(SymbolAnd))
	for _, value := range arr {
		ns := strings.SplitN(string(value), SymbolEqual, 2)
		if len(ns) > 1 {
			ns[0], err = p.queryEncoder.UnEscape(ns[0])
			if err != nil {
				return ErrInvalidParamKey{key: ns[0], err: err}
			}

			ns[1], err = p.queryEncoder.UnEscape(ns[1])
			if err != nil {
				return ErrInvalidParamValue{key: ns[0], val: ns[0], err: err}
			}

			//If last two characters of key equal `[]`, repack it to `[{i++}]`
			l := len(ns[0])
			if l > 2 && ns[0][l-2:] == "[]" {
				//limit iteration to avoid attack of large or dead circle
				for i := 0; i < 1000000; i++ {
					tKey := ns[0][:l-2] + "[" + strconv.Itoa(i) + "]"
					if _, ok := p.container[tKey]; !ok {
						ns[0] = tKey
						break
					}
				}
			}

			p.container[ns[0]] = ns[1]
		}
	}
	return
}

func (p *parser) initValues(urlValues url.Values) {
	for key, values := range urlValues {
		l := len(key)
		if l > 2 && key[l-2:] == "[]" {
			key = key[:l-2]
			for i, v := range values {
				newKey := key + "[" + strconv.Itoa(i) + "]"
				p.container[newKey] = v
			}
		} else {
			p.container[key] = values[0]
		}
	}
}

// reset specified query encoder
func (p *parser) resetQueryEncoder() {
	if p.opts.queryEncoder != nil {
		p.queryEncoder = p.opts.queryEncoder
	} else {
		p.queryEncoder = getQueryEncoder()
	}
}

// generate next parent node key
func (p *parser) genNextParentNode(parentNode, key string) string {
	return genNextParentNode(parentNode, key)
}

// iteratively parse go structure from string
func (p *parser) parse(rv reflect.Value, parentNode string) (found bool) {
	if p.err != nil {
		return
	}

	// Certain types are not meant to be expanded and should
	// just be decoded immediately
	kind := rv.Kind()
	if kind != reflect.Invalid && p.canImmediatelyDecode(rv.Type()) {
		return p.parseValue(rv, parentNode)
	}

	switch kind {
	case reflect.Ptr:
		found = p.parseForPrt(rv, parentNode) || found
	case reflect.Interface:
		found = p.parse(rv.Elem(), parentNode) || found
	case reflect.Map:
		found = p.parseForMap(rv, parentNode) || found
	case reflect.Array:
		for i := 0; i < rv.Cap(); i++ {
			found = p.parse(rv.Index(i), p.genNextParentNode(parentNode, strconv.Itoa(i))) || found
		}
	case reflect.Slice:
		found = p.parseForSlice(rv, parentNode) || found
	case reflect.Struct:
		found = p.parseForStruct(rv, parentNode) || found
	default:
		found = p.parseValue(rv, parentNode) || found
	}
	return found
}

// parse for pointer value
func (p *parser) parseForPrt(rv reflect.Value, parentNode string) (found bool) {
	//If Ptr is nil and can be set, Ptr should be initialized
	if rv.IsNil() {
		if rv.CanSet() {

			//lookup matched map data with prefix key
			matches := p.lookup(parentNode)
			// If none match keep nil
			if len(matches) == 0 {
				return false
			}

			rv.Set(reflect.New(rv.Type().Elem()))
			found = p.parse(rv.Elem(), parentNode) || found
		}
	} else {
		found = p.parse(rv.Elem(), parentNode) || found
	}
	return
}

// reconstruct key name from parts
func keyName(prefix string, index string) string {
	if prefix == "" {
		return index
	}
	return fmt.Sprintf("%s[%s]", prefix, index)
}

// parse for map value
func (p *parser) parseForMap(rv reflect.Value, parentNode string) (found bool) {
	if !rv.CanSet() {
		return
	}

	//limited condition of map key and value type
	//If not meet the condition, will return error
	if !isAccessMapKeyType(rv.Type().Key().Kind()) || !isAccessMapValueType(rv.Type().Elem().Kind()) {
		p.err = ErrInvalidMapKeyType{typ: rv.Type()}
		return
	}

	matches := p.lookup(parentNode)
	size := len(matches)

	if size == 0 {
		return
	}

	mapReflect := reflect.MakeMapWithSize(rv.Type(), size)
	for k := range matches {
		reflectKey, err := p.decode(rv.Type().Key(), k)
		if err != nil {
			p.err = ErrInvalidParamKey{key: keyName(parentNode, k), err: err}
			return
		}

		value, ok := p.get(p.genNextParentNode(parentNode, k))
		if !ok {
			continue
		}

		found = true
		reflectValue, err := p.decode(rv.Type().Elem(), value)
		if err != nil {
			p.err = ErrInvalidParamValue{val: value, key: keyName(parentNode, k), err: err}
			return
		}

		mapReflect.SetMapIndex(reflectKey, reflectValue)
	}
	// actually we only want to replace the nil map if we actually found a value in the map.
	if found {
		rv.Set(mapReflect)
	}
	return
}

// parse for slice value
func (p *parser) parseForSlice(rv reflect.Value, parentNode string) (found bool) {
	if !rv.CanSet() {
		return
	}

	//lookup matched map data with prefix key
	matches, err := p.lookupForSlice(parentNode)
	if err != nil {
		p.err = err
		return
	} else if len(matches) == 0 {
		return
	}

	//get max cap of slice
	maxCap := 0
	for i := range matches {
		if i+1 > maxCap {
			maxCap = i + 1
		}
	}

	// We always create a new slice because we want to maintain the old slice exactly if parse is wrong.
	sliceVal := reflect.MakeSlice(rv.Type(), maxCap, maxCap)
	if !rv.IsNil() {
		reflect.Copy(sliceVal, rv)
	}

	for i := range matches {
		found = p.parse(sliceVal.Index(i), p.genNextParentNode(parentNode, strconv.Itoa(i))) || found
	}
	if found {
		rv.Set(sliceVal)
	}
	return
}

// parse for struct value
func (p *parser) parseForStruct(rv reflect.Value, parentNode string) (found bool) {
	for i := 0; i < rv.NumField(); i++ {
		ft := rv.Type().Field(i)

		//specially handle anonymous fields
		if ft.Anonymous && rv.Field(i).Kind() == reflect.Struct {
			found = p.parse(rv.Field(i), parentNode) || found
			continue
		}

		tag := ft.Tag.Get("query")
		//all ignore
		if tag == "-" {
			continue
		}

		t := newTag(tag)
		name := t.getName()
		if name == "" {
			name = ft.Name
		}

		nodeName := p.genNextParentNode(parentNode, name)
		required := t.contains("required")
		if required {
			if len(p.lookup(nodeName)) == 0 {
				if p.err == nil {
					p.err = ErrMissingRequiredParam{key: nodeName}
				}
				return
			}
		}
		found = p.parse(rv.Field(i), nodeName) || found
		if required && !found {
			if p.err == nil {
				p.err = ErrMissingRequiredParam{key: nodeName}
			}
			return
		}
	}
	return
}

// parse text to specified-type value, set into rv
func (p *parser) parseValue(rv reflect.Value, parentNode string) (found bool) {
	if !rv.CanSet() {
		return
	}

	value, ok := p.get(parentNode)
	if !ok {
		return
	}

	found = true
	typ := rv.Type()
	v, err := p.decode(typ, value)
	if err != nil {
		p.err = ErrInvalidParamValue{key: parentNode, val: value, err: err}
		return
	}

	// This allows converters to handle types as long as they can be converted
	// into the proper type.
	if t := v.Type(); t != typ {
		if rv.CanConvert(t) {
			v = v.Convert(typ)
		} else {
			p.err = ErrInvalidParamValue{key: parentNode, val: value, err: ErrUnhandledType{rv.Type()}}
			return
		}
	}
	rv.Set(v)
	return
}

func (p *parser) canImmediatelyDecode(typ reflect.Type) bool {
	if typ == anyType {
		return false
	}
	for _, d := range p.customDecoders {
		if d.DecodesType(typ) {
			return true
		}
	}
	for _, d := range builtinDecoders {
		if d.DecodesType(typ) {
			return true
		}
	}
	return isDecodable(typ)
}

// parse text to specified-type value
func (p *parser) decode(typ reflect.Type, value string) (v reflect.Value, err error) {
	if typ != reflect.TypeOf(anyType) {
		// Custom decoders override everything
		for _, d := range p.customDecoders {
			if d.DecodesType(typ) {
				return d.Decode(value)
			}
		}
		for _, d := range builtinDecoders {
			if d.DecodesType(typ) {
				return d.Decode(value)
			}
		}
		// Next check for
		if rv, err, ok := maybeDecodeableDecode(typ, value); ok {
			return rv, err
		}
	}
	decodeFunc := p.getDecodeFunc(typ.Kind())
	if decodeFunc == nil {
		err = ErrUnhandledType{typ: typ}
		return
	}
	return decodeFunc(value)
}

// get decode function for specified reflect kind
func (p *parser) getDecodeFunc(kind reflect.Kind) valueDecode {
	if decodeFunc, ok := p.decodeFuncMap[kind]; ok {
		return decodeFunc
	}
	return getDecodeFunc(kind)
}

// lookup by prefix matching
func (p *parser) lookup(prefix string) map[string]bool {
	data := map[string]bool{}
	for k := range p.container {
		if strings.HasPrefix(k, prefix) {
			suf := k[len(prefix):]
			if prefix != "" && len(suf) > 0 && !strings.HasPrefix(k[len(prefix):], "[") {
				continue
			}
			pre, _ := unpackQueryKey(k[len(prefix):])
			data[pre] = true
		}
	}
	return data
}

// lookup by prefix matching
func (p *parser) lookupForSlice(prefix string) (map[int]bool, error) {
	tmp := p.lookup(prefix)
	data := map[int]bool{}
	for k := range tmp {
		i, err := strconv.Atoi(k)
		if err != nil {
			return nil, ErrInvalidParamKey{key: fmt.Sprintf("%s[%s]", prefix, k), err: err}
		}
		data[i] = true
	}
	return data, nil
}

// get value by key from container variable which is map struct
func (p *parser) get(key string) (string, bool) {
	v, ok := p.container[key]
	return v, ok
}

// self-defined valueDecode function
func (p *parser) RegisterDecodeFunc(kind reflect.Kind, decode valueDecode) {
	p.decodeFuncMap[kind] = decode
}

func (p *parser) RegisterValueDecoder(valueDecoder ValueDecoder) {
	if valueDecoder != nil {
		p.customDecoders = append(p.customDecoders, valueDecoder)
	}
}

func (p *parser) UnmarshalValues(values url.Values, v interface{}) (err error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.container = map[string]string{}
	p.err = nil
	p.resetQueryEncoder()

	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return ErrInvalidUnmarshalError{}
	}

	p.initValues(values)
	p.parse(rv, "")
	p.container = nil
	return p.err
}

// Unmarshal is supposed to decode string to go structure
// It is thread safety
func (p *parser) Unmarshal(data []byte, v interface{}) (err error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	//for duplicate use
	p.container = map[string]string{}
	p.err = nil
	p.resetQueryEncoder()

	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return ErrInvalidUnmarshalError{}
	}

	err = p.init(data)
	if err != nil {
		return
	}

	p.parse(rv, "")

	//release resource
	p.container = nil
	return p.err
}

// Unmarshal is supposed to decode string to go structure
// It is threadsafe
func Unmarshal(data []byte, v interface{}) error {
	p := NewParser()
	return p.Unmarshal(data, v)
}

// Unmarshal is supposed to decode string to go structure
// It is threadsafe
func UnmarshalValues(urlValues url.Values, v interface{}) error {
	p := NewParser()
	return p.UnmarshalValues(urlValues, v)
}
