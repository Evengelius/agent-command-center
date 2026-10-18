# Current Plan

The active plan with its progress, the owner's standing decisions, and one
entry per closed plan. Each plan comes from `doc/brief.md` and its
amendments, ends with the evaluation summary of `doc/evaluation.md`, and
waits for the owner's approval before the next one starts. This file keeps
the current state only (the owner, 2026-10-18): a closed plan's full text,
criteria and progress log are in git, `git show 8888f59:plan.md` for plans 1
to 13, and its results in its evaluation file.

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
  "newer than tested" meanwhile.

## Nice to have

Items tagged **[Nice to have]** are future improvements: none is built, and
none is listed as remaining, until the owner asks for one in a new plan
(the owner, 2026-10-18). Amendment 4 names the first ten "future
improvements" (`doc/brief.md`, Amendment 4, "For this phase, do NOT
implement").

- **[Nice to have]** Recent project history.
- **[Nice to have]** Project favourites.
- **[Nice to have]** An advanced project browser.
- **[Nice to have]** Git worktree management.
- **[Nice to have]** Automatic VS Code integration.
- **[Nice to have]** Advanced model configuration.
- **[Nice to have]** Cross-provider agent collaboration.
- **[Nice to have]** Provider switching during an active session.
- **[Nice to have]** Complex session profiles.
- **[Nice to have]** Advanced orchestration configuration.
- **[Nice to have]** Google Antigravity as an agent CLI, removed from the
  MVP by amendment 5; the provider interface stays open to it.

## Plan 14 - Nothing left open

**Done, approved by the owner on 2026-10-18** (committed as 0206ac1). The owner's request (2026-10-18), after plan 13's
approval: settle what is still open so that no plan 15 is needed for it.
Anything not settled is closed with its reason, and nothing is left for
later. Items 5 to 7 were added by the owner on 2026-10-18, after items 1 to
4 were done.

1. Claude Code's question round trip (CC-10 q3): probe whether allow with
   `updatedInput: {questions, answers}` gives the model the owner's
   answers. If it does, the question card answers from the dashboard. If
   it does not, the card keeps deny-and-type, and the reason is recorded.
2. The remaining open questions of `doc/research/claude-code.md` section
   10 (q2, q4 to q7, q9 to q11, q13, q14): each one is settled with
   evidence, or closed with the reason the dashboard does not depend on
   it. A probe runs only where the dashboard depends on the answer.
3. The items the owner closed on 2026-10-18, recorded as the owner's
   decisions:
   - Claude Code's Auto falling back to a prompt after repeated classifier
     blocks: the dashboard relays Claude Code's own mechanism, since any
     `can_use_tool` is already a card, and does not try to provoke it.
   - Codex: a subagent launch naming its model or effort, and Codex's
     default subagent limit: closed.
   - The checks by eye (look and feel; 256- and 16-colour terminals;
     flicker; physical keys; a NUL in a paste) and the excluded list:
     closed.
4. Documents and the full check.
5. The documents hold the latest state; git holds the history. This file
   keeps the active plan, the standing decisions and one entry per closed
   plan; `doc/proposal.md` (plan 3's roadmap, replaced by the design
   documents and the code) is deleted. The brief, the evaluation files and
   plan 1's research stay as they are.
6. A CLI update is never silent. The project bar says when the running CLI
   is newer than the last version probed; Claude Code's one undocumented
   field the dashboard depends on without a visible failure
   (`tool_result_meta[].permission_decision`) counts as unrecognised when
   it is missing; at each session start the agent compares the installed
   CLI versions with the last probed ones, tells the owner of an update,
   and proposes the re-check that `doc/evaluation.md` describes, which
   runs only on the owner's approval.
7. The full check for items 5 and 6.

Rules: the owner's Claude quota, Haiku as a per-process override, each run
in a temporary project outside the repository, with the configuration check
before and after. Per-process flags and protocol parameters only: no CLI
configuration file or system setting changed. Windows 11 and Windows
Terminal only; no new dependency. No commit until the owner approves.

Done when every item has passed its criteria or is closed with why, the
full check passes, no open item is left in plan.md or the research
documents, and the evaluation summary is reported to the owner.

Progress (update as each item finishes; read this first after a context
compaction).
Baseline: 8888f59, 261 top-level tests in 14 packages, 0 FAIL.
- [x] Item 1, Claude Code's question round trip. Criteria, written before
  the work. Probe P14a: a driver with the dashboard's arguments, Haiku, in
  `%TEMP%\acc-p14`. The prompt asks for one AskUserQuestion call with two
  questions, one single-select whose options include the non-ASCII label
  "Rouge café ✓", and one multiSelect.
  (1.1) a `can_use_tool` with `tool_name:"AskUserQuestion"` arrives with
  the documented input `{questions:[{question, header, options:[{label,
  description}], multiSelect}]}`;
  (1.2) the driver answers allow with `updatedInput: {questions, answers}`
  ("Rouge café ✓" for the first question, two labels joined by ", " for
  the second). The tool result is not an error and carries both answers;
  (1.3) the model's next text repeats the chosen labels byte for byte;
  (1.4) configuration check before and after: identical; the folder
  deleted, no CLI process left.
  If 1.1 to 1.3 pass:
  (1.5) the card answers. Keys 1 to 4 choose an option of the selected
  question (they toggle it when the question is multiSelect), ← and →
  select the question, and `a` sends once every question has a choice.
  The response sent is allow with `updatedInput {questions, answers}`;
  `d` and `c` are unchanged. Tests cover the keys, the JSON sent and the
  rendered card;
  (1.6) a live dashboard run (Haiku, Windows Terminal maximised): the card
  answered from the dashboard, the card line reads "answered by you
  HH:MM:SS", and the model's next text names the chosen option.
  Configuration check before and after: identical; the folder deleted, no
  process left.
  If the probe fails, the card stays as it is, and the reason is recorded
  in `doc/research/claude-code.md` and `doc/ui-design.md`.
- [x] Item 2, the open research questions. Criteria: in
  `doc/research/claude-code.md` section 10 (q2, q4 to q7, q9 to q11, q13,
  q14) and `doc/research/codex-cli.md` section 10 (q1 to q10), each
  question is settled, with evidence (a probe, a plan record with its
  date, or a test), or closed, with the reason the dashboard does not
  depend on it. Neither "Still open" list keeps an entry. Each design
  document that cites one of these questions as open (architecture.md,
  reliability.md, events-and-state.md, ui-design.md) says how it ended.
  - For CC-10 q13, a deterministic test passes non-ASCII text and CRLF
    through the fake CLI's pipes, and the text comes back byte for byte.
  - For CC-10 q10, P14a records `get_context_usage`'s latency, and whether
    any assistant, result or usage event follows it.
- [x] Item 3, the owner's closures recorded: Claude Code's Auto fallback
  (Claude Code's own mechanism, not provoked); the Codex subagent model or
  effort at launch, and the default subagent limit (closed); the checks
  by eye and the excluded list (closed). Each document listing one of
  these as open names the owner's decision of 2026-10-18.
- [x] Item 4, documents and the full check: go build and go vet (also
  `-tags live` and `-tags timing`), gofmt, and `go test ./...` with 0 FAIL;
  the timing step if an input, stream or frame path changes; the changed
  files without CR or BOM; a search of the living documents finds no open
  item without its outcome; the summary in `doc/plan14-evaluation.md`.
- [x] Item 5, the documents hold the latest state. Criteria:
  (5.1) plan.md holds the active plan with its progress, the owner's
  standing decisions, and one entry per closed plan under its exact
  `## Plan N - …` heading (approval, evaluation file, commits); every
  `plan.md#plan-…` link in `doc/` resolves to a heading; under 400 lines;
  (5.2) every standing decision found only in the closed plans' logs is
  kept: in plan.md, or in `doc/evaluation.md` for the race detector's
  setup;
  (5.3) `doc/proposal.md` is deleted and no living file (AGENTS.md,
  CLAUDE.md, plan.md, the design documents, the research documents but
  plan 1's evaluation) links to it; architecture.md cites commit d5b910e
  where it cited the proposal;
  (5.4) item 5 changes neither `doc/brief.md`, the evaluation files, plan
  1's reference studies, `doc/research/summary.md` nor
  `doc/research/gemini-cli.md`;
  (5.5) AGENTS.md's status line and layout say what the files now hold.
- [x] Item 6, a CLI update is never silent. Criteria:
  (6.1) each adapter names the last CLI version probed (Claude Code
  2.1.302, Codex CLI 0.163.0), and a test fails if that differs from the
  "Last probed" line of `doc/research/probes/README.md`;
  (6.2) the project bar adds "newer than tested", in the warning style,
  when the session's CLI version is newer than the one its adapter names;
  nothing when it is equal, older, unparsable or unknown, or in a mock
  session; tests cover each case; the existing golden frames are
  unchanged;
  (6.3) a Claude Code tool result with no `permission_decision` in its
  `tool_result_meta` counts as one unrecognised event whose reason names
  the field; every recording replays with none;
  (6.4) `doc/evaluation.md`, "After a CLI update": the session-start
  check (`claude --version` and `codex --version` against the last
  probed versions; the owner told of an update, the re-check proposed)
  and the re-check (the live gates, the probe drivers, new recordings
  added to the replay tests, the "Last probed" line and the adapters'
  versions updated), run only on the owner's approval;
  (6.5) an AGENTS.md working rule for the session-start check.
  Session-start check of 2026-10-18: Claude Code 2.1.302, Codex CLI
  0.163.0, both equal to the last probed; no update.
- [x] Item 7, the full check for items 5 and 6: go build and go vet (also
  `-tags live` and `-tags timing`), gofmt, `go test ./...` 0 FAIL
  (baseline 267); the timing step run alone if a render path changes; the
  changed files without CR or BOM; a second summary block in
  `doc/plan14-evaluation.md`, the first one unchanged.

Plan 14 results (2026-10-18):
- **Item 1, Claude Code's question round trip: done, all six criteria
  PASS.** P14a (about 11:52, Haiku, Claude Code 2.1.302): the
  `can_use_tool` carried `tool_name:"AskUserQuestion"`, the documented
  input and a new `requires_user_interaction:true`; the allow with
  `updatedInput {questions, answers}` produced the tool result "Your
  questions have been answered: "Which colour?"="Rouge café ✓", "Which
  days?"="Monday, Tuesday". …", not an error, and the model replied
  `Colour: "Rouge café ✓"; Days: "Monday, Tuesday"`, byte for byte;
  recorded as `p14a-claude-question-answer.jsonl` (68 lines, in
  `doc/research/probes` and `testdata/probes`); configuration check PASS,
  the folder deleted, no process left. The card (1.5): keys 1 to 4, ←
  and →, `a`; `d` and `c` unchanged; `TestQuestionAnswerKeys`,
  `TestAnswerQuestion`, `TestQuestionCard`, `TestQuestionAnswerRules`.
  The live run (1.6, about 12:22, Haiku, Windows Terminal maximised,
  attempt 2): the card answered 2, then 1 and 3, then `a` in 7.2 s; the
  card read "● Blue", "[x] Monday", "[x] Friday" before sending, then
  "✓ orchestrator · AskUserQuestion answered by you 12:22:17", "Which
  colour? → Blue", "Which days? → Monday, Friday"; the wrapper's log
  holds the allow with `answers {"Which colour?":"Blue", "Which
  days?":"Monday, Friday"}`; the model replied "Colour: Blue; Days:
  Monday, Friday"; Ctrl+Q closed it in 3.1 s; configuration check PASS
  (files identical, `~/.claude.json` only cache-state keys), the folder
  deleted, no process left. Attempt 1 (about 12:14) is not counted: the
  run script waited for "! APPROVAL " but a question card's header reads
  "? QUESTION ", so it never pressed F2; the card was later answered
  (Red, Monday at 12:19:53) by the owner by hand (confirmed by the owner),
  and the round trip worked the same way.
- **Item 2, the open research questions: done.** CC-10 and CX-10 each
  give every question an outcome, settled with evidence or closed with
  the reason; neither keeps a "Still open" list. CC-10 q13: P14a live and
  `TestEchoUnicode` ("café ✓ — 日本\r\nline 2\n" byte for byte through the
  fake CLI's pipes). CC-10 q10: `get_context_usage` answered in 914 ms,
  no assistant, result or `rate_limit_event` after it. The compaction
  row now has its test, `TestCompactBoundary`. reliability.md's "Not
  verified" tables give each row an outcome; architecture.md,
  events-and-state.md and ui-design.md say how each question ended.
- **Item 3, the owner's closures: done.** Claude Code's Auto fallback
  (claude-code.md section 11), the Codex subagent model or effort and its
  default limit (codex-cli.md section 10), the checks by eye and the
  excluded list (ui-design.md, reliability.md) each name the owner's
  decision of 2026-10-18.
- **Item 4: done.** go build, go vet (also `-tags live`, `-tags timing`)
  exit 0; gofmt prints nothing; `go test ./...` 267 top-level tests in 14
  packages, 0 FAIL (baseline 261, +6); the timing step run alone, all
  PASS (keystroke-to-frame p95 1.07 ms, 50,000-line draft p95 2.01 ms,
  long-session delta-to-frame p95 3.05 ms, target ≤ 16 ms); the 40
  changed and new files carry no CR and no BOM; the search of the living
  documents finds no open item without its outcome (the remaining hits
  are headings, pointers to reliability.md, or historical files). The
  summary is in [`doc/plan14-evaluation.md`](doc/plan14-evaluation.md).
- **Item 5, the documents hold the latest state: done, five criteria
  PASS.** plan.md holds the active plan, the owner's standing decisions and
  one entry per closed plan (10 `plan.md#` links, 0 unresolved); the race
  detector's setup moved to `doc/evaluation.md`; `doc/proposal.md`
  deleted, architecture.md cites d5b910e; brief.md, the evaluation files,
  plan 1's reference studies and summary.md untouched (gemini-cli.md's
  note dates from item 3); AGENTS.md's status line, layout and working
  rule updated.
- **Item 6, a CLI update is never silent: done, five criteria PASS.**
  `TestedVersion` in each adapter (Claude Code 2.1.302, Codex CLI 0.163.0)
  with `TestTestedVersion` against the README's "Last probed" bullet; the
  project bar's "newer than tested" in the attention colour
  (`TestNewerThanTested`, `TestProjectBarNewerThanTested`), the golden
  frames unchanged; a tool result without `permission_decision` is one
  unrecognised event naming the field
  (`TestToolResultWithoutPermissionDecision`), the 9 Claude recordings and
  their 17 tool results replay with none
  (`TestRecordingsHavePermissionDecision`); `doc/evaluation.md`, After a
  CLI update; the AGENTS.md session-start rule. The recordings key
  `tool_result_meta` by `"id"`: plan 13's
  `TestVerdictsOnlyForTheClassifier` now uses it too.
- **Item 7: done.** go build, go vet (also `-tags live`, `-tags timing`)
  exit 0; gofmt prints nothing; `go test ./...` 273 top-level tests in 14
  packages, 0 FAIL (baseline 267, +6); the timing step run alone, all PASS
  (keystroke-to-frame p95 1.05 ms, 50,000-line draft p95 2.03 ms,
  long-session delta-to-frame p95 3.29 ms, target ≤ 16 ms); the 49
  changed and new files carry no CR and no BOM. The summary is the second
  block of [`doc/plan14-evaluation.md`](doc/plan14-evaluation.md).

## Closed plans

One entry each: what the plan did, its approval, its evaluation file and
its commits. Their criteria and progress logs: `git show 8888f59:plan.md`.

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
