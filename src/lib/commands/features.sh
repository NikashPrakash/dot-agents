#!/bin/bash
# dot-agents/lib/commands/features.sh
# Manage opt-in features (tasks, history, sync)

# Get feature description
_get_feature_description() {
  local feature="$1"
  case "$feature" in
    tasks) echo "Track tasks across projects (local-first)" ;;
    history) echo "Log agent activity for observability" ;;
    sync) echo "Enable git sync reminders and helpers" ;;
    *) echo "Unknown feature" ;;
  esac
}

cmd_features_help() {
  cat << EOF
${BOLD}dot-agents features${NC} - Manage opt-in features

${BOLD}USAGE${NC}
    dot-agents features [command] [options]

${BOLD}COMMANDS${NC}
    (none)              List all features and their status
    enable <feature>    Enable a feature
    disable <feature>   Disable a feature

${BOLD}OPTIONS${NC}
    --json              Output in JSON format
    --help, -h          Show this help

${BOLD}AVAILABLE FEATURES${NC}
    tasks       Track tasks across projects (local-first)
    history     Log agent activity for observability
    sync        Enable git sync reminders and helpers

${BOLD}EXAMPLES${NC}
    dot-agents features                  # List all features
    dot-agents features enable tasks     # Enable task tracking
    dot-agents features disable history  # Disable history logging
    dot-agents features --json           # JSON output

EOF
}

cmd_features() {
  # Parse flags
  parse_common_flags "$@"
  set -- "${REMAINING_ARGS[@]+"${REMAINING_ARGS[@]}"}"

  # Show help if requested
  if [ "${SHOW_HELP:-false}" = true ]; then
    cmd_features_help
    return 0
  fi

  local config_file="$AGENTS_HOME/config.json"

  # Check if initialized
  if [ ! -f "$config_file" ]; then
    if [ "$JSON_OUTPUT" = true ]; then
      echo '{"error": "Not initialized. Run dot-agents init first."}'
    else
      log_error "Not initialized. Run 'dot-agents init' first."
    fi
    return 1
  fi

  # Check for jq (required for this command)
  if ! command -v jq &>/dev/null; then
    if [ "$JSON_OUTPUT" = true ]; then
      echo '{"error": "jq is required for the features command."}'
    else
      log_error "jq is required for the features command."
      log_info "Install with: brew install jq (macOS) or apt install jq (Linux)"
    fi
    return 1
  fi

  # Get subcommand
  local subcommand="${1:-}"

  case "$subcommand" in
    enable)
      shift
      features_enable "$@"
      ;;
    disable)
      shift
      features_disable "$@"
      ;;
    "")
      features_list
      ;;
    *)
      log_error "Unknown subcommand: $subcommand"
      log_info "Run 'dot-agents features --help' for usage."
      return 1
      ;;
  esac
}

features_list() {
  local config_file="$AGENTS_HOME/config.json"

  if [ "$JSON_OUTPUT" = true ]; then
    features_list_json
  else
    features_list_text
  fi
}

features_list_json() {
  local config_file="$AGENTS_HOME/config.json"

  # Read current feature states
  local tasks_enabled history_enabled sync_enabled
  tasks_enabled=$(jq -r '.features.tasks // false' "$config_file")
  history_enabled=$(jq -r '.features.history // false' "$config_file")
  sync_enabled=$(jq -r '.features.sync // false' "$config_file")

  cat << EOF
{
  "features": {
    "tasks": {
      "enabled": $tasks_enabled,
      "description": "$(_get_feature_description tasks)"
    },
    "history": {
      "enabled": $history_enabled,
      "description": "$(_get_feature_description history)"
    },
    "sync": {
      "enabled": $sync_enabled,
      "description": "$(_get_feature_description sync)"
    }
  }
}
EOF
}

features_list_text() {
  local config_file="$AGENTS_HOME/config.json"

  log_header "dot-agents features"
  echo ""

  # Read current feature states
  local tasks_enabled history_enabled sync_enabled
  tasks_enabled=$(jq -r '.features.tasks // false' "$config_file")
  history_enabled=$(jq -r '.features.history // false' "$config_file")
  sync_enabled=$(jq -r '.features.sync // false' "$config_file")

  echo "Opt-in features (enable with: dot-agents features enable <name>)"
  echo ""

  # Display each feature
  _display_feature "tasks" "$tasks_enabled" "$(_get_feature_description tasks)"
  _display_feature "history" "$history_enabled" "$(_get_feature_description history)"
  _display_feature "sync" "$sync_enabled" "$(_get_feature_description sync)"

  echo ""
}

_display_feature() {
  local name="$1"
  local enabled="$2"
  local description="$3"

  if [ "$enabled" = "true" ]; then
    echo -e "  ${GREEN}●${NC} ${BOLD}$name${NC} ${GREEN}(enabled)${NC}"
  else
    echo -e "  ${DIM}○${NC} ${BOLD}$name${NC} ${DIM}(disabled)${NC}"
  fi
  echo -e "    ${DIM}$description${NC}"
  echo ""
}

features_enable() {
  local feature="$1"

  if [ -z "$feature" ]; then
    log_error "Missing feature name."
    log_info "Usage: dot-agents features enable <feature>"
    log_info "Available: tasks, history, sync"
    return 1
  fi

  # Validate feature name
  if ! _is_valid_feature "$feature"; then
    log_error "Unknown feature: $feature"
    log_info "Available features: tasks, history, sync"
    return 1
  fi

  local config_file="$AGENTS_HOME/config.json"

  # Check if already enabled
  local current_state
  current_state=$(jq -r ".features.$feature // false" "$config_file")

  if [ "$current_state" = "true" ]; then
    if [ "$JSON_OUTPUT" = true ]; then
      echo "{\"feature\": \"$feature\", \"status\": \"already_enabled\"}"
    else
      log_info "Feature '$feature' is already enabled."
    fi
    return 0
  fi

  if [ "$DRY_RUN" = true ]; then
    log_dry "enable feature '$feature' in config.json"
    return 0
  fi

  # Update config
  local json
  json=$(cat "$config_file")
  json=$(echo "$json" | jq ".features.$feature = true")
  echo "$json" | jq '.' > "$config_file"

  if [ "$JSON_OUTPUT" = true ]; then
    echo "{\"feature\": \"$feature\", \"status\": \"enabled\"}"
  else
    log_success "Enabled feature: $feature"
    _show_feature_next_steps "$feature"
  fi
}

features_disable() {
  local feature="$1"

  if [ -z "$feature" ]; then
    log_error "Missing feature name."
    log_info "Usage: dot-agents features disable <feature>"
    return 1
  fi

  # Validate feature name
  if ! _is_valid_feature "$feature"; then
    log_error "Unknown feature: $feature"
    log_info "Available features: tasks, history, sync"
    return 1
  fi

  local config_file="$AGENTS_HOME/config.json"

  # Check if already disabled
  local current_state
  current_state=$(jq -r ".features.$feature // false" "$config_file")

  if [ "$current_state" = "false" ]; then
    if [ "$JSON_OUTPUT" = true ]; then
      echo "{\"feature\": \"$feature\", \"status\": \"already_disabled\"}"
    else
      log_info "Feature '$feature' is already disabled."
    fi
    return 0
  fi

  if [ "$DRY_RUN" = true ]; then
    log_dry "disable feature '$feature' in config.json"
    return 0
  fi

  # Update config
  local json
  json=$(cat "$config_file")
  json=$(echo "$json" | jq ".features.$feature = false")
  echo "$json" | jq '.' > "$config_file"

  if [ "$JSON_OUTPUT" = true ]; then
    echo "{\"feature\": \"$feature\", \"status\": \"disabled\"}"
  else
    log_success "Disabled feature: $feature"
  fi
}

_is_valid_feature() {
  local feature="$1"
  case "$feature" in
    tasks|history|sync) return 0 ;;
    *) return 1 ;;
  esac
}

_show_feature_next_steps() {
  local feature="$1"

  echo ""
  case "$feature" in
    tasks)
      log_info "Next steps:"
      echo "  • Use 'dot-agents tasks' to view tasks (coming soon)"
      echo "  • Use 'dot-agents tasks add \"description\"' to add tasks"
      ;;
    history)
      log_info "Next steps:"
      echo "  • Agent activity will be logged to ~/.agents/history/"
      echo "  • Use 'dot-agents history' to view activity (coming soon)"
      ;;
    sync)
      log_info "Next steps:"
      echo "  • Use 'dot-agents sync init' to set up git sync"
      echo "  • Use 'dot-agents sync status' to check sync state"
      ;;
  esac
}
