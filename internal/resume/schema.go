package resume

import (
	"encoding/json"
	"fmt"

	"resume-editor/internal/apperr"
)

// FieldMeta is the generated field contract used by persistence and the UI.
type FieldMeta struct {
	Index       int    `json:"index"`
	Key         string `json:"key"`
	Group       string `json:"group"`
	GroupTitle  string `json:"groupTitle"`
	GroupEn     string `json:"groupEn"`
	Label       string `json:"label"`
	Type        string `json:"type"`
	Multiline   bool   `json:"multiline"`
	Hint        string `json:"hint,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
}

// GroupMeta defines an editor section and its navigation text.
type GroupMeta struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	English     string `json:"en"`
	Description string `json:"description"`
}

// FieldConfig defines presentation metadata for a template field.
type FieldConfig struct {
	Group       string `json:"group"`
	Label       string `json:"label"`
	Type        string `json:"type"`
	Hint        string `json:"hint,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
}

// EditorConfig is the human-edited source for all field and group metadata.
type EditorConfig struct {
	Groups []GroupMeta            `json:"groups"`
	Fields map[string]FieldConfig `json:"fields"`
}

// ParseSchema reads the generated field contract embedded in the desktop app.
func ParseSchema(source string) ([]FieldMeta, error) {
	var schema []FieldMeta
	if err := json.Unmarshal([]byte(source), &schema); err != nil {
		return nil, apperr.Wrap(apperr.SchemaInvalid, err)
	}
	return schema, nil
}

// BuildSchema keeps template order while resolving metadata from explicit config.
func BuildSchema(config EditorConfig, template *Template) ([]FieldMeta, error) {
	groups := make(map[string]GroupMeta)
	for _, group := range config.Groups {
		groups[group.ID] = group
	}
	var schema []FieldMeta
	for index, field := range template.Fields() {
		meta, ok := config.Fields[field.Key]
		if !ok {
			return nil, apperr.Wrap(apperr.SchemaInvalid, fmt.Errorf("missing field metadata: %s", field.Key))
		}
		group, ok := groups[meta.Group]
		if !ok {
			return nil, apperr.Wrap(apperr.SchemaInvalid, fmt.Errorf("unknown group for %s: %s", field.Key, meta.Group))
		}
		if meta.Type != "text" && meta.Type != "textarea" && meta.Type != "lines" {
			return nil, apperr.Wrap(apperr.SchemaInvalid, fmt.Errorf("unknown control type for %s: %s", field.Key, meta.Type))
		}
		schema = append(schema, FieldMeta{
			Index: index + 1, Key: field.Key, Group: group.ID,
			GroupTitle: group.Title, GroupEn: group.English,
			Label: meta.Label, Type: meta.Type, Multiline: field.Multiline,
			Hint: meta.Hint, Placeholder: meta.Placeholder,
		})
	}
	if len(config.Fields) != len(schema) {
		return nil, apperr.Wrap(apperr.SchemaInvalid, fmt.Errorf("field metadata count does not match template"))
	}
	return schema, nil
}
