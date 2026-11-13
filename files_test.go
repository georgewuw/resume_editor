package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"resume-editor/internal/resume"
)

func TestJDResumeVersions(t *testing.T) {
	home := t.TempDir()
	t.Setenv("RESUME_EDITOR_HOME", home)
	svc, err := newResumeService()
	if err != nil {
		t.Fatal(err)
	}
	workspace, err := svc.Workspace()
	if err != nil || len(workspace.Documents) != 1 {
		t.Fatalf("initial workspace: %+v, %v", workspace, err)
	}
	general := workspace.Current
	general.Content.Fields["name"] = []string{"通用姓名"}
	if _, err := svc.SaveDocument(general.Document.Filename, general.Content, ""); err != nil {
		t.Fatal(err)
	}
	jd := "负责 Go 服务开发\n@@name\n```\n任职要求：三年经验"
	first, err := svc.CreateDocument(resume.NewDocument{Company: "星河/科技", Role: "高级 Go 工程师", JD: jd, Source: general.Document.Filename})
	if err != nil {
		t.Fatal(err)
	}
	if first.Document.Filename != "简历_星河_科技_高级_Go_工程师.md" {
		t.Fatal(first.Document.Filename)
	}
	if first.Content.Fields["name"][0] != "通用姓名" || first.Document.JD != jd {
		t.Fatalf("copy/metadata lost: %+v", first)
	}
	second, err := svc.CreateDocument(resume.NewDocument{Company: "星河/科技", Role: "高级 Go 工程师"})
	if err != nil {
		t.Fatal(err)
	}
	if second.Document.Filename != "简历_星河_科技_高级_Go_工程师_02.md" {
		t.Fatal(second.Document.Filename)
	}
	if second.Content.Fields["name"][0] == "通用姓名" {
		t.Fatal("template source copied the current document")
	}
	first.Content.Fields["name"] = []string{"定制姓名"}
	if _, err := svc.SaveDocument(first.Document.Filename, first.Content, jd+"\n补充要求"); err != nil {
		t.Fatal(err)
	}
	for filename, name := range map[string]string{general.Document.Filename: "通用姓名", first.Document.Filename: "定制姓名", second.Document.Filename: "【姓名】"} {
		loaded, err := svc.OpenDocument(filename)
		if err != nil || loaded.Content.Fields["name"][0] != name {
			t.Fatalf("file isolation %s: %+v %v", filename, loaded, err)
		}
	}
	if _, err := svc.OpenDocument(first.Document.Filename); err != nil {
		t.Fatal(err)
	}
	restarted, err := newResumeService()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := restarted.Workspace()
	if err != nil || restored.Current.Document.Filename != first.Document.Filename || restored.Current.Document.JD != jd+"\n补充要求" {
		t.Fatalf("restore: %+v %v", restored, err)
	}
	data, err := os.ReadFile(filepath.Join(home, first.Document.Filename))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), ":: resume-profile ") {
		t.Fatal("missing metadata header")
	}
	parsed, err := resume.ParseContent(string(data))
	if err != nil || len(parsed.Fields) != 53 || parsed.Fields["name"][0] != "定制姓名" {
		t.Fatalf("header changed resume fields: %+v %v", parsed, err)
	}
	progress, err := svc.Progress(parsed)
	if err != nil || progress.Total != 53 || progress.Filled != 1 || progress.Groups["personal"] != 1 {
		t.Fatalf("progress: %+v %v", progress, err)
	}
}

func TestDocumentFailuresPreserveSavedContent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("RESUME_EDITOR_HOME", home)
	svc, err := newResumeService()
	if err != nil {
		t.Fatal(err)
	}
	document, err := svc.CreateDocument(resume.NewDocument{Role: "产品经理"})
	if err != nil {
		t.Fatal(err)
	}
	if document.Document.Filename != "简历_未指定公司_产品经理.md" {
		t.Fatal(document.Document.Filename)
	}
	path := filepath.Join(home, document.Document.Filename)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SaveDocument(document.Document.Filename, resume.Content{Fields: map[string][]string{"name": {"incomplete"}}}, "lost"); err == nil {
		t.Fatal("invalid content accepted")
	}
	after, err := os.ReadFile(path)
	if err != nil || string(before) != string(after) {
		t.Fatalf("failed save changed data: %v", err)
	}
	for _, filename := range []string{"../简历_外部.md", `..\简历_外部.md`, "C:简历_外部.md", "其他文件.md"} {
		if _, err := svc.OpenDocument(filename); err == nil {
			t.Errorf("accepted %q", filename)
		}
		if _, err := svc.SaveDocument(filename, document.Content, ""); err == nil {
			t.Errorf("saved %q", filename)
		}
	}
	if _, err := svc.CreateDocument(resume.NewDocument{Role: " /:*? "}); err == nil {
		t.Fatal("empty sanitized role accepted")
	}
	if err := os.WriteFile(filepath.Join(home, "简历_损坏.md"), []byte("@@name\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	items, err := svc.ListDocuments()
	if err != nil || len(items) != 2 {
		t.Fatalf("damaged file blocked list: %+v %v", items, err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	workspace, err := svc.Workspace()
	if err != nil || workspace.Current.Document.Filename != "简历内容.md" {
		t.Fatalf("missing active file fallback: %+v %v", workspace, err)
	}
}
