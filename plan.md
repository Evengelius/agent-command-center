# Current Plan

The active plan with its progress, the owner's standing decisions, and one
entry per closed plan. Each plan comes from `doc/brief.md` and its
amendments, ends with the evaluation summary of `doc/evaluation.md`, and
waits for the owner's approval before the next one starts. This file keeps
the current state only (the owner, 2026-10-18): a closed plan's full text,
criteria and progress log are in git (`git show 8888f59:plan.md` for plans 1
to 13, `git show 84d2ad8:plan.md` for plan 14), and its results in its
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
  "newer than tested" meanwhile.

## Nice to have

Items tagged **[Nice to have]** are future improvements: none is built, and
none is listed as remaining, until the owner asks for one in a new plan
(the owner, 2026-10-18). Amendment 4 names the first seven "future
improvements" (`doc/brief.md`, Amendment 4, "For this phase, do NOT
implement"); its other three, recent projects, favourites and a project
browser, are plan 15 (amendment 9).

- **[Nice to have]** Git worktree management.
- **[Nice to have]** Automatic VS Code integration.
- **[Nice to have]** Advanced model configuration.
- **[Nice to have]** Cross-provider agent collaboration.
- **[Nice to have]** Provider switching during an active session.
- **[Nice to have]** Complex session profiles.
- **[Nice to have]** Advanced orchestration configuration.
- **[Nice to have]** Google Antigravity as an agent CLI, removed from the
  MVP by amendment 5; the provider interface stays open to it.

## Plan 15 - Recent projects, favourites and a project browser

**Approved by the owner on 2026-10-18:** 21 criteria, 21 PASS
([`doc/plan15-evaluation.md`](doc/plan15-evaluation.md)). The owner's
request (2026-10-18, `doc/brief.md`, amendment 9): three of amendment 4's
future improvements, built "sequentially, one by
one": recent project history, project favourites, then a project browser.
Each item is checked before the next starts. Everything is on the startup
screen; the dashboard, the agent CLIs and their configuration are not
touched. No new dependency: `encoding/json` and `x/sys/windows` are already
in the graph.

Design, one store for the three items:

- **The store.** `projects.json` in the dashboard's own folder,
  `%LOCALAPPDATA%\agent-command-center\` (beside `drafts\`), written through
  a temporary file renamed over the old one; one entry per project: its
  validated path, when a session last started in it, and whether it is a
  favourite. Paths are compared without case on Windows. It holds no
  conversation data. A store that cannot be read or written is said so on
  screen, never shown as an empty list. Tests and mock runs pass no store:
  nothing is read or written, and the startup screen is unchanged.
- **The startup list.** Under the project field, "Projects": favourites
  first (★), then recent projects, each by its last start, with the
  directory's name, its path and its age ("3 h ago"). It is a Tab stop
  between the project field and Start, shown only when not empty, with as
  many rows as fit so the Start button stays whole at 80x24. ↑↓ move,
  Enter fills the project field with that path and starts (Enter starts on
  this screen), Del forgets the project, `f` stars or unstars it.
- **The browser.** Ctrl+O on the startup screen opens it full screen at the
  project field's directory (or its nearest existing parent, or the home
  directory). It lists directories only, sorted without case, with a git
  mark and ★; hidden and system directories are counted and left out until
  Ctrl+H. The first row chooses the directory shown, then `..`, then the
  subdirectories. Typing filters by name; ↑↓ PgUp PgDn Home End move;
  Enter opens a directory, or on the first row fills the project field and
  starts; ← or Backspace (empty filter) goes up; Ctrl+F stars or unstars
  the selected directory; Esc clears the filter, then closes the browser
  with the field unchanged. On Windows, `..` at a drive's root lists the
  drives. A directory is read off the update loop; an unreadable one shows
  its error and stays browsable upwards.

1. **Recent project history.** Done, 1.1 to 1.6 PASS. The store is
   `internal/input/projects.go` (TestProjectsStore, TestRecordProject);
   the list, its keys and its problems are `internal/app/projects.go` and
   `internal/render/startup.go` (TestProjectRecording, TestProjectListKeys,
   TestProjectStoreProblems, TestAge). Golden frames `startup-projects` and
   `startup-projects-focused` at the four sizes: at 80x24 three rows show,
   windowed around the selection ("6–8 of 10"), and the Start button is
   whole (checked in TestGoldenFrames for every startup frame); the existing
   frames are unchanged. A path is recorded once validation passes, even if
   the agent CLI then fails to start: it is the validated path. go build,
   go vet, gofmt clean; `go test ./...` 279 PASS, 0 FAIL (273 before); the
   timing step PASS (p95 1.04 ms, 2.07 ms, 3.15 ms).
2. **Project favourites.** Done, 2.1 to 2.3 PASS. `Projects.Star` in the
   store (TestStarProject: first, kept past the cap, unstarred a recent
   project again; a directory starred before any session, from the
   browser, goes when unstarred and shows "never" as its age); `f` in the
   list (TestProjectFavourites: stars, the store keeps it, the selection
   follows it first; `f` again unstars; Del forgets a favourite; `f` in the
   project field is typed). Golden frame `startup-favourites` at the four
   sizes; the list's footer gains "f star", so the two
   `startup-projects-focused` frames changed in that line only. go build,
   go vet, gofmt clean; `go test ./...` 281 PASS, 0 FAIL.
3. **A project browser.** Done, 3.1 to 3.4 PASS. The listing is
   `internal/input/browse.go` (TestDiskList: sorting, a hidden directory, a
   git mark from a `.git` directory or file, an empty, a missing and a file
   path; TestSortEntries; TestDiskListWindows: the hidden and system
   attributes, a junction listed as a link, the drives). TestDiskListLink
   SKIPs here: a symbolic link needs a privilege this account lacks; the
   junction covers the link case. The browser is `internal/app/browse.go`
   and `internal/render/browse.go` (TestBrowserOpens, TestBrowserKeys,
   TestBrowserDrives, TestBrowserSlow), listed through `Options.Lister` so
   the tests use a fake one. Golden frames `browser`, `browser-filter` and
   `browser-reading` at the four sizes; the startup footers gain "Ctrl+O
   browse", so the three `startup` and six `startup-msg` frames changed in
   that line only. go build, go vet (also `-tags live`, `-tags timing`),
   gofmt clean; `go test ./...` 288 PASS, 0 FAIL, 1 SKIP; the timing step
   PASS (p95 1.05 ms, 1.56 ms, 3.23 ms).
4. **Documents and the full check.** Done, 4.1 to 4.8 PASS. Amendment 9
   in `doc/brief.md`; the three items leave [Nice to have]; ui-design.md's
   startup screen, reference-design.md's and ui-design.md's image rows,
   architecture.md's Out of scope and doc/evaluation.md's MVP-scope check
   say what is built; go build, go vet (also `-tags live`, `-tags timing`),
   gofmt, `go test ./...` with 0 FAIL; the timing step (the startup screen
   is a render path); a terminal run in Windows Terminal maximised and at
   80x24 with a temporary store, no agent CLI started; changed files without
   CR or BOM; the summary in `doc/plan15-evaluation.md`. Result: `go test
   ./...` 288 PASS, 1 SKIP, 0 FAIL; the timing step PASS (p95 1.01 ms,
   2.03 ms, 3.02 ms); the terminal runs, maximised (156x44) and at 80x24,
   26 checks PASS each, both agent CLIs not installed, none started; 61
   files without CR, BOM or NUL.

### Criteria

Item 1, recent project history:

- 1.1 The store: Save then Load gives the same entries; a missing file is
  an empty list without error; a damaged file is an error naming the file;
  a failed write leaves the previous file whole.
- 1.2 Recording: the project moves to the front with its start time, no
  duplicate (a path differing only in case on Windows is the same
  project), at most 10 recent projects kept.
- 1.3 A session started from the startup screen records its validated
  path; a refused validation, a mock run or no store records nothing.
- 1.4 The startup screen lists the recent projects with name, path and
  age; with no store or an empty one the screen is byte for byte today's
  (the existing golden frames unchanged); with 10 entries the Start button
  is whole at 80x24 (golden frames).
- 1.5 Keys: Tab goes agent, project, list, Start; ↑↓ move in the list and
  still choose the agent in field 1; Enter starts with the entry's path;
  Del forgets it and the store no longer holds it.
- 1.6 A store that cannot be read shows "Recent projects unavailable:" and
  the reason on the startup screen; one that cannot be written shows it
  too.

Item 2, project favourites:

- 2.1 `f` on an entry stars it and the store keeps it; `f` again unstars
  it.
- 2.2 Favourites come first, marked ★, then the others, each group by its
  last start; a favourite is never dropped by the cap of 10.
- 2.3 Del on a favourite forgets it; golden frame of the list with
  favourites.

Item 3, a project browser:

- 3.1 The listing (table tests on a temporary tree): directories only,
  sorted without case; a git mark on a directory holding `.git`; hidden
  and system directories counted and left out, shown with Ctrl+H; a link to
  a directory listed; a missing or unreadable directory gives its message.
- 3.2 The browser opens at the field's directory, else its nearest existing
  parent, else the home directory.
- 3.3 Keys: open, up, filter, Esc twice, Enter on the first row starts with
  that directory, Ctrl+F stars it in the store; on Windows `..` at a root
  lists the drives.
- 3.4 A slow listing shows "reading…" and keys stay handled; golden frames
  of the browser at 80x24 and maximised from a fake listing.

## Closed plans

One entry each: what the plan did, its approval, its evaluation file and
its commits. Their criteria and progress logs: `git show 8888f59:plan.md`
(plans 1 to 13), `git show 84d2ad8:plan.md` (plan 14).

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
