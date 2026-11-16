package resume

import (
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"resume-editor/internal/apperr"
)

// embedLocalPhoto works on a copy so saved content keeps its original path.
func embedLocalPhoto(fields map[string][]string, baseDir string) (map[string][]string, error) {
	value := strings.TrimSpace(strings.Join(fields["photo"], "\n"))
	source, alt := value, "证件照"
	if match := imageSyntax.FindStringSubmatch(value); match != nil {
		source, alt = match[2], match[1]
		if alt == "" {
			alt = "证件照"
		}
	} else if !looksLikePhotoPath(value) {
		return fields, nil
	}
	path, local, err := photoPath(source, baseDir)
	if err != nil {
		return nil, apperr.Wrap(apperr.PhotoRead, errors.New("invalid photo path"))
	}
	if !local {
		return fields, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, apperr.Wrap(apperr.PhotoRead, errors.New("cannot read photo file"))
	}
	contentType := http.DetectContentType(data)
	if !strings.HasPrefix(contentType, "image/") {
		return nil, apperr.Wrap(apperr.PhotoRead, errors.New("photo file is not a supported image"))
	}
	copyFields := make(map[string][]string, len(fields))
	for key, values := range fields {
		copyFields[key] = values
	}
	copyFields["photo"] = []string{"![" + alt + "](data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(data) + ")"}
	return copyFields, nil
}

func looksLikePhotoPath(value string) bool {
	if strings.ContainsAny(value, `/\`) {
		return true
	}
	ext := strings.ToLower(filepath.Ext(value))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".ico":
		return true
	}
	return strings.HasPrefix(strings.ToLower(value), "data:image/")
}

func photoPath(source, baseDir string) (string, bool, error) {
	if isWindowsAbsolutePhotoPath(source) || filepath.IsAbs(source) || strings.HasPrefix(source, `\\`) {
		return source, true, nil
	}
	// Treat ordinary relative values as filesystem paths, including literal % and #.
	if !strings.Contains(source, ":") {
		return filepath.Join(baseDir, filepath.FromSlash(strings.ReplaceAll(source, `\`, "/"))), true, nil
	}
	parsed, err := url.Parse(source)
	if err != nil {
		return "", false, err
	}
	if parsed.Scheme != "" {
		if !strings.EqualFold(parsed.Scheme, "file") {
			return "", false, nil
		}
		path := parsed.Path
		if parsed.Host != "" && !strings.EqualFold(parsed.Host, "localhost") {
			path = "//" + parsed.Host + path
		}
		if strings.HasPrefix(path, "/") && isWindowsAbsolutePhotoPath(path[1:]) {
			path = path[1:]
		}
		return filepath.FromSlash(path), true, nil
	}
	// Relative paths are filesystem paths, not URLs; preserve spaces and # characters.
	return filepath.Join(baseDir, filepath.FromSlash(strings.ReplaceAll(source, `\`, "/"))), true, nil
}

func isWindowsAbsolutePhotoPath(path string) bool {
	return len(path) >= 3 && ((path[0] >= 'A' && path[0] <= 'Z') || (path[0] >= 'a' && path[0] <= 'z')) && path[1] == ':' && (path[2] == '\\' || path[2] == '/')
}
