package backingimage

import (
	"testing"
	"time"

	"github.com/longhorn/go-common-libs/exec"
	"github.com/stretchr/testify/assert"
)

type mockExecutor string

func (m mockExecutor) Execute(envs []string, binary string, args []string, timeout time.Duration) (string, error) {
	return string(m), nil
}

func (m mockExecutor) ExecuteWithStdin(binary string, args []string, stdinString string, timeout time.Duration) (string, error) {
	return m.Execute([]string{}, binary, args, timeout)
}

func (m mockExecutor) ExecuteWithStdinPipe(binary string, args []string, stdinString string, timeout time.Duration) (string, error) {
	return m.Execute([]string{}, binary, args, timeout)
}

var _ exec.ExecuteInterface = (*mockExecutor)(nil)

func TestGetImageInfo(t *testing.T) {
	type testCase struct {
		qemuImgOutput string
		expectedInfo  ImageInfo
		expectedErr   bool
	}
	testCases := map[string]testCase{
		"qcow2": {
			qemuImgOutput: `
{
    "virtual-size": 21474836480,
    "filename": "SLE-Micro.x86_64-5.5.0-Default-qcow-GM.qcow2",
    "cluster-size": 65536,
    "format": "qcow2",
    "actual-size": 1001656320,
    "format-specific": {
        "type": "qcow2",
        "data": {
            "compat": "1.1",
            "compression-type": "zlib",
            "lazy-refcounts": false,
            "refcount-bits": 16,
            "corrupt": false,
            "extended-l2": false
        }
    },
    "dirty-flag": false
}
`,
			expectedInfo: ImageInfo{
				Format:      "qcow2",
				ActualSize:  1001656320,
				VirtualSize: 21474836480,
			},
			expectedErr: false,
		},
		"raw": {
			qemuImgOutput: `
{
    "virtual-size": 14548992000,
    "filename": "SLE-15-SP5-Full-x86_64-GM-Media1.iso",
    "format": "raw",
    "actual-size": 14548996096,
    "dirty-flag": false
}
`,
			expectedInfo: ImageInfo{
				Format:      "raw",
				ActualSize:  14548996096,
				VirtualSize: 14548992000,
			},
			expectedErr: false,
		},
		"invalid output": {
			qemuImgOutput: `invalid JSON`,
			expectedInfo:  ImageInfo{},
			expectedErr:   true,
		},
	}

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			executor := newQemuImgExecutor(mockExecutor(testCase.qemuImgOutput))
			info, err := executor.GetImageInfo("path/to/image")

			assert.Equal(t, testCase.expectedInfo, info)
			if testCase.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
