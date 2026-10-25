package render

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"testing"
)

// Plan 45 (brief amendment 46): the commit controls as the frame draws
// them, in text and as images.

// TestCommitControlsText is criteria 45.1 and 45.2 in text (designs D1 to
// D5, D12): the message box, the button and the note, each exactly inner
// cells wide, in the image's colours; the placeholder, the cursor, the
// label centred, the separator, the chevron; dimmed with nothing to do, the
// focus's outline; the dropdown's band while hot; the question's words and
// choices.
func TestCommitControlsText(t *testing.T) {
	const inner = 26
	v := SourceView{Open: true, Repo: "demo", Branch: "main", Action: ActionCommit}
	ui := UI{Width: 200, Height: 60, Source: v}
	ls := commitControls(ui, inner)
	if len(ls) != 6 {
		t.Fatalf("%d rows, want the box's 3 and the button's 3:\n%s", len(ls), texts(ls))
	}
	want := []string{
		"╭────────────────────────╮",
		"│Message                 │",
		"╰────────────────────────╯",
		"╭────────────────────────╮",
		"│      ✓ Commit      │ ⌄ │",
		"╰────────────────────────╯",
	}
	for i, l := range ls {
		if got := plainOf(l); got != want[i] || l.width() != inner {
			t.Errorf("row %d: %q (%d wide), want %q", i, got, l.width(), want[i])
		}
	}
	if x, _ := spanOf(ls[0], "╭"); x.st != stMsgOutline {
		t.Error("the message box's outline is not its colour")
	}
	if x, _ := spanOf(ls[1], "Message"); x.st != msgGround.on(stMsgHolder) || x.ref.item != ItemMsgHolder {
		t.Error("the placeholder is not in its colour on the box's inside")
	}
	if x, _ := spanOf(ls[4], "✓ Commit"); x.st != btnGround.on(stBtnLabel) || x.ref.item != ItemCommitMain {
		t.Error("the label is not on the fill in its colour, in the left part")
	}
	if x, _ := spanOf(ls[3], "╭"); x.st != stBtnOutline {
		t.Error("the button's outline is not its colour")
	}
	if l := ls[4]; l[2].text != "│" || l[2].st != btnGround.on(stBtnSep) || l[3].text != " ⌄ " || l[3].ref.item != ItemCommitMore {
		t.Errorf("the separator and the chevron: %q %q", l[2].text, l[3].text)
	}
	for _, c := range []struct {
		name string
		v    SourceView
		want string
	}{
		{"push", SourceView{Action: ActionPush}, "│        Push        │ ⌄ │"},
		{"busy", SourceView{Action: ActionCommit, Busy: "Committing…"}, "│    Committing…     │ ⌄ │"},
	} {
		c.v.Open, c.v.Repo = true, "demo"
		if got := plainOf(commitButton(c.v, inner, false)[1]); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
	dim := commitButton(SourceView{Repo: "demo", Dim: true}, inner, false)
	if x, _ := spanOf(dim[1], "✓ Commit"); x.st != btnDimGround.on(stBtnLabelDim) || dim[0][0].st != stBtnOutlineDim {
		t.Error("nothing to do: the button not dimmed")
	}
	focus := commitButton(SourceView{Repo: "demo", Place: PlaceBtn}, inner, false)
	if focus[0][0].st != stBtnFocus {
		t.Error("the keyboard's focus not on the outline")
	}
	box := msgBox(SourceView{Repo: "demo", Place: PlaceMsg}, inner, false)
	if box[0][0].st != stMsgFocus || box[1][1].st != stCursor {
		t.Error("the focused field: no focus outline or no cursor before the placeholder")
	}
	sixel := commitControls(UI{Width: 200, Height: 60, Sixel: true, Source: v}, inner)
	for i, l := range sixel {
		if got := strings.TrimSpace(plainOf(l)); got != "" && i != 1 {
			t.Errorf("with Sixel, row %d holds %q, not blank cells", i, got)
		}
	}
	for _, hot := range []bool{false, true} {
		m := menuLines(SourceView{MenuHot: hot}, inner, false)
		g := menuGround
		if hot {
			g = menuBandGround
		}
		if plainOf(m[1]) != "│  Commit & Push         │" || m[1][1].st != g.on(stMenuText) || m[0][0].st != stMenuOutline {
			t.Errorf("hot %v: %q", hot, plainOf(m[1]))
		}
	}
	for kind, st := range map[byte]any{NoticeCheck: stAttention, NoticeDone: stDone, NoticeFailed: stFailed} {
		n := commitNotice(SourceView{Notice: "Committed abc1234.", NoticeKind: kind}, inner)
		if len(n) != 1 || n[0][0].st != st || n[0][0].ref.item != ItemNotice {
			t.Errorf("note kind %c: %v", kind, n)
		}
	}
	ask := commitNotice(SourceView{Ask: 3}, inner)
	if got := texts(ask); !strings.Contains(strings.Join(strings.Fields(got), " "), "Nothing is staged. Stage the 3 changes listed under CHANGES and commit them?") {
		t.Errorf("the question:\n%s", got)
	}
	if last := ask[len(ask)-1]; last[0].ref.item != ItemAskNo || ask[len(ask)-2][0].ref.item != ItemAskYes || ask[len(ask)-2][0].st != active(stBright) {
		t.Error("the question's choices: Stage and commit focused, then Cancel, each on its row at 26 cells")
	}
	if one := commitNotice(SourceView{Ask: 1, AskPush: true, AskNo: true}, 30); plainOf(one[len(one)-1]) != " Stage and commit   Cancel " {
		t.Errorf("the choices on one row at 30 cells: %q", plainOf(one[len(one)-1]))
	}
	if AskText(1, true) != "Nothing is staged. Stage the change listed under CHANGES and commit and push it?" {
		t.Errorf("one change, Commit & Push: %q", AskText(1, true))
	}
	if commitControls(UI{Source: SourceView{Open: true}}, inner) != nil {
		t.Error("controls drawn without a repository")
	}
	// While an operation runs (design D15): its hint in grey, a refused
	// stop's in the attention colour, no earlier note; the footer's cancel.
	busy := commitNotice(SourceView{Busy: "Pushing… 3 s", BusyHint: "Esc cancels the push.", Notice: "old"}, inner)
	if len(busy) != 1 || strings.TrimSpace(plainOf(busy[0])) != "Esc cancels the push." || busy[0][0].st != stGrey || busy[0][0].ref.item != ItemNotice {
		t.Errorf("the push's hint: %q", texts(busy))
	}
	warn := commitNotice(SourceView{Busy: "Pushing…", BusyHint: "git is updating the repository: try again in a moment.", BusyWarn: true}, inner)
	if len(warn) == 0 || warn[0][0].st != stAttention {
		t.Errorf("a refused stop's note: %q", texts(warn))
	}
	if n := commitNotice(SourceView{Busy: "Committing… 4 s", Notice: "old"}, inner); n != nil {
		t.Errorf("a note while a commit runs: %q", texts(n))
	}
	if h := commitHints(SourceView{Busy: "Pushing… 3 s", Place: PlaceMsg}); len(h) == 0 || !strings.HasPrefix(h[0], "Esc cancel the push") {
		t.Errorf("the footer while pushing: %q", h)
	}
	if look := commitLook(SourceView{Busy: "Pushing… 3 s"}, inner); !look.Live || look.Label != "Pushing… 3 s" || look.Check {
		t.Errorf("a running push's look %+v", look)
	}
}

// labelBlock is a coverage mask as GDI gives a label, as high as the
// button: a solid block w wide over rows y0 to y1.
func labelBlock(w, y0, y1 int) *image.Alpha {
	m := image.NewAlpha(image.Rect(0, 0, w, BtnRows*SixelCellH))
	for y := y0; y < y1; y++ {
		for x := range w {
			m.SetAlpha(x, y, color.Alpha{A: 0xFF})
		}
	}
	return m
}

// bright says c is opaque and far lighter than the button's fill: a
// stroke or a glyph in the label's colour.
func bright(c color.NRGBA) bool { return c.A == 0xFF && int(c.R)+int(c.G)+int(c.B) > 450 }

// nearHex says c is within 12 levels of want on each channel, opaque.
func nearHex(c color.NRGBA, want string) bool {
	w := hexColour(want)
	d := func(a, b uint8) bool { return int(a)-int(b) <= 12 && int(b)-int(a) <= 12 }
	return c.A == 0xFF && d(c.R, w.R) && d(c.G, w.G) && d(c.B, w.B)
}

// TestCommitImages is criterion 45.2's images (design D5, the owner's
// image's colours): the button's fill, outline, separator, chevron and
// label, transparent outside its rounded corners; dimmed and focused; the
// dropdown's ground, outline and band; the message box's inside and
// outline, its strips cut from it.
func TestCommitImages(t *testing.T) {
	const cells = 27
	w := cells * SixelCellW
	label := labelBlock(60, 24, 34) // where the label goes
	img := CommitImage(CommitLook{Cells: cells, Label: "Commit", Check: true}, label)
	if b := img.Bounds(); b.Dx() != w || b.Dy() != BtnRows*SixelCellH {
		t.Fatalf("the button %v", b)
	}
	sx := w - moreImageCells*SixelCellW
	for _, c := range []struct {
		name string
		x, y int
		want string
	}{
		{"fill", 20, 30, cBtnFill.tc},
		{"fill under the chevron's part", sx + 4, 15, cBtnFill.tc},
		{"top outline", w / 2, int(btnTop), cBtnOutline.tc},
		{"bottom outline", w / 2, int(btnBottom) - 1, cBtnOutline.tc},
		{"left outline", 0, 30, cBtnOutline.tc},
		{"separator", sx, 30, cBtnSep.tc},
	} {
		if p := img.NRGBAAt(c.x, c.y); !nearHex(p, c.want) {
			t.Errorf("%s at %d,%d: %v, want %s", c.name, c.x, c.y, p, c.want)
		}
	}
	for _, p := range [][2]int{{0, 0}, {w - 1, 59}, {0, int(btnTop)}, {w - 1, int(btnBottom) - 1}, {w / 2, 5}, {w / 2, 55}} {
		if a := img.NRGBAAt(p[0], p[1]).A; a != 0 {
			t.Errorf("%v not transparent: alpha %d", p, a)
		}
	}
	count := func(x0, x1, y0, y1 int) int {
		n := 0
		for y := y0; y < y1; y++ {
			for x := x0; x < x1; x++ {
				if bright(img.NRGBAAt(x, y)) {
					n++
				}
			}
		}
		return n
	}
	if n := count(0, sx, 20, 40); n < 600 {
		t.Errorf("the label and the check: %d pixels in the label's colour", n)
	}
	if n := count((sx+w)/2-8, (sx+w)/2+8, 24, 36); n < 8 {
		t.Errorf("the chevron: %d pixels in the label's colour", n)
	}
	if n := count(sx+2, w-2, int(btnTop)+2, 24) + count(sx+2, w-2, 36, int(btnBottom)-2); n != 0 {
		t.Errorf("%d bright pixels in the chevron's part off the chevron", n)
	}
	if p := img.NRGBAAt(sx/2, int(btnTop)+3); !nearHex(p, cBtnFill.tc) {
		t.Errorf("above the label: %v, want the fill", p)
	}
	dim := CommitImage(CommitLook{Cells: cells, Label: "Commit", Check: true, Dim: true}, label)
	if !nearHex(dim.NRGBAAt(20, 30), cBtnFillDim.tc) || !nearHex(dim.NRGBAAt(w/2, int(btnTop)), cBtnOutlineDim.tc) {
		t.Error("dimmed: not the dim fill and outline")
	}
	focus := CommitImage(CommitLook{Cells: cells, Label: "Commit", Check: true, Focus: true}, label)
	if !nearHex(focus.NRGBAAt(w/2, int(btnTop)), cTitle.tc) {
		t.Errorf("focused: outline %v", focus.NRGBAAt(w/2, int(btnTop)))
	}

	for _, hot := range []bool{false, true} {
		m := MenuImage(MenuLook{Cells: cells, Hot: hot}, nil)
		band := cMenuGround.tc
		if hot {
			band = cMenuBand.tc
		}
		for _, c := range []struct {
			name string
			x, y int
			want string
		}{
			{"ground above the band", w / 2, 4, cMenuGround.tc},
			{"the band's place", w / 2, 27, band},
			{"top outline", w / 2, 0, cMenuOutline.tc},
			{"left outline", 0, 27, cMenuOutline.tc},
		} {
			if p := m.NRGBAAt(c.x, c.y); !nearHex(p, c.want) {
				t.Errorf("hot %v, %s: %v, want %s", hot, c.name, p, c.want)
			}
		}
		if a := m.NRGBAAt(w/2, 57).A; a != 0 {
			t.Errorf("hot %v: under the dropdown not transparent", hot)
		}
	}

	look := MsgLook{Cells: cells, Rows: 2}
	box := MsgBoxImage(look)
	if p := box.NRGBAAt(w/2, 40); !nearHex(p, cMsgInside.tc) {
		t.Errorf("the box's inside %v", p)
	}
	if p := box.NRGBAAt(w/2, 10); !nearHex(p, cMsgOutline.tc) {
		t.Errorf("the box's outline %v", p)
	}
	if a := box.NRGBAAt(w/2, 4).A; a != 0 {
		t.Error("above the box not transparent")
	}
	if p := MsgBoxImage(MsgLook{Cells: cells, Rows: 2, Focus: true}).NRGBAAt(w/2, 10); !nearHex(p, cMsgFocus.tc) {
		t.Errorf("the focused box's outline %v", p)
	}
	strips := MsgStrips(look)
	sizes := [4][2]int{{w, SixelCellH}, {w, SixelCellH}, {SixelCellW, 2 * SixelCellH}, {SixelCellW, 2 * SixelCellH}}
	for i, s := range strips {
		if b := s.Bounds(); b.Dx() != sizes[i][0] || b.Dy() != sizes[i][1] {
			t.Errorf("strip %d is %v", i, b)
		}
	}
	if strips[1].NRGBAAt(w/2, 5) != box.NRGBAAt(w/2, 3*SixelCellH+5) || strips[3].NRGBAAt(9, 5) != box.NRGBAAt(w-1, SixelCellH+5) {
		t.Error("the strips are not cut from the box")
	}
	if MsgSixels(look)[0] == "" || CommitSixel(CommitLook{Cells: cells, Label: "Push"}, nil) == "" || MenuSixel(MenuLook{Cells: cells}, nil) == "" {
		t.Error("an image not encoded")
	}
	// A running operation's button, its label changing every second, is
	// encoded each time and not kept (design D15); a still one is kept.
	still, live := CommitLook{Cells: cells, Label: "Push"}, CommitLook{Cells: cells, Label: "Pushing… 3 s", Live: true}
	if CommitSixel(live, label) != Sixel(CommitImage(live, label)) {
		t.Error("the live button not encoded")
	}
	if _, kept := commitSixels.Load(fmt.Sprintf("%v %p", live, label)); kept {
		t.Error("the live button's image kept")
	}
	if _, kept := commitSixels.Load(fmt.Sprintf("%v %p", still, (*image.Alpha)(nil))); !kept {
		t.Error("the still button's image not kept")
	}
}
