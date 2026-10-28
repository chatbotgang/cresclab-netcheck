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
		if _, err := f.WriteString(utf8BOM); err != nil {
			r.errs = append(r.errs, err.Error())
			_ = f.Close()
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
	if _, err := r.f.WriteString(platformNewlines(chunk)); err != nil {
		r.errs = append(r.errs, err.Error())
		return
	}
	_ = r.f.Sync()
}

// close finishes the report and rewrites it with the first placeholder
// replaced, for example to put the verdict at the top. The rewrite goes
// through a temporary file; if it fails, the progressively written report
// stays as it is, still complete apart from the replacement.
func (r *report) close(placeholder, replacement string) (string, error) {
	if r.f == nil {
		return "", errors.New(strings.Join(r.errs, "; "))
	}
	if err := r.f.Close(); err != nil {
		r.errs = append(r.errs, err.Error())
	}
	if len(r.errs) > 0 {
		return r.path, errors.New(strings.Join(r.errs, "; "))
	}
	final := utf8BOM + platformNewlines(strings.Replace(r.String(), placeholder, replacement, 1))
	tmp := r.path + ".tmp"
	if err := os.WriteFile(tmp, []byte(final), 0o644); err != nil {
		_ = os.Remove(tmp)
		return r.path, nil
	}
	if err := os.Rename(tmp, r.path); err != nil {
		_ = os.Remove(tmp)
	}
	return r.path, nil
}

// utf8BOM lets Notepad on older Windows detect UTF-8.
const utf8BOM = "\xef\xbb\xbf"

// platformNewlines uses CRLF on Windows so that Notepad shows line breaks.
func platformNewlines(s string) string {
	if runtime.GOOS != "windows" {
		return s
	}
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\n", "\r\n")
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
