package resume

import (
	"fmt"
	"html"
	"regexp"
	"sort"
	"strings"

	"resume-editor/internal/apperr"
)

type renderKind uint8

const (
	renderText renderKind = iota
	renderLines
	renderImage
)

// templatePart is literal HTML or a typed field slot.
type templatePart struct {
	html string
	key  string
	kind renderKind
}

// Template holds compiled HTML segments and field slots for repeated rendering.
type Template struct {
	parts []templatePart
}

// TemplateField describes a unique field in template order.
type TemplateField struct {
	Key       string
	Multiline bool
}

var imageSyntax = regexp.MustCompile(`^!\[(.*?)\]\((.+?)\)$`)

// CompileTemplate is the single data-f scanner used by validation, rendering
// and asset generation. A field occupies the contents of a paired HTML tag.
func CompileTemplate(source string) (*Template, error) {
	rest := strings.Replace(source, `<body data-layout="double">`, `<body data-layout="double" class="filled">`, 1)
	template := &Template{}
	for {
		index := strings.Index(rest, `data-f="`)
		if index < 0 {
			template.parts = append(template.parts, templatePart{html: rest})
			return template, nil
		}
		keyStart := index + len(`data-f="`)
		keyEnd := strings.Index(rest[keyStart:], `"`)
		if keyEnd < 0 {
			return nil, apperr.Wrap(apperr.TemplateInvalid, fmt.Errorf("unclosed data-f attribute"))
		}
		key := rest[keyStart : keyStart+keyEnd]
		openStart := strings.LastIndex(rest[:index], "<")
		openEnd := strings.Index(rest[index:], ">")
		if openStart < 0 || openEnd < 0 {
			return nil, apperr.Wrap(apperr.TemplateInvalid, fmt.Errorf("incomplete placeholder: %s", key))
		}
		openEnd += index
		attributes := rest[openStart+1 : openEnd]
		tag := strings.Fields(attributes)[0]
		closeTag := "</" + tag + ">"
		closeIndex := strings.Index(rest[openEnd+1:], closeTag)
		if closeIndex < 0 {
			return nil, apperr.Wrap(apperr.TemplateInvalid, fmt.Errorf("missing %s for %s", closeTag, key))
		}
		kind := renderText
		if strings.Contains(attributes, "data-multiline") {
			kind = renderLines
		} else if strings.Contains(attributes, `data-render="image"`) || key == "photo" {
			kind = renderImage
		}
		template.parts = append(template.parts,
			templatePart{html: rest[:openEnd+1]},
			templatePart{key: key, kind: kind},
		)
		rest = rest[openEnd+1+closeIndex:]
	}
}

// Fields lists each field once, in template order.
func (t *Template) Fields() []TemplateField {
	var fields []TemplateField
	seen := make(map[string]bool)
	for _, part := range t.parts {
		if part.key != "" && !seen[part.key] {
			fields = append(fields, TemplateField{Key: part.key, Multiline: part.kind == renderLines})
			seen[part.key] = true
		}
	}
	return fields
}

// Validate checks that content and template declare the same field set.
func (t *Template) Validate(content Content) error {
	want := make(map[string]bool)
	var missing, extra []string
	for _, field := range t.Fields() {
		want[field.Key] = true
		if _, ok := content.Fields[field.Key]; !ok {
			missing = append(missing, field.Key)
		}
	}
	for key := range content.Fields {
		if !want[key] {
			extra = append(extra, key)
		}
	}
	if len(missing) > 0 || len(extra) > 0 {
		sort.Strings(extra)
		return apperr.Wrap(apperr.FieldsMismatch, fmt.Errorf("missing %v; extra %v", missing, extra))
	}
	return nil
}

// Fill escapes field values and inserts them into the compiled HTML.
func (t *Template) Fill(fields map[string][]string) string {
	var output strings.Builder
	for _, part := range t.parts {
		output.WriteString(part.html)
		if part.key != "" {
			output.WriteString(renderField(part.kind, fields[part.key]))
		}
	}
	return output.String()
}

func renderField(kind renderKind, lines []string) string {
	if kind == renderImage {
		value := strings.TrimSpace(strings.Join(lines, "\n"))
		if match := imageSyntax.FindStringSubmatch(value); match != nil {
			alt := match[1]
			if alt == "" {
				alt = "证件照"
			}
			return `<img src="` + html.EscapeString(match[2]) + `" alt="` + html.EscapeString(alt) + `">`
		}
		return `<span class="photo-tip">` + html.EscapeString(value) + `</span>`
	}
	parts := make([]string, 0, len(lines))
	for _, line := range lines {
		value := html.EscapeString(line)
		if kind == renderLines {
			value = `<span class="ln">` + value + `</span>`
		}
		parts = append(parts, value)
	}
	separator := "<br>"
	if kind == renderLines {
		separator = ""
	}
	return strings.Join(parts, separator)
}

// Validate checks a document against an uncompiled template.
func Validate(content Content, source string) error {
	template, err := CompileTemplate(source)
	if err != nil {
		return err
	}
	return template.Validate(content)
}

// FillTemplate compiles and fills HTML for callers that render only once.
func FillTemplate(source string, fields map[string][]string) (string, error) {
	template, err := CompileTemplate(source)
	if err != nil {
		return "", err
	}
	return template.Fill(fields), nil
}
