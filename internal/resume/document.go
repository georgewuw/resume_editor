// Package resume 提供简历内容的解析、写回、校验逻辑。
//
// 目录划分参考 MyGit：业务领域下沉到 internal/，不依赖窗口、webview 或
// 具体的资源路径；模板、默认文档、字段定义与输出路径统一由 app 层通过
// Assets 注入，因此可以直接用纯字符串做单元测试。
package resume

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// FieldMeta 描述一个可替换字段，供写回 Markdown 时保持模块结构。
type FieldMeta struct {
	Index      int    `json:"index"`
	Key        string `json:"key"`
	Group      string `json:"group"`
	GroupTitle string `json:"groupTitle"`
	GroupEn    string `json:"groupEn"`
	Label      string `json:"label"`
	Type       string `json:"type"`
	Multiline  bool   `json:"multiline"`
}

// Content 是编辑器与 Go 之间传递的全部内容。
type Content struct {
	Fields map[string][]string `json:"fields"`
}

// Assets 是领域层需要的全部外部数据与路径，由 app 层的 go:embed 资源构造。
type Assets struct {
	TemplateHTML string
	DocumentMD   string
	Schema       []FieldMeta
	ContentFile  string
	OutputDir    string
}

// ParseSchema 解析由 scripts/gen-schema.mjs 生成的字段定义。
func ParseSchema(s string) ([]FieldMeta, error) {
	var out []FieldMeta
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil, err
	}
	return out, nil
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
				return out, errors.New("内容文档出现空的字段 ID")
			}
			if _, dup := out.Fields[current]; dup {
				return out, fmt.Errorf("字段 ID 重复：%s", current)
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
			return out, fmt.Errorf("字段 %s 没有内容", k)
		}
	}
	return out, nil
}

// Validate 检查字段集合是否与模板一致（缺字段或多字段都报错）。
func Validate(c Content, tpl string) error {
	var keys []string
	rest := tpl
	for {
		i := strings.Index(rest, `data-f="`)
		if i < 0 {
			break
		}
		s := i + len(`data-f="`)
		e := strings.Index(rest[s:], `"`)
		if e < 0 {
			break
		}
		keys = append(keys, rest[s:s+e])
		rest = rest[s+e+1:]
	}
	want := map[string]bool{}
	for _, k := range keys {
		want[k] = true
	}
	var missing, extra []string
	for k := range want {
		if _, ok := c.Fields[k]; !ok {
			missing = append(missing, k)
		}
	}
	for k := range c.Fields {
		if !want[k] {
			extra = append(extra, k)
		}
	}
	if len(missing) > 0 || len(extra) > 0 {
		return fmt.Errorf("字段不一致：缺失 %v，多余 %v", missing, extra)
	}
	return nil
}

// WriteMarkdown 按 schema 顺序重建纯数据 Markdown：只包含模块标题与
// 「@@字段ID + 值行」，不含任何说明、注释或分隔线。编辑器保存后写出的
// 文件与 docs/简历内容.md 格式一致，可再次被直接读取、编辑。
func WriteMarkdown(c Content, schema []FieldMeta) (string, error) {
	if len(schema) == 0 {
		return "", errors.New("字段定义为空")
	}
	numbers := []string{"一", "二", "三", "四", "五", "六", "七", "八"}
	groupOrder := []string{}
	groupInfo := map[string]FieldMeta{}
	for _, f := range schema {
		if _, ok := groupInfo[f.Group]; !ok {
			groupOrder = append(groupOrder, f.Group)
			groupInfo[f.Group] = f
		}
	}

	var b strings.Builder
	last := ""
	for _, f := range schema {
		if f.Group != last {
			last = f.Group
			idx := 0
			for i, g := range groupOrder {
				if g == f.Group {
					idx = i
					break
				}
			}
			num := fmt.Sprint(idx + 1)
			if idx < len(numbers) {
				num = numbers[idx]
			}
			info := groupInfo[f.Group]
			b.WriteString("## 模块" + num + " · " + info.GroupTitle +
				"（" + info.GroupEn + "）\n\n")
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
