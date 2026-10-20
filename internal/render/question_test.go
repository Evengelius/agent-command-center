package render

import (
	"strings"
	"testing"
	"time"

	"github.com/Evengelius/agent-command-center/internal/state"
)

const questionInput = `{"questions":[{"question":"Which colour?","header":"Colour","options":[{"label":"Rouge café ✓","description":"warm"},{"label":"Blue","description":"cold"},{"label":"Green","description":"plain"},{"label":"Pink","description":"soft"}],"multiSelect":false},{"question":"Which days?","header":"Days","options":[{"label":"Monday","description":"first"},{"label":"Tuesday","description":"second"}],"multiSelect":true}]}`

func rowsText(ls []line) string {
	var b strings.Builder
	for _, l := range ls {
		for _, sp := range l {
			b.WriteString(sp.text)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// Plan 14, item 1: the pending question card lists every question with its
// options, marks the selected question when it has the focus and the chosen
// options; once answered it reads one line per answer.
func TestQuestionCard(t *testing.T) {
	at := time.Date(2026, 10, 18, 9, 5, 7, 0, time.Local)
	ap := &state.Approval{ID: "q1", Tool: "AskUserQuestion", Question: true, Input: []byte(questionInput), Status: state.Pending, RequestedAt: at, AnsweredAt: at}
	s := &state.State{Approvals: map[string]*state.Approval{"q1": ap}}
	e := state.Entry{Role: "approval", Text: "AskUserQuestion", MessageID: "q1"}
	qs := QuestionsOf(ap)
	pick := NewPick("q1", qs)
	pick.At = 1
	pick.Chosen[0][0] = true
	pick.Chosen[1][1] = true

	got := rowsText(approvalCard(s, UI{Card: "q1", Pick: pick}, 0, e, 150, look{}))
	for _, want := range []string{
		"? QUESTION", "  Colour · Which colour?", "▸ Days · Which days?",
		"1 ● Rouge café ✓ — warm", "2 ○ Blue — cold", "3 ○ Green — plain", "4 ○ Pink — soft",
		"1 [ ] Monday — first", "2 [x] Tuesday — second",
		"▸ 1-4 choose · ← → question · a send the answers · d deny, then answer in your next message · c cancel the turn · Enter full view · Esc later",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("no %q in\n%s", want, got)
		}
	}
	if strings.Contains(got, "▸ Colour") || strings.Contains(got, "more lines") {
		t.Errorf("wrong selection or cut:\n%s", got)
	}
	// Another approval's pick is ignored; without the focus nothing is marked.
	got = rowsText(approvalCard(s, UI{Pick: pick}, 0, e, 150, look{}))
	if strings.Contains(got, "▸") || !strings.Contains(got, "F2 to answer") || !strings.Contains(got, "1 ● Rouge") {
		t.Errorf("unfocused card:\n%s", got)
	}
	pick.ID = "other"
	got = rowsText(approvalCard(s, UI{Card: "q1", Pick: pick}, 0, e, 150, look{}))
	if strings.Contains(got, "●") || strings.Contains(got, "[x]") || !strings.Contains(got, "▸ Colour") {
		t.Errorf("another approval's pick shows:\n%s", got)
	}

	ap.Status, ap.Answer = state.Allowed, "allow"
	ap.Answers = map[string]string{"Which colour?": "Rouge café ✓", "Which days?": "Tuesday"}
	ls := approvalCard(s, UI{}, 0, e, 150, look{})
	got = rowsText(ls)
	// Answered, the card is rows of the tool block, on its ground (plan 19;
	// plan 20: the card around them draws their margin).
	if len(ls) != 3 || !strings.Contains(got, " ✓ ") || !strings.Contains(got, "answered by you "+clockSec(at)) ||
		!strings.Contains(got, "    Which colour? → Rouge café ✓ ") || !strings.Contains(got, "    Which days? → Tuesday ") {
		t.Errorf("answered card:\n%s", got)
	}

	// The viewer: the same list, the new hint, the answered label and outcome.
	ap.Status, ap.Answers = state.Pending, nil
	body := rowsText(viewerBodyFor(s, UI{Pick: NewPick("q1", qs)}, ap, 100))
	if !strings.Contains(body, "▸ Colour · Which colour?") || !strings.Contains(body, "2 [ ] Tuesday — second") {
		t.Errorf("viewer body:\n%s", body)
	}
	if !strings.Contains(viewQuestionKeys, "1-4 choose · ← → question · a send the answers · d deny · c cancel the turn · ↑↓ PgUp PgDn Home End scroll · Esc back to the card") {
		t.Errorf("viewer hint %q", viewQuestionKeys)
	}
	ap.Status, ap.Answers = state.Allowed, map[string]string{"Which colour?": "Blue", "Which days?": "Monday"}
	s.Context.Provider = "claude-code"
	f := rowsText(viewer(s, UI{Width: 100, Height: 30, View: "q1"}, ap, &Screen{}))
	if !strings.Contains(f, "✓ ANSWERED") || !strings.Contains(f, "answered by you "+clockSec(at)) || !strings.Contains(f, "Which colour? → Blue") {
		t.Errorf("answered viewer:\n%s", f)
	}
}
