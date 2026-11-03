package input

import (
	"fmt"
	"os"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

// Plan 75, design D1: the project store's version 2 and the history.

// version1 is a version 1 file as plan 69's build wrote it: a favourite
// first, then two recent projects, one with both CLIs' last sessions, one
// with Claude Code's only, one with a session the file holds without an ID.
const version1 = `{
  "version": 1,
  "projects": [
    {
      "path": "C:\\work\\fav",
      "used": "2026-10-18T12:00:00Z",
      "favourite": true,
      "sessions": {
        "claude-code": {"id": "c-fav", "ended": "2026-10-18T11:00:00Z"},
        "codex": {"id": "x-fav", "ended": "2026-10-18T11:30:00Z"}
      }
    },
    {
      "path": "C:\\work\\b",
      "used": "2026-10-17T12:00:00Z",
      "sessions": {"claude-code": {"id": "c-b", "ended": "2026-10-17T10:00:00Z"}}
    },
    {
      "path": "C:\\work\\c",
      "used": "2026-10-16T12:00:00Z",
      "sessions": {"codex": {"id": "", "ended": "2026-10-16T10:00:00Z"}}
    }
  ]
}
`

func writeFile(t *testing.T, ps Projects, body string) string {
	t.Helper()
	f, err := ps.File()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return f
}

// TestProjectsMigration is criterion 75.1 for a version 1 file: each last
// session an entry of its project's history, newest first, with no model and
// no title; the paths, times, favourites and order as they were; written as
// version 2 at the next change, with no trace of "sessions".
func TestProjectsMigration(t *testing.T) {
	ps := Projects{Dir: t.TempDir()}
	f := writeFile(t, ps, version1)
	got, err := ps.Load()
	if err != nil {
		t.Fatal(err)
	}
	want := []Project{
		{Path: `C:\work\fav`, Used: time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC), Favourite: true, History: []Entry{
			{CLI: "codex", ID: "x-fav", Ended: time.Date(2026, 10, 18, 11, 30, 0, 0, time.UTC)},
			{CLI: "claude-code", ID: "c-fav", Ended: time.Date(2026, 10, 18, 11, 0, 0, 0, time.UTC)},
		}},
		{Path: `C:\work\b`, Used: time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC), History: []Entry{
			{CLI: "claude-code", ID: "c-b", Ended: time.Date(2026, 10, 17, 10, 0, 0, 0, time.UTC)},
		}},
		{Path: `C:\work\c`, Used: time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("migrated:\n got %+v\nwant %+v", got, want)
	}
	for _, e := range got {
		for _, h := range e.History {
			if h.Model != "" || h.Title != "" {
				t.Errorf("a migrated entry has a model or a title: %+v", h)
			}
		}
	}
	if b, _ := os.ReadFile(f); string(b) != version1 {
		t.Error("reading a version 1 file changed it")
	}

	// The next change writes version 2.
	if _, err := ps.Update(func(l []Project) []Project { return ps.Star(l, `C:\work\b`, true) }); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(f)
	if !strings.Contains(string(b), `"version": 2,`) || strings.Contains(string(b), `"sessions"`) || !strings.Contains(string(b), `"id": "x-fav"`) || !strings.Contains(string(b), `"id": "c-b"`) {
		t.Errorf("written:\n%s", b)
	}
	back, err := ps.Load()
	wantBack := slices.Clone(want)
	wantBack[1].Favourite = true // the change made; everything else as it was, in the same order
	if err != nil || !reflect.DeepEqual(back, wantBack) {
		t.Fatalf("read back: got %+v, %v; want %+v", back, err, wantBack)
	}
}

// TestProjectsMigrationKeepsFavourite is the migration's mutants' ground: a
// favourite stays a favourite and every session stays, however many there
// are.
func TestProjectsMigrationKeepsFavourite(t *testing.T) {
	ps := Projects{Dir: t.TempDir()}
	writeFile(t, ps, version1)
	got, err := ps.Load()
	if err != nil {
		t.Fatal(err)
	}
	fav, n := 0, 0
	for _, e := range got {
		if e.Favourite {
			fav++
		}
		n += len(e.History)
	}
	if fav != 1 || !got[0].Favourite || n != 3 {
		t.Errorf("%d favourites, %d entries: %+v", fav, n, got)
	}
}

// TestProjectsVersion3 is criterion 75.1's refusal: a file of a version
// above 2 is refused by Load, and Update neither changes the list nor writes
// anything: the file stays byte for byte.
func TestProjectsVersion3(t *testing.T) {
	ps := Projects{Dir: t.TempDir()}
	body := `{"version":3,"projects":[{"path":"C:\\work\\a","used":"2026-10-18T12:00:00Z","history":[{"cli":"codex","id":"x","ended":"2026-10-18T12:00:00Z"}]}]}`
	f := writeFile(t, ps, body)
	if got, err := ps.Load(); err == nil || got != nil || !strings.Contains(err.Error(), f) || !strings.Contains(err.Error(), "version 3") {
		t.Errorf("Load: %v, %v", got, err)
	}
	called := false
	if _, err := ps.Update(func(l []Project) []Project { called = true; return nil }); err == nil || called {
		t.Errorf("Update: error %v, change called %t", err, called)
	}
	if err := os.WriteFile(f+".keep", []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(f); string(b) != body {
		t.Errorf("the file changed:\n%s", b)
	}
	if entries, _ := os.ReadDir(ps.Dir); len(entries) != 2 {
		t.Errorf("files %v: want projects.json and the copy only", entries)
	}
}

// TestHistoryRecord is criterion 75.1 for the record: a session first; the
// same ID again moves first with its new end time and model and the title it
// had; a 21st entry drops the oldest, the same ID again dropping none.
func TestHistoryRecord(t *testing.T) {
	var p Projects
	list := p.Record(nil, `C:\work\a`, t0)
	at := func(i int) time.Time { return t0.Add(time.Duration(i) * time.Minute) }
	list = p.RecordSession(list, `C:\work\a`, Entry{CLI: "claude-code", ID: "s0", Ended: at(0), Model: "Claude Haiku 5.5", Title: "first words"})
	for i := 1; i < MaxHistory; i++ {
		list = p.RecordSession(list, `C:\work\a`, Entry{CLI: "codex", ID: fmt.Sprintf("s%d", i), Ended: at(i)})
	}
	if h := list[0].History; len(h) != MaxHistory || h[0].ID != "s19" || h[MaxHistory-1].ID != "s0" {
		t.Fatalf("%d entries, first %q, last %q", len(h), h[0].ID, h[len(h)-1].ID)
	}
	// The same ID again: first, no entry dropped, the title as it was.
	list = p.RecordSession(list, `C:\work\a`, Entry{CLI: "claude-code", ID: "s0", Ended: at(30), Model: "Claude Opus 5.5", Title: "other words"})
	h := list[0].History
	if len(h) != MaxHistory || h[0].ID != "s0" || !h[0].Ended.Equal(at(30)) || h[0].Model != "Claude Opus 5.5" || h[0].Title != "first words" {
		t.Fatalf("again: %d entries, first %+v", len(h), h[0])
	}
	if h[1].ID != "s19" || h[MaxHistory-1].ID != "s1" {
		t.Errorf("the others moved: %q ... %q", h[1].ID, h[MaxHistory-1].ID)
	}
	// A record without a model keeps the one there was; a title is set once
	// but may come with a later record of an ID that had none.
	list = p.RecordSession(list, `C:\work\a`, Entry{CLI: "claude-code", ID: "s0", Ended: at(31)})
	if h = list[0].History; h[0].Model != "Claude Opus 5.5" || h[0].Title != "first words" {
		t.Errorf("a record without model or title: %+v", h[0])
	}
	list = p.RecordSession(list, `C:\work\a`, Entry{CLI: "codex", ID: "s5", Ended: at(32), Title: "late title"})
	if h = list[0].History; h[0].ID != "s5" || h[0].Title != "late title" || len(h) != MaxHistory {
		t.Errorf("an ID with no title yet: %+v", h[0])
	}
	// The 21st entry drops the oldest.
	list = p.RecordSession(list, `C:\work\a`, Entry{CLI: "codex", ID: "s99", Ended: at(40)})
	h = list[0].History
	if len(h) != MaxHistory || h[0].ID != "s99" || h[MaxHistory-1].ID != "s2" {
		t.Errorf("the 21st: %d entries, last %q", len(h), h[len(h)-1].ID)
	}
	for _, x := range h {
		if x.ID == "s1" {
			t.Error("the oldest entry was kept past the cap")
		}
	}
	// The same ID under another CLI is another session.
	list = p.RecordSession(list, `C:\work\a`, Entry{CLI: "claude-code", ID: "s99", Ended: at(41)})
	if h = list[0].History; h[0].CLI != "claude-code" || h[1].CLI != "codex" || h[1].ID != "s99" {
		t.Errorf("one ID under two CLIs: %+v %+v", h[0], h[1])
	}
}

// TestHistoryNewest is Ctrl+R's choice (design D1): the newest entry of the
// CLI with an ID, but for the session skipped.
func TestHistoryNewest(t *testing.T) {
	e := Project{Path: `C:\work\a`, History: []Entry{
		{CLI: "codex", ID: "x2"}, {CLI: "claude-code", ID: "c2"}, {CLI: "codex", ID: "x1"}, {CLI: "claude-code", ID: "c1"}, {CLI: "claude-code", ID: ""},
	}}
	for _, c := range []struct{ cli, skip, want string }{
		{"claude-code", "", "c2"}, {"claude-code", "c2", "c1"}, {"codex", "", "x2"}, {"codex", "x2", "x1"}, {"codex", "x1", "x2"}, {"gemini", "", ""},
	} {
		if x, ok := e.Newest(c.cli, c.skip); x.ID != c.want || ok != (c.want != "") {
			t.Errorf("%s skipping %q: %q, %t, want %q", c.cli, c.skip, x.ID, ok, c.want)
		}
	}
	if _, ok := e.Newest("claude-code", "c2"); !ok {
		t.Error("an entry with no ID was taken for the newest or the one before was lost")
	}
}

// TestHistoryDelete is criterion 75.7 for the store: deleting one entry
// removes it alone, by its CLI and ID, in its project only; clearing a
// history empties that project's alone; the rest of the file, the other
// projects and the favourites in their order, is byte for byte what it was.
func TestHistoryDelete(t *testing.T) {
	var p Projects
	list := []Project{
		{Path: `C:\work\fav`, Used: t0, Favourite: true, History: []Entry{{CLI: "codex", ID: "same", Ended: t0}, {CLI: "claude-code", ID: "same", Ended: t0}, {CLI: "claude-code", ID: "other", Ended: t0}}},
		{Path: `C:\work\a`, Used: t0.Add(-time.Hour), History: []Entry{{CLI: "claude-code", ID: "same", Ended: t0}, {CLI: "codex", ID: "x", Ended: t0, Model: "GPT-5.5", Title: "t"}}},
		{Path: `C:\work\b`, Used: t0.Add(-2 * time.Hour)},
	}
	file := func(l []Project) string {
		ps := Projects{Dir: t.TempDir()}
		if err := ps.Save(l); err != nil {
			t.Fatal(err)
		}
		f, _ := ps.File()
		b, _ := os.ReadFile(f)
		return string(b)
	}
	ps := Projects{Dir: t.TempDir()}
	if err := ps.Save(list); err != nil {
		t.Fatal(err)
	}
	f, _ := ps.File()

	// One entry: the Claude Code "same" of project a.
	if _, err := ps.Update(func(l []Project) []Project { return p.ForgetSession(l, `C:\work\a`, "claude-code", "same") }); err != nil {
		t.Fatal(err)
	}
	want := []Project{list[0], {Path: `C:\work\a`, Used: t0.Add(-time.Hour), History: []Entry{{CLI: "codex", ID: "x", Ended: t0, Model: "GPT-5.5", Title: "t"}}}, list[2]}
	if b, _ := os.ReadFile(f); string(b) != file(want) {
		t.Errorf("after deleting one entry:\n%s\nwant\n%s", b, file(want))
	}

	// The same ID under two CLIs is two sessions: only Claude Code's goes.
	if _, err := ps.Update(func(l []Project) []Project { return p.ForgetSession(l, `C:\work\fav`, "claude-code", "same") }); err != nil {
		t.Fatal(err)
	}
	want[0].History = []Entry{list[0].History[0], list[0].History[2]}
	if b, _ := os.ReadFile(f); string(b) != file(want) {
		t.Errorf("after deleting one of two sessions with one ID:\n%s\nwant\n%s", b, file(want))
	}
	list[0].History = want[0].History // what the rest of the test starts from

	// Delete All of project a removes the entries its question counted, not
	// one another dashboard added meanwhile; the others stay byte for byte.
	counted := slices.Clone(want[1].History)
	late := Entry{CLI: "claude-code", ID: "late", Ended: t0}
	if _, err := ps.Update(func(l []Project) []Project { return p.RecordSession(l, `C:\work\a`, late) }); err != nil {
		t.Fatal(err)
	}
	path := `C:\work\a`
	if runtime.GOOS == "windows" {
		path = `c:/WORK/a` // the same project, as Windows compares
	}
	if _, err := ps.Update(func(l []Project) []Project { return p.ForgetSessions(l, path, counted) }); err != nil {
		t.Fatal(err)
	}
	want[1].History = []Entry{late}
	if b, _ := os.ReadFile(f); string(b) != file(want) {
		t.Errorf("after Delete All: got %s, want %s", b, file(want))
	}
	got, _ := ps.Load()
	if len(got[0].History) != 2 || !got[0].Favourite || got[0].Path != `C:\work\fav` || len(got) != 3 {
		t.Errorf("the other projects: %+v", got)
	}
	// The input list is never changed.
	if len(list[1].History) != 2 {
		t.Error("ForgetSessions changed the list it was given")
	}
}

// TestHistoryLoad is design D1's read: a version 2 file's history keeps no
// entry without an ID and at most MaxHistory entries, the newest, in the file's
// order; the file itself is not changed by reading it.
func TestHistoryLoad(t *testing.T) {
	ps := Projects{Dir: t.TempDir()}
	var es []string
	for i := range 30 {
		id := fmt.Sprintf(`"id":"s%02d"`, i)
		if i%5 == 0 {
			id = `"id":""` // a hand-edited file: nothing to resume
		}
		es = append(es, fmt.Sprintf(`{"cli":"codex",%s,"ended":"2026-10-18T12:00:00Z"}`, id))
	}
	body := `{"version":2,"projects":[{"path":"C:/work/a","used":"2026-10-18T12:00:00Z","history":[` + strings.Join(es, ",") + `]}]}`
	f := writeFile(t, ps, body)
	got, err := ps.Load()
	if err != nil || len(got) != 1 {
		t.Fatalf("%v, %v", got, err)
	}
	h := got[0].History
	if len(h) != MaxHistory || h[0].ID != "s01" || h[MaxHistory-1].ID != "s24" {
		t.Fatalf("%d entries, first %q, last %q", len(h), h[0].ID, h[len(h)-1].ID)
	}
	for _, e := range h {
		if e.ID == "" {
			t.Fatal("an entry without an ID was kept")
		}
	}
	if b, _ := os.ReadFile(f); string(b) != body {
		t.Error("reading changed the file")
	}
	// Only entries without an ID: no history at all.
	writeFile(t, ps, `{"version":2,"projects":[{"path":"C:/work/a","used":"2026-10-18T12:00:00Z","history":[{"cli":"codex","id":"","ended":"2026-10-18T12:00:00Z"}]}]}`)
	if got, err = ps.Load(); err != nil || got[0].History != nil {
		t.Errorf("only ID-less entries: %+v, %v", got, err)
	}
}
