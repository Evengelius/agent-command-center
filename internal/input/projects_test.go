package input

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

var t0 = time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)

// TestProjectsStore is criterion 1.1 of plan 15: Save then Load gives the
// same entries; no file is an empty list without an error; a damaged file is
// an error naming the file; a failed write leaves the previous file whole.
func TestProjectsStore(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "acc") // created by the first save
	p := Projects{Dir: dir}
	if got, err := p.Load(); got != nil || err != nil {
		t.Fatalf("no file: got %v, %v", got, err)
	}
	list := []Project{
		{Path: `C:\work\b`, Used: t0.Add(-time.Hour), Favourite: true},
		{Path: `C:\work\a`, Used: t0},
	}
	if err := p.Save(list); err != nil {
		t.Fatal(err)
	}
	got, err := p.Load()
	if err != nil || !reflect.DeepEqual(got, list) {
		t.Fatalf("Save then Load: got %v, %v; want %v", got, err, list)
	}
	f, _ := p.File()
	before, _ := os.ReadFile(f)

	renameFile = func(string, string) error { return errors.New("disk full") }
	err = p.Save([]Project{{Path: `C:\other`, Used: t0}})
	renameFile = os.Rename
	if err == nil {
		t.Fatal("a failed write reported no error")
	}
	if after, _ := os.ReadFile(f); string(after) != string(before) {
		t.Errorf("a failed write changed the file:\n%s", after)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 1 {
		t.Errorf("files %v: want projects.json only, no temporary file left", entries)
	}

	for name, body := range map[string]string{
		"not JSON":     "{",
		"version":      `{"version":2,"projects":[]}`,
		"empty path":   `{"version":1,"projects":[{"path":""}]}`,
		"wrong shape":  `{"version":1,"projects":{}}`,
		"empty object": `{}`,
	} {
		if err := os.WriteFile(f, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		got, err := p.Load()
		if err == nil || got != nil || !strings.Contains(err.Error(), f) {
			t.Errorf("%s: got %v, %v; want an error naming %s", name, got, err, f)
		}
		if _, err := p.Update(func(l []Project) []Project { return nil }); err == nil {
			t.Errorf("%s: Update wrote over a list it could not read", name)
		}
		if b, _ := os.ReadFile(f); string(b) != body {
			t.Errorf("%s: the damaged file was changed", name)
		}
	}
}

// TestRecordProject is criterion 1.2 of plan 15: the project moves to the
// front with its start time, once (a path differing only in case is the same
// project on Windows), and at most RecentKept recent projects are kept; a
// favourite keeps its star and is never dropped (criterion 2.2).
func TestRecordProject(t *testing.T) {
	var p Projects
	var list []Project
	for i := range 12 {
		list = p.Record(list, fmt.Sprintf(`C:\work\p%02d`, i), t0.Add(time.Duration(i)*time.Minute))
	}
	if len(list) != RecentKept || list[0].Path != `C:\work\p11` || list[RecentKept-1].Path != `C:\work\p02` {
		t.Fatalf("cap: got %d entries, first %q, last %q", len(list), list[0].Path, list[len(list)-1].Path)
	}
	list = p.Record(list, `C:\work\p05\`, t0.Add(time.Hour))
	if list[0].Path != `C:\work\p05` || !list[0].Used.Equal(t0.Add(time.Hour)) || len(list) != RecentKept {
		t.Fatalf("again: got first %v, %d entries", list[0], len(list))
	}
	if runtime.GOOS == "windows" {
		list = p.Record(list, `c:\WORK\P07`, t0.Add(2*time.Hour))
		if list[0].Path != `c:\WORK\P07` || len(list) != RecentKept || count(p, list, `C:\work\p07`) != 1 {
			t.Fatalf("case: got first %q, %d entries, %d for p07", list[0].Path, len(list), count(p, list, `C:\work\p07`))
		}
	}

	// A favourite keeps its star when recorded again, comes first, and the
	// cap counts only the others.
	list[RecentKept-1].Favourite = true
	fav := list[RecentKept-1].Path
	SortProjects(list)
	for i := range RecentKept + 2 {
		list = p.Record(list, fmt.Sprintf(`D:\new%02d`, i), t0.Add(3*time.Hour+time.Duration(i)*time.Minute))
	}
	if list[0].Path != fav || !list[0].Favourite || len(list) != RecentKept+1 {
		t.Fatalf("favourite: got first %v, %d entries", list[0], len(list))
	}
	list = p.Record(list, fav, t0.Add(5*time.Hour))
	if !list[0].Favourite {
		t.Error("recording a favourite again lost its star")
	}
	if list = p.Forget(list, fav); count(p, list, fav) != 0 || len(list) != RecentKept {
		t.Errorf("Forget: %d entries, %d for %s", len(list), count(p, list, fav), fav)
	}
}

func count(p Projects, list []Project, path string) int {
	n := 0
	for _, e := range list {
		if p.Same(e.Path, path) {
			n++
		}
	}
	return n
}

// TestStarProject is criteria 2.1 and 2.2 of plan 15: a star moves the
// project first and the cap never drops it; unstarred, it is a recent
// project again, and one no session started in goes.
func TestStarProject(t *testing.T) {
	var p Projects
	var list []Project
	for i := range RecentKept {
		list = p.Record(list, fmt.Sprintf(`C:\work\p%02d`, i), t0.Add(time.Duration(i)*time.Minute))
	}
	list = p.Star(list, `C:\work\p03`, true)
	if list[0].Path != `C:\work\p03` || !list[0].Favourite || len(list) != RecentKept {
		t.Fatalf("star: first %v, %d entries", list[0], len(list))
	}
	list = p.Star(list, `D:\never-started\`, true)
	if list[1].Path != `D:\never-started` || !list[1].Favourite || !list[1].Used.IsZero() || len(list) != RecentKept+1 {
		t.Fatalf("star a new directory: second %v, %d entries", list[1], len(list))
	}
	for i := range 3 {
		list = p.Record(list, fmt.Sprintf(`E:\new%d`, i), t0.Add(time.Hour+time.Duration(i)*time.Minute))
	}
	if count(p, list, `C:\work\p03`) != 1 || count(p, list, `D:\never-started`) != 1 || len(list) != RecentKept+2 {
		t.Fatalf("the cap dropped a favourite: %d entries", len(list))
	}
	list = p.Star(list, `D:\never-started`, false)
	if count(p, list, `D:\never-started`) != 0 || len(list) != RecentKept+1 {
		t.Errorf("unstar a never-started directory: %d entries, %d for it", len(list), count(p, list, `D:\never-started`))
	}
	list = p.Star(list, `C:\work\p03`, false)
	if len(list) != RecentKept || list[0].Favourite {
		t.Errorf("unstar: %d entries, first %v", len(list), list[0])
	}
	if count(p, list, `C:\work\p03`) != 1 || count(p, list, `C:\work\p02`) != 0 {
		t.Error("unstarred, p03 is not a recent project again, or the oldest, p02, is kept past the cap")
	}
}
