// Package apperr owns the stable error codes shared by Go and the frontend.
package apperr

import (
	"encoding/json"
	"errors"
)

// Code is a stable identifier shared by services, UI notices and logs.
type Code string

const (
	Unexpected      Code = "APP_UNEXPECTED"
	AssetRead       Code = "ASSET_READ_FAILED"
	AssetWrite      Code = "ASSET_WRITE_FAILED"
	ConfigInvalid   Code = "CONFIG_INVALID"
	DocumentInvalid Code = "DOCUMENT_INVALID"
	SchemaInvalid   Code = "SCHEMA_INVALID"
	TemplateInvalid Code = "TEMPLATE_INVALID"
	FieldsMismatch  Code = "FIELDS_MISMATCH"
	ContentLoad     Code = "CONTENT_LOAD_FAILED"
	ContentSave     Code = "CONTENT_SAVE_FAILED"
	PreviewFailed   Code = "PREVIEW_FAILED"
	ExportFailed    Code = "EXPORT_FAILED"
	ExportMissing   Code = "EXPORT_OUTPUT_MISSING"
	StorageRead     Code = "STORAGE_READ_FAILED"
	StorageWrite    Code = "STORAGE_WRITE_FAILED"
	BridgeFailed    Code = "BRIDGE_CALL_FAILED"
	LogInit         Code = "LOG_INIT_FAILED"
	LogWrite        Code = "LOG_WRITE_FAILED"
)

var messages = map[Code]string{
	Unexpected:      "发生未预期的错误",
	AssetRead:       "读取应用资源失败",
	AssetWrite:      "写入生成资源失败",
	ConfigInvalid:   "应用配置无效",
	DocumentInvalid: "简历内容格式无效",
	SchemaInvalid:   "字段定义无效",
	TemplateInvalid: "简历模板无效",
	FieldsMismatch:  "内容字段与模板不一致",
	ContentLoad:     "读取简历内容失败",
	ContentSave:     "保存简历内容失败",
	PreviewFailed:   "生成预览失败",
	ExportFailed:    "导出 PDF 失败",
	ExportMissing:   "未生成有效的 PDF 文档",
	StorageRead:     "读取界面偏好失败",
	StorageWrite:    "保存界面偏好失败",
	BridgeFailed:    "调用桌面服务失败",
	LogInit:         "初始化日志失败",
	LogWrite:        "写入日志失败",
}

// Error exposes only a code and a safe message across the mygo bridge.
// The original cause stays in Go for local diagnostics and errors.Is/As.
type Error struct {
	Code    Code   `json:"code"`
	Message string `json:"message"`
	cause   error
}

// Error uses JSON because mygo v0.9 transports Go errors as strings.
func (e *Error) Error() string {
	b, _ := json.Marshal(e)
	return string(b)
}

// Unwrap retains the original diagnostic for local callers.
func (e *Error) Unwrap() error { return e.cause }

// Known reports whether a code is defined in the application catalog.
func Known(code Code) bool {
	_, ok := messages[code]
	return ok
}

// Wrap preserves an existing application error's specific code.
func Wrap(code Code, cause error) *Error {
	var existing *Error
	if errors.As(cause, &existing) {
		return existing
	}
	message, ok := messages[code]
	if !ok {
		code, message = Unexpected, messages[Unexpected]
	}
	return &Error{Code: code, Message: message, cause: cause}
}

// Catalog is copied into the generated frontend data; codes are defined here only.
func Catalog() map[Code]string {
	out := make(map[Code]string, len(messages))
	for code, message := range messages {
		out[code] = message
	}
	return out
}
