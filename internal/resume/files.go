package resume

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"resume-editor/internal/apperr"
)

const profilePrefix = ":: resume-profile "

// DocumentInfo travels with a Markdown file; the legacy document needs no header.
type DocumentInfo struct {
	Filename   string `json:"filename"`
	Company    string `json:"company"`
	Role       string `json:"role"`
	JD         string `json:"jd"`
	ModifiedAt string `json:"modifiedAt"`
}

type ResumeDocument struct {
	Document DocumentInfo `json:"document"`
	Content  Content      `json:"content"`
}

type NewDocument struct {
	Company string `json:"company"`
	Role    string `json:"role"`
	JD      string `json:"jd"`
	Source  string `json:"source"`
}

type DocumentWorkspace struct {
	Documents []DocumentInfo `json:"documents"`
	Current   ResumeDocument `json:"current"`
}

// Workspace restores the last opened file; a removed file falls back to the general resume.
func (r *Resume) Workspace() (DocumentWorkspace, error) {
	documents, err := r.ListDocuments()
	if err != nil {
		return DocumentWorkspace{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	filename := filepath.Base(r.paths.ContentFile)
	if saved, err := os.ReadFile(filepath.Join(filepath.Dir(r.paths.ContentFile), "active-document.json")); err == nil {
		var previous string
		if json.Unmarshal(saved, &previous) == nil {
			for _, document := range documents {
				if document.Filename == previous {
					filename = previous
				}
			}
		}
	}
	current, err := r.readDocument(filename)
	if err != nil {
		return DocumentWorkspace{}, r.failure("workspace", apperr.ContentLoad, err)
	}
	return DocumentWorkspace{Documents: documents, Current: current}, nil
}

func (r *Resume) rememberDocument(filename string) error {
	dir := filepath.Dir(r.paths.ContentFile)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(filename)
	if err != nil {
		return err
	}
	return writeDocumentFile(filepath.Join(dir, "active-document.json"), data)
}

type ContentProgress struct {
	Filled int            `json:"filled"`
	Total  int            `json:"total"`
	Groups map[string]int `json:"groups"`
}

// Progress keeps the definition of a completed field in the document service.
func (r *Resume) Progress(content Content) (ContentProgress, error) {
	initial, err := ParseContent(r.resources.DocumentMD)
	if err != nil {
		return ContentProgress{}, r.failure("progress", apperr.DocumentInvalid, err)
	}
	progress := ContentProgress{Total: len(r.resources.Schema), Groups: map[string]int{}}
	for _, field := range r.resources.Schema {
		value := strings.TrimSpace(strings.Join(content.Fields[field.Key], "\n"))
		original := strings.TrimSpace(strings.Join(initial.Fields[field.Key], "\n"))
		if value == "" || value == original || (strings.HasPrefix(value, "【") && strings.HasSuffix(value, "】")) {
			continue
		}
		progress.Filled++
		progress.Groups[field.Group]++
	}
	return progress, nil
}

// A failed write leaves the last complete version available.
func writeDocumentFile(path string, data []byte) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".resume-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}

func filenamePart(value string) string {
	var out []rune
	for _, ch := range strings.TrimSpace(value) {
		if unicode.IsControl(ch) || strings.ContainsRune(`<>:"/\|?*`, ch) || unicode.IsSpace(ch) {
			ch = '_'
		}
		if ch == '_' && len(out) > 0 && out[len(out)-1] == '_' {
			continue
		}
		out = append(out, ch)
		if len(out) == 40 {
			break
		}
	}
	return strings.Trim(string(out), "_. ")
}

func (r *Resume) documentPath(filename string) (string, error) {
	legacy := filepath.Base(r.paths.ContentFile)
	if filename != legacy && (!strings.HasPrefix(filename, "简历_") || !strings.HasSuffix(filename, ".md")) {
		return "", errors.New("unmanaged document filename")
	}
	if filename == "" || filepath.Base(filename) != filename || strings.ContainsAny(filename, `/\:`) {
		return "", errors.New("invalid document filename")
	}
	return filepath.Join(filepath.Dir(r.paths.ContentFile), filename), nil
}

func (r *Resume) readDocument(filename string) (ResumeDocument, error) {
	path, err := r.documentPath(filename)
	if err != nil {
		return ResumeDocument{}, err
	}
	document := ResumeDocument{Document: DocumentInfo{Filename: filename}}
	stat, err := os.Lstat(path)
	var data []byte
	if os.IsNotExist(err) && filename == filepath.Base(r.paths.ContentFile) {
		data = []byte(r.resources.DocumentMD)
	} else {
		if err != nil {
			return document, err
		}
		if !stat.Mode().IsRegular() {
			return document, errors.New("document is not a regular file")
		}
		data, err = os.ReadFile(path)
		if err != nil {
			return document, err
		}
		document.Document.ModifiedAt = stat.ModTime().UTC().Format(time.RFC3339)
	}
	if strings.HasPrefix(string(data), profilePrefix) {
		header, _, _ := strings.Cut(string(data), "\n")
		if err := json.Unmarshal([]byte(strings.TrimPrefix(header, profilePrefix)), &document.Document); err != nil {
			return document, err
		}
		// Disk identity and modification time never come from the header.
		document.Document.Filename = filename
		if stat != nil {
			document.Document.ModifiedAt = stat.ModTime().UTC().Format(time.RFC3339)
		}
	}
	document.Content, err = ParseContent(string(data))
	if err == nil {
		err = r.template.Validate(document.Content)
	}
	return document, err
}

// ListDocuments includes the bundled general resume before the first save.
func (r *Resume) ListDocuments() ([]DocumentInfo, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	legacy, err := r.readDocument(filepath.Base(r.paths.ContentFile))
	if err != nil {
		return nil, r.failure("list_documents", apperr.ContentLoad, err)
	}
	items := []DocumentInfo{legacy.Document}
	entries, err := os.ReadDir(filepath.Dir(r.paths.ContentFile))
	if err != nil && !os.IsNotExist(err) {
		return nil, r.failure("list_documents", apperr.ContentLoad, err)
	}
	for _, entry := range entries {
		if entry.Name() == legacy.Document.Filename || !strings.HasPrefix(entry.Name(), "简历_") || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		document, err := r.readDocument(entry.Name())
		if err != nil {
			// One damaged file must not prevent access to the other resumes.
			r.failure("list_document_skipped", apperr.ContentLoad, err)
			continue
		}
		items = append(items, document.Document)
	}
	sort.SliceStable(items[1:], func(i, j int) bool {
		return items[i+1].ModifiedAt > items[j+1].ModifiedAt
	})
	return items, nil
}

func (r *Resume) OpenDocument(filename string) (ResumeDocument, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	document, err := r.readDocument(filename)
	if err != nil {
		return document, r.failure("open_document", apperr.ContentLoad, err)
	}
	if err := r.rememberDocument(filename); err != nil {
		return document, r.failure("open_document", apperr.ContentSave, err)
	}
	return document, nil
}

func (r *Resume) documentMarkdown(info DocumentInfo, content Content) (string, error) {
	if err := r.template.Validate(content); err != nil {
		return "", err
	}
	markdown, err := WriteMarkdown(content, r.resources.Schema)
	if err != nil {
		return "", err
	}
	if info.Company != "" || info.Role != "" || info.JD != "" {
		header, err := json.Marshal(struct {
			Company string `json:"company"`
			Role    string `json:"role"`
			JD      string `json:"jd"`
		}{info.Company, info.Role, info.JD})
		if err != nil {
			return "", err
		}
		markdown = profilePrefix + string(header) + "\n\n" + markdown
	}
	return markdown, nil
}

// CreateDocument never overwrites a version, even when company and role match.
func (r *Resume) CreateDocument(input NewDocument) (ResumeDocument, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	info := DocumentInfo{Company: strings.TrimSpace(input.Company), Role: strings.TrimSpace(input.Role), JD: strings.TrimSpace(input.JD)}
	company, role := filenamePart(info.Company), filenamePart(info.Role)
	if role == "" {
		return ResumeDocument{}, r.failure("create_document", apperr.DocumentInvalid, errors.New("role is required"))
	}
	if company == "" {
		company = "未指定公司"
	}
	var content Content
	var err error
	if input.Source == "" {
		content, err = ParseContent(r.resources.DocumentMD)
	} else {
		var source ResumeDocument
		source, err = r.readDocument(input.Source)
		content = source.Content
	}
	if err != nil {
		return ResumeDocument{}, r.failure("create_document", apperr.ContentLoad, err)
	}
	markdown, err := r.documentMarkdown(info, content)
	if err != nil {
		return ResumeDocument{}, r.failure("create_document", apperr.ContentSave, err)
	}
	dir := filepath.Dir(r.paths.ContentFile)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return ResumeDocument{}, r.failure("create_document", apperr.ContentSave, err)
	}
	for number := 1; ; number++ {
		base := "简历_" + company + "_" + role
		if number > 1 {
			base += fmt.Sprintf("_%02d", number)
		}
		info.Filename = base + ".md"
		path := filepath.Join(dir, info.Filename)
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if os.IsExist(err) {
			continue
		}
		if err == nil {
			_, err = file.WriteString(markdown)
			closeErr := file.Close()
			if err == nil {
				err = closeErr
			}
			if err != nil {
				_ = os.Remove(path)
			}
		}
		if err != nil {
			return ResumeDocument{}, r.failure("create_document", apperr.ContentSave, err)
		}
		r.logger.Info("document_created")
		if err := r.rememberDocument(info.Filename); err != nil {
			// The new file is already durable; a preference failure must not duplicate it on retry.
			r.failure("remember_document", apperr.ContentSave, err)
		}
		document, err := r.readDocument(info.Filename)
		if err != nil {
			return document, r.failure("create_document", apperr.ContentLoad, err)
		}
		return document, nil
	}
}

// SaveDocument targets an explicit filename, so a delayed save cannot cross versions.
func (r *Resume) SaveDocument(filename string, content Content, jd string) (DocumentInfo, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	document, err := r.readDocument(filename)
	if err != nil {
		return DocumentInfo{}, r.failure("save_document", apperr.ContentSave, err)
	}
	document.Document.JD = jd
	markdown, err := r.documentMarkdown(document.Document, content)
	if err != nil {
		return DocumentInfo{}, r.failure("save_document", apperr.ContentSave, err)
	}
	path, _ := r.documentPath(filename)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return DocumentInfo{}, r.failure("save_document", apperr.ContentSave, err)
	}
	if err := writeDocumentFile(path, []byte(markdown)); err != nil {
		return DocumentInfo{}, r.failure("save_document", apperr.ContentSave, err)
	}
	stat, err := os.Stat(path)
	if err != nil {
		return DocumentInfo{}, r.failure("save_document", apperr.ContentSave, err)
	}
	document.Document.ModifiedAt = stat.ModTime().UTC().Format(time.RFC3339)
	r.logger.Info("content_saved")
	return document.Document, nil
}
