# Current Plan

The active plan with its progress, the owner's standing decisions, and one
entry per closed plan. Each plan comes from `doc/brief.md` and its
amendments, ends with the evaluation summary of `doc/evaluation.md`, and
waits for the owner's approval before the next one starts. This file keeps
the current state only (the owner, 2026-10-18): a closed plan's full text,
criteria and progress log are in git (`git show 8888f59:plan.md` for plans 1
to 13, `git show 84d2ad8:plan.md` for plan 14, `git show b047434:plan.md`
for plan 15, `git show becfb34:plan.md` for plan 16, `git show f00c2d8:plan.md`
for plan 17), and its results in its evaluation file.

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

## Plan 18 - An adaptive layout

**Approved by the owner on 2026-10-19** ("Do this first, we will tackle
sequentially"); done the same day, 16 PASS, then zone C cleared at each
prompt at the owner's review, 6 PASS
([`doc/plan18-evaluation.md`](doc/plan18-evaluation.md)). **Done and
approved by the owner on 2026-10-19**, its choices recorded in brief.md,
amendment 14. The first of the improvements the owner gives one by one
after plan 17, brief amendment 14. The four zones keep their places; only
how the right column's height is shared between B and C follows the
agents' activity.

The choices it is built with, shown to the owner at review:

- **The agents in play** (`State.InPlay`, their keys, and
  `State.AgentsInPlay`, their count): those running, and those that appeared
  or ended in the turn of the owner's last prompt or later. A prompt the
  owner submits starts that turn; a turn the CLI opens by itself, such as
  Claude Code's for a background agent's notification, does not. Before any
  prompt, every agent counts.
- **C lists only them,** the owner's third message ("It must reset EMPTY to
  its compact layout only when a new user prompt is submitted"): each new
  prompt clears C of the earlier agents, its counts back to 0, "no agents
  yet"; an agent still running stays. The earlier agents stay in zone A's
  tree, from which they are inspected; ↑ ↓ in C and [ ] in an inspection go
  through C's list.
- **None in play:** C is an empty strip of 4 rows from 100 columns, 3 under
  100 columns. B takes the rest of the column.
- **Agents in play:** C is as tall as they need, the owner's answer of
  2026-10-19 ("the space should adapt based on the deployed agent"): its
  title, a row per agent, its key hint and its border, at least the strip;
  at most half the right column, B taking the odd row and keeping at least
  12 rows (14 from 160 columns, where its input is taller); at most 5 rows
  under 100 columns. C's key hint now also shows when the agents leave a
  row for it.
- **From 160 columns,** with agents in play, C keeps at least 8 rows for the
  selected agent's preview beside the list; the strip has no preview.
- **Unchanged:** `z` zoom on C, F6 and every key, the selection, zones A and
  D, and every window. (The second build sized a focused C for every agent
  it listed; with C cleared at each prompt that rule went.)

### Criteria

Item 1, the rule (TestAgentsInPlay, `internal/state`).

- 1.1 No agent: 0. Two agents start in the owner's turn: 2; they complete:
  still 2.
- 1.2 The owner submits the next prompt, every agent done: 0; an agent
  starting in that turn: 1.
- 1.3 A turn the CLI opens by itself after the owner's turn changes nothing,
  with agents in play or none.
- 1.4 An agent still running when the owner submits a prompt: 1; with an
  agent of the new turn: 2; both end in it: still 2; the next prompt: 0.
- 1.5 Before any prompt, in a replay that starts with an agent: 1.

Item 2, the heights (TestAdaptiveLayout, `internal/render`), from P2 (no
agent), P3b (one) and P3b with more agents added running.

- 2.1 At 120x40: no agent, C 4 rows and B 29; one agent, C 4 (its row and
  the key hint); six, C 9 and B 24; twenty, C 16 and B 17 (half); P3b's
  agent after a new prompt, C 4 with the agent listed (now cleared: 5.2).
- 2.2 At 200x60: no agent, C 4 and B 47, no preview; one agent, C 8 with the
  preview; twenty, C 23; forty, C 25 and B 26 (half); after a new prompt, C
  4 without the preview.
- 2.3 At 80x24: no agent, C 3 and B 14; one agent, C 4 with the key hint;
  six, C 5 and B 12.
- 2.4 At 100x24: twenty agents, B keeps 12 rows and C takes 5; no agent, C
  4 and B 13.
- 2.5 Twenty agents done before a new prompt: a strip of 4 rows; focused, C
  16 at 120x40, 23 at 200x60, 5 at 80x24; no agent focused, the strip (now
  cleared, focused or not: 5.2).
- 2.6 `z` zoom on C and F6 as before: the existing tests pass unchanged,
  TestPaneScroll included.

Item 3, frames and checks.

- 3.1 TestGoldenFrames rewritten with `-update`; every changed frame differs
  only where B and C meet (B's extra rows, or C's strip or larger list),
  reviewed frame by frame.
- 3.2 `go build ./...`, `go vet ./...` exit 0, `gofmt -l .` empty;
  `go test ./...` with 0 FAIL and only the unelevated TestDiskListLink
  skipped; the timing step 4/4 PASS.
- 3.3 ui-design.md (Layouts at three sizes, zone C), events-and-state.md
  (Zone C's height), architecture.md (ASM-12) and AGENTS.md (status) say
  what was built; brief.md holds the request (amendment 14), and the
  choices joined it once the owner agreed them.

Item 4, in a real Windows Terminal, maximised first and then 80x24, with
the plan 17 harness (UI Automation, the screen read as text).

- 4.1 The mock `claude-subagent-approval.jsonl` (one agent): C the strip
  before its agent, then its row and the key hint, kept after it completes;
  the mock `codex-subagents.jsonl` (two agents): the strip, one agent's
  height, then two agents', kept after both complete.
- 4.2 The reset on a new prompt, live: Claude Code on Haiku in a temporary
  project, a prompt that launches two subagents, then "Reply with the single
  word ok"; C sized for two agents after the first, the strip after the
  second. About USD 0.05 of the owner's quota; it runs once the owner
  approves it.

Item 5, zone C cleared at each prompt, the owner's third message at review.

- 5.1 `State.InPlay` lists the agents in play, the earlier turns' cleared:
  "bg,new" at the end of 1.4's turn (TestAgentsInPlay).
- 5.2 After a new prompt, every agent done: C the empty strip, "0 ACTIVE ·
  0 DONE" and "no agents yet", 4 rows and B 29 at 120x40, 4 and no preview
  at 200x60, 3 at 80x24, focused or not; six agents in play, focused: as
  unfocused, C 9 (TestAdaptiveLayout).
- 5.3 In every case of TestAdaptiveLayout, C is empty exactly when no agent
  is in play.
- 5.4 TestPaneScroll's C subtests on a prompt that deploys 30 agents (60
  for the zoomed list), their assertions unchanged, all PASS.
- 5.5 Item 3 again: no golden frame changed by the clearing; build, vet,
  gofmt, `go test ./...` and the timing step as in 3.2; ui-design.md and
  events-and-state.md say what was built.
- 5.6 In a real Windows Terminal, maximised and 80x24: the four mocks as in
  4.1; live, C cleared from prompt 2's submission, "0 ACTIVE · 0 DONE" and
  "no agents yet", and after it.

### Progress

- First build (2026-10-19), C half the column with any agent in play: items
  1 to 3 and 4.1 passed (maximised, C 4 then 18 rows; 80x24, 3 then 5). The
  owner then saw one agent leave C mostly empty and asked for its height to
  follow the agents (amendment 14, second message): rebuilt as above.
- Items 1 to 3 done (2026-10-19): `State.AgentsInPlay` counts, with
  `Agent.EndStage` set wherever an agent ends; TestAgentsInPlay PASS, and in
  the first build it failed when the rule counted the CLI's own stages as
  prompts (1.3) or forgot when an agent ended (1.4). The heights in
  `internal/render/dashboard.go`; TestAdaptiveLayout PASS, 20 cases.
  TestPaneScroll failed on the first agent-sized build, its scroll position
  hint missing from a 4-row strip when C is focused: focus now sizes C for
  every agent, and the test passes unchanged. 38 golden frames rewritten and
  reviewed: in each, the two top rows, zone A and zone D unchanged; C 4 rows
  with one agent (8 at 200x60, with the preview) and 5 with two, B the rest;
  the P2 frames C 4 and B 29; the window scenes changed only behind the
  window. `go test ./...` 343 PASS, 1 SKIP (TestDiskListLink), 0 FAIL; vet,
  build and gofmt clean; the timing step 4/4 PASS.
- Item 3.3 done: ui-design.md, events-and-state.md and architecture.md say
  what was built; AGENTS.md's status names plan 18.
- Item 4.1 done again (2026-10-19, `p18mock.ps1`, the plan 17 harness: a
  Windows Terminal window driven by UI Automation, a temporary project,
  store and home, a PATH of System32 only), four runs, 0 FAIL. Maximised,
  156x44: the Claude mock C 4 rows before its agent, with it running, done
  and at the end (B 33); the Codex mock C 4 before, 4 with one agent, 5
  with two, 5 with one and then both done, and at the end (B 32). 80x24:
  the Claude mock C 3, then 4 (B 13) at the same points; the Codex mock C 3,
  4, then 5 (B 12). Each run: the window closed, no agent CLI started (the
  Claude desktop app's processes left aside), nothing written.
- Item 4.2 done (2026-10-19, `p18live.ps1`, approved by the owner): the
  dashboard on Claude Code 2.1.303 and Haiku. Maximised, the strip, 5 rows
  for prompt 1's two agents, the strip from prompt 2's submission; 80x24, 3,
  5, 3. Both runs 0 FAIL, configuration PASS. Two earlier maximised runs
  failed on the script (a variable name, an end of turn read from a lagging
  log) and on Claude Code's own caches in the configuration check; every
  height was right ([`doc/plan18-evaluation.md`](doc/plan18-evaluation.md)).
- Done: 16 criteria, 16 PASS.
- Item 5 done (2026-10-19), after the owner's review asked for C to be
  cleared at each prompt: `State.InPlay`, zone C's list and counts from it,
  ↑ ↓ and [ ] through it, the focus rule removed. TestAdaptiveLayout's
  after-a-prompt cases now expect C empty; TestPaneScroll's C subtests,
  which listed the agents of 60 earlier prompts, use a helper deploying 30
  in one prompt, their assertions unchanged. No golden frame changed. `go
  test ./...` 343 PASS, 1 SKIP, 0 FAIL; vet, build, gofmt clean; timing 4/4.
  The four mocks again 0 FAIL; live (Haiku), maximised C 4, 5, then cleared
  from prompt 2's submission, 80x24 3, 5, then cleared (its agents in the
  background that time, the CLI's turns for their notifications leaving C
  as it was); both 0 FAIL, configuration PASS, about USD 0.03.
- Done: 6 more criteria, 6 PASS. The owner kept a running agent in C at a
  new prompt ("Yes it's ok") and approved plan 18 on 2026-10-19.

## Closed plans

One entry each: what the plan did, its approval, its evaluation file and
its commits. Their criteria and progress logs: `git show 8888f59:plan.md`
(plans 1 to 13), `git show 84d2ad8:plan.md` (plan 14), `git show
b047434:plan.md` (plan 15), `git show becfb34:plan.md` (plan 16), `git show
f00c2d8:plan.md` (plan 17).

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
