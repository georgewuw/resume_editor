package resume

import (
	"os"
	"path/filepath"
	"sync"
)

// Resume 是编辑器页面可以调用的 Go 服务：内容读写、模板填充、成品导出。
// 类型名必须为 Resume，mygo.Bind 据此生成前端的 Resume.* 调用。
type Resume struct {
	mu     sync.Mutex
	assets Assets
}

// New 构造服务；资源与路径由 app 层注入。
func New(a Assets) *Resume { return &Resume{assets: a} }

// dataDir 返回用户内容所在目录，供状态栏显示。
// 不导出，避免 mygo.Bind 把内部辅助方法暴露成前端 API。
func (r *Resume) dataDir() string { return filepath.Dir(r.assets.ContentFile) }

// Load 读取用户内容；首次运行时回退到内嵌的默认文档。
func (r *Resume) Load() (Content, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if b, err := os.ReadFile(r.assets.ContentFile); err == nil {
		return ParseContent(string(b))
	}
	return ParseContent(r.assets.DocumentMD)
}

// Fill 返回填充后的完整 HTML（预览与导出共用同一条渲染链路）。
func (r *Resume) Fill(c Content) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := Validate(c, r.assets.TemplateHTML); err != nil {
		return "", err
	}
	return FillTemplate(r.assets.TemplateHTML, c.Fields)
}

// Save 把当前编辑内容写回 Markdown 文档。
func (r *Resume) Save(c Content) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := Validate(c, r.assets.TemplateHTML); err != nil {
		return "", err
	}
	md, err := WriteMarkdown(c, r.assets.Schema)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(r.assets.ContentFile), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(r.assets.ContentFile, []byte(md), 0o644); err != nil {
		return "", err
	}
	return r.assets.ContentFile, nil
}

// Export 生成 A4 双页 PDF，返回写入的绝对路径。
func (r *Resume) Export(c Content) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return ExportPDF(r.assets, c)
}

// Info 返回运行环境信息，供编辑器状态栏显示。
func (r *Resume) Info() (map[string]string, error) {
	return map[string]string{
		"dataDir": r.dataDir(),
		"mode":    "mygo",
		"version": "0.1.0",
	}, nil
}
