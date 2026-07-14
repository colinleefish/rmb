package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/colinleefish/rmb/internal/client"
	"github.com/colinleefish/rmb/internal/config"
	"github.com/colinleefish/rmb/internal/hook"
	"github.com/colinleefish/rmb/internal/service/recall"
	"github.com/colinleefish/rmb/internal/uri"
)

type ServeFunc func(context.Context) error

type Runner struct {
	Config config.Config
	Serve  ServeFunc
	Stdin  io.Reader
	Stdout io.Writer
}

func (r Runner) Run(ctx context.Context, args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		return r.printHelp(ctx)
	}
	if args[0] == "serve" {
		return r.Serve(ctx)
	}

	switch args[0] {
	case "hook-submit":
		return r.runHookSubmit(ctx, args[1:])
	case "cat", "tree", "meta":
		return r.runInspect(ctx, args[0], args[1:])
	case "search":
		return r.runSearch(ctx, args[1:])
	case "correction":
		return r.runCorrection(ctx, args[1:])
	case "skill":
		return r.runSkill(ctx, args[1:])
	default:
		return fmt.Errorf("unknown command %q\n\n%s", args[0], usage())
	}
}

func (r Runner) runHookSubmit(ctx context.Context, args []string) error {
	source := strings.TrimSpace(parseFlagValue(args, "--source"))
	if source == "" {
		return fmt.Errorf("hook-submit requires --source=<agent> (e.g. cursor, cc)")
	}

	stdin := r.Stdin
	if stdin == nil {
		stdin = os.Stdin
	}

	payload, err := io.ReadAll(stdin)
	if err != nil {
		return fmt.Errorf("read hook-submit stdin: %w", err)
	}

	// Diagnostic log goes to stderr so stdout stays empty.
	// Codex Stop hooks require stdout to be empty or valid JSON on exit 0;
	// writing log lines to stdout causes "invalid stop hook JSON output".
	return hook.Submit(ctx, hook.SubmitInput{
		Source:     source,
		StdinJSON:  payload,
		OutputSink: os.Stderr,
	})
}

func (r Runner) runSearch(ctx context.Context, args []string) error {
	query := strings.TrimSpace(strings.Join(positionalArgs(args), " "))
	if query == "" {
		return fmt.Errorf("usage: rmb search \"<query>\" [--scope=memory,scene,skill] [--k=<n>]")
	}
	k, err := parseK(args, 0) // 0 → server default (5)
	if err != nil {
		return err
	}
	scopes := parseScopes(args)

	cl, ok := client.Resolve()
	if !ok {
		return fmt.Errorf("search requires RMB_URL (the server owns the database)")
	}
	matches, err := cl.Search(ctx, query, k, scopes)
	if err != nil {
		return err
	}
	printMatches(r.stdout(), matches)
	return nil
}

// parseScopes reads --scope=memory,scene and returns the list. Returns nil
// (server default) when the flag is absent or empty.
func parseScopes(args []string) []string {
	raw := strings.TrimSpace(parseFlagValue(args, "--scope"))
	if raw == "" {
		return nil
	}
	var out []string
	for _, s := range strings.Split(raw, ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// runCorrection dispatches the human-correction subcommands:
//
//	rmb correction add <uri> [<uri>...] "statement"
//	rmb correction rm  <correction-uri>
//	rmb correction ls  [<target-uri>]
func (r Runner) runCorrection(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: rmb correction <add|rm|ls> ...")
	}
	switch args[0] {
	case "add":
		return r.runCorrectionAdd(ctx, args[1:])
	case "rm":
		return r.runCorrectionRm(ctx, args[1:])
	case "ls":
		return r.runCorrectionList(ctx, args[1:])
	default:
		return fmt.Errorf("unknown correction action %q (use add|rm|ls)", args[0])
	}
}

func (r Runner) runCorrectionAdd(ctx context.Context, args []string) error {
	pos := positionalArgs(args)

	var targets, words []string
	for _, a := range pos {
		if strings.HasPrefix(a, uri.Scheme+"://") {
			targets = append(targets, a)
		} else {
			words = append(words, a)
		}
	}
	statement := strings.TrimSpace(strings.Join(words, " "))
	if len(targets) == 0 || statement == "" {
		return fmt.Errorf("usage: rmb correction add <rmb://uri> [<uri>...] \"statement\"")
	}

	cl, ok := client.Resolve()
	if !ok {
		return fmt.Errorf("correction add requires RMB_URL (the server owns the database)")
	}
	createdURI, err := cl.CreateCorrection(ctx, targets, statement)
	if err != nil {
		return err
	}
	fmt.Fprintf(r.stdout(), "added correction: %s -> %s\n", strings.Join(targets, ", "), createdURI)
	return nil
}

func (r Runner) runCorrectionList(ctx context.Context, args []string) error {
	pos := positionalArgs(args)
	target := ""
	if len(pos) > 0 {
		target = pos[0]
	}

	cl, ok := client.Resolve()
	if !ok {
		return fmt.Errorf("correction ls requires RMB_URL (the server owns the database)")
	}
	items, err := cl.ListCorrections(ctx, target)
	if err != nil {
		return err
	}
	out := r.stdout()
	if len(items) == 0 {
		fmt.Fprintln(out, "no corrections")
		return nil
	}
	for _, it := range items {
		fmt.Fprintln(out, it.URI)
		fmt.Fprintf(out, "   -> %s\n", strings.Join(it.TargetURIs, ", "))
		if it.Statement != "" {
			fmt.Fprintf(out, "   %s\n", it.Statement)
		}
	}
	return nil
}

func (r Runner) runCorrectionRm(ctx context.Context, args []string) error {
	pos := positionalArgs(args)
	if len(pos) != 1 {
		return fmt.Errorf("usage: rmb correction rm <rmb://corrections/...>")
	}
	cl, ok := client.Resolve()
	if !ok {
		return fmt.Errorf("correction rm requires RMB_URL (the server owns the database)")
	}
	if err := cl.RetractCorrection(ctx, pos[0]); err != nil {
		return err
	}
	fmt.Fprintf(r.stdout(), "retracted: %s\n", pos[0])
	return nil
}

func printMatches(out io.Writer, matches []recall.Match) {
	if len(matches) == 0 {
		fmt.Fprintln(out, "no matches")
		return
	}
	for i, m := range matches {
		fmt.Fprintf(out, "%2d. [%s] %s\n", i+1, orDash(m.Tier), m.URI)
		if s := strings.TrimSpace(m.Snippet); s != "" {
			fmt.Fprintf(out, "      %s\n", s)
		}
		for _, c := range m.Corrections {
			fmt.Fprintf(out, "      \u2691 CORRECTION: %s\n", c.Statement)
		}
	}
}


// positionalArgs returns args that are not flags (do not start with "--").
func positionalArgs(args []string) []string {
	var out []string
	for _, a := range args {
		if strings.HasPrefix(a, "--") {
			continue
		}
		out = append(out, a)
	}
	return out
}

func parseK(args []string, def int) (int, error) {
	if v := strings.TrimSpace(parseFlagValue(args, "--k")); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil || parsed <= 0 {
			return 0, fmt.Errorf("--k must be a positive integer")
		}
		return parsed, nil
	}
	return def, nil
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}

func (r Runner) stdout() io.Writer {
	if r.Stdout != nil {
		return r.Stdout
	}
	return os.Stdout
}

func (r Runner) runInspect(ctx context.Context, command string, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("%s requires exactly one URI argument", command)
	}
	cl, ok := client.Resolve()
	if !ok {
		return fmt.Errorf("%s requires RMB_URL (the server owns the database)", command)
	}
	stdout := r.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	out, err := cl.Inspect(ctx, command, args[0])
	if err != nil {
		return err
	}
	_, err = io.WriteString(stdout, out)
	return err
}

func parseFlagValue(args []string, key string) string {
	for i, a := range args {
		if a == key && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(a, key+"=") {
			return strings.TrimPrefix(a, key+"=")
		}
	}
	return ""
}

func usage() string {
	return strings.TrimSpace(`
Usage:
  rmb [global-options] <command> [options]

Agent Help:
  Profile, Agent guide, and Skills sections above load from the server when RMB_URL is set.
  Do not run serve — recall uses the remote server. Search before asking the user
  (default scope includes skills); then cat / meta / tree as needed. Never invent uris.

Memory uris:
  profile | entities/<slug> | preferences/<slug> | events/<slug> | scenes/<uuid> | skills/<name>

Commands:
  serve
  hook-submit --source=<cursor|cc|codex|pi>
  search "<query>" [--scope=memory|scene|skill|...] [--k=<n>]
  skill ls
  skill put <name> [--dir=<path>]     default dir: ~/.rmb/skills/<name>/
  skill pull <name> [--out=<dir>]     default: ~/.rmb/skills/<name>/
  skill pull --all [--out=<base>]     default base: ~/.rmb/skills/
  tree <uri-prefix>
  cat <uri>
  meta <uri>
  correction add <uri> [<uri>...] "<statement>"
  correction rm <correction-uri>
  correction ls [<target-uri>]
  help

  search [--scope=...]       only search accepts --scope; cat/tree/meta take a single uri
  tree <uri-prefix>          browse: rmb://entities/, rmb://skills/, rmb://profile (not rmb://memories/)

Environment:
  RMB_URL           Remote server (required for recall commands). Env or ~/.rmb.conf / ~/.rmb/config.yaml.
  RMB_USERNAME      Basic auth user (alias: USERNAME).
  RMB_PASSWORD      Basic auth password (alias: PASSWORD).
  RMB_CONFIG        Override client config path (absolute path only for recall CLI).
`)
}

func (r Runner) printHelp(ctx context.Context) error {
	out := r.stdout()
	fmt.Fprintln(out, "rmb - long-term memory for AI agents")
	fmt.Fprintln(out)

	cl, ok := client.Resolve()
	if !ok {
		fmt.Fprintln(out, "(set RMB_URL to load Profile, Agent guide, and Skills from the server)")
		fmt.Fprintln(out)
		fmt.Fprintln(out, usage())
		return nil
	}

	bootstrap := []struct {
		marker string
		title  string
		uri    string
		blurb  string
	}{
		{
			marker: "profile",
			title:  "Profile",
			uri:    uri.BuildProfile(),
			blurb:  "Who the user is — stable identity, work context, and preferences.",
		},
		{
			marker: "agent",
			title:  "Agent guide",
			uri:    uri.BuildAgent(),
			blurb:  "How to use rmb — recall rules, URI shapes, and skill discovery.",
		},
	}
	for _, section := range bootstrap {
		printHelpSectionHeader(out, section.marker, section.title, section.uri, section.blurb)
		body, err := cl.Inspect(ctx, "cat", section.uri)
		if err != nil {
			fmt.Fprintf(out, "  (unavailable: %v)\n", err)
		} else if strings.TrimSpace(body) == "" {
			fmt.Fprintln(out, "  (empty)")
		} else {
			fmt.Fprintln(out, strings.TrimRight(body, "\n"))
		}
		fmt.Fprintln(out)
	}

	if err := r.printSkillsCatalog(ctx, cl, out); err != nil {
		printHelpSectionHeader(out, "skills", "Skills", uri.BuildSkill(""), "Curated playbooks. Read SKILL.md with rmb cat; run scripts via rmb skill pull.")
		fmt.Fprintf(out, "  (unavailable: %v)\n", err)
	}

	printHelpUsageDivider(out)
	fmt.Fprintln(out, usage())
	return nil
}

const helpSectionRule = "════════════════════════════════════════"

func printHelpSectionHeader(out io.Writer, marker, title, targetURI, blurb string) {
	fmt.Fprintln(out, helpSectionRule)
	if marker = strings.TrimSpace(marker); marker != "" {
		fmt.Fprintf(out, "[%s]\n", marker)
	}
	fmt.Fprintf(out, "%s  %s\n", title, targetURI)
	if blurb = strings.TrimSpace(blurb); blurb != "" {
		fmt.Fprintln(out, blurb)
	}
	fmt.Fprintln(out)
}

func printHelpUsageDivider(out io.Writer) {
	fmt.Fprintln(out, helpSectionRule)
	fmt.Fprintln(out, "[usage]")
	fmt.Fprintln(out)
}

const (
	helpSkillCatalogLimit = 20
	helpSkillDescMaxLen   = 120
)

func (r Runner) printSkillsCatalog(ctx context.Context, cl *client.Client, out io.Writer) error {
	items, err := cl.ListSkills(ctx)
	if err != nil {
		return err
	}
	printHelpSectionHeader(out, "skills", "Skills", uri.BuildSkill(""), "Curated playbooks. Read SKILL.md with rmb cat; run scripts via rmb skill pull.")
	if len(items) == 0 {
		fmt.Fprintln(out, "  (no skills)")
		return nil
	}

	sort.Slice(items, func(i, j int) bool { return items[i].URI < items[j].URI })

	limit := len(items)
	if limit > helpSkillCatalogLimit {
		limit = helpSkillCatalogLimit
	}
	for i := 0; i < limit; i++ {
		printHelpSkillEntry(out, items[i])
		if i < limit-1 {
			fmt.Fprintln(out)
		}
	}
	if len(items) > helpSkillCatalogLimit {
		fmt.Fprintf(out, "\n  ... %d more — rmb tree rmb://skills/\n", len(items)-helpSkillCatalogLimit)
	}
	return nil
}

func printHelpSkillEntry(out io.Writer, it client.SkillSummary) {
	fmt.Fprintln(out, it.URI)
	fmt.Fprintf(out, "  [desc] %s\n", formatHelpSkillDescription(it.Description))
	if len(it.Tags) > 0 {
		fmt.Fprintf(out, "  [tags] %s\n", strings.Join(it.Tags, ", "))
	}
}

func formatHelpSkillDescription(desc string) string {
	desc = strings.TrimSpace(desc)
	if desc == "" {
		return "(no description)"
	}
	desc = strings.Join(strings.Fields(strings.ReplaceAll(desc, "\n", " ")), " ")
	if len(desc) <= helpSkillDescMaxLen {
		return desc
	}
	return desc[:helpSkillDescMaxLen-1] + "…"
}
