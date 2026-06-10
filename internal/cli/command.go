package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/colinleefish/mypast/internal/client"
	"github.com/colinleefish/mypast/internal/config"
	"github.com/colinleefish/mypast/internal/db"
	"github.com/colinleefish/mypast/internal/db/pgarray"
	"github.com/colinleefish/mypast/internal/hook"
	"github.com/colinleefish/mypast/internal/llm"
	"github.com/colinleefish/mypast/internal/model"
	"github.com/colinleefish/mypast/internal/service/assertion"
	"github.com/colinleefish/mypast/internal/service/embed"
	"github.com/colinleefish/mypast/internal/service/eval"
	"github.com/colinleefish/mypast/internal/service/extract"
	"github.com/colinleefish/mypast/internal/service/inspect"
	"github.com/colinleefish/mypast/internal/service/memory"
	"github.com/colinleefish/mypast/internal/service/recall"
	"github.com/colinleefish/mypast/internal/service/scene"
	"github.com/colinleefish/mypast/internal/uri"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ServeFunc func(context.Context) error

type Runner struct {
	Config config.Config
	Serve  ServeFunc
	Stdin  io.Reader
	Stdout io.Writer
}

func (r Runner) Run(ctx context.Context, args []string) error {
	if len(args) == 0 || args[0] == "serve" {
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
	case "eval":
		return r.runEval(ctx, args[1:])
	case "embed":
		return r.runEmbed(ctx, args[1:])
	case "find":
		return r.runFind(ctx, args[1:])
	case "search":
		return r.runSearch(ctx, args[1:])
	case "assertion":
		return r.runAssertion(ctx, args[1:])
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
		return fmt.Errorf("usage: mypast t1 backfill [--session=<uuid>]")
	}

	sessionKey := strings.TrimSpace(parseFlagValue(args[1:], "--session"))

	database, err := db.New(ctx, r.Config.DB.URL)
	if err != nil {
		return fmt.Errorf("db connect: %w", err)
	}
	sqlDB, err := database.DB()
	if err != nil {
		return fmt.Errorf("get db handle: %w", err)
	}
	defer sqlDB.Close()

	if err := db.Migrate(ctx, database); err != nil {
		return fmt.Errorf("db migrate: %w", err)
	}

	if sessionKey != "" {
		if err := extract.EnqueueSessionByKey(ctx, database, sessionKey); err != nil {
			return err
		}
		fmt.Fprintln(r.stdout(), "enqueued t1 for session", sessionKey)
		return nil
	}

	type row struct {
		SessionID uuid.UUID
	}
	var rows []row
	if err := database.WithContext(ctx).Raw(`
		SELECT DISTINCT session_id
		FROM session_turns
		WHERE t1_extracted_at IS NULL
	`).Scan(&rows).Error; err != nil {
		return fmt.Errorf("list sessions with pending turns: %w", err)
	}
	for _, row := range rows {
		if err := extract.EnqueueSession(ctx, database, row.SessionID); err != nil {
			return err
		}
	}
	fmt.Fprintf(r.stdout(), "enqueued t1 for %d session(s)\n", len(rows))
	return nil
}

func (r Runner) runT2(ctx context.Context, args []string) error {
	if len(args) == 0 || args[0] != "backfill" {
		return fmt.Errorf("usage: mypast t2 backfill [--session=<uuid>]")
	}

	sessionKey := strings.TrimSpace(parseFlagValue(args[1:], "--session"))

	database, err := db.New(ctx, r.Config.DB.URL)
	if err != nil {
		return fmt.Errorf("db connect: %w", err)
	}
	sqlDB, err := database.DB()
	if err != nil {
		return fmt.Errorf("get db handle: %w", err)
	}
	defer sqlDB.Close()

	if err := db.Migrate(ctx, database); err != nil {
		return fmt.Errorf("db migrate: %w", err)
	}

	if sessionKey != "" {
		if err := scene.EnqueueSessionByKey(ctx, database, sessionKey); err != nil {
			return err
		}
		fmt.Fprintln(r.stdout(), "enqueued t2 for session", sessionKey)
		return nil
	}

	count, err := scene.EnqueueAllSessionsWithAtoms(ctx, database)
	if err != nil {
		return err
	}
	fmt.Fprintf(r.stdout(), "enqueued t2 for %d session(s)\n", count)
	return nil
}

func (r Runner) runT3(ctx context.Context, args []string) error {
	if len(args) == 0 || args[0] != "backfill" {
		return fmt.Errorf("usage: mypast t3 backfill [--session=<uuid>]")
	}

	sessionKey := strings.TrimSpace(parseFlagValue(args[1:], "--session"))

	database, err := db.New(ctx, r.Config.DB.URL)
	if err != nil {
		return fmt.Errorf("db connect: %w", err)
	}
	sqlDB, err := database.DB()
	if err != nil {
		return fmt.Errorf("get db handle: %w", err)
	}
	defer sqlDB.Close()

	if err := db.Migrate(ctx, database); err != nil {
		return fmt.Errorf("db migrate: %w", err)
	}

	if sessionKey != "" {
		if err := memory.EnqueueSessionByKey(ctx, database, sessionKey); err != nil {
			return err
		}
		fmt.Fprintln(r.stdout(), "enqueued t3 for session", sessionKey)
		return nil
	}

	count, err := memory.EnqueueAllSessionsWithScenes(ctx, database)
	if err != nil {
		return err
	}
	fmt.Fprintf(r.stdout(), "enqueued t3 for %d session(s)\n", count)
	return nil
}

func (r Runner) runEval(ctx context.Context, args []string) error {
	path := strings.TrimSpace(parseFlagValue(args, "--queries"))
	if path == "" {
		path = "scripts/eval_queries.txt"
	}
	k := 5
	if v := strings.TrimSpace(parseFlagValue(args, "--k")); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("parse --k: %w", err)
		}
		k = parsed
	}

	probes, err := eval.LoadProbes(path)
	if err != nil {
		return err
	}

	database, err := db.New(ctx, r.Config.DB.URL)
	if err != nil {
		return fmt.Errorf("db connect: %w", err)
	}
	sqlDB, err := database.DB()
	if err != nil {
		return fmt.Errorf("get db handle: %w", err)
	}
	defer sqlDB.Close()

	if err := db.Migrate(ctx, database); err != nil {
		return fmt.Errorf("db migrate: %w", err)
	}

	// Use hybrid (vector + FTS) for the full-stack path when an embed client is
	// configured; fall back to FTS-only otherwise.
	var embedder eval.QueryEmbedder
	if strings.TrimSpace(r.Config.Embed.APIKey) != "" {
		embedder = r.embedQuery
	}

	report, err := eval.Run(ctx, database, probes, k, embedder)
	if err != nil {
		return err
	}

	out := r.stdout()
	fmt.Fprintf(out, "eval: %d probes (k=%d)\n\n", len(report.Results), k)
	for _, res := range report.Results {
		status := "MISS"
		if res.FullHit {
			status = "HIT "
		}
		if res.Regression {
			status = "REGR"
		}
		fmt.Fprintf(out, "[%s] %s\n", status, res.Probe.Query)
		fmt.Fprintf(out, "        expect=%s baseline=%v matched=%s\n",
			orDash(res.Probe.ExpectedPrefix), res.BaselineHas, orDash(res.MatchedURI))
	}
	fmt.Fprintf(out, "\nsummary: %d/%d full hits, %d regressions\n",
		report.FullHits, len(report.Results), report.Regressions)

	if report.Regressions > 0 {
		return fmt.Errorf("eval failed: %d regression(s)", report.Regressions)
	}
	return nil
}

func (r Runner) embedQuery(ctx context.Context, query string) (pgarray.Vector, error) {
	client, err := llm.NewEmbeddingClient(r.Config.Embed)
	if err != nil {
		return nil, fmt.Errorf("embedding client: %w", err)
	}
	vecs, err := client.Embed(ctx, []string{query})
	if err != nil {
		return nil, err
	}
	if len(vecs) != 1 {
		return nil, fmt.Errorf("expected 1 query vector, got %d", len(vecs))
	}
	return pgarray.Vector(vecs[0]), nil
}

// runFind is single-query vector recall over long-term memories. When MYPAST_URL
// is configured it calls a remote server; otherwise it queries the local DB.
func (r Runner) runFind(ctx context.Context, args []string) error {
	query := strings.TrimSpace(strings.Join(positionalArgs(args), " "))
	if query == "" {
		return fmt.Errorf("usage: mypast find <query> [--k=<n>]")
	}
	k := parseK(args, 5)

	if cl, ok := client.Resolve(); ok {
		matches, err := cl.Find(ctx, query, k)
		if err != nil {
			return err
		}
		printMatches(r.stdout(), matches)
		return nil
	}

	queryVec, err := r.embedQuery(ctx, query)
	if err != nil {
		return err
	}
	database, closeDB, err := r.openDB(ctx)
	if err != nil {
		return err
	}
	defer closeDB()

	matches, err := recall.VectorMemories(ctx, database, queryVec, k)
	if err != nil {
		return err
	}
	if err := recall.AttachCorrections(ctx, database, matches); err != nil {
		return err
	}
	printMatches(r.stdout(), matches)
	return nil
}

// runSearch is hybrid recall: vector + FTS across memories and scenes, fused
// with reciprocal rank fusion. When MYPAST_URL is configured it calls a remote
// server; otherwise it queries the local DB. The design's LLM intent analysis
// and hierarchical score propagation are deferred.
func (r Runner) runSearch(ctx context.Context, args []string) error {
	query := strings.TrimSpace(strings.Join(positionalArgs(args), " "))
	if query == "" {
		return fmt.Errorf("usage: mypast search <query> [--k=<n>]")
	}
	k := parseK(args, 8)

	if cl, ok := client.Resolve(); ok {
		matches, err := cl.Search(ctx, query, k)
		if err != nil {
			return err
		}
		printMatches(r.stdout(), matches)
		return nil
	}

	queryVec, err := r.embedQuery(ctx, query)
	if err != nil {
		return err
	}
	database, closeDB, err := r.openDB(ctx)
	if err != nil {
		return err
	}
	defer closeDB()

	perList := k * 2
	vecMem, err := recall.VectorMemories(ctx, database, queryVec, perList)
	if err != nil {
		return err
	}
	vecScene, err := recall.VectorScenes(ctx, database, queryVec, perList)
	if err != nil {
		return err
	}
	ftsMem, err := recall.FTSMemories(ctx, database, query, perList)
	if err != nil {
		return err
	}
	ftsScene, err := recall.FTSScenes(ctx, database, query, perList)
	if err != nil {
		return err
	}

	fused := recall.FuseRRF([][]recall.Match{vecMem, ftsMem, vecScene, ftsScene}, 60, k)
	if err := recall.AttachCorrections(ctx, database, fused); err != nil {
		return err
	}
	printMatches(r.stdout(), fused)
	return nil
}

// runAssertion dispatches the human-correction subcommands:
//
//	mypast assertion add <kind> <uri> [<uri>...] "statement"
//	mypast assertion rm  <assertion-uri>
//	mypast assertion ls  [<target-uri>]
func (r Runner) runAssertion(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: mypast assertion <add|rm|ls> ...")
	}
	switch args[0] {
	case "add":
		return r.runAssertionAdd(ctx, args[1:])
	case "rm":
		return r.runRetract(ctx, args[1:])
	case "ls":
		return r.runAssertionList(ctx, args[1:])
	default:
		return fmt.Errorf("unknown assertion action %q (use add|rm|ls)", args[0])
	}
}

// parseAssertionKind maps a user-facing kind token to a canonical kind. "fix" is
// accepted as a friendly alias for "correct".
func parseAssertionKind(tok string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(tok)) {
	case "correct", "fix":
		return model.AssertionKindCorrect, nil
	case "forget":
		return model.AssertionKindForget, nil
	default:
		return "", fmt.Errorf("unsupported kind %q (v1: correct, forget)", tok)
	}
}

// runAssertionAdd writes a human correction. The first positional arg is the
// kind; positional args with the mypast scheme are targets; the rest form the
// statement. Remote when MYPAST_URL is set, else local DB.
func (r Runner) runAssertionAdd(ctx context.Context, args []string) error {
	pos := positionalArgs(args)
	if len(pos) == 0 {
		return fmt.Errorf("usage: mypast assertion add <correct|forget> <mypast://uri> [<uri>...] \"statement\"")
	}
	kind, err := parseAssertionKind(pos[0])
	if err != nil {
		return err
	}

	var targets, words []string
	for _, a := range pos[1:] {
		if strings.HasPrefix(a, uri.Scheme+"://") {
			targets = append(targets, a)
		} else {
			words = append(words, a)
		}
	}
	statement := strings.TrimSpace(strings.Join(words, " "))
	if len(targets) == 0 {
		return fmt.Errorf("usage: mypast assertion add %s <mypast://uri> [<uri>...] \"statement\"", pos[0])
	}

	if cl, ok := client.Resolve(); ok {
		createdURI, err := cl.CreateAssertion(ctx, kind, targets, statement)
		if err != nil {
			return err
		}
		fmt.Fprintf(r.stdout(), "added %s: %s -> %s\n", kind, strings.Join(targets, ", "), createdURI)
		return nil
	}

	database, closeDB, err := r.openDB(ctx)
	if err != nil {
		return err
	}
	defer closeDB()

	row, err := assertion.NewService(database).Create(ctx, assertion.CreateInput{
		Kind:       kind,
		TargetURIs: targets,
		Statement:  statement,
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(r.stdout(), "added %s: %s -> %s\n", kind, strings.Join(targets, ", "), row.URI)
	return nil
}

// runAssertionList prints active assertions, optionally filtered to one target.
func (r Runner) runAssertionList(ctx context.Context, args []string) error {
	pos := positionalArgs(args)
	target := ""
	if len(pos) > 0 {
		target = pos[0]
	}

	if cl, ok := client.Resolve(); ok {
		items, err := cl.ListAssertions(ctx, target)
		if err != nil {
			return err
		}
		out := r.stdout()
		if len(items) == 0 {
			fmt.Fprintln(out, "no assertions")
			return nil
		}
		for _, it := range items {
			fmt.Fprintf(out, "%s  [%s]\n", it.URI, it.Kind)
			fmt.Fprintf(out, "   -> %s\n", strings.Join(it.TargetURIs, ", "))
			if it.Statement != "" {
				fmt.Fprintf(out, "   %s\n", it.Statement)
			}
		}
		return nil
	}

	database, closeDB, err := r.openDB(ctx)
	if err != nil {
		return err
	}
	defer closeDB()

	rows, err := assertion.NewService(database).List(ctx, target)
	if err != nil {
		return err
	}
	out := r.stdout()
	if len(rows) == 0 {
		fmt.Fprintln(out, "no assertions")
		return nil
	}
	for _, row := range rows {
		fmt.Fprintf(out, "%s  [%s]\n", row.URI, row.Kind)
		fmt.Fprintf(out, "   -> %s\n", strings.Join([]string(row.TargetURIs), ", "))
		if row.Statement != nil && *row.Statement != "" {
			fmt.Fprintf(out, "   %s\n", *row.Statement)
		}
	}
	return nil
}

// runRetract retires a specific correction by its assertion URI. Remote when
// MYPAST_URL is set, else local DB.
func (r Runner) runRetract(ctx context.Context, args []string) error {
	pos := positionalArgs(args)
	if len(pos) != 1 {
		return fmt.Errorf("usage: mypast retract <mypast://assertions/...>")
	}
	target := pos[0]

	if cl, ok := client.Resolve(); ok {
		if err := cl.RetractAssertion(ctx, target); err != nil {
			return err
		}
		fmt.Fprintf(r.stdout(), "retracted: %s\n", target)
		return nil
	}

	database, closeDB, err := r.openDB(ctx)
	if err != nil {
		return err
	}
	defer closeDB()

	if err := assertion.NewService(database).Retract(ctx, target); err != nil {
		return err
	}
	fmt.Fprintf(r.stdout(), "retracted: %s\n", target)
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
			label := "CORRECTION"
			if c.Kind == model.AssertionKindForget {
				label = "RETIRED"
			}
			fmt.Fprintf(out, "      \u2691 %s: %s\n", label, c.Statement)
		}
	}
}

// openDB opens, migrates, and returns the database with a close func.
func (r Runner) openDB(ctx context.Context) (*gorm.DB, func(), error) {
	database, err := db.New(ctx, r.Config.DB.URL)
	if err != nil {
		return nil, nil, fmt.Errorf("db connect: %w", err)
	}
	sqlDB, err := database.DB()
	if err != nil {
		return nil, nil, fmt.Errorf("get db handle: %w", err)
	}
	if err := db.Migrate(ctx, database); err != nil {
		sqlDB.Close()
		return nil, nil, fmt.Errorf("db migrate: %w", err)
	}
	return database, func() { sqlDB.Close() }, nil
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
		return fmt.Errorf("usage: mypast embed status")
	}

	database, err := db.New(ctx, r.Config.DB.URL)
	if err != nil {
		return fmt.Errorf("db connect: %w", err)
	}
	sqlDB, err := database.DB()
	if err != nil {
		return fmt.Errorf("get db handle: %w", err)
	}
	defer sqlDB.Close()

	if err := db.Migrate(ctx, database); err != nil {
		return fmt.Errorf("db migrate: %w", err)
	}

	rows, err := embed.Status(ctx, database)
	if err != nil {
		return err
	}
	out := r.stdout()
	for _, row := range rows {
		fmt.Fprintf(out, "%-10s embedded=%d/%d pending=%d\n",
			row.Tier, row.Total-row.Pending, row.Total, row.Pending)
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

// runInspect runs cat/tree/meta. When MYPAST_URL is configured it calls a remote
// server; otherwise it queries the local database.
func (r Runner) runInspect(ctx context.Context, command string, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("%s requires exactly one URI argument", command)
	}

	stdout := r.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}

	if cl, ok := client.Resolve(); ok {
		out, err := cl.Inspect(ctx, command, args[0])
		if err != nil {
			return err
		}
		_, err = io.WriteString(stdout, out)
		return err
	}

	database, err := db.New(ctx, r.Config.DB.URL)
	if err != nil {
		return fmt.Errorf("db connect: %w", err)
	}
	sqlDB, err := database.DB()
	if err != nil {
		return fmt.Errorf("get db handle: %w", err)
	}
	defer sqlDB.Close()

	if err := db.Migrate(ctx, database); err != nil {
		return fmt.Errorf("db migrate: %w", err)
	}

	svc := inspect.NewService(database)
	switch command {
	case "cat":
		return svc.Cat(ctx, args[0], stdout)
	case "tree":
		return svc.Tree(ctx, args[0], stdout)
	case "meta":
		return svc.Meta(ctx, args[0], stdout)
	default:
		return fmt.Errorf("unknown inspect command %q", command)
	}
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
  mypast serve                Start HTTP server (default)
  mypast hook-submit --source=<cursor|cc|codex>
                              Receive an agent transcript hook payload on stdin
  mypast cat <uri>            Print body / messages_jsonl for a URI
  mypast tree <uri-prefix>    List child URIs under a prefix
  mypast meta <uri>           Print row metadata as JSON
  mypast t1 backfill          Enqueue T1 extraction for sessions with unprocessed turns
                              Optional: --session=<uuid>
  mypast t2 backfill          Enqueue T2 scene build for sessions with atoms
                              Optional: --session=<uuid>
  mypast t3 backfill          Enqueue T3 memory rollup for sessions with scenes
                              Optional: --session=<uuid>
  mypast eval                 Run recall probes (FTS) over memories vs raw turns
                              Optional: --queries=<path> --k=<n>
  mypast embed status         Show embedding coverage across atoms/scenes/memories
  mypast find <query>         Vector recall over long-term memories
                              Optional: --k=<n>
  mypast search <query>       Hybrid recall (vector + FTS) across memories and scenes
                              Optional: --k=<n>
  mypast assertion add <correct|forget> <uri> [<uri>...] "statement"
                              Attach a human correction that overrides memory at recall
  mypast assertion rm <assertion-uri>
                              Retire a specific correction (URI from meta/ls output)
  mypast assertion ls [<target-uri>]
                              List active corrections (optionally for one target)
  mypast store <uri>          Planned
  mypast read <uri>           Planned
  mypast list <prefix>        Planned
  mypast delete <uri>         Planned
  mypast load-context         Planned
`)
}
