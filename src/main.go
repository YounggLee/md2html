package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"
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
