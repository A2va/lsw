package utils

import (
	"fmt"
	"os"

	"charm.land/log/v2"
)

// A callback type for the cli binary
type ProgressStatus int

const (
	ProgressStart ProgressStatus = iota
	ProgressUpdate
	ProgressDone
	ProgressError
)

type ProgressCallbackFunc func(string, ProgressStatus)

var progressCallback ProgressCallbackFunc

func SetProgressCallback(f ProgressCallbackFunc) {
	progressCallback = f
}

func GetProgressCallback() ProgressCallbackFunc {
	return progressCallback
}

func Exists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func CreateDir(dir string, perm os.FileMode) error {
	if Exists(dir) {
		return nil
	}

	if err := os.MkdirAll(dir, perm); err != nil {
		return fmt.Errorf("failed to create directory: '%s', error: '%s'", dir, err.Error())
	}

	return nil
}

func Panic(msg string, errs ...error) {
	var err error
	if len(errs) > 0 {
		err = errs[0]
	}

	if progressCallback != nil {
		progressCallback(msg, ProgressError)
	}

	// Tell the logger to skip source information about this function
	log.Helper()
	if err == nil {
		fmt.Fprintln(os.Stderr, msg)
	} else if msg == "" {
		fmt.Fprintf(os.Stderr, "err: %v\n", err)
	} else {
		fmt.Fprintf(os.Stderr, "%s, err: %v\n", msg, err)
	}

	log.Error(msg, "err", err)
	os.Exit(1)
}
