package render

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

// TestPlusMenu is plan 30's criterion 30.1 (brief amendment 30, the owner's
// image): the menu over zone B above the editor's rule, its item, the
// band on the active ground while it is open, the hover, the panels the
// mouse finds, and the rest of the frame untouched.
func TestPlusMenu(t *testing.T) {
	defer UsePalette(pickTrue)
	s := quiet(t)
	for _, size := range [][2]int{{80, 24}, {120, 40}, {208, 59}} {
		base := UI{Width: size[0], Height: size[1], Now: appClock, Focus: "B"}
		closed := Draw(s, base)
		px, py, ok := closed.find('+')
		if !ok {
			t.Fatalf("%v: no band on the closed frame", size)
		}
		if closed.PanelAt(px+1, py) != '+' {
			t.Errorf("%v: the band answers %q, want '+'", size, closed.PanelAt(px+1, py))
		}
		if _, _, found := closed.find(menuPanel); found || strings.Contains(closed.String(), "Upload from computer") {
			t.Errorf("%v: the menu drawn while closed", size)
		}
		ry := closed.editorRule(py)
		if ry < 2 || ry >= py-2 {
			t.Fatalf("%v: the editor's rule on row %d for the band on row %d", size, ry, py)
		}
		for _, hover := range []bool{false, true} {
			ui := base
			ui.Plus, ui.PlusHover = true, hover
			open := Draw(s, ui)
			rows := strings.Split(strings.TrimSuffix(texts(open.lines), "\n"), "\n")
			box := func(y int) string { return string([]rune(rows[y])[px : px+menuInner+2]) }
			if top, item, bottom := box(ry-2), box(ry-1), box(ry); top != "╭"+strings.Repeat("─", menuInner)+"╮" || item != "│  🗀  Upload from computer │" || bottom != "╰"+strings.Repeat("─", menuInner)+"╯" {
				t.Errorf("%v, hover %v: the menu's rows %q, %q, %q", size, hover, top, item, bottom)
			}
			// The band on the active ground, the "+" in the lavender-white.
			if x, ok := spanFrom(open.lines[py], px, " + "); !ok || x.st != active(stThinkEdge) {
				t.Errorf("%v, hover %v: the band styled %v, want the active ground", size, hover, x.st)
			}
			// The item: its text in the lavender-white, on the active ground
			// under the pointer and on none otherwise; the box in stFrame.
			want := stThinkEdge
			if hover {
				want = active(stThinkEdge)
			}
			if x, ok := spanFrom(open.lines[ry-1], px, "Upload from computer"); !ok || x.st != want {
				t.Errorf("%v, hover %v: the item styled %v, want %v", size, hover, x.st, want)
			}
			if x, ok := spanFrom(open.lines[ry-2], px, "╭"); !ok || x.st != stFrame {
				t.Errorf("%v, hover %v: the box styled %v, want stFrame", size, hover, x.st)
			}
			for _, c := range []struct{ x, y int }{{px, ry - 2}, {px + 1, ry - 1}, {px + 10, ry - 1}, {px + menuInner + 1, ry}} {
				if p := open.PanelAt(c.x, c.y); p != menuPanel {
					t.Errorf("%v, hover %v: cell (%d, %d) answers %q, want 'U'", size, hover, c.x, c.y, p)
				}
			}
			// Every other cell equal between open and closed.
			crows := strings.Split(strings.TrimSuffix(texts(closed.lines), "\n"), "\n")
			for y := range rows {
				a, b := []rune(rows[y]), []rune(crows[y])
				for x := range a {
					inBox := y >= ry-2 && y <= ry && x >= px && x < px+menuInner+2
					inBand := y == py && x >= px && x < px+3
					if !inBox && !inBand && a[x] != b[x] {
						t.Errorf("%v, hover %v: cell (%d, %d) %q differs from the closed frame's %q", size, hover, x, y, string(a[x]), string(b[x]))
						break
					}
				}
			}
			if textWidth("🗀") != 1 {
				t.Errorf("the folder icon is %d cells wide, want 1", textWidth("🗀"))
			}
		}
	}
	// On 16 colours the active ground is not drawn, as zone B's grounds.
	UsePalette(pickANSI)
	ui := UI{Width: 120, Height: 40, Now: appClock, Focus: "B", Plus: true, PlusHover: true}
	open := Draw(s, ui)
	px, py, _ := open.find('+')
	if x, ok := spanFrom(open.lines[py], px, " + "); !ok || x.st != stThinkEdge {
		t.Errorf("16 colours: the band styled %v, want stThinkEdge alone", x.st)
	}
	var _ *lipgloss.Style = stThinkEdge
}
