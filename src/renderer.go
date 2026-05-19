package main

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
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
			util.Prioritized(&paragraphRenderer{}, 100),
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
	lang := strings.ToLower(string(cb.Language(source)))
	body := readNodeText(cb, source)

	if lang == "mermaid" {
		fmt.Fprintf(w, `<pre class="mermaid">%s</pre>`+"\n", escapeHTML(body))
		return ast.WalkSkipChildren, nil
	}
	lang = sanitizeLang(lang)
	fmt.Fprintf(w, `<pre><code class="language-%s">%s</code></pre>`+"\n", lang, escapeHTML(body))
	return ast.WalkSkipChildren, nil
}

// sanitizeLang restricts the fence info string to [a-z0-9-]. Anything else
// (empty, non-ASCII, or attempted attribute breakout) becomes "plaintext".
// Defense in depth: the lang value lands inside a class attribute, so we
// cannot trust raw user input here.
func sanitizeLang(s string) string {
	if s == "" {
		return "plaintext"
	}
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
			return "plaintext"
		}
	}
	return s
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

func (r *taskListRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindList, r.renderList)
	reg.Register(ast.KindListItem, r.renderListItem)
	reg.Register(extast.KindTaskCheckBox, r.renderTaskCheckBox)
}

func (r *taskListRenderer) renderList(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	list := node.(*ast.List)
	tag := "ul"
	if list.IsOrdered() {
		tag = "ol"
	}
	if entering {
		if listContainsTask(list) {
			fmt.Fprintf(w, `<%s class="contains-task-list">`+"\n", tag)
		} else {
			fmt.Fprintf(w, `<%s>`+"\n", tag)
		}
	} else {
		fmt.Fprintf(w, "</%s>\n", tag)
	}
	return ast.WalkContinue, nil
}

func (r *taskListRenderer) renderListItem(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	li := node.(*ast.ListItem)
	if entering {
		if listItemIsTask(li) {
			fmt.Fprint(w, `<li class="task-list-item">`)
		} else {
			fmt.Fprint(w, `<li>`)
		}
	} else {
		fmt.Fprint(w, "</li>\n")
	}
	return ast.WalkContinue, nil
}

func (r *taskListRenderer) renderTaskCheckBox(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	cb := node.(*extast.TaskCheckBox)
	if cb.IsChecked {
		fmt.Fprint(w, `<input type="checkbox" checked disabled> `)
	} else {
		fmt.Fprint(w, `<input type="checkbox" disabled> `)
	}
	return ast.WalkContinue, nil
}

// paragraphRenderer strips the <p> wrapper when the paragraph is the direct
// child of a task-list-item. goldmark promotes paragraphs in loose lists, so
// `- [ ] foo` between blank lines would otherwise render as
// `<li class="task-list-item"><p><input ...> foo</p></li>`, contradicting the
// design rule "task list items have no <p> wrapper".
type paragraphRenderer struct{}

func (r *paragraphRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindParagraph, r.render)
}

func (r *paragraphRenderer) render(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if parent, ok := node.Parent().(*ast.ListItem); ok && listItemIsTask(parent) {
		return ast.WalkContinue, nil
	}
	if entering {
		fmt.Fprint(w, "<p>")
	} else {
		fmt.Fprint(w, "</p>\n")
	}
	return ast.WalkContinue, nil
}

func listItemIsTask(li *ast.ListItem) bool {
	for child := li.FirstChild(); child != nil; child = child.NextSibling() {
		if _, ok := child.(*ast.TextBlock); !ok {
			if _, ok := child.(*ast.Paragraph); !ok {
				continue
			}
		}
		for inline := child.FirstChild(); inline != nil; inline = inline.NextSibling() {
			if _, ok := inline.(*extast.TaskCheckBox); ok {
				return true
			}
		}
	}
	return false
}

func listContainsTask(list *ast.List) bool {
	for item := list.FirstChild(); item != nil; item = item.NextSibling() {
		if li, ok := item.(*ast.ListItem); ok && listItemIsTask(li) {
			return true
		}
	}
	return false
}

type linkRenderer struct{ rewrite bool }

func (r *linkRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindLink, r.renderLink)
	reg.Register(ast.KindAutoLink, r.renderAutoLink)
}

func (r *linkRenderer) renderLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	l := node.(*ast.Link)
	if entering {
		dest := string(l.Destination)
		if r.rewrite {
			dest = rewriteMdToHtml(dest)
		}
		fmt.Fprintf(w, `<a href="%s"`, htmlAttrEscape(dest))
		if len(l.Title) > 0 {
			fmt.Fprintf(w, ` title="%s"`, htmlAttrEscape(string(l.Title)))
		}
		fmt.Fprint(w, `>`)
	} else {
		fmt.Fprint(w, `</a>`)
	}
	return ast.WalkContinue, nil
}

func (r *linkRenderer) renderAutoLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	a := node.(*ast.AutoLink)
	url := string(a.URL(source))
	label := string(a.Label(source))
	if a.AutoLinkType == ast.AutoLinkEmail && !strings.HasPrefix(strings.ToLower(url), "mailto:") {
		url = "mailto:" + url
	}
	fmt.Fprintf(w, `<a href="%s">%s</a>`, htmlAttrEscape(url), escapeHTML(label))
	return ast.WalkSkipChildren, nil
}

// rewriteMdToHtml replaces .md with .html in relative links while preserving
// fragments and query strings. Absolute (http://, https://, //, mailto:, tel:)
// and non-.md paths are returned unchanged.
func rewriteMdToHtml(dest string) string {
	// protocol / scheme guards
	lower := strings.ToLower(dest)
	if strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "//") ||
		strings.HasPrefix(lower, "mailto:") ||
		strings.HasPrefix(lower, "tel:") {
		return dest
	}

	// split path | fragment | query
	path := dest
	rest := ""
	for _, sep := range []string{"#", "?"} {
		if i := strings.Index(path, sep); i >= 0 {
			rest = path[i:] + rest
			path = path[:i]
		}
	}
	if strings.HasSuffix(strings.ToLower(path), ".md") {
		path = path[:len(path)-3] + ".html"
	}
	return path + rest
}

func htmlAttrEscape(s string) string {
	r := strings.NewReplacer(
		`&`, "&amp;",
		`"`, "&quot;",
		`<`, "&lt;",
		`>`, "&gt;",
	)
	return r.Replace(s)
}
