package main

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// report accumulates the report text and appends it to a file on every flush,
// so partial results survive when the program is closed early. When no
// location is writable, the text stays in memory and close reports the error.
type report struct {
	text
	flushed int
	path    string
	f       *os.File
	errs    []string
}

func newReport(name string, dirs []string) *report {
	r := &report{}
	for _, dir := range dirs {
		path := filepath.Join(dir, name)
		f, err := os.Create(path)
		if err != nil {
			r.errs = append(r.errs, err.Error())
			continue
		}
		// The byte order mark lets Notepad on older Windows detect UTF-8.
		if _, err := f.WriteString("\xef\xbb\xbf"); err != nil {
			r.errs = append(r.errs, err.Error())
			f.Close()
			continue
		}
		// Failures in earlier locations were only fallback attempts.
		r.f, r.path, r.errs = f, path, nil
		break
	}
	return r
}

func (r *report) section(title string) {
	r.printf("\n==================== %s ====================\n", title)
}

func (r *report) flush() {
	if r.f == nil {
		return
	}
	chunk := r.String()[r.flushed:]
	r.flushed += len(chunk)
	if runtime.GOOS == "windows" {
		chunk = strings.ReplaceAll(strings.ReplaceAll(chunk, "\r\n", "\n"), "\n", "\r\n")
	}
	if _, err := r.f.WriteString(chunk); err != nil {
		r.errs = append(r.errs, err.Error())
		return
	}
	_ = r.f.Sync()
}

func (r *report) close() (string, error) {
	if r.f == nil {
		return "", errors.New(strings.Join(r.errs, "; "))
	}
	if err := r.f.Close(); err != nil {
		r.errs = append(r.errs, err.Error())
	}
	if len(r.errs) > 0 {
		return r.path, errors.New(strings.Join(r.errs, "; "))
	}
	return r.path, nil
}

// outputDirs lists where the report may be saved, most convenient first. The
// Desktop can be read-only under some company policies, so later entries are
// fallbacks.
func outputDirs() []string {
	var dirs []string
	if d := desktopDir(); d != "" {
		dirs = append(dirs, d)
	}
	if exe, err := os.Executable(); err == nil {
		dirs = append(dirs, filepath.Dir(exe))
	}
	if wd, err := os.Getwd(); err == nil {
		dirs = append(dirs, wd)
	}
	return append(dirs, os.TempDir())
}
