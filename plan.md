# Current Plan

The active plan with its progress, the owner's standing decisions, and one
entry per closed plan. Each plan comes from `doc/brief.md` and its
amendments, ends with the evaluation summary of `doc/evaluation.md`, and
waits for the owner's approval before the next one starts. This file keeps
the current state only (the owner, 2026-10-18): a closed plan's full text,
criteria and progress log are in git (`git show 8888f59:plan.md` for plans 1
to 13, `git show 84d2ad8:plan.md` for plan 14, `git show b047434:plan.md`
for plan 15), and its results in its evaluation file.

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
(the owner, 2026-10-18). Amendment 4 names ten "future improvements"
(`doc/brief.md`, Amendment 4, "For this phase, do NOT implement"). Six
remain below. Recent projects, favourites and a project browser were plan 15
(amendment 9); git worktree management is plan 16 (amendment 10).

- **[Nice to have]** Automatic VS Code integration.
- **[Nice to have]** Advanced model configuration.
- **[Nice to have]** Cross-provider agent collaboration.
- **[Nice to have]** Provider switching during an active session.
- **[Nice to have]** Complex session profiles.
- **[Nice to have]** Advanced orchestration configuration.
- **[Nice to have]** Google Antigravity as an agent CLI, removed from the
  MVP by amendment 5; the provider interface stays open to it.

## Plan 16 - Git worktree management

**Done, awaiting the owner's approval**: 26 criteria, 26 PASS
(`doc/plan16-evaluation.md`). The owner's request (2026-10-18, `doc/brief.md`, amendment
10): the first of amendment 4's other future improvements, with the three
choices approved there: worktrees beside the repository; create, list,
diff, merge and remove; at startup and in session. Git runs through
`os/exec` (no new dependency; `x/sys/windows` is already in the graph). The
agent CLIs and their configuration are not touched. Nothing is written
inside the repository except what git itself writes for an operation the
owner confirmed.

Design:

- **The git layer.** `internal/input/git.go` holds the `Git` interface and
  `GitCLI`, which finds `git` on the PATH. With none on it, the view says
  "git not found: worktrees unavailable".
  - Every call runs `git -C <dir>` with no standard input,
    `GIT_TERMINAL_PROMPT=0` and `GIT_OPTIONAL_LOCKS=0`, so a read never
    writes and nothing waits on a prompt.
  - Every call has a time limit: 15 s for a read, 2 min for an operation.
  - stderr is kept and shown as git's reason.
  - The list comes from `git worktree list --porcelain`, the main worktree
    first. Each row counts its uncommitted changes from `git status
    --porcelain`, and its commits ahead of the main worktree's branch from
    `git rev-list --count`.
  - No call ever passes `--force`, `-f` or `-D`.
- **Create** (startup only).
  - The path is `<main>.worktrees\<name>`, beside the main worktree.
    `<name>` is the branch name's last segment, with every character other
    than letters, digits, `.`, `_` and `-` replaced by `-`.
  - The branch name is checked by `git check-ref-format --branch`, and an
    existing path is refused.
  - An existing branch is checked out there; a new one starts from the main
    worktree's HEAD (`git worktree add -b`).
- **Diff.**
  - A worktree's diff runs against its merge base with the main branch. It
    covers committed and uncommitted changes (`git diff --no-color
    --no-ext-diff --no-textconv <base>`), then names its untracked files.
  - The main worktree's diff is its uncommitted changes.
  - At most 5000 lines show, then "… N lines left out".
- **Merge into main.**
  - It runs `git merge --no-edit <branch>` in the main worktree, only when
    both worktrees are clean and the branch is ahead.
  - If the merge fails and `MERGE_HEAD` is present, `git merge --abort`
    runs and the view says "conflict: merge aborted, main unchanged".
- **Remove.**
  - It runs `git worktree remove <path>`; git refuses a dirty or locked
    worktree.
  - Then `git branch -d <branch>` runs. If git refuses to delete an
    unmerged branch, the view says "branch kept:" with git's reason.
- **Sessions.** `internal/input/sessions.go` records which worktrees a
  dashboard session runs in.
  - Each dashboard session keeps a file open in
    `%LOCALAPPDATA%\agent-command-center\sessions\` holding its project
    path.
  - On Windows, a file another handle holds open is a live session: opening
    it unshared fails with a sharing violation. A file nobody holds is stale
    and is deleted.
  - On other systems the column says N/A.
  - Documented limit: it knows this dashboard's sessions only, not an agent
    CLI started in another terminal.
  - Tests and mock runs register nothing.
- **The Worktrees view.** Ctrl+W opens it full screen, like the browser.
  - **Which repository.** At startup, the one holding the project field's
    directory (the current directory when the field is empty). In session,
    the session's project; for a mock run, the mock file's directory.
  - **One row per worktree:**
    - its branch, or "detached";
    - its state: "clean", or "+N" uncommitted changes;
    - "↑N" commits ahead of main;
    - "this session" or "session" where a dashboard session runs;
    - its path.
  - **Keys:**
    - ↑↓ PgUp PgDn Home End move; Esc closes the view.
    - At startup, Enter fills the project field with the selected path and
      starts.
    - At startup, `n` asks for a branch name and shows the path it will
      create. Enter creates it and selects it; Esc cancels.
    - On both screens, `d` shows the diff full screen (↑↓ PgUp PgDn Home
      End, Esc back).
    - On both screens, `m` merges into main and `x` removes, each after a
      y/n question.
  - **`m` is refused, with the reason said in the view:**
    - on the main worktree, a detached one, or the one this session runs
      in;
    - when either side has changes, or nothing is ahead;
    - when main is detached;
    - when another dashboard session runs in main;
    - when this session runs in main and its turn is not idle.
  - **`x` is refused:** on the main worktree, this session's, a dirty one,
    or one where another session runs.
  - **Git never runs in the update loop.** It runs in a command, with a
    generation number so a stale result is dropped. While it runs, the view
    says "reading…" or what it is doing, and keys stay handled.
  - The startup footers and the key overlay gain Ctrl+W.

1. **The git layer and the session registry.** Done, 1.1 to 1.7 PASS.
   `internal/input/git.go` (TestWorktreePath, TestGitWorktrees, TestGitAdd,
   TestGitMerge, TestGitRemove, TestGitDiff, TestGitMissing) on temporary
   repositories, git run with an empty global and no system configuration;
   git lists the linked worktrees by path, after the main one.
   `internal/input/sessions.go` (TestSessionRegistry: a held file is live,
   a file nobody holds is deleted at the next read, Release leaves nothing).
   `go vet ./internal/input`, gofmt clean; `go test ./internal/input` PASS.
2. **The view at startup.** Done, 2.1 to 2.4 PASS. `internal/app/worktrees.go`
   and `internal/render/worktrees.go`; Ctrl+W on the startup screen.
   TestWorktreesStartup, TestWorktreeCreate on a fake git; golden frames
   `worktrees` and `worktrees-new` at the four sizes; the 18 existing startup
   frames change in their footer line only (`git diff --stat`: 18 files, 18
   lines each way); TestFootersFit (render): every startup and Worktrees
   footer within 79 cells below 100 columns.
3. **The view in session.** Done, 3.1 to 3.7 PASS. Ctrl+W on the dashboard;
   the mouse is off while the view is open; a session registers its project
   (`input.Sessions`) and releases it at Shutdown. TestWorktreesSession,
   TestWorktreeDiff, TestWorktreeMerge, TestWorktreeRemove,
   TestWorktreesSlow, TestSessionRegistered, TestKeysListWorktrees; golden
   frames `worktrees-session`, `worktrees-diff`, `worktrees-confirm` at the
   four sizes. A diff's tabs are drawn as four spaces, as `wrap` draws them
   (found on the `worktrees-diff` frame). `go test ./internal/app
   ./internal/render ./internal/input` PASS.
4. **Documents and the full check.** Done, 4.1 to 4.8 PASS. Amendment 10,
   AGENTS.md, architecture.md (ASM-25), ui-design.md (Worktrees, startup
   keys and footers, Anywhere), reliability.md, evaluation.md.
   TestKeyOverlayRows now reads 29 rows (Ctrl+W added).
   - Build, vet (also `-tags live` and `timing`), gofmt: clean.
   - `go test ./...`: 306 PASS, 1 SKIP, 0 FAIL.
   - Timing: 4/4 PASS, the worst p95 7.08 ms.
   - Both checks ran first with `confcheck.go` at HEAD through `-overlay`,
     because the working copy's uncommitted change broke the build. The
     owner then approved restoring the file to HEAD, and the plain commands
     gave the same results (`doc/plan16-evaluation.md`).
   - Terminal runs: 34/34 PASS maximised and 34/34 at 80x24 (the second
     80x24 run; the first failed one check on the script's side).
   - 65 files free of CR, BOM and NUL.
   - After the commit, at the owner's request: two dashboards and a test
     double for `claude.exe` ran the two items not verified, a session
     started on a worktree and another session's "session" mark. 37/37 PASS
     maximised and at 80x24 (`doc/plan16-evaluation.md`, added section).
   - Then live, on the owner's quota: a real Claude Code 2.1.302 on Haiku
     started on a worktree, wrote a file there and named it; a second
     dashboard marked it "session" until it quit. 80x24: 25/25 PASS;
     maximised: 24 PASS, configuration NOT-VERIFIED (Claude Code rewrote
     `known_marketplaces.json` during the turn). Earlier runs failed on the
     script's side; all are kept (`doc/plan16-evaluation.md`, added section).

### Criteria

Item 1, the git layer and the session registry. The tests run on a
temporary repository and are skipped where git is not installed.

- 1.1 The list (TestGitWorktrees):
  - the main worktree comes first;
  - each row has its branch, or detached;
  - its uncommitted changes and its commits ahead of main are counted;
  - paths are in Windows form;
  - a directory outside any repository gives "Not a git repository" and
    the directory.
- 1.2 Create:
  - the path is `<repo>.worktrees\<name>` (TestWorktreePath);
  - a new branch starts from main's HEAD (TestGitAdd);
  - an existing branch is checked out (TestGitAdd);
  - an invalid name, an existing path and a branch checked out elsewhere
    are refused with the reason, and nothing is created (TestGitAdd).
- 1.3 Merge (TestGitMerge):
  - a branch ahead merges;
  - a conflict is aborted: main's HEAD is unchanged, there is no
    `MERGE_HEAD`, its status is clean, and the error says "conflict".
- 1.4 Remove (TestGitRemove, the calls recorded):
  - a clean worktree goes, and its merged branch is deleted;
  - an unmerged branch is kept, with git's reason;
  - a dirty worktree is refused by git and left whole;
  - no git call in the layer's tests passes `--force`, `-f` or `-D`.
- 1.5 Diff (TestGitDiff): committed and uncommitted changes against the
  merge base; the untracked files named; capped with "… N lines left out".
- 1.6 No git on the PATH (TestGitMissing): every call fails with "git not
  found".
- 1.7 The registry (TestSessionRegistry):
  - a registered session is live for a second reader;
  - once it is released, its file is stale and is deleted at the next
    read.

Item 2, the view at startup. The tests use a fake git, no repository.

- 2.1 Opening and closing (TestWorktreesStartup):
  - Ctrl+W opens the view for the project field's directory, or the
    current one when the field is empty;
  - outside a repository it says "Not a git repository";
  - with no git it says "git not found";
  - Esc closes it with the field unchanged.
- 2.2 Enter on a row fills the project field with its path and starts
  (TestWorktreesStartup).
- 2.3 Create (TestWorktreeCreate):
  - `n` shows the branch field and the path it will create;
  - Enter creates it and selects the new row;
  - a refusal shows the reason, and the list is unchanged;
  - Esc cancels.
- 2.4 Frames:
  - golden frames `worktrees` and `worktrees-new` at 79x24, 80x24, 120x40
    and 200x60;
  - the existing startup frames change in the footer line only;
  - every footer fits within the width.

Item 3, the view in session. The tests use a fake git.

- 3.1 Opening and closing (TestWorktreesSession):
  - Ctrl+W on the dashboard opens the view for the session's project;
  - the session's row is marked "this session", and another dashboard
    session's row "session";
  - Esc returns to the dashboard;
  - an event that arrives meanwhile is in the state after Esc.
- 3.2 The diff (TestWorktreeDiff):
  - `d` shows the diff coloured, scrolls, and shows the cap line;
  - Esc returns to the list.
- 3.3 Merge (TestWorktreeMerge):
  - `m` asks y/n;
  - `y` merges, says so, and reads the list again;
  - `n` cancels;
  - each refusal of the design gives its reason and calls no git merge;
  - a conflict says "conflict: merge aborted".
- 3.4 Remove (TestWorktreeRemove):
  - `x` asks y/n;
  - `y` removes and says what became of the branch;
  - each refusal of the design gives its reason and calls no git removal.
- 3.5 A slow git (TestWorktreesSlow): the view shows "reading…" or the
  operation, keys stay handled, and a stale result is dropped.
- 3.6 Registration (TestSessionRegistered):
  - a session started from the startup screen registers its project and
    releases it at the end;
  - a mock run, or no registry, registers nothing.
- 3.7 Frames:
  - golden frames `worktrees-session`, `worktrees-diff` and
    `worktrees-confirm` at the four sizes;
  - the key overlay lists Ctrl+W.

Item 4, documents and the full check:

- 4.1 The documents:
  - amendment 10 is in `doc/brief.md`;
  - worktrees leave [Nice to have] in `plan.md` and AGENTS.md;
  - architecture.md, ui-design.md, reliability.md and `doc/evaluation.md`
    say what is built.
- 4.2 `go build ./...` and `go vet ./...` (also `-tags live` and `-tags
  timing`) exit 0, and `gofmt -l .` prints nothing.
- 4.3 `go test ./...` gives 0 FAIL, the new tests included.
- 4.4 The timing step, run alone: every test PASS (p95 <= 16 ms).
- 4.5 A terminal run in Windows Terminal, maximised:
  - setup: a temporary repository with two worktrees, a temporary store
    and home, and no agent CLI on the PATH;
  - it covers the view, create, diff, a merge, a refused merge, a removal,
    Esc and Ctrl+Q;
  - no agent CLI is started.
- 4.6 The same run at 80x24.
- 4.7 The changed and new files carry no CR, no BOM and no NUL.
- 4.8 The summary is in `doc/plan16-evaluation.md` and here.

## Closed plans

One entry each: what the plan did, its approval, its evaluation file and
its commits. Their criteria and progress logs: `git show 8888f59:plan.md`
(plans 1 to 13), `git show 84d2ad8:plan.md` (plan 14), `git show
b047434:plan.md` (plan 15).

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
