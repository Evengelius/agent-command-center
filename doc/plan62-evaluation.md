# Plan 62 evaluation (2026-10-28)

The check of plan 62, the Git Graph (brief amendment 64): the sidebar's
third tile shows the repository's history with its lanes, after the owner's
images of an IDE's git graph and history scope picker: the history in
git's topological order, read a page at a time; a scope of references;
a commit unfolded into its files and a file's diff in that commit; the
cursor's row summed up in the footer and a card beside a row. It reads git
and writes nothing.

It covers the 12 criteria of [`plan.md`, Plan 62](../plan.md), written
before the code, with the two reviews the owner folded in. Nothing is
committed or pushed: the owner reviews first.

How it was checked:
- **Input tests on real repositories** (`internal/input/graph_test.go`, 18
  tests; `graphrec_test.go`): histories made with `git fast-import` and
  fixed dates, 50,000 commits among them; a blobless partial clone; every
  git call traced.
- **Rendering tests** (`internal/render/graph_test.go`,
  `graphlanes_test.go` and `graphgit_test.go`, 14 tests, and the
  sidebar's): the panel, the scope popup, the lanes on 40 random histories
  and seven recordings of `git log --graph`, the footer, the card; **golden frames** `graph`, `graph-scope`,
  `graph-diff`, `graph-wide`, `graph-lanes`, `graph-lanes-wide` and
  `graph-card` at 80x24, 120x40, 200x60, 208x59 and 79x24.
- **Application tests** (`internal/app/graph_test.go`, 18 tests) with a
  test double of git that can hold a read, repeat a commit or move a
  reference between two pages.
- **Each change broken on purpose** through `go test -overlay`
  (`scratchpad\plan51\mutrun`; the lists built from the source lines by
  `scratchpad\plan62\mkmutants.mjs`), the repository never written.
- **Windows Terminal**: the real application in the acc tab at 208x59,
  120x40 and 80x24, a mock dashboard over a made-up 321-commit repository,
  a temporary home and app data, no CLI session (`scratchpad\term\
  p62run.ps1`, `p62lanes.ps1` and `p62card.ps1` on `p51.ps1`, with
  `p62demo.mjs`).
- **The race detector** on the three packages the plan changed, **the
  timing step** alone, and **build, vet, gofmt, the whole suite** after
  the last code and document edit.

## The evaluation

```text
Evaluation - Plan 62, the Git Graph - 12 criteria: 9 PASS, 2 FAIL
PASS 62.1 The tile: TestSidebarTiles: three tiles, tops 3, 7 and 11, the Git Graph's in its four looks; TestSideImages: its icon, two lines of history from a fork to a merge, the four commits wider than a link, the ground between and beside them, in each look; TestSideSixel: its four images; TestSideTileCells: its tooltip " Git Graph " at 120x40, 156x44, 200x60 and 208x59; TestGraphShow: Alt+G and a click on the tile show and hide the panel with the focus, the three panels sharing the docked place, each kept as it was, under 100 columns in zone B's place; 87 golden frames changed by the one cell of the tile's "◉"; in Windows Terminal two clicks on the tile hid and showed the panel at 208x59 and 120x40, Alt+G at 80x24
PASS 62.2 The history: TestGraphLogOrder, TestGraphLogDone, TestGraphLogBetweenPages on real repositories: git log --topo-order's order, pages joining with none missing or twice, a commit made after the first page changing nothing of the read; TestGraphRefs: a detached HEAD, an upstream, a branch with no commit yet; TestGraphFollows: Auto with no upstream ("main, no upstream") and detached ("detached at …"), read from HEAD alone; TestGraphHistory: pages of 200 joining, the reader closed at the history's end, when the panel hides and when the dashboard leaves, a history cut short read again when shown, "no commits yet"; TestGraphTipsMoved: commits made on main between two pages, the history read again from the new tips, the cursor kept; TestGraphWatch: a commit made outside seen at the next 2 s tick, at once after the branches popup's fetch, nothing read while hidden; TestGraphPlaceKept: three commits made while hidden seen when shown; TestGraphStalePage: a page of a replaced read dropped, its reader closed; TestGraphLogClose: git gone after Close (its exit status, no goroutine, the folder free); in Windows Terminal Auto's history of the demo repository, End reading the next page
PASS 62.3 The scope: TestGraphScope: Auto at first; All; one reference; two; All and Auto exclusive, the last box cleared giving Auto; the search; Enter applying, Esc and a click outside keeping the scope; the scope row's name; a reference gone; never a switch or a fetch; TestScopePopup, TestScopeRows: the popup as drawn; TestGraphCopyAndScroll: 61 branches scrolled to the end; TestGraphScopeRevList: Auto (13), All (29), one branch (3) and two (5), each the commits git rev-list --topo-order gives; TestGraphShow: the scope held after the tile is switched; every git call traced in the input tests (graphViolation): the five reads only; in Windows Terminal the search "theme", two references ticked and applied ("2 references"), All by a click
PASS 62.4 A commit's files: TestGraphCommit on a real repository: A, M, D, R and T, a rename's two paths, the counts, a binary file, a merge against its first parent, the first commit against nothing; TestCommitFilesMany: 1,500 files listed, all counted past a lowered guard; TestGraphPanel: the letters in Source Control's colours, "+12 −3", "binary", "first 6 of 7 files"; TestGraphFiles: → unfolds, read once, ← folds; TestGraphNoFiles: "no file changed", "no change against its first parent"; TestGraphCopyAndScroll: 1,500 files unfolded, the last scrolled to; in Windows Terminal a commit's and a merge's files
PASS 62.5 The historical diff: TestGraphCommit (checkCommitSides): an added, a modified, a deleted, a renamed and a binary file, the first commit, a merge against its first parent, the sides named by their short hashes; checkReadsWriteNothing: the working tree, the index and the references the same before and after; TestGraphDiff: the path row "Commit …", Source Control's diff kept and shown again with Source Control, nothing added to zone B, no turn started; TestGraphNoLazyFetch: a partial clone's missing blob not fetched; in Windows Terminal README.md's diff in commit 0b5b398 and src/search.ts's in "Merge 703db09, against its first parent 15aa17f"
PASS 62.6 The lanes: TestLanesFigures: a branch merged back, two branches ending on one commit, three parents, a merge that is also two branches' parent, a new branch kept from under a root's dot, a lane passed; TestLanesTopology: 40 random histories of 300 commits, each commit's lines followed as drawn reaching its parents and no other, every lane shown, 3 and 2, "◆" exactly for a commit in a hidden lane; TestLanesAgainstGit: seven recordings of git log --graph --topo-order (one line, a merge, three parents, criss-cross, two roots, 13 lanes, 300 commits with 41 lanes), the same; TestLanesCrossing; TestLanesColours; TestLanesWalk and TestLanesPaged: the rows the same drawn from any kept row and as pages come; TestGraphPanelLanes and TestGraphLanesRoom: the overflow column, six lanes of ten in 31 columns, every lane wide; golden frames graph-lanes and graph-lanes-wide
PASS 62.7 Readable: TestGraphPanel: the subject whole where it fits, cut with "…" where it does not, the labels giving way first; golden frames of the panel, its scope popup, its diff, its lanes, its wide view and its card at 80x24, 120x40, 200x60 and 208x59; in Windows Terminal at 208x59, 120x40 and 80x24, f widening the panel over zone A and giving it back
PASS 62.8 The details: TestGraphFooter: "0000000 · Ana Hubaut · 2 days ago" on a commit, "internal/render/file.go · Modified" and "web/app.ts · Renamed from web/old.ts" on files, the exact date without the frame's time, every relative unit; TestGraphCardDrawn: a commit's card (author, "2 days ago (14/11/2023 22:13)", its message, 19 lines then "…", its files, its labels, its short hash), left of the docked panel beside its row, under its row at 120x40, never over it, a file's card, none out of view or under a popup; TestGraphCard: the pointer's card, its reads once, a motion within the row filtered, off the rows gone, i and i again; TestGraphMouseMode; TestGraphCopyAndScroll: Ctrl+C the full hash, then the file's path, never an interrupt; TestCommitMessage; in Windows Terminal the footer, i, the pointer on a merge and on a file, at 208x59, 120x40 and 80x24
PASS 62.9 Navigation state: TestGraphPlaceKept: hidden and shown again, three commits made meanwhile, the cursor, the first row and the unfolded commit on the same commits; TestGraphScopeKeepsPlace: the scope changed, the cursor's commit, still in the history, kept with its files unfolded, the history's first commit when it is gone; TestGraphShow: the scope held; in Windows Terminal All applied with the cursor's commit kept
FAIL 62.10 Large histories: the first page in 196 to 303 ms over four runs on 50,000 commits with merges (TestGraphLogLarge), all 250 pages in 0.8 to 1.6 s, none missing or twice, a key to its frame with all 50,000 read and their lanes drawn within its target (the timing step); but 209 bytes kept a commit with a 48-byte subject, 21 MB for 100,000 commits, not "about 100 bytes, 10 MB" (TestGraphFrameCost): its entry 91, its row 8, its subject 49, the index by hash and the lanes 61
FAIL 62.12 The checks: go build ./... ok; go vet ./... ok (Windows and Linux); gofmt -l . empty; go test ./... ok (16 packages) after the last code and document edit; the golden frames reviewed: 87 changed by the one cell of the third tile's "◉" (checked by script), 35 new (the panel, its scope popup, its diff, its wide view, its lanes, both wide, its card, at five sizes), no mock file changed; 23 mutants through -overlay, each caught by a failing test in the end (details), the input reads' 18 the implementer's (17 caught, the survivor equivalent) and the lazy fetch's caught; go test -race on internal/app, internal/render and internal/input: no race; but the timing step, run alone, had 17 of its 19 tests within their targets, TestGraphFrameCost p95 1.97 ms with 50,000 commits and their lanes, and TestBannerToFrame (p95 5.53 ms, target 2 ms) and TestHomePictureTime (worst 407.8 ms, target 260 ms) over theirs: the home screen's, which plan 62 does not touch, and as far over on plan 61's tree (HEAD) timed in turn, three runs each (details): the machine's load that afternoon, not this plan's code; to run again on a quiet machine
Not verified: 62.11's last part, the owner's visual review of the three stages and of the tile's icon. Its measured part holds: in Windows Terminal at 208x59, 120x40 and 80x24, the tile, the history and its lanes, the scope popup with a search and a selection, a commit unfolded, a historical diff, the footer and the card, every step as expected; review shots out-p62\graph-90x30\e2.png, e3.png, e10.png, e14.png, graph-max\e2.png, e23.png, icon-90x30\e1-sidebar-x4.png (stage 1), lanes-90x30\e6.png, e7.png, lanes-max\e6.png, e7.png, lanes-60x18\e6.png (stage 2), card-max\e5.png, card-90x30\e3.png, e8.png (stage 3)
Verdict: NOT DONE
Proposed next step: the owner's visual review of the three stages (the shots), the owner's decision on 62.10's memory (accept 209 bytes a commit, or shrink the index by hash and the author's field, about 40 bytes less), and the timing step run again on a quiet machine - waiting for approval
```

## Details

### What the checks found and changed

- **A dot under a root's.** `TestLanesTopology`'s tracer, which follows the
  drawn lines as a reader does, found a new branch taking the lane a root
  had ended on the row just above: two dots one over the other, which reads
  as a child and its parent. A new branch now never takes that lane.
- **A commit that changed nothing.** On the demo repository a merge whose
  tree is its first parent's unfolded into no row at all, as if the
  unfolding had failed. It now says "no file changed", or "no change
  against its first parent" for a merge (`TestGraphNoFiles`), and its card
  the same.
- **A partial clone.** The graph's reads could fetch a blobless clone's
  missing blobs from its remote on demand, which design D14 rules out.
  `GIT_NO_LAZY_FETCH=1` is now on the graph's calls alone, Source
  Control's left as they were; in such a clone a commit's files read fails
  in git's words (`TestGraphNoLazyFetch`, its mutant caught).
- **The card under 100 columns.** Where no sidebar shows, the mouse
  reported motions only with a button held, so the pointer drew no card at
  80x24 in Windows Terminal; every motion is now reported while the Git
  Graph is drawn, as for Source Control (`TestGraphMouseMode`).
- **The tile's icon.** Its first drawing, three commits on a line and a
  branch leaving and joining it, read as a play button at the tile's size;
  it is now two lines of history from a fork to a merge, a commit on each.
- **The scope popup before a frame.** Its keys read the last frame's list
  height without checking a frame was drawn, which the branches popup
  does; they now check.
- **The recordings' trailing spaces.** git's own whitespace check flagged
  the seven recordings of `git log --graph`; they are written without the
  spaces at their lines' ends.
- **Tests the mutants asked for**, below: the boundary of the lanes'
  room (`TestGraphLanesRoom`), a page of a replaced read
  (`TestGraphStalePage`), a commit given twice (`TestGraphDuplicate`).
  And the evidence the criteria name that no test held yet: the 2 s watch
  and the fetch's re-read (`TestGraphWatch`), the scopes against `git
  rev-list` (`TestGraphScopeRevList`), Ctrl+C, the wheel, 1,500 files and a
  long scope list scrolled to (`TestGraphCopyAndScroll`), the cursor kept
  across a scope (`TestGraphScopeKeepsPlace`), no upstream and a detached
  HEAD (`TestGraphFollows`), no turn started by a diff (`TestGraphDiff`).
- **Drawn otherwise than the plan's text**: a rename's source after "←",
  as Source Control writes it (D5 says "old → new"); the labels give way
  whole, the last first, before the subject is cut (D11).

### The mutants

Each through `go test -overlay` (`scratchpad\plan51\mutrun`, the list built
from the source lines by `scratchpad\plan62\mkmutants.mjs`). The input
reads' 18 were the implementer's (17 caught, the survivor equivalent:
reading an added file's old side gives the same missing side), and
`TestGraphNoLazyFetch`'s mutant, the variable dropped, was caught.

| Id | The change | Caught by |
| - | - | - |
| L1 | a merge that ends lanes laid out from the leftmost | TestLanesAgainstGit |
| L2 | its new lanes on either side | TestLanesAgainstGit |
| L3 | a new branch under a root's dot | TestLanesTopology |
| L4 | the root's lane not remembered | TestLanesTopology |
| L5 | a merge's lane looked for at the right only | TestLanesAgainstGit |
| L6 | an ending lane not freed in the walk | TestLanesFigures |
| L7 | a curve drawn over a passing lane | TestGraphPanelLanes |
| L8 | no passing lane drawn | TestGraphPanelLanes |
| P1 | lanes hidden one lane too late | TestGraphLanesRoom (third run) |
| P2 | the labels never all given way | TestGraphPanel |
| C1 | the card over the panel's edge | TestGraphCardDrawn |
| C2 | the card over its row | TestGraphCardDrawn |
| C3 | days counted to 30 | TestGraphFooter |
| C4 | no file in the footer | TestGraphFooter |
| A1 | the lanes kept from the read before | TestGraphTipsMoved |
| A2 | no commit laid out | TestGoldenFrames |
| A3 | a page of a replaced read taken | TestGraphStalePage (second run) |
| A4 | a history cut short not read again | TestGraphHistory |
| A5 | a commit taken twice | TestGraphDuplicate (second run) |
| A6 | a message read at every hover | TestGraphCard |
| A7 | the filter dropping motions onto another row | TestGraphCard |
| A8 | a reference ticked with All | TestGraphScope |
| A9 | no motion reported for the card under 100 columns | TestGraphMouseMode |

In the first run 20 were caught; P1, A3 and A5 survived, the tests named
above then written. The second run caught A3 and A5; P1 survived again,
its test outside the run's pattern (`TestViewLanesRoom`), and was caught in
the third once renamed `TestGraphLanesRoom`. The tracked files' diff hashed
d08d2e99 before and after the second run; during the first, my own edit of
plan.md changed it (a76057fb to d08d2e99); the third was not hashed. Every
new file of the plan is untracked, outside that hash: the overlay runner
never writes the tree.

### The Windows Terminal runs

`scratchpad\term\p62run.ps1`, `p62lanes.ps1` and `p62card.ps1` on
`p51.ps1`: the real application in the acc tab maximised (208x59), at
`-mode 90x30` (120x40 at the acc profile's 9 pt) and at `-mode 60x18`
(80x24), with a temporary home, local and roaming app data and a PATH of
System32 and Git; a mock dashboard, so no CLI was started and no quota
used. The project was a made-up repository outside this one
(`p62demo.mjs`, written with `git fast-import`, git reading none of the
owner's configuration): 321 commits, 300 older ones, then feature/search
and feature/export merged into main, feature/theme open, v1.0 and an
annotated v1.1, origin/main two commits behind main as its upstream,
origin's feature/theme, a rename, a type change to a symbolic link, a
deletion and a binary file, four made-up authors.

- **Stage 1** (`out-p62\graph-max`, `graph-90x30`, `graph-60x18`): e1 Alt+G,
  Auto's history ("main → origin/main" at 120x40 and 80x24); e2 → on the
  newest commit, its README.md and rows.ts; e3 Down and Enter, README.md's
  diff in "Commit 0b5b398"; e6 s, the popup; e8 "theme" searched; e10 both
  references ticked; e11 applied, "2 references", Light theme first; e13
  and e14 a click on All and Enter, the cursor's commit kept first in view;
  e15 End and e16 Home; e17 to e23 two clicks on the feature/search merge
  (feature/export's at 80x24), two on its file, its diff "Merge 703db09,
  against its first parent 15aa17f"; e21 and e22 two clicks on the tile
  (Alt+G twice at 80x24), the panel hidden, then shown as it was.
- **Stage 2** (`lanes-max`, `lanes-90x30`, `lanes-60x18`): e1 Auto's lanes,
  the merges' rings and their curves, the forks; e2 f, the panel from zone
  A's edge, the author and the date in columns; e3 f, back; e6 All's
  lanes, feature/theme's open lane; e7 f on them; e8 End, e9 Home; e10 f.
- **Stage 3** (`card-max`, `card-90x30`, `card-60x18`): e1 the footer,
  "0b5b398 · Bo Li · …"; e2 two Downs, Maximilian Oberhausen's commit in
  the footer; e3 i, its card ("5 months ago (14/05/2026 17:49)"); e4 i, gone;
  e5 the pointer on the feature/export merge, its card ("1 file changed
  against its first parent, +1 −0", v1.1); e6 the pointer off the panel,
  gone; e7 → and e8 the pointer on its file, the file's card ("Modified").
  At 80x24 the panel takes the whole width, so the rows under the pointer
  are other ones and the pointer never leaves the panel: each step still
  showed the card of the row under it.

Every step held what it should; the shots of each stage were sent to the
owner as it ended. The run left no dashboard or git process; it started no
CLI. The owner's terminal-hosted CLI sessions: none were open (the desktop
app's own and VS Code's were left alone).

### The race detector

`go test -race` on the three packages the plan changed, with the portable
WinLibs GCC 16.2.0 first on the run's own PATH and `CGO_ENABLED=1`
(evaluation.md, The race detector): internal/app 770.7 s, internal/render
503.0 s, internal/input 399.3 s, no race reported, every test passing.

### The timing step

Run alone with the exact command of `doc/evaluation.md`, Keystroke timing,
`TestGraphFrameCost` now among its tests: mains power, the battery at 98 %,
the CPU at 0, 4 and 2 % just before; open: Claude, VS Code, Edge, Teams,
Outlook, Windows Terminal, Move Mouse and Settings, and the owner's own
dashboard (`go run`, started at 16:02), idle at 0.05 s of CPU in 3 s. No
other `go` build or test ran.

| Test | Result | Target |
| - | - | - |
| TestGraphFrameCost | p95 1.97 ms (p50 under the clock's tick) | p95 ≤ 16 ms |
| TestBlameFrameCost | p50 0.51 ms, p95 1.50 ms | p95 ≤ 16 ms |
| TestBranchesFrameCost | p50 1.19 ms, p95 3.52 ms | p95 ≤ 16 ms |
| TestKeystrokeToFrame | p95 1.60 ms; the draft window 2.22 ms | p95 ≤ 16 ms |
| TestDraftKeystrokes | p95 3.75 ms | p95 ≤ 16 ms |
| TestLongSession | p95 7.53 ms | p95 ≤ 16 ms |
| TestThinkingFrame | p95 5.65 ms | p95 ≤ 16 ms |
| TestPasteToFrame | max 3.05 ms | ≤ 100 ms |
| TestStreamToFrame | p95 10.25 ms | p95 ≤ 16 ms |
| TestSelectorToFrame | p95 1.51 to 2.11 ms | p95 ≤ 16 ms |
| TestMotionToFrame | p95 1.51 to 3.52 ms | p95 ≤ 16 ms |
| TestDiffToFrame | p95 1.51 to 2.02 ms | p95 ≤ 16 ms |
| TestMessageToFrame | p95 1.67 and 3.05 ms | p95 ≤ 16 ms |
| TestOutputToFrame | p95 1.40 ms | p95 ≤ 16 ms |
| TestToolsToFrame | p95 5.12 ms | p95 ≤ 16 ms |
| TestBannerToFrame | p95 5.53 ms: FAIL | p95 ≤ 2 ms |
| TestHomeKeysToFrame | p95 0.47 ms | p95 ≤ 16 ms |
| TestHomePictureTime | max 407.8 ms: FAIL | ≤ 260 ms |
| TestKeystrokeCost | reports only | none |

Several results were about twice plan 60's (the 2,000-line stream's p95
10.25 ms against 4.85, the long session's 7.53 against 2.95), the sign of a
busier machine. The two failing tests are the home screen's, which plan 62
does not touch. Timed in turn with plan 61's tree, HEAD 895bea5, extracted
with `git archive` into a temporary folder, the working tree untouched:

| Run | This tree: banner p95, picture worst | HEAD: banner p95, picture worst |
| - | - | - |
| 1 | 1.62 ms, 290.0 ms | 2.11 ms, 324.6 ms |
| 2 | 2.89 ms, 329.0 ms | 2.16 ms, 285.2 ms |
| 3 | 1.45 ms, 282.4 ms | 1.49 ms, 288.0 ms |

Both trees over the picture's 260 ms in every run and over the banner's 2 ms
in some: the machine's load that afternoon. Plan 51 saw the banner cross
its target under load too (p95 2.05 ms, the CPU 3 to 25 % busy) and pass in
its other runs (1.27 to 1.76 ms), the picture's worst 200 ms. The targets
stay as they are.

**62.11, the owner's visual review (2026-10-28).** The owner reviewed the
review shots of the three stages and the tile's icon (stage 1: the history
with a commit unfolded, its diff, the scope popup searching "theme" with
two references ticked, All, the docked panel at 208x59 and a merge's diff,
the icon close up; stage 2: All's lanes docked at 120x40 and 208x59, wide,
and at 80x24; stage 3: a merge's card left of the panel, `i`'s card, a
file's card) and wrote "Looks right": 62.11's last part, PASS, so 12
criteria, 10 PASS, 2 FAIL. The Git Graph is kept as the shots show it. Not
committed: the commit waits for the owner's approval, 62.10 and 62.12 for
the owner's decision. The block above stays as it was reported.

**62.12, the timing step again (2026-10-28).** Run alone again with the
exact command, the owner's own dashboard closed by then: mains power, the
battery at 98 %, the CPU 19 to 32 % busy just before by WMI's count (the
busiest processes over 3 s: the Claude desktop app, Move Mouse, Explorer,
Edge, the VPN client's posture scanner at 0.19 s, Teams), the same
applications open. Each of the nineteen within its target:

| Test | Result | Target |
| - | - | - |
| TestGraphFrameCost | p95 1.00 ms | p95 ≤ 16 ms |
| TestBlameFrameCost | p95 1.34 ms | p95 ≤ 16 ms |
| TestBranchesFrameCost | p50 1.00 ms, p95 2.85 ms | p95 ≤ 16 ms |
| TestKeystrokeToFrame | p95 1.06 ms; the draft window 1.12 ms | p95 ≤ 16 ms |
| TestDraftKeystrokes | p95 3.39 ms | p95 ≤ 16 ms |
| TestLongSession | p95 3.54 ms | p95 ≤ 16 ms |
| TestThinkingFrame | p95 3.18 ms | p95 ≤ 16 ms |
| TestPasteToFrame | max 1.67 ms | ≤ 100 ms |
| TestStreamToFrame | p95 5.08 ms | p95 ≤ 16 ms |
| TestSelectorToFrame | p95 1.03 to 1.52 ms | p95 ≤ 16 ms |
| TestMotionToFrame | p95 1.02 to 2.00 ms | p95 ≤ 16 ms |
| TestDiffToFrame | p95 1.02 to 1.32 ms | p95 ≤ 16 ms |
| TestMessageToFrame | p95 1.50 and 2.10 ms | p95 ≤ 16 ms |
| TestOutputToFrame | p95 1.02 ms | p95 ≤ 16 ms |
| TestToolsToFrame | p95 2.88 ms | p95 ≤ 16 ms |
| TestBannerToFrame | p95 1.80 ms | p95 ≤ 2 ms |
| TestHomeKeysToFrame | p95 0.19 ms | p95 ≤ 16 ms |
| TestHomePictureTime | max 186.7 ms | ≤ 260 ms |
| TestKeystrokeCost | reports only | none |

The first run's two failures were the load of that moment, as their
alternation with plan 61's tree showed; most likely the owner's own
dashboard, whose home screen animates its banner in Windows Terminal.
62.12's last part, PASS, so 12 criteria, 11 PASS, 1 FAIL: 62.10, for the
owner's decision. The block above stays as it was reported.

**62.10, accepted as a documented exception (2026-10-28).** Told: 209
bytes kept a commit with a 48-byte subject, about 21 MB for 100,000
commits, about 2 MB for 10,000, at most about 210 MB at the guard's
1,000,000; shrinking the index by hash and the author's field would save
about 40 bytes, and nearer 100 would mean the subjects out of memory. The
owner wrote "Accept it": 62.10 stays FAIL, accepted as a documented
exception, the memory as measured. 12 criteria, 11 PASS, 1 FAIL accepted.
Not committed: the commit waits for the owner's approval. The block above
stays as it was reported.
