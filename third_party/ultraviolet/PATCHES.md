# Patched copy of ultraviolet

## Source

- Module: `github.com/charmbracelet/ultraviolet`
- Version: `v0.0.0-20261008173134-6b8d4baf91b4` (commit `6b8d4baf91b4`), the
  version Bubble Tea v2.1.0 requires; `go.mod` replaces it with this folder.
- Licence: MIT, Copyright (c) Charmbracelet, Inc.; `LICENSE` is the module's
  own, unchanged.
- Copied: the module's `LICENSE`, `go.mod`, `go.sum` and the non-test `.go`
  files of its root package, the only one this application builds
  (`go list -deps ./...`). The subpackages and the module's tests are left
  out. Every file is byte for byte the module's, except the five changes
  below and `patch_test.go` and `patch_windows_test.go`, which are this
  project's (Checks).

## Why

Windows Terminal reports Shift+Enter as a plain Enter unless the application
asks for win32-input-mode (`CSI ?9001h`, doc\ui-design.md, Shift+Enter).
With that mode on, ultraviolet's decoder mishandled two kinds of key that
Windows Terminal 1.23 sends (recorded on 2026-10-16 with `TEA_TRACE`; the
sequences are the constants of `internal\app\term_test.go`):

| Typed | Sequence (key down) | Unpatched | Patched |
| - | - | - | - |
| "Z" injected (no Shift) | `ESC[90;0;90;1;32;1_` | `z` | `Z` |
| Caps Lock+Shift+z | `ESC[90;44;122;1;176;1_` | `Z` | `z` |
| 🙂 (U+1F642) | `ESC[255;0;55357;1;32;1_ESC[255;0;56898;1;32;1_` | dropped | `🙂` |

Patch 3 (plan 43, 2026-10-24): Bubble Tea clears its cell buffer before it
draws each frame, which marks every row touched, and the renderer repaints a
touched row holding a wide cell, or a cell the terminal may measure otherwise
(`lineHasDrift`), from column 0 with an erase to the end of the line, changed
or not (`paintLine`, `repaintLine`). Text written over a Sixel image erases
it (probe P32b), so an image the dashboard draws on such a row vanished with
the next frame and came back only when a frame changed the row's text: plan
43's review found the sidebar's tiles erased by a typed key beside a list of
✅ in zone B, and by each spinner frame of a running turn. Upstream's `main`
repaints these rows the same way (read 2026-10-24).

Patches 4 and 5 (plan 68, 2026-10-29): on Windows the reader runs two
goroutines, `streamData`, which reads the console's input records, and
`StreamEvents`, which scans them into events. `streamData` sent a batch's
bytes, then reset its buffer and wrote the next batch into the same array,
while `StreamEvents` might not have copied the first yet (it logs a read
before it copies it): its copy then held the next batch's bytes, a key
lost and the next doubled. Plan 67's Windows Terminal runs lost a Ctrl+F
that way (`doc\plan67-evaluation.md`, Details, The reader race's
evidence). `streamData` also used the receiver's grapheme state, flushing
it at each batch's end and, with VT input mode off, filling it, while
`StreamEvents` fills and flushes it with an emoji's two halves: the race
detector reports both. Upstream's `main` has the same code (read
2026-10-29).

## Changes

1. `decoder.go`, `parseWin32InputKeyEvent`: the call to `ensureKeyCase` and
   the function are removed. The console's `UnicodeChar` already reflects
   Shift, Caps Lock and the keyboard layout; recasing it from the control key
   state contradicted it.
2. `terminal_reader.go`, `deserializeWin32Input`: a UTF-16 surrogate is
   reassembled whatever its virtual key (`vk == 0 || utf16.IsSurrogate(uc)`);
   Windows Terminal sends both halves of an emoji with vk 255, which the
   original `vk == 0` test passed to the decoder as two lone surrogates.
3. `terminal_renderer.go`, `paintLine`, with `changedEnd` and `repaintTo`: a
   drift-prone row that the render neither forces (`repaintAll`) nor
   disturbed itself (`damaged`) is first compared with the model. With no
   cell changed nothing is written; otherwise its cells are written from
   column 0 through the last changed one, a wide cell there whole, with no
   erase, and the cells after it, which match the model, are left as they
   are. A forced or disturbed row, a row of another width than the buffer's,
   or one with a cell past the right margin before the change is repainted
   whole, as upstream does (`repaintLine`, unchanged); the autowrap-off path
   and the re-anchor are upstream's. A terminal that measured a glyph of the
   rewritten part otherwise than the model would leave the cells after it in
   place instead of shifting them; Windows Terminal reports Unicode core mode
   (2027) permanently set, so Bubble Tea's cell buffer measures grapheme
   widths, as Windows Terminal does.
4. `terminal_reader_windows.go`, `streamData`: each batch is sent as a copy
   (`bytes.Clone`), the receiver's to keep, and the buffer is the sender's
   again at once.
5. `terminal_reader_windows.go`, `streamData` and
   `serializeWin32InputRecords`: the sender's grapheme state is a scanner
   of its own (`newEventScanner()`, passed to
   `serializeWin32InputRecords`), in place of the receiver's
   `d.eventScanner`, for its three grapheme calls.

Each change is marked `PATCHED (agent-command-center, PATCHES.md, patch N)`,
or `PATCHED, patch N` in the comment of a function patch 3 adds and on the
lines patch 5 changes.

## Checks

- `patch_test.go`, `TestPatchedWin32Input` (standard library only): the four
  cases of the removed upstream subtest `TestWin32Functions/ensureKeyCase`,
  same names, fed as win32-input-mode sequences through the scanner, plus
  Caps Lock+Shift and an emoji with vk 255 and with vk 0. It checks the text,
  code, shifted code and modifiers of each key. On the unpatched files it
  fails exactly the three cases the patches change ("uppercase without
  shift", "lowercase with caps lock and shift", "surrogate pair, vk 255").
  The upstream subtest called `ensureKeyCase` on a hand-built key
  (`Code 'a', Text "A"`) that the decoder never produces: for a real Shift+A
  the decoder sets the code from the console's character, `'A'`, patched or
  not. `internal\licenses` runs this module's tests, since `go test ./...`
  skips a nested module; breaking patch 2 fails it.
- `internal\app\TestWin32InputKeys` runs the real program on the recorded
  sequences: it fails on the unpatched module (`"zéZZ\né"`) and passes with
  this copy (`"Z🙂éZz\né"`).
- `patch_test.go`, `TestPatchedDriftRows`: frames rendered as Bubble Tea
  renders them, with and without grapheme widths. An unchanged row with a
  wide cell writes nothing; a change at its left writes the row from column 0
  through the change, with no erase and not the tail; the next identical
  frame writes nothing (the model followed); a change ending on a wide cell
  writes it whole and nothing after it; a change in the last column writes
  the row whole; a resize repaints it whole. On the unpatched file it fails
  the first four checks on both paths.
- `internal\app\TestWideRowsKept` runs the real program at 208x59 with Sixel,
  on a mock whose answer puts a list of ✅ beside both of the sidebar's
  tiles: a key typed once the turn is over, and the spinner's frames while
  it runs, write none of those rows. With the unpatched file (`go test
  -overlay`), the key's frame and 24 spinner frames rewrote them.
- `patch_windows_test.go`, `TestConsoleReader` (Windows only): the reader's
  real console path. Each scenario runs in a child process the test starts
  with `CREATE_NO_WINDOW`, which has a console of its own and no window,
  killed at 60 s; the child writes key records into that console with
  `WriteConsoleInputW`, reads them back through `conInputReader`,
  `streamData` and `StreamEvents`, and the parent compares the events
  whole. The scenarios: Ctrl+F in plan 67's two reads, the receiver held
  in its log of the first; 200 letters one read at a time, the receiver
  held 20 ms in every fifth log; an emoji between two letters in one read,
  split between two reads, cut inside a sequence, and as two surrogate
  records; with VT input mode off, letters, `é` and an emoji split between
  two reads. On the unpatched file the first two fail as plan 67's trace
  showed (`release ctrl+f`, an unknown sequence, `release ctrl+f`, `release
  leftctrl`; every fifth letter lost and the next doubled). Under `-race`
  every scenario reports patch 4's race, and the four that reach the
  grapheme state patch 5's; with patch 4 alone, patch 5's alone.
- `internal\licenses` requires each of this module's tests to report PASS,
  none skipped (`copyTests`), so that a test file left out by a build
  constraint fails it.
- The module's own tests, run on a scratch copy with the patched files:
  all pass (patches 1 and 2 on 2026-10-16, patch 3 on 2026-10-24, patches 4
  and 5 on 2026-10-29, those also under `-race`), after removing the one
  subtest of the deleted `ensureKeyCase` (`TestWin32Functions/ensureKeyCase`).
- The Windows Terminal run of `doc\evaluation.md` (keys K1 to K5, O2).

## Updating

When Bubble Tea moves to another ultraviolet version, check whether upstream
fixed the five cases; if so, drop the `replace` and this folder, otherwise
copy the new version's files and reapply the changes it still needs.
