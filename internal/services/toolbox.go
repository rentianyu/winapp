// Package services 是对外暴露给前端的薄封装层。
//
// 约定：每个方法只做三件事——参数校正、调用 core、错误翻译。
// 不允许在这里写业务逻辑（那样就没法在 Linux 上测了）。
package services

import (
	"runtime"
	"time"

	"github.com/rentianyu/winapp/internal/core"
)

// ToolboxService JSON 工具箱服务。
type ToolboxService struct {
	Version string
	Commit  string
	BuiltAt string
}

// NewToolboxService 构造服务。
func NewToolboxService(version, commit, builtAt string) *ToolboxService {
	return &ToolboxService{Version: version, Commit: commit, BuiltAt: builtAt}
}

// FormatJSON 格式化。
func (s *ToolboxService) FormatJSON(raw string, indent int) (string, error) {
	return core.FormatJSON(raw, indent)
}

// MinifyJSON 压缩。
func (s *ToolboxService) MinifyJSON(raw string) (string, error) {
	return core.MinifyJSON(raw)
}

// Analyze 结构统计。失败时同时返回结构化语法错误，前端可直接定位行列。
func (s *ToolboxService) Analyze(raw string) (core.Stats, error) {
	return core.Analyze(raw)
}

// Check 只校验，返回 null 表示合法，否则返回行列信息。
func (s *ToolboxService) Check(raw string) (*core.SyntaxError, error) {
	se, err := core.ParseError(raw)
	if err != nil {
		return nil, err
	}
	return se, nil
}

// Info 应用信息（供「关于」页面用）。
type Info struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	BuiltAt string `json:"builtAt"`
	GoVer   string `json:"goVersion"`
	OS      string `json:"os"`
	Arch    string `json:"arch"`
	Now     string `json:"now"`
}

// CheckUpdate 查询 GitHub 最新版本并和当前版本比较。
func (s *ToolboxService) CheckUpdate(repo string) (core.Release, error) {
	r, err := core.FetchLatest(repo, 10*time.Second)
	if err != nil {
		return core.Release{}, err
	}
	r.HasNewer = core.CompareVersion(r.Version, s.Version) > 0
	return r, nil
}

// GetInfo 返回运行时信息。
func (s *ToolboxService) GetInfo() Info {
	return Info{
		Version: s.Version,
		Commit:  s.Commit,
		BuiltAt: s.BuiltAt,
		GoVer:   runtime.Version(),
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
		Now:     time.Now().Format("2006-01-02 15:04:05"),
	}
}
