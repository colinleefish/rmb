package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/colinleefish/mem9/internal/client"
	"github.com/colinleefish/mem9/internal/config"
	"github.com/colinleefish/mem9/internal/hook"
	"github.com/colinleefish/mem9/internal/service/recall"
	"github.com/colinleefish/mem9/internal/uri"
)

type ServeFunc func(context.Context) error

type Runner struct {
	Config config.Config
	Serve  ServeFunc
	Stdin  io.Reader
	Stdout io.Writer
}

func (r Runner) Run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		fmt.Fprintln(r.stdout(), usage())
		return nil
	}
	if args[0] == "serve" {
		return r.Serve(ctx)
	}

	switch args[0] {
	case "hook-submit":
		return r.runHookSubmit(ctx, args[1:])
	case "cat", "tree", "meta":
		return r.runInspect(ctx, args[0], args[1:])
	case "t1":
		return r.runT1(ctx, args[1:])
	case "t2":
		return r.runT2(ctx, args[1:])
	case "t3":
		return r.runT3(ctx, args[1:])
	case "embed":
		return r.runEmbed(ctx, args[1:])
	case "search":
		return r.runSearch(ctx, args[1:])
	case "correction":
		return r.runCorrection(ctx, args[1:])
	case "alias":
		return r.runAlias(ctx, args[1:])
	case "store", "read", "list", "delete", "load-context":
		return fmt.Errorf("%q command is planned but not implemented yet", args[0])
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
	stdout := r.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}

	payload, err := io.ReadAll(stdin)
	if err != nil {
		return fmt.Errorf("read hook-submit stdin: %w", err)
	}

	return hook.Submit(ctx, hook.SubmitInput{
		Source:     source,
		StdinJSON:  payload,
		OutputSink: stdout,
	})
}

func (r Runner) runT1(ctx context.Context, args []string) error {
	if len(args) == 0 || args[0] != "backfill" {
		return fmt.Errorf("usage: mem9 t1 backfill [--session=<uuid>]")
	}
	sessionKey := strings.TrimSpace(parseFlagValue(args[1:], "--session"))
	cl, ok := client.Resolve()
	if !ok {
		return fmt.Errorf("t1 backfill requires MEM9_URL (the server owns the database)")
	}
	n, err := cl.Backfill(ctx, "t1", sessionKey)
	if err != nil {
		return err
	}
	if sessionKey != "" {
		fmt.Fprintln(r.stdout(), "enqueued t1 for session", sessionKey)
	} else {
		fmt.Fprintf(r.stdout(), "enqueued t1 for %d session(s)\n", n)
	}
	return nil
}

func (r Runner) runT2(ctx context.Context, args []string) error {
	if len(args) == 0 || args[0] != "backfill" {
		return fmt.Errorf("usage: mem9 t2 backfill [--session=<uuid>]")
	}
	sessionKey := strings.TrimSpace(parseFlagValue(args[1:], "--session"))
	cl, ok := client.Resolve()
	if !ok {
		return fmt.Errorf("t2 backfill requires MEM9_URL (the server owns the database)")
	}
	n, err := cl.Backfill(ctx, "t2", sessionKey)
	if err != nil {
		return err
	}
	if sessionKey != "" {
		fmt.Fprintln(r.stdout(), "enqueued t2 for session", sessionKey)
	} else {
		fmt.Fprintf(r.stdout(), "enqueued t2 for %d session(s)\n", n)
	}
	return nil
}

func (r Runner) runT3(ctx context.Context, args []string) error {
	if len(args) == 0 || args[0] != "backfill" {
		return fmt.Errorf("usage: mem9 t3 backfill [--session=<uuid>]")
	}
	sessionKey := strings.TrimSpace(parseFlagValue(args[1:], "--session"))
	cl, ok := client.Resolve()
	if !ok {
		return fmt.Errorf("t3 backfill requires MEM9_URL (the server owns the database)")
	}
	n, err := cl.Backfill(ctx, "t3", sessionKey)
	if err != nil {
		return err
	}
	if sessionKey != "" {
		fmt.Fprintln(r.stdout(), "enqueued t3 for session", sessionKey)
	} else {
		fmt.Fprintf(r.stdout(), "enqueued t3 for %d session(s)\n", n)
	}
	return nil
}

func (r Runner) runSearch(ctx context.Context, args []string) error {
	query := strings.TrimSpace(strings.Join(positionalArgs(args), " "))
	if query == "" {
		return fmt.Errorf("usage: mem9 search \"<query>\" [--scope=memory,scene] [--k=<n>]")
	}
	k := parseK(args, 0) // 0 → server default (5)
	scopes := parseScopes(args)

	cl, ok := client.Resolve()
	if !ok {
		return fmt.Errorf("search requires MEM9_URL (the server owns the database)")
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
//	mem9 correction add <uri> [<uri>...] "statement"
//	mem9 correction rm  <correction-uri>
//	mem9 correction ls  [<target-uri>]
func (r Runner) runCorrection(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: mem9 correction <add|rm|ls> ...")
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
		return fmt.Errorf("usage: mem9 correction add <mem9://uri> [<uri>...] \"statement\"")
	}

	cl, ok := client.Resolve()
	if !ok {
		return fmt.Errorf("correction add requires MEM9_URL (the server owns the database)")
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
		return fmt.Errorf("correction ls requires MEM9_URL (the server owns the database)")
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
		return fmt.Errorf("usage: mem9 correction rm <mem9://corrections/...>")
	}
	cl, ok := client.Resolve()
	if !ok {
		return fmt.Errorf("correction rm requires MEM9_URL (the server owns the database)")
	}
	if err := cl.RetractCorrection(ctx, pos[0]); err != nil {
		return err
	}
	fmt.Fprintf(r.stdout(), "retracted: %s\n", pos[0])
	return nil
}

// runAlias dispatches the entity-alias subcommands:
//
//	mem9 alias set <alias-uri> <canonical-uri> ["note"]
//	mem9 alias rm  <alias-record-uri>
//	mem9 alias ls  [<uri>]
//	mem9 alias candidates [--status=pending]
//	mem9 alias confirm <candidate-id>
//	mem9 alias reject  <candidate-id>
func (r Runner) runAlias(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: mem9 alias <set|rm|ls|candidates|confirm|reject> ...")
	}
	switch args[0] {
	case "set":
		return r.runAliasSet(ctx, args[1:])
	case "rm":
		return r.runAliasRm(ctx, args[1:])
	case "ls":
		return r.runAliasList(ctx, args[1:])
	case "candidates":
		return r.runAliasCandidates(ctx, args[1:])
	case "confirm":
		return r.runAliasConfirm(ctx, args[1:])
	case "reject":
		return r.runAliasReject(ctx, args[1:])
	default:
		return fmt.Errorf("unknown alias action %q (use set|rm|ls|candidates|confirm|reject)", args[0])
	}
}

func (r Runner) runAliasCandidates(ctx context.Context, args []string) error {
	status := strings.TrimSpace(parseFlagValue(args, "--status"))
	cl, ok := client.Resolve()
	if !ok {
		return fmt.Errorf("alias candidates requires MEM9_URL (the server owns the database)")
	}
	items, err := cl.ListAliasCandidates(ctx, status)
	if err != nil {
		return err
	}
	out := r.stdout()
	if len(items) == 0 {
		fmt.Fprintln(out, "no alias candidates")
		return nil
	}
	for _, it := range items {
		fmt.Fprintf(out, "%s  [%s]\n", it.ID, it.Status)
		fmt.Fprintf(out, "   %s -> %s  (similarity %.3f)\n", it.AliasURI, it.CanonicalURI, it.Similarity)
		if it.Rationale != "" {
			fmt.Fprintf(out, "   %s\n", it.Rationale)
		}
	}
	return nil
}

func (r Runner) runAliasConfirm(ctx context.Context, args []string) error {
	pos := positionalArgs(args)
	if len(pos) != 1 {
		return fmt.Errorf("usage: mem9 alias confirm <candidate-id>")
	}
	cl, ok := client.Resolve()
	if !ok {
		return fmt.Errorf("alias confirm requires MEM9_URL (the server owns the database)")
	}
	if err := cl.ConfirmAliasCandidate(ctx, pos[0]); err != nil {
		return err
	}
	fmt.Fprintf(r.stdout(), "confirmed candidate: %s\n", pos[0])
	return nil
}

func (r Runner) runAliasReject(ctx context.Context, args []string) error {
	pos := positionalArgs(args)
	if len(pos) != 1 {
		return fmt.Errorf("usage: mem9 alias reject <candidate-id>")
	}
	cl, ok := client.Resolve()
	if !ok {
		return fmt.Errorf("alias reject requires MEM9_URL (the server owns the database)")
	}
	if err := cl.RejectAliasCandidate(ctx, pos[0]); err != nil {
		return err
	}
	fmt.Fprintf(r.stdout(), "rejected candidate: %s\n", pos[0])
	return nil
}

func (r Runner) runAliasSet(ctx context.Context, args []string) error {
	pos := positionalArgs(args)

	var uris, words []string
	for _, a := range pos {
		if strings.HasPrefix(a, uri.Scheme+"://") {
			uris = append(uris, a)
		} else {
			words = append(words, a)
		}
	}
	if len(uris) != 2 {
		return fmt.Errorf("usage: mem9 alias set <alias-uri> <canonical-uri> [\"note\"]")
	}
	note := strings.TrimSpace(strings.Join(words, " "))

	cl, ok := client.Resolve()
	if !ok {
		return fmt.Errorf("alias set requires MEM9_URL (the server owns the database)")
	}
	createdURI, err := cl.CreateAlias(ctx, uris[0], uris[1], note)
	if err != nil {
		return err
	}
	fmt.Fprintf(r.stdout(), "added alias: %s -> %s (%s)\n", uris[0], uris[1], createdURI)
	return nil
}

func (r Runner) runAliasList(ctx context.Context, args []string) error {
	pos := positionalArgs(args)
	filter := ""
	if len(pos) > 0 {
		filter = pos[0]
	}

	cl, ok := client.Resolve()
	if !ok {
		return fmt.Errorf("alias ls requires MEM9_URL (the server owns the database)")
	}
	items, err := cl.ListAliases(ctx, filter)
	if err != nil {
		return err
	}
	out := r.stdout()
	if len(items) == 0 {
		fmt.Fprintln(out, "no aliases")
		return nil
	}
	for _, it := range items {
		fmt.Fprintln(out, it.URI)
		fmt.Fprintf(out, "   %s -> %s\n", it.AliasURI, it.CanonicalURI)
		if it.Note != "" {
			fmt.Fprintf(out, "   %s\n", it.Note)
		}
	}
	return nil
}

func (r Runner) runAliasRm(ctx context.Context, args []string) error {
	pos := positionalArgs(args)
	if len(pos) != 1 {
		return fmt.Errorf("usage: mem9 alias rm <mem9://aliases/...>")
	}
	cl, ok := client.Resolve()
	if !ok {
		return fmt.Errorf("alias rm requires MEM9_URL (the server owns the database)")
	}
	if err := cl.RetractAlias(ctx, pos[0]); err != nil {
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

func parseK(args []string, def int) int {
	if v := strings.TrimSpace(parseFlagValue(args, "--k")); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			return parsed
		}
	}
	return def
}

func (r Runner) runEmbed(ctx context.Context, args []string) error {
	if len(args) == 0 || args[0] != "status" {
		return fmt.Errorf("usage: mem9 embed status")
	}
	cl, ok := client.Resolve()
	if !ok {
		return fmt.Errorf("embed status requires MEM9_URL (the server owns the database)")
	}
	rows, err := cl.EmbedStatus(ctx)
	if err != nil {
		return err
	}
	out := r.stdout()
	for _, row := range rows {
		fmt.Fprintf(out, "%-10s embedded=%d/%d pending=%d\n",
			row.Tier, row.Embedded, row.Total, row.Pending)
	}
	return nil
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
		return fmt.Errorf("%s requires MEM9_URL (the server owns the database)", command)
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
  mem9 serve                Start HTTP server
  mem9 hook-submit --source=<cursor|cc|codex>
                              Receive an agent transcript hook payload on stdin
  mem9 cat <uri>            Print body / messages_jsonl for a URI
  mem9 tree <uri-prefix>    List child URIs under a prefix
  mem9 meta <uri>           Print row metadata as JSON
  mem9 t1 backfill          Enqueue T1 extraction for sessions with unprocessed turns
                              Optional: --session=<uuid>
  mem9 t2 backfill          Enqueue T2 scene build for sessions with atoms
                              Optional: --session=<uuid>
  mem9 t3 backfill          Enqueue T3 memory rollup for sessions with scenes
                              Optional: --session=<uuid>
  mem9 embed status         Show embedding coverage across atoms/scenes/memories
  mem9 search "<query>"     Hybrid recall (vector + FTS) across memories and scenes
                              --scope=memory,scene  Tiers to search (default: memory,scene)
                                memory  Long-term distilled facts
                                scene   Per-session conversation summaries
                              --k=<n>              Number of results (default: 5)
  mem9 correction add <uri> [<uri>...] "statement"
                              Attach a human correction that overrides memory at recall
  mem9 correction rm <correction-uri>
                              Retire a specific correction (URI from meta/ls output)
  mem9 correction ls [<target-uri>]
                              List active corrections (optionally for one target)
  mem9 alias set <alias-uri> <canonical-uri> ["note"]
                              Declare alias-uri to be the same entity as canonical-uri
  mem9 alias rm <alias-record-uri>
                              Retract a specific alias (URI from meta/ls output)
  mem9 alias ls [<uri>]     List active aliases (optionally for one URI, either side)
  mem9 alias candidates [--status=pending]
                              List machine-proposed alias candidates (pending|confirmed|rejected|all)
  mem9 alias confirm <candidate-id>
                              Promote a candidate into a live alias
  mem9 alias reject <candidate-id>
                              Reject a candidate so it is never re-proposed
  mem9 store <uri>          Planned
  mem9 read <uri>           Planned
  mem9 list <prefix>        Planned
  mem9 delete <uri>         Planned
  mem9 load-context         Planned
`)
}
