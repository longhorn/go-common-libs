package namespace

import (
	"os"
	"time"

	"github.com/longhorn/go-common-libs/fake"

	. "gopkg.in/check.v1"
)

func (s *TestSuite) TestFileLock(c *C) {
	fakeDir := fake.CreateTempDirectory("", c)
	defer os.RemoveAll(fakeDir)

	type testCase struct {
		timeout     time.Duration
		expectError bool
	}
	testCases := map[string]testCase{
		"File Lock/Unlock(...)": {},
		"File Lock(...): timeout": {
			timeout:     time.Second,
			expectError: true,
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing namespace.%v", testName)

		fakeFile := fake.CreateTempFile(fakeDir, "", "content", c)

		NewJoiner = func(string, time.Duration) (JoinerInterface, error) {
			return &fake.Joiner{
				MockDelay:  testCase.timeout + time.Second,
				MockResult: fakeFile,
			}, nil
		}

		lock := NewLock(fakeFile.Name(), testCase.timeout)

		err := lock.Lock()
		if testCase.expectError {
			c.Assert(err, NotNil)
		} else {
			c.Assert(err, IsNil)
		}

		lock.Unlock()
	}
}
