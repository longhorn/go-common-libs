package sys

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	. "gopkg.in/check.v1"

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
		c.Logf("testing utils.%v", testName)

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
		c.Logf("testing utils.%v", testName)

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
		c.Logf("testing utils.%v", testName)

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
		c.Logf("testing utils.%v", testName)

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
