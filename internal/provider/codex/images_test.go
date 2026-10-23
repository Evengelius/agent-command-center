package codex

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image/png"
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/input/imagetest"
	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// TestCodexImageTurn is plan 31's criterion 31.4 for Codex: the owner's
// message with an image starts a turn whose input is an `image` input
// holding a data URL of the image's exact bytes, then the text, as the
// app-server's UserInput takes them; the event mapped from the request
// carries the image's media type and size and none of its bytes.
func TestCodexImageTurn(t *testing.T) {
	r, _ := start(t, "replay", nil, "", probe("p6b-codex-app-server.jsonl"))
	r.wait(event.SessionStarted)
	var b bytes.Buffer
	if err := png.Encode(&b, imagetest.Draw()); err != nil {
		t.Fatal(err)
	}
	data := b.Bytes()
	msg := provider.Message{Text: "What colours and what text are in this image? Answer in one line.",
		Images: []provider.Image{{Name: "acc-image-upload-test.png", Media: "image/png", Data: data}}}
	r.st.Do(state.Send{Text: msg.Text, Images: []state.Attachment{{Path: `C:\test\acc-image-upload-test.png`, Name: "acc-image-upload-test.png", Size: int64(len(data)), Media: "image/png"}}})
	if err := r.sess.Send(msg); err != nil {
		t.Fatal(err)
	}
	ev := r.wait(event.UserMessage)
	u := ev.Payload.(event.User)
	if u.Text != msg.Text || len(u.Images) != 1 || u.Images[0].Media != "image/png" || u.Images[0].Size != len(data) {
		t.Errorf("the mapped message %+v", u)
	}
	if strings.Contains(ev.Raw, base64.StdEncoding.EncodeToString(data)[:64]) || !strings.Contains(ev.Raw, "bytes not kept>") {
		t.Errorf("the event keeps the image's bytes: %.200s", ev.Raw)
	}
	// The fake CLI logs the line once it has read it: the turn it answers
	// with says so.
	r.wait(event.TurnStarted)
	var line string
	for _, l := range r.written() {
		if strings.Contains(l, `"method":"turn/start"`) {
			line = l
		}
	}
	var req struct {
		Params struct {
			Input []struct {
				Type string `json:"type"`
				Text string `json:"text"`
				URL  string `json:"url"`
			} `json:"input"`
		} `json:"params"`
	}
	if err := json.Unmarshal([]byte(line), &req); err != nil {
		t.Fatalf("no turn/start written: %v", err)
	}
	in := req.Params.Input
	if len(in) != 2 || in[0].Type != "image" || in[1].Type != "text" || in[1].Text != msg.Text {
		t.Fatalf("the turn's input: %.300s", line)
	}
	enc, ok := strings.CutPrefix(in[0].URL, "data:image/png;base64,")
	got, err := base64.StdEncoding.DecodeString(enc)
	if !ok || err != nil || !bytes.Equal(got, data) {
		t.Errorf("the image input's URL holds %d bytes (%v), want the PNG's %d", len(got), err, len(data))
	}
}

// TestCodexImageEcho: the app-server writes the owner's images back in its
// userMessage items (plan 31, seen live in 31.8); the events mapped from
// those lines, and from any line holding an image's data URL in another
// field, keep its media type and size and none of its bytes, while a data
// URL inside the model's text reaches the conversation as written.
func TestCodexImageEcho(t *testing.T) {
	var b bytes.Buffer
	if err := png.Encode(&b, imagetest.Draw()); err != nil {
		t.Fatal(err)
	}
	enc := base64.StdEncoding.EncodeToString(b.Bytes())
	url := "data:image/png;base64," + enc
	kept := fmt.Sprintf("data:image/png;base64,<%d bytes not kept>", b.Len())
	content := `"content":[{"type":"image","url":"` + url + `"},{"type":"text","text":"What colours?","text_elements":[]}]`
	lines := []string{
		`{"method":"item/started","params":{"threadId":"th-1","turnId":"tu-1","item":{"type":"userMessage","id":"u-1","clientId":null,` + content + `}}}`,
		`{"method":"item/completed","params":{"threadId":"th-1","turnId":"tu-1","item":{"type":"userMessage","id":"u-1","clientId":null,` + content + `}}}`,
		`{"method":"codex/event/user_message","params":{"msg":{"type":"user_message","message":"What colours?","images":["` + url + `"]}}}`,
	}
	m := NewMapper()
	for _, l := range lines {
		evs := m.Out([]byte(l))
		if len(evs) == 0 {
			t.Fatalf("no event for %.60s", l)
		}
		for _, ev := range evs {
			if strings.Contains(ev.Raw, enc[:64]) || !strings.Contains(ev.Raw, kept) {
				t.Errorf("%s keeps the image's bytes or loses its size: %.200s", ev.Kind, ev.Raw)
			}
		}
	}
	// The model's text is the conversation: a data URL in it stays whole.
	text := `<img src="` + url + `">`
	q, _ := json.Marshal(text)
	delta := `{"method":"item/agentMessage/delta","params":{"threadId":"th-1","turnId":"tu-1","itemId":"a-1","delta":` + string(q) + `}}`
	found := false
	for _, ev := range m.Out([]byte(delta)) {
		if d, ok := ev.Payload.(event.Delta); ok {
			found = true
			if d.Text != text {
				t.Errorf("the model's text changed: %.120s", d.Text)
			}
		}
	}
	if !found {
		t.Errorf("no delta mapped from %.80s", delta)
	}
}
