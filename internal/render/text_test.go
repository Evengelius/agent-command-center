package render

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

// TestBoxEdges checks every row of a box is exactly w wide and the top edge
// keeps its corner, with and without a right label, down to the label not
// fitting.
func TestBoxEdges(t *testing.T) {
	st := lipgloss.NewStyle()
	for w := 5; w <= 40; w++ {
		for _, right := range []line{nil, plain("2 ACTIVE", st)} {
			ls := box(plain("C · AGENTS", st), right, []line{plain(strings.Repeat("x", 50), st)}, w, 3, st)
			for i, l := range ls {
				if l.width() != w {
					t.Fatalf("w %d, right %v: row %d is %d wide: %q", w, right != nil, i, l.width(), l.String())
				}
			}
			top := ""
			for _, s := range ls[0] {
				top += s.text
			}
			if !strings.HasSuffix(top, "╮") {
				t.Errorf("w %d, right %v: top edge lost its corner: %q", w, right != nil, top)
			}
			if right != nil && strings.Contains(top, "2 ACTIVE") && strings.Contains(top, "…") {
				t.Errorf("w %d: the right label was cut instead of dropped: %q", w, top)
			}
		}
	}
}
