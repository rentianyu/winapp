//go:build windows

package singleinstance

import (
	"syscall"
	"unsafe"
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procCreateMutexW = kernel32.NewProc("CreateMutexW")
)

const errorAlreadyExists = 183

// Acquire 尝试获取命名互斥量。
// alreadyRunning 为 true 表示已有实例在跑，调用方应提示用户并退出。
// release 在进程退出前调用（正常退出时系统也会自动释放）。
func Acquire(name string) (release func(), alreadyRunning bool) {
	p, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return func() {}, false
	}
	h, _, callErr := procCreateMutexW.Call(0, 0, uintptr(unsafe.Pointer(p)))
	if h == 0 {
		return func() {}, false
	}
	handle := syscall.Handle(h)
	release = func() { _ = syscall.CloseHandle(handle) }
	if errno, ok := callErr.(syscall.Errno); ok && errno == errorAlreadyExists {
		return release, true
	}
	return release, false
}
