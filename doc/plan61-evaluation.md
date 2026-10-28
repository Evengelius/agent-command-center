# Plan 61 evaluation (2026-10-28)

The check of plan 61, clicks in the file viewer and the quick open (brief
amendment 63): a click on a line of the viewer puts the cursor on it, the
keys having been the only way; in the quick open a click on a file
selects it and a click on the selected one opens it, as in the Explorer.

It covers the 7 criteria of [`plan.md`, Plan 61](../plan.md), written
before the code. Nothing is committed or pushed: the owner reviews first.

How it was checked:
- **Application tests** (`internal/app/fileclick_test.go`, 5 tests) on
  the file rig, every click drawn first as the terminal shows it, and in
  four places the frame left behind on purpose (a key, another file or a
  query after it).
- **A rendering test** (`internal/render/windowlist_test.go`): what the
  frame records of the window's rows against the rows drawn.
- **Each change broken on purpose** through `go test -overlay`
  (`scratchpad\plan51\mutrun`, the list built from the source lines by
  `scratchpad\plan61\mkmutants.mjs`), the repository never written.
- **Windows Terminal**: the real application in the acc tab at 208x59 and
  120x40, a mock dashboard over plan 60's demo repository, a temporary home
  and app data, no CLI session, real clicks (`scratchpad\term\p61run.ps1`
  on `p51.ps1`).
- **Build, vet, gofmt, the whole suite** after the last code and document
  edit.

## The evaluation

```text
Evaluation - Plan 61, clicks in the file viewer and the quick open - 7 criteria: 6 PASS, 0 FAIL
PASS 61.1 The viewer's click: TestFileClickLine at 120x40, 80x24 and full screen: a click on the code of the middle line in view, the blank after the last one, the number of the first and the row's edge inside the border puts the cursor there ("line N/100"), the view from the same line and the sideways shift kept, nothing selected; a key moving the view after the frame (PgDn, not drawn): the click on the line drawn, the view back on the rows drawn; with the blame on, a click on its column of lines 22 and 5: "9a1b2c3 · Bo Li · 02/03/2023 14:30 · Rework main.go (read 1)" and "f52c343 · Ana Hubaut · 15/11/2022 10:00 · Add main.go" in the footer
PASS 61.2 Not a line: TestFileClickNotALine: the path row, the footer, the four edges of the frame, a row under a three-line file's last line: the cursor where it was (line 5, line 3); in an empty file, a binary one and while a file is read, the cursor at 0; rows drawn for main.go clicked after short.txt replaced it, not yet read: no line taken
PASS 61.3 The selection: TestFileClickSelection: a drag from line 3 copies "// line 3\n// line 4\n// line 5", the cursor on line 3; a double click on line 7 copies "line" and a triple click on line 9 "// line 9", the cursor there; Shift+click on line 6 after a click on line 2 copies lines 2 to 6, the cursor on line 6; a right and a middle click on line 8 move nothing; TestFileClickOutside (plan 58) passing: a click outside closes, a drag released outside copies
PASS 61.4 A field typed: TestFileClickField: with "/" and "line 9" typed, a click on line 4 puts the cursor there, the field open with "line 9", and Enter searches ("line 9" found); with "g" and "1" typed, a click on line 6, the field open with "1", and Enter goes to line 1
PASS 61.5 The quick open: TestFinderClick on the rig's files and 60 more: a click on the third selects it ("3 of" the files listed), a second click opens it in the viewer ("quick open"); a double click on the fifth opens it; on a list scrolled by End, the file under the pointer, the rows as drawn; PgDn after the frame: the file drawn under the pointer, the rows drawn; the field row, the footer, the left edge, a row under the one file f00 leaves, "no file matches" and "listing…": nothing; a row drawn before a query narrowed the list: nothing
PASS 61.7 The checks: go build ./... ok; go vet ./... ok (Windows and Linux); gofmt -l . empty; go test ./... ok (16 packages); no golden frame or mock file changed; TestKeyOverlayRows passing, the two rows reworded within 78 columns; 16 mutants through -overlay, each caught by a failing test, none by a build failure, the tree's hash 95515e21 before and after
Not verified: 61.6's last part, the owner's visual review. Its measured part holds: in Windows Terminal at 208x59 and 120x40, real clicks put the cursor on line 27 (Maximilian Oberhausen's commit in the footer), line 18 (Bo Li's), line 11 ("Not committed yet") and, on the blame's column, line 15; in the quick open a click selected notes.txt ("2 of 3") and a second opened it ("notes.txt · quick open", "not in git yet: no history"); review shots out-p61\click-max\e6.png, click-90x30\e9.png, click-90x30\e13.png, click-90x30\e14.png
Verdict: NOT DONE
Proposed next step: the owner's visual review of the clicks (the shots) - waiting for approval
```

## Details

### What was built

The frame records, beside the window's rectangle (plan 58), what the
window's rows showed: `Screen.WindowList`, a `ListShown` with the first
line or file, the row it is on (under the path or field row), how many
rows follow, and the columns between the side borders. `fileWindow` and
`finderWindow` give the first and the count as they draw the rows, so the
record cannot drift from the drawing. `ListShown.At` maps a cell to its
line or file, -1 off the list.

In `fileMouse`, a left click inside the viewer puts the cursor on the
line under it, `Top` set to the first line drawn so that the view stays
on the rows the owner clicked, then `press` starts the selection as it
did. With the quick open showing, the click goes to `finderClick`: a file
other than the selected one is selected, the selected one opened as
Enter opens it. A click is judged by the frame drawn last, as plan 58's
click outside is, and the line or file must still exist in the model:
the tests click on frames left behind by a key, by another file being
opened and by a narrowing query.

### The mutants

Each through `go test -overlay` (`scratchpad\plan51\mutrun`, the list
built from the source lines by `scratchpad\plan61\mkmutants.mjs`), the
repository never written: `git diff` and `git status` hashed 95515e21
before and after.

| Id | The change | Caught by |
| - | - | - |
| R1 | the left border a line's cell | TestWindowList |
| R2 | the right border a line's cell | TestWindowList |
| R3 | the row under the list a line's | TestWindowList |
| R4 | the path or field row a line's | TestWindowList |
| R5 | the list recorded from the path row | TestWindowList |
| R6 | the side borders inside the list | TestWindowList |
| R7 | the viewer's first line recorded as the file's | TestWindowList |
| R8 | one file more than drawn | TestWindowList |
| A1 | a click on a line moves no cursor | TestFileClickLine |
| A2 | the view not anchored on the rows drawn | TestFileClickLine |
| A3 | a line past the file shown taken | TestFileClickNotALine |
| A4 | the selection not started any more | TestBlameUnchanged |
| A5 | the quick open's clicks sent to the viewer | TestFinderClick |
| A6 | the selected file never opened by a click | TestFinderClick |
| A7 | the quick open's rows not anchored on the rows drawn | TestFinderClick |
| A8 | a file past the list taken | TestFinderClick |

### The Windows Terminal run

`scratchpad\term\p61run.ps1` on `p51.ps1`: the real application in the acc
tab maximised (208x59) and at `-mode 90x30` (120x40 at the acc profile's
9 pt), with a temporary home, local and roaming app data and a PATH of
System32 and Git; a mock dashboard, so no CLI was started and no quota
used. The project was plan 60's demo repository (`p60demo.ps1`): viewer.go
by three authors, line 11 changed and three lines added in the working
tree, notes.txt untracked.

Each step's shot and text are in `scratchpad\term\out-p61\click-max` and
`click-90x30`: e3 viewer.go opened by two clicks in the Explorer; e5 `b`,
the blame, the cursor on line 1; e6 a click on line 27's code, the cursor
there, "9cd6692 · Maximilian Oberhausen" in the footer; e7 line 18, Bo
Li's commit; e8 line 11, "Not committed yet"; e9 a click on "Bo Li" in the
blame's column of line 15, the cursor on that line; e12 Esc then `/`, the
quick open; e13 a click on notes.txt, selected, "2 of 3"; e14 a second
click, notes.txt in the viewer, "not in git yet: no history". The harness
types `/` as a character: on the owner's AZERTY layout it is a shifted
key. The owner's own dashboard, started with `go run` in a terminal with
its Codex session, stayed open: the run started no CLI and read nothing of
theirs. No dashboard or git process of the run was left.

**61.6, the owner's visual review (2026-10-28).** The owner reviewed the
review shots (a click on line 27 at 208x59 with its commit in the footer,
a click on line 15's blame column at 120x40, the quick open after a click
on notes.txt, and notes.txt opened by a second click) and wrote "Looks
right": 61.6's last part, PASS, so 7 criteria, 7 PASS, 0 FAIL. The clicks
are kept as the shots show them. Not committed: the commit waits for the
owner's approval. The block above stays as it was reported.

**Approved and committed (2026-10-28).** The owner wrote "Approve and
commit". Commit b452ade (13 files: 10 modified, 3 added), from the files
reviewed in `git diff`, which `git diff --check` and a byte check found
free of trailing spaces, CRs and BOMs; the whole suite had passed after
the last code edit and after the last edit of a document a test reads. The
author is `Evengelius <molimoabel@gmail.com>`, with no attribution line,
and the working tree was clean after it: PASS. Not pushed: the owner asked
for the commit only. The block above stays as it was reported.

**The push (2026-10-28), at the owner's request.** Commit b452ade and the
approval commit, f99b55d, were pushed to `origin`: `main` fast-forwarded
from 3e9ec79 to f99b55d, nothing forced; `origin/main` equals `main`
(read back by `git ls-remote`), the working tree clean: PASS. The commit
recording this was pushed after it.
