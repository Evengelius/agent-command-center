package app

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/fixture"
	"github.com/Evengelius/agent-command-center/internal/proc"
	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// wireClaude is a claude process the test speaks for: it records each
// spawn's argv and every stdin line, answers initialize, and writes to
// stdout and stderr what the test gives it; it can exit with any code.
type wireClaude struct {
	mu     sync.Mutex
	argv   [][]string
	lines  []string
	procs  []*wireProc
	noInit bool   // initialize is never answered: the session never gets Ready
	onEOF  func() // called when stdin reaches its end
}

type wireProc struct {
	inR        *io.PipeReader
	inW        *io.PipeWriter
	outR, errR *io.PipeReader
	outW, errW *io.PipeWriter
	wmu        sync.Mutex // one stdout writer at a time
	once       sync.Once
	done       chan struct{}
	code       int
	refuse     atomic.Bool // writes to stdin fail
}

func (w *wireClaude) Spawn(sp proc.Spec) (proc.Process, error) {
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	errR, errW := io.Pipe()
	p := &wireProc{inR: inR, inW: inW, outR: outR, outW: outW, errR: errR, errW: errW, done: make(chan struct{})}
	w.mu.Lock()
	w.argv = append(w.argv, append([]string{sp.Path}, sp.Args...))
	w.procs = append(w.procs, p)
	noInit := w.noInit
	w.mu.Unlock()
	go func() {
		sc := bufio.NewScanner(inR)
		sc.Buffer(make([]byte, 64<<10), 64<<20)
		for sc.Scan() {
			l := sc.Text()
			w.mu.Lock()
			w.lines = append(w.lines, l)
			w.mu.Unlock()
			var m struct {
				Type      string `json:"type"`
				RequestID string `json:"request_id"`
				Request   struct {
					Subtype string `json:"subtype"`
				} `json:"request"`
			}
			_ = json.Unmarshal([]byte(l), &m)
			if m.Type == "control_request" && m.Request.Subtype == "initialize" && !noInit {
				id, _ := json.Marshal(m.RequestID)
				p.emit(`{"type":"control_response","response":{"subtype":"success","request_id":` + string(id) + `,"response":{}}}`)
			}
		}
		if w.onEOF != nil {
			w.onEOF()
		}
		p.exit(0, "") // end of input: claude -p exits
	}()
	return p, nil
}

// last is the latest process.
func (w *wireClaude) last() *wireProc {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.procs[len(w.procs)-1]
}

func (w *wireClaude) spawns() [][]string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([][]string(nil), w.argv...)
}

// written is every stdin line of the given types, as written.
func (w *wireClaude) written(types ...string) []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	var out []string
	for _, l := range w.lines {
		var m struct {
			Type    string `json:"type"`
			Request struct {
				Subtype string `json:"subtype"`
			} `json:"request"`
		}
		_ = json.Unmarshal([]byte(l), &m)
		for _, t := range types {
			if t == m.Type || t == m.Type+"/"+m.Request.Subtype {
				out = append(out, l)
			}
		}
	}
	return out
}

func (p *wireProc) emit(lines ...string) {
	p.wmu.Lock()
	defer p.wmu.Unlock()
	for _, l := range lines {
		if _, err := io.WriteString(p.outW, l+"\n"); err != nil {
			return
		}
	}
}

// exit writes stderr, then ends the process with code.
func (p *wireProc) exit(code int, stderr string) {
	p.once.Do(func() {
		if stderr != "" {
			_, _ = io.WriteString(p.errW, stderr)
		}
		p.code = code
		_ = p.errW.Close()
		p.wmu.Lock()
		_ = p.outW.Close()
		p.wmu.Unlock()
		_ = p.inR.Close()
		close(p.done)
	})
}

func (p *wireProc) Stdin() io.WriteCloser { return wireStdin{p} }
func (p *wireProc) Stdout() io.Reader     { return p.outR }
func (p *wireProc) Stderr() io.Reader     { return p.errR }
func (p *wireProc) Pid() int              { return 4243 }
func (p *wireProc) Wait() (int, error)    { <-p.done; return p.code, nil }
func (p *wireProc) Terminate() error      { return p.Kill() }
func (p *wireProc) Kill() error           { p.exit(1, ""); return nil }
func (p *wireProc) Release()              {}

// wireRig opens the dashboard on the real Claude Code adapter over w.
func wireRig(t *testing.T, w *wireClaude, opt Options) *harness {
	t.Helper()
	return wireRigArgs(t, w, opt, nil)
}

// wireRigArgs is wireRig with the adapter's extra arguments (the live gate
// passes --model haiku).
func wireRigArgs(t *testing.T, w *wireClaude, opt Options, args []string) *harness {
	t.Helper()
	opt.Providers = []provider.Provider{installedClaude{&claude.Provider{Spawner: w, Args: args, Now: func() time.Time { return testNow }}}}
	if opt.Cwd == "" {
		opt.Cwd = t.TempDir()
	}
	h := newHarness(t, opt)
	h.settle()
	h.m.ui.Choice = claude.ProviderID
	h.send(key(tea.KeyEnter, 0))
	h.send(tea.WindowSizeMsg{Width: 160, Height: 60})
	return h
}

// typed is a printable key as a terminal reports it.
func typed(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }

// replay sends a probe's stdout lines from..to (1-based, inclusive) through
// p, the session's init moved to the test's project (the reducer crashes a
// session reporting another directory). The CLI's answers to the probe
// driver's own requests are left out; when an approval answer is due the
// owner gives it with F2 and a.
func replay(t *testing.T, h *harness, p *wireProc, recs []fixture.Record, from, to int) {
	t.Helper()
	for i := from - 1; i < to; i++ {
		r := recs[i]
		var m map[string]any
		_ = json.Unmarshal(r.Msg, &m)
		switch {
		case r.Dir == "in" && m["type"] == "control_response":
			h.settle()
			h.send(key(tea.KeyF2, 0))
			h.send(typed('a'))
		case r.Dir != "out" || m["type"] == "control_response":
		case m["type"] == "system" && m["subtype"] == "init":
			m["cwd"] = h.m.State().Context.Project
			b, _ := json.Marshal(m)
			p.emit(string(b))
		default:
			p.emit(string(r.Msg))
		}
	}
	h.settle()
}

func probe(t *testing.T, name string) []fixture.Record {
	t.Helper()
	recs, err := fixture.Read(filepath.Join("..", "..", "testdata", "probes", name))
	if err != nil {
		t.Fatal(err)
	}
	return recs
}

// firstUser is the probe's first message from the owner and its line.
func firstUser(t *testing.T, recs []fixture.Record) (string, int) {
	t.Helper()
	for i, r := range recs {
		var m struct {
			Type    string `json:"type"`
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		}
		if r.Dir == "in" && json.Unmarshal(r.Msg, &m) == nil && m.Type == "user" {
			return m.Message.Content, i + 1
		}
	}
	t.Fatal("no user message in the probe")
	return "", 0
}

// answersFor is each control_response written for request id.
func answersFor(w *wireClaude, id string) []map[string]any {
	var out []map[string]any
	for _, l := range w.written("control_response") {
		var m struct {
			Response struct {
				RequestID string         `json:"request_id"`
				Response  map[string]any `json:"response"`
			} `json:"response"`
		}
		if json.Unmarshal([]byte(l), &m) == nil && m.Response.RequestID == id {
			out = append(out, m.Response.Response)
		}
	}
	return out
}

func frameText(h *harness) string { return plainText(h.m.View().Content) }

// noPersistentAnswer fails when a frame offers an answer that persists a
// rule: Claude Code's are never offered (CC-4).
func noPersistentAnswer(t *testing.T, frame string) {
	t.Helper()
	low := strings.ToLower(frame)
	for _, bad := range []string{"always", "don't ask again", "s allow", "s for the session"} {
		if strings.Contains(low, bad) {
			t.Errorf("frame offers %q:\n%s", bad, frame)
		}
	}
}

// TestApprovalCards is step 4, check 1: probe approvals as cards, answered
// with one key each.
func TestApprovalCards(t *testing.T) {
	t.Run("P2 allow", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			recs := probe(t, "p2-claude-conversation.jsonl")
			const id = "5438db7f-8078-47e9-bfda-767bbf738c1a"
			w := &wireClaude{}
			h := wireRig(t, w, Options{})
			h.check = func(f string) { noPersistentAnswer(t, f) }
			text, at := firstUser(t, recs)
			h.send(tea.PasteMsg{Content: text})
			h.send(key(tea.KeyEnter, 0))
			replay(t, h, w.last(), recs, at+1, 36)

			s := h.m.State()
			if ap := s.Approvals[id]; ap == nil || ap.Status != state.Pending || s.Turn.Phase != state.WaitingApproval {
				t.Fatalf("approval %+v, turn %q; want pending, waiting", s.Approvals[id], s.Turn.Phase)
			}
			f := frameText(h)
			for _, want := range []string{"! APPROVAL · orchestrator · Write", "hello.txt", "1 │ hello", "F2 to answer", "! Waiting for approval"} {
				if !strings.Contains(f, want) {
					t.Errorf("pending card lacks %q:\n%s", want, f)
				}
			}
			h.send(typed('a')) // not focused: typed into the editor, nothing answered
			if n := len(answersFor(w, id)); n != 0 || h.m.edit.Text() != "a" {
				t.Fatalf("a without focus: %d answers, editor %q", n, h.m.edit.Text())
			}
			h.m.edit.Clear()
			h.send(key(tea.KeyF2, 0))
			if f := frameText(h); !strings.Contains(f, "▸ a allow once · d deny · c cancel the turn · Enter full view · Esc later") {
				t.Errorf("focused card lacks its keys:\n%s", f)
			}
			h.send(typed('s')) // allow for the session: never for Claude Code (CC-4; plan 5, item 8)
			if n := len(answersFor(w, id)); n != 0 || h.m.State().Approvals[id].Status != state.Pending || h.m.edit.Text() != "" {
				t.Fatalf("s on a Claude Code card: %d answers, editor %q", n, h.m.edit.Text())
			}
			h.send(typed('a'))
			h.send(typed('a'))
			h.send(typed('d'))
			got := answersFor(w, id)
			if len(got) != 1 || got[0]["behavior"] != "allow" {
				t.Fatalf("answers %v; want one allow", got)
			}
			if in, _ := json.Marshal(got[0]["updatedInput"]); !strings.Contains(string(in), "hello.txt") {
				t.Errorf("allow without the request's input: %s", in)
			}
			if ap := h.m.State().Approvals[id]; ap.Status != state.Allowed {
				t.Errorf("approval %q after allow", ap.Status)
			}
			if f := frameText(h); !strings.Contains(f, "✓ orchestrator · Write") || !strings.Contains(f, "allowed by you") {
				t.Errorf("answered card:\n%s", f)
			}
			if h.m.edit.Text() != "ad" {
				t.Errorf("keys after the answer: editor %q, want them typed", h.m.edit.Text())
			}
		})
	})

	t.Run("P3b subagent deny", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			recs := probe(t, "p3b-claude-subagent-approval.jsonl")
			const id = "42fb272e-1586-4ff9-b7b2-74a74825c280"
			w := &wireClaude{}
			h := wireRig(t, w, Options{})
			h.check = func(f string) { noPersistentAnswer(t, f) }
			text, at := firstUser(t, recs)
			h.send(tea.PasteMsg{Content: text})
			h.send(key(tea.KeyEnter, 0))
			replay(t, h, w.last(), recs, at+1, 101) // line 94's Bash is allowed on the way

			s := h.m.State()
			ap := s.Approvals[id]
			if ap == nil || ap.Status != state.Pending || ap.Agent != "a9dcbbc8d63adf34c" {
				t.Fatalf("approval %+v; want pending for agent a9dcbbc8d63adf34c", ap)
			}
			who := approvalLabel(s, ap)
			f := frameText(h)
			for _, want := range []string{"! APPROVAL · " + who + " · Write", "found.txt", "waiting: approval"} {
				if !strings.Contains(f, want) {
					t.Errorf("subagent card lacks %q:\n%s", want, f)
				}
			}
			h.send(key(tea.KeyF2, 0))
			h.send(typed('d'))
			h.send(typed('d'))
			got := answersFor(w, id)
			if len(got) != 1 || got[0]["behavior"] != "deny" || got[0]["message"] != claude.DenyMessage {
				t.Fatalf("answers %v; want one deny", got)
			}
			if f := frameText(h); !strings.Contains(f, "✗ "+who+" · Write") || !strings.Contains(f, "denied by you") {
				t.Errorf("denied card:\n%s", f)
			}
		})
	})

	t.Run("unreadable request", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			w := &wireClaude{}
			h := wireRig(t, w, Options{})
			h.send(tea.PasteMsg{Content: "go"})
			h.send(key(tea.KeyEnter, 0))
			p := w.last()
			p.emit(`{"type":"system","subtype":"init","cwd":`+quote(h.m.State().Context.Project)+`,"session_id":"s-un","model":"claude-haiku-5-5","permissionMode":"default"}`,
				`{"type":"control_request","request_id":"r-bad","request":{"subtype":"can_use_tool","tool_name":"Write","input":{"file_path":"x.txt"},"agent_id":7}}`)
			h.settle()
			got := answersFor(w, "r-bad")
			if len(got) != 1 || got[0]["behavior"] != "deny" {
				t.Fatalf("answers %v; want one deny", got)
			}
			f := frameText(h)
			if !strings.Contains(f, "denied automatically: request not readable") {
				t.Errorf("unreadable request:\n%s", f)
			}
			h.send(key(tea.KeyF2, 0))
			h.send(typed('a'))
			if n := len(answersFor(w, "r-bad")); n != 1 {
				t.Errorf("%d answers after F2 a; want the one deny", n)
			}
		})
	})

	t.Run("expired", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			w := &wireClaude{}
			h := wireRig(t, w, Options{})
			h.send(tea.PasteMsg{Content: "go"})
			h.send(key(tea.KeyEnter, 0))
			p := w.last()
			p.emit(`{"type":"system","subtype":"init","cwd":`+quote(h.m.State().Context.Project)+`,"session_id":"s-ex","model":"claude-haiku-5-5","permissionMode":"default"}`,
				`{"type":"control_request","request_id":"r-ex","request":{"subtype":"can_use_tool","tool_name":"Bash","input":{"command":"ls"}}}`)
			h.settle()
			if ap := h.m.State().Approvals["r-ex"]; ap == nil || ap.Status != state.Pending {
				t.Fatalf("approval %+v, want pending", ap)
			}
			p.emit(`{"type":"result","subtype":"success","is_error":false,"duration_ms":1,"result":"ok","session_id":"s-ex"}`)
			h.settle()
			if ap := h.m.State().Approvals["r-ex"]; ap.Status != state.Expired {
				t.Errorf("approval %q after the turn ended, want Expired", ap.Status)
			}
			if f := frameText(h); !strings.Contains(f, "expired, not answered") {
				t.Errorf("expired card:\n%s", f)
			}
			h.send(key(tea.KeyF2, 0))
			if h.m.ui.Card != "" || h.m.ui.Toast != "no approval waiting" {
				t.Errorf("F2 with none pending: card %q, toast %q", h.m.ui.Card, h.m.ui.Toast)
			}
			if n := len(w.written("control_response")); n != 0 {
				t.Errorf("%d answers written for an expired request", n)
			}
		})
	})
}

// approvalLabel is how the card names who asks: the agent's label.
func approvalLabel(s *state.State, ap *state.Approval) string {
	if ag := s.Agents[ap.Agent]; ag != nil && ag.Label != "" {
		return ag.Label
	}
	return "agent " + ap.Agent
}

func quote(s string) string { b, _ := json.Marshal(s); return string(b) }

// TestApprovalCancel is step 4, check 2: c on a card writes the deny, then
// the interrupt, in that order, and the session goes on in the same process.
func TestApprovalCancel(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		recs := probe(t, "p2-claude-conversation.jsonl")
		const id = "5438db7f-8078-47e9-bfda-767bbf738c1a"
		w := &wireClaude{}
		h := wireRig(t, w, Options{})
		text, at := firstUser(t, recs)
		h.send(tea.PasteMsg{Content: text})
		h.send(key(tea.KeyEnter, 0))
		p := w.last()
		replay(t, h, p, recs, at+1, 36)
		h.send(key(tea.KeyF2, 0))
		h.send(typed('c'))

		var order []string
		for _, l := range w.written("control_response", "control_request/interrupt") {
			switch {
			case strings.Contains(l, `"control_response"`) && strings.Contains(l, id) && strings.Contains(l, `"deny"`):
				order = append(order, "deny")
			case strings.Contains(l, `"interrupt"`):
				order = append(order, "interrupt")
			default:
				order = append(order, l)
			}
		}
		if !reflect.DeepEqual(order, []string{"deny", "interrupt"}) {
			t.Fatalf("stdin %q; want the deny, then the interrupt", order)
		}
		if s := h.m.State(); s.Turn.Phase != state.Interrupting {
			t.Errorf("turn %q after c, want Interrupting", s.Turn.Phase)
		}
		// Claude Code ends the interrupted turn.
		p.emit(`{"type":"result","subtype":"error_during_execution","is_error":true,"duration_ms":1,"session_id":"00ef0150-56f0-4033-8467-bd6e93e296bb"}`)
		h.settle()
		s := h.m.State()
		if s.Session.Phase != state.Ready || s.Turn.Phase != state.Idle || s.Approvals[id].Status != state.Denied {
			t.Fatalf("session %q, turn %q, approval %q; want Ready, Idle, Denied", s.Session.Phase, s.Turn.Phase, s.Approvals[id].Status)
		}
		h.send(tea.PasteMsg{Content: "next"})
		h.send(key(tea.KeyEnter, 0))
		if us := w.written("user"); len(us) != 2 || !strings.Contains(us[1], `"next"`) {
			t.Errorf("user lines %q; want the next message written", us)
		}
		if n := len(w.spawns()); n != 1 {
			t.Errorf("%d spawns; want the same process", n)
		}
	})
}

// wireStdin is the process's stdin; it refuses writes while refuse is set.
type wireStdin struct{ p *wireProc }

func (s wireStdin) Write(b []byte) (int, error) {
	if s.p.refuse.Load() {
		return 0, errors.New("the pipe is being closed")
	}
	return s.p.inW.Write(b)
}
func (s wireStdin) Close() error { return s.p.inW.Close() }
