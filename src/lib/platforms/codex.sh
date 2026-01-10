#!/bin/bash
# dot-agents/lib/platforms/codex.sh
# OpenAI Codex CLI detection, version, and linking

# Detect Codex CLI version
codex_detect() {
  if command -v codex >/dev/null 2>&1; then
    codex --version 2>/dev/null | head -1
  fi
}

# Check if Codex is installed
codex_is_installed() {
  command -v codex >/dev/null 2>&1
}

# Get Codex version string
codex_version() {
  codex_detect
}

# Create links for Codex (SYMLINKS - works fine)
codex_create_links() {
  local project="$1"
  local repo_path="$2"

  # Link AGENTS.md from global rules if it exists
  if [ -f "$AGENTS_HOME/rules/global/agents.md" ]; then
    ln -sf "$AGENTS_HOME/rules/global/agents.md" "$repo_path/AGENTS.md"
  elif [ -f "$AGENTS_HOME/rules/global/rules.md" ]; then
    # Fall back to global rules.md if no agents-specific file
    ln -sf "$AGENTS_HOME/rules/global/rules.md" "$repo_path/AGENTS.md"
  fi

  # Project-specific AGENTS.md
  if [ -f "$AGENTS_HOME/rules/$project/agents.md" ]; then
    # If project has its own agents.md, use it instead
    ln -sf "$AGENTS_HOME/rules/$project/agents.md" "$repo_path/AGENTS.md"
  fi

  # Create .codex directory for config
  mkdir -p "$repo_path/.codex"

  # Link TOML config if exists (Codex uses TOML, not JSON)
  if [ -f "$AGENTS_HOME/settings/$project/codex.toml" ]; then
    ln -sf "$AGENTS_HOME/settings/$project/codex.toml" "$repo_path/.codex/config.toml"
  elif [ -f "$AGENTS_HOME/settings/global/codex.toml" ]; then
    ln -sf "$AGENTS_HOME/settings/global/codex.toml" "$repo_path/.codex/config.toml"
  fi
}

# Check for deprecated formats (Codex has been stable - no deprecated formats)
codex_has_deprecated_format() {
  local repo_path="$1"
  return 1  # No deprecated formats for Codex
}

# Get deprecated format details
codex_deprecated_details() {
  local repo_path="$1"
  # Codex has no deprecated formats
  echo ""
}
