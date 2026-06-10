package inspect

import (
	"testing"

	"github.com/colinleefish/mypast/internal/uri"
)

func TestClassifySessionPath(t *testing.T) {
	const sess = "0622aba4-f3ae-4bcd-837a-52e170a88c2b"
	const atom = "019eb28e-e351-7812-91a6-328b1f77a4bd"

	cases := []struct {
		name string
		raw  string
		want sessionPathKind
	}{
		{"session root", uri.BuildSession(sess), sessionPathSession},
		{"turn", uri.BuildSessionTurn(sess, 0), sessionPathTurn},
		{"atom", uri.BuildSessionAtom(sess, atom), sessionPathAtom},
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
	// Hand-built shapes that uri.Parse rejects but the dispatcher must still
	// treat as unsupported rather than mis-route.
	cases := []uri.URI{
		{Scope: uri.ScopeSessions},                                                      // no session id
		{Scope: uri.ScopeSessions, Segments: []string{"s", "turns"}},                    // turn index missing
		{Scope: uri.ScopeSessions, Segments: []string{"s", "bogus", "x"}},               // unknown sub-collection
		{Scope: uri.ScopeSessions, Segments: []string{"s", "atoms", "a", "extra"}},      // too deep
	}
	for i, u := range cases {
		if got := classifySessionPath(u); got != sessionPathUnknown {
			t.Fatalf("case %d: classifySessionPath = %d want %d (unknown)", i, got, sessionPathUnknown)
		}
	}
}
