package render

import (
	"encoding/json"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// A question request (Claude Code's AskUserQuestion) is answered on its card
// and in its viewer: the owner picks an option of each question and sends the
// answers (plan 14, item 1; P14a). Only what the request carries is drawn.

// Question is one question of an AskUserQuestion request.
type Question struct {
	Text, Header string
	Options      []Option
	Multi        bool
}

// Option is one choice of a question.
type Option struct{ Label, Description string }

// Pick is the owner's choices for one question approval: the selected
// question and, per question, which options are chosen. It belongs to the
// approval ID and is reset when another approval is shown.
type Pick struct {
	ID     string
	At     int
	Chosen [][]bool
}

// QuestionsOf reads the questions of approval ap; nil when it is not a
// question or its input is not readable.
func QuestionsOf(ap *state.Approval) []Question {
	if ap == nil || !ap.Question {
		return nil
	}
	var in struct {
		Questions []struct {
			Question    string `json:"question"`
			Header      string `json:"header"`
			MultiSelect bool   `json:"multiSelect"`
			Options     []struct {
				Label       string `json:"label"`
				Description string `json:"description"`
			} `json:"options"`
		} `json:"questions"`
	}
	if json.Unmarshal(ap.Input, &in) != nil {
		return nil
	}
	var out []Question
	for _, q := range in.Questions {
		qq := Question{Text: q.Question, Header: q.Header, Multi: q.MultiSelect}
		for _, o := range q.Options {
			qq.Options = append(qq.Options, Option{o.Label, o.Description})
		}
		out = append(out, qq)
	}
	return out
}

// NewPick is an empty pick of approval id over qs.
func NewPick(id string, qs []Question) Pick {
	p := Pick{ID: id, Chosen: make([][]bool, len(qs))}
	for i, q := range qs {
		p.Chosen[i] = make([]bool, len(q.Options))
	}
	return p
}

// Valid says whether p is the pick of approval id over qs.
func (p Pick) Valid(id string, qs []Question) bool {
	if p.ID != id || len(p.Chosen) != len(qs) || p.At < 0 || p.At >= max(len(qs), 1) {
		return false
	}
	for i, q := range qs {
		if len(p.Chosen[i]) != len(q.Options) {
			return false
		}
	}
	return true
}

// Answers are the chosen labels of each question, a multiple choice's joined
// by ", " in option order; ok is false until every question has a choice.
func (p Pick) Answers(qs []Question) (answers map[string]string, ok bool) {
	answers = map[string]string{}
	for i, q := range qs {
		var labels []string
		for j, o := range q.Options {
			if i < len(p.Chosen) && j < len(p.Chosen[i]) && p.Chosen[i][j] {
				labels = append(labels, o.Label)
			}
		}
		if len(labels) == 0 {
			return nil, false
		}
		answers[q.Text] = strings.Join(labels, ", ")
	}
	return answers, len(qs) > 0
}

// questionLines draws the questions with their options: the selected question
// marked when the card has the focus, the chosen options filled. Nothing is
// cut: every option shows.
func questionLines(qs []Question, p Pick, focused bool, w int) []line {
	var out []line
	for i, q := range qs {
		mark := "  "
		if focused && i == p.At {
			mark = "▸ "
		}
		title := q.Text
		if q.Header != "" {
			title = q.Header + " · " + q.Text
		}
		st := stText
		if focused && i == p.At {
			st = stAttentionBold
		}
		out = append(out, wrapDeco("  "+mark+title, w, "      ", st)...)
		for j, o := range q.Options {
			box := "○"
			switch {
			case q.Multi && p.Chosen[i][j]:
				box = "[x]"
			case q.Multi:
				box = "[ ]"
			case p.Chosen[i][j]:
				box = "●"
			}
			text := "    " + strconv.Itoa(j+1) + " " + box + " " + o.Label
			if o.Description != "" {
				text += " — " + o.Description
			}
			ost := stGrey
			if p.Chosen[i][j] {
				ost = stText
			}
			out = append(out, wrapDeco(text, w, "          ", ost)...)
		}
	}
	return out
}

// answeredLines are one grey line per question: the question and its answer.
func answeredLines(qs []Question, answers map[string]string, w int) []line {
	var out []line
	for _, q := range qs {
		if a, ok := answers[q.Text]; ok {
			out = append(out, wrapDeco("  "+q.Text+" → "+a, w, "    ", stGrey)...)
		}
	}
	return out
}

// wrapDeco is decoration text wrapped to w, its next rows under lead.
func wrapDeco(text string, w int, lead string, st *lipgloss.Style) []line {
	var out []line
	for j, l := range wrap(text, max(w-len(lead), 1)) {
		if j > 0 {
			l = lead + l
		}
		out = append(out, decoLine(l, st))
	}
	return out
}

// pickOf is the pick to draw for ap: ui's, or an empty one when it belongs to
// another approval.
func pickOf(ui UI, ap *state.Approval, qs []Question) Pick {
	if ui.Pick.Valid(ap.ID, qs) {
		return ui.Pick
	}
	return NewPick(ap.ID, qs)
}
