package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	"github.com/egoist/mygo"

	"resume-editor/internal/apperr"
	"resume-editor/internal/logging"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() (err error) {
	metadata, err := readMetadata()
	if err != nil {
		return err
	}
	logger, logFile, err := logging.Open(filepath.Join(dataDir(), "logs"), metadata.Name, metadata.Version)
	if err != nil {
		return err
	}
	defer logFile.Close()
	slog.SetDefault(logger)
	logger.Info("application_started", "platform", runtime.GOOS, "arch", runtime.GOARCH)
	defer func() {
		if err != nil {
			logging.Failure(logger, "startup", err)
		}
		logger.Info("application_stopped")
	}()
	service, err := newResumeService()
	if err != nil {
		return err
	}
	mygo.Bind(service)
	mygo.App.WhenReady(func() {
		width, height := initialWindowSize()
		mygo.NewWindow(customTitleBar(mygo.WindowOptions{
			Title: "简历编辑器", URL: "/", Width: width, Height: height,
			MinWidth: minInitialWidth, MinHeight: minInitialHeight, StateKey: "main",
		}))
	})
	if err := mygo.App.Run(); err != nil {
		return apperr.Wrap(apperr.Unexpected, err)
	}
	return nil
}
