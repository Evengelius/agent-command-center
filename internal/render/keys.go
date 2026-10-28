package render

import "github.com/Evengelius/agent-command-center/internal/state"

// keyGroup is one table of the key overlay (ui-design.md, Focus and keys).
type keyGroup struct {
	title string
	keys  [][2]string // key, action
}

// KeyTable is what F1 shows with the default keys: the keys that work, by
// where they work. A key ui-design.md lists but the MVP does not have is left
// out here and listed under the departures there.
var KeyTable = keyTable(nil, nil)

// newlineWhy is what F1 says of a newline key whose use is known.
var newlineWhy = map[string]string{
	"Ctrl+J":      "newline (works in every terminal)",
	"Shift+Enter": "newline (Windows Terminal, through win32-input-mode)",
	"Alt+Enter":   "newline, except on Windows Terminal (full screen)",
}

// keyTable is KeyTable for the send and newline keys in force (plan 21, item
// 5.1): one row per send key and one per newline key, then `\ then` the first
// send key. None given is the defaults.
func keyTable(send, newline []string) []keyGroup {
	if len(send) == 0 {
		send = defaultSendKeys
	}
	if len(newline) == 0 {
		newline = defaultNewlineKeys
	}
	var rows [][2]string
	for _, k := range send {
		rows = append(rows, [2]string{k, "send the draft (in the editor)"})
	}
	for _, k := range newline {
		why := newlineWhy[k]
		if why == "" {
			why = "newline"
		}
		rows = append(rows, [2]string{k, why})
	}
	rows = append(rows, [2]string{`\ then ` + send[0], "newline (Claude Code's convention)"})
	g := fixedKeyTable()
	g[0].keys = append(rows, g[0].keys...)
	return g
}

// fixedKeyTable is the rest of KeyTable, which the configuration does not
// change.
func fixedKeyTable() []keyGroup {
	return []keyGroup{
		{"ANYWHERE", [][2]string{
			{"Ctrl+C", "copy the selection; else interrupt the running turn"},
			{"Ctrl+Shift+C", "copy the selection"},
			{"Ctrl+O", "copy the orchestrator's last message"},
			{"Ctrl+G", "draft window: the draft larger; in it Ctrl+F full screen, Esc back"},
			{"Ctrl+W", "worktrees: list, diff, merge, remove"},
			{"F2", "go to the oldest pending approval"},
			{"F3", "the file in front in the file viewer; else the Explorer, shown or hidden (its icon on the right too)"},
			{"F4", "the modes selector (a click on the mode pill too)"},
			{"Alt+P", "the model selector (a click on the pill too)"},
			{"Alt+B", "the branches popup (a click on the branch name too)"},
			{"F6 / Shift+F6", "next / previous zone (B, the Explorer or Source Control while shown, C, A)"},
			{"Tab / Shift+Tab", "next / previous zone, outside the editor"},
			{"Esc", "close the overlay, leave zoom, inspection or a full view, clear the selection, back to the input"},
			{"Shift+PgUp / Shift+PgDn", "scroll the conversation from any focus"},
			{"F12", "toggle mouse capture (native terminal selection when off)"},
			{"F1, ?", "this overlay (? outside the editor)"},
			{"Ctrl+R", "after a crash: resume the same CLI session"},
			{"Ctrl+Q", "quit; asks to confirm while a turn or an agent runs"},
		}},
		{"A FOCUSED PANEL (A, C, INSPECTION)", [][2]string{
			{"↑ ↓", "move the cursor (node, agent, line); in A through the turns, then TELEMETRY and its sections"},
			{"← →", "in A fold and unfold a turn, SESSION USAGE or QUOTA WINDOWS; in inspection change tab"},
			{"1 2 3 4", "inspection: Messages, Tools, Diff, Raw events"},
			{"Enter", "inspect the agent; in A on TELEMETRY, the errors and every value"},
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
			{"↑ ↓ PgUp PgDn Home End", "move the cursor line; the wheel scrolls, a click on a line moves it there"},
			{"← →", "shift the code sideways"},
			{"/", "search, case ignored without a capital; Enter finds"},
			{"n N", "next / previous match"},
			{"g", "go to a line number"},
			{"f", "full screen or the window"},
			{"b", "git blame: each line's commit date and author beside it, the cursor's commit in the footer; b again hides it"},
			{"Esc", "clear the selection, else back to the Explorer, on the file"},
			{"F3", "close it, and show or hide the Explorer"},
		}},
		{"THE EXPLORER (F3 shows or hides it, with the focus)", [][2]string{
			{"↑ ↓ PgUp PgDn Home End", "move; the wheel over it too, a click on a row"},
			{"→ ←", "open a folder, or its first entry; close it, or go to its folder"},
			{"Enter", "open or close a folder; show a file in the file viewer"},
			{"/", "the quick open: type to narrow, a click selects a file, Enter or a click on the selected one opens it, Esc back"},
			{"Esc", "close the folder the cursor is in; at the top, back to the input, the Explorer kept beside it (hidden where it takes the conversation's place)"},
			{"F3", "hide the Explorer"},
		}},
		{"SOURCE CONTROL (its icon on the right shows or hides it, with the focus)", [][2]string{
			{"↑ ↓ PgUp PgDn Home End", "move; the wheel over it too, a click on a file opens its diff"},
			{"→ ←", "open or fold a section or an untracked folder; ← on a file, to its section"},
			{"Enter", "a file's diff, in the conversation's place; on a section, fold or open it"},
			{"r", "read git's status again"},
			{"Ctrl+click", "select a file for Commit, or deselect it; the first one selected opens its diff"},
			{"Esc", "clear the selection; with none, back to the input, Source Control kept beside it (hidden where it takes the conversation's place)"},
		}},
		{"A FILE'S DIFF (Enter or a click on a file in Source Control)", [][2]string{
			{"↑ ↓ PgUp PgDn Home End", "scroll; the wheel too"},
			{"← →", "shift the lines sideways"},
			{"n N", "the next / previous change"},
			{"Tab", "the file's other comparison, staged or working tree; else the next zone"},
			{"f", "wide, over zone A, and back"},
			{"r", "compare again"},
			{"Esc", "back to Source Control, the conversation as it was"},
		}},
		{"THE MODEL SELECTOR (Alt+P or a click on the pill)", [][2]string{
			{"↑ ↓ Home End", "move over the models, More models, Effort and Ultracode"},
			{"Enter, Space", "choose the model, expand or collapse More models, turn Ultracode on or off"},
			{"← →", "on Effort, the level before or after; a click on the slider picks a level"},
			{"PgUp PgDn", "scroll the list a page; the wheel scrolls too"},
			{"Esc", "close the selector"},
			{"Alt+P", "close the selector"},
			{"any other key", "close the selector, then act as usual (Ctrl+Q quits)"},
		}},
		{"THE BRANCHES POPUP (Alt+B or a click on the branch name)", [][2]string{
			{"↑ ↓ PgUp PgDn Home End", "move over the groups, branches and tags; the wheel moves three rows"},
			{"→ ←", "open or fold a group; ← on a row, to its group"},
			{"Enter", "switch to the branch, confirmed first when changes would come along (a double click too); on a group, fold or open it"},
			{"typing, Backspace", "search the branches and tags, case ignored; a paste too"},
			{"Ctrl+C", "copy the name of the branch or tag"},
			{"F5", "fetch every remote (a click on fetch too)"},
			{"Esc", "stop a fetch, cancel a confirmation, clear the search, else close"},
			{"Alt+B", "close the popup"},
			{"any other key", "close the popup, then act as usual"},
		}},
		{"THE EDITOR", [][2]string{
			{"← → ↑ ↓ Home End", "move; Ctrl+Home and Ctrl+End the whole draft"},
			{"Ctrl+← Ctrl+→", "move by word"},
			{"Shift + a move", "select"},
			{"Ctrl+A", "select all"},
			{"Backspace Delete", "delete; with Ctrl, a word"},
			{"Ctrl+Z / Ctrl+Y", "undo / redo"},
			{"Ctrl+X Ctrl+C Ctrl+V", "cut, copy, paste"},
			{"Alt+V", "attach the clipboard's image (Ctrl+V too, when the clipboard holds no text)"},
			{"Ctrl+Shift+X", "remove the last attached image (a click on a chip's ✕ removes it)"},
			{"Tab", "insert a tab"},
			{"PgUp PgDn", "scroll the conversation"},
		}},
	}
}

// keyCol is the width of the overlay's key column.
const keyCol = 25

// keysBody is the overlay's body with the default keys.
func keysBody(w int) []line { return keysBodyFor(KeyTable, w) }

func keysBodyFor(table []keyGroup, w int) []line {
	var body []line
	for i, g := range table {
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
	return fullScreen(s, ui, sc, head, keysBodyFor(keyTable(ui.SendKeys, ui.NewlineKeys), ui.Width-2), scrollKeys)
}
