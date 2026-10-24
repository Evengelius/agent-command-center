package app

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/input/imagetest"
	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// clip is a fake clipboard with text, an image's PNG bytes and copied
// files (plan 32); reads counts the text reads.
type clip struct {
	mu    sync.Mutex
	text  string
	image []byte
	files []string
	reads int
}

func (c *clip) Read() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.reads++
	if c.text == "" {
		return "", errors.New("clipboard: no data available for the given format")
	}
	return c.text, nil
}

func (c *clip) Write(t string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.text = t
	return nil
}

func (c *clip) Image() ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.image, nil
}

func (c *clip) Files() ([]string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.files, nil
}

func (c *clip) set(text string, image []byte, files ...string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.text, c.image, c.files = text, image, files
}

func (c *clip) readCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.reads
}

// PrimaryDeviceAttributesEvent stands for ultraviolet's event of that name
// in the tests: the dashboard knows it by its name and kind (preview.go).
type PrimaryDeviceAttributesEvent []int

// drawnPNG is the test drawing as PNG at a compression level, so two
// levels give the same image in two different byte strings.
func drawnPNG(t *testing.T, level png.CompressionLevel) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := (&png.Encoder{CompressionLevel: level}).Encode(&b, imagetest.Draw()); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func altV() tea.KeyPressMsg { return key('v', tea.ModAlt) }

// ctrlVRelease is the V key's release with Ctrl held, as Windows Terminal
// forwards it when it keeps Ctrl+V's press (probe P32a).
func ctrlVRelease() tea.KeyReleaseMsg { return tea.KeyReleaseMsg{Code: 'v', Mod: tea.ModCtrl} }

// TestPasteImage is plan 32's criterion 32.1 (design D1 and D3): Alt+V
// attaches the clipboard's image as "pasted-image-N.png" or its copied
// files by name, the draft untouched and nothing sent; the same bytes are
// not attached twice; a copied non-image is refused by name; text alone
// is not pasted; the send carries the pasted image's exact bytes.
func TestPasteImage(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		files, err := imagetest.WriteAll(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		r := scriptRig(t, "claude-code")
		h := r.h
		c := &clip{}
		h.m.opt.Clipboard = c
		for _, ch := range "What is in these images?" {
			h.send(typed(ch))
		}
		h.send(key(tea.KeyHome, 0))
		cursor, text := h.m.edit.Cursor(), h.m.edit.Text()
		pngA, pngB := drawnPNG(t, png.DefaultCompression), drawnPNG(t, png.BestSpeed)
		if bytes.Equal(pngA, pngB) {
			t.Fatal("the two encodings are the same bytes")
		}

		c.set("", pngA)
		h.send(altV())
		want := fmt.Sprintf("pasted-image-1.png · %s ✕", input.FmtBytes(int64(len(pngA))))
		if f := h.frame(); !strings.Contains(f, want) {
			t.Errorf("no chip %q:\n%s", want, f)
		}
		if h.m.edit.Text() != text || h.m.edit.Cursor() != cursor || len(r.sent()) != 0 {
			t.Errorf("after the paste: draft %q cursor %d, %d sent", h.m.edit.Text(), h.m.edit.Cursor(), len(r.sent()))
		}
		h.send(altV())
		if len(h.m.attach) != 1 || !strings.Contains(h.m.ui.Toast, "pasted-image-1.png: already attached") {
			t.Errorf("the same bytes again: %d attachments, toast %q", len(h.m.attach), h.m.ui.Toast)
		}
		c.set("", pngB)
		h.send(altV())
		if len(h.m.attach) != 2 || h.m.attach[1].Name != "pasted-image-2.png" {
			t.Errorf("other bytes: %+v", h.m.attach)
		}

		c.set("", nil, files["png"], files["jpg"])
		h.send(altV())
		if len(h.m.attach) != 4 || h.m.attach[2].Name != "acc-image-upload-test.png" || h.m.attach[3].Name != "acc-image-upload-test.jpg" {
			t.Errorf("copied files: %+v", h.m.attach)
		}
		c.set("", nil, files["text"])
		h.send(altV())
		if name := filepath.Base(files["text"]); len(h.m.attach) != 4 || !strings.Contains(h.m.ui.Toast, name+": ") {
			t.Errorf("a copied text file: %d attachments, toast %q", len(h.m.attach), h.m.ui.Toast)
		}
		c.set("some text", nil)
		h.send(altV())
		if len(h.m.attach) != 4 || h.m.ui.Toast != "no image on the clipboard" || h.m.edit.Text() != text {
			t.Errorf("text alone: %d attachments, toast %q, draft %q", len(h.m.attach), h.m.ui.Toast, h.m.edit.Text())
		}

		h.send(key(tea.KeyEnter, 0))
		msgs := r.script.messages()
		if len(msgs) != 1 || msgs[0].Text != text || len(msgs[0].Images) != 4 {
			t.Fatalf("sent %+v", msgs)
		}
		if img := msgs[0].Images[0]; img.Name != "pasted-image-1.png" || img.Media != "image/png" || !bytes.Equal(img.Data, pngA) {
			t.Errorf("the pasted image sent as %s %s, %d bytes, want the pasted %d", img.Name, img.Media, len(img.Data), len(pngA))
		}
		if len(h.m.attach) != 0 || strings.Contains(h.frame(), "✕") {
			t.Errorf("after the send: %d attachments", len(h.m.attach))
		}
	})
}

// TestPasteCtrlV is criterion 32.2 (design D2): Ctrl+V's lone release, no
// paste after it, attaches the clipboard's image once the wait is over; a
// paste in the wait is the terminal's text, nothing attached; a press then
// the release is the dashboard's own paste, once; a paste of exactly the
// clipboard's files' paths attaches them when all are images, and goes in
// as text otherwise.
func TestPasteCtrlV(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		files, err := imagetest.WriteAll(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		r := scriptRig(t, "claude-code")
		h := r.h
		c := &clip{}
		h.m.opt.Clipboard = c
		var waits []tea.Msg
		h.m.opt.After = func(_ time.Duration, msg tea.Msg) tea.Cmd {
			if _, ok := msg.(pasteWaitMsg); ok {
				waits = append(waits, msg)
			}
			return nil
		}
		pngA := drawnPNG(t, png.DefaultCompression)

		// The lone release, no paste: the image once the wait is over.
		c.set("", pngA)
		h.send(ctrlVRelease())
		if len(waits) != 1 || len(h.m.attach) != 0 {
			t.Fatalf("after the release: %d waits, %d attachments", len(waits), len(h.m.attach))
		}
		h.send(waits[0])
		if len(h.m.attach) != 1 || h.m.attach[0].Name != "pasted-image-1.png" {
			t.Fatalf("after the wait: %+v", h.m.attach)
		}

		// The release, then the terminal's paste within the wait.
		h.send(ctrlVRelease())
		h.send(tea.PasteMsg{Content: "pasted text"})
		h.send(waits[1])
		if len(h.m.attach) != 1 || h.m.edit.Text() != "pasted text" {
			t.Errorf("a paste in the wait: %d attachments, draft %q", len(h.m.attach), h.m.edit.Text())
		}

		// The press, then the release: the dashboard's own paste, once.
		c.set(" and more", pngA)
		reads := c.readCount()
		h.send(key('v', tea.ModCtrl))
		h.send(ctrlVRelease())
		if len(waits) != 2 || c.readCount() != reads+1 || h.m.edit.Text() != "pasted text and more" || len(h.m.attach) != 1 {
			t.Errorf("the press and the release: %d waits, %d reads, draft %q, %d attachments", len(waits), c.readCount()-reads, h.m.edit.Text(), len(h.m.attach))
		}

		// The clipboard's files' paths pasted: images attached, the text kept out.
		c.set("", nil, files["png"], files["jpg"])
		before := h.m.edit.Text()
		h.send(tea.PasteMsg{Content: files["png"] + " " + files["jpg"]})
		if len(h.m.attach) != 3 || h.m.edit.Text() != before {
			t.Errorf("the files' paths: %+v, draft %q", h.m.attach, h.m.edit.Text())
		}
		// One of them not an image: the paths go in as text.
		c.set("", nil, files["gif"], files["text"])
		paths := files["gif"] + " " + files["text"]
		h.send(tea.PasteMsg{Content: paths})
		if len(h.m.attach) != 3 || h.m.edit.Text() != before+paths {
			t.Errorf("a non-image among them: %d attachments, draft %q", len(h.m.attach), h.m.edit.Text())
		}
	})
}

// TestPastePaths is design D2's comparison: a paste is the files' paths
// when it holds each, as it is or quoted, and nothing else.
func TestPastePaths(t *testing.T) {
	files := []string{`C:\shots\a.png`, `C:\my shots\b.png`}
	for _, tc := range []struct {
		text  string
		paths bool
		same  bool
	}{
		{`C:\shots\a.png "C:\my shots\b.png"`, true, true},
		{`"C:\my shots\b.png" C:\shots\a.png`, true, true},
		{"C:\\shots\\a.png\r\nC:\\my shots\\b.png", true, true},
		{`C:\shots\a.png`, true, false},
		{`C:\shots\a.png C:\my shots\b.png and more`, true, false},
		{`see C:\shots\a.png`, false, false},
		{`\\server\share\a.png`, true, false},
	} {
		if got := looksLikePaths(tc.text); got != tc.paths {
			t.Errorf("looksLikePaths(%q) = %v", tc.text, got)
		}
		if got := samePaths(tc.text, files); got != tc.same {
			t.Errorf("samePaths(%q) = %v", tc.text, got)
		}
	}
	if got := joinPaths(files); got != `C:\shots\a.png "C:\my shots\b.png"` || !samePaths(got, files) {
		t.Errorf("joinPaths %q", got)
	}
}

// TestPastedDraft was criterion 32.3 (design D3); since plan 33 (design D8,
// criterion 33.13) the stored draft keeps a pasted image's bytes in one
// file of its folder, named by their SHA-256, and in no other; restored,
// the text, the pasted image and the file come back in their order.
func TestPastedDraft(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		files, err := imagetest.WriteAll(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		pngA := drawnPNG(t, png.DefaultCompression)
		d := input.Drafts{Dir: t.TempDir()}
		cwd := t.TempDir()
		c := &clip{image: pngA}
		p := &picker{answers: [][]string{{files["jpg"]}}}
		h := wireRig(t, &wireClaude{}, Options{Drafts: &d, Cwd: cwd, PickImages: p.pick, Clipboard: c})
		h.send(tea.PasteMsg{Content: "kept"})
		h.send(altV())
		upload(t, h)
		if len(h.m.attach) != 2 {
			t.Fatalf("attached %+v", h.m.attach)
		}
		project := h.m.State().Context.Project
		sum := sha256.Sum256(pngA)
		kept := hex.EncodeToString(sum[:]) + ".png"
		paths := draftList(t, d, project)
		if len(paths) != 2 || paths[0] != "pasted:pasted-image-1.png#"+hex.EncodeToString(sum[:]) || paths[1] != files["jpg"] {
			t.Fatalf("stored images %q", paths)
		}
		// The image's bytes in its one file, in no other, raw or in base64.
		_ = filepath.Walk(d.Dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return err
			}
			b, _ := os.ReadFile(path)
			if filepath.Base(path) == kept {
				if !bytes.Equal(b, pngA) {
					t.Errorf("%s differs from the pasted bytes", path)
				}
				return nil
			}
			if bytes.Contains(b, pngA[8:72]) || bytes.Contains(b, []byte(base64.StdEncoding.EncodeToString(pngA)[:64])) {
				t.Errorf("%s holds the pasted image", path)
			}
			return nil
		})
		h.crash()
		h2 := wireRig(t, &wireClaude{}, Options{Drafts: &d, Cwd: cwd, Clipboard: c})
		if h2.m.edit.Text() != "kept" || len(h2.m.attach) != 2 || h2.m.attach[0].Name != "pasted-image-1.png" || h2.m.attach[1].Name != "acc-image-upload-test.jpg" {
			t.Errorf("restored: draft %q, %+v", h2.m.edit.Text(), h2.m.attach)
		}
		if !bytes.Equal(h2.m.pasted["pasted:pasted-image-1.png"], pngA) {
			t.Error("the restored pasted image differs from the pasted bytes")
		}
		if h2.m.ui.Toast != "draft restored" {
			t.Errorf("toast %q", h2.m.ui.Toast)
		}
	})
}

// TestPreviewDraw is criterion 32.6 (design D5): no drawing before the
// terminal says it draws Sixel; then the thumbnails drawn after the frame
// that shows them, again after a second wait, and after any frame that
// changes their rows, not after one that changes other rows; a vacated
// place written over with the frame's own cells; no strip without Sixel.
func TestPreviewDraw(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := scriptRig(t, "claude-code")
		h := r.h
		h.send(tea.WindowSizeMsg{Width: 200, Height: 60})
		c := &clip{image: drawnPNG(t, png.DefaultCompression)}
		h.m.opt.Clipboard = c
		var later []tea.Msg
		h.m.opt.After = func(_ time.Duration, msg tea.Msg) tea.Cmd {
			if _, ok := msg.(thumbDrawMsg); ok {
				later = append(later, msg)
			}
			return nil
		}
		h.send(altV())
		h.frame()
		if len(h.m.kicks) != 0 || len(h.m.screen.Thumbs()) != 0 {
			t.Fatalf("before the answer: %d drawings asked, %d thumbnails", len(h.m.kicks), len(h.m.screen.Thumbs()))
		}

		h.send(PrimaryDeviceAttributesEvent{61, 4, 6, 7, 14})
		h.frame()
		thumbs := h.m.screen.Thumbs()
		if !h.m.ui.Sixel || len(h.m.kicks) != 1 || len(thumbs) != 1 {
			t.Fatalf("after the answer: Sixel %v, %d drawings asked, %d thumbnails", h.m.ui.Sixel, len(h.m.kicks), len(thumbs))
		}
		th := thumbs[0]
		if th.W != 14 || th.H != 7 {
			t.Errorf("the thumbnail is %dx%d cells, want 14x7", th.W, th.H)
		}
		gen := h.m.kicks[0]
		raws := drawn(h, thumbDrawMsg{gen: gen})
		at := fmt.Sprintf("\x1b[%d;%dH\x1bP0;1;0q\"1;1;140;140", th.Y+1, th.X+1)
		if len(raws) != 1 || !strings.HasPrefix(raws[0], "\x1b[?2026h\x1b7"+at) || !strings.HasSuffix(raws[0], "\x1b\\\x1b8\x1b[?2026l") {
			t.Fatalf("the drawing %q", trunc(raws))
		}
		if len(later) != 1 || later[0] != (thumbDrawMsg{gen: gen, second: true}) {
			t.Errorf("the second drawing asked: %v", later)
		}
		if raws := drawn(h, thumbDrawMsg{gen: gen, second: true}); len(raws) != 1 || !strings.Contains(raws[0], at) {
			t.Errorf("the second drawing %q", trunc(raws))
		}

		// A key typed: the draft's row changes, not the strip's.
		h.send(typed('x'))
		h.frame()
		if len(h.m.kicks) != 1 {
			t.Errorf("a frame changing other rows asked for %d drawings", len(h.m.kicks)-1)
		}
		// The focus to zone C: B's frame, on the strip's rows, changes.
		h.send(key(tea.KeyF6, 0))
		h.frame()
		if len(h.m.kicks) != 2 {
			t.Fatalf("a frame changing the strip's rows: %d drawings asked in all", len(h.m.kicks))
		}
		if raws := drawn(h, thumbDrawMsg{gen: h.m.kicks[1]}); len(raws) != 1 || !strings.Contains(raws[0], at) {
			t.Errorf("drawn again: %q", trunc(raws))
		}
		h.send(key(tea.KeyF6, 0)) // back to B through A
		h.send(key(tea.KeyF6, 0))
		h.frame()

		// The image removed: its place written over with the frame's cells,
		// then nothing more.
		h.send(key('x', tea.ModCtrl|tea.ModShift))
		h.frame()
		n := len(h.m.kicks)
		raws = drawn(h, thumbDrawMsg{gen: h.m.kicks[n-1]})
		place := fmt.Sprintf("\x1b[%d;%dH", th.Y+1, th.X+1)
		if len(raws) != 1 || !strings.Contains(raws[0], place) || strings.Contains(withoutPill(t, h, raws[0]), "\x1bP") {
			t.Errorf("the vacated place: %q", trunc(raws))
		}
		h.send(typed('y'))
		h.frame()
		if len(h.m.kicks) != n {
			t.Errorf("%d drawings asked once the image was gone", len(h.m.kicks)-n)
		}

		// An answer without 4: no strip.
		h.send(PrimaryDeviceAttributesEvent{61, 6})
		h.send(altV())
		h.frame()
		if h.m.ui.Sixel || len(h.m.attach) != 1 || len(h.m.screen.Thumbs()) != 0 {
			t.Errorf("without Sixel: %v, %d attachments, %d thumbnails", h.m.ui.Sixel, len(h.m.attach), len(h.m.screen.Thumbs()))
		}
	})
}

// drawn delivers a drawing message and gives the raw sequences it wrote.
func drawn(h *harness, msg thumbDrawMsg) []string {
	_, cmd := h.m.Update(msg)
	var out []string
	for _, c := range batch(cmd) {
		if raw, ok := c().(tea.RawMsg); ok {
			out = append(out, fmt.Sprint(raw.Msg))
		}
	}
	return out
}

// pillDrawing is the mode pill's part of a drawing on the current frame, ""
// when the frame shows no pill image (plan 39, design D7): its title drawn
// by GDI 13 pixels high from one pixel down.
func pillDrawing(h *harness) string {
	mp := h.m.screen.ModePill
	if !mp.Shown {
		return ""
	}
	label, err := input.TextMask(mp.Look.Title, 13, 1, render.SixelCellH)
	if err != nil {
		return ""
	}
	w := render.ModePillCells(mp.Look)
	at := fmt.Sprintf("%c[%d;%dH", 0x1b, mp.Y+1, mp.X+1)
	return at + h.m.screen.Region(mp.X, mp.Y, w) + at + render.PillSixel(mp.Look, label)
}

// withoutPill is a drawing without the mode pill's part, which every
// drawing on a dashboard with Sixel holds; the pill there once.
func withoutPill(t *testing.T, h *harness, raw string) string {
	t.Helper()
	pill := pillDrawing(h)
	if pill == "" {
		return raw
	}
	if n := strings.Count(raw, pill); n != 1 {
		t.Errorf("the pill drawn %d times in %q", n, trunc([]string{raw}))
	}
	return strings.Replace(raw, pill, "", 1)
}

// trunc shortens raw sequences for a failure's message.
func trunc(raws []string) []string {
	out := make([]string, len(raws))
	for i, r := range raws {
		if len(r) > 120 {
			r = r[:60] + "…" + r[len(r)-60:]
		}
		out[i] = r
	}
	return out
}

// TestWin32PasteRelease drives the real program with what Windows Terminal
// 1.23 sends (probe P32a, out-p32probe-da1): the answer to the device
// attributes request turns the preview on, and Ctrl+V's lone release,
// Ctrl's press and release around it, attaches the clipboard's image once
// the wait is over; the thumbnail is then drawn (criteria 32.2 and 32.6).
func TestWin32PasteRelease(t *testing.T) {
	const (
		da1      = "\x1b[?61;4;6;7;14;21;22;23;24;28;32;42;52c"
		ctrlVOff = "\x1b[17;29;0;1;40;1_\x1b[86;47;22;0;40;1_\x1b[17;29;0;0;32;1_"
	)
	in, keys := io.Pipe()
	typed := false
	updateHook = func(m *Model, _ tea.Msg) []tea.Cmd {
		if !typed && m.s.Screen == state.ScreenDashboard && m.s.Session.Phase == state.Ready {
			typed = true
			go func() {
				_, _ = io.WriteString(keys, da1+ctrlVOff)
				time.Sleep(1500 * time.Millisecond) // the wait, the image read, the thumbnail drawn
				_, _ = io.WriteString(keys, wtCtrlQ)
				time.Sleep(300 * time.Millisecond)
				_, _ = io.WriteString(keys, wtCtrlQ)
			}()
		}
		return nil
	}
	resetTerminal = func(string) {}
	t.Cleanup(func() { resetTerminal, updateHook = consoleReset, nil; _ = in.Close() })

	m := New(Options{
		Config:    Config{Mock: filepath.Join("..", "..", "testdata", "mock", "claude-subagent-approval.jsonl")},
		Cwd:       t.TempDir(),
		Now:       time.Now,
		MockSpeed: 0,
		Clipboard: &clip{image: drawnPNG(t, png.DefaultCompression)},
	})
	done := make(chan int, 1)
	go func() {
		code, _ := Run(m, tea.WithInput(in), tea.WithOutput(io.Discard), tea.WithWindowSize(120, 40), tea.WithoutSignals())
		done <- code
	}()
	select {
	case code := <-done:
		if code != ExitOK {
			t.Fatalf("exit code %d", code)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("the program did not end on Ctrl+Q")
	}
	if !m.ui.Sixel {
		t.Error("the terminal's answer did not turn the preview on")
	}
	if len(m.attach) != 1 || m.attach[0].Name != "pasted-image-1.png" {
		t.Errorf("attached %+v", m.attach)
	}
	// The thumbnail, and the mode pill (plan 39), Item -5.
	if len(m.drawn) != 2 || m.drawn[0].W != 10 || m.drawn[0].H != 5 || m.drawn[1].Item != -5 {
		t.Errorf("drawn %+v, want one thumbnail of 10x5 cells and the mode pill", m.drawn)
	}
}
