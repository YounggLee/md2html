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

func TestCodeBlock_MermaidBranch(t *testing.T) {
	html := renderHTML(t, "```mermaid\nflowchart LR\n  A --> B\n```")
	if !strings.Contains(html, `<pre class="mermaid">`) {
		t.Errorf("mermaid block must use <pre class=\"mermaid\">:\n%s", html)
	}
	if strings.Contains(html, `language-mermaid`) {
		t.Errorf("mermaid block must not have language-* class:\n%s", html)
	}
	if !strings.Contains(html, "flowchart LR") {
		t.Errorf("mermaid content missing:\n%s", html)
	}
}

func TestCodeBlock_RegularLanguage(t *testing.T) {
	html := renderHTML(t, "```go\nfmt.Println(\"hi\")\n```")
	if !strings.Contains(html, `<pre><code class="language-go">`) {
		t.Errorf("regular code block must use <pre><code class=\"language-X\">:\n%s", html)
	}
	if !strings.Contains(html, `fmt.Println(&#34;hi&#34;)`) && !strings.Contains(html, `fmt.Println(&quot;hi&quot;)`) {
		t.Errorf("content must be HTML-escaped:\n%s", html)
	}
}

func TestCodeBlock_NoLanguage(t *testing.T) {
	html := renderHTML(t, "```\nplain text\n```")
	if !strings.Contains(html, `<pre><code class="language-plaintext">`) {
		t.Errorf("missing lang must default to plaintext:\n%s", html)
	}
}

func TestCodeBlock_MermaidEscape(t *testing.T) {
	html := renderHTML(t, "```mermaid\nA-->>B: Mono<Foo>\n```")
	// inside <pre class="mermaid">, < and > must be HTML-escaped so the browser
	// returns the original source via textContent.
	if !strings.Contains(html, "Mono&lt;Foo&gt;") {
		t.Errorf("mermaid content must be HTML-escaped:\n%s", html)
	}
}
