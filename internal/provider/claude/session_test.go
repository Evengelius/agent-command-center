package claude

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// The fake CLI is this test binary started again with ACC_FAKE set (test 10):
// it plays one behaviour of Claude Code and never calls a model.
func TestMain(m *testing.M) {
	if mode := os.Getenv("ACC_FAKE"); mode != "" {
		fake(mode)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func fake(mode string) {
	in := bufio.NewReaderSize(os.Stdin, 1<<20)
	next := func() []byte {
		l, err := in.ReadBytes('\n')
		if err != nil && len(l) == 0 {
			os.Exit(0) // stdin closed: Claude Code exits
		}
		if log := os.Getenv("ACC_FAKE_LOG"); log != "" {
			f, _ := os.OpenFile(log, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
			f.Write(l)
			f.Close()
		}
		return l
	}
	out := func(s string) { os.Stdout.WriteString(s + "\n") }
	// Claude Code answers "/usage" itself, with no model call (P24u), and the
	// dashboard writes it after the handshake and after each turn (plan 24,
	// amendment 23): every mode answers it on the way to the line it waits
	// for, with a session window one point higher at each read.
	reads := 0
	readLine := next
	next = func() []byte {
		for {
			l := readLine()
			var msg struct {
				Type    string `json:"type"`
				Message struct {
					Content json.RawMessage `json:"content"`
				} `json:"message"`
			}
			if json.Unmarshal(l, &msg) != nil || msg.Type != "user" || string(msg.Message.Content) != `"/usage"` {
				return l
			}
			reads++
			out(usageReport(5 + reads))
			out(`{"type":"result","subtype":"success","is_error":false,"duration_ms":2,"duration_api_ms":0,"num_turns":0,"result":"Current session","session_id":"0f5e2a64-0000-4000-8000-000000000001","total_cost_usd":0,"usage":{},"modelUsage":{}}`)
		}
	}
	cwd, _ := os.Getwd()
	init := `{"type":"system","subtype":"init","cwd":` + quote(cwd) + `,"session_id":"0f5e2a64-0000-4000-8000-000000000001","model":"claude-haiku-4-5","permissionMode":"default"}`
	ready := func() {
		next() // initialize
		out(`{"type":"control_response","response":{"subtype":"success","request_id":"req_1","response":{}}}`)
	}
	switch mode {
	case "replay":
		replayFixture(next, out, cwd)
	case "ready": // answers the handshake, then reads
		ready()
	case "silent": // never answers the handshake
	case "echo": // reads one user message, answers with an assistant message holding its content (CC-10 q13)
		ready()
		var msg struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		}
		if json.Unmarshal(next(), &msg) != nil {
			os.Exit(2)
		}
		out(init)
		out(`{"type":"assistant","message":{"id":"msg_echo","type":"message","role":"assistant","model":"claude-haiku-4-5","content":[{"type":"text","text":` + quote(msg.Message.Content) + `}]},"parent_tool_use_id":null,"session_id":"0f5e2a64-0000-4000-8000-000000000001"}`)
	case "usage": // echoes one message as a whole turn, then answers the context usage; the usage reads around them are answered by next (plan 24, amendment 23)
		ready()
		var msg struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		}
		if json.Unmarshal(next(), &msg) != nil {
			os.Exit(2)
		}
		out(init)
		out(`{"type":"assistant","message":{"id":"msg_echo","type":"message","role":"assistant","model":"claude-haiku-4-5","content":[{"type":"text","text":` + quote(msg.Message.Content) + `}]},"parent_tool_use_id":null,"session_id":"0f5e2a64-0000-4000-8000-000000000001"}`)
		out(`{"type":"result","subtype":"success","is_error":false,"duration_ms":10,"num_turns":1,"result":` + quote(msg.Message.Content) + `,"session_id":"0f5e2a64-0000-4000-8000-000000000001","total_cost_usd":0.01,"usage":{"input_tokens":10,"output_tokens":5},"modelUsage":{"claude-haiku-4-5":{"inputTokens":10,"outputTokens":5,"contextWindow":200000}}}`)
		var ctl struct {
			RequestID string `json:"request_id"`
			Request   struct {
				Subtype string `json:"subtype"`
			} `json:"request"`
		}
		if json.Unmarshal(next(), &ctl) != nil || ctl.Request.Subtype != "get_context_usage" {
			fmt.Fprintf(os.Stderr, "fake: wanted get_context_usage, got %q\n", ctl.Request.Subtype)
			os.Exit(2)
		}
		out(`{"type":"control_response","response":{"subtype":"success","request_id":` + quote(ctl.RequestID) + `,"response":{"totalTokens":15,"maxTokens":200000,"autoCompactThreshold":190000}}}`)
	case "question": // reads one user message, then asks the questions of probe P14a
		ready()
		next()
		out(init)
		out(`{"type":"control_request","request_id":"q14","request":{"subtype":"can_use_tool","tool_name":"AskUserQuestion","input":{"questions":[{"question":"Which colour?","header":"Colour","options":[{"label":"Rouge café ✓","description":"Select this colour."},{"label":"Blue","description":"Select this colour."}],"multiSelect":false},{"question":"Which days?","header":"Days","options":[{"label":"Monday","description":"Select Monday."},{"label":"Tuesday","description":"Select Tuesday."}],"multiSelect":true}],"metadata":{"source":"x"}},"tool_use_id":"toolu_q14"}}`)
	case "midturn":
		ready()
		next()
		out(init)
		out(`{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hel"}},"session_id":"0f5e2a64-0000-4000-8000-000000000001","parent_tool_use_id":null}`)
		os.Exit(3)
	case "deaf":
		next()
		os.Stdin.Close() // the next write to it fails
		out(`{"type":"control_response","response":{"subtype":"success","request_id":"req_1","response":{}}}`)
		time.Sleep(30 * time.Second)
		os.Exit(0)
	case "huge":
		ready()
		w := bufio.NewWriterSize(os.Stdout, 1<<20)
		w.WriteString(`{"type":"assistant","pad":"`)
		w.WriteString(strings.Repeat("a", 65<<20))
		w.WriteString("\"}\n")
		w.Flush()
		out(init)
	case "stubborn": // ignores stdin; a grandchild holds stderr
		gc := exec.Command(os.Args[0])
		gc.Env = append(os.Environ(), "ACC_FAKE=sleep")
		gc.Stderr = os.Stderr
		if err := gc.Start(); err != nil {
			fmt.Fprintln(os.Stderr, "grandchild:", err)
			os.Exit(2)
		}
		fmt.Fprintf(os.Stderr, "pids %d %d\n", os.Getpid(), gc.Process.Pid)
		time.Sleep(60 * time.Second)
		os.Exit(0)
	case "sleep":
		time.Sleep(60 * time.Second)
		os.Exit(0)
	}
	for {
		next()
	}
}

func quote(s string) string { b, _ := json.Marshal(s); return string(b) }

// usageReport is Claude Code's answer to "/usage" as P24u recorded it, with
// the session window at pct: the page as text, and the report beside it.
func usageReport(pct int) string {
	p := strconv.Itoa(pct)
	return `{"type":"assistant","message":{"id":"msg_usage","type":"message","role":"assistant","model":"<synthetic>","content":[{"type":"text","text":"Current session: ` + p + `% used"}]},"usage_report":{"rate_limits":{"limits":[{"kind":"session","percent":` + p + `,"resets_at":"2026-10-22T15:20:00.473808+00:00","scope":null,"severity":"normal","is_active":false},{"kind":"weekly_all","percent":24,"resets_at":"2026-10-27T09:00:00.473852+00:00","scope":null,"severity":"normal","is_active":true},{"kind":"weekly_scoped","percent":11,"resets_at":"2026-10-27T09:00:00.474012+00:00","scope":{"model":{"display_name":"Fable"},"surface":null},"severity":"normal","is_active":false}],"extra_usage":{"is_enabled":false}}},"parent_tool_use_id":null,"session_id":"0f5e2a64-0000-4000-8000-000000000001"}`
}

// replayFixture plays probe P2: each recorded input waits for a line on
// stdin, each recorded output is written, with the probe's directory
// replaced by the working directory.
func replayFixture(next func() []byte, out func(string), cwd string) {
	f, err := os.Open(os.Getenv("ACC_FAKE_FILE"))
	if err != nil {
		os.Exit(2)
	}
	defer f.Close()
	dir := strings.Trim(quote(cwd), `"`)
	r := strings.NewReplacer(`<probe-root>\\probe-work\\p2`, dir, `<probe-work>`, dir)
	sc := bufio.NewScanner(f)
	sc.Buffer(nil, 16<<20)
	for sc.Scan() {
		var rec struct {
			Dir string          `json:"dir"`
			Msg json.RawMessage `json:"msg"`
		}
		if json.Unmarshal(sc.Bytes(), &rec) != nil {
			os.Exit(2)
		}
		switch rec.Dir {
		case "in":
			next()
		case "out":
			out(r.Replace(string(rec.Msg)))
		case "exit":
			return
		}
	}
}

// run is one session under test with the reducer it feeds.
type run struct {
	t    *testing.T
	sess provider.Session
	st   *state.State
	evs  []event.Event
	done bool // the events channel closed
}

func start(t *testing.T, mode string, p *Provider) (*run, string) {
	t.Helper()
	t.Setenv("ACC_FAKE", mode)
	dir := t.TempDir()
	if p == nil {
		p = &Provider{}
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	st := state.New()
	st.Context = state.Context{Provider: ProviderID, Project: dir}
	st.Startup.Phase = state.Starting
	sess, err := p.Start(context.Background(), provider.SessionContext{Executable: exe, ProjectPath: dir})
	if err != nil {
		t.Fatal(err)
	}
	r := &run{t: t, sess: sess, st: st}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = sess.Close(ctx)
		r.until(func(event.Event) bool { return false }, 15*time.Second)
	})
	return r, dir
}

// until reads and applies events until one matches or the channel closes;
// false when neither happened within d.
func (r *run) until(match func(event.Event) bool, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	for !r.done {
		select {
		case ev, ok := <-r.sess.Events():
			if !ok {
				r.done = true
				return true
			}
			if ev.Seq != uint64(len(r.evs)+1) {
				r.t.Errorf("event %s has seq %d after %d events", ev.Kind, ev.Seq, len(r.evs))
			}
			r.evs = append(r.evs, ev)
			r.st.Apply(ev)
			if match(ev) {
				return true
			}
		case <-timer.C:
			return false
		}
	}
	return true
}

func (r *run) wait(k event.Kind) event.Event {
	r.t.Helper()
	var got event.Event
	if !r.until(func(ev event.Event) bool { got = ev; return ev.Kind == k }, 10*time.Second) || got.Kind != k {
		r.t.Fatalf("no %s event (%d events, last %s)", k, len(r.evs), got.Kind)
	}
	return got
}

func (r *run) end() event.Ended {
	r.t.Helper()
	ev := r.wait(event.SessionEnded)
	if !r.until(func(event.Event) bool { return false }, 5*time.Second) {
		r.t.Fatal("the channel did not close after session.ended")
	}
	return ev.Payload.(event.Ended)
}

// TestHandshakeAndTurn replays probe P2 through the real process plumbing:
// spawn in the project, handshake, context usage, a turn with an approval
// allowed; the cwd of system/init is the project (proposal step 2, items 1
// and 4).
func TestHandshakeAndTurn(t *testing.T) {
	fixture, _ := filepath.Abs(filepath.Join("..", "..", "..", "testdata", "probes", "p2-claude-conversation.jsonl"))
	log := filepath.Join(t.TempDir(), "stdin.jsonl")
	t.Setenv("ACC_FAKE_FILE", fixture)
	t.Setenv("ACC_FAKE_LOG", log)
	r, dir := start(t, "replay", &Provider{Args: []string{"--model", "haiku"}})

	sp := r.wait(event.SessionSpawned).Payload.(event.Spawned)
	if sp.Dir != dir || !strings.Contains(strings.Join(sp.Argv, " "), "-p --input-format stream-json --output-format stream-json --verbose --include-partial-messages --forward-subagent-text --permission-prompt-tool stdio --permission-mode default --model haiku") {
		t.Errorf("spawned %q in %q", sp.Argv, sp.Dir)
	}
	r.wait(event.SessionReady)
	if r.st.Session.Phase != state.Ready || r.st.Startup.Phase != state.StartReady {
		t.Fatalf("session %q startup %q", r.st.Session.Phase, r.st.Startup.Phase)
	}
	// P2 sent both at once: the context usage answer follows the message.
	if err := r.sess.Refresh(); err != nil {
		t.Fatal(err)
	}

	text := "Create a file named hello.txt in the current directory containing the single word hello. Use the Write tool, then reply with the word done."
	r.st.Do(state.Send{Text: text})
	if err := r.sess.Send(provider.Message{Text: text}); err != nil {
		t.Fatal(err)
	}
	started := r.wait(event.SessionStarted).Payload.(event.Started)
	if started.Cwd != dir || r.st.Session.Phase != state.Ready {
		t.Errorf("cwd %q, project %q, phase %q (%s)", started.Cwd, dir, r.st.Session.Phase, r.st.Session.Reason)
	}
	req := r.wait(event.ApprovalRequest).Payload.(event.Approval)
	if err := r.sess.Answer(req.ID, "allow"); err != nil {
		t.Fatal(err)
	}
	if end := r.wait(event.TurnCompleted).Payload.(event.TurnEnd); end.Outcome != "completed" {
		t.Errorf("turn %+v", end)
	}
	if v := r.st.Metrics["context.window"]; !v.Present || v.Documented {
		t.Errorf("context window %+v", v)
	}
	if err := r.sess.Answer(req.ID, "allow"); err == nil {
		t.Error("an answered request was answered again")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	r.st.Do(state.Quit{})
	if err := r.sess.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if e := r.end(); !e.HasCode || e.Code != 0 {
		t.Errorf("ended %+v", e)
	}
	if r.st.Session.Phase != state.Closed {
		t.Errorf("phase %q (%s)", r.st.Session.Phase, r.st.Session.Reason)
	}

	// What was written, byte for byte.
	b, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	// The usage reads of plan 24 (amendment 23): after the handshake, and
	// with the context usage in Refresh.
	want := []string{
		`{"request":{"subtype":"initialize","hooks":null},"request_id":"req_1","type":"control_request"}`,
		`{"message":{"content":"/usage","role":"user"},"parent_tool_use_id":null,"session_id":"default","type":"user"}`,
		`{"request":{"subtype":"get_context_usage"},"request_id":"req_2","type":"control_request"}`,
		`{"message":{"content":"/usage","role":"user"},"parent_tool_use_id":null,"session_id":"default","type":"user"}`,
		`{"message":{"content":` + quote(text) + `,"role":"user"},"parent_tool_use_id":null,"session_id":"default","type":"user"}`,
		`{"response":{"request_id":"` + req.ID + `","response":{"behavior":"allow","updatedInput":` + string(req.Input) + `},"subtype":"success"},"type":"control_response"}`,
	}
	if len(lines) != len(want) {
		t.Fatalf("%d lines written, want %d:\n%s", len(lines), len(want), b)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Errorf("line %d:\n got %s\nwant %s", i+1, lines[i], want[i])
		}
	}
	if !bytes.Contains(req.Input, []byte(strings.Trim(quote(dir), `"`))) {
		t.Errorf("the approval's input does not name the project: %s", req.Input)
	}
}

// TestNoHandshake: a CLI that never answers initialize ends, after the
// handshake timeout, crashed with its message, and closes.
func TestNoHandshake(t *testing.T) {
	r, _ := start(t, "silent", &Provider{CloseStep: time.Second})
	r.wait(event.SessionSpawned)
	if r.until(func(ev event.Event) bool { return ev.Kind == event.SessionReady }, 300*time.Millisecond) {
		t.Fatal("ready without a handshake")
	}
	cmds := r.st.Do(state.HandshakeTimeout{})
	if len(cmds) != 1 || cmds[0].Kind != state.CmdClose {
		t.Fatalf("commands %+v", cmds)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := r.sess.Close(ctx); err != nil {
		t.Fatal(err)
	}
	r.end()
	if r.st.Session.Phase != state.Crashed || r.st.Session.Reason != "no handshake within 60 s" || r.st.Startup.Phase != state.StartFailed {
		t.Errorf("session %q %q, startup %q", r.st.Session.Phase, r.st.Session.Reason, r.st.Startup.Phase)
	}
}

// TestExitMidTurn: the CLI dies during a turn; the session is crashed, the
// turn failed, and the exit code is kept.
func TestExitMidTurn(t *testing.T) {
	r, _ := start(t, "midturn", nil)
	r.wait(event.SessionReady)
	r.st.Do(state.Send{Text: "hello"})
	if err := r.sess.Send(provider.Message{Text: "hello"}); err != nil {
		t.Fatal(err)
	}
	e := r.end()
	if !e.HasCode || e.Code != 3 || e.Reason != "Claude Code exited with code 3" {
		t.Errorf("ended %+v", e)
	}
	s := r.st
	if s.Session.Phase != state.Crashed || s.Session.Reason != "the CLI exited unexpectedly" || s.Turn.Outcome != "failed" || s.Session.ExitCode != 3 {
		t.Errorf("session %q %q, turn %q, code %d", s.Session.Phase, s.Session.Reason, s.Turn.Outcome, s.Session.ExitCode)
	}
	if err := r.sess.Send(provider.Message{Text: "again"}); err == nil {
		t.Error("a send after the exit succeeded")
	}
}

// TestSendClosedStdin: a write to a CLI that closed its input fails with a
// message, the message is not shown as sent, and the draft comes back.
func TestSendClosedStdin(t *testing.T) {
	r, _ := start(t, "deaf", &Provider{CloseStep: time.Second})
	r.wait(event.SessionReady)
	r.st.Do(state.Send{Text: "hello"})
	err := r.sess.Send(provider.Message{Text: "hello"})
	if err == nil {
		t.Fatal("the write succeeded")
	}
	cmds := r.st.Do(state.WriteFailed{Message: err.Error()})
	if len(cmds) != 1 || cmds[0].Kind != state.CmdRestoreDraft || cmds[0].Text != "hello" {
		t.Errorf("commands %+v", cmds)
	}
	if !strings.HasPrefix(r.st.Turn.Notice, "the message could not be sent: not delivered to Claude Code: ") {
		t.Errorf("notice %q", r.st.Turn.Notice)
	}
	r.until(func(event.Event) bool { return false }, 300*time.Millisecond)
	for _, ev := range r.evs {
		if ev.Kind == event.UserMessage {
			t.Error("the undelivered message became an event")
		}
	}
}

// TestLongLine: a 65 MiB line is skipped with an error giving its size, and
// the next line is read (ASM-20).
func TestLongLine(t *testing.T) {
	r, dir := start(t, "huge", nil)
	r.wait(event.SessionReady)
	ev := r.wait(event.Error)
	want := fmt.Sprintf("a line of %d bytes from Claude Code was skipped (limit 64 MiB)", 65<<20+len(`{"type":"assistant","pad":""}`)+1)
	if msg := ev.Payload.(event.Problem).Message; msg != want {
		t.Errorf("error %q, want %q", msg, want)
	}
	if st := r.wait(event.SessionStarted).Payload.(event.Started); st.Cwd != dir {
		t.Errorf("cwd %q", st.Cwd)
	}
	if r.st.Session.Phase != state.Ready || len(r.st.Errors) != 1 {
		t.Errorf("phase %q, errors %+v", r.st.Session.Phase, r.st.Errors)
	}
}

// pids reads the stubborn CLI's and its grandchild's process ids.
func pids(t *testing.T, r *run) (int, int) {
	t.Helper()
	ev := r.wait(event.Stderr)
	f := strings.Fields(ev.Payload.(event.Line).Text)
	if len(f) != 3 || f[0] != "pids" {
		t.Fatalf("stderr %q", ev.Payload.(event.Line).Text)
	}
	a, _ := strconv.Atoi(f[1])
	b, _ := strconv.Atoi(f[2])
	return a, b
}

func gone(pid int, d time.Duration) bool {
	for end := time.Now().Add(d); ; time.Sleep(20 * time.Millisecond) {
		if !alive(pid) {
			return true
		}
		if time.Now().After(end) {
			return false
		}
	}
}

// TestCloseEscalation: a CLI that ignores its closed stdin is gone, with
// what it started, within 11 s (proposal step 2, item 2).
func TestCloseEscalation(t *testing.T) {
	r, _ := start(t, "stubborn", nil)
	cli, child := pids(t, r)
	t0 := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	r.st.Do(state.Quit{})
	if err := r.sess.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if d := time.Since(t0); d > 11*time.Second || d < CloseStep {
		t.Errorf("exited after %v; want between %v and 11 s", d, CloseStep)
	}
	if !gone(cli, time.Second) || !gone(child, time.Second) {
		t.Errorf("alive: CLI %v, grandchild %v", alive(cli), alive(child))
	}
	if e := r.end(); !strings.HasPrefix(e.Reason, "Claude Code closed") {
		t.Errorf("ended %+v", e)
	}
	if len(r.st.Errors) != 0 {
		t.Errorf("errors %+v", r.st.Errors)
	}
	if r.st.Session.Phase != state.Closed {
		t.Errorf("phase %q", r.st.Session.Phase)
	}
	t.Logf("closed in %v", time.Since(t0).Round(time.Millisecond))
}

// TestForcedKill: cancelling Start's context, the application's forced
// kill, ends the whole tree at once.
func TestForcedKill(t *testing.T) {
	t.Setenv("ACC_FAKE", "stubborn")
	exe, _ := os.Executable()
	ctx, cancel := context.WithCancel(context.Background())
	sess, err := (&Provider{}).Start(ctx, provider.SessionContext{Executable: exe, ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	r := &run{t: t, sess: sess, st: state.New()}
	cli, child := pids(t, r)
	cancel()
	if !gone(cli, time.Second) || !gone(child, time.Second) {
		t.Errorf("alive after 1 s: CLI %v, grandchild %v", alive(cli), alive(child))
	}
	r.end()
}

func TestChildEnv(t *testing.T) {
	env := childEnv([]string{"PATH=x", "claude_code_emit_session_state_events=0"}, provider.Host{})
	if strings.Join(env, ";") != "PATH=x;claude_code_emit_session_state_events=0" {
		t.Errorf("an owner's variable was overridden: %q", env)
	}
	env = childEnv([]string{"PATH=x"}, provider.Host{})
	if strings.Join(env, ";") != "PATH=x;CLAUDE_CODE_EMIT_SESSION_STATE_EVENTS=1" {
		t.Errorf("env %q", env)
	}
	if a := strings.Join(args("abc", nil), " "); !strings.HasSuffix(a, "--permission-mode default --resume abc") {
		t.Errorf("args %q", a)
	}
}

// TestSetApprovalMode is plan 12, item 2, criteria 1 and 2: Manual at every
// launch (the arguments carry --permission-mode default, checked above and
// here), the capability, and the control requests written at once, byte for
// byte.
func TestSetApprovalMode(t *testing.T) {
	log := filepath.Join(t.TempDir(), "stdin.jsonl")
	t.Setenv("ACC_FAKE_LOG", log)
	r, _ := start(t, "ready", nil)
	r.wait(event.SessionReady)
	if !r.sess.Capabilities().AutoApprovals {
		t.Error("Claude Code does not offer Auto approvals")
	}
	if err := r.sess.SetApprovalMode(true); err != nil {
		t.Fatal(err)
	}
	if err := r.sess.SetApprovalMode(false); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	var lines []string
	for time.Now().Before(deadline) {
		b, _ := os.ReadFile(log)
		if lines = strings.Split(strings.TrimSuffix(string(b), "\n"), "\n"); len(lines) >= 4 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	want := []string{
		`{"request":{"subtype":"initialize","hooks":null},"request_id":"req_1","type":"control_request"}`,
		`{"message":{"content":"/usage","role":"user"},"parent_tool_use_id":null,"session_id":"default","type":"user"}`, // plan 24, amendment 23
		`{"request":{"subtype":"set_permission_mode","mode":"auto"},"request_id":"req_2","type":"control_request"}`,
		`{"request":{"subtype":"set_permission_mode","mode":"default"},"request_id":"req_3","type":"control_request"}`,
	}
	if len(lines) != len(want) {
		t.Fatalf("%d lines written, want %d: %q", len(lines), len(want), lines)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Errorf("line %d:\n got %s\nwant %s", i+1, lines[i], want[i])
		}
	}
	if a := strings.Join(args("", nil), " "); !strings.Contains(a, "--permission-mode default") || strings.Contains(a, "auto") {
		t.Errorf("a launch is not Manual: %q", a)
	}
}
