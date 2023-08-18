package namespace

import (
	"fmt"
	"time"

	"github.com/longhorn/go-common-libs/fake"
	"github.com/longhorn/go-common-libs/types"

	. "gopkg.in/check.v1"
)

func (s *TestSuite) TestGetBaseProcessName(c *C) {
	defer func() {
		NewJoiner = newJoiner
		types.CachedOSDistro = ""
	}()

	type testCase struct {
		osDistroFileContent string

		mockError error

		expectedProcess string
	}
	testCases := map[string]testCase{
		"GetBaseProcessName(...)": {
			osDistroFileContent: `ID="sles"`,
			expectedProcess:     types.ProcessNone,
		},
		"GetBaseProcessName(...): Talos Linux": {
			osDistroFileContent: `ID="talos"`,
			expectedProcess:     types.ProcessKubelet,
		},
		"GetBaseProcessName(...): fallback": {
			mockError:       fmt.Errorf("failed"),
			expectedProcess: types.ProcessNone,
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing namespace.%v", testName)

		NewJoiner = func(string, time.Duration) (JoinerInterface, error) {
			return &fake.Joiner{
				MockResult: testCase.osDistroFileContent,
				MockError:  testCase.mockError,
			}, nil
		}

		process := GetDefaultProcessName()
		c.Assert(process, Equals, testCase.expectedProcess, Commentf(TestErrResultFmt, testName))

		types.CachedOSDistro = ""
	}
}
