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
		resolveDevice func(string) (string, error)
		expected      string
		expectError   bool
		errorContains string
	}{
		{
			name: "find device for longhorn mount",
			mountsContent: `/dev/sda1 / ext4 rw,relatime 0 0
/dev/sda2 /home ext4 rw,relatime 0 0
/dev/sdb1 /var/lib/longhorn ext4 rw,relatime 0 0`,
			mountPath: "/var/lib/longhorn",
			resolveDevice: func(device string) (string, error) {
				return device, nil // Mock: return device as-is
			},
			expected: "/dev/sdb1",
		},
		{
			name: "mount path not found",
			mountsContent: `/dev/sda1 / ext4 rw,relatime 0 0
/dev/sda2 /home ext4 rw,relatime 0 0`,
			mountPath: "/nonexistent",
			resolveDevice: func(device string) (string, error) {
				return device, nil
			},
			expectError: true,
		},
		{
			name: "device with UUID resolved to actual device",
			mountsContent: `/dev/disk/by-uuid/12345678-1234-1234-1234-123456789012 /var/lib/longhorn ext4 rw,relatime 0 0
/dev/sda1 / ext4 rw,relatime 0 0`,
			mountPath: "/var/lib/longhorn",
			resolveDevice: func(device string) (string, error) {
				if device == "/dev/disk/by-uuid/12345678-1234-1234-1234-123456789012" {
					return "/dev/sda2", nil // Mock: resolve UUID to actual device
				}
				return device, nil
			},
			expected: "/dev/sda2",
		},
		{
			name: "handle multiple spaces",
			mountsContent: `/dev/sda1    /    ext4    rw,relatime    0    0
/dev/sda2    /home    ext4    rw,relatime    0    0`,
			mountPath: "/home",
			resolveDevice: func(device string) (string, error) {
				return device, nil
			},
			expected: "/dev/sda2",
		},
		{
			name:          "empty mounts file",
			mountsContent: "",
			mountPath:     "/",
			resolveDevice: func(device string) (string, error) {
				return device, nil
			},
			expectError: true,
		},
		{
			name: "LVM device mapper bypass",
			mountsContent: `/dev/mapper/vg0-lv0 /var/lib/longhorn ext4 rw,relatime 0 0
/dev/sda1 / ext4 rw,relatime 0 0`,
			mountPath: "/var/lib/longhorn",
			resolveDevice: func(device string) (string, error) {
				t.Errorf("resolveDevice should not be called for /dev/mapper/ devices")
				return "", assert.AnError
			},
			expected: "/dev/mapper/vg0-lv0",
		},
		{
			name:          "device resolution error",
			mountsContent: `/dev/sda1 /var/lib/longhorn ext4 rw,relatime 0 0`,
			mountPath:     "/var/lib/longhorn",
			resolveDevice: func(device string) (string, error) {
				return "", assert.AnError
			},
			expectError:   true,
			errorContains: "failed to resolve",
		},
		{
			name: "reject pseudo-filesystem mount (tmpfs)",
			mountsContent: `tmpfs /tmp tmpfs rw,nosuid,nodev 0 0
/dev/sda1 / ext4 rw,relatime 0 0`,
			mountPath: "/tmp",
			resolveDevice: func(device string) (string, error) {
				return device, nil
			},
			expectError:   true,
			errorContains: "uses non-block device",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary mounts file
			tmpDir := t.TempDir()
			mountsFile := filepath.Join(tmpDir, "mounts")
			err := os.WriteFile(mountsFile, []byte(tt.mountsContent), 0644)
			assert.NoError(t, err)

			device, err := findBlockDeviceForMountWithDeps(tt.mountPath, mountsFile, tt.resolveDevice)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, device)
			}
		})
	}
}

func TestFindBlockDeviceByMajorMinor(t *testing.T) {
	tests := []struct {
		name          string
		devicePathFn  func(*testing.T) string
		expected      string
		expectError   bool
		errorContains string
		skipIfEmpty   bool
	}{
		{
			name: "resolve real block device",
			devicePathFn: func(t *testing.T) string {
				entries, err := os.ReadDir("/sys/class/block")
				if err != nil {
					t.Skipf("cannot read /sys/class/block: %v", err)
				}

				for _, entry := range entries {
					candidate := filepath.Join("/dev", entry.Name())
					info, statErr := os.Stat(candidate)
					if statErr != nil {
						continue
					}
					mode := info.Mode()
					if mode&os.ModeDevice == 0 || mode&os.ModeCharDevice != 0 {
						continue // not a block device
					}
					return candidate
				}

				return ""
			},
			skipIfEmpty: true,
		},
		{
			name: "symlinked device triggers full scan",
			devicePathFn: func(t *testing.T) string {
				entries, err := os.ReadDir("/sys/class/block")
				if err != nil {
					t.Skipf("cannot read /sys/class/block: %v", err)
				}

				var target string
				for _, entry := range entries {
					candidate := filepath.Join("/dev", entry.Name())
					info, statErr := os.Stat(candidate)
					if statErr != nil {
						continue
					}
					mode := info.Mode()
					if mode&os.ModeDevice == 0 || mode&os.ModeCharDevice != 0 {
						continue // not a block device
					}
					target = candidate
					break
				}

				if target == "" {
					return ""
				}

				link := filepath.Join(t.TempDir(), "devlink")
				if err := os.Symlink(target, link); err != nil {
					t.Skipf("cannot create symlink to %s: %v", target, err)
				}
				// Expect the resolver to return the canonical /dev/<name> (not the symlink path)
				// because it matches by major:minor and maps back to the entry name.
				return link
			},
			// We'll assert the resolved path equals the real target when checking results
			// (set in the test body below when expected is empty).
			skipIfEmpty: true,
		},
		{
			name: "non-existent device returns error",
			devicePathFn: func(*testing.T) string {
				return "/dev/this_device_should_not_exist"
			},
			expectError:   true,
			errorContains: "failed to stat device",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			devicePath := tt.devicePathFn(t)
			if devicePath == "" && tt.skipIfEmpty {
				t.Skip("no block devices available under /dev matching /sys/class/block entries")
			}

			resolved, err := findBlockDeviceByMajorMinor(devicePath)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				return
			}

			assert.NoError(t, err)
			if tt.expected != "" {
				assert.Equal(t, tt.expected, resolved)
			} else {
				// If the path is a symlink, we expect canonical /dev/<name>
				fi, _ := os.Lstat(devicePath)
				if fi.Mode()&os.ModeSymlink != 0 {
					// Resolve the symlink target name for comparison
					target, err := filepath.EvalSymlinks(devicePath)
					assert.NoError(t, err)
					assert.Equal(t, target, resolved)
				} else {
					assert.Equal(t, devicePath, resolved)
				}
			}
		})
	}
}

func TestResolveBlockDeviceToPhysicalDevice(t *testing.T) {
	tests := []struct {
		name           string
		blockDevice    string
		evalSymlinksFn func(string) (string, error)
		expected       string
		expectError    bool
		errorContains  string
	}{
		{
			name:        "nvme partition to top-level controller",
			blockDevice: "/dev/nvme0n1p2",
			evalSymlinksFn: func(s string) (string, error) {
				switch s {
				case "/dev/nvme0n1p2":
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
			name:        "nvme namespace without partition to controller",
			blockDevice: "/dev/nvme0n1",
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
			name:        "sda partition to base device",
			blockDevice: "/dev/sda2",
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
			name:        "sda base device",
			blockDevice: "/dev/sda",
			evalSymlinksFn: func(s string) (string, error) {
				switch s {
				case "/dev/sda":
					return "/dev/sda", nil
				case "/sys/class/block/sda":
					return "/sys/devices/pci0000:00/ahci/host0/target0:0:0/0:0:0:0/block/sda", nil
				default:
					return s, nil
				}
			},
			expected: "/dev/sda",
		},
		{
			name:        "eval symlinks error on device",
			blockDevice: "/dev/invalid_symlink",
			evalSymlinksFn: func(s string) (string, error) {
				if s == "/dev/invalid_symlink" {
					return "", assert.AnError
				}
				return s, nil
			},
			expectError:   true,
			errorContains: "failed to resolve symlink",
		},
		{
			name:        "eval symlinks error on sysfs",
			blockDevice: "/dev/sda1",
			evalSymlinksFn: func(s string) (string, error) {
				switch s {
				case "/dev/sda1":
					return "/dev/sda1", nil
				case "/sys/class/block/sda1":
					return "", assert.AnError
				default:
					return s, nil
				}
			},
			expectError:   true,
			errorContains: "failed to resolve sysfs path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			device, err := resolveBlockDeviceToPhysicalDeviceWithDeps(
				tt.blockDevice, tt.evalSymlinksFn,
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
