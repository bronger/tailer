package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func tail(f *os.File) {
	_, err := io.Copy(os.Stdout, f)
	check(err)
}

func main() {
	filePath, err := filepath.Abs(os.Args[1])
	check(err)
	tailWatcher, err := fsnotify.NewWatcher()
	check(err)
	err = tailWatcher.Add(filePath)
	check(err)
	stopWatcher, err := fsnotify.NewWatcher()
	check(err)
	stopFilePath := filepath.Join(filePath + ".stop")
	err = stopWatcher.Add(filepath.Dir(stopFilePath))
	check(err)
	f, err := os.Open(filePath)
	check(err)
	if _, err = os.Stat(stopFilePath); err == nil {
		tail(f)
		goto exit
	} else if !errors.Is(err, os.ErrNotExist) {
		check(err)
	}
	tail(f)
	for {
		select {
		case event := <-tailWatcher.Events:
			if event.Op&fsnotify.Write == fsnotify.Write {
				tail(f)
			}
		case event := <-stopWatcher.Events:
			if event.Op&fsnotify.Create == fsnotify.Create && event.Name == stopFilePath {
				tail(f)
				goto exit
			}
		case err := <-tailWatcher.Errors:
			check(err)
		case err := <-stopWatcher.Errors:
			check(err)
		}
	}
exit:
	err = f.Close()
	check(err)
}
