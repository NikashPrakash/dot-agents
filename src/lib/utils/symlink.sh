#!/bin/bash
# dot-agents/lib/utils/symlink.sh
# Symlink and hardlink utilities
# Adapted from ~/.agents/install/lib.sh patterns

# Safe symlink creation - idempotent
# Usage: safe_symlink TARGET LINK_PATH
# Returns: 0 on success/skip, 1 on error
safe_symlink() {
  local target="$1"
  local link="$2"
  local link_name=$(basename "$link")
  local display_link="${link/#$HOME/~}"

  # Check if target exists (for absolute paths)
  if [[ "$target" = /* ]] && [ ! -e "$target" ]; then
    log_warn "$link_name -> target does not exist: $target"
    return 1
  fi

  # Case 1: Symlink already exists and points to correct target
  if [ -L "$link" ]; then
    local current_target
    current_target=$(readlink "$link")
    if [ "$current_target" = "$target" ]; then
      [ "$VERBOSE" = true ] && log_skip "$display_link (already correct)"
      return 0
    else
      # Symlink exists but points elsewhere
      if [ "$FORCE" = true ]; then
        if [ "$DRY_RUN" = true ]; then
          log_dry "update $display_link -> $target (was: $current_target)"
        else
          rm "$link"
          ln -sf "$target" "$link"
          log_success "$display_link -> $target (updated)"
        fi
        return 0
      else
        log_warn "$display_link points to $current_target (use --force to update)"
        return 1
      fi
    fi
  fi

  # Case 2: Regular file/directory exists (not a symlink)
  if [ -e "$link" ]; then
    if [ "$FORCE" = true ]; then
      if [ "$DRY_RUN" = true ]; then
        log_dry "replace $display_link (existing file) -> $target"
      else
        rm -rf "$link"
        ln -sf "$target" "$link"
        log_success "$display_link -> $target (replaced existing)"
      fi
      return 0
    else
      log_warn "$display_link exists as regular file (use --force to replace)"
      return 1
    fi
  fi

  # Case 3: Nothing exists - create fresh
  if [ "$DRY_RUN" = true ]; then
    log_dry "create $display_link -> $target"
  else
    # Ensure parent directory exists
    mkdir -p "$(dirname "$link")"
    ln -sf "$target" "$link"
    log_create "$display_link -> $target"
  fi
  return 0
}

# Safe hard link creation for files
# Cursor doesn't follow symlinks for rules, so we use hard links
# Usage: safe_hardlink SOURCE LINK_PATH
# Returns: 0 on success/skip, 1 on error
safe_hardlink() {
  local source="$1"
  local link="$2"
  local link_name=$(basename "$link")
  local display_link="${link/#$HOME/~}"

  # Check source exists
  if [ ! -f "$source" ]; then
    log_warn "$link_name -> source does not exist: $source"
    return 1
  fi

  # Get source inode for comparison (cross-platform)
  local source_inode
  if [[ "$(uname)" == "Darwin" ]]; then
    source_inode=$(stat -f %i "$source" 2>/dev/null)
  else
    source_inode=$(stat -c %i "$source" 2>/dev/null)
  fi

  # Case 1: Link already exists as regular file
  if [ -f "$link" ] && [ ! -L "$link" ]; then
    local link_inode
    if [[ "$(uname)" == "Darwin" ]]; then
      link_inode=$(stat -f %i "$link" 2>/dev/null)
    else
      link_inode=$(stat -c %i "$link" 2>/dev/null)
    fi

    if [ "$source_inode" = "$link_inode" ]; then
      [ "$VERBOSE" = true ] && log_skip "$display_link (already correct hard link)"
      return 0
    else
      # File exists but different inode
      if [ "$FORCE" = true ]; then
        if [ "$DRY_RUN" = true ]; then
          log_dry "replace $display_link with hard link"
        else
          rm "$link"
          ln "$source" "$link"
          log_success "$display_link (replaced with hard link)"
        fi
        return 0
      else
        log_warn "$display_link exists but is not linked to source (use --force)"
        return 1
      fi
    fi
  fi

  # Case 2: Symlink exists (replace with hard link)
  if [ -L "$link" ]; then
    if [ "$DRY_RUN" = true ]; then
      log_dry "replace symlink $display_link with hard link"
    else
      rm "$link"
      ln "$source" "$link"
      log_success "$display_link (symlink replaced with hard link)"
    fi
    return 0
  fi

  # Case 3: Nothing exists - create fresh
  if [ "$DRY_RUN" = true ]; then
    log_dry "create hard link $display_link"
  else
    mkdir -p "$(dirname "$link")"
    ln "$source" "$link"
    log_create "$display_link (hard link)"
  fi
  return 0
}

# Check if path is a valid symlink pointing to expected target
# Usage: is_valid_symlink LINK TARGET
is_valid_symlink() {
  local link="$1"
  local target="$2"
  [ -L "$link" ] && [ "$(readlink "$link")" = "$target" ]
}

# Check if two files are hard-linked (same inode)
# Usage: are_hardlinked FILE1 FILE2
are_hardlinked() {
  local file1="$1"
  local file2="$2"
  local inode1 inode2

  [ -f "$file1" ] && [ -f "$file2" ] || return 1

  if [[ "$(uname)" == "Darwin" ]]; then
    inode1=$(stat -f %i "$file1" 2>/dev/null)
    inode2=$(stat -f %i "$file2" 2>/dev/null)
  else
    inode1=$(stat -c %i "$file1" 2>/dev/null)
    inode2=$(stat -c %i "$file2" 2>/dev/null)
  fi

  [ "$inode1" = "$inode2" ]
}

# Remove symlink if it exists
# Usage: remove_symlink LINK_PATH
remove_symlink() {
  local link="$1"
  local display_link="${link/#$HOME/~}"

  if [ -L "$link" ]; then
    if [ "$DRY_RUN" = true ]; then
      log_dry "remove $display_link"
    else
      rm "$link"
      log_info "Removed $display_link"
    fi
    return 0
  elif [ -e "$link" ]; then
    log_warn "$display_link is not a symlink"
    return 1
  fi
  return 0
}

export -f safe_symlink safe_hardlink is_valid_symlink are_hardlinked remove_symlink
