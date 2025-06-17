package main

import (
	"bufio"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	tbr_errors "gitlab.com/bronger/tools/errors"
	tbr_logging "gitlab.com/bronger/tools/logging"
	"go4.org/must"
)

func tail(logger tbr_logging.Logger, f *os.File) {
	written, err := io.Copy(os.Stdout, f)
	tbr_errors.ExitOnExpectedError(err, "Could not tail file", 9)
	logger.Debug("Synchronised", "bytes", written)
}

const logPath = "/var/log/tailer.log"

func main() {
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o777)
	tbr_errors.ExitOnExpectedError(err, "Could not open log file", 13, "path", logPath)
	defer must.Close(file)
	logWriter := bufio.NewWriter(file)
	defer logWriter.Flush()
	tbr_logging.Init(logWriter, slog.LevelDebug)
	logger := slog.Default()
	filePath, err := filepath.Abs(os.Args[1])
	tbr_errors.ExitOnExpectedError(err, "File not found", 3, "path", filePath)
	tailWatcher, err := fsnotify.NewWatcher()
	tbr_errors.ExitOnExpectedError(err, "Could not set create notifier for original file", 4)
	err = tailWatcher.Add(filePath)
	tbr_errors.ExitOnExpectedError(err, "Could not set set up notifier for original file", 5)
	stopWatcher, err := fsnotify.NewWatcher()
	tbr_errors.ExitOnExpectedError(err, "Could not set create notifier for stop file", 6)
	stopFilePath := filepath.Join(filePath + ".stop")
	err = stopWatcher.Add(filepath.Dir(stopFilePath))
	tbr_errors.ExitOnExpectedError(err, "Could not set set up notifier for stop file", 7)
	f, err := os.Open(filePath)
	tbr_errors.ExitOnExpectedError(err, "Could not open original file", 8, "path", filePath)
	defer must.Close(f)
	if _, err = os.Stat(stopFilePath); err == nil {
		tail(logger, f)
		return
	} else if !errors.Is(err, os.ErrNotExist) {
		tbr_errors.ExitWithExpectedError("Could not access stop file", 10, "error", err, "path", stopFilePath)
	}
	tail(logger, f)
	var latestSync time.Time
	for {
		select {
		case event := <-tailWatcher.Events:
			if true || time.Since(latestSync) > time.Second && event.Op&fsnotify.Write == fsnotify.Write {
				tail(logger, f)
				latestSync = time.Now()
			}
		case event := <-stopWatcher.Events:
			if event.Op&fsnotify.Create == fsnotify.Create && event.Name == stopFilePath {
				tail(logger, f)
				return
			}
		case err := <-tailWatcher.Errors:
			tbr_errors.ExitOnExpectedError(err, "Error with watching original file", 11)
		case err := <-stopWatcher.Errors:
			tbr_errors.ExitOnExpectedError(err, "Error with watching stop file", 12)
		}
	}
}
