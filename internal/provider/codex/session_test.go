package codex

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// The fake app-server is this test binary started again with ACC_FAKE set
// (test 10): it plays recorded app-server sessions and never calls a model.
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
			os.Exit(0) // stdin closed: app-server exits
		}
		if log := os.Getenv("ACC_FAKE_LOG"); log != "" {
			f, _ := os.OpenFile(log, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
			f.Write(l)
			f.Close()
		}
		return l
	}
	switch mode {
	case "replay":
		play(next)
	case "silent": // never answers initialize
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

// segment is one recorded input and the records that follow it, up to the
// next input.
type segment struct {
	file    int
	method  string // "" for an answer to one of the CLI's requests
	thread  string // the threadId the input names, if any
	probeID string // the input's id in the recording
	recs    []record
	used    bool
}

type record struct {
	Dir string          `json:"dir"`
	Msg json.RawMessage `json:"msg"`
}

var probeDir = regexp.MustCompile(`<probe-root>(?:\\\\|/)probe-work(?:\\\\|/)p[0-9a-z]+`)

// play plays the recordings ACC_FAKE_FILE names (separated by ";") as a
// conversation, not a fixed order. Each line read is matched to the first
// unused recorded input of the same method naming the same thread (an
// answer, to the next recorded answer), and the records after that input
// are played: outputs with the recording's ids rewritten to the ones
// received and the probe's directory replaced by the working directory,
// stderr lines, and the test-only records "die" (exit with a code) and
// "huge" (a 65 MiB line). The recordings' own "exit" records are not
// played: the fake exits when its stdin closes. model/list is answered
// from ACC_FAKE_MODELS.
func play(next func() []byte) {
	cwd, _ := os.Getwd()
	dir := strings.Trim(quote(cwd), `"`)
	var segs []*segment
	for i, path := range strings.Split(os.Getenv("ACC_FAKE_FILE"), ";") {
		b, err := os.ReadFile(path)
		if err != nil {
			os.Exit(2)
		}
		var cur *segment
		for _, l := range strings.Split(strings.TrimSpace(string(b)), "\n") {
			var rec record
			if json.Unmarshal([]byte(l), &rec) != nil {
				os.Exit(2)
			}
			if rec.Dir == "in" {
				var m struct {
					ID     json.RawMessage `json:"id"`
					Method string          `json:"method"`
					Params struct {
						ThreadID string `json:"threadId"`
					} `json:"params"`
				}
				_ = json.Unmarshal(rec.Msg, &m)
				cur = &segment{file: i, method: m.Method, thread: m.Params.ThreadID, probeID: string(m.ID)}
				segs = append(segs, cur)
			} else if cur != nil {
				cur.recs = append(cur.recs, rec)
			}
		}
	}
	ids := map[int]map[string]json.RawMessage{} // file -> recorded id -> id received
	for {
		var r struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Params struct {
				ThreadID string `json:"threadId"`
			} `json:"params"`
		}
		if json.Unmarshal(next(), &r) != nil {
			continue
		}
		recorded := false // a script's own model/list answers first (plan 34)
		for _, s := range segs {
			recorded = recorded || !s.used && s.method == "model/list"
		}
		if r.Method == "model/list" && !recorded && os.Getenv("ACC_FAKE_MODELS") != "" {
			b, _ := os.ReadFile(os.Getenv("ACC_FAKE_MODELS"))
			var m map[string]json.RawMessage
			_ = json.Unmarshal(b, &m)
			m["id"] = r.ID
			out, _ := json.Marshal(m)
			os.Stdout.Write(append(out, '\n'))
			continue
		}
		var seg *segment
		for _, s := range segs {
			if !s.used && s.method == r.Method && (s.thread == "" || s.thread == r.Params.ThreadID) {
				seg = s
				break
			}
		}
		if seg == nil {
			continue
		}
		seg.used = true
		if r.Method != "" && seg.probeID != "" && len(r.ID) > 0 {
			if ids[seg.file] == nil {
				ids[seg.file] = map[string]json.RawMessage{}
			}
			ids[seg.file][seg.probeID] = r.ID
		}
		for _, rec := range seg.recs {
			switch rec.Dir {
			case "out":
				os.Stdout.WriteString(rewrite(rec.Msg, ids[seg.file], dir) + "\n")
			case "stderr":
				var s string
				_ = json.Unmarshal(rec.Msg, &s)
				os.Stderr.WriteString(s + "\n")
			case "die":
				code, _ := strconv.Atoi(string(rec.Msg))
				os.Exit(code)
			case "huge":
				w := bufio.NewWriterSize(os.Stdout, 1<<20)
				w.WriteString(`{"method":"pad","params":"`)
				w.WriteString(strings.Repeat("a", 65<<20))
				w.WriteString("\"}\n")
				w.Flush()
			}
		}
	}
}

// rewrite replaces the probe's directory and, in a response, the recorded id.
func rewrite(msg json.RawMessage, ids map[string]json.RawMessage, dir string) string {
	s := probeDir.ReplaceAllLiteralString(string(msg), dir)
	s = strings.ReplaceAll(s, "<probe-work>", dir)
	var m map[string]json.RawMessage
	if json.Unmarshal([]byte(s), &m) != nil {
		return s
	}
	if _, isReq := m["method"]; isReq {
		return s
	}
	if id, ok := ids[string(m["id"])]; ok {
		m["id"] = id
		b, _ := json.Marshal(m)
		return string(b)
	}
	return s
}

func probe(name string) string {
	p, _ := filepath.Abs(filepath.Join("..", "..", "..", "testdata", "probes", name))
	return p
}

// script writes a test-only recording.
func script(t *testing.T, recs ...string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "script.jsonl")
	if err := os.WriteFile(p, []byte(strings.Join(recs, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// handshake is a minimal recorded start: initialize, a thread, the limits.
var handshake = []string{
	`{"dir":"in","msg":{"id":1,"method":"initialize"}}`,
	`{"dir":"out","msg":{"id":1,"result":{"userAgent":"fake"}}}`,
	`{"dir":"in","msg":{"method":"initialized"}}`,
	`{"dir":"in","msg":{"id":2,"method":"thread/start"}}`,
	`{"dir":"out","msg":{"id":2,"result":{"thread":{"id":"th-1"},"model":"gpt-5.6-sol","cwd":"<probe-work>","sandbox":{"type":"readOnly"},"approvalPolicy":"on-request"}}}`,
	`{"dir":"in","msg":{"id":3,"method":"account/rateLimits/read"}}`,
	`{"dir":"out","msg":{"id":3,"result":{"rateLimits":{"limitId":"codex","primary":{"usedPercent":0,"windowDurationMins":300,"resetsAt":1792111369}}}}}`,
}

// run is one session under test with the reducer it feeds.
type run struct {
	t    *testing.T
	sess provider.Session
	st   *state.State
	evs  []event.Event
	done bool // the events channel closed
	log  string
	dir  string // the project directory
}

// start runs the fake in mode, replaying files, with a stdin log.
func start(t *testing.T, mode string, p *Provider, resume string, files ...string) (*run, string) {
	t.Helper()
	t.Setenv("ACC_FAKE", mode)
	t.Setenv("ACC_FAKE_FILE", strings.Join(files, ";"))
	t.Setenv("ACC_FAKE_MODELS", probe("p6-codex-model-list.json"))
	log := filepath.Join(t.TempDir(), "stdin.jsonl")
	t.Setenv("ACC_FAKE_LOG", log)
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
	sess, err := p.Start(context.Background(), provider.SessionContext{Executable: exe, ProjectPath: dir, ResumeID: resume})
	if err != nil {
		t.Fatal(err)
	}
	r := &run{t: t, sess: sess, st: st, log: log, dir: dir}
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

func (r *run) waitFor(what string, match func(event.Event) bool) event.Event {
	r.t.Helper()
	var got event.Event
	if !r.until(func(ev event.Event) bool { got = ev; return match(ev) }, 10*time.Second) || !match(got) {
		r.t.Fatalf("no %s (%d events, last %s %s)", what, len(r.evs), got.Kind, got.Raw)
	}
	return got
}

func (r *run) wait(k event.Kind) event.Event {
	r.t.Helper()
	return r.waitFor(string(k), func(ev event.Event) bool { return ev.Kind == k })
}

// limits waits for the answer to account/rateLimits/read.
func (r *run) limits() {
	r.t.Helper()
	r.waitFor("rate limits answer", func(ev event.Event) bool {
		return ev.Kind == event.Telemetry && strings.HasPrefix(ev.Raw, `{"id":`) && !strings.Contains(ev.Raw, "config/read")
	})
}

func (r *run) end() event.Ended {
	r.t.Helper()
	ev := r.wait(event.SessionEnded)
	if !r.until(func(event.Event) bool { return false }, 5*time.Second) {
		r.t.Fatal("the channel did not close after session.ended")
	}
	return ev.Payload.(event.Ended)
}

func (r *run) close() event.Ended {
	r.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	r.st.Do(state.Quit{})
	if err := r.sess.Close(ctx); err != nil {
		r.t.Fatal(err)
	}
	return r.end()
}

// send is the owner's message as the application sends it.
func (r *run) send(text string) {
	r.t.Helper()
	r.st.Do(state.Send{Text: text})
	if err := r.sess.Send(provider.Message{Text: text}); err != nil {
		r.t.Fatal(err)
	}
}

// written is what the session wrote, each line checked against the requests
// the adapter must never send.
func (r *run) written() []string {
	r.t.Helper()
	b, err := os.ReadFile(r.log)
	if err != nil {
		r.t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	for _, l := range lines {
		if why := forbidden(l); why != "" {
			r.t.Errorf("forbidden write (%s): %s", why, l)
		}
	}
	return lines
}

func (r *run) expect(want []string) {
	r.t.Helper()
	lines := r.written()
	for i := range want {
		want[i] = strings.ReplaceAll(want[i], "<dir>", strings.Trim(quote(r.dir), `"`))
	}
	if len(lines) != len(want) {
		r.t.Fatalf("%d lines written, want %d:\n%s", len(lines), len(want), strings.Join(lines, "\n"))
	}
	for i := range want {
		if lines[i] != want[i] {
			r.t.Errorf("line %d:\n got %s\nwant %s", i+1, lines[i], want[i])
		}
	}
}

// forbiddenMethods are never sent (architecture.md, Codex adapter, "Never
// called"; AGENTS.md: never change a CLI's configuration).
var forbiddenMethods = []string{"config/value", "config/batch", "config/mcp", "skills/config/write", "marketplace/", "plugin/", "externalAgentConfig/", "windowsSandbox/"}

// forbidden says why a written line is one the adapter must never send, ""
// when it is allowed.
func forbidden(line string) string {
	var m struct {
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal([]byte(line), &m); err != nil {
		return "unreadable: " + err.Error()
	}
	for _, f := range forbiddenMethods {
		if strings.HasPrefix(m.Method, f) {
			return "method " + m.Method
		}
	}
	if m.Method == "config/read" {
		// The one config call: the limit only, in the project directory, no layers.
		var p struct {
			Cwd           *string `json:"cwd"`
			IncludeLayers *bool   `json:"includeLayers"`
		}
		if json.Unmarshal(m.Params, &p) != nil || p.Cwd == nil || p.IncludeLayers == nil || *p.IncludeLayers {
			return "config/read with other parameters"
		}
		return ""
	}
	var walk func(v any) string
	walk = func(v any) string {
		switch v := v.(type) {
		case map[string]any:
			for k, x := range v {
				switch {
				case k == "cwd":
					return "cwd sent"
				case strings.Contains(strings.ToLower(k), "execpolicy"), strings.Contains(strings.ToLower(k), "networkpolicy"):
					return k + " sent"
				}
				if why := walk(x); why != "" {
					return why
				}
			}
		case []any:
			for _, x := range v {
				if why := walk(x); why != "" {
					return why
				}
			}
		case string:
			if v == "workspace-write" || v == "danger-full-access" || v == "workspaceWrite" || v == "dangerFullAccess" {
				return "sandbox " + v
			}
		}
		return ""
	}
	var p any
	_ = json.Unmarshal(m.Params, &p)
	if why := turnSandboxForbidden(m.Method, p); why != "" {
		return why
	}
	if why := walk(p); why != "" {
		return why
	}
	if len(m.Result) > 0 {
		var r map[string]json.RawMessage
		if json.Unmarshal(m.Result, &r) == nil && len(r) == 1 && r["currentTimeAt"] != nil {
			return "" // currentTime/read's reply, the one answer that is no decision (plan 38)
		}
		var d struct {
			Decision any `json:"decision"`
		}
		_ = json.Unmarshal(m.Result, &d)
		if s, ok := d.Decision.(string); !ok || !wireWord(s) {
			return fmt.Sprintf("decision %v", d.Decision)
		}
	}
	return ""
}

// wireWord says w is one of the words Codex takes as a decision, as the
// adapter writes them (plan 49, design D10).
func wireWord(w string) bool {
	for _, v := range answers {
		if v == w {
			return true
		}
	}
	return false
}

// turnSandboxForbidden checks the sandboxPolicy of turn/start and
// thread/settings/update and takes it out of params when allowed: one of the
// modes selector's three presets exactly, its sandbox (the type alone), its
// approval policy and its reviewer (plan 39, design D1; plan 12 addendum):
// readOnly with on-request and the owner reviewing, workspaceWrite with
// on-request and Codex's auto_review, dangerFullAccess with never and the
// owner; nothing else, and on no other method.
func turnSandboxForbidden(method string, params any) string {
	p, ok := params.(map[string]any)
	if !ok || p["sandboxPolicy"] == nil {
		return ""
	}
	if method != "turn/start" && method != "thread/settings/update" {
		return "sandboxPolicy on " + method
	}
	sp, ok := p["sandboxPolicy"].(map[string]any)
	if !ok || len(sp) != 1 {
		return fmt.Sprintf("sandboxPolicy %v", p["sandboxPolicy"])
	}
	presets := map[any][2]string{"readOnly": {"on-request", "user"}, "workspaceWrite": {"on-request", "auto_review"}, "dangerFullAccess": {"never", "user"}}
	want, ok := presets[sp["type"]]
	if !ok {
		return fmt.Sprintf("sandbox %v", sp["type"])
	}
	if p["approvalPolicy"] != want[0] || p["approvalsReviewer"] != want[1] {
		return fmt.Sprintf("sandbox %v with policy %v and reviewer %v", sp["type"], p["approvalPolicy"], p["approvalsReviewer"])
	}
	delete(p, "sandboxPolicy")
	return ""
}

// TestForbiddenCheck: the check catches what it is there for, so a clean
// log means something (a planted cwd fails it).
func TestForbiddenCheck(t *testing.T) {
	for _, l := range []string{
		`{"id":2,"method":"thread/start","params":{"cwd":"C:\\x","sandbox":"read-only","approvalPolicy":"on-request"}}`,
		`{"id":2,"method":"thread/start","params":{"sandbox":"workspace-write"}}`,
		`{"id":2,"method":"config/value/write","params":{}}`,
		`{"id":2,"method":"skills/config/write","params":{}}`,
		`{"id":2,"method":"windowsSandbox/setupStart"}`,
		`{"id":0,"result":{"decision":{"acceptWithExecpolicyAmendment":{"execpolicy_amendment":["ls"]}}}}`,
		`{"id":0,"result":{"decision":"allow"}}`,
		`{"id":5,"method":"turn/start","params":{"approvalsReviewer":"user","sandboxPolicy":{"type":"workspaceWrite"},"threadId":"t"}}`,
		`{"id":5,"method":"turn/start","params":{"sandboxPolicy":{"type":"workspaceWrite"},"threadId":"t"}}`,
		`{"id":5,"method":"turn/start","params":{"approvalsReviewer":"auto_review","sandboxPolicy":{"type":"workspaceWrite","networkAccess":true},"threadId":"t"}}`,
		`{"id":5,"method":"turn/start","params":{"approvalsReviewer":"auto_review","sandboxPolicy":{"type":"workspaceWrite","writableRoots":["C:\\"]},"threadId":"t"}}`,
		`{"id":5,"method":"turn/start","params":{"approvalsReviewer":"auto_review","sandboxPolicy":{"type":"dangerFullAccess"},"threadId":"t"}}`,
		`{"id":2,"method":"thread/start","params":{"sandboxPolicy":{"type":"readOnly"}}}`,
		// Plan 39: a mode's settings other than one of the three presets.
		`{"id":5,"method":"turn/start","params":{"approvalPolicy":"on-request","approvalsReviewer":"user","sandboxPolicy":{"type":"dangerFullAccess"},"threadId":"t"}}`,
		`{"id":5,"method":"turn/start","params":{"approvalPolicy":"never","approvalsReviewer":"user","sandboxPolicy":{"type":"workspaceWrite"},"threadId":"t"}}`,
		`{"id":5,"method":"turn/start","params":{"approvalsReviewer":"user","sandboxPolicy":{"type":"readOnly"},"threadId":"t"}}`,
		`{"id":7,"method":"thread/settings/update","params":{"threadId":"t","sandboxPolicy":{"type":"workspaceWrite"},"approvalPolicy":"on-request","approvalsReviewer":"user"}}`,
		`{"id":7,"method":"thread/settings/update","params":{"threadId":"t","sandboxPolicy":{"type":"dangerFullAccess"},"approvalPolicy":"never","approvalsReviewer":"auto_review"}}`,
		`{"id":7,"method":"thread/settings/update","params":{"threadId":"t","sandboxPolicy":{"type":"dangerFullAccess","networkAccess":true},"approvalPolicy":"never","approvalsReviewer":"user"}}`,
		`{"id":7,"method":"thread/settings/update","params":{"threadId":"t","sandboxPolicy":{"type":"externalSandbox"},"approvalPolicy":"never","approvalsReviewer":"user"}}`,
		`{"id":7,"method":"thread/fork","params":{"threadId":"t","sandboxPolicy":{"type":"readOnly"},"approvalPolicy":"on-request","approvalsReviewer":"user"}}`,
	} {
		if forbidden(l) == "" {
			t.Errorf("not caught: %s", l)
		}
	}
	for _, l := range []string{
		`{"id":2,"method":"thread/start","params":{"sandbox":"read-only","approvalPolicy":"on-request","approvalsReviewer":"user"}}`,
		`{"id":5,"method":"turn/start","params":{"approvalPolicy":"on-request","approvalsReviewer":"user","sandboxPolicy":{"type":"readOnly"},"threadId":"t"}}`,
		`{"id":5,"method":"turn/start","params":{"approvalPolicy":"on-request","approvalsReviewer":"auto_review","sandboxPolicy":{"type":"workspaceWrite"},"threadId":"t"}}`,
		`{"id":5,"method":"turn/start","params":{"approvalPolicy":"never","approvalsReviewer":"user","sandboxPolicy":{"type":"dangerFullAccess"},"threadId":"t"}}`,
		`{"id":7,"method":"thread/settings/update","params":{"threadId":"t","sandboxPolicy":{"type":"readOnly"},"approvalPolicy":"on-request","approvalsReviewer":"user"}}`,
		`{"id":7,"method":"thread/settings/update","params":{"threadId":"t","sandboxPolicy":{"type":"workspaceWrite"},"approvalPolicy":"on-request","approvalsReviewer":"auto_review"}}`,
		`{"id":7,"method":"thread/settings/update","params":{"threadId":"t","sandboxPolicy":{"type":"dangerFullAccess"},"approvalPolicy":"never","approvalsReviewer":"user"}}`,
		`{"id":7,"method":"thread/settings/update","params":{"threadId":"t","effort":"low"}}`,
		`{"id":0,"result":{"decision":"cancel"}}`,
		`{"error":{"code":-32601,"message":"x"},"id":9}`,
	} {
		if why := forbidden(l); why != "" {
			t.Errorf("caught (%s): %s", why, l)
		}
	}
}

// TestNoForbiddenMethodInSource: no method string in the adapter is one it
// must never send.
func TestNoForbiddenMethodInSource(t *testing.T) {
	for _, f := range []string{"session.go", "provider.go"} {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range regexp.MustCompile(`"([A-Za-z]+/[A-Za-z/]+)"`).FindAllStringSubmatch(string(b), -1) {
			for _, m := range forbiddenMethods {
				if strings.HasPrefix(s[1], m) {
					t.Errorf("%s names %s", f, s[1])
				}
			}
		}
		if strings.Count(string(b), `"cwd"`) > 1 || strings.Contains(string(b), "workspace-write") || strings.Contains(string(b), "danger-full-access") {
			t.Errorf("%s names cwd or a writable sandbox", f)
		}
		// The writable sandboxes: Approve for me's and Full access's presets,
		// once each (plan 39, design D1; plan 12 addendum).
		for word, preset := range map[string]*regexp.Regexp{
			`"workspaceWrite"`:   regexp.MustCompile(`event\.ModeAuto:\s+\{"workspaceWrite", "on-request", reviewerAuto\}`),
			`"dangerFullAccess"`: regexp.MustCompile(`event\.ModeFull:\s+\{"dangerFullAccess", "never", reviewerUser\}`),
		} {
			if n := strings.Count(string(b), word); n > 1 || n == 1 && !preset.Match(b) {
				t.Errorf("%s names %s other than in its mode's preset", f, word)
			}
		}
	}
}

const (
	p6Thread  = "01a14116-b6ce-7441-8ac1-108f96c85169"
	p6bThread = "01a14117-ac95-7332-a862-043540516dc2"
	p6bTurn2  = "01a14118-2a3c-79b2-b1f9-7b38b04c28cb"
	p7Thread  = "01a1419a-23c6-74f1-837d-b5da2cbe1495"
	p7Child   = "01a1419a-4212-7712-b732-30b9c9bd385c"
)

// initLine is the session's initialize, with Codex's experimental API asked
// for (plan 38, design D1). The keys come sorted, as the encoder writes a map.
const initLine = `{"id":1,"method":"initialize","params":{"capabilities":{"experimentalApi":true},"clientInfo":{"name":"agent_command_center","title":"Agent Command Center","version":"dev"}}}`

// turnLine is the turn/start line the session writes for text: Ask for
// approval's policy, reviewer and sandbox (plan 39), and the reasoning
// summary every turn asks for (plan 22, amendment 20). The keys come
// sorted, as the encoder writes a map.
func turnLine(id int, thread, text string) string {
	return `{"id":` + strconv.Itoa(id) + `,"method":"turn/start","params":{"approvalPolicy":"on-request","approvalsReviewer":"user","input":[{"text":` + quote(text) + `,"type":"text"}],"sandboxPolicy":{"type":"readOnly"},"summary":"auto","threadId":"` + thread + `"}}`
}

// TestHandshakeApprovalInterrupt replays P6b through the real process
// plumbing: app-server spawned in the project, the handshake without cwd, a
// file change approved with accept, then a turn interrupted (proposal step
// 5, items 1 and 2).
func TestHandshakeApprovalInterrupt(t *testing.T) {
	r, dir := start(t, "replay", nil, "", probe("p6b-codex-app-server.jsonl"))
	exe, _ := os.Executable()
	sp := r.wait(event.SessionSpawned).Payload.(event.Spawned)
	if sp.Dir != dir || strings.Join(sp.Argv, " ") != exe+" app-server" {
		t.Errorf("spawned %q in %q", sp.Argv, sp.Dir)
	}
	st := r.wait(event.SessionStarted).Payload.(event.Started)
	if st.SessionID != p6bThread || st.Cwd != dir || st.Model != "gpt-5.6-sol" {
		t.Errorf("started %+v", st)
	}
	if r.st.Session.Phase != state.Ready || r.st.Startup.Phase != state.StartReady {
		t.Fatalf("session %q startup %q", r.st.Session.Phase, r.st.Session.Reason)
	}
	r.limits()

	r.send("make hello.txt")
	ev := r.wait(event.ApprovalRequest)
	req := ev.Payload.(event.Approval)
	if req.ID != "0" || req.Tool != "fileChange" || ev.Agent != "" || r.st.Turn.Phase != state.WaitingApproval {
		t.Errorf("request %+v agent %q, turn %q", req, ev.Agent, r.st.Turn.Phase)
	}
	// Codex's own words, and the CLI's withdrawal, are no owner's answer:
	// the adapter takes the dashboard's (plan 49, design D10).
	for _, d := range []event.Decision{"accept", "approve", event.Withdrawn} {
		if err := r.sess.Answer(req.ID, d); err == nil {
			t.Errorf("answer %q accepted", d)
		}
	}
	if err := r.sess.Answer("1", event.Allow); !errors.Is(err, provider.ErrNoPending) || err.Error() != `no pending request "1"` {
		t.Errorf("an unknown request answered: %v, want ErrNoPending (plan 47)", err)
	}
	if err := r.sess.Answer(req.ID, event.Allow); err != nil {
		t.Fatal(err)
	}
	if end := r.wait(event.TurnCompleted).Payload.(event.TurnEnd); end.Outcome != "completed" {
		t.Errorf("turn %+v", end)
	}
	if err := r.sess.Answer(req.ID, event.Allow); !errors.Is(err, provider.ErrNoPending) {
		t.Errorf("an answered request answered again: %v, want ErrNoPending (plan 47)", err)
	}
	if a := r.st.Approvals["0"]; a == nil || a.Status != state.Allowed || a.Answer != event.Allow {
		t.Errorf("approval %+v", a)
	}

	r.send("count")
	r.waitFor("turn accepted", func(ev event.Event) bool { return ev.Turn == p6bTurn2 })
	r.st.Do(state.Interrupt{})
	if err := r.sess.Interrupt(); err != nil {
		t.Fatal(err)
	}
	r.wait(event.TurnCompleted)
	if r.st.Turn.Outcome != "interrupted" || r.st.Session.Phase != state.Ready {
		t.Errorf("turn %q, session %q (%s)", r.st.Turn.Outcome, r.st.Session.Phase, r.st.Session.Reason)
	}
	if err := r.sess.Interrupt(); err == nil {
		t.Error("an interrupt with no running turn was sent")
	}
	if e := r.close(); !e.HasCode || e.Code != 0 || r.st.Session.Phase != state.Closed {
		t.Errorf("ended %+v, phase %q", e, r.st.Session.Phase)
	}
	if len(r.st.Errors) != 0 {
		t.Errorf("errors %+v", r.st.Errors)
	}
	r.expect([]string{
		initLine,
		`{"method":"initialized"}`,
		`{"id":2,"method":"config/read","params":{"cwd":"<dir>","includeLayers":false}}`,
		`{"id":3,"method":"configRequirements/read"}`,
		`{"id":4,"method":"model/list","params":{}}`,
		`{"id":5,"method":"thread/start","params":{"sandbox":"read-only","approvalPolicy":"on-request","approvalsReviewer":"user"}}`,
		`{"id":6,"method":"account/rateLimits/read"}`,
		turnLine(7, p6bThread, "make hello.txt"),
		`{"id":0,"result":{"decision":"accept"}}`,
		turnLine(8, p6bThread, "count"),
		`{"id":9,"method":"turn/interrupt","params":{"threadId":"` + p6bThread + `","turnId":"` + p6bTurn2 + `"}}`,
	})
}

// TestModelFallback replays P6, whose configured model the account refuses,
// then P6b: the refusal is shown as received, model/list gives the default,
// a new thread starts with it, a notice names both models, and the next
// message goes to the new thread; there is no second fallback (proposal
// step 5, item 3).
func TestModelFallback(t *testing.T) {
	p := &Provider{}
	r, _ := start(t, "replay", p, "", probe("p6-codex-app-server.jsonl"), probe("p6b-codex-app-server.jsonl"))
	if st := r.wait(event.SessionStarted).Payload.(event.Started); st.SessionID != p6Thread || st.Model != "gpt-6-sol" {
		t.Errorf("started %+v", st)
	}
	r.limits()
	r.send("make hello.txt")
	refusal := r.wait(event.Error).Payload.(event.Problem)
	if refusal.Message != "The 'gpt-6-sol' model is not supported when using Codex with a ChatGPT account." {
		t.Errorf("refusal %q", refusal.Message)
	}
	if end := r.wait(event.TurnCompleted).Payload.(event.TurnEnd); end.Outcome != "failed" {
		t.Errorf("turn %+v", end)
	}
	notice := r.waitFor("notice", func(ev event.Event) bool {
		s, ok := ev.Payload.(event.Status)
		return ok && s.Status == "notice"
	}).Payload.(event.Status)
	want := "Codex refused the model gpt-6-sol: a new thread uses gpt-5.6-sol, the default model/list reports. Send your message again."
	if notice.Detail != want {
		t.Errorf("notice %q", notice.Detail)
	}
	if r.st.Session.ID != p6bThread || r.st.Session.Model != "gpt-5.6-sol" || r.st.Session.Phase != state.Ready {
		t.Errorf("session %q %q %q", r.st.Session.ID, r.st.Session.Model, r.st.Session.Phase)
	}
	if p.fallbackModel() != "gpt-5.6-sol" {
		t.Errorf("provider keeps %q for a resume", p.fallbackModel())
	}
	shown := false
	for _, e := range r.st.Conversation {
		shown = shown || e.Text == want
	}
	if !shown {
		t.Error("the notice is not in the conversation")
	}
	r.limits()

	r.send("make hello.txt")
	req := r.wait(event.ApprovalRequest).Payload.(event.Approval)
	if err := r.sess.Answer(req.ID, event.Allow); err != nil {
		t.Fatal(err)
	}
	if end := r.wait(event.TurnCompleted).Payload.(event.TurnEnd); end.Outcome != "completed" {
		t.Errorf("turn %+v", end)
	}
	r.close()
	r.expect([]string{
		initLine,
		`{"method":"initialized"}`,
		`{"id":2,"method":"config/read","params":{"cwd":"<dir>","includeLayers":false}}`,
		`{"id":3,"method":"configRequirements/read"}`,
		`{"id":4,"method":"model/list","params":{}}`,
		`{"id":5,"method":"thread/start","params":{"sandbox":"read-only","approvalPolicy":"on-request","approvalsReviewer":"user"}}`,
		`{"id":6,"method":"account/rateLimits/read"}`,
		turnLine(7, p6Thread, "make hello.txt"),
		`{"id":8,"method":"model/list","params":{}}`,
		`{"id":9,"method":"thread/start","params":{"model":"gpt-5.6-sol","sandbox":"read-only","approvalPolicy":"on-request","approvalsReviewer":"user"}}`,
		`{"id":10,"method":"account/rateLimits/read"}`,
		turnLine(11, p6bThread, "make hello.txt"),
		`{"id":0,"result":{"decision":"accept"}}`,
	})
}

// TestNoSecondFallback: a session already on the fallback model does not
// fall back again when it too is refused.
func TestNoSecondFallback(t *testing.T) {
	p := &Provider{}
	p.setFallbackModel("gpt-6-sol")
	r, _ := start(t, "replay", p, "", probe("p6-codex-app-server.jsonl"))
	r.wait(event.SessionStarted)
	r.limits()
	r.send("make hello.txt")
	r.wait(event.TurnCompleted)
	r.until(func(event.Event) bool { return false }, 300*time.Millisecond)
	r.close()
	// One model/list: the session's models at the start (plan 34), no
	// fallback's.
	lists := 0
	for _, l := range r.written() {
		if strings.Contains(l, "model/list") {
			if lists++; lists > 1 {
				t.Errorf("a second fallback: %s", l)
			}
		}
	}
}

// TestChildApproval replays P7: the approval a subagent's thread asks for is
// routed to that agent and answered with its id (proposal step 5, item 1).
func TestChildApproval(t *testing.T) {
	r, _ := start(t, "replay", nil, "", probe("p7-codex-subagents.jsonl"))
	r.wait(event.SessionStarted)
	r.limits()
	r.send("spawn two")
	ev := r.wait(event.ApprovalRequest)
	if ev.Agent != p7Child || r.st.Agents[p7Child] == nil {
		t.Fatalf("request from %q, agents %v", ev.Agent, r.st.Agents)
	}
	if err := r.sess.Answer(ev.Payload.(event.Approval).ID, event.Allow); err != nil {
		t.Fatal(err)
	}
	r.waitFor("main turn end", func(ev event.Event) bool { return ev.Kind == event.TurnCompleted && ev.Agent == "" })
	r.close()
	r.expect([]string{
		initLine,
		`{"method":"initialized"}`,
		`{"id":2,"method":"config/read","params":{"cwd":"<dir>","includeLayers":false}}`,
		`{"id":3,"method":"configRequirements/read"}`,
		`{"id":4,"method":"model/list","params":{}}`,
		`{"id":5,"method":"thread/start","params":{"sandbox":"read-only","approvalPolicy":"on-request","approvalsReviewer":"user"}}`,
		`{"id":6,"method":"account/rateLimits/read"}`,
		turnLine(7, p7Thread, "spawn two"),
		`{"id":0,"result":{"decision":"accept"}}`,
	})
}

// TestAnswersEchoTheID: decline and cancel are written as such, with the
// request's id exactly as received, a string here (proposal step 5, item 4).
func TestAnswersEchoTheID(t *testing.T) {
	ask := func(id string) []string {
		return []string{
			`{"dir":"out","msg":{"method":"item/commandExecution/requestApproval","id":` + id + `,"params":{"threadId":"th-1","turnId":"tu-1","itemId":"it-` + id[1:2] + `","command":"ls"}}}`,
		}
	}
	recs := append([]string{}, handshake...)
	recs = append(recs, `{"dir":"in","msg":{"id":4,"method":"turn/start","params":{"threadId":"th-1"}}}`,
		`{"dir":"out","msg":{"id":4,"result":{"turn":{"id":"tu-1"}}}}`)
	recs = append(recs, ask(`"r1"`)...)
	recs = append(recs, `{"dir":"in","msg":{"id":"r1","result":{}}}`)
	recs = append(recs, ask(`"r2"`)...)
	recs = append(recs, `{"dir":"in","msg":{"id":"r2","result":{}}}`,
		`{"dir":"out","msg":{"method":"turn/completed","params":{"threadId":"th-1","turn":{"id":"tu-1","status":"completed"}}}}`)
	r, _ := start(t, "replay", nil, "", script(t, recs...))
	r.wait(event.SessionReady)
	r.limits()
	r.send("hi")
	first := r.wait(event.ApprovalRequest).Payload.(event.Approval)
	if err := r.sess.Answer(first.ID, event.Deny); err != nil {
		t.Fatal(err)
	}
	second := r.wait(event.ApprovalRequest).Payload.(event.Approval)
	if err := r.sess.Answer(second.ID, "cancel"); err != nil {
		t.Fatal(err)
	}
	r.wait(event.TurnCompleted)
	r.close()
	lines := r.written()
	got := strings.Join(lines[len(lines)-2:], "\n")
	if got != `{"id":"r1","result":{"decision":"decline"}}`+"\n"+`{"id":"r2","result":{"decision":"cancel"}}` {
		t.Errorf("answers:\n%s", got)
	}
	if a := r.st.Approvals["r1"]; a == nil || a.Status != state.Denied {
		t.Errorf("approval %+v", a)
	}
}

// TestUnknownServerRequest: a request the dashboard does not handle gets a
// JSON-RPC error -32601 and is counted unrecognised, not shown as an error.
func TestUnknownServerRequest(t *testing.T) {
	recs := append(append([]string{}, handshake...),
		`{"dir":"in","msg":{"id":4,"method":"turn/start","params":{"threadId":"th-1"}}}`,
		`{"dir":"out","msg":{"id":4,"result":{"turn":{"id":"tu-1"}}}}`,
		`{"dir":"out","msg":{"method":"item/tool/requestUserInput","id":9,"params":{"threadId":"th-1","turnId":"tu-1","itemId":"it-1","questions":[]}}}`,
		`{"dir":"in","msg":{"id":9,"error":{}}}`,
		`{"dir":"out","msg":{"method":"turn/completed","params":{"threadId":"th-1","turn":{"id":"tu-1","status":"completed"}}}}`)
	r, _ := start(t, "replay", nil, "", script(t, recs...))
	r.wait(event.SessionReady)
	r.limits()
	r.send("hi")
	r.wait(event.TurnCompleted)
	r.close()
	lines := r.written()
	if l := lines[len(lines)-1]; l != `{"error":{"code":-32601,"message":"Agent Command Center does not handle item/tool/requestUserInput"},"id":9}` {
		t.Errorf("reply %s", l)
	}
	if r.st.Unrecognised != 1 || len(r.st.Errors) != 0 || len(r.st.Approvals) != 0 {
		t.Errorf("unrecognised %d, errors %+v, approvals %d", r.st.Unrecognised, r.st.Errors, len(r.st.Approvals))
	}
}

// TestUnreadableRequest: a request without its item is declined by the
// reducer's fallback, and only a decline is taken for it.
func TestUnreadableRequest(t *testing.T) {
	recs := append(append([]string{}, handshake...),
		`{"dir":"in","msg":{"id":4,"method":"turn/start","params":{"threadId":"th-1"}}}`,
		`{"dir":"out","msg":{"id":4,"result":{"turn":{"id":"tu-1"}}}}`,
		`{"dir":"out","msg":{"method":"item/fileChange/requestApproval","id":5,"params":{"threadId":"th-1","turnId":"tu-1"}}}`,
		`{"dir":"in","msg":{"id":5,"result":{}}}`,
		`{"dir":"out","msg":{"method":"turn/completed","params":{"threadId":"th-1","turn":{"id":"tu-1","status":"completed"}}}}`)
	r, _ := start(t, "replay", nil, "", script(t, recs...))
	r.wait(event.SessionReady)
	r.limits()
	r.send("hi")
	if a := r.wait(event.ApprovalRequest).Payload.(event.Approval); a.Problem == "" {
		t.Fatalf("request %+v is readable", a)
	}
	if err := r.sess.Answer("5", event.Allow); err == nil {
		t.Error("an unreadable request was accepted")
	}
	a := r.st.Approvals["5"]
	if a == nil || a.Status != state.Denied || a.Answer != event.Deny {
		t.Fatalf("approval %+v", a)
	}
	if err := r.sess.Answer("5", a.Answer); err != nil {
		t.Fatal(err)
	}
	r.wait(event.TurnCompleted)
	r.close()
	lines := r.written()
	if l := lines[len(lines)-1]; l != `{"id":5,"result":{"decision":"decline"}}` {
		t.Errorf("answer %s", l)
	}
}

// TestResume: a session context with a thread resumes it, with the fallback
// model when this run fell back, and never starts a new one (proposal step
// 5, item 5).
func TestResume(t *testing.T) {
	recs := []string{
		handshake[0], handshake[1], handshake[2],
		`{"dir":"in","msg":{"id":2,"method":"thread/resume","params":{"threadId":"th-9"}}}`,
		`{"dir":"out","msg":{"id":2,"result":{"thread":{"id":"th-9"},"model":"gpt-5.6-sol","cwd":"<probe-work>"}}}`,
		handshake[5], handshake[6],
	}
	p := &Provider{}
	p.setFallbackModel("gpt-5.6-sol")
	r, _ := start(t, "replay", p, "th-9", script(t, recs...))
	if st := r.wait(event.SessionStarted).Payload.(event.Started); st.SessionID != "th-9" {
		t.Errorf("started %+v", st)
	}
	r.limits()
	r.close()
	r.expect([]string{
		initLine,
		`{"method":"initialized"}`,
		`{"id":2,"method":"config/read","params":{"cwd":"<dir>","includeLayers":false}}`,
		`{"id":3,"method":"configRequirements/read"}`,
		`{"id":4,"method":"model/list","params":{}}`,
		`{"id":5,"method":"thread/resume","params":{"threadId":"th-9","model":"gpt-5.6-sol","sandbox":"read-only","approvalPolicy":"on-request","approvalsReviewer":"user"}}`,
		`{"id":6,"method":"account/rateLimits/read"}`,
	})
}

// TestRefusedThreadStart: a refused thread/start fails the handshake with
// Codex's message.
func TestRefusedThreadStart(t *testing.T) {
	recs := []string{handshake[0], handshake[1], handshake[2],
		`{"dir":"in","msg":{"id":2,"method":"thread/start"}}`,
		`{"dir":"out","msg":{"id":2,"error":{"code":-32600,"message":"bad sandbox"}}}`}
	r, _ := start(t, "replay", nil, "", script(t, recs...))
	r.wait(event.Error)
	if r.st.Session.Phase != state.Crashed || !strings.Contains(r.st.Session.Reason, "bad sandbox") {
		t.Errorf("session %q %q", r.st.Session.Phase, r.st.Session.Reason)
	}
}

// TestNoHandshake: an app-server that never answers initialize ends, after
// the handshake timeout, crashed with its message.
func TestNoHandshake(t *testing.T) {
	r, _ := start(t, "silent", &Provider{CloseStep: time.Second}, "")
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
	if r.st.Session.Phase != state.Crashed || r.st.Startup.Phase != state.StartFailed {
		t.Errorf("session %q %q, startup %q", r.st.Session.Phase, r.st.Session.Reason, r.st.Startup.Phase)
	}
}

// TestExitMidTurn: the app-server dies during a turn; the session is
// crashed, the turn failed, and the exit code is kept (proposal step 5,
// item 6).
func TestExitMidTurn(t *testing.T) {
	recs := append(append([]string{}, handshake...),
		`{"dir":"in","msg":{"id":4,"method":"turn/start","params":{"threadId":"th-1"}}}`,
		`{"dir":"out","msg":{"id":4,"result":{"turn":{"id":"tu-1"}}}}`,
		`{"dir":"out","msg":{"method":"item/agentMessage/delta","params":{"threadId":"th-1","turnId":"tu-1","itemId":"m1","delta":"Hel"}}}`,
		`{"dir":"die","msg":3}`)
	r, _ := start(t, "replay", nil, "", script(t, recs...))
	r.wait(event.SessionReady)
	r.limits()
	r.send("hello")
	e := r.end()
	if !e.HasCode || e.Code != 3 || e.Reason != "Codex exited with code 3" {
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

// TestLongLine: a 65 MiB line is skipped with an error giving its size, and
// the next line is read (ASM-20).
func TestLongLine(t *testing.T) {
	recs := append([]string{}, handshake[:6]...)
	recs = append(recs, `{"dir":"huge","msg":null}`, handshake[6])
	r, dir := start(t, "replay", nil, "", script(t, recs...))
	if st := r.wait(event.SessionStarted).Payload.(event.Started); st.Cwd != dir {
		t.Errorf("cwd %q", st.Cwd)
	}
	ev := r.wait(event.Error)
	want := fmt.Sprintf("a line of %d bytes from Codex was skipped (limit 64 MiB)", 65<<20+len(`{"method":"pad","params":""}`)+1)
	if msg := ev.Payload.(event.Problem).Message; msg != want {
		t.Errorf("error %q, want %q", msg, want)
	}
	r.limits() // the line after it is read
	if r.st.Session.Phase != state.Ready || len(r.st.Errors) != 1 {
		t.Errorf("phase %q, errors %+v", r.st.Session.Phase, r.st.Errors)
	}
}

// pids reads the stubborn app-server's and its grandchild's process ids.
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

// TestCloseEscalation: an app-server that ignores its closed stdin is gone,
// with what it started, within 11 s.
func TestCloseEscalation(t *testing.T) {
	r, _ := start(t, "stubborn", nil, "")
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
	if e := r.end(); !strings.HasPrefix(e.Reason, "Codex closed") {
		t.Errorf("ended %+v", e)
	}
	if r.st.Session.Phase != state.Closed {
		t.Errorf("phase %q", r.st.Session.Phase)
	}
}

// TestForcedKill: cancelling Start's context ends the whole tree at once.
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

func TestStartNeedsExecutableAndProject(t *testing.T) {
	if _, err := (&Provider{}).Start(context.Background(), provider.SessionContext{ProjectPath: "x"}); err == nil {
		t.Error("started without an executable")
	}
	if _, err := (&Provider{}).Start(context.Background(), provider.SessionContext{Executable: "x"}); err == nil {
		t.Error("started without a project")
	}
}

// TestConfigReadOnce: config/read is written once, after initialized, with
// the project directory as cwd and no layers; only the limit reaches the
// state, and nothing of the reply stays in an event (plan 5).
func TestConfigReadOnce(t *testing.T) {
	recs := append(append([]string{}, handshake...),
		`{"dir":"in","msg":{"id":9,"method":"config/read"}}`,
		`{"dir":"out","msg":{"id":9,"result":{"config":{"mcp_servers":{"x":{"env":{"TOKEN":"SECRET-MARKER-123"}}},"agents":{"max_concurrent_threads_per_session":3}}}}}`)
	r, _ := start(t, "replay", nil, "", script(t, recs...))
	r.waitFor("the limit", func(ev event.Event) bool { return strings.Contains(ev.Raw, "config/read reply not kept") })
	r.limits()
	if v := r.st.Metrics[event.LimitMetric]; v.V != 3 {
		t.Errorf("limit %+v", v)
	}
	r.close()
	n := 0
	for _, l := range r.written() {
		if strings.Contains(l, `"config/read"`) {
			n++
			want := `{"id":2,"method":"config/read","params":{"cwd":` + quote(r.dir) + `,"includeLayers":false}}`
			if l != want {
				t.Errorf("written %s, want %s", l, want)
			}
		}
	}
	if n != 1 {
		t.Errorf("%d config/read written", n)
	}
	for _, e := range r.evs {
		if strings.Contains(e.Raw, "SECRET-MARKER-123") || strings.Contains(fmt.Sprintf("%+v", e), "SECRET-MARKER-123") {
			t.Errorf("event keeps the secret: %s", e.Kind)
		}
	}
}

// TestSetApprovalMode is plan 12, item 2 (Manual at every launch:
// thread/start carries approvalsReviewer user) and plan 39's criterion
// 39.4: between turns SetApprovalMode writes thread/settings/update with
// Approve for me's three settings at once; P6b's Codex never answers it,
// so the next turn/start carries them, and that turn puts the mode in
// force. A word that is not one of Codex's three is refused, unwritten.
func TestSetApprovalMode(t *testing.T) {
	r, _ := start(t, "replay", nil, "", probe("p6b-codex-app-server.jsonl"))
	r.wait(event.SessionStarted)
	r.limits()
	if !r.sess.Capabilities().Modes {
		t.Error("Codex does not offer its modes")
	}
	if r.st.Mode() != state.ModeManual {
		t.Errorf("mode %q at launch", r.st.Mode())
	}
	for _, m := range []string{event.ModeAcceptEdits, event.ModePlan, event.ModeCustom, "", "bypassPermissions"} {
		if r.sess.SetApprovalMode(m) == nil {
			t.Errorf("Codex took the mode %q", m)
		}
	}
	if err := r.sess.SetApprovalMode(event.ModeAuto); err != nil {
		t.Fatal(err)
	}
	update := `{"id":7,"method":"thread/settings/update","params":{"threadId":"` + p6bThread + `","sandboxPolicy":{"type":"workspaceWrite"},"approvalPolicy":"on-request","approvalsReviewer":"auto_review"}}`
	deadline := time.Now().Add(5 * time.Second)
	for len(r.written()) < 8 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if got := r.written(); len(got) != 8 || got[7] != update {
		t.Errorf("after SetApprovalMode, want the handshake's 7 lines and\n%s\ngot %q", update, got)
	}
	if r.st.Mode() != state.ModeManual {
		t.Errorf("mode %q before Codex reported one", r.st.Mode())
	}
	r.send("make hello.txt")
	req := r.wait(event.ApprovalRequest).Payload.(event.Approval)
	if r.st.Mode() != state.ModeAuto {
		t.Errorf("mode %q after the turn", r.st.Mode())
	}
	if err := r.sess.Answer(req.ID, event.Allow); err != nil {
		t.Fatal(err)
	}
	r.wait(event.TurnCompleted)
	r.close()
	got := r.written()
	if len(got) != 10 || got[9] != `{"id":0,"result":{"decision":"accept"}}` {
		t.Fatalf("written %q", got)
	}
	// Auto is Codex's "Approve for me": auto_review with workspace-write
	// (plan 12 addendum).
	want := strings.NewReplacer(`"approvalsReviewer":"user"`, `"approvalsReviewer":"auto_review"`,
		`"sandboxPolicy":{"type":"readOnly"}`, `"sandboxPolicy":{"type":"workspaceWrite"}`).Replace(turnLine(8, p6bThread, "make hello.txt"))
	if got[8] != want {
		t.Errorf("turn/start\n got %s\nwant %s", got[8], want)
	}
	if s := r.st.Session.Sandbox; s != "workspaceWrite" {
		t.Errorf("sandbox shown %q after the Auto turn, want workspaceWrite", s)
	}
}

// p39xScript is P39x without the updates the adapter never writes: its
// first, VS Code's workspace-write "Ask for approval", and its last, a
// change of nothing.
func p39xScript(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(probe("p39x-codex-permission-modes.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	var recs []string
	skip := false
	for _, l := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		var rec record
		if err := json.Unmarshal([]byte(l), &rec); err != nil {
			t.Fatal(err)
		}
		if rec.Dir == "step" {
			skip = strings.Contains(string(rec.Msg), `"Ask for approval"`) || strings.Contains(string(rec.Msg), `"read-only, no change"`)
		}
		if !skip {
			recs = append(recs, l)
		}
	}
	return script(t, recs...)
}

// TestModesBetweenTurns replays P39x, Codex 0.163.0 taking each preset with
// thread/settings/update (plan 39, criteria 39.3 to 39.5): each change asked
// for in the state, written once with its three settings, in force only
// once thread/settings/updated reports it, Full access included once
// confirmed; the requirements read once, none left out.
func TestModesBetweenTurns(t *testing.T) {
	r, _ := start(t, "replay", nil, "", p39xScript(t))
	r.wait(event.SessionStarted)
	thread := r.st.Session.ID
	if r.st.Mode() != state.ModeManual || thread == "" {
		t.Fatalf("mode %q on thread %q at launch", r.st.Mode(), thread)
	}
	if cmds := r.st.Do(state.ChooseMode{Mode: event.ModeFull, Capable: true}); cmds != nil {
		t.Fatalf("Full access asked for unconfirmed: %+v", cmds)
	}
	from := state.ModeManual
	for step, c := range []struct{ mode, sandbox, policy, reviewer string }{
		{event.ModeAuto, "workspaceWrite", "on-request", "auto_review"},
		{event.ModeFull, "dangerFullAccess", "never", "user"},
		{event.ModeManual, "readOnly", "on-request", "user"},
	} {
		cmds := r.st.Do(state.ChooseMode{Mode: c.mode, Confirmed: true, Capable: true})
		if len(cmds) != 1 || cmds[0].Kind != state.CmdSetApprovals || cmds[0].Mode != c.mode {
			t.Fatalf("%s: commands %+v", c.mode, cmds)
		}
		if err := r.sess.SetApprovalMode(cmds[0].Mode); err != nil {
			t.Fatal(err)
		}
		if r.st.Mode() != from || r.st.Session.ApprovalRequested != c.mode {
			t.Errorf("%s: mode %q requested %q before Codex reported", c.mode, r.st.Mode(), r.st.Session.ApprovalRequested)
		}
		ev := r.waitFor(c.mode+" in force", func(ev event.Event) bool {
			p, ok := ev.Payload.(event.Mode)
			return ok && p.Mode != ""
		})
		if p := ev.Payload.(event.Mode); p.Mode != c.mode || !p.Answers || p.Sandbox != c.sandbox {
			t.Errorf("%s: reported %+v", c.mode, p)
		}
		if r.st.Mode() != c.mode || r.st.Session.ApprovalRequested != "" {
			t.Errorf("%s: mode %q requested %q after the report", c.mode, r.st.Mode(), r.st.Session.ApprovalRequested)
		}
		var updates []string
		for _, l := range r.written() {
			if strings.Contains(l, `"method":"thread/settings/update"`) {
				updates = append(updates, l)
			}
		}
		want := `"method":"thread/settings/update","params":{"threadId":"` + thread + `","sandboxPolicy":{"type":"` + c.sandbox + `"},"approvalPolicy":"` + c.policy + `","approvalsReviewer":"` + c.reviewer + `"}}`
		if n := len(updates); n != step+1 || !strings.HasSuffix(updates[n-1], want) {
			t.Errorf("%s: updates written %q\nwant %d, the last ...%s", c.mode, updates, step+1, want)
		}
		from = c.mode
	}
	n := 0
	for _, l := range r.written() {
		if strings.Contains(l, `"method":"configRequirements/read"`) {
			n++
		}
	}
	if n != 1 || len(r.st.Session.ModesUnavailable) != 0 {
		t.Errorf("%d requirements reads, modes left out %v", n, r.st.Session.ModesUnavailable)
	}
	r.close()
}
