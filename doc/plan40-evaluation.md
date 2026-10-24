# Plan 40 evaluation (2026-10-24)

The check of plan 40, no empty rows in zone C's execution timeline (brief
amendment 41: on Codex, the owner's demo scenario 1 showed a subagent's
rows "assistant │" with no text). It covers the 6 criteria written in
[`plan.md`, Plan 40](../plan.md) before the code; design D1 was widened
while writing the fix, said where it applies (40.2). Nothing is
committed: the owner reviews the screenshots and the results first.

How it was checked:
- The cause, from the code and the recordings (plan.md, What was found):
  a Codex subagent's reasoning summary deltas, asked for since plan 22,
  give it an assistant entry holding only a thinking mark; zone B never
  draws one, zone C's timeline and the inspection's Messages tab drew each
  as a row with no text.
- Deterministic tests, none starting a CLI: TestTimelineNoEmptyRows
  replays P7b (Codex CLI 0.163.0, two subagents) through the real Codex
  mapper with P22s's reasoning item (`item/started`, `summaryPartAdded`,
  `summaryTextDelta`, `item/completed`, Codex CLI 0.163.0) after each of
  the first child's two reasoning items, on its thread and turn, and
  checks the timeline alone (60 cells), zone C in the drawn frame at
  120x40 and 200x60 with that child selected, its inspection's Messages
  tab, and the child with nothing but its thinking;
  TestTimelineNoTextEntries a hand-made agent with a message of spaces, a
  notice with no text and a message that ended empty. Both were run
  against the code before the fix, then against each part of the fix
  broken on purpose (`scratchpad\plan40\mut40.mjs`), 8 breakages.
- Windows Terminal 1.23 in mock mode: the reproduction's records encoded
  as a scratch mock (`scratchpad\plan40\p40-codex-subagent-thinking.jsonl`,
  233 events from 223 records, by the encoder TestMockFiles uses, through
  a temporary test removed afterwards; not added to `testdata\mock`),
  plan 39's build (`scratchpad\term\acc-p39.exe`, SHA-256
  `87b5578d9a206665…`, the code before the fix) and the fixed tree's
  (`scratchpad\term\acc-p40.exe`, SHA-256 `01def79fa23d6612…`), driven by
  `scratchpad\term\p40demo.ps1`, maximised (156x44) and at 120x40: the
  replay's end, F6 into zone C, each child selected with ↓, each one's
  inspection opened with Enter and closed with Esc.
- Build, vet, gofmt, the whole suite, the race detector on render, the
  timing step alone, all on the final tree; no independent review, the
  criteria being settled by the tests, the mutation checks and the
  screenshots.

```text
Evaluation - Plan 40, no empty rows in zone C's execution timeline - 6 criteria: 6 PASS
PASS 40.1 The reproduction gives the first child (/root/list_files) 2 entries holding only a thinking mark (origin ai, no text) and 2 with text; on the code before the fix TestTimelineNoEmptyRows failed: the timeline 4 entry rows for 2, "02:00:09  assistant │" and "02:00:12  assistant │" with no text; the frame at 120x40 3 entry rows, one with no text, a rule at its top (the pane held at its tail), at 200x60 4 entry rows, 2 with no text; the inspection 6 timed rows for 4, "02:00:09" and "02:00:12" with no text; with only thinking, two empty rows and no "no activity yet" nor "no message from this agent yet"; TestTimelineNoTextEntries 5 entry rows for 2 and 6 timed rows for 3
PASS 40.2 After the fix (D1 widened from assistant entries to every entry with no text, as 40.2 reads: Claude Code's mapper passes a subagent's empty text block on as a notice with none): the timeline and the frame at 120x40 and 200x60 draw exactly the two entries with text, "02:00:09  assistant │ I’ll list the current directory and" (its second row "send the filenames back.") and "02:00:13  assistant │ No files found.", in order with their times, one rule between them, none at the top, at the end or twice in a row; the child with only thinking reads "no activity yet"; the hand-made agent's notice "Working directory: /tmp/p40" and message "Found two files." alone, one rule between
PASS 40.3 The inspection's Messages tab: the start, the two messages and the end, 4 timed rows, none with no text or with only an origin label; with only thinking, "no message from this agent yet"; the hand-made agent's start and its two entries with text, 3 timed rows
PASS 40.4 Zone B's code unchanged (the diff: render's look.go, zonec.go and inspect.go only); no mock's agent has an entry with no text (12 agents in 8 mocks, Claude Code's thinking subagent's 4 entries included), so no mock's timeline changes; TestGoldenFrames without -update: the 192 golden frames unchanged, git status on testdata empty; TestZoneCDesign (the mocks' timelines, the hand-made entries' rows) and every other render test pass
PASS 40.5 Build, vet, gofmt clean; go test ./... ok, 15 packages with tests, 683 tests (2 new); -race on render ok, 0 data races; the timing step alone, all eight within target: a first run's long-session delta p95 13.43 ms, then the old and new trees in turn, the long session 6.00 and 6.15 ms against 5.51 and 5.53 ms, the stream 5.02 and 4.54 against 4.26 and 4.11 ms, the thinking cards 4.27 and 3.92 against 4.85 and 4.09 ms, the selector 1.01 to 1.72 against 1.02 to 1.60 ms, the 50,000-line paste 2.79 and 2.56 against 1.79 and 2.50 ms at most (targets 16 ms, 100 ms): the first run's p95 the machine's load, not the fix; the new checks against the whole fix removed fail (40.1), and 8 breakages of the fix, each caught: the timeline drawing entries with no text, a rule before an entry left out, the inspection drawing entries with no text, "no message from this agent yet" only with no entry at all, noText reduced to thinking marks, keeping a text of spaces, or limited to assistant entries
PASS 40.6 Windows Terminal, scratch mock, maximised (156x44) and 120x40: plan 39's build shows /root/list_files with "15:16:46  assistant │" and "15:16:49  assistant │" with no text between its two messages (maximised; at 120x40 the one in view, the pane at its tail), and its inspection timed rows with nothing but "mock"; the fixed build shows its two messages alone, one rule between, and the inspection's start, two messages and end; /root/create_found the same in both
```

Observed and kept:
- In the 120x40 frame, a timeline held at its tail can begin with a rule
  (plan 27's pane): plan 39's build showed one above its first row in
  view; after the fix the reproduction's two messages fit and none shows.
- The text capture (UI Automation) cuts the inspection's "mock" to "moc"
  on rows holding "’" or "✓", in both builds; the screenshots show it
  whole.
- Zone B is unchanged (D2): an orchestrator notice with no text, never
  seen, would still draw its row there.

Not verified:
- A live Codex run with a subagent, the owner's demo scenario 1: it needs
  a model call on the owner's quota. The reproduction replays recorded
  notifications (P7b's subagents and P22s's summaries, both Codex CLI
  0.163.0, the last probed version).
- Claude Code's empty text block becoming a notice with no text: never
  recorded; checked by the hand-made test alone.
