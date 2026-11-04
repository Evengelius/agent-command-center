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
	"time"

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
	// Plan 38, design D6: P34x's first run up to its refused
	// thread/settings/update, a Codex without the experimental API, which
	// takes every change with the next message.
	{"codex-models-next-turn.jsonl", "p34x-codex-model-list-settings.jsonl", "codex", func() fixture.Mapper { return codex.NewMapper() }, codexModelsNextTurn},
	// Plan 72: the approved frames' commands for the owner's Human Evaluation,
	// synthetic, on P12x's auto-review (Codex: a short command, its recorded
	// one reviewed, the long one asked for and allowed, then a long one
	// waiting for the owner) and on P2 (Claude Code: a failed go test, an
	// update denied, its recorded Write allowed).
	{"codex-commands.jsonl", "p12x-codex-auto-review.jsonl", "codex", func() fixture.Mapper { return codex.NewMapper() }, codexCommands},
	{"claude-commands.jsonl", "p2-claude-conversation.jsonl", "claude-code", func() fixture.Mapper { return claude.NewMapper() }, claudeCommands},
	// Plan 76, criterion 76.16: P76a, Claude Code's native agents, an
	// implementer and a reviewer run in turn by the orchestrator; the process
	// "p76a" alone, the driver's own lines left out.
	{"claude-native-agents.jsonl", "p76a-claude-native-agents.jsonl", "claude-code", func() fixture.Mapper { return claude.NewMapper() }, func(_ *testing.T, recs []fixture.Record) []fixture.Record { return fixture.Of(recs, "p76a") }},
}

// p72Launcher is the launcher Codex runs a command with, as its command
// field writes it.
const p72Launcher = `"C:\\windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe"`

// p72Invocation is script in the launcher's -Command, single-quoted,
// POSIX's way (plan 72, V1).
func p72Invocation(script string) string {
	return p72Launcher + " -Command '" + strings.ReplaceAll(script, "'", `'"'"'`) + "'"
}

// The frames' long commands (plan.md, Plan 72, Reference frames).
const (
	p72Todo  = `$files = Get-ChildItem -Recurse -Filter *.go -File; $todos = $files | Select-String -Pattern 'TODO|FIXME' | Sort-Object Path, LineNumber; $todos | ForEach-Object { "- $($_.Path):$($_.LineNumber) $($_.Line.Trim())" } | Set-Content -LiteralPath 'TODO.md' -Encoding utf8; '{0} items in {1} files' -f $todos.Count, ($todos | Select-Object -Unique Path).Count`
	p72Count = `$counts = Get-ChildItem -Recurse -Filter *.go -File | Select-String -Pattern 'TODO' | Group-Object Path | Sort-Object Count -Descending; $counts | ForEach-Object { '| {0} | {1} |' -f $_.Name, $_.Count } | Set-Content -LiteralPath 'TODO-COUNT.md' -Encoding utf8; Write-Output ('{0} files' -f $counts.Count)`
)

// codexCommands is P12x with plan 72's scenes, synthetic: in its first turn,
// a short command before the recorded one, which Codex's auto-review
// approved, and the frames' long command after it, asked for, allowed and
// completed; then a third turn whose long command waits for the owner, the
// session left open.
func codexCommands(t *testing.T, recs []fixture.Record) []fixture.Record {
	const main, turn1, turn3 = "01a14c29-ea47-7f93-892c-2b9ef96177a8", "01a14c29-eecc-7072-82fb-0315a73d6db5", "01a14c2b-0072-7072-8072-000000000072"
	const cwd = `<HOME>\AppData\Local\Temp\acc-p12\x`
	rec := func(ms int64, dir string, v any) fixture.Record {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		return fixture.Record{TMs: ms, Dir: dir, Msg: b}
	}
	note := func(ms int64, method string, params map[string]any) fixture.Record {
		return rec(ms, "out", map[string]any{"method": method, "params": params})
	}
	item := func(id, script, turn, status string, out any, exit any) map[string]any {
		return map[string]any{"type": "commandExecution", "id": id, "command": p72Invocation(script), "cwd": cwd, "processId": nil, "source": "agent", "status": status,
			"commandActions": []any{map[string]any{"type": "unknown", "command": script}}, "aggregatedOutput": out, "exitCode": exit, "durationMs": nil}
	}
	at := func(it map[string]any, turn string) map[string]any {
		return map[string]any{"item": it, "threadId": main, "turnId": turn}
	}
	ask := func(ms int64, id int, itemID, script, turn, reason string) fixture.Record {
		return rec(ms, "out", map[string]any{"method": "item/commandExecution/requestApproval", "id": id, "params": map[string]any{
			"kind": "command", "threadId": main, "turnId": turn, "itemId": itemID, "environmentId": "local", "reason": reason,
			"command": p72Invocation(script), "cwd": cwd, "commandActions": []any{map[string]any{"type": "unknown", "command": script}},
			"availableDecisions": []any{"accept", "cancel"}}})
	}
	var out []fixture.Record
	short, long, waiting := 0, 0, 0
	for _, r := range recs {
		msg := string(r.Msg)
		switch {
		case r.Dir == "out" && strings.Contains(msg, `"method":"item/started"`) && strings.Contains(msg, `"id":"exec-c454af45-5812-423d-a422-84061353273c"`):
			out = append(out, note(r.TMs-400, "item/started", at(item("exec-p72-status", "git status --short", turn1, "inProgress", nil, nil), turn1)),
				note(r.TMs-200, "item/completed", at(item("exec-p72-status", "git status --short", turn1, "completed", " M main.go\r\n", 0), turn1)))
			short++
		case r.Dir == "out" && strings.Contains(msg, `"method":"item/started"`) && strings.Contains(msg, `"phase":"final_answer"`) && strings.Contains(msg, `"turnId":"`+turn1+`"`):
			out = append(out, note(r.TMs-900, "item/started", at(item("exec-p72-todo", p72Todo, turn1, "inProgress", nil, nil), turn1)),
				ask(r.TMs-850, 9001, "exec-p72-todo", p72Todo, turn1, "Create TODO.md from the TODO and FIXME comments of the Go files"),
				rec(r.TMs-600, "in", map[string]any{"id": 9001, "result": map[string]any{"decision": "accept"}}),
				note(r.TMs-300, "item/completed", at(item("exec-p72-todo", p72Todo, turn1, "completed", "14 items in 9 files\r\n", 0), turn1)))
			long++
		case r.Dir == "exit":
			// A third turn, its long command waiting for the owner; the session
			// stays open, as each mock's does.
			text := "Count the TODO comments of each Go file into TODO-COUNT.md."
			out = append(out, rec(r.TMs-40, "in", map[string]any{"id": 9100, "method": "turn/start", "params": map[string]any{"threadId": main, "input": []any{map[string]any{"type": "text", "text": text}}}}),
				note(r.TMs-35, "turn/started", map[string]any{"threadId": main, "turn": map[string]any{"id": turn3, "items": []any{}, "status": "inProgress", "error": nil}}),
				note(r.TMs-30, "item/started", at(map[string]any{"type": "userMessage", "id": "user-p72-count", "content": []any{map[string]any{"type": "text", "text": text}}}, turn3)),
				note(r.TMs-25, "item/completed", at(map[string]any{"type": "userMessage", "id": "user-p72-count", "content": []any{map[string]any{"type": "text", "text": text}}}, turn3)),
				note(r.TMs-20, "item/started", at(item("exec-p72-count", p72Count, turn3, "inProgress", nil, nil), turn3)),
				ask(r.TMs-15, 9002, "exec-p72-count", p72Count, turn3, "Write TODO-COUNT.md, a table of the TODO comments of each Go file"))
			waiting++
		}
		out = append(out, r)
	}
	if short != 1 || long != 1 || waiting != 1 {
		t.Fatalf("P12x: %d short, %d long, %d waiting commands put in; want 1 each", short, long, waiting)
	}
	return out
}

// claudeCommands is P2 with plan 72's Claude Code frame, synthetic: before
// its recorded Write, a go test that fails, then an update the owner denies,
// which Claude Code reports failed after the denial.
func claudeCommands(t *testing.T, recs []fixture.Record) []fixture.Record {
	const session = "00ef0150-56f0-4033-8467-bd6e93e296bb"
	rec := func(ms int64, dir string, v any) fixture.Record {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		return fixture.Record{TMs: ms, Dir: dir, Msg: b}
	}
	call := func(ms int64, id string, in map[string]string) fixture.Record {
		return rec(ms, "out", map[string]any{"type": "assistant", "message": map[string]any{"model": "claude-haiku-5-5", "id": "msg_p72_" + id, "type": "message", "role": "assistant",
			"content": []any{map[string]any{"type": "tool_use", "id": id, "name": "Bash", "input": in}}, "stop_reason": nil}, "parent_tool_use_id": nil, "session_id": session})
	}
	result := func(ms int64, id, content string) fixture.Record {
		return rec(ms, "out", map[string]any{"type": "user", "message": map[string]any{"role": "user", "content": []any{map[string]any{"tool_use_id": id, "type": "tool_result", "content": content, "is_error": true}}},
			"parent_tool_use_id": nil, "session_id": session})
	}
	test := map[string]string{"command": "go test ./...", "description": "Run the tests"}
	update := map[string]string{"command": "git stash && git pull --rebase origin main && go test ./...", "description": "Update from origin and test again"}
	var out []fixture.Record
	put := 0
	for _, r := range recs {
		if r.Dir == "out" && strings.Contains(string(r.Msg), `"name":"Write"`) && strings.Contains(string(r.Msg), `"type":"assistant"`) {
			out = append(out, call(r.TMs-300, "toolu_p72_test", test),
				result(r.TMs-250, "toolu_p72_test", "--- FAIL: TestMarkdownCacheBytes (0.65s)\nFAIL\nFAIL\tgithub.com/x/render\t1.2s"),
				call(r.TMs-200, "toolu_p72_update", update),
				rec(r.TMs-150, "out", map[string]any{"type": "control_request", "request_id": "p72-update", "request": map[string]any{"subtype": "can_use_tool", "tool_name": "Bash", "display_name": "Bash",
					"input": update, "description": update["description"], "tool_use_id": "toolu_p72_update"}}),
				rec(r.TMs-100, "in", map[string]any{"type": "control_response", "response": map[string]any{"subtype": "success", "request_id": "p72-update",
					"response": map[string]any{"behavior": "deny", "message": "The owner denied this tool use."}}}),
				result(r.TMs-50, "toolu_p72_update", "The user doesn't want to proceed with this tool use. The tool use was rejected (eg. if it was a file edit, the new_string was NOT written to the file). STOP what you are doing and wait for the user to tell you how to proceed."))
			put++
		}
		out = append(out, r)
	}
	if put != 1 {
		t.Fatalf("P2: the commands put in %d times, want once", put)
	}
	return out
}

// codexModelsNextTurn is P34x's first run up to the refusal of its
// thread/settings/update, the thread/read written before it left out, as
// codexModels leaves account and remote-control lines out.
func codexModelsNextTurn(t *testing.T, recs []fixture.Record) []fixture.Record {
	var out []fixture.Record
	for _, r := range recs {
		msg := string(r.Msg)
		if r.Dir == "run" && len(out) == 0 {
			r.Dir, r.Msg = "argv", []byte(`["app-server"]`)
			out = append(out, r)
			continue
		}
		if r.Dir != "in" && r.Dir != "out" || strings.Contains(msg, `"method":"account/`) || strings.Contains(msg, `"method":"remoteControl/`) ||
			strings.Contains(msg, `"method":"thread/read"`) || strings.HasPrefix(msg, `{"id":12,"result"`) {
			continue
		}
		out = append(out, r)
		if strings.Contains(msg, "requires experimentalApi capability") {
			return out
		}
	}
	t.Fatal("P34x: no refused thread/settings/update")
	return nil
}

// TestMockCodexChanges is plan 38's design D6: a Codex file answers a change
// at once, as Codex with thread/settings/update; one whose events show the
// update refused takes it with the next message, as plan 34's Codex.
func TestMockCodexChanges(t *testing.T) {
	for _, c := range []struct {
		file     string
		nextTurn bool
	}{{"codex-models.jsonl", false}, {"codex-models-next-turn.jsonl", true}} {
		t.Run(c.file, func(t *testing.T) {
			timed, err := mock.Load(filepath.Join("..", "..", "..", "testdata", "mock", c.file))
			if err != nil {
				t.Fatal(err)
			}
			s, err := (&mock.Provider{Events: timed}).Start(context.Background(), provider.SessionContext{Mock: true})
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close(context.Background())
			for range timed {
				<-s.Events()
			}
			if got := s.Capabilities().ModelsNextTurn; got != c.nextTurn {
				t.Errorf("ModelsNextTurn %v, want %v", got, c.nextTurn)
			}
			if err := s.SetModelSettings(provider.ModelRequest{Effort: "low"}); err != nil {
				t.Fatal(err)
			}
			answered := func() (event.Settings, bool) {
				for {
					select {
					case ev := <-s.Events():
						if p, ok := ev.Payload.(event.Settings); ok && p.Answers {
							return p, true
						}
					default:
						return event.Settings{}, false
					}
				}
			}
			p, ok := answered()
			if ok == c.nextTurn || ok && (p.Effort != "low" || p.Model != "gpt-6-sol") {
				t.Errorf("after the change: answered %v %+v", ok, p)
			}
			if c.nextTurn {
				if err := s.Send(provider.Message{Text: "hello"}); err != nil {
					t.Fatal(err)
				}
				if p, ok := answered(); !ok || p.Effort != "low" {
					t.Errorf("after the message: answered %v %+v", ok, p)
				}
			}
		})
	}
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

// TestMockModes is plan 39's design D10 and criterion 39.12: a mock session
// takes its file's CLI's modes, each answered at once, and refuses the
// other CLI's; one whose CLI is not known yet takes none (plan 12, item 2).
func TestMockModes(t *testing.T) {
	s, err := (&mock.Provider{}).Start(context.Background(), provider.SessionContext{Mock: true})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close(context.Background())
	if s.Capabilities().Modes || s.SetApprovalMode(event.ModeAuto) == nil {
		t.Error("a mock session with no CLI took a mode")
	}
	for _, c := range []struct {
		file      string
		modes, no []string
	}{
		{"claude-models.jsonl", []string{event.ModeAcceptEdits, event.ModePlan, event.ModeAuto, event.ModeManual}, []string{event.ModeFull, event.ModeCustom}},
		{"codex-models.jsonl", []string{event.ModeAuto, event.ModeFull, event.ModeManual}, []string{event.ModeAcceptEdits, event.ModePlan, event.ModeCustom}},
	} {
		t.Run(c.file, func(t *testing.T) {
			timed, err := mock.Load(filepath.Join("..", "..", "..", "testdata", "mock", c.file))
			if err != nil {
				t.Fatal(err)
			}
			s, err := (&mock.Provider{Events: timed}).Start(context.Background(), provider.SessionContext{Mock: true})
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close(context.Background())
			for range timed {
				<-s.Events()
			}
			if !s.Capabilities().Modes {
				t.Fatal("no modes")
			}
			for _, m := range c.modes {
				if err := s.SetApprovalMode(m); err != nil {
					t.Fatal(err)
				}
				select {
				case ev := <-s.Events():
					if p, ok := ev.Payload.(event.Mode); !ok || ev.Kind != event.ApprovalMode || p.Mode != m || !p.Answers {
						t.Errorf("%s answered with %s %+v", m, ev.Kind, ev.Payload)
					}
				case <-time.After(5 * time.Second):
					t.Fatalf("%s not answered", m)
				}
			}
			for _, m := range c.no {
				if s.SetApprovalMode(m) == nil {
					t.Errorf("took %q", m)
				}
			}
		})
	}
}
