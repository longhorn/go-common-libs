package sync

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	. "gopkg.in/check.v1"

	"github.com/longhorn/go-common-libs/test"
	"github.com/longhorn/go-common-libs/test/fake"
)

func TestFileLock(t *testing.T) {
	fakeDir := fake.CreateTempDirectory("", t)
	defer func() {
		errRemove := os.RemoveAll(fakeDir)
		assert.NoError(t, errRemove)
	}()

	type testCase struct {
		fileLockDirectory string

		isLockFileClosed bool

		expectLockError   bool
		expectUnlockError bool
	}
	testCases := map[string]testCase{
		"Directory exists": {},
		"Directory not exist": {
			fileLockDirectory: "not-exist",
			expectLockError:   true,
		},
		"Lock file closed": {
			isLockFileClosed:  true,
			expectUnlockError: true,
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			if testCase.fileLockDirectory == "" {
				testCase.fileLockDirectory = fakeDir
			}

			lockFilePath := filepath.Join(testCase.fileLockDirectory, "lock")
			lockFile, err := LockFile(lockFilePath)
			if testCase.expectLockError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName))

			if testCase.isLockFileClosed {
				err = lockFile.Close()
				assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName))
			}

			err = UnlockFile(lockFile)
			if testCase.expectUnlockError {
				assert.Error(t, err, Commentf(test.ErrErrorFmt, testName))
				return
			}
			assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName))
		})
	}
}
