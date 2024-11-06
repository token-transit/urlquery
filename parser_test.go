package urlquery

import (
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

type testParseChild struct {
	Description string `query:"desc"`
	Long        uint16 `query:",vip"`
	Height      int    `query:"-"`
}

type testParseChildRequired struct {
	Description string `query:"desc,required"`
	Long        uint16 `query:",vip"`
	Height      int    `query:"-"`
}

type testParseEncodedString struct {
	Str string
}

type testTimeType time.Time

func (tpes *testParseEncodedString) UnmarshalQueryParam(value string) error {
	tpes.Str = value
	return nil
}

type testStrArray3 [3]string

func (tsa3 *testStrArray3) UnmarshalQueryParam(value string) error {
	strs := strings.Split(value, ",")
	if len(strs) != 3 {
		return errors.New("testStrArray3 must have exactly 3 components")
	}
	copy(tsa3[:], strs[:3])
	return nil
}

type testIgnoreDecoder []string

func (tid testIgnoreDecoder) UnmarshalQueryParam(value string) error {
	strs := strings.Split(value, ",")
	copy(tid, strs)
	return nil
}

type testEncodedMap map[string]string

func (tem testEncodedMap) UnmarshalQueryParam(value string) error {
	if tem == nil {
		return errors.New("cannot call unmarhsal query param on an uninitialized map")
	}
	parts := strings.Split(value, ",")
	for _, p := range parts {
		queryValue := strings.Split(p, "=")
		if len(queryValue) != 2 {
			return fmt.Errorf("error parsing map value %q does not have single =", p)
		}
		if queryValue[0] == "" {
			return fmt.Errorf("query value key is empty")
		}
		tem[queryValue[0]] = tem[queryValue[1]]
	}
	return nil
}

func (tem testEncodedMap) MarshalQueryParam() string {
	var sb strings.Builder
	for k, v := range tem {
		if sb.Len() > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(v)
	}
	return sb.String()
}

type testParseInfo struct {
	Id               int
	Name             string           `query:"name"`
	Child            testParseChild   `query:"child"`
	ChildPtr         *testParseChild  `query:"childPtr"`
	Children         []testParseChild `query:"children"`
	Params           map[byte]int8
	status           bool
	UintPtr          uintptr
	Tags             []int16 `query:"tags"`
	Int64            int64
	Uint             uint
	Uint32           uint32
	Float32          float32
	Float64          float64
	Bool             bool
	Inter            interface{}             `query:"inter"`
	Time             time.Time               `query:"time"`
	TimePtr          *time.Time              `query:"time_ptr"`
	AlsoTime         testTimeType            `query:"also_time"`
	EncodedString    testParseEncodedString  `query:"encoded_string"`
	EncodedStringPtr *testParseEncodedString `query:"encoded_string_ptr"`
	EncodedMap       testEncodedMap          `query:"tem"`
	EncodedMapPtr    *testEncodedMap         `query:"tem_ptr"`
	StrArray3        testStrArray3           `query:"strarr3"`
	IgnoreDecoder    testIgnoreDecoder       `query:"ignore_decoder"`
}

type testReplacementTimeDecoder struct{}

func (d testReplacementTimeDecoder) DecodesType(typ reflect.Type) bool {
	return TimeDecoder{}.DecodesType(typ)
}
func (d testReplacementTimeDecoder) Decode(s string) (reflect.Value, error) {
	t, err := time.Parse("2006-01-02", s)
	if err == nil {
		return reflect.ValueOf(t), nil
	}
	return TimeDecoder{}.Decode(s)
}

type testBadDecoder struct{}

func (d testBadDecoder) DecodesType(typ reflect.Type) bool {
	return TimeDecoder{}.DecodesType(typ)
}
func (d testBadDecoder) Decode(s string) (reflect.Value, error) {
	return reflect.ValueOf(s), nil
}

type errorQueryEncoder struct {
	times   int
	errorAt int
}

var errQueryEncoder = errors.New("failed")

func (q *errorQueryEncoder) Escape(s string) string {
	return s
}
func (q *errorQueryEncoder) UnEscape(s string) (string, error) {
	q.times++
	if q.times >= q.errorAt {
		return "", errQueryEncoder
	}
	return s, nil
}

func TestParser_Unmarshal_DuplicateCall(t *testing.T) {
	parser := NewParser()

	d1 := "desc=bb&Long=200"
	v1 := &testParseChild{}
	_ = parser.Unmarshal([]byte(d1), v1)

	d2 := "desc=a&Long=100"
	v2 := &testParseChild{}
	err := parser.Unmarshal([]byte(d2), v2)
	if err != nil {
		t.Error(err)
	}
	if v2.Description != "a" || v2.Long != 100 {
		t.Error("failed to Unmarshal duplicate call")
	}
}

func TestParser_Unmarshal_NestedStructure(t *testing.T) {
	tem := testEncodedMap{}
	tem["foo"] = "bar"
	tem["baz"] = "quux"
	var data = "Id=1&name=test&child[desc]=c1&child[Long]=10&childPtr[Long]=2&childPtr[Description]=b" +
		"&children[0][desc]=d1&children[1][Long]=12&children[5][desc]=d5&children[5][Long]=50&desc=rtt" +
		"&Params[120]=1&Params[121]=2&status=1&UintPtr=300&tags[]=1&tags[]=2&Int64=64&Uint=22&Uint32=5&Float32=1.3" +
		"&Float64=5.64&Bool=0&inter=ss&time=2024-01-02T18:30:22Z&time_ptr=2024-01-03T11:00:01Z&also_time=2024-01-02T03:04:05Z" +
		"&encoded_string=foo&encoded_string_ptr=bar&tem=" + tem.MarshalQueryParam() + "&tem_ptr=" + tem.MarshalQueryParam() +
		"&strarr3=foo,bar,baz&ignore_decoder[0]=foo&ignore_decoder[2]=baz"
	data = encodeSquareBracket(data)
	v := &testParseInfo{}
	err := Unmarshal([]byte(data), v)

	if err != nil {
		t.Error(err)
	}

	if v.Id != 1 {
		t.Error("Id wrong")
	}

	if v.Name != "test" {
		t.Error("Name wrong")
	}

	if v.Child.Description != "c1" || v.Child.Long != 10 || v.Child.Height != 0 {
		t.Error("Child wrong")
	}

	if v.ChildPtr == nil || v.ChildPtr.Description != "" || v.ChildPtr.Long != 2 || v.ChildPtr.Height != 0 {
		t.Error("ChildPtr wrong")
	}

	if len(v.Children) != 6 {
		t.Error("Children's length is wrong")
	}

	if v.Children[0].Description != "d1" {
		t.Error("Children[0] wrong")
	}

	if v.Children[1].Description != "" || v.Children[1].Long != 12 {
		t.Error("Children[1] wrong")
	}

	if v.Children[2].Description != "" || v.Children[3].Description != "" || v.Children[4].Description != "" {
		t.Error("Children[2,3,4] wrong")
	}

	if v.Children[5].Description != "d5" || v.Children[5].Long != 50 || v.Children[5].Height != 0 {
		t.Error("Children[5] wrong")
	}

	if len(v.Params) != 2 || v.Params[120] != 1 || v.Params[121] != 2 {
		t.Error("Params wrong")
	}

	if v.status != false {
		t.Error("status wrong")
	}

	if v.UintPtr != uintptr(300) {
		t.Error("UintPtr wrong")
	}

	if len(v.Tags) != 2 {
		t.Error("Tags wrong")
	}
	testTime := time.Date(2024, 1, 2, 18, 30, 22, 0, time.UTC)
	if !v.Time.Equal(testTime) {
		t.Errorf("time is wrong: expected %v, got %v", testTime, v.Time)
	}
	testTimePtr := time.Date(2024, 1, 3, 11, 0, 1, 0, time.UTC)
	if v.TimePtr == nil {
		t.Errorf("time_ptr is nil")
	} else if !(*v.TimePtr).Equal(testTimePtr) {
		t.Errorf("time_ptr is wrong: expected %v, got %v", testTimePtr, *v.TimePtr)
	}
	alsoTime := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	if !time.Time(v.AlsoTime).Equal(alsoTime) {
		t.Errorf("time is wrong: expected %v, got %v", alsoTime, v.AlsoTime)
	}
	if v.EncodedString.Str != "foo" {
		t.Errorf("encoded string is wrong: expected %q, got %q", "foo", v.EncodedString.Str)
	}
	if v.EncodedStringPtr == nil {
		t.Error("encoded string pointer is nil")
	} else if v.EncodedStringPtr.Str != "bar" {
		t.Errorf("encoded string pointer is wrong: expected %q, got %q", "bar", v.EncodedStringPtr.Str)
	}
	if len(v.EncodedMap) != 2 {
		t.Error("invalid parse of encoded map")
	}
	if v.EncodedMapPtr == nil || len(*v.EncodedMapPtr) != 2 {
		t.Error("invalid parse of encoded map pointer")
	}
	if v.StrArray3[0] != "foo" || v.StrArray3[1] != "bar" || v.StrArray3[2] != "baz" {
		t.Errorf("invalid parse of testStrArray3: %+v", v.StrArray3)
	}
	if len(v.IgnoreDecoder) != 3 || v.IgnoreDecoder[0] != "foo" || v.IgnoreDecoder[2] != "baz" {
		t.Errorf("invalid parse of IgnoreDecoder: %+v", v.IgnoreDecoder)
	}
}

func TestParser_UnmarshalValues_NestedStructure(t *testing.T) {
	tem := testEncodedMap{}
	tem["foo"] = "bar"
	tem["baz"] = "quux"
	var data = "Id=1&name=test&child[desc]=c1&child[Long]=10&childPtr[Long]=2&childPtr[Description]=b" +
		"&children[0][desc]=d1&children[1][Long]=12&children[5][desc]=d5&children[5][Long]=50&desc=rtt" +
		"&Params[120]=1&Params[121]=2&status=1&UintPtr=300&tags[]=1&tags[]=2&Int64=64&Uint=22&Uint32=5&Float32=1.3" +
		"&Float64=5.64&Bool=0&inter=ss&time=2024-01-02T18:30:22Z&time_ptr=2024-01-03T11:00:01Z&also_time=2024-01-02T03:04:05Z" +
		"&encoded_string=foo&encoded_string_ptr=bar&tem=" + tem.MarshalQueryParam() + "&tem_ptr=" + tem.MarshalQueryParam() +
		"&strarr3=foo,bar,baz&ignore_decoder[0]=foo&ignore_decoder[2]=baz"
	data = encodeSquareBracket(data)
	v := &testParseInfo{}
	values, err := url.ParseQuery(data)
	if err != nil {
		t.Errorf("unexpected error parsing query")
	}
	err = UnmarshalValues(values, v)

	if err != nil {
		t.Error(err)
	}

	if v.Id != 1 {
		t.Error("Id wrong")
	}

	if v.Name != "test" {
		t.Error("Name wrong")
	}

	if v.Child.Description != "c1" || v.Child.Long != 10 || v.Child.Height != 0 {
		t.Error("Child wrong")
	}

	if v.ChildPtr == nil || v.ChildPtr.Description != "" || v.ChildPtr.Long != 2 || v.ChildPtr.Height != 0 {
		t.Error("ChildPtr wrong")
	}

	if len(v.Children) != 6 {
		t.Error("Children's length is wrong")
	}

	if v.Children[0].Description != "d1" {
		t.Error("Children[0] wrong")
	}

	if v.Children[1].Description != "" || v.Children[1].Long != 12 {
		t.Error("Children[1] wrong")
	}

	if v.Children[2].Description != "" || v.Children[3].Description != "" || v.Children[4].Description != "" {
		t.Error("Children[2,3,4] wrong")
	}

	if v.Children[5].Description != "d5" || v.Children[5].Long != 50 || v.Children[5].Height != 0 {
		t.Error("Children[5] wrong")
	}

	if len(v.Params) != 2 || v.Params[120] != 1 || v.Params[121] != 2 {
		t.Error("Params wrong")
	}

	if v.status != false {
		t.Error("status wrong")
	}

	if v.UintPtr != uintptr(300) {
		t.Error("UintPtr wrong")
	}

	if len(v.Tags) != 2 {
		t.Error("Tags wrong")
	}
	testTime := time.Date(2024, 1, 2, 18, 30, 22, 0, time.UTC)
	if !v.Time.Equal(testTime) {
		t.Errorf("time is wrong: expected %v, got %v", testTime, v.Time)
	}
	testTimePtr := time.Date(2024, 1, 3, 11, 0, 1, 0, time.UTC)
	if v.TimePtr == nil {
		t.Errorf("time_ptr is nil")
	} else if !(*v.TimePtr).Equal(testTimePtr) {
		t.Errorf("time_ptr is wrong: expected %v, got %v", testTimePtr, *v.TimePtr)
	}
	alsoTime := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	if !time.Time(v.AlsoTime).Equal(alsoTime) {
		t.Errorf("time is wrong: expected %v, got %v", alsoTime, v.AlsoTime)
	}
	if v.EncodedString.Str != "foo" {
		t.Errorf("encoded string is wrong: expected %q, got %q", "foo", v.EncodedString.Str)
	}
	if v.EncodedStringPtr == nil {
		t.Error("encoded string pointer is nil")
	} else if v.EncodedStringPtr.Str != "bar" {
		t.Errorf("encoded string pointer is wrong: expected %q, got %q", "bar", v.EncodedStringPtr.Str)
	}
	if len(v.EncodedMap) != 2 {
		t.Error("invalid parse of encoded map")
	}
	if v.EncodedMapPtr == nil || len(*v.EncodedMapPtr) != 2 {
		t.Error("invalid parse of encoded map pointer")
	}
	if v.StrArray3[0] != "foo" || v.StrArray3[1] != "bar" || v.StrArray3[2] != "baz" {
		t.Errorf("invalid parse of testStrArray3: %+v", v.StrArray3)
	}
	if len(v.IgnoreDecoder) != 3 || v.IgnoreDecoder[0] != "foo" || v.IgnoreDecoder[2] != "baz" {
		t.Errorf("invalid parse of IgnoreDecoder: %+v", v.IgnoreDecoder)
	}
}

func TestParser_Unmarshal_MissingRequiredFields(t *testing.T) {
	testCases := []struct {
		data         string
		input        interface{}
		missingParam string
	}{{
		data: "",
		input: &struct {
			Str string `query:"str,required"`
		}{},
		missingParam: "str",
	}, {
		data: "",
		input: &struct {
			I64 int64 `query:"i64,required"`
		}{},
		missingParam: "i64",
	}, {
		data: "strszl=foo",
		input: &struct {
			Strs []string `query:"strs,required"`
		}{},
		missingParam: "strs",
	}, {
		data: "",
		input: &struct {
			Children []testParseChildRequired `query:"children"`
		}{},
		missingParam: "",
	}, {
		data: "",
		input: &struct {
			Children []testParseChildRequired `query:"children,required"`
		}{},
		missingParam: "children",
	}, {
		data: "children[3][Long]=0",
		input: &struct {
			Children []testParseChildRequired `query:"children"`
		}{},
		missingParam: "children[3][desc]",
	}, {
		data: "children[3][Long]=0&children[3][desc][foo]=sdf",
		input: &struct {
			Children []testParseChildRequired `query:"children"`
		}{},
		missingParam: "children[3][desc]",
	}, {
		data: "",
		input: &struct {
			Child *testParseChildRequired `query:"child,required"`
		}{},
		missingParam: "child",
	}, {
		data: "",
		input: &struct {
			Child *testParseChildRequired `query:"child"`
		}{},
		missingParam: "",
	}, {
		data: "child[Long]=0",
		input: &struct {
			Child *testParseChildRequired `query:"child"`
		}{},
		missingParam: "child[desc]",
	}, {
		data: "",
		input: &struct {
			Child testParseChildRequired `query:"child,required"`
		}{},
		missingParam: "child",
	}, {
		data: "",
		input: &struct {
			Child testParseChildRequired `query:"child"`
		}{},
		missingParam: "child[desc]",
	}, {
		data: "child[Long]=0",
		input: &struct {
			Child testParseChildRequired `query:"child"`
		}{},
		missingParam: "child[desc]",
	}, {
		data: "metadata[foo]=bar",
		input: &struct {
			Metadata map[string]string `query:"metadata,required"`
		}{},
		missingParam: "",
	}, {
		data: "",
		input: &struct {
			Metadata map[string]string `query:"metadata,required"`
		}{},
		missingParam: "metadata",
	},
	}
	for i, tc := range testCases {
		if err := Unmarshal([]byte(tc.data), tc.input); err == nil {
			if tc.missingParam == "" {
				continue
			}
			t.Errorf("testCases[%d]: expected missing param error for %q", i, tc.missingParam)
		} else if !IsMissingParamError(err) {
			t.Errorf("testCases[%d]: expected missing param error but got error: %v", i, err)
		} else if paramName := ParamNameFromError(err); paramName != tc.missingParam {
			t.Errorf("testCases[%d]: expected missing param %q but got missing param %q", i, tc.missingParam, paramName)
		}
	}
}

func TestParser_Unmarshal_Map(t *testing.T) {
	var m map[string]string
	data := "id=1&name=ab&arr[0]=6d"
	data = encodeSquareBracket(data)
	err := Unmarshal([]byte(data), &m)

	if err != nil {
		t.Error(err)
	}

	if len(m) != 2 {
		t.Error("length is wrong")
	}
	if v1, ok1 := m["id"]; v1 != "1" || !ok1 {
		t.Error("map[id] is wrong")
	}
	if v2, ok2 := m["name"]; v2 != "ab" || !ok2 {
		t.Error("map[iname] is wrong")
	}
	if _, ok3 := m["arr%5B0%5D"]; ok3 {
		t.Error("map[arr%5B0%5D] should not be exist")
	}
}

func TestParser_Unmarshal_Slice(t *testing.T) {
	var slice []int
	slice = make([]int, 0)
	data := "1=20&3=30"
	err := Unmarshal([]byte(data), &slice)

	if err != nil {
		t.Error(err)
	}

	if len(slice) != 4 {
		t.Error("failed to Unmarshal slice")
	}
}

func TestParser_Unmarshal_Array(t *testing.T) {
	var arr [5]int
	data := "1=20&3=30"
	err := Unmarshal([]byte(data), &arr)

	if err != nil {
		t.Error(err)
	}

	if arr[1] != 20 || arr[3] != 30 || arr[0] != 0 {
		t.Error("failed to Unmarshal array")
	}
}

func TestParser_Unmarshal_Array_Failed(t *testing.T) {
	var arr [5]int
	data := "1=20&3=s"
	err := Unmarshal([]byte(data), &arr)

	if err == nil {
		t.Error("dont return error")
	}
}

func TestParser_Unmarshal_StrArray3(t *testing.T) {
	foo := struct {
		Arr3 testStrArray3 `query:"arr3"`
	}{}
	data := "arr3=foo,bar,baz,quux"
	err := Unmarshal([]byte(data), &foo)

	if err == nil {
		t.Error("expected error but got none")
	}
	if ParamNameFromError(err) != "arr3" {
		t.Error("expected error for param \"arr3\" but did not get it")
	}
}

type testParserPoint struct {
	X, Y int
}

type testParserCircle struct {
	testParserPoint
	R int
}

func TestParser_Unmarshal_AnonymousFields(t *testing.T) {
	v := &testParserCircle{}
	data := "X=12&Y=13&R=1"
	err := Unmarshal([]byte(data), &v)

	if err != nil {
		t.Error(err)
	}

	if v.X != 12 || v.Y != 13 || v.R != 1 {
		t.Error("failed to Unmarshal anonymous fields")
	}
}

type testFormat struct {
	Id uint64
	B  rune `query:"b"`
}

func TestParser_Unmarshal_UnmatchedDataFormat(t *testing.T) {
	var data = "Id=1&b=a"
	data = encodeSquareBracket(data)
	v := &testFormat{}
	err := Unmarshal([]byte(data), v)

	if err == nil {
		t.Error("error should not be ignored")
	}
	var errT ErrTranslated
	if !errors.As(err, &errT) {
		t.Errorf("error type is unexpected. %v", err)
	}
}

func TestParser_Unmarshal_UnhandledType(t *testing.T) {
	var data = "Id=1&b=a"
	data = encodeSquareBracket(data)
	v := &map[interface{}]string{}
	err := Unmarshal([]byte(data), v)

	if err == nil {
		t.Error("error should not be ignored")
	}
	var errMKT ErrInvalidMapKeyType
	if !errors.As(err, &errMKT) {
		t.Errorf("error type is unexpected. %v", err)
	}
}

type TestUnhandled struct {
	Id     int
	Params map[string]testFormat
}

func TestParser_Unmarshal_UnhandledType2(t *testing.T) {
	var data = "Id=1&b=a"
	data = encodeSquareBracket(data)
	v := &TestUnhandled{}
	parser := NewParser(WithQueryEncoder(defaultQueryEncoder))
	err := parser.Unmarshal([]byte(data), v)

	if err == nil {
		t.Error("error should not be ignored")
	}
	var errMKT ErrInvalidMapKeyType
	if !errors.As(err, &errMKT) {
		t.Errorf("error type is unexpected. %v", err)
	}
}

type testDate struct {
	Date time.Time `query:"date"`
}

func TestParser_UnmarshalCustomDecoder(t *testing.T) {
	testCases := []struct {
		customDecoders []ValueDecoder
		data           string
		input          interface{}
		expectedOutput interface{}
		paramName      string
	}{{
		customDecoders: []ValueDecoder{testReplacementTimeDecoder{}},
		data:           "date=2024-01-01",
		input:          &testDate{},
		expectedOutput: &testDate{time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
	}, {
		customDecoders: []ValueDecoder{testReplacementTimeDecoder{}},
		data:           "date=2024-01-02T03:04:05Z",
		input:          &testDate{},
		expectedOutput: &testDate{time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)},
	}, {
		customDecoders: []ValueDecoder{testReplacementTimeDecoder{}},
		data:           "date=dasdfasdf",
		input:          &testDate{},
		paramName:      "date",
	}, {
		customDecoders: []ValueDecoder{testReplacementTimeDecoder{}},
		data:           "",
		input:          &testDate{},
		expectedOutput: &testDate{},
	}, {
		customDecoders: []ValueDecoder{testBadDecoder{}},
		data:           "date=2024-01-02T03:04:05Z",
		input:          &testDate{},
		paramName:      "", // no expected output or paramName means error but no param name
	}}
	for i, tc := range testCases {
		p := NewParser()
		for _, vd := range tc.customDecoders {
			p.RegisterValueDecoder(vd)
		}
		p.Unmarshal([]byte(tc.data), tc.input)
		if p.err == nil {
			if tc.expectedOutput != nil && !reflect.DeepEqual(tc.input, tc.expectedOutput) {
				t.Errorf("testCases[%d]: unexpected output: expected %+v, got %+v", i, tc.expectedOutput, tc.input)
			}
			if tc.paramName != "" {
				t.Errorf("testCases[%d]: expected an error for param %q", i, tc.paramName)
			}
		} else if tc.expectedOutput != nil {
			t.Errorf("testCases[%d]: unexpected error: %v", i, p.err)
		} else if pn := ParamNameFromError(p.err); tc.paramName != "" && tc.paramName != pn {
			t.Errorf("testCases[%d]: unexpected param error: expected error for %q, got error for %q ", i, tc.paramName, pn)
		}
	}
}

func TestParser_init(t *testing.T) {
	query := &errorQueryEncoder{errorAt: 1}
	parser := NewParser(WithQueryEncoder(query))
	parser.resetQueryEncoder()
	var data = "Id=1&b=a"
	err := parser.init([]byte(data))
	if err == nil || !errors.Is(err, errQueryEncoder) {
		t.Error("init error")
	}
}

func TestParser_Unmarshal_InitError(t *testing.T) {
	query := &errorQueryEncoder{errorAt: 2}
	parser := NewParser(WithQueryEncoder(query))
	v := &TestUnhandled{}
	var data = "Id=1&b=a"
	err := parser.Unmarshal([]byte(data), v)
	if err == nil || !errors.Is(err, errQueryEncoder) {
		t.Error("init error")
	}
}

func TestParser_Unmarshal_NonPointer(t *testing.T) {
	parser := NewParser()
	var data = "Id=1&b=a"
	v := TestUnhandled{}
	err := parser.Unmarshal([]byte(data), v)
	var errUnmarshal ErrInvalidUnmarshalError
	if !errors.As(err, &errUnmarshal) {
		t.Error("unmatched error")
	}
}

func TestParser_UnmarshalValues_NonPointer(t *testing.T) {
	parser := NewParser()
	data := url.Values{}
	data.Set("Id", "1")
	data.Set("b", "a")
	v := TestUnhandled{}
	err := parser.UnmarshalValues(data, v)
	var errUnmarshal ErrInvalidUnmarshalError
	if !errors.As(err, &errUnmarshal) {
		t.Error("unmatched error")
	}
}

func TestParser_Unmarshal_MapKey_DecodeError(t *testing.T) {
	parser := NewParser()
	parser.RegisterDecodeFunc(reflect.String, nil)
	var data = "Id=1&b=2"
	v := &map[string]int{}
	err := parser.Unmarshal([]byte(data), v)
	var errUT ErrUnhandledType
	if !errors.As(err, &errUT) {
		t.Error("unmatched error")
	}
	paramName := ParamNameFromError(err)
	if paramName != "Id" && paramName != "b" {
		t.Errorf("unexpected param name: %q", paramName)
	}
}

func TestParser_Unmarshal_MapValue_DecodeError(t *testing.T) {
	parser := NewParser()
	parser.RegisterDecodeFunc(reflect.Int, nil)
	var data = "Id=1&b=2"
	v := &map[string]int{}
	err := parser.Unmarshal([]byte(data), v)
	var errUT ErrUnhandledType
	if !errors.As(err, &errUT) {
		t.Error("unmatched error")
	}
	paramName := ParamNameFromError(err)
	if paramName != "Id" && paramName != "b" {
		t.Errorf("unexpected param name: %q", paramName)
	}
}

func TestParser_RegisterDecodeFunc(t *testing.T) {
	parser := NewParser()
	parser.RegisterDecodeFunc(reflect.String, func(s string) (reflect.Value, error) {
		return reflect.ValueOf("11"), nil
	})
	f := parser.getDecodeFunc(reflect.String)
	v, _ := f("bb")
	if v.String() != "11" {
		t.Error("failed to RegisterDecodeFunc")
	}
}

func TestParser_lookupForSlice(t *testing.T) {
	var data = "Tags[s]=1&Tags[]=2"
	data = encodeSquareBracket(data)
	v := &struct {
		Tags []int
	}{}
	err := Unmarshal([]byte(data), v)
	var errNum *strconv.NumError
	if !errors.As(err, &errNum) {
		t.Error("dont failed for wrong slice data")
	}
	paramName := ParamNameFromError(err)
	if paramName != "Tags[s]" {
		t.Errorf("error did not include param name: expected %s, got %s", "Tags[s]", paramName)
	}
}

func TestParser_SliceEmpty(t *testing.T) {
	var data = ""
	data = encodeSquareBracket(data)
	v := &struct {
		Tags []int
	}{}
	_ = Unmarshal([]byte(data), v)
	if len(v.Tags) != 0 {
		t.Error("not empty slice")
	}
}

func TestParser_decode_UnhandledType(t *testing.T) {
	parser := NewParser()
	parser.RegisterDecodeFunc(reflect.String, nil)
	_, err := parser.decode(reflect.TypeOf(""), "s")
	if _, ok := err.(ErrUnhandledType); !ok {
		t.Error("unmatched error")
	}
}

func TestParser_parseForMap_CanSet(t *testing.T) {
	var x = 3.4
	v := reflect.ValueOf(x)
	parser := NewParser()
	parser.parseForMap(v, "")
}

func TestParser_parseForSlice_CanSet(t *testing.T) {
	var x = 3.4
	v := reflect.ValueOf(x)
	parser := NewParser()
	parser.parseForSlice(v, "")
}

// mock multi-layer nested structure,
// BenchmarkUnmarshal-4   	  208219	     14873 ns/op
func BenchmarkUnmarshal(b *testing.B) {
	var data = "Id=1&name=test&child[desc]=c1&child[Long]=10&childPtr[Long]=2&childPtr[Description]=b" +
		"&children[0][desc]=d1&children[1][Long]=12&children[5][desc]=d5&children[5][Long]=50&desc=rtt" +
		"&Params[120]=1&Params[121]=2&status=1&UintPtr=300"
	data = encodeSquareBracket(data)

	for i := 0; i < b.N; i++ {
		v := &testParseInfo{}
		err := Unmarshal([]byte(data), v)
		if err != nil {
			b.Error(err)
		}
	}
}

func encodeSquareBracket(data string) string {
	data = strings.ReplaceAll(data, "[", "%5B")
	data = strings.ReplaceAll(data, "]", "%5D")
	return data
}
