package sys

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
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

// mockStatBlockDev replaces the stat of mount sources with one that reports
// the given device number for each block device, an error for each path in
// notBlock, and that any other path does not exist.
func mockStatBlockDev(t *testing.T, devByPath map[string][2]uint32, notBlock ...string) {
	t.Helper()

	orig := statBlockDev
	statBlockDev = func(path string) (uint64, error) {
		if slices.Contains(notBlock, path) {
			return 0, fmt.Errorf("%s is not a block device", path)
		}
		dev, ok := devByPath[path]
		if !ok {
			return 0, &os.PathError{Op: "stat", Path: path, Err: unix.ENOENT}
		}
		return unix.Mkdev(dev[0], dev[1]), nil
	}
	t.Cleanup(func() { statBlockDev = orig })
}

// mockMountInfo replaces the mount table with one listing the given mounts,
// each being the device number, mount point, filesystem type and source. The
// mounts get the IDs 100, 101 and so on.
func mockMountInfo(t *testing.T, mounts [][4]string) {
	t.Helper()

	// Like the kernel, escape only spaces, tabs, newlines and backslashes.
	escape := strings.NewReplacer(`\`, `\134`, " ", `\040`, "\t", `\011`, "\n", `\012`).Replace
	var b strings.Builder
	for i, m := range mounts {
		// Alternate between lines with and without optional fields.
		optional := ""
		if i%2 == 0 {
			optional = fmt.Sprintf(" shared:%d master:1", i)
		}
		fmt.Fprintf(&b, "%d 1 %s / %s rw,relatime%s - %s %s rw\n", 100+i, m[0], escape(m[1]), optional, m[2], escape(m[3]))
	}
	file := filepath.Join(t.TempDir(), "mountinfo")
	assert.NoError(t, os.WriteFile(file, []byte(b.String()), 0644))

	orig := mountInfoPath
	mountInfoPath = file
	t.Cleanup(func() { mountInfoPath = orig })
}

// mockMountID replaces the mount ID lookup with the given one.
func mockMountID(t *testing.T, lookup func(path string) (string, error)) {
	t.Helper()

	orig := statMountID
	statMountID = lookup
	t.Cleanup(func() { statMountID = orig })
}

// mountIDOnMountTable returns a mount ID lookup simulating the kernel on the
// mocked mount table: a path is resolved like the kernel does, and is then on
// the last listed of the mounts with the longest mount point containing it,
// unless mountPoints gives the mount point of the mount it is on.
func mountIDOnMountTable(mountPoints map[string]string) func(string) (string, error) {
	return func(path string) (string, error) {
		if !filepath.IsAbs(path) {
			wd, err := os.Getwd()
			if err != nil {
				return "", err
			}
			path = wd + "/" + path
		}
		// Unlike filepath.Clean, this applies ".." after the symlinks before it.
		path, err := filepath.EvalSymlinks(path)
		if err != nil {
			return "", err
		}

		mounts, err := readMountInfo()
		if err != nil {
			return "", err
		}
		mountPoint, ok := mountPoints[path]
		id, longest := "", -1
		for _, mount := range mounts {
			if ok {
				if mount.mountPoint == mountPoint {
					id = mount.id
				}
				continue
			}
			contains := mount.mountPoint == "/" || path == mount.mountPoint || strings.HasPrefix(path, mount.mountPoint+"/")
			if contains && len(mount.mountPoint) >= longest {
				id, longest = mount.id, len(mount.mountPoint)
			}
		}
		if id == "" {
			return "", fmt.Errorf("no mount found for %s", path)
		}
		return id, nil
	}
}

// mockFilesystems replaces /proc/filesystems with a typical one.
func mockFilesystems(t *testing.T) {
	t.Helper()

	file := filepath.Join(t.TempDir(), "filesystems")
	content := "nodev\tsysfs\nnodev\ttmpfs\nnodev\tproc\nnodev\toverlay\nnodev\tfuse\nnodev\tzfs\n" +
		"\text4\n\txfs\n\tbtrfs\n\tfuseblk\n"
	assert.NoError(t, os.WriteFile(file, []byte(content), 0644))

	orig := filesystemsPath
	filesystemsPath = file
	t.Cleanup(func() { filesystemsPath = orig })
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
		"dm-5": {"sdm1"},         // dm-crypt under btrfs
	}
	maps.Copy(slaves, stackedChain(100, maxStackedDeviceDepth, "sdd1"))
	maps.Copy(slaves, stackedChain(200, maxStackedDeviceDepth+1, "sde1"))

	mockSysfs(t,
		map[string]string{
			"8:1":     "sda1",
			"8:2":     "sda2",
			"8:17":    "sdb1",
			"8:145":   "sdj1",
			"253:0":   "dm-0",
			"253:1":   "dm-1",
			"253:2":   "dm-2",
			"253:3":   "dm-3",
			"253:4":   "dm-4",
			"253:5":   "dm-5",
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
			"sda2":  "sda",
			"sda3":  "sda",
			"sdb1":  "sdb",
			"sdb2":  "sdb",
			"sdc1":  "sdc",
			"sdd1":  "sdd",
			"sde1":  "sde",
			"sdf1":  "sdf",
			"sdg1":  "sdg",
			"sdh1":  "sdh",
			"sdj1":  "sdj",
			"sdm1":  "sdm",
			"sdn1":  "sdn",
			"md0p1": "md0",
			"md1p1": "md1",
			"md9p1": "md9",
			// sysfs encodes "/" in device names as "!".
			"cciss!c0d0p1": "cciss!c0d0",
			"cciss!c0d0p2": "cciss!c0d0",
		},
	)

	// Paths must exist, as symlinks in them are resolved. The temporary
	// directory stands in for the root filesystem.
	root, err := filepath.EvalSymlinks(t.TempDir())
	assert.NoError(t, err)
	p := func(rel string) string { return filepath.Join(root, rel) }
	for _, dir := range []string{
		"longhorn", "mnt/disk1/longhorn", "mnt/disk10/longhorn", "mnt/my disk/longhorn", "mnt/over/longhorn",
		"mnt/lvm/longhorn", "mnt/crypt/longhorn", "mnt/spanned/longhorn", "mnt/loop/longhorn",
		"mnt/maxdepth/longhorn", "mnt/toodeep/longhorn", "mnt/md0p1/longhorn", "mnt/crypt-md0p1/longhorn",
		"mnt/md1p1/longhorn", "mnt/md9p1/longhorn", "mnt/cciss/longhorn", "mnt/crypt-cciss/longhorn",
		"mnt/btrfs/subvol/longhorn", "mnt/btrfs-subvol/longhorn", "mnt/btrfs-crypt/longhorn", "mnt/btrfs-gone/longhorn",
		"mnt/btrfs-nodev/longhorn", "mnt/btrfs-nodev-cciss/longhorn", "mnt/btrfs-nodev-gone/longhorn",
		"mnt/tmpfs/longhorn", "mnt/overlay/longhorn", "mnt/zfs/longhorn", "mnt/fuse/longhorn",
		"local/data", "mnt/disk1/child", "mnt/disk1/data",
		"mnt/hidden/sub/longhorn", "mnt/hidden/sub2/longhorn", "mnt/nbsp\u00a0disk/longhorn",
		"mnt/tmpfs-label/longhorn", "mnt/sshfs/longhorn", "mnt/unknownfs/longhorn", "mnt/btrfs-chardev/longhorn",
	} {
		assert.NoError(t, os.MkdirAll(p(dir), 0755))
	}
	assert.NoError(t, os.Symlink(p("mnt/disk1/longhorn"), p("link")))
	assert.NoError(t, os.Symlink(p("mnt/disk1/child"), p("local/link")))

	// The submount at mnt/hidden/sub is hidden by the later mount on its
	// parent directory, so the kernel reports the latter for paths below it.
	mockMountID(t, mountIDOnMountTable(map[string]string{p("mnt/hidden/sub/longhorn"): p("mnt/hidden")}))
	mockFilesystems(t)
	mockMountInfo(t, [][4]string{
		{"8:1", "/", "ext4", "/dev/root"},
		{"8:2", root, "ext4", "/dev/sda2"},
		{"8:17", p("mnt/disk1"), "ext4", "/dev/sdb1"},
		{"8:17", p("mnt/my disk"), "ext4", "/dev/sdb1"},
		{"8:2", p("mnt/over"), "ext4", "/dev/sda2"},
		{"8:17", p("mnt/over"), "ext4", "/dev/sdb1"}, // mounted on top of the previous one
		{"253:0", p("mnt/lvm"), "xfs", "/dev/mapper/vg0-lv0"},
		{"253:2", p("mnt/crypt"), "ext4", "/dev/mapper/crypt"},
		{"253:1", p("mnt/spanned"), "ext4", "/dev/mapper/vg1-lv0"},
		{"253:9", p("mnt/loop"), "ext4", "/dev/dm-9"},
		{"253:100", p("mnt/maxdepth"), "ext4", "/dev/dm-100"},
		{"253:200", p("mnt/toodeep"), "ext4", "/dev/dm-200"},
		{"259:0", p("mnt/md0p1"), "ext4", "/dev/md0p1"},
		{"253:3", p("mnt/crypt-md0p1"), "ext4", "/dev/mapper/crypt-md0p1"},
		{"259:1", p("mnt/md1p1"), "ext4", "/dev/md1p1"},
		{"259:9", p("mnt/md9p1"), "ext4", "/dev/md9p1"},
		{"104:1", p("mnt/cciss"), "ext4", "/dev/cciss/c0d0p1"},
		{"253:4", p("mnt/crypt-cciss"), "ext4", "/dev/mapper/crypt-cciss"},
		// btrfs uses anonymous device numbers, one per subvolume.
		{"0:50", p("mnt/btrfs"), "btrfs", "/dev/sdj1"},
		{"0:51", p("mnt/btrfs-subvol"), "btrfs", "/dev/sdj1"},
		{"0:53", p("mnt/btrfs-crypt"), "btrfs", "/dev/mapper/btrfs-crypt"},
		{"0:54", p("mnt/btrfs-gone"), "btrfs", "/dev/sdz1"},
		// Sources missing from /dev, e.g. in an unprivileged container.
		{"0:55", p("mnt/btrfs-nodev"), "btrfs", "/dev/sdn1"},
		{"0:56", p("mnt/btrfs-nodev-cciss"), "btrfs", "/dev/cciss/c0d0p1"},
		{"0:57", p("mnt/btrfs-nodev-gone"), "btrfs", "/dev/sdy1"},
		{"0:45", p("mnt/tmpfs"), "tmpfs", "tmpfs"},
		{"0:46", p("mnt/overlay"), "overlay", "overlay"},
		{"0:47", p("mnt/zfs"), "zfs", "tank/longhorn"},
		{"0:48", p("mnt/fuse"), "fuse", "/dev/fuse"},
		// A submount hidden by a mount on its parent directory, and one
		// mounted on top of the latter.
		{"8:17", p("mnt/hidden/sub"), "ext4", "/dev/sdb1"},
		{"104:1", p("mnt/hidden"), "ext4", "/dev/cciss/c0d0p1"},
		{"8:17", p("mnt/hidden/sub2"), "ext4", "/dev/sdb1"},
		// Whitespace other than ASCII spaces is not escaped.
		{"8:17", p("mnt/nbsp\u00a0disk"), "ext4", "/dev/sdb1"},
		// Sources of filesystems not needing a block device are just labels.
		{"0:58", p("mnt/tmpfs-label"), "tmpfs", "/dev/sdj1"},
		{"0:59", p("mnt/sshfs"), "fuse.sshfs", "/dev/sdj1"},
		{"0:60", p("mnt/unknownfs"), "unknownfs", "/dev/sdj1"},
		{"0:61", p("mnt/btrfs-chardev"), "btrfs", "/dev/console"},
	})
	mockStatBlockDev(t, map[string][2]uint32{
		"/dev/sdj1":               {8, 145},
		"/dev/mapper/btrfs-crypt": {253, 5},
		"/dev/sdz1":               {65, 144}, // not in sysfs
	}, "/dev/console")

	tests := []struct {
		name          string
		path          string
		expected      string
		expectError   bool
		errorContains string
	}{
		{
			name:     "mount point",
			path:     p("mnt/disk1"),
			expected: "/dev/sdb1",
		},
		{
			name:     "subdirectory of a mount point",
			path:     p("mnt/disk1/longhorn"),
			expected: "/dev/sdb1",
		},
		{
			name:     "directory on the root filesystem",
			path:     p("longhorn"),
			expected: "/dev/sda2",
		},
		{
			name:     "mount point that is a string prefix of the path but not a parent",
			path:     p("mnt/disk10/longhorn"),
			expected: "/dev/sda2",
		},
		{
			name:     "symlink into a mount",
			path:     p("link"),
			expected: "/dev/sdb1",
		},
		{
			name:     "mount point with an escaped space",
			path:     p("mnt/my disk/longhorn"),
			expected: "/dev/sdb1",
		},
		{
			name:     "topmost of several mounts on the same mount point",
			path:     p("mnt/over/longhorn"),
			expected: "/dev/sdb1",
		},
		{
			name:     "LVM logical volume on a single device",
			path:     p("mnt/lvm/longhorn"),
			expected: "/dev/sda3",
		},
		{
			name:     "dm-crypt on top of LVM",
			path:     p("mnt/crypt/longhorn"),
			expected: "/dev/sda3",
		},
		{
			name:          "device spanning multiple devices",
			path:          p("mnt/spanned/longhorn"),
			expectError:   true,
			errorContains: "spans multiple devices",
		},
		{
			name:     "stacked devices at the maximum depth",
			path:     p("mnt/maxdepth/longhorn"),
			expected: "/dev/sdd1",
		},
		{
			name:          "stacked devices one level beyond the maximum depth",
			path:          p("mnt/toodeep/longhorn"),
			expectError:   true,
			errorContains: "stacked more than 8 levels deep",
		},
		{
			name:          "stacked devices too deep",
			path:          p("mnt/loop/longhorn"),
			expectError:   true,
			errorContains: "stacked more than",
		},
		{
			name:          "partition of an array spanning multiple devices",
			path:          p("mnt/md0p1/longhorn"),
			expectError:   true,
			errorContains: "is on device md0, which spans multiple devices [sdf1 sdg1]",
		},
		{
			name:          "dm-crypt on a partition of an array spanning multiple devices",
			path:          p("mnt/crypt-md0p1/longhorn"),
			expectError:   true,
			errorContains: "is on device md0, which spans multiple devices [sdf1 sdg1]",
		},
		{
			name:     "partition of a single-member array",
			path:     p("mnt/md1p1/longhorn"),
			expected: "/dev/sdh1",
		},
		{
			name:     "device name containing an escaped slash",
			path:     p("mnt/cciss/longhorn"),
			expected: "/dev/cciss/c0d0p1",
		},
		{
			name:     "dm-crypt on a device name containing an escaped slash",
			path:     p("mnt/crypt-cciss/longhorn"),
			expected: "/dev/cciss/c0d0p2",
		},
		{
			name:          "stacked devices looping through a partition",
			path:          p("mnt/md9p1/longhorn"),
			expectError:   true,
			errorContains: "stacked more than 8 levels deep",
		},
		{
			name:     "btrfs mount point",
			path:     p("mnt/btrfs"),
			expected: "/dev/sdj1",
		},
		{
			name:     "btrfs subvolume within a mount",
			path:     p("mnt/btrfs/subvol/longhorn"),
			expected: "/dev/sdj1",
		},
		{
			name:     "mounted btrfs subvolume",
			path:     p("mnt/btrfs-subvol/longhorn"),
			expected: "/dev/sdj1",
		},
		{
			name:     "btrfs on dm-crypt",
			path:     p("mnt/btrfs-crypt/longhorn"),
			expected: "/dev/sdm1",
		},
		{
			name:          "btrfs on a device missing from sysfs",
			path:          p("mnt/btrfs-gone/longhorn"),
			expectError:   true,
			errorContains: "block device /dev/sdz1 (65:144) not found in sysfs",
		},
		{
			name:     "btrfs on a device missing from /dev",
			path:     p("mnt/btrfs-nodev/longhorn"),
			expected: "/dev/sdn1",
		},
		{
			name:     "btrfs on a device with \"/\" in its name missing from /dev",
			path:     p("mnt/btrfs-nodev-cciss/longhorn"),
			expected: "/dev/cciss/c0d0p1",
		},
		{
			name:          "btrfs on a device missing from both /dev and sysfs",
			path:          p("mnt/btrfs-nodev-gone/longhorn"),
			expectError:   true,
			errorContains: "failed to stat mount source /dev/sdy1",
		},
		{
			name:          "tmpfs",
			path:          p("mnt/tmpfs/longhorn"),
			expectError:   true,
			errorContains: `is on tmpfs filesystem "tmpfs" mounted at ` + p("mnt/tmpfs") + ", which is not backed by a block device",
		},
		{
			name:          "overlayfs",
			path:          p("mnt/overlay/longhorn"),
			expectError:   true,
			errorContains: "which is not backed by a block device",
		},
		{
			name:          "ZFS dataset",
			path:          p("mnt/zfs/longhorn"),
			expectError:   true,
			errorContains: `is on zfs filesystem "tank/longhorn"`,
		},
		{
			name:          "FUSE filesystem with a device-like source",
			path:          p("mnt/fuse/longhorn"),
			expectError:   true,
			errorContains: `is on fuse filesystem "/dev/fuse" mounted at ` + p("mnt/fuse") + ", which is not backed by a block device",
		},
		{
			name:          "tmpfs with a block device as its source label",
			path:          p("mnt/tmpfs-label/longhorn"),
			expectError:   true,
			errorContains: `is on tmpfs filesystem "/dev/sdj1" mounted at ` + p("mnt/tmpfs-label") + ", which is not backed by a block device",
		},
		{
			name:          "FUSE filesystem with a subtype and a block device as its source label",
			path:          p("mnt/sshfs/longhorn"),
			expectError:   true,
			errorContains: `is on fuse.sshfs filesystem "/dev/sdj1"`,
		},
		{
			name:          "filesystem type missing from /proc/filesystems",
			path:          p("mnt/unknownfs/longhorn"),
			expectError:   true,
			errorContains: `is on unknownfs filesystem "/dev/sdj1"`,
		},
		{
			name:          "mount source that is not a block device",
			path:          p("mnt/btrfs-chardev/longhorn"),
			expectError:   true,
			errorContains: "failed to stat mount source /dev/console",
		},
		{
			name:     "symlink followed by ..",
			path:     root + "/local/link/../data",
			expected: "/dev/sdb1",
		},
		{
			name:     "submount hidden by a mount on its parent directory",
			path:     p("mnt/hidden/sub/longhorn"),
			expected: "/dev/cciss/c0d0p1",
		},
		{
			name:     "submount mounted after the mount on its parent directory",
			path:     p("mnt/hidden/sub2/longhorn"),
			expected: "/dev/sdb1",
		},
		{
			name:     "mount point with an unescaped non-breaking space",
			path:     p("mnt/nbsp\u00a0disk/longhorn"),
			expected: "/dev/sdb1",
		},
		{
			name:          "path that does not exist",
			path:          p("does/not/exist"),
			expectError:   true,
			errorContains: "failed to get mount ID of " + p("does/not/exist"),
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

	t.Run("relative path with a symlink followed by ..", func(t *testing.T) {
		t.Chdir(p("local"))

		device, err := FindBlockDeviceForPath("link/../data")
		assert.NoError(t, err)
		assert.Equal(t, "/dev/sdb1", device)
	})
}

func TestFindBlockDeviceForPathByMountID(t *testing.T) {
	mockSysfs(t, map[string]string{"8:1": "sda1", "8:17": "sdb1", "8:33": "sdc1"}, nil,
		map[string]string{"sda1": "sda", "sdb1": "sdb", "sdc1": "sdc"})
	root, err := filepath.EvalSymlinks(t.TempDir())
	assert.NoError(t, err)
	path := filepath.Join(root, "x", "sub", "longhorn")
	assert.NoError(t, os.MkdirAll(path, 0755))

	// x/sub is listed before x, as when it was moved under x after x was
	// mounted, so the listing order does not tell that it is visible.
	mockMountInfo(t, [][4]string{
		{"8:1", "/", "ext4", "/dev/sda1"},                              // ID 100
		{"8:17", filepath.Join(root, "x", "sub"), "ext4", "/dev/sdb1"}, // ID 101
		{"8:33", filepath.Join(root, "x"), "ext4", "/dev/sdc1"},        // ID 102
	})

	tests := []struct {
		name          string
		id            string
		err           error
		expected      string
		errorContains string
	}{
		{
			name:     "moved mount listed before the mount on its parent directory",
			id:       "101",
			expected: "/dev/sdb1",
		},
		{
			name:     "mount on the parent directory",
			id:       "102",
			expected: "/dev/sdc1",
		},
		{
			name:          "mount ID missing from mountinfo",
			id:            "999",
			errorContains: "mount 999 of " + path + " not found in",
		},
		{
			name:          "mount ID lookup failure",
			err:           assert.AnError,
			errorContains: "failed to get mount ID of " + path,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMountID(t, func(p string) (string, error) {
				assert.Equal(t, path, p)
				return tt.id, tt.err
			})

			device, err := FindBlockDeviceForPath(path)
			if tt.errorContains != "" {
				assert.ErrorContains(t, err, tt.errorContains)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, device)
		})
	}
}

func TestFindBlockDeviceForPathMountInfoErrors(t *testing.T) {
	mockSysfs(t, map[string]string{"8:1": "sda1"}, nil, nil)
	mockMountID(t, func(string) (string, error) { return "100", nil })
	path := t.TempDir()

	t.Run("missing mountinfo", func(t *testing.T) {
		orig := mountInfoPath
		mountInfoPath = filepath.Join(t.TempDir(), "missing")
		t.Cleanup(func() { mountInfoPath = orig })

		_, err := FindBlockDeviceForPath(path)
		assert.ErrorContains(t, err, "failed to read")
	})

	t.Run("malformed mountinfo", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "mountinfo")
		assert.NoError(t, os.WriteFile(file, []byte("1 0 8:1 / / rw shared:1 ext4 /dev/sda1 rw\n"), 0644))
		orig := mountInfoPath
		mountInfoPath = file
		t.Cleanup(func() { mountInfoPath = orig })

		_, err := FindBlockDeviceForPath(path)
		assert.ErrorContains(t, err, "invalid mountinfo line")
	})
}

func TestFindBlockDeviceForPathMountTableReading(t *testing.T) {
	mockSysfs(t, map[string]string{"8:1": "sda1", "8:17": "sdb1"}, nil,
		map[string]string{"sda1": "sda", "sdb1": "sdb"})
	path := t.TempDir()

	t.Run("mount triggered by resolving the path", func(t *testing.T) {
		mockMountInfo(t, [][4]string{{"8:1", "/", "ext4", "/dev/sda1"}})
		// As when the path is under an inactive automount.
		mockMountID(t, func(string) (string, error) {
			f, err := os.OpenFile(mountInfoPath, os.O_APPEND|os.O_WRONLY, 0)
			if err != nil {
				return "", err
			}
			defer f.Close() // nolint: errcheck
			_, err = fmt.Fprintf(f, "200 100 8:17 / %s rw - ext4 /dev/sdb1 rw\n", path)
			return "200", err
		})

		device, err := FindBlockDeviceForPath(path)
		assert.NoError(t, err)
		assert.Equal(t, "/dev/sdb1", device)
	})

	t.Run("line longer than 64 KiB", func(t *testing.T) {
		lowerDirs := strings.Repeat("/var/lib/containers/storage/overlay/l/ABCDEFGHIJKLMNOPQRSTUVWXYZ:", 1600)
		content := "100 1 8:1 / / rw - ext4 /dev/sda1 rw\n" +
			"101 100 0:50 / /merged rw - overlay overlay rw,lowerdir=" + lowerDirs + "\n" +
			fmt.Sprintf("102 100 8:17 / %s rw - ext4 /dev/sdb1 rw\n", path)
		assert.Greater(t, len(content), 100*1024)
		file := filepath.Join(t.TempDir(), "mountinfo")
		assert.NoError(t, os.WriteFile(file, []byte(content), 0644))
		orig := mountInfoPath
		mountInfoPath = file
		t.Cleanup(func() { mountInfoPath = orig })
		mockMountID(t, func(string) (string, error) { return "102", nil })

		device, err := FindBlockDeviceForPath(path)
		assert.NoError(t, err)
		assert.Equal(t, "/dev/sdb1", device)
	})

	// mockReads makes reading the mount table return the given tables in
	// turn, repeating the last one, and returns the number of reads.
	mockReads := func(t *testing.T, tables ...[]mountInfo) *int {
		reads := 0
		orig := readMounts
		readMounts = func() ([]mountInfo, error) {
			table := tables[min(reads, len(tables)-1)]
			reads++
			return table, nil
		}
		t.Cleanup(func() { readMounts = orig })
		return &reads
	}
	root := mountInfo{id: "100", devNum: "8:1", mountPoint: "/", fsType: "ext4", source: "/dev/sda1"}
	disk := mountInfo{id: "200", devNum: "8:17", mountPoint: path, fsType: "ext4", source: "/dev/sdb1"}

	t.Run("mount skipped by a read of a changing mount table", func(t *testing.T) {
		mockMountID(t, func(string) (string, error) { return "200", nil })
		reads := mockReads(t, []mountInfo{root}, []mountInfo{root, disk})

		device, err := FindBlockDeviceForPath(path)
		assert.NoError(t, err)
		assert.Equal(t, "/dev/sdb1", device)
		assert.Equal(t, 2, *reads)
	})

	t.Run("mount missing from every read", func(t *testing.T) {
		mockMountID(t, func(string) (string, error) { return "200", nil })
		reads := mockReads(t, []mountInfo{root})

		_, err := FindBlockDeviceForPath(path)
		assert.ErrorContains(t, err, "mount 200 of "+path+" not found in")
		assert.Equal(t, mountInfoReadAttempts, *reads)
	})
}

func TestParseMountInfoLine(t *testing.T) {
	tests := map[string]struct {
		line     string
		expected mountInfo
	}{
		"optional fields": {
			line:     "36 35 98:0 /mnt1 /mnt2 rw,noatime master:1 - ext3 /dev/root rw,errors=continue",
			expected: mountInfo{id: "36", devNum: "98:0", mountPoint: "/mnt2", fsType: "ext3", source: "/dev/root"},
		},
		"no optional fields": {
			line:     "25 1 8:1 / / rw,relatime - ext4 /dev/sda1 rw",
			expected: mountInfo{id: "25", devNum: "8:1", mountPoint: "/", fsType: "ext4", source: "/dev/sda1"},
		},
		"btrfs subvolume": {
			line:     "95 25 0:52 /@longhorn /var/lib/longhorn rw,relatime shared:50 - btrfs /dev/sdb1 rw,space_cache=v2,subvol=/@longhorn",
			expected: mountInfo{id: "95", devNum: "0:52", mountPoint: "/var/lib/longhorn", fsType: "btrfs", source: "/dev/sdb1"},
		},
		"escaped characters": {
			line:     `96 25 8:17 / /mnt/my\040disk\011x\134y rw - ext4 /dev/disk/by-label/my\040disk rw`,
			expected: mountInfo{id: "96", devNum: "8:17", mountPoint: "/mnt/my disk\tx\\y", fsType: "ext4", source: "/dev/disk/by-label/my disk"},
		},
		"unescaped Unicode whitespace": {
			line:     "98 25 8:17 / /mnt/my\u00a0disk\u2003x rw - ext4 /dev/sdb1 rw",
			expected: mountInfo{id: "98", devNum: "8:17", mountPoint: "/mnt/my\u00a0disk\u2003x", fsType: "ext4", source: "/dev/sdb1"},
		},
		"invalid escape kept as is": {
			line:     `97 25 8:17 / /mnt/a\9b rw - ext4 /dev/sdb1 rw`,
			expected: mountInfo{id: "97", devNum: "8:17", mountPoint: `/mnt/a\9b`, fsType: "ext4", source: "/dev/sdb1"},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mount, err := parseMountInfoLine(tt.line)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, mount)
		})
	}

	for _, line := range []string{
		"",
		"36 35 98:0 /mnt1 /mnt2 rw,noatime master:1 ext3 /dev/root rw",
		"36 35 98:0 /mnt1 /mnt2 rw,noatime master:1 shared:2 - ext3",
	} {
		_, err := parseMountInfoLine(line)
		assert.ErrorContains(t, err, "invalid mountinfo line", line)
	}
}
