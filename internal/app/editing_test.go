package app

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// fakeClipboard is the system clipboard in memory.
type fakeClipboard struct {
	mu     sync.Mutex
	writes []string
	fail   bool
}

func (c *fakeClipboard) Read() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.fail {
		return "", errors.New("clipboard held")
	}
	return "", nil
}
func (c *fakeClipboard) Write(text string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.fail {
		return errors.New("clipboard held")
	}
	c.writes = append(c.writes, text)
	return nil
}
func (c *fakeClipboard) last() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.writes) == 0 {
		return ""
	}
	return c.writes[len(c.writes)-1]
}

// msgsOf runs a command and every command it batches, and returns the
// messages they produce.
func msgsOf(c tea.Cmd) []tea.Msg {
	if c == nil {
		return nil
	}
	switch msg := c().(type) {
	case tea.BatchMsg:
		var out []tea.Msg
		for _, c := range msg {
			out = append(out, msgsOf(c)...)
		}
		return out
	case nil:
		return nil
	default:
		return []tea.Msg{msg}
	}
}

// press sends one key and returns the copy it started, if any; the copy's
// message is then delivered.
func press(t *testing.T, h *harness, k tea.KeyPressMsg) (copiedMsg, bool) {
	t.Helper()
	_, cmd := h.m.Update(k)
	var got copiedMsg
	found := false
	for _, msg := range msgsOf(cmd) {
		if c, ok := msg.(copiedMsg); ok {
			got, found = c, true
		}
		h.deliver(msg)
	}
	h.settle()
	return got, found
}

func copyRig(t *testing.T, env map[string]string) (inputRig, *fakeClipboard) {
	r := scriptRig(t, "codex")
	clip := &fakeClipboard{}
	r.h.m.opt.Clipboard = clip
	r.h.m.opt.Getenv = func(k string) string { return env[k] }
	return r, clip
}

// TestCopyKeys is proposal step 3, item 6, in the editor: the selection is
// copied byte for byte by the native clipboard and OSC 52, tmux-wrapped under
// tmux, OSC 52 only over SSH; Ctrl+C copies a selection and interrupts only
// without one.
func TestCopyKeys(t *testing.T) {
	const text = "alpha\tbeta  \n\tγ 日本 👍🏽\n"
	for _, c := range []struct {
		name   string
		env    map[string]string
		seq    string
		native bool
		toast  string
	}{
		{"native", nil, input.OSC52(text), true, "copied 2 lines · native"},
		{"tmux", map[string]string{"TMUX": "/tmp/tmux-1/default,1,0"}, input.TmuxWrap(input.OSC52(text)), true, "copied 2 lines · native"},
		{"ssh", map[string]string{"SSH_CONNECTION": "10.0.0.1 22 10.0.0.2 22"}, input.OSC52(text), false, "copied · OSC 52 (terminal must allow it)"},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				r, clip := copyRig(t, c.env)
				h := r.h
				h.send(tea.PasteMsg{Content: text})
				h.send(key('a', tea.ModCtrl))
				got, ok := press(t, h, key('c', tea.ModCtrl|tea.ModShift))
				if !ok {
					t.Fatal("Ctrl+Shift+C started no copy")
				}
				if got.seq != c.seq {
					t.Errorf("sequence %q, want %q", got.seq, c.seq)
				}
				if c.native && clip.last() != text {
					t.Errorf("native clipboard %q, want %q", clip.last(), text)
				}
				if !c.native && len(clip.writes) != 0 {
					t.Errorf("native clipboard written over SSH: %q", clip.writes)
				}
				if f := plainText(h.m.View().Content); !strings.Contains(f, c.toast) {
					t.Errorf("no toast %q:\n%s", c.toast, f)
				}
				if h.m.edit.Text() != text {
					t.Errorf("copying changed the draft to %q", h.m.edit.Text())
				}
			})
		})
	}
}

func TestCopyCutAndInterrupt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, clip := copyRig(t, nil)
		h := r.h
		sess := r.script

		// Ctrl+C with a selection copies and does not interrupt.
		h.send(tea.PasteMsg{Content: "one two"})
		h.send(key(tea.KeyLeft, tea.ModShift|tea.ModCtrl))
		if _, ok := press(t, h, key('c', tea.ModCtrl)); !ok || clip.last() != "two" {
			t.Fatalf("Ctrl+C with a selection copied %q, want %q", clip.last(), "two")
		}
		if _, _, n := sess.calls(); n != 0 {
			t.Errorf("Ctrl+C with a selection interrupted %d times", n)
		}

		// Ctrl+X cuts.
		if _, ok := press(t, h, key('x', tea.ModCtrl)); !ok || clip.last() != "two" || h.m.edit.Text() != "one " {
			t.Errorf("Ctrl+X: clipboard %q, draft %q; want %q and %q", clip.last(), h.m.edit.Text(), "two", "one ")
		}

		// Esc clears a selection without moving the cursor.
		h.send(key(tea.KeyLeft, tea.ModShift))
		cur := h.m.edit.Cursor()
		h.send(key(tea.KeyEscape, 0))
		if h.m.edit.HasSelection() || h.m.edit.Cursor() != cur {
			t.Errorf("Esc: selection %v, cursor %d; want none at %d", h.m.edit.HasSelection(), h.m.edit.Cursor(), cur)
		}

		// Ctrl+C with no selection during a turn interrupts.
		h.send(key(tea.KeyEnter, 0))
		r.emit(event.TurnStarted, nil)
		h.settle()
		press(t, h, key('c', tea.ModCtrl))
		if _, _, n := sess.calls(); n != 1 {
			t.Errorf("Ctrl+C without a selection during a turn: %d interrupts, want 1", n)
		}

		// A failing native clipboard still prints OSC 52.
		clip.fail = true
		h.send(tea.PasteMsg{Content: "x"})
		h.send(key('a', tea.ModCtrl))
		if got, ok := press(t, h, key('c', tea.ModCtrl|tea.ModShift)); !ok || got.seq != input.OSC52("x") {
			t.Errorf("with the native clipboard failing: %+v, want the OSC 52 sequence", got)
		}
	})
}

// TestCopyLastAnswer is Ctrl+O: the orchestrator's last answer, never a
// thinking-only entry; an answer that had a thinking block counts (its Text
// is the answer: the state keeps no thinking text). Found in conhost, plan 4
// step 3, criterion 9.
func TestCopyLastAnswer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := &state.State{Conversation: []state.Entry{
			{Role: "assistant", Text: "first answer"},
			{Role: "user", Text: "question"},
			{Role: "assistant", Text: "an earlier answer"},
			{Role: "assistant", Text: "the answer\n\twith a tab", Thinking: true},
			{Role: "assistant", Thinking: true},
		}}
		if got := lastAnswer(s); got != "the answer\n\twith a tab" {
			t.Errorf("lastAnswer %q", got)
		}
		if got := lastAnswer(&state.State{}); got != "" {
			t.Errorf("lastAnswer of nothing %q", got)
		}

		r, clip := copyRig(t, nil)
		r.h.m.s.Conversation = append(r.h.m.s.Conversation, state.Entry{Role: "assistant", Text: "copied\n\tanswer"})
		if _, ok := press(t, r.h, key('o', tea.ModCtrl)); !ok || clip.last() != "copied\n\tanswer" {
			t.Errorf("Ctrl+O copied %q", clip.last())
		}
	})
}

// TestToast: a toast replaces the key hints until its own timer ends; an
// older timer leaves a newer toast.
func TestToast(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, _ := copyRig(t, nil)
		h := r.h
		h.m.toast("first")
		old := h.m.toastGen
		h.m.toast("second")
		h.send(toastDoneMsg{old})
		if f := plainText(h.m.View().Content); !strings.Contains(f, "second") {
			t.Errorf("an older timer cleared the toast:\n%s", f)
		}
		h.send(toastDoneMsg{h.m.toastGen})
		if f := plainText(h.m.View().Content); strings.Contains(f, "second") || !strings.Contains(f, "Ctrl+Q quit") {
			t.Errorf("toast not replaced by the key hints:\n%s", f)
		}
	})
}

// TestEditorCommand is ui-design.md, $EDITOR: VISUAL, then EDITOR, split as
// the platform's shell would; else a detected editor, named.
func TestEditorCommand(t *testing.T) {
	found := func(names ...string) func(string) (string, error) {
		return func(n string) (string, error) {
			for _, f := range names {
				if f == n {
					return "/bin/" + n, nil
				}
			}
			return "", errors.New("not found")
		}
	}
	quoted, quotedWant := `'/opt/my editor/ed' -w`, []string{"/opt/my editor/ed", "-w"}
	fallback := []string{"nano"}
	if runtime.GOOS == "windows" {
		quoted, quotedWant = `"C:\Program Files\My Editor\ed.exe" -w`, []string{`C:\Program Files\My Editor\ed.exe`, "-w"}
		fallback = []string{"notepad"}
	}
	for _, c := range []struct {
		name  string
		env   map[string]string
		look  func(string) (string, error)
		want  []string
		note  string
		error bool
	}{
		{"VISUAL first", map[string]string{"VISUAL": "code --wait", "EDITOR": "vi"}, found(), []string{"code", "--wait"}, "", false},
		{"EDITOR", map[string]string{"EDITOR": "  vim  "}, found(), []string{"vim"}, "", false},
		{"quoted path", map[string]string{"EDITOR": quoted}, found(), quotedWant, "", false},
		{"code found", nil, found("code", fallback[0]), []string{"code", "--wait"}, "no $VISUAL or $EDITOR: using code --wait", false},
		{"fallback", nil, found(fallback[0]), fallback, "no $VISUAL or $EDITOR: using " + fallback[0], false},
		{"none", nil, found(), nil, "", true},
	} {
		args, note, err := editorCommand(func(k string) string { return c.env[k] }, c.look)
		if (err != nil) != c.error || !reflect.DeepEqual(args, c.want) || note != c.note {
			t.Errorf("%s: %q, %q, %v; want %q, %q, error %v", c.name, args, note, err, c.want, c.note, c.error)
		}
	}
}

// TestEditedDraft: the edited file replaces the draft byte for byte as one
// undo step and is deleted; a failed editor keeps the draft; an editor that
// returns at once is named.
func TestEditedDraft(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, _ := copyRig(t, nil)
		h := r.h
		h.send(tea.PasteMsg{Content: "before"})
		write := func(text string) string {
			p := filepath.Join(t.TempDir(), "acc-draft-1.md")
			if err := os.WriteFile(p, []byte(text), 0o600); err != nil {
				t.Fatal(err)
			}
			return p
		}

		const after = "after\r\n\twith CRLF kept\x01 \n"
		p := write(after)
		h.send(editedMsg{path: p, took: 5 * time.Second})
		if h.m.edit.Text() != after {
			t.Errorf("draft %q, want %q", h.m.edit.Text(), after)
		}
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("the draft file was not deleted: %v", err)
		}
		h.send(key('z', tea.ModCtrl))
		if h.m.edit.Text() != "before" {
			t.Errorf("one undo gives %q, want %q", h.m.edit.Text(), "before")
		}

		p = write("lost")
		h.send(editedMsg{path: p, took: 5 * time.Second, err: errors.New("exit status 1")})
		if h.m.edit.Text() != "before" {
			t.Errorf("a failed editor changed the draft to %q", h.m.edit.Text())
		}
		if f := plainText(h.m.View().Content); !strings.Contains(f, "the draft is kept") {
			t.Errorf("no failure toast:\n%s", f)
		}

		h.send(editedMsg{path: write("quick"), took: 100 * time.Millisecond})
		if f := plainText(h.m.View().Content); !strings.Contains(f, "the editor returned immediately") {
			t.Errorf("no warning for an editor that returned at once:\n%s", f)
		}
	})
}

// TestEditorDisplay is ui-design.md, Input editor: tabs as four spaces,
// control characters visible, long lines wrapped, and "N lines".
func TestEditorDisplay(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, _ := copyRig(t, nil)
		h := r.h
		long := strings.Repeat("0123456789", 30)
		h.send(tea.PasteMsg{Content: "a\tb\x01c\x7fd\u0085e\n" + long})
		f := plainText(h.m.View().Content)
		for _, want := range []string{"a    b^Ac^?d<85>e", "2 lines · Ctrl+G editor"} {
			if !strings.Contains(f, want) {
				t.Errorf("no %q in the editor:\n%s", want, f)
			}
		}
		// The 300-character line is drawn whole, over several rows.
		var drawn strings.Builder
		for _, l := range strings.Split(f, "\n") {
			i := strings.LastIndex(l, "││")
			if i < 0 {
				continue
			}
			row := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(l[i+len("││"):]), "│"))
			if row != "" && strings.Trim(row, "0123456789") == "" {
				drawn.WriteString(row)
			}
		}
		if got := drawn.String(); got != long {
			t.Errorf("wrapped line reads %d characters, want the 300 of the draft", len(got))
		}
		if h.m.edit.Text() != "a\tb\x01c\x7fd\u0085e\n"+long {
			t.Error("drawing changed the draft")
		}
	})
}

// TestRawPasteFailure: a raw Ctrl+V whose clipboard read fails says so
// instead of doing nothing (AGENTS.md, keep failures visible).
func TestRawPasteFailure(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, clip := copyRig(t, nil)
		h := r.h
		clip.fail = true
		press(t, h, key('v', tea.ModCtrl))
		if f := plainText(h.m.View().Content); !strings.Contains(f, "paste failed: clipboard held") {
			t.Errorf("a failed clipboard read is not shown:\n%s", f)
		}
	})
}
