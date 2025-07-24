package ns

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/longhorn/go-common-libs/test/fake"
)

func TestFileLock(t *testing.T) {
	fakeDir := fake.CreateTempDirectory("", t)
	defer func() {
		errRemove := os.RemoveAll(fakeDir)
		assert.NoError(t, errRemove)
	}()

	type testCase struct {
		timeout     time.Duration
		expectError bool
	}
	testCases := map[string]testCase{
		"Valid Lock": {},
		"Timeout": {
			timeout:     time.Second,
			expectError: true,
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			fakeFile := fake.CreateTempFile(fakeDir, "", "content", t)

			NewJoiner = func(string, time.Duration) (JoinerInterface, error) {
				return &fake.Joiner{
					MockDelay:  testCase.timeout + time.Second,
					MockResult: fakeFile,
				}, nil
			}

			lock := NewLock(fakeFile.Name(), testCase.timeout)

			err := lock.Lock()
			if testCase.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			lock.Unlock()
		})

	}
}
