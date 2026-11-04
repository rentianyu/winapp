//go:build windows

package msgbox

import (
	"syscall"
	"unsafe"
)

var (
	user32       = syscall.NewLazyDLL("user32.dll")
	procMsgBoxW  = user32.NewProc("MessageBoxW")
)

// Info 弹一个只有「确定」的提示框。
func Info(title, text string) {
	t, _ := syscall.UTF16PtrFromString(title)
	b, _ := syscall.UTF16PtrFromString(text)
	// MB_OK | MB_ICONINFORMATION
	procMsgBoxW.Call(0, uintptr(unsafe.Pointer(b)), uintptr(unsafe.Pointer(t)), 0x00000040)
}
