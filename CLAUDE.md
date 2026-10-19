@AGENTS.md

## Claude Code in this repository

How the delegation rule of AGENTS.md applies to Claude Code. It is about how we
develop, not about the application: nothing here goes into its provider code.

- **Main session: Opus 5.5 at Max**, the orchestrator. Settings cannot save Max
  (they stop at xhigh): the owner selects it each session with `/effort max` or
  the VS Code effort selector; at session start, if `CLAUDE_EFFORT` (Bash) is
  not `max`, say so.
- **Project subagents** in `.claude/agents/`, each with its model and effort in
  its frontmatter; invoke one only when the delegation rule says so:
  - `Explore`: Haiku, max, read-only (Read, Grep, Glob, WebFetch, WebSearch);
    it replaces the built-in Explore, which would otherwise run on Opus.
  - `implementer`: Sonnet, high, edits and runs commands.
  - `verifier`: Sonnet, high, no edit tools; reports in the format of
    `doc/evaluation.md`.
- **Roles.** Use the fewest agents the task needs, often none. Choose the role
  by the task's actual difficulty: its reasoning complexity, ambiguity,
  technical uncertainty, dependencies and implementation risk, not how many of
  the five parts it touches or how it is described. Pass the Agent tool's
  `model` or `effort` only where a role says so; otherwise the frontmatter
  applies:
  - Exploration: `Explore` (Haiku 5.5, max).
  - Standard coding: `implementer` (Sonnet 5.5, high), the default for a
    specified change, including a straightforward one across several parts.
  - Specialized demanding coding: `implementer` with `effort: "max"`
    (Sonnet 5.5, max), only when deeper reasoning genuinely helps a
    well-defined change (many interacting cases, an intricate algorithm),
    not because a task is described as demanding.
  - Complex coding: `implementer` with `model: "opus"` and `effort: "high"`
    (Opus 5.5, high), when those factors make the change hard (high reasoning
    complexity, ambiguity, technical uncertainty, intricate dependencies or a
    high implementation risk), even within a single part: the exception to
    AGENTS.md's mid-range model for implementation.
  - Independent verification, only when an independent look adds value beyond
    the deterministic checks: `verifier` (Sonnet 5.5, high).
- Built-in subagents (`general-purpose`, `Plan`) run on the main model at the
  session's effort, Opus at Max: prefer a project agent, or pass `model` and
  `effort` from the roles when spawning one.
- Three concurrent subagents at most: a ceiling, not a target. No Workflow, no
  Ultracode, no agent swarm unless the owner asks for one in the request.
- `isolation: worktree` only when parallel agents would edit the same files.
- A subagent's files under `~/.claude/projects/<project>/<session>/subagents/`
  show what it ran with: `"model"` and `"output_config":{"effort":...}` in
  `agent-*.jsonl`, the `model` and `effort` passed in `agent-*.meta.json`.
  Check them after a change to this setup or a CLI update.
- Context and compaction use the CLI's defaults: a 1M window for Opus, Sonnet
  and Haiku 5.5, auto-compaction at about 967K, no override variable; any
  change to them is the owner's.
