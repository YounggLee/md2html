package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseArgs_Defaults(t *testing.T) {
	o, err := parseArgs([]string{"foo.md"})
	if err != nil {
		t.Fatal(err)
	}
	if o.outDir != "/tmp/md2html" {
		t.Errorf("outDir = %q, want /tmp/md2html", o.outDir)
	}
	if o.noOpen || o.noLinkRewrite {
		t.Errorf("bool defaults wrong: %+v", o)
	}
	if len(o.files) != 1 || o.files[0] != "foo.md" {
		t.Errorf("files = %v", o.files)
	}
}

func TestParseArgs_AllFlags(t *testing.T) {
	o, err := parseArgs([]string{
		"--no-open", "--out-dir", "/tmp/x",
		"--shell", "/path/shell.html", "--no-link-rewrite",
		"a.md", "b.md",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !o.noOpen || !o.noLinkRewrite || o.outDir != "/tmp/x" || o.shellPath != "/path/shell.html" {
		t.Errorf("flags not parsed: %+v", o)
	}
	if len(o.files) != 2 {
		t.Errorf("files = %v", o.files)
	}
}

func TestParseArgs_NoFiles(t *testing.T) {
	_, err := parseArgs([]string{})
	if err == nil {
		t.Fatal("expected error for no files")
	}
}

func TestProcessFiles_WritesOutput(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "foo.md")
	if err := os.WriteFile(in, []byte("# title\nbody"), 0o644); err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(dir, "out")
	written, err := processFiles([]string{in}, outDir, "<html>{{TITLE}}|{{TOC}}|{{CONTENT}}</html>", ConvertOptions{LinkRewrite: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(written) != 1 {
		t.Fatalf("expected 1 file, got %v", written)
	}
	wantPath := filepath.Join(outDir, "foo.html")
	if written[0] != wantPath {
		t.Errorf("got %q, want %q", written[0], wantPath)
	}
	b, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "title") {
		t.Errorf("output missing title: %s", b)
	}
}

func TestProcessFiles_FilenameCollision(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	b := filepath.Join(dir, "b")
	os.MkdirAll(a, 0o755)
	os.MkdirAll(b, 0o755)
	for _, p := range []string{filepath.Join(a, "spec.md"), filepath.Join(b, "spec.md")} {
		if err := os.WriteFile(p, []byte("# x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	outDir := filepath.Join(dir, "out")
	written, err := processFiles(
		[]string{filepath.Join(a, "spec.md"), filepath.Join(b, "spec.md")},
		outDir, "<html>{{TITLE}}{{TOC}}{{CONTENT}}</html>", ConvertOptions{LinkRewrite: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(written[0], "/spec.html") {
		t.Errorf("first: %s", written[0])
	}
	if !strings.HasSuffix(written[1], "/spec-2.html") {
		t.Errorf("second should have -2 suffix: %s", written[1])
	}
}

func TestProcessFiles_MissingFileContinues(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good.md")
	os.WriteFile(good, []byte("# g"), 0o644)
	written, err := processFiles(
		[]string{filepath.Join(dir, "missing.md"), good},
		filepath.Join(dir, "out"), "<html>{{TITLE}}{{TOC}}{{CONTENT}}</html>",
		ConvertOptions{LinkRewrite: true})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if len(written) != 1 {
		t.Fatalf("expected 1 successful write, got %v", written)
	}
}
