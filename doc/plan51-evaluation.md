# Plan 51 evaluation (2026-10-27)

The check of plan 51, the home screen redesign (brief amendment 52): the
startup screen drawn as the owner's image, `references/16-home-screen.webp`,
with an animated Space Invaders banner, its functions kept.

It covers the 11 criteria of [`plan.md`, Plan 51](../plan.md), written
before the code. Nothing is committed or pushed: the owner reviews first.

How it was checked:
- **Picture fidelity** (51.1): Windows Terminal screenshots at 208x59 (the
  acc tab, 9 pt), 150x40 and 120x40 (the default profile) measured against
  the design scaled by k (`scratchpad\plan51\fid`): the screenshot's
  origin and the terminal's resampling fitted on the panels' four outer
  edges, then 32 other edges, the project rows, three texts' advances and
  seven flat colours measured where the design puts them.
- **The banner's scene and view** by deterministic tests: a fixed seed
  over 1,000 frames (`TestBannerScene`), and the terminal simulated from
  the frames' Sixel images (`TestBannerFrames`).
- **Each change broken on purpose** through `go test -overlay`
  (`scratchpad\plan51\mutrun`), the repository never written: the hash
  of `git diff` and the untracked files the same before and after.
- **Build, vet, gofmt, the whole suite** after the last code and document
  edit; the race detector on `internal/app` and `internal/render`; the
  timing step alone.
- **Windows Terminal**: the banner's cost sampled over 20 s with and
  without the focus (`scratchpad\term\p51banner.ps1`, `p51cpu.ps1`), the
  seven sizes of D2, and the end-to-end run (`p51e2e.ps1`): the real
  application on a temporary home and app data, a temporary project, the
  desktop session's `CLAUDE_` and `ANTHROPIC_` variables cleared, no
  message sent, `confcheck` before and after.

## The evaluation

```text
Evaluation - Plan 51, the home screen redesign - 11 criteria: 10 PASS, 1 FAIL
FAIL 51.1 Picture fidelity: expected every edge within 2 screen pixels of the design scaled by k, the resampling apart; found 31 of 33 edges a size within 1.91 (208x59 with the probe's resampling, 0.9 x 0.85: 1.71; 150x40: 1.91; 120x40: 1.60, the resampling fitted, 1.199 x 1.145), and the project field's box low: its top 3.63 and 6.50 pixels at 208x59 and 120x40, its bottom 1.73 and 5.43 (at 150x40 within 1.6); text advances within 0.6% of the design's (NEW SESSION, PROJECTS, the title, Agent CLI); flat colours within 1 per channel (the page, the chosen option, the field, START SESSION, the band, a key cap) and the footer's line within 2; the seven project rows' ascender tops within 1 pixel of the design's spacing at the three sizes; every element of the design present, the departures listed in reference-design.md
PASS 51.2 TestHomeLayout's table: 208x59 picture, banner 26 rows; 150x40, 16; 120x40, 20; 120x30, 10; 120x26 picture without one; 100x40 grid, 22; 80x24 grid without one; Windows Terminal at the seven sizes shows each so (the banner running at 208x59, 150x40, 120x40, 120x30 and 100x40)
PASS 51.3 TestHomeBanner: at 120x26 and 80x24 the scene's view is dropped (no background copy, sprites, palette or scene kept), a pending frame writes nothing and asks for no other, and no frame is asked for; mutants M13 and M14 caught
PASS 51.4 the whole suite passes; the expectations changed are drawn text the design changes: "START SESSION →" for "Start session" and its place above the footer (golden_test), ages "now", "1m ago", "1h ago", "1d ago" (projects_test), the footer's "Ctrl+Q  Quit" and its width at 80 columns (codexproxy_test, worktrees_test), the messages read in NEW SESSION wherever they wrap (golden_test, validate_test, codexproxy_test), the 18 golden frames of the startup screen; the cases added are Ctrl+N's (machines_test, TestHomeCtrlN) and the glyphs' (TestGlyph); every key as before; the field's editing and paste code unchanged, its text terminal text (TestHomeSequence: the picture leaves its cells out), selected by Windows Terminal (51.9)
PASS 51.5 TestHomeMouse, picture and grid: an option chooses its CLI (not one not installed), the field takes the focus, the folder button opens the browser (not while validating), a row is selected, a star toggles the favourite, START SESSION starts; a double click, the wheel and the footer do nothing; F12 gives the mouse back; in Windows Terminal the field, a row, a star, an option and START SESSION by the mouse (step 4, and 51.9)
PASS 51.6 TestBannerScene, seed 51, 1,000 frames of a 1347x324 scene: aliens appeared in each eighth of the width; each on screen 50 frames or more stepped sideways and descended; 32 aliens and waiting ones all along, never two within 4 pixels; 139 appeared in all (replaced); the ship moved left, right, up and down; 100 bolts, in pairs a frame apart; 38 hits, 38 explosions
PASS 51.7 the timing step: TestHomeKeysToFrame, a key in the field with the banner running, p95 0.15 ms (target 16 ms); TestBannerToFrame, a banner frame from its message through Update to its frame, p50 0.45 ms, p95 1.27 ms, max 2.94 ms (target 2 ms), 8,253 bytes; TestHomePictureTime, the picture on entering, the 120 ms wait included, 193 ms at worst, 233 ms with the terminal's 40 (target 300); Windows Terminal at 208x59 while the banner runs, 23 s: 6.6% of a core (WindowsTerminal 4.99%, its OpenConsole 1.62%; target 8%), without the focus 0.7%
PASS 51.8 the golden frames at 79x24 (the too-small screen, unchanged), 80x24, 120x40 and 200x60, and the six message frames, rewritten and each reviewed as a diff in step 3; TestHomeGrid, TestHomeGridMessages; TestHomeGridFocus in 16 colours: no ground, START SESSION's label in the button's colour
PASS 51.9 in Windows Terminal, maximised then at 120x40: OpenAI Codex chosen by a click, then Ctrl+N: Claude Code chosen again, the starting folder in the field, the focus on Agent CLI; Enter started a session (the dashboard, Claude Code 2.1.303, idle); in a second run the field's path selected by the terminal (Shift and a drag) and a click on START SESSION started one; confcheck PASS (28 files identical, no new session file); the clipboard the same before and after (its formats and hash) - the path's copy and paste back not verified (below)
PASS 51.10 23 mutants through -overlay: 22 caught (the scene: the aliens' room, the ship's way up, the second bolt, the explosion, the replacement, the width; the view: the old place and a gone sprite left undrawn, the unchanged pixels sent, the runs' rows, the cells within the banner; the encoder's band left uncleared; the application: frames and their timer without the focus, a hidden banner's view kept, the second drawing without the scene, the banner in HomeSixel, no focus reports, the scene dropped at a size report, a grown banner not rewritten, an option's click, Ctrl+N), 1 equivalent (M23: a reset while Detecting, which holds no answer); TestBannerScene's overlap check rewritten with its own arithmetic after M1 survived it, the test having called the function under test
PASS 51.11 go build ./... ok; go vet ./... (also -tags timing and -tags live) ok; gofmt -l . empty; go test ./... ok, 2,435 tests; -race on internal/app (362 s) and internal/render (163 s) ok; the timing step alone, on mains, CPU 25% before (the owner's Task Manager open), 12 logical processors: 16 PASS; go.mod and go.sum unchanged; no CR or BOM in the files changed; AGENTS.md 20,706 bytes, TestAgentsMDWithinCodexLimit passes
Not verified: 51.9's copy of the path and its paste back. The harness that saved and restored the owner's clipboard around them was quarantined by the antivirus as it ran (the clipboard was not touched); copying without restoring would overwrite the owner's clipboard (an image). The harness was rebuilt without any clipboard code. The paste goes through the field's unchanged code (app.go, paste: the field takes input.CleanPath of the text).
Verdict: NOT DONE
Likely cause: 51.1: D4 puts the field's box on the terminal row nearest the design's, centred on it, so the box moves by up to half a row: 3.6 pixels at 208x59, 6.5 at 120x40.
Proposed next step: 51.1: accept it as D4's consequence, a documented exception (the field's text kept in the middle of its box), or draw the box at the design's place with the text off its middle - waiting for approval. 51.9's paste back: the owner selects the path with Shift and a drag, copies it, empties the field and pastes it (half a minute), or approves another way to protect the clipboard.
```

**The owner's decision on 51.1 (2026-10-27).** The owner accepted 51.1 as
a documented exception: the project field's box stays on the terminal row
nearest the design's, centred on it, the path kept in its middle (D4). It
stays FAIL above, as reported.

**51.9's paste back, checked by the owner by hand (2026-10-27).** The
owner started the dashboard with `.\acc` from the repository in Windows
Terminal, selected the field's path, `C:\javadev\projets\agent-command-center`,
with Shift and a drag, copied it with Ctrl+C, emptied the field with
Backspace and pasted it with Ctrl+V: the same path came back exactly. The
"Not verified" above stays as it was reported.

## After the evaluation: the owner's pace

The owner found the animation too slow. `render/banner.go` now marches the
aliens of a band of rows every 0.4 s, two bands in turn (0.6 s, three);
they drift 10 to 26 design pixels a second (4 to 14) and descend 4 to 10
(2 to 6); the ship wanders at 70 to 150 (50 to 120); its gun reloads in
1.1 to 1.9 s (1.4 to 2.4) and fires unaimed one time in five (six); the
bolts fly 420 a second (360); an explosion lasts 0.55 s (0.75), its
particles faster (30 to 100 a second) and shorter-lived (0.3 to 0.55 s).
The pace was chosen among five run through overlays for the rows a frame
touches (`scratchpad\plan51\pacerun`): 11.8 rows a frame, from 10.4; the
same livelier aliens with a 0.4 s march and the old explosions 12.6, a
march every frame 13.2, 12.5 frames a second 14.5 (rows a frame at 10
frames a second).

```text
Evaluation - Plan 51, the banner's pace after the owner's remark - 2 criteria: 1 PASS, 1 FAIL
PASS 51.6 TestBannerScene, seed 51, 1,000 frames: aliens in each eighth of the width; each on screen 50 frames or more stepped sideways and descended; 32 aliens and waiting ones all along, never two within 4 pixels; 211 appeared in all; the ship moved left, right, up and down; 122 bolts in pairs a frame apart; 59 hits, 59 explosions; TestBannerFrames and TestBannerKept pass (241 cells and 11,149 bytes a frame)
FAIL 51.7 the application's part: TestBannerToFrame p50 0.68 ms, p95 1.66 ms, max 3.01 ms (target 2 ms), 11,227 bytes a frame; TestHomeKeysToFrame p95 0.18 ms (target 16 ms). Expected the terminal at most 8% of a core while the banner runs; found, Windows Terminal and its OpenConsole at 208x59, seven samples: 8.37, 8.15, 7.13, 7.76 and 5.68% over 23 s, 8.87 and 8.28% over 60 s, mean 7.75%, four over 8%; the old pace between them, the same hours: 5.96 and 7.39% over 23 s, 8.32% over 60 s (6.6% at the evaluation)
Verdict: NOT DONE
Likely cause: the livelier pace takes about 13% more rows a frame (the model) and the samples 10% more terminal time (the means); the machine's own load, swinging from 1 to 74% within seconds, moves a sample by up to 2.5 points, so that the old pace too measured 8.32% in these hours.
Proposed next step: keep the pace and accept the terminal at about 7.5 to 8% (a documented exception), or take back some of the action for margin (the shots as before: 11.0 rows a frame, about 7%), or measure again on a quiet machine - waiting for approval.
```

The owner then asked for the ship faster again, not drastically. A dash
of 520 to 720 design pixels a second with afterimages, tried at the
owner's request just before, was withdrawn at the owner's request, and
nothing of it remains. The ship's legs now go at 100 to 220 design pixels
a second (70 to 150), each leg's pace still drawn at random, slow to fast;
the aliens, shots and explosions keep the pace above.

```text
Evaluation - Plan 51, the ship's pace after the owner's second remark - 4 criteria: 4 PASS, 0 FAIL
PASS 1  The dash withdrawn: banner.go, bannerview.go and banner_test.go rebuilt by reversing the session's edits since the dash request; banner.go is the pace above (pacerun's V5) but for one comment; TestBannerScene again 211 appeared, 122 bolts, 59 hits, and TestBannerFrames 241 cells and 11,149 bytes a frame, as above
PASS 2  The ship's legs at 100 to 220 design pixels a second (render/banner.go, moveShip); in Windows Terminal at 208x59, 30 shots about 0.4 s apart, a leg crossed 654 screen pixels in 3.5 s (about 130 design pixels a second), no afterimage and nothing left behind
PASS 3  51.6 TestBannerScene, seed 51, 1,000 frames, passes: 217 appeared in all, the ship moved left, right, up and down, 128 bolts in pairs a frame apart, 68 hits and 68 explosions; TestBannerFrames (250 cells and 11,317 bytes a frame) and TestBannerKept pass; 11.8 rows a frame in pacerun's model, as before
PASS 4  51.7 the application's part: TestBannerToFrame p50 0.69 ms, p95 1.60 ms, max 3.25 ms (target 2 ms), 11,290 bytes a frame; go build, go vet, gofmt -l and go test ./... pass
Not verified: Windows Terminal's share of a core at this pace, not sampled again (a sample swings by up to 2.5 points with the machine's load; the rows a frame, which its cost follows, are unchanged), so the 51.7 FAIL above stands for the owner's decision
Verdict: DONE
```

The owner asked for the ship a little faster still, not drastically: its
legs now go at 125 to 275 design pixels a second (100 to 220).

```text
Evaluation - Plan 51, the ship a little faster after the owner's third remark - 4 criteria: 3 PASS, 1 FAIL
PASS 1  The ship's legs at 125 to 275 design pixels a second (render/banner.go, moveShip); in Windows Terminal at 208x59, 30 shots about 0.4 s apart, a leg crossed 715 screen pixels in 3.1 s (about 160 design pixels a second), nothing left behind
PASS 2  51.6 TestBannerScene, seed 51, 1,000 frames, passes: 209 appeared in all, the ship moved left, right, up and down, 128 bolts in pairs a frame apart, 68 hits and 68 explosions; TestBannerFrames (245 cells and 11,165 bytes a frame) and TestBannerKept pass; 11.6 rows a frame in pacerun's model (11.8 before)
FAIL 3  51.7 the application's part: expected TestBannerToFrame p95 within 2 ms; found p50 0.74 ms, p95 2.05 ms, max 4.67 ms, 11,355 bytes a frame (on mains; the CPU 3 to 25% busy over 12 logical processors; Task Manager, Edge, Move Mouse, Teams, VS Code and the Claude app open)
PASS 4  go build, go vet, gofmt -l and go test ./... pass
Not verified: Windows Terminal's share of a core at this pace, not sampled again (the rows a frame unchanged)
Verdict: NOT DONE
Likely cause: the machine's load, not the pace: timed in turn right after, this pace's p95 1.92, 1.89, 1.66 and 2.21 ms, the previous pace's (1.60 ms an hour earlier) 1.84, 1.82, 1.94 and 2.70 ms, the bytes a frame within 0.6%; Task Manager took 45% of a core, Edge 39%, Move Mouse 20%.
Proposed next step: run the timing step again with the owner's applications idle (Task Manager closed), or accept this miss as the machine's load - waiting for approval.
```

The owner closed Task Manager and asked for the timing step again. It ran
alone, all sixteen tests, on 2026-10-27 from 06:47 to 06:48: on mains,
the CPU 5 to 28% busy over 12 logical processors just before (a burst of
Teams' renderer, 1.3 cores, had settled to nothing), Claude, VS Code, Move
Mouse, Teams, Ranorex's report viewer and Settings open and idle.

```text
Evaluation - Plan 51, the timing step again with the owner's applications idle - 1 criterion: 1 PASS, 0 FAIL
PASS 51.7 the application's part: TestBannerToFrame p50 0.69 ms, p95 1.76 ms, max 2.80 ms (target 2 ms), 11,355 bytes a frame; TestHomeKeysToFrame p95 0.12 ms (target 16 ms); TestHomePictureTime worst 200 ms (target 260 ms); the step's thirteen other tests pass
Not verified: Windows Terminal's share of a core at this pace; the terminal's part of 51.7 stays as the first block of this addendum found it, for the owner's decision
Verdict: DONE
```

The owner's applications still idle, Windows Terminal was sampled at the
final pace (2026-10-27, 07:01 to 07:08; `term\p51cpuseq.ps1`): five runs
of 60 s in turn, the final pace and the evaluation's build (the first
pace, 6.6% at the evaluation), each in a Windows Terminal of its own at
208x59, the home screen and the banner running from the first shot to the
last; on mains, the machine about 20% busy just before each run (0 to 54%,
12 logical processors).

```text
Evaluation - Plan 51, the terminal's share at the final pace with the owner's applications idle - 1 criterion: 1 PASS, 0 FAIL
PASS 51.7 the terminal's part: expected Windows Terminal and its OpenConsole at most 8% of a core while the banner runs; found over 60 s at the final pace 7.54, 7.46 and 7.70% (mean 7.57%), the evaluation's build between them 7.16 and 7.04% (mean 7.10%)
Verdict: DONE
```

## Details

### 51.1, the measurement

`fid shot.png cols rows`: the design's place on screen is the picture's
(`OX + x·k`; y from the top for the header, from the bottom below it), the
terminal's resampling of 10x20 image pixels to its cell, and the
screenshot's origin. The panels' outer sides and their top and bottom fix
the origin and the resampling; a thin line is found by its prominence over
both sides three pixels away, an edge between two fills where the luma
crosses midway. At 208x59 the fitted resampling, 0.8993 x 0.8471, is the
probe's 0.9 x 0.85 within 0.1%; with the probe's exactly the header's
bars land within 0.04 to 0.90 pixels (the fitted scale put the third at
2.29). The rows are compared by their names' ascender tops, the names' ink
middles moving with their descenders.

| Size | Edges (worst) | Field's box, top / bottom | Advances | Colours |
| - | - | - | - | - |
| 208x59 | 1.71 | +3.63 / +1.73 | within 0.1% | within 1 |
| 150x40 | 1.91 | -0.69 / -1.57 | within 0.3% | within 1 |
| 120x40 | 1.60 | +6.50 / +5.43 | within 0.6% | within 1 (the field's sample on its text) |

### 51.7, the banner's cost

At 12.5 frames a second with each alien on its own clock, Windows Terminal
took 10.9% of a core (WindowsTerminal 9.7%, its OpenConsole 1.2%), a frame
touching 14.4 of the banner's 25 rows on average; its cost follows the rows
an image touches more than its bytes (the probe of step 1: a two-row
change 0.6 to 1.6 ms). At 10 frames a second, D12's least, with the aliens
marching by bands of rows, 10.4 rows a frame: 6.6%. The application took
4.6% of a core with the banner and 3.0% on the same screen without it
(below). The first frames took 4 ms at p50 and sent 40 KB; three changes
brought them to 0.45 ms and 8 KB: the pixels a frame did not change sent
transparent, the sprites and the explosion in few colours, and the Sixel
encoder reading each band once (its output byte for byte the old one's on
3,000 random images).

### 51.9, the run

`p51e2e.ps1`, two runs a size, on `%TEMP%\p51`: `keys` (Codex's option
clicked, Ctrl+N, the path checked on screen, Enter) and `mouse` (the
field's path selected with Shift and a drag, then START SESSION clicked);
each ends at "CONVERSATION" (the dashboard) and quits with Ctrl+Q.
Windows Terminal's `copyOnSelect` is off, so the selection left the
clipboard alone, and no Enter followed it (Windows Terminal copies a
selection on Enter).

### Outside the plan

- The application takes 3.0% of a core on a screen with nothing moving
  (the home screen at 120x26, no banner); its profile shows Bubble Tea's
  input reader (`TerminalReader.streamData`) and its system calls. This
  is not plan 51's, and every screen has it.
- The antivirus quarantined the harness script when it gained the
  clipboard's save and restore beside its simulated input; the owner may
  see its notice. The script was rebuilt without them, from `p50.ps1`.

**The commit (2026-10-27), after the owner's approval.** The owner
approved plan 51 on 2026-10-27. Commit 7e60f73 (60 files: 46 modified, 14
added), then the approval commit; the working tree is clean, and the
author is `Evengelius <molimoabel@gmail.com>`, with no attribution line:
PASS. Not pushed: the owner asked for the commit. The blocks above stay
as they were reported.
