package app

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Images attached to the draft (plan 31, brief amendment 31; ui-design.md,
// Input editor): "Upload from computer" opens the system's Open dialog;
// each file chosen is read and checked in a command; the accepted images
// become the draft's attachments, its text and cursor untouched; a chip's
// ✕ or Ctrl+Shift+X removes one; the send key sends them with the text,
// each read and checked again when the message is written. Choosing an
// image never sends anything.

type (
	// pickedMsg is the dialog's answer: the files chosen, none when it was
	// cancelled.
	pickedMsg struct {
		paths []string
		err   error
	}
	// inspectedMsg is the files read and checked: the images accepted and a
	// note for each refused. restore marks the stored draft's images;
	// thumbs are the images' thumbnails by key (plan 32, design D4).
	inspectedMsg struct {
		imgs    []input.Image
		refused []string
		restore bool
		thumbs  map[string]thumbSixel
	}
)

// limits are the session's limits on a message's images; none before the
// session starts.
func (m *Model) limits() provider.ImageLimits {
	if m.sess == nil {
		return provider.ImageLimits{}
	}
	return m.sess.Capabilities().Images
}

// inputLimits are a provider's limits as the image reader takes them.
func inputLimits(l provider.ImageLimits) input.ImageLimits {
	return input.ImageLimits{MaxCount: l.MaxCount, MaxBytes: l.MaxBytes, MaxSide: l.MaxSide, MaxPatches: l.MaxPatches, PatchSize: l.PatchSize}
}

// pickImages opens the Open dialog in a command; one at a time.
func (m *Model) pickImages() []tea.Cmd {
	switch {
	case m.opt.PickImages == nil:
		return m.toast("no file dialog in this run")
	case m.sess == nil:
		return m.toast("no session to attach images to yet")
	case m.limits().MaxCount == 0:
		return m.toast("this session takes no images")
	case m.picking:
		return nil
	}
	m.picking = true
	pick := m.opt.PickImages
	return []tea.Cmd{func() tea.Msg {
		paths, err := pick()
		return pickedMsg{paths, err}
	}}
}

// picked reads and checks the files chosen, in a command; a cancelled
// dialog changes nothing.
func (m *Model) picked(msg pickedMsg) []tea.Cmd {
	m.picking = false
	if msg.err != nil {
		return m.toast("file dialog: " + msg.err.Error())
	}
	if len(msg.paths) == 0 {
		return nil
	}
	return []tea.Cmd{inspectImages(msg.paths, inputLimits(m.limits()), false)}
}

// inspectImages reads and checks each file, outside the update loop, and
// encodes its thumbnail for the preview (plan 32, design D5).
func inspectImages(paths []string, lim input.ImageLimits, restore bool) tea.Cmd {
	return func() tea.Msg {
		out := inspectedMsg{restore: restore, thumbs: map[string]thumbSixel{}}
		for _, p := range paths {
			img, im, err := input.InspectPreview(p, lim)
			if err != nil {
				out.refused = append(out.refused, filepath.Base(p)+": "+err.Error())
				continue
			}
			out.imgs = append(out.imgs, img)
			out.thumbs[img.Path] = thumbsOf(im)
		}
		return out
	}
}

// inspected attaches the images accepted, in the order chosen, after the
// ones already attached: one already attached, or one past the message's
// count or bytes, is not; the toast says what was refused and what the
// sending changes.
func (m *Model) inspected(msg inspectedMsg) []tea.Cmd {
	lim := m.limits()
	m.setThumbs(msg.thumbs) // those not attached are dropped after this update
	notes := append([]string(nil), msg.refused...)
	for _, img := range msg.imgs {
		switch {
		case m.attachedAt(img.Path) >= 0:
			notes = append(notes, img.Name+": already attached")
		case lim.MaxCount > 0 && len(m.attach) >= lim.MaxCount:
			notes = append(notes, fmt.Sprintf("%s: over %d images a message", img.Name, lim.MaxCount))
		case lim.MaxTotal > 0 && m.attachBytes()+img.Sent > lim.MaxTotal:
			notes = append(notes, fmt.Sprintf("%s: over %s of images a message", img.Name, input.FmtBytes(lim.MaxTotal)))
		default:
			m.attach = append(m.attach, img)
			if img.Note != "" {
				notes = append(notes, img.Name+": "+img.Note)
			}
		}
	}
	cmds := m.draftChanged()
	if len(notes) > 0 {
		prefix := ""
		if msg.restore {
			prefix = "the draft's images: "
		}
		cmds = append(cmds, m.toast(prefix+strings.Join(notes, " · "))...)
	}
	return cmds
}

// attachedAt is the index of the attachment of the file, -1 when none;
// Windows paths compare without case.
func (m *Model) attachedAt(path string) int {
	p := filepath.Clean(path)
	for i, a := range m.attach {
		q := filepath.Clean(a.Path)
		if q == p || runtime.GOOS == "windows" && strings.EqualFold(q, p) {
			return i
		}
	}
	return -1
}

// attachBytes is the attachments' bytes as sent.
func (m *Model) attachBytes() int64 {
	var n int64
	for _, a := range m.attach {
		n += a.Sent
	}
	return n
}

// removeAttachment removes the i-th attachment, the draft's text untouched.
func (m *Model) removeAttachment(i int) []tea.Cmd {
	if i < 0 || i >= len(m.attach) {
		return nil
	}
	name := m.attach[i].Name
	m.attach = append(m.attach[:i:i], m.attach[i+1:]...)
	return append(m.draftChanged(), m.toast(name+" removed")...)
}

// chips are the attachments as the editor's row shows them.
func (m *Model) chips() []render.Attachment {
	if len(m.attach) == 0 {
		return nil
	}
	out := make([]render.Attachment, len(m.attach))
	for i, a := range m.attach {
		format := a.Format
		if format == "" {
			format = strings.ToUpper(strings.TrimPrefix(a.Media, "image/"))
		}
		out[i] = render.Attachment{Name: a.Name, Size: input.FmtBytes(a.Sent), Pixels: m.thumbs[a.Path] != thumbSixel{}, Format: format}
	}
	return out
}

// sentImages are the attachments as the state keeps them with the message.
func (m *Model) sentImages() []state.Attachment {
	if len(m.attach) == 0 {
		return nil
	}
	out := make([]state.Attachment, len(m.attach))
	for i, a := range m.attach {
		out[i] = state.Attachment{Path: a.Path, Name: a.Name, Size: a.Sent, Media: a.Media, Note: a.Note}
	}
	return out
}

// restoreImages gives a message's images back to the editor before the
// ones attached since, without doubles.
func (m *Model) restoreImages(as []state.Attachment) {
	if len(as) == 0 {
		return
	}
	since := m.attach
	m.attach = nil
	for _, a := range as {
		m.attach = append(m.attach, input.Image{Path: a.Path, Name: a.Name, Sent: a.Size, Media: a.Media, Note: a.Note})
	}
	for _, a := range since {
		if m.attachedAt(a.Path) < 0 {
			m.attach = append(m.attach, a)
		}
	}
}

// imagePaths are the attachments' paths, as the stored draft lists them.
func imagePaths(imgs []input.Image) []string {
	var out []string
	for _, a := range imgs {
		out = append(out, a.Path)
	}
	return out
}

// message reads the message's images again as they are now and builds the
// message: a file gone, changed past a limit, or the images past the
// message's bytes fail the send, the draft kept with its images. A pasted
// image comes from its bytes in memory, pasted by key (plan 32, design D3).
func message(text string, imgs []state.Attachment, lim provider.ImageLimits, pasted map[string][]byte) (provider.Message, error) {
	msg := provider.Message{Text: text}
	var total int64
	for _, a := range imgs {
		var img input.Image
		var data []byte
		var err error
		switch b, ok := pasted[a.Path]; {
		case ok:
			img, data, _, err = input.InspectData(a.Name, b, inputLimits(lim))
		case isPasted(a.Path):
			err = errors.New("the pasted image was not kept")
		default:
			img, data, err = input.LoadImage(a.Path, inputLimits(lim))
		}
		if err != nil {
			return provider.Message{}, fmt.Errorf("%s: %v", a.Name, err)
		}
		total += int64(len(data))
		if lim.MaxTotal > 0 && total > lim.MaxTotal {
			return provider.Message{}, fmt.Errorf("the images are over %s a message", input.FmtBytes(lim.MaxTotal))
		}
		msg.Images = append(msg.Images, provider.Image{Name: img.Name, Media: img.Media, Data: data})
	}
	return msg, nil
}
