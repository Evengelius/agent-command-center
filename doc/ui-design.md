# UI design

Plan 2, 2026-10-16. What the owner sees and does: the startup screen, the
transition, the dashboard's four zones at three terminal sizes, the keys, the
input editor, the clipboard and selection, and how origins, states and
failures look. The parts behind the screen are in
[architecture.md](architecture.md), the events and state machines in
[events-and-state.md](events-and-state.md), failures, fallbacks and tests in
[reliability.md](reliability.md). Citation keys (`INT`, `TEL`, `TI`, `CC`,
`CX`, `RD`, `P0`-`P7`, `AM2`-`AM5`, `ASM-n`) are those of
[architecture.md, Conventions](architecture.md#conventions).

The interface renders state and nothing else: events update the state, the
state is drawn, and no model is ever called to draw or update the screen
(AGENTS.md; AM2 E1, E2).

## Mock-ups

The screens below are text mock-ups, drawn to exact sizes and checked by a
script for width and height. **Every value in them is illustrative**: the
model names, percentages, token counts, times and file names show where a
value goes and how it is labelled, not what any session reported. The
standard and narrow mock-ups show a Claude Code session; the wide one a
Codex session, so that both providers' differences appear (model N/A per
child, cost N/A, an observed plan). Colours are described in
[Colours and symbols](#colours-and-symbols); the mock-ups are monochrome.

## Startup screen

The first screen (AM4 S1-S15). Two mandatory questions, validated before the
dashboard opens; nothing else.

```text
 >_ AGENT COMMAND CENTER                                            New session 
────────────────────────────────────────────────────────────────────────────────
                                                                                
    Configure your workspace                                                    
    Select an agent CLI and the project to work on.                             
                                                                                
    1. Agent CLI *                                                              
       (•) Claude Code     2.1.300   C:\Users\…\.local\bin\claude.exe           
       ( ) OpenAI Codex    0.145.0   C:\Users\…\codex.exe                       
    ────────────────────────────────────────────────────────────────────────    
    2. Project directory *                                                      
       ╭────────────────────────────────────────────────────────────────╮       
       │ C:\javadev\projets\agent-command-center█                       │       
       ╰────────────────────────────────────────────────────────────────╯       
       → C:\javadev\projets\agent-command-center   git: main                    
       Type a full path; ~ and /mnt/c/… are expanded. Default: current dir.     
    ────────────────────────────────────────────────────────────────────────    
                   ╭──────────────────────────────────────────╮                 
                   │            Start session  →              │                 
                   ╰──────────────────────────────────────────╯                 
                                                                                
                                                                                
                                                                                
 ↑↓ agent · Tab next field · Enter start · Ctrl+Q quit                          
```

### Questions

1. **Agent CLI**: radio buttons, Claude Code then OpenAI Codex (AM4 S1;
   AM5). Each row shows what detection found: the version and the
   executable's path, with the user folder elided (`C:\Users\…`). While
   detection runs (startup machine `Detecting`), each row shows
   "detecting…" with a spinner and Start is disabled. A CLI not found is
   shown "not installed" with the reason (`codex not found on PATH`,
   "version unreadable: exit 1") and is **not selectable** (AM4 S2, S3;
   events-and-state.md, Startup machine). The selection defaults to
   `--provider` if given and installed, else the first installed CLI
   (ASM-8).
2. **Project directory**: a one-line text field, prefilled with the current
   directory or `--project` (AM4 S7, S12; ASM-8). Under it, a live line
   shows the **resolved path** the agent will receive, recomputed on every
   keystroke without any I/O (steps 1 to 3 of INT paths, "Validation, in
   order": trim, expand `~`, translate WSL forms, classify, make absolute);
   `Stat` and the read check (steps 4 and 5) run only on Start. When the
   directory is a git work tree, the line adds the branch read from
   `.git/HEAD` after validation (display only; architecture.md, Session
   context). One help line: "Type a full path; ~ and /mnt/c/… are expanded.
   Default: current dir."

A paste into the path field arrives as one bracketed paste (TI 3.2):
surrounding quotes (Explorer's "Copy as path") and the trailing newline are
removed; a paste never starts the session.

### Validation and messages

On Start (Enter anywhere on the screen), both answers are validated; the
session starts only if both pass (AM4 S9, S10, S14). A failure keeps the
screen, puts the message under its field in red with ✗, and moves the focus
there. Messages, from INT paths, each with the resolved path or the
corrected example:

| Case | Message under the field |
| - | - |
| No CLI installed | ✗ No supported agent CLI found. Install Claude Code or Codex CLI, then restart. (Start stays disabled) |
| Selected CLI fails its version check | ✗ Claude Code: `claude --version` failed (exit 1): `<first stderr line>` |
| Path does not exist | ✗ Does not exist: `C:\Users\…\projets\agent-comand-center` |
| Path is a file | ✗ Is a file, not a directory: `<resolved path>` |
| Not readable | ✗ Not readable (permission denied): `<resolved path>` |
| Drive-relative `C:dev\app` | ✗ Drive-relative path. Type a full path, for example `C:\dev\app` |
| Root-relative `\dev\app` | ✗ Root-relative path. Type a full path with its drive, for example `C:\dev\app` |
| `~bob/app` | ✗ Another user's home is not expanded. Type the full path |
| Other device path (`\\.\PhysicalDrive0`) | ✗ Not a directory path |
| POSIX path on Windows (`/home/u/app`) | ✗ A Linux path. On Windows use `C:\…`, or `/mnt/c/…` for a Windows drive |
| Windows path in Linux or macOS | ✗ A Windows path |
| Translated WSL path fails `Stat` | ✗ Does not exist: `<translated path>` (the WSL mount root may not be `/mnt/`) |

The screen with two failures, Codex missing and a mistyped path:

```text
 >_ AGENT COMMAND CENTER                                            New session 
────────────────────────────────────────────────────────────────────────────────
                                                                                
    Configure your workspace                                                    
    Select an agent CLI and the project to work on.                             
                                                                                
    1. Agent CLI *                                                              
       (•) Claude Code     2.1.300   C:\Users\…\.local\bin\claude.exe           
       ( ) OpenAI Codex    not installed: codex not found on PATH               
    ────────────────────────────────────────────────────────────────────────    
    2. Project directory *                                                      
       ╭────────────────────────────────────────────────────────────────╮       
       │ ~/projets/agent-comand-center█                                 │       
       ╰────────────────────────────────────────────────────────────────╯       
       ✗ Does not exist: C:\Users\…\projets\agent-comand-center                 
       Type a full path; ~ and /mnt/c/… are expanded. Default: current dir.     
    ────────────────────────────────────────────────────────────────────────    
                   ╭──────────────────────────────────────────╮                 
                   │            Start session  →              │                 
                   ╰──────────────────────────────────────────╯                 
                                                                                
                                                                                
                                                                                
 ↑↓ agent · Tab next field · Enter start · Ctrl+Q quit                          
```

Warnings (shown in amber with !, Start still allowed): a UNC path ("a
network share can be slow or disappear"); a `\\wsl$` or `\\wsl.localhost`
path ("files in Linux, used by a Windows agent CLI: slow, and unverified").

### Keys

| Key | Action |
| - | - |
| ↑ / ↓ | choose the agent CLI (on question 1) |
| Tab / Shift+Tab | next / previous field (question 1, question 2, Start) |
| Enter | Start: validate both answers, then start |
| Editing keys | as in the [Input editor](#input-editor), on one line |
| Ctrl+Q | quit |

### Follows and departs

Against [RD, Image 1](reference-design.md#image-1-startup-configuration):

| Image 1 | Here | Why |
| - | - | - |
| Title bar "AGENT COMMAND CENTER", "New session" on the right | Same | AM4 S15 |
| Violet heading "Configure your workspace" and one line of help | Same; the help says "agent CLI" for "AI agent" | AM4 S15 |
| Two numbered questions, red asterisks, thin separators | Same | AM4 S15 |
| Three agents, including Google Gemini CLI | Two: Claude Code and OpenAI Codex | AM5 |
| Each agent a bare name | Each row adds the detected version and path, or "not installed" | AM4 S2, S3 |
| "Browse directories, recent projects, or enter a path" | A help line for manual entry, and the resolved path shown live | AM4 S13, X1-X3 (RD, "Where the images and amendment 4's text differ") |
| "Optional session settings" | Absent | AM4: exactly two questions; X10 |
| One wide light button "Preview configuration →" | One wide light button "Start session →", which validates then starts | AM4 S14 |

## Transition

On Start with both answers valid, the screen is swapped for the dashboard
**at once**, in its starting state: the project bar filled from the session
context, zone B showing "Starting Claude Code 2.1.300 in
`C:\…\agent-command-center`…" with a spinner, zones A, C and D empty with
"waiting for the session" and every value "N/A (session starting)". The
input editor is live: the owner can type, and send is disabled until the
session machine reaches `Ready` ("send available when the session is
ready"). No slide or fade: one application, one frame swap (AM4 D1, F11;
AM2 U1).

On `Ready`, zone B shows "Ready to work on this project." and the input
placeholder "Ask Claude Code anything…" (RD Image 2), and send is enabled.

If the start fails (startup machine `StartFailed`), zone B shows an error
card instead: the cause (spawn error, handshake timeout after 60 s, `cwd`
mismatch, process exit), the exit code and the last lines of stderr, and
"Esc back to startup (answers kept) · Ctrl+Q quit" (events-and-state.md,
Startup machine; architecture.md, How it starts, step 6). The draft, if
any, is kept across the return.

## Dashboard

The working screen, persistent for the session. Zones A to D of AM2 and AM4
D3, under a title bar and a project bar, with a key-hint line at the bottom.
The conversation (B) is the primary surface and holds the default focus (AM4
D4).

### Follows and departs

Against [RD, Image 2](reference-design.md#image-2-dashboard-empty-state) and
the [layout mock-up](reference-design.md#layout-mock-up):

| Reference | Here | Why |
| - | - | - |
| Title "AGENT COMMAND CENTER" and "● LIVE" badge | Same; "◆ MOCK — not a live session" replaces "● LIVE" in mock mode | AM4; B3.6 |
| Project bar: folder name, branch, agent CLI on the right; path below | One line at 100+ columns: name, branch, path, then provider and version on the right; the path drops at 80 columns | AM4 W1 |
| A left, full height: orchestrator, model, stage tree with agents indented, a status symbol per line | Same; the tree is labelled with its origin | AM2 A1-A4; [Origins on screen](#origins-on-screen) |
| B upper right: "Streaming" badge, YOU and CLAUDE labelled, an in-progress line, the input at the bottom | Same, with a multiline editor, approval cards inline, and tool lines | AM2 B1-B6 |
| C lower right: one line per agent (model, task, status), an active count, "select agent to inspect" | Same, adding elapsed time and tokens; at 160+ columns a live preview of the selected agent beside the list | AM2 C1-C3, C11 |
| D bottom, full width: two quota bars with percentages, agents, context, tokens | Same, plus resets, model, cost, time, errors and retries; every missing value "N/A" with its reason | AM2 D1-D9 |
| Mock-up figures (42%, "Agents 4/16") | Real values or "N/A" | AGENTS.md; RD "Layout mock-up" |
| Image 2's empty state: "Waiting for instructions", "Ready to work on this project." | Same texts in the ready state | AM4 D2 |
| A one-line status bar (first reference) | A key-hint line; the essentials live in D | RD "What the owner wants from it" |

## Project bar

One line under the title (AM4 W1; architecture.md, Session context):

- **left**: ▣ the project name (last element of the path), then "branch
  `<name>`" when the project is a git work tree, read once from
  `.git/HEAD` (a detached head shows the short commit; no git command is
  run);
- **middle**: the resolved absolute path, shortened from the left with "…"
  when it does not fit, and dropped below 100 columns;
- **right**: the provider's name and version; for Codex also the model,
  sandbox and approval policy reported by `thread/start` (P6b), because
  they decide what the agent may do.

## Layouts at three sizes

Three layouts, chosen by the terminal's width: narrow below 100 columns,
standard from 100 to 159, wide from 160; widths and splits below are ASM-12. The layout is recomputed on
every resize message; the state does not change. Panel heights share what
the fixed lines leave: B takes about 60% of the right column, C the rest; at
narrow width C has 3 rows and D 1.

### Narrow, 80x24

```text
 >_ AGENT COMMAND CENTER                                                 ● LIVE 
 ▣ agent-command-center  branch main                                Claude Code 
 A ✓ Turn 1 › ◉ Turn 2: scan-routes ◉ write-tests !   (inferred)   F6: tree     
╭─ B · CONVERSATION ───────────────────────────────────────────── ◉ Streaming ─╮
│ YOU  Refactor the auth module and add tests for the token refresh.           │
│ CLAUDE  I'll read the current module first, then split the refresh           │
│   logic out of `middleware.go` and cover it with table tests.                │
│   ▸ Read src/auth/middleware.go                                     observed │
│ ! APPROVAL write-tests: Edit refresh_test.go +12  a allow  d deny            │
│ ──────────────────────────────────────────────────────────────────────────── │
│ > Also check the logout path█                                                │
│                                                                              │
│                                                                              │
│   turn running · Ctrl+C interrupt                               1 line       │
╰──────────────────────────────────────────────────────────────────────────────╯
╭─ C · AGENTS ────────────────────────────────────────────────────── 2 ACTIVE ─╮
│ ◉ scan-routes  sonnet-5-5  Bash go test ./auth           41s   12.1k         │
│ ! write-tests  sonnet-5-5  waiting: approval             18s   6.4k          │
│ ✓ explore-1    haiku-5-5   completed                     9s    3.0k          │
╰──────────────────────────────────────────────────────────────────────────────╯
╭─ D ──────────────────────────────────────────────────────────────────────────╮
│ 5h ▰▰▰▱▱▱▱▱▱▱ 23%† 7d ▰▰▰▰▰▱▱▱▱▱ 45%† ctx 12%  tok 41.2k  $0.42 est.  err 0  │
╰──────────────────────────────────────────────────────────────────────────────╯
 Enter send · Ctrl+J newline · F6 zone · F2 approve · F1 keys                   
```

- A collapses to one line, the current path through the stage tree with a
  status symbol per step. Focusing A (F6) opens the full tree as an overlay
  over B and C; Esc closes it.
- B keeps every function at reduced size: messages, tool lines, a one-line
  approval card, the editor (3 visible rows; it grows to half the screen as
  lines are added, then scrolls).
- C shows three agents, most recently active first; more are reached by
  scrolling or zoom.
- D is one line: both quotas, context, tokens, cost, errors; the rest is in
  the key overlay's "telemetry" page and in the zoomed D panel.

### Standard, 120x40

```text
 >_ AGENT COMMAND CENTER                                                                                         ● LIVE 
 ▣ agent-command-center   branch main   C:\javadev\projets\agent-command-center                     Claude Code 2.1.300 
╭─ A · ORCHESTRATOR ─────────────╮╭─ B · CONVERSATION ─────────────────────────────────────────────────── ◉ Streaming ─╮
│ claude-opus-5-5 · orchestrator ││ YOU  14:02                                                                         │
│ turn 2 · running 1m 12s        ││   Refactor the auth module and add tests for the token refresh.                    │
│                                ││ CLAUDE  claude-opus-5-5                                                            │
│ STAGES  inferred from activity ││   I'll read the current module first, then split the refresh logic                 │
│ ✓ Turn 1  Explain auth flow    ││   out of `middleware.go` and cover it with table tests.                            │
│ ◉ Turn 2  Refactor auth        ││   ▸ Read src/auth/middleware.go · 212 lines                               observed │
│   ├─ ✓ Read  src/auth/*.ts     ││   ▸ Agent scan-routes started                                             observed │
│   ├─ ◉ agent  scan-routes      ││   ┌ go ───────────────────────────────────                                         │
│   │  └─ ◉ Bash go test ./auth  ││   │ func refresh(ctx context.Context, t Token) (Token, error) {▌                   │
│   ├─ ◉ agent  write-tests      ││ ╭ APPROVAL · write-tests · Edit src/auth/refresh_test.go ────────────────────────╮ │
│   │  └─ ! approval Edit        ││ │ +12 −0   F2 focus, then a allow · d deny · c cancel turn · Enter diff          │ │
│   └─ ○ (more as events arrive) ││ ╰────────────────────────────────────────────────────────────────────────────────╯ │
│                                ││ ────────────────────────────────────────────────────────────────────────────────── │
│                                ││ > Also check the logout path█                                                      │
│                                ││                                                                                    │
│                                ││   turn running: the draft waits · Ctrl+C interrupt          1 line · Ctrl+G editor │
│                                ││                                                                                    │
│                                ││                                                                                    │
│                                ││                                                                                    │
│                                │╰────────────────────────────────────────────────────────────────────────────────────╯
│                                │╭─ C · LIVE AGENTS ────────────────────────────────────────────── 2 ACTIVE · 1 DONE ─╮
│                                ││ ◉ scan-routes   sonnet-5-5    Bash go test ./auth              41s     12.1k tok   │
│                                ││ ! write-tests   sonnet-5-5    waiting: approval Edit           18s     6.4k tok    │
│                                ││ ✓ explore-1     haiku-5-5     completed · 3 files read         9s      3.0k tok    │
│                                ││                                                                                    │
│                                ││                                                                                    │
│                                ││                                                                                    │
│                                ││                                                                                    │
│                                ││                                                                                    │
│                                ││                                                                                    │
│ ↑↓ move ←→ fold Enter inspect  ││ Enter inspect · z zoom · F6 next zone                                              │
╰────────────────────────────────╯╰────────────────────────────────────────────────────────────────────────────────────╯
╭─ D · QUOTA & TELEMETRY ────────────────────────────────────────────────────── † undocumented field · updated 4s ago ─╮
│ 5h ▰▰▰▰▰▱▱▱▱▱▱▱▱▱▱▱▱▱▱▱ 23%†  resets 16:40   7d ▰▰▰▰▰▰▰▰▰▱▱▱▱▱▱▱▱▱▱▱ 45%†  resets Mon 09:00   model claude-opus-5-5  │
│ context 12% of 1M   tokens 41.2k   cost $0.42 API-equiv. estimate   agents 2 / limit N/A   time 3m 05s   errors 0    │
╰──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╯
 Enter send · Ctrl+J newline · Ctrl+C interrupt · Ctrl+G editor · F6 zone · F2 approvals · F1 keys · Ctrl+Q quit        
```

- A is a 34-column column, full height: orchestrator, model, turn and its
  elapsed time, then the stage tree.
- B above C on the right, as in the layout mock-up. B's input area sits at
  B's bottom edge, separated by a rule.
- D is two lines.

### Wide, 200x60

```text
 >_ AGENT COMMAND CENTER     ■ gpt-5.6-sol                                                                                                                                                       ● LIVE 
 ▣ agent-command-center   branch main   C:\javadev\projets\agent-command-center                                           OpenAI Codex 0.145.0 · gpt-5.6-sol · sandbox read-only · approvals on-request 
╭─ A · ORCHESTRATOR ───────────────────────╮╭─ B · CONVERSATION ───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────── ◉ Streaming ─╮
│ gpt-5.6-sol · orchestrator thread        ││ YOU  14:02                                                                                                                                               │
│ turn 2 · running 2m 40s                  ││   Refactor the auth module and add tests for the token refresh.                                                                                          │
│                                          ││                                                                                                                                                          │
│ PLAN  observed (turn/plan/updated)       ││ CODEX  gpt-5.6-sol                                                                                                                                       │
│ ✓ 1 Inspect the auth module              ││   The refresh logic lives in `middleware.go` and is called from two routes. Plan:                                                                        │
│ ◉ 2 Split refresh logic                  ││   1. move `refresh` into its own file;                                                                                                                   │
│   ├─ ✓ commandExecution  rg refresh      ││   2. add table tests for expired, revoked and valid tokens;                                                                                              │
│   ├─ ◉ fileChange  src/auth/refresh.go   ││   3. run `go test ./...`.                                                                                                                                │
│   │    └─ ! approval pending             ││   ▸ commandExecution  rg -n refresh src/auth          exit 0 · 0.4s                                                                             observed │
│   ├─ ◉ child  /root/scan                 ││   ▸ child /root/scan started                                                                                                                    observed │
│   │    └─ ◉ commandExecution  go vet     ││   ▸ child /root/docs started · completed 22s                                                                                                    observed │
│   └─ ✓ child  /root/docs                 ││   Moving the function now.▌                                                                                                                              │
│ ○ 3 Add table tests                      ││                                                                                                                                                          │
│ ○ 4 Run the test suite                   ││ ╭ APPROVAL · orchestrator · fileChange src/auth/refresh.go (new file, +48) ────────────────────────────────────────────────────────────────────────────╮ │
│                                          ││ │ @@ -0,0 +1,48 @@                                                                                                                                     │ │
│ TURN 1  ✓ completed · 31s                ││ │ +package auth                                                                                                                                        │ │
│   ✓ Explain the auth flow                ││ │ +func refresh(ctx context.Context, t Token) (Token, error) {                                                                                         │ │
│                                          ││ │ …  45 more lines · Enter full diff                                                                                                                   │ │
│                                          ││ │ a allow once   s allow for session   d decline   c cancel turn   Esc later                                                                           │ │
│                                          ││ ╰──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╯ │
│                                          ││ ──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────── │
│                                          ││ > Also check the logout path, and keep the old                                                                                                           │
│                                          ││   function name as a deprecated alias█                                                                                                                   │
│                                          ││                                                                                                                                                          │
│                                          ││                                                                                                                                                          │
│                                          ││                                                                                                                                                          │
│                                          ││   turn running: the draft waits until the turn ends · Ctrl+C interrupt                                   2 lines · Ctrl+G editor                         │
│                                          ││                                                                                                                                                          │
│                                          ││                                                                                                                                                          │
│                                          │╰──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╯
│                                          │╭─ C · LIVE AGENTS ────────────────────────────────── 1 ACTIVE · 1 DONE ─╮╭─ /root/scan · LIVE ───────────────────────────────── model N/A (not reported) ─╮
│                                          ││ ◉ /root/scan   model N/A   commandExecution go vet   55s    30.4k tok  ││ 14:03:10  item/started  commandExecution  go vet ./src/auth            observed│
│                                          ││ ✓ /root/docs   model N/A   completed                 22s    77.3k tok  ││ 14:03:11  agent  Checking the callers of refresh before the move…            ai│
│                                          ││                                                                        ││ 14:03:12  item/completed  commandExecution  exit 0 · 1.2s              observed│
│                                          ││                                                                        ││ 14:03:12  tokens  30,455 total · context 12% of 258.4k                 observed│
│                                          ││                                                                        ││                                                                                │
│                                          ││                                                                        ││                                                                                │
│                                          ││                                                                        ││                                                                                │
│                                          ││                                                                        ││                                                                                │
│                                          ││                                                                        ││                                                                                │
│                                          ││                                                                        ││                                                                                │
│                                          ││                                                                        ││                                                                                │
│                                          ││                                                                        ││                                                                                │
│                                          ││                                                                        ││                                                                                │
│                                          ││                                                                        ││                                                                                │
│                                          ││                                                                        ││                                                                                │
│                                          ││                                                                        ││                                                                                │
│                                          ││ ↑↓ select · Enter inspect · z zoom                                     ││ Enter full screen · [ ] previous / next agent                                  │
│ ↑↓ move  ←→ fold  Enter inspect  z zoom  ││                                                                        ││                                                                                │
╰──────────────────────────────────────────╯╰────────────────────────────────────────────────────────────────────────╯╰────────────────────────────────────────────────────────────────────────────────╯
╭─ D · QUOTA & TELEMETRY ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────── quota read 14:03:12 · pushed ─╮
│ 5-hour  ▰▰▰▰▰▰▱▱▱▱▱▱▱▱▱▱▱▱▱▱▱▱▱▱▱▱▱▱▱▱ 21%  resets 16:40          7-day  ▰▰▰▰▰▰▰▰▰▰▰▱▱▱▱▱▱▱▱▱▱▱▱▱▱▱▱▱▱▱ 38%  resets Mon 09:00          plan plus · credits N/A                                       │
│ model gpt-5.6-sol   context 30% of 258.4k (orchestrator)   tokens 184.3k session (derived: sum of 3 threads)   cost N/A (Codex reports none)                                                         │
│ agents 1 active (derived) / limit N/A (agents.max_threads not set)   time 2m 40s turn · 5m 12s session   errors 0   retries 0   unrecognised events 0                                                │
│                                                                                                                                                                                                      │
╰──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╯
 Enter send · Ctrl+J newline · Ctrl+C copy or interrupt · Ctrl+G editor · F6 / Shift+F6 zone · F2 approvals · z zoom · F12 mouse capture · F1 keys · Ctrl+Q quit                                        
```

- A grows to 44 columns, with turn history below the current plan.
- C splits into the agent list (74 columns) and a live preview of the
  selected agent's last events, so several agents can be watched while the
  orchestrator talks (AM2 C11).
- D is three lines and names each source's time ("quota read 14:03:12 ·
  pushed").

### Below 80x24

Below 80 columns or 24 rows, the screen shows only "Terminal too small:
80x24 needed, now 72x20" and the quit key. The session keeps running and
every event is still applied to the state, so nothing is lost; the dashboard
returns as soon as the terminal is large enough.

## Interaction flow

The loop of AM4 F1-F10 and AM2 I1-I9, with the machine that each step
drives (events-and-state.md):

```text
 launch ──► detect CLIs ──► STARTUP SCREEN ◄─────────────── Esc (answers kept)
 (F1, I1)   (S2)            choose CLI (F3)                          │
                            enter directory (F4)                     │
                            Enter: validate (F5) ──✗──► message      │
                                  │ ✓                   under field  │
                                  ▼                                  │
                            DASHBOARD, starting (F6) ──handshake──► StartFailed
                                  │ handshake ok                     card
                                  ▼
                 ┌──────────► READY: type in B (F7, I2) ◄──────────────────┐
                 │                │ Enter                                  │
                 │                ▼                                        │
                 │          turn Submitted ─► Running (F8, I3)             │
                 │                │                                        │
                 │   events ──────┼──► B streams text and tool lines       │
                 │   (genuine,    ├──► A stage tree updates (I4)           │
                 │    F9, I6)     ├──► C agents appear and run (I5)        │
                 │                ├──► D quota, tokens, context            │
                 │                │                                        │
                 │                ├──► approval card ─► F2: a / s / d / c  │
                 │                │    (WaitingApproval, I7)               │
                 │                ├──► Enter on an agent: inspection (I7)  │
                 │                │    Esc back                            │
                 │                ├──► Ctrl+C: Interrupting ─► interrupted │
                 │                ▼                                        │
                 │          turn completed / failed; result in B (I8)      │
                 └────────── follow-up, same session (F10, I9) ────────────┘

 Ctrl+Q at any point: confirm if a turn runs, close the CLI, restore the terminal.
```

While a turn runs, the draft can be edited but not sent: the editor's status
line says "turn running: the draft waits · Ctrl+C interrupt" (ASM-15).

## Zone A

The orchestrator and its workflow tree (AM2 A1-A6; AM4 D3; B2.2).

- **Header**: the orchestrator's model (from `session.started`), "turn N ·
  running 1m 12s" or "idle", the turn's outcome when it ended.
- **Stage tree**, with its origin in the heading:
  - Codex with a reported plan: "PLAN observed (turn/plan/updated)", the
    plan's steps with their status (events-and-state.md, Agent tree);
  - otherwise, and always for Claude Code: "STAGES inferred from activity",
    one stage per user turn, its agents and tool groups as children, each
    stage's status derived from its children (CC-10 q7).
  - Under a stage: tools (`✓ Read src/auth/*.ts`), agents (`◉ agent
    scan-routes`) and, under each agent, its current tool or pending
    approval, with `├─ └─ │` connectors.
  - A last grey line "○ (more as events arrive)" while the turn runs: the
    tree never pretends to know future steps.
- **Previous turns** fold to one line each ("TURN 1 ✓ completed · 31s"),
  unfoldable.
- **Keys** when focused: ↑ ↓ move, ← → fold and unfold, Enter inspects the
  agent or tool under the cursor, z zooms. Long trees scroll; the cursor's
  node stays visible.

## Zone B

The real conversation with the orchestrator (AM2 B1-B11; AM4 D3). Every
entry comes from the conversation record, which is built only from CLI
events (events-and-state.md, Conversation record); nothing is simulated.

- **Header**: "B · CONVERSATION" and a badge: "◉ Streaming" while text
  arrives, "◉ Running" during tools, "! Waiting for approval", "Idle",
  "Interrupting…".
- **Entries**:
  - **YOU** with the time: the message as sent, byte for byte (a long
    paste is shown folded, below);
  - **CLAUDE** or **CODEX** with the model: the AI text, streamed, rendered
    as Markdown;
  - tool lines `▸ Read src/auth/middleware.go · 212 lines`, with the
    status symbol and, right-aligned, the origin label `observed`; a failed
    tool shows `✗` and its error's first line;
  - agent lines `▸ Agent scan-routes started` (the agent's own text stays in
    C and in inspection: subagent text is not streamed by Claude Code,
    events-and-state.md, `conversation.message.delta`);
  - approval cards ([Approvals](#approvals));
  - dashboard notices in grey italics, marked "dashboard": "interrupted by
    you", "turn failed: …", "send refused: session not ready".
- **Long pasted messages**: a sent message over 20 lines shows its first 5 (ASM-22)
  and "… 9,995 more lines · Enter expand" (the CLIs fold large pastes the
  same way, TI 3.4); the record keeps the full text.
- **Follow mode**: new text keeps the view at the bottom; once the owner
  scrolls up, follow stops and "↓ 12 new" shows at the bottom edge; End
  resumes following.
- **The input** is the bottom of B, separated by a rule: the
  [Input editor](#input-editor).

### Markdown

AI text is Markdown (AM2 B4): headings, emphasis, lists, quotes, links
shown as text with their URL, tables, and fenced code blocks with a language
label and syntax highlighting.

- **Renderer**: Glamour v2 with Chroma v2 (both MIT), the pair a reference
  project uses with Bubble Tea v2 ([agent-dashboard.md](research/agent-dashboard.md),
  dependencies); or a smaller renderer of our own if Glamour's cost on long
  streamed messages is too high. The choice and its cost were not studied in
  plan 1 (ASM-17); either needs the owner's approval as a dependency.
- **Streaming**: only the open entry (the message receiving deltas) is
  re-rendered, at most once per frame; earlier entries are rendered once
  and cached by width. An unclosed code fence renders as code up to the
  cursor `▌`, so text does not jump between "plain" and "code" as the fence
  closes.
- **Code blocks** keep every character, including tabs (shown at 4
  columns) and trailing spaces; copying one copies its exact text (see
  [Selection and copy](#selection-and-copy)).
- **Thinking text** (`thinking_delta`, `item/reasoning/*Delta`) is shown
  (ASM-22) folded under the message as "thinking · 312 tokens", in grey, unfoldable.

## Zone C

The live subagents (AM2 C1-C11; AM4 D3).

- **Header**: "C · LIVE AGENTS", then "N ACTIVE · M DONE" on the right,
  counts derived from the agent machines (`Running` and `WaitingApproval`
  are active).
- **One row per agent**: status symbol, label (Claude Code's description or
  `agent_id`; Codex `agentPath`), model in its colour or "model N/A"
  (Codex children, ASM-14), current task (last tool and its target, or
  "waiting: approval Edit", or "completed · 3 files read"), elapsed time,
  tokens. A retry shows "↻ 2" after the time (AM2 C7). Rows are ordered:
  waiting for approval, running, then ended, most recent first (ASM-22).
- **Preview** (wide only): the selected agent's last events, timestamped,
  each with its origin label; it follows the selection.
- **Keys** when focused: ↑ ↓ select, Enter inspects, z zooms.
- Background agents that outlive their turn stay listed and running (P3,
  P3b): the agent machine, not the turn, decides their status.

## Zone D

Quota and telemetry (AM2 D1-D9; AM4 D3, D5, D6). Every value is a `Value`
(events-and-state.md, State model): present with its origin and source, or
"N/A" with its reason. Values by provider:

| Value | Claude Code | Codex |
| - | - | - |
| 5-hour, 7-day quota | percentage bar, from `unifiedWindows` **U**, marked † "undocumented field"; reset time | percentage bar from `account/rateLimits/*`, windows identified by `windowDurationMins`; reset time |
| Context | % of the window, from `get_context_usage` or derived from usage | % of the window, from `thread/tokenUsage/updated` |
| Tokens | session total from `result.usage` and messages | session total, derived: the sum over threads, labelled so |
| Cost | "$0.42 API-equiv. estimate", from `total_cost_usd`, never mixed with quota | "cost N/A (Codex reports none)" |
| Model | from `system/init` and each message | from `thread/start`; children "N/A" |
| Agents | active count, derived; "limit N/A" | active count, derived; "limit N/A (agents.max_threads not set)" |
| Time | turn and session durations | same |
| Errors, retries, unrecognised events | counters; Enter on D lists the last errors | same |

- **Bars**: 5-hour violet, 7-day amber (RD "Layout mock-up"); a bar turns
  red from 90% (ASM-13). A window whose reset time has passed shows "N/A
  (window reset; waiting for the next report)" until a new value arrives
  (TEL rules).
- **Freshness**: the header shows when the quota was last read ("updated 4s
  ago", or "quota read 14:03:12 · pushed"), so a stale value is visible.
- **N/A** is grey, never zero, and always carries a short reason on screen
  or in the zoomed panel: "N/A (no report yet)", "N/A (API key: no
  subscription window)", "N/A (plan has no 5-hour limit)" (TEL CC, TEL CX).

## Agent inspection

Enter on an agent in A or C opens it full screen (AM2 C5, C6, C8, U7; B2.3):

```text
 >_ AGENT COMMAND CENTER                                                 ● LIVE
 ◀ back (Esc)   scan-routes · sonnet-5-5 · ◉ running 41s · 12.1k tok   [ ] prev/next
 ───────────────────────────────────────────────────────────────────────────────
  Messages │ Tools │ Diff │ Raw events
 ───────────────────────────────────────────────────────────────────────────────
  14:02:31  started by orchestrator · "Scan the auth routes"            observed
  14:02:33  ▸ Grep "refresh" src/                                       observed
  14:02:34  ✓ 14 matches                                                observed
  14:02:40  Checking which routes call refresh before the move…               ai
  14:02:52  ▸ Bash go test ./auth                                       observed
  …
```

- **Tabs** (← → or 1 to 4): **Messages** (its text, as the CLI delivered it;
  for Claude Code whole messages, forwarded); **Tools** (each call with its
  input, status, duration and result, including test output: AM2 C6);
  **Diff** (its file changes: Codex `fileChange` diffs observed, Claude Code
  diffs derived from `Edit` and `Write` inputs and labelled "derived");
  **Raw events** (the agent's raw event lines, for diagnosis, unrecognised
  ones included).
- `[` and `]` move to the previous and next agent; Esc returns to the
  dashboard with the same focus as before (AM2 C9).
- The view updates live: inspection is a rendering of the same state.

## Approvals

The CLI's native approvals, relayed (AM2 B6, I7; AM4 W3; events-and-state.md,
Approval machine). An approval request appears as a card in B, attached to
the agent that asked, and marks that agent "!" in A and C.

- **The card**: who asks (orchestrator or agent label), the tool and its
  target, a summary (`+12 −0`, the command line), and the first lines of the
  diff or command; the available answers as keys.
- **Answers** (keys act only when the card has focus; F2 gives it focus,
  oldest pending first; ↑ ↓ move between pending cards):

  | Key | Answer | Claude Code | Codex |
  | - | - | - | - |
  | a | allow once | allow (**P**) | `accept` (**P**) |
  | s | allow for the session | not offered in the MVP (CC-4) | `acceptForSession` (**D**), shown only when offered |
  | d | deny / decline | deny with a message (**D**) | `decline` (**D**) |
  | c | cancel the turn | deny, then interrupt | `cancel` (**D**) |
  | Enter | full diff or command, full screen | | |
  | Esc | later: back to the input, still pending | | |

- **Typing never answers an approval**: letters typed in the editor stay in
  the draft; only a focused card takes a, s, d, c.
- Options that would persist a rule in the CLI's configuration ("always
  allow", execpolicy amendments) are never offered
  ([reliability.md, Configuration untouched](reliability.md#configuration-untouched)).
- After the answer, the card shrinks to one line: "✓ allowed by you
  14:03:20" or "✗ denied by you", observed as sent. An approval that
  expires (turn ended, CLI resolved it) shows "expired, not answered".
- **Questions** (`AskUserQuestion`, Claude Code): shown as a card with the
  questions and options; until the round trip is verified (CC-10 q3) the
  answer is typed as the next message, as events-and-state.md describes.

## States

How each state looks, wherever it occurs (B3.3; AM2 D9):

| State | Shown as |
| - | - |
| Loading | a restrained spinner and what is awaited: "detecting…", "starting Claude Code…", "waiting for the session" |
| Streaming | "◉ Streaming" in B's header; the cursor `▌` at the end of the open entry |
| Running a tool | the tool line with ◉; in C, the agent's task |
| Waiting for approval | "!" in the colour of attention, on the agent in A and C, the card in B, "! Waiting for approval" in B's header |
| Interrupting | "Interrupting…" in B's header; after 10 s, "the turn did not stop · F to force stop (ends the session)" (ASM-16) |
| Interrupted, cancelled | a notice "interrupted by you" in B; open tools end "unfinished" (inferred) |
| No data | "N/A" in grey with its reason; never a zero, never a blank |
| Error | ✗ in red with the first line of the error; the full error in inspection or in D's error list |
| Session crashed | B shows the exit code and stderr tail; send is disabled; "Ctrl+R resume · Ctrl+Q quit" (resume only once the session had reached Ready; earlier, the StartFailed card); the draft is kept; nothing restarts without the key ([reliability.md, Resumption](reliability.md#resumption)) |
| CLI not installed | on the startup screen only, "not installed" with the reason; not selectable |
| Unrecognised events | counted in D; listed in Raw events; never dropped |
| Mock mode | "◆ MOCK — not a live session" in the title bar of every screen |

## Origins on screen

Each line or value carries its origin (events-and-state.md, Origins; AM2
U9; TEL rules):

| Origin | Look |
| - | - |
| `ai` | normal text, under the model's label; in lists and previews, right-aligned "ai" |
| `observed` | right-aligned grey "observed" on event lines; values plain |
| `derived` | right-aligned "derived", and the rule in the zoomed panel ("sum of 3 threads") |
| `inferred` | italic, with "inferred" (the stage tree's heading says "inferred from activity") |
| `mock` | the title badge "◆ MOCK — not a live session"; every value also labelled mock in inspection |
| undocumented source | "†" after the value and "† undocumented field" in the panel's header (AGENTS.md, "Prefer documented interfaces") |

At narrow width the right-aligned labels are dropped from conversation
lines but kept in inspection and in the zoomed panels; the inferred stage
strip keeps "(inferred)".

## Colours and symbols

The dark technical style of RD: the terminal's own dark background, thin
rounded frames, pale violet panel titles (RD "Layout mock-up").

| Symbol | Meaning | Colour |
| - | - | - |
| ✓ | done | green |
| ◉ | in progress | amber |
| ○ | waiting, queued | grey |
| ! | needs the owner (approval, question) | pink |
| ✗ | failed, denied | red |
| ↻ n | retries | amber |
| ▸ | a tool or agent event line | grey |

Model colours, after RD's legend (one colour per model family, ASM-13):
Opus violet, Sonnet orange, Haiku blue, OpenAI GPT models teal, unknown or
N/A grey. The model name is always written next to its colour, and every
status has its symbol, so nothing depends on colour alone (a 16-colour
terminal or `NO_COLOR` still reads correctly). Colours are configurable in
the application's own configuration file (ASM-7).

| Role | Value (ASM-13) |
| - | - |
| Frame | `#3B3F51` |
| Panel title | pale violet `#C4B5FD` |
| Done | `#4ADE80` |
| In progress, 7-day bar | `#FBBF24` |
| Attention | `#F472B6` |
| Failed | `#F87171` |
| Waiting, hints, origin labels | `#6B7280` |
| 5-hour bar, Opus | `#A78BFA` |
| Sonnet | `#FB923C` |
| Haiku | `#60A5FA` |
| GPT models | `#2DD4BF` |

How Lip Gloss v2 maps these on a 256- or 16-colour terminal was not studied
in plan 1: Not verified, checked by hand in plan 4.

## Animations

Restrained, and only while something runs (AM2 U1; RD "Animated dots on
connectors").

- A spinner on the one line that is running (the streaming badge, a running
  tool, a starting session), at 10 frames per second; it stops the moment
  the work stops (ASM-18).
- Elapsed times tick once per second.
- In zone A, the connector of the running branch carries a moving dot, the
  reference's "animated dots", at the same 10 fps (ASM-18).
- No slide, fade or blink; no animation when idle, so an idle dashboard
  draws nothing.

### Rendering

- Events are coalesced: the forwarder hands the state at most one batch per
  frame, at most 30 frames per second (ASM-3; architecture.md, Threads of
  execution). Typing is not delayed by the batch: a key is applied and drawn
  in the next frame.
- Bubble Tea v2's cell renderer redraws only the cells that changed
  (`cursed_renderer.go`, TI 2.2), so streaming text does not repaint the
  screen and does not flicker (AM2 U8). Whether it also wraps frames in
  synchronized output (mode 2026) was not read in plan 1: Not verified.
- Claude Code documents stale fragments on WT through ConPTY in its
  alternate screen (TI 6.2); if seen, an option forces full repaints.
- The renderer never computes text: wrapping, Markdown and highlighting are
  cached per entry and width, and only visible rows are drawn.

## Focus and keys

The input in B has the focus by default (AM4 D4; AM2 U10); Esc always
returns to it. Bindings are lists, configurable in the application's own
configuration, never a CLI's (TI 4.3; ASM-7, ASM-9). The defaults avoid the
keys Windows Terminal takes (TI 9). F1 or `?` (outside the editor) shows
this table as an overlay; whether VS Code's terminal passes F1 through is
Not verified (TI 9).

**Anywhere**

| Key | Action | Source |
| - | - | - |
| Enter | send the draft (in the editor) | TI 4.3 |
| Ctrl+J | newline (works in every terminal) | TI 4.1, 4.3 |
| Shift+Enter | newline, when the terminal disambiguates it | TI 4.3 |
| Alt+Enter | newline, except on WT (fullscreen) | TI 4.3, 9 |
| `\` then Enter | newline (Claude Code's convention) | TI 4.2 |
| Ctrl+C | copy the selection if there is one; else interrupt the running turn; else show "Ctrl+Q quits" | TI 6.3 item 4; ASM-9 |
| Ctrl+Shift+C | copy the selection | TI 6.3 |
| Ctrl+O | copy the orchestrator's last message (as Codex does) | TI 6.2; ASM-9 |
| Ctrl+G | open the draft in `$EDITOR` | TI 8 (both CLIs use Ctrl+G) |
| F2 | go to the oldest pending approval | ASM-9 |
| F6 / Shift+F6 | next / previous zone (B, C, A, D) | ASM-9 |
| Tab / Shift+Tab | next / previous zone, outside the editor (in the editor Tab inserts a tab) | ASM-9 |
| Esc | close the overlay, leave zoom or inspection, clear the selection, back to the input | ASM-9 |
| Shift+PgUp / Shift+PgDn | scroll the conversation from any focus | ASM-9 |
| F12 | toggle mouse capture (native terminal selection when off) | TI 6.3 item 6 |
| F1, `?` | key overlay | ASM-9 |
| Ctrl+R | after a crash only: resume the same CLI session in a new process ([reliability.md, Resumption](reliability.md#resumption)) | ASM-9 |
| Ctrl+Q | quit; asks to confirm while a turn or an agent runs | ASM-9 |

**A focused panel** (A, C, D, inspection)

| Key | Action |
| - | - |
| ↑ ↓ | move the cursor (node, agent, line) |
| ← → | in A fold and unfold; in inspection change tab |
| Enter | inspect the agent; in D, list errors |
| z | zoom the panel full screen; z or Esc again restores |
| PgUp PgDn Home End | scroll; End resumes follow mode |
| `[` `]` | previous / next agent (inspection) |

**The approval card**: a, s, d, c, Enter, Esc, ↑ ↓ ([Approvals](#approvals)).

**The editor**: see [Input editor](#input-editor).

## Scrolling, collapsing, full screen

AM2 U4, U5, U7.

- Every panel scrolls on its own: PgUp and PgDn on the focused panel; the
  mouse wheel scrolls the panel under the pointer, whatever has the focus;
  Shift+PgUp and Shift+PgDn scroll the conversation from anywhere.
- Follow mode in B, C's preview and inspection: on at the bottom, off when
  scrolled up, with "↓ N new" shown; End resumes.
- Collapsing: tree nodes and previous turns in A; thinking text, long
  pastes and long tool outputs in B (first lines and "… N more lines");
  everything expands with Enter or → on the focused item.
- Full screen: z zooms the focused panel; Enter on an agent opens
  inspection; Enter on an approval card shows the whole diff. Esc goes back
  to the exact previous view.

## Input editor

Our own component (TI 11 item 2; AM3 M1-M9; AM2 B2), the same for every
provider: it has no provider code and sends a string to part 5's caller
(architecture.md, The five parts).

- **Keys**: arrows, Home and End (line), Ctrl+Home and Ctrl+End (buffer),
  Ctrl+← and Ctrl+→ (word); Shift with any movement selects; Ctrl+A selects
  all; Backspace and Delete, Ctrl+Backspace and Ctrl+Delete (word); Ctrl+Z
  undo, Ctrl+Y redo (both free on WT, TI 9; Ctrl+Z does not suspend on
  Windows); Ctrl+X cut, Ctrl+C copy (with a selection), Ctrl+V paste.
  Ctrl+V normally arrives as the terminal's bracketed paste (TI 9); if a
  terminal sends a raw Ctrl+V key, the editor reads the native clipboard and
  inserts it as a paste. Mouse: click places the cursor, drag selects,
  double-click a word, triple-click a line.
- **Size**: it grows with its content up to half of B, then scrolls; the
  status line shows the line count and the newline key that works in the
  detected terminal (TI 4.3, "On screen").
- **While a turn runs**: editable, not sendable; "turn running: the draft
  waits · Ctrl+C interrupt" (ASM-15). Enter then does nothing but flash that
  line.
- **After send**: the editor is cleared only once the message is written to
  the CLI; if the write fails, the draft is restored (events-and-state.md,
  Turn machine).
- **Content**: stored as given. No tab expansion, no trimming, no control
  stripping in storage; control characters are drawn visibly (`^X`); the
  sent string is exactly the stored text (AM3 M9).

### Paste

- A paste arrives as one `PasteMsg` and is inserted as text at the cursor,
  replacing the selection; it is never replayed as keys and **never
  submits**, whatever it contains (TI 3.2; AM3 M5).
- WT turns CRLF and LF into CR before the application sees them (TI 3.3):
  a pasted CR is stored as LF, one per line break. Tabs, trailing spaces
  and every other rune are kept. Bubble Tea v2 drops invalid UTF-8 and
  U+FFFD (TI 3.3): a documented limit, not fixed in the MVP.
- Between `PasteStartMsg` and `PasteEndMsg` the editor shows "pasting…"; if
  the end never arrives, it stays in paste state and no later key submits
  (TI 3.2 item 3).
- WT's large-paste and multiline-paste warnings are WT dialogs; the
  application neither suppresses them nor asks the owner to (TI 3.1).
- A paste over 20 lines stays fully in the buffer and editable; the editor
  shows it normally (the folding is only for sent messages in B).

### Budget

- Storage: a piece table or rope; one paste is one undo step.
- Rendering: only the visible rows, with wrap computed lazily for those
  rows (TI 3.4).
- The target: a keystroke drawn within 16 ms with a 10,000-line buffer;
  a 50,000-line paste inserted and drawn within 100 ms (ASM-10;
  reliability.md, Test plan). No line cap; if one is ever needed, it is
  visible.

### `$EDITOR`

Ctrl+G, as in both CLIs (TI 8; AM3 M8):

1. Resolve `VISUAL`, then `EDITOR`; split the value with `CommandLineToArgvW`
   rules on Windows and shell rules elsewhere, so `code --wait` works.
2. If neither is set, use a detected editor and say which ("no $VISUAL or
   $EDITOR: using `code --wait`").
3. Write the draft to a temporary file and close it; run the editor with
   `ExecProcess`; read the file back byte for byte; delete it.
4. A GUI editor without its blocking flag returns at once; the status line
   warns "the editor returned immediately: add --wait (or your editor's
   blocking flag)". Notepad's behaviour is Not verified (TI 8).
5. If the editor fails or is not found, show the error; **the draft is
   never discarded**.
6. While the editor runs, the tea loop is blocked; CLI events are buffered
   outside it and applied when the editor returns (architecture.md,
   Threads of execution). On return, the dashboard catches up in one frame.

## Clipboard

Copy and paste the same way for every provider (AM3 M2, M12; TI 5.3, 11
item 4):

- **Copy**: the native clipboard first (Win32 through
  `golang.design/x/clipboard` on Windows and macOS; `clip.exe` or
  PowerShell under WSL; `wl-copy` or `xclip` on Linux), and also OSC 52
  through `x/ansi` (wrapped in passthrough under tmux), plus `tmux
  load-buffer -w` under tmux. Over SSH, OSC 52 alone.
- **Feedback**: a short toast in the hint line names the path used: "copied
  42 lines · native", "copied · OSC 52 (terminal must allow it)", or the
  failure: "copy failed: no clipboard tool found (install wl-copy or
  xclip)".
- **Paste**: through the terminal's bracketed paste only; the application
  never reads the clipboard with OSC 52 (WT does not answer it, TI 10). The
  one native read is the raw-Ctrl+V case above.

## Selection and copy

Mouse selection in any panel, with clean text (AM3 M3; TI 6.3):

- A press is hit-tested with Lip Gloss's `Compositor.Hit` to pick **one**
  panel; the selection stays in it and maps to that panel's content model
  (logical lines without borders, padding or labels). A drag leaving the
  panel is clamped; at its edges the panel auto-scrolls.
- Double-click selects a word, triple-click a line; Shift+click and
  Shift+arrows extend.
- Copy on release by default (configurable; ASM-22), or Ctrl+Shift+C, or Ctrl+C
  while a selection exists. Without a selection, Ctrl+C keeps its interrupt
  meaning (WT passes Ctrl+C through when WT itself has no selection, TI
  6.3).
- What is copied is the content, not the drawing: a code block copies its
  exact text; a wrapped line copies as one line; origin labels and
  symbols are not copied.
- **Native escape**: F12 turns mouse capture off; the hint line then says
  "mouse capture off: Shift-drag selects (whole screen, borders included)".
  When capture is on, the key overlay names the terminal's bypass key
  (Shift in WT).

## What works where

From TI 10, for this design. "Check" is the manual check in plan 4.

| Need | Windows Terminal (native) | VS Code terminal | conhost | WSL2 in WT | tmux in WSL | Linux, macOS |
| - | - | - | - | - | - | - |
| Paste never submits | yes | yes (check) | VT mode only (check) | yes | yes (check) | yes |
| Byte-for-byte paste | CR→LF; controls stripped by WT; `trimPaste` effect Not verified | Not verified; may drop characters on large pastes | Not verified | as WT | Not verified | yes, terminal-dependent |
| Newline key | Ctrl+J; Shift+Enter Not verified through ConPTY | Ctrl+J; Shift+Enter needs a keybinding | Ctrl+J | Ctrl+J; Shift+Enter Not verified | Ctrl+J; Shift+Enter with extended-keys | Ctrl+J; Shift+Enter in most (not VTE) |
| Copy | native + OSC 52 | native | native; OSC 52 build-dependent | `clip.exe` or PowerShell + OSC 52 | OSC 52 passthrough + tmux buffer | native tools, then OSC 52 |
| Paste | Ctrl+V bracketed | Ctrl+V | VT mode | Ctrl+V bracketed | bracketed through tmux | bracketed |
| Mouse selection in a panel | yes | yes | Not verified (QuickEdit) | yes | yes | yes |
| Native escape | Shift-drag, F12 | Shift-drag, F12 | Not verified | Shift-drag, F12 | outer Shift-drag | Shift (Option on iTerm2, Fn on Terminal.app) |
| `$EDITOR` | `code --wait`; Notepad Not verified | same | same | `nvim` | same | `$EDITOR` |

The MVP's target is the first column (AGENTS.md: one Windows 11 machine,
architecture.md, Where it runs); the others are designed for and checked by
hand when available. Linux and macOS builds are Not verified.

## Assumptions used here

ASM-3 (30 fps), ASM-7 (own configuration file), ASM-8 (prefill flags),
ASM-9 (default keys), ASM-10 (editor budget), ASM-12 (layout geometry), ASM-13
(colours), ASM-14 (Codex child model N/A), ASM-15 (no send while a turn
runs), ASM-16 (interrupt timeout), ASM-17 (Markdown renderer), ASM-18
(animation rates), ASM-21 (timeouts), ASM-22 (display defaults); all are
listed in [architecture.md, Assumptions](architecture.md#assumptions).
