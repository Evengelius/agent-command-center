package render

// The file viewer's git blame (plan 60, brief amendment 62; ui-design.md,
// File viewer): left of the line numbers, each line's commit date and
// author, after the owner's image of an IDE's blame; in the footer the
// cursor line's commit. It claims what git says only: "Not committed" is a
// line that differs from the last commit, whoever changed it, and an author
// is the commit's (design D2).

// FileBlame is the viewer's blame as the application read it.
type FileBlame struct {
	// On is the switch (design D1): the column and the footer's commit show.
	On bool
	// Lines are each line's commit, by its index in Commits, -1 none; nil
	// while git reads or when it gave none.
	Lines   []int
	Commits []BlameCommit
	// Note is what the footer says in the commit's place: "blame…" while
	// git reads, or why there is no blame; NoteBad a failure (design D6).
	Note    string
	NoteBad bool
}

// BlameCommit is a commit as the viewer draws it, its texts made once when
// git answers.
type BlameCommit struct {
	Short, Author, Initials, Summary string
	// Date is "15/11/2022", DateShort "15/11/22", When "15/11/2022 10:00".
	Date, DateShort, When string
	Uncommitted           bool
}

// The column's forms and limits (designs D2 and D3).
const (
	blameFull    = 25 // the date, a space, the name, two spaces
	blameCompact = 14 // the short date, a space, the initials, two spaces
	blameName    = 12 // the most cells of a name
	blameCode    = 60 // the fewest cells of code beside the full column
)

// blameCols is the column's cells in a row inner cells wide whose numbers
// take nw: none with the blame off, the full form where the code keeps
// blameCode cells, else the compact one.
func blameCols(f FileView, inner, nw int) int {
	switch {
	case !f.Blame.On:
		return 0
	case inner-nw-2-blameFull >= blameCode:
		return blameFull
	}
	return blameCompact
}

// blameCell is line i's column, bw cells: its commit's date and author, or
// "Not committed" in the attention colour; blank while there is no blame.
// On the cursor's row it takes the row's ground.
func blameCell(b FileBlame, i, bw int, cur bool) line {
	st, unc := stGrey, stAttention
	if cur {
		st, unc = stFileRowNum, stFileRowMark
	}
	text := ""
	if i < len(b.Lines) && b.Lines[i] >= 0 && b.Lines[i] < len(b.Commits) {
		c := b.Commits[b.Lines[i]]
		switch {
		case c.Uncommitted && bw == blameFull:
			text, st = "Not committed", unc
		case c.Uncommitted:
			text, st = "uncommitted", unc
		case bw == blameFull:
			text = c.Date + " " + fitText(c.Author, blameName)
		default:
			text = c.DateShort + " " + c.Initials
		}
	}
	return line{deco(padTo(fitText(text, bw-2), bw), st)}
}

// blameFooter is the footer's left while the blame shows, room cells at
// most: the note, else the cursor line's commit, its short hash, author,
// date and time and subject, or that the line is not committed; nil when
// the line has no commit.
func blameFooter(b FileBlame, cursor, room int) line {
	var l line
	switch {
	case b.Note != "":
		st := stGrey
		if b.NoteBad {
			st = stAttention
		}
		l = line{deco(b.Note, st)}
	case cursor < 0 || cursor >= len(b.Lines) || b.Lines[cursor] < 0 || b.Lines[cursor] >= len(b.Commits):
		return nil
	case b.Commits[b.Lines[cursor]].Uncommitted:
		l = line{deco("Not committed yet: changed in the working tree", stAttention)}
	default:
		c := b.Commits[b.Lines[cursor]]
		dot := deco(" · ", stGrey)
		l = line{deco(c.Short, stTitle), dot, deco(c.Author, stText), dot, deco(c.When, stGrey), dot, deco(c.Summary, stText)}
	}
	if l.width() > room {
		l = l.fit(room)
	}
	return l
}
