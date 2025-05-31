//go:build windows

package appmutex

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procCreateMutexW = kernel32.NewProc("CreateMutexW")
	procReleaseMutex = kernel32.NewProc("ReleaseMutex")
	procCloseHandle  = kernel32.NewProc("CloseHandle")
)

const ERROR_ALREADY_EXISTS = 183

func CreateMutex(name string) (syscall.Handle, error) {
	if runtime.GOOS != "windows" {
		return 0, fmt.Errorf("not supported on %s", runtime.GOOS)
	}

	namePtr, err := syscall.UTF16PtrFromString("Local\\" + name)
	if err != nil {
		return 0, fmt.Errorf("invalid mutex name: %w", err)
	}

	handle, _, errCreate := procCreateMutexW.Call(0, 0, uintptr(unsafe.Pointer(namePtr)))

	if handle == 0 {
		return 0, fmt.Errorf("CreateMutexW failed: %v", errCreate)
	}

	// The error returned from Call may indicate the mutex already existed
	if errCreate == syscall.ERROR_ALREADY_EXISTS {
		// Handle exists but we don't want to use it
		procCloseHandle.Call(handle)
		return 0, fmt.Errorf("mutex already exists")
	}

	return syscall.Handle(handle), nil
}

// ReleaseMutex releases and closes the handle.
func ReleaseMutex(handle syscall.Handle) {
	if handle != 0 {
		procReleaseMutex.Call(uintptr(handle))
		procCloseHandle.Call(uintptr(handle))
	}
}
