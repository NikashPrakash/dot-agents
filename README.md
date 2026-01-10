# dot-agents

**Unified config layer for AI coding agents**

One CLI to manage configurations across Cursor, Claude Code, Codex, and more.

```bash
# Install
brew tap dot-agents/tap && brew install dot-agents

# Set up
dot-agents init
dot-agents add ~/Github/myproject

# Check status
dot-agents status
dot-agents doctor
```

---

## The Problem

Every AI coding agent has its own config location and format:

| Agent | Config Location | Format |
|-------|-----------------|--------|
| Cursor | `.cursor/rules/*.mdc` | MDC (Markdown) |
| Claude Code | `CLAUDE.md`, `.claude/` | Markdown, JSON |
| Codex | `AGENTS.md` | Markdown |

This leads to:
- **Duplicated rules** across every repository
- **No way to share** common configurations
- **Inconsistent setups** between machines

## The Solution

**dot-agents** creates a single source of truth at `~/.agents/`:

```
~/.agents/
├── config.json              # Projects, settings, feature flags
├── rules/
│   ├── global/              # Applied to ALL projects
│   │   ├── coding-style.mdc
│   │   └── security.mdc
│   └── myproject/           # Project-specific rules
│       └── api-patterns.mdc
├── settings/
│   └── global/
└── mcp/
    └── global/
```

Then **symlinks and hard links** distribute configs to your projects automatically:

```
~/Github/myproject/
├── .cursor/rules/
│   ├── global--coding-style.mdc  → ~/.agents/rules/global/...
│   └── myproject--api-patterns.mdc → ~/.agents/rules/myproject/...
├── CLAUDE.md                     → ~/.agents/rules/global/claude-code.mdc
└── (your code)
```

## Installation

### Homebrew (recommended)

```bash
brew tap dot-agents/tap
brew install dot-agents
```

### Direct Install

```bash
curl -fsSL https://raw.githubusercontent.com/dot-agents/dot-agents/main/scripts/install.sh | bash
```

### Manual

```bash
git clone https://github.com/dot-agents/dot-agents ~/.dot-agents
export PATH="$HOME/.dot-agents/src/bin:$PATH"
```

## Quick Start

```bash
# 1. Initialize ~/.agents/
dot-agents init

# 2. Add a project
dot-agents add ~/Github/myproject

# 3. Add your rules to ~/.agents/rules/global/
#    They'll be linked to all projects automatically

# 4. Check what's applied
dot-agents audit
```

## Commands

### Core

| Command | Description |
|---------|-------------|
| `init` | Initialize `~/.agents/` directory |
| `add <path>` | Add a project to management |
| `remove <project>` | Remove a project |
| `status` | Show all managed projects |
| `doctor` | Health check and diagnostics |
| `audit` | Show which configs are applied where |

### Sync

| Command | Description |
|---------|-------------|
| `sync init` | Initialize git repo in `~/.agents/` |
| `sync status` | Show git status |
| `sync commit` | Commit all changes |
| `sync push` | Push to remote |
| `sync pull` | Pull from remote |

### Utilities

| Command | Description |
|---------|-------------|
| `context` | Output JSON for AI agents |
| `--help` | Show help for any command |
| `--version` | Show version |

## How It Works

### Cursor Rules (Hard Links)

Cursor doesn't follow symlinks for `.cursor/rules/`, so dot-agents uses **hard links**:

```bash
# In your project
.cursor/rules/global--coding-style.mdc  # Hard link to ~/.agents/rules/global/coding-style.mdc
```

Hard links share the same file content (same inode), so edits in either location are reflected in both.

### Claude Code / Codex (Symlinks)

For `CLAUDE.md` and `AGENTS.md`, standard symlinks work:

```bash
CLAUDE.md → ~/.agents/rules/global/claude-code.mdc
```

### Naming Convention

Files in `.cursor/rules/` are prefixed to show their source:
- `global--*.mdc` → From `~/.agents/rules/global/`
- `{project}--*.mdc` → From `~/.agents/rules/{project}/`

## Syncing Across Machines

Your `~/.agents/` directory is designed to be git-tracked:

```bash
# First time setup
dot-agents sync init
cd ~/.agents
git remote add origin git@github.com:YOU/agents-config.git
dot-agents sync push

# On another machine
git clone git@github.com:YOU/agents-config.git ~/.agents
dot-agents add ~/Github/myproject  # Re-link your projects
```

## Supported Agents

| Agent | Status | Config Files |
|-------|--------|--------------|
| **Cursor** | ✅ Full | `.cursor/rules/*.mdc` |
| **Claude Code** | ✅ Full | `CLAUDE.md`, `.claude/` |
| **Codex** | ✅ Full | `AGENTS.md` |
| **OpenCode** | ⚠️ Basic | Detection only |

## Requirements

- **macOS** or **Linux**
- **Bash** 3.2+ (ships with macOS)
- **jq** (recommended, for JSON features)
- **git** (for sync features)

## Configuration

### config.json

```json
{
  "schema_version": "1.0",
  "projects": {
    "myproject": {
      "path": "/Users/you/Github/myproject",
      "added": "2026-01-10T10:00:00Z"
    }
  },
  "defaults": {
    "link_type": "auto"
  },
  "features": {
    "tasks": false,
    "history": false
  }
}
```

## FAQ

**Q: Why hard links for Cursor?**

Cursor's rule system doesn't follow symlinks. Hard links share the actual file content, so changes sync automatically.

**Q: Can I use this with existing projects?**

Yes! `dot-agents add` won't overwrite existing files unless you use `--force`.

**Q: Is my config private?**

Yes. Everything stays in `~/.agents/` on your machine. Git sync is optional and to your own repo.

**Q: What if I don't use all the agents?**

That's fine! dot-agents only creates config files for agents it detects or that you have rules for.

## Contributing

Contributions welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) first.

## License

[MIT](LICENSE)

---

Built for developers who use AI coding agents daily.
