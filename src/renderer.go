package main

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

type ConvertOptions struct {
	LinkRewrite bool
}

// convertMarkdown parses + renders markdown bytes per PR #99 guide.
// Returns: rendered HTML body, title (first h1 or empty), TOC HTML, error.
func convertMarkdown(src []byte, opts ConvertOptions) (htmlOut, title, tocHTML string, err error) {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(),
		goldmark.WithRendererOptions(html.WithUnsafe()),
	)

	doc := md.Parser().Parse(text.NewReader(src))

	// Single pass: buildTOC walks h1/h2/h3, assigns slug IDs onto each heading
	// node (via headingIDAttr), and returns TOC HTML for h2/h3 only. The heading
	// renderer below reads the stored attribute back. This guarantees TOC links
	// and heading IDs match even when h1/h2 texts collide.
	tocHTML = buildTOC(doc, src, NewSlugResolver())
	title = extractTitle(doc, src, "")

	// Register our custom renderers on top of goldmark's default HTML renderer.
	md.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.Prioritized(&headingRenderer{}, 100),
			util.Prioritized(&codeBlockRenderer{}, 100),
			util.Prioritized(&taskListRenderer{}, 100),
			util.Prioritized(&linkRenderer{rewrite: opts.LinkRewrite}, 100),
		),
	)

	var buf bytes.Buffer
	if err := md.Renderer().Render(&buf, src, doc); err != nil {
		return "", "", "", fmt.Errorf("render: %w", err)
	}
	return buf.String(), title, tocHTML, nil
}

// --- heading ---
// headingIDAttr is defined in toc.go; the renderer here only reads it.

type headingRenderer struct{}

func (r *headingRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindHeading, r.render)
}

func (r *headingRenderer) render(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	h := node.(*ast.Heading)
	if entering {
		idAttr, _ := h.AttributeString(headingIDAttr)
		idBytes, _ := idAttr.([]byte)
		if h.Level >= 1 && h.Level <= 3 && len(idBytes) > 0 {
			fmt.Fprintf(w, `<h%d id="%s">`, h.Level, string(idBytes))
		} else {
			fmt.Fprintf(w, `<h%d>`, h.Level)
		}
	} else {
		fmt.Fprintf(w, "</h%d>\n", h.Level)
	}
	return ast.WalkContinue, nil
}

// --- placeholder stubs for next tasks (so package compiles) ---

type codeBlockRenderer struct{}

func (r *codeBlockRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindFencedCodeBlock, r.renderFenced)
	reg.Register(ast.KindCodeBlock, r.renderIndented)
}

func (r *codeBlockRenderer) renderFenced(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	cb := node.(*ast.FencedCodeBlock)
	lang := string(cb.Language(source))
	body := readNodeText(cb, source)

	if lang == "mermaid" {
		fmt.Fprintf(w, `<pre class="mermaid">%s</pre>`+"\n", escapeHTML(body))
		return ast.WalkSkipChildren, nil
	}
	if lang == "" {
		lang = "plaintext"
	}
	fmt.Fprintf(w, `<pre><code class="language-%s">%s</code></pre>`+"\n", lang, escapeHTML(body))
	return ast.WalkSkipChildren, nil
}

func (r *codeBlockRenderer) renderIndented(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	body := readNodeText(node, source)
	fmt.Fprintf(w, `<pre><code class="language-plaintext">%s</code></pre>`+"\n", escapeHTML(body))
	return ast.WalkSkipChildren, nil
}

func readNodeText(n ast.Node, source []byte) string {
	var b strings.Builder
	for i := 0; i < n.Lines().Len(); i++ {
		seg := n.Lines().At(i)
		b.Write(seg.Value(source))
	}
	return b.String()
}

type taskListRenderer struct{}

func (r *taskListRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {}

type linkRenderer struct{ rewrite bool }

func (r *linkRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {}

// --- helpers used by later renderers ---

var _ = url.PathEscape // suppress unused import until link renderer task
