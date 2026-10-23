package claude

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/input/imagetest"
	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// testPNG is the dashboard's test image as PNG.
func testPNG(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, imagetest.Draw()); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// TestClaudeImageMessage is plan 31's criterion 31.4 for Claude Code: the
// owner's message with an image, sent through the real process plumbing,
// reaches the CLI's stdin as a stream-json user message whose content is
// an image block, base64 of the image's exact bytes with its media type,
// then the text; the event mapped from the line carries the image's media
// type and size and none of its bytes; the state's entry names the file.
func TestClaudeImageMessage(t *testing.T) {
	fixture, _ := filepath.Abs(filepath.Join("..", "..", "..", "testdata", "probes", "p2-claude-conversation.jsonl"))
	log := filepath.Join(t.TempDir(), "stdin.jsonl")
	t.Setenv("ACC_FAKE_FILE", fixture)
	t.Setenv("ACC_FAKE_LOG", log)
	r, _ := start(t, "replay", nil)
	r.wait(event.SessionReady)

	data := testPNG(t)
	msg := provider.Message{Text: "What colours and what text are in this image? Answer in one line.",
		Images: []provider.Image{{Name: "acc-image-upload-test.png", Media: "image/png", Data: data}}}
	r.st.Do(state.Send{Text: msg.Text, Images: []state.Attachment{{Path: `C:\test\acc-image-upload-test.png`, Name: "acc-image-upload-test.png", Size: int64(len(data)), Media: "image/png"}}})
	if err := r.sess.Send(msg); err != nil {
		t.Fatal(err)
	}
	ev := r.wait(event.UserMessage)
	u := ev.Payload.(event.User)
	if u.Text != msg.Text || len(u.Images) != 1 || u.Images[0].Media != "image/png" || u.Images[0].Size != len(data) {
		t.Errorf("the mapped message %+v, want the text and one image/png of %d bytes", u, len(data))
	}
	if strings.Contains(ev.Raw, base64.StdEncoding.EncodeToString(data)[:64]) || !strings.Contains(ev.Raw, "bytes not kept>") {
		t.Errorf("the event keeps the image's bytes: %.200s", ev.Raw)
	}
	var entry state.Entry
	for _, e := range r.st.Conversation {
		if e.Role == "user" {
			entry = e
		}
	}
	if len(entry.Images) != 1 || entry.Images[0].Name != "acc-image-upload-test.png" {
		t.Errorf("the conversation's entry images %+v", entry.Images)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	r.st.Do(state.Quit{})
	if err := r.sess.Close(ctx); err != nil {
		t.Fatal(err)
	}
	r.end()

	// What the CLI read on its stdin.
	b, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	var line string
	for _, l := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		if strings.Contains(l, `"type":"image"`) {
			line = l
		}
	}
	var m struct {
		Type    string `json:"type"`
		Message struct {
			Role    string `json:"role"`
			Content []struct {
				Type   string `json:"type"`
				Text   string `json:"text"`
				Source struct {
					Type      string `json:"type"`
					MediaType string `json:"media_type"`
					Data      string `json:"data"`
				} `json:"source"`
			} `json:"content"`
		} `json:"message"`
	}
	if err := json.Unmarshal([]byte(line), &m); err != nil {
		t.Fatalf("no user line with an image block: %v\n%s", err, b)
	}
	c := m.Message.Content
	if m.Type != "user" || m.Message.Role != "user" || len(c) != 2 || c[0].Type != "image" || c[1].Type != "text" || c[1].Text != msg.Text {
		t.Fatalf("the line written: %.300s", line)
	}
	got, err := base64.StdEncoding.DecodeString(c[0].Source.Data)
	if err != nil || c[0].Source.Type != "base64" || c[0].Source.MediaType != "image/png" || !bytes.Equal(got, data) {
		t.Errorf("the image block: %s %s, %d bytes (%v), want the PNG's %d", c[0].Source.Type, c[0].Source.MediaType, len(got), err, len(data))
	}

	// A message without images keeps its string content.
	if b, _ := json.Marshal(userContent(provider.Message{Text: "plain"})); !strings.Contains(string(b), `"content":"plain"`) {
		t.Errorf("a plain message: %s", b)
	}
}
