# Changelog

All notable changes to dot-agents will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-01-10

### Added

- Initial release
- **Core Commands**
  - `init` - Initialize `~/.agents/` directory structure
  - `add <path>` - Add a project to dot-agents management
  - `remove <project>` - Remove a project from management
  - `status` - Show managed projects and their status
  - `doctor` - Health check and diagnostics
  - `audit` - Show which configs are applied where
- **Sync Commands**
  - `sync init` - Initialize git repository in `~/.agents/`
  - `sync status` - Show git status
  - `sync commit` - Commit all changes
  - `sync push` - Push to remote
  - `sync pull` - Pull from remote
  - `sync log` - Show recent commits
- **Utility Commands**
  - `context` - Output configuration as JSON for AI agents
- **Agent Support**
  - Cursor (`.cursor/rules/` with hard links)
  - Claude Code (`CLAUDE.md`, `.claude/` with symlinks)
  - Codex (`AGENTS.md` with symlinks)
  - OpenCode (detection only)
- **Installation**
  - Homebrew formula
  - curl install script
- **Features**
  - Automatic agent detection
  - Hard links for Cursor (required - doesn't follow symlinks)
  - Symlinks for Claude Code and Codex
  - JSON output for all inspection commands
  - Dry-run mode for all mutating commands
  - XDG-compliant state storage

### Notes

- Windows support deferred to future release
- Tasks and History features are opt-in and not yet implemented

[Unreleased]: https://github.com/dot-agents/dot-agents/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/dot-agents/dot-agents/releases/tag/v0.1.0
