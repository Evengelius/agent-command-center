# jev-pilot (Akramovic1)

Plan 1, step 6: reference project review. Read only; nothing built, installed
or run. Sources: `R` = github.com/Akramovic1/jev-pilot @ `5d27135` (paths
below are relative to the repository root); web pages accessed 2026-10-15.

## 1. Source and commit

- URL: https://github.com/Akramovic1/jev-pilot
- Commit: `5d27135b34edf81b40c0bf1df01ae401f6b72ea7`,
  2026-09-30T22:47:38+03:00, "0.12.1: fix effort routing for Chinese prompts"
  (shallow clone, `git log -1`).
- Activity (GitHub page, 2026-10-15): 11 stars, 2 forks, 39 commits, 2 open
  issues, 1 open pull request, no GitHub release (versions are plugin
  versions, tagged `jev-pilot--v<version>`, R `README.md:506`). Topics:
  claude-code-plugin, jev, llm-routing, openrouter, typesafe.
- Age: the changelog runs from 0.4.0 (2026-09-23) to 0.12.1 (2026-09-30)
  (R `CHANGELOG.md:3,223`): about one week of very fast, single-author
  iteration, 15 days before the access date.

## 2. License

- `LICENSE`, first line: `MIT License`; lines 3-6: `Copyright (c) 2026 Ahmed
  Akram` and `Portions copyright (c) 2025 Daniel (San) Ávila — the
  jev-model-router and jev-skill-suggestion mods from
  https://github.com/davila7/claude-code-templates` (R `LICENSE:1-6`). SPDX:
  **MIT**. `plugin.json` and `marketplace.json` say the same
  (R `.claude-plugin/plugin.json:10`).
- The upstream mods are said to be MIT (R `README.md:515`); not checked in
  davila7/claude-code-templates here.
- For a proprietary project: fork, copy and modify are allowed, in closed
  source, if both copyright lines and the permission notice travel with the
  copied parts. Per AGENTS.md, record source, commit and license next to
  anything taken.

## 3. What JEV is

Three different things share the name; none is a "fork layer".

1. **Jev, the model.** TypeSafe AI's first "System One model", in early
   access: "a frontier-intelligence function call: unstructured state in,
   typed probabilistic decisions out"; outputs are typed values "defined in
   advance", each with "calibrated probabilities and confidence scores"; it
   does not generate free text (typesafe.ai/blog/introducing-system-one-models-and-jev,
   dated Sep 15, 2026). It is a hosted, paid decision model, not open source
   and not a coding agent. Reached at `POST api.typesafe.ai/v1/systemone`,
   `POST openrouter.ai/api/v1/systemone` (model `~typesafe/jev-latest`) or
   the Vercel AI Gateway (R `hooks/model-router.policy.ts:10-18,209-211`;
   R `README.md:428-433`).
2. **jev-pilot, this repository.** A **Claude Code plugin containing a mod**
   (in-process JavaScript/TypeScript event handlers, "function hooks"),
   not a CLI, not a dashboard, not a proxy in front of the user. Before
   every turn it asks Jev, in one request (~0.5 s, capped by `timeoutMs`
   1500), a few typed questions about the prompt and recent context, then
   (R `README.md:29-38`, R `hooks/jev-model-router.ts:1-55`):
   - sets the main loop's **reasoning effort** (low..xhigh) at `turn.step`,
     raised up to `max` mid-turn after 2 failed tool calls in a row
     (`escalateAfterErrors`);
   - picks each **subagent's model** (haiku/sonnet/opus, or a custom
     OpenRouter model) at `agent.spawn`, and its effort at its requests;
   - attaches **strategy advice** (direct, delegate, parallel, graph) and the
     **one skill** the prompt needs (hiding the skill listing);
   - records a **decision ledger** and suggests its own tuning
     (`/jev-pilot:report`, `/jev tune`);
   - draws **"Claude the pilot"**, an animated pixel-art pet with a speech
     bubble, in the `AbovePrompt` band of Claude Code's own terminal UI.
   With no key, Claude Code's own `$.model.classify` stands in ("built-in"
   backend, R `hooks/jev-model-router.ts:609`).
3. **jev-router and `claude-jev`.** An optional local HTTP proxy
   (`router/jev-router.mjs`) and a bash launcher (`bin/claude-jev`) that set
   `ANTHROPIC_BASE_URL` to the proxy so custom OpenRouter models can serve
   subagents ("the crew", R `README.md:242-345`).

**The reference screen's JEV elements.** No source found connects jev-pilot
or Jev to "forks", "ULTRACODE" or "jev [3,366 forks]": `grep -i fork` over R
finds only Claude Code's fork subagents, which jev-pilot deliberately leaves
alone (`if (!routeSubagentModel() || e.fork) return next(e)`,
R `hooks/jev-model-router.ts:1035-1036`); a web search (2026-10-15) found
nothing either. What follows is inference, labelled as such:
- The green **routing panel** ("which file", "which model", "retry or stop",
  bar meters with probabilities) has the shape of a System One decision: a
  typed choice with a probability per option. jev-pilot really makes the
  "which model" decision (subagent model, R `hooks/jev-model-router.ts:1020`)
  and an effort decision with a confidence; "retry or stop" has at most a
  loose analogue (the mid-turn effort raise after failures,
  R `README.md:156`); "which file" has none.
- **"jev · forks" / "3,366 forks"** most plausibly counts decision points
  (one Jev answer per prompt or subagent), as jev-pilot's ledger counts
  turns. The figure is a mock-up number (doc/reference-design.md); jev-pilot
  exposes no such counter outside its own `/jev` text and pet bubble.

## 4. Stack and dependencies

- Language: TypeScript (TSX for the pet) executed **inside Claude Code** as a
  mod; types come from the virtual module `claude-code` and
  `claude-code/testing` that Claude Code supplies (R `hooks/jev-pilot.ts:12`;
  `tsconfig.json` extends `.claude-plugin/types/tsconfig.json`, which is not
  in the repository and is written by Claude Code).
- UI library: none of its own; it returns `Box`/`Text` element trees to
  Claude Code's mod renderer (`$.ui.resolve(e)`, R `hooks/jev-pet.tsx:371-405`).
- No `package.json` at the root, so **no third-party runtime dependency**.
  The router is plain Node 18+ or Bun with `node:` built-ins only
  (R `router/jev-router.mjs:30,38-42`). Unit tests run on Bun (`bun:test`,
  R `bunfig.toml`); engine tests on `claude plugin test`. Asset generators are
  Python standard library (R `scripts/assets/*.py`). CI: `actions/checkout@v4`,
  `oven-sh/setup-bun@v2` (R `.github/workflows/test.yml`).
  `bench/fixture/package.json` is a test fixture with no dependencies.
- External services (not dependencies, but runtime requirements for the
  routing): TypeSafe API, OpenRouter or Vercel AI Gateway, each needing a key
  and paid per request (R `.claude-plugin/plugin.json` `userConfig`).

## 5. Architecture

Components (R `README.md:471-492`):

| Path | Role |
| --- | --- |
| `.claude-plugin/plugin.json` | manifest, ~60 `userConfig` options (3 keys marked `sensitive`) |
| `hooks/hooks.json` | `"modules": ["./jev-pilot.ts"]`: the one hooks module |
| `hooks/jev-pilot.ts` | `register(on, options)`: registers router, skill suggester, pet (lines 19-39) |
| `hooks/jev-model-router.ts` | effort, subagent model, strategy, raise, ledger (1,176 lines) |
| `hooks/model-router.policy.ts` | pure decision logic and request building (1,257 lines) |
| `hooks/jev-skill-suggestion.ts`, `skill-suggestion.policy.ts` | skill listing and one-skill pick |
| `hooks/jev-call.ts` | merges both modules' questions into one Jev request per prompt |
| `hooks/context.ts` | recent messages (text and tool names only) and request signals |
| `hooks/ledger.ts`, `tuning.ts`, `session-stats.ts` | decision record in `$.store`, report, tuning |
| `hooks/jev-pet.tsx`, `pet-art.ts`, `features.ts`, `summary.ts` | pet, `/jev` command, switches |
| `hooks/crew*.ts` | custom models, Codex/OpenCode reviewers, health checks |
| `router/jev-router.mjs` | local proxy 127.0.0.1:8799 (secret path), OpenRouter or Anthropic |
| `bin/claude-jev`, `install.sh` | launcher (bash) and installer |

Data flow: `prompt.submit` (classify: one Jev request; attach strategy/skill
as `context`) → `turn.step` (apply effort, optionally model, to each request)
→ `tool.call` (count consecutive failures, raise once) → `agent.spawn`
(route subagent model) → `turn.complete` (close the ledger entry with
`e.usage.output_tokens`, `e.durationMs`) (R `hooks/jev-model-router.ts:418,628,875,1020,905-960`).

Rendering and updates: **event-driven**, inside Claude Code's renderer. The
pet redraws by `$.ui.invalidate('ui.render')` on events, plus timers
(R `hooks/jev-pet.tsx:54-62`): frame every 200 ms during a turn (`FLY_MS`),
450 ms while background subagents run (`CRUISE_MS`), a **poll of
`$.agent.list()` every 1,500 ms** to see whether subagents are running
(`AGENTS_EVERY_MS`, lines 139-160), blink every 4.6 s, an idle play every 9 s
at 180 ms a step. Claude Code throttles redraws to 30/s in the terminal
(code.claude.com/docs/en/plugins/mods/reference, "Limits").

## 6. Data sources

| Source | Where | Status |
| --- | --- | --- |
| Mod events `session.start/end/compact`, `prompt.submit`, `prompt.attachment` (`skill_listing`), `turn.start/step/complete`, `tool.call`, `agent.spawn`, `skill.prompt`, `command.run`, `ui.render` (`AbovePrompt`) | R `hooks/jev-model-router.ts:388,418,628,875,905,966,988,996,1020`; `jev-skill-suggestion.ts:224,274,597,609,619,692`; `jev-pet.tsx:95,168,180,308,328,343,356,371` | documented: https://code.claude.com/docs/en/plugins/mods/reference ("Events") |
| Mods API `$.session.messages/cwd`, `$.agent.list/register`, `$.store`, `$.fs`, `$.http.fetch`, `$.process.run`, `$.model.classify`, `$.command.list/register`, `$.tool.list`, `$.ui.*`, `$.clock.*`, `$.env.get` | e.g. R `hooks/jev-pet.tsx:143` (`$.agent.list`), `ledger.ts:6` (`$.store`) | documented: same page, "Mods API methods" |
| Jev endpoints (TypeSafe, OpenRouter `/v1/systemone`, Vercel Gateway) | R `hooks/model-router.policy.ts:10-18,209-211`, request at `jev-model-router.ts:146` | third-party; TypeSafe docs cited at R `hooks/skill-suggestion.policy.ts:10` (docs.typesafe.ai/cookbooks/skill_suggestion); not verified here |
| OpenRouter `/api/v1/models`, `/api/v1/messages` (1-token health check) | R `hooks/crew-run.ts:88,251` | third-party, not verified here |
| `codex login status`, `codex debug models`, `codex exec -s read-only`, `opencode --version`, `opencode auth list`, `opencode models`, `opencode run --agent plan` | R `hooks/crew-run.ts:103,112-113,122,129`; `hooks/crew.ts:595-596` | Codex/OpenCode CLI commands; `codex debug models` looks undocumented; none verified against their docs here |
| `~/.claude/settings.json` (pluginConfigs, skill overrides) | read: R `hooks/jev-model-router.ts:1004`, `jev-skill-suggestion.ts:648-652`, `router/jev-router.mjs:121` | settings file; documented location |
| `~/.claude/plugins/installed_plugins.json`, `~/.claude/skills/synced/…`, project and user `.claude/skills` | R `hooks/jev-skill-suggestion.ts:356-374,398-403` | internal file layout: **undocumented** |
| Own files `~/.claude/jev-pilot/{models.json,picker.json,router-secret,router.log,router-models.json,router.out}` | R `hooks/crew-run.ts:36,42,50-51`; `router/jev-router.mjs:47-53,145`; `bin/claude-jev:56,136` | its own |
| **`~/.claude/.credentials.json`** | **R `router/jev-router.mjs:121`** (looks for the OpenRouter key under pluginConfigs); **R `bench/run.ts:39,128`** (reads it and copies `claudeAiOauth` into a temporary config dir) | **credential file: red flag** |

Telemetry relevant to our zone D:
- **Quota / rate limits**: not read anywhere in R.
- **Tokens**: `turn.complete` `e.usage` (output tokens per turn into the
  ledger, R `hooks/jev-model-router.ts:939`; subagent input, output and cache
  tokens summed, R `hooks/session-stats.ts:78-86`).
- **Cost**: deliberately none ("Counts, not dollars", R
  `hooks/session-stats.ts:7-8`); only the bench reads `total_cost_usd` from
  `claude -p --output-format json` (R `bench/run.ts:155,173`).
- **Subagents**: `agent.spawn` (model choice) and `$.agent.list()` status
  polling (R `hooks/jev-pet.tsx:143`).
- Not used by jev-pilot but documented on the same page and relevant to us:
  `$.session.usage()` returns `{ startedAt, context, rateLimits, cost }`
  with `rateLimits` as `{ kind, percentUsed, resetsAt }`, the
  `session.measure` event ("after each turn, and when a plan limit's percent
  used changes"), and `turn.step` results with per-request `usage`
  (code.claude.com/docs/en/plugins/mods/reference, "Session", "Mods API
  methods"; code.claude.com/docs/en/plugins/mods/events, "Follow a turn").

## 7. Agent CLIs supported

- **Claude Code only** as the host: Claude Code ≥ 2.1.278 with
  `CLAUDE_CODE_ENABLE_FUNCTION_HOOKS=1` (R `install.sh:27`,
  `bin/claude-jev:43`). Since v2.1.287 mods are on by default and the
  variable is ignored (code.claude.com/docs/en/plugins/mods/overview, "Turn
  mods on or off"); the owner has 2.1.300 (AGENTS.md).
- **Codex CLI and OpenCode** only as *reviewers* run by a Haiku subagent with
  Bash, read-only: `codex exec … -s read-only` and `opencode run --agent plan`
  (R `hooks/crew.ts:595-596`), health-checked via `codex login status`
  (R `hooks/crew-run.ts:103`). Not driven as the main agent; no Gemini CLI.

## 8. Conversation

It neither hosts nor drives a conversation. It runs inside the user's own
Claude Code session and **intervenes** in it: rewrites each turn's effort and
subagent models, adds context to prompts, hides the skill listing, can
append a transcript line per turn (`display: transcript`). The conversation
stays Claude Code's own TUI. No streaming client, no approvals relay, nothing
reusable for zone B. The bench is the only place that drives a CLI, headless:
`claude -p … --output-format json --permission-mode acceptEdits` (R
`bench/run.ts:155`).

## 9. Configuration side effects

Against our rule (never change a CLI's configuration without consent):
- **Installs a plugin** into Claude Code (`claude plugin marketplace add`,
  `claude plugin install … --config key=…`, R `install.sh:247,262`); clone
  mode **rewrites `~/.claude/settings.json`** (`pluginConfigs["jev-pilot"]`,
  API key in plain text) after a timestamped backup (R `install.sh:58-104,219-227`).
- **Sets `ANTHROPIC_BASE_URL`** to a local proxy through which *all*
  Anthropic traffic, including the Claude login headers, passes
  (R `bin/claude-jev:134-145`; `router/jev-router.mjs:244-252`); also sets
  `ENABLE_TOOL_SEARCH=true` and passes `--settings picker.json`
  (R `bin/claude-jev:145-161`). The proxy runs detached under a `nohup`/`setsid`
  supervisor that survives the terminal (R `bin/claude-jev:132-136`).
  Documented variable: "Override the API endpoint to route requests through a
  proxy or gateway…" (code.claude.com/docs/en/env-vars, `ANTHROPIC_BASE_URL`).
- Writes `~/.claude/jev-pilot/*` and a launcher symlink in `~/.local/bin`
  (R `install.sh:153-161`).
- `/jev-pilot:setup` makes Claude edit `~/.claude/settings.json` skill
  overrides after an explicit yes, with a backup (R
  `hooks/jev-skill-suggestion.ts:647-652`; `skill-suggestion.policy.ts:1023`).
- At run time it **changes model behaviour**: effort, subagent models, injected
  context, hidden skill listing. Any `claude` session our app starts on a
  machine where jev-pilot is installed and enabled will carry these changes,
  since mods load in every session, `claude -p` and the Agent SDK included
  (mods overview, "Where mods run"). The prompt text and recent messages are
  sent to the Jev backend (R `README.md:445-450`).
- On this machine (checked 2026-10-15, existence only): no
  `~/.claude/jev-pilot`, no `installed_plugins.json`, only the
  `claude-plugins-official` marketplace, no `claude-jev` on the PATH:
  **jev-pilot is not installed.**

## 10. Platform support

- The installer says "macOS and Linux" (R `install.sh:2`). The launcher is
  bash and uses `setsid`, `ss`, `/proc/<pid>/cmdline`, `od /dev/urandom`,
  `chmod 600`, `ln -s` (R `bin/claude-jev:64,80,112-113,133`): Linux first,
  macOS mostly, **Windows native not supported** (Git Bash partly at best;
  unverified). The hooks read `HOME` (R `hooks/jev-model-router.ts:394`),
  which Windows native processes may not set (unverified).
- The mod itself runs wherever Claude Code mods run: terminal, Desktop Code
  tab, VS Code; not in Desktop WSL sessions; hooks but no drawing in
  `claude -p`/Agent SDK (mods overview, "Where mods run").

## 11. UI and visual relevance

- Its only UI is a pixel-art pet and a rounded speech bubble drawn in Claude
  Code's `AbovePrompt` band (R `hooks/jev-pet.tsx:371-405`, sprites in
  `hooks/pet-art.ts`). Nothing resembling our four zones, frames, workflow
  tree, quota bars or log.
- Small transferable ideas: a braille spinner `⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏`
  (R `hooks/jev-pet.tsx:63`); a mood-coloured status bubble; activity words
  mapped from tool names (thinking, reading, searching, writing, running,
  working; R `README.md:209-217`, `actOfTool` in `hooks/pet-art.ts`); decision
  text with confidence (`xhigh · /systematic-debugging · parallel · 88% sure`,
  R `README.md:219`), which is what a JEV line in our dashboard could show.

## 12. Quality

- Tests: 13 unit spec files (`tests/*.spec.ts`, ~3,240 lines) and 2 engine
  test files (`engine/*.test.ts`, 400 lines); CI runs `bun test`, `bash -n`
  on the scripts and a version-consistency check (R `.github/workflows/test.yml`).
- Docs: long README (534 lines), `CHANGELOG.md` (239 lines), the upstream
  mods' docs in `docs/`, a benchmark harness with 10 tasks and 6 result files
  (R `bench/`).
- Size: ~7,400 lines of TypeScript in `hooks/`, 389 in the router, 244 in the
  launcher, 288 in the installer; 111 files.
- Maturity: v0.12.1 after one week, early-access platform at the time,
  11 stars, single author; the code is carefully commented and fail-open
  (every Jev failure leaves the turn unchanged, R
  `hooks/jev-model-router.ts:44-45`).

## 13. Reusable parts

All MIT, so reusable with both copyright notices:
- **Patterns, not code, for an optional companion mod** (if plan 2 chooses
  one and the owner approves it): observing `turn.step`/`turn.complete` usage,
  `tool.call` results (`isError`), `agent.spawn`, and polling
  `$.agent.list()` statuses (R `hooks/jev-pet.tsx:139-160,308-369`;
  `hooks/jev-model-router.ts:905-913`). Our language is Go, so this is study,
  not copy, unless we write that mod in TypeScript.
- `hooks/context.ts` ideas (recent context limited to text and tool names) and
  `codexVerdict` (parse `codex login status`, R `hooks/crew-state.ts:108-113`):
  a few lines, easier to rewrite than to carry a notice for.

Not to reuse:
- `router/jev-router.mjs` and `bin/claude-jev`: they set `ANTHROPIC_BASE_URL`,
  proxy the user's Claude login, run a detached daemon, and read
  `~/.claude/.credentials.json` (line 121): against three hard constraints.
- `bench/run.ts`: reads and copies the Claude OAuth token (lines 39, 128).
- `install.sh`: edits `~/.claude/settings.json` and installs a plugin.
- The routing logic and prompts: they change the agent's behaviour, which is
  not our job (the CLI does the work), and they depend on a paid third-party
  model.

## 14. Verdict

**Not a fork candidate.** jev-pilot is a Claude Code plugin that steers the
agent (effort, subagent model, skill) with a third-party decision model; it is
not a dashboard, has no Go or Bubble Tea code, no conversation, no
quota/cost data, no Windows support for its launcher, and its optional parts
change Claude Code's configuration, proxy its traffic and read its credential
file. The reference screen's "jev · forks" and routing panel are mock-up
decoration with, at most, a loose analogy to Jev's typed decisions.

Its real value to us is indirect: it is a working example of Claude Code
**mods**, whose documented events and API (`turn.step` usage, `agent.spawn`,
`$.agent.list()` statuses, `$.session.usage()` with `rateLimits` and `cost`,
`session.measure`) are a strong candidate data source for zones A, C and D.
A mod can be loaded for one session with `claude --plugin-dir <dir>`, without
installing anything (mods reference, "Commands"), but it runs with the user's
full permissions; plan 2 should weigh it, and the owner approve it, like any
other integration.

How "JEV optional" should be handled:
- **Never required, never installed, never launched.** The app starts plain
  `claude`, never `claude-jev`, and never sets `ANTHROPIC_BASE_URL`.
- **Detection, read only, no credentials:** jev-pilot is present when
  `claude plugin list` names `jev-pilot` (command not verified in the docs
  read here; or, as an undocumented fallback,
  `~/.claude/plugins/installed_plugins.json` does) or the session's model
  traffic goes through a `127.0.0.1:8799` base URL; `~/.claude/jev-pilot/`
  existing means it was used. Do not read `~/.claude/settings.json`'s
  `pluginConfigs` values: a clone install keeps the OpenRouter key there in
  plain text (R `install.sh:75-77`).
- **What the dashboard can show if present:** a "JEV: active" badge (the
  plugin is enabled, so it is steering our session too), the subagent models
  and efforts actually observed (from the CLI's own events, not from Jev), and
  "N/A" for Jev's decisions and confidences, which jev-pilot exposes only in
  its pet bubble, its `/jev` text and its private `$.store` ledger
  (location undocumented). If absent: "JEV: not installed", nothing else.
