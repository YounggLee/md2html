package main

import (
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
)

func parseAST(src string) (goldmark.Markdown, []byte, interface{}) {
	md := goldmark.New()
	source := []byte(src)
	doc := md.Parser().Parse(text.NewReader(source))
	return md, source, doc
}

func TestExtractTitle_FirstH1(t *testing.T) {
	_, source, doc := parseAST("# Hello World\n## sub\nbody")
	got := extractTitle(doc.(rootNode), source, "fallback")
	if got != "Hello World" {
		t.Errorf("got %q, want %q", got, "Hello World")
	}
}

func TestExtractTitle_FallbackToFilename(t *testing.T) {
	_, source, doc := parseAST("no heading here\njust body")
	got := extractTitle(doc.(rootNode), source, "fallback")
	if got != "fallback" {
		t.Errorf("got %q, want %q", got, "fallback")
	}
}

func TestBuildTOC_H2H3Only(t *testing.T) {
	src := "# Title\n## 개요\n### 배경\n### 목표\n## 수용 기준\n#### 너무 깊음\n"
	_, source, doc := parseAST(src)
	resolver := NewSlugResolver()
	toc := buildTOC(doc.(rootNode), source, resolver)
	want := []string{"개요", "배경", "목표", "수용 기준"}
	for _, w := range want {
		if !strings.Contains(toc, w) {
			t.Errorf("TOC missing %q\nTOC:\n%s", w, toc)
		}
	}
	if strings.Contains(toc, "너무 깊음") {
		t.Errorf("TOC must not include h4: %s", toc)
	}
	if strings.Contains(toc, "Title") {
		t.Errorf("TOC must not include h1: %s", toc)
	}
}

func TestBuildTOC_Empty(t *testing.T) {
	_, source, doc := parseAST("just text, no heading")
	toc := buildTOC(doc.(rootNode), source, NewSlugResolver())
	if !strings.Contains(toc, "(목차 없음)") {
		t.Errorf("expected '(목차 없음)' fallback, got: %s", toc)
	}
}
