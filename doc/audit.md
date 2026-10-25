# Architecture and code-quality audit (2026-10-25)

The owner asked on 2026-10-25 for an independent, read-only Staff-engineer
audit of the whole codebase at commit `75c133d`, then for a final review of
its findings' severities and of its roadmap. Part 1 is the audit as it was
delivered; Part 2, the final assessment, re-examines it with new
reproductions and measurements and supersedes Part 1 wherever the two
differ (severities, recommendations, roadmap). Plan 46 implements the first
three fixes of Part 2's roadmap (brief amendment 47).

# Part 1: the audit

Date: 2026-10-25. Subject: commit `75c133d` on `main`, working tree clean. Scope: the whole active codebase, 43,500 source lines in 19 packages and 57,300 test lines. The patched copy of `charmbracelet/ultraviolet` was reviewed at its patch points only.

### How the audit was done

- **Four independent read-only reviewers**, each a complete pass of one area:
  - rendering (`internal/render`);
  - the application layer (`internal/app`);
  - the protocol, state and process boundary (`internal/provider/*`, `proc`, `event`, `state`);
  - input, the git engine and the tooling (`internal/input`, `diff`, `confcheck`, `cmd`, the build and the test data).

  Each was briefed with the review dimensions only. None was given a list of suspected problems.
- **My own repository-wide evidence:** complexity and size metrics from an AST scan, change hotspots from git history, statement coverage, how golden frames move together, the import-boundary test, and build, vet and gofmt.
- **Every finding a reviewer raised was checked by me in the code.** Where reading could not settle a claim, I ran a probe that writes nothing into the repository: a test added only through `go test -overlay`, or a git probe on temporary repositories. Two reviewer claims were refuted that way; see "Checked and refuted".
- **The repository was never modified;** `git status` stayed empty throughout. One reviewer's shell pipeline left a scratch file, `/tmp/fns.txt`, outside the repository.
- **Disclosure:** several findings concern code I wrote in plan 45 (the commit flow, the quit path, the live label). The reviewers did not know that.

Status labels: **Confirmed** means shown by code evidence or a reproducible check. **Potential** means plausible but not demonstrated, and names the test that would settle it. **Not verified** means the evidence is insufficient.

---

### A. Executive assessment

**Overall, this is a well-engineered codebase. It does not need broad refactoring.**

**What is well designed:**
- **The documented architecture is the real one, and it is enforced.** Five parts with one-way data flow. `internal/arch` fails the build if render, input or state reach a provider, a process or the app, or if the reducer imports I/O or reads the clock. A planted-fault test proves the rule can fail.
- **The reducer is pure and deterministic.** It is replayed against 36 recordings of real CLI sessions; `TestReplayDeterministic` compares two replays.
- **The composition root injects every effect** (providers, clipboard, git, drafts, projects, sessions, clock, timers). That is why 57,000 lines of tests run deterministically under `synctest` with no sleeping.
- **Coverage is high where the logic is:** diff 99.5%, state 94.1%, app 89.6%, both adapters 88.1%, render 87.1%, input 81.2%. proc reads 0.0% only because its test runs the code in a child process.
- **Function sizes are healthy:** median 10 lines, p90 42. Large files are cohesive by feature.
- **The delicate parts are done carefully:**
  - process trees (suspended start in a Job Object, close escalation, drain);
  - write ordering (Hold/Release);
  - stale-result discipline (generation counters);
  - crash-safe drafts (temp file, sync, rename, held locks);
  - fail-safe approvals;
  - secret redaction;
  - the git commit and push verification chain;
  - a documented and tested fork of one dependency.

**What may need improvement.** No defect found is Critical or High. The important findings fall into four groups:
1. **Recoverable events end the whole session:** a refused optional read during the Codex handshake, and an approval answered just as the CLI withdraws it.
2. **Robustness at untrusted-input and failure boundaries:**
   - image decoders that a crafted file can drive into unbounded allocation;
   - no panic recovery in goroutines the program starts itself;
   - no fuzz tests at all.
3. **One structural weakness with demonstrated small defects:** which overlay, popup, window or panel owns the screen and the input is encoded separately in several places that must agree, and already disagree.
4. **A few git lifecycle edges and correctness slips,** each cheap to fix.

**Technical risks.**
- **Most likely to hurt the owner:** a dropped session (findings 1 and 2), and a misaligned screen on rows containing emoji sequences (finding 5).
- **Most expensive in the long run:** overlay logic that every new UI feature must thread through five places (finding 4).
- **Performance:** about 3x headroom today. The timing tests do not exercise large inputs, so the margin under heavy tool output is unmeasured (finding 7).

**Is significant architectural change warranted?** No. The project would gain more from:
- targeted fixes;
- characterization tests in four or five specific gaps;
- hardening of the failure boundaries;
- optionally, one bounded structural consolidation (interaction layers), when the next overlay feature is planned.

---

### B. Architecture assessment

**Parts and data flow** (as documented in `doc/architecture.md`, and as implemented):
1. **Execution, `internal/provider/{claude,codex}` and `internal/proc`.** Spawns the CLI suspended inside a Job Object (KILL_ON_JOB_CLOSE), then resumes it. Writes user messages and control answers; handles interrupt and shutdown (close stdin, then terminate, then kill).
2. **Collection.** Each adapter's `Mapper` turns CLI lines into normalised `event.Event`s. Reader goroutines push them into an unbounded, ordered `pipe.Queue`, so the CLI's pipe never blocks. Unknown lines become `Unrecognised` events, never silently dropped.
3. **State, `internal/state`.** One reducer, `Apply(event)` and `Do(action)`. It returns `Command`s, and the app executes them. There is no I/O and no clock.
4. **Rendering, `internal/render`.** `Draw(state, UI) -> *Screen`. It is pure: no goroutines, no clock, no I/O. Spans carry refs, so drawing, hit-testing and copy come from one structure. Sixel images are written after the frame onto cells the frame left blank, and only where the final frame still shows them.
5. **Input, `internal/input`.** In practice this is the provider-independent layer for everything outside: the editor, keys, clipboard, drafts, projects, session locks, the file viewer, image decoding, the GDI text masks and the git engine.

`internal/app` is the Bubble Tea model. It owns the state, render's `UI` and three editors. It turns commands into `tea.Cmd`s, runs session writes through one FIFO worker, and guards asynchronous results with 13 generation counters.

**Strengths, from evidence:**
- The boundaries hold in the code, not only in the documents. The reducer's purity makes every state transition testable without a terminal or a CLI.
- **Concurrency is concentrated.** Explicit goroutines exist only in app (3, the write worker among them) and the adapters (16), with none in input, render, state or proc. The app's other asynchronous work runs as Bubble Tea commands. Ownership is clear: one event loop, one writer, readers per process.
- **Tests replay real protocol traffic** (`testdata/probes`) and drive the real program with real key and mouse messages; 220 golden frames at four sizes.

**Weaknesses, from evidence:**
- **The interaction layer has no single model.** Each overlay, popup or window added by plans 30 to 45 added conditions in:
  - `render.Draw` (cyclomatic complexity 87);
  - `Model.mouse` (91), `Model.dashboardKey` (79) and `Model.Update` (69);
  - `motionFilter`;
  - `syncUI`;
  - the panel open and close paths.

  These are among the most-changed files: `render/dashboard.go` in 31 commits, `app/app.go` 29, `app/keys.go` 24, `render/select.go` 22. That is finding 4.
- **The two adapters duplicate their session lifecycle** and have started to drift (finding 9).
- **Some per-frame work grows with session length or input size,** and the timing tests do not exercise that (finding 7).

---

### C. Findings

Ordered by real risk and value.

#### 1. Recoverable protocol events end the whole session
- **Priority:** Medium. **Category:** reliability and failure handling. **Confidence:** high.
- **Status:** handshake path Confirmed by code. Answer path: the crash is Confirmed by code, the triggering race is Potential.

**Evidence, handshake:**
- After `initialize`, the Codex session sends `config/read`, `configRequirements/read` and `model/list` before `thread/start` (`internal/provider/codex/session.go:396-410`).
- The mapper turns a refused `config/read` (`codex/mapper.go:1317-1320`) or `model/list` (`codex/mapper.go:313-323`, `:377-384`) into an `event.Error`.
- The reducer crashes on any Error while handshaking (`internal/state/reduce.go:188-190`): `if ev.Kind == event.Error && s.Session.Phase == Handshaking { return s.crash(ev, "handshake failed: "+p.Message, true) }`.
- The intent is visibly non-fatal elsewhere: a refused `configRequirements/read` is only a status line, and a refused `model/list` should "say why" in the selector. During the handshake, the accompanying Error ends the session first.

**Evidence, answers:**
- `state/actions.go:201-205` crashes the session on any `AnswerFailed`.
- The adapters return "no pending request" (`claude/session.go:415-420`; Codex likewise) when the CLI has already withdrawn the request (`control_cancel_request`, `serverRequest/resolved`) but the reducer has not yet applied the withdrawal.
- An answer typed in that window ends the session (`app/app.go:758-764` turns the error into `AnswerFailed`).

**Root cause.** Fatality is decided from the event's kind (Error) rather than from what failed.

**Impact.** A usable live session is closed by:
- a refusal of an optional read (for example after a Codex update that renames a method);
- a harmless race between the owner's key and the CLI's cancel.

**Proposed improvement:**
- Mark explicitly in the `Problem` payload which refusals are fatal during the handshake (`initialize`, `thread/start`, `thread/resume`), and let the reducer read that mark.
- Return a sentinel `provider.ErrNoPending` from `Answer`. The reducer then expires the approval with a notice and crashes only on write failures.

**Alternatives.** Reclassify optional reads as `Status` events in the mapper. That is simpler, but it hides refusals the owner should see.

**Trade-offs and risks.** A small payload change. Some mock or golden updates.

**Complexity:** small.

**Verification:**
- Feed a `model/list` error response during the handshake: the session stays in Handshaking, then reaches Ready.
- A fake session that returns `ErrNoPending` after a cancel: the approval is expired and the session stays alive.
- The existing replay tests must not change.

#### 2. Image decoders can be driven to unbounded memory and CPU by small crafted files
- **Priority:** Medium. **Category:** robustness and security (untrusted bytes). **Confidence:** high on the mechanism.
- **Status:** mechanism Confirmed by code; not executed (it would exhaust the machine's memory).

**Evidence:**
- **TIFF:** `internal/input/imagetiff.go:46-65` allocates `make([]uint32, count)` for every directory entry with `count <= 1<<20`, and keeps it in `tags[tag]`. A directory can hold 65,535 entries, and their data offsets may overlap. A file of a few MB can therefore ask for hundreds of GB and about 10^10 loop iterations, far beyond the 64 MB inspection limit; only about nine tags are ever used.
- **ICO:** `imageico.go:47` calls `png.Decode` on an embedded PNG without `DecodeConfig` and a size check first.
- **GIF:** `image.go:361` calls `gif.DecodeAll`, which keeps every frame.
- Saved draft images are inspected again on restart (`app/draft.go:219`). If such a file reached a draft, the dashboard could crash at every start. Not verified.

**Impact.** A crash with Go's unrecoverable out-of-memory error, from a clipboard image or an uploaded file. In a single-owner setting this is mainly a robustness issue.

**Proposed improvement:**
- TIFF: keep only the tags read.
- ICO: `png.DecodeConfig` plus the existing `tooBig` check before decoding.
- GIF: decode the first frame only, or cap the frame count.

**Alternatives.** A global `GOMEMLIMIT` does not prevent the crash.

**Trade-offs.** None material.

**Complexity:** small.

**Verification.** Crafted-file cases for each decoder under `GOMEMLIMIT`, plus fuzz targets (finding 6).

#### 3. Git operation lifecycle edges
- **Priority:** Medium (low likelihood, but one path is destructive). **Category:** repository safety and resource lifecycle. **Confidence:** high.

**3a. `GitCLI.Merge` can abort a merge it did not start.**
- Confirmed by code: `internal/input/git.go:353-364`.
- If `git merge` fails while a MERGE_HEAD exists, the code runs `git merge --abort`, including when the MERGE_HEAD predates the call. That discards an owner's in-progress merge resolution.
- The app's "main has changes" guard (`app/worktrees.go:237`) covers most cases, but not a merge begun in a terminal between the check and the call.
- Fix: read MERGE_HEAD before merging, refuse if present, and abort only a merge this call started.

**3b. Quitting ignores a running commit or push.**
- Confirmed by code: nothing outside `app/commit.go` consults `m.op`; `Quit` and `Shutdown` (`app/app.go:969-971`) never do.
- Ctrl+Q mid-commit leaves git running as an orphan, or kills it with the console. Probe P45a showed that a stopped git leaves `index.lock` behind.
- Fix: treat a running operation like a running turn in the quit question.

**3c. Git timeouts do not bound a child that keeps the pipe.**
- Confirmed absence: `input/git.go` `call` and `stream` use `CommandContext` with no `cmd.WaitDelay`. The project already knows the remedy: `provider/host.go:53` sets it for detection.
- A killed git whose hook or helper holds stdout keeps `Wait` blocked, so the status poll or an operation stays busy for ever. Potential.
- Fix: `WaitDelay` on reads and operations.

**Complexity:** small for each.

**Verification:**
- A pre-existing MERGE_HEAD case for 3a.
- Quit with a blocking fake commit for 3b.
- A fake git on PATH whose child holds stdout, for 3c.

#### 4. Interaction-layer precedence is encoded in several places that must agree
- **Priority:** Medium. **Category:** maintainability, with demonstrated small defects. **Confidence:** high on the pattern.
- **Status:** pattern and two defects Confirmed.

**Evidence:**
- `render/select.go:166-213` repeats a seven-way precedence by hand:

  ```text
  ui.Plus && !WindowOpen(ui) && !ui.DraftWin.Open && !ui.Keys && !ui.Worktrees.Open && ui.Width >= MinWidth ...
  ```

  The copies already differ: the commit dropdown omits `WindowOpen` and `DraftWin`, and the floating window omits `DraftWin`.
- The same exclusions are encoded again in the key and mouse routing (`app/keys.go`, `app/models.go:54`, `app/modes.go:45`, `app/mouse.go:45`, cyclomatic complexity 91), in `motionFilter`, and in `syncUI`.
- Panel transitions are not funnelled. `openExplorer` hides Source Control directly (`app/explorer.go:44`) instead of through `closeSource` (`app/source.go:256-263`), and F3 passes through while the staging question is open (`app/commit.go:499`). The question therefore stays armed, and the next click anywhere is swallowed by `app/mouse.go:59-69`. That is a real defect, from plan 45.
- Popups find their anchors by scanning the drawn frame for particular glyphs and styles (`render/menu.go:72-92`; `render/commit.go:358-419` requires `"│"`). A decoration change can silently disable a popup.

**Root cause.** Each feature added its overlay where it was needed. No component owns "which layer is on top, takes input, and is reset by what".

**Impact.**
- Every new UI feature must be threaded through five places.
- Disagreements surface as swallowed clicks, stale state or a popup drawn under the wrong window.
- The cost and the risk grow with each plan.

**Proposed improvement:**
- One ordered model of the interaction layers, with the predicate "is layer X active and on top" computed once and consumed by `Draw`, key routing, mouse routing and `motionFilter`.
- Panel and popup transitions through single open and close methods.
- Popup anchors recorded by the draw function that owns them (in `Screen`), as `Conv`, `Panel`, `Side` and `Diff` already are.

**Alternatives.** Fix the two defects in place and stop. That is reasonable today; it leaves the cost of the next feature unchanged.

**Trade-offs and risks.** This touches the hottest, most-changed code. Behaviour must be pinned first.

**Complexity:** medium.

**Verification:**
- An overlay-pair matrix written before any change: draw order, hit areas, key ownership and resets for every pair among Plus, Models, Modes, the commit dropdown, the staging question, the file window and the draft window.
- The 220 golden frames and the key and mouse suites must stay identical.

#### 5. Text width is measured per code point, not per grapheme cluster
- **Priority:** Medium-Low. **Category:** rendering correctness. **Confidence:** high on the discrepancy.
- **Status:** discrepancy Confirmed by an overlay test; on-screen effect Not verified.

**Evidence.** `render/text.go:189-213` sums `runeWidth` per code point. Measured against lipgloss's cluster-aware width:

| Text | Per code point | Per cluster |
| - | - | - |
| ❤️ | 1 | 2 |
| 👍🏽 | 4 | 2 |
| 👨‍👩‍👦 | 6 | 2 |
| 🇫🇷 | 4 | 2 |
| Combining accents | equal | equal |

Bubble Tea measures clusters once Windows Terminal reports Unicode mode 2027 (`third_party/ultraviolet/PATCHES.md`), so render's padding and cutting disagree with the terminal on rows containing such sequences.

**Impact.** Shifted borders and misaligned zones on rows that show emoji sequences: messages, file contents, commit messages.

**Proposed improvement.** A cluster-aware `textWidth`, `cut`, `cells` and `units`, keeping the ASCII fast path.

**Trade-offs.** It touches hot paths. Needs the keystroke timing tests and a styled-frame check.

**Complexity:** medium.

**Verification.** A test that compares each drawn row's `ansi.StringWidth` with the screen width, for VS16, skin-tone, ZWJ and flag clusters.

#### 6. Failures outside Bubble Tea's goroutines are not contained, and nothing is fuzzed
- **Priority:** Medium-Low. **Category:** failure containment and testing. **Confidence:** high on the absence.
- **Status:** absence Confirmed; consequence Potential.

**Evidence:**
- No `recover` anywhere in `internal/provider`, `input` or `proc`.
- In app, the write worker (`app/writer.go:23-31`) and the detection goroutine have none either. Bubble Tea recovers only its own update loop and commands.
- A panic in a mapper (on unexpected but valid JSON from the CLI) or in a provider write therefore ends the process with the terminal still in alternate-screen, mouse and win32-input modes. The Job Object does kill the CLI tree, so nothing is orphaned.
- No `func Fuzz` exists in the repository, although several parsers take external input: the two protocol mappers, five image decoders, git porcelain and branch parsing, and the pasted-path parser.

**Proposed improvement:**
- Recover at the goroutine boundaries the program owns: the reader loops and the write worker. Turn a panic into a visible Error event and an orderly end of the session.
- Fuzz targets seeded from `testdata/probes` and `testdata/images`, run for a bounded time in the evaluation's routine.

**Trade-offs.** A swallowed panic hides a bug unless it is shown. It must be shown.

**Complexity:** small.

**Verification.** A fake session whose `Send` panics, run through the real program: the terminal is restored, and the error is shown or logged.

#### 7. Performance that scales with input size is not measured
- **Priority:** Medium-Low. **Category:** performance. **Confidence:** medium.
- **Status:** mechanisms Confirmed by code; frame impact Potential.

**Evidence:**
- **Quadratic, uncapped tool output:**
  - `state/reduce.go:496` appends each Codex `outputDelta` (`codex/mapper.go:642-645`) with `t.Output += p.Output`. 10 MB in 1 KB deltas copies about 50 GB in the reducer, which runs on the event loop.
  - `render/content.go:60` splits the whole output of a failed command on every frame.
- **Parsing on every frame:** each tool row parses its tool's full input JSON into a map twice (`render/tree.go:287-290`, `:411-420` via `render/approval.go:275-277`), for every tool of the open stage, before the visible window is cut.
- **Growth with session length:** per-frame scans over all tools and entries (`tree.go:205-230`, `zonec.go:91-110`, `look.go:80-148`). Measured on this tree: `TestLongSession`'s frame p95 rises from 1.8 ms in the first tenth of a two-hour synthetic session to 4.2 ms in the last, and the end-of-session p95 is 5.0 ms. That leaves about 3x headroom against 16 ms.
- The timing tests use small inputs, so none of this is measured.

**Proposed improvement:**
- First, a large-input timing test: about 30 Write calls of 100 KB, and 10,000 output deltas of 1 KB.
- Only if a target is missed: keep the tail of streaming output (the completed item replaces it anyway), and memoise per-tool derived fields keyed by tool and input length.

**Trade-offs.** Cache invalidation.

**Complexity:** small to medium.

#### 8. The approval card shows Unicode format characters as they are
- **Priority:** Low-Medium. **Category:** security hardening. **Confidence:** medium.
- **Status:** Confirmed in code; visual effect in Windows Terminal Not verified.

**Evidence.** `render/text.go:59-87` (`visible`) neutralises C0, DEL and C1 only. Bidi overrides (U+202E, U+2066 to U+2069) and zero-width characters (U+200B to U+200F) reach the approval card unchanged (`render/approval.go:86-137`).

**Impact.** A proposed command can read differently from what runs. How much depends on the terminal's bidi support. The approval card is the owner's main safety gate.

**Proposed improvement.** Draw Unicode format (Cf) characters as visible atoms, `<U+202E>`, as C1 controls already are.

**Complexity:** small.

#### 9. The adapters duplicate their session lifecycle, and it has drifted
- **Priority:** Low-Medium. **Category:** defect risk and concurrency. **Confidence:** medium.
- **Status:** duplication and drift Confirmed; deadlock Potential.

**Evidence:**
- The same constants, framing, stderr capture, wait and drain, escalation and `Close` appear in `claude/session.go` and `codex/session.go`.
- The Codex reader never writes to stdin itself (`codex/session.go:211`, "the reader never waits on stdin").
- The Claude reader writes inline: `get_settings` and "/usage" at the handshake, and the settings read after a change (`claude/session.go:183-205`). It takes the same write lock as a large `Send`.
- If the CLI stops reading stdin while its stdout fills, the reader blocks on the lock, and the pipes deadlock.

**Proposed improvement:**
- Move the Claude reader's writes off the read loop, as Codex does, keeping the handshake's hold.
- Extract a shared lifecycle runner only if a third adapter arrives, or the two drift again.

**Verification.** A fake process that stops reading stdin.

**Complexity:** small for the fix; medium for the extraction.

#### 10. Source Control shows a stale untracked folder
- **Priority:** Low-Medium. **Category:** correctness. **Confidence:** high.
- **Status:** Confirmed.

**Evidence:**
- An opened untracked folder is listed once (`app/source.go:397-399`) and never read again while it stays untracked (`:431-443`).
- `folderRead` ignores the generation counter it is given.

**Impact.** While the agent adds files to a new folder, the panel keeps showing the first listing, even though the commit question is right.

**Fix.** Re-read opened folders on each snapshot, as the Explorer does.

**Complexity:** small.

#### 11. Reducer correctness slips
- **Priority:** Low. **Category:** correctness. **Confidence:** high.
- **Status:** Confirmed.

**Evidence:**
- `case event.Mode:` (`state/reduce.go:213-214`) has no `return`. Every Mode event therefore falls out to `s.note(ev, "event without a known payload")` (`:238`). The replay of p12c, p12x, p13c and p303c shows these false notes. They push real anomalies out of the 50-entry diagnostic ring; no screen shows the ring.
- Provider vocabulary leaks into the reducer: `reduce.go:706` picks "decline" from `s.Context.Provider`, while `actions.go:329` uses `s.CLI()`, and the two disagree in mock replays of Codex.

**Fix:**
- `return nil`, plus a replay assertion that no such note appears.
- Optionally, normalised decisions that each adapter translates.

**Complexity:** trivial to small.

#### 12. Test-suite gaps that matter before any refactoring
- **Priority:** Low, but enabling. **Category:** testability. **Confidence:** high.
- **Status:** Confirmed.

**Evidence:**
- **The queues:** `pipe.Queue`, the ordering primitive between readers and the program (Hold, Release, Seq), has no test beyond `TestReadLine`. The app's write queue has none either; no test mentions `writeQueue` or its "too many writes waiting" refusal.
- **Reducer notes are never asserted,** which is how finding 11 went unseen.
- **Golden frames:**
  - they are text-only; style is covered only by palette tests;
  - a change to shared screen parts has rewritten 50 to 81 frames in one commit (plans 24, 29, 34, 39 and 43), which makes reviewing them costly.
- **Wall time:** `internal/input` takes about 174 s, 135 s of it in five sequential git tests; there is no `t.Parallel` in input or diff.

**Proposed improvement:**
- Tests for both queues.
- A notes assertion.
- A per-zone summary of golden-frame changes, so a 60-frame update can be reviewed by zone.
- `t.Parallel` in the git tests, which use their own temporary repositories.

**Complexity:** small.

#### 13. Hygiene and documentation drift
- **Priority:** Low. **Category:** maintainability. **Confidence:** high.
- **Status:** Confirmed.

**Items:**
- **Stale comments:**
  - `cmd/agent-command-center/main.go` says the adapters' sessions "arrive in steps 2 and 5"; both have long been complete;
  - input's package comment (`editor.go:1`) still calls it "the input editor and the clipboard".
- **A latent path bug:** `input/browse_windows.go:62,65` holds `` `\?\` `` and `` `\?\UNC\` `` with one backslash, so `realPath` never strips the `\\?\` prefix it means to strip. Behaviour is right today because comparisons are real against real.
- **Dead code and duplicate helpers:**
  - dead `style()` (`render/render.go:165`);
  - `fitText` and `fitLabel` behave identically;
  - three `base` variants that differ at edge cases.
- **Inconsistencies in the git engine:**
  - worktree and diff reads bypass `runRead`, so they skip `core.fsmonitor=false`;
  - git inherits `GIT_DIR`, `GIT_WORK_TREE` and `GIT_INDEX_FILE` from the environment;
  - `Projects.Update` is a read-modify-write with no lock between two dashboards. The file itself is safe: it is written to a temp file, synced, then renamed.

**Complexity:** trivial each.

---

### Checked and refuted

- **"Push could force-push to a mirror-configured remote"** (input reviewer). Refuted on temporary repositories with git 2.56: with `remote.origin.mirror=true`, the dashboard's exact push command fails with `fatal: --mirror can't be combined with refspecs` (exit 128), and the remote is unchanged.
- **"`TextMask` leaks two GDI objects per call"** (input reviewer). Refuted by measurement, through an overlay test: 500 calls changed the process's GDI object count by 0, while a control that leaves 10 brushes undeleted moved it by 10.
- **"`internal/proc` is untested"** (the coverage figure). Not so: `TestJobKillsTreeWithDashboard` runs the code in a child process, which coverage cannot count.

---

### D. What should remain unchanged

1. **The five-part architecture and `internal/arch`.** The rules are enforced, and the planted-fault test proves they can fail. Weakening or "simplifying" them would lose the main structural guarantee.
2. **The pure reducer with returned `Command`s,** and the Mapper/Session/State split. Every state transition is replayable from real recordings. `reduce.go` and `mapper.go` are large but cohesive and fixture-backed; size is no reason to split them.
3. **The `Provider`/`Session`/`Capabilities` interface.** It fits both CLIs, with capability flags for the differences. More abstraction would add indirection without a second use.
4. **Process control:**
   - the suspended start in a Job Object, the close escalation and the drain;
   - Hold/Release write ordering;
   - the unbounded event queue. It is a documented choice that keeps the CLI from blocking; its memory is bounded by the CLI's output.
5. **The composition root and the injected seams** (`app.Options`). They are what makes the large deterministic test suite possible.
6. **The large `Model` and the file-per-feature organisation of `internal/app`.** One event loop and one owner of state is right for Bubble Tea. Splitting the Model would add indirection and prevent no demonstrated defect. If anything, group the Source Control fields into one sub-struct when that area next changes.
7. **Generation counters, the write queue and the draft store.** They are correct as written. Replacing them is churn.
8. **Draft persistence.** Temp file, sync, rename, held folders, never touching a held folder, setting a damaged list aside. This is the most carefully reasoned I/O in the codebase, and it is covered by crash tests.
9. **The commit and push verification chain and `GitStop`.** Every precondition is checked, success is confirmed from git, every git call is traced in tests that forbid force, amend, no-verify and all-refs flags, and the push stop is reasoned from probes. Add the edges of finding 3 inside them; do not restructure.
10. **Render's span/ref model, `layout()`, the `WrapCache` with its sweep, the shared-style fast path,** and post-frame Sixel images with the final-frame check. These are the reasons rendering is fast and correct by construction.
11. **The by-value `UI`.** Callers rely on copy semantics: `zonec.go:239` assigns to its own copy. The 2.4 KB copies are inside the measured budget.
12. **`internal/input` as the outside-world adapter layer.** It is cohesive by role, not by name. Fix its package comment. A git-engine package of its own is optional: worth it if the git code keeps growing, or to shorten the test cycle (see roadmap phase 5).
13. **`internal/diff` as a pure package,** with its minimality and speed tests.
14. **The patched ultraviolet copy and its process.** `PATCHES.md` gives the source, a reason and a check for each patch. The patch tests run in the suite through `internal/licenses`. There is an update procedure. Keep it until Bubble Tea moves.

---

### E. Recommended roadmap

Small phases that can each be verified on their own. Phase 0 makes the later ones safe. Phases 1 and 2 hold most of the value. Phases 3 to 5 are conditional or optional.

#### Phase 0: characterization tests, no production change
- **Objective:** pin the behaviour the later phases touch, and expose the gaps of finding 12.
- **Scope:**
  - `pipe.Queue` tests: Hold, Release with a drop, Close, Seq order;
  - write-queue tests: FIFO answer-then-interrupt, the full queue;
  - a reducer-notes assertion over all recordings;
  - a handshake-fatality matrix;
  - answer-failure cases;
  - an overlay-pair matrix;
  - a large-input timing test;
  - a grapheme-width row test;
  - a pre-existing-MERGE_HEAD case;
  - fuzz targets for the image decoders and both mappers, seeded from `testdata`.
- **Benefit:** later changes become provably behaviour-preserving, and the fuzzers may find real panics.
- **Risk:** none to production.
- **Acceptance:** the new tests pass on today's code, except those written to expose a confirmed defect, which are recorded as expected failures. The fuzzers run for a fixed time without a crash, or their crashes become Phase 1 or 2 items.

#### Phase 1: targeted correctness and safety fixes
- **Objective:** remove the confirmed defects, each with its test from phase 0.
- **Scope:**
  - finding 1: explicit handshake fatality; `ErrNoPending` expires instead of crashing;
  - finding 3: Merge pre-check; quit asks while a git operation runs; `WaitDelay`;
  - finding 4's defect: `openExplorer` through `closeSource`;
  - finding 10: re-read opened folders;
  - finding 11: the `return`;
  - finding 13: the path literals and the stale comments.
- **Benefit:** fewer dropped sessions, no destructive merge abort, no stale panel.
- **Risk:** low; each change is local.
- **Tests:** phase 0's, plus the full suite and the golden frames unchanged.
- **Acceptance:** every confirmed defect's test passes; `go test ./...`, `-race` on render, app and input, and the timing step are within their targets.

#### Phase 2: robustness at the boundaries
- **Objective:** a bad input or a panic never takes the dashboard down silently.
- **Scope:**
  - finding 2: decoder bounds;
  - finding 6: recover at the goroutine boundaries the program owns, shown as errors;
  - finding 9: Claude's reader writes off the read loop;
  - finding 8: format characters drawn visibly on the approval card.
- **Benefit:** no crash loops; the terminal is restored on a panic; the safety gate is honest.
- **Risk:** low to medium. The Claude handshake ordering must keep its hold.
- **Tests:** crafted files under `GOMEMLIMIT`, the fuzzers, a panicking fake session, a stalled-stdin fake process, a U+202E approval card.
- **Acceptance:** all of these pass, and the probes' replays are unchanged.

#### Phase 3, conditional: performance headroom
- **Condition:** only if phase 0's large-input timing test misses a target.
- **Scope:** keep the tail of streaming tool output, memoise per-tool derived fields, and bound the per-frame scans (finding 7).
- **Acceptance:** the large-input test is within 16 ms at p95, with the existing timing tests no worse.

#### Phase 4, optional: consolidate the interaction layers
- **When:** the one refactoring with a defensible long-term benefit (finding 4), best done just before the next overlay feature, and after phase 0's overlay matrix exists.
- **Scope:**
  - one ordered layer model consumed by `Draw`, key routing, mouse routing and `motionFilter`;
  - single open and close methods for panels and popups;
  - draw-recorded popup anchors.
- **Benefit:** a new overlay is added in one place; precedence bugs become impossible by construction.
- **Risk:** medium; this is the hottest code.
- **Tests:** the overlay matrix, the 220 golden frames byte for byte, the key and mouse suites, and the timing step.
- **Acceptance:** zero golden changes, the matrix green, and the timing step within targets.

#### Phase 5, optional polish
- Cluster-aware width (finding 5), if misalignment is seen in practice.
- A separate git-engine package, or simply `t.Parallel` in the git tests (finding 12).
- Normalised approval decisions (finding 11).
- Extracting the adapters' shared lifecycle (finding 9), if a third adapter or more drift appears.

---

### F. Overall recommendation

Agent Command Center does not need a broad refactoring. Its architecture is sound, documented and enforced. Its riskiest mechanisms (process trees, write ordering, approvals, drafts, git writes) are carefully reasoned and tested against real recordings. Its code is consistent and readable at the function level.

What it needs is modest and specific, in this order:
1. **Characterization tests** in a few gaps: the two queues, reducer notes, large inputs, overlay pairs and fuzzing.
2. **About ten small fixes** for confirmed defects: dropped sessions, a destructive merge-abort edge, quit during git, stale panel state, a reducer fall-through and a path literal.
3. **Hardening at the untrusted-input and failure boundaries.**
4. **Optionally, one bounded consolidation of the interaction layers,** timed with the next overlay feature, since that is where future features become expensive.

Everything else should stay as it is.

---

### Not verified

- On-screen effect of finding 5 (cluster width) in Windows Terminal; of finding 8 (bidi) in Windows Terminal; finding 2's crash-loop via drafts; finding 3c's hang; finding 9's deadlock; finding 7's frame cost with large inputs. Each finding names the test that would settle it.
- The race detector and the timing step were not run again for this audit (both passed on this tree today, in plan 45's evaluation); no live CLI run was made.


# Part 2: the final assessment

Date: 2026-10-25, commit `75c133d`. It follows Part 1 and does not repeat it.

New evidence this round (nothing in the repository was changed):
- The merge, reproduced on temporary repositories.
- The Codex handshake, reproduced through the real mapper and reducer in an overlay test.
- A crafted TIFF, measured at a bounded size.
- The reducer's cost on streamed command output, timed.
- Windows Terminal's rendering of bidi and zero-width characters, screenshotted.
- Code traces of key precedence, of the dialog paths, and of image persistence.

### A. Severity reassessment

#### Merge abort (finding 3a): Medium, retained, now reproduced
- **What is wrong:** `GitCLI.Merge` runs `git merge --abort` whenever a merge fails while a MERGE_HEAD exists, including one that existed before.
- **Reproduction:** with an owner's terminal merge in progress and a conflict file partly edited, the code's own sequence ran. The merge was refused (exit 128), the abort succeeded, MERGE_HEAD was gone, and the file was back to the main side. The resolution was lost.
- **Reachability:** the guard "main has changes" uses the worktree list read when the view opened or after the last operation (`worktrees.go:83`, `:167`). It is not re-read on confirmation. The path is: the owner opens Worktrees, merges in a terminal in the main worktree, resolves partly, then confirms a dashboard merge.
- **Conditions:** no malice and no tight timing; only an unusual but ordinary multi-tool workflow.
- **Consequence:** uncommitted work lost (data integrity). The repository itself is not corrupted.

#### Optional reads refused during the Codex handshake (finding 1a): Medium, retained, reproduced
- **Reproduction:** a refused `model/list` or `config/read`, fed through the real mapper and reducer before the session is ready, gives a crash and closes the CLI. A refused `configRequirements/read` does not, and after Ready none of them crashes.
- **What is theoretical:** the trigger. Every recorded session shows these reads succeeding; a refusal needs a Codex version change, or a catalogue or config problem.
- **Consequence:** Codex cannot be started from the dashboard at all while the condition lasts (availability). There is no data loss, and the message is clear.

#### Stale untracked-folder listing (finding 10): raised from Low-Medium to Medium-Low
- **What is wrong:** an opened untracked folder's listing is never re-read (`source.go:397-399`, `:431-443`).
- **New reasoning:** "Stage and commit" stages the folder entry itself (`git add -A -- web/`, as `TestCommitStaging` shows), and the staged-set check accepts the files under a folder entry. So files the panel never listed can be committed and pushed, for example a file an agent added after the folder was opened.
- **Reachability:** normal usage; no timing or malice needed.
- **Mitigation:** none today. Note that even a fresh listing leaves a short TOCTOU window; that window is inherent to staging a folder.

#### Quadratic streamed output (finding 7a): Medium-Low, retained, now measured
- **Measured:** streaming 4 KiB deltas cost the reducer 93 ms in total for 2 MiB, 1.6 s for 8 MiB and 18.3 s for 32 MiB, with late deltas taking about 17 ms each. This runs on the event loop that draws the screen.
- **What is theoretical:** whether Codex streams that much output; not verified without a live run.
- **Mitigation:** the completed item replaces the output (`t.Output = p.Output`), so the cost is transient and nothing would be lost by capping the stream.

#### Image decoders (finding 2): lowered from Medium to Low-Medium
- **Measured:** a crafted 1.0 MiB TIFF made the tag loop allocate 64 MiB with 16 entries and 256 MiB with 64, about 4 MiB per entry. At the format's maximum of 65,535 entries, a file under 2 MB asks for about 256 GiB: an unrecoverable out-of-memory crash. The ICO-embedded PNG path is the same class.
- **New evidence that lowers it:**
  - no persisted crash loop: a pasted image is attached, and so saved in the draft, only after inspection succeeds (`paste.go:113-120`), and restore re-inspects only images that were accepted before;
  - the trigger needs a deliberately crafted file that the owner chooses to attach.
- **GIF:** a large legitimate animated GIF costs frames × area in memory: a spike, normally not a crash.
- **Consequence:** availability only, with no data loss beyond at most 0.5 s of draft.

#### Overlay precedence spread over several places (finding 4): lowered from Medium to Low-Medium
- **Verified that no wrong action is possible today:**
  - key routing is a window-first `switch` (`keys.go:178-191`), so the commit dropdown cannot receive keys under a window;
  - a staging question left armed by F3 cannot be confirmed, because Source Control reopens only through a click (`mouse.go:87`), and the first click cancels the hidden question (`mouse.go:59-69`).
- **Real effect:** one swallowed click, and a dropdown drawn beside a window.
- **What remains:** a maintainability concern, not a correctness one.

#### Answer race (finding 1b, split out): lowered to Low
It needs the owner's key to land within milliseconds of the CLI withdrawing the request, before the reducer applies the withdrawal.

#### Approval card control characters (finding 8): lowered from Low-Medium to Low
- **Verified:** zero-width characters reach Windows Terminal, folded into the previous cell (`ultraviolet/styled.go:270-283`).
- **Screenshot:** Windows Terminal reorders nothing for U+202E or the isolates, and draws them, and U+200B, as nothing.
- **Effect:** no reordering deception and no hiding of visible text; only invisible extra characters. Homoglyphs, a different threat, are out of scope of this finding.

#### Other changes
- **Quit during a git operation (3b):** Low-Medium. Git is orphaned, or killed with the console, possibly leaving `index.lock`.
- **Reader writing to stdin inline (9):** Low-Medium, retained. Potential deadlock, never observed.
- **Panics and fuzzing (6):** Low. No panic observed; the mappers decode typed JSON.
- **Cluster width (5):** Low, optional. The discrepancy is confirmed, but cosmetic: a misaligned row.
- **Git WaitDelay (3c):** Low.
- **Per-frame parsing and session-length scans (7b and 7c):** Low, conditional. 3x headroom measured.
- **Reducer slips (11), test gaps (12) and hygiene (13):** unchanged at Low.

#### Revised recommendations
- **Approval card (8):** neutralise only the bidi controls (U+061C, U+200E, U+200F, U+202A to U+202E, U+2066 to U+2069) and the zero-width spaces (U+200B, U+2060, U+FEFF), on the approval card only. Not all Unicode format characters: U+200D, the zero-width joiner, is needed by emoji sequences and some scripts.
- **Handshake (1a):** mark the two optional reads non-fatal. Fatal stays the default, and the 60 s handshake timeout remains the backstop.
- **Overlays (4):** no layer model now. Fix the one transition, add an overlay matrix test, and extract a single draw predicate only when the next overlay arrives.
- **Images (2):** fix TIFF and ICO now. First-frame-only GIF decoding is optional.
- **Streamed output (7a):** the cap moves to phase 2. It is small and safe, and it is measured.
- **Phase 0:** keep only characterization tests that pass today. Each defect's test lands with its fix.

### B. Final prioritised findings

No finding is Critical or High.

| ID | Severity | Category | Confidence | Practical impact | Action |
| - | - | - | - | - | - |
| 3a | Medium | Data integrity | High, reproduced | Loses an in-progress merge resolution | Refuse when MERGE_HEAD exists; abort only a merge this call started |
| 1a | Medium | Availability | High, reproduced | Codex unusable while an optional read is refused | Mark `model/list` and `config/read` refusals non-fatal |
| 10 | Medium-Low | Data integrity | High | Unseen files in an opened untracked folder can be committed | Re-read opened folders with each snapshot |
| 7a | Medium-Low | Availability and performance | Medium, measured | Seconds of frozen UI while large output streams | Keep only the tail while streaming |
| 2 | Low-Medium | Availability (hardening) | High, measured | A crafted TIFF or ICO crashes the dashboard | Read only the needed TIFF tags; `DecodeConfig` before the ICO's PNG |
| 3b | Low-Medium | Lifecycle | Medium | Quitting mid-commit can leave `index.lock` | Count a running git operation in the quit question |
| 9 | Low-Medium | Concurrency | Medium | Possible pipe deadlock | Move the Claude reader's writes off the read loop |
| 4 | Low-Medium | Maintainability | High | One swallowed click; a cosmetic overlay | Route `openExplorer` through `closeSource`; overlay matrix test |
| 1b | Low | Reliability | Medium | Rare session crash | Return `ErrNoPending`; expire the approval with a notice |
| 11 | Low | Correctness | High | False diagnostic notes | `return nil`, plus a notes assertion |
| 3c | Low | Lifecycle | Medium | A git child can keep an operation busy for ever | `WaitDelay` |
| 8 | Low | Security hardening | High, screenshot | Invisible characters on the approval card | Neutralise the bidi and zero-width set above |
| 6 | Low | Safeguards | Medium | Unrestored terminal on a panic | Fuzz targets; `recover` at reader and worker boundaries |
| 12 | Low (prerequisite) | Safeguards | High | Untested queues; unasserted notes | Phase 0 tests |
| 7b, 7c | Low, conditional | Performance | Medium | Headroom shrinks with session length and input size | Only if the timing tests miss |
| 5 | Optional | Rendering | High | Misaligned rows with emoji sequences | Only if observed |
| 13 | Low or optional | Hygiene | High | Latent path literal; stale comments | Trivial fixes |

### C. Validated roadmap

Kept: the phase structure. Changed:
- phase 0 holds only passing tests;
- the streamed-output cap moves into phase 2;
- the overlay consolidation shrinks and becomes optional.

#### Phase 0: regression safeguards (no production change)
- **Objective:** pin current behaviour where later phases work.
- **Findings:** 12, 6 (fuzzing), 4 (the matrix), 7 (measurement).
- **Scope:**
  - `pipe.Queue`: Hold, Release with a drop, Close, Seq order;
  - the app's write queue: FIFO answer-then-interrupt, the full queue;
  - an overlay-pair matrix of draw order, key ownership and click ownership, recording today's behaviour;
  - a large-output timing test under the `timing` tag;
  - fuzz targets for the image decoders and `InspectData`, and for both mappers' `Out`, seeded from `testdata/probes`.
- **Expected:** a safety net for phases 1 to 3; the fuzzers may surface real panics.
- **Possible regressions:** none, since there is no production change.
- **Acceptance:** all pass on today's code; each fuzzer runs a fixed time (for example 60 s) without a crash, or its crash becomes a phase 2 item.
- **Commands:**
  - `go test ./internal/provider/pipe ./internal/app -run 'Queue|Overlay' -count=1`
  - `go test ./internal/input -run '^$' -fuzz FuzzInspectData -fuzztime 60s`, and likewise for the other targets
  - the timing step.

#### Phase 1: correctness and availability fixes (test first; each fix independent)
- **Findings:** 3a, 1a, 10, 3b, 1b, 11, the `openExplorer` path from 4, and the path literal and stale comments from 13.
- **Prerequisite tests:**
  - **For 3a:** today's reproduction, as a test: a pre-existing MERGE_HEAD with an edited conflict file. Expect a refusal, no abort, the file intact.
  - **For 1a:** the overlay probe, as a test: a refused optional read during the handshake keeps Handshaking, then reaches Ready.
  - **For 10:** an opened untracked folder gains a file, and the next snapshot lists it.
  - **For 3b:** Quit with a blocking fake commit asks first.
  - **For 1b:** `ErrNoPending` expires the approval and the session stays.
  - **For 11:** no "known payload" note across all recordings.
  - **For the `openExplorer` path from 4:** F3 with the question open, then a click acts normally.
- **Expected:** no loss of a merge resolution, no Codex startup failure on optional reads, an honest folder listing.
- **Possible regressions:**
  - handshake ordering, guarded by the probe replays and the handshake tests;
  - the Worktrees flow, guarded by the worktree tests.
- **Acceptance:**
  - each new test fails before its fix and passes after;
  - `go build ./...`, `go vet ./...` and `gofmt -l .` are clean;
  - `go test ./...` passes, with the golden frames unchanged;
  - `-race` passes on render, app, input, state and the providers;
  - the timing step meets its targets;
  - the probe replays are identical, except that the false notes are gone.

#### Phase 2: robustness hardening
- **Findings:** 2 (TIFF and ICO), 7a (the streaming cap), 9, 3c, 6 (`recover`), 8.
- **Prerequisite tests:**
  - crafted TIFF and ICO files under `GOMEMLIMIT` refuse with an error and stay bounded;
  - the phase 0 large-output test stays well under 16 ms per delta, and the completed output is unchanged;
  - a fake process that stops reading stdin causes no deadlock, and the handshake ordering test is unchanged;
  - a fake git whose child keeps stdout makes the operation end within its timeout plus `WaitDelay`;
  - a panicking fake session gives a visible error and an orderly end;
  - an approval card containing U+202E and U+200B shows visible markers, while an emoji ZWJ sequence still renders.
- **Expected:** no crash from crafted images; no UI freeze from large output; no hang on git children; a cleaner end after a panic.
- **Possible regressions:**
  - images that are valid but carry unusual tags; covered by the existing image tests and fixtures;
  - live output now shows only its tail;
  - Claude's handshake sequencing.
- **Acceptance:** as in phase 1, plus the fuzzers' bounded runs.

#### Phase 3, conditional: performance
- **Condition:** only if phase 0's timing tests miss a target with realistic large inputs.
- **Scope:** memoise per-tool derived fields; bound the per-frame scans (findings 7b and 7c).
- **Acceptance:** the timing step and the large-input tests within 16 ms at p95, and no golden changes.

#### Phase 4, optional: maintainability
- One shared overlay predicate in `render.Draw`, when the next overlay is added.
- An extraction of the shared adapter lifecycle, only on a third adapter or further drift.
- `t.Parallel` in the git tests, or a git package of its own.
- Normalised approval decisions.
- Cluster-aware width (finding 5).
- First-frame-only GIF decoding.

Each item goes in only with the phase 0 matrix and timing tests green, and zero golden changes unless intended.

### D. Architectural boundaries and regressions to prevent

**Preserve unchanged:**
- the five parts and `internal/arch`, with its planted-fault test;
- the pure reducer with returned commands;
- the Mapper/Session/State split;
- the `Provider` interface;
- the suspended start in a Job Object, and the close escalation;
- Hold/Release ordering and the unbounded event queue;
- the injected seams in `app.Options`;
- the generation counters;
- the draft store and its write order;
- the commit and push verification chain, and `GitStop`;
- the span/ref model, `layout()` and `WrapCache`;
- post-frame Sixel images with the final-frame check;
- the by-value `UI`;
- `internal/input` as the outside-world layer;
- the patched ultraviolet copy and its process.

**Behaviour future work must keep:**
- **Protocol:**
  - approval ids echoed exactly;
  - an unreadable request only ever denied;
  - answer-before-interrupt order (the write queue's FIFO);
  - the handshake's events held until its writes are made;
  - no rule-persisting decision ever sent;
  - permission and sandbox flags untouched;
  - secrets redacted from events;
  - the 36 probe replays unchanged unless a change is intended.
- **Git:**
  - never force, amend, no-verify, all-refs or tags flags;
  - push only to the configured upstream;
  - a commit's success only from HEAD advancing;
  - the staged-set and blob checks;
  - a stop only to git's own process group, with its pid held;
  - lock files never deleted;
  - no automatic pull, merge, rebase or reset.
- **Drafts:** temp file, sync, rename; never touch a held folder; never drop a damaged list.
- **Performance:** keystroke and frame p95 within 16 ms (the timing step); no I/O while drawing a frame.
- **Compatibility:**
  - Windows Terminal: the win32-input patches, Sixel detection, and graceful text fallback without Sixel;
  - both CLIs optional;
  - golden frames byte-identical unless a change is intended.

### E. Final recommendation

**Proceed with the roadmap.** Nothing found justifies a broad refactoring.

**Implement first, test-first:**
1. the merge pre-check (3a);
2. the non-fatal optional Codex reads (1a);
3. the untracked-folder refresh (10).

These are the only findings with data-integrity or availability effects shown by reproduction or by direct code paths in normal usage, and each fix is small and local.

**Then:**
- the rest of phase 1;
- phase 0's queue, matrix, timing and fuzz safeguards, which can be built alongside, since none is a prerequisite of these three fixes;
- phase 2.

**Phases 3 and 4 stay conditional or optional.**
