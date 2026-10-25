# Plan 45 evaluation (2026-10-25)

The check of plan 45, Commit, Push and Commit & Push in Source Control
(brief amendment 46: under the panel's repository row a message field
"Message", a purple "✓ Commit" button with a separator and a chevron, and
under it, only after a click on the chevron, a dropdown with one item,
"Commit & Push"; Commit then Push as two actions, or both in turn; Git's
index respected, nothing forced, the button's state read from Git; after the
owner's image `references/15-commit-push.png`). It covers the 19 criteria of
[`plan.md`, Plan 45](../plan.md): the 17 written before the code, and 45.18
and 45.19, written for the owner's two additions after the first report (the
staged files' contents compared at the commit; a long operation's elapsed
time and a push's cancel). Nothing is committed: the owner reviews the
screenshots and the results first. No test commit or push touched this
repository: every Git test ran on temporary repositories outside it, their
remotes local bare repositories or a local server that never answers.

How it was checked:
- The finding from the code, git's documentation of `commit`, `push
  --porcelain`, `for-each-ref`, `status --branch` and `cat-file`, and the
  owner's image read pixel by pixel (`scratchpad\plan45\imgpx.ps1`;
  plan.md, What was found).
- Probe P45a, what git leaves when stopped, and probes P45b to P45e, where a
  console Ctrl+Break goes, each caller alone in a hidden console of its own
  (`scratchpad\plan45\probe` to `probe5`; plan.md, What was found).
- Deterministic tests, none starting a CLI, none touching this repository:
  - the engine, on temporary repositories made by real git with an empty
    global configuration, their remotes local bare repositories or a local
    server that accepts and never answers (`internal/input`):
    TestScrubURLs, TestSourceStage (7 cases), TestSourceCommit (20),
    TestSourcePushSucceeds (4), TestSourcePushRefused (12),
    TestSourcePushFails (4), TestSourceTraceForbidsForce,
    TestSourceStagedContents, TestSourcePushStop,
    TestCancelledPushNamesLocks, TestGitStopUnderLock; TestParseSnapshot
    and TestSourceBranch with the branch line's new fields, TestSourceStates
    with each staged entry's blob (their expectations extended, none
    removed);
  - the drawing (`internal/render`): TestCommitControlsText,
    TestCommitImages; TestSourcePanel's heights raised by the controls' six
    rows, its checks unchanged;
  - the application, keys and mouse messages through the program's update
    and its filter, git's writes from a test double (`internal/app`):
    TestCommitLayout, TestCommitMessageField, TestCommitValidation,
    TestCommitStaging, TestCommitMainAction (6 cases), TestCommitWorkflowA,
    TestCommitWorkflowB, TestCommitOneAtATime, TestCommitDropdown,
    TestCommitKeyboard, TestCommitLongOperation, TestCommitTicks,
    TestCommitImagesDrawn, TestCommitLiveLabel; and with git itself on a
    temporary repository and a bare remote, TestCommitGit;
    TestMessageToFrame in the timing step.
- 12 golden frames added (`commit-menu`, `commit-sixel`, `commit-push`, each
  at 80x24, 120x40, 200x60 and 79x24); 7 of the existing 208 changed, the
  Source Control frames, by the controls' six rows alone; none changed by
  the owner's two additions.
- The engine (`internal/input/sourcewrite.go`, the branch line in
  `explorer.go` and `source.go`) built by an implementer subagent to the API
  the plan's step 2 fixed; its report checked against its code and tests.
  The additions' engine (the staged blobs, `gitstop.go`) written directly.
- Each new behaviour broken on purpose through `go test -overlay`, nothing
  written in the repository (`scratchpad\plan45\mutate45.mjs`,
  `mutate45b.mjs`, `mutate45c.mjs`).
- Build, vet, gofmt, the whole suite, the race detector on render, app and
  input, the timing step alone, all on the final tree.
- Windows Terminal in mock mode, real keys, clicks, a pointer and a
  Ctrl+Enter chord, the final build: on temporary repositories with a local
  bare remote (`scratchpad\plan45\mkrepo45.mjs`, `scratchpad\term\p45.ps1`,
  `p45allf.ps1`), and for the additions on repositories whose pre-commit
  hook sleeps 8 s and whose upstream is a local server that never answers
  (`mkrepo45c.mjs`, `silent45.mjs`, `p45c.ps1` with steps that make and
  remove a lock file, `p45call.ps1`).

```text
Evaluation - Plan 45, Commit, Push and Commit & Push in Source Control - 19 criteria: 19 PASS
PASS 45.1 TestCommitLayout at 200x60: under the repository row the message box (its top on the next row, "Message"), the button three rows under it ("✓ Commit", the separator, "⌄"), the boxes from the panel's inner first column, the button as wide as the inner width (26); no dropdown by default; the chevron opens it right under the button, its three rows as wide as the button, "Commit & Push" once above the footer; no controls once git says no repository; TestCommitControlsText: the six rows exactly 26 cells, the box's 3 and the button's 3, with Sixel blank cells but the placeholder's row; golden frames commit-menu, commit-sixel and commit-push at 80x24, 120x40, 200x60 and 79x24, and the seven changed Source Control frames
PASS 45.2 TestCommitControlsText: the outlines, the inside, the placeholder (#71789E on #151722), the label on the fill (#ECE5F9 on #312C55), the separator #745EBA, the dimmed button halfway to the page, the focus outline in the title colour, the band #23213C only while hot, the notes' colours; TestCommitImages on the images: the fill, the four outlines, the one-pixel separator at the chevron part's edge, the chevron (bright pixels in its box, none elsewhere in its part), the label and the check (600 pixels at least), transparent corners and margins, the dimmed and focused colours, the dropdown's ground, outline and band, the message box's inside and outline and its four strips cut from it; TestCommitImagesDrawn: the raw sequence draws the button at its first cell, the four strips, the dropdown under the button, again when the item turns hot, and writes the dropdown's cells back once it closes, inside synchronized output
PASS 45.3 TestCommitMessageField: typing, Enter a new line ("Fix\nbody"), Shift+← selections typed over, Ctrl+Z, Ctrl+A and Ctrl+X, a bracketed paste of three lines kept byte for byte ("one\ntwo\tthree\nfour  "), never committing and never reaching zone B's draft; r, ?, z and F typed, no zoom, no key page; the text the same after the Explorer's tile, Source Control's again, and Esc to zone B; zone B's draft taking no paste or image while the field has the keys; no git write at all
PASS 45.4 TestCommitValidation: an empty message and "  \n  " say "Enter a commit message." in the attention colour, the field shown, its text kept, no git write; editing the message clears the note; with no change the button is dimmed and a click says "No changes to commit.", no git write
PASS 45.5 TestCommitStaging: staged changes commit the index alone, expected as shown (MM src/Main.java, A docs/readme.md), nothing staged by the dashboard; nothing staged asks "Nothing is staged. Stage the 4 changes listed under CHANGES and commit them?" with Stage and commit and Cancel; Esc and Cancel stage and commit nothing; Stage and commit stages exactly main.go, src/Main.java, gone.txt and web/, then commits them; a conflict says "Resolve the merge conflicts first." with no git write; TestSourceStage on real git: modified, deleted, untracked files, an untracked folder without its ignored file, a space, a bracket and a leading dash literal, exactly the listed paths, a project folder's paths; TestSourceCommit: a partially staged file's unstaged part still in git diff after the commit
PASS 45.6 TestSourceCommit on temporary repositories (20 cases): HEAD a new commit on the old one, the message as git cleans it (a trailing blank line and spaces dropped, "# not a comment" kept), a multibyte message byte for byte, a root commit, a rename found once both sides are staged, an untracked folder matching its folder, a resolved merge with two parents; refused with HEAD and the index unchanged: nothing staged, a failing pre-commit hook and a rejecting commit-msg hook (their output in the error), no identity (useConfigOnly), a conflict, staged changes outside the project (1 and 3), a staged set other than the expected (5 forms), a rebase, a cherry-pick, a revert and an am session in progress; TestSourceTraceForbidsForce and every test's trace: no --allow-empty, --amend or --no-verify
PASS 45.7 TestSourcePushSucceeds on a temporary repository with a local bare remote: the remote's main equals the local HEAD and the branch line says ahead 0 after, two commits pushed at once, an upstream of another name, push.followTags sending no tag; TestSourcePushRefused (12): no remote, no upstream with one and two remotes, a local upstream, a gone upstream, a pushRemote in three forms, push.default=nothing, simple with another name, nothing to push, a detached HEAD, each with its message and no push in the trace; TestSourcePushFails: fetch first, then non-fast-forward, the local and remote branches unchanged; the remote's pre-receive hook rejecting with its reason; an unreachable remote in git's words; a password in the URL never in the message; TestScrubURLs; no --force, --force-with-lease, -f, "+" refspec, --mirror, --all, --tags or --delete in any trace
PASS 45.8 TestParseSnapshot: "main...origin/main" (upstream), "[ahead 1]", "[behind 2]", "[ahead 1, behind 2]", "feature/x...fork/feature/x [ahead 12, behind 30]", "[gone]", "HEAD (no branch)", "No commits yet on main" with and without its upstream and gone, git's older "Initial commit on master", each exact; TestSourceBranch on real git: ahead 1 of origin/main
PASS 45.9 TestCommitMainAction, each a new model on its first read (a restart): changes with no upstream Commit; nothing Commit dimmed; ahead 1 Push; ahead 1 with changes Push; a gone upstream Commit; no remote Commit dimmed; the button's label on screen each time; TestCommitKeyboard: ahead 1 with a file staged, Push stays the main action and Commit & Push commits, then pushes all
PASS 45.10 TestCommitWorkflowA: a message and a click make one commit ("commit Add Source Control diff viewer"), the field cleared, "Committed abc0001." in green, the button on Push once the read says ahead 1; Push pushes, "Pushed to origin/main.", the button on Commit, not dimmed with changes left; TestCommitGit with git: the commit's subject in git log, Push, the bare remote's main equal to the local HEAD
PASS 45.11 TestCommitWorkflowB: Commit & Push runs target, commit, push in that order, "Committed abc0001 and pushed to origin/main.", the dropdown closed, the field cleared; a failing commit runs no push, keeps "Fails" and shows git's words in red; a rejected push says "Committed abc0002, but the push failed: The remote rejected the push: pre-receive hook declined.", clears the field and shows Push, which then pushes once and commits nothing (3 pushes, 2 commits made); a refused destination (no upstream) commits nothing and says why in the attention colour; TestCommitGit with git: Commit & Push against a bare remote whose pre-receive hook refuses keeps the commit, says so, then Push once the hook is gone, the remote's log "Second change, First change, init", each once, the working tree clean
PASS 45.12 TestCommitOneAtATime: ten clicks and three Ctrl+Enter during a held commit make one commit, the button saying "Committing…"; the chevron opens nothing and Commit & Push does nothing meanwhile; a failed commit is not retried
PASS 45.13 TestCommitDropdown with Sixel: hidden by default; the pointer over the button changes neither its look nor anything else; a click on the left part commits and opens nothing; the chevron opens it (its image recorded); a click in zone A closes it and changes nothing else (focus, cursor, writes); Esc closes it; the pointer on its item puts the band in its look, off it not; choosing the item closes it and pushes
PASS 45.14 TestCommitKeyboard: Home then ↑ gives the button the keys (its footer "Enter commit · → Commit & Push · …"), ↑ the field, ↓↓ back to the list; Enter on the button commits; r on the button types and reads nothing; → opens the dropdown with its item hot and Enter runs Commit & Push (target, commit, push); Esc gives zone B the focus; plan 44's list keys unchanged (TestSourceList, TestSourceHover, TestDiffKeys pass)
PASS 45.15 7 of the 208 existing golden frames changed (source at 80x24, 120x40 and 200x60, source-diff at 120x40 and 200x60, source-diff-wide at 120x40 and 200x60), each by the controls' six rows and the list moved down under them, nothing else; 12 added; none changed by the owner's two additions; plan 44's tests, the Explorer's, F3's, the draft's and the mouse's pass; adapted: TestSourcePanel's panel heights, 12 to 18 rows, so the same list shows under the controls
PASS 45.16 Build, vet, gofmt clean; go test ./... ok, 16 packages, 788 tests; -race on render, app and input ok (100 s, 330 s, 204 s), 0 data races; the timing step alone, eleven tests, the laptop on mains power (CPU 2% just before; VS Code, Edge, Teams, Paint and the desktop app open): TestMessageToFrame, typing in the message field with Sixel on, the field grown to three rows and scrolling, p95 0.86 ms at 120x40 and 1.59 ms at 208x59 (max 8.8 ms); the other ten within their targets (p95 at most 5.0 ms, the 50,000-line paste 2.2 ms at worst, 800 of 800 idle motion reports dropped in each case); mutations through go test -overlay: 81 of 81 caught, the first build's 47 (44, then the 3 whose first form left a variable unused and did not build, made again) and the additions' 34: the blobs not compared, the snapshot's blobs not kept, a staged deletion not "-", Snapshot reading none, the Explorer's read reading them; the lock check ignored, no lock given to the stop, the stop not under its lock, the tracking ref's and the branch's locks not watched, no break sent, the stop not given to git, the cancel not said, a lock left not named; Esc not cancelling, the seconds and the minutes not shown, a refused stop not said, the push's hint not said, "Cancelling…" not shown, Commit & Push's cancel said as a failure, no stop for a push, no first tick, the ticks not renewed, a stale tick renewed, one tick sequence for all, the live label kept, the live label drawn every frame; the busy note not drawn, the warning in grey, the old note shown while busy, the footer's cancel not said, a busy look not live, the live image kept; afterwards no mutant's text in the tree's Go files; one form not run: a stop never taken back after git's end, which would send a real Ctrl+Break to a released pid (TestSourcePushStop's last stop would catch it)
PASS 45.17 Windows Terminal, the final build (`scratchpad\term\acc-p45c.exe`, SHA-256 467d764d27073f29…), the first report's run made again on the build with the owner's two additions (`scratchpad\term\p45allf.ps1`): maximised in the acc tab's profile (208x59) and 120x40 in the default profile, mock mode on a temporary repository with a local bare remote, its upstream set (two staged, one modified, an untracked folder; `scratchpad\plan45\mkrepo45.mjs`, made again for each size), real keys, clicks, a pointer and a Ctrl+Enter chord: A, the message field and the button "✓ Commit"; B, the field given the keys by ↑ ↑ and "add commit and push" typed; C, a click on the chevron opening the dropdown under the button; D, the pointer on its item, the band, the footer "Enter Commit & Push · Esc close · F1 keys", off it none, Esc closing it; E, a click on the button committing, "Committed c42939f.", the button on Push; F, a click pushing, "Pushed to origin/main.", the bare remote's main at the commit; G, "next" typed and Ctrl+Enter asking "Nothing is staged. Stage the 2 changes listed under CHANGES and commit them?", Esc staging nothing; at 120x40 the same ("Committed a4011b0.", the remote's main at a4011b0); the colours not read again, the drawing of these states unchanged (its golden frames and image tests the same); no terminal-hosted agent session was open; this repository's HEAD, origin/main, status and stashes the same before and after both terminal runs; the clipboard unchanged
PASS 45.18 TestSourceStagedContents on a temporary repository: the snapshot's a.txt carries the index's blob (git rev-parse :0:a.txt), the staged deletion b.txt "-", the untracked c.txt none; a.txt staged again with other content refuses the commit with "The staged changes changed since they were shown: check them, then commit again.", HEAD and the index unchanged; staged back with the content shown, committed, HEAD:a.txt that content; TestSourceStates on real git: every staged entry's blob the index's, "-" for each staged deletion, none for the others; TestCommitStaging: the commit's expected entries carry the snapshot's blobs; TestGitStatus: the Explorer's read runs no cat-file
PASS 45.19 TestCommitLongOperation: "Pushing…" under 2 s, "Pushing… 12 s" and "Pushing… 1:05" with "Esc cancels the push." in grey and the footer "Esc cancel the push"; Esc while a lock file is held: "git is updating the repository: try again in a moment." in the attention colour, the push going on; Esc then: "Cancelling…", the note "The push was cancelled. The remote may have received it: Push again to check.", the button on Push; Commit & Push's push cancelled: "Committed abc0002; the push was cancelled. …"; Esc during a commit leaves it running; TestCommitTicks: a tick a second from the start, the label then "Committing… 3 s", a tick of an ended or earlier operation asking none; TestSourcePushStop on a temporary repository pushing to a local server that accepts and never answers: each of the five lock files (index, HEAD, packed-refs, the branch's, the remote-tracking ref's) refuses the stop with its message, the push going on; with none, the stop ends the push within 5 s with the cancelled refusal, every connection to the server closed within 5 s (no transport left), no lock file, the remote-tracking ref unchanged, a stop afterwards naming nothing; TestCancelledPushNamesLocks: a lock file left is named, not deleted; TestGitStopUnderLock: git's end waits for a stop under way; Windows Terminal at 208x59 and 120x40 (the final build, a pre-commit hook sleeping 8 s, the upstream a silent local server): "Committing… 3 s" in the button's image, "Committed …", the button on Push; a click, "Pushing… 5 s" and "Esc cancels the push."; a lock file made and Esc: the refusal in the attention colour, "Pushing… 8 s"; the lock removed and Esc: "The push was cancelled. …", the button on Push; the server's connection reset as Esc was pressed (10.4 s after it opened, at each size); after each run, the dashboard closed: no git process of the repository or the server, no lock file, origin/main unchanged, the commit made, the tree clean
```

Changed from the design, after the code or the engine's report:
- **The push destination.** Design D9 compared git's push destination with
  the upstream by `%(push:remoteref)`, which git 2.56 leaves empty; the
  engine compares `%(push)` with `%(upstream)` and their remotes, the same
  test. Under `push.default=simple` with a pushRemote, git gives no
  destination at all, said "Git's push settings do not push main to its
  upstream", not "send main to fork/main".
- **The push sends the branch alone**: `push.followTags` and
  `push.recurseSubmodules` are off for its call, so the owner's settings
  cannot add tags or submodules; an exit 0 with no ref line is a failure,
  not a success. An am session is told from a rebase by
  `rebase-apply/applying`.
- **The separator is one pixel wide** in the image, crisp at the cells'
  scale; the chevron's part is four cells in the image (the image's last
  sixth) and five in text, where the chevron is then centred.
- **The message starts in the box's second cell**, as in the image, where
  the first design kept a cell of padding.
- **A double click on the chevron opens the dropdown once.**
- **While the message field has the keys, zone B's draft beside the panel
  takes no paste and no image** (plan 43 gives it the draft's clipboard keys
  beside the panel otherwise): a Ctrl+V with an image on the clipboard does
  nothing there.
- **The staged blobs come from `git cat-file --batch-check`** of
  `:0:<path>` (design D14 first named `git diff --cached --raw`): one read,
  only for the staged entries of Source Control's read, a staged deletion
  "-"; the Explorer's read takes none. A path holding a line break, which
  cat-file cannot be asked, is compared by its path alone.
- **Past a minute the label reads "Pushing… 1:05"**, not "1 min 05 s",
  shorter in the button; the button's image is not kept while its label
  changes each second, nor its GDI label beyond the one showing.
- **The stop's safety, after probes P45b to P45e**: a Ctrl+Break sent to a
  pid that leads no group reaches that process's parent, the dashboard,
  which Bubble Tea would quit. Git's pid is held by a handle from its start
  until its end takes the stop back, and the lock check and the break run
  under the stop's lock, so a break names git, running or ended, and
  nothing else.
- **TestCommitGit waits for git's first status read** and the refresh's, as
  it already waited for each operation: under the race detector, with the
  other packages' tests running, one 300 ms wait was too short and the test
  failed once at its first check; alone it passed 3 of 3, and since the
  change in every run.

Observed and kept:
- **After a commit on a branch with an upstream, the main action is Push**,
  Ctrl+Enter included, as the owner's table asks; another commit before the
  push goes through Commit & Push, which commits, then pushes all.
- **A click on the button leaves the keyboard's focus where it was**: the
  focus outline shows only when ↑ or ↓ gave the button the keys.
- The field wraps by characters, as zone B's draft; Ctrl+C with nothing
  selected in it interrupts the turn, as in zone B.
- The message is kept across the panels and the focus, not across a restart.
- **Thin lines blend at the terminal's scale.** Windows Terminal draws an
  image's 10x20 pixels a cell into the acc tab's 9x17, so a horizontal
  outline of 1.2 pixels spreads over two device rows: the button's top edge
  reads `#584795` and `#463672` for `#8062D9`, its vertical edge `#7F60D7`;
  the message box's `#2E304E` outline reads `#222239`.
- **A stopped push exits 0xc000013a** (git's error carries it when the
  cancelled refusal is not made); git's transport, in its group, ends with
  it: the silent server saw its connection reset as Esc was pressed.
- **A Ctrl+Break the dashboard itself receives quits it**, as before plan
  45 (Bubble Tea's interrupt): the owner's own Ctrl+Break in Windows
  Terminal does that; the stop never sends one there.

Not verified:
- An SSH remote: git is told `SSH_ASKPASS_REQUIRE=force` with an askpass
  program that does not exist, so ssh fails instead of prompting on the
  dashboard's console (OpenSSH's documented behaviour); no SSH server was
  reachable to show it, nor a push over SSH cancelled.
- An HTTPS remote with Git Credential Manager: every push went to a local
  bare repository or the silent local server; a stop while its window shows
  was not tried.
- An agent staging files while a commit runs, live: the staged set's paths
  and contents are checked with the double and the engine's "changed"
  cases.
- "Cancelling…" on screen: it lasts less than the 700 ms between Esc and
  the next screenshot; checked through the application.
- A terminal without Sixel, on screen: the text form is the golden frames'.

Limitations:
- A commit, its hooks and the question's staging cannot be cancelled: git,
  stopped, leaves its lock files behind (probe P45a); a hook that hangs
  keeps "Committing… N s" until git ends.
- A push cannot be cancelled while git holds one of its lock files ("try
  again in a moment"), and a cancelled push may have reached the remote: the
  note says so, and Push again shows where it stands.
- The push goes to the configured upstream only: a branch with no upstream
  needs `git push -u` in a terminal once; push settings sending the branch
  elsewhere (a pushRemote) are refused, explained.
- The labels are the dashboard's monospace, where the image uses a
  proportional face.

The first report's block (2026-10-25, before the owner's two additions),
kept as it was given:

```text
Evaluation - Plan 45, Commit, Push and Commit & Push in Source Control - 17 criteria: 17 PASS
PASS 45.1 TestCommitLayout at 200x60: under the repository row the message box (its top on the next row, "Message"), the button three rows under it ("✓ Commit", the separator, "⌄"), the boxes from the panel's inner first column, the button as wide as the inner width (26); no dropdown by default; the chevron opens it right under the button, its three rows as wide as the button, "Commit & Push" once above the footer; no controls once git says no repository; TestCommitControlsText: the six rows exactly 26 cells, the box's 3 and the button's 3, with Sixel blank cells but the placeholder's row; golden frames commit-menu, commit-sixel and commit-push at 80x24, 120x40, 200x60 and 79x24, and the seven changed Source Control frames
PASS 45.2 TestCommitControlsText: the outlines, the inside, the placeholder (#71789E on #151722), the label on the fill (#ECE5F9 on #312C55), the separator #745EBA, the dimmed button halfway to the page, the focus outline in the title colour, the band #23213C only while hot, the notes' colours; TestCommitImages on the images: the fill, the four outlines, the one-pixel separator at the chevron part's edge, the chevron (bright pixels in its box, none elsewhere in its part), the label and the check (600 pixels at least), transparent corners and margins, the dimmed and focused colours, the dropdown's ground, outline and band, the message box's inside and outline and its four strips cut from it; TestCommitImagesDrawn: the raw sequence draws the button at its first cell, the four strips, the dropdown under the button, again when the item turns hot, and writes the dropdown's cells back once it closes, inside synchronized output
PASS 45.3 TestCommitMessageField: typing, Enter a new line ("Fix\nbody"), Shift+← selections typed over, Ctrl+Z, Ctrl+A and Ctrl+X, a bracketed paste of three lines kept byte for byte ("one\ntwo\tthree\nfour  "), never committing and never reaching zone B's draft; r, ?, z and F typed, no zoom, no key page; the text the same after the Explorer's tile, Source Control's again, and Esc to zone B; zone B's draft taking no paste or image while the field has the keys; no git write at all
PASS 45.4 TestCommitValidation: an empty message and "  \n  " say "Enter a commit message." in the attention colour, the field shown, its text kept, no git write; editing the message clears the note; with no change the button is dimmed and a click says "No changes to commit.", no git write
PASS 45.5 TestCommitStaging: staged changes commit the index alone, expected as shown (MM src/Main.java, A docs/readme.md), nothing staged by the dashboard; nothing staged asks "Nothing is staged. Stage the 4 changes listed under CHANGES and commit them?" with Stage and commit and Cancel; Esc and Cancel stage and commit nothing; Stage and commit stages exactly main.go, src/Main.java, gone.txt and web/, then commits them; a conflict says "Resolve the merge conflicts first." with no git write; TestSourceStage on real git: modified, deleted, untracked files, an untracked folder without its ignored file, a space, a bracket and a leading dash literal, exactly the listed paths, a project folder's paths; TestSourceCommit: a partially staged file's unstaged part still in git diff after the commit
PASS 45.6 TestSourceCommit on temporary repositories (20 cases): HEAD a new commit on the old one, the message as git cleans it (a trailing blank line and spaces dropped, "# not a comment" kept), a multibyte message byte for byte, a root commit, a rename found once both sides are staged, an untracked folder matching its folder, a resolved merge with two parents; refused with HEAD and the index unchanged: nothing staged, a failing pre-commit hook and a rejecting commit-msg hook (their output in the error), no identity (useConfigOnly), a conflict, staged changes outside the project (1 and 3), a staged set other than the expected (5 forms), a rebase, a cherry-pick, a revert and an am session in progress; TestSourceTraceForbidsForce and every test's trace: no --allow-empty, --amend or --no-verify
PASS 45.7 TestSourcePushSucceeds on a temporary repository with a local bare remote: the remote's main equals the local HEAD and the branch line says ahead 0 after, two commits pushed at once, an upstream of another name, push.followTags sending no tag; TestSourcePushRefused (12): no remote, no upstream with one and two remotes, a local upstream, a gone upstream, a pushRemote in three forms, push.default=nothing, simple with another name, nothing to push, a detached HEAD, each with its message and no push in the trace; TestSourcePushFails: fetch first, then non-fast-forward, the local and remote branches unchanged; the remote's pre-receive hook rejecting with its reason; an unreachable remote in git's words; a password in the URL never in the message; TestScrubURLs; no --force, --force-with-lease, -f, "+" refspec, --mirror, --all, --tags or --delete in any trace
PASS 45.8 TestParseSnapshot: "main...origin/main" (upstream), "[ahead 1]", "[behind 2]", "[ahead 1, behind 2]", "feature/x...fork/feature/x [ahead 12, behind 30]", "[gone]", "HEAD (no branch)", "No commits yet on main" with and without its upstream and gone, git's older "Initial commit on master", each exact; TestSourceBranch on real git: ahead 1 of origin/main
PASS 45.9 TestCommitMainAction, each a new model on its first read (a restart): changes with no upstream Commit; nothing Commit dimmed; ahead 1 Push; ahead 1 with changes Push; a gone upstream Commit; no remote Commit dimmed; the button's label on screen each time; TestCommitKeyboard: ahead 1 with a file staged, Push stays the main action and Commit & Push commits, then pushes all
PASS 45.10 TestCommitWorkflowA: a message and a click make one commit ("commit Add Source Control diff viewer"), the field cleared, "Committed abc0001." in green, the button on Push once the read says ahead 1; Push pushes, "Pushed to origin/main.", the button on Commit, not dimmed with changes left; TestCommitGit with git: the commit's subject in git log, Push, the bare remote's main equal to the local HEAD
PASS 45.11 TestCommitWorkflowB: Commit & Push runs target, commit, push in that order, "Committed abc0001 and pushed to origin/main.", the dropdown closed, the field cleared; a failing commit runs no push, keeps "Fails" and shows git's words in red; a rejected push says "Committed abc0002, but the push failed: The remote rejected the push: pre-receive hook declined.", clears the field and shows Push, which then pushes once and commits nothing (3 pushes, 2 commits made); a refused destination (no upstream) commits nothing and says why in the attention colour; TestCommitGit with git: Commit & Push against a bare remote whose pre-receive hook refuses keeps the commit, says so, then Push once the hook is gone, the remote's log "Second change, First change, init", each once, the working tree clean
PASS 45.12 TestCommitOneAtATime: ten clicks and three Ctrl+Enter during a held commit make one commit, the button saying "Committing…"; the chevron opens nothing and Commit & Push does nothing meanwhile; a failed commit is not retried
PASS 45.13 TestCommitDropdown with Sixel: hidden by default; the pointer over the button changes neither its look nor anything else; a click on the left part commits and opens nothing; the chevron opens it (its image recorded); a click in zone A closes it and changes nothing else (focus, cursor, writes); Esc closes it; the pointer on its item puts the band in its look, off it not; choosing the item closes it and pushes
PASS 45.14 TestCommitKeyboard: Home then ↑ gives the button the keys (its footer "Enter commit · → Commit & Push · …"), ↑ the field, ↓↓ back to the list; Enter on the button commits; r on the button types and reads nothing; → opens the dropdown with its item hot and Enter runs Commit & Push (target, commit, push); Esc gives zone B the focus; plan 44's list keys unchanged (TestSourceList, TestSourceHover, TestDiffKeys pass)
PASS 45.15 7 of the 208 existing golden frames changed (source at 80x24, 120x40 and 200x60, source-diff at 120x40 and 200x60, source-diff-wide at 120x40 and 200x60), each by the controls' six rows and the list moved down under them, nothing else; 12 added; plan 44's tests, the Explorer's, F3's, the draft's and the mouse's pass; adapted: TestSourcePanel's panel heights, 12 to 18 rows, so the same list shows under the controls
PASS 45.16 Build, vet, gofmt clean; go test ./... ok, 16 packages, 781 tests; -race on render, app and input ok (95 s, 338 s, 203 s), 0 data races; the timing step alone, eleven tests, the laptop on mains power: TestMessageToFrame, typing in the message field with Sixel on, the field grown to three rows and scrolling, p95 1.03 ms at 120x40 and 1.52 ms at 208x59 (max 14.6 ms); the other ten within their targets as before (p95 at most 4.9 ms, the 50,000-line paste 1.7 ms at worst, 800 of 800 idle motion reports dropped in each case); mutations through go test -overlay: 47 of 47 caught (44, then the 3 whose first form left a variable unused and did not build, made again: the chevron drawn in the fill, ahead kept 0, push settings never refused)
PASS 45.17 Windows Terminal, the final build (`scratchpad\term\acc-p45.exe`, SHA-256 132ec4592dbf940b…), maximised in the acc tab's profile (208x59) and 120x40 in the default profile, mock mode on a temporary repository with a local bare remote, its upstream set (two staged, one modified, an untracked folder; `scratchpad\plan45\mkrepo45.mjs`, made again for each size), real keys, clicks, a pointer and a Ctrl+Enter chord: A, the message field with its placeholder and the button "✓ Commit" with its separator and chevron, the list's cursor; B, the field given the keys by ↑ ↑ and "add commit and push" typed; C, a click on the chevron opening the dropdown right under the button, its edges the button's; D, the pointer on its item, the rounded band, off it none, Esc closing it; E, a click on the button committing, "Committed 361b575." in green, the button on Push, the field cleared; F, a click pushing, "Pushed to origin/main.", the button on Commit, the bare remote's main at the commit; G, "next" typed and Ctrl+Enter asking "Nothing is staged. Stage the 2 changes listed under CHANGES and commit them?", Esc staging nothing; at 120x40 the same, the panel in zone B's place ("Committed 48a7f17.", the remote's main at it, app.go and notes/ left as they were); the colours read back (`scratchpad\term\px44.ps1`, `pick45.ps1`): the box's inside #151722 and placeholder #71789E, the button's fill #302B54, label #E6DFF4, vertical outline #7F60D7, separator #6F5BB4, the dropdown's ground #0F0F1A, outline #915CF0, item text #EAE4F4, band #24213D, the note #4ADE80, the question's choice on #272745; no terminal-hosted agent session was open; this repository's HEAD, origin/main, log, status and stashes the same before and after; the clipboard unchanged
```

**The commit and the push (2026-10-25), after the owner's approval.** The
owner approved plan 45 on 2026-10-25; commit f71e022 (55 files: 29
modified, 26 added) and the approval commit were pushed to `origin`;
`origin/main` equals `main`, the working tree is clean, and the author is
`Evengelius <molimoabel@gmail.com>`, with no attribution line: PASS. The
blocks above stay as they were reported.
