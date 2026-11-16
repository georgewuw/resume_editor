package main

import (
	"fmt"
	"strings"
	"testing"

	"resume-editor/internal/resume"
)

func TestProjectCountsPersistAndRender(t *testing.T) {
	for count := 0; count <= 6; count++ {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			t.Setenv("RESUME_EDITOR_HOME", t.TempDir())
			svc, err := newResumeService()
			if err != nil {
				t.Fatal(err)
			}
			workspace, err := svc.Workspace()
			if err != nil {
				t.Fatal(err)
			}
			document := workspace.Current
			for i := count + 1; i <= 6; i++ {
				delete(document.Content.Fields, fmt.Sprintf("proj%dName", i))
				delete(document.Content.Fields, fmt.Sprintf("proj%dDesc", i))
			}
			if _, err := svc.SaveDocument(document.Document.Filename, document.Content, ""); err != nil {
				t.Fatal(err)
			}
			restarted, err := newResumeService()
			if err != nil {
				t.Fatal(err)
			}
			loaded, err := restarted.OpenDocument(document.Document.Filename)
			if err != nil {
				t.Fatal(err)
			}
			if got := len(loaded.Content.Fields); got != 41+count*2 {
				t.Fatalf("project count changed on reopen: %d fields", got)
			}
			output, err := restarted.Fill(loaded.Content)
			if err != nil {
				t.Fatal(err)
			}
			for i := 1; i <= 6; i++ {
				if visible := strings.Contains(output, fmt.Sprintf(`data-f="proj%dName"`, i)); visible != (i <= count) {
					t.Errorf("project %d visible=%v for count=%d", i, visible, count)
				}
			}
			if strings.Contains(output, `id="sec-project"`) != (count > 0) {
				t.Error("empty project section remains in output")
			}
			if strings.Count(output, `class="page"`) != 2 {
				t.Error("project removal changed page containers")
			}
			progress, err := restarted.Progress(loaded.Content)
			if err != nil || progress.Total != 41+count*2 {
				t.Fatalf("progress includes removed projects: %+v %v", progress, err)
			}
			copy, err := restarted.CreateDocument(resume.NewDocument{Role: "项目数量验证", Source: document.Document.Filename})
			if err != nil || len(copy.Content.Fields) != len(loaded.Content.Fields) {
				t.Fatalf("project count changed when copying document: %+v %v", copy, err)
			}
			delete(loaded.Content.Fields, "name")
			if _, err := restarted.Fill(loaded.Content); err == nil {
				t.Error("missing required personal field was accepted")
			}
		})
	}
}
