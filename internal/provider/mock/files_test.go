package mock_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/fixture"
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
	// Synthetic: cases no probe has, each a real recording with only the
	// lines of its case edited to the documented shape (plan 4, step 7).
	{"claude-api-key-context-refused.jsonl", "p2-claude-conversation.jsonl", "claude-code", func() fixture.Mapper { return claude.NewMapper() }, claudeAPIKeyContextRefused},
	{"codex-api-key.jsonl", "p7-codex-subagents.jsonl", "codex", func() fixture.Mapper { return codex.NewMapper() }, codexAPIKey},
	{"codex-no-5h-window.jsonl", "p7-codex-subagents.jsonl", "codex", func() fixture.Mapper { return codex.NewMapper() }, codexNo5hWindow},
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
			}
			t.Logf("%d events from %d records", len(evs), len(recs))
		})
	}
}
