# Plan 52 evaluation (2026-10-27)

The check of plan 52, the system's folder dialog for the project (brief
amendment 53): the home screen's folder button and Ctrl+O open the Windows
dialog for choosing a folder, and plan 15's project browser is removed.

It covers the 5 criteria of [`plan.md`, Plan 52](../plan.md), written
before the code. Nothing is committed or pushed: the owner reviews first.

How it was checked:
- **State tests** with a stand-in dialog that records where each one
  opened and gives its answers (`TestFolderDialog`, `TestFolderStart`,
  `TestHomeMouse` in picture and grid mode).
- **The real dialog**: a Windows test that creates and sets it up through
  COM without showing it; Windows Terminal at 208x59, the acc tab
  (`scratchpad\term\p52dialog.ps1`): the real application on a temporary
  home and app data, the design's seven projects seeded, no session
  started.
- **Each change broken on purpose** through `go test -overlay`
  (`scratchpad\plan51\mutrun`, the list in `scratchpad\plan52`), the
  repository never written.
- **Build, vet (also for Linux), gofmt, the whole suite** after the last
  code and document edit; the race detector on `internal/app` and
  `internal/input`.

## The evaluation

```text
Evaluation - Plan 52, the system's folder dialog for the project - 5 criteria: 5 PASS, 0 FAIL
PASS 52.1 Opening: TestHomeMouse, picture (208x59) and grid (80x24): a click on the folder button opens the dialog; TestFolderDialog: Ctrl+O opens it at the field's directory (the starting folder, then the folder chosen); a second Ctrl+O while one is open opens none (6 dialogs, not 7); none while a start is validated; TestFolderStart: a directory, else its nearest existing parent (a child missing two levels down, a file), else the starting folder ("" and a drive that does not exist)
PASS 52.2 The answer: the folder chosen in the field, the field focused, its resolved path the folder's; cancelled: the field and the focus unchanged; a failure: "✗ Folder dialog: broken" on the screen, the field unchanged, the note gone with the next dialog and with Ctrl+N; with no dialog, "✗ No folder dialog in this run"
PASS 52.3 The real dialog: TestFolderDialog (input) creates it through COM and reads back its options, FOS_PICKFOLDERS, FOS_FORCEFILESYSTEM, FOS_PATHMUSTEXIST and FOS_DONTADDTORECENT (written out in the test), and its folder, a start named "my project é", a missing start leaving it where it opens by itself; in Windows Terminal at 208x59 the folder button opened "Choose a project directory" (a #32770 window) over the terminal at %TEMP%\p52\start (its address bar), with the system's own Folder box and Select Folder button (in French); %TEMP%\p52 typed and Enter opened that folder in the dialog, Enter again chose it: the field "C:\Users\…\Temp\p52", focused; a second dialog closed with Esc: the field unchanged; no frame written while either dialog was open (the focus reports, out then in), the home screen whole after
PASS 52.4 The browser gone: app/browse.go, render/browse.go, input/browse.go, their tests, the 12 browser golden frames, Options.Lister and UI.Browser removed; realPath and dosPath moved unchanged (TestRealPathDOS passes); plural, mkdirs and ctrl kept where still used; no document presents the browser as present (ui-design.md, architecture.md, evaluation.md, reference-design.md, AGENTS.md; the closed plans' records kept); the startup screen's 18 golden frames unchanged (TestGoldenFrames without -update)
PASS 52.5 The checks: go build ./... ok; go vet ./... ok, and with GOOS=linux on internal/...; gofmt -l . empty; go test ./... ok; -race on internal/app (334 s) and internal/input (118 s) ok; 17 mutants through -overlay, 17 killed, none by a build failure (N14 written again as N14b, its first form not compiling); no mutant's text in the tree after, its hash the same around N14b's run
Verdict: DONE
```

## Details

### Choosing a typed folder

In a folder dialog, a folder's path typed into its Folder box with Enter
opens that folder in the dialog, the system's own behaviour; Enter again,
or a click on Select Folder, chooses it. The run typed the path, so it
took the two Enters; a folder clicked in the list and Select Folder take
one.

### The mutants

| Id | The change | Caught by |
| - | - | - |
| N1 | a second dialog while one is open | TestFolderDialog: 7 dialogs, not 6 |
| N2 | a dialog while a start is validated | TestFolderDialog |
| N3 | the failure's note kept when the next dialog opens | TestFolderDialog |
| N4 | no walk up to an existing parent | TestFolderStart |
| N5 | no fallback | TestFolderStart |
| N6 | the field not focused after a choice | TestFolderDialog |
| N7 | a failure not shown | TestFolderDialog |
| N8 | Cancel taken as a choice of "" | TestFolderDialog |
| N9 | the dialog never marked closed | TestFolderDialog |
| N10 | Ctrl+N keeping the failure's note | TestFolderDialog |
| N11 | Ctrl+O doing nothing | TestFolderDialog |
| N12 | the folder button doing nothing | TestHomeMouse |
| N13 | the note not drawn | TestFolderDialog |
| N14b | the dialog opened at the starting folder, not the field's | TestFolderDialog |
| N15 | the dialog not limited to folders | TestFolderDialog (input) |
| N16 | the start folder not set | TestFolderDialog (input) |
| N17 | a file taken as the folder to open at | TestFolderStart |

The first run's hashes of the tree differ by an edit of `plan.md` made
during it; the tree held none of the mutants' text after it, and N14b's
run left the hash as it was.

**The commit (2026-10-27), after the owner's approval.** The owner
approved plan 52 on 2026-10-27. Commit 9b8b6e5 (51 files: 23 modified, 19
deleted, 8 added, 1 renamed), then the approval commit; the working tree
is clean, and the author is `Evengelius <molimoabel@gmail.com>`, with no
attribution line: PASS. Not pushed: the owner asked for the commit. The
block above stays as it was reported.
