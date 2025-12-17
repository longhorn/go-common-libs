package ns

import (
	"fmt"
	"testing"

	"github.com/longhorn/go-common-libs/types"
)

func testCaseGetArch(t *testing.T) map[string]testCaseNamespaceMethods {
	return map[string]testCaseNamespaceMethods{
		"GetArch/Local": {
			method: func(args ...interface{}) (interface{}, error) {
				return GetArch()
			},
			mockResult: "result",
		},
		"GetArch/Failed to run": {
			method: func(args ...interface{}) (interface{}, error) {
				return GetArch()
			},
			mockError:   fmt.Errorf("failed"),
			expectError: true,
		},
		"GetArch/Failed to cast result": {
			method: func(args ...interface{}) (interface{}, error) {
				return GetArch()
			},
			mockResult:  int(1),
			expectError: true,
		},
	}
}

func testCaseKernelRelease(t *testing.T) map[string]testCaseNamespaceMethods {
	return map[string]testCaseNamespaceMethods{
		"GetKernelRelease/Local": {
			method: func(args ...interface{}) (interface{}, error) {
				return GetKernelRelease()
			},
			mockResult: "result",
		},
		"GetKernelRelease/Failed to run": {
			method: func(args ...interface{}) (interface{}, error) {
				return GetKernelRelease()
			},
			mockError:   fmt.Errorf("failed"),
			expectError: true,
		},
		"GetKernelRelease/Failed to cast result": {
			method: func(args ...interface{}) (interface{}, error) {
				return GetKernelRelease()
			},
			mockResult:  int(1),
			expectError: true,
		},
	}
}

func testCaseSync(t *testing.T) map[string]testCaseNamespaceMethods {
	return map[string]testCaseNamespaceMethods{
		"Sync/Local": {
			method: func(args ...interface{}) (interface{}, error) {
				return nil, Sync()
			},
		},
	}
}

func testCaseGetOSDistro(t *testing.T) map[string]testCaseNamespaceMethods {
	return map[string]testCaseNamespaceMethods{
		"GetOSDistro/SLES": {
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
		"GetOSDistro/Failed to run": {
			method: func(args ...interface{}) (interface{}, error) {
				return GetOSDistro()
			},
			mockError:   fmt.Errorf("failed"),
			expectError: true,
		},
		"GetOSDistro/Failed to cast result": {
			method: func(args ...interface{}) (interface{}, error) {
				return GetOSDistro()
			},
			mockResult:  "invalid",
			expectError: true,
		},
	}
}

func testCaseGetSystemBlockDevices(t *testing.T) map[string]testCaseNamespaceMethods {
	return map[string]testCaseNamespaceMethods{
		"GetSystemBlockDevices/Local": {
			method: func(args ...interface{}) (interface{}, error) {
				return GetSystemBlockDevices()
			},
			mockResult: map[string]types.BlockDeviceInfo{
				"sda": {Name: "sda", Major: 8, Minor: 0},
			},
			mockError: nil,
		},
		"GetSystemBlockDevices/Failed to run": {
			method: func(args ...interface{}) (interface{}, error) {
				return GetSystemBlockDevices()
			},
			mockError:   fmt.Errorf("failed"),
			expectError: true,
		},
		"GetSystemBlockDevices/Failed to cast result": {
			method: func(args ...interface{}) (interface{}, error) {
				return GetSystemBlockDevices()
			},
			mockResult:  "invalid",
			expectError: true,
		},
	}
}

func testCaseResolveBlockDeviceToPhysicalDevice(t *testing.T) map[string]testCaseNamespaceMethods {
	return map[string]testCaseNamespaceMethods{
		"ResolveBlockDeviceToPhysicalDevice/Local": {
			method: func(args ...interface{}) (interface{}, error) {
				return ResolveBlockDeviceToPhysicalDevice("/dev/sda1")
			},
			mockResult: "/dev/sda",
			expected:   "/dev/sda",
		},
		"ResolveBlockDeviceToPhysicalDevice/Failed to run": {
			method: func(args ...interface{}) (interface{}, error) {
				return ResolveBlockDeviceToPhysicalDevice("/dev/sda1")
			},
			mockError:   fmt.Errorf("failed"),
			expectError: true,
		},
		"ResolveBlockDeviceToPhysicalDevice/Failed to cast result": {
			method: func(args ...interface{}) (interface{}, error) {
				return ResolveBlockDeviceToPhysicalDevice("/dev/sda1")
			},
			mockResult:  int(1),
			expectError: true,
		},
	}
}
