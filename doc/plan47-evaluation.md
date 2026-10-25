# Plan 47 evaluation (2026-10-25)

The check of plan 47, the rest of the audit's phase 1 ([`audit.md`](audit.md),
Part 2, C; brief amendment 48):
- finding 3b: Ctrl+Q during a commit or push quit at once;
- finding 1b: an approval answered as the CLI withdrew it crashed the
  session;
- finding 11: the reducer's Mode fall-through and its provider word;
- from finding 4: the Explorer left Source Control's staging question armed;
- from finding 13: the path literal and two stale comments.

It covers the 10 criteria of [`plan.md`, Plan 47](../plan.md), written
before the code. Each fix was test-first: its test was written and seen
failing on the old code, then the change made it pass. Nothing is
committed: the owner reviews first. No git command touched this
repository; every git test ran on temporary repositories.

How it was checked:
- **Test-first, per fix**, each seen failing on the old code first:
  - `TestCommitQuitAsks` (`internal/app`): Ctrl+Q during a held commit
    asked nothing and quit;
  - `TestAnswerWithdrawn` (`internal/app`): the withdrawn answer crashed
    the session and closed the CLI;
  - `TestReplayNotes`, one in each adapter's package: 38 false notes in 11
    recordings, more than the audit listed;
  - `TestUnreadableWordInMock` (`internal/state`): "deny" in a mock replay
    of Codex;
  - `TestCommitExplorerOverAsk` (`internal/app`): the click on Source
    Control's tile only cancelled the hidden question;
  - `TestRealPathDOS` (`internal/input`): `\\?\C:\…` came back.
- **The documented state machines:** each new rule has its row in
  `events-and-state.md` and its case in `TestStateMachines`.
- **Each fix broken on purpose** through `go test -overlay`, nothing
  written in the repository (`scratchpad\plan47\mutate47.mjs`).
- **Build, vet, gofmt, the whole suite** after the last document edit, the
  race detector on the touched packages, and the timing step alone.
- **Windows Terminal in mock mode** on a temporary repository whose
  pre-commit hook sleeps 8 s (`scratchpad\plan47\mkrepo47.mjs`,
  `scratchpad\term\p46.ps1`, `p47call.ps1`).

```text
Evaluation - Plan 47, the rest of the audit's phase 1 - 10 criteria: 10 PASS
PASS 47.1 TestCommitQuitAsks, through the application, the commit held by the git double: Ctrl+Q leaves the program running, QuitAsked set, the editor's status "quit? the commit is still running · Ctrl+Q again to quit · Esc to stay"; Esc stays, the commit still running; the commit released and a push held: "quit? the push is still running · Ctrl+Q again to quit · Esc to stay"; a second Ctrl+Q quits (the startup Exited, the question cleared). On the old code Ctrl+Q during the commit asked nothing ("asked false")
PASS 47.2 TestStateMachines, "`Ready`, `Crashed` | owner quits while a git operation runs": ready, Quit{GitRunning: true} asks with no command, Stay keeps the session, two Quits close it; crashed, it asks, then exits; Quit{} while crashed exits at once; the existing quit cases ("`Ready` | owner quits", "… while a turn or an agent runs") pass unchanged
PASS 47.3 TestAnswerWithdrawn, through the application on a scripted session: an Answer error wrapping provider.ErrNoPending leaves the approval Expired, the session Ready, the turn Running, no Close, and the status line "the request was withdrawn before the answer reached it"; the withdrawal arriving after ("expired") leaves no "second answer" note; a pipe error still crashes and closes (Crashed, one Close). TestStateMachines: the same with the withdrawal applied first; a withdrawal of an approval already ended ("expired", "resolved") ignored without a note. Claude Code's Answer and AnswerQuestion and Codex's Answer, for an unknown or answered id: errors.Is matches ErrNoPending, the text as before (no pending request "<id>"). On the old code: Crashed, Idle, one Close
PASS 47.4 TestReplayNotes over every recording, the 19 of Claude Code and the 14 of the app server: no note "event without a known payload". On the old code 38 such notes in 11 recordings: Claude Code's p12c switch (2), p13c (1), p303c (2) and p39c (10); Codex's p12x auto review (2), p12x sandbox (3), p12x switch (2), p34x (4), p38t (5), p38x (3) and p39x (4)
PASS 47.5 TestUnreadableWordInMock: in a mock state whose events are Codex's, an unreadable request gets one answer, "decline", the approval Denied with "decline"; with Claude Code's events, "deny". On the old code: "deny" for Codex
PASS 47.6 TestCommitExplorerOverAsk: the staging question open (nothing staged, 4 changes), F3 shows the Explorer and hides Source Control; one click on Source Control's tile shows the panel with no question and nothing staged; then, the question open again at 90x40, Shift+F6 gives zone B the focus and the compact rule hides the panel with its question (none left armed). On the old code the click only cancelled the hidden question and the panel stayed closed
PASS 47.7 TestRealPathDOS: realPath of a temporary folder gives C:\Users\…\001, without \\?\, absolute, a drive letter volume, the same when resolved again; dosPath gives C:\Users\x for \\?\C:\Users\x, \\server\share\dir for \\?\UNC\server\share\dir, and leaves C:\plain. On the old code: \\?\C:\Users\…. Both comments say what the code does
PASS 47.8 Build, vet, gofmt clean; go test ./... ok, 16 packages, 799 tests (792 and the 7 new, 5 of them skipped as before), run again after the last document edit, TestStateMachines included (on the pushed tree a0f45f3 every package passed but internal/state, where only TestStateMachines failed); no file under testdata changed (0 golden frames; the probe recordings and mock files as they were); -race ok on input (225 s), app (337 s), state (16 s) and the five provider packages, 0 data races; the timing step alone, eleven tests within their targets, on mains power with the CPU 16% busy just before: TestMessageToFrame p95 1.09 ms at 120x40 and 1.70 ms at 208x59, the highest p95 5.05 ms (TestLongSession's end), the 50,000-line paste 2.1 ms at worst, 800 of 800 idle motion reports dropped in each case
PASS 47.9 Mutations through go test -overlay: 25 of 25 caught: the quit key ignoring git, the reducer ignoring it, the question naming no git, the push named the commit; the application never marking Gone, Claude Code's Answer and AnswerQuestion and Codex's Answer unwrapped, a gone answer crashing (state and application), no notice, the turn left waiting, the withdrawal first losing the notice, other failures no longer crashing, the late withdrawal noted (state and application); the Mode case falling through (both adapters' replays), the session's provider word; the Explorer, closeSource and the helper keeping the question; the drive prefix kept, a share given one backslash, the old literals
PASS 47.10 Windows Terminal, the final build (scratchpad\term\acc-p47.exe, SHA-256 64fb66dac1f23ab5…, the tree's own), maximised in the acc tab's profile (208x59), then 120x40, mock mode on a temporary repository whose pre-commit hook sleeps 8 s: during the commit ("Committing… 2 s"), Ctrl+Q showed "quit? the commit is still running · Ctrl+Q again to quit · Esc to stay", in the editor's status row at 208x59 and in the bottom row at 120x40, where the panel takes zone B's place; Esc took the question away, the commit going on ("Committing… 4 s"), and it completed ("Committed 0345dad.", "Committed eb5071d."; git log "quit test" on "Initial commit", no index.lock); then, notes/ untracked and nothing staged, Ctrl+Enter asked "Nothing is staged. Stage the change listed under CHANGES and commit it?", F3 showed the Explorer, and one click on Source Control's tile showed the panel with the message kept ("all"), no question, the button on Commit, notes/ still untracked; no terminal-hosted agent session was open; this repository's status, diff, HEAD and origin/main the same before and after; the clipboard unchanged
```

Changed from the design:
- **D2 also silences a false note.** Claude Code's withdrawal arriving
  after the approval had ended (`"expired"`) was noted "second answer
  ignored", so the fixed race would have left that note. Like Codex's
  `"resolved"`, it is now ignored without a note. Criterion 47.3 was
  amended to cover it and the order where the withdrawal comes first.
- **D5's prefixes live in `dosPath`,** so that the UNC case is tested
  without a network share.
- **47.4 found more than the audit:** 11 recordings, not the four it
  named.
- **47.8 was amended** before any code: plan 46 had left
  `TestStateMachines` failing (plan.md, Plan 47, What was found).
- **`TestCommitExplorerOverAsk` also hides the panel through the compact
  rule.** The "closeSource keeps the question" mutant had no test that
  could catch it: no test hid the panel with the question open.

Observed and kept:
- **Where the panel takes zone B's place** (120x40 here), the quit question
  shows in the bottom row, in the same words, since the editor's status row
  is not drawn.
- **A withdrawn approval's card reads "expired, not answered"**, as for
  any request the CLI resolved by itself; the status line says why.
- **The question names the step running now:** "the staging", "the
  commit" or "the push". Stage and commit, quit during its staging, says
  "the staging".

Not verified:
- **A live withdrawal racing an answer:** it needs the owner's key within
  milliseconds of Claude Code's `control_cancel_request` or Codex's
  `serverRequest/resolved`, which cannot be provoked on purpose. It is
  checked through the application with a scripted session, and in the
  reducer in both orders.
- **Quitting for good while git runs, in the real terminal:** the second
  Ctrl+Q is checked through the application. The terminal run quit only
  after the commit had finished.
- **The push's question in the real terminal:** a push needs a remote; it
  is checked through the application (`TestCommitQuitAsks`).
- **A project on a network share:** `dosPath`'s UNC case is unit-tested;
  no share was set up.

Limitations:
- A second Ctrl+Q quits without stopping git, as D1 designed: a commit or
  push left running finishes on its own, or dies with the console and can
  leave `index.lock` behind (probe P45a).
- `ErrNoPending` is the adapters' own check. An answer written before the
  adapter reads the withdrawal reaches the CLI, which handles a stale
  answer in its own way (not probed).

**The commit and the push (2026-10-25), after the owner's approval.** The
owner approved plan 47 on 2026-10-25; commit d44bc0f (34 files: 33
modified, 1 added) and the approval commit were pushed to `origin`;
`origin/main` equals `main`, the working tree is clean, and the author is
`Evengelius <molimoabel@gmail.com>`, with no attribution line: PASS. The
block above stays as it was reported.
