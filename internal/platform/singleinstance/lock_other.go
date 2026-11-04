//go:build !windows

package singleinstance

// Acquire 在非 Windows 平台上是空实现，保证交叉编译与单测可用。
func Acquire(name string) (release func(), alreadyRunning bool) {
	return func() {}, false
}
