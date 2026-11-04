package core

import (
	"strings"
	"testing"
)

func TestFormatJSON(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		indent  int
		want    string
		wantErr bool
	}{
		{"对象", `{"a":1}`, 2, "{\n  \"a\": 1\n}", false},
		{"嵌套", `{"a":{"b":[1,2]}}`, 4, "{\n    \"a\": {\n        \"b\": [\n            1,\n            2\n        ]\n    }\n}", false},
		{"indent越界回落2", `{"a":1}`, 99, "{\n  \"a\": 1\n}", false},
		{"indent为0回落2", `{"a":1}`, 0, "{\n  \"a\": 1\n}", false},
		{"数组", `[1,2]`, 2, "[\n  1,\n  2\n]", false},
		{"空串", "  ", 2, "", true},
		{"非法", `{"a":1,}`, 2, "", true},
		{"中文引号", `{“a”:1}`, 2, "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := FormatJSON(c.in, c.indent)
			if c.wantErr {
				if err == nil {
					t.Fatalf("期望报错，实际成功: %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("意外报错: %v", err)
			}
			if got != c.want {
				t.Errorf("结果不符\n期望:\n%s\n实际:\n%s", c.want, got)
			}
		})
	}
}

func TestFormatJSONPreservesOrder(t *testing.T) {
	// json.Indent 不应重排键顺序
	in := `{"z":1,"a":2,"m":3}`
	got, err := FormatJSON(in, 2)
	if err != nil {
		t.Fatal(err)
	}
	zi, ai, mi := strings.Index(got, `"z"`), strings.Index(got, `"a"`), strings.Index(got, `"m"`)
	if !(zi < ai && ai < mi) {
		t.Errorf("键顺序被改变: %s", got)
	}
}

func TestMinifyJSON(t *testing.T) {
	got, err := MinifyJSON("{\n  \"a\": 1,\n  \"b\": [1, 2]\n}")
	if err != nil {
		t.Fatal(err)
	}
	if got != `{"a":1,"b":[1,2]}` {
		t.Errorf("压缩结果不符: %q", got)
	}
	if _, err := MinifyJSON(""); err == nil {
		t.Error("空输入应报错")
	}
}

func TestParseErrorLineCol(t *testing.T) {
	in := "{\n  \"a\": 1,\n  \"b\": oops\n}"
	se, err := ParseError(in)
	if err != nil {
		t.Fatalf("ParseError 不应返回 err: %v", err)
	}
	if se == nil {
		t.Fatal("非法 JSON 应返回 SyntaxError")
	}
	if se.Line != 3 {
		t.Errorf("行号期望 3，实际 %d (%v)", se.Line, se)
	}
	if se.Column < 1 {
		t.Errorf("列号应 >=1，实际 %d", se.Column)
	}
	if !strings.Contains(se.Error(), "第 3 行") {
		t.Errorf("错误文案应含行号: %s", se.Error())
	}
}

func TestParseErrorValid(t *testing.T) {
	if se, err := ParseError(`{"a":1}`); se != nil || err != nil {
		t.Errorf("合法 JSON 不应报错: %v %v", se, err)
	}
}

func TestAnalyze(t *testing.T) {
	s, err := Analyze(`{"a":1,"b":"x","c":[true,null,{"d":2.5}]}`)
	if err != nil {
		t.Fatal(err)
	}
	if s.TopLevel != "object" {
		t.Errorf("TopLevel = %s", s.TopLevel)
	}
	if s.Types["object"] != 2 {
		t.Errorf("object 计数 = %d", s.Types["object"])
	}
	if s.Types["array"] != 1 {
		t.Errorf("array 计数 = %d", s.Types["array"])
	}
	if s.Types["string"] != 1 || s.Types["number"] != 2 || s.Types["boolean"] != 1 || s.Types["null"] != 1 {
		t.Errorf("类型统计不符: %v", s.Types)
	}
	if s.Depth < 2 {
		t.Errorf("深度应 >=2，实际 %d", s.Depth)
	}
	if s.Bytes != len(`{"a":1,"b":"x","c":[true,null,{"d":2.5}]}`) {
		t.Errorf("字节数不符: %d", s.Bytes)
	}
}

func TestAnalyzeEmpty(t *testing.T) {
	if _, err := Analyze("   "); err == nil {
		t.Error("空输入应报错")
	}
}
