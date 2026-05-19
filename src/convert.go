package main

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

type RenderInput struct {
	Source    []byte
	Filename  string // used for title fallback (basename without .md)
	ShellHTML string // raw shell HTML containing {{TITLE}} {{TOC}} {{CONTENT}}
	Options   ConvertOptions
}

// RenderDocument runs the full pipeline: markdown parse → custom render → shell
// placeholder substitution → leftover placeholder check.
func RenderDocument(in RenderInput) (string, error) {
	for _, p := range []string{"{{TITLE}}", "{{TOC}}", "{{CONTENT}}"} {
		if !strings.Contains(in.ShellHTML, p) {
			return "", fmt.Errorf("shell missing placeholder: %s", p)
		}
	}

	body, title, toc, err := convertMarkdown(in.Source, in.Options)
	if err != nil {
		return "", err
	}
	if title == "" {
		title = filenameTitle(in.Filename)
	}

	out := in.ShellHTML
	out = strings.ReplaceAll(out, "{{TITLE}}", htmlAttrEscape(title))
	out = strings.ReplaceAll(out, "{{TOC}}", toc)
	out = strings.ReplaceAll(out, "{{CONTENT}}", body)

	if loc := leftoverPlaceholderRE.FindString(out); loc != "" {
		return "", fmt.Errorf("leftover placeholder in shell: %s", loc)
	}
	return out, nil
}

// filenameTitle strips directory and .md extension.
func filenameTitle(path string) string {
	base := filepath.Base(path)
	if i := strings.LastIndex(base, "."); i > 0 {
		base = base[:i]
	}
	return base
}

var leftoverPlaceholderRE = regexp.MustCompile(`\{\{[A-Z_]+\}\}`)
