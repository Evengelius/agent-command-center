# Plan 53 evaluation (2026-10-27)

The check of plan 53, a project chosen in the list fills the field (brief
amendment 54): choosing a project in the home screen's PROJECTS list puts
its path into the Project directory field, so that the field shows what
Start uses.

It covers the 5 criteria of [`plan.md`, Plan 53](../plan.md), written
before the code. Nothing is committed or pushed: the owner reviews first.

How it was checked:
- **State tests**: `TestProjectChosen` (the keys), `TestHomeMouse` (the
  clicks, in picture and grid mode), `TestProjectListKeys` (Enter on the
  list, as before).
- **Golden frames**: the six that move the selection rewritten, each
  reviewed as a diff.
- **Windows Terminal** at 208x59, the acc tab (`scratchpad\term\p53list.ps1`):
  the real application on a temporary home and app data, the design's
  seven projects seeded, no session started.
- **Each change broken on purpose** through `go test -overlay`
  (`scratchpad\plan51\mutrun`, the list in `scratchpad\plan53`), the
  repository never written.
- **Build, vet, gofmt, the whole suite** after the last code and document
  edit.

## The evaluation

```text
Evaluation - Plan 53, a project chosen in the list fills the field - 5 criteria: 5 PASS, 0 FAIL
PASS 53.1 Choosing: TestProjectChosen: ↓ in the list puts the second project's path in the field, its resolved path the same, the list keeping the focus; ↑ the first's; TestHomeMouse, picture (208x59) and grid (80x24): a click on the third row puts its path in the field, the list focused; a click on its star, the field set elsewhere before, puts that project's path there
PASS 53.2 Not choosing: TestProjectChosen: Tab onto the list leaves the starting folder in the field; f, the field set elsewhere, leaves it; Del leaves the forgotten project's path, the selection on the next row; Enter on the list starts with the selected row's path, the field holding it (also TestProjectListKeys, unchanged)
PASS 53.3 Golden frames: the six that move the selection by ↓ changed one line each, the field's text, now the selected row's path: C:\javadev\projets\scratch in startup-favourites (row 1) and C:\javadev\projets\web in startup-projects-focused (row 6), at 80x24, 120x40 and 200x60, each reviewed against the row marked selected; no other frame changed
PASS 53.4 Windows Terminal at 208x59: the field C:\Users\…\Temp\p53; a click on the second project's row: C:\Projects\personal-assistant in the field (its terminal text); ↓: C:\Projects\code-workflow, the band on code-workflow (shot)
PASS 53.5 The checks: go build ./... ok; go vet ./... ok; gofmt -l . empty; go test ./... ok; 8 mutants through -overlay, 8 killed by a failing test (P1 to P8: no fill on ↓ or a click, an empty chooseProject, Enter's fill removed, a fill on Tab, on Del, on f, the first row's path for any row), the tree's hash the same before and after
Verdict: DONE
```

## Details

`TestProjectChosen` first missed one mutant in writing: after ↓ the field
already held the selected row's path, so a fill on f would have changed
nothing. The test sets the field elsewhere before f; P7 is caught by it.
The race detector was not run: the change adds no goroutine and no shared
state, only a field's text set in the update loop.
