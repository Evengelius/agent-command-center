package codex

import (
	"bytes"
	"encoding/json"
)

// creditsKey holds account-linked rate-limit reset credit ids: never shown
// or logged (reliability.md, Telemetry sources: Plan and credits).
const creditsKey = "rateLimitResetCredits"

// redactCredits returns the line with every creditsKey value replaced, so the
// ids never reach an event's Raw, the log or the screen. A line without the
// key is returned as is; one that cannot be re-read loses its body instead.
func redactCredits(raw []byte) []byte {
	if !bytes.Contains(raw, []byte(creditsKey)) {
		return raw
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var v any
	if d.Decode(&v) != nil {
		return []byte(`{"redacted":"line with ` + creditsKey + ` not kept"}`)
	}
	out, err := json.Marshal(dropCredits(v))
	if err != nil {
		return []byte(`{"redacted":"line with ` + creditsKey + ` not kept"}`)
	}
	return out
}

func dropCredits(v any) any {
	switch v := v.(type) {
	case map[string]any:
		for k, x := range v {
			if k == creditsKey {
				v[k] = "<not kept>"
				continue
			}
			v[k] = dropCredits(x)
		}
	case []any:
		for i, x := range v {
			v[i] = dropCredits(x)
		}
	}
	return v
}
