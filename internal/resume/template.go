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
	any  []string
	end  bool
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
var placeholderValue = regexp.MustCompile(`^【[^】\n]*】$`)
var boldSyntax = regexp.MustCompile(`\*\*([^*\n]+)\*\*`)
var formattedField = regexp.MustCompile(`^(brief|adv[1-6]Text|work[1-5]Desc|proj[1-6]Desc|edu[1-3](Honor|Achievement)|certLines)$`)

// CompileTemplate is the single data-f scanner used by validation, rendering
// and asset generation. Fields occupy paired tags; optional comment regions
// render only when at least one of their declared fields exists.
func CompileTemplate(source string) (*Template, error) {
	rest := strings.Replace(source, `<body data-layout="double">`, `<body data-layout="double" class="filled">`, 1)
	template := &Template{}
	depth := 0
	for {
		index := strings.Index(rest, `data-f="`)
		start := strings.Index(rest, "<!-- optional:")
		end := strings.Index(rest, "<!-- /optional -->")
		if start >= 0 && (index < 0 || start < index) && (end < 0 || start < end) {
			close := strings.Index(rest[start:], "-->")
			if close < 0 {
				return nil, apperr.Wrap(apperr.TemplateInvalid, fmt.Errorf("unclosed optional region"))
			}
			keys := strings.TrimSpace(rest[start+len("<!-- optional:") : start+close])
			template.parts = append(template.parts, templatePart{html: rest[:start]}, templatePart{any: strings.Split(keys, ",")})
			rest = rest[start+close+3:]
			depth++
			continue
		}
		if end >= 0 && (index < 0 || end < index) {
			if depth == 0 {
				return nil, apperr.Wrap(apperr.TemplateInvalid, fmt.Errorf("unexpected optional region end"))
			}
			template.parts = append(template.parts, templatePart{html: rest[:end]}, templatePart{end: true})
			rest = rest[end+len("<!-- /optional -->"):]
			depth--
			continue
		}
		if index < 0 {
			if depth != 0 {
				return nil, apperr.Wrap(apperr.TemplateInvalid, fmt.Errorf("unclosed optional region"))
			}
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

// Validate requires all template fields except repeated groups omitted together.
func (t *Template) Validate(content Content) error {
	want := make(map[string]bool)
	var missing, extra []string
	fields := t.Fields()
	for _, field := range fields {
		want[field.Key] = true
	}
	for _, field := range fields {
		if _, ok := content.Fields[field.Key]; !ok {
			if match := advantageField.FindStringSubmatch(field.Key); match != nil && strings.HasSuffix(field.Key, "Lead") {
				if _, hasText := content.Fields[match[1]+"Text"]; hasText {
					continue
				}
			}
			// 学术成果可省略，兼容旧文档中没有此字段的教育经历。
			if educationField.MatchString(field.Key) && strings.HasSuffix(field.Key, "Achievement") {
				continue
			}
			group := optionalFieldGroup(field.Key)
			present := false
			for _, key := range group {
				if want[key] {
					if _, exists := content.Fields[key]; exists {
						present = true
					}
				}
			}
			if len(group) > 0 && !present {
				continue
			}
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
	visible := []bool{true}
	for _, part := range t.parts {
		if part.any != nil {
			present := false
			for _, key := range part.any {
				key = strings.TrimSpace(key)
				values, ok := fields[key]
				if ok && isOptionalFieldPresent(key, values) {
					present = true
				}
			}
			visible = append(visible, visible[len(visible)-1] && present)
			continue
		}
		if part.end {
			visible = visible[:len(visible)-1]
			continue
		}
		if !visible[len(visible)-1] {
			continue
		}
		output.WriteString(part.html)
		if part.key != "" {
			output.WriteString(renderField(part.kind, fields[part.key], formattedField.MatchString(part.key)))
		}
	}
	return output.String()
}

func isOptionalFieldPresent(key string, values []string) bool {
	if advantageField.MatchString(key) && strings.HasSuffix(key, "Lead") {
		value := strings.TrimSpace(strings.Join(values, "\n"))
		return value != "" && !placeholderValue.MatchString(value)
	}
	return true
}

func renderField(kind renderKind, lines []string, bold bool) string {
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
		if bold {
			value = boldSyntax.ReplaceAllString(value, `<strong>$1</strong>`)
		}
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
