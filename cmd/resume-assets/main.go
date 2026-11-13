// resume-assets generates derived resume data using the runtime Go rules.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"resume-editor/internal/apperr"
	"resume-editor/internal/resume"
)

type frontendData struct {
	Metadata resume.Metadata        `json:"metadata"`
	Groups   []resume.GroupMeta     `json:"groups"`
	Schema   []resume.FieldMeta     `json:"schema"`
	Initial  resume.Content         `json:"initial"`
	Errors   map[apperr.Code]string `json:"errors"`
}

func main() {
	if err := run(); err != nil {
		err = apperr.Wrap(apperr.Unexpected, err)
		fmt.Fprintln(os.Stderr, err)
		if appErr, ok := err.(*apperr.Error); ok && appErr.Unwrap() != nil {
			fmt.Fprintln(os.Stderr, appErr.Unwrap())
		}
		os.Exit(1)
	}
}

func run() error {
	root, err := os.Getwd()
	if err != nil {
		return apperr.Wrap(apperr.ConfigInvalid, err)
	}
	read := func(name string) ([]byte, error) {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			return nil, apperr.Wrap(apperr.AssetRead, err)
		}
		return data, nil
	}
	templateSource, err := read("docs/简历模板.html")
	if err != nil {
		return err
	}
	document, err := read("docs/简历内容.md")
	if err != nil {
		return err
	}
	configSource, err := read("docs/editor.json")
	if err != nil {
		return err
	}
	metadataSource, err := read("mygo.json")
	if err != nil {
		return err
	}
	var config resume.EditorConfig
	if err := json.Unmarshal(configSource, &config); err != nil {
		return apperr.Wrap(apperr.SchemaInvalid, err)
	}
	var metadata resume.Metadata
	if err := json.Unmarshal(metadataSource, &metadata); err != nil {
		return apperr.Wrap(apperr.ConfigInvalid, err)
	}
	template, err := resume.CompileTemplate(string(templateSource))
	if err != nil {
		return err
	}
	schema, err := resume.BuildSchema(config, template)
	if err != nil {
		return err
	}
	content, err := resume.ParseContent(string(document))
	if err != nil {
		return err
	}
	if err := template.Validate(content); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(root, "assets"), 0o755); err != nil {
		return apperr.Wrap(apperr.AssetWrite, err)
	}
	schemaJSON, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return err
	}
	metadataJSON, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}
	files := map[string][]byte{
		"简历模板.html":   templateSource,
		"简历内容.md":     document,
		"schema.json": append(schemaJSON, '\n'),
		"app.json":    append(metadataJSON, '\n'),
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(root, "assets", name), data, 0o644); err != nil {
			return apperr.Wrap(apperr.AssetWrite, err)
		}
	}
	return json.NewEncoder(os.Stdout).Encode(frontendData{
		Metadata: metadata, Groups: config.Groups, Schema: schema,
		Initial: content, Errors: apperr.Catalog(),
	})
}
