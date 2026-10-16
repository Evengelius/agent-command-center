package app

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// draftText has what a draft must keep byte for byte: tabs, trailing
// spaces, blank lines, wide characters, emoji, controls and a NUL.
const draftText = "\tfirst line   \n\n日本語 👍🏽 e\u0301\n\x1b[31mred\x00end \t"

// draftFile is the stored draft's bytes, or nil without one.
func draftFile(t *testing.T, d input.Drafts, project string) []byte {
	t.Helper()
	p, err := d.Path(project)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestCrashAndResume is step 4, check 3.
func TestCrashAndResume(t *testing.T) {
	t.Run("after Ready", func(t *testing.T) {
		recs := probe(t, "p2-claude-conversation.jsonl")
		d := input.Drafts{Dir: t.TempDir()}
		w := &wireClaude{}
		h := wireRig(t, w, Options{Drafts: &d})
		text, at := firstUser(t, recs)
		h.send(tea.PasteMsg{Content: text})
		h.send(key(tea.KeyEnter, 0))
		replay(t, h, w.last(), recs, at+1, 48) // the first turn, its approval allowed
		s := h.m.State()
		id := s.Session.ID
		if s.Session.Phase != state.Ready || s.Turn.Phase != state.Idle || id == "" {
			t.Fatalf("session %q (%q), turn %q after the first turn", s.Session.Phase, id, s.Turn.Phase)
		}
		h.send(tea.PasteMsg{Content: draftText})
		project := s.Context.Project
		conv := append([]state.Entry(nil), s.Conversation...)
		file := draftFile(t, d, project)
		if !bytes.Equal(file, []byte(draftText)) {
			t.Fatalf("stored draft %q before the crash", file)
		}

		w.last().exit(3, "Error: something broke\n    at main (cli.js:1:1)\nfatal: giving up\n")
		h.settle()
		s = h.m.State()
		if s.Session.Phase != state.Crashed || !s.Session.HasExitCode || s.Session.ExitCode != 3 {
			t.Fatalf("session %q, exit code %d (%v); want Crashed, 3", s.Session.Phase, s.Session.ExitCode, s.Session.HasExitCode)
		}
		f := frameText(h)
		for _, want := range []string{"The session ended unexpectedly", "exit code 3", "stderr:", "fatal: giving up", "Ctrl+R resume"} {
			if !strings.Contains(f, want) {
				t.Errorf("crash card lacks %q:\n%s", want, f)
			}
		}
		if len(s.Conversation) < len(conv) || !reflect.DeepEqual(s.Conversation[:len(conv)], conv) {
			t.Errorf("the record changed in the crash")
		}
		if h.m.edit.Text() != draftText || !bytes.Equal(draftFile(t, d, project), []byte(draftText)) {
			t.Errorf("draft %q, stored %q after the crash; want both unchanged", h.m.edit.Text(), draftFile(t, d, project))
		}

		h.send(key('r', tea.ModCtrl))
		argv := w.spawns()
		if len(argv) != 2 {
			t.Fatalf("%d spawns after Ctrl+R", len(argv))
		}
		want := append(append([]string(nil), argv[0]...), "--resume", id)
		if !reflect.DeepEqual(argv[1], want) {
			t.Errorf("resume argv\n got %q\nwant %q", argv[1], want)
		}
		s = h.m.State()
		if s.Session.Phase != state.Ready || s.Session.ID != id {
			t.Errorf("after Ctrl+R: session %q (%q); want Ready, %q", s.Session.Phase, s.Session.ID, id)
		}
		if h.m.edit.Text() != draftText {
			t.Errorf("draft %q after the resume", h.m.edit.Text())
		}
	})

	t.Run("before Ready", func(t *testing.T) {
		w := &wireClaude{noInit: true}
		h := wireRig(t, w, Options{})
		if s := h.m.State(); s.Session.Phase != state.Handshaking {
			t.Fatalf("session %q, want Handshaking", s.Session.Phase)
		}
		w.last().exit(3, "Error: cannot start\n")
		h.settle()
		s := h.m.State()
		if s.Startup.Phase != state.StartFailed || s.Session.Phase != state.Crashed {
			t.Fatalf("startup %q, session %q; want StartFailed, Crashed", s.Startup.Phase, s.Session.Phase)
		}
		if f := frameText(h); !strings.Contains(f, "The session could not start") || !strings.Contains(f, "exit code 3") || strings.Contains(f, "Ctrl+R") {
			t.Errorf("start failure:\n%s", f)
		}
		h.send(key('r', tea.ModCtrl))
		if n := len(w.spawns()); n != 1 {
			t.Errorf("%d spawns after Ctrl+R; want none more", n)
		}
	})
}

// TestDraftRestored is step 4, check 4: a draft written, the model dropped
// without shutdown, a new model in the same project opens with it.
func TestDraftRestored(t *testing.T) {
	d := input.Drafts{Dir: t.TempDir()}
	cwd := t.TempDir()
	w := &wireClaude{}
	h := wireRig(t, w, Options{Drafts: &d, Cwd: cwd})
	h.send(tea.PasteMsg{Content: draftText})
	project := h.m.State().Context.Project
	if got := draftFile(t, d, project); !bytes.Equal(got, []byte(draftText)) {
		t.Fatalf("stored draft %q", got)
	}
	// The first model is dropped as a crash would: no shutdown, no flush.

	w2 := &wireClaude{}
	h2 := wireRig(t, w2, Options{Drafts: &d, Cwd: cwd})
	if p := h2.m.State().Context.Project; p != project {
		t.Fatalf("second project %q, want %q", p, project)
	}
	if got := h2.m.edit.Text(); got != draftText {
		t.Errorf("restored draft\n got %q\nwant %q", got, draftText)
	}
	if h2.m.ui.Toast != "draft restored" {
		t.Errorf("toast %q", h2.m.ui.Toast)
	}
	// Restoring writes nothing back.
	if got := draftFile(t, d, project); !bytes.Equal(got, []byte(draftText)) {
		t.Errorf("stored draft %q after the restore", got)
	}
}

// TestDraftAcceptedDeletes is step 4, check 4: the draft is deleted only once
// Claude Code accepted the message; until then the file holds it.
func TestDraftAcceptedDeletes(t *testing.T) {
	d := input.Drafts{Dir: t.TempDir()}
	w := &wireClaude{}
	h := wireRig(t, w, Options{Drafts: &d})
	project := h.m.State().Context.Project
	h.send(tea.PasteMsg{Content: draftText})
	h.send(key(tea.KeyEnter, 0))
	if s := h.m.State(); s.Turn.Phase != state.Submitted {
		t.Fatalf("turn %q after Enter", s.Turn.Phase)
	}
	if got := draftFile(t, d, project); !bytes.Equal(got, []byte(draftText)) {
		t.Fatalf("stored draft %q while the message waits; want the message", got)
	}
	h.send(typed('x')) // typed while it waits: not saved over the message
	if got := draftFile(t, d, project); !bytes.Equal(got, []byte(draftText)) {
		t.Fatalf("stored draft %q after typing while the message waits", got)
	}
	h.send(key(tea.KeyBackspace, 0))
	// P2 lines 7 to 9: init, status, then the answer starts: accepted.
	replay(t, h, w.last(), probe(t, "p2-claude-conversation.jsonl"), 7, 9)
	if s := h.m.State(); s.Turn.Phase != state.Running {
		t.Fatalf("turn %q after message_start", s.Turn.Phase)
	}
	if got := draftFile(t, d, project); got != nil {
		t.Errorf("stored draft %q after the message was accepted; want none", got)
	}
}

// TestDraftFailedSendKeeps is step 4, check 4: a send that fails keeps the
// stored draft and puts the message back in the editor.
func TestDraftFailedSendKeeps(t *testing.T) {
	d := input.Drafts{Dir: t.TempDir()}
	w := &wireClaude{}
	h := wireRig(t, w, Options{Drafts: &d})
	project := h.m.State().Context.Project
	h.send(tea.PasteMsg{Content: draftText})
	w.last().refuse.Store(true)
	h.send(key(tea.KeyEnter, 0))
	s := h.m.State()
	if s.Turn.Phase != state.Idle || !strings.Contains(s.Turn.Notice, "could not be sent") {
		t.Fatalf("turn %q, notice %q; want a failed send", s.Turn.Phase, s.Turn.Notice)
	}
	if h.m.edit.Text() != draftText {
		t.Errorf("editor %q after the failed send", h.m.edit.Text())
	}
	if got := draftFile(t, d, project); !bytes.Equal(got, []byte(draftText)) {
		t.Errorf("stored draft %q after the failed send", got)
	}
}

// TestDraftPath is step 4, check 4: drafts live in the user's local
// application data, one file per project named by the SHA-256 of its path.
func TestDraftPath(t *testing.T) {
	local := os.Getenv("LOCALAPPDATA")
	if local == "" {
		t.Skip("LOCALAPPDATA is not set: not Windows")
	}
	project := `C:\Work\Some Project\`
	p, err := input.Drafts{}.Path(project)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(strings.ToLower(filepath.Clean(project))))
	want := filepath.Join(local, "agent-command-center", "drafts", hex.EncodeToString(sum[:])+".txt")
	if p != want {
		t.Errorf("draft path\n got %s\nwant %s", p, want)
	}
	if q, _ := (input.Drafts{}).Path(`c:\work\some project`); q != p {
		t.Errorf("the same project in another case: %s", q)
	}
}

// TestDraftKeystrokes is step 4, check 4: typing in a 50,000-line draft
// while it is saved stays under test 6's keystroke target.
func TestDraftKeystrokes(t *testing.T) {
	d := input.Drafts{Dir: t.TempDir()}
	w := &wireClaude{}
	h := wireRig(t, w, Options{Drafts: &d})
	project := h.m.State().Context.Project
	// Saves at their real pace; the other timers stay off.
	h.m.opt.After = func(dur time.Duration, msg tea.Msg) tea.Cmd {
		if _, ok := msg.(draftSaveMsg); !ok {
			return nil
		}
		return func() tea.Msg { time.Sleep(dur); return msg }
	}
	h.send(tea.PasteMsg{Content: draftLines(50_000)})
	h.send(key(tea.KeyHome, tea.ModCtrl))
	keys := []tea.KeyPressMsg{
		typed('x'), key(tea.KeyRight, 0), key(tea.KeyDown, 0), key(tea.KeyBackspace, 0),
		key('j', tea.ModCtrl), key(tea.KeyUp, 0), key(tea.KeyRight, tea.ModCtrl), key(tea.KeyDown, tea.ModShift),
	}
	gen := h.m.draftGen
	var ds []time.Duration
	for i := 0; i < 800; i++ {
		start := time.Now()
		_, cmd := h.m.Update(keys[i%len(keys)])
		_ = h.m.View()
		ds = append(ds, time.Since(start))
		h.exec(cmd)
		for drained := false; !drained; {
			select {
			case msg := <-h.msgs:
				h.deliver(msg)
			default:
				drained = true
			}
		}
		time.Sleep(3 * time.Millisecond) // a fast typist's pace, so saves run between keys
	}
	for deadline := time.Now().Add(5 * time.Second); h.m.draftTimer && time.Now().Before(deadline); {
		h.settle() // the last save is due
	}
	h.settle()
	saves := h.m.draftGen - gen
	p50, p95, worst := percentile(ds, 0.50), percentile(ds, 0.95), percentile(ds, 1)
	t.Logf("keystroke-to-frame, 50,000-line draft, %d keys, %d saves: p50 %v, p95 %v, max %v (target p95 ≤ %v)", len(ds), saves, p50, p95, worst, keystrokeTarget)
	if p95 > keystrokeTarget {
		t.Errorf("p95 %v over the %v target", p95, keystrokeTarget)
	}
	if saves < 2 {
		t.Errorf("%d saves while typing; want the saves running", saves)
	}
	if got := draftFile(t, d, project); string(got) != h.m.edit.Text() {
		t.Errorf("stored draft (%d bytes) differs from the editor (%d bytes)", len(got), len(h.m.edit.Text()))
	}
}
