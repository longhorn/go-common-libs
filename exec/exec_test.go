package exec

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
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

			assert.Equal(t, strings.TrimSpace(testCase.expected), strings.TrimSpace(output))

		})
	}
}

func TestExecuteTimeoutKillsProcesses(t *testing.T) {
	// Unique sleep duration so pgrep only matches processes from this test.
	const uniqueCommand = "sleep 297.53"

	executor := NewExecutor()
	// "& wait" makes the shell spawn sleep as a child process, verifying that
	// the timeout kills the whole process group, not just the direct child.
	_, err := executor.Execute(nil, "sh", []string{"-c", uniqueCommand + " & wait"}, time.Second)
	assert.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "timeout executing"))

	// Give the kill a moment to be delivered.
	time.Sleep(100 * time.Millisecond)
	err = exec.Command("pgrep", "-f", uniqueCommand).Run()
	assert.Error(t, err, "processes spawned by the command should be killed after timeout")
}

const helperProcessEnv = "GO_EXEC_TEST_HELPER_PROCESS"
const helperProcessPidFileEnv = "GO_EXEC_TEST_PID_FILE"

func TestHelperProcess(t *testing.T) {
	if os.Getenv(helperProcessEnv) != "1" {
		return
	}

	pidFile := os.Getenv(helperProcessPidFileEnv)
	if pidFile == "" {
		os.Exit(1)
	}

	// Spawn a child process in a new process group so that this process
	// can change its PGID to the child's PGID.
	child := exec.Command("sleep", "30")
	child.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := child.Start(); err != nil {
		os.Exit(1)
	}
	defer func() {
		_ = child.Process.Kill()
		_ = child.Wait()
	}()

	// Change the process group of the current process away from its original PGID.
	if err := syscall.Setpgid(0, child.Process.Pid); err != nil {
		os.Exit(1)
	}

	// Write PID to file so the test can verify whether this process was terminated.
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())), 0644); err != nil {
		os.Exit(1)
	}

	// Sleep until killed by timeout.
	time.Sleep(30 * time.Second)
	os.Exit(0)
}

func TestExecuteTimeoutFallbackKillProcess(t *testing.T) {
	tmpDir := t.TempDir()
	pidFile := filepath.Join(tmpDir, "helper.pid")

	executor := NewExecutor()
	envs := []string{
		helperProcessEnv + "=1",
		helperProcessPidFileEnv + "=" + pidFile,
	}
	args := []string{"-test.run=^TestHelperProcess$", "--"}

	_, err := executor.Execute(envs, os.Args[0], args, 500*time.Millisecond)
	assert.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "timeout executing"))

	data, err := os.ReadFile(pidFile)
	assert.NoError(t, err, "helper process should write its PID to file")
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	assert.NoError(t, err)
	assert.Greater(t, pid, 0)

	// Give the kill a moment to be delivered.
	time.Sleep(100 * time.Millisecond)

	// Verify that the command PID was terminated by fallback cmd.Process.Kill()
	// even though the process changed its process group.
	err = syscall.Kill(pid, 0)
	assert.Error(t, err, "command PID should be terminated after timeout")
}
