package ns

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/longhorn/go-common-libs/test/fake"
	"github.com/longhorn/go-common-libs/types"
)

func TestIsLikelyStaleNamespaceError(t *testing.T) {
	examples := []struct {
		name  string
		err   error
		want  bool
	}{
		{"nil", nil, false},
		{"unrelated", errors.New("failed to execute: iscsiadm"), false},
		{
			"stale mnt from nsenter",
			errors.New(`failed to execute: /usr/bin/nsenter [nsenter --mount=/host/proc/9134/ns/mnt --net=/host/proc/9134/ns/net iscsiadm --version], output , stderr nsenter: cannot open /host/proc/9134/ns/mnt: No such file or directory: exit status 1`),
			true,
		},
		{
			"non-stale iscsi error",
			errors.New("failed to execute: /usr/bin/nsenter iscsiadm: iSCSI error: exit status 12"),
			false,
		},
	}
	for _, e := range examples {
		t.Run(e.name, func(t *testing.T) {
			assert.Equal(t, e.want, isLikelyStaleNamespaceError(e.err), e.name)
		})
	}
}

// retryingStubExecutor returns a stale error on the first run and "ok" on the second, for retry behavior.
type retryingStubExecutor struct{ calls int }

func (e *retryingStubExecutor) Execute(_ []string, _ string, _ []string, _ time.Duration) (string, error) {
	e.calls++
	if e.calls == 1 {
		return "", errors.New(`failed to execute: /usr/bin/nsenter [nsenter --mount=/host/proc/9134/ns/mnt], output , stderr nsenter: cannot open /host/proc/9134/ns/mnt: No such file or directory: exit status 1`)
	}
	return "ok", nil
}
func (e *retryingStubExecutor) ExecuteWithStdin(_ string, _ []string, _ string, _ time.Duration) (string, error) {
	return "", errors.New("not used")
}
func (e *retryingStubExecutor) ExecuteWithStdinPipe(_ string, _ []string, _ string, _ time.Duration) (string, error) {
	return "", errors.New("not used")
}

// TestExecuteRefreshesNamespaceOnStaleError checks one retry re-runs the inner command after
// a stale path error, when namespace re-resolution succeeds.
func TestExecuteRefreshesNamespaceOnStaleError(t *testing.T) {
	inner, err := NewNamespaceExecutor(types.ProcessNone, types.HostProcDirectory, []types.Namespace{types.NamespaceMnt})
	if err != nil {
		t.Skipf("no namespace executor (host /proc not suitable): %v", err)
	}
	nsexec := &Executor{
		namespaces:     inner.namespaces,
		nsDirectory:    inner.nsDirectory,
		processName:    types.ProcessNone,
		procDirectory:  types.HostProcDirectory,
		executor:       &retryingStubExecutor{},
	}
	out, err := nsexec.Execute(nil, "true", nil, types.ExecuteDefaultTimeout)
	assert.NoError(t, err)
	assert.Equal(t, "ok", out)
}

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
