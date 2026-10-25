package app

import (
	"context"
	"strings"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// TestWritePanic is plan 48's criterion 48.8 (design D5; audit finding 6): a
// panic in a write to the session ends the session, not the program: the
// session crashes with the panic as its reason, shown on its card, the CLI
// is closed, and the message is back in the editor. TestPanic covers a panic
// in Update itself.
func TestWritePanic(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, cli := readyScript(t, "claude-code", false, event.Started{SessionID: "s1", Version: "2.1.300", PermissionMode: "default", ApprovalMode: event.ModeManual})
		cli.sess.sendPanic = "the adapter broke"
		h.send(tea.PasteMsg{Content: "hello"})
		h.send(key(tea.KeyEnter, 0))
		s := h.m.State()
		if s.Session.Phase != state.Crashed || !strings.HasPrefix(s.Session.Reason, "writing to the CLI failed: panic: the adapter broke") || cli.sess.closed() != 1 || h.quit {
			t.Fatalf("session %q, reason %q, %d closes, quit %v; want Crashed with the panic, closed once, running", s.Session.Phase, s.Session.Reason, cli.sess.closed(), h.quit)
		}
		f := h.frame()
		for _, want := range []string{"The session ended unexpectedly", "panic: the adapter broke"} {
			if !strings.Contains(f, want) {
				t.Errorf("the card lacks %q:\n%s", want, f)
			}
		}
		if got := h.m.edit.Text(); got != "hello" {
			t.Errorf("the editor holds %q, want the message back", got)
		}
	})
}

// panicCLI is a CLI whose detection panics.
type panicCLI struct{ scriptCLI }

func (p *panicCLI) Detect(context.Context) provider.Detection { panic("detection broke") }

// TestDetectPanic is plan 48's criterion 48.8, the detection (design D5): a
// panic in an adapter's detection is its CLI's problem, and the startup
// screen goes on.
func TestDetectPanic(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cli := &panicCLI{scriptCLI{id: "codex", sess: newScriptSession()}}
		h := newHarness(t, Options{Providers: []provider.Provider{cli}, Cwd: t.TempDir()})
		h.settle()
		s := h.m.State()
		var codex state.CLI
		for _, c := range s.Startup.CLIs {
			if c.ID == "codex" {
				codex = c
			}
		}
		if s.Startup.Phase != state.Asking || codex.Installed || !strings.HasPrefix(codex.Problem, "detection failed: panic: detection broke") || h.quit {
			t.Fatalf("startup %q, codex %+v, quit %v; want asking, its problem the panic", s.Startup.Phase, codex, h.quit)
		}
	})
}
