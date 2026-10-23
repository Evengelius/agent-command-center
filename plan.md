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
8e2a721:plan.md` for plan 31, `git show b56d23e:plan.md` for plan 32), and its
results in its evaluation file.

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
- **[Nice to have]** Starting it like an app (the owner, 2026-10-19): a built
  `agent-command-center.exe` started from a Start-menu shortcut, pinnable to
  the taskbar, whose target is `wt.exe -d <projects folder> <path to the
  .exe>`, so that it always opens in Windows Terminal (the only supported
  terminal, amendments 6 and 7) with the project field at that folder; the
  `.exe` is rebuilt after each code change.

## Plan 33 - Unsent drafts and images through crashes and restarts

Brief amendment 33 (2026-10-23): the owner asked whether unsent image
attachments survive a crash, a forced termination, a restart and a
resumed session, and approved the investigation's nine corrections,
pasted images kept on disk until their message is accepted (item 8).
The criteria below were written before the code; nothing is committed
until the owner has reviewed the implementation and the results.

### The investigation (2026-10-23, nothing changed)

A scratch copy of e5796cb: 24 deterministic tests, four of them the real
program in a child process; Windows Terminal at 156x44 maximised and at
120x40, a scratch build with drafts on in mock mode, three generated
images through the Open dialog. The evidence stays in the session's
scratchpad (`plan33\tree\internal\app\zz_recovery*_test.go`,
`term\out-p33\`).

- **Safe.** A clean quit, a panic and a closed window store the text and
  the images, even within the 500 ms save delay: Go turns the console's
  CTRL_CLOSE_EVENT into SIGTERM, Bubble Tea into a quit, and the process
  ended by itself in 125 ms (exit code 0). A restart restores the text,
  the files in order with their names (Unicode, spaces) and the
  thumbnails, and the images sent afterwards are the files' bytes. Ctrl+R
  after a CLI crash keeps every unsent attachment and resends a message
  that was in flight. A file deleted after attaching makes the send fail
  with the draft kept, and a restart drops it with a note. The acceptance
  deletes the stored draft; no temporary file was ever left.
- **Failing.** (1) Back after a failed start kept the attachments, sent
  with the next session's message, in another project too. (2) Two
  dashboards on one project shared one stored draft: one's accepted
  message deleted the other's. (3) A hard kill lost the last 500 ms of
  changes, images included. (4) What was typed or attached while a sent
  message waited for its acceptance was stored only at the acceptance or
  a clean exit. (5) Saving an image-only draft deleted its images list
  before writing it again. (6) A damaged images list hid the text for the
  run and turned drafts off. (7) A file replaced after attaching was sent
  with its new bytes under the old preview. And by plan 32's design D3, a
  pasted image never survived a restart.

### Design

- **D1 One stored draft per dashboard.** The drafts folder
  (`%LOCALAPPDATA%\agent-command-center\drafts`) holds one folder per
  dashboard's draft, `<P>-<K>`: P the SHA-256 of the project's path as
  before (cleaned, lower-cased), K 16 random hexadecimal digits. It holds
  `lock`, `draft.txt` (the text, byte for byte), `images.json` (the
  images list) and the pasted images' files (D8). The dashboard whose
  draft it is holds `lock` open with no sharing and writes its CLI's id in
  it; a folder whose `lock` nobody holds is a stored draft waiting for its
  project, told the way plan 16's session registry tells a running
  session, by a file held open. At a session's start the dashboard takes
  one stored draft of its project: one of its own CLI first (a Claude
  Code draft back in Claude Code, a Codex one in Codex), then the newest
  by its last save; taking it is opening its `lock` with no sharing, so
  two dashboards never take the same one. The others stay, and the
  restore's toast counts them ("1 more stored draft of this project").
  A draft from before plan 33 (`<P>.txt`, `<P>.images.json` beside the
  folders) is taken as one of any CLI, moved into a new folder. A new
  draft's folder is made at its first save. The dashboard lets its draft
  go, once written, when it quits, goes Back or ends; a folder holding no
  draft is then deleted. Drafts stay off in mock mode.
- **D2 Back** after a failed start writes the draft, lets it go, and
  forgets the attachments, their pasted bytes and their thumbnails with
  the text: the next session, in any project, starts without them, and
  the same project's next start restores them.
- **D3 Attachments saved at once.** Attaching, pasting or removing an
  image writes the draft at once; typing keeps the 500 ms pacing.
- **D4 While a sent message waits for the CLI's acceptance**, the stored
  draft is that message followed by what was typed and attached since:
  the texts joined by a newline, the sent images first, none twice, as a
  clean exit stores it already; the acceptance leaves what was typed
  since. A failed send or a crash gives the message back to the editor as
  before; an image-only message given back no longer puts a newline
  before what was typed since.
- **D5 The save** writes the pasted images' files first, then
  `images.json`, then `draft.txt`, each through a temporary file renamed
  over the old one; an empty text removes `draft.txt` only, an empty list
  `images.json` only; a pasted image's file the list no longer names is
  deleted last.
- **D6 A damaged images list** is renamed `images.json.unreadable` in its
  folder (deleted with the draft) and the text restored, with the toast
  "draft restored; its images list was unreadable and was set aside";
  drafts stay on. A text that cannot be read keeps today's rule: drafts
  off for the run, said on the screen.
- **D7 A file changed since it was attached.** The SHA-256 of a file's
  bytes is kept when it is attached or restored. At the send, a file
  whose bytes differ is refused: "the message could not be sent:
  shot.png: changed since it was attached; attached again as it is now",
  the draft kept, and the dashboard reads the file again, its chip and
  thumbnail replaced in place ("shot.png: updated"): the next send sends
  what the preview shows. Attaching the same file again by hand does the
  same; an unchanged one is "already attached" as before. The bytes
  rather than the report's size and time: a file saved again unchanged
  is not refused, and an edit keeping both is not missed.
- **D8 Pasted images kept until their message is accepted** (amending
  plan 32's D3): a pasted image is written as `<sha256>.png` in the
  dashboard's draft folder when the draft is saved, listed as
  `pasted:pasted-image-N.png#<sha256>`; its bytes stay in memory as
  before, and nothing else is written. A restart restores it with its
  name, its bytes (checked against the SHA-256) and its thumbnail; the
  next paste is numbered after it. Its file is deleted when its chip is
  removed or its message accepted (D5). A stored copy missing or not
  matching is said ("pasted-image-1.png: its stored copy is missing",
  "...is damaged") and left out; a plan 32 entry (`pasted:name`, no
  SHA-256) is said as before ("was pasted, not kept").
- **D9 The sweep**, at each session's start before a draft is taken: in
  each draft folder nobody holds, the temporary `.draft-*` files and the
  pasted images' files its list does not name are deleted, and the
  folder when it holds no draft; a folder without `lock` is left a minute
  (it is being made); a `.draft-*` file beside the folders (a version
  before plan 33) is deleted once a minute old. A held folder is never
  touched.
- **D10 Docs.** brief.md (amendment 33), architecture.md (ASM-23, the
  Input row), ui-design.md (the stored draft, Back, pasted images, the
  changed file), reliability.md (the draft in Resumption), AGENTS.md
  (status), `doc/plan33-evaluation.md`.

### Criteria

- 33.1 TestDraftSlots (`internal/input`): nothing in the drafts folder
  before the first save; the first save makes one folder named 64
  hexadecimal digits, "-", 16 more, holding `lock` (the CLI's id),
  `draft.txt` (byte for byte) and `images.json`; while it is held, a
  second store of the project takes nothing and leaves every file of it;
  let go, the next store of the project takes it, text and list
  unchanged, its name kept; two let go, one per CLI: a Claude Code store
  takes the Claude Code one, a Codex store the Codex one; two of one CLI:
  the newer, the other counted (1); a `<P>.txt` and `<P>.images.json`
  from before: taken, moved into a new folder, none left beside it; a
  draft let go empty: its folder deleted.
- 33.2 TestTwoDashboards (`internal/app`): two dashboards on one project,
  each with text and an image: two folders; one's message accepted: the
  other's stored draft unchanged byte for byte; each CLI started again
  restores its own text and image.
- 33.3 TestBackForgetsAttachments (`internal/app`): Esc on "The session
  could not start" with a file and a pasted image attached: no
  attachment, pasted bytes or thumbnail left; that project's stored draft
  holds the text, the file's path and the pasted image's file; a session
  in another project starts without them and sends none; the first
  project again: the text, the file and the pasted image restored once,
  no "already attached".
- 33.4 TestAttachSavesNow (`internal/app`, the save timer never firing):
  after two files from the dialog, Alt+V's image and the first file's ✕,
  the stored list equals the chips each time (2, 3, then 2 entries), the
  pasted image's file there; typed text alone is stored only when the
  timer fires. A child process killed (TerminateProcess) as soon as its
  list is on disk, under 500 ms after its last image: the next run
  restores every image.
- 33.5 TestDraftWhileWaiting (`internal/app`): a message ("sent" and file
  A) waiting for its acceptance, then "typed", A and B attached: stored
  "sent\ntyped" with A then B, A once; a crash then: a new model restores
  "sent\ntyped", A and B; the acceptance instead: stored "typed" with A
  and B. An image-only message given back after "x" was typed: the
  editor holds "x".
- 33.6 TestImageOnlySave (`internal/input`): a save with images and no
  text leaves `images.json` and no `draft.txt`; text, then no text, with
  the same images: `images.json` after each save; no text and no images:
  nothing left but `lock`.
- 33.7 TestDamagedImagesList (`internal/app`): `images.json` holding
  `["C:\\half`: the text restored, the toast "draft restored; its images
  list was unreadable and was set aside", `images.json.unreadable` with
  the same bytes, drafts on (the next change stored).
- 33.8 TestChangedFile (`internal/app`): a file rewritten with other bytes
  after attaching: no message written, the notice "the message could not
  be sent: shot.png: changed since it was attached; attached again as it
  is now", the text and the chip kept, the toast "shot.png: updated", its
  thumbnail from the new bytes; Enter: the new bytes sent; the same file
  attached again by hand: unchanged "already attached", changed "updated"
  with one chip; a file rewritten with the same bytes: sent.
- 33.9 TestPastedKept (`internal/app`): Alt+V: `<sha256>.png` in the
  dashboard's folder, equal to the pasted bytes, listed as
  `pasted:pasted-image-1.png#<sha256>`, no other file holding them; a new
  model in the project restores "pasted-image-1.png" and its thumbnail,
  and Enter sends the same bytes; the next paste is "pasted-image-2.png";
  the chip's ✕ deletes the file; an accepted message's file deleted; a
  file altered on disk: "pasted-image-1.png: its stored copy is damaged",
  not attached; a plan 32 entry: "pasted-image-1.png was pasted, not
  kept".
- 33.10 TestDraftSweep (`internal/input`): at a store's start, in a
  folder let go, a `.draft-*` file and a pasted image's file it does not
  list deleted, its draft kept; a folder let go holding only `lock`:
  deleted; a held folder: every file kept; beside the folders, a
  `.draft-*` two minutes old deleted, one just written kept.
- 33.11 The recovery matrix (`internal/app`, child processes: the real
  program, the fake Claude Code, a draft of text, three generated files,
  one named in Unicode and one with spaces, and a pasted image): ended by
  a panic, by the quit a closed console becomes, by TerminateProcess
  after the saves, and by TerminateProcess under 500 ms after the last
  image: each next run restores the four images in order with their
  names and bytes, and the text but what was typed in the last 500 ms
  before a TerminateProcess. TestCodexRestart: a Codex (scripted)
  session's text, two files and a pasted image restored, the images sent
  byte for byte.
- 33.12 Windows Terminal (a scratch build of this tree with drafts on in
  mock mode, maximised, then at 120x40; the owner's clipboard saved first
  and restored after): three generated files through the Open dialog, a
  generated PNG pasted with Ctrl+V, text typed; the window closed about
  200 ms after the last key: the next run shows the text, the four
  images' chips and their four thumbnails; the same with TerminateProcess
  instead: the four images back; no `.draft-*` file left.
- 33.13 Unchanged elsewhere: the golden frames; go build, go vet, gofmt;
  the suite, TestDraftAcceptedDeletes and TestPastedDraft changed as D4
  and D8 say; the race detector on `internal/input`, `internal/state` and
  `internal/app`; the timing step alone within every target.
- 33.14 Docs as D10; every changed text file read byte by byte (no BOM,
  no CR); git status lists the intended files only; nothing committed.

No live run: nothing in how a message reaches either CLI changes.

### Progress

Corrections made as the work went, each with its reason:
- D7 and 33.8: the refusal attaches the changed file again itself, where
  the criteria first had the owner attach it again by hand. The notice is
  too long to share the editor's row with the pills, so plan 29's rule
  gives it the row alone and the "+" band is not there until the next
  send: the Open dialog was out of reach just when the notice asked for
  it.

- Saves refused by Windows: a run of the recovery matrix stored three of
  its four images and never the fourth. The child's own check read
  `images.json` while a save renamed over it, and Windows refuses a
  rename over a file another program holds open without sharing its
  deletion (an antivirus, an indexer and Go's `os.Open` do so); the save
  failed once and nothing tried it again until the next change. The
  store now tries a refused rename or delete again for about a second
  (TestSaveWhileRead), and the app tries a failed save again every
  second while it fails, the toast "draft not saved: ..." each time
  (TestDraftSaveRetried).
- 33.11: the pasted image and the three files have different bytes (the
  generated PNG files had the same bytes as the drawing first pasted, so
  the byte check could not tell them apart).
- D2, found on reading the code again: a stored draft's images read back
  after that draft was let go (Back while they were read) went into the
  editor Back had just emptied; the reading now names its draft and is
  dropped when that draft is no longer the dashboard's
  (TestStaleRestoreDropped).
- The independent review (the `verifier` agent) found four defects, all
  fixed:
  - D4: Enter again while a message waits for its acceptance is refused,
    but the state took the refused text as the turn's draft, in place of
    the waiting message (since the MVP: a crash then gave back the wrong
    text); with D4 the stored draft took it too, and the waiting
    message's pasted copy was deleted. A refused send no longer replaces
    a waiting message (the state's turn test; TestDraftWhileWaiting's
    second Enter).
  - D8: a stored draft's image past this session's limits (a Codex draft
    of two 4 MB pasted images in Claude Code, which sends 7.4 MB a
    message) was refused at the restore and its pasted copy deleted by
    the next save. A restored image is attached whatever the limits, the
    toast saying the draft holds more than the session sends, and the
    send refuses the message with the reason, its count now checked as
    its bytes were (TestRestoreOverLimits).
  - D1: a stored draft whose lock another program held a moment was
    passed over in silence. Taking it is tried again for about a second,
    then said ("a stored draft of this project is in use: the next start
    takes it") and counted; a pasted copy's read is tried again the same
    way (TestTakeWhileOpen, TestDraftInUse).
  - D5: a pasted image's file that could not be deleted failed the save
    after the draft was written; that deletion is now done when it can,
    by the next save or the sweep (TestPruneWhileOpen).
- Its second pass confirmed the four fixes and found one more defect,
  fixed: the wait for a held lock also waited a second for a draft
  another dashboard had just taken, then called it in use. A lock that
  refuses the unshared open but can still be read is open in another
  program and is waited for, within one second for the whole start; one
  that cannot be read is another dashboard's and is left at once
  (TestTakeOwned). Its notes, taken too: a start that meets another
  start's sweep looks at the lock once more 20 ms later; the toast about
  the session's limits is shown only when the images were read with
  those limits known, the send saying it otherwise; the send's count
  check is tested (TestRestoreOverCount).
- The owner's questions on the evaluation's "Not verified" list
  (2026-10-23): a Codex session resumed with Ctrl+R with images attached,
  simulated (TestCodexResumeImages); a stored draft whose images lie on a
  path that does not answer, simulated with a read held by the test
  (TestRestoreSlowPath): it lost what was typed and attached while the
  read hung, a crash, a quit or Back then, since nothing was saved during
  the read. Saves now go on while the stored draft's images are read
  back, listing them first as stored; once read, they take their place
  before the ones attached meanwhile (TestRestoreSlowPathAnswers). A
  Windows logoff or shutdown cannot be simulated without ending the
  owner's session or a second Windows account; the dashboard loads
  user32 from its start (seen on a running one), so Windows sends it no
  logoff or shutdown console event, and the two ways it can end it, its
  console closed or the process ended, are 33.12's.
- The review's third pass, on that change, found no loss and two more
  gaps, fixed: a stored draft's file whose read fails at last because its
  share stays offline was dropped from the draft (Windows calls a server
  it cannot find a path that does not exist); a file whose drive or share
  cannot be reached is now kept, its chip "not reachable", the send
  refused with "its folder cannot be reached" until it answers or the
  chip is removed (TestRestoreShareOffline). And the order changed when
  the read answered while a message waited, and an image the stored
  draft held could be pasted twice: the waiting message's images come
  first, then the stored draft's, then the new ones, before and after the
  read, and such a paste is "already attached" (TestRestoreSlowPathWaiting).
  The stored list is compared as a save writes it, so a restore no longer
  writes it again unchanged.

- [x] Brief amendment 33, this plan, AGENTS.md's status.
- [x] D1, D5, D6, D9 the store (`internal/input/draft*.go`) and its tests
  (TestDraftSlots, TestDraftKey, TestImageOnlySave, TestDraftPasted,
  TestDamagedList, TestDraftSweep, TestSaveWhileRead, TestDraftImages).
- [x] D2 to D4 and D8 in the app (`internal/app/draft.go`, `paste.go`,
  `keys.go`, `run.go`, `app.go`) and their tests (TestTwoDashboards,
  TestBackForgetsAttachments, TestAttachSavesNow, TestDraftWhileWaiting,
  TestDamagedImagesList, TestPastedKept, TestDraftSaveRetried; the
  updated TestDraftAcceptedDeletes and TestPastedDraft).
- [x] D7 the changed file (`internal/input/image.go`, `internal/state`,
  `internal/app/attach.go`, `preview.go`) and TestChangedFile.
- [x] The recovery matrix (TestRecovery, TestCodexRestart).
- [x] Windows Terminal: the window closed and the process killed, each
  maximised and at 120x40, then a restart
  (`scratchpad\term\out-p33b\`).
- [x] The independent review, two passes; its five defects fixed.
- [x] The checks (33.13) on the final code: the suite, the race detector
  on input, state and app (0 races), the timing step alone (7 of 7); the
  Windows Terminal runs made again with the final build.
- [x] D10 and the evaluation (`doc/plan33-evaluation.md`: 14 of 14 PASS).
- [x] The owner's questions before approving: Codex resumed with images,
  a path that does not answer and a share that stays offline simulated
  and passing after the fixes above; a logoff or shutdown not simulated,
  what decides it checked (the evaluation's follow-up: 3 PASS, 1 not
  run); the checks and the Windows Terminal runs made again.
- [ ] The owner's review; then the commit.

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
