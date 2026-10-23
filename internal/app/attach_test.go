package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/input/imagetest"
	"github.com/Evengelius/agent-command-center/internal/provider"
)

// messages are the messages the scripted session received, their images
// with them.
func (s *scriptSession) messages() []provider.Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]provider.Message(nil), s.msgs...)
}

// picker is a fake Open dialog: each call gives the next answer, none when
// they are used up (a cancel).
type picker struct {
	answers [][]string
	calls   int
}

func (p *picker) pick() ([]string, error) {
	p.calls++
	if len(p.answers) == 0 {
		return nil, nil
	}
	a := p.answers[0]
	p.answers = p.answers[1:]
	return a, nil
}

// clickAt presses and releases the left button on the cell.
func clickAt(t *testing.T, h *harness, x, y int) {
	t.Helper()
	mouseAt(t, h, click(x, y))
	mouseAt(t, h, tea.MouseReleaseMsg{X: x, Y: y, Button: tea.MouseLeft})
}

// upload opens the "+" menu and clicks "Upload from computer", as the owner
// does.
func upload(t *testing.T, h *harness) {
	t.Helper()
	x, y := at(t, h, "+     ", 0)
	clickAt(t, h, x+1, y)
	ix, iy := at(t, h, "Upload from computer", 0)
	clickAt(t, h, ix+3, iy)
	if h.m.ui.Plus {
		t.Fatal("the menu stayed open after its item was clicked")
	}
}

// TestAttachments is plan 31's criterion 31.3 (design D3 to D5): images
// chosen in the Open dialog become the draft's attachments, their chips
// shown, the text and the cursor untouched and nothing sent; a file already
// attached is skipped, a refused one said, a cancel changes nothing; a
// chip's ✕ and Ctrl+Shift+X remove one; the send key sends the text and
// the images' exact bytes and clears the chips; a file gone before the send
// keeps the draft with its images.
func TestAttachments(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		files, err := imagetest.WriteAll(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		r := scriptRig(t, "claude-code")
		h := r.h
		p := &picker{}
		h.m.opt.PickImages = p.pick
		h.send(tea.PasteMsg{Content: "What colours and what text are in these images?"})
		h.send(key(tea.KeyHome, 0))
		cursor, text := h.m.edit.Cursor(), h.m.edit.Text()

		// Two images chosen at once: two chips, the draft as it was, nothing sent.
		p.answers = [][]string{{files["png"], files["jpg"]}}
		upload(t, h)
		f := h.frame()
		for _, want := range []string{"acc-image-upload-test.png · 1 KB ✕", "acc-image-upload-test.jpg · 6 KB ✕"} {
			if !strings.Contains(f, want) {
				t.Errorf("no chip %q:\n%s", want, f)
			}
		}
		if h.m.edit.Text() != text || h.m.edit.Cursor() != cursor || len(r.sent()) != 0 || p.calls != 1 {
			t.Errorf("after attaching: draft %q cursor %d, %d sent, %d dialogs", h.m.edit.Text(), h.m.edit.Cursor(), len(r.sent()), p.calls)
		}

		// The same file again, a refused one, then a cancel.
		p.answers = [][]string{{files["png"]}}
		upload(t, h)
		if len(h.m.attach) != 2 || !strings.Contains(h.m.ui.Toast, "acc-image-upload-test.png: already attached") {
			t.Errorf("a double: %d attachments, toast %q", len(h.m.attach), h.m.ui.Toast)
		}
		p.answers = [][]string{{files["svg"]}}
		upload(t, h)
		if len(h.m.attach) != 2 || !strings.Contains(h.m.ui.Toast, "drawing.svg: SVG is not an image the providers take") {
			t.Errorf("an SVG: %d attachments, toast %q", len(h.m.attach), h.m.ui.Toast)
		}
		upload(t, h) // no answer left: cancelled
		if len(h.m.attach) != 2 || p.calls != 4 {
			t.Errorf("a cancel: %d attachments, %d dialogs", len(h.m.attach), p.calls)
		}

		// A click on the first chip's ✕ removes it; Ctrl+Shift+X the last.
		x, y := at(t, h, "✕", 0)
		clickAt(t, h, x, y)
		if len(h.m.attach) != 1 || h.m.attach[0].Name != "acc-image-upload-test.jpg" || strings.Contains(h.frame(), "acc-image-upload-test.png ·") {
			t.Errorf("after the ✕: %+v", h.m.attach)
		}
		h.send(key('x', tea.ModCtrl|tea.ModShift))
		if len(h.m.attach) != 0 || strings.Contains(h.frame(), "✕") || h.m.edit.Text() != text {
			t.Errorf("after Ctrl+Shift+X: %+v, draft %q", h.m.attach, h.m.edit.Text())
		}

		// Attached again, with a Unicode name and a path with spaces, then sent.
		p.answers = [][]string{{files["png"], files["unicode"], files["spaces"]}}
		upload(t, h)
		h.send(key(tea.KeyEnter, 0))
		msgs := r.script.messages()
		if len(msgs) != 1 || msgs[0].Text != text || len(msgs[0].Images) != 3 {
			t.Fatalf("sent %+v", msgs)
		}
		png, _ := os.ReadFile(files["png"])
		for i, name := range []string{"acc-image-upload-test.png", filepath.Base(files["unicode"]), "acc image upload test.png"} {
			img := msgs[0].Images[i]
			if img.Name != name || img.Media != "image/png" || !bytes.Equal(img.Data, png) {
				t.Errorf("image %d: %s %s, %d bytes, want %s and the PNG's %d", i, img.Name, img.Media, len(img.Data), name, len(png))
			}
		}
		if len(h.m.attach) != 0 || h.m.edit.Text() != "" || strings.Contains(h.frame(), "✕") {
			t.Errorf("after the send: %d attachments, draft %q", len(h.m.attach), h.m.edit.Text())
		}
		// The message as an adapter maps the line it wrote: the images' media
		// types and sizes, no names; the entry in zone B lists them under the
		// text, named from the attachments.
		var imgs []event.Image
		for range 3 {
			imgs = append(imgs, event.Image{Media: "image/png", Size: len(png)})
		}
		r.emit(event.UserMessage, event.User{Text: text, Images: imgs})
		h.settle()
		if f := h.frame(); !strings.Contains(f, "🗀  acc-image-upload-test.png · 1 KB") || !strings.Contains(f, "🗀  acc image upload test.png · 1 KB") {
			t.Errorf("the sent entry does not list its images:\n%s", f)
		}
		r.end()

		// A file deleted between attaching and sending: not sent, the draft
		// and its image kept, the reason said.
		gone := filepath.Join(filepath.Dir(files["png"]), "gone.png")
		if err := os.WriteFile(gone, png, 0o644); err != nil {
			t.Fatal(err)
		}
		p.answers = [][]string{{gone}}
		h.send(tea.PasteMsg{Content: "again"})
		upload(t, h)
		if err := os.Remove(gone); err != nil {
			t.Fatal(err)
		}
		h.send(key(tea.KeyEnter, 0))
		if n := len(r.script.messages()); n != 1 {
			t.Errorf("%d messages sent, want the first only", n)
		}
		if h.m.edit.Text() != "again" || len(h.m.attach) != 1 || h.m.attach[0].Name != "gone.png" {
			t.Errorf("after the failed send: draft %q, %+v", h.m.edit.Text(), h.m.attach)
		}
		if n := h.m.State().Turn.Notice; !strings.Contains(n, "gone.png: the file is gone") {
			t.Errorf("notice %q", n)
		}
	})
}

// TestAttachmentsDraft is design D6: the stored draft lists its images; a
// new model in the same project restores the text and the images still
// there, and says which are gone.
func TestAttachmentsDraft(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		files, err := imagetest.WriteAll(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		gone := filepath.Join(filepath.Dir(files["png"]), "gone.png")
		png, _ := os.ReadFile(files["png"])
		if err := os.WriteFile(gone, png, 0o644); err != nil {
			t.Fatal(err)
		}
		d := input.Drafts{Dir: t.TempDir()}
		cwd := t.TempDir()
		p := &picker{answers: [][]string{{files["jpg"], gone}}}
		h := wireRig(t, &wireClaude{}, Options{Drafts: &d, Cwd: cwd, PickImages: p.pick})
		h.send(tea.PasteMsg{Content: "kept"})
		upload(t, h)
		project := h.m.State().Context.Project
		if paths, err := d.LoadImages(project); err != nil || len(paths) != 2 {
			t.Fatalf("stored images %q, %v", paths, err)
		}
		// The first model dropped as a crash would; one image deleted since.
		if err := os.Remove(gone); err != nil {
			t.Fatal(err)
		}
		h2 := wireRig(t, &wireClaude{}, Options{Drafts: &d, Cwd: cwd})
		if h2.m.edit.Text() != "kept" || len(h2.m.attach) != 1 || h2.m.attach[0].Name != "acc-image-upload-test.jpg" {
			t.Errorf("restored: draft %q, %+v", h2.m.edit.Text(), h2.m.attach)
		}
		if !strings.Contains(h2.m.ui.Toast, "gone.png: the file is gone") {
			t.Errorf("toast %q", h2.m.ui.Toast)
		}
	})
}
