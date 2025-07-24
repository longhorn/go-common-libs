package nfs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	. "gopkg.in/check.v1"

	"github.com/longhorn/go-common-libs/test"
	"github.com/longhorn/go-common-libs/test/fake"
	"github.com/longhorn/go-common-libs/types"
)

func TestGetSystemDefaultNFSVersion(t *testing.T) {
	genNFSMountConf := func(dir string, nfsVer string) {
		data := fmt.Sprintf("[ NFSMount_Global_Options ]\nDefaultvers=%s\n", nfsVer)
		err := os.WriteFile(filepath.Join(dir, types.NFSMountFileName), []byte(data), 0644)
		assert.Nil(t, err)
	}

	type testCase struct {
		setup           func(dir string)
		expectedMajor   int
		expectedMinor   int
		expectedMissing bool
		expectedError   bool
	}
	testCases := map[string]testCase{
		"System NFS mount config absent": {
			setup: func(dir string) {
				_, err := os.Stat(filepath.Join(dir, types.NFSMountFileName))
				assert.True(t, os.IsNotExist(err))
			},
			expectedMajor:   0,
			expectedMinor:   0,
			expectedMissing: true,
			expectedError:   true,
		},
		"System NFS mount config exist without default ver": {
			setup: func(dir string) {
				data := "[ NFSMount_Global_Options ]\notherkey=val\n"
				err := os.WriteFile(filepath.Join(dir, types.NFSMountFileName), []byte(data), 0644)
				assert.Nil(t, err)
			},
			expectedMajor:   0,
			expectedMinor:   0,
			expectedMissing: true,
			expectedError:   true,
		},
		"System NFS mount config set default ver to 3": {
			setup:           func(dir string) { genNFSMountConf(dir, "3") },
			expectedMajor:   3,
			expectedMinor:   0,
			expectedMissing: false,
			expectedError:   false,
		},
		"System NFS mount config set default ver to 4": {
			setup:           func(dir string) { genNFSMountConf(dir, "4") },
			expectedMajor:   4,
			expectedMinor:   0,
			expectedMissing: false,
			expectedError:   false,
		},
		"System NFS mount config set default ver to 4.0": {
			setup:           func(dir string) { genNFSMountConf(dir, "4.0") },
			expectedMajor:   4,
			expectedMinor:   0,
			expectedMissing: false,
			expectedError:   false,
		},
		"System NFS mount config set default ver to 4.2": {
			setup:           func(dir string) { genNFSMountConf(dir, "4.2") },
			expectedMajor:   4,
			expectedMinor:   2,
			expectedMissing: false,
			expectedError:   false,
		},
		"System NFS mount config default ver to invalid value": {
			setup:           func(dir string) { genNFSMountConf(dir, "???") },
			expectedMajor:   0,
			expectedMinor:   0,
			expectedMissing: false,
			expectedError:   true,
		},
		"System NFS mount config default ver to empty value": {
			setup:           func(dir string) { genNFSMountConf(dir, "") },
			expectedMajor:   0,
			expectedMinor:   0,
			expectedMissing: false,
			expectedError:   true,
		},
	}

	configDir := fake.CreateTempDirectory("", t)
	defer func() {
		_ = os.RemoveAll(configDir)
	}()

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			func() {
				defer func() {
					errRemove := os.RemoveAll(filepath.Join(configDir, types.NFSMountFileName))
					assert.Nil(t, errRemove, Commentf(test.ErrErrorFmt, testName))
				}()
				testCase.setup(configDir)

				major, minor, err := GetSystemDefaultNFSVersion(configDir)
				assert.Equal(t, testCase.expectedMajor, major, Commentf(test.ErrResultFmt, testName))
				assert.Equal(t, testCase.expectedMinor, minor, Commentf(test.ErrResultFmt, testName))
				if testCase.expectedMissing {
					assert.True(t, errors.Is(err, types.ErrNotConfigured), Commentf(test.ErrResultFmt, testName))
				} else if testCase.expectedError {
					assert.NotNil(t, err, Commentf(test.ErrErrorFmt, testName))
				} else {
					assert.Nil(t, err, Commentf(test.ErrErrorFmt, testName))
				}
			}()
		})
	}
}
