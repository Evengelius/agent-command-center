package app

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/fixture"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// offering is P6b with its request listing decisions as L3's did (Codex
// 0.145.0), plus acceptForSession and decline: no recording lists them, so
// the list is synthetic.
func offering(t *testing.T, recs []fixture.Record) []fixture.Record {
	t.Helper()
	out := append([]fixture.Record(nil), recs...)
	r := &out[36]
	var msg map[string]json.RawMessage
	var params map[string]any
	if json.Unmarshal(r.Msg, &msg) != nil || json.Unmarshal(msg["params"], &params) != nil || !strings.Contains(string(msg["method"]), "requestApproval") {
		t.Fatalf("p6b line 37 is not the request: %s", r.Msg)
	}
	params["availableDecisions"] = []any{"accept", "acceptForSession",
		map[string]any{"acceptWithExecpolicyAmendment": map[string]any{"execpolicy_amendment": []string{"go", "vet"}}}, "decline", "cancel"}
	msg["params"], _ = json.Marshal(params)
	r.Msg, _ = json.Marshal(msg)
	return out
}

// TestCodexSessionAllow is plan 5, item 8, criteria 2, 4 and 5: s answers
// acceptForSession, from the card or the full view, only when the request
// lists it; the hint says so only then; the amendment is never offered.
func TestCodexSessionAllow(t *testing.T) {
	const hint = "s allow for the session"
	allowed := []string{`{"id":0,"result":{"decision":"acceptForSession"}}`}
	for _, c := range []struct {
		name    string
		offered bool
		view    bool // answer from the full view
		want    []string
	}{
		{"card", true, false, allowed},
		{"full view", true, true, allowed},
		{"not offered", false, false, nil},
		{"not offered, full view", false, true, nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				recs := probe(t, "p6b-codex-app-server.jsonl")
				if c.offered {
					recs = offering(t, recs)
				}
				text, at := firstTurn(t, recs, 1)
				w := &wireCodex{thread: p6bThread}
				h := codexWire(t, w, Options{})
				h.check = func(f string) {
					if strings.Contains(strings.ToLower(f), "execpolicy") || strings.Contains(f, "acceptWith") {
						t.Errorf("frame offers the amendment:\n%s", f)
					}
					if !c.offered {
						noSessionAnswer(t, f)
					}
				}
				h.send(tea.PasteMsg{Content: text})
				h.send(key(tea.KeyEnter, 0))
				codexPlay(t, h, w, recs, at+1, 37)
				ap := h.m.State().Approvals["0"]
				if ap == nil || ap.ForSession != c.offered || ap.Status != state.Pending {
					t.Fatalf("approval %+v; want pending, for session %v", ap, c.offered)
				}
				before := len(w.written(0))
				h.send(key(tea.KeyF2, 0))
				keys := "▸ a allow once · d deny · c cancel the turn"
				if c.offered {
					keys = "▸ a allow once · " + hint + " · d deny · c cancel the turn"
				}
				if f := frameText(h); !strings.Contains(f, keys) {
					t.Errorf("focused card lacks %q:\n%s", keys, f)
				}
				if c.view {
					h.send(key(tea.KeyEnter, 0))
					f := frameText(h)
					if c.offered != strings.Contains(f, "a allow once · "+hint+" · d deny") {
						t.Errorf("full view's hint, offered %v:\n%s", c.offered, f)
					}
				}
				h.send(typed('s'))
				h.send(typed('s'))
				h.settle()
				if got := w.written(before); !reflect.DeepEqual(got, c.want) {
					t.Errorf("written\n got %q\nwant %q", got, c.want)
				}
				ap = h.m.State().Approvals["0"]
				if !c.offered {
					if ap.Status != state.Pending {
						t.Errorf("approval %q after s, want pending", ap.Status)
					}
					for _, n := range h.m.State().Notes { // the key is not offered: never sent to be refused
						if strings.Contains(n, "answer refused") {
							t.Errorf("note %q", n)
						}
					}
					return
				}
				if ap.Status != state.Allowed || ap.Answer != "acceptForSession" {
					t.Errorf("approval %q, answer %q", ap.Status, ap.Answer)
				}
				if f := frameText(h); !strings.Contains(f, "allowed for the session by you") {
					t.Errorf("answered card:\n%s", f)
				}
			})
		})
	}
}
