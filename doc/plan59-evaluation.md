# Plan 59 evaluation (2026-10-28)

The check of plan 59, the branches popup (brief amendment 61): a click on
the project bar's branch name, or Alt+B, opens a popup of the repository's
branches and tags, after the owner's image of an IDE's branch popup and
with only what is relevant: a search, Recent, Local, Remote and Tags, a
message where a section is empty; a switch, a fetch, a name copied; the
header's branch kept current. New branch is left out by the owner.

It covers the 10 criteria of [`plan.md`, Plan 59](../plan.md), written
before the code. Nothing is committed or pushed: the owner reviews first.

How it was checked:
- **Input tests on real repositories** (`internal/input/branches_test.go`,
  6 tests): the reads, the switches, the fetch, each worktree's HEAD file;
  every git call traced for a force, a stash, a reset, a merge or a prune.
- **Rendering tests** (`internal/render/branches_test.go`, 4) and **golden
  frames** `branches`, `branches-search` and `branches-confirm` at 80x24,
  120x40, 200x60 and 79x24.
- **State test** `TestBranchSwitched`; **application tests**
  (`internal/app/branches_test.go`, 8) with a fake git, the HEAD watch on
  real repositories (the project, and a linked worktree as the project).
- **Each change broken on purpose** through `go test -overlay`
  (`scratchpad\plan51\mutrun`, the lists built from the source lines by
  `scratchpad\plan59\mkmutants.mjs` and `mkmutants2.mjs`), the repository
  never written: `git diff` and `git status` hashed the same before and
  after each run.
- **Windows Terminal**: the real application in the acc tab, maximised
  (208x59) and at 120x40, a temporary home and app data, on a demo
  repository, its Claude Code session started with no message sent
  (`scratchpad\term\p59run.ps1` on `p51.ps1`, with `p59demo.ps1` and
  `p59ext.ps1`); confcheck before and after.
- **The timing step** alone, and **build, vet, gofmt, the whole suite**
  after the last code and document edit.

## The evaluation

```text
Evaluation - Plan 59, the branches popup - 10 criteria: 9 PASS, 0 FAIL
PASS 59.1 Opening and closing: TestBranchesOpenClose: a click on "branch main" opens the popup (the lists read once, Recent and Local open, Remote and Tags folded, the cursor on main) and a second click closes it; Alt+B opens and closes it; Esc with the search empty closes it, Esc on a search clears it first; a click outside, on zone B's text, closes it and leaves the interface as it was before it opened, field for field, no drag started; a click on the Explorer's tile closes it and shows no panel; Alt+B closes the "+" menu, and a click on "+" closes the popup without opening the menu; no popup over the file viewer (Alt+B; a click on the name outside it closes the viewer only, plan 58) nor over the details view, drawn or not yet, the toast saying why; outside a repository no branch in the header and Alt+B's toast; TestBranchesFromPanels: Alt+B opens it with the focus in the Explorer, Source Control's list, its commit field and button, and a file's diff
PASS 59.2 The lists: TestBranchesRead on a real repository: Local feat, fix/login, main, old, wtb (wtb in another worktree, main current); Remote backup/feat, backup/main, origin/main (origin/HEAD left out); Tags v2, v3, v1, newest first; Recent feat, old, fix/login from the reflog, at most 5, the current branch left out; main → origin/main ahead 3, behind 1; TestBranchRows: the groups' first folding and the rows' order; TestBranchesKeys: Recent's rows carry Local's upstreams, the totals 2, 4, 4, 3
PASS 59.3 The search: TestBranchRows: "FEA" shows Recent, Local and Remote open with feat and origin/feat and hides Tags; "login" finds Feature/Login, case ignored; "zzz" gives "No branch or tag matches “zzz”"; TestBranchesKeys: typed (the cursor on the first match), Backspace, a paste (its first line, the draft untouched), Esc clearing it before closing
PASS 59.4 Messages: TestBranchRows: "No recent branches", the cap's "first 2 shown"; TestBranchesPopupStates: "reading…", "git not found", "detached at a1b2c3d", "main, no commits yet", a long note of git's in three rows, the last cut with "…"; TestBranchesProblems: "reading…" while git reads, "git not found", "not a git repository", git's own error in the list's place; TestBranchesHead: detached, and a branch with no commits; TestBranchesCap: 1,001 tags, 1,000 shown
PASS 59.5 Switching: TestBranchSwitch with git itself: a local branch; origin/away made a branch away tracking it; origin/main refused with "A local branch main exists already: switch to it in Local."; git's two refusals of changes it would lose (a.txt differing on feat, away.txt missing on main), the files as they were; a.txt's change and the untracked new.txt carried; every call traced: no force, stash, reset, merge or prune; TestBranchesSwitch: Enter and a double click; with 3 changes the confirmation, Esc and another key cancelling it, Enter switching; refused while a turn runs ("the agent is working in this folder: switch once its turn ends"); wtb's note; the current branch and a tag asking git nothing; TestBranchesPopupStates: the confirmation's words, plural and singular
PASS 59.6 After a switch: TestBranchesSwitch: the popup closed; the header "branch feat"; one entry, Role "dashboard", "Switched from main to feat · the agent's conversation still reflects main", with "· 3 uncommitted changes came along" when they did; nothing sent to the CLI; the empty draft's note, a draft holding text kept; the Explorer's markers read again (200x60), and Source Control's status when it shows instead; TestBranchSwitched: 0, 1 and 3 changes, with and without a session
PASS 59.7 Fetch: TestBranchFetch with git itself: --all --no-prune, a remote branch deleted upstream kept; a missing remote failing; TestBranchesFetch: F5 and a click on "fetch"; "fetching… Esc stops"; Esc stopping it ("The fetch was stopped."); a stop refused while git holds a lock showing why, the next Esc stopping it; FetchLimit's stop ("The fetch ran past its 2 minutes and was stopped."); a failure in git's words; "Fetched every remote."; the lists read again after each; a fetch outliving the popup, shown again when it reopens and ending in a toast once closed; the quit question naming "the fetch"
PASS 59.8 The header kept current: TestBranchesHeadWatch on real repositories, the watch started by the validation: a switch made with git outside the dashboard, in the project and in a linked worktree that is the project, shown in the header at the next check; a check with no switch, and a switch in the main worktree, leaving it as it was; no repository, no watch (the HEAD file never asked); in Windows Terminal the header back on main 1.9 s after a switch made outside
PASS 59.10 The checks: go build ./... ok; go vet ./... ok (Windows and Linux); gofmt -l . empty; go test ./... ok (16 packages); 12 new golden frames reviewed (branches, branches-search, branches-confirm at 80x24, 120x40, 200x60 and 79x24), no other golden frame or mock file changed; 35 mutants through -overlay in two rounds, each caught by a failing test in the end, none by a build failure (details), the tree's hashes the same around each run; the timing step alone (mains power, CPU 17 %, the owner's usual applications open): its seventeen tests within their targets, the popup's 3,000 refs p50 0.55 ms, p95 1.51 ms from a key to its frame (target 16 ms)
Not verified: 59.9's last part, the owner's visual review. Its measured part holds: in Windows Terminal at 208x59 and 120x40, Alt+B reached the dashboard and opened the popup, the confirmation named the change, Enter switched (the header, zone B's notice, the draft's note), a click on the name reopened it on the new branch, a switch made outside showed in the header 1.9 s later, a search narrowed the lists; confcheck PASS; review shots out-p59\branches-max\s2.png, s6.png, s11.png, branches-90x30\s4.png, s5.png
Verdict: NOT DONE
Proposed next step: the owner's visual review of the popup (the shots) - waiting for approval
```

## Details

### What the checks found and changed

- **Alt+B from the docked panels.** The Explorer, Source Control, its
  commit field and button, and a file's diff each take the keys while they
  have the focus and pass on a fixed list of global keys; the list had
  Alt+P, not Alt+B, so with the focus in any of them Alt+B did nothing.
  Added to the four lists (`explorerKey`, `sourceKey`, `globalKey`,
  `diffViewKey`); `TestBranchesFromPanels` checks each place.
- **A popup over a view not drawn yet.** `toggleBranches` first judged by
  the frame drawn last; Alt+B right after a full view opened, before its
  frame, opened the popup under it. It now asks `render.Layers` too.
- **The frame's cost.** With 3,000 refs the first version took p95 15 ms
  from a key to its frame: the list copied every group each frame, drew
  every row and made every row's key. Now only the rows in view are drawn,
  the widths counted without drawing (`branchAsideWidth`, tested against the
  drawing), the keys made when asked.
- **One change in the singular.** The Windows Terminal run's single
  uncommitted change read "1 uncommitted change. Git keeps them…"; now "Git
  keeps it on feat unless it touches a file that differs there; then Git
  refuses and nothing changes."
- **A note of git's that runs long** takes three rows, the last cut with
  "…", so that the box keeps its list on screen.
- **A slip of mine, repaired.** To see whether the Ctrl+V release check
  could fail, `paste.go` was mutated in the tree with sed instead of through
  an overlay, and the mutation then undone with `git checkout --`, which
  also removed this plan's own edit of that file. The edit was applied
  again from the session's record and checked in `git diff`; the sed had
  matched nothing, so that probe proved nothing, and the check was proved
  by the overlay mutant A10 instead.

### The mutants

Each through `go test -overlay` (`scratchpad\plan51\mutrun`, the lists
built from the source lines by `scratchpad\plan59\mkmutants.mjs` and
`mkmutants2.mjs`), the repository never written: `git diff` and `git
status` hashed the same before and after each run (77b45de2 and 3a3c7bd2
around the first; 37889e5f and 36db427d around the second).

| Id | The change | Caught by |
| - | - | - |
| I1 | a remote's HEAD listed | TestBranchesRead |
| I2 | Recent names the current branch | TestBranchesRead |
| I3 | a remote branch switched to without --track | TestBranchSwitch |
| I4 | a local name taken not refused | TestBranchSwitch |
| I5 | the fetch prunes | TestBranchFetch |
| I6 | tags oldest first | TestBranchesRead |
| I7 | another worktree's branch not marked, the current one marked | TestBranchesRead |
| R1 | a folded group shows its rows | TestBranchRows |
| R2 | the search minds the case | TestBranchRows |
| R3 | no cap's note | TestBranchRows (second round) |
| R4 | the cursor matches a name in any group | TestBranchesKeys |
| R5 | the list's first rows drawn when scrolled | TestBranchesPopupStates (second round) |
| R6 | one change in the plural | TestBranchesPopupStates |
| R7 | a long note's rows uncapped | TestBranchesPopupStates |
| R8 | ahead counted a cell short | TestBranchAsideWidth |
| S1 | the agent's conversation never named | TestBranchSwitched |
| S2 | one change in the plural | TestBranchSwitched |
| A1 | a switch during a turn | TestBranchesSwitch |
| A2 | no confirmation with changes | TestBranchesSwitch |
| A3 | the note added to a draft with text | TestBranchesSwitch |
| A4 | Esc closes before it clears the search | TestBranchesOpenClose |
| A5 | a click outside closes it and acts too | TestBranchesOpenClose |
| A6 | the watch's branch not shown | TestBranchesHeadWatch |
| A7 | the lists not read after a fetch | TestBranchesFetch |
| A8 | FetchLimit's stop said to be the owner's | TestBranchesFetch |
| A9 | opened over a full view not drawn yet | TestBranchesOpenClose |
| A10 | Ctrl+V's lone release reads an image under the popup | TestBranchesKeys (second round) |
| A11 | the quit question does not name the fetch | TestBranchesFetch |
| A12 | the wheel down moves nothing | TestBranchesKeys |
| A13 | a heading's second click folds it again | TestBranchesKeys |
| A14 | Source Control not read again after a switch | TestBranchesSwitch |
| A15 | Source Control's list keeps Alt+B | TestBranchesFromPanels |
| A16 | the diff keeps Alt+B | TestBranchesFromPanels |
| A17 | the commit field and button keep Alt+B | TestBranchesFromPanels |
| A18 | the Explorer keeps Alt+B | TestBranchesFromPanels |

In the first round R3 failed to build (it named `BranchCap`, another
package's), so it proved nothing; rewritten, the cap's check catches it. R5
and A10 survived: no test checked which rows a scrolled list draws, and the
Ctrl+V release check could not fail, the test's frozen clock putting the
search's paste just before the release, which the dashboard then takes for
the terminal's own paste. Both tests were strengthened and both mutants
caught. A14 to A18 cover the second round's additions.

### The Windows Terminal run

`scratchpad\term\p59run.ps1` on `p51.ps1`: the real application in the acc
tab, maximised (208x59) and at 120x40, with a temporary home, local and
roaming app data and a PATH of System32, Git and the Claude Code CLI's
folder; CLAUDE_* and ANTHROPIC_* cleared, updates off. The project was a
demo repository outside this one (`p59demo.ps1`): main one commit ahead,
feat on origin, fix/login and old switched to once each, wtb in a linked
worktree, origin (its HEAD set) and backup, two tags, a.txt changed in the
working tree. Claude Code 2.1.303 started there and reached Ready with no
message sent, so no model was called: a mock run reads no branch, and the
header needs one.

Each step's shot and text are in `scratchpad\term\out-p59\branches-max` and
`branches-90x30`: s1 the dashboard on main; s2 Alt+B, the popup (Nerd Font
icons; Recent old, fix/login, feat; wtb "in another worktree"); s4 ↑ ↑ and
Enter on feat, the confirmation naming the one change; s5 Enter, the
switch: "branch feat" in the header, the grey notice in zone B, the note in
the draft; s6 a click on "feat" in the header, the popup with "feat →
origin/feat" and Recent main, old, fix/login; s9 the header back on main
1.9 s after `p59ext.ps1` switched the repository outside the dashboard; s11
Alt+B and "fix", fix/login in Recent and Local, the cursor on the first.
confcheck before and after: PASS, every file identical (the owner's
desktop and VS Code sessions ran meanwhile). No dashboard, Claude or git
process was left; the demo repository ended on main with its change
carried. The only Claude and Codex processes running before were the
desktop app's and VS Code's, none terminal-hosted, so none was closed.

The run used the build before the second round's Alt+B fix; that fix
changes only which keys the docked panels pass on, which the run did not
exercise (its Alt+B presses came with the focus in zone B), and
`TestBranchesFromPanels` covers it.

### The timing step

Run alone with the exact command of `doc/evaluation.md`, Keystroke timing,
`TestBranchesFrameCost` now among its tests: mains power, the battery at
96 %, the CPU 17 % just before, the owner's usual applications open
(Claude, VS Code, IntelliJ IDEA, Edge, Teams, Outlook, Notepad, Windows
Terminal). Each of the seventeen within its target:

| Test | Result | Target |
| - | - | - |
| TestBranchesFrameCost | p50 0.55 ms, p95 1.51 ms | p95 ≤ 16 ms |
| TestKeystrokeToFrame | p95 1.01 ms; the draft window 1.24 ms | p95 ≤ 16 ms |
| TestDraftKeystrokes | p95 2.54 ms | p95 ≤ 16 ms |
| TestLongSession | p95 3.01 ms | p95 ≤ 16 ms |
| TestThinkingFrame | p95 2.94 ms | p95 ≤ 16 ms |
| TestPasteToFrame | max 1.76 ms | ≤ 100 ms |
| TestStreamToFrame | p95 4.51 ms | p95 ≤ 16 ms |
| TestSelectorToFrame | p95 1.01 to 1.16 ms | p95 ≤ 16 ms |
| TestMotionToFrame | p95 1.02 to 2.00 ms | p95 ≤ 16 ms |
| TestDiffToFrame | p95 0.99 to 1.12 ms | p95 ≤ 16 ms |
| TestMessageToFrame | p95 1.03 and 1.51 ms | p95 ≤ 16 ms |
| TestOutputToFrame | p95 1.03 ms | p95 ≤ 16 ms |
| TestToolsToFrame | p95 2.38 ms | p95 ≤ 16 ms |
| TestBannerToFrame | p95 1.58 ms | p95 ≤ 2 ms |
| TestHomeKeysToFrame | p95 0.13 ms | p95 ≤ 16 ms |
| TestHomePictureTime | max 184.7 ms | ≤ 260 ms |
| TestKeystrokeCost | reports only | none |

**59.9, the owner's visual review (2026-10-28).** The owner reviewed the
review shots (the popup at 208x59, the confirmation and the switch at
120x40, the popup reopened on the new branch, a search) and wrote "The
branches popup looks right: plan 59's visual review is done": 59.9's last
part, PASS, so 10 criteria, 10 PASS, 0 FAIL. The popup is kept as the
shots show it. Not committed: the commit waits for the owner's approval.
The block above stays as it was reported.

**Approved and committed (2026-10-28).** The owner wrote "Plan 59 is
approved: review git status and the diff, then commit it." Commit 5c8c9be
(46 files: 25 modified, 21 added), from the files reviewed in `git diff`;
the whole suite had passed after the last code edit and after the last
edit of a document a test reads. The review found `doc/evaluation.md`
damaged by a script of this plan: adding `TestBranchesFrameCost` to the
timing step's command, a JavaScript `String.replace` read the command's
`$'` as "the text after the match" and copied the rest of the document
into the command line. The file was taken back from the last commit, it
had no other change, and its two intended edits were made again, its diff
checked to hold only them; the timing step itself had run its own command
line, not the document's. The author is `Evengelius
<molimoabel@gmail.com>`, with no attribution line, and the working tree
was clean after it: PASS. Not pushed: the owner asked for the commit only.
The block above stays as it was reported.
