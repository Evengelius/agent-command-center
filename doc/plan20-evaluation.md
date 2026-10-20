# Plan 20 evaluation (2026-10-20)

The final check of plan 20, strict conformance to the owner's two approved
templates (brief amendment 16). It checks the plan's five items in
[`plan.md`, Plan 20](../plan.md#plan-20---strict-conformance-to-the-approved-templates)
against the yes/no criteria written there before the code, and those added
after the first screenshots were compared with the templates (2.6, 3.11,
3.12, 4.4 to 4.6, marked "added" in the plan). The CLI's real events decide
what is shown, the templates how.

The evidence is the tests, the golden frames, and real Windows Terminal
screenshots of the owner's four scenarios, each compared side by side with
its template, and read as text through UI Automation for the geometry. The
block below is the summary reported to the owner. The results are not to be
rewritten.

```text
Evaluation - Plan 20, strict conformance to the approved templates - 32 criteria: 32 PASS
PASS 1.1 At 120x40, 156x44, 160x45 and 200x60: A and B on row 3, their bottoms on one row, A 26, 34, 35, 44 columns, B from the next column to the edge (TestDashboardGeometry)
PASS 1.2 C under A and B from the first column to the last, D under C, the footer last, no row wider than the screen (TestDashboardGeometry; the geometry script on 23 dashboard frames)
PASS 1.3 80x24: the A row, then B, C and D full width, C 3 rows (TestDashboardGeometry; TestAdaptiveLayout: B 14, C 3)
PASS 2.1 No agent in play: C 3 rows at 120x40, 156x44, 200x60, 100x24 and 80x24, "no agents yet" and "0 ACTIVE · 0 DONE" (TestAdaptiveLayout)
PASS 2.2 120x40: one agent C 4, six 14, twenty 16 (the upper row 17); 200x60: one agent 8 with the preview inside C, twenty and forty 25; 80x24: one row an agent, six agents C 5 (TestAdaptiveLayout)
PASS 2.3 A completed agent keeps its rows (C 4); the owner's next prompt: C 3; an agent of the new turn: in play again; a turn the CLI opens: unchanged (TestAdaptiveLayout, TestAgentsInPlay); on screen, scenarios C and D
PASS 2.4 From P3b's mock at the Write request, at 156x44 and 120x40: "●  general-purpose", "haiku-5-5", "waiting: approval", then "00:10   21.2k tok   tool: Bash ✓   next: Write (approval pending)", "1 ACTIVE · 0 DONE" (TestAgentCardTemplate2)
PASS 2.5 C focused: the footer reads C's keys, no key hint row in C, the agents out of view counted in its top border (TestAdaptiveLayout, TestPaneScroll); the scroll and zoom tests pass
PASS 2.6 The dot #F472B6, the state and "next: …" #A1A9C0, two cells after the dot and three after the longest label; at 80x24 "● general-purpose … waiting: approval" (TestAgentCardTemplate2; TestAgentModelColumn, TestAgentSourceAndEffort, TestAgentLabelColumn, TestSyntheticScenarios, TestSpawnRequestLabel updated)
PASS 3.1 YOU: ▍ and label #A78BFA, the prompt #F9FAFB, the card ground #13162A on every cell; the prompt beside the label from 90 cells (at column 14), under it at 76 (TestConversationLook 3.1-3.4, 3.1 under 90 cells)
PASS 3.2 "▍ CLAUDE  Haiku 5.5  <time>" once per exchange, the bar on every row of the CLI's part; "▍ CODEX  gpt-5.5" (TestConversationLook 3.1-3.4, 3.2)
PASS 3.3 A blank row between two cards; inside one, only the card's own rows between its parts (TestConversationLook 3.1-3.4)
PASS 3.4 Tool rows: the tool ground #1E2133 from column 2 to one cell before the edge, the name #93C5FD bold, the target #A1A9C0, the outcomes and times right-aligned, "allowed by you" only on an answered approval, the agent's identity on its requests (TestConversationLook 3.1-3.4, 3.4)
PASS 3.5 "▸ 9 earlier tool events · 1 failed · Expand", expanded "▾ … · Fold"; ↑, Enter, ← and a click as in plan 19 (TestToolFold, TestToolFoldKeys)
PASS 3.6 The pink box's corners and edges; "! APPROVAL · orchestrator · Write" with its time and origin, the target on its own row, "▸ Preview of found.txt (2 lines)", " 1 │ alpha.txt", " 2 │ beta.md", "▐ F2 to answer ▌"; with the focus its keys; no green in it (TestConversationLook 3.6, 3.7; TestApprovalLineKeepsOutcome)
PASS 3.7 Answered: allowed #4ADE80, denied #F87171, expired grey, one row each (TestConversationLook 3.6, 3.7)
PASS 3.8 The RESULT box over the last message of a completed exchange; none while it runs, after a failed or an interrupted turn, for a lone answer, or while the exchange's agent runs; once the agent completed and the CLI's turn ended, on its message; the agent failed: none (TestConversationLook 3.8)
PASS 3.9 "∴ thinking" while a thinking block streams without text; nothing once it ended (TestConversationLook 3.9)
PASS 3.10 The rule, one input row, the status row, for an empty and a one-line draft; three input rows for three lines; at 80x24, 120x40 and 200x60 (TestCompactEditor)
PASS 3.11 From 90 cells the preview in its own grey box; Write's sentence with and without content, Edit's with and without replace_all; none for Bash, mcp__fs__write or Codex's fileChange; at 97 and 160 cells (TestConversationLook 3.6, TestApprovalEffect)
PASS 3.12 The card's rows, ▍ on the card ground, between "I'll read them." and the calls and between the calls and "I read both." (TestConversationLook 3.1-3.4, rows 4 and 7)
PASS 4.1 38 golden frames rewritten; by script, item 1's geometry in 23 dashboard frames (the 24th, the 80x24 tree frame, is F6's full tree, by design), zone D's rows identical to HEAD's in the 15 dashboard frames that have them, and the window's rectangle in the 15 window frames identical to HEAD's
PASS 4.2 go build ./... and go vet ./... exit 0, gofmt -l . empty; go test ./... 353 PASS, 1 SKIP (TestDiskListLink), 0 FAIL in 14 packages; the timing step 4/4 PASS: keystroke-to-frame p95 1.01 ms (10,000 lines), 1.03 ms (draft window), 2.07 ms (50,000 lines); long session delta-to-frame p95 5.52 ms at 200x60 (plan 19: 7.0 ms)
PASS 4.3 ui-design.md (Dashboard, Layouts with the renderer's frames at three sizes, Zone B, Zone C, Approvals, States, Colours and symbols, Scrolling, Input editor, Departures), events-and-state.md (Conversation record, a Turn row), architecture.md (ASM-12), reference-design.md and AGENTS.md say what was built; brief.md holds amendment 16
PASS 4.4 Eight exchanges of seven calls: "▸ 4 earlier tool events · 1 failed · Expand", the deep path ending "\f76.go"; PgUp to "Exchange 0" and PgDn back; a drag over "Reading the files of exchange 7." copies exactly that; at 80x24, 120x40, 200x60, 100x30 and 156x44 the geometry holds and the newest answer is in view (TestLongConversation)
PASS 4.5 A conversation.user event in an idle session after the CLI's turn: Submitted, not by the CLI, the notice cleared (TestStateMachines)
PASS 4.6 7 backgrounds; #A1A9C0 to 145, the card ground #13162A to 234 (a background), the pill's ground #3B1531 to 53; no two foregrounds and no two backgrounds on one index (TestPalette256); the 16-colour palette holds (TestPalette16)
PASS 5.1 Scenario A, 156x44, beside Template 1: the conversation's start and its end; each difference fixed or listed below as a deviation; the geometry read from the screen
PASS 5.2 Scenario B, 156x44, beside Template 2: the pending Write in its pink box, the preview in its grey box, the sentence, the pill; C's two rows; no RESULT; each difference fixed or listed below
PASS 5.3 Scenario C, 156x44: the agent completed, C still two rows for it, "0 ACTIVE · 1 DONE", RESULT over the CLI turn's answer
PASS 5.4 Scenario D, 156x44: after "Reply with only the word ok.", C 3 rows, "0 ACTIVE · 0 DONE"; TURN 1 unfolded in A shows the earlier agent; the status row "Enter send · Ctrl+J newline"
PASS 5.5 Read from the screen: scenarios A and B at 120x40 and 80x24 and a 208x59 grid, item 1's geometry in each, no row wider than the screen; at 26 columns zone A's longest rows end in "…"
Not verified: a 200x60 window on screen: this monitor holds 157x44 at the default font, so a 208x59 grid (Ctrl+minus three times, Windows Terminal's own per-window action, no setting changed) was read instead, and 200x60 itself is checked by the golden frames and TestDashboardGeometry. No live run: plan 20 changes how the session is drawn, not what is sent to the CLI; the scenes are the recorded P3b session in mock mode, with synthetic prompts after it
Verdict: DONE
```

## The owner's conformance review

The review of the owner's section 14, requirement by requirement. Each
visual verdict comes from the screenshots themselves, compared side by side
with the templates, not from a component existing.

| Requirement | Verdict | Evidence | Deviations |
| - | - | - | - |
| A/B upper-row geometry | PASS | 1.1, 1.3; every screenshot's geometry read from the screen (5.5) | none; at 120 columns A's longest rows end in "…" |
| Full-width Zone C | PASS | 1.2; scenarios A to D | none |
| Adaptive Zone C lifecycle | PASS | 2.1 to 2.3; scenarios C (kept after the agent completed) and D (compact after the next prompt) | none |
| Zone C history preservation | PASS | 2.3; scenario D: the earlier agent under TURN 1 in A, inspectable | none |
| Zone D placement | PASS | 1.2; its rows identical to HEAD's in the golden frames (4.1) | none |
| Zone B header | PASS | "B · CONVERSATION" at the left, the session's state at the right ("Idle"; the badges of the turn machine, unchanged) | none |
| User-message hierarchy | PASS | 3.1; scenarios A and B | the card is a bar and a ground, not a thin border (terminal rows); the prompt beside "YOU" from 90 cells, under it below |
| Assistant-message hierarchy | PASS | 3.2, 3.12; scenarios A and B | as above, no thin border |
| Tool styling and grouping | PASS | 3.4, 3.12; scenarios A and B | Template 1's group rather than Template 2's boxed rows (the templates differ); ✓ for Template 1's wrench; a failed call's error in place of its path |
| Collapsed tool history | PASS | 3.5, 4.4; scenario A | the owner's section 4.6 wording, the real counts, not Template 1's invented "+9 more Read calls (f12.go … f21.go)" |
| Pending approval styling | PASS | 3.6, 3.11; scenario B at 156x44, 120x40, 80x24 | the path on its own row (section 5.2); a one-row pill for the outlined button |
| Approval-state correctness | PASS | 3.6, 3.7; "allowed by you" only on answered approvals (scenario A's Read calls show their times, no approval) | none |
| Result styling | PASS | 3.8; scenarios A and C | none |
| Result semantic correctness | PASS | 3.8; scenario B: "I've launched the subagent…" stays in the CLI's card | none |
| Input/footer presentation | PASS | 3.10, 2.5; every screenshot | the status row kept under the input (Template 2's) where Template 1 has none |
| Responsive terminal behavior | PASS | 4.4, 5.5 | 200x60 on screen not reachable on this monitor: 208x59 read instead |
| Visual screenshot comparison | PASS | 5.1 to 5.4, side by side | those of this table, each a terminal limit, a difference between the two templates, or a real event |
| Functional regression tests | PASS | 4.1, 4.2, 4.5 | none |
| Performance regression tests | PASS | 4.2: the timing step 4/4, p95 well under 16 ms | none |

The deviations are listed with their reasons in
[ui-design.md, Dashboard](ui-design.md#dashboard) ("Against the owner's two
templates") and in the plan.

## What the screenshots changed

The first screenshots, of scenarios A and B maximised, matched the templates'
geometry and cards. Compared side by side with them, these differences were
actionable and were fixed, then the screenshots taken again:

- **The preview's own box.** Template 2 draws the content preview in a grey
  box inside the pink one; it now has it from 90 cells inside B (narrower,
  the rows go straight in the card, for the card to fit B at 80x24).
- **Template 2's sentence.** "This will create or overwrite the file at …
  with the above content." was first left out as not in the request. It is
  now drawn from the request's own tool and target, for the two tools whose
  effect Claude Code documents (Write, Edit; its tools reference, read on
  2026-10-20), and for no other tool or CLI: a fact of the tool, not an
  invented event.
- **Separation inside the card.** The owner's section 7.3 asks for room
  between narration and tool groups; the card now has one of its own rows
  between them, and before a request's box.
- **Zone C's row.** Template 2 shows a pink dot, the state in grey and wider
  gaps; C had "!" in pink, a pink state and single spaces.
- **A stale status row.** Scenario A's end read "turn started by the CLI"
  after two later prompts: in mock mode a prompt arrives as an event, which
  opened the turn without clearing the CLI turn's notice, as sending does.
  It now does (`reduce.go`; a new row of the Turn machine).

## What the build found

- **A verdict pushed off its row.** TestClassifierDenialInZoneB (plan 13)
  failed on the first build: a refused call's long error, after its name,
  pushed "denied by auto-review" off the row. The error is now cut to the
  room the outcome leaves.
- **Earlier tests on the old look.** The tests of plans 5 to 19 that read
  rows were updated to the cards, their intent kept: the cursor in a card's
  margin (TestEntryViewer), the label column at the new widths
  (TestSpawnRequestLabel, TestSyntheticScenarios), the approval heading of
  section 5.2 (TestApprovalLineKeepsOutcome), the preview inside C
  (TestAgentLabelColumn: a 40-cell label is cut beside the preview from 160
  columns, whole at 120 and in the preview's heading).
- **The palette test's terminal.** TestPalette256 renders one row per style
  in a 60x40 terminal; with the plan's new styles the list passed 40 rows,
  so its terminal is now 60x80.

## The terminal runs

`p20shot.ps1`, on the plan 17 harness: a Windows Terminal window driven by
UI Automation, a temporary project, store and home, a PATH of System32 only,
so no agent CLI is found or started; the screen read as text, then
captured. The scenes are built from the Claude mock (the recorded P3b
session) by `mkscenes.js`: A, the session with each approval held 4 s,
then twelve Read calls of the orchestrator, the fourth failed, an answer,
and "Reply with only the word ok."; B, the session up to the subagent's
Write request; C, the session whole; D, the session and then "Reply with
only the word ok.". What follows the recorded session is synthetic, for the
eyes.

| Scene | Size | Checks | Result |
| - | - | - | - |
| A | 156x44 | the end, and PgUp to the start; beside Template 1 | geometry ok, as compared above |
| B | 156x44 | the pending Write; beside Template 2 | geometry ok, as compared above |
| C | 156x44 | the agent completed, C two rows, "0 ACTIVE · 1 DONE", RESULT | geometry ok |
| D | 156x44 | F6 F6 ↑ ↑ ↑ →: TURN 1 unfolded, the earlier agent; C 3 rows | geometry ok |
| A | 120x40, 80x24 | the end; at 80x24 labels over their text | geometry ok |
| B | 120x40, 80x24 | the whole pending box; C's dot at 80x24 | geometry ok |
| B | 208x59 | Ctrl+minus ×3: the wide layout, the preview inside C, D 3 rows | geometry ok |

Each run: the window closed, no process left, no agent CLI started, nothing
written in the project. A first run of B asked for 200x60 and got the
monitor's 157x44; that capture is kept as it was. The screenshots are kept
for the owner's review outside the repository.

## The owner's live review

After this evaluation the owner ran the scenarios live, on Claude Code
2.1.303 with Opus 5.5, and sent the screenshots one by one; each correction
is recorded here, the block above unchanged.

- **Scenario 1** (four screenshots: the prompt typed, the subagent's Write
  waiting, its card focused, a second run completed). Drawn as designed:
  the pink box with the preview in its own box, the sentence and the pill;
  "! Waiting for approval" in B's header; C's pink dot, grey state and
  "next: Write (approval pending)"; in the completed run the answered
  PowerShell request, RESULT once the agent and the turn completed, and C
  kept at "0 ACTIVE · 1 DONE". One correction: zone D read "context N/A (no
  turn yet)" through the first turn, while a turn ran; the dashboard reads
  Claude Code's context after each turn end, so what is missing then is the
  turn's result, and D now reads "N/A (no result yet)" once a turn has
  started (TestTelemetryNA, "claude context while the first turn runs").
  Not the dashboard's: the second run found `found.txt` from the first (the
  folder was not reset), so its subagent wrote nothing; the answer's Px
  reminder comes from the owner's global CLAUDE.md, which every Claude Code
  session reads.
- **Scenario 2** (five screenshots: twelve Read calls, the fourth on a
  missing file; the run folded, expanded with the cursor on a call, then
  zone A's turn folded and unfolded). Drawn as designed: "▸ 9 earlier tool
  events · 1 failed · Expand", the three last calls, RESULT, the expanded
  run, the entry footer, zone A's fold. Three corrections:
  - the failed row named no file, only "✗ Read · File does not exist.
    Note: …": a failed call now shows its target, then its error, which
    keeps half the room ("✗ Read  …\Local\Temp\acc-demo\src\f03.go · File
    does not exist. …   failed 12:15:19"), as Template 1's failed row
    names its file;
  - on the entry cursor's row, where the hint takes room, the path was
    cut inside an element, "…rs\…\AppData\…": paths in tool rows are now
    cut at an element, "…\AppData\Local\Temp\acc-demo\src\f00.go";
  - zone A's twelve Read rows all read "C:\User…": a path there keeps its
    file name, "✓ Read  …\f00.go".

  Checked by TestFailedRowNamesItsTarget (156x44, 120x40, 200x60, 80x24)
  and TestElidePath; TestSyntheticScenarios now expects the failed Codex
  command's row with its command, cut, then "· exit 1 · fatal: No
  configured push destination."; three golden frames rewritten, their
  paths cut at an element ("…\p3b\found.txt", "…\probe-work\p2\hello.txt");
  go test ./... 0 FAIL. Not the dashboard's: the markers of the answer's
  Markdown ("**Read:**") are drawn as received, by design (ui-design.md,
  Markdown); the Px reminder again.
- **Before scenario 3**, the owner's request: → on B's entry cursor
  opened the entry's full view, as Enter does, which read as opening a
  file while → unfolds elsewhere. → now only expands a folded run (zone A's
  meaning, unfold) and opens nothing; Enter alone opens the full view. The
  footers read "Enter full view"; the keys page lists Enter, → and ←
  apart. TestEntryViewer checks → leaves the view closed and the cursor
  where it was; TestToolFoldKeys checks → on a call opens nothing and → on
  the run's row expands it, ← folding it again. TestEntryCursor and
  TestKeyOverlayRows updated to the footer and the keys page; six file
  viewer golden frames rewritten, only their last row, the dashboard's
  footer under the window; go test ./... 0 FAIL.
- **After scenario 3**, the owner's amendment 17, with an image of the
  target: zone B's messages read as a two-column log, their bodies beside
  their labels, on tinted bands. Each message is now a block: a narrow bar
  `▎` of its colour on each of its rows; its header on its own row ("YOU
  14:29", "CLAUDE  Opus 5.5  14:29", the model and the time muted); its
  body under the header from the same column, at every width; no ground
  behind it, the card ground removed. A thin rule of `─` in the frames'
  colour across B's inner width separates two messages, a blank row on each
  side: before each of the owner's messages but the first, and between the
  owner's message and the CLI's part; never inside a message (narration,
  tool groups, a request's box and the RESULT box keep their own spacing),
  never at the start, never before the input's rule, never two together; a
  blank row opens the conversation and one ends it. Causes: the frame drew
  the card ground under every cell of a message (`onCard`), the labels
  shared their row with the body from 90 cells (`besideFrom`, `labelCol`),
  and the cards' only boundary was a blank row (`look.gap`).

  Checked by TestConversationLook: 3.1 to 3.4 (the 29 rows of a two
  exchange conversation, the bars' and labels' colours, the rule's, no
  ground behind a message, the tool rows' ground), "3.1 the same layout at
  every width" (60, 76, 100 and 160 cells; a long prompt wrapping at the
  content column) and "3.3 the rules between messages" (two prompts in a
  row, a CLI turn of its own: two rules, each with a blank row on either
  side, none at the start or end, never two together); TestToolFold 2.5
  (the row-count cache, with the new look bits); the tests of plans 5 to 20
  that read B's rows updated to the blocks, their intent kept; 38 golden
  frames rewritten, item 1's geometry in each dashboard frame and every
  window's rectangle unchanged; go test ./... 0 FAIL; the timing step 4/4
  (long session delta-to-frame p95 5.49 ms). Windows Terminal, maximised:
  the minimal conversation (and at 80x24), the longer one from its start
  and at its end, the pending approval; each compared with the owner's
  image and the templates, the geometry read from the screen. What the
  character grid cannot draw: a timestamp smaller than its label (it is
  grey instead) and space inside a row (the image's line spacing is a blank
  row around the rule).
- **Scenarios 1, 2 and 4 in one session**, on the new layout (eight
  screenshots: the pending Write and its focused keys, scenario 1 done with
  RESULT and C kept, scenario 2 done with C cleared, its run expanded on
  the failed row, scrolled to the first prompt, the narrow tree and the
  narrow layout): drawn as designed; the failed row names
  `…\src\f03.go`, zone A's Read rows their file names, → on a call opens
  nothing, a selection copied "I could read 11 of the 12 files: …" without
  the bars. One correction: scrolled to the first prompt, the window ended
  on the rule before scenario 2's CLAUDE header, which sat one blank row
  above the input's rule as a second line (the owner's section 2.5, "two
  consecutive horizontal rules"). A rule at the window's edge, with nothing
  of the message beyond it in view, is now a blank row; its row stays, so
  nothing moves while scrolling (TestConversationLook, "3.3 no rule at the
  window's edges": the window ending after a rule, starting with one,
  showing it with both messages); no golden frame changed; go test ./...
  0 FAIL.
