package hclencoder

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

type encoderTest struct {
	ID     string
	Input  interface{}
	Output string
	Error  bool
}

func TestEncoder(t *testing.T) {
	tests := []encoderTest{
		{
			ID:     "empty struct",
			Input:  struct{}{},
			Output: "empty",
		},
		{
			ID: "basic struct",
			Input: struct {
				String string
				Int    int
				Bool   bool
				Float  float64
				Node   string `hcle:"node"`
			}{
				"bar",
				123,
				true,
				4.56,
				"baz",
			},
			Output: "basic",
		},
		{
			ID: "labels changed",
			Input: struct {
				String string `hcl:"foo"`
				Int    int    `hcl:"baz"`
			}{
				"bar",
				123,
			},
			Output: "label-change",
		},
		{
			ID: "primitive list",
			Input: struct {
				Widgets []string
				Gizmos  []int
				Single  []string
			}{
				[]string{"foo", "bar", "baz"},
				[]int{4, 5, 6},
				[]string{"foo"},
			},
			Output: "primitive-lists",
		},
		{
			ID: "nested struct",
			Input: struct {
				Foo  struct{ Bar string }
				Fizz struct{ Buzz float64 }
				Bar  struct {
					Bar  string `hcle:"node"`
					Fizz string
				}
			}{
				struct{ Bar string }{Bar: "baz"},
				struct{ Buzz float64 }{Buzz: 1.23},
				struct {
					Bar  string `hcle:"node"`
					Fizz string
				}{Bar: "baz", Fizz: "bar"},
			},
			Output: "nested-structs",
		},
		{
			ID: "keyed nested struct",
			Input: struct {
				Foo struct {
					Key  string `hcl:",key"`
					Fizz string
				}
			}{
				struct {
					Key  string `hcl:",key"`
					Fizz string
				}{
					"bar",
					"buzz",
				},
			},
			Output: "keyed-nested-structs",
		},
		{
			ID: "multiple keys nested structs",
			Input: struct {
				Foo struct {
					Key      string `hcl:",key"`
					OtherKey string `hcl:",key"`
					Fizz     string
				}
			}{
				struct {
					Key      string `hcl:",key"`
					OtherKey string `hcl:",key"`
					Fizz     string
				}{
					"bar",
					"baz",
					"buzz",
				},
			},
			Output: "multiple-keys-nested-structs",
		},
		{
			ID: "nested struct slice",
			Input: struct {
				Widget []struct {
					Foo string `hcl:"foo,key"`
				}
			}{
				[]struct {
					Foo string `hcl:"foo,key"`
				}{
					{"bar"},
					{"baz"},
				},
			},
			Output: "nested-struct-slice",
		},
		{
			ID: "nested struct slice no key",
			Input: struct {
				Widget []struct {
					Foo string `hcle:"omitempty"`
					Bar string `hcle:"omitempty,node"`
				}
			}{
				Widget: []struct {
					Foo string `hcle:"omitempty"`
					Bar string `hcle:"omitempty,node"`
				}{
					{"bar", ""},
					{"", "fizz"},
					{"baz", ""},
				},
			},
			Output: "nested-struct-slice-no-key",
		},
		{
			ID: "nested slices",
			Input: map[string]interface{}{
				"foo": []interface{}{
					"bar", "baz",
				},
				"bar": []interface{}{
					[]interface{}{
						"bar",
					},
					[]interface{}{
						"baz",
					},
					[]interface{}{
						"buzz",
					},
				},
			},
			Output: "nested-slices",
		},
	}

	for _, test := range tests {
		actual, err := Encode(test.Input)

		if test.Error {
			assert.Error(t, err, test.ID)
		} else {
			expected, ferr := ioutil.ReadFile(fmt.Sprintf("_tests/%s.hcl", test.Output))
			if ferr != nil {
				t.Fatal(test.ID, "- could not read output HCL: ", ferr)
				continue
			}

			assert.NoError(t, err, test.ID)
			assert.EqualValues(t,
				string(expected),
				string(actual),
				fmt.Sprintf("%s\nExpected:\n%s\nActual:\n%s", test.ID, expected, actual),
			)
		}
	}
}
