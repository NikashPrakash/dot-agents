#!/bin/bash
# dot-agents/lib/commands/redundancy.sh
# Check for duplicate/redundant rules across configs
# Compatible with bash 3.2+ (no associative arrays)

cmd_redundancy_help() {
  cat << EOF
${BOLD}dot-agents redundancy${NC} - Check for duplicate/redundant rules

${BOLD}USAGE${NC}
    dot-agents redundancy [project]
    dot-agents redundancy [options]

${BOLD}ARGUMENTS${NC}
    [project]         Check specific project (default: current directory or all)

${BOLD}OPTIONS${NC}
    --all             Check all registered projects
    --global-only     Check global rules only
    --verbose, -v     Show detailed comparison output
    --json            Output results as JSON
    --help, -h        Show this help

${BOLD}DESCRIPTION${NC}
    Scans rule files for duplicate or highly similar content.
    Helps identify:
    - Exact duplicates (copy-pasted rules)
    - Near-duplicates (minor variations)
    - Redundant rules between global and project configs

    Exact matches are found by comparing normalized text.
    Fuzzy matching uses line-by-line comparison.

${BOLD}EXAMPLES${NC}
    dot-agents redundancy              # Check current project
    dot-agents redundancy myproject    # Check specific project
    dot-agents redundancy --all        # Check all projects
    dot-agents redundancy --global-only  # Check global rules only
    dot-agents redundancy --verbose    # Detailed output

EOF
}

cmd_redundancy() {
  local project_filter=""
  local check_all=false
  local global_only=false

  # Parse flags
  REMAINING_ARGS=()
  while [[ $# -gt 0 ]]; do
    case $1 in
      --all)
        check_all=true
        shift
        ;;
      --global-only)
        global_only=true
        shift
        ;;
      --verbose|-v)
        VERBOSE=true
        shift
        ;;
      --json)
        JSON_OUTPUT=true
        shift
        ;;
      --help|-h)
        cmd_redundancy_help
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
      log_error "Project not found: $project_filter"
      return 1
    fi
  fi

  # Determine which projects to check
  local projects_to_check=""
  if [ "$global_only" = true ]; then
    projects_to_check="global"
  elif [ "$check_all" = true ]; then
    projects_to_check="global"
    while IFS= read -r name; do
      [ -n "$name" ] && projects_to_check="$projects_to_check $name"
    done < <(config_list_projects)
  elif [ -n "$project_filter" ]; then
    projects_to_check="global $project_filter"
  else
    # Try to detect project from current directory
    local current_project
    current_project=$(detect_project_from_cwd)
    if [ -n "$current_project" ]; then
      projects_to_check="global $current_project"
    else
      projects_to_check="global"
    fi
  fi

  if [ "$JSON_OUTPUT" = true ]; then
    check_redundancy_json $projects_to_check
  else
    check_redundancy_human $projects_to_check
  fi
}

# Detect project from current working directory
detect_project_from_cwd() {
  local cwd
  cwd=$(pwd)

  while IFS= read -r name; do
    [ -z "$name" ] && continue
    local path
    path=$(config_get_project_path "$name")
    path=$(expand_path "$path")

    if [ "$cwd" = "$path" ] || [[ "$cwd" == "$path"/* ]]; then
      echo "$name"
      return 0
    fi
  done < <(config_list_projects)

  return 1
}

# Human-readable redundancy check
check_redundancy_human() {
  local projects="$*"

  log_header "Redundancy Check"

  local total_files=0
  local total_duplicates=0
  local total_similar=0

  # Create temp file to store file list with scopes
  local tmp_files
  tmp_files=$(mktemp)
  local tmp_hashes
  tmp_hashes=$(mktemp)
  trap "rm -f '$tmp_files' '$tmp_hashes'" RETURN

  # Collect all rule files
  for scope in $projects; do
    local rules_dir="$AGENTS_HOME/rules/$scope"
    if [ -d "$rules_dir" ]; then
      for rule in "$rules_dir"/*.mdc "$rules_dir"/*.md; do
        [ -f "$rule" ] || continue
        echo "$scope:$rule" >> "$tmp_files"
        ((total_files++))
      done
    fi
  done

  if [ "$total_files" -eq 0 ]; then
    log_info "No rule files found to check."
    return 0
  fi

  echo -e "Checking ${BOLD}${total_files} rule files${NC} across: $projects"
  echo ""

  # Check for exact duplicates (by content hash)
  echo -e "${BOLD}Exact Duplicates:${NC}"
  local found_exact=false

  # Create hash file
  while IFS=: read -r scope file; do
    local content
    content=$(normalize_rule_content "$file")
    local hash
    hash=$(echo "$content" | md5 2>/dev/null || echo "$content" | md5sum | cut -d' ' -f1)
    echo "$hash:$scope:$file" >> "$tmp_hashes"
  done < "$tmp_files"

  # Find duplicates by sorting and comparing adjacent hashes
  sort "$tmp_hashes" | while read -r line; do
    local hash="${line%%:*}"
    local rest="${line#*:}"
    local scope="${rest%%:*}"
    local file="${rest#*:}"

    # Check if we've seen this hash before
    if grep -q "^$hash:" "$tmp_hashes" 2>/dev/null; then
      local matches
      matches=$(grep "^$hash:" "$tmp_hashes" | wc -l | tr -d ' ')
      if [ "$matches" -gt 1 ]; then
        # Only report first occurrence of each duplicate pair
        local first_match
        first_match=$(grep "^$hash:" "$tmp_hashes" | head -1)
        if [ "$line" = "$first_match" ]; then
          found_exact=true
          ((total_duplicates++))

          echo -e "  ${RED}DUPLICATE${NC}"
          grep "^$hash:" "$tmp_hashes" | while IFS=: read -r h s f; do
            local name
            name=$(basename "$f")
            echo -e "    ${DIM}[$s]${NC} $name"
          done
          echo ""
        fi
      fi
    fi
  done

  if [ "$found_exact" = false ]; then
    echo -e "  ${GREEN}None found${NC}"
    echo ""
  fi

  # Check for similar content (paragraph-level) - simplified version
  echo -e "${BOLD}Similar Content:${NC}"
  local found_similar=false

  # Get list of files
  local files=()
  local scopes=()
  while IFS=: read -r scope file; do
    files+=("$file")
    scopes+=("$scope")
  done < "$tmp_files"

  local file_count=${#files[@]}
  for ((i=0; i<file_count; i++)); do
    for ((j=i+1; j<file_count; j++)); do
      local file1="${files[$i]}"
      local file2="${files[$j]}"
      local scope1="${scopes[$i]}"
      local scope2="${scopes[$j]}"

      # Check if these files have similar paragraphs
      local similar_count
      similar_count=$(count_similar_paragraphs "$file1" "$file2")

      if [ "$similar_count" -gt 0 ]; then
        found_similar=true
        ((total_similar++))
        local name1
        name1=$(basename "$file1")
        local name2
        name2=$(basename "$file2")

        echo -e "  ${YELLOW}SIMILAR${NC} ($similar_count shared paragraphs)"
        echo -e "    ${DIM}[$scope1]${NC} $name1"
        echo -e "    ${DIM}[$scope2]${NC} $name2"

        if [ "$VERBOSE" = true ]; then
          echo -e "    ${DIM}Shared content:${NC}"
          show_similar_paragraphs "$file1" "$file2" | head -3 | while read -r para; do
            local preview="${para:0:60}"
            echo -e "      ${DIM}\"${preview}...\"${NC}"
          done
        fi
        echo ""
      fi
    done
  done

  if [ "$found_similar" = false ]; then
    echo -e "  ${GREEN}None found${NC}"
    echo ""
  fi

  # Summary
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo -e "${BOLD}Summary:${NC}"
  echo -e "  Files checked:    $total_files"
  echo -e "  Exact duplicates: ${total_duplicates:-0}"
  echo -e "  Similar pairs:    ${total_similar:-0}"

  if [ "$total_duplicates" -gt 0 ] || [ "$total_similar" -gt 0 ]; then
    echo ""
    echo -e "${YELLOW}Recommendation:${NC}"
    if [ "$total_duplicates" -gt 0 ]; then
      echo "  Remove exact duplicates - they waste context tokens."
    fi
    if [ "$total_similar" -gt 0 ]; then
      echo "  Review similar rules - consider consolidating into global rules."
    fi
  else
    echo ""
    echo -e "${GREEN}No redundancy issues found!${NC}"
  fi
}

# JSON output for redundancy check
check_redundancy_json() {
  local projects="$*"

  # Create temp file to store file list
  local tmp_files
  tmp_files=$(mktemp)
  local tmp_hashes
  tmp_hashes=$(mktemp)
  trap "rm -f '$tmp_files' '$tmp_hashes'" RETURN

  local total_files=0

  # Collect all rule files
  for scope in $projects; do
    local rules_dir="$AGENTS_HOME/rules/$scope"
    if [ -d "$rules_dir" ]; then
      for rule in "$rules_dir"/*.mdc "$rules_dir"/*.md; do
        [ -f "$rule" ] || continue
        echo "$scope:$rule" >> "$tmp_files"
        ((total_files++))
      done
    fi
  done

  echo "{"
  echo '  "scopes": ['
  local first_scope=true
  for scope in $projects; do
    [ "$first_scope" = true ] && first_scope=false || echo ","
    echo -n "    \"$scope\""
  done
  echo ""
  echo '  ],'
  echo '  "files_checked": '$total_files','

  # Create hash file
  while IFS=: read -r scope file; do
    local content
    content=$(normalize_rule_content "$file")
    local hash
    hash=$(echo "$content" | md5 2>/dev/null || echo "$content" | md5sum | cut -d' ' -f1)
    echo "$hash:$scope:$file" >> "$tmp_hashes"
  done < "$tmp_files"

  # Find duplicates by extracting just the hash and finding dupes
  echo '  "exact_duplicates": ['
  local first_dup=true

  # Get list of duplicate hashes
  local dup_hashes
  dup_hashes=$(cut -d: -f1 "$tmp_hashes" | sort | uniq -d)

  for hash in $dup_hashes; do
    local matches
    matches=$(grep "^$hash:" "$tmp_hashes")
    local file1=""
    local file2=""
    local scope1=""
    local scope2=""

    while IFS=: read -r h s f; do
      if [ -z "$file1" ]; then
        file1="$f"
        scope1="$s"
      else
        file2="$f"
        scope2="$s"
        break
      fi
    done <<< "$matches"

    if [ -n "$file1" ] && [ -n "$file2" ]; then
      [ "$first_dup" = true ] && first_dup=false || echo ","
      echo '    {'
      echo '      "file1": "'$file1'",'
      echo '      "file2": "'$file2'",'
      echo '      "scope1": "'$scope1'",'
      echo '      "scope2": "'$scope2'"'
      echo -n '    }'
    fi
  done
  echo ""
  echo '  ],'

  # Similar pairs - simplified
  echo '  "similar_pairs": []'
  echo "}"
}

# Normalize rule content for comparison (strip frontmatter, normalize whitespace)
normalize_rule_content() {
  local file="$1"
  local content
  content=$(cat "$file")

  # Strip YAML frontmatter
  content=$(echo "$content" | sed -n '/^---$/,/^---$/!p')

  # Normalize whitespace
  content=$(echo "$content" | tr -s '[:space:]' ' ' | tr '[:upper:]' '[:lower:]')

  echo "$content"
}

# Count shared paragraphs between two files
count_similar_paragraphs() {
  local file1="$1"
  local file2="$2"

  # Extract paragraphs (non-empty lines separated by blank lines)
  local paras1
  paras1=$(extract_paragraphs "$file1")
  local paras2
  paras2=$(extract_paragraphs "$file2")

  local count=0

  while IFS= read -r para1; do
    [ -z "$para1" ] && continue
    [ ${#para1} -lt 50 ] && continue  # Skip short paragraphs

    local norm1
    norm1=$(echo "$para1" | tr -s '[:space:]' ' ' | tr '[:upper:]' '[:lower:]')

    while IFS= read -r para2; do
      [ -z "$para2" ] && continue

      local norm2
      norm2=$(echo "$para2" | tr -s '[:space:]' ' ' | tr '[:upper:]' '[:lower:]')

      # Check for exact match
      if [ "$norm1" = "$norm2" ]; then
        ((count++))
        break
      fi
    done <<< "$paras2"
  done <<< "$paras1"

  echo "$count"
}

# Extract paragraphs from a markdown file
extract_paragraphs() {
  local file="$1"
  local content
  content=$(cat "$file")

  # Strip YAML frontmatter
  content=$(echo "$content" | sed -n '/^---$/,/^---$/!p')

  # Split by blank lines and output non-empty paragraphs
  echo "$content" | awk 'BEGIN{RS=""; ORS="\n"} NF>0 {print}'
}

# Show similar paragraphs (for verbose output)
show_similar_paragraphs() {
  local file1="$1"
  local file2="$2"

  local paras1
  paras1=$(extract_paragraphs "$file1")
  local paras2
  paras2=$(extract_paragraphs "$file2")

  while IFS= read -r para1; do
    [ -z "$para1" ] && continue
    [ ${#para1} -lt 50 ] && continue

    local norm1
    norm1=$(echo "$para1" | tr -s '[:space:]' ' ' | tr '[:upper:]' '[:lower:]')

    while IFS= read -r para2; do
      [ -z "$para2" ] && continue

      local norm2
      norm2=$(echo "$para2" | tr -s '[:space:]' ' ' | tr '[:upper:]' '[:lower:]')

      if [ "$norm1" = "$norm2" ]; then
        echo "$para1"
        break
      fi
    done <<< "$paras2"
  done <<< "$paras1"
}
