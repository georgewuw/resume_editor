package resume

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportsKeepEachResumeSeparate(t *testing.T) {
	root := t.TempDir()
	r, err := New(Resources{
		TemplateHTML: `<span data-f="name"></span>`,
		DocumentMD:   "@@name\n默认\n",
		Schema:       []FieldMeta{{Key: "name", Group: "personal"}},
	}, StoragePaths{ContentFile: filepath.Join(root, "简历内容.md"), OutputDir: filepath.Join(root, "output")}, func(path string) ([]byte, error) {
		html, err := os.ReadFile(path)
		return append([]byte("%PDF-"), html...), err
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	first, err := r.CreateDocument(NewDocument{Company: "公司甲", Role: "工程师"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := r.CreateDocument(NewDocument{Company: "公司甲", Role: "工程师"})
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		doc        ResumeDocument
		text, base string
	}{
		{first, "甲内容", "成品简历_公司甲_工程师"},
		{second, "乙内容", "成品简历_公司甲_工程师_02"},
	} {
		if _, err := r.OpenDocument(item.doc.Document.Filename); err != nil {
			t.Fatal(err)
		}
		item.doc.Content.Fields["name"] = []string{item.text}
		path, err := r.Export(item.doc.Document.Filename, item.doc.Content)
		if err != nil {
			t.Fatal(err)
		}
		if filepath.Base(path) != item.base+".pdf" {
			t.Fatalf("export path %q", path)
		}
	}
	first.Content.Fields["name"] = []string{"甲更新"}
	if _, err := r.Export(first.Document.Filename, first.Content); err != nil {
		t.Fatal(err)
	}
	general, err := r.OpenDocument("简历内容.md")
	if err != nil {
		t.Fatal(err)
	}
	path, err := r.Export(general.Document.Filename, general.Content)
	if err != nil || filepath.Base(path) != "成品简历.pdf" {
		t.Fatalf("general export: %q %v", path, err)
	}
	for _, item := range []struct{ base, text string }{
		{"成品简历_公司甲_工程师", "甲更新"}, {"成品简历_公司甲_工程师_02", "乙内容"}, {"成品简历", "默认"},
	} {
		for _, ext := range []string{".html", ".pdf"} {
			data, err := os.ReadFile(filepath.Join(root, "output", item.base+ext))
			if err != nil || !strings.Contains(string(data), item.text) {
				t.Fatalf("%s%s: %v, %s", item.base, ext, err, data)
			}
		}
	}
	for _, invalid := range []string{"../简历_公司甲_工程师.md", "简历_不存在.md"} {
		if _, err := r.Export(invalid, first.Content); err == nil {
			t.Fatalf("accepted invalid document %q", invalid)
		}
	}
}
