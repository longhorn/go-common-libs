package sys

import (
	"compress/gzip"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	. "gopkg.in/check.v1"

	"github.com/longhorn/go-common-libs/io"
	"github.com/longhorn/go-common-libs/test"
	"github.com/longhorn/go-common-libs/test/fake"
	"github.com/longhorn/go-common-libs/types"
	"github.com/longhorn/go-common-libs/utils"
)

func TestGetArch(t *testing.T) {
	type testCase struct{}
	testCases := map[string]testCase{
		"Local": {},
	}
	for testName := range testCases {
		t.Run(testName, func(t *testing.T) {
			result, err := GetArch()
			assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName, err))
			assert.NotEqual(t, "", result, Commentf(test.ErrResultFmt, testName))

			// https://longhorn.io/docs/1.9.0/best-practices/#architecture
			supportedArch := []string{
				"x86_64",  // amd64
				"aarch64", // arm64
				"s390x",   // s390x
			}
			assert.True(t, utils.Contains(supportedArch, result), Commentf(test.ErrResultFmt, testName))
		})
	}
}

func TestGetKernelRelease(t *testing.T) {
	type testCase struct{}
	testCases := map[string]testCase{
		"Local": {},
	}
	for testName := range testCases {
		t.Run(testName, func(t *testing.T) {
			result, err := GetKernelRelease()
			assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName, err))
			assert.NotEqual(t, "", result, Commentf(test.ErrResultFmt, testName))
		})
	}
}

// TestGetHostOSDistro tests the success cases of GetHostOSDistro
func TestGetHostOSDistro(t *testing.T) {
	type testCase struct {
		mockFileContent string

		expected string
	}
	testCases := map[string]testCase{
		"SLES": {
			mockFileContent: `NAME="SLES"
VERSION="15-SP3"
VERSION_ID="15.3"
PRETTY_NAME="SUSE Linux Enterprise Server 15 SP3"
ID="sles"
ID_LIKE="suse"`,
			expected: "sles",
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			result, err := GetOSDistro(testCase.mockFileContent)
			assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName, err))
			assert.Equal(t, testCase.expected, result, Commentf(test.ErrResultFmt, testName))
		})
	}
}

// TestGetHostOSDistroFailures tests the failure cases of GetHostOSDistro
// it cannot be run in TestGetHostOSDistro because the cache is not cleared
// between tests
func TestGetHostOSDistroFailures(t *testing.T) {
	type testCase struct {
		mockFileContent string
	}
	testCases := map[string]testCase{
		"Missing ID": {
			mockFileContent: `NAME="SLES"
VERSION="15-SP3"
VERSION_ID="15.3"
PRETTY_NAME="SUSE Linux Enterprise Server 15 SP3"
ID_LIKE="suse"`,
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			_, err := GetOSDistro(testCase.mockFileContent)
			assert.Error(t, err)
		})
	}
}

func TestGetSystemBlockDevices(t *testing.T) {
	fakeDir := fake.CreateTempDirectory("", t)
	defer func() {
		_ = os.RemoveAll(fakeDir)
	}()

	type testCase struct {
		mockDirEntries []os.DirEntry
		mockData       []byte

		expected map[string]types.BlockDeviceInfo
	}
	testCases := map[string]testCase{
		"Success": {
			mockDirEntries: []fs.DirEntry{
				fake.DirEntry("sda", true),
				fake.DirEntry("sdb", true),
				fake.DirEntry("sdc", true),
			},
			expected: map[string]types.BlockDeviceInfo{
				"sda": {Name: "sda", Major: 8, Minor: 0},
				"sdb": {Name: "sdb", Major: 8, Minor: 1},
				"sdc": {Name: "sdc", Major: 8, Minor: 2},
			},
		},
		"Invalid file content": {
			mockDirEntries: []fs.DirEntry{
				fake.DirEntry("sda", true),
				fake.DirEntry("sdb", true),
				fake.DirEntry("sdc", true),
			},
			mockData: []byte("invalid file content"),
			expected: map[string]types.BlockDeviceInfo{},
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			fakeFS := fake.FileSystem{
				DirEntries: testCase.mockDirEntries,
				Data:       testCase.mockData,
			}

			for _, entry := range testCase.mockDirEntries {
				// Create device directory
				deviceDir := filepath.Join(fakeDir, entry.Name())
				err := os.MkdirAll(deviceDir, 0755)
				assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName, err))

				// Create device file
				devicePath := filepath.Join(deviceDir, "dev")
				deviceFile, err := os.Create(devicePath)
				errClose := deviceFile.Close()
				assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName, err))
				assert.NoError(t, errClose, Commentf(test.ErrErrorFmt, testName, errClose))
			}

			result, err := getSystemBlockDeviceInfo(fakeDir, fakeFS.ReadDir, fakeFS.ReadFile)
			assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName, err))
			assert.True(t, reflect.DeepEqual(result, testCase.expected), Commentf(test.ErrResultFmt, testName))
		})
	}
}

func TestGetBootKernelConfigMap(t *testing.T) {
	type testCase struct {
		mockFileContent   string
		expectedConfigMap map[string]string
		expectedError     bool
	}
	testCases := map[string]testCase{
		"Read kernel config": {
			mockFileContent: `CONFIG_DM_CRYPT=y
# comment should be ignored
CONFIG_NFS_V4=m
CONFIG_NFS_V4_1=m
CONFIG_NFS_V4_2=y`,
			expectedConfigMap: map[string]string{
				"CONFIG_DM_CRYPT": "y",
				"CONFIG_NFS_V4":   "m",
				"CONFIG_NFS_V4_1": "m",
				"CONFIG_NFS_V4_2": "y",
			},
			expectedError: false,
		},
		"Empty kernel config": {
			mockFileContent:   "",
			expectedConfigMap: map[string]string{},
			expectedError:     false,
		},
		"Invalid content": {
			mockFileContent:   "key=val\nCONFIG_invalid_content\n",
			expectedConfigMap: nil,
			expectedError:     true,
		},
	}

	bootDir := fake.CreateTempDirectory("", t)
	kernelVersion := "1.2.3.4"

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			err := os.WriteFile(filepath.Join(bootDir, "config-"+kernelVersion), []byte(testCase.mockFileContent), 0644)
			assert.NoError(t, err)

			exact, err := GetBootKernelConfigMap(bootDir, kernelVersion)
			assert.Equal(t, testCase.expectedConfigMap, exact, Commentf(test.ErrResultFmt, testName))
			if testCase.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName, err))
			}
		})
	}
}
func TestGetProcKernelConfigMap(t *testing.T) {
	genKernelConfig := func(dir string, lines ...string) {
		path := filepath.Join(dir, types.SysKernelConfigGz)
		file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		assert.NoError(t, err)
		defer func() {
			errClose := file.Close()
			assert.NoError(t, errClose)
		}()
		gzWriter := gzip.NewWriter(file)
		defer func() {
			errClose := gzWriter.Close()
			assert.NoError(t, errClose)
		}()
		for _, line := range lines {
			_, err := fmt.Fprintln(gzWriter, line)
			assert.NoError(t, err)
		}
	}

	type testCase struct {
		setup                    func(dir string)
		expectedConfigMap        map[string]string
		expectedConfigMapChecker func(map[string]string) bool
		expectedError            bool
	}

	testCases := map[string]testCase{
		"Read kernel config": {
			setup: func(dir string) {
				genKernelConfig(dir,
					"CONFIG_DM_CRYPT=y",
					"# comment should be ignored",
					"CONFIG_NFS_V4=m",
					"CONFIG_NFS_V4_1=m",
					"CONFIG_NFS_V4_2=y")
			},
			expectedConfigMap: map[string]string{
				"CONFIG_DM_CRYPT": "y",
				"CONFIG_NFS_V4":   "m",
				"CONFIG_NFS_V4_1": "m",
				"CONFIG_NFS_V4_2": "y",
			},
			expectedError: false,
		},
		"Empty kernel config": {
			setup:             func(dir string) { genKernelConfig(dir) },
			expectedConfigMap: map[string]string{},
			expectedError:     false,
		},
		"Invalid content": {
			setup:             func(dir string) { genKernelConfig(dir, "key=val\nCONFIG_invalid_content\n") },
			expectedConfigMap: nil,
			expectedError:     true,
		},
	}

	procDir := fake.CreateTempDirectory("", t)

	realConfigPath := filepath.Join(types.SysProcDirectory, types.SysKernelConfigGz)
	if _, err := os.ReadFile(realConfigPath); err == nil {
		testCases["Real config"] = testCase{
			setup: func(dir string) {
				err := io.CopyFile(realConfigPath, filepath.Join(dir, types.SysKernelConfigGz), true)
				assert.NoError(t, err)
			},
			expectedConfigMapChecker: func(cm map[string]string) bool {
				return len(cm) > 0
			},
			expectedError: false,
		}
	}

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			testCase.setup(procDir)

			exact, err := GetProcKernelConfigMap(procDir)
			if testCase.expectedConfigMapChecker == nil {
				assert.Equal(t, testCase.expectedConfigMap, exact, Commentf(test.ErrResultFmt, testName))
			} else {
				assert.True(t, testCase.expectedConfigMapChecker(exact), Commentf(test.ErrResultFmt, testName))
			}
			if testCase.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName, err))
			}
		})

	}
}
