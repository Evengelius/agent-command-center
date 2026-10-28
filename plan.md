# Current Plan

The active plan with its progress, the owner's standing decisions, and one
entry per closed plan. Each plan comes from `doc/brief.md` and its
amendments, ends with the evaluation summary of `doc/evaluation.md`, and
waits for the owner's approval before the next one starts. This file keeps
the current state only (the owner, 2026-10-18): a closed plan's full text,
criteria and progress log are in git (`git show 8888f59:plan.md` for plans 1
to 13, `git show 84d2ad8:plan.md` for plan 14, `git show b047434:plan.md`
for plan 15, `git show becfb34:plan.md` for plan 16, `git show f00c2d8:plan.md`
for plan 17, `git show 8b3b16b:plan.md` for plan 18, `git show
d84e5d4:plan.md` for plan 20, `git show f939d96:plan.md` for plan 21,
`git show d13ef71:plan.md` for plan 22, `git show e8d3e3b:plan.md` for
plan 23, `git show a3bfe5d:plan.md` for plan 24, `git show cb8f76d:plan.md`
for plan 25, `git show a9a5145:plan.md` for plan 26, `git show 041e7b8:plan.md`
for plan 27, `git show 8fe0077:plan.md` for plan 28, `git show d696f22:plan.md`
for plan 29, `git show 6025513:plan.md` for plan 30, `git show
8e2a721:plan.md` for plan 31, `git show b56d23e:plan.md` for plan 32, `git show
bc5d92b:plan.md` for plan 33, `git show 47b50ce:plan.md` for plan 34, `git show
b43cf46:plan.md` for plan 35, `git show c0cebc4:plan.md` for plan 36, `git show
31c22ef:plan.md` for plan 37, `git show dc9c6d9:plan.md` for plan 38, `git show
55e2bed:plan.md` for plan 39), and its results in its evaluation file.

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
- **Zone A shows values, not notes** (amendment 25, 2026-10-22, plan 26):
  its telemetry pane carries no notes row, legend or counter, now or in
  any later plan; errors, retries, unrecognised lines, freshness and the
  legends live in the details view (Enter on TELEMETRY), where each value
  has its source and age. An undocumented value keeps its "†" mark where
  it is shown.

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
(amendment 9), the browser replaced by the system's folder dialog in plan
52 (amendment 53); git worktree management was plan 16 (amendment 10). The
automatic VS Code integration was replaced by the owner with a file viewer in
the terminal, plan 17 (amendment 11). The session's model, effort and
Ultracode selection was plan 34 (amendment 34); configuring models beyond the
session stays below.

- **[Nice to have]** Advanced model configuration.
- **[Nice to have]** Cross-provider agent collaboration.
- **[Nice to have]** Provider switching during an active session.
- **[Nice to have]** Complex session profiles.
- **[Nice to have]** Advanced orchestration configuration.
- **[Nice to have]** Google Antigravity as an agent CLI, removed from the
  MVP by amendment 5; the provider interface stays open to it.
- **[Nice to have]** Starting it like an app (the owner, 2026-10-19): a built
  `agent-command-center.exe` started from a Start-menu shortcut, pinnable to
  the taskbar, whose target is `wt.exe -d <projects folder> <path to the
  .exe>`, so that it always opens in Windows Terminal (the only supported
  terminal, amendments 6 and 7) with the project field at that folder; the
  `.exe` is rebuilt after each code change.

## Plan 62 - Git Graph

Brief amendment 64 (2026-10-28): a Git Graph after the owner's images of
an IDE's, in a sidebar tile of its own: the history with coloured lanes,
a commit's files with their statuses, a file's diff in that commit in the
existing diff viewer, a commit's or a file's details, and a scope selector
filtering the history by references; read only. The owner's requirements
(the amendment's long quote) set its scope, its three stages and its
checks. Approved by the owner on 2026-10-28 ("Take that in account and
let's fold in and start"), with another agent's two readings folded in:
the topological order always; no ceiling on how far back, but a guard;
every read started from fixed tips, its pages from one `git log`
process, ended on every change; every reference the graph follows
watched; the visual review of the three stages and the measured results
conditions of the commit.

### What was found (2026-10-28, on 895bea5)

- The right sidebar (plan 43; ui-design.md, Right sidebar): 8 columns
  from 100 columns of screen, two tiles of 6 cells by 3 rows, the
  Explorer's on its rows 1 to 3 and Source Control's on 5 to 7, each in
  four looks, a Sixel image (60x60, its icon drawn on a 24-unit grid,
  `sidebarimage.go`) or a text glyph ("≡", "⑂"), and a tooltip. Their
  panels share the docked place, one at a time, each keeping its state.
  Under 100 columns there is no sidebar: F3 shows the Explorer in zone
  B's place.
- The docked panel (`explorerWidth`) is 15% of the screen's width, 26 to
  36 columns, and docks beside zone B only where zone B keeps 64 columns:
  31 columns at 208x59; at 120x40 it takes zone B's place, 76 columns, and
  under 100 columns the whole width (`PanelCompact`).
- The branches popup (plan 59) reads refs/heads, refs/remotes and
  refs/tags with one `for-each-ref` (the names, HEAD, the upstream and its
  tracking, a worktree, the dates; 1,000 a group, `BranchCap`), lists them
  in foldable groups with a search, and takes the keys, the mouse and a
  paste through `render.Layers`. It reads no commit's hash, and it
  switches branches, which the scope selector never does.
- The diff viewer (plan 44): `GitCLI.Sides` reads a diff's two blobs with
  one `git cat-file --batch` (names such as `HEAD:<path>`, `:0:<path>`);
  the diff is computed in a command and shown in zone B's place
  (`DiffView`, `DiffBox`), side by side from `diffSideMin` columns, `f`
  over zone A too; a note in the lines' place for binary content, an LFS
  pointer, line endings only, no textual change, a file too large, git's
  failure. Its sides are named "HEAD", "index", "ours", "working tree",
  and it is Source Control's: one `ui.Diff`.
- Git's reads go through `runRead` and `stream`: the file system monitor
  off, no optional lock, git's words in English, 15 s at most a call.
- Alt+P, Alt+B and Alt+V are taken, Alt+G is free; the docked panels,
  their commit controls and the diff pass a fixed list of global keys on
  (plan 59: a new global key goes into each list).
- Hover: while the sidebar shows, the mouse reports every motion, and the
  program's filter drops one that changes no hover (plan 43, design D4);
  Source Control's rows already hover (plan 44, design D5).
- This repository: 154 commits on one branch, no merge, no tag; the tests
  and the Windows Terminal run need histories made for them.

### Design

Stage 1: the tile, the history, a commit's files, the historical diff,
the scope.

- **D1. The tile:** a third tile, "Git Graph", on the sidebar's rows 9 to
  11. Its icon, on the others' grid and stroke: a vertical line through
  three commit dots, a second line leaving it at the lowest and joining
  it again at the highest, a branch and its merge; its four looks, its
  Sixel image and the text glyph "◉"; the tooltip " Git Graph ". A click
  on it shows or hides its panel with the focus, as Source Control's does;
  Alt+G does the same from any focus, and under 100 columns, where no
  sidebar is, shows the panel in zone B's place, as F3 the Explorer. The
  three panels share the docked place, each keeping its state.
- **D2. The panel:** "GIT GRAPH" in its header; under it the scope row,
  "◎ Auto ⌄" with the active scope: "Auto", "All", a reference's name, or
  "3 references"; for Auto, the branch it follows ("main → origin/main",
  "main, no upstream", "detached at 1a2b3c4"); then the history; the
  footer the keys, or the cursor row's summary (D12).
- **D3. The scope:** a click on the scope row, or `s`, opens the scope
  popup through `render.Layers`, over the panel and zone B:
  - A search field; then "All · every branch, remote branch and tag" and
    "Auto · the current branch and its upstream", the mode of the two
    marked "◉", else "○"; then Local branches, Remote branches and Tags,
    foldable groups as in the branches popup (Remote and Tags folded at
    first), each reference with a check box ("☐", "☑"), its name and
    its commit's short hash ("Remote branch at 9a1b2c3" for a remote
    one); every reference listed and searched, the rows in view drawn; an
    empty group says so.
  - Space or a click on All or Auto makes it the mode, every box cleared;
    on a reference it ticks or clears its box, the scope then the
    references ticked; the last box cleared, Auto again.
  - Typing searches the references, case ignored, a name holding the
    text; Backspace; a paste, its first line. ↑↓ PgUp PgDn Home End and
    the wheel move the cursor and scroll.
  - Enter applies the scope and closes the popup; Esc first clears the
    search, then closes it, the scope as it was; a click outside closes
    it so too, doing nothing else (plan 58's rule).
  - Its read: one `for-each-ref` of refs/heads, refs/remotes and
    refs/tags with each commit's short hash (`%(objectname:short)`), which
    the branches popup's read lacks. It never switches, fetches, or
    writes anything.
  - Auto at first; the scope holds for the session, the panel hidden and
    shown again, the tile switched.
  - No upstream: Auto is HEAD alone, said in the scope row. Detached: HEAD's
    commit. No commit: "no commits yet" in the history's place, the popup
    listing the references there are. A reference of the selection gone
    since: left out at the next read, the scope row saying "1 reference
    gone"; none left, Auto again, said.
- **D4. The history:** a read starts from fixed tips: the scope's
  references turned into commit hashes once (`for-each-ref` with their
  objects, HEAD's by `rev-parse`): Auto HEAD and its upstream when there
  is one; All HEAD and every branch, remote branch and tag (no stash, no
  notes); a selection its references. One `git log --topo-order
  --format=<hash, parents, author, author time, subject> <tip hashes> --`
  then runs as a stream (`stream`'s kind: the file system monitor off, no
  optional lock), read 200 commits a page: the first at once, the next
  when the cursor or the view comes within 50 rows of the last read; git
  waits on its pipe between pages. Always the topological order, children
  before their parents and each line of history kept together; never the
  date order. No ceiling on how far back: every commit read is kept in a
  compact form (its hash, its parents as rows, its subject, author and
  time: about 100 bytes, 100,000 commits about 10 MB), the labels from the
  tips' references; only the rows in view are drawn; a guard against a
  runaway at 1,000,000 commits, then "first 1,000,000 commits". A commit
  already read is never added twice (a check by hash besides the fixed
  tips). "reading…" while the first page comes, git's failure in the
  history's place. The process ends, its context cancelled and git waited
  for, when its last commit is read, the scope changes, the history is
  rebuilt, the panel hides, the session ends or the dashboard quits;
  never one left behind.
- **D4b. A change of the references:** the tips are compared again before
  each next page, when the panel shows, after Source Control's commit or
  push and the branches popup's switch or fetch, and while the panel
  shows every 2 s by one `for-each-ref` of the references the scope
  follows (HEAD, its branch and upstream for Auto, every reference for
  All, the selected ones), read only, none while it is hidden; so a commit
  of the agent's on any of them, local or remote, is seen. Moved, the
  history is rebuilt from the new tips, the cursor, the unfolded commits
  and the first row in view kept by hash (D7).
- **D5. A commit's files:** → or Enter on a commit, or a click on the
  cursor's commit, unfolds it into its files, read in a command the first
  time (`git diff-tree -r -z --no-commit-id --name-status -M`, and
  `--numstat` for the counts; a merge against its first parent; the first
  commit, with `--root`, against nothing) and kept for the session; ←
  folds it (on a file, goes back to its commit). Each file: its status
  letter, A, M, D, R or T, in the colours of Source Control's changes, its
  name, its folder dimmed, "+12 −3" where room remains, "binary" in the
  counts' place for a binary file; a rename "old → new". Every file of the
  commit read in that one call and scrolled as the history, the rows in
  view drawn; a guard at 100,000 files, then "first 100,000 of N files".
- **D6. The historical diff:** Enter or a click on a file shows what the
  commit changed in it, in the diff viewer (plan 44), read only: its
  parent's version (`<parent>:<old path>`) against the commit's
  (`<commit>:<path>`), both through one `git cat-file --batch`; an added
  file nothing against the commit's, a deleted one the parent's against
  nothing, a rename from its old path; the first commit's files nothing
  against theirs; a merge against its first parent, the path row saying
  "merge, against its first parent 1a2b3c4"; binary content, an LFS
  pointer, a file too large and git's failure with the viewer's notes. The
  sides named by their short hashes. It writes no file, adds nothing to
  zone B's conversation and sends nothing to the CLI; Source Control's
  own diff, when one is open, is kept as it was and shown again with
  Source Control (the viewer's drawing shared, its state apart). Esc
  closes it, the cursor back on the file.
- **D7. Navigation state:** the panel keeps for the session the scope,
  the cursor (a commit's hash, or the hash and a file's path), the
  unfolded commits (their hashes) and the first row in view (its hash).
  Shown again, it reads the history again, as many pages as reach the
  first row kept, and puts the view and the cursor
  back on their commits where they still are; commits added on top move
  neither, the rows above them growing; a cursor commit gone (a reset made
  elsewhere): the first row. A scope changed keeps the cursor and the
  unfolded commits that remain in the new history.
- **D8. Keys and the mouse:** ↑↓ PgUp PgDn Home End move the cursor; → and
  Enter unfold, ← folds; Enter on a file its diff; `s` the scope; `f` the
  wide view (D10); `i` the card (D13); Ctrl+C the cursor's full hash, or
  a file's path; Esc back to the input, the panel kept, as the Explorer's;
  Alt+G hides it. A click on a row puts the cursor there; on the cursor's
  commit it unfolds or folds it, on the cursor's file it opens its diff
  (the Explorer's rule, as plan 61's quick open); the wheel moves the
  cursor 3 rows; a click on the scope row opens the popup. The key
  overlay and ui-design.md name them; Alt+G goes into the pass-through
  lists of the other panels, their commit controls and the diff.

Stage 2: the lanes.

- **D9. The lanes:** a column of lanes at the left of each row, one row a
  commit, 2 cells a lane: the commit's dot in its lane, "●", a merge's a
  ring, "◉"; the other lanes' lines going on, "│"; and in the commit's
  own row the curves to its parents' lanes and from its children's,
  "╮" "╯" "╭" "╰" "─", the figure of `git log --graph` without its rows in
  between. Each lane keeps its colour, one of 8 that read on the dark
  ground. Laid out from each commit's parents in the order read: a lane
  freed where its line ends goes to the next that needs one; computed as
  each page comes from a small running state, the rows already drawn never
  moving. Our own code: other tools' graphs studied, none copied; git's
  own `git log --graph` a check in the tests, not used on screen (its
  curves take rows between the commits, and `--graph-lane-limit` is too
  recent to rely on).
- **D10. The lane limit, never misrepresenting:** in the panel, as many
  lanes as leave the subject 12 cells, two at least; beyond them one
  overflow column, "┆" where hidden lanes pass and "◆" for a commit in
  one, with no curve drawn to or from it: no line ever joins two commits
  that git does not join, and the footer says "lanes hidden: f widens the
  graph". `f` gives the panel zone A's columns too, as the diff's `f`
  does, with more lanes and the author and date in columns of their own;
  `f` again gives them back.
- **D11. A row:** the lanes, the commit's labels, then the subject: its
  branches, remote branches and tags as short pills in their colours,
  HEAD's branch first and marked, cut before the subject is; the subject
  cut with "…" only where the row ends; the author dimmed where room
  remains.

Stage 3: the details.

- **D12. The footer:** on a commit, its short hash, author and date ("f52c343
  · Ana Hubaut · 2 days ago"); on a file, its full path and its status in
  words ("Modified", "Added", "Deleted", "Renamed from <old>", "Type
  changed").
- **D13. The card:** with the pointer on a row, or `i` for the cursor's
  row: on a commit its author, the date relative and exact in the
  machine's zone ("2 days ago (26/10/2026 09:41)", `Options.Location`),
  the whole message wrapped (20 lines at most, then "…"), "3 files
  changed, +12 −3", its branches, remote branches and tags, the short hash;
  on a file its full path and status. Drawn beside the row through
  `render.Layers`, never over the row itself; gone when the pointer leaves
  the row or with `i` again. The motion filter's hover rule takes the
  graph's rows: a frame only when the row under the pointer changes.
- **D14. Unchanged and never:** git read only, `log`, `for-each-ref`,
  `rev-parse`, `diff-tree` and `cat-file` through `runRead` and `stream`,
  every call traced in the tests; no checkout, switch, cherry-pick,
  revert, reset, branch or tag creation, fetch, pull or push from the
  graph; nothing written to the working tree or the index; no avatars, no
  animations, no search of the commits' messages (the popup's search of
  references stays). Source Control, the Explorer, the branches popup
  and the diff viewer's other uses as they are.
- **D15. The budget:** a key to its frame within 16 ms with 50,000 commits
  read, their lanes drawn (the rows in view only, the lanes and the rows'
  texts made once a page); the first page shown within 1 s on the
  generated 50,000-commit history; timing tests in the timing step; the
  results reported as measured, the targets never lowered.

### Criteria

1. **62.1 The tile:** the third tile in its four looks, Sixel and text
   forms, and its tooltip; a click and Alt+G show and hide the panel, the
   three sharing the docked place and keeping their states; under 100
   columns Alt+G shows it in zone B's place; golden frames.
2. **62.2 The history:** on a made repository, Auto's commits in
   `git log --topo-order`'s order, each row's subject, author and labels;
   pages of 200 joining with no commit missing or twice, also when commits
   are made on a followed reference between two pages (the tips fixed,
   then the history rebuilt, the cursor kept); read again after a commit,
   a push, a switch, a fetch and a reference moved outside, local or
   remote, seen within the watch's 2 s; "no commits yet", a detached
   HEAD, no upstream; the reading process ended when its last commit is
   read, the scope changes, the history is rebuilt, the panel hides, the
   session ends and the dashboard quits, no git process left after each.
3. **62.3 The scope:** Auto at first; All; one reference; several; All
   and Auto exclusive, a ticked reference leaving them, the last cleared
   giving Auto; the search, its scroll; Enter applying, Esc and a click
   outside keeping the scope; the scope row's name each time; a reference
   gone; the history each scope's, compared with `git rev-list` of the
   same revisions; the scope held after the tile is switched; every git
   call traced: no switch, no fetch, no write.
4. **62.4 A commit's files:** A, M, D, R and T, their letters and
   colours, a rename's two paths, the counts and "binary"; a merge's files
   against its first parent; the first commit's; 1,500 files, every one
   listed and scrolled to; the guard's note past 100,000.
5. **62.5 The historical diff:** an added, a modified, a deleted, a
   renamed and a binary file; the first commit; a merge against its first
   parent, said; the sides' names; the working tree's files, Source
   Control's state and its open diff, zone B's entries and the CLI's
   input the same before and after, compared.
6. **62.6 The lanes:** made histories, each row's lanes against the
   expected figure: one line; a branch and its merge; a merge of three
   parents; criss-cross merges; two roots; more lanes than fit, the
   overflow column drawing no join git lacks (every curve checked against
   the parents, and the commits joined the ones `git log --graph` joins);
   the rows unchanged as pages come; the wide view's lanes.
7. **62.7 Readable:** at 208x59, 120x40 and 80x24 and in the wide view,
   the subject whole where it fits and cut with "…" where it does not,
   the labels cut first; golden frames.
8. **62.8 The details:** the footer on a commit and on a file; the card's
   lines for a commit and a file, by hover and by `i`; it goes with the
   pointer and with `i`; Ctrl+C the full hash, the path; the dates in
   the zone given.
9. **62.9 Navigation state:** the tile switched and back: the scope, the
   unfolded commits, the cursor and the first row kept; three commits
   added meanwhile, the cursor and the first row on the same commits; a
   scope changed, the visible ones kept.
10. **62.10 Large histories:** a generated history of 50,000 commits with
    branches and merges: the first page within 1 s, pages joining to its
    first commit, the memory kept about 100 bytes a commit, a key to its
    frame within 16 ms with all 50,000 read (the timing step); the
    results as measured.
11. **62.11 Windows Terminal:** in the acc tab at 208x59, 120x40 and
    80x24 (Alt+G), real keys and clicks on a made repository: the tile,
    the history and its lanes, the scope popup with a search and a
    selection, a commit unfolded, a historical diff, the card; shots of
    each stage; the owner's visual review of the three stages, the tile's
    icon with them, before the commit.
12. **62.12 The checks:** build, vet (Windows, Linux), gofmt and the whole
    suite; the golden frames reviewed; the changes broken on purpose
    through `go test -overlay`, each caught; the timing step with the
    graph's tests.

### Steps

1. Amendment 64 and this plan.
2. Stage 1: the tile and its panel (D1, D2), the history and its pages
   (D4), the scope popup (D3), a commit's files (D5), the historical diff
   (D6), the navigation state (D7), the keys and the mouse (D8); the
   rows a dot each, no lanes yet; tests on made repositories, the
   50,000-commit one generated with `git fast-import`; golden frames.
   Its shots sent when it ends, for the owner's look; the work goes on.
3. Stage 2: the lanes, their limit and the wide view (D9 to D11);
   topology tests; golden frames; the frame's timing. Its shots sent.
4. Stage 3: the footer and the card (D12, D13); tests. Its shots sent.
5. The documents.
6. Windows Terminal (62.11); the owner's visual review of the three
   stages.
7. The checks, the mutants, the timing and `doc/plan62-evaluation.md`
   with the measured results; the commit only after the owner's review of
   them and of the three stages.

### Progress

- 2026-10-28: step 1: amendment 64 and this plan, proposed; another
  agent's two readings folded in (D4, D4b, D5, D9, D15, 62.2, 62.4, 62.6,
  62.10, 62.11, the steps); approved by the owner ("Take that in account
  and let's fold in and start").
- 2026-10-28: step 2, stage 1, done:
  - `internal/input/graph.go`, the four reads (an implementer agent, then
    reviewed): `GraphRefs`; `GraphLog`, one `git log` process read page by
    page, ended on every end (`ErrGraphClosed` once closed); `CommitFiles`;
    `CommitSides`. 16 tests on real repositories, histories made with
    `git fast-import`, every git call traced (the five reads only, the file
    system monitor off). On 50,000 commits the first page came in 196 to
    303 ms over four runs, all 250 pages in 0.8 to 1.6 s; 18 mutants of the
    reads, 17 caught, the survivor equivalent.
  - `GIT_NO_LAZY_FETCH=1` on the graph's calls alone (D14): a blobless
    partial clone's blob stays out (`TestGraphNoLazyFetch`, its mutant
    caught); Source Control's reads unchanged. In such a clone a commit's
    files read fails in git's words.
  - The panel, the scope popup and the third tile (`render/graph.go`,
    `render/graphscope.go`, the sidebar), the application's side
    (`app/graph.go`, `app/graphscope.go`): keys, mouse, watch, the place
    kept. The tile's icon redrawn as two lines of history from a fork to a
    merge, the first drawing reading as a play button. A commit that changed
    no file says so ("no file changed", "no change against its first
    parent") instead of unfolding into nothing (`TestGraphNoFiles`), found
    on the demo repository.
  - `TestGraphFrameCost`: 50,000 commits read by End, a page each, none
    missing or twice; 135 bytes kept a commit, the subjects' text aside
    (D4: about 100); a key to its frame p95 1.0 ms outside the timing step.
  - Golden frames: 87 changed by one cell each, line 13's third tile "◉"
    (checked by script); 12 new, `graph`, `graph-scope` and `graph-diff` at
    the four sizes.
  - Windows Terminal (`p62run.ps1` over `p62demo.mjs`'s 321 commits) at
    208x59 and 120x40: Alt+G; → and a diff; the popup's search and two
    references ticked; All; End; clicks on a merge and its file, its diff
    against its first parent; the tile hiding and showing the panel. Every
    step as expected; the shots sent to the owner.
  - Drawn otherwise than the text above: a rename's source after "←", as
    Source Control writes it (D5: "old → new"); the labels give way whole,
    the last first, before the subject is cut (D11).
- 2026-10-28: step 3, stage 2, started: `render/graphlanes.go`, the lanes'
  layout and figures, not yet drawn in the panel. `TestLanesTopology`, on
  40 random histories of 300 commits: each commit's lines, followed as
  drawn, reach its parents and no other commit, every lane shown, 3 and 2.
  The test found a new branch's dot right under a root's, which reads as
  its child: a new branch now never takes the lane a root ended on the row
  above.
- 2026-10-28: step 3, stage 2, done:
  - The lanes drawn in the panel (`render/graph.go`, `viewLanes`): each
    row's from the nearest row kept whole (64 apart), in eight colours
    (`laneColour`); the files and notes under the lanes going on; as many
    lanes as leave the subject 12 cells, two at least, then one overflow
    column, "┆" and "◆", no curve to or from a hidden lane, the footer's
    "lanes hidden: f widens the graph". The application lays out each
    read's commits afresh, every parent decoded once (`app/graph.go`).
  - `f`, the wide view: from zone A's edge to the panel's, while the panel
    has the focus and no diff is open; the author and the date
    ("02/01/2006 15:04", `Options.Location`) in columns of their own.
    Under 100 columns, where the panel spans the width already, `f` gives
    the columns alone.
  - Tests: `TestLanesAgainstGit` on seven recordings of `git log --graph
    --topo-order` (`testdata/graph`, made by `TestGraphRecordings -update`
    in `internal/input`, an implementer agent; the spaces at the lines'
    ends cut): line, a merge, three parents, criss-cross, two roots,
    13 lanes, 300 random commits with 41; each commit's drawn lines reach
    its git parents and no other commit, every lane shown, 3 and 2.
    `TestLanesPaged`: the rows unchanged as pages come. `TestGraphPanelLanes`,
    `TestGraphWide`; the lanes of a read again counted in
    `TestGraphTipsMoved`.
  - Golden frames: `graph-wide`, `graph-lanes` (a merge of ten parents)
    and `graph-lanes-wide`, and every graph frame at 208x59 too
    (criterion 62.7); the 87 others unchanged.
  - `TestGraphFrameCost` with the lanes drawn: a key to its frame p95
    1.0 ms on 50,000 commits; 145 bytes kept a commit, the subjects' text
    aside. Measured with the subjects' text since, fresh as a real read
    gives it, 48 bytes each: 209 bytes a commit, its entry 91 (88 and the
    slice's room), its row 8, its subject 49, the index by hash and the
    lanes 61; about 21 MB for 100,000 commits, not D4's 10 MB.
  - Windows Terminal (`p62lanes.ps1`) at 208x59, 120x40 and 80x24: Auto's
    lanes, `f` and back, All's lanes, `f` on them, End and Home; every step
    as expected; the shots sent to the owner.
- 2026-10-28: step 4, stage 3, done:
  - The footer (`render/graphcard.go`, `graphFooter`): the cursor's commit,
    its short hash, author and when ("2 days ago", the exact date without
    the frame's time), or its file's path and status in words, before the
    keys that fit.
  - The card (`graphCard`, `Stack.Card`: under every popup, window and full
    view): with the pointer on a row, or `i` for the cursor's; left of the
    docked panel beside its row, else in the panel under or over it, never
    over the row; a commit's author, when relative and exact, its whole
    message (20 lines at most, then "…"), its files summed up, a merge's
    "against its first parent", its labels, its short hash in the top
    edge; a file's path and status. `CommitMessage` (`git log -1
    --format=%B`, `TestCommitMessage`); the application's side in
    `app/graphcard.go`: the reads once each, their messages and files kept
    up to 1,024, then those of the commits not unfolded dropped; the motion
    filter lets through only a motion onto another row.
  - Tests: `TestGraphFooter`, `TestGraphCardDrawn` (render), `TestGraphCard`
    (the application); the golden frame `graph-card` at the five sizes;
    the graph frames' footers now with the cursor's commit.
  - Windows Terminal (`p62card.ps1`) at 208x59 and 120x40: the footer on a
    commit, `i` and `i` again, the pointer on a merge and off the panel, a
    file's card after → ; every step as expected; the shots sent to the
    owner.
- 2026-10-28: step 5, the documents: ui-design.md's Git Graph section
  (the panel, the history, a row, the lanes, the files, the historical
  diff, the scope, the footer and the card), its two key tables and Alt+G
  among the keys anywhere, the sidebar's third tile; the key overlay's
  rows (95, `TestKeyOverlayRows`); architecture.md's Git Graph; the timing
  step's list in evaluation.md with `TestGraphFrameCost` (nineteen tests).
- 2026-10-28: step 6, Windows Terminal (criterion 62.11): the three stages
  at 208x59, 120x40 and 80x24 (`p62run.ps1`, `p62lanes.ps1`,
  `p62card.ps1`), every step as expected; at 80x24 the merge and file
  clicked are feature/export's, in view there, and Alt+G twice stands for
  the tile's two clicks (no sidebar under 100 columns). Found there: the
  card took no pointer under 100 columns, the mouse reporting motions only
  with a button held where no sidebar shows; every motion is now reported
  while the Git Graph is drawn, as for Source Control
  (`TestGraphMouseMode`). The shots of each stage sent to the owner; the
  owner's visual review of the three stages and the tile's icon waits.
- 2026-10-28: step 7, the checks, in `doc/plan62-evaluation.md`: 12
  criteria, 9 PASS, 2 FAIL, 62.11's owner's review not yet given.
  - 23 mutants through `go test -overlay`, all caught in the end: 20 in
    the first run, then `TestGraphLanesRoom`, `TestGraphStalePage` and
    `TestGraphDuplicate` written for the three that survived. Tests added
    for evidence the criteria name: `TestGraphWatch`,
    `TestGraphScopeRevList`, `TestGraphCopyAndScroll`,
    `TestGraphScopeKeepsPlace`, `TestGraphFollows`. The scope popup's keys
    check a frame was drawn, as the branches popup's do.
  - `go test -race` on the three packages: no race.
  - 62.10 FAIL: 209 bytes kept a commit, not about 100; the speed within
    its targets.
  - 62.12 FAIL: the timing step's TestBannerToFrame and
    TestHomePictureTime over their targets, the home screen's, as on plan
    61's tree timed in turn: the machine's load; to run again on a quiet
    machine. The rest of the step, the graph's p95 1.97 ms, within.
  - Waiting: the owner's visual review of the three stages and the tile's
    icon, the decision on 62.10, then the commit.
- 2026-10-28: the owner's visual review of the three stages and the tile's
  icon: "Looks right" (62.11 PASS, 10 PASS and 2 FAIL in all).
- 2026-10-28: the timing step again, the owner's dashboard closed: all
  nineteen within their targets (the banner p95 1.80 ms, the picture's worst
  186.7 ms, the graph's p95 1.00 ms): 62.12 PASS, 11 PASS and 1 FAIL in all,
  62.10's memory for the owner's decision.
- 2026-10-28: 62.10 accepted by the owner as a documented exception ("Accept
  it"): 209 bytes a commit, the criterion FAIL as measured. The commit waits
  for the owner's approval.

## Closed plans

One entry each: what the plan did, its approval, its evaluation file and
its commits. Their criteria and progress logs: `git show 8888f59:plan.md`
(plans 1 to 13), `git show 84d2ad8:plan.md` (plan 14), `git show
b047434:plan.md` (plan 15), `git show becfb34:plan.md` (plan 16), `git show
f00c2d8:plan.md` (plan 17), `git show 8b3b16b:plan.md` (plan 18), `git show
d84e5d4:plan.md` (plan 20), `git show f939d96:plan.md` (plan 21), `git show
d13ef71:plan.md` (plan 22), `git show e8d3e3b:plan.md` (plan 23), `git show
a3bfe5d:plan.md` (plan 24), `git show cb8f76d:plan.md` (plan 25), `git show
a9a5145:plan.md` (plan 26).

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

## Plan 17 - A file viewer in the terminal

Brief amendments 11, 12 and 13: a file viewer in the terminal with Chroma's
highlighting, an Explorer of the project (F3) and the draft in a window
(Ctrl+G), with no external editor. Approved on 2026-10-19, 91 PASS
([`doc/plan17-evaluation.md`](doc/plan17-evaluation.md)); commits 1748808,
10c3e74, 904f9a6, 749cef4.

## Plan 18 - An adaptive layout

Brief amendment 14: zone C sized by the agents in play, a strip of 4 rows (3
under 100 columns) when there are none, and cleared at each new prompt.
Approved on 2026-10-19, 22 PASS
([`doc/plan18-evaluation.md`](doc/plan18-evaluation.md)); commit 8b3b16b.

## Plan 19 - Zone B as the centerpiece

Brief amendment 15: one speaker label per exchange, RESULT over an
exchange's outcome, compact tool rows and folds of long runs of calls, each
entry's turn (`Entry.Stage`). Done on 2026-10-20, 19 PASS
([`doc/plan19-evaluation.md`](doc/plan19-evaluation.md)). The owner
reviewed it in a demo the same day and asked for plan 20 (amendment 16),
which keeps its logic and replaces its look; not committed on its own, it
was committed with plan 20 once the owner approved that, on 2026-10-20;
commit d84e5d4.

## Plan 20 - Strict conformance to the approved templates

Brief amendments 16 and 17: the owner's two templates as one design
system, A and B in one upper row and C full width under them, zone B's
messages as blocks with a rule between them, the tool rows, the folds, the
pending approval's pink box and RESULT. Approved on 2026-10-20 after the
owner's live review, 32 of 32 PASS
([`doc/plan20-evaluation.md`](doc/plan20-evaluation.md)); commit d84e5d4,
with plan 19.

## Plan 21 - What the review of d84e5d4 found

A read-only review of commit d84e5d4: zone D's session time past 2,000
events, two timed tests outside the timing step, Claude Code's
undocumented `api_message_id` labelled with a fallback, zone C's floor at
160x24 to 160x26, and documents promising what the code did not do (the
send and newline keys built in `config.json`, the others corrected).
Approved on 2026-10-20 with the recommended choices, 38 PASS and 2 FAIL,
then the long session's frame time fixed at the owner's request, 6 PASS
([`doc/plan21-evaluation.md`](doc/plan21-evaluation.md)); commits bdc4250,
f939d96.

## Plan 22 - A live thinking indicator

Brief amendments 18 to 20: zone B's live thinking card for Claude Code and
Codex, with its animations, a summary row for each thinking phase that
opens on its details, and an activity line while a turn works; corrections
C1 to C12 from a read-only investigation
([`doc/plan22-investigation.md`](doc/plan22-investigation.md)); Codex
started through the local proxy that `config.json` names (amendment 19);
Codex's reasoning summaries asked for and Claude Code's hook events known
(amendment 20), checked live with one Codex turn (L20). Approved on
2026-10-21: 31 of 31 PASS, then the corrections 17 of 18 with 5.14 fixed,
amendment 20 13 of 13, and L20 8 of 8 once two criteria were corrected
([`doc/plan22-evaluation.md`](doc/plan22-evaluation.md)); commits
e6895c6, 6306bfb and d13ef71, then the approval commit.

## Plan 23 - The acc launcher in the project

Brief amendment 21: `acc`, typed in Windows Terminal, opens the dashboard
in a tab of its own at the owner's size, a 9 pt font. `acc.cmd` at the root
runs `scripts\acc.ps1`, which writes a Terminal fragment for the profile
and makes Terminal reload. The day's machine-local changes were undone.
Approved on 2026-10-22, 6 of 6 PASS before the commit, then committed and
pushed ([`doc/plan23-evaluation.md`](doc/plan23-evaluation.md)); commit
e8d3e3b, then the approval commit.

## Plan 24 - Zone A with the telemetry, zone D removed

Brief amendment 22: zone A redrawn after the owner's image, holding the
orchestrator's dynamic status, the stages as blocks and zone D's telemetry
(session usage, quota windows with lavender bars and countdowns, the notes,
the details view); zone D removed; Fable and Mythos as model families.
Amendment 23, after the owner's live session: the Claude adapter reads the
`/usage` report after the handshake and after each turn, so the
model-scoped "Fable limit" shows live, the exchange kept out of the
conversation. Approved on 2026-10-22: 35 of 35 PASS with the conformance
table 22 rows PASS, amendment 23 6 of 6 and its live check 8 of 8
([`doc/plan24-evaluation.md`](doc/plan24-evaluation.md)); commits 6209267
and a3bfe5d, then the approval commit.

## Plan 25 - The unrecognised lines named

Brief amendment 24: the owner's scenario 1 run on Fable 5.1 ended with
"unrecognised 2" and no way to see which lines. The details view now lists
the last 20 unrecognised lines with their time, type, reason and text; a
replay tool names them from a logging wrapper's record; the owner's two,
named by a logged rerun, were `system/task_summary`, known with no event
since. Approved on 2026-10-22: 7 of 7 PASS, two live runs
([`doc/plan25-evaluation.md`](doc/plan25-evaluation.md)); commit cb8f76d,
then the approval commit.

## Plan 26 - Zone A without notes

Brief amendment 25: the owner asked to get rid of zone A's notes
("unrecognised 1", "quota updated", "† undocumented field", "cost at API
rates") completely and forever. The telemetry pane now ends with the last
quota window's rows; the counts, ages and legends live in the details view,
where an undocumented value keeps its "†" and the heading explains it; a
standing decision. Approved on 2026-10-22: 6 of 6 PASS, 51 golden frames
reviewed ([`doc/plan26-evaluation.md`](doc/plan26-evaluation.md)); commit
a9a5145, then the approval commit.

## Plan 27 - Zone C with the execution timeline

Brief amendments 26 and 27: the owner's design of zone C and its
close-ups, to respect strictly. Zone C from 100 columns: the header
inside the frame, a rule and the panels' line in the title colour, the
agents' panel as wide as its two-column cards (the label over the clock
and the tokens, a grey bar, the model over the current tool; the selected
agent's rows on a band) and the selected agent's execution timeline with
its status at the heading's right end; the narrow layout keeps plan 18's
row. Approved on 2026-10-22: 15 of 16 PASS, 27.6 corrected with a note
at the owner's decision, 46 golden frames, Windows Terminal screenshots
at 208x59 and 120x40 against the images
([`doc/plan27-evaluation.md`](doc/plan27-evaluation.md)); commit 041e7b8,
then the approval commit.

## Plan 28 - The top right after the owner's image

Brief amendment 28: the owner's close-up of the top right, "● LIVE" over
"Claude Code 2.1.303" over zone B's edge with "Idle". The project bar's
right end is the provider's name and version alone while approvals are
Manual with no change asked; the mode shows there only as Auto or while
changing. Approved on 2026-10-22: 6 of 6 PASS, 27 golden frames, a
Windows Terminal screenshot against the image
([`doc/plan28-evaluation.md`](doc/plan28-evaluation.md)); commit 8fe0077,
then the approval commit.

## Plan 29 - The input editor's row after the owner's image

Brief amendment 29: the owner's image of the editor's row, design only.
Under the draft a separator, then " + " and the pill " <model> High ·
Ultracode " on the tool rows' ground, the editor's messages and the line
count in the free middle, "↯ Auto" at the right end; the "+", "High ·
Ultracode" and "Auto" are placeholders with no function. Zone B is one
row shorter. Approved on 2026-10-22: 7 of 7 PASS, 59 golden frames,
Windows Terminal screenshots at 208x59 and 120x40 against the image
([`doc/plan29-evaluation.md`](doc/plan29-evaluation.md)); commit d696f22,
then the approval commit.

## Plan 30 - The "+" menu after the owner's image

Brief amendment 30: the owner's image of the "+" clicked, design first. A
click on the editor row's band opens a three-row box above the editor's
rule, "🗀  Upload from computer", the band and the item under the
pointer on the active ground (`#272745`), the terminal asked for every
mouse motion only while it is open; Esc, any key or a click elsewhere
closes it; the item does nothing yet, the upload being a later plan's.
Approved on 2026-10-22: 6 of 6 PASS, four golden frames added, Windows
Terminal screenshots with a real click and pointer moves at 208x59 and
120x40 against the image
([`doc/plan30-evaluation.md`](doc/plan30-evaluation.md)); commit 6025513,
then the approval commit.

## Plan 31 - Image upload

Brief amendment 31: the "+" menu's "Upload from computer" works. The
system's Open dialog, several images at once, each checked by its bytes
(PNG, JPEG, GIF, BMP, TIFF, ICO and WebP accepted, the orientation
applied, conversions to PNG, an image over a provider's limits scaled
down and said so; HEIC, AVIF, JPEG XL, JPEG 2000, PSD, TGA, DDS, HDR,
EXR, camera raws and SVG refused by name); chips under the editor's
rule, removable; sent only when the owner submits, as base64 image
blocks to Claude Code and `image` inputs holding data URLs to Codex, no
event keeping the bytes (Codex writes them back). Approved on
2026-10-23: 9 of 9 PASS, eight golden frames added, the real dialog in
Windows Terminal, live turns on Claude Code (Haiku 5.5) and Codex
(gpt-6-sol) naming the colours and the text, and a third on Codex
checking the echo fix
([`doc/plan31-evaluation.md`](doc/plan31-evaluation.md)); commit 8e2a721,
then the approval commit.

## Plan 32 - Image paste and preview

Brief amendment 32: paste an image as in the owner's chat, and a preview
before sending. Alt+V attaches the clipboard's images (copied files, the
"PNG" format's bytes, else the bitmap as PNG); Ctrl+V pastes text as
before and, when Windows Terminal keeps its press for want of text,
attaches the image on the key's lone release (undocumented, probe P32a);
a paste of the copied files' paths attaches them when all are images;
a pasted image is held in memory only. With Sixel graphics, the images
show as thumbnails between the rule and the chips, rounded squares cut
from their centre, 14x7 cells from 50 rows, 10x5 from 30, side by side
one cell apart (the owner's remark on the screenshots), drawn after the
frame and again after any frame that changes their rows, a vacated
place written over with the frame's cells. Approved on 2026-10-23: 10 of
10 PASS, four golden frames added, Windows Terminal at 208x59 and
120x40 with the owner's clipboard saved and restored, the remark in a
follow-up ([`doc/plan32-evaluation.md`](doc/plan32-evaluation.md));
commit b56d23e, then the approval commit.

## Plan 33 - Unsent drafts and images through crashes and restarts

Brief amendment 33: whether unsent image attachments survive a crash, a
forced termination, a restart and a resumed session; an investigation
first, then its nine corrections. Each dashboard's draft has a folder of
its own, its lock held while the dashboard owns it, taken back by the
project's next start (its own CLI's first, then the newest); images are
saved at once, the text every 500 ms; pasted images are kept as
`<sha256>.png` until their message is accepted; a message waiting for
its acceptance is stored with what was typed since; Back forgets the
attachments; a damaged list is set aside; a changed file is attached
again as it is now; refused renames and failed saves are tried again;
unused files are swept at each start. An independent review found five
defects, fixed; on the owner's questions, a Codex session resumed with
images, a slow and an offline share were simulated, fixed where they
lost data, and a logoff was not simulated. Approved on 2026-10-23: 14 of
14 PASS and the follow-up
([`doc/plan33-evaluation.md`](doc/plan33-evaluation.md)); commit
bc5d92b, then the approval commit.

## Plan 34 - The model selector, its effort and Ultracode

Brief amendment 34: a click on the editor row's model pill, or Alt+P,
opens the active CLI's own models with the effort and, for Claude Code,
Ultracode, after the owner's image. Claude Code's list comes from
`initialize`, its changes go by `set_model` and `apply_flag_settings`
and are read back by the undocumented `get_settings`; Codex's list from
every page of `model/list`, its changes ride the next `turn/start`,
read back by `thread/read`. The screen shows a value only once the CLI
reports it in force; an effort held by `CLAUDE_CODE_EFFORT_LEVEL` is
shown and never changed; Claude Code's changes wait for a turn's end. An
independent review found four defects and six doubts, fixed, and three
small points on its second pass. At the owner's request
`CLAUDE_CODE_EFFORT_LEVEL` was removed from their Claude Code settings,
and the dashboard then set the effort live; amendment 35, an opt-in line
telling the agent its settings, was built and withdrawn. Approved on
2026-10-24: 20 of 21 PASS, 34.18 (golden frames changed beyond the
pill's values) accepted
([`doc/plan34-evaluation.md`](doc/plan34-evaluation.md)); commit
47b50ce, then the approval commit.

## Plan 35 - The Ultracode toggle after the owner's image

Brief amendment 36: the owner's image of the Ultracode toggle, a lavender
pill with fully rounded ends and a round white knob, against the
dashboard's `▐   ●▌`. Where the terminal draws Sixel, the toggle is an
image over blank cells of its row, drawn as the preview's thumbnails are:
a pill of 40x20 pixels in an image of five cells whose last carries the
band, since the renderer writes every cell between a row's first and last
changes and a band cell after the image made it blink when the band came
or went (bursts of captures in Windows Terminal); without Sixel, a text
pill of Powerline's half circles. Approved on 2026-10-24: 8 of 8 PASS
([`doc/plan35-evaluation.md`](doc/plan35-evaluation.md)); commit
b43cf46, then the approval commit.

## Plan 36 - The selector's cursor and a pointer that has not moved

Brief amendment 37: Alt+P could reopen the model selector with its
cursor on the row under a resting pointer, so that Enter acted on that
row. The probe (Bubble Tea's `TEA_TRACE`) showed Windows Terminal
reporting a pointer put down by software, without its moving, once the
selector asked for every motion, its reports wandering a cell. The
selector now moves its cursor only when the pointer comes onto another
row than the last report's, the first report after Alt+P only recording
where it rests. Approved on 2026-10-24: 7 of 7 PASS
([`doc/plan36-evaluation.md`](doc/plan36-evaluation.md)); commit
c0cebc4, then the approval commit.

## Plan 37 - The effort slider after the owner's images

Brief amendment 38: the owner's five images of one pill-shaped effort
slider at five levels replaced the selector's effort dropdown. Effort
reads "Effort (<level>)", "Controls reasoning depth" and the slider: the
model's own levels as dots, a fill of `#6784C1` growing to the thumb,
the whole pill lit by a purple gradient with specks at the highest
level, a level asked for ringed, a held effort greyed; with Sixel an
image of 170x20 pixels over 17 blank cells, its thumb gliding in 180 ms;
without, 16 text cells. Effort and Ultracode share label, description
and control columns; ← → and a click on the slider ask for a level.
Approved on 2026-10-24: 9 of 9 PASS
([`doc/plan37-evaluation.md`](doc/plan37-evaluation.md)); commit
31c22ef, then the approval commit.

## Plan 38 - Instant Codex model and effort changes, as in VS Code

Brief amendment 39: a Codex model or effort change took effect only with
the next message, where VS Code's Codex extension changes it at once. The
dashboard now asks for Codex's experimental API at initialize, as that
extension does, and writes a change between turns at once with
`thread/settings/update`, the `thread/settings/updated` that follows
putting it in force (P38x: 1 to 3 ms); a change asked during a turn is
written at its end; a Codex that refuses the update takes every change
with the next message, as before. Approved on 2026-10-24: 12 of 12 PASS
([`doc/plan38-evaluation.md`](doc/plan38-evaluation.md)); commit
dc9c6d9, then the approval commit.

## Plan 39 - The permission modes selector after the owner's images

Brief amendment 40: the editor row's right-hand "Auto" became the mode
pill of the owner's third image, and opens a "MODES" popup after the
first two: the active CLI's own modes (Claude Code's Manual, Edit
automatically, Plan and Auto; Codex's Ask for approval, Approve for me and
Full access), each applied through the CLI (`set_permission_mode`; the
experimental `thread/settings/update` with the presets of VS Code's Codex
extension) and shown in force only once the CLI reports it; Codex's Full
access only after a confirmation. The icons, the check and the band's
rounded ends are Sixel strips beside the text cells, the pill a Sixel
image whose title Windows' GDI draws. Approved on 2026-10-24: 15 of 15
PASS ([`doc/plan39-evaluation.md`](doc/plan39-evaluation.md)); commit
55e2bed, then the approval commit.

## Plan 40 - No empty rows in zone C's execution timeline

Brief amendment 41: on Codex, zone C's execution timeline and the
inspection's Messages tab showed a subagent's rows "assistant │" with no
text, one for each of its reasoning steps: since plan 22 a subagent's
reasoning summaries leave assistant entries holding only a thinking mark,
which zone B never drew. Neither draws an entry with no text now (a
thinking mark, a message that ended empty, a notice with none), a rule
only between two drawn entries. Approved on 2026-10-24: 6 of 6 PASS
([`doc/plan40-evaluation.md`](doc/plan40-evaluation.md)); commit
4158751, then the approval commit.

## Plan 41 - No authorization on low-risk review rows

Brief amendment 42: a Codex auto-review row rated low risk read
"authorization unknown": Codex's reviewer answers a low-risk action with
a bare allow, and Codex then reports the authorization as unknown with a
stock rationale. After a preview, the owner chose to drop the
authorization from review rows rated low risk, in zone B, zone C's
timeline and the inspection; at medium, high and critical it stays.
Approved on 2026-10-24: 7 of 7 PASS
([`doc/plan41-evaluation.md`](doc/plan41-evaluation.md)); commit
3904f13, then the approval commit.

## Plan 42 - The mouse in zone B: a click to fold, a double click to open

Brief amendment 43: beside the keyboard's path, which the owner approved
and which stays as it is, a click anywhere on a foldable item's row in
zone B (a run of calls, a thinking phase's summary row) unfolds it or
folds it again, and a double click on a row of an entry Enter shows whole
opens its full view as Enter does, an approval's rows the approval whole
as from its card. The mouse puts the entry cursor nowhere and shows none
of the keyboard's hints. On those rows a double click no longer selects a
word; a drag still does. Approved on 2026-10-24: 7 of 7 PASS
([`doc/plan42-evaluation.md`](doc/plan42-evaluation.md)); commit
11f794d, then the approval commit.

## Plan 43 - A right activity sidebar with the Explorer docked

Brief amendment 44: a narrow activity sidebar right of zone B, from 100
columns, with the Explorer's and Skills' icons after the owner's two
images. The Explorer, a floating window since plan 17, is a panel docked
between zone B and the sidebar where zone B keeps 64 columns, and in zone
B's place where it does not and under 100 columns; F3 and the Explorer's
icon show and hide that one panel, the focus with it, F3 keeping its file
meaning. The icons' four looks are Sixel images, their hover from
every motion reported, filtered before Update; Skills is a placeholder.
The review led to a third patch of the vendored ultraviolet: a row
holding a wide character is written only as far as it changed, so the
icons beside it stay. Approved on 2026-10-25: 15 of 15 PASS
([`doc/plan43-evaluation.md`](doc/plan43-evaluation.md)); commit
48bde52, then the approval commit.

## Plan 44 - Source Control and a Git diff viewer

Brief amendment 45: the sidebar's second icon, Skills' placeholder until
then, is Source Control. Its panel shares the Explorer's docked place and
lists the project's Git changes from one git status read, the Explorer's
too: MERGE CHANGES, CHANGES and STAGED CHANGES, each file with its real
status, a file staged and changed again in both. A file opens its diff in
zone B's place, the panel beside it from 118 columns: git's blobs read raw
by cat-file and the working file, compared by a Myers diff of our own,
unified or side by side, with line numbers, green and red rows and
navigation between changes; read only. The engine was built by an
implementer subagent. Approved on 2026-10-25: 14 of 14 PASS
([`doc/plan44-evaluation.md`](doc/plan44-evaluation.md)); commit
d9d7d83, then the approval commit.

## Plan 45 - Commit, Push and Commit & Push in Source Control

Brief amendment 46: under Source Control's repository row, a message
field, a purple "✓ Commit" button with a separator and a chevron, and a
dropdown with one item, Commit & Push, after the owner's image. Commit
takes the index as staged (asking before staging anything), refuses a
staged set other than the one shown and counts once HEAD is a new commit
on the old one; Push goes to the configured upstream only, never forced;
the main action is read from git's branch line. After the first report
the owner asked for two additions: the staged files' contents compared
too, and a long operation's elapsed time with Esc cancelling a push (a
Ctrl+Break to git's own process group, behind a check of its lock files;
probes P45a to P45e). The engine's first version was built by an
implementer subagent. Approved on 2026-10-25: 19 of 19 PASS
([`doc/plan45-evaluation.md`](doc/plan45-evaluation.md)); commit
f71e022, then the approval commit.

## Plan 46 - Three fixes from the audit

Brief amendment 47: the first three fixes of the read-only audit of
2026-10-25 ([`doc/audit.md`](doc/audit.md)), each test-first.
- The worktree merge reads the main worktree first, refuses a merge in
  progress or uncommitted changes, and aborts only its own conflict
  (MERGE_HEAD the branch's tip).
- A refused Codex `config/read` or `model/list` no longer ends the
  handshake (`event.Problem.Optional`).
- An opened untracked folder in Source Control is read again after each
  status read and each opening.

Approved on 2026-10-25: 10 of 10 PASS
([`doc/plan46-evaluation.md`](doc/plan46-evaluation.md)); commit 9375233,
then the approval commit.

## Plan 47 - The rest of the audit's phase 1

Brief amendment 48: the rest of phase 1 of the audit's roadmap
([`doc/audit.md`](doc/audit.md), Part 2, C), each fix test-first.
- Ctrl+Q while a commit, push or staging runs asks to confirm, naming
  it.
- An answer to a request the CLI withdrew first expires the approval and
  the session goes on (`provider.ErrNoPending`).
- The reducer's Mode case returns, so mode events leave no false note,
  and a mock replay of Codex declines in Codex's word.
- The Explorer opened over the staging question drops it.
- `realPath` strips the `\\?\` prefix; two stale comments are corrected.
- `TestStateMachines`, failing since plan 46's commit, is repaired.

Approved on 2026-10-25: 10 of 10 PASS
([`doc/plan47-evaluation.md`](doc/plan47-evaluation.md)); commit d44bc0f,
then the approval commit.

## Plan 48 - The audit's phase 2: robustness at the boundaries

Brief amendment 49: phase 2 of the audit's roadmap
([`doc/audit.md`](doc/audit.md), Part 2, C), each fix test-first.
- The TIFF and ICO readers are bounded: the tags read from the file when
  used, an icon's PNG size checked before decoding.
- A running command's streamed output keeps its last 64 KiB, the rest
  counted and said.
- The Claude Code reader never writes to stdin.
- Git calls are bounded by `WaitDelay` once git has ended.
- A panic in the dashboard's own goroutines ends its session in order
  (`Fatal`, `Panicked`); fuzz targets for the image readers and both
  mappers.
- Bidi controls and zero-width spaces are drawn as `<U+XXXX>` on the
  approval card.

Approved on 2026-10-25: 13 of 14 PASS; 48.5 FAIL, its prediction about
the old code, accepted as a documented exception
([`doc/plan48-evaluation.md`](doc/plan48-evaluation.md)); commit 067aeb4,
then the approval commit.

## Plan 49 - The rest of phase 0, and phases 3 and 4 of the audit

Brief amendment 50: the rest of phase 0 of the audit's roadmap and its
phases 3 and 4 ([`doc/audit.md`](doc/audit.md), Part 2, C).
- Phase 0: the event queue's and the write queue's tests, and a matrix of
  the ten overlays alone and in their 90 ordered pairs.
- Tool rows memoised and long sessions' scans bounded (`render.Memo`,
  `State.ToolOrder`, `State.ParentRev`, zone A's rows made when shown);
  `TestToolsToFrame` joins the timing step.
- One owner of the overlays (`render.Layers`), the matrix's five defects
  fixed.
- The adapters' shared lifecycle (`pipe.Proc`).
- The git tests in parallel, git's environment set once in `TestMain`.
- Normalised approval decisions (`event.Decision`), each adapter writing
  its CLI's words byte for byte as before.
- Widths by grapheme cluster (`render/cluster.go`).
- A GIF's first frame alone decoded, its other frames walked by their
  blocks.

Approved on 2026-10-25: 11 of 16 PASS; 49.7 and 49.10, short of their
targets, and 49.8, 49.9 and 49.11, on tests that changed how they call
the code, FAIL, accepted as documented exceptions
([`doc/plan49-evaluation.md`](doc/plan49-evaluation.md)); commit 5c06d39,
then the approval commit.

## Plan 50 - Source Control multi-selection

Brief amendment 51: Ctrl+click selects files in Source Control, a "▎" on
each selected row and "Commit N files" on the button, and Commit and
Commit & Push commit them alone, by `git commit --only`: each selected
file staged whole, a partly staged one refused, unstaged and untracked
ones staged only after the question's yes, the rest of the index left as
it is. Windows Terminal passes Ctrl with a click, never Shift. The
owner's question before the approval of the plan: when the commit fails
after the question's staging, the staging stays, as git leaves it, and
the note says so. After the first evaluation (13 of 13 PASS) the owner
asked that a plain click on a file clear the selection, Commit being the
whole index as before with none selected, and that the note say
"changes"; the review of the complete diff then found a file moved on
disk, both its rows selected, refused once staged, fixed by reading
git's status path by path. Approved on 2026-10-26: 14 of 14 PASS
([`doc/plan50-evaluation.md`](doc/plan50-evaluation.md)); commit
b9f8323, then the approval commit.

## Plan 51 - Home screen redesign

Brief amendment 52: the home screen drawn as the owner's image,
`references/16-home-screen.webp`: picture mode (Sixel images, their text
by GDI in Cascadia Mono) from 118 columns with Sixel, grid mode otherwise;
the project field's path terminal text, its editing, selection, copy and
paste as before; an animated Space Invaders banner, ten frames a second,
only the pixels that change sent, in a fixed palette, paused without the
focus and hidden on small screens; Ctrl+N resets the form, the mouse
works the screen, Ctrl+P is left out. After the evaluation the owner
asked for a livelier pace, then twice for a faster ship (the march every
0.4 s, the ship's legs at 125 to 275 design pixels a second; a dash with
afterimages tried and withdrawn). Approved on 2026-10-27: 10 of 11 PASS
at the evaluation; 51.1, the field's box up to half a row from the
design's place, FAIL, accepted as a documented exception; 51.7 measured
again on a quiet machine, PASS; 51.9's paste back checked by the owner
([`doc/plan51-evaluation.md`](doc/plan51-evaluation.md)); commit
7e60f73, then the approval commit.

## Plan 52 - The system's folder dialog for the project

Brief amendment 53: the home screen's folder button and Ctrl+O open the
system's dialog for choosing a folder (IFileOpenDialog, folders only),
over the terminal, at the field's directory, else its nearest existing
parent, else the starting folder; the folder chosen goes into the field,
which takes the focus, Cancel changes nothing, a failure is a note. Plan
15's project browser removed at the owner's choice, `realPath` and
`dosPath` kept in `input/realpath_*.go`. Approved on 2026-10-27: 5 of 5
PASS ([`doc/plan52-evaluation.md`](doc/plan52-evaluation.md)); commit
9b8b6e5, then the approval commit.

## Plan 53 - A project chosen in the list fills the field

Brief amendment 54: choosing a project in the home screen's PROJECTS
list, by up and down or a click on its row or star, puts its path into
the Project directory field, so that the field shows what Start uses;
Tab onto the list, f and Del leave the field as it is. Approved on
2026-10-27: 5 of 5 PASS
([`doc/plan53-evaluation.md`](doc/plan53-evaluation.md)); commit
1818367, then the approval commit, both pushed.

## Plan 54 - The mouse wheel over the project list

Brief amendment 55: the wheel over the home screen's PROJECTS panel, its
rows, headings and frame, moves the list's selection one project a notch,
as up and down do, choosing it (its path into the field) and giving the
list the focus; the panel's other cells carry `HomeItemList`. Approved on
2026-10-27: 4 of 4 PASS
([`doc/plan54-evaluation.md`](doc/plan54-evaluation.md)); commit
38ee6d7, then the approval commit, both pushed.

## Plan 55 - Nerd Font icons in the Explorer

Brief amendment 56: in the acc tab, with Cascadia Mono NF installed for
the owner (release 2407.24, per user, registered from Windows Terminal:
the Claude app's shells write the registry into its private copy), the
Explorer's rows draw Nerd Font icons in each kind's colour and the
codicons' chevrons; anywhere else (another terminal or profile, no font,
a terminal started before it) the panel's own glyphs. The launcher names
the face in the acc tab's fragment by the same rule. Approved on
2026-10-27: 5 PASS, 0 FAIL, and 55.5's visual review by the owner
([`doc/plan55-evaluation.md`](doc/plan55-evaluation.md)); commit
374c23e, then the approval commit, both pushed.

## Plan 56 - The Explorer's font installed by acc

Brief amendments 57 and 59: Cascadia Mono NF bundled in the repository as
released (`third_party/fonts`, the SIL OFL 1.1 with its symbols'
notices) and installed by acc for the user, under Windows Terminal, when
Windows has no font of that name: the files checked against
`SHA256SUMS`, registered under HKCU, one line printed; a failure one
line, the launch going on; plan 55's rule unchanged, the icons from
Terminal's next start. Withdrawn on 2026-10-28 for plan 57 and reinstated
the same day. Approved on 2026-10-28 without repeating its end-to-end
test, run with the owner on 2026-10-27: 6 PASS, 0 FAIL
([`doc/plan56-evaluation.md`](doc/plan56-evaluation.md)); commit
d88c392, then the approval commit, both pushed; its full text in git
(`git show d88c392:plan.md`).

## Plan 57 - Segoe Fluent Icons in the Explorer

Brief amendment 58: the Explorer's icons from Segoe Fluent Icons, a font
that comes with Windows 11, named after Cascadia Mono in the acc tab's
face, nothing to install. Proposed on 2026-10-28 and withdrawn by the
owner the same day without being implemented (amendment 59), for plan
56's Nerd Font icons and the same look on any Windows version; its
proposal in git (`git show d88c392:plan.md`).

## Plan 58 - A click outside the window closes it

Brief amendment 60: a left click outside the window floating over the
dashboard, the file viewer or the quick open, closes it as Esc's last step
does and does nothing else, the click judged against the frame drawn last
(`Screen.Window`); full screen, its first row is outside; the draft
window keeps its rule. Approved on 2026-10-28: 5 PASS, 0 FAIL
([`doc/plan58-evaluation.md`](doc/plan58-evaluation.md)); commit
0408228, then the approval commit, both pushed; its full text in git
(`git show 0408228:plan.md`).

## Plan 59 - The branches popup

Brief amendment 61: a click on the project bar's branch name, or Alt+B,
opens a popup of the repository's branches and tags: a search, Recent,
Local, Remote and Tags, a message where a group is empty; a switch (git
switch, a remote branch as a tracking one), confirmed when uncommitted
changes would come along and refused while the agent works, a notice in
zone B and a note in an empty draft after it; a fetch of every remote,
never pruning, Esc and a two-minute limit stopping it; a name copied; the
header's branch following the HEAD file, checked every 2 s. New branch
left out by the owner. Approved on 2026-10-28: 10 PASS, 0 FAIL after the
owner's visual review ([`doc/plan59-evaluation.md`](doc/plan59-evaluation.md));
commit 5c8c9be, then the approval commit, both pushed; its full text in
git (`git show 5c8c9be:plan.md`).

## Plan 60 - Git blame in the file viewer

Brief amendment 62: `b` in the file viewer shows, left of the line
numbers, the date and author of the commit that last changed each line,
after the owner's image of an IDE's blame, and in the footer the cursor
line's commit; a line not committed shows "Not committed" and claims no
author; a compact form where the code would keep fewer than 60 cells. Git
is given the very bytes shown (`git blame --porcelain --contents -`), an
answer is kept only for the same file and the latest read, and git's
refusals are named. Approved on 2026-10-28: 9 PASS, 0 FAIL after the
owner's visual review ([`doc/plan60-evaluation.md`](doc/plan60-evaluation.md));
commit 0050409, then the approval commit, both pushed; its full text in
git (`git show 0050409:plan.md`).

## Plan 61 - Clicks in the file viewer and the quick open

Brief amendment 63: a left click on a line of the file viewer puts the
cursor on it, anywhere on its row between the side borders, the view
kept and the selection starting as before; in the quick open a click on
a file selects it and a click on the selected one opens it, as in the
Explorer. A click is judged by the rows the frame drew
(`Screen.WindowList`). Approved on 2026-10-28: 7 PASS, 0 FAIL after the
owner's visual review ([`doc/plan61-evaluation.md`](doc/plan61-evaluation.md));
commit b452ade, then the approval commit, both pushed; its full text in
git (`git show b452ade:plan.md`).
