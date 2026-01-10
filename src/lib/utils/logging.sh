#!/bin/bash
# dot-agents/lib/utils/logging.sh
# Logging functions for consistent CLI output
# Requires: colors.sh to be sourced first

# Logging functions with prefixes
log_info()    { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[OK]${NC} $1"; }
log_warn()    { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error()   { echo -e "${RED}[ERROR]${NC} $1" >&2; }
log_skip()    { echo -e "${GRAY}[SKIP]${NC} $1"; }
log_create()  { echo -e "${GREEN}[CREATE]${NC} $1"; }
log_dry()     { echo -e "${YELLOW}[DRY-RUN]${NC} $1"; }
log_debug()   { [ "${VERBOSE:-false}" = true ] && echo -e "${GRAY}[DEBUG]${NC} $1"; }

# Plain output (no prefix, for status tables etc.)
log_plain() { echo -e "$1"; }

# Header output
log_header() {
  echo ""
  echo -e "${BOLD}$1${NC}"
  echo -e "${GRAY}$(printf '%.0s─' {1..50})${NC}"
}

# Section divider
log_section() {
  echo ""
  echo -e "${CYAN}▸ $1${NC}"
}

# For JSON output mode
json_output() {
  if [ "${JSON_OUTPUT:-false}" = true ]; then
    echo "$1"
    return 0
  fi
  return 1
}

# Die with error message
die() {
  log_error "$1"
  exit "${2:-1}"
}

# Die if command fails
die_on_fail() {
  if ! "$@"; then
    die "Command failed: $*"
  fi
}
