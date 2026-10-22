package state

import (
	"testing"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// L5: Codex has no slash commands through app-server; one typed in the
// dashboard is held back once, with a note of the dashboard's, then sent if
// the owner sends it again. Claude Code runs its own, so they always go.

func readyOn(t *testing.T, provider string) *bench {
	b := newBench(t)
	b.asking().do(Confirm{Provider: provider, Project: project})
	b.do(ValidationOK{Provider: provider, ProviderName: provider, Project: project})
	b.ev(event.SessionSpawned, "", event.Spawned{Dir: project})
	b.ev(event.SessionReady, "", event.Ready{})
	return b
}

func TestSlashCommand(t *testing.T) {
	for _, c := range []struct{ text, want string }{
		{"/status", "/status"}, {"  /model gpt-6\nmore", "/model"}, {"/review-branch", "/review-branch"},
		{"/", ""}, {"/Status", ""}, {"/path/to/file is wrong", ""}, {"//x", ""}, {"say /status", ""}, {"/1", ""},
	} {
		if got := slashCommand(c.text); got != c.want {
			t.Errorf("slashCommand(%q) = %q, want %q", c.text, got, c.want)
		}
	}
}

func TestSlashHeldOnCodex(t *testing.T) {
	b := readyOn(t, "codex")
	b.do(Send{Text: "/status"})
	b.eq("commands", b.kinds(), "")
	b.eq("turn", b.s.Turn.Phase, Idle)
	b.eq("turns", b.s.Turn.Count, 0)
	b.eq("draft kept", b.s.Turn.Draft, "/status")
	if n := len(b.s.Conversation); n != 1 {
		t.Fatalf("%d entries, want the note only", n)
	}
	e := b.s.Conversation[0]
	b.eq("role", e.Role, "dashboard")
	b.eq("note", e.Text, "/status is a command of Codex's own terminal; it does not work here, so nothing was sent. Zone A's quota windows show the 5h and weekly limits.\nPress Enter again to send it to the model as a message.")

	b.do(Send{Text: "/status"}) // again: sent as a message
	b.eq("second send", b.kinds(), string(CmdWrite))
	b.eq("turn", b.s.Turn.Phase, Submitted)
	b.eq("held cleared", b.s.Turn.SlashHeld, "")
}

func TestSlashOtherCommandHeldAgain(t *testing.T) {
	b := readyOn(t, "codex")
	b.do(Send{Text: "/model"})
	b.eq("held", b.kinds(), "")
	b.eq("note", b.s.Conversation[0].Text, "/model is a command of Codex's own terminal; it does not work here, so nothing was sent.\nPress Enter again to send it to the model as a message.")
	b.do(Send{Text: "/status"}) // a different text: held too
	b.eq("different text", b.kinds(), "")
	b.eq("notes", len(b.s.Conversation), 2)
}

func TestSlashSentOnClaude(t *testing.T) {
	b := readyOn(t, "claude-code")
	b.do(Send{Text: "/usage"})
	b.eq("sent at once", b.kinds(), string(CmdWrite))
	b.eq("entries", len(b.s.Conversation), 0)
}

func TestPathNotHeldOnCodex(t *testing.T) {
	b := readyOn(t, "codex")
	b.do(Send{Text: "/tmp/x.txt is empty"})
	b.eq("sent at once", b.kinds(), string(CmdWrite))
}
