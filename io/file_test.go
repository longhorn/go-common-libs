package io

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	. "gopkg.in/check.v1"

	"github.com/longhorn/go-common-libs/exec"
	"github.com/longhorn/go-common-libs/test"
	"github.com/longhorn/go-common-libs/test/fake"
	"github.com/longhorn/go-common-libs/types"
)

func (s *TestSuite) TestCreateDirectory(c *C) {
	fakeDir := fake.CreateTempDirectory("", c)
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
		"CreateDirectory(...)": {
			modTime: time.Date(2023, time.July, 1, 12, 0, 0, 0, time.UTC),
		},
		"CreateDirectory(...) existing": {
			modTime:            time.Now(),
			isExistingDir:      true,
			existingDirModTime: time.Date(2023, time.July, 1, 12, 0, 0, 0, time.UTC),
			expectedModTime:    time.Date(2023, time.July, 1, 12, 0, 0, 0, time.UTC),
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing io.%v", testName)

		if testCase.expectedModTime.IsZero() {
			testCase.expectedModTime = testCase.modTime
		}

		dirPath := filepath.Join(fakeDir, fmt.Sprintf("test-%v", time.Now().UnixNano()))

		if testCase.isExistingDir {
			_, err := CreateDirectory(dirPath, testCase.existingDirModTime)
			c.Assert(err, IsNil)
		}

		createdPath, err := CreateDirectory(dirPath, testCase.modTime)
		if testCase.expectError {
			c.Assert(err, NotNil)
			continue
		}
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName, err))
		c.Assert(createdPath, Equals, dirPath, Commentf(test.ErrResultFmt, testName))

		fileInfo, err := os.Stat(createdPath)
		c.Assert(err, IsNil)
		c.Assert(
			fileInfo.ModTime().Equal(testCase.expectedModTime), Equals, true,
			Commentf("Unexpected mod time for test case: %s: expected: %v, got: %v",
				testName, testCase.expectedModTime, fileInfo.ModTime()),
		)
	}
}

func (s *TestSuite) TestCopyDirectory(c *C) {
	fakeSourceParentDir := fake.CreateTempDirectory("", c)
	defer func() {
		_ = os.RemoveAll(fakeSourceParentDir)
	}()

	fakeDestParentDir := fake.CreateTempDirectory("", c)
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
		"CopyDirectory(...)": {},
		"CopyDirectory(...): not existing destination directory": {
			notExistingDestDirName: "should-create",
			doOverWrite:            true,
		},
		"CopyDirectory(...): do overwrite": {
			doOverWrite: true,
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing io.%v", testName)

		fakeSourceDir := fake.CreateTempDirectory(fakeSourceParentDir, c)
		fakeSourceFiles := make([]string, 3)
		for i := 0; i < 3; i++ {
			fakeSourceFile := fake.CreateTempFile(fakeSourceDir, fmt.Sprintf(fakeFileNameFmt, i), fmt.Sprintf("test-%v", i), c)
			fakeSourceFiles[i] = fakeSourceFile.Name()
			_ = fakeSourceFile.Close()
		}

		fakeDestDir := filepath.Join(fakeDestParentDir, testCase.notExistingDestDirName)
		if testCase.notExistingDestDirName == "" {
			fakeDestDir = fake.CreateTempDirectory(fakeDestParentDir, c)
		}

		if !testCase.doOverWrite {
			for i := range fakeSourceFiles {
				fake.CreateTempFile(fakeDestDir, fmt.Sprintf(fakeFileNameFmt, i), fmt.Sprintf("do-not-overwrite-%v", i), c)
			}
		}

		err := CopyDirectory(fakeSourceDir, fakeDestDir, testCase.doOverWrite)
		if testCase.expectError {
			c.Assert(err, NotNil)
			continue
		}
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName, err))

		for i, sourceFile := range fakeSourceFiles {
			destFile := filepath.Join(fakeDestDir, filepath.Base(sourceFile))
			content, err := os.ReadFile(destFile)
			c.Assert(err, IsNil)

			if !testCase.doOverWrite {
				c.Assert(string(content), Equals, fmt.Sprintf("do-not-overwrite-%v", i))
			} else {
				c.Assert(string(content), Equals, fmt.Sprintf("test-%v", i))
			}
		}
	}
}

func (s *TestSuite) TestCopyFiles(c *C) {
	sourceParentDir := fake.CreateTempDirectory("", c)
	defer func() {
		_ = os.RemoveAll(sourceParentDir)
	}()

	destParentDir := fake.CreateTempDirectory("", c)
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
		"CopyFiles(...)": {},
		"CopyFiles(...): in sub directories": {
			isInSubDirs: true,
		},
		"CopyFiles(...): source is a file": {
			isSourceAFile: true,
		},
		"CopyFiles(...): not existing source directory": {
			notExistingSourceDirName: "not-existing",
			expectError:              true,
		},
		"CopyFiles(...): not existing destination directory": {
			notExistingDestDirName: "should-create",
			doOverWrite:            true,
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing io.%v", testName)

		sourceSubDirName := ""
		sourceFiles := make([]string, 3)

		sourceDir := filepath.Join(sourceParentDir, testCase.notExistingSourceDirName)
		if testCase.notExistingSourceDirName == "" {
			sourceDir = fake.CreateTempDirectory(sourceParentDir, c)

			if testCase.isInSubDirs {
				sourceSubDirName = path.Base(fake.CreateTempDirectory(sourceDir, c))
			}

			fileDir := path.Join(sourceDir, sourceSubDirName)

			for i := 0; i < 3; i++ {
				sourceFile := fake.CreateTempFile(fileDir, fmt.Sprintf(fileNameFmt, i), fmt.Sprintf("test-%v", i), c)
				sourceFiles[i] = sourceFile.Name()
				_ = sourceFile.Close()
			}
		}

		destDir := filepath.Join(destParentDir, testCase.notExistingDestDirName)
		if testCase.notExistingDestDirName == "" {
			destDir = fake.CreateTempDirectory(destParentDir, c)
		}

		if !testCase.doOverWrite {
			for i := range sourceFiles {
				destFileDir := filepath.Join(destDir, sourceSubDirName)

				err := os.MkdirAll(destFileDir, 0755)
				c.Assert(err, IsNil)

				fake.CreateTempFile(destFileDir, fmt.Sprintf(fileNameFmt, i), fmt.Sprintf("do-not-overwrite-%v", i), c)
			}
		}

		if testCase.isSourceAFile {
			for _, sourceFile := range sourceFiles {
				destFile := filepath.Join(destDir, filepath.Base(sourceFile))
				err := CopyFiles(sourceFile, destFile, testCase.doOverWrite)
				if testCase.expectError {
					c.Assert(err, NotNil)
					continue
				}
				c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName, err))
			}
		} else {
			err := CopyFiles(sourceDir, destDir, testCase.doOverWrite)
			if testCase.expectError {
				c.Assert(err, NotNil)
				continue
			}
			c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName, err))
		}

		for i, sourceFile := range sourceFiles {
			destFile := filepath.Join(destDir, sourceSubDirName, path.Base(sourceFile))

			content, err := os.ReadFile(destFile)
			c.Assert(err, IsNil)

			if !testCase.doOverWrite {
				c.Assert(string(content), Equals, fmt.Sprintf("do-not-overwrite-%v", i))
			} else {
				c.Assert(string(content), Equals, fmt.Sprintf("test-%v", i))
			}
		}
	}
}

func (s *TestSuite) TestCopyFile(c *C) {
	fakeSourceParentDir := fake.CreateTempDirectory("", c)
	defer func() {
		_ = os.RemoveAll(fakeSourceParentDir)
	}()

	fakeDestParentDir := fake.CreateTempDirectory("", c)
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
		"CopyFile(...)": {},
		"CopyFile(...): not existing source file": {
			notExistingSourceFileName: "not-existing",
			expectError:               true,
		},
		"CopyFile(...): not existing destination directory": {
			notExistingDestDirName: "should-create",
			doOverWrite:            true,
			expectedSameSize:       true,
		},
		"CopyFile(...): do overwrite": {
			doOverWrite:      true,
			expectedSameSize: true,
		},
		"CopyFile(...): sparse file": {
			doOverWrite:      true,
			sparseSize:       4097,
			expectedSameSize: false,
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing io.%v", testName)

		fakeSourceDir := fake.CreateTempDirectory(fakeSourceParentDir, c)
		fakeSourceFile := filepath.Join(fakeSourceDir, testCase.notExistingSourceFileName)
		if testCase.notExistingSourceFileName == "" {
			var fakeFile *os.File
			if testCase.sparseSize != 0 {
				fakeFile = fake.CreateTempSparseFile(fakeSourceDir, fmt.Sprintf("test-%v", time.Now().UnixNano()), testCase.sparseSize, "content", c)
			} else {
				fakeFile = fake.CreateTempFile(fakeSourceDir, fmt.Sprintf("test-%v", time.Now().UnixNano()), "content", c)
			}
			fakeSourceFile = fakeFile.Name()
			_ = fakeFile.Close()
		}

		fakeDestDir := filepath.Join(fakeDestParentDir, testCase.notExistingDestDirName)
		if testCase.notExistingDestDirName == "" {
			fakeDestDir = fake.CreateTempDirectory(fakeDestParentDir, c)
		}

		if !testCase.doOverWrite && testCase.notExistingDestDirName == "" {
			fake.CreateTempFile(fakeDestDir, filepath.Base(fakeSourceFile), "do-not-overwrite", c)
		}

		fakeDestPath := filepath.Join(fakeDestDir, filepath.Base(fakeSourceFile))
		err := CopyFile(fakeSourceFile, fakeDestPath, testCase.doOverWrite)
		if testCase.expectError {
			c.Assert(err, NotNil)
			continue
		}
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName, err))

		destFile := filepath.Join(fakeDestDir, filepath.Base(fakeSourceFile))
		content, err := os.ReadFile(destFile)
		c.Assert(err, IsNil)

		if !testCase.doOverWrite {
			c.Assert(string(content), Equals, "do-not-overwrite")
		} else {
			expectedContent := "content"
			if testCase.sparseSize != 0 {
				expectedContent = expectedContent + strings.Repeat("\x00", int(testCase.sparseSize)-len(expectedContent))
			}
			c.Assert(string(content), Equals, expectedContent)
			err := CheckIsFileSizeSame(destFile, fakeSourceFile)
			if testCase.expectedSameSize {
				c.Assert(err, IsNil)
			} else {
				c.Assert(err, NotNil)
			}
		}
	}
}

func (s *TestSuite) TestFindFiles(c *C) {
	fakeDir := fake.CreateTempDirectory("", c)
	defer func() {
		_ = os.RemoveAll(fakeDir)
	}()

	// Prepare sub directory
	fakeDirSub := fake.CreateTempDirectory(fakeDir, c)

	// Prepare 2 existing files in root of the fake directory,
	// and 2 existing file in sub directory.
	existingFileCount := 2
	existingFilePaths := make(map[string]bool, 3)
	existingFilePaths[fakeDir] = true
	existingFilePaths[fakeDirSub] = true
	for _, dir := range []string{fakeDir, fakeDirSub} {
		for i := 0; i < existingFileCount; i++ {
			file := fake.CreateTempFile(dir, fmt.Sprintf("test-%v", i), "content", c)
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
		"FindFiles(...)": {
			expectedFilePaths: []string{
				fakeDir,
				filepath.Join(fakeDir, "test-0"),
				filepath.Join(fakeDir, "test-1"),
				fakeDirSub,
				filepath.Join(fakeDirSub, "test-0"),
				filepath.Join(fakeDirSub, "test-1"),
			},
		},
		"FindFiles(...): find file with name": {
			findFileWithName: "test-0",
			expectedFilePaths: []string{
				filepath.Join(fakeDir, "test-0"),
				filepath.Join(fakeDirSub, "test-0"),
			},
		},
		"FindFiles(...): max depth": {
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
		c.Logf("testing io.%v", testName)

		result, err := FindFiles(fakeDir, testCase.findFileWithName, testCase.maxDepth)
		if testCase.expectError {
			c.Assert(err, NotNil)
			continue
		}
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName, err))
		c.Assert(len(result), Equals, len(testCase.expectedFilePaths), Commentf(test.ErrResultFmt, testName))
		for _, filePath := range result {
			c.Assert(existingFilePaths[filePath], Equals, true)
		}
	}
}

func (s *TestSuite) TestGetEmptyFiles(c *C) {
	fakeDir := fake.CreateTempDirectory("", c)
	defer func() {
		_ = os.RemoveAll(fakeDir)
	}()

	fakeSubDir := fake.CreateTempDirectory(fakeDir, c)

	fileWithContent := fake.CreateTempFile(fakeDir, "regular-file", "content", c)
	err := fileWithContent.Close()
	c.Assert(err, IsNil)

	fileWithoutContent := fake.CreateTempFile(fakeDir, "empty-file-0", "", c)
	err = fileWithoutContent.Close()
	c.Assert(err, IsNil)

	fileWithoutContentInSubDir := fake.CreateTempFile(fakeSubDir, "empty-file-1", "", c)
	defer func() {
		_ = fileWithoutContentInSubDir.Close()
	}()

	type testCase struct {
		directory      string
		expectedResult map[string]bool
		expectError    bool
	}
	testCases := map[string]testCase{
		"GetEmptyFiles(...)": {
			expectedResult: map[string]bool{
				fileWithoutContent.Name():         true,
				fileWithoutContentInSubDir.Name(): true,
			},
		},
		"GetEmptyFiles(...): not existing directory": {
			directory:   "not-existing-directory",
			expectError: true,
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing io.%v", testName)

		if testCase.directory == "" {
			testCase.directory = fakeDir
		}
		result, err := GetEmptyFiles(testCase.directory)
		if testCase.expectError {
			c.Assert(err, NotNil)
			continue
		}
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName, err))
		c.Assert(len(result), Equals, len(testCase.expectedResult), Commentf(test.ErrResultFmt, testName))
		for _, filePath := range result {
			c.Assert(testCase.expectedResult[filePath], Equals, true)
		}
	}
}

func (s *TestSuite) TestReadFileContent(c *C) {
	fakeDir := fake.CreateTempDirectory("", c)
	defer func() {
		_ = os.RemoveAll(fakeDir)
	}()

	fileContentFmt := "test: %v"

	type testCase struct {
		isFileExist bool

		expectError bool
	}
	testCases := map[string]testCase{
		"ReadFileContent(...)": {
			isFileExist: true,
		},
		"ReadFileContent(...): not existing file": {
			isFileExist: false,
			expectError: true,
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing io.%v", testName)

		expectedContent := fmt.Sprintf(fileContentFmt, testName)

		filePath := filepath.Join(fakeDir, "not-exist")
		if testCase.isFileExist {
			file := fake.CreateTempFile(fakeDir, "", expectedContent, c)
			filePath = file.Name()
			_ = file.Close()
		}

		content, err := ReadFileContent(filePath)
		if testCase.expectError {
			c.Assert(err, NotNil)
			continue
		}
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName, err))
		c.Assert(content, Equals, expectedContent)
	}
}

func (s *TestSuite) TestSyncFile(c *C) {
	fakeDir := fake.CreateTempDirectory("", c)
	defer func() {
		_ = os.RemoveAll(fakeDir)
	}()

	type testCase struct {
		isFileExist bool

		expectError bool
	}
	testCases := map[string]testCase{
		"SyncFile(...)": {
			isFileExist: true,
		},
		"SyncFile(...): not existing file": {
			isFileExist: false,
			expectError: true,
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing io.%v", testName)

		filePath := filepath.Join(fakeDir, "not-exist")
		if testCase.isFileExist {
			file := fake.CreateTempFile(fakeDir, "", "content", c)
			filePath = file.Name()
			_ = file.Close()
		}

		err := SyncFile(filePath)
		if testCase.expectError {
			c.Assert(err, NotNil)
			continue
		}
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName, err))
	}
}

func (s *TestSuite) TestGetDiskStat(c *C) {
	fakeDir := fake.CreateTempDirectory("", c)
	defer func() {
		_ = os.RemoveAll(fakeDir)
	}()

	type testCase struct {
		isPathExist bool
		expectError bool
	}
	testCases := map[string]testCase{
		"GetDiskStat(...)": {
			isPathExist: true,
		},
		"GetDiskStat(...): not existing path": {
			isPathExist: false,
			expectError: true,
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing io.%v", testName)

		testDir := fake.CreateTempDirectory(fakeDir, c)
		if !testCase.isPathExist {
			_ = os.RemoveAll(testDir)
		}

		diskStat, err := GetDiskStat(testDir)
		if testCase.expectError {
			c.Assert(err, NotNil)
			continue
		}
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName, err))

		expectedDiskStat, err := getDiskStat(testDir)
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName, err))

		// On the running system, FreeBlocks/StorageAvailable might be changing with time.
		// So we only compare the following fields
		c.Assert(diskStat.DiskID, Equals, expectedDiskStat.DiskID)
		c.Assert(diskStat.Path, Equals, expectedDiskStat.Path)

		// FIXME: overlayfs is not supported in the github.com/shirou/gopsutil/v3
		if expectedDiskStat.Type != "overlayfs" {
			c.Assert(diskStat.Type, Equals, expectedDiskStat.Type)
		}

		c.Assert(diskStat.TotalBlocks, Equals, expectedDiskStat.TotalBlocks)
		c.Assert(diskStat.BlockSize, Equals, expectedDiskStat.BlockSize)
		c.Assert(diskStat.StorageMaximum, Equals, expectedDiskStat.StorageMaximum)
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

func (s *TestSuite) TestListOpenFiles(c *C) {
	fakeDir := fake.CreateTempDirectory("", c)
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
			dir: fake.CreateTempDirectory(fakeDir, c),
		},
		close: _dirInfo{
			dir: fake.CreateTempDirectory(fakeDir, c),
		},
	}

	fakeDirs.open.files = append(fakeDirs.open.files, fake.CreateTempFile(fakeDirs.open.dir, "file1", "content", c))
	fakeDirs.open.files = append(fakeDirs.open.files, fake.CreateTempFile(fakeDirs.open.dir, "file2", "content", c))
	defer func() {
		for _, file := range fakeDirs.open.files {
			err := file.Close()
			c.Assert(err, IsNil)
		}
	}()

	// Create and close files in the close directory
	fakeDirs.close.files = append(fakeDirs.close.files, fake.CreateTempFile(fakeDirs.close.dir, "file1", "content", c))
	fakeDirs.close.files = append(fakeDirs.close.files, fake.CreateTempFile(fakeDirs.close.dir, "file2", "content", c))
	for _, file := range fakeDirs.close.files {
		err := file.Close()
		c.Assert(err, IsNil)
	}

	time.Sleep(100 * time.Millisecond)

	type testCase struct {
		directory         string
		expectedOpenFiles []string
		expectError       bool
	}
	testCases := map[string]testCase{
		"ListOpenFiles(...)": {
			directory: fakeDir,
			expectedOpenFiles: []string{
				filepath.Join(fakeDirs.open.dir, "file1"),
				filepath.Join(fakeDirs.open.dir, "file2"),
			},
		},
		"ListOpenFiles(...): not existing path": {
			directory:   "not-existing-path",
			expectError: true,
		},
		"ListOpenFiles(...): no open files": {
			directory:   fakeDirs.close.dir,
			expectError: false,
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing io.%v", testName)

		if testCase.directory == "" {
			testCase.directory = fakeDir
		}

		openFiles, err := ListOpenFiles("/proc", testCase.directory)
		if testCase.expectError {
			c.Assert(err, NotNil)
			continue
		}
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName, err))
		c.Assert(len(openFiles), Equals, len(testCase.expectedOpenFiles),
			Commentf(test.ErrResultFmt, fmt.Sprintf("%s: %v", testName, openFiles)),
		)
		for i, openFile := range openFiles {
			c.Assert(openFile, Equals, testCase.expectedOpenFiles[i])
		}
	}
}

func (s *TestSuite) TestIsDirectoryEmpty(c *C) {
	fakeDir := fake.CreateTempDirectory("", c)
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
		"IsDirectoryEmpty(...)": {
			isEmpty:      true,
			expectResult: true,
		},
		"IsDirectoryEmpty(...): not empty": {
			isEmpty:      false,
			expectResult: false,
		},
		"IsDirectoryEmpty(...): not existing path": {
			isNotExist:   true,
			expectError:  true,
			expectResult: false,
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing io.%v", testName)

		testDir := fake.CreateTempDirectory(fakeDir, c)

		if !testCase.isEmpty {
			fake.CreateTempFile(testDir, "file", "content", c)
		}

		if testCase.isNotExist {
			err := os.RemoveAll(testDir)
			c.Assert(err, IsNil)
		}

		result, err := IsDirectoryEmpty(testDir)
		if testCase.expectError {
			c.Assert(err, NotNil)
			continue
		}
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName, err))
		c.Assert(result, Equals, testCase.expectResult, Commentf(test.ErrResultFmt, testName))
	}
}

func (s *TestSuite) TestCheckIsFileSizeSame(c *C) {
	fakeDir := fake.CreateTempDirectory("", c)
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
		"CheckIsFileSizeSame(...)": {},
		"CheckIsFileSizeSame(...): not existing path": {
			notFileExist: true,
			expectError:  true,
		},
		"CheckIsFileSizeSame(...): different size": {
			isDifferent: true,
			expectError: true,
		},
		"CheckIsFileSizeSame(...): directory": {
			isDirectory: true,
			expectError: true,
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing io.%v", testName)

		testDir := fake.CreateTempDirectory(fakeDir, c)

		fileName1 := "file1"
		fileName2 := "file2"

		var file1 *os.File
		var file2 *os.File
		switch {
		case testCase.isDifferent:
			file1 = fake.CreateTempFile(testDir, fileName1, "content", c)
			file2 = fake.CreateTempFile(testDir, fileName2, "different-content", c)
		default:
			file1 = fake.CreateTempFile(testDir, fileName1, "content", c)
			file2 = fake.CreateTempFile(testDir, fileName2, "content", c)
		}

		if testCase.notFileExist {
			err := os.RemoveAll(testDir)
			c.Assert(err, IsNil)
		}

		var err error
		if testCase.isDirectory {
			err = CheckIsFileSizeSame(file1.Name(), file2.Name(), testDir)
		} else {
			err = CheckIsFileSizeSame(file1.Name(), file2.Name())
		}
		if testCase.expectError {
			c.Assert(err, NotNil)
			continue
		}
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName, err))
	}
}
