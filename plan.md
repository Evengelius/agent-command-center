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

## Plan 45 - Commit, Push and Commit & Push in Source Control

Brief amendment 46 (2026-10-25): under the Source Control panel's
repository row, a message field with the placeholder "Message", a purple
"✓ Commit" button with a separator and a chevron, and under it, only after
a click on the chevron, a dropdown with one item, "Commit & Push"; Commit
then Push as two actions, or both in turn; Git's index respected, nothing
forced, the button's state read from Git; after the owner's image
`references/15-commit-push.png`. The criteria below were written before the
code. Nothing is committed until the owner has reviewed the result, and no
test commits or pushes in this repository: every Git test runs on temporary
repositories outside it, their remotes local bare repositories.

### What was found

- **The panel** (plan 44; `internal/render/source.go`,
  `internal/app/source.go`): header, rule, repository row, then the list
  (`SourceRows`), its rows' cells items 1 and up of panel G; the focus "R";
  ↑↓ PgUp PgDn Home End → ← Enter r Esc, the global keys passed on
  (`sourceKey`); a click opens a row (`sourceClick`), the hover follows the
  pointer (`sourceHoverAt`). One status read at a time; `applySnapshot`
  updates both views.
- **The branch line**: `GitCLI.Snapshot` reads `status --branch`, whose line
  carries "main...origin/main [ahead 1, behind 2]" or "[gone]", but
  `parseBranch` keeps the name alone: no upstream, ahead or behind yet.
- **The git runner** (`internal/input/git.go`): `command` adds `-C dir -c
  core.quotepath=off` and `GIT_TERMINAL_PROMPT=0`, `GIT_OPTIONAL_LOCKS=0`,
  `LC_ALL=C`; `call` kills git at a time limit; `stream` feeds standard
  input; git's failure is its standard error on one line (`failure`). Every
  Source Control call so far reads (plan 44, design D10).
- **The editor**: zone B's (`input.Editor`: text, cursor, selection, undo,
  a paste kept byte for byte) is drawn by `shownRows` and `drawRow`, whose
  spans are panel E's; `editKey` and `editorKey` handle its keys.
- **Images** (plans 35, 37, 39, 43): an outlined, filled shape with rounded
  corners is a Sixel image over blank cells (`PillImage`, `SideImage`, the
  modes selector's strips), its text drawn by GDI in Cascadia Mono
  (`input.TextMask`); text and an image cannot share a cell; the application
  draws the images once the frame is out and again when their rows change
  (`previewSignature`, `thumbSequence`); without Sixel, text.
- **The owner's image** (604x508, read with `scratchpad\plan45\imgpx.ps1`):
  the page #0C0D0F; the message box's inside #151722, its outline #2E304E,
  the placeholder #71789E; the button's fill #312C55, outline #8062D9,
  label and chevron #ECE5F9, separator #745EBA, the chevron's part its last
  sixth; the dropdown's ground #10101A, outline #935EF5, the hovered item on
  a rounded, inset band #23213C, its text #EDE7F8. The three boxes span the
  panel's inner width, each two rows high to the eye, the button and the
  dropdown a fifth of a row apart; the header and the repository row are
  plan 44's.
- **Probe P45a** (2026-10-25, git 2.56.0.windows.2, a temporary repository,
  `scratchpad\plan45\probe`), for the owner's two additions (D14, D15): a
  console Ctrl+Break sent to git's own process group (git started with
  `CREATE_NEW_PROCESS_GROUP`) ends git and its children at once (exit
  0xc000013a), no hook or filter process left; but git does not clean up: a
  `git add` stopped while its clean filter ran left `index.lock`, and the
  next commit failed on it ("Unable to create '…/index.lock': File exists").
  A commit stopped in its pre-commit hook held no lock (git 2.56 takes none
  there) and left none; a push stopped while waiting for a server that never
  answers left none, its remote-tracking ref unchanged.
- **Probes P45b to P45e** (2026-10-25, `scratchpad\plan45\probe2` to
  `probe5`, each caller alone in a hidden console of its own, its children
  and itself counting the interrupts they receive): where a Ctrl+Break sent
  by `GenerateConsoleCtrlEvent` goes. To a group of its own whose root and
  its child run (git and its transport): the two, never the caller (12 of
  12). Its root exited, a member left: that member (6 of 6), none left:
  nobody (6 of 6); the same with a handle to the exited root held open (15
  of 15). To a pid that leads no group, a child in the caller's group: **the
  caller, its parent** (6 of 6), the target never; Bubble Tea would take it
  as an interrupt and quit. A pid that names no process: "The parameter is
  incorrect.". So a stop may name git's own group only, and its pid only
  while it cannot name another process (D15).

### Design

- **D1. The panel's rows**: the header, the rule and the repository row as
  in plan 44; the message box (its outline, the message's rows, one to
  three, its outline); the button (three rows); the note or the question of
  D6 when there is one; then the list as in plan 44, in the rows left. The
  boxes span the panel's inner width. They are drawn while the panel shows
  a repository (its repository row); not for "No Git repository found",
  "git not found" or the first read.
- **D2. The message field**: a second editor, Source Control's, zone B's
  code: its text kept while the panel hides, while the Explorer shows and
  as the focus moves; cleared only once Git has confirmed the commit made
  from it, and only if it still holds the text committed. "Message" in the
  placeholder colour while empty. A click or ↑ from the button gives it the
  focus; its keys are the editor's: characters, Enter a new line, the
  arrows, Home, End, Ctrl+←→, Shift selections, Ctrl+A, Ctrl+Z, Ctrl+C,
  Ctrl+X, Ctrl+V and a bracketed paste (byte for byte, never committing); ↓
  from its last line to the button; Esc back to zone B. Typing never runs
  Git.
- **D3. The button**: three rows, "✓ Commit" or "Push" (D7) centred in the
  left part, the separator, the chevron in the last cells. A click on the
  left part runs the main action, on the chevron opens or closes the
  dropdown; no hover look. Its keyboard focus (↓ from the field, ↑ from the
  list's first row) shows as its outline in the title colour; there Enter or
  Space runs the main action, → or Alt+↓ opens the dropdown, ↑ goes to the
  field, ↓ to the list, Esc back to zone B. Ctrl+Enter, from the field, the
  button or the list, runs the main action. Dimmed when there is nothing to
  commit or push; a click then says "No changes to commit.". While an
  operation runs it says "Staging…", "Committing…" or "Pushing…".
- **D4. The dropdown**: hidden by default, opened only by the chevron (a
  click, or → or Alt+↓ on the button); three rows right under the button,
  its left and right columns the button's, over the panel's own rows below
  it, never outside the panel; one item, "Commit & Push", on the band while
  the pointer is over it or the keys opened the dropdown. Closed by a click
  outside it (that click does nothing else), Esc, or the item chosen (a
  click, Enter or Space). It does not open while an operation runs.
- **D5. Drawing**: with Sixel, the button and the dropdown are one image
  each over blank cells (fill, outline, rounded corners, check mark,
  separator, chevron, band; the labels drawn by GDI in Cascadia Mono, as the
  mode pill's), and the message box's outline four images around its text
  cells (its top and bottom rows, its left and right cells), the text on its
  inside colour; without Sixel or the font, rounded box characters in the
  same colours, the fills as the cells' ground. The colours are the image's
  (What was found); a disabled button's are halfway to the page's.
- **D6. Staging**: the commit takes the index as it is. Something staged:
  that alone, a partially staged file's unstaged part left out (`git
  commit` with no path). Nothing staged, changes under CHANGES: a question
  under the button, "Nothing is staged. Stage the N changes listed under
  CHANGES and commit them?" ("commit and push them?" for Commit & Push),
  with "Stage and commit" and "Cancel"; "Stage and commit" stages exactly
  those paths (`git add -A`, literal pathspecs on standard input: an
  untracked folder's files as git adds them, ignored files never), then
  commits; Cancel, Esc or a click elsewhere does nothing. No change at all:
  "No changes to commit.", no Git. A conflict: "Resolve the merge conflicts
  first.", no Git. No commit during a rebase, cherry-pick, revert or am
  ("A rebase is in progress: finish it in a terminal first."); a merge's
  commit, once its conflicts are resolved, is git's own and allowed.
- **D7. The main action, from Git**: the status read's branch line gives
  the upstream, ahead, behind and gone. Push when the branch has an upstream
  that exists and is ahead of it; otherwise Commit, enabled when there are
  changes, dimmed when not. So Push after a commit on a branch with an
  upstream, Commit after the push or with nothing to push, and after a
  restart whatever Git says. Staged changes and commits to push together:
  Push pushes the commits, Commit & Push commits the staged changes then
  pushes all. No upstream, a gone upstream, no remote or a detached HEAD
  leave Commit; a push asked for there is explained (D9).
- **D8. Commit**: the message first: empty or white space alone says "Enter
  a commit message.", no Git. Then, in a command: no rebase, cherry-pick,
  revert or am in progress; the whole repository's staged entries (`git
  status --porcelain=v1 -z --untracked-files=no`): none says "No changes to
  commit."; a conflict refuses; any outside the project's folder refuses
  ("N staged changes outside this folder would be committed too: commit them
  from a terminal, or unstage them."); those inside must be those the panel
  showed when the action was asked for, else "The staged changes changed
  since they were shown: check them, then commit again." and the panel is
  read again (an agent may have staged meanwhile). Then `git -c
  user.useConfigOnly=true commit --file=-`, the message on standard input as
  typed, git's own clean-up, hooks and signing as configured; never
  `--allow-empty`, `--amend` or `--no-verify`. Success only once git exits
  0 and HEAD is a new commit whose first parent is the old HEAD: "Committed
  <id>.", the field cleared. A failure shows git's message, a hook's output
  included, the message kept. No time limit and no cancellation: a commit
  ends when git ends it, its elapsed time shown (D15).
- **D9. Push**: to the branch's upstream only, `git push --porcelain
  <remote> refs/heads/<branch>:<upstream's ref>`; never `--force`,
  `--force-with-lease`, a "+" refspec, `--mirror`, `--all`, `--tags` or
  `--delete`. First, from `symbolic-ref`, `for-each-ref` (the upstream, the
  push destination, the track) and `remote`, each of these refuses with its
  explanation and runs nothing: a detached HEAD ("HEAD is detached: there is
  no branch to push."), no remote ("No remote is configured for this
  repository. Add one in a terminal: git remote add <name> <url>"), no
  upstream ("The branch main has no upstream. Set one in a terminal: git
  push -u origin main", the remote named when there is one only), a local
  upstream, a gone upstream, push settings (pushRemote, push.default)
  sending the branch elsewhere, nothing to push. Credentials go through
  git's own helper (Git Credential Manager's window); git never prompts on
  the dashboard's console (`GIT_TERMINAL_PROMPT=0`, and
  `SSH_ASKPASS_REQUIRE=force` with no askpass program unless the owner set
  one). A rejection is read from `--porcelain`'s lines: non-fast-forward
  ("The push was rejected: origin/main has commits that main does not
  (non-fast-forward). Pull or merge them in a terminal, then push."), the
  remote's (a hook, a protected branch: "The remote rejected the push:
  <reason>."); other failures (network, authentication, permission) in
  git's words; a URL's user and password never shown. Nothing pulled,
  merged, rebased, reset or pushed again automatically. No time limit; the
  owner can cancel it (D15).
- **D10. Commit & Push**: the message, the changes (D6) and the push's
  destination (D9's refusals, before any commit) checked; the commit; if it
  fails, stop, the message kept, no push; then the push. Both done:
  "Committed <id> and pushed to origin/main.", the dropdown closed, the
  field cleared. The push failing: "Committed <id>, but the push failed:
  <reason>", the commit kept and never made again, the field cleared, the
  button on Push for the push alone.
- **D11. One operation at a time**: while one runs, the button shows it and
  every action (a click, Ctrl+Enter, Enter, the dropdown, the question) is
  ignored, so repeated clicks or keys start one; the rest of the dashboard
  keeps working; nothing is retried automatically. When it ends, git's
  status is read again in the same command and the panel updated from it
  with the result, so the button changes on Git's word.
- **D12. Notes**: one under the button, wrapped: a validation in the
  attention colour, a success in green, a failure in red, or the question;
  replaced by the next action, cleared when the message is edited. The
  footer shows the keys of what has the focus: the field's, the button's,
  the dropdown's, the question's.
- **D13. Unchanged**: the header, the repository row, the list, its keys and
  mouse, the diff view, the Explorer, the sidebar, zones A, B and C, the
  telemetry, the model, effort and modes selectors, Ultracode, the
  attachments, the footer's other keys.
- **D14. Staged contents** (the owner's addition, 2026-10-25): Source
  Control's status read also takes each staged file's blob in the index
  (one `git cat-file --batch-check` of `:0:<path>`, a read, none with
  nothing staged; "-" for a staged deletion; the Explorer's read takes
  none), kept with its entry; the commit compares, besides the paths (D8),
  each expected path's blob with the index's at the time of the commit, so
  a file staged again with other content since the panel showed it refuses
  with the same "The staged changes changed since they were shown: check
  them, then commit again.". A path holding a line break, which cat-file
  cannot be asked, is compared by its path alone. The question's staging
  (D6) takes the files as they are when the owner confirms: they were not
  staged when shown.
- **D15. A long operation** (the owner's addition, 2026-10-25, after probe
  P45a): from its second second the button's label carries the elapsed
  time ("Committing… 3 s", past a minute "Pushing… 1:05"), redrawn by a
  tick a second (the image's label not kept, as it changes each second). A
  push can be cancelled: while it runs, the note under the button says "Esc
  cancels the push." (grey) and the footer "Esc cancel the push"; Esc in
  the panel asks git to stop, a console Ctrl+Break to git's own process
  group (git, its transport, a credential helper), and the button says
  "Cancelling…". Before that, the dashboard checks that none of the lock
  files a push could hold exists (the index's, HEAD's, packed-refs, the
  branch's and its remote-tracking ref's): if one does, "git is updating
  the repository: try again in a moment." (the attention colour) and
  nothing is sent. The break names only git's group (P45b to P45e): git's
  pid is held by a handle from its start until the stop is taken back, the
  check and the break run under the stop's lock, and git's end takes the
  stop back under that lock before letting the pid go, so a break names
  git, running or ended, never a process whose parent would receive it. A
  stopped push ends "The push was cancelled. The remote may have received
  it: Push again to check." ("Committed <id>; the push was cancelled. …"
  for Commit & Push), any lock file it left named, never deleted. A commit,
  the question's staging and Commit & Push's commit cannot be cancelled:
  P45a showed git, stopped while holding `index.lock`, leaving it behind.

### Criteria

1. **45.1 The layout**: at 200x60 (docked) and 208x59, under the repository
   row the message box's three rows ("Message" on the placeholder colour),
   then the button's three rows ("✓ Commit" in the left part, the
   separator, "⌄"), then the list as in plan 44; the boxes' left and right
   columns the panel's inner first and last; no dropdown by default; no box
   without a repository; golden frames at 200x60 and 120x40, with and
   without Sixel.
2. **45.2 The colours**: in text, the outlines, fills and labels in the
   image's colours (What was found); the images' pixels at fixed points:
   the button's fill, outline, separator and label, the dropdown's ground,
   outline and band, the message box's outline and inside, transparent
   outside the rounded corners.
3. **45.3 The message field**: typing, Enter a new line, the arrows,
   Shift selections, Ctrl+A, Ctrl+Z, Ctrl+C, Ctrl+X, Ctrl+V, a bracketed
   paste of three lines kept byte for byte and never committing; its text
   the same after Explorer → Source Control, the panel hidden and shown, the
   focus on zone B and back; no Git call while typing.
4. **45.4 Validation**: an empty message or "  \n  " says "Enter a commit
   message." with no Git call, the field shown and its text unchanged; no
   change at all says "No changes to commit." with no Git call.
5. **45.5 Staging**: staged changes commit the index alone (a partially
   staged file's unstaged part still in `git diff` after); nothing staged
   asks the question with N; Cancel and Esc stage and commit nothing; "Stage
   and commit" stages exactly the CHANGES paths (modified, deleted,
   untracked file, untracked folder), then commits; a conflict says "Resolve
   the merge conflicts first." with no Git call.
6. **45.6 Commit**, on temporary repositories: HEAD a new commit, its
   message the text with git's white space clean-up, its parent the old
   HEAD; nothing staged refused, HEAD unchanged; a pre-commit hook exiting 1
   gives its output, HEAD and the index unchanged; no identity configured
   refused, no commit; staged changes outside the project's folder refused;
   a staged set other than the expected refused; a rebase in progress
   refused; the trace never holds `--allow-empty`, `--amend` or
   `--no-verify`.
7. **45.7 Push**, on a temporary repository with a local bare remote:
   success leaves the bare repository's branch at the local HEAD and ahead
   0; no remote, no upstream, a gone upstream, a pushRemote elsewhere,
   nothing to push and a detached HEAD each refused with its explanation and
   no `push` in the trace; non-fast-forward rejected as such, the local and
   the remote branch unchanged; the remote's pre-receive hook rejecting
   gives its reason; an unreachable remote gives git's error; a password in
   the remote's URL never in a message; the trace never holds `--force`,
   `--force-with-lease`, `-f`, a "+" refspec, `--mirror`, `--all`,
   `--tags` or `--delete`.
8. **45.8 The branch line**: "main...origin/main [ahead 1]", "[behind 2]",
   "[ahead 1, behind 2]", "[gone]", "main", "No commits yet on main" and a
   detached HEAD give the exact upstream, ahead, behind and gone.
9. **45.9 The main action**: changes with no upstream, Commit; nothing,
   Commit dimmed; ahead 1, Push; ahead 1 with staged changes, Push, and
   Commit & Push commits then pushes; a gone upstream or no remote, Commit;
   a new model (a restart) on a snapshot ahead 1, Push.
10. **45.10 Workflow A**, with fakes and with git on a temporary repository
    and a bare remote: a message, then Commit, makes one commit, clears the
    field, and shows Push once the status read says ahead 1; Push pushes and
    shows Commit; new changes after it, Commit.
11. **45.11 Workflow B**: Commit & Push commits then pushes, in that order
    in the trace, says "Committed <id> and pushed to origin/main.", closes
    the dropdown and clears the field; a commit failing (a hook) runs no
    push and keeps the message with git's error; a push failing (a rejecting
    remote) says "Committed <id>, but the push failed: …", clears the field
    and shows Push; Push then runs one push and no commit; the remote's
    history holds the commit once.
12. **45.12 One at a time**: ten clicks on Commit during one operation make
    one commit; repeated Ctrl+Enter likewise; while an operation runs the
    dropdown does not open and Commit & Push does nothing; a failure is not
    retried (the fake's call counts).
13. **45.13 The dropdown**: hidden by default; a click on the button's left
    part runs the action and does not open it; a click on the chevron opens
    it with exactly one item, "Commit & Push", its left and right columns
    the button's; a click outside closes it and does nothing else; Esc
    closes it; choosing the item closes it; the pointer over the item puts
    it on the band, off it not; the button's look is the same with the
    pointer over it as without.
14. **45.14 The keyboard**: ↑ from the list's first row to the button, ↑ to
    the field, ↓ back down; Enter on the button runs the action; → opens the
    dropdown and Enter chooses its item; Esc leaves; plan 44's list keys
    unchanged (their tests pass).
15. **45.15 Unchanged elsewhere**: every golden frame change explained by
    the panel's new rows; plan 44's tests, the Explorer's, F3's, the
    draft's and the mouse's pass.
16. **45.16 The checks**: build, vet, gofmt; `go test ./...`; `-race` on
    render, app and input; the timing step alone, with a test of typing in
    the message field (p95 within 16 ms); breakages of the new code caught
    by tests (`go test -overlay`).
17. **45.17 Windows Terminal**, the acc tab's profile (208x59) and 120x40,
    on a temporary repository with a local bare remote: A, the message field
    (the placeholder, then a message); B, the Commit button; C, the dropdown
    open; D, its item hovered; E, the button on Push after a commit; F,
    after Push, the button on Commit and the commit in the bare remote; the
    colours read back; this repository's HEAD and status and the clipboard
    unchanged.
18. **45.18 Staged contents** (D14), on a temporary repository: a file
    staged and read by the snapshot, then staged again with other content,
    refuses the commit with "The staged changes changed since they were
    shown: …", HEAD and the index unchanged; staged again with the same
    content, committed; the snapshot's staged entries carry the index's
    blob, an unstaged one none; through the application the commit's
    expected entries carry the snapshot's blobs.
19. **45.19 A long operation** (D15): the label "Committing…" under 2 s,
    then with its seconds, past a minute in minutes and seconds; on a
    temporary repository pushing to a server that never answers, a stop
    ends the push within 5 s with the cancelled refusal, no lock file left,
    the remote-tracking ref unchanged, no git process left; a stop while a
    lock file exists is refused with its message and sends nothing; through
    the application, Esc during a push says "Cancelling…", then "The push
    was cancelled. …", the button left on Push, and Esc during a commit
    leaves it running; in Windows Terminal, a push to a silent server
    cancelled by Esc, and a commit held by a sleeping hook showing its
    seconds.

### Steps

1. The finding, brief amendment 46, the image, the criteria.
2. The engine (input): the branch line, staging, commit and push, tested on
   temporary repositories with bare remotes.
3. The message field, the button and the dropdown: render, keys, mouse.
4. The operations wired: the main action, the question, the notes, one at a
   time.
5. The checks; Windows Terminal; the documents; the evaluation
   (`doc/plan45-evaluation.md`); the report.
6. The owner's two additions (2026-10-25): probe P45a; the staged contents
   compared (D14); the elapsed time and a push's cancel (D15); their checks,
   Windows Terminal, the documents and the evaluation again.

### Progress

- 2026-10-25: step 1 done; steps 2 and 3 in progress.
- 2026-10-25: step 2 done by an implementer subagent and checked against
  its report (`internal/input/sourcewrite.go`: the branch line's upstream,
  ahead, behind and gone; `Stage`, `Commit`, `PushTarget`, `Push`, each
  refusal a `Refusal`, a rejection a `PushRejected`; tested on temporary
  repositories with local bare remotes, the trace free of forcing flags).
  Two choices of its own kept: `%(push:remoteref)` is empty in git 2.56,
  so the push destination is compared as `%(push)` against `%(upstream)`;
  `push.followTags` and `push.recurseSubmodules` are off for the push.
  Steps 3 and 4 in code and tests: the controls drawn in text and as Sixel
  images (`internal/render/commit.go`, `commitimage.go`), the field, the
  button, the dropdown, the question, the notes and the operations
  (`internal/app/commit.go`); ten application tests with a double and one
  with git on a temporary repository and a bare remote (Commit, Push,
  Commit & Push against a refusing pre-receive hook, Push again: the remote
  holds each commit once); twelve golden frames added (`commit-menu`,
  `commit-sixel`, `commit-push` at four sizes), seven of plan 44 changed by
  the panel's six new rows alone; `TestMessageToFrame` added to the timing
  step. Step 5 next.
- 2026-10-25: step 5 done: 17 of 17 PASS
  ([`doc/plan45-evaluation.md`](doc/plan45-evaluation.md)). Mutations 47 of
  47 caught (three made again, their first form not building); Windows
  Terminal at 208x59 and 120x40 on temporary repositories with a local bare
  remote: the field, the button, the dropdown, the hovered item, Push after
  the commit, Commit after the push, Ctrl+Enter's question, the colours read
  back; this repository and the clipboard unchanged. Two changes after the
  first terminal run, both run again: while the message field has the keys,
  zone B's draft beside the panel takes no paste or image; a duplicate
  branch of `stepDone` merged. On the final tree build, vet, gofmt clean,
  `go test ./...` ok (781 tests), `-race` on render, app and input ok, the
  timing step's eleven tests within their targets (TestMessageToFrame p95
  1.03 and 1.52 ms). Waiting for the owner's review; nothing committed.
- 2026-10-25: the owner asked for two additions, after the report's
  limitations: the staged files' contents compared, and the elapsed time
  with a cancel. Probe P45a done (What was found): git does not clean its
  lock when stopped, so a push alone can be cancelled, behind a check of its
  lock files; design D14 and D15, criteria 45.18 and 45.19, step 6 in
  progress.
- 2026-10-25: step 6's code and tests done: the staged blobs (Snapshot,
  Commit), the push's stop (GitStop, Push(dir, stop)), the elapsed time,
  Esc's cancel, the notes. Probes P45b to P45e (What was found): a break to
  a pid that leads no group reaches its parent, so git's pid is held while
  its stop is armed and the break runs under the stop's lock. TestCommitGit
  made to wait for git's first status read, which the race detector's load
  made slower than one wait. 34 of 34 new mutants caught; go test ./... ok
  (788 tests); Windows Terminal at 208x59 and 120x40: the commit's seconds,
  the push's hint and seconds, a refused stop, the cancel, no git process
  or lock left, the remote-tracking ref unchanged.
- 2026-10-25: step 6 done: -race on render, app and input ok; the timing
  step's eleven tests within their targets (TestMessageToFrame p95 0.86
  and 1.59 ms); the first report's terminal run made again on the final
  build; the documents and `doc/plan45-evaluation.md` updated, 19 of 19
  PASS. Waiting for the owner's review; nothing committed.

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
