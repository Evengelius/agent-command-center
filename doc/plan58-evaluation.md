# Plan 58 evaluation (2026-10-28)

The check of plan 58, a click outside the window closes it (brief
amendment 60): the window floating over the dashboard, the file viewer or
the quick open, closes on a left click outside it, as it does on Esc, and
the click does nothing else.

It covers the 5 criteria of [`plan.md`, Plan 58](../plan.md), written
before the code. Nothing is committed or pushed: the owner reviews first.

How it was checked:
- **Application tests**: `TestFileClickOutside` (new), each closing run
  twice, in two rigs: closed with Esc, and with a click at one of four
  cells around the window; the two interfaces left must be the same,
  field for field, the rig's own project folder put aside;
  `TestFileMouse` (plan 17's criterion 3.7, its click behind the window a
  right click now); `TestLayerMatrix` (plan 49), its 19 cells the new rule
  moves reviewed and written again.
- **Each change broken on purpose** through `go test -overlay`
  (`scratchpad\plan51\mutrun`, the list built from the source lines by
  `scratchpad\plan58\mkmutants.mjs`), the repository never written: a hash
  of `git diff` and `git status` the same before and after.
- **Windows Terminal**: the real application in the acc tab, maximised
  (208x59) and at 120x40, on plan 55's demo project, a mock dashboard, a
  temporary home and app data, no CLI session (`scratchpad\term\p58click.ps1`
  on `p51.ps1`).
- **Build, vet, gofmt, the whole suite** after the last code and document
  edit.

## The evaluation

```text
Evaluation - Plan 58, a click outside the window closes it - 5 criteria: 5 PASS, 0 FAIL
PASS 58.1 The viewer: TestFileClickOutside, a left click on the title bar (0, 0), left of the window, right of it and under it closes the viewer and leaves the interface Esc leaves, field for field (the Explorer's panel open with the focus, its cursor on src/Main.java, src open; no selection; nothing else changed); full screen, a click on the first row closes it, the Explorer's cursor on main.go, the focus R
PASS 58.2 Inside, and the rest: a click on the window's frame, its footer and the file's text leaves it open, nothing selected, the focus kept; a right click, a middle click, a motion and a release outside change nothing; the wheel outside scrolls the window 3 lines; a drag on the text released outside copies ("// line 3\n// line 4\n..."); full screen, a click on the window's top row leaves it open; the window recorded is the one drawn (its title's corner and the opposite one in the frame)
PASS 58.3 The quick open: a click at the same four cells closes it as Esc does (nothing open); over the viewer, the quick open closes and the viewer, src/Main.java, is in front again
PASS 58.4 Windows Terminal, at 208x59 and at 120x40: F3, then two clicks on acc.cmd in the Explorer open it in the viewer ("acc.cmd · Explorer", its footer "Esc back · F3 close"); a click on zone A's title, outside the window, closes it: the window's path row and footer gone, the Explorer's panel focused with its cursor on acc.cmd and the Explorer's keys in the footer (shots out-p58\click-max\e6.png, click-90x30\e6.png), nothing selected or changed in zone A
PASS 58.5 The checks: go build ./... ok; go vet ./... ok (Windows and Linux); gofmt -l . empty; go test ./... ok (16 packages; TestFileClickOutside, TestFileMouse, TestLayerMatrix and TestGoldenFrames passing); no golden frame or mock file changed; 12 mutants through -overlay, 12 killed by a failing test, none by a build failure, the tree's hash the same around the run (4b14d637); the test that reads ui-design.md, TestKeyOverlayRows, passing (TestDocFrames, which the criterion names, only writes frames where ACC_DOC_FRAMES is set and is skipped otherwise)
Verdict: DONE
```

## Details

### The click

`fileMouse` takes the mouse while the window is the top layer (plan 49,
design D7). A left click whose cell lies outside the window's rectangle
closes it: the quick open by `closeFinder`, the viewer behind it in front
again; else the viewer by `backToExplorer`, Esc's last step. The click goes
no further. The rectangle is the one the frame drew last, `Screen.Window`,
set where the window is drawn (`floatWindow`), so a click is judged against
what was on screen even when the model has moved on since that frame (a
key toggling full screen just before). Every other mouse message keeps its
rule.

### The layer matrix

`TestLayerMatrix` opens each pair of layers and records what Esc and a
click at the top-left cell leave. With the window on top at that click, 16
cells now close it (`click=` in place of `click=finder` or `click=viewer`):
finder, finder>draft, finder>dropdown, finder>models, finder>modes,
finder>plus, finder>question, models>finder, modes>finder, plus>finder,
plus>viewer, viewer, viewer>draft, viewer>models, viewer>modes and
viewer>plus. In three more the second layer's opener is a click outside
the viewer, which now closes it and does nothing else: viewer>dropdown and
viewer>question leave nothing open, and in viewer>finder the opener's `/`
then falls on the Explorer, which opens the quick open. The 19 cells were
recorded (`ACC_LAYERS_RECORD`), diffed against the expectations and
written again; the other 81 are unchanged.

### The mutants

| Id | The change | Caught by |
| - | - | - |
| M1 | any button outside closes it | TestFileMouse (its right click) |
| M2 | a click inside closes it, outside not | TestFileMouse |
| M3 | the quick open goes back to the Explorer | TestFileClickOutside |
| M4 | the quick open closes the viewer too | TestFileClickOutside |
| M5 | the viewer closed without going back | TestFileClickOutside |
| M6 | the column right of the window inside | TestFileClickOutside |
| M7 | the row under the window inside | TestFileClickOutside |
| M8 | the window's first column outside | TestFileClickOutside |
| M9 | the window's first row outside | TestFileClickOutside |
| M10 | the window not recorded | TestFileMouse |
| M11 | the window recorded a row short | TestFileClickOutside |
| M12 | full screen recorded as the window | TestFileClickOutside |

M11 led to the check that the window recorded is the one drawn: the test
takes its cells from the recorded rectangle, so before it a rectangle
recorded wrong would have moved the cells with it.

### The Windows Terminal run

The owner's Windows Terminal (started 23:47:53 on 2026-10-27, the acc
profile naming Cascadia Mono NF) hosted the harness's windows; the owner's
own dashboard, with a live Claude session in its acc tab, was left as it
was, the runs being mock-only on a temporary home and app data. The
dashboard's footer shows the Explorer's keys before the click as after it,
the viewer opened from the Explorer keeping the panel's focus; the
window's own path row and footer tell open from closed.

**Approved and committed (2026-10-28).** The owner wrote "Plan 58 is
approved: review git status and the diff, then commit it." Commit 0408228
(11 files: 10 modified, 1 added), from the files reviewed in `git diff`;
the whole suite had passed after the last code and document edit that a
test reads. The author is `Evengelius <molimoabel@gmail.com>`, with no
attribution line, and the working tree was clean after it: PASS. Not
pushed: the owner asked for the commit only. The block above stays as it
was reported.

**The push (2026-10-28), at the owner's request.** Commit 0408228 and the
approval commit, 916a1e3, were pushed to `origin`: `main` fast-forwarded
from 585a485 to 916a1e3, nothing forced; `origin/main` equals `main`
(read back by `git ls-remote`), the working tree clean: PASS. The commit
recording this was pushed after it.
