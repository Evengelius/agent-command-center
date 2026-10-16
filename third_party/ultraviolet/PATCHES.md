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
  out. Every file is byte for byte the module's, except the two changes below
  and `patch_test.go`, which is this project's (Checks).

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

## Changes

1. `decoder.go`, `parseWin32InputKeyEvent`: the call to `ensureKeyCase` and
   the function are removed. The console's `UnicodeChar` already reflects
   Shift, Caps Lock and the keyboard layout; recasing it from the control key
   state contradicted it.
2. `terminal_reader.go`, `deserializeWin32Input`: a UTF-16 surrogate is
   reassembled whatever its virtual key (`vk == 0 || utf16.IsSurrogate(uc)`);
   Windows Terminal sends both halves of an emoji with vk 255, which the
   original `vk == 0` test passed to the decoder as two lone surrogates.

Each change is marked `PATCHED (agent-command-center, PATCHES.md, patch N)`.

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
- The module's own tests, run on a scratch copy with these two files: all
  pass, after removing the one subtest of the deleted `ensureKeyCase`
  (`TestWin32Functions/ensureKeyCase`).
- The Windows Terminal run of `doc\evaluation.md` (keys K1 to K5, O2).

## Updating

When Bubble Tea moves to another ultraviolet version, check whether upstream
fixed both cases; if so, drop the `replace` and this folder, otherwise copy the
new version's files and reapply the two changes.
