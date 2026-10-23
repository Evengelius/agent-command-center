package app

import (
	"fmt"
	"image"
	"reflect"
	"slices"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/render"
)

// The preview (plan 32, brief amendment 32; doc/research/image-paste.md;
// ui-design.md, Input editor): when the terminal draws Sixel images, each
// attached image's thumbnail above its chip. The frame keeps the
// thumbnail's cells blank; the dashboard draws the images over them with a
// raw sequence once the frame is out, and again after any frame that
// changes those rows, since the renderer's writes erase an image's cells
// (design D5).

// askAttributes is the primary device attributes request; the terminal's
// answer says whether it draws Sixel images (attribute 4).
const askAttributes = "\x1b[c"

// The thumbnails in pixels, render.ThumbCells' cells at render.SixelCellW
// pixels a cell, and the rounding of their corners (design D4).
const (
	thumbBig, thumbBigRadius     = 140, 8
	thumbSmall, thumbSmallRadius = 100, 6
)

// thumbSixel is an attachment's thumbnail encoded at each size; none for an
// image with no pixels to show (a WebP).
type thumbSixel struct{ big, small string }

// thumbDrawMsg asks for the thumbnails drawn after the frame of generation
// gen; second marks the drawing that follows the first.
type thumbDrawMsg struct {
	gen    int
	second bool
}

// thumbFirst and thumbSecond are how long after its frame the thumbnails
// are drawn: first once that frame is surely out (Bubble Tea writes a frame
// at most every 16.7 ms, the raw sequences queued before it), then again in
// case a busy moment held that frame back.
const (
	thumbFirst  = 50 * time.Millisecond
	thumbSecond = 250 * time.Millisecond
)

// thumbsOf encodes the decoded image's thumbnail at both sizes; none for a
// nil image.
func thumbsOf(im image.Image) thumbSixel {
	if im == nil {
		return thumbSixel{}
	}
	return thumbSixel{
		big:   render.Sixel(input.Thumbnail(im, thumbBig, thumbBigRadius, render.ThumbBorder)),
		small: render.Sixel(input.Thumbnail(im, thumbSmall, thumbSmallRadius, render.ThumbBorder)),
	}
}

// deviceAttributes reads the terminal's answer to askAttributes. Bubble Tea
// hands ultraviolet's PrimaryDeviceAttributesEvent to the model as it is
// (doc/research/image-paste.md, The renderer); its type is read by its name
// and kind, so the application imports nothing of ultraviolet, which only
// Bubble Tea depends on.
func deviceAttributes(msg tea.Msg) ([]int, bool) {
	v := reflect.ValueOf(msg)
	if !v.IsValid() || v.Kind() != reflect.Slice || v.Type().Name() != "PrimaryDeviceAttributesEvent" || v.Type().Elem().Kind() != reflect.Int {
		return nil, false
	}
	out := make([]int, v.Len())
	for i := range out {
		out[i] = int(v.Index(i).Int())
	}
	return out, true
}

// attributesAnswered takes the terminal's answer: Sixel graphics or not.
func (m *Model) attributesAnswered(attrs []int) {
	m.ui.Sixel = slices.Contains(attrs, 4)
}

// setThumbs keeps the thumbnails of images attached.
func (m *Model) setThumbs(thumbs map[string]thumbSixel) {
	for k, t := range thumbs {
		if t.big == "" && t.small == "" {
			continue
		}
		if m.thumbs == nil {
			m.thumbs = map[string]thumbSixel{}
		}
		m.thumbs[k] = t
	}
}

// previewFrame follows the frame View just drew: when its thumbnails, their
// places or the rows they lie on differ from the frame before, they are
// drawn again once this frame is out (design D5). It schedules through the
// program; in a test, without one, it records.
func (m *Model) previewFrame() {
	sig := ""
	if m.ui.Sixel && m.screen != nil {
		sig = m.previewSignature()
	}
	if sig == m.thumbSig {
		return
	}
	m.thumbSig = sig
	m.thumbGen++
	if sig == "" && len(m.drawn) == 0 {
		return // nothing to draw, and no image left on the screen to repaint
	}
	gen := m.thumbGen
	if m.send == nil {
		m.kicks = append(m.kicks, gen)
		return
	}
	send := m.send
	time.AfterFunc(thumbFirst, func() { send(thumbDrawMsg{gen: gen}) })
}

// previewSignature is what the frame's thumbnails depend on: their items,
// places and images, and the rows they lie on as drawn.
func (m *Model) previewSignature() string {
	ts := m.screen.Thumbs()
	if len(ts) == 0 {
		return ""
	}
	var b strings.Builder
	var rows []int
	for _, t := range ts {
		key, pixels, sum := "", false, [32]byte{}
		if t.Item < len(m.attach) {
			key, sum = m.attach[t.Item].Path, m.attach[t.Item].Sum // a file attached again: its new image (plan 33, design D7)
			pixels = m.thumbs[key] != thumbSixel{}
		}
		fmt.Fprintf(&b, "%d %d %d %d %d %t %t %s %x\n", t.Item, t.X, t.Y, t.W, t.H, t.Text, pixels, key, sum)
		for y := t.Y; y < t.Y+t.H; y++ {
			if !slices.Contains(rows, y) {
				rows = append(rows, y)
			}
		}
	}
	slices.Sort(rows)
	for _, y := range rows {
		b.WriteString(m.screen.Row(y))
		b.WriteByte('\n')
	}
	return b.String()
}

// drawThumbs draws the frame's thumbnails when it is still the frame the
// drawing was asked for; the first drawing asks for a second.
func (m *Model) drawThumbs(msg thumbDrawMsg) []tea.Cmd {
	if msg.gen != m.thumbGen || m.screen == nil {
		return nil // a later frame changed them: its own drawing follows
	}
	seq := m.thumbSequence()
	if seq == "" {
		return nil
	}
	cmds := []tea.Cmd{tea.Raw(seq)}
	if !msg.second {
		cmds = append(cmds, m.opt.After(thumbSecond, thumbDrawMsg{gen: msg.gen, second: true}))
	}
	return cmds
}

// thumbSequence is one raw sequence, inside synchronized output, the
// cursor saved then restored where the renderer left it: first each place
// an image was drawn and no thumbnail of this frame takes, written over
// with what the frame holds there, which erases the image (a cell the
// renderer leaves alone keeps it, probe P32b); then each thumbnail's image
// at its first cell. It records where images now lie.
func (m *Model) thumbSequence() string {
	var now []render.Thumb
	var draw strings.Builder
	if m.ui.Sixel {
		for _, t := range m.screen.Thumbs() {
			if t.Text || t.Item >= len(m.attach) {
				continue
			}
			ts := m.thumbs[m.attach[t.Item].Path]
			six := ts.small
			if t.W*render.SixelCellW >= thumbBig {
				six = ts.big
			}
			if six == "" {
				continue
			}
			fmt.Fprintf(&draw, "\x1b[%d;%dH%s", t.Y+1, t.X+1, six)
			now = append(now, t)
		}
	}
	var clear strings.Builder
	for _, old := range m.drawn {
		if slices.ContainsFunc(now, func(t render.Thumb) bool {
			return t.X == old.X && t.Y == old.Y && t.W == old.W && t.H == old.H
		}) {
			continue // drawn over by this frame's image
		}
		for y := old.Y; y < old.Y+old.H; y++ {
			fmt.Fprintf(&clear, "\x1b[%d;%dH%s", y+1, old.X+1, m.screen.Region(old.X, y, old.W))
		}
	}
	m.drawn = now
	if clear.Len() == 0 && draw.Len() == 0 {
		return ""
	}
	return "\x1b[?2026h\x1b7" + clear.String() + draw.String() + "\x1b8\x1b[?2026l"
}
