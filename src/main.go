package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed shell.html
var embeddedShell string

const defaultOutDir = "/tmp/md2html"

type cliOptions struct {
	outDir        string
	noOpen        bool
	shellPath     string
	noLinkRewrite bool
	files         []string
}

func parseArgs(argv []string) (cliOptions, error) {
	fs := flag.NewFlagSet("md2html", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	outDir := fs.String("out-dir", defaultOutDir, "output directory")
	noOpen := fs.Bool("no-open", false, "do not open the first output in a browser")
	shellPath := fs.String("shell", "", "use a custom shell file (default: embedded)")
	noLinkRewrite := fs.Bool("no-link-rewrite", false, "disable .md → .html link rewriting")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: md2html [options] <file.md> [<file.md> ...]

Convert local Markdown files to HTML using the aac-core PR #99 interactive
shell. Output goes to %s/<basename>.html and the first file is opened in the
default browser.

Options:
`, defaultOutDir)
		fs.PrintDefaults()
	}

	if err := fs.Parse(argv); err != nil {
		return cliOptions{}, err
	}
	files := fs.Args()
	if len(files) == 0 {
		fs.Usage()
		return cliOptions{}, fmt.Errorf("at least one .md file required")
	}
	return cliOptions{
		outDir:        *outDir,
		noOpen:        *noOpen,
		shellPath:     *shellPath,
		noLinkRewrite: *noLinkRewrite,
		files:         files,
	}, nil
}

func processFiles(files []string, outDir, shellHTML string, convOpts ConvertOptions) ([]string, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", outDir, err)
	}
	used := map[string]int{}
	var written []string
	var firstErr error
	for _, src := range files {
		body, err := os.ReadFile(src)
		if err != nil {
			fmt.Fprintf(os.Stderr, "md2html: %v\n", err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		body = stripBOM(body)
		base := filepath.Base(src)
		stem := strings.TrimSuffix(base, ".md")
		stem = strings.TrimSuffix(stem, ".MD")
		n := used[stem]
		used[stem] = n + 1
		outName := stem + ".html"
		if n > 0 {
			outName = fmt.Sprintf("%s-%d.html", stem, n+1)
		}
		outPath := filepath.Join(outDir, outName)

		rendered, err := RenderDocument(RenderInput{
			Source:    body,
			Filename:  base,
			ShellHTML: shellHTML,
			Options:   convOpts,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "md2html: %s: %v\n", src, err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if err := os.WriteFile(outPath, []byte(rendered), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "md2html: write %s: %v\n", outPath, err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		fmt.Fprintf(os.Stdout, "%s → %s\n", src, outPath)
		written = append(written, outPath)
	}
	return written, firstErr
}

func stripBOM(b []byte) []byte {
	if len(b) >= 3 && b[0] == 0xEF && b[1] == 0xBB && b[2] == 0xBF {
		return b[3:]
	}
	return b
}

func main() {
	opts, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "md2html:", err)
		os.Exit(2)
	}
	// placeholder until Task 15
	fmt.Println("parsed:", opts)
	_ = embeddedShell
}
