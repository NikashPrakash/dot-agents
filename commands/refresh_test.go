package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/NikashPrakash/dot-agents/internal/config"
)

const refreshCanonicalAgentPath = "agents/proj/my-agent/AGENT.md"

// ---------- mapResourceRelToDest ----------

func TestMapResourceRelToDest_MCPCanonicalization(t *testing.T) {
	cases := []struct {
		relPath  string
		expected string
	}{
		// All platform MCP files must normalize to the canonical mcp.json
		{".mcp.json", "mcp/proj/mcp.json"},
		{".cursor/mcp.json", "mcp/proj/mcp.json"},
		{".vscode/mcp.json", "mcp/proj/mcp.json"},
		// Other mappings must remain intact
		{".cursor/settings.json", "settings/proj/cursor.json"},
		{".cursorignore", "settings/proj/cursorignore"},
		{".claude/settings.local.json", "settings/proj/claude-code.json"},
		{"opencode.json", "settings/proj/opencode.json"},
		{"AGENTS.md", "rules/proj/agents.md"},
		{".codex/instructions.md", "rules/proj/agents.md"},
		{".codex/rules.md", "rules/proj/agents.md"},
		{".codex/config.toml", "settings/proj/codex.toml"},
		{".codex/hooks.json", "hooks/proj/codex.json"},
		{".github/copilot-instructions.md", "rules/proj/copilot-instructions.md"},
		{".github/hooks/pre-tool.json", "hooks/proj/pre-tool/HOOK.yaml"},
	}
	for _, c := range cases {
		got := mapResourceRelToDest("proj", c.relPath)
		if got != c.expected {
			t.Errorf("mapResourceRelToDest(%q) = %q, want %q", c.relPath, got, c.expected)
		}
	}
}

func TestMapResourceRelToDest_SkillsAndAgents(t *testing.T) {
	cases := []struct {
		relPath  string
		expected string
	}{
		{".agents/skills/my-skill/SKILL.md", "skills/proj/my-skill/SKILL.md"},
		{".claude/skills/my-skill/SKILL.md", "skills/proj/my-skill/SKILL.md"},
		{".github/agents/my-agent.agent.md", refreshCanonicalAgentPath},
		{".codex/agents/my-agent/AGENT.md", refreshCanonicalAgentPath},
		{".opencode/agent/my-agent.md", refreshCanonicalAgentPath},
		{".opencode/plugins/review-toolkit/index.ts", "plugins/proj/review-toolkit/files/index.ts"},
	}
	for _, c := range cases {
		got := mapResourceRelToDest("proj", c.relPath)
		if got != c.expected {
			t.Errorf("mapResourceRelToDest(%q) = %q, want %q", c.relPath, got, c.expected)
		}
	}
}

func TestMapResourceRelToDest_CursorRules(t *testing.T) {
	cases := []struct {
		relPath  string
		expected string
	}{
		{".cursor/rules/global--rules.mdc", "rules/global/rules.mdc"},
		{".cursor/rules/proj--rules.mdc", "rules/proj/rules.mdc"},
		{".cursor/rules/some-rule.mdc", "rules/proj/some-rule.mdc"},
	}
	for _, c := range cases {
		got := mapResourceRelToDest("proj", c.relPath)
		if got != c.expected {
			t.Errorf("mapResourceRelToDest(%q) = %q, want %q", c.relPath, got, c.expected)
		}
	}
}

func TestMapResourceRelToDest_PassThrough(t *testing.T) {
	cases := []struct {
		relPath  string
		expected string
	}{
		// Already under known ~/.agents dirs — pass through unchanged
		{"rules/proj/rules.mdc", "rules/proj/rules.mdc"},
		{"mcp/proj/mcp.json", "mcp/proj/mcp.json"},
		{"settings/proj/cursor.json", "settings/proj/cursor.json"},
		{"plugins/proj/review-toolkit/files/index.ts", "plugins/proj/review-toolkit/files/index.ts"},
	}
	for _, c := range cases {
		got := mapResourceRelToDest("proj", c.relPath)
		if got != c.expected {
			t.Errorf("mapResourceRelToDest(%q) = %q, want %q", c.relPath, got, c.expected)
		}
	}
}

func TestRestoreFromResourcesCountedCanonicalizesPackagePluginTrees(t *testing.T) {
	tmp := t.TempDir()
	agentsHome := filepath.Join(tmp, ".agents")
	t.Setenv("AGENTS_HOME", agentsHome)

	claudeRoot := filepath.Join(agentsHome, "resources", "proj", ".claude-plugin")
	writePackagePluginFixture(t, filepath.Join(claudeRoot, "plugin.json"), `{
  "name": "claude-review",
  "version": "1.2.3",
  "description": "Claude review toolkit",
  "repository": "https://github.com/example/claude-review",
  "license": "MIT",
  "keywords": ["review", "claude"]
}
`)
	writePackagePluginFixture(t, filepath.Join(claudeRoot, "commands", "run.md"), "# run\n")
	writePackagePluginFixture(t, filepath.Join(claudeRoot, "README.md"), "claude overlay\n")

	cursorRoot := filepath.Join(agentsHome, "resources", "proj", ".cursor-plugin")
	writePackagePluginFixture(t, filepath.Join(cursorRoot, "plugin.json"), `{
  "name": "cursor-review",
  "version": "0.4.0",
  "description": "Cursor review toolkit",
  "repository": "https://github.com/example/cursor-review",
  "license": "Apache-2.0",
  "keywords": ["review", "cursor"]
}
`)
	writePackagePluginFixture(t, filepath.Join(cursorRoot, "rules", "global.mdc"), "---\ndescription: global\n---\n")
	writePackagePluginFixture(t, filepath.Join(cursorRoot, "README.md"), "cursor overlay\n")

	codexRoot := filepath.Join(agentsHome, "resources", "proj", ".codex-plugin")
	writePackagePluginFixture(t, filepath.Join(codexRoot, "plugin.json"), `{
  "name": "codex-review",
  "version": "2.0.0",
  "description": "Codex review toolkit",
  "repository": "https://github.com/example/codex-review",
  "license": "MIT",
  "keywords": ["review", "codex"]
}
`)
	writePackagePluginFixture(t, filepath.Join(codexRoot, "skills", "review", "SKILL.md"), "# skill\n")
	writePackagePluginFixture(t, filepath.Join(agentsHome, "resources", "proj", ".agents", "plugins", "marketplace.json"), `{
  "name": "codex-review-marketplace",
  "plugins": [
    {
      "name": "codex-review",
      "source": {
        "source": "local",
        "path": "."
      }
    }
  ]
}
`)

	copilotRoot := filepath.Join(agentsHome, "resources", "proj")
	writePackagePluginFixture(t, filepath.Join(copilotRoot, "plugin.json"), `{
  "name": "copilot-review",
  "version": "3.1.0",
  "description": "Copilot review toolkit",
  "repository": "https://github.com/example/copilot-review",
  "license": "MIT",
  "keywords": ["review", "copilot"]
}
`)
	writePackagePluginFixture(t, filepath.Join(copilotRoot, "agents", "reviewer", "AGENT.md"), "# reviewer\n")
	writePackagePluginFixture(t, filepath.Join(copilotRoot, ".github", "plugin", "marketplace.json"), `{
  "name": "copilot-review-copilot-marketplace",
  "plugins": [
    {
      "name": "copilot-review",
      "source": "."
    }
  ]
}
`)

	restored := restoreFromResourcesCounted("proj", filepath.Join(tmp, "repo"))
	if restored < 15 {
		t.Fatalf("restoreFromResourcesCounted restored %d files, want at least 15", restored)
	}

	checks := []string{
		filepath.Join(agentsHome, "plugins", "proj", "claude-review", "PLUGIN.yaml"),
		filepath.Join(agentsHome, "plugins", "proj", "claude-review", "platforms", "claude", "plugin.json"),
		filepath.Join(agentsHome, "plugins", "proj", "claude-review", "platforms", "claude", "README.md"),
		filepath.Join(agentsHome, "plugins", "proj", "claude-review", "resources", "commands", "run.md"),
		filepath.Join(agentsHome, "plugins", "proj", "cursor-review", "PLUGIN.yaml"),
		filepath.Join(agentsHome, "plugins", "proj", "cursor-review", "platforms", "cursor", "README.md"),
		filepath.Join(agentsHome, "plugins", "proj", "cursor-review", "resources", "rules", "global.mdc"),
		filepath.Join(agentsHome, "plugins", "proj", "codex-review", "PLUGIN.yaml"),
		filepath.Join(agentsHome, "plugins", "proj", "codex-review", "resources", "skills", "review", "SKILL.md"),
		filepath.Join(agentsHome, "plugins", "proj", "codex-review", "platforms", "codex", "marketplace.json"),
		filepath.Join(agentsHome, "plugins", "proj", "copilot-review", "PLUGIN.yaml"),
		filepath.Join(agentsHome, "plugins", "proj", "copilot-review", "platforms", "copilot", "plugin.json"),
		filepath.Join(agentsHome, "plugins", "proj", "copilot-review", "resources", "agents", "reviewer", "AGENT.md"),
		filepath.Join(agentsHome, "plugins", "proj", "copilot-review", "platforms", "copilot", "marketplace.json"),
	}
	for _, path := range checks {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected canonical package plugin file at %s: %v", path, err)
		}
	}
}

func TestRestoreFromResourcesCountedCanonicalizesManifestDeclaredPackageDirectPaths(t *testing.T) {
	tmp := t.TempDir()
	agentsHome := filepath.Join(tmp, ".agents")
	t.Setenv("AGENTS_HOME", agentsHome)

	codexRoot := filepath.Join(agentsHome, "resources", "proj")
	writePackagePluginFixture(t, filepath.Join(codexRoot, ".codex-plugin", "plugin.json"), `{
  "name": "codex-review",
  "skills": "./dev/skills/",
  "hooks": "./dev/runtime/codex-hooks.json",
  "mcpServers": "./config/codex-mcp.json",
  "apps": "./config/codex-apps.json"
}
`)
	writePackagePluginFixture(t, filepath.Join(codexRoot, "dev", "skills", "review", "SKILL.md"), "# skill\n")
	writePackagePluginFixture(t, filepath.Join(codexRoot, "dev", "runtime", "codex-hooks.json"), "{\"hooks\":[]}\n")
	writePackagePluginFixture(t, filepath.Join(codexRoot, "config", "codex-mcp.json"), "{\"mcp\":true}\n")
	writePackagePluginFixture(t, filepath.Join(codexRoot, "config", "codex-apps.json"), "{\"apps\":[]}\n")

	writePackagePluginFixture(t, filepath.Join(codexRoot, "plugin.json"), `{
  "name": "copilot-review",
  "agents": "./copilot/agents/",
  "commands": "./copilot/commands/",
  "hooks": "./copilot/runtime/hooks.json",
  "mcpServers": "./copilot/config/mcp.json"
}
`)
	writePackagePluginFixture(t, filepath.Join(codexRoot, "copilot", "agents", "reviewer", "AGENT.md"), "# reviewer\n")
	writePackagePluginFixture(t, filepath.Join(codexRoot, "copilot", "commands", "summary.md"), "# summary\n")
	writePackagePluginFixture(t, filepath.Join(codexRoot, "copilot", "runtime", "hooks.json"), "{\"hooks\":[]}\n")
	writePackagePluginFixture(t, filepath.Join(codexRoot, "copilot", "config", "mcp.json"), "{\"mcp\":true}\n")
	writePackagePluginFixture(t, filepath.Join(codexRoot, "notes.txt"), "ambiguous overlay should stay deferred\n")

	restored := restoreFromResourcesCounted("proj", filepath.Join(tmp, "repo"))
	if restored < 9 {
		t.Fatalf("restoreFromResourcesCounted restored %d files, want at least 9", restored)
	}

	for _, path := range []string{
		filepath.Join(agentsHome, "plugins", "proj", "codex-review", "resources", "skills", "review", "SKILL.md"),
		filepath.Join(agentsHome, "plugins", "proj", "codex-review", "platforms", "codex", "hooks.json"),
		filepath.Join(agentsHome, "plugins", "proj", "codex-review", "platforms", "codex", ".mcp.json"),
		filepath.Join(agentsHome, "plugins", "proj", "codex-review", "platforms", "codex", ".app.json"),
		filepath.Join(agentsHome, "plugins", "proj", "copilot-review", "resources", "agents", "reviewer", "AGENT.md"),
		filepath.Join(agentsHome, "plugins", "proj", "copilot-review", "resources", "commands", "summary.md"),
		filepath.Join(agentsHome, "plugins", "proj", "copilot-review", "platforms", "copilot", "hooks.json"),
		filepath.Join(agentsHome, "plugins", "proj", "copilot-review", "platforms", "copilot", ".mcp.json"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected canonical direct package-plugin file at %s: %v", path, err)
		}
	}

	if _, err := os.Stat(filepath.Join(agentsHome, "settings", "proj", "notes.txt")); !os.IsNotExist(err) {
		t.Fatalf("ambiguous repo-root overlay should remain deferred, stat err=%v", err)
	}
}

func TestMapResourceRelToDest_UnknownReturnsEmpty(t *testing.T) {
	got := mapResourceRelToDest("proj", ".some/unknown/path.json")
	if got != "" {
		t.Errorf("expected empty for unknown path, got %q", got)
	}
}

func TestWriteRefreshMetadataStoresDetailsInAgentsRC(t *testing.T) {
	tmp := t.TempDir()
	agentsHome := filepath.Join(tmp, ".agents")
	projectPath := filepath.Join(tmp, "repo")
	t.Setenv("AGENTS_HOME", agentsHome)

	if err := os.MkdirAll(projectPath, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectPath, ".agents-refresh"), []byte("legacy"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := writeRefreshMetadata("proj", projectPath, "1.2.3", "abcdef123456", "v1.2.3"); err != nil {
		t.Fatalf("writeRefreshMetadata: %v", err)
	}

	rc, err := config.LoadAgentsRC(projectPath)
	if err != nil {
		t.Fatalf("LoadAgentsRC: %v", err)
	}
	if rc.Refresh == nil {
		t.Fatal("Refresh metadata missing from manifest")
	}
	if rc.Refresh.Version != "1.2.3" || rc.Refresh.Commit != "abcdef123456" || rc.Refresh.Describe != "v1.2.3" {
		t.Fatalf("unexpected refresh metadata: %+v", *rc.Refresh)
	}
	if rc.Refresh.RefreshedAt == "" {
		t.Fatal("Refresh.RefreshedAt should be populated")
	}
	if _, err := os.Stat(filepath.Join(projectPath, ".agents-refresh")); !os.IsNotExist(err) {
		t.Fatalf("legacy .agents-refresh should be removed, stat err=%v", err)
	}
}
