package render

import "github.com/Evengelius/agent-command-center/internal/state"

// keyGroup is one table of the key overlay (ui-design.md, Focus and keys).
type keyGroup struct {
	title string
	keys  [][2]string // key, action
}

// KeyTable is what F1 shows: the keys that work, by where they work. A key
// ui-design.md lists but the MVP does not have is left out here and listed
// under the departures there.
var KeyTable = []keyGroup{
	{"ANYWHERE", [][2]string{
		{"Enter", "send the draft (in the editor)"},
		{"Ctrl+J", "newline (works in every terminal)"},
		{"Shift+Enter", "newline (Windows Terminal, through win32-input-mode)"},
		{"Alt+Enter", "newline, except on Windows Terminal (full screen)"},
		{`\ then Enter`, "newline (Claude Code's convention)"},
		{"Ctrl+C", "copy the selection; else interrupt the running turn"},
		{"Ctrl+Shift+C", "copy the selection"},
		{"Ctrl+O", "copy the orchestrator's last message"},
		{"Ctrl+G", "draft window: the draft larger; in it Ctrl+F full screen, Esc back"},
		{"Ctrl+W", "worktrees: list, diff, merge, remove"},
		{"F2", "go to the oldest pending approval"},
		{"F3", "the file in front in the file viewer; else the Explorer; in them, close"},
		{"F4", "approvals Manual / Auto"},
		{"F6 / Shift+F6", "next / previous zone (B, C, A, D)"},
		{"Tab / Shift+Tab", "next / previous zone, outside the editor"},
		{"Esc", "close the overlay, leave zoom, inspection or a full view, clear the selection, back to the input"},
		{"Shift+PgUp / Shift+PgDn", "scroll the conversation from any focus"},
		{"F12", "toggle mouse capture (native terminal selection when off)"},
		{"F1, ?", "this overlay (? outside the editor)"},
		{"Ctrl+R", "after a crash: resume the same CLI session"},
		{"Ctrl+Q", "quit; asks to confirm while a turn or an agent runs"},
	}},
	{"A FOCUSED PANEL (A, C, D, INSPECTION)", [][2]string{
		{"↑ ↓", "move the cursor (node, agent, line)"},
		{"← →", "in A fold and unfold; in inspection change tab"},
		{"1 2 3 4", "inspection: Messages, Tools, Diff, Raw events"},
		{"Enter", "inspect the agent; in D, list errors and telemetry"},
		{"z", "zoom the panel full screen; z or Esc again restores"},
		{"PgUp PgDn Home End", "page, top, end; End follows again (A, inspection)"},
		{"[ ]", "previous / next agent (inspection)"},
	}},
	{"THE APPROVAL CARD (F2 gives it the focus)", [][2]string{
		{"a", "allow once"},
		{"s", "allow for the session, when a Codex request offers it"},
		{"d", "deny / decline"},
		{"c", "cancel the turn"},
		{"Enter", "the whole command, diff or question, full screen; a s d c answer there too"},
		{"↑ ↓", "previous / next pending card"},
		{"Esc", "later: back to the input, still pending"},
	}},
	{"B'S ENTRIES (↑ from an empty input)", [][2]string{
		{"↑ ↓", "previous / next folded message, tool, failure or run of tool events; ↓ past the last, the input"},
		{"Enter", "the whole entry, full screen; Esc back to the cursor; on a run of tool events, expand it (a click too)"},
		{"→", "on a run of tool events, expand it; it opens nothing"},
		{"←", "on a call of an expanded run, fold the run again (a click on its row too)"},
		{"Esc, or typing", "back to the input"},
		{"F3", "the file a tool entry read or changed"},
	}},
	{"THE FILE VIEWER (F3)", [][2]string{
		{"↑ ↓ PgUp PgDn Home End", "move the cursor line; the wheel scrolls"},
		{"← →", "shift the code sideways"},
		{"/", "search, case ignored without a capital; Enter finds"},
		{"n N", "next / previous match"},
		{"g", "go to a line number"},
		{"f", "full screen or the window"},
		{"Esc", "clear the selection, else back to the Explorer, on the file"},
		{"F3", "close, back to the dashboard"},
	}},
	{"THE EXPLORER (F3 with no file in front)", [][2]string{
		{"↑ ↓ PgUp PgDn Home End", "move; the wheel too, a click on a row"},
		{"→ ←", "open a folder, or its first entry; close it, or go to its folder"},
		{"Enter", "open or close a folder; show a file in the file viewer"},
		{"/", "the quick open: type to narrow, Enter opens, Esc back"},
		{"Esc", "close the folder the cursor is in; at the top, close"},
		{"F3", "close, back to the dashboard"},
	}},
	{"THE EDITOR", [][2]string{
		{"← → ↑ ↓ Home End", "move; Ctrl+Home and Ctrl+End the whole draft"},
		{"Ctrl+← Ctrl+→", "move by word"},
		{"Shift + a move", "select"},
		{"Ctrl+A", "select all"},
		{"Backspace Delete", "delete; with Ctrl, a word"},
		{"Ctrl+Z / Ctrl+Y", "undo / redo"},
		{"Ctrl+X Ctrl+C Ctrl+V", "cut, copy, paste"},
		{"Tab", "insert a tab"},
		{"PgUp PgDn", "scroll the conversation"},
	}},
}

// keyCol is the width of the overlay's key column.
const keyCol = 25

func keysBody(w int) []line {
	var body []line
	for i, g := range KeyTable {
		if i > 0 {
			body = append(body, line{})
		}
		body = append(body, plain(g.title, stBold))
		for _, k := range g.keys {
			lead := line{sp(padTo(k[0], keyCol), stTitle)}
			for j, l := range wrap(k[1], max(w-keyCol, 10)) {
				if j > 0 {
					lead = line{deco(padTo("", keyCol), stNone)}
				}
				body = append(body, append(lead, sp(l, stText)))
			}
		}
	}
	return body
}

// keysOverlay is the key overlay, full screen; it opens at its top and
// scrolls as the details view does.
func keysOverlay(s *state.State, ui UI, sc *Screen) []line {
	head := spread(line{deco(" ◀ back (Esc)   ", stTitle), deco("KEYS", stBold)},
		decoLine("F1 or Esc closes ", stGrey), ui.Width)
	return fullScreen(s, ui, sc, head, keysBody(ui.Width-2), scrollKeys)
}
