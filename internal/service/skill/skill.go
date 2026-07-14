package skill

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/colinleefish/rmb/internal/db/pgarray"
	"github.com/colinleefish/rmb/internal/model"
	"github.com/colinleefish/rmb/internal/uri"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	// ManifestPath is the required skill entrypoint file.
	ManifestPath   = "SKILL.md"
	maxFileBytes   = 512 * 1024
	maxBundleBytes = 2 * 1024 * 1024
)

var allowedExtensions = map[string]struct{}{
	".md":   {},
	".txt":  {},
	".py":   {},
	".sh":   {},
	".js":   {},
	".ts":   {},
	".mjs":  {},
	".json": {},
	".yaml": {},
	".yml":  {},
	".toml": {},
}

// FileInput is one file in a skill bundle upload.
type FileInput struct {
	Path    string
	Content string
}

// BundleInput is the validated payload for ReplaceBundle.
type BundleInput struct {
	Files []FileInput
}

// CatalogEntry is tier-1 discovery metadata.
type CatalogEntry struct {
	URI         string   `json:"uri"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

// FileNode is a node in the skill file tree (for UI / inspect).
type FileNode struct {
	Name     string     `json:"name"`
	Path     string     `json:"path"`
	Type     string     `json:"type"` // "file" or "dir"
	Children []FileNode `json:"children,omitempty"`
}

// Detail is the full skill bundle for browse/UI.
type Detail struct {
	Skill SkillMeta         `json:"skill"`
	Tree  []FileNode        `json:"tree"`
	Files map[string]string `json:"files"`
}

// SkillMeta is skill row metadata without file bodies.
type SkillMeta struct {
	URI          string    `json:"uri"`
	Slug         string    `json:"slug"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Tags         []string  `json:"tags"`
	Version      int       `json:"version"`
	BundleSHA256 string    `json:"bundle_sha256"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ReplaceResult is returned after a successful put.
type ReplaceResult struct {
	URI     string `json:"uri"`
	Version int    `json:"version"`
	NoOp    bool   `json:"no_op"`
}

type frontmatter struct {
	Name        string
	Description string
	Tags        []string
}

// ReplaceBundle versions the active skill (supersede + insert) when the bundle changed.
func ReplaceBundle(ctx context.Context, db *gorm.DB, slug string, input BundleInput) (ReplaceResult, error) {
	slug = strings.TrimSpace(slug)
	if err := uri.ValidateSkillName(slug); err != nil {
		return ReplaceResult{}, err
	}

	normalized, bundleHash, meta, err := normalizeBundle(slug, input.Files)
	if err != nil {
		return ReplaceResult{}, err
	}

	targetURI := uri.BuildSkill(slug)
	now := time.Now().UTC()
	result := ReplaceResult{URI: targetURI}

	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var active model.Skill
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("slug = ? AND superseded_at IS NULL", slug).
			Take(&active).Error
		version := 1
		switch {
		case err == gorm.ErrRecordNotFound:
		case err != nil:
			return fmt.Errorf("load active skill: %w", err)
		default:
			if active.BundleSHA256 == bundleHash {
				result.Version = active.Version
				result.NoOp = true
				return nil
			}
			if err := tx.Model(&model.Skill{}).
				Where("id = ?", active.ID).
				Update("superseded_at", now).Error; err != nil {
				return fmt.Errorf("supersede skill: %w", err)
			}
			version = active.Version + 1
		}

		id, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("generate skill id: %w", err)
		}
		row := model.Skill{
			ID:           id,
			Slug:         slug,
			URI:          targetURI,
			Version:      version,
			Name:         meta.Name,
			Description:  meta.Description,
			Tags:         pgarray.TextArray(meta.Tags),
			BundleSHA256: bundleHash,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := tx.Create(&row).Error; err != nil {
			return fmt.Errorf("insert skill: %w", err)
		}

		for _, f := range normalized {
			fid, err := uuid.NewV7()
			if err != nil {
				return fmt.Errorf("generate file id: %w", err)
			}
			fileRow := model.SkillFile{
				ID:            fid,
				SkillID:       id,
				RelPath:       f.Path,
				Content:       f.Content,
				ByteSize:      len([]byte(f.Content)),
				ContentSHA256: sha256Hex([]byte(f.Content)),
				CreatedAt:     now,
			}
			if err := tx.Create(&fileRow).Error; err != nil {
				return fmt.Errorf("insert skill file %q: %w", f.Path, err)
			}
		}

		result.Version = version
		return nil
	})
	if err != nil {
		return ReplaceResult{}, err
	}
	return result, nil
}

// ListCatalog returns active skills for tier-1 discovery.
func ListCatalog(ctx context.Context, db *gorm.DB) ([]CatalogEntry, error) {
	var rows []model.Skill
	if err := db.WithContext(ctx).
		Where("superseded_at IS NULL").
		Order("name ASC").
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list skills: %w", err)
	}
	out := make([]CatalogEntry, 0, len(rows))
	for _, r := range rows {
		out = append(out, CatalogEntry{
			URI:         r.URI,
			Name:        r.Name,
			Description: r.Description,
			Tags:        append([]string(nil), []string(r.Tags)...),
		})
	}
	return out, nil
}

// LoadActive loads the active skill row by slug.
func LoadActive(ctx context.Context, db *gorm.DB, slug string) (model.Skill, error) {
	var row model.Skill
	if err := db.WithContext(ctx).
		Where("slug = ? AND superseded_at IS NULL", slug).
		Take(&row).Error; err != nil {
		return model.Skill{}, err
	}
	return row, nil
}

// LoadFiles loads all files for a skill version.
func LoadFiles(ctx context.Context, db *gorm.DB, skillID uuid.UUID) ([]model.SkillFile, error) {
	var files []model.SkillFile
	if err := db.WithContext(ctx).
		Where("skill_id = ?", skillID).
		Order("rel_path ASC").
		Find(&files).Error; err != nil {
		return nil, fmt.Errorf("load skill files: %w", err)
	}
	return files, nil
}

// ReadFile returns content at relPath within an active skill.
func ReadFile(ctx context.Context, db *gorm.DB, slug, relPath string) (string, error) {
	skill, err := LoadActive(ctx, db, slug)
	if err != nil {
		return "", fmt.Errorf("load skill: %w", err)
	}
	relPath = strings.TrimSpace(relPath)
	if relPath == "" {
		relPath = ManifestPath
	}
	var file model.SkillFile
	if err := db.WithContext(ctx).
		Where("skill_id = ? AND rel_path = ?", skill.ID, relPath).
		Take(&file).Error; err != nil {
		return "", fmt.Errorf("load file %q: %w", relPath, err)
	}
	return file.Content, nil
}

// ListTreeChildren lists immediate children under a skill path prefix for inspect tree.
func ListTreeChildren(ctx context.Context, db *gorm.DB, slug string, prefix string) ([]string, error) {
	files, err := loadActiveFiles(ctx, db, slug)
	if err != nil {
		return nil, err
	}
	prefix = strings.Trim(prefix, "/")

	type child struct {
		name  string
		isDir bool
	}
	children := map[string]child{}

	for _, f := range files {
		rel := f.RelPath
		rest := rel
		if prefix != "" {
			if rel == prefix {
				continue
			}
			if !strings.HasPrefix(rel, prefix+"/") {
				continue
			}
			rest = strings.TrimPrefix(rel, prefix+"/")
		}
		if rest == "" {
			continue
		}
		part := rest
		if i := strings.Index(rest, "/"); i >= 0 {
			part = rest[:i]
			children[part] = child{name: part, isDir: true}
		} else if _, ok := children[part]; !ok {
			children[part] = child{name: part, isDir: false}
		}
	}

	names := make([]string, 0, len(children))
	for n := range children {
		names = append(names, n)
	}
	sort.Strings(names)

	out := make([]string, 0, len(names))
	for _, n := range names {
		ch := children[n]
		p := n
		if prefix != "" {
			p = prefix + "/" + n
		}
		if ch.isDir {
			out = append(out, uri.BuildSkill(slug, p)+"/")
		} else {
			out = append(out, uri.BuildSkill(slug, p))
		}
	}
	return out, nil
}

// GetDetail returns metadata, tree, and all file contents for browse/UI.
func GetDetail(ctx context.Context, db *gorm.DB, slug string) (Detail, error) {
	skill, err := LoadActive(ctx, db, slug)
	if err != nil {
		return Detail{}, err
	}
	files, err := LoadFiles(ctx, db, skill.ID)
	if err != nil {
		return Detail{}, err
	}
	fileMap := make(map[string]string, len(files))
	for _, f := range files {
		fileMap[f.RelPath] = f.Content
	}
	return Detail{
		Skill: skillMetaFrom(skill),
		Tree:  BuildFileTree(files),
		Files: fileMap,
	}, nil
}

// SkillManifestBody returns SKILL.md for meta/cat default.
func SkillManifestBody(ctx context.Context, db *gorm.DB, slug string) (string, error) {
	return ReadFile(ctx, db, slug, ManifestPath)
}

func skillMetaFrom(s model.Skill) SkillMeta {
	return SkillMeta{
		URI:          s.URI,
		Slug:         s.Slug,
		Name:         s.Name,
		Description:  s.Description,
		Tags:         append([]string(nil), []string(s.Tags)...),
		Version:      s.Version,
		BundleSHA256: s.BundleSHA256,
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
	}
}

func loadActiveFiles(ctx context.Context, db *gorm.DB, slug string) ([]model.SkillFile, error) {
	skill, err := LoadActive(ctx, db, slug)
	if err != nil {
		return nil, err
	}
	return LoadFiles(ctx, db, skill.ID)
}

// BuildFileTree builds a nested tree from flat file paths.
func BuildFileTree(files []model.SkillFile) []FileNode {
	type node struct {
		name     string
		path     string
		typ      string
		children map[string]*node
	}
	root := map[string]*node{}
	for _, f := range files {
		parts := strings.Split(f.RelPath, "/")
		cur := root
		var built string
		for i, part := range parts {
			if i == 0 {
				built = part
			} else {
				built = built + "/" + part
			}
			if cur[part] == nil {
				typ := "dir"
				if i == len(parts)-1 {
					typ = "file"
				}
				cur[part] = &node{name: part, path: built, typ: typ, children: map[string]*node{}}
			}
			if i < len(parts)-1 {
				cur[part].typ = "dir"
				cur = cur[part].children
			}
		}
	}
	var convert func(m map[string]*node) []FileNode
	convert = func(m map[string]*node) []FileNode {
		names := make([]string, 0, len(m))
		for n := range m {
			names = append(names, n)
		}
		sort.Strings(names)
		out := make([]FileNode, 0, len(names))
		for _, n := range names {
			nd := m[n]
			fn := FileNode{Name: nd.name, Path: nd.path, Type: nd.typ}
			if nd.typ == "dir" && len(nd.children) > 0 {
				fn.Children = convert(nd.children)
			}
			out = append(out, fn)
		}
		return out
	}
	return convert(root)
}

func normalizeBundle(slug string, files []FileInput) ([]FileInput, string, frontmatter, error) {
	if len(files) == 0 {
		return nil, "", frontmatter{}, fmt.Errorf("bundle must include at least one file")
	}

	byPath := make(map[string]string, len(files))
	total := 0
	for _, f := range files {
		rel, err := normalizeRelPath(f.Path)
		if err != nil {
			return nil, "", frontmatter{}, err
		}
		if !utf8.ValidString(f.Content) {
			return nil, "", frontmatter{}, fmt.Errorf("file %q is not valid UTF-8 text", rel)
		}
		if strings.Contains(f.Content, "\x00") {
			return nil, "", frontmatter{}, fmt.Errorf("file %q appears to be binary", rel)
		}
		size := len([]byte(f.Content))
		if size > maxFileBytes {
			return nil, "", frontmatter{}, fmt.Errorf("file %q exceeds %d byte limit", rel, maxFileBytes)
		}
		total += size
		if total > maxBundleBytes {
			return nil, "", frontmatter{}, fmt.Errorf("bundle exceeds %d byte limit", maxBundleBytes)
		}
		if _, dup := byPath[rel]; dup {
			return nil, "", frontmatter{}, fmt.Errorf("duplicate path %q", rel)
		}
		byPath[rel] = f.Content
	}

	body, ok := byPath[ManifestPath]
	if !ok {
		return nil, "", frontmatter{}, fmt.Errorf("bundle must include %s", ManifestPath)
	}

	meta, err := parseFrontmatter(body)
	if err != nil {
		return nil, "", frontmatter{}, err
	}
	if strings.TrimSpace(meta.Description) == "" {
		return nil, "", frontmatter{}, fmt.Errorf("%s: description is required", ManifestPath)
	}
	if meta.Name != "" && meta.Name != slug {
		return nil, "", frontmatter{}, fmt.Errorf("frontmatter name %q does not match slug %q", meta.Name, slug)
	}
	if meta.Name == "" {
		meta.Name = slug
	}

	normalized := make([]FileInput, 0, len(byPath))
	paths := make([]string, 0, len(byPath))
	for p := range byPath {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		normalized = append(normalized, FileInput{Path: p, Content: byPath[p]})
	}

	hash := bundleSHA256(normalized)
	return normalized, hash, meta, nil
}

func normalizeRelPath(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "/")
	raw = path.Clean(raw)
	if raw == "." || raw == "" {
		return "", fmt.Errorf("invalid file path")
	}
	if strings.HasPrefix(raw, "..") || strings.Contains(raw, "/..") {
		return "", fmt.Errorf("invalid file path %q", raw)
	}
	ext := strings.ToLower(path.Ext(raw))
	if ext == "" && raw != ManifestPath {
		return "", fmt.Errorf("file %q has no allowed extension", raw)
	}
	if ext != "" {
		if _, ok := allowedExtensions[ext]; !ok {
			return "", fmt.Errorf("extension %q not allowed for %q", ext, raw)
		}
	}
	return raw, nil
}

func parseFrontmatter(body string) (frontmatter, error) {
	body = strings.TrimPrefix(body, "\uFEFF")
	if !strings.HasPrefix(body, "---") {
		return frontmatter{}, fmt.Errorf("%s: missing YAML frontmatter", ManifestPath)
	}
	rest := body[3:]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return frontmatter{}, fmt.Errorf("%s: unclosed frontmatter", ManifestPath)
	}
	yamlBlock := strings.TrimSpace(rest[:end])
	lines := strings.Split(yamlBlock, "\n")

	var meta frontmatter
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		val = strings.Trim(val, `"'`)

		switch key {
		case "name":
			meta.Name = val
		case "description":
			if isYAMLBlockScalar(val) {
				i++
				var parts []string
				for i < len(lines) {
					next := lines[i]
					if strings.TrimSpace(next) == "" {
						i++
						continue
					}
					if !strings.HasPrefix(next, " ") && !strings.HasPrefix(next, "\t") {
						break
					}
					parts = append(parts, strings.TrimSpace(next))
					i++
				}
				i--
				meta.Description = strings.Join(parts, " ")
			} else {
				meta.Description = val
			}
		case "tags":
			tags, err := parseSkillTags(val)
			if err != nil {
				return frontmatter{}, err
			}
			meta.Tags = tags
		}
	}

	if strings.TrimSpace(meta.Name) != "" {
		if err := uri.ValidateSkillName(strings.TrimSpace(meta.Name)); err != nil {
			return frontmatter{}, err
		}
		meta.Name = strings.TrimSpace(meta.Name)
	}
	meta.Description = strings.TrimSpace(meta.Description)
	return meta, nil
}

func isYAMLBlockScalar(val string) bool {
	switch strings.TrimSpace(val) {
	case ">-", ">", "|-", "|", "":
		return true
	default:
		return false
	}
}

func parseSkillTags(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, p := range parts {
		tag := strings.TrimSpace(p)
		if tag == "" {
			continue
		}
		if err := ValidateSkillTag(tag); err != nil {
			return nil, err
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		out = append(out, tag)
		if len(out) > 8 {
			return nil, fmt.Errorf("%s: at most 8 tags allowed", ManifestPath)
		}
	}
	sort.Strings(out)
	return out, nil
}

// ValidateSkillTag checks a skill tag (lowercase [a-z0-9-], 1-32 chars).
func ValidateSkillTag(tag string) error {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return fmt.Errorf("%s: empty tag", ManifestPath)
	}
	if len(tag) > 32 {
		return fmt.Errorf("%s: tag %q too long", ManifestPath, tag)
	}
	for i, r := range tag {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
			if i == 0 {
				return fmt.Errorf("%s: tag %q must start with a letter", ManifestPath, tag)
			}
		case r == '-':
			if i == 0 || i == len(tag)-1 {
				return fmt.Errorf("%s: tag %q must not start or end with hyphen", ManifestPath, tag)
			}
		default:
			return fmt.Errorf("%s: invalid tag %q", ManifestPath, tag)
		}
	}
	return nil
}

func bundleSHA256(files []FileInput) string {
	h := sha256.New()
	for _, f := range files {
		h.Write([]byte(f.Path))
		h.Write([]byte{0})
		h.Write([]byte(sha256Hex([]byte(f.Content))))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
