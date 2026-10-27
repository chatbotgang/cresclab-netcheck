package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	// Windows writes CRLF line endings for Notepad, so compare line contents only.
	if got, want := strings.ReplaceAll(string(b), "\r\n", "\n"), "\xef\xbb\xbffirst\nsecond\n"; got != want {
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

func TestHTTPProbeReportsUnreachableHost(t *testing.T) {
	if testing.Short() {
		t.Skip("waits for a network timeout")
	}
	// 192.0.2.1 is reserved for documentation (RFC 5737), so nothing answers.
	start := time.Now()
	detail, status := httpProbe("https://example.invalid/", probeOptions{dialIP: "192.0.2.1"})
	if !strings.HasPrefix(status, "FAIL") {
		t.Errorf("status = %q, want FAIL", status)
	}
	if !strings.Contains(detail, "tls=-") {
		t.Errorf("detail = %q, want tls=- to show the stalled step", detail)
	}
	if elapsed := time.Since(start); elapsed > requestTimeout+2*time.Second {
		t.Errorf("took %s, want at most about %s", elapsed, requestTimeout)
	}
}
