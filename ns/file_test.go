package ns

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/longhorn/go-common-libs/test/fake"
	"github.com/longhorn/go-common-libs/types"
)

func testCaseCopyDirectory(t *testing.T) map[string]testCaseNamespaceMethods {
	return map[string]testCaseNamespaceMethods{
		"CopyDirectory/Failed to run": {
			method: func(args ...interface{}) (interface{}, error) {
				return nil, CopyDirectory("test", "test", false)
			},
			mockError:   fmt.Errorf("failed"),
			expectError: true,
		},
	}
}

func testCaseCreateDirectory(t *testing.T) map[string]testCaseNamespaceMethods {
	return map[string]testCaseNamespaceMethods{
		"CreateDirectory/Success": {
			method: func(args ...interface{}) (interface{}, error) {
				return CreateDirectory("test", time.Now())
			},
			mockResult: "result",
		},
		"CreateDirectory/Failed to run": {
			method: func(args ...interface{}) (interface{}, error) {
				return CreateDirectory("test", time.Now())
			},
			mockError:   fmt.Errorf("failed"),
			expectError: true,
		},
	}
}

func testCaseDeleteDirectory(t *testing.T) map[string]testCaseNamespaceMethods {
	return map[string]testCaseNamespaceMethods{
		"DeleteDirectory/Failed to run": {
			method: func(args ...interface{}) (interface{}, error) {
				return nil, DeleteDirectory("test")
			},
			mockError:   fmt.Errorf("failed"),
			expectError: true,
		},
	}
}

func testCaseReadDirectory(t *testing.T) map[string]testCaseNamespaceMethods {
	mockResult, err := os.ReadDir("/tmp")
	assert.NoError(t, err)

	return map[string]testCaseNamespaceMethods{
		"ReadDirectory/Failed to run": {
			method: func(args ...interface{}) (interface{}, error) {
				return ReadDirectory("test")
			},
			mockResult: mockResult,
		},
	}
}

func testCaseCopyFiles(t *testing.T) map[string]testCaseNamespaceMethods {
	return map[string]testCaseNamespaceMethods{
		"CopyFiles/Failed to run": {
			method: func(args ...interface{}) (interface{}, error) {
				return nil, CopyFiles("test", "test", false)
			},
			mockError:   fmt.Errorf("failed"),
			expectError: true,
		},
	}
}

func testCaseGetEmptyFiles(t *testing.T) map[string]testCaseNamespaceMethods {
	return map[string]testCaseNamespaceMethods{
		"GetEmptyFiles/Success": {
			method: func(args ...interface{}) (interface{}, error) {
				return GetEmptyFiles("test")
			},
			mockResult: []string{"test"},
		},
		"GetEmptyFiles/Failed to run": {
			method: func(args ...interface{}) (interface{}, error) {
				return GetEmptyFiles("test")
			},
			mockError:   fmt.Errorf("failed"),
			expectError: true,
		},
		"GetEmptyFiles/Failed to cast result": {
			method: func(args ...interface{}) (interface{}, error) {
				return GetEmptyFiles("test")
			},
			mockResult:  int(1),
			expectError: true,
		},
	}
}

func testCaseGetFileInfo(t *testing.T) map[string]testCaseNamespaceMethods {
	fakeDir := fake.CreateTempDirectory("", t)
	defer func() {
		errRemove := os.RemoveAll(fakeDir)
		assert.NoError(t, errRemove)
	}()

	fakeFile := fake.CreateTempFile(fakeDir, "", "content", t)
	defer func() {
		errClose := fakeFile.Close()
		assert.NoError(t, errClose)
	}()

	mockResult, err := fakeFile.Stat()
	assert.NoError(t, err)

	return map[string]testCaseNamespaceMethods{
		"GetFileInfo/Success": {
			method: func(args ...interface{}) (interface{}, error) {
				return GetFileInfo("test")
			},
			mockResult: mockResult,
		},
		"GetFileInfo/Failed to run": {
			method: func(args ...interface{}) (interface{}, error) {
				return GetFileInfo("test")
			},
			mockError:   fmt.Errorf("failed"),
			expectError: true,
		},
		"GetFileInfo/Failed to cast result": {
			method: func(args ...interface{}) (interface{}, error) {
				return GetFileInfo("test")
			},
			mockResult:  int(1),
			expectError: true,
		},
	}
}

func testCaseReadFileContent(t *testing.T) map[string]testCaseNamespaceMethods {
	return map[string]testCaseNamespaceMethods{
		"ReadFileContent/Success": {
			method: func(args ...interface{}) (interface{}, error) {
				return ReadFileContent("test")
			},
			mockResult: "result",
		},
		"ReadFileContent/Failed to run": {
			method: func(args ...interface{}) (interface{}, error) {
				return ReadFileContent("test")
			},
			mockError:   fmt.Errorf("failed"),
			expectError: true,
		},
		"ReadFileContent/Failed to cast result": {
			method: func(args ...interface{}) (interface{}, error) {
				return ReadFileContent("test")
			},
			mockResult:  int(1),
			expectError: true,
		},
	}
}

func testCaseSyncFile(t *testing.T) map[string]testCaseNamespaceMethods {
	return map[string]testCaseNamespaceMethods{
		"SyncFile/Failed to run": {
			method: func(args ...interface{}) (interface{}, error) {
				return nil, SyncFile("test")
			},
			mockError:   fmt.Errorf("failed"),
			expectError: true,
		},
	}
}

func testCaseWriteFile(t *testing.T) map[string]testCaseNamespaceMethods {
	return map[string]testCaseNamespaceMethods{
		"WriteFile/Failed to run": {
			method: func(args ...interface{}) (interface{}, error) {
				return nil, WriteFile("test", "test")
			},
			mockError:   fmt.Errorf("failed"),
			expectError: true,
		},
	}
}

func testCaseDeletePath(t *testing.T) map[string]testCaseNamespaceMethods {
	return map[string]testCaseNamespaceMethods{
		"DeletePath/Failed to run": {
			method: func(args ...interface{}) (interface{}, error) {
				return nil, DeletePath("test")
			},
			mockError:   fmt.Errorf("failed"),
			expectError: true,
		},
	}
}

func testCaseGetDiskStat(t *testing.T) map[string]testCaseNamespaceMethods {
	return map[string]testCaseNamespaceMethods{
		"GetDiskStat/Success": {
			method: func(args ...interface{}) (interface{}, error) {
				return GetDiskStat("test")
			},
			mockResult: types.DiskStat{},
		},
		"GetDiskStat/Failed to run": {
			method: func(args ...interface{}) (interface{}, error) {
				return GetDiskStat("test")
			},
			mockError:   fmt.Errorf("failed"),
			expectError: true,
		},
		"GetDiskStat/Failed to cast result": {
			method: func(args ...interface{}) (interface{}, error) {
				return GetDiskStat("test")
			},
			mockResult:  int(1),
			expectError: true,
		},
	}
}
