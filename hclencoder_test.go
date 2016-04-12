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
			}{
				"bar",
				123,
				true,
				4.56,
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
			}{
				[]string{"foo", "bar", "baz"},
				[]int{4, 5, 6},
			},
			Output: "primitive-lists",
		},
		{
			ID: "nested struct",
			Input: struct {
				Foo  struct{ Bar string }
				Fizz struct{ Buzz float64 }
			}{
				struct{ Bar string }{Bar: "baz"},
				struct{ Buzz float64 }{Buzz: 1.23},
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
			assert.EqualValues(t, string(expected), string(actual), test.ID+"\n"+string(actual))
		}
	}
}

func TestEncode_KitchenSink(t *testing.T) {
	type Farm struct {
		Name     string    `hcl:"name"`
		Owned    bool      `hcl:"owned"`
		Location []float64 `hcl:"location"`
	}

	type Farmer struct {
		Name                 string `hcl:"name"`
		Age                  int    `hcl:"age"`
		SocialSecurityNumber string `hcle:"omit"`
	}

	type Animal struct {
		Name  string `hcl:",key"`
		Sound string `hcl:"says" hcle:"omitempty"`
	}

	type Config struct {
		Farm      `hcl:",squash"`
		Farmer    Farmer            `hcl:"farmer"`
		Animals   []Animal          `hcl:"animal"`
		Buildings map[string]string `hcl:"buildings"`
	}

	input := Config{
		Farm: Farm{
			Name:     "Ol' McDonald's Farm",
			Owned:    true,
			Location: []float64{12.34, -5.67},
		},
		Farmer: Farmer{
			Name:                 "Robert Beauregard-Michele McDonald, III",
			Age:                  65,
			SocialSecurityNumber: "please-dont-share-me",
		},
		Animals: []Animal{
			{
				Name:  "cow",
				Sound: "moo",
			},
			{
				Name:  "pig",
				Sound: "oink",
			},
			{
				Name: "rock",
			},
		},
		Buildings: map[string]string{
			"House": "123 Numbers Lane",
			"Barn":  "456 Digits Drive",
		},
	}

	expected, err := ioutil.ReadFile("_tests/kitchen-sink.hcl")
	if err != nil {
		t.Fatal("could not read output HCL: ", err)
		return
	}

	actual, err := Encode(input)
	assert.NoError(t, err)
	assert.EqualValues(t, string(expected), string(actual), string(actual))
}
