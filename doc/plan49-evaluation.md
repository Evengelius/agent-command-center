# Plan 49 evaluation (2026-10-25)

The check of plan 49, the rest of the audit's phase 0 and its phases 3 and
4 ([`audit.md`](audit.md); brief amendment 50):
- phase 0: the event queue's and the write queue's tests, the overlay
  matrix;
- phase 3: tool rows memoised, the scans of a long session bounded, the
  timing step with a large tool history;
- phase 4: one owner of the overlays, the adapters' shared lifecycle, the
  git tests in parallel, normalised decisions, grapheme widths, a GIF's
  first frame.

It covers the 16 criteria of [`plan.md`, Plan 49](../plan.md), written
before the code. Nothing is committed or pushed: the owner reviews first.
No git command touched this repository; every git test ran on temporary
repositories.

How it was checked:
- **Today's code, first:** a clean copy of HEAD (`git archive`, in the
  scratchpad) ran each new test that states today's behaviour before the
  change: the queue and write tests, the matrix (95 cells hold, 5 are the
  defects D7 fixed), `TestToolsToFrame`, `TestAnswerLines` (the 9 lines
  written), `TestClusterFrames` (26 rows off), the GIF tests (the bytes
  sent, the refusals, 65.8 MiB), and the timing step.
- **The memos against no memo:** every mock file and probe replay drawn
  after each event both ways.
- **Each change broken on purpose** through `go test -overlay`, nothing
  written in the repository (`scratchpad\plan49\mutate49.mjs`).
- **The widths against lipgloss's segmenter:** a scratch program
  (`scratchpad\plan49\clusters`) compared the dashboard's clusters and
  widths with uax29 v2.7.0, the segmenter lipgloss measures with, over
  every code point and 5,000,000 random strings.
- **Fuzzing** of the widths, the image readers, the GIF walk and both
  mappers, 60 s each.
- **Build, vet, gofmt, the whole suite** after the last document edit, the
  race detector on the touched packages, the timing step alone on both
  trees.
- **Windows Terminal 1.23.20211.0** in mock mode, the new build and today's
  (`scratchpad\term\p49.ps1`, `p49call.ps1`; `scratchpad\plan49\mkrepo49.mjs`,
  `mkmock49b.mjs`).

```text
Evaluation - Plan 49, the rest of phase 0 and phases 3 and 4 of the audit - 16 criteria: 11 PASS, 5 FAIL
PASS 49.1 pipe/queue_test.go (8 producers x 1,000 events; the holds; the close; a hold across the close; 100,000 events unread) and TestSessionEventsInOrder (2,048 deltas through the listen, then the end) pass on the clean copy of today's code and at the end; the mutants of Forward past a hold ("b" left) and of the dropped events numbered (Seq 6 read) fail them, and three more queue mutants too
PASS 49.2 writer_test.go (c's cancel then interrupt; the quit's order then its close; the full queue refused at once and the 65 in order; a failure and a panic stopping nothing; the close at once when full) passes on today's code and at the end; each write in its own goroutine fails TestWriteOrder ("written while the CLI holds the answer: [interrupt]"), and the full queue waiting and the close queued behind it fail theirs
PASS 49.3 TestLayerMatrix: 100 cells, the ten layers alone and the 90 ordered pairs, at 200x60 with Source Control docked, a frame after every message; on today's code exactly the five recorded cells fail (dropdown>worktrees, question>worktrees, question>keys, question>models, question>modes); at the end all 100 pass
PASS 49.4 TestMemoInputs: 30 Write calls with 100 KB inputs drawn again with nothing changed read no input (the mutant keeping none reads 33); a call's new input and its new status show in the next frame; the replaced input is no longer held two frames later
PASS 49.5 TestMemoChangesNoFrame: the 13 mock files and 35 probe recordings, 5,545 events, each drawn at 200x60 with the memo and without (every third and the last also unfolded, an agent inspected, and at 80x24): the same frames; the 220 golden frames under testdata/frames unchanged
PASS 49.6 TestMemoIndex: a call added to the long session's built index visits that call alone (the mutant building it again visits 279,801); TestMemoThinking: Animating and ActivityOn visit the thinking entries only; TestTreeRowsLazy: no row's line made before zone A shows it, the 240 turns folded, then unfolded with their 4,800 call rows and 720 agent rows
FAIL 49.7 The timing step alone, today's code then this tree, on mains power (CPU 41% then 20% busy just before): every test within its target; TestToolsToFrame p95 42.19 ms on today's code (over 16 ms, p50 25.3, max 81.6), 2.40 ms now (p50 1.00, max 4.08); TestThinkingFrame p95 2.55 ms (3.76 ms today), at most half of the 8.14 ms recorded; TestLongSession's last tenth's p95 0.95 times its first tenth's (2.37 against 2.48 ms). Not met: "the end's p95 at most half of today's, in the same conditions": 3.00 ms against 5.42 ms in the step, and over eight runs taking turns with today's binary 2.28 to 3.52 ms against 4.52 to 11.52 ms, medians 2.62 and 5.06 ms: 52%. None slower than today's beyond the step's noise: TestStreamToFrame p95 4.00 ms (5.03), TestKeystrokeToFrame 1.01 ms (1.05), TestDraftKeystrokes 3.01 ms (3.20), TestOutputToFrame 1.01 ms (1.03), TestMessageToFrame 1.03 and 1.62 ms (1.65 and 2.00), the others within 0.5 ms of today's, 800 of 800 idle motion reports dropped in each case
FAIL 49.8 Draw, the keys (key, dashboardKey by the top layer), the mouse (Screen.Layers.Top), motionFilter and the mouse mode read render.Layers; the other reads of the popups' state are each layer's own handlers, no precedence; the matrix passes with its defects fixed; the golden frames unchanged; the key and mouse suites pass. Not met: "the key and mouse suites are unchanged": one line of draftwin_test.go calls key instead of dashboardKey, which now takes the top layer; what it expects is unchanged
FAIL 49.9 Both sessions run on pipe.Proc (the adapters 464 lines fewer, 61 more), its tests pass (the exit the last event; the readers given up 2 s after the exit with the CLI's name in the message; Close's terminate then kill at its steps; a write's hold released or dropped); the probe replays give the same events and the messages are word for word today's. Not met: "every adapter test passes unchanged": TestReaderPanic in each adapter reached into the reader the change moved, and now drives it through Proc with the same assertions; D10's type changed four more adapter test files (below)
FAIL 49.10 17 git tests and the subtests of five run in parallel, git's environment set once in TestMain (a temporary global configuration, no system one, a test identity); every repository in the tests' temporary folders. -count=3 passes (275 s), -race passes (164 s). Not met: "within 60 s": alone 79.1 s; back to back with today's code on a busier machine (CPU 38% and 45% busy before), 188.6 s today (its FAIL is plan 49's TestGIFMemory, which today's code fails by design) and 114.2 s now
FAIL 49.11 No CLI word for an answer is left in internal/state, internal/app or internal/render (the auto-review entry's own status, "inProgress" or "approved", stays as Codex reports it: the reviewer's words, shown as written); TestAnswerLines, run first on today's code: its 9 lines (5 Claude Code, 4 Codex) are written byte for byte; the golden frames unchanged; the 4 Codex mock files differ in their decisions only; FuzzMapperOut, 60 s each with -fuzzminimizetime 2s, 599,717 Claude Code scripts and 535,253 Codex scripts without a crash. Not met: "the approval tests pass unchanged": they pass, but 15 test files changed the type of a decision and the word an approval's answer is stored as (Codex's accept is now allow); no written line, status or frame they expect changed
PASS 49.12 TestClusterWidths: combining marks, VS15 and VS16, skin tones, ZWJ sequences, flags (an odd third indicator too), a tag sequence, keycaps, Hangul jamo and syllables, Indic conjuncts, CJK, a prepended mark, CR LF, with ASCII around them or not: each text cut into its clusters, textWidth equal to lipgloss.Width, cut and tailCut keeping the most whole clusters at every width; TestClusterUnits and TestClusterCells: the hit-testing's units, the cells and the wrap by cluster; FuzzTextWidth, 60 s with -fuzzminimizetime 2s: 986,605 inputs, none where textWidth differs from lipgloss.Width or a cut splits a cluster; TestClusterFrames: a message, a tool row, a file and a commit message holding them at 120x40 and 200x60, every row exactly the screen's width by lipgloss.Width, where today's code has 26 rows off; the golden frames unchanged; the keystroke and stream timing tests within their targets (49.7). Against uax29 v2.7.0 itself: every code point's properties and width, then 5,000,000 random strings, no cluster split and no width different
PASS 49.13 TestGIFSent, run first on today's code: the single frame's file as it is and four animations' first frame as PNG, the same length and SHA-256 each; TestGIFMemory: a 200-frame 640x480 GIF inspected with 2.6 MiB allocated (65.8 MiB on today's code); TestGIFRefused: cut after the first frame, no trailer, an unknown block, an unknown extension, a graphic control's size or terminator, a later frame's bounds, code size or data cut, each refused in today's words; a later frame's damaged pixels, refused before, now taken; 60 s each with -fuzzminimizetime 2s: FuzzInspectData 169,750 inputs without a crash, and the new FuzzGIFFrames 768,547 inputs, none that DecodeAll reads and the walk refuses or counts otherwise
PASS 49.14 Build, vet and gofmt clean; go test ./... ok, 16 packages, 849 tests (814 and the 35 new, 5 of them skipped as before), run again after the last document edit; the 220 golden frames and the probe recordings unchanged, the 4 Codex mock files changed by D10 only (their decisions); -race ok on input (164 s), app (406 s), state (19 s), render (245 s), the five provider packages and pipe, no data race
PASS 49.15 Mutations through go test -overlay: 35 of 35 caught, phase 0's eight among them: the memo keeping no input or dropping nothing, the index built again, a parent's change ignored (twice), the thinking entries found again, a tool row made early; the key overlay over a popup, a popup not shown left open, an opener leaving the popups open; the end before the readers, Close never killing, a failed write's events released; Codex's allow for the session as accept, Claude Code's cancel as an allow, Codex's decline read as an allow, a cancel resolved as expired; marks never joining, every regional indicator pairing, ASCII before a mark counted alone, cut and the cells and the units by code point; an unknown block passed, every GIF one frame, the file's end taken as its trailer, a frame's bounds unchecked. The first run caught 31: the two layer mutants are caught by the whole app suite (TestSelectorNoRoom, TestCommitWorkflowB), and two test gaps were closed (TestTreeRowsLazy unfolded, TestClusterCells), then both caught. No mutant code left in the tree
PASS 49.16 Windows Terminal 1.23.20211.0, maximised in the acc profile (208x59) and 120x40, mock mode on a temporary repository, the new build and today's: the ten layers opened in turn (the key overlay, both selectors, the draft window, the Worktrees view, the "+" menu, the commit dropdown, the staging question, the quick open and the file viewer on a file of clusters), and the five former defects' pairs: the frames are the same text for text but where the defects were, which today's build shows in the terminal too (the staging question back after the Worktrees view and after the key overlay; with a selector opened over it, the next click closing the hidden question and leaving the selector open) and the new build does not; at 120x40 with Source Control docked the selectors do not open over the question ("the model selector needs the model pill on screen"); a reply holding the VS16 heart, the skin tone, the family and the flag: zone B's border straight on every row (today's build breaks it on each of those rows); the 200-frame GIF attached through "Upload from computer": its first frame in the preview and the toast "anim.gif: animated: the first frame is sent" (the chip "anim.gif · 6 KB"). No terminal-hosted agent session was open; no test window left; the clipboard the same before and after
```

Changed from the design:
- **Three criteria said tests would pass "unchanged"** (49.8, 49.9, 49.11),
  and some had to change how they call what changed: one call of the draft
  window's test (the key routing now takes the top layer), each adapter's
  `TestReaderPanic` (the reader it drove moved into `pipe.Proc`), and 15
  test files for the decision's type and the word an approval's answer is
  stored as. No expected line, status, event or frame changed; the
  criteria are reported FAIL on that word, as written.
- **D5's `AgentRev` became `ParentRev`:** only a change of a known
  agent's parent moves it, so the index takes new agents and calls in
  place instead of rebuilding (the first version rebuilt on every new
  agent: 8% of a frame).
- **D11 found two more things to follow lipgloss on:** the soft hyphen and
  the C1 controls take no cell (today's `runeWidth` gave them one), and
  Go 1.27's tables are Unicode 17, with 20 linkers for GB9c, not 6. A
  code point's facts are kept in one table read without a lock, so a row
  of box-drawing characters costs less than today's lock per character.
- **D9's target was not reached** (49.10): 17 git tests and the subtests
  of five run in parallel, but the parallel phase is bound by how many git
  processes Windows starts, not by the tests' order. Splitting the two
  longest tests' cases into parallel subtests made the package slower
  (100 s against 79 s), so they were left as they were.

Observed and kept:
- **Windows Terminal 1.23 and lipgloss disagree on one kind of cluster:**
  the terminal draws a Devanagari conjunct (क्ष) two cells wide, lipgloss
  and Bubble Tea's renderer count one. Every other kind tested agrees: the
  VS16 heart, a skin tone, ZWJ families and flags, regional flags, marks, a
  tag sequence, jamo and syllables, CJK, VS15. The dashboard follows its
  renderer, which places the cells.
- **The terminal draws a keycap as a digit and a box** (its font has no
  glyph for the sequence): the row's border stays straight.
- **The old build's dropdown under the Worktrees view** (a defect of the
  matrix) did not come back in the terminal after Esc there; the staging
  question's defect did (back after the key overlay).
- **The old build decodes the 200-frame GIF visibly later:** 1.5 s after
  the dialog closed the new build's toast was up, the old build's not yet.
- **Bubble Tea's cell buffer measures what lipgloss measures:** with
  lipgloss as its oracle, `TestClusterFrames` fails on today's code for the
  same clusters whose rows the terminal draws broken.
- **Today's build's 120x40 run of the first pass stopped** at the second
  click on the "+" band, whose row was not found after that build's
  question steps (it is one Esc behind there): the frames before it were
  compared.

Not verified:
- **A live CLI's text with clusters:** the conversation, the tool rows, a
  file and a commit message were tested with mock replays and scripted
  events.
- **Other terminals' cluster widths** (only Windows Terminal 1.23 here).
- **The memo's frames with a live session:** compared with no memo over
  every mock file and probe recording.

Limitations:
- A row holding a Devanagari conjunct is a cell off in Windows Terminal
  1.23 (above); text only, its characters kept.
- Where UAX #29 starts a cluster after a joiner or a linker before a code
  point that is not a pictograph or a consonant, the dashboard keeps them
  together (5% of the random strings of the check, none in real text
  tested): a cut there keeps both or neither, the width right.
- The auto-review entry keeps Codex's status words, the reviewer's report
  shown as written.
- The git tests take 79 to 114 s here, over the 60 s target.
- The fuzzers ran 60 s each: a sample of their inputs, not a proof.
