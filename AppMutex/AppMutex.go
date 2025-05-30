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

func CreateMutex(name string) (any, error) {
	if runtime.GOOS != "windows" {
		return nil, fmt.Errorf("not supported on %s", runtime.GOOS)
	}
	namePtr, _ := syscall.UTF16PtrFromString("Global\\" + name)
	handle, _, _ := procCreateMutexW.Call(0, 0, uintptr(unsafe.Pointer(namePtr)))

	if handle == 0 {
		return nil, fmt.Errorf("CreateMutex failed")
	}

	lastErr := syscall.GetLastError()
	if lastErr == syscall.Errno(ERROR_ALREADY_EXISTS) {
		procCloseHandle.Call(handle)
		return nil, fmt.Errorf("mutex already exists")
	}

	return syscall.Handle(handle), nil
}

func ReleaseMutex(handle any) {
	h, ok := handle.(syscall.Handle)
	if ok && h != 0 {
		procReleaseMutex.Call(uintptr(h))
		procCloseHandle.Call(uintptr(h))
	}
}
