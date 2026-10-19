package sys

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/unix"
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

// mockSysfs replaces the sysfs root with a minimal tree laid out like the real
// one. devs maps "major:minor" to a device name; slaves maps a device name to
// its underlying devices; partitions maps a partition name to its parent
// device.
func mockSysfs(t *testing.T, devs map[string]string, slaves map[string][]string, partitions map[string]string) {
	t.Helper()

	dir := t.TempDir()
	deviceDir := func(name string) string {
		if parent, ok := partitions[name]; ok {
			return filepath.Join(dir, "devices", "block", parent, name)
		}
		return filepath.Join(dir, "devices", "block", name)
	}

	names := map[string]struct{}{}
	for _, name := range devs {
		names[name] = struct{}{}
	}
	for name, lowers := range slaves {
		names[name] = struct{}{}
		for _, lower := range lowers {
			names[lower] = struct{}{}
		}
	}
	for part, parent := range partitions {
		names[part] = struct{}{}
		names[parent] = struct{}{}
	}

	assert.NoError(t, os.MkdirAll(filepath.Join(dir, "class", "block"), 0755))
	assert.NoError(t, os.MkdirAll(filepath.Join(dir, "dev", "block"), 0755))
	for name := range names {
		assert.NoError(t, os.MkdirAll(deviceDir(name), 0755))
		assert.NoError(t, os.Symlink(deviceDir(name), filepath.Join(dir, "class", "block", name)))
	}
	for part := range partitions {
		assert.NoError(t, os.WriteFile(filepath.Join(deviceDir(part), "partition"), []byte("1\n"), 0644))
	}
	for devNum, name := range devs {
		assert.NoError(t, os.Symlink(deviceDir(name), filepath.Join(dir, "dev", "block", devNum)))
	}
	for name, lowers := range slaves {
		for _, lower := range lowers {
			assert.NoError(t, os.MkdirAll(filepath.Join(deviceDir(name), "slaves", lower), 0755))
		}
	}

	orig := sysfsDir
	sysfsDir = dir
	t.Cleanup(func() { sysfsDir = orig })
}

// mockStat replaces stat with one that reports the given device number for
// each path, and the given error for any other path.
func mockStat(t *testing.T, devByPath map[string][2]uint32, otherErr error) {
	t.Helper()

	orig := statDev
	statDev = func(path string) (uint64, error) {
		dev, ok := devByPath[path]
		if !ok {
			return 0, otherErr
		}
		return unix.Mkdev(dev[0], dev[1]), nil
	}
	t.Cleanup(func() { statDev = orig })
}

// stackedChain returns a slaves map stacking count device-mapper layers,
// dm-<first> on top of dm-<first+1> and so on, on top of bottom.
func stackedChain(first, count int, bottom string) map[string][]string {
	slaves := make(map[string][]string, count)
	for i := range count {
		lower := bottom
		if i < count-1 {
			lower = fmt.Sprintf("dm-%d", first+i+1)
		}
		slaves[fmt.Sprintf("dm-%d", first+i)] = []string{lower}
	}
	return slaves
}

func TestFindBlockDeviceForPath(t *testing.T) {
	slaves := map[string][]string{
		"dm-0": {"sda3"},         // LVM LV on a single PV
		"dm-1": {"sdb2", "sdc1"}, // LV spanning two PVs
		"dm-2": {"dm-0"},         // dm-crypt on top of LVM
		"dm-9": {"dm-9"},         // self-reference, must not loop forever
		"md0":  {"sdf1", "sdg1"}, // array spanning two disks, partitioned
		"dm-3": {"md0p1"},        // dm-crypt on a partition of md0
		"md1":  {"sdh1"},         // single-member array, partitioned
		"md9":  {"md9p1"},        // loop through a partition, must not loop forever
		"dm-4": {"cciss!c0d0p2"}, // dm-crypt on a device with "/" in its name
	}
	maps.Copy(slaves, stackedChain(100, maxStackedDeviceDepth, "sdd1"))
	maps.Copy(slaves, stackedChain(200, maxStackedDeviceDepth+1, "sde1"))

	mockSysfs(t,
		map[string]string{
			"8:1":     "sda1",
			"8:17":    "sdb1",
			"253:0":   "dm-0",
			"253:1":   "dm-1",
			"253:2":   "dm-2",
			"253:3":   "dm-3",
			"253:4":   "dm-4",
			"104:1":   "cciss!c0d0p1",
			"253:9":   "dm-9",
			"253:100": "dm-100",
			"253:200": "dm-200",
			"259:0":   "md0p1",
			"259:1":   "md1p1",
			"259:9":   "md9p1",
		},
		slaves,
		map[string]string{
			"sda1":  "sda",
			"sda3":  "sda",
			"sdb1":  "sdb",
			"sdb2":  "sdb",
			"sdc1":  "sdc",
			"sdd1":  "sdd",
			"sde1":  "sde",
			"sdf1":  "sdf",
			"sdg1":  "sdg",
			"sdh1":  "sdh",
			"md0p1": "md0",
			"md1p1": "md1",
			"md9p1": "md9",
			// sysfs encodes "/" in device names as "!".
			"cciss!c0d0p1": "cciss!c0d0",
			"cciss!c0d0p2": "cciss!c0d0",
		},
	)
	mockStat(t, map[string][2]uint32{
		"/var/lib/longhorn/":        {8, 1},
		"/mnt/disk1/longhorn":       {8, 17},
		"/longhorn":                 {253, 0},
		"/mnt/crypt/longhorn":       {253, 2},
		"/mnt/spanned/longhorn":     {253, 1},
		"/mnt/loop/longhorn":        {253, 9},
		"/mnt/maxdepth/longhorn":    {253, 100},
		"/mnt/toodeep/longhorn":     {253, 200},
		"/mnt/md0p1/longhorn":       {259, 0},
		"/mnt/crypt-md0p1/longhorn": {253, 3},
		"/mnt/md1p1/longhorn":       {259, 1},
		"/mnt/md9p1/longhorn":       {259, 9},
		"/mnt/cciss/longhorn":       {104, 1},
		"/mnt/crypt-cciss/longhorn": {253, 4},
		"/mnt/tmpfs/longhorn":       {0, 45},
	}, assert.AnError)

	tests := []struct {
		name          string
		path          string
		expected      string
		expectError   bool
		errorContains string
	}{
		{
			name:     "mount point",
			path:     "/var/lib/longhorn/",
			expected: "/dev/sda1",
		},
		{
			name:     "subdirectory of a mount point",
			path:     "/mnt/disk1/longhorn",
			expected: "/dev/sdb1",
		},
		{
			name:     "LVM logical volume on a single device",
			path:     "/longhorn",
			expected: "/dev/sda3",
		},
		{
			name:     "dm-crypt on top of LVM",
			path:     "/mnt/crypt/longhorn",
			expected: "/dev/sda3",
		},
		{
			name:          "device spanning multiple devices",
			path:          "/mnt/spanned/longhorn",
			expectError:   true,
			errorContains: "spans multiple devices",
		},
		{
			name:     "stacked devices at the maximum depth",
			path:     "/mnt/maxdepth/longhorn",
			expected: "/dev/sdd1",
		},
		{
			name:          "stacked devices one level beyond the maximum depth",
			path:          "/mnt/toodeep/longhorn",
			expectError:   true,
			errorContains: "stacked more than 8 levels deep",
		},
		{
			name:          "stacked devices too deep",
			path:          "/mnt/loop/longhorn",
			expectError:   true,
			errorContains: "stacked more than",
		},
		{
			name:          "partition of an array spanning multiple devices",
			path:          "/mnt/md0p1/longhorn",
			expectError:   true,
			errorContains: "is on device md0, which spans multiple devices [sdf1 sdg1]",
		},
		{
			name:          "dm-crypt on a partition of an array spanning multiple devices",
			path:          "/mnt/crypt-md0p1/longhorn",
			expectError:   true,
			errorContains: "is on device md0, which spans multiple devices [sdf1 sdg1]",
		},
		{
			name:     "partition of a single-member array",
			path:     "/mnt/md1p1/longhorn",
			expected: "/dev/sdh1",
		},
		{
			name:     "device name containing an escaped slash",
			path:     "/mnt/cciss/longhorn",
			expected: "/dev/cciss/c0d0p1",
		},
		{
			name:     "dm-crypt on a device name containing an escaped slash",
			path:     "/mnt/crypt-cciss/longhorn",
			expected: "/dev/cciss/c0d0p2",
		},
		{
			name:          "stacked devices looping through a partition",
			path:          "/mnt/md9p1/longhorn",
			expectError:   true,
			errorContains: "stacked more than 8 levels deep",
		},
		{
			name:          "filesystem without a block device",
			path:          "/mnt/tmpfs/longhorn",
			expectError:   true,
			errorContains: "is not backed by a block device (device 0:45)",
		},
		{
			name:          "stat failure",
			path:          "/does/not/exist",
			expectError:   true,
			errorContains: "failed to stat /does/not/exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			device, err := FindBlockDeviceForPath(tt.path)

			if tt.expectError {
				assert.ErrorContains(t, err, tt.errorContains)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, device)
		})
	}
}
