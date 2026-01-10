# ~/.agents

Your unified config layer for AI coding agents.

Managed by [dot-agents](https://github.com/dot-agents/dot-agents).

## Structure

```
~/.agents/
├── config.json           # Main configuration
├── rules/                # Agent instruction files
│   ├── global/           # Apply to all projects
│   └── {project}/        # Project-specific rules
├── settings/             # Agent settings (JSON/TOML)
│   ├── global/
│   └── {project}/
├── mcp/                  # MCP server configurations
│   ├── global/
│   └── {project}/
├── commands/             # Custom commands
├── scripts/              # Utility scripts
└── local/                # Machine-specific (gitignored)
```

## Quick Start

```bash
# Check status
dot-agents status

# Add a project
dot-agents add ~/Github/my-project

# Check health
dot-agents doctor

# See what configs are applied where
dot-agents audit
```

## Documentation

- [Specification](https://github.com/dot-agents/dot-agents/blob/main/SPEC.md)
- [Getting Started](https://github.com/dot-agents/dot-agents#readme)
