package main

import (
	"strings"
	"testing"
)

func renderHTML(t *testing.T, src string) string {
	t.Helper()
	out, _, _, err := convertMarkdown([]byte(src), ConvertOptions{LinkRewrite: true})
	if err != nil {
		t.Fatalf("convertMarkdown error: %v", err)
	}
	return out
}

func TestHeading_KoreanSlugID(t *testing.T) {
	html := renderHTML(t, "## API 설계\n### 1. 개요")
	if !strings.Contains(html, `id="api-설계"`) {
		t.Errorf("missing korean slug id on h2:\n%s", html)
	}
	if !strings.Contains(html, `id="1-개요"`) {
		t.Errorf("missing slug id on h3:\n%s", html)
	}
}

func TestHeading_CollisionSuffix(t *testing.T) {
	html := renderHTML(t, "## 개요\n## 개요\n## 개요")
	if !strings.Contains(html, `id="개요"`) ||
		!strings.Contains(html, `id="개요-2"`) ||
		!strings.Contains(html, `id="개요-3"`) {
		t.Errorf("collision suffix not applied:\n%s", html)
	}
}
