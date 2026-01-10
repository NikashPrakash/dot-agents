#!/bin/bash
# dot-agents/lib/commands/internal.sh
# Internal state inspection
# Provides read-only access to CLI state data

cmd_internal_help() {
  cat << EOF
${BOLD}dot-agents internal${NC} - Inspect internal CLI state

${BOLD}USAGE${NC}
    dot-agents internal [subcommand]

${BOLD}SUBCOMMANDS${NC}
    (none)            Show summary of internal state
    version           Show CLI version information
    state             Show state.json contents
    paths             Show all important paths
    env               Show environment variables

${BOLD}OPTIONS${NC}
    --json            Output as JSON
    --verbose, -v     Show detailed information
    --help, -h        Show this help

${BOLD}DESCRIPTION${NC}
    Provides read-only access to dot-agents internal state.
    Useful for debugging and understanding the CLI setup.

    State locations:
    - ~/.local/state/dot-agents/     CLI state (if exists)
    - ~/.cache/dot-agents/           Cache data (if exists)
    - ~/.agents/config.json          User configuration

${BOLD}EXAMPLES${NC}
    dot-agents internal              # Summary overview
    dot-agents internal version      # Version info
    dot-agents internal paths        # Show all paths
    dot-agents internal --json       # Full dump as JSON

EOF
}

cmd_internal() {
  local subcommand=""

  # Parse flags
  REMAINING_ARGS=()
  while [[ $# -gt 0 ]]; do
    case $1 in
      --json)
        JSON_OUTPUT=true
        shift
        ;;
      --verbose|-v)
        VERBOSE=true
        shift
        ;;
      --help|-h)
        cmd_internal_help
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

  # Get subcommand from remaining args
  if [ ${#REMAINING_ARGS[@]} -gt 0 ]; then
    subcommand="${REMAINING_ARGS[0]}"
  fi

  if [ "$JSON_OUTPUT" = true ]; then
    internal_json_output
    return 0
  fi

  # Route to subcommand
  case "$subcommand" in
    ""|summary)
      internal_summary
      ;;
    version)
      internal_version
      ;;
    state)
      internal_state
      ;;
    paths)
      internal_paths
      ;;
    env)
      internal_env
      ;;
    *)
      log_error "Unknown subcommand: $subcommand"
      echo ""
      echo "Available subcommands: version, state, paths, env"
      return 1
      ;;
  esac
}

internal_summary() {
  cat << EOF
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
 dot-agents Internal State
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
EOF

  echo ""
  echo -e "${BOLD}Version${NC}"
  echo "  CLI Version:      $DOT_AGENTS_VERSION"
  echo "  Version Date:     $DOT_AGENTS_VERSION_DATE"
  echo ""

  echo -e "${BOLD}Paths${NC}"
  echo "  Agents Home:      $AGENTS_HOME"
  echo "  Config File:      $AGENTS_HOME/config.json"
  local state_dir="${XDG_STATE_HOME:-$HOME/.local/state}/dot-agents"
  echo "  State Dir:        $state_dir"
  local cache_dir="${XDG_CACHE_HOME:-$HOME/.cache}/dot-agents"
  echo "  Cache Dir:        $cache_dir"
  echo ""

  echo -e "${BOLD}Status${NC}"
  if [ -d "$AGENTS_HOME" ]; then
    echo -e "  Agents Home:      ${GREEN}exists${NC}"
  else
    echo -e "  Agents Home:      ${RED}not found${NC}"
  fi

  if [ -f "$AGENTS_HOME/config.json" ]; then
    echo -e "  Config:           ${GREEN}exists${NC}"
    local project_count
    project_count=$(config_list_projects | wc -l | tr -d ' ')
    echo "  Projects:         $project_count registered"
  else
    echo -e "  Config:           ${RED}not found${NC}"
  fi

  if [ -d "$state_dir" ]; then
    echo -e "  State Dir:        ${GREEN}exists${NC}"
  else
    echo -e "  State Dir:        ${DIM}not created${NC}"
  fi

  if [ -d "$cache_dir" ]; then
    echo -e "  Cache Dir:        ${GREEN}exists${NC}"
  else
    echo -e "  Cache Dir:        ${DIM}not created${NC}"
  fi

  echo ""
  echo -e "${BOLD}Installed Agents${NC}"
  internal_detect_agents
}

internal_version() {
  echo -e "${BOLD}dot-agents Version Information${NC}"
  echo ""
  echo "  Version:          $DOT_AGENTS_VERSION"
  echo "  Release Date:     $DOT_AGENTS_VERSION_DATE"
  echo ""

  # Show script location
  local script_path="${BASH_SOURCE[0]}"
  while [ -L "$script_path" ]; do
    script_path=$(readlink "$script_path")
  done
  local script_dir
  script_dir=$(cd "$(dirname "$script_path")" && pwd)

  echo "  Install Location: $(dirname "$(dirname "$script_dir")")"
  echo "  Library Dir:      $script_dir"
  echo ""

  # Check for updates hint
  echo -e "${DIM}To check for updates: brew upgrade dot-agents${NC}"
}

internal_state() {
  local state_dir="${XDG_STATE_HOME:-$HOME/.local/state}/dot-agents"
  local state_file="$state_dir/state.json"

  echo -e "${BOLD}CLI State${NC}"
  echo ""

  if [ -f "$state_file" ]; then
    echo "  State File: $state_file"
    echo ""
    echo "  Contents:"
    if command -v jq >/dev/null 2>&1; then
      jq '.' "$state_file" | sed 's/^/    /'
    else
      cat "$state_file" | sed 's/^/    /'
    fi
  else
    echo -e "  ${DIM}No state.json file (first-time setup or not using XDG state)${NC}"
    echo ""
    echo "  The state directory is created when:"
    echo "    - CLI needs to store internal state"
    echo "    - Migrations are run"
    echo "    - Backups are created"
  fi
}

internal_paths() {
  echo -e "${BOLD}dot-agents Path Configuration${NC}"
  echo ""

  echo "User Content:"
  echo "  AGENTS_HOME:      $AGENTS_HOME"
  echo "    config.json:    $AGENTS_HOME/config.json"
  echo "    rules/:         $AGENTS_HOME/rules/"
  echo "    settings/:      $AGENTS_HOME/settings/"
  echo "    mcp/:           $AGENTS_HOME/mcp/"
  echo "    commands/:      $AGENTS_HOME/commands/"
  echo "    scripts/:       $AGENTS_HOME/scripts/"
  echo ""

  echo "CLI State (XDG):"
  local state_dir="${XDG_STATE_HOME:-$HOME/.local/state}/dot-agents"
  local cache_dir="${XDG_CACHE_HOME:-$HOME/.cache}/dot-agents"
  echo "  XDG_STATE_HOME:   ${XDG_STATE_HOME:-$HOME/.local/state}"
  echo "    State Dir:      $state_dir"
  echo "  XDG_CACHE_HOME:   ${XDG_CACHE_HOME:-$HOME/.cache}"
  echo "    Cache Dir:      $cache_dir"
  echo ""

  echo "CLI Installation:"
  local bin_path
  bin_path=$(command -v dot-agents 2>/dev/null || echo "not in PATH")
  echo "  Binary:           $bin_path"

  # Show actual resolved paths
  if [ "$VERBOSE" = true ]; then
    echo ""
    echo "Resolved Paths:"
    echo "  LIB_DIR:          ${LIB_DIR:-unknown}"
    echo "  SHARE_DIR:        ${SHARE_DIR:-unknown}"
  fi
}

internal_env() {
  echo -e "${BOLD}Environment Variables${NC}"
  echo ""

  echo "dot-agents Variables:"
  echo "  DOT_AGENTS_VERSION:      ${DOT_AGENTS_VERSION:-unset}"
  echo "  DOT_AGENTS_VERSION_DATE: ${DOT_AGENTS_VERSION_DATE:-unset}"
  echo "  AGENTS_HOME:             ${AGENTS_HOME:-unset}"
  echo ""

  echo "XDG Variables:"
  echo "  XDG_STATE_HOME:          ${XDG_STATE_HOME:-unset (using ~/.local/state)}"
  echo "  XDG_CACHE_HOME:          ${XDG_CACHE_HOME:-unset (using ~/.cache)}"
  echo "  XDG_CONFIG_HOME:         ${XDG_CONFIG_HOME:-unset (using ~/.config)}"
  echo ""

  echo "Shell Environment:"
  echo "  SHELL:                   ${SHELL:-unset}"
  echo "  TERM:                    ${TERM:-unset}"
  echo "  HOME:                    ${HOME:-unset}"
  echo ""

  if [ "$VERBOSE" = true ]; then
    echo "Internal Flags:"
    echo "  VERBOSE:                 ${VERBOSE:-false}"
    echo "  DRY_RUN:                 ${DRY_RUN:-false}"
    echo "  FORCE:                   ${FORCE:-false}"
    echo "  YES:                     ${YES:-false}"
    echo "  JSON_OUTPUT:             ${JSON_OUTPUT:-false}"
  fi
}

internal_detect_agents() {
  # Cursor
  echo -n "  Cursor:           "
  if [ -d '/Applications/Cursor.app' ]; then
    local ver
    ver=$(defaults read /Applications/Cursor.app/Contents/Info.plist CFBundleShortVersionString 2>/dev/null || echo "unknown")
    echo -e "${GREEN}installed${NC} (v$ver)"
  elif command -v cursor >/dev/null 2>&1; then
    echo -e "${GREEN}CLI installed${NC}"
  else
    echo -e "${DIM}not detected${NC}"
  fi

  # Claude Code
  echo -n "  Claude Code:      "
  if command -v claude >/dev/null 2>&1; then
    local ver
    ver=$(claude --version 2>/dev/null | head -1 || echo "unknown")
    echo -e "${GREEN}installed${NC} ($ver)"
  else
    echo -e "${DIM}not detected${NC}"
  fi

  # Codex
  echo -n "  Codex:            "
  if command -v codex >/dev/null 2>&1; then
    local ver
    ver=$(codex --version 2>/dev/null | head -1 || echo "unknown")
    echo -e "${GREEN}installed${NC} ($ver)"
  else
    echo -e "${DIM}not detected${NC}"
  fi

  # OpenCode
  echo -n "  OpenCode:         "
  if command -v opencode >/dev/null 2>&1; then
    local ver
    ver=$(opencode --version 2>/dev/null | head -1 || echo "unknown")
    echo -e "${GREEN}installed${NC} ($ver)"
  else
    echo -e "${DIM}not detected${NC}"
  fi
}

internal_json_output() {
  local state_dir="${XDG_STATE_HOME:-$HOME/.local/state}/dot-agents"
  local cache_dir="${XDG_CACHE_HOME:-$HOME/.cache}/dot-agents"

  echo "{"
  echo '  "version": {'
  echo '    "cli_version": "'$DOT_AGENTS_VERSION'",'
  echo '    "version_date": "'$DOT_AGENTS_VERSION_DATE'"'
  echo '  },'

  echo '  "paths": {'
  echo '    "agents_home": "'$AGENTS_HOME'",'
  echo '    "config_file": "'$AGENTS_HOME'/config.json",'
  echo '    "state_dir": "'$state_dir'",'
  echo '    "cache_dir": "'$cache_dir'"'
  echo '  },'

  echo '  "status": {'
  echo '    "agents_home_exists": '$( [ -d "$AGENTS_HOME" ] && echo "true" || echo "false")','
  echo '    "config_exists": '$( [ -f "$AGENTS_HOME/config.json" ] && echo "true" || echo "false")','
  echo '    "state_dir_exists": '$( [ -d "$state_dir" ] && echo "true" || echo "false")','
  echo '    "cache_dir_exists": '$( [ -d "$cache_dir" ] && echo "true" || echo "false")
  echo '  },'

  echo '  "agents": {'
  echo -n '    "cursor": '$( [ -d '/Applications/Cursor.app' ] || command -v cursor >/dev/null 2>&1 && echo "true" || echo "false")
  echo ','
  echo -n '    "claude_code": '$( command -v claude >/dev/null 2>&1 && echo "true" || echo "false")
  echo ','
  echo -n '    "codex": '$( command -v codex >/dev/null 2>&1 && echo "true" || echo "false")
  echo ','
  echo -n '    "opencode": '$( command -v opencode >/dev/null 2>&1 && echo "true" || echo "false")
  echo ""
  echo '  },'

  # Project count
  local project_count=0
  if [ -f "$AGENTS_HOME/config.json" ]; then
    project_count=$(config_list_projects 2>/dev/null | wc -l | tr -d ' ')
  fi
  echo '  "project_count": '$project_count

  echo "}"
}
