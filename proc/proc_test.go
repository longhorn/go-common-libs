package proc

import (
	"fmt"
	"os"
	"testing"

	"github.com/mitchellh/go-ps"
	"github.com/stretchr/testify/assert"
	. "gopkg.in/check.v1"

	"github.com/longhorn/go-common-libs/test"
)

func Test(t *testing.T) { TestingT(t) }

type TestSuite struct{}

var _ = Suite(&TestSuite{})

func TestGetProcessPID(t *testing.T) {
	type testCase struct {
		process  string
		procPath string
	}
	testCases := map[string]testCase{
		"Go": {
			process:  "go",
			procPath: "/proc",
		},
		"Fallback when process not found": {
			process:  "not-exist",
			procPath: "/proc",
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			result, err := GetProcessPIDs(testCase.process, testCase.procPath)
			assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName, err))
			assert.NotEmpty(t, result, Commentf(test.ErrResultFmt, testName))
		})
	}
}

func TestGetHostNamespacePid(t *testing.T) {
	type testCase struct {
		procPath string

		expected uint64
	}
	testCases := map[string]testCase{
		"Current process": {
			procPath: "/proc",
			expected: 1,
		},
		"Fallback when process path not found": {
			procPath: "/not-exist",
			expected: 1,
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			result := GetHostNamespacePID(testCase.procPath)
			assert.Equal(t, testCase.expected, result, Commentf(test.ErrResultFmt, testName))
		})
	}
}

func TestGetProcessAncestorNamespaceDirectory(t *testing.T) {
	type testCase struct {
		process  string
		procPath string

		expected string
	}
	testCases := map[string]testCase{
		"Current process": {
			process:  "go",
			procPath: "/proc",
			// expected: "/proc/1/ns",
		},
	}

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			result, err := GetProcessAncestorNamespaceDirectory(testCase.process, testCase.procPath)
			assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName, err))

			// Other processes with the same name may be running concurrently
			// (e.g. "go vet" or "golangci-lint" in a parallel CI stage), so
			// compare against the actual ancestor of the test process.
			ancestorPid := findAncestorPIDByName(t, testCase.process)
			if testCase.expected == "" {
				testCase.expected = fmt.Sprintf("/proc/%d/ns", ancestorPid)
			}

			assert.Equal(t, testCase.expected, result, Commentf(test.ErrResultFmt, testName))

			procs, err := FindProcessByCmdline(testCase.process)
			assert.NoError(t, err)
			pids := make([]int, 0, len(procs))
			for _, p := range procs {
				pids = append(pids, p.Pid)
			}
			assert.Contains(t, pids, ancestorPid)
		})

	}
}

// findAncestorPIDByName returns the PID of the closest ancestor of the current
// process whose executable has the given name.
func findAncestorPIDByName(t *testing.T, name string) int {
	t.Helper()

	for pid := os.Getppid(); pid > 0; {
		p, err := ps.FindProcess(pid)
		if !assert.NoError(t, err) || p == nil {
			break
		}
		if p.Executable() == name {
			return pid
		}
		pid = p.PPid()
	}
	t.Fatalf("no ancestor process named %q found", name)
	return 0
}

func TestGetProcessNamespaceDirectory(t *testing.T) {
	type testCase struct {
		process  string
		procPath string

		expected string
	}
	testCases := map[string]testCase{
		"Current process": {
			process:  "go",
			procPath: "/proc",
		},
		"Host namespace": {
			procPath: "/proc",
			expected: "/proc/1/ns",
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			if testCase.expected == "" {
				pids, err := GetProcessPIDs(testCase.process, testCase.procPath)
				assert.NoError(t, err)

				testCase.expected = fmt.Sprintf("/proc/%d/ns", pids[0])
			}

			result, err := GetProcessNamespaceDirectory(testCase.process, testCase.procPath)
			assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName, err))
			assert.Equal(t, testCase.expected, result, Commentf(test.ErrResultFmt, testName))
		})
	}
}
