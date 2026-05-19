package main

import "testing"

func TestSlugify(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"API 설계", "api-설계"},
		{"1. 개요", "1-개요"},
		{"Hello, World!", "hello-world"},
		{"foo / bar : baz . qux", "foo-bar-baz-qux"},
		{"  공백  양옆  ", "공백-양옆"},
		{"a—b·c(d)", "abcd"},
		{"", ""},
		{"한글", "한글"},
		{"Mixed 한글 English", "mixed-한글-english"},
	}
	for _, tt := range tests {
		got := slugify(tt.in)
		if got != tt.want {
			t.Errorf("slugify(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestSlugifyCollisionResolver(t *testing.T) {
	r := NewSlugResolver()
	if got := r.Resolve("api 설계"); got != "api-설계" {
		t.Errorf("first call: got %q, want %q", got, "api-설계")
	}
	if got := r.Resolve("api 설계"); got != "api-설계-2" {
		t.Errorf("second call: got %q, want %q", got, "api-설계-2")
	}
	if got := r.Resolve("api 설계"); got != "api-설계-3" {
		t.Errorf("third call: got %q, want %q", got, "api-설계-3")
	}
}
