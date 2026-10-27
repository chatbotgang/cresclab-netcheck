package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunCheckRecoversFromPanic(t *testing.T) {
	res := runCheck(check{"boom", func(tx *text) string {
		tx.printf("partial output\n")
		panic("unexpected")
	}})
	if !strings.Contains(res.body, "partial output") || !strings.Contains(res.body, "unexpected") {
		t.Errorf("body = %q, want partial output and the panic value", res.body)
	}
	if res.summary != "boom: check stopped unexpectedly" {
		t.Errorf("summary = %q", res.summary)
	}
}

func TestReportFallsBackToNextWritableDir(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	good := t.TempDir()
	r := newReport("report.txt", []string{missing, good})
	r.printf("first\n")
	r.flush()
	r.printf("second\n")
	r.flush()
	path, err := r.close()
	if err != nil {
		t.Fatalf("close: %v", err)
	}
	if want := filepath.Join(good, "report.txt"); path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(b), "\xef\xbb\xbffirst\nsecond\n"; got != want {
		t.Errorf("file = %q, want %q", got, want)
	}
}

func TestReportWithoutWritableDirReportsError(t *testing.T) {
	r := newReport("report.txt", []string{filepath.Join(t.TempDir(), "missing")})
	r.printf("kept in memory\n")
	r.flush()
	if _, err := r.close(); err == nil {
		t.Error("close returned nil error, want the create failure")
	}
}
