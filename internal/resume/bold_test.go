package resume

import (
	"strings"
	"testing"
)

func TestDescriptionBoldRendering(t *testing.T) {
	template := `<span data-f="brief"></span><span data-f="name"></span><span data-f="work1Desc"></span><div data-f="certLines" data-multiline="1"></div>`
	fields := map[string][]string{
		"brief":     {"提升**30%**，覆盖**10个**团队"},
		"name":      {"**名字**"},
		"work1Desc": {"**<script>alert(1)</script>** & **未闭合"},
		"certLines": {"通过**考试**", "节省**50%**"},
	}
	out, err := FillTemplate(template, fields)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`提升<strong>30%</strong>，覆盖<strong>10个</strong>团队`,
		`<span data-f="name">**名字**</span>`,
		`<strong>&lt;script&gt;alert(1)&lt;/script&gt;</strong> &amp; **未闭合`,
		`<span class="ln">通过<strong>考试</strong></span><span class="ln">节省<strong>50%</strong></span>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %s", want, out)
		}
	}
	if strings.Contains(out, "<script>") {
		t.Fatal("user HTML was not escaped")
	}
}

func TestDescriptionBoldMarkdownRoundTrip(t *testing.T) {
	content := Content{Fields: map[string][]string{"brief": {"增长**30%**", "覆盖**10个**团队"}}}
	markdown, err := WriteMarkdown(content, []FieldMeta{{Key: "brief", Group: "advantage"}})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseContent(markdown)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(parsed.Fields["brief"], "\n") != strings.Join(content.Fields["brief"], "\n") {
		t.Fatalf("markers lost: %q", markdown)
	}
}
