package exec

import (
	"strings"
	"testing"
	"time"

	. "gopkg.in/check.v1"

	"github.com/longhorn/go-common-libs/test"
	"github.com/longhorn/go-common-libs/types"
)

func Test(t *testing.T) { TestingT(t) }

type TestSuite struct{}

var _ = Suite(&TestSuite{})

func (s *TestSuite) TestExecute(c *C) {
	type testCase struct {
		command []string
		timeout time.Duration

		expected            string
		expectedErrorPrefix string
	}
	testCases := map[string]testCase{
		"Execute(...)": {
			command:  []string{"echo", "hello"},
			timeout:  types.ExecuteNoTimeout,
			expected: "hello\n",
		},
		"Execute(...) with error": {
			command:             []string{"ls", "/not-exist"},
			timeout:             types.ExecuteNoTimeout,
			expectedErrorPrefix: "failed to execute",
		},
		"Execute(...): with timeout": {
			command: []string{"sleep", "1"},
			timeout: 2 * time.Second,
		},
		"Execute(...): with timeout and error": {
			command:             []string{"sleep", "1"},
			timeout:             time.Nanosecond,
			expectedErrorPrefix: "timeout executing",
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing utils.%v", testName)

		executor := NewExecutor()
		output, err := executor.Execute(nil, testCase.command[0], testCase.command[1:], testCase.timeout)
		if testCase.expectedErrorPrefix != "" {
			c.Assert(err, NotNil)
			c.Assert(strings.HasPrefix(err.Error(), testCase.expectedErrorPrefix), Equals, true, Commentf(test.ErrErrorFmt, testName))
			continue
		}
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName, err))
		c.Assert(output, Equals, testCase.expected, Commentf(test.ErrResultFmt, testName))
	}
}

func (s *TestSuite) TestExecuteWithStdin(c *C) {
	type testCase struct {
		commandStdin string
		timeout      time.Duration

		expected    string
		expectError bool
	}
	testCases := map[string]testCase{
		"ExecuteWithStdin(...)": {
			commandStdin: "foo",
			expected:     "foo\n",
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing utils.%v", testName)

		if testCase.timeout == 0 {
			testCase.timeout = types.ExecuteDefaultTimeout
		}

		executor := NewExecutor()

		binary := "bash"
		args := []string{"-c", "read input; echo ${input}"}
		output, err := executor.ExecuteWithStdin(binary, args, testCase.commandStdin, testCase.timeout)
		if testCase.expectError {
			c.Assert(err, NotNil)
			continue
		}
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName, err))

		c.Assert(output, Equals, testCase.expected, Commentf(test.ErrResultFmt, testName))

	}
}

func (s *TestSuite) TestExecuteWithStdinPipe(c *C) {
	type testCase struct {
		command      []string
		commandStdin string
		timeout      time.Duration

		expected    string
		expectError bool
	}
	testCases := map[string]testCase{
		"ExecuteWithStdinPipe(...)": {
			command:      []string{"wc", "-c"},
			commandStdin: "count me",
			expected:     "8\n",
		},
		"ExecuteWithStdinPipe(...): timeout": {
			command:      []string{"sleep", "1"},
			commandStdin: "ignore me",
			timeout:      time.Nanosecond,
			expectError:  true,
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing utils.%v", testName)

		if testCase.timeout == 0 {
			testCase.timeout = types.ExecuteDefaultTimeout
		}

		executor := NewExecutor()

		output, err := executor.ExecuteWithStdinPipe(testCase.command[0], testCase.command[1:], testCase.commandStdin, testCase.timeout)
		if testCase.expectError {
			c.Assert(err, NotNil)
			continue
		}
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName, err))

		c.Assert(output, Equals, testCase.expected, Commentf(test.ErrResultFmt, testName))

	}
}
