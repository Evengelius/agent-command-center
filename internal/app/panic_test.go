package app

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// TestPanic is step 4, check 5: a panic inside Update, where the reducer
// runs, ends the real program loop with exit code 2, the draft written, the
// CLI's input closed and the resume command printed. The program runs in a
// child process, as the dashboard would.
func TestPanic(t *testing.T) {
	if os.Getenv("ACC_PANIC_CHILD") == "1" {
		panicChild()
		return
	}
	dir := t.TempDir()
	drafts, project, eof := filepath.Join(dir, "drafts"), filepath.Join(dir, "project"), filepath.Join(dir, "eof")
	if err := os.Mkdir(project, 0o700); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestPanic$", "-test.count=1")
	cmd.Env = append(os.Environ(), "ACC_PANIC_CHILD=1", "ACC_PANIC_DRAFTS="+drafts, "ACC_PANIC_PROJECT="+project, "ACC_PANIC_EOF="+eof)
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	done := make(chan error, 1)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	go func() { done <- cmd.Wait() }()
	var err error
	select {
	case err = <-done:
	case <-time.After(60 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatalf("the child did not end:\n%s", out.String())
	}
	var ee *exec.ExitError
	if !errors.As(err, &ee) || ee.ExitCode() != ExitPanic {
		t.Fatalf("child ended with %v; want exit code %d:\n%s", err, ExitPanic, out.String())
	}
	got := out.String()
	for _, want := range []string{
		"agent-command-center stopped on an internal error",
		"injected panic",
		"session s-panic · resume it in " + project + " with: claude --resume s-panic",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output lacks %q:\n%s", want, got)
		}
	}
	if stored := draftFile(t, input.Drafts{Dir: drafts}, project); string(stored) != draftText {
		t.Errorf("stored draft %q; want %q", stored, draftText)
	}
	if _, err := os.Stat(eof); err != nil {
		t.Errorf("the fake CLI saw no end of input: %v", err)
	}
}

// panicChild runs the program with a fake Claude Code, opens a session,
// gets its id, types a draft and panics; it exits as main does.
func panicChild() {
	w := &wireClaude{onEOF: func() { _ = os.WriteFile(os.Getenv("ACC_PANIC_EOF"), nil, 0o600) }}
	m := New(Options{
		Providers: []provider.Provider{installedClaude{&claude.Provider{Spawner: w}}},
		Cwd:       os.Getenv("ACC_PANIC_PROJECT"),
		Now:       time.Now,
		Drafts:    &input.Drafts{Dir: os.Getenv("ACC_PANIC_DRAFTS")},
		MockSpeed: 1,
	})
	step := 0
	updateHook = func(m *Model, msg tea.Msg) []tea.Cmd {
		s := m.s
		switch {
		case step == 0 && s.Startup.Phase == state.Asking:
			step++
			m.ui.Choice = claude.ProviderID
			return []tea.Cmd{func() tea.Msg { return tea.KeyPressMsg{Code: tea.KeyEnter} }}
		case step == 1 && s.Session.Phase == state.Ready && m.draftReady:
			step++
			return []tea.Cmd{tea.Sequence(
				func() tea.Msg { return tea.PasteMsg{Content: "hello"} },
				func() tea.Msg { return tea.KeyPressMsg{Code: tea.KeyEnter} })}
		case step == 2 && s.Turn.Phase == state.Submitted:
			step++
			p, project := w.last(), s.Context.Project
			return []tea.Cmd{func() tea.Msg {
				p.emit(`{"type":"system","subtype":"init","cwd":`+quote(project)+`,"session_id":"s-panic","model":"claude-haiku-5-5","permissionMode":"default"}`,
					`{"type":"result","subtype":"success","is_error":false,"duration_ms":1,"result":"ok","session_id":"s-panic"}`)
				return nil
			}}
		case step == 3 && s.Session.ID == "s-panic" && s.Turn.Phase == state.Idle:
			step++
			m.edit.SetText(draftText)
			panic("injected panic")
		}
		return nil
	}
	code, out := Run(m, tea.WithInput(nil), tea.WithOutput(io.Discard), tea.WithWindowSize(120, 40), tea.WithoutSignals())
	for _, l := range out {
		fmt.Fprintln(os.Stderr, l)
	}
	os.Exit(code)
}
