package app

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 67, the characters typed with AltGr (brief amendment 69). Each key
// as ACC's decoder makes it from the records Windows Terminal 1.23 sent on
// a Belgian AZERTY keyboard (plan 67, What was found): AltGr's character
// with Ctrl, Alt and its text; Ctrl with the left Alt with Ctrl, Alt and no
// text, the decoder giving a text to AltGr's state alone. Windows Terminal
// sends the modifiers' own presses before the character and the releases
// after it, as altGrKeys and ctrlLeftAltKeys do (plan 67, step 3).

// altGr is AltGr and the key of r, as ACC receives it.
func altGr(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r, Text: string(r), Mod: tea.ModCtrl | tea.ModAlt}
}

// ctrlLeftAlt is Ctrl with the left Alt and the key that gives r with AltGr,
// as ACC receives it: the character its code, no text.
func ctrlLeftAlt(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r, Mod: tea.ModCtrl | tea.ModAlt}
}

// altGrKeys is AltGr and the key of r as Windows Terminal sends them: the
// left Ctrl's own press, the right Alt's, the character, then the three
// releases, each as the decoder makes it from run C's records.
func altGrKeys(r rune) []tea.Msg {
	return []tea.Msg{
		tea.KeyPressMsg{Code: tea.KeyLeftCtrl, Mod: tea.ModCtrl},
		tea.KeyPressMsg{Code: tea.KeyRightAlt, Mod: tea.ModCtrl | tea.ModAlt},
		altGr(r),
		tea.KeyReleaseMsg(altGr(r)),
		tea.KeyReleaseMsg{Code: tea.KeyLeftCtrl, Mod: tea.ModAlt},
		tea.KeyReleaseMsg{Code: tea.KeyRightAlt},
	}
}

// ctrlLeftAltKeys is Ctrl with the left Alt and the key of r, the same way.
func ctrlLeftAltKeys(r rune) []tea.Msg {
	return []tea.Msg{
		tea.KeyPressMsg{Code: tea.KeyLeftCtrl, Mod: tea.ModCtrl},
		tea.KeyPressMsg{Code: tea.KeyLeftAlt, Mod: tea.ModCtrl | tea.ModAlt},
		ctrlLeftAlt(r),
		tea.KeyReleaseMsg(ctrlLeftAlt(r)),
		tea.KeyReleaseMsg{Code: tea.KeyLeftAlt, Mod: tea.ModCtrl},
		tea.KeyReleaseMsg{Code: tea.KeyLeftCtrl},
	}
}

// sendAll sends msgs in turn.
func sendAll(h *harness, msgs []tea.Msg) {
	for _, msg := range msgs {
		h.send(msg)
	}
}

// altGrChars are the characters AltGr types at once on the owner's
// keyboard (plan 67, What was found).
const altGrChars = `@\#|{}[]`

// TestKeyText is criterion 67.1: the rule, on the events as the decoder
// makes them.
func TestKeyText(t *testing.T) {
	for _, tc := range []struct {
		name string
		k    tea.KeyPressMsg
		want string
	}{
		{"AltGr+2", altGr('@'), "@"},
		{"AltGr+<", altGr('\\'), `\`},
		{"AltGr+2 with Caps Lock", tea.KeyPressMsg{Code: '@', Text: "@", Mod: tea.ModCtrl | tea.ModAlt | tea.ModCapsLock}, "@"},
		{"AltGr+2 with Num Lock", tea.KeyPressMsg{Code: '@', Text: "@", Mod: tea.ModCtrl | tea.ModAlt | tea.ModNumLock}, "@"},
		{"Ctrl with the left Alt and <", ctrlLeftAlt('\\'), ""},
		{"a", typed('a'), "a"},
		{"Shift+a", tea.KeyPressMsg{Code: 'a', Text: "A", Mod: tea.ModShift}, "A"},
		{"a with Caps Lock", tea.KeyPressMsg{Code: 'a', Text: "A", Mod: tea.ModCapsLock}, "A"},
		{"a dead key's ê", tea.KeyPressMsg{Code: 'ê', Text: "ê"}, "ê"},
		{"Ctrl+k carrying a text", tea.KeyPressMsg{Code: 'k', Text: "k", Mod: tea.ModCtrl}, ""},
		{"Alt+k carrying a text", tea.KeyPressMsg{Code: 'k', Text: "k", Mod: tea.ModAlt}, ""},
		{"Enter, no text", key(tea.KeyEnter, 0), ""},
	} {
		if got := keyText(tc.k); got != tc.want {
			t.Errorf("%s: keyText %q, want %q", tc.name, got, tc.want)
		}
	}
}

// altGrField is a field of the dashboard: a rig with it open, and its
// text.
type altGrField struct {
	name  string
	open  func(t *testing.T) *harness
	text  func(h *harness) string
	types bool // AltGr's "@" typed; false for the go-to field's digits alone
}

// altGrFields are every field of criterion 67.2.
func altGrFields() []altGrField {
	dash := func(t *testing.T) *harness {
		h, _ := readyScript(t, claude.ProviderID, false, event.Started{})
		return h
	}
	viewer := func(field rune) func(t *testing.T) *harness {
		return func(t *testing.T) *harness {
			r, _, _ := fileRig(t)
			openPath(r.h, "main.go")
			r.h.send(typed(field))
			if r.h.m.ui.File.Field != byte(field) {
				t.Fatalf("%c opened no field: %q", field, r.h.m.ui.File.Field)
			}
			return r.h
		}
	}
	draft := func(h *harness) string { return h.m.edit.Text() }
	return []altGrField{
		{"the draft", dash, draft, true},
		{"the draft window", func(t *testing.T) *harness {
			h := dash(t)
			h.send(key('g', tea.ModCtrl))
			if !h.m.ui.DraftWin.Open {
				t.Fatal("Ctrl+G opened no draft window")
			}
			return h
		}, draft, true},
		{"the startup screen's project field", func(t *testing.T) *harness {
			h := newHarness(t, Options{Providers: []provider.Provider{
				fakeCLI{provider.Detection{ID: "claude-code", Name: "Claude Code", Installed: true, Version: "2.1.300", Path: `C:\Tools\claude.exe`}},
			}, Cwd: `C:\work\demo-project`})
			h.settle()
			if s := h.m.State(); s.Screen != state.ScreenStartup || s.Startup.Phase != state.Asking {
				t.Fatalf("startup: screen %q, phase %q", s.Screen, s.Startup.Phase)
			}
			h.m.ui.Field = 1
			return h
		}, func(h *harness) string { return h.m.path.Text() }, true},
		{"Source Control's message box", func(t *testing.T) *harness {
			h, _ := commitRig(t)
			typeMessage(t, h, "")
			return h
		}, func(h *harness) string { return h.m.msg.Text() }, true},
		{"the Worktrees view's new branch", func(t *testing.T) *harness {
			h := worktreesRig(true, typed('n'))(t) // a new worktree is the startup screen's
			if h.m.wt.mode != 'n' {
				t.Fatalf("n: mode %q", h.m.wt.mode)
			}
			return h
		}, func(h *harness) string { return h.m.wt.branch.Text() }, true},
		{"the branches popup's search", func(t *testing.T) *harness {
			r, _, _ := branchesRig(t)
			openBranches(t, r.h)
			return r.h
		}, func(h *harness) string { return h.m.ui.Branches.Query }, true},
		{"the conversation's search bar", func(t *testing.T) *harness {
			h := dash(t)
			openBar(h)
			return h
		}, func(h *harness) string { return h.m.ui.Search.Query }, true},
		{"the file viewer's search field", viewer('/'), func(h *harness) string { return h.m.ui.File.Typed }, true},
		{"the file viewer's go-to field", viewer('g'), func(h *harness) string { return h.m.ui.File.Typed }, false},
		{"the quick open", func(t *testing.T) *harness {
			h, _ := explorerRig(t, &fakeGit{})
			h.send(key(tea.KeyF3, 0))
			h.send(typed('/'))
			if !h.m.ui.Finder.Open {
				t.Fatal("/ opened no quick open")
			}
			return h
		}, func(h *harness) string { return h.m.ui.Finder.Query }, true},
		{"the history scope popup's search", func(t *testing.T) *harness {
			h, _ := graphRig(t, 10)
			h.send(altG())
			h.send(typed('s'))
			if !h.m.ui.Scope.Open {
				t.Fatal("s opened no scope popup")
			}
			return h
		}, func(h *harness) string { return h.m.ui.Scope.Query }, true},
	}
}

// TestAltGrFields is criterion 67.2's fields: AltGr's "@" typed in each, as
// Windows Terminal sends it, the draft untouched by a field's typing; Ctrl
// with the left Alt, Ctrl+K and Alt+K carrying a text typing nothing in any.
func TestAltGrFields(t *testing.T) {
	for _, f := range altGrFields() {
		t.Run(f.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				h := f.open(t)
				before, draft := f.text(h), h.m.edit.Text()
				sendAll(h, altGrKeys('@'))
				want := before
				if f.types {
					want += "@"
				}
				if got := f.text(h); got != want {
					t.Errorf("AltGr+2: %q, want %q", got, want)
				}
				if f.name != "the draft" && f.name != "the draft window" && h.m.edit.Text() != draft {
					t.Errorf("AltGr+2 reached the draft: %q", h.m.edit.Text())
				}
			})
			for _, keys := range [][]tea.Msg{
				ctrlLeftAltKeys('\\'),
				{tea.KeyPressMsg{Code: 'k', Text: "k", Mod: tea.ModCtrl}},
				{tea.KeyPressMsg{Code: 'k', Text: "k", Mod: tea.ModAlt}},
			} {
				synctest.Test(t, func(t *testing.T) {
					h := f.open(t)
					before, draft := f.text(h), h.m.edit.Text()
					sendAll(h, keys)
					if got := f.text(h); strings.ContainsAny(strings.TrimPrefix(got, before), `\k`) || h.m.edit.Text() != draft {
						t.Errorf("%v: the field %q (was %q), the draft %q (was %q)", keys, got, before, h.m.edit.Text(), draft)
					}
				})
			}
		})
	}
}

// TestBranchesModifierPress is the branches popup's part of criterion
// 67.4 (plan 67, step 3): a modifier's own press keeps the popup open, its
// search as it was, so AltGr's character and a capital after Shift's press
// reach the search, and its own chords, Ctrl+C and Alt+B, act as Windows
// Terminal sends them, the modifier's press first; a key it does not take
// still closes it.
func TestBranchesModifierPress(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, _, clip := branchesRig(t)
		h := r.h
		openBranches(t, h)
		for _, c := range []rune{tea.KeyLeftShift, tea.KeyRightShift, tea.KeyLeftCtrl, tea.KeyRightCtrl,
			tea.KeyLeftAlt, tea.KeyRightAlt, tea.KeyLeftSuper, tea.KeyRightSuper, tea.KeyLeftHyper, tea.KeyRightHyper,
			tea.KeyLeftMeta, tea.KeyRightMeta, tea.KeyIsoLevel3Shift, tea.KeyIsoLevel5Shift} {
			k := tea.KeyPressMsg{Code: c}
			h.send(k)
			if !h.m.ui.Branches.Open || h.m.ui.Branches.Query != "" || h.m.edit.Text() != "" {
				t.Fatalf("%s's own press: open %v, search %q, draft %q", k, h.m.ui.Branches.Open, h.m.ui.Branches.Query, h.m.edit.Text())
			}
		}
		sendAll(h, altGrKeys('@'))
		sendAll(h, []tea.Msg{
			tea.KeyPressMsg{Code: tea.KeyLeftShift, Mod: tea.ModShift},
			tea.KeyPressMsg{Code: 'F', Text: "F", Mod: tea.ModShift},
			tea.KeyReleaseMsg{Code: 'F', Text: "F", Mod: tea.ModShift},
			tea.KeyReleaseMsg{Code: tea.KeyLeftShift},
		})
		if !h.m.ui.Branches.Open || h.m.ui.Branches.Query != "@F" || h.m.edit.Text() != "" {
			t.Fatalf("AltGr+2, Shift+F: open %v, search %q, draft %q", h.m.ui.Branches.Open, h.m.ui.Branches.Query, h.m.edit.Text())
		}
		h.send(key(tea.KeyEscape, 0)) // the search cleared
		branchTo(t, h, "b:1:main")
		sendAll(h, []tea.Msg{tea.KeyPressMsg{Code: tea.KeyLeftCtrl, Mod: tea.ModCtrl}, ctrl('c'),
			tea.KeyReleaseMsg(ctrl('c')), tea.KeyReleaseMsg{Code: tea.KeyLeftCtrl}})
		if got := strings.Join(clip.writes, "|"); got != "main" || !h.m.ui.Branches.Open {
			t.Fatalf("Ctrl+C: copied %q, open %v", got, h.m.ui.Branches.Open)
		}
		sendAll(h, []tea.Msg{tea.KeyPressMsg{Code: tea.KeyLeftAlt, Mod: tea.ModAlt}, altB})
		if h.m.ui.Branches.Open {
			t.Fatal("Alt+B left the popup open")
		}
		openBranches(t, h)
		h.send(key(tea.KeyF6, 0))
		if h.m.ui.Branches.Open {
			t.Error("F6 left the popup open")
		}
	})
}

// TestAltGrCard is criterion 67.2's approval card: focused, it lets none of
// AltGr's characters into the draft; the request stays pending and nothing
// is written to the CLI.
func TestAltGrCard(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, cli := readyScript(t, "claude-code", true, event.Started{SessionID: "s1", ApprovalMode: event.ModeManual})
		in, _ := json.Marshal(map[string]string{"command": "echo hi"})
		pushEvents(h, "claude-code", evOf(event.TurnStarted, nil),
			evOf(event.ApprovalRequest, event.Approval{ID: "r1", Tool: "Bash", ToolID: "t1", Input: in}))
		h.send(key(tea.KeyF2, 0))
		if h.m.ui.Card != "r1" {
			t.Fatalf("F2: card %q, want r1", h.m.ui.Card)
		}
		sent, _, _ := cli.sess.calls()
		for _, r := range altGrChars {
			sendAll(h, altGrKeys(r))
		}
		if h.m.edit.Text() != "" || h.m.ui.Card != "r1" {
			t.Errorf("AltGr's characters with the card focused: the draft %q, card %q", h.m.edit.Text(), h.m.ui.Card)
		}
		if a := h.m.State().Approvals["r1"]; a == nil || a.Status != state.Pending {
			t.Errorf("the request: %+v, want pending", a)
		}
		if now, _, _ := cli.sess.calls(); len(now) != len(sent) {
			t.Errorf("written to the CLI: %q", now[len(sent):])
		}
	})
}

// uiState is what a key could change besides the draft: popups, windows,
// views, the focus, a toast.
func uiState(h *harness) string {
	u := h.m.ui
	return fmt.Sprintf("plus %v models %v modes %v branches %v scope %v keys %v draftwin %v file %v finder %v search %v explorer %v source %v graph %v card %q focus %q toast %q view %q %d inspect %q details %v zoom %v worktrees %v",
		u.Plus, u.Models.Open, u.Modes.Open, u.Branches.Open, u.Scope.Open, u.Keys, u.DraftWin.Open, u.File.Open, u.Finder.Open, u.Search.Open,
		u.Explorer.Open, u.Source.Open, u.Graph.Open, u.Card, u.Focus, u.Toast, u.View, u.ViewEntry, u.Inspect, u.Details, u.Zoom, h.m.wt != nil)
}

// TestAltGrShortcuts is criterion 67.3: each of AltGr's characters typed in
// the draft adds itself and changes nothing else; Ctrl with the left Alt's
// keys change nothing at all; a dead key's composed character is typed.
func TestAltGrShortcuts(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := readyScript(t, claude.ProviderID, false, event.Started{})
		was := uiState(h)
		for i, r := range altGrChars {
			sendAll(h, altGrKeys(r))
			if got, want := h.m.edit.Text(), altGrChars[:i+len(string(r))]; got != want {
				t.Fatalf("AltGr's %c: the draft %q, want %q", r, got, want)
			}
			if now := uiState(h); now != was {
				t.Fatalf("AltGr's %c changed more than the draft:\n%s\nwas\n%s", r, now, was)
			}
		}
		for _, r := range altGrChars {
			sendAll(h, ctrlLeftAltKeys(r))
			if h.m.edit.Text() != altGrChars || uiState(h) != was {
				t.Fatalf("Ctrl with the left Alt and %c: the draft %q, the dashboard\n%s", r, h.m.edit.Text(), uiState(h))
			}
		}
		h.send(tea.KeyPressMsg{Code: 'ê', Text: "ê"})
		if h.m.edit.Text() != altGrChars+"ê" {
			t.Errorf("a dead key's ê: the draft %q", h.m.edit.Text())
		}
	})
}
