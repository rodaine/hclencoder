package hclencoder

import (
	"reflect"
	"sort"
	"testing"

	"github.com/hashicorp/hcl/hcl/ast"
	"github.com/hashicorp/hcl/hcl/token"
	"github.com/stretchr/testify/assert"
)

type encodeFunc func(reflect.Value) (ast.Node, *ast.ObjectKey, error)

type encodeTest struct {
	ID       string
	Input    reflect.Value
	Expected ast.Node
	Key      *ast.ObjectKey
	Error    bool
}

func (test encodeTest) Test(f encodeFunc, t *testing.T) (node ast.Node, key *ast.ObjectKey, err error) {
	node, key, err = f(test.Input)

	if test.Error {
		assert.Error(t, err, test.ID)
		return
	}

	assert.NoError(t, err, test.ID)
	assert.EqualValues(t, test.Key, key, test.ID)
	assert.EqualValues(t, test.Expected, node, test.ID)

	return
}

func RunAll(tests []encodeTest, f encodeFunc, t *testing.T) {
	for _, test := range tests {
		test.Test(f, t)
	}
}

func TestEncode(t *testing.T) {
	tests := []encodeTest{
		{
			ID:       "primitive int",
			Input:    reflect.ValueOf(123),
			Expected: &ast.LiteralType{Token: token.Token{Type: token.NUMBER, Text: "123"}},
		},
		{
			ID:       "primitive string",
			Input:    reflect.ValueOf("foobar"),
			Expected: &ast.LiteralType{Token: token.Token{Type: token.STRING, Text: `"foobar"`}},
		},
		{
			ID:       "primitive bool",
			Input:    reflect.ValueOf(true),
			Expected: &ast.LiteralType{Token: token.Token{Type: token.BOOL, Text: "true"}},
		},
		{
			ID:       "primitive float",
			Input:    reflect.ValueOf(float64(1.23)),
			Expected: &ast.LiteralType{Token: token.Token{Type: token.FLOAT, Text: "1.23"}},
		},
		{
			ID:    "list",
			Input: reflect.ValueOf([]int{1, 2, 3}),
			Expected: &ast.ListType{List: []ast.Node{
				&ast.LiteralType{Token: token.Token{Type: token.NUMBER, Text: "1"}},
				&ast.LiteralType{Token: token.Token{Type: token.NUMBER, Text: "2"}},
				&ast.LiteralType{Token: token.Token{Type: token.NUMBER, Text: "3"}},
			}},
		},
		{
			ID:    "map",
			Input: reflect.ValueOf(map[string]int{"foo": 1, "bar": 2}),
			Expected: &ast.ObjectType{List: &ast.ObjectList{Items: []*ast.ObjectItem{
				&ast.ObjectItem{
					Keys: []*ast.ObjectKey{{Token: token.Token{Type: token.IDENT, Text: "bar"}}},
					Val:  &ast.LiteralType{Token: token.Token{Type: token.NUMBER, Text: "2"}},
				},
				&ast.ObjectItem{
					Keys: []*ast.ObjectKey{{Token: token.Token{Type: token.IDENT, Text: "foo"}}},
					Val:  &ast.LiteralType{Token: token.Token{Type: token.NUMBER, Text: "1"}},
				},
			}}},
		},
		{
			ID:    "struct",
			Input: reflect.ValueOf(TestStruct{Bar: "fizzbuzz"}),
			Expected: &ast.ObjectType{List: &ast.ObjectList{Items: []*ast.ObjectItem{
				&ast.ObjectItem{
					Keys: []*ast.ObjectKey{{Token: token.Token{Type: token.IDENT, Text: "Bar"}}},
					Val:  &ast.LiteralType{Token: token.Token{Type: token.STRING, Text: `"fizzbuzz"`}},
				},
			}}},
		},
	}

	RunAll(tests, encode, t)
}

func TestEncodePrimitive(t *testing.T) {
	tests := []encodeTest{
		{
			ID:       "int",
			Input:    reflect.ValueOf(123),
			Expected: &ast.LiteralType{Token: token.Token{Type: token.NUMBER, Text: "123"}},
		},
		{
			ID:       "string - never ident",
			Input:    reflect.ValueOf("foobar"),
			Expected: &ast.LiteralType{Token: token.Token{Type: token.STRING, Text: `"foobar"`}},
		},
		{
			ID:       "uint",
			Input:    reflect.ValueOf(uint(1)),
			Expected: &ast.LiteralType{Token: token.Token{Type: token.NUMBER, Text: "1"}},
		},
	}

	RunAll(tests, encodePrimitive, t)
}

func TestEncodeList(t *testing.T) {
	tests := []encodeTest{
		{
			ID:    "primitive - int",
			Input: reflect.ValueOf([]int{1, 2, 3}),
			Expected: &ast.ListType{List: []ast.Node{
				&ast.LiteralType{Token: token.Token{Type: token.NUMBER, Text: "1"}},
				&ast.LiteralType{Token: token.Token{Type: token.NUMBER, Text: "2"}},
				&ast.LiteralType{Token: token.Token{Type: token.NUMBER, Text: "3"}},
			}},
		},
		{
			ID:    "primitive - string - never ident",
			Input: reflect.ValueOf([]string{"foo", "bar"}),
			Expected: &ast.ListType{List: []ast.Node{
				&ast.LiteralType{Token: token.Token{Type: token.STRING, Text: `"foo"`}},
				&ast.LiteralType{Token: token.Token{Type: token.STRING, Text: `"bar"`}},
			}},
		},
		{
			ID:       "primitive - nil",
			Input:    reflect.ValueOf([]string(nil)),
			Expected: &ast.ListType{List: []ast.Node{}},
		},
		{
			ID:    "primitive - nil item",
			Input: reflect.ValueOf([]*string{strAddr("fizz"), nil, strAddr("buzz")}),
			Expected: &ast.ListType{List: []ast.Node{
				&ast.LiteralType{Token: token.Token{Type: token.STRING, Text: `"fizz"`}},
				&ast.LiteralType{Token: token.Token{Type: token.STRING, Text: `"buzz"`}},
			}},
		},
		{
			ID:    "primitive - uint",
			Input: reflect.ValueOf([]uint{123}),
			Expected: &ast.ListType{List: []ast.Node{
				&ast.LiteralType{Token: token.Token{Type: token.NUMBER, Text: "123"}},
			}},
			//Error: true,
		},
		{
			ID:    "block",
			Input: reflect.ValueOf([]TestStruct{{}, {Bar: "fizzbuzz"}}),
			Expected: &ast.ListType{List: []ast.Node{
				&ast.ObjectType{List: &ast.ObjectList{
					Items: []*ast.ObjectItem{{
						Keys: []*ast.ObjectKey{{Token: token.Token{Type: token.IDENT, Text: "Bar"}}},
						Val:  &ast.LiteralType{Token: token.Token{Type: token.STRING, Text: `""`}},
					}},
				}},
				&ast.ObjectType{List: &ast.ObjectList{
					Items: []*ast.ObjectItem{{
						Keys: []*ast.ObjectKey{{Token: token.Token{Type: token.IDENT, Text: "Bar"}}},
						Val:  &ast.LiteralType{Token: token.Token{Type: token.STRING, Text: `"fizzbuzz"`}},
					}},
				}},
			}},
		},
		{
			ID:       "block - nil",
			Input:    reflect.ValueOf([]TestStruct(nil)),
			Expected: &ast.ObjectList{Items: []*ast.ObjectItem{}},
		},
		{
			ID:    "block - nil item",
			Input: reflect.ValueOf([]*TestStruct{&TestStruct{}, nil, &TestStruct{Bar: "fizzbuzz"}}),
			Expected: &ast.ListType{List: []ast.Node{
				&ast.ObjectType{List: &ast.ObjectList{
					Items: []*ast.ObjectItem{{
						Keys: []*ast.ObjectKey{{Token: token.Token{Type: token.IDENT, Text: "Bar"}}},
						Val:  &ast.LiteralType{Token: token.Token{Type: token.STRING, Text: `""`}},
					}},
				}},
				&ast.ObjectType{List: &ast.ObjectList{
					Items: []*ast.ObjectItem{{
						Keys: []*ast.ObjectKey{{Token: token.Token{Type: token.IDENT, Text: "Bar"}}},
						Val:  &ast.LiteralType{Token: token.Token{Type: token.STRING, Text: `"fizzbuzz"`}},
					}},
				}},
			}},
		},
		{
			ID:    "block - interface",
			Input: reflect.ValueOf([]TestInterface{TestStruct{}, TestStruct{Bar: "fizzbuzz"}}),
			Expected: &ast.ListType{List: []ast.Node{
				&ast.ObjectType{List: &ast.ObjectList{
					Items: []*ast.ObjectItem{{
						Keys: []*ast.ObjectKey{{Token: token.Token{Type: token.IDENT, Text: "Bar"}}},
						Val:  &ast.LiteralType{Token: token.Token{Type: token.STRING, Text: `""`}},
					}},
				}},
				&ast.ObjectType{List: &ast.ObjectList{
					Items: []*ast.ObjectItem{{
						Keys: []*ast.ObjectKey{{Token: token.Token{Type: token.IDENT, Text: "Bar"}}},
						Val:  &ast.LiteralType{Token: token.Token{Type: token.STRING, Text: `"fizzbuzz"`}},
					}},
				}},
			}},
		},
		{
			ID:    "block - key field",
			Input: reflect.ValueOf([]KeyStruct{{Bar: "foo"}}),
			Expected: &ast.ObjectList{Items: []*ast.ObjectItem{&ast.ObjectItem{
				Keys: []*ast.ObjectKey{&ast.ObjectKey{Token: token.Token{Type: token.STRING, Text: `"foo"`}}},
				Val:  &ast.ObjectType{List: &ast.ObjectList{Items: []*ast.ObjectItem{}}},
			}}},
		},
		{
			ID:    "block - invalid",
			Input: reflect.ValueOf([]InvalidStruct{{}}),
			Error: true,
		},
	}

	RunAll(tests, encodeList, t)
}

func TestEncodeMap(t *testing.T) {
	tests := []encodeTest{
		{
			ID:    "primitive",
			Input: reflect.ValueOf(map[string]int{"foo": 1, "bar": 2}),
			Expected: &ast.ObjectType{List: &ast.ObjectList{Items: []*ast.ObjectItem{
				&ast.ObjectItem{
					Keys: []*ast.ObjectKey{{Token: token.Token{Type: token.IDENT, Text: "bar"}}},
					Val:  &ast.LiteralType{Token: token.Token{Type: token.NUMBER, Text: "2"}},
				},
				&ast.ObjectItem{
					Keys: []*ast.ObjectKey{{Token: token.Token{Type: token.IDENT, Text: "foo"}}},
					Val:  &ast.LiteralType{Token: token.Token{Type: token.NUMBER, Text: "1"}},
				},
			}}},
		},
		{
			ID:    "invalid key",
			Input: reflect.ValueOf(map[int]string{}),
			Error: true,
		},
		{
			ID:    "invalid value",
			Input: reflect.ValueOf(map[string]InvalidStruct{"foo": InvalidStruct{}}),
			Error: true,
		},
		{
			ID:       "nil value",
			Input:    reflect.ValueOf(map[string]*TestStruct{"fizz": nil}),
			Expected: &ast.ObjectType{List: &ast.ObjectList{Items: []*ast.ObjectItem{}}},
		},
		{
			ID:    "key field",
			Input: reflect.ValueOf(map[string]KeyStruct{"fizz": {Bar: "buzz"}}),
			Expected: &ast.ObjectType{List: &ast.ObjectList{Items: []*ast.ObjectItem{
				&ast.ObjectItem{
					Keys: []*ast.ObjectKey{
						{Token: token.Token{Type: token.IDENT, Text: "fizz"}},
						{Token: token.Token{Type: token.STRING, Text: `"buzz"`}},
					},
					Val: &ast.ObjectType{List: &ast.ObjectList{Items: []*ast.ObjectItem{}}},
				},
			}}},
		},
		{
			ID: "keyed list",
			Input: reflect.ValueOf(map[string][]map[string]interface{}{
				"obj1": {
					{"foo": "bar"},
					{"boo": "hoo"},
				},
				"obj2": {
					{"foo": "bar"},
					{"boo": "hoo"},
				},
			}),
			Expected: &ast.ObjectType{List: &ast.ObjectList{Items: []*ast.ObjectItem{
				&ast.ObjectItem{
					Keys: []*ast.ObjectKey{{Token: token.Token{Type: token.IDENT, Text: "obj1"}}},
					Val: &ast.ListType{List: []ast.Node{
						&ast.ObjectType{List: &ast.ObjectList{Items: []*ast.ObjectItem{
							{
								Keys: []*ast.ObjectKey{{Token: token.Token{Type: token.IDENT, Text: "foo"}}},
								Val:  &ast.LiteralType{Token: token.Token{Type: token.STRING, Text: `"bar"`}},
							},
						}}},
						&ast.ObjectType{List: &ast.ObjectList{Items: []*ast.ObjectItem{
							{
								Keys: []*ast.ObjectKey{{Token: token.Token{Type: token.IDENT, Text: "boo"}}},
								Val:  &ast.LiteralType{Token: token.Token{Type: token.STRING, Text: `"hoo"`}},
							},
						}}},
					}},
				},
				&ast.ObjectItem{
					Keys: []*ast.ObjectKey{{Token: token.Token{Type: token.IDENT, Text: "obj2"}}},
					Val: &ast.ListType{List: []ast.Node{
						&ast.ObjectType{List: &ast.ObjectList{Items: []*ast.ObjectItem{
							{
								Keys: []*ast.ObjectKey{{Token: token.Token{Type: token.IDENT, Text: "foo"}}},
								Val:  &ast.LiteralType{Token: token.Token{Type: token.STRING, Text: `"bar"`}},
							},
						}}},
						&ast.ObjectType{List: &ast.ObjectList{Items: []*ast.ObjectItem{
							{
								Keys: []*ast.ObjectKey{{Token: token.Token{Type: token.IDENT, Text: "boo"}}},
								Val:  &ast.LiteralType{Token: token.Token{Type: token.STRING, Text: `"hoo"`}},
							},
						}}},
					}},
				},
			}}},
		},
	}

	RunAll(tests, encodeMap, t)
}

func TestEncodeStruct(t *testing.T) {
	tests := []encodeTest{
		{
			ID:    "basic",
			Input: reflect.ValueOf(TestStruct{Bar: "fizzbuzz"}),
			Expected: &ast.ObjectType{List: &ast.ObjectList{Items: []*ast.ObjectItem{
				&ast.ObjectItem{
					Keys: []*ast.ObjectKey{{Token: token.Token{Type: token.IDENT, Text: "Bar"}}},
					Val:  &ast.LiteralType{Token: token.Token{Type: token.STRING, Text: `"fizzbuzz"`}},
				},
			}}},
		},
		{
			ID:       "debug fields",
			Input:    reflect.ValueOf(DebugStruct{Decoded: []string{}, Unused: []string{}}),
			Expected: &ast.ObjectType{List: &ast.ObjectList{Items: []*ast.ObjectItem{}}},
		},
		{
			ID:       "omit field",
			Input:    reflect.ValueOf(OmitStruct{"foo"}),
			Expected: &ast.ObjectType{List: &ast.ObjectList{Items: []*ast.ObjectItem{}}},
		},
		{
			ID:       "omitempty field - empty",
			Input:    reflect.ValueOf(OmitEmptyStruct{}),
			Expected: &ast.ObjectType{List: &ast.ObjectList{Items: []*ast.ObjectItem{}}},
		},
		{
			ID:    "omitempty field - not empty",
			Input: reflect.ValueOf(OmitEmptyStruct{"foo"}),
			Expected: &ast.ObjectType{List: &ast.ObjectList{Items: []*ast.ObjectItem{
				&ast.ObjectItem{
					Keys: []*ast.ObjectKey{{Token: token.Token{Type: token.IDENT, Text: "Bar"}}},
					Val:  &ast.LiteralType{Token: token.Token{Type: token.STRING, Text: `"foo"`}},
				},
			}}},
		},
		{
			ID:       "nil field",
			Input:    reflect.ValueOf(NillableStruct{}),
			Expected: &ast.ObjectType{List: &ast.ObjectList{Items: []*ast.ObjectItem{}}},
		},
		{
			ID:    "invalid key type",
			Input: reflect.ValueOf(InvalidKeyStruct{123}),
			Error: true,
		},
		{
			ID:    "squash anonymous field",
			Input: reflect.ValueOf(SquashStruct{TestStruct: TestStruct{"foo"}}),
			Expected: &ast.ObjectType{List: &ast.ObjectList{Items: []*ast.ObjectItem{
				&ast.ObjectItem{
					Keys: []*ast.ObjectKey{{Token: token.Token{Type: token.IDENT, Text: "Bar"}}},
					Val:  &ast.LiteralType{Token: token.Token{Type: token.STRING, Text: `"foo"`}},
				},
			}}},
		},
		{
			ID:    "keyed child struct",
			Input: reflect.ValueOf(KeyChildStruct{Foo: KeyStruct{Bar: "baz"}}),
			Expected: &ast.ObjectType{List: &ast.ObjectList{Items: []*ast.ObjectItem{
				&ast.ObjectItem{
					Keys: []*ast.ObjectKey{{Token: token.Token{Type: token.IDENT, Text: "Foo"}}, {Token: token.Token{Type: token.STRING, Text: `"baz"`}}},
					Val:  &ast.ObjectType{List: &ast.ObjectList{Items: []*ast.ObjectItem{}}},
				},
			}}},
		},
		{
			ID:    "nested unkeyed struct slice",
			Input: reflect.ValueOf(struct{ Foo []TestStruct }{[]TestStruct{{"Test"}}}),
			Expected: &ast.ObjectType{List: &ast.ObjectList{Items: []*ast.ObjectItem{
				&ast.ObjectItem{
					Keys: []*ast.ObjectKey{{Token: token.Token{Type: token.IDENT, Text: "Foo"}}},
					Val: &ast.ListType{List: []ast.Node{&ast.ObjectType{
						List: &ast.ObjectList{
							Items: []*ast.ObjectItem{{
								Keys: []*ast.ObjectKey{{Token: token.Token{Type: token.IDENT, Text: "Bar"}}},
								Val:  &ast.LiteralType{Token: token.Token{Type: token.STRING, Text: `"Test"`}},
							}},
						}}},
					},
				},
			}}},
		},
	}

	RunAll(tests, encodeStruct, t)
}

func TestTokenize(t *testing.T) {
	is := assert.New(t)

	tests := []struct {
		ID       string
		Input    reflect.Value
		Ident    bool
		Expected token.Token
		Err      bool
	}{
		{
			"bool",
			reflect.ValueOf(true),
			false,
			token.Token{Type: token.BOOL, Text: "true"},
			false,
		},
		{
			"int",
			reflect.ValueOf(123),
			false,
			token.Token{Type: token.NUMBER, Text: "123"},
			false,
		},
		{
			"float",
			reflect.ValueOf(float64(4.56)),
			false,
			token.Token{Type: token.FLOAT, Text: "4.56"},
			false,
		},
		{
			"float - superfluous",
			reflect.ValueOf(float64(78.9000000000)),
			false,
			token.Token{Type: token.FLOAT, Text: "78.9"},
			false,
		},
		{
			"float - scientific notation",
			reflect.ValueOf(float64(1234567890)),
			false,
			token.Token{Type: token.FLOAT, Text: "1.23456789e+09"},
			false,
		},
		{
			"string",
			reflect.ValueOf("foobar"),
			false,
			token.Token{Type: token.STRING, Text: `"foobar"`},
			false,
		},
		{
			"ident",
			reflect.ValueOf("fizzbuzz"),
			true,
			token.Token{Type: token.IDENT, Text: "fizzbuzz"},
			false,
		},
	}

	for _, test := range tests {
		tkn, err := tokenize(test.Input, test.Ident)
		if test.Err {
			is.Error(err, test.ID)
		} else {
			is.NoError(err, test.ID)
			is.EqualValues(test.Expected, tkn, test.ID)
		}
	}
}

func TestExtractFieldMeta(t *testing.T) {
	is := assert.New(t)

	fieldName := "Foo"

	tests := []struct {
		Tag      string
		Expected fieldMeta
	}{
		{
			"",
			fieldMeta{name: fieldName},
		},
		{
			`hcl:"bar"`,
			fieldMeta{name: "bar"},
		},
		{
			`hcl:"bar,key"`,
			fieldMeta{name: "bar", key: true},
		},
		{
			`hcl:",squash"`,
			fieldMeta{name: fieldName, squash: true},
		},
		{
			`hcl:",decodedFields,unusedKeys"`,
			fieldMeta{name: fieldName, decodedFields: true, unusedKeys: true},
		},
		{
			`hcl:",key" hcle:"omit"`,
			fieldMeta{name: fieldName, key: true, omit: true},
		},
		{
			`hcle:"omitempty"`,
			fieldMeta{name: fieldName, omitEmpty: true},
		},
	}

	for _, test := range tests {
		input := reflect.StructField{
			Name: fieldName,
			Tag:  reflect.StructTag(test.Tag),
		}
		is.EqualValues(test.Expected, extractFieldMeta(input))
	}

	input := reflect.StructField{
		Anonymous: true,
		Type:      reflect.TypeOf(TestStruct{}),
	}
	expected := fieldMeta{
		name:      input.Type.Name(),
		anonymous: true,
	}
	is.EqualValues(expected, extractFieldMeta(input))
}

func TestDeref(t *testing.T) {
	is := assert.New(t)

	var IFace TestInterface
	IFace = TestStruct{"baz"}
	var nilIFace TestInterface

	var nilPtr *TestStruct

	tests := []struct {
		Input    interface{}
		Expected interface{}
		IsNil    bool
		Message  string
	}{
		{
			IFace,
			TestStruct{"baz"},
			false,
			"interface",
		},
		{
			&TestStruct{"fizz"},
			TestStruct{"fizz"},
			false,
			"pointer",
		},
		{
			nil,
			nil,
			true,
			"nil",
		},
		{
			nilIFace,
			nil,
			true,
			"interface - nil",
		},
		{
			nilPtr,
			nil,
			true,
			"pointer - nil",
		},
		{
			[]string{"foo", "bar"},
			[]string{"foo", "bar"},
			false,
			"slice",
		},
		{
			[]string(nil),
			nil,
			true,
			"slice - nil",
		},
	}

	for _, test := range tests {
		expected := reflect.ValueOf(test.Expected)
		val, isNil := deref(reflect.ValueOf(test.Input))

		if test.IsNil {
			is.Equal(test.IsNil, isNil, "%s", test.Message)
		} else {
			is.EqualValues(expected.Type(), val.Type(), "%s", test.Message)
		}
	}
}

func TestObjectItems(t *testing.T) {
	noKeys := &ast.ObjectItem{}
	bar := &ast.ObjectItem{Keys: []*ast.ObjectKey{{Token: token.Token{Text: "bar"}}}}
	foo := &ast.ObjectItem{Keys: []*ast.ObjectKey{{Token: token.Token{Text: "foo"}}}}
	foobar := &ast.ObjectItem{Keys: []*ast.ObjectKey{{Token: token.Token{Text: "foo"}}, {Token: token.Token{Text: "bar"}}}}

	oi := objectItems{
		foobar,
		foo,
		bar,
		noKeys,
	}

	expected := objectItems{
		noKeys,
		bar,
		foo,
		foobar,
	}

	sort.Sort(oi)
	assert.EqualValues(t, expected, oi)
}

type TestInterface interface {
	Foo()
}

type TestStruct struct {
	Bar string
}

func (TestStruct) Foo() {}

type KeyStruct struct {
	Bar string `hcl:",key"`
}

func (KeyStruct) Foo() {}

type KeyChildStruct struct {
	Foo KeyStruct
}

type DebugStruct struct {
	Decoded []string `hcl:",decodedFields"`
	Unused  []string `hcl:",unusedKeys"`
}

type OmitStruct struct {
	Bar string `hcle:"omit"`
}

type OmitEmptyStruct struct {
	Bar string `hcle:"omitempty"`
}

type InvalidStruct struct {
	Chan chan struct{}
}

type InvalidKeyStruct struct {
	Bar int `hcl:",key"`
}

type NillableStruct struct {
	Bar *string
}

type SquashStruct struct {
	TestStruct `hcl:",squash"`
}

func strAddr(s string) *string {
	return &s
}
