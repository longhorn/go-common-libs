package utils

import (
	. "gopkg.in/check.v1"

	"github.com/longhorn/go-common-libs/test"
)

func (s *TestSuite) TestGetGRPCAddress(c *C) {
	type testCase struct {
		inputAddr string

		expected string
	}
	testCases := map[string]testCase{
		"GetGRPCAddress(...): prefix `tcp`": {
			inputAddr: "tcp://localhost:1234",
			expected:  "localhost:1234",
		},
		"GetGRPCAddress(...): prefix `http`": {
			inputAddr: "http://localhost:1234",
			expected:  "localhost:1234",
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing utils.%v", testName)

		result := GetGRPCAddress(testCase.inputAddr)
		c.Assert(result, Equals, testCase.expected, Commentf(test.ErrResultFmt, testName))
	}
}
