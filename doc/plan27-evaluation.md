# Plan 27 evaluation (2026-10-22)

The check of plan 27, zone C with the execution timeline (brief amendment
26, the owner's image `references/04-zone-c-timeline.png`: "improve zone C
to this exac design. Respect this design strictly"). It covers the criteria
written in [`plan.md`, Plan 27](../plan.md) before the code. Nothing is
committed: the owner reviews the Windows Terminal screenshots first.

How it was checked:
- Deterministic tests in `internal/render` (`zonec_test.go`:
  TestZoneCDesign, TestCHeight; `layout_test.go`, `zonecfloor_test.go`,
  `model_test.go`, `retries_test.go`, `zonea_test.go` adapted) and
  `internal/app` (`scenarios_test.go`, `zoom_test.go`,
  `longconv_test.go`, `zonea_test.go` adapted), on the probe recordings and
  the mock files.
- Every golden frame regenerated (`-update`) and reviewed by script
  (`scratchpad\plan24\framecheck27.cjs`): 46 changed; 36 dashboards hold
  zone C's header row inside its frame, the panels' headings and the
  agent's status or, with no agent, the 4-row empty C; 5 frames with a
  window over the dashboard (draft, explorer, file, file finder, file
  search at 200x60) read by eye, the window covering the panels' headings
  and the header row visible at the left; 5 frames without zone C (the same
  views at 120x40, the window over C); the old preview's texts searched in
  all 46.
- Windows Terminal, the executable built from this tree
  (`scratchpad\term\acc-p27.exe`), the plan 24 harness
  (`scratchpad\term\p27demo.ps1`): each scene in a window of its own with
  a temporary home and no agent CLI on the PATH, maximised at 9 pt
  (208x59) or at 120x40, the rows read through UI Automation at each
  screenshot; the quota mock (the agent completed) at both sizes, the
  thinking-subagent mock (a four-entry timeline) at 208x59, and a copy of
  the P3b mock whose Write approval stays pending 20 s longer
  (`scratchpad\plan24\mkwait.mjs`) for the agent waiting at 208x59. The
  screenshots are in `scratchpad\term\out-p27\` (`quota-max`,
  `quota-120x40`, `thinking-max`, `waiting-max`).
- Build, vet, gofmt, the whole suite, the race detector on the render and
  app packages, the timing step alone (`scratchpad\plan24\p27-checks.log`).

```text
Evaluation - Plan 27, zone C with the execution timeline - 9 criteria: 8 PASS, 1 FAIL
PASS 27.1 TestZoneCDesign on P3b at its Write and on the quota mock, at 120x40 and 208x59: no title in C's top edge; row 1 "C · LIVE AGENTS" ending with "1 ACTIVE · 0 DONE" (P3b) or "0 ACTIVE · 1 DONE" (quota); row 2 a rule; row 3 "AGENTS" then " │ " at the list's width then "EXECUTION TIMELINE" ending with "● WAITING" (P3b) or "✓ COMPLETED" (quota); " │ " on every row from 3 to the last; the list 45 cells at 120 columns and 80 at 208 (39% of the inner width, rounded)
PASS 27.2 TestAgentCardTemplate2 at 208x59 and 156x44: "●  general-purpose", "haiku-5-5", "waiting: approval", "│ tool: Bash" with no symbol after the completed call, "next: Write" ("next: Write (approval pending)" whole at 208), the dot in stAttention, the state in stMuted, "Bash" in stBold; TestZoneCDesign: a running agent's status "◉ RUNNING"
PASS 27.3 TestZoneCDesign: the quota mock's entries "<HH:MM:SS>  <role> │ <text>" the newest last, rules between them and none after the last; on a hand-made agent a 400-cell notice wraps onto two rows, the second "{20 spaces}│ x x … x…", a rule, then "11:30:01  assistant │ short second line"; "notice    " in stSpeaker, "assistant " in stResult; every row 60 cells; held at the top the heading's right end reads "↓ 2 · End follows"; an agent with no entry reads "no activity yet" (P3b's subagent at its Write request has none)
PASS 27.4 TestCHeight: 1 agent gives 16 at region 56 and 57, 13 at 41, 12 at 37, 7 at 21 to 23, 8 at 24; 6 agents 17 at 37, 20 agents 18 at 37, 28 at 57, 9 at 21 (B keeping 12); none 4; the narrow layout 3, 3 and 5; TestAdaptiveLayout: 120x40 one agent A and B 25 rows and C 12, six agents 20 and 17, twenty 19 and 18, none 33 and 4; 200x60 one agent 41 and 16, twenty 29 and 28, none 53 and 4; 156x44 one agent 28 and 13; 100x24 twenty agents 12 and 9; the 80x24 cases unchanged (17 and 3, 15 and 5); TestZoneCFloor at 160x24 to 160x27: C 7, 7, 7 and 8 with the agent's label left of the rule and "EXECUTION TIMELINE" right of it; TestDashboardGeometry at the five sizes: A and B on row 2 to one row, C full width under them to the footer, three boxes
PASS 27.5 TestZoom (A and C zoomed at 80x24, 120x40 and 200x60, C's heading on its first inner row from 100 columns and in the edge under it), the select, pane and key tests of internal/app, TestLongConversation (its box helper naming the untitled C) all pass; the wheel scrolls panel P (sc.Preview holds the timeline's rows)
FAIL 27.6 expected no frame with "· LIVE", "(reported)" or "no activity yet" beside a list, found "no activity yet" in 2 of 46 (codex-thought at 120x40 and 200x60): the timeline of the spawned Codex agent /root/list_files, running with no entry yet, the text 27.3 specifies for that state (the old preview showed it for the same agent at 200x60). The rest holds: 46 frames regenerated and read (36 by script, the 5 with a window over the panels' headings and the 5 without zone C by eye), 0 with "(reported)", "(not reported)" or "-purpose · LIVE", every dashboard frame's C with the header row and the panels, the narrow ones (80x24) with their titled edge
PASS 27.7 go build ./..., go vet ./... and gofmt -l . clean, nothing printed; go test ./... 14 packages ok (internal/app 39.9 s, internal/render 6.6 s); the race run on internal/render (60.5 s) and internal/app (147.9 s) ok, no race; the timing step alone 7 of 7: TestKeystrokeToFrame p95 1.02 ms (1.04 ms in the draft window), TestDraftKeystrokes 2.12 ms, TestKeystrokeCost 216 µs and 503 µs a key, TestLongSession delta-to-frame p95 5.99 ms after 26,209 events, TestPasteToFrame p50 1.17 ms, TestStreamToFrame p95 4.00 ms, TestThinkingFrame p95 3.55 ms, every target p95 ≤ 16 ms
PASS 27.8 Windows Terminal: quota-max C-completed-4s (14.9 s), quota-120x40 C-completed (14.7 s), thinking-max C-entries-3s (18.6 s), waiting-max C-waiting (13.6 s, "approval pending" found): the conformance table below, 17 rows PASS, 1 documented deviation (the label not bold, the model in its family's colour), 1 row not a design element (the viewer's controls); the owner reviews the screenshots before any commit
PASS 27.9 ui-design.md (Zone C rewritten; the Layouts section's two frames redrawn by TestDocFrames from P3b at its Write request and their three bullets; the mock-up table's row), reference-design.md (the image's section and its table row), AGENTS.md (status), brief.md (amendment 26), plan.md (D1 to D6, 27.1 to 27.9, progress); every changed text file read byte by byte: no BOM, no CR; git status lists the intended files only
Not verified: the owner's review of the screenshots, the plan's gate before any commit; no live run: the change is rendering only, driven by the recorded probes and the mocks
Verdict: NOT DONE (27.6, the owner's decision)
Likely cause: 27.6's clause, written before the code against the old preview's three texts, while 27.3, written with it, keeps "no activity yet" for the new timeline's empty state; the two frames show that state, the dashboard behaving as designed
Proposed next step: correct 27.6 with a note, to "no frame holds the old preview's heading ("<label> · LIVE") or its origin labels ("(reported)", "(not reported)")"; or, if the owner prefers the image's silence, leave the timeline blank for an agent without entry (one line in zonec.go, 27.3 and ui-design.md adjusted); no new run - waiting for approval
```

**Conformance with the image** (`references/04-zone-c-timeline.png`), read
against `thinking-max\C-entries-3s.png` (the four-entry timeline),
`waiting-max\C-waiting.png` and `quota-max\C-completed-4s.png`:

| The image | The dashboard | Result |
| - | - | - |
| A rounded dark card with the header row inside it | zone C's box with its rounded corners, no title in the top edge, the header on its first inner row | PASS |
| "C · LIVE AGENTS" in white, bold, at the left | row 1, white bold (stBold) | PASS |
| "0 ACTIVE · 1 DONE" in grey at the right end | row 1's right end, grey; "1 ACTIVE · 0 DONE" with the agent waiting | PASS |
| A thin rule under the header | row 2, a rule across the inner width | PASS |
| Two panels, the left about two fifths of the width | the list 39% of the inner width (80 of 204 cells at 208 columns), the timeline the rest | PASS |
| A thin vertical line between the panels from the headings to the bottom | "│" in the frame's colour on every row from row 3 to the last | PASS |
| "AGENTS" in light blue, bold | the title colour, bold | PASS |
| The agent row: a green "✓", the label in white bold, the model in grey, "…" for the rest | a green "✓" (a pink "●" while waiting), the label in white, the model in its family's colour, the state cut with "…" where the panel ends | deviation: the label is white but not bold, and the model keeps the colour every zone gives a model family (plan 7), rather than grey |
| The second row "00:41   30.1k tok", a thin bar, "tool:" and "Write" in white bold | "00:09   21.5k tok  │ tool: Write", the time and tokens grey, the bar in the frame's colour, "Write" in white bold, no symbol after a completed call | PASS |
| "EXECUTION TIMELINE" in light blue, bold | the title colour, bold | PASS |
| "✓ COMPLETED" in green at the heading's right end, dynamic | the selected agent's status: "✓ COMPLETED" green, "● WAITING" pink, "◉ RUNNING" amber, from the state | PASS |
| Each entry: the time in grey, the role "notice" in blue or "assistant" in green | "18:48:49  notice" (zone B's speaker blue), "18:48:53  assistant" (its result green) | PASS |
| A grey vertical bar before the text, on both of its rows | "│" in the frame's colour on both rows | PASS |
| The text wrapped onto two rows at most, the second ending in "…" when it goes on | wrapTwo: "Do the following in the current working directory, and nothing else: 1. List the names of the / files in the current working directory (top level only, not recursive). Use Glob with the pattern…" | PASS |
| Thin rules between the entries, none after the last | a rule across the panel between entries, none after the last | PASS |
| Four entries | the pane sized for four entries at 208x59 (16 rows); the thinking-subagent mock shows four | PASS |
| No preview heading with the label and model, no origin label on the rows, no key hint row | none; the model's source and the effort are the inspection's | PASS |
| The agent's "…" after the model when the row is cut | at 120x40 (45 cells) "general-purpo…   haiku-5-5   completed · …"; at 208 the state shows whole | PASS |
| The "Modifier" button and the share icon at the card's bottom | the image viewer's controls, not the design: nothing drawn | not a design element |

**What differs from the image, and why.**
- The label is white but not bold, and the model is in its family's colour
  (haiku-5-5 blue, gpt-6-sol its own), not grey: every zone colours a model
  by its family since plan 5 (ui-design.md, Colours and symbols; plan 7's
  short name), and the image's grey model would make zone C the one place
  without that cue. A line to change if the owner prefers the image's grey.
- The panel headings are in the dashboard's title colour (lavender-blue),
  the image's light blue being its own rendering of that colour; the image
  has no colour the palette lacks.
- An agent with no entry yet reads "no activity yet" in grey under
  "EXECUTION TIMELINE" (27.3): the image shows no empty timeline, and the
  dashboard shows a missing value rather than a blank (AGENTS.md, Keep
  failures visible). 27.6's clause disagrees; the owner decides.

**What it costs the other zones.** C's rows are fixed by the design's four
entries: 16 at 208x59 (8 before), 12 at 120x40 (4 before). Zone A's
telemetry pane at 120x40 takes its compact form (no blank rows between its
sections and windows) and the tree keeps its 6 rows; at 208x59 A keeps the
full form with 19 tree rows. Tests that assumed the old 4-row C
(TestZoneAStructure's fold check, TestCountdownRedraw, TestClockRedraw)
read zone A at 160x60 or 208x59 now, and those that assumed a 70-cell list
(the label column, the retries row, the spawn label) read the list's width.

**Evidence outside the repository**, in the session's scratchpad
(`C:\Users\<user>\AppData\Local\Temp\claude\C--javadev-projets-agent-command-center\dcc3e64d-0d35-4470-ab9a-d258f8675bae\scratchpad\`):
`term\out-p27\` (the screenshots, their UI Automation rows and `log.txt`),
`term\p27demo.ps1`, `term\acc-p27.exe`, `plan24\p27-checks.log` (vet, the
suite, the race run, the timing step), `plan24\framecheck27.cjs`,
`plan24\docframes\` (the frames TestDocFrames wrote for ui-design.md),
`plan24\mkwait.mjs` and `p27-wait-mock.jsonl`, the patch files
`p27-*.patch`.

## Second pass: the owner's correction (2026-10-22, 19:06 to 19:46)

The owner ran the dashboard on Claude Code after the first pass and found
the design not respected (brief amendment 27: "the separation "|" is not
respected either", "on the left, there is too much space"), sending two
close-ups of it (`references/05-zone-c-timeline-close.png`,
`references/06-zone-c-timeline-close.webp`). Plan 27's second pass
(plan.md, Plan 27, Second pass: design D7 to D9, criteria 27.10 to 27.16,
written before the code) redraws the lines in the title colour, the card
as two columns with a grey bar on both rows and nothing else, the selected
agent's rows on zone B's tool rows' ground, the panel as wide as its
cards, the model grey. The first block above stands as it was; 27.6's FAIL
and its proposed correction still wait for the owner.

How it was checked:
- Deterministic tests: `internal/render` (`zonec_test.go`: TestZoneCDesign
  with 27.10's checks, TestCardColumns new; `model_test.go`,
  `layout_test.go` adapted) and `internal/app` (`approval_test.go`,
  `scenarios_test.go` adapted); TestColourProfiles caught the first
  wording's "▸" for 16 colours, which the criterion now records.
- Every golden frame regenerated and reviewed by script
  (`scratchpad\plan24\framecheck27b.cjs`): each dashboard frame from 100
  columns with an agent has the header row, the panels' line on every row
  from the headings, each card's bar on both rows at the same cell, "tool:
  " on each second row and no state text on the cards; the 10 frames with
  a window over the dashboard reported, not judged.
- Windows Terminal, the executable built from this tree
  (`scratchpad\term\acc-p27b.exe`), the plan 24 harness as
  `scratchpad\term\p27bdemo.ps1`: each scene in a window of its own with a
  temporary home and no agent CLI on the PATH, in mock mode (nothing of
  the owner's read or written, no quota), maximised at 9 pt (208x59) or at
  120x40; the quota mock at both sizes, the thinking-subagent mock and the
  waiting copy of P3b at 208x59. The screenshots are in
  `scratchpad\term\out-p27b\`. The owner's own Windows Terminal window,
  with the dashboard session of 19:06 still open, was left as it was: the
  harness finds its window by its own title, and a mock run touches
  nothing the owner's session uses.
- Build, vet, gofmt, the whole suite, the race detector on the render and
  app packages, then the timing step alone (`scratchpad\plan24\p27b-checks.log`).

```text
Evaluation - Plan 27, second pass: the owner's correction - 7 criteria: 7 PASS, 0 FAIL
PASS 27.10 TestZoneCDesign on P3b at its Write and the quota mock, at 120x40 and 208x59: "C · LIVE AGENTS" in stTitle; row 2's rule and the " │ " at the panel's edge on every row from 3 in stPanelRule; the panel 40 cells at both sizes; the card's rows " ●  general-purpose     │  haiku-5-5" (the quota mock's "✓") and "    00:06   21.2k tok   │  tool: Write" (the clock and the tokens by the mock), the bars on cell 24 of both; the label in stBold, the model, the bar and "tool: " in stGrey, "Write" in stBold, every one tinted on the ground; no "waiting", "completed", "running", "next:", "approval", "Files" or "Bash" on the rows; every span between the margins on the ground and none outside; two hand-made agents, the later one selected: its rows on the ground, the other's not, "tool: none yet" without a call; on the 16-colour palette the same characters, no ground
PASS 27.11 TestAgentLabelColumn: the panel 45, 61 and 74 cells at 120, 160 and 200 columns, the label column 17, 33 and 40, the tool column 16, 16 and 22, every model at the same cell, the 40-cell label whole at 200 only, the 18-cell one at 160 and 200, "tool: abcdefghi…" at 120 and 160 and "tool: abcdefghijklmno…" at 200, the long label whole in the inspection, the 80x24 row cut as before; TestCardColumns: P3b's card 40 at inner widths 96 to 204, the Codex card 52 at 208 columns and 45 at 120; TestAgentModelColumn: the model grey after the bar from 100 columns, in its family's colour at 80; TestAgentSourceAndEffort: "gpt-6-mini*" grey, no "high" or "low" on the cards, the sources and efforts in the inspection; TestListModelAcrossWidths from 160 to 220 columns; TestAgentRetries: "↻ 2" after the time; TestAgentCardTemplate2: "│  tool: Write" with no "Bash", "waiting" or "next:" at 208x59 and 156x44, "● general-purpose … waiting: approval" at 80x24; the app's TestApprovalCards ("│  tool: Write", "● WAITING"), TestSyntheticScenarios ("/root/list_files     │  gpt-6-mini*", "/root/create_found   │  model N/A") and TestSpawnRequestLabel ("✓  task: Count the …   │  gpt-5.6-luna*" at 120x40, the label whole at 200x60) pass
PASS 27.12 TestCHeight, TestAdaptiveLayout, TestZoneCFloor and TestDashboardGeometry unchanged and passing: C 16 rows with one agent at 208x59 and 200x60, 13 at 156x44, 12 at 120x40, 7 at 160x24 to 160x26, 8 at 160x27
PASS 27.13 46 golden frames regenerated; framecheck27b.cjs: 36 ok (each with the header inside, the panels' line at one cell on every row, 1 or 2 cards with their bars on both rows, "tool: " on each second row, no state text), 0 bad, 10 other (5 views over the dashboard without zone C, 5 with a window over the panels' headings); TestDocFrames redrew ui-design.md's two Layouts frames (C 12 and 16 rows, the panel 40 cells)
PASS 27.14 go build ./..., go vet ./... and gofmt -l . clean; go test ./... 14 packages ok; the race run on internal/render and internal/app ok, no race; the timing step alone 7 of 7: TestKeystrokeToFrame p95 1.07 ms (1.51 ms in the draft window), TestDraftKeystrokes 2.52 ms, TestKeystrokeCost 221 µs and 513 µs a key, TestLongSession delta-to-frame p95 4.58 ms after 26,209 events, TestPasteToFrame p50 1.02 ms, TestStreamToFrame p95 3.73 ms, TestThinkingFrame p95 3.68 ms, every target p95 ≤ 16 ms
PASS 27.15 Windows Terminal: quota-max C-completed (15.0 s, "1 DONE" found) and C-completed-later (20.1 s), quota-120x40 C-completed (14.5 s), thinking-max C-entries (18.6 s) and C-entries-later (22.1 s), waiting-max C-waiting (13.6 s, "WAITING" found): the conformance table below against the close-ups, 18 rows PASS, 0 deviations; the owner reviews the screenshots before any commit
PASS 27.16 ui-design.md (Zone C rewritten for D7 to D9; the Layouts frames and their two zone C bullets), reference-design.md (the close-ups' row and section), brief.md (amendment 27), AGENTS.md (status), plan.md (the second pass, its criteria, 27.10 corrected with a note, its progress); every changed text file read byte by byte: no BOM, no CR; git status lists the intended files only
Not verified: the owner's review of the screenshots, the plan's gate before any commit; no live run of the second pass (rendering only; the owner's own run of 19:06 to 19:08 showed the first pass)
Verdict: DONE for the second pass; the first block's 27.6 waits for the owner's decision
```

**Conformance with the close-ups** (`references/05-zone-c-timeline-close.png`,
`06-zone-c-timeline-close.webp`), read against
`out-p27b\quota-max\C-completed-later.png` (the completed agent),
`out-p27b\waiting-max\C-waiting.png` (the agent waiting for its Write)
and `out-p27b\thinking-max\C-entries-later.png` (four entries):

| The close-ups | The dashboard | Result |
| - | - | - |
| "C · LIVE AGENTS" in lavender, bold, at the left | row 1 in the title colour, bold (stTitle) | PASS |
| "0 ACTIVE · 1 DONE" in grey at the right end | row 1's right end in the palette's grey; "1 ACTIVE · 0 DONE" with the agent waiting | PASS |
| A thin lavender rule under the header | row 2 in the title colour, not bold (stPanelRule) | PASS |
| A lavender vertical line between the panels, from the headings to the bottom | " │ " in stPanelRule on every row from row 3 | PASS |
| "AGENTS" and "EXECUTION TIMELINE" in lavender, bold | the title colour, bold | PASS |
| The left panel as wide as its card, about two fifths of a 108-column card | 40 cells for a Claude Code subagent at every width, the timeline taking the rest (161 cells at 208 columns) | PASS |
| The card on a slightly lighter band spanning the panel but for a margin cell on each side | the selected agent's two rows on zone B's tool rows' ground from the cell after the margin to the cell before the panel's end | PASS |
| "✓" green, then "general-purpose" in white bold | a green "✓" (a pink "●" while waiting), the label in stBold | PASS |
| A grey vertical bar between the columns on both rows | "│" in stGrey on cell 24 of both rows | PASS |
| "fable-5-1" in grey, right of the bar | "haiku-5-5" in stGrey (the mock's model) | PASS |
| "00:41   30.1k tok" in grey under the label | "00:09   21.5k tok" in stGrey under the label | PASS |
| "tool:" in grey, "Write" in white bold, right of the bar | "tool: " in stGrey, "Write" in stBold; the waiting agent's Write likewise | PASS |
| Nothing else on the card: no state word, no task text | nothing else; the state at the timeline heading's right end | PASS |
| "✓ COMPLETED" green at the heading's right end | "✓ COMPLETED" green; "● WAITING" pink | PASS |
| Each entry: the time grey, "notice" blue bold or "assistant" green bold, a grey bar on both rows, the text white | "19:44:15  notice" (stSpeaker), "19:44:19  assistant" (stResult), "│" in stGrey on both rows, the text in stText | PASS |
| Thin rules between the entries, none after the last | rules in the frame's colour between entries, none after the last | PASS |
| Four entries, two rows each | four entries at 208x59 (the thinking-subagent mock) | PASS |
| "…" ending a second row that goes on | "Include subdirectory names if any…" | PASS |

**Nuances, not deviations.** The counts' and the model's grey is the
palette's (`#6B7280`), a shade darker than the close-ups' light grey; the
timeline of an agent with no entry reads "no activity yet" in grey, which
the close-ups never show (27.6, the owner's decision); on 16 colours the
band has no ground and the selected card no mark, as zone B's tool rows
there.

**What changed from the first pass, in code.** `internal/render/zonec.go`
holds the card (`agentCard`, `cardColumns`, `currentTools`, `cardClock`,
`cardModel`) and the lines; `dashboard.go`'s old card is gone;
`palette.go` adds `stPanelRule`. The panel's width comes from the cards,
so a 208-column screen gives the timeline 161 cells where the first pass
gave it 121.

**Evidence outside the repository**, in the session's scratchpad:
`term\out-p27b\` (the screenshots, their UI Automation rows and `log.txt`),
`term\p27bdemo.ps1`, `term\acc-p27b.exe`, `plan24\p27b-checks.log`,
`plan24\framecheck27b.cjs`, `plan24\docframes\` (the redrawn frames),
`plan24\zonec-section2.md` and `uidesign27b.cjs` (the document's
rewrite), the patch files `p27-second-pass.patch`, `p27-sixteen.patch`,
`agents27b.patch`.

**27.6, corrected at the owner's decision (2026-10-22).** The owner
approved plan 27 with the proposed correction: "correct 27.6 with a note,
the timeline keeps "no activity yet" for an agent without entry". The
criterion in plan.md now reads: no frame holds the old preview's heading
("<label> · LIVE") or its origin labels ("(reported)", "(not reported)"),
and the timeline keeps "no activity yet" for an agent without entry, as
27.3 asks; the two codex-thought frames show that state. The first block
stays as it was reported, FAIL by its first wording; no code changed.

**The commit and the push (2026-10-22), after the owner's approval.** The
owner approved plan 27 on 2026-10-22, 27.6 corrected with a note; commit
041e7b8 (70 files: 63 modified, 46 of them golden frames, 7 added) and
the approval commit were pushed to `origin`; `origin/main` equals `main`,
the working tree is clean, and the author is `Evengelius
<molimoabel@gmail.com>`, with no attribution line: PASS. The blocks above
stay as they were reported.
