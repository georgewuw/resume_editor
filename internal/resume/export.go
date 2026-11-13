package resume

import (
	"bytes"
	"os"
	"path/filepath"

	"resume-editor/internal/apperr"
)

// PDFRenderer prints a local HTML document using the desktop's WebView.
// The source directory is also the base for relative image paths.
type PDFRenderer func(sourcePath string) ([]byte, error)

// writePrintSource 把已渲染的 HTML 写入 PDF 打印的工作目录。
// 它是导出 PDF 的中间产物，不再作为独立的「导出 HTML」对外提供。
func writePrintSource(outputDir, html string) (string, error) {
	outputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	p := filepath.Join(outputDir, "成品简历.html")
	if err := os.WriteFile(p, []byte(html), 0o644); err != nil {
		return "", err
	}
	return p, nil
}

// exportPDF owns output persistence; the desktop adapter owns native rendering.
func exportPDF(outputDir, html string, render PDFRenderer) (string, error) {
	htmlPath, err := writePrintSource(outputDir, html)
	if err != nil {
		return "", err
	}
	data, err := render(htmlPath)
	if err != nil {
		return "", err
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		return "", apperr.Wrap(apperr.ExportMissing, nil)
	}
	pdf := filepath.Join(filepath.Dir(htmlPath), "成品简历.pdf")
	if err := os.WriteFile(pdf, data, 0o644); err != nil {
		return "", err
	}
	return pdf, nil
}
