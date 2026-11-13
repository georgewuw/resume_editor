package main

// app 层：把 go:embed 资源与用户数据路径注入 internal/resume 领域服务。
// 对应 MyGit 根级文件负责装配，internal/ 只处理纯领域逻辑。

import (
	"embed"
	"os"
	"path/filepath"

	"mygo-app/internal/resume"
)

//go:embed assets
var assets embed.FS

const (
	tplPath    = "assets/简历模板.html"
	mdPath     = "assets/简历内容.md"
	schemaPath = "assets/schema.json"
)

func readAsset(name string) (string, error) {
	b, err := assets.ReadFile(name)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// dataDir 是应用保存用户内容的目录，与开发目录隔离。
// RESUME_EDITOR_HOME 可覆盖该目录（便携模式与测试用）。
func dataDir() string {
	if d := os.Getenv("RESUME_EDITOR_HOME"); d != "" {
		return d
	}
	if d, err := os.UserConfigDir(); err == nil {
		return filepath.Join(d, "resume-editor")
	}
	h, _ := os.UserHomeDir()
	return filepath.Join(h, ".resume-editor")
}

func newResumeService() (*resume.Resume, error) {
	tpl, err := readAsset(tplPath)
	if err != nil {
		return nil, err
	}
	md, err := readAsset(mdPath)
	if err != nil {
		return nil, err
	}
	s, err := readAsset(schemaPath)
	if err != nil {
		return nil, err
	}
	schema, err := resume.ParseSchema(s)
	if err != nil {
		return nil, err
	}
	dir := dataDir()
	return resume.New(resume.Assets{
		TemplateHTML: tpl,
		DocumentMD:   md,
		Schema:       schema,
		ContentFile:  filepath.Join(dir, "简历内容.md"),
		OutputDir:    filepath.Join(dir, "output"),
	}), nil
}
