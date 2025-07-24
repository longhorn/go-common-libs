package proc

import (
	"fmt"
	"testing"

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

			if testCase.expected == "" {
				p, err := FindProcessByName(testCase.process)
				assert.NoError(t, err)
				testCase.expected = fmt.Sprintf("/proc/%d/ns", p.Pid)
			}

			assert.Equal(t, testCase.expected, result, Commentf(test.ErrResultFmt, testName))

			ps, err := FindProcessByCmdline(testCase.process)
			assert.NoError(t, err)
			assert.Equal(t, 1, len(ps))
			if len(ps) > 0 {
				assert.Equal(t, fmt.Sprintf("/proc/%d/ns", ps[0].Pid), result, Commentf(test.ErrResultFmt, testName))
			}
		})

	}
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
