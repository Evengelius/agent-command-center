# Plan 16 evaluation (2026-10-18)

The final check of plan 16, git worktree management (amendment 10). It checks
the plan's four items in
[`plan.md`, Plan 16](../plan.md#plan-16---git-worktree-management) against the
yes/no criteria written there before each item's work started. It also checks
the plan's rules: git runs through `os/exec`, never with `--force`, `-f` or
`-D`; the agent CLIs and their configuration are untouched; there is no new
dependency; no agent CLI is started.

The evidence is `plan.md`'s record of each item, the tests, the golden frames
and the two terminal runs. The block below is the summary reported to the
owner. The results are not to be rewritten.

```text
Evaluation - Plan 16, git worktree management - 26 criteria: 26 PASS
PASS 1.1 The list: main first, branch or detached, uncommitted changes and commits ahead counted, Windows paths; outside a repository "Not a git repository" and the directory (TestGitWorktrees)
PASS 1.2 Create: <repo>.worktrees\<name> (TestWorktreePath); a new branch from main's HEAD, an existing branch checked out; an invalid name, an existing path and a branch checked out elsewhere refused with the reason, nothing created (TestGitAdd)
PASS 1.3 Merge: a branch ahead merges; a conflict is aborted, main's HEAD unchanged, no MERGE_HEAD, status clean, the error says "conflict" (TestGitMerge)
PASS 1.4 Remove: a clean worktree goes and its merged branch is deleted; an unmerged branch is kept with git's reason; a dirty worktree is refused by git and left whole; no recorded call passes --force, -f or -D (TestGitRemove)
PASS 1.5 Diff: committed and uncommitted changes against the merge base, untracked files named, capped with "… N lines left out" (TestGitDiff)
PASS 1.6 No git on the PATH: every call fails with "git not found" (TestGitMissing)
PASS 1.7 The registry: a registered session is live for a second reader; once released its file is stale and deleted at the next read (TestSessionRegistry)
PASS 2.1 Ctrl+W opens the view for the field's directory or the current one; "Not a git repository"; "git not found"; Esc closes with the field unchanged (TestWorktreesStartup)
PASS 2.2 Enter on a row fills the project field with its path and starts (TestWorktreesStartup); live: the field held ...\demo.worktrees\clash and validation said no agent CLI, at both sizes
PASS 2.3 Create: n shows the branch field and the path; Enter creates and selects the new row; a refusal shows the reason, list unchanged; Esc cancels (TestWorktreeCreate)
PASS 2.4 Golden frames worktrees and worktrees-new at 79x24, 80x24, 120x40, 200x60; the 18 existing startup frames change in the footer line only (git diff --stat: 18 files, 18 lines each way); every footer within the width (TestFootersFit)
PASS 3.1 Ctrl+W on the dashboard opens the view for the session's project; "this session" and "session" rows; Esc returns; an event arriving meanwhile is in the state after Esc (TestWorktreesSession)
PASS 3.2 d shows the diff coloured, scrolls, shows the cap line; Esc returns to the list (TestWorktreeDiff)
PASS 3.3 m asks y/n; y merges, says so, reads the list again; n cancels; each refusal of the design gives its reason and calls no git merge; a conflict says "conflict: merge aborted" (TestWorktreeMerge)
PASS 3.4 x asks y/n; y removes and says what became of the branch; each refusal gives its reason and calls no git removal (TestWorktreeRemove)
PASS 3.5 A slow git: "reading…" or the operation shown, keys handled, a stale result dropped (TestWorktreesSlow)
PASS 3.6 A session started from the startup screen registers its project and releases it at the end; a mock run or no registry registers nothing (TestSessionRegistered); live: no session file left after either mock run
PASS 3.7 Golden frames worktrees-session, worktrees-diff, worktrees-confirm at the four sizes; the key overlay lists Ctrl+W (TestKeysListWorktrees; TestKeyOverlayRows reads 29 rows from ui-design.md, was 28)
PASS 4.1 Amendment 10 in doc/brief.md; worktrees left [Nice to have] in plan.md and AGENTS.md; architecture.md (ASM-25, coverage), ui-design.md (Worktrees, startup Keys and footers, Anywhere), reliability.md and doc/evaluation.md's MVP-scope check say what is built
PASS 4.2 go build ./... and go vet ./... (also -tags live, -tags timing) exit 0; gofmt -l . prints nothing (after confcheck.go was restored, see below)
PASS 4.3 go test ./... 306 top-level tests PASS and 1 SKIP in 14 packages, 0 FAIL (baseline b047434 288, +18: 8 for item 1, 3 for item 2, 7 for item 3)
PASS 4.4 The timing step run alone: TestKeystrokeToFrame p95 1.07 ms (max 2.49 ms), TestDraftKeystrokes p95 3.05 ms (max 5.96 ms), TestKeystrokeCost 666 µs per key at 10k lines and 1.17 ms at 50k, TestLongSession delta-to-frame p95 7.08 ms (26209 events, heap 7.1 MiB), all PASS (target p95 <= 16 ms)
PASS 4.5 Terminal run, Windows Terminal maximised (156x44): 34 checks PASS, 0 FAIL; a temporary repository with three linked worktrees, a temporary store and home, PATH holding System32 and git's cmd directory only: no agent CLI found or started; Ctrl+Q closed both windows, no process left
PASS 4.6 Terminal run, Windows Terminal at 80x24: the same 34 checks PASS, 0 FAIL (second run; the first gave 33 PASS, 1 FAIL, a fault of the script's check, see below)
PASS 4.7 The 65 changed and new files (this one included; confcheck.go no longer among them) carry no CR, no BOM and no NUL
PASS 4.8 This summary written here and in plan.md's progress list
Not verified: the "session" mark for another live dashboard session was not run in a terminal (it needs a real agent CLI session); TestWorktreesSession and TestSessionRegistered cover it, on Windows file handles. A real agent CLI started on a worktree was not run live, by design (no CLI started); it is the startup screen's existing Start path. A symbolic link in the project browser (TestDiskListLink SKIPs, as in plan 15).
Verdict: DONE
```

## The working copy's confcheck.go

`internal/confcheck/confcheck.go` carries an uncommitted change that is not
part of plan 16. It adds 25 lines that declare four `Entry` fields, the
`Read` field and the `keyLevel` method a second time. With it, the plain
`go build ./...` fails ("Parsed redeclared", "Keys redeclared"), and so do
`go vet` and `go test` for every package that imports it.

At first the file was left as it was, since uncommitted work is preserved.
The checks in 4.2 to 4.4 ran with Go's `-overlay`, which substitutes the HEAD
version of that one file (`git show HEAD:internal/confcheck/confcheck.go`)
for the build and changes no file on disk:

```text
go build -overlay overlay.json ./...
go vet -overlay overlay.json ./...            (also -tags live, -tags timing)
go test -overlay overlay.json -count=1 -v ./...
go test -overlay overlay.json -tags timing ./internal/app -run '^(TestKeystrokeToFrame|TestDraftKeystrokes|TestKeystrokeCost|TestLongSession)$' -count=1 -v
```

Every line the change added was already in the HEAD version; the only
difference was the wording of `keyLevel`'s comment. With the owner's
approval (2026-10-18), the file was restored to HEAD, and the plain commands
ran again on the working copy. `go build ./...`, `go vet ./...` (also `-tags
live` and `-tags timing`) exited 0, and `gofmt -l .` printed nothing. `go
test -count=1 -v ./...` gave 306 PASS, 1 SKIP and 0 FAIL in 14 packages,
the same as with the overlay. The timing step was not run again: it runs in
`internal/app`, which does not depend on `confcheck`.

## The terminal runs

Both runs used the same script. It sent real keys to a Windows Terminal
window and read the screen through UI Automation. The repository `demo` had
main and three linked worktrees in `demo.worktrees`:

- `feature`: one commit adding a file with a tab;
- `dirty`: one uncommitted change;
- `clash`: one commit that conflicts with main.

Main moved on after them. `LOCALAPPDATA`, `USERPROFILE` and `HOME` pointed at
temporary folders, so the owner's store and session registry were neither
read nor written. `PATH` held `C:\Windows\System32` and git's `cmd` directory
only, so no agent CLI was found or started. Run A used `--project <repo>`;
run B used `--mock`, with the mock file in the repository, excluded through
`.git/info/exclude`.

The 34 checks, each against git's own view of the repository as well as the
screen:

- the setup and the size;
- the startup footer's "Ctrl+W worktrees";
- the view, its repository and four rows (`clean ↑1`, `+1 ↑0`), and the
  footer;
- `n` and "release-1": the form, the note, the directory, git's list, and the
  new row selected;
- `d` on feature: the diff with its tab line (`+    return`), then Esc;
- `m` on dirty, refused;
- `m` and `y` on feature: main's HEAD moved, the commit in main's log, main
  clean, and the row at `↑0`;
- `x` on dirty, refused;
- `x` and `y` on release-1: the directory gone and the branch deleted;
- Esc to the startup screen;
- Enter on clash: the field holds it, and validation refuses (no agent CLI);
- Ctrl+Q;
- in the mock session:
  - the view and its footer;
  - `n` refused;
  - `m` and `y` on clash: "conflict: merge aborted, main unchanged", HEAD
    unchanged, no `MERGE_HEAD`, main clean;
  - `x` and `y` on clash: "branch kept:", the directory gone, the branch
    still there;
  - Esc to the dashboard;
  - Ctrl+Q;
- no session file left.

The first run at 80x24 failed one check, "A Enter start here". The screen
was right: the field showed `…s\amolimom\AppData\Local\Temp\p16-small\demo.worktrees\clash`,
cut on the left to fit the field, and the validation message was there. The
script looked for the whole path, or the path with the user folder shortened
to `C:\Users\…`. The check was changed to look for the end of the path
(`p16-small\demo.worktrees\clash`), and the whole 80x24 run was repeated:
34 PASS.
