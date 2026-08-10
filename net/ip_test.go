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

type syntheticAddr struct {
	network string
	address string
}

func (a syntheticAddr) Network() string {
	return a.network
}

func (a syntheticAddr) String() string {
	return a.address
}

func TestParseIPFamily(t *testing.T) {
	testCases := []struct {
		name     string
		value    string
		expected IPFamily
		valid    bool
	}{
		{name: "unspecified", value: "", expected: IPFamilyUnspecified, valid: true},
		{name: "IPv4", value: "ipv4", expected: IPFamilyIPv4, valid: true},
		{name: "IPv6", value: "ipv6", expected: IPFamilyIPv6, valid: true},
		{name: "uppercase IPv4", value: "IPV4", expected: IPFamilyIPv4, valid: true},
		{name: "mixed case IPv6", value: "IPv6", expected: IPFamilyIPv6, valid: true},
		{name: "whitespace padded", value: " ipv4 "},
		{name: "unknown", value: "unknown"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			family, err := ParseIPFamily(testCase.value)
			if testCase.valid {
				assert.NoError(t, err)
				assert.Equal(t, testCase.expected, family)
				return
			}
			assert.Error(t, err)
			assert.Empty(t, family)
		})
	}
}

func TestGetLocalIPFromAddrsByFamily(t *testing.T) {
	testCases := []struct {
		name     string
		addrs    []net.Addr
		family   IPFamily
		expected string
	}{
		{
			name: "IPv4 from dual-stack addresses",
			addrs: []net.Addr{
				&net.IPNet{IP: net.ParseIP("2001:db8::1")},
				&net.IPNet{IP: net.ParseIP("192.0.2.10")},
			},
			family:   IPFamilyIPv4,
			expected: "192.0.2.10",
		},
		{
			name: "IPv6 from dual-stack addresses",
			addrs: []net.Addr{
				&net.IPAddr{IP: net.ParseIP("192.0.2.10")},
				&net.IPAddr{IP: net.ParseIP("2001:db8::1")},
			},
			family:   IPFamilyIPv6,
			expected: "2001:db8::1",
		},
		{
			name: "ignores unsupported and malformed addresses",
			addrs: []net.Addr{
				syntheticAddr{network: "synthetic", address: "192.0.2.11/24"},
				(*net.IPNet)(nil),
				(*net.IPAddr)(nil),
				&net.IPNet{IP: net.IP{1, 2, 3}},
				&net.IPAddr{IP: nil},
			},
			family: IPFamilyIPv4,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ip := getLocalIPFromAddrsByFamily(testCase.addrs, testCase.family)
			assert.Equal(t, testCase.expected, ip)
		})
	}
}

type testInterfaceResult struct {
	addrs []net.Addr
	found bool
	err   error
}

type testIPForPodHooks struct {
	interfaces       map[string]testInterfaceResult
	interfaceName    string
	interfaceNameErr error
	interfaceCalls   []string
	nameCalls        int
}

func (h *testIPForPodHooks) interfaceAddrs(name string) ([]net.Addr, bool, error) {
	h.interfaceCalls = append(h.interfaceCalls, name)
	result, ok := h.interfaces[name]
	if !ok {
		return nil, false, nil
	}
	return result.addrs, result.found, result.err
}

func (h *testIPForPodHooks) interfaceNameByIP(net.IP) (string, error) {
	h.nameCalls++
	return h.interfaceName, h.interfaceNameErr
}

func TestGetIPForPod(t *testing.T) {
	dualStack := []net.Addr{
		&net.IPNet{IP: net.ParseIP("2001:db8::2")},
		&net.IPNet{IP: net.ParseIP("192.0.2.20")},
	}
	testCases := []struct {
		name            string
		family          IPFamily
		podIP           string
		hooks           *testIPForPodHooks
		expected        string
		expectedErr     string
		expectedErrIs   error
		expectedNameUse int
		expectedCalls   []string
	}{
		{
			name:   "unspecified family preserves legacy IPv4 storage preference",
			family: IPFamilyUnspecified,
			podIP:  "2001:db8::20",
			hooks: &testIPForPodHooks{
				interfaces: map[string]testInterfaceResult{
					StorageNetworkInterface: {addrs: dualStack, found: true},
				},
			},
			expected:        "192.0.2.20",
			expectedNameUse: 0,
			expectedCalls:   []string{StorageNetworkInterface},
		},
		{
			name:   "unspecified family falls back from IPv6-only storage to PodIP",
			family: IPFamilyUnspecified,
			podIP:  "2001:db8::21",
			hooks: &testIPForPodHooks{
				interfaces: map[string]testInterfaceResult{
					StorageNetworkInterface: {
						addrs: []net.Addr{&net.IPNet{IP: net.ParseIP("2001:db8::2")}},
						found: true,
					},
				},
			},
			expected:        "2001:db8::21",
			expectedNameUse: 0,
			expectedCalls:   []string{StorageNetworkInterface},
		},
		{
			name:   "unspecified family ignores storage read failure and uses PodIP",
			family: IPFamilyUnspecified,
			podIP:  "192.0.2.22",
			hooks: &testIPForPodHooks{
				interfaces: map[string]testInterfaceResult{
					StorageNetworkInterface: {found: true, err: assert.AnError},
				},
			},
			expected:        "192.0.2.22",
			expectedNameUse: 0,
			expectedCalls:   []string{StorageNetworkInterface},
		},
		{
			name:   "unspecified family without an address returns legacy error",
			family: IPFamilyUnspecified,
			hooks: &testIPForPodHooks{
				interfaces: map[string]testInterfaceResult{
					StorageNetworkInterface: {found: false},
				},
			},
			expectedErr:     "can't get a ip from either the specified interface or the environment variable",
			expectedNameUse: 0,
			expectedCalls:   []string{StorageNetworkInterface},
		},
		{
			name:   "IPv4 storage address is authoritative",
			family: IPFamilyIPv4,
			podIP:  "198.51.100.20",
			hooks: &testIPForPodHooks{
				interfaces: map[string]testInterfaceResult{
					StorageNetworkInterface: {addrs: dualStack, found: true},
				},
			},
			expected:        "192.0.2.20",
			expectedNameUse: 0,
			expectedCalls:   []string{StorageNetworkInterface},
		},
		{
			name:   "IPv6 storage address is authoritative",
			family: IPFamilyIPv6,
			podIP:  "192.0.2.20",
			hooks: &testIPForPodHooks{
				interfaces: map[string]testInterfaceResult{
					StorageNetworkInterface: {addrs: dualStack, found: true},
				},
			},
			expected:        "2001:db8::2",
			expectedNameUse: 0,
			expectedCalls:   []string{StorageNetworkInterface},
		},
		{
			name:   "IPv6 storage request rejects IPv4-only storage",
			family: IPFamilyIPv6,
			podIP:  "2001:db8::20",
			hooks: &testIPForPodHooks{
				interfaces: map[string]testInterfaceResult{
					StorageNetworkInterface: {
						addrs: []net.Addr{&net.IPNet{IP: net.ParseIP("192.0.2.21")}},
						found: true,
					},
				},
			},
			expectedErr:     "storage network interface lhnet1 has no ipv6 address",
			expectedNameUse: 0,
			expectedCalls:   []string{StorageNetworkInterface},
		},
		{
			name:   "absent storage uses alternate family on PodIP interface",
			family: IPFamilyIPv6,
			podIP:  "192.0.2.30",
			hooks: &testIPForPodHooks{
				interfaces: map[string]testInterfaceResult{
					StorageNetworkInterface: {found: false},
					"eth0":                  {addrs: dualStack, found: true},
				},
				interfaceName: "eth0",
			},
			expected:        "2001:db8::2",
			expectedNameUse: 1,
			expectedCalls:   []string{StorageNetworkInterface, "eth0"},
		},
		{
			name:   "absent storage rejects opposite-family PodIP fallback",
			family: IPFamilyIPv6,
			podIP:  "192.0.2.31",
			hooks: &testIPForPodHooks{
				interfaces: map[string]testInterfaceResult{
					StorageNetworkInterface: {found: false},
					"eth0": {
						addrs: []net.Addr{&net.IPNet{IP: net.ParseIP("192.0.2.31")}},
						found: true,
					},
				},
				interfaceName: "eth0",
			},
			expectedErr:     "can't get a ip from either the specified interface or the environment variable",
			expectedNameUse: 1,
			expectedCalls:   []string{StorageNetworkInterface, "eth0"},
		},
		{
			name:            "invalid typed family returns an error",
			family:          IPFamily("invalid"),
			podIP:           "2001:db8::35",
			hooks:           &testIPForPodHooks{},
			expectedErr:     `invalid IP family "invalid"`,
			expectedNameUse: 0,
		},
		{
			name:   "storage address read failure propagates",
			family: IPFamilyIPv4,
			hooks: &testIPForPodHooks{
				interfaces: map[string]testInterfaceResult{
					StorageNetworkInterface: {
						found: true,
						err:   assert.AnError,
					},
				},
			},
			expectedErrIs:   assert.AnError,
			expectedNameUse: 0,
			expectedCalls:   []string{StorageNetworkInterface},
		},
		{
			name:   "cluster address read failure propagates",
			family: IPFamilyIPv6,
			podIP:  "192.0.2.36",
			hooks: &testIPForPodHooks{
				interfaces: map[string]testInterfaceResult{
					StorageNetworkInterface: {found: false},
					"eth0": {
						found: true,
						err:   assert.AnError,
					},
				},
				interfaceName: "eth0",
			},
			expectedErrIs:   assert.AnError,
			expectedNameUse: 1,
			expectedCalls:   []string{StorageNetworkInterface, "eth0"},
		},
		{
			name:   "no usable address returns legacy generic error",
			family: IPFamilyIPv4,
			hooks: &testIPForPodHooks{
				interfaces: map[string]testInterfaceResult{
					StorageNetworkInterface: {found: false},
				},
			},
			expectedErr:     "can't get a ip from either the specified interface or the environment variable",
			expectedNameUse: 0,
			expectedCalls:   []string{StorageNetworkInterface},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ip, err := getIPForPod(
				testCase.family,
				testCase.podIP,
				testCase.hooks.interfaceAddrs,
				testCase.hooks.interfaceNameByIP,
			)

			if testCase.expectedErr != "" {
				assert.EqualError(t, err, testCase.expectedErr)
				assert.Empty(t, ip)
			} else if testCase.expectedErrIs != nil {
				assert.ErrorIs(t, err, testCase.expectedErrIs)
				assert.Empty(t, ip)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testCase.expected, ip)
			}
			assert.Equal(t, testCase.expectedNameUse, testCase.hooks.nameCalls)
			assert.Equal(t, testCase.expectedCalls, testCase.hooks.interfaceCalls)
		})
	}
}
func TestGetLocalIPFromAddrsByFamilyIPv6(t *testing.T) {
	testCases := []struct {
		name     string
		addrs    []net.Addr
		expected string
	}{
		{
			name: "skips link-local before global-unicast",
			addrs: []net.Addr{
				&net.IPNet{IP: net.ParseIP("fe80::1")},
				&net.IPNet{IP: net.ParseIP("2001:db8::1")},
			},
			expected: "2001:db8::1",
		},
		{
			name: "rejects link-local only",
			addrs: []net.Addr{
				&net.IPNet{IP: net.ParseIP("fe80::1")},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, getLocalIPFromAddrsByFamily(testCase.addrs, IPFamilyIPv6))
		})
	}
}

func TestIPMatchesFamily(t *testing.T) {
	testCases := []struct {
		name   string
		ip     net.IP
		family IPFamily
		match  bool
	}{
		{name: "nil IPv4", family: IPFamilyIPv4},
		{name: "IPv4", ip: net.ParseIP("192.0.2.1"), family: IPFamilyIPv4, match: true},
		{name: "global IPv6", ip: net.ParseIP("2001:db8::1"), family: IPFamilyIPv6, match: true},
		{name: "link-local IPv6", ip: net.ParseIP("fe80::1"), family: IPFamilyIPv6},
		{name: "IPv4 against IPv6", ip: net.ParseIP("192.0.2.1"), family: IPFamilyIPv6},
		{name: "IPv6 against IPv4", ip: net.ParseIP("2001:db8::1"), family: IPFamilyIPv4},
		{name: "invalid family", ip: net.ParseIP("192.0.2.1"), family: IPFamily("invalid")},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.match, ipMatchesFamily(testCase.ip, testCase.family))
		})
	}
}

func TestGetIPForPodFamilyFallbacks(t *testing.T) {
	const expectedError = "can't get a ip from either the specified interface or the environment variable"
	testCases := []struct {
		name            string
		family          IPFamily
		podIP           string
		hooks           *testIPForPodHooks
		expected        string
		expectedError   string
		expectedNameUse int
		expectedCalls   []string
	}{
		{
			name:   "explicit family rejects malformed PodIP",
			family: IPFamilyIPv4,
			podIP:  "not-an-ip",
			hooks: &testIPForPodHooks{
				interfaces: map[string]testInterfaceResult{
					StorageNetworkInterface: {found: false},
				},
			},
			expectedError: expectedError,
			expectedCalls: []string{StorageNetworkInterface},
		},
		{
			name:   "explicit family rejects opposite-family fallback",
			family: IPFamilyIPv6,
			podIP:  "192.0.2.44",
			hooks: &testIPForPodHooks{
				interfaces: map[string]testInterfaceResult{
					StorageNetworkInterface: {found: false},
					"synthetic0": {
						addrs: []net.Addr{&net.IPNet{IP: net.ParseIP("192.0.2.44")}},
						found: true,
					},
				},
				interfaceName: "synthetic0",
			},
			expectedError:   expectedError,
			expectedNameUse: 1,
			expectedCalls:   []string{StorageNetworkInterface, "synthetic0"},
		},
		{
			name:   "unspecified preserves raw fallback",
			family: IPFamilyUnspecified,
			podIP:  "not-an-ip",
			hooks: &testIPForPodHooks{
				interfaces: map[string]testInterfaceResult{
					StorageNetworkInterface: {found: false},
				},
			},
			expected:      "not-an-ip",
			expectedCalls: []string{StorageNetworkInterface},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ip, err := getIPForPod(
				testCase.family,
				testCase.podIP,
				testCase.hooks.interfaceAddrs,
				testCase.hooks.interfaceNameByIP,
			)

			if testCase.expectedError != "" {
				assert.EqualError(t, err, testCase.expectedError)
				assert.Empty(t, ip)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testCase.expected, ip)
			}
			assert.Equal(t, testCase.expectedNameUse, testCase.hooks.nameCalls)
			assert.Equal(t, testCase.expectedCalls, testCase.hooks.interfaceCalls)
		})
	}
}

func TestGetIPForPodPublicAPIs(t *testing.T) {
	t.Setenv(EnvPodIP, "legacy-pod-ip")
	testCases := []struct {
		name          string
		resolve       func() (string, error)
		expectedError string
	}{
		{
			name:    "unspecified wrapper",
			resolve: GetIPForPod,
		},
		{
			name: "invalid typed family",
			resolve: func() (string, error) {
				return GetIPForPodByFamily(IPFamily("invalid"))
			},
			expectedError: `invalid IP family "invalid"`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ip, err := testCase.resolve()
			if testCase.expectedError != "" {
				assert.EqualError(t, err, testCase.expectedError)
				assert.Empty(t, ip)
				return
			}
			assert.NoError(t, err)
			assert.NotEmpty(t, ip)
		})
	}
}

func TestGetInterfaceNameByIPWithHooks(t *testing.T) {
	listError := errors.New("synthetic interface list failure")
	addressError := errors.New("synthetic address lookup failure")
	testCases := []struct {
		name                   string
		ip                     net.IP
		interfaces             []net.Interface
		interfaceError         error
		addrs                  map[string][]net.Addr
		addressErrors          map[string]error
		expected               string
		expectedError          error
		expectedErrorMessage   string
		expectedInterfaceCalls int
		expectedAddressCalls   int
	}{
		{
			name: "nil IP",
		},
		{
			name:                   "interface list error",
			ip:                     net.ParseIP("2001:db8::2"),
			interfaceError:         listError,
			expectedError:          listError,
			expectedInterfaceCalls: 1,
		},
		{
			name:       "address error",
			ip:         net.ParseIP("2001:db8::2"),
			interfaces: []net.Interface{{Name: "broken0"}},
			addressErrors: map[string]error{
				"broken0": addressError,
			},
			expectedError:          addressError,
			expectedErrorMessage:   "interface broken0 doesn't have address: synthetic address lookup failure",
			expectedInterfaceCalls: 1,
			expectedAddressCalls:   1,
		},
		{
			name:       "match",
			ip:         net.ParseIP("2001:db8::2"),
			interfaces: []net.Interface{{Name: "match0"}},
			addrs: map[string][]net.Addr{
				"match0": {&net.IPNet{IP: net.ParseIP("2001:db8::2")}},
			},
			expected:               "match0",
			expectedInterfaceCalls: 1,
			expectedAddressCalls:   1,
		},
		{
			name:       "no match",
			ip:         net.ParseIP("2001:db8::2"),
			interfaces: []net.Interface{{Name: "ipv40"}, {Name: "other0"}},
			addrs: map[string][]net.Addr{
				"ipv40":  {&net.IPNet{IP: net.ParseIP("192.0.2.1")}},
				"other0": {&net.IPNet{IP: net.ParseIP("2001:db8::3")}},
			},
			expectedInterfaceCalls: 1,
			expectedAddressCalls:   2,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			interfaceCalls := 0
			addressCalls := 0
			name, err := getInterfaceNameByIPWithHooks(
				testCase.ip,
				func() ([]net.Interface, error) {
					interfaceCalls++
					return testCase.interfaces, testCase.interfaceError
				},
				func(iface net.Interface) ([]net.Addr, error) {
					addressCalls++
					return testCase.addrs[iface.Name], testCase.addressErrors[iface.Name]
				},
			)

			if testCase.expectedError != nil {
				assert.ErrorIs(t, err, testCase.expectedError)
				if testCase.expectedErrorMessage != "" {
					assert.EqualError(t, err, testCase.expectedErrorMessage)
				}
				assert.Empty(t, name)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testCase.expected, name)
			}
			assert.Equal(t, testCase.expectedInterfaceCalls, interfaceCalls)
			assert.Equal(t, testCase.expectedAddressCalls, addressCalls)
		})
	}
}
