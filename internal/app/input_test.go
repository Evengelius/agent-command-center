package app

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/proc"
	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// fakeClaude is an in-memory claude process for the real Claude Code
// adapter: it keeps every line written to its stdin, answers initialize and
// ends each user turn with an init and a result.
type fakeClaude struct {
	mu    sync.Mutex
	lines [][]byte
}

func (f *fakeClaude) Spawn(sp proc.Spec) (proc.Process, error) {
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	errR, errW := io.Pipe()
	p := &fakeProcess{inR: inR, inW: inW, outR: outR, outW: outW, errR: errR, errW: errW, done: make(chan struct{})}
	cwd, _ := json.Marshal(sp.Dir)
	go func() {
		defer close(p.done)
		defer errW.Close()
		defer outW.Close()
		sc := bufio.NewScanner(inR)
		sc.Buffer(make([]byte, 64<<10), 64<<20)
		for sc.Scan() {
			l := append([]byte(nil), sc.Bytes()...)
			f.mu.Lock()
			f.lines = append(f.lines, l)
			f.mu.Unlock()
			var m struct {
				Type      string `json:"type"`
				RequestID string `json:"request_id"`
				Request   struct {
					Subtype string `json:"subtype"`
				} `json:"request"`
			}
			_ = json.Unmarshal(l, &m)
			var out string
			switch {
			case m.Type == "control_request" && m.Request.Subtype == "initialize":
				id, _ := json.Marshal(m.RequestID)
				out = `{"type":"control_response","response":{"subtype":"success","request_id":` + string(id) + `,"response":{}}}` + "\n"
			case isUsageRead(string(l)):
				// The adapter's own usage read, answered as Claude Code does:
				// the report, then a result that ends no turn (plan 24, amendment 23).
				out = usageReport + "\n" + `{"type":"result","subtype":"success","is_error":false,"duration_ms":2,"duration_api_ms":0,"num_turns":0,"result":"Current session","session_id":"s-usage","total_cost_usd":0,"usage":{},"modelUsage":{}}` + "\n"
			case m.Type == "user":
				out = `{"type":"system","subtype":"init","cwd":` + string(cwd) + `,"session_id":"s-input","model":"claude-haiku-5-5","permissionMode":"default"}` + "\n" +
					`{"type":"result","subtype":"success","is_error":false,"duration_ms":1,"result":"ok","session_id":"s-input"}` + "\n"
			}
			if _, err := io.WriteString(outW, out); err != nil {
				return
			}
		}
	}()
	return p, nil
}

// users is every user message written to stdin, as written, without the
// adapter's own usage reads (plan 24, amendment 23).
func (f *fakeClaude) users() [][]byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	var us [][]byte
	for _, l := range f.lines {
		var m struct {
			Type string `json:"type"`
		}
		if json.Unmarshal(l, &m) == nil && m.Type == "user" && !isUsageRead(string(l)) {
			us = append(us, l)
		}
	}
	return us
}

type fakeProcess struct {
	inR        *io.PipeReader
	inW        *io.PipeWriter
	outR, errR *io.PipeReader
	outW, errW *io.PipeWriter
	done       chan struct{}
}

func (p *fakeProcess) Stdin() io.WriteCloser { return p.inW }
func (p *fakeProcess) Stdout() io.Reader     { return p.outR }
func (p *fakeProcess) Stderr() io.Reader     { return p.errR }
func (p *fakeProcess) Pid() int              { return 4242 }
func (p *fakeProcess) Wait() (int, error)    { <-p.done; return 0, nil }
func (p *fakeProcess) Terminate() error      { return p.Kill() }
func (p *fakeProcess) Kill() error {
	_ = p.inR.Close()
	_ = p.outW.Close()
	_ = p.errW.Close()
	return nil
}
func (p *fakeProcess) Release() {}

// installedClaude is the real Claude Code adapter, found installed without
// looking for claude.exe.
type installedClaude struct{ *claude.Provider }

func (installedClaude) Detect(context.Context) provider.Detection {
	return provider.Detection{ID: claude.ProviderID, Name: claude.Name, Installed: true, Version: "2.1.300", Path: `C:\Tools\claude.exe`}
}

// inputRig is one adapter under the input tests (proposal step 3, item 8).
type inputRig struct {
	h    *harness
	sent func() []string // the texts the session received, in order
	end  func()          // ends the turn a send started, if the adapter does not
	// raw checks the bytes written for the i-th message, when the adapter
	// writes any.
	raw func(t *testing.T, i int, want string)
	// script and emit are the scripted session and its events, for scriptRig.
	script *scriptSession
	emit   func(k event.Kind, p any)
}

func claudeRig(t *testing.T) inputRig { return claudeRigWith(t, Options{}) }

// claudeRigWith is claudeRig with the given options (the keys of plan 21, item 5.1).
func claudeRigWith(t *testing.T, opt Options) inputRig {
	f := &fakeClaude{}
	cli := installedClaude{&claude.Provider{Spawner: f, Now: func() time.Time { return testNow }}}
	opt.Providers, opt.Cwd = []provider.Provider{cli}, t.TempDir()
	h := newHarness(t, opt)
	h.settle()
	h.m.ui.Choice = claude.ProviderID
	h.send(key(tea.KeyEnter, 0))
	h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
	if s := h.m.State(); s.Session.Phase != state.Ready {
		t.Fatalf("Claude session %q, startup %q: want Ready", s.Session.Phase, s.Startup.Phase)
	}
	return inputRig{
		h: h,
		sent: func() []string {
			var out []string
			for _, l := range f.users() {
				var m struct {
					Message struct {
						Content string `json:"content"`
					} `json:"message"`
				}
				if err := json.Unmarshal(l, &m); err != nil {
					t.Fatalf("user line %q: %v", l, err)
				}
				out = append(out, m.Message.Content)
			}
			return out
		},
		end: func() {},
		raw: func(t *testing.T, i int, want string) {
			t.Helper()
			var b bytes.Buffer
			enc := json.NewEncoder(&b)
			enc.SetEscapeHTML(false)
			_ = enc.Encode(map[string]any{
				"type":               "user",
				"message":            map[string]any{"role": "user", "content": want},
				"parent_tool_use_id": nil,
				"session_id":         "default",
			})
			us := f.users()
			if i >= len(us) {
				t.Fatalf("message %d not written (%d written)", i, len(us))
			}
			if got := string(us[i]) + "\n"; got != b.String() {
				t.Errorf("stdin line\n got %q\nwant %q", got, b.String())
			}
		},
	}
}

// scriptRig drives a scripted session under the given adapter id, for UI
// tests that emit arbitrary events; the real adapters have their own rigs.
func scriptRig(t *testing.T, id string) inputRig {
	h, cli := startScript(t, id)
	ev := func(k event.Kind, p any) {
		cli.sess.events <- event.Event{Provider: id, Kind: k, Origin: event.Observed, Payload: p}
	}
	ev(event.SessionSpawned, event.Spawned{Argv: []string{`C:\Tools\cli.exe`}})
	ev(event.SessionReady, event.Ready{Detail: "scripted"})
	h.settle()
	return inputRig{
		h:      h,
		sent:   func() []string { s, _, _ := cli.sess.calls(); return s },
		script: cli.sess,
		emit:   ev,
		end: func() {
			ev(event.TurnStarted, nil)
			ev(event.TurnCompleted, event.TurnEnd{Outcome: "completed", Subtype: "success"})
			h.settle()
		},
	}
}

func mockRig(t *testing.T) inputRig { return mockRigWith(t, Options{}) }

// mockRigWith is mockRig with the given options.
func mockRigWith(t *testing.T, opt Options) inputRig {
	opt.Config = Config{Mock: filepath.Join("..", "..", "testdata", "mock", "claude-subagent-approval.jsonl")}
	opt.Providers, opt.Cwd = []provider.Provider{spawnTrap{t}}, t.TempDir()
	h := newHarness(t, opt)
	h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
	base := len(h.m.State().Log)
	return inputRig{
		h: h,
		sent: func() []string {
			var out []string
			for _, e := range h.m.State().Log[base:] {
				if u, ok := e.Payload.(event.User); ok && e.Kind == event.UserMessage {
					out = append(out, u.Text)
				}
			}
			return out
		},
		end: func() {},
	}
}

func rigs() map[string]func(*testing.T) inputRig {
	return map[string]func(*testing.T) inputRig{
		"claude-code": claudeRig,
		"codex":       codexRig,
		"mock":        mockRig,
	}
}

// rigsWith are the three rigs with the given options.
func rigsWith(opt Options) map[string]func(*testing.T) inputRig {
	return map[string]func(*testing.T) inputRig{
		"claude-code": func(t *testing.T) inputRig { return claudeRigWith(t, opt) },
		"codex":       func(t *testing.T) inputRig { return codexRigWith(t, opt) },
		"mock":        func(t *testing.T) inputRig { return mockRigWith(t, opt) },
	}
}

// pasteCases are test 5's inputs: CR is stored as LF, nothing else changes.
var pasteCases = []struct{ name, in, want string }{
	{"LF", "one\ntwo\n\nthree", "one\ntwo\n\nthree"},
	{"CRLF", "one\r\ntwo\r\nthree", "one\ntwo\nthree"},
	{"lone CR", "one\rtwo\r\rthree", "one\ntwo\n\nthree"},
	{"mixed breaks", "a\r\nb\rc\nd\r\n\re", "a\nb\nc\nd\n\ne"},
	{"tabs", "\tfunc() {\n\t\treturn\t1\n\t}", "\tfunc() {\n\t\treturn\t1\n\t}"},
	{"trailing spaces", "line   \n  \nend \t ", "line   \n  \nend \t "},
	{"combining marks", "e\u0301 n\u0303 a\u0308\u0304 ก\u0e34", "e\u0301 n\u0303 a\u0308\u0304 ก\u0e34"},
	{"emoji", "👍🏽 👨‍👩‍👧 🇧🇪 ❤️ <&>", "👍🏽 👨‍👩‍👧 🇧🇪 ❤️ <&>"},
	{"wide", "日本語 한국어 中文　全角", "日本語 한국어 中文　全角"},
	{"C0 controls", "a\x01b\x07c\x1b[31md\x7fe\x00f\x1fg", "a\x01b\x07c\x1b[31md\x7fe\x00f\x1fg"},
}

// TestInputByteForByte is test 5 (proposal step 3, items 1 and 8): each
// paste is stored as expected and reaches the session unchanged, the same
// with every adapter; with Claude Code the stdin line is checked byte for
// byte.
func TestInputByteForByte(t *testing.T) {
	for name, rig := range rigs() {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				r := rig(t)
				for i, c := range pasteCases {
					r.h.send(tea.PasteMsg{Content: c.in})
					if got := r.h.m.edit.Text(); got != c.want {
						t.Errorf("%s: stored %q, want %q", c.name, got, c.want)
					}
					r.h.send(key(tea.KeyEnter, 0))
					r.end()
					sent := r.sent()
					if len(sent) != i+1 {
						t.Fatalf("%s: %d messages sent, want %d (notice %q)", c.name, len(sent), i+1, r.h.m.State().Turn.Notice)
					}
					if sent[i] != c.want {
						t.Errorf("%s: sent %q, want %q", c.name, sent[i], c.want)
					}
					if r.raw != nil {
						r.raw(t, i, c.want)
					}
					if d := r.h.m.edit.Text(); d != "" {
						t.Errorf("%s: draft %q left after sending", c.name, d)
					}
				}
			})
		})
	}
}

// TestPasteNeverSubmits is proposal step 3, item 2: a paste ending in a
// newline writes nothing until Enter, and Enter inside an unfinished paste
// writes nothing and shows "pasting…".
func TestPasteNeverSubmits(t *testing.T) {
	for name, rig := range rigs() {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				r := rig(t)
				r.h.send(tea.PasteMsg{Content: "first line\nsecond line\n"})
				if n := len(r.sent()); n != 0 {
					t.Fatalf("a paste ending in a newline wrote %d messages, want 0", n)
				}
				r.h.send(key(tea.KeyEnter, 0))
				r.end()
				if sent := r.sent(); len(sent) != 1 || sent[0] != "first line\nsecond line\n" {
					t.Fatalf("after Enter: sent %q, want the paste once", sent)
				}

				r.h.send(tea.PasteStartMsg{})
				r.h.send(tea.PasteMsg{Content: "partial"})
				r.h.send(key(tea.KeyEnter, 0))
				if n := len(r.sent()); n != 1 {
					t.Errorf("Enter during a paste: %d messages in all, want 1", n)
				}
				if f := plainText(r.h.m.View().Content); !strings.Contains(f, "pasting…") {
					t.Errorf("no \"pasting…\" while the paste is open:\n%s", f)
				}
				if got := r.h.m.edit.Text(); got != "partial\n" {
					t.Errorf("draft %q during the paste, want %q", got, "partial\n")
				}
				r.h.send(tea.PasteEndMsg{})
				if f := plainText(r.h.m.View().Content); strings.Contains(f, "pasting…") {
					t.Error("\"pasting…\" still shown after the paste ended")
				}
				r.h.send(key(tea.KeyEnter, 0))
				r.end()
				if sent := r.sent(); len(sent) != 2 || sent[1] != "partial\n" {
					t.Errorf("after the paste ended, Enter sent %q, want a second message %q", sent, "partial\n")
				}
			})
		})
	}
}
