package mock_test

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/fixture"
	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/provider/codex"
	"github.com/Evengelius/agent-command-center/internal/provider/mock"
)

var update = flag.Bool("update", false, "rewrite testdata/mock from the probe recordings")

// mockFiles are the mock files in testdata/mock, each the replay of a probe
// recording through its adapter's mapper, without the final session.ended
// so the replayed session stays open.
var mockFiles = []struct {
	file, probe, provider string
	mapper                func() fixture.Mapper
	edit                  func(t *testing.T, recs []fixture.Record) []fixture.Record // a synthetic case; nil for the recording as is
}{
	{"claude-subagent-approval.jsonl", "p3b-claude-subagent-approval.jsonl", "claude-code", func() fixture.Mapper { return claude.NewMapper() }, nil},
	{"codex-subagents.jsonl", "p7-codex-subagents.jsonl", "codex", func() fixture.Mapper { return codex.NewMapper() }, nil},
	{"codex-spawn-request.jsonl", "p9a-codex-spawn-request.jsonl", "codex", func() fixture.Mapper { return codex.NewMapper() }, nil}, // plan 10, item 1
	// Plan 22, criterion 3.3: Claude Code's thinking, live estimates and exact
	// counts, with no subagent (P12c-block, 4 thinking blocks) and with one
	// (P303b, 3 thinking blocks and a subagent's 4 that zone B never shows).
	{"claude-thinking.jsonl", "p12c-block-claude-auto-boundary-edit.jsonl", "claude-code", func() fixture.Mapper { return claude.NewMapper() }, nil},
	{"claude-thinking-subagent.jsonl", "p303b-claude-subagent-approval.jsonl", "claude-code", func() fixture.Mapper { return claude.NewMapper() }, nil},
	// Synthetic: cases no probe has, each a real recording with only the
	// lines of its case edited to the documented shape (plan 4, step 7).
	{"claude-api-key-context-refused.jsonl", "p2-claude-conversation.jsonl", "claude-code", func() fixture.Mapper { return claude.NewMapper() }, claudeAPIKeyContextRefused},
	{"codex-api-key.jsonl", "p7-codex-subagents.jsonl", "codex", func() fixture.Mapper { return codex.NewMapper() }, codexAPIKey},
	{"codex-no-5h-window.jsonl", "p7-codex-subagents.jsonl", "codex", func() fixture.Mapper { return codex.NewMapper() }, codexNo5hWindow},
	{"codex-plan6-limits.jsonl", "p7-codex-subagents.jsonl", "codex", func() fixture.Mapper { return codex.NewMapper() }, codexPlan6Limits},
	// Plan 24, criterion 24.17: P3b with its rate_limit_event carrying three
	// windows, five_hour, seven_day and seven_day_fable, at the owner's
	// image's values (0%, 17%, 0%), resetting 4h 12m and 5d 3h after the
	// application tests' clock (2026-10-15 09:30 UTC). No probe recorded a
	// model-scoped window; "Fable limit" is the owner's (brief amendment
	// 22), its key the documented seven_day_<model> form.
	{"claude-quota-windows.jsonl", "p3b-claude-subagent-approval.jsonl", "claude-code", func() fixture.Mapper { return claude.NewMapper() }, claudeQuotaWindows},
	// Plan 34: the model selector's lists, with no message sent. P34c up to
	// its first settings read (the owner's 13 models, Fable 5.1 at Max with
	// Ultracode); P34x's first run up to the thread's start (7 pages of one
	// model, the thread on gpt-6-sol, High).
	{"claude-models.jsonl", "p34c-claude-model-settings.jsonl", "claude-code", func() fixture.Mapper { return claude.NewMapper() }, claudeModels},
	{"codex-models.jsonl", "p34x-codex-model-list-settings.jsonl", "codex", func() fixture.Mapper { return codex.NewMapper() }, codexModels},
}

// claudeModels is P34c's start: the process, the handshake and its models,
// and the first get_settings answer; none of the changes after it. The
// owner's SessionStart hook lines are left out: the probe kept only their
// type, which would read as a failed hook.
func claudeModels(t *testing.T, recs []fixture.Record) []fixture.Record {
	var out []fixture.Record
	for _, r := range recs {
		if r.Dir != "meta" && r.Dir != "in" && r.Dir != "out" || strings.Contains(string(r.Msg), `"subtype":"hook_`) {
			continue
		}
		out = append(out, r)
		if r.Dir == "out" && strings.Contains(string(r.Msg), `"applied"`) {
			return out
		}
	}
	t.Fatal("P34c: no get_settings answer")
	return nil
}

// codexModels is P34x's first run up to the thread's start, before its
// thread/read: the process (its "run" record as the argv record the other
// Codex recordings have), the handshake, model/list's pages and the thread.
func codexModels(t *testing.T, recs []fixture.Record) []fixture.Record {
	var out []fixture.Record
	for _, r := range recs {
		msg := string(r.Msg)
		if r.Dir == "in" && strings.Contains(msg, `"method":"thread/read"`) {
			return out
		}
		if r.Dir == "run" && len(out) == 0 {
			r.Dir, r.Msg = "argv", []byte(`["app-server"]`)
			out = append(out, r)
			continue
		}
		if r.Dir != "in" && r.Dir != "out" || strings.Contains(msg, `"method":"account/`) || strings.Contains(msg, `"method":"remoteControl/`) {
			continue
		}
		out = append(out, r)
	}
	t.Fatal("P34x: no thread/read")
	return nil
}

// claudeQuotaWindows is P3b with every rate_limit_event's unifiedWindows
// replaced by the three windows of plan 24's criterion 24.17.
func claudeQuotaWindows(t *testing.T, recs []fixture.Record) []fixture.Record {
	out := append([]fixture.Record(nil), recs...)
	n := 0
	for i, r := range out {
		if r.Dir != "out" || !strings.Contains(string(r.Msg), `"type":"rate_limit_event"`) {
			continue
		}
		var v map[string]any
		if err := json.Unmarshal(r.Msg, &v); err != nil {
			t.Fatal(err)
		}
		info, _ := v["rate_limit_info"].(map[string]any)
		if info == nil {
			t.Fatalf("record %d: no rate_limit_info", i)
		}
		info["resetsAt"] = 1792071720
		info["unifiedWindows"] = map[string]any{
			"five_hour":       map[string]any{"utilization": 0.0, "resetsAt": 1792071720},
			"seven_day":       map[string]any{"utilization": 0.17, "resetsAt": 1792499400},
			"seven_day_fable": map[string]any{"utilization": 0.0, "resetsAt": 1792499400},
		}
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		out[i].Msg = b
		n++
	}
	if n == 0 {
		t.Fatal("P3b has no rate_limit_event")
	}
	return out
}

// codexPlan6Limits is P7 with the lines plan 6's simulated run needs and no
// recording has, each in the shape Codex 0.163.0's generated schema gives
// (plan 6): each child's spawnAgent collabAgentToolCall, the first requesting
// a model and an effort, the second neither (item 4); an early orchestrator
// answer in Markdown whose code holds strings over several lines, long enough
// to scroll zone B while later entries arrive (items 2 and 3); and a failed
// orchestrator command whose aggregated output ends with its error after its
// standard output (item 1).
func codexPlan6Limits(t *testing.T, recs []fixture.Record) []fixture.Record {
	const main, turn = "01a1419a-23c6-74f1-837d-b5da2cbe1495", "01a1419a-253e-7461-9514-7e3acbeebd84"
	rec := func(ms int64, method string, it map[string]any) fixture.Record {
		b, err := json.Marshal(map[string]any{"method": method, "params": map[string]any{"item": it, "threadId": main, "turnId": turn}})
		if err != nil {
			t.Fatal(err)
		}
		return fixture.Record{TMs: ms, Dir: "out", Msg: b}
	}
	spawn := func(id, status string, receivers []string, prompt string, model, effort any) map[string]any {
		return map[string]any{"type": "collabAgentToolCall", "id": id, "tool": "spawnAgent", "status": status,
			"senderThreadId": main, "receiverThreadIds": receivers, "prompt": prompt, "model": model,
			"reasoningEffort": effort, "agentsStates": map[string]any{}}
	}
	children := []struct {
		thread, call, prompt string
		model, effort        any
	}{
		{"01a1419a-384f-7d13-bd22-b375637ac31b", "call_plan6_spawn_list", "List the files in the current directory.", "gpt-6-mini", "low"},
		{"01a1419a-4212-7712-b732-30b9c9bd385c", "call_plan6_spawn_found", "Create found.txt containing the single word found.", nil, nil},
	}
	answer := strings.Join([]string{
		"I'll check the plan 6 fixes before the two subagents start.",
		"",
		"## Strings over several lines",
		"",
		"```go",
		"query := `SELECT name",
		"  FROM \"agents\" -- not a comment",
		"  WHERE id = 1 /* nor this */`",
		"count := 3 // a comment again",
		"```",
		"",
		"```python",
		"doc = \"\"\"First line",
		"  # not a comment",
		"still the string\"\"\"",
		"total = 42  # a comment",
		"```",
		"",
		"```ts",
		"const banner = `agent ${name}",
		"  // still the string",
		"done`;",
		"let n = 7;",
		"```",
		"",
		"```sh",
		"echo `date",
		"ls -la",
		"```",
		"",
		"## Checklist",
		"",
	}, "\n")
	for i := 1; i <= 30; i++ {
		answer += fmt.Sprintf("%d. Checklist step %02d of the simulated run.\n", i, i)
	}
	answer = strings.TrimSuffix(answer, "\n")
	command := map[string]any{"type": "commandExecution", "id": "exec-plan6-push",
		"command": `"C:\\windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe" -Command 'git status; git push'`,
		"cwd":     `<probe-root>\probe-work\p7`, "processId": nil, "source": "agent", "status": "inProgress",
		"commandActions":   []any{map[string]any{"type": "unknown", "command": "git status; git push"}},
		"aggregatedOutput": nil, "exitCode": nil, "durationMs": nil}
	done := func() map[string]any {
		c := map[string]any{}
		for k, v := range command {
			c[k] = v
		}
		c["status"], c["exitCode"], c["durationMs"] = "failed", 1, 512
		c["aggregatedOutput"] = "On branch main\r\nnothing to commit, working tree clean\r\nfatal: No configured push destination.\r\n"
		return c
	}
	var out []fixture.Record
	spawned, answered, failed := 0, 0, 0
	for _, r := range recs {
		msg := string(r.Msg)
		switch {
		case r.Dir == "out" && strings.Contains(msg, `"type":"subAgentActivity"`) && strings.Contains(msg, `"kind":"started"`):
			for _, c := range children {
				if strings.Contains(msg, `"agentThreadId":"`+c.thread+`"`) {
					out = append(out, rec(r.TMs-1, "item/started", spawn(c.call, "inProgress", []string{}, c.prompt, c.model, c.effort)), r,
						rec(r.TMs+1, "item/completed", spawn(c.call, "completed", []string{c.thread}, c.prompt, c.model, c.effort)))
					spawned++
				}
			}
			continue
		case r.Dir == "out" && strings.Contains(msg, `"method":"item/completed"`) && strings.Contains(msg, `"type":"userMessage"`) && strings.Contains(msg, `"threadId":"`+main+`"`):
			msgItem := func(text any) map[string]any {
				return map[string]any{"type": "agentMessage", "id": "msg_plan6_markdown", "text": text, "phase": "commentary", "memoryCitation": nil}
			}
			out = append(out, r, rec(r.TMs+600, "item/started", msgItem("")), rec(r.TMs+900, "item/completed", msgItem(answer)))
			answered++
			continue
		case r.Dir == "out" && strings.Contains(msg, `"method":"item/started"`) && strings.Contains(msg, `"phase":"final_answer"`) && strings.Contains(msg, `"threadId":"`+main+`"`):
			out = append(out, rec(r.TMs-1000, "item/started", command), rec(r.TMs-500, "item/completed", done()))
			failed++
		}
		out = append(out, r)
	}
	if spawned != 2 || answered != 1 || failed != 1 {
		t.Fatalf("P7: %d spawns, %d answers, %d commands put in; want 2, 1, 1", spawned, answered, failed)
	}
	return out
}

// claudeAPIKeyContextRefused is P2 signed in with an API key, which refuses
// get_context_usage: init's apiKeySource names a key (claude-code.md, section
// 4: system/init), the reply to req_2 is the error a CLI without the request
// sends (step 2's finding), and the rate_limit_event lines are gone, an API
// key having no subscription window (reliability.md, Telemetry sources).
func claudeAPIKeyContextRefused(t *testing.T, recs []fixture.Record) []fixture.Record {
	var out []fixture.Record
	keyed, refused := 0, 0
	for _, r := range recs {
		msg := string(r.Msg)
		switch {
		case r.Dir == "out" && strings.Contains(msg, `"type":"rate_limit_event"`):
			continue
		case r.Dir == "out" && strings.Contains(msg, `"request_id":"req_2"`):
			r.Msg = []byte(`{"type":"control_response","response":{"subtype":"error","request_id":"req_2","error":"Unsupported control request subtype: get_context_usage"}}`)
			refused++
		case strings.Contains(msg, `"apiKeySource":"none"`):
			r.Msg = []byte(strings.ReplaceAll(msg, `"apiKeySource":"none"`, `"apiKeySource":"ANTHROPIC_API_KEY"`))
			keyed++
		}
		out = append(out, r)
	}
	if keyed == 0 || refused != 1 {
		t.Fatalf("P2: %d init lines keyed, %d replies refused; want some and 1", keyed, refused)
	}
	return out
}

// codexAPIKey is P7 signed in with an API key: every account/rateLimits/read
// is refused with the message Codex sends then (codex-cli.md, 5-hour quota:
// account_processor.rs:1016-1030), and no account/rateLimits/updated is
// pushed.
func codexAPIKey(t *testing.T, recs []fixture.Record) []fixture.Record {
	var out []fixture.Record
	refused := 0
	for _, r := range recs {
		msg := string(r.Msg)
		switch {
		case r.Dir == "out" && strings.Contains(msg, `"method":"account/rateLimits/updated"`):
			continue
		case r.Dir == "out" && strings.Contains(msg, `"result":{"rateLimits"`):
			var reply struct{ ID json.RawMessage }
			if err := json.Unmarshal(r.Msg, &reply); err != nil {
				t.Fatal(err)
			}
			r.Msg = []byte(`{"id":` + string(reply.ID) + `,"error":{"code":-32600,"message":"chatgpt authentication required to read rate limits"}}`)
			refused++
		}
		out = append(out, r)
	}
	if refused == 0 {
		t.Fatal("P7 has no account/rateLimits/read reply")
	}
	return out
}

// codexNo5hWindow is P7 on a plan with no 5-hour limit (Pro, codex-cli.md,
// 5-hour quota: "primary may be absent or another window"): in every read
// and push, the weekly window takes primary's place and secondary is null.
func codexNo5hWindow(t *testing.T, recs []fixture.Record) []fixture.Record {
	out := append([]fixture.Record(nil), recs...)
	moved := 0
	var walk func(v any)
	walk = func(v any) {
		switch v := v.(type) {
		case map[string]any:
			if p, ok := v["primary"].(map[string]any); ok && p["windowDurationMins"] == 300.0 {
				v["primary"], v["secondary"] = v["secondary"], nil
				moved++
			}
			for _, x := range v {
				walk(x)
			}
		case []any:
			for _, x := range v {
				walk(x)
			}
		}
	}
	for i, r := range out {
		if r.Dir != "out" || !strings.Contains(string(r.Msg), `"windowDurationMins":300`) {
			continue
		}
		var v any
		if err := json.Unmarshal(r.Msg, &v); err != nil {
			t.Fatal(err)
		}
		walk(v)
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		out[i].Msg = b
	}
	if moved == 0 {
		t.Fatal("P7 has no 5-hour window")
	}
	return out
}

func TestMockFiles(t *testing.T) {
	for _, f := range mockFiles {
		t.Run(f.file, func(t *testing.T) {
			recs, err := fixture.Read(filepath.Join("..", "..", "..", "testdata", "probes", f.probe))
			if err != nil {
				t.Fatal(err)
			}
			if f.edit != nil {
				recs = f.edit(t, recs)
			}
			evs := fixture.Flatten(fixture.Replay(f.provider, recs, f.mapper()))
			for len(evs) > 0 && evs[len(evs)-1].Kind == event.SessionEnded {
				evs = evs[:len(evs)-1]
			}
			var buf bytes.Buffer
			if err := mock.Encode(&buf, evs); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join("..", "..", "..", "testdata", "mock", f.file)
			if *update {
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, buf.Bytes()) {
				t.Fatalf("%s differs from the replay of %s; rerun with -update and review the diff", f.file, f.probe)
			}
			timed, err := mock.Load(path)
			if err != nil {
				t.Fatal(err)
			}
			if len(timed) != len(evs) {
				t.Fatalf("%d events decoded, %d encoded", len(timed), len(evs))
			}
			for i, tm := range timed {
				if tm.Event.Origin != event.Mock {
					t.Fatalf("event %d: origin %q, want mock", i, tm.Event.Origin)
				}
				if tm.Event.Kind != evs[i].Kind {
					t.Fatalf("event %d: kind %q, want %q", i, tm.Event.Kind, evs[i].Kind)
				}
				// A marker keeps its block index, thinking count and step name
				// through the file (plan 22).
				if want, ok := evs[i].Payload.(event.Marker); ok && tm.Event.Payload != want {
					t.Fatalf("event %d: marker %+v decoded as %+v", i, want, tm.Event.Payload)
				}
			}
			t.Logf("%d events from %d records", len(evs), len(recs))
		})
	}
}

// A mock session offers no Manual or Auto switch (plan 12, item 2).
func TestMockHasNoApprovalMode(t *testing.T) {
	s, err := (&mock.Provider{}).Start(context.Background(), provider.SessionContext{Mock: true})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close(context.Background())
	if s.Capabilities().AutoApprovals {
		t.Error("the mock offers Auto approvals")
	}
	if s.SetApprovalMode(true) == nil {
		t.Error("the mock accepted a mode")
	}
}
