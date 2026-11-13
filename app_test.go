package main

import (
	"strings"
	"testing"

	"github.com/egoist/mygo"
	"mygo-app/internal/resume"
)

// 根级集成测试：验证 go:embed 资源、领域层装配与窗口顶栏配置。
// 领域层的纯逻辑测试放在 internal/resume/resume_test.go。

func TestEmbeddedAssets(t *testing.T) {
	tpl, err := readAsset(tplPath)
	if err != nil {
		t.Fatalf("读取内嵌模板失败：%v", err)
	}
	md, err := readAsset(mdPath)
	if err != nil {
		t.Fatalf("读取内嵌内容文档失败：%v", err)
	}
	schemaJSON, err := readAsset(schemaPath)
	if err != nil {
		t.Fatalf("读取字段定义失败：%v", err)
	}
	schema, err := resume.ParseSchema(schemaJSON)
	if err != nil {
		t.Fatal(err)
	}
	if len(schema) != 53 {
		t.Fatalf("字段数应为 53，实际 %d", len(schema))
	}
	c, err := resume.ParseContent(md)
	if err != nil {
		t.Fatalf("解析内容文档失败：%v", err)
	}
	if len(c.Fields) != 53 {
		t.Fatalf("解析出的字段数应为 53，实际 %d", len(c.Fields))
	}
	if err := resume.Validate(c, tpl); err != nil {
		t.Fatalf("字段与模板不一致：%v", err)
	}
	out, err := resume.FillTemplate(tpl, c.Fields)
	if err != nil {
		t.Fatalf("填充模板失败：%v", err)
	}
	if !strings.Contains(out, `class="filled"`) {
		t.Error("成品未取消占位高亮")
	}
	if !strings.Contains(out, `【姓名】`) {
		t.Error("填充后应保留占位文本")
	}
	written, err := resume.WriteMarkdown(c, schema)
	if err != nil {
		t.Fatalf("写回 Markdown 失败：%v", err)
	}
	if written != md {
		t.Error("写回的 Markdown 应与内容文档逐字一致（纯数据、无说明）")
	}
	if strings.Contains(written, "使用说明") || strings.Contains(written, "::") {
		t.Error("写回的 Markdown 不应包含说明或注释")
	}
	got, err := resume.ParseContent(written)
	if err != nil {
		t.Fatalf("写回的 Markdown 无法解析：%v", err)
	}
	if len(got.Fields) != 53 {
		t.Fatalf("写回后字段数应为 53，实际 %d", len(got.Fields))
	}
}

func TestCustomTitleBar(t *testing.T) {
	o := customTitleBar(mygo.WindowOptions{Title: "x"})
	if o.TitleBarStyle != mygo.TitleBarHidden {
		t.Error("应隐藏原生标题栏")
	}
	if !o.HideWindowButtons {
		t.Error("应隐藏系统窗口按钮，改由页面自绘")
	}
	if o.TitleBarHeight != titleBarHeight {
		t.Errorf("顶栏高度应为 %d，实际 %d", titleBarHeight, o.TitleBarHeight)
	}
	if o.Title != "x" || o.TrafficLightPosition == nil {
		t.Error("不得丢失原有的窗口选项")
	}
}

func TestInitialWindowSize(t *testing.T) {
	// sizeForWorkArea：工作区 → 窗口尺寸。宽度取 86%、高度取 90%，
	// 夹在 1080×680（三栏下限）与 1320×880（大屏上限）之间。
	cases := []struct {
		name                  string
		areaWidth, areaHeight int
		wantWidth, wantHeight int
	}{
		{"2240×1400 缩放 150%，工作区 1493×885", 1493, 885, 1283, 796},
		{"1920×1080 缩放 100%，工作区 1920×1040", 1920, 1040, 1320, 880},
		{"1920×1080 缩放 125%，工作区 1536×824", 1536, 824, 1320, 741},
		{"1366×768 缩放 100%，工作区 1366×728", 1366, 728, 1174, 680},
		{"1024×768 极小工作区，不得超出屏幕", 1024, 728, 1024, 680},
	}
	for _, c := range cases {
		width, height := sizeForWorkArea(c.areaWidth, c.areaHeight)
		if width != c.wantWidth || height != c.wantHeight {
			t.Errorf("%s：期望 %d×%d，实际 %d×%d",
				c.name, c.wantWidth, c.wantHeight, width, height)
		}
	}
}

func TestNewResumeServiceWiring(t *testing.T) {
	svc, err := newResumeService()
	if err != nil {
		t.Fatal(err)
	}
	info, err := svc.Info()
	if err != nil {
		t.Fatal(err)
	}
	if info["mode"] != "mygo" || info["dataDir"] == "" {
		t.Errorf("服务装配异常：%v", info)
	}
}
