package resume

import (
	"encoding/base64"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"resume-editor/internal/apperr"
)

func TestLocalPhotoPreviewAndExport(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "output")
	if err := os.MkdirAll(output, 0o755); err != nil {
		t.Fatal(err)
	}
	photo, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/aWQAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(output, "证件照 #100%.png")
	if err := os.WriteFile(path, photo, 0o600); err != nil {
		t.Fatal(err)
	}
	want := "data:image/png;base64," + base64.StdEncoding.EncodeToString(photo)
	var exported string
	r, err := New(Resources{TemplateHTML: `<span data-f="photo"></span>`, DocumentMD: "@@photo\n【证件照】\n", Schema: []FieldMeta{{Key: "photo", Group: "personal"}}}, StoragePaths{OutputDir: output, ContentFile: filepath.Join(root, "resume.md")}, func(path string) ([]byte, error) {
		data, err := os.ReadFile(path)
		exported = string(data)
		return []byte("%PDF-test"), err
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	urlPath := filepath.ToSlash(path)
	if filepath.VolumeName(path) != "" {
		urlPath = "/" + urlPath
	}
	fileURL := (&url.URL{Scheme: "file", Path: urlPath}).String()
	for _, value := range []string{"![证件照](" + path + ")", "![证件照](./证件照 #100%.png)", "![证件照](" + fileURL + ")", path, "./证件照 #100%.png"} {
		content := Content{Fields: map[string][]string{"photo": {value}}}
		html, err := r.Fill(content)
		if err != nil || !strings.Contains(html, want) {
			t.Fatalf("Fill(%q): %v, %s", value, err, html)
		}
		if content.Fields["photo"][0] != value {
			t.Fatal("photo field was modified")
		}
		if _, err := r.Export("resume.md", content); err != nil || !strings.Contains(exported, want) {
			t.Fatalf("Export(%q): %v, %s", value, err, exported)
		}
		if _, err := r.Save(content); err != nil {
			t.Fatal(err)
		}
		saved, err := os.ReadFile(filepath.Join(root, "resume.md"))
		if err != nil || !strings.Contains(string(saved), value) || strings.Contains(string(saved), "base64,") {
			t.Fatalf("saved content changed: %v, %s", err, saved)
		}
	}
}

func TestLocalPhotoFailureIsReported(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "invalid.png"), []byte("not an image"), 0o600); err != nil {
		t.Fatal(err)
	}
	r, err := New(Resources{TemplateHTML: `<span data-f="photo"></span>`, DocumentMD: "@@photo\n【证件照】\n"}, StoragePaths{OutputDir: root, ContentFile: filepath.Join(root, "resume.md")}, func(string) ([]byte, error) { return nil, errors.New("renderer should not be called") }, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"missing.jpg", "invalid.png"} {
		content := Content{Fields: map[string][]string{"photo": {"![证件照](" + path + ")"}}}
		for name, action := range map[string]func() error{
			"preview": func() error { _, err := r.Fill(content); return err },
			"export":  func() error { _, err := r.Export("resume.md", content); return err },
		} {
			t.Run(name, func(t *testing.T) {
				var appErr *apperr.Error
				if err := action(); !errors.As(err, &appErr) || appErr.Code != apperr.Code("PHOTO_READ_FAILED") {
					t.Fatalf("expected photo error, got %v", err)
				}
			})
		}
	}
}

func TestNonLocalPhotoAndPlaceholderRemainUnchanged(t *testing.T) {
	r, err := New(Resources{TemplateHTML: `<span data-f="photo"></span>`}, StoragePaths{OutputDir: t.TempDir()}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ value, want string }{
		{"【证件照】", "【证件照】"},
		{"![证件照](https://example.com/photo.jpg)", "https://example.com/photo.jpg"},
		{"![证件照](data:image/png;base64,AAAA)", "data:image/png;base64,AAAA"},
	} {
		value := test.value
		html, err := r.Fill(Content{Fields: map[string][]string{"photo": {value}}})
		if err != nil || !strings.Contains(html, test.want) {
			t.Fatalf("Fill(%q): %v, %s", value, err, html)
		}
	}
}
