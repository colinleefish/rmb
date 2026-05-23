package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/colinleefish/mypast/internal/config"
	"github.com/colinleefish/mypast/internal/db"
	"github.com/colinleefish/mypast/internal/hook"
	"github.com/colinleefish/mypast/internal/service/extract"
	"github.com/colinleefish/mypast/internal/service/inspect"
	"github.com/google/uuid"
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
	case "store", "read", "list", "delete", "search", "load-context":
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

	stdout := r.Stdout
	if stdout == nil {
		stdout = os.Stdout
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
  mypast store <uri>          Planned
  mypast read <uri>           Planned
  mypast list <prefix>        Planned
  mypast delete <uri>         Planned
  mypast search <query>       Planned
  mypast load-context         Planned
`)
}
