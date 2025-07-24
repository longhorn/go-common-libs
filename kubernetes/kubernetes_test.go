package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	. "gopkg.in/check.v1"

	"github.com/longhorn/go-common-libs/test"
)

func TestGetInClusterConfig(t *testing.T) {
	type testCase struct {
		expectError bool
	}
	testCases := map[string]testCase{
		"Not in cluster": {
			expectError: true,
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			_, err := GetInClusterConfig()

			if testCase.expectError {
				assert.NotNil(t, err, Commentf(test.ErrErrorFmt, testName, err))
				return
			}
			assert.Nil(t, err, Commentf(test.ErrErrorFmt, testName, err))
		})
	}
}
