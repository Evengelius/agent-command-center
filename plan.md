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

## Roadmap

The owner's order (amendment 65, 2026-10-28), plan 64 put first by
amendment 66 and now closed: a turn-by-turn review of what the agent
changed (plan 65, below), then a search of the conversation,
then the session's resumption from the home screen. Each starts as a plan
for the owner's review when the one before is closed.

## Plan 65 - The turn-by-turn review

Brief amendment 67 (2026-10-28), the first of the roadmap's three
(amendment 65): after each turn, the files the turn changed in the
project, and each file's change of that turn in plan 44's diff view.
Proposed for the owner's review, another agent's three reviews folded
in, and approved by the owner on 2026-10-29.

### What was found (2026-10-28, on 2af203a)

- Neither CLI reports a turn's changes with the whole files. Codex sends
  `turn/diff/updated`, the turn's patch as one text (P6b, P7b, P12x: new
  files written through its patch tool, a subagent's in its own thread's
  turn); the state keeps the last one (`state.Diffs`) and nothing draws
  it. Claude Code's stream gives each file tool's result,
  `tool_use_result`, the file before the edit in `originalFile`:
  undocumented, and recorded only for new files written (P2, P12c,
  P303a). No recording has a file changed by a command (`gofmt -w`, `sed
  -i`), which neither report is known to cover.
- Git can record the project's files as a tree without touching the
  project or its `.git`, with an index and an object folder of the
  dashboard's own: `git add -A`, then `git write-tree --missing-ok`, both
  with `GIT_INDEX_FILE` and `GIT_OBJECT_DIRECTORY` set to them (git
  2.56.0 here; `scratchpad\plan65\probe65*.sh`, in scratch clones and
  repositories only):
  - nothing in `.git` changed, path, size or time, in a clone of this
    repository, a linked worktree and a repository of 20,000 files; the
    reads of the trees (`diff-tree`, `cat-file`), the repository's
    objects behind the store's, changed nothing either;
  - with the repository's objects as alternates while git wrote, git
    refreshed the time of a pack file in `.git` (its "freshening" of an
    object it finds already stored): the writes go to the store alone,
    the repository's objects only read;
  - `git write-tree` writes the index too, which ran the repository's
    `post-index-change` hook: with `core.hooksPath` at an empty folder on
    every call, no hook ran; a filter driver given empty `clean` and
    `process` commands did not run; git's own `add`, the control, ran
    both;
  - a file is stored as git's own `add` stores it (a CRLF file, `text
    eol=crlf` under `core.autocrlf` true, as LF), so a turn's counts are
    git's;
  - the file viewer's credential names (`input.CredentialName`), as
    excluded pathspecs without case, kept every such file out (`auth.json`,
    `sub/AUTH.JSON`, `.env.local`, `key.pem`, `Secrets.yaml`, …) and
    `secret.md` in; `.env`'s three examples are left out too;
  - the store's index seeded from the repository's, the first snapshot of
    this repository (1,019 files) took 0.72 s and the store held 44
    objects (118 KiB) after a small turn; unseeded, it hashed every file:
    6.3 s and 15.5 MB. Then 0.45 to 0.54 s a snapshot here, 0.7 to 1.1 s
    with 20,000 files (the first there 5.4 s, its files just committed,
    so that git compared them by their content); git's start alone takes
    0.24 s on this machine;
  - an unchanged folder gives the same tree twice, and one `git diff-tree
    -r -z --raw --numstat -M` of a turn's two trees lists its files with
    the right letters and counts.
- A listing first, and what it must not let through (2026-10-29,
  `scratchpad\plan65\probe65e.sh` to `probe65g.sh`, scratch repositories
  only):
  - `git status --porcelain=v2 -z --untracked-files=all` against the
    store's index, the repository's objects behind it to read, lists what
    changed since the store's last snapshot in 0.4 to 0.65 s here, with
    1,019 or 20,000 files (`git ls-files -m -o`: 1.0 to 1.3 s with
    20,000), `.git` unchanged; `git add -A` of exactly those files, then
    `write-tree`, takes about 0.5 s more.
  - Both list the files inside a junction to a folder outside the
    project (`link-dir/notes.txt`): git's walk does not keep them out, so
    each file listed is checked as the file viewer checks one before git
    reads it (`input.ReadFile`: its links resolved inside the project).
  - Files that each held a marker, looked for afterwards in every object
    of the store, none found: ignored by `.gitignore`, by a folder in it,
    by `.git/info/exclude` and by `core.excludesFile`; `auth.json` and
    `sub/.env.local`; the file behind the junction, left out by the
    check; a 40 MiB file, left out by its size. A symbolic link could not
    be made here: Windows asks for administrator rights.
  - A file appended to every 50 ms while a snapshot was taken was listed
    again right after it.
  - After four snapshots, the store held 7 objects, 7 KiB.
- The store's size (2026-10-29, `scratchpad\plan65\probe65h.sh`): 200
  tracked files, 21.0 MiB (the probe's 21,508 KiB, first written here as
  21.5 MiB). The seeded first snapshot took nothing in;
  each turn that added a line to one 105 KiB file listed that file alone
  and added about 12 KiB to the store; a turn's next start with nothing
  changed, and every file touched without a change, listed nothing; after
  eight turns and one file put back as it was, the store held 120 KiB.
  The dashboard's folder in `%LOCALAPPDATA%` takes that folder's access:
  the user, SYSTEM and the administrators in full, and a local group of
  this machine, of two members, allowed to read.
- The Git Graph lists a commit's files from two git objects and shows a
  file's change from them (`input.CommitFiles`, `CommitSides`, plan 62);
  both take trees as well. The diff view draws hunks (`render.DiffView`),
  whatever compared the sides.
- A send is refused while a turn runs ("turn running: wait, or
  interrupt"): each of the owner's turns starts with a send, in the write
  queue's order (`internal/app/writer.go`).

### Design (proposed)

- **D1. Git, not the CLIs' reports.** The review compares the project's
  files at the turn's start and at its end, as git sees them: one way for
  both CLIs, with what the agent's commands and subagents changed, whole
  files for the diff view, and no undocumented field.
- **D2. The snapshot.** First the listing: `git status --porcelain=v2 -z
  --untracked-files=all` against the store's index, the credential names
  excluded by pathspecs without case, git's ignore rules as they are
  (`.gitignore`, `.git/info/exclude`, `core.excludesFile`; never `add
  -f`). Each file listed then passes the file viewer's checks before git
  reads it (plan 17; `input.ReadFile`'s, without reading): no credential
  name, its own or its link's target's; inside the project once its links
  and junctions are resolved; 32 MiB at most. One that fails them is left
  out, never read, its size and time kept to tell whether it changed.
  When only such files are listed, nothing changed and the snapshot is
  the last one's tree; else `git add -A` of exactly the files listed and
  passed, then `git write-tree --missing-ok`. Every call:
  `GIT_INDEX_FILE` and `GIT_OBJECT_DIRECTORY` at the store, the
  repository's objects behind it to read (the listing, `diff-tree`,
  `cat-file`) and never while git writes; `core.hooksPath` at an empty
  folder; each filter driver of git's configuration given empty commands;
  `core.fsmonitor`, `core.splitIndex` and `core.safecrlf` off;
  `GIT_OPTIONAL_LOCKS=0`. Never a commit, a stash, a reference, the
  repository's index or a project file written.
- **D3. The store and its limits.** One store for the session, in
  `%LOCALAPPDATA%\agent-command-center\turns\`, held as a draft's folder
  is (plan 33, design D1), its folder made with an access list of its
  own, not inherited: the user alone, where the rest of `%LOCALAPPDATA%`
  lets SYSTEM, the administrators and, on this machine, a local group in.
  It holds an index, seeded once from the repository's, then kept, so
  that only what changed is read; an object folder, where git keeps each
  version it takes in once, compressed; an empty hooks folder. The store
  is given to the review's own git calls alone, never to Source
  Control's, the Git Graph's or a commit's, and nothing of it is written
  to a log or a file of the dashboard's (the screen's trace that
  `TEA_TRACE` turns on for debugging records what the screen shows, of it
  as of the rest).
  - It takes in the files whose content changed since the last snapshot
    (git's listing compares the content of a file whose time changed, so
    a file merely touched is not listed); at the first, those whose
    content differs from the repository's index (the owner's uncommitted
    and untracked ones); each at most 32 MiB. A tracked file that did not
    change is never read or copied: its version stays in the
    repository's own objects.
  - At most 256 MiB on disk: before a snapshot, the store's size plus
    the sizes of the files it is about to take in, an upper bound of
    their compressed copies with git's headers; after it, the store's
    size measured again. A snapshot that would pass the limit is not
    taken, and none is once the store has passed it; a turn without both
    of its snapshots has no review, its row saying why, the message
    going all the same, and what was taken stays readable. With 20 MiB
    of tracked files, eight turns that each added a line to one 105 KiB
    file left the store at 108 KiB.
  - Every snapshot of the session is kept, two a turn at most (a turn's
    end serves as the next one's start when nothing changed between),
    their trees' names in the dashboard's memory. Nothing is removed
    while the dashboard runs, so every turn's diffs stay readable. An
    object of the repository's own that only the store's index names
    could be pruned meanwhile by a `git gc --prune=now`: its side then
    shows git's error, never another content.
  - Removed when the dashboard quits, once its git processes have
    stopped. One left by a crash or a kill stays on this machine until a
    dashboard starts again, which removes every store no running
    dashboard holds; ui-design.md names the folder, to empty by hand
    meanwhile.
  - Not a backup: no history kept, nothing restored, nothing kept past
    the dashboard.
- **D4. The turn's bounds.** Its start, for the owner's message: a
  snapshot in the write queue before the message, which goes once it is
  taken, after 5 s at most: a snapshot not done by then is stopped, the
  message goes and the turn has no review. About 0.5 s here when nothing
  changed since the last snapshot (the listing alone), about 1 s
  otherwise, to be judged in the owner's use; the editor's row says "the
  turn's snapshot…" meanwhile. For
  a turn the CLI opens (a background agent's report), where the turn
  before ended, so that the agent's work between them shows with its
  report; with no turn before, and in a mock session, whose messages are
  replayed, when the turn opens. Its end: a snapshot when the turn ends,
  completed, failed or interrupted, or when the session ends during it;
  then the listing once more, and one `git diff-tree -r -z --raw
  --numstat -M` of the turn's two trees.
- **D5. What is compared.** Each file as git would store it in this
  repository: its line-ending rules applied (`core.autocrlf`, the `text`
  and `eol` attributes), so that a CRLF file they normalize compares as
  LF, and a change of line endings alone that git would normalize away
  does not show; no filter run, so that a filtered file (Git LFS)
  compares as its bytes on disk, against its pointer where the
  repository's index has one, as the diff view says. Not the bytes on
  disk; the diff view then draws the lines as it draws Source Control's
  (plan 44, design D6). A snapshot is not atomic: git reads the files one
  after another, and one written meanwhile may be caught before or after.
  The listing after the end catches a file written while the snapshot
  was taken or just after it: the row then says the folder was still
  changing, and that file's later changes are in no turn's review, as
  nothing written between two turns is.
- **D6. What it covers.** Every file of the project folder that git does
  not ignore, changed while the turn ran, by the agent's file tools, its
  commands or its subagents, or by anything else that wrote there
  meanwhile; ui-design.md says so. Never in it: a file git ignores; a
  credential file by the file viewer's names, `.env`'s three examples
  too; a file outside the project, behind a link or a junction or not; a
  file over 32 MiB, whose row says it changed, without its content; a
  submodule's files, its commit only, as git records it; what changed
  between two turns (Source Control shows it). A secret in a file with an
  ordinary name that git does not ignore is taken in as `git add` would
  take it, into the store on this machine, removed as D3 says. No review
  outside a git repository, where Source Control says why.
- **D7. In zone B,** the dashboard's own rows, drawn from its state:
  never a message of the conversation, never sent to the CLI, never in
  the CLI's session or history. Under a turn's last row, its RESULT box
  or else its CLI block's last row, on the tool rows' ground as a run's
  fold row is (plan 19): "▸ 3 files changed in turn 2 · +12 −4 · Review",
  the count muted, "+12" green, "−4" red, "git" at the right as its
  origin; "· still changing" after it in the attention colour when D5's
  last listing found a file written meanwhile. Unfolded, "▾ … · Fold" and
  the files under it as the Git Graph lists a commit's (plan 62, design
  D5): the letter in Source Control's colours, the name, its folder
  dimmed, a rename's source after "←", "+8 −2" or "binary"; a file left
  out with why ("over 32 MiB, not compared", "outside the project, not
  compared"); past 1,000 files, "first 1,000 of N files". A turn that
  changed nothing has no row; one without a review has "✗ No review of
  turn 2: <why>" in the failure colour. Folded at first, the owner's
  choice holding for the session. At 120x40, unfolded:

  ```text
  ▎  ✓ Edit   …\internal\render\viewer.go            11:30:02  observed
  ▎  ✓ Bash   gofmt -w internal/render                11:30:04  observed

  ╭──────────────────────────────────────────────────────────────────╮
  │ RESULT  11:30                                                    │
  │ I renamed the field and reformatted the package.                 │
  ╰──────────────────────────────────────────────────────────────────╯
   ▾ 3 files changed in turn 2 · +12 −4 · Fold                    git
       M  viewer.go  internal/render/                           +8 −2
       A  viewer_test.go  internal/render/                      +4 −0
       D  old.txt                                               +0 −2
  ```

- **D8. Keys and the mouse,** through zone B's entry cursor (plan 5, item
  4; plan 42): ↑ ↓ stop on the summary row and on each file row shown;
  Enter or → on the summary unfolds it, Enter or ← folds it; Enter on a
  file opens its diff, ← there folds the list, the cursor back on the
  summary; a click on the summary folds or unfolds it; a double click on
  a file opens its diff. A file left out opens nothing.
- **D9. The diff,** in zone B's place as Source Control's (plan 44):
  "DIFF · viewer.go", its path row "internal/render/viewer.go · Turn 2:
  start → end", the two sides read from the store by one `git cat-file
  --batch`; unified or side by side, with every key of the diff view, a
  turn having one comparison; Esc back to zone B as it was, the cursor on
  the file. It writes nothing and sends nothing to the CLI.
- **D10. Never:** a write in the project or its `.git`; a hook or a
  filter of the repository run; a credential file, a file outside the
  project or one over 32 MiB read; a CLI's configuration changed; the
  conversation or the CLI's session added to; a message held past 5 s;
  a store over 256 MiB, open to anyone but the user, or kept past its
  dashboard; a file watcher, a background snapshot or another cache to
  shorten the wait before the owner's use asks for one; a new
  dependency. Once approved, AGENTS.md's git constraint
  says so in one line, and its MVP scope lists the review.

### Criteria (proposed)

1. **65.1 The snapshot** (input tests on real git, in temporary
   repositories): an unchanged folder's tree the same twice, from the
   listing alone; after a turn that modifies, adds, removes and renames
   files and writes a binary one, exactly those files with the Git
   Graph's letters and counts; every file of the project and of its
   `.git`, path, size and time, the same before and after, in a main and
   in a linked worktree; the repository's filter driver and
   `post-index-change` hook not run (no mark left), git's own `add`
   running both; a CRLF file stored as git's own `add` stores it, a
   filtered one as its bytes; a tracked file the owner changed before
   the first turn, and an untracked one made before it, both changed
   again by the turn: its review shows the turn's change alone, never
   the owner's earlier one.
2. **65.2 What never reaches the store,** each such file holding a
   marker looked for in every object the store holds, none found: files
   ignored by `.gitignore`, by `.git/info/exclude` and by
   `core.excludesFile`; every name the file viewer refuses, in upper case
   and in a subfolder too, and a link to such a file; a file outside the
   project behind a junction, and behind a symbolic link where Windows
   lets the tests make one, said not verified where it does not; a file
   over 32 MiB, its row saying it changed; a credential file held open
   without sharing failing nothing.
3. **65.3 The store** (input and application tests): at most 32 MiB a
   file and 256 MiB on disk, projected before git reads and measured
   after each snapshot; the limit reached on a small limit the test
   sets, no large fixture: the message sent all the same, that turn's
   row saying its snapshot was not taken, the earlier turns' diffs still
   readable, no review shown in part, the store removed at the end; over
   turns with no change, with one file changed and with every file
   touched, a file unchanged or merely touched never counted or copied,
   a changed one once, the store growing by the changed files alone
   (20 MiB of tracked files, eight turns of a line each: under 1 MiB);
   its folder's access list the user's alone, not inherited, the user
   making, reading and removing it, no other entry in it; its index
   never given to Source Control's or the Git Graph's git; every
   snapshot readable until the end; removed at the dashboard's end after
   its git processes, one left unheld removed at the next start, another
   running dashboard's never; a start snapshot past 5 s stopped, the
   message going, its reason said.
4. **65.4 The turns** (application tests on a fake store and a fake
   CLI): the owner's message written after its snapshot, the writes'
   order kept, and written still when the snapshot fails or is stopped;
   the end snapshot at a completed, a failed and an interrupted turn and
   at the session's end; a turn the CLI opens starting at the previous
   end; a file written during the end's snapshot marking the turn "still
   changing"; no row without a change; nothing outside a git repository;
   nothing sent to the CLI, no conversation entry added.
5. **65.5 Zone B** (render tests and golden frames at 80x24, 120x40 and
   200x60): the row folded and unfolded under a RESULT box and under a
   CLI block without one, a CLI-opened turn's own row, a left-out file's
   row, "still changing", the failure's row; the RESULT box holding the
   same message; the cursor's stops, the keys, the click and the double
   click as D8 says; the existing golden frames unchanged.
6. **65.6 The diff:** Enter on a file opens it with "Turn 2: start →
   end", unified and side by side, with its keys; a binary or too large
   file said as the diff view says it; Esc back with the cursor on the
   file.
7. **65.7 The documents:** ui-design.md, a Turn review section with its
   frame (`TestDocFrames`); architecture.md; events-and-state.md
   (`TestStateMachines`); reliability.md, the store's limits and end;
   brief amendment 67; AGENTS.md.
8. **65.8 Windows Terminal:** in the acc tab, `claude-subagent-approval`'s
   mock on a temporary repository, files changed while its approval
   waits: once answered, the row under the turn, unfolded, a file's diff
   opened and closed; at 208x59 and 120x40; the owner's look at the
   shots.
9. **65.9 The checks:** build, vet (Windows, Linux), gofmt and the whole
   suite; the changes broken on purpose through `go test -overlay`, each
   caught; `-race` on the changed packages; the timing step on mains
   power, a turn of 1,000 files unfolded within the keystroke's 16 ms;
   the message's added wait measured here and with 20,000 files.

### Steps

1. Amendment 67 and this plan, for the owner's review.
2. The store and the snapshot (input): 65.1 to 65.3.
3. The turns (state, app): 65.4.
4. Zone B's rows and the diff (render, app): 65.5 and 65.6.
5. The documents: 65.7.
6. Windows Terminal: 65.8.
7. The checks, the mutants, the timing and `doc/plan65-evaluation.md`
   (65.9); the commit after the owner's review.

No live run: a turn's bounds are events the dashboard already reads, and
the mock, with files changed during its turn, checks the rest; no CLI
configuration touched, no quota used.

### Progress

- 2026-10-28: step 1: amendment 67 and this plan, proposed for the
  owner's review, with the findings above (the probe scripts in the
  scratchpad, nothing of them in the repository).
- 2026-10-29: another agent's review, shared by the owner, folded in, its
  three points answered: the store's limits, lifetime and end (D3); what
  never reaches the store and the checks that keep it out (D2, D6,
  65.2); what is compared, and a snapshot that is not atomic (D5); the
  rows the dashboard's own, never the conversation's (D7). The probe run
  again for them (What was found, a listing first): git's walk enters a
  junction to a folder outside the project, so each file listed passes
  the file viewer's checks before git reads it.
- 2026-10-29: the other agent's second review, shared by the owner, its
  one question answered: the 256 MiB are the store's size on disk,
  checked before each snapshot with the sizes of the files it is about
  to take in, and identical content is never counted twice (measured,
  What was found, the store's size); the store's folder the user's alone
  (D3); the message's wait to be judged in the owner's use (D4).
- 2026-10-29: the other agent's final review, shared by the owner,
  approving the design for implementation; its focused tests folded in
  (65.1, a working tree changed before the first turn; 65.3, the store
  measured after each snapshot as well, turns with little or no change,
  the limit reached without blocking the message, the folder's access);
  no watcher, background snapshot or extra cache before real use asks
  for one (D10). Waiting for the owner's own approval to start step 2.
- 2026-10-29: approved by the owner ("Approve and start"); AGENTS.md's
  status, its git constraint and its MVP scope updated. Step 2 begins.
- 2026-10-29: step 2, the store and the snapshot, given to an implementer
  subagent (`internal/input/turns*.go`, `git.go`'s `env` and `opts`,
  `file.go`'s checks shared), its interface fixed first so that steps 3 and
  4 went on beside it. Steps 3 and 4: the state's review
  (`internal/state/review.go`: `TurnReview`, `ReviewState`, the commands
  and actions; the send's start before its write, a stage the send did not
  open, a CLI-opened stage from the stage before's end, the turn's end, a
  crash during it; the editor's notice "the turn's snapshot…"); zone B's
  rows (`internal/render/review.go`, counted in the last entry's span, under
  a RESULT box with no block, in the CLI's block otherwise), the cursor's
  stops and keys (`UI.EntryReview`; `stepCursor`, `reviewKey`), a double
  click on a file (`press`); the application's store, its ordered queue of
  snapshots, the message's gate from the send (5 s), the end's 30 s, the
  files, the diff ("Turn 2: start → end", "r" again, Esc back to zone B),
  a panic one turn's problem (`internal/app/review.go`). Tests:
  `internal/state/review_test.go` (8), `internal/render/turnreview_test.go`
  (5), `internal/app/review_test.go` (8); golden frames `review`,
  `review-open` and `review-diff` at the four sizes, the turns' snapshots
  given as the state's own actions after the mock's replay; every other
  golden frame unchanged. A slip: `internal/render/review_test.go`, plan
  41's, overwritten by a new file of the same name, restored from git at
  once (no difference from HEAD), the new tests moved to
  `turnreview_test.go`. Step 5 in progress: ui-design.md's Turn review and
  B's entries, events-and-state.md's Turn review (under State model, so that
  the seven machines stay seven), reliability.md's The turn review,
  architecture.md's The turn review, the key overlay's rows.
- 2026-10-29: step 2 done by the implementer subagent and reviewed:
  `internal/input/turns.go` (`OpenTurnStore`, `Take`, `Changing`,
  `Files`, `Sides`, `Size`, `Close`), `turns_windows.go` (the folder's
  access list, SDDL `D:P(A;OICI;FA;;;<user SID>)`; the index copied while
  sharing its deletion), `turns_other.go`, `git.go`'s `env` and `opts`,
  `file.go`'s `checkFile` shared with `ReadFile`; `turns_test.go` and
  `turns_windows_test.go`, 26 tests, `TestTurnLinks` skipped (Windows
  refuses a symbolic link without its privilege). Its findings, kept in
  reliability.md: git writes about 7 ms a new file (the start's 5 s past
  about 700 files, the end's 30 s past about 4,000); a split index's
  shared file has its time refreshed by any read of it, Source Control's
  too; with Git LFS the first snapshot takes every LFS file in as bytes.
  Steps 5 and 6: the documents; go build, vet (Windows, Linux), gofmt and
  go test ./... ok (16 packages); Windows Terminal (`scratchpad\term\
  p65run.ps1`, the mock on a repository of its own, git on the PATH, a
  temporary LOCALAPPDATA): at 208x59 and 120x40, an edit during turn 1 and
  one between the turns, "2 files changed in turn 1 · +3 −0" and "2 files
  changed in turn 2 · +1 −3" (the CLI's turn from turn 1's end), unfolded,
  old.txt's diff "Turn 2: start → nothing", Esc back on the file; the
  store protected, the user alone, 793 then 971 bytes, gone after the
  quit; no terminal-hosted Claude Code or Codex session open. The
  owner's look at the shots waits.
- 2026-10-29: step 7, the checks, in `doc/plan65-evaluation.md`: 9
  criteria, 8 PASS, 0 FAIL, 65.8's look the owner's. The mutants run again
  on the final tree through `go test -overlay`, the code's hash the same
  before and after: the state's, render's and application's 41 all caught,
  the store's 27 caught but for 2 equivalent on Windows; `-race` on the
  four packages, no race; the timing step alone on mains power, all twenty
  tests within their targets, a turn of 1,000 files unfolded p95 1.17 ms;
  the real store measured (the message's wait 0.11 to 0.12 s here, 0.43 to
  0.47 s with 20,000 files; eight turns of a line each over 21.0 MiB,
  70,195 bytes). Writing the evaluation found that no test pressed "r" in
  a turn's diff: `TestTurnReviewDiffReload` added, three mutants of it
  caught. Not verified: a symbolic link (65.2, Windows refuses the tests
  one). Waiting for the owner's look at the shots and approval; nothing
  committed.
- 2026-10-29: the owner shared, with no words of their own, another
  agent's review of the implementation and of the shots: 65.8 approved
  ("I approve criterion 65.8 — the human visual review"); before the
  commit, two points (brief amendment 67, its last paragraph), both done.
  A file with more than one name is left out, "a hard link", opened for
  its attributes alone (`linkCount`, `TestTurnHardLink`: a credential file
  outside the project linked in under an ordinary name, held open without
  sharing, never in the store). A split index's refresh is verified as a
  documented exception to 65.1 (`TestTurnSplitIndex`: its shared file's
  time alone, nothing after the first snapshot), in reliability.md and
  AGENTS.md. Besides: `TestTurnReviewCursorReveals` (↓ onto a file below
  the view shows it), the summary's count grouped, "2,345 files changed";
  the counts the keys cut on a narrow row kept as built; ui-design.md says
  the review lists files changed, never who changed them. Checked again:
  the suite, 46 mutants caught and the store's 27 but the 2 equivalent,
  `-race`. The timing step ran on a busy machine: the home screen's two
  tests over their targets, HEAD's tree alike in turn, all else within;
  its rerun on a quiet machine is still to do. A `found.txt`, a listing
  of this folder written at 10:32 while, by its folders' times, a
  dashboard of this tree ran here (10:29 to 10:57, its stores' folder
  empty after), is not plan 65's: left in place, out of the commit.
  Evaluation:
  `doc/plan65-evaluation.md`, after the block; waiting for the owner's
  approval to commit.
- 2026-10-29: the other agent's next review, shared by the owner: the
  corrections approved, no code change asked, the split index's exception
  accepted; the timing step open until a complete run passes on a quiet
  machine, else the owner's explicit decision. The machine stayed busy
  (10 to 41 % CPU, 11:42 to 11:51); the complete step at 11:52, alone,
  on mains power: 19 PASS, TestBannerToFrame p95 2.05 ms over its 2 ms
  (the home picture 248.5 ms, within its 260 ms). Waiting for the owner:
  a rerun on a quiet machine, or a documented timing exception; then the
  commit, on the owner's approval.
- 2026-10-29: the other agent's final review, shared by the owner with no
  words of their own: "Accept the documented timing exception and proceed
  to commit", plan 65's files only, `found.txt` untouched, no push until
  allowed. Recorded in `doc/plan65-evaluation.md`: TestBannerToFrame's
  FAIL kept as measured, accepted as a documented timing exception, not a
  PASS; 9 criteria, 7 PASS and 2 accepted with a documented exception
  (65.1, 65.9). The commit follows.

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
