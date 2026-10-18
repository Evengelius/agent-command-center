# Plan 17 evaluation (2026-10-18)

The final check of plan 17, a file viewer in the terminal (amendment 11). It
checks the plan's four items in
[`plan.md`, Plan 17](../plan.md#plan-17---a-file-viewer-in-the-terminal)
against the yes/no criteria written there before the work started. It also
checks the plan's rules: the viewer only reads, project files only,
credential files refused by name before they are opened, one new direct
dependency (Chroma, approved by the owner on 2026-10-18), no agent CLI
started.

The evidence is the tests, the golden frames and the two terminal runs. The
block below is the summary reported to the owner. The results are not to be
rewritten.

```text
Evaluation - Plan 17, a file viewer in the terminal - 28 criteria: 28 PASS
PASS 1.1 A project file reads by absolute and relative path; CRLF lines come without CR; invalid UTF-8 shows U+FFFD (TestReadFile)
PASS 1.2 Refused with the reason: outside the project, a link inside pointing outside (a junction, TestReadFileJunction; a symbolic link where one can be made), a directory, over the lowered limit (TestReadFile)
PASS 1.3 Every credential name of the plan refused, the check before the open; token.go, secrets.go, .env.example, auth.go not refused (TestCredentialName, TestReadFile)
PASS 1.4 A NUL in the first 8 KiB: binary, with the size (TestReadFile)
PASS 1.5 The listing sorted, without .git, credential files and links, cut at the lowered limit and saying so (TestListFiles)
PASS 2.1 Go, JavaScript, TypeScript, Java, Python, Rust, C#, SQL, YAML, JSON, Markdown and Bash recognised by name, a keyword and a string of each in the keyword and string colours (TestHighlight)
PASS 2.2 Unknown name and content: "Plain text", every run in the text colour (TestHighlight)
PASS 2.3 Each line's runs joined give the line back byte for byte (TestHighlight)
PASS 2.4 The 16-colour palette: each syntax style takes its ANSI colour (TestSyntaxPalette)
PASS 2.5 A 1 MiB Go file highlighted whole; a larger one only in the range asked, its window saying "coloured around the view" (TestHighlight, TestFileStates)
PASS 3.1 76x22 at 80x24, 96x32 at 120x40, 160x48 at 200x60, centred; full screen fills the screen; the rows outside the window are the dashboard's and an event arriving meanwhile changes them (TestFileRect, TestFileOverDashboard)
PASS 3.2 A Read with offset 40 opens at line 40; an Edit marks its new_string's lines; a Codex fileChange marks its first added lines; an entry naming no file opens the quick open (TestFileFromEntry, TestToolFile)
PASS 3.3 The quick open lists a temporary project, filters, opens the selection, and says when the listing was cut (TestQuickOpen, TestFilterFiles)
PASS 3.4 Each key does what the plan says; no key the window takes changes the draft (TestFileKeys)
PASS 3.5 Outside the project, credential, binary, too large, not found, "reading…", "colouring…" shown in the window (TestFileStates, TestFileStatesApp)
PASS 3.6 A result for an older open, colouring or search is dropped (TestFileStale)
PASS 3.7 A drag across three lines copies exactly their text; a click on the dashboard behind selects nothing; the wheel scrolls the window (TestFileMouse)
PASS 3.8 Golden frames file, file-full, file-search, file-finder at 79x24, 80x24, 120x40, 200x60 (16 new, reviewed); no existing frame changed (git diff --stat testdata/frames: empty)
PASS 3.9 The key overlay lists F3 and the window's keys (TestKeyOverlayRows); every footer fits (TestFootersFit, TestFileFooters)
PASS 4.1 THIRD_PARTY.md lists chroma/v2 v2.27.0 (MIT, direct) and regexp2/v2 v2.2.1 (MIT, indirect); TestThirdPartyTable PASS; go.mod has Chroma as its fifth direct dependency
PASS 4.2 AGENTS.md (status, scope, dependencies), architecture.md, ui-design.md, reliability.md and evaluation.md say what was built; amendment 11 in brief.md
PASS 4.3 go build ./... and go vet ./... (also -tags live, -tags timing) exit 0; gofmt -l . prints nothing; the binary 9,999,872 bytes before (becfb34), 14,086,144 after
PASS 4.4 go test ./...: 324 top-level tests PASS (795 with subtests), 1 SKIP (TestDiskListLink, as in plans 15 and 16), 0 FAIL, in 14 packages (baseline becfb34 306, +18)
PASS 4.5 The timing step run alone, 4/4 PASS: p95 TestKeystrokeToFrame 1.03 ms, TestDraftKeystrokes 2.28 ms, TestLongSession delta-to-frame 3.59 ms (target 16 ms)
PASS 4.6 Terminal run, Windows Terminal maximised (156x44), mock run in a temporary project (Java, Go, TypeScript, .env, a binary .png): 29 checks PASS, 0 FAIL (third run; the first two failed through the script, see below)
PASS 4.7 Terminal run at 80x24: the same 29 checks PASS, 0 FAIL
PASS 4.8 The 53 changed and new files (this one included) carry no CR, no BOM and no NUL
PASS 4.9 This summary written here and in plan.md's progress list
Not verified: a real symbolic link in ReadFile and ListFiles (this account cannot create one unelevated; TestReadFileJunction covers a junction, the same resolution path). The viewer opened from a live agent CLI's tool entry (the runs replay a mock; F3 on an entry reads the same tool input).
Verdict: DONE
```

## The terminal runs

Both runs used one script. It sent real keys and a real mouse drag to a
Windows Terminal window, read the screen through UI Automation and took a
screenshot at each step. The binary was built from the working copy.

The project `demo`, under `%TEMP%`, outside the repository, held:

- `Main.java` (28 lines), `main.go` (19 lines) and `app.ts` (11 lines);
- `.env`, a made-up line (`DEMO_VALUE=not-a-real-secret`), to be refused;
- `logo.png`, 20 bytes: a PNG header and NULs;
- `mock.jsonl`, the mock run: a user message, then Claude Code's Read of
  `main.go` at offset 10 and its Read of `.env`, then the turn's end.

`LOCALAPPDATA`, `USERPROFILE` and `HOME` pointed at temporary folders, and
`PATH` held `C:\Windows\System32` only, so no agent CLI could be found or
started. The owner's clipboard was saved before the run and restored after
it.

```text
A file viewer in the terminal - 29 checks, maximised (156x44): 29 PASS; 80x24: 29 PASS
PASS No agent CLI on PATH; the project set up; the mock dashboard; the size (maximised, or 24 rows of at most 80 columns)
PASS F3 from the input: "Quick open", "5 files"; listed exactly Main.java, app.ts, logo.png, main.go, mock.jsonl (.env absent); typing "env": "no file matches"
PASS Typing "java": "▸ Main.java", "1 of 1"; Enter: "Java · 28 lines", "Main.java · quick open", lines 1 and 6 numbered; the dashboard's header still around the window
PASS Colours inside the window (screenshot, colours over 40 px): Java 75 (65 at 80x24), keywords A78BFA, strings 86EFAC, classes and functions 93C5FD, comments 6B7280, text E5E7EB, cursor row 2E3247
PASS g 20 Enter: "line 20/28"
PASS / class Enter: "match 1 of 2", line 6 on screen; n: "match 2 of 2"; n: "match 1 of 2" (wrapped)
PASS f: the window's top edge on row 1, the dashboard hidden; then a drag over line 26 copied exactly "int answer = 42; // the line copied with the mouse"
PASS Esc cleared the selection, the viewer open; f: the window over the dashboard again; Esc: closed, the dashboard
PASS ↑ to the .env Read entry, F3: "a credential file: not opened"
PASS ↑ to the main.go Read entry, F3: "Go · 19 lines", "main.go · read by Read", "line 10/19"; colours (64 maximised, 58 at 80x24; numbers FB923C)
PASS The quick open, "app.ts" Enter: "TypeScript · 11 lines"; colours (57); "logo" Enter: "binary file, not shown · 20 bytes"
PASS Ctrl+Q: the window closed, no dashboard process left; no new claude or codex process; the project's six files unchanged in size, nothing written
```

The screenshots were also looked at: the code is coloured as the plan
says, the cursor line on zone A's cursor background, and the window's
frame in the title colour over the dashboard. At 80x24 the window is 76x22
and the footer takes its short form (`↑↓ PgUp PgDn · / search · n N · g
line · f full · Esc close`).

The block above is maximised run 3 and the one 80x24 run. The two earlier
maximised runs failed through faults of the script, not of the dashboard:

- Run 1, 7 FAIL, stopped at the Go colour check. The script built the mock
  file's lines in a PowerShell array where `+` bound to the array, so the
  file was one line; the dashboard reported "mock.jsonl: line 1: invalid
  character '{' after top-level value" and no tool entry existed. The F3 on
  the .env and main.go entries therefore opened the quick open. The listing
  check read dashboard text beside the window; the three search checks also
  required "line N/28", which the footer replaces with "match i of n" while
  a search is active; the "Esc clears the selection" check required a line
  number for the same reason. The colour count, finding no window in a
  frame it could not parse, threw and stopped the script before Ctrl+Q; the
  window left open was closed before the next run. The checks before it
  (the quick open, open, colours, g, f, the mouse copy, f back, Esc) gave
  the results recorded here again.
- Run 2, 28 PASS and 1 FAIL, "credential not listed". The listing, now
  read inside the window's columns, counted the filter's input row
  (`› ▌`) as a file; the five files were right and .env absent. The input
  row is now left out.

## Not verified

- A real symbolic link in `ReadFile` and `ListFiles`. This account has no
  SeCreateSymbolicLink unelevated; the symbolic-link cases of TestReadFile
  and TestListFiles are skipped (logged, the tests pass), and TestReadFileJunction runs the same path through a directory
  junction (resolved, then refused as outside the project).
- The viewer opened from a live agent CLI's tool entry. No live run was in
  plan 17's scope; the mock replays Claude Code's recorded Read event shape,
  and TestFileFromEntry covers Edit and Codex's fileChange.

## Amendment 12, the Explorer (2026-10-18)

The owner added the Explorer to plan 17 before approving it (amendment 12
in [`doc/brief.md`](brief.md)): F3 with no file in front opens the project
as a tree with git's markers, Enter shows a file in the viewer, Esc goes
back one level, and F3 closes every window. Its criteria, 5.1 to 5.13, are
in [`plan.md`, Plan 17](../plan.md#plan-17---a-file-viewer-in-the-terminal),
written before the work started. The block above stays as it was reported.
The block below is the summary of this item, and it is not to be rewritten
either.

```text
Evaluation - Plan 17, amendment 12, the Explorer - 13 criteria: 13 PASS
PASS 5.1 ListDir: folders first, then files, sorted without case; .git, credential names and links left out; outside the project and a link pointing outside refused; the lowered limit cuts and says so (TestListDir)
PASS 5.2 Status: M, A, R (both paths), U, !, ignored files and folders, untracked folders, a project in a subfolder of its repository parsed; a real temporary repository gives the expected markers, .git/index not rewritten; no repository: "not a git repository" (TestParseStatus, TestGitStatus)
PASS 5.3 F3 with no file in front opens the Explorer; the root read, then a folder on → or Enter; Enter on a file shows it in the viewer (TestExplorerOpen; TestExplorerRows for "reading…", a reason, "empty folder", "first N entries")
PASS 5.4 ↑↓ PgUp PgDn Home End, → ← Enter and / as the design says; no key the Explorer takes changes the draft (TestExplorerKeys, TestExplorerCursor)
PASS 5.5 Esc: viewer → Explorer on the file, its folders open (also from a tool entry); quick open → Explorer; Explorer → the cursor's folder closed, then closed at the top level; F3 closes the viewer, the quick open and the Explorer in one key (TestExplorerBack)
PASS 5.6 M yellow; U, A, R green; ! and D red; ● yellow on each folder above a change; ignored entries dimmed with no marker, on the cursor row too; no repository: the note (TestExplorerMarks, app and render)
PASS 5.7 A listing or status for an older opening dropped; reopening keeps the open folders and the cursor (TestExplorerStale)
PASS 5.8 The wheel moves the cursor; a click selects a row, a click on the cursor row opens it (TestExplorerMouse)
PASS 5.9 Golden frames explorer at 79x24, 80x24, 120x40, 200x60 (4 new, reviewed); the 16 file frames' footers read "Esc back · F3 close"; no tracked frame changed (git diff --stat testdata/frames: empty); the overlay lists the Explorer's keys (TestKeyOverlayRows, 45 rows); every footer fits at 80 to 200 columns (TestFileFooters, TestFootersFit)
PASS 5.10 brief.md (amendment 12), AGENTS.md, architecture.md, ui-design.md, reliability.md and evaluation.md say what was built
PASS 5.11 go build ./... and go vet ./... (also -tags live, -tags timing) exit 0; gofmt -l . prints nothing; go test ./...: 336 top-level PASS (817 with subtests), 1 SKIP (TestDiskListLink), 0 FAIL, in 14 packages (324 before, +12); timing 4/4 PASS, p95 TestKeystrokeToFrame 1.02 ms, TestDraftKeystrokes 2.02 ms, TestLongSession delta-to-frame 3.08 ms (target 16 ms); the binary 14,171,136 bytes (14,086,144 before)
PASS 5.12 Terminal runs on a mock run in a temporary git repository: maximised (156x44) 39 checks PASS, 0 FAIL; 80x24 the same 39 checks PASS, 0 FAIL; no agent CLI started; .git/index, every file's size and time, and git status unchanged
PASS 5.13 The changed and new files carry no CR, no BOM and no NUL; this summary written here and in plan.md's progress list
Not verified: the ! (conflict) and D markers live, and "not a git repository" and "git not found" live (TestParseStatus, TestGitStatus and TestExplorerMarks cover them). The diff of the 16 file frames against their first version (untracked since then; their footers were checked). The Explorer from a live agent CLI (the runs replay a mock). A real symbolic link in ListDir (as in 1.2).
Verdict: DONE
```

### The terminal runs

The script of the first runs, extended. The binary was built from the
working copy. Each run had its own repository, `demo` under `%TEMP%`,
outside this repository, made with a throwaway identity
(`p17 <p17@example.invalid>`). Its first commit held `.gitignore`
(`build/`), `main.go`, `src/Main.java`, `src/util/Strings.java`,
`docs/old.md`, and `mock.jsonl`, the mock run: a user message, then Claude
Code's Read of `src\Main.java`, then the turn's end. The run then changed it:

- `main.go` and `src/util/Strings.java` modified (`M`);
- `src/New.java` added and staged (`A`);
- `docs/old.md` moved with `git mv` to `docs/guide.md` (`R`);
- `notes.txt` untracked (`U`);
- `build/out.txt`, ignored;
- `.env`, a made-up line, untracked, to be hidden.

`PATH` held `C:\Windows\System32` and Git's `cmd` folder only, so git could
run and no agent CLI could be found. `LOCALAPPDATA`, `USERPROFILE` and
`HOME` pointed at temporary folders. The claude and codex processes running
before the runs belonged to VS Code, not to a terminal, so none was closed.

The checks, the same at both sizes:

- The tree at the top level: `build`, `docs ●`, `src ●`, `.gitignore`,
  `main.go M`, `mock.jsonl`, `notes.txt U`; no `.env`, no `.git`.
- The colours, counted in the screenshots' pixels: `M` #FBBF24 (40 px),
  `U` #4ADE80 (31), `src`'s ● #FBBF24 (71), `R` #4ADE80 (37), `A` #4ADE80
  (35), the ignored `build` #6B7280 (151, none in the folder colour), `src`
  in the folder colour #C4B5FD (115).
- The keys: → opens `docs` and reaches `guide.md R`, ← back, ← closes;
  Enter opens `src`, → reaches `src/util/Strings.java M`; Enter shows it
  ("Java · 8 lines"); Esc back to the Explorer on it, then `src/util/`
  closed, then `src/`, then the Explorer closed; F3 reopens it on `src/`;
  `/` opens the quick open ("9 files") and Esc returns; End reaches
  `notes.txt`, Enter opens it, F3 closes everything.
- From the Read entry, F3 shows `Main.java` ("read by Read"); Esc gives the
  Explorer on `src/Main.java` with `src` open; two more Esc close it.
- The mouse: a click selects `main.go`, a second click opens it ("Go · 5
  lines"); the wheel up reaches `docs/`, down `main.go`.
- Ctrl+Q leaves no process; no claude or codex process started.
- git wrote nothing: the SHA-256 of `.git/index`, the size and time of the
  45 files in the project and `.git`, and `git status` were the same after
  as before.

The screenshots of the tree and the viewer were looked at: the markers in
their colours at the right edge, the ignored folder dimmed, the cursor row
on zone A's cursor background. At 80x24 the footers take their short forms
(`↑↓ · → open · ← close · Enter view · / find · Esc back · F3 close`).

The work found one fault before the runs, in the code: a folder left open
above a file the viewer went back to was never read, so its entries did not
appear. The fix reads every open folder not yet read when its parent's
listing arrives; TestExplorerBack covers it.

## Live and simulated checks (2026-10-18)

The owner asked for the items left not verified above to be tested, and
approved the use of their quota for it. The blocks above stay as they were
reported; this section adds what the new runs showed. Each run was in a
temporary project under `%TEMP%`, outside this repository, maximised
(156x44) first, then at 80x24, with the binary built from the working copy.

```text
Live and simulated checks - Plan 17 and amendment 12 - 5 items: 3 verified, 2 not verified
PASS The ! marker and the notes, simulated in a real terminal: 27 checks PASS maximised, 27 PASS at 80x24, no agent CLI started
PASS The Explorer from live Claude Code 2.1.302 (claude-haiku-5-5): 24 checks PASS maximised, the same at 80x24; $0.00436 and $0.00433
PASS The Explorer from live Codex 0.163.0 (gpt-5.6-luna): every check PASS maximised and at 80x24
Not verified: the D marker on a row (a deleted file has no row; see below). The 16 file frames against their first version (no copy of it exists). A real symbolic link in ListDir and the reading (this account cannot create one).
```

### The ! marker and the notes

Three repositories, git only on `PATH`, no agent CLI.

- A conflict, made with a merge of two branches changing `conflict.txt`:
  the top level reads `old ●`, `conflict.txt !`, `mock.jsonl`; `!` in red
  #F87171 (11 px), `●` #FBBF24 (71). `old` opened reads `keep.txt`,
  `tracked.txt U` (`git rm --cached`, green #4ADE80, 31 px). The conflict
  file shows in the viewer with its `<<<<<<<` and `>>>>>>>` lines; Esc goes
  back to the Explorer on it. The SHA-256 of `.git/index`, the size and time
  of the 51 files and `git status` were the same after as before.
- A plain folder: the note "not a git repository" in grey (456 px, no red),
  the rows `src`, `a.txt`, `mock.jsonl` with no marker.
- A repository with git left off `PATH`: the note "git not found", the rows
  `old`, `conflict.txt`, `mock.jsonl` with no marker.

The `D` marker cannot show on a row: the Explorer lists what is on the disk,
so a file deleted from the working tree has no row, and only the folders
above it get `●`. VS Code's Explorer behaves the same way. TestParseStatus
and TestExplorerMarks cover `D` for the status and the colour.

### The Explorer from a live agent CLI

The configuration check passed before and after every run (only the
sensitive keys not compared). No claude or codex process was left after
Ctrl+Q; the claude and codex processes running in VS Code were not touched.

Claude Code, approvals Manual. The project: `README.md` (committed, then
modified) and `src/Main.java`. The prompt asked for a Read of
`src/Main.java`, a Write of `notes/live.txt` containing `live`, and the reply
`done`. At both sizes:

- the Explorer before the turn: `src`, `README.md M`;
- the Write allowed on its card, the turn ended with success and `done`;
- the Explorer after: `notes U`, `src`, `README.md M`; `U` #4ADE80 (31 px),
  `M` #FBBF24 (40);
- the Write entry, one ↑ from the input, F3: "notes/live.txt · written by
  Write", line 1 `live`; Esc: the Explorer on `notes/live.txt`, rows
  `notes U`, `live.txt U`, `src`, `README.md M`;
- the Read entry, two ↑, F3: "src/Main.java · read by Read", showing
  `public class Main`; Esc: the Explorer on `src/Main.java`;
- the project's status after: ` M README.md`, `?? notes/live.txt`.

Codex, approvals Manual, read-only sandbox, through the wrapper of plan 16
that sets the model with `-c`. The prompt asked for `notes/live.txt`
containing `live` by editing files only, then the reply `done`. At both
sizes: one approval card, allowed, none denied; the items `userMessage`,
`reasoning`, `fileChange`, `agentMessage`, then `turn/completed`; the
thread's working directory the project; the Explorer after as for Claude
Code, with the same colours; the fileChange entry, one ↑, F3: "notes/live.txt
· changed by fileChange", line 1 marked as changed (`1▌ live`); Esc: the
Explorer on `notes/live.txt`.

The quota used: about $0.019 of Claude Code (four runs, two of them the
first attempts below) and five short Codex turns.

### Faults found, all in the test scripts

None was in the application. A mock with the same Read and Write as the
live run, driven by the same keys, opened the viewer as expected, which
settled the first one.

- The entry checks sent no key: a loop variable `$k` hid the key table `$K`,
  since PowerShell's names ignore case. Renamed.
- The colours counted 0 pixels: the launch inherited `NO_COLOR=1` from the
  shell. The launch now clears it.
- At 80x24 the Codex card wraps its path, so the script denied both cards
  and no file was written. The script now compares the path with the card's
  spaces and borders removed.
- The line check expected a space after `1`, where a fileChange puts its
  mark; it now expects the mark, which makes it stricter.

### Still not verified

- **A real symbolic link** in `ListDir` and the reading: this account lacks
  the right to create one, so TestDiskListLink skips and a junction is
  tested. A run from an elevated terminal would settle it:
  `go test ./internal/input -run 'TestListDir|TestReadFile|TestListFiles' -v`.
- **The 16 file frames against their first version**: they were untracked
  when their footers changed, so no copy of the first version exists. The
  current frames were reviewed and their footers checked (5.9).
- **The `D` marker on a row**: not reachable, as above; listing deleted
  files would be a design change for the owner to decide.

## Item 6, deleted files in the Explorer (2026-10-18)

The live checks above showed that `D` could never reach a row, since git
reports a deleted file and the disk no longer lists it. The owner asked for
deleted files to be listed (item 6): a file git versions and the working
tree no longer has keeps a row, so that a deletion, by the owner or by an
agent, shows where it happened; a file git never versioned leaves no trace.
Its criteria, 6.1 to 6.8, are in
[`plan.md`, Plan 17](../plan.md#plan-17---a-file-viewer-in-the-terminal),
written before the work started. The blocks above stay as they were
reported; the line "Not verified: the D marker on a row" above is settled by
this item.

```text
Evaluation - Plan 17, item 6, deleted files in the Explorer - 8 criteria: 8 PASS
PASS 6.1 Each deleted path a row under its folder, its name and D red, in the folder's order; a deleted folder rebuilt from the paths under it, a folder row with D; a name also on the disk listed once, from the disk; no deleted folder added under a cut listing (TestExplorerDeleted, render)
PASS 6.2 A folder empty on the disk lists its deleted paths, not "empty folder"; the folders above a deletion keep their ● (TestExplorerDeleted, render)
PASS 6.3 The path row reads "<path> · deleted, not on disk"; Enter and a click on a deleted file open nothing; → opens a deleted folder without reading the disk, → again enters it, Esc closes it; the Explorer opened again reads no deleted folder; a folder on the disk lists its deleted file among its entries (TestExplorerDeleted, app)
PASS 6.4 A real temporary repository with a top-level file and a whole folder deleted (two files, one in a subfolder) gives D for each file (TestGitStatus); the golden frames explorer show a deleted file and a deleted folder at 80x24, 120x40, 200x60 (79x24 is the "too small" screen), reviewed; with the old fixture only those 3 fail, so no other frame depends on the change; no tracked frame changed
PASS 6.5 ui-design.md (Explorer: frame sample, Deleted files, the D marker, Enter) and architecture.md (Rendering) say what was built
PASS 6.6 go build ./... and go vet ./... (also -tags live, -tags timing) exit 0; gofmt -l . prints nothing; go test ./...: 338 top-level PASS (819 with subtests), 1 SKIP (TestDiskListLink), 0 FAIL (336 before, +2); timing 4/4 PASS, p95 TestKeystrokeToFrame 1.10 ms, TestDraftKeystrokes 3.13 ms, TestLongSession delta-to-frame 5.26 ms (target 16 ms); the binary 14,179,328 bytes
PASS 6.7 Terminal runs on a mock run in a temporary git repository (top.txt, lib/ whole and src/lost.go deleted): maximised (156x44) 21 checks PASS, 0 FAIL; 80x24 the same 21 checks PASS, 0 FAIL; the deleted names and their D #F87171 (pixels counted, 0 on a name not deleted), src's ● #FBBF24; Enter on top.txt and src/lost.go opens nothing; lib and lib/deep open from git's paths, Esc closes each; no agent CLI started; .git/index, every file's size and time, and git status unchanged
PASS 6.8 The changed and new files carry no CR, no BOM and no NUL; this summary written here and in plan.md's progress list
Not verified: none for this item.
Verdict: DONE
```

The script, `p17del.ps1`, follows the `p17git` runs above: a Windows
Terminal window driven by UI Automation, the screen read as text, each
colour counted in a screenshot. The repository: `README.md`, `top.txt`,
`lib/a.go`, `lib/deep/b.go`, `src/keep.go`, `src/lost.go` and the mock
committed, then `top.txt`, `lib/` and `src/lost.go` deleted from the disk,
git status ` D lib/a.go |  D lib/deep/b.go |  D src/lost.go |  D top.txt`.
The rows, after F3: `lib[D], src[●], mock.jsonl, README.md, top.txt[D]`;
`lib` and then `deep` opened: `lib[D], deep[D], b.go[D], a.go[D], …`; `src`
opened: `keep.go, lost.go[D]`. On a deleted row the footer still reads
"Enter view file", as on any file; the path row says why Enter does
nothing.

## A real symbolic link (2026-10-18)

Unelevated, this account cannot create a symbolic link ("A required privilege
is not held by the client"), so `TestListDir`, `TestReadFile` and
`TestListFiles` pass with their link cases skipped. As in plan 16, the owner
accepted one UAC prompt. The test binary was built unelevated from the
working copy (`go test -c ./internal/input`). An elevated PowerShell then ran
only that binary, with `-test.run '^(TestListDir|TestReadFile|TestListFiles)$'
-test.v -test.count=1`. Nothing else ran elevated, and the tests wrote only
in their own temporary folders.

```text
Evaluation - Plan 17, a real symbolic link - 3 criteria: 3 PASS
PASS Elevated: the runner reported elevated: True, exit 0
PASS TestListDir, TestReadFile, TestListFiles elevated: --- PASS each, PASS, exit 0
PASS Every link case ran: no "symbolic link(s) not made" line in the elevated log (unelevated, the same binary logs it once in each test)
Not verified: none for this item.
Verdict: DONE
```

The link cases are then checked on real links:
- `ListDir` leaves out a link to a file outside the project and a link to a
  folder outside it.
- `ReadFile` refuses a link to a file outside the project ("outside the
  project: link.go"). It reads a link to a file inside the project, under the
  link's own name. It refuses a link with an ordinary name to `.env` as a
  credential file.
- `ListFiles` leaves out both links.

This settles the real symbolic link that items 1, 5 and 6 left not verified.
In an unelevated `go test ./...` the link cases still skip.

## The 16 file frames against their first version (2026-10-18)

The frames were never committed, but their first version was found in two
independent places.

- **Git's object store.** Some tool outside the application stores changed
  files of this repository as unreachable blobs. It also does so after an
  edit of documentation only, and it stores documents too. Its copy of
  19:29:45 UTC holds all 16 frames with their first footers.
- **The development session's transcript.** It printed seven of the frames
  in full, with the first three lines of the 79x24 frame, while the first
  version was reviewed (18:12 to 18:18 UTC).

`-update` first wrote the 16 frames at 18:12 UTC. It next rewrote them at
19:30:13 UTC, after the snapshot. No other `-update` ran in between, in the
session or in a subagent. The snapshot is therefore the first version.

```text
Evaluation - Plan 17, the 16 file frames against their first version - 4 criteria: 4 PASS
PASS A first version of all 16: the blobs of 19:29:45 UTC, the last state before the first rewrite (19:30:13 UTC) of the frames -update wrote at 18:12 UTC
PASS Cross-checked: the transcript's 6 frames printed in full, file-79x24's first 3 lines and file-search-80x24's last 5 are byte for byte the snapshot's
PASS The changes: each of the 12 frames at 80x24, 120x40 and 200x60 differs in one line, its footer; the four 79x24 frames are unchanged (blob 55296a5, the "Terminal too small" frame already committed under other names)
PASS Each footer change is the Esc and F3 change asked for (amendment 12): "Esc close" now reads "Esc back · F3 close", with key hints shortened where the longer text no longer fits
Not verified: none for this item.
Verdict: DONE
```

The hints shortened to fit:
- `file-80x24` and `file-full-80x24` drop "PgUp PgDn".
- `file-120x40` turns "f full screen" into "f full".
- `file-search-120x40` drops "←→ shift" and "f full screen" turns into
  "f full", beside "match 10 of 17".
- Each frame at 200x60 keeps every hint.

Every hint is still in the F1 key overlay. The first versions are blobs
`c9ccdfda`, `669ad97b`, `e25468ea` (file), `a2a31b3c`, `dd164407`, `eaa23c89`
(search), `354d1ff5`, `5929884d`, `332c0d65` (quick open), `a451b89a`,
`003aa380` and `963f5d9d` (full screen), at 80x24, 120x40 and 200x60. Being
unreachable, they last until git prunes them. This settles the 16 file frames
that items 5 and 6 left not verified.

## F3 in the dashboard footer (2026-10-18)

The owner asked for F3 in the dashboard's key-hint line, and approved the
proposal: "F3 Explorer" after the F2 key, since F3 with no file in front
opens the Explorer. When the whole line does not fit, it leaves out
"Ctrl+J newline", which B's input shows just above it, so that "Ctrl+Q quit"
stays visible. The proposal said the 80-column footer would not change, but
it is a separate, shorter line that fits whole, so it too now has "F3
Explorer". At 160 columns and up, "Ctrl+J newline" is also left out while
the line is too long; before this change, the end of that line was already
cut from 160 to 176 columns.

```text
Evaluation - Plan 17, F3 in the dashboard footer - 6 criteria: 6 PASS
PASS "F3 Explorer" in the footer at every layout: compact (80, 99), standard (100 to 159), wide (160 to 200); TestDashboardF3
PASS "Ctrl+Q quit" whole at 120x40, and from 109 columns (standard) and 174 (wide); "Ctrl+J newline" left out only below 126 and 191; at 80 the compact line fits whole; TestDashboardF3, which fails at 9 widths without the rule
PASS On an entry that names a file the footer says "F3 file", not "F3 Explorer"; on another entry, neither; TestDashboardF3 at 80, 120 and 200 columns
PASS Golden frames rewritten: 29 frames changed, each in one line, its last, the footer; the only changes are the 3 footers (14 at 120 columns, 8 at 200, 7 at 80)
PASS ui-design.md: the three footer samples, and the rule under the standard layout
PASS go build, go vet clean; gofmt -l prints nothing; go test ./... -count=1: 339 PASS, 1 SKIP, 0 FAIL
Not verified: the footer in a real terminal (the golden frames are the rendered text).
Verdict: DONE
```
