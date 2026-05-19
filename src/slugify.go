package main

import (
	"fmt"
	"strings"
	"unicode"
)

// slugify converts a heading's plain text into an anchor ID per PR #99 guide:
// - lowercase
// - replace spaces, "/", ".", ":" with "-"
// - drop other non-word characters (Korean is kept)
// - collapse consecutive "-" and trim leading/trailing "-"
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == ' ' || r == '/' || r == '.' || r == ':':
			b.WriteRune('-')
		case r == '-' || r == '_':
			b.WriteRune('-')
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		default:
			// drop other non-word characters (·, —, (, ), !, , etc.)
		}
	}
	// collapse runs of "-"
	out := b.String()
	for strings.Contains(out, "--") {
		out = strings.ReplaceAll(out, "--", "-")
	}
	return strings.Trim(out, "-")
}

// SlugResolver tracks slugs already used in a single document and resolves
// collisions by appending -2, -3, ...
type SlugResolver struct {
	seen map[string]int
}

func NewSlugResolver() *SlugResolver {
	return &SlugResolver{seen: map[string]int{}}
}

func (r *SlugResolver) Resolve(text string) string {
	base := slugify(text)
	n := r.seen[base]
	r.seen[base] = n + 1
	if n == 0 {
		return base
	}
	return fmt.Sprintf("%s-%d", base, n+1)
}
