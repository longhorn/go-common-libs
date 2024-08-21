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

func (s *TestSuite) TestIsEngineProcess(c *C) {
	type testCase struct {
		input    string
		expected bool
	}
	testCases := map[string]testCase{
		"IsEngineProcess(...):": {
			input:    "pvc-5a8ee916-5989-46c6-bafc-ddbf7c802499-e-0",
			expected: true,
		},
		"IsEngineProcess(...): engine": {
			input:    "nginx-e-0",
			expected: true,
		},
		"IsEngineProcess(...): engine-2": {
			input:    "nginx-r-e-0",
			expected: true,
		},
		"IsEngineProcess(...): engine-3": {
			input:    "pvc-669e5426-8c62-42df-979d-1be22a30cd0a-e-cc7d5051",
			expected: true,
		},
		"IsEngineProcess(...): engine-4": {
			input:    "pvc-3308aae1-b3c4-4ea3-a6b8-d1fc16cea03b-e-8e24327e",
			expected: true,
		},
		"IsEngineProcess(...): replica": {
			input:    "nginx-r-0",
			expected: false,
		},
		"IsEngineProcess(...): replica-2": {
			input:    "nginx-e-r-0",
			expected: false,
		},
		"IsEngineProcess(...): invalid": {
			input:    "invalid-string",
			expected: false,
		},
		"IsEngineProcess(...): invalid-2": {
			input:    "-e-0",
			expected: false,
		},
		"IsEngineProcess(...): invalid-3": {
			input:    "abc-eee-0",
			expected: false,
		},
		"IsEngineProcess(...): invalid-4": {
			input:    "nginx-er-0",
			expected: false,
		},
		"IsEngineProcess(...): invalid-5": {
			input:    "nginx-e--0",
			expected: false,
		},
		"IsEngineProcess(...): invalid-6": {
			input:    "nginx-e-0-abcd",
			expected: false,
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing utils.%v", testName)

		result := IsEngineProcess(testCase.input)
		c.Assert(result, Equals, testCase.expected, Commentf(test.ErrResultFmt, testName))
	}
}
