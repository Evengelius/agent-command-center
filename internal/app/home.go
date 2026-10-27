package app

import (
	"fmt"
	"image"
	"reflect"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Version is the application's version, which the home screen's header
// shows (plan 51, design D5).
const Version = "0.1.0"

// systemName is the operating system as the home screen's header names it
// (plan 51, design D5).
func systemName(goos string) string {
	switch goos {
	case "windows":
		return "Windows"
	case "darwin":
		return "macOS"
	case "linux":
		return "Linux"
	}
	return goos
}

// gdiGlyphs gives the home screen's picture its characters as GDI draws
// them (plan 51, design D3), each drawn once; failed says GDI could not
// draw one, the picture then left for the text screen.
type gdiGlyphs struct {
	cache  map[glyphKey]*image.Alpha
	failed bool
}

type glyphKey struct {
	r    rune
	em   int
	bold bool
}

func (g *gdiGlyphs) Glyph(r rune, em int, bold bool) *image.Alpha {
	k := glyphKey{r, em, bold}
	if m, ok := g.cache[k]; ok {
		return m
	}
	m, err := input.Glyph(r, em, bold)
	if err != nil {
		g.failed = true
		m = image.NewAlpha(image.Rect(0, 0, max(em/2, 1), max(em, 1)))
	}
	if g.cache == nil {
		g.cache = map[glyphKey]*image.Alpha{}
	}
	g.cache[k] = m
	return m
}

// homePic is the home screen's picture as last written (plan 51, design
// D13): its look, its pixels, the frame's rows it was written over, and,
// when it was written whole, the sequence that wrote all but the banner;
// and the banner's scene, drawn into the pixels (design D12).
type homePic struct {
	look   render.HomeLook
	pixels *image.NRGBA
	rows   []string
	whole  string
	banner *render.BannerView
}

// homeQuiet is how long the picture waits after a size report before it is
// drawn whole: Windows Terminal reports a window dragged every few tens of
// milliseconds (plan 51, design D13).
const homeQuiet = 120 * time.Millisecond

// homeSettle is how much longer the picture waits for the size to settle
// before it is drawn whole; 0 when it need not wait.
func (m *Model) homeSettle() time.Duration {
	if m.screen == nil || !m.screen.Home.Shown || !m.homeWhole(m.screen.Home.Look) {
		return 0
	}
	return max(homeQuiet-m.opt.Now().Sub(m.resized), 0)
}

// homeWhole says the look needs the picture drawn whole: none yet, or a new
// geometry, the field's cells moved.
func (m *Model) homeWhole(l render.HomeLook) bool {
	h := m.home
	return h.pixels == nil || h.look.Geom != l.Geom || h.look.FieldRow != l.FieldRow || h.look.FieldCols != l.FieldCols
}

// homeSequence writes into draw the home screen's picture the frame asked
// for (plan 51, design D13), and gives where it lies. It is written whole
// when it was not yet, or after the screen was erased (a new size, a
// palette); else the cells a new look changed, and the rows the frame wrote
// again, whose cells lost their pixels. A second drawing, in case the frame
// came out after the first (homeSecond), writes it whole again only when
// the first did, with the same sequence, the banner as its scene is now.
// The banner's cells are written in its palette (design D12).
func (m *Model) homeSequence(draw *strings.Builder) []render.Thumb {
	hs := m.screen.Home
	if !hs.Shown {
		m.home = homePic{}
		return nil
	}
	l := hs.Look
	rects := render.HomeRects(l)
	rows := make([]string, m.ui.Height)
	for y := range rows {
		rows[y] = m.screen.Row(y)
	}
	h := &m.home
	var cells []image.Rectangle
	whole := false
	switch {
	case m.homeWhole(l):
		h.pixels = image.NewNRGBA(l.Geom.Pixels())
		render.HomePaint(h.pixels, l, m.homeGlyphs(), h.pixels.Rect)
		m.homeBanner(l, h.pixels)
		cells, whole = rects, true
	case m.homeSecond && h.whole != "":
		draw.WriteString(h.whole)
		if h.banner != nil {
			cells = []image.Rectangle{h.banner.Cells()} // the scene may have moved since
		}
	default:
		h.whole = ""
		if !reflect.DeepEqual(h.look, l) {
			pic := image.NewNRGBA(l.Geom.Pixels())
			render.HomePaint(pic, l, m.homeGlyphs(), pic.Rect)
			if m.homeBanner(l, pic) {
				cells = append(cells, h.banner.Cells()) // a new scene, in a new palette
			}
			cells = append(cells, render.HomeChanges(h.pixels, pic, rects)...)
			h.pixels = pic
		}
		for y, r := range rows {
			if y < len(h.rows) && r != h.rows[y] && y != l.FieldRow {
				cells = append(cells, image.Rect(0, y, l.Geom.W, y+1)) // a row the frame wrote again
			}
		}
	}
	h.look, h.rows = l, rows
	if m.glyphs.failed {
		m.ui.PillNoFont = true // no font: the next frame draws the screen in text
	}
	var tiles []image.Rectangle
	for _, c := range cells {
		c = render.BottomBlock(c, l.Geom.H)
		for _, r := range rects {
			if x := c.Intersect(r); !x.Empty() {
				tiles = append(tiles, x)
			}
		}
	}
	seq, scene := h.encode(tiles)
	if whole {
		h.whole = seq
	}
	draw.WriteString(seq)
	draw.WriteString(scene)
	now := make([]render.Thumb, len(rects))
	for i, r := range rects {
		now[i] = render.Thumb{Item: -20 - i, X: r.Min.X, Y: r.Min.Y, W: r.Dx(), H: r.Dy()}
	}
	return now
}

// encode is the sequences writing the picture's cells tiles, each image at
// its first cell: seq those HomeSixel encodes; scene the banner's cells, in
// its scene's palette, so that its tiles and its frames' meet without a
// seam (plan 51, design D12). The images are encoded at once, each
// HomeSixel one on its own goroutine, the banner's on one between them,
// their palette's cache being one; all are done when it returns.
func (h *homePic) encode(tiles []image.Rectangle) (seq, scene string) {
	var z image.Rectangle
	if h.banner != nil {
		z = h.banner.Cells()
	}
	var picture, banner []image.Rectangle
	for _, x := range tiles {
		in := x.Intersect(z)
		if in.Empty() {
			picture = append(picture, x)
			continue
		}
		// The banner spans the screen's width: the rest of x lies above or
		// below.
		for _, c := range []image.Rectangle{{Min: x.Min, Max: image.Pt(x.Max.X, min(x.Max.Y, z.Min.Y))}, {Min: image.Pt(x.Min.X, max(x.Min.Y, z.Max.Y)), Max: x.Max}} {
			if !c.Empty() {
				picture = append(picture, c)
			}
		}
		banner = append(banner, in)
	}
	at := func(c image.Rectangle) string { return fmt.Sprintf("\x1b[%d;%dH", c.Min.Y+1, c.Min.X+1) }
	pics, scenes := make([]string, len(picture)), make([]string, len(banner))
	var wg sync.WaitGroup
	for i, c := range picture {
		wg.Go(func() { pics[i] = at(c) + render.HomeSixel(h.pixels, c) })
	}
	wg.Go(func() {
		for i, c := range banner {
			scenes[i] = at(c) + h.banner.Encode(h.pixels, c)
		}
	})
	wg.Wait()
	return strings.Join(pics, ""), strings.Join(scenes, "")
}

// homeBanner keeps the banner's scene for the look l, pic its picture just
// painted (plan 51, design D12): the same view while its place stays, its
// scene drawn into pic again; else a new one, the scene kept when only its
// height changed; none when the banner is hidden, which keeps no buffer. It
// says the view is new, its palette with it: all its cells to be written.
func (m *Model) homeBanner(l render.HomeLook, pic *image.NRGBA) bool {
	h := &m.home
	switch {
	case !l.Geom.Banner:
		h.banner = nil
		return false
	case h.banner != nil && h.banner.Fits(l):
		h.banner.Overlay(pic)
		return false
	}
	h.banner = render.NewBannerView(l, pic, m.opt.Now().UnixNano(), h.banner)
	return true
}

// bannerFrame is a frame of the home screen's banner (plan 51, design D12):
// its scene moved on, the cells it changed written in its palette; none
// while the window lacks the focus or the picture waits to be drawn whole.
func (m *Model) bannerFrame() []tea.Cmd {
	m.bannering = false
	h := &m.home
	if h.banner == nil || h.pixels == nil || m.blurred || m.homeSettle() > 0 {
		return nil
	}
	h.banner.B.Step()
	if seq := h.banner.BannerSequence(h.pixels, h.banner.Frame(h.pixels)); seq != "" {
		return []tea.Cmd{tea.Raw(seq)}
	}
	return nil
}

// homeGlyphs is the glyph source the picture is drawn with.
func (m *Model) homeGlyphs() render.Glyphs { return &m.glyphs }

// homeMouse is a click on the home screen (plan 51, design D11): an option
// chooses its agent CLI, the field takes the focus, the folder button
// opens the folder dialog as Ctrl+O does (plan 52), a project's row is
// selected, its star
// toggles the favourite as f does, START SESSION starts as Enter does.
// Only a left click acts: no double click, wheel or footer click.
func (m *Model) homeMouse(msg tea.MouseMsg) []tea.Cmd {
	click, ok := msg.(tea.MouseClickMsg)
	if !ok || click.Button != tea.MouseLeft || m.screen == nil {
		return nil
	}
	p, item := m.screen.ItemAt(click.X, click.Y)
	if p != render.HomePanel {
		return nil
	}
	kind, i := render.HomeItemOf(item)
	switch kind {
	case render.HomeItemOption:
		if i < len(supported) && contains(m.s.Startup.Installed, supported[i].id) {
			m.ui.Choice, m.ui.Field = supported[i].id, 0
		}
	case render.HomeItemField:
		m.ui.Field = 1
	case render.HomeItemFolder:
		return m.openFolder()
	case render.HomeItemStart:
		return m.confirmStart()
	case render.HomeItemRow, render.HomeItemStar:
		if i >= len(m.projectList) {
			return nil
		}
		m.ui.ProjectSel, m.ui.Field = i, 3
		m.chooseProject() // the field takes its path (plan 53, design D1)
		if kind == render.HomeItemStar {
			return []tea.Cmd{m.starProject()}
		}
	}
	return nil
}

// resetForm is Ctrl+N (plan 51, design D9): the form as the screen opened
// it, the default agent CLI chosen, the starting project folder in the
// field, the focus on Agent CLI, the list's first row, no message; not
// while a start is validated.
func (m *Model) resetForm() []tea.Cmd {
	if m.s.Startup.Phase != state.Asking && m.s.Startup.Phase != state.Detecting {
		return nil
	}
	if inst := m.s.Startup.Installed; len(inst) > 0 {
		m.ui.Choice = inst[0]
		if contains(inst, m.opt.Provider) {
			m.ui.Choice = m.opt.Provider
		}
	}
	m.path.SetText(m.startProject())
	m.ui.Field, m.ui.ProjectSel = 0, 0
	m.ui.FolderProblem = "" // the folder dialog's failure too (plan 52, design D3)
	return m.do(state.ResetForm{})
}

// startProject is the project folder the screen opens with: --project,
// else the current directory.
func (m *Model) startProject() string {
	if m.opt.Project != "" {
		return m.opt.Project
	}
	return m.opt.Cwd
}
