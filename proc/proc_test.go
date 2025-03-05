package proc

import (
	"fmt"
	"testing"

	. "gopkg.in/check.v1"

	"github.com/longhorn/go-common-libs/test"
)

func Test(t *testing.T) { TestingT(t) }

type TestSuite struct{}

var _ = Suite(&TestSuite{})

func (s *TestSuite) TestGetProcessPID(c *C) {
	type testCase struct {
		process  string
		procPath string
	}
	testCases := map[string]testCase{
		"GetProcessPIDs(...)": {
			process:  "go",
			procPath: "/proc",
		},
		"GetProcessPIDs(...): fallback when process not found": {
			process:  "not-exist",
			procPath: "/proc",
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing proc.%v", testName)

		result, err := GetProcessPIDs(testCase.process, testCase.procPath)
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName, err))
		c.Assert(len(result), Not(Equals), 0, Commentf(test.ErrResultFmt, testName))
	}
}

func (s *TestSuite) TestGetHostNamespacePid(c *C) {
	type testCase struct {
		procPath string

		expected uint64
	}
	testCases := map[string]testCase{
		"GetHostNamespacePID(...)": {
			procPath: "/proc",
			expected: 1,
		},
		"GetHostNamespacePID(...): fallback": {
			procPath: "/not-exist",
			expected: 1,
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing proc.%v", testName)

		result := GetHostNamespacePID(testCase.procPath)
		c.Assert(result, Equals, testCase.expected, Commentf(test.ErrResultFmt, testName))
	}
}

func (s *TestSuite) TestGetProcessAncestorNamespaceDirectory(c *C) {
	type testCase struct {
		process  string
		procPath string

		expected string
	}
	testCases := map[string]testCase{
		"GetProcessAncestorNamespaceDirectory(...)": {
			process:  "go",
			procPath: "/proc",
			// expected: "/proc/1/ns",
		},
	}

	for testName, testCase := range testCases {
		c.Logf("testing proc.%v", testName)

		result, err := GetProcessAncestorNamespaceDirectory(testCase.process, testCase.procPath)
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName, err))

		if testCase.expected == "" {
			p, err := FindProcessByName(testCase.process)
			c.Assert(err, IsNil)
			testCase.expected = fmt.Sprintf("/proc/%d/ns", p.Pid)
		}

		c.Assert(result, Equals, testCase.expected, Commentf(test.ErrResultFmt, testName))

		ps, err := FindProcessByCmdline(testCase.process)
		c.Assert(err, IsNil)
		c.Assert(len(ps), Equals, 1)
		if len(ps) > 0 {
			c.Assert(result, Equals, fmt.Sprintf("/proc/%d/ns", ps[0].Pid), Commentf(test.ErrResultFmt, testName))
		}
	}
}

func (s *TestSuite) TestGetProcessNamespaceDirectory(c *C) {
	type testCase struct {
		process  string
		procPath string

		expected string
	}
	testCases := map[string]testCase{
		"GetProcessNamespaceDirectory(...)": {
			process:  "go",
			procPath: "/proc",
		},
		"GetProcessNamespaceDirectory(...): host namespace": {
			procPath: "/proc",
			expected: "/proc/1/ns",
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing proc.%v", testName)

		if testCase.expected == "" {
			pids, err := GetProcessPIDs(testCase.process, testCase.procPath)
			c.Assert(err, IsNil)

			testCase.expected = fmt.Sprintf("/proc/%d/ns", pids[0])
		}

		result, err := GetProcessNamespaceDirectory(testCase.process, testCase.procPath)
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName, err))
		c.Assert(result, Equals, testCase.expected, Commentf(test.ErrResultFmt, testName))
	}
}
