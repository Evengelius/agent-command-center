package input

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// RecentKept is how many projects that are not favourites the list keeps
// (plan 15, item 1).
const RecentKept = 10

// Project is one entry of the startup screen's project list (plan 15): a
// validated project path, when a session last started in it, and whether
// the owner starred it.
type Project struct {
	Path      string    `json:"path"`
	Used      time.Time `json:"used"`
	Favourite bool      `json:"favourite,omitempty"`
}

// Projects keeps the project list in projects.json, beside the drafts
// (plan 15). It holds paths and times only, no conversation data.
type Projects struct {
	Dir string // "" is the parent of DraftDir()
}

// renameFile is os.Rename; a test makes it fail.
var renameFile = os.Rename

type projectsFile struct {
	Version  int       `json:"version"`
	Projects []Project `json:"projects"`
}

// File is the list's file: projects.json in the dashboard's own folder,
// %LOCALAPPDATA%\agent-command-center on Windows.
func (p Projects) File() (string, error) {
	dir := p.Dir
	if dir == "" {
		d, err := DraftDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Dir(d)
	}
	return filepath.Join(dir, "projects.json"), nil
}

// Load reads the list, favourites first, then by last start. No file is an
// empty list without an error; a damaged one is an error naming the file,
// never an empty list.
func (p Projects) Load() ([]Project, error) {
	f, err := p.File()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(f)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var pf projectsFile
	if err := json.Unmarshal(b, &pf); err != nil {
		return nil, fmt.Errorf("%s: %w", f, err)
	}
	if pf.Version != 1 {
		return nil, fmt.Errorf("%s: version %d, expected 1", f, pf.Version)
	}
	for _, e := range pf.Projects {
		if e.Path == "" {
			return nil, fmt.Errorf("%s: an entry without a path", f)
		}
	}
	SortProjects(pf.Projects)
	return pf.Projects, nil
}

// Save writes the list through a temporary file renamed over the old one,
// so a failed write leaves the previous list whole.
func (p Projects) Save(list []Project) error {
	f, err := p.File()
	if err != nil {
		return err
	}
	if list == nil {
		list = []Project{}
	}
	b, err := json.MarshalIndent(projectsFile{Version: 1, Projects: list}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(f), 0o700); err != nil {
		return err
	}
	t, err := os.CreateTemp(filepath.Dir(f), ".projects-*")
	if err != nil {
		return err
	}
	tmp := t.Name()
	_, werr := t.Write(append(b, '\n'))
	if err := errors.Join(werr, t.Sync(), t.Close()); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := renameFile(tmp, f); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// Update reads the list, changes it and writes it back, returning the list
// written. A list that cannot be read is not overwritten.
func (p Projects) Update(change func([]Project) []Project) ([]Project, error) {
	list, err := p.Load()
	if err != nil {
		return nil, err
	}
	list = change(list)
	if err := p.Save(list); err != nil {
		return nil, err
	}
	return list, nil
}

// Record puts path first with its start time, once: a path differing only
// in case is the same project on Windows. Past RecentKept, the oldest
// projects that are not favourites are dropped.
func (p Projects) Record(list []Project, path string, at time.Time) []Project {
	path = filepath.Clean(path)
	out := []Project{{Path: path, Used: at}}
	for _, e := range list {
		if p.Same(e.Path, path) {
			out[0].Favourite = e.Favourite
			continue
		}
		out = append(out, e)
	}
	return capped(out)
}

// Star stars path, or unstars it. A directory starred before any session
// started in it (from the browser) is added; unstarred, it goes again.
func (p Projects) Star(list []Project, path string, on bool) []Project {
	path = filepath.Clean(path)
	var out []Project
	found := false
	for _, e := range list {
		if p.Same(e.Path, path) {
			found, e.Favourite = true, on
			if !on && e.Used.IsZero() {
				continue
			}
		}
		out = append(out, e)
	}
	if !found && on {
		out = append(out, Project{Path: path, Favourite: true})
	}
	return capped(out)
}

// capped sorts the list and keeps every favourite and the RecentKept latest
// other projects.
func capped(list []Project) []Project {
	SortProjects(list)
	kept, recent := list[:0], 0
	for _, e := range list {
		if !e.Favourite {
			if recent == RecentKept {
				continue
			}
			recent++
		}
		kept = append(kept, e)
	}
	return kept
}

// Forget removes path from the list.
func (p Projects) Forget(list []Project, path string) []Project {
	var out []Project
	for _, e := range list {
		if !p.Same(e.Path, path) {
			out = append(out, e)
		}
	}
	return out
}

// Same reports whether two paths name the same project.
func (p Projects) Same(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// SortProjects orders the list: favourites first, then each group by its
// last start, the latest first.
func SortProjects(list []Project) {
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].Favourite != list[j].Favourite {
			return list[i].Favourite
		}
		return list[i].Used.After(list[j].Used)
	})
}
