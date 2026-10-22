package render

import (
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/provider/claude"
)

// TestZoneCFloor is plan 21's item 4: from 160 columns, B's 14 rows left
// zone C 3 rows at 24 to 26 rows, too few for an agent's two, and C listed
// none. With an agent in play C keeps one agent's rows, A and B giving
// them. Since plan 27 (design D4) those rows are 7, one agent's two with
// the three heading rows and the edges: C has 7 at 24 to 26 rows and 8 at
// 27, a third of the rows the zones share.
func TestZoneCFloor(t *testing.T) {
	one := probeState(t, "p3b-claude-subagent-approval.jsonl", claude.ProviderID, claude.NewMapper())
	if n := one.AgentsInPlay(); n != 1 {
		t.Fatalf("P3b: %d agents in play, want 1", n)
	}
	for _, c := range []struct{ h, top, c int }{{24, 14, 7}, {25, 15, 7}, {26, 16, 7}, {27, 16, 8}} {
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
		// C's rows inside its border: the header, its rule, the panels'
		// headings, then the agent's label in the list left of the vertical
		// rule and the timeline's first row right of it (plan 27).
		head, first := rows[zc.top+3], rows[zc.top+4]
		rule := strings.Index(first, " │ ")
		label := strings.Index(first, "general-purpose")
		if rule < 0 || label < 0 || label > rule || !strings.Contains(head, "AGENTS") || !strings.Contains(head[strings.Index(head, " │ "):], "EXECUTION TIMELINE") {
			t.Errorf("160x%d: C's rows list no agent left of the timeline:\n%s\n%s", c.h, head, first)
		}
	}
}
