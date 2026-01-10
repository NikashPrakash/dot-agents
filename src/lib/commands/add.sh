#!/bin/bash
# dot-agents/lib/commands/add.sh
# Add a project to dot-agents management

cmd_add_help() {
  cat << EOF
${BOLD}dot-agents add${NC} - Add a project to dot-agents management

${BOLD}USAGE${NC}
    dot-agents add <path> [options]

${BOLD}ARGUMENTS${NC}
    <path>            Path to project directory (required)

${BOLD}OPTIONS${NC}
    --name <name>     Override project name (default: directory name)
    --dry-run         Show what would be done without making changes
    --force, -f       Overwrite existing configurations
    --verbose, -v     Show detailed output
    --help, -h        Show this help

${BOLD}DESCRIPTION${NC}
    Registers a project with dot-agents and sets up configuration symlinks.

    Creates:
    - ~/.agents/rules/<project>/         Project-specific rules
    - ~/.agents/settings/<project>/      Project settings
    - ~/.agents/mcp/<project>/           Project MCP configs

    Links (in project directory):
    - .cursor/rules/                     Hard links to rules (Cursor)
    - .claude/                           Symlink to settings (Claude Code)
    - CLAUDE.md                          Symlink to rules (Claude Code)

${BOLD}EXAMPLES${NC}
    dot-agents add ~/Github/myproject
    dot-agents add . --name my-api
    dot-agents add ~/work/app --dry-run

EOF
}

cmd_add() {
  # Parse flags
  local project_name=""
  local project_path=""

  REMAINING_ARGS=()
  while [[ $# -gt 0 ]]; do
    case $1 in
      --name)
        project_name="$2"
        shift 2
        ;;
      --dry-run)
        DRY_RUN=true
        shift
        ;;
      --force|-f)
        FORCE=true
        shift
        ;;
      --yes|-y)
        YES=true
        shift
        ;;
      --verbose|-v)
        VERBOSE=true
        shift
        ;;
      --help|-h)
        cmd_add_help
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

  # Get project path from remaining args
  if [ ${#REMAINING_ARGS[@]} -eq 0 ]; then
    log_error "Project path required"
    echo ""
    echo "Usage: dot-agents add <path>"
    return 1
  fi

  project_path="${REMAINING_ARGS[0]}"

  # Expand and validate path
  project_path=$(expand_path "$project_path")

  if [ ! -d "$project_path" ]; then
    log_error "Directory not found: $project_path"
    return 1
  fi

  # Derive project name from directory if not specified
  if [ -z "$project_name" ]; then
    project_name=$(basename "$project_path")
  fi

  # Validate project name (alphanumeric, hyphens, underscores)
  if ! [[ "$project_name" =~ ^[a-zA-Z0-9_-]+$ ]]; then
    log_error "Invalid project name: $project_name"
    log_info "Use --name to specify a valid name (alphanumeric, hyphens, underscores)"
    return 1
  fi

  local display_path="${project_path/#$HOME/~}"

  log_header "dot-agents add"
  echo -e "Adding project: ${BOLD}$project_name${NC}"
  echo -e "Path: ${DIM}$display_path${NC}"

  # Step 1: Scan project
  init_steps 4
  step "Scanning project..."

  # Check if it's a git repository
  if [ -d "$project_path/.git" ]; then
    bullet "ok" "Valid git repository"
  else
    bullet "none" "Not a git repository (optional)"
  fi

  # Check if already registered
  local existing_path
  existing_path=$(config_get_project_path "$project_name")
  if [ -n "$existing_path" ]; then
    if [ "$FORCE" != true ]; then
      bullet "warn" "Already registered at: $existing_path"
      echo ""
      log_info "Use --force to update, or --name to use a different name"
      return 1
    else
      bullet "warn" "Will update existing registration (--force)"
    fi
  else
    bullet "ok" "Not yet registered"
  fi

  # Check for deprecated formats
  local has_deprecated=false
  if cursor_has_deprecated_format "$project_path"; then
    bullet "warn" "Found deprecated .cursorrules file"
    has_deprecated=true
  fi
  if claude_has_deprecated_format "$project_path"; then
    bullet "warn" "Found deprecated .claude.json file"
    has_deprecated=true
  fi

  if [ "$has_deprecated" = true ]; then
    warn_box "Deprecated Formats Found" \
      "After adding this project, run:" \
      "  dot-agents migrate cursorrules $display_path" \
      "  dot-agents migrate claude-json $display_path"
  fi

  # Step 2: Preview what will be created
  step "The following will be created:"

  preview_section "~/.agents/" \
    "rules/$project_name/              (project rules)" \
    "settings/$project_name/           (project settings)" \
    "mcp/$project_name/                (project MCP configs)"

  preview_section "$display_path/" \
    ".cursor/rules/global--*.mdc       (hard links to global rules)" \
    "CLAUDE.md                         (symlink to rules)" \
    "AGENTS.md                         (symlink to rules)" \
    ".opencode/                        (symlinks to configs)"

  info_box "About Link Types" \
    "Cursor uses HARD LINKS (required by IDE)." \
    "Other agents use symlinks for flexibility."

  # Handle dry-run mode
  if [ "$DRY_RUN" = true ]; then
    echo ""
    log_info "DRY RUN - no changes made"
    return 0
  fi

  # Confirm before proceeding
  if ! confirm_action "Proceed?"; then
    log_info "Add cancelled."
    return 0
  fi

  # Step 3: Create project structure
  step "Creating project structure..."
  create_project_dirs_silent "$project_name"
  bullet "ok" "Created ~/.agents/ directories"

  # Step 4: Link to project
  step "Linking to project..."

  # Cursor: .cursor/rules/ with HARD LINKS
  cursor_create_rule_links "$project_name" "$project_path"
  bullet "ok" ".cursor/rules/ (hard links)"

  # Claude Code: CLAUDE.md and settings symlinks
  claude_create_links "$project_name" "$project_path"
  bullet "ok" "Claude Code links (symlinks)"

  # Codex: AGENTS.md symlink
  codex_create_links "$project_name" "$project_path"
  bullet "ok" "Codex links (symlinks)"

  # OpenCode: .opencode/ symlinks
  opencode_create_links "$project_name" "$project_path"
  bullet "ok" "OpenCode links (symlinks)"

  # Register in config.json
  config_add_project "$project_name" "$project_path"
  bullet "ok" "Registered in config.json"

  # Build next steps based on context
  local next_steps=()
  next_steps+=("Add project rules: edit ~/.agents/rules/$project_name/rules.md")
  if [ "$has_deprecated" = true ]; then
    next_steps+=("Migrate deprecated formats: dot-agents migrate detect")
  fi
  next_steps+=("Check applied configs: dot-agents audit $project_name")

  success_with_next_steps "Project '$project_name' added successfully!" "${next_steps[@]}"

  show_test_commands \
    "dot-agents status" \
    "ls -la $display_path/.cursor/rules/"
}

# Create project directories in ~/.agents/ (silent version for bulk)
create_project_dirs_silent() {
  local project="$1"
  local agents_home="$AGENTS_HOME"

  mkdir -p "$agents_home/rules/$project"
  mkdir -p "$agents_home/settings/$project"
  mkdir -p "$agents_home/mcp/$project"
  mkdir -p "$agents_home/commands/$project"
}

# Create project directories in ~/.agents/ (verbose version)
create_project_dirs() {
  local project="$1"
  local agents_home="$AGENTS_HOME"

  local dirs=(
    "$agents_home/rules/$project"
    "$agents_home/settings/$project"
    "$agents_home/mcp/$project"
    "$agents_home/commands/$project"
  )

  for dir in "${dirs[@]}"; do
    local display_dir="${dir/#$HOME/~}"
    if [ -d "$dir" ]; then
      [ "$VERBOSE" = true ] && log_skip "$display_dir/ (exists)"
    elif [ "$DRY_RUN" = true ]; then
      log_dry "mkdir $display_dir/"
    else
      mkdir -p "$dir"
      log_create "$display_dir/"
    fi
  done
}

# Set up symlinks and hard links in the project directory
setup_project_links() {
  local project="$1"
  local repo="$2"

  # Check for deprecated formats and warn
  check_deprecated_formats "$repo"

  # Use platform modules for linking (sourced via core.sh)
  if [ "$DRY_RUN" = true ]; then
    log_dry "Create Cursor hard links in .cursor/rules/"
    log_dry "Create Claude Code symlinks (CLAUDE.md, .claude/)"
    log_dry "Create Codex symlinks (AGENTS.md, .codex/)"
    log_dry "Create OpenCode symlinks (.opencode/)"
  else
    # Cursor: .cursor/rules/ with HARD LINKS (Cursor doesn't follow symlinks)
    cursor_create_rule_links "$project" "$repo"
    log_create ".cursor/rules/ (hard links)"

    # Claude Code: CLAUDE.md and settings symlinks
    claude_create_links "$project" "$repo"
    log_create "Claude Code links (symlinks)"

    # Codex: AGENTS.md symlink
    codex_create_links "$project" "$repo"
    log_create "Codex links (symlinks)"

    # OpenCode: .opencode/ symlinks
    opencode_create_links "$project" "$repo"
    log_create "OpenCode links (symlinks)"
  fi
}

# Check for deprecated config formats and warn
check_deprecated_formats() {
  local repo="$1"
  local found_deprecated=false

  if cursor_has_deprecated_format "$repo"; then
    log_warn "Found deprecated .cursorrules file"
    log_info "  → Run: dot-agents migrate cursorrules $repo"
    found_deprecated=true
  fi

  if claude_has_deprecated_format "$repo"; then
    log_warn "Found deprecated .claude.json file"
    log_info "  → Run: dot-agents migrate claude-json $repo"
    found_deprecated=true
  fi

  if [ "$found_deprecated" = true ]; then
    echo ""
  fi
}

# Legacy functions removed - now using platform modules:
#   cursor_create_rule_links() from platforms/cursor.sh
#   claude_create_links() from platforms/claude-code.sh
#   codex_create_links() from platforms/codex.sh
#   opencode_create_links() from platforms/opencode.sh
