// Package confcheck is the configuration check of doc/reliability.md
// ("Configuration untouched"): a test and probe tool, never part of the
// running application.
//
// Rules it enforces: credential-named files are never opened or stat-read;
// the only code that reads a file's bytes is Hasher (SHA-256 over the whole
// file); no content is printed or stored, only names and digests. The one
// exception is the key-level diff of ~/.claude.json (keys.go, the owner's
// approval, 2026-10-17).
package confcheck

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Class of a file.
type Class string

const (
	ClassConfig  Class = "configuration"
	ClassMixed   Class = "mixed"
	ClassSession Class = "session"
)

// Entry states recorded in a baseline.
const (
	StateHashed  = "hashed"
	StateAbsent  = "absent"
	StateSkipped = "skipped"
)

// Hasher returns the hex SHA-256 of the whole file. It is the only code that
// sees file bytes. Injectable so tests can prove which paths were opened.
type Hasher func(path string) (string, error)

// ProcessLister reports the image names of running processes.
type ProcessLister interface {
	Names() ([]string, error)
}

// Entry is one file in a snapshot: names and digests only.
type Entry struct {
	Class  Class  `json:"class"`
	Name   string `json:"name"`
	State  string `json:"state"`
	Digest string `json:"digest,omitempty"`

	// Key-level data for .claude.json only (keys.go): digests of compacted
	// values by key name; Skipped lists sensitive names, never hashed.
	Parsed   bool                         `json:"parsed,omitempty"`
	Keys     map[string]string            `json:"keys,omitempty"`
	Projects map[string]map[string]string `json:"projects,omitempty"`
	Skipped  []string                     `json:"skippedKeys,omitempty"`
}

// Snapshot is the baseline: no file content.
type Snapshot struct {
	Time     time.Time `json:"time"`
	Claude   bool      `json:"claudeRunning"`
	Codex    bool      `json:"codexRunning"`
	Project  string    `json:"project,omitempty"`
	Entries  []Entry   `json:"entries"`
	Sessions []string  `json:"sessionFiles"`
}

// Checker takes snapshots. Home is injectable for tests.
type Checker struct {
	Home    string
	Project string
	Hash    Hasher
	Read    func(path string) ([]byte, error) // .claude.json only; default os.ReadFile
	Procs   ProcessLister
	Now     func() time.Time
}

var credentialName = regexp.MustCompile(`(?i)credentials|auth|oauth|token|secret`)

// IsCredentialName reports whether the base name matches the skip rule.
func IsCredentialName(path string) bool {
	return credentialName.MatchString(filepath.Base(path))
}

// FileHash is the default Hasher: io.Copy of the whole file into SHA-256.
func FileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

var fixedConfig = []string{
	".claude/settings.json", ".claude/settings.local.json", ".claude/CLAUDE.md",
	".claude/remote-settings.json", ".claude/keybindings.json",
	".codex/config.toml", ".codex/AGENTS.md", ".codex/.codex-global-state.json",
}

var mixedFiles = []string{".claude.json", ".claude/plugins/known_marketplaces.json"}

// walkFiles lists regular files under root up to maxDepth directory levels
// below root (files directly in root are depth 1; 0 means unlimited).
// Listing never opens files.
func walkFiles(root string, maxDepth int, keep func(string) bool) []string {
	var out []string
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		depth := 0
		if rel != "." {
			depth = strings.Count(rel, string(filepath.Separator)) + 1
		}
		if d.IsDir() {
			if maxDepth > 0 && depth >= maxDepth {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type().IsRegular() && (keep == nil || keep(p)) {
			out = append(out, p)
		}
		return nil
	})
	return out
}

func hasExt(exts ...string) func(string) bool {
	return func(p string) bool {
		e := strings.ToLower(filepath.Ext(p))
		for _, x := range exts {
			if e == x {
				return true
			}
		}
		return false
	}
}

// display elides the home folder: C:\Users\…\.claude\settings.json.
func (c *Checker) display(path string) string {
	if rel, ok := within(c.Home, path); ok {
		return `C:\Users\…\` + rel
	}
	if c.Project != "" {
		if rel, ok := within(c.Project, path); ok {
			return `<project>\` + rel
		}
	}
	return filepath.Base(path)
}

func within(root, path string) (string, bool) {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return "", false
	}
	return rel, true
}

// Snapshot hashes every checked file and lists session file names.
func (c *Checker) Snapshot() (*Snapshot, error) {
	hash := c.Hash
	if hash == nil {
		hash = FileHash
	}
	now := time.Now
	if c.Now != nil {
		now = c.Now
	}
	s := &Snapshot{Time: now().UTC(), Project: c.Project}
	if c.Procs != nil {
		names, err := c.Procs.Names()
		if err != nil {
			return nil, fmt.Errorf("listing processes: %w", err)
		}
		for _, n := range names {
			switch strings.ToLower(n) {
			case "claude.exe":
				s.Claude = true
			case "codex.exe":
				s.Codex = true
			}
		}
	}

	type cand struct {
		path  string
		class Class
	}
	seen := map[string]bool{}
	var cands []cand
	add := func(p string, cl Class) {
		if !seen[p] {
			seen[p] = true
			cands = append(cands, cand{p, cl})
		}
	}
	h := func(rel string) string { return filepath.Join(c.Home, filepath.FromSlash(rel)) }
	for _, r := range mixedFiles {
		add(h(r), ClassMixed)
	}
	for _, r := range fixedConfig {
		add(h(r), ClassConfig)
	}
	for _, d := range []string{".claude/skills", ".claude/agents", ".claude/commands", ".claude/hooks", ".claude/plugins"} {
		for _, p := range walkFiles(h(d), 2, hasExt(".json")) {
			add(p, ClassConfig)
		}
	}
	for _, d := range []string{".claude/skills", ".claude/agents", ".claude/commands", ".codex/plugins"} {
		for _, p := range walkFiles(h(d), 3, hasExt(".md", ".toml")) {
			add(p, ClassConfig)
		}
	}
	if c.Project != "" {
		for _, d := range []string{".claude", ".codex"} {
			for _, p := range walkFiles(filepath.Join(c.Project, d), 0, nil) {
				add(p, ClassConfig)
			}
		}
	}

	for _, cd := range cands {
		e := Entry{Class: cd.class, Name: c.display(cd.path)}
		if IsCredentialName(cd.path) { // before any open or stat
			e.State = StateSkipped
		} else if d, err := hash(cd.path); err == nil {
			e.State, e.Digest = StateHashed, d
			if cd.path == h(".claude.json") {
				c.keyLevel(&e, cd.path)
			}
		} else if errors.Is(err, fs.ErrNotExist) {
			e.State = StateAbsent
		} else {
			return nil, fmt.Errorf("hashing %s: %w", e.Name, err)
		}
		s.Entries = append(s.Entries, e)
	}
	sort.Slice(s.Entries, func(i, j int) bool { return s.Entries[i].Name < s.Entries[j].Name })

	// Session data: names only, never hashed or opened.
	for _, d := range []string{".claude/projects", ".codex/sessions", ".codex/log"} {
		for _, p := range walkFiles(h(d), 0, nil) {
			if !IsCredentialName(p) {
				s.Sessions = append(s.Sessions, c.display(p))
			}
		}
	}
	sort.Strings(s.Sessions)
	return s, nil
}

// Verdict of a comparison.
type Verdict string

const (
	Pass        Verdict = "PASS"
	Fail        Verdict = "FAIL"
	NotVerified Verdict = "NOT-VERIFIED"
)

// ExitCode: 0 PASS, 1 FAIL, 2 NOT-VERIFIED.
func (v Verdict) ExitCode() int {
	switch v {
	case Pass:
		return 0
	case Fail:
		return 1
	}
	return 2
}

// Report is the result of Compare.
type Report struct {
	Lines       []string
	NewSessions int
	CLIRan      bool
	Verdict     Verdict
}

func (r *Report) String() string {
	var b strings.Builder
	for _, l := range r.Lines {
		b.WriteString(l + "\n")
	}
	fmt.Fprintf(&b, "session data: %d new file(s) (count only)\n", r.NewSessions)
	fmt.Fprintf(&b, "cli ran during window: %t\n", r.CLIRan)
	fmt.Fprintf(&b, "configuration: %s\n", r.Verdict)
	return b.String()
}

// Compare diffs two snapshots. cliRan forces the "a CLI ran" answer.
func Compare(before, after *Snapshot, cliRan bool) *Report {
	r := &Report{CLIRan: cliRan || before.Claude || before.Codex || after.Claude || after.Codex}
	idx := func(s *Snapshot) map[string]Entry {
		m := map[string]Entry{}
		for _, e := range s.Entries {
			m[string(e.Class)+"|"+e.Name] = e
		}
		return m
	}
	b, a := idx(before), idx(after)
	keys := map[string]bool{}
	for k := range b {
		keys[k] = true
	}
	for k := range a {
		keys[k] = true
	}
	sorted := make([]string, 0, len(keys))
	for k := range keys {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)

	fail, mixed := false, false
	for _, k := range sorted {
		be, bok := b[k]
		ae, aok := a[k]
		e := ae
		if !aok {
			e = be
		}
		var res string
		diff, keyLevel := false, false
		switch {
		case be.State == StateSkipped || ae.State == StateSkipped:
			res = "skipped (credential name)"
		case !bok || be.State == StateAbsent:
			if !aok || ae.State == StateAbsent {
				res = "absent"
			} else {
				res, diff = "appeared", true
			}
		case !aok || ae.State == StateAbsent:
			res, diff = "disappeared", true
		case be.Digest == ae.Digest:
			res = "identical"
		default:
			res, diff = "changed", true
			keyLevel = be.Parsed && ae.Parsed
		}
		var extra []string
		if diff {
			if e.Class == ClassMixed {
				if keyLevel {
					project := after.Project
					if project == "" {
						project = before.Project
					}
					var f, m bool
					extra, f, m = diffClaudeJSON(e.Name, be, ae, project, r.CLIRan)
					fail, mixed = fail || f, mixed || m
					res = "changed (key-level diff follows)"
				} else {
					if res == "changed" {
						res = "changed, not classified (contents not inspected)"
					}
					mixed = true
				}
			} else {
				fail = true
			}
		}
		line := fmt.Sprintf("%s  %s  %s", e.Class, e.Name, res)
		if diff && ae.Digest != "" {
			line += "  sha256=" + ae.Digest
		}
		r.Lines = append(r.Lines, line)
		r.Lines = append(r.Lines, extra...)
	}

	old := map[string]bool{}
	for _, n := range before.Sessions {
		old[n] = true
	}
	for _, n := range after.Sessions {
		if !old[n] {
			r.NewSessions++
		}
	}

	switch {
	case fail:
		r.Verdict = Fail
	case mixed && r.CLIRan:
		r.Verdict = NotVerified
	case mixed:
		r.Verdict = Fail
	default:
		r.Verdict = Pass
	}
	return r
}

// keyLevel adds the key-level digests of ~/.claude.json to e (keys.go).
// Invalid JSON (mid-write) leaves e as a whole-file entry.
func (c *Checker) keyLevel(e *Entry, path string) {
	read := c.Read
	if read == nil {
		read = os.ReadFile
	}
	data, err := read(path)
	if err != nil {
		return
	}
	keys, projects, skipped, ok := keyDigests(data)
	if !ok {
		return
	}
	e.Parsed, e.Keys, e.Projects, e.Skipped = true, keys, projects, skipped
}
