package fake

import (
	"sync"
	"time"
)

// ExecutorResult defines the output and error for a single execution call.
type ExecutorResult struct {
	Output string
	Err    error
}

// Executor is a configurable mock for exec.ExecuteInterface.
// If Results is nil, it returns ("output", nil) for every call (backward compatible).
// If Results is set, it returns them in sequence; calls beyond the slice length
// return ("output", nil).
// All methods are safe for concurrent use.
type Executor struct {
	mu        sync.Mutex
	Results   []ExecutorResult
	CallCount int
}

func (e *Executor) nextResult() (string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	idx := e.CallCount
	e.CallCount++
	if e.Results != nil && idx < len(e.Results) {
		return e.Results[idx].Output, e.Results[idx].Err
	}
	return "output", nil
}

// GetCallCount returns the current call count in a thread-safe manner.
func (e *Executor) GetCallCount() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.CallCount
}

func (e *Executor) Execute([]string, string, []string, time.Duration) (string, error) {
	return e.nextResult()
}

func (e *Executor) ExecuteWithTimeout([]string, string, []string, time.Duration) (string, error) {
	return e.nextResult()
}

func (e *Executor) ExecuteWithoutTimeout([]string, string, []string, time.Duration) (string, error) {
	return e.nextResult()
}

func (e *Executor) ExecuteWithStdin(string, []string, string, time.Duration) (string, error) {
	return e.nextResult()
}

func (e *Executor) ExecuteWithStdinPipe(string, []string, string, time.Duration) (string, error) {
	return e.nextResult()
}

type Joiner struct {
	MockDelay  time.Duration
	MockResult interface{}
	MockError  error
}

func (nsjoin *Joiner) Run(fn func() (interface{}, error)) (interface{}, error) {
	time.Sleep(nsjoin.MockDelay)
	return nsjoin.MockResult, nsjoin.MockError
}

func (nsjoin *Joiner) Revert() error {
	return nil
}
