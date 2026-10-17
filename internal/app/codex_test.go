package app

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/fixture"
	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/proc"
	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/provider/codex"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// wireCodex is a codex app-server the test speaks for, on the real Codex
// adapter (plan 4, step 5): it records each spawn's argv and every stdin
// line, answers the handshake (a thread in the spawn's directory) and the
// limits; with auto it also answers each turn/start and completes the turn,
// otherwise the test plays the turn's records.
type wireCodex struct {
	mu     sync.Mutex
	argv   [][]string
	lines  []string
	procs  []*wireProc
	thread string // the thread thread/start reports
	auto   bool
}

func (w *wireCodex) Spawn(sp proc.Spec) (proc.Process, error) {
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	errR, errW := io.Pipe()
	p := &wireProc{inR: inR, inW: inW, outR: outR, outW: outW, errR: errR, errW: errW, done: make(chan struct{})}
	w.mu.Lock()
	w.argv = append(w.argv, append([]string{sp.Path}, sp.Args...))
	w.procs = append(w.procs, p)
	w.mu.Unlock()
	go func() {
		sc := bufio.NewScanner(inR)
		sc.Buffer(make([]byte, 64<<10), 64<<20)
		turns := 0
		for sc.Scan() {
			l := sc.Text()
			w.mu.Lock()
			w.lines = append(w.lines, l)
			thread, auto := w.thread, w.auto
			w.mu.Unlock()
			var m struct {
				ID     json.RawMessage `json:"id"`
				Method string          `json:"method"`
				Params struct {
					ThreadID string `json:"threadId"`
				} `json:"params"`
			}
			_ = json.Unmarshal([]byte(l), &m)
			id := string(m.ID)
			started := func(th string) {
				p.emit(`{"id":` + id + `,"result":{"thread":{"id":"` + th + `"},"model":"gpt-5.6-sol","cwd":` + quote(sp.Dir) + `,"approvalPolicy":"on-request","sandbox":{"type":"readOnly"}}}`)
			}
			switch {
			case m.Method == "initialize":
				p.emit(`{"id":` + id + `,"result":{"userAgent":"fake"}}`)
			case m.Method == "thread/start":
				started(thread)
			case m.Method == "thread/resume":
				started(m.Params.ThreadID)
			case m.Method == "account/rateLimits/read":
				p.emit(`{"id":` + id + `,"result":{"rateLimits":{"limitId":"codex","primary":{"usedPercent":1,"windowDurationMins":300,"resetsAt":1792111369}}}}`)
			case m.Method == "turn/start" && auto:
				turns++
				tu := `"tu-` + strconv.Itoa(turns) + `"`
				p.emit(`{"id":`+id+`,"result":{"turn":{"id":`+tu+`,"status":"inProgress"}}}`,
					`{"method":"turn/started","params":{"threadId":"`+m.Params.ThreadID+`","turn":{"id":`+tu+`,"status":"inProgress"}}}`,
					`{"method":"turn/completed","params":{"threadId":"`+m.Params.ThreadID+`","turn":{"id":`+tu+`,"status":"completed"}}}`)
			}
		}
		p.exit(0, "") // end of input: app-server exits
	}()
	return p, nil
}

func (w *wireCodex) last() *wireProc {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.procs[len(w.procs)-1]
}

func (w *wireCodex) spawns() [][]string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([][]string(nil), w.argv...)
}

// written is every stdin line, as written, after the first skip.
func (w *wireCodex) written(skip int) []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]string(nil), w.lines[skip:]...)
}

// method is every written request of the given method.
func (w *wireCodex) method(name string) []string {
	var out []string
	for _, l := range w.written(0) {
		var m struct {
			Method string `json:"method"`
		}
		if json.Unmarshal([]byte(l), &m) == nil && m.Method == name {
			out = append(out, l)
		}
	}
	return out
}

// lastID is the id of the latest written request of method.
func (w *wireCodex) lastID(t *testing.T, name string) string {
	t.Helper()
	ls := w.method(name)
	if len(ls) == 0 {
		t.Fatalf("no %s written", name)
	}
	var m struct {
		ID json.RawMessage `json:"id"`
	}
	_ = json.Unmarshal([]byte(ls[len(ls)-1]), &m)
	return string(m.ID)
}

// installedCodex is the real Codex adapter, found installed without looking
// for codex.exe.
type installedCodex struct{ *codex.Provider }

func (installedCodex) Detect(context.Context) provider.Detection {
	return provider.Detection{ID: codex.ProviderID, Name: codex.Name, Installed: true, Version: "0.145.0", Path: `C:\Tools\codex.exe`}
}

// codexWire opens the dashboard on the real Codex adapter over w.
func codexWire(t *testing.T, w *wireCodex, opt Options) *harness {
	t.Helper()
	opt.Providers = []provider.Provider{installedCodex{&codex.Provider{Spawner: w, Now: func() time.Time { return testNow }}}}
	if opt.Cwd == "" {
		opt.Cwd = t.TempDir()
	}
	h := newHarness(t, opt)
	h.settle()
	h.m.ui.Choice = codex.ProviderID
	h.send(key(tea.KeyEnter, 0))
	h.send(tea.WindowSizeMsg{Width: 160, Height: 60})
	if s := h.m.State(); s.Session.Phase != state.Ready || s.Session.ID != w.thread {
		t.Fatalf("Codex session %q (%q), startup %q: want Ready", s.Session.Phase, s.Session.ID, s.Startup.Phase)
	}
	return h
}

// codexRig is the input tests' rig on the real Codex adapter (proposal
// step 3, item 8; step 5, check 7): each message is a turn/start line.
func codexRig(t *testing.T) inputRig {
	w := &wireCodex{thread: "th-input", auto: true}
	h := codexWire(t, w, Options{})
	texts := func() []map[string]any {
		var out []map[string]any
		for _, l := range w.method("turn/start") {
			var m map[string]any
			if err := json.Unmarshal([]byte(l), &m); err != nil {
				t.Fatalf("turn/start line %q: %v", l, err)
			}
			out = append(out, m)
		}
		return out
	}
	return inputRig{
		h: h,
		sent: func() []string {
			var out []string
			for _, m := range texts() {
				in := m["params"].(map[string]any)["input"].([]any)
				out = append(out, in[0].(map[string]any)["text"].(string))
			}
			return out
		},
		end: func() {},
		raw: func(t *testing.T, i int, want string) {
			t.Helper()
			ls := w.method("turn/start")
			if i >= len(ls) {
				t.Fatalf("message %d not written (%d written)", i, len(ls))
			}
			var m struct {
				ID json.RawMessage `json:"id"`
			}
			_ = json.Unmarshal([]byte(ls[i]), &m)
			var b bytes.Buffer
			enc := json.NewEncoder(&b)
			enc.SetEscapeHTML(false)
			_ = enc.Encode(map[string]any{"id": json.RawMessage(m.ID), "method": "turn/start", "params": map[string]any{
				"threadId": "th-input", "input": []map[string]string{{"type": "text", "text": want}}}})
			if got := ls[i] + "\n"; got != b.String() {
				t.Errorf("stdin line\n got %q\nwant %q", got, b.String())
			}
		},
	}
}

// codexPlay sends a probe's stdout lines from..to (1-based, inclusive)
// through p. A response gets the id the adapter used for the recorded
// request's method; the probe's own inputs are the test's keys.
func codexPlay(t *testing.T, h *harness, w *wireCodex, recs []fixture.Record, from, to int) {
	t.Helper()
	methods := map[string]string{}
	for _, r := range recs {
		var m struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		if r.Dir == "in" && json.Unmarshal(r.Msg, &m) == nil && m.Method != "" {
			methods[string(m.ID)] = m.Method
		}
	}
	p := w.last()
	for i := from - 1; i < to; i++ {
		r := recs[i]
		if r.Dir != "out" {
			continue
		}
		var m map[string]json.RawMessage
		_ = json.Unmarshal(r.Msg, &m)
		if _, isReq := m["method"]; !isReq && m["id"] != nil {
			name, ok := methods[string(m["id"])]
			if !ok {
				t.Fatalf("line %d answers no recorded request", i+1)
			}
			h.settle()
			m["id"] = json.RawMessage(w.lastID(t, name))
			b, _ := json.Marshal(m)
			p.emit(string(b))
			continue
		}
		p.emit(string(r.Msg))
	}
	h.settle()
}

// firstTurn is the probe's first turn/start text and its line.
func firstTurn(t *testing.T, recs []fixture.Record, nth int) (string, int) {
	t.Helper()
	for i, r := range recs {
		var m struct {
			Method string `json:"method"`
			Params struct {
				Input []struct {
					Text string `json:"text"`
				} `json:"input"`
			} `json:"params"`
		}
		if r.Dir == "in" && json.Unmarshal(r.Msg, &m) == nil && m.Method == "turn/start" {
			if nth--; nth == 0 {
				return m.Params.Input[0].Text, i + 1
			}
		}
	}
	t.Fatal("no such turn in the probe")
	return "", 0
}

// noSessionAnswer fails when a frame offers allow for the session: no
// recorded Codex request carries availableDecisions (step 5, check 4; plan 5,
// item 8).
func noSessionAnswer(t *testing.T, frame string) {
	t.Helper()
	low := strings.ToLower(frame)
	for _, bad := range []string{"always", "for the session", "s allow", "acceptforsession"} {
		if strings.Contains(low, bad) {
			t.Errorf("frame offers %q:\n%s", bad, frame)
		}
	}
}

const (
	p6bThread = "01a14117-ac95-7332-a862-043540516dc2"
	p7Thread  = "01a1419a-23c6-74f1-837d-b5da2cbe1495"
	p7Child   = "01a1419a-4212-7712-b732-30b9c9bd385c"
)

// TestCodexApprovalCards is step 5, checks 1 and 4, in the app: probe
// approvals as cards, a, d and c writing accept, decline and cancel (then
// turn/interrupt), each once, with the request's id.
func TestCodexApprovalCards(t *testing.T) {
	recs := probe(t, "p6b-codex-app-server.jsonl")
	text, at := firstTurn(t, recs, 1)
	for _, c := range []struct {
		key  rune
		want []string // the lines written from the answer on
		ap   state.ApprovalStatus
	}{
		{'a', []string{`{"id":0,"result":{"decision":"accept"}}`}, state.Allowed},
		{'d', []string{`{"id":0,"result":{"decision":"decline"}}`}, state.Denied},
		{'c', []string{`{"id":0,"result":{"decision":"cancel"}}`, `{"id":6,"method":"turn/interrupt","params":{"threadId":"` + p6bThread + `","turnId":"01a14117-b213-7692-a2f6-654541dd8c8f"}}`}, state.Denied},
	} {
		t.Run(string(c.key), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				w := &wireCodex{thread: p6bThread}
				h := codexWire(t, w, Options{})
				h.check = func(f string) { noSessionAnswer(t, f) }
				h.send(tea.PasteMsg{Content: text})
				h.send(key(tea.KeyEnter, 0))
				codexPlay(t, h, w, recs, at+1, 37)
				s := h.m.State()
				if ap := s.Approvals["0"]; ap == nil || ap.Status != state.Pending || s.Turn.Phase != state.WaitingApproval {
					t.Fatalf("approval %+v, turn %q; want pending, waiting", s.Approvals["0"], s.Turn.Phase)
				}
				f := frameText(h)
				for _, want := range []string{"! APPROVAL orchestrator · fileChange", "hello.txt", "F2 to answer"} {
					if !strings.Contains(f, want) {
						t.Errorf("pending card lacks %q:\n%s", want, f)
					}
				}
				before := len(w.written(0))
				h.send(key(tea.KeyF2, 0))
				if f := frameText(h); !strings.Contains(f, "▸ a allow once · d deny · c cancel the turn · Enter full view · Esc later") {
					t.Errorf("focused card lacks its keys:\n%s", f)
				}
				h.send(typed(c.key))
				h.send(typed(c.key))
				h.send(typed('a'))
				h.settle()
				if got := w.written(before); !reflect.DeepEqual(got, c.want) {
					t.Errorf("written\n got %q\nwant %q", got, c.want)
				}
				if ap := h.m.State().Approvals["0"]; ap.Status != c.ap {
					t.Errorf("approval %q, want %q", ap.Status, c.ap)
				}
			})
		})
	}
}

// TestCodexChildApproval is step 5, check 1: P7's approval, requested by a
// child thread, is labelled with that thread's agent and answered with id 0.
func TestCodexChildApproval(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		recs := probe(t, "p7-codex-subagents.jsonl")
		text, at := firstTurn(t, recs, 1)
		w := &wireCodex{thread: p7Thread}
		h := codexWire(t, w, Options{})
		h.send(tea.PasteMsg{Content: text})
		h.send(key(tea.KeyEnter, 0))
		codexPlay(t, h, w, recs, at+1, 89)
		s := h.m.State()
		ap := s.Approvals["0"]
		if ap == nil || ap.Agent != p7Child || ap.Status != state.Pending {
			t.Fatalf("approval %+v; want pending, from %s", ap, p7Child)
		}
		label := s.Agents[p7Child].Label
		if f := frameText(h); label == "" || !strings.Contains(f, "! APPROVAL "+label) {
			t.Errorf("card not labelled %q:\n%s", label, f)
		}
		before := len(w.written(0))
		h.send(key(tea.KeyF2, 0))
		h.send(typed('a'))
		h.settle()
		if got := w.written(before); !reflect.DeepEqual(got, []string{`{"id":0,"result":{"decision":"accept"}}`}) {
			t.Errorf("written %q", got)
		}
	})
}

// TestCodexInterrupt is step 5, check 1: P6b's second turn, interrupted
// with Ctrl+C, writes turn/interrupt with the thread and the running turn, and
// ends interrupted with the session Ready.
func TestCodexInterrupt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		recs := probe(t, "p6b-codex-app-server.jsonl")
		w := &wireCodex{thread: p6bThread}
		h := codexWire(t, w, Options{})
		text, at := firstTurn(t, recs, 2)
		h.send(tea.PasteMsg{Content: text})
		h.send(key(tea.KeyEnter, 0))
		codexPlay(t, h, w, recs, at+1, 79)
		if s := h.m.State(); s.Turn.Phase != state.Running {
			t.Fatalf("turn %q, want running", s.Turn.Phase)
		}
		before := len(w.written(0))
		h.send(key('c', tea.ModCtrl))
		h.settle()
		want := []string{`{"id":` + w.lastID(t, "turn/interrupt") + `,"method":"turn/interrupt","params":{"threadId":"` + p6bThread + `","turnId":"01a14118-2a3c-79b2-b1f9-7b38b04c28cb"}}`}
		if got := w.written(before); !reflect.DeepEqual(got, want) {
			t.Errorf("written\n got %q\nwant %q", got, want)
		}
		codexPlay(t, h, w, recs, 81, 91)
		if s := h.m.State(); s.Turn.Outcome != "interrupted" || s.Session.Phase != state.Ready || len(s.Errors) != 0 {
			t.Errorf("turn %q, session %q, errors %+v", s.Turn.Outcome, s.Session.Phase, s.Errors)
		}
	})
}

// TestCodexCrashAndResume is step 5, check 5, in the app: after a crash,
// Ctrl+R spawns codex app-server again and resumes the crashed thread, the
// draft kept.
func TestCodexCrashAndResume(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := input.Drafts{Dir: t.TempDir()}
		w := &wireCodex{thread: "th-crash", auto: true}
		h := codexWire(t, w, Options{Drafts: &d})
		h.send(tea.PasteMsg{Content: "first"})
		h.send(key(tea.KeyEnter, 0))
		h.settle()
		h.send(tea.PasteMsg{Content: draftText})
		w.last().exit(3, "Error: something broke\n")
		h.settle()
		s := h.m.State()
		if s.Session.Phase != state.Crashed || s.Session.ExitCode != 3 {
			t.Fatalf("session %q, exit code %d; want Crashed, 3", s.Session.Phase, s.Session.ExitCode)
		}
		if f := frameText(h); !strings.Contains(f, "Ctrl+R resume") {
			t.Errorf("crash card:\n%s", f)
		}
		skip := len(w.written(0))
		h.send(key('r', tea.ModCtrl))
		argv := w.spawns()
		if len(argv) != 2 || !reflect.DeepEqual(argv[0], argv[1]) || strings.Join(argv[1][1:], " ") != "app-server" {
			t.Fatalf("spawns %q; want app-server twice", argv)
		}
		got := w.written(skip)
		if len(got) < 4 || got[3] != `{"id":3,"method":"thread/resume","params":{"threadId":"th-crash","sandbox":"read-only","approvalPolicy":"on-request"}}` {
			t.Errorf("after Ctrl+R:\n%s", strings.Join(got, "\n"))
		}
		if n := len(w.method("thread/start")); n != 1 {
			t.Errorf("%d thread/start written; want the first only", n)
		}
		s = h.m.State()
		if s.Session.Phase != state.Ready || s.Session.ID != "th-crash" || h.m.edit.Text() != draftText {
			t.Errorf("after Ctrl+R: session %q (%q), draft %q", s.Session.Phase, s.Session.ID, h.m.edit.Text())
		}
	})
}

// L5: "/status" typed for Codex is held back once with the dashboard's note,
// the draft kept and no YOU entry; Enter again sends it as a message.
func TestCodexSlashCommandNote(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := &wireCodex{thread: "th-slash", auto: true}
		h := codexWire(t, w, Options{})
		h.send(tea.PasteMsg{Content: "/status"})
		h.send(key(tea.KeyEnter, 0))
		if n := len(w.method("turn/start")); n != 0 {
			t.Fatalf("%d turn/start written, want none", n)
		}
		if got := h.m.edit.Text(); got != "/status" {
			t.Errorf("editor %q, want the draft kept", got)
		}
		f := frameText(h)
		for _, want := range []string{"· /status is a command of Codex's own terminal", "    Press Enter again to send it to the model as a message."} {
			if !strings.Contains(f, want) {
				t.Errorf("frame lacks %q:\n%s", want, f)
			}
		}
		if strings.Contains(f, "YOU") {
			t.Errorf("a YOU entry for a message not sent:\n%s", f)
		}
		h.send(key(tea.KeyEnter, 0))
		if ls := w.method("turn/start"); len(ls) != 1 || !strings.Contains(ls[0], `"text":"/status"`) {
			t.Fatalf("turn/start %q, want /status sent once", ls)
		}
		if got := h.m.edit.Text(); got != "" {
			t.Errorf("editor %q after sending, want empty", got)
		}
	})
}
