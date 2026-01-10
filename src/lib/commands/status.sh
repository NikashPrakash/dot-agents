#!/bin/bash
# dot-agents/lib/commands/status.sh
# Show status of managed projects

cmd_status_help() {
  cat << EOF
${BOLD}dot-agents status${NC} - Show status of managed projects

${BOLD}USAGE${NC}
    dot-agents status [options]

${BOLD}OPTIONS${NC}
    --json            Output in JSON format
    --verbose, -v     Show detailed information
    --help, -h        Show this help

${BOLD}DESCRIPTION${NC}
    Shows all projects registered with dot-agents and their current status.

    Status indicators:
    - [OK]    Project directory exists and configs are linked
    - [WARN]  Project has issues (missing links, outdated config)
    - [ERR]   Project directory not found

${BOLD}EXAMPLES${NC}
    dot-agents status          # Show all projects
    dot-agents status --json   # Output as JSON

EOF
}

cmd_status() {
  # Parse flags
  parse_common_flags "$@"
  set -- "${REMAINING_ARGS[@]+"${REMAINING_ARGS[@]}"}"

  # Show help if requested
  if [ "${SHOW_HELP:-false}" = true ]; then
    cmd_status_help
    return 0
  fi

  local config_file="$AGENTS_HOME/config.json"

  # Check if config exists
  if [ ! -f "$config_file" ]; then
    if [ "$JSON_OUTPUT" = true ]; then
      echo '{"error": "Not initialized. Run dot-agents init first."}'
    else
      log_error "Not initialized. Run 'dot-agents init' first."
    fi
    return 1
  fi

  # Get projects list
  local projects
  projects=$(config_list_projects)

  if [ "$JSON_OUTPUT" = true ]; then
    output_status_json "$projects"
  else
    output_status_text "$projects"
  fi
}

output_status_json() {
  local projects="$1"
  local config_file="$AGENTS_HOME/config.json"

  echo "{"
  echo '  "version": "'$DOT_AGENTS_VERSION'",'
  echo '  "agents_home": "'$AGENTS_HOME'",'
  echo '  "projects": {'

  local first=true
  while IFS= read -r name; do
    [ -z "$name" ] && continue

    local path
    path=$(config_get_project_path "$name")
    local status="ok"
    local issues=""

    # Check project status
    if [ ! -d "$path" ]; then
      status="error"
      issues="directory not found"
    else
      # Check for symlink issues
      local link_issues
      link_issues=$(check_project_links "$name" "$path")
      if [ -n "$link_issues" ]; then
        status="warn"
        issues="$link_issues"
      fi
    fi

    if [ "$first" = true ]; then
      first=false
    else
      echo ","
    fi

    echo -n '    "'$name'": {"path": "'$path'", "status": "'$status'"'
    if [ -n "$issues" ]; then
      echo -n ', "issues": "'$issues'"'
    fi
    echo -n "}"
  done <<< "$projects"

  echo ""
  echo "  }"
  echo "}"
}

output_status_text() {
  local projects="$1"

  log_header "dot-agents status"
  log_info "Home: $AGENTS_HOME"
  echo ""

  # Count projects
  local count=0
  while IFS= read -r name; do
    [ -n "$name" ] && ((count++)) || true
  done <<< "$projects"

  if [ $count -eq 0 ]; then
    log_info "No projects registered."
    echo ""
    echo "Add a project with: dot-agents add <path>"
    return 0
  fi

  echo "Projects ($count):"
  echo ""

  # Display each project
  while IFS= read -r name; do
    [ -z "$name" ] && continue

    local path
    path=$(config_get_project_path "$name")
    local display_path="${path/#$HOME/~}"

    # Check project status
    if [ ! -d "$path" ]; then
      echo -e "  ${RED}[ERR]${NC}  $name"
      echo -e "         ${DIM}$display_path${NC} (not found)"
    else
      # Check for link issues
      local issues_output
      issues_output=$(check_project_links_verbose "$name" "$path")

      if [ -z "$issues_output" ]; then
        echo -e "  ${GREEN}[OK]${NC}   $name"
        echo -e "         ${DIM}$display_path${NC}"
      else
        echo -e "  ${YELLOW}[WARN]${NC} $name"
        echo -e "         ${DIM}$display_path${NC}"
        while IFS= read -r issue; do
          echo -e "         ${YELLOW}↳${NC} $issue"
        done <<< "$issues_output"
      fi
    fi
    echo ""
  done <<< "$projects"
}

# Check project links and return comma-separated issues
# Usage: issues=$(check_project_links PROJECT_NAME PROJECT_PATH)
check_project_links() {
  local name="$1"
  local path="$2"
  local issues=""

  # Check .cursor/rules exists
  if [ ! -d "$path/.cursor/rules" ]; then
    issues="missing .cursor/rules/"
  fi

  # Check for CLAUDE.md if claude rules exist
  if [ -f "$AGENTS_HOME/rules/global/claude-code.mdc" ] || [ -f "$AGENTS_HOME/rules/$name/claude-code.mdc" ]; then
    if [ ! -e "$path/CLAUDE.md" ]; then
      [ -n "$issues" ] && issues="$issues,"
      issues="${issues}missing CLAUDE.md"
    fi
  fi

  echo "$issues"
}

# More detailed check for text output - outputs one issue per line
# Usage: check_project_links_verbose PROJECT_NAME PROJECT_PATH
check_project_links_verbose() {
  local name="$1"
  local path="$2"

  # Check .cursor/rules exists and has files
  if [ ! -d "$path/.cursor/rules" ]; then
    echo "missing .cursor/rules/"
  elif [ -z "$(ls -A "$path/.cursor/rules" 2>/dev/null)" ]; then
    echo ".cursor/rules/ is empty"
  fi

  # Check for CLAUDE.md
  if [ -f "$AGENTS_HOME/rules/global/claude-code.mdc" ] || [ -f "$AGENTS_HOME/rules/$name/claude-code.mdc" ]; then
    if [ ! -e "$path/CLAUDE.md" ]; then
      echo "missing CLAUDE.md"
    elif [ -L "$path/CLAUDE.md" ]; then
      local target
      target=$(readlink "$path/CLAUDE.md")
      if [ ! -f "$target" ]; then
        echo "CLAUDE.md → broken symlink"
      fi
    fi
  fi

  # Check for AGENTS.md (Codex)
  if [ -f "$AGENTS_HOME/rules/global/agents.md" ] || [ -f "$AGENTS_HOME/rules/$name/agents.md" ]; then
    if [ ! -e "$path/AGENTS.md" ]; then
      echo "missing AGENTS.md"
    fi
  fi
}
