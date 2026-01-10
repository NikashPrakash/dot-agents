#!/bin/bash
# dot-agents/lib/platforms/claude-code.sh
# Claude Code CLI detection, version, and linking

# Detect Claude Code CLI version
claude_detect() {
  if command -v claude >/dev/null 2>&1; then
    claude --version 2>/dev/null | head -1
  fi
}

# Check if Claude Code is installed
claude_is_installed() {
  command -v claude >/dev/null 2>&1
}

# Get Claude Code version string
claude_version() {
  claude_detect
}

# Create links for Claude Code (SYMLINKS - works fine)
claude_create_links() {
  local project="$1"
  local repo_path="$2"

  # Link CLAUDE.md from global rules if it exists
  if [ -f "$AGENTS_HOME/rules/global/claude.md" ]; then
    ln -sf "$AGENTS_HOME/rules/global/claude.md" "$repo_path/CLAUDE.md"
  elif [ -f "$AGENTS_HOME/rules/global/rules.md" ]; then
    # Fall back to global rules.md if no claude-specific file
    ln -sf "$AGENTS_HOME/rules/global/rules.md" "$repo_path/CLAUDE.md"
  fi

  # Project-specific CLAUDE.md
  if [ -f "$AGENTS_HOME/rules/$project/claude.md" ]; then
    # If project has its own claude.md, use it instead
    ln -sf "$AGENTS_HOME/rules/$project/claude.md" "$repo_path/CLAUDE.md"
  fi

  # Create .claude directory for settings
  mkdir -p "$repo_path/.claude"

  # Link settings.local.json if exists
  if [ -f "$AGENTS_HOME/settings/$project/claude-code.json" ]; then
    ln -sf "$AGENTS_HOME/settings/$project/claude-code.json" "$repo_path/.claude/settings.local.json"
  fi

  # Link MCP config if exists
  if [ -f "$AGENTS_HOME/mcp/$project/claude.json" ]; then
    ln -sf "$AGENTS_HOME/mcp/$project/claude.json" "$repo_path/.mcp.json"
  elif [ -f "$AGENTS_HOME/mcp/global/claude.json" ]; then
    ln -sf "$AGENTS_HOME/mcp/global/claude.json" "$repo_path/.mcp.json"
  fi
}

# Check for deprecated .claude.json file
claude_has_deprecated_format() {
  local repo_path="$1"
  [ -f "$repo_path/.claude.json" ]
}

# Get deprecated format details
claude_deprecated_details() {
  local repo_path="$1"

  if [ -f "$repo_path/.claude.json" ]; then
    echo ".claude.json → .claude/settings.json"
  fi
}
