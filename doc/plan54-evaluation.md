# Plan 54 evaluation (2026-10-27)

The check of plan 54, the mouse wheel over the project list (brief
amendment 55): over the home screen's PROJECTS panel the wheel moves the
list's selection one project a notch, as ↑ and ↓ do.

It covers the 4 criteria of [`plan.md`, Plan 54](../plan.md), written
before the code. Nothing is committed or pushed: the owner reviews first.

How it was checked:
- **State tests**: `TestHomeWheel` (the app, in picture and grid mode);
  `TestHomeTargets` and `TestHomeTargetsStacked` (the panel's cells in
  picture mode, grid mode and grid mode's stacked layout).
- **Windows Terminal** at 208x59, the acc tab (`scratchpad\term\p54wheel.ps1`,
  a `wheel` step added to `p51.ps1`): the real application on a temporary
  home and app data, the design's seven projects seeded, no session
  started.
- **Each change broken on purpose** through `go test -overlay`
  (`scratchpad\plan51\mutrun`, the lists in `scratchpad\plan54`), the
  repository never written.
- **Build, vet, gofmt, the whole suite** after the last code and document
  edit.

## The evaluation

```text
Evaluation - Plan 54, the mouse wheel over the project list - 4 criteria: 4 PASS, 0 FAIL
PASS 54.1 The wheel over the list: TestHomeWheel, picture (208x59) and grid (80x24): a notch down over the first row selects the second project, down over its star the third, up over the panel's other cells (its title, headings, frame) the second, each with that project's path in the field and the list focused; up past the first and down past the last stay there; TestHomeTargets: the list's cells right of NEW SESSION's and on rows with no project (the headings) in picture and grid mode; TestHomeTargetsStacked: also under START SESSION in grid mode's stacked layout
PASS 54.2 Elsewhere nothing: TestHomeWheel: the wheel over the project field and over the footer, and a click on the panel's other cells, leave the selection, the focus and the field's text as they were; the golden frames unchanged (TestGoldenFrames without -update)
PASS 54.3 Windows Terminal at 208x59: the field C:\Users\…\Temp\p54; a notch down over the list: C:\Projects\personal-assistant; another: C:\Projects\code-workflow; one up: C:\Projects\personal-assistant, the band on it (shot): one project a notch, each notch one wheel event
PASS 54.4 The checks: go build ./... ok; go vet ./... ok; gofmt -l . empty; go test ./... ok; 11 mutants through -overlay, 11 killed by a failing test (W4 written again as W4b, its first form not compiling; W11, the stacked layout's list item, killed once TestHomeTargetsStacked was added), the tree's hash the same around each run
Verdict: DONE
```

## Details

### One project a notch

The dashboard's Explorer and Source Control move their cursor three rows
a notch (`wheelRows`); here each step chooses a project and puts its path
in the field (plan 53), so a notch moves one, as ↓ does. Windows Terminal
sends one wheel report a notch: the run's three notches moved the
selection three times by one.

### The panel's runs

In picture mode the cells under the picture are runs of targets
(`homeTargetRow`). The PROJECTS panel's target lies under its rows' and
stars', so a run now ends where an earlier target begins; before that
change the panel's run covered the rows (`TestHomeTargets` failed with no
cell for a row), and mutant W8 puts it back.

### The mutants

| Id | The change | Caught by |
| - | - | - |
| W1 | the wheel ignoring the panel's other cells | TestHomeWheel |
| W2 | the list not focused by the wheel | TestHomeWheel |
| W3 | three projects a notch | TestHomeWheel |
| W4b | the wheel acting anywhere on the home screen | TestHomeWheel |
| W5 | the wheel not handled | TestHomeWheel |
| W6 | no panel target in picture mode | TestHomeTargets |
| W7 | no list item in grid mode | TestHomeTargets |
| W8 | a run not ending where a row begins | TestHomeTargets |
| W9 | up and down the same way | TestHomeWheel |
| W10 | the list's item over the rows' and stars' | TestHomeTargets |
| W11 | no list item in the stacked layout | TestHomeTargetsStacked |

**The commit and the push (2026-10-27), after the owner's approval.** The
owner approved plan 54 on 2026-10-27 and asked for the push. Commit
38ee6d7 (13 files: 12 modified, 1 added) was pushed to `origin`, `main`
fast-forwarded from efe42ca, nothing forced; the approval commit
recording this was pushed after it. The author is `Evengelius
<molimoabel@gmail.com>`, with no attribution line, and the working tree is
clean: PASS. The block above stays as it was reported.
