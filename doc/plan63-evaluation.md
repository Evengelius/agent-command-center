# Plan 63 evaluation (2026-10-28)

The check of plan 63, the ping (brief amendment 65): when the agent needs
the owner while the dashboard's window, or its tab, lacks the focus, one
BEL in ACC's own Windows Terminal profile, whose `bellStyle` "taskbar"
flashes the taskbar button with no sound, and the reason in the window's
title; nowhere else a BEL. Nothing is drawn in the frame.

It covers the 6 criteria of [`plan.md`, Plan 63](../plan.md), written
before the code, with another agent's review the owner folded in, and the
two corrections the build found (a failed turn's own title; a turn's end
leaving the title once another turn starts). Nothing is committed or
pushed: the owner reviews first.

How it was checked:
- **Application tests** (`internal/app/ping_test.go`, 7 tests) on a
  scripted Claude Code session, the window's profile, clock and focus set
  by the test, the BELs counted from what the commands give the terminal
  (the harness's new `raw` hook), the ping's timers kept and delivered by
  the test; one on the `claude-subagent-approval` mock; one running Bubble
  Tea itself for the title it clears at the end.
- **Each change broken on purpose** through `go test -overlay`
  (`scratchpad\plan51\mutrun`; the list built from the source lines by
  `scratchpad\plan63\mkmutants.mjs`), the repository never written.
- **The launcher**: `acc --mock …` run twice from a new Windows Terminal
  window (`scratchpad\term\p63frag.ps1`), the fragment and settings.json
  hashed and timed before and after each run.
- **Windows Terminal**: the real application, built from this tree, in a
  tab of ACC's profile and in one of Terminal's default profile, the
  `claude-subagent-approval` mock and a copy of it cut after its first
  approval, a temporary home and app data, no CLI session, Bubble Tea's
  trace on (`scratchpad\term\p63.ps1`, from `p51.ps1`; the trace's lone
  BELs counted by `p63bells.mjs`); the focus moved to the desktop's
  window and back, the window's title read, the taskbar shot.
- **The timing step** alone, and **build, vet, gofmt, the whole suite**
  after the last code and document edit.

## The evaluation

```text
Evaluation - Plan 63, the ping - 6 criteria: 4 PASS, 0 FAIL
PASS 63.1 When: TestPingNeeds, the window without the focus in ACC's profile: one BEL each for an approval, a question, a turn completed, a turn failed and the session's end; none for streamed text after the approval rang, a tool, a subagent's end or an interrupted turn; the dashboard asking for focus reports (the home screen's: TestHomeBanner); TestPingFocused: with the focus, which the window has until Terminal says otherwise, no BEL and the title plain for three approvals, a question, a turn completed and a turn failed, nor once the focus went; an approval rung once not rung again after the focus came and went; TestPingMock: the claude-subagent-approval mock pings the same way, one BEL, "✓ Turn done"; in Windows Terminal the mock's turn end rang once in ACC's tab, the window without the focus, at 208x59 and 120x40
PASS 63.2 The bell's place and its limit: TestPingBell: no BEL with WT_PROFILE_ID unset or another profile's, one with ACC's in lower and in upper case, the title "◆ Approval waiting" in all four; TestPingLimit: a turn's end rung at once; a question 2 s after it asking a timer for the 3 s left, an approval 1 s later none more, one BEL for both at the 5 s mark; an approval answered before its mark, and one with the focus back before it, ringing none; 5 s after a BEL, at once again; in Windows Terminal, Command Prompt's profile wrote no BEL and changed the title all the same; in ACC's, an approval 0.65 s after the turn's BEL rang 5 s after it (18:48:56, 18:49:01), and the mock's first approval, which the recording answers 2 ms after it comes, rang none
PASS 63.3 The title: TestPingTitle: "◆ Question waiting · Agent Command Center", then "◆ Approval waiting · …" over it, kept with the focus back until answered, the question's again after the approval's answer, "Agent Command Center" after both; "✓ Turn done · …" until the focus came back; "✗ Turn failed · …" until another turn ran; "✗ Session ended · …" over that turn's failure until the focus; TestPingTitleCleared: Bubble Tea's own output, a view's title set and then cleared (ESC ] 2 ; BEL) when the program ends; in Windows Terminal the tab read "✓ Turn done · Agent Com…" and "◆ Approval waiting · Ager…", plain with the focus back (the turn's end) and after F2, a (the approval)
PASS 63.4 The acc profile: two acc --mock runs from a new Windows Terminal window (p63frag.ps1): the first rewrote the fragment at 18:41:40.654 (sha256 895E52E2… to A0999EBA…), the Cascadia Mono NF face kept and "bellStyle": "taskbar" added, and set settings.json's time (18:41:40.657), its bytes the same (sha256 7834893B… before and after); the second rewrote nothing, the fragment's and settings.json's times and hashes unchanged; each run's dashboard shown in the acc tab
Not verified: 63.5's last parts, the sound, which the harness cannot hear (the BEL went to ACC's profile, whose only bell style is the fragment's "taskbar": Terminal's settings hold no entry, bell style or default for it), and the owner's look at the shots. Its measured part holds: in ACC's tab at 208x59 and 120x40, the focus on the desktop's window before the mock's turn ended, one lone BEL in Bubble Tea's trace, the tab "✓ Turn done · Agent Command Center" with Terminal's own bell mark, the auto-hidden taskbar come up with the Terminal button flagged; the focus back, the title plain and the mark gone; the cut mock's waiting approval the same way; in Command Prompt's profile no BEL, no mark, the taskbar left hidden, the title changing all the same; review shots out-p63\turn-max\blurred-tabs-x2.png, focused-tabs-x2.png, taskbar-button-x3.png, blurred.png, approval-max\blurred-tabs-x2.png, answered.png, plain-max\blurred-tabs-x2.png, done-taskbar-3.png. 63.6's timing step: the laptop on battery (68 %) at 18:59, which the step's conditions exclude; its other parts hold: go build ./... ok; go vet ./... ok (Windows and Linux); gofmt -l . empty; go test ./... ok (16 packages) after the last code and document edit; no golden frame or mock file changed; 24 mutants through -overlay, each caught by a failing test (details); go test -race ./internal/app: no race, 418.7 s
Verdict: NOT DONE
Proposed next step: the owner's look at the shots and a listen to a ping in the acc tab; the timing step on mains power - waiting for approval
```

## After the evaluation: the timing step on mains power

The owner plugged the laptop in and the timing step ran alone, its exact
command, on 2026-10-28 from 19:30:02 to 19:31:22, once the VPN client's
system scan had gone quiet (details, The timing step).

```text
Evaluation - Plan 63, the timing step on mains power - 1 criterion: 1 PASS, 0 FAIL
PASS 63.6 The checks: the timing step, run alone with its exact command on mains power, all nineteen tests within their targets: TestBannerToFrame p95 1.58 ms (target 2 ms), TestHomePictureTime worst 190.4 ms (target 260 ms), TestKeystrokeToFrame p95 1.08 ms, TestStreamToFrame p95 4.36 ms, TestGraphFrameCost p95 1.00 ms, the others' p95 2.58 ms at most (target 16 ms), TestPasteToFrame max 3.00 ms (target 100 ms); with the first block's build, vet, gofmt, whole suite, golden frames, mutants and race detector
Not verified: 63.5's last parts, as the first block says: the sound, which the harness cannot hear, and the owner's look at the shots
Verdict: NOT DONE
Proposed next step: the owner's look at the shots and a listen to a ping in the acc tab - waiting for approval
```

**63.5, the owner's review (2026-10-28).** The owner ran a ping in the acc
tab and sent their own shot of the taskbar at 19:42, the Terminal button in
Windows' colour for a window that needs attention, writing "I like it".
Asked whether the button takes that colour whenever the dashboard has
something for them, they were told yes: for an approval or a question
waiting, a turn completed or failed, the session ended on its own, only
while the acc tab lacks the focus, at most once in 5 s, the reason in the
window's title, until they switch to the window; and that Terminal's own
setting, the profile's bell notification style, can change it. Asked
whether the ping played any sound: "No sound". 63.5's last parts, PASS, so
6 criteria, 6 PASS, 0 FAIL. Not committed: the commit waits for the
owner's approval. The blocks above stay as they were reported.

## Details

### What the checks found and changed

- **A failed turn's title.** The plan listed four reasons; a turn that
  failed would have read "✓ Turn done". It reads "✗ Turn failed · Agent
  Command Center", between the session's end and a turn completed in
  urgency (design D4, as amended in step 2).
- **A turn's end once another turn started.** The first version kept a
  turn's end in the title while the turn machine was idle. TestPingNeeds
  caught it: a turn started and interrupted within one batch of events
  left the failed turn before it in the title. A turn's end now waits only
  while it is the last turn (`State.Stages`), which also drops it when the
  CLI opens a turn itself.
- **The plain title at the end.** The review asked that the title be
  restored; Bubble Tea already does it: `cursedRenderer.close` writes an
  empty title when the last view had one, on every end, a recovered
  panic's too (`stopRenderer` closes the renderer with or without the
  final flush). TestPingTitleCleared holds it on Bubble Tea's own output,
  so the dashboard keeps no flag of its own. Windows Terminal then shows
  the tab's own title.
- **The harness.** `harness.raw` (`mock_test.go`) sees what a command gives
  the terminal, as the program writes it, for the BEL count.

### The mutants

Each change broken on purpose, one at a time, through `go test -overlay`
with `-run 'TestPing|TestHomeBanner'` on `internal/app` (mutrun; the list
from `scratchpad\plan63\mkmutants.mjs`), the repository never written (its
`git status` the same before and after each run). All 24 caught by a
failing test:

| Mutant | Broken | Caught by |
| - | - | - |
| P1 | focus reports on the home screen only | TestPingNeeds: no focus reports on the dashboard |
| P2 | approvals noticed with the focus | TestPingFocused: the title "◆ Approval waiting" with the focus |
| P3 | a question taken for an approval | TestPingNeeds: the question's title |
| P4 | an interrupted turn pings | TestPingNeeds: 1 BEL for it |
| P5 | a failed turn titled done | TestPingNeeds: "✓ Turn done" for it |
| P6 | the session's end not noticed | TestPingNeeds: 0 BELs for it |
| P7 | a turn's end waiting through a turn started and ended since | TestPingNeeds: "✗ Turn failed" after the interrupted turn |
| P8 | the focus back leaves a turn's end in the title | TestPingFocused: "✗ Session ended" with the focus |
| P9 | the focus back drops an approval still waiting | TestPingFocused: the title plain |
| P10 | the focus back leaves a deferred BEL due | TestPingLimit: 4 BELs |
| P11 | the bell in every profile | TestPingBell: 1 BEL with no profile |
| P12 | the profile's id compared with its case | TestPingBell: 0 BELs for the id in capitals |
| P13 | no limit | TestPingLimit: 2 BELs 2 s apart |
| P14 | the timer for 5 s whatever is left | TestPingLimit: a 5 s timer |
| P15 | a second timer while one is due | TestPingLimit: timers 3 s, 2 s |
| P16 | the timer's message leaves it due | TestPingLimit: no timer asked after it |
| P17 | the BEL's time not kept | TestPingLimit: 2 BELs 2 s apart |
| P18 | the least urgent need titled | TestPingLimit: the question's title over the approval |
| P19 | the title always the name | TestPingNeeds: the approval's title |
| P20 | needs never dropped | TestPingNeeds: the approval's title once answered |
| P21 | an approval waiting once answered | TestPingNeeds: the same |
| P22 | no ping at all | TestPingNeeds: 0 BELs |
| P23 | the focus back not seen by the ping | TestPingFocused: "✗ Session ended" with the focus |
| P24 | something else than a BEL written | TestPingNeeds: 0 BELs |

The first run applied 21: mutrun matches whole lines, and three old texts
(P5, P14, P19) ended inside a line; two (P11, P12) left an import unused and
failed to build, which counts for nothing. The five rewritten, a second run
caught each by a test; the generator now refuses an old text that does not
end a line.

### The launcher

`scratchpad\term\p63frag.ps1`: `acc.cmd --mock
testdata\mock\claude-subagent-approval.jsonl` run twice, each from a new
Windows Terminal window, its dashboard quit with Ctrl+Q once shown and the
window closed by its handle.

| When | Fragment | settings.json |
| - | - | - |
| before | 23:48:14.284 the day before, sha256 895E52E263E594F7…, no bellStyle | 00:19:07.749, sha256 7834893B6571EFEA… |
| after the first run | 18:41:40.654, sha256 A0999EBA465A9403…, `"bellStyle": "taskbar",` after `closeOnExit`, the face kept | 18:41:40.657, the same hash |
| after the second run | unchanged | unchanged |

The first attempt found `acc.cmd` "not recognised": this shell's
`NoDefaultCurrentDirectoryInExePath` reaches a Terminal started from it,
so `cmd` no longer looks in the current folder; the script names it by its
full path.

### The Windows Terminal runs

`scratchpad\term\p63.ps1`, from `p51.ps1`'s launch: the application built
from this tree (`acc-p63.exe`), a temporary home, local and roaming app
data, a PATH of System32 only, Bubble Tea's trace on (`TEA_TRACE`), the
tab's `WT_PROFILE_ID` written to `profile.txt`; a mock dashboard, so no
CLI was started and no quota used. Windows Terminal 1.23.20211.0. Unlike
`p51.ps1` the application's title is left as it sets it (no `--title`, no
`--suppressApplicationTitle`). The focus goes to the desktop's window
(`GetShellWindow`) and back; the window's title is read with
`GetWindowTextW`; the lone BELs, those that end no OSC, DCS, APC or PM
string, are counted in the trace by `p63bells.mjs` (a title's OSC 2 ends
with a BEL too).

| Run | Profile | Result |
| - | - | - |
| `turn-max`, 208x59 | ACC's, `{27195de0-…}` | 0 BELs before the turn's end; at it 1 BEL (18:47:54), the title "✓ Turn done · Agent Command Center"; the focus back at 14.1 s, the title plain; 1 BEL in all |
| `turn-90x30`, 120x40 | ACC's | the same (1 BEL, 18:50:17) |
| `approval-max`, 208x59, the mock cut after its first approval | ACC's | the turn's BEL (18:48:56), then the approval's 5 s later (18:49:01); "◆ Approval waiting · Agent Command Center" while away and after the focus came back; F2, a: "allowed by you 18:49:07", the title plain; 2 BELs in all |
| `plain-max`, 208x59 | Command Prompt's, `{0caa0dad-…}` | no BEL; "✓ Turn done · Agent Command Center" while away, plain with the focus back |

Seen in the shots: Terminal marks the tab with a bell beside the title
while a BEL waits for the focus, and clears it with the focus; on this
machine the taskbar hides itself (`StuckRects3`, read through WMI), and a
BEL brings it up with the Terminal button in Windows' colour for a window
that needs attention, steady over the six shots 300 ms apart; in Command
Prompt's profile, no mark and the taskbar left hidden.

Two harness faults found on the way: `Term.Focus` restores a window with
SW_RESTORE, which un-maximised the window brought back from the desktop
(`P63.Front` restores only a minimised one); `GetWindowText`'s ANSI form
reads "✓" as "?" (the trace showed the dashboard writing "✓").

The owner's terminal-hosted CLI sessions: none were open, and no Windows
Terminal ran before the check (the desktop app's own sessions were left
alone). Each run's window closed when its dashboard quit; none was left.

### The race detector

`go test -race ./internal/app -count=1`, the package the plan changed,
with the portable WinLibs GCC 16.2.0 first on the run's own PATH and
`CGO_ENABLED=1` (evaluation.md, The race detector;
`libsynchronization.a` resolved): ok in 418.7 s, no race reported. The
ping adds no goroutine: it runs in `Update`, its timer through
`Options.After`.

### The timing step

First due at 18:59 and 19:07, the laptop on battery both times (68 %,
the CPU at 17 to 51 % over 2 s; 61 %, 23 to 70 %), which the step's
conditions exclude: not run then. Once the owner had plugged it in, the
VPN client's system scan (`cscan`) took about 6 % of the machine at 19:29
and went quiet at 19:29:42. The step then ran alone with the exact command
of `doc/evaluation.md`, Keystroke timing, from 19:30:02 to 19:31:22: mains
power, the battery at 58 %, the CPU at 25, 43, 54, 19 and 17 % just
before; open: Chrome, Edge, Teams, Outlook, Claude, the VPN client's
window, Move Mouse and Settings. No other `go` build or test ran.

| Test | Result | Target |
| - | - | - |
| TestGraphFrameCost | p95 1.00 ms (p50 under the clock's tick) | p95 ≤ 16 ms |
| TestBlameFrameCost | p50 0.60 ms, p95 1.51 ms | p95 ≤ 16 ms |
| TestBranchesFrameCost | p50 1.00 ms, p95 2.01 ms | p95 ≤ 16 ms |
| TestKeystrokeToFrame | p95 1.08 ms; the draft window 1.11 ms | p95 ≤ 16 ms |
| TestDraftKeystrokes | p95 2.56 ms | p95 ≤ 16 ms |
| TestLongSession | p95 2.58 ms | p95 ≤ 16 ms |
| TestThinkingFrame | p95 2.39 ms | p95 ≤ 16 ms |
| TestPasteToFrame | max 3.00 ms | ≤ 100 ms |
| TestStreamToFrame | p95 4.36 ms | p95 ≤ 16 ms |
| TestSelectorToFrame | p95 1.00 to 1.06 ms | p95 ≤ 16 ms |
| TestMotionToFrame | p95 1.02 to 1.52 ms | p95 ≤ 16 ms |
| TestDiffToFrame | p95 1.01 to 1.10 ms | p95 ≤ 16 ms |
| TestMessageToFrame | p95 1.01 and 1.72 ms | p95 ≤ 16 ms |
| TestOutputToFrame | p95 1.02 ms | p95 ≤ 16 ms |
| TestToolsToFrame | p95 2.44 ms | p95 ≤ 16 ms |
| TestBannerToFrame | p95 1.58 ms, max 2.86 ms, 11,355 bytes a frame | p95 ≤ 2 ms |
| TestHomeKeysToFrame | p95 0.15 ms | p95 ≤ 16 ms |
| TestHomePictureTime | p50 176.4 ms, max 190.4 ms | ≤ 260 ms |
| TestKeystrokeCost | 218 µs, 448,657 bytes and 597 allocations a key (10,000 lines); 558 µs, 1,452,974 bytes and 597 allocations (50,000 lines) | reports only |

The ping adds to each message a look at the approvals and turns added
since the last one and at the needs that wait, a handful at most, and to
each frame the title's choice among them.
