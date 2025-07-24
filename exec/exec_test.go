package exec

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/longhorn/go-common-libs/types"
)

func TestExecute(t *testing.T) {
	type testCase struct {
		command []string
		timeout time.Duration

		expected            string
		expectedErrorPrefix string
	}
	testCases := map[string]testCase{
		"Valid command": {
			command:  []string{"echo", "hello"},
			timeout:  types.ExecuteNoTimeout,
			expected: "hello\n",
		},
		"With error": {
			command:             []string{"ls", "/not-exist"},
			timeout:             types.ExecuteNoTimeout,
			expectedErrorPrefix: "failed to execute",
		},
		"With timeout": {
			command: []string{"sleep", "1"},
			timeout: 2 * time.Second,
		},
		"With timeout and error": {
			command:             []string{"sleep", "1"},
			timeout:             time.Nanosecond,
			expectedErrorPrefix: "timeout executing",
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			executor := NewExecutor()
			output, err := executor.Execute(nil, testCase.command[0], testCase.command[1:], testCase.timeout)
			if testCase.expectedErrorPrefix != "" {
				assert.Error(t, err)
				assert.True(t, strings.HasPrefix(err.Error(), testCase.expectedErrorPrefix))
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, testCase.expected, output)

		})
	}
}

func TestExecuteWithStdin(t *testing.T) {
	type testCase struct {
		commandStdin string
		timeout      time.Duration

		expected    string
		expectError bool
	}
	testCases := map[string]testCase{
		"Echo stdin input": {
			commandStdin: "foo",
			expected:     "foo\n",
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			if testCase.timeout == 0 {
				testCase.timeout = types.ExecuteDefaultTimeout
			}

			executor := NewExecutor()

			binary := "bash"
			args := []string{"-c", "read input; echo ${input}"}
			output, err := executor.ExecuteWithStdin(binary, args, testCase.commandStdin, testCase.timeout)
			if testCase.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			assert.Equal(t, testCase.expected, output)
		})
	}
}

func TestExecuteWithStdinPipe(t *testing.T) {
	type testCase struct {
		command      []string
		commandStdin string
		timeout      time.Duration

		expected    string
		expectError bool
	}
	testCases := map[string]testCase{
		"Counts stdin bytes using wc": {
			command:      []string{"wc", "-c"},
			commandStdin: "count me",
			expected:     "8\n",
		},
		"Command times out": {
			command:      []string{"sleep", "1"},
			commandStdin: "ignore me",
			timeout:      time.Nanosecond,
			expectError:  true,
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			if testCase.timeout == 0 {
				testCase.timeout = types.ExecuteDefaultTimeout
			}

			executor := NewExecutor()

			output, err := executor.ExecuteWithStdinPipe(testCase.command[0], testCase.command[1:], testCase.commandStdin, testCase.timeout)
			if testCase.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			assert.Equal(t, testCase.expected, output)

		})
	}
}
