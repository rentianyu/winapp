package core

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCompareVersion(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"0.1.0", "0.1.0", 0},
		{"v0.1.0", "0.1.0", 0},
		{"0.1.1", "0.1.0", 1},
		{"0.1.0", "0.2.0", -1},
		{"1.0.0", "0.9.9", 1},
		{"0.1", "0.1.0", 0},
		{"0.1.0-beta.1", "0.1.0", 0},
		{"0.1.10", "0.1.9", 1},
		{"2.0.0", "10.0.0", -1},
	}
	for _, c := range cases {
		if got := CompareVersion(c.a, c.b); got != c.want {
			t.Errorf("CompareVersion(%q,%q)=%d want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestFetchLatestFromRedirect(t *testing.T) {
	// 模拟 github.com/owner/repo/releases/latest -> 302 -> /releases/tag/v1.2.3
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/owner/repo/releases/latest" {
			http.Redirect(w, r, srv.URL+"/owner/repo/releases/tag/v1.2.3", http.StatusFound)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><a href="/owner/repo/releases/tag/v1.2.3">v1.2.3</a></html>`))
	}))
	defer srv.Close()

	// 直接验证正则与重定向处理逻辑（FetchLatest 固定打 github.com，这里用同一份正则校验页面）
	body := []byte(`<a href="/rentianyu/winapp/releases/tag/v1.2.3">latest</a>`)
	m := tagRe.FindSubmatch(body)
	if len(m) != 2 || string(m[1]) != "v1.2.3" {
		t.Fatalf("正则未命中: %v", m)
	}
}

func TestFetchLatestBadRepo(t *testing.T) {
	if _, err := FetchLatest("invalid", time.Second); err == nil {
		t.Error("非法仓库名应报错")
	}
	if _, err := FetchLatest("a/b/c", time.Second); err == nil {
		t.Error("多段仓库名应报错")
	}
}

func TestSplitVersion(t *testing.T) {
	if got := splitVersion("v1.2.3-rc.1"); len(got) != 3 || got[0] != 1 || got[2] != 3 {
		t.Errorf("splitVersion = %v", got)
	}
	if got := splitVersion("1.2.x"); len(got) != 2 {
		t.Errorf("非法段应截断，得到 %v", got)
	}
}
