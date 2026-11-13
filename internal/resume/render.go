package resume

// 模板填充与成品导出。与 frontend/app.js 的 fillTemplate 使用同一套
// data-f 扫描规则，保证桌面端与浏览器降级模式渲染结果一致。

import (
	"errors"
	"fmt"
	"html"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var imageSyntax = regexp.MustCompile(`^!\[(.*?)\]\((.+?)\)$`)

// FillTemplate 用扫描器替换每个 data-f 占位元素，避免依赖正则的反向引用。
func FillTemplate(tpl string, fields map[string][]string) (string, error) {
	var b strings.Builder
	rest := tpl
	for {
		idx := strings.Index(rest, `data-f="`)
		if idx < 0 {
			b.WriteString(rest)
			break
		}
		keyStart := idx + len(`data-f="`)
		keyEnd := strings.Index(rest[keyStart:], `"`)
		if keyEnd < 0 {
			return "", errors.New("模板中存在未闭合的 data-f 属性")
		}
		key := rest[keyStart : keyStart+keyEnd]

		openStart := strings.LastIndex(rest[:idx], "<")
		openEnd := strings.Index(rest[idx:], ">")
		if openStart < 0 || openEnd < 0 {
			return "", fmt.Errorf("字段 %s 的占位标签不完整", key)
		}
		openEndAbs := idx + openEnd
		attrs := rest[openStart+1 : openEndAbs]
		tag := attrs
		if i := strings.IndexAny(tag, " \t\n"); i >= 0 {
			tag = tag[:i]
		}
		closeTag := "</" + tag + ">"
		closeRel := strings.Index(rest[openEndAbs+1:], closeTag)
		if closeRel < 0 {
			return "", fmt.Errorf("字段 %s 缺少闭合标签 %s", key, closeTag)
		}
		closeAbs := openEndAbs + 1 + closeRel

		lines := fields[key]
		var inner string
		switch {
		case strings.Contains(attrs, "data-multiline"):
			for _, l := range lines {
				inner += `<span class="ln">` + html.EscapeString(l) + `</span>`
			}
		case key == "photo":
			val := strings.TrimSpace(strings.Join(lines, "\n"))
			if m := imageSyntax.FindStringSubmatch(val); m != nil {
				alt := m[1]
				if alt == "" {
					alt = "证件照"
				}
				inner = `<img src="` + html.EscapeString(m[2]) + `" alt="` + html.EscapeString(alt) + `">`
			} else {
				inner = `<span class="photo-tip">` + html.EscapeString(val) + `</span>`
			}
		default:
			parts := make([]string, 0, len(lines))
			for _, l := range lines {
				parts = append(parts, html.EscapeString(l))
			}
			inner = strings.Join(parts, "<br>")
		}

		b.WriteString(rest[:openStart])
		b.WriteString("<" + attrs + ">")
		b.WriteString(inner)
		b.WriteString(closeTag)
		rest = rest[closeAbs+len(closeTag):]
	}
	out := b.String()
	// 成品取消模板态的占位高亮
	out = strings.Replace(out, `<body data-layout="double">`,
		`<body data-layout="double" class="filled">`, 1)
	return out, nil
}

// FindBrowser 定位用于无头打印 PDF 的 Edge / Chrome。
func FindBrowser() (string, error) {
	candidates := []string{
		`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
		`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
		`C:\Program Files\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	for _, n := range []string{"msedge", "chrome", "google-chrome"} {
		if p, err := exec.LookPath(n); err == nil {
			return p, nil
		}
	}
	return "", errors.New("未找到 Microsoft Edge / Chrome，无法导出 PDF")
}

// writePrintSource 校验并把填充结果写成 Edge 无头打印所需的本地 HTML。
// 它是导出 PDF 的中间产物，不再作为独立的「导出 HTML」对外提供。
func writePrintSource(a Assets, c Content) (string, error) {
	if err := Validate(c, a.TemplateHTML); err != nil {
		return "", err
	}
	out, err := FillTemplate(a.TemplateHTML, c.Fields)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(a.OutputDir, 0o755); err != nil {
		return "", err
	}
	p := filepath.Join(a.OutputDir, "成品简历.html")
	if err := os.WriteFile(p, []byte(out), 0o644); err != nil {
		return "", err
	}
	return p, nil
}

// ExportPDF 复用 Edge 无头打印，输出 A4 双页 PDF。
func ExportPDF(a Assets, c Content) (string, error) {
	htmlPath, err := writePrintSource(a, c)
	if err != nil {
		return "", err
	}
	browser, err := FindBrowser()
	if err != nil {
		return "", err
	}
	tmp, err := os.MkdirTemp("", "resume-mygo-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)
	pdf := filepath.Join(a.OutputDir, "成品简历.pdf")
	u := "file://" + filepath.ToSlash(htmlPath)
	if !strings.HasPrefix(u, "file:///") {
		u = "file:///" + strings.TrimPrefix(u, "file://")
	}
	cmd := exec.Command(browser,
		"--headless=new", "--disable-gpu", "--no-sandbox",
		"--disable-extensions", "--no-first-run",
		"--user-data-dir="+filepath.Join(tmp, "profile"),
		"--no-pdf-header-footer", "--virtual-time-budget=6000",
		"--print-to-pdf="+pdf, u)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("PDF 打印失败：%v %s", err, string(out))
	}
	if st, err := os.Stat(pdf); err != nil || st.Size() < 2000 {
		return "", errors.New("PDF 未生成或文件过小")
	}
	return pdf, nil
}
