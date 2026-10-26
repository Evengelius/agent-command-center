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

## Plan 50 - Source Control multi-selection

Brief amendment 51 (2026-10-26): Ctrl + left-click selects several files in
Source Control, and Commit and Commit & Push commit only those. A plain
click keeps today's behaviour, and, at the owner's word after the
evaluation, clears the selection when it opens a file's diff; the first
file selected becomes the active one, its diff shown, and the files added
after it leave the diff as it is. With none selected, Commit and Commit &
Push commit the whole index, as before, whatever diff shows.
The selection says what is committed, the active file which diff is shown.
Unselected files, staged or not, are left as they are, and amendment 46's
staging rules hold: only staged content is committed, never a change left
unstaged, and nothing is staged without the question's yes.

The criteria below were written before the code. Nothing is committed until
the owner has reviewed the result, and nothing is pushed without the
owner's word.

### What was found (2026-10-26, on eeec3b7)

- **The click.** A click on a Source Control row moves the cursor and acts
  as Enter: a file's diff opens, its section's comparison
  (`app/source.go:476`, `:406`, `:543`; ui-design.md, Source Control, "The
  mouse"). The click's modifiers reach `mouse()` (`tea.Mouse.Mod`, Shift
  read at `app/mouse.go:318`), not `sourceClick`.
- **Windows Terminal 1.23.20211.0** (microsoft/terminal at that tag) sends
  a Ctrl+click to the application with its Ctrl bit, 0x10 of the SGR
  encoding (`src/terminal/input/mouseInput.cpp:239`), unless a hyperlink is
  under the pointer, which Ctrl+click opens instead
  (`src/cascadia/TerminalControl/ControlInteractivity.cpp:249-264`); only
  Shift keeps a click from the application (`:651-659`). The dashboard
  draws no hyperlink. ultraviolet reads the bit from an SGR report
  (`third_party/ultraviolet/decoder.go:1661-1682`) and from a console mouse
  event (`terminal_reader_windows.go:125-137`).
- **The commit** (plan 45). Commit commits the whole index after its checks
  (`input/sourcewrite.go:242`): no rebase, am, cherry-pick or revert, no
  conflict, nothing staged outside the project, the staged paths and blobs
  those shown (designs D8, D14). With nothing staged, the question stages
  every change first (design D6). Nothing commits part of the index.
- **`git commit --only`** (git 2.56, `scratchpad\p50-git-only.sh`, in a
  scratch repository):
  - a fully staged file: committed alone, another staged file left staged;
  - a partly staged file (`MM`): its unstaged line committed too;
  - half a rename: accepted, an add with no delete; both sides: `R100`;
  - an untracked file: "pathspec 'u.txt' did not match any file(s) known
    to git";
  - a file deleted from the index but on disk: its disk copy taken (here,
    nothing left to commit);
  - a name with a space, a bracket and a leading dash, as a literal
    pathspec from a file, the message on standard input: committed;
  - the pre-commit hook runs, and sees the selection alone staged;
  - during a merge: "fatal: cannot do a partial commit during a merge."
- **A failure after the question's staging** (plan 45). Nothing unstages:
  the files stay staged, and the note shows git's words alone
  (`app/commit.go:370`); no test runs real git through it.

### Design

- **D1. The selection.** A set of project-relative file paths, held by the
  application (`SourceView.Selected`), not saved. A file row is a change's
  (not an untracked folder's) or a file's of an opened untracked folder; a
  path on two rows, staged and unstaged, is selected on both. Each status
  read drops the paths that are no longer a change (a folder's file: no
  longer under an untracked folder).
- **D2. Ctrl+click.** On a file row, Ctrl+click toggles its path. When it
  starts a selection (none before), it also does what a plain click on the
  row does: the cursor goes there and its diff opens. Otherwise the cursor
  and the diff stay as they are. On the panel's headers, folders, notes,
  blank rows and controls, Ctrl+click does nothing. A Ctrl+click counts as
  no click of a double click.
- **D3. The active file.** The file whose diff is open: a plain click's,
  Enter's, or the first Ctrl+click's. Deselecting it changes nothing else:
  its diff and the cursor stay.
- **D4. Plain clicks and keys** (changed at the owner's word after the
  evaluation, amendment 51). A plain click on a file's row clears the
  selection and shows that file's diff, as a click does today. A plain
  click on a header, a folder or the commit controls, Enter and the arrows
  do what they do today and leave the selection as it is. Esc on the list
  clears a selection; with none, it does what it does today. With none
  selected, Commit and Commit & Push are plan 45's, the whole index,
  whatever diff shows.
- **D5. The look.** A selected file's row: "▎" in its first cell in the
  title colour, on the sidebar's selected ground (`cSideSel`), its hover
  shade (`cSideSelHover`) under the pointer; the cursor's row keeps the
  band, with the "▎" when it is selected. No new colour.
- **D6. The button.** With a selection, the action is Commit, the branch
  ahead or not, and the label "Commit N files" ("Commit 1 file"); the
  dropdown's Commit & Push commits the selection, then pushes. The footer's
  hints and the key list name Ctrl+click and Esc.
- **D7. The commit.** Commit with a selection:
  1. at once, from the panel's status: a conflict, or a selected file
     partly staged (staged and unstaged changes, or deleted from the index
     with its file on disk), refuses, naming it;
  2. selected files with unstaged changes only, or untracked: the question
     (plan 45, design D6) offers to stage them, with their number; Cancel
     does nothing;
  3. a check before any write (`SourceWrite.CheckSelection`): a merge,
     rebase, am, cherry-pick or revert in progress; a conflict; a selected
     file changed since it was shown, partly staged, or a rename's source
     outside the project or back on disk: each refuses. The check reads
     git's status path by path (`--no-renames`), as `--only` takes the
     paths: a rename shown is its file added and its source deleted, and a
     file moved on disk with both its rows selected, which the staging
     makes a rename, stays two paths (the review, step 7);
  4. on the question's yes, the staging of exactly those files (`Stage`);
  5. the commit (`SourceWrite.CommitSelection`): the check again, the files
     the staging staged taken as they now are; then
     `git -c user.useConfigOnly=true commit --only --file=-`, the selected
     paths and a rename's source as literal pathspecs from a file
     (`GIT_LITERAL_PATHSPECS=1`, `--pathspec-from-file`,
     `--pathspec-file-nul`). Git builds the commit from HEAD in its own
     transaction, with the owner's hooks, clean-up and signing, and leaves
     the rest of the index as it is. No selected file has an unstaged
     change, so the working-tree content `--only` takes is its staged
     content;
  6. after it: HEAD a new commit on the old one, holding exactly the
     selected paths and their sources, each blob the one the check read
     from the index. A mismatch is told as a failure, the commit kept
     (nothing is rewritten), and Commit & Push does not push.
- **D8. A failure after the question's staging** (the owner's question,
  amendment 51). The staging is not undone: the files stay staged, as git
  leaves them after a failed `git commit`, their content on disk and in the
  index, the working tree and the other files untouched. Undoing it would
  write the index again after a failure, and could destroy what something
  else staged meanwhile. The note gives git's words, then "The N changes
  staged for it stay staged." ("The change staged for it stays staged.";
  "changes", not "files", at the owner's word after the evaluation), for
  this question and for plan 45's; the message is kept, and the selection;
  a retry asks nothing for the files now staged.
- **D9. The selection's end.** A commit made from it clears it; a refusal
  or a failure keeps it.

### Criteria

1. **50.1 The modifier.** In Windows Terminal, with `TEA_TRACE`, maximised
   and at 120x40: a Ctrl+left-click on a Source Control row arrives as
   `ESC[<16;x;yM`, a plain click as `ESC[<0;x;yM`; Ctrl held alone types
   nothing and does nothing in zone B's editor, the message field and the
   panel's list.
2. **50.2 Nothing changes without a selection.** No existing test changes
   what it expects; every golden frame and mock replay is unchanged.
3. **50.3 Ctrl+click.** State tests with `tea.ModCtrl` clicks: D2, D3, D4
   and D9 each as written (D4 as the owner changed it), on rows of both
   sections, a partly staged file's two rows, a folder's file, and the
   headers, folders, notes and controls.
4. **50.4 What is committed** (real git): the commit holds exactly the
   selection, for a fully staged file; an unstaged one and an untracked
   one, staged by the question; a folder's file; a staged deletion and an
   unstaged one; a rename, both sides; a file moved on disk, both its rows
   selected, neither side staged before, its file or its deletion (the
   review, step 7); names with a space, a bracket and a leading dash; a
   project in a subfolder of its repository (a file staged outside it left
   staged); a branch's first commit.
5. **50.5 What is left alone** (the same tests): the unselected files'
   index entries (`git ls-files -s`) and working-tree content byte for byte
   the same, staged, unstaged or partly staged; no working-tree file
   changed.
6. **50.6 Refusals before any write.** Each D7 refusal leaves the index file
   and HEAD byte for byte unchanged and says why: a partly staged file, one
   deleted from the index with its file on disk, a conflict, a merge, a
   rebase, a change since shown, a rename's source outside the project, a
   rename's source back on disk or back in the index (the review, step 7),
   an empty message.
7. **50.7 Hooks.** A pre-commit hook runs once and sees the selection alone
   staged; one that exits 1 makes no commit.
8. **50.8 A failure after the question** (amendment 51; real git, a
   pre-commit hook exiting 1), for the selection's question and for plan
   45's: HEAD unchanged; exactly the question's files staged, their blobs
   their working-tree content; the working tree and the other files
   unchanged; the note git's words, then "The N changes staged for it stay
   staged."; the message and the selection kept; once the hook is removed,
   Commit commits exactly them, asking nothing.
9. **50.9 Commit & Push.** The destination checked first; the selection
   committed, then pushed; a failed check after the commit pushes nothing.
10. **50.10 The look.** Golden frames of a selected row, a selected cursor
    row and a hovered selected row, and of "Commit N files", at 120x40 and
    208x59, each reviewed as a diff; the terminal run's PNGs (50.12) show
    them.
11. **50.11 Mutants.** Removing the partly staged refusal, the merge
    refusal, a rename's source from the pathspecs, the check of the
    committed paths, D8's note, or the first Ctrl+click's diff: each caught
    by a test.
12. **50.12 The terminal run.** In Windows Terminal, maximised then at
    120x40, on a demo repository: three files selected by Ctrl+click (one
    staged, one unstaged, one untracked), the question answered, Commit;
    git shows exactly those committed and the others as they were.
13. **50.13 The checks.** `go build ./...`, `go vet ./...`, `gofmt -l .`
    empty, `go test ./...`, `-race` on `internal/input` and `internal/app`,
    and the timing step run alone all pass, `TestAgentsMDWithinCodexLimit`
    included.
14. **50.14 Without a selection, the whole index** (added at the owner's
    word after the evaluation, amendment 51). A plain click on a file's row
    clears the selection and shows that file's diff; a plain click on a
    header, a folder or the controls keeps it. With none selected, Commit
    and Commit & Push run plan 45's whole-index commit (the double's log
    `commit <message>`, its expected entries every staged one; never
    `commit-sel`). In Windows Terminal, maximised then at 120x40: two files
    Ctrl+clicked, a third file plain-clicked, then Commit: git shows the
    whole index committed, the unstaged changes as they were.

### Steps

1. Record amendment 51 and this plan; then 50.1, the modifier, which
   decides whether the rest goes on.
2. `input`: `CheckSelection` and `CommitSelection`, with their real-git
   tests.
3. `app`: the selection, the clicks and Esc, the button, the question, the
   steps, D8's note.
4. `render`: the rows, the label, the hints and the key list.
5. The documents: ui-design.md (Source Control), architecture.md, and
   AGENTS.md (the hard constraint, the status).
6. Verification (50.2 to 50.13), then `doc/plan50-evaluation.md`.
7. The owner's change after the evaluation (amendment 51): D4's plain
   click, D8's words, 50.14; the checks again; a review of the complete
   diff for git safety, regressions and unintended changes; the evaluation
   brought up to date.

### Progress

- 2026-10-26: step 1 done; step 2 in progress.
  - Amendment 51 and this plan recorded.
  - 50.1 on today's build (eeec3b7, `scratchpad\term\acc-p50a.exe`), in
    Windows Terminal maximised (208x59, the acc profile) and at 120x40, the
    plan 45 repository, `TEA_TRACE` on (`term\p50.ps1`: p49.ps1 with
    `-trace` and the `cclick` and `ctrlhold` steps; `Term.CtrlClick` and
    `CtrlHold` release Ctrl in a `finally`):
    - the Ctrl+click on `reduce.go` read as `ESC[<16;186;22M` and
      `ESC[<16;60;22M`, then its `m`; the plain clicks as `ESC[<0;...M`;
      no other modifier bit on a click;
    - Ctrl held (12 key-downs, then the key-up; read as
      `ESC[17;29;0;1;40;1_`, up `ESC[17;29;0;0;32;1_`) in the list, the
      message field, zone B's editor and the diff view: no row of the
      screen changed, at both sizes (`out-p50a`, `out-p50b`);
    - no terminal-hosted Claude Code or Codex session was open (the
      desktop app's and VS Code's left alone); the clipboard unchanged.
- 2026-10-26: steps 2 to 5 done; step 6 in progress.
  - Step 2, `input` (`sourcewrite.go`, `source.go`): `CheckSelection`,
    `CommitSelection`, `WholeRefusal`, `merging`, `holds`, `blobsAt`,
    `Unexpected`; `Commit`'s failure mapping and its check of the new
    commit moved, unchanged, into `commitFailed` and `madeOn`, which both
    commits use. `selection_test.go`: 21 subtests on real git (what is
    committed, what is left alone, nine refusals with the index file and
    HEAD byte for byte unchanged, the hook's view, a refusing hook leaving
    the question's files staged, two Unexpected). The input package's
    tests pass whole (95 s).
  - Step 3, `app`: `sourceCtrlClick`, Esc, `stillChanged`,
    `commitSelected`, `selectionSteps` with `stepCheck`, `stayStaged`
    (a full stop added when git's words end with none),
    `clearSelection`; `selection_test.go`: `TestSelectionClicks`,
    `TestSelectionCommit`, `TestSelectionFailures` over the double,
    `TestSelectionGit` on real git (both questions, a hook exiting 1).
    The fake writes gained `CheckSelection` and `CommitSelection`.
  - Step 4, `render`: the selected row (`sourceRow`'s `sel`), "Commit N
    files", `AskSelText`, `sourceSelHints` (the hints without a selection
    unchanged), the key list's Ctrl+click row and Esc's words;
    `TestSourceRowSelected`; the golden case `source-selection` at the
    four sizes and 208x59, five new files, no other golden changed. Six
    existing calls of `sourceRow` and two of `sourceKeys` pass the new
    argument, "no selection"; what they expect is unchanged.
  - Step 5: ui-design.md (Source Control's key table, "The mouse", "The
    selection", "Commit", "A selection's commit"), architecture.md (plan
    50's table), AGENTS.md (the hard constraint, the MVP scope; 20,598
    bytes). Ctrl+click is told in prose, not as a row of a key table:
    `TestKeyOverlayRows` counts those rows (67).
  - Step 6, the mutants (`scratchpad\plan50\mutate50.mjs`, through
    `-overlay`; 428 s): 50.11's six caught, and 14 of 15 more. The one
    missed, the selection kept after its commit, showed D9 untested apart
    from the status read's pruning: `TestSelectionClearedByCommit` (a
    file listed again after its commit) added, and the mutant run again:
    caught, 21 of 21.
- 2026-10-26: step 6 done. The whole suite (179 s), `-race` on input and
  app (0 races), the timing step alone (every p95 at most 4.01 ms), the
  terminal run at both sizes (one commit of exactly the three files, the
  other three as they were): 13 of 13 PASS
  ([`doc/plan50-evaluation.md`](doc/plan50-evaluation.md)). Waiting for the
  owner's review; nothing committed.
- 2026-10-26: step 7, the owner's change after the evaluation (amendment
  51), done, and the review's defect fixed.
  - D4: a plain click on a file's row clears the selection (`sourceClick`);
    D8's note says "changes" (`stayStaged`); 50.3 and 50.8 changed, 50.14
    added. `TestSelectionClicks` checks D4 as changed;
    `TestSelectionWholeIndex` is 50.14 over the double.
  - The review of the complete diff (the orchestrator, and the verifier
    agent read-only) found one defect. A file moved on disk, both its rows
    selected (` D` and `??`), was refused once the question had staged
    them: git's status, asked for both paths, paired them as one rename, and
    the check found no entry for the source ("old.txt changed since it was
    shown"); the same with either side staged before. It failed safe (no
    commit, both files staged, the note true). Fixed test first: three
    real-git cases in `input` and one through the application in
    `TestSelectionGit`, each failing before; `selection()` reads
    `git status --no-renames` and checks path by path, a rename shown being
    its file, added, and its source, deleted and not back on disk (two
    refusal cases more). Two comments the plan had made stale corrected
    (`sourcewrite.go`'s header, `git.go`'s `runRead`).
  - D7.3 and criteria 50.4 and 50.6 extended with the fix; ui-design.md,
    architecture.md and the brief's note on amendment 51 with them.
  - Checks on the final tree: the whole suite (185 s); `-race` on input
    (137 s) and app (277 s), 0 races; the mutants 27 of 27 (737 s, six
    new: the click both ways, the fix four ways); the terminal runs on
    `acc-p50c.exe`, maximised and at 120x40 (`term\p50call2.ps1`,
    `out-p50d`): 50.12 again, and 50.14 (two files Ctrl+clicked, a third
    plain-clicked, the marks gone, Commit: the whole index committed, the
    unstaged changes and the untracked file as they were), the clipboard
    unchanged; the timing step alone (on mains, every p95 at most 4.52 ms):
    14 of 14 PASS
    ([`doc/plan50-evaluation.md`](doc/plan50-evaluation.md), the first
    evaluation's block kept as reported). Waiting for the owner's approval;
    nothing committed.

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
