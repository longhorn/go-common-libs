package longhorn

import (
	"testing"

	. "gopkg.in/check.v1"

	"github.com/longhorn/go-common-libs/test"
)

func Test(t *testing.T) { TestingT(t) }

type TestSuite struct{}

var _ = Suite(&TestSuite{})

func (s *TestSuite) TestGetVolumeNameFromReplicaDataDirectoryName(c *C) {
	type testCase struct {
		replicaName string

		expected    string
		expectError bool
	}
	testCases := map[string]testCase{
		"GetVolumeNameFromReplicaDataDirectoryName(...): normal case": {
			replicaName: "pvc-0e045ff8-4ea6-4573-889b-afc9aa147f95-971c46f6",
			expected:    "pvc-0e045ff8-4ea6-4573-889b-afc9aa147f95",
		},
		"GetVolumeNameFromReplicaDataDirectoryName(...): empty replica name": {
			replicaName: "",
			expectError: true,
		},
		"GetVolumeNameFromReplicaDataDirectoryName(...): invalid replica name": {
			replicaName: "pvc-0e045ff8-4ea6-4573-889b-afc9aa147f95-00",
			expectError: true,
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing longhorn.%v", testName)

		result, err := GetVolumeNameFromReplicaDataDirectoryName(testCase.replicaName)
		if testCase.expectError {
			c.Assert(err, NotNil, Commentf(test.ErrErrorFmt, testName, err))
			continue
		}
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName, err))

		c.Assert(result, Equals, testCase.expected, Commentf(test.ErrResultFmt, testName))
	}
}
