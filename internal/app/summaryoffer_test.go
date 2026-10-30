package app

import (
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/provider/codex"
	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 70: the one-time summary suggestion after a resume from the home
// screen, through the real adapters over the fake CLIs; criteria 70.1 and
// 70.3 to 70.8 at the application's level.

const summaryQuestion = "Want a summary of your earlier work?"

// offer is a session of cli resumed from the home screen with Ctrl+R in
// project dir, its last session id.
type offer struct {
	h            *harness
	wc           *wireClaude
	wx           *wireCodex
	ps           *input.Projects
	dir, cli, id string
}

var offerCLIs = []string{claude.ProviderID, codex.ProviderID}

// offerRig is the home screen on a store holding cli's last session in a
// new project folder, resumed with Ctrl+R, the suggestion shown; the draft
// store d (nil for none) and the Open dialog's answers as given.
func offerRig(t *testing.T, cli string, d *input.Drafts, answers ...[]string) *offer {
	t.Helper()
	o := &offer{wc: &wireClaude{}, wx: &wireCodex{thread: "th-new", auto: true}, dir: t.TempDir(), cli: cli, id: "s-0070-asked"}
	if cli == codex.ProviderID {
		o.id = "th-0070-asked"
	}
	o.ps = withSession(t, o.dir, cli, o.id, testNow.Add(-time.Hour))
	o.home(t, d, answers...)
	o.resume(t, o.id)
	if s := o.h.m.State(); !s.Session.SummaryOffered || s.Session.Phase != state.Ready || !says(o.h.frame(), summaryQuestion) {
		t.Fatalf("%s: offered %v, session %q:\n%s", cli, s.Session.SummaryOffered, s.Session.Phase, o.h.frame())
	}
	return o
}

// home opens a dashboard on o's store at 120x40, o's CLI chosen.
func (o *offer) home(t *testing.T, d *input.Drafts, answers ...[]string) {
	t.Helper()
	opt := Options{Providers: []provider.Provider{
		installedClaude{&claude.Provider{Spawner: o.wc, Now: func() time.Time { return testNow }}},
		installedCodex{&codex.Provider{Spawner: o.wx, Now: func() time.Time { return testNow }}},
	}, Cwd: o.dir, Projects: o.ps, Drafts: d}
	if len(answers) > 0 {
		opt.PickImages = (&picker{answers: answers}).pick
	}
	o.h = newHarness(t, opt)
	o.h.settle()
	o.h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
	if o.cli == codex.ProviderID {
		o.h.send(key(tea.KeyDown, 0))
	}
}

// resume is Ctrl+R on the home screen, the CLI naming named: Codex answers
// thread/resume with the thread asked for.
func (o *offer) resume(t *testing.T, named string) {
	t.Helper()
	o.h.send(ctrl('r'))
	if o.cli == claude.ProviderID {
		o.wc.last().emit(claudeInit(o.dir, named))
		o.h.settle()
	}
}

// lines are the owner's messages the CLI read, as written: Claude Code's
// user messages without the adapter's usage reads, Codex's turn/start.
func (o *offer) lines() []string {
	if o.cli == claude.ProviderID {
		return o.wc.written("user")
	}
	return o.wx.method("turn/start")
}

// sent is the text of each message the CLI read.
func (o *offer) sent(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, l := range o.lines() {
		var m struct {
			Message struct {
				Content json.RawMessage `json:"content"`
			} `json:"message"`
			Params struct {
				Input []struct{ Type, Text string } `json:"input"`
			} `json:"params"`
		}
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatalf("%s: %v", l, err)
		}
		text := ""
		var s string
		if json.Unmarshal(m.Message.Content, &s) == nil {
			text = s
		}
		for _, in := range m.Params.Input {
			if in.Type == "text" {
				text += in.Text
			}
		}
		out = append(out, text)
	}
	return out
}

// shapes are the messages as written with their text and request id taken
// out: what the CLI reads beside the owner's words.
func (o *offer) shapes(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, l := range o.lines() {
		var m map[string]any
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatal(err)
		}
		delete(m, "id")
		if msg, ok := m["message"].(map[string]any); ok {
			msg["content"] = ""
		}
		if p, ok := m["params"].(map[string]any); ok {
			for _, in := range p["input"].([]any) {
				in.(map[string]any)["text"] = ""
			}
		}
		b, _ := json.Marshal(m)
		out = append(out, string(b))
	}
	return out
}

// answer completes the turn of the owner's last message: Codex's fake does
// it by itself.
func (o *offer) answer(text string) {
	if o.cli == claude.ProviderID {
		o.wc.last().emit(claudeTurn(o.id, text)...)
	}
	o.h.settle()
}

// shown says the frame shows the suggestion.
func (o *offer) shown() bool { return says(o.h.frame(), summaryQuestion) }

// storeBytes is every file of the projects store, name and content.
func storeBytes(t *testing.T, ps *input.Projects) string {
	t.Helper()
	es, err := os.ReadDir(ps.Dir)
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	for _, e := range es {
		data, err := os.ReadFile(ps.Dir + string(os.PathSeparator) + e.Name())
		if err != nil {
			t.Fatal(err)
		}
		b.WriteString(e.Name() + "\n" + string(data) + "\n")
	}
	return b.String()
}

// TestSummaryOfferPrepare is criteria 70.5 and 70.7: Prepare summary puts
// the owner's request in the empty draft, the cursor at its end, nothing
// written; the suggestion gone and not back once the draft is cleared;
// Enter sends the request byte for byte as an ordinary message, with no
// setting changed.
func TestSummaryOfferPrepare(t *testing.T) {
	for _, cli := range offerCLIs {
		t.Run(cli, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				o := offerRig(t, cli, nil)
				h := o.h
				if !strings.Contains(h.frame(), "╭─ SESSION RESUMED · SUGGESTION ─") || !strings.Contains(h.frame(), "[ Prepare summary ]     [ Not now ]") ||
					!says(h.frame(), "The request will be added to your draft.") || !says(h.frame(), "Press Enter to send it.") {
					t.Fatalf("the suggestion:\n%s", h.frame())
				}
				// The settings the adapters can change: Claude Code's model,
				// effort and Ultracode, and mode; Codex's thread settings.
				changes := func() int {
					return len(o.wc.written("control_request/set_model", "control_request/apply_flag_settings", "control_request/set_permission_mode")) +
						len(o.wx.method("thread/settings/update"))
				}
				before := changes()
				store := storeBytes(t, o.ps)
				h.send(key(tea.KeyUp, 0))
				if h.m.ui.Summary != (render.SummaryFocus{On: true}) {
					t.Fatalf("↑ from the empty input: focus %+v", h.m.ui.Summary)
				}
				if !says(h.frame(), "←→ select · Enter choose · ↑ entries · Esc back to the input") {
					t.Errorf("no footer of the suggestion's keys:\n%s", h.frame())
				}
				h.send(key(tea.KeyEnter, 0))
				if got := h.m.edit.Text(); got != summaryRequest || h.m.edit.Cursor() != len(summaryRequest) {
					t.Fatalf("the draft %q, cursor %d", got, h.m.edit.Cursor())
				}
				if n := len(o.sent(t)); n != 0 {
					t.Fatalf("Prepare summary wrote %d messages", n)
				}
				if got := storeBytes(t, o.ps); got != store {
					t.Errorf("Prepare summary changed the store:\n%s\nwas\n%s", got, store)
				}
				if s := h.m.State(); s.Session.SummaryOffered || h.m.ui.Summary.On || o.shown() || h.m.ui.Focus != "B" {
					t.Errorf("after Prepare: offered %v, focus %+v %q, shown %v", s.Session.SummaryOffered, h.m.ui.Summary, h.m.ui.Focus, o.shown())
				}
				h.send(ctrl('a'))
				h.send(key(tea.KeyBackspace, 0))
				if h.m.edit.Text() != "" || o.shown() || h.m.State().Session.SummaryOffered {
					t.Errorf("the draft cleared: %q, shown %v", h.m.edit.Text(), o.shown())
				}
				h.send(ctrl('z'))
				if h.m.edit.Text() != summaryRequest {
					t.Fatalf("undone: %q", h.m.edit.Text())
				}
				h.send(key(tea.KeyEnter, 0))
				if got := o.sent(t); len(got) != 1 || got[0] != summaryRequest {
					t.Fatalf("sent %q, want the request byte for byte", got)
				}
				o.answer("the summary")
				h.send(tea.PasteMsg{Content: "go on"})
				h.send(key(tea.KeyEnter, 0))
				o.answer("going on")
				if got := o.sent(t); !slices.Equal(got, []string{summaryRequest, "go on"}) {
					t.Fatalf("sent %q", got)
				}
				if sh := o.shapes(t); len(sh) != 2 || sh[0] != sh[1] {
					t.Errorf("the request is not written as an ordinary message:\n%s\n%s", sh[0], sh[1])
				}
				if n := changes() - before; n != 0 {
					t.Errorf("%d settings changes written after the resume", n)
				}
				if o.shown() || h.m.State().Session.SummaryOffered {
					t.Error("the suggestion back after the turns")
				}
			})
		})
	}
}

// TestSummaryOfferNotNow is criterion 70.6: Not now, by keys and by a
// click, withdraws the offer, writes nothing, leaves the draft and its
// images as they are and the store unchanged; ↑ is the entry cursor's again.
func TestSummaryOfferNotNow(t *testing.T) {
	for _, cli := range offerCLIs {
		t.Run(cli, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				o := offerRig(t, cli, nil)
				h := o.h
				store := storeBytes(t, o.ps)
				h.send(key(tea.KeyUp, 0))
				h.send(key(tea.KeyRight, 0))
				if h.m.ui.Summary != (render.SummaryFocus{On: true, NotNow: true}) {
					t.Fatalf("→: focus %+v", h.m.ui.Summary)
				}
				h.send(key(tea.KeyLeft, 0))
				if h.m.ui.Summary != (render.SummaryFocus{On: true}) {
					t.Fatalf("←: focus %+v", h.m.ui.Summary)
				}
				h.send(key(tea.KeyRight, 0))
				h.send(key(tea.KeyEnter, 0))
				if s := h.m.State(); s.Session.SummaryOffered || o.shown() || h.m.ui.Summary.On {
					t.Errorf("after Not now: offered %v, shown %v, focus %+v", s.Session.SummaryOffered, o.shown(), h.m.ui.Summary)
				}
				if h.m.edit.Text() != "" || len(o.sent(t)) != 0 {
					t.Errorf("Not now: draft %q, sent %q", h.m.edit.Text(), o.sent(t))
				}
				if got := storeBytes(t, o.ps); got != store {
					t.Errorf("the store changed:\n%s\nwas\n%s", got, store)
				}
				h.send(key(tea.KeyUp, 0))
				if h.m.ui.Summary.On {
					t.Errorf("↑ with no suggestion: focus %+v", h.m.ui.Summary)
				}

				// A click on Not now, with the owner's words in the draft.
				c := offerRig(t, cli, nil)
				c.h.send(tea.PasteMsg{Content: "my own words"})
				clickOn(c.h, render.SummaryPanel, render.SummaryNotNow)
				if s := c.h.m.State(); s.Session.SummaryOffered || c.shown() {
					t.Errorf("after a click on Not now: offered %v, shown %v", s.Session.SummaryOffered, c.shown())
				}
				if c.h.m.edit.Text() != "my own words" || len(c.sent(t)) != 0 {
					t.Errorf("a click on Not now: draft %q, sent %q", c.h.m.edit.Text(), c.sent(t))
				}
			})
		})
	}
}

// TestSummaryOfferDraftKept is criterion 70.5's protection: Prepare summary
// over text, an attached image or a draft restored at the session's start
// changes nothing, writes nothing, says why in a toast and keeps the
// suggestion.
func TestSummaryOfferDraftKept(t *testing.T) {
	kept := func(t *testing.T, o *offer, text string, images int) {
		t.Helper()
		if o.h.m.ui.Toast != summaryBusy {
			t.Errorf("toast %q, want %q", o.h.m.ui.Toast, summaryBusy)
		}
		if o.h.m.edit.Text() != text || len(o.h.m.attach) != images {
			t.Errorf("the draft %q with %d images, want %q with %d", o.h.m.edit.Text(), len(o.h.m.attach), text, images)
		}
		if !o.h.m.State().Session.SummaryOffered || !o.shown() {
			t.Error("the suggestion gone")
		}
		if n := len(o.sent(t)); n != 0 {
			t.Errorf("%d messages written", n)
		}
	}
	for _, cli := range offerCLIs {
		t.Run(cli, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				// Text typed, then a click on Prepare summary.
				o := offerRig(t, cli, nil)
				o.h.send(tea.PasteMsg{Content: "my own words"})
				clickOn(o.h, render.SummaryPanel, render.SummaryPrepare)
				kept(t, o, "my own words", 0)

				// An image and no text: ↑ gives the suggestion the focus;
				// Enter changes nothing, the focus kept.
				files := rcFiles(t)
				i := offerRig(t, cli, nil, []string{files["png"]})
				upload(t, i.h)
				if len(i.h.m.attach) != 1 {
					t.Fatalf("attached %d images", len(i.h.m.attach))
				}
				i.h.send(key(tea.KeyUp, 0))
				i.h.send(key(tea.KeyEnter, 0))
				kept(t, i, "", 1)
				if i.h.m.ui.Summary != (render.SummaryFocus{On: true}) {
					t.Errorf("the focus after a refused Prepare: %+v", i.h.m.ui.Summary)
				}

				// A draft restored at the session's start.
				d := input.Drafts{Dir: t.TempDir()}
				r := offerRig(t, cli, &d)
				r.h.send(tea.PasteMsg{Content: "words from before"})
				r.h.crash()
				r.home(t, &d)
				r.resume(t, r.id)
				if r.h.m.edit.Text() != "words from before" || r.h.m.ui.Toast != "draft restored" {
					t.Fatalf("restored %q, toast %q", r.h.m.edit.Text(), r.h.m.ui.Toast)
				}
				clickOn(r.h, render.SummaryPanel, render.SummaryPrepare)
				kept(t, r, "words from before", 0)
			})
		})
	}
}

// TestSummaryOfferFirstMessage is criterion 70.1 at the application's
// level: the owner's first message hides the suggestion while it is sent
// and withdraws the offer once the CLI accepts it, never to come back; a
// write that fails restores the draft and the suggestion with it.
func TestSummaryOfferFirstMessage(t *testing.T) {
	for _, cli := range offerCLIs {
		t.Run(cli, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				o := offerRig(t, cli, nil)
				o.h.send(tea.PasteMsg{Content: "an ordinary first message"})
				o.h.send(key(tea.KeyEnter, 0))
				if got := o.sent(t); !slices.Equal(got, []string{"an ordinary first message"}) {
					t.Fatalf("sent %q", got)
				}
				if o.shown() {
					t.Error("the suggestion drawn under the message being sent")
				}
				o.answer("OK")
				if s := o.h.m.State(); s.Session.SummaryOffered || o.shown() || s.Turn.Phase != state.Idle {
					t.Errorf("after the turn: offered %v, shown %v, turn %q", s.Session.SummaryOffered, o.shown(), s.Turn.Phase)
				}

				f := offerRig(t, cli, nil)
				if cli == claude.ProviderID {
					f.wc.last().refuse.Store(true)
				} else {
					f.wx.last().refuse.Store(true)
				}
				f.h.send(tea.PasteMsg{Content: "lost on the way"})
				f.h.send(key(tea.KeyEnter, 0))
				if s := f.h.m.State(); s.Turn.Outcome != "failed" || f.h.m.edit.Text() != "lost on the way" {
					t.Fatalf("the failed write: outcome %q, draft %q", s.Turn.Outcome, f.h.m.edit.Text())
				}
				if !f.h.m.State().Session.SummaryOffered || !f.shown() {
					t.Error("a write that failed withdrew the suggestion")
				}
			})
		})
	}
}

// TestSummaryOfferKeys is criterion 70.3: the suggestion's keys while it
// has the focus, the way back to the input, the entry cursor before it;
// Enter on the empty input does nothing; a key it does not take, or text a
// paste puts in the draft, ends its focus.
func TestSummaryOfferKeys(t *testing.T) {
	for _, cli := range offerCLIs {
		t.Run(cli, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				o := offerRig(t, cli, nil)
				h := o.h
				focus := func(what string, want render.SummaryFocus) {
					t.Helper()
					if h.m.ui.Summary != want {
						t.Errorf("%s: focus %+v, want %+v", what, h.m.ui.Summary, want)
					}
				}
				h.send(key(tea.KeyEnter, 0))
				if s := h.m.State(); !s.Session.SummaryOffered || h.m.edit.Text() != "" || len(o.sent(t)) != 0 || h.m.ui.Summary.On {
					t.Fatalf("Enter on the empty input: offered %v, draft %q, sent %q", s.Session.SummaryOffered, h.m.edit.Text(), o.sent(t))
				}
				h.send(key(tea.KeyUp, 0))
				focus("↑", render.SummaryFocus{On: true})
				h.send(key(tea.KeyUp, 0))
				focus("↑ with no entry to stop on", render.SummaryFocus{On: true})
				// A turn the CLI opens itself before the owner's first message,
				// a call in it: the suggestion waits for its end, the offer
				// kept, and the call is a stop above it.
				pushEvents(h, cli, evOf(event.MessageMarker, event.Marker{Name: "message_start"}),
					evOf(event.ToolStarted, event.Tool{ToolID: "t1", Name: "Read", Input: json.RawMessage(`{"file_path":"notes.md"}`)}))
				if o.shown() || h.m.ui.Summary.On || !h.m.State().Session.SummaryOffered {
					t.Errorf("a turn of the CLI's: shown %v, focus %+v, offered %v", o.shown(), h.m.ui.Summary, h.m.State().Session.SummaryOffered)
				}
				pushEvents(h, cli, evOf(event.ToolCompleted, event.Tool{ToolID: "t1", Name: "Read", Output: "notes"}),
					evOf(event.TurnCompleted, event.TurnEnd{Outcome: "completed"}))
				if !o.shown() || !h.m.State().Session.SummaryOffered {
					t.Fatalf("after the CLI's turn: shown %v: %s", o.shown(), h.frame())
				}
				call := slices.IndexFunc(h.m.State().Conversation, func(e state.Entry) bool { return e.Role == "tool" }) + 1
				h.send(key(tea.KeyUp, 0))
				focus("↑ after the CLI's turn", render.SummaryFocus{On: true})
				h.send(key(tea.KeyUp, 0))
				focus("↑ to the call", render.SummaryFocus{})
				if call == 0 || h.m.ui.Entry != call {
					t.Errorf("↑ to the call: entry %d, want %d", h.m.ui.Entry, call)
				}
				h.send(key(tea.KeyDown, 0))
				focus("↓ past the newest entry", render.SummaryFocus{On: true})
				if h.m.ui.Entry != 0 {
					t.Errorf("↓ past the newest entry: entry %d", h.m.ui.Entry)
				}
				h.send(key(tea.KeyRight, 0))
				h.send(key(tea.KeyEscape, 0))
				focus("Esc", render.SummaryFocus{})
				if h.m.ui.Focus != "B" || !o.shown() {
					t.Errorf("Esc: focus %q, shown %v", h.m.ui.Focus, o.shown())
				}
				h.send(key(tea.KeyUp, 0))
				h.send(key(tea.KeyDown, 0))
				focus("↓", render.SummaryFocus{})
				h.send(key(tea.KeyUp, 0))
				h.send(typed('a'))
				focus("typing", render.SummaryFocus{})
				if h.m.edit.Text() != "a" {
					t.Errorf("typing: the draft %q", h.m.edit.Text())
				}
				h.send(key(tea.KeyBackspace, 0))
				h.send(key(tea.KeyUp, 0))
				h.send(key(tea.KeyF2, 0))
				focus("F2", render.SummaryFocus{})
				if h.m.ui.Toast != "no approval waiting" {
					t.Errorf("F2: toast %q", h.m.ui.Toast)
				}
				h.send(key(tea.KeyUp, 0))
				focus("↑ once more", render.SummaryFocus{On: true})
				h.send(tea.PasteMsg{Content: "pasted words"})
				focus("a paste", render.SummaryFocus{})
				h.send(key(tea.KeyEnter, 0))
				if got := o.sent(t); !slices.Equal(got, []string{"pasted words"}) {
					t.Errorf("Enter after the paste sent %q", got)
				}
			})
		})
	}
}

// TestSummaryOfferScrolled is criterion 70.3 with a conversation taller
// than zone B: scrolled up, the suggestion out of view, ↑ from the input is
// the entry cursor's as before; ↓ past the newest entry gives the
// suggestion the focus and B follows its end; ↑ with text in the draft
// moves in the draft.
func TestSummaryOfferScrolled(t *testing.T) {
	for _, cli := range offerCLIs {
		t.Run(cli, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				o := offerRig(t, cli, nil)
				h := o.h
				var lines []string
				for i := 1; i <= 60; i++ {
					lines = append(lines, "line "+strings.Repeat("x", i%7+1))
				}
				// A turn the CLI opens itself: a call, then a long answer.
				pushEvents(h, cli, evOf(event.MessageMarker, event.Marker{Name: "message_start"}),
					evOf(event.ToolStarted, event.Tool{ToolID: "t1", Name: "Read", Input: json.RawMessage(`{"file_path":"notes.md"}`)}),
					evOf(event.ToolCompleted, event.Tool{ToolID: "t1", Name: "Read", Output: "notes"}),
					evOf(event.MessageCompleted, event.Completed{MessageID: "m1", Text: strings.Join(lines, "\n")}),
					evOf(event.TurnCompleted, event.TurnEnd{Outcome: "completed"}))
				if !o.shown() || !h.m.screen.Conv.Summary {
					t.Fatalf("after the CLI's turn, B following its end: shown %v:\n%s", o.shown(), h.frame())
				}
				h.send(key(tea.KeyPgUp, 0))
				if o.shown() || h.m.screen.Conv.Summary {
					t.Fatalf("PgUp: the suggestion still in view:\n%s", h.frame())
				}
				h.send(key(tea.KeyUp, 0))
				call := slices.IndexFunc(h.m.State().Conversation, func(e state.Entry) bool { return e.Role == "tool" }) + 1
				if h.m.ui.Summary.On || call == 0 || h.m.ui.Entry != call {
					t.Errorf("↑ with the suggestion out of view: focus %+v, entry %d, want the call's %d", h.m.ui.Summary, h.m.ui.Entry, call)
				}
				h.frame() // the cursor's entry revealed, as the program draws it
				h.send(key(tea.KeyDown, 0))
				if h.m.ui.Summary != (render.SummaryFocus{On: true}) {
					t.Errorf("↓ past the newest entry: focus %+v", h.m.ui.Summary)
				}
				if !o.shown() || !h.m.screen.Conv.Summary {
					t.Errorf("↓ past the newest entry: the suggestion not in view:\n%s", h.frame())
				}
				h.send(key(tea.KeyEscape, 0))
				h.send(tea.PasteMsg{Content: "line 1\nline 2"})
				h.send(key(tea.KeyUp, 0))
				if h.m.ui.Summary.On || h.m.edit.CursorLine() != 0 {
					t.Errorf("↑ with text in the draft: focus %+v, the draft's line %d, want 0", h.m.ui.Summary, h.m.edit.CursorLine())
				}
			})
		})
	}
}

// TestSummaryOfferMouse is criterion 70.4: a click on Prepare summary acts
// without the focus; a click elsewhere in zone B leaves the offer.
func TestSummaryOfferMouse(t *testing.T) {
	for _, cli := range offerCLIs {
		t.Run(cli, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				o := offerRig(t, cli, nil)
				x, y := at(t, o.h, summaryQuestion, 0)
				clickAt(t, o.h, x+3, y)
				if !o.h.m.State().Session.SummaryOffered || !o.shown() || o.h.m.ui.Summary.On {
					t.Errorf("a click on the question: offered %v, shown %v, focus %+v", o.h.m.State().Session.SummaryOffered, o.shown(), o.h.m.ui.Summary)
				}
				clickOn(o.h, render.SummaryPanel, render.SummaryPrepare)
				if o.h.m.edit.Text() != summaryRequest || o.h.m.State().Session.SummaryOffered || o.shown() || o.h.m.ui.Focus != "B" {
					t.Errorf("a click on Prepare summary: draft %q, offered %v, focus %q", o.h.m.edit.Text(), o.h.m.State().Session.SummaryOffered, o.h.m.ui.Focus)
				}
				if n := len(o.sent(t)); n != 0 {
					t.Errorf("%d messages written", n)
				}
			})
		})
	}
}

// TestSummaryOfferNotOffered is criteria 70.1 and 70.8: no suggestion for
// a new session, for another session than the one asked for, or after a
// crash resume; zone B not drawn hides it, the offer kept.
func TestSummaryOfferNotOffered(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		none := func(what string, h *harness) {
			t.Helper()
			if s := h.m.State(); s.Session.SummaryOffered || says(h.frame(), summaryQuestion) {
				t.Errorf("%s: offered %v:\n%s", what, s.Session.SummaryOffered, h.frame())
			}
		}
		// START SESSION, the project's last session recorded.
		n := &offer{wc: &wireClaude{}, wx: &wireCodex{thread: "th-new", auto: true}, dir: t.TempDir(), cli: claude.ProviderID, id: "s-0070-asked"}
		n.ps = withSession(t, n.dir, n.cli, n.id, testNow.Add(-time.Hour))
		n.home(t, nil)
		n.h.send(key(tea.KeyEnter, 0))
		n.wc.last().emit(claudeInit(n.dir, "s-new"))
		n.h.settle()
		none("a new session", n.h)

		// Another session named.
		a := &offer{wc: &wireClaude{}, wx: &wireCodex{}, dir: t.TempDir(), cli: claude.ProviderID, id: "s-0070-asked"}
		a.ps = withSession(t, a.dir, a.cli, a.id, testNow.Add(-time.Hour))
		a.home(t, nil)
		a.resume(t, "s-0070-other")
		none("another session named", a.h)

		// A crash, then Ctrl+R: the crash resume.
		c := offerRig(t, claude.ProviderID, nil)
		c.wc.last().exit(1, "")
		c.h.settle()
		if s := c.h.m.State(); s.Session.Phase != state.Crashed || c.shown() {
			t.Fatalf("after the exit: %q, shown %v", s.Session.Phase, c.shown())
		}
		c.h.send(ctrl('r'))
		c.wc.last().emit(claudeInit(c.dir, c.id))
		c.h.settle()
		if s := c.h.m.State(); s.Session.Phase != state.Ready || s.Session.ID != c.id {
			t.Fatalf("the crash resume: %q, %q", s.Session.Phase, s.Session.ID)
		}
		none("a crash resume", c.h)

		// Zone B not drawn: under 100 columns zone A's tree takes its place.
		x := offerRig(t, codex.ProviderID, nil)
		x.h.send(tea.WindowSizeMsg{Width: 80, Height: 24})
		if !x.shown() {
			t.Fatalf("not shown at 80x24:\n%s", x.h.frame())
		}
		x.h.send(key(tea.KeyF6, tea.ModShift)) // to zone A
		if x.h.m.ui.Focus != "A" || x.shown() || !x.h.m.State().Session.SummaryOffered {
			t.Errorf("zone A's tree in B's place: focus %q, shown %v, offered %v", x.h.m.ui.Focus, x.shown(), x.h.m.State().Session.SummaryOffered)
		}
		x.h.send(key(tea.KeyEscape, 0))
		if x.h.m.ui.Focus != "B" || !x.shown() {
			t.Errorf("back: focus %q, shown %v", x.h.m.ui.Focus, x.shown())
		}
	})
}
