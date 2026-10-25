# Plan 44 evaluation (2026-10-25)

The check of plan 44, Source Control and a Git diff viewer (brief amendment
45: the sidebar's second icon is Source Control, not Skills; its panel, in
the Explorer's docked place, lists the project's Git changes, staged and
unstaged apart, each with its real status; a file opened shows its diff in
the conversation's place, read only, after the owner's image
`references/14-source-control.png` and example layout). It covers the 14
criteria of [`plan.md`, Plan 44](../plan.md), written before the code.
Nothing is committed: the owner reviews the screenshots and the results
first.

How it was checked:
- The finding from the code, git's documentation of `status --porcelain=v1
  -z --branch` and `cat-file --batch`, and probe P44a on a temporary
  repository (plan.md, What was found).
- Deterministic tests, none starting a CLI, none touching this repository:
  - the engine, on temporary repositories made by real git with an empty
    global configuration (`internal/input`): TestParseSnapshot,
    TestConflictName, TestSourceStates (20 cases), TestSourceConflicts (a
    two-branch merge giving the seven conflicts), TestSourceBranch,
    TestSnapshotUntrackedProject, TestSourceNoRepo, TestCatFileNames,
    TestCatFileSubmodule, TestWorkingSide, TestUntrackedFiles,
    TestSourceJunction; TestParseStatus unchanged;
  - the comparison (`internal/diff`): TestCompareHunks, TestCompareEnds,
    TestCompareUTF8, TestCompareRandomScripts, TestCompareMinimal,
    TestCompareMinimalSkewed, TestCompareMinimalLarger, TestCompareBound,
    TestCompareBoundSearch, TestCompareSpeed, TestCompareMessy;
  - the drawing (`internal/render`): TestSourceRows, TestSourceCursor,
    TestSourceRowLooks, TestSourcePanel, TestLayoutDiff, TestDiffLines,
    TestDiffView;
  - the application, keys and mouse messages through the program's update
    and its filter, git's answers from a test double (`internal/app`):
    TestSourceTile, TestSourceList, TestSourceRefresh, TestSourcePoll,
    TestDiffOpen, TestDiffKeys, TestDiffRefresh, TestDiffNotes,
    TestDiffCompact, TestDiffApproval, TestSourceReadOnly, TestSourceHover,
    and TestDiffToFrame in the timing step.
- 16 golden frames added (`source`, `source-none`, `source-diff`,
  `source-diff-wide`, each at 80x24, 120x40, 200x60 and 79x24); none of the
  existing 192 changed.
- The engine (`internal/input/source.go`, the snapshot in `explorer.go` and
  `git.go`, `internal/diff`) built by an implementer subagent to the API
  the plan's step 2 fixed; its report checked against its code and tests,
  and six of its points acted on (below).
- Each new behaviour broken on purpose through `go test -overlay`, nothing
  written in the repository (`scratchpad\plan44\mutate44.mjs`,
  `mutate44b.mjs`, `mutate44c.mjs`); the engine's own 30 mutants run by the
  subagent.
- Build, vet, gofmt, the whole suite, the race detector on render, app,
  input and diff, the timing step alone, all on the final tree.
- Windows Terminal in mock mode, real keys and clicks, on a temporary
  repository outside the project (`scratchpad\plan44\mkrepo.mjs`).

```text
Evaluation - Plan 44, Source Control and a Git diff viewer - 14 criteria: 14 PASS
PASS 44.1 TestSourceTile at 200x60 with Sixel: a click on the second tile shows Source Control docked (focus R, the tile selected and hovered, zone B 106, git's status asked once) and a second hides it (focus B, zone B 136, the tile hovered); F3 with Source Control shown shows the Explorer in its place, F3 again hides it (plan 43's F3); switching by the tiles keeps each panel's state (the Explorer's folders and cursor, Source Control's cursor); the tooltip " Source Control " or " Explorer " left of the sidebar on the hovered tile's middle row, gone off it, none after a click until the pointer leaves; TestSideTileCells: the hover changes the tooltip's cells only, the tiles' 18 cells the same in every look; the four looks' tests (TestSidebarTiles, TestSideImages, TestSideSixel) unchanged but for their comments; the tile is Source Control in the code (SideSource), the F1 page and the documents, Skills kept only as history
PASS 44.2 TestParseSnapshot (6 subtests): M, A, D, R with its source, C, T, both sides, intent to add, the seven conflicts, untracked file and folder, ignored never an entry, spaces, Unicode, nested paths, recorded git 2.56.0 output, the branch forms (a branch, its upstream ahead or behind or gone, HEAD (no branch), No commits yet, git's older Initial commit), a project in a folder of the repository, ignored files inside an untracked folder; TestParseStatus unchanged
PASS 44.3 On temporary repositories made by real git: TestSourceStates (20 cases, each of 44.2 made with git commands), TestSourceConflicts (a two-branch merge giving UU AA DD AU UA DU UD), TestSourceBranch (no commits, an upstream ahead, detached with its short id), TestSnapshotUntrackedProject (a project folder untracked inside a repository told as such), TestSourceNoRepo; Status equals Snapshot's markers
PASS 44.4 TestSourceRows, TestSourceCursor, TestSourceRowLooks, TestSourcePanel: the header with its ⌄, the rule, "<repository> · <branch>", MERGE CHANGES, CHANGES and STAGED CHANGES in that order with their counts, blank rows around the headers as in the owner's example, files three cells in, the letters' colours (M amber; A, U, R, C green; D and ! red; T blue), the directory grey, a rename's source on its staged row only, an untracked folder opened on its files, a cut list said, the cursor's band and the hover's ground across the row, a note whole at 31 columns; TestSourceList and TestSourceHover through the application; golden frames source and source-none at 80x24, 120x40, 200x60 and 79x24
PASS 44.5 TestSourceRefresh: a read when shown and on r; an equal read leaves the frame byte for byte; a new change appears, a revert goes, the cursor kept, a deletion staged taking the cursor to its STAGED row; a tool's end and a turn's end each ask a read; three r during a read make one more read, never two at once; none with no panel or diff; TestSourcePoll: every 3 s, a read of 1 s spacing them 10 s, of 5 s 30 s at most, a stale timer and one with nothing shown reading nothing; TestDiffRefresh: an equal read lays nothing out again, a changed working file is compared again, the scroll kept
PASS 44.6 TestSourceStates on real git, each with its exact sides: unstaged (index → working file), staged (HEAD → index), a file with both (two different pairs), a staged add and AM, untracked and an untracked folder's file, deleted unstaged and staged, renamed with changes (HEAD:old → index:new), a copy, a type change, binary, an LFS pointer, line endings only (EOLOnly), no newline at the end, too large, Unicode names, a credential file refused before any git call; TestSourceConflicts: ours (stage 2, else HEAD) → working file, named, never resolved; TestDiffNotes through the application: binary with both sizes, LFS, too large, line endings only, no textual change, git's error, the viewer's refusal said as the viewer says it, a deleted file to nothing, a missing line end said where its last line shows and only there
PASS 44.7 TestCompareHunks (19 exact cases), TestCompareEnds (14), TestCompareUTF8, TestCompareRandomScripts (3,000 scripts replayed to the new side, minimal against an LCS brute force), TestCompareMinimal (40,000 pairs), TestCompareMinimalSkewed (20,000), TestCompareMinimalLarger, TestCompareMessy (6,000), three lines of context and their merging; TestCompareBound: a middle over 100,000 lines or a difference over 20,000 edits bounded and said (Whole), at the limits not, a 100,001-line file with one changed line compared exactly; TestCompareSpeed: 100,000 lines with 50 edits 38 ms, wholly different 64 ms (target 1 s); a one-off check against git diff --no-index --minimal on 400 random pairs gave the same hunks (the subagent's)
PASS 44.8 TestLayoutDiff, TestDiffLines, TestDiffView: unified rows with both numbers in grey, "−" and "+", the removed row light red on #3A161A and the added light green on #12301F the whole row, context muted, "@@ -a,b +c,d @@" in lavender; side by side from 150 columns of text, a removed run facing the added run, blank where a side has none, ORIGINAL and MODIFIED with what each side is; long lines cut with "…", the shift up to the longest line's end, tabs to 4; each note in the lines' place; the box's columns at 200x60, 120x40, 110x40 and 80x24, narrow and wide; golden frames source-diff and source-diff-wide at the four sizes
PASS 44.9 TestDiffKeys at 200x24: ↓ PgDn ↑ End Home, n n N N under each change's hunk header, the wheel, → to the shift's limit and ←, f wide (162 columns, side by side) and back, r compares again, Tab the other comparison of a file with both, else the next zone; TestDiffOpen: Enter or a click on a file shows its diff in zone B's place, zone A where it was, the panel beside it; F6 to the panel, where a paste goes nowhere; Esc gives back the frame byte for byte, the draft and the conversation's scroll, the focus on the panel's file; TestDiffCompact: at 120x40 the diff 50 columns with the panel 26 beside it, F6 reaching both, F3 putting the Explorer there with the diff kept; at 110x40 the diff in the panel's place, Esc back to it, F3 showing the Explorer and closing the diff; at 80x24 the whole width; TestDiffApproval: the waiting approval first in the footer, F2 from the diff or the panel closing it and focusing the card
PASS 44.10 The engine's read-only test: the traced calls are rev-parse, status and cat-file only, each with -c core.fsmonitor=false and core.quotepath=off and no --force, -f, --textconv, --filters or --no-index, GIT_OPTIONAL_LOCKS=0; a full Snapshot and Sides run over every state leaves every file of the repository, .git included, the same by size, time and SHA-256, with a racy index set as a trap; TestSourceReadOnly: showing, refreshing, opening, Tab, n, f, r and Esc ask the double for status and sides only; the Windows Terminal runs left the temporary repository the same, every file, .git included, by SHA-256, size and time
PASS 44.11 TestSourceList: no repository says "No Git repository found for this project." whole and runs nothing else; git missing "git not found"; git failing its error in the attention colour; a project folder untracked in a larger repository said so; TestSourceNoRepo: one rev-parse, then NotRepoError; golden frame source-none
PASS 44.12 0 of the 192 existing golden frames changed, 16 added; the Explorer's tests (plan 17's and plan 43's), F3's, the draft's and the mouse's pass; adapted: TestExplorerToggle's pattern of the selected tile (the tooltip covers the panel's edge cell beside it), TestSideHover, TestSideDraw and TestSideTileCells (a hover may change the tooltip's cells and no other), TestZoneAKeyWords (the F6 row's words), TestKeyOverlayRows (55 rows to 67, the two new tables); TestSkillsTile removed with the placeholder it checked, TestSourceTile in its place
PASS 44.13 Build, vet, gofmt clean; go test ./... ok, 16 packages, 762 tests; -race on render, app, input and diff ok (86 s, 291 s, 58 s, 8 s), 0 data races; the timing step alone, ten tests: TestDiffToFrame, the 20,000-line diff scrolled by ↓ ↑ PgDn PgUp n N and the wheel, p95 1.02 to 1.78 ms at 120x40 and 208x59, unified and side by side; the other nine within their targets as before (p95 at most 5.0 ms, the 50,000-line paste 2.7 ms at worst; 800 of 800 idle motion reports dropped in each case); mutations through go test -overlay: 58 of 58 caught (45; then 10, the two of the first list whose mutant did not build made again; then 5 for the fixes the terminal runs called for), and the subagent's 30 on the engine
PASS 44.14 Windows Terminal, the final build (`scratchpad\term\acc-p44.exe`, SHA-256 c9caddc00baf14e7…), maximised in the acc tab's profile (208x59) and 120x40 in the default profile, mock mode on a temporary repository outside the project (M, D, MM, R, an untracked file and folder; `scratchpad\plan44\mkrepo.mjs`), real clicks and keys: A, the panel listing them with their letters, the tooltip of a hovered tile, a hovered row; B, a click on internal/app/app.go showing its diff with the panel beside it, unified with both numbers, then f, side by side over zone A; C, a click on internal/state/reduce.go, its working tree diff, then Tab, its staged diff, the panel's cursor following; D, the Explorer and Source Control from their tiles, zone C full width; E, the untracked folder opened on its files; at 120x40 the diff 50 columns with the panel beside it; the colours read back from the screenshots (`scratchpad\term\px44.ps1`): added rows #12301F with #B4F1C6 text, removed #3A161A with #FFC4C4, the hunk header #C4B5FD, the numbers #6B7280, context #A1A9C0, +8 #4ADE80 and −3 #F87171, the cursor band #342B47, the hovered row #221D2E, the tooltip #342B47 with #E5E7EB, M #FBBF24, D #F87171, U and R #4ADE80, the header #C4B5FD, the repository's name #A0ABD8, the tiles within 1 level of plan 43's values; no terminal-hosted agent session was open; the clipboard unchanged
```

Changed from the design, after the code or the engine's report:
- **The panel stays beside the diff where it would take zone B's place.**
  Design D8 put the diff in the panel's place there (100 to 135 columns).
  The owner's text asks that "The Source Control sidebar stays visible": the
  diff now takes that place with the panel docked beside it at its own width
  while the diff keeps 48 columns, from 118 columns (at 120x40 the diff 50,
  the panel 26); narrower, and under 100 columns, the diff takes the whole
  place and Esc brings the panel back.
- **The owner's example layout.** Blank rows around each section's header
  and files three cells in, as the amendment's example draws them, where
  the first frames had none and two cells.
- **The engine's report** (the subagent's points, each checked in its code):
  the bound now applies to the changed middle between the common head and
  tail, not the whole file, so a 100,001-line file with one changed line is
  compared exactly rather than flagged "too large to align"
  (TestCompareBound adapted); a project folder untracked in a larger
  repository, which git names once by the project's prefix, is said so
  (`GitSnapshot.Untracked`) rather than "No changes"; an untracked folder's
  files that git ignores are left out (`GitMarks.IsIgnored`); the viewer's
  refusals ("a credential file: not opened") are said as the viewer says
  them, not as git's errors; a side's missing line end is said where its
  last line shows; control characters in a line show as the editor shows
  them ("^M").
- **F2 beside the diff** closes it: the approval's card is in zone B, which
  the diff covers. "f wide" is left out of the footer under 100 columns,
  where there is no zone A to widen over.
- **The tooltip** goes on a click on its tile until the pointer leaves the
  tile: left after a click, it covered the panel's first rows.
- **What the first Windows Terminal runs showed**, fixed and the runs made
  again: at 208 columns the panel is 31 wide, and "internal/state/reduce.go"
  was cut before its name, so the run's click on "reduce.go" found nothing;
  the directory now gives way first, its leading folders as "…/", and a
  rename's source is cut before its name ("…/util/text.go ← int…"). The side
  by side heads ended with a stray "…" at odd widths. Tab now puts the
  panel's cursor on the row of the comparison shown, and the footer leaves
  out Tab for a file with one comparison.

Observed and kept:
- **Side by side needs 150 columns of text.** At 208x59, zone B's 111
  columns show the diff unified; "f" widens it over zone A to 169 columns,
  side by side (screenshots B and B2). Each side would otherwise keep about
  45 columns of code, the "excessive truncation" the owner asks to avoid.
- **Source Control has no key.** The owner's text keeps F3 for the
  Explorer and names no key for Source Control: it is shown by its tile,
  from 100 columns; under 100, where the sidebar is not drawn, it is shown
  only when shown wider first.
- **Nothing is selected in the panel or the diff**, as in the Explorer's
  panel since plan 43: F12 and the terminal's own selection copy from them.
- **Git's indent heuristic is not done.** A diff of repeated lines is
  minimal, checked against a brute force and against `git diff --minimal`,
  but may place a blank line differently from git's default output.
- **The input package's tests take about 35 s more** (61 s), the repositories
  of 44.3 and 44.6 being made by real git.
- The refresh reads git's status every 3 s while a panel or a diff shows,
  and a file's sides at each read while its diff shows, comparing them again
  only when a blob's id or the working file's size or time changed; none
  while neither shows.

Not verified:
- A live session: the refresh after a tool call or a turn was checked on
  the scripted session's events, not on a model's.
- Git LFS with its filter installed, on screen: the pointer case is checked
  by the engine's tests; P44a showed `git status` running the filter.
- A terminal without Sixel, on screen: the text form is the golden frames'.

**The commit and the push (2026-10-25), after the owner's approval.** The
owner approved plan 44 on 2026-10-25; commit d9d7d83 (61 files: 29
modified, 32 added) and the approval commit were pushed to `origin`;
`origin/main` equals `main`, the working tree is clean, and the author is
`Evengelius <molimoabel@gmail.com>`, with no attribution line: PASS. The
blocks above stay as they were reported.
