# Current Plan

The active plan with its progress, the owner's standing decisions, and one
entry per closed plan. Each plan comes from `doc/brief.md` and its
amendments, ends with the evaluation summary of `doc/evaluation.md`, and
waits for the owner's approval before the next one starts. This file keeps
the current state only (the owner, 2026-10-18): a closed plan's full text,
criteria and progress log are in git (`git show 8888f59:plan.md` for plans 1
to 13, `git show 84d2ad8:plan.md` for plan 14, `git show b047434:plan.md`
for plan 15, `git show becfb34:plan.md` for plan 16, `git show f00c2d8:plan.md`
for plan 17, `git show 8b3b16b:plan.md` for plan 18, `git show
d84e5d4:plan.md` for plan 20, `git show f939d96:plan.md` for plan 21,
`git show d13ef71:plan.md` for plan 22, `git show e8d3e3b:plan.md` for
plan 23, `git show a3bfe5d:plan.md` for plan 24, `git show cb8f76d:plan.md`
for plan 25, `git show a9a5145:plan.md` for plan 26), and its results in its
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
- **Zone A shows values, not notes** (amendment 25, 2026-10-22, plan 26):
  its telemetry pane carries no notes row, legend or counter, now or in
  any later plan; errors, retries, unrecognised lines, freshness and the
  legends live in the details view (Enter on TELEMETRY), where each value
  has its source and age. An undocumented value keeps its "†" mark where
  it is shown.

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

## Plan 27 - Zone C with the execution timeline

Brief amendment 26 (2026-10-22), the owner's image
`references/04-zone-c-timeline.png`
([reference-design.md](doc/reference-design.md#zone-c-with-the-execution-timeline)):
zone C redrawn after the image, strictly, from 100 columns; the status at
the timeline heading's right end is the selected agent's, from the state.
The criteria below were written before the code; the owner reviews the
Windows Terminal screenshots before any commit.

### Design

- **D1 The frame.** Zone C's top edge carries no title (as zone A since
  plan 24); the border's colour marks the focus. Inside, row 1: "C · LIVE
  AGENTS" in the title colour at the left, and at the right end the counts
  "N ACTIVE · M DONE" in grey, preceded as today by the agents out of view
  ("↑ 2 above · ↓ 5 below", amber) and the "* requested model" footnote.
  Row 2: a rule across the inner width. Row 3: the panels' headings, "AGENTS"
  over the list and "EXECUTION TIMELINE" over the timeline, in the title
  colour, with the selected agent's status at the timeline's right end: its
  symbol and its state word in capitals, in the symbol's colour ("✓
  COMPLETED" green, "◉ RUNNING" amber, "● WAITING" pink, "✗ FAILED" red,
  "○ STOPPED" grey, "◉ ANNOUNCED" amber). A vertical rule "│" in the
  frame's colour separates the panels on every row from row 3 to the
  bottom; rows 1 and 2 span the inner width.
- **D2 The list**, the left panel: 39% of the inner width, rounded, at
  least 30 cells (45 at 120 columns, 80 at 208). Two rows an agent, plan 20's
  Template 2 with the image's changes: the first row the selection mark,
  the status symbol, the label, the model, then the state as today
  ("waiting: approval", the running word, "completed · " and the summary's
  first line, "failed · " and the error's), cut to the panel with "…"; the
  second row the elapsed time, the retries, the tokens, then a vertical
  bar in the frame's colour and "tool: " with the last call's name in
  white, its symbol only while it runs or when it failed (the image's
  completed Write shows none), then "next: <tool> (approval pending)" as
  today. The rows are ordered as today; the list scrolls with the
  selection; "no agents yet: they appear as the CLI starts them" on the
  row under the header when none is in play (the panels and their
  headings are not drawn then).
- **D3 The timeline**, the right panel: the selected agent's entries (its
  messages, then its history lines, as the preview showed them), the
  newest last, one entry each: the time "17:11:31" in grey, the role in
  its colour padded to 9 cells ("notice" in zone B's speaker blue,
  "assistant" in its result green, any other role grey), a vertical bar in
  the frame's colour, then the text wrapped onto at most two rows of the
  room left, the second cut with "…" when the text goes on, the bar drawn
  on both rows; a rule in grey across the panel between entries, none
  after the last. The panel shows the tail; the wheel scrolls it (panel P,
  by rows), and while it is held away from the tail the heading's right
  end reads "↑ n ↓ m · End follows" in place of the status. "no activity
  yet" in grey when the agent has no entry. The agent's model with its
  source and its effort, which the old preview heading carried, stay in
  the list's model column and in the inspection (Enter).
- **D4 Height.** With no agent in play, 4 rows: the edges, the header row
  and "no agents yet…". With agents in play from 100 columns, C needs its
  three heading rows, the larger of the list's rows (two an agent) and the
  timeline's eleven (four entries of two rows and three rules, the image's
  four entries), and its two edges: 16 rows for one agent. The list's rows
  take up to half the rows A, B and C share (plan 18: six agents at 120x40
  give C 17 rows, twenty 18; twenty at 200x60 give 28), the timeline's up
  to a third of them, and C never has less than one agent's rows with the
  headings (7): with one agent 16 at 208x59 and 200x60, 13 at 156x44
  (three entries), 12 at 120x40 (two entries), 7 at 160x24 to 160x26 and 8
  at 160x27 (plan 21's floor, one entry). B keeps its least rows as today
  (twenty agents at 100x24 leave C 9). Under 100 columns the narrow layout
  keeps plan 18's one-row list with its title in the edge and its heights.
- **D5 Keys and mouse** are unchanged: ↑ ↓ select, Enter inspects, z
  zooms (the same drawing over the middle and bottom), PgUp, PgDn, Home
  and End scroll the list; a click on C focuses it and starts a
  selection, as today; the wheel over the timeline scrolls it; the
  footer's hints as today.
- **D6 Docs.** ui-design.md (Layouts' two frames' zone C rows and their
  text, Zone C rewritten), reference-design.md (the image), AGENTS.md
  (status), brief.md (amendment 26), `doc/plan27-evaluation.md` with the
  conformance table against the image.

### Criteria

- 27.1 TestZoneCDesign (`internal/render`): on the P3b mock at the Write
  approval and on the quota mock, at 120x40 and 208x59: C's top edge holds
  no title; its row 1 reads "C · LIVE AGENTS" at the left and ends with
  "1 ACTIVE · 0 DONE" (P3b) or "0 ACTIVE · 1 DONE" (quota); row 2 is a
  rule; row 3 reads "AGENTS" at the left and "EXECUTION TIMELINE" at the
  list's right, ending with "● WAITING" (P3b) or "✓ COMPLETED" (quota);
  every row from 3 to the last has "│" at the list's right edge; the list
  is 45 cells at 120 columns and 80 at 208.
- 27.2 TestZoneCDesign: P3b's agent rows read "▸● general-purpose",
  "haiku-5-5", "waiting: approval", then "00:13", "21.2k tok", "│ tool:
  Bash" with no symbol after the completed call, "next: Write (approval
  pending)"; a running call keeps its symbol and a failed one its "✗"
  (TestAgentCardTemplate2 adapted).
- 27.3 TestZoneCDesign: the timeline's entries read "<HH:MM:SS>  <role>
  │ <text>", the role padded to 9 cells, "notice" in stSpeaker and
  "assistant" in stResult, a text longer than two rows cut with "…" on
  its second row, the bar on both rows, a grey rule between entries and
  none after the last, the newest entry last; the tail shown at 120x40
  (two entries) and 208x59 (four); scrolled up, the heading's right end
  reads "↑ n ↓ m · End follows"; an agent with no entry reads "no activity
  yet".
- 27.4 TestDashboardGeometry, TestAdaptiveLayout, TestZoneCFloor and
  TestCHeight: C's rows with one agent 16 at 208x59 and 200x60, 13 at
  156x44, 12 at 120x40, 7 at 160x24 to 160x26 and 8 at 160x27; 17 with
  six agents and 18 with twenty at 120x40, 28 with twenty at 200x60, 9
  with twenty at 100x24; 4 with no agent; A and B share the rest, B
  keeping its least rows; the narrow layout's numbers unchanged.
- 27.5 The keys and the mouse: the C cases of the key, zoom, select and
  pane tests pass with the new rows (a click on C focuses it and selects
  text; the wheel scrolls the timeline, panel P).
- 27.6 Every golden frame regenerated and read; no frame holds "· LIVE",
  "(reported)" or "no activity yet" beside a list; each dashboard frame's
  C holds the heading row and the panels. Corrected on 2026-10-22 at the
  owner's decision, after the evaluation found "no activity yet" in two
  frames as 27.3 asks: no frame holds the old preview's heading ("<label>
  · LIVE") or its origin labels ("(reported)", "(not reported)"), and the
  timeline keeps "no activity yet" for an agent without entry.
- 27.7 `go build ./...`, `go vet ./...`, `gofmt -l .` empty, `go test
  ./...` all ok, the race run on `internal/render` and `internal/app`
  clean, the timing step alone 7 of 7.
- 27.8 Windows Terminal, maximised at 9 pt (208x59) and at 120x40, on the
  P3b mock (the agent waiting) and the quota mock (completed): the
  screenshots compared with the image in a conformance table, each
  element PASS or a documented deviation; the owner reviews them before
  any commit.
- 27.9 The documents of D6 updated; every changed text file UTF-8 without
  BOM, LF; `git status` lists the intended files only.

### Progress

- [x] D1 to D5 (`internal/render/zonec.go`: the frame, the list, the
  timeline, the height rule; `dashboard.go`'s old zone C, preview and
  constants gone, the card's tool bar) and their tests (TestZoneCDesign,
  TestCHeight; the layout, floor, model, retries, scenario, zoom and
  long-conversation tests adapted to the list's width and C's rows); 46
  golden frames regenerated and reviewed by script.
- [x] The suite (`go test ./...`, 14 packages ok), the race run on
  `internal/render` and `internal/app` (clean), the timing step alone (7 of
  7: keystroke-to-frame p95 1.0 ms, the 50,000-line draft 2.1 ms,
  delta-to-frame after two hours 6.0 ms, the streamed answer 4.0 ms, the
  thinking card 3.5 ms, paste-to-frame 1.2 ms), build, vet and gofmt:
  `scratchpad\plan24\p27-checks.log`.
- [x] The Windows Terminal screenshots (`scratchpad\term\p27demo.ps1`, the
  plan 24 harness on this tree's executable): the completed agent on the
  quota mock at 208x59 and 120x40, the agent waiting for its Write on a
  copy of the P3b mock whose approval stays pending 20 s longer
  (`plan24\mkwait.mjs`), a four-entry timeline on the thinking-subagent
  mock at 208x59; the conformance table in the evaluation.
- [x] D6 (ui-design.md's Zone C and Layouts, reference-design.md's section
  and table, brief.md's amendment 26, AGENTS.md's status) and the
  evaluation (`doc/plan27-evaluation.md`: the criteria, the conformance
  table, the deviations).
- [ ] The owner's review of the screenshots and the decision on 27.6 (the
  evaluation's one FAIL: "no activity yet" in the timeline of an agent
  without entry, which 27.3 asks for); then the commit.

### Second pass: the owner's correction (2026-10-22)

The owner ran the dashboard live on Claude Code (three screenshots, 19:06
to 19:08, a general-purpose subagent on Fable 5.1 writing found.txt in
acc-demo) and found the design not respected, then sent two close-ups of
it (`references/05-zone-c-timeline-close.png`,
`references/06-zone-c-timeline-close.webp`; brief amendment 27): "the
separation "|" is not respected either", "on the left, there is too much
space". What the close-ups show that the first pass missed, and the design
that follows:

- **D7 The lines and the title.** "C · LIVE AGENTS" in the title colour,
  bold, as "AGENTS" and "EXECUTION TIMELINE" (the first image's white was
  its scale); the counts grey. The rule under the header and the vertical
  line between the panels, from the headings to the bottom, in the title
  colour, not bold (a shared style, `stPanelRule`; white on 16 colours),
  not the frame's dim grey. The bars inside the panels, the card's column
  bar and the bar before each entry's text, grey (stGrey); the rules
  between entries stay in the frame's colour, the close-ups' thin dim
  lines.
- **D8 The card**, two columns on two rows, a grey bar between them on
  both rows, nothing else:

  ```text
   ✓  general-purpose     │  fable-5-1
      00:41   30.1k tok   │  tool: Write
  ```

  Row 1: a margin cell, the status symbol (the pink "●" waiting), two
  cells, the label in white bold padded to the label column; three cells,
  "│", two cells; the model in grey ("model N/A" without one, "*" after a
  requested one, footnoted in the header as today). Row 2: four cells, the
  elapsed clock, the retries "↻ 2" in amber when any, three cells, the
  tokens, padded to the label column; three cells, "│", two cells; "tool:
  " in grey and the agent's current tool in white bold: its last call's
  name, a call waiting for the owner included (P3b's "Write"), else the
  tool the CLI reported, else "none yet" in grey; a failed call's red "✗"
  after the name, no symbol otherwise. No state word, no summary or error
  text, no effort, no "next:" on the rows: the inspection has them, the
  timeline heading the status. The selected agent's two rows lie on zone
  B's tool rows' ground (plan 19) from the cell after the margin to the
  cell before the panel's end, the close-ups' band; on 16 colours, which
  have no ground (as zone B's tool rows there), no mark either: both
  palettes draw the same characters (TestColourProfiles), and Enter
  inspects the selected agent.
- **D9 The width.** The panel is as wide as its card, not two fifths of
  the screen: 12 cells (the margins, the gaps and the bar) plus the label
  column, the longest label or second-row content over the listed agents,
  at least 17 ("00:41   30.1k tok"), plus the tool column, the longest
  model with its mark or "tool: <name>", 11 ("tool: Write") to 22
  ("tool: commandExecution"). The whole never exceeds the larger of 39%
  of the inner width and 40, nor the inner width less 23 (the timeline
  keeps 20 cells): when it would, the tool column gives way down to 16,
  then the label column down to 17, then the tool column down to 11,
  names cut with "…". P3b's card gives 40 at every width from 100
  columns; the timeline takes the rest (73 cells at 120 columns, 161 at
  208). The heights of D4 do not change.

Criteria, written before the code:

- 27.10 TestZoneCDesign on P3b at its Write and the quota mock, at 120x40
  and 208x59: "C · LIVE AGENTS" in stTitle; row 2's rule and the " │ " at
  the panel's edge on every row from 3 in stPanelRule; the panel 40
  cells; P3b's card rows " ●  general-purpose     │  haiku-5-5" and
  "    <mm:ss>   <n>k tok   │  tool: Write", padded to 40, the bars on
  the same cell; the label in stBold, the model and the bar in stGrey,
  "tool: " in stGrey, "Write" in stBold; no "waiting", "completed",
  "running", "next:" or summary text on the list's rows; every span of
  the selected agent's rows between the margins on the ground (tint) and
  none of a second agent's; on the 16-colour palette no ground and the
  same characters (corrected as the tests ran: the first wording gave "▸"
  in the margin cell, which TestColourProfiles forbids).
- 27.11 TestAgentLabelColumn (the 40-cell label, the 26-cell tool name):
  the panel 45 cells at 120 columns, 61 at 160 and 74 at 200, the label
  column 17, 33 and 40, the tool column 16, 16 and 22, the long label
  whole only at 200, the 18-cell one at 160 and 200, the tool name "tool:
  abcdefghi…" at 120 and 160 and "tool: abcdefghijklmno…" at 200;
  TestAgentModelColumn, TestAgentSourceAndEffort,
  TestListModelAcrossWidths: the model grey from 100 columns, in its
  family colour under 100, the effort in the inspection only;
  TestAgentRetries: "↻ 2" after the time on the second row;
  TestAgentCardTemplate2: P3b's waiting agent reads "tool: Write" with no
  "next:"; the 80x24 row unchanged.
- 27.12 TestCHeight, TestAdaptiveLayout, TestZoneCFloor and
  TestDashboardGeometry pass unchanged: the heights of 27.4 hold.
- 27.13 Every golden frame regenerated and read by script: each dashboard
  frame from 100 columns with an agent holds the card's "│" on both rows
  at the same cell, the panel's "│" on every row from the headings, and
  no "waiting: approval", "completed ·", "next:" or running word on the
  list's rows; ui-design.md's two Layouts frames redrawn by TestDocFrames.
- 27.14 `go build ./...`, `go vet ./...`, `gofmt -l .` empty, `go test
  ./...` all ok, the race run on `internal/render` and `internal/app`
  clean, the timing step alone 7 of 7.
- 27.15 Windows Terminal, maximised at 9 pt (208x59) and at 120x40, on
  the quota mock, the waiting copy of P3b and the thinking-subagent mock:
  a conformance table against the close-ups, each element PASS or a
  documented deviation; the owner reviews the screenshots before any
  commit.
- 27.16 ui-design.md (Zone C, the Layouts frames), reference-design.md
  (the close-ups), brief.md (amendment 27), plan.md updated; every
  changed text file UTF-8 without BOM, LF; `git status` lists the
  intended files only.

Progress of the second pass:

- [x] D7 to D9 (`internal/render/zonec.go`: the card's two columns and
  bar, the panel's width from the cards, the lines in the title colour,
  `stPanelRule` in `palette.go`; `dashboard.go`'s old card gone) and their
  tests (TestZoneCDesign's 27.10 checks, TestCardColumns; the model,
  layout, approval and scenario tests adapted); 46 golden frames
  regenerated and reviewed by script; ui-design.md's two frames redrawn.
- [x] The checks (27.14): build, vet, gofmt, the suite (14 packages ok),
  the race run on render and app (clean), the timing step alone (7 of 7:
  keystroke-to-frame p95 1.1 ms, the 50,000-line draft 2.5 ms,
  delta-to-frame after two hours 4.6 ms, the streamed answer 3.7 ms, the
  thinking card 3.7 ms, paste-to-frame 1.0 ms): `scratchpad\plan24\p27b-checks.log`.
- [x] The screenshots (`scratchpad\term\p27bdemo.ps1` on this tree's
  executable, `out-p27b\`: the completed agent at 208x59 and 120x40, the
  waiting agent and a four-entry timeline at 208x59) and the conformance
  table against the close-ups, 18 rows PASS (`doc/plan27-evaluation.md`,
  Second pass).
- [x] The documents (ui-design.md, reference-design.md, brief.md's
  amendment 27, AGENTS.md) and the evaluation's second block.
- [ ] The owner's review of the new screenshots and the decision on 27.6;
  then the commit.

## Closed plans

One entry each: what the plan did, its approval, its evaluation file and
its commits. Their criteria and progress logs: `git show 8888f59:plan.md`
(plans 1 to 13), `git show 84d2ad8:plan.md` (plan 14), `git show
b047434:plan.md` (plan 15), `git show becfb34:plan.md` (plan 16), `git show
f00c2d8:plan.md` (plan 17), `git show 8b3b16b:plan.md` (plan 18), `git show
d84e5d4:plan.md` (plan 20), `git show f939d96:plan.md` (plan 21), `git show
d13ef71:plan.md` (plan 22), `git show e8d3e3b:plan.md` (plan 23), `git show
a3bfe5d:plan.md` (plan 24), `git show cb8f76d:plan.md` (plan 25), `git show
a9a5145:plan.md` (plan 26).

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
was committed with plan 20 once the owner approved that, on 2026-10-20;
commit d84e5d4.

## Plan 20 - Strict conformance to the approved templates

Brief amendments 16 and 17: the owner's two templates as one design
system, A and B in one upper row and C full width under them, zone B's
messages as blocks with a rule between them, the tool rows, the folds, the
pending approval's pink box and RESULT. Approved on 2026-10-20 after the
owner's live review, 32 of 32 PASS
([`doc/plan20-evaluation.md`](doc/plan20-evaluation.md)); commit d84e5d4,
with plan 19.

## Plan 21 - What the review of d84e5d4 found

A read-only review of commit d84e5d4: zone D's session time past 2,000
events, two timed tests outside the timing step, Claude Code's
undocumented `api_message_id` labelled with a fallback, zone C's floor at
160x24 to 160x26, and documents promising what the code did not do (the
send and newline keys built in `config.json`, the others corrected).
Approved on 2026-10-20 with the recommended choices, 38 PASS and 2 FAIL,
then the long session's frame time fixed at the owner's request, 6 PASS
([`doc/plan21-evaluation.md`](doc/plan21-evaluation.md)); commits bdc4250,
f939d96.

## Plan 22 - A live thinking indicator

Brief amendments 18 to 20: zone B's live thinking card for Claude Code and
Codex, with its animations, a summary row for each thinking phase that
opens on its details, and an activity line while a turn works; corrections
C1 to C12 from a read-only investigation
([`doc/plan22-investigation.md`](doc/plan22-investigation.md)); Codex
started through the local proxy that `config.json` names (amendment 19);
Codex's reasoning summaries asked for and Claude Code's hook events known
(amendment 20), checked live with one Codex turn (L20). Approved on
2026-10-21: 31 of 31 PASS, then the corrections 17 of 18 with 5.14 fixed,
amendment 20 13 of 13, and L20 8 of 8 once two criteria were corrected
([`doc/plan22-evaluation.md`](doc/plan22-evaluation.md)); commits
e6895c6, 6306bfb and d13ef71, then the approval commit.

## Plan 23 - The acc launcher in the project

Brief amendment 21: `acc`, typed in Windows Terminal, opens the dashboard
in a tab of its own at the owner's size, a 9 pt font. `acc.cmd` at the root
runs `scripts\acc.ps1`, which writes a Terminal fragment for the profile
and makes Terminal reload. The day's machine-local changes were undone.
Approved on 2026-10-22, 6 of 6 PASS before the commit, then committed and
pushed ([`doc/plan23-evaluation.md`](doc/plan23-evaluation.md)); commit
e8d3e3b, then the approval commit.

## Plan 24 - Zone A with the telemetry, zone D removed

Brief amendment 22: zone A redrawn after the owner's image, holding the
orchestrator's dynamic status, the stages as blocks and zone D's telemetry
(session usage, quota windows with lavender bars and countdowns, the notes,
the details view); zone D removed; Fable and Mythos as model families.
Amendment 23, after the owner's live session: the Claude adapter reads the
`/usage` report after the handshake and after each turn, so the
model-scoped "Fable limit" shows live, the exchange kept out of the
conversation. Approved on 2026-10-22: 35 of 35 PASS with the conformance
table 22 rows PASS, amendment 23 6 of 6 and its live check 8 of 8
([`doc/plan24-evaluation.md`](doc/plan24-evaluation.md)); commits 6209267
and a3bfe5d, then the approval commit.

## Plan 25 - The unrecognised lines named

Brief amendment 24: the owner's scenario 1 run on Fable 5.1 ended with
"unrecognised 2" and no way to see which lines. The details view now lists
the last 20 unrecognised lines with their time, type, reason and text; a
replay tool names them from a logging wrapper's record; the owner's two,
named by a logged rerun, were `system/task_summary`, known with no event
since. Approved on 2026-10-22: 7 of 7 PASS, two live runs
([`doc/plan25-evaluation.md`](doc/plan25-evaluation.md)); commit cb8f76d,
then the approval commit.

## Plan 26 - Zone A without notes

Brief amendment 25: the owner asked to get rid of zone A's notes
("unrecognised 1", "quota updated", "† undocumented field", "cost at API
rates") completely and forever. The telemetry pane now ends with the last
quota window's rows; the counts, ages and legends live in the details view,
where an undocumented value keeps its "†" and the heading explains it; a
standing decision. Approved on 2026-10-22: 6 of 6 PASS, 51 golden frames
reviewed ([`doc/plan26-evaluation.md`](doc/plan26-evaluation.md)); commit
a9a5145, then the approval commit.
