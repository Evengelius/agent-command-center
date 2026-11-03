package input

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
	"time"
)

// RecentKept is how many projects that are not favourites the list keeps
// (plan 15, item 1).
const RecentKept = 10

// MaxHistory is how many sessions a project's history keeps; a newer one
// drops the oldest (plan 75, design D1).
const MaxHistory = 20

// Project is one entry of the startup screen's project list (plan 15): a
// validated project path, when a session last started in it, whether the
// owner starred it, and the sessions the dashboard ran in it.
type Project struct {
	Path      string    `json:"path"`
	Used      time.Time `json:"used"`
	Favourite bool      `json:"favourite,omitempty"`
	// History is the sessions the dashboard ran in the project, newest first,
	// at most MaxHistory, of any agent CLI (plan 75, design D1); it replaces
	// version 1's Sessions as the one record.
	History []Entry `json:"history,omitempty"`
	// Sessions is version 1's record, the last session with each agent CLI by
	// provider id (plan 69, design D2): read only to migrate a version 1 file
	// (Load) or a list built in that shape (Save); both fold it into History,
	// so it is empty in every list Load gives and is never written.
	Sessions map[string]Session `json:"sessions,omitempty"`
}

// Session is a session of one agent CLI as version 1 kept it (plan 69,
// design D2): the ID the CLI reported for it and when its last turn ended.
type Session struct {
	ID    string    `json:"id"`
	Ended time.Time `json:"ended"`
}

// Entry is one session of a project's history (plan 75, design D1): the agent
// CLI, the ID it reported, when the owner's last turn there ended, the model
// as zone A names it ("" when not recorded) and the title (design D1a; "" when
// the session has none).
type Entry struct {
	CLI   string    `json:"cli"`
	ID    string    `json:"id"`
	Ended time.Time `json:"ended"`
	Model string    `json:"model,omitempty"`
	Title string    `json:"title,omitempty"`
}

// Projects keeps the project list in projects.json, beside the drafts
// (plan 15). It holds paths, times, session IDs, the models' names and the
// sessions' titles, a title being the first words of the owner's first
// message in the session: the one conversation text it keeps, an exception
// to ASM-24 (plan 75, design D1a; brief amendment 91).
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
// never an empty list. A version 1 file is read as version 2: each project's
// last sessions become its history (plan 75, design D1); a version above 2 is
// refused, as an older build refuses version 2.
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
	if pf.Version != 1 && pf.Version != 2 {
		return nil, fmt.Errorf("%s: version %d, expected 1 or 2", f, pf.Version)
	}
	for i, e := range pf.Projects {
		if e.Path == "" {
			return nil, fmt.Errorf("%s: an entry without a path", f)
		}
		if pf.Version == 1 {
			pf.Projects[i] = e.settled()
		} else {
			pf.Projects[i] = e.read() // version 2 has no Sessions
		}
	}
	SortProjects(pf.Projects)
	return pf.Projects, nil
}

// read is a project of a version 2 file as Load gives it: no Sessions (that
// field is version 1's), no history entry without an ID (nothing to resume,
// as the migration leaves such a session out), at most MaxHistory entries, the
// newest, which the file lists first. The project it is called on is not
// changed.
func (e Project) read() Project {
	e.Sessions = nil
	e.History = slices.DeleteFunc(slices.Clone(e.History), func(x Entry) bool { return x.ID == "" })
	if len(e.History) > MaxHistory {
		e.History = e.History[:MaxHistory]
	}
	if len(e.History) == 0 {
		e.History = nil
	}
	return e
}

// settled is the project as version 2 keeps it: its Sessions, if any, folded
// into its history, newest first, an ID the history already holds kept as it
// is, a session without an ID left out (nothing to resume), at most
// MaxHistory. The project it is called on is not changed.
func (e Project) settled() Project {
	if len(e.Sessions) == 0 {
		e.Sessions = nil
		return e
	}
	h := slices.Clone(e.History)
	for cli, s := range e.Sessions {
		if s.ID != "" && !slices.ContainsFunc(h, func(x Entry) bool { return x.CLI == cli && x.ID == s.ID }) {
			h = append(h, Entry{CLI: cli, ID: s.ID, Ended: s.Ended})
		}
	}
	sort.SliceStable(h, func(i, j int) bool {
		if !h[i].Ended.Equal(h[j].Ended) {
			return h[i].Ended.After(h[j].Ended)
		}
		return h[i].CLI < h[j].CLI
	})
	if len(h) > MaxHistory {
		h = h[:MaxHistory]
	}
	e.History, e.Sessions = h, nil
	return e
}

// Save writes the list as version 2 through a temporary file renamed over the
// old one, so a failed write leaves the previous list whole.
func (p Projects) Save(list []Project) error {
	f, err := p.File()
	if err != nil {
		return err
	}
	out := make([]Project, len(list))
	for i, e := range list {
		out[i] = e.settled()
	}
	b, err := json.MarshalIndent(projectsFile{Version: 2, Projects: out}, "", "  ")
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
// in case is the same project on Windows. Its star and its history stay. Past
// RecentKept, the oldest projects that are not favourites are dropped.
func (p Projects) Record(list []Project, path string, at time.Time) []Project {
	path = filepath.Clean(path)
	out := []Project{{Path: path, Used: at}}
	for _, e := range list {
		if p.Same(e.Path, path) {
			out[0].Favourite, out[0].History = e.Favourite, e.History
			continue
		}
		out = append(out, e)
	}
	return capped(out)
}

// RecordSession puts the session s first in the history of the project at
// path (plan 75, design D1): the same agent CLI's same ID moves first, with
// its new end time and its new model when s has one, and its title kept
// (design D1a: a title is set once); a past MaxHistory drops the oldest. A
// project the list no longer has is not added back: it was dropped or
// forgotten, and its history with it.
func (p Projects) RecordSession(list []Project, path string, s Entry) []Project {
	out := slices.Clone(list)
	for i, e := range out {
		if !p.Same(e.Path, path) {
			continue
		}
		h := make([]Entry, 0, min(len(e.History)+1, MaxHistory))
		h = append(h, s)
		for _, x := range e.History {
			if x.CLI != s.CLI || x.ID != s.ID {
				h = append(h, x)
				continue
			}
			h[0].Title = x.Title // set once, whatever this record brings
			if h[0].Title == "" {
				h[0].Title = s.Title
			}
			if h[0].Model == "" {
				h[0].Model = x.Model
			}
		}
		out[i].History = h[:min(len(h), MaxHistory)]
	}
	return out
}

// ForgetSession drops the session id of agent CLI provider from the history
// of the project at path, and no other (plan 69, design D7; plan 75, design
// D4): its CLI has no such session, or the owner deleted it from the list.
// The CLI's own record is never touched.
func (p Projects) ForgetSession(list []Project, path, provider, id string) []Project {
	return p.ForgetSessions(list, path, []Entry{{CLI: provider, ID: id}})
}

// ForgetSessions drops from the history of the project at path exactly the
// sessions gone names, each by its agent CLI and ID, and no other session and
// no other project's (plan 75, design D4: Delete All removes what its question
// counted, even from a file that holds more by now).
func (p Projects) ForgetSessions(list []Project, path string, gone []Entry) []Project {
	named := func(x Entry) bool {
		return slices.ContainsFunc(gone, func(g Entry) bool { return x.CLI == g.CLI && x.ID == g.ID })
	}
	out := slices.Clone(list)
	for i, e := range out {
		if !p.Same(e.Path, path) || !slices.ContainsFunc(e.History, named) {
			continue
		}
		var h []Entry
		for _, x := range e.History {
			if !named(x) {
				h = append(h, x)
			}
		}
		out[i].History = h
	}
	return out
}

// Newest is the newest entry of the project's history that agent CLI cli ran
// with an ID, other than the session skip (""): what Ctrl+R resumes (plan 75,
// design D1); skip is a session its CLI has just refused for good.
func (e Project) Newest(cli, skip string) (Entry, bool) {
	for _, x := range e.History {
		if x.CLI == cli && x.ID != "" && x.ID != skip {
			return x, true
		}
	}
	return Entry{}, false
}

// Find is the project the list holds for path, as Same compares them.
func (p Projects) Find(list []Project, path string) (Project, bool) {
	for _, e := range list {
		if p.Same(e.Path, path) {
			return e, true
		}
	}
	return Project{}, false
}

// LastSession is provider's last session in the project at path (plan 69,
// design D3; plan 75, design D1): the project that is the same as path, as
// Same compares them, its history's newest session of provider, and whether it
// has one with an ID.
func (p Projects) LastSession(list []Project, path, provider string) (Project, Session, bool) {
	e, found := p.Find(list, path)
	if !found {
		return Project{}, Session{}, false
	}
	x, ok := e.Newest(provider, "")
	return e, Session{ID: x.ID, Ended: x.Ended}, ok
}

// Star stars path, or unstars it. A directory starred before any session
// started in it (from plan 15's browser) is added; unstarred, it goes again.
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
