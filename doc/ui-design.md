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
dashboard opens; since plan 15, the recent and favourite projects under them
and a project browser (amendment 9).

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
                                                                                
                                                                                
                                                                                
 ↑↓ agent · Tab next field · Ctrl+O browse · Ctrl+W worktrees · Ctrl+Q quit    
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
                                                                                
                                                                                
                                                                                
 ↑↓ agent · Tab next field · Ctrl+O browse · Ctrl+W worktrees · Ctrl+Q quit    
```

Warnings (shown in amber with !, Start still allowed): a UNC path ("a
network share can be slow or disappear"); a `\\wsl$` or `\\wsl.localhost`
path ("files in Linux, used by a Windows agent CLI: slow, and unverified").

### Projects

The project store, `projects.json` in the dashboard's own folder
(`%LOCALAPPDATA%\agent-command-center\`, beside `drafts\`), keeps one entry
per project: its validated path, when a session last started in it, and
whether it is a favourite. It is written through a temporary file renamed
over the old one; paths are compared without case on Windows; it holds no
conversation data. A path is recorded once validation passes, even if the
agent CLI then fails to start. At most 10 recent projects are kept; a
favourite is never dropped. A mock run passes no store: nothing is read or
written and the screen is unchanged.

Under the project field, "Projects" lists favourites first (★), then the
other recent projects, each group by its last start: the directory's name,
its path and its age ("3 h ago", "never" for a directory starred from the
browser before any session). It is a Tab stop between the project field
and Start, shown only when not empty, with as many rows as fit so the Start
button stays whole at 80x24; when rows are left out the heading shows the
window ("1–3 of 11"). A store that cannot be read shows "Recent projects
unavailable:" and the reason in its place, never an empty list; one that
cannot be written shows "Recent projects not saved:" and the reason.

```text
    Projects   1–3 of 11
         ★ docs                C:\javadev\projets\docs               3 h ago
       › ★ scratch             C:\javadev\projets\scratch            1 d ago
         ★ acme-portal         D:\clients\acme-portal                  never
```

### Project browser

Ctrl+O on the startup screen (not while validating) opens the browser full
screen at the project field's resolved directory, else its nearest existing
parent, else the home directory. It lists directories only, sorted without
case, with ★ for a favourite, "git" for a directory holding `.git` (a
directory or a file) and "link" for a link or junction to a directory.
Hidden and system directories are left out and counted until Ctrl+H. The
first row, "Use this directory", chooses the directory shown; then `..`;
then the subdirectories. A directory is read off the update loop, with
"reading…" until it is listed; one that cannot be listed shows its error in
red with ✗, worded as validation words it, and stays browsable upwards. On
Windows, `..` at a drive's root lists the drives. The rows are windowed
around the selection, their window on the right of the count line.

```text
 >_ AGENT COMMAND CENTER                                        Project browser
────────────────────────────────────────────────────────────────────────────────

    Choose a project directory
    In      C:\javadev\projets
    Filter  █ type to filter
    ────────────────────────────────────────────────────────────────────────
        Use this directory                            start the session here
        ..                                                            parent
      ★ agent-command-center                                             git
    ›   billing-service                                                  git
        docs-site
        legacy-portal                                                   link
        tools                                                      link  git
    ────────────────────────────────────────────────────────────────────────
    5 directories · 1 hidden, Ctrl+H shows them
 Enter open · ↑↓ move · ← up · Ctrl+F star · Ctrl+H hidden · Esc back
```

| Key | Action |
| - | - |
| Typing, paste | filter by name, without case (a paste's first line); the first match is selected |
| ↑ ↓ PgUp PgDn Home End | move |
| Enter / → | open the directory; on "Use this directory" (Enter only), fill the project field and start, validated as a typed path |
| ← / Backspace | with an empty filter, go up with the directory left selected; Backspace otherwise edits the filter |
| Ctrl+F | star or unstar the selected directory (with a store) |
| Ctrl+H | show or hide the hidden directories |
| Esc | clear the filter, then close with the project field unchanged |

### Keys

| Key | Action |
| - | - |
| ↑ / ↓ | choose the agent CLI (on question 1); move in Projects |
| Tab / Shift+Tab | next / previous field (question 1, question 2, Projects, Start) |
| Enter | Start: validate both answers, then start; in Projects, with that project's path |
| f / Del | in Projects: star or unstar / forget that project |
| Ctrl+O | open the project browser |
| Ctrl+W | open the [Worktrees](#worktrees) view on the project field's repository |
| Editing keys | as in the [Input editor](#input-editor), on one line |
| Ctrl+Q | quit |

Below 100 columns the footer leaves Enter out to fit Ctrl+W within 79 cells;
from 100 it shows "Enter start" too. On Projects it reads "↑↓ · Enter start
· f star · Del forget · Ctrl+W worktrees · Ctrl+Q quit" (plan 16).

### Follows and departs

Against [RD, Image 1](reference-design.md#image-1-startup-configuration):

| Image 1 | Here | Why |
| - | - | - |
| Title bar "AGENT COMMAND CENTER", "New session" on the right | Same | AM4 S15 |
| Violet heading "Configure your workspace" and one line of help | Same; the help says "agent CLI" for "AI agent" | AM4 S15 |
| Two numbered questions, red asterisks, thin separators | Same | AM4 S15 |
| Three agents, including Google Gemini CLI | Two: Claude Code and OpenAI Codex | AM5 |
| Each agent a bare name | Each row adds the detected version and path, or "not installed" | AM4 S2, S3 |
| "Browse directories, recent projects, or enter a path" | A help line for manual entry and the resolved path shown live; the recent and favourite projects under it, and Ctrl+O for the browser | AM4 S13, X1-X3; amendment 9 (RD, "Where the images and amendment 4's text differ") |
| "Optional session settings" | Absent | AM4: exactly two questions; X10 |
| One wide light button "Preview configuration →" | One wide light button "Start session →", which validates then starts | AM4 S14 |

## Worktrees

Ctrl+W opens the Worktrees view full screen, on the startup screen (not
while validating) and on the dashboard (plan 16, amendment 10; ASM-25). It
lists one repository's git worktrees and creates, diffs, merges and removes
them through git itself (architecture.md, Session context). Its repository
is, at startup, the one holding the project field's directory (the current
directory when the field is empty); in session, the session's project; for
a mock run, the mock file's directory.

```text
 >_ AGENT COMMAND CENTER                                              Worktrees
────────────────────────────────────────────────────────────────────────────────

    Git worktrees
    Repo    C:\javadev\projets\demo-project
    ────────────────────────────────────────────────────────────────────────
    › main               clean                      …ev\projets\demo-project
      feature            clean   ↑2    session      …oject.worktrees\feature
      fix-login          +3      ↑1                 …ect.worktrees\fix-login
      docs               clean   ↑0                 …-project.worktrees\docs
      old-spike          missing                    …ect.worktrees\old-spike

    ────────────────────────────────────────────────────────────────────────
    5 worktrees
 Enter start here · n new · d diff · m merge into main · x remove · Esc back
```

**The rows.** The main worktree first, then the linked ones as git lists
them (`git worktree list --porcelain`). Each row shows:

- its branch, bold for main, or "detached" in grey;
- its state: "clean" in green, "+N" uncommitted changes (`git status
  --porcelain`) in amber, "missing" in red when its directory is gone,
  "bare", or "N/A" in red when its state cannot be read (the reason under
  the list while it is selected);
- "↑N" commits ahead of main's branch (`git rev-list --count`), "↑N/A"
  when they cannot be counted; nothing for main;
- "this session" (violet) where this dashboard runs, "session" (amber)
  where another dashboard session runs, else "locked" for a locked
  worktree;
- its path, elided from the left.

Rows are windowed around the selection, the window on the right of the
count line. Outside Windows the count line adds "other sessions: N/A: no
session registry" (ASM-25). With no git on the PATH the view says "✗ git
not found: worktrees unavailable"; outside a repository, git's "Not a git
repository" and the directory, both in red.

**New worktree** (startup only). `n` opens a one-line branch field under
the list and shows the path it will create, `<repo>.worktrees\<name>`,
`<name>` being the branch's last segment with every character but letters,
digits, `.`, `_` and `-` made `-`. An existing branch is checked out there;
a new one starts from main's HEAD. Enter creates it and selects its row;
git's reason is shown for an invalid name, an existing path or a branch
checked out elsewhere, and nothing is created.

```text
    ────────────────────────────────────────────────────────────────────────
    New worktree
    Branch  release-2.1█
    Creates C:\javadev\projets\demo-project.worktrees\release-2.1
    ────────────────────────────────────────────────────────────────────────
    5 worktrees
 Enter create · Esc cancel · Ctrl+Q quit
```

**Diff.** `d` shows the selected worktree's diff full screen: committed and
uncommitted changes against its merge base with main, then its untracked
files; for main, its uncommitted changes. Lines are coloured as git
colours them (headers bold, hunks violet, additions green, removals red),
a tab drawn as four spaces. At most 5000 lines, then "… N lines left out".

```text
 >_ AGENT COMMAND CENTER                                              Worktrees
────────────────────────────────────────────────────────────────────────────────
 Diff  feature against main
────────────────────────────────────────────────────────────────────────────────
 diff --git a/internal/auth/login.go b/internal/auth/login.go
 index 3f1c2a0..9b8e7d4 100644
 --- a/internal/auth/login.go
 +++ b/internal/auth/login.go
 @@ -12,7 +12,9 @@ func Login(user, password string) (*Session, error) {
      if user == "" {
          return nil, ErrNoUser
      }
 -    s, err := store.Open(user)
 +    s, err := store.Open(user, store.WithTimeout(5*time.Second))
 …
────────────────────────────────────────────────────────────────────────────────
 1–17 of 18 lines
 ↑↓ PgUp PgDn Home End scroll · Esc back to the list
```

**Merge and remove.** `m` merges the selected branch into main (`git merge
--no-edit` in the main worktree) and `x` removes the selected worktree
(`git worktree remove`, then `git branch -d`), each after "? Merge feature
into main?  y/n" or its remove question in amber. The outcome follows in
green with ✓, or in red with ✗ and git's reason. A conflict is aborted
(`git merge --abort`): "conflict: merge aborted, main unchanged". A branch
git will not delete because it is not merged stays: "branch kept:" and
git's reason. No call passes `--force`, `-f` or `-D`.

Refused before git runs, with the reason in red:

| Key | Refused when |
| - | - |
| `m` | the selection is main, missing, detached, or the worktree this session runs in; its state or main's cannot be read; either side has uncommitted changes; nothing is ahead, or that cannot be counted; main is detached; another dashboard session runs in main; this session runs in main and its turn is not idle |
| `x` | the selection is main, this session's worktree, one with uncommitted changes or an unreadable state, or one where another dashboard session runs (git itself refuses a locked one) |
| `n` | in session ("a new worktree is created from the startup screen"); while the list is being read |
| Enter, `d` | the selection's directory is missing |
| any operation | while another one runs ("wait: " and what it is doing) |

| Key | Action |
| - | - |
| ↑ ↓ PgUp PgDn Home End | move (in a diff, scroll) |
| Enter | at startup: fill the project field with the selected path and start, validated as a typed path |
| n | at startup: a new worktree |
| d | the diff, full screen; Esc back to the list |
| m / x | merge into main / remove, after y/n |
| y / n | answer the question |
| Esc | cancel the field or the question, leave the diff, close the view (not while an operation runs) |
| Ctrl+Q | close the view and quit, its question shown on the screen beneath |

Git never runs in the update loop: each call is a command with a time limit
(15 s for a read, 2 min for an operation) and a generation number, so a
stale result is dropped; meanwhile the view says "reading…" or the
operation running, and keys stay handled. The mouse is off while the view
is open. The session context is never changed by the view: in session it
only reads, merges and removes; a worktree is entered by starting a new
session on it from the startup screen. "session" knows the dashboard's own
sessions only, not an agent CLI started in another terminal (ASM-25).

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
- **right**: the provider's name and version, followed by "newer than
  tested" in the attention colour when the CLI is newer than the version its
  adapter was last probed with ([evaluation.md](evaluation.md), After a CLI
  update; plan 14, item 6); for Codex also the model,
  sandbox and approval policy reported by `thread/start` (P6b), because
  they decide what the agent may do; last, "approvals Manual" or
  "approvals Auto" (Auto in the attention colour), with "→ Auto …" while
  Claude Code is asked and "→ Auto (next turn)" until the next Codex
  message; when narrow, Codex's sandbox and policy drop first (plan 12).

## Layouts at three sizes

Three layouts, chosen by the terminal's width: narrow below 100 columns,
standard from 100 to 159, wide from 160; widths and splits below are ASM-12. The layout is recomputed on
every resize message; the state does not change. Panel heights share what
the fixed lines leave, and zone C's follows the agents (plan 18, brief
amendment 14): with no agent in play, C is an empty strip of 4 rows, its
title and "no agents yet" (3 rows at narrow width); with agents in play, as
many rows as they need, its title, a row each, its key hint and its border,
at most half the right column, B taking the odd row and keeping at least 12
rows (14 at wide width, where C also keeps 8 rows for the preview), and at
most 5 rows at narrow width. B takes the rest; D has 1 row at narrow width.
The agents in play are the state's list
([events-and-state.md](events-and-state.md#zone-cs-height)): those running,
or that appeared or ended since the owner's last prompt; C lists only them.

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
 Enter send · Ctrl+J newline · F6 zone · F2 approve · F3 Explorer · F1 keys     
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
│                                ││   turn running: the draft waits · Ctrl+C interrupt    1 line · Ctrl+G draft window │
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
 Enter send · Ctrl+C interrupt · Ctrl+G draft · F6 zone · F2 approvals · F3 Explorer · F1 keys · Ctrl+Q quit            
```

- A is a 34-column column, full height: orchestrator, model, turn and its
  elapsed time, then the stage tree.
- B above C on the right, as in the layout mock-up. B's input area sits at
  B's bottom edge, separated by a rule.
- D is two lines.
- The footer names F3 at every size: "F3 Explorer", the Explorer F3 opens
  when no file is in front ("F3 file" while B's entry cursor is on an entry
  that names one). When the whole line does not fit, as at 120 columns, it
  leaves out "Ctrl+J newline", which B's input shows above it, so that
  "Ctrl+Q quit" stays visible (from 108 columns here, from 173 in the wide
  layout) (plan 17).

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
│                                          ││   turn running: the draft waits until the turn ends · Ctrl+C interrupt                             2 lines · Ctrl+G draft window                         │
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
│ agents 1 active (derived) / limit N/A (not set; default not reported)   time 2m 40s turn · 5m 12s session   errors 0   retries 0   unrecognised events 0                                                │
│                                                                                                                                                                                                      │
╰──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╯
 Enter send · Ctrl+J newline · Ctrl+C copy or interrupt · Ctrl+G draft · F6 / Shift+F6 zone · F2 approvals · F3 Explorer · z zoom · F12 mouse capture · F1 keys · Ctrl+Q quit                           
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
  node stays visible. PgUp, PgDn and the wheel scroll the tree, Home shows
  its first row and End its tail (plan 5, item 5): see [Scrolling,
  collapsing, full screen](#scrolling-collapsing-full-screen).

## Zone B

The real conversation with the orchestrator (AM2 B1-B11; AM4 D3). Every
entry comes from the conversation record, which is built only from CLI
events (events-and-state.md, Conversation record); nothing is simulated.

- **Header**: "B · CONVERSATION" and a badge: "⠋ Streaming" while text
  arrives, "⠋ Running" while the turn runs otherwise (tools, subagents,
  thinking), the spinner turning in both (plan 4, step 8); "! Waiting for
  approval", "Interrupting…", and "Idle" when the session is ready with no
  turn.
- **Entries**:
  - **YOU** with the time: the message as sent, byte for byte (a long
    paste is shown folded, below);
  - **CLAUDE** or **CODEX** with the model: the AI text, streamed, rendered
    as Markdown;
  - tool lines `▸ Read src/auth/middleware.go · 212 lines`, with,
    right-aligned, the origin label `observed`; a failed call reads
    `✗ Bash · exit status 1: FAIL pkg/a`, its output's first non-blank
    line in red (plan 5, item 6), `✗ Grep · failed` when it wrote nothing,
    and a call denied at its approval `✗ Edit · denied`. The error is
    content: a selection copies the name and the error as two lines;
    **a Codex command** (`commandExecution`) that exited non-zero reads
    `✗ commandExecution · exit 2 · <last non-blank line>`, or
    `✗ commandExecution · exit 2` when its output is blank (plan 6, item 1).
    Why: Codex 0.163.0 sends one output, "aggregated from stdout and
    stderr" (its generated schema, `commandExecution.aggregatedOutput`), and
    no notification of an agent's command separates the streams, so the
    error line cannot be singled out; the exit code comes from the item's
    `exitCode`, never parsed from the text, and a failing command most often
    ends with its error. Enter still shows the whole output; Claude Code's
    failures and Codex's without an exit code keep the first-line rule;
  - agent lines `▸ Agent scan-routes started` (the agent's own text stays in
    C and in inspection: subagent text is not streamed by Claude Code,
    events-and-state.md, `conversation.message.delta`);
  - approval cards ([Approvals](#approvals));
  - dashboard notices in grey italics, marked "dashboard": "interrupted by
    you", "turn failed: …", "send refused: session not ready".
  - **Slash commands on Codex**: Codex runs `/status`, `/model` and the rest
    in its own terminal only; through `app-server` the text would reach the
    model as a message (plan 4, L5). A message whose first word is a slash
    command (`/` then lowercase letters and `-`; a path such as `/tmp/x`
    has a second `/` and is not one) is held back once, with a grey note
    of the dashboard's (role `dashboard`, every row under the text):
    "· /status is a command of Codex's own terminal; it does not work
    here, so nothing was sent. Zone D shows the 5h and weekly limits." and
    "Press Enter again to send it to the model as a message.". The draft
    stays; the same text sent again goes to the model, another text is
    checked anew. Claude Code runs its slash commands through the stream,
    so nothing is held there.
- **Long pasted messages**: a sent message over 20 lines shows its first 5 (ASM-22)
  and "… 9,995 more lines" (the CLIs fold large pastes the
  same way, TI 3.4); the record keeps the full text.
- **Expanding** (plan 5, item 4): B has an entry cursor. ↑ from an empty
  input puts it on the newest entry B shows in part: a folded message, a
  tool line (one row, its output not shown), a failure (one row, cut to the
  width). Its row turns to the attention colour, `▸` in front and
  "Enter full view · ↑↓ entries · Esc back" on the right, without changing
  B's height; the bottom hints say the same. ↑ ↓ move between such
  entries, B scrolling to keep the marked row in view; ↓ past the newest,
  Esc or a typed character return to the input (the character is typed
  there). With a draft, ↑ moves in the draft as before. A mouse click is not
  used: a click on a row starts a selection. Enter or → shows the entry
  whole, full screen, as the approval's full view does: a message with
  its time and line count, every line; a tool with its name, status,
  elapsed time, origin, its input (a diff coloured, a command, else the raw
  input) and its whole output, or "no output", updated live while it runs;
  a failure with every line. It scrolls with ↑ ↓ PgUp PgDn Home End and the
  wheel, copies with a selection, and Esc returns to B as it was, the
  cursor on the same entry.
- **Follow mode**: new text keeps the view at the bottom; once the owner
  scrolls up, follow stops and "↓ 12 new" shows at the bottom edge, at the
  right end of B's last row only: the row's start stays drawn, cut before the
  badge, and a selection across it copies its text, never the badge (plan 6,
  item 2); the entry cursor reveals one row less while the badge shows, so
  its marked row is never the badge's row. End resumes following.
- **The input** is the bottom of B, separated by a rule: the
  [Input editor](#input-editor).

### Markdown

AI text is Markdown (AM2 B4): headings, emphasis, lists, quotes, links
shown as text with their URL, tables, and fenced code blocks with a language
label and syntax highlighting.

- **Renderer**: our own, with no new dependency (ASM-17; the owner's
  decision of 2026-10-16). Plan 4 step 3 measured Glamour v2 re-rendering a
  streamed 2,000-line message at p95 590 ms per frame, and 3.5 ms with a
  per-block cache. Ours (`markdown.go`, plan 4, step 8) styles and never
  rewrites: every character is drawn as received, markers included (`#`,
  `**`, backticks, `|`, fences, a link's URL), so a row's text, its wrap and
  what a selection copies are those of the plain text
  (`TestMarkdownSameText` compares the two frames at three widths). A
  line's block kind comes from its own text and the fences above it; its
  inline styles are computed only for the rows a frame shows, over the whole
  source line, so a code span that wraps keeps its style on the next row.
- **Styles**: markers grey; headings pale violet bold; list bullets pale
  violet; strong bold; emphasis and quotes italic; inline code and code
  blocks blue; links blue underlined, their URL grey italic; a fence's
  language amber. In code: keywords violet bold, strings green, numbers
  orange, comments grey italic. Comments start with `#` in shell, Python,
  YAML, TOML, Ruby, PowerShell and the like, `--` in SQL, Lua and Haskell,
  `//` otherwise. Outside the `#` languages a `/*` comment runs on to its
  `*/` over the lines of its block, the code after `*/` highlighted again,
  and one left open ends with its fence (plan 5, item 12). A string ends with
  its line, except a backtick one in `go`, `js`, `javascript`, `jsx`, `mjs`,
  `ts`, `typescript` and `tsx` fences and a `"""` or `'''` one in `python` and
  `py` fences: it runs on, every line in the string colour, to its closing
  delimiter, the code after it highlighted again; one left open ends with its
  fence, and the next fence starts as code (plan 6, item 3). Inside such a
  string `/*` is string, inside a block comment a backtick is comment. One keyword set
  serves every language, since a word wrongly coloured costs only a colour.
  Colours in [Colours and symbols](#colours-and-symbols).
- **Streaming**: each source line's wrap is cached by text and widths
  (`WrapCache`, kept for one frame after its last use), and rows are drawn
  only when shown, so a frame costs the visible rows plus one cached lookup
  per line. Measured by `TestStreamToFrame`, a 2,000-line answer at one line
  per delta: p95 4.4 to 5.5 ms over three runs drawn plain, 2.1 to 5.7 ms
  styled (step 8). An unclosed code fence renders as code up to the
  cursor `▌`, so text does not jump between "plain" and "code" as the fence
  closes.
- **The inspection's Messages tab** styles an agent's AI messages the same
  way (plan 5, item 11); its notices, start, end, error and history lines
  stay as they were. The tab's wrap is unchanged (tabs as four spaces,
  control characters dropped), so its rows, hits and copies are those of
  the plain text (`TestInspectMarkdown`, at three sizes, scrolled top to
  bottom). An AI message's later rows are drawn in text colour, as in
  zone B, where they were grey.
- **Code blocks** keep every character, including tabs (shown at 4
  columns) and trailing spaces; copying one copies its exact text (see
  [Selection and copy](#selection-and-copy)).
- **Thinking text** (`thinking_delta`, `item/reasoning/*Delta`) is shown
  (ASM-22) folded under the message as "thinking · 312 tokens", in grey, unfoldable.

## Zone C

The live subagents (AM2 C1-C11; AM4 D3).

- **Header**: "C · LIVE AGENTS", then "N ACTIVE · M DONE" on the right,
  counts of the agents it lists, derived from the agent machines (`Running`
  and `WaitingApproval` are active).
- **The agents in play only** (plan 18, brief amendment 14, the owner's
  third message): each prompt the owner submits clears C of the agents of
  earlier prompts, back to "0 ACTIVE · 0 DONE" and "no agents yet"; an agent
  still running stays, and so do the agents of turns the CLI opens by itself
  for a background agent's notification. The earlier agents stay in zone A's
  tree, from which they can be inspected.
- **One row per agent**: status symbol, label (Claude Code's description or
  `agent_id`; Codex `agentPath`; else, for a Codex agent its spawn request
  alone names, "task: " and the first line of the request's `prompt`, cut
  at 24 characters with "…", replaced by the agent's own label when one
  comes, the whole prompt in the inspection; plan 10, item 1; a column as wide as the longest label
  and a space, at least 14 cells, so every model starts at the same cell;
  never so wide that the task keeps fewer than 12 cells, a longer label
  ending in "…", whole in the preview and the inspection; plan 7, item 1),
  the agent's own model in its colour
  (Claude Code: its forwarded messages' `message.model`, shown without the
  "claude-" prefix to fit the column, in full in the preview and the
  inspection; Codex: `subAgentActivity.model`, from Codex CLI 0.163.0, else the
  model its `spawnAgent` call requested, `collabAgentToolCall.model` for each
  of its `receiverThreadIds`, source "spawn request": the agent's own report
  always wins, in either order; plan 6, item 4, from the generated schema;
  recorded by P9a on the multi-agent v1 backend, plan 9) or
  "model N/A" (older Codex children, ASM-14; plan 5, item 7); a requested
  model is followed by a grey "*", footnoted "* requested model" in the
  header, as D's "†" (plan 6, item 5); then, when any listed agent has one,
  its reasoning effort in grey, a column as wide as the longest (Codex
  `subAgentActivity.reasoningEffort`, else the spawn request's, by the
  model's rule; plan 6, item 6), current task
  (last tool and its target, or "waiting: approval Edit", or "completed · 3
  files read"), elapsed time, tokens. A retry shows "↻ 2" after the time
  (AM2 C7). Rows are ordered: waiting for approval, running, then ended,
  most recent first (ASM-22).
- **Preview** (wide only): the selected agent's last events, timestamped,
  each with its origin label; its header shows the model with its source,
  "(reported)" or "(requested)", and the effort when there is one, "·
  effort high (reported)" (plan 6, items 5 and 6); where the whole header
  does not fit, it drops the effort, then the source, keeping the model
  whole, and only then nothing (plan 7, item 2); it follows the selection, and the wheel
  scrolls it.
- **Height** (plan 18, brief amendment 14): an empty strip while no agent is
  in play, 4 rows (3 at narrow width), the title with "0 ACTIVE · 0 DONE"
  and "no agents yet". As soon as the turn deploys agents, C grows to the
  rows they need, a row each with the title, the key hint and the border, up
  to half the right column (5 rows at narrow width), the list scrolling with
  the selection past that, and keeps that height after they finish, until
  the owner's next prompt clears it; a turn the CLI opens by itself does not.
  The key hint shows from 4 body rows, or when the agents leave a row for
  it. At wide width the strip spans the right column without the preview;
  with agents in play C keeps 8 rows for the preview beside the list.
- **Keys** when focused: ↑ ↓ select, Enter inspects, z zooms; PgUp, PgDn,
  Home and End scroll the list, the selection kept.
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
| Agents | active count, derived; "limit N/A" | active count, derived; "limit 3 (config)" from `config/read`, or "limit N/A (not set; default not reported)", or "limit N/A (set per backend; backend not reported)" when `features.multi_agent_v2` sets a different limit, or "limit N/A (not read)" |
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
 ◀ back (Esc)   scan-routes · claude-sonnet-5-5 (reported) · ◉ running 41s · 12.1k tok · effort N/A (not reported)
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
- **Header**: the label, the model in full with its source, "(reported)"
  by the agent or "(requested)" by the spawn call, or "model N/A (not
  reported)"; the status, time and tokens; then the reasoning effort with
  its source, or "effort N/A (not reported)" (Claude Code reports none per
  subagent); "[ ] prev/next" on the right when it fits, the footer keeping
  "[ ] agent" (plan 6, items 5 and 6).
- `[` and `]` move to the previous and next agent; Esc returns to the
  dashboard with the same focus as before (AM2 C9).
- The view updates live: inspection is a rendering of the same state.

## File viewer

F3 on the dashboard shows a project file in a window floating over it,
read-only, with syntax highlighting (plan 17, brief amendment 11;
architecture.md, File viewer). The dashboard is drawn and updated behind
it; only the window's cells cover it. Golden frame `file`, 80x24:

```text
 >_ AGENT COMMAND CENTER                            ◆ MOCK — not a live session
 ▣╭─ viewer.go ────────────────────────────────────────────── Go · 58 lines ─╮l
 A│ sample/viewer.go · edited by Edit                                        │e
╭─│ 29          }                                                            │─╮
│ │ 30      }                                                                │ │
│ │ 31      return Colour{}, false                                           │ │
│ │ 32  }                                                                    │ │
│ │ 33                                                                       │ │
│ │ 34  // Hex is the colour as "#rrggbb".                                   │ │
│ │ 35▌ func (c Colour) Hex() string {                                       │ │
│ │ 36▌     return fmt.Sprintf("#%06x", c.RGB)                               │ │
│ │ 37▌ }                                                                    │ │
│ │ 38                                                                       │ │
│ │ 39  // Names lists the palette's colour names, in order.                 │ │
╰─│ 40  func Names() []string {                                              │─╯
╭─│ 41      out := make([]string, 0, len(palette))                           │─╮
│ │ 42      for _, c := range palette {                                      │ │
│ │ 43          out = append(out, c.Name)                                    │ │
│ │ 44      }                                                                │ │
╰─│ 45      return out                                                       │─╯
╭─│ 46  }                                                                    │─╮
│ │ ↑↓ · / search · n N · g line · f full · Esc back · F3 close   line 35/58 │ │
╰─╰──────────────────────────────────────────────────────────────────────────╯─╯
 Enter or → full view · ↑↓ entries · Esc back to the input · F3 file · F1 keys
```

**The window.** `max(W*4/5, min(W-4, 84))` columns by `max(H*4/5,
min(H-2, 22))` rows, centred: 76x22 at 80x24, 96x32 at 120x40, 160x48 at
200x60; `f` fills the screen and back. A rounded frame in the title colour;
on its top edge the file's name on the left, "<language> · <N> lines" on
the right; the first row, the path shortened as elsewhere and what opened
it ("read by Read", "edited by Edit", "changed by fileChange", "quick
open", "Explorer"). Each line: its number right-aligned in grey, then the code, tabs to
the next multiple of 4, a long line cut with "…" and ←→ shifting it by 8
columns. The cursor line has zone A's cursor row background; the lines the
agent edited have their numbers in the attention colour and "▌" after them.
The footer: the keys on the left, "line <cursor>/<N>" on the right, or
"match <i> of <n>" after a search, or the field being typed; with an
approval pending, "◆ approval waiting · Esc, then F2" in amber.

**Colours.** Chroma's token types, coloured from the palette (not a Chroma
theme): keywords violet and bold, strings green, numbers orange, comments
grey and italic, functions and classes light blue, built-ins and types
blue, a diff's added lines green and removed lines red, headings in the
title colour, everything else (Chroma's error tokens too) in the text
colour; the 16-colour palette maps the same way. Matches of a search on an
amber background. The language comes from the name, else from the text,
else "Plain text".

**States**, in the window instead of the code: "reading…"; "colouring…" in
the header until the colours arrive, the lines shown uncoloured meanwhile;
"coloured around the view" for a file over 1 MiB, coloured 1000 lines on
either side of the view; "empty file". Refused, in red, before the file is
opened: "outside the project: <path>"; "a credential file: not opened";
"a directory, not a file"; "too large to show: <size> (32.0 MiB at most)";
"not found: <path>". A file with a NUL in its first 8 KiB: "binary file,
not shown · <size>".

**Opening.** F3 on zone B's entry cursor or in an entry's full view, on a
tool entry that names a file, opens that file (the entry hint shows "F3
file"):

| Tool | Cursor line | Marked |
| - | - | - |
| Read | its `offset` | nothing |
| Edit, MultiEdit | the first `new_string` found in the file, else the first `old_string` | those lines |
| Write | line 1 | nothing |
| Codex's fileChange | the first change's first added line | the first hunk's first added lines |
| any other with `file_path`, `notebook_path` or `path` | line 1 | nothing |

Anywhere else, or on an entry naming no file, F3 opens the
[Explorer](#explorer), and `/` there the quick open, a window of the same
size (golden frame `file-finder`, 80x24, inside the window):

```text
╭─ Quick open ─────────────────────────────────────────────────── 8 files ─╮
│ › vi▌                                                                    │
│ ▸ sample/viewer.go                                                       │
│                                                                          │
│ type to filter · ↑↓ select · Enter open · Esc back · F3 close     1 of 1 │
╰──────────────────────────────────────────────────────────────────────────╯
```

It lists every file under the project, relative paths sorted, without
`.git` folders, credential files or anything behind a link; at most
100,000, then "first 100,000 files". Typing filters, case ignored, the
query's letters in order: a match in the name first, then in the path,
then the shortest path. Esc goes back to the Explorer; F3 closes both.

| Key | Action |
| - | - |
| ↑ ↓ PgUp PgDn Home End | the cursor line, a line, a page, the first or last |
| ← → | shift the code sideways by 8 columns |
| / | type a search, case ignored unless it has a capital; Enter finds the next match; at most 10,000 |
| n / N | the next / previous match |
| g | type a line number; Enter goes there |
| f | full screen or the window |
| F2 | close the window and go to the oldest pending approval |
| Esc | cancel the field, else clear the selection, else back to the Explorer, its cursor on the file and the folders above it open, a file opened from an entry too |
| F3 | close the viewer and the Explorer, back to the dashboard |
| F1, F4, F12, Ctrl+C, Ctrl+Q | as on the dashboard |

Every other key is the window's: none reaches the draft. The mouse stays
captured: a drag selects in the window only and its release copies the
file's text it covers, as in the other panels; the wheel over the window
scrolls it; the dashboard behind takes no click. Reading, listing,
colouring and searching each run in a command with a generation number, so
a result for an older open is dropped and keys stay handled.

## Explorer

F3 with no file in front shows the project as a tree, as VS Code's Explorer
does, in a window of the viewer's size (plan 17, brief amendment 12;
architecture.md, File viewer). Golden frame `explorer`, 80x24, the cursor
on a modified file, a deleted file and a deleted folder listed below it:

```text
 ▣╭─ Explorer ───────────────────────────────────────────── mock · 11 items ─╮l
 A│ sample/viewer.go                                                         │e
╭─│ ▾ sample                                                              ●  │─╮
│ │     viewer.go                                                         M  │ │
│ │     viewer_test.go                                                    D  │ │
│ │ ▸ tools                                                               D  │ │
│ │   claude-api-key-context-refused.jsonl                                   │ │
│ │   claude-subagent-approval.jsonl                                         │ │
│ │   codex-api-key.jsonl                                                 U  │ │
│ │   codex-no-5h-window.jsonl                                               │ │
│ │   codex-plan6-limits.jsonl                                            !  │ │
│ │   codex-spawn-request.jsonl                                           A  │ │
│ │   codex-subagents.jsonl                                                  │ │
⋮
│ │ Enter view · / find · Esc back · F3 close                        2 of 11 │ │
╰─╰──────────────────────────────────────────────────────────────────────────╯─╯
```

**The window.** The viewer's frame and size, never full screen. On its top
edge "Explorer" on the left, the project's name and "<N> items" (the entry
rows shown) on the right; the first row, the cursor's path in the project
(a folder's ending in "/") and, on the right, git's note: "git status…"
until it answers, "not a git repository", "git not found" in grey, git's
own error in the attention colour. Each row: two cells of indent a level,
"▾" before an open folder and "▸" before a closed one in grey, folders in
the title colour and bold, files in the text colour, git's marker on the
right; the cursor row on zone A's cursor row background. Under a folder
being read, "reading…"; an empty one, "empty folder"; one that cannot be
read, its reason in the attention colour; a listing cut at 10,000 entries,
"first 10,000 entries". The footer: the keys, and "<i> of <N>".

**Reading.** Each folder is read when it is opened, in a command, inside
the project only: its folders, then its files, each sorted without case;
no `.git`, no credential file (by name, as in the quick open), nothing
behind a link. It opens as it was left in the session, its open folders
and cursor kept, each open folder read again.

**Deleted files.** A path git versions and the disk no longer has keeps a
row, so that a deletion, by the owner or by an agent, shows where it
happened (plan 17, item 6); one git never versioned leaves no trace. Git
reports each deleted file, never a folder: each one is a row under its
folder, in the folder's order, its name and its `D` in red; a folder deleted
whole is rebuilt from the paths under it, a folder row with `D`, never read
from the disk. A name the disk also has is listed once, from the disk; under
a listing cut at 10,000 entries no deleted folder is added, since it may be
past the cut. A folder empty on the disk but holding deleted paths lists
them. On a deleted row the path row reads "<path> · deleted, not on disk";
Enter or a click on a deleted file opens nothing.

**Git's markers.** One `git --no-optional-locks status --porcelain=v1 -z
--untracked-files=normal --ignored -- .` each time it opens, in a command;
it writes nothing.

| Marker | Meaning | Colour |
| - | - | - |
| M | modified | amber |
| A, R | added, renamed | green |
| U | untracked; every entry under an untracked folder too | green |
| ! | in conflict | red |
| D | deleted: versioned by git, gone from the disk; its name red too | red |
| ● | a folder on the disk holding any change, a deleted file's too | amber |
| (dimmed name) | ignored by git, and all under it; still listed | grey |

| Key | Action |
| - | - |
| ↑ ↓ PgUp PgDn Home End | move the cursor over the entries, notes skipped |
| → | open a folder; on an open one, its first entry |
| ← | close an open folder; else the folder holding the entry |
| Enter | open or close a folder; show a file in the [file viewer](#file-viewer); nothing on a deleted file |
| / | the quick open; Esc there comes back |
| Esc | close the folder the cursor is in, the cursor on it; at the top level, close |
| F3 | close, back to the dashboard |
| F2 | close and go to the oldest pending approval |
| F1, ?, F4, F12, Ctrl+C, Ctrl+Q | as on the dashboard |

Every other key, and a paste, is the Explorer's: none reaches the draft.
The mouse: the wheel moves the cursor by 3 entries; a click puts the cursor
on a row, a click on the cursor's row is Enter. Nothing is selected in it,
as in the quick open; F12 and the terminal's own selection copy from it.

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
  | a | allow once | allow (**P**); not for a question (plan 5, item 10) | `accept` (**P**) |
  | s | allow for the session | not offered (CC-4) | `acceptForSession` (**D**), offered unless the request's `availableDecisions` omits it; on Codex CLI 0.163.0, offered on file changes (no list), not on commands (their list omits it) (plan 5, item 8; plan 8) |
  | d | deny / decline | deny with a message (**D**) | `decline` (**D**) |
  | c | cancel the turn | deny, then interrupt | `cancel` (**D**) |
  | Enter | the whole diff, command or question, full screen; a, s, d, c answer there, Esc back to the card | Edit, MultiEdit, Write from the request | `fileChange` from its tool call's `changes[].diff` |
  | Esc | later: back to the input, still pending | | |

- **Manual or Auto** (plan 12, brief amendment 8): in Manual, every
  request the CLI sends is a card. In Auto, the CLI's own reviewer decides
  what it can (Claude Code's classifier, Codex's auto-review) and only what
  it escalates is a card, exactly as in Manual. A call Claude Code's
  classifier decided reads "▸ Write · approved by auto-review" in B; a
  Codex review is a "◈ auto-review approved · risk low · …" entry, its
  inspector marked [UNSTABLE]. Codex's Auto is its "Approve for me", so
  the bar's "sandbox" reads workspaceWrite once an Auto turn is sent, and
  readOnly again after a Manual one.
- **Typing never answers an approval**: letters typed in the editor stay in
  the draft; only a focused card takes a, s, d, c.
- Options that would persist a rule in the CLI's configuration ("always
  allow", execpolicy amendments) are never offered
  ([reliability.md, Configuration untouched](reliability.md#configuration-untouched)).
- After the answer, the card shrinks to one line: "✓ allowed by you
  14:03:20" or "✗ denied by you", observed as sent. An approval that
  expires (turn ended, CLI resolved it) shows "expired, not answered".
- **Questions** (`AskUserQuestion`, Claude Code): shown as a card with
  every question and every option, numbered, nothing cut; the selected
  question is marked ▸ while the card has the focus (plan 14, item 1; the
  round trip settled by P14a, CC-10 q3). Its keys, on the card and in its
  full view: `1`-`4` choose an option of the selected question (a single
  choice moves on to the next question; a multiple choice toggles `[x]`),
  ← → select a question, `a` sends the answers once every question has a
  choice (otherwise the toast "choose an option for each question first"
  and nothing is sent). The card then reads "✓ answered by you 14:03:20",
  with one grey line per question, "<question> → <answer>"; the full view's
  title reads "✓ ANSWERED". `d` denies it with "The user saw these
  questions in Agent Command Center, which cannot answer them here, and
  will answer them in their next message. Wait for that message; do not ask
  them again.", and the card reads "✗ … denied by you 14:03:20 · answer in
  your next message", for an answer the options do not hold; `c` (or
  Ctrl+C, or quitting) cancels it with the usual deny message and card line
  (plan 5, item 10).

## States

How each state looks, wherever it occurs (B3.3; AM2 D9):

| State | Shown as |
| - | - |
| Loading | a restrained spinner and what is awaited: "detecting…", "starting Claude Code…", "waiting for the session" |
| Streaming | "⠋ Streaming", its spinner turning, in B's header; the cursor `▌` at the end of the open entry |
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
Opus violet, Sonnet orange, Haiku blue, OpenAI GPT models teal, any other
model in the text colour. The model name is always written next to its colour, and every
status has its symbol, so nothing depends on colour alone (a 16-colour
terminal or `NO_COLOR` still reads correctly). Colours are configurable in
the application's own configuration file (ASM-7).

| Role | Value (ASM-13) | 256 colours | 16 colours |
| - | - | - | - |
| Frame | `#3B3F51` | 238 | 8 bright black |
| Panel title, 5-hour bar | pale violet `#C4B5FD` | 183 | 15 bright white |
| Text | `#E5E7EB` | 189 | 7 white |
| Done | `#4ADE80` | 78 | 10 bright green |
| In progress, 7-day bar | `#FBBF24` | 214 | 11 bright yellow |
| Attention | `#F472B6` | 205 | 13 bright magenta |
| Failed | `#F87171` | 203 | 9 bright red |
| Waiting, hints, origin labels | `#6B7280` | 60 | 8 bright black |
| Opus | `#A78BFA` | 141 | 5 magenta |
| Sonnet | `#FB923C` | 209 | 3 yellow |
| Haiku | `#60A5FA` | 75 | 12 bright blue |
| GPT models | `#2DD4BF` | 43 | 14 bright cyan |
| Code, inline code | `#93C5FD` | 117 | 6 cyan |
| Code keywords | `#A78BFA`, bold | 141 | 5 magenta |
| Code strings | `#86EFAC` | 121 | 2 green |
| Code numbers | `#FB923C` | 209 | 3 yellow |
| Code comments, Markdown markers | `#6B7280` (comments italic) | 60 | 8 bright black |
| Links | `#60A5FA`, underlined | 75 | 12 bright blue |
| Text on a title-coloured button or cursor | `#111827` | 234 | 0 black |
| Editor selection, background | `#4C4F6B` | 240 | 4 blue |
| Zone A cursor row, background (text `#E5E7EB`) | `#2E3247` | 23 | 8 bright black (text 15) |

The application draws the true colours whatever the terminal, except on a
16-colour one (plan 5, item 13). Bubble Tea detects the terminal's colour
profile and sends it at startup, and again when the terminal reports RGB
support; its renderer then writes each true colour as is, maps it to the
nearest of the 256, or drops every colour when the output is no terminal.
The 256 column is that mapping as measured (`internal\render\palette_test.go`,
TestPalette256): one index per colour, no two colours on the same index; the
zone A cursor row comes out a dark teal. On a 16-colour profile the renderer's
own mapping folds the roles together (done, GPT and strings on one green;
in progress, attention, failed and Sonnet on one red; title, text, Opus, Haiku
and code on one blue), so the application switches to the explicit palette of
the 16 column instead, and back when the profile changes, drawing every row
again (`internal\render\palette.go`). In it the statuses differ from one
another, as do the four model families and the code colours; frame, hints and
the cursor row share bright black, which the symbols and text keep readable.
Checked offline by TestColourProfiles and TestPaletteSwitch
(`internal\app\colour_test.go`): the same characters under true colour, 256,
16 and no terminal, as Windows Terminal, an `xterm-256color` and an `xterm`
receive them.

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
  screen and does not flicker (AM2 U8). It also wraps each frame in
  synchronized output (mode 2026), once the terminal reports the mode
  supported: at start, Bubble Tea v2.1.0 asks for modes 2026 and 2027
  (`tea.go`, `shouldQuerySynchronizedOutput`, true under WT), and on a report
  of 2026 supported and off it writes every frame inside `?2026h … ?2026l`
  (`cursed_renderer.go`); it also measures width by grapheme when 2027 is
  set. Measured in WT 1.23.20211.0 on 2026-10-17 (plan 5, item 14), the
  application on a mock replay with `TEA_TRACE`: WT answers
  `?2026;2$y ?2027;3$y` as plain VT input, and 92 of the 95 writes are
  wrapped; the other three are the query, the first frame, drawn before the
  answer arrives, and the exit sequence. `TestSynchronizedOutput` replays
  WT's recorded answer through the real program: every frame after it is
  wrapped, and with no answer or another one no frame is.
- Claude Code documents stale fragments on WT through ConPTY in its
  alternate screen (TI 6.2); if seen, an option forces full repaints.
- The renderer never computes text: wrapping, Markdown and highlighting are
  cached per entry and width, and only visible rows are drawn.

## Focus and keys

The input in B has the focus by default (AM4 D4; AM2 U10); Esc always
returns to it. Bindings are lists, configurable in the application's own
configuration, never a CLI's (TI 4.3; ASM-7, ASM-9). The defaults avoid the
keys Windows Terminal takes (TI 9). F1 or `?` (outside the editor) shows
this table as an overlay.

**Anywhere**

| Key | Action | Source |
| - | - | - |
| Enter | send the draft (in the editor) | TI 4.3 |
| Ctrl+J | newline (works in every terminal) | TI 4.1, 4.3 |
| Shift+Enter | newline; in Windows Terminal through win32-input-mode (below) | TI 4.3 |
| Alt+Enter | newline, except on WT (fullscreen) | TI 4.3, 9 |
| `\` then Enter | newline (Claude Code's convention) | TI 4.2 |
| Ctrl+C | copy the selection if there is one; else interrupt the running turn; else show "Ctrl+Q quits" | TI 6.3 item 4; ASM-9 |
| Ctrl+Shift+C | copy the selection | TI 6.3 |
| Ctrl+O | copy the orchestrator's last message (as Codex does) | TI 6.2; ASM-9 |
| Ctrl+G | open the draft in the [draft window](#draft-window), in the terminal; in it Ctrl+F full screen, Esc back | TI 8 (both CLIs use Ctrl+G); brief amendment 13 |
| F2 | go to the oldest pending approval | ASM-9 |
| F3 | the file the entry, full view or request in front names, in the [file viewer](#file-viewer); else the [Explorer](#explorer); in either, close back to the dashboard | plan 17; brief amendments 11 and 12 |
| F4 | approvals Manual ⇄ Auto, from any focus; Manual at every launch, never saved | plan 12; brief amendment 8 |
| F6 / Shift+F6 | next / previous zone (B, C, A, D) | ASM-9 |
| Tab / Shift+Tab | next / previous zone, outside the editor (in the editor Tab inserts a tab) | ASM-9 |
| Esc | close the overlay, leave zoom or inspection, clear the selection, back to the input | ASM-9 |
| Shift+PgUp / Shift+PgDn | scroll the conversation from any focus | ASM-9 |
| F12 | toggle mouse capture (native terminal selection when off) | TI 6.3 item 6 |
| Ctrl+W | the [Worktrees](#worktrees) view of the session's repository: list, diff, merge, remove | plan 16; brief amendment 10 |
| F1, `?` | key overlay | ASM-9 |
| Ctrl+R | after a crash only: resume the same CLI session in a new process ([reliability.md, Resumption](reliability.md#resumption)) | ASM-9 |
| Ctrl+Q | quit; asks to confirm while a turn or an agent runs | ASM-9 |

**Shift+Enter in Windows Terminal.** In VT input mode the console reports
Shift+Enter as a plain carriage return, the same as Enter (measured, plan 4,
step 3). The dashboard therefore asks for win32-input-mode (`CSI ?9001h`) when
it starts and turns it off when it ends, a panic included (`internal/app/run.go`);
each key then arrives with its modifiers. Two decoder faults of ultraviolet
in that mode (letter case recomputed from the modifiers; emoji
halves with virtual key 255 dropped) are fixed in a patched copy,
`third_party/ultraviolet/PATCHES.md`. Measured in Windows Terminal 1.23,
maximised and at the default size, 2026-10-16: Shift+Enter inserts a newline,
Enter sends, typed text and emoji are kept exactly, and the next program in
the console reads plain VT input after Ctrl+Q.

**A focused panel** (A, C, D, inspection)

| Key | Action |
| - | - |
| ↑ ↓ | move the cursor (node, agent, line) |
| ← → | in A fold and unfold; in inspection change tab |
| Enter | inspect the agent; in D, list errors |
| z | zoom the panel full screen; z or Esc again restores |
| PgUp PgDn Home End | page, top, end; End follows again (A, inspection) |
| `[` `]` | previous / next agent (inspection) |

**The approval card**: a, s, d, c, Enter, Esc, ↑ ↓ ([Approvals](#approvals)).

**B's entries** (↑ from an empty input, [Zone B](#zone-b))

| Key | Action |
| - | - |
| ↑ ↓ | previous / next folded message, tool or failure; ↓ past the last, the input |
| Enter or → | the whole entry, full screen; Esc back to the cursor |
| Esc, or typing | back to the input |
| F3 | the file a tool entry read or changed, in the [file viewer](#file-viewer) |

**The file viewer** ([File viewer](#file-viewer))

| Key | Action |
| - | - |
| ↑ ↓ PgUp PgDn Home End | move the cursor line; the wheel scrolls |
| ← → | shift the code sideways |
| / | search, case ignored without a capital; Enter finds |
| n N | next / previous match |
| g | go to a line number |
| f | full screen or the window |
| Esc | clear the selection, else back to the Explorer, on the file |
| F3 | close, back to the dashboard |

**The Explorer** (F3 with no file in front, [Explorer](#explorer))

| Key | Action |
| - | - |
| ↑ ↓ PgUp PgDn Home End | move; the wheel too, a click on a row |
| → ← | open a folder, or its first entry; close it, or go to its folder |
| Enter | open or close a folder; show a file in the file viewer |
| / | the quick open: type to narrow, Enter opens, Esc back |
| Esc | close the folder the cursor is in; at the top, close |
| F3 | close, back to the dashboard |

**The editor**: see [Input editor](#input-editor).

## Scrolling, collapsing, full screen

AM2 U4, U5, U7.

- Every panel scrolls on its own: PgUp and PgDn (a page, its rows less
  one), Home and End on the focused panel; the mouse wheel, 3 rows, scrolls
  the panel under the pointer, whatever has the focus, the focus unchanged;
  Shift+PgUp and Shift+PgDn scroll the conversation from anywhere.
- Follow mode in B and inspection: on at the bottom, off when scrolled up,
  with "↓ N new" shown; End resumes.
- Zone A's tree and C's preview rest at their tail; zone C's list at its
  top, the selection in view (plan 5, item 5). Scrolled, a pane is held:
  rows coming and going do not move it, ↑ ↓ keep the cursor or selection
  in view with the least move, and its hint row says so in amber, the rows
  out of view, then the way back: "↑ 12 ↓ 25 · End follows" (A), "· Home
  top" (C), "· scroll to the end follows" (the preview). A and the preview
  follow again at their end; the preview also for another agent. End in A
  with the cursor above the tail shows the tail and keeps following it. A
  three-row C (80x24) has no hint row. Zoomed panes scroll the same way.
- Collapsing: tree nodes and previous turns in A, unfolded with ← →; in B,
  long pastes (first lines and "… N more lines"), tool lines and failures
  (one row), each shown whole with Enter or → on B's entry cursor
  ([Zone B](#zone-b), Expanding).
- Full screen: z zooms the focused panel; Enter on an agent opens
  inspection; Enter on an approval card shows the whole diff; Enter on B's
  entry cursor shows the entry whole. Esc goes back to the exact previous
  view.

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

### Draft window

Ctrl+G, the key both CLIs give their external editor (TI 8; AM3 M8), opens
the draft in a window over the dashboard instead, in the terminal: no
external editor, the application never starts VS Code or any other program
for the draft (brief amendment 13). At 80x24:

```text
 >_ AGENT COMMAND CENTER                            ◆ MOCK — not a live session
 ▣╭─ Draft ──────────────────────────────────────────────────────── 7 lines ─╮l
 A│ > Review the colour palette in sample/viewer.go:                         │e
╭─│                                                                          │─╮
│ │   1. keep the five accents of the reference screen;                      │ │
│ │   2. check each one against the dark background;                         │ │
│ │   3. report any pair under 4.5:1, with its hex values.                   │ │
│ │                                                                          │ │
│ │   Do not change the file yet.                                            │ │
│ │                                                                          │ │
│ │                                                                          │ │
│ │                                                                          │ │
│ │                                                                          │ │
│ │                                                                          │ │
╰─│                                                                          │─╯
╭─│                                                                          │─╮
│ │                                                                          │ │
│ │                                                                          │ │
│ │                                                                          │ │
╰─│                                                                          │─╯
╭─│                                                                          │─╮
│ │ Enter send · Ctrl+J newline · Esc back           turn started by the CLI │ │
╰─╰──────────────────────────────────────────────────────────────────────────╯─╯
 Enter send · Ctrl+J newline · F6 zone · F2 approve · F3 Explorer · F1 keys
```

- **The window**: the file viewer's place and size (FileRect, [File
  viewer](#file-viewer)): 76x22 at 80x24, 96x32 at 120x40, 160x48 at
  200x60, centred; **Ctrl+F** fills the screen and back (`f` types a letter
  in a draft). The dashboard keeps updating behind it. On the top edge,
  "Draft" on the left, the line count on the right.
- **The same draft, the same editor**: the window shows `m.edit`, the input
  box's buffer, with every key of the [Input editor](#input-editor):
  Enter sends and closes the window (on an empty draft it does nothing and
  the window stays), Ctrl+J and the other newline keys add a line, Ctrl+C
  copies a selection, Ctrl+Z and Ctrl+Y undo and redo, a paste is kept byte
  for byte. While a turn runs, Enter does not send, as in the input box.
- **Esc** clears a selection first, then goes back to the input box with the
  draft kept. **Ctrl+G** inside the window does nothing more. F1 shows the
  key page over it and Esc comes back to it; F4, F12, Ctrl+O and Ctrl+Q do
  what they do everywhere. F2, F3 and F6 / Shift+F6 close the window, then
  go to the approvals, the Explorer or the next zone, as the footer says.
- **The footer**: the keys on the left, cut to fit ("Enter send · Ctrl+J
  newline · Ctrl+F full screen · Ctrl+C copy · F1 keys · Esc back" at its
  longest); on the right "line <cursor>/<N>", or "pasting…", or the
  editor's status when Enter would not send ("turn running: the draft
  waits · Ctrl+C interrupt").
- **The mouse**: a drag in the window selects in the draft and copies on
  release, as in the input box; the wheel moves the cursor 3 lines; a click
  outside the window does nothing while it is open.
- **Budget**: the window draws only its rows, from the editor's view around
  the cursor, as the input box does ([Budget](#budget)); a keystroke in it
  is measured with a 10,000-line draft by the timing step, and its
  allocations are bounded by the same guard as the input box's
  (`TestKeystrokeAllocs`).

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

- A press is hit-tested against the drawn frame to pick **one** panel:
  every span carries a reference to the content it shows (panel, entry,
  logical line, byte offset), or marks itself decoration. The selection
  stays in that panel and maps to its content (logical lines without
  borders, padding or labels). A drag leaving the panel is clamped; zone B
  scrolls when the pointer leaves it upward or downward. (Lip Gloss's
  `Compositor.Hit` was not needed.)
- A plain click focuses the panel and selects nothing; double-click selects
  a word, triple-click a line; Shift+click extends. In the editor the mouse
  sets the editor's own selection, and Shift+arrows extend it.
- Copy on release by default (configurable; ASM-22), or Ctrl+Shift+C, or Ctrl+C
  while a selection exists. Without a selection, Ctrl+C keeps its interrupt
  meaning (WT passes Ctrl+C through when WT itself has no selection, TI
  6.3).
- What is copied is the content, not the drawing: a code block copies its
  exact text; a wrapped line copies as one line; origin labels and
  symbols are not copied.
- **Native escape**: F12 turns mouse capture off; the hint line then says
  "mouse capture off: the terminal selects (whole screen, borders
  included) · F12 back on".
- B's session state (a failure card, "Starting …", "Session closed.",
  "Waiting for instructions") is content after the last entry: a failure
  card copies its title, each reason and stderr line whole (a cut line's
  "…" stands for its rest), "exit code N" and "stderr:", never the ✗, the
  indentation, the spinner or the key hint (plan 5, item 9).
- MVP limit (plan 4 step 3): only B scrolls during a drag.
  When capture is on, the key overlay names the terminal's bypass key
  (Shift in WT).

## What works where

From TI 10, for this design; the Windows Terminal column is measured (plan
4, step 3, criterion 9: real keyboard, mouse and clipboard input, maximised
and at the default size, 2026-10-16). The VS Code terminal and conhost are
out of scope (brief, amendments 6 and 7).

| Need | Windows Terminal (native) | WSL2 in WT | tmux in WSL | Linux, macOS |
| - | - | - | - | - |
| Paste never submits | yes (measured, 2,000-line paste included) | yes | yes (check) | yes |
| Byte-for-byte paste | yes, CRLF→LF: tabs, trailing spaces, wide, emoji, 94,908 chars (measured) | as WT | Not verified | yes, terminal-dependent |
| Newline key | Ctrl+J and Shift+Enter, through win32-input-mode (measured) | Ctrl+J; Shift+Enter Not verified | Ctrl+J; Shift+Enter with extended-keys | Ctrl+J; Shift+Enter in most (not VTE) |
| Typed text | exact, emoji included (measured) | as WT | Not verified | yes |
| Copy | native + OSC 52 (measured) | `clip.exe` or PowerShell + OSC 52 | OSC 52 passthrough + tmux buffer | native tools, then OSC 52 |
| Paste | Ctrl+V bracketed | Ctrl+V bracketed | bracketed through tmux | bracketed |
| Mouse selection in a panel | yes (measured) | yes | yes | yes |
| Native escape | F12, then the terminal's own drag and Ctrl+C (measured); Shift-drag Not verified | Shift-drag, F12 | outer Shift-drag | Shift (Option on iTerm2, Fn on Terminal.app) |
| Draft window (Ctrl+G) | in the terminal, no external editor (brief amendment 13; tested) | same | same | same |

The MVP's target is the first column (AGENTS.md: one Windows 11 machine,
architecture.md, Where it runs). The other columns are design notes, not
checked: plan 14 closed them (WSL and tmux are not used, and the dashboard
does not depend on them; the owner closed Linux and macOS and the checks by
hand, item 3), and the "Not verified" cells above are closed the same way.

## Departures

Where the MVP departs from this design, as found in plan 4, step 8's review
of every screen; each is shown honestly (no key or symbol advertised for
what is not built).

| This design | The MVP | Why |
| - | - | - |
| Approval card: s allows for the session | Never for Claude Code; for Codex unless the request's `availableDecisions` omits `acceptForSession`. Codex CLI 0.163.0 sends no list with a file-change request, so it is offered there; a command request lists accept, the amendment and cancel (probe P8), so it is not | Claude Code: CC-4; Codex: plan 5, item 8; plan 8 |
| Tool lines in B carry a status symbol, and a failed one `✗` with its error | `✗` and the error for a failed or denied call; `▸` for the others, their status in A and the viewer | Only the calls that need a look are marked in B (plan 5, item 6) |
| Thinking text folded under the message, "thinking · 312 tokens", unfoldable | "∴ thinking" marks a message that had a thinking block; nothing unfolds | The conversation record keeps that a thinking block arrived (`Entry.Thinking`), not its text (found in plan 5, item 4) |

## Assumptions used here

ASM-3 (30 fps), ASM-7 (own configuration file), ASM-8 (prefill flags),
ASM-9 (default keys), ASM-10 (editor budget), ASM-12 (layout geometry), ASM-13
(colours), ASM-14 (Codex child model N/A before 0.163.0), ASM-15 (no send
while a turn runs), ASM-16 (interrupt timeout), ASM-17 (Markdown renderer),
ASM-18 (animation rates), ASM-21 (timeouts), ASM-22 (display defaults); all are
listed in [architecture.md, Assumptions](architecture.md#assumptions).
