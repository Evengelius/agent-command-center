package render

import (
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/provider/claude"
)

// TestZoneCFloor is plan 21's item 4: from 160 columns, B's 14 rows left
// zone C 3 rows at 24 to 26 rows, too few for an agent's two, and C listed
// none. With an agent in play C now keeps one agent's rows, A and B giving
// them; from 27 rows the heights are those of plan 20.
func TestZoneCFloor(t *testing.T) {
	one := probeState(t, "p3b-claude-subagent-approval.jsonl", claude.ProviderID, claude.NewMapper())
	if n := one.AgentsInPlay(); n != 1 {
		t.Fatalf("P3b: %d agents in play, want 1", n)
	}
	for _, c := range []struct{ h, top, c int }{{24, 11, 4}, {25, 12, 4}, {26, 13, 4}, {27, 14, 4}} {
		rows := strings.Split(strings.TrimSuffix(texts(Draw(one, UI{Width: 160, Height: c.h, Focus: "B"}).lines), "\n"), "\n")
		if len(rows) != c.h {
			t.Fatalf("160x%d: %d rows", c.h, len(rows))
		}
		for i, r := range rows {
			if w := textWidth(r); w > 160 {
				t.Errorf("160x%d: row %d is %d cells wide", c.h, i, w)
			}
		}
		z := zoneBoxes(rows)
		a, b, zc := z["A"], z["B"], z["C"]
		if a.bottom != b.bottom || a.top != b.top {
			t.Errorf("160x%d: A rows %d-%d, B rows %d-%d: not one upper row", c.h, a.top, a.bottom, b.top, b.bottom)
		}
		if top, ch := b.bottom-b.top+1, zc.bottom-zc.top+1; top != c.top || ch != c.c {
			t.Errorf("160x%d: A and B %d rows, C %d; want %d and %d", c.h, top, ch, c.top, c.c)
		}
		// C's first row inside its border: the agent's label in the list,
		// left of the rule, the preview's heading right of it.
		first := rows[zc.top+1]
		rule := strings.Index(first, " │ ")
		label := strings.Index(first, "general-purpose")
		if rule < 0 || label < 0 || label > rule || !strings.Contains(first[rule:], "general-purpose · LIVE") {
			t.Errorf("160x%d: C's first row lists no agent left of the preview:\n%s", c.h, first)
		}
	}
}
