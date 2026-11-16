package main

import (
	"context"
	"fmt"
	"time"

	"github.com/egoist/mygo"
)

// renderPDF runs from the bound service's worker goroutine. Native window
// creation and navigation stay on the UI thread; waiting never blocks it.
func renderPDF(sourcePath string) ([]byte, error) {
	if !mygo.App.IsReady() {
		return nil, fmt.Errorf("PDF rendering requires the desktop event loop")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	loaded := make(chan error, 1)
	complete := func(err error) {
		select {
		case loaded <- err:
		default:
		}
	}
	var window *mygo.Window
	mygo.RunOnMain(func() {
		window = mygo.NewWindow(mygo.WindowOptions{
			Title: "PDF 导出", Hidden: true, SkipTaskbar: true,
			Width: 794, Height: 1123, UseContentSize: true,
			Page: mygo.PageOptions{Untrusted: true, Session: mygo.NewEphemeralSession()},
		})
		page := window.Page()
		page.OnDidFinishLoad(func() { complete(nil) })
		page.OnDidFailLoad(func(err *mygo.LoadError) { complete(err) })
		window.OnClosed(func() { complete(fmt.Errorf("PDF rendering window closed")) })
		if err := page.LoadFile(sourcePath); err != nil {
			complete(err)
		}
	})
	if window == nil || window.IsDestroyed() {
		return nil, fmt.Errorf("PDF rendering window unavailable")
	}
	defer window.Destroy()
	select {
	case err := <-loaded:
		if err != nil {
			return nil, fmt.Errorf("load PDF source: %w", err)
		}
	case <-ctx.Done():
		return nil, fmt.Errorf("load PDF source: %w", ctx.Err())
	}

	page := window.Page()
	if _, err := page.EvalContext(ctx, `
		await document.fonts.ready;
		await Promise.all(Array.from(document.images, image => image.decode()));
		await window.resumeLayoutReady;
		return document.body.offsetHeight;
	`); err != nil {
		return nil, fmt.Errorf("prepare PDF layout: %w", err)
	}
	return page.PrintToPDF(mygo.PDFOptions{
		// Use exact millimetres: rounding A4 to inches can add a blank page.
		PageSize: mygo.PageSize{Width: 210 / 25.4, Height: 297 / 25.4},
		Margins:  &mygo.Margins{}, Background: true,
	})
}
