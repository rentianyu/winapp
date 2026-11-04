// winapp — Windows 桌面应用骨架（Wails v3 + Go + TypeScript）
//
// 本文件只做四件事：日志落地、单实例检查、创建应用与窗口、跑起来。
// 任何业务逻辑都不允许写在这里。
package main

import (
	"embed"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/rentianyu/winapp/internal/platform/msgbox"
	"github.com/rentianyu/winapp/internal/platform/singleinstance"
	"github.com/rentianyu/winapp/internal/services"
)

// 前端构建产物内嵌进二进制。构建前必须先执行 npm run build 生成 frontend/dist。
//
//go:embed all:frontend/dist
var assets embed.FS

// 由 -ldflags "-X main.version=..." 注入
var (
	version = "0.0.0-dev"
	commit  = "none"
	builtAt = "unknown"
)

const appName = "JSON Toolbox"
const logMaxBytes = 5 << 20 // 5 MB

func main() {
	closeLog := setupLogging()
	defer closeLog()

	log.Printf("启动 %s version=%s commit=%s builtAt=%s", appName, version, commit, builtAt)

	// 单实例：避免多开抢占 WebView2 用户数据目录
	release, alreadyRunning := singleinstance.Acquire(`Global\winapp.singleinstance`)
	defer release()
	if alreadyRunning {
		log.Println("已有实例在运行，退出")
		msgbox.Info(appName, "程序已经在运行了。")
		return
	}

	// 调试端口：只有显式设置了 WINAPP_DEBUG_PORT 才开，避免正式版暴露 CDP
	winOpts := application.WindowsOptions{}
	if port := os.Getenv("WINAPP_DEBUG_PORT"); port != "" {
		if n, err := strconv.Atoi(port); err == nil && n > 1024 && n < 65536 {
			winOpts.AdditionalBrowserArgs = []string{
				"--remote-debugging-port=" + port,
				"--remote-allow-origins=*",
			}
			log.Printf("WebView2 CDP 已开启: http://127.0.0.1:%s", port)
		} else {
			log.Printf("WINAPP_DEBUG_PORT 非法，已忽略: %q", port)
		}
	}

	app := application.New(application.Options{
		Name:        appName,
		Description: "JSON 格式化 / 压缩 / 结构分析",
		Services: []application.Service{
			application.NewService(services.NewToolboxService(version, commit, builtAt)),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Windows: winOpts,
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            appName,
		Width:            1080,
		Height:           720,
		BackgroundColour: application.NewRGB(248, 249, 251),
		URL:              "/",
	})

	if err := app.Run(); err != nil {
		log.Printf("运行失败: %v", err)
		msgbox.Info(appName, "启动失败，详情见日志。\n\n"+err.Error())
		os.Exit(1)
	}
}

// setupLogging 把日志同时写到 stderr 与 %APPDATA%\winapp\logs\app.log（超 5MB 轮转一次）。
func setupLogging() func() {
	dir, err := os.UserConfigDir()
	if err != nil {
		return func() {}
	}
	logDir := filepath.Join(dir, "winapp", "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return func() {}
	}
	p := filepath.Join(logDir, "app.log")
	if fi, err := os.Stat(p); err == nil && fi.Size() > logMaxBytes {
		_ = os.Rename(p, p+".1")
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return func() {}
	}
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.SetOutput(io.MultiWriter(os.Stderr, f))
	log.Printf("日志文件: %s", p)
	return func() { _ = f.Close() }
}
