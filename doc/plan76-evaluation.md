# Plan 76 evaluation (2026-11-04)

The check of plan 76, Claude first: the review cycle's corrections without a
fixed limit, stopping on evidence; execution supervision; Codex paused and
the provider-parity record; and Claude Code's native Implementer and Reviewer
seen in zones A and C (brief, amendments 95 to 103). Steps 2 and 3 were
written by an Implementer, step 1 by the orchestrator, and each was judged by
a new Reviewer; the orchestrator ran step 4: the live run L76, the checks and
the final reviews.

**Written for the owner's decision. Plan 76 is not closed: nothing is
staged, committed or pushed, and HEAD is `56e09e6`.** The fifth final review
gave PASS on the 22 criteria it judged, 76.1 to 76.22. The first failed 76.3
and 76.22, the second 76.21 and the third 76.22; the fourth ended at its turn
limit with no report and counted for nothing. 76.23 is the owner's, from
L76's screenshots.

**Then approved and closed, 2026-11-04** (amendment 104). The owner approved
76.23 from L76's screenshots, kept the recording's personal text, and asked
for the documentation fixes in a commit of their own after the plan's. The
closure moved the plan's section here (The plan as it ran), and the plan's
files were committed, without a push.

## The result

```text
Evaluation - plan 76, Claude first - 23 criteria: 23 PASS, 0 FAIL
PASS  76.1   the Review cycle: no count of corrections, no same-finding stop; D1 in full in doc/evaluation.md; decisions in the notes
PASS  76.2   what a PASS means: the owner's five points in doc/evaluation.md's opening
PASS  76.3   Execution supervision held in doc/evaluation.md; a PASS here means specified and recorded, not proven in behaviour
PASS  76.4   CLAUDE.md: the tier rule counts introduced defects; Long jobs and agents
PASS  76.5   implementer.md holds D2's four points, its frontmatter unchanged; reviewer.md unchanged; TestReviewerDefinition passes
PASS  76.6   no three-correction limit, no same-finding stop; AGENTS.md within its ceiling; the suite passes
PASS  76.7   main.go's provider list holds Claude Code alone, with the comment; no existing Codex file changed; Codex's tests pass
PASS  76.8   the paused row in both modes, greyed, not choosable; the 12 frames differ on that row and the rows it moves
PASS  76.9   a paused CLI's history entries listed and greyed, starting nothing ("OpenAI Codex is paused"); Delete and Delete All as before
PASS  76.10  --provider codex chooses Claude Code and says why; a saved Codex choice gives the first CLI offered
PASS  76.11  Codex's recordings replay as before; no Codex frame changed
PASS  76.12  D6's sentences word for word in AGENTS.md and plan.md; architecture.md, How it starts
PASS  76.13  doc/provider-parity.md: purpose, update rule, paused status, checklist, three rows; the Codex item left Nice to have
PASS  76.14  m1 to m4 through go test -overlay, each caught, the code's hash unchanged
PASS  76.15  P76a on 2.1.303, sanitized, with its README entry; each agent's task_started and model; SendMessage offered
PASS  76.16  claude-native-agents.jsonl from P76a; the new frames' cards and tree; besides them only the 9 listing frames (amendment 101)
PASS  76.17  m9 to m11 each fail the new frames
PASS  76.18  L76: the paused row, both agents in zones A and C, the configuration unchanged, no CLI left, projects.json put back, the projects recycled
PASS  76.19  build, vet on Windows and Linux, gofmt and the suite; -race on the changed packages; the timing step alone, on mains power
PASS  76.20  only the plan's files differ; none holds a BOM, a CR or the Windows user name
PASS  76.21  the documents describe what the code and the cycle do
PASS  76.22  each rule of D1 and D3 classed specified, observed or not exercised, the decisions recorded; a PASS here means specified and recorded, not proven in behaviour
PASS  76.23  the paused row and the agents' cards and tree at full size, from L76's screenshots: approved by the owner as shown (amendment 104)
```

76.3 and 76.22 are about rules for how the plan was carried out. Their PASS
says the rules are written and their application recorded; behaviour shows
only in what is classed Observed below, and the classes Not exercised make no
claim.

## The checks

Go 1.27.2 on Windows, one heavy job at a time, each run's fingerprint the same
before and after it.

- **The full runs.** `go build ./...`, `go vet ./...` and
  `GOOS=linux go vet ./...` exit 0, `gofmt -l .` prints nothing, and
  `go test -count=1 ./...` exits 0, 16 packages ok, in each of three runs:
  - 10:37 to 10:42, fingerprint `a9ae5580…` (`internal/app` 273 s);
  - 11:32 to 11:37, `7822b3b3…`, after the first final review's corrections
    (265 s);
  - 12:13 to 12:18, `5383d3d8…`, after the second's (271 s). The third and
    later final reviews judged this tree;
  - 13:48 to 13:52, `ba44cc33…`, once this file and amendment 103 were written:
    the same results, the byte check covering 59 files.
- **The bytes**, in each full run: none of the 58 files git lists as the
  plan's starts with a BOM, holds a CR or the Windows user name, or lacks its
  final newline.
- **`-race`** on the changed packages, one at a time, each with
  `-timeout=20m` (10:42 to 10:57, fingerprint `a9ae5580…`, the WinLibs gcc
  16.2.0): `internal/state` 8.8 s, `internal/fixture` 2.2 s,
  `internal/provider/mock` 2.7 s, `internal/provider/claude` 29.0 s,
  `internal/render` 216.7 s, `internal/app` 514.8 s, all ok;
  `cmd/agent-command-center` has no test file.
- **The timing step**, alone (10:57 to 10:59, `a9ae5580…`), with the command of
  `doc/evaluation.md`, Keystroke timing: on mains power, the CPU at 42 % before
  and 26 % after; 26 of 26 pass, 111 s.
- **The Go files since.** The corrections after the first and second final
  reviews changed documents only: the 559 Go files' hashes were identical
  before and after them, so `-race`, the timing step and L76's build hold for
  the final tree.
- **The mutants**, all eleven, three times (11:00, 11:37 and 12:18), each
  through `go test -overlay` with `-timeout=120s`, the code's hash equal before
  and after: step 2's m1 to m8 and step 3's m9 to m11 all caught, none
  surviving or invalid.

| Mutant | What it does | Caught by |
|---|---|---|
| m1 | the paused row choosable | `TestPausedRow` |
| m2 | a paused CLI's history entry resumed | `TestPausedHistory` |
| m3 | `--provider codex` falling back without the note | `TestPausedNoFallback` |
| m4 | the paused row saying "not installed" | `TestPausedRow`, `TestPausedRowPicture`, `TestPausedNoFallback` |
| m5 | the old "no CLI" note while a CLI is paused | `TestPausedNoCLINote` |
| m6 | `--provider codex`'s note beside it | `TestPausedNoCLINote` |
| m7 | Codex's proxy lines while it is paused | `TestPausedProxyLines` |
| m8 | the proxy lines hidden with Codex's adapter | `TestCodexProxyOnStartup`, `TestCodexProxyLinesTogether`, `TestCodexProxyProblemKeepsTheKeys` |
| m9 | zone C's card without the name | `native-agents` and `native-agents-tree` at 120x40 and 200x60 |
| m10 | zone C's card without the model | the same four frames |
| m11 | zone A's tree without the agent | `native-agents-tree` at all four sizes (at 79x24, its rig's `t.Fatalf`) |

### The probe P76a

Claude Code 2.1.303 (2026-11-04, 02:35 local time, 18 s), `ANTHROPIC_MODEL=haiku`,
every `CLAUDE*` and `ANTHROPIC_*` variable dropped, in a new folder under
`%LOCALAPPDATA%\Temp` holding copies of `implementer.md` and `reviewer.md`
(hashes equal to the repository's). No Claude Code or Codex terminal session
was open before. `confcheck` compared PASS: the configuration identical,
`~/.claude.json` changed only in Claude Code's cache keys, 5 new session
files. Nothing was written in the work folder, and no process was left.

- Each agent's `system/task_started` gives its name (`subagent_type`
  `implementer`, then `reviewer`), and its own messages its model:
  `claude-sonnet-5-5` and `claude-opus-5-5`, as the definitions set.
- `system/init` offered SendMessage to the orchestrator, which did not use
  it: Haiku passed the clause on to the agents, which had none. So a continued
  agent stays unverified, for that reason rather than the one the plan's
  Known limits foresaw (a CLI offering no SendMessage).
- No event gives an agent's effort.
- The recording, sanitized (an organisation prefix that matches nothing), is in
  `testdata/probes` and `doc/research/probes`, identical; a scan by counts finds
  no user, host or domain name, home path, proxy host or e-mail address. It
  keeps, as P69a and P24u do, the "/usage" answer (its time zone, usage
  percentages and command names), the subscription type, and the owner's Px
  reminder from the global instructions, which the mock file and the new
  frames show too.

### The live run L76

The build of the final Go files (`acc.exe` `3ebd3e44…`, cgo off) in ACC's 9 pt
profile, maximised; Claude Code 2.1.303 with `ANTHROPIC_MODEL=haiku`, the
window's `CLAUDE*` and `ANTHROPIC_*` variables removed, Px through the
environment; a new project under `%LOCALAPPDATA%\Temp` with the two agent
copies; `confcheck` before and after; `projects.json` copied before and put
back after.

| L76 item | First attempt, 10:32 | Second run, 10:34 |
|---|---|---|
| 1 before: no CLI in a terminal, Px, the version, the copies, the snapshot | PASS | PASS |
| 2 the home screen, Codex's row paused | FAIL: the driver's check (below) | the shot: "OpenAI Codex paused", greyed |
| 3 zone C: both cards, their names and models | not reached | PASS: `implementer │ sonnet-5-5`, `reviewer │ opus-5-5`, 0 ACTIVE · 2 DONE |
| 4 zone A: the tree names both agents | not reached | PASS |
| 5 zone A focused: both agents' rows | not reached | PASS |
| 6 Ctrl+Q: the window closed | PASS | PASS |
| 7 after: the configuration unchanged, no CLI left, `projects.json` put back | PASS | PASS |

- The first attempt's L76.2 was the driver's fault: it read the row in the
  screen's text, but the home screen draws its text as images in picture mode,
  and UI Automation found only the path field. The shot showed the row as
  expected. It stopped before Enter, with no model call. The driver then
  judged the row from the shot, and the run went again under a new folder.
- While the implementer ran, zone C showed its card with "model N/A" until its
  first message gave its model. The live session is Turn 1: the dashboard
  hides its own "/usage", where the mock file replays it as a first turn.
- Its usage, with P76a's: two Haiku turns, two Sonnet and two Opus canaries,
  no continuation.
- The two projects and P76a's work folder, each holding the two agent copies
  alone, went to the Recycle Bin. Each move is proven by the path gone and a
  `$I` record naming it.
- The CLI's own records of these sessions stay under `~/.claude/projects`.

## The decisions' record

Each decision to continue or stop, with its evidence (D1; `doc/evaluation.md`,
The Reviewer, Convergence, The decisions' record). They were kept in the
orchestrator's notes during the plan. Step 1 ran under the rules it replaced,
so its decision follows the old count of corrections; from step 2 on, under
the new ones.

| # | When | Finding or event | Cause, by its evidence | Prediction | Decision | What came of it |
|---|---|---|---|---|---|---|
| 1 | Step 1, after its review | The review passed all six criteria, with observations on the new rules' text: the evidence template printed no time, the race command had no `-timeout`, the ambiguity stop did not say "code and evidence unchanged", jobs run by an agent had no deadline of their own, an agent with no record had no first deadline, the live run's row mixed its columns | The text as written (`doc/evaluation.md`, `AGENTS.md`, `CLAUDE.md`) | Each fix makes the text say what D1 and D3 say; the final review judges it | Continue to step 2 (a PASS); fix those six in `doc/evaluation.md` after the review, for the final review to judge | Done, but two of the fixes changed approved texts, D3's deadlines for an agent's jobs and D1's ambiguity stop, without the owner, and one put a defect in the evidence template (decisions 5 and 8). Left as written: 70 minutes against 70.2 recorded, a long run of corrections that each introduce a defect (a risk signal by design), alternating failures (left to the look), the tier raise from `xhigh` reaching `max` |
| 2 | Step 2, after its review | The review passed all eight criteria, with observations: with Claude Code missing and Codex paused, the home screen said "Install Claude Code or Codex CLI"; Codex's proxy line showed while Codex is paused; the Startup machine's row and the messages table did not mention the paused CLI; the parity record's plan 69 row lacked "design D1" | The code paths in `internal/render/home.go` (the note's fixed text, the proxy lines shown unconditionally) and the documents' text | A note built from the CLI list, and the proxy lines tied to the build's providers, make the new tests pass and the old texts stay byte for byte where nothing is paused | Continue to step 3 (a PASS); fix those four after the review, with a test and a mutant each, for the final review to judge | Done: `TestPausedNoCLINote`, `TestPausedProxyLines`; mutants m5 to m8 caught; the evidence `4cf0f3ca…`. Left: the resume note during detection with `--provider codex` (as before the plan, for a CLI not installed), Ctrl+N clearing the note and "(detecting…)" on the paused row (as specified), information only. The proxy lines' hiding goes beyond D5; the owner kept it (amendment 103) |
| 3 | Step 3, before its code | A file added to `testdata/mock` changes existing frames, which 76.16's "No existing frame changes" forbids | The Explorer and quick-open frames list `testdata/mock`; shown in a copy of the tree with a stand-in file of the same name: exactly 9 frames failed | Adding the mock file changes these 9 frames and no other, only by the listing | **Stop for the owner:** a fix would change a criterion | The owner kept the mock file (amendment 101). The 9 frames rewritten by the step are byte-identical to the stand-in's |
| 4 | Step 3, the Implementer's first evidence | Two tests failed on the new recording, `TestFixturesMatchRecordings` and `TestStreamEventsTakeTheMessageStartID`, through files outside the plan's, one under `internal/provider` | Every recording in `testdata/probes` needs its copy in `doc/research/probes` and a table entry; the stream-event total counts the new recording's 86 events, each mapped correctly (the per-line check passed) | The copy, the entry and the total turn the suite green, with no other change | **Stop for the owner:** a fix would change the scope | The owner allowed the three (amendment 102). The evidence `3251aa5e…`: the suite green, every exit line true |
| 5 | Step 3, the same report | The evidence's exit lines printed 0 after failing tests | `echo "$(t) test exit $?"`: the time's command substitution resets `$?`, shown by `false; echo "$(t) $?"` printing 0 and by saving `rc=$?` first printing 1. The baseline's template had no substitution: step 1's correction (decision 1) introduced the defect | With the status saved first, an exit line reports a failure; the failing run had shown "test exit 0" beside its FAIL lines | Fix the template in `doc/evaluation.md`. A defect a correction introduced raises the next Reviewer one tier, and the next review of step 1's text was the final review, at `xhigh`: from there the raise reaches `max`, which only the owner or the plan asks for, so none was applied then | The corrected evidence's exit lines are true. Every earlier exit line of this plan is unreliable; their `ok`, `FAIL` and `CAUGHT` lines stand |
| 6 | Step 3, after its review | The review passed all three criteria, with observations: the quick open's title counts one file more (17 files, not 16), not literally "only that row and the rows it shifts"; mutant m11 also stops the tree rig at 79x24; the recording keeps the "/usage" text and the subscription type, and the driver's comment said "no account… MCP names" | The listing's count follows from the new file; the comment described the filtered views only | With amendment 101's record and the driver's comment corrected, both say what the frames and the recording show; the final review judges it (this prediction added after the fifth final review, at its request) | Continue to step 4 (a PASS); correct the record of amendment 101 (it had said "scroll mark") and the driver's comment; note the rest here | Done; the owner's words in 76.16 unchanged; the owner kept the count in amendment 101's record (amendment 103) |
| 7 | Step 4, L76's first attempt | L76.2 failed: the driver found no "OpenAI Codex paused" in the screen's text | Picture mode draws the home screen's text as images: UI Automation read only the path field; the screenshot showed the row as expected | A shot judged by the orchestrator, then the rest of L76, passes | Run L76 again: a safe re-run of an approved run; the first attempt stopped before Enter, with no model call | The second run passed every item |
| 8 | Step 4, the final review | It failed 76.3 and 76.22: `doc/evaluation.md` did not hold D3's deadline for a long job "under an agent", and the classification called that rule observed; 76.22 also lacked the result block's sentence and a row for what a PASS means, and gave a wrong figure (239 s). Its observations named D1's ambiguity stop changed the same way, and the tier raise meeting `max` | Decision 1 had rewritten D3's and D1's approved texts in `doc/evaluation.md` without the owner; the record was drafted from the step 2 evidence's figure | With D3 and D1 amended to the texts as run, or the texts restored, 76.3 holds; with the classification and the figure corrected and the result block drafted, 76.22 does | **Stop for the owner:** a change of the approved design is the owner's | The owner kept both texts as run, asked for the next final review at `max`, and kept the two additions (amendment 103). `plan.md`'s D1 and D3 now say what `doc/evaluation.md` says; the classification, the figure and the parity record's tool list corrected; the result block drafted; a new final review at `max`, which passed 21 of the 22 criteria, 76.3 and 76.22 among them, and failed 76.21 (decision 9) |
| 9 | Step 4, the second final review | It failed 76.21: `doc/architecture.md` still said a problem in the `codex` setting "is named on its own line", where the paused build shows none. Its observations named the same omission in `doc/reliability.md` and `doc/provider-parity.md`; `doc/evaluation.md`'s After a CLI update still proposing a re-check for Codex; the README's P76a row saying who replied what wrongly; rows classed Observed with sub-rules that never occurred; long jobs the orchestrator ran in the foreground with no deadline; and P76a run while an Implementer worked | Decision 2's addition, Codex's proxy lines hidden while Codex is paused, reached `ui-design.md` alone; D6's sentence on the re-check reached `AGENTS.md` and `plan.md` but not `doc/evaluation.md`; the classification had been drafted from the rules rather than from each run | With `architecture.md`, `evaluation.md`, `provider-parity.md` and the README aligned with the code and the recording, and each row classed by what occurred, 76.21 holds and 76.22's rows are true | Correct the plan's documents; leave `doc/reliability.md:143`, outside the plan's files, to the owner; correct the classification; the full run again, then a third final review at `max`, the final review's raised tier | The third final review passed 21 of the 22 criteria, 76.21 among them, and failed 76.22 (decision 10) |
| 10 | Step 4, the third final review | It failed 76.22 alone: two rules of D3's "What the look decides", a probable stall and an agent repeating failures, had no class, and "Stop: a refuted prediction" was classed not exercised although decision 1's prediction had been refuted at the first final review | The same cause as decision 9's last point, in a new scenario: the classification listed the rules as recalled, not enumerated from D1's and D3's text, and decision 9 had acted on the rows the second review named | With one row for each rule of D1 and D3, in the plan's order and quoting its words, each classed by what occurred, 76.22 holds | Correct the classification, acting on the cause; the tree unchanged, so its evidence stands; a fourth final review at `max` | The fourth final review ended with no report (decision 11); the fifth passed 76.22 (decision 12) |
| 11 | Step 4, the fourth final review | It ended at its 100-turn limit with no report. The look at its 25-minute deadline, 16 s before the end, had found a legitimate run: 251 tool calls, the last record 15 s old, varied searches | The Reviewer's definition caps it at 100 turns (`maxTurns`), and this review spent them all on inputs larger by the classification and its coverage listing | A fresh review that batches its independent reads and reports before its turns run out ends with a report | A review that ended with no report counts for nothing: it runs once more, fresh (`AGENTS.md`, Verdicts stay out of the tree). Its brief says it has 100 turns, to batch independent reads in one turn and to report before they run out; a second one with no report goes to the owner | The fifth final review reported within its turns (83 requests): PASS on all 22 criteria (decision 12) |
| 12 | Step 4, the fifth final review | It passed all 22 criteria, judging the classification as drafted. Its observations asked for three changes: D1.5 rested on decision 2, a fix after a PASS whose tests were not run before the fix; the look printed each command and whether its result had come, not the result; decision 6 had no prediction | D1.5 had been classed from decision 2 without checking when its tests ran; the look's script printed no result text | With the three rows corrected as asked, the classification says what occurred | Continue to the records and the report, the plan's last step, correcting the three as asked and marking each as made after the review | Done: D1.5 not exercised; D3.16 and D3.20 state the look's gap; decision 6's prediction added |

## Where each rule of D1 and D3 stands

One row for each rule of D1 and D3, in the plan's order (`plan.md`, Plan 76,
Design), each in the plan's words, shortened. A PASS on 76.3 or on 76.22 means
the rules are specified and recorded, not proven in behaviour. Three rows,
D1.5, D3.16 and D3.20, and decision 6's prediction were corrected after the
fifth final review, at its request; each says so.

**Specified:** written, and checked by reading. **Observed:** applied during
plan 76, with its evidence. **Not exercised:** no occurrence, so no claim.

Each bullet of D1 and D3, numbered in the plan's order (`p76-cover.mjs`), and
its rows. A bullet and its sub-bullets often make one rule; a paragraph between
bullets can make another:

- D1: D1-02 → D1.1, D1.2; D1-03 to D1-05 → D1.3, D1.4; D1-06 to D1-08 → D1.5
  to D1.7; D1-09 → D1.8; D1-10 to D1-14 → D1.9; D1-15 → D1.10; D1-16 to
  D1-20 → D1.11 to D1.14; D1-21 → D1.15; D1-22 to D1-27 → D1.16.
- D3: D3-02 → D3.1; D3-03 → D3.2, D3.3; D3-04 → D3.4; D3-05 to D3-11 → D3.5,
  and the paragraph after them → D3.6; D3-12 → D3.7, D3.8; D3-13 → D3.9;
  D3-14 to D3-22 → D3.10 to D3.13, and the paragraph after them → D3.14;
  D3-23 to D3-25 → D3.15, D3.16; D3-26 → D3.17, D3.18; D3-27 → D3.19;
  D3-28 → D3.20; D3-29 → D3.21.

### D1, convergence

`AGENTS.md`, Review cycle 5 and Verdicts stay out of the tree;
`doc/evaluation.md`, its opening and The Reviewer, Convergence.

| # | Rule, in the plan's words | Class | Evidence |
|---|---|---|---|
| D1.1 | No fixed limit: "At most three corrections per step" and "the same finding twice" leave the Review cycle | Observed | Three corrections followed the final reviews (decisions 8, 9 and 10), counted nowhere; each rested on its evidence |
| D1.2 | Two things still stop the plan at once: a fix that would change a criterion, the scope or the architecture, and a question only the owner can settle | Observed | Decisions 3, 4 and 8, answered by the owner (amendments 101 to 103) |
| D1.3 | A finding is known by its cause, found by code-path analysis, by reproducing the failure and by discriminating evidence; a test's status alone never decides it: a passing test proves only its own scenario, and an earlier regression test that fails again proves that its scenario broke again | Observed | Decisions 3 to 5 and 7 to 10: each cause shown by a reproduction or a reading (the stand-in copy, the failing tests, the shell's `$?`, the screen's text, the texts against `plan.md`, the documents against the code, the classification against D1's and D3's text) |
| D1.4 | A different defect with the same symptom is a new finding; the same cause in another scenario is the same finding | Observed | Decision 9's 76.21 was a new finding; decision 10's is decision 9's cause in another scenario, so the same finding |
| D1.5 | Evidence for each correction, where the project's tools can reproduce the failure: a regression test reproducing it, written before the fix, shown failing on the unfixed code, then passing | Not exercised | No correction after a FAIL changed code, so none called for a regression test. Decision 2, a fix after a PASS, added `TestPausedNoCLINote` and `TestPausedProxyLines` with its fix; their mutants (m5 to m8) show them failing on the old behaviour, but they were not run before the fix. Classed after the fifth final review, at its request; it had been Observed |
| D1.6 | Where they cannot: other verifiable evidence, with one line saying why; for prose, the Reviewer's reading | Observed | Decisions 8 to 10, all prose: the texts compared with the plan and the code, then a new Reviewer's reading |
| D1.7 | The evidence of every earlier finding still holds | Observed | The last evidence ran the whole suite and all eleven mutants, decision 2's among them, on the final tree |
| D1.8 | A falsifiable explanation: it names the mechanism and predicts what the correction will make pass | Observed | Decisions 3 to 5 and 7 to 10 each predicted what would pass; 3, 4, 5, 7 and 8's held; 9's was refuted (decision 10); 10's held at the fifth final review |
| D1.9 | The same cause in a new scenario: the next correction acts on the cause, its evidence covering the cause's representative cases, chosen by risk (representative scenarios, boundaries, the known defects' regressions, a property or fuzz test where its value justifies its cost); whatever stays uncovered is a residual risk | Observed | Decision 10 acts on the cause: one row per rule, enumerated from the plan's text rather than from the rows named. Its cases: every bullet of D1 and D3, listed by a script against the rows (below), and the rows the earlier reviews named, kept as corrected (the known defects' regressions); no boundary, property or fuzz test applies to prose. Residual risk: how a rule is split into rows is a reading of its text |
| D1.10 | A defect a correction reintroduced or introduced is investigated, raises the next Reviewer one tier, and is a risk signal, not a stop | Observed | Decisions 5 and 8: three defects from decision 1's correction; decision 9: decision 2's addition not carried into `architecture.md`. The raise from `xhigh` meets the Exceptional tier: none at decision 5; at decision 8 the owner asked for `max` (amendment 103), and the final reviews stayed there. None was taken for a stop |
| D1.11 | Stop: a correction's prediction is refuted and nothing new explains why | Observed | Two predictions were refuted: decision 1's at the first final review (decision 8) and decision 9's at the third (decision 10). Each time something new explained why, the texts changed without the owner, then a correction acting on scenarios rather than on the cause, so the stop did not apply |
| D1.12 | Stop: no falsifiable explanation can be written | Not exercised | Each correction had one |
| D1.13 | Stop: two reviews give different verdicts on one criterion for code and evidence unchanged between them (amendment 103) | Observed | The second and third final reviews gave 76.22 different verdicts, but its evidence, the classification, had changed between them, so the stop did not apply |
| D1.14 | Stop: the two stops that end the plan at once | Observed | As D1.2 |
| D1.15 | The record: each decision to continue or stop, with the finding, its cause, the evidence, the prediction and what came of it, in the orchestrator's notes, then in the evaluation file | Observed | The record above; amendment 103, whose question names a verdict, held outside the tree until the last review |
| D1.16 | What a PASS means: the criteria satisfied on the evidence available; reliability rests on sufficient evidence, no unresolved known blocking defect, regressions detected and handled, justified convergence decisions, residual risks written down | Observed | Each review marked each criterion's basis, read or reported; regressions by the whole suite, `-race` and the eleven mutants; the convergence decisions in the record above; the residual risks in Known limits and in what the final review left open |

### D3, execution supervision

`doc/evaluation.md`, Execution supervision; `CLAUDE.md`, Long jobs and agents.

| # | Rule, in the plan's words | Class | Evidence |
|---|---|---|---|
| D3.1 | Its aim: bounded execution, evidence-based detection and autonomous recovery, with minimal overhead; a stall far less likely and shown sooner, not impossible | Specified | An aim, no claim |
| D3.2 | A limit on every long command, before it starts, its first run included; where none exists, the orchestrator sets one before launching it | Observed | P76a's driver cap (300 s), the stand-in golden run (`timeout 600`, `-timeout=10m`), each mutant's `-timeout=120s`, L76's six-minute cap, the full runs' Go default, `-race`'s `-timeout=20m`, the timing step's Go default |
| D3.3 | Observation deadlines are only supplementary, never the only protection against a blocked run | Observed | Every job and agent given a deadline also had a limit of its own |
| D3.4 | The limit ends the whole process tree: preferably one the process enforces itself, otherwise a wrapper that stops the whole tree | Specified | The runs were bounded by limits their processes enforce (Go's `-timeout`, P76a's driver cap with `taskkill /T /F`), and none fired. The Implementer's evidence command also had an outer `timeout` wrapper, which never fired and whose tree kill on Windows is unverified |
| D3.5 | The limits, each from its measured duration: a mutant's `-timeout=120s`, the suite's Go default, `-race`'s `-timeout=20m`, the timing step's Go default, a live run's driver limit, a new kind of job's from comparable evidence, written with its reason | Observed | Every run ended within its limit: the three full runs' longest package, `internal/app`, 273 s, 265 s and 271 s, against Go's 10 minutes; `-race`'s `internal/app` 515 s against 20 minutes; the timing step 111 s; L76 48 s against its 6 minutes; every mutant within 120 s |
| D3.6 | A hang becomes a failure, with Go's dump of every goroutine where Go enforces the limit | Not exercised | No run reached its limit |
| D3.7 | Tests that fail fast: a test holding fakes releases them in `t.Cleanup` and stops with `t.Fatalf` | Observed | Mutant m11 stopped the tree rig with `t.Fatalf` at once |
| D3.8 | A mutant whose run panics or times out counts as caught | Not exercised | Every mutant failed a test; none panicked or reached its limit |
| D3.9 | One heavy job at a time; an Implementer's evidence command counts as one, and the orchestrator starts none while an agent runs | Observed | With one exception: P76a, a live run of 18 s, ran at 02:35 while step 2's correction Implementer was working; that agent's evidence command started at 02:38, after P76a had ended, but the rule says the orchestrator starts none while an agent runs. Otherwise the stand-in golden run waited for the Implementer's evidence, and L76, the full runs, `-race`, the timing step and the mutant sets ran one after the other |
| D3.10 | Observation deadlines: one for each long job the orchestrator starts and one for each agent it starts; a long job that an agent runs comes under that agent's deadline (amendment 103) | Observed | Each Implementer (70, 32, 70 and 32 minutes) and each Reviewer; the orchestrator's background jobs (the full runs, `-race`, the timing step, the last mutant set); the Implementers' evidence runs under their agents' deadlines, bounded by their own limits. Two exceptions: the mutant set at 11:37 and L76's second run, run in the foreground with their own limits and the shell tool's timeout, had no deadline, although their first runs had given a record (73 s; 19 s) |
| D3.11 | Each deadline at the longest recorded comparable duration: the suite 6.5 minutes, `-race` 19.5, the timing step 3, a review at `xhigh` 21, an Implementer's first implementation 70, a correction 32 | Observed | Each deadline came from these. The first full run's 8 minutes were set above the recorded 6.5, which already included the build and vets; 70 minutes against 70.2 recorded is D3's own figure. `doc/evaluation.md`'s row for an agent with no comparable record applies this rule: step 3's Reviewer at `medium` and the first review at `max` took a review at `xhigh`'s 21 minutes; the later reviews at `max` took 25, from the first's 24.8 |
| D3.12 | A job made of measured parts: the sum of their durations | Observed | The first full run's 8 minutes, set as the suite's 6.5 plus the build and vets; the recorded 6.5 already included them |
| D3.13 | A new kind of job: no deadline on its first run, which its limit bounds; its first measured duration then becomes its record | Observed | P76a, the stand-in golden run, the first mutant set and L76's first attempt ran under their limits alone; the mutant set's record, 2 minutes, set the last set's deadline (12:18, ended in 100 s) |
| D3.14 | The deadline set with Claude Code's Monitor tool or one background timer; no polling and no watchdog process | Observed | One background timer per job and per agent; the look read the transcript twice, nothing polled |
| D3.15 | When a deadline passes, the look at a job: its process tree, each process's age and CPU time read twice a few seconds apart, its output's last line with when it was written | Not exercised | No job's deadline passed |
| D3.16 | The look at an agent: also its progress, its last commands and their results, read by a small script; the transcript never read whole | Observed | Twice. The second final review's deadline passed at 11:05:54Z, and `p76-look.mjs` read its transcript's tail twice, 20 s apart (11:06:17Z and 11:06:37Z); the fourth's passed at 12:18:18Z, read at 12:18:28Z and 12:18:49Z. With one gap: the script printed each command and whether its result had come, not the result itself, and each call twice, once with no time (stated after the fifth final review, at its request) |
| D3.17 | What the look decides, a legitimate run: a job is left to its limit | Not exercised | No job was looked at |
| D3.18 | A legitimate run of an agent: the next observation armed for the same interval | Observed | The second final review: 497 records, the last 4 s old, 19 in the last two minutes, varied reads of `AGENTS.md`, `internal/state` and `internal/app`; the next 21 minutes armed, and the review ended before them (11:09:39Z). The fourth: 565 records, the last 15 s old, 21 in the last two minutes, varied searches of the documents, the recording and the frames; it ended at its turn limit at 12:18:34Z, between the two reads, before a next deadline was armed |
| D3.19 | A job with no CPU time and no new line: a probable stall, recorded; nothing stopped before its limit; the limit ends the job with its diagnostics; with Go's dump the cause is fixed; where a wrapper ended it, the stalled test runs again alone with a short `-timeout` | Not exercised | No job was looked at, and none stalled |
| D3.20 | An agent repeating failures with no new evidence: stopped and resumed with what the look found; a loop the orchestrator cannot explain goes to the owner | Not exercised | Both looks found legitimate runs. As the look was run, it showed commands, not their results, so it could not have told an agent's failing runs from passing ones (stated after the fifth final review, at its request) |
| D3.21 | What it cannot guarantee: an agent looked at no later than one interval after a loop starts, the loop identified there only if the evidence shows the lack of progress, so detection not guaranteed in every case | Specified | A limit, no claim; also `plan.md`, Known limits |

### Two rules of D2 that `doc/evaluation.md`'s Execution supervision repeats

| # | Rule | Class | Evidence |
|---|---|---|---|
| D2.a | After a correction, only the mutants whose code or tests it touched run again; all of them run in the last evidence before the final review | Observed | After step 2's batch, m1 to m8 ran; the last evidence ran all eleven |
| D2.b | Each run prints a timestamped line per phase and per mutant | Observed | Every evidence file of steps 2 and 3, and the orchestrator's runs |

## The result block's lines for 76.3 and 76.22, as drafted

The final reviews judged these two lines as drafted, before the verdicts; the
result block above carries them with theirs. They hold the sentence 76.22
asks for, whatever their verdict:

```text
<verdict> 76.3   Execution supervision held in doc/evaluation.md; a PASS here means specified and recorded, not proven in behaviour
<verdict> 76.22  each rule of D1 and D3 classed specified, observed or not exercised, the decisions recorded; a PASS here means specified and recorded, not proven in behaviour
```

## The reviews

Each run is a new `reviewer` instance, never resumed, from the transcripts
under `~/.claude/projects`. Every one was offered exactly `Read`, `Grep`,
`Glob`, its report tool and `advisor`; none read a path outside the
repository and its inputs or opened `plan.md` or a plan's evaluation file; no
instruction file was in its context at the start; and the fingerprint was the
same after it as before.

| Review | Agent | Model, effort, requests | Tier and reason | Minutes | Verdict |
|---|---|---|---|---|---|
| Step 1 | `a36cfc3ac44c64512` | Opus 5.5, `xhigh`, 20 | Deep: the agent rules | 6.5 | PASS, 6 of 6 |
| Step 2 | `aae21f0acd4c2e98f` | Opus 5.5, `xhigh`, 56 | Deep: `AGENTS.md`'s hard constraints and the start path | 9.4 | PASS, 8 of 8 |
| Step 3 | `a0bd427db5a425c80` | Opus 5.5, `medium`, 19 | Light: recordings, tests and documents only | 2.3 | PASS, 3 of 3 |
| Final, 1 | `a34c1191b8e1f4236` | Opus 5.5, `xhigh`, 66 | the final review, steps 1 and 2 being Deep | 14.8 | FAIL: 76.3, 76.22 |
| Final, 2 | `a32ee636e37a3b97d` | Opus 5.5, `max`, 94 | raised one tier: a defect a correction introduced; `max` at the owner's request (amendment 103) | 24.8 | FAIL: 76.21 |
| Final, 3 | `ad0d926503fcccd3e` | Opus 5.5, `max`, 64 | the same | 24.3 | FAIL: 76.22 |
| Final, 4 | `affb28495c8e9b6f2` | Opus 5.5, `max`, 100 | the same | 25.3 | none: its turn limit, no report; counted for nothing |
| Final, 5 | `a94b39e7ea9b3d455` | Opus 5.5, `max`, 83 | the same | 21.3 | PASS, 22 of 22 |

- **Each request** ran on the model and at the effort passed: the counts are
  the transcripts' requests, each checked.
- **The Agent calls' descriptions** named the tier and its reason for steps 1
  and 2, as `doc/evaluation.md`, The Reviewer, The record, asks; from step 3
  on they did not, and this table gives them.
- **The fifth final review's brief** added one line to the others: it had at
  most 100 turns, should make its independent reads in one turn, and should
  report before its turns ran out, a criterion it could not finish being NOT
  VERIFIED. It reported after 83.
- **Earlier verdicts:** the final reviews saw the decisions' record, which
  holds the earlier reviews' verdicts, as 76.22's evidence, which `AGENTS.md`
  allows; each said so and judged the other criteria on the tree.

## What the final review left open

The fifth final review's observations, and what came of each; then the points
the earlier final reviews left that are the owner's.

| # | Observation | What came of it |
|---|---|---|
| O1 | `doc/reliability.md:143` says Codex's proxy is "shown on the startup screen", and `:351` that a missing CLI shows "not installed"; while Codex is paused, neither holds. That file is outside the plan's files | Fixed after the close, in a commit of its own (amendment 104) |
| O2 | 76.12 names `architecture.md`, for which D6 gives no sentence; its How it starts carries D6's clause | A reading, noted |
| O3 | 76.7's "Nothing under `testdata/probes` or `testdata/mock` changes" read as "no existing file changes"; step 3 adds two new files there, as the plan lists | A reading, noted |
| O4 | The quick open's count, 16 files to 17, in the 9 listing frames, beyond "that row and the rows it shifts" | Kept by the owner (amendment 103) |
| O5 | The paused row reads "(detecting…)" until detection returns | As specified (`ui-design.md`, the first question) |
| O6 | `doc/provider-parity.md` says deadlines are set "with Claude Code's Monitor tool"; plan 76 used background timers, which `doc/evaluation.md` allows | Fixed after the close, in a commit of its own (amendment 104) |
| O7 | D1.5 was classed Observed on decision 2, a fix after a PASS whose tests were not run before the fix | Reclassed Not exercised, as it asked |
| O8 | The look printed each command and whether its result had come, not the result | Stated in D3.16 and D3.20, as it asked. The look's script could print each result's status and first line in a later plan |
| O9 | Decision 6 had no prediction | Added, as it asked |
| O10 | The evaluation file and amendment 103 were not in the tree yet | Written at the close, as drafted, with O7 to O9's corrections |
| O11 | `doc/evaluation.md` says each phase prints its start and its end with the time; the template prints the fingerprint untimed and gofmt's end alone | Fixed after the close, in a commit of its own (amendment 104) |
| O12 | Ctrl+N clears "OpenAI Codex is paused" with the other messages | As specified |
| O13 | The recording and the mock file keep the owner's "/usage" report, the subscription type and command names; the sanitizer's rules cover names, not these | Kept by the owner, as P69a and P24u (amendment 104) |

**The owner's, from the earlier final reviews:**

- **The tier raise against the Exceptional tier.** `CLAUDE.md` raises the next
  Reviewer one tier for a defect a correction introduced; from `xhigh` that is
  `max`, which only the owner or the plan asks for. Plan 76 asked the owner
  (amendment 103). A rule settling it for later plans is the owner's.
- **A deadline for an agent with no comparable record.** `doc/evaluation.md`'s
  deadlines table has this row, added after step 1's review: the closest
  comparable agent's duration, written with its reason. It agrees with D3's
  "longest recorded comparable duration", but neither D3 nor amendment 103
  names it.
- **`doc/evaluation.md`'s "once per plan".** A known limit of The Reviewer,
  older than plan 76, says the orchestrator's full run happens once per plan;
  `AGENTS.md`, Review cycle 6, repeats it after the last correction, as plan
  76 did twice. Fixed with them (amendment 104).
- **Before the plan, and unchanged by it:** with `--provider codex` and a Codex
  session in the project's history, the home screen offers it during detection
  (Ctrl+R is ignored then); when Claude Code is installed but its detection
  fails, the note still asks to install it.

## Known limits, as they turned out

- The plan's Known limits stand, with one change: SendMessage was offered to
  the orchestrator in stream-json mode, but it passed the request on to the
  agents instead of using it, so a continued agent stays unverified for that
  reason.
- Execution supervision met two deadlines while an agent ran, both on a
  legitimate run; it never met a stall or a loop, so those branches make no
  claim (D3.19, D3.20).
- The process deviations stay as recorded: P76a ran while an Implementer
  worked (D3.9), and two second runs of new kinds of job had no deadline
  (D3.10).

## The plan as it ran

`plan.md`'s section of plan 76 as it stood when the owner approved the
closure on 2026-11-04, word for word, its headings one level down.

### Plan 76 - Claude first: convergence, supervision, Codex paused

From the owner's messages of 2026-11-03 and 2026-11-04 (brief amendments 95
to 100), after three assessments. Claude Code comes first, and Codex is
paused without being removed. A step's corrections stop on demonstrated
non-convergence, not at a fixed count. Long runs are bounded and supervised.
The native Implementer and Reviewer are verified in zones A and C. A short
record keeps the differences between the CLIs. Zones A, B and C, their
approved layouts and the provider code stay as they are.

- **Baseline:** `56e09e6`.
- **Files:**
  - step 1: `AGENTS.md` (Review cycle), `CLAUDE.md`, `doc/evaluation.md`,
    `.claude/agents/implementer.md`. Approving this plan is the owner's
    request that `AGENTS.md` requires before `.claude` changes;
  - step 2: `cmd/agent-command-center/main.go`; `internal/app`,
    `internal/render`, `internal/state` and their tests; `testdata/frames`;
    `doc/ui-design.md`, `doc/events-and-state.md`, `doc/architecture.md`;
    `AGENTS.md`; `plan.md`; `doc/provider-parity.md` (new);
  - step 3: `doc/research/probes/drivers/p76-claude-agents.mjs` (new),
    `doc/research/probes/README.md`,
    `testdata/probes/p76a-claude-native-agents.jsonl` (new) and its copy
    `doc/research/probes/p76a-claude-native-agents.jsonl` (new), one entry
    in `internal/provider/mock/files_test.go`,
    `testdata/mock/claude-native-agents.jsonl` (new),
    `internal/app/golden_test.go`, new frames in `testdata/frames` and the
    9 that 76.16 names (amendment 101), the recording's entry in
    `internal/fixture/fixture_test.go` and its total in
    `internal/provider/claude/msgid_test.go` (amendment 102);
  - at the close: `doc/brief.md`, `doc/plan76-evaluation.md` (new),
    `plan.md`, `AGENTS.md` (status);
  - nothing else under `internal/provider` (`msgid_test.go`'s total aside,
    amendment 102), `scripts` or `.claude` changes, `reviewer.md` included;
    `go.mod` and `go.sum` stay the same.
- **Tiers:**
  - step 1: Deep (the agent rules), reviewed at `xhigh`;
  - step 2: Deep (`AGENTS.md`'s hard constraints and the start path), at
    `xhigh`;
  - step 3: Light (recordings, tests and documents only), at `medium`;
  - the final review: `xhigh`.
- **Who:**
  - step 1: the orchestrator writes it and runs its evidence command;
  - step 2: the orchestrator first writes D6's and D7's texts, then the
    Implementer, on Sonnet at `high`, does the rest;
  - step 3: the orchestrator records the probe; the Implementer, on Sonnet at
    `high`, builds the mock file and the frames;
  - the live run: the orchestrator.
- **The rules in force:** step 1 is reviewed under the current rules. Once
  its review has passed, steps 2 and 3 and the final review run under the new
  ones.

#### Design

- **D1. Convergence.** Changes `AGENTS.md`, Review cycle, step 5 and "Verdicts
  stay out of the tree"; `doc/evaluation.md` gains The Reviewer, Convergence.
  - **No fixed limit.** "At most three corrections per step" and "the same
    finding twice" leave the Review cycle. Two things still stop the plan at
    once: a fix that would change a criterion, the scope or the architecture,
    and a question only the owner can settle.
  - **A finding is known by its cause.** The cause is the code path, or the
    interaction of components, that produces the failure. It is found by
    code-path analysis, by reproducing the failure and by discriminating
    evidence. A test's status alone never decides it:
    - a passing test proves only its own scenario;
    - an earlier regression test that fails again proves that its scenario
      broke again.

    A different defect with the same symptom is a new finding; the same cause
    in another scenario is the same finding.
  - **Evidence for each correction.** Each finding gets discriminating
    evidence, meaning evidence that could have failed:
    - where the project's tools can reproduce the failure (golden frames,
      the Windows Terminal harness, synctest and fakes, the tests that read
      documents): a regression test that reproduces the Reviewer's stated
      failure. It is written before the fix, shown failing on the unfixed
      code, then passing;
    - where they cannot: other verifiable evidence, with one line saying why.
      That is a before-and-after pair at real size, a live run's output or a
      trace; for prose, the Reviewer's reading.

    The evidence of every earlier finding still holds.
  - **A falsifiable explanation.** A correction goes ahead on an explanation
    that names the mechanism (the code paths and how they interact). It also
    predicts what the correction will make pass, the earlier findings'
    evidence included.
  - **The same cause in a new scenario.** The next correction acts on the
    cause, not on the scenario. Its evidence covers the cause's
    representative cases, chosen by risk:
    - representative scenarios;
    - boundaries;
    - the known defects' regressions;
    - a property or fuzz test where its value justifies its cost.

    Whatever stays uncovered is written down as a residual risk.
  - **A defect a correction reintroduced or introduced** is investigated:
    what the earlier evidence missed, and how the correction broke it. It
    raises the next Reviewer one tier (`CLAUDE.md`). One such defect, or a
    short run of them, is a risk signal, not a stop.
  - **The stops for the owner:**
    - a correction's prediction is refuted and nothing new explains why.
      Oscillation, incompatible fixes and an unresolved cause all come to
      this;
    - no falsifiable explanation can be written;
    - two reviews give different verdicts on one criterion for code and
      evidence unchanged between them: the criterion can be read two ways
      (amendment 103);
    - the two stops that end the plan at once, above.
  - **The record.** Each decision to continue or stop is kept with its
    evidence: the finding, its cause, the evidence, the prediction and what
    came of it. It stays in the orchestrator's notes during the plan, as the
    count of corrections does today, and goes into the plan's evaluation
    file at its close.
  - **What a PASS means.** The criteria were satisfied on the evidence
    available; that does not exclude every defect. Reliability rests on
    five things:
    - sufficient evidence for the criteria;
    - no unresolved known blocking defect;
    - regressions detected and handled;
    - justified convergence decisions;
    - residual risks written down.

    This goes in `doc/evaluation.md`'s opening (amendment 97).
- **D2. The Implementer** (`.claude/agents/implementer.md`; its frontmatter
  unchanged):
  - "after three attempts without progress on the same failure, stop and
    report it" becomes: stop and report when an attempt leaves the failure as
    it was and brings no new evidence about its cause;
  - **corrections:** D1's evidence for each finding, and a cause found by its
    code path, never by a test's status alone;
  - **bounded runs:** every long command it starts has its D3 limit before
    it starts, and one heavy job runs at a time.
    The evidence and mutant runs print a timestamped line per phase and per
    mutant. After a correction, only the mutants whose code or tests it
    touched run again; all of them run in the last evidence before the final
    review;
  - **no history of the reviews in the tree until the final review is
    complete:** until then, no comment, document or test name records this
    plan's reviews, findings, corrections or verdicts. Afterwards, that
    history belongs in `doc/plan76-evaluation.md`, as D1's record requires.
- **D3. Execution supervision**, a new section of `doc/evaluation.md`.
  - **Its aim:** bounded execution, evidence-based detection and autonomous
    recovery, with minimal overhead. These safeguards make a stall like plan
    75's far less likely and show it sooner; they cannot make one impossible.
  - **A limit on every long command, before it starts,** its first run
    included; where none exists, the orchestrator sets one before launching
    it. Observation deadlines, below, are only supplementary safeguards,
    never the only protection against a blocked run.
    - **The limit ends the whole process tree.** It is preferably one the
      process enforces itself, Go's `-timeout` or a driver's own cap.
      Otherwise a wrapper stops the whole tree (`taskkill /T /F`), never one
      that ends only the top process.
    - **The limits,** each from its measured duration (plan 75's
      evaluation):
      - a mutant's run: `-timeout=120s`; all of plan 75's 51 finished
        within it;
      - the whole suite: Go's default of 10 minutes per package;
        internal/app's longest run took 346 s;
      - `-race`: `-timeout=20m` per package; internal/app's longest took
        572 s;
      - the timing step: Go's default; its longest run took 114 s;
      - a live run: its driver's own limit (L75's: 6 minutes per CLI);
      - a new kind of job: a limit from comparable evidence or from its
        known parts, written down with its reason, then refined from its
        measured durations.

    A hang becomes a failure, with Go's dump of every goroutine where Go
    enforces the limit.
  - **Tests that fail fast.** A test holding fakes releases them in
    `t.Cleanup` and stops with `t.Fatalf`, so a broken mutant fails in
    seconds. A mutant whose run panics or times out counts as caught.
  - **One heavy job at a time:** a full suite, `-race`, the timing step, a
    mutant set or a live run. An Implementer's evidence command counts as
    one, and the orchestrator starts none while an agent's runs.
  - **Observation deadlines, supplementary.** The orchestrator sets one for
    each long job it starts and for each agent it starts; a long job that an
    agent runs comes under that agent's deadline (amendment 103). Each is set
    at the longest recorded comparable duration:
    - the full suite: 6.5 minutes;
    - `-race`: 19.5 minutes;
    - the timing step: 3 minutes;
    - a review at `xhigh`: 21 minutes;
    - an Implementer's first implementation of a step: 70 minutes;
    - an Implementer's correction: 32 minutes. These two come from plan
      75's agent transcripts. Its one stalled correction, 78 minutes, is
      left out as the incident it was;
    - a job made of measured parts: the sum of their durations;
    - a new kind of job: none on its first run, which its limit bounds. Its
      first measured duration then becomes its record.

    The deadline is set with Claude Code's Monitor tool, or with one
    background timer. There is no polling and no watchdog process.
  - **When a deadline passes, the orchestrator looks once:**
    - **for a job:** its process tree, each process's age and CPU time read
      twice a few seconds apart, and its output's last line with when it was
      written;
    - **for an agent:** also its progress, meaning its last commands and
      their results. A small script reads them from its transcript; the
      transcript is never read whole.

    What the look decides:
    - **CPU time growing, new lines, or varied work:** a legitimate run. A
      job is left to its limit; for an agent, the next observation is armed
      for the same interval.
    - **A job with no CPU time and no new line:** a probable stall, recorded.
      Nothing is stopped before its limit. The limit ends the job with its
      diagnostics, and that end wakes the orchestrator. With Go's dump in
      hand, the cause is fixed; where a wrapper ended the job, the stalled
      test is first run again alone with a short `-timeout` to get the dump.
    - **An agent repeating failures with no new evidence:** it is stopped and
      resumed with what the look found. A loop the orchestrator cannot
      explain goes to the owner (D1: no falsifiable explanation).
  - **What it cannot guarantee.** An agent is looked at no later than one
    interval after a loop starts. The loop is identified there if the
    evidence the look reads shows the lack of progress. A loop whose
    repetitions don't show in recent commands and results can pass as
    productive work, so detection cannot be guaranteed in every case.
- **D4. `CLAUDE.md`:**
  - "A second FAIL on one criterion, or a reasoning question a Reviewer could
    not close, raises the next Reviewer one tier, once per step" also counts
    a defect a correction reintroduced or introduced;
  - a new item, **Long jobs and agents:** the orchestrator sets one
    observation deadline per long job and per agent it starts, and looks
    once when it passes. After a productive look at an agent, it arms the
    next deadline (D3).
- **D5. Codex paused.** Changes `cmd/agent-command-center/main.go`, `internal/app`,
  `internal/render`.
  - **The adapter:** `main.go`'s provider list leaves Codex's adapter out,
    with a comment citing amendment 96. Codex's packages, tests, probes and
    mock files do not change, and they keep running in `go test ./...`.
    Reactivating Codex means putting the line back.
  - **The home screen:** a supported CLI whose adapter the build leaves out
    keeps its row in both modes, in its place, greyed and not choosable, with
    "paused" where its version stands, instead of the "not installed:
    adapter not built yet" that a missing adapter gets today. The approved
    layout does not move.
  - **The session history:** a paused CLI's entries stay listed, greyed.
    Enter or a double click on one starts nothing and says "OpenAI Codex is
    paused". Delete and Delete All work as before.
  - **No silent fallback** (`AGENTS.md`, Keep failures visible):
    - with `--provider codex`, Claude Code is chosen, and the CLI question's
      note says "OpenAI Codex is paused";
    - a saved Codex choice gives the first CLI offered, with the paused row
      in view.
  - **Mock replays:** those of Codex's recordings run as before; the mock
    provider needs no adapter.
- **D6. The texts of Claude first**, which the orchestrator writes:
  - `AGENTS.md`, What this project is, after "the only two supported
    (amendment 5; the provider interface stays open to others later).":
    "Codex is paused since plan 76 (amendment 96): its adapter stays, built
    and tested, but the dashboard does not offer it
    (`doc\provider-parity.md`)."
  - `AGENTS.md`, Hard constraints, the first sentence of "Every agent CLI is
    optional" becomes: "The dashboard depends on neither CLI and shows one it
    cannot use as not installed, or as paused: Codex, while it is paused
    (amendment 96)."
  - `AGENTS.md`, Working rules, after "build one agent CLI end to end before
    the others.": "Claude Code first (amendment 96): a new capability is
    designed, built and validated for it, and a meaningful functional
    difference it makes between the CLIs goes in `doc\provider-parity.md`."
  - `AGENTS.md`, Working rules, the [Recurring] check, after "which runs only
    once the owner approves it": "; while Codex is paused, its version is
    reported and no re-check proposed".
  - `plan.md`, The owner's standing decisions, a new item: "**Claude Code
    first, Codex paused** (amendment 96, plan 76): new capabilities are
    designed, built and validated for Claude Code; Codex's adapter stays,
    built and tested, and the dashboard does not offer it; each meaningful
    functional difference goes in `doc/provider-parity.md`, which holds the
    reactivation checklist."
  - `plan.md`, Recurring, the CLI update item gains: "While Codex is paused,
    its version is reported and no re-check proposed (amendment 96)."
  - `plan.md`, Nice to have: the item "Codex's earlier turns drawn after a
    resume" leaves the list and becomes a row of `doc/provider-parity.md`.
  - `AGENTS.md` stays under its ceiling of 28,672 bytes; it holds 26,344
    today.
- **D7. `doc/provider-parity.md`** (new), written by the orchestrator:
  - **Its purpose:** what differs between the CLIs, feature by feature, so
    that a CLI paused or added later can reach the same function, not
    necessarily the same implementation.
  - **Its update rule:** only when a plan introduces a meaningful functional
    difference, as one of that plan's files, with no audit of earlier plans.
    The specifications' own "Claude Code | Codex" tables stay the reference
    for detailed behaviour.
  - **"Codex: paused":** what the pause does (D5), and the reactivation
    checklist:
    1. put the adapter back in `main.go`;
    2. run the whole suite;
    3. compare `codex --version` with the last probed version, and if it is
       newer run After a CLI update for Codex;
    4. one live run on Codex, as L75;
    5. go through the rows and decide, in a plan, which to close.
  - **The table:** Feature, Claude Code, Codex, What parity needs, Where. It
    starts with three rows:
    - the read-only Reviewer of ACC's development (plan 74; amendment 88);
    - convergence and execution supervision (this plan);
    - earlier turns after a resume from the home screen, the Nice to have
      item moved here (plan 69, design D1).
- **D8. The native agents in zones A and C.** Code reading says ACC already
  shows them:
  - the name comes from `subagent_type`;
  - each agent's model comes from its own messages;
  - zone C draws an effort column only when some agent has an effort, and
    Claude Code reports none for its subagents.

  What is missing is evidence, so the probe P76a records it, and the
  recording becomes a mock file. Golden frames then hold zone C's cards and
  zone A's tree, with the layouts unchanged. The live run L76 shows them at
  full size. A gap in either goes to the owner with a proposed fix; this plan
  does not change what zones A and C draw.

#### Steps

1. **The rules** (D1 to D4; 76.1 to 76.6).
2. **Claude first** (D5 to D7; 76.7 to 76.14; that step's mutants).
3. **The native agents** (D8; 76.15 to 76.17): the probe, then the mock file
   and the frames.
4. **Verification:** L76, of which the owner is told first, and 76.18 to
   76.22. Then the final review on the diff from `56e09e6`.
5. **Records and report, then stop.** No staging, commit or push without
   the owner. After commit and push, the plan's temporary files go to the
   Recycle Bin.

Work stops at once on anything unexpected.

#### Probe P76a and live run L76

Their approval comes with the plan's.

- **Setup, both:**
  - Claude Code 2.1.303, the last probed version;
  - a new temporary project under `%LOCALAPPDATA%\Temp`, holding copies of
    this repository's `.claude/agents/implementer.md` (as step 1 leaves it)
    and `reviewer.md`. That project's `.claude` folder is made for the run and
    goes to the Recycle Bin afterwards; this approval covers it;
  - the window's `CLAUDE*` and `ANTHROPIC_*` variables removed, and Haiku as
    the orchestrator's model;
  - `confcheck snapshot` once the copies are made, and `compare -cli-ran`
    after;
  - the owner's own Claude Code terminals closed first.
- **P76a:** a driver over stream-json, as `p69-claude-resume.mjs`, in one
  turn: "Use the Agent tool to run the implementer agent, then the reviewer
  agent, each with this prompt: Use no tool. Reply READY. If you have a
  SendMessage tool, then send the implementer agent: Reply READY again.
  Finally reply DONE." The recording is kept raw, then sanitized with
  `sanitize.mjs`.
- **L76,** in ACC's 9 pt profile, maximised, on the final build:
  - the home screen, with Codex's row paused;
  - a session on the temporary project, and the same turn;
  - zones A and C at full size, while the agents show and after;
  - Ctrl+Q;
  - `projects.json` copied before and put back after.
- **Usage:** two Haiku turns, two Sonnet and two Opus canaries of a few
  seconds, and at most two continuations.
- **Stops at:**
  - a screen not expected;
  - a gap in what zones A or C draw;
  - a configuration difference.

#### Criteria

- **76.1 The Review cycle.**
  - `AGENTS.md`'s step 5 holds neither a number of corrections nor the
    same-finding stop. It states the rule in short and points to
    `doc/evaluation.md`, The Reviewer, Convergence.
  - That part holds D1 in full: the cause, the evidence and when other
    evidence is accepted, the falsifiable explanation, the same cause in a
    new scenario, reintroduced defects, the stops and the record.
  - "A fix that would change a criterion, the scope or the architecture, or
    a question only the owner can settle" still stops the plan at once.
  - "Verdicts stay out of the tree" keeps convergence decisions, not a
    count, in the orchestrator's notes.
- **76.2 What a PASS means.** `doc/evaluation.md` states it with the owner's
  five points, as D1 lists them.
- **76.3 Execution supervision.** `doc/evaluation.md` holds D3:
  - a limit on every long command before it starts, its first run included;
    the limit ends the whole process tree, and a new kind of job gets its
    limit from comparable evidence;
  - each limit and each recorded duration, with its source;
  - observation deadlines as supplementary safeguards, for long jobs and for
    every agent the orchestrator starts;
  - the tests that fail fast, and mutants caught by a panic or a timeout;
  - one heavy job at a time;
  - the single look:
    - the process tree read twice, and an agent's last commands and results;
    - nothing stopped before a limit;
    - the next observation armed after a productive look at an agent;
  - what the safeguards can and cannot guarantee, the limit of detection
    included.
- **76.4 `CLAUDE.md`.** The tier rule counts reintroduced and newly
  introduced defects. "Long jobs and agents" states:
  - the observation deadline per long job and per agent;
  - the single look;
  - the next deadline armed after a productive look at an agent.
- **76.5 The Implementer.** `implementer.md` holds D2's four points, and its
  frontmatter is unchanged: name, description, model `sonnet`, effort
  `high`, tools and color. `reviewer.md` is unchanged, and
  `TestReviewerDefinition` passes.
- **76.6 Nothing contradicts.**
  - A search of `AGENTS.md`, `CLAUDE.md`, `doc/evaluation.md` and
    `.claude/agents` finds no three-correction limit and no same-finding
    stop.
  - `doc/evaluation.md`'s "within its limits" (The loop) says convergence.
  - `AGENTS.md` stays within its ceiling (`TestAgentsMDWithinCodexLimit`),
    and the suite passes.
- **76.7 The pause.**
  - `main.go`'s provider list holds Claude Code's adapter alone, with the
    comment.
  - Nothing under `internal/provider/codex`, `testdata/probes` or
    `testdata/mock` changes, Codex's tests included.
  - `go test ./internal/provider/codex` passes.
- **76.8 The paused row.**
  - A supported CLI without an adapter keeps its row in both modes,
    greyed, not choosable, "paused" in its version's place.
  - The frames that showed "not installed: adapter not built yet" show
    "paused" and differ on that row only. No other frame changes.
  - `ui-design.md` and `events-and-state.md` say so.
- **76.9 The history.**
  - A paused CLI's entries stay listed, greyed.
  - Enter and a double click on one start nothing and say "OpenAI Codex is
    paused".
  - Delete and Delete All work as before.
  - Each case has its test.
- **76.10 No silent fallback.**
  - `--provider codex` chooses Claude Code, and the note says "OpenAI Codex
    is paused".
  - A saved Codex choice gives the first CLI offered.
  - Each case has its test.
- **76.11 Codex's recordings.** The mock replays of Codex's recordings run as
  before, and their golden frames are unchanged.
- **76.12 The texts.** `AGENTS.md`, `plan.md` and `doc/architecture.md`
  (How it starts) hold D6's sentences word for word.
- **76.13 `doc/provider-parity.md`.** It holds D7's purpose, update rule,
  paused status, reactivation checklist and three rows, each pointing to
  where its feature is specified. The Codex item has left `plan.md`'s Nice to
  have.
- **76.14 Broken on purpose.**
  - Run through `go test -overlay`, with the code hashed before and after.
  - The mutants:
    - the paused row choosable;
    - a paused CLI's entry resumed;
    - `--provider codex` falling back without the note;
    - the paused row saying "not installed".
  - A test fails for each of them.
- **76.15 The probe.**
  - P76a is recorded with Claude Code 2.1.303, sanitized, and has its
    README entry.
  - It shows each agent's `task_started` with its name, and its messages
    with its own model.
  - It records whether the CLI offered SendMessage.
- **76.16 The mock file and the frames.**
  - `claude-native-agents.jsonl` comes from P76a through `TestMockFiles`,
    reviewed as a diff.
  - The new frames, at the sizes the existing zone C frames use, show:
    - zone C's two cards, with their names and models and no effort column;
    - zone A's tree, with both agents.
  - No existing frame changes, except the 9 frames that list
    `testdata/mock` (`explorer`, `explorer-nerd` and `file-finder`, at
    80x24, 120x40 and 200x60), which gain the new file's row: each differs
    only by that row and the rows it shifts (amendment 101).
- **76.17 Broken on purpose, the agents.** Three mutants each make the new
  frames fail:
  - zone C's card without the name;
  - zone C's card without the model;
  - zone A's tree without the agent.
- **76.18 The live run.** Every item of L76 passes:
  - the paused row;
  - zone A's tree shows both agents by name, and zone C's cards show their
    names and models;
  - an unchanged configuration;
  - no CLI left;
  - `projects.json` put back;
  - the temporary project removed.
- **76.19 The checks.**
  - Build, vet (Windows and Linux) and the whole suite exit 0, and
    `gofmt -l .` prints nothing, after the last code edit.
  - `-race` passes on the changed packages.
  - The keystroke timing step passes alone, on mains power, within its
    budgets.
- **76.20 The scope.**
  - Against `56e09e6`, only the plan's files differ.
  - None holds a BOM, a CR or the Windows user name.
- **76.21 The documents.** `ui-design.md`, `events-and-state.md`,
  `architecture.md`, `evaluation.md`, `provider-parity.md`, `AGENTS.md`,
  `CLAUDE.md` and `plan.md` describe what the code and the cycle do.
- **76.22 Specified, observed, not exercised.**
  - From step 2 on, each decision to continue or stop is recorded with its
    evidence, as D1 says.
  - The evaluation file places each rule of D1 and D3 in one of three
    classes:
    - specified: where it is written, checked by reading;
    - observed: applied during plan 76, with its evidence;
    - not exercised: no occurrence, so no claim.
  - A PASS on 76.3 or on this criterion means specified and recorded, not
    proven in behaviour, and the result block says so.
- **76.23 The owner's judgement.** The paused row and the agents' cards and
  tree at full size, from L76's screenshots: the owner's to judge, not
  verified by the orchestrator.

#### Known limits

- Execution supervision makes stalls like plan 75's less likely and shows
  them sooner; it cannot make them impossible.
- Detection is not guaranteed. An agent is looked at no later than one
  interval after a loop starts, and the loop is identified there if the
  evidence the look reads shows the lack of progress. A loop whose
  repetitions don't show in recent commands and results can pass as
  productive work.
- A new kind of job's first limit comes from comparable evidence, not from
  a measurement. If it is too tight, the run fails visibly and the limit is
  corrected.
- The Implementer's observation deadlines come from plan 75's tasks, so a
  much smaller step is looked at later than its own duration would
  warrant.
- Convergence decisions rest on judgments, recorded with their evidence:
  whether two findings share a cause, and whether evidence discriminates.
  They are checkable, not infallible.
- A changed agent definition can reach new agents late (plan 74). After
  step 1, the transcripts show which definition each Implementer got.
- Step 1 is reviewed under the rules it replaces, as plan 74's rules were.
- While Codex is paused:
  - a machine with Codex alone has no usable CLI;
  - Codex's sessions in the history cannot be resumed;
  - Codex is not re-probed after its updates, so its reactivation starts
    with After a CLI update;
  - ACC keeps 20 sessions per project, of any CLI, and the 10 most recent
    projects that are not favourites. As Claude Code sessions are recorded,
    older Codex entries can drop out of a project's history, and a project
    that leaves the recent list takes its history with it. Favourites are
    always kept, and Codex's own session store is unaffected.
- Claude Code reports no effort for its subagents, so zone C shows none for
  them.
- If the CLI offers no SendMessage in stream-json mode, a continued agent
  stays unverified.

#### Progress

- Step 1: done.
- Step 2: done.
- Step 3: done.
- Step 4: done.
- Step 5: the records written (`doc/plan76-evaluation.md`, amendment 103);
  waiting for the owner.

## The reports, word for word

From each Reviewer's own hand-back in its transcript; the Windows user name is replaced by `<user>`.

### Step 1, its review

Agent `a36cfc3ac44c64512`, effort passed xhigh; 2026-11-04T00:43:39.062Z to 2026-11-04T00:50:07.391Z, 6.5 minutes.

````text
Review - plan 76, step 1 - 6 criteria: 6 PASS, 0 FAIL, 0 NOT VERIFIED
PASS 1  76.1 The Review cycle [read]: AGENTS.md:90-102. Step 5 gives no number of corrections and no same-finding stop. It says "The corrections have no fixed limit", rests each correction on evidence and a falsifiable explanation, and points to "(`doc\evaluation.md`, The Reviewer, Convergence)" at :99-100. "A fix that would change a criterion, the scope or the architecture, or a question only the owner can settle, stops the plan at once" is kept at :100-102 and again at evaluation.md:556-557. "Verdicts stay out of the tree" (AGENTS.md:113-114) now keeps "its convergence decisions, with their evidence, in its memory notes", with no count. doc/evaluation.md:508-561 holds D1 in full: no fixed limit (:509-510), cause (:512-521), evidence and when other evidence is accepted (:522-533), falsifiable explanation (:534-536), same cause in a new scenario (:537-545), reintroduced or introduced defects (:546-549), the stops including the two that end the plan at once (:550-557), the record (:558-561). Apart from small connecting words, the text matches D1 word for word. The record drops "as the count of corrections does today", which no longer applies.
PASS 2  76.2 What a PASS means [read]: doc/evaluation.md:11-18, in the opening before "## The loop". It says a PASS means the criteria were satisfied on the evidence available, and lists the five points as D1 does (sufficient evidence, no unresolved known blocking defect, regressions detected and handled, justified convergence decisions, residual risks written down), citing amendment 97.
PASS 3  76.3 Execution supervision [read]: doc/evaluation.md:609-695, a new section.
  - Limit before start, first run included, at :616-619. It ends the whole process tree at :621-625. A new kind of job takes its limit from comparable evidence at :637.
  - Each limit and recorded duration with its source, in the tables at :626-637 and :658-667. I checked these figures against the raw logs and transcript durations in orchestrator.txt and they match: 346 s (345.991 s, :113); race 549 s and 572 s (:141-142); timing 114 s (113.821 s) and 2:38 (21:48:10 to 21:50:48); suite 4:58 to 6:25 (22:23:10-22:28:08 and 21:22:25-21:28:50); race 18:34 and 19:19; Implementer 46.7 and 70.2 min (14:05:30 to the hand-back at 15:15:40); corrections 15.6 to 31.5 min; the stalled correction 77.5 min (16:23:36 to 17:41:04).
  - Observation deadlines are supplementary, for long jobs and every agent the orchestrator starts, at :652-656.
  - Tests that fail fast, and mutants caught by a panic or timeout, at :641-644.
  - One heavy job at a time at :645-647.
  - The single look at :669-689: the process tree read twice, an agent's last commands and results, nothing stopped before its limit, the next observation armed after a productive look.
  - Guarantees and the limit of detection at :613-614 and :691-695.
PASS 4  76.4 CLAUDE.md [read]: CLAUDE.md:53-55. The tier rule now also counts "a defect a correction reintroduced or introduced". "Long jobs and agents" (:58-63) gives one observation deadline per long job and per agent, "looks once when it passes", and "after a productive look at an agent, it arms the next one".
PASS 5  76.5 The Implementer [read]: .claude/agents/implementer.md holds D2's four points:
  - the new stop rule at :22-24;
  - Corrections at :25-31;
  - Bounded runs at :32-37;
  - no review history in the tree until the final review is complete at :38-41.
  The diff's only hunk starts at line 20, so the frontmatter at :1-8 is unchanged: name implementer, description, model sonnet, effort high, tools Read, Edit, Write, Grep, Glob, Bash, color green. reviewer.md is in neither the diff nor git status. Its frontmatter (reviewer.md:1-10) matches reviewerFields (internal/arch/reviewer_test.go:21-30), and no verifier.md exists. evidence.txt:9-10 reports TestReviewerDefinition passing.
PASS 6  76.6 Nothing contradicts [reported]:
  - My own search of AGENTS.md, CLAUDE.md, doc/evaluation.md and all of .claude/agents, explore.md included, finds no three-correction limit and no same-finding stop. The patterns covered three/twice/at most/attempt/correction/same finding. The remaining hits are about other things: subagents "at most three at once", "once per step" for tier raises, and the definition at evaluation.md:521.
  - The evidence grep, evidence.txt:3-4, exits 1. It discriminates: each of its patterns matched the baseline text.
  - "within its limits" now reads "within its convergence rules (The Reviewer, Convergence)" (evaluation.md:32-33).
  - TestAgentsMDWithinCodexLimit reports 26557 bytes against a ceiling of 28672 (evidence.txt:6-8).
  - The suite exits 0, with build, vet and gofmt clean (evidence.txt:15-37).
Observations:
  - Medium, high confidence: doc/evaluation.md:494-496. The evidence-command template prints no timestamped line per phase. evaluation.md:650 and implementer.md:34-35 now require one, and implementer.md:42-46 tells the Implementer to run the brief's command "exactly as given". If a brief copies the template, the two rules conflict. Steps 2 and 3 run under the new rules. Step 1's own evidence has no timestamps, which is correct under the old rules.
  - Medium, high confidence: doc/evaluation.md:285, The race detector. The race command (`go test -race ./... -count=1`) has no `-timeout=20m`, but the limits table (:634) sets that limit. Followed as written, it runs under Go's 10-minute default. Plan 75's internal/app took 549 s under -race (orchestrator.txt:141), about 91% of that default.
  - Low, medium confidence: evaluation.md:664 sets the first-implementation deadline at 70 minutes, below its own record of 70.2. :654 says each deadline is set at the longest recorded duration. The 70 comes from D3 and does no harm: the look finds work going on and re-arms.
  - Low-medium, medium confidence: evaluation.md:554-555, taken literally, would stop the cycle when a NOT VERIFIED becomes a PASS after a requested test is only run (AGENTS.md:91 "a test to add or run"), with the code unchanged. That is an unwanted stop. The wording comes from D1 word for word.
  - Low, medium confidence: evaluation.md:546-557. A long run of defects that corrections introduced matches no named stop, as long as each prediction holds. Only "a short run ... is a risk signal" hints that a long run might be a stop. Comes from D1.
  - Low, medium confidence: evaluation.md:652-653 sets a deadline when a long job starts "under an agent". The orchestrator cannot see that start without reading the agent's transcript, which would amount to polling. CLAUDE.md:58-59 ("per long job and per agent it starts") reads as covering only the jobs the orchestrator starts. In practice the agent-level deadline covers these jobs. Comes from D3 and D4.
  - Low, medium confidence: evaluation.md:654 and :667 give no first deadline for an agent with no comparable record (for example Explore, or a Reviewer at max). The "new kind of job" row relies on a limit, and an agent has none (no maxTurns, amendment 100).
  - Low, medium confidence: implementer.md:22-24. The self-stop needs both "the failure as it was" and "no new evidence". Attempts that alternate between two failures never meet the first condition, so only the orchestrator's look (evaluation.md:687-689) stops them. Comes from D2.
  - Low, pre-existing: CLAUDE.md:53-55. Raising a Deep (xhigh) Reviewer one tier reaches Exceptional, which CLAUDE.md:47 allows "only when the owner or the plan asks". The new reintroduced-defect trigger makes that raise more frequent.
  - Info: I could not check these figures, because their sources are doc/plan75-evaluation.md (closed to me) or L75's driver (not in the repository): "all of plan 75's 51", "6 minutes per CLI" and the review range "13.3 to 20.6 minutes". The "6 minutes" sits in the column headed "Its longest measured run" (evaluation.md:630, :636) but is the driver's cap, not a measured run. D3 gives 78 minutes for the stalled correction; the doc's 77.5 matches orchestrator.txt.
  - Info: changes.diff includes plan.md's diff. I read only the part on its first page, the plan's head and design, which matches criteria.txt, and went no further. I saw no earlier verdict and no account by the Implementer.
Requested tests: none
Verdict: PASS
````

### Step 2, its review

Agent `aae21f0acd4c2e98f`, effort passed xhigh; 2026-11-04T01:18:45.529Z to 2026-11-04T01:28:07.317Z, 9.4 minutes.

````text
Review - plan 76, step 2 - 8 criteria: 8 PASS, 0 FAIL, 0 NOT VERIFIED

I saw no earlier verdict on this plan's work and no account by the Implementer of its own work. plan.md's "Step 1: done / Step 2: in progress", seen only through changes.diff, is a progress line, not a verdict.

PASS 76.7 The pause [reported]: bullets 1 and 2 are settled by reading the code; bullet 3 rests on the evidence only.
- The provider list holds Claude Code's adapter alone, at C:\javadev\projets\agent-command-center\cmd\agent-command-center\main.go:56. The comment at :39-47 cites amendment 96 and gives the exact line to put back, with the codex import.
- changes.diff touches nothing under internal/provider/codex, testdata/probes or testdata/mock, and new-files.txt adds nothing there.
- Bullet 3 is reported: evidence.txt:22 shows "ok .../internal/provider/codex (cached)" inside `go test ./...`.

PASS 76.8 The paused row [read]
- **The code.** detectionDone records a CLI without an adapter as `Paused: true`, not installed and with no Problem (C:\javadev\projets\agent-command-center\internal\app\app.go:1157-1165). homeCLIs then writes `PausedText` ("paused") with Usable false (C:\javadev\projets\agent-command-center\internal\render\home.go:332-333). The row stays in `supported` order.
- **Greyed in both modes.** Usable false greys the name in text mode (C:\javadev\projets\agent-command-center\internal\render\homegrid.go:319-321) and in picture mode (C:\javadev\projets\agent-command-center\internal\render\homepaint.go:670-672).
- **Not choosable.** A click chooses only an installed CLI (C:\javadev\projets\agent-command-center\internal\app\home.go:279), and the arrows move only among installed ones (C:\javadev\projets\agent-command-center\internal\app\keys.go:192-208).
- **The frames.** Only the 12 startup frames changed, and no frame in testdata/frames still holds "adapter not built yet" (grep). I checked startup-projects-80x24 and startup-resume-120x40 by hand against the diff, and orchestrator.txt's panel-by-panel check agrees for all 12: right panel identical, left panel's non-blank lines identical except the Codex row.
- **The docs.** C:\javadev\projets\agent-command-center\doc\ui-design.md:157-160 and :4115, and C:\javadev\projets\agent-command-center\doc\events-and-state.md:365-367, say so. The latter clause matches the renamed invalid case at C:\javadev\projets\agent-command-center\internal\state\machines_test.go:1259.

PASS 76.9 The history [read]
- **Code.** Rows are dimmed when their CLI is not installed, which a paused CLI never is (C:\javadev\projets\agent-command-center\internal\app\history.go:319). historyResume checks `m.paused` before the installed check and returns with "OpenAI Codex is paused", starting nothing (history.go:213-216). Enter and the double click both go through it (:133, :289). Delete and Delete All are unchanged code (:144-192).
- **Tests.** TestPausedHistory in C:\javadev\projets\agent-command-center\internal\app\paused_test.go:127-193 has one check per case:
  - listed and dimmed (:137-142);
  - Enter: no spawn, the view stays, the note shows, the store is unchanged (:143-147, :158);
  - double click (:152-157);
  - Delete removes only the Codex entry (:162-176);
  - Delete All removes all three (:177-192).

PASS 76.10 No silent fallback [read]
- **--provider codex.** detectionDone sets the note when `m.opt.Provider` names a paused CLI (app.go:1162-1164), and the choice falls back to installed[0] (app.go:1173-1177). The reducer puts the note on the provider field (C:\javadev\projets\agent-command-center\internal\state\actions.go:150-152), and it is drawn as a ✗ line (home.go:169-170).
- **Tests.** TestPausedNoFallback (paused_test.go:198-219) has two subtests: one for --provider codex (choice, field, message, frame) and one for the saved choice (first CLI offered, the row reading "paused").

PASS 76.11 Codex's recordings [read]
- A mock start builds mock.Provider and never looks at Options.Providers (app.go:1201-1207), and Launch with a mock file skips detection (actions.go:140-146).
- No Codex golden frame (dashboard-codex, models-codex*, search-codex, approvals-requested and approvals-review) is in the diff.
- The suite passing is reported (evidence.txt:10, :23).

PASS 76.12 The texts [read]
- **AGENTS.md** holds D6's four sentences word for word, each in its place (changes.diff:43-45, :126-128, :137-140, :149-152).
- **plan.md** holds the standing decision (:1301-1305) and the Recurring sentence (:1313-1314). The Nice to have item is gone (:1322-1325).
- **architecture.md.** D6 gives it no sentence of its own. Its How it starts states the pause in D6's words (C:\javadev\projets\agent-command-center\doc\architecture.md:421-427); see the observation.
- **The ceiling.** TestAgentsMDWithinCodexLimit passing is reported (evidence.txt:11).

PASS 76.13 doc/provider-parity.md [read]
C:\javadev\projets\agent-command-center\doc\provider-parity.md holds everything D7 asks for:
- the purpose (:3-5) and the update rule, with no audit and the spec tables as the reference (:7-14);
- "Codex: paused" with what the pause does (:16-34) and the five-step reactivation checklist (:36-43);
- the table with D7's five columns and three rows, each with a Where that exists: AGENTS.md Review cycle; evaluation.md Convergence and Execution supervision; reliability.md:452 Resumption and ui-design.md:701-708, the resumed session's note (:47-51).
The Codex item has left plan.md's Nice to have list.

PASS 76.14 Broken on purpose [reported]
- evidence.txt:28-41 shows the four named mutants run through `go test -overlay` with a 120 s limit. Each is CAUGHT by a named failing test (TestPausedRow, TestPausedHistory, TestPausedNoFallback, TestPausedRow/TestPausedRowPicture/TestPausedNoFallback), the code hash is equal before and after, and there are 0 survivors.
- The mutants themselves (Temp\acc-mutants-76\step2.sh) are outside the review inputs and were not read. By reading the tests, each named change would fail the test cited.

Observations:
1. **Medium, confidence high.** With Claude Code not installed and Codex installed:
   - the home screen says "No supported agent CLI found. Install Claude Code or Codex CLI, then restart." (C:\javadev\projets\agent-command-center\internal\render\home.go:167-168; ui-design.md:195). Installing Codex cannot help while it is paused.
   - with --provider codex, this first case of the switch hides the "OpenAI Codex is paused" note (home.go:166-171). The row still reads "paused", so nothing is silent.
   - The Known limit covers this state, not the wording of the message.
2. **Low, confidence high.** C:\javadev\projets\agent-command-center\doc\events-and-state.md:344 still reads "both detections returned". The Startup machine does not record that DetectionDone now puts "<CLI> is paused" on the provider field when --provider names a paused CLI. This is for 76.21 at the close.
3. **Low, confidence medium.** During Detecting with --provider codex, ui.Choice is "codex" (app.go:460). resumeNotes (app.go:870-872, C:\javadev\projets\agent-command-center\internal\app\lastsession.go:68-69) can then offer "OpenAI Codex's last session here · Ctrl+R continues it" until detection returns.
   - Ctrl+R meanwhile is refused (actions.go:155), so nothing starts.
   - A click that lands after detection resumes Claude's last session instead of the one shown.
   - This was already true for a not-installed --provider; the pause makes it the usual case for --provider codex.
4. **Low, confidence high.** The home screen still shows "Codex through the proxy … (config.json)" and the codex proxy problem while Codex is paused (home.go:204-212; main.go:51-52).
5. **Low, confidence high.** Ctrl+N clears the "OpenAI Codex is paused" message, as ui-design.md:482 specifies (actions.go:170-173; C:\javadev\projets\agent-command-center\internal\app\home.go:329-343). After it, --provider codex is dropped without the note.
6. **Low, confidence high.** While detecting, the paused row reads "(detecting…)" like every option (home.go:323-324), though nothing is detected for it. This follows ui-design.md:152-153 as approved.
7. **Info.** On 76.8's "differ on that row only": the rows below move up. In startup-resume-80x24 the existing spacers also restore a blank line between the project field and the last-session note (homegrid.go:302-306). No layout code changed, and I judged this the layout's own rule.
8. **Info, confidence medium.** D6 assigns no sentence to doc/architecture.md, so 76.12's architecture part was judged against D6's wording. If D6's "Codex is paused since plan 76 …" sentence was meant to appear there word for word, it does not.
9. **Info.**
   - The parity row omits "design D1" of D7's "(plan 69, design D1)" (provider-parity.md:51).
   - plan.md's Nice to have keeps a one-line pointer to the moved item.
   - The reference frame at C:\javadev\projets\agent-command-center\doc\ui-design.md:71 still shows "OpenAI Codex (0.163.0)".
   - The messages table (ui-design.md:193-206) does not list the paused message.
10. **Info.** No test checks text-mode greying, since the golden frames are plain text. Greying rests on Usable=false, which TestPausedRowPicture checks. That test skips without GDI; the m4 mutant output shows it ran here.

Requested tests:
- Put the four mutant overlays (or their diffs against the code) in the review inputs, so that 76.14 can be read and not only reported.
- Optional, for observation 3: a test with --provider codex and a project whose history holds a Codex entry. It would assert that the frame shown while detecting holds no "OpenAI Codex's last session" note.
- Optional, for observation 1: a test with Codex installed but its adapter left out and no Claude Code. It would check which message the home screen gives.

Verdict: PASS
````

### Step 3, its review

Agent `a0bd427db5a425c80`, effort passed medium; 2026-11-04T09:26:57.190Z to 2026-11-04T09:29:12.365Z, 2.3 minutes.

````text
Review - plan 76, step 3 - 3 criteria: 3 PASS, 0 FAIL, 0 NOT VERIFIED

I found no earlier verdict on this plan's work and no account of it by the Implementer in my inputs.

PASS 76.15 The probe [read, with the run itself reported]:
- I read the recording at C:\javadev\projets\agent-command-center\testdata\probes\p76a-claude-native-agents.jsonl:
  - line 6: `initialize` gives `claude_code_version` "2.1.303".
  - lines 47 and 84: each `system/task_started` carries `subagent_type` "implementer", then "reviewer", with `task_type` "local_agent".
  - lines 52-53: the implementer's messages carry `parent_tool_use_id` toolu_0126… and model `claude-sonnet-5-5`.
  - lines 88-89: the reviewer's messages carry toolu_01AJ… and `claude-opus-5-5`. The orchestrator's own messages are `claude-haiku-5-5`.
  - lines 11 and 17: `system/init` has `agentTools` ["Task","SendMessage"], so SendMessage was offered.
  - line 135: the summary has `sendMessageUsed` false.
- The README entry is at C:\javadev\projets\agent-command-center\doc\research\probes\README.md (diff lines 1115-1117, 1125 and 1133-1139). It matches the recording.
- Sanitizing is reported only: orchestrator.txt section 6 gives the scan counts, all 0, and the `cmp` showing both copies identical. The driver's own replacements are at C:\javadev\projets\agent-command-center\doc\research\probes\drivers\p76-claude-agents.mjs:34-46.

PASS 76.16 The mock file and the frames [read, test results reported]:
- **From P76a through TestMockFiles:** the entry is at C:\javadev\projets\agent-command-center\internal\provider\mock\files_test.go:71-74. It takes `fixture.Of(recs, "p76a")` through the Claude mapper, and TestMockFiles fails when the file differs from the replay (files_test.go:608). The mock file is new, so its diff is the whole file. That the suite passes is reported (evidence.txt line 23).
- **Sizes:** the new frames are native-agents and native-agents-tree at 79x24, 80x24, 120x40 and 200x60, the same sizes as dashboard-* and tree-*.
- **Zone C:**
  - native-agents-120x40.txt lines 32-35, and the same rows in the 200x60 frame: the cards "reviewer │ opus-5-5" and "implementer │ sonnet-5-5", with no effort column.
  - 80x24, lines 21-22: the compact rows, also with name and model and no effort.
  - golden_test.go (diff lines 1332-1350) also asserts 2 agents, their models, and `Effort.Present` false.
- **Zone A:** native-agents-tree-120x40.txt lines 13-16, the 80x24 frame lines 10-13 and the 200x60 frame lines 13-16 show the rows "agent implementer" and "agent reviewer" under Turn 2.
- **Existing frames:** in the diff, the 9 named frames (diff lines 2200-2455) change only by the claude-native-agents.jsonl row and the rows it shifts, apart from file-finder's file count (Observation 1). That no other existing frame changed because of step 3 is reported: orchestrator.txt sections 7-8, where the mock file alone changes exactly these 9 frames and the committed 9 are identical to that output. The startup-* frames in the diff are step 2's work.

PASS 76.17 Broken on purpose, the agents [reported]:
- orchestrator.txt section 9 prints the three mutant diffs:
  - m9, internal/render/zonec.go:184: the card's label becomes "".
  - m10, zonec.go:185: the model becomes "".
  - m11, internal/render/tree.go:479: the agent row is dropped.
- evidence.txt lines 52-60 report each one CAUGHT. m9 and m10 broke native-agents and native-agents-tree at 120x40 and 200x60; m11 broke the four tree frames. The code hash was the same before and after, with 0 survivors.
- The diffs do change what each criterion point names. I did not run them.

Observations:
1. Low severity, high confidence: "each differs only by that row and the rows it shifts" does not literally cover the file-finder frames. Their Quick open title changes from "16 files" to "17 files" (diff lines 2390-2391 and 2423-2424). In file-finder-80x24 (diff lines 2451-2452) that title is the only change, and the new row is not visible at all. The count is correct and follows directly from the new file, so I count no failure. The criterion's wording, or the evaluation record, should name the count.
2. Low severity, high confidence: m11 also "fails" native-agents-tree-79x24, which is only the "Terminal too small" screen (C:\javadev\projets\agent-command-center\testdata\frames\native-agents-tree-79x24.txt). So it is caught by the setup's `t.Fatalf`, which finds no "a:" cursor (golden_test.go, diff lines 1357-1363), not only by comparing frames. The 80x24, 120x40 and 200x60 tree frames would still differ under m11, so the criterion holds.
3. Information, high confidence: the recording keeps the /usage text (time zone "Europe/Brussels", the usage percentages, the MCP name "ccd_turn") and `subscriptionType` "Claude Ultra" (lines 6, 12, 13 and 135). Earlier probes kept the same (p69a, p24u), so this follows existing practice. The driver's comment that the views hold no MCP names (p76-claude-agents.mjs:20-21) is true only of the filtered init views, not of that text.
4. Information: at 80x24, zone A's one-line view reads "Turn 2 › ✓ implementer › ✓ reviewer". This is the existing flat list from internal/render/dashboard.go:475-479, not nesting; the tree frames show the two agents as siblings.

Requested tests: none

Verdict: PASS
````

### The first final review

Agent `a34c1191b8e1f4236`, effort passed xhigh; 2026-11-04T10:02:10.726Z to 2026-11-04T10:17:01.034Z, 14.8 minutes.

````text
Review - plan 76, final review - 22 criteria: 20 PASS, 2 FAIL, 0 NOT VERIFIED

PASS 1  The Review cycle [read]: Step 5 in C:\javadev\projets\agent-command-center\AGENTS.md:92-104 has no count of corrections and no same-finding stop. It states the rule in short, points to doc\evaluation.md, The Reviewer, Convergence, and keeps the two stops that end the plan at once (:102-104). C:\javadev\projets\agent-command-center\doc\evaluation.md:511-564 holds all of D1: the cause, the evidence (other evidence allowed with one line saying why), the falsifiable explanation, the same cause in a new scenario, reintroduced or introduced defects, the stops and the record. AGENTS.md:112-118 keeps the convergence decisions and their evidence in the orchestrator's notes. See observation 2.
PASS 2  What a PASS means [read]: doc\evaluation.md:12-19 has the five points in D1's order.
FAIL 3  Execution supervision: doc\evaluation.md does not hold D3 on observation deadlines for long jobs that an agent starts.
  - D3 (criteria.txt:177-179) says: "The orchestrator sets one when a long job starts, under an agent or under itself, and one for each agent it starts."
  - C:\javadev\projets\agent-command-center\doc\evaluation.md:655-657 says the opposite: "The orchestrator sets one for each long job it starts and for each agent it starts; a long job that an agent runs comes under that agent's deadline."
  - Example: an Implementer's evidence run (go test ./..., which took about 4.5 min in internal/app) gets no deadline of its own. The first look at it comes when the agent's 70- or 32-minute deadline passes, not at the suite's 6.5 minutes. Plan 75's incident, an Implementer stuck for nearly an hour on stalled tests, is exactly that case.
  - It also happened in plan 76: the nine timers in C:\Users\<user>\AppData\Local\Temp\acc-review-inputs\orchestrator.txt:686-696 cover only the agents and the orchestrator's own three jobs.
  - This breaks 76.3 ("doc/evaluation.md holds D3", "observation deadlines … for long jobs"). The rest of D3 is there: :619-653 and :662-700.
  - Either the text says what D3 says again, or the owner amends D3. Changing the approved design is the owner's call (AGENTS.md:102-104).
PASS 4  CLAUDE.md [read]: In C:\javadev\projets\agent-command-center\CLAUDE.md:53-57 the tier rule counts reintroduced and introduced defects. "Long jobs and agents" at :58-63 has the deadline per long job and per agent, the single look, and the next deadline armed after a productive look.
PASS 5  The Implementer [read; reported for the test]: C:\javadev\projets\agent-command-center\.claude\agents\implementer.md:22-41 holds D2's four points. The diff's only hunk starts at line 20, so the frontmatter (lines 1-8: sonnet, high, tools, color) is unchanged. reviewer.md is not in the diff. internal/arch passes (evidence.txt:18).
PASS 6  Nothing contradicts [read; reported for the suite]: A search of AGENTS.md, CLAUDE.md, doc\evaluation.md and .claude\agents finds no three-correction limit and no same-finding stop. The only hits are "At most three at once" about subagents and "the same cause … is the same finding". The Loop says "within its convergence rules" (doc\evaluation.md:32-33). internal/arch and the whole suite pass (evidence.txt:15-34).
PASS 7  The pause [read; reported for the test]: The provider list in C:\javadev\projets\agent-command-center\cmd\agent-command-center\main.go:56 holds Claude Code alone, with the amendment-96 comment at :39-47. No existing file under internal/provider/codex, testdata/probes or testdata/mock is modified; the only additions are step 3's new files. go test ./internal/provider/codex passes (evidence.txt:29).
PASS 8  The paused row [read]:
  - C:\javadev\projets\agent-command-center\internal\app\app.go:1173-1181 marks a CLI without an adapter as Paused. C:\javadev\projets\agent-command-center\internal\render\home.go:359-360 shows PausedText and leaves Usable false, which greys the row in text mode (render\homegrid.go:319-321) and in picture mode (homepaint.go:670).
  - The row cannot be chosen: a click is refused at internal\app\home.go:279, and the arrow keys cycle only the installed CLIs (keys.go:191-209).
  - The 12 startup frames change only on the Codex row; the rows below move up (observation 10).
  - doc\ui-design.md:157-166 and :4117, and doc\events-and-state.md:344, 362-371, describe it.
  - Tested by TestPausedRow and TestPausedRowPicture.
PASS 9  The history [read]: Paused entries stay listed and are dimmed (history.go:319). Enter and a double click on one say "<CLI> is paused" and start nothing (C:\javadev\projets\agent-command-center\internal\app\history.go:213-216). C:\javadev\projets\agent-command-center\internal\app\paused_test.go:127-193 tests listed and greyed, Enter, double click, the store unchanged, Delete and Delete All.
PASS 10  No silent fallback [read]: The --provider note is set at app.go:1177-1179 and reaches the state at state\actions.go:150-152. The choice is reset to the first installed CLI at app.go:1188-1193. Both cases are tested in paused_test.go:198-219.
PASS 11  Codex's recordings [read; reported]: No Codex mock file or Codex frame changed (dashboard-codex, approvals-requested and -review, models-codex*, codex-step and codex-thought, search-codex). A mock run builds mock.Provider and needs no adapter (app.go:1216-1222), and the golden mock rigs pass no Providers at all. TestMockFiles and TestGoldenFrames pass.
PASS 12  The texts [read]:
  - All four AGENTS.md sentences are there word for word: AGENTS.md:8-10, 186-188, 413-415 and 424-427.
  - Both plan.md items are there word for word (changes.diff:1621-1625 and 1634-1635).
  - D6 gives no sentence for doc\architecture.md. Its How it starts section (:421-427) holds D6's clause "it stays, built and tested, but the dashboard does not offer it" (observation 9).
PASS 13  provider-parity.md [read]: C:\javadev\projets\agent-command-center\doc\provider-parity.md has the purpose (:3-5), the update rule (:7-14), "Codex: paused" with what the pause does (:16-34), the five-step checklist (:36-43), and three rows each with a "Where" that exists: reliability.md Resumption is at :452, and the ui-design note is at :703-710. The Nice-to-have item has left plan.md (changes.diff:1643-1649).
PASS 14  Broken on purpose [reported]: In evidence.txt:40-62, mutants m1-m4 run through -overlay are each caught by a failing test, and the hashes before and after are equal. The mutant diffs (orchestrator.txt:521-578) hit the intended lines.
PASS 15  The probe [read; reported for sanitizing]: In C:\javadev\projets\agent-command-center\testdata\probes\p76a-claude-native-agents.jsonl:
  - line 6 shows claude_code_version 2.1.303;
  - line 47 shows task_started with subagent_type implementer;
  - line 52 shows the agent's own message on model claude-sonnet-5-5, with parent_tool_use_id;
  - lines 11 and 17 show agentTools ["Task","SendMessage"], so SendMessage was offered.
  The README row is present. The scan and cmp output is at orchestrator.txt:483-491.
PASS 16  The mock file and the frames [read; reported]:
  - The mock file has its entry in files_test.go (changes.diff:1442-1445), and TestMockFiles regenerates it and compares byte for byte (files_test.go:576-609).
  - The new frames are at 79x24, 80x24, 120x40 and 200x60, the dashboard frames' sizes. native-agents-120x40.txt:31-35 shows two cards with name and model and no effort column. native-agents-tree-120x40.txt:13-16 shows both agents in the tree.
  - Only the 9 listing frames otherwise changed, by the new row and its shifts plus the quick open's count "17 files" (named in amendment 101's record; observation 11). The stand-in copy reproduced those 9 frames byte for byte (orchestrator.txt:493-517).
PASS 17  Broken on purpose, the agents [reported]: m9, m10 and m11 are caught on the new frames (evidence.txt:63-74; diffs at orchestrator.txt:631-668).
PASS 18  The live run [reported; screenshots read]: Second run results are at orchestrator.txt:394-402; compare PASS, no CLI left and projects.json sha256 equal at :402 and :411-426; the projects recycled at :429-433. C:\Users\<user>\AppData\Local\Temp\acc-review-inputs\l76-01-home.png shows "OpenAI Codex paused" greyed. l76-04-done.png shows the zone C cards "reviewer opus-5-5" and "implementer sonnet-5-5". l76-05-tree.png shows both agents in zone A's tree.
PASS 19  The checks [reported]:
  - Build, vet, Linux vet, gofmt and the whole suite are clean on fingerprint a9ae55…, the same before and after (evidence.txt:2-39).
  - -race passes on the seven changed packages (orchestrator.txt:180-207).
  - The timing step ran alone, on mains power ("Online"), with the exact command of doc\evaluation.md:148, and every test was within its target (orchestrator.txt:209-380).
PASS 20  The scope [read; reported]: The diff has 44 tracked files and new-files.txt has 14, all within the plan's files and amendments 101 and 102; go.mod and go.sum are unchanged. evidence.txt:36 reports 58 files with no BOM, CR or user name. My searches for the user name across the tree, and for CR in the new and rule files, found none.
PASS 21  The documents [read]:
  - ui-design.md (:157-166, 196-197, 388-391, 438-441, 4117, 4382-4386), events-and-state.md, architecture.md, provider-parity.md, AGENTS.md, CLAUDE.md and plan.md match the code: app.go:457-476 (proxy lines only while the build holds Codex), render\home.go:166-171 and 220-241, history.go, state.
  - doc\evaluation.md describes the cycle as it was run. Its divergence from D3 is judged under 76.3.
FAIL 22  Specified, observed, not exercised: the draft classes as Observed a part of D3 that plan 76 did not apply.
  - Basis: the draft the orchestrator gives as the evaluation file's content (orchestrator.txt:701-752); the file itself is not in the tree yet.
  - orchestrator.txt:748 classes "One observation deadline per long job and per agent" as Observed. Its evidence lists only the agents and the orchestrator's own jobs.
  - Under D3, the long jobs the Implementers started (their evidence runs in steps 2 and 3) should have had deadlines; they had none (orchestrator.txt:686-696).
  - That part was not applied, yet it is classed Observed. 76.22 forbids exactly that ("observed: applied during plan 76, with its evidence"). This is the same cause as 76.3.
  - Also not settled:
    - the "result block" that must say a PASS means specified and recorded is not drafted;
    - D1's "What a PASS means" is not classified;
    - :743 cites "the suite's longest package 239 s (internal/input)". The final run's longest package was internal/app at 273.234 s, and internal/input took 230.443 s (evidence.txt:17,23). 239.008 s is the internal/input figure in plan 75's third full run (orchestrator.txt:165).
  - Decision record (point 1), reported: decisions 2 to 7, from step 2 on, carry the finding, cause, evidence, prediction and outcome. Decision 1 is step 1's (orchestrator.txt:710-718).

Observations:
1. (info, high) orchestrator.txt:701-718 summarises the step reviews' verdicts and observations. They are given as 76.22's evidence, so they are not an earlier verdict on this work. I judged every criterion independently.
2. (low, high) doc\evaluation.md:557-558 narrows D1's ambiguity stop from "code unchanged" to "code and evidence unchanged". That fits D1's own reason, but it changes the approved text.
3. (medium, high) The tier-raise rule conflicts with the Exceptional tier. CLAUDE.md:53-55 raises the next Reviewer one tier for an introduced defect, but `max` is "only when the owner or the plan asks" (CLAUDE.md:47). Decision 5 resolved this by not raising, without the owner (orchestrator.txt:716, 735).
4. (low, high) Two deadlines are set below or away from "the longest recorded comparable duration":
   - the Implementer's first-step deadline is 70 min against 70.2 recorded (doc\evaluation.md:668; as D3 approved);
   - the full run's deadline was 8 min, not 6.5, with no reason written, although plan 75's 6:25 already included the build and both vets (orchestrator.txt:61-84, 696).
5. (low, medium) The draft's row for "limit ends the whole process tree" (orchestrator.txt:742) says the Implementer used an outer `timeout` wrapper whose tree kill on Windows is unverified. D3 rules out a wrapper that ends only the top process. Go's own -timeout presumably bounded the run.
6. (low, high) The recording, the mock file and four frames keep the owner's "/usage" breakdown, the "Claude Ultra" subscription type and the Px reminder (native-agents-200x60.txt:4-10, 35; recording line 6). There is no credential or user name.
7. (low, high) The proxy line and its problem line are now hidden while Codex is paused (app.go:472-476). This goes beyond D5's text and was added after step 2's review. It is documented in ui-design.md:4382-4386 and tested by TestPausedProxyLines.
8. (low, high) The parity row in provider-parity.md:49 says the Reviewer is "offered only Read, Grep and Glob". It is also offered its report tool and `advisor` (doc\evaluation.md:455-457).
9. (low, medium) Criterion 76.12 names architecture.md, but D6 gives no sentence for it. I judged it on the D6 clause it holds.
10. (low, high) In the startup frames at 80x24 and 120x40, the rows below the Codex row move up 1 or 2 lines, and startup-resume-80x24 places its blank lines differently. This follows from the shorter row; the right panel is identical.
11. (low, high) The quick open's "16 files"→"17 files" title change is in amendment 101's "Taken as" paragraph, which was corrected after step 3's review. It is not in the option the owner chose ("only by that row and the rows it shifts"); the owner may want to confirm it.
12. (low, medium) L76 was re-run without the owner after an L76.2 failure caused by the driver's detection method (decision 7). The screenshot shows the expected screen.

Requested tests:
- For 76.3 and 76.22: the owner's decision on long jobs that agents start, or doc\evaluation.md:655-657 rewritten to D3's rule. After that, the 76.22 row reclassified to match.
- The draft result block for 76.22's third point.
- The run behind "239 s (internal/input)" (orchestrator.txt:743), or the figure corrected to the final run's.

Verdict: FAIL
````

### The second final review

Agent `a32ee636e37a3b97d`, effort passed max; 2026-11-04T10:44:54.205Z to 2026-11-04T11:09:39.396Z, 24.8 minutes.

````text
Review - plan 76, final review - 22 criteria: 21 PASS, 1 FAIL, 0 NOT VERIFIED

PASS 76.1 The Review cycle [read]:
- AGENTS.md:92-104 (step 5) has no count of corrections and no same-finding stop. It states the rule in short and points to doc\evaluation.md, The Reviewer, Convergence. Lines 102-104 keep the two stops that end the plan at once.
- doc/evaluation.md:511-564 holds the amended D1 word for word: the cause (515-524), the evidence and when other evidence is accepted (525-536), the falsifiable explanation (537-539), the same cause in a new scenario (540-548), reintroduced defects (549-552), the four stops (553-560) and the record (561-564).
- AGENTS.md:112-116 keeps "its convergence decisions, with their evidence" in the notes.

PASS 76.2 What a PASS means [read]: doc/evaluation.md:11-18 gives D1's five points.

PASS 76.3 Execution supervision [read]: doc/evaluation.md:612-700 matches the amended D3 item by item.
- A limit before the start, first run included: 619-622. The limit ends the whole tree: 624-628.
- The limits and their sources: 629-643. The 346 s, 549/572 s and 114 s match orchestrator.txt:113, 141-142 and 150.
- Deadlines for each long job and each agent started, an agent's jobs under its deadline, with sources: 655-672. The figures 4:58-6:25, 18:34/19:19, 2:38, 46.7/70.2 and 15.6-31.5 (77.5 left out) match orchestrator.txt:3-26 and 61-178.
- Tests that fail fast, and a mutant that panics or times out counts as caught: 644-647. One heavy job at a time: 648-650.
- The single look: 674-694 (the process tree read twice, an agent's last commands, nothing stopped before its limit, the next observation armed).
- What it cannot guarantee, the limit of detection included: 696-700.
- This means specified and recorded, not proven in behaviour.

PASS 76.4 CLAUDE.md [read]: CLAUDE.md:53-55 counts a defect a correction reintroduced or introduced. CLAUDE.md:58-63 sets one deadline per long job and per agent started, looks once, and arms the next deadline after a productive look.

PASS 76.5 The Implementer [read; the test run reported]:
- .claude/agents/implementer.md holds D2's four points: 22-24, 25-31, 32-37 and 38-41.
- The frontmatter (lines 1-8) is unchanged; the diff's hunk starts at line 20. reviewer.md is not in the diff.
- TestReviewerDefinition (internal/arch/reviewer_test.go:34): the internal/arch package passes (evidence.txt:18).

PASS 76.6 Nothing contradicts [read; the suite reported]:
- Searching AGENTS.md, CLAUDE.md, doc/evaluation.md and .claude/agents finds no three-correction limit and no same-finding stop. The only hit, AGENTS.md:52, is the subagent ceiling.
- doc/evaluation.md:30-33 says "within its convergence rules".
- internal/arch (TestAgentsMDWithinCodexLimit) and the whole suite pass (evidence.txt:14-34).

PASS 76.7 The pause [read; the Codex tests reported]:
- main.go:56 holds &claude.Provider alone. The comment at 39-47 cites amendment 96 and gives the line to put back.
- No existing file under internal/provider/codex, testdata/probes or testdata/mock is in the diff (see O7).
- internal/provider/codex passes (evidence.txt:29).

PASS 76.8 The paused row [read]:
- internal/app/app.go:1173-1181 marks a supported CLI with no adapter as Paused. internal/render/home.go:359-360 then shows "paused" with Usable false.
- Both modes grey it: homegrid.go:313-321 (text) and homepaint.go:670-672 (picture).
- It cannot be chosen: the click is gated at internal/app/home.go:279, and the arrows cycle installed CLIs only (keys.go:192-209).
- The 12 startup frames change only the Codex row and the rows its height moves (see O5). No other frame changed in step 2.
- ui-design.md:157-166 and events-and-state.md:344 and 362-371 say so. Tested by TestPausedRow and TestPausedRowPicture (paused_test.go:59-122).

PASS 76.9 The history [read; the test runs reported]:
- internal/app/history.go:213-216 refuses a paused CLI's entry after the detection check, and line 319 greys it.
- Enter (132-133) and a double click (287-289) both reach historyResume. Delete and Delete All are unchanged.
- TestPausedHistory (paused_test.go:127-193) covers each case: listed and greyed, Enter, double click, store unchanged, Delete, Delete All.

PASS 76.10 No silent fallback [read; the test runs reported]:
- app.go:1177-1179 sets the note for --provider codex, and 1188-1193 falls back to the first installed CLI.
- state/actions.go:150-152 puts the note on question 1, and render/home.go:166-171 shows it.
- TestPausedNoFallback (paused_test.go:198-219) covers both cases. TestPausedNoCLINote (230-259) covers the screen with no usable CLI.

PASS 76.11 Codex's recordings [read; the suite reported]:
- app.go:410-412: a mock run holds no adapter, so main.go's list cannot reach it.
- No Codex frame is in the diff (dashboard-codex, approvals-*, models-codex*, search-codex, tree).
- TestMockFiles and TestGoldenFrames pass (evidence.txt:17 and 30).

PASS 76.12 The texts [read]:
- AGENTS.md:8-10, 186-188, 414-415 and 424-427 hold D6's sentences word for word.
- plan.md holds its two sentences and the moved item (diff lines 1622-1626, 1635-1636 and 1644-1650).
- architecture.md:421-427 (How it starts) carries "it stays, built and tested, but the dashboard does not offer it".

PASS 76.13 provider-parity.md [read]:
- The purpose and the update rule: 3-14. The paused status: 16-34. The five-step checklist: 36-43.
- The table (47-51) has the five columns and the three rows, each with where it is specified. The places it names exist, for example reliability.md:452 and ui-design.md:703-708.
- The Nice to have item has left plan.md (diff lines 1644-1650).

PASS 76.14 Broken on purpose [reported]:
- evidence.txt:40-62: m1 to m4 ran through go test -overlay, each under a 120 s limit. The code hash was equal before and after (7822b3b3…).
- Each was caught by a failing test.
- The mutants' diffs are at orchestrator.txt:530-588.

PASS 76.15 The probe [read; the sanitize run reported]:
- The recording shows claude_code_version 2.1.303 (lines 6 and 11).
- task_started names implementer (47) and reviewer (84). Each agent's messages carry its own model: sonnet-5-5 (52-53) and opus-5-5 (88-89).
- SendMessage was offered: agentTools lists Task and SendMessage (11, 17).
- The README entry is at diff line 1127, the configuration note at 1135-1141. The sanitize run and its scans are at orchestrator.txt:493-501.

PASS 76.16 The mock file and the frames [read; the test runs reported]:
- testdata/mock/claude-native-agents.jsonl, read whole: the labels come from agent.started (46, 84) and the models from agent.progress (51, 88), with no effort. TestMockFiles produces it from the files_test.go:71-74 entry.
- The new frames use the same sizes as the tree-* and dashboard-* frames. The cards appear at native-agents-120x40.txt:32-35, native-agents-200x60.txt:48-51 and native-agents-80x24.txt:21-22, with no effort column.
- The tree rows are at native-agents-tree-120x40.txt:13-16. The test rig checks both agents, their models and no effort (golden_test.go:151-189).
- The 9 listing frames differ by the new row and the rows it moves (see O6).

PASS 76.17 Broken on purpose, the agents [reported]: evidence.txt:63-74. m9 and m10 fail the native-agents and native-agents-tree frames at 120x40 and 200x60; m11 fails the four tree frames. The diffs are at orchestrator.txt:641-678.

PASS 76.18 The live run [reported; screenshots read]:
- orchestrator.txt:404-412: the second run passes every item. The configuration compare exits 0, no CLI is left, and projects.json's sha256 matches.
- l76-01-home.png shows "OpenAI Codex paused" greyed. l76-04 and l76-05 show the cards and the tree.
- Both projects went to the Recycle Bin (orchestrator.txt:439-443). The exe was built from the same Go files (orchestrator.txt:382-385 and 393).

PASS 76.19 The checks [reported]:
- evidence.txt:2-39 (the whole suite, its fingerprint the same before and after): build, vet and Linux vet exit 0, gofmt prints nothing, and every package passes.
- -race passes on every changed package (orchestrator.txt:180-207).
- The timing step ran alone, on mains power, within every budget (orchestrator.txt:209-380).

PASS 76.20 The scope [reported]:
- The diff's 44 tracked files and new-files.txt's 14 all belong to the plan's file lists.
- The byte check reports 58 files with no BOM, no CR and no Windows user name (evidence.txt:35-37). My own searches found no CR and no user name in them.
- git status itself is not among the inputs.

FAIL 76.21 The documents:
- The case: the shipped build (Codex paused, main.go:56) with config.json holding a refused codex setting, for example {"codex":{"proxy":"ftp://x"}}.
- What happens: the startup screen names no problem. newUI copies CodexProxyProblem only when opt.holds("codex") (internal/app/app.go:472-476), so render/home.go:204-206 adds no line. ui-design.md:4381-4385 correctly says this.
- What the document says: doc/architecture.md:418-419 (How it starts, step 1) still says "A problem in the `codex` setting alone is named on its own line".
- architecture.md therefore does not describe what the code does, which 76.21 requires. The proxy hiding came in after step 2's review, and only ui-design.md was updated for it.

PASS 76.22 Specified, observed, not exercised [reported]:
- The decisions' record (orchestrator.txt:714-731) holds decisions 2 to 8 from step 2 on. Each has its finding, its cause with evidence, its prediction and its outcome.
- The classification (733-766) places every rule of D1 and D3 as specified, observed or not exercised.
- The drafted result-block lines for 76.3 and 76.22 say "a PASS here means specified and recorded, not proven in behaviour" (768-777).
- These are drafts; see O9.

Observations:
- O1 (info, high): orchestrator.txt:713-731 states earlier reviews' verdicts on this plan. Steps 1 to 3 passed, and the first final review failed 76.3 and 76.22. They are 76.22's evidence, which AGENTS.md:121-123 allows. I used them for 76.22 only and judged every other criterion on the tree.
- O2 (medium, high): doc/evaluation.md:411-415 (After a CLI update) still has the agent propose a re-check whenever claude or codex is newer. AGENTS.md:411-415 and plan.md's Recurring item (diff lines 1635-1636) now say no re-check is proposed for Codex while it is paused. Two rule documents give opposite instructions. If 76.21's "the cycle" covers this recurring check, this is a second instance of that FAIL.
- O3 (low, high): doc/reliability.md:143 says Codex's proxy is "shown on the startup screen". That is not true while Codex is paused (app.go:472-476). reliability.md is not in 76.21's list.
- O4 (low, high): doc/provider-parity.md:23-29 (What the pause does) leaves out that Codex's proxy line and its problem line are hidden while paused, the addition the owner kept. They come back with the adapter (app.go:472).
- O5 (low, medium): I read "differ on that row only" (76.8) as the Codex row plus the reflow it causes.
  - In startup-resume-80x24 (diff lines 2737-2761) the layout also gives back the spacer above the resume note and one at the bottom, which the baseline had dropped for lack of height.
  - No layout code changed, and every non-blank line other than the Codex row is the same (orchestrator.txt:681-694).
- O6 (low, high): file-finder-80x24 differs only by the quick open's count, 16 files to 17 files (diff lines 2453-2456). Literally, that is not "that row and the rows it shifts". Amendment 101's record (diff lines 829-834) and the owner's decision kept for amendment 103 (orchestrator.txt:390) cover it.
- O7 (low, high): 76.7's "nothing under testdata/probes or testdata/mock changes" is read with D5, which speaks of Codex's probes and mock files. No existing file there changed. The two new Claude files are step 3's own planned files.
- O8 (low, medium): two behaviours during detection, both already in the decision record as information:
  - The paused row reads "(detecting…)" for up to 10 s (render/home.go:350-351), although nothing detects Codex.
  - With --provider codex, the last-session note can offer a Codex session during that time (lastsession.go:37 and 69). Confirm ignores it while detecting (state/actions.go:155-157).
- O9 (low, medium): 76.22 is judged on drafts.
  - doc/plan76-evaluation.md does not exist yet, so the file written at the close must carry them unchanged.
  - Rows marked Observed fold in sub-rules that never occurred: a mutant that panics or times out counts as caught, and a hang becomes a failure with Go's dump.
  - No deadline is listed for the two mutant sets started at 11:37:59 (evidence.txt:40-74; orchestrator.txt:696-708).
  - "-race -timeout=20m" and L76's six-minute cap are claimed but not shown in the printed commands (orchestrator.txt:180-205 and 392-412).
  - "273 s" comes from the 10:37 full run, whose output is not among the inputs.
- O10 (low, low): parts of the orchestrator's evidence are condensed or filtered rather than raw (orchestrator.txt:421, 447 and 455-456), where evaluation.md:478-481 asks for raw output. No criterion depends on them.
- O11 (low, medium): the README's P76a row (diff line 1127) says "Each agent replied READY, the orchestrator DONE". In the recording the agents replied DONE and the orchestrator answered with a summary (lines 53, 89 and 128).
- O12 (low, medium): a continued agent stays unverified for a reason the plan's Known limits do not state. The CLI did offer SendMessage to the orchestrator (recording lines 11 and 17), but the orchestrator passed that instruction on to the agents (lines 45, 53 and 89). The Known limit is worded for a CLI that offers no SendMessage.
- O13 (low, high): evaluation.md:657-668 says each deadline is set at the longest recorded duration, yet the table gives 70 minutes against 70.2 recorded for an Implementer's first implementation. This is the approved D3's own figure.
- O14 (low, low): P76a ran at 02:35 local (orchestrator.txt:461-464) while step 2's correction Implementer, started at 02:33:05 (orchestrator.txt:699), was working. The evidence does not show whether a heavy job overlapped (evaluation.md:648-650).

Requested tests:
- None for the FAIL. It is prose, so the next Reviewer reads doc/architecture.md:418-419 against app.go:472-476 and ui-design.md:4381-4385.
- Not blocking, as evidence for 76.22's record: the -race script's full command lines, L76's driver cap line, and the passing lines of the 10:37 full run behind "273 s".

Verdict: FAIL
````

### The third final review

Agent `ad0d926503fcccd3e`, effort passed max; 2026-11-04T11:22:23.212Z to 2026-11-04T11:46:38.479Z, 24.3 minutes.

````text
I saw earlier review verdicts on this plan's work in orchestrator.txt:809-879, the orchestrator's decisions' record: steps 1 to 3 passed, the first final review failed 76.3 and 76.22, and the second failed 76.21. They are given as evidence for 76.22, which is about them, so under the rule they are not such a verdict. Every judgment below rests on my own reading. Paths are relative to C:\javadev\projets\agent-command-center. orchestrator.txt, evidence.txt, changes.diff and l76-*.png are in C:\Users\<user>\AppData\Local\Temp\acc-review-inputs.

Review - plan 76, final review - 22 criteria: 21 PASS, 1 FAIL, 0 NOT VERIFIED

PASS 76.1  The Review cycle [read]:
- Step 5 (AGENTS.md:92-104) has no count of corrections and no same-finding stop. It states the rule in short and points to doc\evaluation.md, The Reviewer, Convergence.
- AGENTS.md:102-104 keep the two stops that end the plan at once.
- AGENTS.md:112-116 keep "its convergence decisions, with their evidence" in the memory notes, and no count.
- doc/evaluation.md:512-565 holds D1 word for word:
  - the cause, 516-525;
  - the evidence and when other evidence is accepted, 526-537;
  - the falsifiable explanation, 538-540;
  - the same cause in a new scenario, 541-549;
  - reintroduced defects, 550-553;
  - the stops, with amendment 103's "code and evidence unchanged", 554-561;
  - the record, 562-565.

PASS 76.2  What a PASS means [read]: doc/evaluation.md:11-18, in the opening before The loop, gives the owner's five points as D1 lists them.

PASS 76.3  Execution supervision [read]: doc/evaluation.md:613-701 holds D3.
- A limit on every long command before it starts, its first run included: 620-623.
- The limit ends the whole process tree: 625-629. A new kind of job takes its limit from comparable evidence: 641.
- Each limit and each recorded duration has its source: 630-642 and 663-670. The figures match plan 75's logs and transcript tasks in orchestrator.txt:1-178 (346 s; 549/572 s; 114 s; 4:58 to 6:25; 18:34 and 19:19; 2:38; 46.7 and 70.2 min; 15.6 to 31.5 min and 77.5 min).
- Observation deadlines are supplementary, for long jobs and for every agent: 656-661.
- Tests that fail fast, and a mutant caught by a panic or a timeout: 645-648.
- One heavy job at a time: 649-651.
- The single look:
  - the process tree and CPU time read twice, and an agent's last commands and results: 677-681;
  - nothing stopped before a limit: 688-692;
  - the next observation armed after a productive look: 685-687.
- What the safeguards can and cannot guarantee, the limit of detection included: 617-618 and 697-701.
A PASS here means specified, not proven in behaviour.

PASS 76.4  CLAUDE.md [read]:
- CLAUDE.md:53-57: the tier rule counts "a defect a correction reintroduced or introduced".
- CLAUDE.md:58-63, Long jobs and agents: one deadline per long job and per agent, one look when it passes, and the next deadline armed after a productive look at an agent.

PASS 76.5  The Implementer [read; TestReviewerDefinition reported]:
- implementer.md holds D2's four points:
  - lines 22-24: stop when an attempt leaves the failure as it was and brings no new evidence;
  - lines 25-31: discriminating evidence, and the cause found by its code path;
  - lines 32-37: limits, one heavy job, timestamped lines, the touched mutants then all of them;
  - lines 38-41: no review history until the final review.
- The frontmatter (lines 1-8) lies outside the diff's only hunk (changes.diff:5); name, description, sonnet, high, tools and green are unchanged.
- reviewer.md is not in the diff.
- TestReviewerDefinition (internal/arch/reviewer_test.go:34) runs in "ok internal/arch" (evidence.txt:18).

PASS 76.6  Nothing contradicts [read; the ceiling and the suite reported]:
- A search of AGENTS.md, CLAUDE.md, doc/evaluation.md and .claude/agents finds only AGENTS.md:52 (three subagents at once) and doc/evaluation.md:525 (the same cause is the same finding). Neither is a correction limit or a same-finding stop.
- doc/evaluation.md:30-33 now says "within its convergence rules".
- TestAgentsMDWithinCodexLimit (internal/arch/agentsmd_test.go:26) and the whole suite pass (evidence.txt:15-34).

PASS 76.7  The pause [read; Codex's tests reported]:
- main.go:56 holds Claude Code's adapter alone. The comment at 39-47 cites amendment 96 and gives the line and the import to put back.
- No file under internal/provider/codex is in the diff, and no existing file under testdata/probes or testdata/mock. git status's modified list matches changes.diff.
- ok internal/provider/codex (evidence.txt:29).

PASS 76.8  The paused row [read; runs reported]:
- internal/app/app.go:1171-1181 marks a supported CLI with no adapter as Paused.
- internal/render/home.go:359-360 puts "paused" in the version's place, not usable. The name is greyed in the text mode (homegrid.go:319-321) and the picture mode (homepaint.go:670-672).
- It cannot be chosen by a click (internal/app/home.go:279), the arrows (keys.go:192-209) or Confirm (state/actions.go:164-166).
- The 12 frames that showed "not installed: adapter not built yet" now show "OpenAI Codex paused". Their right panels are identical and the rows below shift (changes.diff:2474-2777; orchestrator.txt:773-786).
- ui-design.md:157-166 and 4117, and events-and-state.md:344 and 362-371, say so.
- TestPausedRow and TestPausedRowPicture (paused_test.go:59-122) catch m1 and m4 (evidence.txt:44, 50).

PASS 76.9  The history [read; runs reported]:
- history.go:213-216: Enter or a double click on a paused entry gives "OpenAI Codex is paused" and starts nothing.
- history.go:319 dims the entries of a CLI not installed, a paused one included.
- TestPausedHistory (paused_test.go:127-193) covers each case: listed and greyed, Enter, a double click, the store unchanged, Delete, and Delete All.
- m2 is caught (evidence.txt:46).

PASS 76.10  No silent fallback [read; runs reported]:
- app.go:1177-1179 and 1188-1194, with actions.go:150-152: --provider codex chooses the first installed CLI and gives question 1 the message "OpenAI Codex is paused".
- A Codex choice held before detection falls to the first CLI offered.
- TestPausedNoFallback (paused_test.go:198-219) covers both cases. m3 is caught (evidence.txt:48).

PASS 76.11  Codex's recordings [read; the golden run reported]:
- app.go:1216-1222 starts a mock run on mock.Provider without any adapter.
- No Codex replay frame (dashboard-codex, codex-step, codex-thought, approvals-requested, approvals-review, models-codex) is in the diff or in git status.
- TestGoldenFrames passes (evidence.txt:17).

PASS 76.12  The texts [read]:
- AGENTS.md:8-10, 186-188, 414-415 and 424-427 hold D6's four sentences word for word, at the places D6 names.
- plan.md's diff holds the standing decision (changes.diff:1636-1640), the Recurring sentence (1649-1650) and the Nice to have item moved out (1658-1664).
- D6 gives no sentence for architecture.md. Its How it starts carries D6's wording of the pause (doc/architecture.md:418-431).

PASS 76.13  doc/provider-parity.md [read]:
- Lines 3-14: the purpose, the update rule, and the specifications' own tables as the reference.
- Lines 16-36: Codex: paused.
- Lines 38-45: the five steps of the reactivation checklist.
- Lines 49-53: the five columns and the three rows, each pointing to where its feature is specified.
- The item has left plan.md's Nice to have.

PASS 76.14  Broken on purpose [reported; the mutants read]: evidence.txt:41-62.
- They ran through go test -overlay with -timeout 120s, the code hash equal before and after.
- m1, m2, m3 and m4 were each caught by a failing test.
- Their diffs (orchestrator.txt:623-680) match the code at home.go:279, history.go:213-220, app.go:1178 and render/home.go:360.

PASS 76.15  The probe [read; the sanitizer's scan reported]: in p76a-claude-native-agents.jsonl:
- Claude Code 2.1.303 (line 6), and the <HOME>, <user>, <probe-work> and <signature> placeholders (lines 1, 11, 21, 55, 91).
- task_started with subagent_type implementer and reviewer (lines 47, 84).
- Each agent's messages carry its own model, claude-sonnet-5-5 and claude-opus-5-5 (lines 52-53, 88-89).
- SendMessage was offered (lines 11, 17) and not used (line 135).
- The README entry is at README.md:25-27, 119 and 347-353.

PASS 76.16  The mock file and the frames [read; TestMockFiles reported]:
- files_test.go:74 replays P76a's process "p76a", and the mock file equals that replay (evidence.txt:30).
- The new frames, at 79x24, 80x24, 120x40 and 200x60, show:
  - zone C's two cards with their names and models and no effort column (native-agents-120x40.txt:32-35, -200x60.txt:48-51, -80x24.txt:21-22);
  - zone A's tree with both agents (native-agents-tree-80x24.txt:10-13, -120x40.txt:13-16, -200x60.txt:13-16).
- The 9 listing frames gain the new row and shift (changes.diff:2218-2473).

PASS 76.17  Broken on purpose, the agents [reported; the mutants read]: evidence.txt:63-74.
- m9 (card without the name) and m10 (card without the model) each fail native-agents-120x40, native-agents-200x60, native-agents-tree-120x40 and native-agents-tree-200x60.
- m11 (tree without the agent) fails the four tree frames.

PASS 76.18  The live run [reported; screenshots read]:
- l76-01-home.png: "OpenAI Codex paused", greyed, under "Claude Code (2.1.303)".
- l76-05-tree.png: zone A's tree names "agent implementer" and "agent reviewer".
- l76-04-done.png and l76-05-tree.png: zone C shows "reviewer opus-5-5" and "implementer sonnet-5-5".
- orchestrator.txt:495-504: no CLI left, and projects.json's sha256 equal after it was put back.
- orchestrator.txt:513-528: configuration PASS.
- orchestrator.txt:531-535: both temporary projects recycled.

PASS 76.19  The checks [reported]:
- evidence.txt:1-39, at the tree's fingerprint: build, vet and Linux vet exit 0 with no output; nothing printed between "gofmt -l ." and "gofmt done"; every package ok.
- -race with -timeout=20m on every changed package is ok (orchestrator.txt:180-207). Every Go file is unchanged since that run (382-385).
- The timing step ran alone, on mains power, with the exact command of doc/evaluation.md:148. All 26 tests are within their targets (orchestrator.txt:209-380).

PASS 76.20  The scope [read; the untracked list and BOM reported]:
- Every file in changes.diff and new-files.txt is one of the plan's files, amendments 101 and 102 included.
- A search of the repository (plan.md and the plan evaluation files left out) and of changes.diff finds no Windows user name.
- No CR is in the plan's files or in the diff.
- The byte check reports BOM 0, CR 0 and user name 0 across 58 files (evidence.txt:35-36).

PASS 76.21  The documents [read]: the listed documents match the code and the cycle:
- ui-design.md:157-166, 196-197, 387-391, 438-441, 4117 and 4381-4385;
- events-and-state.md:344 and 362-371;
- architecture.md:418-431;
- evaluation.md:11-18, 30-33, 415-416, 492-500, 512-565 and 613-701;
- provider-parity.md, AGENTS.md and CLAUDE.md;
- checked against app.go:466-476 and 1163-1209, history.go:213-220, and render/home.go:166-171, 220-243 and 344-371.
Observations below.

FAIL 76.22  Specified, observed, not exercised: the classification orchestrator.txt gives "as the evaluation file will hold" it (orchestrator.txt:830-868) misses two rules and mis-classes a third. The evaluation file itself comes at the close, so this judgment rests on that draft.
- Two rules of D3's "What the look decides" are in no class:
  - a probable stall: recorded, nothing stopped before its limit, the stalled test run again alone for its dump (evaluation.md:688-692);
  - an agent repeating failures: stopped and resumed, and a loop it cannot explain goes to the owner (evaluation.md:693-695).
- The draft's only row for the look (orchestrator.txt:865) is "Observed", on a single look that found a legitimate run. Neither scenario ever occurred, so each is either unclassed or claimed observed without an occurrence. 76.3 itself lists "nothing stopped before a limit" as a point of D3, beside "the next observation armed", which the draft does class (866).
- "Stop: a refuted prediction" is classed not exercised, with the evidence "No prediction was refuted" (847). Yet the draft counts decision 1's fixes as a correction ("three defects from decision 1's correction", 846). Their prediction, "Each fix makes the text say what D1 and D3 say" (820), was refuted at the first final review (827), and a new cause then explained it. By its own record, the rule had an occurrence.
This breaks 76.22's second point, on exactly the scenarios not exercised that amendment 100's observation 3 asked to have named.

Observations:
- Medium, high confidence: two tier rules conflict when a Deep plan's next Reviewer is already at xhigh. CLAUDE.md:53-55 (and evaluation.md:550-553) raise it one tier for a defect a correction introduced; CLAUDE.md:47 allows max only when the owner or the plan asks. Neither says which prevails. Decision 5 applied no raise and decision 8 asked the owner (orchestrator.txt:824, 827). D1 would treat this as a question only the owner can settle.
- Low to medium, high confidence: process deviations, recorded and no longer correctable:
  - P76a started while step 2's correction Implementer worked (orchestrator.txt:858; evaluation.md:649-651);
  - the second mutant set and L76's second run had no observation deadline (863; evaluation.md:656-658, 673);
  - the Implementer's outer timeout wrapper's tree kill on Windows is unverified (853; evaluation.md:627-629).
- Low, high confidence: while detection runs, the paused row reads "(detecting…)" (render/home.go:350-351), though no adapter is detecting it. ui-design.md:152-153's general rule covers it, and it is transient.
- Low, high confidence: with --provider codex and a Codex entry in the project's history, the home screen offers "OpenAI Codex's last session here: … · Ctrl+R continues it" during detection (lastsession.go:37, 67-69; Choice comes from opt.Provider at app.go:470). Ctrl+R is ignored while detecting (actions.go:155-157). It was the same before the plan for a CLI not installed.
- Low, high confidence: doc/reliability.md:143 still says Codex's proxy is "shown on the startup screen", which the paused build no longer does (app.go:472-476). That file is outside the plan's files and 76.21's list.
- Low, high confidence: 76.16's words say "only by that row and the rows it shifts", but the three file-finder frames also change the quick open's count to "17 files" (changes.diff:2408-2409, 2441-2442, 2469-2470). In file-finder-80x24 the count is the only change. Amendment 101's record names the count and the owner kept it (orchestrator.txt:482); the criterion's text was not amended.
- Low, medium confidence: in startup-resume-80x24 the two freed rows also restore the blank line between the field and the last-session note (startup-resume-80x24.txt:15). This is the layout's own spacing, not a plain shift; no other row's content changed.
- Low, medium confidence: 76.7's literal "nothing under testdata/probes or testdata/mock changes" meets step 3's two new files there. No existing file changed, which is D5's meaning.
- Low, medium confidence: m11 is caught by the tree rig's own guard (golden_test.go:186, t.Fatalf when no agent cursor is reached) before any frame comparison. The subtests fail as 76.17 asks, but the frames' text is not what catches it.
- Low, medium confidence: the deadline row for "an agent with no comparable record" (evaluation.md:671) is in neither D3's approved list nor amendment 103's decisions. It agrees with D3's "longest recorded comparable duration".
- Low, medium confidence: evaluation.md:591-593 says the orchestrator's full run happens "once per plan". AGENTS.md:105-108 repeats it after the last correction, and this plan ran it three times (orchestrator.txt:795, 799, 802).
- Low, medium confidence: noCLIText asks to install every CLI that is not paused (render/home.go:236-238). So "Install Claude Code" also shows when Claude Code is installed but its detection failed; the old text had the same gap.
- Low, high confidence: the decisions' record carries the orchestrator's readings of criteria ("76.21 holds and 76.22's rows are true", orchestrator.txt:828), which evaluation.md:479-482 keeps from a Reviewer. Here they come as 76.22's evidence, which D1's record requires.

Requested tests: none. The correction is prose, the classification, and its evidence is the Reviewer's reading.

Verdict: FAIL
````

### The fourth final review, with no report

Agent `affb28495c8e9b6f2`, effort passed max; 2026-11-04T11:53:17.508Z to 2026-11-04T12:18:34.256Z, 25.3 minutes.

It reached its 100-turn limit and handed back no report; under `AGENTS.md` it counted for nothing, and the fifth ran in its place.

### The fifth final review

Agent `a94b39e7ea9b3d455`, effort passed max; 2026-11-04T12:21:36.185Z to 2026-11-04T12:42:56.785Z, 21.3 minutes.

````text
Earlier verdicts I saw: C:\Users\<user>\AppData\Local\Temp\acc-review-inputs\orchestrator.txt:892-1005 (the decisions' record and the classification) quotes the verdicts of this plan's step reviews and of four earlier final reviews. It is given as 76.22's evidence, which AGENTS.md:121-123 says is not "such a verdict", so I used it for 76.22 only. Repository paths below are relative to C:\javadev\projets\agent-command-center. orchestrator.txt, evidence.txt, changes.diff and the l76-*.png screenshots are in the inputs folder named above.

Review - plan 76, final review - 22 criteria: 22 PASS, 0 FAIL, 0 NOT VERIFIED
PASS 76.1  The Review cycle [read]:
- AGENTS.md:92-104 (step 5) holds no correction count and no same-finding stop. It states the rule in short and points to doc\evaluation.md, The Reviewer, Convergence.
- AGENTS.md:102-104 keeps the two stops that end the plan at once.
- doc/evaluation.md:512-565 holds D1 in full: the cause 516-525, the evidence and when other evidence is accepted 526-537, the falsifiable explanation 538-540, the same cause in a new scenario 541-549, reintroduced defects 550-553, the stops 554-561 and the record 562-565.
- AGENTS.md:115-116 keeps "convergence decisions, with their evidence" in the orchestrator's notes.
PASS 76.2  What a PASS means [read]: doc/evaluation.md:11-18, in the opening before The loop, gives D1's five points in D1's words.
PASS 76.3  Execution supervision [read]: doc/evaluation.md:613-701 holds D3:
- a limit before every long command, first run included (620-623), ending the whole tree (625-629);
- the limits table with each measured run and its source (630-644), a new kind of job from comparable evidence (641);
- supplementary deadlines for long jobs and for every agent started, an agent's jobs under its deadline (656-673, each with its "Recorded" source);
- fail-fast tests, and mutants caught by a panic or a timeout (645-648); one heavy job at a time (649-651);
- the look: the process tree read twice (677-678), an agent's last commands and results (679-681), nothing stopped before its limit (688-692), the next observation armed (685-687);
- the limits of the safeguards and of detection (617-618, 697-701).
The figures match plan 75's logs in orchestrator.txt:61-178 (4:58 to 6:25; 18:34 and 19:19; 2:38; 346 s; 549 and 572 s; 114 s).
PASS 76.4  CLAUDE.md [read]: CLAUDE.md:53-57 counts "a defect a correction reintroduced or introduced" in the tier raise. CLAUDE.md:58-63 sets one deadline per long job and per agent, one look when it passes, and the next deadline after a productive look at an agent.
PASS 76.5  The Implementer [read; the test reported]:
- .claude/agents/implementer.md holds D2's four points at 22-24, 25-31, 32-37 and 38-41.
- The diff's only hunk starts at line 20 (changes.diff:5), so the frontmatter (lines 1-8) is unchanged.
- reviewer.md is not in the diff.
- internal/arch passes: evidence.txt:18.
PASS 76.6  Nothing contradicts [read; the suite reported]:
- A search of AGENTS.md, CLAUDE.md, doc/evaluation.md and .claude/agents finds no correction limit and no same-finding stop. The one "same finding" (evaluation.md:525) defines a finding.
- evaluation.md:32-33 now says "within its convergence rules".
- TestAgentsMDWithinCodexLimit (ceiling 28,672, internal/arch/agentsmd_test.go:21) and the whole suite pass: evidence.txt:15-34.
PASS 76.7  The pause [read; the codex tests reported]:
- main.go:56 holds Claude Code's adapter alone. The comment at 39-47 cites amendment 96 and gives the line to put back.
- No existing file under internal/provider/codex, testdata/probes or testdata/mock is modified (O3).
- internal/provider/codex passes: evidence.txt:29.
PASS 76.8  The paused row [read]:
- internal/app/app.go:1171-1181 marks a supported CLI with no adapter Paused.
- internal/render/home.go:359-360 shows "paused", not usable, so the row is greyed in the text mode (homegrid.go:319-321) and the picture mode (homepaint.go:670-672).
- It cannot be chosen: a click needs the CLI installed (internal/app/home.go:279), and the keys cycle installed CLIs only (keys.go:192-209).
- The 12 frames that showed "adapter not built yet" (startup-projects, startup-projects-focused, startup-favourites and startup-resume at 3 sizes) now read "paused". Their right panels are identical, and the other left rows keep their content, moved up as the row got shorter (changes.diff:2474-2775; orchestrator.txt:793-806).
- No other frame changes, apart from 76.16's 9.
- ui-design.md:157-166, 4117 and events-and-state.md:344, 362-371 say so.
PASS 76.9  The history [read; the tests reported]:
- history.go:319 dims an entry whose CLI is not installed, paused included.
- history.go:213-216 stops Enter and the double click (287-289) with "OpenAI Codex is paused" before any start.
- Delete and Delete All are unchanged (history.go:142-192).
- Each case has its test in internal/app/paused_test.go:127-193; mutant m2 is caught (evidence.txt:46).
PASS 76.10 No silent fallback [read; the tests reported]:
- app.go:1177-1179 sets the note for --provider codex, and app.go:1188-1193 chooses the first installed CLI.
- state/actions.go:150-152 and render/home.go:166-171 show the note.
- Both cases are tested in paused_test.go:198-219; mutant m3 is caught (evidence.txt:48).
PASS 76.11 Codex's recordings [read; the replays reported]: no Codex golden frame is in the diff. The mock run starts mock.Provider with no adapter (app.go:1216-1222). TestMockFiles and TestGoldenFrames pass (evidence.txt:17, 30).
PASS 76.12 The texts [read]:
- AGENTS.md:8-10, 186-188, 413-415 and 424-427 hold D6's sentences word for word.
- plan.md holds the standing decision, the Recurring sentence and the moved Nice to have item (changes.diff:1636-1640, 1649-1650, 1658-1664).
- architecture.md:418-428 (How it starts) carries D6's clause verbatim (O2).
PASS 76.13 doc/provider-parity.md [read]:
- It holds the purpose (3-5), the update rule (7-14), Codex: paused (16-36), D7's five-step checklist (38-45) and the table with D7's columns and three rows, each with its "Where" (47-53).
- The Nice to have item has left the list (changes.diff:1658-1664).
PASS 76.14 Broken on purpose [reported; the mutant diffs read]:
- m1 to m4 (orchestrator.txt:643-700) are the four named mutants.
- Each fails a test under go test -overlay, with the code hash equal before and after (evidence.txt:40-62).
- The catching assertions are at paused_test.go:63-65, 84-98, 115, 145-157 and 202-207.
PASS 76.15 The probe [read]:
- testdata/probes/p76a-claude-native-agents.jsonl records Claude Code 2.1.303 (lines 6 and 11) and is sanitized (placeholders at 1, 21 and 55).
- task_started names implementer (47) and reviewer (84); their messages carry claude-sonnet-5-5 (52-53) and claude-opus-5-5 (88-89).
- SendMessage was offered to the orchestrator: agentTools at 11 and 17.
- The README entry is at doc/research/probes/README.md:25-27, 119 and 347-354.
PASS 76.16 The mock file and the frames [read; the replay equality reported]:
- The files_test.go entry (changes.diff:1457-1460) and the byte comparison in files_test.go:576-609 tie the new mock file to P76a. I read it whole: the agents at 46 and 84, their models at 51 and 88.
- The new frames are at golden_test.go:40's four sizes:
  - native-agents-120x40.txt:31-35 and -200x60.txt:47-51 show both cards with name and model and no effort column;
  - native-agents-tree-80x24.txt:9-13 and -120x40.txt:12-16 show the tree with both agents.
- The 9 listing frames differ by the new row and the rows it shifts, plus the count (O4). They are byte-identical to a stand-in run made before any step 3 code (orchestrator.txt:615-639).
PASS 76.17 Broken on purpose, the agents [reported; the diffs read]: m9, m10 and m11 (orchestrator.txt:753-790) each fail the new frames (evidence.txt:63-74).
PASS 76.18 The live run [reported; the screenshots read]:
- The second run passes every item (orchestrator.txt:515-548): compare "configuration: PASS", no CLI left, projects.json sha256 equal.
- The folders were moved to the Recycle Bin (551-555).
- l76-01-home.png shows "OpenAI Codex paused", greyed. l76-04-done.png and l76-05-tree.png show reviewer opus-5-5, implementer sonnet-5-5 and both agents in the tree.
- The binary was built from Go files identical to today's (382-385, 505).
PASS 76.19 The checks [reported]:
- On fingerprint 5383d3d8, build, vet, linux vet and every package pass, and gofmt prints nothing (evidence.txt:2-39).
- -race passes on all 7 changed packages, each with -timeout=20m (orchestrator.txt:180-207).
- The timing step ran alone, on mains power ("Online"), every budget met (209-380).
PASS 76.20 The scope [read; the untracked list and the byte check reported]:
- The 44 modified and 14 new files are all in the plan's lists (steps 1-3, the close, amendments 101-102).
- go.mod, go.sum, scripts, reviewer.md and the rest of internal/provider are untouched.
- 58 files: no BOM, no CR, no user name (evidence.txt:35-37). My own searches found no CR and no user name either.
PASS 76.21 The documents [read]: these match the code and the cycle read above, with O1 and O6 as notes:
- ui-design.md:157-166, 196-197, 386-391, 436-441, 4117 and 4381-4385;
- events-and-state.md:344 and 362-371;
- architecture.md:418-428;
- evaluation.md (as in 76.1-76.3, and 415-416);
- provider-parity.md, AGENTS.md, CLAUDE.md, and plan.md through the diff.
PASS 76.22 Specified, observed, not exercised [reported]:
- The record (orchestrator.txt:893-913) gives decisions 2-11 from step 2 on, each with its finding, its cause and evidence, a prediction (none for decision 6, O9), the decision and what came of it.
- Every D1 and D3 bullet (831-890) maps to a row (928-935), and each row has one class and its evidence; exceptions are stated (D3.9, D3.10, D3.11).
- The result block's lines say "a PASS here means specified and recorded, not proven in behaviour" (1002-1005).
- Judged on this draft (O10); O7 and O8 qualify two rows.

Observations:
- O1 (low; high confidence): doc/reliability.md:143 and :351 are outside the plan's files and are now wrong. They still say Codex's proxy is "shown on the startup screen" and that a missing CLI shows "not installed". While Codex is paused, the home screen hides the proxy lines (app.go newUI) and shows "paused".
- O2 (low; medium confidence): 76.12 names architecture.md (How it starts), but D6 gives it no sentence. I accepted architecture.md:422-425, which carries D6's clause "it stays, built and tested, but the dashboard does not offer it" verbatim. A reading that demands AGENTS.md:8-10's whole sentence there would differ.
- O3 (low; high confidence): I read 76.7's "Nothing under testdata/probes or testdata/mock changes" as "no existing file changes". Step 3 adds two new files there, as the plan's file list says.
- O4 (low; high confidence): the 9 listing frames also change the quick open's count from 16 to 17 files; in file-finder-80x24 it is the only change. Literally that is beyond "that row and the rows it shifts". Amendment 101's record covers it, and the owner kept it (orchestrator.txt:502).
- O5 (low; high confidence): until detection returns, the paused row reads "(detecting…)" (render/home.go:350-351), although no detection runs for it. This follows ui-design.md:152-153.
- O6 (low; medium confidence): provider-parity.md:52 says deadlines are "set with Claude Code's Monitor tool". Plan 76's deadlines were background timers (orchestrator.txt:808-826), which evaluation.md:659-660 allows.
- O7 (medium; medium confidence): the D1.5 row's "Observed" (orchestrator.txt:948) rests on decision 2, a fix of a PASS's observations, not a correction after a FAIL. Its mutants ran after the fix, so they show the tests failing on the old behaviour but not that the tests were "written before the fix". No correction after a FAIL in this plan used a regression test.
- O8 (medium; medium confidence): the agent look's output (orchestrator.txt:459-497) prints each call twice, the second line with an "undefined" time and no result. It shows commands but not "their results" (evaluation.md:679-681). Such a look could not tell a looping Implementer's failing runs from passing ones (D3.20).
- O9 (low; high confidence): decision 6 records no prediction ("—", orchestrator.txt:908).
- O10 (low; high confidence): the evaluation file and amendment 103 are not in the tree yet. AGENTS.md:30 and plan.md's D1 and D3 cite amendment 103. Both must be written at the close as drafted.
- O11 (low; high confidence): evaluation.md:494 says each phase prints its start and its end with the time. The template at 499 prints only an end line for gofmt, and an untimed fingerprint.
- O12 (low; medium confidence): Ctrl+N clears "OpenAI Codex is paused" with the other messages (state/actions.go:170-174), as ui-design.md's Ctrl+N rule says.
- O13 (low; medium confidence): the P76a recording and its mock keep the owner's /usage report (Europe/Brussels reset time, request counts), the "Claude Ultra" subscription, and the names of skills and MCP commands (p76a jsonl:6, 12-13, 19, 135; mock:10). The sanitizer's rules do not cover these, and p69a and p24u hold the same kind of text.

Requested tests:
- O7: the step 2 correction's transcript or evidence showing TestPausedNoCLINote and TestPausedProxyLines failing on the unfixed tree before the fix. Otherwise, class D1.5 as not exercised after a FAIL, and name decision 2 as its application to observations.
- O8: run p76-look.mjs on an Implementer transcript that holds a failing go test, and show that its output carries each result.

Verdict: PASS
````
