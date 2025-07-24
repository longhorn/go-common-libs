package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	. "gopkg.in/check.v1"

	"github.com/longhorn/go-common-libs/test"
)

func TestGetGRPCAddress(t *testing.T) {
	type testCase struct {
		inputAddr string

		expected string
	}
	testCases := map[string]testCase{
		"Prefix tcp": {
			inputAddr: "tcp://localhost:1234",
			expected:  "localhost:1234",
		},
		"Prefix http": {
			inputAddr: "http://localhost:1234",
			expected:  "localhost:1234",
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			result := GetGRPCAddress(testCase.inputAddr)
			assert.Equal(t, testCase.expected, result, Commentf(test.ErrResultFmt, testName))
		})
	}
}
