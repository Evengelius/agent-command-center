# Plan 2 evaluation (2026-10-16)

The final check of plan 2's 15 criteria (`plan.md`), as reported to the owner,
word for word. The results are not to be rewritten.

```text
Evaluation - Plan 2, architecture and reliability - 15 criteria: 15 PASS, 0 FAIL
PASS 1  4 files in doc/ (architecture 564, events-and-state 338, ui-design 939, reliability 576 lines), each with its sections
PASS 2  Coverage table: every item of the brief's phases 2-3 and amendments 2-4 has a section (AM4 S0 and N1-N4 added after the verifier's check)
PASS 3  Both screens have a "Follows and departs" section against the reference images
PASS 4  Every design choice names its research finding, brief item or assumption; untraced choices are now assumptions ASM-21 and ASM-22, and ASM-12 is widened
PASS 5  Every conceptual event has a source per CLI with evidence, or "unavailable" with a fallback
PASS 6  Every telemetry and integration source has a fallback in reliability.md
PASS 7  Six state machines, each with its states, transitions, invalid transitions and a column per CLI (6 rows added after the check)
PASS 8  Mock mode is its own provider, starts only with --mock, is labelled on every screen, and cannot reach a live session; the flag rules are defined
PASS 9  Mock-ups at 80x24, 120x40 and 200x60, plus the startup screen
PASS 10 No workflow engine, PTY or screen scraping
PASS 11 Gemini and Antigravity appear only as out of scope; none of amendment 4's excluded features is in the design
PASS 12 16 ms per keystroke with a 10,000-line buffer; a 50,000-line paste drawn within 100 ms (ASM-10)
PASS 13 The configuration check separates configuration files, mixed-state files and session data; volatile keys are an explicit list; no file is excluded whole; credential values are skipped, never read
PASS 14 Every open limitation from plan 1 is in "Not verified" with the probe or test that settles it (2 rows added after the check)
PASS 15 git status: only plan.md and the 4 new doc files beyond plan 1's; UTF-8 without BOM, LF only; links 0 bad; no personal names in the files
Not verified: any live CLI behaviour (this plan is design only, no code); Linux and macOS; external URLs.
Verdict: DONE
```

## How the check ran

- **First pass, independent:** one verifier agent checked the criteria that
  need judgement. It found criterion 14 failing (CC-10 q2's untried part and
  Codex `alternate_screen` missing from "Not verified") and gaps under
  criteria 4, 7, 8 and 13.
- **Fixes:** all of them were fixed in the main session.
- **Re-check, not independent:** the fixes were re-checked only by the main
  session's searches and the link checker. **The verifier did not run a
  second time**, so the final 15 PASS rests on the main session's re-check
  for criteria 4, 7, 8, 13 and 14.

## The owner's decision (2026-10-16)

The owner approved plan 2 and started plan 3, asking that the distinction
above be kept. The same message superseded criterion 13's design: "Never
inspect, parse, expose, or modify credential values. Use only permitted
integrity-verification methods, such as opaque file hashing where
appropriate." The streaming tokenizer over mixed-state files is therefore
withdrawn; plan 3 replaces it with opaque whole-file hashing and states the
limitation ([reliability.md, Configuration untouched](reliability.md#configuration-untouched)).
Criterion 13's PASS above stays as recorded: it judged the design as it stood.

The performance figures (criterion 12) are targets, not results, until
plan 4's benchmarks measure them (the owner's instruction of the same day).
