package mock_test

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
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
}{
	{"claude-subagent-approval.jsonl", "p3b-claude-subagent-approval.jsonl", "claude-code", func() fixture.Mapper { return claude.NewMapper() }},
	{"codex-subagents.jsonl", "p7-codex-subagents.jsonl", "codex", func() fixture.Mapper { return codex.NewMapper() }},
}

func TestMockFiles(t *testing.T) {
	for _, f := range mockFiles {
		t.Run(f.file, func(t *testing.T) {
			recs, err := fixture.Read(filepath.Join("..", "..", "..", "testdata", "probes", f.probe))
			if err != nil {
				t.Fatal(err)
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
