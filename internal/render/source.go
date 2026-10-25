package render

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// Source Control (plan 44, brief amendment 45, the owner's image
// references/14-source-control.png): the project's Git changes, listed in
// the Explorer's docked place while its tile is selected, one panel at a
// time (design D1). The application reads git's status in commands and
// holds the result here; this draws it (design D4).

// SourceView is the Source Control panel as the application holds it.
type SourceView struct {
	Open bool
	// Repo is the repository's name and Branch its branch line: a branch, or
	// "detached at <id>", "no commits yet on <branch>".
	Repo, Branch string
	// Note is said in the changes' place: "reading…", "No Git repository
	// found for this project.", "git not found", or git's own error (Failed).
	Note   string
	Failed bool
	// Entries are the changes, as git's status lists them, ignored paths
	// left out.
	Entries []SourceEntry
	// Folded holds the sections folded by their key, Opened the untracked
	// folders opened, Folders their files once read.
	Folded  map[string]bool
	Opened  map[string]bool
	Folders map[string]*SourceFolder
	// Cursor is the cursor's row by its key, Top the first row shown, Hover
	// the row under the pointer plus one (0 none).
	Cursor string
	Top    int
	Hover  int
}

// SourceEntry is one path in git's status: X the index's letter, Y the
// working tree's, '?' and '?' untracked; Path relative to the project, an
// untracked folder's ending in "/"; Orig a rename's or a copy's source.
type SourceEntry struct {
	X, Y       byte
	Path, Orig string
}

// SourceFolder is an untracked folder's files, relative to the project, or
// why there are none.
type SourceFolder struct {
	Files     []string
	Truncated bool
	Note      string
	Failed    bool
}

// The sections, in their order (design D4), with their titles.
const (
	SectionMerge   = "merge"
	SectionChanges = "changes"
	SectionStaged  = "staged"
)

var sectionTitles = map[string]string{SectionMerge: "MERGE CHANGES", SectionChanges: "CHANGES", SectionStaged: "STAGED CHANGES"}

// Conflict names an unmerged entry's state, "" for a merged one (git's
// unmerged pairs, git-status(1)).
func Conflict(e SourceEntry) string {
	switch string([]byte{e.X, e.Y}) {
	case "UU":
		return "both modified"
	case "AA":
		return "both added"
	case "DD":
		return "both deleted"
	case "AU":
		return "added by us"
	case "UA":
		return "added by them"
	case "DU":
		return "deleted by us"
	case "UD":
		return "deleted by them"
	}
	return ""
}

// Untracked is true for an untracked entry.
func Untracked(e SourceEntry) bool { return e.X == '?' && e.Y == '?' }

// Sections are the sections entry e appears in: MERGE CHANGES for a
// conflict; else STAGED CHANGES when the index changed it and CHANGES when
// the working tree did, both for a file staged and changed again since; an
// untracked path in CHANGES.
func Sections(e SourceEntry) []string {
	switch {
	case Conflict(e) != "":
		return []string{SectionMerge}
	case Untracked(e):
		return []string{SectionChanges}
	case e.X == '!':
		return nil
	}
	var out []string
	if e.X != ' ' {
		out = append(out, SectionStaged)
	}
	if e.Y != ' ' {
		out = append(out, SectionChanges)
	}
	return out
}

// SourceLetter is entry e's status letter in section: the index's in
// STAGED CHANGES, the working tree's in CHANGES ("U" untracked), "!" for a
// conflict.
func SourceLetter(e SourceEntry, section string) byte {
	switch {
	case section == SectionMerge:
		return '!'
	case Untracked(e):
		return 'U'
	case section == SectionStaged:
		return e.X
	}
	return e.Y
}

// SourceRow is a row of the panel's list: a section's header, a file, an
// untracked folder, a file in an opened untracked folder, a note, or a
// blank row around a header, as the owner's example lays them out.
type SourceRow struct {
	Key     string
	Kind    byte // 'h' header, 'f' file, 'd' untracked folder, 'u' a file in it, 'n' note, 'b' blank
	Section string
	Entry   int // the entry's index, -1 for a header, a note or a folder's file
	Letter  byte
	Path    string // project-relative; a folder's ends in "/"
	Orig    string
	Open    bool // a header unfolded, a folder opened
	Count   int  // a header's files
	Note    string
	Failed  bool
}

// Selectable is true for a row the cursor stops on: a header, a file or a
// folder, not a note or a blank row.
func (r SourceRow) Selectable() bool { return r.Kind != 'n' && r.Kind != 'b' }

// SourceRows is the list as drawn, after the owner's example (brief
// amendment 45): each section with entries, a blank row, its header, then
// unless folded a blank row and its files in git's order; an untracked
// folder opened its files under it, or its note.
func SourceRows(v SourceView) []SourceRow {
	var rows []SourceRow
	blank := func() { rows = append(rows, SourceRow{Key: fmt.Sprint("b:", len(rows)), Kind: 'b', Entry: -1}) }
	for _, sec := range []string{SectionMerge, SectionChanges, SectionStaged} {
		var files []SourceRow
		for i, e := range v.Entries {
			for _, s := range Sections(e) {
				if s != sec {
					continue
				}
				r := SourceRow{Kind: 'f', Section: sec, Entry: i, Letter: SourceLetter(e, sec), Path: e.Path}
				if r.Letter == 'R' || r.Letter == 'C' {
					r.Orig = e.Orig // the source of the section's rename or copy, not another's
				}
				if Untracked(e) && strings.HasSuffix(e.Path, "/") {
					r.Kind, r.Open = 'd', v.Opened[e.Path]
				}
				r.Key = "f:" + sec + ":" + e.Path
				files = append(files, r)
			}
		}
		if len(files) == 0 {
			continue
		}
		open := !v.Folded[sec]
		blank()
		rows = append(rows, SourceRow{Key: "h:" + sec, Kind: 'h', Section: sec, Entry: -1, Open: open, Count: len(files)})
		if !open {
			continue
		}
		blank()
		for _, f := range files {
			rows = append(rows, f)
			if f.Kind != 'd' || !f.Open {
				continue
			}
			d := v.Folders[f.Path]
			switch {
			case d == nil:
				rows = append(rows, SourceRow{Key: "n:" + f.Path, Kind: 'n', Section: sec, Entry: -1, Note: "reading…"})
			case d.Note != "":
				rows = append(rows, SourceRow{Key: "n:" + f.Path, Kind: 'n', Section: sec, Entry: -1, Note: d.Note, Failed: d.Failed})
			default:
				for _, p := range d.Files {
					rows = append(rows, SourceRow{Key: "u:" + p, Kind: 'u', Section: sec, Entry: -1, Letter: 'U', Path: p})
				}
				if d.Truncated {
					rows = append(rows, SourceRow{Key: "n:" + f.Path, Kind: 'n', Section: sec, Entry: -1, Note: fmt.Sprintf("first %d files", len(d.Files))})
				}
			}
		}
	}
	return rows
}

// SourceCursor is the row of rows the cursor is on: the one with key, else
// the first selectable; -1 with none.
func SourceCursor(rows []SourceRow, key string) int {
	first := -1
	for i, r := range rows {
		if r.Key == key && r.Selectable() {
			return i
		}
		if first < 0 && r.Selectable() {
			first = i
		}
	}
	return first
}

// SourcePanel is the Source Control panel's letter (design D4). Each row's
// cells carry its index in SourceRows plus one, the header's "⌄"
// ItemPanelHide, the other cells 0.
const SourcePanel = 'G'

// sourceFixed are the panel's rows outside its list: the frame's two edges,
// the header, the rule and the repository's row.
const sourceFixed = 5

// SourceRoom is how many rows of the list the panel shows on this screen,
// what PgUp and PgDn move by.
func SourceRoom(s *state.State, ui UI) int { return max(layout(s, ui).upper-sourceFixed, 0) }

// sourcePanel is the Source Control panel w by h (design D4): the header
// "SOURCE CONTROL" with its "⌄", a rule, the repository's row, then the
// sections or the note in their place, scrolling to keep the cursor in
// view; its frame in the title colour while it has the focus.
func sourcePanel(ui UI, w, h int) []line {
	v := ui.Source
	inner := w - 4
	body := []line{
		spread(decoLine("SOURCE CONTROL", stTitle), line{sourceDeco("⌄", stTitle, ItemPanelHide)}, inner),
		teleRule(inner),
		repoRow(v, inner),
	}
	room := max(h-sourceFixed, 0)
	rows := SourceRows(v)
	switch {
	case v.Note != "":
		// A blank row, as before a section, then the note whole, on as many
		// rows as it takes.
		body = append(body, nil)
		body = append(body, wrapDeco(v.Note, inner, "", either(v.Failed, stAttention, stGrey))...)
	case len(rows) == 0:
		body = append(body, nil, decoLine("No changes", stGrey))
	default:
		cur := SourceCursor(rows, v.Cursor)
		top := ListTop(v.Top, max(cur, 0), room, len(rows))
		for i := top; i < top+room && i < len(rows); i++ {
			r := sourceRow(rows[i], i == cur, v.Hover == i+1, inner)
			for j := range r {
				r[j].ref = ref{panel: SourcePanel, kind: refDeco, item: i + 1}
			}
			body = append(body, r)
		}
	}
	return box(SourcePanel, nil, nil, body, w, h, borderFor(ui.Focus == "R"))
}

// repoRow is the repository's row: its name in a folder's colour and its
// branch line muted; empty before git answers.
func repoRow(v SourceView, w int) line {
	if v.Repo == "" {
		return nil
	}
	l := line{deco(v.Repo, stExpDirName)}
	if v.Branch != "" {
		l = append(l, deco(" · "+v.Branch, stMuted))
	}
	return l.fit(w)
}

// sourceDeco is a span of the panel: decoration, panel G, its item.
func sourceDeco(text string, st *lipgloss.Style, item int) span {
	return span{text: text, st: st, ref: ref{panel: SourcePanel, kind: refDeco, item: item}}
}

// letterStyle is a status letter's colour (design D4): modified amber;
// added, untracked, renamed and copied green; deleted and a conflict red; a
// type change blue.
func letterStyle(c byte) *lipgloss.Style {
	switch c {
	case 'M':
		return stProgress
	case 'A', 'U', 'R', 'C':
		return stDone
	case 'D', '!':
		return stFailed
	case 'T':
		return stInfo
	}
	return stGrey
}

// sourceRow is row r, w cells: a header "▾ CHANGES" in the title colour,
// bold, with its count at the right; a file three cells in, as in the
// owner's example, its status letter, its directory muted and its name, a
// rename's source after "←"; an untracked folder with "›" or "⌄"; a file of
// an opened folder two cells deeper; a note in grey; a blank row. The
// cursor's row on the purple band, the hovered one on a subtler ground.
func sourceRow(r SourceRow, cur, hover bool, w int) line {
	if r.Kind == 'b' {
		return blankRow(w)
	}
	g := (*ground)(nil)
	switch {
	case cur:
		g = bandGround
	case hover:
		g = sourceHoverGround
	}
	on := func(st *lipgloss.Style) *lipgloss.Style {
		if g == nil {
			return st
		}
		return g.on(st)
	}
	var l, right line
	switch r.Kind {
	case 'h':
		mark := "▾ "
		if !r.Open {
			mark = "▸ "
		}
		l = line{deco(mark, on(stTitle)), deco(sectionTitles[r.Section], on(stTitleBold))}
		right = line{deco(fmt.Sprint(r.Count), on(stMuted))}
	case 'n':
		l = line{deco("     "+r.Note, on(either(r.Failed, stAttention, stGrey)))}
	default:
		indent := "   "
		if r.Kind == 'u' {
			indent = "     "
		}
		l = line{deco(indent, on(stNone)), deco(string(r.Letter), on(letterStyle(r.Letter))), deco("  ", on(stNone))}
		if r.Kind == 'd' {
			mark := "› "
			if r.Open {
				mark = "⌄ "
			}
			l = append(l, deco(mark, on(stGrey)))
		}
		dir, name := splitPath(r.Path)
		room := w - l.width()
		if r.Orig != "" && textWidth(r.Path+" ← "+r.Orig) > room {
			room -= textWidth(" ← …") // the source shown cut, the name whole
		}
		if dir = shortDir(dir, name, room); dir != "" {
			l = append(l, deco(dir, on(stGrey)))
		}
		l = append(l, deco(name, on(stExpFileName)))
		if r.Orig != "" {
			l = append(l, deco(" ← "+r.Orig, on(stGrey)))
		}
	}
	if right != nil {
		l = spreadOn(l.fit(max(w-right.width()-1, 1)), right, w, on(stNone))
	} else {
		l = l.fit(w)
	}
	for i := range l {
		if g != nil && l[i].st == stNone {
			l[i].st = on(stNone) // fit's padding on the row's ground too
		}
	}
	return l
}

// shortDir is dir as a row room cells wide shows it before name: whole when
// both fit, else its leading folders given way to "…/" so that the name
// stays in view, "" when even "…/" leaves no room.
func shortDir(dir, name string, room int) string {
	if textWidth(dir)+textWidth(name) <= room {
		return dir
	}
	segs := strings.Split(strings.TrimSuffix(dir, "/"), "/")
	for k := 1; k <= len(segs); k++ {
		d := "…/" + strings.Join(segs[k:], "/")
		if k < len(segs) {
			d += "/"
		}
		if textWidth(d)+textWidth(name) <= room {
			return d
		}
	}
	return ""
}

// splitPath is a project-relative path's directory, with its "/", and its
// name; a folder's name keeps its "/".
func splitPath(p string) (dir, name string) {
	trim := strings.TrimSuffix(p, "/")
	i := strings.LastIndex(trim, "/")
	if i < 0 {
		return "", p
	}
	return p[:i+1], p[i+1:]
}

// sourceHints are the panel's key hints in the footer while it has the
// focus, the longest first: the first that fits shows (design D12).
var sourceHints = []string{
	"↑↓ PgUp PgDn Home End move · → open · ← fold · Enter diff · r refresh · Esc back to the input · F6 zone · F3 Explorer · F1 keys",
	"↑↓ move · → open · ← fold · Enter diff · r refresh · Esc back to the input · F6 zone · F3 Explorer · F1 keys",
	"↑↓ move · ←→ fold · Enter diff · r refresh · Esc back · F3 Explorer · F1 keys",
	"↑↓ · Enter diff · r refresh · Esc back · F3 Explorer",
}

// sourceKeys is the footer's text while the panel has the focus, on a
// screen w columns wide.
func sourceKeys(w int) string {
	for _, k := range sourceHints {
		if textWidth(k)+1 <= w {
			return k
		}
	}
	return sourceHints[len(sourceHints)-1]
}
