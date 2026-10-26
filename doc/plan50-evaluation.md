# Plan 50 evaluation (2026-10-26)

The check of plan 50, Source Control multi-selection (brief amendment 51):
Ctrl+click selects files in Source Control, and Commit and Commit & Push
commit them alone, the rest of the index and the working tree left as
they are, amendment 46's staging rules kept; and, the owner's question, the
question's staging kept, and said to be kept, when the commit after it
fails.

It covers the 13 criteria of [`plan.md`, Plan 50](../plan.md), written
before the code, then step 7: the owner's change after the first
evaluation (a plain click on a file clears the selection, D8's note says
"changes", criterion 50.14 added) and the review of the complete diff the
owner asked for, which found one defect, fixed test first (below). The
first evaluation's block stays as it was reported; the second is the plan
as it now stands. Nothing is committed or pushed: the owner reviews first.
No git command touched this repository; every git test ran on temporary
repositories.

How it was checked:
- **The modifier first** (50.1, the go/no-go): Windows Terminal
  1.23.20211.0's source at that tag, then today's build in the terminal
  with `TEA_TRACE` (`scratchpad\term\p50.ps1`, `p50aall.ps1`,
  `p50ball.ps1`; `Term.CtrlClick` and `CtrlHold` release Ctrl in a
  `finally`).
- **`git commit --only` probed** before the design, on a scratch repository
  (`scratchpad\p50-git-only.sh`): what it does to each kind of entry, its
  hook, and a merge.
- **Real git** for the commit and its refusals (`internal/input`,
  `selection_test.go`) and for the owner's case through the application
  (`TestSelectionGit`); the double for the clicks and the flows.
- **Each change broken on purpose** through `go test -overlay`, nothing
  written in the repository (`scratchpad\plan50\mutate50.mjs`).
- **Build, vet, gofmt, the whole suite** after the last code and document
  edit, the race detector on the touched packages, the timing step alone.
- **Windows Terminal end to end**, maximised then at 120x40, on a demo
  repository with every kind of change (`scratchpad\term\p50call.ps1`,
  then `p50call2.ps1`; `scratchpad\plan50\mkrepo50.mjs`, `check50.mjs`,
  `check50d.mjs`).
- **The review** (step 7): the complete diff read for git safety,
  regressions and unintended changes, by the orchestrator and,
  independently and read-only, by the verifier agent.

## The first evaluation (steps 1 to 6)

```text
Evaluation - Plan 50, Source Control multi-selection - 13 criteria: 13 PASS, 0 FAIL
PASS 50.1 Windows Terminal 1.23.20211.0, TEA_TRACE, maximised (208x59) and 120x40: the Ctrl+click on reduce.go read as ESC[<16;186;22M and ESC[<16;60;22M, then its m; the plain clicks as ESC[<0;...M; no other modifier bit; Ctrl held (12 key-downs, then the key-up, read as ESC[17;29;0;1;40;1_ and ESC[17;29;0;0;32;1_) in the list, the message field, zone B's editor and the diff view changed no row of the screen at either size
PASS 50.2 no existing test changes what it expects (six calls of sourceRow and two of sourceKeys pass the new argument, "no selection"); TestGoldenFrames -update wrote five new files (source-selection) and changed none of the others; the hints without a selection are byte for byte as before; every mock replay and document test passes
PASS 50.3 TestSelectionClicks: the first Ctrl+click selects main.go, the cursor there, its diff open, the focus on it; the second adds docs/readme.md, the cursor and the diff as they were; src/Main.java's two rows toggle one path; deselecting main.go leaves its diff and the cursor; a plain click on gone.txt opens its diff, the selection kept; a Ctrl+click on a header, a folder, a blank row, the field and the button changes nothing and writes nothing; a folder's file selected; Esc clears, then Esc goes to zone B; a status read drops docs/readme.md; a Ctrl+click between two plain clicks makes no double click; TestSelectionClearedByCommit: a commit clears the selection though its file is a change again
PASS 50.4 TestSelectionCommit (input, real git): the commit changes exactly the selection for a fully staged file; an unstaged and an untracked one, staged by the question; a folder's file, its other file left untracked; a staged and an unstaged deletion; a rename, both sides; "a b.txt", "[x].txt" and "-dash.txt" (x.txt, which "[x].txt" matches as a pattern, left staged); a project in a subfolder (top.txt left staged); a branch's first commit
PASS 50.5 the same tests: the unselected files' git ls-files -s entries and working-tree SHA-256 the same, staged (b.txt), unstaged (c.txt), partly staged (p.txt) and untracked (u.txt); status after "M  b.txt", " M c.txt", "MM p.txt", "?? u.txt"
PASS 50.6 TestSelectionRefusals: nine refusals (a partly staged file; one deleted in the index, still on disk; a conflict; a merge; a rebase; a file staged since shown; one staged again with other content; a rename's source outside the project; nothing selected), each by CheckSelection and by CommitSelection, with its message, the index file's SHA-256 and HEAD unchanged; in the application a partly staged file and an empty field are refused before any write, a refused check before the staging (the double's log: the check alone)
PASS 50.7 TestSelectionHooks: the pre-commit hook's log "a.txt" once; a hook exiting 1 makes no commit
PASS 50.8 TestSelectionGit (real git, a pre-commit hook exiting 1), for the selection's question and for plan 45's: HEAD unchanged; b.txt and u.txt (then a.txt and v.txt) staged with their working-tree blobs (git hash-object); status "M  b.txt", "M  c.txt", "A  u.txt"; the working tree unchanged; the note git's words ("not today") then " The 2 files staged for it stay staged."; the field and the selection kept; the hook removed, Commit asks nothing and commits exactly them; over the double, "hook says no. The file staged for it stays staged."
PASS 50.9 TestSelectionCommit: Commit & Push writes target, commit-sel, push, then "Committed abc0003 and pushed to origin/main."; TestSelectionFailures: an Unexpected after the commit writes target and commit-sel, no push, and says "Committed abc0001, but it also holds x.go, which was not selected: check it in a terminal."
PASS 50.10 the golden frames source-selection at 120x40 and 208x59 (and 80x24, 200x60, 79x24): three rows with "▎" (the cursor's, a staged one, a hovered one) and "✓ Commit 3 files"; TestSourceRowSelected: the mark in the title colour, the grounds sideSelGround, sideSelHoverGround and bandGround; the terminal run's PNGs at both sizes show the three looks and the button's image "Commit 3 files"
PASS 50.11 mutate50.mjs through -overlay: the six caught (the partly staged refusal, "is no Refusal, want partial"; the merge refusal; a rename's source, "it leaves out old.txt"; the paths' check, "want an Unexpected"; D8's note, the note "hook says no"; the first Ctrl+click's diff), and 15 more; the one missed at first, the selection kept after its commit, caught once TestSelectionClearedByCommit was added: 21 of 21
PASS 50.12 Windows Terminal, maximised then 120x40, mkrepo50.mjs: guide.md (staged), app.go (unstaged) and TODO.md (untracked) Ctrl+clicked, "Commit three selected files" typed, Ctrl+Enter, the question, Enter: one new commit holding exactly TODO.md, docs/guide.md and internal/app/app.go; reduce.go (staged), main.go (unstaged) and README.md (partly staged) with the same index entry and bytes, at both sizes; the clipboard unchanged
PASS 50.13 go build ./... and go vet ./... ok, gofmt -l . empty; go test -count=1 ./... exit 0 in 179 s; -race on internal/input (142 s) and internal/app (350 s), 0 data races; the timing step alone, on mains, CPU 17% before, 13 PASS, every p95 at most 4.01 ms (target 16 ms); TestAgentsMDWithinCodexLimit passes, AGENTS.md 20,598 bytes
Not verified: a run with an agent CLI live (nothing of the CLIs changed; the terminal runs used the Claude mock); the selection with Codex as the provider in the terminal (Source Control is provider-independent); a second process editing a selected file during a real commit (the check after the commit is tested with hooks that change the commit, not with another process)
Verdict: DONE
```

## Step 7: the owner's change and the review

**The owner's change** (amendment 51, after the first evaluation). D4: a
plain click on a file's row clears the selection and shows that file's
diff (`sourceClick`); a plain click on a header, a folder or the commit
controls, Enter and the arrows keep it; Esc on the list clears it. With
none selected, Commit and Commit & Push are plan 45's whole-index commit,
whatever diff shows (criterion 50.14, new). D8: the note says "The N
changes staged for it stay staged." ("The change staged for it stays
staged."), for both questions (`stayStaged`). `TestSelectionClicks` checks
D4 as changed, `TestSelectionWholeIndex` is 50.14 over the double, and two
mutants check the click both ways.

**The review's findings.** Read for git safety, regressions and unintended
changes:
1. **A defect, fixed.** A file moved on disk shows as two rows, its
   source deleted (` D old.txt`) and the file untracked (`?? new.txt`).
   With both Ctrl+clicked, the question stages them, and git's status, asked
   for both paths, then pairs them as one rename (`R  old.txt ->
   new.txt`). The check, which looked each selected path up in that status,
   found no entry for `old.txt` and refused: "old.txt changed since it was
   shown: check the selection, then commit again. The 2 changes staged for
   it stay staged." The same with either side staged before ("new.txt
   changed since it was shown" when the file was). It failed safe: no
   commit, both files staged, the note true, a retry committing the rename
   once the panel showed it as one. Found by the verifier, confirmed by
   three new real-git cases in `internal/input` and one through the
   application in `TestSelectionGit`, all four failing before the fix. The
   fix (`input.selection`): the check reads git's status path by path
   (`git status --no-renames`), as `git commit --only` takes the paths and
   as the check after the commit (`holds`) already compared them; a
   rename shown in the panel is its file, added, and its source, deleted
   from the index and not back on disk. Two refusal cases pin the source's
   checks (back on disk, back in the index), and four mutants the fix.
2. **Comments made stale.** `sourcewrite.go`'s header held a line over 80
   columns: reflowed. `git.go`'s `runRead` said only rev-parse, status and
   cat-file go through it; it now names every read that does, `diff-tree`
   of plan 50's check after the commit among them (and `symbolic-ref`,
   `for-each-ref` and `remote`, plan 45's).
3. **Kept as designed, said here.**
   - A selected file stays selected while its section is folded or its
     untracked folder closed: the button's count ("Commit 3 files") and
     the question say how many, and Esc clears.
   - A selected file saved by an editor between the check and git's
     reading of it is committed as saved: `holds` tells it after the commit
     ("<path> changed during the commit, which holds the newer content"),
     the commit kept, never rewritten (D7.6).
   - The pathspec file is written to the system's temporary folder
     (`acc-selection-*.txt`, the paths alone) and removed after the commit.
   - Enter on a file and the arrows keep the selection (D4), so that the
     keyboard can read the selected files' diffs.
4. **No regression without a selection.** `Commit`'s failure mapping and
   its check of the new commit moved unchanged into `commitFailed` and
   `madeOn`; `stagedBlobs` asks git for the same `:0:<path>`; with
   `Selected` empty, `selection()` is nil and Commit, Commit & Push, Push,
   the button, the rows and the hints take plan 45's paths, byte for byte
   (the golden frames). The one change seen without a selection is D8's
   sentence after plan 45's question, as designed and tested.
5. **No unintended change.** No golden frame or mock file changed; the
   existing tests changed only for the two new arguments; every git call is
   one of D7's, with no `--no-verify`, `--amend`, `--force`, `reset` or
   unstaging anywhere.

```text
Evaluation - Plan 50, Source Control multi-selection, after step 7 - 14 criteria: 14 PASS, 0 FAIL
PASS 50.1 the input path unchanged since the first evaluation (mouse.go passes tea.ModCtrl to sourceClick; ultraviolet untouched); on the final build (acc-p50c.exe), Windows Terminal maximised (208x59) and 120x40, the runs of 50.12 and 50.14 Ctrl+clicked guide.md, app.go and TODO.md, each selected ("▎" rows in the runs' grids: 2, then 3)
PASS 50.2 no existing test changes what it expects; TestGoldenFrames passes with no frame rewritten in step 7 (the five source-selection files as first written, no other changed); every mock replay and document test passes
PASS 50.3 TestSelectionClicks, D4 as the owner changed it: the first Ctrl+click selects main.go, the cursor there, its diff open, the focus on it; the second adds docs/readme.md, the cursor and the diff as they were; src/Main.java's two rows toggle one path; deselecting main.go leaves its diff and the cursor; a plain click on gone.txt clears the selection and opens gone.txt's diff; a Ctrl+click on a header, a folder, a blank row, the field and the button changes nothing and writes nothing; a plain click on a header (folding, then unfolding it), on a folder and on the field keeps the selection; a folder's file selected; Enter keeps the selection; Esc on the list clears it, then Esc goes to zone B; a status read drops docs/readme.md; a Ctrl+click between two plain clicks makes no double click; TestSelectionClearedByCommit: a commit clears the selection though its file is a change again
PASS 50.4 TestSelectionCommit (input, real git): every case of the first evaluation, and a file moved on disk with both its rows selected, neither side staged before, its file staged, or its deletion staged: the check, the question's staging, then one commit "R100 old.txt new.txt", the status empty after; through the application (TestSelectionGit), b.txt moved to moved.txt on disk, both rows Ctrl+clicked, "Stage the 2 selected files and commit them?", Stage and commit: the commit "R100 b.txt moved.txt", the status empty, the selection cleared; all four failing before the fix ("old.txt changed since it was shown", "new.txt ..." with the file staged; through the application "b.txt changed since it was shown: check the selection, then commit again. The 2 changes staged for it stay staged.")
PASS 50.5 the same tests as in the first evaluation: the unselected files' git ls-files -s entries and working-tree SHA-256 the same, staged (b.txt), unstaged (c.txt), partly staged (p.txt) and untracked (u.txt); status after "M  b.txt", " M c.txt", "MM p.txt", "?? u.txt"
PASS 50.6 TestSelectionRefusals: eleven refusals, the first evaluation's nine and a rename's source back on disk ("old.txt, the source of the rename to new.txt, is back on disk: commit it from a terminal.") or back in the index ("new.txt changed since it was shown: check the selection, then commit again."), each by CheckSelection and by CommitSelection, with its message, the index file's SHA-256 and HEAD unchanged; in the application, as in the first evaluation, a partly staged file and an empty field refused before any write, a refused check before the staging (the double's log: the check alone)
PASS 50.7 TestSelectionHooks, as in the first evaluation: the pre-commit hook's log "a.txt" once; a hook exiting 1 makes no commit
PASS 50.8 TestSelectionGit (real git, a pre-commit hook exiting 1), for the selection's question and for plan 45's: HEAD unchanged; b.txt and u.txt (then a.txt and v.txt) staged with their working-tree blobs; status "M  b.txt", "M  c.txt", "A  u.txt"; the working tree unchanged; the note git's words ("not today"), then " The 2 changes staged for it stay staged."; the field and the selection kept; the hook removed, Commit asks nothing and commits exactly them; over the double (TestSelectionFailures), "hook says no. The change staged for it stays staged." and, for plan 45's question, "hook says no. The 2 changes staged for it stay staged."
PASS 50.9 as in the first evaluation: TestSelectionCommit's Commit & Push writes target, commit-sel, push, then "Committed abc0003 and pushed to origin/main."; TestSelectionFailures' Unexpected writes target and commit-sel, no push, and says "Committed abc0001, but it also holds x.go, which was not selected: check it in a terminal."
PASS 50.10 the golden frames as in the first evaluation, unchanged since; the step-7 runs' PNGs at both sizes show the three looks (a selected row, the hovered one, the cursor's band with its "▎") and the button's image "Commit 3 files", "Commit 2 files", then, after the plain click, no mark and "Commit"
PASS 50.11 mutate50.mjs through -overlay on the final tree (737 s): the six caught, and 21 more, 27 of 27, among them the owner's change both ways (a plain click on a file keeping the selection; a plain click anywhere clearing it) and the review's fix four ways (the check's status pairing renames; a rename's file compared as a rename; a rename's source not checked as deleted; a rename's source back on disk taken)
PASS 50.12 Windows Terminal, maximised (208x59) then 120x40, acc-p50c.exe, a fresh mkrepo50.mjs repository each time: guide.md (staged), app.go (unstaged) and TODO.md (untracked) Ctrl+clicked, "Commit three selected files" typed, Ctrl+Enter, the question, Enter: one new commit holding exactly TODO.md, docs/guide.md and internal/app/app.go; reduce.go (staged), main.go (unstaged) and README.md (partly staged) with the same index entry and bytes, at both sizes; the clipboard's formats and text SHA-256 the same before and after the four runs
PASS 50.13 go build ./... and go vet ./... ok, gofmt -l . empty; go test -count=1 ./... exit 0 in 185 s, after the last code edit (the later edits, to brief.md, plan.md and this file, are read by no test); -race on internal/input (137 s) and internal/app (277 s), 0 data races; the timing step alone, on mains, CPU 14 to 26% before, Chrome, Edge, VS Code, Outlook, Notepad, Windows Terminal and the Claude app open: 13 PASS in 51 s, every p95 at most 4.52 ms (target 16 ms); TestAgentsMDWithinCodexLimit passes, AGENTS.md 20,598 bytes
PASS 50.14 TestSelectionWholeIndex: docs/readme.md and main.go Ctrl+clicked, then a plain click on gone.txt: the selection empty, gone.txt's diff open, the button "Commit"; Commit writes "commit Whole" (plan 45's commit), expecting every staged entry (src/Main.java, docs/readme.md), never commit-sel; after a plain click on main.go, Commit & Push writes target, "commit Whole and push", push; Windows Terminal, maximised then 120x40: guide.md and app.go Ctrl+clicked (two "▎" rows, "Commit 2 files"), TODO.md plain-clicked (no "▎" row, "Commit", TODO.md's diff), "Commit the whole index" typed, Ctrl+Enter: one new commit holding the whole index, README.md's staged part, docs/guide.md and internal/state/reduce.go, README.md's unstaged line left out; " M README.md", " M cmd/main.go", " M internal/app/app.go", "?? TODO.md" after, at both sizes
Not verified: a run with an agent CLI live (nothing of the CLIs changed; the terminal runs used the Claude mock); the selection with Codex as the provider in the terminal (Source Control is provider-independent); a second process editing a selected file during a real commit (the check after the commit is tested with hooks that change the commit, not with another process); a moved file in Windows Terminal (the fix is in the git layer, run on real git through the application's model in TestSelectionGit)
Verdict: DONE
```

## Details

**The modifier (50.1).** Windows Terminal encodes Ctrl as 0x10 of the
SGR button (`src/terminal/input/mouseInput.cpp:239`) and sends a click to
the application unless Shift is held (`ControlInteractivity.cpp:651-659`)
or a hyperlink is under the pointer, which Ctrl+click opens
(`:249-264`); the dashboard draws none. In the probe's first run, the
Ctrl+click opened a diff (today's build takes it as a plain click), so a
second run held Ctrl on the list and, at 120x40 where Esc hides the
panel, in the message field (`out-p50a`, `out-p50b`).

**The commit (50.4 to 50.8).** `git commit --only` builds the commit from
HEAD inside git's own transaction, so no commit made meanwhile can be
undone by it, with the owner's hooks, clean-up and signing, and the rest
of the index untouched (probed). It takes each selected file's working
tree content: the files are therefore staged whole (partly staged ones
refused, unstaged and untracked ones staged after the question), so that
what it commits is the staged content. Two alternatives were set aside: a
temporary index then `git commit` (a commit landing between them would be
reverted silently) and `commit-tree` with `update-ref` (safe, but a second
`git commit` to keep alike). A selected file edited on disk during the
commit itself goes in as edited: the check after the commit says so
(`Unexpected`), as it says a hook's change.

**Documents.** The key tables of ui-design.md hold keys, and
`TestKeyOverlayRows` counts their rows (67): Ctrl+click is told in "The
selection", the key overlay's row beside its keys, Esc's row reworded.
The sentence of D8 counts the paths the question staged: with plan 45's
question an untracked folder is one of them, counted as one change.

**Changed:** `internal/input/sourcewrite.go`, `source.go`, `git.go` (a
comment); `internal/app/source.go`, `commit.go`, `mouse.go`;
`internal/render/source.go`, `commit.go`, `dashboard.go`, `keys.go`; the
tests `internal/input/selection_test.go`, `internal/app/selection_test.go`,
`commit_test.go` (the double's two writes), `golden_test.go` (the case),
`internal/render/source_test.go`; `testdata/frames/source-selection-*.txt`
(five); `doc/brief.md` (amendment 51), `doc/ui-design.md`,
`doc/architecture.md`, `AGENTS.md`, `plan.md`, and this file.
