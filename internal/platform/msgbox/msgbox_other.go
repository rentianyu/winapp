//go:build !windows

package msgbox

import "fmt"

// Info 非 Windows 平台退化为标准输出，便于单测与容器内运行。
func Info(title, text string) {
	fmt.Printf("[msgbox] %s: %s\n", title, text)
}
