# Plan 69 evaluation (2026-10-30)

The check of plan 69, the session's resumption from the home screen (brief
amendment 71): each project keeps, per agent CLI, the last session the
dashboard ran there; the home screen offers the chosen CLI's for the
chosen project, and Ctrl+R or a click on its note starts the CLI on it,
Claude Code with `--resume <id>`, Codex with `thread/resume` and
`excludeTurns: true`; zone B says the session was resumed and that its
earlier entries are not shown; a refusal for want of the session clears
the record, any other failure keeps it, and nothing falls back to a new
session. The owner approved the implementation on 2026-10-29: "Plan 69 is
approved for implementation, including D9 and the updated telemetry
acceptance criteria.", the crash resume kept as it is.

It covers the 13 criteria of [`plan.md`, Plan 69](../plan.md), written
before the code. Nothing is committed or pushed: the owner reviews the
results and the diff first.

How it was checked:
- **The probes first** (69.1), P69a and P69b, before any code: each CLI's
  resume in a new process and its refusal of a session it does not have,
  recorded and replayed by the tests.
- **Tests** on the store (`internal/input`), each adapter on P69a's and
  P69b's lines and on its fake CLI, the state machines (one case per row of
  events-and-state.md), the application through the real adapters over the
  fake CLIs, as after a full restart (`internal/app/lastsession_test.go`),
  and the rendering (`internal/render/resume_test.go`); four golden frames.
- **Each change broken on purpose** through `go test -overlay`, the
  repository never written, its code hashed before and after: 57
  mutants (`scratchpad\plan69\mkmutants69.mjs`, `mutrun69.sh`).
- **Windows Terminal** (69.10): ACC's profile, maximised and at 120x40, a
  stand-in `claude.exe` (no CLI ran, no quota).
- **The live run** (69.11): both CLIs, two turns each, in a temporary
  project, the configuration check before and after.
- **Build, vet (Windows and Linux), gofmt, the whole suite** after the
  last code edit, `-race` on the changed packages' tests, and **the timing
  step** alone, on mains power.

## The evaluation

```text
Evaluation - Plan 69, the session's resumption from the home screen - 13 criteria: 13 PASS
PASS 69.1 The probes: P69a (Claude Code 2.1.303, 20:48 to 20:49) and P69b (Codex 0.163.0, about 20:50) on 2026-10-29, before any code, within the four model calls the proposal named (two on Haiku, two on Codex's configured model), in throwaway folders outside the repository; configuration PASS before and after (confcheck -cli-ran); recorded with their drivers in doc/research/probes and copied to testdata/probes (77 and 100 lines, each naming its process). No central assumption contradicted; they refined D5 (excludeTurns in the stable schema; thread/goal/cleared after every resume; the full-history answer's deprecationNotice) and D7 (Claude Code's result error_during_execution "No conversation found with session ID: <id>" before the handshake's answer, then exit 1; Codex's -32600 "no rollout found for thread id <id>" and "invalid session id: …")
PASS 69.2 The record: TestProjectSessions: per agent CLI and project, the latest replacing the one before, the list given never changed; kept by Record and by a star; gone with its project; never added for a project the list no longer has; cleared only while it names the refused session; written in version 1 and read back; a file without it read as before; one without an ID never offered. TestRecordSession: the owner's turn ended, completed, failed or interrupted, records the CLI's ID with the turn end's time; a turn the CLI opened (the usage read), a mock run or a session with no ID yet records nothing. TestLastSessionRestart: written when the owner's turn ends, not at the handshake; through Update, a project another dashboard recorded meanwhile kept
PASS 69.3 Matching and restart: TestLastSessionRestart: a new dashboard on the same store, as after a full restart, offers the record ("Claude Code's last session here: now · Ctrl+R continues it"), Ctrl+R spawning Claude Code with --resume s-first; TestLastSessionMatching: only the chosen CLI's record in the chosen project, the path in capitals and with slashes the same project; never for Codex, whose session is not recorded, for another project or for a worktree of it (<project>.worktrees\feature), Ctrl+R then starting nothing; TestStateMachines' row "Validating | both valid, the last session's project not the path validated": back to Asking on the project field, "Not resumed: the last session is C:\work\other's, not C:\work\proj's"
PASS 69.4 The note and the key: the golden frames startup-resume at 80x24, 120x40 and 200x60 (and 79x24's too-small screen), "→ Claude Code's last session here: 3m ago · Ctrl+R continues it" under the field, the only frames that changed; its time as the list writes one ("now", "3m ago", "2h ago"); none while a start is validated (TestResumeNoteValidating); TestLastSessionCodex: Codex's note, a click on it in picture mode (208x59) and in grid mode (80x24) resuming the thread, thread/resume with "threadId":"th-0069" and "excludeTurns":true, no thread/start; TestLastSessionRestart: Ctrl+R and --resume <id> for Claude Code, START SESSION still a new session; TestResumeNoteTarget: the note a click target on its line's middle row and on no row it does not touch, at every picture-mode size of the even widths 100 to 240 and the heights 30 to 70; the other startup frames unchanged
PASS 69.5 The resume, bounded: TestResumeLast: the home screen's thread/resume {"threadId":"th-9",…,"excludeTurns":true}, the rest of the handshake as a crash resume's; the crash resume's request as before (TestResume unchanged and passing; mutant C1, excludeTurns for every resume, caught by it); TestGoalCleared: thread/goal/cleared after each of P69b's resumes a status ("goal cleared"), so the resume without its turns has no unrecognised line, the full-history resume's deprecationNotice unrecognised as the crash resume has it today; Claude Code's arguments a start's with --resume <id> (one args function for both resumes, TestChildEnv's "--permission-mode default --resume abc"; TestLastSessionRestart); TestResumeLastWired: the home screen's resume told to the mapper, the crash resume's not
PASS 69.6 Zone B resumed: once the CLI names the session, "Resumed Claude Code session s-first · Claude Code has its earlier context; its earlier messages are not shown here" (TestLastSessionRestart) and "Resumed OpenAI Codex thread th-0069 · OpenAI Codex has its earlier context; its earlier turns are not shown here" (TestLastSessionCodex); another ID: "Claude Code reports session s-other-, not s-asked-: its earlier context may not be there", that session then the record (TestLastSessionOtherID); TestStateMachines' Session row "Ready | the first session.started after a resume from the home screen"; the starting line "Resuming Claude Code session 21f3b775 in C:\work\demo-project…" and "Resuming OpenAI Codex thread 01a18936 in …", a new session's "Starting Claude Code in …" (TestResumingLine); then the turns as usual (the live run's turn under the note)
PASS 69.7 Telemetry after a resume: TestResumedTelemetry, P69b's and P69a's lines through the mappers, the state and zone A's pane: Codex without its turns, once answered, context 6% of 258.4k, tokens 16.4k, cost N/A (Codex reports none), as with its turns; after the first new turn tokens 32.9k (32,897: 16,430 and 16,467), context 6% of 258.4k, the turn's duration; the replay's line taken out, context and tokens N/A (no update since the resume), then 32.9k. Claude Code: context N/A (no turn since the resume), tokens and cost N/A (no result since the resume) until the first new turn's result, then 4% of 1M, 85.3k and $0.01; a new session's reasons unchanged. On the screen, TestResumedUsageRows at 208x59: "Context N/A no turn since the resume", "Tokens N/A no result since the resume", "Est. cost N/A no result since the resume", the section's three rows while the reasons do not fit the columns, the columns once the values come; Codex's columns with its replay, its rows without it; a new session's columns as before
PASS 69.8 A resume that fails: Claude Code's TestResumeRefusal: P69a's refusal for good for the home screen's resume, the crash resume's as before, other words or another ID another failure, after the handshake's answer a turn's; Codex's TestResumeLastRefused (no rollout for th-9 and the malformed id for good; "thread th-9 is locked", another thread's id and -32603 another failure; no thread/start after any) and TestResumeRefusalP69b (P69b's two refusals for good for the home screen's resume of that id only); TestLastSessionRefused, four cases: "✗ The session could not be resumed" with "Claude Code no longer has this session: it is no longer offered" or "The session stays offered: Ctrl+R on the startup screen tries again"; the record cleared or kept; one spawn, no new session asked for; back home "Not resumed: Claude Code no longer has its last session here", "Not resumed: the CLI exited unexpectedly" or "Not resumed: handshake failed: thread/resume refused by Codex: thread th-kept is locked", drawn, for the chosen CLI in the chosen project only; START SESSION after a refusal a new session; Ctrl+R again after a kept one; TestLastSessionRefusedUnsaved: the store not writable, the refused session not offered again, "Recent projects not saved: …"; TestStateMachines' rows (the refusal, the widened Starting and Back rows)
PASS 69.9 Nothing else: the crash resume's requests and tests unchanged and passing (TestResume, TestCrashAndResume, TestCodexCrashAndResume, TestLastSessionCodex's crash resume without excludeTurns); the golden frames: the four new startup-resume frames alone, no other changed; the existing test files: helpers alone (startCtx in both adapters, wireCodex's resumeErr, the fake Claude Code's noconversation mode, a golden case) and two counts moved with P69a and P69b, each with its history comment (TestStreamEventsTakeTheMessageStartID, 1,454 stream events in 16 of 20 recordings; TestReplayNotes); go test ./... ok
PASS 69.10 Windows Terminal: ACC's profile, 1.23.20211.0, maximised (208x59) and at 120x40, the stand-in claude.exe, no quota, the final build: the dashboard quit and started again on its projects.json (s69-0001 recorded); the note at both sizes; Ctrl+R and a click on the note starting the resume, zone B's "Resumed Claude Code session s69-0001"; a refused resume's card, then "✗ Not resumed: Claude Code no longer has its last session here" and no offer, Enter starting a new session; a kept one's card, then the offer and "✗ Not resumed: the CLI exited unexpectedly", Ctrl+R trying again; 16 runs (10, and 6 with a Start menu folder), every poll found. The note's click target at 120x40 fixed on the way (Details)
PASS 69.11 Live: a temporary project, the configuration check PASS before and after each run, the owner's projects.json put back; Claude Code 2.1.303 on Haiku 5.5 and Codex 0.163.0 on gpt-6-sol, two turns each, 47 to 51 s a run: a session started from the home screen and told a code word, recorded; the dashboard quit and started again; Ctrl+R on the home screen; zone B's "Resumed Claude Code session 47d88972" and "Resumed OpenAI Codex thread 01a189d7"; zone A as 69.7 expects: Codex's 6% of 258.4k │ 16.4k │ N/A once resumed, Claude Code's three N/A "since the resume"; after the turn 4% of 1M† │ 88.4k │ $0.01 and 6% of 258.4k │ 32.8k │ N/A; "CINNABAR-69" and "BASALT-69" answered, the same ID recorded. Claude Code's first run failed L69.4 on zone A's columns; fixed, it passed (Details)
PASS 69.12 The checks: go build ./... ok; go vet ./... ok (Windows and Linux); gofmt -l . empty; go test ./... ok (16 packages, 351 s) after the last code edit; 57 mutants through -overlay on the final code, the repository never written (its hash 3a9616c6 before and after): 57 caught by a failing test, none by the compiler, none void; -race on the changed packages' tests: 0 race reports; the timing step alone, on mains power, at 00:41:53: its 21 tests within their targets
PASS 69.13 The documents: ui-design.md (the home screen's last session with its 80x24 excerpt, Ctrl+R and the click, zone B's note, starting line and card, the usage section's rows after a resume; TestDocFrames ok); events-and-state.md (the start that resumes, the failure's two outcomes, the Session row; TestStateMachines ok); reliability.md (Resumption across dashboard runs, the refusals' words labelled undocumented; After a resume from the home screen; Not verified's new row, settled); architecture.md (ASM-24, the section and its parts, excludeTurns); codex-cli.md (finding 4) and telemetry-sources.md (Tokens), P69b's usage replay; brief amendment 71; AGENTS.md's status and MVP scope (TestAgentsMDWithinCodexLimit ok) and plan.md's two [Nice to have] entries
Not verified: the owner's visual review (the home note, zone B's note and card, zone A's rows: the shots in Details); a refused resume through the dashboard on a real CLI (P69a and P69b recorded both refusals live, which the tests replay; the live run resumed sessions the CLIs had, as 69.11 asks); grid mode in Windows Terminal (a window too small for the picture: the golden frames at 80x24 and 79x24 and TestLastSessionCodex's grid case); a session the CLI has pruned or rotated away, which it would refuse as P69a and P69b recorded
Verdict: DONE
Likely cause: -
Proposed next step: the owner's review of this report and of the diff, then the commits on the owner's approval; the findings outside plan 69 (Details) for the owner to decide - waiting for approval
```

## The owner's acceptance run (2026-10-30)

The owner ran the [manual checklist](#manual-acceptance-checklist-for-the-owner)
in Windows Terminal on Claude Code 2.1.303 and OpenAI Codex 0.163.0: "Core
resume functionality PASS, with two UI issues requiring review before final
approval." Both issues were reproduced, traced to their cause and fixed,
each with a regression test seen failing first. Neither came from plan 69's
code: each shows on the last commit, 3d395a8; plan 69's offer only made the
first more visible. The 13 results above stay as reported; these four
follow the owner's required next actions (fix P69-M01; reproduce, triage
and fix P69-M02; a regression test each; the checks; the documents),
written down after the fixes, which came first. The shots are in
`scratchpad\plan69\review`, numbered after the first review's 16.

```text
Evaluation - Plan 69, the owner's acceptance findings - 4 criteria: 4 PASS
PASS A1 P69-M01, the stale validation error: TestHomeFailureFollowsField: START SESSION on a missing folder shows "✗ Does not exist: …"; the project then chosen in the list (its row clicked) takes it away and shows its last session's offer; the failed text typed back shows it again, one character less takes it away. Failed before the fix (the failure beside the offer, as in the owner's screenshot), passes after; at 3d395a8 the same steps keep the failure (a scratch test in a temporary checkout, removed). The fix: the project's failure shows while the field holds the text that failed (internal/render/home.go:160). Windows Terminal, the stand-in: the build before the fix shows the owner's screen (17), the fixed one the offer alone (18)
PASS A2 P69-M02, the field blank after a zoom: reproduced in Windows Terminal, maximised, at the first Ctrl and numpad plus (19). TEA_TRACE: two size reports, 188x59 then 188x53; the frame wrote the path on row 37; the next write rewrote the cells of the images drawn at 208x59 at their old size, 65 writes outside the 188x53 screen, rows 54 to 59 clamped to row 53, each wrap there scrolling the screen, 7 rows in all; the terminal's text then held the path on row 30, under the picture, row 37 blank. The fix: a vacated image's cells written over only where the screen still has them (internal/app/preview.go:416-429). TestHomeShrink (208x59, then 188x53): every write inside the screen, the field's row holding the path; 65 writes outside with preview.go as at 3d395a8 (-overlay), none after. Windows Terminal, the fixed build: two zooms (188x53, 170x48) and Ctrl+0, the path on its row each time, 0 writes outside in 119 image writes (20 to 22); at 120x40, a zoom into grid mode (108x36), the path shown and Ctrl+R resuming there (24, 25), where the build before scrolled the whole screen (23); a maximised window restored to 160x40, the field blank before (26, 79 writes outside), its path after (27, 0); the dashboard under 100 columns, the sidebar's tiles past its edge, zone B's first cells overwritten before (28, 37 writes outside), whole after (29, 0)
PASS A3 The checks after the fixes, on the final tree (code hash b54e3362 throughout): go build ./... ok; go vet ./... ok; gofmt -l . empty; 9 mutants through -overlay, the repository never written: 9 caught by a failing test, none void (a first run's two survivors gave TestHomeShrink its check that no old place is cut short; Details); -race on internal/render and on internal/app's 48 tests of the home screen and its images: 0 reports; go test ./... ok (16 packages, 1,108 tests, 5 skipped as before, 247 s; a run before failed TestWideRowsKept's 600 ms precondition under load, which passed 10 times of 10 alone); the timing step alone, on mains power, at 03:26:14: its 21 tests within their targets (a first run's TestBannerToFrame p95, 2.21 ms over 2 ms, timed in turn with the tree before the fixes: no code cause; Details)
PASS A4 The documents: ui-design.md (the project's failure follows the field; the picture's old places written over within the screen; the vacated images' rule), TestKeyOverlayRows ok (the test that reads it); TestDocFrames run with ACC_DOC_FRAMES set: its 80x24 frame the document's, line for line, its 120x40 and 200x60 frames one line apart, the Git Graph's tile since plan 62 (finding 5); this section, the checklist's results and plan.md's Progress
Not verified: the owner's look at the two fixes; a window made smaller by dragging its edge (the restore's path, 26 and 27; not run); the dashboard's other images after a smaller size (thumbnails, toggle, slider, mode pill, commit controls), written over through the same clipped write, not run
Verdict: DONE
Likely cause: -
Proposed next step: the owner's check of the two fixes, then the commits on the owner's approval - waiting for approval
```

**One existing test's expectation changed**, for the owner's review:
`TestSideDraw`'s step under 100 columns (plan 43) asserted that the
sidebar tiles' places were written over at column 114 of a 99-column
screen. A write there lands at column 99 and wraps onto the next row's
first cells, the damage of shot 28. The step now asserts that nothing is
written there; the code before the fix fails it (`-overlay`: one drawing,
`ESC[4;114H` and six cells, and each of the tiles' other rows the same).

### The owner's results

- **Claude Code**, session `a4ed17dd-eb15-447d-804a-320935d0c194` in
  `acc-p69-manual`: created and recorded in `projects.json`; after a full
  restart, the note; resumed with Ctrl+R and with a click on the note,
  maximised, at exactly 120x40 and in a narrower layout; "TEAL-69"
  recalled; zone A's reasons "since the resume" before the next turn, its
  values after it, the history's tokens included.
- **Codex**, thread `01a18a53-b72f-7993-a2b2-8512023e2340`, GPT-6-Sol at
  Low: recorded beside Claude Code's; after a restart, Codex's own note;
  resumed with Ctrl+R; "COBALT-69" recalled; zone A at once 6% of 258.4k,
  about 16.4k tokens, EST. COST N/A; after the first new turn about 32.8k.
- **Isolation**: the offer follows the chosen CLI; Ctrl+R does nothing for
  a CLI with no session; another project with no session shows no offer,
  Ctrl+R doing nothing; back on the project, the offer again.
- **Layouts**: maximised, 120x40 and narrower; the note clickable when it
  wraps.

### The owner's regression run of the fixes

The owner checked both fixes in Windows Terminal on the final code (the
build from it is byte for byte the one of shots 18, 20 to 22, 24, 25, 27
and 29, `6c5aeaec…`), with no model turn: "Both P69-M01 and P69-M02 are
confirmed fixed in Windows Terminal."
- **P69-M01, PASS**: the original scenario again, a path that does not
  exist, "Does not exist" shown, then `acc-p69-manual` chosen in RECENT
  PROJECTS: the path updated, the stale error gone at once, Claude Code's
  resume note still shown.
- **P69-M02, PASS**: maximised, one zoom and two zooms, then Ctrl+0; 120x40
  exactly, the home screen as it opens, then zoomed in step by step up to
  nine steps; the path visible in each, cut to fit at the narrow widths;
  the resume note reachable, never over START SESSION; no corruption, no
  panel out of place. Nine steps in, Ctrl+R resumed Claude Code's session
  `a4ed17dd-eb15-447d-804a-320935d0c194`, zone B saying it was resumed, the
  compact screen working.

This settles the A block's first item not verified, the owner's look at
the two fixes; the block stays as reported. Its two other items stayed
not verified then: a window made smaller by dragging its edge, and the
dashboard's other images (thumbnails, toggle, slider, mode pill, commit
controls) after a smaller size; the owner's final report, below, settles
both in the scenarios it names.

### The owner's final acceptance report

The owner's report of 30 October 2026, on the corrected build: "Overall
owner acceptance result: PASS for all manually executed scenarios." No
model prompt was submitted and no project code changed during these runs.
- **P69-M01**: PASS, as above.
- **P69-M02, the window's edge dragged**: PASS. Besides the zoom in nine
  steps at 120x40 (above), the Windows Terminal window narrowed by its
  edge from about 950 to 770 pixels with a resumed Claude Code session on
  the dashboard: the compact layout taking over, zones B and C and the
  controls whole, the sidebar shown or hidden as the width allows; widened
  again, the full layout back. No first characters missing, no scroll, no
  unexpected wrap, nothing left behind.
- **The other images after a resize**: PASS, in these scenarios:
  - Source Control and its commit controls, a Git repository made in
    `%TEMP%\acc-p69-manual` alone with an untracked test file: the message
    field, the Commit button, its dropdown, the changes counter and the
    file's row; the window narrowed with Source Control open, the file's
    diff opened in the compact layout, then widened. No commit made.
  - The Claude models popup: the Effort slider, the Ultracode toggle and
    the mode pill, the window narrowed and widened with the popup open; the
    popup placed again, each control's state kept. No model or setting
    changed.
  - An image thumbnail: a 120x80 PNG of 319 bytes attached through Upload
    from computer, no message sent; its thumbnail, name, size and ✕ shown;
    narrowed and widened with no clipping or leftover; removed with ✕.
- **A real Claude Code refusal, end to end**: PASS, on Claude Code 2.1.303,
  which the owner first saw refuse the nonexistent ID in PowerShell. With
  no ACC running, `projects.json` backed up, the backup's SHA-256 the
  original's (1,879 bytes); the test project's Claude Code record pointed
  at the nonexistent ID `00000000-0000-4000-8000-000000000069`, of the
  same length; the offer shown; Ctrl+R: "The session could not be
  resumed", Claude Code's own "No conversation found with session ID: …",
  exit code 1 and its stderr, "Claude Code no longer has this session: it
  is no longer offered", zone A ENDED with its values N/A, no new session
  started. The registry, read while ACC ran: the `claude-code` record gone,
  the `codex` record kept. Esc: the home screen's red note, no offer,
  Ctrl+R doing nothing. ACC quit, the registry restored from the backup:
  its SHA-256 the original's, the real session's ID back, the test ID
  gone. No Claude Code or Codex session file modified or deleted.

Codex's refusal was not run against the real Codex through the dashboard:
it rests on P69b's live refusals, replayed by the tests. The scenarios
above are the ones the owner ran, not every size, image or layout.

### Verification boundaries

| Category | What |
| - | - |
As they stand after the owner's final report:

| Category | What |
| - | - |
| Manually verified by the owner, on the real CLIs | the record, both CLIs side by side; the full restart; Ctrl+R and the click on the note (maximised, 120x40, narrower); the context kept (TEAL-69, COBALT-69); zone A before and after the first new turn; the offer by CLI and by project; Ctrl+R doing nothing without a session; both fixes (zooms, Ctrl+0, the window's edge dragged); after a resize, the scenarios of the final report (Source Control and its commit controls, the models popup's Effort slider, Ultracode toggle and mode pill, an image thumbnail); a real Claude Code 2.1.303 refusing a resume through the dashboard, the record cleared and nothing offered after |
| Verified by the probes' recordings and automated tests, and by the stand-in in Windows Terminal | Codex's refusal for good (P69b's live refusals replayed); any other failure, the record kept (the stand-in's exit 3); another ID named by the CLI; the store not writable; the crash resume unchanged |
| Not manually verified | a real Codex refusing a resume through the dashboard, not run; the images and sizes beyond the scenarios run |
| Partially verified by the owner at first, now verified | grid mode: the owner exercised narrower layouts; the zoom from 120x40 to 108x36 drew the grid form (the terminal's text holding its box lines and the note as a text line; the trace's only image the banner strip, 1080x220 from row 3), and Ctrl+R resumed there with the stand-in (24, 25); then the owner, with the fixes, at 120x40 zoomed in up to nine steps, Ctrl+R resuming the real session there |

Retained as they were: the four findings outside plan 69 (Details), 120x40's
narrow form cutting a resumed session's reasons among them, and the fifth,
the Layouts frames' missing tile (below).

### The model calls

| Run | Claude Code, Haiku 5.5 | Codex |
| - | - | - |
| The probes P69a and P69b (69.1) | 2 | 2, its configured model |
| The live run (69.11) | 4: the 2 approved, and 2 beyond them, its run made twice | 2, gpt-6-sol |
| The owner's acceptance run | 2 (the checklist's 2.2 and 4.2) | 2, GPT-6-Sol at Low |
| The findings' work | 0: the stand-in `claude.exe` | 0: Codex not started |
| The owner's regression run of the fixes | 0: a resume makes no model call | 0 |
| The owner's final report (the edge drag, the images, the real refusal) | 0: no prompt; the refused resume makes no model call | 0 |
| All | 8 | 6 |

The usage reads ("/usage") are answered locally, with no model call.

### P69-M01: cause and fix

The validation's failure lives in the state (`Startup.Field`,
`Startup.Message`) until the next start or Ctrl+N, while choosing a project
in the list, the folder dialog or an edit change only the field's text (plan
53's list choice; the rule shown since plan 51). The note was drawn
whatever the field held, so it named a path no longer there, beside the
offer of the project chosen. Now the note is drawn while the field holds the
text that failed (`Startup.Project`, the field's text at Start), as plan
69's failure note shows while its CLI and project are the chosen ones
(design D7): a rendering rule, with no new action or state. The fixture of
`TestHomePictureNotes`, which set a failure for a text the field did not
hold, now holds it, as a start leaves them.

### P69-M02: cause and fix

`thumbSequence` (`internal/app/preview.go`) writes over the places of the
images drawn before that no image of the new frame takes, from the frame's
own cells, which erases them (plan 32). After a smaller size those places
still had the old size: `Region` pads past the frame, so each row of a
208-column image was written 208 cells wide on a 188-column screen, wrapping
20 cells onto the next row, and the rows past the 53rd, clamped to the last
row, scrolled the screen at each wrap. The rows the frame had just written,
the field's text among them, moved up under the picture, which is drawn
after at the new rows, leaving the field's own row blank. Now each old place
is cut to the screen first; what lies past it is gone with the columns and
rows the terminal dropped, and Bubble Tea's renderer erases the screen on
each size report. Any image drawn before a smaller size went this way: the
home screen's picture (a zoom, a window restored or made smaller) and the
dashboard's sidebar tiles under 100 columns (28).

### The checks after the fixes (A3)

On the final tree, its code hashed `b54e3362` (the Go files and
`testdata` against HEAD) before and after each run; evidence in
`scratchpad\plan69\m02\evidence`.

| Mutant (`-overlay`, `mkmut.mjs`, `mutrun.sh`) | Verdict |
| - | - |
| R1 the failure whatever the field holds (the code before the fix) | caught: `TestHomeFailureFollowsField` |
| R2 the failure only once the field changed | caught: `TestHomeFailureFollowsField`, `TestHomePictureNotes`, `TestStartupMessageFrames` |
| R3 the resolved path compared with the text typed | caught: `TestHomePictureNotes`, `TestStartupMessageFrames` |
| P1 no clip (the code before the fix) | caught: `TestHomeShrink`, `TestSideDraw` |
| P2 one row past the bottom | caught: `TestHomeShrink` |
| P3 one column past the right edge | caught: `TestHomeShrink` |
| P4 the last row not written over | caught: `TestHomeShrink` |
| P5 the first column not written over | caught: `TestHomeShrink` |
| P6 the old width written from the clipped place | caught: `TestHomeShrink` |

The first run let P4 and P5 survive, and its P1, the clip removed, did not
build (`screen` unused): `TestHomeShrink` then gained its check that each
row of the old picture still on the screen is written over from its first
column, and P1 became an identity in the clip's place; the second run caught
all nine.

Two results depend on the machine's load, the owner's applications open
throughout (CPU 5 to 48 %):
- **`TestWideRowsKept`** (plan 43), which runs the real program on the real
  clock and waits 600 ms for the tiles to be drawn before it checks
  anything, failed that precondition once in a whole-suite run ("0 Sixel
  images drawn before the frames checked"); alone, 10 runs of 10 passed,
  and the whole suites before and after passed it. The tiles' images are
  drawn by code the fix does not touch.
- **`TestBannerToFrame`**'s p95 was 2.21 ms in a first timing step, over
  its 2 ms target (1.77 ms in step 5). The banner's frames do not pass
  through the changed code. Timed in turn, the final tree 1.65, 1.72,
  1.68 and 1.69 ms, the tree without the fixes (`-overlay`) 2.62, 1.58,
  1.69 and 1.67 ms; the timing step run again at 03:26:14, CPU 20 to
  34 %: its 21 tests within their targets, the banner's p95 1.69 ms.

### A correction to 69.13, and a fifth finding outside plan 69

- 69.13 reads "TestDocFrames ok" for ui-design.md. That test reads
  nothing: it writes the Layouts section's frames into the folder
  `ACC_DOC_FRAMES` names, and without it is skipped, as it was. The test
  that reads ui-design.md is `TestKeyOverlayRows`, which passed then and
  passes now. The block stays as reported.
- **5. The Layouts frames at 120x40 and 200x60 lack the Git Graph's
  tile.** Drawn by `TestDocFrames` (A4), each differs from the document on
  one line, the sidebar's third tile ("◉", plan 62, e906b73), which the
  document's frames, last redrawn by plan 43, do not show; 80x24, which
  has no sidebar, matches. Older than plan 69 and outside it: one line in
  each frame, for the owner to decide.

### Seen once, not a defect

A shot 1.2 s after ↓ chose the project, with the build before A1's fix and
the test suite running beside it, showed the path a row above its box: the
note line more had grown the panels, and Windows Terminal was still drawing
the picture's whole redraw (360 KB). In a run with nothing beside it, the
screen was whole 1 s and 20 s after the key (17). The whole redraw on a new
layout is plan 51's (D13).

### The shots (`scratchpad\plan69\review`)

| Shot | What |
| - | - |
| 17, 18 | P69-M01 maximised: before the fix, "✗ Does not exist" beside the offer after the project chosen in the list; after, the offer alone |
| 19 | P69-M02 before the fix: maximised, one zoom, the field blank |
| 20, 21, 22 | after the fix: one zoom (188x53), two (170x48), Ctrl+0 (208x59), the path in its field each time |
| 23 | before the fix: 120x40 zoomed into grid mode, the whole screen scrolled up |
| 24, 25 | after the fix: the same, the grid form whole; Ctrl+R there resuming (the stand-in) |
| 26, 27 | a maximised window restored to 160x40: the field blank before, its path after |
| 28, 29 | the dashboard (mock) zoomed under 100 columns: zone B's first cells overwritten before, whole after |

### Approval, commit and push

**Approved and committed (2026-10-30).** The owner's final report closes
"Owner approval is granted." and allows the documents, the commit, the
push and the test data's cleanup once the final review finds no blocker;
it found none. Commit aba673a (60 files: 43 modified, 17 added), from the
files reviewed in `git status` and `git diff`, each of them plan 69's;
the owner's temporary Git repository and probe files, under `%TEMP%`, are
not among them. `git diff --cached --check` reported only the blank line
at the end of `testdata/frames/startup-resume-79x24.txt`: the too-small
screen padded to 24 rows, as every 79x24 frame, which `TestGoldenFrames`
needs whole. No file holds a CR or a BOM, read from the bytes; the only
trailing spaces are those ui-design.md and brief.md already had; a scan
of the change found no placeholder, leftover or private path. In that
commit, no code file and no document a test reads changed after the last
whole suite began (03:27:41): only plan.md and this file, which no test
reads. The author is `Evengelius <molimoabel@gmail.com>`, with no
attribution line, and the working tree was clean after it. Then the
approval commit: plan.md and AGENTS.md close plan 69, and this record;
AGENTS.md is read by a test, so the whole suite runs on the committed
tree. Not pushed yet: the final repository checks come first, then the
push. The blocks above stay as they were reported.

## Details

### What the checks found and changed

- **Zone A's usage after a resume, found by the live run.** The first live
  run of Claude Code (`acc-p69.exe`) failed L69.4: once the session was
  resumed, zone A's three columns took the reasons' widths ("no turn since
  the resume", 24 cells; "no result since the resume", 26), 82 cells with
  their rules where zone A has 56, and pushed EST. COST out: "CONTEXT │
  TOKENS …", "N/A │ N/A …", "no turn since the resume │ no result since
  the resu…". D9's wording, approved, is kept; the section now takes its
  three rows while a resumed session's reasons do not fit the columns
  ("Context N/A no turn since the resume", "Tokens N/A no result since the
  resume", "Est. cost N/A no result since the resume"), the same three
  rows high, and the columns again once the values come (`usageRows`,
  `internal/render/telemetry.go`;
  ui-design.md, Transition and Session usage). `TestResumedUsageRows`
  checks it at 208x59 on P69a's and P69b's lines; through `-overlay`
  without the change it fails with the very rows the live run showed.
  Limited to a resumed session: three of Codex's frames at 200x60 already
  cut its cost's reason before plan 69 (the findings below), and changing
  them is not plan 69's (69.9). The live run, re-run on the fixed build
  (`acc-p69b.exe`), passed (69.11).
- **The note's click target at 120x40, found in Windows Terminal.** In
  picture mode the note's line, 16 design pixels, is less than a row high
  at 120x40: its own band rounded to no row, so a click on it did nothing.
  The target is now every row the line touches (`homeTargets`,
  `internal/render/home.go`); `TestResumeNoteTarget` checks that the
  line's middle row is a target, and no row the line does not touch, at
  every picture-mode size of the even widths 100 to 240 and the heights 30
  to 70, the note alone, after one more note and after a long one wrapping
  over several lines; with the line's own band it fails (mutant R5).
- **Tests the mutants asked for.** On the tests first written, mutants
  survived, or had no test that could catch them, where a case was
  untested; each now has its test: Codex's refusal naming another thread
  is another failure (C5; `TestResumeLastRefused`); a stored session
  without an ID is never offered (S6; `TestProjectSessions`); zone B's
  starting line names the session or thread it resumes (R7;
  `TestResumingLine`); the note is not offered while a start is validated
  (R2; `TestResumeNoteValidating`); the home screen draws the failure's
  note, the tests having read only the field (R13; `TestNotResumedNote`,
  and the frame in `TestLastSessionRefused`); that note only for the
  chosen CLI in the chosen project (A9; `TestLastSessionRefused`, the other
  CLI and another project chosen, then the first again); a session its
  CLI refused for good not offered again when the store cannot be written
  and its record stays (A8; `TestLastSessionRefusedUnsaved`, the store's
  problem said). Two cases of the criteria had no test either: a worktree
  of the project, at `<project>.worktrees\<name>` where the Worktrees view
  adds one, its path beginning with the project's (S8, a prefix match;
  `TestLastSessionMatching`), and the record written through `Update`, a
  project another dashboard recorded meanwhile kept (A15, the dashboard's
  own list written; `TestLastSessionRestart`).
- **Two guards removed** (`internal/app/lastsession.go`), the mutants
  having shown they changed nothing: `resumeLast`'s check of the startup
  phase (A7), which the state's own refusal of a Confirm outside Asking
  duplicates, as START SESSION has it, and `lastSession`'s check of a mock
  run (A11), which cannot be reached: a mock run never shows the home
  screen. The live runs' build (`acc-p69b.exe`) still had them; on the
  paths those runs took, Ctrl+R while Asking and a session that is no
  mock, the two behave the same.
- **Two counts of the recordings** moved with P69a and P69b, each with its
  history comment: `TestStreamEventsTakeTheMessageStartID`, 1,454 stream
  events in 16 of 20 recordings (P69a's 19, in two processes); Codex's
  `TestReplayNotes`, the recordings `appServer` lists and the two with
  their own tests (P39x, P69b).
- **Void kills in the mutants' first runs**, none counted: three runs of
  `internal/input`'s whole suite at once made its git tests fail
  (`TestCatFileNames`, `TestGitAdd` and others), so the store's mutants run
  its store tests alone; three test binaries that did not start
  (`0xc0000142`) when a run was stopped. The runner now reports a failure
  no test names as VOID, never as a kill.

### Findings outside plan 69, for the owner to decide

1. **The project field's box over the first note line** (plan 51's
   layout). In picture mode the box's bottom edge reaches into the first
   note line at about 13 to 17 % of the sizes, by up to 4.9 pixels, with
   any number of notes; at 120x40 with four note lines, 1.0 pixel, the
   line looking struck through, and with one line 3.7 pixels. Not changed:
   it is the approved home screen's layout, and plan 69's note is one note
   among those it already draws. A separate follow-up.
2. **Zone A's columns before Codex's first model call, at 200x60.** The
   reasons "no model call yet", "no update yet" and "Codex reports none"
   are a few cells too long for the pane's three columns, so "Codex
   reports none" is cut ("EST. COST    …") in three approved golden frames
   (`models-codex-200x60`, `models-codex-more-200x60`, `codex-step-200x60`).
   The rows form plan 69 uses for a resumed session would show them whole,
   at the cost of those three frames; left as it was (69.9).
3. **120x40's narrow form cuts a resumed session's reasons.** Zone A is 32
   cells wide there, so "Context N/A no turn since th…" and the like until
   the first new turn's result; each value reads N/A, the details view
   (Enter on TELEMETRY) gives the reasons whole. The narrow form cuts any
   text too long for it the same way.
4. **The harness's "notifications off" note.** In Windows Terminal the
   home screen of the stand-in runs adds "✗ notifications off: …", the
   temporary profile having no Start menu folder; the "-sm" runs make that
   folder, and the note then names the next thing the temporary profile
   lacks, `wt.exe` under its `WindowsApps` (plan 64's checks); the owner's
   profile has both. A harness artifact, not plan 69's.

### Verified and not verified

What each behaviour rests on: the tests (deterministic, on the probes'
recordings of the real CLIs and on fake CLIs), Windows Terminal with the
stand-in `claude.exe` (69.10), and the live run on the real CLIs (69.11).
As it stood before the owner's acceptance run; the boundaries since are in
[Verification boundaries](#verification-boundaries).

| Behaviour | Tests | Windows Terminal (stand-in) | Real CLIs |
| - | - | - | - |
| The record at the owner's turn end, kept across a full restart | yes | yes, both sizes | yes, both CLIs |
| Offered only for the chosen CLI and project (case, separators, a worktree, another CLI) | yes | the one CLI the stand-in is | no (the manual checklist's 6) |
| Ctrl+R resuming: Claude Code `--resume`, Codex `thread/resume` with `excludeTurns` | yes | yes, both sizes | yes, both CLIs |
| A click on the note resuming, picture mode | yes | yes, both sizes (120x40's fix) | no (the checklist's 5) |
| Zone B's note and starting line; another ID said as such | yes | the note, both sizes | the note, both CLIs |
| Zone A after a resume: values, reasons, the rows while the reasons do not fit | yes, on P69a's and P69b's lines | yes, both sizes | yes, both CLIs, maximised |
| A refusal for good: no fallback, the record cleared, the card and the home note | yes, on the refusals P69a and P69b recorded live | yes, the stand-in refusing as P69a recorded | **not verified**: no real CLI refused through the dashboard |
| Any other failure: the record kept, the card, the note, Ctrl+R again | yes | yes, the stand-in's exit 3 | **not verified** with a real CLI |
| The grid form of the home screen (a terminal too small for the picture) | yes: golden frames at 80x24 and 79x24, `TestLastSessionCodex/grid` | **not verified** in Windows Terminal | no (the checklist's 8.4, optional, covers it) |
| The crash resume unchanged | yes (`TestResume`, `TestCrashAndResume`, `TestCodexCrashAndResume`) | not re-run | not re-run (plan 4's live runs; its requests unchanged) |

The refusal through the dashboard on a real CLI would need a session the
CLI no longer has, that is deleting one of its sessions or writing a record
by hand: neither is done, as the owner asked.

### The live run (69.11)

ACC's profile (9 pt), maximised (208x59), Windows Terminal 1.23.20211.0;
the owner's home and the CLIs' own logins, every `CLAUDE*` and `ANTHROPIC_*`
variable of the window dropped, `ANTHROPIC_MODEL=haiku` for Claude Code,
Codex on its configured model (gpt-6-sol), Px through `HTTPS_PROXY` and
`HTTP_PROXY`; a new temporary project under `%LOCALAPPDATA%\Temp` each run,
removed after; `confcheck snapshot` before and `compare -cli-ran` after;
no Claude Code or Codex session in a terminal before (checked); the
owner's `projects.json` copied before and put back after, the only change
found being the run's project and the one recent project the cap of 10
dropped for it (SHA-256 equal to the copy after). Each run: the startup
screen with the project, Enter, a turn telling a code word ("Reply with
the single word OK"), the record awaited in `projects.json`, Ctrl+Q; a new
dashboard on the same project, Ctrl+R on the home screen, zone B's note,
zone A; a turn asking for the code word; Ctrl+Q
(`scratchpad\term\p69live.ps1`; shots and results in
`out-p69-live-claude-build1`, `out-p69-live-claude` and
`out-p69-live-codex`).

| Run | L69.1 before | L69.2 recorded | L69.3 quit | L69.4 resumed, zone A | L69.5 the code word, zone A | L69.6 after |
| - | - | - | - | - | - | - |
| Claude Code 2.1.303, Haiku 5.5, `acc-p69.exe`, 23:32:54, 49 s | PASS | PASS: d8a9cf59-…, ended 23:33:10 | PASS, the record kept | **FAIL**: "Resumed Claude Code session d8a9cf59" shown; zone A's columns with EST. COST pushed out (above) | PASS: "CINNABAR-69"; the record the same ID, ended 23:33:29; 4% of 1M† │ 88.5k │ $0.01 | PASS: configuration PASS, no CLI left, `projects.json` put back |
| Claude Code, `acc-p69b.exe`, 23:44:30, 51 s | PASS | PASS: 47d88972-…, ended 23:44:48 | PASS | PASS: "Resumed Claude Code session 47d88972"; "Context N/A no turn since the resume", "Tokens N/A no result since the resume", "Est. cost N/A no result since the resume" | PASS: "CINNABAR-69"; the same ID, ended 23:45:07; 4% of 1M† │ 88.4k │ $0.01 | PASS |
| Codex 0.163.0, gpt-6-sol, `acc-p69b.exe`, 23:45:29, 47 s | PASS | PASS: thread 01a189d7-…, ended 23:45:45 | PASS | PASS: "Resumed OpenAI Codex thread 01a189d7"; 6% of 258.4k │ 16.4k │ N/A "Codex reports none" | PASS: "BASALT-69"; the same thread, ended 23:46:02; 6% of 258.4k │ 32.8k │ N/A | PASS |

- **The configuration**: PASS each run; `~/.claude.json` changed in Claude
  Code's cache-state keys alone, one new session file a run (the resume
  continued the session it resumed), nothing else.
- **The quota**: four Haiku turns for Claude Code, its run made twice, the
  second after the fix the first found (each run the approved two turns):
  **two Haiku calls beyond the two 69.11 approved**, taken as a safe re-run
  of an approved run and reported here; two turns for Codex, as approved;
  the usage reads answered locally, with no model call.
- **What it shows beyond the tests**: each CLI resumed in a new process,
  after the dashboard quit and a new one started, kept the session's
  context (the code word); the record kept the same session or thread ID
  before and after the turn; zone A's values as 69.7 expects them: Codex's
  replayed at once (16.4k, the first turn's, as P69b's 16,430), both CLIs'
  totals after the turn holding the history (Claude Code 88.4k, Codex
  32.8k).

### Windows Terminal (69.10)

Windows Terminal 1.23.20211.0, ACC's profile, maximised (208x59) and at
120x40 (`-mode 90x30` at the profile's 9 pt), a temporary home and app
data, each run's `projects.json` put in place first (`seed69.ps1`); a
stand-in `claude.exe` in place of Claude Code (`scratchpad\plan69\standin`:
it answers the version, starts session s69-0001, resumes the IDs it made,
exits 3 for s69-transient and refuses any other ID as P69a recorded):
no CLI ran, no quota (`scratchpad\term\p69runs.ps1`, `p69.ps1`). No Claude
Code or Codex session ran in a terminal before the runs. The first build's
runs found the note's click target at 120x40 (above); after its fix the 16
runs passed on `acc-p69.exe` (`out-p69-build1`), and again on the final
build, `acc-p69d.exe` (`out-p69`), every poll finding its text:

| Run | What it does | 208x59 | 120x40 |
| - | - | - | - |
| first | a session from the home screen, a turn, Ctrl+Q: `projects.json` records s69-0001 | PASS | PASS |
| resume | a new dashboard on that store: the note; Ctrl+R; zone B's "Resumed Claude Code session s69-0001", zone A's three rows | PASS, and "-sm" | PASS, and "-sm" |
| click | the same with a click on the note, 3 rows under the field's text at 208x59 and 2 at 120x40 (2 and 1 with "-sm", one note fewer) | PASS, and "-sm" | PASS, and "-sm" |
| gone | dead-0001, refused as P69a: the card "The session could not be resumed"; Esc: "✗ Not resumed: Claude Code no longer has its last session here", no offer; Enter: a new session, "Ready to work" | PASS | PASS |
| kept | s69-transient, its exit 3: the card; Esc: the offer and "✗ Not resumed: the CLI exited unexpectedly"; Ctrl+R: the card again | PASS, and "-sm" | PASS, and "-sm" |

"-sm": the temporary profile's Start menu folder made (finding 4). The
shots to look at first: `out-p69\gone-max\g4.png` (the home screen after
a refusal for good), `kept-sm-90x30\t4.png` (the offer and a kept
failure's note at 120x40), `resume-max\r3.png` (zone B's note and zone A's
three rows); from the live run, `out-p69-live-claude\05-resumed.png` and
`out-p69-live-codex\05-resumed.png` (both CLIs resumed), and
`out-p69-live-claude-build1\05-resumed.png` (the columns it fixed).

### The mutants

Through `go test -overlay`, the repository never written (its code hashed
before and after, the same: 3a9616c6), three at a time, each run stopping
at its first failing package (`scratchpad\plan69\mkmutants69.mjs`,
`mutrun69.sh`; logs in `mut\logs\`, the verdicts in
`mut\verdicts-final.txt`). The store's runs are its four store tests,
`internal/app`'s the tests of plan 69's code (`TestLastSession`,
`TestGoldenFrames`, `TestStartup`, `TestHome`); every other package's whole
suite. On the final code and tests: 57 mutants, 57 caught by a failing
test, none by the compiler, none void.

| Mutant | Broken | Caught by |
| - | - | - |
| S1 | `Record` drops the project's sessions | TestProjectSessions |
| S2 | `RecordSession` records nothing | TestProjectSessions |
| S4 | `ForgetSession` clears the session whatever ID it names | TestProjectSessions |
| S6 | `LastSession` offers a session without an ID | TestProjectSessions |
| S7 | `LastSession` compares paths exactly (case, separators) | TestProjectSessions |
| S8 | a path beginning with the project's taken for it, as a worktree's | TestLastSessionMatching |
| C1 | `excludeTurns` for every resume, the crash's too | TestResume (the crash resume's, unchanged) |
| C2 | never `excludeTurns` | TestResumeLast |
| C3 | no refusal for good | TestResumeLastRefused, TestResumeRefusalP69b |
| C4 | every -32600 for good | TestResumeLastRefused |
| C5 | "no rollout found" for good whatever thread it names | TestResumeLastRefused |
| C7 | `thread/goal/cleared` unrecognised | TestGoalCleared |
| C8 | the home screen's resume not told to the mapper | TestResumeLastRefused |
| C9 | nor to the session | TestResumeLast |
| K1 | "No conversation found" for good whatever session it names | TestResumeRefusal (other words or another ID) |
| K2 | no refusal for good | TestResumeLastWired, TestResumeRefusal |
| K3 | the home screen's resume not told to the mapper | TestResumeLastWired |
| K4 | the resume asked not cleared at the handshake's answer | TestResumeRefusal (after the handshake's answer) |
| K5 | the refusal not marked for good | TestResumeLastWired, TestResumeRefusal |
| T1 | a validated path not the record's project resumed all the same | TestStateMachines (its row) |
| T2 | the start not marked as a resume | TestStateMachines (the refusal's and the failure's rows) |
| T3 | its command not the home screen's resume | TestStateMachines (a session to resume) |
| T4 | the refusal for want of the session as any failure | TestStateMachines (the refusal's row) |
| T6 | the failure not kept for the home screen | TestStateMachines (back; the refusal) |
| T7 | no note once the CLI names the session | TestStateMachines (the Session row) |
| T8 | the note on every `session.started` | the same |
| T9 | another ID said as the one asked for | the same |
| T10 | recorded in a mock run | TestRecordSession |
| T11 | recorded at a turn the CLI opened, the usage read's | TestRecordSession |
| T12 | a start leaves the last resume's failure | TestStateMachines (resume pressed) |
| T13 | the record not cleared after the refusal | TestStateMachines (the refusal's row) |
| T14 | the refusal not kept (`ResumeGone`) | the same |
| R1 | no note | TestNotResumedNote, TestResumeNoteTarget, TestResumeNoteValidating |
| R2 | the note while a start is validated | TestResumeNoteValidating |
| R3 | the grid's note no click target | TestLastSessionCodex/grid |
| R4 | the picture's note no click target | TestResumeNoteTarget |
| R5 | the target the line's own band, the first version | TestResumeNoteTarget |
| R5b | the target's last row floored | TestResumeNoteTarget |
| R6 | the card says "stays offered" after a refusal for good | TestLastSessionRefused |
| R7 | the starting line says "Starting" | TestResumingLine |
| R8 | a resumed session's reasons as a new session's | TestResumedTelemetry, TestResumedUsageRows |
| R10 | Codex's "no model call since the resume" | TestResumedTelemetry, TestResumedUsageRows |
| R11 | the columns whatever the reasons | TestResumedUsageRows |
| R12 | the rows for any session whose reasons overflow | TestGoldenFrames (models-codex-200x60, models-codex-more-200x60) |
| R13 | no failure note | TestNotResumedNote |
| A1 | Ctrl+R does nothing | TestLastSessionRestart |
| A2 | a click on the note does nothing | TestLastSessionCodex |
| A3 | the record not written | TestLastSessionRestart, TestLastSessionOtherID |
| A4 | the record not cleared | TestLastSessionRefused |
| A5 | the home screen's resume started as the crash's | TestLastSessionCodex, TestLastSessionRefused |
| A6 | the session context without `ResumeLast` | TestLastSessionCodex, TestLastSessionRefused |
| A8 | a session refused for good offered while its record stays | TestLastSessionRefusedUnsaved |
| A9 | the failure's note for any CLI and project | TestLastSessionRefused |
| A10 | the failure's note without "no longer has" | TestLastSessionRefused |
| A12 | the notes never computed | TestGoldenFrames (startup-resume) |
| A14 | the Confirm without its session: a new session | TestLastSessionCodex, TestLastSessionOtherID, TestLastSessionRefused |
| A15 | the dashboard's own list written for the record | TestLastSessionRestart |

Not made, equivalent today: `missing`'s source-reason check (no reason a
source gives ends in "yet"); the state's check that a refusal comes while
Starting, and Codex's clearing of the thread asked once `thread/resume` is
answered (a refusal comes only in that answer, before the handshake ends;
Claude Code's clearing is K4); `ForgetSession`'s path check (a session ID
belongs to one project). A7 and A11 survived the first full
run and led to the two guards' removal (above).

### The checks

| Check | Result |
| - | - |
| build, vet (Windows, Linux), gofmt | ok, ok, ok, nothing printed |
| `go test ./...`, after the last code edit | 16 packages ok, 351 s |
| `-race` (WinLibs GCC 16.2.0, `libsynchronization.a` resolved) | the adapters, the state, the rendering and the fixtures whole, ok (179 s); `internal/app`'s tests of plan 69's code and of the starts and the crash resume it touches (`TestLastSession`, `TestGoldenFrames`, `TestStartup`, `TestHome`, `TestProject`, `TestCodex`, `TestCrash`, `TestResume`), ok (52 s); `internal/input`'s store tests, ok; 0 race reports |
| the mutants | 57 on the final code and tests, all caught (above) |
| Windows Terminal, the final build | 16 runs, every poll found (above) |
| the live run | Claude Code twice (the first run's FAIL fixed), Codex once (above) |
| the build the live run used | `acc-p69b.exe` (6fe5faeb), the final code but for the two guards removed after it (above); the final build `acc-p69d.exe` (6c1c6544), which Windows Terminal's runs used |
| the files | 56, UTF-8 without a BOM, LF only (read from the bytes); no added line names the user's folder, no TODO |
| the timing step, alone, on mains power | below |

### The timing step

The command of evaluation.md, the twenty-one tests, alone, at 00:41:53,
95 s: on mains power, battery 98 %; CPU load over the 2 s before, 33, 26,
28, 26 and 26 %; the windows open, Chrome, Claude, Edge, Notepad, Task
Manager, Move Mouse, Settings, Microsoft Lightbox; Defender idle (0 s of
CPU). Plan 69 adds no timed test; on a frame's path it adds the home
screen's two notes, a resumed session's reasons and the usage section's
choice between its rows and its columns.

| Test | Measured | Target | Result |
| - | - | - | - |
| TestKeystrokeToFrame | 10,000-line draft p95 1.05 ms; in the draft window 1.11 ms | 16 ms | PASS |
| TestDraftKeystrokes | 50,000 lines, 9 saves, p95 3.18 ms | 16 ms | PASS |
| TestKeystrokeCost | reports only | | PASS |
| TestLongSession | a delta at the end, p95 3.00 ms | 16 ms | PASS |
| TestThinkingFrame | p95 2.58 ms | 16 ms | PASS |
| TestPasteToFrame | the worst 4.40 ms | 100 ms | PASS |
| TestStreamToFrame | p95 4.66 ms (the worst 17.2 ms) | 16 ms at p95 | PASS |
| TestSelectorToFrame | p95 1.01 to 1.53 ms | 16 ms | PASS |
| TestMotionToFrame | p95 0.72 to 1.52 ms; idle motion all dropped | 16 ms | PASS |
| TestDiffToFrame | p95 1.02 to 1.51 ms | 16 ms | PASS |
| TestMessageToFrame | p95 1.00 and 2.03 ms | 16 ms | PASS |
| TestOutputToFrame | p95 1.01 ms | 16 ms | PASS |
| TestToolsToFrame | p95 2.49 ms | 16 ms | PASS |
| TestBannerToFrame | p95 1.767 ms, max 3.48 ms, 11,355 bytes a frame | 2 ms | PASS |
| TestHomeKeysToFrame | p95 0.18 ms | 16 ms | PASS |
| TestHomePictureTime | the worst 192 ms | 260 ms | PASS |
| TestBranchesFrameCost | p95 1.53 ms | 16 ms | PASS |
| TestBlameFrameCost | p95 1.37 ms | 16 ms | PASS |
| TestGraphFrameCost | p95 1.00 ms | 16 ms | PASS |
| TestTurnReviewFrameCost | p95 1.24 ms | 16 ms | PASS |
| TestSearchToFrame | keys, two hours: p50 4.00, p95 6.06, max 9.06 ms; answers ten times longer: p50 7.30, p95 9.56, max 13.3 ms; deltas, the bar open: p50 2.99, p95 4.54, max 6.01 ms | 16 ms | PASS |

## Manual acceptance checklist (for the owner)

Asked by the owner on 2026-10-30, before the commits: plan 69 checked by
hand in Windows Terminal. **Quota: two Claude Code turns** (2.2 and 4.2,
on Haiku), Codex none (6.4, optional: one Codex turn); a resume, the home
screen and quitting make no model call. **Nothing is deleted or
reconfigured**: two new empty folders under `%TEMP%`; ACC adds the test
folder to its recent projects (backed up first, put back at the end if
wished); the test's Claude Code session stays in Claude Code's own history
like any other; `ANTHROPIC_MODEL` lives in one tab only; 8.4's font step is
Windows Terminal's own, reset with Ctrl+0. Each step: what to do, what to
see, PASS when it matches.

**1. Prerequisites** (no quota)
- 1.1 In any PowerShell tab:
  `New-Item -ItemType Directory -Force "$env:TEMP\acc-p69-manual", "$env:TEMP\acc-p69-other" | Out-Null`
  then `Copy-Item "$env:LOCALAPPDATA\agent-command-center\projects.json" "$env:TEMP\projects.json.before-p69"`.
  PASS: no error.
- 1.2 Maximise Windows Terminal; open a tab with the profile "Agent Command
  Center" (˅ beside +): "Agent Command Center: type acc in a project folder
  to start the dashboard." Run `$Host.UI.RawUI.WindowSize`. PASS: about
  208 x 59 (picture mode needs 118 x 19 at least).
- 1.3 In that tab: `Set-Location "$env:TEMP\acc-p69-manual"`, then
  `$env:ANTHROPIC_MODEL = 'haiku'`, then
  `& C:\javadev\projets\agent-command-center\acc.cmd` (`go run` of the
  working tree under review, in this tab, its project this folder). See:
  the home screen as a picture (the banner), "Claude Code (2.1.303)"
  selected, "OpenAI Codex (0.163.0)", the project field ending in
  `acc-p69-manual`. PASS: all of it, and **no** "last session here" note.

**2. A resumable session** (1 Claude Code turn)
- 2.1 Enter (START SESSION). See: zone B "Ready to work on this project.",
  zone A "Claude Haiku 5.5". PASS: Ready on Haiku.
- 2.2 Type `Remember this code word for later in this conversation: TEAL-69. Reply with the single word OK, and nothing else.`
  and Enter. PASS: "OK" answered. **1 turn.**
- 2.3 In another tab, read only:
  `(Get-Content "$env:LOCALAPPDATA\agent-command-center\projects.json" -Raw | ConvertFrom-Json).projects | Where-Object { $_.path -like '*\acc-p69-manual' } | ForEach-Object { $_.sessions.'claude-code' }`.
  PASS: an `id` and its `ended` time; note the id's first 8 characters.

**3. Full restart** (no quota)
- 3.1 In the ACC tab, Ctrl+Q, then Ctrl+Q again at "quit? …": back at
  the prompt. (Stricter: close Windows Terminal, reopen, redo 1.2 and
  1.3's first two lines.)
- 3.2 Run `& C:\javadev\projets\agent-command-center\acc.cmd` again.
  PASS: under the field, "→ Claude Code's last session here: now · Ctrl+R
  continues it" (or "1m ago" and so on).

**4. Ctrl+R** (1 Claude Code turn)
- 4.1 Ctrl+R. See: briefly "Resuming Claude Code session <8 characters>
  in …", then zone B "Resumed Claude Code session <8 characters> · Claude
  Code has its earlier context; its earlier messages are not shown here";
  zone A's STAGES without the earlier turn; SESSION USAGE as three rows,
  "Context N/A no turn since the resume", "Tokens N/A no result since the
  resume", "Est. cost N/A no result since the resume". PASS: the 8
  characters are 2.3's, no earlier message drawn, the three rows whole.
  No quota.
- 4.2 Type `What is the code word I asked you to remember earlier in this conversation? Reply with the code word only.`
  and Enter. PASS: "TEAL-69"; SESSION USAGE back in three columns with
  values, EST. COST a dollar figure. **1 turn.** Then Ctrl+Q twice.

**5. The mouse** (no quota)
- 5.1 Maximised: run acc again, click the note's text. PASS: the resume
  of 4.1 starts. Ctrl+Q twice.
- 5.2 120x40: in any PowerShell, `wt.exe -w new --size "90,30"` (a smaller
  window; move it on screen if it opens partly below). In it, an "Agent
  Command Center" tab; `$Host.UI.RawUI.WindowSize` should read 120 x 40
  (resize until it does if your default profile's font is not 12 pt).
  `Set-Location "$env:TEMP\acc-p69-manual"`, then acc.cmd as in 1.3. Click
  the note's text. PASS: the resume starts (the case plan 69 fixed at this
  size); zone A's rows cut with "…" (limitation 3). Ctrl+Q twice; keep the
  window for 8.4.

**6. Project and CLI isolation** (no quota)
- 6.1 Maximised, acc in `acc-p69-manual`, the note shown. ↓ (Tab to the
  Agent CLI first if needed): OpenAI Codex selected. PASS: the note gone,
  Ctrl+R does nothing; ↑ brings it back with Claude Code.
- 6.2 Tab to PROJECTS, ↑/↓ onto another project (the field takes its
  path; nothing starts). PASS: no note, Ctrl+R does nothing; back on
  `acc-p69-manual`, the note returns.
- 6.3 Ctrl+Q twice.
- 6.4 Optional, **1 Codex turn**: Codex selected, Enter, `Reply with the single word OK.`,
  Ctrl+Q twice; acc again. PASS: with Codex, "OpenAI Codex's last session
  here: …", with Claude Code its own note; Ctrl+R with Codex: "Resumed
  OpenAI Codex thread …", zone A's CONTEXT and TOKENS with values at once,
  EST. COST N/A "Codex reports none". Ctrl+Q twice.

**7. No resumable session** (no quota)
- 7.1 `Set-Location "$env:TEMP\acc-p69-other"` and acc.cmd. PASS: no note
  with Claude Code nor with Codex; Ctrl+R with each does nothing (no
  "Resuming…", no dashboard). Esc quits. Nothing is recorded: a project
  enters the list only when a session starts there.

**8. Visual checks** (no quota)
- 8.1 The home screen (5.1, 5.2): the note in grey after "→", under the
  field, above START SESSION. Limitation 1: the field box's bottom edge
  can reach into the note's first line (up to about 5 px; about 3.7 px at
  120x40 with one note). PASS: the note reads whole.
- 8.2 Zone B after a resume: the grey note, then only new turns. PASS.
- 8.3 Zone A after a resume: maximised, three rows with whole reasons
  until the first new turn's result; at 120x40 cut with "…"; F6 until zone
  A has the focus, ↓ onto TELEMETRY, Enter: each value with its reason
  whole; Esc. PASS: the reasons whole there.
- 8.4 Optional, the grid form (closes a "not verified" item): in 5.2's
  window, acc in `acc-p69-manual`, Ctrl+= once (Windows Terminal's font one
  step larger, about 108 columns). See: the panels redrawn in text
  characters (as the 80x24 golden frame), the note a text line under the
  field. Click it, or Ctrl+R. PASS: the resume of 4.1. Ctrl+Q twice, then
  Ctrl+0.

**Afterwards** (optional, no quota): with no ACC running,
`Copy-Item "$env:TEMP\projects.json.before-p69" "$env:LOCALAPPDATA\agent-command-center\projects.json" -Force`
puts the project list back as it was (if ACC was used for nothing else
meanwhile); or keep it, Del on the test project's row in PROJECTS forgets
it. The two test folders hold nothing of yours.

**Results:** the owner's run on 2026-10-30, both CLIs: every scenario PASS,
with two UI findings, P69-M01 and P69-M02, since fixed, and both confirmed
fixed by the owner's regression run; then the owner's final report: the
window's edge dragged, the images after a resize and a real Claude Code
refusal, PASS (see
[The owner's acceptance run](#the-owners-acceptance-run-2026-10-30)).
