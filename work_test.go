package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestWorkExperienceCountsPersistAndRender(t *testing.T) {
	for count := 0; count <= 5; count++ {
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
			for i := 3; i <= count; i++ {
				for _, suffix := range []string{"Period", "Company", "Dept", "Role", "Desc"} {
					document.Content.Fields[fmt.Sprintf("work%d%s", i, suffix)] = []string{fmt.Sprintf("工作经历 %d", i)}
				}
			}
			for i := count + 1; i <= 5; i++ {
				for _, suffix := range []string{"Period", "Company", "Dept", "Role", "Desc"} {
					delete(document.Content.Fields, fmt.Sprintf("work%d%s", i, suffix))
				}
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
			if got, want := len(loaded.Content.Fields), 43+count*5; got != want {
				t.Fatalf("work count changed on reopen: got %d fields, want %d", got, want)
			}
			output, err := restarted.Fill(loaded.Content)
			if err != nil {
				t.Fatal(err)
			}
			for i := 1; i <= 5; i++ {
				if visible := strings.Contains(output, fmt.Sprintf(`data-f="work%dPeriod"`, i)); visible != (i <= count) {
					t.Errorf("work experience %d visible=%v for count=%d", i, visible, count)
				}
			}
			if strings.Contains(output, `<span class="sec-title">工作经历`) != (count > 0) {
				t.Error("empty work experience section remains in output")
			}
			progress, err := restarted.Progress(loaded.Content)
			if err != nil || progress.Total != 43+count*5 {
				t.Fatalf("progress includes removed work fields: %+v %v", progress, err)
			}
		})
	}
}
