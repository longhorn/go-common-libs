package sys

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindBlockDeviceForMountWithFile(t *testing.T) {
	tests := []struct {
		name          string
		mountsContent string
		mountPath     string
		expected      string
		expectError   bool
	}{
		{
			name: "find device for longhorn mount",
			mountsContent: `/dev/sda1 / ext4 rw,relatime 0 0
/dev/sda2 /home ext4 rw,relatime 0 0
/dev/sdb1 /var/lib/longhorn ext4 rw,relatime 0 0`,
			mountPath: "/var/lib/longhorn",
			expected:  "/dev/sdb1",
		},
		{
			name: "mount path not found",
			mountsContent: `/dev/sda1 / ext4 rw,relatime 0 0
/dev/sda2 /home ext4 rw,relatime 0 0`,
			mountPath:   "/nonexistent",
			expectError: true,
		},
		{
			name: "device with UUID",
			mountsContent: `/dev/disk/by-uuid/12345678-1234-1234-1234-123456789012 /var/lib/longhorn ext4 rw,relatime 0 0
/dev/sda1 / ext4 rw,relatime 0 0`,
			mountPath: "/var/lib/longhorn",
			expected:  "/dev/disk/by-uuid/12345678-1234-1234-1234-123456789012",
		},
		{
			name: "handle multiple spaces",
			mountsContent: `/dev/sda1    /    ext4    rw,relatime    0    0
/dev/sda2    /home    ext4    rw,relatime    0    0`,
			mountPath: "/home",
			expected:  "/dev/sda2",
		},
		{
			name:          "empty mounts file",
			mountsContent: "",
			mountPath:     "/",
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary mounts file
			tmpDir := t.TempDir()
			mountsFile := filepath.Join(tmpDir, "mounts")
			err := os.WriteFile(mountsFile, []byte(tt.mountsContent), 0644)
			assert.NoError(t, err)

			device, err := findBlockDeviceForMountWithFile(tt.mountPath, mountsFile)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, device)
			}
		})
	}
}

func TestResolveMountPathToPhysicalDevice(t *testing.T) {
	tests := []struct {
		name           string
		mountsContent  string
		mountPath      string
		evalSymlinksFn func(string) (string, error)
		expected       string
		expectError    bool
		errorContains  string
	}{
		{
			name: "nvme partition to top-level controller",
			mountsContent: `/dev/disk/by-uuid/1234 /var/lib/longhorn ext4 rw,relatime 0 0
		/dev/sda1 / ext4 rw,relatime 0 0`,
			mountPath: "/var/lib/longhorn",
			evalSymlinksFn: func(s string) (string, error) {
				switch s {
				case "/dev/disk/by-uuid/1234":
					return "/dev/nvme0n1p2", nil
				case "/sys/class/block/nvme0n1p2":
					return "/sys/devices/pci0000:00/0000:00:01.0/nvme/nvme0/nvme0n1/nvme0n1p2", nil
				default:
					return s, nil
				}
			},
			expected: "/dev/nvme0",
		},
		{
			name: "nvme namespace without partition to controller",
			mountsContent: `/dev/nvme0n1 /var/lib/longhorn ext4 rw,relatime 0 0
		/dev/sda1 / ext4 rw,relatime 0 0`,
			mountPath: "/var/lib/longhorn",
			evalSymlinksFn: func(s string) (string, error) {
				switch s {
				case "/dev/nvme0n1":
					return "/dev/nvme0n1", nil
				case "/sys/class/block/nvme0n1":
					return "/sys/devices/pci0000:00/0000:00:02.0/nvme/nvme0/nvme0n1", nil
				default:
					return s, nil
				}
			},
			expected: "/dev/nvme0",
		},
		{
			name: "sda partition to base device",
			mountsContent: `/dev/sda2 /var/lib/longhorn ext4 rw,relatime 0 0
		/dev/sda1 / ext4 rw,relatime 0 0`,
			mountPath: "/var/lib/longhorn",
			evalSymlinksFn: func(s string) (string, error) {
				switch s {
				case "/dev/sda2":
					return "/dev/sda2", nil
				case "/sys/class/block/sda2":
					return "/sys/devices/pci0000:00/ahci/host0/target0:0:0/0:0:0:0/block/sda/sda2", nil
				default:
					return s, nil
				}
			},
			expected: "/dev/sda",
		},
		{
			name:          "eval symlinks error",
			mountsContent: `/dev/invalid_symlink /var/lib/longhorn ext4 rw,relatime 0 0`,
			mountPath:     "/var/lib/longhorn",
			evalSymlinksFn: func(s string) (string, error) {
				return "", assert.AnError
			},
			expectError:   true,
			errorContains: "failed to resolve symlink",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			mountsFile := filepath.Join(tmpDir, "mounts")
			err := os.WriteFile(mountsFile, []byte(tt.mountsContent), 0644)
			assert.NoError(t, err)

			device, err := resolveMountPathToPhysicalDeviceWithDeps(
				tt.mountPath, mountsFile, tt.evalSymlinksFn,
			)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, device)
		})
	}
}
