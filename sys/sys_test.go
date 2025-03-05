package sys

import (
	"compress/gzip"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	. "gopkg.in/check.v1"

	"github.com/longhorn/go-common-libs/io"
	"github.com/longhorn/go-common-libs/test"
	"github.com/longhorn/go-common-libs/test/fake"
	"github.com/longhorn/go-common-libs/types"
)

func Test(t *testing.T) { TestingT(t) }

type TestSuite struct{}

var _ = Suite(&TestSuite{})

func (s *TestSuite) TestGetKernelRelease(c *C) {
	type testCase struct{}
	testCases := map[string]testCase{
		"GetKernelRelease(...)": {},
	}
	for testName := range testCases {
		c.Logf("testing sys.%v", testName)

		result, err := GetKernelRelease()
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName, err))
		c.Assert(result, Not(Equals), "", Commentf(test.ErrResultFmt, testName))
	}
}

// TestGetHostOSDistro tests the success cases of GetHostOSDistro
func (s *TestSuite) TestGetHostOSDistro(c *C) {
	type testCase struct {
		mockFileContent string

		expected string
	}
	testCases := map[string]testCase{
		"GetOSDistro(...)": {
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
		c.Logf("testing sys.%v", testName)

		result, err := GetOSDistro(testCase.mockFileContent)
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName, err))
		c.Assert(result, Equals, testCase.expected, Commentf(test.ErrResultFmt, testName))
	}
}

// TestGetHostOSDistroFailures tests the failure cases of GetHostOSDistro
// it cannot be run in TestGetHostOSDistro because the cache is not cleared
// between tests
func (s *TestSuite) TestGetHostOSDistroFailures(c *C) {
	type testCase struct {
		mockFileContent string
	}
	testCases := map[string]testCase{
		"GetOSDistro(...): missing ID": {
			mockFileContent: `NAME="SLES"
VERSION="15-SP3"
VERSION_ID="15.3"
PRETTY_NAME="SUSE Linux Enterprise Server 15 SP3"
ID_LIKE="suse"`,
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing sys.%v", testName)

		_, err := GetOSDistro(testCase.mockFileContent)
		c.Assert(err, NotNil)
	}
}

func (s *TestSuite) TestGetSystemBlockDevices(c *C) {
	fakeDir := fake.CreateTempDirectory("", c)
	defer func() {
		_ = os.RemoveAll(fakeDir)
	}()

	type testCase struct {
		mockDirEntries []os.DirEntry
		mockData       []byte

		expected map[string]types.BlockDeviceInfo
	}
	testCases := map[string]testCase{
		"getSystemBlockDeviceInfo(...)": {
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
		"getSystemBlockDeviceInfo(...): invalid file content": {
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
		c.Logf("testing sys.%v", testName)

		fakeFS := fake.FileSystem{
			DirEntries: testCase.mockDirEntries,
			Data:       testCase.mockData,
		}

		for _, entry := range testCase.mockDirEntries {
			// Create device directory
			deviceDir := filepath.Join(fakeDir, entry.Name())
			err := os.MkdirAll(deviceDir, 0755)
			c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName, err))

			// Create device file
			devicePath := filepath.Join(deviceDir, "dev")
			deviceFile, err := os.Create(devicePath)
			deviceFile.Close()
			c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName, err))
		}

		result, err := getSystemBlockDeviceInfo(fakeDir, fakeFS.ReadDir, fakeFS.ReadFile)
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName, err))
		c.Assert(reflect.DeepEqual(result, testCase.expected), Equals, true, Commentf(test.ErrResultFmt, testName))
	}
}

func (s *TestSuite) TestGetBootKernelConfigMap(c *C) {
	type testCase struct {
		mockFileContent   string
		expectedConfigMap map[string]string
		expectedError     bool
	}
	testCases := map[string]testCase{
		"GetBootKernelConfigMap(...): read kernel config": {
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
		"GetBootKernelConfigMap(...): empty kernel config": {
			mockFileContent:   "",
			expectedConfigMap: map[string]string{},
			expectedError:     false,
		},
		"GetBootKernelConfigMap(...): invalid content": {
			mockFileContent:   "key=val\nCONFIG_invalid_content\n",
			expectedConfigMap: nil,
			expectedError:     true,
		},
	}

	bootDir := c.MkDir()
	kernelVersion := "1.2.3.4"

	for testName, testCase := range testCases {
		c.Logf("testing sys.%v", testName)

		err := os.WriteFile(filepath.Join(bootDir, "config-"+kernelVersion), []byte(testCase.mockFileContent), 0644)
		c.Assert(err, IsNil)

		exact, err := GetBootKernelConfigMap(bootDir, kernelVersion)
		c.Assert(exact, DeepEquals, testCase.expectedConfigMap, Commentf(test.ErrResultFmt, testName))
		if testCase.expectedError {
			c.Assert(err, NotNil, Commentf(test.ErrErrorFmt, testName, err))
		} else {
			c.Assert(err, IsNil)
		}
	}
}
func (s *TestSuite) TestGetProcKernelConfigMap(c *C) {
	genKernelConfig := func(dir string, lines ...string) {
		path := filepath.Join(dir, types.SysKernelConfigGz)
		file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		c.Assert(err, IsNil)
		defer file.Close()
		gzWriter := gzip.NewWriter(file)
		defer gzWriter.Close()
		for _, line := range lines {
			_, err := fmt.Fprintln(gzWriter, line)
			c.Assert(err, IsNil)
		}
	}

	type testCase struct {
		setup                    func(dir string)
		expectedConfigMap        map[string]string
		expectedConfigMapChecker func(map[string]string) bool
		expectedError            bool
	}

	testCases := map[string]testCase{
		"GetProcKernelConfigMap(...): read kernel config": {
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
		"GetProcKernelConfigMap(...): empty kernel config": {
			setup:             func(dir string) { genKernelConfig(dir) },
			expectedConfigMap: map[string]string{},
			expectedError:     false,
		},
		"GetProcKernelConfigMap(...): invalid content": {
			setup:             func(dir string) { genKernelConfig(dir, "key=val\nCONFIG_invalid_content\n") },
			expectedConfigMap: nil,
			expectedError:     true,
		},
	}

	procDir := c.MkDir()

	realConfigPath := filepath.Join(types.SysProcDirectory, types.SysKernelConfigGz)
	if _, err := os.ReadFile(realConfigPath); err == nil {
		testCases["GetProcKernelConfigMap(...): real config"] = testCase{
			setup: func(dir string) {
				err := io.CopyFile(realConfigPath, filepath.Join(dir, types.SysKernelConfigGz), true)
				c.Assert(err, IsNil)
			},
			expectedConfigMapChecker: func(cm map[string]string) bool {
				return len(cm) > 0
			},
			expectedError: false,
		}
	}

	for testName, testCase := range testCases {
		c.Logf("testing sys.%v", testName)

		testCase.setup(procDir)

		exact, err := GetProcKernelConfigMap(procDir)
		if testCase.expectedConfigMapChecker == nil {
			c.Assert(exact, DeepEquals, testCase.expectedConfigMap, Commentf(test.ErrResultFmt, testName))
		} else {
			c.Assert(testCase.expectedConfigMapChecker(exact), Equals, true, Commentf(test.ErrResultFmt, testName))
		}
		if testCase.expectedError {
			c.Assert(err, NotNil, Commentf(test.ErrErrorFmt, testName, err))
		} else {
			c.Assert(err, IsNil)
		}
	}
}
