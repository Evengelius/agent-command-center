package state

import (
	"testing"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// TestSentImages is plan 31's reducer part (design D5): a message's images
// ride on the write command and stay with the turn until it is accepted;
// the entry the written line makes is named from them; a failed write gives
// the text and the images back.
func TestSentImages(t *testing.T) {
	ready := func() *State {
		s := New()
		s.Screen, s.Session.Phase, s.Turn.Phase = ScreenDashboard, Ready, Idle
		return s
	}
	imgs := []Attachment{{Path: `C:\a\one.png`, Name: "one.png", Size: 10, Media: "image/png"}, {Path: `C:\a\two.jpg`, Name: "two.jpg", Size: 20, Media: "image/jpeg"}}

	s := ready()
	cmds := s.Do(Send{Text: "look", Images: imgs})
	if len(cmds) != 1 || cmds[0].Kind != CmdWrite || len(cmds[0].Images) != 2 || cmds[0].Images[1].Name != "two.jpg" {
		t.Fatalf("commands %+v", cmds)
	}
	s.Apply(event.Event{Seq: 1, Kind: event.UserMessage, Payload: event.User{Text: "look", Images: []event.Image{{Media: "image/png", Size: 10}, {Media: "image/jpeg", Size: 20}}}})
	e := s.Conversation[len(s.Conversation)-1]
	if e.Role != "user" || len(e.Images) != 2 || e.Images[0].Name != "one.png" || e.Images[1].Name != "two.jpg" || e.Images[1].Size != 20 {
		t.Errorf("the entry %+v", e)
	}
	s.Apply(event.Event{Seq: 2, Kind: event.TurnStarted})
	if s.Turn.Draft != "" || s.Turn.DraftImages != nil {
		t.Errorf("accepted: draft %q, images %+v", s.Turn.Draft, s.Turn.DraftImages)
	}

	// A failed write gives both back.
	s = ready()
	s.Do(Send{Text: "", Images: imgs})
	cmds = s.Do(WriteFailed{Message: "two.jpg: the file is gone"})
	if len(cmds) != 1 || cmds[0].Kind != CmdRestoreDraft || cmds[0].Text != "" || len(cmds[0].Images) != 2 || s.Turn.DraftImages != nil {
		t.Errorf("restore %+v, turn images %+v", cmds, s.Turn.DraftImages)
	}

	// A line with a different count keeps its images unnamed; a CLI's own
	// prompt (no draft) too.
	s = ready()
	s.Do(Send{Text: "x", Images: imgs[:1]})
	s.Apply(event.Event{Seq: 1, Kind: event.UserMessage, Payload: event.User{Text: "x", Images: []event.Image{{Media: "image/png"}, {Media: "image/png"}}}})
	if e := s.Conversation[len(s.Conversation)-1]; e.Images[0].Name != "" {
		t.Errorf("named from a different count: %+v", e.Images)
	}
}
