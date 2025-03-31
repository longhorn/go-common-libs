package nfs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	. "gopkg.in/check.v1"

	"github.com/longhorn/go-common-libs/types"

	"github.com/longhorn/go-common-libs/test"
)

func Test(t *testing.T) { TestingT(t) }

type TestSuite struct{}

var _ = Suite(&TestSuite{})

func (s *TestSuite) TestGetSystemDefaultNFSVersion(c *C) {
	genNFSMountConf := func(dir string, nfsVer string) {
		data := fmt.Sprintf("[ NFSMount_Global_Options ]\nDefaultvers=%s\n", nfsVer)
		err := os.WriteFile(filepath.Join(dir, types.NFSMountFileName), []byte(data), 0644)
		c.Assert(err, IsNil)
	}

	type testCase struct {
		setup           func(dir string)
		expectedMajor   int
		expectedMinor   int
		expectedMissing bool
		expectedError   bool
	}
	testCases := map[string]testCase{
		"GetSystemDefaultNFSVersion(...): system NFS mount config absent": {
			setup: func(dir string) {
				_, err := os.Stat(filepath.Join(dir, types.NFSMountFileName))
				c.Assert(os.IsNotExist(err), Equals, true)
			},
			expectedMajor:   0,
			expectedMinor:   0,
			expectedMissing: true,
			expectedError:   true,
		},
		"GetSystemDefaultNFSVersion(...): system NFS mount config exist without default ver": {
			setup: func(dir string) {
				data := "[ NFSMount_Global_Options ]\notherkey=val\n"
				err := os.WriteFile(filepath.Join(dir, types.NFSMountFileName), []byte(data), 0644)
				c.Assert(err, IsNil)
			},
			expectedMajor:   0,
			expectedMinor:   0,
			expectedMissing: true,
			expectedError:   true,
		},
		"GetSystemDefaultNFSVersion(...): system NFS mount config set default ver to 3": {
			setup:           func(dir string) { genNFSMountConf(dir, "3") },
			expectedMajor:   3,
			expectedMinor:   0,
			expectedMissing: false,
			expectedError:   false,
		},
		"GetSystemDefaultNFSVersion(...): system NFS mount config set default ver to 4": {
			setup:           func(dir string) { genNFSMountConf(dir, "4") },
			expectedMajor:   4,
			expectedMinor:   0,
			expectedMissing: false,
			expectedError:   false,
		},
		"GetSystemDefaultNFSVersion(...): system NFS mount config set default ver to 4.0": {
			setup:           func(dir string) { genNFSMountConf(dir, "4.0") },
			expectedMajor:   4,
			expectedMinor:   0,
			expectedMissing: false,
			expectedError:   false,
		},
		"GetSystemDefaultNFSVersion(...): system NFS mount config set default ver to 4.2": {
			setup:           func(dir string) { genNFSMountConf(dir, "4.2") },
			expectedMajor:   4,
			expectedMinor:   2,
			expectedMissing: false,
			expectedError:   false,
		},
		"GetSystemDefaultNFSVersion(...): system NFS mount config default ver to invalid value": {
			setup:           func(dir string) { genNFSMountConf(dir, "???") },
			expectedMajor:   0,
			expectedMinor:   0,
			expectedMissing: false,
			expectedError:   true,
		},
		"GetSystemDefaultNFSVersion(...): system NFS mount config default ver to empty value": {
			setup:           func(dir string) { genNFSMountConf(dir, "") },
			expectedMajor:   0,
			expectedMinor:   0,
			expectedMissing: false,
			expectedError:   true,
		},
	}

	configDir := c.MkDir()

	for testName, testCase := range testCases {
		func() {
			c.Logf("testing nfs.%v", testName)

			defer func() {
				errRemove := os.RemoveAll(filepath.Join(configDir, types.NFSMountFileName))
				c.Assert(errRemove, IsNil, Commentf(test.ErrErrorFmt, testName))
			}()
			testCase.setup(configDir)

			major, minor, err := GetSystemDefaultNFSVersion(configDir)
			c.Assert(major, Equals, testCase.expectedMajor, Commentf(test.ErrResultFmt, testName))
			c.Assert(minor, Equals, testCase.expectedMinor, Commentf(test.ErrResultFmt, testName))
			if testCase.expectedMissing {
				c.Assert(errors.Is(err, types.ErrNotConfigured), Equals, true, Commentf(test.ErrResultFmt, testName))
			} else if testCase.expectedError {
				c.Assert(err, NotNil, Commentf(test.ErrErrorFmt, testName))
			} else {
				c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))
			}
		}()
	}
}
