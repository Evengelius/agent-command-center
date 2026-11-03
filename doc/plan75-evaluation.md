# Plan 75 evaluation (2026-11-03)

The check of plan 75, the session history on the home screen and the return to it from the
dashboard (brief, amendments 89 to 94). Alt+H on the home screen lists the sessions ACC ran in the
chosen project, each with its title from its first prompt, to resume or delete, in `projects.json`
version 2; Esc on an idle dashboard asks whether to return to the home screen, and never while work
runs. Each step was written by an Implementer and judged by a new Reviewer at `xhigh`; the
orchestrator ran step 3's checks and the live run L75.

**Written for the owner's decision. Plan 75 is not closed: nothing is staged, committed or pushed,
and HEAD is `aa77d6e`.** The final review gave PASS on the 21 criteria it judged, 75.1 to 75.21.
It was the second: the first found 75.19 over its usage and 75.20 without its byte check, and its
findings were corrected once. The owner approved 75.22 from the live run's screenshots, and asked
for one fix after the review, the delete question's arrows.

**Then approved and closed, 2026-11-03.** The owner answered "Close and commit": the closure moved
the plan's section here (The plan as it ran), and the plan's files were committed, without a push.

## The result

```text
Evaluation - plan 75, the session history and the return home - 22 criteria: 22 PASS, 0 FAIL
PASS 75.1   the record: version 2 written, version 1 migrated, version 3 refused, 20 entries at most, a title of at most 60 clusters, none cut
PASS 75.2   Ctrl+R and START SESSION as before; TestProjectSessions in version 2's form, every check kept (amendment 92)
PASS 75.3   Alt+H opens for the resolved path, never during a validated start, from Worktrees or on a refused path; frames at three sizes
PASS 75.4   newest first: the title, then date, age, CLI, model or N/A and ID in grey; the scroll mark
PASS 75.5   every key and click of D3; a right click does nothing; Ctrl+N, Ctrl+O, Ctrl+R, Ctrl+W nothing, Ctrl+Q quits
PASS 75.6   Enter and a double click resume (--resume, thread/resume with excludeTurns); a refused resume removes the entry
PASS 75.7   Delete and Delete All by CLI and ID, the whole file compared; Cancel the default
PASS 75.8   nothing under ~/.claude or ~/.codex: the diff searched, and the live run's configuration checks
PASS 75.9   "Alt+H History" between Ctrl+O and Ctrl+Q in both modes; the startup frames differ only on the footer row
PASS 75.10  every earlier Esc kept, one ladder row each; only the Esc that did nothing asks
PASS 75.11  the question's text, buttons, No the default; frames at three sizes
PASS 75.12  never while work runs: a turn, an agent, an approval, a commit, push, stage, switch or fetch
PASS 75.13  stdin closed with quit's escalation, the home screen once the process exits; at once after a crash; no question in a mock run
PASS 75.14  the state New()'s but the answers, every model field fresh, the draft back, a second session's own entries only
PASS 75.15  Esc does nothing on the home screen; Ctrl+Q quits
PASS 75.16  every new or changed machine row has its case; TestStateMachines passes
PASS 75.17  23 and 28 mutants through -overlay, all caught, the code's hash unchanged
PASS 75.18  build, vet on Windows and Linux, gofmt, the whole suite, -race, the timing step alone: all within their targets
PASS 75.19  L75 on the final build, its nine items on each CLI; its usage approved by the owner (amendment 93)
PASS 75.20  the documents describe the code; no BOM, CR or Windows user name in the plan's files
PASS 75.21  only the plan's files differ from aa77d6e
PASS 75.22  the owner's judgement of the view, the question and the footer: "Approved as shown" (amendment 94)
After the final review: the delete question's arrows, the owner's one-off fix (below), checked by its test, a mutant and the whole suite.
Verdict: DONE, waiting for the owner's decision
```

75.1 to 75.21 are the final review's verdicts, its report below; 75.22 is the owner's.

## The checks

On the code the final review judged, the fingerprint
`32076e9fd8a891b01a3d16b8b0631ecd5386de507e968e13ae97bf000e8cf89b` before and after each run, the
same as the Implementer's last evidence file. Times are +01:00.

- **75.18, build, vet, tests and gofmt** (21:22 to 21:28): `go build ./...`, `go vet ./...` and
  `GOOS=linux go vet ./...` exit 0; `gofmt -l .` printed nothing; `go test -count=1 ./...` exit 0,
  16 packages ok (internal/app 346 s, internal/input 328 s, internal/render 46 s), three without
  tests.
- **75.18, `-race`** (21:28 to 21:48), WinLibs GCC 16.2.0, `CGO_ENABLED=1`, one package at a time,
  `-timeout=40m`: internal/state 9 s, internal/input 306 s, internal/render 204 s, internal/app
  549 s; each exit 0, no race report.
- **75.18, the timing step** (21:48), alone, its exact command: on mains power (battery 99 %), the
  CPU at 28 to 33 % by WMI just before it, no other Go build or test; open: Chrome, Edge, Teams,
  Outlook, Notepad, Move Mouse, the Claude app, the VPN client and Windows' own. All 26 tests passed
  within their targets:
  - a key to its frame (p95, target 16 ms): the 10,000-line draft 1.04 ms (its window 1.13 ms); the
    50,000-line draft with 9 saves 2.70 ms; the model selector 1.06 to 1.40 ms; the sidebar hover
    1.06 to 1.93 ms, every idle motion dropped by the filter; the diff view 1.03 to 1.50 ms; the
    commit message 1.07 and 1.50 ms; the branches popup 2.08 ms; the blame 1.33 ms; the Git Graph
    1.01 ms; the turn review 2.01 ms; the search bar 6.09 ms, with answers ten times longer
    12.13 ms; around a 50,000-line chip 1.02 to 2.01 ms; the home screen's field 0.18 ms;
  - a delta or event to its frame (p95, 16 ms): a long session's end 3.03 ms; the thinking card
    2.51 ms; a 2,000-line answer 1.99 ms; a command's 64 MiB output 1.04 ms; 2,400 calls in one
    turn 2.55 ms; the search open 5.44 ms; a 50,000-line chip in view 1.51 ms;
  - the banner, frame to frame, p95 1.83 ms (2 ms); the picture on entering, at most 187.5 ms
    (260 ms); a 50,000-line paste, at most 0.84 ms, 7.93 ms with the draft window open (100 ms); a
    chip's key to its frame at most 4.21 ms;
  - logged with no target, as since plan 73: the full view with the search's bar open, p95 41.1
    and 42.1 ms; `TestKeystrokeCost`, 598 allocations a key.
- **75.17, the mutants** (the Implementer's last evidence file): 23 of 23 caught for step 1 and 28
  of 28 for step 2, through `go test -overlay`, the fingerprint the same before and after each set.
- **75.20, the bytes** (the same evidence file): none of the plan's 107 files, `plan.md` included,
  starts with a BOM or holds a CR or the Windows user name.
- **75.19, the live run L75 on the final build** (21:51 to 21:54). The dashboard and confcheck were
  built from that code (`acc.exe` `5df11ccec16aa313…`, cgo off) and run in ACC's 9 pt profile,
  maximised. Each run had a new project under `%LOCALAPPDATA%\Temp`. The window's `CLAUDE*` and
  `ANTHROPIC_*` variables were removed, with `ANTHROPIC_MODEL=haiku`; Codex ran on its configured
  model. `confcheck snapshot` ran before and `compare -cli-ran` after, and `projects.json` was
  copied before and put back after. No terminal-hosted CLI was running.

  | L75 item | Claude Code 2.1.303 | Codex 0.163.0 |
  |---|---|---|
  | 1 before: no CLI in a terminal, Px, the version, the snapshot | PASS | PASS |
  | 2 a turn gives the code word; the entry recorded, version 2, model, title | PASS: `3b5c6b15`, Claude Haiku 5.5 | PASS: `01a1a330`, GPT-6-sol |
  | 3 Esc, Yes: the home screen, no CLI process left | PASS | PASS |
  | 4 Alt+H: the entry, its CLI, model, title and ID | PASS | PASS |
  | 5 Enter: the session resumed | PASS | PASS |
  | 6 a turn asks for the code word: it comes back, the title kept | PASS: GARNET-75 | PASS: COBALT-75 |
  | 7 Esc, Yes again: the home screen, no CLI left | PASS | PASS |
  | 8 Ctrl+Q: the window closed | PASS | PASS |
  | 9 after: the configuration unchanged, no CLI left, `projects.json` put back byte for byte | PASS | PASS |

  - Each answer was the CLI's own after the resume: "▎ GARNET-75" under Claude Haiku 5.5 and
    "▎ COBALT-75" under gpt-6-sol. The title shown is cut at 60 characters, before the word.
- **L75's two earlier runs** (19:44 to 19:48), on the build before the final review's correction
  (`acc.exe` `97321bac1beac0f2…`):
  - The first Claude Code attempt stopped on a fault in the driver, after its item 2. In
    PowerShell, `$home` is the read-only `$HOME`, since names are case-insensitive. So GoHome's
    assignment threw after Yes had been pressed. Its items 1, 2, 8 and 9 passed. The variable was
    renamed, and the run went again under a new folder.
  - That re-run, and the run on Codex, passed all nine items: Claude Code `eabd2472`, Codex
    `01a1a2bc`.
  - L75's usage in all: five Haiku turns and four Codex turns, where L75 states two of each. The
    owner approved the third Haiku turn and the run on the final build (amendment 93).
- **The checks on that earlier code**, the fingerprint `29763b30…` (19:16 to 19:41): the full run,
  `-race` (internal/app 538 s) and the timing step (26 of 26), all passed.
- The five temporary projects were empty and went to the Recycle Bin. Each move is proven by the
  path gone and a `$I` record naming it.
- The CLIs' own records of these test sessions stay where the CLIs keep them: Claude Code's under
  `~/.claude/projects`, Codex's threads in its state. The plan touches none of them (75.8).

**After the final review: the delete question's arrows.** This is the owner's one-off fix
(amendment 94), checked without a further review, as the owner chose.

- **The fix:** the final review's Medium observation was that ← and → both flipped the selection,
  so with Cancel selected, → then Enter deleted the entry. Now ← selects Delete and → Cancel, as
  they are drawn, like the return question's arrows (`internal/app/history.go`, `historyAskKey`).
  `doc/ui-design.md` says so.
- **The tests:** `TestHistoryDelete` now checks four things: → keeps Cancel, ← selects Delete and
  a second ← keeps it, and → selects Cancel again. The deletions in `TestHistoryCtrlR`,
  `TestHistoryDelete` and `TestHistoryDeleteAllExact` now press ←. The rig of the
  `history-ask-all` frames presses ← and draws the same frames as before.
- **The mutant:** the old toggle, put back through `go test -overlay`, makes `TestHistoryDelete`
  fail ("→ left Cancel"). The tree's hash was the same before and after.
- **Then the whole suite:** 22:23 to 22:28, the fingerprint `f8c86259a6bf3f41cb6f08758a1a43f9b83bb57a7449ec8f288d084404ed62f0` before and after: `go build ./...`, `go vet ./...` and `GOOS=linux go vet ./...` exit 0, `gofmt -l .` printed nothing, and `go test -count=1 ./...` exit 0, 16 packages ok (internal/app 265 s).
- **The records' bytes:** after the records were written, the 108 files git lists as the plan's (`doc/brief.md`, `plan.md` and this file included) were checked again in the evidence's way: none starts with a BOM, none holds a CR or the Windows user name, and each ends with a newline.

## The reviews

Each run is a new `reviewer` instance, never resumed, from the transcripts under `~/.claude/projects`
(`agent-<id>.jsonl` and `.meta.json`); times are UTC.

| Run | Started | Model | Effort passed, used | Description: tier, reason | Minutes | Requests | Tools offered | Result |
|---|---|---|---|---|---|---|---|---|
| `a1b61db1def468daf` | 11:46:59 | claude-opus-5-5 | xhigh, xhigh | Review plan 75 step 1: Deep, store | 14.3 | 52 | Read, Grep, Glob, SubagentHandback, advisor | FAIL |
| `a64f2d7fd907a77f1` | 12:24:54 | claude-opus-5-5 | xhigh, xhigh | Review plan 75 step 1: Deep, store | 14.2 | 66 | Read, Grep, Glob, SubagentHandback, advisor | FAIL |
| `a263c39073f99486f` | 13:10:41 | claude-opus-5-5 | xhigh, xhigh | Review plan 75 step 1: Deep, store | 13.3 | 49 | Read, Grep, Glob, SubagentHandback, advisor | FAIL |
| `a187589889107187a` | 13:50:14 | claude-opus-5-5 | xhigh, xhigh | Review plan 75 step 1: Deep, store | 13.9 | 70 | Read, Grep, Glob, SubagentHandback, advisor | PASS |
| `ac139a66f5e537f89` | 15:16:33 | claude-opus-5-5 | xhigh, xhigh | Review plan 75 step 2: Deep, process | 16.2 | 79 | Read, Grep, Glob, SubagentHandback, advisor | FAIL |
| `af3178579a9405a2d` | 16:06:45 | claude-opus-5-5 | xhigh, xhigh | Review plan 75 step 2: Deep, process | 13.5 | 73 | Read, Grep, Glob, SubagentHandback, advisor | void (FAIL) |
| `a61e3534e9b9787ae` | 17:42:09 | claude-opus-5-5 | xhigh, xhigh | Review plan 75 step 2: Deep, process | 14.8 | 70 | Read, Grep, Glob, SubagentHandback, advisor | PASS |
| `ab0fd5b8cbb9fd068` | 18:52:41 | claude-opus-5-5 | xhigh, xhigh | Final review plan 75: Deep steps | 20.6 | 83 | Read, Grep, Glob, SubagentHandback, advisor | FAIL |
| `a2f08e84067b38b56` | 20:55:56 | claude-opus-5-5 | xhigh, xhigh | Final review plan 75: Deep steps | 18.6 | 84 | Read, Grep, Glob, SubagentHandback, advisor | PASS |

- **The transcripts**, each checked after its run: each run was offered exactly the five tools, and
  every request ran on `claude-opus-5-5` at `xhigh`. No run opened `plan.md`, an evaluation file,
  `~/.claude`, `~/.codex`, or a folder outside the repository and its inputs. The inputs'
  fingerprint was unchanged from each run's start to its end. Reviews 2 and 3 of step 1 read part
  of `plan.md`'s hunk in the diff. The first final review noted that hunk's Progress lines, "Step 1:
  done. Step 2: done. Step 3: in progress.", and gave them no weight. They hold no verdict, as
  `AGENTS.md` allows.
- **The Implementers:**
  - step 1 on `claude-sonnet-5-5` at `high`, the definition's (`a945a0c4c139af3e4`, 302 requests);
  - step 2, the fixes after step 2's review and the final review's correction on
    `claude-opus-5-5` at `high`, as passed (`ac0022a7a02644139`, 359 requests).
- **Before step 1's first review,** one orchestrator decision, not a correction: D6's "where it
  fits" was read by plan 16's rule in `TestFootersFit`, the full row from 100 columns. The
  Implementer had relaxed that test; it was restored as at `aa77d6e`, with the threshold at 100.
- **The ruling.** Step 1's first review found 75.2 in conflict with D1. Plan 69's store test could
  not pass unchanged under version 2. The plan stopped, and the owner answered "Yes, adapt it" to
  "May plan 69's store test change to version 2's format, all its checks kept?" (amendment 92).
  75.2 gained that sentence.
- **Step 1's three corrections, the limit:**
  - After review 1:
    - the ruling applied;
    - 75.3: Alt+H on a path Start refuses now opens nothing, and the field shows the refusal;
    - 75.17: the mutants' hashes now cover every plan file, the untracked ones included;
    - the seven observations, and the requested tests.
  - After review 2:
    - 75.6: a double click on a scrolling list resumed another entry. The list now keeps its
      scroll position, and a double click resumes only when both presses hit the same entry;
    - the architecture document;
    - `Load` now drops entries without an ID and caps the history at 20;
    - the withdrawn question's Enter;
    - the title's scan is bounded, and format characters are removed.
  - After review 3:
    - 75.1: a title kept more than 60 clusters when a space followed the 60th. It is rewritten
      linear, with `FuzzSessionTitle`;
    - Alt+H while the CLIs are detected is back to D2. The orchestrator's choice in the first
      correction had refused it;
    - F12 in the view;
    - tag characters are kept;
    - the Ctrl+R test.
- **Step 2's two corrections:**
  - After review 1:
    - 75.11: the question's title was drawn pink (`#F472B6`), and its test locked the pink. It is
      now amber (`#FBBF24`), as is step 1's delete question;
    - the Not now notice now clears when the work it names ends;
    - Esc keeps the force-stop offer;
    - the session's end runs off the event loop, and the next start waits for it;
    - the ended session's late results are dropped;
    - no git operation starts during the close;
    - Back after a failed start now closes the turn store.
  - After review 2, which counted for nothing: it had seen the Implementer's own account of earlier
    reviews, in two test comments and `ui-design.md`. Those mentions were removed, and the
    Implementer was told to keep any history of the reviews out of the tree. The review's finding
    was real and was fixed all the same. The Git Graph card's reads, untagged, answered after the
    return into nil maps, a panic. They are now tagged with their session and guarded, and every
    other session read was checked. Also fixed:
    - the worktrees start gate;
    - the staging question;
    - Shutdown's bounded wait;
    - the kill fallback;
    - a queued approval.
    The next review ran fresh.
- **After step 2's PASS,** three of its observations were fixed before the final review, which
  judged them:
  - Ctrl+Q during the return's kill grace stays the quit: `ReturnGrace` is guarded by `Exited`,
    tested by `TestReturnKillGraceQuit` and a `TestStateMachines` case;
  - the question no longer opens over a pinned Git Graph card, a new row of the Esc ladder;
  - `ui-design.md`'s sentence on a crash.

  Both new mutants were caught.
- **The stall.** During step 2's second correction, the owner wrote "No something is wrong find what
  it is, fix it and continue, it is an order". Three causes were found:
  - mutants that made a held test fail hung its binary to the 10-minute timeout. The test used
    `t.Errorf`, and its fakes were never released;
  - the runner counted only `--- FAIL` lines, while a synctest deadlock prints none, so four
    mutants were counted as survivors;
  - a standalone mutant run had started beside the evidence command.

  Both runs were stopped. Held fakes are now released in `t.Cleanup` and tests use `t.Fatalf`; a
  panic or a timeout counts as caught; every mutant's `go test` has `-timeout=120s`; and one job
  runs at a time. Separately, a check of the orchestrator's read its own output file in a loop,
  9.6 MB, until stopped. None of this changed the application's code.
- **The first final review** found two things:
  - 75.19 FAIL: three Haiku turns where L75 states two, with no approval in its inputs;
  - 75.20 NOT VERIFIED: the evidence held no byte check.

  It also observed that a Worktrees merge or removal could still start while the return closed the
  CLI, and its result was then dropped unseen, against `reliability.md`. The plan stopped for the
  owner, who approved the third turn and a second L75 run on the corrected build (amendment 93).
- **Its correction,** the final review's one:
  - the byte check in the evidence command;
  - the Worktrees question now refuses a merge or a removal while the return closes the CLI, with
    "Returning to the home screen" (`TestReturnNoWorktreeOpWhileClosing`, a step2.sh mutant);
  - `TestSessionTitleCost` counts the bytes the title's scan reads instead of timing it, with a
    counter, `titleScanned`, in `internal/render/history.go`, and a quadratic-scan mutant;
  - `reliability.md`'s citation of the Known limits, which move into this file at the close, and
    two long lines wrapped.

  Then the full run, `-race`, the timing step and L75 ran again, on the corrected code.
- **`plan.md` during the plan.** Its Progress said "Step 1: in progress." from the approval, "Step 1:
  done. Step 2: in progress." after step 1's last review, and "Step 2: done. Step 3: in progress."
  after step 2's. None held a verdict.
- **Cost:**
  - nine Reviewer runs at `xhigh`, 139.4 minutes of their time;
  - the two Implementers, 302 and 359 requests;
  - the orchestrator's two full runs of 5 to 6 minutes, two `-race` runs of 19 minutes, and two
    timing steps of 2 to 3 minutes;
  - L75: five Haiku turns and four Codex turns.

## What the final review left open

Its observations, none blocking, with what became of each:

- **Medium:** the delete question's arrows flipped the selection. Fixed after the review, at the
  owner's request (The checks, above).
- **Low:** a title can lose its last letter in one case. A first message over 8 KiB whose first
  8 KiB holds fewer than 60 visible characters gives, for example, "hell" for "hello" followed by
  9,000 spaces. A test pins the rule ("ab " gives "a"). The owner kept it (amendment 94): the case
  is rare, and the rule errs on the side of caution.
- **Low:** an images-only first message marks the session as prompted with no text, so it never
  gets a title. That is D1a's "images do not count", read so; the documents do not say it.
- **Low:** Back after a failed start now runs a session end. Until it finishes, START SESSION and
  Ctrl+R say "The last session is still closing", normally for milliseconds.
- **Low:** `reliability.md` cites this file, which exists from the close.
- **Low:** `architecture.md` gains two whole sections, beyond "where it describes a session's
  life". The file is in the plan's scope.
- **Low:** "Not now: an approval is waiting · F2 answers it" is not one of D8's texts; no CLI event
  leads to that state.
- **Low:** the two changed startup rows keep plan 69's machine cases. Their plan 75 parts are tested
  in the application only (`TestHistoryResume`, `TestHistoryCtrlR`).
- **Low:** production code counts each title cluster step in a package-level counter that only
  `TestSessionTitleCost` reads.
- **None:** grid mode shows the full footer row from 100 columns, plan 16's rule.

From the first final review, also left as they were:

- `RecordSession` lets a later record give a title to an entry that has none. "Set once" rests on
  the state never giving a resumed session a first prompt;
- the history view stays open, unseen, in a terminal too small to draw it, and still takes Enter
  and Del, as the Worktrees view does;
- after a return through the kill's grace, a process that survives its forced kill leaves the old
  session's reader and writer blocked on it.

## The plan as it ran

`plan.md`'s section of plan 75 as it stood when the owner approved the closure on 2026-11-03,
word for word, its headings one level down.

### Plan 75 - Session history and the return to the home screen

From the owner's request of 2026-11-03 (brief amendments 89 to 91), with
the assessment's changes and the owner's two adjustments, `projects.json`
version 2 and session titles. The home screen gains a list of the sessions
ACC ran in the chosen project, with resume and delete. The dashboard can
return to the home screen while the session is idle. Zones A, B and C and
the provider code stay as they are.

- **Baseline:** `aa77d6e`.
- **Files:**
  - `internal/input`, `internal/state`, `internal/app`, `internal/render`
    and their tests;
  - `testdata/frames`: the startup frames' footer row, plus new frames;
  - `doc/ui-design.md`, `doc/events-and-state.md`, `doc/reliability.md`,
    and `doc/architecture.md` where it describes a session's life;
  - `AGENTS.md` (the scope list, the status), `plan.md`, `doc/brief.md`;
  - at the close, `doc/plan75-evaluation.md` (new).
  - Nothing under `internal/provider`, `cmd`, `scripts` or `.claude`
    changes, and `go.mod` and `go.sum` stay the same.
- **Tiers:** both steps Deep. Step 1 changes `projects.json`, where a fault
  could silently lose projects, and `AGENTS.md`; step 2 touches the CLI
  process's lifetime, approvals and saved drafts. Every review runs at
  `xhigh`, the final one included.
- **Implementers:** step 1 on Sonnet at `high`; step 2 on Opus at `high`,
  since the teardown has many parts that interact.

#### Design

- **D1. The record.**
  - Only sessions ACC itself ran. `projects.json` becomes version 2: each
    project entry has a `history` list, newest first, of at most 20
    entries: the CLI, the session ID, when its last turn ended, the model
    as zone A names it, and a title (D1a).
  - The list replaces version 1's `sessions` as the one record: Ctrl+R
    resumes the chosen CLI's newest entry.
  - Migration on read: a version 1 file's `sessions` become entries, with
    model "N/A" and no title, its paths, times, favourites and order
    unchanged; the file is written as version 2 at its next change. A
    version above 2 is refused, as today.
  - Older builds keep their protection: they refuse any version but 1 and
    never write a file they cannot read (`Load` and `Update` in
    `internal/input/projects.go` at `aa77d6e`), so a version 2 file shows
    as unavailable there and is never overwritten.
  - Written when the owner's turn ends, as `sessions` is today. Nothing is
    read from `~/.claude` or `~/.codex`, and Codex's `thread/list` is not
    used. A project's history goes with it when the list drops the
    project.
- **D1a. The title.**
  - From the first message the owner sends in a session started new in
    ACC, as the CLI received it; images do not count. Set once: a resume
    never changes it.
  - Control characters removed, so no escape sequence reaches the
    terminal; runs of whitespace made one space; ends trimmed; at most 60
    characters, counted as grapheme clusters so none is cut; shortened to
    fit when drawn.
  - Entries migrated from version 1, and resumed sessions that have none,
    have no title.
  - The one exception to the standing decision on conversation data (The
    owner's standing decisions, above; amendment 91), its limit accepted:
    a first prompt can hold sensitive text.
- **D2. The view.**
  - Alt+H on the home screen opens it, for the project field's path as
    Start resolves it, except while a start is being validated. Ctrl+H is
    taken: Windows Terminal sends Ctrl+Backspace as it.
  - A full-screen view, as Worktrees is, so nothing is drawn over the
    Sixel picture: a centred box in the home panels' style, titled SESSION
    HISTORY. The picture and the banner come back when it closes.
  - Each entry: its title, and under it in grey the date and time
    ("2026-11-03 09:27"), its age as PROJECTS writes it, the CLI, the model
    or "N/A", and the session ID's first 8 characters; an entry without a
    title has the grey line alone.
  - The list scrolls to follow the selection, with the slim scroll mark.
    With no entries: "No sessions recorded for this project yet".
- **D3. Keys and mouse.**
  - Keys: ↑/↓, PgUp, PgDn, Home and End move; Enter resumes; Del deletes;
    Shift+Del deletes all; Esc closes; Ctrl+Q quits.
  - Mouse: a click selects, a double click resumes, a click outside the
    box closes, the wheel moves one row per notch.
  - A right click does nothing, as everywhere in ACC: there is no context
    menu.
- **D4. Deleting.**
  - A question appears in the box, in amber: "DELETE THIS SESSION FROM THE
    HISTORY?" or "DELETE ALL N SESSIONS OF <project>?". It offers Delete
    and Cancel, Cancel selected; ←/→, Enter and a click answer it, and Esc
    cancels.
  - It removes ACC's entries only, never another project's, and never the
    CLIs' own files; Ctrl+R then offers that CLI's next newest entry, if
    any.
  - The text says so: "ACC's list only: Claude Code keeps its own record".
- **D5. Resuming.**
  - Through plan 69's path, with the entry's CLI and ID, and question 1 set
    to that CLI: Claude Code with `--resume <id>`, Codex with
    `thread/resume` without its turns.
  - If that CLI isn't installed, the row is dimmed and Enter says so.
  - If the CLI no longer has the session, plan 69's refusal shows and the
    entry is removed.
- **D6. The footer.**
  - "Alt+H History" goes before "Ctrl+Q Quit".
  - In picture mode it is a fourth group at the design's spacing, a
    departure from the owner's image recorded in "Follows and departs".
  - In grid mode, the full row where it fits (99 cells), otherwise the
    short row ` › Ctrl+N New │ Ctrl+O Open │ Alt+H History │ Ctrl+Q Quit`
    (65 cells).
- **D7. The question.**
  - Esc opens it only where Esc does nothing today: the focus already in
    zone B, and no popup, card, window, view, search or selection open.
  - Title "RETURN TO THE HOME SCREEN?" in amber, then "The session ends and
    Claude Code closes. Its changes stay in the project, and the session
    stays in the history."
  - Yes and No, with No selected. ←/→ move, Enter answers, a click answers;
    Esc or a click outside means No.
  - Built as a `render.Layers` popup, with the summary card's buttons.
- **D8. Never while work runs.** No question opens while a turn or agent
  runs (a waiting approval counts as part of the turn), or while a Source
  Control or branches git operation runs. Instead the input row says "Not
  now: the turn is still running · Ctrl+C interrupts it", or "Not now: the
  commit is still running". Nothing is interrupted or answered.
- **D9. The return.**
  - Yes closes the CLI as Ctrl+Q does: stdin closed, then a forced kill
    after 5 s and 5 s more.
  - Once the CLI has exited, the session's part of `Shutdown()` and
    `closeSession()` runs. Then Back's fresh state, with the answers kept,
    shows the home screen.
  - The draft is saved for the project's next start (plan 33). The turn
    review's store goes, as when quitting.
  - After a crash, Yes returns at once. Never in a mock run.
  - The failed-start card's Esc stays the direct back it is today. Ctrl+Q
    while the session is closing still quits.
- **D10. Esc on the home screen** no longer quits; only Ctrl+Q does.
- **D11. The rules.**
  - In `events-and-state.md`, "No transition leads from `Ready` back to
    `Asking`" becomes: each session is one CLI and one project, and one run
    of ACC can hold several sessions, one after another.
  - `AGENTS.md`'s MVP scope list gains both features.
  - The Nice to have item "A list of a project's earlier sessions to
    resume" leaves `plan.md`.

#### Steps

1. **Session History** (D1 to D6; criteria 75.1 to 75.9; that step's
   mutants of 75.17).
2. **Return to Home** (D7 to D11; criteria 75.10 to 75.16; that step's
   mutants).
3. **Verification:** 75.18 and the live run L75, of which the owner is
   told first. Then the final review on the diff from `aa77d6e`.
4. **Records and report, then stop.** No staging, commit or push without
   the owner. After commit and push, the plan's temporary files are
   cleaned up.

Work stops at once on anything unexpected.

#### Live run L75

Its approval comes with the plan's.

- **Setup:** both CLIs, in ACC's 9 pt profile maximised, in a new
  temporary project under `%LOCALAPPDATA%\Temp`, removed afterwards; the
  owner's home folder and the CLIs' own logins, with the window's
  `CLAUDE*` and `ANTHROPIC_*` variables removed and
  `ANTHROPIC_MODEL=haiku`, Codex on its configured model; `confcheck
  snapshot` before and `compare -cli-ran` after; `projects.json` copied
  before and put back after; the owner's own Claude Code and Codex
  terminals closed first, and nothing else using the machine (the keys and
  clicks are real).
- **For each CLI:** start a session, and run a turn that gives a code
  word; Esc, Yes: the home screen, with no CLI process left; Alt+H: the
  entry, with its model and title; Enter: the session resumes; a turn
  asking for the code word, which comes back; Esc, Yes; then Ctrl+Q.
- **Usage:** two Haiku turns and two Codex turns, plus the CLIs' local
  usage reads.
- **Stops at:** a screen not expected, a CLI process left after a return,
  or a configuration difference.

#### Criteria

- **75.1 The record.**
  - A recorded session goes first in its project's `history`; the same ID
    recorded again moves first, with its new end time and model, its title
    kept. A 21st entry drops the oldest.
  - A mock run, a session without an ID, or a turn the CLI opened records
    nothing.
  - The file is written as version 2. A version 1 file reads with each
    CLI's last session as an entry (model "N/A", no title), its paths,
    times, favourites and order unchanged, and is written as version 2 at
    its next change. A version 3 file is refused and left byte for byte.
  - At `aa77d6e`, `Load` refuses a version 2 file and `Update` then writes
    nothing (read in that commit's code).
  - A project dropped from the list takes its history with it.
  - The title follows D1a: from the first message of a session started
    new, as the CLI received it; set once, a resume never changing it;
    control characters removed, whitespace runs one space, ends trimmed,
    at most 60 grapheme clusters, none cut; none for migrated entries.
- **75.2 Unchanged.** Ctrl+R and START SESSION behave as before, plan 69's
  and plan 70's tests passing unchanged, except where a refusal or a
  deletion removed the newest entry: Ctrl+R then offers that CLI's
  previous entry, as documented. Plan 69's store test,
  `TestProjectSessions`, follows version 2's format, every check kept (the
  owner's ruling of 2026-11-03).
- **75.3 Opening.** Alt+H opens the view for the resolved path. It opens
  nothing while a start is validated, from the Worktrees view, or with a
  path Start refuses, and in that last case the field shows the refusal.
  Golden frames at 80x24, 120x40 and 200x60, with entries and with none.
- **75.4 The rows.** Newest first. Each entry shows its title, then in
  grey its date and time, age, CLI, model ("N/A" when not recorded) and
  the ID's first 8 characters; an entry without a title the grey line
  alone. A selection beyond the visible rows scrolls the list, with the
  scroll mark.
- **75.5 Keys and mouse.** Each key and click of D3 has its test. A right
  click changes nothing. While the view is open, Ctrl+N, Ctrl+O, Ctrl+R
  and Ctrl+W do nothing, and Ctrl+Q quits.
- **75.6 Resume.**
  - Enter and a double click start the entry's CLI on that session:
    `--resume <id>` for Claude Code, `thread/resume` with
    `"excludeTurns":true` for Codex, and question 1 then shows that CLI.
  - A CLI not installed starts nothing, and says so.
  - After a refused resume, the entry is gone, and Ctrl+R then offers that
    CLI's previous entry, if any.
- **75.7 Delete.** Del then Delete removes that entry only. Shift+Del then
  Delete empties that project's `history`. Every other project, the
  favourites and their order stay byte for byte (the file is compared).
  Cancel, Esc or a click elsewhere removes nothing, and Cancel is selected
  when the question opens.
- **75.8 The CLIs' data.** Nothing the plan adds opens, writes or removes
  anything under `~/.claude` or `~/.codex`. Checked by searching the diff,
  and by the live run's configuration check.
- **75.9 The footer.** "Alt+H History" sits between Ctrl+O and Ctrl+Q in
  both modes. In picture mode the first three groups stay where they are.
  Grid mode uses the rows of D6. The startup golden frames differ only on
  the footer row.
- **75.10 Esc unchanged.** Every existing use of Esc on the dashboard
  keeps its effect, one test row each: popups, cards, inspection, details,
  summary, entries, search, docked panels, selections, zoom, focus back to
  zone B, and the failed start's back. Only the Esc that did nothing opens
  the question.
- **75.11 The question.** Text, buttons and default as in D7; ←/→ move,
  Enter answers, Esc or a click outside means No. Golden frames at all
  three sizes.
- **75.12 Never while work runs.** During a turn, an agent, a waiting
  approval, a commit, push, stage, switch or fetch, Esc writes nothing to
  the CLI, interrupts nothing and opens no question. The input row shows
  D8's texts.
- **75.13 The return.** With the session idle, Yes closes stdin with
  quit's escalation. The home screen shows once the process has exited,
  with the answers kept and the session in the history. After a crash it
  comes at once. In a mock run there is no question. Ctrl+Q while the
  session closes quits.
- **75.14 Nothing left behind.**
  - After a return, the state equals `state.New()` except for the startup
    answers.
  - Everything in the model tied to the session equals a fresh model's:
    the session, panels, popups, views, search, Source Control, Git Graph,
    turn-review store, notification, attachments and lock.
  - The draft is saved, and comes back with the next start in that
    project.
  - A second session draws only its own entries in zones A, B and C.
- **75.15 Esc on the home screen** does nothing, and Ctrl+Q quits. The
  startup machine's table says so.
- **75.16 The machines.** Each new or changed row in `events-and-state.md`
  has its test, and TestStateMachines passes.
- **75.17 Broken on purpose.**
  - Run through `go test -overlay`: the repository is never written, and
    its code is hashed before and after.
  - The mutants: Delete All reaching another project; a delete reaching
    another entry; a 21st entry kept; the migration dropping a favourite
    or an entry; a title over 60 grapheme clusters, cut inside one, or
    keeping a control character; a resume changing a title; D8's refusal
    removed; the return leaving the notification, the Git Graph's read or
    the turn-review store behind; Esc quitting on the home screen; Yes as
    the question's default.
  - A test fails for each of them.
- **75.18 The checks.**
  - Build, vet (Windows and Linux) and the whole test suite exit 0, and
    `gofmt -l .` prints nothing, after the last code edit.
  - `-race` on the changed packages.
  - The keystroke timing step alone, on mains power, within its budgets.
- **75.19 The live run.** Every item of L75 passes.
- **75.20 The documents.** `ui-design.md`, `events-and-state.md`,
  `reliability.md`, `AGENTS.md`, `plan.md` and amendments 89 to 91 describe
  what the code does. No BOM, CR or Windows user name in the plan's files.
- **75.21 The scope.** Against `aa77d6e`, only the plan's files differ.
- **75.22 The owner's judgement.** How the view, the question and the
  footer look at full size, from L75's screenshots: the owner's to judge,
  not verified by the orchestrator.

#### Known limits

- Sessions from before this plan: only each CLI's last one per project
  appears, without a title.
- A resumed session shows none of its earlier messages (plan 69).
- Claude Code may have deleted an old session already: its
  `cleanupPeriodDays` setting defaults to 30 days, according to its
  settings documentation. The resume then fails, the entry goes, and
  Ctrl+R offers the previous one, which may be gone too.
- Two dashboards on the same session are not detected.
- An older ACC build shows the project list as unavailable and records
  nothing until it is updated; going back to an older build loses the list
  there.
- A title can hold sensitive text from a first prompt, kept until its
  entry is deleted, after Claude Code's own cleanup (accepted, amendment
  91).
- When a session ends, its turn review and anything the CLI still runs in
  the background end with it, as when quitting.

#### Progress

- Steps 1 to 3: done. The final review, at `xhigh`, gave PASS on the 21
  criteria it judged, after its one correction; the owner approved 75.22
  (`doc/plan75-evaluation.md`).
- Step 4: the records written (`doc/brief.md`, amendments 92 to 94;
  `doc/plan75-evaluation.md`), after the owner's one-off fix of the delete
  question's arrows. Waiting for the owner's decision: no staging, commit
  or push without it.

## The reports, word for word

From each Reviewer's own hand-back in its transcript; a Windows user name is replaced by `<user>`.

### Step 1, its first review

`a1b61db1def468daf`

````text
{
  "message": "Review - plan 75, step 1 (Session History) - 10 criteria: 7 PASS, 2 FAIL, 1 NOT VERIFIED\n\nPASS 75.1  The record [read]: `RecordSession` (internal/input/projects.go:230-254) puts the new entry first. The same CLI and ID moves first with the new end time and model, keeps the old model when the new record has none, and keeps its title (:243-246). The list is capped at 20 (:251), and a project the list no longer has is never added back (:232-235). Nothing is recorded unless the run is not a mock, the turn is not the CLI's own and the session has an ID (internal/state/reduce.go:971); plan 69's `TestRecordSession` (state/resume_test.go:14-62) is unchanged. `Save` writes version 2 (projects.go:170). `Load` folds a version 1 file's sessions into entries with no model and no title, sorted newest first, and leaves paths, `used`, favourites and `SortProjects` alone (:121-123, :135-157). Any other version is refused (:114-116), and `Update` returns before `Save` (:196-200). At aa77d6e, `Load` refuses any version but 1 (orchestrator.txt:91-93) and `Update` returns on that error before `Save` (:139-143); every app write goes through `Update` (internal/app/projects.go:45, not in the diff). History is a field of `Project`, so `Forget` and `capped` drop it with the project. Title: it is taken from the first accepted message only when `Startup.Resume == \"\"` (reduce.go:113-121), a crash restart keeps it (actions.go:223), and `SessionTitle` (render/history.go:91-112) removes controls, turns white-space runs into one space, trims, and cuts at 60 grapheme clusters through `next`. Migrated entries get no title. The input, state, render and app tests are [reported] passing.\n\nFAIL 75.2  Unchanged: plan 69's store test `TestProjectSessions` (\"criterion 69.2 of plan 69 for the store\") does not pass unchanged. It was rewritten: internal/input/projects_test.go:121-128 says \"Adapted from plan 69's\", :134-136 change the `RecordSession` call shape, :142-147 replace `want` (a map) with `[]Entry`, :189 asserts `\"version\": 2,` / `\"history\": [` where the baseline asserted `\"version\": 1,` / `\"sessions\": {`, and :196 adds a History check. The criterion allows one exception (a refusal or a deletion removing the newest entry), and these changes go beyond it.\n- The file-format assertions cannot pass under D1's version 2 with any implementation. The criterion as worded conflicts with D1 there, so this needs a ruling, not just code.\n- The call-shape change was the Implementer's choice.\n- I found no weakening: every old assertion has an equivalent or stronger one.\n- Plan 69's app tests (lastsession_test.go) and plan 70's tests are untouched. They still exercise the history through the `Sessions` maps that `Save` folds in and the `LastSession` wrapper.\n\nFAIL 75.3  Opening (confidence medium): the field holds a path that `checkProject` refuses, for example a folder that no longer exists, a file, or an unreadable folder (internal/app/validate.go:46-59, \"Does not exist: …\"). Alt+H then opens the view (internal/app/history.go:38-42 checks only `input.ResolvePath`'s Problem), and the field shows no refusal, since `PathProblem` comes from `ResolvePath` alone (app.go:815-816). The criterion says it \"opens nothing … with a path Start refuses, and … the field shows the refusal\".\n- If \"Start refuses\" means only the resolution's refusal (D2: \"as Start resolves it\"), this passes [read]: history.go:35-41, `TestHistoryOpen`, and the golden frames history-*/history-none-* at 80x24, 120x40 and 200x60.\n- Validating, Worktrees and resolution refusals behave as the criterion says.\n\nPASS 75.4  The rows [read]: the grey line is date and time, `age()` (the PROJECTS function), CLI, model or \"N/A\", and `ShortID` (history.go:246-252). The title row exists only when the entry has a title (render/history.go:169-172). The list centres on the selection and draws the slim ▐ mark (render/history.go:212-251). testdata/frames/history-80x24.txt lines 7-19 and history-scroll-80x24.txt lines 7-19 (mark on 13-17) match.\n\nPASS 75.5  Keys and mouse [read]: every key but Ctrl+Q (keys.go:39-44) goes to `historyKey` (keys.go:88-90), which has no case for Ctrl+N, Ctrl+O, Ctrl+R or Ctrl+W (history.go:60-91). A paste is dropped (app.go paste switch, `m.hist` case). The mouse goes to `historyMouse` (mouse.go:47-49); a right click returns at history.go:188. Each D3 key and click has a row in `TestHistoryKeysAndMouse`, `TestHistoryOpen`, `TestHistoryResume` or `TestHistoryDelete` [reported].\n\nPASS 75.6  Resume [read]: if the entry's CLI is not installed, a note appears and nothing starts (history.go:155-157). Otherwise `ui.Choice` is set to the entry's CLI and `Confirm` runs with Resume set to the entry's ID, which is plan 69's path (history.go:160-162). A refusal runs forget-session by CLI and ID (lastsession.go:102-105, projects.go:260-275), and `lastSession` skips the refused ID (lastsession.go:33-38). Covered by the four `TestHistoryResume` cases: `--resume`, `\"excludeTurns\":true`, not installed, refusal then Ctrl+R [reported].\n\nPASS 75.7  Delete [read]:\n- Cancel is selected when the question opens (history.go:100).\n- Esc cancels (:112-113), and a click outside the box cancels (:203-205).\n- Delete removes by CLI and ID through `ForgetSession`, or empties the project through `ClearHistory` (projects.go:279-287). Both run on the re-read file through `Update` (history.go:131-141).\n- The whole file is compared in app history_test.go:597-630 and input projects_history_test.go:227-284 [reported].\n\nPASS 75.8  The CLIs' data [read]: I searched changes.diff and the new files. `.claude` / `.codex` appear only in plan.md and doc text. No home-directory access, open or remove appears outside projects.go's own temporary file. `TestHistoryNoCLIData` checks the same. The live run's configuration check is L75's (step 3) and is not judged here.\n\nPASS 75.9  The footer [read]: `homeKeys` is New, Open, History, Quit (home.go). Grid mode shows the 99-cell full row from 100 columns and the 65-cell short row below that (homegrid.go). In picture mode the first three entries of footCaps/footKeys/footActions/footBars are the baseline values, with a fourth group added (homepaint.go:835-838; diff lines 1135-1142). Each startup-*.txt frame's diff is one hunk on its last row only.\n\nNOT VERIFIED 75.17  Broken on purpose:\n- What the evidence shows [reported]: the overlay commands, and all 10 step-1 mutants caught. The tests named do assert each mutant's behaviour.\n- What it does not show: the before/after hashes are of \"index\" and \"diff\", which, going by their names, cover git's index and the tracked files. They would not cover the new untracked files (internal/render/history.go, internal/app/history.go …), and render/history.go is the file mutants 6-8 target.\n- What would settle it: re-run the mutants with a hash over every file of plan-files.txt plus `git status --porcelain=v1 -uall`, taken before and after, and show the two equal.\n\nObservations:\n- Medium severity, high confidence. Alt+H works while detection is still running (history.go:35 excludes only Validating), but `Startup.Installed` is empty until `DetectionDone` (actions.go:128). So every row is dimmed and Enter says \"✗ Claude Code is not installed\" even when it is installed.\n- Low, high. The Ctrl+N check in `TestHistoryKeysAndMouse` cannot fail. The field is the starting folder (homeRig `Cwd: dir`), and Choice and focus are not checked (history_test.go:335-341). The code is right by reading.\n- Low, high. internal/input/projects_history_test.go:269 calls `ClearHistory(l, \"c:/WORK/a\")`, which matches only with Windows' case and separator folding, so the test fails on Linux. projects_test.go:162 guards the same kind of check with `runtime.GOOS`.\n- Low, high. A version 2 entry with an empty ID (only possible in a hand-edited file) is listed, and Enter sends `Confirm` with Resume \"\", which starts a new session (history.go:154-162). `Newest` skips such entries (projects.go:294); the view does not.\n- Low, high. While the delete question is open, the keys row still says \"↑↓ move · Enter resume · Del delete …\" (render/history.go:261-278).\n- Low, medium. After the view resumes an older entry and the CLI refuses it, the note says \"no longer has its last session here\" (lastsession.go:71).\n- Low, medium. The Delete All question counts N from the rows shown, but `ClearHistory` runs on the re-read file and can remove more than N (history.go:131-134, render/history.go:67).\n- Low, medium. At exactly 99 columns the short row shows, although the 99-cell full row would fit (`homeFooterFull = 100`, home.go). This follows the existing keep-the-last-column-free rule and the doc records it.\n- Low, medium. The view's keys and mouse go through `m.hist != nil` checks, not `render.Layers`' Top (keys.go:88-90, mouse.go:47-49), as the Worktrees view already does.\n\nRequested tests:\n- The 75.17 hash re-run described above.\n- A field holding a missing directory: Alt+H should open nothing and the field should show \"Does not exist: …\", if 75.3 is read literally.\n- Alt+H then Enter during Detecting, after deciding what it should do.\n- The Ctrl+N row with the field away from the starting folder, and Choice and Field checked.\n- `go test ./internal/input -run TestHistoryDelete` on Linux.\n\nVerdict: FAIL"
}
````

### Step 1, its second review

`a64f2d7fd907a77f1`

````text
{
  "message": "Note first: inside changes.diff I read part of the plan.md hunk (diff lines 1368-1498) before I skipped the rest of it. That part held only the plan's header, its file list and designs D1 to D5, the same text as criteria.txt. I saw no earlier verdict and no account by the Implementer.\n\nReview - plan 75, step 1 - 10 criteria: 8 PASS, 1 FAIL, 1 NOT VERIFIED\n\nPASS 75.1 The record [read]:\n- Load accepts versions 1 and 2 and refuses any other (internal/input/projects.go:114).\n- A version 1 file's sessions become history entries through settled() (projects.go:135-159), with no model and no title. Paths, times, favourites and SortProjects' order are unchanged. Save always writes version 2 (projects.go:170).\n- RecordSession (projects.go:230-251) puts the entry first. The same CLI and ID moves first with its new end time and model, keeps its old title (line 243), and the list is capped at MaxHistory (line 251).\n- The record condition (mock run, no ID, turn the CLI opened) is unchanged (internal/state/reduce.go:971). Plan 69's TestRecordSession, unchanged, covers it.\n- At aa77d6e: Load refuses `pf.Version != 1` (orchestrator.txt:91-93), and Update returns before Save on that error (orchestrator.txt:139-143).\n- History is a field of Project, so Forget and capped drop it with its project.\n- The title text is the Draft, taken once, only when Startup.Resume is \"\" (reduce.go:113-121). The Draft is the same text CmdWrite sends (state/actions.go:377, 402). The crash resume keeps it (actions.go:201).\n- SessionTitle (internal/render/history.go:91-112): spaces mapped, Cc removed, Fields-joined, cut at 60 clusters with the cluster-aware next(), then trimmed.\n- Tests: TestProjectsMigration, TestProjectsVersion3, input and app TestHistoryRecord, TestSessionTitle, TestFirstPrompt and TestFirstPromptNone.\n\nPASS 75.2 Unchanged [reported]:\n- lastsession_test.go and plan 70's tests are not in the diff. Their lists in version 1 shape (`Sessions:` maps) pass through Save's settled().\n- lastSession now takes Newest(choice, refused ID) (internal/app/lastsession.go:22-39). confirmStart is unchanged.\n- TestProjectSessions (changes.diff:1018-1094): every check is rewritten from Sessions to History and none is removed; `History != nil` is added to the check of an older build's file.\n- That the tests pass rests on the evidence only.\n\nPASS 75.3 Opening [read]:\n- openHistory opens nothing outside Asking, with Worktrees open, or with a path whose resolution is refused (internal/app/history.go:48-63). Otherwise it checks the path on disk in a Cmd, with checkProject, Start's own check.\n- historyChecked keeps the refusal (history.go:68-77), and render/home.go:176-178 shows it under the field.\n- Worktrees takes the keys first (internal/app/keys.go:85-90).\n- The frames history-{80x24,120x40,200x60} and history-none-* exist and are listed at internal/app/golden_test.go:651-652.\n- Tests: TestHistoryOpen, TestHistoryStartRefuses, TestHistoryDetecting.\n\nPASS 75.4 The rows [read]:\n- history.go:292-298 builds the grey line: \"2006-01-02 15:04\", age() as PROJECTS writes it, CLI name, model or \"N/A\", ShortID (first 8 characters).\n- render/history.go:150-173 adds the title row only when the entry has a title.\n- The scroll and the ▐ mark: render/history.go:212-251.\n- Tests: TestHistoryRows, TestHistoryScroll; frames history-scroll-*.\n\nPASS 75.5 Keys and mouse [read]:\n- Every key and click of D3 has a test:\n  - TestHistoryKeysAndMouse: ↑, ↓, PgUp, PgDn, Home, End, click, right click, wheel, double click, click outside, Ctrl+N/O/R/W, Ctrl+Q.\n  - TestHistoryOpen: Esc. TestHistoryResume: Enter. TestHistoryDelete: Del and Shift+Del.\n- In the code, a click that is not a left click is ignored (history.go:221-224). The view takes the keys before Ctrl+W (keys.go:88-90), and Ctrl+Q is handled first (keys.go:39-44).\n\nFAIL 75.6 Resume:\n- A double click on an entry of a list that scrolls resumes a different entry.\n- Case: 120x40, 20 entries, selection 0. Entry e10's grey line is at list row 17.\n  - The first press selects e10 (history.go:249-253).\n  - The next frame centres the list on it again: `first` is recomputed from Sel on every frame (render/history.go:215-231) and goes from 0 to 2.\n  - The second press, on the same cell within 400 ms (commit.go:772-780), reads the item now under that cell, e12's title row (history.go:226). It sets sel to 12 and resumes it (history.go:254-256).\n  - Claude Code starts with `--resume c-12-21000012`, not c-10-21000010.\n- At 80x24 this happens from 9 entries. In the history-scroll frame, a double click on e11's title (row 8) resumes e8.\n- This breaks 75.6 (\"a double click start[s] the entry's CLI on that session\") and D3.\n- TestHistoryKeysAndMouse's double click (history_test.go:374-380) is on a list that does not move, so it cannot catch this.\n- The other parts pass by reading (history.go:180-199; lastsession.go:22-39 and 64-79; TestHistoryResume): Enter, Claude Code `--resume`, Codex thread/resume with excludeTurns, question 1 set, CLI not installed, refusal followed by the previous entry.\n\nPASS 75.7 Delete [read]:\n- The question opens with Cancel selected (askSel 0; history.go:129-140).\n- Delete removes by CLI and ID, in that path's project only (history.go:161-174; projects.go:268-289).\n- Esc cancels (history.go:151-153). A click outside the box cancels (history.go:237-239).\n- App TestHistoryDelete compares the whole file with Save of the expected list after every step. A favourite and another project share the deleted IDs. Input TestHistoryDelete covers the store.\n\nNOT VERIFIED 75.8 The CLIs' data:\n- The diff and the new files have no `.claude`, `.codex` or UserHomeDir reference. The only removals are of the store's own temporary file (projects.go:184, 188).\n- The criterion's second check, the live run's configuration check, is not in the evidence. It is settled by the configuration check's output before and after a live run.\n\nPASS 75.9 The footer [read]:\n- Order N, O, H, Q (render/home.go:288).\n- Grid mode: the full row (99 cells) from 100 columns, the short row (65 cells) below (render/homegrid.go:177-198; render/home.go:294).\n- Picture mode: the first three footCaps positions are unchanged (render/homepaint.go:835).\n- Each startup frame hunk changes only the last row (changes.diff:1718-1927).\n- Test: TestHomeFooter.\n\nPASS 75.17 Broken on purpose, this step's mutants [reported]:\n- evidence.txt:25-108 lists 11 mutants, run through overlays and each caught. The fingerprint, index and diff hashes are equal before and after.\n- The mutants cover every one this step names: Delete All reaching another project, a delete reaching another entry, a 21st entry kept, the migration dropping a favourite or an entry, the three title faults, a resume changing a title.\n- The mutant sources are outside the repository and the brief's folder, so I did not read them. Tests that would catch each kind exist.\n\nObservations:\n1. Medium, high: in a scrolling list a single click moves the list under the cursor, because the list is centred on the selection again every frame (render/history.go:215-231). This is the root of the 75.6 failure. Either keep a scroll position that moves only when the selection leaves the view, or resume only when the second press lands on the entry the first one selected.\n2. Medium, medium: doc/architecture.md:822 still describes version 1: `Project.Sessions` as the store's field, ForgetSession \"only while the record names the refused session\", LastSession. No table names plan 75's parts (history.go, LayerHistory, Entry, Newest, ForgetSessions). This goes against \"Documents hold the latest state\" (AGENTS.md:397). The plan does list architecture.md among its files, possibly for later.\n3. Low, high: D6's short row as quoted, single spaces, is 57 cells, but D6 says 65 cells. The code draws the 65-cell row with the key caps' padding: it follows the count, not the literal text.\n4. Low, medium: the 99-cell full row appears only from 100 columns (render/home.go:294). At exactly 99 columns it would fit, as D6 words it.\n5. Low, medium: Delete All counts only entries with an ID (history.go:82-88, 136), so an ID-less entry from a hand-edited file survives \"delete all\". The migration also drops a version 1 session with an empty ID (projects.go:142).\n6. Low, medium: a delete question withdrawn because the list changed under it (history.go:281-283) leaves the next Enter to the list, which resumes the selected entry (history.go:117-118). An Enter meant as an answer then starts a session.\n7. Low, medium: SessionTitle runs over the whole first prompt, on the event loop, at every turn end (lastsession.go:98). A multi-megabyte first paste pays that cost each turn; a bounded prefix would be enough.\n8. Low, low: unicode.IsControl matches Cc only, so titles keep Unicode format characters: bidi controls (U+202A to U+202E) and zero-width spaces (render/history.go:92-100). visible() does not show them, unlike revealed() on the approval cards. No escape sequence gets through.\n9. Low, medium: Load does not cap an oversized version 2 history (settled() returns early when Sessions is empty); only RecordSession caps it.\n10. Low, high: TestHistoryNoCLIData (history_test.go:918-931) only searches source text. It proves no behaviour.\n\nRequested tests:\n1. 120x40, histEntries(20), view open, selection 0: two MouseClickMsg on the same cell of HistoryItemRow+10 within 400 ms. Expect one Claude Code spawn with `--resume c-10-21000010`. The same at 80x24 with 9 or more entries, on a row away from the selection.\n2. For 75.8: the live run's configuration check output, before and after.\n3. Open Delete All, change m.projectList so the question is withdrawn, then send Enter: expect no spawn, or document that a resume is intended.\n\nVerdict: FAIL"
}
````

### Step 1, its third review

`a263c39073f99486f`

````text
{
  "message": "Review - plan 75, step 1 - 10 criteria: 9 PASS, 1 FAIL, 0 NOT VERIFIED\n\nProcess note: changes.diff includes plan.md's diff (lines 1453-1802). I read lines 1453-1699, which hold the plan's design and criteria and match criteria.txt, and read nothing past that. In what I read I saw no earlier verdict on this work and no account of it by the Implementer.\n\nFAIL 1  75.1 The record (title \"at most 60 grapheme clusters, none cut\"): if the 60th kept cluster of the first message is followed by white space and more text, the title is not cut at 60. Two examples. `SessionTitle(\"Add a history view to the home screen with resume and delete entries please\", false)` returns all 76 clusters (\"delete\" ends on character 60). `SessionTitle(strings.Repeat(\"x\", 60)+\" yyy\", false)` returns 64 clusters. The cause is in C:\\javadev\\projets\\agent-command-center\\internal\\render\\history.go:113-128. The collapsed space and the next character are appended in the same iteration, so the inner loop (119-125) completes the 60th cluster and the space together and `done` goes from 59 to 61. The check `if done == TitleChars` (126) then never fires, the function reads the whole FirstPrompt (up to MaxFirstPrompt, 8 KiB), and returns all of it, or all but its last cluster when partial (130-131). C:\\javadev\\projets\\agent-command-center\\internal\\app\\lastsession.go:98 stores that string as the entry's Title in projects.json. Drawing hides the problem, because rows are elided. This breaks 75.1 and D1a. It also breaks the owner's accepted limit (amendment 91, \"the first 60 characters\"; plan.md's standing decision), which reliability.md and ui-design.md also give as \"60 grapheme clusters at most\". Probable fix: test `done >= TitleChars`, or check inside the inner loop; in the jump case the extra cluster is the space, which TrimSpace removes. The rest of 75.1 passes by reading:\n- same ID moves first, title kept, model kept when the new record has none: projects.go:247-271; 21st entry dropped: 268;\n- mock run, no ID, turn the CLI opened: reduce.go:990;\n- written as version 2: projects.go:187; migration: 121-122 and 152-174;\n- version 3 refused: 114-116, and Update returns before Save: 213-217;\n- at aa77d6e, Load refuses any version but 1 and Update returns before Save (orchestrator.txt, Load 91-93, Update 140-143);\n- a dropped project takes its history: Forget and capped drop the whole struct, and RecordSession never adds a project back (250);\n- title set once: projects.go:260-263, reduce.go:132-140; migrated entries have Title \"\".\n\nPASS 2  75.2 Unchanged [reported]: Plan 69's and plan 70's test files are not in the diff. The only test edits are:\n- golden_test.go:591, the footer assertion;\n- TestHomeGrid's footer row;\n- TestProjectsStore, version 2 changed to 3;\n- TestProjectSessions, every assertion kept, mapped from Sessions to History, and the file check made stricter (\"no sessions\") (projects_test.go:129-216 against the diff's removed lines).\n\nCtrl+R offers the newest entry, skipping the refused one: lastsession.go:22-39 and projects.go:308-315. TestHistoryResume covers the refusal and TestHistoryCtrlR the deletion. The suite passes per evidence lines 2-24.\n\nPASS 3  75.3 Opening [read]:\n- openHistory (app/history.go:50-65) opens only in Asking, never from Worktrees (keys.go:85-87 takes the key first), and returns nothing on a resolution problem, which the field already shows;\n- checkProject runs in a command (validate.go:40);\n- a refusal goes to OpenProblem (history.go:74-76), shown at render/home.go:176-178 and cleared once the text changes (app.go:842-845);\n- golden frames history-* and history-none-* exist at 80x24, 120x40 and 200x60.\n\nTests: TestHistoryOpen, TestHistoryStartRefuses.\n\nPASS 4  75.4 The rows [read]:\n- the grey line is date \"2006-01-02 15:04\", age, CLI name, model or \"N/A\", and ShortID (8 characters, reduce.go:362) (app/history.go:285-291);\n- the title row exists only when Title != \"\" (render/history.go:253-256);\n- the top moves only as the selection needs (191-222), with the slim mark (292-303);\n- frame history-scroll-80x24 shows both.\n\nPASS 5  75.5 Keys and mouse [read]:\n- Ctrl+Q is handled before the view (keys.go:39-44);\n- every other key goes to historyKey (keys.go:88-90), which binds only D3's keys (history.go:103-124), so Ctrl+N, O, R and W do nothing;\n- paste is dropped (app.go:1255);\n- right click returns at history.go:223, the wheel is at 209-221, a click outside closes at 245-247, and a double click on the same entry resumes at 258.\n\nEvery D3 key and click is tested in TestHistoryKeysAndMouse (see the observation on its Ctrl+R check).\n\nPASS 6  75.6 Resume [read]:\n- historyResume (history.go:181-200) checks that the CLI is installed, else sets the note and starts nothing;\n- it sets Choice to the entry's CLI and Confirms with Resume and ResumeProject, which goes through plan 69's start with Last (actions.go:275-279);\n- a refusal removes that CLI and ID (lastsession.go:106-109, projects.go:277), and lastSession skips the refused ID until the store catches up.\n\nTestHistoryResume checks the `--resume` argument, `\"excludeTurns\":true`, the not-installed note and the refusal.\n\nPASS 7  75.7 Delete [read]:\n- the question opens with Cancel selected (history.go:136);\n- ←/→, Enter and Esc are at 145-156, and clicks at 233-243;\n- deletion is by path and by CLI plus ID (projects.go:285-303).\n\nThe file is compared byte for byte in app/history_test.go:538-645 and input/projects_history_test.go:228-295.\n\nPASS 8  75.8 The CLIs' data [read]: None of these appears in the diff's code hunks or in the new non-test files: `.claude`, `.codex`, UserHomeDir, os.Remove, RemoveAll, os.Open, os.Create, WriteFile, thread/list. The only hits are a pre-existing test line (1145) and plan.md text. The plan's code writes only projects.json through Save. Alt+H's disk check is Start's read-only checkProject.\n\nPASS 9  75.9 The footer [read]:\n- homeKeys order is Ctrl+N, Ctrl+O, Alt+H, Ctrl+Q (render/home.go:288);\n- grid mode has the full row of 99 cells and the short row of 65 (homegrid.go:195-214; TestHomeFooter);\n- in picture mode the first three slots are unchanged (homepaint.go:835-838);\n- every startup frame's hunk (changes.diff 1803-2013) changes only the last row.\n\nPASS 10  75.17 Broken on purpose (step 1's mutants) [reported]: Evidence lines 25-144:\n- the runs use overlays under %TEMP%;\n- the fingerprint, index and diff hashes are equal before and after (26-28, 140-143);\n- mutants m1, m2, m4, m5, m6, m7, m8, m9, m16 and m17 cover step 1's list, and each is caught by a named test.\n\nThe mutant sources are outside the brief's folder and were not read.\n\nObservations:\n- Medium, high confidence: TestSessionTitle cannot catch FAIL 1. Its \"ends trimmed after the cut\" case (render/history_test.go:32, 59 x then space) and its \"a long message\" case (58, \"word \" repeated) both put the 60th cluster on a space. Mutant 7 is caught, yet the real code exceeds 60 clusters. Also, the function's comment \"reads text no further than the TitleChars clusters need\" (history.go:99-100) is false in that case.\n- Medium, high confidence: Alt+H opens nothing during Detecting (app/history.go:51, TestHistoryDetecting). D2 approved only \"except while a start is being validated\". The added exception is documented in ui-design.md but was not in the approved design.\n- Low, medium confidence: SessionTitle runs synchronously on the event loop at every turn end (lastsession.go:98), and `next(string(out), start)` copies `out` on every call (history.go:120). With FAIL 1's input that is about 8K copies of up to 8 KiB per turn end. A first message that is one long combining-mark cluster also re-scans the growing cluster on every rune and caches each prefix in clusterWidth (cluster.go:279-291, up to 4096 keys of up to 8 KiB): quadratic lipgloss.Width work.\n- Low, high confidence: the grid footer switches to the short row under 100 columns (render/home.go:1193, homegrid.go:197). D6 says the full row is used \"where it fits (99 cells)\", so at exactly 99 columns the short row is drawn although the full row fits.\n- Low, high confidence: the Ctrl+R check in TestHistoryKeysAndMouse cannot fail. The field is `away`, a folder with no history, and Codex is chosen, so a Ctrl+R passed through would also start nothing (app/history_test.go:338-346). The behaviour holds by reading keys.go:88-90.\n- Low, high confidence: Delete All removes the entries it counted (history.go:137, 172-174). If another dashboard adds an entry in between, the history is not empty, which 75.7 literally asks for. This is a documented choice.\n- Low, medium confidence: if F12 turned the mouse off before Alt+H, the view has no mouse and F12 does nothing inside it (historyKey ignores f12). ui-design says \"the view keeps it on\".\n- Very low: Unicode tag characters (U+E0020-E007F, category Cf) are stripped from titles, so a subdivision flag becomes a bare black flag (render/history.go:110).\n\nRequested tests:\n- In TestSessionTitle:\n  - `SessionTitle(strings.Repeat(\"x\", 60)+\" yyy\", b) == strings.Repeat(\"x\", 60)` for b false and true;\n  - `SessionTitle(\"Add a history view to the home screen with resume and delete entries please\", false) == \"Add a history view to the home screen with resume and delete\"`;\n  - a loop over n in 55..65 of `strings.Repeat(\"x\", n)+\" \"+strings.Repeat(\"y\", 10)`, asserting at most 60 clusters and that the result is a prefix of the cleaned input.\n- An app-level TestHistoryRecord variant whose first message has its 60th character at the end of a word, asserting projects.json's title is its first 60 clusters.\n- TestHistoryKeysAndMouse's Ctrl+R with the field on `dir` and Claude Code chosen, so that a Ctrl+R passed through would spawn; assert no spawn and the view still open.\n- A timing or allocation bound for SessionTitle on an 8 KiB first message, both plain text and one combining-mark cluster.\n\nVerdict: FAIL"
}
````

### Step 1, its fourth review

`a187589889107187a`

````text
{
  "message": "Note first: I saw no earlier verdict on this plan's work and no account of it by the Implementer. One disclosure: changes.diff includes plan.md's diff (lines 1502-1851), and I did not read that range. A Grep I ran over the whole diff for `.claude`/`.codex` did match three lines inside it (1560, 1587, 1766). They are plan text (D1 and 75.8 wording, plus a scope line), not a verdict.\n\nReview - plan 75, step 1 - 10 criteria: 10 PASS, 0 FAIL, 0 NOT VERIFIED\n\nPASS 75.1 The record [read]:\n- **Record.** `RecordSession` puts the session first. A record with the same CLI and ID moves first with its new end time, and with its new model when it has one. It keeps the existing title (`projects.go` 960-977). The list is capped at `MaxHistory`, so a 21st entry drops the oldest.\n- **What records nothing.** A mock run, a session with no ID and a turn the CLI opened are excluded by the unchanged guard in `reduce.go` `turnEnd` (`!Mock && !ByCLI && ID != \"\"`). Plan 69's `state/resume_test.go` `TestRecordSession` tests this and is unchanged.\n- **Versions.** `Save` writes version 2 (`projects.go:187`). `Load` folds a version 1 file through `settled()`: each entry has no model and no title, and the view shows \"N/A\". Paths, times and favourites are untouched and `SortProjects` is the same. A version above 2 is refused, and `Update` writes nothing. Tests: `TestProjectsMigration`, `TestProjectsVersion3` (the file is left byte for byte, no stray file), `TestHistoryLoad`.\n- **At `aa77d6e`** (orchestrator.txt): `Load` returns an error when `Version != 1` (lines 91-93), and `Update` returns before `Save` (139-143). In the current tree `Save` is still called only from `Update`.\n- **Dropped projects.** A project dropped by `capped`/`Forget` takes its history with it, and `RecordSession` never adds a project back.\n- **Title.** `toRunning` (`reduce.go` 132-139) keeps the text of the first accepted message, only when `Startup.Resume == \"\"`. It is kept through a crash resume (`actions.go:201`), capped at 8 KiB and flagged when cut. The text is `Send.Text`, the editor's full text, which is what the CLI is written. `render.SessionTitle` removes Cc and Cf characters (except ZWNJ, ZWJ and tag characters), turns each whitespace run into one space, trims the ends and keeps at most 60 UAX #29 clusters, none cut. `clusterEnd` is an equivalent extraction from `cluster`. Tests: `TestSessionTitle`, `TestSessionTitleBoundary`, `FuzzSessionTitle`, and app `TestHistoryRecord` (a resume keeps the title; a migrated entry stays untitled).\n\nPASS 75.2 Unchanged [read; that the tests pass is reported]:\n- **Ctrl+R** resumes the newest entry of the chosen CLI (`lastsession.go` 22-39 via `Newest`). After a refusal for good it skips the refused ID and offers the previous entry, as documented.\n- **START SESSION:** `confirmStart` is untouched, and `Record` keeps the history.\n- **Plan 69 and 70 tests:** none of their test files is in the diff. The `startup-resume-*` frames change only on the footer row, which 75.9 allows.\n- **`TestProjectSessions`** (`projects_test.go` 129-216) keeps all 14 of its checks, adapted to `History` and `\"history\": [`, and adds a check that `\"sessions\"` is absent. The `ForgetSession` checks (168-178) are unchanged lines.\n- The evidence shows every package `ok`.\n\nPASS 75.3 Opening [read]:\n- **Opening:** `openHistory`/`historyChecked` (`app/history.go` 50-79) open the view for `checkProject`'s path, the same check Start uses, run in a command. Nothing opens while validating or with the Worktrees view open; from the Worktrees view, `keys.go:85` also routes the key to `worktreesKey` first.\n- **Refused paths:** a path whose resolution is refused already shows `PathProblem`. A path refused on disk sets `OpenProblem`, drawn by `home.go:176` and cleared when the path changes (`app.go:512`).\n- **Tests:** `TestHistoryOpen`, `TestHistoryStartRefuses`, `TestHistoryDetecting`.\n- **Frames:** `history-*` and `history-none-*` exist at 80x24, 120x40 and 200x60 (plus 79x24), and I read them. That they match the current output is reported.\n\nPASS 75.4 The rows [read]:\n- **Order and content:** entries are listed in history order, newest first. The grey line is `2006-01-02 15:04 · age() · CLI · model or \"N/A\" · ShortID` (`app/history.go` 300-306). An untitled entry is drawn as the grey line alone (`render/history.go` 262-265).\n- **Scrolling:** `historyTop` keeps the selection in view, and the `▐` mark appears when not everything fits.\n- **Tests:** `TestHistoryRows`, `TestHistoryScroll`, `TestHistoryTop`, and the `history-scroll-*` frames.\n\nPASS 75.5 Keys and mouse [read]:\n- **Keys:** `historyKey` handles ↑, ↓, PgUp, PgDn, Home, End, Enter, Del, Shift+Del and Esc. Every other key does nothing, and Ctrl+Q is handled earlier in `keys.go:39`.\n- **Mouse:** `historyMouse` does click, double click (same entry), click outside to close, and the wheel one entry per notch. A right click returns before anything else (`history.go:238`).\n- **Tests:** `TestHistoryKeysAndMouse` (including the Ctrl+N, O, R and W loop and the paste), `TestHistoryCtrlRDoesNothing`, `TestHistoryDoubleClickScrolls`.\n\nPASS 75.6 Resume [read]:\n- **Resume path:** `historyResume` sets `ui.Choice` to the entry's CLI and sends `Confirm` with the entry's ID and the project's path (plan 69's path).\n- **Tests:** `TestHistoryResume` checks `--resume <id>`, and `thread/resume` with `\"excludeTurns\":true` and no `thread/start`. A CLI that is not installed dims the row, starts nothing and shows \"✗ OpenAI Codex is not installed\". After a refusal the store holds only the older entry and Ctrl+R resumes it. `TestHistoryOlderRefusal` covers a refused entry that was not the newest.\n\nPASS 75.7 Delete [read]:\n- **What it deletes:** the question asks about the selected entry or all entries, captured by CLI and ID. Cancel is selected first (`historyAsk`), and `ForgetSessions` removes exactly those entries in that project only (`projects.go` 997-1021).\n- **Tests:** app `TestHistoryDelete` compares the whole file with `Save` of the expected list after each step: Cancel, Esc and a click elsewhere change nothing; Delete removes one entry; Delete All empties the history. Input `TestHistoryDelete` covers the same ID under two CLIs and a case-different path. `TestHistoryDeleteAllExact` covers an entry another dashboard adds meanwhile.\n\nPASS 75.8 The CLIs' data, search of the diff [read]:\n- Outside plan.md's diff there is no `.claude`, `.codex`, `UserHomeDir`, `thread/list`, or new `os.Open`/`Remove`/`WriteFile` in non-test code. The only writes are `projects.json` through the existing `Save`. The new `history.go` files have no file I/O.\n- `TestHistoryNoCLIData` encodes this check. The live configuration check is step 3's.\n\nPASS 75.9 The footer [read]:\n- `homeKeys` order is Ctrl+N, Ctrl+O, Alt+H, Ctrl+Q (`home.go:1236` in the diff).\n- Picture mode: `footCaps[0..2]` are unchanged and the fourth group is at the design's spacing (`homepaint.go` 835-838). Grid mode uses the 99-cell full row or the 65-cell short row (`homegrid.go` 1284-1306).\n- Each of the 21 `startup-*` frame diffs changes the last row only.\n\nPASS 75.17 Broken on purpose, this step's mutants [reported]:\n- The evidence shows the run through `go test -overlay`, with the fingerprint, index and diff hashes equal before and after.\n- It reports mutants 1, 2, 4, 5, 6, 7, 8, 9, 16 and 17 caught, matching each listed D1-D6 mutant. The listed tests can fail for such mutants, as I read them.\n- I could not see the mutant sources or recompute the fingerprint.\n\nObservations:\n1. **Footer at exactly 99 columns.** Low, medium confidence. `homeFooterFull = 100`, so the code uses `short := w < 100` (`render/home.go`; `homegrid.go` diff line 1287). In grid mode at 99 columns the short row shows, though D6 says \"the full row where it fits (99 cells)\". The codebase's own `fitHint` keeps a one-cell margin (`source.go:515`). But the \"plan 16\" rule cited in `ui-design.md:479` is nowhere in the docs I searched. The owner should confirm the 100-column threshold.\n2. **Image-only first message.** Low, medium confidence. Such a message sets `Prompted` with empty text, so the session is never titled (`reduce.go` 132-139; test `state/history_test.go` 67-79). This is one reading of \"images do not count\". The other reading, that the next text message gives the title, is also plausible.\n3. **Folder dialog result while the view is open.** Low, high confidence. `openHistory` does not refuse while `m.folderOpen` is set (`app/history.go:51`), and `folderPicked` can then change the field while the view is open (`folder.go:67`). Enter then sends `Confirm{Project: <new field>, ResumeProject: <view's project>}`, and validation refuses: \"Not resumed: the last session is X's, not Y's\". This is safe (no wrong session resumes), but the chosen entry is not resumed.\n4. **`SessionTitle` with `partial`.** Low, high confidence. The last cluster is dropped even when whitespace followed it (\"ab \" gives \"a\"; `render/history.go:126`). The test's reason, \"the cluster before it may still grow\" (`render/history_test.go:51`), is wrong, because a space never extends the preceding cluster. The result is conservative and stays within D1a. It only matters when the first 8 KiB clean to under 60 clusters.\n5. **Late title.** Info, high confidence. `RecordSession` gives an untitled entry a later record's title (`projects.go` 967-970). I found no path where a resume carries text, so \"set once\" holds.\n6. **Deletion when the store cannot be written.** Info. The entry disappears from `m.projectList` at once (`app/history.go:181`) while the file keeps it, and the note shows \"Recent projects not saved\". This is the same pattern as `forgetProject`.\n7. **No feedback during the Alt+H check.** Low. On a slow path, such as an unreachable network share, the press shows nothing until `os.Stat` returns (`app/history.go` 61-64).\n8. **Older tests adapted.** Info.\n   - `TestProjectsStore`'s \"version\" case changed from 2 to 3 (`projects_test.go:56`).\n   - `TestHomeGrid`'s footer row at 80 columns.\n   - The golden footer marker changed from \"Shortcuts:\" to \"Alt+H\" (`golden_test.go:594`).\n   Each follows D1 or D6 and removes no check.\n\nRequested tests:\n- Alt+H while the folder dialog is pending, then a folder chosen while the view is open: the view should either not open, or Enter should resume the entry shown.\n- An owner ruling, or a test that states it, for the grid footer at exactly 99 columns.\n\nVerdict: PASS"
}
````

### Step 2, its first review

`ac139a66f5e537f89`

````text
{
  "message": "Note first: I saw no earlier verdict on this plan's work and no account by the Implementer. I did not read the plan.md part of changes.diff (lines 2493-2843) or the mutant folder C:\\Users\\<user>\\AppData\\Local\\Temp\\acc-mutants-75, which is outside the brief's folder. The repository mentions \"step 1's review\" only as the source of a follow-up (ui-design.md, Session history; internal/app/returnclose_test.go:21-22). It does not give that review's verdict, and I did not rely on it.\n\nReview - plan 75, step 2 - 8 criteria: 7 PASS, 1 FAIL, 0 NOT VERIFIED\n\nPASS 1  75.10 Esc unchanged [read]: Esc now reaches askReturn only at the very end of the ladder (internal/app/keys.go:363-388). Every earlier step runs first and is unchanged: the failed start's back, the selections, zoom, focus back to zone B, every layer's handler (keys.go:47-90, 232-329) and the quit question's Stay (keys.go:47-49). TestReturnEscLadder (internal/app/returnhome_test.go:69-470) has one row for each use: popups, cards, full views, inspection, details, summary, entries, search, the three docked panels, the diff, the commit dropdown, windows, selections, zoom, focus from A and from C, the quit question, Worktrees and the failed start's back. Each row checks the old effect and that no question opened. Only \"the Esc that did nothing\" opens it, with No selected.\n\nFAIL 2  75.11 The question: at every size the title \"RETURN TO THE HOME SCREEN?\" is drawn in pink, not amber.\n- Code: internal/render/returnask.go:79 uses stAttentionBold. That style is the Attention colour cAttention #F472B6, ANSI 13 bright magenta (internal/render/palette.go:26, 405, 410).\n- What should be used: D7 says amber, and the step's own text in ui-design.md says \"The title in amber, bold\". The palette table in ui-design.md:4138-4139 calls that colour \"Attention\", pink, and #FBBF24 \"In progress\". The code itself uses stProgress wherever it says \"amber\" (render/dashboard.go:1036-1044, render/source.go:353-359).\n- The test locks the pink in: internal/render/returnask_test.go:84-86 asserts stAttentionBold (\"want the attention colour\"), while the test comments at returnask_test.go:37-38 and returnhome_test.go:504 say \"amber\".\n- The rest of 75.11 passes:\n  - the wording, Yes and No with No selected, ←/→, Enter, Esc = No, a click on a button, a click outside = No (returnhome.go:65-109; TestReturnQuestion, returnhome_test.go:482-598);\n  - the golden frames return-* and return-yes-* exist at 80x24, 120x40, 200x60 and 79x24 (golden_test.go:40, 1003).\n\nPASS 3  75.12 Never while work runs [read]:\n- The refusal: ReturnRefusal (internal/state/actions.go:368-376) refuses when Turn.Phase is not Idle, which covers a message waiting (Submitted), a running turn, a waiting approval and an interrupt under way. It also refuses while any agent is not finished, and while a git operation runs (gitRunning, returnhome.go:37-45). On a refusal, AskReturn and ReturnHome only set the notice (actions.go:196-205, 385-393) and askReturn opens nothing (returnhome.go:52-60).\n- The tests: TestReturnRefused (returnhome_test.go:604-785) covers a message waiting, a turn, a waiting approval, a background agent, a commit, a push, a staging, a fetch and a switch. For each it checks that nothing was written to the CLI, the phases are unchanged, the approval and the agent are untouched, and D8's text is in the frame. There are machine cases too.\n\nPASS 4  75.13 The return [read]:\n- Yes takes the same close path as a confirmed quit: stopWork, then CmdClose (actions.go:394-400 against 314-317), with the close timer tied to the session (app.go:1059-1062).\n- The home screen comes only once the process has exited (reduce.go:394-402).\n- After a crash or a forced stop it comes at once (actions.go:394-396). A mock run cannot return (actions.go:341-350).\n- Ctrl+Q during the close kills the CLI and then exits (actions.go:321-322, reduce.go:395-396).\n- Tests:\n  - TestReturnHome: stdin gets only its EOF, the 5 s close timer is set, the answers are kept and the session is in the history.\n  - TestReturnEscalation: a CLI that ignores the end of its input is ended around 5 s.\n  - TestReturnCrashMockQuit: the crash case, the mock run, and Ctrl+Q while closing.\n\nPASS 5  75.14 Nothing left behind [read]:\n- The state: s.home() (actions.go:329-334) is shared with Back. The reducer test (machines_test.go diff lines 2218-2222) and TestReturnHome (returnhome_test.go:887-892) both check the state equals New() except for the startup answers.\n- The model: goHome and resetSession (returnhome.go:120-190) rebuild it. sameAsFresh (returnhome_test.go:981-1029) compares every model field and UI field against a fresh model by reflection, and checks every generation counter moved past the session's.\n- Tests for the rest:\n  - the notification taken out once, the Git Graph reader closed, the turn store closed, no registration left (returnhome_test.go:893-904);\n  - the draft stored, then back at the next start (905-907, 922);\n  - the second session's frames at 120x40 and 200x60 show none of the first session's entries, and its state has one stage and no agents (930-944).\n- See Observations 4 and 5 for results still in flight at the moment of the return.\n\nPASS 6  75.15 Esc on the home screen [read]: startupKey's Esc returns nil (keys.go:111-117). The startup machine's quit row now says Esc on the home screen does nothing (events-and-state.md). TestHomeEsc (returnclose_test.go:151-167) checks that Esc on fields 0 to 2 leaves the form as it was and that Ctrl+Q quits.\n\nPASS 7  75.16 The machines [reported]: every new or changed step-2 row has its case in machines_test.go: the two Startup rows, the two Session rows and the two Invalid clauses (diff lines 2195-2252, 2260-2321, 2329-2373). The case-to-row check (machines_test.go:1565-1622) fails if any row lacks one. That TestStateMachines passes is only reported (evidence line 23, cached). Mutant 2's output names the D8 row's subtest.\n\nPASS 8  75.17 Broken on purpose, this step's mutants [reported]:\n- The evidence (lines 181-234) shows seven mutants run through go test -overlay: two removing D8's refusal (app and state), the notification, the Git Graph's read and the turn store left behind, Esc quitting on the home screen, and Yes as the default.\n- All seven were caught, and the fingerprint, index and diff hashes are equal before and after.\n- I could not read the mutants themselves. Reading the tests, each would be caught by an existing check: noQuestion and the state rows, returnhome_test.go:895-901, TestHomeEsc, and returnhome_test.go:462-463 with TestReturnQuestion.\n\nObservations:\n1. Medium, high confidence: the \"Not now\" notice outlives the work it describes.\n   - D8's notice is written to s.Turn.Notice (actions.go:202, 391). Nothing clears it when the turn or the git operation ends: turnEnd sets no notice, and review.go:23-27 only clears the snapshot notice.\n   - The input row shows Turn.Notice first (render/dashboard.go:948-949).\n   - So after a refused Esc, once the turn or commit has finished, the row still says \"Not now: the turn is still running · Ctrl+C interrupts it\" (or \"the commit is still running\") until the next send.\n2. Medium, high confidence: Esc hides the force-stop offer. The setup: an interrupt got no answer within 10 s, so Turn.Notice says \"no turn end after the interrupt: force stop closes the session\" (actions.go:232). That notice is the only place the screen offers F.\n   - Esc then replaces it with \"Not now … Ctrl+C interrupts it\".\n   - In that phase Ctrl+C only answers \"nothing to interrupt · Ctrl+Q quits\" (actions.go:510-511).\n   - F still works (keys.go:403-404) but nothing on screen names it any more.\n3. Medium, medium confidence: goHome blocks the event loop, which AGENTS.md's Engineering Standards forbid (\"no blocking call on the event loop\").\n   - It runs synchronously inside Update (returnhome.go:121-128):\n     - stopNotifying waits up to 2 s for Unnotify (notify.go:121-131);\n     - closeTurns waits up to 3 s for the snapshot worker (review.go:312-319);\n     - TurnStore.Close then takes the store's lock and deletes the store directory (input/turns.go:734-741);\n     - the draft is written.\n   - This is acceptable when the program quits, but after a return the program goes on. A long end snapshot (reliability.md: up to 30 s past about 4,000 files) can still be running when the owner returns.\n4. Low-medium, medium confidence: results of the ended session can still arrive after the return. These carry no session number: the turn review's results (review.go:175, 211, 258, 264), write failures and the write worker's panic (app.go:1000-1039, writer.go:51), and pasted or picked images (inspectedMsg).\n   - A snapshot cancelled by closeTurns answers after goHome. The fresh state then logs \"turn snapshot for no turn\" (state/review.go:195-197), so the state is no longer New()'s.\n   - If such an answer arrives after a second session has reached the same turn number, it is applied to that session's review.\n   - The new text in reliability.md says nothing of the session reaches the next one, but that is only true for reads and timers.\n5. Low, medium confidence: work that starts during the return's close is dropped silently.\n   - A commit, switch or fetch started during the close (up to 10 s) is wiped from the model by resetSession (returnhome.go:175) without being stopped.\n   - Its outcome is never shown.\n   - A Ctrl+Q asked meanwhile (QuitAsked) is lost when the return completes.\n6. Medium, medium confidence; pre-existing, not in this diff: Back after a failed start never closes the turn store.\n   - Neither the Back path nor closeSession closes it (keys.go:364-368, 448-464).\n   - The next start's openTurns then does nothing (review.go:96), so that session gets no turn review, and the old project's store stays until quit.\n   - This now matters more, since one run can hold several sessions.\n7. Low, high confidence: step 1's delete question has the same colour mismatch. Its title is described as \"in amber\" but drawn with stAttentionBold (render/history.go:341).\n8. Info: LayerReturn is a layer in render.Layers but sits outside Layer.Popup()'s range (render/layers.go:33). It is drawn alone and takes every input, which is a reasonable reading of D7's \"render.Layers popup\".\n9. Info: D11's plan.md part (the Nice to have item removed) is not verified, because plan.md is outside what I may open. No criterion of this step requires it.\n\nRequested tests:\n- In TestReturnBox, assert the title is drawn in the amber D7 names (the palette's In progress #FBBF24, bold), in place of stAttentionBold.\n- Esc refused during a turn, and separately during a commit; then the work ends. Decide what the input row should show after that and test it (the \"Not now\" text gone).\n- Esc during Interrupting with the force stop offered: the input row still names the force stop.\n- A return while the turn's end snapshot is held past goHome. After its answer arrives, the state equals New() except for the answers, and a second session's turn 1 review does not take it.\n- Measure goHome's time on the event loop with a slow Unnotify and with a held snapshot.\n- Back after a handshake timeout, then a start in another project: the turn review works for the new project.\n\nVerdict: FAIL"
}
````

### Step 2, its second review, void

`af3178579a9405a2d`

````text
{
  "message": "First, some of the files contain the Implementer's own account of earlier reviews. `internal/app/returnfix_test.go:20-23` calls itself \"Plan 75, step 2's first correction\". `internal/app/returnclose_test.go:20-21` and `:169-172` cite \"step 1's review\", and so does the ui-design.md diff (Session history paragraph). Those reviews count for nothing here; I judged the code as it stands. I did not open plan.md, so D11's third item (the Nice to have entry leaving plan.md) is not reviewed.\n\nReview - plan 75, step 2 - 8 criteria: 7 PASS, 1 FAIL, 0 NOT VERIFIED\n\nPASS 75.10 Esc unchanged [read + reported]: in `internal/app/keys.go:366-396` the ladder is unchanged. Only the final branch is new: focus already \"B\" now calls `askReturn` instead of re-setting the focus. TestReturnEscLadder (`internal/app/returnhome_test.go:65-470`) has one row per listed use: the \"+\" menu, the model, modes, branches and scope popups, the commit dropdown, the approval card, its full view, the inspection, the details view, the summary suggestion, the entry cursor and its full view, the search bar, the Git Graph, Explorer and Source Control panels, the diff, the mouse and draft selections, zoom, focus from C and A, the quit question, Worktrees, and the failed start's back. Each checks the old effect and that no question opened. A last row checks the Esc that did nothing opens the question with No selected. Passing runs are reported only. One corner case is listed under Observations.\n\nPASS 75.11 The question [read]:\n- Title, text, buttons and keys row: `internal/render/returnask.go:22-25,38-40,73-80`. The title uses `stProgressBold`, which is amber `cProgress` #FBBF24 (`palette.go:25,404,2317`).\n- No is the default: `returnhome.go:104`.\n- ←/→, Enter and Esc as No: `returnhome.go:112-124`. A click on a button answers; a click outside means No: `returnhome.go:129-146`.\n- It is a `render.Layers` layer (`layers.go:85-88`) drawn with the summary card's `cardButton` (`summaryoffer.go:2406-2412`).\n- Golden frames `testdata/frames/return-{80x24,120x40,200x60}.txt`, and the `return-yes-*` frames, show the D7 text.\n- TestReturnBox checks the styles at all three sizes; TestReturnQuestion checks the keys and the mouse.\n\nPASS 75.12 Never while work runs [read + reported]:\n- `askReturn` (`returnhome.go:99-107`) opens the question only when `CanReturn` holds and `ReturnRefusal` is empty.\n- `AskReturn` and `ReturnHome` return no command while `Running()` is set or a git operation is named (`actions.go:2537-2568,2600-2607`). `Running()` covers Submitted, Running, WaitingApproval, Interrupting and live agents.\n- The texts match D8.\n- TestReturnRefused (`returnhome_test.go:604-785`) covers a waiting message, a turn, an approval, a background agent, a commit, a push, a staging, a fetch and a switch. It checks that nothing is written to the CLI, that the turn or git operation goes on, and that the notice is on the input row. Runs reported.\n\nPASS 75.13 The return [read + reported]:\n- `returnHome` sets `Returning` and `Closing`, then calls `stopWork` and `CmdClose`, exactly as the quit does (`actions.go:2608-2614`). `CmdClose` arms the 5 s timer (`app.go:1081`).\n- `ended` calls `home()` and `CmdHome` only once the process has exited (`reduce.go:395-403`).\n- A crashed or closed session goes back at once (`actions.go:2608-2610`).\n- A mock run gets no question (`CanReturn`, `actions.go:2507-2516`).\n- Ctrl+Q during the close: the Exited case comes first in `ended`.\n- Tests: TestReturnHome, TestReturnEscalation, TestReturnCrashMockQuit, and the Session and Startup machine rows. Runs reported.\n\nFAIL 75.14 Nothing left behind: the session's Git Graph card reads outlive the return and crash the program.\n- **The reads:** hovering a Git Graph row starts `readCommitMessage` and `readCommitFiles` (`graphcard.go:40-46,79,86,102-105`; `graph.go:842-845`). Unfolding a commit also starts `readCommitFiles`. Their messages carry no generation and are not wrapped by `sessionCmd`.\n- **The return:** Esc does not close the Git Graph (`returnhome.go:99-107`). Yes, then `goHome`, then `resetSession`, runs `*m = Model{}` and restores only `graph.gen`, `refsGen` and `watchGen` (`returnhome.go:395,408`). So `m.graph.messages` and `m.graph.files` are nil maps.\n- **The crash:** when a read answers after the return without an error, `graphMessageRead` writes `g.messages[msg.hash] = msg.msg` (`graphcard.go:123`), or `graphFilesRead` writes `g.files[msg.hash] = fs` (`graph.go:873`). Both assign into a nil map, which panics. Bubble Tea ends the program with ExitPanic on the home screen.\n- **When it happens:** a read slower than Esc, ←, Enter after a crash (the return is immediate), or slower than that plus the CLI's ~1.4 s exit when idle. Likely on a large commit or a WSL path.\n- **What it breaks:** the session's reads are left behind and reach the fresh model. reliability.md's new paragraph promises that every read of the session is dropped. TestReturnHome only sets `graph.reader` (`returnhome_test.go:833-834`), so no test can catch this.\n\nPASS 75.15 Esc on the home screen [read + reported]:\n- `startupKey` now returns nil on Esc (`keys.go:113-117`); Ctrl+Q still quits (`keys.go:39-46`).\n- The startup row reads \"quit (Ctrl+Q; Esc on the home screen does nothing, plan 75, design D10)\".\n- TestHomeEsc (`returnclose_test.go:151-167`) checks fields 0 to 2 and the quit.\n\nPASS 75.16 The machines [read + reported]:\n- Each new or changed row has exactly one case under `docMachines`' matching rules (`machines_test.go:1531-1641`):\n  - Startup: \"`Ready` | owner returns home, idle\" (prefix) and \"`Ready`, the session `Crashed` or `Closed` | owner returns home\".\n  - Session: \"`Ready` | owner returns home\" (exact) and the \"while a turn, an agent, an approval or a git operation runs\" row.\n  - Invalid clauses: \"a return to the home screen is ignored in a mock run\" and \"Each session is one CLI and one project, and one run of ACC can hold several sessions\".\n- The Esc part of the changed quit row is app-level and is covered by TestHomeEsc.\n- TestStateMachines passes per evidence lines 23-24; mutant 2 shows the new subtest running.\n\nPASS 75.17 Broken on purpose, this step's mutants [reported]: evidence lines 181-300.\n- Each run uses `go test -overlay`.\n- Fingerprint, index and diff hashes are equal before and after.\n- Every mutant this step owns is caught:\n  - m1 and m2: D8's refusal removed.\n  - m3, m4 and m5: notification, Git Graph read and turn-review store left behind.\n  - m15: Esc quitting on the home screen.\n  - m18: Yes as the default.\n- The mutant scripts are outside the repository and the brief's folder, so I did not read them.\n\nObservations:\n- Medium severity, high confidence. The Worktrees view's Enter on the home screen starts a session without the `stillClosing()` check (`internal/app/worktrees.go:319-332`). The guard is applied only in `keys.go:180`, `lastsession.go:46` and `history.go:202`. Right after a return, Ctrl+W then Enter on the main worktree can start a session in the same project while `runEnd` is still saving and releasing the draft and removing the store. The new start's draft `Take` can then miss the saved draft, against 75.14's \"comes back with the next start\".\n- Low to medium severity, high confidence. The staging question (LayerAsk) passes F6 on (`commit.go:602-615`, `source.go:312-313`), so focus can reach zone B with it still open. Esc there now calls `askReturn`, which drops the staging question (`closePopups`, `returnhome.go:103`) and opens the return question. D7 says the question opens only with no popup open. The baseline Esc did nothing here, so 75.10's literal text holds. The ladder has no row for this case.\n- Low severity, medium confidence. `Shutdown` waits on `m.ending` with no bound (`app.go:1271-1273`). That channel is closed only by the `runEnd` command (`returnhome.go:315-326`), which Bubble Tea must still run inside nested batches. If the program ends between the `CmdHome` update and that command starting (a quit, or a panic such as the one above), the exit hangs.\n- Low severity, medium confidence. During a return, `CmdKill` arms the `leaveMsg` fallback only when Exited (`app.go:1082-1088`). A CLI whose forced kill yields no session end leaves the dashboard \"closing\" until Ctrl+Q.\n- Low severity, low confidence on reachability. `ReturnRefusal` counts the turn and live agents, not `Queue` (`actions.go:460-484`). An approval pending while the turn is Idle and no agent runs would let the question open, and Yes would answer it `cancel` through `stopWork` (`actions.go:493-500`), against D8's \"nothing answered\".\n- Not reviewed: D11's plan.md item, since I may not open plan.md.\n\nRequested tests:\n- TestReturnLateGraphCard: show the Git Graph with a fake git whose CommitMessage and CommitFiles block. Hover a row (and unfold one) so both reads start. Return home both after a crash (immediate) and from idle. Release both reads after `CmdHome`. Expect no panic, the state equal to New's except the answers, and `sameAsFresh` to hold.\n- Worktrees start while the last session closes: hold `runEnd` as TestReturnOffLoop does, then Ctrl+W and Enter on the project's worktree. Expect no spawn and \"The last session is still closing\". After the release, expect the draft back at that start.\n- Staging question with focus moved to zone B by F6, then Esc: expect no return question and the staging question as the baseline left it.\n\nVerdict: FAIL"
}
````

### Step 2, its third review

`a61e3534e9b9787ae`

````text
{
  "message": "Review - plan 75, step 2 (Return to Home, D7 to D11) - 8 criteria: 8 PASS, 0 FAIL, 0 NOT VERIFIED\n\nI found no earlier verdict on this plan's work and no account of it by the Implementer. I did not open plan.md: I skipped its hunk in changes.diff (lines 3192 to 3542).\n\nPASS 75.10 Esc unchanged [read]: Esc still takes the same path as before. The top layer handles it first (C:\\javadev\\projets\\agent-command-center\\internal\\app\\keys.go:52-90), then the windows and panels (:235-272), then card, view, inspection, details, summary and entry (:303-332). The failed start's Back comes first in the esc case (:367), then selection, draft selection, zoom and focus back to B (:378-393). Only the old no-op, with the focus already on B, now calls askReturn (:396). askReturn opens nothing while any layer is still on top, the staging question included (C:\\javadev\\projets\\agent-command-center\\internal\\app\\returnhome.go:101). C:\\javadev\\projets\\agent-command-center\\internal\\app\\returnhome_test.go:69-481 has one row per category: popups (+ menu, model, modes, branches, scope, commit dropdown, staging, key overlay), cards, approval and entry full views, inspection, details, summary, entry cursor, search, the three docked panels, the diff, mouse and draft selections, zoom, focus from C and from A, the quit question, Worktrees, and the failed start's back. Each row asserts that no question opened, and \"the Esc that did nothing\" asserts the question opens with No selected. syncUI turns a focus of B into D while the diff is open (app.go:817-818), so the diff view cannot be hidden under the question.\n\nPASS 75.11 The question [read]: the title, the D7 text (with ProviderName) and the [ Yes ] / [ No ] buttons are in C:\\javadev\\projets\\agent-command-center\\internal\\render\\returnask.go:22-40,73-79. The title is amber and bold (stProgressBold, palette.go:463, cProgress #FBBF24). The buttons come from the summary card's cardButton (summaryoffer.go). The question is its own layer, LayerReturn, in render.Layers (layers.go:85-88), wired for keys (keys.go:54-57), mouse (mouse.go:62-63) and paste (app.go:1315). In returnhome.go:117-161, ← selects Yes and → selects No; Enter answers the selected button; Esc is No; a left click on a button answers with it, and a click outside the card is No. The question opens with Yes=false (:109). Golden frames exist at 80x24, 120x40 and 200x60 (plus 79x24), C:\\javadev\\projets\\agent-command-center\\testdata\\frames\\return-*.txt; they are plain text, so the look of the selected button rests on TestReturnBox.\n\nPASS 75.12 Never while work runs [read]: AskReturn and ReturnHome both go through refuseReturn (C:\\javadev\\projets\\agent-command-center\\internal\\state\\actions.go:205-210,423-443,485). That function only sets the notice: no command is written, nothing is interrupted or answered. Running() treats Submitted, Running, WaitingApproval and live agents as running (:499-523); git comes from gitRunning(), with the quit question's names (returnhome.go:85-93). TestReturnRefused (returnhome_test.go:615-797) covers a waiting message, a turn, a pending approval, a background agent, a commit, a push, a staging, a fetch and a switch. It counts the bytes on the CLI's wire (or the scripted session's calls) and checks D8's text on the input row.\n\nPASS 75.13 The return [read]: Yes on a Ready session sets Returning and Closing and issues stopWork plus CmdClose (actions.go:492-494), the quit's own path. CloseTimeout gives CmdKill, then a KillGrace timer (app.go:1014-1028), as the quit does. The home screen comes only in ended() once the process has exited (C:\\javadev\\projets\\agent-command-center\\internal\\state\\reduce.go:395-403). A crashed or closed session goes home at once (actions.go:488-490). CanReturn refuses a mock run (:359-368). A Ctrl+Q during the close takes the quit's branch (the Exited case at reduce.go:396, then CmdExit). Tests: TestReturnHome (stdin EOF alone, the 5 s close timer, the answers, the history), TestReturnEscalation, TestReturnCrashMockQuit and TestReturnKillGrace.\n\nPASS 75.14 Nothing left behind [read]: s.home() rebuilds the state from New() and keeps only Startup (actions.go:347-352). goHome takes the notification, the Git Graph's reader, the turn store and the draft off the event loop, closes the views, the write queue and the session, then resetSession sets `*m = Model{}` and restores only the application's fields, every generation set past the session's (returnhome.go:175-193,396-435). lockSync releases the lock (worktrees.go:582-585). Late messages carry the session's number (sessionMsg) and are dropped (app.go:528-535). TestReturnHome checks every model field against a fresh model by reflection (sameAsFresh), the stored draft and its return at the next start, and that the second session's frames at 120x40 and 200x60 show none of the first's entries. TestReturnLate* and TestReturnOffLoop cover reads that answer late.\n\nPASS 75.15 Esc on the home screen [read]: startupKey's esc case returns nil (keys.go:113-117); Ctrl+Q stays in key() (:39-46). The startup machine's quit row says Esc does nothing (C:\\javadev\\projets\\agent-command-center\\doc\\events-and-state.md:360). The test is TestHomeEsc (returnclose_test.go:150-166).\n\nPASS 75.16 The machines [reported]: the new rows each have a case in C:\\javadev\\projets\\agent-command-center\\internal\\state\\machines_test.go: three Startup, three Session and two invalid clauses. The changed quit row keeps its Quit case; its Esc part is an application key, tested by TestHomeEsc. TestStateMachines requires exactly one case per row and per clause (:1624-1668). That it passes is only reported (state \"ok (cached)\").\n\nPASS 75.17 Broken on purpose, this step's mutants [reported]: the evidence's step-2 run uses go test -overlay. All six required mutants fail a test: D8's refusal removed (m1, m2), the notification left (m3), the Git Graph's read left (m4), the turn store left (m5), Esc quitting on the home screen (m21), and Yes as the default (m24). The fingerprint, index and diff hashes are equal before and after. I could not read the mutant scripts: they sit outside the repository and outside the brief's folder.\n\nObservations:\n- Low severity, high confidence: a Ctrl+Q during the return's kill grace is briefly undone. After Yes, if the CLI ignores its forced kill, CmdKill schedules returnGraceMsg (app.go:1024-1028). A Ctrl+Q pressed then sets Startup Exited but leaves Returning and Closing as they are (actions.go:330-340). The earlier timer arrives first, and ReturnGrace (actions.go:213-219; app.go:578-582) calls home(): Startup goes from Exited back to Asking and the home screen shows until the quit's leaveMsg ends the program up to 5 s later. The program still quits, so 75.13 holds, but the state goes back on an exit. Suggested fix: guard ReturnGrace with `Startup.Phase != Exited`.\n- Low, high: there is a third refusal text, \"Not now: an approval is waiting · F2 answers it\" (actions.go:384-404). D8 names two texts and counts a waiting approval as part of the turn. The test itself says no CLI event reaches that state (returnmore_test.go:190-191); the new text is documented in ui-design.md and events-and-state.md.\n- Low, medium: the question can open while a Git Graph card is pinned. Esc from the graph moves the focus to B and keeps the card (graph.go:738-740), and the card is not a top layer. D7 lists \"no ... card ... open\", but Esc did nothing there before, which D7's first clause allows; returnlate_test.go:74-77 relies on it.\n- Low, medium: after a crash from a recovered adapter panic, the CLI may still be closing. crash(..., true) issues CmdClose (reduce.go:413-416,448-450), and a Yes \"at once\" then runs closeSession's m.cancel() (keys.go:457-459): a forced kill with no grace. So ui-design.md's \"the process is gone already\" does not always hold. This stays within D9's \"at once\".\n- Note: D11's third bullet, the Nice to have item leaving plan.md, is not checked: plan.md is off limits to me, and no criterion of this step names it. D11's other two bullets are done (events-and-state.md; AGENTS.md:245-247).\n\nRequested tests: Yes with a CLI that ignores its kill (wireProc.deaf); deliver the close timer, press Ctrl+Q, then deliver the returnGraceMsg. Startup.Phase must stay Exited, no CmdHome may run, and the program must quit.\n\nVerdict: PASS"
}
````

### The first final review

`ab0fd5b8cbb9fd068`

````text
{
  "message": "First, two things in the inputs that count for nothing. The plan.md hunk of changes.diff (lines 3546-3550) holds the Progress lines \"Step 1: done. Step 2: done. Step 3: in progress.\", which are the implementers' own account, and I gave them no weight. Also, one of my CR/BOM greps used the glob `doc/*.md`, which also matches `doc/plan*-evaluation.md`. It returned \"No files found\", so I saw nothing from those files.\n\nReview - plan 75, final review (complete plan, diff from aa77d6e) - 21 criteria: 19 PASS, 1 FAIL, 1 NOT VERIFIED\nPASS 1  75.1 The record [read + reported]: The version check is at projects.go:114. Version 1 is folded in by settled() (:152-176), which sorts newest first, keeps paths, times and favourites, and adds no model or title. read() (:136-150) drops entries with no ID and caps at 20. Save writes version 2 (:187). RecordSession (:247-270) moves the same CLI+ID first, takes the new end time and model, keeps the old title, and caps at MaxHistory. The mock / no ID / CLI-opened guard is at reduce.go:997-1000. The first prompt is taken only when Startup.Resume==\"\" (reduce.go:131-139) and kept through a crash resume (actions.go:236). The title is SessionTitle (render/history.go:104-132): Cc/Cf removed, whitespace collapsed, 60 clusters, none cut. The baseline Load refuses any version but 1 and Update returns before Save (orchestrator.txt:91-93, 139-143). The only writer is Update (app/projects.go:45, unchanged). Tests: projects_history_test.go, TestHistoryRecord in app, TestSessionTitle*, TestFirstPrompt*.\nPASS 2  75.2 Unchanged [read + reported]: The only plan 69/70 test changed is TestProjectSessions (projects_test.go:129-216), and every check is kept in version 2 form, including the older-session forget at :168. Ctrl+R uses Newest with the refused ID skipped (lastsession.go). TestHistoryCtrlR and TestHistoryResume cover the previous entry being offered.\nPASS 3  75.3 Opening [read + reported]: openHistory/historyChecked (app/history.go:50-82) refuse while validating, from Worktrees and under the folder dialog. Start's own check (validate.go:40) runs in a command, and its refusal shows via OpenProblem (render/home.go). The frames history-* and history-none-* exist at 80x24, 120x40 and 200x60. Tests: TestHistoryOpen, TestHistoryStartRefuses, TestHistoryPicture.\nPASS 4  75.4 Rows [read + reported]: historyView (app/history.go:288-317) builds the meta line, uses \"N/A\" when there is no model, and ShortID. historyEntryRows gives a title row only when there is a title. historyTop and the ▐ mark are at render/history.go:200-312. TestHistoryRows; history-scroll-80x24.txt.\nPASS 5  75.5 Keys and mouse [read + reported]: historyKey and historyMouse (app/history.go:105-285). A right click returns nil. Ctrl+N/O/R/W fall into the view and do nothing; Ctrl+Q is taken first (keys.go:39-46). Tests: TestHistoryKeysAndMouse, TestHistoryCtrlRDoesNothing.\nPASS 6  75.6 Resume [read + reported]: historyResume (app/history.go:195-222) sets ui.Choice and Confirm with Resume. TestHistoryResume checks `--resume <id>`, `\"excludeTurns\":true`, an uninstalled CLI being dimmed with its note, and a refused resume removing the entry so that Ctrl+R offers the previous one.\nPASS 7  75.7 Delete [read + reported]: historyAsk/historyAnswer remove by CLI+ID (app/history.go:144-189), with Cancel selected by default. ForgetSessions is projects.go:285. TestHistoryDelete (app and input) compares the whole file to Save(expected).\nPASS 8  75.8 The CLIs' data [read + reported]: No added diff line and no new code file touches .claude, .codex, UserHomeDir, os.Remove or os.Open. TestHistoryNoCLIData exists. The configuration compare passed for both CLIs (orchestrator.txt:909-911, 1495-1497).\nPASS 9  75.9 Footer [read]: homeKeys and the 100-column threshold are in render/home.go. homegrid.go builds the 99-cell and 65-cell rows. footCaps keeps 141/357.3/582.5 and adds a fourth group (homepaint.go). Every startup frame hunk in changes.diff:3554-3761 replaces only the last row. TestHomeFooter.\nPASS 10  75.10 Esc unchanged [read + reported]: The ladder in keys.go:366-396 ends in askReturn, which also refuses under any layer or a pinned card (returnhome.go:100-115). TestReturnEscLadder has one row per use, including the failed start's back.\nPASS 11  75.11 Question [read + reported]: returnask.go uses LayerReturn and cardButton, with No as the default (ReturnAsk zero value). returnKey/returnMouse are at returnhome.go:120-154. TestReturnQuestion, TestReturnBox (amber stProgressBold); frames return-* and return-yes-* at three sizes.\nPASS 12  75.12 Never while work runs [read + reported]: ReturnRefusal/refuseReturn (actions.go:395-448) emit no command. gitRunning is at returnhome.go:85. TestReturnRefused counts stdin lines and the scripted session's calls across a turn, an agent, an approval, a commit, a push, a staging, a switch and a fetch.\nPASS 13  75.13 The return [read + reported]: returnHome (actions.go:481) runs stopWork+CmdClose. Mock, starting and closing sessions are blocked by CanReturn (:360). ended() goes home only after Closed (reduce.go:383-403). Tests: TestReturnHome (EOF alone, 5 s CloseTimeout), TestReturnEscalation, TestReturnCrashMockQuit (crash: at once; mock: no question; Ctrl+Q while closing quits).\nPASS 14  75.14 Nothing left behind [read + reported]: goHome and resetSession (returnhome.go:178-196, 399-438). TestReturnHome checks state DeepEqual New()+Startup, sameAsFresh on every model and UI field, the draft saved and restored, and a second session drawing only its own entries. Note: `lock` is in appFields (returnhome_test.go:988), so the lock is checked through reg.Live() being empty, not by field equality.\nPASS 15  75.15 Esc on the home screen [read + reported]: startupKey returns nil on Esc (keys.go:113-117). The startup table row says so (events-and-state.md). TestHomeEsc, mutant 21.\nPASS 16  75.16 Machines [reported]: TestStateMachines (machines_test.go:1626-1683) parses the doc and fails on any row or clause without exactly one case. The new rows and the reworded invalid paragraph have cases. The run passed (orchestrator.txt:315).\nPASS 17  75.17 Mutants [reported]: 23 + 26 mutants were run with -overlay, with the fingerprint, index and diff hashes equal before and after, and every listed mutant was caught (evidence.txt:25-352). The mutation code itself is in scripts outside the repository and the brief, so I could not read it.\nPASS 18  75.18 Checks [reported]: Build, vet, linux vet, gofmt (empty) and tests passed at the same fingerprint as the evidence (orchestrator.txt:290-318). -race exited 0 for state, input, render and app (:341-354). The timing step ran on mains power with no go process running, every test PASS (:387-558).\nFAIL 19  75.19 Live run: L75's approved Usage is \"two Haiku turns and two Codex turns\", but three Haiku turns ran. The first Claude Code attempt sent prompt1 (session 12a9063d, orchestrator.txt:802-805) and then aborted on the driver's own `$home` bug (:807-813). A second attempt about one minute later ran two more turns (:866-874). The plan says \"Work stops at once on anything unexpected\", and AGENTS.md:255-258 says a live run \"runs only within its approved scope\". No owner approval of the second attempt appears in the inputs. Every L75 check of the second Claude Code run and of the Codex run passed (:866-874, :1456-1464). Confidence medium: this becomes PASS if the owner approved the re-run.\nNOT VERIFIED 20  75.20 Documents: The documents match the code apart from Observation 1. Two parts of the byte rule are not settled. (a) plan.md cannot be opened, and a BOM at its byte 0 cannot appear in a diff whose hunks start at line 42. (b) Grep's BOM detection is unreliable, because ripgrep may strip a UTF-8 BOM before matching. CR and the user name: none in changes.diff or in the plan's other files (read). What settles it: a byte check of every file in plan-files.txt in the evidence, covering the first three bytes, a 0x0D count and the user name.\nPASS 21  75.21 Scope [read]: Every path in changes.diff and new-files.txt is in the plan's Files list. Nothing under provider, cmd, scripts or .claude changed, and go.mod and go.sum are unchanged.\nObservations:\n- Medium (high confidence in the code path): a Worktrees merge or remove can start while the return closes the CLI, and its result is then dropped unseen. Ctrl+W still opens the view during Closing (keys.go:100-103), and wtQuestionKey (worktrees.go:421-466) has no `m.s.Returning` check, unlike startOp (commit.go:413), switchRefused and fetchBranches (branches.go). goHome's resetSession (returnhome.go:423) drops m.wt and bumps wtGen, so worktreeDone (worktrees.go:146-150) discards the merge's outcome, conflict or failure included. reliability.md:443-444 says \"While the CLI closes, no git operation starts … so that none is dropped unseen\". Plan 47's quit check also leaves worktree ops out, so the project may not count them as \"git operations\". Fix one or the other: refuse with returningNote, or narrow the doc.\n- Low: TestSessionTitleCost (render/history_test.go:496-502) judges a wall-clock bound (5 ms) inside `go test ./...`. doc/evaluation.md:114 says \"A time is judged against its target only in the timing step\". The flake risk is small.\n- Low: an images-only first message leaves the session without a title for good (reduce.go:131-139; state/history_test.go:67-79). D1a's \"images do not count\" could also mean that message is not the first. The chosen reading is documented in ui-design.\n- Low: RecordSession (projects.go:260-263) lets a later record set a title on an entry that has none. \"Set once\" therefore relies on the state never giving a resumed session a FirstPrompt.\n- Low: reliability.md:420 cites \"plan.md, Plan 75, Known limits\", which moves out of plan.md at the close. Two lines added to reliability.md, the ones ending \"…writes into a map the fresh model leaves empty. One of a session…\" and \"…whatever the turn's phase, so that the return's close never answers one. The state is then…\" (changes.diff:327, 334), are not wrapped.\n- Low: architecture.md's new \"The session history\" section goes beyond \"where it describes a session's life\" in the plan's Files list. The file itself is in scope.\n- Low: Back after a failed start now runs a session end (keys.go:367-376). START SESSION and Ctrl+R are refused with \"The last session is still closing\" until the turn store is removed (up to 3 s).\n- Low: when the terminal is too small (history-79x24.txt), the history view stays open, invisible, and still takes Enter and Del. The return question closes in that case (layers.go closeUnshown). Worktrees behaves the same way already.\n- Low: after a return through ReturnGrace (the forced kill gave no end), the old session's pending listen command and the writer may stay blocked on a process that never ends.\nRequested tests:\n- A byte check of every file in plan-files.txt, plan.md included (first 3 bytes ≠ EF BB BF, no 0x0D, no Windows user name), with its output in the evidence (75.20).\n- Yes to the return with the CLI held open (onEOF blocking), then Ctrl+W, a non-main worktree, m, y: expect no Merge call and a \"Returning to the home screen\" note. Or narrow reliability.md:443.\n- The owner's recorded approval of the second Claude Code attempt and its third Haiku turn (75.19).\nVerdict: FAIL"
}
````

### The final review

`a2f08e84067b38b56`

````text
{
  "message": "Review - plan 75, final review (complete plan, diff from aa77d6e) - 21 criteria: 21 PASS, 0 FAIL, 0 NOT VERIFIED\n\nPASS 1  75.1 The record [read]: The newest entry goes first, the same CLI and ID moves first and keeps its title, and the list is cut to 20 (`internal/input/projects.go:247-268`). Load accepts versions 1 and 2 only (`projects.go:114`), migrates version 1 through `settled()` (`projects.go:152-176`) and filters version 2 through `read()` (`projects.go:136-150`). Save always writes version 2 (`projects.go:187`). The baseline code refuses any version but 1 and `Update` writes nothing after a failed Load (`orchestrator.txt:91-93`, `139-149`). The title text is taken once, only for a session started new (`internal/state/reduce.go:131-140`), and carried at the turn end (`reduce.go:1000`). The cleanup and the cut at 60 grapheme clusters are in `SessionTitle` (`internal/render/history.go:105-133`). Mock runs, sessions without an ID and turns the CLI opened still record nothing (`internal/state/resume_test.go:36-61`). Tests: `TestProjectsMigration`, `TestProjectsVersion3` (file compared byte for byte), `TestHistoryRecord` in input and app, `TestSessionTitle*`, `FuzzSessionTitle`, `TestFirstPrompt*`.\nPASS 2  75.2 Unchanged [read]: Ctrl+R takes the CLI's newest entry, skipping one just refused (`internal/app/lastsession.go:30-44`). Plan 69's and 70's tests are not in the diff (`lastsession_test.go`, `state/resume_test.go`, `app/home_test.go`), and Save folds their old `Sessions` lists into the history. `TestProjectSessions` keeps every check in version 2's shape (`internal/input/projects_test.go:129-216`). `TestHistoryResume` and `TestHistoryCtrlR` cover the previous entry offered after a refusal or a delete. The suite's exit 0 is reported (`orchestrator.txt:345`).\nPASS 3  75.3 Opening [read]: `openHistory` and `historyChecked` refuse while a start is validated or the folder dialog is open, and on a path Start refuses, whose refusal the field then shows (`internal/app/history.go:50-82`). The Worktrees view takes keys before Alt+H can reach it (`internal/app/keys.go:91-96`). Tests: `TestHistoryOpen`, `TestHistoryStartRefuses`, `TestHistoryFolderDialog`. Golden frames `history-*` and `history-none-*` exist at 80x24, 120x40 and 200x60 (`internal/app/golden_test.go`, historyRig).\nPASS 4  75.4 The rows [read]: Each row is the title, then one grey line: date, age, CLI, model or \"N/A\", and `ShortID` (`internal/app/history.go:307-313`; `internal/render/history.go:249-274`). The list scrolls only as the selection needs, with the slim mark (`render/history.go:208-239`, `309-320`). Tests: `TestHistoryRows`, `TestHistoryScroll`, `TestHistoryTop`.\nPASS 5  75.5 Keys and mouse [read]: Keys are handled in `history.go:105-140`, the mouse in `history.go:229-285`, and Ctrl+Q before either (`keys.go:39-46`). `TestHistoryKeysAndMouse` covers every key and click of D3, the right click doing nothing, Ctrl+N/O/R/W doing nothing (\"norw\") and Ctrl+Q quitting.\nPASS 6  75.6 Resume [read]: `historyResume` sets question 1 to the entry's CLI and confirms with the entry's ID (`history.go:195-222`). `TestHistoryResume` checks `--resume <id>`, `thread/resume` with `\"excludeTurns\":true`, question 1 showing that CLI, the not-installed note, and the entry gone after a refusal with Ctrl+R then offering `s-older`.\nPASS 7  75.7 Delete [read]: The question counts exactly the entries it names, by CLI and ID (`history.go:144-189`; `projects.go:285`). `TestHistoryDelete` compares the whole file with what Save writes for the expected list (other projects, the favourite and the order included). It also checks Cancel selected at open, and that Cancel, Esc and a click outside remove nothing. `TestHistoryDeleteAllExact` covers an entry another dashboard adds meanwhile.\nPASS 8  75.8 The CLIs' data [read]: Searching the diff's added lines finds no `.claude`, `.codex`, `UserHomeDir` or file write/remove call, only plan text (`changes.diff:3270`, `3297`, `3476`). Alt+H's check only stats the project path (`internal/app/validate.go:40-60`). The live run's configuration compare passed for both CLIs (`orchestrator.txt:879`, `1465`).\nPASS 9  75.9 The footer [read]: \"Alt+H History\" sits between Ctrl+O and Ctrl+Q (`internal/render/home.go:299`). Grid mode uses the 99-cell full row from 100 columns and the 65-cell short row under that (`homegrid.go`). In picture mode the first three positions are unchanged (`homepaint.go` footCaps). `TestHomeFooter` checks all three. The startup frames' diffs touch only the last row (`changes.diff:3564-3773`).\nPASS 10  75.10 Esc unchanged [read]: The question is reached only at the end of the Esc ladder (`keys.go:366-396`), and `askReturn` also refuses under any layer or a pinned Git Graph card (`internal/app/returnhome.go:100-115`). `TestReturnEscLadder` has one row per earlier use of Esc, about 31 in all, including the failed start's back, each checking the old effect and that no question opens (`returnhome_test.go:109-494`).\nPASS 11  75.11 The question [read]: The card is drawn in `internal/render/returnask.go:22-114`. ←/→ select, Enter answers, and Esc or a click outside means No (`returnhome.go:120-164`). Tests: `TestReturnQuestion`, `TestReturnBox` (title amber and bold, No selected), `TestReturnLayer`. Frames `return-*` and `return-yes-*` exist at all three sizes.\nPASS 12  75.12 Never while work runs [read]: `ReturnRefusal` and `refuseReturn` (`internal/state/actions.go:395-446`) are checked before the question opens (`returnhome.go:109-113`). `TestReturnRefused` covers a message waiting, a turn, an approval, an agent, a commit, a push, a staging, a fetch and a switch. Each checks that nothing is written to the CLI, the phases are unchanged, and the input row shows the \"Not now\" text.\nPASS 13  75.13 The return [read]: Yes marks the session returning and closes it as quit does (`actions.go:481-496`). The home screen comes once the process has exited (`reduce.go:395-403`), and after the forced kill's grace if no exit is seen (`app.go`, CmdKill and returnGraceMsg). Tests: `TestReturnHome` (stdin EOF alone, the 5 s close timer), `TestReturnEscalation`, `TestReturnCrashMockQuit` (crash at once, no question in a mock run, Ctrl+Q while closing quits), `TestReturnKillGrace` and `TestReturnKillGraceQuit`.\nPASS 14  75.14 Nothing left behind [read]: The state resets through `home()` (`actions.go:348-353`), the application through `goHome` and `resetSession` (`returnhome.go:178-196`, `399-438`). `TestReturnHome` checks the state equals `New()` but for the answers, and `sameAsFresh` compares every model and UI field by reflection. It also checks the notification, Git Graph read, store and registration end, the draft is saved and comes back at the next start, and a second session draws only its own entries at 120x40 and 200x60.\nPASS 15  75.15 Esc on the home screen [read]: Esc returns nil on the home screen (`keys.go:113-117`), and the startup machine's quit row in `doc/events-and-state.md` says so. `TestHomeEsc` checks it, and Ctrl+Q still quits.\nPASS 16  75.16 The machines [read]: Every new row and invalid clause has its case, and the coverage check requires exactly one case per row (`internal/state/machines_test.go:1632-1676`). That `TestStateMachines` passes is reported, from the uncached run.\nPASS 17  75.17 Broken on purpose [reported]: `evidence.txt:134-473` lists 23 step-1 and 28 step-2 mutants run through `go test -overlay`, all caught, covering every mutant 75.17 names. The fingerprint, index and diff hashes are equal before and after. The mutant scripts are outside the repository, so what each mutant changes is the evidence's claim.\nPASS 18  75.18 The checks [reported]: build, vet on Windows and Linux, `gofmt -l` (prints nothing) and `go test -count=1 ./...` all exit 0, the fingerprint 32076e9… equal before and after (`orchestrator.txt:319-347`). `-race` passes on state, input, render and app (`orchestrator.txt:370-383`). The timing step ran alone, on mains power, with doc/evaluation.md:138's exact command, all within budget (`orchestrator.txt:416-587`).\nPASS 19  75.19 The live run [reported]: On the final build (sha 5df11cce…), L75.1 to L75.9 passed for Claude Code (`orchestrator.txt:829-839`) and Codex (`1422-1432`). The screen text shows the entry with its model and title (`1146-1147`, `1732-1733`). projects.json was put back with equal SHA-256, and the temporary projects were moved to the Recycle Bin (`2224-2230`). The owner approved the extra turns (`287-297`).\nPASS 20  75.20 The documents [read]: The changes to `ui-design.md`, `events-and-state.md`, `reliability.md`, `architecture.md`, AGENTS.md (`AGENTS.md:27`, `245-247`) and amendments 89 to 91 in `doc/brief.md` match the code at every point checked. The Nice to have item leaves plan.md (`changes.diff:3232-3233`). The BOM/CR/user-name check is reported (`evidence.txt:2-110`); my own search found no CR and no user name in the new files and the docs.\nPASS 21  75.21 The scope [read]: Every tracked file in the diff and every file in new-files.txt is within the plan's Files. Nothing under `internal/provider`, `cmd`, `scripts` or `.claude` changes, and `go.mod`/`go.sum` are not in the diff.\n\nObservations:\n- Medium, high confidence: in the delete question, ← and → both toggle the selection (`internal/app/history.go:162-163`). The buttons are drawn Delete (left), then Cancel (right) (`internal/render/history.go:344-347`). So with Cancel selected, → moves to Delete, and → then Enter deletes the entry. The return question's arrows are directional (`returnhome.go:122-125`). No criterion is broken (D4 says only \"←/→ … answer it\"), but an entry can be deleted by mistake.\n- Low, high confidence: a title can lose a letter in one edge case. When the first message is over 8 KiB and its first 8 KiB has fewer than 60 visible characters, the last character is dropped although white space follows it in the message: \"hello\" + 9,000 spaces + \"world\" gives \"hell\" (`internal/render/history.go:125-131`). A test pins this behaviour (\"ab \" gives \"a\", `render/history_test.go:52`).\n- Low, medium confidence: an image-only first message marks the session as prompted with no text (`reduce.go:133-140`), so the session never gets a title, even from a later text message (`state/history_test.go:67-79`). D1a's \"images do not count\" allows this reading, and the docs don't say it.\n- Low, high confidence: Back after a failed start now runs a session end (`keys.go:370-374`). Until it finishes, START SESSION and Ctrl+R are refused with \"The last session is still closing\" (`keys.go:180-182`, `lastsession.go:46-48`). It is normally a matter of milliseconds and is justified (the old turn store was kept), but D9 keeps Back \"the direct back it is today\" and the home screen's description of Back doesn't mention it.\n- Low, high confidence: `doc/reliability.md:420` cites `doc/plan75-evaluation.md`, which does not exist until step 4.\n- Low, high confidence: `doc/architecture.md:842` and `:857` add two whole sections, beyond the plan's \"where it describes a session's life\". 75.21 holds at file level.\n- Low, high confidence: \"Not now: an approval is waiting · F2 answers it\" (`actions.go:387`) is not one of D8's texts, though D8 counts a waiting approval as part of the turn. The test notes no CLI event leads to that state (`returnmore_test.go:245-246`).\n- Low, high confidence: the two changed startup rows (`Asking | resume pressed…`, `Starting | a resume the CLI refuses…`) keep plan 69's machine cases unchanged. Their plan 75 parts are tested only at the application level (`TestHistoryResume`, `TestHistoryCtrlR`).\n- Low, high confidence: production code counts every title cluster step in a package-level atomic counter used only by `TestSessionTitleCost` (`render/history.go:138-151`).\n- None (interpretation only): grid mode shows the full footer row only from 100 columns (`render/home.go:297`); at 99 columns the 99-cell row would just fit. This follows plan 16's one-cell rule, documented in ui-design.\n\nRequested tests:\n- If the owner wants the delete question's arrows directional: with Cancel selected, → keeps Cancel and ← selects Delete.\n- A title case: `SessionTitle(\"hello\" + 9,000 spaces, true)` gives \"hello\", or a documented decision that it gives \"hell\".\n\nVerdict: PASS"
}
````
