# Terminal input, clipboard and selection (amendment 3)

Plan 1, step 5. This file covers two things for the multiline editor of
zone B:

- what each candidate UI library offers;
- what each terminal and OS allows for paste, Enter and newline keys,
  clipboard, mouse selection, tmux and `$EDITOR`.

Written 2026-10-15 and 2026-10-16, before any code. Scope follows amendment
5: only Claude Code and Codex CLI. Gemini CLI and Antigravity are not covered.

Conventions:
- **unverified**: no source confirmed the claim, or it rests only on a
  secondary or third-party source.
- **undocumented**: the behaviour exists in code but no official
  documentation states it.
- `repo@commit path:line` cites a shallow clone read on 2026-10-15. Page
  citations give the URL and section; every page was accessed on 2026-10-15
  unless another date is given.
- Nothing was built or run. Every finding comes from reading code and pages;
  the manual checks are listed in section 10.

## 1. Sources

### Repositories (cloned and read only)

| Repo | Commit (tag) | Date | License (SPDX) |
|---|---|---|---|
| github.com/charmbracelet/bubbletea (v2) | f9df43c (v2.1.0) | 2026-10-08 | MIT |
| github.com/charmbracelet/bubbletea (v1 branch) | 9edf69c (v1.3.10) | 2025-09-17 | MIT |
| github.com/charmbracelet/bubbles | c3bde0d (after v2.2.1) | 2026-10-01 | MIT |
| github.com/charmbracelet/lipgloss | 19f011f | 2026-10-09 | MIT |
| github.com/charmbracelet/ultraviolet | 6b8d4ba | 2026-10-08 | MIT |
| github.com/charmbracelet/x (ansi) | 43427fd | 2026-10-12 | MIT |
| github.com/aymanbagabas/go-osc52 | 5c9b696 | 2023-06-20 | MIT |
| github.com/atotto/clipboard | 6a3273b | 2026-10-02 | BSD-3-Clause |
| github.com/golang-design/clipboard | d95c794 (v0.11.0) | 2026-09-26 | MIT |
| github.com/rivo/tview | c15b79f | 2026-08-11 | MIT |
| github.com/gdamore/tcell | 4bed180 (v3.6.0) | 2026-10-10 | Apache-2.0 |
| github.com/rhysd/tui-textarea | 4d18622 (0.7.0) | 2024-12-01 | MIT |
| github.com/Textualize/textual | 5e5b7ef (v8.2.8) | 2026-10-09 | MIT |
| github.com/vadimdemedes/ink | 26d2c3f (v8.0.0) | 2026-10-03 | MIT |
| github.com/openai/codex | 1ef30bc | 2026-10-15 | Apache-2.0 (from `doc/research/codex-cli.md`) |

### Pages (accessed 2026-10-15)

- **Windows Terminal (learn.microsoft.com/windows/terminal):**
  - customize-settings/interaction (doc commit 0cf143a);
  - customize-settings/actions (doc commit 5b3d5eb);
  - selection, section "Mouse support" (updated 2025-11-12).
- **Windows Terminal (github.com/microsoft/terminal):**
  - PRs #5823, #9034, #15155, #18949, #19034, #19817, #20728 and issue #2946;
  - releases page (v1.25.2733.0, 2026-10-02);
  - Windows Command Line blog post on Preview 1.25 (2026-03-05);
  - source, read on GitHub at main on 2026-10-15: `src/types/utils.cpp`
    (`FilterStringForPaste`), `src/cascadia/TerminalControl/ControlCore.cpp`
    (`PasteText`), `src/terminal/parser/OutputStateMachineEngine.cpp`
    (OSC 52), and `src/host/VtIo.cpp`.
- **Windows console:** learn.microsoft.com/windows/console/setconsolemode.
- **tmux:**
  - github.com/tmux/tmux `options-table.c`, `key-bindings.c` and `CHANGES`
    (master);
  - wiki pages "Clipboard" and "Modifier-Keys";
  - man tmux (man7.org).
- **Claude Code (code.claude.com/docs/en):**
  - terminal-config, sections "Enter multiline prompts", "Configure tmux"
    and "Paste large content";
  - interactive-mode, sections on keyboard shortcuts and multiline input;
  - fullscreen, sections "Keep native text selection" and copy behaviour.
  - Issues github.com/anthropics/claude-code #18990, #24077 and #80184.
- **Codex CLI:** github.com/openai/codex PR #16966 (merged 2026-04-09) and PR
  #7606 (merged 2025-12-22); learn.chatgpt.com/docs/codex/cli.
- **Other terminals:**
  - can-i-use-terminal.github.io/features/osc52copy.html;
  - sw.kovidgoyal.net/kitty/conf;
  - wezterm.org: escape-sequences.html (OSC 52) and
    config/lua/config/bypass_mouse_reporting_modifiers.html;
  - alacritty.org/config-alacritty.html, sections TERMINAL and MOUSE;
  - codeberg.org/dnkl/foot `foot.ini`;
  - invisible-island.net: xterm(1) and ctlseqs;
  - iterm2.com/documentation-preferences-general.html;
  - bugs.kde.org #372116 and #513274.
- **Editors and other TUIs:**
  - textual.textualize.io: FAQ and api/app;
  - zellij.dev/documentation/options.html;
  - lazygit `docs/Config.md`;
  - docs.helix-editor.com/editor.html;
  - neovim `runtime/doc/provider.txt`;
  - git-scm.com/docs/git-var;
  - code.visualstudio.com/docs/configure/command-line.
- **Secondary sources:** elevenforum and windowsnews.ai for WT
  `compatibility.allowOSC52`; codex.danielvaughan.com (2026-08-28) for Codex
  `/copy`; pi.dev/docs/latest/tmux.

## 2. Candidate UI libraries

### 2.1 Comparison

| Library | Selection | Cursor moves | Undo | Very long input | Bracketed paste | License |
|---|---|---|---|---|---|---|
| **Bubble Tea v2** (runtime) | n/a | n/a | n/a | n/a | On by default; `PasteMsg` | MIT |
| **Bubble Tea v1** | n/a | n/a | n/a | n/a | Unix only; `KeyMsg.Paste` | MIT |
| **Bubbles textarea** (v2 line) | Shift-based selection plus mouse API, since v2.2.1 | Full set | **None** | Capped at 10 000 lines; O(n) render | Through the sanitizer, which is lossy | MIT |
| **Lip Gloss v2** | n/a (layout); `Compositor.Hit` | n/a | n/a | n/a | n/a | MIT |
| **tview TextArea** (tcell) | Shift and mouse | Full set | Yes (Ctrl+Z / Ctrl+Y) | Piece chain | Verbatim | MIT (tcell Apache-2.0) |
| **tui-textarea** (Ratatui, Rust) | Yes | Full set | Yes, 50 steps by default | No soft wrap | termwiz `Key::Paste` only | MIT |
| **Textual TextArea** (Python) | Yes | Full set | Yes, 50 checkpoints | Soft wrap | Verbatim `_on_paste` | MIT |
| **Ink** (React, Node) | None in core | None in core | None in core | n/a | `usePaste` hook | MIT |

### 2.2 Bubble Tea v2 (charm.land/bubbletea/v2, bubbletea@f9df43c)

`tea.View` (tea.go:84) declares the terminal modes the program wants:

- **Bracketed paste:** on unless `DisableBracketedPasteMode` is set
  (tea.go:186-187). The renderer writes `SetModeBracketedPaste`
  (cursed_renderer.go:141). A paste arrives as one `PasteMsg{Content}`, framed
  by `PasteStartMsg` and `PasteEndMsg` (paste.go).
- **Mouse:** `MouseMode` is None, CellMotion or AllMotion (tea.go:189-191).
  The renderer enables 1002 or 1003 with SGR 1006 (cursed_renderer.go:146-151).
  The `OnMouse` callback is at tea.go:98-126.
- **Keyboard enhancements:** `KeyboardEnhancements` (tea.go:193-203). Its doc
  comment says "shift+enter" becomes distinguishable when the terminal
  supports them. The renderer always requests both xterm modifyOtherKeys 2 and
  the Kitty protocol (`PushKittyKeyboard`) (cursed_renderer.go:165-170,
  429-436). Support is reported back in `KeyboardEnhancementsMsg`, with
  `SupportsKeyDisambiguation()` and similar helpers (keyboard.go).
- **Clipboard over OSC 52:**
  - `SetClipboard` (clipboard.go:30) and `ReadClipboard` (clipboard.go:43),
    plus their primary-selection variants;
  - replies arrive as `ClipboardMsg{Content, Selection}`;
  - they are sent as `ansi.SetSystemClipboard` and `RequestSystemClipboard`
    (tea.go:834-844), with no tmux passthrough wrapping.
- **Windows input:**
  - Sets `ENABLE_VIRTUAL_TERMINAL_INPUT` (tty_windows.go:30); suspend is not
    supported (tty_windows.go:62).
  - Never enables win32-input-mode. `ansi.SetModeWin32Input`
    (x/ansi@43427fd ansi/mode.go:687) exists but is unused.
  - ultraviolet can decode win32-input key records
    (ultraviolet@6b8d4ba decoder.go:525-540, 1841), and its paste buffer
    handles them (terminal_reader.go:298-308). An application could therefore
    request mode 9001 itself with `tea.Raw` (raw.go:33). This is unverified:
    see section 4.
- **External programs:** `tea.ExecProcess` (exec.go:50); see section 8.

The paste decoder (ultraviolet@6b8d4ba terminal_reader.go:289-365):

- **Buffering:** while a paste is open, key text is appended as is. Keys
  without text, such as CR or tab, are appended as their raw sequence bytes.
  The whole buffer is flushed as a single event at paste end.
- **Data loss at paste end (lines 356-363):** the buffer is decoded rune by
  rune and every `utf8.RuneError` is dropped. Invalid UTF-8 is lost, and so is
  a literal U+FFFD. The application cannot recover either.
- **Ignored events:** unknown events inside a paste are dropped
  (`default:` at line 326).

**Windows console reader** (terminal_reader_windows.go:20-60, 88-110). It
reads with `ReadConsoleInput`. In VT input mode it keeps only `KeyEvent.Char`,
so Shift+Enter is distinguishable only if the console itself produces an
escape sequence for it.

The console-mode setup differs by path, and which one Bubble Tea v2 takes in
conhost is unverified:

| Path | Console mode set |
|---|---|
| `preparePollConsole` (poll_windows.go:179-202) | Clears `ENABLE_MOUSE_INPUT`; sets `ENABLE_QUICK_EDIT_MODE` and VT input |
| Cancel reader (cancelreader_windows.go:52-55) | Sets VT input, window input and extended flags |

### 2.3 Bubble Tea v1 (bubbletea@9edf69c, v1.3.10)

- **Paste:** a paste arrives as `KeyMsg` with `Paste bool` (key.go:58), and
  bracketed paste is enabled by `EnableBracketedPaste` (screen.go:124).
- **Windows input:** v1 reads `KEY_EVENT_RECORD`s through coninput, with no
  VT and no bracketed paste (key_windows.go:12-57). A pasted newline therefore
  arrives as an Enter key and cannot be told apart from typing.
- **Missing:** no Kitty protocol and no OSC 52 commands.
- **Verdict:** v1 does not meet amendment 3 on Windows.

### 2.4 Bubbles textarea (bubbles@c3bde0d, textarea/textarea.go)

**Selection and keys.** Selection arrived in v2.2.1.

- **Selection API** (selection.go): `PositionAt` (line 63), `BeginSelection`,
  `ExtendSelection` and `EndSelection` (lines 105-128), `SelectAll` (136),
  `ClearSelection` (149), `HasSelection` (157), `Selection` (163) and
  `SelectedText` (176). The application wires the mouse to these calls itself.
- **Selection keys** (KeyMap, lines 89-126): Shift+arrows select by character
  and Ctrl/Alt+Shift+Left/Right by word; Ctrl+G selects all.
- **Editing and movement keys:**
  - Enter and Ctrl+M insert a newline;
  - Alt or Ctrl+arrows move by word;
  - Home/End and Ctrl+A/E go to line start and end;
  - PgUp/PgDn and Ctrl+Home/End move by page and to the document ends.
- **Copy and paste:** Ctrl+Shift+C copies and Ctrl+V pastes, both through
  atotto: `clipboard.WriteAll` (around line 1835) and `clipboard.ReadAll`
  (2013).

**Unfit as-is.** Five findings rule it out for zone B:

1. **Lossy paste sanitizer.** Inserts go through `m.san().Sanitize`
   (lines 543-547), defined in internal/runeutil/runeutil.go:
   - tabs become 4 spaces (`replaceTab`, line 27);
   - `RuneError` is dropped (line 67);
   - **each** `\r` and each `\n` becomes `\n` (line 70), so CRLF becomes two
     newlines;
   - other control characters are dropped (line 88).

   The sanitizer is private (`rsan`, line 382, initialised at 858-864) and
   cannot be configured from outside. This breaks "content kept byte for
   byte".
2. **Silent truncation.** `maxLines = 10000` (line 37, with the comment "XXX:
   in v2, make max lines dynamic"). Inserts beyond it are cut without any
   sign (around lines 582-585). `defaultMaxHeight = 99` (line 33).
3. **Cost per keystroke.**
   - `view()` renders **every** line (1480 calls `m.Value()`; 1495 loops over
     all lines), and runs from both Update (1456) and View (1595).
   - Wrap lookups hash each line with sha256 over `fmt.Sprintf`
     (lines 269-272), and the memo cache is resized to MaxHeight (1266-1267).

   Each keystroke therefore costs O(total text). Issue #285, "textarea.View
   very slow", was closed in 2022; it was not reproduced here.
4. **No undo or redo.** A grep for undo and redo in textarea/ found nothing.
5. **Paste ignores CharLimit.** Issue #468, closed 2024-01. Issue #887 (cursor
   stuck while navigating) is still open (2026-02).

The viewport has `SetHighlights` (viewport.go:605) but no selection.

### 2.5 Lip Gloss v2 (lipgloss@19f011f) and ultraviolet

- **Hit testing:** `Compositor.Hit(x, y) LayerHit` (layer.go:289) returns the
  topmost layer with an ID at a point. This maps a mouse cell to a panel.
- **Reading screen cells:** an ultraviolet `Buffer` exposes `CellAt`
  (buffer.go:288) and `Line.String` (buffer.go:111). These read cells, not the
  panel's logical text, so selection should use the panel's content model
  (section 6).

### 2.6 tview TextArea (tview@c15b79f) on tcell v3 (tcell@4bed180)

- **Storage and editing:**
  - text is held in a piece chain (textarea.go:47-60);
  - undo is Ctrl+Z and redo Ctrl+Y;
  - Shift selects; mouse selection supports click, double-click and
    Shift+click;
  - text wraps.
- **Clipboard:**
  - copy is **Ctrl+Q**, because Ctrl+C is reserved;
  - the clipboard is internal unless the application supplies
    `SetClipboard(copy, paste)` (textarea.go:924);
  - `PasteHandler` inserts verbatim (around line 2424), and `NewLine = "\n"`.
- **tcell v3:**
  - delivers `EventPaste`;
  - offers OSC 52 `SetClipboard` and `GetClipboard` (tscreen.go:1852-1876),
    with reads disabled for st;
  - documents WezTerm quirks (tscreen.go:1586-1600);
  - uses win32-input-mode on Windows (README-windows.md).
- **Why it is only a reference:** it uses a different runtime from Bubble Tea
  and cannot be embedded in it. Its piece chain and undo design are worth
  studying (MIT).

### 2.7 Other stacks (for comparison; not candidates for a Go application)

- **tui-textarea 0.7.0** (Rust, Ratatui), tui-textarea@4d18622:
  - undo and redo with `set_max_histories` (default 50), selection, mouse
    scrolling and search;
  - a yank buffer rather than the system clipboard;
  - no soft wrap is listed in the README;
  - paste only through termwiz `Key::Paste`;
  - last commit 2024-12.
- **Textual v8.2.8**, textual@5e5b7ef:
  - **TextArea editing** (src/textual/widgets/_text_area.py):
    - Shift selection bindings (lines 288-330);
    - Ctrl+Z undo (line 411; `max_checkpoints=50`);
    - `soft_wrap` (line 514);
    - `_on_paste` inserts verbatim (line 1982).
  - **Clipboard:** `App.copy_to_clipboard` writes `ESC]52;c;…BEL`
    (app.py:1770-1786).
  - **App-wide selection:** arbitrary text selection, limited to one widget,
    since 2.0.0 (2025-02-16, PR #5409). `screen.copy_text` is bound to Ctrl+C
    and Super+C (screen.py:272, 985).
- **Ink v8.0.0**, ink@26d2c3f: a `usePaste` hook and Kitty keyboard support;
  no text area in core.

### 2.8 Other Go editors with selection

No maintained, permissively licensed Go component for Bubble Tea was found
that offers selection, undo and a scalable buffer together (unverified: based
on the libraries above plus the Bubbles issue tracker). The closest are the
Bubbles textarea (no undo, lossy paste, O(n) render) and tview's TextArea
(different runtime).

## 3. Bracketed paste

### 3.1 Per terminal

| Terminal | Bracketed paste | How newlines and tabs arrive | Source |
|---|---|---|---|
| Windows Terminal | Yes since Preview 1.7.572 (PR #9034, merged 2021-02-08) | Before wrapping, `FilterStringForPaste(…, CarriageReturnNewline \| ControlCodes)` runs. **CRLF becomes CR, and a lone LF becomes CR.** C0 and C1 controls are removed except HT, LF and CR; DEL and ESC are removed. Tabs are kept. | utils.cpp `FilterStringForPaste` (around 616-687); ControlCore.cpp `PasteText` |
| conhost (inbox console) | Only in VT input mode (PR #15155, merged 2023-04-26); ESC is stripped when bracketed | Same filter family (assumed; unverified). Which Windows 11 build ships it is unverified. | PR #15155 |
| VS Code terminal (xterm.js) | Yes (unverified: no xterm.js source read). Claude Code warns that VS Code "can drop characters on large pastes" | Unverified | terminal-config, "Paste large content" |
| tmux | `paste-buffer -p` brackets "if the application has requested bracketed paste mode"; `]` uses `-p` since 3.2. 3.4 recognises outer bracketed pastes. 3.7 adds a 5-second limit for lost paste-end sequences and a longer escape delay for partial ones, citing WT (issue 5088). 3.8 fixes a flag collision (issue 5622). How an outer paste is re-wrapped for the pane is unverified. | Passes bytes through (unverified) | man tmux; CHANGES |
| xterm, kitty, WezTerm, Alacritty, foot, iTerm2, GNOME/VTE, Terminal.app | Supported (unverified: general knowledge, not fetched) | Unverified per terminal; most send LF as CR | n/a |

**WT paste settings** (customize-settings/interaction):

| Setting | Default | Effect |
|---|---|---|
| `trimPaste` | true | "trim trailing whitespace characters when pasting". The exact rule was not read in code, so its effect on byte-for-byte paste is unverified. It must be tested by hand and documented as an owner setting. |
| `largePasteWarning` | true | Warns above 5 KiB |
| `multiLinePasteWarning` | true | Warns on a multiline paste |

Both warnings are dialogs in WT, before any byte reaches the application.
The application cannot suppress them, and it must not ask the owner to.

### 3.2 Guaranteeing that a paste never submits

1. **Accept pasted text only through `PasteMsg`, and insert it as text.**
   Never replay it as keys. The application must never run its submit
   binding on `PasteMsg` content. Bubble Tea v2 buffers the whole paste
   before emitting it (ultraviolet terminal_reader.go:289-365), so Enter keys
   inside a paste never reach the key handler.
2. **Paste without brackets** (Bubble Tea v1 on Windows, or a terminal
   without mode 2004) arrives as keystrokes. Codex CLI handles this case with
   a timing heuristic, `PasteBurst` (codex@1ef30bc
   codex-rs/tui/src/bottom_pane/paste_burst.rs:1-10):
   - "On some platforms (notably Windows), pastes often arrive as a rapid
     stream of `KeyCode::Char`, `KeyCode::Enter`…";
   - a burst is at least 3 characters within 8 ms;
   - Enter is suppressed for 120 ms after a burst (lines 159-170).

   With Bubble Tea v2 on WT, conhost in VT mode, or a Unix terminal, this
   heuristic should not be needed. Keep it as a documented fallback only if
   the manual check in section 10 finds a terminal that does not bracket.
3. **Paste end lost** (terminal or tmux bug). Until `PasteEndMsg` arrives,
   keep the editor in paste state and show it. Do not treat later keys as
   submit. tmux 3.7 uses a 5-second limit for the same case (CHANGES).

### 3.3 Byte-for-byte content

What can be preserved depends on the layer:

- **Terminal (WT):** it rewrites newlines (LF and CRLF become CR) and strips
  controls. The application cannot recover the original line endings or a
  stray ESC. This is a documented limit: store pasted CR as LF (one per
  newline), and keep tabs and every other rune.
- **Bubble Tea v2:** it drops invalid UTF-8 and U+FFFD (terminal_reader.go:
  356-363). Document it as a limit; fixing it would need an upstream change.
- **Application editor:** keep every rune it receives. No tab expansion and
  no control stripping in storage; render controls visibly instead (for
  example as `^X`). Store text as the user gave it and send exactly that to
  the CLI.

### 3.4 Thousands of lines without freezing

- **Event delivery:** one `PasteMsg` carries the whole paste, so the event
  loop gets a single event, not one per key.
- **Storage and render:** the editor must use a structure with cheap
  insertion (piece table or rope; tview's piece chain is a model). It must
  render only the visible rows and wrap lazily per visible line. The Bubbles
  textarea fails this (section 2.4).
- **How the CLIs show large pastes:**
  - Claude Code collapses pastes over 800 characters or 3 lines into
    `[Pasted text #N]` (terminal-config, "Paste large content").
  - Codex inserts a placeholder element, "expanded on submit", and keeps the
    full text in `pending_pastes` (codex@1ef30bc
    codex-rs/tui/src/bottom_pane/chat_composer/paste_input.rs:93, 146).

  A placeholder display is a reasonable option for zone B, but the stored
  text must stay complete.

## 4. Enter versus newline

### 4.1 Encodings

In legacy encoding:

- Enter sends CR (0x0D).
- Ctrl+J sends LF (0x0A), which is distinct from Enter in every terminal
  without any protocol.
- Alt+Enter sends ESC CR, which is also distinct.
- Shift+Enter and Ctrl+Enter send plain CR in most terminals, so they need
  the Kitty protocol, modifyOtherKeys or win32-input-mode to be told apart.

Bubble Tea v2 requests both the Kitty protocol and modifyOtherKeys (2.2).

| Terminal | Shift+Enter | Ctrl+Enter | Alt+Enter | Ctrl+J | Source |
|---|---|---|---|---|---|
| Windows Terminal ≥ 1.25 | Kitty protocol: PR #19817 (merged 2026-02-17); "allows applications to receive Shift+Enter" (Preview 1.25 blog). Stable v1.25.2733.0 shipped 2026-10-02 with Kitty fixes #20500, #20551 and #20052. **Whether the Kitty protocol reaches a native console app through ConPTY is unsettled** (a TODO in VtIo.cpp about not translating win32-input to the Kitty protocol). Claude Code says Shift+Enter works in WT "without setup" (terminal-config). | Same open question | **Taken by WT: fullscreen** (actions) | Works | as cited |
| WT with win32-input-mode (9001) requested by the app | Full key records, which ultraviolet decodes (decoder.go:525-540) | Same | Taken by WT | Works | Unverified that WT honours a 9001 request from the app; manual probe |
| conhost | Unverified | Unverified | Unverified | Works | n/a |
| VS Code terminal | Needs Claude Code's `/terminal-setup`, which adds a VS Code keybinding | Unverified | Unverified | Works | terminal-config |
| Ghostty, kitty, iTerm2, WezTerm, Warp, Apple Terminal | Works "without setup" | Unverified | macOS Option depends on the "Option as Meta" setting; Codex replaced Alt+C with Ctrl+O because "Alt is unreliable on macOS" (PR #16966) | Works | terminal-config; PR #16966 |
| foot; Alacritty ≥ 0.16 | Works through the Kitty protocol | Unverified | Unverified | Works | terminal-config |
| Alacritty < 0.16 | Needs setup | n/a | n/a | Works | terminal-config |
| GNOME Terminal (VTE), JetBrains | Not available | n/a | n/a | Works | terminal-config |
| tmux | Needs `extended-keys on` and `terminal-features 'xterm*:extkeys'` (Claude docs). Kitty protocol in tmux: not in any release (section 7) | Same | Works | Works | terminal-config, "Configure tmux" |

### 4.2 What the two supported CLIs do

- **Claude Code:**
  - newline keys: `\`+Enter, Option+Enter, Shift+Enter, Ctrl+J, and paste
    mode (interactive-mode);
  - Ctrl+J and `\`+Enter work everywhere (terminal-config);
  - Ctrl+_ is undo.
- **Codex CLI** (codex@1ef30bc codex-rs/tui/src/keymap.rs):
  - composer `submit` defaults to plain Enter (line 1711);
  - editor `insert_newline` defaults to Ctrl+J, Ctrl+M, Enter, Shift+Enter
    and Alt+Enter (lines 1721-1727, test at 4024-4037); the composer's submit
    binding takes precedence for Enter;
  - both are configurable (`editor.insert_newline`, line 2269).

### 4.3 Recommendation for bindings

- **Defaults:**
  - Enter submits;
  - **Ctrl+J always inserts a newline**: it is the only universal key;
  - Shift+Enter inserts a newline when `KeyboardEnhancementsMsg` reports key
    disambiguation;
  - Alt+Enter inserts a newline except on WT, where WT takes it;
  - a trailing `\` before Enter inserts a newline (Claude Code's convention).
- **Configuration:** bindings are configurable as lists, as in Codex's
  keymap. They are stored in the application's own configuration, never the
  CLI's.
- **On screen:** the input hint shows the newline key that works in the
  detected terminal.

## 5. Clipboard

### 5.1 OSC 52 per terminal

| Terminal | Write | Read (`OSC 52;c;?`) | Default and limit | Source |
|---|---|---|---|---|
| **Windows Terminal** | Yes since v1.2 (PR #5823, merged 2020-06-30). "Enabled by default" | **No**: "Pasting with OSC 52 was not implemented as it is a bit of a security hole". The `?` query is ignored in OutputStateMachineEngine.cpp. | `compatibility.allowOSC52` defaults to true since Preview 1.23 (secondary sources). No documented size limit; tested to 100 MiB as "relatively snappy" (PR #18949 review). A ~5 s freeze when a title change and a clipboard write land in one frame was fixed by PR #20728 (2026-09-28), backported to 1.24 and 1.25. | #2946 (lhecker, DHowett); PRs cited |
| **conhost** | Yes, PR #18949 (merged 2025-06-10). DA1 advertises it only "when writing to the clipboard is actually permitted" (PR #19034). The Windows build that ships it is unverified. | Not covered | n/a | PRs |
| **WSL2 in WT** | Through ConPTY to WT; exact path unverified. Third-party report: writes from WT's OpenConsole reach the clipboard but writes from the system conhost do not reach RDP redirection (claude-code#67576, via a mirror). | No | n/a | as cited |
| **VS Code terminal** | Since v1.93 | Unverified | n/a | can-i-use-terminal |
| **tmux** | See section 7 | 3.7 `get-clipboard`, default off | n/a | CHANGES |
| **iTerm2** | Off until "Applications in terminal may access clipboard" is enabled | Separate consent, asked each time | Limit unverified | iTerm2 prefs; Claude fullscreen |
| **Terminal.app** | **No** | No | Status on macOS 26 unverified | can-i-use-terminal; Textual api/app |
| **GNOME Terminal / VTE** | **No.** Some versions print the sequence as text. VTE #2495 open (third-party; GitLab returned 500) | No | n/a | tmux wiki "Clipboard"; claude-code#80184 |
| **Konsole** | Yes from 24.12 | **No**, by design | Old 256-character limit fixed in 2024 | bugs.kde.org #372116 |
| **kitty** | Yes; the tmux wiki reports an append bug and recommends `no-append` (whether current kitty still appends is unverified) | Asks the user | `clipboard_max_size` 512 MB | kitty conf |
| **WezTerm** | Yes | **Ignored** | Limit unverified | wezterm escape-sequences |
| **Alacritty** | Yes | Only if configured | `terminal.osc52 = OnlyCopy` | alacritty config |
| **foot** | Yes | Configurable | `osc52=enabled` (presumed default; unverified); limit unverified | foot.ini |
| **xterm** | Opt-in: `SetSelection` is in `disallowedWindowOps` by default | Yes when allowed | `limitResponse` 1024 caps replies | xterm(1); ctlseqs |

**Conclusions:**
- OSC 52 **write** is the portable copy path, and the only one that crosses
  SSH. Claude Code falls back to it over SSH (fullscreen docs). Codex: "SSH
  still uses OSC 52 exclusively" (PR #16966).
- OSC 52 **read** must not be relied on: WT, WezTerm and Konsole refuse it,
  kitty and iTerm2 ask, Alacritty refuses by default. Read queries can also
  hang, and the reply goes to whichever process reads the tty first
  (unverified, third-party).
- Paste therefore comes through bracketed paste (the terminal's own Ctrl+V)
  or the native API.

### 5.2 Native Go clipboard libraries

| | atotto/clipboard@6a3273b | golang.design/x/clipboard@d95c794 (v0.11.0) |
|---|---|---|
| Windows | Win32 `OpenClipboard`, `CF_UNICODETEXT` (clipboard_windows.go:24-100); no cgo | Win32; no cgo |
| macOS | Runs `pbcopy` / `pbpaste` | Native pasteboard through purego (v0.10.1); README: "No Cgo on the desktop" |
| Linux | Order (clipboard_unix.go:50-97): `wl-copy`/`wl-paste` when `WAYLAND_DISPLAY` is set, then `xclip`, `xsel`, termux | Wayland data-control or X11, loaded at runtime. Platform notes: the data is lost when the program exits unless a clipboard manager keeps it. GNOME before 49 goes through XWayland. |
| WSL | Last fallback: `clip.exe` to write, `powershell.exe Get-Clipboard` to read. `trimDos` strips the last 2 bytes (lines 123-125). | No WSL-specific path. It would reach the Windows clipboard only through WSLg's X11/Wayland bridge (unverified). |
| License | BSD-3-Clause | MIT |
| Used by | Bubbles textarea | n/a |

**Risks:**
- atotto's WSL path launches `powershell.exe`, which is slow (unverified;
  not measured).
- `clip.exe` handles non-ASCII text according to the console code page
  (unverified).

### 5.3 Copy paths per environment

| Environment | Copy | Paste |
|---|---|---|
| WT or VS Code terminal, app running natively on Windows | Native Win32 (either library); OSC 52 also works in WT | The terminal's Ctrl+V (WT) arrives as a bracketed paste; native read is possible |
| conhost | Native Win32; OSC 52 on builds with PR #18949 (unverified) | Paste in VT mode is bracketed (PR #15155) |
| WSL2 (Linux binary) in WT | OSC 52 to WT, or `clip.exe` / `powershell.exe Set-Clipboard` | WT's Ctrl+V as bracketed paste; a native read means a `powershell.exe` round trip |
| tmux in WSL | OSC 52 wrapped in passthrough, or with `set-clipboard on`; the tmux buffer through `tmux load-buffer -w -` (section 7) | Bracketed paste through tmux |
| Linux and macOS | Native tools first, then OSC 52 | Bracketed paste |

**What the two supported CLIs do:**
- **Claude Code (fullscreen docs):**
  - native tools first: `pbcopy`; `wl-copy`, `xclip` or `xsel`; PowerShell
    `Set-Clipboard` on Windows and WSL;
  - the tmux paste buffer too, inside tmux;
  - OSC 52 over SSH;
  - a toast names the path used.
- **Codex (PR #16966):**
  - arboard first, PowerShell under WSL, OSC 52 last;
  - OSC 52 only over SSH, wrapped in passthrough under `TMUX`;
  - a `ClipboardLease` keeps X11/Wayland clipboard ownership alive on Linux.

**Tools for wrapping OSC 52:**
- go-osc52@5c9b696 offers `.Tmux()` (needs `allow-passthrough`), `.Screen()`
  and `.Limit()`.
- x/ansi offers `TmuxPassthrough` and `ScreenPassthrough` (ansi/passthrough.go).

## 6. Mouse capture and selection

### 6.1 Bypassing the application's mouse capture

These apply once the application enables mouse modes 1000, 1002, 1003 or
1006.

| Terminal | Native-selection bypass | Block selection | Source |
|---|---|---|---|
| Windows Terminal | **Shift**: "If an application is in mouse mode, you can hold down Shift to make a selection instead of sending VT input" | **Alt**+drag; `trimBlockSelection` (true); `copyOnSelect` (false); mark mode | WT selection, "Mouse support"; interaction |
| conhost | Unverified. QuickEdit takes the mouse unless the app clears `ENABLE_QUICK_EDIT_MODE` (needs `ENABLE_EXTENDED_FLAGS`); ultraviolet's poll path *sets* QuickEdit (poll_windows.go:196) | Alt at the start of the selection | SetConsoleMode docs |
| VS Code terminal | Shift, or Option with `terminal.integrated.macOptionClickForcesSelection` | Unverified | Claude fullscreen |
| GNOME Terminal | Shift | Unverified | Textual FAQ |
| Konsole, xterm | Shift by convention (unverified) | Konsole Ctrl+Alt+drag (2009 bug; unverified) | n/a |
| kitty | Shift (`mouse_map shift+left press grabbed mouse_selection normal`) | Ctrl+Alt (unverified) | kitty conf |
| WezTerm | `bypass_mouse_reporting_modifiers` = SHIFT | Unverified | wezterm docs |
| Alacritty | Shift | Vi mode; mouse gesture unverified | alacritty MOUSE |
| foot | `selection-override-modifiers=Shift` | `Control+BTN_LEFT` | foot.ini |
| iTerm2 | **Option** | Unverified | Claude fullscreen; Textual FAQ |
| Terminal.app | **Fn** | Unverified | Claude fullscreen |
| tmux `mouse on` | The outer terminal's Shift bypasses tmux and the app together; that selection spans the whole screen and crosses panes | Copy mode | n/a |

Native selection is screen-wide.

- **Line selection** crosses side-by-side panels and picks up borders.
- **Block selection** avoids the neighbouring panel but still takes the frame
  columns inside the rectangle and pads lines with spaces. WT's
  `trimBlockSelection` trims the spaces.

Neither gives clean text from one panel of a four-zone layout, so in-app
selection is required.

### 6.2 How existing TUIs do it

- **Claude Code, fullscreen mode:**
  - **Selection:** it captures the mouse and selects in the app.
    Double-click selects a word, triple-click a line. It copies on mouse
    release (configurable in `/config`); Ctrl+Shift+C copies, and so does
    Ctrl+C when a selection exists. Shift+arrows extend the selection;
    `selectionBg` sets its colour.
  - **Native escape:** an on-screen hint shows the terminal's bypass key.
    `CLAUDE_CODE_DISABLE_MOUSE=1` turns capture off, and Ctrl+O then `[`
    dumps the transcript into native scrollback.
  - **WT/ConPTY quirk:** stale fragments may stay on screen;
    `CLAUDE_CODE_ALT_SCREEN_FULL_REPAINT=1` forces full repaints.
- **Codex CLI:** `/copy` and Ctrl+O copy the last response (PR #16966). The
  alternate screen is configurable (`tui.alternate_screen`, `--no-alt-screen`;
  repo mirror and blog only: unverified).
- **Textual:**
  - `ALLOW_SELECT` turns on app-wide selection by click and drag, and Ctrl+C
    copies (FAQ);
  - selection is limited to one widget, so frames are not selected (the
    mechanism was not read: unverified).
- **Zellij:** `copy_on_select` default true; `copy_command` or OSC 52;
  `mouse_mode` default true.
- **lazygit:**
  - `gui.mouseEvents` default true; selection then needs a modifier ("option
    key on macOS");
  - `os.copyToClipboardCmd` has an OSC 52 recipe, which needs
    `allow-passthrough` under tmux.
- **helix:** the `clipboard-provider` list includes `termcode` (OSC 52).
- **neovim:** an ordered provider list ending in `osc52`; the TUI detects
  OSC 52 support.

### 6.3 Recommendation

1. Hit-test the press with Lip Gloss `Compositor.Hit` (layer.go:289) to
   choose **one** panel. The selection then stays in that panel.
2. Map cells to positions in the panel's **content model**: unwrapped logical
   lines, without borders, padding or decorations. Clamp drags that leave
   the panel. Auto-scroll at the panel's edges.
3. Use the familiar gestures: double-click for a word, triple-click for a
   line, Shift+arrows to extend, and Shift+click to extend.
4. Copy on release (configurable), or with Ctrl+Shift+C. Copy with Ctrl+C
   only while a selection exists; otherwise Ctrl+C keeps its interrupt
   meaning.
   - WT intercepts Ctrl+C and Ctrl+Shift+C only when *WT* has a selection,
     and sends them on otherwise (actions page).
   - Without the Kitty protocol, Ctrl+Shift+C arrives as Ctrl+C (0x03).
5. Copy through the layered clipboard (section 11) and show which path was
   used.
6. Keep native escapes:
   - a key that toggles mouse capture off;
   - an on-screen hint with the bypass key (Shift in WT);
   - "open panel in `$EDITOR`" (section 8).

## 7. tmux (WSL only on this machine)

Defaults from `options-table.c` (master, 2026-10-15) and the wiki:

| Option | Values | Default | Notes |
|---|---|---|---|
| `set-clipboard` | off / external / on | **external** | external: tmux sets the outer clipboard, but programs inside cannot; on: they can. Needs `Ms`, auto-added for `xterm*`; otherwise `terminal-features ',X:clipboard'` (3.2+). 3.6 detects OSC 52 from DA. |
| `get-clipboard` (3.7) | n/a | off | Forwards a pane's OSC 52 read to the outer terminal (CHANGES 3.7). Exact values unverified. |
| `copy-command` | string | "" | Used by `copy-pipe` without arguments (3.2+). Using both set-clipboard and copy-pipe can conflict (wiki). |
| `allow-passthrough` | off / on / all | **off** | 3.3+; `all` (3.4) also covers invisible panes. Needed for wrapped OSC 52. |
| `mouse` | on / off | off up to 3.7; **on from 3.8** | With app mouse reporting, drags go to the app; otherwise into copy mode (`MouseDrag1Pane`, key-bindings.c). Copy mode: `MouseDragEnd1Pane` runs `copy-pipe-and-cancel`. |
| `extended-keys` | off / on / always | off (options-table.c) | The wiki says 3.5 "defaults to supporting extended-keys in a way similar to xterm", while Claude docs say Shift+Enter submits in tmux by default. The effective default is unverified. |
| `extended-keys-format` | xterm / csi-u | xterm | 3.5+; pi.dev recommends `csi-u` on 3.5+. |
| `terminal-features` | n/a | `xterm*:clipboard:ccolour:cstyle:focus:title,screen*:title,rxvt*:ignorefkeys` | Add `xterm*:extkeys` for modified keys. |
| `escape-time` | ms | 10 | n/a |
| `prefix` | key | C-b | Takes Ctrl+B from the application. |

**Kitty keyboard protocol:** not in any released tmux (no CHANGES entry).
PR #5600 (`extended-keys-format kitty`) was closed into PR #5615, still under
review on 2026-10-15. Bubble Tea's modifyOtherKeys request (section 2.2) is
what tmux can serve. That it qualifies as "the app requested extended keys"
is unverified.

**Configuration the owner would add** (Claude Code's recommendation,
terminal-config "Configure tmux"; proposed, never applied by the
application):

```text
set -g allow-passthrough on
set -s extended-keys on
set -as terminal-features 'xterm*:extkeys'
```

**For the application:**
- wrap OSC 52 for passthrough when `TMUX` is set;
- also write the tmux buffer with `tmux load-buffer -w -`, the "-w" variant
  (3.2) that sends to the outer clipboard too;
- tmux is never a dependency;
- tmux has no native Windows build (CHANGES lists only Cygwin and MSYS2
  fixes), so it is WSL only here.

## 8. `$EDITOR` and `$VISUAL`

**Bubble Tea:**
- `tea.ExecProcess(c *exec.Cmd, fn)` (exec.go:50) releases the terminal, runs
  the command and restores the terminal (exec.go:102-129).
- tea.go:858 notes "NB: this blocks", and `msgs` is unbuffered
  (tea.go:620). While the editor runs, **CLI events stop being processed**,
  so streaming from the CLI must be buffered outside the tea loop (in the
  provider goroutine) to avoid back-pressure on the CLI's pipe.
- Its example `exec.Command(cmp.Or(os.Getenv("EDITOR"), "vim"))`
  (examples/exec/main.go:15-16) cannot run `code --wait`, because the
  arguments must be split first.

**How the two supported CLIs and Git do it:**
- **Git:** `GIT_EDITOR`, then `core.editor`, `VISUAL`, `EDITOR`, then a
  compile-time default. The value is "interpreted by the shell" and may carry
  arguments (git-scm.com/docs/git-var).
- **Claude Code:**
  - Ctrl+G (alias Ctrl+X Ctrl+E) "honor[s] `VISUAL` and then `EDITOR`", and
    only otherwise detects VS Code, vi or nano. For GUI editors, "use a
    command that blocks until the file is closed (e.g. `code --wait`)"
    (bcherny, claude-code#18990, 2026-08-17).
  - Its Windows fallback is undocumented. In #24077 (Notepad on Windows),
    Claude Code resumed after the user saved and closed Notepad.
- **Codex CLI:**
  - Ctrl+G opens the prompt in `$VISUAL` or `$EDITOR` (PR #7606, merged
    2025-12-22, rust-v0.78.0);
  - the TUI stays in raw mode;
  - it uses a temp *path*, not a held handle, because Windows will not let
    another process save a file that is held open;
  - a commit "use winsplit for windows parsing" splits arguments with Windows
    rules;
  - cancelling during `code --wait` was deferred.
- **VS Code:** `-w, --wait`, "Wait for the files to be closed before
  returning" (code.visualstudio.com command-line docs).

**Windows defaults and pitfalls:**
- `EDITOR` and `VISUAL` are usually unset on Windows (unverified for this
  machine; not read).
- Whether the Store Notepad blocks when launched from a console is
  **unverified**. It needs a manual check.
- Prefer `code --wait`, `notepad++ -multiInst -nosession`, or a terminal
  editor (`nvim`, `edit`).
- The Git for Windows installer's default editor is unverified.

**Design:**
1. Resolve `VISUAL`, then `EDITOR`.
2. Split the value with shell rules on Unix and `CommandLineToArgvW` rules on
   Windows.
3. Write the draft to a temp file, then close it.
4. `ExecProcess`, wait, read the file back byte for byte, and delete it.
5. If neither variable is set, offer a detected editor and show which one was
   used.
6. Warn that a GUI editor needs its blocking flag.
7. Show the failure if the editor exits non-zero or is not found; never
   discard the draft.

## 9. Shortcuts already taken

| Layer | Keys it takes before the application sees them | Source |
|---|---|---|
| **Windows Terminal** (defaults) | See the list below the table | WT actions page |
| **Windows 11** | Win+… keys; Alt+Tab; Alt+F4; Ctrl+Alt+Del; Win+V clipboard history; Ctrl+Shift or Alt+Shift input-language switching when several layouts are installed (unverified: OS defaults, not fetched) | n/a |
| **conhost** | Ctrl+M or the menu for mark mode; QuickEdit mouse selection; Alt+Enter fullscreen (unverified) | SetConsoleMode; 2014 console blog |
| **VS Code terminal** | Workbench shortcuts (Ctrl+P, Ctrl+Shift+P, Ctrl+\`, and others) are taken unless listed in `terminal.integrated.commandsToSkipShell`, or unless `sendKeybindingsToShell` is set (unverified: VS Code terminal docs not fetched) | n/a |
| **tmux** | Prefix Ctrl+B and the bindings after it; with `mouse on`, the wheel and drags outside mouse-mode apps | options-table.c |
| **macOS** | Cmd+… keys never reach the app (Cmd+C/V copy and paste through the terminal); Option sends characters unless "Option as Meta" is on, so Alt bindings are unreliable (Codex PR #16966) | as cited |
| **GNOME Terminal, Konsole** | Ctrl+Shift+C/V/T/N/W, F11 (unverified: defaults not fetched) | n/a |
| **CLI conventions** | Ctrl+C interrupts; Ctrl+D is EOF; Ctrl+Z suspends (not supported on Windows in Bubble Tea, tty_windows.go:62) | n/a |

Windows Terminal default keys (actions page):

- **Copy:** Ctrl+C, Ctrl+Shift+C, Ctrl+Insert and Enter. They copy only when
  WT has a selection, and are sent to the app otherwise.
- **Paste:** Ctrl+V, Ctrl+Shift+V and Shift+Insert.
- **Window and tab:**
  - Alt+Enter and F11 toggle fullscreen.
  - Ctrl+Shift+T opens a new tab; Ctrl+Shift+1..9 open a tab with a given
    profile; Ctrl+Shift+D duplicates the tab.
  - Ctrl+Tab and Ctrl+Shift+Tab switch to the next and previous tab;
    Ctrl+Alt+1..8 switch to a given tab.
  - Ctrl+Shift+N opens a new window; Alt+F4 closes it.
  - Ctrl+Shift+Space opens the new-tab dropdown.
  - Ctrl+, , Ctrl+Shift+, and Ctrl+Alt+, open the settings.
  - Alt+Space opens the system menu.
- **Panes:**
  - Alt+Shift+D, Alt+Shift+- and Alt+Shift+Plus split the pane.
  - Ctrl+Shift+W closes the pane.
  - Alt+arrows move focus; Ctrl+Alt+Left goes to the previous pane.
  - Alt+Shift+arrows resize the pane.
- **Search, scroll and font:**
  - Ctrl+Shift+F finds.
  - Ctrl+Shift+Up, Down, PgUp, PgDn, Home and End scroll.
  - Ctrl+Plus, Ctrl+Minus and Ctrl+0 change the font size.
- **Selection:** Ctrl+Shift+A selects all; Ctrl+Shift+M enters mark mode.
- **Not extracted:** the command-palette key and the quake-mode key.

**Consequences for the application's defaults:**
- Avoid Alt+Enter, Alt+arrows, Alt+Shift+…, Ctrl+Shift+letters and Ctrl+Tab
  on WT.
- Ctrl+V reaches the application as a bracketed paste, not as a key.
- Ctrl+C is free when WT has no selection.
- Ctrl+J, Ctrl+G, Ctrl+O, Ctrl+Y and Ctrl+Z (for undo, since suspend does not
  apply on Windows) are free on WT.

## 10. Summary matrix

| Requirement | Feasible where | How | Limits | Verify by hand in the MVP |
|---|---|---|---|---|
| Paste never submits | Bubble Tea v2 on WT, conhost (VT mode), VS Code, Unix terminals, tmux | `PasteMsg` only; never submit on it | Unbracketed terminals need a burst heuristic (Codex) | Paste 3 lines in WT, conhost, VS Code and tmux-in-WSL: nothing is sent |
| Byte-for-byte content | Everywhere, within terminal limits | Own editor with no sanitizer | WT turns LF/CRLF into CR and strips controls; Bubble Tea drops invalid UTF-8 and U+FFFD; `trimPaste` effect unverified | Paste tabs, CRLF, trailing spaces, emoji and U+FFFD; diff what the CLI receives |
| Thousands of lines | All | Piece table or rope; render only visible rows | WT's 5 KiB and multiline warning dialogs | Paste 10 000 and 50 000 lines; check input latency and memory |
| Selection, cursor, undo | All | Own editor (Shift+arrows, word moves, undo and redo) | n/a | Keyboard walk-through |
| Newline key | Ctrl+J everywhere; Shift+Enter with Kitty or modifyOtherKeys (WT 1.25 open question); Alt+Enter except WT | Configurable lists | WT and ConPTY Kitty passthrough unsettled; VTE has no Shift+Enter | Shift+Enter in WT 1.25 (native and WSL), with and without `tea.Raw(CSI ?9001h)`; VS Code; tmux with extended-keys |
| Copy to system clipboard | WT, VS Code, conhost (build-dependent), WSL, tmux, Linux, macOS | Native first, then OSC 52 (passthrough under tmux), then the tmux buffer | OSC 52: no Terminal.app or VTE, iTerm2 opt-in; Linux native data lost on exit | Copy in each environment and paste into Notepad |
| Paste from system clipboard | All | The terminal's paste (bracketed); native read optional | No OSC 52 read on WT | Ctrl+V in WT and VS Code; Shift+Insert |
| Mouse selection in any panel | Terminals with SGR mouse | Hit test, then the content model, then copy | Native selection crosses panels | Drag across panel edges; double and triple click; borders are never copied |
| Native selection escape | All | Shift (WT), Option (iTerm2), Fn (Terminal.app); toggle off capture | Screen-wide, includes borders | Shift+drag in WT |
| `$EDITOR` | All | `ExecProcess`, split argv, temp file | Event loop blocks; GUI editors need a wait flag; Notepad unverified | `code --wait`, `notepad`, `nvim` in WSL |

## 11. Recommendation

1. **Runtime:** Bubble Tea v2 with Lip Gloss v2 (MIT). It is the only Go
   option with all of these:
   - one-message bracketed paste on Windows (VT input);
   - Kitty and modifyOtherKeys requests;
   - OSC 52 commands;
   - mouse SGR;
   - layer hit testing.

   Bubble Tea v1 fails on Windows paste.
2. **Editor component:** our own, in the application. Not the Bubbles
   textarea as-is.
   - piece table or rope storage;
   - undo and redo;
   - selection with keyboard and mouse;
   - render only the visible rows, with lazy wrap;
   - verbatim insertion of `PasteMsg`, with WT's CR stored as LF;
   - no tab expansion;
   - no line cap, or a visible one.

   Ideas and code may come from the Bubbles textarea and its selection.go
   (MIT) and from tview TextArea (MIT), keeping their notices and recording
   source, commit and license.

   Contributing a fix to the Bubbles textarea upstream is the alternative.
   It is not recommended: it needs four upstream changes (sanitizer,
   maxLines, render, undo).
3. **Keys:** Enter submits; Ctrl+J and Shift+Enter (when distinguishable)
   insert a newline. Keep `\`+Enter. Bindings are configurable; avoid WT's
   taken keys.
4. **Clipboard:**
   - **Copy** goes native first:
     - Win32 when running on Windows: golang.design/x/clipboard (MIT, no
       cgo) or atotto (BSD-3-Clause; needs its notice);
     - `powershell.exe Set-Clipboard` or `clip.exe` under WSL;
     - wl-copy or xclip on Linux;
     - pbcopy on macOS.
   - **Also** OSC 52, wrapped for tmux when `TMUX` is set (go-osc52 or
     x/ansi passthrough), plus `tmux load-buffer -w`. Use OSC 52 alone over
     SSH. Show the path used.
   - **Paste** comes only through bracketed paste; never OSC 52 read.
5. **Mouse:** capture on; in-app selection per panel (section 6.3); a toggle
   to release capture; Shift-drag hint.
6. **`$EDITOR`:** use the section 8 design. Buffer CLI events while the
   editor runs.

**Risks:**
- Shift+Enter on Windows through ConPTY is unsettled. Ctrl+J is the
  guaranteed fallback, and a win32-input-mode probe is a plan-4 check.
- WT rewrites pasted line endings and may trim trailing whitespace
  (`trimPaste`). Byte for byte holds only for what reaches the application.
  Document this, and if confirmed, tell the owner about `trimPaste`.
- Bubble Tea drops U+FFFD and invalid UTF-8 inside pastes.
- Writing our own editor is the largest single input task in plan 4. Its
  performance must be measured: 50 000 lines, latency per keystroke.
- `ExecProcess` blocks the event loop.
- A Linux native clipboard loses its data on exit without a clipboard manager.
- The behaviour of conhost (QuickEdit, OSC 52 and bracketed paste per Windows
  build) is unverified.

**Open questions (unverified):**
- **Windows:** WT 1.25 Kitty protocol through ConPTY; WT honouring a 9001
  request from the app; `trimPaste` semantics; which Windows 11 build ships
  conhost bracketed paste and OSC 52; the conhost mouse bypass; the
  Notepad-blocks question.
- **Bubble Tea:** which console reader path Bubble Tea v2 takes.
- **VS Code terminal:** Kitty protocol, Ctrl+Enter and its shortcuts.
- **tmux:** the effective `extended-keys` default and how it re-wraps outer
  pastes.
- **Other terminals:** OSC 52 size limits beyond kitty and WT; the
  block-selection gestures of several terminals.
- **Codex:** the `alternate_screen` option.
