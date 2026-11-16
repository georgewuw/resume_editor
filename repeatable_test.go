package main

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"resume-editor/internal/resume"
)

func TestAdvantageAndEducationCountsPersistAndRender(t *testing.T) {
	for advantages := 0; advantages <= 6; advantages++ {
		for education := 0; education <= 3; education++ {
			t.Run(fmt.Sprintf("advantages_%d_education_%d", advantages, education), func(t *testing.T) {
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
				fields := document.Content.Fields
				for key := range fields {
					if strings.HasPrefix(key, "adv") || strings.HasPrefix(key, "edu") {
						delete(fields, key)
					}
				}
				fields["brief"] = []string{"个人简介独立保留"}
				for i := 1; i <= advantages; i++ {
					fields[fmt.Sprintf("adv%dLead", i)] = []string{fmt.Sprintf("优势 %d", i)}
					fields[fmt.Sprintf("adv%dText", i)] = []string{fmt.Sprintf("优势描述 %d", i)}
				}
				for i := 1; i <= education; i++ {
					for _, suffix := range []string{"Period", "School", "Major", "Honor", "Achievement"} {
						fields[fmt.Sprintf("edu%d%s", i, suffix)] = []string{fmt.Sprintf("教育 %d %s", i, suffix)}
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
				if !reflect.DeepEqual(fields, loaded.Content.Fields) {
					t.Fatal("saving changed advantage or education values or restored deleted fields")
				}
				output, err := restarted.Fill(loaded.Content)
				if err != nil {
					t.Fatal(err)
				}
				for i := 1; i <= 6; i++ {
					if strings.Contains(output, fmt.Sprintf(`data-f="adv%dLead"`, i)) != (i <= advantages) {
						t.Errorf("advantage %d has incorrect visibility", i)
					}
				}
				for i := 1; i <= 3; i++ {
					if strings.Contains(output, fmt.Sprintf(`data-f="edu%dSchool"`, i)) != (i <= education) {
						t.Errorf("education %d has incorrect visibility", i)
					}
				}
				if !strings.Contains(output, "个人简介独立保留") {
					t.Error("removing advantage items removed the brief")
				}
				if strings.Contains(output, `<span class="sec-title">教育背景`) != (education > 0) {
					t.Error("empty education section remains in output")
				}
				progress, err := restarted.Progress(loaded.Content)
				if err != nil || progress.Total != 32+advantages*2+education*5 {
					t.Fatalf("progress includes deleted fields: %+v %v", progress, err)
				}
				copied, err := restarted.CreateDocument(resume.NewDocument{Role: "条目数量验证", Source: document.Document.Filename})
				if err != nil || !reflect.DeepEqual(copied.Content.Fields, fields) {
					t.Fatalf("copying changed repeated entries: %v", err)
				}
			})
		}
	}
}
