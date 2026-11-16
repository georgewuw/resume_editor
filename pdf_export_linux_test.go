//go:build linux

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/egoist/mygo"
	"resume-editor/internal/resume"
)

const pdfE2EEnv = "RESUME_EDITOR_PDF_E2E"

func TestMain(m *testing.M) {
	if os.Getenv(pdfE2EEnv) != "1" {
		os.Exit(m.Run())
	}

	result := make(chan int, 1)
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title: "PDF test", Hidden: true, SkipTaskbar: true,
			Width: 1, Height: 1, UseContentSize: true,
			Page: mygo.PageOptions{Untrusted: true, Session: mygo.NewEphemeralSession()},
		})
		go func() {
			result <- m.Run()
			mygo.App.Quit()
		}()
	})
	if err := mygo.App.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(<-result)
}

func TestLinuxPDFExportHasTwoPages(t *testing.T) {
	if os.Getenv(pdfE2EEnv) != "1" {
		t.Skip("set " + pdfE2EEnv + "=1 in a Linux desktop session")
	}

	source, err := readAsset(tplPath)
	if err != nil {
		t.Fatal(err)
	}
	template, err := resume.CompileTemplate(source)
	if err != nil {
		t.Fatal(err)
	}
	fields := make(map[string][]string)
	for _, field := range template.Fields() {
		fields[field.Key] = []string{""}
	}
	html := template.Fill(fields)
	path := filepath.Join(t.TempDir(), "resume.html")
	if err := os.WriteFile(path, []byte(html), 0o600); err != nil {
		t.Fatal(err)
	}
	pdf, err := renderPDF(path)
	if err != nil {
		t.Fatal(err)
	}
	if pages := len(regexp.MustCompile(`/Type\s*/Page[^s]`).FindAll(pdf, -1)); pages != 2 {
		t.Fatalf("exported %d PDF pages, want 2", pages)
	}
}
