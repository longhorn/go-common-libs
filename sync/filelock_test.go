package sync

import (
	"os"
	"path/filepath"

	"github.com/longhorn/go-common-libs/test"
	"github.com/longhorn/go-common-libs/test/fake"

	. "gopkg.in/check.v1"
)

func (s *TestSuite) TestFileLock(c *C) {
	fakeDir := fake.CreateTempDirectory("", c)
	defer os.RemoveAll(fakeDir)

	type testCase struct {
		fileLockDirectory string

		isLockFileClosed bool

		expectLockError   bool
		expectUnlockError bool
	}
	testCases := map[string]testCase{
		"LockFile/UnlockFile(...)": {},
		"LockFile(...): directory not exist": {
			fileLockDirectory: "not-exist",
			expectLockError:   true,
		},
		"LockFile(...): lock file closed": {
			isLockFileClosed:  true,
			expectUnlockError: true,
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing sync.%v", testName)

		if testCase.fileLockDirectory == "" {
			testCase.fileLockDirectory = fakeDir
		}

		lockFilePath := filepath.Join(testCase.fileLockDirectory, "lock")
		lockFile, err := LockFile(lockFilePath)
		if testCase.expectLockError {
			c.Assert(err, NotNil)
			continue
		}
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName, err))

		if testCase.isLockFileClosed {
			err = lockFile.Close()
			c.Assert(err, IsNil)
		}

		err = UnlockFile(lockFile)
		if testCase.expectUnlockError {
			c.Assert(err, NotNil)
			continue
		}
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName, err))
	}
}
