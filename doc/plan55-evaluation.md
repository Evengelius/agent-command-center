# Plan 55 evaluation (2026-10-27)

The check of plan 55, Nerd Font icons in the Explorer (brief amendment 56):
Cascadia Mono NF installed for the owner, the acc tab's face, and the
Explorer's rows drawing an icon of each kind in its colour where that font
is there, the panel's own glyphs anywhere else.

It covers the 6 criteria of [`plan.md`, Plan 55](../plan.md), written
before the code, and the owner's three requirements: R1 no visual change to
plan 51's home screen, R2 a reliable fallback, R3 nothing beyond the
Explorer's icons and the font's configuration. Nothing is committed or
pushed: the owner reviews first.

How it was checked:
- **Unit and golden tests**: the icons by name and extension, their widths
  and their presence in the installed font; the rows with and without the
  icons; the fallback rule on its eight cases; four new golden frames, the
  old ones unchanged.
- **Windows Terminal**, the real application on a temporary home and app
  data, no CLI session: before and after shots of the home screen and a
  mock dashboard in the acc tab, compared row by row (`scratchpad\plan55\shotcmp`);
  the Explorer on a copy of this repository at 208x59 and 120x40, each
  icon's colour measured (`plan55\iconcolors`); the default profile.
- **The font's coverage** against Windows Terminal's own Cascadia Mono, and
  the sources searched for what differs (`plan55\cmapdiff`).
- **Each change broken on purpose** through `go test -overlay`
  (`scratchpad\plan51\mutrun`, the lists in `scratchpad\plan55`), the
  repository never written: a hash of `git diff` and the plan's untracked
  files the same before and after.
- **The timing step** alone, and **build, vet, gofmt, the whole suite**
  after the last code and document edit.

## The evaluation

```text
Evaluation - Plan 55, Nerd Font icons in the Explorer - 6 criteria: 5 PASS, 0 FAIL
PASS 55.1 The font: CascadiaCode-2407.24.zip, 150,454,761 bytes, SHA-256 E67A68EE3386DB63F48B9054BD196EA752BC6A4EBB4DF35ADCE6733DA50C8474; CascadiaMonoNF.ttf (2,526,924 bytes, SHA-256 5B12CBEF13B6DE9E773F585917783583D40D944A2578C3DAE99B5E4FD913A14E) and CascadiaMonoNFItalic.ttf (2,336,432 bytes, 0D3729255BA46B06EB292F9EADADDEF55A106632D3EAD4D1E8EC31D44952F534) in %LOCALAPPDATA%\Microsoft\Windows\Fonts; "Cascadia Mono NF (TrueType)" and "Cascadia Mono NF Italic (TrueType)" in the user's Fonts key, read through WMI's registry provider; no administrator prompt; the uninstall in D1 (from a Windows Terminal window). The first registration, from this session's shell, had stayed in the Claude app's private registry: redone from Windows Terminal (details)
PASS 55.2 The acc tab and R1: the fragment without a face where the font was not registered or newer than the terminal (the owner's .\acc in the Terminal of 15:38, the font unregistered: 0.6 s, nothing written; in that Terminal after the repair, the launcher's function False, the font of 15:58 newer), with "face": "Cascadia Mono NF" after Terminal started again (16:01:29; written 16:01:41 by the owner's .\acc); no font warning, the maximised tab 208x59; before and after, row by row: the home screen's 38 rows outside the banner's 23 the same pixels in both shots, the field's path the same text; the dashboard's 55 rows the same pixels, 5 rows differing only where their text did (the run's folder, the mock's clocks), 1 row moving within each run; no row with the same text differing by a level
PASS 55.3 The icons: TestExpIconFor (57 names to their code point and colour, an upper-case extension among them), TestExpIconsInFont (every glyph in the installed font's character map), TestExpIconWidths (each icon one cell, each chevron with its space two)
PASS 55.4 The fallback (R2): TestNerdIconsUsable's 8 cases (no Windows Terminal, another profile, none, no font, a terminal started before the font or as it was installed give the panel's own glyphs; the acc tab after the font, its id in either case, the icons); in Windows Terminal, the Terminal of 11:12, started before the install, and the default profile (0 private use glyphs; 24 folder and file glyphs, 12 chevrons of the panel's own) draw the panel's own glyphs, so did the owner's Terminal of 15:38 while the font was unregistered; the explorer golden frames unchanged; explorer-nerd at 79x24, 80x24, 120x40 and 200x60, differing from explorer only in the chevrons and the icons (80x24: 13 lines, 120x40: 18, 200x60: 18, 79x24: none)
PASS 55.6 The checks and R3: go build ./... ok; go vet ./... ok (Windows and Linux; the timing tag too); gofmt -l . empty; go test ./... ok; the timing step's TestMotionToFrame p95 1.02 to 2.02 ms, the panel shown or not, the icons on or off (target 16 ms); no golden frame changed outside the four new ones; 23 mutants through -overlay, 22 killed by a failing test, U3 equivalent (details), the tree's hash the same around each run; Cascadia Mono NF adds 7 code points outside the private use area, none in the sources; the panel's header, rows, indents, marks and widths unchanged
Not verified: 55.5's last part, the owner's visual review, asked before the plan closes. Its measured part holds: on a copy of this repository at 208x59 and 120x40 in the acc tab, every row's icon drawn, none a missing-glyph box, each glyph's pixels a blend of its D4 colour with the row's ground within 0.9 of a level (the chevrons grey, the ignored .exe grey, the cursor row's icon on the row's ground); review shots plan55\review-208x59-explorer-x3.png, review-120x40-explorer-x3.png, explorer-before-after-x2.png
Verdict: NOT DONE
Proposed next step: the owner's visual review of the Explorer in the acc tab (or the review shots) - waiting for approval
```

## Details

### The font's registration

The install of step 2 copied the two files into the user's font folder and
wrote their two values from this session's shell. That shell runs under the
Claude desktop app, an MSIX package: its registry writes go to the
package's private hive (`Packages\Claude_pzs8sxrjxfjjc\SystemAppData\Helium`),
even from processes without the package's identity, while its file writes
reach the real folders. So no other program saw the font: after the first
restart of Windows Terminal, the launcher's check and the dashboard's both
found no registered font and kept the panel's own glyphs (R2 held: no
warning, no missing-glyph box). WMI's registry provider, outside both
packages, showed the user's Fonts key without the two values, as
Terminal's tabs did.

The repair: the app's private values removed; from a Windows Terminal
window (`term\p55repair.ps1`), the two files copied back with their hashes
checked and the two values written to the real key, read back through WMI.
Windows' file name tunnelling kept the files' old creation time on the copy
back (a file made again under the same name within 15 s takes the old
one's), so their creation time was set to the registration's, 15:58:25:
the rule takes the files' time for the install's, and the Terminal started
at 15:38 had not seen the font. The launcher's function, copied unchanged,
then answered False in that Terminal (`term\p55walk.ps1`), and True's
consequence, the face, came after the next restart.

### R1, row by row

`shotcmp` compares two shots of the same screen on the terminal's grid,
measured on the shots (row 0 at y 58, rows of 17 pixels: the bands of the
dashboard's highlighted tool rows): a row whose text is the same in both
runs must have the same pixels, within one level a channel; a row that
moved between a run's two shots is left out (the banner, a spinner); a row
whose text changed is listed. Every row with the same text matched to the
pixel: Cascadia Mono NF draws text exactly as Terminal's own Cascadia Mono.
The only code points it adds outside the private use area are U+23FB to
U+23FE, U+2630, U+26A1 and U+2B58, none of them in the sources.

### The timing step

On mains power, the CPU 16 to 22% busy just before, the owner's usual
applications open (VS Code, Edge, Teams, Outlook, Windows Terminal with
the owner's dashboard and a Claude session, Move Mouse). Fifteen of the
sixteen tests within their targets. `TestBannerToFrame`, plan 51's banner,
not a criterion of this plan: p95 2.35 ms over its 2 ms target. Timed then
in turn with the tree before this plan (a clone of HEAD, each test binary
built first): before 2.45, 1.62 and 1.62 ms, after 1.66, 1.61 and 1.63 ms;
the miss is the machine's load, as in plan 51, not this change, which
leaves the banner's code as it was.

### The mutants

| Id | The change | Caught by |
| - | - | - |
| R1 | names not lower-cased | TestExpIconFor |
| R2 | folders looked up as written | TestExpIconFor |
| R3 | open and closed folders swapped | TestExpIconFor |
| R4 | files' names looked up as written | TestExpIconFor |
| R5 | `LICENSE.*` not a licence | TestExpIconFor |
| R6 | `.env.*` not configuration | TestExpIconFor |
| R7 | `Dockerfile.*` not Docker | TestExpIconFor |
| R8 | extensions looked up as written | TestExpIconFor (the upper-case case added for it) |
| R9 | any other file a binary's icon | TestExpIconFor |
| R10 | a closed folder drawn open | TestExpIconFor |
| E1 | an ignored row's icon in its own colour | TestExplorerNerdRows |
| E2 | a deleted row's icon in its own colour | TestExplorerNerdRows |
| E3 | the cursor row's icon off its ground | TestExplorerNerdRows |
| E4 | the chevrons swapped | TestExplorerNerdRows |
| E5 | an open folder's icon closed | TestExplorerNerdRows |
| E6 | the root row's folder closed | TestExplorerNerdRows |
| U1b | the profile's id compared with its case | TestNerdIconsUsable |
| U2 | no font needed | TestNerdIconsUsable |
| U3 | no Windows Terminal check | equivalent: a zero start is never after a file's time, so `Before` already refuses it; the check is kept for the reader |
| U4 | a terminal started as the font was installed | TestNerdIconsUsable |
| U5 | the times compared the other way | TestNerdIconsUsable |
| W1 | the face's name misread | TestNerdIconsHere |
| W2 | the font never found | TestNerdIconsHere |

U1 as first written removed `strings`' only use and did not build; U1b
keeps it. The first list's E1 to E5 and U1 to U3 did not apply (one tab
too many, and parts of a line where the runner matches lines) and were
written again. `acc.ps1`'s `NerdFontReady` cannot be mutated through Go: it
was checked live, False in the Terminal of 15:38 and the face written after
the restart.

### Review points for the owner

Two of Seti's glyphs read less well at 9 pt than the others: Go's
(`seti-go`, U+E627) is a small cyan block, Markdown's (`seti-markdown`,
U+E609) a down-arrow. The font also has the devicons' Go (`dev-go`,
U+E724) and Markdown (`dev-markdown`, U+E73E) and the octicons' Markdown
(`oct-markdown`, U+F48A), checked in its character map; left as D4 chose
them unless the owner prefers another.

**55.5, the owner's visual review, and the commit (2026-10-27).** The
owner reviewed the Explorer's icons in the acc tab and the review shots
and wrote "The Explorer's icons look right: plan 55 is approved", asking
for the commit: 55.5's last part, PASS, the icons kept as D4 chose them.
Commit 374c23e (29 files: 17 modified, 12 added), from the files reviewed
in `git diff`; the whole suite passed again after the last edits, two
passages rewrapped (`acc.ps1`'s header comment, evaluation.md's
`TestMotionToFrame` entry). The author is `Evengelius
<molimoabel@gmail.com>`, with no attribution line, and the working tree
held only this record's changes after it: PASS. Not pushed: the owner
asked for the commit only. The block above stays as it was reported.

**The push (2026-10-27), at the owner's request.** Commit 374c23e and the
approval commit, 7319752, were pushed to `origin`: `main` fast-forwarded
from 41ff147 to 7319752, nothing forced; `origin/main` equals `main`
(read back by `git ls-remote`), the working tree clean: PASS. The commit
recording this was pushed after it.
