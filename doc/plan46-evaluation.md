# Plan 46 evaluation (2026-10-25)

The check of plan 46, three fixes from the audit ([`audit.md`](audit.md),
Part 2; brief amendment 47):
- finding 3a: the worktree merge that could abort a merge it did not
  start;
- finding 1a: Codex's handshake ended by a refused optional read;
- finding 10: an opened untracked folder in Source Control listed once and
  never again.

It covers the 10 criteria of [`plan.md`, Plan 46](../plan.md), written
before the code. Each fix was test-first: its test was written and seen
failing on the old code, then the change made it pass. Nothing is
committed: the owner reviews first. No git command touched this
repository; every git test ran on temporary repositories.

How it was checked:
- **Test-first, per fix:**
  - `TestGitMergeRefusals` (`internal/input`), on temporary repositories
    with real git;
  - `TestOptionalReadsKeepHandshake` (`internal/provider/codex`), the real
    mapper feeding the real reducer;
  - `TestSourceFoldersFollow` (`internal/app`), on the application with a
    real folder on disk.

  Each one failed on the old code first, with the failure the audit
  described.
- `TestSourceFolderReadsOrdered` was written with the fix, since it needs
  the numbered reads. Its mutant shows that it fails on the old behaviour.
- **Each fix broken on purpose** through `go test -overlay`, nothing
  written in the repository (`scratchpad\plan46\mutate46.mjs`). The tree's
  diff hash was compared before and after.
- **Build, vet, gofmt, the whole suite,** the race detector on the touched
  packages, and the timing step alone. Because the timing figures had
  risen since the morning, the committed tree was timed in turn with this
  one.
- **Windows Terminal in mock mode** on temporary repositories
  (`scratchpad\plan46\mkrepo46.mjs`, `scratchpad\term\p46.ps1` with a run
  step, `p46call.ps1`).

```text
Evaluation - Plan 46, three fixes from the audit - 10 criteria: 10 PASS
PASS 46.1 TestGitMergeRefusals: the main worktree with the owner's merge in progress (a conflict with topic) and its conflict file edited ("resolving"); Merge returns "the main worktree has a merge in progress: finish or abort it first"; the trace holds no "merge" call; MERGE_HEAD still topic's commit and the file still "resolving". On the old code: "conflict: merge aborted, main unchanged", git merge and merge --abort in the trace, MERGE_HEAD gone, the file back to "ours"
PASS 46.2 TestGitMergeRefusals: one modified file gives "the main worktree has 1 uncommitted change: commit them first", one modified and one untracked "the main worktree has 2 uncommitted changes: commit them first"; no "merge" in the trace, the files and HEAD unchanged. On the old code the merge ran and main moved
PASS 46.3 TestGitMerge: a clean merge merges; this call's conflict gives "conflict: merge aborted, main unchanged", MERGE_HEAD gone, HEAD and status as before; TestGitMergeRefusals: a merge started between the check and the call (the trace runs git merge topic just before the call's merge) is refused by git and not aborted, its MERGE_HEAD still topic's; TestGitMissing: ErrNoGit
PASS 46.4 TestOptionalReadsKeepHandshake through the real Codex mapper and the reducer: a refused config/read, then a refused model/list, answered before thread/start's answer: phase Handshaking, no close command, each kept as an error ("config/read refused by Codex: unavailable", "model/list refused by Codex: unavailable"), the selector's reason for the list set (ListProblem); thread/start's answer: Ready. On the old code: Crashed, "handshake failed: config/read refused by Codex: unavailable", the CLI closed
PASS 46.5 the same test: a refused initialize, thread/start and thread/resume during the handshake: Crashed, "handshake failed: <method> refused by Codex…", the CLI closed; once ready, a refused config/read and model/list: Ready, no close; the state and provider suites pass with the probe replays and mock files unchanged (no -update)
PASS 46.6 TestSourceFoldersFollow: Source Control shown, web/ opened (web/app.ts listed); web/secret.env written; after the next status read the panel lists web/secret.env and web/app.ts; folded, web/two.txt written, opened again: web/two.txt listed without a status read. On the old code neither appeared
PASS 46.7 TestSourceFolderReadsOrdered: two reads of web/ answered newer first then older: the newer listing kept; a read for the folder once folded dropped; a read for a folder no longer untracked dropped (its listing gone)
PASS 46.8 Build, vet, gofmt clean; go test ./... ok, 16 packages, 792 tests (788 and the 4 new); no file under testdata changed (0 golden frames); -race ok on input (232 s), app (534 s), state (20 s) and the five provider packages, 0 data races; the timing step alone, eleven tests within their targets, on mains power with the CPU 14% busy just before: TestMessageToFrame p95 1.08 ms at 120x40 and 3.88 ms at 208x59, the highest p95 12.2 ms (TestLongSession's end), the 50,000-line paste 4.1 ms at worst, 800 of 800 idle motion reports dropped in each case; the committed tree timed in turn with this one gave the same range (TestLongSession end 12.4 and 8.7 ms against 12.8 and 11.1 ms; streaming 8.7 and 9.0 against 8.7 and 10.3; the thinking card 8.3 and 10.1 against 8.9 and 8.7), so the rise from the morning's run (5.0, 4.8 and 4.1 ms) is the machine's, not the code's
PASS 46.9 Mutations through go test -overlay: 12 of 12 caught: the pre-check skipped, a merge in progress not refused, uncommitted changes not refused, any MERGE_HEAD aborted, no abort at all; model/list not marked optional, config/read not marked optional, the reducer ignoring Optional, every refusal optional; no reads after a status read, no read on opening, the sequence not checked; the tree's diff hash the same before and after the run (e0b98ddc1944d680)
PASS 46.10 Windows Terminal, the final build (scratchpad\term\acc-p46.exe, SHA-256 b1fe9d8ec1fac5e7…), maximised in the acc tab's profile (208x59), then 120x40, mock mode on a temporary repository: Source Control's notes/ opened lists todo.md; new-idea.md, created outside, listed 2.4 s later at 208x59 and 2.2 s at 120x40, within one poll; then notes/ committed, the Worktrees view opened (main clean), a merge of topic started outside in the main worktree and a.txt edited ("owner resolution"), feature selected, m, y: "✗ the main worktree has a merge in progress: finish or abort it first" in the view's note; afterwards MERGE_HEAD still topic's commit, a.txt "owner resolution", feature not merged, main's log unchanged; no terminal-hosted agent session was open; this repository's status, diff, HEAD and origin/main the same before and after; the clipboard unchanged
```

Changed from the design:
- **The abort is narrower than D1 said.** D1 aborted only when MERGE_HEAD
  existed after the merge and not before. The code goes further: it aborts
  only when MERGE_HEAD is the merged branch's tip, so even a merge started
  in the instant between the pre-check and the merge is left alone. The
  trace hook makes that instant testable, and criterion 46.3 was extended
  to cover it.
- **`folderMsg`'s unused generation is replaced by a per-folder
  sequence;** the folder read on opening is now unconditional. The last
  listing stays on screen until the new one arrives.

Observed and kept:
- **After a refused merge, the Worktrees view reads its list again**, as
  after any operation, so main then shows the change its merge in progress
  holds ("+1").
- A merge in progress is checked before uncommitted changes, so its message
  is the more specific one, even though its conflict files are also listed
  as changes.

Not verified:
- **A live Codex session refusing `model/list` or `config/read`:** none of
  the recorded sessions shows one, and none can be provoked without a fault
  in Codex. The path is checked through the real mapper and the reducer.
- **The merge race in the real terminal:** a merge started in the instant
  between the pre-check and the merge is covered deterministically by the
  trace-hook test.

Limitations:
- An opened folder is read again at each status read: every 3 s, longer
  after slow reads. A file added within that interval and staged by "Stage
  and commit" before the next read is still not listed. "Stage and commit"
  stages a folder whole by design.
- A merge that git refuses for a reason the pre-check does not cover (for
  example local changes the merge would overwrite, appearing after the
  check) is reported in git's words, not the dashboard's.
