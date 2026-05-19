package main

import "testing"

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
