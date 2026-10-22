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
for plan 25), and its results in its evaluation file.

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

## Plan 26 - Zone A without notes

Brief amendment 25 (2026-10-22): the owner's dashboard showed "ⓘ
unrecognised 1 · quota updated 4m 36s ago" and "† undocumented field · cost
at API rates" under zone A's quota windows, and the owner asked to get rid
of that section completely and forever. The criteria below were written
before the code; nothing is committed until the owner approves.

### Design

- **D1 No notes.** The telemetry pane ends with the last quota window's
  rows: no rule after them, no "ⓘ" row, in any of its three forms (the
  full form loses its rule and two rows, the compact and tight forms one
  row; the tree gains them). Nothing of what the notes said is drawn in
  zone A: not the counts of errors, retries and unrecognised lines, not
  the quota's age, not "context derived from usage" or "tokens summed over
  threads", not "† undocumented field" or "cost at API rates", not the
  overage, plan and credits words, and not "values from the CLI's events".
- **D2 Where it lives.** The details view keeps all of it, as it already
  did: the errors, the unrecognised lines (plan 25), every value with its
  source and age ("quota.5h 35%† · rate_limit_event.unifiedWindows 4s
  ago", "(derived)" after a derived value), the counters. Its TELEMETRY
  heading gains the legend: "value · source · received · † undocumented
  field". The way there stays "Enter details" at the TELEMETRY heading's
  right end.
- **D3 What stays marked.** An undocumented value keeps its "†" in zone A
  and in the details view (AGENTS.md: anything undocumented is labelled
  where it is used); the usage column keeps "API-equivalent" under the
  cost. A derived value carries no mark in zone A.
- **D4 Standing decision.** "Zone A shows values, not notes" joins the
  owner's standing decisions in plan.md: no later plan adds a notes row,
  a legend or a counter to zone A; diagnostics go to the details view.
- **D5 Docs.** ui-design.md (Layouts' two frames and their text, Zone A's
  forms and the notes bullet, the telemetry table, Freshness, the states
  and symbols tables, Animations), events-and-state.md and reliability.md
  (where the counts are shown), AGENTS.md (status), brief.md (amendment
  25), `doc/plan26-evaluation.md`.

### Criteria

- 26.1 TestNoNotes (`internal/render`): on the P3b mock with two errors,
  retries, three unrecognised lines, a derived context and undocumented
  quota values, and on the Codex mock, zone A's rows at 120x40 and 208x59
  hold none of "ⓘ", "errors ", "retries ", "unrecognised", "quota
  updated", "derived", "undocumented field", "cost at API rates",
  "overage", "plan plus", "credits", "values from the CLI's events"; the
  pane's last row is the last window's "Resets in" row (full form) and the
  TELEMETRY heading row still ends with "Enter details"; an empty state
  draws no "ⓘ" either.
- 26.2 TestZoneAStructure: at 120x40 and 208x59 the full form's two "Resets
  in" rows, no rule after the last window, the tree at least 6 rows and
  3 rows more than plan 24's; at 100x24 the tight form on one row a window
  and the tree at least 3 rows.
- 26.3 TestDetailsUnrecognised: the details view's TELEMETRY heading reads
  "value · source · received · † undocumented field"; TestDetailsErrors
  and the values' "†" and "(derived)" unchanged.
- 26.4 Every dashboard golden frame regenerated and read: each loses the
  rule and the notes rows under its quota windows and nothing else changes
  but the tree's extra rows (the diff reviewed; no "ⓘ" left in
  `testdata/frames`).
- 26.5 `go build ./...`, `go vet ./...`, `gofmt -l .` empty, `go test
  ./...` all ok, the race run on `internal/render` clean, the timing step
  alone 7 of 7.
- 26.6 The documents of D5 updated, `grep -n "ⓘ" doc` finding only
  historical mentions; every changed text file UTF-8 without BOM, LF;
  `git status` lists the intended files only; nothing committed until the
  owner approves.

### Progress

- [x] D1 to D3 (`internal/render/telemetry.go`: the notes code gone, the
  legend in the details view's heading) and their tests (TestNoNotes,
  TestZoneAStructure, TestDetailsUnrecognised; the summary helper and the
  parsing, N/A and synthetic-scenario tests reading the same facts in the
  details view); 51 golden frames regenerated and reviewed by script
  (`scratchpad\plan24\framecheck.cjs`: the B and C sides unchanged, the
  notes rows gone, the tree's blank rows added).
- [x] D4 (the standing decision in plan.md), D5 (ui-design.md's two
  frames redrawn and its text, events-and-state.md, reliability.md,
  AGENTS.md, brief.md amendment 25).
- [x] The checks: `go test ./...` ok, the race run on `internal/render`
  clean, the timing step alone 7 of 7 within every target, its figures of
  this hour doubled on plan 25's tree alike (timed in turn: the machine,
  not the change); the evaluation
  [`doc/plan26-evaluation.md`](doc/plan26-evaluation.md): 6 of 6 PASS.
  Waiting for the owner's approval before any commit.

## Closed plans

One entry each: what the plan did, its approval, its evaluation file and
its commits. Their criteria and progress logs: `git show 8888f59:plan.md`
(plans 1 to 13), `git show 84d2ad8:plan.md` (plan 14), `git show
b047434:plan.md` (plan 15), `git show becfb34:plan.md` (plan 16), `git show
f00c2d8:plan.md` (plan 17), `git show 8b3b16b:plan.md` (plan 18), `git show
d84e5d4:plan.md` (plan 20), `git show f939d96:plan.md` (plan 21), `git show
d13ef71:plan.md` (plan 22), `git show e8d3e3b:plan.md` (plan 23), `git show
a3bfe5d:plan.md` (plan 24), `git show cb8f76d:plan.md` (plan 25).

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
