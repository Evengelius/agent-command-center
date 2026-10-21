package mock_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/fixture"
	"github.com/Evengelius/agent-command-center/internal/provider/codex"
	"github.com/Evengelius/agent-command-center/internal/provider/mock"
)

// A reasoning item's summary goes through a mock file and back, so a mock
// made from a recording with summaries replays them (plan 22, amendment 20,
// criterion 20.12: a mock of P22s, made in a scratch copy and not kept); an
// item without one writes no field, so the mock files written before stay as
// they are.
func TestMarkerSummaryInAMockFile(t *testing.T) {
	recs, err := fixture.Read("../../../testdata/probes/p22s-codex-reasoning-summary.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	evs := fixture.Flatten(fixture.Replay(codex.ProviderID, recs, codex.NewMapper()))
	var buf bytes.Buffer
	if err := mock.Encode(&buf, evs); err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(buf.String(), `"Summary"`); n != 1 || !strings.Contains(buf.String(), `"Summary":"**Computing f(10)**"`) {
		t.Errorf("%d Summary fields in the file, want the one of the second item", n)
	}
	timed, err := mock.Decode(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if len(timed) != len(evs) {
		t.Fatalf("%d events decoded, %d encoded", len(timed), len(evs))
	}
	got := 0
	for i, tm := range timed {
		want, ok := evs[i].Payload.(event.Marker)
		if !ok {
			continue
		}
		if tm.Event.Payload != want {
			t.Errorf("event %d: marker %+v decoded as %+v", i, want, tm.Event.Payload)
		}
		if p, ok := tm.Event.Payload.(event.Marker); ok && p.Summary != "" {
			got++
			if p.Name != "item_completed" || p.Summary != "**Computing f(10)**" || tm.After != evs[i].At.Sub(evs[0].At) {
				t.Errorf("event %d: %+v after %v", i, p, tm.After)
			}
		}
	}
	if got != 1 {
		t.Errorf("%d decoded markers carry a summary, want 1", got)
	}
}
