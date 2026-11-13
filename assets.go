package main

import (
	"embed"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"

	"resume-editor/internal/apperr"
	"resume-editor/internal/resume"
)

//go:embed assets
var assets embed.FS

const (
	tplPath      = "assets/简历模板.html"
	mdPath       = "assets/简历内容.md"
	schemaPath   = "assets/schema.json"
	metadataPath = "assets/app.json"
)

func readAsset(name string) (string, error) {
	data, err := assets.ReadFile(name)
	if err != nil {
		return "", apperr.Wrap(apperr.AssetRead, err)
	}
	return string(data), nil
}

func readMetadata() (resume.Metadata, error) {
	var metadata resume.Metadata
	source, err := readAsset(metadataPath)
	if err != nil {
		return metadata, err
	}
	if err := json.Unmarshal([]byte(source), &metadata); err != nil {
		return metadata, apperr.Wrap(apperr.ConfigInvalid, err)
	}
	return metadata, nil
}

// dataDir supports an explicit portable or development data directory.
func dataDir() string {
	if dir := os.Getenv("RESUME_EDITOR_HOME"); dir != "" {
		return dir
	}
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "resume-editor")
	}
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".resume-editor")
}

func newResumeService() (*resume.Resume, error) {
	template, err := readAsset(tplPath)
	if err != nil {
		return nil, err
	}
	document, err := readAsset(mdPath)
	if err != nil {
		return nil, err
	}
	schemaSource, err := readAsset(schemaPath)
	if err != nil {
		return nil, err
	}
	schema, err := resume.ParseSchema(schemaSource)
	if err != nil {
		return nil, err
	}
	metadata, err := readMetadata()
	if err != nil {
		return nil, err
	}
	dir := dataDir()
	return resume.New(resume.Resources{
		TemplateHTML: template, DocumentMD: document, Schema: schema, Metadata: metadata,
	}, resume.StoragePaths{
		ContentFile: filepath.Join(dir, "简历内容.md"), OutputDir: filepath.Join(dir, "output"),
	}, renderPDF, slog.Default())
}
