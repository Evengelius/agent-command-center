package app

import (
	"fmt"
	"strings"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/render"
)

// TestCommitImagesDrawn is criterion 45.2's images through the application
// (design D5): with Sixel, the frame's raw sequence draws the button's
// image at its first cell, the message box's four strips around its text
// cells and, once open, the dropdown's image under the button, each over
// its cells written first; the label is GDI's; a look that changes (the
// item hot) draws the image again; closed, the dropdown's cells are written
// back, its image gone.
func TestCommitImagesDrawn(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := commitRig(t)
		h.send(PrimaryDeviceAttributesEvent{61, 4, 6, 7, 14})
		sc := screenNow(h)
		cm, mb := sc.Commit, sc.MsgBox
		if !cm.Shown || !mb.Shown || sc.Menu.Shown {
			t.Fatalf("shown: button %v, box %v, dropdown %v", cm.Shown, mb.Shown, sc.Menu.Shown)
		}
		label, err := h.m.textLabel(cm.Look.Label, render.LabelTop, cm.Look.Live)
		if err != nil {
			t.Skipf("GDI: %v", err) // no Cascadia Mono or Consolas: the text form is the golden frames'
		}
		seq := h.m.thumbSequence()
		at := func(x, y int) string { return fmt.Sprintf("\x1b[%d;%dH", y+1, x+1) }
		if !strings.Contains(seq, at(cm.X, cm.Y)+render.CommitSixel(cm.Look, label)) {
			t.Error("the button's image not drawn at its first cell")
		}
		strips := render.MsgSixels(mb.Look)
		for i, p := range [][2]int{{mb.X, mb.Y}, {mb.X, mb.Y + mb.Look.Rows + 1}, {mb.X, mb.Y + 1}, {mb.X + mb.Look.Cells - 1, mb.Y + 1}} {
			if !strings.Contains(seq, at(p[0], p[1])+strips[i]) {
				t.Errorf("strip %d not drawn at %v", i, p)
			}
		}
		if !strings.HasPrefix(seq, "\x1b[?2026h\x1b7") || !strings.HasSuffix(seq, "\x1b8\x1b[?2026l") {
			t.Error("the sequence not inside synchronized output, the cursor saved and restored")
		}
		clickItem(t, h, render.ItemCommitMore, 0, 0)
		sc = screenNow(h)
		mn := sc.Menu
		ml, err := h.m.textLabel(render.MenuLabel, render.MenuTextTop, false)
		if !mn.Shown || err != nil || mn.X != cm.X || mn.Y != cm.Y+render.BtnRows || mn.Look.Cells != cm.Look.Cells {
			t.Fatalf("the dropdown: shown %v at %d,%d, %d cells; the button at %d,%d, %d cells", mn.Shown, mn.X, mn.Y, mn.Look.Cells, cm.X, cm.Y, cm.Look.Cells)
		}
		if seq := h.m.thumbSequence(); !strings.Contains(seq, at(mn.X, mn.Y)+render.MenuSixel(mn.Look, ml)) {
			t.Error("the dropdown's image not drawn under the button")
		}
		mouseAt(t, h, moveTo(mn.X+4, mn.Y+1))
		sc = screenNow(h)
		if !sc.Menu.Look.Hot || h.m.previewSignature() == "" {
			t.Fatal("the item hot: not in the look")
		}
		if seq := h.m.thumbSequence(); !strings.Contains(seq, render.MenuSixel(sc.Menu.Look, ml)) {
			t.Error("the hot item's image not drawn")
		}
		h.send(keyOf(tea.KeyEscape))
		sc = screenNow(h)
		seq = h.m.thumbSequence()
		if sc.Menu.Shown || !strings.Contains(seq, at(mn.X, mn.Y)+sc.Region(mn.X, mn.Y, mn.Look.Cells)) {
			t.Error("closed: the dropdown's cells not written back")
		}
	})
}

// TestCommitLiveLabel is design D15's labels: a running operation's, its
// seconds changing every second, kept only while it shows, the same mask
// while it does; the controls' other labels kept.
func TestCommitLiveLabel(t *testing.T) {
	m := &Model{}
	a, err := m.textLabel("Pushing… 2 s", render.LabelTop, true)
	if err != nil {
		t.Skipf("GDI: %v", err) // no Cascadia Mono or Consolas
	}
	if b, _ := m.textLabel("Pushing… 2 s", render.LabelTop, true); b != a {
		t.Error("the label showing drawn again")
	}
	for i := 3; i < 23; i++ {
		if _, err := m.textLabel(fmt.Sprintf("Pushing… %d s", i), render.LabelTop, true); err != nil {
			t.Fatal(err)
		}
	}
	if len(m.pillLabels) != 0 || m.liveLabel.key != "Pushing… 22 s\x00"+fmt.Sprint(render.LabelTop) {
		t.Errorf("%d live labels kept; showing %q", len(m.pillLabels), m.liveLabel.key)
	}
	c, _ := m.textLabel("Commit", render.LabelTop, false)
	if d, _ := m.textLabel("Commit", render.LabelTop, false); d != c || len(m.pillLabels) != 1 {
		t.Errorf("a control's label not kept: %d kept", len(m.pillLabels))
	}
}
