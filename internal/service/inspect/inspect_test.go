package inspect

import (
	"testing"

	"github.com/colinleefish/rmb/internal/uri"
)

func TestClassifySessionPath(t *testing.T) {
	const sess = "0622aba4-f3ae-4bcd-837a-52e170a88c2b"

	cases := []struct {
		name string
		raw  string
		want sessionPathKind
	}{
		{"session root", uri.BuildSession(sess), sessionPathSession},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			u, err := uri.Parse(c.raw)
			if err != nil {
				t.Fatalf("parse %q: %v", c.raw, err)
			}
			if got := classifySessionPath(u); got != c.want {
				t.Fatalf("classifySessionPath(%q) = %d want %d", c.raw, got, c.want)
			}
		})
	}
}

func TestClassifySessionPath_unknown(t *testing.T) {
	cases := []uri.URI{
		{Scope: uri.ScopeSessions},
		{Scope: uri.ScopeSessions, Segments: []string{"s", "turns"}},
		{Scope: uri.ScopeSessions, Segments: []string{"s", "bogus", "x"}},
		{Scope: uri.ScopeSessions, Segments: []string{"s", "turns", "0"}},
		{Scope: uri.ScopeSessions, Segments: []string{"s", "atoms", "a", "extra"}},
	}
	for i, u := range cases {
		if got := classifySessionPath(u); got != sessionPathUnknown {
			t.Fatalf("case %d: classifySessionPath = %d want %d (unknown)", i, got, sessionPathUnknown)
		}
	}
}
