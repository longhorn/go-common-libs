package net

import (
	"errors"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	. "gopkg.in/check.v1"

	"github.com/longhorn/go-common-libs/test"
)

func TestGetAnyExternalIPSelection(t *testing.T) {
	errListInterfaces := errors.New("failed to list interfaces")
	errListAddrs := errors.New("failed to list addresses")

	testCases := []struct {
		name        string
		interfaces  []net.Interface
		addrs       map[int][]net.Addr
		listErr     error
		addrErr     map[int]error
		expected    string
		wantErr     bool
		expectedErr error
	}{
		{
			name: "prefers IPv4 over IPv6",
			interfaces: []net.Interface{
				{Index: 1, Flags: net.FlagUp},
				{Index: 2, Flags: net.FlagUp},
			},
			addrs: map[int][]net.Addr{
				1: {&net.IPNet{IP: net.ParseIP("2001:db8::10")}},
				2: {&net.IPNet{IP: net.ParseIP("192.0.2.10")}},
			},
			expected: "192.0.2.10",
		},
		{
			name:       "falls back to global unicast IPv6",
			interfaces: []net.Interface{{Index: 1, Flags: net.FlagUp}},
			addrs: map[int][]net.Addr{
				1: {&net.IPAddr{IP: net.ParseIP("fd00:168:1::2")}},
			},
			expected: "fd00:168:1::2",
		},
		{
			name:       "rejects unusable IPv6 addresses",
			interfaces: []net.Interface{{Index: 1, Flags: net.FlagUp}},
			addrs: map[int][]net.Addr{
				1: {
					&net.IPNet{IP: net.ParseIP("fe80::1")},
					&net.IPNet{IP: net.ParseIP("ff02::1")},
				},
			},
			wantErr: true,
		},
		{
			name: "skips down and loopback interfaces",
			interfaces: []net.Interface{
				{Index: 1},
				{Index: 2, Flags: net.FlagUp | net.FlagLoopback},
			},
			addrs: map[int][]net.Addr{
				1: {&net.IPNet{IP: net.ParseIP("192.0.2.10")}},
				2: {&net.IPNet{IP: net.ParseIP("2001:db8::10")}},
			},
			wantErr: true,
		},
		{
			name:        "propagates interface list error",
			listErr:     errListInterfaces,
			expectedErr: errListInterfaces,
		},
		{
			name:        "propagates address list error",
			interfaces:  []net.Interface{{Index: 1, Flags: net.FlagUp}},
			addrErr:     map[int]error{1: errListAddrs},
			expectedErr: errListAddrs,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual, err := getAnyExternalIP(
				func() ([]net.Interface, error) {
					return testCase.interfaces, testCase.listErr
				},
				func(iface net.Interface) ([]net.Addr, error) {
					return testCase.addrs[iface.Index], testCase.addrErr[iface.Index]
				},
			)

			assert.Equal(t, testCase.expected, actual)
			switch {
			case testCase.expectedErr != nil:
				assert.ErrorIs(t, err, testCase.expectedErr)
			case testCase.wantErr:
				assert.Error(t, err)
			default:
				assert.NoError(t, err)
			}
		})
	}
}

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

func TestGetLocalIPFromInterface(t *testing.T) {
	interfaces, err := net.Interfaces()
	assert.Nil(t, err)

	for _, iface := range interfaces {
		ip, err := GetLocalIPFromInterface(iface.Name)
		if err != nil {
			assert.True(t, strings.Contains(err.Error(), "doesn't have an IPv4 or a global unicast IPv6 address"))
			continue
		}

		parsed := net.ParseIP(ip)
		assert.NotNil(t, parsed, Commentf("interface %v returned an unparsable IP %q", iface.Name, ip))
		assert.True(t, parsed.To4() != nil || parsed.IsGlobalUnicast(),
			Commentf("interface %v returned a non-routable IPv6 address %q", iface.Name, ip))
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
			assert.NotNil(t, net.ParseIP(ip), Commentf(test.ErrResultFmt, testName))
		})
	}
}

func isIPv4(ip string) bool {
	return strings.Count(ip, ":") < 2
}
