package ns

import (
	"fmt"
	"strings"
	"sync"
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

func TestExecuteRetryOnStaleNsDir(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		nsDirectory string
		results     []fake.ExecutorResult
		expectCalls int
		expectErr   bool
		expectOut   string
	}{
		"stale ns dir error triggers retry and succeeds": {
			nsDirectory: "/host/proc/12345",
			results: []fake.ExecutorResult{
				{Output: "", Err: fmt.Errorf("failed to execute: /usr/bin/nsenter [nsenter --mount=/host/proc/12345/ns/mnt iscsiadm --version], output , stderr nsenter: cannot open /host/proc/12345/ns/mnt: No such file or directory: exit status 1")},
				{Output: "iscsiadm version 2.1.9", Err: nil},
			},
			expectCalls: 2,
			expectErr:   false,
			expectOut:   "iscsiadm version 2.1.9",
		},
		"non-stale error is not retried": {
			nsDirectory: "/host/proc/12345",
			results: []fake.ExecutorResult{
				{Output: "", Err: fmt.Errorf("failed to execute: /usr/bin/nsenter [nsenter --mount=/host/proc/12345/ns/mnt iscsiadm -m session], output , stderr iscsiadm: No active sessions: exit status 21")},
			},
			expectCalls: 1,
			expectErr:   true,
			expectOut:   "",
		},
		"stale ns dir error exhausts retries": {
			nsDirectory: "/host/proc/99999",
			results: func() []fake.ExecutorResult {
				r := make([]fake.ExecutorResult, maxNsDirRefreshRetries)
				for i := range r {
					r[i] = fake.ExecutorResult{Output: "", Err: fmt.Errorf("failed to execute: /usr/bin/nsenter [nsenter --mount=/host/proc/99999/ns/mnt iscsiadm --version], output , stderr nsenter: cannot open /host/proc/99999/ns/mnt: No such file or directory: exit status 1")}
				}
				return r
			}(),
			expectCalls: int(maxNsDirRefreshRetries),
			expectErr:   true,
			expectOut:   "",
		},
		"success on first attempt without retry": {
			nsDirectory: "/host/proc/12345",
			results: []fake.ExecutorResult{
				{Output: "ok", Err: nil},
			},
			expectCalls: 1,
			expectErr:   false,
			expectOut:   "ok",
		},
	}

	for testName, tc := range testCases {
		tc := tc
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			mock := &fake.Executor{Results: tc.results}

			nsexec := &Executor{
				namespaces:  []types.Namespace{types.NamespaceMnt},
				nsDirectory: tc.nsDirectory,
				processName: "test-process",
				processDir:  "/host/proc",
				executor:    mock,
			}

			output, err := nsexec.Execute(nil, "iscsiadm", []string{"--version"}, 10*time.Second)

			assert.Equal(t, tc.expectCalls, mock.GetCallCount(), "unexpected number of execute calls")
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.expectOut, output)
		})
	}

	t.Run("concurrent executions on shared executor without retries", func(t *testing.T) {
		t.Parallel()

		const goroutines = 10
		mock := &fake.Executor{} // returns ("output", nil) for all calls

		nsexec := &Executor{
			namespaces:  []types.Namespace{types.NamespaceMnt},
			nsDirectory: "/host/proc/12345",
			processName: "test-process",
			processDir:  "/host/proc",
			executor:    mock,
		}

		var wg sync.WaitGroup
		errs := make([]error, goroutines)
		outputs := make([]string, goroutines)

		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				out, err := nsexec.Execute(nil, "binary", []string{"arg"}, 5*time.Second)
				outputs[idx] = out
				errs[idx] = err
			}(i)
		}

		wg.Wait()

		for i := 0; i < goroutines; i++ {
			assert.NoError(t, errs[i], "goroutine %d returned error", i)
			assert.Equal(t, "output", outputs[i], "goroutine %d wrong output", i)
		}
		assert.Equal(t, goroutines, mock.GetCallCount())
	})

	t.Run("concurrent executions on shared executor with retries", func(t *testing.T) {
		t.Parallel()

		const goroutines = 5
		// Provide alternating stale-error/success results for all goroutines.
		totalResults := goroutines * 2
		results := make([]fake.ExecutorResult, totalResults)
		for i := 0; i < totalResults; i++ {
			if i%2 == 0 {
				results[i] = fake.ExecutorResult{
					Output: "",
					Err:    fmt.Errorf("failed to execute: /usr/bin/nsenter [nsenter --mount=/host/proc/12345/ns/mnt cmd], output , stderr nsenter: cannot open /host/proc/12345/ns/mnt: No such file or directory: exit status 1"),
				}
			} else {
				results[i] = fake.ExecutorResult{Output: "ok", Err: nil}
			}
		}

		mock := &fake.Executor{Results: results}

		nsexec := &Executor{
			namespaces:  []types.Namespace{types.NamespaceMnt},
			nsDirectory: "/host/proc/12345",
			processName: "test-process",
			processDir:  "/host/proc",
			executor:    mock,
		}

		var wg sync.WaitGroup
		errs := make([]error, goroutines)
		outputs := make([]string, goroutines)

		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				out, err := nsexec.Execute(nil, "cmd", []string{"arg"}, 10*time.Second)
				outputs[idx] = out
				errs[idx] = err
			}(i)
		}

		wg.Wait()

		successCount := 0
		for i := 0; i < goroutines; i++ {
			if errs[i] == nil {
				successCount++
				assert.Equal(t, "ok", outputs[i])
			}
		}
		assert.Greater(t, successCount, 0, "at least one goroutine should succeed")
		assert.Greater(t, mock.GetCallCount(), goroutines, "retries should produce more calls than goroutine count")
	})
}
