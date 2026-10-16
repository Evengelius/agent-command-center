#!/bin/sh
# Hash the agent CLIs' configuration files. Credential files are never opened.
cd "$HOME" || exit 1
{
  for f in .claude/settings.json .claude/settings.local.json .claude/CLAUDE.md .claude/remote-settings.json .claude/keybindings.json .claude.json \
           .codex/config.toml .codex/AGENTS.md .codex/.codex-global-state.json; do
    [ -f "$f" ] && echo "$f"
  done
  find .claude/skills .claude/agents .claude/commands .claude/hooks .claude/plugins -maxdepth 2 -type f -name '*.json' 2>/dev/null
  find .claude/skills .claude/agents .claude/commands .codex/plugins -maxdepth 3 -type f \( -name '*.md' -o -name '*.toml' \) 2>/dev/null
} | grep -v -E 'credentials|auth\.json|oauth|token|secret' | sort -u | while read -r f; do sha256sum "$f"; done
