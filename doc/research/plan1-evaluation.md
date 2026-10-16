# Plan 1 evaluation (2026-10-16)

The final check of plan 1's 11 criteria (`plan.md`), as reported to the owner,
word for word. The results are not to be rewritten: the two FAILs stay FAILs.

```text
Evaluation - Plan 1, research - 11 criteria: 9 PASS, 2 FAIL
PASS 1  10 files in doc/research/ with their step's sections: all present (headings checked per file)
PASS 2  claude-code.md, codex-cli.md: docs pages with access date, ways to drive, subagents, observation points, detection/version, working directory, permission/sandbox modes, documented/undocumented marked (sections 1-6, "Documented" columns)
PASS 3  integration.md: 18 answers (9 questions x 2 CLIs), each with Sources, Limitations, Fallback (18/18/18 lines) + path forms (Windows, ~, /mnt/c, \\wsl$/\\wsl.localhost, Linux, macOS)
PASS 4  telemetry-sources.md: every zone D value for both CLIs, source or N/A
PASS 5  terminal-input.md: every amendment 3 item (sections 2-9)
PASS 6  4 reference files: full SHA, commit date, SPDX MIT read from the license file
PASS 7  Every claim sourced: verifier sampled 64 claims in integration/telemetry/summary, 62 supported; the 2 others (Codex resume replay cited to P6b) fixed to cite [T] README and marked not probed
FAIL 8  Probes only after owner's approval: expected all approved first, found P3b re-run without its own approval, before you set the re-run rule (documented in probes/README.md)
FAIL 9  Config hashes identical before/after: expected 22/22, found 20/22 on 2026-10-15 (~/.claude.json, known_marketplaces.json: Claude Code's own state rewrites, not the probes); P7 on 2026-10-16: 22/22
PASS 10 git status: only doc/, plan.md, AGENTS.md, CLAUDE.md, .claude/agents/ (3 files) + GEMINI.md deleted under amendment 5
PASS 11 summary.md: build from scratch with reasons; Claude Code first
Not verified: external URLs (not re-fetched); spawned subagents' models; Explore override; concurrency limit; compaction trigger; UNC working directory; wsl.exe piping
Verdict: NOT DONE (strict) - pending your decision on 8 and 9
Likely cause: 8: P3 needed a re-run after a driver fix, before the rule existed. 9: Claude Code rewrites its own state files; this research session ran concurrently; only a timestamp changed, no setting.
Proposed next step: 8, 9: accept as recorded in probes/README.md; for future probes, hash Claude's state files apart from settings - waiting for approval
```

## The owner's decision (2026-10-16)

The owner accepted the two historical failures as documented exceptions,
without changing their FAIL status, and accepted plan 1 for progression to
plan 2, with these instructions:

1. Preserve the evidence for P3b and the configuration hash differences:
   [`probes/README.md`](probes/README.md), "Approval" and "Configuration
   check", and this file.
2. Do not exclude whole Claude state files from future integrity checks:
   distinguish volatile runtime state from meaningful configuration changes,
   without reading credential contents. The design is in
   [`../reliability.md`](../reliability.md), "Configuration untouched".
3. Do not retry the blocked `.claude/settings.json` write or bypass it.
4. Gemini CLI and Google Antigravity stay outside the MVP;
   [`gemini-cli.md`](gemini-cli.md) stays, marked out of scope.
5. No expensive live probe when recorded fixtures and deterministic tests
   suffice; a probe that needs approval under `doc/evaluation.md` still gets
   it.
6. Every unresolved technical limitation stays marked "Not verified".
