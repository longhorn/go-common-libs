package net

import (
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	. "gopkg.in/check.v1"

	"github.com/longhorn/go-common-libs/test"
)

func TestIsLoopbackHost(t *testing.T) {
	type testCase struct {
		host     string
		expected bool
	}

	testCases := map[string]testCase{
		"Localhost": {
			host:     "localhost",
			expected: true,
		},
		"127.0.0.1": {
			host:     "127.0.0.1",
			expected: true,
		},
		"0.0.0.0": {
			host:     "0.0.0.0",
			expected: true,
		},
		"::1": {
			host:     "::1",
			expected: true,
		},
		"Empty": {
			host:     "",
			expected: true,
		},
		"8.8.8.8": {
			host:     "8.8.8.8",
			expected: false,
		},
	}

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			result := IsLoopbackHost(testCase.host)
			assert.Equal(t, testCase.expected, result, Commentf(test.ErrResultFmt, testName))
		})
	}
}

func TestGetLocalIPv4fromInterface(t *testing.T) {
	type testCase struct {
		host     string
		expected bool
	}

	testCases := map[string]testCase{
		"Local": {
			host:     "",
			expected: true,
		},
	}

	for testName := range testCases {
		t.Run(testName, func(t *testing.T) {
			interfaces, err := net.Interfaces()
			assert.Nil(t, err, Commentf(test.ErrErrorFmt, testName, err))

			for _, iface := range interfaces {
				ip, err := GetLocalIPv4fromInterface(iface.Name)
				if err != nil {
					assert.True(t, strings.Contains(err.Error(), "don't have an IPv4 address"), Commentf(test.ErrResultFmt, testName))
					continue
				}

				assert.True(t, isIPv4(ip), Commentf(test.ErrResultFmt, testName))
			}
		})
	}
}

func TestGetAnyExternalIP(t *testing.T) {
	type testCase struct {
		host     string
		expected bool
	}

	testCases := map[string]testCase{
		"Local": {
			host:     "",
			expected: true,
		},
	}

	for testName := range testCases {
		t.Run(testName, func(t *testing.T) {
			ip, err := GetAnyExternalIP()
			assert.Nil(t, err, Commentf(test.ErrErrorFmt, testName, err))
			assert.True(t, isIPv4(ip), Commentf(test.ErrResultFmt, testName))
		})
	}
}

func isIPv4(ip string) bool {
	return strings.Count(ip, ":") < 2
}
