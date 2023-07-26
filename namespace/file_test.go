package namespace

import (
	"fmt"
	"os"
	"time"

	"github.com/longhorn/go-common-libs/fake"
	"github.com/longhorn/go-common-libs/types"

	. "gopkg.in/check.v1"
)

func testCaseCopyDirectory(c *C) map[string]testCaseNamespaceMethods {
	return map[string]testCaseNamespaceMethods{
		"CopyDirectory(...):": {
			method: func(args ...interface{}) (interface{}, error) {
				return nil, CopyDirectory("test", "test", false)
			},
			mockError:   fmt.Errorf("failed"),
			expectError: true,
		},
	}
}

func testCaseCreateDirectory(c *C) map[string]testCaseNamespaceMethods {
	return map[string]testCaseNamespaceMethods{
		"CreateDirectory(...):": {
			method: func(args ...interface{}) (interface{}, error) {
				return CreateDirectory("test", time.Now())
			},
			mockResult: "result",
		},
		"CreateDirectory(...): failed to run": {
			method: func(args ...interface{}) (interface{}, error) {
				return CreateDirectory("test", time.Now())
			},
			mockError:   fmt.Errorf("failed"),
			expectError: true,
		},
	}
}

func testCaseDeleteDirectory(c *C) map[string]testCaseNamespaceMethods {
	return map[string]testCaseNamespaceMethods{
		"DeleteDirectory(...): failed to run": {
			method: func(args ...interface{}) (interface{}, error) {
				return nil, DeleteDirectory("test")
			},
			mockError:   fmt.Errorf("failed"),
			expectError: true,
		},
	}
}

func testCaseReadDirectory(c *C) map[string]testCaseNamespaceMethods {
	mockResult, err := os.ReadDir("/tmp")
	c.Assert(err, IsNil)

	return map[string]testCaseNamespaceMethods{
		"ReadDirectory(...): failed to run": {
			method: func(args ...interface{}) (interface{}, error) {
				return ReadDirectory("test")
			},
			mockResult: mockResult,
		},
	}
}

func testCaseCopyFiles(c *C) map[string]testCaseNamespaceMethods {
	return map[string]testCaseNamespaceMethods{
		"CopyFiles(...): failed to run": {
			method: func(args ...interface{}) (interface{}, error) {
				return nil, CopyFiles("test", "test", false)
			},
			mockError:   fmt.Errorf("failed"),
			expectError: true,
		},
	}
}

func testCaseGetEmptyFiles(c *C) map[string]testCaseNamespaceMethods {
	return map[string]testCaseNamespaceMethods{
		"GetEmptyFiles(...)": {
			method: func(args ...interface{}) (interface{}, error) {
				return GetEmptyFiles("test")
			},
			mockResult: []string{"test"},
		},
		"GetEmptyFiles(...): failed to run": {
			method: func(args ...interface{}) (interface{}, error) {
				return GetEmptyFiles("test")
			},
			mockError:   fmt.Errorf("failed"),
			expectError: true,
		},
		"GetEmptyFiles(...): failed to cast result": {
			method: func(args ...interface{}) (interface{}, error) {
				return GetEmptyFiles("test")
			},
			mockResult:  int(1),
			expectError: true,
		},
	}
}

func testCaseGetFileInfo(c *C) map[string]testCaseNamespaceMethods {
	fakeDir := fake.CreateTempDirectory("", c)
	defer os.RemoveAll(fakeDir)

	fakeFile := fake.CreateTempFile(fakeDir, "", "content", c)
	defer fakeFile.Close()

	mockResult, err := fakeFile.Stat()
	c.Assert(err, IsNil)

	return map[string]testCaseNamespaceMethods{
		"GetFileInfo(...)": {
			method: func(args ...interface{}) (interface{}, error) {
				return GetFileInfo("test")
			},
			mockResult: mockResult,
		},
		"GetFileInfo(...): failed to run": {
			method: func(args ...interface{}) (interface{}, error) {
				return GetFileInfo("test")
			},
			mockError:   fmt.Errorf("failed"),
			expectError: true,
		},
		"GetFileInfo(...): failed to cast result": {
			method: func(args ...interface{}) (interface{}, error) {
				return GetFileInfo("test")
			},
			mockResult:  int(1),
			expectError: true,
		},
	}
}

func testCaseReadFileContent(c *C) map[string]testCaseNamespaceMethods {
	return map[string]testCaseNamespaceMethods{
		"ReadFileContent(...)": {
			method: func(args ...interface{}) (interface{}, error) {
				return ReadFileContent("test")
			},
			mockResult: "result",
		},
		"ReadFileContent(...): failed to run": {
			method: func(args ...interface{}) (interface{}, error) {
				return ReadFileContent("test")
			},
			mockError:   fmt.Errorf("failed"),
			expectError: true,
		},
		"ReadFileContent(...): failed to cast result": {
			method: func(args ...interface{}) (interface{}, error) {
				return ReadFileContent("test")
			},
			mockResult:  int(1),
			expectError: true,
		},
	}
}

func testCaseSyncFile(c *C) map[string]testCaseNamespaceMethods {
	return map[string]testCaseNamespaceMethods{
		"SyncFile(...): failed to run": {
			method: func(args ...interface{}) (interface{}, error) {
				return nil, SyncFile("test")
			},
			mockError:   fmt.Errorf("failed"),
			expectError: true,
		},
	}
}

func testCaseWriteFile(c *C) map[string]testCaseNamespaceMethods {
	return map[string]testCaseNamespaceMethods{
		"WriteFile(...): failed to run": {
			method: func(args ...interface{}) (interface{}, error) {
				return nil, WriteFile("test", "test")
			},
			mockError:   fmt.Errorf("failed"),
			expectError: true,
		},
	}
}

func testCaseDeletePath(c *C) map[string]testCaseNamespaceMethods {
	return map[string]testCaseNamespaceMethods{
		"DeletePath(...): failed to run": {
			method: func(args ...interface{}) (interface{}, error) {
				return nil, DeletePath("test")
			},
			mockError:   fmt.Errorf("failed"),
			expectError: true,
		},
	}
}

func testCaseGetDiskStat(c *C) map[string]testCaseNamespaceMethods {
	return map[string]testCaseNamespaceMethods{
		"GetDiskStat(...)": {
			method: func(args ...interface{}) (interface{}, error) {
				return GetDiskStat("test")
			},
			mockResult: types.DiskStat{},
		},
		"GetDiskStat(...): failed to run": {
			method: func(args ...interface{}) (interface{}, error) {
				return GetDiskStat("test")
			},
			mockError:   fmt.Errorf("failed"),
			expectError: true,
		},
		"GetDiskStat(...): failed to cast result": {
			method: func(args ...interface{}) (interface{}, error) {
				return GetDiskStat("test")
			},
			mockResult:  int(1),
			expectError: true,
		},
	}
}
