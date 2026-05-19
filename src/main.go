package main

import (
	_ "embed"
	"fmt"
	"os"
)

//go:embed shell.html
var embeddedShell string

func main() {
	fmt.Fprintln(os.Stderr, "md2html: not implemented yet")
	_ = embeddedShell
	os.Exit(1)
}
