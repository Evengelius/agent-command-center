# UI design

Plan 2, 2026-10-16. What the owner sees and does: the startup screen, the
transition, the dashboard's three zones at three terminal sizes, the keys, the
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
`C:\…\agent-command-center`…" with a spinner, zones A and C empty with
"waiting for the session" and every value "N/A (session starting)", A's
model row reading "model N/A (session starting)" and "◉ STARTING". The
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

The working screen, persistent for the session. Zones A to C, under a title
bar and a project bar, with a key-hint line at the bottom; AM2 and AM4 D3
named a zone D too, whose telemetry plan 24 moved into zone A. The
conversation (B) is the primary surface and holds the default focus (AM4
D4).

### Follows and departs

Against [RD, Image 2](reference-design.md#image-2-dashboard-empty-state) and
the [layout mock-up](reference-design.md#layout-mock-up):

| Reference | Here | Why |
| - | - | - |
| Title "AGENT COMMAND CENTER" and "● LIVE" badge | Same; "◆ MOCK — not a live session" replaces "● LIVE" in mock mode | AM4; B3.6 |
| Project bar: folder name, branch, agent CLI on the right; path below | One line at 100+ columns: name, branch, path, then provider and version on the right; the path drops at 80 columns | AM4 W1 |
| A left, full height: orchestrator, model, stage tree with agents indented, a status symbol per line | Same content; since plan 20 A is the upper row's left, beside B, both ending on one row, as the owner's two templates draw it; since plan 24 the telemetry pane is at its bottom, as the owner's image draws it; the tree is labelled with its origin | AM2 A1-A4; brief amendments 16 and 22; [Origins on screen](#origins-on-screen) |
| B upper right: "Streaming" badge, YOU and CLAUDE labelled, an in-progress line, the input at the bottom | Same, with a multiline editor, approval cards inline, and tool lines | AM2 B1-B6 |
| C lower right: one line per agent (model, task, status), an active count, "select agent to inspect" | Since plan 20 the whole width under A and B, two rows an agent from 100 columns (Template 2), adding elapsed time and tokens; since plan 27 the owner's image: the header inside the frame, the agents' panel and the selected agent's execution timeline with its status (brief amendment 26) | AM2 C1-C3, C11; brief amendments 16 and 26 |
| D bottom, full width: two quota bars with percentages, agents, context, tokens | No zone D since plan 24: the mock-up's D is zone A's telemetry pane, which holds the session usage (context, tokens, estimated cost) and the quota windows with their reset countdowns, the rest being in the details view; every missing value "N/A" with its reason | AM2 D1-D9; brief amendment 22 |
| Mock-up figures (42%, "Agents 4/16") | Real values or "N/A" | AGENTS.md; RD "Layout mock-up" |
| Image 2's empty state: "Waiting for instructions", "Ready to work on this project." | Same texts in the ready state | AM4 D2 |
| A one-line status bar (first reference) | A key-hint line; the essentials live in A's telemetry pane | RD "What the owner wants from it" |

Against the owner's two templates (plan 20, brief amendment 16: Template 1,
a completed multi-turn conversation; Template 2, a subagent's Write waiting
for approval), the CLI's real events decide what is shown, the templates
how:

| Template | Here | Why |
| - | - | - |
| YOU and CLAUDE cards outlined by thin borders between rows (Template 1) | A narrow bar `▎` on each row of a message, on no ground; a thin rule between messages, a blank row on each side; RESULT and a pending request keep full borders | The owner's amendment 17 and image: the bar, the header over the body, the rule; a border takes a whole row |
| The prompt beside "YOU" (Template 1), under it with a rule after it (Template 2) | Under its header at every width, the rule between messages | The owner's amendment 17: a body never beside its label |
| "APPROVAL general-purpose Write <path>" on one row (Template 2) | "! APPROVAL · general-purpose · Write", the path on its own row | The owner's section 5.2; a long path would push the time off the row |
| A thin pink outlined "F2 to answer" button | A one-row pill, pink bold on a dark pink ground, half-block ends | An outline needs three rows |
| "allowed by you" on plain Read calls; "f00.go failed"; "+9 more Read calls (f12.go … f21.go)" (Template 1) | The real outcomes, files and counts; "▸ 9 earlier tool events · 1 failed · Expand" (the owner's section 4.6) | Nothing invented: no Read call was approved by the owner, the failed file was f03.go of twelve |
| "▶ subagent" and "▶ tool" kinds, the narration before the Agent call (Template 2) | The real tool's name ("Agent", "Bash") and the agent's identity; the events' order | The real events decide the content |
| Turns 2 to 4 "○ pending" and the agent's tool history in zone A (Template 2) | The real stages; A's content unchanged | Nothing invented; the owner's section 8 keeps A's model |
| A wrench on approved rows, a circled "!" on a failed one (Template 1) | ✓ and ✗ | No single-cell glyph in Windows Terminal's font; the status symbols of [Colours and symbols](#colours-and-symbols) |
| "1 line · Ctrl+G draft window" on the input row, no status row (Template 1) | Since plan 29 the owner's row under a separator (amendment 29): the "+" box, the model pill, "↯ Auto", and in its middle the editor's messages ("turn running: the draft waits", quit confirmations) and the line count past one line | The row is where the messages were; the keys are the footer's and the F1 page's |
| A failed call's path, "failed" at the right (Template 1) | Its path, then its error, "failed" and the time at the right | The error is what needs a look, copyable (plan 5, item 6); it keeps half the row (the owner's live review, 2026-10-20) |

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
  they decide what the agent may do; then one cell to the edge, nothing
  else while approvals are Manual with no change asked (plan 28, the
  owner's image of the top right: "● LIVE" over "Claude Code 2.1.303"
  over zone B's edge with its "Idle"). The approvals mode shows only
  while it is not that: "approvals Auto" in the attention colour while
  Auto is in force (brief amendment 8 keeps it visible), and a change
  asked for and not yet in force after an arrow, "approvals Manual →
  Auto …" while Claude Code is asked, "approvals Manual → Auto (next
  turn)" until the next Codex message, "approvals Auto → Manual …" back;
  the freed cells go to the middle's path. When narrow, Codex's sandbox
  and policy drop first, then the branch; at the narrowest the mode
  alone when there is one, else the name alone (plan 12, plan 28).

## Layouts at three sizes

Three layouts, chosen by the terminal's width: narrow below 100 columns,
standard from 100 to 159, wide from 160; widths and splits below are
ASM-12, as plan 20 set them after the owner's two templates (brief,
amendment 16) and plan 24 moved them for zone A's telemetry (amendment 22).
The layout is recomputed on every resize message; the state does not
change.

From 100 columns A and B share the upper row: the same height, their top
borders on the row under the project bar, their bottom borders on one row.
A takes 28% of the width, rounded, at least 36 columns (36 at 120, 44 at
156, 45 at 160, 56 at 200, 58 at 208), B the rest. C spans the whole width
under them, and the key footer is the last row. The rows A, B and C share
are the height less the two bars and the footer; zone D, which sat under C
until plan 23, is gone, and its rows went to A and B. Under 100 columns A
is one row under the project bar, F6 opening it, its telemetry included,
over B and C, and B and C are full width.

C's height follows the agents in play (plan 18, kept by plan 20;
[events-and-state.md](events-and-state.md#zone-cs-height)): with none, 3
rows, its title and counts in its top border and "no agents yet: they
appear as the CLI starts them"; with agents in play, two rows each from 100
columns and one under 100, with its borders, at most half the rows A, B and
C share (B keeping 12 rows, 14 from 160 columns) and at most 5 rows under
100 columns; from 160 columns, with agents in play, at least 8 rows, for the
selected agent's preview inside C, right of the list. Agents that finish
keep their rows; only the owner's next prompt clears C. A and B take the
rows C leaves.

The frames below are the renderer's own, drawn from the recorded P3b
session in mock mode stopped at the subagent's Write request, Template 2's
state (`testdata\mock\claude-subagent-approval.jsonl`; the golden frames
hold more states).

### Narrow, 80x24

```text
 >_ AGENT COMMAND CENTER                            ◆ MOCK — not a live session 
 ▣ claude-subagent-approval.jsonl                             Mock data 2.1.300 
 A ✓ Turn 1 › ! general-purpose                           (inferred)   F6: tree 
╭─ B · CONVERSATION ──────────────────────────────────────────────────── Idle ─╮
│ ▎ ╭───────────────────────────────────────────────────────────────────────╮  │
│ ▎ │ ! APPROVAL · general-purpose · Write                      11:30  mock │  │
│ ▎ │   <probe-root>\probe-work\p3b\found.txt                               │  │
│ ▎ │ ▸ Preview of found.txt (2 lines)                                      │  │
│ ▎ │  1 │ alpha.txt                                                        │  │
│ ▎ │  2 │ beta.md                                                          │  │
│ ▎ │ This will create or overwrite the file at                             │  │
│ ▎ │ `<probe-root>\probe-work\p3b\found.txt` with the above content.       │  │
│ ▎ │ ▐ F2 to answer ▌                                                      │  │
│ ▎ ╰───────────────────────────────────────────────────────────────────────╯  │
│                                                                              │
│ ──────────────────────────────────────────────────────────────────────────── │
│ >  Ask Mock data anything…                                                   │
│ ──────────────────────────────────────────────────────────────────────────── │
│   +     Haiku 5.5 High · Ultracode                                   ↯ Auto  │
╰──────────────────────────────────────────────────────────────────────────────╯
╭─ C · AGENTS ───────────────────────────────────────────── 1 ACTIVE · 0 DONE ─╮
│  ● general-purpose haiku-5-5   waiting: approval              6s   21.2k tok │
╰──────────────────────────────────────────────────────────────────────────────╯
 Enter send · Ctrl+J newline · F6 zone · F2 approve · F3 Explorer · F1 keys     
```

- A is one row: the current path through the stage tree, a status symbol
  per step. Focusing A (F6) opens the whole zone over B and C, the tree and
  its telemetry pane, which takes its tight form here (one row a window, no
  bar); Esc closes it.
- B keeps every function at reduced size: each message under its header,
  a thin rule between messages, the tool rows, a pending request's box
  with its preview's rows straight in it (under 90 cells), the editor (one
  row, growing with the draft to half the screen, then scrolling). Here the
  box fills B; the narration above it is a scroll away.
- C is one row an agent, at most 5 rows; more are reached by scrolling or
  zoom.

### Standard, 120x40

```text
 >_ AGENT COMMAND CENTER                                                                    ◆ MOCK — not a live session 
 ▣ claude-subagent-approval.jsonl   ..\..\testdata\mock\claude-subagent-approval.jsonl                Mock data 2.1.300 
╭──────────────────────────────────╮╭─ B · CONVERSATION ──────────────────────────────────────────────────────── Idle ─╮
│ A · ORCHESTRATOR                 ││ ▎ CLAUDE  Haiku 5.5  11:30                                                       │
│ ✿ Haiku 5.5      ! AGENT WAITING ││ ▎  ✓ Agent  general-purpose · List directory and write fo…  11:30:06      mock   │
│ ──────────────────────────────── ││ ▎                                                                                │
│ STAGES · inferred from activity  ││ ▎ I've launched the subagent in the background to list the directory and write   │
│ ✓ TURN 1                      4s ││ ▎ `found.txt`. I'll report its result when it finishes.                          │
│   Use the Agent tool to launch … ││ ▎                                                                                │
│   Agent · agent general-purpose  ││ ▎  ✓ general-purpose · Bash  ls -p "<probe…  allowed by you 11:30:08      mock   │
│                                  ││ ▎                                                                                │
│                                  ││ ▎ ╭───────────────────────────────────────────────────────────────────────────╮  │
│                                  ││ ▎ │ ! APPROVAL · general-purpose · Write                          11:30  mock │  │
│ ──────────────────────────────── ││ ▎ │   <probe-root>\probe-work\p3b\found.txt                                   │  │
│ ✦ TELEMETRY        Enter details ││ ▎ │ ▸ Preview of found.txt (2 lines)                                          │  │
│ ▾ SESSION USAGE                  ││ ▎ │  1 │ alpha.txt                                                            │  │
│   Context 3% of 1M ░░░░░░░░░░░░░ ││ ▎ │  2 │ beta.md                                                              │  │
│   Tokens 61.9k session total     ││ ▎ │ This will create or overwrite the file at                                 │  │
│   Est. cost $0.00 API-equiv.     ││ ▎ │ `<probe-root>\probe-work\p3b\found.txt` with the above content.           │  │
│ ▾ QUOTA WINDOWS                  ││ ▎ │ ▐ F2 to answer ▌                                                          │  │
│   Session (5hr)             35%† ││ ▎ ╰───────────────────────────────────────────────────────────────────────────╯  │
│   ███████████░░░░░░░░░░░░░░░░░░░ ││                                                                                  │
│   Resets in 14h 39m              ││ ──────────────────────────────────────────────────────────────────────────────── │
│   Weekly (7 day)            47%† ││ >  Ask Mock data anything…                                                       │
│   ██████████████░░░░░░░░░░░░░░░░ ││ ──────────────────────────────────────────────────────────────────────────────── │
│   Resets in 1d 20h               ││   +     Haiku 5.5 High · Ultracode                                       ↯ Auto  │
╰──────────────────────────────────╯╰──────────────────────────────────────────────────────────────────────────────────╯
╭──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╮
│ C · LIVE AGENTS                                                                                    1 ACTIVE · 0 DONE │
│ ──────────────────────────────────────────────────────────────────────────────────────────────────────────────────── │
│ AGENTS                                   │ EXECUTION TIMELINE                                              ● WAITING │
│  ●  general-purpose     │  haiku-5-5     │ no activity yet                                                           │
│     00:06   21.2k tok   │  tool: Write   │                                                                           │
│                                          │                                                                           │
│                                          │                                                                           │
│                                          │                                                                           │
│                                          │                                                                           │
│                                          │                                                                           │
╰──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╯
 Enter send · Ctrl+C interrupt · Ctrl+G draft · F6 zone · F2 approvals · F3 Explorer · F1 keys · Ctrl+Q quit            
```

- A, 36 columns here (the least; 28% of 120 is 34), and B end on the same
  row, 33 rows from the row under the project bar: the height less the two
  bars, C's four rows and the footer. A's frame carries no title; inside
  it, "A · ORCHESTRATOR" alone on its row, then the model row, the model at
  the left and the orchestrator's status at the right end of that same row
  ("! AGENT WAITING": the turn completed and the subagent waits for the
  owner; the title "Claude Haiku 5.5" does not fit beside it, so the row
  takes the short name), a rule, "STAGES · inferred from activity" and a
  block a turn, here TURN 1 folded, being closed: its prompt's first line,
  cut with "…", and one grey row naming its items.
- Under the tree's free rows, a rule and the telemetry pane in its compact
  form, since zone C's 12 rows leave A 25 (plan 27): "✦ TELEMETRY" with
  "Enter details" at its right end; "▾ SESSION USAGE", three rows at this
  width (the context with its bar, the tokens, the cost); "▾ QUOTA
  WINDOWS", three rows a window (its percentage at the right end, its bar,
  "Resets in …") with no blank row between them, and nothing under the
  last window: the pane carries no notes (plan 26, the owner's standing
  decision).
- B is the conversation ([Zone B](#zone-b)): each message a block, a
  narrow bar of its colour, its header on its own row and its body under it,
  a thin rule between messages (amendment 17); the input sits at B's
  bottom edge under its rule.
- C is full width and 12 rows (plan 27): inside its frame the header "C ·
  LIVE AGENTS" with "1 ACTIVE · 0 DONE", a rule, then "AGENTS" over the
  cards' panel (40 cells, as wide as the card) and "EXECUTION TIMELINE"
  over the timeline with the agent's status "● WAITING" at its right end;
  the agent's card, " ●  general-purpose     │  haiku-5-5" over "00:06
  21.2k tok   │  tool: Write", left of the panels' line, "no activity yet"
  right of it (the mock's subagent has sent no message yet).
- The footer names F3 at every size: "F3 Explorer", the Explorer F3 opens
  when no file is in front ("F3 file" while B's entry cursor is on an entry
  that names one). When the whole line does not fit, as at 120 columns, it
  leaves out "Ctrl+J newline", which B's input shows above it, so that
  "Ctrl+Q quit" stays visible (from 108 columns here, from 173 in the wide
  layout) (plan 17).

### Wide, 200x60

```text
 >_ AGENT COMMAND CENTER     ■ claude-haiku-5-5                                                                                                                             ◆ MOCK — not a live session 
 ▣ claude-subagent-approval.jsonl   ..\..\testdata\mock\claude-subagent-approval.jsonl                                                                                                Mock data 2.1.300 
╭──────────────────────────────────────────────────────╮╭─ B · CONVERSATION ──────────────────────────────────────────────────────────────────────────────────────────────────────────────────── Idle ─╮
│ A · ORCHESTRATOR                                     ││                                                                                                                                              │
│ ✿ Claude Haiku 5.5                   ! AGENT WAITING ││ ▎ YOU  11:30                                                                                                                                 │
│ ──────────────────────────────────────────────────── ││ ▎ Use the Agent tool to launch one general-purpose subagent with this task: list the files in the current directory, write their names into  │
│ STAGES · inferred from activity                      ││ ▎ a new file named found.txt with the Write tool, and report the names. Then tell me in one line what it found.                              │
│ ✓ TURN 1                                          4s ││                                                                                                                                              │
│   Use the Agent tool to launch one general-purpose … ││ ──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────── │
│   Agent · agent general-purpose                      ││                                                                                                                                              │
│                                                      ││ ▎ CLAUDE  Haiku 5.5  11:30                                                                                                                   │
│                                                      ││ ▎  ✓ Agent  general-purpose · List directory and write found.txt                                                        11:30:06      mock   │
│                                                      ││ ▎                                                                                                                                            │
│                                                      ││ ▎ I've launched the subagent in the background to list the directory and write `found.txt`. I'll report its result when it finishes.         │
│                                                      ││ ▎                                                                                                                                            │
│                                                      ││ ▎  ✓ general-purpose · Bash  ls -p "<probe-root>/probe-work/p3b" | grep -v '/$'                          allowed by you 11:30:08      mock   │
│                                                      ││ ▎                                                                                                                                            │
│                                                      ││ ▎ ╭───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╮  │
│                                                      ││ ▎ │ ! APPROVAL · general-purpose · Write                                                                                      11:30  mock │  │
│                                                      ││ ▎ │   <probe-root>\probe-work\p3b\found.txt                                                                                               │  │
│                                                      ││ ▎ │ ╭───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╮ │  │
│                                                      ││ ▎ │ │ ▸ Preview of found.txt (2 lines)                                                                                                  │ │  │
│                                                      ││ ▎ │ │  1 │ alpha.txt                                                                                                                    │ │  │
│                                                      ││ ▎ │ │  2 │ beta.md                                                                                                                      │ │  │
│                                                      ││ ▎ │ ╰───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╯ │  │
│                                                      ││ ▎ │ This will create or overwrite the file at `<probe-root>\probe-work\p3b\found.txt` with the above content.                             │  │
│                                                      ││ ▎ │ ▐ F2 to answer ▌                                                                                                                      │  │
│ ──────────────────────────────────────────────────── ││ ▎ ╰───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╯  │
│ ✦ TELEMETRY                            Enter details ││                                                                                                                                              │
│ ▾ SESSION USAGE                                      ││                                                                                                                                              │
│   CONTEXT           │ TOKENS        │ EST. COST      ││                                                                                                                                              │
│   3% of 1M          │ 61.9k         │ $0.00          ││                                                                                                                                              │
│   █░░░░░░░░░░░░░░░░ │ session total │ API-equivalent ││                                                                                                                                              │
│                                                      ││                                                                                                                                              │
│ ▾ QUOTA WINDOWS                                      ││                                                                                                                                              │
│   Session (5hr)                                 35%† ││                                                                                                                                              │
│   ██████████████████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ ││                                                                                                                                              │
│   Resets in 14h 39m                                  ││                                                                                                                                              │
│                                                      ││ ──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────── │
│   Weekly (7 day)                                47%† ││ >  Ask Mock data anything…                                                                                                                   │
│   ████████████████████████░░░░░░░░░░░░░░░░░░░░░░░░░░ ││ ──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────── │
│   Resets in 1d 20h                                   ││   +     Haiku 5.5 High · Ultracode                                                                                                   ↯ Auto  │
╰──────────────────────────────────────────────────────╯╰──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╯
╭──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╮
│ C · LIVE AGENTS                                                                                                                                                                    1 ACTIVE · 0 DONE │
│ ──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────── │
│ AGENTS                                   │ EXECUTION TIMELINE                                                                                                                              ● WAITING │
│  ●  general-purpose     │  haiku-5-5     │ no activity yet                                                                                                                                           │
│     00:06   21.2k tok   │  tool: Write   │                                                                                                                                                           │
│                                          │                                                                                                                                                           │
│                                          │                                                                                                                                                           │
│                                          │                                                                                                                                                           │
│                                          │                                                                                                                                                           │
│                                          │                                                                                                                                                           │
│                                          │                                                                                                                                                           │
│                                          │                                                                                                                                                           │
│                                          │                                                                                                                                                           │
│                                          │                                                                                                                                                           │
╰──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╯
 Enter send · Ctrl+J newline · Ctrl+C copy or interrupt · Ctrl+G draft · F6 / Shift+F6 zone · F2 approvals · F3 Explorer · F4 Manual/Auto · z zoom · F12 mouse capture · F1 keys · Ctrl+Q quit          
```

- A grows to 56 columns (28% of 200), and with B to 49 rows (60 less the
  two bars, C's 8 rows and the footer): the model row has room for the
  whole title beside the status ("✿ Claude Haiku 5.5"), the usage section
  draws its three columns (CONTEXT, TOKENS and EST. COST, "API-equivalent"
  in full) and each quota window's bar runs across the pane.
- C is 16 rows (plan 27): its header, a rule, "AGENTS" over the cards'
  panel (40 cells here too: the card, not the width, sizes it) and
  "EXECUTION TIMELINE" with "● WAITING" over the timeline, 153 cells, so
  several agents can be watched while the orchestrator talks (AM2 C11).

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
                 │                ├──► A telemetry: quota, tokens, context │
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

The orchestrator, its workflow tree and, since plan 24, the telemetry (AM2
A1-A6 and D1-D9; AM4 D3; B2.2, B2.4; brief amendment 22, the owner's image
in [reference-design.md](reference-design.md#zone-a-with-the-telemetry)).
Plan 24 removed zone D: its values are the pane at A's bottom, and the
details view opens from it ([Zone A: telemetry](#zone-a-telemetry)). From
the top, inside a frame whose top edge carries no title:

- **Heading**: "A · ORCHESTRATOR" in the title colour, alone on its row.
- **Model row**: an icon in the model's colour, "✿" for a Claude model and
  "◆" for any other, then the model's title: `Claude <Family> <major>.<minor>`
  for a Claude id of the families Opus, Sonnet, Haiku, Fable and Mythos
  (`claude-haiku-5-5` reads "Claude Haiku 5.5", `claude-fable-5-1` "Claude
  Fable 5.1"), `GPT-` and
  the rest for an id starting with `gpt-` (`gpt-5.6-sol` reads
  "GPT-5.6-sol"), any other id as the CLI gives it. Before `session.started`
  it reads "model N/A (<reason>)", as "model N/A (session starting)". The
  status ends this row, never the heading's. Where the title and the status
  do not both fit, the row takes the short name of zone B's labels
  ("✿ Haiku 5.5"), then cuts the title with "…".
- **Status**, from the state, never from text. The first row of this table
  that applies: the session's phase comes before the turn's, the turn's
  before the agents', the agents' before the turn's outcome (⠋ stands for
  the turning spinner):

  | Status | When | Colour |
  | - | - | - |
  | ◉ STARTING, ◉ CLOSING | the session is starting or closing | amber |
  | ✗ ENDED | the session crashed | red |
  | ○ CLOSED | the session is closed | grey |
  | ! WAITING | an approval or a question waits for the owner | pink |
  | ◉ INTERRUPTING | the owner interrupted and the turn has not ended | amber |
  | ⠋ THINKING | a thinking phase is live (plan 22) | title colour |
  | ⠋ RUNNING | the turn works otherwise | amber |
  | ! AGENT WAITING | the turn is idle and an agent waits for the owner | pink |
  | ◉ AGENT RUNNING, ◉ 2 AGENTS RUNNING | the turn is idle and one agent, or two, still run | amber |
  | ✓ COMPLETED | the turn completed | green |
  | ✗ FAILED | the turn failed | red |
  | ○ INTERRUPTED | the turn was interrupted | amber |
  | ○ IDLE | no turn yet | grey |

  So a completed turn with a background agent still working never reads
  COMPLETED (brief amendment 22, section 2).
- **Stages**, under a rule, headed by their origin:
  - Codex with a reported plan: "PLAN · observed (turn/plan/updated)", the
    plan's steps with their status (events-and-state.md, Agent tree);
  - otherwise, and always for Claude Code: "STAGES · inferred from
    activity", "STAGES" in white and the rest grey, with no chevron; one
    stage per user turn, its agents and tool groups as children, each
    stage's status derived from its children (CC-10 q7).
- **A turn's block** (the owner's image):
  - its head row: the status symbol, "TURN n" in white, then "· failed",
    "· interrupted", "· running" or "· waiting for approval" when that is
    its word (a completed turn says nothing more), and its elapsed time at
    the row's right end;
  - unfolded, under it: the prompt's first line (or "started by the CLI")
    in grey, then tools (`✓ Read src/auth/*.ts`), agents (`◉ agent
    scan-routes`) and, under each agent, its current tool or pending
    approval, with `├─ └─ │` connectors. A path too long for the row keeps
    its file name, cut at its left end, `✓ Read  …\f03.go` (the owner's
    live review, 2026-10-20: twelve Read rows read "C:\User…" alike). While
    the turn runs, the connector of the running branch carries the moving
    dot and a last grey line reads "○ (more as events arrive)": the tree
    never pretends to know future steps;
  - folded, the image's block: the same first line, then one grey row of
    the turn's items in order, a tool by its name, an agent as "agent
    <label>", each name once, joined by " · " ("Agent · Bash · Write"), cut
    at the width;
  - a closed turn folds by default and an open one is unfolded; → and ←
    change it, and the owner's choice stays (plan 4). A blank row separates
    two blocks, and the cursor row's background covers every row of a
    folded turn's block.
- **The telemetry pane**, fixed at A's bottom and never scrolled, after a
  rule: "✦ TELEMETRY" in the title colour with "Enter details" in grey at
  its right end; "▾ SESSION USAGE" and its rows; a blank row; "▾ QUOTA
  WINDOWS" and its rows, and nothing after them. It takes the rows its form
  needs and the tree the rest, at least 6. Three forms, the largest that
  leaves the tree its 6 rows:
  - *full*, the image's: the usage section's three rows, three rows a quota
    window with a blank row between windows;
  - *compact*: no blank row;
  - *tight*: the usage on two rows ("Context 6% of 258.4k" and its bar, then
    "Tokens 184.3k · Cost N/A"), a window on one row with its percentage and
    its countdown and no bar ("Session (5hr)  1% · 15h 13m").

  On a terminal too short for the tight form beside 6 tree rows, the tight
  form is drawn all the same and the tree keeps what is left, 3 rows at the
  least, the pane cut beyond that. Each section folds with ← → on its
  heading ("▸ SESSION USAGE") and the pane shrinks, the tree taking its
  rows.
- **Session usage**: from 41 inner cells (A 45 columns wide, from 159
  columns), three columns with thin rules between them: "CONTEXT", "TOKENS"
  and "EST. COST" in grey capitals, the values in white, the support in grey
  under them. The support is the context's bar (the used share of its
  window, a lavender fill on its track), "session total" under the tokens
  and "API-equivalent" under the cost ("API-equiv." under 45 cells). Under
  41 cells, three rows: "Context 3% of 1M" and its bar, "Tokens 61.9k
  session total", "Est. cost $0.00 API-equivalent". A value no source gave
  reads "N/A" with its reason as its support ("no result yet", "Codex
  reports none"), never 0; "†" follows an undocumented value; a derived
  value carries no mark here, "(derived)" follows it in the details view.
- **Quota windows**, a block a window, in this order: "Session (5hr)"
  (`quota.5h`), "Weekly (7 day)" (`quota.7d`), each model-scoped weekly
  window the CLI reports ("Fable limit" in the image: `quota.7d.<model>`,
  from Claude Code's `unifiedWindows` key or its `rateLimitType` of the form
  `seven_day_<model>`, as `seven_day_opus` is documented), then any other
  Codex window by its duration ("90 min window"). A model-scoped window
  shows only once reported, never assumed from the model or the plan. A
  block:
  - the label, and at the row's right end the percentage ("17%", "17%†"
    when the field is undocumented);
  - the bar, "█" on "░" across the pane, its fill round(p × width) cells,
    clamped to the width, lavender, red from 90%;
  - "Resets in <countdown>" in grey from the reported reset time: "5d 3h"
    ("5d" when no hours are left), "4h 12m" ("4h" when no minutes are left),
    "12m", "<1m"; "Reset time unavailable" when the CLI gave none.

  The countdown is drawn from the frame's time, so it changes as the clock
  ticks and never goes negative: at the reset the clock tick (plan 4) turns
  the value N/A and the block reads "N/A" with "window reset; waiting for
  the next update". A window the CLI reported N/A (an API key, a plan with
  no 5-hour limit, a read that failed) reads "N/A" with its reason on the
  second row and no bar; a status word with no percentage reads the word and
  "no percentage reported". A new report replaces the value and the bar's
  fill.
- **No notes** (plan 26, brief amendment 25, the owner's standing
  decision): the pane ends with the last window's rows. The notes it drew
  until plan 25 under a rule ("errors N", "retries N", "unrecognised N",
  "quota updated 14s ago", "context derived from usage", "tokens summed over
  threads", "† undocumented field", "cost at API rates", "overage in use",
  "plan <plan>", "credits <credits>", or "values from the CLI's events") are
  gone for good; the details view holds every one of those facts (the
  errors, the unrecognised lines, each value with its source and age,
  "(derived)" after a derived value, the counters, the "†" legend in its
  TELEMETRY heading). The way there, "Enter details", is at the right end
  of the TELEMETRY heading.
- **Keys** when focused: ↑ ↓ move the cursor through the turns, then
  TELEMETRY, SESSION USAGE and QUOTA WINDOWS, and back; ← → fold and unfold
  a turn, SESSION USAGE or QUOTA WINDOWS; Enter inspects the agent or tool
  under the cursor and, on one of the three headings, opens the details
  view; z zooms A alone under the bars, its telemetry in the full form
  where the rows allow it. A click on a tree node moves the cursor to it, a
  click on the TELEMETRY row opens the details view, a click on a section's
  heading folds or unfolds it. A has no key hint row, as C has none since
  plan 20: while it has the focus the footer reads "↑↓ move · ←→ fold ·
  Enter inspect or details · z zoom · PgUp PgDn Home End scroll · F6 zone ·
  Esc back to the input · F1 keys".
- **Scrolling**: the tree scrolls, resting at its tail, and the pane under
  it does not. PgUp, PgDn and the wheel scroll the tree, Home shows its
  first row and End its tail (plan 5, item 5); the cursor's node stays
  visible. A tree held away from its tail says so in A's top edge, in amber,
  with the rows out of view and the way back, "↑ 12 ↓ 25 · End follows": see
  [Scrolling, collapsing, full screen](#scrolling-collapsing-full-screen).

## Zone B

The real conversation with the orchestrator (AM2 B1-B11; AM4 D3). Every
entry comes from the conversation record, which is built only from CLI
events (events-and-state.md, Conversation record); nothing is simulated.

- **Header**: "B · CONVERSATION" and a badge: "⠋ Streaming" while text
  arrives, "⠋ Thinking" while a Claude Code thinking phase is live (plan
  22), "⠋ Running" while the turn runs otherwise (tools, subagents, a
  Codex model step), the spinner turning in each (plan 4, step 8); "!
  Waiting for approval", "Interrupting…", and "Idle" when the session is
  ready with no turn.
- **The visual language** (plan 20; brief amendments 16 and 17, the
  owner's templates and image): each message is a block. An *exchange* is
  a prompt and all that follows it up to the next prompt, the turns the CLI
  opens by itself included; it draws as the owner's message, then the CLI's
  under its header, then a RESULT box when it has a result. A block is a
  narrow bar `▎` of its colour on each of its rows, its header on its own
  row ("YOU  14:29", "CLAUDE  Opus 5.5  14:29"), and its body under the
  header from the same column, on the zone's own dark background: no tinted
  band behind a message, a ground only behind tool rows, a fold's row and a
  request's preview. A thin rule in the frames' colour (`#3B3F51`), across
  B's inner width, separates two messages, a blank row on each side: before
  each of the owner's messages but the first, and between the owner's
  message and the CLI's part. Never inside a message: between narration, a
  group of tool rows and a request waiting in its box, a row of the block,
  its bar (the owner's section 7.3); before the RESULT box, a blank row. A
  blank row opens the conversation under B's border and one ends it above
  the input's rule, which is the last message's: no rule of its own there,
  none at the start, never two together; scrolled, a rule at the window's
  edge, the message beyond it out of view, is a blank row, so it never
  doubles B's border or the input's rule (the owner's live review). The
  same layout at every width.

  ```text
  ▎ YOU  14:29                                       violet bar and label, the time muted
  ▎ Read the twelve files src\f00.go to src\f11.go…   the prompt under its header

  ──────────────────────────────────────────────     the rule between two messages

  ▎ CLAUDE  Opus 5.5  14:29                          blue, once per exchange; the model muted
  ▎ I'll read the twelve files one by one.           narration under the header
  ▎
  ▎  ▸ 9 earlier tool events · 1 failed · Expand     tool rows on their ground
  ▎  ✓ Read  …\src\f09.go              12:15:23  observed

  ╭────────────────────────────────────────────╮
  │ RESULT  12:15                              │   green box, the answer under its label
  │ I read 11 of the 12 files.                 │
  ╰────────────────────────────────────────────╯
  ```
- **Entries**:
  - **YOU** with the time, violet (`#A78BFA`), on its own row; under it
    the message as sent, byte for byte, in a brighter text colour
    (`#F9FAFB`; a long paste is shown folded, below); then one row an
    image the message carried as image input (plan 31), "🗀  name · 12
    KB", muted, its file's name from the attachments, else its media type
    as the line written gives it;
  - **CLAUDE** or **CODEX** with the model and the time, the label blue
    (`#60A5FA`), the model and the time muted, once per exchange, above its
    first message or tool call: under it the AI text, streamed, rendered as
    Markdown; the exchange's later messages carry no label. The model reads
    `<Family> <major>.<minor>` for a `claude-<family>-<major>-<minor>` id,
    dated or not (`Haiku 5.5`, as the owner drew it; the families Opus,
    Sonnet, Haiku, Fable and Mythos since plan 24), any other id as the
    CLI gives it, none while the CLI has given none; zone C keeps the id, as
    the details view and the wide title bar do, and zone A's model row has
    its own title ([Zone A](#zone-a));
  - **RESULT**, green (`#4ADE80`), a rounded box, "RESULT  01:54" on its
    own row and the answer under it, holds the exchange's last message once
    the exchange is over (no turn running), its last turn completed, not
    failed or interrupted, every agent started in the exchange completed,
    and a tool call, an approval, a review or another message came before
    it. An agent still running, failed or stopped leaves the message in the
    CLI's block ("I've launched the subagent…" is never a result, the
    owner's section 6.1), and so does a lone answer. When the CLI opens a
    turn of its own after the owner's, for a background agent's report,
    RESULT moves to that turn's last message once it completes. It is
    decided from the turn machine, the agents' machines and the turn each
    entry belongs to (events-and-state.md, Conversation record), never from
    the text;
  - tool rows: one per call, inside the CLI's card, from column 2 on the
    tool ground (`#1E2133`, lighter than the card's; none on a 16-colour
    terminal): the status symbol (✓ done, ✗ failed or denied, the spinner
    while it runs in a running turn, ◉ otherwise, ! waiting for the owner,
    ○ unfinished); on a subagent's request, who asked ("general-purpose ·
    "); the tool's name in light blue (`#93C5FD`), bold; its target in a
    readable muted text (`#A1A9C0`): a path, its left end elided at one of
    its elements (`…\Local\Temp\demo\src\f00.go`, never cut inside one), a
    command's first line, a pattern, a URL, for an agent its type in blue
    and its description, the targets of one group in one column. At the
    right, how it ended and when, then the origin: "allowed by you
    01:54:35" in green only for an answered approval, "denied by you …" in
    red, "failed 01:54:59" in red, "approved by auto-review" in green or
    "denied by auto-review" in red when the CLI's reviewer decided (shown
    while the call runs too), "running 4s" in amber, "waiting for you" in
    pink, a completed call's end time in grey; then `observed` or
    `mock`. Running a tool is never shown as the owner's approval. A
    failed call names what failed, as Template 1's row does, then its
    error, its output's first non-blank line in red (plan 5, item 6):
    `✗ Read  …\src\f03.go · File does not exist.`, or `✗ Bash · exit
    status 1: FAIL pkg/a` without a target. The error keeps half the room,
    or its whole width when less, the target the rest, and both are cut to
    the room the outcome leaves (plan 13, item 1: the verdict stays; the
    owner's live review of 2026-10-20). A call that failed with no output
    reads "failed" in its outcome column, one denied at its approval
    "denied". The name and the error are content (the target and the
    symbols are decoration): a selection copies the name and the error as
    two lines;
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
  - an agent's launch is its Agent call's row, `✓ Agent  general-purpose ·
    <description>` (the agent's own text and calls stay in C and in
    inspection: subagent text is not streamed by Claude Code,
    events-and-state.md, `conversation.message.delta`); only the
    orchestrator's calls are B's;
  - a thinking phase: its live card while it runs, then its summary row
    among the tool rows ([Thinking](#thinking), plan 22);
  - approval cards ([Approvals](#approvals)): a pending one in its pink
    box, Template 2's; an answered one is one row among the tool rows, on
    their ground, who asked kept, its outcome in its colour;
  - what the code shows around a turn (plan 21, item 5.6c): a failed
    turn, the error as received, "✗ turn failed: " and its first line, in
    red (`internal/render/cards.go`; the details view lists it whole); an
    interrupted turn, no entry of the dashboard's: the CLI's own note, such
    as Claude Code's "[Request interrupted by user]", in amber with "· "
    (observed), zone A's stage "○ interrupted" (`internal/render/tree.go`),
    its open tools "unfinished"; a refused send, no entry either: the
    editor's row reads "not sent: the session is <phase>; the draft
    is kept", or "turn running: wait, or interrupt" while a turn runs
    (`internal/state/actions.go`). The one note of the dashboard's own in B
    is the Codex slash command held back, below, in grey with "· ".
  - **Slash commands on Codex**: Codex runs `/status`, `/model` and the rest
    in its own terminal only; through `app-server` the text would reach the
    model as a message (plan 4, L5). A message whose first word is a slash
    command (`/` then lowercase letters and `-`; a path such as `/tmp/x`
    has a second `/` and is not one) is held back once, with a grey note
    of the dashboard's (role `dashboard`, every row under the text):
    "· /status is a command of Codex's own terminal; it does not work
    here, so nothing was sent." (for /status, the row goes on to point at
    the 5h and weekly limits, which zone A's quota windows show) and
    "Press Enter again to send it to the model as a message.". The draft
    stays; the same text sent again goes to the model, another text is
    checked anew. Claude Code runs its slash commands through the stream,
    so nothing is held there.
- **Long pasted messages**: a sent message over 20 lines shows its first 5 (ASM-22)
  and "… 9,995 more lines" (the CLIs fold large pastes the
  same way, TI 3.4); the record keeps the full text.
- **Runs of tool calls** (plan 19; the owner's section 4.6): consecutive
  calls, with the thinking marks between them, form a run. A run of 5 calls
  or more shows its last 3 and, where the others were, one row on the
  ground: `▸ 9 earlier tool events · 1 failed · Expand`, the `▸` blue,
  the count in the muted text, then how many of them failed (red), run
  (amber) or wait for the owner (pink). Enter on it, or a click, expands
  the run under the row `▾ 9 earlier tool events · Fold`; ← on one of its
  calls, or a click on that row, folds it again; the owner's choice holds
  for the session. Messages, approvals, reviews, notices and errors end a
  run and are never folded, so a pending approval is never hidden, and the
  order of events is kept. Only the orchestrator's calls are in B, so a
  fold never hides which agent acted. A selection copies the conversation
  as before, the folded calls' names included.
- **Expanding** (plan 5, item 4): B has an entry cursor. ↑ from an empty
  input puts it on the newest entry B shows in part: a folded message, a
  tool line (one row, its output not shown), a failure (one row, cut to the
  width), a folded run's row (plan 19), never a call the run folds. Its row
  gets `▸` in the attention colour in its card's margin, right of the
  bar and "Enter full view · ↑↓ entries · Esc back" on the right
  ("Enter expands" on a run's row, "← fold" added on a call of an expanded
  run), without changing B's height; the bottom hints say the same. ↑ ↓
  move between such entries, B scrolling to keep the marked row in view; ↓
  past the newest, Esc or a typed character return to the input (the
  character is typed there). With a draft, ↑ moves in the draft as before.
  A mouse click is not used, except on a run's row (plan 19): a click on a
  row starts a selection. Enter shows the entry
  whole, full screen (→ only expands a folded run, as it unfolds in zone
  A: it opens nothing; the owner's live review, 2026-10-20), as the approval's full view does: a message with
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

### Thinking

The live thinking indicator (plan 22, brief amendment 18, the owner's three
images): each thinking phase of the orchestrator is an entry of the
conversation record (events-and-state.md, Conversation record), drawn in
the CLI's block of its exchange, in the violet of thinking. What each CLI
reports decides what is shown (claude-code.md and codex-cli.md, section
12); nothing is estimated by the dashboard and no number is invented.

- **The live card**, while a phase runs: a nested box in the thinking
  violet (`#7C6BB3`), never a band behind the message, a blank row of the
  block between it and the speaker's label above it (the owner's drawing),
  a row of the block between it and what follows:

  ```text
  ▎ ╭──────────────────────────────────────────────────────────╮
  ▎ │ ◝ Pondering...                                     00:03 │   spinner, expression, elapsed
  ▎ │                                                          │
  ▎ │ ≈ 650 thinking tokens · live estimate                    │   the count, or why there is none
  ▎ │                                                          │
  ▎ │ ━━━━━━━━━━━━━━━━━━━╸━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━  2k │   the bar and its scale
  ▎ ╰──────────────────────────────────────────────────────────╯
  ```

  Seven rows when zone B's conversation has 20 rows or more, five (no
  blank rows) from 12, two without a border below (80x24: the spinner's row
  with the count after the expression, then the bar); the size holds while
  the phase lasts unless zone B's changes. The spinner turns through ◜ ◠ ◝
  ◞ ◡ ◟, 100 ms each, a still ◌ while a request waits for the owner. The
  expression is one of the owner's ten, word for word ("Crunching...",
  "Vibing...", "Connecting dots...", "Smoothing...", "Pondering...",
  "Untangling...", "Brewing ideas...", "Cooking...", "Piecing it
  together...", "Making magic..."), a new one every 4 s, never twice in a
  row, dimmed for its first 200 ms: decoration, never a claim about what
  the model does. The time is the phase's, mm:ss, h:mm:ss past an hour.
  The 2-row card puts the count after the expression in a short form when
  the long one would not fit ("≈ 650 tokens · estimate", "Model working ·
  tokens after the step"), measured against the row it is drawn in, its
  exit's longer one included, and leaves the count out of the exit's row
  when neither form fits beside its head, never cut (plan 22, correction
  C7).
- **The count and the bar**, by what the CLI reports:
  - **A live count, no budget (mode B): Claude Code.** The CLI's running
    estimate (`system/thinking_tokens`), "≈ 650 thinking tokens · live
    estimate", the number bright and lit for 400 ms when it changes, never
    an in-between value. The bar's scale starts at 2,000 tokens and
    doubles as the estimate passes it (2k, 4k, 8k…), its label at the
    bar's right end lit for a second when it changes, so a shorter fill
    under a larger scale never reads as fewer tokens; the fill is the
    estimate over the scale, to half a cell (`╸`). After each new value the
    fill grows over 300 ms (ease out), its leading cell lit, settling over
    400 ms more; between values a shimmer crosses the filled part, the fill
    unchanged. The bar measures tokens, never the task's progress: it never
    fills because a phase ended.
  - **A budget (mode A):** "Thinking · 1,284 / 4,096 tokens", the fill
    proportional. Neither CLI reports a budget: tested, never shown.
  - **No live count (mode C).** Before Claude Code's first estimate (0.15
    to 0.4 s into a block, none for a very short one): "Thinking · token
    usage pending". A Codex model step: "Model working · reasoning tokens
    reported when the step ends", since Codex reports a step's reasoning
    tokens only once the step ends. The bar is indeterminate, a segment
    sweeping the track every 1.6 s, with no number.
- **Codex's card is the model step's.** It shows from the step's start,
  inferred (the turn's message recorded, then the previous step's response
  complete and its tools finished), to its first output: a reasoning item
  turns it into the phase's summary, "Thought for up to 3.1s", the step's
  time up to the reasoning's end, an upper bound, its count row reading
  "Thinking" as it closes; any other first output collapses it away over
  300 ms, leaving no row (28 of 76 recorded steps had no reasoning). The
  dashboard asks Codex for reasoning summaries (`summary: "auto"` on every
  `turn/start`; plan 22, amendment 20), so a step's reasoning item starts
  while the model works. The card then reads "Thinking · reasoning tokens
  reported when the step ends" from that start until the item completes
  (2.5 and 3.4 s in P22s), and "Model working" before it.
- **Entrance** (280 ms, a phase drawn live): the rows reserved from the
  first frame, so nothing below moves twice; the border dim with a light
  sweeping its top edge, then the spinner's row, the count's and the
  track, each from dim to full, the track drawn from the left.
- **Exit** (300 ms): for 100 ms, ✓ in place of the spinner (✗ red for a
  phase interrupted or ended by a failure) and the duration in place of the
  expression, lit, the bar frozen, Claude Code's count row giving the exact
  count once it came ("2,852 thinking tokens", "… over 2 phases" for a
  message's count), else the estimate without "live" ("≈ 2,700 thinking
  tokens · estimate"), or "Thinking · no exact count reported" when none
  can come any more (plan 22, correction C8);
  for the next 100 ms the card closes to
  that row in its border, the count's, the bar's and the blank rows gone;
  then, from 200 ms, the summary row in its own place, lit, its colours
  settling to muted by 300 ms, so nothing moves when the exit ends. Only a phase
  some frame drew live plays it: a phase that began and ended in one batch
  of events (a replay, a resume) is drawn as its summary row at once.
- **The summary row**, kept in the conversation, on the tool rows' ground
  and aligned with them: `▸ ✓ Thought for 2.8s · 719 thinking tokens`,
  the duration words in the thinking lavender, the rest muted, its end time
  and origin at the right, the origin left out whole when the row cannot
  hold both, never cut (80 columns; plan 22, correction C9). Claude Code's
  count is the message's exact one
  (`message_delta`), arriving 0 to 2.4 s after the phase ended, lit for
  600 ms when it comes; until then "≈ 650 thinking tokens (estimate)", or
  no count. Codex: "Thought for up to 3.1s · 83 reasoning tokens", the
  step's count. A count shared by several phases of one message or step is
  shown once, on its last, "over 2 phases". "Thinking interrupted after
  0.2s" when the turn was interrupted, with the count when one was
  reported: a message cut short reports none, the 0 its CLI sends with it
  being no count (plan 22, correction C1); "Thinking stopped after 0.2s ·
  the turn failed" when it
  failed or the session crashed; a Codex phase's time is "after up to".
  Durations: one decimal under 10 s, seconds under a minute, then "1m
  05s".
- **Details**: → on a summary row unfolds its details under it (▾), ←
  folds them, Enter shows the phase whole: its start and end and how each
  is known (observed, or the step's start inferred), each count's source
  and scope ("exact, for this phase · message_delta
  usage.output_tokens_details.thinking_tokens"), the last live estimate,
  and that the CLI sends no thinking text. A phase whose end the CLI did
  not report (its block's stop never came, its turn ended it, or a Codex
  reasoning item's next step began: correction C4) says so.
  A count that can no longer come reads "no exact count reported": a phase
  whose turn or session ended with none takes none later, neither a
  resume's replayed usage nor a later turn's (plan 22, correction C2). A
  message cut short reads "no exact count: the CLI reports 0 for an
  interrupted message", beside its last live estimate (C1). A Codex phase's
  details also give its reasoning item's start and end, observed, and
  Codex's summary in its own words when it sends one ("Codex's summary, its
  own text: **Computing f(10)**"), else "no reasoning summary: Codex sent
  none" (plan 22, amendment 20). The entry cursor stops on a
  summary row, never on a live card. A summary row between tool calls folds
  with their run; a run's fold row still counts calls only.
- **The activity line**, while the turn works and nothing else is live
  (no card, no text streaming, no request waiting), at the end of the
  exchange in the CLI's block, its header first when the CLI has said
  nothing yet: the spinner, an expression, the turn's time at the right
  (`◝ Vibing...   00:24`), and "◇ 1 subagent working" while agents run
  (one waiting for the owner makes the turn wait, and the line is then
  off). It shows once that has held 200 ms, longer than any gap a turn
  leaves between two of its parts (an answer's end and its turn's: 27 to
  145 ms in Codex, 0 to 27 ms in Claude Code), so it never flickers, and
  it comes 200 ms after a prompt (plan 22, correction C6). The work itself
  is what the tool rows show; the line is decoration.
- **While a request waits** for the owner, nothing of the turn animates:
  the pink box is what moves the turn on.
- **The editor's placeholder** reads "Claude Code is processing your
  request…" while a turn is under way, a request waiting or an interrupt
  asked for included, and the draft is empty (image 1).

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
- **Thinking text**: neither CLI sends any (Claude Code 2.1.303's
  `thinking_delta` lines are empty; Codex 0.163.0's reasoning items too,
  no summary being asked for: claude-code.md and codex-cli.md, section
  12). A phase is its summary row, whose details unfold ([Thinking](#thinking);
  ASM-22).

## Zone C

The live subagents (AM2 C1-C11; AM4 D3), the whole width under A and B
since plan 20, drawn after the owner's image and its close-ups since plan
27 (brief amendments 26 and 27; [reference-design.md, Zone C with the
execution timeline](reference-design.md#zone-c-with-the-execution-timeline)):
the header inside the frame, the agents' panel on the left, as wide as its
cards, and the execution timeline on the right.

- **The frame**: its top edge carries no title (as zone A since plan 24);
  the border's colour marks the focus. Inside, row 1 is the header, "C ·
  LIVE AGENTS" in the title colour, bold, at the left and "N ACTIVE · M
  DONE" in grey at the right end, counts of the agents it lists, derived
  from the agent machines (`Running` and `WaitingApproval` are active);
  before them, in amber, how many agents are out of view, "↑ 2 above · ↓ 5
  below", and the "* requested model" footnote. Row 2 is a rule across the
  inner width in the title colour (`stPanelRule`, not bold; white on 16
  colours), the close-ups' line, not the frame's dim grey. Row 3 holds the
  panels' headings, "AGENTS" over the cards and "EXECUTION TIMELINE" over
  the timeline, in the title colour, with the selected agent's status at
  the timeline's right end: its symbol and its state in capitals, in the
  symbol's colour ("✓ COMPLETED" green, "◉ RUNNING" amber, "● WAITING"
  pink, "✗ FAILED" red, "○ STOPPED" grey, "◉ ANNOUNCED" amber), dynamic as
  the state changes. A vertical line in the title colour separates the
  panels from row 3 to the bottom. With no agent in play, C is 4 rows: the
  header with "0 ACTIVE · 0 DONE" and "no agents yet: they appear as the
  CLI starts them".
- **The agents in play only** (plan 18, brief amendment 14, the owner's
  third message): each prompt the owner submits clears C of the agents of
  earlier prompts, back to "0 ACTIVE · 0 DONE" and "no agents yet"; an agent
  still running stays, and so do the agents of turns the CLI opens by itself
  for a background agent's notification. The earlier agents stay in zone A's
  tree, from which they can be inspected.
- **The cards**, the left panel: two rows an agent, the close-ups' card
  (plan 27, design D8):

  ```text
   ●  general-purpose     │  haiku-5-5
      00:06   21.2k tok   │  tool: Write
  ```

  The first row: a margin cell, the status symbol (a pink `●` for an agent
  waiting for the owner, where zone A keeps its `!`), two cells, then the
  label in white bold (Claude Code's description or `agent_id`; Codex
  `agentPath`; else, for a Codex agent its spawn request alone names,
  "task: " and the first line of the request's `prompt`, cut at 24
  characters with "…", replaced by the agent's own label when one comes,
  the whole prompt in the inspection; plan 10, item 1) padded to the label
  column, three cells, a grey bar, two cells, then the model in grey
  (Claude Code: its forwarded messages' `message.model`, shown without the
  "claude-" prefix, in full in the inspection; Codex:
  `subAgentActivity.model`, from Codex CLI 0.163.0, else the model its
  `spawnAgent` call requested, `collabAgentToolCall.model` for each of its
  `receiverThreadIds`, source "spawn request": the agent's own report
  always wins, in either order; plan 6, item 4, from the generated schema;
  recorded by P9a on the multi-agent v1 backend, plan 9) or "model N/A"
  (older Codex children, ASM-14; plan 5, item 7), a requested model
  followed by "*", footnoted "* requested model" in the header (plan 6,
  item 5). The second row, under the label: the elapsed time as a clock
  ("01:11"), an agent's retries as "↻ 2" in amber after it (none while it
  has none; AM2 C7; plan 21, item 5.6a: Codex reports an agent's retries,
  Claude Code's `system/api_retry` is the orchestrator's; its documented
  `tool_progress` `subagent_retry` is not mapped, since no recording holds
  one), the tokens, padded to the label column, then the bar at the same
  cell and "tool: " in grey with the agent's current tool in white bold:
  its last call's name, a call waiting for the owner included (the Write
  of P3b's waiting agent), else the tool the CLI reported (plan 7, item
  1), else "none yet" in grey; a failed call's red "✗" after the name, no
  symbol otherwise. Nothing else is on the rows: the state is the timeline
  heading's; the summary, the error, the effort and the model's source are
  the inspection's (Enter). The selected agent's two rows lie on the tool
  rows' ground (plan 19) from the cell after the margin to the cell before
  the panel's end, the close-ups' band; on 16 colours, which have no
  ground, nothing marks it, both palettes drawing the same characters.
  Rows are ordered: waiting for approval, running, then ended, most recent
  first (ASM-22); the panel scrolls with the selection, which stays in
  view.
- **The panel's width** (design D9): as wide as its cards, not a share of
  the screen, since the close-ups' card fills its panel and a two-fifths
  share of 208 columns left most of it empty: 12 cells of margins, gaps
  and bar, plus the label column, the longest label or second-row content
  over the listed agents, at least 17 ("00:41   30.1k tok"), plus the tool
  column, the longest model with its mark or "tool: <name>", 11 ("tool:
  Write") to 22 ("tool: commandExecution"). The whole never exceeds the
  larger of 39% of the inner width and 40, nor the inner width less 23
  (the timeline keeps 20 cells): over that, the tool column gives way down
  to 16, then the label column down to 17, then the tool column down to
  11, names cut with "…", whole in the inspection. A Claude Code
  subagent's card gives 40 cells at every width from 100 columns (the
  timeline 73 at 120 columns, 153 at 200, 161 at 208); two Codex agents
  with "tool: commandExecution" give 52 at 208 and 45 at 120.
- **The execution timeline**, the right panel: the selected agent's
  entries (its messages, then its history lines), the newest last, one
  entry each: the time "17:11:31" in grey, the role in its colour padded to
  9 cells ("notice" in zone B's speaker blue, "assistant" in its result
  green, any other role grey), a grey vertical bar, then the text wrapped
  onto at most two rows of the room left, the second cut with "…" when the
  text goes on, the bar drawn on both rows; a rule across the panel in the
  frame's colour between entries, none after the last. The panel shows the
  tail; the wheel scrolls it (panel P, by rows), and while it is held away
  from the tail the heading's right end reads "↑ n ↓ m · End follows" in
  place of the status. "no activity yet" in grey when the agent has no
  entry. The agent's model with its source and its effort, which the
  preview's heading carried until plan 26, are the inspection's; the rows
  carry no origin label, the role names the source.
- **Height**: 4 rows while no agent is in play. With agents in play from
  100 columns, C needs its three heading rows, the larger of the cards'
  rows (two an agent) and the timeline's eleven (four entries of two rows
  and three rules, the image's four), and its two edges: 16 rows for one
  agent. The cards' rows take up to half the rows A, B and C share (plan
  18: six agents at 120x40 give C 17 rows, twenty 18; twenty at 200x60 give
  28, the panel scrolling with the selection past that), the timeline's up
  to a third of them, and C never has less than one agent's rows with the
  headings, 7, A and B giving the rows (plan 21, item 4): with one agent
  16 at 208x59 and 200x60, 13 at 156x44 (three entries), 12 at 120x40 (two
  entries), 7 at 160x24 to 160x26 and 8 at 160x27. B keeps its least rows
  (twenty agents at 100x24 leave C 9). C keeps its height after the agents
  finish, until the owner's next prompt clears it; a turn the CLI opens by
  itself does not.
- **Under 100 columns** the narrow layout keeps plan 18's one-row list:
  "C · AGENTS" in its top edge, the counts at the edge's right, one row an
  agent (symbol, label, the model in its family's colour, the state, then
  the time and tokens at the right), no timeline, 3 rows while no agent is
  in play and at most 5.
- **Keys** when focused: ↑ ↓ select, Enter inspects, z zooms; PgUp, PgDn,
  Home and End scroll the list, the selection kept. C has no key hint row
  since plan 20: while C has the focus the footer reads "↑↓ select · Enter
  inspect · z zoom · PgUp PgDn Home End scroll · F6 zone · Esc back to the
  input · F1 keys".
- Background agents that outlive their turn stay listed and running (P3,
  P3b): the agent machine, not the turn, decides their status.

## Zone A: telemetry

The values of zone A's telemetry pane and of its details view (AM2 D1-D9;
AM4 D3, D5, D6), which were zone D's until plan 23. Every value is a `Value`
(events-and-state.md, State model): present with its origin and source, or
"N/A" with its reason. Values by provider:

| Value | Claude Code | Codex |
| - | - | - |
| 5-hour, 7-day quota | percentage bar, from the `/usage` report read right after the handshake and after each turn (documented; plan 24, amendment 23), updated between reads by `unifiedWindows` **U**, marked † "undocumented field"; reset time | percentage bar from `account/rateLimits/*`, windows identified by `windowDurationMins`; reset time |
| Model-scoped weekly windows | one block each, "<Model> limit", only once reported: the report's `weekly_scoped` rows, named by `scope.model.display_name` ("Fable limit"), a `unifiedWindows` key or a `rateLimitType` of the form `seven_day_<model>` (`quota.7d.<model>`); never assumed from the model or the plan | none |
| Context | % of the window, from `get_context_usage` or derived from usage | % of the window, from `thread/tokenUsage/updated` |
| Tokens | session total from `result.usage` and messages | session total, derived: the sum over threads, labelled so |
| Cost | "$0.42 API-equiv. estimate", from `total_cost_usd`, never mixed with quota | "cost N/A (Codex reports none)" |
| Model | from `system/init` and each message | from `thread/start`; children "N/A" |
| Agents | active count, derived; "limit N/A" | active count, derived; "limit 3 (config)" from `config/read`, or "limit N/A (not set; default not reported)", or "limit N/A (set per backend; backend not reported)" when `features.multi_agent_v2` sets a different limit, or "limit N/A (not read)" |
| Time | turn and session durations | same |
| Errors, retries, unrecognised events | counters in the details view only (plan 26: zone A's notes are gone), which lists the last errors and the last 20 unrecognised lines, each with its time, the mapper's type and reason, and the line (plan 25) | same |

- **Bars**: lavender for every window, the 7-day's no longer amber (brief
  amendment 22, section 6.1: warning colours are for warnings), red from 90%
  (ASM-13). A window whose reset time has passed reads "N/A" with "window
  reset; waiting for the next update" until a new value arrives (TEL rules).
- **Freshness**: the details view shows each value's age ("4s ago" after
  its source); zone A shows no age since plan 26, and a window whose reset
  passed reads "N/A" there (above), so a report that stopped coming is
  still visible.
- **Before the first turn**: Claude Code's windows are in zone A as soon
  as the session is ready, from the usage report the adapter asks for
  right after the handshake; the exchange is never a conversation entry,
  a stage or an agent (events-and-state.md, Polling; plan 24, amendment
  23).
- **N/A** is grey, never zero, and always carries a short reason on screen
  or in the details view: on a quota window's second row ("no report yet",
  "API key: no subscription window", "plan has no 5-hour limit"), as the
  support of a usage column ("no result yet", "Codex reports none") (TEL CC,
  TEL CX).
- **Errors and telemetry**, the details view. Enter on any of the pane's
  three headings, or a click on TELEMETRY, opens it full screen: "◀ back
  (Esc)   ERRORS AND TELEMETRY" with "N errors · last 50 kept" at its right
  end (the title lost its "D ·" in plan 24). It lists the last errors,
  newest first and whole; then "UNRECOGNISED  lines the dashboard could not
  map, newest first", the last 20 of them, each on two rows: its time, the
  mapper's type ("system/tool_use_summary", "user/tool_result_meta") and
  its reason when it gave one, then the line itself cut to the width with
  "…", or "none" (plan 25); then, under "TELEMETRY  value · source ·
  received · † undocumented field" (the legend lives here since plan 26),
  every value with its source and age ("quota.5h 35%† ·
  rate_limit_event.unifiedWindows 0s ago", "(derived)" after a derived
  value), then, under the values, the counters zone D's rows held until
  plan 23: agents, session time, retries and unrecognised events. ↑ ↓,
  PgUp, PgDn, Home and End scroll it, it updates live, and Esc returns to
  the dashboard with A focused and its cursor where it was.

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
 Enter full view · ↑↓ entries · Esc back to the input · F3 file · F1 keys
```

**The window.** `max(W*4/5, min(W-4, 84))` columns by `max(H*4/5,
min(H-2, 22))` rows, centred: 76x22 at 80x24, 96x32 at 120x40, 160x48 at
200x60; `f` fills the screen under the title bar, as the inspection does,
so the title bar and a mock run's "◆ MOCK — not a live session" stay on
screen (plan 21, item 5.7), and back. A rounded frame in the title colour;
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
the agent that asked, and marks that agent "!" in A and a pink "●" in C.

- **The card** (plan 20, Template 2, the owner's section 5.2): a pink
  rounded box inside the CLI's card, never folded, never green:

  ```text
  ╭─────────────────────────────────────────────────────────────────────────────────────╮
  │ ! APPROVAL · general-purpose · Write                                    01:55  mock │
  │   <probe-root>\probe-work\p3b\found.txt                                             │
  │ ╭─────────────────────────────────────────────────────────────────────────────────╮ │
  │ │ ▸ Preview of found.txt (2 lines)                                                │ │
  │ │  1 │ alpha.txt                                                                  │ │
  │ │  2 │ beta.md                                                                    │ │
  │ ╰─────────────────────────────────────────────────────────────────────────────────╯ │
  │ This will create or overwrite the file at `<probe-root>\probe-work\p3b\found.txt`   │
  │ with the above content.                                                             │
  │ ▐ F2 to answer ▌                                                                    │
  ╰─────────────────────────────────────────────────────────────────────────────────────╯
  ```

  Its heading, "! APPROVAL" bold pink (a question's "? QUESTION"), who asks
  (orchestrator or the agent's label) and the tool, bold pink, with the
  request's time and origin at the right; the target on its own row (a path
  losing its left end, a command its right end, only when the box is too
  narrow); then the questions, or a preview of what the request would do,
  on the tool rows' ground: a file's content numbered ("Preview of
  found.txt (2 lines)"), a change's lines removed and added ("Change to
  a.go"), a command's lines ("Command"), at most 6 lines then "… N more
  lines". From zone B's 90 cells the preview has its own grey box, as
  Template 2 draws it; narrower, its rows go straight in the card. Then the
  request's own description, when it says more than the target; then, for
  the tools whose effect Claude Code documents, one sentence on what
  allowing does, from the request's own tool and target (its tools
  reference, code.claude.com/docs/en/tools-reference.md, read on
  2026-10-20: Write "creates a new file or overwrites an existing one with
  the full content provided", Edit "performs exact string replacement"):
  "This will create or overwrite the file at `<path>` with the above
  content." (Template 2's), "…, leaving it empty." for empty content, "This
  will replace the text marked - with the text marked + in `<path>`."
  ("every occurrence of" with `replace_all`); no sentence for another
  tool or for Codex, nothing invented. Last, the pill "F2 to answer", pink
  bold on a dark pink ground with half-block ends (in brackets on a
  16-colour terminal); once F2 gives the card the focus, its keys in its
  place.
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
  classifier decided reads "✓ Write  <path> · approved by auto-review" in
  B, the verdict green, red when it denied (plan 19); a
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
  Since plan 19 that line is a row among B's tool rows, indented to them
  and on their ground, the outcome green, red or grey; a pending card stays
  apart from them, never on the ground and never folded, so a request
  waiting for the owner never looks like one already answered.
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
| Waiting for approval | "!" in the colour of attention on the agent in A, a pink "●" in C, the pink card in B, "! Waiting for approval" in B's header |
| Interrupting | "Interrupting…" in B's header; after 10 s, the editor's row reads "no turn end after the interrupt: force stop closes the session", and F force-stops (ASM-16; `internal/state/actions.go`) |
| Interrupted, cancelled | the CLI's own note in B when it sends one (Claude Code's "[Request interrupted by user]", amber, observed); zone A's stage "○ interrupted"; open tools end "unfinished" (inferred); no notice of the dashboard's (plan 21, item 5.6c) |
| No data | "N/A" in grey with its reason; never a zero, never a blank |
| Error | ✗ in red with the first line of the error; the full error in inspection or in the details view's error list |
| Session crashed | B shows the exit code and stderr tail; send is disabled; "Ctrl+R resume · Ctrl+Q quit" (resume only once the session had reached Ready; earlier, the StartFailed card); the draft is kept; nothing restarts without the key ([reliability.md, Resumption](reliability.md#resumption)) |
| CLI not installed | on the startup screen only, "not installed" with the reason; not selectable |
| Unrecognised events | counted in the details view, which lists the last 20 with their time, type, reason and line (plan 25; zone A's note "unrecognised N" went in plan 26); listed in Raw events; never dropped |
| Mock mode | "◆ MOCK — not a live session" in the title bar of every screen |

## Origins on screen

Each line or value carries its origin (events-and-state.md, Origins; AM2
U9; TEL rules):

| Origin | Look |
| - | - |
| `ai` | normal text with no label, under the model's header in B (CLAUDE or CODEX, the model, the time); in C's preview the row's role, "assistant", and no origin label; in the inspection no label. AI text is the text without a label, every other origin keeping its own (`originLabel`, `internal/render/dashboard.go`; plan 21, item 5.6b) |
| `observed` | right-aligned grey "observed" on event lines; values plain |
| `derived` | right-aligned "derived", and the rule in the details view ("sum of 3 threads") |
| `inferred` | italic, with "inferred" (the stage tree's heading says "inferred from activity") |
| `mock` | the title badge "◆ MOCK — not a live session"; every value also labelled mock in inspection |
| undocumented source | "†" after the value, in zone A and in the details view, whose TELEMETRY heading carries the legend "† undocumented field" (AGENTS.md, "Prefer documented interfaces"; the legend was in zone A's notes until plan 26) |

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
| ● | an agent waiting for the owner, in zone C (plan 20, Template 2) | pink |
| ✗ | failed, denied | red |
| ↻ n | an agent's retries, after its time in C (plan 21, item 5.6a) | amber |
| ▸ | the entry cursor in B, in its card's margin (attention); a call with no record (grey) | |
| ▎ | a message's bar in B, on each of its rows: YOU, CLAUDE or CODEX (plan 20, amendment 17) | violet, blue |
| ─ | the rule between two messages in B, and B's input under its own | the frames' |
| ▸ ▾ | a run of tool events in B, folded or expanded (plan 19); a pending request's preview; a section of zone A's telemetry pane, folded or open (plan 24) | blue; grey |
| ▐ ▌ | the ends of the "F2 to answer" pill (plan 20) | dark pink |
| ✿ ◆ | the orchestrator's model icon in zone A: a Claude model, any other (plan 24) | the model's |
| ✦ | the TELEMETRY heading of zone A (plan 24) | title |


Model colours, after RD's legend (one colour per model family, ASM-13):
Opus violet, Sonnet orange, Haiku blue, OpenAI GPT models teal, any other
model in the text colour. The model name is always written next to its colour, and every
status has its symbol, so nothing depends on colour alone (a 16-colour
terminal or `NO_COLOR` still reads correctly). The colours are fixed in the
code (`internal/render/palette.go`): the application's own configuration
file holds the send and newline keys only (ASM-7; plan 21, item 5.1).

| Role | Value (ASM-13) | 256 colours | 16 colours |
| - | - | - | - |
| Frame | `#3B3F51` | 238 | 8 bright black |
| Panel title, zone A's heading, the quota and context bars' fill | pale violet `#C4B5FD` | 183 | 15 bright white |
| Text | `#E5E7EB` | 189 | 7 white |
| Done | `#4ADE80` | 78 | 10 bright green |
| In progress | `#FBBF24` | 214 | 11 bright yellow |
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
| Zone B: YOU's bar and label (plan 19) | `#A78BFA`, bold | 141 | 5 magenta |
| Zone B: the owner's prompts | `#F9FAFB` | 231 | 15 bright white |
| Zone B: CLAUDE's or CODEX's bar and label | `#60A5FA`, bold | 75 | 12 bright blue |
| Zone B: RESULT's box and label | `#4ADE80`, bold | 78 | 10 bright green |
| Zone B: tool rows' ground, background | `#1E2133` | 17 | none |
| Zone B: a tool's name (plan 20) | `#93C5FD`, bold | 117 | 6 cyan |
| Zone B: a tool's target, a fold's count, the model in a message's header, an agent's state in C (plan 20) | `#A1A9C0` | 145 | 7 white |
| Zone B: a fold's mark, an agent's type (plan 20) | `#60A5FA` | 75 | 12 bright blue |
| Zone B: the pill's ground, background (plan 20) | `#3B1531` | 53 | none (in brackets) |

The bars of zone A's telemetry, the context's and the quota windows', fill
in the title colour, red from 90%, on the thinking card's track colour
(`#2E2A47`, 8 bright black on 16 colours; plan 22), so a bar in A and the
card in B share one track.

The application draws the true colours whatever the terminal, except on a
16-colour one (plan 5, item 13). Bubble Tea detects the terminal's colour
profile and sends it at startup, and again when the terminal reports RGB
support; its renderer then writes each true colour as is, maps it to the
nearest of the 256, or drops every colour when the output is no terminal.
The 256 column is that mapping as measured (`internal\render\palette_test.go`,
TestPalette256): one index per colour, no two foregrounds and no two
backgrounds on the same index; the zone A cursor row comes out a dark
teal. On a 16-colour profile the renderer's
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
  tool, a starting session, zone A's status), at 10 frames per second; it
  stops the moment the work stops (ASM-18).
- Elapsed times tick once per second, and so do zone A's quota countdowns
  ("Resets in 4h 12m") and the details view's ages ("4s ago"): they
  follow the one-second clock, with no timer of their own, and are drawn
  from the frame's time, so a countdown that changed rewrites its own row
  and none of B's or C's (plan 24).
- In zone A, the connector of the running branch carries a moving dot, the
  reference's "animated dots", at the same 10 fps (ASM-18).
- Zone B's thinking card (plan 22, brief amendment 18; Zone B, Thinking):
  its circular spinner (six frames, 100 ms each), its expression changing
  every 4 s, its bar growing over 300 ms after each new value with a
  shimmer between values or, with no count, a segment sweeping the track;
  its entrance over 280 ms and its exit over 300 ms. While one of these
  runs a frame comes every 50 ms (20 fps, ASM-18), the 10 fps otherwise.
  Every animation is drawn from the events' times and the frame's time,
  never stored, but for two facts of the interface: the phases a frame drew
  live, which alone play their exit, and when the activity line's
  condition began, which it shows 200 ms after (plan 22, correction C6).
- Elsewhere no slide, fade or blink: the thinking card's dim-to-full
  transitions are the only fades. No animation when idle, so an idle
  dashboard draws nothing (plan 22, criterion 3.2).

### Rendering

- Events come in bursts, not coalesced by kind: each message to the
  program carries the events waiting, at most 512 (`maxDrain`), applied as
  one update, so a burst is one screen update; Bubble Tea draws at most 60
  frames per second, its default (ASM-3; architecture.md, Threads of
  execution; plan 21, item 5.4). Typing is not delayed by a burst: a key is
  applied and drawn in the next frame.
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
- What is cached across frames (`internal/render/content.go`): each source
  line's wrap, by the line and the widths (`WrapCache`), and a finished
  entry's row count, by the entry, the width and its shape (`rowCount`); a
  streaming entry, an approval, and a thinking phase while live or in its
  exit are counted afresh (plan 22). Markdown is
  classified again for each entry drawn and styled for each row drawn
  (`mdLines`, `internal/render/cards.go`), with no cache; only the rows in
  view are built, and the stream's frame time is within its target without
  one (`TestStreamToFrame`; plan 21, item 5.6d). The file viewer's
  colouring runs in a command, never while drawing (File viewer).

## Focus and keys

The input in B has the focus by default (AM4 D4; AM2 U10); Esc always
returns to it. The editor's send keys and newline keys are lists,
configurable in the application's own file, never a CLI's (TI 4.3; ASM-7;
plan 21, item 5.1); every other binding is fixed in the code (ASM-9). The
defaults avoid the keys Windows Terminal takes (TI 9). F1 or `?` (outside
the editor) shows this table as an overlay, with the keys in force.

The file is `%APPDATA%\agent-command-center\config.json`, read once at
start and never written: `{"keys": {"send": ["ctrl+enter"], "newline":
["enter"]}}` makes Ctrl+Enter send and Enter add a newline. A list left out
keeps its default, Enter to send and Ctrl+J, Shift+Enter and Alt+Enter for a
newline. A key is Enter with any of Ctrl, Alt and Shift, or Ctrl+J; the
modifiers are written in any order and case. A file that cannot be read or
is not JSON, an unknown setting, another key, an empty list, or a key in
both lists or twice in one: the first screen names `config.json` and the
problem, the startup screen under its project field, a mock run's editor
row in red, the row its alone (only a quit question goes before it) until
a message is sent, and the default keys apply. The footer, the draft
window's hints and this overlay show the keys in force (the editor's row
carries no keys since plan 29); a send key keeps Enter's rules: never
inside a paste, and after `\` a newline.

The same file can name Codex's proxy (plan 22, amendment 19).
- **Setting.** `{"codex": {"proxy": "http://127.0.0.1:3128"}}` starts
  Codex with `HTTPS_PROXY` and `HTTP_PROXY` set to that address.
  - It is meant for a local proxy that signs in for the owner, such as Px.
  - The commands Codex runs inherit it.
  - Claude Code takes its proxy from its own settings, so the setting
    leaves it alone.
- **Startup screen.** It shows the address in force on a grey line,
  "Codex through the proxy http://127.0.0.1:3128 (config.json)".
- **Refused values.** The address must be `http://` or `https://` with a
  host, no user name or password, no query or fragment, and no path beyond
  a trailing slash. A user name or password is never repeated in a message. The startup screen names
  each problem as a red line under the keys' problem: `config.json: …; no
  proxy for Codex`. A problem is any of:
  - an address that does not fit those rules;
  - a value that is not a string;
  - an unknown setting under `codex`.
- **Independence.** A problem in `codex` leaves the keys in force, and
  Codex inherits the dashboard's environment as it does with no setting. A
  file that cannot be used as a whole gives no proxy; the keys' problem
  already names it.

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
| F6 / Shift+F6 | next / previous zone (B, C, A) | ASM-9 |
| Tab / Shift+Tab | next / previous zone, outside the editor (in the editor Tab inserts a tab) | ASM-9 |
| Esc | close the "+" menu (plan 30), close the overlay, leave zoom or inspection, clear the selection, back to the input | ASM-9 |
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

**A focused panel** (A, C, inspection)

| Key | Action |
| - | - |
| ↑ ↓ | move the cursor (node, agent, line); in A through the turns, then TELEMETRY and its sections |
| ← → | in A fold and unfold a turn, SESSION USAGE or QUOTA WINDOWS; in inspection change tab |
| Enter | inspect the agent; in A on TELEMETRY (or either section) the errors and every value |
| z | zoom the panel full screen; z or Esc again restores |
| PgUp PgDn Home End | page, top, end; End follows again (A, inspection) |
| `[` `]` | previous / next agent (inspection) |

**The approval card**: a, s, d, c, Enter, Esc, ↑ ↓ ([Approvals](#approvals)).

**B's entries** (↑ from an empty input, [Zone B](#zone-b))

| Key | Action |
| - | - |
| ↑ ↓ | previous / next folded message, tool or failure; ↓ past the last, the input |
| Enter | the whole entry, full screen; Esc back to the cursor; on a run of tool events, expand it |
| → | on a run of tool events, expand it; it opens nothing |
| ← | on a call of an expanded run, fold the run again |
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
  top, the selection in view (plan 5, item 5). Zone A's telemetry pane,
  under its tree, does not scroll (plan 24). Scrolled, a pane is held: rows
  coming and going do not move it, ↑ ↓ keep the cursor or selection in view
  with the least move, and the pane says so in amber, the rows out of view,
  then the way back: A, which has no hint row since plan 24, in its top
  edge, "↑ 12 ↓ 25 · End follows"; the preview in its heading, "↑ 4 ↓ 2 ·
  End follows"; C, which has no hint row since plan 20, in its top border,
  "↑ 2 above · ↓ 5 below". A and the preview follow again at their end; the
  preview also for another agent. End in A with the cursor above the tail
  shows the tail and keeps following it. Zoomed panes scroll the same way.
- Collapsing: in A, with ← →, closed turns fold to their block by default
  and open ones are unfolded, and SESSION USAGE and QUOTA WINDOWS fold too;
  in B, long pastes (first lines and "… N more lines"), tool lines and
  failures (one row), each shown whole with Enter on B's entry cursor
  ([Zone B](#zone-b), Expanding).
- Full screen: z zooms A or C; Enter on an agent opens inspection; Enter on
  TELEMETRY or on a section of A's pane opens the details view; Enter on an
  approval card shows the whole diff; Enter on B's entry cursor shows the
  entry whole. Esc goes back to the exact previous view.

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
- **Size**: one row when empty or on one line, under the rule, then a
  separator and the row, no empty row between them (plan 20, the owner's
  templates; plan 29, the owner's image); it grows with its content up to
  half of B, the separator and the row counted in that half, then scrolls.
- **The row** (plan 29, brief amendment 29; [reference-design.md, The
  input editor's row](reference-design.md#the-input-editors-row)), under
  a separator in the frame's colour:

  ```text
   +     Haiku 5.5 High · Ultracode    12 lines · turn running: the draft waits · Ctrl+C interrupt    ↯ Auto
  ```

  A margin cell; " + " on zone B's tool rows' ground (plan 19; the image's
  grey against the ground's), the "+" in the text colour; three cells; the
  pill on the ground, " <model> High · Ultracode ": the model the session's
  own as zone B's speaker names it ("Haiku 5.5", "Fable 5.1"; a Codex id as
  given; "model N/A" before the CLI reports one), "High" in the text
  colour, the dot in the title colour, "Ultracode" in violet bold (YOU's
  colour); three cells after the pill, in grey, the draft's line count
  once it passes one line, then what the status line said when it had
  something to say (a turn running, "pasting…", a notice, the session's
  phase, the quit question), joined by " · "; at the right end "↯ Auto"
  and one cell. When the middle does not fit beside the pills, the row is
  the middle alone, as the status line was; the configuration file's
  problem keeps its red row alone (plan 21, item 5.1). The "+" opens its
  menu (plan 30, the next bullet); "High · Ultracode" and "Auto" are the
  owner's design placeholders for the model and the effort selections: no
  key, no click, no event, no state behind them; a click on the row
  elsewhere focuses the editor. The idle
  hint "Enter send · Ctrl+J newline" and "Ctrl+G draft window" went with
  the row: the footer and the F1 page keep the keys (the footer leaves the
  newline key out when it has no room). On 16 colours the ground is not
  drawn, as zone B's tool rows there.
- **The "+" menu** (plan 30, brief amendment 30; [reference-design.md,
  The "+" menu](reference-design.md#the-plus-menu)): a click on the band
  opens a three-row box above the editor's rule, its left edge on the
  band's cell, its bottom edge on the rule:

  ```text
  ╭──────────────────────────╮
  │  🗀  Upload from computer │
  ╰──────────────────────────╯
  ```

  The box in the frame's colour, the item (the folder icon U+1F5C0 and
  the text) in the thinking edge's lavender-white. While the menu is open
  the band lies on the active ground, `#272745`, the image's indigo (ANSI
  4 on 16 colours), its "+" in that lavender-white, and the dashboard
  asks the terminal for every mouse motion (cell motion otherwise, as
  before, so idle moves cost nothing): the item's inner row lies on the
  active ground while the pointer is over it. A second click on the band,
  a click on the item, a click anywhere else, Esc or any other key closes
  it; Esc does nothing more, the other click and the other keys act as
  usual. The band is panel '+', the menu panel 'U'. On 16 colours the
  ground is not drawn, as zone B's grounds.
- **Attached images** (plan 31, brief amendment 31; the formats and the
  providers' limits in [doc/research/image-input.md](research/image-input.md)
  and plan 31's matrix): a click on "Upload from computer" opens the
  system's Open dialog over the terminal, "Upload from computer", the
  filter "Images" with every extension the owner listed and "All files",
  several files at once; cancelling it changes nothing. Each file chosen
  is read and checked by its bytes in a command; the accepted ones become
  the draft's attachments, after those already attached, the text and the
  cursor untouched; a file already attached is skipped, a refused one is
  named with its reason in a toast, and so is what the sending changes
  ("converted from BMP", "animated: the first frame is sent", "resized
  from 9000x6000 to 8000x5333 for the provider's limits"). Choosing an
  image never sends anything. While the draft has attachments, a row of
  chips sits between the editor's rule and the prompt, within the
  editor's half of B:

  ```text
   acc-image-upload-test.png · 1 KB ✕    acc-image-upload-test.jpg · 6 KB ✕
  ```

  Each chip on the tool rows' ground, the name cut at 28 cells, its size
  as sent in grey, a ✕ that removes it (panel 'X', its index); two cells
  between chips, those past the width folded into "+N more"; Ctrl+Shift+X
  removes the last. The send key sends the text and the images together,
  a message of images alone included; each image is read and checked
  again as it is sent, and a file gone or changed past a limit fails the
  send, the text and the images back in the editor with the reason ("the
  message could not be sent: x.png: the file is gone"). A file whose bytes
  changed since it was attached (their SHA-256, kept when it was attached)
  fails it too, "x.png: changed since it was attached; attached again as
  it is now": the dashboard reads it again and replaces its chip and
  thumbnail in place ("x.png: updated"), so the next send sends what the
  preview shows; attaching that file again by hand does the same, and an
  unchanged one is "already attached" (plan 33, design D7). The stored
  draft lists its images beside its text; a restart restores the ones
  still there and names the ones gone.
- **The stored draft** (ASM-23; plan 33, brief amendment 33): each
  dashboard's draft has a folder of its own in
  `%LOCALAPPDATA%\agent-command-center\drafts`, named by the SHA-256 of
  the project's path and a random key, holding the text byte for byte
  (`draft.txt`), the images list (`images.json`: files' paths, pasted
  images as `pasted:pasted-image-N.png#<sha256>`), the pasted images'
  files (`<sha256>.png`) and `lock`, which the dashboard holds open with
  no sharing while the draft is its own, its CLI's id in it. Typing is
  saved at most every 500 ms; attaching, pasting or removing an image at
  once. While a sent message waits for the CLI's acceptance, the stored
  draft is that message, then what was typed and attached since (the
  texts joined by a newline, no image twice); the acceptance leaves what
  was typed since. Quitting, a panic, a closed window and Back write the
  draft and let it go; a draft let go, or left by a dashboard killed,
  waits for its project: a session's start takes one of its own CLI
  first, then the newest, and says how many more wait ("draft restored ·
  1 more stored draft of this project"); two dashboards on one project
  each keep their own. One whose lock another program has open a moment
  (an antivirus, an indexer) is waited for, a second at most for the
  whole start, then said ("a stored draft of this project is in use: the
  next start takes it"); one another dashboard has just taken is left at
  once. A stored draft's images are attached
  whatever this session's limits (a draft stored by the other CLI), the
  toast saying "more than this session sends in a message: remove some
  before sending", and the send refuses the message with the reason
  until some are removed. Enter again while a message waits for its
  acceptance is refused and leaves that message in place. A draft from before plan 33 (`<sha256>.txt` and
  `.images.json` beside the folders) is taken into a folder. A save
  Windows refuses because another program holds the file is tried again
  for about a second, then said ("draft not saved: ...") and tried again
  every second while it fails. An images list that cannot be read is
  set aside as `images.json.unreadable` and the text restored, "draft
  restored; its images list was unreadable and was set aside". While a
  stored draft's images are read back, however long that takes (a
  network share gone offline), saves go on, listing them first as
  stored, after a message waiting for its acceptance; once read, they
  take their place before the ones attached meanwhile; pasting again one
  of them is "already attached". A stored file whose drive or share
  cannot be reached is kept: its chip reads "name · not reachable", its
  thumbnail's box "offline", the toast "x.jpg: its folder cannot be
  reached; kept in the draft", and the send is refused with "x.jpg: its
  folder cannot be reached" until it answers or the chip is removed; a
  file gone from a folder that answers is left out, said. At each
  start, folders nobody holds lose their temporary `.draft-*` files and
  the pasted images' files their list does not name, and are deleted
  when they hold no draft.
- **Back** after a failed start writes the draft and lets it go, and the
  editor forgets the text, the images, their pasted bytes and their
  thumbnails: the next session, in any project, starts without them, and
  the project's next start restores them (plan 33, design D2).
- **Pasted images** (plan 32, brief amendment 32;
  [doc/research/image-paste.md](research/image-paste.md)): Alt+V attaches
  what the clipboard holds as images, its copied files first, each checked
  as an upload is, else its image (the "PNG" format's original bytes, else
  the bitmap as PNG), else the toast "no image on the clipboard"; it never
  inserts text and never sends. Ctrl+V pastes text as before; Windows
  Terminal keeps its press for its own paste, and when the clipboard holds
  no text it pastes nothing and forwards only the V key's release
  (undocumented, probe P32a): the dashboard, seeing that release with no
  press and no paste within 200 ms, attaches the clipboard's image as
  Alt+V does. A paste that is exactly the paths of the files the clipboard
  holds (Windows Terminal pastes a copied file's path) attaches them when
  every one is a supported image, and goes in as text otherwise. A pasted
  image is "pasted-image-N.png" (N from 1 in a run, after the restored
  ones), held in memory as PNG bytes and, since plan 33 (design D8), kept
  in the stored draft's folder as `<sha256>.png` until its message is
  accepted or its chip removed, nowhere else; the same bytes twice are
  "already attached". A restart restores it with its name, its bytes
  checked against the SHA-256, and its thumbnail; a copy missing or not
  matching is said ("pasted-image-1.png: its stored copy is damaged") and
  left out; a draft of plan 32, whose pasted images were never kept, says
  "pasted-image-1.png was pasted, not kept".
- **The preview** (plan 32;
  [reference-design.md](reference-design.md#the-image-preview)): when the
  terminal answers the device attributes request (`ESC[c`, sent once at
  start) with Sixel graphics (attribute 4), a strip of thumbnails lies
  between the editor's rule and the chips' row, side by side in the
  attachments' order, one cell apart (the owner: the space between them
  "way shorter"), the first above the first chip at its first cell, as
  many as the width holds, past the chips folded into "+N more": 14
  columns by 7 rows (140x140 image pixels, Windows Terminal giving a cell
  10x20 of them) from 50 rows, 10 by 5 (100x100) from 30 to 49, none
  under 30. The
  image is cut from its centre to a square, scaled by area averaging, its
  corners rounded (8 pixels, 6 at 100) and transparent, a 1-pixel border
  in the frame's colour. The frame keeps the strip's cells blank (panel
  'T', the image's index); the dashboard draws the images over them with
  Sixel, the cursor saved and restored, inside synchronized output, 50 ms
  after the frame that showed them and again 250 ms later, and again
  after any frame that changes a strip row; a place an image left is
  written over with the frame's own cells, which erases it. A WebP, which
  nothing here decodes, gets a box of text with "WebP" in it. Without
  Sixel, the chips alone.
- **While a turn runs**: editable, not sendable; "turn running: the draft
  waits · Ctrl+C interrupt" in the row's middle (ASM-15). Enter then does
  nothing but flash that line.
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
- An image is pasted by Alt+V, or by Ctrl+V when the clipboard holds no
  text; a paste of exactly the copied files' paths attaches them when they
  are all images (Input editor, Pasted images; plan 32).

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
  200x60, centred; **Ctrl+F** fills the screen under the title bar, as the
  file viewer's `f` does (plan 21, item 5.7), and back (`f` types a letter
  in a draft). The dashboard keeps updating behind it. On the top edge,
  "Draft" on the left, the line count on the right.
- **The same draft, the same editor**: the window shows `m.edit`, the input
  box's buffer, with every key of the [Input editor](#input-editor): a
  send key, Enter by default, sends and closes the window (on an empty
  draft it does nothing and the window stays), Ctrl+J and the other newline
  keys add a line (the keys in force, [Focus and keys](#focus-and-keys)), Ctrl+C
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
| Thinking text folded under the message, "thinking · 312 tokens", unfoldable | No thinking text; each phase a live card, then a summary row, "✓ Thought for 2.8s · 719 thinking tokens", whose details unfold (plan 22, [Thinking](#thinking)) | Neither CLI sends thinking text (claude-code.md and codex-cli.md, section 12); the owner's thinking display is brief amendment 18 |

## Assumptions used here

ASM-3 (60 fps, Bubble Tea's default), ASM-7 (own configuration file, the send and newline keys), ASM-8 (prefill flags),
ASM-9 (default keys), ASM-10 (editor budget), ASM-12 (layout geometry), ASM-13
(colours), ASM-14 (Codex child model N/A before 0.163.0), ASM-15 (no send
while a turn runs), ASM-16 (interrupt timeout), ASM-17 (Markdown renderer),
ASM-18 (animation rates), ASM-21 (timeouts), ASM-22 (display defaults); all are
listed in [architecture.md, Assumptions](architecture.md#assumptions).
