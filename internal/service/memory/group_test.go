package memory

import (
	"testing"

	"github.com/colinleefish/rmb/internal/db/pgarray"
	"github.com/colinleefish/rmb/internal/model"
	"github.com/colinleefish/rmb/internal/uri"
	"github.com/google/uuid"
)

func strPtr(s string) *string { return &s }

func TestGroupAtomsIntoBuckets(t *testing.T) {
	atoms := []model.Atom{
		{URI: "p1", Category: model.AtomCategoryProfile, Content: "lives in Beijing"},
		{URI: "p2", Category: model.AtomCategoryProfile, Content: "allergic to peanuts"},
		{URI: "pr1", Category: model.AtomCategoryPreferences, Slug: strPtr("ai-tone"), Content: "short answers"},
		{URI: "pr2", Category: model.AtomCategoryPreferences, Slug: strPtr("ai-tone"), Content: "no fluff"},
		{URI: "e1", Category: model.AtomCategoryEntities, Slug: strPtr("tesla"), Content: "drives a Tesla"},
		{URI: "ev1", Category: model.AtomCategoryEvents, Slug: strPtr("2026-05-17-postgres"), Content: "chose postgres"},
		{URI: "x1", Category: model.AtomCategoryPreferences, Content: "no slug here"},
	}

	buckets, skipped := groupAtomsIntoBuckets(atoms)
	if skipped != 1 {
		t.Fatalf("expected 1 slug-less atom skipped, got %d", skipped)
	}

	byURI := make(map[string]memoryBucket)
	for _, b := range buckets {
		byURI[b.URI] = b
	}

	profile, ok := byURI["rmb://profile"]
	if !ok || len(profile.Atoms) != 2 {
		t.Fatalf("profile bucket wrong: %+v", profile)
	}
	pref, ok := byURI["rmb://preferences/ai-tone"]
	if !ok || len(pref.Atoms) != 2 {
		t.Fatalf("preferences/ai-tone bucket wrong: %+v", pref)
	}
	if _, ok := byURI["rmb://entities/tesla"]; !ok {
		t.Fatal("missing entities/tesla bucket")
	}
	if _, ok := byURI["rmb://events/2026-05-17-postgres"]; !ok {
		t.Fatal("missing events bucket")
	}
}

func TestGroupAtomsIntoBuckets_noProfileWhenEmpty(t *testing.T) {
	atoms := []model.Atom{
		{URI: "e1", Category: model.AtomCategoryEntities, Slug: strPtr("tesla"), Content: "x"},
	}
	buckets, _ := groupAtomsIntoBuckets(atoms)
	for _, b := range buckets {
		if b.URI == "rmb://profile" {
			t.Fatal("should not create an empty profile bucket")
		}
	}
}
func TestChunkAtoms(t *testing.T) {
	atoms := make([]model.Atom, 130)
	chunks := chunkAtoms(atoms, 60)
	if len(chunks) != 3 {
		t.Fatalf("got %d chunks want 3", len(chunks))
	}
	if len(chunks[0]) != 60 || len(chunks[1]) != 60 || len(chunks[2]) != 10 {
		t.Fatalf("unexpected chunk sizes: %d %d %d", len(chunks[0]), len(chunks[1]), len(chunks[2]))
	}
	if got := chunkAtoms(atoms[:10], 60); len(got) != 1 {
		t.Fatalf("small input should be one chunk, got %d", len(got))
	}
}

func TestEqualStringSets(t *testing.T) {
	cases := []struct {
		a, b []string
		want bool
	}{
		{[]string{"x", "y"}, []string{"y", "x"}, true},      // order-insensitive
		{[]string{"x", "x", "y"}, []string{"x", "y"}, true}, // duplicate-insensitive
		{[]string{"x"}, []string{"x", "y"}, false},
		{[]string{"x"}, []string{"z"}, false},
		{nil, nil, true},
		{[]string{}, nil, true},
	}
	for i, c := range cases {
		if got := equalStringSets(c.a, c.b); got != c.want {
			t.Fatalf("case %d: equalStringSets(%v,%v)=%v want %v", i, c.a, c.b, got, c.want)
		}
	}
}

func TestBuildAtomSceneIndexAndProvenance(t *testing.T) {
	a1 := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	a2 := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	a3 := uuid.MustParse("00000000-0000-4000-8000-000000000003")
	scenes := []model.Scene{
		{ID: uuid.MustParse("00000000-0000-4000-8000-000000000010"), SourceAtoms: pgarray.UUIDArray{a1, a2}},
		{ID: uuid.MustParse("00000000-0000-4000-8000-000000000020"), SourceAtoms: pgarray.UUIDArray{a2, a3}},
	}
	index := buildAtomSceneIndex(scenes)

	bucket := memoryBucket{Atoms: []model.Atom{
		{URI: uri.BuildAtom(a2.String())},
		{URI: uri.BuildAtom(a3.String())},
	}}
	got := sourceSceneURIsFor(bucket, index)
	if len(got) != 2 || got[0] != "rmb://scenes/00000000-0000-4000-8000-000000000010" || got[1] != "rmb://scenes/00000000-0000-4000-8000-000000000020" {
		t.Fatalf("unexpected source scenes: %+v", got)
	}
}
