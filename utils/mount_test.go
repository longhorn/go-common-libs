package utils

import (
	. "gopkg.in/check.v1"

	"github.com/longhorn/go-common-libs/test"
	"k8s.io/mount-utils"
)

func (s *TestSuite) TestIsMountPointReadOnly(c *C) {
	type testCase struct {
		input    mount.MountPoint
		expected bool
	}
	testCases := map[string]testCase{
		"readOnly": {
			input: mount.MountPoint{
				Opts: []string{"ro"},
			},
			expected: true,
		},
		"readWrite": {
			input: mount.MountPoint{
				Opts: []string{"rw"},
			},
			expected: false,
		},
		"empty": {
			input: mount.MountPoint{
				Opts: []string{},
			},
			expected: false,
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing utils.%v", testName)

		result := IsMountPointReadOnly(testCase.input)
		c.Assert(result, Equals, testCase.expected, Commentf(test.ErrResultFmt, testName))
	}
}
