package ns

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/longhorn/go-common-libs/test/fake"
	"github.com/longhorn/go-common-libs/types"
)

func TestExecute(t *testing.T) {
	type testCase struct {
		nsDirectory string
	}
	testCases := map[string]testCase{
		"Current namespace": {},
		"Different namespace": {
			nsDirectory: "/mock",
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			namespaces := []types.Namespace{types.NamespaceMnt, types.NamespaceNet}
			nsexec, err := NewNamespaceExecutor(types.ProcessNone, types.HostProcDirectory, namespaces)
			assert.Nil(t, err)

			nsexec.nsDirectory = testCase.nsDirectory
			nsexec.executor = &fake.Executor{}

			output, err := nsexec.Execute(nil, "binary", []string{"arg1", "arg2"}, types.ExecuteDefaultTimeout)
			assert.Nil(t, err)
			assert.Equal(t, "output", output)

		})
	}
}

func TestExecuteWithTimeout(t *testing.T) {
	type testCase struct {
		timeout time.Duration
	}
	testCases := map[string]testCase{
		"Current namespace": {
			timeout: types.ExecuteNoTimeout,
		},
		"Different namespace": {
			timeout: types.ExecuteNoTimeout,
		},
		"With timeout": {
			timeout: 10 * time.Second,
		},
		"With namespace and timeout": {
			timeout: 10 * time.Second,
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			namespaces := []types.Namespace{types.NamespaceMnt, types.NamespaceNet}
			nsexec, err := NewNamespaceExecutor(types.ProcessNone, types.HostProcDirectory, namespaces)
			assert.Nil(t, err)

			nsexec.executor = &fake.Executor{}

			output, err := nsexec.Execute(nil, "binary", []string{"arg1", "arg2"}, testCase.timeout)
			assert.Nil(t, err)
			assert.Equal(t, "output", output)
		})
	}
}

func TestExecuteWithStdinPipe(t *testing.T) {
	type testCase struct {
		nsDirectory string
	}
	testCases := map[string]testCase{
		"Current namespace": {},
		"Different namespace": {
			nsDirectory: "/mock",
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			namespaces := []types.Namespace{types.NamespaceMnt, types.NamespaceNet}
			nsexec, err := NewNamespaceExecutor(types.ProcessNone, types.HostProcDirectory, namespaces)
			assert.NoError(t, err)

			nsexec.nsDirectory = testCase.nsDirectory
			nsexec.executor = &fake.Executor{}

			output, err := nsexec.ExecuteWithStdinPipe(nil, "binary", []string{"arg1", "arg2"}, "stdin", types.ExecuteDefaultTimeout)
			assert.NoError(t, err)
			assert.Equal(t, "output", output)
		})
	}
}

func TestExecuteWithEnvs(t *testing.T) {
	type testCase struct {
		timeout time.Duration
	}
	testCases := map[string]testCase{
		"Current namespace": {
			timeout: types.ExecuteNoTimeout,
		},
		"Different namespace": {
			timeout: types.ExecuteNoTimeout,
		},
	}
	for testName := range testCases {
		t.Run(testName, func(t *testing.T) {
			namespaces := []types.Namespace{}
			nsexec, err := NewNamespaceExecutor(types.ProcessNone, types.HostProcDirectory, namespaces)
			assert.NoError(t, err)

			output, err := nsexec.Execute([]string{"K1=V1", "K2=V2"}, "env", nil, types.ExecuteDefaultTimeout)
			assert.NoError(t, err)
			assert.True(t, strings.Contains(output, "K1=V1\nK2=V2\n"))
		})
	}
}
