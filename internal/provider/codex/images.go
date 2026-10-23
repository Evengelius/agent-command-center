package codex

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// dataURLImage is an `image` input's media type and size when its URL is a
// base64 data URL (plan 31); a web URL gives neither.
func dataURLImage(url string) event.Image {
	media, data, ok := splitDataURL(url)
	if !ok {
		return event.Image{}
	}
	return event.Image{Media: media, Size: base64Size(data)}
}

// splitDataURL splits "data:<media>;base64,<data>".
func splitDataURL(url string) (media, data string, ok bool) {
	rest, found := strings.CutPrefix(url, "data:")
	if !found {
		return "", "", false
	}
	head, data, found := strings.Cut(rest, ",")
	if !found || !strings.HasSuffix(head, ";base64") {
		return "", "", false
	}
	return strings.TrimSuffix(head, ";base64"), data, true
}

// base64Size is the bytes a standard base64 string holds.
func base64Size(s string) int {
	return base64.StdEncoding.DecodedLen(len(s)) - strings.Count(s[max(len(s)-2, 0):], "=")
}

// withoutImageURLs is the line with each base64 data URL of an image (an
// `image` input's url, or any other string that is one) replaced by its
// media type and size, so the events, the log and the screen never hold
// the image (plan 31). A line that cannot be re-read loses its body
// instead.
func withoutImageURLs(raw []byte) []byte {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var v any
	if d.Decode(&v) != nil {
		return []byte(`{"redacted":"a line with images not kept"}`)
	}
	var walk func(any) any
	walk = func(v any) any {
		switch x := v.(type) {
		case map[string]any:
			for k, y := range x {
				x[k] = walk(y)
			}
		case []any:
			for i, y := range x {
				x[i] = walk(y)
			}
		case string:
			if media, data, ok := splitDataURL(x); ok && strings.HasPrefix(media, "image/") {
				return fmt.Sprintf("data:%s;base64,<%d bytes not kept>", media, base64Size(data))
			}
		}
		return v
	}
	v = walk(v)
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if enc.Encode(v) != nil {
		return []byte(`{"redacted":"a line with images not kept"}`)
	}
	return bytes.TrimRight(b.Bytes(), "\n")
}

// keepNoImages applies withoutImageURLs to the raw line each event keeps,
// once the line is mapped, so the conversation is unchanged while the
// event log, the inspection view and the unrecognised lines hold no image:
// the app-server writes the owner's images back in its userMessage items
// (plan 31, seen live in 31.8).
func keepNoImages(evs []event.Event) []event.Event {
	for i := range evs {
		if strings.Contains(evs[i].Raw, ";base64,") {
			evs[i].Raw = string(withoutImageURLs([]byte(evs[i].Raw)))
		}
	}
	return evs
}
