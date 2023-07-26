package namespace

import (
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/longhorn/go-common-libs/fake"

	. "gopkg.in/check.v1"
)

func (s *TestSuite) TestRun(c *C) {
	stopCh := make(chan struct{})
	defer close(stopCh)

	type testCase struct {
		procCmd       []string
		procDirectory string
		timeout       time.Duration
		expectError   bool
	}
	testCases := map[string]testCase{
		"Run(...)": {
			procDirectory: "/proc",
			expectError:   false,
		},
		"Run(...): invalid proc directory": {
			procDirectory: "/invalid",
			expectError:   true,
		},
		"Run(...): default proc directory": {
			procDirectory: "",
			expectError:   false,
		},
		"Run(...): run in specific process namespace": {
			procCmd:       []string{"sleep", "infinity"},
			procDirectory: "/proc",
			expectError:   false,
		},
		"Run(...): timeout": {
			procDirectory: "/proc",
			timeout:       1 * time.Second,
			expectError:   true,
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing namespace.%v", testName)

		proc := ""
		if len(testCase.procCmd) != 0 {
			proc = testCase.procCmd[0]
			dummpyProcCmd := exec.Command(proc, testCase.procCmd[1:]...)
			err := dummpyProcCmd.Start()
			c.Assert(err, IsNil)
			defer dummpyProcCmd.Process.Kill()
		}

		// Create the Enter instance using mock ProcessNamespace
		nsjoin, err := newJoiner(testCase.procDirectory, testCase.timeout)
		c.Assert(err, IsNil)

		// Define a function to be executed in the namespace
		wait := testCase.timeout + time.Second
		fn := func() (interface{}, error) {
			time.Sleep(wait)
			return os.Stat("/tmp")
		}

		_, err = nsjoin.Run(fn)
		if testCase.expectError {
			c.Assert(err, NotNil)

			if testCase.timeout > 0 {
				c.Assert(strings.HasPrefix(err.Error(), "timeout"), Equals, true)
			}
			continue
		}
		c.Assert(err, IsNil)
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

func runNamespaceMethodTest(c *C, testName string, testCase testCaseNamespaceMethods) {
	c.Logf("testing namespace.%v", testName)

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
		c.Assert(err, NotNil)
		return
	}

	c.Assert(err, IsNil, Commentf(TestErrErrorFmt, testName, err))

	if testCase.expected != nil {
		c.Assert(result, Equals, testCase.expected, Commentf(TestErrResultFmt, testName))
	}
}

func (s *TestSuite) TestNamespaceMethods(c *C) {
	testMethods := []map[string]testCaseNamespaceMethods{
		testCaseKernelRelease(c),
		testCaseSync(c),
		testCaseGetOSDistro(c),
		testCaseGetSystemBlockDevices(c),
	}
	testCases := make(map[string]testCaseNamespaceMethods)
	for _, testMethod := range testMethods {
		for testName, testCase := range testMethod {
			testCases[testName] = testCase
		}
	}
	for testName, testCase := range testCases {
		runNamespaceMethodTest(c, testName, testCase)
	}
}
