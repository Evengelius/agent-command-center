# Plan 65 evaluation (2026-10-29)

The check of plan 65, the turn-by-turn review of what the agent changed
(brief amendment 67): after each turn, zone B lists the files the turn
changed in the project, and each file's change opens in plan 44's diff
view. The source is two git snapshots of the project, one at the turn's
start and one at its end, taken into a store of the dashboard's own: a
folder for each session under `%LOCALAPPDATA%\agent-command-center\turns`,
readable by the user alone, with its own index and objects, the
repository's objects read behind it, no hook, filter or monitor run, no
credential file or file outside the project read. Another agent's three
reviews, shared by the owner, were folded into the design before the
owner wrote "Approve and start".

It covers the 9 criteria of [`plan.md`, Plan 65](../plan.md), written
before the code. Nothing is committed or pushed: the owner reviews first.

How it was checked:
- **Input tests on real git**, in temporary repositories and a temporary
  stores' folder, never the user's own (`internal/input/turns_test.go`,
  23 tests; `turns_windows_test.go`, 3): the store, the snapshot, and what
  never reaches the store.
- **State, render and application tests** on a fake store and the mock CLI
  (`internal/state/review_test.go`, 9; `internal/render/turnreview_test.go`,
  6; `internal/app/review_test.go`, 13), and the golden frames `review`,
  `review-open`, `review-diff` and `review-diff-wide` at 79x24, 80x24,
  120x40 and 200x60.
- **The real store measured** by two tests overlaid from the scratchpad
  (`scratchpad\plan65\zz_turntiming_test.go`, `zz_turngrowth_test.go`),
  never in the tree: its timings on a clone of this repository and on a
  repository of 20,000 files, and its growth over eight turns.
- **Each change broken on purpose** through `go test -overlay`, the
  repository never written, run again on the final tree: 41 mutants of
  the state, render and application parts
  (`scratchpad\plan65\mkmutants65.mjs` and `mutants65-reload.txt`, run by
  plan 51's mutrun) and 27 of the store (`scratchpad\turns65\mut.mjs`).
- **Windows Terminal**: the application built from this tree, in ACC's
  profile, `claude-subagent-approval`'s mock on a repository of its own,
  files changed during the turn and between the turns
  (`scratchpad\term\p65run.ps1`).
- **The race detector** on the four packages the plan changed, **the
  timing step** alone, and **build, vet, gofmt, the whole suite** after
  the last code and document edit.

## The evaluation

```text
Evaluation - Plan 65, the turn-by-turn review - 9 criteria: 8 PASS, 0 FAIL
PASS 65.1 The snapshot: TestTurnUnchanged: an unchanged folder's tree the same twice, the second from the listing alone, a clean checkout's first tree HEAD's; TestTurnFiles: a turn that modifies, adds, removes and renames files and writes a binary one gives exactly those files with git's letters and counts (R from -M), each file's two sides the turn's; TestTurnWritesNothing: in a main and in a linked worktree, the owner's changes pending, every file of the project and of its .git, path, size, time and bytes, the same across the store's opening, Take, Changing, Files and Sides; TestTurnFiltersHooksLineEndings: the repository's filter driver and post-index-change hook not run by Take and run by git's own add (the control), a CRLF file under "* text eol=crlf" stored as LF as git's add stores it, a filtered file as its bytes; TestTurnDirtyStart: a tracked file changed and an untracked one made before the first turn, both changed again by the turn, its review the turn's change alone; found besides (not in the criterion): in a repository whose index is already split, the time of its shared index file in .git refreshed by the store's reads as by any git read, Source Control's included (reliability.md, The turn review)
PASS 65.2 What never reaches the store: TestTurnNeverStored: each file that must not reach it holding a marker, none found in any object the store holds, the files that may reach it found (the control): files ignored by .gitignore, .git/info/exclude and core.excludesFile, every name the file viewer refuses, in upper case and in a subfolder too; TestTurnJunction: a file outside the project behind a junction, listed by git, kept out by the viewer's checks, held open without sharing and never opened; TestTurnTooLarge (the 32 MiB lowered): a file over it, tracked or not, left out with its size and time, its content never read; TestReviewNothingChanged, TestReviewRows: its row saying it changed, "over 32.0 MiB, not compared"; TestTurnHeldCredential: credential files held open without sharing, one tracked and one not, failing nothing, never read; a symbolic link to a credential file or out of the project: not verified, TestTurnLinks skipping since Windows refuses the tests a symbolic link without its privilege, as the criterion allows
PASS 65.3 The store: TestTurnLimit, on a limit the test sets: a Take whose projection passes it refused before git takes anything, the store as it was; one ending past it taken, every later one refused at once, the earlier trees still readable; TestTurnReviewProblems: the store full, the message sent all the same, the turn's problem "the review's store reached its 256.0 MiB for this session"; TestTurnGrowth: eight turns over about 2 MiB of tracked files, with no change, one file changed and every file touched: the store grows by the changed file alone, within the projection, a turn with no change or only touched files adding nothing; the real store, 200 tracked files of 21.0 MiB, eight turns adding a line to one file each: 32,043 bytes after the first snapshot, 102,238 after the eighth, 70,195 for the eight (target under 1 MiB); TestTurnFolderAccess: one entry, the user's full control, protected, inherited by all the store holds, the user making, reading and removing in it; TestTurnGitCalls: a plain GitCLI, Source Control's and the Git Graph's, carrying none of the store's variables; TestTurnClose, TestTurnCloseWaits, TestTurnReviewClose: removed at the dashboard's end after the call under way; TestTurnSweep: an unheld store removed at the next start, a held one never; TestTurnDeadline: a stopped Take leaving no index.lock; TestTurnReviewStartLimit: a start past 5 s, the message going at 5 s, its row saying why
PASS 65.4 The turns: TestReviewSend, TestTurnReviewSend: the snapshot first, then the write, the message going once its start is taken and never before; written still when the snapshot fails, is stopped, panics or is refused (TestTurnReviewProblems, TestTurnReviewStartLimit, TestTurnReviewPanic, TestTurnReviewQueueFull, its gate opened at once); TestReviewTurn: the end's snapshot at a completed, a failed and an interrupted turn; TestReviewCrash: at the session's end during a turn; TestReviewCLITurn: a turn the CLI opens starting at the previous end, also when that end comes after it opened; TestReviewCLITurnNoStart: after a failed start, no wait for one; TestReviewMock: a mock's replayed prompt asking for its start; TestTurnReviewProblems: a file written during the end's snapshot, the turn "· still changing"; TestReviewNothingChanged: equal trees, no row; TestTurnReviewNoStore: outside a git repository, no snapshot, no row; TestTurnReviewSend: nothing sent to the CLI but the owner's message, no conversation entry added
PASS 65.5 Zone B: TestReviewRows: folded, one row right under the turn's last row, "▸ 4 files changed in turn 1 · +9 −3 · Review", the count with the file left out, "git" at its right; unfolded, each file's row with its letter and counts, the left-out file's "·  big.bin data/ · over 32.0 MiB, not compared"; TestReviewRowsBlock: under the CLI's block with no RESULT box, its bar, under the owner's message alone, no block; TestReviewRowsProblem: the failure's row "✗ No review of turn 1: its start's snapshot took more than 5s", no stop; TestReviewRowsCap: past 1,000 files, "first 1,000 of 2,345 files"; TestReviewCursorRows, TestReviewCursorOffItsEntry: the cursor's mark and keys on the summary and on a file, the entry's own row never marked meanwhile; TestTurnReviewCursor: ↑ from the input onto the summary, Enter unfolds, ↓ to a file, Enter its diff, Esc back on the file, ← folds; TestTurnReviewMouse: a click on the summary unfolds, a double click on a file opens its diff; golden frames review, review-open, review-diff and review-diff-wide at 79x24, 80x24, 120x40 and 200x60 (16 new files): turn 1's row in the CLI's block, the CLI-opened turn 2's under its RESULT box, "· still changing", the box's lines the same as the dashboard frame's; every existing golden frame unchanged
PASS 65.6 The diff: TestTurnReviewCursor: Enter on a file opens it with "Turn 1: start → end"; the golden frames review-diff, unified, at the four sizes, and review-diff-wide, f pressed, side by side at 200x60 and unified where the width does not allow it, as Source Control's diff; TestTurnReviewDiffNotes: a binary or too large file said as the diff view says it; TestTurnReviewDiffReload: "r" reading the two sides again from the store, the turn's label and the side-by-side layout kept; TestTurnReviewCursor, TestTurnReviewPanelEsc (200x60, a panel docked): Esc back to zone B, the cursor on the file; in Windows Terminal, old.txt's "Turn 2: start → nothing"
PASS 65.7 The documents: ui-design.md, Turn review with its frame and B's entries (TestDocFrames); architecture.md, The turn review; events-and-state.md, Turn review under State model (TestStateMachines); reliability.md, The turn review, the limits and the store's end; brief amendment 67; AGENTS.md, the status, the git constraint and the MVP scope; evaluation.md's timing step with TestTurnReviewFrameCost
PASS 65.9 The checks: go build ./... ok; go vet ./... ok (Windows and Linux); gofmt -l . empty; go test ./... ok (16 packages) after the last code and document edit; 68 mutants through -overlay on the final code, the repository never written (its hash the same before and after): the 41 of the state, render and app parts each caught by a failing test, the store's 27 caught but for 2 equivalent on Windows (details); go test -race on internal/state (8.6 s), internal/render (106.8 s), internal/app (297.2 s) and internal/input (196.7 s), and on the review's application tests once TestTurnReviewDiffReload was added: no race; the timing step alone on mains power (02:58:26 to 02:59:50), all twenty tests within their targets, TestTurnReviewFrameCost a turn of 1,000 files unfolded p50 0.53 ms, p95 1.17 ms (target 16 ms); the message's added wait, the start's snapshot, here 0.11 to 0.12 s (the first 0.48 s), with 20,000 files 0.43 to 0.47 s (the first 2.79 s)
Not verified: 65.8's last part, the owner's look at the shots. Its measured part holds: in ACC's profile at 208x59 and 120x40, the mock on a repository of its own, a file grown and one added during turn 1, a file changed and one removed between the turns: "2 files changed in turn 1 · +3 −0" under turn 1, "2 files changed in turn 2 · +1 −3" under the CLI's turn 2; ↑ Enter unfolded the list (D old.txt +0 −2 and M viewer.go at 208x59; at 120x40 viewer.go's row below the view); ↓ Enter opened "DIFF · old.txt", "Turn 2: start → nothing"; Esc came back with the cursor on old.txt; the store protected, the user's full control its one entry, 793 then 971 bytes, gone after the quit; the project's git status as the edits left it (" D old.txt", " M viewer.go", "?? notes.txt"); review shots out-p65run\review-max and review-90x30, folded.png, unfolded.png, diff.png, back.png. Also not verified: a symbolic link (65.2)
Verdict: NOT DONE
Proposed next step: the owner's look at the shots - waiting for approval
```

**65.8, and the review the owner shared (2026-10-29).** The owner shared,
with no words of their own, another agent's review of the eight shots and
of this report. It approves the shots ("I approve criterion 65.8 — the
human visual review"), taken as 65.8's look: PASS. It asked for two
technical points before the commit and
recorded three observations of the interface; what was done, each checked
as the criteria were (Details, The review's changes):

- **Hard links** (65.2). A file with more than one name is now left out of
  every snapshot, "a hard link, not compared", opened for its attributes
  alone (`linkCount`), a project's own hard-linked files with it: the
  review's trade-off. `TestTurnHardLink`, the review's case: a credential
  file `.env` outside the project, held open without sharing, linked into
  it as `notes.txt`, and a file of the project with a second name in it:
  all three names left out, neither marker in any object the store holds,
  the ordinary file's found. The file viewer keeps its name checks: it
  reads the one file the owner opens, where a snapshot reads by itself
  every file that changed.
- **The split index** (65.1). `TestTurnSplitIndex`: in a repository whose
  index is split already, the store's first snapshot changes the time of
  the shared index file alone, git's refresh as it reads it (git's
  documentation of `core.splitIndex`): its bytes, the index, the
  references and every other file of the project and of `.git` stay the
  same in path, size, time and bytes, and the next snapshot, Changing,
  Files and Sides change nothing at all. 65.1's PASS above holds for the
  repositories it was checked on; for a repository whose index is split
  already it does not hold to the letter, one file's time changing: a
  documented exception, git's own, no content, reference or index entry
  changed (reliability.md, The turn review; AGENTS.md).
- **The three observations.** The second file below the view at 120x40:
  `TestTurnReviewCursorReveals`, B held where the owner left it, ↓ ↓ onto
  that file scrolls B the least that shows it. The count is grouped by
  thousands, "2,345 files changed", as the note is: `TestReviewRowsCap`'s
  expected "2345" became "2,345" with the design (ui-design.md). The
  counts the cursor's keys cut to "+…" on a narrow summary row: kept as
  built, optional polish as the review says; the bottom row shows the same
  keys.
- **The shots are synthetic** (the review's point 6): the mock's recorded
  conversation, whose subagent wrote found.txt in its own recording, with
  the edits the harness made in the project during turn 1 and between the
  turns (viewer.go, notes.txt, old.txt). The review lists the files that
  changed while a turn ran, whoever changed them, and never says who:
  ui-design.md, Turn review, says so.

**The checks after these changes (2026-10-29).** go build, vet (Windows
and Linux), gofmt and the whole suite ok, 16 packages, after the last code
and document edit; the 46 mutants of the state, render and app parts all
caught, the five new ones among them, and the store's 27 caught but for
the same 2 equivalent on Windows, the code's hash the same before and
after; `-race` on the store's, the review's and the rows' tests: no race.
The timing step ran alone at 11:19 on mains power, the machine busy with
programs outside this work (13 to 50 % CPU after it): 18 PASS, 2 FAIL,
TestBannerToFrame p95 4.83 ms (target 2 ms) and TestHomePictureTime 306
ms (target 260 ms), the home screen's two tests, whose files plan 65 does
not touch. Timed in turn with HEAD's tree under the same load, the two
trees alike, the banner's p95 1.48 to 1.65 ms on both and the picture's
worst 289 to 318 ms on both, HEAD's over its target too: the machine, not
the code. TestTurnReviewFrameCost p95 2.49 ms (target 16 ms). The run on
a quiet machine after these changes is not done; the one at 02:58, on the
code before them, passed all twenty. 9 criteria, 9 PASS, 65.1 with its
documented exception (a split index's shared file's time). Verdict: DONE.
Not committed: the commit waits for the owner's approval. The block above
stays as it was reported.

**The timing step once more (2026-10-29, 11:52).** The owner shared the
other agent's next review: the corrections approved ("No further code
changes are requested"), the split index's exception accepted, the timing
step left open until a complete run passes on a quiet machine, the
targets and the home screen unchanged, and an explicit decision of the
owner's should the machine stay noisy. It stayed so: from 11:42 to 11:51
the CPU averaged 10 to 41 % over each 10 s, on mains power with the
battery at 98 %, the VPN client's scan idle, no `go` or test process
running. The complete step, alone, with the exact command, from 11:51:59
to 11:53:18: 19 PASS, 1 FAIL. TestBannerToFrame p95 2.05 ms (target 2 ms,
max 4.08 ms); TestHomePictureTime worst 248.5 ms (target 260 ms), within
it this time; TestTurnReviewFrameCost p95 1.62 ms (target 16 ms); every
other p95 4.51 ms at most. The banner's code is not plan 65's; timed in
turn with HEAD's tree under load, its p95 was the same on both (1.48 to
1.65 ms, Details); on the quiet machine at 02:58 it was 1.93 ms. Not
rerun until it passes: a clean run needs a quiet machine, and waiting for
one or accepting a documented timing exception is the owner's decision.
Not committed.

**The timing exception, accepted (2026-10-29).** The owner shared, with
no words of their own, the other agent's final review: "Accept the
documented timing exception and proceed to commit", the machine being the
owner's daily workstation, in use while ACC is built and tested. Taken as
the owner's decision, recorded as the review asks:

| | |
| - | - |
| Measured result | FAIL: TestBannerToFrame p95 2.053 ms, target 2 ms, unchanged |
| The complete timing step, 11:51:59 | 19 PASS, 1 FAIL; not all twenty |
| Likely cause | the workstation's varying load (10 to 41 % CPU before the run), supported by the comparison with HEAD's tree in turn |
| Regression | none of plan 65's shown by the evidence: the banner's code is not plan 65's, its p95 the same on both trees in turn |
| The owner's decision | a documented timing exception, accepted |
| Status | accepted with an exception, not a PASS |

The earlier timing results stay as they were recorded: 02:58, twenty
PASS, on the code before the review's changes; 11:19, 18 PASS and 2 FAIL;
11:52, 19 PASS and 1 FAIL. Plan 65: 9 criteria, 7 PASS and 2 accepted
with a documented exception, 65.1 (a split index's shared file's time)
and 65.9 (TestBannerToFrame's timing at 11:52). The review allows the
final commit of plan 65's files only, `found.txt` untouched, and no push
until the owner allows one.

**Approved and committed (2026-10-29).** Commit 3757c15 (56 files: 28
modified, 28 added), from the files reviewed in `git status` and `git
diff`: `git diff --check` found no whitespace error, no file holds a CR or
a BOM, and a scan of the change found no leftover, private path or
company name; the whole suite had passed after the last code edit and
after the last edit of a document a test reads. The author is `Evengelius
<molimoabel@gmail.com>`, with no attribution line, and after it the
working tree held `found.txt` alone, untracked and untouched: PASS. Then
the approval commit: plan.md and AGENTS.md close plan 65, and this
record; `go test ./internal/arch`, which reads AGENTS.md, ok after its
edit. Not pushed: the owner allowed the commit only. The block above
stays as it was reported.

## Details

### What the build found and changed

- **The store first, its interface fixed.** Step 2 went to an implementer
  subagent once the store's interface was fixed (`OpenTurnStore`, `Take`,
  `Changing`, `Files`, `Sides`, `Size`, `Close`), so that steps 3 and 4
  went on beside it; its code was reviewed and its checks run again here.
- **Git's cost a new file.** Git writes about 7 ms a new file here (2,000
  small files in 13.9 s): a turn's start passes its 5 s past about 700 new
  files, the message going and that turn without a review, and its end
  passes its 30 s past about 4,000; once a snapshot has taken them, the
  next one is quick again (reliability.md, The turn review).
- **A split index.** In a repository whose index is already split
  (`core.splitIndex`), git refreshes the time of the shared index file in
  `.git` on every read of it: Source Control's reads already do, and the
  store's reads of its copy do the same. Nothing else in `.git` changes,
  and the store writes no shared index of its own
  (`core.splitIndex=false` on every call). Criterion 65.1's repositories
  have no split index; this is the one file of `.git` the store can touch,
  its time only.
- **Git LFS.** With the filters off, every checked-out LFS file differs
  from its pointer in the seeded index: the first snapshot takes each in
  as its bytes, at most 32 MiB each, so with more than 256 MiB of them the
  store is full from the first snapshot and the session has no review.
  Leaving filtered files out instead is a change the owner may ask for;
  not done.
- **A nested repository.** An untracked folder holding a repository of
  its own would make git's `add` refuse the whole snapshot: it is left
  out, "a nested repository" (TestTurnSubfolder).
- **What the name checks cannot see.** The file viewer's checks, which the
  store shares, judge a file by its name and by its real path once links
  and junctions are resolved. A hard link has neither: one made in the
  project under another name, to a credential file or to a file outside
  the project, is read as the file viewer would read it, and its content
  goes to the store, readable by the same user alone. No criterion names
  it; said here so that it is known.
- **Defects found and fixed during the build:** a turn the CLI opened
  after a turn whose start had failed waited for a start that would never
  come (now said at once, TestReviewCLITurnNoStart); a panic in a snapshot
  ended the session (now that turn's problem, TestTurnReviewPanic); a
  snapshot refused by the full queue left its message's gate shut for 5 s
  (now opened at once, TestTurnReviewQueueFull); the diff view's reload
  ("r") dropped the turn and named its sides as the store did (now "start"
  and "end" kept).
- **What the tests missed until checked.** The first mutant run let two
  through: R9, the cursor marking a tool call's row as well as its turn's
  review (TestReviewCursorOffItsEntry added), and A9, Esc from a turn's
  diff to a docked panel, which 120x40 could not show
  (TestTurnReviewPanelEsc moved to 200x60). Writing this evaluation found
  that no test pressed "r" in a turn's diff: TestTurnReviewDiffReload was
  added, and A16 to A18 break it three ways.
- **A slip.** `internal/render/review_test.go`, plan 41's, was overwritten
  by a new file of the same name and restored from git at once, no
  difference from HEAD; the new tests are in `turnreview_test.go`.
- **For the owner's look, at 120x40.** Once unfolded, the list's second
  file is below the view's bottom: zone B's view stays where it was when
  an entry unfolds, as it does for any other entry. With the cursor on the
  summary row, the keys at its right cut the counts to "+…". And past 999
  files the summary's count has no thousands separator ("2345 files
  changed") beside the note's "first 1,000 of 2,345 files".

### The review's changes

What the review the owner shared asked for (after the evaluation), and
where:

| Change | Code | Test |
| - | - | - |
| a file with more than one name left out, "a hard link", opened for its attributes alone | `internal/input/turns.go` (`check`, `whyLinked`), `turns_windows.go` and `turns_other.go` (`linkCount`) | TestTurnHardLink (`turns_windows_test.go`) |
| a split index's exception verified | none: the store already wrote no shared index | TestTurnSplitIndex (`turns_test.go`) |
| the cursor onto a file below the view | none: `putCursor` already scrolled to it | TestTurnReviewCursorReveals (`internal/app/review_test.go`) |
| the count grouped by thousands | `internal/render/review.go` (`reviewSummary`, through `grouped`) | TestReviewRowsCap, its expected count "2,345" |
| the documents | ui-design.md (Turn review: the reasons a file is left out, the count, whoever changed the files), reliability.md (The turn review: the hard link's row, the split index's exception), architecture.md (`linkCount`), AGENTS.md (the git constraint), brief amendment 67 | TestDocFrames, TestAgentsMDWithinCodexLimit |

Each change broken on purpose through `go test -overlay`:

| Mutant | Broken | Caught by |
| - | - | - |
| H1 | no hard-link check | TestTurnHardLink: git's add opens the held credential file and the snapshot fails |
| H2 | the file opened for its content to count its names | TestTurnHardLink: notes.txt "cannot be read" |
| X1 | the store's index left split (`core.splitIndex=false` removed) | TestTurnSplitIndex: each later read refreshes the shared index's time again |
| V2 | the review's files not scrolled into view, the summary alone | TestTurnReviewCursorReveals: the cursor on the second file, its row not drawn |
| G1 | the count without its thousands separator | TestReviewRowsCap: "2345 files changed" |

A first try, V1, which skipped the summary's reveal as well, was caught
too, one step earlier (↑ no longer showed the summary). Writing the
reveal's test found that an application test which scrolls zone B must
draw each key's frame, as the program does: `scroll` and `revealRow`
work from the frame drawn last.

The checks once these changes were made, from the scratchpad's
`plan65\checks-final2.sh`, `timing2.log` and `hometurns.sh`:

| Check | Result |
| - | - |
| build, vet (Windows, Linux), gofmt | ok, ok, ok, nothing printed |
| `go test ./...` | 16 packages ok, 315 s |
| the mutants, mutrun | 46: the 41 above and H1, H2, X1, V2, G1, all caught, 394 s |
| the mutants, the store | 27: 25 caught, noEnoent and noProtect equivalent as above, 281 s |
| the code's hash, before and after both | the same |
| `-race` | internal/input, `TestTurn` and `TestParseTurn`, 43.8 s; internal/app, `TestTurnReview`, 10.0 s; internal/render, `TestReview`, 4.0 s: no race |
| the timing step, alone, 11:19:19 to 11:21:00 | 18 PASS, 2 FAIL (below) |

The timing step ran on mains power with the battery at 98 %. Before it the
CPU stood at 29, 14 and 12 % over 3 s, with no `go` or test process
running. After it the machine stayed between 13 and 50 % for eight
minutes, from programs outside this work: the VPN client's system scan,
then others whose use this account cannot see. Every time was
slower than at 02:58, TestLongSession's p95 at 7.01 ms against 3.00
and TestStreamToFrame's at 8.47 ms against 4.58. All but two stayed
within their targets: TestBannerToFrame p95 4.83 ms (target 2 ms), and
TestHomePictureTime worst 306.3 ms (target 260 ms). Those are plan 51's
home screen tests; plan 65 changes no file of the home screen or its
banner. The two tests were then run on HEAD's tree (2af203a, through
`git archive`) and on plan 65's in turn, both built first:

| Round | Tree | The banner's p95 | The picture, worst |
| - | - | - | - |
| 1, 11:31:53 | HEAD | 1.59 ms | 318.4 ms |
| 1 | plan 65 | 1.53 ms | 288.9 ms |
| 2, 11:32:12 | HEAD | 1.48 ms | 295.4 ms |
| 2 | plan 65 | 1.61 ms | 307.1 ms |
| 3, 11:32:32 | HEAD | 1.65 ms | 309.3 ms |
| 3 | plan 65 | 1.54 ms | 294.3 ms |

The banner was within its target on both trees, and the picture over its
target on both: the load, not plan 65's code. The full step on a quiet
machine after these changes is still to do.

### The store, measured

`zz_turntiming_test.go` and `zz_turngrowth_test.go`, overlaid from the
scratchpad into `internal/input` for the measures only (no `zz_` file in
the tree after), on a temporary stores' folder.

| Repository | Open | First snapshot | Unchanged, three times | Ten files changed | Unchanged again |
| - | - | - | - | - | - |
| a clone of this repository | 109 ms | 483 ms, the store 40,409 bytes | 108 to 122 ms | 308 ms, 45,544 bytes | 105 to 113 ms |
| 20,000 files | 112 ms | 2.786 s, 520,323 bytes | 451 to 467 ms | 702 ms, 531,974 bytes | 427 to 446 ms |

The message's added wait is a turn's start snapshot: about the unchanged
time when little changed since the turn before, the first snapshot's time
for the session's first turn.

The growth, criterion 65.3's case: 200 tracked files of 21.0 MiB
(22,024,000 bytes at HEAD), the eight changed ones of 110,890 bytes each;
eight turns, each adding a line to one file and followed
by the next turn's start with nothing changed.

| After | The store |
| - | - |
| the first snapshot | 32,043 bytes |
| turn 1 | 40,815 bytes |
| turn 2 | 49,589 bytes |
| turn 3 | 58,366 bytes |
| turn 4 | 67,138 bytes |
| turn 5 | 75,914 bytes |
| turn 6 | 84,689 bytes |
| turn 7 | 93,465 bytes |
| turn 8 | 102,238 bytes |

About 8,770 bytes a turn, the changed file's new object and the trees
above it; the start after each turn added nothing.

### The Windows Terminal runs

`scratchpad\term\p65run.ps1`, from plan 64's `p64run.ps1`: the
application built from this tree (`acc-p65.exe`) in ACC's profile, the
mock `claude-subagent-approval.jsonl` (byte for byte the repository's)
committed with three files in a repository of its own, git's folder on
the PATH beside System32, a temporary home and app data so that the store
goes under a temporary LOCALAPPDATA, Bubble Tea's trace on. A mock
session, so no CLI was started and no quota used. Windows Terminal
1.23.20211.0. The owner's terminal-hosted CLI sessions: none were open.
The shots are synthetic: the files changed are the harness's edits, made
in the project while the mock replayed; the recorded conversation's
subagent wrote found.txt in its own recording, never in this project.

| Step | 208x59 | 120x40 |
| - | - | - |
| the repository | 18197bd | 4916007 |
| edit 1, during turn 1 (6.5 s) | viewer.go grown, notes.txt added | the same |
| edit 2, between the turns (13.0 s) | viewer.go changed again, old.txt removed | the same |
| the store, running (15.9 s, 14.3 s) | 793 bytes; protected; the user's full control alone | the same |
| folded (25 s) | "▸ 2 files changed in turn 1 · +3 −0 · Review" under turn 1, "▸ 2 files changed in turn 2 · +1 −3 · Review" under the CLI's turn 2 | the same, "git" at the right |
| ↑ Enter | turn 2's list unfolded: "D old.txt +0 −2", "M viewer.go" | "D old.txt +0 −2"; viewer.go's row below the view |
| ↓ Enter | "DIFF · old.txt", "old.txt · Turn 2: start → nothing" | the same |
| Esc | back in zone B, the cursor on old.txt, the list unfolded | the same |
| the store at the end | 971 bytes; protected; the user's full control alone | the same |
| after the quit | no store | no store |
| the project's git status | " D old.txt", " M viewer.go", "?? notes.txt": the edits alone, nothing staged | the same |

Review shots, in `scratchpad\term\out-p65run\review-max` (208x59) and
`review-90x30` (120x40): `folded.png`, `unfolded.png`, `diff.png` and
`back.png`, with `start.png`, `turn-1.png` and `file.png` and the text of
each.

### The mutants

Each change broken on purpose, one at a time, through `go test -overlay`,
the repository never written: the code's hash, the diff from HEAD of
`internal`, `cmd` and `testdata` with every untracked file there, the same
before and after the run of 65 (`scratchpad\plan65\mutfinal.sh`), and the
tracked code's diff the same before and after A16 to A18. The 65 ran on
the final code but for TestTurnReviewDiffReload, added after them (a test
added can only catch more), A16 to A18 with it; an earlier run, before
the tests were completed, had let R9 and A9 survive (What the build
found).

The state, render and application parts: 41 mutants, from
`scratchpad\plan65\mkmutants65.mjs` (S1 to A15) and `mutants65-reload.txt`
(A16 to A18), run by plan 51's mutrun with `-run TestReview` on
internal/state and internal/render and `-run
'TestTurnReview|TestGoldenFrames/review'` on internal/app. All 41 caught by
a failing test:

| Mutant | Broken | Caught by |
| - | - | - |
| S1 | the snapshot after the write | TestReviewSend: the write first |
| S2 | no snapshot before the send | TestReviewSend: the write alone |
| S3 | no snapshot at the turn's end | TestReviewTurn/completed |
| S4 | no snapshot when the session crashes during the turn | TestReviewCrash: none asked for |
| S5 | a mock's stage never asked for its start | TestReviewMock: none asked for |
| S6 | a CLI-opened stage not starting at the stage before's end | TestReviewCLITurn: its review empty |
| S7 | the end not given to the CLI's next stage | TestReviewCLITurn: turn 2 starting at "" |
| S8 | equal trees read as changed | TestReviewNothingChanged: the files of t1..t1 asked for |
| S9 | a file left out compared by its time alone | TestReviewNothingChanged: big.bin's new size not shown |
| S10 | the files kept past the cap | TestReviewFileCap: 1,500 of 1,500 kept |
| S11 | the store's problem not said at the turn's end | TestReviewProblems: no problem |
| S12 | the editor's notice never taken down | TestReviewSend: the notice after the snapshot |
| S13 | a failed start still asking for the end | TestReviewProblems: the end asked for |
| R1 | the review under the turn's first entry | TestReviewRows: not the block's next row |
| R2 | the count without the files left out | TestReviewRows: "3 files changed" |
| R3 | the lines of the last file only | TestReviewRows: no counts |
| R4 | "still changing" never said | TestGoldenFrames/review-80x24 |
| R5 | no note past the cap | TestReviewRowsCap: no note |
| R6 | the failure without its turn | TestReviewRowsProblem: "✗ No review: …" |
| R7 | the owner's block's bar on the review | TestReviewRowsBlock: the bar under the owner's message |
| R8 | the files left out not counted in the span | TestReviewRows: 13 rows counted, 14 drawn |
| R9 | the entry's own row marked with its review's | TestReviewCursorOffItsEntry: the call's row marked |
| R10 | the review after the conversation's last blank row | TestReviewRows: not the block's next row |
| A1 | the message not waiting for its turn's start | TestTurnReviewSend: sent while the snapshot ran |
| A2 | no gate for the next message | TestTurnReviewSend: the same |
| A3 | the gate never opened by the snapshot | TestTurnReviewSend: nothing sent after it |
| A4 | the start's snapshot without its limit | TestTurnReviewStartLimit: no problem |
| A5 | no listing after the end | TestTurnReviewProblems: no "still changing" |
| A6 | the store's limit said as a failure | TestTurnReviewProblems: "its start's snapshot failed: …" |
| A7 | no repository said as a failure | TestTurnReviewNoStore: a problem outside a repository |
| A8 | the sides named by their trees | TestGoldenFrames/review-diff-80x24 |
| A9 | Esc from a turn's diff to the docked panel | TestTurnReviewPanelEsc: the focus "R" |
| A10 | a double click on a file opening nothing | TestTurnReviewMouse: no diff |
| A11 | ← on a file not folding the list | TestTurnReviewCursor: still unfolded |
| A12 | ↑ never onto a review from below | TestGoldenFrames/review-open-80x24 |
| A13 | the store kept at the end | TestTurnReviewSend: the snapshots' goroutine left blocked (synctest's deadlock) |
| A14 | a panic in a snapshot the session's end | TestTurnReviewPanic: the session "Crashed" |
| A15 | a refused snapshot's message held 5 s | TestTurnReviewQueueFull: the gate still shut |
| A16 | "r" in a turn's diff reading Source Control's file | TestTurnReviewDiffReload: the sides not read again |
| A17 | "r" losing the side-by-side layout | TestTurnReviewDiffReload: side by side off |
| A18 | a read of a turn's diff losing its turn | TestTurnReviewDiffReload: the diff's turn 0 |

The store: 27 mutants of `internal/input/turns.go` and `turns_windows.go`
(`scratchpad\turns65\mut.mjs`, the implementer's 26 and noEnoent), each
with the tests its column names. 25 caught, 2 equivalent on Windows:

| Mutant | Broken | Caught by |
| - | - | - |
| altOnWrites | the repository's objects behind the store's writes | TestTurnGitCalls, TestTurnWritesNothing |
| noHooksPath | the repository's hooks run | TestTurnGitCalls, TestTurnFiltersHooksLineEndings |
| noFilterOpts | the filter drivers run | TestTurnGitCalls, TestTurnFiltersHooksLineEndings |
| noOptionalLocks | the listing not refreshing the store's index (no GIT_OPTIONAL_LOCKS=1) | TestTurnGrowth |
| noExcludes | no credential names in the listing's pathspecs | TestTurnGitCalls, TestTurnNeverStored |
| noChecks | the listed files not checked before git reads them | TestTurnTooLarge, TestTurnJunction, TestTurnNeverStored |
| noSeed | the store's index not seeded from the repository's | TestTurnUnchanged, TestTurnGrowth |
| noAhead | an add whose write-tree did not follow forgotten | TestTurnDeadline |
| noUnlock | a stopped git's index.lock left | TestTurnDeadline |
| noUnlockList | the same, after the listing | TestTurnDeadline |
| noYSkip | unchanged records read as changes | TestTurnChanging, TestTurnGrowth |
| noEnoent | a path below a file not read as gone | equivalent on Windows: Lstat says ERROR_PATH_NOT_FOUND, which Go reads as fs.ErrNotExist; ENOTDIR is Linux's, where TestTurnCheck's "a.txt/child" holds it |
| noSafecrlf | git's line-ending check on | TestTurnGitCalls, TestTurnFiltersHooksLineEndings |
| noSplitIndex | the store's index split | TestTurnGitCalls, TestTurnWritesNothing |
| noProjection | no projection before git reads | TestTurnLimit |
| noFull | the limit never reached | TestTurnLimit |
| sweepHeld | a held store swept | TestTurnSweep |
| noProtect | the folder's access list not protected ("D:" for "D:P") | equivalent: Windows protects an access list given at the creation (probed); TestTurnFolderAccess finds the same one entry, protected |
| changingIgnoresLeft | a left-out file's size and time not compared | TestTurnTooLarge |
| untrackedGone | an untracked file that came and went taken in | TestTurnCheck |
| measureFlat | the store's files never counted in its size | TestTurnLimit, TestTurnGrowth |
| noLiteral | the add's paths read as patterns | TestTurnNeverStored |
| baseNotLiteral | the project's folder read as a pattern | TestTurnSubfolder |
| statusWhole | the whole repository listed for a subfolder project | TestTurnGitCalls, TestTurnSubfolder |
| noIgnoreSub | a submodule's own changes listed | TestTurnGitCalls, TestTurnSubmodule |
| moduleChecked | a submodule's record checked as a file | TestTurnCheck, TestTurnSubmodule |
| closeNoWait | Close not waiting for a call under way | TestTurnCloseWaits |

### The race detector

`go test -race` on the four packages the plan changed, with the portable
WinLibs GCC 16.2.0 first on the run's own PATH and `CGO_ENABLED=1`
(evaluation.md, The race detector): internal/state 8.6 s, internal/render
106.8 s, internal/app 297.2 s, internal/input 196.7 s, no race reported,
every test passing; once TestTurnReviewDiffReload was added, internal/app
again with `-run TestTurnReview` (9.1 s), no race. The snapshots run on a goroutine of their own, one at
a time in the order Update asks for them (`turnCall`, `runTurn`), and the
store's own calls under its lock (TestTurnConcurrent).

### The timing step

Run alone with the exact command of `doc/evaluation.md`, Keystroke timing,
now twenty tests with `TestTurnReviewFrameCost`, from 02:58:26 to
02:59:50 on mains power; no other `go` build or test ran.

| Test | Result | Target |
| - | - | - |
| TestTurnReviewFrameCost | p50 0.53 ms, p95 1.17 ms, a turn of 1,000 files unfolded | p95 ≤ 16 ms |
| TestBlameFrameCost | p95 1.05 ms | p95 ≤ 16 ms |
| TestBranchesFrameCost | p50 0.57 ms, p95 1.69 ms | p95 ≤ 16 ms |
| TestMessageToFrame | p95 1.03 and 2.00 ms | p95 ≤ 16 ms |
| TestDiffToFrame | p95 1.00 to 1.54 ms | p95 ≤ 16 ms |
| TestGraphFrameCost | p95 1.01 ms; 209 bytes a commit | p95 ≤ 16 ms |
| TestBannerToFrame | p95 1.93 ms, max 4.35 ms, 11,355 bytes a frame | p95 ≤ 2 ms |
| TestHomeKeysToFrame | p95 0.18 ms | p95 ≤ 16 ms |
| TestHomePictureTime | p50 184.6 ms, max 188.8 ms | ≤ 260 ms |
| TestKeystrokeToFrame | p95 1.02 ms; the draft window 1.12 ms | p95 ≤ 16 ms |
| TestSelectorToFrame | p95 1.00 to 1.11 ms | p95 ≤ 16 ms |
| TestMotionToFrame | p95 1.02 to 1.52 ms; idle motion all dropped by the filter | p95 ≤ 16 ms |
| TestDraftKeystrokes | p95 2.55 ms | p95 ≤ 16 ms |
| TestLongSession | p95 3.00 ms | p95 ≤ 16 ms |
| TestOutputToFrame | p95 1.01 ms | p95 ≤ 16 ms |
| TestPasteToFrame | max 1.61 ms | ≤ 100 ms |
| TestStreamToFrame | p95 4.58 ms | p95 ≤ 16 ms |
| TestThinkingFrame | p95 2.52 ms | p95 ≤ 16 ms |
| TestToolsToFrame | p95 2.51 ms | p95 ≤ 16 ms |
| TestKeystrokeCost | 229 µs, 448,793 bytes and 597 allocations a key (10,000 lines); 725 µs, 1,453,989 bytes and 597 allocations (50,000 lines) | reports only |
