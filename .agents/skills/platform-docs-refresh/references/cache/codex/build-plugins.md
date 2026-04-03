---
title: build-plugins
platform: codex
topic: plugins
canonical_url: https://developers.openai.com/codex/plugins/build.md
source_url:
checked_on: 2026-04-03
stale_url:
format: normalized-summary
---

# Summary

Current Codex docs for local plugin packaging and marketplace files.

# Key facts

- Plugin packages require `.codex-plugin/plugin.json` and can also include `skills/`, `.app.json`, `.mcp.json`, and `assets/`.
- Repo marketplaces live at `$REPO_ROOT/.agents/plugins/marketplace.json`; personal marketplaces live at `~/.agents/plugins/marketplace.json`.
- The docs use `$REPO_ROOT/plugins/` and `~/.codex/plugins/` as example storage locations, but `source.path` resolves relative to the marketplace root.

# Refresh notes

- No public markdown source was found during this pilot; this cache is a normalized summary of the rendered docs page.
