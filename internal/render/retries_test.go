package render

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/fixture"
	"github.com/Evengelius/agent-command-center/internal/provider/codex"
)

// TestAgentRetries is plan 21's item 5.6a (AM2 C7): a Codex child thread's
// `error` notifications with willRetry, through Codex's own mapper, give its
// row in C "↻ 2" in amber after its time, at every width; an agent with
// none shows no "↻".
func TestAgentRetries(t *testing.T) {
	recs := records(t, "p7-codex-subagents.jsonl")
	s, _ := replayed(codex.ProviderID, codex.NewMapper(), recs)
	child := ""
	for _, k := range s.Order {
		if a := s.Agents[k]; a.Kind == "thread" {
			child = k
			break
		}
	}
	if child == "" {
		t.Fatal("P7: no child thread")
	}
	last := recs[len(recs)-1].TMs
	retry := func(i int) fixture.Record {
		return rec(last+int64(i+1), "out", fmt.Sprintf(`{"method":"error","params":{"threadId":%q,"turnId":"t-retry","error":{"message":"stream disconnected before completion"},"willRetry":true}}`, child))
	}
	s, now := replayed(codex.ProviderID, codex.NewMapper(), append(append([]fixture.Record{}, recs...), retry(0), retry(1)))
	a := s.Agents[child]
	if a == nil || a.Retries != 2 {
		t.Fatalf("child %s after two willRetry errors: %+v, want 2 retries", child, a)
	}
	label := a.Label
	for _, size := range [][2]int{{120, 40}, {200, 60}, {80, 24}} {
		ui := UI{Width: size[0], Height: size[1], Now: now, Focus: "B"}
		sc := Draw(s, ui)
		rows := strings.Split(strings.TrimSuffix(texts(sc.lines), "\n"), "\n")
		z := zoneBoxes(rows)
		zc := z["C"]
		found, others := false, false
		for y := zc.top + 1; y < zc.bottom; y++ {
			// From 100 columns the count is on the agent's second row, under
			// its label; under 100 on its one row.
			// The label may end in "…" in the list's column (plan 27): its
			// first 12 cells name the row.
			head := label[:min(len(label), 12)]
			own := strings.Contains(rows[y], head) || (size[0] >= 100 && y > zc.top+1 && strings.Contains(rows[y-1], head))
			has := strings.Contains(rows[y], "↻")
			switch {
			case own && has:
				found = true
				if !strings.Contains(rows[y], "↻ 2") {
					t.Errorf("%dx%d: %q has no \"↻ 2\"", size[0], size[1], rows[y])
				}
				// After the time: the count follows the elapsed time.
				if i, j := strings.Index(rows[y], ":"), strings.Index(rows[y], "↻"); size[0] >= 100 && (i < 0 || i > j) {
					t.Errorf("%dx%d: \"↻ 2\" not after the time: %q", size[0], size[1], rows[y])
				}
			case has:
				others = true
			}
		}
		if !found {
			t.Errorf("%dx%d: no \"↻ 2\" on %s's row in C:\n%s", size[0], size[1], label, strings.Join(rows[zc.top:zc.bottom+1], "\n"))
		}
		if others {
			t.Errorf("%dx%d: a \"↻\" on an agent with no retry:\n%s", size[0], size[1], strings.Join(rows[zc.top:zc.bottom+1], "\n"))
		}
		// In amber.
		var amber bool
		for _, l := range sc.lines {
			for _, sp := range l {
				if strings.Contains(sp.text, "↻ 2") && sp.st == stProgress {
					amber = true
				}
			}
		}
		if !amber {
			t.Errorf("%dx%d: \"↻ 2\" is not amber (stProgress)", size[0], size[1])
		}
	}

	// No retry, no "↻": the same replay without the two errors.
	s0, now0 := replayed(codex.ProviderID, codex.NewMapper(), recs)
	for _, size := range [][2]int{{120, 40}, {200, 60}, {80, 24}} {
		if f := texts(Draw(s0, UI{Width: size[0], Height: size[1], Now: now0, Focus: "B"}).lines); strings.Contains(f, "↻") {
			t.Errorf("%dx%d: \"↻\" with no retry", size[0], size[1])
		}
	}
}
