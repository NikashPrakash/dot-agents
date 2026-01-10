#!/bin/bash
# dot-agents/lib/commands/context.sh
# Agent-friendly context dump

cmd_context_help() {
  cat << EOF
${BOLD}dot-agents context${NC} - Output configuration context for AI agents

${BOLD}USAGE${NC}
    dot-agents context [options]
    dot-agents context [project]

${BOLD}ARGUMENTS${NC}
    [project]         Show context for a specific project (optional)

${BOLD}OPTIONS${NC}
    --full            Include file contents (rules, configs)
    --compact         Minimal output (paths only)
    --verbose, -v     Show detailed information
    --help, -h        Show this help

${BOLD}DESCRIPTION${NC}
    Outputs a JSON dump of the dot-agents configuration that AI coding
    agents can parse to understand the current setup.

    Includes:
    - Registered projects and paths
    - Available rules (global and per-project)
    - Agent configurations
    - Feature flags

    The output is designed to be consumed by AI agents for:
    - Understanding project context
    - Finding relevant configuration files
    - Debugging configuration issues

${BOLD}EXAMPLES${NC}
    dot-agents context                # Full context as JSON
    dot-agents context myproject      # Context for specific project
    dot-agents context --full         # Include file contents
    dot-agents context --compact      # Minimal paths only

EOF
}

cmd_context() {
  local project_filter=""
  local full_output=false
  local compact_output=false

  # Parse flags
  REMAINING_ARGS=()
  while [[ $# -gt 0 ]]; do
    case $1 in
      --full)
        full_output=true
        shift
        ;;
      --compact)
        compact_output=true
        shift
        ;;
      --verbose|-v)
        VERBOSE=true
        shift
        ;;
      --help|-h)
        cmd_context_help
        return 0
        ;;
      -*)
        log_error "Unknown option: $1"
        return 1
        ;;
      *)
        REMAINING_ARGS+=("$1")
        shift
        ;;
    esac
  done

  # Get project filter from remaining args
  if [ ${#REMAINING_ARGS[@]} -gt 0 ]; then
    project_filter="${REMAINING_ARGS[0]}"

    # Validate project exists
    local path
    path=$(config_get_project_path "$project_filter")
    if [ -z "$path" ]; then
      log_error "Project not found: $project_filter" >&2
      return 1
    fi
  fi

  if [ "$compact_output" = true ]; then
    output_context_compact "$project_filter"
  elif [ "$full_output" = true ]; then
    output_context_full "$project_filter"
  else
    output_context_standard "$project_filter"
  fi
}

# Standard context output
output_context_standard() {
  local project_filter="$1"

  echo "{"
  echo '  "version": "'$DOT_AGENTS_VERSION'",'
  echo '  "agents_home": "'$AGENTS_HOME'",'
  echo '  "timestamp": "'$(date -u +"%Y-%m-%dT%H:%M:%SZ")'",'

  # Projects section
  echo '  "projects": {'
  local projects
  if [ -n "$project_filter" ]; then
    projects="$project_filter"
  else
    projects=$(config_list_projects)
  fi

  local first_project=true
  while IFS= read -r name; do
    [ -z "$name" ] && continue

    local path
    path=$(config_get_project_path "$name")
    local exists="true"
    [ ! -d "$path" ] && exists="false"

    [ "$first_project" = true ] && first_project=false || echo ","
    echo -n '    "'$name'": {'
    echo -n '"path": "'$path'"'
    echo -n ', "exists": '$exists
    echo -n '}'
  done <<< "$projects"
  echo ""
  echo '  },'

  # Global rules
  echo '  "global_rules": ['
  output_rules_list "$AGENTS_HOME/rules/global"
  echo '  ],'

  # Global settings
  echo '  "global_settings": ['
  output_settings_list "$AGENTS_HOME/settings/global"
  echo '  ],'

  # Agents detected
  echo '  "agents": {'
  output_agents_info
  echo '  },'

  # Feature flags (from config.json)
  echo '  "features": {'
  output_features_info
  echo '  }'

  echo "}"
}

# Compact context output (paths only)
output_context_compact() {
  local project_filter="$1"

  echo "{"
  echo '  "agents_home": "'$AGENTS_HOME'",'

  # Projects as simple object
  echo '  "projects": {'
  local projects
  if [ -n "$project_filter" ]; then
    projects="$project_filter"
  else
    projects=$(config_list_projects)
  fi

  local first=true
  while IFS= read -r name; do
    [ -z "$name" ] && continue
    local path
    path=$(config_get_project_path "$name")
    [ "$first" = true ] && first=false || echo ","
    echo -n '    "'$name'": "'$path'"'
  done <<< "$projects"
  echo ""
  echo '  },'

  # Rules as simple array of paths
  echo '  "global_rules": ['
  local first_rule=true
  if [ -d "$AGENTS_HOME/rules/global" ]; then
    shopt -s nullglob
    for rule in "$AGENTS_HOME/rules/global"/*.mdc; do
      [ "$first_rule" = true ] && first_rule=false || echo ","
      echo -n '    "'$rule'"'
    done
    shopt -u nullglob
  fi
  echo ""
  echo '  ]'

  echo "}"
}

# Full context output (includes file contents)
output_context_full() {
  local project_filter="$1"

  echo "{"
  echo '  "version": "'$DOT_AGENTS_VERSION'",'
  echo '  "agents_home": "'$AGENTS_HOME'",'
  echo '  "timestamp": "'$(date -u +"%Y-%m-%dT%H:%M:%SZ")'",'

  # Projects section with detailed info
  echo '  "projects": {'
  local projects
  if [ -n "$project_filter" ]; then
    projects="$project_filter"
  else
    projects=$(config_list_projects)
  fi

  local first_project=true
  while IFS= read -r name; do
    [ -z "$name" ] && continue

    local path
    path=$(config_get_project_path "$name")

    [ "$first_project" = true ] && first_project=false || echo ","
    echo '    "'$name'": {'
    echo '      "path": "'$path'",'
    echo '      "exists": '$( [ -d "$path" ] && echo "true" || echo "false")','

    # Project-specific rules
    echo '      "rules": ['
    output_rules_with_content "$AGENTS_HOME/rules/$name"
    echo '      ],'

    # Applied links
    echo '      "applied_links": {'
    output_applied_links "$name" "$path"
    echo '      }'
    echo -n '    }'
  done <<< "$projects"
  echo ""
  echo '  },'

  # Global rules with content
  echo '  "global_rules": ['
  output_rules_with_content "$AGENTS_HOME/rules/global"
  echo '  ],'

  # Agents
  echo '  "agents": {'
  output_agents_info
  echo '  }'

  echo "}"
}

# Output list of rules (name, path)
output_rules_list() {
  local dir="$1"
  local first=true

  if [ -d "$dir" ]; then
    shopt -s nullglob
    for rule in "$dir"/*.mdc "$dir"/*.md; do
      [ -f "$rule" ] || continue
      local basename
      basename=$(basename "$rule")
      local display_path="${rule/#$HOME/~}"

      [ "$first" = true ] && first=false || echo ","
      echo -n '    {"name": "'$basename'", "path": "'$display_path'"}'
    done
    shopt -u nullglob
  fi
  echo ""
}

# Output list of settings
output_settings_list() {
  local dir="$1"
  local first=true

  if [ -d "$dir" ]; then
    shopt -s nullglob
    for file in "$dir"/*.json "$dir"/*.yaml "$dir"/*.yml; do
      [ -f "$file" ] || continue
      local basename
      basename=$(basename "$file")
      local display_path="${file/#$HOME/~}"

      [ "$first" = true ] && first=false || echo ","
      echo -n '    {"name": "'$basename'", "path": "'$display_path'"}'
    done
    shopt -u nullglob
  fi
  echo ""
}

# Output rules with file content
output_rules_with_content() {
  local dir="$1"
  local first=true

  if [ -d "$dir" ]; then
    shopt -s nullglob
    for rule in "$dir"/*.mdc "$dir"/*.md; do
      [ -f "$rule" ] || continue
      local basename
      basename=$(basename "$rule")
      local content
      content=$(cat "$rule" 2>/dev/null | jq -Rs '.' 2>/dev/null || echo '""')

      [ "$first" = true ] && first=false || echo ","
      echo '      {'
      echo '        "name": "'$basename'",'
      echo '        "path": "'$rule'",'
      echo '        "content": '$content
      echo -n '      }'
    done
    shopt -u nullglob
  fi
  echo ""
}

# Output applied links for a project
output_applied_links() {
  local name="$1"
  local path="$2"
  local first=true

  # Check Cursor rules
  if [ -d "$path/.cursor/rules" ]; then
    shopt -s nullglob
    for rule in "$path/.cursor/rules"/*.mdc; do
      local basename
      basename=$(basename "$rule")
      [ "$first" = true ] && first=false || echo ","
      echo -n '        ".cursor/rules/'$basename'": "hardlink"'
    done
    shopt -u nullglob
  fi

  # Check CLAUDE.md
  if [ -L "$path/CLAUDE.md" ]; then
    local target
    target=$(readlink "$path/CLAUDE.md")
    [ "$first" = true ] && first=false || echo ","
    echo -n '        "CLAUDE.md": "'$target'"'
  fi

  # Check AGENTS.md
  if [ -L "$path/AGENTS.md" ]; then
    local target
    target=$(readlink "$path/AGENTS.md")
    [ "$first" = true ] && first=false || echo ","
    echo -n '        "AGENTS.md": "'$target'"'
  fi

  echo ""
}

# Output agent detection info
output_agents_info() {
  local first=true

  # Cursor
  echo -n '    "cursor": {'
  if [ -d '/Applications/Cursor.app' ] || command -v cursor >/dev/null 2>&1; then
    local ver
    ver=$(cursor --version 2>/dev/null | head -1 || echo "unknown")
    echo -n '"installed": true, "version": "'$ver'"'
  else
    echo -n '"installed": false'
  fi
  echo '},'

  # Claude Code
  echo -n '    "claude-code": {'
  if command -v claude >/dev/null 2>&1; then
    local ver
    ver=$(claude --version 2>/dev/null | head -1 || echo "unknown")
    echo -n '"installed": true, "version": "'$ver'"'
  else
    echo -n '"installed": false'
  fi
  echo '},'

  # Codex
  echo -n '    "codex": {'
  if command -v codex >/dev/null 2>&1; then
    local ver
    ver=$(codex --version 2>/dev/null | head -1 || echo "unknown")
    echo -n '"installed": true, "version": "'$ver'"'
  else
    echo -n '"installed": false'
  fi
  echo '}'
}

# Output feature flags
output_features_info() {
  local config_file="$AGENTS_HOME/config.json"

  if [ -f "$config_file" ] && command -v jq >/dev/null 2>&1; then
    # Check for feature flags in config
    local tasks_enabled history_enabled
    tasks_enabled=$(jq -r '.features.tasks // false' "$config_file" 2>/dev/null)
    history_enabled=$(jq -r '.features.history // false' "$config_file" 2>/dev/null)

    echo -n '    "tasks": '$tasks_enabled','
    echo ""
    echo -n '    "history": '$history_enabled
  else
    echo -n '    "tasks": false,'
    echo ""
    echo -n '    "history": false'
  fi
  echo ""
}
