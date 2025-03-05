package ns

import (
	"fmt"

	. "gopkg.in/check.v1"

	"github.com/longhorn/go-common-libs/types"
)

func testCaseGetArch(c *C) map[string]testCaseNamespaceMethods {
	return map[string]testCaseNamespaceMethods{
		"GetGetArch(...)": {
			method: func(args ...interface{}) (interface{}, error) {
				return GetArch()
			},
			mockResult: "result",
		},
		"GetArch(...): failed to run": {
			method: func(args ...interface{}) (interface{}, error) {
				return GetArch()
			},
			mockError:   fmt.Errorf("failed"),
			expectError: true,
		},
		"GetArch(...): failed to cast result": {
			method: func(args ...interface{}) (interface{}, error) {
				return GetArch()
			},
			mockResult:  int(1),
			expectError: true,
		},
	}
}

func testCaseKernelRelease(c *C) map[string]testCaseNamespaceMethods {
	return map[string]testCaseNamespaceMethods{
		"GetKernelRelease(...)": {
			method: func(args ...interface{}) (interface{}, error) {
				return GetKernelRelease()
			},
			mockResult: "result",
		},
		"GetKernelRelease(...): failed to run": {
			method: func(args ...interface{}) (interface{}, error) {
				return GetKernelRelease()
			},
			mockError:   fmt.Errorf("failed"),
			expectError: true,
		},
		"GetKernelRelease(...): failed to cast result": {
			method: func(args ...interface{}) (interface{}, error) {
				return GetKernelRelease()
			},
			mockResult:  int(1),
			expectError: true,
		},
	}
}

func testCaseSync(c *C) map[string]testCaseNamespaceMethods {
	return map[string]testCaseNamespaceMethods{
		"Sync(...)": {
			method: func(args ...interface{}) (interface{}, error) {
				return nil, Sync()
			},
		},
	}
}

func testCaseGetOSDistro(c *C) map[string]testCaseNamespaceMethods {
	return map[string]testCaseNamespaceMethods{
		"GetOSDistro(...)": {
			method: func(args ...interface{}) (interface{}, error) {
				return GetOSDistro()
			},
			mockResult: `VERSION="15-SP3"
VERSION_ID="15.3"
PRETTY_NAME="SUSE Linux Enterprise Server 15 SP3"
ID="sles"
ID_LIKE="suse"`,
			expected: "sles",
		},
		"GetOSDistro(...): failed to run": {
			method: func(args ...interface{}) (interface{}, error) {
				return GetOSDistro()
			},
			mockError:   fmt.Errorf("failed"),
			expectError: true,
		},
		"GetOSDistro(...): failed to cast result": {
			method: func(args ...interface{}) (interface{}, error) {
				return GetOSDistro()
			},
			mockResult:  "invalid",
			expectError: true,
		},
	}
}

func testCaseGetSystemBlockDevices(c *C) map[string]testCaseNamespaceMethods {
	return map[string]testCaseNamespaceMethods{
		"GetSystemBlockDevices(...)": {
			method: func(args ...interface{}) (interface{}, error) {
				return GetSystemBlockDevices()
			},
			mockResult: map[string]types.BlockDeviceInfo{
				"sda": {Name: "sda", Major: 8, Minor: 0},
			},
			mockError: nil,
		},
		"GetSystemBlockDevices(...): failed to run": {
			method: func(args ...interface{}) (interface{}, error) {
				return GetSystemBlockDevices()
			},
			mockError:   fmt.Errorf("failed"),
			expectError: true,
		},
		"GetSystemBlockDevices(...): failed to cast result": {
			method: func(args ...interface{}) (interface{}, error) {
				return GetSystemBlockDevices()
			},
			mockResult:  "invalid",
			expectError: true,
		},
	}
}
