package claude

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// contentImages are a message content's image blocks: their media types and
// their sizes as sent (plan 31). A string content has none.
func contentImages(c json.RawMessage) []event.Image {
	var blocks []struct {
		Type   string `json:"type"`
		Source struct {
			Type      string `json:"type"`
			MediaType string `json:"media_type"`
			Data      string `json:"data"`
		} `json:"source"`
	}
	if json.Unmarshal(c, &blocks) != nil {
		return nil
	}
	var out []event.Image
	for _, b := range blocks {
		if b.Type == "image" {
			out = append(out, event.Image{Media: b.Source.MediaType, Size: base64Size(b.Source.Data)})
		}
	}
	return out
}

// base64Size is the bytes a standard base64 string holds.
func base64Size(s string) int {
	return base64.StdEncoding.DecodedLen(len(s)) - strings.Count(s[max(len(s)-2, 0):], "=")
}

// withoutImageData is the line with each base64 image source's data
// replaced by its size, so the event, the log and the screen never hold the
// image (plan 31). A line that cannot be re-read loses its body instead.
func withoutImageData(raw []byte) []byte {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var v any
	if d.Decode(&v) != nil {
		return []byte(`{"redacted":"a user message with images not kept"}`)
	}
	var walk func(any)
	walk = func(v any) {
		switch v := v.(type) {
		case map[string]any:
			if v["type"] == "base64" {
				if s, ok := v["data"].(string); ok {
					v["data"] = fmt.Sprintf("<%d bytes not kept>", base64Size(s))
				}
			}
			for _, x := range v {
				walk(x)
			}
		case []any:
			for _, x := range v {
				walk(x)
			}
		}
	}
	walk(v)
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if enc.Encode(v) != nil {
		return []byte(`{"redacted":"a user message with images not kept"}`)
	}
	return bytes.TrimRight(b.Bytes(), "\n")
}
