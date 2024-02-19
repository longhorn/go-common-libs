package ns

import (
	"time"

	. "gopkg.in/check.v1"

	"github.com/longhorn/go-common-libs/test/fake"
	"github.com/longhorn/go-common-libs/types"
)

func (s *TestSuite) TestExecute(c *C) {
	type testCase struct {
		nsDirectory string
	}
	testCases := map[string]testCase{
		"Execute(...)": {},
		"Execute(...): with namespace": {
			nsDirectory: "/mock",
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing namespace.%v", testName)

		namespaces := []types.Namespace{types.NamespaceMnt, types.NamespaceNet}
		nsexec, err := NewNamespaceExecutor(types.ProcessNone, types.HostProcDirectory, namespaces)
		c.Assert(err, IsNil)

		nsexec.nsDirectory = testCase.nsDirectory
		nsexec.executor = &fake.Executor{}

		output, err := nsexec.Execute(nil, "binary", []string{"arg1", "arg2"}, types.ExecuteDefaultTimeout)
		c.Assert(err, IsNil)
		c.Assert(output, Equals, "output")
	}
}

func (s *TestSuite) TestExecuteWithTimeout(c *C) {
	type testCase struct {
		timeout time.Duration
	}
	testCases := map[string]testCase{
		"Execute(...):": {
			timeout: types.ExecuteNoTimeout,
		},
		"Execute(...): with namespace": {
			timeout: types.ExecuteNoTimeout,
		},
		"Execute(...): with timeout": {
			timeout: 10 * time.Second,
		},
		"Execute(...): with namespace and timeout": {
			timeout: 10 * time.Second,
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing namespace.%v", testName)

		namespaces := []types.Namespace{types.NamespaceMnt, types.NamespaceNet}
		nsexec, err := NewNamespaceExecutor(types.ProcessNone, types.HostProcDirectory, namespaces)
		c.Assert(err, IsNil)

		nsexec.executor = &fake.Executor{}

		output, err := nsexec.Execute(nil, "binary", []string{"arg1", "arg2"}, testCase.timeout)
		c.Assert(err, IsNil)
		c.Assert(output, Equals, "output")
	}
}

func (s *TestSuite) TestExecuteWithStdinPipe(c *C) {
	type testCase struct {
		nsDirectory string
	}
	testCases := map[string]testCase{
		"ExecuteWithStdinPipe(...)": {},
		"ExecuteWithStdinPipe(...): with namespace": {
			nsDirectory: "/mock",
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing namespace.%v", testName)

		namespaces := []types.Namespace{types.NamespaceMnt, types.NamespaceNet}
		nsexec, err := NewNamespaceExecutor(types.ProcessNone, types.HostProcDirectory, namespaces)
		c.Assert(err, IsNil)

		nsexec.nsDirectory = testCase.nsDirectory
		nsexec.executor = &fake.Executor{}

		output, err := nsexec.ExecuteWithStdinPipe(nil, "binary", []string{"arg1", "arg2"}, "stdin", types.ExecuteDefaultTimeout)
		c.Assert(err, IsNil)
		c.Assert(output, Equals, "output")
	}
}
