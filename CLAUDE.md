@AGENTS.md

## Claude Code in this repository

How the delegation rule of AGENTS.md applies to Claude Code. It is about how we
develop, not about the application: nothing here goes into its provider code.

- **Main session: Opus**, effort medium by default, high only for architecture,
  hard decisions and reviewing consolidated findings; set it with `/effort` or
  the VS Code effort selector.
- **Project subagents** in `.claude/agents/`, each with its model and effort in
  its frontmatter; invoke one only when the delegation rule says so:
  - `Explore`: Haiku, read-only (Read, Grep, Glob, WebFetch, WebSearch); it
    replaces the built-in Explore, which would otherwise run on Opus.
  - `implementer`: Sonnet, edits and runs commands; for plan 4 code changes.
  - `verifier`: Sonnet, no edit tools; reports in the format of
    `doc/evaluation.md`.
- Built-in `general-purpose` and `Plan` subagents run on the main model (Opus):
  prefer a project agent, or pass `model` when spawning one.
- Three concurrent subagents at most: a ceiling, not a target. No Workflow, no
  Ultracode, no agent swarm unless the owner asks for one in the request.
- `isolation: worktree` only when parallel agents would edit the same files.
- `/tasks` shows the model and effort each running subagent uses.
- The auto-compact variables (`CLAUDE_CODE_AUTO_COMPACT_WINDOW`,
  `CLAUDE_AUTOCOMPACT_PCT_OVERRIDE`) are the owner's; leave them as they are.
