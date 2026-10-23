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
bc5d92b:plan.md` for plan 33), and its results in its evaluation file.

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

## Plan 34 - The model selector, its effort and Ultracode

Brief amendment 34 (the owner's request, 2026-10-23) and its image,
`references/11-model-selector.webp` (reference-design.md, The model
selector): a click on the editor row's model pill opens the active CLI's
models with the effort and, for Claude Code, Ultracode; a choice is
applied through the CLI's own interface and shown once the CLI reports it
in force. Nothing is committed until the owner has reviewed the
implementation and the screenshots.

### What the research found

Sources: the installed Claude Code 2.1.303 and Codex CLI 0.163.0 (the
versions last probed); the Agent SDK reference and the `model-config` and
`workflows` pages (read 2026-10-23); Codex's app-server schema as the
installed CLI generates it (`codex app-server generate-json-schema`, with
and without `--experimental`) and its source at `rust-v0.163.0`; four
probes that send no message, so no model call (P34c, P34c2, P34c3 and
P34x in `doc/research/probes/README.md`; the configuration check PASS
around them). The details: `doc/research/claude-code.md` and
`doc/research/codex-cli.md`, "Models, effort and Ultracode (plan 34)".

1. Claude Code's catalogue is the `initialize` response's `models`, the
   documented `ModelInfo` (`value`, `resolvedModel`, `displayName`,
   `description`, `supportsEffort`, `supportedEffortLevels`): 13 entries
   on the owner's account, five aliases first (Default (recommended), Opus
   5.5, Fable 5.1, Sonnet 5.5, Haiku 5.5), then eight full model ids. The
   descriptions change with the account (P303a's and P34c's differ).
2. `set_model {model}` (documented, `setModel`) changes the model at
   once: answered in 3 to 1,530 ms; an unknown model is refused ("Model
   'claude-not-a-model-34' not found"). `apply_flag_settings {settings}`
   (documented, `applyFlagSettings`) takes `effortLevel` and `ultracode`
   for the session only. Neither writes a settings file (P34c: settings.json
   identical).
3. `get_settings` (in the CLI's own schema, absent from the documentation:
   undocumented) answers `applied`: `model`, `effort`, `ultracode`,
   `ultracodeRequested` and `ultracodeAvailable` ("Hosts should offer an
   Ultracode control only while this is true"), in 6 to 50 ms, with no
   model call. It is the only source of the effort and Ultracode in force:
   `apply_flag_settings` answers success even for a level it then drops
   ("extreme", P34c).
4. The owner's Claude Code settings set `CLAUDE_CODE_EFFORT_LEVEL` in their
   `env` block; the variable outranks every other effort setting
   (`model-config`, "Adjust effort level"), and `applied.effort` stayed
   "max" through every change (P34c). With the variable emptied for one
   process (P34c3), each level asked for applied, Opus 4.6 ran "xhigh" as
   "high" (the documented fallback), and clearing the level gave the
   model's default.
5. Changing `effortLevel` alone turned Ultracode off (P34c3 after "low" and
   "medium", P34c after "max"), where the documentation says a level picked
   in `/effort` leaves it as it was; `{effortLevel, ultracode}` in one
   request kept it (P34c3). Turning Ultracode on or off left the effort as
   it was. Ultracode is unavailable on a model without "xhigh" (Opus 4.6,
   `ultracodeAvailable` false) and when workflows are off (documented).
6. Codex's catalogue is `model/list`, paged: one model a page gave
   `nextCursor` "1" to "6", then none (P34x); 7 visible models, 10 with
   `includeHidden` (an experiment, a reserve model, the auto-review model).
   Each has `displayName`, `description`, `isDefault`,
   `supportedReasoningEfforts` (low to max, "ultra" on five of them) and
   `defaultReasoningEffort`; no field marks a model as older.
7. A Codex thread starts on the configured model (`gpt-6-sol`, "high",
   P34x), not the catalogue's default (`gpt-6.1-sol`); `thread/read` reads
   both back with no model call. `turn/start` takes `model` and `effort`
   "for this turn and subsequent turns" (stable). The immediate
   `thread/settings/update` is refused without the experimental API
   ("thread/settings/update requires experimentalApi capability"); with
   it, it took an unknown model and an unknown effort, both echoed by
   `thread/settings/updated` (P34x), so it confirms no more than a turn
   does.

### Design

- **D1. Where the choices come from.** Claude Code: the `initialize`
  response's models; the settings in force read by `get_settings` after
  the handshake, after each change and after each turn, its answer reduced
  to `applied` (and, from `effective`, the model and the one variable of
  D5) before it becomes an event, so no other setting reaches the screen.
  Codex: every page of `model/list` without hidden models; the settings in
  force from the `thread/start` response, then from `thread/read` after a
  turn that carried a change. The details view names each source, the
  undocumented `get_settings` marked "†". No model list is written in the
  code: the mock provider replays the probes' lists. The CLIs' own lists
  already leave out what an account or an organisation excludes.
- **D2. The main list and More models.** The first five models in the
  CLI's order are the list, the others under More models, collapsed. On
  the owner's accounts: Claude Code's five aliases, then eight models;
  Codex's four GPT-6 models and GPT-5.6-Sol, then GPT-5.6-Terra and
  GPT-5.6-Luna. Codex's default model reads "(default)" after its name.
  The model in force is visible at opening: under More models, the section
  opens expanded and scrolled to it.
- **D3. Claude Code's changes.** A model: `set_model` with the entry's
  value. An effort: `apply_flag_settings` with `effortLevel` and the
  Ultracode in force (finding 5). Ultracode: `apply_flag_settings` with
  `ultracode` alone. Each answer is followed by `get_settings`; the pill,
  the check and zone A's and zone B's model take a new value only from
  that read. A refusal, or a read that shows another value than the one
  asked for, leaves the screen as the read shows it and puts the CLI's
  words, or what it kept, in zone B as a notice.
- **D4. Codex's changes go with the next message** (finding 7): the model
  and effort asked for are sent with the next `turn/start`; until then
  the pill shows what is in force and the editor row's middle "→ <model>
  <effort> (next turn)", as plan 12 shows Codex's approvals. Once that
  `turn/start` is answered, `thread/read`, and the pill takes its values;
  a refused turn leaves them. A model change sends an effort the new model
  lists: the one in force when it lists it, else its default. The
  experimental `thread/settings/update` is not used.
- **D5. Effort.** The list is the chosen model's own levels (Claude Code's
  `supportedEffortLevels`, Codex's `supportedReasoningEfforts`), in the
  CLI's order, labelled Low, Medium, High, XHigh, Max, Ultra (another word
  with its first letter upper), Codex's default one followed by
  "(default)". A model without levels shows "N/A" in the box with no
  chevron, not clickable. When `CLAUDE_CODE_EFFORT_LEVEL` is set, in the
  dashboard's environment or in a settings `env` block `get_settings`
  reports, the box shows the level in force without the chevron, and a
  click puts the reason in zone B and sends nothing.
- **D6. Ultracode, Claude Code only.** The row reads "Ultracode" and
  "Enables dynamic workflows"; its toggle shows the Ultracode in force.
  While `ultracodeAvailable` is false or unknown the row is dimmed, its
  toggle off, and a click puts the reason in zone B and sends nothing.
  Codex has no such row.
- **D7. The pill.** " <model> <Effort>", then " · Ultracode" only while it
  is in force: Claude Code's model by the catalogue's name for the model in
  force (else its id, read as zone B's speaker reads it), Codex's by its
  display name; no effort while none is known. The "+" band, the middle and
  "↯ Auto" are unchanged. The pill carries no "†" (the owner's examples
  have none); the details view does.
- **D8. The popup** (reference-design.md, The model selector, for the
  columns and colours): its bottom edge on the editor's rule, its left
  edge one cell left of the pill, a "▾" in the bottom edge above the
  pill's first cell; "CLAUDE MODELS" or "CODEX MODELS", "Current session"
  at the right; each model two rows, its name and its description, one
  blank row between models; the row under the cursor on the band with a
  pink "▶", the model in force with "✓"; separators around "More models";
  the effort row; the Ultracode row; the bottom edge on the next row. At
  least 63 cells wide, wider for a longer description, never wider than
  the screen; the list scrolls inside, the right edge its scroll bar, the
  popup no taller than the rows above the editor's rule.
- **D9. Keys and mouse.** A click on the pill, or Alt+P (Claude Code's own
  key for its model picker), opens and closes it. While it is open: ↑↓
  move the cursor over the models, More models, Effort and Ultracode;
  Enter or Space acts on it; PgUp, PgDn and the wheel scroll; Esc closes
  the effort list, then the popup; the pointer's hover moves the cursor
  (all mouse motion asked for, as plan 30's menu); a click on a row acts
  on it; a click outside closes it and goes on, as plan 30's. No key it
  takes reaches the editor, and Enter never sends; Ctrl+Q still quits.
- **D10. What does not change.** No settings file, `config.toml` or
  credential is written or read; no restart; the session id and the
  conversation stay; the draft, its images and the focus stay; `--effort
  ultracode` is not used. A restarted or resumed CLI starts with what its
  own settings give, shown as read.
- **D11. The agent told its settings** (brief amendment 35, 2026-10-24),
  withdrawn: an opt-in line telling the agent its model and effort before
  the first message after a change was built and checked (criteria 22 to
  26 PASS), then removed the same day at the owner's word ("I don't like
  it actually, remove this last implementation"); nothing of it remains in
  the code.

### Criteria

Checked by tests unless marked WT (Windows Terminal screenshots) or live
(the real CLIs with no message sent: no model call).

1. Claude Code's catalogue: from P34c's `initialize` answer, the popup
   lists Default (recommended), Opus 5.5, Fable 5.1, Sonnet 5.5 and Haiku
   5.5 with P34c's descriptions, then More models with the 8 others in
   P34c's order; its heading "CLAUDE MODELS"; no Codex model.
2. Codex's catalogue: a fake app-server answering one model a page with
   P34x's cursors receives 7 `model/list` requests (no cursor, then "1" to
   "6"); the popup lists GPT-6.1-Sol (default), GPT-6-Astra, GPT-6-Sol,
   GPT-6-Luna and GPT-5.6-Sol, then More models with GPT-5.6-Terra and
   GPT-5.6-Luna; its heading "CODEX MODELS"; none of the 3 hidden models;
   no Ultracode row.
3. In force at the start: with P34c's first `get_settings`, Fable 5.1 has
   the check and the band and the pill reads " Fable 5.1 Max · Ultracode ";
   with P34x's thread start, GPT-6-Sol has them and the pill reads
   " GPT-6-Sol High ".
4. A model in force under More models (Opus 4.6) opens the popup with More
   models expanded and its row on screen.
5. Expanding More models changes neither the popup's height nor its width;
   the list scrolls by the wheel, PgUp, PgDn and the cursor.
6. Effort: Fable 5.1 offers Low, Medium, High, XHigh and Max; Opus 4.6
   Low, Medium, High and Max; Haiku 4.5 none (the box "N/A", no chevron,
   a click sends nothing); Codex's GPT-6-Luna Low to Max without Ultra,
   GPT-6-Sol with Ultra, "(default)" after Medium for both.
7. Claude Code's requests as written: a model, `set_model` with the
   entry's value; an effort, `apply_flag_settings` with `effortLevel` and
   the `ultracode` in force; Ultracode, `apply_flag_settings` with
   `ultracode` alone; each answer followed by `get_settings`.
8. Locked effort: with `CLAUDE_CODE_EFFORT_LEVEL` in the reported `env`,
   the box shows "Max" without a chevron, and a click writes nothing to
   the CLI and puts the reason in zone B.
9. Codex's change: after a choice nothing is written to the app-server
   until the next message; that message's `turn/start` carries `model`
   and `effort` (the new model's default when it lacks the effort in
   force); meanwhile the editor row shows "→ GPT-6.1-Sol High (next
   turn)" for GPT-6.1-Sol chosen with High in force; after its answer,
   `thread/read` is sent and its values reach the pill.
10. Ultracode independent of effort: turning it on with Max in force sends
    no effort, and P34c3's read keeps the pill at Max.
11. Ultracode's availability: with `ultracodeAvailable` false (P34c3, Opus
    4.6) the row is dimmed, its toggle off, and a click writes nothing;
    the pill shows Ultracode only while `applied.ultracode` is true.
12. Refusals: P34c's `set_model` error leaves the check, the pill and
    zone B's speaker as they were and shows "Model
    'claude-not-a-model-34' not found" in zone B; an `apply_flag_settings`
    success whose read shows the old value leaves the old value shown and
    says so in zone B.
13. Only after acceptance: between a choice and the read that confirms
    it, the pill and the check show the old values, for both CLIs.
14. Keys: Alt+P opens and closes; ↑↓, Enter, Space, PgUp, PgDn and Esc act
    as D9; Enter with a draft written never sends it (no write command).
15. Mouse: a click on the pill opens; the hover moves the band; clicks
    choose a model, expand More models, open the effort list and pick a
    level, and turn Ultracode on and off; a click outside closes the popup
    and goes on.
16. The draft kept: opening, using and closing the popup leaves the
    draft's text, its attachments and the focus as they were.
17. The session kept: no change restarts the CLI; Claude Code's session id
    and Codex's thread id are the same before and after.
18. No regression: `go build ./...`, `go vet ./...`, `gofmt -l .` empty,
    `go test ./...` all ok; the race detector over input, state, app and
    the adapters, 0 races; the golden frames changed only by the pill's
    real values, each changed frame reviewed.
19. Performance: the timing step 8 of 8 within its budgets, its new
    `TestSelectorToFrame` timing Alt+P and the cursor in the expanded list
    at 120x40 and 208x59 (p95 ≤ 16 ms).
20. Visual (WT, mock, maximised and 120x40): Claude Code's popup and
    Codex's, each collapsed and with More models expanded, match
    `references/11-model-selector.webp` on every point of
    reference-design.md's list: colours, columns and rows, the band, "▶"
    and "✓", the separators, "More models" and its "›", the effort box,
    the toggle, the description "Enables dynamic workflows", the bottom
    edge right under the Ultracode row, no footer, the position over the
    pill.
21. Live, no message: on Claude Code 2.1.303 the dashboard lists the
    owner's 13 models with Fable 5.1 checked and Max locked by the owner's
    variable; a model change and Ultracode on and off are confirmed by
    `get_settings`; on Codex 0.163.0, the 7 models with GPT-6-Sol checked;
    the configuration check PASS before and after.

Added by brief amendment 35 (design D11), then withdrawn with it
(2026-10-24): criteria 22 to 26, on the setting, the line in Claude Code's
and Codex's messages, the draft kept as typed and zone B's grey line, all
PASS before the removal (`doc/plan34-evaluation.md`, the amendment 35
block).

### Steps

1. Research (done 2026-10-23): the findings above, probes P34c, P34c2,
   P34c3 and P34x, this plan.
2. Records: brief amendment 34 (done); reference-design.md's section; the
   research and probe records; AGENTS.md's status.
3. Events and adapters: `event.Models` and `event.Settings`; Claude Code's
   catalogue, `get_settings` reduced, its requests and reads; Codex's paged
   `model/list`, the next turn's model and effort, `thread/read`; the mock
   provider's lists from the probes and its local answers.
4. State: the catalogue, the settings in force, a change asked for, the
   actions and commands.
5. Rendering and input: the pill, the popup, the effort list, the keys and
   the mouse, the details view's sources.
6. Tests: unit, wire (the real adapters over fakes replaying the probes),
   golden frames, race, timing.
7. WT screenshots; the live checks with no message; the evaluation
   (`doc/plan34-evaluation.md`); the report.
8. Amendment 35 (design D11): built, checked, then withdrawn by the owner
   and removed (2026-10-24).

### Progress

- 2026-10-23: step 1 done; step 2 in progress.
- 2026-10-23: steps 2 to 5 done: the records (brief, reference design,
  research, probes); `event.Models` and `event.Settings`; Claude Code's
  catalogue, `get_settings` read reduced to `applied`, `set_model` and
  `apply_flag_settings` with Ultracode carried; Codex's paged `model/list`,
  the next turn's model and effort, `thread/read`, reroutes; the mock's
  local answers and two mock files from P34c and P34x; the state's
  `Models`; the pill, the selector, the effort list, Alt+P, keys, mouse.
  Step 6 in progress: the suite, the new tests and golden frames pass; the
  race detector 0 races (its five 80x24 selector frames failed only because
  they were rewritten during the run: to run again); `TestSelectorToFrame`
  p95 1.1 to 2.7 ms. Step 7 begun: Windows Terminal screenshots (mock,
  maximised 188x53 and 120x40, both CLIs, More models, the effort list,
  Codex's change for the next turn) and the live checks with no message
  (Claude Code 2.1.303: 10 of 10 PASS; Codex 0.163.0: 8 of 8 PASS;
  configuration PASS both).
- 2026-10-23: the race detector over every package, 0 races; the docs
  (ui-design.md, architecture.md, events-and-state.md, reliability.md).
  The independent review (`verifier`) found four defects and six doubts,
  all fixed, each with a test that fails without its fix:
  1. an open selector the frame could not draw (a tall draft at 80x24, a
     turn's status in the pill's place, a full view) still took keys and
     could change the model unseen: it now opens only with room for it
     ("no room for the model selector on this screen"), and a frame that
     cannot draw it, or its effort list, closes it;
  2. Alt+P moved the focus to zone B (design D10 keeps it): no longer;
  3. a `turn/start` Codex refused (a carried change it rejects) left the
     turn "Submitted", only Ctrl+Q left: the turn now ends, the message
     goes back to the editor, and zone B says the change was not taken;
  4. a middle text that just fitted touched "↯ Auto" at 80x24: three
     cells before it now, the four frames back to the last commit's row;
  a. `model/list` followed a cursor that came back without end, and a
     refused page left "has not listed its models yet" for good: a
     repeated cursor or a hundredth page ends the list, a refused page
     gives the toast "no model list: <the CLI's words>";
  b. the effort box and the list's "✓" showed Codex's level asked for:
     both show the level in force, the asked level a grey "…"; a model's
     "✓" stays while a choice of it again waits;
  c. the "▾" left the pill when the box was pushed left: it follows it;
  d. a configured send key closed the popup and sent the draft: it acts
     as Enter there; a paste closes the popup and goes to the draft;
  e. Claude Code changes during a turn (not probed) are not sent, zone B
     saying they wait for the turn's end;
  f. a change waiting when the session ends is dropped, said in zone B.
  Also added: the tests criteria 1, 4, 5, 6, 14 and 16 lacked
  (TestSelectorScroll, TestEffortList, TestSelectorKeepsDraft), Alt+P and
  the selector's keys on the F1 page, and the selector tests on the
  virtual clock (14 s to 0.4 s). The review's second pass confirmed the
  ten fixes and found three small points, fixed: zone B kept a refused
  message's entry with no word (a notice under it now says it was not
  sent and is back in the editor); a second refused choice repeated the
  same notice (no longer); while a turn's status has the editor row alone
  (plan 29, at 120x40 with the longer pills) the pill is off screen, so
  the selector does not open: the toast now says it needs the pill on
  screen, rather than "no room" (plan 29's row is kept).
- 2026-10-23: steps 6 and 7 done on the final code: the suite (623
  top-level tests, 588 on the last commit), the race detector over every
  package (0 races), the timing step alone (8 of 8; the long session timed
  in turn against the last commit, no difference); Windows Terminal
  screenshots again with the final build; the live checks again (Claude
  Code 11 of 11 with every one of the 13 models on screen, Codex 9 of 9,
  configuration PASS). `doc/plan34-evaluation.md`: 20 PASS, 1 FAIL (34.18:
  the golden frames changed beyond the pill's values, explained, no
  regression; an exception for the owner). Waiting for the owner's review;
  nothing committed.
- 2026-10-23: at the owner's request ("the goal is to control it through
  my tool"), `CLAUDE_CODE_EFFORT_LEVEL` removed from the env block of
  `~/.claude/settings.json`, nothing else in the file changed. Live, no
  message: the effort box opens and Low, then Max, are set and confirmed,
  Ultracode kept (11 of 11, configuration PASS; the evaluation's
  follow-up block). New Claude Code sessions now start at the file's
  per-model levels (Fable 5.1 XHigh, Opus 5.5 High), not Max. The owner's
  own session then confirmed it with real turns: Claude Code's transcript
  records `output_config.effort` "max" before the change and "medium" on
  every request from the next message on.
- 2026-10-24: step 8 done (brief amendment 35, design D11): the
  `models.announce` setting (`input.LoadAnnounce`, its startup line and
  problem), the line before the text written of the first message after a
  change (`state/announce.go`; never in the draft; due again after a
  give-back; none before a slash command), zone B's grey line and zone A's
  turn name without it. Criteria 22 to 26 PASS (the evaluation's
  amendment 35 block): 634 top-level tests, the race detector on the four
  changed packages 0 races, no golden frame changed. Waiting for the
  owner's review; nothing committed.
- 2026-10-24: amendment 35 withdrawn by the owner ("I don't like it
  actually, remove this last implementation"): its code, tests and design
  text removed (the files it added deleted, the others back to plan 34's
  code before it); the brief, this plan and the evaluation keep the
  record.

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
