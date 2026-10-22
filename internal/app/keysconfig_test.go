package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/provider"
)

// Plan 21, item 5.1: the editor's send and newline keys from the
// configuration file. configII is criterion (ii)'s file.
const configII = `{"keys": {"send": ["ctrl+enter"], "newline": ["enter"]}}`

var (
	ctrlEnter  = key(tea.KeyEnter, tea.ModCtrl)
	plainEnter = key(tea.KeyEnter, 0)
	ctrlJ      = key('j', tea.ModCtrl)
	shiftEnter = key(tea.KeyEnter, tea.ModShift)
	altEnter   = key(tea.KeyEnter, tea.ModAlt)
)

// keysFrom loads a configuration file's content as main would.
func keysFrom(t *testing.T, content string) Options {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	keys, problem := input.LoadKeys(p)
	return Options{Keys: keys, KeysProblem: problem}
}

// TestKeyNames holds the names the keys of these tests have in Bubble Tea.
func TestKeyNames(t *testing.T) {
	for want, k := range map[string]tea.KeyPressMsg{
		"enter": plainEnter, "ctrl+enter": ctrlEnter, "ctrl+j": ctrlJ, "shift+enter": shiftEnter, "alt+enter": altEnter,
	} {
		if got := k.String(); got != want {
			t.Errorf("key %+v is %q, want %q", k, got, want)
		}
	}
}

// TestConfiguredKeys is criterion (ii), with the three adapters (vi): Ctrl+Enter
// sends, Enter adds a newline, Ctrl+J and Shift+Enter do neither.
func TestConfiguredKeys(t *testing.T) {
	for name, rig := range rigsWith(keysFrom(t, configII)) {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				r := rig(t)
				e := &r.h.m.edit
				r.h.send(tea.PasteMsg{Content: "hello"})
				for _, k := range []tea.KeyPressMsg{ctrlJ, shiftEnter} {
					r.h.send(k)
					if got := e.Text(); got != "hello" {
						t.Errorf("%s: draft %q, want it unchanged", k, got)
					}
					if n := len(r.sent()); n != 0 {
						t.Errorf("%s sent %d messages", k, n)
					}
				}
				r.h.send(plainEnter)
				if got := e.Text(); got != "hello\n" {
					t.Errorf("Enter: draft %q, want a newline added", got)
				}
				if n := len(r.sent()); n != 0 {
					t.Errorf("Enter sent %d messages", n)
				}
				r.h.send(tea.PasteMsg{Content: "world"})
				r.h.send(ctrlEnter)
				r.end()
				if sent := r.sent(); len(sent) != 1 || sent[0] != "hello\nworld" {
					t.Errorf("Ctrl+Enter sent %q, want the draft once", sent)
				}
				if got := e.Text(); got != "" {
					t.Errorf("draft %q left after sending", got)
				}
				// An empty draft sends nothing, and `\` then a send key is a newline.
				r.h.send(ctrlEnter)
				r.h.send(tea.PasteMsg{Content: `a\`})
				r.h.send(ctrlEnter)
				if got := e.Text(); got != "a\n" || len(r.sent()) != 1 {
					t.Errorf("`\\` then Ctrl+Enter: draft %q, %d sent; want a newline and no send", got, len(r.sent()))
				}
			})
		})
	}
}

// TestConfiguredKeysShown is criterion (iii): the footer, the editor's status
// row and the F1 page show the keys in force.
func TestConfiguredKeysShown(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := claudeRigWith(t, keysFrom(t, configII))
		f := plainText(r.h.m.View().Content)
		rows := strings.Split(strings.TrimRight(f, "\n"), "\n")
		// At 120 columns the footer leaves the newline key out for room
		// (TestDashboardF3); the editor's row carries no keys since plan 29,
		// the F1 page names both.
		if n := strings.Count(f, "Ctrl+Enter send · Enter newline"); n != 0 {
			t.Errorf("%d places show \"Ctrl+Enter send · Enter newline\" at 120 columns, want none:\n%s", n, f)
		}
		if foot := strings.TrimSpace(rows[len(rows)-1]); !strings.HasPrefix(foot, "Ctrl+Enter send · Ctrl+C interrupt") {
			t.Errorf("footer %q, want it to name Ctrl+Enter as the send key", foot)
		}
		if strings.Contains(f, "Ctrl+J") {
			t.Errorf("Ctrl+J still shown:\n%s", f)
		}
		// Wider, the footer holds both keys.
		r.h.send(tea.WindowSizeMsg{Width: 200, Height: 60})
		if n := strings.Count(plainText(r.h.m.View().Content), "Ctrl+Enter send · Enter newline"); n != 1 {
			t.Errorf("at 200 columns %d places show the keys, want the footer alone", n)
		}
		r.h.send(key(tea.KeyF1, 0))
		f = plainText(r.h.m.View().Content)
		for _, want := range []string{"Ctrl+Enter", "send the draft (in the editor)", `\ then Ctrl+Enter`} {
			if !strings.Contains(f, want) {
				t.Errorf("the F1 page lacks %q:\n%s", want, f)
			}
		}
		for _, bad := range []string{"Ctrl+J", "Shift+Enter", "Alt+Enter", `\ then Enter`} {
			if strings.Contains(f, bad) {
				t.Errorf("the F1 page still shows %q:\n%s", bad, f)
			}
		}
		var newline string
		for _, l := range strings.Split(f, "\n") {
			if strings.Contains(l, "Enter") && strings.Contains(l, "newline") && !strings.Contains(l, `\ then`) {
				newline = l
			}
		}
		if !strings.Contains(newline, "Enter") || strings.Contains(newline, "(") {
			t.Errorf("the F1 page's row for the configured Enter newline key: %q", newline)
		}
	})
}

// TestConfiguredDraftWindow: the draft window closes after any send key, and
// shows the configured keys.
func TestConfiguredDraftWindow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := claudeRigWith(t, keysFrom(t, configII))
		r.h.send(tea.PasteMsg{Content: "from the window"})
		r.h.send(key('g', tea.ModCtrl))
		if !r.h.m.ui.DraftWin.Open {
			t.Fatal("Ctrl+G did not open the draft window")
		}
		if f := plainText(r.h.m.View().Content); !strings.Contains(f, "Ctrl+Enter send · Enter newline") || strings.Contains(f, "Ctrl+J") {
			t.Errorf("the draft window does not show the configured keys:\n%s", f)
		}
		r.h.send(plainEnter)
		if !r.h.m.ui.DraftWin.Open || len(r.sent()) != 0 {
			t.Errorf("Enter (a newline key): window open %v, %d sent", r.h.m.ui.DraftWin.Open, len(r.sent()))
		}
		r.h.send(ctrlEnter)
		r.end()
		if r.h.m.ui.DraftWin.Open {
			t.Error("the draft window stays open after Ctrl+Enter sent")
		}
		if sent := r.sent(); len(sent) != 1 || sent[0] != "from the window\n" {
			t.Errorf("sent %q, want the draft once", sent)
		}
	})
}

// TestPasteNeverSubmitsConfigured is criterion (iv) with criterion (ii)'s file:
// TestPasteNeverSubmits' scenario, Ctrl+Enter for Enter.
func TestPasteNeverSubmitsConfigured(t *testing.T) {
	for name, rig := range rigsWith(keysFrom(t, configII)) {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				r := rig(t)
				r.h.send(tea.PasteMsg{Content: "first line\nsecond line\n"})
				if n := len(r.sent()); n != 0 {
					t.Fatalf("a paste ending in a newline wrote %d messages, want 0", n)
				}
				r.h.send(ctrlEnter)
				r.end()
				if sent := r.sent(); len(sent) != 1 || sent[0] != "first line\nsecond line\n" {
					t.Fatalf("after Ctrl+Enter: sent %q, want the paste once", sent)
				}

				r.h.send(tea.PasteStartMsg{})
				r.h.send(tea.PasteMsg{Content: "partial"})
				r.h.send(ctrlEnter)
				if n := len(r.sent()); n != 1 {
					t.Errorf("Ctrl+Enter during a paste: %d messages in all, want 1", n)
				}
				if f := plainText(r.h.m.View().Content); !strings.Contains(f, "pasting…") {
					t.Errorf("no \"pasting…\" while the paste is open:\n%s", f)
				}
				if got := r.h.m.edit.Text(); got != "partial\n" {
					t.Errorf("draft %q during the paste, want %q", got, "partial\n")
				}
				r.h.send(tea.PasteEndMsg{})
				r.h.send(ctrlEnter)
				r.end()
				if sent := r.sent(); len(sent) != 2 || sent[1] != "partial\n" {
					t.Errorf("after the paste ended, Ctrl+Enter sent %q, want a second message %q", sent, "partial\n")
				}
			})
		})
	}
}

// problemFiles are the files of criterion (v), each with a problem.
var problemFiles = map[string]string{
	"invalid JSON":    `{"keys": `,
	"unknown setting": `{"colours": {}}`,
	"unknown key":     `{"keys": {"send": ["ctrl+s"]}}`,
	"empty list":      `{"keys": {"send": []}}`,
	"key in both":     `{"keys": {"send": ["enter"], "newline": ["enter"]}}`,
}

// TestKeysProblemOnStartup is criterion (v): the problem, naming config.json,
// on the startup screen; the default keys apply.
func TestKeysProblemOnStartup(t *testing.T) {
	for name, content := range problemFiles {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				opt := keysFrom(t, content)
				if !strings.HasPrefix(opt.KeysProblem, "config.json: ") || !strings.HasSuffix(opt.KeysProblem, "; default keys used") {
					t.Fatalf("problem %q", opt.KeysProblem)
				}
				cli := &scriptCLI{id: "claude-code", sess: newScriptSession()}
				opt.Providers, opt.Cwd = []provider.Provider{cli}, t.TempDir()
				h := newHarness(t, opt)
				h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
				f := plainText(h.m.View().Content)
				// The message wraps: its words, in order, whatever the line breaks.
				flat := strings.Join(strings.Fields(strings.NewReplacer("✗", " ", "│", " ").Replace(f)), " ")
				if !strings.Contains(flat, opt.KeysProblem) {
					t.Errorf("the startup screen lacks %q:\n%s", opt.KeysProblem, f)
				}
				if got := h.m.opt.Keys; !got.IsSend("enter") || !got.IsNewline("ctrl+j") || len(got.Send) != 1 || len(got.Newline) != 3 {
					t.Errorf("keys in force %v, want the defaults", got)
				}
				// A narrow screen wraps it; no row is wider than the screen.
				h.send(tea.WindowSizeMsg{Width: 80, Height: 24})
				for _, l := range strings.Split(plainText(h.m.View().Content), "\n") {
					if w := len([]rune(l)); w > 80 {
						t.Errorf("row of %d runes at 80 columns: %q", w, l)
					}
				}
			})
		})
	}
}

// TestKeysProblemInMockRun: a mock run has no startup screen, so the editor's
// status row shows the problem in place of the idle hint until the first
// message is sent; Enter sends (the default keys).
func TestKeysProblemInMockRun(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		opt := keysFrom(t, problemFiles["unknown key"])
		r := mockRigWith(t, opt)
		// B's status row is one row of B's width: at 120 columns the message is
		// cut at its end; wide enough, it shows whole.
		if f := plainText(r.h.m.View().Content); !strings.Contains(f, `config.json: "ctrl+s" is not a send or newline key (use Enter with Ctrl, `) {
			t.Errorf("the status row lacks the problem's start:\n%s", f)
		}
		r.h.send(tea.WindowSizeMsg{Width: 200, Height: 60})
		f := plainText(r.h.m.View().Content)
		if !strings.Contains(f, opt.KeysProblem) {
			t.Errorf("the status row lacks %q:\n%s", opt.KeysProblem, f)
		}
		r.h.send(tea.PasteMsg{Content: "hello"})
		r.h.send(plainEnter)
		if sent := r.sent(); len(sent) != 1 || sent[0] != "hello" {
			t.Errorf("Enter sent %q, want the default keys to apply", sent)
		}
		if f := plainText(r.h.m.View().Content); strings.Contains(f, "config.json") {
			t.Errorf("the problem is still shown after the first message:\n%s", f)
		}
	})
}
