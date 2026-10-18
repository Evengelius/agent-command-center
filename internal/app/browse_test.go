package app

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// fakeLister lists dirs; a directory not in it does not exist. With block
// set, every listing waits for it to close.
type fakeLister struct {
	dirs   map[string][]input.DirEntry
	errs   map[string]error
	drives []string
	block  chan struct{}
}

func (f *fakeLister) List(dir string) ([]input.DirEntry, error) {
	if f.block != nil {
		<-f.block
	}
	if err := f.errs[dir]; err != nil {
		return nil, &fs.PathError{Op: "open", Path: dir, Err: err}
	}
	es, ok := f.dirs[dir]
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: dir, Err: fs.ErrNotExist}
	}
	return es, nil
}

func (f *fakeLister) Drives() []string { return f.drives }

func ctrl(r rune) tea.KeyPressMsg { return key(r, tea.ModCtrl) }

// rowNames is the browser's rows as names, the use row as "=".
func rowNames(h *harness) []string {
	var out []string
	for _, r := range h.m.ui.Browser.Rows {
		if r.Kind == 'u' {
			out = append(out, "=")
		} else {
			out = append(out, r.Name)
		}
	}
	return out
}

func mkdirs(t *testing.T, root string, dirs ...string) {
	t.Helper()
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

// TestBrowserOpens is criterion 3.2 of plan 15: the browser opens at the
// field's directory, else its nearest existing parent, else home.
func TestBrowserOpens(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := t.TempDir()
		mkdirs(t, root, `a\b`)
		h := newHarness(t, Options{Providers: bothCLIs, Cwd: root})
		h.settle()
		for _, c := range []struct{ field, want string }{
			{filepath.Join(root, `a\b`), filepath.Join(root, `a\b`)},
			{filepath.Join(root, `a\missing\deeper`), filepath.Join(root, "a")},
			{"/usr/local", root}, // refused on Windows: the current directory
			{"", root},
		} {
			h.m.path.SetText(c.field)
			h.send(ctrl('o'))
			if b := h.m.browse; b == nil || b.dir != c.want || b.reading || b.problem != "" {
				t.Fatalf("field %q: browser %+v, want %s", c.field, b, c.want)
			}
			h.send(key(tea.KeyEscape, 0))
			if h.m.browse != nil || h.m.path.Text() != c.field {
				t.Fatalf("Esc: browser %v, field %q", h.m.browse, h.m.path.Text())
			}
		}

		// Nothing on the way up exists: home.
		fake := &fakeLister{dirs: map[string][]input.DirEntry{`C:\home\me`: {{Name: "x"}}}}
		h = newHarness(t, Options{Providers: bothCLIs, Cwd: root, Home: `C:\home\me`, Lister: fake})
		h.settle()
		h.m.path.SetText(`Q:\gone\deeper`)
		h.send(ctrl('o'))
		if b := h.m.browse; b == nil || b.dir != `C:\home\me` || len(b.entries) != 1 {
			t.Fatalf("browser %+v, want home", b)
		}
	})
}

// TestBrowserKeys is criterion 3.3 of plan 15.
func TestBrowserKeys(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := t.TempDir()
		mkdirs(t, root, "alpha", "Beta", `gamma\inner`, ".hid", `repo\.git`)
		store := storeWith(t, nil)
		h := newHarness(t, Options{Providers: bothCLIs, Cwd: root, Projects: store})
		h.settle()
		h.send(ctrl('o'))
		if got, want := strings.Join(rowNames(h), " "), "= .. alpha Beta gamma repo"; got != want {
			t.Fatalf("rows %q, want %q", got, want)
		}
		if v := h.m.ui.Browser; v.Shown != 4 || v.Hidden != 1 || !v.Rows[5].Git || v.Rows[2].Git {
			t.Fatalf("view %+v", v)
		}
		if f := h.frame(); !strings.Contains(f, "4 directories · 1 hidden, Ctrl+H shows them") || !strings.Contains(f, "Choose a project directory") {
			t.Fatalf("frame:\n%s", f)
		}

		// Ctrl+H shows the hidden directories, and hides them again.
		h.send(ctrl('h'))
		if got := strings.Join(rowNames(h), " "); got != "= .. .hid alpha Beta gamma repo" {
			t.Fatalf("Ctrl+H: rows %q", got)
		}
		h.send(ctrl('h'))

		// Typing filters and selects the first match; Enter opens it.
		for _, r := range "GAM" {
			h.send(typed(r))
		}
		if got := strings.Join(rowNames(h), " "); got != "= .. gamma" || h.m.browse.sel != 2 {
			t.Fatalf("filter: rows %q, row %d", got, h.m.browse.sel)
		}
		h.send(key(tea.KeyEnter, 0))
		if b := h.m.browse; b.dir != filepath.Join(root, "gamma") || len(b.filter) != 0 || strings.Join(rowNames(h), " ") != "= .. inner" {
			t.Fatalf("Enter: dir %s, filter %q, rows %v", b.dir, string(b.filter), rowNames(h))
		}

		// Backspace with an empty filter goes up, the child left selected;
		// so does ←.
		h.send(key(tea.KeyBackspace, 0))
		if b := h.m.browse; b.dir != root || rowNames(h)[b.sel] != "gamma" {
			t.Fatalf("Backspace: dir %s, row %d %v", b.dir, b.sel, rowNames(h))
		}
		h.send(key(tea.KeyLeft, 0))
		if b := h.m.browse; b.dir != filepath.Dir(root) || rowNames(h)[b.sel] != filepath.Base(root) {
			t.Fatalf("←: dir %s, row %d", b.dir, b.sel)
		}
		h.send(key(tea.KeyEnter, 0))

		// Esc clears the filter, then closes.
		h.send(typed('z'))
		if strings.Join(rowNames(h), " ") != "= .." || !strings.Contains(h.frame(), "no directory matches the filter") {
			t.Fatalf("filter z: rows %v", rowNames(h))
		}
		h.send(key(tea.KeyEscape, 0))
		if h.m.browse == nil || len(h.m.browse.filter) != 0 {
			t.Fatal("the first Esc did not only clear the filter")
		}

		// Ctrl+F stars the selected directory, then unstars it.
		h.send(key(tea.KeyDown, 0))
		h.send(key(tea.KeyDown, 0))
		h.send(ctrl('f'))
		alpha := filepath.Join(root, "alpha")
		got, err := store.Load()
		if err != nil || len(got) != 1 || got[0].Path != alpha || !got[0].Favourite || !got[0].Used.IsZero() || !h.m.ui.Browser.Rows[2].Favourite {
			t.Fatalf("Ctrl+F: stored %v, %v", got, err)
		}
		h.send(ctrl('f'))
		if got, _ := store.Load(); len(got) != 0 || h.m.ui.Browser.Rows[2].Favourite {
			t.Fatalf("Ctrl+F again: stored %v", got)
		}
		// On .. it does nothing; on the first row it stars the directory shown.
		h.send(key(tea.KeyUp, 0))
		h.send(ctrl('f'))
		h.send(key(tea.KeyHome, 0))
		h.send(ctrl('f'))
		if got, _ := store.Load(); len(got) != 1 || got[0].Path != root || !h.m.ui.Browser.Rows[0].Favourite {
			t.Fatalf("Ctrl+F on the first row: stored %v", got)
		}
		h.send(key(tea.KeyEscape, 0))
		if h.m.browse != nil || h.m.path.Text() != root {
			t.Fatalf("Esc: browser %v, field %q", h.m.browse, h.m.path.Text())
		}

		// Enter on the first row starts with that directory.
		h.send(ctrl('o'))
		h.send(key(tea.KeyEnd, 0)) // repo
		h.send(key(tea.KeyEnter, 0))
		h.send(key(tea.KeyHome, 0))
		h.send(key(tea.KeyEnter, 0))
		repo := filepath.Join(root, "repo")
		if s := h.m.State(); h.m.browse != nil || h.m.path.Text() != repo || s.Context.Project != repo {
			t.Fatalf("Enter on the first row: browser %v, field %q, project %q", h.m.browse, h.m.path.Text(), s.Context.Project)
		}
		if got, _ := store.Load(); len(got) != 2 || got[0].Path != root || got[1].Path != repo || got[1].Used.IsZero() {
			t.Fatalf("after the start: stored %v", got)
		}
	})
}

// TestBrowserDrives: on Windows, .. at a drive's root lists the drives
// (criterion 3.3); a directory that cannot be read shows why (3.1).
func TestBrowserDrives(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		fake := &fakeLister{
			dirs:   map[string][]input.DirEntry{`C:\`: {{Name: "Users"}, {Name: "work"}}, `D:\`: {{Name: "data"}}},
			errs:   map[string]error{`C:\locked`: fs.ErrPermission},
			drives: []string{`C:\`, `D:\`},
		}
		h := newHarness(t, Options{Providers: bothCLIs, Cwd: `C:\`, Lister: fake})
		h.settle()
		h.send(ctrl('o'))
		if got := strings.Join(rowNames(h), " "); got != `= .. Users work` {
			t.Fatalf("C:\\ rows %q", got)
		}
		h.send(key(tea.KeyDown, 0))
		h.send(key(tea.KeyEnter, 0))
		if b := h.m.browse; b.dir != "" || strings.Join(rowNames(h), " ") != `C:\ D:\` || b.sel != 0 || !strings.Contains(h.frame(), "2 drives") {
			t.Fatalf("..: dir %q, rows %v, row %d", b.dir, rowNames(h), b.sel)
		}
		h.send(key(tea.KeyBackspace, 0)) // nothing above the drives
		h.send(key(tea.KeyDown, 0))
		h.send(key(tea.KeyEnter, 0))
		if b := h.m.browse; b.dir != `D:\` || strings.Join(rowNames(h), " ") != `= .. data` {
			t.Fatalf("D:\\: dir %q, rows %v", b.dir, rowNames(h))
		}

		h.m.path.SetText(`C:\locked`)
		h.send(key(tea.KeyEscape, 0))
		h.send(ctrl('o'))
		want := `Not readable (permission denied): C:\locked`
		if b := h.m.browse; b.dir != `C:\locked` || b.problem != want || strings.Join(rowNames(h), " ") != ".." || !strings.Contains(h.frame(), "✗ "+want) {
			t.Fatalf("locked: %+v, rows %v\n%s", b, rowNames(h), h.frame())
		}
		if !errors.Is(fake.errs[`C:\locked`], fs.ErrPermission) {
			t.Fatal("fixture")
		}
	})
}

// TestBrowserSlow is criterion 3.4 of plan 15: a slow listing shows
// "reading…", and keys stay handled while it runs.
func TestBrowserSlow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		fake := &fakeLister{dirs: map[string][]input.DirEntry{`C:\slow`: {{Name: "x"}}}, block: make(chan struct{})}
		h := newHarness(t, Options{Providers: bothCLIs, Cwd: `C:\slow`, Lister: fake})
		h.settle()
		h.send(ctrl('o'))
		if b := h.m.browse; b == nil || !b.reading || !strings.Contains(h.frame(), "reading…") {
			t.Fatalf("browser %+v\n%s", b, h.frame())
		}
		h.send(typed('q'))
		if string(h.m.browse.filter) != "q" || !strings.Contains(h.frame(), "Filter  q") {
			t.Fatalf("typing while reading: filter %q", string(h.m.browse.filter))
		}
		h.send(key(tea.KeyEscape, 0))
		h.send(key(tea.KeyEscape, 0))
		if h.m.browse != nil || h.m.State().Screen != state.ScreenStartup {
			t.Fatal("Esc did not close the browser while it read")
		}
		// The late listing is dropped; a new browser lists anew.
		h.send(ctrl('o'))
		close(fake.block)
		h.settle()
		if b := h.m.browse; b == nil || b.reading || len(b.entries) != 1 {
			t.Fatalf("after the listing: %+v", b)
		}
	})
}

// demoTree is the fake listing the browser's golden frames show.
func demoTree() *fakeLister {
	return &fakeLister{dirs: map[string][]input.DirEntry{`C:\javadev\projets`: {
		{Name: ".cache", Hidden: true},
		{Name: "agent-command-center", Git: true},
		{Name: "billing-service", Git: true},
		{Name: "docs-site"},
		{Name: "infra-terraform", Git: true},
		{Name: "legacy-portal", Link: true},
		{Name: "ml-experiments"},
		{Name: "mobile-app", Git: true},
		{Name: "notes"},
		{Name: "payments-api", Git: true},
		{Name: "sandbox"},
		{Name: "tools", Git: true, Link: true},
	}}, drives: []string{`C:\`, `D:\`}}
}

// browserRig opens the browser on demoTree, keys sent after it opens; slow
// leaves the listing waiting.
func browserRig(slow bool, keys ...tea.KeyPressMsg) func(*testing.T) *harness {
	return func(t *testing.T) *harness {
		fake := demoTree()
		if slow {
			fake.block = make(chan struct{})
			t.Cleanup(func() { close(fake.block) })
		}
		store := storeWith(t, []input.Project{{Path: `C:\javadev\projets\agent-command-center`, Used: testNow.Add(-time.Hour), Favourite: true}})
		h := newHarness(t, Options{Providers: bothCLIs[:1], Cwd: `C:\javadev\projets`, Projects: store, Lister: fake})
		h.settle()
		h.send(ctrl('o'))
		for _, k := range keys {
			h.send(k)
		}
		return h
	}
}
