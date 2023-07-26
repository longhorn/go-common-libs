package fake

import (
	"time"
)

type Executor struct{}

func (nsexec *Executor) Execute([]string, string, []string, time.Duration) (string, error) {
	return "output", nil
}

func (nsexec *Executor) ExecuteWithTimeout([]string, string, []string, time.Duration) (string, error) {
	return "output", nil
}

func (nsexec *Executor) ExecuteWithoutTimeout([]string, string, []string, time.Duration) (string, error) {
	return "output", nil
}

func (nsexec *Executor) ExecuteWithStdin(string, []string, string, time.Duration) (string, error) {
	return "output", nil
}

func (nsexec *Executor) ExecuteWithStdinPipe(string, []string, string, time.Duration) (string, error) {
	return "output", nil
}
