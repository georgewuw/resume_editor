package resume

import (
	"strings"
	"testing"
)

// 领域层用纯字符串做单元测试，不需要窗口、webview 或内嵌资源。

func TestParseContentSkipsFencesAndComments(t *testing.T) {
	src := strings.Join([]string{
		"# 说明",
		"```bash",
		"@@name   # 代码块里的示例不应被当成字段",
		"```",
		"---",
		"## 模块一 · 个人信息（Profile）",
		"@@name",
		":: 姓名 · 个人信息",
		"【姓名】",
		"",
		"@@email",
		":: 邮箱 · 个人信息",
		"【电子邮箱】",
		"",
	}, "\n")
	c, err := ParseContent(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Fields) != 2 {
		t.Fatalf("应解析出 2 个字段，实际 %d：%v", len(c.Fields), c.Fields)
	}
	if got := c.Fields["name"][0]; got != "【姓名】" {
		t.Errorf("name = %q", got)
	}
}

func TestValidateDetectsMissingAndExtra(t *testing.T) {
	tpl := `<span data-f="a">A</span><span data-f="b">B</span>`
	ok := Content{Fields: map[string][]string{"a": {"1"}, "b": {"2"}}}
	if err := Validate(ok, tpl); err != nil {
		t.Fatal(err)
	}
	bad := Content{Fields: map[string][]string{"a": {"1"}, "z": {"9"}}}
	if err := Validate(bad, tpl); err == nil {
		t.Error("缺字段 / 多字段应报错")
	}
}

func TestFillTemplateEscapesAndMultiline(t *testing.T) {
	tpl := `<p data-f="x">a</p><span data-f="photo">b</span><div data-multiline data-f="lines">c</div>`
	out, err := FillTemplate(tpl, map[string][]string{
		"x":     {`<b>&"'`},
		"photo": {"![证件照](./photo.jpg)"},
		"lines": {"第一行", "第二行"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`&lt;b&gt;&amp;`,
		`<img src="./photo.jpg"`,
		`<span class="ln">第一行</span><span class="ln">第二行</span>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("输出缺少 %q：%s", want, out)
		}
	}
}

func TestWriteMarkdownRoundTrip(t *testing.T) {
	schema := []FieldMeta{
		{Key: "name", Group: "personal", GroupTitle: "个人信息", GroupEn: "Profile", Label: "姓名"},
		{Key: "brief", Group: "advantage", GroupTitle: "个人优势", GroupEn: "Self-Assessment", Label: "简述"},
	}
	c := Content{Fields: map[string][]string{
		"name":  {"张三"},
		"brief": {"一句话概述"},
	}}
	md, err := WriteMarkdown(c, schema)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"## 模块一 · 个人信息（Profile）",
		"## 模块二 · 个人优势（Self-Assessment）",
		"@@name\n张三", "@@brief\n一句话概述",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("输出缺少 %q：%s", want, md)
		}
	}
	for _, bad := range []string{"::", "---", "```", "使用说明"} {
		if strings.Contains(md, bad) {
			t.Errorf("输出仍含说明性内容 %q：%s", bad, md)
		}
	}
	got, err := ParseContent(md)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Fields) != 2 || got.Fields["name"][0] != "张三" || got.Fields["brief"][0] != "一句话概述" {
		t.Fatalf("写回后无法正确解析：%v", got.Fields)
	}
}
