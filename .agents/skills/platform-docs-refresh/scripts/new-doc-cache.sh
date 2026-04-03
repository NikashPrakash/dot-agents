#!/bin/zsh
set -euo pipefail

if [[ $# -lt 3 || $# -gt 4 ]]; then
  echo "usage: $0 <platform> <slug> <canonical-url> [stale-url]" >&2
  exit 1
fi

platform="$1"
slug="$2"
canonical_url="$3"
stale_url="${4:-}"

skill_dir="${0:A:h:h}"
cache_dir="$skill_dir/references/cache/$platform"
target="$cache_dir/$slug.md"

mkdir -p "$cache_dir"

if [[ -e "$target" ]]; then
  echo "$target"
  exit 0
fi

cat > "$target" <<EOF
---
title: $slug
platform: $platform
topic: $slug
canonical_url: $canonical_url
source_url:
checked_on:
stale_url: $stale_url
format:
---

# Summary

# Key facts

- 

# Refresh notes

- 
EOF

echo "$target"
