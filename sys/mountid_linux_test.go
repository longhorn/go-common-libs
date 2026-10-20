package sys

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/unix"
)

// rootMountID returns the ID of the mount on / reported by statx, or by fdinfo
// before Linux 5.8, checking that mountinfo lists that mount on /.
func rootMountID(t *testing.T) string {
	t.Helper()

	id, ok, err := statxMountID("/")
	assert.NoError(t, err)
	if !ok {
		id, err = fdinfoMountID("/")
		assert.NoError(t, err)
	}
	mounts, err := readMountInfo()
	assert.NoError(t, err)
	i := slices.IndexFunc(mounts, func(mount mountInfo) bool { return mount.id == id })
	if assert.GreaterOrEqual(t, i, 0) {
		assert.Equal(t, "/", mounts[i].mountPoint)
	}
	return id
}

func TestGetMountID(t *testing.T) {
	id, err := getMountID("/")
	assert.NoError(t, err)
	assert.Equal(t, rootMountID(t), id)

	_, err = getMountID("/does/not/exist")
	assert.Error(t, err)
}

func TestGetMountIDWithoutStatx(t *testing.T) {
	orig := statxMountIDFunc
	t.Cleanup(func() { statxMountIDFunc = orig })

	// As before Linux 5.8, or when statx is blocked by seccomp.
	statxMountIDFunc = func(string) (string, bool, error) { return "", false, nil }
	id, err := getMountID("/")
	assert.NoError(t, err)
	assert.Equal(t, rootMountID(t), id)

	statxMountIDFunc = func(string) (string, bool, error) { return "", false, assert.AnError }
	_, err = getMountID("/")
	assert.ErrorIs(t, err, assert.AnError)
}

func TestFdinfoMountID(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file")
	assert.NoError(t, os.WriteFile(file, nil, 0000)) // O_PATH needs no permission.
	// Neither O_PATH variant blocks on a FIFO without a writer.
	fifo := filepath.Join(dir, "fifo")
	assert.NoError(t, unix.Mkfifo(fifo, 0600))

	for _, path := range []string{"/", "/proc", dir, file, fifo} {
		id, err := fdinfoMountID(path)
		assert.NoError(t, err, path)

		statxID, ok, err := statxMountID(path)
		assert.NoError(t, err, path)
		if ok {
			assert.Equal(t, statxID, id, path)
		}
	}

	_, err := fdinfoMountID("/does/not/exist")
	assert.Error(t, err)
}

// forEachMountIDSource runs f with mount IDs from statx, and from fdinfo as
// before Linux 5.8.
func forEachMountIDSource(t *testing.T, f func(t *testing.T)) {
	t.Run("statx", f)
	t.Run("fdinfo", func(t *testing.T) {
		orig := statxMountIDFunc
		statxMountIDFunc = func(string) (string, bool, error) { return "", false, nil }
		t.Cleanup(func() { statxMountIDFunc = orig })
		f(t)
	})
}

func TestFindMountForPathResolvedByKernel(t *testing.T) {
	dir := t.TempDir()
	dirMount, err := findMountForPath(dir)
	assert.NoError(t, err)
	procSysMount, err := findMountForPath("/proc/sys")
	assert.NoError(t, err)
	assert.NotEqual(t, dirMount, procSysMount)

	assert.NoError(t, os.Symlink("/proc/sys/kernel", filepath.Join(dir, "link")))

	// An unlinked file is still reachable through its descriptor, whose
	// magic link reads as "<path> (deleted)".
	f, err := os.CreateTemp(dir, "unlinked")
	assert.NoError(t, err)
	defer f.Close() // nolint: errcheck
	assert.NoError(t, os.Remove(f.Name()))
	fdPath := fmt.Sprintf("/proc/self/fd/%d", f.Fd())

	forEachMountIDSource(t, func(t *testing.T) {
		mount, err := findMountForPath(fdPath)
		assert.NoError(t, err)
		assert.Equal(t, dirMount, mount)

		device, err := FindBlockDeviceForPath(fdPath)
		dirDevice, dirErr := FindBlockDeviceForPath(dir)
		assert.Equal(t, dirDevice, device)
		assert.Equal(t, dirErr == nil, err == nil, "%v, %v", err, dirErr)

		// ".." applies to the target of the symlink before it, i.e. this is
		// /proc/sys rather than dir.
		mount, err = findMountForPath(filepath.Join(dir, "link") + "/..")
		assert.NoError(t, err)
		assert.Equal(t, procSysMount, mount)

		t.Chdir(dir)
		mount, err = findMountForPath("link/..")
		assert.NoError(t, err)
		assert.Equal(t, procSysMount, mount)
	})
}

func TestFdinfoMountIDWhileThreadsExit(t *testing.T) {
	// Terminate OS threads continuously, as a goroutine exiting while locked to
	// its thread does, so that the goroutine reading fdinfo would sometimes
	// have its opening thread terminated if it could move to another thread.
	var stop atomic.Bool
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for !stop.Load() {
				done := make(chan struct{})
				go func() {
					runtime.LockOSThread()
					close(done)
				}()
				<-done
			}
		}()
	}
	defer func() {
		stop.Store(true)
		wg.Wait()
	}()

	for range 10000 {
		_, err := fdinfoMountID("/")
		if !assert.NoError(t, err) {
			break
		}
	}
}
