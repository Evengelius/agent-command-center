package app

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Images pasted (plan 32, brief amendment 32; doc/research/image-paste.md;
// ui-design.md, Input editor): Alt+V attaches the clipboard's images; Ctrl+V
// pastes text as before and, when Windows Terminal keeps its press because
// the clipboard holds no text, its release alone attaches the clipboard's
// image; a paste that is exactly the paths of the files the clipboard holds
// attaches them when every one is an image. A pasted image's bytes stay in
// memory, and in the stored draft's folder until its message is accepted or
// its chip removed (plan 33, design D8); nothing is ever sent by a paste.

type (
	// clipMsg is what the clipboard held when a paste asked for images:
	// copied files, else an image's PNG bytes; ctrlV marks Ctrl+V's ask,
	// whose empty answer says there was nothing at all to paste.
	clipMsg struct {
		files []string
		data  []byte
		err   error
		ctrlV bool
	}
	// pastedMsg is a pasted image read and checked, before it is named.
	pastedMsg struct {
		img   input.Image
		data  []byte
		sum   [32]byte
		thumb thumbSixel
		err   error
	}
	// pasteWaitMsg ends the wait after Ctrl+V's lone release.
	pasteWaitMsg struct{ seq int }
	// pathsMsg is a paste that looked like files' paths, with the files the
	// clipboard holds to compare it with.
	pathsMsg struct {
		text  string
		files []string
	}
)

// pasteWindow is how long Ctrl+V's lone release waits for the terminal's
// paste before it reads the clipboard's image: the terminal's text came 3
// ms after the release in probe P32a.
const pasteWindow = 200 * time.Millisecond

// pastedPrefix marks a pasted image's key where a file has its path: no
// file is named so.
const pastedPrefix = "pasted:"

func isPasted(path string) bool { return strings.HasPrefix(path, pastedPrefix) }

// clipImages is the clipboard as images and files; nil when the run's
// clipboard gives text only.
func (m *Model) clipImages() input.ClipboardImages {
	ci, _ := m.opt.Clipboard.(input.ClipboardImages)
	return ci
}

// editorTakesPaste says a paste goes to the draft: the dashboard, B
// focused or the Explorer's panel docked beside it (draftBeside), no window
// over it, nor the branches popup, whose search takes it (app.go, paste).
func (m *Model) editorTakesPaste() bool {
	top := render.Layers(m.s, m.ui).Top // the draft on top, or the dashboard itself (plan 49, design D7)
	return m.s.Screen == state.ScreenDashboard && m.draftBeside() && m.wt == nil &&
		(top == render.LayerDraft || top == render.LayerNone || top.Popup() && top != render.LayerBranches)
}

// pasteImages reads the clipboard's images in a command (design D1): its
// copied files first, else its image.
func (m *Model) pasteImages(ctrlV bool) []tea.Cmd {
	ci := m.clipImages()
	switch {
	case ci == nil:
		return m.toast("no clipboard in this run")
	case m.sess == nil:
		return m.toast("no session to attach images to yet")
	case m.limits().MaxCount == 0:
		return m.toast("this session takes no images")
	}
	return []tea.Cmd{func() tea.Msg {
		if files, err := ci.Files(); err == nil && len(files) > 0 {
			return clipMsg{files: files, ctrlV: ctrlV}
		}
		data, err := ci.Image()
		return clipMsg{data: data, err: err, ctrlV: ctrlV}
	}}
}

// clipped attaches what the clipboard held: its files through plan 31's
// checks, or its image, read and checked in a command.
func (m *Model) clipped(msg clipMsg) []tea.Cmd {
	switch {
	case msg.err != nil:
		return m.toast("clipboard: " + msg.err.Error())
	case len(msg.files) > 0:
		return []tea.Cmd{inspectImages(msg.files, inputLimits(m.limits()), false)}
	case len(msg.data) == 0 && msg.ctrlV:
		return m.toast("nothing to paste: no text and no image on the clipboard")
	case len(msg.data) == 0:
		return m.toast("no image on the clipboard")
	}
	data, lim := msg.data, inputLimits(m.limits())
	return []tea.Cmd{func() tea.Msg {
		img, _, im, err := input.InspectData("pasted image", data, lim)
		out := pastedMsg{img: img, data: data, sum: sha256.Sum256(data), err: err}
		if err == nil {
			out.thumb = thumbsOf(im)
		}
		return out
	}}
}

// pastedImage attaches a pasted image as "pasted-image-N.png", N the run's
// count; the same bytes as an image attached are not attached twice.
func (m *Model) pastedImage(msg pastedMsg) []tea.Cmd {
	if msg.err != nil {
		return m.toast("the pasted image: " + msg.err.Error())
	}
	for _, a := range m.attach {
		if sum, ok := m.pastedSum[a.Path]; ok && sum == msg.sum {
			return m.toast(a.Name + ": already attached")
		}
	}
	for _, e := range m.unread {
		// The stored draft's pasted images still being read back (plan 33).
		if name, sum, ok := input.ParsePasted(e); ok && sum == hex.EncodeToString(msg.sum[:]) {
			return m.toast(name + ": already attached")
		}
	}
	m.pasteN++
	img := msg.img
	img.Name = fmt.Sprintf("pasted-image-%d.png", m.pasteN)
	img.Path = pastedPrefix + img.Name
	if m.pasted == nil {
		m.pasted, m.pastedSum = map[string][]byte{}, map[string][32]byte{}
	}
	m.pasted[img.Path], m.pastedSum[img.Path] = msg.data, msg.sum
	return m.inspected(inspectedMsg{imgs: []input.Image{img}, thumbs: map[string]thumbSixel{img.Path: msg.thumb}})
}

// pastedBack takes the stored draft's pasted images back (plan 33, design
// D8): each under its name with its bytes and SHA-256, unless this run
// pasted other bytes under that name meanwhile, when it takes the next
// name; the run's count goes past the names restored.
func (m *Model) pastedBack(msg inspectedMsg) inspectedMsg {
	if m.pasted == nil {
		m.pasted, m.pastedSum = map[string][]byte{}, map[string][32]byte{}
	}
	for i, img := range msg.imgs {
		b, ok := msg.pasted[img.Path]
		if !ok {
			continue
		}
		if n, ok := pastedNumber(img.Name); ok && n > m.pasteN {
			m.pasteN = n
		}
		if sum, taken := m.pastedSum[img.Path]; taken && sum != img.Sum {
			m.pasteN++
			old := img.Path
			img.Name = fmt.Sprintf("pasted-image-%d.png", m.pasteN)
			img.Path = pastedPrefix + img.Name
			msg.thumbs[img.Path] = msg.thumbs[old]
			delete(msg.thumbs, old)
			msg.imgs[i] = img
		}
		m.pasted[img.Path], m.pastedSum[img.Path] = b, img.Sum
	}
	return msg
}

// pastedNumber is N of "pasted-image-N.png".
func pastedNumber(name string) (int, bool) {
	s, ok := strings.CutPrefix(name, "pasted-image-")
	if !ok {
		return 0, false
	}
	if s, ok = strings.CutSuffix(s, ".png"); !ok {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	return n, err == nil
}

// terminalPaste is the terminal's bracketed paste: text for the draft as
// before, unless it looks like copied files' paths and the clipboard's
// files are to be compared with it (design D2).
func (m *Model) terminalPaste(text string) []tea.Cmd {
	m.pasteCount++
	m.lastPaste = m.ui.Now
	if top := render.Layers(m.s, m.ui).Top; m.msgFocused() && top != render.LayerWindow && top != render.LayerKeys {
		m.paste(text) // the commit message takes text alone (plan 45, design D2)
		return nil
	}
	ci := m.clipImages()
	if ci == nil || !m.editorTakesPaste() || m.sess == nil || m.limits().MaxCount == 0 || !looksLikePaths(text) {
		m.paste(text)
		return nil
	}
	return []tea.Cmd{func() tea.Msg {
		files, err := ci.Files()
		if err != nil {
			files = nil
		}
		return pathsMsg{text: text, files: files}
	}}
}

// pastedPaths attaches the files when the paste is exactly their paths and
// every one is an image; otherwise the text goes into the draft.
func (m *Model) pastedPaths(msg pathsMsg) []tea.Cmd {
	if !samePaths(msg.text, msg.files) {
		m.paste(msg.text)
		return nil
	}
	text, files, lim := msg.text, msg.files, inputLimits(m.limits())
	return []tea.Cmd{func() tea.Msg {
		out := inspectedMsg{thumbs: map[string]thumbSixel{}}
		for _, f := range files {
			img, im, err := input.InspectPreview(f, lim)
			if err != nil {
				return pasteMsg{text} // not every file is an image: the paths go in as text
			}
			out.imgs = append(out.imgs, img)
			out.thumbs[img.Path] = thumbsOf(im)
		}
		return out
	}}
}

// looksLikePaths says the paste may be copied files' paths as Windows
// Terminal pastes them (probe P32a): a drive's path or a share's.
func looksLikePaths(text string) bool {
	t := strings.TrimLeft(strings.TrimSpace(text), `"`)
	return len(t) > 2 && t[1] == ':' && (t[2] == '\\' || t[2] == '/') || strings.HasPrefix(t, `\\`)
}

// samePaths says the paste is the files' paths and nothing else: each as
// it is or quoted, in any order, blanks between them.
func samePaths(text string, files []string) bool {
	if len(files) == 0 {
		return false
	}
	rest := text
	for _, f := range files {
		switch q := `"` + f + `"`; {
		case strings.Contains(rest, q):
			rest = strings.Replace(rest, q, " ", 1)
		case strings.Contains(rest, f):
			rest = strings.Replace(rest, f, " ", 1)
		default:
			return false
		}
	}
	return strings.TrimSpace(rest) == ""
}

// joinPaths writes copied files' paths as a paste of them reads: quoted
// when they hold a blank, a space between.
func joinPaths(files []string) string {
	out := make([]string, len(files))
	for i, f := range files {
		out[i] = f
		if strings.ContainsAny(f, " \t") {
			out[i] = `"` + f + `"`
		}
	}
	return strings.Join(out, " ")
}

// keyReleased handles the release of a key: only Ctrl+V's matters.
func (m *Model) keyReleased(k tea.KeyReleaseMsg) []tea.Cmd {
	switch k.String() {
	case "ctrl+v", "ctrl+shift+v":
		if m.editorTakesPaste() {
			return m.ctrlVReleased()
		}
	}
	return nil
}

// ctrlVReleased is the V key's release with Ctrl held. Windows Terminal
// keeps Ctrl+V's press for its own paste and forwards the release even
// when it pastes nothing, the clipboard holding no text (probe P32a;
// undocumented behaviour, observed only): a release without its press
// waits for that paste, and when none comes reads the clipboard's image
// (design D2). Alt+V works without it.
func (m *Model) ctrlVReleased() []tea.Cmd {
	if m.ctrlVDown {
		m.ctrlVDown = false // the press reached the dashboard: its own paste ran
		return nil
	}
	if !m.lastPaste.IsZero() && m.ui.Now.Sub(m.lastPaste) < pasteWindow {
		return nil // the terminal's paste came just before the release
	}
	m.pasteWait++
	m.waitPastes = m.pasteCount
	return []tea.Cmd{m.opt.After(pasteWindow, pasteWaitMsg{m.pasteWait})}
}

// pasteWaited ends the wait: a paste came, the terminal had text; none
// came, the clipboard's image is read.
func (m *Model) pasteWaited(msg pasteWaitMsg) []tea.Cmd {
	if msg.seq != m.pasteWait || m.pasteCount != m.waitPastes || !m.editorTakesPaste() {
		return nil
	}
	return m.pasteImages(true)
}

// pastedFor copies the pasted bytes the message's images name, for the
// write's own goroutine: the bytes are never changed once pasted.
func (m *Model) pastedFor(imgs []state.Attachment) map[string][]byte {
	var out map[string][]byte
	for _, a := range imgs {
		if b, ok := m.pasted[a.Path]; ok {
			if out == nil {
				out = map[string][]byte{}
			}
			out[a.Path] = b
		}
	}
	return out
}

// pruneImages drops the thumbnails and the pasted bytes that no
// attachment and no message waiting for its acceptance holds any more
// (design D3).
func (m *Model) pruneImages() {
	if len(m.thumbs) == 0 && len(m.pasted) == 0 {
		return
	}
	keep := map[string]bool{}
	for _, a := range m.attach {
		keep[a.Path] = true
	}
	for _, a := range m.s.Turn.DraftImages {
		keep[a.Path] = true
	}
	for k := range m.thumbs {
		if !keep[k] {
			delete(m.thumbs, k)
		}
	}
	for k := range m.pasted {
		if !keep[k] {
			delete(m.pasted, k)
			delete(m.pastedSum, k)
		}
	}
}
