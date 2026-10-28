package main

import (
	"os"
	"path"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// readmes lists README.md and its translations. Each translation must keep the
// same structure so that a section added in one language is not missed in
// the others.
var readmes = []string{"README.md", "README.zh-TW.md", "README.ja.md", "README.th.md"}

var relativeLink = regexp.MustCompile(`\]\(([^)#:]+)(#[^)]*)?\)`)

type docShape struct {
	headings  []int
	tableRows int
	fences    int
	links     []string // relative link targets other than the READMEs
}

func shapeOf(t *testing.T, name string) docShape {
	t.Helper()
	b, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	var s docShape
	inFence := false
	for _, line := range strings.Split(string(b), "\n") {
		switch {
		case strings.HasPrefix(line, "```"):
			s.fences++
			inFence = !inFence
		case inFence:
		case strings.HasPrefix(line, "#"):
			s.headings = append(s.headings, len(line)-len(strings.TrimLeft(line, "#")))
		case strings.HasPrefix(line, "|"):
			s.tableRows++
		}
	}
	for _, m := range relativeLink.FindAllStringSubmatch(string(b), -1) {
		target := m[1]
		if _, err := os.Stat(path.Clean(target)); err != nil {
			t.Errorf("%s links to missing file %s", name, target)
		}
		if !slices.Contains(readmes, target) && !slices.Contains(s.links, target) {
			s.links = append(s.links, target)
		}
	}
	slices.Sort(s.links)
	return s
}

func TestReadmeTranslationsMatchStructure(t *testing.T) {
	want := shapeOf(t, readmes[0])
	for _, name := range readmes {
		got := shapeOf(t, name)
		if !slices.Equal(got.headings, want.headings) {
			t.Errorf("%s heading levels = %v, want %v as in README.md", name, got.headings, want.headings)
		}
		if got.tableRows != want.tableRows || got.fences != want.fences {
			t.Errorf("%s has %d table rows and %d code fences, want %d and %d as in README.md",
				name, got.tableRows, got.fences, want.tableRows, want.fences)
		}
		if !slices.Equal(got.links, want.links) {
			t.Errorf("%s relative links = %v, want %v as in README.md", name, got.links, want.links)
		}
		b, _ := os.ReadFile(name)
		for _, label := range []string{"English", "繁體中文", "日本語", "ไทย"} {
			if !strings.Contains(strings.SplitN(string(b), "\n", 4)[2], label) {
				t.Errorf("%s language bar misses %s", name, label)
			}
		}
	}
}
