package io

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	. "gopkg.in/check.v1"

	"github.com/longhorn/go-common-libs/exec"
	"github.com/longhorn/go-common-libs/test"
	"github.com/longhorn/go-common-libs/test/fake"
	"github.com/longhorn/go-common-libs/types"
)

func TestCreateDirectory(t *testing.T) {
	fakeDir := fake.CreateTempDirectory("", t)
	defer func() {
		_ = os.RemoveAll(fakeDir)
	}()

	type testCase struct {
		modTime time.Time

		isExistingDir      bool
		existingDirModTime time.Time

		expectedModTime time.Time
		expectError     bool
	}
	testCases := map[string]testCase{
		"Not existing directory": {
			modTime: time.Date(2023, time.July, 1, 12, 0, 0, 0, time.UTC),
		},
		"Existing directory": {
			modTime:            time.Now(),
			isExistingDir:      true,
			existingDirModTime: time.Date(2023, time.July, 1, 12, 0, 0, 0, time.UTC),
			expectedModTime:    time.Date(2023, time.July, 1, 12, 0, 0, 0, time.UTC),
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			if testCase.expectedModTime.IsZero() {
				testCase.expectedModTime = testCase.modTime
			}

			dirPath := filepath.Join(fakeDir, fmt.Sprintf("test-%v", time.Now().UnixNano()))

			if testCase.isExistingDir {
				_, err := CreateDirectory(dirPath, testCase.existingDirModTime)
				assert.NoError(t, err)
			}

			createdPath, err := CreateDirectory(dirPath, testCase.modTime)
			if testCase.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName, err))
			assert.Equal(t, dirPath, createdPath, Commentf(test.ErrResultFmt, testName))

			fileInfo, err := os.Stat(createdPath)
			assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName, err))
			assert.Equal(t, fileInfo.ModTime().Equal(testCase.expectedModTime), true,
				Commentf("Unexpected mod time for test case: %s: expected: %v, got: %v",
					testName, testCase.expectedModTime, fileInfo.ModTime()),
			)
		})
	}
}

func TestCopyDirectory(t *testing.T) {
	fakeSourceParentDir := fake.CreateTempDirectory("", t)
	defer func() {
		_ = os.RemoveAll(fakeSourceParentDir)
	}()

	fakeDestParentDir := fake.CreateTempDirectory("", t)
	defer func() {
		_ = os.RemoveAll(fakeDestParentDir)
	}()

	fakeFileNameFmt := "file-%v.temp"

	type testCase struct {
		doOverWrite bool

		notExistingDestDirName string

		expectError bool
	}
	testCases := map[string]testCase{
		"Existing directory without overwrite": {},
		"Not existing destination directory": {
			notExistingDestDirName: "should-create",
			doOverWrite:            true,
		},
		"Do overwrite": {
			doOverWrite: true,
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			fakeSourceDir := fake.CreateTempDirectory(fakeSourceParentDir, t)
			fakeSourceFiles := make([]string, 3)
			for i := 0; i < 3; i++ {
				fakeSourceFile := fake.CreateTempFile(fakeSourceDir, fmt.Sprintf(fakeFileNameFmt, i), fmt.Sprintf("test-%v", i), t)
				fakeSourceFiles[i] = fakeSourceFile.Name()
				_ = fakeSourceFile.Close()
			}

			fakeDestDir := filepath.Join(fakeDestParentDir, testCase.notExistingDestDirName)
			if testCase.notExistingDestDirName == "" {
				fakeDestDir = fake.CreateTempDirectory(fakeDestParentDir, t)
			}

			if !testCase.doOverWrite {
				for i := range fakeSourceFiles {
					fake.CreateTempFile(fakeDestDir, fmt.Sprintf(fakeFileNameFmt, i), fmt.Sprintf("do-not-overwrite-%v", i), t)
				}
			}

			err := CopyDirectory(fakeSourceDir, fakeDestDir, testCase.doOverWrite)
			if testCase.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName, err))

			for i, sourceFile := range fakeSourceFiles {
				destFile := filepath.Join(fakeDestDir, filepath.Base(sourceFile))
				content, err := os.ReadFile(destFile)
				assert.NoError(t, err)

				if !testCase.doOverWrite {
					assert.Equal(t, string(content), fmt.Sprintf("do-not-overwrite-%v", i))
				} else {
					assert.Equal(t, string(content), fmt.Sprintf("test-%v", i))
				}
			}

		})

	}
}

func TestCopyFiles(t *testing.T) {
	sourceParentDir := fake.CreateTempDirectory("", t)
	defer func() {
		_ = os.RemoveAll(sourceParentDir)
	}()

	destParentDir := fake.CreateTempDirectory("", t)
	defer func() {
		_ = os.RemoveAll(destParentDir)
	}()

	fileNameFmt := "file-%v.temp"

	type testCase struct {
		doOverWrite bool

		isSourceAFile bool
		isInSubDirs   bool

		notExistingDestDirName   string
		notExistingSourceDirName string

		expectError bool
	}
	testCases := map[string]testCase{
		"Copy files to existing directory": {},
		"Copy files in subdirectories": {
			isInSubDirs: true,
		},
		"Copy single file instead of directory": {
			isSourceAFile: true,
		},
		"Fails when source directory does not exist": {
			notExistingSourceDirName: "not-existing",
			expectError:              true,
		},
		"Create destination directory if it does not exist": {
			notExistingDestDirName: "should-create",
			doOverWrite:            true,
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			sourceSubDirName := ""
			sourceFiles := make([]string, 3)

			sourceDir := filepath.Join(sourceParentDir, testCase.notExistingSourceDirName)
			if testCase.notExistingSourceDirName == "" {
				sourceDir = fake.CreateTempDirectory(sourceParentDir, t)

				if testCase.isInSubDirs {
					sourceSubDirName = path.Base(fake.CreateTempDirectory(sourceDir, t))
				}

				fileDir := path.Join(sourceDir, sourceSubDirName)

				for i := 0; i < 3; i++ {
					sourceFile := fake.CreateTempFile(fileDir, fmt.Sprintf(fileNameFmt, i), fmt.Sprintf("test-%v", i), t)
					sourceFiles[i] = sourceFile.Name()
					_ = sourceFile.Close()
				}
			}

			destDir := filepath.Join(destParentDir, testCase.notExistingDestDirName)
			if testCase.notExistingDestDirName == "" {
				destDir = fake.CreateTempDirectory(destParentDir, t)
			}

			if !testCase.doOverWrite {
				for i := range sourceFiles {
					destFileDir := filepath.Join(destDir, sourceSubDirName)

					err := os.MkdirAll(destFileDir, 0755)
					assert.NoError(t, err)

					fake.CreateTempFile(destFileDir, fmt.Sprintf(fileNameFmt, i), fmt.Sprintf("do-not-overwrite-%v", i), t)
				}
			}

			if testCase.isSourceAFile {
				for _, sourceFile := range sourceFiles {
					destFile := filepath.Join(destDir, filepath.Base(sourceFile))
					err := CopyFiles(sourceFile, destFile, testCase.doOverWrite)
					if testCase.expectError {
						assert.Error(t, err)
						continue
					}
					assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName, err))
				}
			} else {
				err := CopyFiles(sourceDir, destDir, testCase.doOverWrite)
				if testCase.expectError {
					assert.Error(t, err)
					return
				}
				assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName, err))
			}

			for i, sourceFile := range sourceFiles {
				destFile := filepath.Join(destDir, sourceSubDirName, path.Base(sourceFile))

				content, err := os.ReadFile(destFile)
				assert.NoError(t, err)

				if !testCase.doOverWrite {
					assert.Equal(t, string(content), fmt.Sprintf("do-not-overwrite-%v", i))
				} else {
					assert.Equal(t, string(content), fmt.Sprintf("test-%v", i))
				}
			}
		})

	}
}

func TestCopyFile(t *testing.T) {
	fakeSourceParentDir := fake.CreateTempDirectory("", t)
	defer func() {
		_ = os.RemoveAll(fakeSourceParentDir)
	}()

	fakeDestParentDir := fake.CreateTempDirectory("", t)
	defer func() {
		_ = os.RemoveAll(fakeDestParentDir)
	}()

	type testCase struct {
		doOverWrite bool
		sparseSize  int64

		notExistingSourceFileName string
		notExistingDestDirName    string

		expectError      bool
		expectedSameSize bool
	}
	testCases := map[string]testCase{
		"Basic copy": {},
		"Fails if source file does not exist": {
			notExistingSourceFileName: "not-existing",
			expectError:               true,
		},
		"Creates destination directory if it does not exist": {
			notExistingDestDirName: "should-create",
			doOverWrite:            true,
			expectedSameSize:       true,
		},
		"Overwrite": {
			doOverWrite:      true,
			expectedSameSize: true,
		},
		"Handle sparse file": {
			doOverWrite:      true,
			sparseSize:       4097,
			expectedSameSize: false,
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			fakeSourceDir := fake.CreateTempDirectory(fakeSourceParentDir, t)
			fakeSourceFile := filepath.Join(fakeSourceDir, testCase.notExistingSourceFileName)
			if testCase.notExistingSourceFileName == "" {
				var fakeFile *os.File
				if testCase.sparseSize != 0 {
					fakeFile = fake.CreateTempSparseFile(fakeSourceDir, fmt.Sprintf("test-%v", time.Now().UnixNano()), testCase.sparseSize, "content", t)
				} else {
					fakeFile = fake.CreateTempFile(fakeSourceDir, fmt.Sprintf("test-%v", time.Now().UnixNano()), "content", t)
				}
				fakeSourceFile = fakeFile.Name()
				_ = fakeFile.Close()
			}

			fakeDestDir := filepath.Join(fakeDestParentDir, testCase.notExistingDestDirName)
			if testCase.notExistingDestDirName == "" {
				fakeDestDir = fake.CreateTempDirectory(fakeDestParentDir, t)
			}

			if !testCase.doOverWrite && testCase.notExistingDestDirName == "" {
				fake.CreateTempFile(fakeDestDir, filepath.Base(fakeSourceFile), "do-not-overwrite", t)
			}

			fakeDestPath := filepath.Join(fakeDestDir, filepath.Base(fakeSourceFile))
			err := CopyFile(fakeSourceFile, fakeDestPath, testCase.doOverWrite)
			if testCase.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName, err))

			destFile := filepath.Join(fakeDestDir, filepath.Base(fakeSourceFile))
			content, err := os.ReadFile(destFile)
			assert.NoError(t, err)

			if !testCase.doOverWrite {
				assert.Equal(t, string(content), "do-not-overwrite")
			} else {
				expectedContent := "content"
				if testCase.sparseSize != 0 {
					expectedContent = expectedContent + strings.Repeat("\x00", int(testCase.sparseSize)-len(expectedContent))
				}
				assert.Equal(t, string(content), expectedContent)
				err := CheckIsFileSizeSame(destFile, fakeSourceFile)
				if testCase.expectedSameSize {
					assert.NoError(t, err)
				} else {
					assert.Error(t, err)
				}
			}
		})

	}
}

func TestFindFiles(t *testing.T) {
	fakeDir := fake.CreateTempDirectory("", t)
	defer func() {
		_ = os.RemoveAll(fakeDir)
	}()

	// Prepare sub directory
	fakeDirSub := fake.CreateTempDirectory(fakeDir, t)

	// Prepare 2 existing files in root of the fake directory,
	// and 2 existing file in sub directory.
	existingFileCount := 2
	existingFilePaths := make(map[string]bool, 3)
	existingFilePaths[fakeDir] = true
	existingFilePaths[fakeDirSub] = true
	for _, dir := range []string{fakeDir, fakeDirSub} {
		for i := 0; i < existingFileCount; i++ {
			file := fake.CreateTempFile(dir, fmt.Sprintf("test-%v", i), "content", t)
			existingFilePaths[file.Name()] = true
			_ = file.Close()
		}
	}

	type testCase struct {
		findFileWithName string
		maxDepth         int

		expectedFilePaths []string
		expectError       bool
	}
	testCases := map[string]testCase{
		"Find all files": {
			expectedFilePaths: []string{
				fakeDir,
				filepath.Join(fakeDir, "test-0"),
				filepath.Join(fakeDir, "test-1"),
				fakeDirSub,
				filepath.Join(fakeDirSub, "test-0"),
				filepath.Join(fakeDirSub, "test-1"),
			},
		},
		"Find file with name": {
			findFileWithName: "test-0",
			expectedFilePaths: []string{
				filepath.Join(fakeDir, "test-0"),
				filepath.Join(fakeDirSub, "test-0"),
			},
		},
		"Max depth": {
			maxDepth: 1,
			expectedFilePaths: []string{
				fakeDir,
				fakeDirSub,
				filepath.Join(fakeDir, "test-0"),
				filepath.Join(fakeDir, "test-1"),
			},
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			result, err := FindFiles(fakeDir, testCase.findFileWithName, testCase.maxDepth)
			if testCase.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName, err))
			assert.Equal(t, len(result), len(testCase.expectedFilePaths), Commentf(test.ErrResultFmt, testName))
			for _, filePath := range result {
				assert.True(t, existingFilePaths[filePath])
			}
		})

	}
}

func TestGetEmptyFiles(t *testing.T) {
	fakeDir := fake.CreateTempDirectory("", t)
	defer func() {
		_ = os.RemoveAll(fakeDir)
	}()

	fakeSubDir := fake.CreateTempDirectory(fakeDir, t)

	fileWithContent := fake.CreateTempFile(fakeDir, "regular-file", "content", t)
	err := fileWithContent.Close()
	assert.NoError(t, err)

	fileWithoutContent := fake.CreateTempFile(fakeDir, "empty-file-0", "", t)
	err = fileWithoutContent.Close()
	assert.NoError(t, err)

	fileWithoutContentInSubDir := fake.CreateTempFile(fakeSubDir, "empty-file-1", "", t)
	defer func() {
		_ = fileWithoutContentInSubDir.Close()
	}()

	type testCase struct {
		directory      string
		expectedResult map[string]bool
		expectError    bool
	}
	testCases := map[string]testCase{
		"Valid directory": {
			expectedResult: map[string]bool{
				fileWithoutContent.Name():         true,
				fileWithoutContentInSubDir.Name(): true,
			},
		},
		"Not existing directory": {
			directory:   "not-existing-directory",
			expectError: true,
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			if testCase.directory == "" {
				testCase.directory = fakeDir
			}
			result, err := GetEmptyFiles(testCase.directory)
			if testCase.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName, err))
			assert.Equal(t, len(result), len(testCase.expectedResult), Commentf(test.ErrResultFmt, testName))
			for _, filePath := range result {
				assert.True(t, testCase.expectedResult[filePath])
			}
		})
	}
}

func TestReadFileContent(t *testing.T) {
	fakeDir := fake.CreateTempDirectory("", t)
	defer func() {
		_ = os.RemoveAll(fakeDir)
	}()

	fileContentFmt := "test: %v"

	type testCase struct {
		isFileExist bool

		expectError bool
	}
	testCases := map[string]testCase{
		"Valid file": {
			isFileExist: true,
		},
		"Not existing file": {
			isFileExist: false,
			expectError: true,
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			expectedContent := fmt.Sprintf(fileContentFmt, testName)

			filePath := filepath.Join(fakeDir, "not-exist")
			if testCase.isFileExist {
				file := fake.CreateTempFile(fakeDir, "", expectedContent, t)
				filePath = file.Name()
				_ = file.Close()
			}

			content, err := ReadFileContent(filePath)
			if testCase.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName, err))
			assert.Equal(t, content, expectedContent)
		})
	}
}

func TestSyncFile(t *testing.T) {
	fakeDir := fake.CreateTempDirectory("", t)
	defer func() {
		_ = os.RemoveAll(fakeDir)
	}()

	type testCase struct {
		isFileExist bool

		expectError bool
	}
	testCases := map[string]testCase{
		"Existing file": {
			isFileExist: true,
		},
		"Not existing file": {
			isFileExist: false,
			expectError: true,
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			filePath := filepath.Join(fakeDir, "not-exist")
			if testCase.isFileExist {
				file := fake.CreateTempFile(fakeDir, "", "content", t)
				filePath = file.Name()
				_ = file.Close()
			}

			err := SyncFile(filePath)
			if testCase.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName, err))
		})
	}
}

func TestGetDiskStat(t *testing.T) {
	fakeDir := fake.CreateTempDirectory("", t)
	defer func() {
		_ = os.RemoveAll(fakeDir)
	}()

	type testCase struct {
		isPathExist bool
		expectError bool
	}
	testCases := map[string]testCase{
		"Existing path": {
			isPathExist: true,
		},
		"Not existing path": {
			isPathExist: false,
			expectError: true,
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			testDir := fake.CreateTempDirectory(fakeDir, t)
			if !testCase.isPathExist {
				_ = os.RemoveAll(testDir)
			}

			diskStat, err := GetDiskStat(testDir)
			if testCase.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName, err))

			expectedDiskStat, err := getDiskStat(testDir)
			assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName, err))

			// On the running system, FreeBlocks/StorageAvailable might be changing with time.
			// So we only compare the following fields
			assert.Equal(t, diskStat.DiskID, expectedDiskStat.DiskID)
			assert.Equal(t, diskStat.Path, expectedDiskStat.Path)

			// FIXME: overlayfs is not supported in the github.com/shirou/gopsutil/v3
			if expectedDiskStat.Type != "overlayfs" {
				assert.Equal(t, diskStat.Type, expectedDiskStat.Type)
			}

			assert.Equal(t, diskStat.TotalBlocks, expectedDiskStat.TotalBlocks)
			assert.Equal(t, diskStat.BlockSize, expectedDiskStat.BlockSize)
			assert.Equal(t, diskStat.StorageMaximum, expectedDiskStat.StorageMaximum)
		})
	}
}

func getDiskStat(path string) (*types.DiskStat, error) {
	args := []string{"-fc", "{\"path\":\"%n\",\"fsid\":\"%i\",\"type\":\"%T\",\"freeBlock\":%f,\"totalBlock\":%b,\"blockSize\":%S}", path}
	output, err := exec.NewExecutor().Execute(nil, "stat", args, types.ExecuteDefaultTimeout)
	if err != nil {
		return nil, err
	}
	output = strings.ReplaceAll(output, "\n", "")

	type FsStat struct {
		Fsid       string
		Path       string
		Type       string
		FreeBlock  int64
		TotalBlock int64
		BlockSize  int64
	}
	fsStat := &FsStat{}
	err = json.Unmarshal([]byte(output), fsStat)
	if err != nil {
		return nil, err
	}

	return &types.DiskStat{
		DiskID:           fsStat.Fsid,
		Path:             fsStat.Path,
		Type:             fsStat.Type,
		Driver:           types.DiskDriverNone,
		FreeBlocks:       fsStat.FreeBlock,
		TotalBlocks:      fsStat.TotalBlock,
		BlockSize:        fsStat.BlockSize,
		StorageMaximum:   fsStat.TotalBlock * fsStat.BlockSize,
		StorageAvailable: fsStat.FreeBlock * fsStat.BlockSize,
	}, nil
}

func TestListOpenFiles(t *testing.T) {
	fakeDir := fake.CreateTempDirectory("", t)
	defer func() {
		_ = os.RemoveAll(fakeDir)
	}()

	type _dirInfo struct {
		dir   string
		files []*os.File
	}

	type _fakeDirs struct {
		open  _dirInfo
		close _dirInfo
	}

	// create fake directories:
	// 1. open: contains 2 opened files
	// 2. close: contains 2 closed files
	fakeDirs := _fakeDirs{
		open: _dirInfo{
			dir: fake.CreateTempDirectory(fakeDir, t),
		},
		close: _dirInfo{
			dir: fake.CreateTempDirectory(fakeDir, t),
		},
	}

	fakeDirs.open.files = append(fakeDirs.open.files, fake.CreateTempFile(fakeDirs.open.dir, "file1", "content", t))
	fakeDirs.open.files = append(fakeDirs.open.files, fake.CreateTempFile(fakeDirs.open.dir, "file2", "content", t))
	defer func() {
		for _, file := range fakeDirs.open.files {
			err := file.Close()
			assert.NoError(t, err)
		}
	}()

	// Create and close files in the close directory
	fakeDirs.close.files = append(fakeDirs.close.files, fake.CreateTempFile(fakeDirs.close.dir, "file1", "content", t))
	fakeDirs.close.files = append(fakeDirs.close.files, fake.CreateTempFile(fakeDirs.close.dir, "file2", "content", t))
	for _, file := range fakeDirs.close.files {
		err := file.Close()
		assert.NoError(t, err)
	}

	time.Sleep(100 * time.Millisecond)

	type testCase struct {
		directory         string
		expectedOpenFiles []string
		expectError       bool
	}
	testCases := map[string]testCase{
		"Existing directory with open files": {
			directory: fakeDir,
			expectedOpenFiles: []string{
				filepath.Join(fakeDirs.open.dir, "file1"),
				filepath.Join(fakeDirs.open.dir, "file2"),
			},
		},
		"Not existing path": {
			directory:   "not-existing-path",
			expectError: true,
		},
		"No open files": {
			directory:   fakeDirs.close.dir,
			expectError: false,
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			if testCase.directory == "" {
				testCase.directory = fakeDir
			}

			openFiles, err := ListOpenFiles("/proc", testCase.directory)
			if testCase.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, len(openFiles), len(testCase.expectedOpenFiles),
				Commentf(test.ErrResultFmt, fmt.Sprintf("%s: %v", testName, openFiles)),
			)
			for i, openFile := range openFiles {
				assert.Equal(t, openFile, testCase.expectedOpenFiles[i])
			}
		})
	}
}

func TestIsDirectoryEmpty(t *testing.T) {
	fakeDir := fake.CreateTempDirectory("", t)
	defer func() {
		_ = os.RemoveAll(fakeDir)
	}()

	type testCase struct {
		isEmpty      bool
		isNotExist   bool
		expectError  bool
		expectResult bool
	}
	testCases := map[string]testCase{
		"Empty directory": {
			isEmpty:      true,
			expectResult: true,
		},
		"Not empty directory": {
			isEmpty:      false,
			expectResult: false,
		},
		"Not existing path": {
			isNotExist:   true,
			expectError:  true,
			expectResult: false,
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			testDir := fake.CreateTempDirectory(fakeDir, t)

			if !testCase.isEmpty {
				fake.CreateTempFile(testDir, "file", "content", t)
			}

			if testCase.isNotExist {
				err := os.RemoveAll(testDir)
				assert.NoError(t, err)
			}

			result, err := IsDirectoryEmpty(testDir)
			if testCase.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, result, testCase.expectResult)
		})
	}
}

func TestCheckIsFileSizeSame(t *testing.T) {
	fakeDir := fake.CreateTempDirectory("", t)
	defer func() {
		_ = os.RemoveAll(fakeDir)
	}()

	type testCase struct {
		isDifferent  bool
		isDirectory  bool
		notFileExist bool

		expectError bool
	}
	testCases := map[string]testCase{
		"File sizes are same": {},
		"Not existing path": {
			notFileExist: true,
			expectError:  true,
		},
		"Different size": {
			isDifferent: true,
			expectError: true,
		},
		"Directory": {
			isDirectory: true,
			expectError: true,
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			testDir := fake.CreateTempDirectory(fakeDir, t)

			fileName1 := "file1"
			fileName2 := "file2"

			var file1 *os.File
			var file2 *os.File
			switch {
			case testCase.isDifferent:
				file1 = fake.CreateTempFile(testDir, fileName1, "content", t)
				file2 = fake.CreateTempFile(testDir, fileName2, "different-content", t)
			default:
				file1 = fake.CreateTempFile(testDir, fileName1, "content", t)
				file2 = fake.CreateTempFile(testDir, fileName2, "content", t)
			}

			if testCase.notFileExist {
				err := os.RemoveAll(testDir)
				assert.NoError(t, err)
			}

			var err error
			if testCase.isDirectory {
				err = CheckIsFileSizeSame(file1.Name(), file2.Name(), testDir)
			} else {
				err = CheckIsFileSizeSame(file1.Name(), file2.Name())
			}
			if testCase.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName, err))
		})
	}
}
