#!/bin/bash
# dot-agents/lib/commands/migrate.sh
# Migrate deprecated config formats to current standards

cmd_migrate_help() {
  cat << EOF
${BOLD}dot-agents migrate${NC} - Migrate deprecated config formats

${BOLD}USAGE${NC}
    dot-agents migrate <subcommand> [options]

${BOLD}SUBCOMMANDS${NC}
    detect              Scan registered projects for deprecated formats
    cursorrules [path]  Convert .cursorrules → .cursor/rules/*.mdc
    claude-json [path]  Convert .claude.json → .claude/settings.json

${BOLD}OPTIONS${NC}
    --dry-run           Show what would be done without making changes
    --json              Output in JSON format
    --help, -h          Show this help

${BOLD}DEPRECATED FORMATS${NC}
    .cursorrules        → .cursor/rules/*.mdc (Cursor v0.45+)
    .claude.json        → .claude/settings.json (Claude Code v2.0.67+)

${BOLD}EXAMPLES${NC}
    dot-agents migrate detect                         # Scan all projects
    dot-agents migrate cursorrules --dry-run          # Preview .cursorrules migration
    dot-agents migrate cursorrules ~/Github/project   # Migrate specific project
    dot-agents migrate claude-json                    # Migrate .claude.json

EOF
}

cmd_migrate() {
  # Parse flags
  parse_common_flags "$@"
  set -- "${REMAINING_ARGS[@]+"${REMAINING_ARGS[@]}"}"

  # Show help if requested or no subcommand
  if [ "${SHOW_HELP:-false}" = true ] || [ $# -eq 0 ]; then
    cmd_migrate_help
    return 0
  fi

  local subcommand="$1"
  shift

  case "$subcommand" in
    detect)
      migrate_detect "$@"
      ;;
    cursorrules)
      migrate_cursorrules "$@"
      ;;
    claude-json)
      migrate_claude_json "$@"
      ;;
    *)
      log_error "Unknown subcommand: $subcommand"
      log_info "Run 'dot-agents migrate --help' for usage."
      return 1
      ;;
  esac
}

# Detect deprecated formats in registered projects
migrate_detect() {
  local config_file="$AGENTS_HOME/config.json"

  if [ ! -f "$config_file" ]; then
    log_error "Not initialized. Run 'dot-agents init' first."
    return 1
  fi

  if [ "$JSON_OUTPUT" = true ]; then
    migrate_detect_json
  else
    migrate_detect_text
  fi
}

migrate_detect_text() {
  local config_file="$AGENTS_HOME/config.json"
  local found_count=0

  log_header "dot-agents migrate detect"
  echo ""
  echo "Scanning registered projects for deprecated config formats..."
  echo ""

  # Get list of projects
  local projects
  projects=$(jq -r '.projects | keys[]' "$config_file" 2>/dev/null)

  if [ -z "$projects" ]; then
    log_info "No projects registered."
    return 0
  fi

  for project in $projects; do
    local project_path
    project_path=$(jq -r ".projects[\"$project\"].path" "$config_file")
    project_path=$(expand_path "$project_path")

    if [ ! -d "$project_path" ]; then
      continue
    fi

    local project_has_deprecated=false

    # Check for .cursorrules
    if [ -f "$project_path/.cursorrules" ]; then
      if [ "$project_has_deprecated" = false ]; then
        echo -e "${BOLD}$project${NC} ($project_path)"
        project_has_deprecated=true
      fi
      echo -e "  ${YELLOW}⚠${NC}  .cursorrules ${DIM}(deprecated)${NC}"
      echo -e "      ${DIM}→ Run: dot-agents migrate cursorrules $project_path${NC}"
      ((found_count++))
    fi

    # Check for .claude.json
    if [ -f "$project_path/.claude.json" ]; then
      if [ "$project_has_deprecated" = false ]; then
        echo -e "${BOLD}$project${NC} ($project_path)"
        project_has_deprecated=true
      fi
      echo -e "  ${YELLOW}⚠${NC}  .claude.json ${DIM}(deprecated)${NC}"
      echo -e "      ${DIM}→ Run: dot-agents migrate claude-json $project_path${NC}"
      ((found_count++))
    fi

    if [ "$project_has_deprecated" = true ]; then
      echo ""
    fi
  done

  # Summary
  if [ $found_count -eq 0 ]; then
    echo -e "${GREEN}✓${NC} No deprecated formats found."
  else
    echo "────────────────────────────────────────────────────"
    echo -e "Found ${YELLOW}$found_count${NC} deprecated config(s)"
  fi
}

migrate_detect_json() {
  local config_file="$AGENTS_HOME/config.json"
  local found_items=()

  # Get list of projects
  local projects
  projects=$(jq -r '.projects | keys[]' "$config_file" 2>/dev/null)

  for project in $projects; do
    local project_path
    project_path=$(jq -r ".projects[\"$project\"].path" "$config_file")
    project_path=$(expand_path "$project_path")

    if [ ! -d "$project_path" ]; then
      continue
    fi

    # Check for .cursorrules
    if [ -f "$project_path/.cursorrules" ]; then
      found_items+=("{\"project\":\"$project\",\"path\":\"$project_path\",\"format\":\".cursorrules\",\"migration\":\"cursorrules\"}")
    fi

    # Check for .claude.json
    if [ -f "$project_path/.claude.json" ]; then
      found_items+=("{\"project\":\"$project\",\"path\":\"$project_path\",\"format\":\".claude.json\",\"migration\":\"claude-json\"}")
    fi
  done

  # Output JSON
  echo "{"
  echo "  \"deprecated\": ["
  local first=true
  for item in "${found_items[@]}"; do
    if [ "$first" = true ]; then
      first=false
    else
      echo ","
    fi
    echo -n "    $item"
  done
  echo ""
  echo "  ],"
  echo "  \"count\": ${#found_items[@]}"
  echo "}"
}

# Migrate .cursorrules to .cursor/rules/*.mdc
migrate_cursorrules() {
  local target_path="${1:-.}"
  target_path=$(expand_path "$target_path")
  local display_path="${target_path/#$HOME/~}"

  if [ ! -d "$target_path" ]; then
    log_error "Directory not found: $target_path"
    return 1
  fi

  if [ ! -f "$target_path/.cursorrules" ]; then
    log_error "No .cursorrules found in $target_path"
    return 1
  fi

  local rules_dir="$target_path/.cursor/rules"
  local source_file="$target_path/.cursorrules"
  local target_file="$rules_dir/legacy-rules.mdc"
  local backup_file="$target_path/.cursorrules.backup-$(date +%Y%m%d-%H%M%S)"
  local display_backup="${backup_file/#$HOME/~}"

  # Get file info
  local file_info
  file_info=$(show_file_info "$source_file")

  log_header "dot-agents migrate cursorrules"
  show_intro "Migrating .cursorrules → .cursor/rules/*.mdc"
  echo -e "Project: ${BOLD}$display_path${NC}"

  # Step 1: Analyze
  init_steps 4
  step "Analyzing current state..."
  bullet "found" "Found .cursorrules ($file_info)"
  if [ -d "$rules_dir" ]; then
    bullet "ok" "Existing .cursor/rules/ directory"
  else
    bullet "none" "No existing .cursor/rules/ directory"
  fi

  # Step 2: Preview
  step "Migration preview:"
  preview_changes "Source:" ".cursorrules"
  preview_changes "Will create:" ".cursor/rules/legacy-rules.mdc"
  echo ""
  echo -e "  ${DIM}The new file will include:${NC}"
  echo -e "  ${DIM}---${NC}"
  echo -e "  ${DIM}alwaysApply: true${NC}"
  echo -e "  ${DIM}description: Migrated from .cursorrules on $(date +%Y-%m-%d)${NC}"
  echo -e "  ${DIM}---${NC}"
  echo -e "  ${DIM}[original content]${NC}"

  info_box "Important" \
    "A backup will be saved to:" \
    "$display_backup" \
    "" \
    "The original .cursorrules will NOT be deleted." \
    "You can remove it manually after verifying."

  # Handle dry-run mode
  if [ "$DRY_RUN" = true ]; then
    echo ""
    log_info "DRY RUN - no changes made"
    echo ""
    echo -e "${DIM}Content preview:${NC}"
    echo "────────────────────────────────────────────────────"
    head -10 "$source_file"
    if [ "$(wc -l < "$source_file")" -gt 10 ]; then
      echo "... ($(wc -l < "$source_file") lines total)"
    fi
    return 0
  fi

  # Confirm before proceeding
  if ! confirm_action "Proceed with migration?"; then
    log_info "Migration cancelled."
    return 0
  fi

  # Step 3: Create backup
  step "Creating backup..."
  cp "$source_file" "$backup_file"
  bullet "ok" "Backup saved: $display_backup"

  # Step 4: Migrate
  step "Migrating..."

  # Create rules directory
  mkdir -p "$rules_dir"

  # Convert to .mdc with frontmatter
  {
    echo "---"
    echo "alwaysApply: true"
    echo "description: Migrated from .cursorrules on $(date +%Y-%m-%d)"
    echo "---"
    echo ""
    cat "$source_file"
  } > "$target_file"

  bullet "ok" "Created .cursor/rules/legacy-rules.mdc"

  # Note: We're NOT removing the original anymore - let user do it
  bullet "skip" "Original .cursorrules preserved (remove manually when ready)"

  success_with_next_steps "Migration complete!" \
    "Verify the migrated file: cat $display_path/.cursor/rules/legacy-rules.mdc" \
    "Test in Cursor IDE to ensure rules apply" \
    "Remove old file when ready: rm $display_path/.cursorrules"

  show_test_commands \
    "dot-agents audit" \
    "dot-agents doctor"
}

# Migrate .claude.json to .claude/settings.json
migrate_claude_json() {
  local target_path="${1:-.}"
  target_path=$(expand_path "$target_path")
  local display_path="${target_path/#$HOME/~}"

  if [ ! -d "$target_path" ]; then
    log_error "Directory not found: $target_path"
    return 1
  fi

  if [ ! -f "$target_path/.claude.json" ]; then
    log_error "No .claude.json found in $target_path"
    return 1
  fi

  local claude_dir="$target_path/.claude"
  local source_file="$target_path/.claude.json"
  local target_file="$claude_dir/settings.json"
  local backup_file="$target_path/.claude.json.backup-$(date +%Y%m%d-%H%M%S)"
  local display_backup="${backup_file/#$HOME/~}"

  # Get file info
  local file_info
  file_info=$(show_file_info "$source_file")

  log_header "dot-agents migrate claude-json"
  show_intro "Migrating .claude.json → .claude/settings.json"
  echo -e "Project: ${BOLD}$display_path${NC}"

  # Step 1: Analyze
  init_steps 4
  step "Analyzing current state..."

  # Check if valid JSON
  if ! jq -e '.' "$source_file" >/dev/null 2>&1; then
    bullet "error" ".claude.json is not valid JSON"
    log_error "Cannot proceed - fix JSON syntax first"
    return 1
  fi

  bullet "found" "Found .claude.json ($file_info)"
  bullet "ok" "Valid JSON format"

  if [ -d "$claude_dir" ]; then
    bullet "ok" "Existing .claude/ directory"
  else
    bullet "none" "No existing .claude/ directory"
  fi

  # Step 2: Preview
  step "Migration preview:"
  preview_changes "Source:" ".claude.json"
  preview_changes "Will create:" ".claude/settings.json"

  echo ""
  echo -e "  ${DIM}Fields detected in .claude.json:${NC}"
  jq -r 'keys[]' "$source_file" | while read -r key; do
    echo -e "    ${DIM}• $key${NC}"
  done

  echo ""
  echo -e "  ${DIM}Settings to extract:${NC}"
  echo -e "    ${DIM}• model, permissions, allowedTools, mcpServers${NC}"

  info_box "Important" \
    "A backup will be saved to:" \
    "$display_backup" \
    "" \
    "The original .claude.json will NOT be deleted." \
    "You can remove it manually after verifying."

  # Handle dry-run mode
  if [ "$DRY_RUN" = true ]; then
    echo ""
    log_info "DRY RUN - no changes made"
    return 0
  fi

  # Confirm before proceeding
  if ! confirm_action "Proceed with migration?"; then
    log_info "Migration cancelled."
    return 0
  fi

  # Step 3: Create backup
  step "Creating backup..."
  cp "$source_file" "$backup_file"
  bullet "ok" "Backup saved: $display_backup"

  # Step 4: Migrate
  step "Migrating..."

  # Create .claude directory
  mkdir -p "$claude_dir"

  # Extract settings (filter out runtime state)
  jq '{
    model: .model,
    permissions: .permissions,
    allowedTools: .allowedTools,
    mcpServers: .mcpServers
  } | with_entries(select(.value != null))' "$source_file" > "$target_file"

  bullet "ok" "Created .claude/settings.json"
  bullet "skip" "Original .claude.json preserved (remove manually when ready)"

  success_with_next_steps "Migration complete!" \
    "Verify the migrated file: cat $display_path/.claude/settings.json" \
    "Test in Claude Code to ensure settings apply" \
    "Remove old file when ready: rm $display_path/.claude.json"

  show_test_commands \
    "dot-agents audit" \
    "dot-agents doctor"
}
