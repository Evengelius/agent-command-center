# Image paste and preview in Windows Terminal (plan 32, 2026-10-23)

What plan 32 needed to know before its design: which keys reach the
dashboard when the clipboard holds an image, a copied file or text; what
the clipboard library reads; whether Windows Terminal can draw an image
and how; how Bubble Tea's renderer treats the cells an image covers.
Every claim names its source and the date it was read. The probes ran on
the owner's machine, Windows 11, Windows Terminal 1.23.20211.0
(`Get-AppxPackage Microsoft.WindowsTerminal`); their program and harness
are outside the repository (`scratchpad\plan32\probe-tree\cmd\p32probe`,
`scratchpad\term\p32probe.ps1`), their output in
`scratchpad\term\out-p32probe-keys-z0\` and `out-p32probe-sixel-z0\`.

## Keys: probe P32a

The probe put the console in raw mode with bracketed paste and
win32-input-mode on, as the dashboard does, and logged every chunk read.
The harness saved the owner's clipboard, put on it in turn an image
(`Clipboard.SetImage`: formats `System.Drawing.Bitmap`, `Bitmap`), a file
(`SetFileDropList`: `FileDrop`, `FileNameW`, `FileName`) and text, sent
each of Ctrl+V, Alt+V, Ctrl+Shift+V and Ctrl+Alt+V to the window, then
restored the clipboard (its formats before: Bitmap, PNG and two of the
Windows clipboard history's; after: the same).

The owner's `settings.json` binds `ctrl+v` to
`Terminal.PasteFromClipboard`; Windows Terminal binds `ctrl+shift+v` to
the same by default. What reached the probe, in win32-input-mode's
encoding (`ESC[Vk;Sc;Uc;Kd;Cs;Rc_`, Kd 1 a press, 0 a release):

| Clipboard | Ctrl+V | Alt+V | Ctrl+Shift+V | Ctrl+Alt+V |
| - | - | - | - | - |
| an image | Ctrl's press and release, and V's release only (`ESC[86;47;22;0;40;1_`) | Alt, V pressed and released | as Ctrl+V, Shift too | Ctrl, Alt, V pressed and released |
| a copied file | its full path as a bracketed paste, V's release | Alt, V pressed and released | the path, V's release | the keys |
| text | the text as a bracketed paste, V's release (3 ms before the paste) | the keys | the text, V's release | the keys |

So Windows Terminal keeps Ctrl+V's press for its own paste, pastes
nothing when the clipboard holds no text, and forwards the V key's
release anyway; that release is undocumented behaviour, observed only.
Alt+V and Ctrl+Alt+V are not bound by the terminal and reach the
application as keys. Claude Code gives its image paste to Alt+V on
Windows for this reason: its users report Ctrl+V absorbed by Windows
Terminal and Alt+V working
(https://github.com/anthropics/claude-code/issues/59661 and
https://claudeissues.com/issue/24092-bug-image-paste-shortcut-appears-swapped-between-windows-and-mac-ctrl-v-vs-alt-v,
read 2026-10-23); a Codex fork's patch for WSL says the same of Ctrl+V
and uses Ctrl+Alt+V
(https://forge.lthn.ai/core/core-agent-ide/commit/ce0b38c05613bda491b3aed0dfd9f9dca4c6e558,
read 2026-10-23).

## The clipboard library

`golang.design/x/clipboard` v0.11.0, the approved dependency
(`clipboard.go`, `clipboard_windows.go` in the module cache, read
2026-10-23): `Read(ctx, FmtImage)` returns PNG bytes whatever the copying
application wrote, from `CF_DIBV5`, else `CF_DIB`; `Register("image/png")`
names the Windows clipboard format "PNG", which "Chromium, Firefox,
Microsoft Office, Snip & Sketch and Paint" publish with the original PNG
bytes, read as they are; `ReadFiles(ctx)` returns the paths of copied
files (`CF_HDROP`); `Write` and `WriteFiles` put an image or files on the
clipboard. No cgo on Windows.

## Images: probe P32b

The probe drew the generated test image (`internal/input/imagetest`) as
Sixel (DCS `q`, raster attributes `"1;1;W;H`, a 216-colour palette) at
known cells over a numbered grid, then wrote text and spaces over parts
of it; the screenshots are the evidence.

- Windows Terminal 1.23 draws Sixel images.
- A cell holds 10x20 of the image's pixels, whatever the font: 200x100
  pixels covered 20 columns by 5 rows, 320x160 covered 32 by 8, 100x100
  covered 10 by 5, at a cell of 12x23 screen pixels; the terminal scales
  the image to the cells.
- Text written into cells the image covers erases the image there and
  shows the text; spaces erase it too; cells left alone keep it.

## The renderer

Bubble Tea v2.1.0 (module cache, read 2026-10-23) and the ultraviolet
version it requires (this repository's `third_party/ultraviolet`):

- `terminal_renderer.go`, `transformLine` and `putRange`: a changed line
  is rewritten from its first to its last changed cell, the unchanged
  cells between them included (the skip of unchanged runs needs more of
  them than the range holds); blank runs may go out as erase sequences.
  So a frame that changes a row on both sides of an image, or the image's
  own cells, erases the image there.
- `tea.go`, `startRenderer`: at each frame tick the program writes the
  raw sequences queued with `tea.Raw` (`p.flush`), then the frame
  (`p.renderer.flush`); a sequence queued after a frame has gone out
  lands before the next frame's changes.
- `input.go`, `translateInputEvent`: events it does not know are passed
  to the model as they are; ultraviolet decodes the answer to the primary
  device attributes request (`ESC[c`) as `PrimaryDeviceAttributesEvent`,
  a list of numbers, 4 meaning Sixel graphics (`decoder.go`, `event.go`).

## What follows for the dashboard

- Alt+V is the paste key that always works; Ctrl+V works for text as
  before, and can attach an image through the forwarded release, labelled
  undocumented; a copied file pasted as its path can be recognised by
  comparing the paste with the clipboard's file list.
- A pasted image is PNG bytes in memory, from the "PNG" format when
  present, else the bitmap.
- An inline preview is possible: the frame keeps the cells blank and the
  dashboard draws the images over them after the frame is out, again
  after any frame that changes those rows. The thumbnail's size is in
  cells: 10 columns hold 100 pixels across and 5 rows 100 down.
