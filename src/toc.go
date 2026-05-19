package main

import (
	"strings"

	"github.com/yuin/goldmark/ast"
)

// rootNode is what goldmark's Parser().Parse() returns. We expose this alias
// so tests can pass the parser result through `interface{}` cleanly.
type rootNode = ast.Node

// headingIDAttr is the AST attribute key under which buildTOC stores the
// resolved slug ID for h1/h2/h3 nodes. The heading renderer reads it back
// during render, ensuring h2/h3 IDs match the TOC links exactly.
const headingIDAttr = "__md2html_id"

// extractTitle returns the plain text of the first H1 in the document, or
// fallback if none.
func extractTitle(doc rootNode, source []byte, fallback string) string {
	var title string
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if h, ok := n.(*ast.Heading); ok && h.Level == 1 {
			title = string(h.Text(source))
			return ast.WalkStop, nil
		}
		return ast.WalkContinue, nil
	})
	if title == "" {
		return fallback
	}
	return title
}

// buildTOC walks h1/h2/h3 in one pass with a SHARED resolver so the resolved
// IDs are stable for both the heading renderer and the TOC links. It writes
// each resolved ID onto the heading node via headingIDAttr, then emits TOC
// HTML for h2/h3 only (h1 IDs are still assigned for anchor links but are
// not listed in TOC). Returns "<p>(목차 없음)</p>" if no h2/h3 found.
func buildTOC(doc rootNode, source []byte, resolver *SlugResolver) string {
	type entry struct {
		level int
		text  string
		id    string
	}
	var entries []entry
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		h, ok := n.(*ast.Heading)
		if !ok || h.Level < 1 || h.Level > 3 {
			return ast.WalkContinue, nil
		}
		text := string(h.Text(source))
		id := resolver.Resolve(text)
		h.SetAttributeString(headingIDAttr, []byte(id))
		if h.Level == 2 || h.Level == 3 {
			entries = append(entries, entry{h.Level, text, id})
		}
		return ast.WalkContinue, nil
	})
	if len(entries) == 0 {
		return "<p>(목차 없음)</p>"
	}

	var b strings.Builder
	b.WriteString("<ul>\n")
	inH2 := false
	inH3List := false
	for _, e := range entries {
		if e.level == 2 {
			if inH3List {
				b.WriteString("    </ul>\n")
				inH3List = false
			}
			if inH2 {
				b.WriteString("  </li>\n")
			}
			b.WriteString("  <li><a href=\"#" + e.id + "\">" + escapeHTML(e.text) + "</a>")
			inH2 = true
		} else { // level 3
			if !inH3List {
				b.WriteString("\n    <ul>\n")
				inH3List = true
			}
			b.WriteString("      <li><a href=\"#" + e.id + "\">" + escapeHTML(e.text) + "</a></li>\n")
		}
	}
	if inH3List {
		b.WriteString("    </ul>\n")
	}
	if inH2 {
		b.WriteString("  </li>\n")
	}
	b.WriteString("</ul>")
	return b.String()
}

func escapeHTML(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;")
	return r.Replace(s)
}
