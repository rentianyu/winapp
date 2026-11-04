// Package core 包含全部纯业务逻辑。
//
// 铁律：本包不得 import 任何与操作系统相关的包（不许出现 windows、syscall、
// golang.org/x/sys/windows 等），保证 100% 可在 Linux/macOS 上跑单测。
// 需要平台能力时，通过接口注入，实现放 internal/platform/<os>。
package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// ErrEmpty 输入为空。
var ErrEmpty = errors.New("输入为空")

// FormatJSON 格式化 JSON。indent 为缩进空格数，取值 1..8，越界时回落 2。
func FormatJSON(raw string, indent int) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ErrEmpty
	}
	if indent < 1 || indent > 8 {
		indent = 2
	}
	if !json.Valid([]byte(raw)) {
		if _, err := ParseError(raw); err != nil {
			return "", err
		}
		return "", errors.New("JSON 不合法")
	}
	var out bytes.Buffer
	if err := json.Indent(&out, []byte(raw), "", strings.Repeat(" ", indent)); err != nil {
		return "", fmt.Errorf("格式化失败: %w", err)
	}
	return out.String(), nil
}

// MinifyJSON 压缩 JSON（去掉所有空白）。
func MinifyJSON(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ErrEmpty
	}
	var out bytes.Buffer
	if err := json.Compact(&out, []byte(raw)); err != nil {
		if _, perr := ParseError(raw); perr != nil {
			return "", perr
		}
		return "", fmt.Errorf("压缩失败: %w", err)
	}
	return out.String(), nil
}

// ParseError 把 encoding/json 的报错翻译成人话，并带上行号列号。
// 返回 nil 表示解析成功。
func ParseError(raw string) (*SyntaxError, error) {
	var v any
	err := json.Unmarshal([]byte(raw), &v)
	if err == nil {
		return nil, nil
	}
	var se *json.SyntaxError
	if errors.As(err, &se) {
		line, col := offsetToLineCol(raw, int(se.Offset))
		return &SyntaxError{
			Offset: int(se.Offset),
			Line:   line,
			Column: col,
			Msg:    humanize(se.Error()),
		}, nil
	}
	return &SyntaxError{Msg: err.Error()}, nil
}

// SyntaxError 结构化语法错误。
type SyntaxError struct {
	Offset int    `json:"offset"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
	Msg    string `json:"msg"`
}

func (e *SyntaxError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("第 %d 行第 %d 列: %s", e.Line, e.Column, e.Msg)
	}
	return e.Msg
}

// Stats 结构统计，用于 UI 上的一行摘要。
type Stats struct {
	Bytes    int            `json:"bytes"`
	Chars    int            `json:"chars"`
	Depth    int            `json:"depth"`
	Nodes    int            `json:"nodes"`
	Types    map[string]int `json:"types"`
	TopLevel string         `json:"topLevel"`
}

// Analyze 统计 JSON 结构信息。
func Analyze(raw string) (Stats, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Stats{}, ErrEmpty
	}
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		if se, _ := ParseError(raw); se != nil {
			return Stats{}, se
		}
		return Stats{}, err
	}
	s := Stats{Bytes: len(raw), Chars: len([]rune(raw)), Types: map[string]int{}}
	s.TopLevel, s.Depth, s.Nodes = walk(v, 0, &s)
	return s, nil
}

func walk(v any, depth int, s *Stats) (kind string, maxDepth, nodes int) {
	nodes = 1
	switch t := v.(type) {
	case map[string]any:
		kind = "object"
		s.Types["object"]++
		maxDepth = depth
		for _, child := range t {
			k, d, n := walk(child, depth+1, s)
			_ = k
			nodes += n
			if d > maxDepth {
				maxDepth = d
			}
		}
	case []any:
		kind = "array"
		s.Types["array"]++
		maxDepth = depth
		for _, child := range t {
			k, d, n := walk(child, depth+1, s)
			_ = k
			nodes += n
			if d > maxDepth {
				maxDepth = d
			}
		}
	case string:
		kind = "string"
		s.Types["string"]++
		maxDepth = depth
	case float64:
		kind = "number"
		s.Types["number"]++
		maxDepth = depth
	case bool:
		kind = "boolean"
		s.Types["boolean"]++
		maxDepth = depth
	case nil:
		kind = "null"
		s.Types["null"]++
		maxDepth = depth
	default:
		kind = "unknown"
		maxDepth = depth
	}
	return kind, maxDepth, nodes
}

func offsetToLineCol(s string, offset int) (line, col int) {
	if offset > len(s) {
		offset = len(s)
	}
	line, col = 1, 1
	for i := 0; i < offset; i++ {
		if s[i] == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}
	return line, col
}

func humanize(msg string) string {
	switch {
	case strings.Contains(msg, "unexpected end of JSON input"):
		return "内容不完整（可能少了右括号/引号）"
	case strings.Contains(msg, "invalid character"):
		return "存在非法字符（注意中文引号、单引号、尾随逗号）"
	case strings.Contains(msg, "cannot unmarshal"):
		return "类型不匹配"
	}
	return msg
}
