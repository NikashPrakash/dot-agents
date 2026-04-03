# Cache Policy

Use this file when saving source snapshots during a docs refresh.

## Preferred order

1. Raw or source markdown published by the vendor
2. Markdown file from the vendor's docs repo
3. A concise normalized summary when only rendered HTML exists

## Cache file template

```md
---
title: <short page title>
platform: <cursor|claude|codex|opencode|copilot>
topic: <rules|skills|agents|mcp|hooks|plugins|...>
canonical_url: <current official URL>
source_url: <markdown source URL if different>
checked_on: YYYY-MM-DD
stale_url: <old URL if this refresh replaced one>
format: <markdown-source|markdown-rendered|normalized-summary>
---

# Summary

Short summary of what this page documents and why it matters to the repo doc.

# Key facts

- Fact tied to the current platform behavior
- Fact tied to the path or precedence rules

# Refresh notes

- Anything that moved, disappeared, or remains unclear
```

## Rules

- Overwrite the file on refresh instead of versioning copies by date.
- Keep cached files small and scannable.
- If a page is long, keep only the sections relevant to the repo's platform matrix and note the omitted areas.
