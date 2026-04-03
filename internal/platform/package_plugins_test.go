package platform

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClaudeCreateLinksEmitsAndRemovesPackagePlugin(t *testing.T) {
	tmp := t.TempDir()
	agentsHome := filepath.Join(tmp, ".agents")
	home := filepath.Join(tmp, "home")
	repo := filepath.Join(tmp, "repo")

	t.Setenv("AGENTS_HOME", agentsHome)
	t.Setenv("HOME", home)

	pluginDir := filepath.Join(agentsHome, "plugins", "proj", "claude-review")
	writeTextFile(t, filepath.Join(pluginDir, "PLUGIN.yaml"), `kind: package
name: claude-review
version: 1.2.3
description: Claude review toolkit
homepage: https://example.com/claude-review
license: MIT
platforms:
  - claude
marketplace:
  repo: https://github.com/example/claude-review
  tags:
    - review
    - claude
`)
	writeTextFile(t, filepath.Join(pluginDir, "resources", "commands", "run.md"), "# Run\n")
	writeTextFile(t, filepath.Join(pluginDir, "resources", "agents", "reviewer", "AGENT.md"), "# Reviewer\n")
	writeTextFile(t, filepath.Join(pluginDir, "resources", "skills", "review", "SKILL.md"), "# Skill\n")
	writeTextFile(t, filepath.Join(pluginDir, "resources", "hooks", "hooks.json"), "{\"hooks\":[]}\n")
	writeTextFile(t, filepath.Join(pluginDir, "resources", "mcp", ".mcp.json"), "{\"mcp\":true}\n")
	writeTextFile(t, filepath.Join(pluginDir, "files", "shared.txt"), "shared\n")
	writeTextFile(t, filepath.Join(pluginDir, "platforms", "claude", "README.md"), "claude overlay\n")
	mkdirAll(t, repo)

	specs, err := listPackagePluginsForPlatformInScope(agentsHome, "proj", "claude")
	if err != nil {
		t.Fatalf("listPackagePluginsForPlatformInScope failed: %v", err)
	}
	if len(specs) != 1 {
		t.Fatalf("expected one claude package plugin, got %d", len(specs))
	}

	if err := NewClaude().CreateLinks("proj", repo); err != nil {
		t.Fatalf("CreateLinks failed: %v", err)
	}

	manifest := filepath.Join(repo, ".claude-plugin", "plugin.json")
	marketplace := filepath.Join(repo, ".claude-plugin", "marketplace.json")
	assertPackagePluginManifestContains(t, manifest, []string{
		`"name": "claude-review"`,
		`"version": "1.2.3"`,
		`"description": "Claude review toolkit"`,
		`"homepage": "https://example.com/claude-review"`,
		`"repository": "https://github.com/example/claude-review"`,
		`"license": "MIT"`,
		`"keywords": [`,
		`"commands": "./commands/"`,
		`"agents": "./agents/"`,
		`"skills": "./skills/"`,
		`"hooks": "./hooks/hooks.json"`,
		`"mcpServers": "./.mcp.json"`,
	})
	assertPackagePluginManifestContains(t, marketplace, []string{
		`"name": "claude-review-claude-marketplace"`,
		`"claude-review"`,
		`"https://github.com/example/claude-review"`,
	})
	assertSymlinkTarget(t, filepath.Join(repo, ".claude-plugin", "commands", "run.md"), filepath.Join(pluginDir, "resources", "commands", "run.md"))
	assertSymlinkTarget(t, filepath.Join(repo, ".claude-plugin", "agents", "reviewer", "AGENT.md"), filepath.Join(pluginDir, "resources", "agents", "reviewer", "AGENT.md"))
	assertSymlinkTarget(t, filepath.Join(repo, ".claude-plugin", "skills", "review", "SKILL.md"), filepath.Join(pluginDir, "resources", "skills", "review", "SKILL.md"))
	assertSymlinkTarget(t, filepath.Join(repo, ".claude-plugin", "hooks", "hooks.json"), filepath.Join(pluginDir, "resources", "hooks", "hooks.json"))
	assertSymlinkTarget(t, filepath.Join(repo, ".claude-plugin", ".mcp.json"), filepath.Join(pluginDir, "resources", "mcp", ".mcp.json"))
	assertSymlinkTarget(t, filepath.Join(repo, ".claude-plugin", "shared.txt"), filepath.Join(pluginDir, "files", "shared.txt"))
	assertSymlinkTarget(t, filepath.Join(repo, ".claude-plugin", "README.md"), filepath.Join(pluginDir, "platforms", "claude", "README.md"))

	if err := NewClaude().RemoveLinks("proj", repo); err != nil {
		t.Fatalf("RemoveLinks failed: %v", err)
	}

	assertNoFile(t, manifest)
	assertNoFile(t, filepath.Join(repo, ".claude-plugin", "commands", "run.md"))
	assertNoFile(t, filepath.Join(repo, ".claude-plugin", "agents", "reviewer", "AGENT.md"))
	assertNoFile(t, filepath.Join(repo, ".claude-plugin", "skills", "review", "SKILL.md"))
	assertNoFile(t, filepath.Join(repo, ".claude-plugin", "hooks", "hooks.json"))
	assertNoFile(t, filepath.Join(repo, ".claude-plugin", ".mcp.json"))
	assertNoFile(t, marketplace)
	assertNoFile(t, filepath.Join(repo, ".claude-plugin", "shared.txt"))
	assertNoFile(t, filepath.Join(repo, ".claude-plugin", "README.md"))
}

func TestClaudeCreateLinksPrunesPackagePluginWhenSourceDisappears(t *testing.T) {
	tmp := t.TempDir()
	agentsHome := filepath.Join(tmp, ".agents")
	home := filepath.Join(tmp, "home")
	repo := filepath.Join(tmp, "repo")

	t.Setenv("AGENTS_HOME", agentsHome)
	t.Setenv("HOME", home)

	pluginDir := filepath.Join(agentsHome, "plugins", "proj", "claude-review")
	writeTextFile(t, filepath.Join(pluginDir, "PLUGIN.yaml"), `kind: package
name: claude-review
platforms:
  - claude
marketplace:
  repo: https://github.com/example/claude-review
`)
	writeTextFile(t, filepath.Join(pluginDir, "resources", "commands", "run.md"), "# Run\n")
	mkdirAll(t, repo)

	if err := NewClaude().CreateLinks("proj", repo); err != nil {
		t.Fatalf("initial CreateLinks failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, ".claude-plugin", "plugin.json")); err != nil {
		t.Fatalf("expected initial package plugin manifest to exist: %v", err)
	}

	if err := os.Remove(filepath.Join(pluginDir, "PLUGIN.yaml")); err != nil {
		t.Fatalf("Remove(plugin manifest): %v", err)
	}
	if err := NewClaude().CreateLinks("proj", repo); err != nil {
		t.Fatalf("CreateLinks after manifest removal failed: %v", err)
	}

	assertNoFile(t, filepath.Join(repo, ".claude-plugin"))
	assertNoFile(t, filepath.Join(repo, ".claude-plugin", "marketplace.json"))
}

func TestCursorCreateLinksEmitsAndRemovesPackagePlugin(t *testing.T) {
	tmp := t.TempDir()
	agentsHome := filepath.Join(tmp, ".agents")
	home := filepath.Join(tmp, "home")
	repo := filepath.Join(tmp, "repo")

	t.Setenv("AGENTS_HOME", agentsHome)
	t.Setenv("HOME", home)

	pluginDir := filepath.Join(agentsHome, "plugins", "proj", "cursor-review")
	writeTextFile(t, filepath.Join(pluginDir, "PLUGIN.yaml"), `kind: package
name: cursor-review
version: 0.4.0
description: Cursor review toolkit
homepage: https://example.com/cursor-review
license: Apache-2.0
platforms:
  - cursor
marketplace:
  repo: https://github.com/example/cursor-review
  tags:
    - review
    - cursor
`)
	writeTextFile(t, filepath.Join(pluginDir, "resources", "rules", "global.mdc"), "---\ndescription: global rules\n---\n")
	writeTextFile(t, filepath.Join(pluginDir, "resources", "commands", "run.md"), "# Run\n")
	writeTextFile(t, filepath.Join(pluginDir, "resources", "agents", "reviewer", "AGENT.md"), "# Reviewer\n")
	writeTextFile(t, filepath.Join(pluginDir, "resources", "skills", "review", "SKILL.md"), "# Skill\n")
	writeTextFile(t, filepath.Join(pluginDir, "resources", "hooks", "hooks.json"), "{\"hooks\":[]}\n")
	writeTextFile(t, filepath.Join(pluginDir, "resources", "mcp", "mcp.json"), "{\"mcp\":true}\n")
	writeTextFile(t, filepath.Join(pluginDir, "files", "shared.txt"), "shared\n")
	writeTextFile(t, filepath.Join(pluginDir, "platforms", "cursor", "README.md"), "cursor overlay\n")
	mkdirAll(t, repo)

	specs, err := listPackagePluginsForPlatformInScope(agentsHome, "proj", "cursor")
	if err != nil {
		t.Fatalf("listPackagePluginsForPlatformInScope failed: %v", err)
	}
	if len(specs) != 1 {
		t.Fatalf("expected one cursor package plugin, got %d", len(specs))
	}

	if err := NewCursor().CreateLinks("proj", repo); err != nil {
		t.Fatalf("CreateLinks failed: %v", err)
	}

	manifest := filepath.Join(repo, ".cursor-plugin", "plugin.json")
	marketplace := filepath.Join(repo, ".cursor-plugin", "marketplace.json")
	assertPackagePluginManifestContains(t, manifest, []string{
		`"name": "cursor-review"`,
		`"version": "0.4.0"`,
		`"description": "Cursor review toolkit"`,
		`"homepage": "https://example.com/cursor-review"`,
		`"repository": "https://github.com/example/cursor-review"`,
		`"license": "Apache-2.0"`,
		`"keywords": [`,
		`"rules": "./rules/"`,
		`"commands": "./commands/"`,
		`"agents": "./agents/"`,
		`"skills": "./skills/"`,
		`"hooks": "./hooks/hooks.json"`,
		`"mcpServers": "./mcp.json"`,
	})
	assertPackagePluginManifestContains(t, marketplace, []string{
		`"name": "cursor-review-cursor-marketplace"`,
		`"cursor-review"`,
		`"https://github.com/example/cursor-review"`,
	})
	assertSymlinkTarget(t, filepath.Join(repo, ".cursor-plugin", "rules", "global.mdc"), filepath.Join(pluginDir, "resources", "rules", "global.mdc"))
	assertSymlinkTarget(t, filepath.Join(repo, ".cursor-plugin", "commands", "run.md"), filepath.Join(pluginDir, "resources", "commands", "run.md"))
	assertSymlinkTarget(t, filepath.Join(repo, ".cursor-plugin", "agents", "reviewer", "AGENT.md"), filepath.Join(pluginDir, "resources", "agents", "reviewer", "AGENT.md"))
	assertSymlinkTarget(t, filepath.Join(repo, ".cursor-plugin", "skills", "review", "SKILL.md"), filepath.Join(pluginDir, "resources", "skills", "review", "SKILL.md"))
	assertSymlinkTarget(t, filepath.Join(repo, ".cursor-plugin", "hooks", "hooks.json"), filepath.Join(pluginDir, "resources", "hooks", "hooks.json"))
	assertSymlinkTarget(t, filepath.Join(repo, ".cursor-plugin", "mcp.json"), filepath.Join(pluginDir, "resources", "mcp", "mcp.json"))
	assertSymlinkTarget(t, filepath.Join(repo, ".cursor-plugin", "shared.txt"), filepath.Join(pluginDir, "files", "shared.txt"))
	assertSymlinkTarget(t, filepath.Join(repo, ".cursor-plugin", "README.md"), filepath.Join(pluginDir, "platforms", "cursor", "README.md"))

	if err := NewCursor().RemoveLinks("proj", repo); err != nil {
		t.Fatalf("RemoveLinks failed: %v", err)
	}

	assertNoFile(t, manifest)
	assertNoFile(t, filepath.Join(repo, ".cursor-plugin", "rules", "global.mdc"))
	assertNoFile(t, filepath.Join(repo, ".cursor-plugin", "commands", "run.md"))
	assertNoFile(t, filepath.Join(repo, ".cursor-plugin", "agents", "reviewer", "AGENT.md"))
	assertNoFile(t, filepath.Join(repo, ".cursor-plugin", "skills", "review", "SKILL.md"))
	assertNoFile(t, filepath.Join(repo, ".cursor-plugin", "hooks", "hooks.json"))
	assertNoFile(t, filepath.Join(repo, ".cursor-plugin", "mcp.json"))
	assertNoFile(t, marketplace)
	assertNoFile(t, filepath.Join(repo, ".cursor-plugin", "shared.txt"))
	assertNoFile(t, filepath.Join(repo, ".cursor-plugin", "README.md"))
}

func assertPackagePluginManifestContains(t *testing.T, path string, wants []string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	got := string(content)
	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Fatalf("manifest %s missing %q:\n%s", path, want, got)
		}
	}
}
