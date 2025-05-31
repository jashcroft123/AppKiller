package appmutex

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/gofrs/flock"
)

var lock *flock.Flock

// CreateMutex tries to acquire a file lock to prevent multiple instances.
func CreateMutex(name string) error {
	lockFile := getLockFilePath(name)
	lock = flock.New(lockFile)

	locked, err := lock.TryLock()
	if err != nil {
		return fmt.Errorf("could not acquire lock: %w", err)
	}
	if !locked {
		return fmt.Errorf("another instance is already running")
	}

	return nil
}

// ReleaseMutex releases the file lock.
func ReleaseMutex() {
	if lock != nil {
		_ = lock.Unlock()
	}
}

func getLockFilePath(name string) string {
	var dir string
	if runtime.GOOS == "windows" {
		dir = os.Getenv("APPDATA")
	} else {
		dir = os.TempDir()
	}
	return filepath.Join(dir, name+".lock")
}
