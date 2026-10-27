package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// folderPicker is a stand-in for the system's folder dialog: it records
// where each one opened and gives its answers in turn, "" (cancelled) once
// they run out, or err; hold, when set, keeps a dialog open until closed.
type folderPicker struct {
	mu      sync.Mutex
	starts  []string
	answers []string
	err     error
	hold    chan struct{}
}

func (p *folderPicker) pick(start string) (string, error) {
	p.mu.Lock()
	p.starts = append(p.starts, start)
	hold := p.hold
	p.mu.Unlock()
	if hold != nil {
		<-hold
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.err != nil {
		return "", p.err
	}
	if len(p.answers) == 0 {
		return "", nil
	}
	a := p.answers[0]
	p.answers = p.answers[1:]
	return a, nil
}

// opened is where the dialogs opened, in turn.
func (p *folderPicker) opened() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.starts...)
}

// TestFolderStart is plan 52's design D2: a directory, else its nearest
// existing parent, else the folder the screen opened with.
func TestFolderStart(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	cases := []struct{ in, want string }{
		{dir, dir},
		{filepath.Join(dir, "missing", "deeper"), dir},
		{file, dir},
		{"", "fallback"},
	}
	for c := 'Z'; c >= 'D'; c-- {
		if _, err := os.Stat(string(c) + `:\`); err != nil {
			cases = append(cases, struct{ in, want string }{string(c) + `:\nowhere`, "fallback"}) // no such drive
			break
		}
	}
	for _, c := range cases {
		if got := folderStart(c.in, "fallback"); got != c.want {
			t.Errorf("folderStart(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestFolderDialog is plan 52's criteria 52.1 and 52.2 with a stand-in
// dialog: Ctrl+O opens it at the field's directory (design D2); a folder
// chosen is in the field, which takes the focus, its live checks as when
// typed; Cancel changes nothing; a failure is a note until the dialog opens
// again or Ctrl+N (design D3); one dialog at a time, none while a start is
// validated; with no dialog the screen says so.
func TestFolderDialog(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		work := t.TempDir()
		chosen := filepath.Join(work, "chosen")
		if err := os.Mkdir(chosen, 0o755); err != nil {
			t.Fatal(err)
		}
		h := newHarness(t, Options{Providers: bothCLIs, Cwd: work})
		h.settle()
		h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
		fp := &folderPicker{answers: []string{chosen}}
		h.m.opt.PickFolder = fp.pick

		h.send(key('o', tea.ModCtrl))
		if got := fp.opened(); len(got) != 1 || got[0] != work {
			t.Fatalf("Ctrl+O: dialogs opened at %q, want %q", got, work)
		}
		if h.m.path.Text() != chosen || h.m.ui.Field != 1 || h.m.ui.Resolved != chosen {
			t.Fatalf("chosen: field %q, focus %d, resolved %q", h.m.path.Text(), h.m.ui.Field, h.m.ui.Resolved)
		}

		h.m.ui.Field = 0
		h.send(key('o', tea.ModCtrl)) // cancelled
		if got := fp.opened(); len(got) != 2 || got[1] != chosen {
			t.Errorf("the second dialog opened at %q, want the field's %q", got, chosen)
		}
		if h.m.path.Text() != chosen || h.m.ui.Field != 0 {
			t.Errorf("cancelled: field %q, focus %d", h.m.path.Text(), h.m.ui.Field)
		}

		fp.err = errors.New("broken")
		h.send(key('o', tea.ModCtrl))
		if f := h.frame(); !strings.Contains(f, "✗ Folder dialog: broken") || h.m.path.Text() != chosen {
			t.Errorf("a failure: field %q, no note in\n%s", h.m.path.Text(), f)
		}
		fp.err = nil
		h.send(key('o', tea.ModCtrl)) // cancelled: the note gone with the new dialog
		if f := h.frame(); strings.Contains(f, "Folder dialog") {
			t.Errorf("the note outlived the next dialog:\n%s", f)
		}
		fp.err = errors.New("broken")
		h.send(key('o', tea.ModCtrl))
		h.send(key('n', tea.ModCtrl))
		if f := h.frame(); strings.Contains(f, "Folder dialog") {
			t.Errorf("the note outlived Ctrl+N:\n%s", f)
		}

		fp.mu.Lock()
		fp.err, fp.hold, fp.answers = nil, make(chan struct{}), []string{chosen}
		fp.mu.Unlock()
		h.send(key('o', tea.ModCtrl))
		h.send(key('o', tea.ModCtrl)) // the first still open
		if n := len(fp.opened()); n != 6 {
			t.Errorf("%d dialogs opened, want 6: one at a time", n)
		}
		close(fp.hold)
		h.settle()
		if h.m.path.Text() != chosen || h.m.folderOpen {
			t.Errorf("after the held dialog: field %q, still open %v", h.m.path.Text(), h.m.folderOpen)
		}

		h.m.s.Startup.Phase = state.Validating
		h.send(key('o', tea.ModCtrl))
		h.m.s.Startup.Phase = state.Asking
		if n := len(fp.opened()); n != 6 {
			t.Errorf("a dialog opened while validating: %d", n)
		}

		h.m.opt.PickFolder = nil
		h.send(key('o', tea.ModCtrl))
		if f := h.frame(); !strings.Contains(f, "✗ No folder dialog in this run") {
			t.Errorf("no dialog: no note in\n%s", f)
		}
	})
}
