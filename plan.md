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
for plan 25, `git show a9a5145:plan.md` for plan 26, `git show 041e7b8:plan.md`
for plan 27, `git show 8fe0077:plan.md` for plan 28, `git show d696f22:plan.md`
for plan 29, `git show 6025513:plan.md` for plan 30, `git show
8e2a721:plan.md` for plan 31, `git show b56d23e:plan.md` for plan 32, `git show
bc5d92b:plan.md` for plan 33, `git show 47b50ce:plan.md` for plan 34, `git show
b43cf46:plan.md` for plan 35, `git show c0cebc4:plan.md` for plan 36, `git show
31c22ef:plan.md` for plan 37, `git show dc9c6d9:plan.md` for plan 38, `git show
55e2bed:plan.md` for plan 39), and its results in its evaluation file.

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
- **The Git Graph's watch stays as it is** (amendment 65, 2026-10-28):
  one `git for-each-ref` every 2 s while the panel shows, told its cost
  (about 6 % of one core here); a watch of the reference files' times is
  not proposed again unless the owner asks.

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
(amendment 9), the browser replaced by the system's folder dialog in plan
52 (amendment 53); git worktree management was plan 16 (amendment 10). The
automatic VS Code integration was replaced by the owner with a file viewer in
the terminal, plan 17 (amendment 11). The session's model, effort and
Ultracode selection was plan 34 (amendment 34); configuring models beyond the
session stays below.

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
- **[Nice to have]** Codex's earlier turns drawn after a resume from the
  home screen, paged with `thread/turns/list` and `thread/items/list`, with
  their rendering cost, their tool events and very long sessions (plan 69,
  design D1).
- **[Nice to have]** A list of a project's earlier sessions to resume, not
  only each agent CLI's last one (plan 69, design D8).

## Roadmap

The owner's order (amendment 65, 2026-10-28), plans 64 to 66 now closed,
and plans 67 and 68, which the owner put before it, closed too (the
characters typed with AltGr; the keyboard reader's races): the session's
resumption from the home screen, plan 69, below, the last of the four
suggestions. Each starts as a plan for the owner's review when the one
before is closed.

## Plan 69 - The session's resumption from the home screen

Brief amendment 71 (2026-10-29): the last of the four suggestions the
owner kept (amendment 65), "continuing the last session from the home
screen through each CLI's own documented resume option, after a probe of
both". Proposed for the owner's review, then revised around the four
points of the owner's design review (conditional approval, 8.5/10), the
probes P69a and P69b run first, as the review asked. The owner's final
review accepted the design but for one detail, Codex's telemetry after a
resume without its turns, now settled (D9, criterion 69.7), and kept the
crash resume as it is (D5, D8). Approved for implementation by the owner
on 2026-10-29 (amendment 71).

### What was found (2026-10-29, on 3d395a8)

- **What resumes today.** A crashed session, in the same run
  (reliability.md, Resumption): Ctrl+R (`state.Resume`,
  `internal/state/actions.go`) starts the CLI again on the same session
  (`CmdStartSession` with its ID, `start(resume)`, `internal/app/app.go`):
  Claude Code with `--resume <id>` (`internal/provider/claude/session.go`),
  Codex with `thread/resume` (`internal/provider/codex/session.go`). The
  conversation shown is the dashboard's own record, kept in memory
  (ASM-11). Across dashboard runs nothing resumes, "not in the MVP"
  (reliability.md), the project store keeping paths and times only
  (ASM-24).
- **Claude Code resumed in a new process** (P69a on 2.1.303, the
  dashboard's command line and handshake; `doc/research/probes/README.md`):
  the same session ID from the first line (the owner's `SessionStart` hook,
  run as `SessionStart:resume`) to `system/init`; no earlier message on
  stdout, as in P4; the code word answered; the session's cost carried over
  (the first result reports the earlier process's 0.0048 USD). An unknown
  ID: no `initialize` answer; 1.1 s after the start a `result`
  `error_during_execution` whose `errors` hold "No conversation found with
  session ID: <id>", the same words on stderr, then exit 1, no model call.
  These words are the CLI's own, not a documented contract.
- **Codex resumed in a new app-server** (P69b on 0.163.0): the same thread.
  The dashboard's `thread/resume` gets every turn with every item (3,360
  bytes for one turn), and Codex follows it with a `deprecationNotice`:
  full-history hydration is deprecated for paginated threads, to be
  replaced by `excludeTurns: true` with `thread/turns/list` and
  `thread/items/list`. `excludeTurns` is in 0.163.0's stable schema: with
  it the answer has no turn (2,703 bytes) and the code word was answered.
  After each resume Codex replays the thread's token usage at once and
  sends `thread/goal/cleared`, which the mapper reports as unrecognised. An
  unknown thread: error -32600 "no rollout found for thread id <id>"; a
  malformed ID: -32600 "invalid session id: …".
- **Telemetry after a resume** (P69a's and P69b's lines replayed through
  the mappers, the state and zone A's pane; a scratch test, deleted after;
  no new probe). Codex, the resume answered: the same values with the turns
  or without (`excludeTurns`), from Codex's replay of the thread's usage 2
  ms after the answer: CONTEXT "6% of 258.4k", TOKENS "16.4k", EST. COST
  N/A "Codex reports none", as in any Codex session. After the first new
  turn: TOKENS "32.9k", the thread's total with its history (16,430 before,
  16,467 for the turn), CONTEXT the new call's 6%, the turn's duration
  2.4 s. The quota comes from the dashboard's own `account/rateLimits/read`
  after any resume's answer (`internal/provider/codex/session.go`), which
  the probe's driver did not send, then from Codex's update after the turn.
  Claude Code: the quota at once, from the handshake's usage read; CONTEXT,
  TOKENS and EST. COST N/A until the first new turn's result, then "4% of
  1M", "85.3k" and "$0.01", the history included (the earlier process's
  40,396 tokens and 0.0048 USD). The usage read's result carries those
  restored totals, but the dashboard keeps its figures out of the session's
  (P24u). Until a value comes, zone A's reasons are a new session's, "no
  turn yet" and "no result yet", untrue of a session that has had turns.
- **A start that fails today** (`internal/state/reduce.go`,
  `internal/render/dashboard.go`): a refused request during the handshake
  ends the start ("handshake failed: …"), and so does the CLI's exit ("the
  CLI exited unexpectedly"); zone B then shows "The session could not
  start" with the reason, and Esc goes back to the home screen, its answers
  kept. Nothing falls back to a new session. Claude Code's refusal reaches
  the event log only (its mapper reads the `result` as a failed turn), not
  the card.
- **The home screen** (plan 51): the notes between the project field and
  START SESSION are the dashboard's own lines in the design's style
  (`homeNotes`, design D6); the footer lists Ctrl+N, Ctrl+O and Ctrl+Q
  (`homeKeys`). Ctrl+R is free there; on the dashboard it means "after a
  crash only: resume the same CLI session". At 80x24, the smallest size
  (`render.MinWidth`, `MinHeight`), a three-line note and a two-line one
  fit above START SESSION (Mockups, below).
- **The project store** (`internal/input/projects.go`, `projects.json`,
  version 1): per project its path, its last start and its star, "paths
  and times only, no conversation data". Every change re-reads the file and
  writes it through a temporary file renamed over it (`Update`), so another
  dashboard's entries are kept; `Same` compares paths cleaned, without case
  on Windows; `Record` rebuilds a project's entry, keeping only its star;
  ten recent projects are kept besides the favourites.

### Design (revised)

- **D1. The resumption only.** Plan 69 continues the CLI's session in a new
  process, across dashboard runs; zone B does not draw the earlier
  conversation (D6). Drawing Codex's earlier turns is a plan of its own,
  added as **[Nice to have]** at the approval: paged with
  `thread/turns/list` and `thread/items/list`, as Codex now asks, with its
  rendering cost, its tool events and very long sessions. Claude Code sends
  no earlier message (P4, P69a), and its transcripts are an internal format
  (AGENTS.md, Prefer documented interfaces). Not chosen either: a store of
  conversations of the dashboard's own (ASM-11).
- **D2. The record.** Each project's entry in `projects.json` keeps, per
  agent CLI, the last session the dashboard ran there: the ID the CLI
  reported (`event.Started`) and when the owner's last turn there ended,
  written then, whether the turn completed, was interrupted or failed. The
  dashboard's own usage read is no turn; a mock run records nothing. An
  optional field: version 1 kept, an older build reading the file as before
  (and dropping the field when it writes it). IDs and times only (ASM-24
  extended). Written through `Update`, so other dashboards' entries stay;
  `Record` and a star keep the entry's sessions; a project the list drops
  or forgets takes them with it.
- **D3. Matching.** Only the record of the chosen agent CLI in the chosen
  project is offered: the field's path, resolved as validation resolves
  it, the same project as an entry's path as `Same` compares them. Nothing
  else: not the other CLI's record (no provider switch), not a session the
  dashboard did not run (no `--continue`, no `thread/list`), not a
  repository's record for one of its worktrees (a worktree is a project of
  its own). With no matching record there is no note, and Ctrl+R does
  nothing on the home screen. A validated path that is no longer the
  record's project refuses the start, with the reason. The crash resume
  keeps its own path (D8).
- **D4. The home screen.** With a matching record, a grey note under the
  project field: "→ <CLI>'s last session here: <when> · Ctrl+R continues
  it", <when> as the project list writes a time. Ctrl+R, or a click on the
  note, validates the project as START SESSION does and starts that CLI on
  that session; START SESSION still starts a new one. The note follows the
  CLI and the project as they change, and is gone while a start is
  validated. A note of plan 51's design D6: no new picture, the footer and
  the artwork as they are (Mockups).
- **D5. Each CLI's own resume, bounded.** Claude Code: `--resume <id>` with
  a start's arguments; nothing of the history comes on stdout. Codex:
  `thread/resume` with the thread's ID, a start's parameters and
  `excludeTurns: true`, an answer of the same size whatever the thread's
  length. The session's context tells this resume from the crash resume,
  whose `thread/resume` stays as it is, the full history and Codex's
  deprecation notice with it (the owner's final review).
  `thread/goal/cleared`, which follows every resume, is logged as a status,
  as `account/updated` is. The model, effort and mode are what each CLI
  reports after the resume; the telemetry is D9's.
- **D6. Zone B, resumed.** Once the CLI confirms the session, zone B opens
  with a dashboard note, grey, as its other notes: "Resumed Claude Code
  session 21f3b775 · Claude Code has its earlier context; its earlier
  messages are not shown here", for Codex "Resumed Codex thread 01a18936 ·
  Codex has its earlier context; its earlier turns are not shown here".
  Confirmed is Claude Code's `system/init` with the ID asked for, Codex's
  `thread/resume` answer with that thread. Another ID is said as such
  ("Claude Code reports session <x>, not <y>: its earlier context may not
  be there") and becomes the record. While the CLI starts, zone B's
  starting line names the resume: "Resuming Claude Code session 21f3b775
  in <project>…".
- **D7. A resume that fails.** Never a fallback: a refused resume starts
  nothing, neither `thread/start` nor Claude Code without `--resume`; a new
  session starts only from START SESSION. Zone B's card says "The session
  could not be resumed", with the CLI's own words and what became of the
  record. Two outcomes:
  - **Refused for good**: the CLI says it has no such session, Claude
    Code's "No conversation found with session ID: <id>" or Codex's -32600
    "no rollout found for thread id <id>" or "invalid session id", for the
    ID asked for. The adapter reports it in the dashboard's words (a field
    of `event.Problem`), the CLIs' words labelled undocumented where they
    are matched. The record is cleared if it still names that session
    (another dashboard may have written a newer one). Back on the home
    screen: "✗ Not resumed: <CLI> no longer has its last session here",
    until the next start or another CLI or project is chosen.
  - **Any other failure**: the CLI not found or not starting, no handshake
    within 60 s, an exit or a refused request in other words, an
    authentication or connection error. The record is kept. Back on the
    home screen, the note as before and "✗ Not resumed: <reason>" under
    it; Ctrl+R tries again.

  A later CLI that words its refusal otherwise falls under the second
  outcome: the record kept and the failure shown, nothing lost.
- **D8. Nothing else.** The crash resume as it is, its requests and what it
  shows, but for `thread/goal/cleared` logged as a status (D5); one session
  per agent CLI and project, no list of sessions (**[Nice to have]** at the
  approval); no CLI configuration touched.
- **D9. Telemetry after a resume.** Each value as its CLI reports it, never
  kept in the record nor added up across processes. Codex's own replay of
  the thread's usage gives CONTEXT and TOKENS once the resume is answered,
  with or without the turns, and its totals after a turn hold the history
  (What was found); EST. COST stays N/A, as in any Codex session. Claude
  Code's values come with the first new turn's result, the history
  included. Until a value comes, its N/A says why as a resumed session has
  it: each reason's "yet" reads "since the resume" ("no turn since the
  resume", "no result since the resume"), and Codex's "no model call yet"
  reads "no update since the resume", which is also what a Codex that did
  not replay its usage would show, never a zero. A new session's reasons
  as they are. Not chosen: reading the restored totals from the usage
  read's result, undocumented, and zeros in a new session (P24u). Found
  by the live run (69.11): those reasons are too long for the usage
  section's three columns, which pushed EST. COST out; while they do not
  fit, the section takes its three rows, one value a row (ui-design.md,
  Transition).
- **D10. Parts.** Input: the store's field. State: a start that resumes
  (its ID through validation), the resumed note, the failure's two
  outcomes. Rendering: the home notes, zone B's note, starting line and
  card, a resumed session's N/A reasons. The application: Ctrl+R and the
  click on the home screen, the record's writes and its clearing. The
  adapters: Codex's `excludeTurns` for the home screen's resume, told
  apart in `provider.SessionContext`, and `thread/goal/cleared`; each
  CLI's refusal reported as for good.

### Mockups

The home screen as today's renderer draws it, the proposed notes put in
its existing note slots, a "→ " grey one and a "✗ " failure one (a scratch
test, deleted after); the text frames as the golden frames show them, the
picture and its banner as plan 51 draws them. The normal state at 80x24,
the smallest size, Claude Code chosen and its last session recorded for
the project:

```text
 ⊞ AGENT COMMAND CENTER                       │  v0.1.0  │  Windows  │  ● Ready

 ╭──────────────────────────────────╮ ╭───────────────────────────────────────╮
 │ › NEW SESSION  ───────────────── │ │ › PROJECTS  ───────────────────────── │
 │                                  │ │                                       │
 │  1.  Agent CLI                   │ │  ◷ RECENT PROJECTS                    │
 │ ▏ ◉  Claude Code (2.1.300)     ▕ │ │  ☆  agent-command-center         now  │
 │                                  │ │                                       │
 │   ○  OpenAI Codex (0.145.0)      │ │  ☆  billing-service           8m ago  │
 │                                  │ │ ───────────────────────────────────── │
 │  2.  Project directory           │ │  ☆  a-project-with-a-very-l… 58m ago  │
 │ ╭──────────────────────────┬───╮ │ │ ───────────────────────────────────── │
 │ │ …projets\billing-service │ 🗀 │ │ │  ☆  docs                      3h ago  │
 │ ╰──────────────────────────┴───╯ │ │ ───────────────────────────────────── │
 │                                  │ │  ☆  infra                     7h ago  │
 │  → Claude Code's last session    │ │ ───────────────────────────────────── │
 │    here: 3m ago · Ctrl+R         │ │  ☆  mobile-app               14h ago  │
 │    continues it                  │ │                                       │
 │                                  │ │                                       │
 │         START SESSION →          │ │                                       │
 │                                  │ │                                       │
 ╰──────────────────────────────────╯ ╰───────────────────────────────────────╯

 › Shortcuts:   Ctrl+N  New Session  │   Ctrl+O  Open Project  │   Ctrl+Q  Quit
```

The other states at 80x24, NEW SESSION's rows only (PROJECTS, the header
and the footer as above). No session: Codex chosen, none recorded for it
here, so no note and nothing to resume, Claude Code's record not offered.
Not resumed, any other failure: the record kept, the note still offered,
the reason under it. Not resumed, refused for good: the record cleared.

```text
 No session                              Not resumed, kept                       Not resumed, cleared
 ╭──────────────────────────────────╮    ╭──────────────────────────────────╮    ╭──────────────────────────────────╮
 │ › NEW SESSION  ───────────────── │    │ › NEW SESSION  ───────────────── │    │ › NEW SESSION  ───────────────── │
 │                                  │    │                                  │    │                                  │
 │  1.  Agent CLI                   │    │  1.  Agent CLI                   │    │  1.  Agent CLI                   │
 │   ○  Claude Code (2.1.300)       │    │ ▏ ◉  Claude Code (2.1.300)     ▕ │    │ ▏ ◉  Claude Code (2.1.300)     ▕ │
 │                                  │    │                                  │    │                                  │
 │ ▏ ◉  OpenAI Codex (0.145.0)    ▕ │    │   ○  OpenAI Codex (0.145.0)      │    │   ○  OpenAI Codex (0.145.0)      │
 │                                  │    │                                  │    │                                  │
 │  2.  Project directory           │    │  2.  Project directory           │    │  2.  Project directory           │
 │ ╭──────────────────────────┬───╮ │    │ ╭──────────────────────────┬───╮ │    │ ╭──────────────────────────┬───╮ │
 │ │ …projets\billing-service │ 🗀 │ │    │ │ …projets\billing-service │ 🗀 │ │    │ │ …projets\billing-service │ 🗀 │ │
 │ ╰──────────────────────────┴───╯ │    │ ╰──────────────────────────┴───╯ │    │ ╰──────────────────────────┴───╯ │
 │                                  │    │  → Claude Code's last session    │    │                                  │
 │                                  │    │    here: 3m ago · Ctrl+R         │    │  ✗ Not resumed: Claude Code no   │
 │         START SESSION →          │    │    continues it                  │    │    longer has its last session   │
 │                                  │    │  ✗ Not resumed: no handshake     │    │    here                          │
 │                                  │    │    within 60 s                   │    │                                  │
 │                                  │    │                                  │    │         START SESSION →          │
 │                                  │    │         START SESSION →          │    │                                  │
 ╰──────────────────────────────────╯    ╰──────────────────────────────────╯    ╰──────────────────────────────────╯
```

At 120x40 the note takes two rows, at 200x60 one (NEW SESSION's rows 10 to
17 at 120x40):

```text
 │ ╭────────────────────────────────────────────┬───╮ │
 │ │ C:\javadev\projets\billing-service         │ 🗀 │ │
 │ ╰────────────────────────────────────────────┴───╯ │
 │                                                    │
 │  → Claude Code's last session here: 3m ago ·       │
 │    Ctrl+R continues it                             │
 │                                                    │
 │                  START SESSION →                   │
```

Zone B, proposed lines (not rendered yet): the resumed note, then a
refused and a kept failure's card, in the failure card's form.

```text
 · Resumed Claude Code session 21f3b775 · Claude Code has its earlier
   context; its earlier messages are not shown here

 ✗ The session could not be resumed
   Claude Code: No conversation found with session ID: 21f3b775-162f-47e1-bb16-aa91f8a808b9
   Claude Code no longer has this session: it is no longer offered
   exit code 1
   Esc back to startup (answers kept) · Ctrl+Q quit

 ✗ The session could not be resumed
   no handshake within 60 s
   The session stays offered: Ctrl+R on the startup screen tries again
   Esc back to startup (answers kept) · Ctrl+Q quit
```

### Criteria (revised)

1. **69.1 The probes** (step 2, run on 2026-10-29 before any code): P69a
   and P69b as found above, four model calls (two Haiku, two Codex) in
   temporary projects outside the repository, the configuration check PASS
   before and after (`doc/research/probes/README.md`). No central
   assumption of the design contradicted; they refined D5 (`excludeTurns`,
   `thread/goal/cleared`) and D7 (each CLI's words for a refusal).
2. **69.2 The record** (the store's tests): written when the owner's turn
   ends, completed, interrupted or failed, per agent CLI and project, with
   the CLI's ID and the time; not by the usage read, nor in a mock run;
   kept by `Record` and by a star; written through `Update`, another
   dashboard's entries kept; a `projects.json` without it read as before;
   gone with its project.
3. **69.3 Matching and restart** (application tests): a new Model on the
   same store, as after a full restart of the application, offers the
   record; only for the chosen agent CLI and the chosen project, a path
   differing in case or separators the same project on Windows; never for
   the other CLI, another project or a worktree of it; Ctrl+R with no
   matching record does nothing; a validated path that is no longer the
   record's project refuses the start with the reason.
4. **69.4 The note and the key** (application tests; golden frames at
   80x24, 120x40 and 200x60): the note for each CLI, its time as the list
   writes one, none while a start is validated; Ctrl+R and a click on the
   note validate and start the CLI on the session (the fakes see
   `--resume <id>`, and `thread/resume` with its `threadId` and
   `excludeTurns: true`); START SESSION still starts a new session; the
   other startup frames unchanged.
5. **69.5 The resume, bounded** (adapter tests on P69a's and P69b's
   recordings): the home screen's `thread/resume` with `excludeTurns:
   true`; the crash resume's as today, its tests' expected requests
   unchanged; `thread/goal/cleared` a status, so a resumed session has no
   unrecognised line; Claude Code's arguments a start's with
   `--resume <id>`.
6. **69.6 Zone B resumed**: the note once each CLI confirms the session;
   another ID said as such and recorded; the starting line naming the
   resume; then the turns as usual.
7. **69.7 Telemetry after a resume** (tests replaying P69a's and P69b's
   recordings through the mappers, the state and zone A's pane): Codex
   resumed without its turns, once the answer and its replay are in:
   CONTEXT "6% of 258.4k", TOKENS "16.4k" and EST. COST N/A "Codex reports
   none", the values of the resume with its turns; after the first new
   turn, TOKENS "32.9k", CONTEXT "6% of 258.4k" and the turn's duration;
   the replay's line taken out, CONTEXT and TOKENS N/A "no update since the
   resume" until that turn, then "32.9k". Claude Code: CONTEXT N/A "no turn
   since the resume", TOKENS and EST. COST N/A "no result since the
   resume" until the first new turn's result, then "4% of 1M", "85.3k" and
   "$0.01". A new session's reasons unchanged.
8. **69.8 A resume that fails** (adapter, state and application tests):
   each CLI's refusal as recorded in P69a and P69b clears the record only
   while it names that session, the card and the home note saying so;
   every other failure (the CLI not starting, no handshake, an exit or a
   refusal in other words) keeps the record, the card and the home note
   saying why; after a refused resume, no `thread/start` and no Claude Code
   without `--resume`; START SESSION still starts a new session.
9. **69.9 Nothing else**: the crash resume's requests and tests, the other
   screens' golden frames and every existing test unchanged and passing.
10. **69.10 Windows Terminal** (the acc tab maximised and at 120x40, the
    fakes, no quota): the dashboard quit and started again, the note at
    both sizes, Ctrl+R and the click starting the resume, a refused
    resume's card and home note, a kept one's.
11. **69.11 Live** (proposed with the plan: a temporary project outside the
    repository, the configuration check before and after; for each agent
    CLI two turns, Claude Code on Haiku, Codex on its configured model;
    stopped after the code word's answer, or at 5 minutes a CLI): a session
    started from the home screen and told a code word, the dashboard quit
    and started again, Ctrl+R on the home screen, zone B's note, zone A's
    values as 69.7 expects them (Codex's once the resume is answered, both
    CLIs' after the turn), the code word answered.
12. **69.12 The checks**: build, vet (Windows, Linux), gofmt and the whole
    suite; the change broken on purpose through `go test -overlay`, each
    mutant caught; `-race` on the changed packages' tests; the timing step
    alone, on mains power.
13. **69.13 The documents**: ui-design.md (the home screen's note and key,
    zone B's note, starting line and card), events-and-state.md (the start
    that resumes, the failure's two outcomes), reliability.md (Resumption
    across dashboard runs, the refusal words labelled undocumented;
    Telemetry sources, a resumed session's values and reasons),
    architecture.md (ASM-24, the parts, `excludeTurns`), codex-cli.md
    (finding 4) and telemetry-sources.md (Tokens), whose usage replay P69b
    now records, brief amendment 71, and at the approval AGENTS.md's MVP
    scope and status and the two **[Nice to have]** entries.

### Steps

1. Amendment 71 and this plan, for the owner's review; revised after each
   review; approved on 2026-10-29.
2. The probes, P69a and P69b: 69.1 (done).
3. The record, matching, the note and the key, the resume, zone B, the
   telemetry, the failures: 69.2 to 69.9.
4. Windows Terminal, then the live run: 69.10, 69.11.
5. The checks, the mutants, the timing and `doc/plan69-evaluation.md`
   (69.12, 69.13); the commit after the owner's review.

### Progress

- 2026-10-29: step 1: amendment 71 and this plan, proposed for the
  owner's review, with the findings above (from the repository's
  research, probe recordings and schema; no CLI run).
- 2026-10-29: the owner's design review: conditional approval, 8.5/10,
  four points (the resumption alone; transient failures kept apart from a
  confirmed refusal, with no fallback; matching and a full restart; a
  mockup), the minimal probes first (amendment 71). Step 2 done: P69a
  (20:48 to 20:49) and P69b (about 20:50), two Haiku calls and two Codex
  calls, configuration PASS, recorded with their drivers in
  `doc/research/probes`; no central assumption contradicted. The plan
  revised around the four points, the mockups drawn by today's renderer;
  for the owner's final design approval. No code changed.
- 2026-10-29: the owner's final review: every design decision accepted
  but one detail, Codex's telemetry after a resume without its turns, and
  the crash resume kept as it is (amendment 71). Settled from P69a's and
  P69b's recordings, no new probe: their lines replayed through the
  mappers, the state and zone A's pane (a scratch test, deleted after)
  give Codex's values once the resume is answered the same with or
  without the turns, both CLIs' totals after the first new turn with the
  history, and N/A reasons a resumed session makes untrue (D9, criterion
  69.7). `excludeTurns` for the home screen's resume only (D5). For the
  owner's implementation approval. No code changed.
- 2026-10-29: the owner approved plan 69 for implementation, D9 and the
  telemetry criteria included, the crash resume unchanged (amendment 71):
  AGENTS.md's status and MVP scope, and the two **[Nice to have]**
  entries, written. Step 3 begun.
- 2026-10-29: step 3 done: the store's field (`internal/input/projects.go`),
  the adapters (Claude Code's refusal for the home screen's resume; Codex's
  `excludeTurns` for it alone, its refusal, `thread/goal/cleared`), the
  state (the start that resumes, zone B's note, the two failures, the
  record at the owner's turn end), the home screen's note and its click,
  zone B's starting line and card, a resumed session's N/A reasons, and
  Ctrl+R (`internal/app/lastsession.go`); P69a and P69b copied to
  `testdata/probes`; the tests of 69.2 to 69.9 (`TestProjectSessions`,
  `TestResumeRefusal`, `TestResumeLastWired`, `TestResumeLast`,
  `TestResumeLastRefused`, `TestResumeRefusalP69b`, `TestGoalCleared`,
  `TestRecordSession`, the machines' rows, `TestLastSession*`,
  `TestResumedTelemetry`) and four golden frames (`startup-resume-*`, the
  only frames that changed); two counts of the recordings moved with them
  (`TestStreamEventsTakeTheMessageStartID` 1,454 stream events in 16 of 20,
  `TestReplayNotes`); the documents of 69.13. The whole suite, vet and
  gofmt pass.
- 2026-10-29: step 4: Windows Terminal (69.10), ACC's profile maximised and
  at 120x40, a stand-in `claude.exe`, no quota: the note's click target at
  120x40, its line's band less than a row high, rounded to no row; fixed
  (the rows the line touches, `TestResumeNoteTarget`), then 16 runs pass.
  The live run (69.11), in its scope: Claude Code's first run found zone
  A's three columns pushing EST. COST out once resumed (L69.4 FAIL); fixed
  (the section's three rows while a resumed session's reasons do not fit
  them, D9; `TestResumedUsageRows`); Claude Code run again, two more Haiku
  turns, and Codex: 6 PASS each, configuration PASS, the owner's
  `projects.json` put back each time.
- 2026-10-30: step 5: 57 mutants through `go test -overlay`; the first runs'
  survivors asked for six tests and two untested cases of the criteria (a
  worktree of the project, another dashboard's entries kept), and showed
  two guards that changed nothing, removed (`internal/app/lastsession.go`);
  on the final code and tests all 57 caught. The whole suite (16 packages)
  after the last code edit, `-race` (0 reports), Windows Terminal's 16
  runs again on the final build, the timing step alone (21 PASS).
  [`doc/plan69-evaluation.md`](doc/plan69-evaluation.md): 13 PASS, with
  four findings outside plan 69, for the owner's review with the diff.
  Nothing committed.
- 2026-10-30: the owner's acceptance run, the manual checklist in Windows
  Terminal on both CLIs (two Haiku turns, two Codex turns): every scenario
  PASS, two UI findings, both older than plan 69 (3d395a8). P69-M01, a
  validation failure left beside the offer of a project then chosen in the
  list: the failure now shows while the field holds the text that failed
  (`internal/render/home.go`). P69-M02, the project field blank after
  Windows Terminal's font zoom: `TEA_TRACE` showed the vacated images'
  cells written over at their old size past the smaller screen, each wrap
  on its last row scrolling it; now cut to the screen
  (`internal/app/preview.go`), which also ends the sidebar tiles' writes
  past 99 columns that `TestSideDraw` asserted (its expectation changed,
  for the owner's review). `TestHomeFailureFollowsField` and
  `TestHomeShrink`, each seen failing first; 9 mutants, all caught once
  `TestHomeShrink` also checked that no old place is cut short; the whole suite,
  `-race`, the timing step; Windows Terminal before and after (the
  stand-in, no quota). A correction to 69.13 (`TestDocFrames` reads
  nothing) and a fifth finding (the Layouts frames lack the Git Graph's
  tile since plan 62). [`doc/plan69-evaluation.md`](doc/plan69-evaluation.md),
  The owner's acceptance run: 4 PASS, for the owner's check. Nothing
  committed.
- 2026-10-30: the owner's regression run of the two fixes in Windows
  Terminal, on the final code, no model turn: both PASS (the original
  M01 scenario; zooms maximised, Ctrl+0, and 120x40 zoomed in nine steps,
  Ctrl+R resuming the real session there, which also verifies grid mode).
  The evaluation records it. The final checks apply to the final tree: its
  code hash `b54e3362` unchanged, the binary rebuilt from it identical to
  the one tested, the documents the tests read unchanged since the last
  suite. For the owner's decision on the commits.
- 2026-10-30: the owner's final acceptance report, on the corrected build,
  no model prompt: PASS for every scenario run. The window's edge dragged
  with a resumed session (about 950 to 770 pixels and back); after a
  resize, Source Control and its commit controls, the models popup's
  Effort slider, Ultracode toggle and mode pill, and an image thumbnail;
  and a real Claude Code 2.1.303 refusal through the dashboard, the
  registry's record pointed at a nonexistent ID and restored after from a
  verified backup: the card, the record cleared, nothing offered after.
  Codex's refusal stays on P69b's recordings and the tests. The evaluation
  records it; the owner approved plan 69 for closure, the commits, the
  push and the test data's cleanup.

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

## Plan 27 - Zone C with the execution timeline

Brief amendments 26 and 27: the owner's design of zone C and its
close-ups, to respect strictly. Zone C from 100 columns: the header
inside the frame, a rule and the panels' line in the title colour, the
agents' panel as wide as its two-column cards (the label over the clock
and the tokens, a grey bar, the model over the current tool; the selected
agent's rows on a band) and the selected agent's execution timeline with
its status at the heading's right end; the narrow layout keeps plan 18's
row. Approved on 2026-10-22: 15 of 16 PASS, 27.6 corrected with a note
at the owner's decision, 46 golden frames, Windows Terminal screenshots
at 208x59 and 120x40 against the images
([`doc/plan27-evaluation.md`](doc/plan27-evaluation.md)); commit 041e7b8,
then the approval commit.

## Plan 28 - The top right after the owner's image

Brief amendment 28: the owner's close-up of the top right, "● LIVE" over
"Claude Code 2.1.303" over zone B's edge with "Idle". The project bar's
right end is the provider's name and version alone while approvals are
Manual with no change asked; the mode shows there only as Auto or while
changing. Approved on 2026-10-22: 6 of 6 PASS, 27 golden frames, a
Windows Terminal screenshot against the image
([`doc/plan28-evaluation.md`](doc/plan28-evaluation.md)); commit 8fe0077,
then the approval commit.

## Plan 29 - The input editor's row after the owner's image

Brief amendment 29: the owner's image of the editor's row, design only.
Under the draft a separator, then " + " and the pill " <model> High ·
Ultracode " on the tool rows' ground, the editor's messages and the line
count in the free middle, "↯ Auto" at the right end; the "+", "High ·
Ultracode" and "Auto" are placeholders with no function. Zone B is one
row shorter. Approved on 2026-10-22: 7 of 7 PASS, 59 golden frames,
Windows Terminal screenshots at 208x59 and 120x40 against the image
([`doc/plan29-evaluation.md`](doc/plan29-evaluation.md)); commit d696f22,
then the approval commit.

## Plan 30 - The "+" menu after the owner's image

Brief amendment 30: the owner's image of the "+" clicked, design first. A
click on the editor row's band opens a three-row box above the editor's
rule, "🗀  Upload from computer", the band and the item under the
pointer on the active ground (`#272745`), the terminal asked for every
mouse motion only while it is open; Esc, any key or a click elsewhere
closes it; the item does nothing yet, the upload being a later plan's.
Approved on 2026-10-22: 6 of 6 PASS, four golden frames added, Windows
Terminal screenshots with a real click and pointer moves at 208x59 and
120x40 against the image
([`doc/plan30-evaluation.md`](doc/plan30-evaluation.md)); commit 6025513,
then the approval commit.

## Plan 31 - Image upload

Brief amendment 31: the "+" menu's "Upload from computer" works. The
system's Open dialog, several images at once, each checked by its bytes
(PNG, JPEG, GIF, BMP, TIFF, ICO and WebP accepted, the orientation
applied, conversions to PNG, an image over a provider's limits scaled
down and said so; HEIC, AVIF, JPEG XL, JPEG 2000, PSD, TGA, DDS, HDR,
EXR, camera raws and SVG refused by name); chips under the editor's
rule, removable; sent only when the owner submits, as base64 image
blocks to Claude Code and `image` inputs holding data URLs to Codex, no
event keeping the bytes (Codex writes them back). Approved on
2026-10-23: 9 of 9 PASS, eight golden frames added, the real dialog in
Windows Terminal, live turns on Claude Code (Haiku 5.5) and Codex
(gpt-6-sol) naming the colours and the text, and a third on Codex
checking the echo fix
([`doc/plan31-evaluation.md`](doc/plan31-evaluation.md)); commit 8e2a721,
then the approval commit.

## Plan 32 - Image paste and preview

Brief amendment 32: paste an image as in the owner's chat, and a preview
before sending. Alt+V attaches the clipboard's images (copied files, the
"PNG" format's bytes, else the bitmap as PNG); Ctrl+V pastes text as
before and, when Windows Terminal keeps its press for want of text,
attaches the image on the key's lone release (undocumented, probe P32a);
a paste of the copied files' paths attaches them when all are images;
a pasted image is held in memory only. With Sixel graphics, the images
show as thumbnails between the rule and the chips, rounded squares cut
from their centre, 14x7 cells from 50 rows, 10x5 from 30, side by side
one cell apart (the owner's remark on the screenshots), drawn after the
frame and again after any frame that changes their rows, a vacated
place written over with the frame's cells. Approved on 2026-10-23: 10 of
10 PASS, four golden frames added, Windows Terminal at 208x59 and
120x40 with the owner's clipboard saved and restored, the remark in a
follow-up ([`doc/plan32-evaluation.md`](doc/plan32-evaluation.md));
commit b56d23e, then the approval commit.

## Plan 33 - Unsent drafts and images through crashes and restarts

Brief amendment 33: whether unsent image attachments survive a crash, a
forced termination, a restart and a resumed session; an investigation
first, then its nine corrections. Each dashboard's draft has a folder of
its own, its lock held while the dashboard owns it, taken back by the
project's next start (its own CLI's first, then the newest); images are
saved at once, the text every 500 ms; pasted images are kept as
`<sha256>.png` until their message is accepted; a message waiting for
its acceptance is stored with what was typed since; Back forgets the
attachments; a damaged list is set aside; a changed file is attached
again as it is now; refused renames and failed saves are tried again;
unused files are swept at each start. An independent review found five
defects, fixed; on the owner's questions, a Codex session resumed with
images, a slow and an offline share were simulated, fixed where they
lost data, and a logoff was not simulated. Approved on 2026-10-23: 14 of
14 PASS and the follow-up
([`doc/plan33-evaluation.md`](doc/plan33-evaluation.md)); commit
bc5d92b, then the approval commit.

## Plan 34 - The model selector, its effort and Ultracode

Brief amendment 34: a click on the editor row's model pill, or Alt+P,
opens the active CLI's own models with the effort and, for Claude Code,
Ultracode, after the owner's image. Claude Code's list comes from
`initialize`, its changes go by `set_model` and `apply_flag_settings`
and are read back by the undocumented `get_settings`; Codex's list from
every page of `model/list`, its changes ride the next `turn/start`,
read back by `thread/read`. The screen shows a value only once the CLI
reports it in force; an effort held by `CLAUDE_CODE_EFFORT_LEVEL` is
shown and never changed; Claude Code's changes wait for a turn's end. An
independent review found four defects and six doubts, fixed, and three
small points on its second pass. At the owner's request
`CLAUDE_CODE_EFFORT_LEVEL` was removed from their Claude Code settings,
and the dashboard then set the effort live; amendment 35, an opt-in line
telling the agent its settings, was built and withdrawn. Approved on
2026-10-24: 20 of 21 PASS, 34.18 (golden frames changed beyond the
pill's values) accepted
([`doc/plan34-evaluation.md`](doc/plan34-evaluation.md)); commit
47b50ce, then the approval commit.

## Plan 35 - The Ultracode toggle after the owner's image

Brief amendment 36: the owner's image of the Ultracode toggle, a lavender
pill with fully rounded ends and a round white knob, against the
dashboard's `▐   ●▌`. Where the terminal draws Sixel, the toggle is an
image over blank cells of its row, drawn as the preview's thumbnails are:
a pill of 40x20 pixels in an image of five cells whose last carries the
band, since the renderer writes every cell between a row's first and last
changes and a band cell after the image made it blink when the band came
or went (bursts of captures in Windows Terminal); without Sixel, a text
pill of Powerline's half circles. Approved on 2026-10-24: 8 of 8 PASS
([`doc/plan35-evaluation.md`](doc/plan35-evaluation.md)); commit
b43cf46, then the approval commit.

## Plan 36 - The selector's cursor and a pointer that has not moved

Brief amendment 37: Alt+P could reopen the model selector with its
cursor on the row under a resting pointer, so that Enter acted on that
row. The probe (Bubble Tea's `TEA_TRACE`) showed Windows Terminal
reporting a pointer put down by software, without its moving, once the
selector asked for every motion, its reports wandering a cell. The
selector now moves its cursor only when the pointer comes onto another
row than the last report's, the first report after Alt+P only recording
where it rests. Approved on 2026-10-24: 7 of 7 PASS
([`doc/plan36-evaluation.md`](doc/plan36-evaluation.md)); commit
c0cebc4, then the approval commit.

## Plan 37 - The effort slider after the owner's images

Brief amendment 38: the owner's five images of one pill-shaped effort
slider at five levels replaced the selector's effort dropdown. Effort
reads "Effort (<level>)", "Controls reasoning depth" and the slider: the
model's own levels as dots, a fill of `#6784C1` growing to the thumb,
the whole pill lit by a purple gradient with specks at the highest
level, a level asked for ringed, a held effort greyed; with Sixel an
image of 170x20 pixels over 17 blank cells, its thumb gliding in 180 ms;
without, 16 text cells. Effort and Ultracode share label, description
and control columns; ← → and a click on the slider ask for a level.
Approved on 2026-10-24: 9 of 9 PASS
([`doc/plan37-evaluation.md`](doc/plan37-evaluation.md)); commit
31c22ef, then the approval commit.

## Plan 38 - Instant Codex model and effort changes, as in VS Code

Brief amendment 39: a Codex model or effort change took effect only with
the next message, where VS Code's Codex extension changes it at once. The
dashboard now asks for Codex's experimental API at initialize, as that
extension does, and writes a change between turns at once with
`thread/settings/update`, the `thread/settings/updated` that follows
putting it in force (P38x: 1 to 3 ms); a change asked during a turn is
written at its end; a Codex that refuses the update takes every change
with the next message, as before. Approved on 2026-10-24: 12 of 12 PASS
([`doc/plan38-evaluation.md`](doc/plan38-evaluation.md)); commit
dc9c6d9, then the approval commit.

## Plan 39 - The permission modes selector after the owner's images

Brief amendment 40: the editor row's right-hand "Auto" became the mode
pill of the owner's third image, and opens a "MODES" popup after the
first two: the active CLI's own modes (Claude Code's Manual, Edit
automatically, Plan and Auto; Codex's Ask for approval, Approve for me and
Full access), each applied through the CLI (`set_permission_mode`; the
experimental `thread/settings/update` with the presets of VS Code's Codex
extension) and shown in force only once the CLI reports it; Codex's Full
access only after a confirmation. The icons, the check and the band's
rounded ends are Sixel strips beside the text cells, the pill a Sixel
image whose title Windows' GDI draws. Approved on 2026-10-24: 15 of 15
PASS ([`doc/plan39-evaluation.md`](doc/plan39-evaluation.md)); commit
55e2bed, then the approval commit.

## Plan 40 - No empty rows in zone C's execution timeline

Brief amendment 41: on Codex, zone C's execution timeline and the
inspection's Messages tab showed a subagent's rows "assistant │" with no
text, one for each of its reasoning steps: since plan 22 a subagent's
reasoning summaries leave assistant entries holding only a thinking mark,
which zone B never drew. Neither draws an entry with no text now (a
thinking mark, a message that ended empty, a notice with none), a rule
only between two drawn entries. Approved on 2026-10-24: 6 of 6 PASS
([`doc/plan40-evaluation.md`](doc/plan40-evaluation.md)); commit
4158751, then the approval commit.

## Plan 41 - No authorization on low-risk review rows

Brief amendment 42: a Codex auto-review row rated low risk read
"authorization unknown": Codex's reviewer answers a low-risk action with
a bare allow, and Codex then reports the authorization as unknown with a
stock rationale. After a preview, the owner chose to drop the
authorization from review rows rated low risk, in zone B, zone C's
timeline and the inspection; at medium, high and critical it stays.
Approved on 2026-10-24: 7 of 7 PASS
([`doc/plan41-evaluation.md`](doc/plan41-evaluation.md)); commit
3904f13, then the approval commit.

## Plan 42 - The mouse in zone B: a click to fold, a double click to open

Brief amendment 43: beside the keyboard's path, which the owner approved
and which stays as it is, a click anywhere on a foldable item's row in
zone B (a run of calls, a thinking phase's summary row) unfolds it or
folds it again, and a double click on a row of an entry Enter shows whole
opens its full view as Enter does, an approval's rows the approval whole
as from its card. The mouse puts the entry cursor nowhere and shows none
of the keyboard's hints. On those rows a double click no longer selects a
word; a drag still does. Approved on 2026-10-24: 7 of 7 PASS
([`doc/plan42-evaluation.md`](doc/plan42-evaluation.md)); commit
11f794d, then the approval commit.

## Plan 43 - A right activity sidebar with the Explorer docked

Brief amendment 44: a narrow activity sidebar right of zone B, from 100
columns, with the Explorer's and Skills' icons after the owner's two
images. The Explorer, a floating window since plan 17, is a panel docked
between zone B and the sidebar where zone B keeps 64 columns, and in zone
B's place where it does not and under 100 columns; F3 and the Explorer's
icon show and hide that one panel, the focus with it, F3 keeping its file
meaning. The icons' four looks are Sixel images, their hover from
every motion reported, filtered before Update; Skills is a placeholder.
The review led to a third patch of the vendored ultraviolet: a row
holding a wide character is written only as far as it changed, so the
icons beside it stay. Approved on 2026-10-25: 15 of 15 PASS
([`doc/plan43-evaluation.md`](doc/plan43-evaluation.md)); commit
48bde52, then the approval commit.

## Plan 44 - Source Control and a Git diff viewer

Brief amendment 45: the sidebar's second icon, Skills' placeholder until
then, is Source Control. Its panel shares the Explorer's docked place and
lists the project's Git changes from one git status read, the Explorer's
too: MERGE CHANGES, CHANGES and STAGED CHANGES, each file with its real
status, a file staged and changed again in both. A file opens its diff in
zone B's place, the panel beside it from 118 columns: git's blobs read raw
by cat-file and the working file, compared by a Myers diff of our own,
unified or side by side, with line numbers, green and red rows and
navigation between changes; read only. The engine was built by an
implementer subagent. Approved on 2026-10-25: 14 of 14 PASS
([`doc/plan44-evaluation.md`](doc/plan44-evaluation.md)); commit
d9d7d83, then the approval commit.

## Plan 45 - Commit, Push and Commit & Push in Source Control

Brief amendment 46: under Source Control's repository row, a message
field, a purple "✓ Commit" button with a separator and a chevron, and a
dropdown with one item, Commit & Push, after the owner's image. Commit
takes the index as staged (asking before staging anything), refuses a
staged set other than the one shown and counts once HEAD is a new commit
on the old one; Push goes to the configured upstream only, never forced;
the main action is read from git's branch line. After the first report
the owner asked for two additions: the staged files' contents compared
too, and a long operation's elapsed time with Esc cancelling a push (a
Ctrl+Break to git's own process group, behind a check of its lock files;
probes P45a to P45e). The engine's first version was built by an
implementer subagent. Approved on 2026-10-25: 19 of 19 PASS
([`doc/plan45-evaluation.md`](doc/plan45-evaluation.md)); commit
f71e022, then the approval commit.

## Plan 46 - Three fixes from the audit

Brief amendment 47: the first three fixes of the read-only audit of
2026-10-25 ([`doc/audit.md`](doc/audit.md)), each test-first.
- The worktree merge reads the main worktree first, refuses a merge in
  progress or uncommitted changes, and aborts only its own conflict
  (MERGE_HEAD the branch's tip).
- A refused Codex `config/read` or `model/list` no longer ends the
  handshake (`event.Problem.Optional`).
- An opened untracked folder in Source Control is read again after each
  status read and each opening.

Approved on 2026-10-25: 10 of 10 PASS
([`doc/plan46-evaluation.md`](doc/plan46-evaluation.md)); commit 9375233,
then the approval commit.

## Plan 47 - The rest of the audit's phase 1

Brief amendment 48: the rest of phase 1 of the audit's roadmap
([`doc/audit.md`](doc/audit.md), Part 2, C), each fix test-first.
- Ctrl+Q while a commit, push or staging runs asks to confirm, naming
  it.
- An answer to a request the CLI withdrew first expires the approval and
  the session goes on (`provider.ErrNoPending`).
- The reducer's Mode case returns, so mode events leave no false note,
  and a mock replay of Codex declines in Codex's word.
- The Explorer opened over the staging question drops it.
- `realPath` strips the `\\?\` prefix; two stale comments are corrected.
- `TestStateMachines`, failing since plan 46's commit, is repaired.

Approved on 2026-10-25: 10 of 10 PASS
([`doc/plan47-evaluation.md`](doc/plan47-evaluation.md)); commit d44bc0f,
then the approval commit.

## Plan 48 - The audit's phase 2: robustness at the boundaries

Brief amendment 49: phase 2 of the audit's roadmap
([`doc/audit.md`](doc/audit.md), Part 2, C), each fix test-first.
- The TIFF and ICO readers are bounded: the tags read from the file when
  used, an icon's PNG size checked before decoding.
- A running command's streamed output keeps its last 64 KiB, the rest
  counted and said.
- The Claude Code reader never writes to stdin.
- Git calls are bounded by `WaitDelay` once git has ended.
- A panic in the dashboard's own goroutines ends its session in order
  (`Fatal`, `Panicked`); fuzz targets for the image readers and both
  mappers.
- Bidi controls and zero-width spaces are drawn as `<U+XXXX>` on the
  approval card.

Approved on 2026-10-25: 13 of 14 PASS; 48.5 FAIL, its prediction about
the old code, accepted as a documented exception
([`doc/plan48-evaluation.md`](doc/plan48-evaluation.md)); commit 067aeb4,
then the approval commit.

## Plan 49 - The rest of phase 0, and phases 3 and 4 of the audit

Brief amendment 50: the rest of phase 0 of the audit's roadmap and its
phases 3 and 4 ([`doc/audit.md`](doc/audit.md), Part 2, C).
- Phase 0: the event queue's and the write queue's tests, and a matrix of
  the ten overlays alone and in their 90 ordered pairs.
- Tool rows memoised and long sessions' scans bounded (`render.Memo`,
  `State.ToolOrder`, `State.ParentRev`, zone A's rows made when shown);
  `TestToolsToFrame` joins the timing step.
- One owner of the overlays (`render.Layers`), the matrix's five defects
  fixed.
- The adapters' shared lifecycle (`pipe.Proc`).
- The git tests in parallel, git's environment set once in `TestMain`.
- Normalised approval decisions (`event.Decision`), each adapter writing
  its CLI's words byte for byte as before.
- Widths by grapheme cluster (`render/cluster.go`).
- A GIF's first frame alone decoded, its other frames walked by their
  blocks.

Approved on 2026-10-25: 11 of 16 PASS; 49.7 and 49.10, short of their
targets, and 49.8, 49.9 and 49.11, on tests that changed how they call
the code, FAIL, accepted as documented exceptions
([`doc/plan49-evaluation.md`](doc/plan49-evaluation.md)); commit 5c06d39,
then the approval commit.

## Plan 50 - Source Control multi-selection

Brief amendment 51: Ctrl+click selects files in Source Control, a "▎" on
each selected row and "Commit N files" on the button, and Commit and
Commit & Push commit them alone, by `git commit --only`: each selected
file staged whole, a partly staged one refused, unstaged and untracked
ones staged only after the question's yes, the rest of the index left as
it is. Windows Terminal passes Ctrl with a click, never Shift. The
owner's question before the approval of the plan: when the commit fails
after the question's staging, the staging stays, as git leaves it, and
the note says so. After the first evaluation (13 of 13 PASS) the owner
asked that a plain click on a file clear the selection, Commit being the
whole index as before with none selected, and that the note say
"changes"; the review of the complete diff then found a file moved on
disk, both its rows selected, refused once staged, fixed by reading
git's status path by path. Approved on 2026-10-26: 14 of 14 PASS
([`doc/plan50-evaluation.md`](doc/plan50-evaluation.md)); commit
b9f8323, then the approval commit.

## Plan 51 - Home screen redesign

Brief amendment 52: the home screen drawn as the owner's image,
`references/16-home-screen.webp`: picture mode (Sixel images, their text
by GDI in Cascadia Mono) from 118 columns with Sixel, grid mode otherwise;
the project field's path terminal text, its editing, selection, copy and
paste as before; an animated Space Invaders banner, ten frames a second,
only the pixels that change sent, in a fixed palette, paused without the
focus and hidden on small screens; Ctrl+N resets the form, the mouse
works the screen, Ctrl+P is left out. After the evaluation the owner
asked for a livelier pace, then twice for a faster ship (the march every
0.4 s, the ship's legs at 125 to 275 design pixels a second; a dash with
afterimages tried and withdrawn). Approved on 2026-10-27: 10 of 11 PASS
at the evaluation; 51.1, the field's box up to half a row from the
design's place, FAIL, accepted as a documented exception; 51.7 measured
again on a quiet machine, PASS; 51.9's paste back checked by the owner
([`doc/plan51-evaluation.md`](doc/plan51-evaluation.md)); commit
7e60f73, then the approval commit.

## Plan 52 - The system's folder dialog for the project

Brief amendment 53: the home screen's folder button and Ctrl+O open the
system's dialog for choosing a folder (IFileOpenDialog, folders only),
over the terminal, at the field's directory, else its nearest existing
parent, else the starting folder; the folder chosen goes into the field,
which takes the focus, Cancel changes nothing, a failure is a note. Plan
15's project browser removed at the owner's choice, `realPath` and
`dosPath` kept in `input/realpath_*.go`. Approved on 2026-10-27: 5 of 5
PASS ([`doc/plan52-evaluation.md`](doc/plan52-evaluation.md)); commit
9b8b6e5, then the approval commit.

## Plan 53 - A project chosen in the list fills the field

Brief amendment 54: choosing a project in the home screen's PROJECTS
list, by up and down or a click on its row or star, puts its path into
the Project directory field, so that the field shows what Start uses;
Tab onto the list, f and Del leave the field as it is. Approved on
2026-10-27: 5 of 5 PASS
([`doc/plan53-evaluation.md`](doc/plan53-evaluation.md)); commit
1818367, then the approval commit, both pushed.

## Plan 54 - The mouse wheel over the project list

Brief amendment 55: the wheel over the home screen's PROJECTS panel, its
rows, headings and frame, moves the list's selection one project a notch,
as up and down do, choosing it (its path into the field) and giving the
list the focus; the panel's other cells carry `HomeItemList`. Approved on
2026-10-27: 4 of 4 PASS
([`doc/plan54-evaluation.md`](doc/plan54-evaluation.md)); commit
38ee6d7, then the approval commit, both pushed.

## Plan 55 - Nerd Font icons in the Explorer

Brief amendment 56: in the acc tab, with Cascadia Mono NF installed for
the owner (release 2407.24, per user, registered from Windows Terminal:
the Claude app's shells write the registry into its private copy), the
Explorer's rows draw Nerd Font icons in each kind's colour and the
codicons' chevrons; anywhere else (another terminal or profile, no font,
a terminal started before it) the panel's own glyphs. The launcher names
the face in the acc tab's fragment by the same rule. Approved on
2026-10-27: 5 PASS, 0 FAIL, and 55.5's visual review by the owner
([`doc/plan55-evaluation.md`](doc/plan55-evaluation.md)); commit
374c23e, then the approval commit, both pushed.

## Plan 56 - The Explorer's font installed by acc

Brief amendments 57 and 59: Cascadia Mono NF bundled in the repository as
released (`third_party/fonts`, the SIL OFL 1.1 with its symbols'
notices) and installed by acc for the user, under Windows Terminal, when
Windows has no font of that name: the files checked against
`SHA256SUMS`, registered under HKCU, one line printed; a failure one
line, the launch going on; plan 55's rule unchanged, the icons from
Terminal's next start. Withdrawn on 2026-10-28 for plan 57 and reinstated
the same day. Approved on 2026-10-28 without repeating its end-to-end
test, run with the owner on 2026-10-27: 6 PASS, 0 FAIL
([`doc/plan56-evaluation.md`](doc/plan56-evaluation.md)); commit
d88c392, then the approval commit, both pushed; its full text in git
(`git show d88c392:plan.md`).

## Plan 57 - Segoe Fluent Icons in the Explorer

Brief amendment 58: the Explorer's icons from Segoe Fluent Icons, a font
that comes with Windows 11, named after Cascadia Mono in the acc tab's
face, nothing to install. Proposed on 2026-10-28 and withdrawn by the
owner the same day without being implemented (amendment 59), for plan
56's Nerd Font icons and the same look on any Windows version; its
proposal in git (`git show d88c392:plan.md`).

## Plan 58 - A click outside the window closes it

Brief amendment 60: a left click outside the window floating over the
dashboard, the file viewer or the quick open, closes it as Esc's last step
does and does nothing else, the click judged against the frame drawn last
(`Screen.Window`); full screen, its first row is outside; the draft
window keeps its rule. Approved on 2026-10-28: 5 PASS, 0 FAIL
([`doc/plan58-evaluation.md`](doc/plan58-evaluation.md)); commit
0408228, then the approval commit, both pushed; its full text in git
(`git show 0408228:plan.md`).

## Plan 59 - The branches popup

Brief amendment 61: a click on the project bar's branch name, or Alt+B,
opens a popup of the repository's branches and tags: a search, Recent,
Local, Remote and Tags, a message where a group is empty; a switch (git
switch, a remote branch as a tracking one), confirmed when uncommitted
changes would come along and refused while the agent works, a notice in
zone B and a note in an empty draft after it; a fetch of every remote,
never pruning, Esc and a two-minute limit stopping it; a name copied; the
header's branch following the HEAD file, checked every 2 s. New branch
left out by the owner. Approved on 2026-10-28: 10 PASS, 0 FAIL after the
owner's visual review ([`doc/plan59-evaluation.md`](doc/plan59-evaluation.md));
commit 5c8c9be, then the approval commit, both pushed; its full text in
git (`git show 5c8c9be:plan.md`).

## Plan 60 - Git blame in the file viewer

Brief amendment 62: `b` in the file viewer shows, left of the line
numbers, the date and author of the commit that last changed each line,
after the owner's image of an IDE's blame, and in the footer the cursor
line's commit; a line not committed shows "Not committed" and claims no
author; a compact form where the code would keep fewer than 60 cells. Git
is given the very bytes shown (`git blame --porcelain --contents -`), an
answer is kept only for the same file and the latest read, and git's
refusals are named. Approved on 2026-10-28: 9 PASS, 0 FAIL after the
owner's visual review ([`doc/plan60-evaluation.md`](doc/plan60-evaluation.md));
commit 0050409, then the approval commit, both pushed; its full text in
git (`git show 0050409:plan.md`).

## Plan 61 - Clicks in the file viewer and the quick open

Brief amendment 63: a left click on a line of the file viewer puts the
cursor on it, anywhere on its row between the side borders, the view
kept and the selection starting as before; in the quick open a click on
a file selects it and a click on the selected one opens it, as in the
Explorer. A click is judged by the rows the frame drew
(`Screen.WindowList`). Approved on 2026-10-28: 7 PASS, 0 FAIL after the
owner's visual review ([`doc/plan61-evaluation.md`](doc/plan61-evaluation.md));
commit b452ade, then the approval commit, both pushed; its full text in
git (`git show b452ade:plan.md`).

## Plan 62 - Git Graph

Brief amendment 64: the sidebar's third tile, or Alt+G, shows the
repository's history in the docked place, in git's topological order,
read a page at a time from fixed tips: coloured lanes, an overflow column
that joins no commits git does not join, labels; a commit's files and a
file's diff in that commit; a scope popup of references (All, Auto,
branches, remote branches, tags); a wide view; the cursor's row in the
footer and a card beside a row. Read only, nothing fetched. Approved on
2026-10-28 after the owner's visual review of its three stages: 11 PASS,
1 FAIL accepted as a documented exception (62.10, 209 bytes kept a
commit, not about 100) ([`doc/plan62-evaluation.md`](doc/plan62-evaluation.md));
commit e906b73, then the approval commit, both pushed; its full text in
git (`git show e906b73:plan.md`).

## Plan 63 - The ping

Brief amendment 65: when the agent needs the owner while the dashboard's
window, or its tab, lacks the focus (an approval or a question waiting, a
turn completed or failed, the session ended on its own), one BEL in ACC's
own Windows Terminal profile, whose fragment sets `"bellStyle":
"taskbar"`, at most one in 5 s and none lost to that limit, and the most
urgent reason in the window's title; nowhere else a BEL. Approved on
2026-10-28: 6 PASS, 0 FAIL after the owner's review ("I like it", "No
sound") ([`doc/plan63-evaluation.md`](doc/plan63-evaluation.md)); commit
4a049f3, then the approval commit, both pushed; its full text in git
(`git show 4a049f3:plan.md`).

## Plan 64 - Windows notifications

Brief amendment 66: with each ring of plan 63's ping, a silent Windows
notification of the most urgent need rung, under the name Agent Command
Center, its text generic (the CLI's name only), one at a time and taken
out of Windows' list once nothing waits; its app ID that of a Start menu
shortcut the dashboard makes when missing, pointing at Windows Terminal's
alias with ACC's profile; a feasibility probe first. Approved on
2026-10-28: 7 PASS, 0 FAIL after the owner's review ("Looks right", "No
sound") ([`doc/plan64-evaluation.md`](doc/plan64-evaluation.md)); commit
0111dbd, then the approval commit, both pushed; its full text in git
(`git show 0111dbd:plan.md`).

## Plan 65 - The turn-by-turn review

Brief amendment 67: after each turn, zone B lists the files that changed
in the project while the turn ran, whoever changed them, each file's
change of that turn in plan 44's diff view; two git snapshots in a store
of the dashboard's own, readable by the user alone, no hook or filter
run, no credential file, hard link or file outside the project read.
Approved on 2026-10-29 through the reviews the owner shared: 7 PASS and
2 accepted with a documented exception, 65.1 (a split index's shared
file's time) and 65.9 (TestBannerToFrame's timing on a busy machine)
([`doc/plan65-evaluation.md`](doc/plan65-evaluation.md)); commit
3757c15, then the approval commit, both pushed; its full text in git
(`git show 3757c15:plan.md`).

## Plan 66 - The conversation's search

Brief amendment 68: Ctrl+F's bar at the foot of zone B finds a query in
what B holds as text, a long paste's folded lines and the calls' and
requests' targets among them, the newest match first, ↑ older and ↓
newer, each lit on the file viewer's amber and shown, Tab the match's
entry whole; nothing written or sent to the CLI. Approved on 2026-10-29
through the review the owner shared and the owner's acceptance of 66.9's
exception: 11 PASS and 1 accepted with a documented exception, 66.9
(TestBannerToFrame's timing on a busy machine)
([`doc/plan66-evaluation.md`](doc/plan66-evaluation.md)); commit
a5116aa, then the approval commit, both pushed; its full text in git
(`git show a5116aa:plan.md`).

## Plan 67 - The characters typed with AltGr

Brief amendment 69: every field of the dashboard types the characters
AltGr types, one rule for all (`keyText`: a key's text with no Ctrl and
no Alt, or with both), Ctrl or Alt alone and Ctrl with the left Alt
typing nothing; the approval card lets no character into the draft; the
branches popup keeps a modifier's own press, which Windows Terminal sends
before AltGr's character; no shortcut changed. Approved on 2026-10-29
through the review the owner shared and the owner's keyboard check: 6
PASS ([`doc/plan67-evaluation.md`](doc/plan67-evaluation.md)); commit
e888947, then the approval commit, both pushed; its full text in git
(`git show e888947:plan.md`).

## Plan 68 - The keyboard reader's races

Brief amendment 70: the Windows reader of the vendored ultraviolet sent a
console read's bytes and wrote the next read into the same array before
the receiver had copied them, a key lost and the next doubled, and its two
goroutines shared the grapheme state; patch 4 sends each read as a copy,
patch 5 gives the sender a grapheme state of its own, tested on the real
console path in a console of the test's own, the suite requiring the
copy's tests to run. Approved on 2026-10-29 through the design and
implementation reviews the owner shared: 9 PASS
([`doc/plan68-evaluation.md`](doc/plan68-evaluation.md)); commit 61dd8ad,
then the approval commit, both pushed; its full text in git (`git show
61dd8ad:plan.md`).
