package app

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// validate checks the answers on Start (INT paths, "Validation, in order").
// The provider was checked by the reducer against detection; the project is
// resolved, then checked on disk.
func (m *Model) validate() tea.Cmd {
	id, raw, env := m.s.Startup.Provider, m.s.Startup.Project, m.env
	name := ""
	for _, c := range supported {
		if c.id == id {
			name = c.name
		}
	}
	return func() tea.Msg {
		p, branch, problem := checkProject(raw, env)
		if problem != "" {
			return actionMsg{state.ValidationFailed{Field: "project", Message: problem}}
		}
		return actionMsg{state.ValidationOK{Provider: id, ProviderName: name, Project: p, Branch: branch}}
	}
}

// checkProject resolves raw (steps 1 to 3), checks it is an existing
// directory (step 4) the user can list (step 5), and reads its git branch.
// The messages are ui-design.md's, Validation and messages.
func checkProject(raw string, env input.PathEnv) (path, branch, problem string) {
	r := input.ResolvePath(raw, env)
	if r.Problem != "" {
		return "", "", r.Problem
	}
	p := r.Path
	fi, err := os.Stat(p)
	switch {
	case errors.Is(err, fs.ErrPermission):
		return "", "", "Not readable (permission denied): " + p
	case errors.Is(err, fs.ErrNotExist):
		msg := "Does not exist: " + p
		if r.Translated {
			msg += " (the WSL mount root may not be /mnt/)"
		}
		return "", "", msg
	case err != nil:
		return "", "", "Not readable: " + p + ": " + cause(err)
	case !fi.IsDir():
		return "", "", "Is a file, not a directory: " + p
	}
	d, err := os.Open(p)
	if err == nil {
		_, err = d.ReadDir(1)
		d.Close()
		if err == io.EOF {
			err = nil // an empty directory
		}
	}
	switch {
	case errors.Is(err, fs.ErrPermission):
		return "", "", "Not readable (permission denied): " + p
	case err != nil:
		return "", "", "Not readable: " + p + ": " + cause(err)
	}
	return p, readBranch(p), ""
}

// cause is err without the path a *PathError repeats.
func cause(err error) string {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		return pe.Err.Error()
	}
	return err.Error()
}

// readBranch returns the branch of the git work tree holding dir, read from
// .git/HEAD without running git (ui-design.md, Project bar): the branch
// name, the short commit of a detached head, or "" when dir is in no work
// tree or HEAD cannot be read. Display only: a wrong or missing value changes
// nothing else.
func readBranch(dir string) string {
	for d := dir; ; {
		g := filepath.Join(d, ".git")
		if fi, err := os.Stat(g); err == nil {
			head := filepath.Join(g, "HEAD")
			if !fi.IsDir() {
				// A worktree or submodule: .git is a file "gitdir: <path>".
				s, ok := strings.CutPrefix(readSmall(g), "gitdir:")
				if !ok {
					return ""
				}
				gd := strings.TrimSpace(s)
				if !filepath.IsAbs(gd) {
					gd = filepath.Join(d, gd)
				}
				head = filepath.Join(gd, "HEAD")
			}
			return parseHead(readSmall(head))
		}
		up := filepath.Dir(d)
		if up == d {
			return ""
		}
		d = up
	}
}

// parseHead reads HEAD's content: "ref: refs/heads/<branch>", or a commit id.
func parseHead(s string) string {
	s = strings.TrimSpace(s)
	if ref, ok := strings.CutPrefix(s, "ref:"); ok {
		ref = strings.TrimSpace(ref)
		if b, ok := strings.CutPrefix(ref, "refs/heads/"); ok {
			return b
		}
		return ref
	}
	if (len(s) == 40 || len(s) == 64) && strings.Trim(s, "0123456789abcdef") == "" {
		return s[:7]
	}
	return ""
}

// readSmall reads at most 4 KiB of a file: HEAD and a .git file are a line.
func readSmall(p string) string {
	f, err := os.Open(p)
	if err != nil {
		return ""
	}
	defer f.Close()
	b, _ := io.ReadAll(io.LimitReader(f, 4096))
	return string(b)
}
