package core

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// 公开发布后用户需要知道有没有新版。这里刻意不用 GitHub API：
// 未认证的 API 按出口 IP 限流 60 次/小时，用户挂代理时极易 403。
// 抓 releases/latest 页面（会 302 到具体 tag）更稳。
var tagRe = regexp.MustCompile(`releases/tag/(v?[0-9]+\.[0-9]+\.[0-9]+[A-Za-z0-9._-]*)`)

const maxHTMLBytes = 2 << 20 // 2 MB 足够，防止异常大响应

// Release 描述一次版本查询的结果。
type Release struct {
	Tag      string `json:"tag"`
	Version  string `json:"version"`
	URL      string `json:"url"`
	HasNewer bool   `json:"hasNewer"`
}

// FetchLatest 抓取 GitHub 仓库最新 release 的 tag。
// repo 形如 "rentianyu/winapp"。
func FetchLatest(repo string, timeout time.Duration) (Release, error) {
	repo = strings.Trim(strings.TrimSpace(repo), "/")
	if strings.Count(repo, "/") != 1 {
		return Release{}, fmt.Errorf("仓库格式应为 owner/name，收到 %q", repo)
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	client := &http.Client{Timeout: timeout}

	req, err := http.NewRequest(http.MethodGet, "https://github.com/"+repo+"/releases/latest", nil)
	if err != nil {
		return Release{}, err
	}
	req.Header.Set("User-Agent", "winapp-updater/1.0")
	req.Header.Set("Accept", "text/html")

	resp, err := client.Do(req)
	if err != nil {
		return Release{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("查询失败: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxHTMLBytes))
	if err != nil {
		return Release{}, err
	}

	// 优先用重定向后的最终 URL（最准确），其次正则兜底
	tag := ""
	if resp.Request != nil && resp.Request.URL != nil {
		if m := tagRe.FindStringSubmatch(resp.Request.URL.String()); len(m) == 2 {
			tag = m[1]
		}
	}
	if tag == "" {
		if m := tagRe.FindSubmatch(body); len(m) == 2 {
			tag = string(m[1])
		}
	}
	if tag == "" {
		return Release{}, errors.New("没找到版本号（仓库可能还没有 Release）")
	}

	return Release{
		Tag:     tag,
		Version: strings.TrimPrefix(tag, "v"),
		URL:     "https://github.com/" + repo + "/releases/tag/" + tag,
	}, nil
}

// CompareVersion 比较两个版本号（允许前缀 v）。
// 返回 -1 / 0 / 1，语义同 strings.Compare。
func CompareVersion(a, b string) int {
	pa := splitVersion(a)
	pb := splitVersion(b)
	n := len(pa)
	if len(pb) > n {
		n = len(pb)
	}
	for i := 0; i < n; i++ {
		var x, y int
		if i < len(pa) {
			x = pa[i]
		}
		if i < len(pb) {
			y = pb[i]
		}
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	return 0
}

func splitVersion(v string) []int {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	// 去掉 -beta.1 之类的后缀，只比数字段
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			break
		}
		out = append(out, n)
	}
	return out
}
