package kubernetes

import (
	"testing"

	. "gopkg.in/check.v1"

	"github.com/longhorn/go-common-libs/test"
)

func Test(t *testing.T) { TestingT(t) }

type TestSuite struct{}

var _ = Suite(&TestSuite{})

func (s *TestSuite) TestGetInClusterConfig(c *C) {
	type testCase struct {
		expectError bool
	}
	testCases := map[string]testCase{
		"GetInClusterConfig(...): not in cluster": {
			expectError: true,
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing kubernetes.%v", testName)

		_, err := GetInClusterConfig()

		if testCase.expectError {
			c.Assert(err, NotNil, Commentf(test.ErrErrorFmt, testName, err))
			continue
		}
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName, err))
	}
}
