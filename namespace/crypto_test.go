package namespace

import (
	"github.com/longhorn/go-common-libs/fake"
	"github.com/longhorn/go-common-libs/types"

	. "gopkg.in/check.v1"
)

func (s *TestSuite) TestLuksOpen(c *C) {
	type testCase struct{}
	testCases := map[string]testCase{
		"LuksOpen(...)": {},
	}
	for testName := range testCases {
		c.Logf("testing namespace.%v", testName)

		namespaces := []types.Namespace{types.NamespaceMnt, types.NamespaceIpc}
		nsexec, err := NewNamespaceExecutor(types.ProcessNone, types.HostProcDirectory, namespaces)
		c.Assert(err, IsNil)

		nsexec.executor = &fake.Executor{}

		output, err := nsexec.LuksOpen("", "", "", types.LuksTimeout)
		c.Assert(err, IsNil)
		c.Assert(output, Equals, "output")
	}
}

func (s *TestSuite) TestLuksClose(c *C) {
	type testCase struct{}
	testCases := map[string]testCase{
		"LuksClose(...)": {},
	}
	for testName := range testCases {
		c.Logf("testing namespace.%v", testName)

		namespaces := []types.Namespace{types.NamespaceMnt, types.NamespaceIpc}
		nsexec, err := NewNamespaceExecutor(types.ProcessNone, types.HostProcDirectory, namespaces)
		c.Assert(err, IsNil)

		nsexec.executor = &fake.Executor{}

		output, err := nsexec.LuksClose("", types.LuksTimeout)
		c.Assert(err, IsNil)
		c.Assert(output, Equals, "output")
	}
}

func (s *TestSuite) TestLuksFormat(c *C) {
	type testCase struct{}
	testCases := map[string]testCase{
		"LuksFormat(...)": {},
	}
	for testName := range testCases {
		c.Logf("testing namespace.%v", testName)

		namespaces := []types.Namespace{types.NamespaceMnt, types.NamespaceIpc}
		nsexec, err := NewNamespaceExecutor(types.ProcessNone, types.HostProcDirectory, namespaces)
		c.Assert(err, IsNil)

		nsexec.executor = &fake.Executor{}

		output, err := nsexec.LuksFormat("", "", "", "", "", "", types.LuksTimeout)
		c.Assert(err, IsNil)
		c.Assert(output, Equals, "output")
	}
}

func (s *TestSuite) TestLuksResize(c *C) {
	type testCase struct{}
	testCases := map[string]testCase{
		"LuksResize(...)": {},
	}
	for testName := range testCases {
		c.Logf("testing namespace.%v", testName)

		namespaces := []types.Namespace{types.NamespaceMnt, types.NamespaceIpc}
		nsexec, err := NewNamespaceExecutor(types.ProcessNone, types.HostProcDirectory, namespaces)
		c.Assert(err, IsNil)

		nsexec.executor = &fake.Executor{}

		output, err := nsexec.LuksResize("", "", types.LuksTimeout)
		c.Assert(err, IsNil)
		c.Assert(output, Equals, "output")
	}
}

func (s *TestSuite) TestLuksStatus(c *C) {
	type testCase struct{}
	testCases := map[string]testCase{
		"LuksStatus(...)": {},
	}
	for testName := range testCases {
		c.Logf("testing namespace.%v", testName)

		namespaces := []types.Namespace{types.NamespaceMnt, types.NamespaceIpc}
		nsexec, err := NewNamespaceExecutor(types.ProcessNone, types.HostProcDirectory, namespaces)
		c.Assert(err, IsNil)

		nsexec.executor = &fake.Executor{}

		output, err := nsexec.LuksStatus("", types.LuksTimeout)
		c.Assert(err, IsNil)
		c.Assert(output, Equals, "output")
	}
}
