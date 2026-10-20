# Current Plan

The active plan with its progress, the owner's standing decisions, and one
entry per closed plan. Each plan comes from `doc/brief.md` and its
amendments, ends with the evaluation summary of `doc/evaluation.md`, and
waits for the owner's approval before the next one starts. This file keeps
the current state only (the owner, 2026-10-18): a closed plan's full text,
criteria and progress log are in git (`git show 8888f59:plan.md` for plans 1
to 13, `git show 84d2ad8:plan.md` for plan 14, `git show b047434:plan.md`
for plan 15, `git show becfb34:plan.md` for plan 16, `git show f00c2d8:plan.md`
for plan 17, `git show 8b3b16b:plan.md` for plan 18), and its results in its
evaluation file.

## The owner's standing decisions

Decisions of closed plans that still apply; `AGENTS.md` and
`doc/evaluation.md` hold the others.

- **Live runs.** Each one is proposed first with its scope, model, expected
  usage and stopping conditions, and runs only once approved (plan 3,
  2026-10-16); Claude runs use Haiku as a per-process override.
- **The configuration check is hash-only**: credential contents are never
  read, and a file that cannot be classified safely is "changed, not
  classified", never reported unchanged (2026-10-16). No parser runs over a
  file that may hold credentials (the mixed-state tokenizer was withdrawn,
  2026-10-16); the one exception is confcheck's key-level comparison of
  `~/.claude.json`, sensitive keys skipped (plan 5).
- **Measured, not assumed.** Performance figures are targets until a test
  measures them; a documented design is never presented as verified runtime
  behaviour (2026-10-16).
- **The integration.** If driving a CLI without a terminal falls short, the
  limit is documented and the smallest adjustment proposed; the architecture
  never changes silently (2026-10-16).
- **Drafts and session data** survive cancellation, crashes and recovery; no
  conversation data is stored beyond the draft, and nothing sensitive goes
  into a log (2026-10-16).

## Recurring

Tasks tagged **[Recurring]** repeat on an outside event: they are never
closed, never listed as remaining, and each occurrence is reported in the
summary format (the owner, 2026-10-18).

- **[Recurring] After a CLI update.** At session start, `claude --version`
  and `codex --version` are compared with the "Last probed" line of
  `doc/research/probes/README.md`; a newer one is told to the owner with the
  re-check proposed (`doc/evaluation.md`, After a CLI update), which runs
  only once the owner approves it. The dashboard's project bar shows
  "newer than tested" meanwhile. Last occurrence: Claude Code 2.1.303 on
  2026-10-19, approved, every check PASS (P303a to P303d in
  `doc/research/probes/README.md`); it found that plan 14's shape check
  counted an interrupted tool use as unrecognised, fixed the same day at the
  owner's request (P303e).

## Nice to have

Items tagged **[Nice to have]** are future improvements: none is built, and
none is listed as remaining, until the owner asks for one in a new plan
(the owner, 2026-10-18). Amendment 4 names ten "future improvements"
(`doc/brief.md`, Amendment 4, "For this phase, do NOT implement"). Five
remain below. Recent projects, favourites and a project browser were plan 15
(amendment 9); git worktree management was plan 16 (amendment 10). The
automatic VS Code integration was replaced by the owner with a file viewer in
the terminal, plan 17 (amendment 11).

- **[Nice to have]** Advanced model configuration.
- **[Nice to have]** Cross-provider agent collaboration.
- **[Nice to have]** Provider switching during an active session.
- **[Nice to have]** Complex session profiles.
- **[Nice to have]** Advanced orchestration configuration.
- **[Nice to have]** Google Antigravity as an agent CLI, removed from the
  MVP by amendment 5; the provider interface stays open to it.
- **[Nice to have]** Starting it like an app (the owner, 2026-10-19): a built
  `agent-command-center.exe` started from a Start-menu shortcut, pinnable to
  the taskbar, whose target is `wt.exe -d <projects folder> <path to the
  .exe>`, so that it always opens in Windows Terminal (the only supported
  terminal, amendments 6 and 7) with the project field at that folder; the
  `.exe` is rebuilt after each code change.

## Plan 20 - Strict conformance to the approved templates

**Done and approved by the owner on 2026-10-20**, after the live review of
amendment 17 ("Yes you can commit, and push, I approve"): every criterion
PASS ([`doc/plan20-evaluation.md`](doc/plan20-evaluation.md)), committed
together with plan 19.

The owner's request of 2026-10-20, brief amendment 16, with its two
templates: Template 1, a completed multi-turn conversation, and Template 2,
a subagent's Write waiting for approval. They are the target design, to be
implemented as one design system: a new dashboard geometry (A and B in one
upper row, C full width under both), a full-width adaptive zone C, and zone
B's conversation as the templates draw it. Built on plan 19's logic (one
speaker label per exchange, RESULT, the folds, `Entry.Stage`), whose look
it replaces. Precedence, the owner's: the CLI's real events decide what is
shown, the templates how; nothing is invented to match a picture.

The design, in terminal cells (each choice checked against both templates):

- **Geometry, from 100 columns.** Title and project bar; then A beside B,
  the same height, their bottom borders on the same row; then C at the full
  width; then D at the full width; then the key footer. A takes 22% of the
  width, rounded, at least 24 columns: 26 at 120, 34 at 156, 35 at 160, 44
  at 200 (the templates draw about 20 to 22%). Under 100 columns the
  existing fallback stays: A as one row under the project bar (F6 opens its
  tree), B, C and D at the full width.
- **Zone C.** No agent in play: 3 rows, its title and counts in the top
  border, "no agents yet: they appear as the CLI starts them". With agents
  in play, two rows each (Template 2): the status, label, model and state
  ("waiting: approval", "running", "completed · …", "failed · …"), then the
  elapsed time (01:11), tokens, the last tool and its status ("tool: Bash
  ✓"), and an approval waiting for the owner ("next: Write (approval
  pending)"); one row each under 100 columns. As tall as its agents need, at
  most half the rows under the title, B keeping 12 rows (14 from 160
  columns); from 160 columns, with agents in play, at least 8 rows, the
  selected agent's preview inside C, right of a 70-cell list and a rule.
  Template 2's spacing: a pink "●" for an agent waiting for the owner (zone
  A keeps its "!"), two cells after it, three after the longest label, the
  state and "next: …" in grey. Plan 18's lifecycle is kept: a new prompt
  clears C; finishing agents, CLI turns and idle never do. No key hint row
  in C: the footer shows C's keys while C has the focus, and C's top border
  how many agents are out of view.
- **Zone B, messages.** As built first: each message group a card, a bar
  `▍` and a card ground (`#13162A`), labels beside their text from 90
  cells. Since the owner's amendment 17 and image (2026-10-20): each
  message is a block, a narrow bar `▎` of its colour on each of its rows,
  on no ground; its header on its own row, YOU (violet `#A78BFA`) "YOU
  14:29", CLAUDE or CODEX (blue `#60A5FA`) once per exchange "CLAUDE  Opus
  5.5  14:29", the model and the time muted; its body under the header from
  the same column, at every width; the prompt's text `#F9FAFB`. A thin
  rule in the frames' colour across B's width between two messages, a
  blank row on each side: before each of the owner's messages but the
  first, and between the owner's message and the CLI's part; a blank row
  under B's border and one above the input's rule, no rule at the start,
  none before the input, never two together. Inside the CLI's part, a row
  of the block, its bar, separates narration, a group of tool rows and a
  request's box (the owner's section 7.3); a blank row before the RESULT
  box.
- **Tool rows.** Inside the CLAUDE card, from column 2 on the tool ground
  (`#1E2133`): the status symbol; the agent ("general-purpose · ") on a
  subagent's request; the tool's name in light blue, bold; its target in a
  readable muted text (`#A1A9C0`, not plan 19's dim grey, which the owner
  found too faint); at the right, the outcome and its time ("allowed by
  you 01:54:35" green only for an answered approval, "denied by you …"
  red, "failed 01:54:59" red, "approved by auto-review" green, a completed
  call its end time in grey, a running one "running 4s" amber, one waiting
  for the owner "waiting for you" pink) and the origin ("mock",
  "observed"). An agent's launch reads "Agent  general-purpose · List
  directory and write found.txt". A failed call shows its target, then its
  error in red, the error keeping half the room (since the owner's live
  review; first, its error in the target's place).
- **Folds.** As plan 19, the row now "▸ 9 earlier tool events · 1 failed ·
  Expand", the triangle blue; expanded "▾ … · Fold".
- **A pending approval.** A pink rounded box inside the CLAUDE card
  (Template 2), in the structure of the owner's section 5.2: "! APPROVAL ·
  general-purpose · Write" with its time and the origin at the right, the
  target on its own row; "▸ Preview of found.txt (2 lines)" and the lines
  numbered in a gutter on an inset ground (a diff for an edit, the command
  for a shell, at most 6 lines then "… N more lines"), in its own grey box
  from 90 cells inside B (Template 2), straight in the card below; the
  request's own description when it says more than the target; for Claude
  Code's Write and Edit, Template 2's sentence, "This will create or
  overwrite the file at `<path>` with the above content.", from the
  request's own tool and target and the CLI's documented behaviour (its
  tools reference, code.claude.com/docs/en/tools-reference.md, read on
  2026-10-20: Write "creates a new file or overwrites an existing one with
  the full content provided", Edit "performs exact string replacement"),
  none for another tool or for Codex; then a pink "F2 to answer" pill, or
  the card's keys once F2 gives it the focus. Never folded, never green.
  Answered, it becomes a tool row with its outcome, as above.
- **RESULT.** A green rounded box (Template 1) holding "RESULT  01:54" and
  the answer beside it from 90 cells, else under it. Plan 19's rule, with
  the owner's correction (6.1): the exchange's last message, once no turn
  runs, its last turn completed, and every agent started in it completed;
  an agent still running, failed or stopped leaves it neutral, in the
  CLAUDE card. Never from the text.
- **Thinking.** "∴ thinking" only while a thinking block streams with no
  text yet; a finished one is not drawn (the record keeps it, for the
  thinking display the owner plans separately).
- **Editor.** The rule, one input row (it grows with the draft), the status
  row: no empty rows under the prompt.
- **Unchanged:** zone D, the footer's place, every key, the selection and
  its copy, the draft and its window, the inspection, the folds' keys and
  click.

Deviations from the templates, for terminal reasons or the real events:

- Template 1 outlines every card with thin borders between the rows; a
  terminal border takes a whole row, so YOU and CLAUDE cards are a ground
  with a bar and no border rows; RESULT and a pending approval keep full
  borders, the two emphasized states.
- Template 2's sentence under the preview is not in Claude Code's request
  (it carries the tool, its input, a description such as "found.txt", and
  its suggestions): it is drawn for the tools whose effect the CLI
  documents, Write and Edit, from the request's tool and target; another
  tool, and Codex, get none, nothing invented. The owner's example "File
  creation or overwrite requires approval." gives way to Template 2's
  sentence, which names the target.
- Template 2 draws the path on the heading row; section 5.2's structure,
  the path on its own row, is followed: a long path would push the time off
  the row.
- Template 2's outlined "F2 to answer" button: a one-row pill, pink on a
  dark pink ground with half-block ends; an outline takes three rows.
- Template 1's wrench on approved rows and circled "!" on a failed one: ✓
  and ✗, single-cell glyphs of the terminal's font.
- Template 1's failed row shows its file and "failed": a failed call shows
  its file too, then its error (plan 5, item 6), "failed" and the time at
  the right.
- Template 2's zone A lists the agent's tool history ("tool Bash ✓
  allowed"): zone A's content is unchanged (the owner's section 8).
- The templates' "allowed by you" on plain Read calls, "f00.go failed" and
  "+9 more Read calls (f12.go … f21.go)" are not what the replay holds; the
  rows show the real outcomes and counts.
- Template 2's "▶ subagent" and "▶ tool" kinds: the row names the real tool
  ("Agent", "Bash"), the agent's identity in its place.
- Template 1's input row carries "1 line · Ctrl+G draft window" at its
  right and no status row; the status row stays, Template 2's layout, as it
  carries the editor's messages ("turn running: the draft waits", quit
  confirmations).
- Template 2's turns "TURN 2 ○ pending" to "TURN 4" do not exist in the
  session; zone A shows the real stages.

### Criteria

Item 1, geometry (TestDashboardGeometry, `internal/render`).

- 1.1 At 120x40, 156x44, 160x45 and 200x60: A's and B's top borders on row
  3 and their bottom borders on the same row; A 26, 34, 35, 44 columns
  wide, B the rest, A's right border next to B's left one.
- 1.2 C's top border on the row under A's and B's bottom, from column 1 to
  the last; D's under C's, full width; the footer last; no row wider than
  the screen.
- 1.3 At 80x24: the A row, then B, C, D full width; C compact 3 rows.

Item 2, zone C (TestAdaptiveLayout, `internal/render`; TestPaneScroll).

- 2.1 No agent in play: C 3 rows at every size, "no agents yet" and "0
  ACTIVE · 0 DONE".
- 2.2 One agent at 120x40: C 4 rows, its two rows; six: 14; twenty: half
  the rows under the title; at 200x60 with one agent, 8 rows and the
  preview; at 80x24, one row per agent, at most 5 rows.
- 2.3 Its agents done: the same height; a new prompt: 3 rows, the agents
  still in A's tree and inspectable; agents of the new turn: it grows
  again; a turn the CLI opens: unchanged.
- 2.4 Two rows of the Template 2 agent: "waiting: approval", "tool: Bash
  ✓", "next: Write (approval pending)", from a state built from P3b's
  events.
- 2.5 C focused: the footer reads C's keys; agents out of view: C's top
  border says how many; the existing scroll and zoom tests pass.
- 2.6 (added on 2026-10-20, after comparing the first screenshots with
  Template 2) C's first row in Template 2's spacing, "●  general-purpose
  haiku-5-5   waiting: approval", the dot pink, the state and "next: …"
  grey; under 100 columns the one row with the same dot
  (TestAgentCardTemplate2; the column tests of plans 6, 7 and 10).

Item 3, zone B (TestConversationLook, TestToolFold, `internal/render`).

- 3.1 A YOU card: the violet bar `▍` on each of its rows, the card ground
  on each cell from column 0 to the edge, "YOU" violet bold, the time grey,
  the text `#F9FAFB`; beside the label from 90 cells (text at column 14),
  under it below 90.
- 3.2 A CLAUDE card: "CLAUDE  Haiku 5.5  <time>" once per exchange, the
  blue bar on every row of the exchange's AI part, the narration at
  column 14 (beside) or 2.
- 3.3 A blank row between two cards; none inside one.
- 3.4 Tool rows: the tool ground from column 2 to one cell before the
  card's edge; name `#93C5FD` bold; target `#A1A9C0`; outcomes and times as
  listed above, right-aligned; "allowed by you" only on an answered
  approval; the subagent's identity on its rows.
- 3.5 Folds: "▸ 9 earlier tool events · 1 failed · Expand", the counts in
  their colours; expanded "▾ … · Fold"; plan 19's keys and click
  (TestToolFoldKeys).
- 3.6 A pending approval: the pink box's corners and edges; its header,
  "▸ Preview of found.txt (2 lines)", " 1 │ alpha.txt", " 2 │ beta.md", the
  "F2 to answer" pill; with the focus, its keys; no green in it.
- 3.7 Answered: allowed green, denied red, expired grey, each one row.
- 3.8 RESULT: the green box, "RESULT  <time>" and the answer; over the
  last message of a completed exchange; none while it runs, after a failed
  or interrupted turn, for a lone answer, or while an agent of the exchange
  runs ("I've launched the subagent…" stays narration); after the agent
  completes and the CLI's turn ends, on its message; an agent failed:
  none.
- 3.9 Thinking: "∴ thinking" while a thinking block streams without text;
  none once it ends.
- 3.10 The editor: the rule, one input row, the status row; a 3-line draft,
  three input rows (TestCompactEditor).
- 3.11 (added likewise) A pending Write from 90 cells inside B: its preview
  in its own grey box; "This will create or overwrite the file at `<path>`
  with the above content." for Claude Code's Write, ", leaving it empty."
  without content, the Edit sentence with and without `replace_all`; none
  for Bash, an MCP tool or Codex (TestConversationLook 3.6,
  TestApprovalEffect).
- 3.12 (added likewise) Inside the CLI's card, a row of the card between
  narration and tool rows and before a request's box (TestConversationLook
  3.1 to 3.4).
- 3.13 (added for amendment 17) Each message's header on its own row and
  its body under it from the same column at 60, 76, 100 and 160 cells;
  the bar `▎` on every row of the message, violet or blue, no ground behind
  it; the model muted; a rule of `─` in the frames' colour across B's
  width between two messages, a blank row on each side, none inside a
  message, at the start or before the input, never two in a row; a long
  prompt wrapping at the content column (TestConversationLook 3.1 to 3.4,
  3.1 the same layout at every width, 3.3 the rules between messages).

Item 4, regressions and checks.

- 4.1 Every golden frame rewritten and reviewed: the geometry of item 1 in
  each; the window scenes changed only around their window.
- 4.2 `go build ./...`, `go vet ./...` exit 0, `gofmt -l .` empty; `go
  test ./...` with 0 FAIL and only TestDiskListLink skipped; the timing
  step 4/4 within its targets.
- 4.3 ui-design.md (Layouts, Zone B, Zone C, Approvals, Colours),
  events-and-state.md (RESULT with agents), architecture.md (ASM-12) and
  AGENTS.md say what was built; brief.md holds amendment 16.
- 4.4 (added) The owner's scenario E: eight exchanges of seven calls, each
  run folded with its failure counted, deep paths keeping their file names,
  PgUp to the first prompt and PgDn back, a selection copying a narration's
  text alone, five resizes keeping the geometry (TestLongConversation).
- 4.5 (added) A mock file's prompt in an idle session clears the CLI
  turn's notice, as sending does (TestStateMachines, its row in
  events-and-state.md's Turn machine).
- 4.6 (added) The new colours on 256 colours, none on another of its kind's
  index (TestPalette256).

Item 5, a real Windows Terminal, maximised, then 120x40, 200x60 and 80x24,
screenshots compared with the templates (the four scenarios of the owner's
section 13 maximised; the other sizes checked as text).

- 5.1 Template 1: a replay of the demo's three exchanges (the recorded
  subagent session, twelve Read calls with one failed, "ok"); its
  screenshot beside Template 1, each difference listed and either fixed or
  a documented deviation.
- 5.2 Template 2: the recorded session stopped at the Write approval; the
  same comparison with Template 2.
- 5.3 The agent completed: C still two rows for it, "0 ACTIVE · 1 DONE".
- 5.4 A new prompt: C back to 3 rows, the earlier agent in A's tree.
- 5.5 At 120x40, 200x60 and 80x24: the geometry of item 1 read from the
  screen, nothing cut.

### Progress

- Started on 2026-10-20: the request recorded as brief amendment 16; this
  plan written.
- Built on 2026-10-20: the geometry (`dashboard.go`: `zoneAWidth`,
  `cHeight`, C and D full width, the preview inside C, the footer with C's
  keys); the cards (`cards.go`: the frame, the labels, the RESULT box, the
  tool rows' columns and outcomes, the fold row; `look.go`: the card kinds,
  the gaps, the name column, RESULT with the agents); the pending box
  (`approval.go`); C's two rows (`agentCard`); the compact editor; the
  colours (`palette.go`: the card ground, the muted text, the pill).
- Then, after the first screenshots against both templates: the preview's
  own box and the documented sentence on the pending card; the card's rows
  between narration, tool rows and a request; C's dot, grey state and wider
  gaps; a failed call's error cut before the verdict, never the verdict
  (plan 13); a mock file's prompt clearing the CLI turn's notice
  (`reduce.go`). The tests of earlier plans updated to the new look, their
  intent kept; new: TestAgentCardTemplate2, TestApprovalEffect,
  TestCompactEditor, TestLongConversation, a Turn row of TestStateMachines.
- Checked: `go build`, `go vet`, `gofmt -l .` empty; `go test ./...`
  0 FAIL, TestDiskListLink skipped; the timing step 4/4; 38 golden frames
  rewritten and checked by script (the geometry in each dashboard frame;
  every window's rectangle unchanged); Windows Terminal: the four scenarios
  maximised, 120x40, 80x24 and a 208x59 grid, their geometry read from the
  screen. Results in `doc/plan20-evaluation.md`.
- The owner's live review, from 2026-10-20, screenshot by screenshot.
  Scenario 1 (Claude Code 2.1.303, Opus 5.5): drawn as designed; one
  correction, zone D's context reading "N/A (no turn yet)" through the
  first turn, now "N/A (no result yet)" once a turn has started
  (`telemetry.go`; TestTelemetryNA's new case; reliability.md).
  Scenario 2 (twelve Reads, one failing): three corrections. The failed
  row named no file: it now shows its target, then its error, the error
  keeping half the room. A path was cut inside an element ("…rs\…\AppData")
  on the entry cursor's row: paths in tool rows are now cut at an element
  (`elidePath`). Zone A's twelve Read rows all read "C:\User…": a path
  there keeps its file name. Tests: TestFailedRowNamesItsTarget,
  TestElidePath; TestSyntheticScenarios updated (the failed Codex command
  now shows its command, cut, before its error); three golden frames
  rewritten (the cleaner path cuts).
  Then, on the owner's request: → on B's entry cursor no longer opens
  the full view, as Enter does; it only expands a folded run, as it
  unfolds in zone A, so → never opens and Enter is the one key that does
  (`inspect.go`, the footers and the keys page; TestEntryViewer and
  TestToolFoldKeys).
  Then the owner's amendment 17, Zone B's message layout and separators,
  with an image of the target: each message a block, its bar, its header
  on its own row and its body under it, a thin rule between messages, no
  ground behind them (`cards.go`: the frame, `entryRows` and `entryBody`,
  `messageRule`; `look.go`: `sep`, `top` and `tail`; the card ground
  removed from `palette.go`). The tests of plans 5 to 20 that read B's rows
  updated to the blocks, their intent kept; new: 3.13; 38 golden frames
  rewritten; Windows Terminal: the minimal conversation, the longer one and
  the pending approval, maximised, and the minimal one at 80x24.
  The owner's live run of scenarios 1, 2 and 4 in one session, on the new
  layout: drawn as designed; one correction, a rule at the scrolled
  window's edge, its next message out of view, sat one blank row above the
  input's rule as a second line: such a rule is now a blank row
  (`conversation` in `dashboard.go`; TestConversationLook "3.3 no rule at
  the window's edges").

## Closed plans

One entry each: what the plan did, its approval, its evaluation file and
its commits. Their criteria and progress logs: `git show 8888f59:plan.md`
(plans 1 to 13), `git show 84d2ad8:plan.md` (plan 14), `git show
b047434:plan.md` (plan 15), `git show becfb34:plan.md` (plan 16), `git show
f00c2d8:plan.md` (plan 17), `git show 8b3b16b:plan.md` (plan 18).

## Plan 1 - Research

The agent CLIs and how to drive them, telemetry, terminal input, the
reference projects, and the decision to build rather than fork
(`doc/research/`). Accepted on 2026-10-16, 9 PASS and criteria 8 and 9 kept as
FAIL, documented exceptions
([`doc/research/plan1-evaluation.md`](doc/research/plan1-evaluation.md));
commit 7e4d67c.

## Plan 2 - Architecture and reliability

`doc/architecture.md`, `doc/events-and-state.md`, `doc/ui-design.md` and
`doc/reliability.md`. Approved on 2026-10-16, 15 of 15 PASS
([`doc/plan2-evaluation.md`](doc/plan2-evaluation.md)); commit d5b910e.

## Plan 3 - Proposal

The MVP roadmap and the decisions it asked for (stack, dependencies, live
runs, configuration check). Approved on 2026-10-16, 12 of 12 PASS; commit
d5b910e, whose `doc/proposal.md` holds the roadmap. The design documents
and the code have replaced it since.

## Plan 4 - MVP (from amendment 4; steps detailed by plan 3)

The MVP, steps 0 to 8: startup, Claude Code, the streaming conversation,
approvals and resumption, Codex, agent activity, telemetry, the UI.
Approved on 2026-10-17 after a live demo on both CLIs, 14 of 14 PASS
([`doc/plan4-evaluation.md`](doc/plan4-evaluation.md)); commits ded53d8 to
cf58739.

## Plan 5 - Post-MVP improvements

The list plan 4 left open, Windows only. Approved on 2026-10-17, 17 of 17
PASS, then 15 of 15 live and 6 of 6 for the follow-up
([`doc/plan5-evaluation.md`](doc/plan5-evaluation.md)); commits d42531f,
897f640, 6813c0e.

## Plan 6 - Plan 5's known limits

Four known limits fixed, then items 5 and 6 added by the owner. Approved on
2026-10-17, 6 of 6 and 8 of 8 PASS
([`doc/plan6-evaluation.md`](doc/plan6-evaluation.md)); commits df273fe,
0814d67.

## Plan 7 - The open list after plan 6

The open list cleaned. Approved on 2026-10-17, 7 of 7 PASS
([`doc/plan7-evaluation.md`](doc/plan7-evaluation.md)); commit bffe9a5.

## Plan 8 - Codex's allow for the session

`acceptForSession` on Codex 0.163.0. Approved on 2026-10-17, 6 of 6 PASS
and its follow-up 3 of 3 ([`doc/plan8-evaluation.md`](doc/plan8-evaluation.md));
commits e75d591, 1daeec1.

## Plan 9 - Plan 8's items left not verified

Two items verified, the third recorded as not verifiable. Approved on
2026-10-18, 6 of 6 PASS
([`doc/plan9-evaluation.md`](doc/plan9-evaluation.md)); commits
c9576d9, 8dea129.

## Plan 10 - A spawned Codex agent's label and the timing step

Both items verified. Approved on 2026-10-18, 7 of 7 PASS
([`doc/plan10-evaluation.md`](doc/plan10-evaluation.md)); commits b71c114,
621609e.

## Plan 11 - A live approval offering the allow for the session

The live approval closed as not verifiable, by the owner's choice.
Approved on 2026-10-18, 3 of 3 PASS
([`doc/plan11-evaluation.md`](doc/plan11-evaluation.md)); commits 81f3bf8,
cb206b6.

## Plan 12 - Manual or Auto approvals

Brief amendment 8: a per-session switch between Manual and each CLI's own
Auto. Approved on 2026-10-18, 18 of 19 PASS and 1 FAIL accepted
([`doc/plan12-evaluation.md`](doc/plan12-evaluation.md)); commits cb31557,
83fa023.

## Plan 13 - A reviewer's refusal live

Claude Code's classifier denial seen live and a render defect it exposed
fixed; Codex's reviewer not reached, closed by the owner. Approved on
2026-10-18, 14 PASS and 3 not observed
([`doc/plan13-evaluation.md`](doc/plan13-evaluation.md));
commits d68bb55, 8888f59.

## Plan 14 - Nothing left open

Every open question of the research files settled or closed by the owner,
Claude Code's question card built and seen live, the documents reduced to
the latest state and a CLI update made visible. Approved on 2026-10-18, 18
PASS and 15 PASS ([`doc/plan14-evaluation.md`](doc/plan14-evaluation.md));
commits 0206ac1, ce06b97.

## Plan 15 - Recent projects, favourites and a project browser

Brief amendment 9: the startup screen's project list (`projects.json`,
favourites first) and its Ctrl+O browser. Approved on 2026-10-18, 21 PASS
([`doc/plan15-evaluation.md`](doc/plan15-evaluation.md)); commits bd9ffa3,
b047434.

## Plan 16 - Git worktree management

Brief amendment 10: the Worktrees view (Ctrl+W, at startup and in session)
with create, list, diff, merge and remove, git through `os/exec`. Approved
on 2026-10-18, 26 PASS
([`doc/plan16-evaluation.md`](doc/plan16-evaluation.md)); commits ce95ba2,
fad6d20, 642c5b9, becfb34.

## Plan 17 - A file viewer in the terminal

Brief amendments 11, 12 and 13: a file viewer in the terminal with Chroma's
highlighting, an Explorer of the project (F3) and the draft in a window
(Ctrl+G), with no external editor. Approved on 2026-10-19, 91 PASS
([`doc/plan17-evaluation.md`](doc/plan17-evaluation.md)); commits 1748808,
10c3e74, 904f9a6, 749cef4.

## Plan 18 - An adaptive layout

Brief amendment 14: zone C sized by the agents in play, a strip of 4 rows (3
under 100 columns) when there are none, and cleared at each new prompt.
Approved on 2026-10-19, 22 PASS
([`doc/plan18-evaluation.md`](doc/plan18-evaluation.md)); commit 8b3b16b.

## Plan 19 - Zone B as the centerpiece

Brief amendment 15: one speaker label per exchange, RESULT over an
exchange's outcome, compact tool rows and folds of long runs of calls, each
entry's turn (`Entry.Stage`). Done on 2026-10-20, 19 PASS
([`doc/plan19-evaluation.md`](doc/plan19-evaluation.md)). The owner
reviewed it in a demo the same day and asked for plan 20 (amendment 16),
which keeps its logic and replaces its look; not committed on its own, it
was committed with plan 20 once the owner approved that, on 2026-10-20.
