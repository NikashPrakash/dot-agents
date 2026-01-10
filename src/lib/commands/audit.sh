#!/bin/bash
# dot-agents/lib/commands/audit.sh
# Show which configs are applied where

cmd_audit_help() {
  cat << EOF
${BOLD}dot-agents audit${NC} - Show which configs are applied where

${BOLD}USAGE${NC}
    dot-agents audit [options]
    dot-agents audit [project]

${BOLD}ARGUMENTS${NC}
    [project]         Audit a specific project (optional)

${BOLD}OPTIONS${NC}
    --agent <name>    Filter by agent (cursor, claude-code, codex)
    --json            Output in JSON format
    --verbose, -v     Show detailed information
    --help, -h        Show this help

${BOLD}DESCRIPTION${NC}
    Shows which configuration files from ~/.agents/ are being applied
    to your projects, and how they're linked.

${BOLD}EXAMPLES${NC}
    dot-agents audit                  # Audit all projects
    dot-agents audit myproject        # Audit specific project
    dot-agents audit --agent cursor   # Show only Cursor configs

EOF
}

cmd_audit() {
  local project_filter=""
  local agent_filter=""

  # Parse flags
  REMAINING_ARGS=()
  while [[ $# -gt 0 ]]; do
    case $1 in
      --agent)
        agent_filter="$2"
        shift 2
        ;;
      --json)
        JSON_OUTPUT=true
        shift
        ;;
      --verbose|-v)
        VERBOSE=true
        shift
        ;;
      --help|-h)
        cmd_audit_help
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
  fi

  if [ "$JSON_OUTPUT" = true ]; then
    run_audit_json "$project_filter" "$agent_filter"
  else
    run_audit_text "$project_filter" "$agent_filter"
  fi
}

run_audit_text() {
  local project_filter="$1"
  local agent_filter="$2"

  log_header "dot-agents audit"
  log_info "Source: $AGENTS_HOME"
  echo ""

  # Get projects to audit
  local projects
  if [ -n "$project_filter" ]; then
    # Check if project exists
    local path
    path=$(config_get_project_path "$project_filter")
    if [ -z "$path" ]; then
      log_error "Project not found: $project_filter"
      return 1
    fi
    projects="$project_filter"
  else
    projects=$(config_list_projects)
  fi

  # Check if any projects
  if [ -z "$projects" ]; then
    log_info "No projects registered."
    echo "Add a project with: dot-agents add <path>"
    return 0
  fi

  # Audit each project
  while IFS= read -r name; do
    [ -z "$name" ] && continue

    local path
    path=$(config_get_project_path "$name")
    local display_path="${path/#$HOME/~}"

    echo -e "${BOLD}$name${NC}"
    echo -e "${DIM}$display_path${NC}"
    echo ""

    if [ ! -d "$path" ]; then
      echo -e "  ${RED}Directory not found${NC}"
      echo ""
      continue
    fi

    # Audit Cursor configs
    if [ -z "$agent_filter" ] || [ "$agent_filter" = "cursor" ]; then
      audit_cursor_text "$name" "$path"
    fi

    # Audit Claude Code configs
    if [ -z "$agent_filter" ] || [ "$agent_filter" = "claude-code" ]; then
      audit_claude_text "$name" "$path"
    fi

    # Audit Codex configs
    if [ -z "$agent_filter" ] || [ "$agent_filter" = "codex" ]; then
      audit_codex_text "$name" "$path"
    fi

  done <<< "$projects"
}

audit_cursor_text() {
  local name="$1"
  local path="$2"

  echo -e "  ${CYAN}Cursor${NC}"

  # Check .cursor/rules/
  local rules_dir="$path/.cursor/rules"
  if [ -d "$rules_dir" ]; then
    local rule_count=0
    shopt -s nullglob
    for rule in "$rules_dir"/*.mdc; do
      local basename
      basename=$(basename "$rule")
      local source_type=""
      local linked_to=""

      # Check if it's a hard link to ~/.agents/
      if [ -f "$rule" ]; then
        # Try to find the source
        if [[ "$basename" == global--* ]]; then
          source_type="global"
          local source_name="${basename#global--}"
          linked_to="~/.agents/rules/global/$source_name"
        elif [[ "$basename" == "${name}--"* ]]; then
          source_type="project"
          local source_name="${basename#${name}--}"
          linked_to="~/.agents/rules/$name/$source_name"
        else
          source_type="local"
        fi

        if [ "$source_type" = "local" ]; then
          echo -e "    ${DIM}○${NC} $basename ${DIM}(local file)${NC}"
        else
          # Check if source exists and matches
          local source_path="${linked_to/#\~/$HOME}"
          if [ -f "$source_path" ] && are_hardlinked "$rule" "$source_path"; then
            echo -e "    ${GREEN}✓${NC} $basename ${DIM}← $linked_to${NC}"
          elif [ -f "$source_path" ]; then
            echo -e "    ${YELLOW}!${NC} $basename ${DIM}(not linked to $linked_to)${NC}"
          else
            echo -e "    ${YELLOW}!${NC} $basename ${DIM}(source missing: $linked_to)${NC}"
          fi
        fi
        ((rule_count++)) || true
      fi
    done
    shopt -u nullglob

    if [ $rule_count -eq 0 ]; then
      echo -e "    ${DIM}(no rules)${NC}"
    fi
  else
    echo -e "    ${DIM}(no .cursor/rules/)${NC}"
  fi
  echo ""
}

audit_claude_text() {
  local name="$1"
  local path="$2"

  echo -e "  ${CYAN}Claude Code${NC}"

  # Check CLAUDE.md
  local claude_md="$path/CLAUDE.md"
  if [ -e "$claude_md" ]; then
    if [ -L "$claude_md" ]; then
      local target
      target=$(readlink "$claude_md")
      local display_target="${target/#$HOME/~}"
      if [ -f "$target" ]; then
        echo -e "    ${GREEN}✓${NC} CLAUDE.md ${DIM}→ $display_target${NC}"
      else
        echo -e "    ${RED}✗${NC} CLAUDE.md ${DIM}→ $display_target (broken)${NC}"
      fi
    else
      echo -e "    ${DIM}○${NC} CLAUDE.md ${DIM}(local file)${NC}"
    fi
  else
    echo -e "    ${DIM}(no CLAUDE.md)${NC}"
  fi

  # Check .claude/ directory
  local claude_dir="$path/.claude"
  if [ -e "$claude_dir" ]; then
    if [ -L "$claude_dir" ]; then
      local target
      target=$(readlink "$claude_dir")
      local display_target="${target/#$HOME/~}"
      if [ -d "$target" ]; then
        echo -e "    ${GREEN}✓${NC} .claude/ ${DIM}→ $display_target${NC}"
      else
        echo -e "    ${RED}✗${NC} .claude/ ${DIM}→ $display_target (broken)${NC}"
      fi
    else
      echo -e "    ${DIM}○${NC} .claude/ ${DIM}(local directory)${NC}"
    fi
  fi
  echo ""
}

audit_codex_text() {
  local name="$1"
  local path="$2"

  echo -e "  ${CYAN}Codex${NC}"

  # Check AGENTS.md
  local agents_md="$path/AGENTS.md"
  if [ -e "$agents_md" ]; then
    if [ -L "$agents_md" ]; then
      local target
      target=$(readlink "$agents_md")
      local display_target="${target/#$HOME/~}"
      if [ -f "$target" ]; then
        echo -e "    ${GREEN}✓${NC} AGENTS.md ${DIM}→ $display_target${NC}"
      else
        echo -e "    ${RED}✗${NC} AGENTS.md ${DIM}→ $display_target (broken)${NC}"
      fi
    else
      echo -e "    ${DIM}○${NC} AGENTS.md ${DIM}(local file)${NC}"
    fi
  else
    echo -e "    ${DIM}(no AGENTS.md)${NC}"
  fi

  # Check .codex/ directory
  local codex_dir="$path/.codex"
  if [ -d "$codex_dir" ]; then
    echo -e "    ${DIM}○${NC} .codex/ ${DIM}(local directory)${NC}"
  fi
  echo ""
}

run_audit_json() {
  local project_filter="$1"
  local agent_filter="$2"

  echo "{"
  echo '  "agents_home": "'$AGENTS_HOME'",'
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

    if [ "$first" = true ]; then
      first=false
    else
      echo ","
    fi

    echo -n '    "'$name'": {'
    echo -n '"path": "'$path'"'

    if [ -d "$path" ]; then
      # Cursor
      if [ -z "$agent_filter" ] || [ "$agent_filter" = "cursor" ]; then
        echo -n ', "cursor": {'
        echo -n '"rules": ['
        local rules_first=true
        if [ -d "$path/.cursor/rules" ]; then
          shopt -s nullglob
          for rule in "$path/.cursor/rules"/*.mdc; do
            [ "$rules_first" = true ] && rules_first=false || echo -n ","
            echo -n '"'$(basename "$rule")'"'
          done
          shopt -u nullglob
        fi
        echo -n ']}'
      fi

      # Claude
      if [ -z "$agent_filter" ] || [ "$agent_filter" = "claude-code" ]; then
        echo -n ', "claude-code": {'
        echo -n '"CLAUDE.md": '
        if [ -e "$path/CLAUDE.md" ]; then
          echo -n 'true'
        else
          echo -n 'false'
        fi
        echo -n '}'
      fi

      # Codex
      if [ -z "$agent_filter" ] || [ "$agent_filter" = "codex" ]; then
        echo -n ', "codex": {'
        echo -n '"AGENTS.md": '
        if [ -e "$path/AGENTS.md" ]; then
          echo -n 'true'
        else
          echo -n 'false'
        fi
        echo -n '}'
      fi
    fi

    echo -n '}'
  done <<< "$projects"

  echo ""
  echo "  }"
  echo "}"
}
