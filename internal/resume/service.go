package resume

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"resume-editor/internal/apperr"
	"resume-editor/internal/logging"
)

// Resume is the desktop service. Only intentional frontend APIs are exported.
type Resume struct {
	mu                sync.Mutex
	resources         Resources
	paths             StoragePaths
	template          *Template
	renderPDF         PDFRenderer
	logger            *slog.Logger
	webDAVCredentials webDAVCredentialStore
}

// New compiles shared definitions and assembles the desktop service.
func New(resources Resources, paths StoragePaths, renderPDF PDFRenderer, logger *slog.Logger) (*Resume, error) {
	template, err := CompileTemplate(resources.TemplateHTML)
	if err != nil {
		return nil, err
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Resume{
		resources: resources, paths: paths, template: template, renderPDF: renderPDF,
		logger: logger.With("component", "resume"), webDAVCredentials: systemWebDAVCredentialStore{},
	}, nil
}

func (r *Resume) failure(operation string, code apperr.Code, cause error) error {
	err := apperr.Wrap(code, cause)
	logging.Failure(r.logger, operation, err)
	return err
}

// Load reads saved content, falling back to the bundled document on first use.
func (r *Resume) Load() (Content, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	source := "user"
	data, err := os.ReadFile(r.paths.ContentFile)
	if os.IsNotExist(err) {
		data, err = []byte(r.resources.DocumentMD), nil
		source = "default"
	}
	if err != nil {
		return Content{}, r.failure("load", apperr.ContentLoad, err)
	}
	content, err := ParseContent(string(data))
	if err != nil {
		return Content{}, r.failure("load", apperr.ContentLoad, err)
	}
	r.logger.Info("content_loaded", "source", source)
	return content, nil
}

// Fill returns the same HTML used by desktop PDF export.
func (r *Resume) Fill(content Content) (string, error) {
	if err := r.template.Validate(content); err != nil {
		return "", r.failure("preview", apperr.PreviewFailed, err)
	}
	html, err := r.filledHTML(content.Fields)
	if err != nil {
		return "", r.failure("preview", apperr.PreviewFailed, err)
	}
	return html, nil
}

func (r *Resume) filledHTML(fields map[string][]string) (string, error) {
	fields, err := embedLocalPhoto(fields, r.paths.OutputDir)
	if err != nil {
		return "", err
	}
	return r.template.Fill(fields), nil
}

// Save serializes content using schema order and writes the user's document.
func (r *Resume) Save(content Content) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.template.Validate(content); err != nil {
		return "", r.failure("save", apperr.ContentSave, err)
	}
	markdown, err := WriteMarkdown(content, r.resources.Schema)
	if err != nil {
		return "", r.failure("save", apperr.ContentSave, err)
	}
	if err := os.MkdirAll(filepath.Dir(r.paths.ContentFile), 0o755); err != nil {
		return "", r.failure("save", apperr.ContentSave, err)
	}
	if err := os.WriteFile(r.paths.ContentFile, []byte(markdown), 0o644); err != nil {
		return "", r.failure("save", apperr.ContentSave, err)
	}
	r.logger.Info("content_saved")
	return r.paths.ContentFile, nil
}

// Export renders the document and delegates PDF creation to the desktop adapter.
func (r *Resume) Export(filename string, content Content) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, err := r.readDocument(filename); err != nil {
		return "", r.failure("export", apperr.ExportFailed, err)
	}
	if err := r.template.Validate(content); err != nil {
		return "", r.failure("export", apperr.ExportFailed, err)
	}
	started := time.Now()
	r.logger.Info("export_started")
	html, err := r.filledHTML(content.Fields)
	if err != nil {
		return "", r.failure("export", apperr.ExportFailed, err)
	}
	base := "成品简历"
	if filename != filepath.Base(r.paths.ContentFile) {
		base += "_" + strings.TrimSuffix(strings.TrimPrefix(filename, "简历_"), ".md")
	}
	result, err := exportPDF(r.paths.OutputDir, base, html, r.renderPDF)
	if err != nil {
		return "", r.failure("export", apperr.ExportFailed, err)
	}
	r.logger.Info("export_completed", "duration_ms", time.Since(started).Milliseconds())
	return result, nil
}

// Info returns application paths and identity for the desktop editor.
func (r *Resume) Info() (map[string]string, error) {
	return map[string]string{
		"dataDir": filepath.Dir(r.paths.ContentFile),
		"logDir":  filepath.Join(filepath.Dir(r.paths.ContentFile), "logs"),
		"mode":    "mygo",
		"version": r.resources.Metadata.Version,
	}, nil
}

// ClientLog deliberately has no free-form context or resume-content fields.
type ClientLog struct {
	Level string      `json:"level"`
	Event string      `json:"event"`
	Code  apperr.Code `json:"code,omitempty"`
}

// Log records frontend operation names and centrally defined error codes.
func (r *Resume) Log(entry ClientLog) {
	level := slog.LevelInfo
	switch entry.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	attrs := []any{"source", "frontend"}
	if entry.Code != "" {
		if !apperr.Known(entry.Code) {
			entry.Code = apperr.Unexpected
		}
		attrs = append(attrs, "code", entry.Code)
	}
	r.logger.Log(context.Background(), level, entry.Event, attrs...)
}
