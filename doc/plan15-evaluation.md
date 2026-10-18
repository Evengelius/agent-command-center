# Plan 15 evaluation (2026-10-18)

The final check of plan 15, recent projects, favourites and a project
browser (amendment 9): its four items of
[`plan.md`, Plan 15](../plan.md#plan-15---recent-projects-favourites-and-a-project-browser)
against the yes/no criteria written there before each item's work started,
and the plan's rules (the startup screen only, the dashboard, the agent CLIs
and their configuration untouched, no new dependency, no agent CLI started).
The evidence is `plan.md`'s record of each item, the tests, the golden frames
and the two terminal runs. The block below is the summary reported to the
owner. The results are not to be rewritten.

```text
Evaluation - Plan 15, recent projects, favourites and a project browser - 21 criteria: 21 PASS
PASS 1.1 The store: Save then Load gives the same entries; a missing file is an empty list without error; a damaged file is an error naming the file; a failed write leaves the previous file whole (TestProjectsStore)
PASS 1.2 Recording moves the project to the front with its start time, no duplicate (a path differing only in case is the same project on Windows), at most 10 recent projects (TestRecordProject)
PASS 1.3 A session started from the startup screen records its validated path; a refused validation, a mock run or no store records nothing (TestProjectRecording); live, "Use this directory" then a refused validation left projects.json unchanged, at both sizes
PASS 1.4 The list shows name, path and age; with no store or an empty one the existing startup frames are unchanged; with 10 entries at 80x24 three rows show ("6–8 of 10") and the Start button is whole (golden frames startup-projects, startup-projects-focused; TestGoldenFrames checks the button in every startup frame; TestAge)
PASS 1.5 Tab goes agent, project, list, Start; ↑↓ move in the list and still choose the agent in field 1; Enter starts with the entry's path; Del forgets it and the store no longer holds it (TestProjectListKeys)
PASS 1.6 A store that cannot be read shows "Recent projects unavailable:" and the reason; one that cannot be written shows "Recent projects not saved:" and the reason (TestProjectStoreProblems)
PASS 2.1 f stars an entry and the store keeps it; f again unstars it; f in the project field is typed (TestProjectFavourites, TestStarProject)
PASS 2.2 Favourites first with ★, then the others, each group by its last start; a favourite is never dropped by the cap of 10; a directory starred before any session shows "never" (TestStarProject); live: "★ alpha … never" in the list at both sizes
PASS 2.3 Del on a favourite forgets it (TestProjectFavourites); golden frame startup-favourites at the four sizes
PASS 3.1 The listing on a temporary tree: directories only, sorted without case; git from a .git directory or file; hidden and system directories counted and left out; a junction listed as a link; a missing, empty or file path gives its message (TestDiskList, TestSortEntries, TestDiskListWindows); live: alpha "git", link "link  git", "3 directories · 1 hidden, Ctrl+H shows them"
PASS 3.2 The browser opens at the field's directory, else its nearest existing parent, else the home directory (TestBrowserOpens); live: Ctrl+O opened at …\p15-<size>\tree, the field's directory
PASS 3.3 Keys: open, up with the directory left selected, filter, Esc clears then closes with the field unchanged, Enter on the first row fills the field and starts, Ctrl+F stars in the store, the drives above a root (TestBrowserKeys, TestBrowserDrives); live in Windows Terminal, every one at both sizes, Ctrl+H arriving as ctrl+h and not as Backspace, the drives after 8 x ←
PASS 3.4 A slow listing shows "reading…" with an empty count and keys stay handled (TestBrowserSlow); golden frames browser, browser-filter, browser-reading at 79x24, 80x24, 120x40, 200x60 from a fake listing
PASS 4.1 Amendment 9 in doc/brief.md; the three items left [Nice to have] in plan.md and AGENTS.md; ui-design.md (Startup screen, Projects, Project browser, Keys, the image row), reference-design.md's image row, architecture.md (Out of scope, AM4 S13, X1-X3, X4-X10, ASM-24), reliability.md and doc/evaluation.md's MVP-scope check say what is built
PASS 4.2 go build ./... and go vet ./... (also -tags live, -tags timing) exit 0; gofmt -l . prints nothing
PASS 4.3 go test ./... 288 top-level tests PASS and 1 SKIP in 14 packages, 0 FAIL (baseline 84d2ad8 273, +16: 6 for item 1, 2 for item 2, 8 for item 3)
PASS 4.4 The timing step run alone (the startup screen is a render path): TestKeystrokeToFrame p95 1.01 ms, TestDraftKeystrokes p95 2.03 ms, TestKeystrokeCost, TestLongSession delta-to-frame p95 3.02 ms, all PASS (target p95 <= 16 ms)
PASS 4.5 Terminal run, Windows Terminal maximised (156x44): 26 checks PASS, 0 FAIL; a temporary tree, store and home, PATH without any agent CLI: both shown not installed, none started; Ctrl+Q closed the window, no process left
PASS 4.6 Terminal run, Windows Terminal at 80x24: the same 26 checks PASS, 0 FAIL
PASS 4.7 The 61 changed and new files (this one included) carry no CR, no BOM and no NUL
PASS 4.8 This summary written here and in plan.md's progress list
Not verified: a symbolic link in the listing (TestDiskListLink SKIPs: creating one needs a privilege this account lacks; the junction covers the link case, in the tests and live). Enter on "Use this directory" starting a real agent CLI was not run live, by design (no CLI started); TestBrowserKeys covers it, and it is the startup screen's existing Start path.
Verdict: DONE
```

## The terminal runs

Both runs used the same script, real keys sent to a Windows Terminal window
and the screen read through UI Automation. The tree held `alpha` (with a
`.git` folder), `beta`, a hidden `.hidden` and `link`, a junction to `alpha`.
`LOCALAPPDATA` and `USERPROFILE` pointed at temporary folders, so the
owner's store was neither read nor written, and `PATH` held only
`C:\Windows\System32`, so no agent CLI was found or started.

The 26 checks:

- the startup screen and its size;
- both agent CLIs shown as not installed;
- "Ctrl+O browse" in the footer;
- no Projects list with an empty store;
- the browser opening at the field's directory;
- the rows and their marks;
- the hidden count, then Ctrl+H showing and hiding;
- the filter and its selection, then Esc clearing it;
- Ctrl+F starring alpha and the store writing it;
- opening alpha, going up, the drives;
- Esc closing with the field unchanged;
- the starred project in the list;
- "Use this directory" filling the field, validation refusing it (no agent
  CLI), nothing recorded;
- Ctrl+Q.
