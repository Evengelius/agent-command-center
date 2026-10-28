# Plan 64 evaluation (2026-10-28)

The check of plan 64, Windows notifications for the ping (brief amendment
66): with each ring of plan 63's ping, a silent Windows notification of
the most urgent need rung, under the name Agent Command Center, its text
generic; one at a time, taken out of Windows' list once nothing waits and
when the dashboard quits. Windows takes a desktop program's notifications
under the app ID of a Start menu shortcut, which the dashboard makes at
its start when missing, pointing at Windows Terminal's own alias with
ACC's profile. A probe first decided that Windows takes ACC's
notifications cleanly; the owner then said "Go".

It covers the 7 criteria of [`plan.md`, Plan 64](../plan.md), written
before the code, with another agent's two reviews and the owner's
clarifications folded in. Nothing is committed or pushed: the owner
reviews first.

How it was checked:
- **The probe** (`scratchpad\plan64\probe`, x/sys only, outside the
  repository; `scratchpad\term\p64probe.ps1`): five runs in Windows
  Terminal under a test shortcut and app ID, removed after each.
- **Input tests** (`internal/input/notify_test.go`,
  `notify_windows_test.go`): the content; the shortcut in temporary
  folders, never the Start menu. No test shows a notification.
- **Application tests** (`internal/app/notify_test.go`, 7 tests) on a fake
  notifier recording every call in order and the one notification up;
  `internal/render/home_test.go`'s `TestHomeNotifyNote`.
- **Each change broken on purpose** through `go test -overlay`
  (`scratchpad\plan51\mutrun`; the list built by
  `scratchpad\plan63\mkmutants64.mjs`), the repository never written.
- **Windows Terminal**: the real application built from this tree, in
  ACC's tab and in Command Prompt's profile, the mock and a copy cut after
  its first approval, the user's own home as `acc` runs it, Bubble Tea's
  trace on; Windows' list counted by the probe's `-count` from a Terminal
  window of its own (`scratchpad\term\p64run.ps1`).
- **The race detector** on the two packages the plan changed, **the timing
  step** alone, and **build, vet, gofmt, the whole suite** after the last
  code and document edit.

## The evaluation

```text
Evaluation - Plan 64, Windows notifications for the ping - 7 criteria: 6 PASS, 0 FAIL
PASS 64.1 The probe: in Windows Terminal 1.23 on this machine, five runs under the test shortcut "ACC notification probe" (app ID Evengelius.AgentCommandCenter.Probe, target %LOCALAPPDATA%\Microsoft\WindowsApps\wt.exe, arguments -p "Agent Command Center", read back each time): the banner "ACC notification probe", "Approval required" over "Claude Code is waiting for your approval.", from the built program and from go run (its program in Go's temporary build folder), the content silent, x/sys its only module, no notification policy set; Windows' list 1, then 0 after RemoveGroupedTagWithId; the shortcut gone after each run, the Start menu's 47 entries the same before and after; a shortcut made 2 s before a notification gave no banner, 15 s gave one; the click not recorded, the banner having no window of its own (not a condition); the owner's go, "Go"
PASS 64.2 When: TestNotifyWhen, in ACC's profile and outside it: none while the window has the focus, none for streamed text, a tool or an interrupted turn, one notification at a turn's end (its BEL in ACC's profile only), a question 2 s later notified when the 5 s are up (a 3 s timer), with its BEL in ACC's profile only; TestNotifyMostUrgentRung: a question rung while an earlier approval waits notified as the question; in Windows Terminal, ACC's tab and Command Prompt's profile alike: a notification at the turn's end and at the approval's ring 5 s after it, the BELs in ACC's tab only (2 and 0)
PASS 64.3 The text: TestNotifyText, Claude Code and OpenAI Codex: "Approval required" over "<CLI> is waiting for your approval.", "Question waiting" over "<CLI> has a question for you.", "Turn completed" over "<CLI> has finished its task.", "Turn failed" over "<CLI>'s turn ended with an error.", "Session ended" over "<CLI> stopped unexpectedly.", and in no call the command, the question or the project; a mock "The mock session has finished its task."; TestToastXML: the title over the text, escaped, <audio silent="true"/>, nothing else; one tag and group for every call (internal/input/notify.go); in Windows Terminal the banners "Turn completed" over "The mock session has finished its task." and "Approval required" over "The mock session is waiting for your approval."
PASS 64.4 Replace and remove: TestNotifyClear: two approvals, two notifications, one up; one answered, nothing taken out; both answered, taken out; a turn's end then the focus back, taken out; the dashboard quitting with one up, taken out; nothing up, no call; TestNotifyFailure: a refusal said in the toast, the BEL rung all the same; in Windows Terminal, Windows' list 1 after the turn's notification and still 1 once the approval's replaced it, 0 once answered (ACC's tab, 208x59 and 120x40), 0 after the focus came back from a turn's end, 0 after the dashboard quit with the approval waiting (Command Prompt's profile)
PASS 64.5 The identity: TestNotifyShortcut, in temporary folders: made when missing, its target the Terminal alias, its arguments -p "Agent Command Center", its app ID Evengelius.AgentCommandCenter; not rewritten once there (its time unchanged); one of that name with another app ID left byte for byte, no notifier, the note naming it and its app ID; no Terminal alias, no shortcut and a note; TestHomeNotifyNote: the note a red line of the startup screen; TestNotifySetUp: the set-up in a command at the start, its calls by message, its note; in Windows Terminal the dashboard's first start made the real one at 21:46:05 (read back: its target, arguments and description), the Start menu's other 47 entries unchanged, Terminal's settings.json and ACC's fragment with plan 63's times and hashes; removed as ui-design.md says, by deleting it (gone at 22:16:58), made again by the next start (22:17:01)
PASS 64.7 The checks: go build ./... ok; go vet ./... ok (Windows and Linux); gofmt -l . empty; go test ./... ok (16 packages) after the last code and document edit; no golden frame or mock file changed; 20 mutants through -overlay, each caught by a failing test (details); go test -race on internal/app (425.9 s) and internal/input (233.7 s): no race; the timing step, run alone on mains power (22:36 to 22:37), all nineteen tests within their targets: TestBannerToFrame p95 1.62 ms (target 2 ms), TestHomePictureTime worst 191.9 ms (target 260 ms), TestStreamToFrame p95 4.00 ms, the others' p95 2.98 ms at most (target 16 ms), TestPasteToFrame max 1.23 ms (target 100 ms)
Not verified: 64.6's last parts, the owner's look at the shots and the sound, which the harness cannot hear (the content's <audio silent="true"/>). Its measured part holds: in ACC's tab at 208x59 and 120x40 and in Command Prompt's profile at 208x59, the focus on the desktop's window, the banners under "Agent Command Center", "Turn completed" over "The mock session has finished its task." and then "Approval required" over "The mock session is waiting for your approval.", the newer replacing the older (1 in Windows' list after each); gone from the list after the focus came back from a turn's end, after the answer, and after the dashboard quit with the approval waiting; in Command Prompt's profile the same notifications and no BEL; review shots out-p64run\banner-A-turn.png, banner-A-approval.png, banner-B-approval.png
Verdict: NOT DONE
Proposed next step: the owner's look at the shots and a listen to a notification - waiting for approval
```

**64.6, the owner's review (2026-10-28).** The owner looked at the review
shots, the banners of a turn's end and of an approval in ACC's tab and of
the approval in Command Prompt's profile, and wrote "Looks right": 64.6's
look, PASS. Its sound, which the harness cannot hear and the content
mutes, is still the owner's to confirm. Not committed. The block above
stays as it was reported.

Asked whether the notifications played any sound, the owner wrote "No
sound": 64.6's last part, PASS, so 7 criteria, 7 PASS, 0 FAIL. Not
committed: the commit waits for the owner's approval. The block above
stays as it was reported.

**Approved and committed (2026-10-28).** The owner wrote "Approve and
commit". Plan 63's push record went first, alone, as d057bd9 (2 files).
Commit 0111dbd (20 files: 12 modified, 8 added), from the files reviewed
in `git status` and `git diff`: `git diff --check` found no whitespace
error and no file holds a CR or a BOM; the whole suite had passed after
the last code edit and after the last edit of a document a test reads.
The author is `Evengelius <molimoabel@gmail.com>`, with no attribution
line, and the working tree was clean after it: PASS. Not pushed: the
owner asked for the commit only. The block above stays as it was
reported.

**Pushed (2026-10-28).** The owner wrote "Push them": commits d057bd9,
0111dbd and 2ecd5ff fast-forwarded origin's main from 9cdf1ae to
2ecd5ff, nothing forced, origin/main read back equal to main. The commit
record above, which says "Not pushed" as written, stays as it was.

## Details

### What the build found and changed

- **Who makes the shortcut.** The probe's notes said `acc` would make it
  before it starts the dashboard. The dashboard makes it instead, at its
  start, in a command (design D5 allowed either): one implementation in
  Go, the probe's, and no COM code in `acc.ps1`. The startup screen gives
  Windows its seconds before any ping; only a very first run can miss a
  banner in its first seconds.
- **Off the first frame's path.** The set-up's first call in a process
  took 1.01 s (the shell's libraries loading), the next ones about 42 ms:
  it runs in a command at the start (`Options.NotifySetUp`,
  `notifySetUpMsg`), never before the first frame.
- **Known folders follow the environment.** Windows reads the user's Start
  menu and local app data with the process's APPDATA and LOCALAPPDATA:
  under the terminal harness's temporary home the dashboard looked for
  Terminal's alias in the temporary folder, found none and correctly made
  no shortcut. The Windows Terminal check ran with the user's own home, as
  `acc` runs; a comment that said the opposite was corrected.
- **The most urgent need rung.** No test held design D3's "most urgent
  need rung" against "the most urgent need waiting" until the mutants were
  written: `TestNotifyMostUrgentRung` was added, and caught N3.

### The probe

`scratchpad\plan64\probe` (a module of its own, x/sys v0.49.0 its only
dependency, outside the repository), run by `scratchpad\term\
p64probe.ps1` in a Windows Terminal window, five runs, each with the test
shortcut "ACC notification probe", app ID
`Evengelius.AgentCommandCenter.Probe`, made, read back and removed.

| Run | The notification | Windows' list | After |
| - | - | - | - |
| built, 2 s after the shortcut | no banner, not in the notification centre | 1, then 0 | the shortcut gone; no record of the app ID |
| built, 2 s, the centre opened | the same | 1, then 0 | the same |
| built, 15 s | the banner "ACC notification probe" | 1, then 0 | the shortcut gone; Windows' record of the app ID |
| go run, 15 s | the same banner | 1, then 0 | the same |
| built, 15 s, the click | the banner; no window of its own to click | 1, then 0 | the same |

The Start menu held its 47 entries before and after every run. The click
was not made: the banner is drawn by Explorer in a full-screen layer
reported hidden and cloaked, and a click by its place could land on
another application's window under it. Windows' own record of the test
app ID, `HKCU\Software\Microsoft\Windows\CurrentVersion\Notifications\
Settings\Evengelius.AgentCommandCenter.Probe`, stays until the owner asks
for it to be removed.

### The Windows Terminal runs

`scratchpad\term\p64run.ps1`, from plan 63's `p63.ps1`: the application
built from this tree (`acc-p64.exe`), the user's own home and app data as
`acc` runs it (`-realHome`), a PATH of System32 only, Bubble Tea's trace
on; Windows' list counted by the probe's `-count` for
`Evengelius.AgentCommandCenter`, in a Terminal window of its own, outside
this shell's app container. A mock dashboard, so no CLI was started and no
quota used. Windows Terminal 1.23.20211.0.

| Run | Profile | Result |
| - | - | - |
| the first start | ACC's | `Agent Command Center.lnk` made at 21:46:05: target `%LOCALAPPDATA%\Microsoft\WindowsApps\wt.exe`, arguments `-p "Agent Command Center"`, description "Agent Command Center: its Windows Terminal profile"; the Start menu's other 47 entries unchanged |
| A, 208x59, the mock cut after its first approval | ACC's | the banner "Agent Command Center", "Turn completed" over "The mock session has finished its task."; 1 in Windows' list; 5 s later "Approval required" over "The mock session is waiting for your approval.", still 1; 2 BELs (21:47:00, 21:47:05); the title "◆ Approval waiting", kept with the focus back; F2, a: the title plain, 0 in the list |
| A, 120x40 | ACC's | the same: 1, 1, 2 BELs, the titles, 0 |
| B, 208x59, the cut mock | Command Prompt's | the same banners and counts, 0 BELs; the dashboard quit with the approval waiting: 0 in the list after |
| C, 208x59, the mock | ACC's | the turn's banner, 1 in the list, 1 BEL; the focus back: the title plain, 0 in the list |
| removal | - | the shortcut deleted from a Terminal window (gone at 22:16:58), made again by the dashboard's next start (22:17:01) |

Terminal's settings.json and ACC's fragment kept their times and hashes
of plan 63's run (18:41:40, sha256 7834893B… and A0999EBA…). The owner's
terminal-hosted CLI sessions: none were open; each run's windows closed.

### The mutants

Each change broken on purpose, one at a time, through `go test -overlay`
(mutrun; the list from `scratchpad\plan63\mkmutants64.mjs`), with `-run
'TestNotify|TestPing'` on `internal/app` and `-run
'TestToastXML|TestNotifyShortcut'` on `internal/input`, the repository
never written (its `git status` the same before and after). All 20 caught
by a failing test:

| Mutant | Broken | Caught by |
| - | - | - |
| N1 | no notifier ever used | TestNotifyWhen: no call at a turn's end |
| N2 | no notification outside ACC's profile | TestNotifyWhen, profile "": no call |
| N3 | the most urgent need waiting, not of those rung | TestNotifyMostUrgentRung: the approval notified again |
| N4 | the ring without its notification | TestNotifyWhen: no call |
| N5 | never taken out once nothing waits | TestNotifyClear: no clear when both approvals were answered |
| N6 | a notification not marked up | TestNotifyClear: the same |
| N7 | a clear not marked done | TestNotifyClear never settles: each message queues one more clear, its answer one more message; the test ends on Go's timeout |
| N8 | no clear when the dashboard quits | TestNotifyClear: no clear at Shutdown |
| N9 | a mock named by its provider's name | TestNotifyText: "Mock data has finished its task." |
| N10 | a failed turn said to have finished | TestNotifyText: the failed turn's text |
| N11 | a refused notification not said | TestNotifyFailure: no toast |
| N12 | the set-up's calls not kept | TestNotifySetUp: no calls once ready |
| N13 | no set-up at the start | TestNotifySetUp: no set-up |
| I1 | the sound not muted | TestToastXML: the content |
| I2 | an ampersand not escaped | TestToastXML: "a & b" |
| I3 | another program's shortcut taken for ACC's | TestNotifyShortcut: a notifier and no note |
| I4 | a shortcut made with no Terminal | TestNotifyShortcut: a shortcut made |
| I5 | the shortcut without ACC's profile | TestNotifyShortcut: its arguments "" |
| I6 | the shortcut without its app ID | TestNotifyShortcut: its app ID "" |
| I7 | an existing shortcut not looked at | TestNotifyShortcut: the other program's taken |

`Notifier.Show` and `Clear` have no mutant: no test shows a real
notification, and the Windows Terminal runs hold them (64.4, 64.6).

### The race detector

`go test -race` on the two packages the plan changed, with the portable
WinLibs GCC 16.2.0 first on the run's own PATH and `CGO_ENABLED=1`
(evaluation.md, The race detector): internal/app 425.9 s, internal/input
233.7 s, no race reported, every test passing. The notification calls run
on one goroutine of their own, fed in Update's order (`notifyCall`).

### The timing step

Run alone with the exact command of `doc/evaluation.md`, Keystroke timing,
from 22:36:00 to 22:37:19: mains power, the battery at 99 %, the CPU at
60, 0, 59, 0 and 27 % over 2 s just before, the VPN client's system scan
idle; open: Chrome, Edge, Teams, Outlook, Claude, Calculator, Notepad,
Windows Terminal, Move Mouse and Settings. No other `go` build or test
ran.

| Test | Result | Target |
| - | - | - |
| TestGraphFrameCost | p95 1.00 ms (p50 under the clock's tick) | p95 ≤ 16 ms |
| TestBlameFrameCost | p95 1.44 ms | p95 ≤ 16 ms |
| TestBranchesFrameCost | p50 0.97 ms, p95 2.02 ms | p95 ≤ 16 ms |
| TestKeystrokeToFrame | p95 1.06 ms; the draft window 1.05 ms | p95 ≤ 16 ms |
| TestDraftKeystrokes | p95 2.42 ms | p95 ≤ 16 ms |
| TestLongSession | p95 2.98 ms | p95 ≤ 16 ms |
| TestThinkingFrame | p95 2.10 ms | p95 ≤ 16 ms |
| TestPasteToFrame | max 1.23 ms | ≤ 100 ms |
| TestStreamToFrame | p95 4.00 ms | p95 ≤ 16 ms |
| TestSelectorToFrame | p95 1.01 to 1.53 ms, with TestMotionToFrame | p95 ≤ 16 ms |
| TestMotionToFrame | as above | p95 ≤ 16 ms |
| TestDiffToFrame | p95 1.02 to 1.06 ms | p95 ≤ 16 ms |
| TestMessageToFrame | p95 1.04 and 2.01 ms | p95 ≤ 16 ms |
| TestOutputToFrame | p95 1.02 ms | p95 ≤ 16 ms |
| TestToolsToFrame | p95 2.43 ms | p95 ≤ 16 ms |
| TestBannerToFrame | p95 1.62 ms, max 3.29 ms, 11,355 bytes a frame | p95 ≤ 2 ms |
| TestHomeKeysToFrame | p95 0.15 ms | p95 ≤ 16 ms |
| TestHomePictureTime | p50 181.9 ms, max 191.9 ms | ≤ 260 ms |
| TestKeystrokeCost | 220 µs, 448,629 bytes and 597 allocations a key (10,000 lines); 633 µs, 1,453,189 bytes and 597 allocations (50,000 lines) | reports only |
