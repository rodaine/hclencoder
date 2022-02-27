package hclencoder

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

var stringTests = []struct {
	input    string // input
	expected string // expected result
}{
	{"\n", "\\n"},
	{"\t", "\\t"},
	{"\r", "\\r"},
	{"\\", "\\\\"},
	{"\"", "\\\""},
	{"${\"test\"}", "${\"test\"}"},
	{"\"${\"test\"}\"", "\\\"${\"test\"}\\\""},
	{"${\"\\ \"\"}", "${\"\\ \"\"}"},
	{"${\"\n\"}", "${\"\n\"}"},
	{"${\"}\\\"}", "${\"}\\\"}"},
	{"${\"${\" \n \"} \n \"}", "${\"${\" \n \"} \n \"}"},
}

func TestStrings(t *testing.T) {
	for _, tt := range stringTests {
		assert.Equal(t, tt.expected, EscapeString(tt.input))
	}
}
