package ns

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	. "gopkg.in/check.v1"

	"github.com/longhorn/go-common-libs/test"
	"github.com/longhorn/go-common-libs/test/fake"
	"github.com/longhorn/go-common-libs/types"
)

func TestGetBaseProcessName(t *testing.T) {
	defer func() {
		NewJoiner = newJoiner
	}()

	type testCase struct {
		osDistroFileContent string

		mockError error

		expectedProcess string
	}
	testCases := map[string]testCase{
		"SLES": {
			osDistroFileContent: `ID="sles"`,
			expectedProcess:     types.ProcessNone,
		},
		"Talos Linux": {
			osDistroFileContent: `ID="talos"`,
			expectedProcess:     types.ProcessKubelet,
		},
		"Fallback": {
			mockError:       fmt.Errorf("failed"),
			expectedProcess: types.ProcessNone,
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			NewJoiner = func(string, time.Duration) (JoinerInterface, error) {
				return &fake.Joiner{
					MockResult: testCase.osDistroFileContent,
					MockError:  testCase.mockError,
				}, nil
			}

			process := GetDefaultProcessName()
			assert.Equal(t, testCase.expectedProcess, process, Commentf(test.ErrResultFmt, testName))
		})
	}
}
