package app

// Plan 33 (brief amendment 33): an unsent draft and its images through
// crashes, Back, two dashboards on one project and restarts; criteria 33.2
// to 33.9, and the Codex part of 33.11. A crash drops a model as a killed
// process would (harness.crash): its draft's lock closed, nothing written
// after.

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/input/imagetest"
	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/state"
)

func rcFiles(t *testing.T) imagetest.Files {
	t.Helper()
	files, err := imagetest.WriteAll(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func rcRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// rcNames are the attachments' names, in order.
func rcNames(m *Model) []string {
	var out []string
	for _, a := range m.attach {
		out = append(out, a.Name)
	}
	return out
}

// rcRig is a wired Claude Code session with a draft store, an Open dialog
// answering each list in turn, a clipboard, and the terminal saying it
// draws Sixel.
func rcRig(t *testing.T, d *input.Drafts, cwd string, c *clip, w *wireClaude, answers ...[]string) *harness {
	t.Helper()
	opt := Options{Drafts: d, Cwd: cwd, Clipboard: c}
	if len(answers) > 0 {
		p := &picker{answers: answers}
		opt.PickImages = p.pick
	}
	h := wireRig(t, w, opt)
	h.send(PrimaryDeviceAttributesEvent{61, 4, 6})
	return h
}

// rcSent is the images of the last user message the fake Claude Code
// read, in order; none without one.
func rcSent(t *testing.T, w *wireClaude) [][]byte {
	t.Helper()
	lines := w.written("user")
	if len(lines) == 0 {
		return nil
	}
	var m struct {
		Message struct {
			Content json.RawMessage `json:"content"`
		} `json:"message"`
	}
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &m); err != nil {
		t.Fatal(err)
	}
	var blocks []struct {
		Type   string `json:"type"`
		Source struct {
			Data string `json:"data"`
		} `json:"source"`
	}
	_ = json.Unmarshal(m.Message.Content, &blocks)
	var out [][]byte
	for _, b := range blocks {
		if b.Type == "image" {
			data, err := base64.StdEncoding.DecodeString(b.Source.Data)
			if err != nil {
				t.Fatal(err)
			}
			out = append(out, data)
		}
	}
	return out
}

// codexSent is the images of the last turn/start the fake app-server read.
func codexSent(t *testing.T, w *wireCodex) [][]byte {
	t.Helper()
	ls := w.method("turn/start")
	if len(ls) == 0 {
		t.Fatal("no turn/start written")
	}
	var m struct {
		Params struct {
			Input []struct {
				Type string `json:"type"`
				URL  string `json:"url"`
			} `json:"input"`
		} `json:"params"`
	}
	if err := json.Unmarshal([]byte(ls[len(ls)-1]), &m); err != nil {
		t.Fatal(err)
	}
	var out [][]byte
	for _, in := range m.Params.Input {
		if in.Type != "image" {
			continue
		}
		_, data, _ := strings.Cut(in.URL, ";base64,")
		b, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, b)
	}
	return out
}

// redPNG is another image than the test drawing: a red rectangle.
func redPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 64, 48))
	for y := 0; y < 48; y++ {
		for x := 0; x < 64; x++ {
			img.Set(x, y, color.RGBA{R: 220, A: 255})
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func hexSum(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// snapshot is a draft folder's files but its lock, held by its dashboard.
func snapshot(t *testing.T, dir string) map[string]string {
	t.Helper()
	es, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, e := range es {
		if e.Name() != "lock" {
			out[e.Name()] = string(rcRead(t, filepath.Join(dir, e.Name())))
		}
	}
	return out
}

// TestTwoDashboards is criterion 33.2 (design D1): two dashboards on one
// project, Claude Code and Codex, each keep a draft of their own.
func TestTwoDashboards(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		files := rcFiles(t)
		d := input.Drafts{Dir: t.TempDir()}
		cwd := t.TempDir()
		wa := &wireClaude{}
		ha := rcRig(t, &d, cwd, &clip{}, wa, []string{files["png"]}, []string{files["gif"]})
		wb := &wireCodex{thread: "th-two"}
		pb := &picker{answers: [][]string{{files["jpg"]}}}
		hb := codexWire(t, wb, Options{Drafts: &d, Cwd: cwd, PickImages: pb.pick})
		ha.send(tea.PasteMsg{Content: "claude draft"})
		upload(t, ha)
		hb.send(tea.PasteMsg{Content: "codex draft"})
		upload(t, hb)
		project := ha.m.State().Context.Project
		if p := hb.m.State().Context.Project; p != project {
			t.Fatalf("projects %q and %q", project, p)
		}
		if n := len(draftSlots(t, d, project)); n != 2 {
			t.Fatalf("%d draft folders; want one per dashboard", n)
		}
		hb.m.draft.mu.Lock()
		bDir := hb.m.draft.slot.Dir()
		hb.m.draft.mu.Unlock()
		before := snapshot(t, bDir)

		// Claude Code's message accepted, then a new draft there.
		ha.send(key(tea.KeyEnter, 0))
		replay(t, ha, wa.last(), probe(t, "p2-claude-conversation.jsonl"), 7, 9)
		if s := ha.m.State(); s.Turn.Phase != state.Running {
			t.Fatalf("turn %q after the acceptance", s.Turn.Phase)
		}
		ha.send(tea.PasteMsg{Content: "claude after"})
		upload(t, ha)
		if got := snapshot(t, bDir); !reflect.DeepEqual(got, before) {
			t.Errorf("Codex's stored draft changed with Claude Code's acceptance:\n got %q\nwant %q", got, before)
		}

		ha.crash()
		hb.crash()
		hc := rcRig(t, &d, cwd, &clip{}, &wireClaude{})
		if hc.m.edit.Text() != "claude after" || !slices.Equal(rcNames(hc.m), []string{"acc-image-upload-test.gif"}) {
			t.Errorf("Claude Code restored %q with %q", hc.m.edit.Text(), rcNames(hc.m))
		}
		if hc.m.ui.Toast != "draft restored · 1 more stored draft of this project" {
			t.Errorf("toast %q", hc.m.ui.Toast)
		}
		hd := codexWire(t, &wireCodex{thread: "th-two-again"}, Options{Drafts: &d, Cwd: cwd})
		if hd.m.edit.Text() != "codex draft" || !slices.Equal(rcNames(hd.m), []string{"acc-image-upload-test.jpg"}) {
			t.Errorf("Codex restored %q with %q", hd.m.edit.Text(), rcNames(hd.m))
		}
	})
}

// TestBackForgetsAttachments is criterion 33.3 (design D2).
func TestBackForgetsAttachments(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		files := rcFiles(t)
		pngA := drawnPNG(t, png.DefaultCompression)
		sumA := sha256.Sum256(pngA)
		d := input.Drafts{Dir: t.TempDir()}
		cwd, other := t.TempDir(), t.TempDir()
		c := &clip{image: pngA}
		w := &wireClaude{noInit: true}
		h := rcRig(t, &d, cwd, c, w, []string{files["png"]})
		if s := h.m.State(); s.Session.Phase != state.Handshaking {
			t.Fatalf("session %q", s.Session.Phase)
		}
		h.send(tea.PasteMsg{Content: "first project's words"})
		upload(t, h)
		h.send(altV())
		if len(h.m.attach) != 2 || len(h.m.pasted) != 1 || len(h.m.thumbs) != 2 {
			t.Fatalf("attached %q, %d pasted, %d thumbnails", rcNames(h.m), len(h.m.pasted), len(h.m.thumbs))
		}
		first := h.m.State().Context.Project
		w.last().exit(3, "Error: cannot start\n")
		h.settle()
		if f := frameText(h); !strings.Contains(f, "The session could not start") {
			t.Fatalf("no failed start:\n%s", f)
		}
		h.send(key(tea.KeyEscape, 0))
		if s := h.m.State(); s.Screen != state.ScreenStartup {
			t.Fatalf("screen %q after Esc", s.Screen)
		}
		if len(h.m.attach) != 0 || len(h.m.pasted) != 0 || len(h.m.thumbs) != 0 || h.m.edit.Text() != "" {
			t.Errorf("after Back: %q, %d pasted, %d thumbnails, editor %q", rcNames(h.m), len(h.m.pasted), len(h.m.thumbs), h.m.edit.Text())
		}
		if got := string(draftFile(t, d, first)); got != "first project's words" {
			t.Errorf("first project's stored text %q", got)
		}
		if got, want := draftList(t, d, first), []string{files["png"], input.PastedEntry("pasted-image-1.png", sumA)}; !slices.Equal(got, want) {
			t.Errorf("first project's stored images %q, want %q", got, want)
		}
		if b := rcRead(t, filepath.Join(draftSlot(t, d, first), hexSum(pngA)+".png")); !bytes.Equal(b, pngA) {
			t.Error("the pasted image's stored copy differs")
		}

		w.mu.Lock()
		w.noInit = false
		w.mu.Unlock()
		h.m.path.SetText(other)
		h.send(key(tea.KeyEnter, 0))
		h.send(tea.WindowSizeMsg{Width: 160, Height: 60})
		if s := h.m.State(); s.Session.Phase != state.Ready || s.Context.Project == first {
			t.Fatalf("second session %q in %q", s.Session.Phase, s.Context.Project)
		}
		if len(h.m.attach) != 0 {
			t.Errorf("the other project's session starts with %q", rcNames(h.m))
		}
		h.send(tea.PasteMsg{Content: "other project's words"})
		h.send(key(tea.KeyEnter, 0))
		if lines := w.written("user"); len(lines) == 0 {
			t.Fatal("the other project's message was not written")
		}
		if n := len(rcSent(t, w)); n != 0 {
			t.Errorf("%d images sent with the other project's message", n)
		}

		// The first project again, in a new dashboard: restored once.
		h2 := rcRig(t, &d, cwd, c, &wireClaude{})
		if h2.m.edit.Text() != "first project's words" || !slices.Equal(rcNames(h2.m), []string{"acc-image-upload-test.png", "pasted-image-1.png"}) {
			t.Errorf("first project restored %q with %q", h2.m.edit.Text(), rcNames(h2.m))
		}
		if h2.m.ui.Toast != "draft restored" {
			t.Errorf("toast %q", h2.m.ui.Toast)
		}

		// Back, then the same project in the same dashboard: restored once.
		cwd3 := t.TempDir()
		w3 := &wireClaude{noInit: true}
		h3 := rcRig(t, &d, cwd3, c, w3, []string{files["jpg"]})
		h3.send(tea.PasteMsg{Content: "again"})
		upload(t, h3)
		h3.send(altV())
		w3.last().exit(3, "Error: cannot start\n")
		h3.settle()
		h3.send(key(tea.KeyEscape, 0))
		w3.mu.Lock()
		w3.noInit = false
		w3.mu.Unlock()
		h3.send(key(tea.KeyEnter, 0))
		h3.send(tea.WindowSizeMsg{Width: 160, Height: 60})
		if h3.m.edit.Text() != "again" || !slices.Equal(rcNames(h3.m), []string{"acc-image-upload-test.jpg", "pasted-image-1.png"}) {
			t.Errorf("same project again: %q with %q", h3.m.edit.Text(), rcNames(h3.m))
		}
		if h3.m.ui.Toast != "draft restored" {
			t.Errorf("toast %q", h3.m.ui.Toast)
		}
	})
}

// TestStaleRestoreDropped: a stored draft's images read back after that
// draft was let go (Back while they were read) are not attached to the
// editor; the live draft's are (design D2).
func TestStaleRestoreDropped(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		files := rcFiles(t)
		d := input.Drafts{Dir: t.TempDir()}
		h := rcRig(t, &d, t.TempDir(), &clip{}, &wireClaude{})
		img, _, err := input.InspectPreview(files["png"], input.ImageLimits{})
		if err != nil {
			t.Fatal(err)
		}
		h.send(inspectedMsg{restore: true, store: &draftStore{}, imgs: []input.Image{img}, thumbs: map[string]thumbSixel{}})
		if len(h.m.attach) != 0 {
			t.Errorf("a draft let go gave %q to the editor", rcNames(h.m))
		}
		h.send(inspectedMsg{restore: true, store: h.m.draft, imgs: []input.Image{img}, thumbs: map[string]thumbSixel{}})
		if len(h.m.attach) != 1 {
			t.Errorf("the live draft's images: %q", rcNames(h.m))
		}
	})
}

// TestAttachSavesNow is criterion 33.4 (design D3): the images are stored
// at once, the text when the timer fires.
func TestAttachSavesNow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		files := rcFiles(t)
		pngA := drawnPNG(t, png.DefaultCompression)
		sumA := sha256.Sum256(pngA)
		d := input.Drafts{Dir: t.TempDir()}
		h := rcRig(t, &d, t.TempDir(), &clip{image: pngA}, &wireClaude{}, []string{files["png"], files["jpg"]})
		fired := 0
		h.m.opt.After = func(_ time.Duration, msg tea.Msg) tea.Cmd {
			if _, ok := msg.(draftSaveMsg); ok {
				fired++
				return func() tea.Msg { return nil } // the timer never fires
			}
			return nil
		}
		project := h.m.State().Context.Project
		upload(t, h)
		want := []string{files["png"], files["jpg"]}
		if got := draftList(t, d, project); !slices.Equal(got, want) {
			t.Fatalf("after the dialog: stored %q", got)
		}
		h.send(altV())
		want = append(want, input.PastedEntry("pasted-image-1.png", sumA))
		if got := draftList(t, d, project); !slices.Equal(got, want) {
			t.Fatalf("after Alt+V: stored %q", got)
		}
		if b := rcRead(t, filepath.Join(draftSlot(t, d, project), hexSum(pngA)+".png")); !bytes.Equal(b, pngA) {
			t.Fatal("the pasted image's stored copy differs")
		}
		x, y := at(t, h, "✕", 0)
		clickAt(t, h, x, y)
		if got := draftList(t, d, project); !slices.Equal(got, want[1:]) {
			t.Fatalf("after the first chip's ✕: stored %q", got)
		}
		if fired != 0 {
			t.Errorf("%d timers armed for the images", fired)
		}
		h.send(tea.PasteMsg{Content: "typed"})
		if got := draftFile(t, d, project); got != nil || fired != 1 {
			t.Errorf("text stored before its timer: %q, %d timers", got, fired)
		}
		h.send(draftSaveMsg{})
		if got := string(draftFile(t, d, project)); got != "typed" {
			t.Errorf("stored text %q once the timer fired", got)
		}
	})
}

// TestDraftWhileWaiting is criterion 33.5 (design D4).
func TestDraftWhileWaiting(t *testing.T) {
	setup := func(t *testing.T) (*harness, *wireClaude, input.Drafts, string, string, string) {
		files := rcFiles(t)
		a, b := files["png"], files["jpg"]
		d := input.Drafts{Dir: t.TempDir()}
		cwd := t.TempDir()
		w := &wireClaude{}
		h := rcRig(t, &d, cwd, &clip{}, w, []string{a}, []string{a, b})
		h.send(tea.PasteMsg{Content: "sent"})
		upload(t, h)
		h.send(key(tea.KeyEnter, 0))
		if s := h.m.State(); s.Turn.Phase != state.Submitted {
			t.Fatalf("turn %q after Enter", s.Turn.Phase)
		}
		h.send(tea.PasteMsg{Content: "typed"})
		upload(t, h)
		project := h.m.State().Context.Project
		if got := string(draftFile(t, d, project)); got != "sent\ntyped" {
			t.Errorf("stored text %q while the message waits", got)
		}
		if got := draftList(t, d, project); !slices.Equal(got, []string{a, b}) {
			t.Errorf("stored images %q while the message waits; want A then B, A once", got)
		}
		return h, w, d, cwd, a, b
	}
	t.Run("crash", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			h, _, d, cwd, a, b := setup(t)
			h.crash()
			h2 := rcRig(t, &d, cwd, &clip{}, &wireClaude{})
			if h2.m.edit.Text() != "sent\ntyped" || !slices.Equal(rcNames(h2.m), []string{filepath.Base(a), filepath.Base(b)}) {
				t.Errorf("restored %q with %q", h2.m.edit.Text(), rcNames(h2.m))
			}
		})
	})
	t.Run("accepted", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			h, w, d, _, a, b := setup(t)
			replay(t, h, w.last(), probe(t, "p2-claude-conversation.jsonl"), 7, 9)
			project := h.m.State().Context.Project
			if got := string(draftFile(t, d, project)); got != "typed" {
				t.Errorf("stored text %q after the acceptance", got)
			}
			if got := draftList(t, d, project); !slices.Equal(got, []string{a, b}) {
				t.Errorf("stored images %q after the acceptance", got)
			}
		})
	})
	t.Run("second Enter", func(t *testing.T) {
		// Enter again while the message waits: refused, the waiting
		// message keeps its place in the state and the stored draft, its
		// pasted image too.
		synctest.Test(t, func(t *testing.T) {
			pngA := drawnPNG(t, png.DefaultCompression)
			d := input.Drafts{Dir: t.TempDir()}
			cwd := t.TempDir()
			h := rcRig(t, &d, cwd, &clip{image: pngA}, &wireClaude{})
			h.send(tea.PasteMsg{Content: "sent"})
			h.send(altV())
			h.send(key(tea.KeyEnter, 0))
			h.send(tea.PasteMsg{Content: "second"})
			h.send(key(tea.KeyEnter, 0))
			s := h.m.State()
			if s.Turn.Notice != "turn running: wait, or interrupt" || s.Turn.Draft != "sent" || len(s.Turn.DraftImages) != 1 {
				t.Fatalf("after the second Enter: notice %q, waiting %q with %d images", s.Turn.Notice, s.Turn.Draft, len(s.Turn.DraftImages))
			}
			project := s.Context.Project
			if got := string(draftFile(t, d, project)); got != "sent\nsecond" {
				t.Errorf("stored text %q", got)
			}
			if got := draftList(t, d, project); len(got) != 1 {
				t.Errorf("stored images %q", got)
			}
			if _, err := os.Stat(filepath.Join(draftSlot(t, d, project), hexSum(pngA)+".png")); err != nil {
				t.Errorf("the waiting message's pasted copy: %v", err)
			}
			h.crash()
			h2 := rcRig(t, &d, cwd, &clip{}, &wireClaude{})
			if h2.m.edit.Text() != "sent\nsecond" || !slices.Equal(rcNames(h2.m), []string{"pasted-image-1.png"}) {
				t.Errorf("restored %q with %q", h2.m.edit.Text(), rcNames(h2.m))
			}
		})
	})
	t.Run("image-only given back", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			files := rcFiles(t)
			d := input.Drafts{Dir: t.TempDir()}
			w := &wireClaude{}
			h := rcRig(t, &d, t.TempDir(), &clip{}, w, []string{files["png"]})
			upload(t, h)
			h.send(key(tea.KeyEnter, 0))
			if s := h.m.State(); s.Turn.Phase != state.Submitted {
				t.Fatalf("turn %q after Enter", s.Turn.Phase)
			}
			h.send(typed('x'))
			w.last().exit(3, "Error: something broke\n")
			h.settle()
			if h.m.edit.Text() != "x" || !slices.Equal(rcNames(h.m), []string{"acc-image-upload-test.png"}) {
				t.Errorf("given back: editor %q with %q", h.m.edit.Text(), rcNames(h.m))
			}
		})
	})
}

// TestDamagedImagesList is criterion 33.7 (design D6).
func TestDamagedImagesList(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		files := rcFiles(t)
		d := input.Drafts{Dir: t.TempDir()}
		cwd := t.TempDir()
		h := rcRig(t, &d, cwd, &clip{}, &wireClaude{}, []string{files["png"]})
		h.send(tea.PasteMsg{Content: "precious text"})
		upload(t, h)
		project := h.m.State().Context.Project
		dir := draftSlot(t, d, project)
		h.crash()
		damaged := `["C:\\half`
		if err := os.WriteFile(filepath.Join(dir, "images.json"), []byte(damaged), 0o600); err != nil {
			t.Fatal(err)
		}
		h2 := rcRig(t, &d, cwd, &clip{}, &wireClaude{})
		if h2.m.edit.Text() != "precious text" || len(h2.m.attach) != 0 {
			t.Errorf("restored %q with %q", h2.m.edit.Text(), rcNames(h2.m))
		}
		if h2.m.ui.Toast != "draft restored; its images list was unreadable and was set aside" {
			t.Errorf("toast %q", h2.m.ui.Toast)
		}
		if got := string(rcRead(t, filepath.Join(dir, "images.json.unreadable"))); got != damaged {
			t.Errorf("set aside %q", got)
		}
		if h2.m.draft == nil {
			t.Fatal("drafts off")
		}
		h2.send(typed('!'))
		if got := string(draftFile(t, d, project)); got != "precious text!" {
			t.Errorf("stored %q after a change", got)
		}
	})
}

// TestChangedFile is criterion 33.8 (design D7).
func TestChangedFile(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		files := rcFiles(t)
		mine := filepath.Join(t.TempDir(), "shot.png")
		original := rcRead(t, files["png"])
		if err := os.WriteFile(mine, original, 0o644); err != nil {
			t.Fatal(err)
		}
		d := input.Drafts{Dir: t.TempDir()}
		w := &wireClaude{}
		h := rcRig(t, &d, t.TempDir(), &clip{}, w, []string{mine}, []string{mine})
		h.send(tea.PasteMsg{Content: "which image?"})
		upload(t, h)
		before := h.m.thumbs[mine]
		red := redPNG(t)
		if err := os.WriteFile(mine, red, 0o644); err != nil {
			t.Fatal(err)
		}
		h.send(key(tea.KeyEnter, 0))
		if n := len(w.written("user")); n != 0 {
			t.Errorf("%d messages written with the file changed", n)
		}
		if got := h.m.State().Turn.Notice; got != "the message could not be sent: shot.png: changed since it was attached; attached again as it is now" {
			t.Errorf("notice %q", got)
		}
		if h.m.edit.Text() != "which image?" || !slices.Equal(rcNames(h.m), []string{"shot.png"}) {
			t.Errorf("after the refusal: %q with %q", h.m.edit.Text(), rcNames(h.m))
		}
		if h.m.ui.Toast != "shot.png: updated" || h.m.thumbs[mine] == before {
			t.Errorf("attached again: toast %q, the old thumbnail %v", h.m.ui.Toast, h.m.thumbs[mine] == before)
		}
		h.send(key(tea.KeyEnter, 0))
		if sent := rcSent(t, w); len(sent) != 1 || !bytes.Equal(sent[0], red) {
			t.Errorf("sent %d images, not the new bytes", len(sent))
		}
	})
	t.Run("attached again by hand", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			files := rcFiles(t)
			mine := filepath.Join(t.TempDir(), "shot.png")
			if err := os.WriteFile(mine, rcRead(t, files["png"]), 0o644); err != nil {
				t.Fatal(err)
			}
			d := input.Drafts{Dir: t.TempDir()}
			h := rcRig(t, &d, t.TempDir(), &clip{}, &wireClaude{}, []string{mine}, []string{mine}, []string{mine})
			upload(t, h)
			upload(t, h)
			if h.m.ui.Toast != "shot.png: already attached" {
				t.Errorf("unchanged, attached again: toast %q", h.m.ui.Toast)
			}
			before := h.m.thumbs[mine]
			if err := os.WriteFile(mine, redPNG(t), 0o644); err != nil {
				t.Fatal(err)
			}
			upload(t, h)
			if h.m.ui.Toast != "shot.png: updated" || len(h.m.attach) != 1 || h.m.thumbs[mine] == before {
				t.Errorf("changed, attached again: toast %q, %q", h.m.ui.Toast, rcNames(h.m))
			}
		})
	})
	t.Run("saved again unchanged", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			files := rcFiles(t)
			same := rcRead(t, files["jpg"])
			d := input.Drafts{Dir: t.TempDir()}
			w := &wireClaude{}
			h := rcRig(t, &d, t.TempDir(), &clip{}, w, []string{files["jpg"]})
			upload(t, h)
			if err := os.WriteFile(files["jpg"], same, 0o644); err != nil {
				t.Fatal(err)
			}
			h.send(tea.PasteMsg{Content: "the same"})
			h.send(key(tea.KeyEnter, 0))
			if sent := rcSent(t, w); len(sent) != 1 || !bytes.Equal(sent[0], same) {
				t.Errorf("sent %d images; want the file as it is", len(sent))
			}
		})
	})
}

// TestPastedKept is criterion 33.9 (design D8).
func TestPastedKept(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		pngA, pngB := drawnPNG(t, png.DefaultCompression), redPNG(t)
		sumA := sha256.Sum256(pngA)
		d := input.Drafts{Dir: t.TempDir()}
		cwd := t.TempDir()
		c := &clip{image: pngA}
		h := rcRig(t, &d, cwd, c, &wireClaude{})
		h.send(altV())
		project := h.m.State().Context.Project
		dir := draftSlot(t, d, project)
		keptA := filepath.Join(dir, hexSum(pngA)+".png")
		if b := rcRead(t, keptA); !bytes.Equal(b, pngA) {
			t.Fatal("the stored copy differs from the pasted bytes")
		}
		if got := draftList(t, d, project); !slices.Equal(got, []string{input.PastedEntry("pasted-image-1.png", sumA)}) {
			t.Fatalf("stored images %q", got)
		}
		_ = filepath.Walk(d.Dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || path == keptA {
				return err
			}
			if b, _ := os.ReadFile(path); bytes.Contains(b, pngA[8:72]) || bytes.Contains(b, []byte(base64.StdEncoding.EncodeToString(pngA)[:64])) {
				t.Errorf("%s holds the pasted image too", path)
			}
			return nil
		})

		h.crash()
		w2 := &wireClaude{}
		h2 := rcRig(t, &d, cwd, c, w2)
		if !slices.Equal(rcNames(h2.m), []string{"pasted-image-1.png"}) || h2.m.thumbs["pasted:pasted-image-1.png"] == (thumbSixel{}) {
			t.Fatalf("restored %q, thumbnail %v", rcNames(h2.m), h2.m.thumbs["pasted:pasted-image-1.png"] != thumbSixel{})
		}
		h2.send(key(tea.KeyEnter, 0))
		if sent := rcSent(t, w2); len(sent) != 1 || !bytes.Equal(sent[0], pngA) {
			t.Fatalf("sent %d images, not the pasted bytes", len(sent))
		}
		if _, err := os.Stat(keptA); err != nil {
			t.Errorf("the waiting message's image not kept: %v", err)
		}
		replay(t, h2, w2.last(), probe(t, "p2-claude-conversation.jsonl"), 7, 9)
		if _, err := os.Stat(keptA); !os.IsNotExist(err) {
			t.Errorf("the accepted message's image kept: %v", err)
		}

		c.mu.Lock()
		c.image = pngB
		c.mu.Unlock()
		h2.send(altV())
		keptB := filepath.Join(dir, hexSum(pngB)+".png")
		if !slices.Equal(rcNames(h2.m), []string{"pasted-image-2.png"}) {
			t.Fatalf("the next paste: %q", rcNames(h2.m))
		}
		if _, err := os.Stat(keptB); err != nil {
			t.Fatalf("the next paste not kept: %v", err)
		}
		h2.send(key('x', tea.ModCtrl|tea.ModShift))
		if _, err := os.Stat(keptB); !os.IsNotExist(err) {
			t.Errorf("a removed image's copy kept: %v", err)
		}

		// A stored copy altered on disk.
		h2.send(altV())
		h2.crash()
		if err := os.WriteFile(keptB, append(append([]byte(nil), pngB...), '!'), 0o600); err != nil {
			t.Fatal(err)
		}
		h3 := rcRig(t, &d, cwd, c, &wireClaude{})
		if len(h3.m.attach) != 0 || !strings.Contains(h3.m.ui.Toast, "pasted-image-3.png: its stored copy is damaged") {
			t.Errorf("an altered copy: %q, toast %q", rcNames(h3.m), h3.m.ui.Toast)
		}

		// A plan 32 entry.
		d2 := input.Drafts{Dir: t.TempDir()}
		cwd2 := t.TempDir()
		key2 := input.DraftKey(cwd2)
		if err := os.WriteFile(filepath.Join(d2.Dir, key2+".txt"), []byte("legacy"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d2.Dir, key2+".images.json"), []byte(`["pasted:pasted-image-1.png"]`), 0o600); err != nil {
			t.Fatal(err)
		}
		h4 := rcRig(t, &d2, cwd2, c, &wireClaude{})
		if h4.m.edit.Text() != "legacy" || h4.m.ui.Toast != "draft restored; pasted-image-1.png was pasted, not kept" {
			t.Errorf("a plan 32 draft: %q, toast %q", h4.m.edit.Text(), h4.m.ui.Toast)
		}
	})
}

// TestDraftSaveRetried: a save Windows refuses for longer than the store's
// own retries is said and tried again a second later, so a draft refused
// once is not left behind until the next change (plan 33).
func TestDraftSaveRetried(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows refuses the rename")
	}
	synctest.Test(t, func(t *testing.T) {
		files := rcFiles(t)
		d := input.Drafts{Dir: t.TempDir()}
		h := rcRig(t, &d, t.TempDir(), &clip{}, &wireClaude{}, []string{files["png"]}, []string{files["jpg"]})
		retries := 0
		h.m.opt.After = func(_ time.Duration, msg tea.Msg) tea.Cmd {
			if _, ok := msg.(draftRetryMsg); ok {
				retries++
			}
			return nil
		}
		upload(t, h)
		project := h.m.State().Context.Project
		list := filepath.Join(draftSlot(t, d, project), "images.json")
		f, err := os.Open(list) // read without sharing its deletion: the rename over it refused
		if err != nil {
			t.Fatal(err)
		}
		upload(t, h)
		time.Sleep(2 * time.Second) // the store's own retries, about a second, run out
		h.settle()
		if !strings.HasPrefix(h.m.ui.Toast, "draft not saved: ") || retries != 1 {
			f.Close()
			t.Fatalf("toast %q, %d retries armed", h.m.ui.Toast, retries)
		}
		if got := draftList(t, d, project); !slices.Equal(got, []string{files["png"]}) {
			t.Fatalf("stored %q while refused", got)
		}
		f.Close()
		h.send(draftRetryMsg{})
		if got := draftList(t, d, project); !slices.Equal(got, []string{files["png"], files["jpg"]}) {
			t.Errorf("stored %q after the retry", got)
		}
	})
}

// TestDraftInUse: a stored draft another program holds past the retries is
// said, not skipped in silence, and taken by the next start (plan 33).
func TestDraftInUse(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("a held file is Windows'")
	}
	synctest.Test(t, func(t *testing.T) {
		d := input.Drafts{Dir: t.TempDir()}
		cwd := t.TempDir()
		h := rcRig(t, &d, cwd, &clip{}, &wireClaude{})
		h.send(tea.PasteMsg{Content: "held elsewhere"})
		dir := draftSlot(t, d, h.m.State().Context.Project)
		h.crash()
		f, err := os.Open(filepath.Join(dir, "lock"))
		if err != nil {
			t.Fatal(err)
		}
		h2 := rcRig(t, &d, cwd, &clip{}, &wireClaude{})
		time.Sleep(2 * time.Second) // the take's retries, about a second, run out
		h2.settle()
		if h2.m.edit.Text() != "" || h2.m.ui.Toast != "a stored draft of this project is in use: the next start takes it" {
			t.Errorf("editor %q, toast %q", h2.m.edit.Text(), h2.m.ui.Toast)
		}
		f.Close()
		h2.crash()
		h3 := rcRig(t, &d, cwd, &clip{}, &wireClaude{})
		if h3.m.edit.Text() != "held elsewhere" {
			t.Errorf("the next start: %q, toast %q", h3.m.edit.Text(), h3.m.ui.Toast)
		}
	})
}

// noisePNG is a PNG of about the given bytes that does not compress.
func noisePNG(t *testing.T, seed int64, w, h int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	r := rand.New(rand.NewSource(seed))
	r.Read(img.Pix)
	for i := 3; i < len(img.Pix); i += 4 {
		img.Pix[i] = 255
	}
	var b bytes.Buffer
	if err := (&png.Encoder{CompressionLevel: png.BestSpeed}).Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// TestRestoreOverLimits: a Codex draft of two pasted images, 7 MB in all,
// restored in Claude Code, which sends 7.4 MB of images a message at
// most: both attached and kept, the toast says it, the send refuses them
// with the reason, nothing deleted (plan 33).
func TestRestoreOverLimits(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		a, b := noisePNG(t, 1, 1300, 1100), noisePNG(t, 2, 1300, 1100)
		if n := len(a) + len(b); n <= 7_400_000 || len(a) > 7_400_000 {
			t.Fatalf("the images are %d and %d bytes: want each under 7.4 MB, both over", len(a), len(b))
		}
		d := input.Drafts{Dir: t.TempDir()}
		cwd := t.TempDir()
		c := &clip{image: a}
		h := codexWire(t, &wireCodex{thread: "th-limits"}, Options{Drafts: &d, Cwd: cwd, Clipboard: c})
		h.send(altV())
		c.mu.Lock()
		c.image = b
		c.mu.Unlock()
		h.send(altV())
		if len(h.m.attach) != 2 {
			t.Fatalf("Codex attached %q", rcNames(h.m))
		}
		project := h.m.State().Context.Project
		dir := draftSlot(t, d, project)
		h.crash()
		w := &wireClaude{}
		h2 := rcRig(t, &d, cwd, &clip{}, w)
		if !slices.Equal(rcNames(h2.m), []string{"pasted-image-1.png", "pasted-image-2.png"}) {
			t.Fatalf("Claude Code restored %q, toast %q", rcNames(h2.m), h2.m.ui.Toast)
		}
		// The note needs the session's limits, known once it has started:
		// a draft read back before that is told at the send instead.
		if got := h2.m.ui.Toast; got != "the draft's images: more than this session sends in a message: remove some before sending" && got != "draft restored" {
			t.Errorf("toast %q", got)
		}
		t.Logf("restored with the toast %q", h2.m.ui.Toast)
		h2.send(tea.PasteMsg{Content: "x"})
		h2.send(key(tea.KeyEnter, 0))
		if n := len(w.written("user")); n != 0 {
			t.Errorf("%d messages written over the limits", n)
		}
		if got := h2.m.State().Turn.Notice; got != "the message could not be sent: the images are over 7.4 MB a message" {
			t.Errorf("notice %q", got)
		}
		for _, p := range [][]byte{a, b} {
			if _, err := os.Stat(filepath.Join(dir, hexSum(p)+".png")); err != nil {
				t.Errorf("a pasted copy deleted: %v", err)
			}
		}
		if len(h2.m.attach) != 2 {
			t.Errorf("after the refusal: %q", rcNames(h2.m))
		}
	})
}

// TestRestoreOverCount: a stored draft of three files restored in a
// session that takes two images a message: all three attached and said,
// the send refused with the count, sent once one is removed (plan 33).
func TestRestoreOverCount(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		files := rcFiles(t)
		d := input.Drafts{Dir: t.TempDir()}
		cwd := t.TempDir()
		h := rcRig(t, &d, cwd, &clip{}, &wireClaude{}, []string{files["png"], files["jpg"], files["gif"]})
		upload(t, h)
		h.crash()
		cli := &scriptCLI{id: "claude-code", sess: newScriptSession()}
		cli.sess.maxCount = 2
		h2 := newHarness(t, Options{Providers: []provider.Provider{cli}, Cwd: cwd, Drafts: &d})
		h2.settle()
		h2.m.ui.Choice = "claude-code"
		h2.send(key(tea.KeyEnter, 0))
		cli.sess.events <- event.Event{Provider: "claude-code", Kind: event.SessionSpawned, Origin: event.Observed, Payload: event.Spawned{Argv: []string{`C:\Tools\claude.exe`}}}
		cli.sess.events <- event.Event{Provider: "claude-code", Kind: event.SessionReady, Origin: event.Observed, Payload: event.Ready{Detail: "scripted"}}
		h2.settle()
		h2.send(tea.WindowSizeMsg{Width: 160, Height: 60})
		if len(h2.m.attach) != 3 {
			t.Fatalf("restored %q", rcNames(h2.m))
		}
		t.Logf("restored with the toast %q", h2.m.ui.Toast)
		h2.send(key(tea.KeyEnter, 0))
		if got := h2.m.State().Turn.Notice; got != "the message could not be sent: 3 images, over 2 a message" || len(cli.sess.messages()) != 0 {
			t.Fatalf("notice %q, %d messages", got, len(cli.sess.messages()))
		}
		h2.send(key('x', tea.ModCtrl|tea.ModShift))
		h2.send(key(tea.KeyEnter, 0))
		if msgs := cli.sess.messages(); len(msgs) != 1 || len(msgs[0].Images) != 2 {
			t.Errorf("after one removed: %d messages", len(msgs))
		}
	})
}

// TestCodexResumeImages: a Codex session that crashes with images attached
// and not sent, or with a message of images waiting for its acceptance, is
// resumed by Ctrl+R (thread/resume, the same thread): its images, a pasted
// one among them, reach the resumed thread byte for byte (plan 33, the
// owner's question on resuming a session).
func TestCodexResumeImages(t *testing.T) {
	setup := func(t *testing.T, auto bool) (*harness, *wireCodex, [][]byte) {
		files := rcFiles(t)
		pngA := redPNG(t)
		d := input.Drafts{Dir: t.TempDir()}
		w := &wireCodex{thread: "th-resume", auto: auto}
		p := &picker{answers: [][]string{{files["png"], files["jpg"]}}}
		h := codexWire(t, w, Options{Drafts: &d, PickImages: p.pick, Clipboard: &clip{image: pngA}})
		h.send(PrimaryDeviceAttributesEvent{61, 4, 6})
		h.send(tea.PasteMsg{Content: "look at these"})
		upload(t, h)
		h.send(altV())
		if !slices.Equal(rcNames(h.m), []string{"acc-image-upload-test.png", "acc-image-upload-test.jpg", "pasted-image-1.png"}) {
			t.Fatalf("attached %q", rcNames(h.m))
		}
		return h, w, [][]byte{rcRead(t, files["png"]), rcRead(t, files["jpg"]), pngA}
	}
	sentAll := func(t *testing.T, w *wireCodex, want [][]byte) {
		t.Helper()
		sent := codexSent(t, w)
		if len(sent) != len(want) {
			t.Fatalf("the resumed thread received %d images, want %d", len(sent), len(want))
		}
		for i := range want {
			if !bytes.Equal(sent[i], want[i]) {
				t.Errorf("image %d is not the original bytes", i)
			}
		}
	}
	resume := func(t *testing.T, h *harness, w *wireCodex) {
		t.Helper()
		h.send(key('r', tea.ModCtrl))
		if s := h.m.State(); s.Session.Phase != state.Ready || s.Session.ID != "th-resume" || len(w.method("thread/resume")) != 1 || len(w.spawns()) != 2 {
			t.Fatalf("after Ctrl+R: %q (%q), %d resumes, %d spawns", s.Session.Phase, s.Session.ID, len(w.method("thread/resume")), len(w.spawns()))
		}
	}
	t.Run("unsent", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			h, w, want := setup(t, true)
			w.last().exit(3, "Error: something broke\n")
			h.settle()
			if s := h.m.State(); s.Session.Phase != state.Crashed {
				t.Fatalf("session %q", s.Session.Phase)
			}
			if h.m.edit.Text() != "look at these" || len(h.m.attach) != 3 || len(h.m.thumbs) != 3 || len(h.m.pasted) != 1 {
				t.Errorf("after the crash: %q with %q, %d thumbnails, %d pasted", h.m.edit.Text(), rcNames(h.m), len(h.m.thumbs), len(h.m.pasted))
			}
			resume(t, h, w)
			h.send(key(tea.KeyEnter, 0))
			sentAll(t, w, want)
		})
	})
	t.Run("waiting", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			h, w, want := setup(t, false) // turn/start never answered: the message waits
			h.send(key(tea.KeyEnter, 0))
			if s := h.m.State(); s.Turn.Phase != state.Submitted {
				t.Fatalf("turn %q", s.Turn.Phase)
			}
			w.last().exit(3, "Error: something broke\n")
			h.settle()
			if h.m.edit.Text() != "look at these" || !slices.Equal(rcNames(h.m), []string{"acc-image-upload-test.png", "acc-image-upload-test.jpg", "pasted-image-1.png"}) {
				t.Errorf("given back: %q with %q", h.m.edit.Text(), rcNames(h.m))
			}
			w.mu.Lock()
			w.auto = true
			w.mu.Unlock()
			resume(t, h, w)
			h.send(key(tea.KeyEnter, 0))
			if n := len(w.method("turn/start")); n != 2 {
				t.Fatalf("%d turn/start written, want the first and the one after Ctrl+R", n)
			}
			sentAll(t, w, want)
		})
	})
}

// TestRestoreSlowPath: a stored draft whose image lies on a path that does
// not answer (a network share gone offline), simulated by a read held
// until the test lets it go, no network involved. While it hangs, what is
// typed and attached is stored with the stored draft's own images, in
// their order: a crash then, a quit then, or Back then, loses nothing
// (plan 33, the owner's question after the evaluation).
func TestRestoreSlowPath(t *testing.T) {
	type end struct {
		name string
		do   func(t *testing.T, h *harness)
	}
	for _, e := range []end{
		{"crash", func(t *testing.T, h *harness) { h.crash() }},
		{"quit", func(t *testing.T, h *harness) {
			if err := h.m.letDraftGo(); err != nil {
				t.Fatal(err)
			}
		}},
		{"Back", func(t *testing.T, h *harness) { _ = h.m.closeSession() }},
	} {
		t.Run(e.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				files := rcFiles(t)
				slow := files["jpg"]
				pngA := redPNG(t)
				d := input.Drafts{Dir: t.TempDir()}
				cwd := t.TempDir()
				h := rcRig(t, &d, cwd, &clip{image: pngA}, &wireClaude{}, []string{files["png"], slow})
				h.send(tea.PasteMsg{Content: "before"})
				upload(t, h)
				h.send(altV())
				h.crash()

				release := make(chan struct{})
				inspectPreview = func(path string, lim input.ImageLimits) (input.Image, image.Image, error) {
					if path == slow {
						<-release // the share does not answer
					}
					return input.InspectPreview(path, lim)
				}
				defer func() { inspectPreview = input.InspectPreview }()
				h2 := rcRig(t, &d, cwd, &clip{}, &wireClaude{}, []string{files["gif"]})
				if len(h2.m.unread) != 3 || h2.m.edit.Text() != "before" || len(h2.m.attach) != 0 {
					t.Fatalf("while the read hangs: %d unread, %q with %q", len(h2.m.unread), h2.m.edit.Text(), rcNames(h2.m))
				}
				h2.send(tea.PasteMsg{Content: " and after"})
				upload(t, h2)
				project := h2.m.State().Context.Project
				t.Logf("stored while the read hangs: %q, %d images", draftFile(t, d, project), len(draftList(t, d, project)))
				e.do(t, h2)
				close(release)
				h2.settle()

				h3 := rcRig(t, &d, cwd, &clip{}, &wireClaude{})
				want := []string{"acc-image-upload-test.png", "acc-image-upload-test.jpg", "pasted-image-1.png", "acc-image-upload-test.gif"}
				if h3.m.edit.Text() != "before and after" || !slices.Equal(rcNames(h3.m), want) {
					t.Errorf("restored after the %s: %q with %q; want %q with %q", e.name, h3.m.edit.Text(), rcNames(h3.m), "before and after", want)
				}
			})
		})
	}
}

// TestRestoreSlowPathAnswers: the path answers at last while the dashboard
// runs: the stored draft's images take their place first, the one attached
// meanwhile after them, in the editor and the stored draft (plan 33).
func TestRestoreSlowPathAnswers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		files := rcFiles(t)
		slow := files["jpg"]
		d := input.Drafts{Dir: t.TempDir()}
		cwd := t.TempDir()
		h := rcRig(t, &d, cwd, &clip{image: redPNG(t)}, &wireClaude{}, []string{files["png"], slow})
		h.send(tea.PasteMsg{Content: "before"})
		upload(t, h)
		h.send(altV())
		h.crash()
		release := make(chan struct{})
		inspectPreview = func(path string, lim input.ImageLimits) (input.Image, image.Image, error) {
			if path == slow {
				<-release
			}
			return input.InspectPreview(path, lim)
		}
		defer func() { inspectPreview = input.InspectPreview }()
		h2 := rcRig(t, &d, cwd, &clip{}, &wireClaude{}, []string{files["gif"]})
		upload(t, h2)
		close(release)
		h2.settle()
		want := []string{"acc-image-upload-test.png", "acc-image-upload-test.jpg", "pasted-image-1.png", "acc-image-upload-test.gif"}
		if !slices.Equal(rcNames(h2.m), want) || len(h2.m.unread) != 0 {
			t.Errorf("once read: %q, %d unread; want %q", rcNames(h2.m), len(h2.m.unread), want)
		}
		project := h2.m.State().Context.Project
		if got := draftList(t, d, project); len(got) != 4 || got[0] != files["png"] || got[1] != slow || got[3] != files["gif"] {
			t.Errorf("stored %q", got)
		}
	})
}

// TestRestoreSlowPathWaiting: while the stored draft's images are read
// back, a message is sent and waits for its acceptance: the stored order
// is the waiting message's image, then the stored draft's, before and after
// the read answers, and so are the chips the message's failure gives back;
// pasting again an image the stored draft holds is "already attached"
// (plan 33, the review's FAIL 8).
func TestRestoreSlowPathWaiting(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		files := rcFiles(t)
		slow := files["jpg"]
		pngA := redPNG(t)
		d := input.Drafts{Dir: t.TempDir()}
		cwd := t.TempDir()
		h := rcRig(t, &d, cwd, &clip{image: pngA}, &wireClaude{}, []string{files["png"], slow})
		h.send(tea.PasteMsg{Content: "stored"})
		upload(t, h)
		h.send(altV())
		h.crash()
		release := make(chan struct{})
		inspectPreview = func(path string, lim input.ImageLimits) (input.Image, image.Image, error) {
			if path == slow {
				<-release
			}
			return input.InspectPreview(path, lim)
		}
		defer func() { inspectPreview = input.InspectPreview }()
		w := &wireClaude{}
		h2 := rcRig(t, &d, cwd, &clip{image: pngA}, w, []string{files["gif"]})
		h2.send(altV())
		if !strings.HasSuffix(h2.m.ui.Toast, "pasted-image-1.png: already attached") || len(h2.m.attach) != 0 {
			t.Errorf("the stored image pasted again: toast %q, %q", h2.m.ui.Toast, rcNames(h2.m))
		}
		h2.m.edit.Clear()
		upload(t, h2)
		h2.send(tea.PasteMsg{Content: "sent meanwhile"})
		h2.send(key(tea.KeyEnter, 0))
		if s := h2.m.State(); s.Turn.Phase != state.Submitted {
			t.Fatalf("turn %q", s.Turn.Phase)
		}
		project := h2.m.State().Context.Project
		wantList := []string{files["gif"], files["png"], slow}
		before := draftList(t, d, project)
		if len(before) != 4 || !slices.Equal(before[:3], wantList) {
			t.Errorf("stored while the read hangs: %q", before)
		}
		close(release)
		h2.settle()
		if after := draftList(t, d, project); !slices.Equal(after, before) {
			t.Errorf("stored once read: %q, before %q", after, before)
		}
		w.last().exit(3, "Error: something broke\n")
		h2.settle()
		if want := []string{"acc-image-upload-test.gif", "acc-image-upload-test.png", "acc-image-upload-test.jpg", "pasted-image-1.png"}; !slices.Equal(rcNames(h2.m), want) {
			t.Errorf("given back: %q, want %q", rcNames(h2.m), want)
		}
	})
}

// TestRestoreShareOffline: a stored draft's file on a share that stays
// offline, its read failing at last (Windows says a server it cannot find
// is a path that does not exist), stays in the draft, shown "not
// reachable", the send refused with the reason; once the share answers
// the send carries it; a file gone from a folder that answers is still
// left out (plan 33, the review of the owner's offline-share question).
func TestRestoreShareOffline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		files := rcFiles(t)
		share := files["jpg"] // stands for \\server\share\...\x.jpg
		pngA := redPNG(t)
		d := input.Drafts{Dir: t.TempDir()}
		cwd := t.TempDir()
		h := rcRig(t, &d, cwd, &clip{image: pngA}, &wireClaude{}, []string{files["png"], share})
		h.send(tea.PasteMsg{Content: "with the share's image"})
		upload(t, h)
		h.send(altV())
		h.crash()

		offline := true
		inspectPreview = func(path string, lim input.ImageLimits) (input.Image, image.Image, error) {
			if path == share && offline {
				return input.Image{}, nil, errors.New("the file is gone")
			}
			return input.InspectPreview(path, lim)
		}
		loadImage = func(path string, lim input.ImageLimits) (input.Image, []byte, error) {
			if path == share && offline {
				return input.Image{}, nil, errors.New("the file is gone")
			}
			return input.LoadImage(path, lim)
		}
		unreachable = func(path string) bool { return path == share && offline }
		defer func() {
			inspectPreview, loadImage, unreachable = input.InspectPreview, input.LoadImage, input.Unreachable
		}()

		w := &wireClaude{}
		h2 := rcRig(t, &d, cwd, &clip{}, w)
		if want := []string{"acc-image-upload-test.png", "acc-image-upload-test.jpg", "pasted-image-1.png"}; !slices.Equal(rcNames(h2.m), want) {
			t.Fatalf("restored %q, want %q", rcNames(h2.m), want)
		}
		if h2.m.ui.Toast != "the draft's images: acc-image-upload-test.jpg: its folder cannot be reached; kept in the draft" {
			t.Errorf("toast %q", h2.m.ui.Toast)
		}
		if f := frameText(h2); !strings.Contains(f, "acc-image-upload-test.jpg · not reachable") {
			t.Errorf("the chip does not say it:\n%s", f)
		}
		h2.send(typed('!'))
		project := h2.m.State().Context.Project
		if got := draftList(t, d, project); len(got) != 3 || got[1] != share {
			t.Errorf("stored %q; the share's file must stay listed", got)
		}
		h2.send(key(tea.KeyEnter, 0))
		if n := len(w.written("user")); n != 0 {
			t.Errorf("%d messages written with the share offline", n)
		}
		if got := h2.m.State().Turn.Notice; got != "the message could not be sent: acc-image-upload-test.jpg: its folder cannot be reached" {
			t.Errorf("notice %q", got)
		}
		offline = false // the share answers again
		h2.send(key(tea.KeyEnter, 0))
		sent := rcSent(t, w)
		if len(sent) != 3 || !bytes.Equal(sent[1], rcRead(t, share)) || !bytes.Equal(sent[2], pngA) {
			t.Errorf("once the share answers, sent %d images", len(sent))
		}
	})
}

// TestCodexRestart is criterion 33.11's Codex part: a Codex session's
// draft of text, two files and a pasted image restored, the images sent
// byte for byte.
func TestCodexRestart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		files := rcFiles(t)
		pngA := drawnPNG(t, png.DefaultCompression)
		d := input.Drafts{Dir: t.TempDir()}
		cwd := t.TempDir()
		c := &clip{image: pngA}
		p := &picker{answers: [][]string{{files["png"], files["jpg"]}}}
		h := codexWire(t, &wireCodex{thread: "th-restart"}, Options{Drafts: &d, Cwd: cwd, PickImages: p.pick, Clipboard: c})
		h.send(tea.PasteMsg{Content: "codex draft"})
		upload(t, h)
		h.send(altV())
		h.crash()
		w2 := &wireCodex{thread: "th-restart-2", auto: true}
		h2 := codexWire(t, w2, Options{Drafts: &d, Cwd: cwd, Clipboard: c})
		if h2.m.edit.Text() != "codex draft" || !slices.Equal(rcNames(h2.m), []string{"acc-image-upload-test.png", "acc-image-upload-test.jpg", "pasted-image-1.png"}) {
			t.Fatalf("restored %q with %q", h2.m.edit.Text(), rcNames(h2.m))
		}
		h2.send(key(tea.KeyEnter, 0))
		sent := codexSent(t, w2)
		want := [][]byte{rcRead(t, files["png"]), rcRead(t, files["jpg"]), pngA}
		if len(sent) != len(want) {
			t.Fatalf("sent %d images", len(sent))
		}
		for i := range want {
			if !bytes.Equal(sent[i], want[i]) {
				t.Errorf("image %d sent is not the original bytes", i)
			}
		}
	})
}
