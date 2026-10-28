# Plan 60 evaluation (2026-10-28)

The check of plan 60, git blame in the file viewer (brief amendment 62):
`b` in the viewer shows, left of the line numbers, the date and author of
the commit that last changed each line, after the owner's image of an IDE's
blame, and in the footer the cursor line's commit; the lines not committed
yet shown apart, claiming no author; `b` again hides them.

It covers the 9 criteria of [`plan.md`, Plan 60](../plan.md), written
before the code, with the three precisions of the review the owner folded
in. Nothing is committed or pushed: the owner reviews first.

How it was checked:
- **Input tests on real repositories** (`internal/input/blame_test.go`,
  4 tests; `TestReadFile`): the porcelain read, the bytes given against the
  disk, a CRLF file, git's refusals, the time limit; every git call traced.
- **Rendering tests** (`internal/render/blame_test.go`, 3) and **golden
  frames** `file-blame` at 80x24, 120x40, 200x60 and 79x24.
- **Application tests** (`internal/app/blame_test.go`, 8) with a test double
  of git that can hold each call, and one test with git itself.
- **Each change broken on purpose** through `go test -overlay`
  (`scratchpad\plan51\mutrun`, the lists built from the source lines by
  `scratchpad\plan60\mkmutants.mjs`), the repository never written.
- **Windows Terminal**: the real application in the acc tab at 208x59,
  120x40 and 80x24, a mock dashboard over a demo repository, a temporary
  home and app data, no CLI session (`scratchpad\term\p60run.ps1` on
  `p51.ps1`, with `p60demo.ps1`).
- **The timing step** alone, and **build, vet, gofmt, the whole suite**
  after the last code and document edit.

## The evaluation

```text
Evaluation - Plan 60, git blame in the file viewer - 9 criteria: 8 PASS, 0 FAIL
PASS 60.1 The switch: TestBlameSwitch: off at start, git never asked, the hints naming "b blame"; b blames main.go from its bytes (one call, the bytes the disk's), the column and the footer's commit shown; b again hides them and the hints name b again; on, the next file shown (web/app.ts) blamed from its own read, its one line drawn; TestBlameAnswers: "blame…" on the next file while git reads; TestKeyOverlayRows: the key overlay's b row, 78 rows, ui-design.md's tables the same; in Windows Terminal b on, off and on, and notes.txt opened next with the blame still on
PASS 60.2 The column: TestBlameColumn at 120x40: "15/11/2022 Ana Hubaut    1  alpha", "02/03/2023 Bo Li         2  beta", "Not committed            3  gamma", aligned; "Not committed" in the attention colour, a committed line grey, the cursor row's column on its ground; a long name cut to 12 cells ("Christophe-…"); blank while git reads, not drawn when off; TestBlameCommits: the dates in the zone given (Bo Li's 08:30 -0500 at 14:30 +0100), the short hash, the initials (three at most, "ÉG"), a line not committed with none of git's author; TestBlameGit with git itself: "15/11/2022 Ana Hubaut      1  // line 1", the line changed "Not committed"; golden frames file-blame at 80x24, 120x40, 200x60 and 79x24
PASS 60.3 The narrow form: TestBlameColumn: at 80x24 "15/11/22 AH   1  alpha", "uncommitted   3  gamma" in the attention colour; full screen at 80x24 compact, at 120x40 full; the full form from exactly 60 cells of code and the compact one a cell below, windowed (115 and 114 wide) and full screen (92 and 91); TestBlameWidth: a row as wide as asked, a long line's code cut to what the column leaves; in Windows Terminal the full form at 208x59 and 120x40, the compact one at 80x24, full screen too
PASS 60.4 The footer: TestBlameFooter: "f52c343 · Ana Hubaut · 15/11/2022 10:00 · Add f.txt with three lines" on the first line, Bo Li's on the fourth, "Not committed yet: changed in the working tree" in the attention colour on the third; a long subject cut with "…" before " line 1/4"; a toast first; the search field's own hints while it is typed; the keys with the blame off; TestBlameSwitch: g 21 and g 36 move the footer to Bo Li's commit and to a line not committed; in Windows Terminal g to Bo Li's line, a line not committed and Maximilian Oberhausen's, cut with "…" at 80x24
PASS 60.5 The messages: TestBlameMessages: "not in git yet: no history", "no commits yet", "blame stopped after 15 s", "not a git repository", "git not found", git's own words ("fatal: bad object HEAD"), each with the file's 100 lines shown; two lines blamed for 100, and a file changed on disk since it was shown: "the file changed after it was read: show it again", git not asked for the second; TestBlameFooter: "blame…" grey, a failure in the attention colour; TestBlameRefusals with git itself: untracked, no commits, outside git, the time limit; in Windows Terminal "not in git yet: no history" on an untracked file, the column blank
PASS 60.6 Unchanged: TestBlameUnchanged: the blame off, four files shown and git never asked; b typed in the search goes to the search; a drag from the column's first cell copies the code alone ("// line 3", no date or name); full screen draws the column; plan 58's click outside closes the viewer; TestBlameWidth: the shift's limit the code's width, less the column only while it shows; the eight viewer frames whose hints row now names "b blame" changed in that row alone
PASS 60.7 Git and the answers: TestBlame on a real repository: one "blame --porcelain --contents - -- f.txt" traced, the file system monitor off, through stream, the read call that takes a standard input (runRead takes none), its environment runRead's: no optional lock, git's words in English; a commit's headers read once and its later lines by hash; the bytes given, not the disk's; TestBlameCRLF: a CRLF file blamed as committed from its raw bytes, as changed without its CRs; TestReadFile: the bytes read kept with their CRs, none for a binary file; TestBlameAnswers: an answer dropped when another file was shown during its read, when it came while the next file was still being read, when the viewer was closed during it, and when b went off and on during it, only the latest read drawn; TestBlameRead: blameRead's rule whatever the numbers (another file, another read, the viewer closed, the blame off); TestBlameGit: one call with git itself
PASS 60.9 The checks: go build ./... ok; go vet ./... ok (Windows and Linux); gofmt -l . empty; go test ./... ok (16 packages); 4 new golden frames reviewed (file-blame at 80x24, 120x40, 200x60 and 79x24), 8 viewer frames changed in their hints row alone, no mock file changed; 46 mutants through -overlay in three runs, each caught by a failing test in the end, none by a build failure (details); the timing step alone (mains power, CPU 36 % just before, the owner's usual applications open): its eighteen tests within their targets, a key to its frame with 10,000 lines blamed p50 0.53 ms, p95 1.52 ms (target 16 ms)
Not verified: 60.8's last part, the owner's visual review. Its measured part holds: in Windows Terminal at 208x59, 120x40 and 80x24, b showed the column and the footer's commit (the full form, then the compact one at 80x24, full screen too), g moved the footer to each author's line and to a line not committed, b hid and showed them again, an untracked file opened next said "not in git yet: no history"; review shots out-p60\blame-max\e8.png, blame-90x30\e7.png, blame-60x18\e8.png, e9.png, blame-90x30\e17.png
Verdict: NOT DONE
Proposed next step: the owner's visual review of the blame (the shots) - waiting for approval
```

## Details

### What the checks found and changed

- **The bytes read, with their CRs.** No test held what `ReadFile` keeps
  for git blame: a version dropping the CRs passed (I11), and git would
  then call every line of a CRLF file changed. `TestReadFile` now checks
  `FileText.Raw` against the bytes on disk, and none for a binary file.
- **The full form's boundary.** Nothing drew the column at exactly 60
  cells of code (R1); `TestBlameColumn` now finds the two widths around the
  boundary, windowed and full screen, and expects each form.
- **Three drawing details** no test separated: "uncommitted" in the
  attention colour (R4), a long subject cut by the footer itself, whose
  only trace once `spread` fits the left part is the space before the line
  count (R9), and a row's own width, which the frame's cut hides on screen
  (R14, now checked on `codeRow`).
- **The answer arriving while the next file is read.** The one case the
  file's generation alone decides: the answer for the file shown before
  comes after the next one opens, before its read is back. No test had it
  (A1); `TestBlameAnswers` now does. The checks of the viewer open and the
  blame on (A3, A4) never decide in the dashboard's own flow, since every
  close changes the file's generation and every `b` the read's number;
  `TestBlameRead` checks `blameRead`'s rule directly, whatever the
  numbers. And nothing checked that the next file shows "blame…" while git
  reads (A11).
- **The read's environment.** Design D5 names `runRead`; git blame takes
  the file's bytes on its standard input, which `runRead` cannot give, so
  it goes through `stream`, the read call that takes one (plan 44's for
  `cat-file`). Both build git's command with `command`, whose read form
  turns the file system monitor off (in the trace) and whose environment
  turns optional locks off and keeps git's words in English. No test held
  that environment; `TestBlame` now does, and I14 proves it can fail.
- **The harness's digits.** In Windows Terminal, `g` then the harness's
  digit keys moved nothing: on the owner's AZERTY layout those keys
  without Shift give "&", "é" and the like. The run types the line
  numbers; the dashboard was not at fault.

### The mutants

Each through `go test -overlay` (`scratchpad\plan51\mutrun`, the lists
built from the source lines by `scratchpad\plan60\mkmutants.mjs`, its
second round by the same script given the ids). The second round ran with
nothing edited meanwhile, `git diff` and `git status` hashed ae9c911b
before and after. During the first, this plan's own test edits for the
packages already done changed that hash (53586754 to ae9c911b); the seven
source files the mutants replace all date from before the run.

| Id | The change | Caught by |
| - | - | - |
| I1 | the disk blamed, not the bytes given | TestBlame |
| I2 | a commit's later lines not found by hash | TestBlame |
| I3 | the all-zero hash not a line not committed | TestBlame |
| I4 | the author's zone dropped | TestBlame |
| I5 | a zone west of Greenwich read east | TestBlame |
| I6 | an untracked file not named | TestBlameRefusals |
| I7 | no commits not named | TestBlameRefusals |
| I8 | a folder outside git not named | TestBlameRefusals |
| I9 | the time limit not named | TestBlameRefusals |
| I10 | a time limit not DeadlineExceeded to errors.Is | TestBlameRefusals |
| I11 | the bytes read without their CRs | TestReadFile (second round) |
| I12 | the subject not read | TestBlame |
| I13 | the author not read | TestBlame |
| I14 | the read's optional locks left on | TestBlame (third run) |
| R1 | the full form only past 60 cells of code | TestBlameColumn (second round) |
| R2 | the compact form everywhere | TestBlameColumn |
| R3 | "Not committed" grey | TestBlameColumn |
| R4 | "uncommitted" grey | TestBlameColumn (second round) |
| R5 | the cursor row's ground not under the column | TestBlameColumn |
| R6 | the compact form's long date | TestBlameColumn |
| R7 | the footer's commit not the cursor line's | TestBlameFooter |
| R8 | the footer's note not shown | TestBlameFooter |
| R9 | the footer's commit not cut to fit | TestBlameFooter (second round) |
| R10 | a line not committed shown as a commit in the footer | TestBlameFooter |
| R11 | the commit before a toast | TestBlameFooter |
| R12 | the commit in place of a field's hints | TestBlameFooter |
| R13 | FileCodeWidth with the column's cells | TestBlameWidth |
| R14 | the row's code as wide as without the column | TestBlameWidth (second round) |
| A1 | an answer for a file since replaced kept | TestBlameAnswers (second round) |
| A2 | an answer for a read since replaced kept | TestBlameAnswers |
| A3 | an answer kept after the viewer closed | TestBlameRead (second round) |
| A4 | an answer kept with the blame off | TestBlameRead (second round) |
| A5 | `b`'s second read not compared with the lines shown | TestBlameMessages (second round) |
| A6 | git's lines and the viewer's not compared in number | TestBlameMessages |
| A7 | a file shown blamed without the bytes read | TestBlameSwitch |
| A8 | `b`'s read blamed without the bytes read | TestBlameSwitch |
| A9 | the next file shown not blamed | TestBlameSwitch |
| A10 | a file blamed with the blame off | TestBlameSwitch |
| A11 | the next file read without the blame on | TestBlameAnswers (second round) |
| A12 | `b` not a key of the viewer | TestBlameSwitch |
| A13 | `b`'s read keeps the number of the read before | TestBlameAnswers |
| A14 | the dates in the author's zone | TestBlameSwitch |
| A15 | four initials | TestBlameCommits |
| A16 | the full hash | TestBlameSwitch |
| A17 | a line not committed drawn as a commit | TestBlameSwitch |
| A18 | no git said in git's words | TestBlameMessages |

In the first round 35 were caught by a failing test and A5 by a build
failure, which proves nothing (its first form left `shown` unused); nine
survived: I11, R1, R4, R9, R14, A1, A3, A4 and A11. With the tests above
strengthened and A5 rewritten to compare the lines shown with themselves,
the second round caught the ten, each by a failing test. A third run,
`scratchpad\plan60\mutants3.txt`, broke the shared environment's
optional-lock setting once `TestBlame` checked it: caught, the hash
1fe144b9 before and after.

### The Windows Terminal run

`scratchpad\term\p60run.ps1` on `p51.ps1`: the real application in the acc
tab maximised (208x59), at `-mode 90x30` (120x40 at the acc profile's 9
pt) and at `-mode 60x18` (80x24), with a temporary home, local and roaming
app data and a PATH of System32 and Git; a mock dashboard, so no CLI was
started and no quota used. The project was a demo repository outside this
one (`p60demo.ps1`): `viewer.go` committed by Ana Hubaut (15/11/2022 10:00
+0100), three of its lines reworked by Bo Li (02/03/2023 08:30 -0500), two
by Maximilian Oberhausen (07/06/2024 16:45 +0200), a name longer than the
column's 12 cells; in the working tree line 11 changed and three lines
added after line 22; `notes.txt` untracked; LF line endings.

Each step's shot and text are in `scratchpad\term\out-p60\blame-max`,
`blame-90x30` and `blame-60x18`: e3 the file opened from the Explorer (two
clicks), its hints naming `b blame`; e4 and e5 `b`: "15/11/2022 Ana
Hubaut", "02/03/2023 Bo Li", "07/06/2024 Maximilian …" and "Not committed"
on lines 11 and 24 to 26 at 208x59 and 120x40, "15/11/22 AH",
"uncommitted" at 80x24, the footer "525954b · Ana Hubaut · 15/11/2022
10:00 · Add the viewer"; e6 to e8 `g` to lines 18, 11 and 27: Bo Li's
commit at 14:30 in the machine's zone, "Not committed yet: changed in the
working tree", "9cd6692 · Maximilian Oberhausen · 07/06/2024 16:45 · Name
the footer's parts", cut with "…" at 80x24; e9 full screen; e11 `b` off,
the hints back; e12 and e13 `b` on, the column back; e16 and e17
`notes.txt` opened next, the blame still on: "not in git yet: no history".
The owner's own dashboard, started with `go run` in a terminal with its
Codex session, stayed open: the run started no CLI and read nothing of
theirs. No dashboard or git process of the run was left.

### The timing step

Run alone with the exact command of `doc/evaluation.md`, Keystroke timing,
`TestBlameFrameCost` now among its tests: mains power, the battery at
97 %, the CPU 36 % just before (33, 29 and 45 % over two seconds; the
busiest process over three seconds, Move Mouse, used 0.67 s), the owner's
usual applications open (Claude, Edge, Teams, Outlook, Windows Terminal,
Move Mouse), the only `go` process the owner's idle `go run`. Each of the
eighteen within its target:

| Test | Result | Target |
| - | - | - |
| TestBlameFrameCost | p50 0.53 ms, p95 1.52 ms | p95 ≤ 16 ms |
| TestBranchesFrameCost | p50 0.69 ms, p95 2.00 ms | p95 ≤ 16 ms |
| TestKeystrokeToFrame | p95 1.04 ms; the draft window 1.35 ms | p95 ≤ 16 ms |
| TestDraftKeystrokes | p95 2.47 ms | p95 ≤ 16 ms |
| TestLongSession | p95 2.95 ms | p95 ≤ 16 ms |
| TestThinkingFrame | p95 2.59 ms | p95 ≤ 16 ms |
| TestPasteToFrame | max 1.72 ms | ≤ 100 ms |
| TestStreamToFrame | p95 4.85 ms | p95 ≤ 16 ms |
| TestSelectorToFrame | p95 1.01 to 1.19 ms | p95 ≤ 16 ms |
| TestMotionToFrame | p95 0.99 to 1.51 ms | p95 ≤ 16 ms |
| TestDiffToFrame | p95 1.00 to 1.51 ms | p95 ≤ 16 ms |
| TestMessageToFrame | p95 1.11 and 1.53 ms | p95 ≤ 16 ms |
| TestOutputToFrame | p95 1.01 ms | p95 ≤ 16 ms |
| TestToolsToFrame | p95 2.53 ms | p95 ≤ 16 ms |
| TestBannerToFrame | p95 1.64 ms | p95 ≤ 2 ms |
| TestHomeKeysToFrame | p95 0.15 ms | p95 ≤ 16 ms |
| TestHomePictureTime | max 202.2 ms | ≤ 260 ms |
| TestKeystrokeCost | reports only | none |

**60.8, the owner's visual review (2026-10-28).** The owner reviewed the
review shots (the full column at 208x59 with Maximilian Oberhausen's
commit in the footer, a line not committed at 120x40, the compact form and
full screen at 80x24, an untracked file with the blame still on) and wrote
"Looks right": 60.8's last part, PASS, so 9 criteria, 9 PASS, 0 FAIL. The
blame is kept as the shots show it. Not committed: the commit waits for
the owner's approval. The block above stays as it was reported.

**Approved and committed (2026-10-28).** The owner wrote "Approve and
commit". Commit 0050409 (34 files: 23 modified, 11 added), from the files
reviewed in `git diff`, which `git diff --check` and a byte check found
free of trailing spaces, CRs and BOMs; the whole suite had passed after
the last code edit and after the last edit of a document a test reads. The
author is `Evengelius <molimoabel@gmail.com>`, with no attribution line,
and the working tree was clean after it: PASS. Not pushed: the owner asked
for the commit only. The block above stays as it was reported.

**The push (2026-10-28), at the owner's request.** Commit 0050409 and the
approval commit, 3ed1ad4, were pushed to `origin`: `main` fast-forwarded
from 191117f to 3ed1ad4, nothing forced; `origin/main` equals `main`
(read back by `git ls-remote`), the working tree clean: PASS. The commit
recording this was pushed after it.
