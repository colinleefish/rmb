package cli

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/colinleefish/rmb/internal/client"
	"github.com/colinleefish/rmb/internal/service/skill"
	"github.com/colinleefish/rmb/internal/uri"
)

func (r Runner) runSkill(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: rmb skill <ls|put|pull> ...")
	}
	switch args[0] {
	case "ls":
		return r.runSkillList(ctx)
	case "put":
		return r.runSkillPut(ctx, args[1:])
	case "pull":
		return r.runSkillPull(ctx, args[1:])
	default:
		return fmt.Errorf("unknown skill action %q (use ls|put|pull)", args[0])
	}
}

func (r Runner) runSkillList(ctx context.Context) error {
	cl, ok := client.Resolve()
	if !ok {
		return fmt.Errorf("skill ls requires RMB_URL")
	}
	items, err := cl.ListSkills(ctx)
	if err != nil {
		return err
	}
	out := r.stdout()
	if len(items) == 0 {
		fmt.Fprintln(out, "no skills")
		return nil
	}
	for _, it := range items {
		tags := strings.Join(it.Tags, ", ")
		if tags == "" {
			tags = "-"
		}
		fmt.Fprintf(out, "%s\t[%s]\t%s\n", it.URI, tags, it.Description)
	}
	return nil
}

func (r Runner) runSkillPut(ctx context.Context, args []string) error {
	pos := positionalArgs(args)
	if len(pos) == 0 {
		return fmt.Errorf("usage: rmb skill put <name> [--dir=<path>]")
	}
	name := pos[0]
	if err := uri.ValidateSkillName(name); err != nil {
		return err
	}

	dir := strings.TrimSpace(parseFlagValue(args, "--dir"))
	if dir == "" {
		defaultDir, err := skillDir(name)
		if err != nil {
			return err
		}
		if st, err := os.Stat(defaultDir); err != nil || !st.IsDir() {
			return fmt.Errorf("default dir %s not found; pass --dir=<path>", defaultDir)
		}
		dir = defaultDir
	}

	files, err := walkSkillDir(dir)
	if err != nil {
		return err
	}

	cl, ok := client.Resolve()
	if !ok {
		return fmt.Errorf("skill put requires RMB_URL")
	}
	result, err := cl.PutSkill(ctx, name, files)
	if err != nil {
		return err
	}
	if result.NoOp {
		fmt.Fprintf(r.stdout(), "unchanged: %s (version %d)\n", result.URI, result.Version)
		return nil
	}
	fmt.Fprintf(r.stdout(), "uploaded: %s (version %d)\n", result.URI, result.Version)
	return nil
}

func (r Runner) runSkillPull(ctx context.Context, args []string) error {
	all := false
	for _, a := range args {
		if a == "--all" {
			all = true
		}
	}
	pos := positionalArgs(args)
	outBase := strings.TrimSpace(parseFlagValue(args, "--out"))
	if outBase == "" {
		base, err := skillsRoot()
		if err != nil {
			return err
		}
		outBase = base
	}

	cl, ok := client.Resolve()
	if !ok {
		return fmt.Errorf("skill pull requires RMB_URL")
	}

	if all {
		items, err := cl.ListSkills(ctx)
		if err != nil {
			return err
		}
		for _, it := range items {
			parsed, err := uri.Parse(it.URI)
			if err != nil || len(parsed.Segments) == 0 {
				continue
			}
			dest := filepath.Join(outBase, parsed.Segments[0])
			if err := r.materializeSkill(ctx, cl, parsed.Segments[0], dest); err != nil {
				return err
			}
		}
		return nil
	}

	if len(pos) == 0 {
		return fmt.Errorf("usage: rmb skill pull <name> [--out=<dir>] | rmb skill pull --all [--out=<base>]")
	}
	name := pos[0]
	dest := outBase
	if len(pos) == 1 && !strings.HasSuffix(outBase, name) {
		// single skill: default ~/.rmb/skills/<name>/
		if strings.TrimSpace(parseFlagValue(args, "--out")) == "" {
			var err error
			dest, err = skillDir(name)
			if err != nil {
				return err
			}
		} else {
			dest = filepath.Join(outBase, name)
		}
	}
	return r.materializeSkill(ctx, cl, name, dest)
}

func (r Runner) materializeSkill(ctx context.Context, cl *client.Client, name, dest string) error {
	detail, err := cl.GetSkill(ctx, name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", dest, err)
	}
	for rel, content := range detail.Files {
		target := filepath.Join(dest, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", target, err)
		}
	}
	manifest := filepath.Join(dest, skill.ManifestPath)
	fmt.Fprintf(r.stdout(), "%s\n", manifest)
	return nil
}

func skillsRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".rmb", "skills"), nil
}

func skillDir(name string) (string, error) {
	root, err := skillsRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, name), nil
}

func walkSkillDir(dir string) ([]client.SkillFile, error) {
	var files []client.SkillFile
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files = append(files, client.SkillFile{Path: rel, Content: string(data)})
		return nil
	})
	return files, err
}
