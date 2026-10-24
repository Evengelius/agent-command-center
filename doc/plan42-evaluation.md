# Plan 42 evaluation (2026-10-24)

The check of plan 42, the mouse in zone B (brief amendment 43: beside the
keyboard's path, which stays as it is, a click on a foldable item unfolds
it, a double click on an element opens its full view as Enter does, and
the mouse shows none of the keyboard's hints). It covers the 7 criteria
written in [`plan.md`, Plan 42](../plan.md) before the code; one was
amended in step 5, said where it applies (42.3). Nothing is committed: the
owner reviews the screenshots and the results first.

How it was checked:
- The finding from the code (plan.md, What was found): zone B's two
  foldable items share `ui.Unfolded` and the application's `unfold`; the
  entries Enter shows whole are `render.Targets`, plus an approval from
  its card; the keyboard's hints belong to the entry cursor alone.
- Deterministic tests, none starting a CLI: the render's marks
  (TestConvClickMarks, on P12c-block replayed: each summary row its
  details' key and entry, each row of an entry's content its entry, no
  label, rule, blank row between entries or session-state row marked); the
  application, with mouse messages through the program's update
  (TestThinkingClicks on P12c-block's last phase, TestEntryClicks on a
  conversation with every kind of row, TestFoldRowDoubleClick on a run of
  12 calls); the keyboard's tests unchanged (TestThinkingKeys,
  TestToolFoldKeys); TestFailedToolRow takes the new rule (a double click
  on a call's row opens it; it selected a word of the error since plan
  5). Each was run against the change broken on purpose
  (`scratchpad\plan42\mut42.mjs`), 12 breakages.
- Windows Terminal 1.23 in mock mode, real mouse clicks (`Term.cs`,
  SendInput) on `testdata\mock\claude-thinking-subagent.jsonl` (P303b:
  three thinking phases, an Agent call, the subagent's Write approved by
  the owner, a Glob call, the answer): plan 41's build
  (`scratchpad\term\acc-p41.exe`) and the new tree's
  (`scratchpad\term\acc-p42.exe`), driven by `scratchpad\term\p42demo.ps1`
  (plan 41's harness with click and double click steps), maximised in the
  acc tab's profile (208x59) and at 120x40. The owner's clipboard was
  read before the runs and found unchanged after: plan 41's build got
  single clicks only, its double click copying a word.
- Build, vet, gofmt, the whole suite, the race detector on render and
  app, the timing step alone, all on the final tree; no independent
  review, the criteria being settled by the tests, the mutation checks and
  the screenshots.

```text
Evaluation - Plan 42, the mouse in zone B: a click to fold, a double click to open - 7 criteria: 7 PASS
PASS 42.1 TestThinkingClicks (P12c-block's last phase, "Thought for 2.2s", 160x50): a click on its summary text, on its origin at the row's right end and on its ▸ each unfolds its details ("▾ ✓ Thought for 2.2s", 4 rows more), and the next click folds them; no entry cursor and no "Enter full view", "← fold" nor "Esc back" in the frame, the keyboard's cursor that was on the newest entry gone; B held as with →; TestToolFoldKeys: a run's row still toggles on a click; TestConvClickMarks: each summary row in view carries its details' key and its entry
PASS 42.2 TestEntryClicks: a click on the Agent call, the Write call, the review, the failure, the long message, the answer and the answered approval opens nothing, unfolds nothing, puts no cursor and no hint; TestFailedToolRow/copy and TestMouseSelection: a drag selects and copies as before
PASS 42.3 (amended in step 5, the approval added) TestEntryClicks: a double click on the Agent call, the Write call, the review, the failure and the long message opens the entry's view, frame for frame the one Enter opens from the entry cursor; on the answered approval's row and on a pending approval's card, the approval whole ("◀ back (Esc)"), the pending one still pending; Esc back to B with no cursor and no hint; TestThinkingClicks: a double click on the summary row shows the phase whole ("THINKING"), its details folded as before and B's hold as before; on a detail row, the phase whole, the details left open; TestFailedToolRow/double_click: the failed call's viewer, nothing copied
PASS 42.4 TestEntryClicks: a double click on the answer copies "both", a triple click its line "Found alpha.txt and beta.md; both names are in found.txt.", no view; TestFoldRowDoubleClick: a double click on "▸ 9 earlier tool events · Expand" leaves it "▾ 9 earlier tool events · Fold", no view, nothing copied
PASS 42.5 TestThinkingKeys, TestToolFoldKeys and TestFailedToolRow's cursor and viewer pass unchanged (↑ to the row, → unfolds with the hint, ← folds, Enter opens, Esc returns with the cursor on the entry); the 192 golden frames unchanged
PASS 42.6 Build, vet, gofmt clean; go test ./... ok, 15 packages with tests, 691 tests (4 new); -race on render and app ok, 0 data races; the timing step alone, all eight within target: the long session's delta p95 5.17 ms, the stream's 5.00 ms, the thinking cards' 3.56 ms, the selector's 1.04 to 1.56 ms, the 50,000-line paste 1.93 ms at most (targets 16 ms, 100 ms); 12 breakages, each caught by a test: the summary row not a fold row, no entry marks, labels and rules marked, the marks not recorded, a double click opening any entry, approvals not opened, a single click opening, the view not opened, the first click's fold kept, B's hold not restored, the keyboard's cursor kept, a fold row's double click selecting
PASS 42.7 Windows Terminal, real clicks on claude-thinking-subagent.jsonl, maximised in the acc tab's profile (208x59) and at 120x40: plan 41's build, a click on "Thought for 2.8s" and on the ▸ of "Thought for 1.4s · 281" unfolds nothing; the new build (SHA-256 219aa919a40573d0…), a click far right on "Thought for 2.8s" unfolds its four details under "▾" with no hint, a second click folds them, a click on the ▸ of "Thought for 1.4s · 281" unfolds it; double clicks open the Agent call's view, the Write approval's ("◀ back (Esc)  ✓ ALLOWED  general-purpose · Write", "allowed by you") and the view of "Thought for 1.4s · 268" ("THINKING"); Esc returns to zone B with no row hint each time; the owner's clipboard unchanged
```

Observed and kept:
- A full view keeps its own bottom line of keys ("↑↓ PgUp PgDn scroll ·
  Home top · End bottom · Esc back", grey), however it was opened; the
  keyboard's marks the mouse leaves out are the entry cursor's ▸ in the
  margin and its hint in the attention colour on the row.
- A click on a summary row or a run's row cannot start a drag selection
  there, as on a run's row since plan 19; the summary's words copy by a
  drag started on another row.
- On a double click on a summary row, its details open for the time
  between the two clicks, then the full view covers them and the fold
  goes back.
- On the rows a double click now opens (calls, reviews, failures, long
  messages of the owner's, approvals), it no longer selects a word, which
  plan 5 (item 6) had it do on a failed call's error; a drag still does.
- At 120x40 the harness's window shows the same rows; the run's title
  keeps plan 40's prefix ("ACCP40…"), the harness's.

Not verified:
- A live session: the clicks were checked on recordings and events, the
  mouse handling being the same whatever the CLI; a live turn would need a
  model call.
- A double click on a pending approval in Windows Terminal: checked by
  TestEntryClicks alone, the recording's approval being answered.

**The commit and the push (2026-10-24), after the owner's approval.** The
owner approved plan 42 on 2026-10-24; commit 11f794d (15 files: 12
modified, 3 added) and the approval commit were pushed to `origin`;
`origin/main` equals `main`, the working tree is clean, and the author is
`Evengelius <molimoabel@gmail.com>`, with no attribution line: PASS. The
blocks above stay as they were reported.
