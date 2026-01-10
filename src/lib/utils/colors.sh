#!/bin/bash
# dot-agents/lib/utils/colors.sh
# Terminal color definitions
# Source this file to get color variables

# Only use colors if terminal supports it and not in CI/non-interactive
if [ -t 1 ] && [ -z "${NO_COLOR:-}" ] && [ "${TERM:-dumb}" != "dumb" ]; then
  # Use $'...' syntax to interpret escape sequences at definition time
  RED=$'\033[0;31m'
  GREEN=$'\033[0;32m'
  YELLOW=$'\033[1;33m'
  BLUE=$'\033[0;34m'
  MAGENTA=$'\033[0;35m'
  CYAN=$'\033[0;36m'
  WHITE=$'\033[1;37m'
  GRAY=$'\033[0;90m'
  BOLD=$'\033[1m'
  DIM=$'\033[2m'
  NC=$'\033[0m'  # No Color / Reset
else
  RED=''
  GREEN=''
  YELLOW=''
  BLUE=''
  MAGENTA=''
  CYAN=''
  WHITE=''
  GRAY=''
  BOLD=''
  DIM=''
  NC=''
fi

# Export for subshells
export RED GREEN YELLOW BLUE MAGENTA CYAN WHITE GRAY BOLD DIM NC
