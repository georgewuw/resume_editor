package main

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"mygo-app/internal/resume"
)

// 端到端验证 Go 服务：加载 → 写回 → 填充 → 打印 PDF。
// 用临时目录隔离，不触碰用户真实数据。
func TestResumeServiceRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("RESUME_EDITOR_HOME", home)

	svc, err := newResumeService()
	if err != nil {
		t.Fatal(err)
	}
	c, err := svc.Load()
	if err != nil {
		t.Fatalf("Load 失败：%v", err)
	}
	if len(c.Fields) != 53 {
		t.Fatalf("Load 应返回 53 个字段，实际 %d", len(c.Fields))
	}

	mdPath, err := svc.Save(c)
	if err != nil {
		t.Fatalf("Save 失败：%v", err)
	}
	mdBytes, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mdBytes), "@@brief") {
		t.Error("写回的 Markdown 缺少 brief 字段")
	}
	if strings.Contains(string(mdBytes), "使用说明") ||
		strings.Contains(string(mdBytes), ":: ") ||
		strings.Contains(string(mdBytes), "字段索引") {
		t.Error("写回的 Markdown 应只包含纯数据，不应含说明或注释")
	}
	reloaded, err := resume.ParseContent(string(mdBytes))
	if err != nil || len(reloaded.Fields) != 53 {
		t.Fatalf("写回的 Markdown 无法解析：%v，字段数 %d", err, len(reloaded.Fields))
	}

	html, err := svc.Fill(c)
	if err != nil {
		t.Fatalf("Fill 失败：%v", err)
	}
	if !strings.Contains(html, `class="filled"`) || !strings.Contains(html, "个人优势") {
		t.Error("Fill 输出缺少成品标记或栏目标题")
	}

	if _, err := resume.FindBrowser(); err != nil {
		t.Skip("本机没有 Edge / Chrome，跳过 PDF 导出验证")
	}
	pdfPath, err := svc.Export(c)
	if err != nil {
		t.Fatalf("导出 PDF 失败：%v", err)
	}
	b, err := os.ReadFile(pdfPath)
	if err != nil {
		t.Fatal(err)
	}
	pages := len(regexp.MustCompile(`/Type\s*/Page\b`).FindAll(b, -1))
	if pages != 2 {
		t.Errorf("PDF 页数应为 2，实际 %d", pages)
	}
}
