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

## The two items not verified, run with a test double (2026-10-18)

At the owner's request, the first two items of the "Not verified" line were
run in a terminal after the plan was committed (ce95ba2). The summary block
above is left as it was reported. No real agent CLI was started and no
quota was used: `claude.exe` on the launch `PATH` was a test double, a small
Go program kept outside the repository. It answers `-v` with
`2.1.302 (Claude Code)` and the `initialize` control request with success.
It emits a `system/init` event naming its working directory, with the model
`fakeclaude-test-double`, and records that directory and its arguments in a
file beside itself. It exits when its stdin closes and calls no model.

The setup was as for the runs above: a temporary repository `demo` with
`feature` and `clash` worktrees, each one commit ahead, and a temporary
`LOCALAPPDATA`, `USERPROFILE` and `HOME`. `PATH` held the double's
directory, `C:\Windows\System32` and git's `cmd` directory. Two dashboards
ran in two Windows Terminal windows sharing that `LOCALAPPDATA`, so they
shared one session registry:

```text
Two dashboard sessions, test double for claude.exe - 37 checks, maximised (156x44): 37 PASS; 80x24: 37 PASS
PASS The double detected as Claude Code 2.1.302 on the startup screen
PASS Dashboard 1: Ctrl+W, Enter on feature started the session; the CLI's working directory was <base>\demo.worktrees\feature, its arguments -p --input-format stream-json ... --permission-mode default
PASS Dashboard 1: the header names feature and its branch (and its path, maximised); one session file, holding the worktree's path
PASS Dashboard 1: Ctrl+W marks feature "this session"; x refused "this session runs in this worktree"; m refused "this session runs in feature: merge it from another directory"
PASS Dashboard 2, at startup: feature marked "session", main and clash unmarked; x on feature refused "another dashboard session runs in this worktree", directory and branch kept
PASS Dashboard 2: Enter on main started a session whose working directory was <base>\demo; two session files; its view marks main "this session" and feature "session"
PASS Dashboard 1 again: main marked "session"; m on clash refused "another dashboard session runs in main", main's HEAD unchanged, no MERGE_HEAD
PASS Dashboard 1, Ctrl+Q: window closed, the double saw its stdin close, one session file left (main's)
PASS Dashboard 2: feature no longer marked; x on feature now asks "Remove <path>? ..."; n kept the directory and the branch
PASS Dashboard 2, Ctrl+Q: no session file, no dashboard and no double process left; main's HEAD and status unchanged
```

Two runs did not complete cleanly, both through faults of the script, not of
the dashboard:

- The first maximised run stopped after 11 checks. The script read the
  session file with .NET's `ReadAllText`, which does not share write
  access, and the dashboard holds the file open for writing. The read now
  shares read and write access, as the dashboard's own reader does. That
  first run left dashboard 1 open, and Ctrl+Q closed it: no dashboard, no
  double and no session file were left. Its results file was overwritten by
  the repeated run; this paragraph is its record.
- The first 80x24 run gave 36 PASS and 1 FAIL, "dashboard on the worktree".
  The check looked for the worktree's path in the header. At 80 columns the
  header shows `▣ feature   branch feature` without the path, as ui-design.md
  draws the narrow header. The path is now required only maximised, and the
  whole 80x24 run was repeated: 37 PASS.

Still not verified: a real Claude Code or Codex CLI started on a worktree.
That would be a live run under doc/evaluation.md, on the owner's quota and
with the owner's approval. The double shows the dashboard's side, which is
the same Start path, the same arguments and the same registry. A symbolic
link in the project browser (TestDiskListLink SKIPs) is unchanged.

## A real Claude Code started on a worktree, live (2026-10-18)

At the owner's request ("do that live test simulation with my quota"), the
item the double left open was run live, before the evaluation was
committed. The CLI was Claude Code 2.1.302, the version last probed, on
Haiku: `ANTHROPIC_MODEL=haiku` was set in the test window only. Each run
used a new temporary repository `demo` under `%TEMP%`, outside the
repository, with one worktree `feature` one commit ahead. Its two dashboards
shared a temporary `LOCALAPPDATA`, so they shared one session registry. The
real home was kept, so that Claude Code was signed in. The launch `PATH`
began with the logging wrapper of the earlier live runs, which starts the
real `claude.exe` with the same arguments and working directory and records
its stdin and stdout. confcheck ran before and after each run, with the
worktree as the project. Rules fixed beforehand: one prompt per run ("Use
the Write tool to create a file named live.txt in the current directory
containing the single word live. Then reply with only the absolute path of
the current directory."); allow only the first Write card naming live.txt
in the worktree, deny every other card; three minutes at most for the turn.

```text
A real Claude Code (2.1.302, Haiku) started on a worktree - 25 checks, maximised (156x44): 24 PASS, 1 NOT-VERIFIED (configuration); 80x24: 25 PASS
PASS Startup screen: Claude Code 2.1.302 chosen, through the wrapper on PATH
PASS Dashboard 1: Ctrl+W, Enter on feature; "Ready to work on this project." within 60 s; argv -p --input-format stream-json --output-format stream-json --verbose --include-partial-messages --forward-subagent-text --permission-prompt-tool stdio --permission-mode default
PASS Dashboard 1: the header shows "▣ feature   branch feature" (and the path, maximised); one session file, holding the worktree's path; Ctrl+W marks feature "this session"
PASS Dashboard 2, at startup: feature marked "session", main unmarked; x on feature refused "another dashboard session runs in this worktree", directory kept
PASS Dashboard 1, the turn: the Write card named <base>\demo.worktrees\feature\live.txt and was allowed; result success; model claude-haiku-5-5
PASS Claude Code's system/init cwd was <base>\demo.worktrees\feature; live.txt written there and not in <base>\demo
PASS The reply, <base>\demo.worktrees\feature, shown whole in zone B
PASS Dashboard 1, Ctrl+Q: window closed, no Claude Code process left from the run, no session file left
PASS Dashboard 2: feature no longer marked, and now shows +1 (the new live.txt); Ctrl+Q closed it
PASS Nothing left (session files, dashboards, Claude Code processes); main's HEAD and status unchanged
PASS 80x24 / NOT-VERIFIED maximised: configuration; ~\.claude.json: claudeCodeFirstTokenDate and oauthAccount not compared (sensitive name); maximised, ~\.claude\plugins\known_marketplaces.json changed at 17:34:23, during the turn, not classified (contents not inspected)
```

`known_marketplaces.json` is hash-only (reliability.md, mixed state): a
change there makes the run NOT-VERIFIED, never PASS. Its change at 17:34:23
came in the maximised run's turn; plan 1 and plan 4 recorded the same
rewrite as Claude Code's marketplace auto-update. The three maximised runs
before it, with the same CLI and the same prompt, had configuration PASS.
In two runs, 80x24 run 3 and maximised run 5, Haiku first asked to run
`(Get-Location).Path` in PowerShell. That card was denied by the rule above.
Claude Code then wrote live.txt through the Write card, which was allowed.

The block above is the final runs, maximised run 5 and 80x24 run 4. The
earlier runs are kept in this record. Each failure was a fault of the
script, not of the dashboard:

- Maximised run 1, 19 PASS and 6 FAIL. The Write card, naming the worktree's
  live.txt, showed collapsed ("F2 to answer") because the input had the
  focus. The script waited for the expanded keys line and never answered.
  The turn stayed open for the three minutes, so Ctrl+Q asked the quit
  question instead of quitting, and dashboard 2 kept marking feature
  "session", rightly. The script's clean-up then denied the card,
  interrupted the turn and closed dashboard 1. Nothing was left, and
  configuration was PASS. The script now presses F2 when a card is collapsed.
- Maximised run 2, 25 PASS, with the reply check looking only for
  "demo.worktrees" on the screen.
- 80x24 run 1, 24 PASS and 1 FAIL, "reply on screen". At 80 columns zone B
  wraps the reply mid-word ("…demo.worktre" / "es\feature"), and that check
  did not read across the wrap. The check now requires the whole reply,
  read across the wrap.
- Maximised runs 3 and 4, 24 PASS and 1 FAIL, the same check. It read
  whole rows, so zone A's text, beside zone B, came between the wrapped
  halves. Run 4's fix to read zone B's column did not take effect: a
  PowerShell pipeline passed the frame as a single item. Run 4's saved
  frame shows the reply whole, and the check, corrected, passes on it.
  The 80x24 runs 2 and 3, 25 PASS each.

Quota: nine turns on Haiku, API-equivalent $0.0039 to $0.0061 each, $0.037
for the eight that ended. For run 1's turn, which was interrupted, no
result and no cost were reported, so its cost is N/A. Zone D showed the
5-hour window at 3% throughout.

Now verified: a real Claude Code started on a worktree, and another
dashboard's "session" mark for it, at both sizes. Codex CLI was not started
on a worktree: it uses the same Start path and registry, and was not in the
approved scope. A symbolic link in the project browser (TestDiskListLink
SKIPs) is unchanged.
