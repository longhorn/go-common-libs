package utils

import (
	. "gopkg.in/check.v1"

	"github.com/longhorn/go-common-libs/test"
)

func (s *TestSuite) TestIsEngineProcess(c *C) {
	type testCase struct {
		input    string
		expected bool
	}
	testCases := map[string]testCase{
		"engine": {
			input:    "pvc-5a8ee916-5989-46c6-bafc-ddbf7c802499-e-0",
			expected: true,
		},
		"engine-2": {
			input:    "nginx-e-0",
			expected: true,
		},
		"replica": {
			input:    "nginx-r-0",
			expected: false,
		},
		"invalid": {
			input:    "invalid-string",
			expected: false,
		},
		"invalid-2": {
			input:    "-e-0",
			expected: false,
		},
		"invalid-3": {
			input:    "abc-eee-0",
			expected: false,
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing utils.%v", testName)

		result := IsEngineProcess(testCase.input)
		c.Assert(result, Equals, testCase.expected, Commentf(test.ErrResultFmt, testName))
	}
}
