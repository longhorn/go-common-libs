package utils

import (
	"net"
	"strings"

	. "gopkg.in/check.v1"

	"github.com/longhorn/go-common-libs/test"
)

func (s *TestSuite) GetIsLoopbackHost(c *C) {
	type testCase struct {
		host     string
		expected bool
	}
	testCases := map[string]testCase{
		"IsLoopbackHost(...): localhost": {
			host:     "localhost",
			expected: true,
		},
		"IsLoopbackHost(...): 127.0.0.1": {
			host:     "127.0.0.1",
			expected: true,
		},
		"IsLoopbackHost(...): 0.0.0.0": {
			host:     "0.0.0.0",
			expected: true,
		},
		"IsLoopbackHost(...): ::1": {
			host:     "::1",
			expected: true,
		},
		"IsLoopbackHost(...): ": {
			host:     "",
			expected: true,
		},
		"IsLoopbackHost(...): 8.8.8.8": {
			host:     "8.8.8.8",
			expected: false,
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing utils.%v", testName)

		result := IsLoopbackHost(testCase.host)
		c.Assert(result, Equals, testCase.expected, Commentf(test.ErrResultFmt, testName))
	}
}

func (s *TestSuite) TestGetLocalIPv4fromInterface(c *C) {
	type testCase struct {
		host     string
		expected bool
	}
	testCases := map[string]testCase{
		"GetLocalIPv4fromInterface(...):": {
			host:     "",
			expected: true,
		},
	}
	for testName := range testCases {
		c.Logf("testing utils.%v", testName)

		interfaces, err := net.Interfaces()
		c.Assert(err, IsNil)

		for _, iface := range interfaces {
			ip, err := GetLocalIPv4fromInterface(iface.Name)
			if err != nil {
				c.Assert(strings.Contains(err.Error(), "don't have an IPv4 address"),
					Equals, true, Commentf(test.ErrResultFmt, testName))
				continue
			}
			c.Assert(isIPv4(ip), Equals, true, Commentf(test.ErrResultFmt, testName))
		}
	}
}

func (s *TestSuite) TestGetAnyExternalIP(c *C) {
	type testCase struct {
		host     string
		expected bool
	}
	testCases := map[string]testCase{
		"GetLocalIPv4fromInterface(...):": {
			host:     "",
			expected: true,
		},
	}
	for testName := range testCases {
		c.Logf("testing utils.%v", testName)

		ip, err := GetAnyExternalIP()
		c.Assert(err, IsNil)
		c.Assert(isIPv4(ip), Equals, true, Commentf(test.ErrResultFmt, testName))
	}
}

func isIPv4(ip string) bool {
	return strings.Count(ip, ":") < 2
}
