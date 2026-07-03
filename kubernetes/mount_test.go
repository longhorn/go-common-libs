package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	. "gopkg.in/check.v1"

	"github.com/longhorn/go-common-libs/test"

	"k8s.io/mount-utils"
)

func TestIsMountPointReadOnly(t *testing.T) {
	type testCase struct {
		input    mount.MountPoint
		expected bool
	}
	testCases := map[string]testCase{
		"ReadOnly": {
			input: mount.MountPoint{
				Opts: []string{"ro"},
			},
			expected: true,
		},
		"ReadWrite": {
			input: mount.MountPoint{
				Opts: []string{"rw"},
			},
			expected: false,
		},
		"EmergencyReadOnly": {
			// ext4 emergency read-only (kernel >= v6.12): the mount still
			// shows "rw" but writes fail with EROFS.
			input: mount.MountPoint{
				Opts: []string{"rw", "seclabel", "relatime", "emergency_ro"},
			},
			expected: true,
		},
		"ReadWriteMultipleOpts": {
			input: mount.MountPoint{
				Opts: []string{"rw", "seclabel", "relatime"},
			},
			expected: false,
		},
		"Empty": {
			input: mount.MountPoint{
				Opts: []string{},
			},
			expected: false,
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			result := IsMountPointReadOnly(testCase.input)
			assert.Equal(t, testCase.expected, result, Commentf(test.ErrResultFmt, testName))
		})
	}
}
