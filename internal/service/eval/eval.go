// Package eval runs fixed recall probes against the memory pyramid to detect
// consolidation drift: cases where the raw conversation contained the answer but
// the distilled memories tier fails to surface it. v1 is lexical (FTS) only.
package eval

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/colinleefish/mypast/internal/service/recall"
	"gorm.io/gorm"
)

// Probe is one recall test: a natural-language query and the URI prefix the
// answer is expected to live under (optional).
type Probe struct {
	Query          string
	ExpectedPrefix string
}

// Result is the outcome for one probe.
type Result struct {
	Probe       Probe
	FullHit     bool   // expected memory surfaced in top-k
	MatchedURI  string // first full-stack match (for display)
	BaselineHas bool   // raw turns contained the topic
	Regression  bool   // baseline had it but full missed it
}

// Report aggregates probe results.
type Report struct {
	Results     []Result
	FullHits    int
	Regressions int
}

// LoadProbes reads `query<TAB>expected-uri-prefix` lines, skipping blanks/comments.
func LoadProbes(path string) ([]Probe, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open queries: %w", err)
	}
	defer f.Close()

	var probes []Probe
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		query := strings.TrimSpace(parts[0])
		if query == "" {
			continue
		}
		prefix := ""
		if len(parts) == 2 {
			prefix = strings.TrimSpace(parts[1])
		}
		probes = append(probes, Probe{Query: query, ExpectedPrefix: prefix})
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read queries: %w", err)
	}
	if len(probes) == 0 {
		return nil, fmt.Errorf("no probes found in %s", path)
	}
	return probes, nil
}

// hitByPrefix reports whether any match URI satisfies the expected prefix. An
// empty prefix means "any memory hit counts".
func hitByPrefix(matches []recall.Match, prefix string) bool {
	if prefix == "" {
		return len(matches) > 0
	}
	for _, m := range matches {
		if strings.HasPrefix(m.URI, prefix) {
			return true
		}
	}
	return false
}

// classify derives a Result from full-stack and baseline matches.
func classify(probe Probe, full, baseline []recall.Match) Result {
	fullHit := hitByPrefix(full, probe.ExpectedPrefix)
	baselineHas := len(baseline) > 0
	matched := ""
	if len(full) > 0 {
		matched = full[0].URI
	}
	return Result{
		Probe:       probe,
		FullHit:     fullHit,
		MatchedURI:  matched,
		BaselineHas: baselineHas,
		Regression:  baselineHas && !fullHit,
	}
}

// Run executes every probe and aggregates a report.
func Run(ctx context.Context, db *gorm.DB, probes []Probe, k int) (Report, error) {
	var rep Report
	for _, probe := range probes {
		full, err := recall.FTSMemories(ctx, db, probe.Query, k)
		if err != nil {
			return Report{}, err
		}
		baseline, err := recall.FTSTurns(ctx, db, probe.Query, k)
		if err != nil {
			return Report{}, err
		}
		res := classify(probe, full, baseline)
		if res.FullHit {
			rep.FullHits++
		}
		if res.Regression {
			rep.Regressions++
		}
		rep.Results = append(rep.Results, res)
	}
	return rep, nil
}
