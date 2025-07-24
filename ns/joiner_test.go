package ns

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	. "gopkg.in/check.v1"

	"github.com/longhorn/go-common-libs/test"
	"github.com/longhorn/go-common-libs/test/fake"
)

func TestRun(t *testing.T) {
	stopCh := make(chan struct{})
	defer close(stopCh)

	type testCase struct {
		procCmd       []string
		procDirectory string
		timeout       time.Duration
		expectError   bool
	}
	testCases := map[string]testCase{
		"Run codes": {
			procDirectory: "/proc",
			expectError:   false,
		},
		"Invalid proc directory": {
			procDirectory: "/invalid",
			expectError:   true,
		},
		"Default proc directory": {
			procDirectory: "",
			expectError:   false,
		},
		"Run command in specific process namespace": {
			procCmd:       []string{"sleep", "infinity"},
			procDirectory: "/proc",
			expectError:   false,
		},
		"Timeout": {
			procDirectory: "/proc",
			timeout:       1 * time.Second,
			expectError:   true,
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			proc := ""
			if len(testCase.procCmd) != 0 {
				proc = testCase.procCmd[0]
				dummpyProcCmd := exec.Command(proc, testCase.procCmd[1:]...)
				err := dummpyProcCmd.Start()
				assert.NoError(t, err)
				_ = dummpyProcCmd.Process.Kill()
			}

			// Create the Enter instance using mock ProcessNamespace
			nsjoin, err := newJoiner(testCase.procDirectory, testCase.timeout)
			assert.NoError(t, err)

			// Define a function to be executed in the namespace
			wait := testCase.timeout + time.Second
			fn := func() (interface{}, error) {
				time.Sleep(wait)
				return os.Stat("/tmp")
			}

			_, err = nsjoin.Run(fn)
			if testCase.expectError {
				assert.Error(t, err)

				if testCase.timeout > 0 {
					assert.True(t, strings.HasPrefix(err.Error(), "timeout"))
				}
				return
			}
			assert.NoError(t, err)
		})
	}
}

type testCaseNamespaceMethods struct {
	method           func(...interface{}) (interface{}, error)
	methodArgs       []interface{}
	methodPostAction func()

	mockResult interface{}
	mockError  error

	expected    interface{}
	expectError bool
}

func runNamespaceMethodTest(t *testing.T, testName string, testCase testCaseNamespaceMethods) {
	NewJoiner = func(string, time.Duration) (JoinerInterface, error) {
		return &fake.Joiner{
			MockResult: testCase.mockResult,
			MockError:  testCase.mockError,
		}, nil
	}
	defer func() {
		NewJoiner = newJoiner
		if testCase.methodPostAction != nil {
			testCase.methodPostAction()
		}
	}()

	result, err := testCase.method(testCase.methodArgs...)
	if testCase.expectError {
		assert.Error(t, err, Commentf(test.ErrErrorFmt, testName, err))
		return
	}

	assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName, err))

	if testCase.expected != nil {
		assert.Equal(t, testCase.expected, result, Commentf(test.ErrResultFmt, testName))
	}
}

func TestNamespaceMethods(t *testing.T) {
	testMethods := []map[string]testCaseNamespaceMethods{
		testCaseGetArch(t),
		testCaseKernelRelease(t),
		testCaseSync(t),
		testCaseGetOSDistro(t),
		testCaseGetSystemBlockDevices(t),
		testCaseCopyDirectory(t),
		testCaseCreateDirectory(t),
		testCaseDeleteDirectory(t),
		testCaseReadDirectory(t),
		testCaseCopyFiles(t),
		testCaseGetEmptyFiles(t),
		testCaseGetFileInfo(t),
		testCaseReadFileContent(t),
		testCaseSyncFile(t),
		testCaseWriteFile(t),
		testCaseDeletePath(t),
		testCaseGetDiskStat(t),
	}
	testCases := make(map[string]testCaseNamespaceMethods)
	for _, testMethod := range testMethods {
		for testName, testCase := range testMethod {
			testCases[testName] = testCase
		}
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			runNamespaceMethodTest(t, testName, testCase)
		})
	}
}
