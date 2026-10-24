# Plan 41 evaluation (2026-10-24)

The check of plan 41, no authorization on low-risk review rows (brief
amendment 42: a Codex auto-review row rated low risk no longer shows the
reviewer's authorization, which after the reviewer's short answer is a
stock `unknown`; at medium, high and critical it stays). It covers the 7
criteria written in [`plan.md`, Plan 41](../plan.md) before the code; one
was amended in step 3, said where it applies (41.5). Nothing is
committed: the owner reviews the screenshots and the results first.

How it was checked:
- The cause, from the code, the recordings and Codex CLI 0.163.0's
  binary (plan.md, What was found): Codex's reviewer answers a low-risk
  action with `{"outcome":"allow"}` alone, and Codex then reports the
  authorization `unknown` with the stock rationale "Auto-review returned a
  low-risk allow decision.".
- Deterministic tests, none starting a CLI: the text (TestReviewAuthorization,
  17 combinations of verdict, risk, authorization and rationale, and a
  review in progress; TestReviewEntries and the P12x replays updated to
  the new low-risk text), the owner's case (TestReviewRowsOwnerCase: P7b
  replayed through the Codex mapper with P12x's two review notifications
  moved onto each of the four subagent commands, the completion carrying
  the owner's values), zone B (TestReviewRowsZoneB on P12x as recorded),
  medium and above (TestReviewRowsMediumAndUp), the application
  (TestReviewerInZoneB) and the golden frames. Each was run against the
  rule broken on purpose (`scratchpad\plan41\mut41.mjs`), 5 breakages.
- Windows Terminal 1.23 in mock mode: the owner's case as a scratch mock
  (`scratchpad\plan41pre\p41-codex-subagent-reviews.jsonl`, the same
  construction as the test, 235 events) and P12x as one
  (`scratchpad\plan41\p41-codex-zone-b-reviews.jsonl`, 178 events), each
  encoded as TestMockFiles does through a temporary test outside the
  repository; plan 40's build (`scratchpad\term\acc-p41-before.exe`,
  SHA-256 `6160e71fdf2568cc…`, HEAD's code) and the new tree's
  (`scratchpad\term\acc-p41.exe`, SHA-256 `e2fd5e4bb76d4494…`), driven by
  `scratchpad\term\p41demo.ps1` (plan 40's harness, with a Windows
  Terminal profile and a crop of zone B), maximised in the acc tab's
  profile (9 pt, 208x59) and at 120x40.
- Build, vet, gofmt, the whole suite, the race detector on state, render
  and app, the timing step alone, all on the final tree; no independent
  review, the criteria being settled by the tests, the mutation checks and
  the screenshots.

```text
Evaluation - Plan 41, no authorization on low-risk review rows - 7 criteria: 7 PASS
PASS 41.1 TestReviewAuthorization: risk low with authorization unknown, low, medium and high (approved), and with unknown denied, timed out and aborted: "auto-review <verdict> · risk low: <rationale>", no "authorization" (the owner's row: "auto-review approved · risk low: Auto-review returned a low-risk allow decision."); medium (unknown, high), high (medium, low) and critical (high): "risk <level> · authorization <value>: fine"; no risk reported: "auto-review approved · authorization high: fine"; a risk word outside the schema ("elevated"): kept; no rationale: "auto-review approved · risk low", "auto-review approved · risk medium · authorization high", "auto-review reported · risk low"; in progress: "auto-review in progress: go vet ./..."; TestReviewEntries ("auto-review approved · risk low: fine", denied, timed out, aborted) and TestProbeCodexReviews (both P12x recordings, 3 reviews: the prefix "auto-review approved · risk low: ", no "authorization") updated
PASS 41.2 TestReviewRowsOwnerCase: /root/list_files 1 review and /root/create_found 3, each entry exactly "auto-review approved · risk low: Auto-review returned a low-risk allow decision."; the timeline at 160 cells one row each, no "authorization"; zone C in the frame at 120x40 and 200x60 with each child selected: the review rows "review    │ auto-review approved · risk low: Auto-review", no "authorization" anywhere in zone C; the inspection's Messages tab: 1 and 3 rows with that text, no "authorization"
PASS 41.3 TestReviewRowsZoneB: P12x's two review entries exactly "auto-review approved · risk low: Creates the explicitly requested hello.txt with the word hello in the current directory and reads it back; this is a routine, reversible local file operation." and "auto-review approved · risk low: Creates the specifically requested single local file in the parent directory and verifies it; the narrowly scoped operation has low risk."; zone B at 120x40 and 200x60: "◈ auto-review approved · risk low: Creates the", no "authorization"; TestReviewerInZoneB: "auto-review approved · risk low: a harmless write", updated in place
PASS 41.4 TestReviewRowsMediumAndUp: the orchestrator's and a subagent's reviews at medium (high), high (medium) and critical (low, denied): zone B "◈ auto-review <verdict> · risk <level> · authorization <value>: a check", zone C's timeline "review    │ auto-review <verdict> · risk <level> · authorization <value>: a check", all six
PASS 41.5 (amended in step 3) only the three approvals-review frames change, 18 lines: at 200x60 the review's row alone ("◈ auto-review approved · risk low: a read-only check of the project", "· authorization high" gone); at 120x40 and 80x24 the row takes one line instead of two, so zone B, drawn from its tail, shows one row more above it: every other row of B the old row above it, zone A's column unchanged, at 80x24 the new top row a thinking row, at 120x40 "▎ word done." with the rule under it now drawn (blank before, at the view's edge: amendment 17, 2.5); the other 189 frames unchanged
PASS 41.6 Build, vet, gofmt clean; go test ./... ok, 15 packages with tests, 687 tests (4 new); -race on state, render and app ok, 0 data races; the timing step alone, all eight within target: the long session's delta p95 5.52 ms, the stream's 4.26 ms, the thinking cards' 5.02 ms, the selector's 1.01 to 1.54 ms, the 50,000-line paste 2.20 ms at most (targets 16 ms, 100 ms); 5 breakages of the rule, each caught: removed (state, render and app: TestReviewEntries, TestReviewAuthorization, TestReviewRowsOwnerCase, TestReviewRowsZoneB, TestReviewerInZoneB, TestGoldenFrames), widened to medium (TestReviewAuthorization, TestReviewRowsMediumAndUp), narrowed to unknown alone (TestReviewEntries, TestReviewAuthorization, TestReviewRowsZoneB, TestReviewerInZoneB, TestGoldenFrames), dropped at every risk (TestReviewAuthorization, TestReviewRowsMediumAndUp), dropped with no risk reported (TestReviewAuthorization)
PASS 41.7 Windows Terminal, maximised in the acc tab's profile (208x59) and at 120x40: the owner's case, plan 40's build "auto-review approved · risk low · authorization unknown: Auto-review returned a low-risk allow decision." on every review row (zone C: /root/create_found 3 rows at 208x59, 2 in view at 120x40, /root/list_files 1; the inspection 3 and 1), the new build the same rows with "· authorization unknown" gone and nothing else changed; P12x, zone B: plan 40's build "◈ auto-review approved · risk low · authorization high: Creates the …" (2 rows at 208x59, 1 in view at 120x40), the new build "◈ auto-review approved · risk low: Creates the …"
```

Observed and kept:
- The stock rationale stays word for word ("Auto-review returned a low-risk
  allow decision."), though "approved · risk low" says the same: it is
  Codex's text.
- A full answer at low risk loses its authorization too (P12x's `high`);
  its rationale says it in words ("explicitly requested", "specifically
  requested").
- The rule reads Codex's risk word exactly (`low`, the schema's spelling);
  a word outside the schema keeps the authorization.

Not verified:
- A live Codex run with an auto-review: it needs a model call on the
  owner's quota. The rows are built from recorded notifications (P12x,
  Codex CLI 0.163.0) and, for the owner's case, from P12x's notifications
  carrying the values of the owner's screenshot.
- A review at medium, high or critical risk from Codex itself: none was
  ever recorded; the rows are checked with hand-made events (41.4).
