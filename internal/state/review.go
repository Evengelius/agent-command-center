package state

import "time"

// The turn review (plan 65, brief amendment 67): for each stage, the
// project's files at its start and at its end, as git sees them, compared.
// The application takes the snapshots and reads the files when the state
// asks for them (CmdTurnSnapshot, CmdTurnFiles) and gives back what they
// gave (TurnSnapshot, TurnFiles); the state keeps it, and zone B draws it
// under the turn (ui-design.md, Turn review). Nothing of it is a message of
// the conversation (design D7).

// ReviewFileCap is how many of a turn's files the state keeps; the others
// are counted (design D7: "first 1,000 of N files").
const ReviewFileCap = 1000

// reviewNotice is the editor's row while the owner's message waits for its
// turn's start (design D4).
const reviewNotice = "the turn's snapshot…"

// reviewNoticeDone takes reviewNotice down once the start's snapshot is
// taken or the turn runs; any other notice stays.
func (s *State) reviewNoticeDone() {
	if s.Turn.Notice == reviewNotice {
		s.Turn.Notice = ""
	}
}

// ReviewState is whether the session's turns are reviewed (design D2): On
// once the application has made the session's store. Problem says why it
// could not be made; "" when On, and outside a git repository or without
// git, where no turn is reviewed and zone B says nothing (design D6).
type ReviewState struct {
	On      bool
	Problem string
}

// TurnReview is a stage's review (designs D3 to D5).
type TurnReview struct {
	// From and To are the trees of its start's and of its end's snapshots,
	// "" until taken; FromLeft and ToLeft the files each left out.
	From, To         string
	FromLeft, ToLeft []ReviewLeft
	// Asked says its start's snapshot was asked for; Prev that its start is
	// the stage before's end, a turn the CLI opened (design D4); Ended that
	// its end's snapshot was asked for.
	Asked, Prev, Ended bool
	// Files are the files it changed, the first ReviewFileCap; Total how many
	// it changed.
	Files []ReviewFile
	Total int
	// Left are the files left out whose size or time changed during it,
	// listed without their content (design D6).
	Left []ReviewLeft
	// Changing says the listing after its end found a file written while the
	// snapshot was taken or just after it (design D5).
	Changing bool
	// Problem is why it has no review: a snapshot that failed or was not
	// taken, the store's limit, the store that could not be made.
	Problem string
	// Done says its files are read, or Problem says why there are none.
	Done bool
}

// Shown says the review has a row in zone B: files changed, files left out
// that changed, or why it has no review. A turn that changed nothing has
// none (design D7).
func (r TurnReview) Shown() bool {
	return r.Done && (r.Total > 0 || len(r.Left) > 0 || r.Problem != "")
}

// ReviewFile is a file a turn changed, as git's diff-tree of its two trees
// gives it: the Git Graph's letter, path, rename source and counts.
type ReviewFile struct {
	Status         byte // 'A', 'M', 'D', 'R' or 'T'
	Path, Orig     string
	Added, Removed int
	Binary         bool
}

// ReviewLeft is a file a snapshot left out, never read (design D2): its
// path from the repository's top level, why, and its size and time, which
// tell whether it changed.
type ReviewLeft struct {
	Path, Why string
	Size      int64
	ModTime   time.Time
}

// The review's actions, what the application's commands gave.
type (
	// ReviewStore is the session's store made, On, or not: Problem why, ""
	// outside a git repository or without git.
	ReviewStore struct {
		On      bool
		Problem string
	}
	// TurnSnapshot is the snapshot of turn Turn's start, or its End: its
	// tree, the files it left out, for its end whether the folder was still
	// changing, or Problem, why it was not taken (designs D3 to D5).
	TurnSnapshot struct {
		Turn     int
		End      bool
		Tree     string
		Left     []ReviewLeft
		Changing bool
		Problem  string
	}
	// TurnFiles is the files turn Turn changed, Total of them, or Problem.
	TurnFiles struct {
		Turn    int
		Files   []ReviewFile
		Total   int
		Problem string
	}
)

func (ReviewStore) action()  {}
func (TurnSnapshot) action() {}
func (TurnFiles) action()    {}

// reviewStore records whether the session's turns are reviewed.
func (s *State) reviewStore(a ReviewStore) {
	s.Reviewing = ReviewState{On: a.On, Problem: a.Problem}
}

// reviewStart asks for the start's snapshot of a stage nothing asked one
// for: a prompt a mock file replays, which the dashboard did not send, or a
// turn the CLI opened with no stage before it (design D4). The owner's send
// asks for its own before the message is written (send).
func (s *State) reviewStart() []Command {
	n := len(s.Stages)
	if !s.Reviewing.On || n == 0 {
		return nil
	}
	st := &s.Stages[n-1]
	if !st.Open || st.Review.Asked || st.Review.Prev || st.Review.Done {
		return nil
	}
	st.Review.Asked = true
	return []Command{{Kind: CmdTurnSnapshot, Turn: st.N}}
}

// reviewOpen gives a stage opening its review's start: the snapshot the send
// asked for, or, for a turn the CLI opened, the stage before's end, so that
// a background agent's work shows with its report (design D4).
func (s *State) reviewOpen(st *Stage) {
	st.Review.Asked, s.Turn.ReviewAsked = s.Turn.ReviewAsked, false
	n := len(s.Stages)
	if !st.ByCLI || n == 0 || st.Review.Asked || !s.Reviewing.On {
		return
	}
	prev := s.Stages[n-1].Review
	if !prev.Asked && !prev.Prev {
		return // the stage before was not reviewed: this one asks for its own
	}
	st.Review.Prev, st.Review.From, st.Review.FromLeft = true, prev.To, prev.ToLeft
	if prev.Done && prev.To == "" {
		// The stage before has no end and will have none: its start failed,
		// so its end was never asked for, or its end failed.
		st.Review.Problem, st.Review.Done = "no snapshot at its start: "+prev.Problem, true
	}
}

// reviewEnd asks for the end's snapshot of the stage the turn closes, when
// its start was asked for or is the stage before's end. With the store that
// could not be made, the stage says so instead (design D7).
func (s *State) reviewEnd() []Command {
	n := len(s.Stages)
	if n == 0 {
		return nil
	}
	st := &s.Stages[n-1]
	if !st.Open || st.Review.Ended || st.Review.Done {
		return nil
	}
	if !s.Reviewing.On {
		if s.Reviewing.Problem != "" {
			st.Review.Problem, st.Review.Done = s.Reviewing.Problem, true
		}
		return nil
	}
	if !st.Review.Asked && !st.Review.Prev {
		return nil
	}
	st.Review.Ended = true
	return []Command{{Kind: CmdTurnSnapshot, Turn: st.N, End: true}}
}

// turnSnapshot keeps a snapshot the application took (designs D4 and D5):
// its start's or its end's tree, the files it left out, and, for its end,
// whether the folder was still changing. Its end's tree is also the start
// of the turn after when the CLI opened that one.
func (s *State) turnSnapshot(a TurnSnapshot) []Command {
	if a.Turn < 1 || a.Turn > len(s.Stages) {
		s.doNote("turn snapshot for no turn")
		return nil
	}
	r := &s.Stages[a.Turn-1].Review
	if !a.End {
		s.reviewNoticeDone()
	}
	if r.Done {
		return nil
	}
	if !a.End {
		r.From, r.FromLeft = a.Tree, a.Left
		if a.Problem != "" {
			r.Problem, r.Done = a.Problem, true
			return nil
		}
		return s.reviewFiles(a.Turn)
	}
	r.To, r.ToLeft, r.Changing = a.Tree, a.Left, a.Changing
	if a.Problem != "" {
		r.Problem, r.Done = a.Problem, true
	}
	cmds := s.reviewFiles(a.Turn)
	if a.Turn < len(s.Stages) {
		if next := &s.Stages[a.Turn].Review; next.Prev && next.From == "" && !next.Done {
			if a.Problem != "" {
				next.Problem, next.Done = "no snapshot at its start: "+a.Problem, true
			} else {
				next.From, next.FromLeft = a.Tree, a.Left
				cmds = append(cmds, s.reviewFiles(a.Turn+1)...)
			}
		}
	}
	return cmds
}

// reviewFiles asks for stage n's files once both its snapshots are taken.
// Two equal trees changed nothing: no file is read, and only the files left
// out that changed are left to show.
func (s *State) reviewFiles(n int) []Command {
	r := &s.Stages[n-1].Review
	if r.Done || r.From == "" || r.To == "" {
		return nil
	}
	r.Left = changedLeft(r.FromLeft, r.ToLeft)
	if r.From == r.To {
		r.Done = true
		return nil
	}
	return []Command{{Kind: CmdTurnFiles, Turn: n, From: r.From, To: r.To}}
}

// turnFiles keeps the files a turn changed, the first ReviewFileCap.
func (s *State) turnFiles(a TurnFiles) {
	if a.Turn < 1 || a.Turn > len(s.Stages) {
		s.doNote("turn files for no turn")
		return
	}
	r := &s.Stages[a.Turn-1].Review
	if r.Done {
		return
	}
	r.Files, r.Total, r.Problem, r.Done = a.Files[:min(len(a.Files), ReviewFileCap)], a.Total, a.Problem, true
}

// changedLeft is the files the end's snapshot left out that the start's did
// not, or with another size or time: those the turn changed, whose content
// is never read (design D6).
func changedLeft(from, to []ReviewLeft) []ReviewLeft {
	was := make(map[string]ReviewLeft, len(from))
	for _, l := range from {
		was[l.Path] = l
	}
	var out []ReviewLeft
	for _, l := range to {
		if w, ok := was[l.Path]; !ok || w.Size != l.Size || !w.ModTime.Equal(l.ModTime) {
			out = append(out, l)
		}
	}
	return out
}
