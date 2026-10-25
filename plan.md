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

## Plan 44 - Source Control and a Git diff viewer

Brief amendment 45 (2026-10-25): the sidebar's second icon is Source
Control, not Skills; its panel, in the Explorer's docked place, lists the
project's Git changes, staged and unstaged apart, each with its real
status; a file opened shows its diff (the working tree against the index,
or the index against HEAD), green and red, in the conversation's place;
read-only, after the owner's image `references/14-source-control.png`. The
criteria below were written before the code; nothing is committed until
the owner has reviewed the result.

### What was found

- **Git status** (plan 17; `internal/input/explorer.go`): `GitCLI.Status`
  runs `git rev-parse --show-prefix` and `git status --porcelain=v1 -z
  --untracked-files=normal --ignored -- .` through `GitCLI.run` (no prompt,
  no optional lock, English); `ParseStatus` keeps one marker a path for the
  Explorer, X and Y merged, a rename's source skipped, ignored paths apart.
  The Explorer asks once each time it shows (`openExplorer`); nothing else
  reads it. The Worktrees view's `Diff` runs `git diff --no-ext-diff
  --no-textconv` (plan 16).
- **Probe P44a** (a temporary repository, `scratchpad\plan44\lfsprobe`, git
  2.56.0.windows.2, the owner's global `filter.lfs.*`): a traced `git
  status` on an LFS-tracked file whose stat alone changed ran `git-lfs
  filter-process` through `sh.exe`; with empty `filter.lfs.*` overrides it
  ran nothing but called the unchanged file modified. `git cat-file
  --batch` gave the index's blob raw (the LFS pointer), no filter run.
- **The docked area** (plan 43): `ui.Explorer.Open` decides the panel's
  place (`layout`), the focus "R", its keys and mouse, the motion filter
  and the Explorer's tile selected; the second tile is `SideSkills`, a
  placeholder; no tooltip (plan 43, design D2).
- **Drawing**: zone B is `zoneB` inside `dashboard`; the full views
  (approval, entry, inspection, details) replace the whole screen in
  `Draw`; the file viewer (plan 17) draws numbered code lines, ←→ shifting
  them 8 columns.
- **Architecture**: render imports neither input nor the application;
  the application turns input's results into render's types; a pure
  package is allowed beside the five parts (`internal/arch`).

### Design

- **D1. The sidebar.** The second tile is Source Control (`SideSource`),
  its icon and looks unchanged. A click on it shows the Source Control
  panel in the docked place, a second click hides it; a click on the
  Explorer's tile shows the Explorer in its place. One panel at a time,
  each keeping its own state (folders and cursor; sections and cursor)
  while the other shows. F3 is the Explorer's, exactly as in plan 43: with
  Source Control shown, F3 shows the Explorer. While a tile is hovered, a
  label left of the sidebar beside it names it, "Explorer" or "Source
  Control", on the selectors' hover ground.
- **D2. One status.** `GitCLI.Status` keeps its two calls, `rev-parse`
  (now `--show-prefix --show-toplevel`) and the same `status`, with
  `--branch` and `-c core.fsmonitor=false`. `ParseStatus` gives, besides
  the Explorer's markers derived as before, each entry's X and Y, its path
  and a rename's or copy's original path, and the branch line (a branch, a
  detached HEAD, no commits yet). Ignored entries are never changes. Each
  status read updates both views from the same snapshot.
- **D3. Refresh.** A status is read when a panel shows, on "r" in the
  Source Control panel or the diff, after a tool call or a turn ends, and
  while a panel or a diff shows every 3 s; never two at once, the interval
  10 times a slow read's duration (at most 30 s); none while neither shows.
  A read equal to the last changes nothing; a new one updates the lists,
  each cursor kept on its path. The open diff is compared again when its
  file's status, its blobs' ids or the working file's size and time
  change.
- **D4. The Source Control panel**, docked as the Explorer (same width,
  compact rule, focus "R", border, header "⌄"): "SOURCE CONTROL" in the
  title colour, bold, a rule, the repository row "<repository> · <branch>"
  (or "detached at <id>", "no commits yet on <branch>"), then the sections,
  each "▾"/"▸", its name in the title colour and its count at the right:
  MERGE CHANGES (conflicts, when any), CHANGES (the working tree: Y, and
  untracked), STAGED CHANGES (the index: X). A file's row: its status
  letter in its colour (M amber; A, U, R, C green; D red; T blue; "!" red
  for a conflict), its directory muted, its name, a rename's or copy's
  original muted after "←". An untracked folder is a row "U dir/" that
  opens (→, Enter, a click) on its files, read in a command, at most 1,000.
  The cursor row on the purple band, the hovered row on a subtler ground.
  Empty states: "No changes", "No Git repository found for this project.",
  "git not found", git's own error in the attention colour.
- **D5. Keys and mouse of the panel**: ↑↓ PgUp PgDn Home End over the
  rows; → and ← open and fold a section or an untracked folder, ← on a
  file to its section; Enter on a file its diff (the comparison of its
  section), on a header folds it; "r" refreshes; Esc back to zone B, the
  panel kept; the global keys as beside the Explorer's panel, the draft's
  clipboard keys included (plan 43). A click on a file opens its diff, on
  a header folds it; the wheel scrolls the list; the hover follows the
  pointer (plan 36's rule), the motion filter passing a report that
  changes the hovered row.
- **D6. The diff's two sides, read raw.** HEAD's and the index's blobs by
  one `git cat-file --batch` (no filter, no textconv, no pager, no external
  diff), the working file from the disk with the viewer's refusals (inside
  the project, no credential file, 32 MiB). Unstaged: the index (stage 0)
  against the working file; staged: HEAD at the original path against the
  index; untracked: nothing against the working file; deleted: the old side
  against nothing; a conflict: "ours" (stage 2, else HEAD) against the
  working file, the conflict named ("both modified", "added by us", …),
  never called resolved. A CR before LF is dropped on both sides, a
  difference of line endings alone said so; a NUL in a side's first 8,000
  bytes: binary, both sizes said, no lines; a Git LFS pointer against
  content: said, no lines.
- **D7. The lines compared** by a Myers diff, own code in a pure package
  `internal/diff`, 3 lines of context, hunks; bounded: over 20,000 edits
  or 100,000 lines a side, the middle between the common head and tail
  shown removed then added, said so; computed in a command with a
  generation, "comparing…" meanwhile.
- **D8. The diff view** in zone B's place, zone A, the panel, the sidebar
  and zone C as they were; zone B's conversation, draft and scroll kept,
  drawn again as they were when it closes. Its box: the file's name; the
  path ("old → new" for a rename) and the comparison ("Working tree:
  index → working tree" or "Staged: HEAD → index") with "+N −M"; then the
  lines. Unified under 150 columns: the old and new line numbers in grey,
  "−"/"+", the text; removed lines on a dark red ground, added on a dark
  green one, context in the muted text colour, "@@ -a,b +c,d @@" in
  lavender. Side by side from 150 columns: ORIGINAL and MODIFIED columns,
  each numbered, a removed line facing its replacement, blank where a side
  has none. Long lines cut with "…", ←→ shifting 8 columns. "f" widens the
  view over zone A, and back. Messages in the lines' place: no textual
  change, line endings only, binary, LFS, too large.
- **D9. The diff's keys**: ↑↓ PgUp PgDn Home End scroll, the wheel too; n
  and N the next and previous change; Tab the file's other comparison when
  it has both; "r" compares again; "f" wide; Esc back to the panel, its
  cursor on the file, zone B as it was. F3, F2 and the global keys as
  usual.
- **D10. Read only.** Every git call is `rev-parse`, `status` or
  `cat-file`, never one that writes, each with no optional lock and
  `core.fsmonitor=false`; the index and the working tree are never
  touched. Git's own clean filters still run during `status`, as in any
  `git status` (P44a); a diff never runs one.
- **D11. Not a repository**: the panel shows "No Git repository found for
  this project."; nothing else is run or invented.
- **D12. Unchanged**: zone A, zone C, zone B's design, the Explorer and
  its keys, F3, the sidebar's geometry and looks; the footer shows the
  Source Control panel's or the diff's keys while they have the focus, a
  waiting approval first (plan 43).

### Criteria

1. **44.1 The sidebar**: the second tile named Source Control everywhere
   (code, F1, documents), its four looks unchanged; a click shows its
   panel (the tile selected) and a second hides it; Explorer ↔ Source
   Control switch by their tiles, each state kept (the Explorer's folders
   and cursor, the sections and cursor); F3's tests of plan 43 unchanged,
   and with Source Control shown F3 shows the Explorer; the label
   "Source Control" or "Explorer" beside a hovered tile, gone off it.
2. **44.2 Status parsing** over recorded porcelain records: M, A, D, R
   (with its original path), C, T, untracked file and folder, staged,
   unstaged, both, the seven conflicts (UU AA DD AU UA DU UD), spaces,
   Unicode, nested paths, the branch line (a branch, detached, no commits),
   ignored never a change; the Explorer's markers from the same parse as
   before (TestParseStatus's cases unchanged).
3. **44.3 Status from git** on temporary repositories (git on the PATH,
   outside the project): each case of 44.2 made with real git commands,
   a merge conflict included, read by `GitCLI.Status`.
4. **44.4 The panel**: header, rule, repository row, the sections in
   order with their counts, rows with their letters' colours, muted
   directories, renames, an untracked folder opened on its files; the
   cursor's band and the hover's ground; the empty states; golden frames
   at 120x40 (compact) and 200x60 (docked).
5. **44.5 Refresh**: a file changed between reads appears with its
   status at the next read, a revert or an external commit removes it; a
   tool call's end and a turn's end ask a read; "r" asks one; never two
   at once; none while no panel or diff shows; an equal read leaves the
   view as it was; the open diff compared again when its file changes.
6. **44.6 The two sides**, on temporary repositories: unstaged (index →
   working file), staged (HEAD → index), both on one file (two different
   diffs), staged add, untracked, deleted unstaged and staged, renamed
   with changes (HEAD:old → index:new), a conflict (ours → working file,
   named, not resolved), binary, line endings only, no newline at the
   end, an LFS pointer; each with its exact expected lines.
7. **44.7 The diff algorithm**: random edits, the script applied gives
   the new side; minimal on small cases against a brute force; hunks with
   3 lines of context; the bound applied and said, within time.
8. **44.8 The diff view drawn**: unified and side by side at their
   widths, the line numbers, the red and green grounds, the hunk headers,
   cut lines and the shift, each message; golden frames.
9. **44.9 The diff's navigation**: scrolling by key and wheel, n and N,
   Tab between the comparisons, f wide and back, Esc back: zone B's frame,
   draft and scroll identical to before opening, the focus on the panel's
   file.
10. **44.10 Read only**: every git call's arguments (traced) are
    `rev-parse`, `status` or `cat-file` with the safety settings; a full
    run on a temporary repository leaves `.git/index` and the working tree
    byte for byte as they were.
11. **44.11 Not a repository**: the empty state, no other git call.
12. **44.12 Unchanged elsewhere**: every golden frame change explained
    by a script (the tile's name and the new panel only); the Explorer's
    tests, F3's, the draft's and the mouse's pass.
13. **44.13 The checks**: build, vet, gofmt; `go test ./...`; `-race` on
    render, app, input and diff; the timing step alone, with a test of a
    large diff's scrolling (p95 within 16 ms); breakages of the new code
    caught by tests (`go test -overlay`).
14. **44.14 Windows Terminal**, the acc tab's profile (208x59) and 120x40,
    a temporary repository: A, the panel listing modified, deleted and
    untracked files; B, a diff with added and removed lines and their
    numbers; C, one file's staged and unstaged diffs apart; D, the Explorer
    and Source Control from their tiles, zone C full width; the colours
    read back.

### Steps

1. The finding, brief amendment 45, the criteria.
2. The status (input) and the diff engine (input, internal/diff).
3. The sidebar's second tile, the panel, its keys, mouse and refresh.
4. The diff view.
5. The checks; Windows Terminal; the documents; the evaluation
   (`doc/plan44-evaluation.md`); the report.

### Progress

- 2026-10-25: step 1 done; steps 2 and 3 in progress.
- 2026-10-25: steps 3 and 4 done in code and tests (the panel, the
  tooltip, the refresh, the diff view, its keys; `TestDiffToFrame` added to
  the timing step); the owner's example layout followed (blank rows around
  the headers, files three cells in); where the panel takes zone B's place
  the diff docks it beside itself from 118 columns ("The Source Control
  sidebar stays visible"); 16 golden frames added, none changed. Step 2's
  engine (`internal/input/source.go`, `internal/diff`) built by an
  implementer subagent; step 5 next.
- 2026-10-25: step 2 done and checked against its report: the bound applied
  to the changed middle, not the whole file (a 100,001-line file with one
  changed line was flagged "too large to align"); a project folder
  untracked in a larger repository said so (`GitSnapshot.Untracked`), not
  "No changes"; an untracked folder's ignored files left out
  (`GitMarks.IsIgnored`); the viewer's refusals said as the viewer says
  them, not "git: …"; "no newline at the end" said in the unified view when
  that line is shown; F2 from the diff, or the panel beside it, closes the
  diff and focuses the approval's card; "f wide" left out of the footer
  under 100 columns. Step 5 in progress: build, vet, gofmt and `go test
  ./...` pass; the race detector, the mutations, the timing step and
  Windows Terminal running.
- 2026-10-25: step 5: mutations 58 of 58 caught; Windows Terminal at
  208x59 and 120x40 on a temporary repository, the colours read back; the
  first runs showed a path cut before its name in the 31-column panel, a
  rename cutting its name, a stray "…" in the side by side heads: fixed
  (the directory gives way as "…/"), Tab moving the panel's cursor, the
  footer without Tab for one comparison, and the runs made again; the
  repository and the clipboard unchanged. The final suite, race detector
  and timing step on the final tree, then `doc/plan44-evaluation.md`.
- 2026-10-25: step 5 done: 14 of 14 PASS
  ([`doc/plan44-evaluation.md`](doc/plan44-evaluation.md)); on the final
  tree build, vet, gofmt clean, `go test ./...` ok (762 tests), `-race` on
  render, app, input and diff ok, the timing step's ten tests within their
  targets (TestDiffToFrame p95 1.0 to 1.8 ms). Waiting for the owner's
  review; nothing committed.

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
