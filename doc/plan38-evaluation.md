# Plan 38 evaluation (2026-10-24)

The check of plan 38, instant Codex model and effort changes as in VS Code
(brief amendment 39: a Codex change takes effect at once, as Claude
Code's, through Codex's experimental `thread/settings/update`, as OpenAI's
Codex extension for VS Code sends it; one or two live Codex turns
approved). It covers the 12 criteria written in
[`plan.md`, Plan 38](../plan.md) before the code; two were amended during
step 2, each said where it applies (38.7, 38.9). Nothing is committed: the
owner reviews the result first.

How it was checked:
- The probe first (plan.md, What the probe found): P38x, no turn, and
  P38t, the first approved turn, both through
  `doc/research/probes/drivers/p38-codex-settings.mjs` on Codex CLI
  0.163.0 with the dashboard's handshake and the opt-in, each in a new
  temporary project with the configuration check before and after;
  OpenAI's Codex extension for VS Code (`openai.chatgpt` 26.1021.22027,
  installed here) read for what it sends.
- Deterministic tests, none starting a CLI: the Codex adapter
  (TestSettingsUpdateWire, TestSettingsDuringTurn, TestSettingsBackstop,
  TestSettingsRefused, TestCurrentTime, TestUpdateRefusals,
  TestSettingsUpdatedAnswers, TestP38Replay; TestModelChangeWire, plan
  34's next-turn wire, now through a Codex that refuses the update;
  TestEveryLineMaps with P34x, P38x and P38t added), the state
  (TestCodexAtOnce, TestCodexDuringTurn, TestCodexFallback,
  TestClaudeSameStillAsks), the application on the real Codex adapter
  over a wire fake (TestCodexSliderAtOnce), the mock (TestMockCodexChanges,
  TestMockFiles with `codex-models-next-turn.jsonl`). Each new check was
  run once against code broken on purpose, 16 breakages in all, and each
  failed: no opt-in; never written at once; the notification never
  answering; no hold during a turn; no write at the turn's end; no
  fallback; no backstop; the time not answered; a read answering after
  the notification; the moderation metadata unknown; Codex's choice of
  what is in force asked; no effort with a Codex model at once; the
  next-turn event ignored; the mock always at once; the application never
  at once; a refused change still carried.
- Windows Terminal 1.23 in mock mode, the build of the final tree
  (`scratchpad\term\acc-p38.exe`, SHA-256 `fb82d4c2e9946cef…`, the same
  bytes as a build of the final tree), driven by `scratchpad\term\p35demo.ps1`,
  maximised (156x44) and at 120x40: `codex-models.jsonl` and
  `codex-models-next-turn.jsonl`.
- Live in Windows Terminal, the second approved turn: the same build on
  Codex CLI 0.163.0 through the test-only logging wrapper,
  `scratchpad\term\p38live.ps1`, in a new temporary project with the
  configuration check before and after.
- Build, vet, gofmt, the whole suite, the race detector on codex, state,
  app and mock, the timing step alone, all on the final tree; no
  independent review, the criteria being settled by the tests, the 16
  mutation checks, the screenshots and the live run.

```text
Evaluation - Plan 38, instant Codex model and effort changes, as in VS Code - 12 criteria: 12 PASS
PASS 38.1 P38x recorded and sanitized, 40 lines, in doc/research/probes and testdata/probes (no user, home or host name; nothing of the proxy): thread/settings/update {effort: "low"} answered {} 2 ms after it, thread/settings/updated with effort "low" 3 ms after it; the model alone (gpt-6.1-sol) kept "low"; the update to the effort in force ("xhigh") brought no notification in 1.5 s; the thread/read written right after the {} showed "low"; the configuration check PASS before and after (0 new session files)
PASS 38.2 TestSettingsUpdateWire: the initialize line {"id":1,"method":"initialize","params":{"capabilities":{"experimentalApi":true},"clientInfo":{...}}}; between turns SetModelSettings({Effort: "low"}) wrote thread/settings/update {"threadId":"th-38","effort":"low"} alone and no turn/start; the {} a status "settings update acknowledged", no settings event before the answer; P38x's notification gave {Model gpt-6-sol, Choice gpt-6-sol, Effort low, EffortKnown, Source thread/settings/updated, Answers}, then in force; TestSettingsUpdatedAnswers: with no change waiting Answers false, another thread's no settings event, a second notification Answers false, a read after the notification no answer; TestP38Replay: P38x's and P38t's settings in order, the turn/start's own notification a report
PASS 38.3 TestSettingsDuringTurn: ModelsNextTurn true from the turn/start written and during the turn; a change asked then wrote nothing; after turn/interrupt and turn/completed one thread/settings/update {"threadId":"th-38","model":"gpt-6-luna","effort":"low"}, answered by its notification; ModelsNextTurn false after it
PASS 38.4 TestUpdateRefusals: -32600 "thread/settings/update requires experimentalApi capability", -32601 "Method not found" and "Invalid request: unknown variant `thread/settings/update`, ..." each gave {Source thread/settings/update, NextTurn, Text "Codex takes model changes with the next message: thread/settings/update refused by Codex: <Codex's words>"}; another method's unknown variant not taken for this one; -32602 gave the error and {Answers, Refused, "the model change was not taken: thread/settings/update refused by Codex: invalid params"}; TestModelChangeWire: after P34x's refusal, ModelsNextTurn true, the next turn/start carried the model and the effort, thread/read answered; TestSettingsRefused: after -32602 the next turn/start carried no model or effort
PASS 38.5 TestSettingsBackstop: an update answered {} with no notification; the next turn/start carried "effort":"low"; its thread/read answered (Source thread/read, Answers)
PASS 38.6 TestCurrentTime: the reply {"id":0,"result":{"currentTimeAt":1792829210}} for a clock at 1792829210.6 s; the request a status "current time asked", the reply a Sent "currentTime/read reply"; no unrecognised event, no approval asked or resolved
PASS 38.7 TestCodexAtOnce: GPT-6-Luna asked once at its default "medium" (no Ultra), no label; a second choice before the answer asked nothing; GPT-6-Sol with "medium", in force, then Max; each answer in force with no notice; the effort and the model in force asked nothing; TestCodexDuringTurn: a choice at once during a turn asked nothing, "Codex's model settings change between turns: wait for the turn to end"; one with the next turn "→ GPT-6-Sol Low (next turn)", in force after its answer; TestCodexFallback: the next-turn event turned the waiting change into {Effort max, NextTurn}, "→ GPT-6-Sol Max (next turn)", one notice in Codex's words, the settings in force unchanged; amended in step 2: Claude Code's choice of what is in force still asks (TestClaudeSameStillAsks; plan 36's TestSelectorPointerAtRest passes), TestChoiceDuringTurn's Claude Code notice unchanged
PASS 38.8 TestCodexSliderAtOnce on the real Codex adapter: → on Effort wrote one thread/settings/update {"threadId":"th-38","effort":"xhigh"}; the Effort row "Effort (XHigh)" with no "○", the pill "GPT-6-Sol XHigh", no "(next turn)", nothing asked any more, no turn/start; with a Codex refusing the update: asked {Effort xhigh, NextTurn}, the row "Effort (High)" with "○", "→ GPT-6-Sol XHigh (next turn)", zone B "Codex takes model changes with the next message: ..."; a second → asked Max with no update written
PASS 38.9 build, vet, gofmt clean; go test ./... ok, 15 packages with tests, 649 tests; -race on codex, state, app and mock ok, 0 data races; golden frames: amended in step 2, only the Explorer's and the quick open's 6, which list testdata/mock, its new file and 17 items in place of 16; the timing step alone: TestSelectorToFrame p95 1.03 to 1.56 ms (target 16 ms), the seven others within target; TestEveryLineMaps with P34x, P38x and P38t: no unrecognised line
PASS 38.10 P38t (the first approved turn), 73 lines: after updates to GPT-6-Luna at "low", one turn whose rollout's one turn_context reads gpt-6-luna and "low" (its model and effort alone read); the command approval (availableDecisions accept, the execpolicy amendment, cancel; no additionalPermissions) answered acceptForSession: the command ran (probe38.txt "ok") and the turn completed in 5.8 s; no currentTime/read came; the configuration check PASS before and after (1 new session file)
PASS 38.11 Live in WT (the second approved turn), Codex 0.163.0 through the wrapper: L38.1 no terminal CLI session, Px listening, the version, the snapshot; L38.2 Ready, initialize with "capabilities":{"experimentalApi":true}; L38.3 Alt+P, End, ←: "Effort (XHigh)" to "Effort (High)" on screen 120 ms after the key, the pill "GPT-6-Sol High", no "(next turn)", no ring (03-changed.png), the wrapper: the update {"effort":"high"} written, its {} in the same millisecond, thread/settings/updated 1 ms later, no turn/start before the message; L38.4 "Reply with the single word ready.": the turn completed, "ready", its turn/start with no model or effort, its rollout's one turn_context gpt-6-sol and "high"; L38.5 the configuration check PASS
PASS 38.12 WT on the mocks, maximised (156x44) and 120x40: codex-models.jsonl, GPT-6-Sol from High through XHigh, Max, Ultra and back down to Low by → and ←, each label and the pill at the level at once, no "(next turn)", the slider's image with no ring (c-xhigh, c-ultra); codex-models-next-turn.jsonl: zone B "Codex takes model changes with the next message: thread/settings/update refused by Codex: thread/settings/update requires experimentalApi capability", → twice: "Effort (High)" with the ring on Max, "→ GPT-6-Sol Max (next turn)"; after "go", Max in force
```

Not verified:
- What `thread/settings/update` does to a running turn: the dashboard
  writes none during one (design D3).
- A `currentTime/read` from a live Codex: none came in P38t or the live
  run; the answer is checked by TestCurrentTime alone.
- A live Codex that lacks the method or refuses the opt-in: the fallback
  is checked with P34x's real refusal (-32600) and scripted ones.
- The live gate (`TestLiveGate`) with the opt-in: its several turns
  exceed the approved quota; approvals with the opt-in were checked by
  P38t's one command approval.
- `additionalPermissions` on a command approval, which no probe received:
  the approval card does not show it.
- Other terminals than Windows Terminal 1.23.

Quota used: two Codex turns, as approved: P38t, GPT-6-Luna at "low",
39,923 tokens, the thread's last total, of which 27,776 cached input;
the live run, GPT-6-Sol at "high", 16,334 tokens, of which 12,544 cached
input.

Evidence outside the repository, in the session's scratchpad
(`C:\Users\amolimom\AppData\Local\Temp\claude\C--javadev-projets-agent-command-center\dcc3e64d-0d35-4470-ab9a-d258f8675bae\scratchpad\`):
`plan38\` (the schemas with and without `--experimental` and
`schemadiff.mjs`, the probes' configuration snapshots and hashes,
`mut38.mjs` and its results `mut38.txt`, the doc scripts),
`term\out-p38\` (the mock runs, with their step plans),
`term\out-p38-live\` (the live run's results, screenshots and
configuration check), `live\log-p38\` (the wrapper's log),
`term\p38live.ps1`.

**The commit and the push (2026-10-24), after the owner's approval.** The
owner approved plan 38 on 2026-10-24; commit dc9c6d9 (38 files: 28
modified, 10 added) and the approval commit were pushed to `origin`;
`origin/main` equals `main`, the working tree is clean, and the author is
`Evengelius <molimoabel@gmail.com>`, with no attribution line: PASS. One
comment of `SetModelSettings` was reworded before the commit, after the
checks: the build's bytes are unchanged (`fb82d4c2e9946cef…`) and the
Codex package's tests pass again. The blocks above stay as they were
reported.
