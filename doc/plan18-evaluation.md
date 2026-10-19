# Plan 18 evaluation (2026-10-19)

The final check of plan 18, an adaptive layout (brief amendment 14, with the
owner's second message). It checks the plan's four items in
[`plan.md`, Plan 18](../plan.md#plan-18---an-adaptive-layout) against the
yes/no criteria written there, rewritten after the owner's second message
and before the rebuilt layout was checked. It also checks the plan's rule:
the four zones keep their places, and only the height zone C takes from B
changes.

The evidence is the tests, the golden frames, four mock runs and two live
runs in a real Windows Terminal; two earlier live runs, whose failures were
the test script's and Claude Code's own caches, are recorded below. The
block below is the summary reported to the owner. The results are not to be
rewritten.

```text
Evaluation - Plan 18, an adaptive layout - 16 criteria: 16 PASS
PASS 1.1 No agent: 0 in play; two agents of the owner's turn: 2, still 2 once done (TestAgentsInPlay)
PASS 1.2 The next prompt, every agent done: 0; an agent of that turn: 1 (TestAgentsInPlay)
PASS 1.3 A turn the CLI opens by itself changes nothing, with agents in play or none (TestAgentsInPlay; it failed when the rule counted the CLI's stages as prompts)
PASS 1.4 An agent running at a prompt: 1; with the new turn's agent: 2; both ending in it: 2; the next prompt: 0 (TestAgentsInPlay; it failed when the rule forgot when an agent ended)
PASS 1.5 Before any prompt, a replay's agent: 1 (TestAgentsInPlay)
PASS 2.1 120x40: no agent C 4 and B 29; one agent C 4 with its row and the key hint; six C 9 and B 24; twenty C 16 and B 17; after a new prompt C 4, the agent listed (TestAdaptiveLayout)
PASS 2.2 200x60: no agent C 4 and B 47, no preview; one agent C 8 with the preview; twenty C 23; forty C 25 and B 26; after a new prompt C 4, no preview (TestAdaptiveLayout)
PASS 2.3 80x24: no agent C 3 and B 14; one agent C 4 with the key hint; six C 5 and B 12 (TestAdaptiveLayout)
PASS 2.4 100x24: twenty agents, B 12 and C 5; no agent, C 4 and B 13 (TestAdaptiveLayout)
PASS 2.5 Twenty agents done before a new prompt: the strip; focused, C 16 at 120x40, 23 at 200x60, 5 at 80x24; no agent focused, the strip (TestAdaptiveLayout)
PASS 2.6 z zoom on C and F6 as before: the existing tests pass unchanged, TestPaneScroll included (it failed on the first agent-sized build, its scroll hint missing from a focused 4-row strip)
PASS 3.1 38 golden frames rewritten and reviewed against HEAD by script: in each, the title and project rows, zone A's columns and zone D unchanged; C 4 rows with one agent (8 at 200x60), 5 with two, B the rest; the P2 frames C 4 and B 29; window scenes changed only behind the window; 79x24 unchanged
PASS 3.2 go build ./... and go vet ./... exit 0, gofmt -l . empty; go test ./... 343 PASS, 1 SKIP (TestDiskListLink), 0 FAIL in 14 packages; the timing step 4/4 PASS
PASS 3.3 ui-design.md (layouts, zone C), events-and-state.md (Zone C's height), architecture.md (ASM-12) and AGENTS.md (status) say what was built; brief.md holds amendment 14 with the owner's second message
PASS 4.1 Mocks in a real Windows Terminal, 4 runs, 0 FAIL: maximised 156x44, Claude (one agent) C 4 throughout, Codex (two) C 4, 4, then 5 kept; 80x24, Claude C 3 then 4, Codex C 3, 4, then 5
PASS 4.2 Live, Claude Code 2.1.303 on Haiku: maximised, the strip (4), then 5 rows for the two agents of prompt 1, the strip from prompt 2's submission and after it; 80x24, 3, 5, 3, 3; each run 0 FAIL, configuration PASS, the project unchanged, no process left
Not verified: none
Verdict: DONE
```

## The first build and the owner's second message

The first build gave C half the right column whenever an agent was in play.
It passed items 1 to 3 and 4.1 (maximised, C 4 rows then 18; 80x24, 3 then
5). The owner saw one agent leave C mostly empty rows and answered that the
space should adapt to the deployed agents. The rule became a count, C's
height the rows its agents need, and the criteria above were rewritten
before the rebuilt layout was checked. The key hint also shows when the
agents leave a row for it, so a box sized to its agents has no blank row.

Two consequences, both in the plan's choices: at 160 columns and wider, C
keeps 8 rows while agents are in play, for the selected agent's preview;
and C, while focused, is sized for every agent it lists. The second came
from TestPaneScroll, which failed on the first agent-sized build: focusing
C to scroll earlier agents left a 4-row strip with no row for its scroll
position hint. The test was not changed; it passes with the focus rule.

## The terminal runs

`p18mock.ps1` uses the plan 17 harness: a Windows Terminal window driven by
UI Automation, the screen read as text, a temporary project, store and
home, and a PATH of System32 only, so no agent CLI is found. It reads B's
and C's heights at each step and computes what plan 18 expects at the size
on screen. Each of the four runs: 0 FAIL, the window closed, no agent CLI
started (the Claude desktop app's processes left aside), nothing written in
the project.

| Run | Before the agents | One agent | Two agents | After they finish |
| - | - | - | - | - |
| Claude mock, 156x44 | C 4, B 33 | C 4 | - | C 4 |
| Codex mock, 156x44 | C 4, B 33 | C 4 | C 5, B 32 | C 5 |
| Claude mock, 80x24 | C 3, B 14 | C 4, B 13 | - | C 4 |
| Codex mock, 80x24 | C 3, B 14 | C 4 | C 5, B 12 | C 5 |

## The live runs

`p18live.ps1`, approved by the owner on 2026-10-19: the dashboard on
Claude Code 2.1.303, Haiku through `ANTHROPIC_MODEL` in that window only,
the CLI through the logging wrapper of plan 17, a temporary project
(`README.md`, `notes.txt`), the real home so that Claude Code is signed in,
and the configuration check before and after. Prompt 1 asks for two
general-purpose subagents in parallel (a Glob, a Read); prompt 2 is "Reply
with only the word ok." Any card other than a read would be denied and fail
the run; none came.

The runs that count, at 23:27 and 23:28:

| Size | Before any prompt | Prompt 1 done | Prompt 2 submitted | Prompt 2 done | Cost |
| - | - | - | - | - | - |
| 156x44 | C 4, B 33 | C 5, B 32, "0 ACTIVE · 2 DONE" | C 4 | C 4, both agents listed | USD 0.016 |
| 80x24 | C 3, B 14 | C 5, B 12 | C 3 | C 3 | USD 0.011 |

Both runs: model claude-haiku-5-5, two Agent calls, the results "done" and
"ok", 0 FAIL, configuration PASS, the project unchanged, no Claude process
left. At 80x24 the first reply also carried the owner's Px reminder: the
owner's global `~/.claude/CLAUDE.md` asks every Claude Code session for it,
and the sessions the dashboard starts read that file too.

### Two earlier runs, both maximised

Run 1, at 23:18, as reported:

```text
23:18:36 [the strip before any prompt] PASS  ready True; C 4 rows (want 4), B 33; ''
23:19:01 [prompt 1: C sized for its agents] FAIL  turn ended True; ''; C 5 rows (want 4 for 0 agents), B 32
23:19:01 [prompt 1: two agents deployed] FAIL  ''
23:19:04 [prompt 2 submitted: the strip] PASS  turn running on screen True; C 4 rows (want 4), B 33; ''
23:19:06 [prompt 2 done: still the strip] FAIL  turn ended True; C 4 rows (want 4), B 33; '' still listed
23:19:14 [confcheck] FAIL  exit 1; ... | configuration: FAIL
23:19:14 [total] FAIL  4 FAIL
```

The three layout FAILs were the script's: PowerShell names ignore case, and
the function's `$mid`, a row count, hid the script's `$MID`, the "·" of the
agent counts, so the counts were never read (`''`) and the script expected
the strip for "0 agents" while C showed 5 rows for the two. The
configuration FAIL came from Claude Code's own caches, written during the
run: its synced skills manifest at 23:18:46 and
`plugins\plugin-directory-cache-v2.json` at 23:19:03; no settings file
changed (`settings.json` last written at 23:00, before the run). The
agents ran in the background that time: four results, two from turns the
CLI opened for their notifications, which left C as it was.

Run 2, at 23:21, after the variable was renamed:

```text
23:24:53 [prompt 1: C sized for its agents] FAIL  turn ended False; '0 ACTIVE · 2 DONE'; C 5 rows (want 5 for 2 agents), B 32
23:25:06 [confcheck] FAIL  exit 2; ... known_marketplaces.json  changed, not classified (contents not inspected) ... | configuration: NOT-VERIFIED
23:25:07 [total] FAIL  2 FAIL
```

Every height was right. The FAIL was the script's end-of-turn detection,
read from the wrapper's log, which lagged: prompt 1 had ended after 13.4 s
and the screen showed the idle input, but the script waited its 3 minutes.
It now reads the screen, as the two runs that count did. The configuration
check found `plugins\known_marketplaces.json` changed: Claude Code's
marketplace refresh, its one `lastUpdated` written at 23:23:50 during the
run, as in run 1 at 23:18:58.

The four live runs cost about USD 0.07, API-equivalent.

## Zone C cleared at each prompt (2026-10-19)

At review, the owner asked for zone C to be cleared, "back to empty", when
a new prompt is submitted: "Because it is a new prompt, no need to keep the
history" (brief amendment 14, third message). The block above stays as it
was reported; this one checks item 5 of the plan, written for the change.
C now lists only the agents in play, its counts too; ↑ ↓ in C and [ ] in an
inspection go through that list; the earlier agents stay in zone A's tree.
An agent still running at a new prompt stays, as do the agents of turns the
CLI opens by itself. The second build's rule sizing a focused C for every
agent it listed went, having nothing left to show.

```text
Evaluation - Plan 18, zone C cleared at each prompt - 6 criteria: 6 PASS
PASS 5.1 State.InPlay lists the agents in play, the earlier turns' cleared: "bg,new" at the end of 1.4's turn (TestAgentsInPlay)
PASS 5.2 After a new prompt, every agent done: C empty, "0 ACTIVE · 0 DONE" and "no agents yet", C 4 and B 29 at 120x40, 4 without the preview at 200x60, 3 at 80x24, focused or not; six agents in play focused as unfocused, C 9 (TestAdaptiveLayout)
PASS 5.3 In each of TestAdaptiveLayout's 20 cases, C empty exactly when no agent is in play
PASS 5.4 TestPaneScroll's list, wheel and zoom subtests on a prompt deploying 30 agents (60 zoomed), the agents of 60 earlier prompts no longer listed; assertions unchanged, all PASS
PASS 5.5 No golden frame changed by the clearing (TestGoldenFrames without -update); go build, go vet exit 0, gofmt -l . empty; go test ./... 343 PASS, 1 SKIP (TestDiskListLink), 0 FAIL; the timing step 4/4; ui-design.md and events-and-state.md updated
PASS 5.6 Real Windows Terminal: the four mocks 0 FAIL as in 4.1; live (Haiku) maximised C 4, 5, then from prompt 2's submission C 4 with "0 ACTIVE · 0 DONE" and "no agents yet", and after it; 80x24 3, 5, then 3 cleared, the same; both 0 FAIL, configuration PASS
Not verified: none
Verdict: DONE
```

The live runs, at 23:49 and 23:50:

| Size | Before any prompt | Prompt 1 done | Prompt 2 submitted | Prompt 2 done | Cost |
| - | - | - | - | - | - |
| 156x44 | C 4, "0 ACTIVE · 0 DONE" | C 5, "0 ACTIVE · 2 DONE" | C 4, "0 ACTIVE · 0 DONE", "no agents yet" | the same | USD 0.016 |
| 80x24 | C 3 | C 5, "0 ACTIVE · 2 DONE" | C 3, cleared | the same | USD 0.018 |

At 80x24 the agents ran in the background: the first reply came before
them, and the CLI opened two turns of its own for their notifications. C
kept them through those turns and cleared only at the owner's prompt, the
"only when a new user prompt is submitted" of the owner's message.
