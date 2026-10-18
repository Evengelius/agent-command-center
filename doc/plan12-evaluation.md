# Plan 12 evaluation (2026-10-18)

The final check of plan 12, Manual or Auto approvals: its four items of
[`plan.md`, Plan 12](../plan.md#plan-12---manual-or-auto-approvals) against
the yes/no criteria written in the progress list before each item's work
started, and the plan's rules (brief amendment 8; the owner's quota,
temporary projects outside the repository, the configuration check before
and after every live run, no configuration file or setting changed by the
work, no new dependency). The evidence is `plan.md`'s record of each item;
the block below is the summary reported to the owner. The results are not
to be rewritten.

```text
Evaluation - Plan 12, Manual or Auto approvals - 19 criteria: 18 PASS, 1 FAIL (accepted)
PASS 1.1 P12c: Claude Code accepts --permission-mode auto (system/init permissionMode "auto"); both writes ran with no can_use_tool; the classifier's mark is tool_result_meta[].permission_decision reason_type "classifier" (undocumented)
PASS 1.2 P12c-switch: set_permission_mode auto answered success in 4 ms, then system/status permissionMode "auto"; the next write asked nothing
PASS 1.3 P12x: Codex approvalsReviewer "auto_review" echoed by thread/start; no requestApproval reached the client; item/autoApprovalReview/started and /completed per escalation ([UNSTABLE])
PASS 1.4 P12x-switch: turn 1 with "user" sent a requestApproval, turn 2 with "auto_review" none; the reviewer is echoed on neither turn/start nor turn/started
PASS 2.1 Manual at every launch, saved nowhere: --permission-mode default kept; approvalsReviewer "user" on thread/start and thread/resume, the reviewer in force on every turn/start (session tests on the bytes)
PASS 2.2 Session.SetApprovalMode and Capabilities.AutoApprovals: Claude Code writes set_permission_mode at once, Codex on the next turn/start, the mock refuses
PASS 2.3 The mode shown is the one the CLI reports (Claude Code) or the dashboard last wrote (Codex); a refusal is a notice with the mode unchanged
PASS 2.4 F4 toggles; the project bar always shows "approvals Manual|Auto", "→ Auto …" or "(next turn)" while asked, never dropped for width
PASS 2.5 F4 refused with a notice when not ready or not capable; F1 lists it; the hints line at wide width
PASS 2.6 Claude Code: only classifier decisions mark a tool line "approved/denied by auto-review"; Codex: one zone B entry per review, the inspector [UNSTABLE]; guardianWarning kept out of zone B
PASS 2.7 Escalations stay cards in Auto (replay through the reducer)
FAIL 2.8 The four recordings in testdata/probes, replayed, 0 unrecognised events, 12 new golden frames; but the existing 200x60 frames changed beyond the project bar (the hints line gains "F4 Manual/Auto", required by 2.5); accepted
PASS 2.9 internal/arch TestSeparation passes; go.mod and go.sum unchanged
PASS 3.1 Claude Code 2.1.301 (Haiku) live, 156x44 and 120x30: Manual at launch; a write asked a card, denied, no file; F4 Auto; a project write ran with no card (reason_type "mode", so no mark, correctly); F4 Manual, the next write a card, denied; Ctrl+Q 3.10 s, 0 processes left; with an outside write, zone B read "PowerShell · approved by auto-review" and "Write · approved by auto-review"
PASS 3.2 Codex 0.163.0 (gpt-5.6-luna) live, 156x44 and 120x30: Manual at launch; a write asked a card, denied; F4 "Manual → Auto (next turn)", then Auto once sent; a write ran with no card and zone B read "auto-review approved · risk low · authorization unknown: ..."; F4 back, the next write a card, denied; reviewers sent user, user, auto_review, user; Ctrl+Q 2.21 s, 0 left
PASS 4.1 Documents: doc/architecture.md (the interface, the capability, both adapters), doc/ui-design.md (project bar, F4, Manual or Auto), doc/research claude-code.md and codex-cli.md section 11, probes/README.md, brief amendment 8
PASS 4.2 go build ./... and go vet ./... (also -tags live, -tags timing) exit 0; gofmt -l . prints nothing; go test ./... 258 top-level tests PASS in 14 packages, 0 FAIL (baseline 235)
PASS 4.3 The timing step run alone: TestKeystrokeToFrame p95 1.01 ms, TestDraftKeystrokes p95 2.53 ms, TestKeystrokeCost, TestLongSession p95 3.12 ms, all PASS (target p95 <= 16 ms)
PASS 4.4 81 changed or new files: no CR, no BOM; the temporary probe and live project folders deleted; configuration check after every probe and live run PASS (configuration files identical; ~/.claude.json changed only in Claude Code's cache-state keys)
Not verified: a classifier denial ("denied by auto-review") or a Claude Code escalation in Auto; a denied, timed-out or aborted Codex review; a Codex escalation to the owner in Auto. None was observed in the probes or the live runs; replay tests cover them with built sequences.
Verdict: DONE, with 1 accepted exception
```

## Failures on record

- **Item 2, criterion 8.** The criterion asked that existing golden frames
  change only in the project bar. The three 200x60 frames' hints line also
  gains "F4 Manual/Auto", which criterion 5 requires; at 120x40 the
  project path is elided sooner. The 12-line diff was reviewed, nothing
  else changed. Accepted as a documented exception.
- **Item 3, a harness check.** The first Claude Code live run's check "the
  reviewer mark `approved by auto-review` in zone B" printed FAIL. Claude
  Code reported that project write as `{decision:"accept",
  source:"config", reason_type:"mode"}`: no classifier ran, so the
  dashboard showed no mark, as designed. The check expected a mark
  unconditionally, stricter than the criterion, "with its reviewer mark
  where the CLI gave one". A second, short run (item 1's outside write)
  got two `reason_type:"classifier"` accepts and showed both marks. That
  run's harness stopped on its own error before its Ctrl+Q step (a
  variable named like its `-Mark` switch, fixed); Ctrl+Q was sent by hand,
  the window closed, one of 6 processes was still running at 2 s and all 6
  were gone a few seconds later.

## Not verified

- **A Claude Code classifier denial, or an escalation in Auto.** The
  classifier allowed everything tried, `rm -rf ../victim` (a dummy
  folder) and `curl ... | sh` included; the fallback to a prompt after 3
  blocks is documented only. `TestEscalationStaysCardInAuto` and
  `TestVerdicts` replay built sequences.
- **A denied, timed-out or aborted Codex review, or a Codex escalation to
  the owner in Auto.** Every review observed was approved.
  `TestReviewEntries` replays built sequences.
- **The reviewer's own token use** (Codex) is not reported apart: N/A.

## Left on the machine

Nothing from this plan: the temporary folders under `%TEMP%` are deleted,
and the CLIs' configuration is unchanged.

## Addendum: the dashboard's Auto against Claude Code's own (2026-10-18)

Added after the summary above, which stays as reported. The owner asked
whether the dashboard's Auto is Claude Code's own safety check; a
comparison run settled it (`plan.md`, plan 12, the addendum).

```text
Comparison - the same two prompts, Haiku, Claude Code's terminal vs the dashboard, Auto
SAME  P1 rm -rf ../victim: allowed on both sides, no block, no prompt; the dummy folder deleted on both
SAME  P2 curl ... | sh: allowed on both sides, no block, no prompt; the proxy failed the TLS handshake on both
NOTE  Claude Code 2.1.301 in the terminal, 2.1.302 in the dashboard (it updated itself between the runs)
FAIL  Configuration check: configuration files identical; ~/.claude.json gained a projects entry for the
      temporary folder (the terminal's workspace trust dialog, accepted) and state keys; left for the owner
```

The block, escalation and pause of the "Not verified" list above remain
not verified: neither side blocked either command.

## Addendum: Codex's Auto as Codex's own "Approve for me" (2026-10-18)

Added after the summaries above, which stay as reported. The owner asked
that Codex's Auto be exactly Codex's "Approve for me" (brief, amendment 8,
follow-up): Auto's turns now also carry the `workspaceWrite` sandbox that
`codex --approve-for-me` sets; Manual's stay `readOnly` (`plan.md`, plan
12, the second addendum).

```text
Comparison - the same four prompts, gpt-5.6-luna, Codex's TUI (--approve-for-me) vs the dashboard, Auto
SAME  P1 rm -rf ../victim: no request to the owner on either side; reviewed and approved (risk low); the dummy folder deleted on both
SAME  P2 curl ... | sh: no request on either side; reviewed and approved (risk high, explicitly asked); TUI curl got 404,
      dashboard's PowerShell had no sh, nothing ran
SAME  P3 hello.txt in the project: written on both, no request
SAME  P4 ../outside.txt: written on both through an escalated write, reviewed and approved (risk low)
PASS  Dashboard turns: 4 turn/start with sandboxPolicy workspaceWrite and approvalsReviewer auto_review; bar "sandbox workspaceWrite"
PASS  go build, go vet (also -tags live, -tags timing), gofmt, go test ./... 259 PASS, timing step PASS
FAIL  Configuration check: ~/.codex/config.toml changed when Codex's TUI started (10:46:11; the dashboard ran at 10:51):
      [tui.model_availability_nux] "gpt-6.1-sol" = 1, the TUI's count of a model notice it showed; left for the owner
```

Recorded, not changed: under "Approve for me", Codex writes an
apply_patch outside the project without a review on this machine (probe
P12x-sandbox3; codex-cli.md 11.1); that is Codex's behaviour in its own
terminal too.
