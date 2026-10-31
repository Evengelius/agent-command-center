package render

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/fixture"
	"github.com/Evengelius/agent-command-center/internal/provider/codex"
)

// Plan 41 (brief amendment 42): a review row rated low risk shows no
// authorization; from medium up it does (design D1).

// p41CreateFound is P7b's second child thread (/root/create_found); the
// first, /root/list_files, is plan 40's p40Child.
const p41CreateFound = "01a14937-2f86-7543-9d91-e0a0e52125cc"

// p41Stock is the rationale Codex reports after its reviewer's short
// answer for a low-risk action (plan.md, Plan 41, What was found).
const p41Stock = "Auto-review returned a low-risk allow decision."

// p41Records is the owner's case (39.png) on recordings: P7b's run with
// P12x's two review notifications (item/autoApprovalReview/started, then
// completed) after each subagent command's item/started, as P12x has them,
// moved onto the command's thread, turn and item, the completion carrying
// the owner's values: approved, risk low, authorization unknown, the stock
// rationale.
func p41Records(t *testing.T) []fixture.Record {
	t.Helper()
	var started, completed json.RawMessage
	for _, r := range records(t, "p12x-codex-auto-review.jsonl") {
		switch {
		case r.Dir != "out":
		case started == nil && strings.Contains(string(r.Msg), `"method":"item/autoApprovalReview/started"`):
			started = r.Msg
		case completed == nil && strings.Contains(string(r.Msg), `"method":"item/autoApprovalReview/completed"`):
			completed = r.Msg
		}
	}
	if started == nil || completed == nil {
		t.Fatal("P12x's review lines not found")
	}
	moved := func(line json.RawMessage, thread, turn, item, command string, k int, final bool) []byte {
		var msg map[string]any
		if err := json.Unmarshal(line, &msg); err != nil {
			t.Fatal(err)
		}
		p := msg["params"].(map[string]any)
		p["threadId"], p["turnId"], p["targetItemId"] = thread, turn, item
		p["reviewId"] = fmt.Sprintf("p41-review-%d", k)
		p["action"].(map[string]any)["command"] = command
		if final {
			p["review"] = map[string]any{"status": "approved", "riskLevel": "low", "userAuthorization": "unknown", "rationale": p41Stock}
		}
		b, err := json.Marshal(msg)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	var out []fixture.Record
	k := 0
	for _, r := range records(t, "p7b-codex-subagents.jsonl") {
		out = append(out, r)
		if r.Dir != "out" || !strings.Contains(string(r.Msg), `"method":"item/started"`) || !strings.Contains(string(r.Msg), `"type":"commandExecution"`) {
			continue
		}
		var m struct {
			Params struct {
				Item struct {
					ID      string `json:"id"`
					Command string `json:"command"`
				} `json:"item"`
				ThreadID string `json:"threadId"`
				TurnID   string `json:"turnId"`
			} `json:"params"`
		}
		if err := json.Unmarshal(r.Msg, &m); err != nil {
			t.Fatal(err)
		}
		if m.Params.ThreadID != p40Child && m.Params.ThreadID != p41CreateFound {
			continue
		}
		it := m.Params.Item
		out = append(out,
			fixture.Record{TMs: r.TMs, Dir: "out", Msg: moved(started, m.Params.ThreadID, m.Params.TurnID, it.ID, it.Command, k, false)},
			fixture.Record{TMs: r.TMs, Dir: "out", Msg: moved(completed, m.Params.ThreadID, m.Params.TurnID, it.ID, it.Command, k, true)})
		k++
	}
	if k != 4 {
		t.Fatalf("P7b's subagent commands: %d, want 4", k)
	}
	return out
}

// TestReviewRowsOwnerCase is criterion 41.2: each of the four subagent
// reviews reads the verdict, the risk and the stock rationale, with no
// authorization, in the state, in zone C's timeline (alone and in the
// frame at 120x40 and 200x60) and in the inspection.
func TestReviewRowsOwnerCase(t *testing.T) {
	s, now := replayed(codex.ProviderID, codex.NewMapper(), p41Records(t))
	want := "auto-review approved · risk low: " + p41Stock
	for _, c := range []struct {
		key     string
		reviews int
	}{{p40Child, 1}, {p41CreateFound, 3}} {
		a := s.Agents[c.key]
		if a == nil {
			t.Fatalf("no agent %s", c.key)
		}
		n := 0
		for _, e := range a.Messages {
			if e.Role == "review" {
				n++
				if e.Text != want {
					t.Errorf("%s: review entry %q, want %q", a.Label, e.Text, want)
				}
			}
		}
		if n != c.reviews {
			t.Errorf("%s: %d review entries, want %d", a.Label, n, c.reviews)
		}
		// The timeline alone, wide enough for one row an entry, each row
		// without the padding to the panel's width.
		_, rows := timeline(s, UI{Selected: c.key}, 160, 100, &Screen{})
		var trimmed []string
		for _, r := range strings.Split(texts(rows), "\n") {
			trimmed = append(trimmed, strings.TrimRight(r, " "))
		}
		tl := strings.Join(trimmed, "\n")
		if got := strings.Count(tl, "  review    │ "+want+"\n"); got != c.reviews || strings.Contains(tl, "authorization") {
			t.Errorf("%s: the timeline has %d review rows reading %q, want %d, and no authorization:\n%s", a.Label, got, want, c.reviews, tl)
		}
		for _, size := range [][2]int{{120, 40}, {200, 60}} {
			inner, _, _, lw := zoneCInner(t, s, UI{Width: size[0], Height: size[1], Now: now, Focus: "C", Selected: c.key})
			var right []string
			for _, r := range inner[3:] {
				right = append(right, strings.TrimRight(string([]rune(r)[lw+3:]), " "))
			}
			view := strings.Join(right, "\n")
			if strings.Contains(strings.Join(inner, "\n"), "authorization") || !strings.Contains(view, "review    │ auto-review approved · risk low: Auto-review") {
				t.Errorf("%s at %dx%d: zone C\n%s", a.Label, size[0], size[1], strings.Join(inner, "\n"))
			}
		}
		ins := texts(inspectMessages(s, nil, a, 200))
		if got := strings.Count(ins, want); got != c.reviews || strings.Contains(ins, "authorization") {
			t.Errorf("%s: the inspection has %d rows reading %q, want %d, and no authorization:\n%s", a.Label, got, want, c.reviews, ins)
		}
	}
}

// TestReviewRowsZoneB is criterion 41.3: P12x's two reviews, the
// orchestrator's, read their verdict, risk and rationale with no
// authorization, and zone B draws them so.
func TestReviewRowsZoneB(t *testing.T) {
	s, now := replayed(codex.ProviderID, codex.NewMapper(), records(t, "p12x-codex-auto-review.jsonl"))
	want := []string{
		"auto-review approved · risk low: Creates the explicitly requested hello.txt with the word hello in the current directory and reads it back; this is a routine, reversible local file operation.",
		"auto-review approved · risk low: Creates the specifically requested single local file in the parent directory and verifies it; the narrowly scoped operation has low risk.",
	}
	var got []string
	for _, e := range s.Conversation {
		if e.Role == "review" {
			got = append(got, e.Text)
		}
	}
	if !slices.Equal(got, want) {
		t.Fatalf("P12x's review entries %q, want %q", got, want)
	}
	for _, size := range [][2]int{{120, 40}, {200, 60}} {
		f := texts(Draw(s, UI{Width: size[0], Height: size[1], Now: now, Focus: "B"}).lines)
		if strings.Contains(f, "authorization") || !strings.Contains(f, "◈ auto-review approved · risk low: Creates the") {
			t.Errorf("%dx%d: zone B\n%s", size[0], size[1], f)
		}
	}
}

// TestReviewRowsMediumAndUp is criterion 41.4: reviews at medium, high and
// critical, the orchestrator's and a subagent's, keep their authorization
// in zone B and in zone C's timeline.
func TestReviewRowsMediumAndUp(t *testing.T) {
	b := newConvBench("codex", "gpt-6-sol")
	b.prompt("check the project")
	b.agent("a1", false, false)
	cases := []struct{ status, risk, auth, verdict string }{
		{"approved", "medium", "high", "approved"},
		{"approved", "high", "medium", "approved"},
		{"denied", "critical", "low", "denied"},
	}
	for i, c := range cases {
		for _, agent := range []string{"", "a1"} {
			b.seq++
			b.s.Apply(event.Event{Seq: b.seq, At: lookAt.Add(time.Duration(b.seq) * time.Second), Kind: event.ReviewUpdated, Origin: event.Observed, Agent: agent,
				Payload: event.Review{ReviewID: fmt.Sprintf("r%d%s", i, agent), Command: "go vet ./...", Status: c.status, Risk: c.risk, Authorization: c.auth, Rationale: "a check"}})
		}
	}
	f := texts(Draw(b.s, UI{Width: 200, Height: 60, Now: lookAt.Add(time.Minute), Focus: "B"}).lines)
	_, rows := timeline(b.s, UI{Selected: "a1"}, 120, 40, &Screen{})
	tl := texts(rows)
	for _, c := range cases {
		w := "auto-review " + c.verdict + " · risk " + c.risk + " · authorization " + c.auth + ": a check"
		if !strings.Contains(f, "◈ "+w) {
			t.Errorf("zone B has no %q:\n%s", "◈ "+w, f)
		}
		if !strings.Contains(tl, "review    │ "+w) {
			t.Errorf("zone C's timeline has no %q:\n%s", w, tl)
		}
	}
}
