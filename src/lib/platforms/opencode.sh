#!/bin/bash
# dot-agents/lib/platforms/opencode.sh
# OpenCode CLI detection, version, and linking
# OpenCode is actively maintained (v1.1.12+, 59k+ GitHub stars)

# Detect OpenCode CLI version
opencode_detect() {
  if command -v opencode >/dev/null 2>&1; then
    opencode --version 2>/dev/null | head -1
  fi
}

# Check if OpenCode is installed
opencode_is_installed() {
  command -v opencode >/dev/null 2>&1
}

# Get OpenCode version string
opencode_version() {
  opencode_detect
}

# Create links for OpenCode (SYMLINKS - works fine)
opencode_create_links() {
  local project="$1"
  local repo_path="$2"

  # Link opencode.json config if exists
  if [ -f "$AGENTS_HOME/settings/$project/opencode.json" ]; then
    ln -sf "$AGENTS_HOME/settings/$project/opencode.json" "$repo_path/opencode.json"
  elif [ -f "$AGENTS_HOME/settings/global/opencode.json" ]; then
    ln -sf "$AGENTS_HOME/settings/global/opencode.json" "$repo_path/opencode.json"
  fi

  # Create .opencode/agent directory for agent definitions
  mkdir -p "$repo_path/.opencode/agent"

  # Link agent definition files
  if [ -d "$AGENTS_HOME/rules/$project" ]; then
    for agent_file in "$AGENTS_HOME/rules/$project"/opencode-*.md; do
      [ -f "$agent_file" ] || continue
      local basename
      basename=$(basename "$agent_file")
      # Remove opencode- prefix for the target name
      local target_name="${basename#opencode-}"
      ln -sf "$agent_file" "$repo_path/.opencode/agent/$target_name"
    done
  fi

  # Also link global agent definitions
  if [ -d "$AGENTS_HOME/rules/global" ]; then
    for agent_file in "$AGENTS_HOME/rules/global"/opencode-*.md; do
      [ -f "$agent_file" ] || continue
      local basename
      basename=$(basename "$agent_file")
      local target_name="${basename#opencode-}"
      # Only link if doesn't exist from project
      [ -f "$repo_path/.opencode/agent/$target_name" ] || \
        ln -sf "$agent_file" "$repo_path/.opencode/agent/$target_name"
    done
  fi
}

# Check for deprecated formats (OpenCode has been stable - no deprecated formats)
opencode_has_deprecated_format() {
  local repo_path="$1"
  return 1  # No deprecated formats for OpenCode
}

# Get deprecated format details
opencode_deprecated_details() {
  local repo_path="$1"
  # OpenCode has no deprecated formats
  echo ""
}
