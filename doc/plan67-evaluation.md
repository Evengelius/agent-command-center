# Plan 67 evaluation (2026-10-29)

The check of plan 67, the characters typed with AltGr (brief amendment
69): every field of the dashboard types the characters AltGr types, as
plan 66's search bar does; Ctrl or Alt alone types nothing, and so does
Ctrl with the left Alt; the approval card lets no character into the draft
under it; the shortcuts stay as they were. Another agent's review, shared
by the owner, asked whether the key event ACC receives tells AltGr from
Ctrl with the left Alt; answered by measuring, it was folded into the
plan, and the owner approved it: "Plan 67 approved. You can proceed with
implementation."

It covers the 6 criteria of [`plan.md`, Plan 67](../plan.md), written
before the code. Nothing is committed or pushed: the owner reviews the
results and the diff first.

How it was checked:
- **Application tests** (`internal/app/altgr_test.go`, 5 tests) on the
  scripted CLI and the rigs of each field: the rule on the events as ACC's
  decoder makes them from the records Windows Terminal sent; AltGr's "@" in
  each of 11 fields, sent as Windows Terminal sends it (the left Ctrl's own
  press, the right Alt's, the character, then the three releases); Ctrl
  with the left Alt, Ctrl+K and Alt+K carrying a text in each; the
  approval card; the shortcuts; the branches popup and a modifier's own
  press.
- **A render test** (`internal/render/altgr_test.go`): the key overlay's
  table holds no Ctrl+Alt chord.
- **Each change broken on purpose** through `go test -overlay`, the
  repository never written, its code hashed before and after: 23 mutants
  (`scratchpad\plan67\mkmutants67.mjs`, run by plan 51's mutrun).
- **Windows Terminal**: the application built from this tree, in ACC's
  profile, maximised (208x59) and at 120x40, on the owner's Belgian AZERTY
  layout, AltGr presses injected at the keyboard's level (the right Alt as
  an extended key, Windows adding the left Ctrl as it does for a physical
  press; no physical press, the review below), the records traced
  (TEA_TRACE), on a demo repository (`scratchpad\term\p67runs.ps1`,
  `p67runc.ps1`). The branches popup on a stand-in `claude.exe`: no CLI
  ran.
- **The race detector** on internal/app's tests of the changed code, and
  **build, vet (Windows and Linux), gofmt, the whole suite** after the
  last code and document edit; **the timing step** alone, on mains power.

## The evaluation

```text
Evaluation - Plan 67, the characters typed with AltGr - 6 criteria: 6 PASS
PASS 67.1 The rule: TestKeyText, on the events as ACC's decoder makes them from Windows Terminal's records: AltGr's "@" and "\" (their text, Mod Ctrl|Alt) typed, with Caps Lock or Num Lock too; Ctrl with the left Alt's "\" (no text, Mod Ctrl|Alt) typing nothing; "a" typed with no modifier, "A" with Shift and with Caps Lock; a dead key's "ê" typed; "k" with Ctrl alone or Alt alone typing nothing though it carries the text "k"; Enter, no text, nothing (12 cases)
PASS 67.2 Every field: TestAltGrFields, AltGr+2 as Windows Terminal sends it (the left Ctrl's own press, the right Alt's, "@" with Mod Ctrl|Alt and its text, the three releases): "@" typed in the draft, the draft window, the startup screen's project field, Source Control's message box, the Worktrees view's new branch, the branches popup's search, the search bar, the file viewer's search field, the quick open and the scope popup's search, the draft untouched by the other fields; nothing in the file viewer's go-to field (digits alone); in each of the 11, Ctrl with the left Alt and "\" as Windows Terminal sends it, Ctrl+K and Alt+K carrying the text "k" typing nothing, the draft as it was; TestAltGrCard: the approval card focused, AltGr's eight characters as Windows Terminal sends them typed nowhere: the draft empty, the card on r1, the request pending, nothing written to the CLI
PASS 67.3 Nothing else, the shortcuts kept: TestKeyTableNoCtrlAlt: none of the 124 keys of the key overlay's 14 groups a Ctrl+Alt chord; TestAltGrShortcuts: each of @ \ # | { } [ ] typed with AltGr in the draft, as Windows Terminal sends it, adding itself and nothing else changed (the popups, windows, views, the overlay, the focus, a toast); Ctrl with the left Alt and each of them changing nothing at all; a dead key's "ê" typed; every existing test passing unchanged (no tracked test file in the diff; go test ./... ok), the key overlay's table (internal/render/keys.go) and the golden frames (testdata) untouched, TestKeyOverlayRows and TestGoldenFrames ok
PASS 67.4 Windows Terminal: ACC's profile, Windows Terminal 1.23.20211.0, maximised (208x59) and at 120x40, fr-BE, real AltGr presses, the records traced, the final build: "@\#|{}[]" in the draft; "~", "ê" and "à" composed; Ctrl with the left Alt and <: nothing, its records with 0x2A; Ctrl with the left Alt and 2: its press kept by Windows Terminal, its release alone received, nothing typed; "@" in the draft window, the search bar ("/ @"), Source Control's message box, the startup screen's project field, the Worktrees view's new branch ("Branch @") and the branches popup's search (on a stand-in claude.exe: no CLI ran), then "@F" there with Shift+F, Alt+B closing and opening it; the draft unchanged by every other field (Details, Windows Terminal). The first pass found the branches popup closed by AltGr's own presses: fixed (Details, What the build found and changed)
PASS 67.5 The checks: go build ./... ok; go vet ./... ok (Windows and Linux); gofmt -l . empty; go test ./... ok (16 packages) after the last code edit and the last edit of a document a test reads; 23 mutants through -overlay on the final code, the repository never written (its hash 13de3450 before and after): 23 caught by a failing test, none by the compiler; -race on internal/app's tests of the changed code (97 tests: plan 67's 5, and those of the branches popup, the search bar, the editor, the draft window, Source Control, the Worktrees view, the file viewer, the quick open, the Git Graph and its scope popup, the approval cards and the startup screen) and on TestKeyTableNoCtrlAlt: no race; the timing step alone, on mains power, at 17:52:58: its 21 tests within their targets
PASS 67.6 The documents: ui-design.md, Input editor's keys and Focus and keys' Typed characters (TestDocFrames ok); architecture.md, The characters typed with AltGr (keyText beside editKey, modifierKey); brief amendment 69, the owner's words, the review and the approval; AGENTS.md's status at the approval (TestAgentsMDWithinCodexLimit ok)
Not verified: a physical AltGr press (the presses were injected, the right Alt as an extended key, which Windows turns into the left Ctrl and the right Alt as a physical press does, the records the ones plan 67's probe measured); a Codex session in Windows Terminal (the fields are the dashboard's own, the same with both CLIs, D4; the terminal runs used Claude Code's mock and a stand-in claude.exe)
Verdict: DONE
Likely cause: -
Proposed next step: the owner's review of this report and of the diff, then the commits on the owner's approval; the two findings outside plan 67 (Details) for the owner to decide - waiting for approval
```

**The review the owner shared (2026-10-29).** The owner shared, with no
words of their own, another agent's review of this report and the diff:
"Implementation: APPROVED. Final closure: PENDING the two verification
clarifications above." It agrees with both departures from the design,
the three fields taking `keyText` and the branches popup keeping a
modifier's own press, and asks for no change of implementation. The block
above stays as it was reported; its two points, answered:

- **67.4, physical presses.** The criterion asked for real AltGr presses;
  the review reads them as physical ones, and the runs injected them.
  The two are distinguished:

  | 67.4's part | Evidence | Status |
  | - | - | - |
  | Injected presses through Windows' keyboard layout into Windows Terminal | the right Alt pressed as an extended key (`keybd_event`, `0xA5`), Windows adding the left Ctrl as it does for AltGr on this layout; the records traced; each field's result (Details, Windows Terminal) | measured: each step as 67.4 says |
  | Physical presses on the owner's Belgian AZERTY keyboard | none yet | not verified: the owner's keyboard check below, or the owner's explicit acceptance of the injected presses as sufficient |

  With the review, plan 67 stands at 5 PASS and 67.4 measured on injected
  presses, its physical presses not verified: NOT DONE until the owner
  checks the keys or accepts the injected presses.
- **The concurrent runs.** The final mutant run (17:44:09 to 17:51:27)
  and the suite (17:44:09 to 17:50:34) ran at the same time, and `-race`
  (17:50:34 to 17:52:25) overlapped the mutants' last 53 s, read from the
  files' times. This breaks no requirement. evaluation.md asks for "no
  other `go` build or test at the same time" in the timing step's
  conditions (Keystroke timing; AGENTS.md, Commands: "it runs alone"), and
  for fuzzing "each target alone". It sets no such condition for the
  suite, the mutants or `-race`, and `go test ./...` runs its packages
  concurrently itself. The timing step ran alone, from 17:52:58, with no
  `go` or mutrun process running before it. The evidence holds as it is:
  the suite's tests are deterministic, their times judged in the timing
  step alone (`overTarget`), and each mutant was killed by an assertion
  naming its failure, none by a timeout (The mutants). Nothing was run
  again.
- **The reader race** stays apart from plan 67, the vendored reader
  unchanged. Its reproduction is kept below (Details, The reader race's
  evidence) and in plan.md's Roadmap as a high-priority candidate for the
  next plan.

**The owner's keyboard check** (67.4's physical part). In the acc tab,
maximised, on the Belgian AZERTY keyboard: `.\acc --provider claude-code`
in this repository's folder. Enter starts the session at step 3 only,
which sends no message; never press Enter in the draft or in the new
branch's field.

| # | Where | Keys | Expected |
| - | - | - | - |
| 1 | the startup screen's project field | AltGr+2, then Backspace | the path ends with `@`, then without it |
| 2 | Ctrl+W, then n: the Worktrees view's new branch | AltGr+2, then Esc twice | `Branch @`; back on the startup screen, nothing created |
| 3 | the startup screen | Enter | the dashboard, "branch main" in the header |
| 4 | the draft | AltGr with 2, <, 3, 1, 4, 0, 5 and $; AltGr+= then Space; ^ then E; AltGr+µ then A; Ctrl with the left Alt and < | `@\#\|{}[]~êà`, the last chord adding nothing |
| 5 | Ctrl+G: the draft window | AltGr+2, then Esc | `@` added in the window and the draft |
| 6 | Ctrl+F: the search bar | AltGr+2, then Esc | `/ @` in the bar; the draft as it was |
| 7 | Alt+B: the branches popup | AltGr+2, Shift+F, then Alt+B | `@`, then `@F`, in its search, the popup staying open; Alt+B closes it; the draft as it was |
| 8 | a click on the ⑂ tile in the right sidebar, then in Source Control's message box | AltGr+2, Backspace, then Esc | `@` in the message box, then gone; the draft as it was |
| 9 | the draft | Ctrl+A, Backspace, then Ctrl+Q | the draft empty; the dashboard closes |

A step that does not show what it expects is the step to name. The file
viewer's fields, the quick open and the scope popup's search take the same
rule (TestAltGrFields), outside 67.4's list.

**The owner's keyboard check, done (2026-10-29).** The owner ran the check
above and wrote, word for word: "All steps passed". 67.4's physical
presses are verified with it, so 67.4 passes in both its parts and plan 67
stands at 6 PASS: DONE, for the owner's final approval before the commit.
The block above stays as it was reported.

**Approved and committed (2026-10-29).** The owner wrote, word for word:
"Approve, commit". Commit e888947 (13 files: 10 modified, 3 added), from
the files reviewed in `git status` and `git diff`, each of them plan 67's;
`git diff --cached --check` reported nothing. No file holds a CR or a BOM,
read from the bytes, and a scan of the change found no placeholder,
leftover or private path. No code file, and no document a test reads,
changed after the final suite began (17:44:09): only plan.md and this
file, which no test reads. The author is `Evengelius
<molimoabel@gmail.com>`, with no attribution line, and the working tree
was clean after it. Then the approval commit: plan.md and AGENTS.md close
plan 67, and this record; `go test ./internal/arch`, which reads
AGENTS.md, ok after its edit. Not pushed: the owner has not asked for it.
The block above stays as it was reported.

**Pushed (2026-10-29).** The owner then wrote, word for word: "Push them".
The checks on the committed tree, 9d1c5f7, the working tree clean before
and after: go build ok, go vet ok (Windows and Linux), gofmt -l . empty,
go test ./... ok (16 packages, 274 s). Then commits e888947 and 9d1c5f7
fast-forwarded origin's main from 3d3289b to 9d1c5f7, nothing forced,
origin's main read back equal to main (`git ls-remote`). The commit
record above, which says "Not pushed" as written, stays as it was, the
push following it.

## Details

### What the build found and changed

- **Three fields typed any key's text** (step 2). The file viewer's search
  and go-to fields (`fieldKey`), the quick open (`finderKey`) and the
  scope popup's search (`scopeKey`) took `k.Text` whatever the modifiers.
  The decoder gives no Ctrl or Alt chord a text but AltGr's, so they typed
  what `keyText` would, but 67.2's Ctrl+K and Alt+K carrying the text "k"
  typed "k" there. D2 had them keep their code; they take `keyText` too,
  the one rule of the fields.
- **The branches popup closed on AltGr's own presses** (step 3, run C).
  Windows Terminal sends a key with its modifiers as records of their own:
  AltGr+2 is the left Ctrl's press, the right Alt's, the "@" with Ctrl,
  Alt and its text, then the three releases (run C's trace; the decoder's
  events from them, `scratchpad\plan67\zz_sequence_test.go`, overlaid).
  The branches popup closes on a key it does not take and passes it on
  (plan 59, design D10): the left Ctrl's press closed it, so the "@" went
  to the draft (the trace's frame after that press redraws where the popup
  was). The application tests had sent the character alone. The same press
  comes before the popup's own chords, so by the code it closed the popup
  first there too: Shift before a capital, Ctrl before Ctrl+C (which then
  reached the dashboard, where Ctrl+C interrupts a running turn), Alt
  before Alt+B (which then opened the popup again). Fixed, D2
  extended: the popup keeps a modifier's own press (`modifierKey`,
  `branchesKey`), as the scope popup keeps the keys it has no use for. The
  other fields do nothing with a key that has no text (the search bar,
  `editKey`'s fields, the file viewer's fields, the quick open, the scope
  popup), and the approval card passes it on to the draft, where it types
  nothing. The tests send each key as Windows Terminal does (`altGrKeys`,
  `ctrlLeftAltKeys`); `TestBranchesModifierPress` pins the popup. The
  popup's switch confirmation is still cancelled by any other key, a
  modifier's own press among them, as before.
- **An AltGr key's name is its text.** Bubble Tea names a key that has a
  text by its text ("@", "["), not "ctrl+alt+@" as plan 67's findings
  said; corrected in plan.md. No conflict either way: no binding is
  `@`, `\`, `#`, `|`, `{` or `}`, and the bindings on `[` and `]`, zone C's
  and the inspection's previous and next agent, take AltGr's as before
  plan 67, which changes no name.

### Found outside plan 67

Neither is changed by plan 67; both are for the owner.

- **A race in ultraviolet's Windows reader.** `streamData`
  (`third_party/ultraviolet/terminal_reader_windows.go`) sends
  `buf.Bytes()` on its channel, then resets the buffer and fills it with
  the next console records, which `StreamEvents` may not have copied yet.
  Run A's first pass at 120x40 lost a Ctrl+F so: the trace logged the read
  as Ctrl's and F's presses (33 bytes), the reader processed 17 bytes of
  it and none of the next read. The scanner given the logged bytes
  processes all 33; given the next read's bytes in their place, it
  processes 17, then none, then on the timeout an unknown sequence and
  F's and Ctrl's releases: the press never reached the dashboard (The
  reader race's evidence, below). Three such events in the 17 traced runs, at both sizes: the
  Ctrl+F; Ctrl with the left Alt and < in run A's second pass at 120x40,
  whose Ctrl and Alt presses were lost and whose < press still arrived and
  typed nothing; and in the final pass at 208x59 a 13-byte read overwritten
  by the next record, most likely the resting pointer's 13-byte report,
  which the reads before and after it hold, nothing seen lost. The trace's
  logging, done before the copy, widens the window. Upstream's main has the
  same code (fetched 2026-10-29). A copy of the bytes before the send would
  close it; it is the vendored module's code, a patch for a later plan.
- **The other popups close on a modifier's own press.** The model and
  modes selectors, the "+" menu and the commit menu close on any key they
  do not take (`modelsKey`, `modesKey`, `key`, `commit.go`), a modifier's
  own press among them, as the branches popup did: in Windows Terminal a
  chord of theirs may close and reopen them, Alt+P the model selector.
  None has a field; not checked in Windows Terminal.

### The reader race's evidence

Kept for the next plan (the review the owner shared: "Please preserve the
reproduction evidence and record it as a high-priority candidate for the
next plan."); plan 67 changes nothing of it.

**The code**, the vendored ultraviolet (`v0.0.0-20261008173134-6b8d4baf91b4`,
its three patches elsewhere; `third_party/ultraviolet/PATCHES.md`). The
Windows reader serializes a batch of console records into one buffer,
sends the buffer's bytes, then resets it for the next batch
(`terminal_reader_windows.go`, `streamData`):

```go
		d.serializeWin32InputRecords(records, &buf)  // line 62
		...
		case readc <- buf.Bytes():                   // line 67
		...
		buf.Reset()                                  // line 70
```

The receiver logs the bytes, then copies them (`terminal_reader.go`,
`StreamEvents`):

```go
		case read := <-readc:            // line 199
			d.logf("input: %q", read)    // line 200
			buf.Write(read)              // line 201
```

`readc` is unbuffered, so the send completes when the receiver takes the
slice, and the sender goes straight on to `Reset` and its next console
read, writing into the same array before the receiver's `Write` if the
receiver is slower; under TEA_TRACE the log, a file write, comes first. The
reader of other systems has no such sharing: `sendBytes` reads into a
fresh array each time (`var readBuf [readBufSize]byte`, line 123, sent at
line 132). Upstream's main has the same three lines (`terminal_reader_windows.go`,
https://raw.githubusercontent.com/charmbracelet/ultraviolet/main/terminal_reader_windows.go,
fetched 2026-10-29).

**The trace**, run A's first pass at 120x40 (`trace-A-90x30.log`, 16:48:22,
the timestamps' prefix left out), Ctrl+F sent as four records:

```text
input: "\x1b[17;29;0;1;40;1_\x1b[70;0;6;1;40;1_"
processed 17 bytes from buffer
resetting timeout for remaining buffer after parse
input: "\x1b[70;33;6;0;40;1_\x1b[17;29;0;0;32;1_"
resetting timeout for remaining buffer after parse
timeout reached
timeout expired, processing buffer
```

**The replay**, the reader's own scanner and key table, a test overlaid in
`third_party/ultraviolet` (`go test -overlay`, never in the tree):

```go
package uv

import (
	"strings"
	"testing"
)

func TestZZAliasedRead(t *testing.T) {
	read1 := "\x1b[17;29;0;1;40;1_\x1b[70;0;6;1;40;1_"
	read2 := "\x1b[70;33;6;0;40;1_\x1b[17;29;0;0;32;1_"
	show := func(events []Event) string {
		var s []string
		for _, e := range events {
			switch e := e.(type) {
			case KeyPressEvent:
				s = append(s, "press "+e.String())
			case KeyReleaseEvent:
				s = append(s, "release "+e.String())
			default:
				s = append(s, "unknown")
			}
		}
		return "[" + strings.Join(s, ", ") + "]"
	}
	replay := func(label, first, second string) {
		s := NewTerminalReader(strings.NewReader(""), "xterm-256color").eventScanner
		buf := first
		n, ev := s.scanEvents([]byte(buf), false)
		t.Logf("%s: read 1 (%d bytes): %d processed, %s", label, len(first), n, show(ev))
		buf = buf[n:] + second
		n, ev = s.scanEvents([]byte(buf), false)
		t.Logf("%s: read 2 appended (%d bytes): %d processed, %s", label, len(buf), n, show(ev))
		buf = buf[n:]
		n, ev = s.scanEvents([]byte(buf), true)
		t.Logf("%s: timeout expired (%d bytes): %d processed, %s", label, len(buf), n, show(ev))
	}
	replay("as logged", read1, read2)
	replay("overwritten", read2[:len(read1)], read2)
}
```

```text
as logged: read 1 (33 bytes): 33 processed, [press leftctrl, press ctrl+f]
as logged: read 2 appended (34 bytes): 34 processed, [release ctrl+f, release leftctrl]
as logged: timeout expired (0 bytes): 0 processed, []
overwritten: read 1 (33 bytes): 17 processed, [release ctrl+f]
overwritten: read 2 appended (50 bytes): 0 processed, []
overwritten: timeout expired (50 bytes): 50 processed, [unknown, release ctrl+f, release leftctrl]
```

The logged bytes give 33 bytes processed, where the trace says 17: they
are not what the scanner read. The next batch's bytes in their place give
the trace's numbers exactly, 17, then none, then all on the timeout, and
no Ctrl+F press.

**How often.** Each trace scanned for a read the reader did not consume
whole at once (a read, then the next "processed N bytes" line): in the 17
traced runs, three events, at both sizes; the other reads not consumed
whole were the last one before a quit.

| Trace | Reads | What was lost |
| - | - | - |
| `trace-A-90x30.log`, lines 192 and 195 | 33 bytes, 17 processed; 34, none | Ctrl+F's two presses: the bar never opened |
| `trace-altgrA2-90x30.log`, lines 176, 179 and 181 | 34 bytes, 18 processed; 37 and 34, none | the Ctrl and left Alt presses before Ctrl with the left Alt and <; its < press arrived, twice, and typed nothing |
| `trace-A3-max.log`, lines 223, 225 and 227 | 13 bytes, none; 17 and 34, none | a 13-byte batch, most likely the resting pointer's report, which the reads before and after it hold; nothing seen lost |

**For the next plan** (not applied): the batch's bytes copied before the
send, or a buffer of its own for each batch, as `sendBytes` does; a test of
it needs the console reads behind a seam or a recorded batch sequence. The
traces and the scan (`tracecheck.mjs`) are in the session's scratchpad
(`term\out-p67\`, `plan67\`), not in the repository.

### The mutants

Broken through `go test -overlay` by plan 51's mutrun, the repository
never written (its code hashed before and after, the same), the
application's mutants against plan 67's tests alone
(`TestKeyText|TestAltGr|TestBranchesModifierPress`), the key overlay's
against `TestKeyTableNoCtrlAlt`. The run on the code before the last
comment edits: 23 mutants, 271 s, 23 caught by a failing test, the hash
e4345e98 before and after. On the final code: 23 mutants, 436 s (the
suite running beside it), 23 caught by a failing test, each by the test
below, the hash 13de3450 before and after.

| Mutant | Broken | Caught by |
| - | - | - |
| K1 | AltGr refused, the rule before plan 67 | TestKeyText: AltGr+2 |
| K2 | Ctrl or Alt alone typing | TestKeyText: Ctrl+k carrying a text |
| K3 | Alt alone typing | TestKeyText: Alt+k carrying a text |
| K4 | Ctrl alone typing | TestKeyText: Ctrl+k carrying a text |
| K5 | Shift and the locks counted | TestKeyText: AltGr+2 with Caps Lock |
| E1 | `editKey` with the rule before plan 67 | TestAltGrFields: the draft |
| E2 | `editKey` typing any key's text | TestAltGrFields: the draft |
| C1 | the card letting characters on | TestAltGrCard: the draft `@\#\|{}[]` |
| C2 | the card with the rule before plan 67 | TestAltGrCard: the draft `@\#\|{}[]` |
| B1 | the branches popup closing on a modifier's own press | TestAltGrFields: the branches popup's search |
| B2 | the popup's search with the rule before plan 67 | TestAltGrFields: the branches popup's search |
| B3 | the popup's search typing any key's text | TestAltGrFields: the branches popup's search |
| S1 | the search bar typing any key's text | TestAltGrFields: the search bar |
| S2 | the search bar refusing AltGr | TestAltGrFields: the search bar |
| F1 | the file viewer's fields typing any key's text | TestAltGrFields: the file viewer's search field |
| F2 | the quick open typing any key's text | TestAltGrFields: the quick open |
| G1 | the scope popup typing any key's text | TestAltGrFields: the scope popup's search |
| M1 | no modifier key known | TestAltGrFields: the branches popup's search |
| M2 | Shift no modifier | TestBranchesModifierPress: leftshift's own press |
| M3 | the left Ctrl no modifier | TestAltGrFields: the branches popup's search |
| M4 | the right Alt no modifier | TestAltGrFields: the branches popup's search |
| M5 | an ISO level shift no modifier | TestBranchesModifierPress: isolevel5shift's own press |
| T1 | a Ctrl+Alt chord in the key overlay's table | TestKeyTableNoCtrlAlt |

No mutant was killed by the compiler or without a failing test named.

### The checks

| Check | Result |
| - | - |
| build, vet (Windows, Linux), gofmt | ok, ok, ok, nothing printed |
| `go test ./...`, after the last code edit | 16 packages ok; internal/app 273.4 s, internal/input 367.3 s, internal/render 48.9 s |
| the mutants, mutrun, on the final code | 23: all caught by a failing test, 436 s |
| the code's hash, before and after the mutants | the same, 13de3450 |
| `-race` | internal/app, the 97 tests of the changed code, 62.9 s; internal/render, `TestKeyTableNoCtrlAlt`: no race |
| Windows Terminal, the final build | 208x59 and 120x40, runs A, B and C, each step as 67.4 says |
| the files | UTF-8 without a BOM, LF only (read from the bytes); no scratchpad path, user folder or leftover in the code |
| the timing step, alone, on mains power | below |

### The timing step

The command of evaluation.md, the twenty-one tests, alone, at 17:52:58,
93 s: on mains power, battery 98 %; CPU load over the 2 s before, 30, 80,
20, 23 and 13 %; the windows open, Chrome, Claude, Edge, Teams, Outlook,
Notepad, Task Manager, Move Mouse, Settings, Microsoft Lightbox; Defender
idle (0 s of CPU).
Plan 67 adds no timed test; its code on a frame's path is the branches
popup's key handler, whose frame is TestBranchesFrameCost's.

| Test | Measured | Target | Result |
| - | - | - | - |
| TestKeystrokeToFrame | 10,000-line draft p95 1.01 ms; in the draft window 1.11 ms | 16 ms | PASS |
| TestDraftKeystrokes | 50,000 lines, 8 saves, p95 2.06 ms | 16 ms | PASS |
| TestKeystrokeCost | reports only | | PASS |
| TestLongSession | a delta at the end, p95 2.94 ms | 16 ms | PASS |
| TestThinkingFrame | p95 2.51 ms | 16 ms | PASS |
| TestPasteToFrame | the worst 3.75 ms | 100 ms | PASS |
| TestStreamToFrame | p95 5.00 ms | 16 ms | PASS |
| TestSelectorToFrame | p95 1.00 to 1.51 ms | 16 ms | PASS |
| TestMotionToFrame | p95 0.68 to 1.54 ms; idle motion all dropped | 16 ms | PASS |
| TestDiffToFrame | p95 1.00 to 1.23 ms | 16 ms | PASS |
| TestMessageToFrame | p95 1.12 and 1.55 ms | 16 ms | PASS |
| TestOutputToFrame | p95 1.01 ms | 16 ms | PASS |
| TestToolsToFrame | p95 2.53 ms | 16 ms | PASS |
| TestBannerToFrame | p95 1.632 ms, max 3.36 ms, 11,355 bytes a frame | 2 ms | PASS |
| TestHomeKeysToFrame | p95 0.16 ms | 16 ms | PASS |
| TestHomePictureTime | the worst 209 ms | 260 ms | PASS |
| TestBranchesFrameCost | p95 2.32 ms | 16 ms | PASS |
| TestBlameFrameCost | p95 1.12 ms | 16 ms | PASS |
| TestGraphFrameCost | p95 0.77 ms | 16 ms | PASS |
| TestTurnReviewFrameCost | p95 1.64 ms | 16 ms | PASS |
| TestSearchToFrame | keys, two hours: p50 3.14, p95 5.28, max 7.59 ms; answers ten times longer: p50 7.53, p95 14.05, max 20.9 ms; deltas, the bar open: p50 2.64, p95 4.16, max 5.80 ms | 16 ms | PASS |

### Windows Terminal

Windows Terminal 1.23.20211.0, ACC's profile, maximised (208x59) and at
120x40 (`-mode 90x30` at the profile's 9 pt), the Belgian AZERTY layout
(fr-BE, `0000080C`), a temporary home and app data; each AltGr press the
right Alt as an extended key (`keybd_event`, `0xA5`), Windows adding the
left Ctrl as a physical press does; the records traced (TEA_TRACE). On a
demo repository (`scratchpad\term\p59demo.ps1`):

- **Run A**, the mock dashboard (`p67runs.ps1`). AltGr with 2, <, 3, 1, 4,
  0, 5 and $ in the draft: `@\#|{}[]`. AltGr+= then Space: "~"; ^ then E:
  "ê"; AltGr+µ then A: "à". Ctrl with the left Alt and <: nothing, its
  records with 0x2A (`\x1b[226;0;92;1;42;1_`). Ctrl with the left Alt and
  2: the 2 key's press never reached the dashboard, Windows Terminal's own
  key; its release did (`\x1b[50;3;64;0;42;1_`), as Ctrl+V's does (plan
  32), and a release types nothing. Ctrl+G, AltGr+2: "@" in the draft
  window. Ctrl+F, AltGr+2: "/ @" in the search bar, the draft unchanged.
  Source Control's tile, its message box, AltGr+2: "@" in the box, the
  draft unchanged.
- **Run B**, the startup screen with `--project` (`p67runs.ps1`; no CLI on
  the run's PATH, both shown as not installed). Tab to the project field,
  AltGr+2: the path ends with "@". Backspace; Ctrl+W, n, AltGr+2: "Branch
  @" in the Worktrees view's new branch.
- **Run C**, the branches popup (`p67runc.ps1`). A mock reads no branch
  (it is read when a project is validated) and plan 67 has no live run: a
  stand-in `claude.exe` built in the scratchpad
  (`scratchpad\plan67\standin`) answers the detection's `-v` with a version
  and then reads its input and writes nothing. Enter validated the demo
  repository and started the session, which waited for its handshake;
  "branch main" in the header; Alt+B opened the popup. AltGr+2: "@" in its
  search; Shift+F: "@F"; Alt+B closed it; Alt+B opened it again, its search
  empty; Ctrl with the left Alt and <: closed it, a key it does not take,
  nothing typed. The draft empty throughout. No CLI ran.

| Pass | Build | 208x59 | 120x40 |
| - | - | - | - |
| A | 16:45, before the fix | all as above | the Ctrl+F lost before the dashboard (the reader's race), so no bar and its "@" in the draft; the rest as above |
| A2 | 16:45 | - | all as above |
| B | 16:45 | all as above | all as above |
| C | 16:45 | the popup closed by AltGr's own presses, "@" in the draft (the finding) | the same |
| C2 | 17:18, the fix | "@" in the search; then Ctrl with the left Alt and < closed the popup before Shift+F and Alt+B (the steps' order, changed in C3) | the same |
| C3 | 17:18 | all as above | all as above |
| A3, B3, C4 | 17:39, the final code | all as above | all as above |

The shots are in `scratchpad\term\out-p67\`, a folder each pass and size
(`altgrA3-max`, `altgrB3-90x30`, `altgrC4-90x30` and the others), each step
a PNG and its text, with the run's `log.txt`; the traces beside them,
`trace-<pass>-<size>.log`. The ones to look at first: `altgrA3-max\draft`
(the eight characters and the composed ones), `altgrA3-max\f1` (the bar),
`altgrA3-max\sc2` (the message box), `altgrB3-90x30\b3` and `b9`,
`altgrC4-90x30\c4` (the popup's "@F"). No Claude Code or Codex session
ran in a terminal before the runs (the desktop app and this session
only), so none was closed.
