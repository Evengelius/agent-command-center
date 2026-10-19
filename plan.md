# Current Plan

The active plan with its progress, the owner's standing decisions, and one
entry per closed plan. Each plan comes from `doc/brief.md` and its
amendments, ends with the evaluation summary of `doc/evaluation.md`, and
waits for the owner's approval before the next one starts. This file keeps
the current state only (the owner, 2026-10-18): a closed plan's full text,
criteria and progress log are in git (`git show 8888f59:plan.md` for plans 1
to 13, `git show 84d2ad8:plan.md` for plan 14, `git show b047434:plan.md`
for plan 15, `git show becfb34:plan.md` for plan 16), and its results in its
evaluation file.

## The owner's standing decisions

Decisions of closed plans that still apply; `AGENTS.md` and
`doc/evaluation.md` hold the others.

- **Live runs.** Each one is proposed first with its scope, model, expected
  usage and stopping conditions, and runs only once approved (plan 3,
  2026-10-16); Claude runs use Haiku as a per-process override.
- **The configuration check is hash-only**: credential contents are never
  read, and a file that cannot be classified safely is "changed, not
  classified", never reported unchanged (2026-10-16). No parser runs over a
  file that may hold credentials (the mixed-state tokenizer was withdrawn,
  2026-10-16); the one exception is confcheck's key-level comparison of
  `~/.claude.json`, sensitive keys skipped (plan 5).
- **Measured, not assumed.** Performance figures are targets until a test
  measures them; a documented design is never presented as verified runtime
  behaviour (2026-10-16).
- **The integration.** If driving a CLI without a terminal falls short, the
  limit is documented and the smallest adjustment proposed; the architecture
  never changes silently (2026-10-16).
- **Drafts and session data** survive cancellation, crashes and recovery; no
  conversation data is stored beyond the draft, and nothing sensitive goes
  into a log (2026-10-16).

## Recurring

Tasks tagged **[Recurring]** repeat on an outside event: they are never
closed, never listed as remaining, and each occurrence is reported in the
summary format (the owner, 2026-10-18).

- **[Recurring] After a CLI update.** At session start, `claude --version`
  and `codex --version` are compared with the "Last probed" line of
  `doc/research/probes/README.md`; a newer one is told to the owner with the
  re-check proposed (`doc/evaluation.md`, After a CLI update), which runs
  only once the owner approves it. The dashboard's project bar shows
  "newer than tested" meanwhile. Last occurrence: Claude Code 2.1.303 on
  2026-10-19, approved, every check PASS (P303a to P303d in
  `doc/research/probes/README.md`); it found that plan 14's shape check
  counted an interrupted tool use as unrecognised, fixed the same day at the
  owner's request (P303e).

## Nice to have

Items tagged **[Nice to have]** are future improvements: none is built, and
none is listed as remaining, until the owner asks for one in a new plan
(the owner, 2026-10-18). Amendment 4 names ten "future improvements"
(`doc/brief.md`, Amendment 4, "For this phase, do NOT implement"). Five
remain below. Recent projects, favourites and a project browser were plan 15
(amendment 9); git worktree management was plan 16 (amendment 10). The
automatic VS Code integration was replaced by the owner with a file viewer in
the terminal, plan 17 (amendment 11).

- **[Nice to have]** Advanced model configuration.
- **[Nice to have]** Cross-provider agent collaboration.
- **[Nice to have]** Provider switching during an active session.
- **[Nice to have]** Complex session profiles.
- **[Nice to have]** Advanced orchestration configuration.
- **[Nice to have]** Google Antigravity as an agent CLI, removed from the
  MVP by amendment 5; the provider interface stays open to it.

## Plan 17 - A file viewer in the terminal

**Approved by the owner on 2026-10-18; amendment 12, the Explorer, added
to it the same day, and amendment 13, the draft window, on 2026-10-19.
Done and approved by the owner on 2026-10-19**: every evaluation in
`doc/plan17-evaluation.md` PASS, nothing left not verified. The
owner's request
(2026-10-18, `doc/brief.md`, amendment 11): instead of the VS Code
integration, a file shown in a window floating over the dashboard, with
syntax highlighting of any language. The choices approved there: about 80%
of the screen, the dashboard updating behind it; Chroma for the
highlighting; opened from a file the agent read or edited, or by a quick
search by name; scroll, `/` search, `g` line, `f` full screen, mouse copy,
Esc; read-only; only project files, credential files refused, binary files
not shown, very large files coloured around the view.

The one new dependency is `github.com/alecthomas/chroma/v2` v2.27.0 (MIT),
which brings `github.com/dlclark/regexp2/v2` v2.2.1 (MIT); both approved by
the owner on 2026-10-18. Nothing is written anywhere: the viewer only reads.

Design:

- **Reading a file** (`internal/input/file.go`, the input part). It runs in
  a command, never in the update loop, with a generation number so a stale
  result is dropped.
  - A relative path is taken from the project directory.
  - **Refused before the file is opened**, with the reason shown in the
    window:
    - outside the project, after both paths' symbolic links are resolved:
      "outside the project: <path>";
    - a credential file, by its name only: ".credentials.json",
      "auth.json", "oauth_creds.json", ".git-credentials", ".netrc",
      "_netrc", ".npmrc", ".pypirc", "id_rsa", "id_dsa", "id_ecdsa",
      "id_ed25519", ".env" and ".env.*" (except ".env.example",
      ".env.sample", ".env.template"); the extensions ".pem", ".key",
      ".p12", ".pfx", ".jks", ".keystore", ".kdbx"; and a name whose stem
      is "credentials", "secret", "secrets", "token" or "tokens" with no
      extension or ".json", ".yaml", ".yml", ".toml", ".txt", ".ini",
      ".conf" or ".cfg": "a credential file: not opened";
    - a directory: "a directory, not a file";
    - over 32 MiB: "too large to show: <size> (32 MiB at most)".
  - A NUL byte in the first 8 KiB makes it binary: "binary file, not shown"
    and its size.
  - Lines split at LF, a CR before it dropped; invalid UTF-8 shows as U+FFFD.
- **Listing the project's files** for the quick open, in a command: every
  file under the project, relative paths sorted, without `.git` folders,
  credential files, or anything behind a symbolic link; at most 100,000,
  then "first 100,000 files".
- **Highlighting** (`internal/render/syntax.go`; Chroma is pure
  computation, so the rendering part may use it).
  - The language comes from the file name, else from Chroma's analysis of
    the text, else "Plain text".
  - Chroma's token types are coloured from the palette, not from a Chroma
    theme: keywords violet and bold, strings green, numbers orange,
    comments grey and italic, functions and classes light blue, built-ins
    and types blue, a diff's added lines green and removed lines red,
    headings in the title colour, anything else in the text colour. Chroma's
    error tokens are plain text. The 16-colour palette maps the same way.
  - It runs in a command, never while drawing. A file up to 1 MiB is
    highlighted whole. A larger one is highlighted 1000 lines on either side
    of the view, again when the view leaves that range; the window says
    "coloured around the view", since a comment or string that starts
    above the range may be coloured wrongly. Until the colours arrive the
    lines show uncoloured, "colouring…" in the header.
- **The window** (`internal/render/file.go`).
  - Floating: `max(W*4/5, min(W-4, 84))` columns by `max(H*4/5, min(H-2,
    22))` rows, centred over the dashboard, which is drawn and updated
    behind it: 76x22 at 80x24, 96x32 at 120x40, 160x48 at 200x60. `f`
    toggles full screen.
  - A rounded frame in the title colour. The top edge has the file's name
    on the left and "<language> · <N> lines" on the right; the first row,
    the path shortened as elsewhere and what opened it ("read by Read",
    "edited by Edit"…).
  - Each line: its number right-aligned in grey, then the code. Tabs to the
    next multiple of 4; long lines cut with "…", ←→ shift the view by 8
    columns.
  - The cursor line has zone A's cursor row background. The lines the agent
    edited are marked: their numbers in the attention colour and "▌" after
    them.
  - The footer: the keys on the left; "line <cursor>/<N>" on the right, or
    the search or go-to-line field while typing it. With an approval
    pending: "◆ approval waiting · Esc, then F2".
- **Opening** (`F3`, on the dashboard).
  - On zone B's entry cursor, or in an entry's full view, on a tool entry
    that names a file: that file. The entry hint adds "F3 file".
    - Claude Code's Read: at its `offset`, nothing marked.
    - Edit and MultiEdit: the lines of the first `new_string` found in the
      file (else the first `old_string`), marked; Write: line 1.
    - Codex's fileChange: its first change's path, the first added lines
      of its diff's first hunk marked.
    - Any other tool with `file_path`, `notebook_path` or `path`: line 1.
  - Elsewhere, or on an entry that names no file: the quick open, a window
    of the same size. Typing filters (case ignored; the letters in order;
    a name match first, then a path match, then the shortest path); ↑↓
    PgUp PgDn select; Enter opens; Esc closes.
- **Keys in the window.** ↑↓ move the cursor line, PgUp PgDn by a page,
  Home End to the first and last line, ←→ shift sideways; `/` types a
  search (case ignored unless it has a capital), Enter finds the next match,
  `n` and `N` the next and previous, matches shown on an amber background
  with "match <i> of <n>"; `g` types a line number, Enter goes there; `f`
  full screen; Esc cancels a field, then clears a selection, then closes.
  F1, F4, F12, Ctrl+C and Ctrl+Q keep their meaning; F2 closes the window
  and goes to the approval; everything else is taken by the window. The
  search runs in a command, at most 10,000 matches.
- **Mouse.** Capture stays on: a drag selects in the window only, and the
  release copies the file's text it covers, as the other panels do; the
  wheel over the window scrolls it. The dashboard behind takes no click.

**Amendment 12, the Explorer** (the owner, 2026-10-18, `doc/brief.md`): F3
opens the project as a tree, as VS Code's Explorer does; the rules above
stand except where this changes them.

- **Opening.** F3 on an entry or a request naming a file opens that file,
  as before; anywhere else it opens the Explorer, a window of the viewer's
  size titled "Explorer". It opens as it was left in the session (its open
  folders and cursor), every open folder read again.
- **Reading a folder** (`input.ListDir`, in a command, a generation number
  dropping a stale result): only a folder inside the project, its links
  resolved; its folders, then its files, each sorted without case; without
  `.git`, credential files (by name, as in the listing) or links; at most
  10,000 entries, then "first 10,000 entries". A folder is read only when it
  is opened, so a large `node_modules` costs nothing until then.
- **Git's markers** (`input.GitCLI.Status`, one call each time the Explorer
  opens, in a command; `git --no-optional-locks status --porcelain=v1 -z
  --untracked-files=normal --ignored -- .`, so it writes nothing): `M`
  modified, `A` added, `R` renamed, `U` untracked, `!` in conflict, at the
  row's right; "●" on a folder holding any of these or a deleted file; a
  file or folder git ignores, and all under it, dimmed. Outside a
  repository, or without git, no marker, and the path row says "not a git
  repository" or "git not found".
- **The window.** The top edge: "Explorer", and on the right the project's
  name and "<N> items", the rows the tree shows. The first row: the cursor's
  path in the project. Then the rows: two cells of indent a level, "▸" or
  "▾" before a folder, folders in the title colour and bold, files in the
  text colour, the marker on the right, the cursor row on zone A's cursor
  background. Under a folder being read, "reading…"; an empty one, "empty
  folder"; one that cannot be read, its reason in the attention colour. The
  footer: the keys, and "<i> of <N>".
- **Keys in the Explorer.** ↑↓ PgUp PgDn Home End move; → opens a folder,
  or on an open one goes to its first entry; ← closes an open folder, else
  goes to the parent folder; Enter opens or closes a folder, or shows a file
  in the viewer; `/` opens the quick open; F1, F4, F12, Ctrl+C and Ctrl+Q
  keep their meaning, F2 closes and goes to the approval; everything else is
  taken. Mouse: the wheel moves the cursor; a click puts the cursor on a
  row, a click on the cursor row is Enter.
- **Esc goes back one level, F3 closes** (in place of "Esc closes" above):
  - in the viewer, Esc cancels a field, then clears a selection, then goes
    to the Explorer with the file under the cursor, its folders opened (a
    file opened from an entry too);
  - in the quick open, Esc goes back to the Explorer;
  - in the Explorer, Esc closes the folder the cursor is in and puts the
    cursor on it; at the top level, Esc closes the Explorer;
  - F3 in any of the three closes them all, back to the dashboard.

1. **Reading and listing.** `internal/input/file.go`; tests on temporary
   directories.
2. **Highlighting.** `internal/render/syntax.go`; the palette's syntax
   styles.
3. **The window and the quick open.** `internal/render/file.go`,
   `internal/app/file.go`; keys, mouse, the key overlay, the footers.
4. **Documents and the full check.** `THIRD_PARTY.md`, AGENTS.md,
   architecture.md, ui-design.md, reliability.md, evaluation.md; the
   checks, the terminal runs, `doc/plan17-evaluation.md`.
5. **The Explorer** (amendment 12). `input.ListDir`, `GitCLI.Status`, the
   Explorer window and its keys, Esc and F3 changed in the viewer and the
   quick open; the documents, the full check again, its results added to
   `doc/plan17-evaluation.md` beside the first ones, which stand.

### Criteria

Item 1, reading and listing (TestReadFile, TestCredentialName, TestListFiles).

- 1.1 A file inside the project reads, by absolute and by relative path;
  CRLF lines come without CR; invalid UTF-8 shows U+FFFD.
- 1.2 Refused, each with its reason above: a path outside the project, a
  symbolic link inside it pointing outside (skipped where links cannot be
  made), a directory, a file over the limit (a lowered limit in the test).
- 1.3 Every credential name listed above is refused and a credential file
  that exists is not opened (TestCredentialName covers the names; the check
  comes before the open in `ReadFile`); "token.go", "secrets.go",
  ".env.example" and "auth.go" are not refused.
- 1.4 A file with a NUL in its first 8 KiB is binary.
- 1.5 The listing is sorted, leaves out `.git`, credential files and links,
  and stops at its limit (lowered in the test) saying so.

Item 2, highlighting (TestHighlight, TestSyntaxPalette).

- 2.1 Go, JavaScript, TypeScript, Java, Python, Rust, C#, SQL, YAML, JSON,
  Markdown and a shell script are recognised by name with their language
  name; a keyword and a string of each take the keyword and string
  colours.
- 2.2 A file with an unknown name and no recognisable content is "Plain
  text", every run in the text colour.
- 2.3 The runs of every line, joined, give the line back byte for byte.
- 2.4 With the 16-colour palette each syntax style uses its ANSI colour.
- 2.5 A 1 MiB Go file is highlighted whole; a larger one only in the range
  asked, and its window says "coloured around the view".

Item 3, the window and the quick open (internal/app and internal/render
tests).

- 3.1 Geometry: 76x22 at 80x24, 96x32 at 120x40, 160x48 at 200x60, centred;
  full screen fills the screen; the dashboard rows outside the window are
  the dashboard's own, and an event arriving while it is open changes them.
- 3.2 Opening from an entry: a Read with offset 40 puts the cursor on line
  40; an Edit marks the lines of its `new_string`; a Codex fileChange marks
  its first added lines; F3 on an entry without a file opens the quick
  open.
- 3.3 The quick open lists a temporary project's files, filters as
  described, opens the selection, and says when the listing was cut.
- 3.4 Keys: each key above does what it says, and every key the window
  takes leaves the conversation's draft unchanged.
- 3.5 Refusals and states show in the window: outside the project, a
  credential file, binary, too large, not found, "reading…", "colouring…".
- 3.6 A result for an older open, colouring or search is dropped.
- 3.7 Mouse: a drag across three lines in the window copies exactly those
  lines' text; a click on the dashboard behind selects nothing; the wheel
  scrolls the window.
- 3.8 Golden frames `file`, `file-full`, `file-search` and `file-finder` at
  the four sizes; the existing frames change only where the entry hint
  gains "F3 file" (reviewed in the diff).
- 3.9 The key overlay lists F3 and the window's keys; every footer fits
  (TestFootersFit).

Item 4, documents and the full check.

- 4.1 `THIRD_PARTY.md` lists Chroma and regexp2 and every module the graph
  gains; the licence test passes; `go.mod` has Chroma as its fifth direct
  dependency.
- 4.2 AGENTS.md, architecture.md, ui-design.md, reliability.md and
  evaluation.md say what was built.
- 4.3 `go build ./...`, `go vet ./...` (also with `-tags live` and
  `timing`) and `gofmt -l .` are clean; the binary's size before and after
  is recorded.
- 4.4 `go test ./...`: 0 FAIL.
- 4.5 The timing step: 4/4 PASS, p95 at most 16 ms.
- 4.6 A terminal run, maximised, on a mock run in a temporary project with
  Go, TypeScript, Java and a credential-named file: F3 opens the quick
  open; typing a name and Enter opens the file in the floating window with
  line numbers and its language; `g`, `/`, `n`, `f` and Esc work; the
  credential file is refused; a mouse drag copies a line to the
  clipboard; screenshots show the colours. No agent CLI starts.
- 4.7 The same run at 80x24.
- 4.8 The changed and new files carry no CR, no BOM and no NUL.
- 4.9 The summary is in `doc/plan17-evaluation.md` and here.

Item 5, the Explorer (amendment 12).

- 5.1 `ListDir` on a temporary project: folders first, then files, each
  sorted without case; `.git`, credential files and links left out; a path
  outside the project, or a link inside pointing outside, refused; the
  limit (lowered in the test) cuts it and says so (TestListDir).
- 5.2 `Status` parses git's output: `M`, `A`, `R` (both paths), `U`, `!`,
  ignored files and folders, untracked folders, a project in a subfolder of
  its repository; on a real temporary repository the markers come out as
  expected and `.git/index` is not rewritten (TestGitStatus,
  TestParseStatus). Without a repository: "not a git repository".
- 5.3 F3 with no file in front opens the Explorer; the root is read, then a
  folder when → or Enter opens it; Enter on a file shows it in the viewer
  (TestExplorerOpen).
- 5.4 Keys: ↑↓ PgUp PgDn Home End, → ← Enter and `/` do what the design
  says; no key the Explorer takes changes the draft (TestExplorerKeys).
- 5.5 Esc: viewer → Explorer, the file under the cursor and its folders
  opened (also for a file opened from an entry); quick open → Explorer;
  Explorer → the cursor's folder closed, the cursor on it; at the top level
  → closed. F3 closes the viewer, the quick open and the Explorer, each in
  one key (TestExplorerBack).
- 5.6 Markers: a modified file `M`, an untracked one `U`, a folder holding
  a change "●", an ignored file dimmed, each in its colour; outside a
  repository "not a git repository" (TestExplorerMarks).
- 5.7 A folder listing or status for an older opening is dropped; reopening
  keeps the open folders and the cursor (TestExplorerStale).
- 5.8 Mouse: the wheel moves the cursor; a click selects a row, a click on
  the cursor row opens it (TestExplorerMouse).
- 5.9 Golden frames `explorer` at the four sizes, reviewed; the `file`
  frames change only in the footer's keys ("Esc back · F3 close"); the key
  overlay lists the Explorer's keys; every footer fits.
- 5.10 Documents: brief (amendment 12), AGENTS.md, architecture.md,
  ui-design.md, reliability.md, evaluation.md say what was built.
- 5.11 Build, vet (also `-tags live`, `timing`), gofmt clean; `go test
  ./...` 0 FAIL; the timing step 4/4 PASS, p95 at most 16 ms.
- 5.12 A terminal run, maximised then at 80x24, on a mock run in a
  temporary git repository (a modified file, an untracked file, an ignored
  folder, a credential file): F3 opens the Explorer with its markers; →
  and Enter open folders; Enter shows a file; Esc returns to the Explorer
  on that file, then closes folders level by level, then closes; F3 closes
  from the viewer in one key; `/` opens the quick open and Esc returns;
  screenshots show the colours. No agent CLI starts; git writes nothing
  (the repository's status and `.git/index` unchanged).
- 5.13 The changed and new files carry no CR, no BOM and no NUL; the
  results are added to `doc/plan17-evaluation.md` and here.

Item 6, deleted files in the Explorer (the owner, 2026-10-18, after the live
checks showed that `D` could never reach a row). A file git versions and the
working tree no longer has is still listed, so that a deletion, by the owner
or by an agent, shows where it happened. A file git never versioned leaves
no trace and is not shown.

- 6.1 Each path git reports deleted has a row under its folder, named in red
  with a red `D`, in the folder's order (folders first, then files, without
  case); a deleted folder is rebuilt from the paths under it, its row a
  folder with `D`; a name also on the disk is listed once, from the disk
  (TestExplorerDeleted, render).
- 6.2 A folder empty on the disk but holding deleted paths lists them, not
  "empty folder"; the folders above a deletion keep their "●".
- 6.3 The cursor on a deleted row: the path row reads "<path> · deleted, not
  on disk"; Enter or a click on a deleted file opens nothing; → and Enter
  open a deleted folder without reading the disk, ← and Esc close it; the
  Explorer opened again reads no deleted folder (TestExplorerDeleted, app).
- 6.4 A real temporary repository with a file deleted at the top level and a
  folder deleted whole gives those rows (TestGitStatus or a new test, real
  git); the golden frames `explorer` show a deleted row at the four sizes,
  reviewed; no other frame changes.
- 6.5 ui-design.md (Explorer) and architecture.md say what was built.
- 6.6 Build, vet (also `-tags live`, `timing`), gofmt clean; `go test ./...`
  0 FAIL; the timing step 4/4 PASS, p95 at most 16 ms.
- 6.7 A terminal run, maximised then at 80x24, in a temporary git repository
  with a top-level file and a whole folder deleted: their rows in red with
  `D` (pixels counted); Enter on the file opens nothing; the folder opens and
  shows its deleted files; git writes nothing. No agent CLI starts.
- 6.8 The changed and new files carry no CR, no BOM and no NUL; the results
  are added to `doc/plan17-evaluation.md` and here.

### Progress

- Step 1 done: `internal/input/file.go`; TestReadFile, TestCredentialName,
  TestListFiles, TestReadFileJunction pass.
- Step 2 done: `internal/render/syntax.go`, the palette's syntax styles;
  TestHighlight, TestSyntaxPalette pass.
- Step 3 done: the window (`internal/render/file.go`), the app wiring
  (`internal/app/file.go`, keys, mouse, paste), the key overlay and footers,
  and the `THIRD_PARTY.md` rows. TestFileRect, TestFileStates, TestToolFile,
  TestFileFooters (render), TestFileOverDashboard, TestFileFromEntry,
  TestQuickOpen, TestFilterFiles, TestFileKeys, TestFileStatesApp,
  TestFileStale, TestFileMouse (app) pass; the golden frames `file`,
  `file-full`, `file-search` and `file-finder` at the four sizes are
  reviewed, the existing frames unchanged. The colouring is injected
  (`Options.Highlight`): the application tests colour outside their synctest
  bubble (`internal/app/main_test.go`), since regexp2's clock goroutine,
  started by Chroma's match timeout, would outlive a bubble.
- Step 4 done: the documents (amendment 11, AGENTS.md, architecture.md,
  ui-design.md, reliability.md, evaluation.md, `THIRD_PARTY.md`); build, vet
  and gofmt clean; `go test ./...` 324 PASS, 1 SKIP, 0 FAIL; the timing step
  4/4 PASS; the terminal runs 29 of 29 PASS maximised and at 80x24.
- Evaluation: 28 criteria, 28 PASS
  ([`doc/plan17-evaluation.md`](doc/plan17-evaluation.md)). Not verified: a
  real symbolic link in the reading and the listing (a junction is tested),
  and the viewer opened from a live agent CLI's entry (the runs replay a
  mock). Amendment 12 was added before the owner approved it.
- Step 5 done: the Explorer (amendment 12 in `doc/brief.md`, the design and
  criteria 5.1 to 5.13 above). `ListDir` and git's `Status` in
  `internal/input/explorer.go` and `git.go`; the window in
  `internal/render/explorer.go`; the app wiring in `internal/app/explorer.go`
  (F3, Esc one level back, F3 closing every window, the mouse). A fault found
  on the way and fixed: a folder left open above a file the viewer went back
  to was never read. TestListDir, TestGitStatus, TestParseStatus,
  TestExplorerOpen, TestExplorerKeys, TestExplorerBack, TestExplorerMarks
  (app and render), TestExplorerStale, TestExplorerMouse, TestExplorerRows,
  TestExplorerCursor pass; the golden frames `explorer` at the four sizes are
  reviewed, the 16 `file` frames' footers read "Esc back · F3 close". Build,
  vet and gofmt clean; `go test ./...` 336 PASS, 1 SKIP, 0 FAIL; the timing
  step 4/4 PASS; the binary 14,171,136 bytes; the terminal runs 39 of 39 PASS
  maximised and at 80x24, git writing nothing.
- Evaluation of amendment 12: 13 criteria, 13 PASS
  ([`doc/plan17-evaluation.md`](doc/plan17-evaluation.md#amendment-12-the-explorer-2026-10-18)).
  Not verified: the `!` and `D` markers, "not a git repository" and "git
  not found" live (unit tests cover them); the 16 `file` frames against
  their first version (untracked; their footers checked); the Explorer from
  a live agent CLI (the runs replay a mock); a real symbolic link in
  `ListDir`.
- Live and simulated checks, at the owner's request with their quota
  ([`doc/plan17-evaluation.md`](doc/plan17-evaluation.md#live-and-simulated-checks-2026-10-18)):
  the `!` marker, "not a git repository" and "git not found" in a real
  terminal (27 of 27 PASS maximised and at 80x24); the Explorer and the
  viewer from live Claude Code (Read and Write entries) and live Codex
  (fileChange entry), 0 FAIL at both sizes, the configuration check PASS.
  The faults found were in the test scripts, none in the application. Still
  not verified: a real symbolic link (needs an elevated terminal); the 16
  `file` frames against their first version (no copy exists); `D` on a row,
  not reachable since a deleted file has no row.
- Item 6, deleted files in the Explorer, at the owner's request: 8 criteria,
  8 PASS
  ([`doc/plan17-evaluation.md`](doc/plan17-evaluation.md#item-6-deleted-files-in-the-explorer-2026-10-18)).
  Each path git reports deleted is a red row with a red `D` under its
  folder, a folder deleted whole rebuilt from git's paths and never read;
  Enter on a deleted file opens nothing, its path row says "deleted, not on
  disk". TestExplorerDeleted (render, app) and TestGitStatus (a real
  repository, a folder deleted whole) pass; the `explorer` frames show a
  deleted file and folder, no other frame changed; `go test ./...` 338 PASS,
  1 SKIP, 0 FAIL; timing 4/4 PASS; the terminal runs 21 of 21 PASS
  maximised and at 80x24, git writing nothing. This settles `D` on a row.
- A real symbolic link, at the owner's request: 3 criteria, 3 PASS
  ([`doc/plan17-evaluation.md`](doc/plan17-evaluation.md#a-real-symbolic-link-2026-10-18)).
  After one UAC prompt, an elevated PowerShell ran only the test binary built
  from the working copy: TestListDir, TestReadFile and TestListFiles PASS,
  every link case run, none skipped. This settles the real symbolic link;
  unelevated, those cases still skip.
- The 16 file frames against their first version, at the owner's request:
  4 criteria, 4 PASS
  ([`doc/plan17-evaluation.md`](doc/plan17-evaluation.md#the-16-file-frames-against-their-first-version-2026-10-18)).
  The first version, found as unreachable git blobs and in the session's
  transcript, both copies identical: each of the 12 frames at 80x24, 120x40
  and 200x60 differs in its footer alone, "Esc close" now "Esc back · F3
  close"; the four 79x24 frames are unchanged. This settles the file frames.
- F3 in the dashboard footer, at the owner's request: 6 criteria, 6 PASS
  ([`doc/plan17-evaluation.md`](doc/plan17-evaluation.md#f3-in-the-dashboard-footer-2026-10-18)).
  "F3 Explorer" after the F2 key at every layout; when the line does not
  fit, "Ctrl+J newline" (shown in B's input) is left out so that "Ctrl+Q
  quit" stays visible, at 120x40 too. 29 golden frames changed in their
  footer alone; `go test ./...` 339 PASS, 1 SKIP, 0 FAIL.
- The draft window, at the owner's request (brief amendment 13): 12
  criteria, 12 PASS
  ([`doc/plan17-evaluation.md`](doc/plan17-evaluation.md#the-draft-window-2026-10-19)).
  Ctrl+G opens the same draft in a window "Draft" over the dashboard, placed
  as the file viewer, Ctrl+F full screen (`f` types a letter), Esc back with
  the draft kept; the external editor is removed. A 10,000-line draft stays
  byte for byte; the window's keystroke allocations are under the unchanged
  bound (998); `go test ./...` 341 PASS, 1 SKIP, 0 FAIL; timing 4/4 PASS.
  In a real Windows Terminal, maximised and 80x24: 16 checks PASS at each
  size, Ctrl+F included, a real paste, mouse copy and wheel
  ([the terminal run](doc/plan17-evaluation.md#the-draft-window-in-a-real-terminal-2026-10-19)).

## Closed plans

One entry each: what the plan did, its approval, its evaluation file and
its commits. Their criteria and progress logs: `git show 8888f59:plan.md`
(plans 1 to 13), `git show 84d2ad8:plan.md` (plan 14), `git show
b047434:plan.md` (plan 15), `git show becfb34:plan.md` (plan 16).

## Plan 1 - Research

The agent CLIs and how to drive them, telemetry, terminal input, the
reference projects, and the decision to build rather than fork
(`doc/research/`). Accepted on 2026-10-16, 9 PASS and criteria 8 and 9 kept as
FAIL, documented exceptions
([`doc/research/plan1-evaluation.md`](doc/research/plan1-evaluation.md));
commit 7e4d67c.

## Plan 2 - Architecture and reliability

`doc/architecture.md`, `doc/events-and-state.md`, `doc/ui-design.md` and
`doc/reliability.md`. Approved on 2026-10-16, 15 of 15 PASS
([`doc/plan2-evaluation.md`](doc/plan2-evaluation.md)); commit d5b910e.

## Plan 3 - Proposal

The MVP roadmap and the decisions it asked for (stack, dependencies, live
runs, configuration check). Approved on 2026-10-16, 12 of 12 PASS; commit
d5b910e, whose `doc/proposal.md` holds the roadmap. The design documents
and the code have replaced it since.

## Plan 4 - MVP (from amendment 4; steps detailed by plan 3)

The MVP, steps 0 to 8: startup, Claude Code, the streaming conversation,
approvals and resumption, Codex, agent activity, telemetry, the UI.
Approved on 2026-10-17 after a live demo on both CLIs, 14 of 14 PASS
([`doc/plan4-evaluation.md`](doc/plan4-evaluation.md)); commits ded53d8 to
cf58739.

## Plan 5 - Post-MVP improvements

The list plan 4 left open, Windows only. Approved on 2026-10-17, 17 of 17
PASS, then 15 of 15 live and 6 of 6 for the follow-up
([`doc/plan5-evaluation.md`](doc/plan5-evaluation.md)); commits d42531f,
897f640, 6813c0e.

## Plan 6 - Plan 5's known limits

Four known limits fixed, then items 5 and 6 added by the owner. Approved on
2026-10-17, 6 of 6 and 8 of 8 PASS
([`doc/plan6-evaluation.md`](doc/plan6-evaluation.md)); commits df273fe,
0814d67.

## Plan 7 - The open list after plan 6

The open list cleaned. Approved on 2026-10-17, 7 of 7 PASS
([`doc/plan7-evaluation.md`](doc/plan7-evaluation.md)); commit bffe9a5.

## Plan 8 - Codex's allow for the session

`acceptForSession` on Codex 0.163.0. Approved on 2026-10-17, 6 of 6 PASS
and its follow-up 3 of 3 ([`doc/plan8-evaluation.md`](doc/plan8-evaluation.md));
commits e75d591, 1daeec1.

## Plan 9 - Plan 8's items left not verified

Two items verified, the third recorded as not verifiable. Approved on
2026-10-18, 6 of 6 PASS
([`doc/plan9-evaluation.md`](doc/plan9-evaluation.md)); commits
c9576d9, 8dea129.

## Plan 10 - A spawned Codex agent's label and the timing step

Both items verified. Approved on 2026-10-18, 7 of 7 PASS
([`doc/plan10-evaluation.md`](doc/plan10-evaluation.md)); commits b71c114,
621609e.

## Plan 11 - A live approval offering the allow for the session

The live approval closed as not verifiable, by the owner's choice.
Approved on 2026-10-18, 3 of 3 PASS
([`doc/plan11-evaluation.md`](doc/plan11-evaluation.md)); commits 81f3bf8,
cb206b6.

## Plan 12 - Manual or Auto approvals

Brief amendment 8: a per-session switch between Manual and each CLI's own
Auto. Approved on 2026-10-18, 18 of 19 PASS and 1 FAIL accepted
([`doc/plan12-evaluation.md`](doc/plan12-evaluation.md)); commits cb31557,
83fa023.

## Plan 13 - A reviewer's refusal live

Claude Code's classifier denial seen live and a render defect it exposed
fixed; Codex's reviewer not reached, closed by the owner. Approved on
2026-10-18, 14 PASS and 3 not observed
([`doc/plan13-evaluation.md`](doc/plan13-evaluation.md));
commits d68bb55, 8888f59.

## Plan 14 - Nothing left open

Every open question of the research files settled or closed by the owner,
Claude Code's question card built and seen live, the documents reduced to
the latest state and a CLI update made visible. Approved on 2026-10-18, 18
PASS and 15 PASS ([`doc/plan14-evaluation.md`](doc/plan14-evaluation.md));
commits 0206ac1, ce06b97.

## Plan 15 - Recent projects, favourites and a project browser

Brief amendment 9: the startup screen's project list (`projects.json`,
favourites first) and its Ctrl+O browser. Approved on 2026-10-18, 21 PASS
([`doc/plan15-evaluation.md`](doc/plan15-evaluation.md)); commits bd9ffa3,
b047434.

## Plan 16 - Git worktree management

Brief amendment 10: the Worktrees view (Ctrl+W, at startup and in session)
with create, list, diff, merge and remove, git through `os/exec`. Approved
on 2026-10-18, 26 PASS
([`doc/plan16-evaluation.md`](doc/plan16-evaluation.md)); commits ce95ba2,
fad6d20, 642c5b9, becfb34.
