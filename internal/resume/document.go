// Package resume provides resume document rules and application services.
package resume

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"resume-editor/internal/apperr"
)

// Content is the document passed between the editor and Go.
type Content struct {
	Fields map[string][]string `json:"fields"`
}

var projectField = regexp.MustCompile(`^(proj[1-9][0-9]*)(Name|Desc)$`)
var workField = regexp.MustCompile(`^(work[1-5])(Period|Company|Dept|Role|Desc)$`)
var advantageField = regexp.MustCompile(`^(adv[1-6])(Lead|Text)$`)
var educationField = regexp.MustCompile(`^(edu[1-3])(Period|School|Major|Honor|Achievement)$`)

// Optional field groups are omitted as a whole when their repeated item is deleted.
func optionalFieldGroup(key string) []string {
	if match := projectField.FindStringSubmatch(key); match != nil {
		return []string{match[1] + "Name", match[1] + "Desc"}
	}
	if match := workField.FindStringSubmatch(key); match != nil {
		prefix := match[1]
		return []string{prefix + "Period", prefix + "Company", prefix + "Dept", prefix + "Role", prefix + "Desc"}
	}
	if match := advantageField.FindStringSubmatch(key); match != nil {
		return []string{match[1] + "Lead", match[1] + "Text"}
	}
	if match := educationField.FindStringSubmatch(key); match != nil {
		prefix := match[1]
		return []string{prefix + "Period", prefix + "School", prefix + "Major", prefix + "Honor", prefix + "Achievement"}
	}
	return nil
}

// ParseContent 按内容文档的字段锚点规则解析：
// 跳过代码块、模块标题、--- 分隔线、:: 注释行与空行。
func ParseContent(src string) (Content, error) {
	out := Content{Fields: map[string][]string{}}
	current := ""
	inFence := false
	for _, raw := range strings.Split(src, "\n") {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if strings.HasPrefix(line, "@@") {
			current = strings.TrimSpace(strings.TrimPrefix(line, "@@"))
			if current == "" {
				return out, apperr.Wrap(apperr.DocumentInvalid, errors.New("empty field ID"))
			}
			if _, dup := out.Fields[current]; dup {
				return out, apperr.Wrap(apperr.DocumentInvalid, fmt.Errorf("duplicate field ID: %s", current))
			}
			out.Fields[current] = []string{}
			continue
		}
		if strings.HasPrefix(line, "## ") || line == "---" {
			current = ""
			continue
		}
		if strings.HasPrefix(line, "::") || line == "" {
			continue
		}
		if current != "" {
			out.Fields[current] = append(out.Fields[current], line)
		}
	}
	for k, v := range out.Fields {
		if len(v) == 0 {
			return out, apperr.Wrap(apperr.DocumentInvalid, fmt.Errorf("empty field: %s", k))
		}
	}
	return out, nil
}

// WriteMarkdown 按 schema 顺序重建纯数据 Markdown：只包含模块标题与
// 「@@字段ID + 值行」，不含任何说明、注释或分隔线。编辑器保存后写出的
// 文件与 docs/简历内容.md 格式一致，可再次被直接读取、编辑。
func WriteMarkdown(c Content, schema []FieldMeta) (string, error) {
	if len(schema) == 0 {
		return "", apperr.Wrap(apperr.SchemaInvalid, errors.New("empty schema"))
	}
	numbers := []string{"一", "二", "三", "四", "五", "六", "七", "八"}
	groupNumbers := make(map[string]int)
	for _, field := range schema {
		if _, ok := groupNumbers[field.Group]; !ok {
			groupNumbers[field.Group] = len(groupNumbers)
		}
	}

	var b strings.Builder
	last := ""
	for _, f := range schema {
		if _, exists := c.Fields[f.Key]; !exists && len(optionalFieldGroup(f.Key)) > 0 {
			continue
		}
		if f.Group != last {
			last = f.Group
			idx := groupNumbers[f.Group]
			num := fmt.Sprint(idx + 1)
			if idx < len(numbers) {
				num = numbers[idx]
			}
			b.WriteString("## 模块" + num + " · " + f.GroupTitle +
				"（" + f.GroupEn + "）\n\n")
		}
		b.WriteString("@@" + f.Key + "\n")
		vals := c.Fields[f.Key]
		if len(vals) == 0 {
			vals = []string{"【待填写】"}
		}
		for _, v := range vals {
			if strings.TrimSpace(v) != "" {
				b.WriteString(v + "\n")
			}
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n") + "\n", nil
}
