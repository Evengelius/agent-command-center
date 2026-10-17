# Plan 8 evaluation (2026-10-17)

The final check of plan 8, Codex's allow for the session: its one item of
[`plan.md`, Plan 8](../plan.md#plan-8---codexs-allow-for-the-session) against
the yes/no criteria written in the progress list before its work started,
and the plan's rules (one live Codex session approved by the owner, Claude
Code unchanged, the amendment decisions never offered, no new dependency).
The evidence is `plan.md`'s record of the item; the block below is the
summary reported to the owner; the owner's follow-up is at the end. The
results are not to be rewritten.

```text
Evaluation - Plan 8, Codex's allow for the session - 6 criteria: 6 PASS, 0 FAIL
PASS 1  The mapper rule: offered when a command or file-change request carries no availableDecisions (absent or null) or lists acceptForSession; not for L3's list, an empty list, a list without it, or an unreadable request; TestOfferedDecisions covers each case and every recorded request of P6b, P7 and P7b (no list: offered)
PASS 2  The app: on P6b's request as recorded, the hint "s allow for the session" shows, s pressed twice writes {"id":0,"result":{"decision":"acceptForSession"}} once, the card reads "allowed for the session by you", from the card and the full view; with L3's list no hint, nothing written, no refusal note; no frame names an amendment; Claude Code's cards unchanged (CC-4)
PASS 3  Mutants: 3 killed, 0 survived (the list required again: 3 tests fail; offered whenever the request is readable: 2; the unreadable request ignored: 1)
PASS 4  Live, Codex 0.163.0, one session (temporary project acc-p8codex-231009): 15 of 15 harness checks, 3 of 3 on quit; one file-change approval in a two-patch turn, s answered it, "✓ orchestrator · fileChange  allowed for the session by you"; the second patch asked for nothing; note.txt "one", "two"; frame closed and card shown at 156 columns maximised and at 120 by 30; Ctrl+Q 0.21 s, 0 of 2 CLI processes left; configuration check PASS before and after
PASS 5  Documents: doc/ui-design.md (the s row, Departures), doc/events-and-state.md, doc/research/codex-cli.md (question 4, the 0.163.0 schema quotes), doc/reliability.md; doc/architecture.md already right; code comments in mapper.go, keys.go, session.go, state.go
PASS 6  go build ./... and go vet ./... (also -tags live, -tags timing) exit 0; gofmt -l . prints nothing; go test ./... 231 top-level tests PASS in 14 packages (231 at bffe9a5, new cases are subtests), 0 FAIL; the changed files have no CR and no BOM; go.mod and go.sum unchanged, 4 direct modules
Mutation checks: 3 mutants killed, 0 survived.
Not verified: a live command-execution approval answered s (the turn asked only for a file change); a live request carrying availableDecisions (0.163.0 sends none); the timing step (no input or stream path changed). Each with its reason below.
Verdict: DONE
```

## Not verified

What no check of this plan settles, and why. None of it is reported as PASS.

- **A live command-execution approval answered `s`.** The live turn was
  written to change one file twice and run no command, so it asked for one
  file-change approval only. No recording holds a command approval request
  (P6b's is a file change), so the command case is covered by the mapper
  test's synthetic request alone (`TestOfferedDecisions`, "command, no
  list"); past the mapper, both kinds take the same field and key path.
  The generated schema documents `acceptForSession` for both.
- **A live request carrying `availableDecisions`.** Codex 0.163.0 sends no
  list; L3's list (Codex 0.145.0) and a synthetic list holding
  `acceptForSession` are covered by the deterministic tests only.
- **The keystroke timing step.** Plan 8 changed which approvals offer `s`,
  not the input, the stream or the frame loop; the step was not run.

## Failures on record

None. The live run passed on its first attempt; no criterion was amended.

## Criteria amended

None.

## Follow-up (2026-10-17)

The owner asked for the three items above to be tested ("can we test it so
it becoems verified?"). The block below is the summary reported after it;
the one above stays as it was reported. Evidence: `plan.md`, item 1's
Follow-up.

```text
Evaluation - Plan 8 follow-up, the items not verified - 3 criteria: 3 PASS, 0 FAIL (one earlier run, 5 FAIL, on record)
PASS 1  The recorded command request: probe P8 (Codex 0.163.0, the dashboard's handshake, acc-p8probe-232303) recorded two escalated command requests carrying availableDecisions ["accept",{"acceptWithExecpolicyAmendment":...},"cancel"]; TestOfferedDecisions holds one as recorded (cwd replaced): offered accept, cancel; no session answer; the experimental field and Codex's default list confirmed in openai/codex 0.145.0 (item.rs, approvals.rs)
PASS 4c Live command case (acc-p8cmd2-232435, fully scripted): 15 of 15 harness checks, 3 of 3 on quit; 2 command approvals, each card "▸ a allow once · d deny · c cancel the turn" without s; s left each pending; a answered each, "allowed by you"; no refusal note; note.txt "one", "one"; frame closed and answered card shown at 156 columns maximised and at 120 by 30; Ctrl+Q 0.21 s, 0 of 2 CLI processes left; configuration check PASS for P8, run 1 and run 2
PASS T  Timing step, 4 of 4: keystroke-to-frame p95 1.06 ms (10,000-line draft) and 2.48 ms (50,000 lines, 8 saves); delta-to-frame p95 3.14 ms after a two-hour session (26,209 events); all under 16 ms
Documents corrected where they said 0.163.0 sends no list: doc/research/codex-cli.md (question 4), doc/events-and-state.md, doc/ui-design.md, mapper.go and session_allow_test.go comments.
Full check: go build and go vet (also -tags live, -tags timing) exit 0; gofmt -l . prints nothing; go test ./... 231 top-level PASS in 14 packages, 0 FAIL; go.mod and go.sum unchanged.
Not verified: a live request whose list holds acceptForSession (only a network approval gets one; none exercised).
Verdict: DONE
```

### Not verified

- **A live request whose list holds `acceptForSession`.** Codex adds it only
  to a network approval (`default_available_decisions`), which no run
  exercised. The case is covered by `TestOfferedDecisions` (a synthetic
  list holding it) and `TestCodexSessionAllow` ("listed").

Of the three items above: a command answered `s` cannot occur on Codex
0.163.0, which does not offer it for an ordinary command; run 2 verified
the dashboard follows that. The live request with a list is verified (P8,
run 2). The timing step passed.

### Failures on record

- **Command run 1** (`acc-p8cmd-231916`, 23:19 to 23:20): 5 FAIL of 15
  (the first approval offers `s`; the card reads "allowed for the session
  by you"; the second command asks for nothing; the answered card at each
  size). Causes: the criterion expected Codex to offer `acceptForSession`
  on a command, which its list does not; and the owner, not told the run
  was scripted, pressed `a` on the first card. Both cards showed no `s`
  before any key was pressed. Fixed by reading Codex's source and probe P8,
  then run 2, fully scripted, with the corrected expectation (4c). Quit 3 of
  3 PASS; configuration check PASS.

### Criteria amended

- **Criterion 1** said 0.163.0 sends no list; amended: a file-change
  request sends none, a command request lists accept, the amendment and
  cancel (P8), and the mapper test holds that recorded request.
- **Criterion 4** gains the command case, 4c: no `s` on a command card,
  `s` leaves it pending, `a` answers it. Not weakened: it records what
  Codex offers, from its source and a recording, and still requires the
  file-change case's 15 of 15.
