# Plan 23 evaluation (2026-10-21)

The check of plan 23, the `acc` launcher in the project (brief amendment 21).
It covers the criteria written in [`plan.md`, Plan 23](../plan.md) before the
code. Typed in Windows Terminal, `acc` opens the dashboard in a tab of its
own at the owner's size, a 9 pt font, and only there. The launcher lives in
the repository: `acc.cmd`, which runs `scripts\acc.ps1`.

How it was checked:
- Each case ran in a Windows Terminal window of its own, opened maximised
  and idle (`scratchpad\term\plan23-test.ps1`).
  - The first tab was measured through UI Automation before `acc` was
    typed, while the dashboard's tab was up, and after it closed.
  - The windows were closed by their handles. The owner's own windows were
    never touched.
- The cases used a mock replay and the startup screen only: no agent CLI
  session, no quota.
- Before the first case, Terminal was made to forget the profile: the
  fragment deleted, then a reload. That is a fresh machine's state for the
  launcher.
- The earlier runs failed. Each cause was found and fixed before the next:
  - the harness's `;`, which `wt` takes as its separator;
  - the fragment unread until a settings reload (the script now makes one);
  - leftover windows of earlier runs;
  - the profile's command line matching other PowerShell tabs (now unique);
  - a wrapped line that UI Automation reads as one row (the harness's
    measure).

```text
Evaluation - Plan 23, the acc launcher in the project - 6 criteria checked before the commit: 6 PASS
PASS 23.1 From a copy of the repository, .\acc --mock testdata/mock/claude-subagent-approval.jsonl: the fragment written; the dashboard in a second tab, "Agent Command Center", of the same window at 208x59 maximised; the first tab 156x44 before and after; Ctrl+Q closed the dashboard's tab
PASS 23.2 From %TEMP%\acc-demo, the owner's acc --provider codex: the startup screen in that tab at 208x59, naming acc-demo and Codex; Ctrl+Q quit with no CLI session (no codex process left); the tab closed; the first tab 156x44 before and after
PASS 23.3 The second run left the fragment's bytes and time unchanged; Terminal's settings.json kept its bytes (the script sets only its time, and only when it writes the fragment); Terminal added no entry, having seen the profile's id in the earlier runs
PASS 23.4 Outside Windows Terminal, acc.cmd --provider bogus ran the program in place: its own error, "--provider "bogus": use claude-code or codex", and exit 1
PASS 23.5 Terminal's settings.json as it was this morning (3,064 bytes, no test or acc profile); the test fragments removed; the owner's profile's acc one line calling acc.cmd; no backup left
PASS 23.6 No Go file changed; acc.cmd CRLF through the .gitattributes rule *.cmd text eol=crlf; acc.cmd and scripts\acc.ps1 ASCII; AGENTS.md, plan.md, brief.md and this file UTF-8 without BOM, LF
Not verified: a fresh machine, where Terminal has not seen the profile's id and adds its entry once; Windows Terminal Preview and an unpackaged Terminal, whose settings' time the script also sets; acc typed in a tab of the profile itself (WT_PROFILE_ID); 23.7, the commit and the push, checked once they are made
Verdict: DONE
```

**23.7, once the commit and the push were made (2026-10-21).** Commit
e8d3e3b was pushed to `origin`. `origin/main` equals `main`, the working
tree is clean, and the author is `Evengelius <molimoabel@gmail.com>`, with
no attribution line: PASS. The owner approved plan 23 on 2026-10-22. The
block above stays as it was reported.

What is left outside the repository, by design:
- The owner's PowerShell profile has a one-line `acc` calling the
  repository's `acc.cmd`. PowerShell never runs a command from the current
  folder by its bare name. In the repository's folder, `.\acc` needs
  nothing.
- The fragment the script writes, in
  `%LOCALAPPDATA%\Microsoft\Windows Terminal\Fragments\AgentCommandCenter`.
- Terminal's own record of the profile, and its `state.json` list of the
  profile ids it has seen. That list still names the two test profiles of
  the checks, which is harmless.
