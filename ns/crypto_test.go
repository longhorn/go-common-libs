package ns

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/longhorn/go-common-libs/test/fake"
	"github.com/longhorn/go-common-libs/types"
)

func TestLuksOpen(t *testing.T) {
	type testCase struct{}
	testCases := map[string]testCase{
		"Execute command": {},
	}
	for testName := range testCases {
		t.Run(testName, func(t *testing.T) {
			namespaces := []types.Namespace{types.NamespaceMnt, types.NamespaceIpc}
			nsexec, err := NewNamespaceExecutor(types.ProcessNone, types.HostProcDirectory, namespaces)
			assert.Nil(t, err)

			nsexec.executor = &fake.Executor{}

			output, err := nsexec.LuksOpen("", "", "", types.LuksTimeout)
			assert.Nil(t, err)
			assert.Equal(t, "output", output)
		})
	}
}

func TestLuksClose(t *testing.T) {
	type testCase struct{}
	testCases := map[string]testCase{
		"Execute command": {},
	}
	for testName := range testCases {
		t.Run(testName, func(t *testing.T) {
			namespaces := []types.Namespace{types.NamespaceMnt, types.NamespaceIpc}
			nsexec, err := NewNamespaceExecutor(types.ProcessNone, types.HostProcDirectory, namespaces)
			assert.Nil(t, err)

			nsexec.executor = &fake.Executor{}

			output, err := nsexec.LuksClose("", types.LuksTimeout)
			assert.Nil(t, err)
			assert.Equal(t, "output", output)
		})
	}
}

func TestLuksFormat(t *testing.T) {
	type testCase struct{}
	testCases := map[string]testCase{
		"Execute command": {},
	}
	for testName := range testCases {
		t.Run(testName, func(t *testing.T) {
			namespaces := []types.Namespace{types.NamespaceMnt, types.NamespaceIpc}
			nsexec, err := NewNamespaceExecutor(types.ProcessNone, types.HostProcDirectory, namespaces)
			assert.Nil(t, err)

			nsexec.executor = &fake.Executor{}

			output, err := nsexec.LuksFormat("", "", "", "", "", "", types.LuksTimeout)
			assert.Nil(t, err)
			assert.Equal(t, "output", output)
		})
	}
}

func TestLuksResize(t *testing.T) {
	type testCase struct{}
	testCases := map[string]testCase{
		"Execute command": {},
	}
	for testName := range testCases {
		t.Run(testName, func(t *testing.T) {
			namespaces := []types.Namespace{types.NamespaceMnt, types.NamespaceIpc}
			nsexec, err := NewNamespaceExecutor(types.ProcessNone, types.HostProcDirectory, namespaces)
			assert.Nil(t, err)

			nsexec.executor = &fake.Executor{}

			output, err := nsexec.LuksResize("", "", types.LuksTimeout)
			assert.Nil(t, err)
			assert.Equal(t, "output", output)
		})
	}
}

func TestLuksStatus(t *testing.T) {
	type testCase struct{}
	testCases := map[string]testCase{
		"Execute command": {},
	}
	for testName := range testCases {
		t.Run(testName, func(t *testing.T) {
			namespaces := []types.Namespace{types.NamespaceMnt, types.NamespaceIpc}
			nsexec, err := NewNamespaceExecutor(types.ProcessNone, types.HostProcDirectory, namespaces)
			assert.Nil(t, err)

			nsexec.executor = &fake.Executor{}

			output, err := nsexec.LuksStatus("", types.LuksTimeout)
			assert.Nil(t, err)
			assert.Equal(t, "output", output)
		})
	}
}
