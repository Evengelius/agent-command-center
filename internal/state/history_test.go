package state

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// Plan 75, design D1a for the state: the first message the owner sends in a
// session started new is the text of its title, taken once the CLI accepted
// it; a resumed session has none; a message a turn the CLI opened or a
// refused one never stands in for it. The application makes the title.

// recordedText is the Text of the record the last step asked for, and whether
// it asked for one.
func recordedText(b *bench) (string, bool) {
	for _, c := range b.cmds {
		if c.Kind == CmdRecordSession {
			return c.Text, true
		}
	}
	return "", false
}

func TestFirstPrompt(t *testing.T) {
	b := newBench(t).ready()
	b.ev(event.SessionStarted, "", event.Started{SessionID: "s-1", Cwd: project})
	b.do(Send{Text: "remember the code word"})
	b.eq("not before the CLI accepts it", b.s.Session.FirstPrompt, "")
	b.ev(event.UserMessage, "", event.User{Text: "remember the code word"})
	b.ev(event.TurnStarted, "", nil)
	b.eq("accepted", b.s.Session.FirstPrompt, "remember the code word")
	b.turnEnd("", "completed")
	if text, ok := recordedText(b); !ok || text != "remember the code word" {
		t.Errorf("the first record: %q, %t", text, ok)
	}

	// A second message never replaces it, and every record names it.
	b.do(Send{Text: "and now?"})
	b.ev(event.UserMessage, "", event.User{Text: "and now?"})
	b.ev(event.TurnStarted, "", nil)
	b.turnEnd("", "completed")
	if text, ok := recordedText(b); !ok || text != "remember the code word" {
		t.Errorf("the second record: %q, %t", text, ok)
	}

	// The session crashes and starts again: the first message stays.
	b.exit(1)
	b.do(Resume{})
	b.eq("after a crash resume", b.s.Session.FirstPrompt, "remember the code word")
	b.eq("and the first message was accepted", b.s.Session.Prompted, true)
}

func TestFirstPromptNone(t *testing.T) {
	// A resume from the home screen has no title, whatever the owner sends.
	b := newBench(t).resumedReady("claude-code", "s-1", "s-1")
	b.do(Send{Text: "go on"})
	b.ev(event.UserMessage, "", event.User{Text: "go on"})
	b.ev(event.TurnStarted, "", nil)
	b.turnEnd("", "completed")
	if text, ok := recordedText(b); !ok || text != "" || b.s.Session.FirstPrompt != "" {
		t.Errorf("a resumed session: record %q, %t, first prompt %q", text, ok, b.s.Session.FirstPrompt)
	}

	// A first message of images alone has no text: the next one is not the first.
	c := newBench(t).ready()
	c.ev(event.SessionStarted, "", event.Started{SessionID: "s-2", Cwd: project})
	c.do(Send{Text: "", Images: []Attachment{{Path: `C:\x.png`, Name: "x.png"}}})
	c.ev(event.TurnStarted, "", nil)
	c.turnEnd("", "completed")
	c.do(Send{Text: "second"})
	c.ev(event.UserMessage, "", event.User{Text: "second"})
	c.ev(event.TurnStarted, "", nil)
	c.turnEnd("", "completed")
	if text, _ := recordedText(c); text != "" {
		t.Errorf("a later message took the title: %q", text)
	}

	// A turn the CLI opened is no message of the owner's.
	d := newBench(t).ready()
	d.ev(event.SessionStarted, "", event.Started{SessionID: "s-3", Cwd: project})
	d.ev(event.MessageMarker, "", event.Marker{MessageID: "m1", Name: "message_start"})
	d.eq("a turn the CLI opened", d.s.Session.Prompted, false)
}

// TestFirstPromptBounded is design D1a's bound: the session holds the start
// of a very long first message, cut between two characters, and says it is
// only the start; a short one is kept whole.
func TestFirstPromptBounded(t *testing.T) {
	long := strings.Repeat("日", MaxFirstPrompt) // 3 bytes a character: the bound falls inside one
	b := newBench(t).ready()
	b.ev(event.SessionStarted, "", event.Started{SessionID: "s-1", Cwd: project})
	b.do(Send{Text: long})
	b.ev(event.UserMessage, "", event.User{Text: long})
	b.ev(event.TurnStarted, "", nil)
	p := b.s.Session.FirstPrompt
	if len(p) > MaxFirstPrompt || len(p) < MaxFirstPrompt-3 || !utf8.ValidString(p) || !strings.HasPrefix(long, p) || !b.s.Session.FirstPromptCut {
		t.Errorf("the first prompt: %d bytes, valid %t, cut %t", len(p), utf8.ValidString(p), b.s.Session.FirstPromptCut)
	}
	b.turnEnd("", "completed")
	var rec Command
	for _, c := range b.cmds {
		if c.Kind == CmdRecordSession {
			rec = c
		}
	}
	if rec.Text != p || !rec.Partial {
		t.Errorf("the record: %d bytes, partial %t", len(rec.Text), rec.Partial)
	}

	c := newBench(t).ready()
	c.ev(event.SessionStarted, "", event.Started{SessionID: "s-2", Cwd: project})
	c.do(Send{Text: "short"})
	c.ev(event.UserMessage, "", event.User{Text: "short"})
	c.ev(event.TurnStarted, "", nil)
	c.turnEnd("", "completed")
	if text, ok := recordedText(c); !ok || text != "short" || c.s.Session.FirstPromptCut {
		t.Errorf("a short message: %q, cut %t", text, c.s.Session.FirstPromptCut)
	}
}
