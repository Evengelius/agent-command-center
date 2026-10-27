# Plan 56 evaluation (2026-10-27)

The check of plan 56, the Explorer's font installed by acc (brief
amendment 57): Cascadia Mono NF bundled in the repository under its
licences, and installed by `acc` for the user when Windows has no font of
that name, so that a clone shows the Explorer's icons on any Windows
computer once Windows Terminal starts again.

It covers the 6 criteria of [`plan.md`, Plan 56](../plan.md), written
before the code. Nothing is committed or pushed: the owner reviews first.

How it was checked:
- **The bundle**: the files' SHA-256 against the release's
  (`sha256sum -c SHA256SUMS`), `git check-attr`, the licence test.
- **The install script** in Windows Terminal windows, on temporary fonts
  folders and keys under `HKCU\Software\AgentCommandCenterTest`
  (`scratchpad\term\p56install.ps1`, `p56reuse.ps1`): each path of D2 and
  D3, and its cost with the real keys; the real Fonts key and folder the
  same before and after each run.
- **End to end on this computer**, with the owner: the font uninstalled
  through Windows' Settings, Windows Terminal started again twice and
  `.\acc` run by the owner each time; the registry read through WMI's
  registry provider (`term\p56state.ps1`), the owner's tabs through UI
  Automation (`term\p56line.ps1`), the Explorer in a window of the owner's
  Terminal compared row by row with plan 55's shots (`plan55\shotcmp`);
  D4's probe (`term\p56probe.ps1`).
- **Build, vet, gofmt, the whole suite** after the last edit.

No mutants: the plan's one Go change is the glyph test's source, and the
install script is PowerShell, which `go test -overlay` cannot reach; each
of its paths was run instead (Details).

## The evaluation

```text
Evaluation - Plan 56, the Explorer's font installed by acc - 6 criteria: 6 PASS, 0 FAIL
PASS 56.1 The bundle: third_party/fonts/cascadia-code-2407.24 holds CascadiaMonoNF.ttf (2,526,924 bytes, SHA-256 5B12CBEF13B6DE9E773F585917783583D40D944A2578C3DAE99B5E4FD913A14E) and CascadiaMonoNFItalic.ttf (2,336,432 bytes, 0D3729255BA46B06EB292F9EADADDEF55A106632D3EAD4D1E8EC31D44952F534), the release's (sha256sum -c SHA256SUMS: the three files OK); LICENSE the file of tag v2407.24 (4,489 bytes, CRLF, SHA-256 51882cd3...4f958238); NOTICE.md names the 13 symbol sets with their licences and sources, the CC BY 4.0 holders by name; git check-attr: both TTFs binary set, text and diff unset, LICENSE text unset; THIRD_PARTY.md's Bundled font section and AGENTS.md's licence rule name the font; the licence test passes
PASS 56.2 Installed when missing: run A, a fonts folder and key that did not exist: both made, both files the bundled bytes, both values with the files' full paths, the line, True returned, the files created during the run; run B, A's again: False, no line, the files not written again; run G, the two files there already with the bundled bytes and a creation time of 2020-01-01, no value: reused (not written again), both values, the line, their creation time the run's
PASS 56.3 Failures: run C, a copy one byte changed: "Cascadia Mono NF not installed: ...\bundleC\CascadiaMonoNF.ttf is not the bundled file (SHA-256 410E0805...)", no file, no value; run D, "not a font" under the font's name: "... is another file of that name, left as it is", that file unchanged, no value; run E, a key under HKLM: "... Access to the registry key 'HKEY_LOCAL_MACHINE\SOFTWARE\AgentCommandCenterTest\Fonts' is denied.", no key made; each one line and True returned, so that acc goes on to its tab
PASS 56.4 Installed already, here: run F, the real keys with the font registered: False, no line, nothing made, median 9.83 ms of 20 runs (max 34.72 ms; under 20 ms asked); the owner's .\acc in the Terminal started again at 23:47:53: the fragment written at 23:48:14 with "face": "Cascadia Mono NF", no font warning in the window (no element naming a font), the font's files and values as the first run left them
PASS 56.5 End to end here: the font uninstalled by the owner through Windows' Settings (WMI: no Cascadia Mono NF value in either Fonts key, no file; more than the two values the criterion names), Terminal started again (23:37:07) and .\acc run by the owner: the two values back in the real HKCU key (WMI), the two files the bundled bytes, created 23:37:29.855, after Terminal's start; the line in the tab acc ran in; the fragment rewritten without the face (23:37:33); the Explorer in a window of that Terminal the same pixels as plan 55's before shots on every row whose text was the same (the mock's clocks alone differ): its own glyphs; the probe's answer recorded: a running Terminal draws a font installed after it started, in a window opened after a settings reload (plan.md, Details); Terminal started again (23:47:53): the face, and the Explorer the same pixels as plan 55's approved after shots on every row whose text was the same: the icons
PASS 56.6 The checks: TestExpIconsInFont on the bundle, no skip: PASS; go build ./... ok; go vet ./... ok (Windows and Linux); gofmt -l . empty; go test ./... ok (16 packages; TestNerdIconsHere passing too); no golden frame or mock file changed (git status testdata empty)
Not verified: another computer (none at hand); the nearest are run A (a fonts folder and key that did not exist) and this computer after Windows' uninstall. How soon after an install a running Terminal can draw the font (the probe ran 8 minutes after): not a criterion, left for the decision D4 defers
Verdict: DONE
```

## Details

### The install script's runs

Each run in a Windows Terminal window, so that its registry writes reach
the real registry rather than the Claude app's private copy (plan 55,
The font's registration); `-NoSession` on the temporary runs, so that no
font was loaded into the session.

| Run | Before the run | Returned | Line | After the run |
| - | - | - | - | - |
| A | no fonts folder, no key | True | installed | both files, the bundled bytes, created during the run; both values |
| B | A's result | False | none | the files not written again |
| G | both files, the bundled bytes, created 2020-01-01; no key | True | installed | the files not written again, created 23:52:48 (the run's); both values |
| C | the bundle one byte changed | True | not the bundled file | no file, no value |
| D | a 10-byte "not a font" under the name | True | another file of that name | that file unchanged; no value |
| E | the key under HKLM | True | access denied | no key |
| F | the real keys, the font registered | False | none | nothing made; median 9.83 ms of 20, max 34.72 ms |

A to F ran in step 3 (`term\p56install.log`), G after the end-to-end
run (`term\p56reuse.log`). After each script the real Fonts key and folder
were as before, creation times included for G, and the temporary keys
and folders were gone.

### End to end

At 21:51, after the owner's uninstall, WMI showed no Cascadia Mono NF value
in the user's Fonts key or the machine's, the user's font folder held no
Cascadia file, and no Windows Terminal ran. The owner then started
Terminal (process started 23:37:07) and ran `.\acc` in the repository: the
install at 23:37:29.855, the line left 3 s, the fragment written at
23:37:33 without the face. The acc tab opened on the home screen; the
first tab still showed the line, read through UI Automation:

```text
Agent Command Center: Cascadia Mono NF installed for this account; the Explorer's icons appear once Windows Terminal starts again.
```

The Explorer was shot in a window of that Terminal (`term\p55shots.ps1`,
the mock dashboard on plan 55's demo project, F3 then folders opened),
and the owner's Terminal again after it started at 23:47:53 and the
owner's second `.\acc`. `shotcmp` compared each with plan 55's shots on
the grid (row 0 at y 58, rows of 17 pixels, within one level a channel):

| Shots | Rows the same | Text changed | Moved within a run | Same text, other pixels |
| - | - | - | - | - |
| first run e1 / before | 58 | 3 (clocks) | 0 | 0 |
| first run e2 / before | 57 | 4 (clocks) | 0 | 0 |
| first run e3 / before | 55 | 3 (clocks) | 3 | 0 |
| second start e1 / after | 44 | 17 | 0 | 0 |
| second start e2 / after | 56 | 5 (clocks) | 0 | 0 |
| second start e3 / after | 55 | 3 (clocks) | 3 | 0 |

The second start's e1 lists 14 rows whose text changed with no pixel
changed: UI Automation read them at another moment of the mock's playback
and counted zone C's columns otherwise; their pixels are the same.

During the second shots the owner had started a live Claude session in the
acc tab; the shots were mock-only, in a window of their own on a
temporary home and app data, and that session was left as it was. The
first tab, behind it, was not read on that run: the install script did
nothing there, as the font's files and values were as the first run left
them and the face was written, which needs the registration the script
checks first.

### The probe (D4)

In the Terminal started at 23:37:07, before the install, a throwaway
fragment named the face in a profile of its own; its GUID was plan 23's
fragment test's, already in Terminal's generated profiles, so that
Terminal recorded nothing new. settings.json's time was set, its bytes
untouched, as `acc` does, and a new window of that Terminal (the same
process) opened with the profile and printed U+E627, U+EAC4, U+E5FF,
U+E5FE, U+EA85, U+F51B and U+E609 between "PROBE" and "END"; a control
window of the acc profile, then without the face, printed the same line.
The probe's window drew the seven icons and showed no font warning (no
element naming a font beyond the windows' titles); the control drew
missing-glyph marks. Of the glyphs' 14 cells, 397 pixels differ between
the two; "PROBE " and " END" match to the pixel. The windows were closed,
the fragment removed, settings.json's time set again: its bytes and
state.json's generated profiles were as before.

So plan 55's assumption, that Terminal reads fonts only when it starts,
does not hold for a window opened after a settings reload, 8 minutes after
the install here. Plan 55's rule stays, as D4 says: a font newer than the
running Terminal gives the tab no face and the Explorer its own glyphs.
Whether the first run could show the icons, with both checks relaxed, is a
later decision; it would first need how soon after an install the font
can be drawn.

**Withdrawn (2026-10-28).** Asked whether plan 56 was approved for its
commit, the owner asked for icons that need nothing installed and chose
Segoe Fluent Icons, which comes with Windows 11 (brief amendment 58): plan
57 was proposed to replace plan 55's Nerd Font icons and drop this plan's
bundle and install. Nothing of plan 56 was committed. The block above stays
as it was reported; its runs and its probe (a running Terminal draws a font
installed after it started, in a window opened after a settings reload)
are kept for the record.

**Reinstated (2026-10-28).** The owner reconsidered before approving plan
57 (brief amendment 59): Nerd Fonts and plan 56 kept, for the languages'
icons and the same look on any Windows version; plan 57 withdrawn without
being implemented; plan 55 unchanged. Plan 56's code, scripts and bundle
were not touched in between (`git diff` against 5f6f974 the same four
files and lines as at this evaluation, plan 55's code unchanged). Run again
on the reinstated tree: go build, go vet (Windows, Linux) and gofmt clean,
go test ./... ok (16 packages; `TestExpIconsInFont`, `TestNerdIconsHere`
and `TestAgentsMDWithinCodexLimit` passing), no golden frame or mock file
changed, `sha256sum -c SHA256SUMS` OK for the three files, the TTFs binary
for git. This computer as the end-to-end run left it (WMI): the two values,
the two files the bundled bytes created by acc at 23:37:29.855 on
2026-10-27, the fragment naming the face. The block above stands.
