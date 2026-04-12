package platform

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const codexAgentMarkdownFile = "AGENT.md"

func TestRenderCodexAgentTomlUsesFrontmatterAndBody(t *testing.T) {
	tmp := t.TempDir()
	agentDir := filepath.Join(tmp, "agents", "global", "reviewer")
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		t.Fatal(err)
	}
	agentMD := filepath.Join(agentDir, codexAgentMarkdownFile)
	content := `---
name: reviewer
description: reviews changes
model: gpt-5.1-codex
is_background: true
---

# Reviewer

Use "safe" defaults and avoid shell footguns.
`
	if err := os.WriteFile(agentMD, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := renderCodexAgentToml(agentMD)
	if err != nil {
		t.Fatalf("renderCodexAgentToml failed: %v", err)
	}

	out := string(got)
	for _, want := range []string{
		`name = "reviewer"`,
		`description = "reviews changes"`,
		`model = "gpt-5.1-codex"`,
		`developer_instructions = """`,
		`# Reviewer`,
		`Use "safe" defaults and avoid shell footguns.`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("render output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "\ninstructions = ") || strings.HasPrefix(out, "instructions = ") {
		t.Fatalf("render output should not contain legacy instructions key:\n%s", out)
	}
}

func TestCodexCreateLinksWritesNativeAgentTomlAndCleansCompat(t *testing.T) {
	tmp := t.TempDir()
	agentsHome := filepath.Join(tmp, ".agents")
	home := filepath.Join(tmp, "home")
	repo := filepath.Join(tmp, "repo")

	t.Setenv("AGENTS_HOME", agentsHome)
	t.Setenv("HOME", home)

	globalAgentDir := filepath.Join(agentsHome, "agents", "global", "reviewer")
	projectAgentDir := filepath.Join(agentsHome, "agents", "proj", "implementer")
	for _, dir := range []string{globalAgentDir, projectAgentDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	mustWriteCodexFixtureFile(t, filepath.Join(globalAgentDir, codexAgentMarkdownFile), `---
name: reviewer
description: global reviewer
model: gpt-5.1-codex
---

# Reviewer
`)
	mustWriteCodexFixtureFile(t, filepath.Join(projectAgentDir, codexAgentMarkdownFile), `---
name: implementer
description: project implementer
is_background: false
---

# Implementer

Build the feature and keep tests green.
`)

	if err := os.MkdirAll(repo, 0755); err != nil {
		t.Fatal(err)
	}

	if err := NewCodex().CreateLinks("proj", repo); err != nil {
		t.Fatalf("CreateLinks failed: %v", err)
	}

	projectToml := filepath.Join(repo, ".codex", "agents", "implementer.toml")
	assertCodexFileContains(t, "project toml", projectToml, []string{
		`name = "implementer"`,
		`description = "project implementer"`,
		`Build the feature and keep tests green.`,
	})

	userToml := filepath.Join(home, ".codex", "agents", "reviewer.toml")
	assertCodexFileContains(t, "user toml", userToml, []string{
		`name = "reviewer"`,
		`description = "global reviewer"`,
		`model = "gpt-5.1-codex"`,
	})

	assertCodexPathNotExists(t, filepath.Join(repo, ".claude", "agents"), "legacy compat path should be cleaned up")

	if err := NewCodex().RemoveLinks("proj", repo); err != nil {
		t.Fatalf("RemoveLinks failed: %v", err)
	}

	assertCodexPathNotExists(t, projectToml, "project native agent should be removed")
	assertCodexPathNotExists(t, filepath.Join(repo, ".claude", "agents"), "legacy compat path should stay removed")
}

func TestCodexCreateLinksEmitsPackagePluginAndRemoveLinksCleansIt(t *testing.T) {
	tmp := t.TempDir()
	agentsHome := filepath.Join(tmp, ".agents")
	home := filepath.Join(tmp, "home")
	repo := filepath.Join(tmp, "repo")
	pluginDir := filepath.Join(agentsHome, "plugins", "proj", "codex-review")

	t.Setenv("AGENTS_HOME", agentsHome)
	t.Setenv("HOME", home)

	mustWriteCodexFixtureFile(t, filepath.Join(pluginDir, PluginManifestName), `kind: package
name: codex-review
version: 1.2.3
display_name: Codex Review
description: Review helpers for Codex.
authors:
  - Review Team
homepage: https://example.com/codex-review
license: MIT
platforms:
  - codex
marketplace:
  repo: https://github.com/example/codex-review
  tags:
    - review
    - codex
`)
	mustWriteCodexFixtureFile(t, filepath.Join(pluginDir, "resources", "skills", "review", "SKILL.md"), `---
name: review
description: review helper
---

# Review
`)
	mustWriteCodexFixtureFile(t, filepath.Join(pluginDir, "files", "notes.txt"), "from files\n")
	mustWriteCodexFixtureFile(t, filepath.Join(pluginDir, "platforms", "codex", "hooks.json"), "{\"hooks\":[]}\n")
	mustWriteCodexFixtureFile(t, filepath.Join(pluginDir, "platforms", "codex", ".mcp.json"), "{\"mcp\":true}\n")
	mustWriteCodexFixtureFile(t, filepath.Join(pluginDir, "platforms", "codex", ".app.json"), "{\"apps\":[]}\n")

	if err := os.MkdirAll(repo, 0755); err != nil {
		t.Fatal(err)
	}

	if err := NewCodex().CreateLinks("proj", repo); err != nil {
		t.Fatalf("CreateLinks failed: %v", err)
	}

	manifestPath := filepath.Join(repo, ".codex-plugin", "plugin.json")
	marketplacePath := filepath.Join(repo, ".agents", "plugins", "marketplace.json")
	assertCodexFileContains(t, "codex plugin manifest", manifestPath, []string{
		`"name": "codex-review"`,
		`"version": "1.2.3"`,
		`"description": "Review helpers for Codex."`,
		`"repository": "https://github.com/example/codex-review"`,
		`"license": "MIT"`,
		`"keywords": [`,
		`"codex"`,
		`"review"`,
		`"skills": "./skills/"`,
		`"hooks": "./hooks.json"`,
		`"mcpServers": "./.mcp.json"`,
		`"apps": "./.app.json"`,
		`"displayName": "Codex Review"`,
		`"shortDescription": "Review helpers for Codex."`,
		`"developerName": "Review Team"`,
	})
	assertCodexFileContains(t, "codex marketplace manifest", marketplacePath, []string{
		`"name": "codex-review-codex-marketplace"`,
		`"displayName": "Codex Review"`,
		`"name": "codex-review"`,
		`"source": "local"`,
		`"path": "."`,
	})

	assertCodexSymlinkTarget(t, filepath.Join(repo, "skills", "review", "SKILL.md"), filepath.Join(pluginDir, "resources", "skills", "review", "SKILL.md"))
	assertCodexSymlinkTarget(t, filepath.Join(repo, "notes.txt"), filepath.Join(pluginDir, "files", "notes.txt"))
	assertCodexSymlinkTarget(t, filepath.Join(repo, "hooks.json"), filepath.Join(pluginDir, "platforms", "codex", "hooks.json"))
	assertCodexSymlinkTarget(t, filepath.Join(repo, ".mcp.json"), filepath.Join(pluginDir, "platforms", "codex", ".mcp.json"))
	assertCodexSymlinkTarget(t, filepath.Join(repo, ".app.json"), filepath.Join(pluginDir, "platforms", "codex", ".app.json"))

	if err := NewCodex().RemoveLinks("proj", repo); err != nil {
		t.Fatalf("RemoveLinks failed: %v", err)
	}

	assertCodexPathNotExists(t, manifestPath, "codex package manifest should be removed")
	assertCodexPathNotExists(t, marketplacePath, "codex marketplace manifest should be removed")
	assertCodexPathNotExists(t, filepath.Join(repo, "skills", "review", "SKILL.md"), "codex package skill file should be removed")
	assertCodexPathNotExists(t, filepath.Join(repo, "notes.txt"), "codex package file should be removed")
	assertCodexPathNotExists(t, filepath.Join(repo, "hooks.json"), "codex package hooks file should be removed")
	assertCodexPathNotExists(t, filepath.Join(repo, ".mcp.json"), "codex package mcp file should be removed")
	assertCodexPathNotExists(t, filepath.Join(repo, ".app.json"), "codex package app file should be removed")
}

func TestCodexCreateLinksCleansPackagePluginWhenBundleDisappears(t *testing.T) {
	tmp := t.TempDir()
	agentsHome := filepath.Join(tmp, ".agents")
	home := filepath.Join(tmp, "home")
	repo := filepath.Join(tmp, "repo")
	pluginDir := filepath.Join(agentsHome, "plugins", "proj", "codex-review")

	t.Setenv("AGENTS_HOME", agentsHome)
	t.Setenv("HOME", home)

	mustWriteCodexFixtureFile(t, filepath.Join(pluginDir, PluginManifestName), `kind: package
name: codex-review
platforms:
  - codex
`)
	mustWriteCodexFixtureFile(t, filepath.Join(pluginDir, "files", "notes.txt"), "from files\n")

	if err := os.MkdirAll(repo, 0755); err != nil {
		t.Fatal(err)
	}
	if err := NewCodex().CreateLinks("proj", repo); err != nil {
		t.Fatalf("CreateLinks failed: %v", err)
	}

	assertCodexSymlinkTarget(t, filepath.Join(repo, "notes.txt"), filepath.Join(pluginDir, "files", "notes.txt"))

	if err := os.RemoveAll(pluginDir); err != nil {
		t.Fatal(err)
	}
	if err := NewCodex().CreateLinks("proj", repo); err != nil {
		t.Fatalf("CreateLinks cleanup failed: %v", err)
	}

	assertCodexPathNotExists(t, filepath.Join(repo, ".codex-plugin", "plugin.json"), "stale codex package manifest should be removed")
	assertCodexPathNotExists(t, filepath.Join(repo, "notes.txt"), "stale codex package file should be removed")
}

func mustWriteCodexFixtureFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func assertCodexFileContains(t *testing.T, label, path string, want []string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected %s at %s: %v", label, path, err)
	}
	got := string(content)
	for _, snippet := range want {
		if !strings.Contains(got, snippet) {
			t.Fatalf("%s missing %q:\n%s", label, snippet, got)
		}
	}
}

func assertCodexPathNotExists(t *testing.T, path, message string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("%s, got %v", message, err)
	}
}

func assertCodexSymlinkTarget(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.Readlink(path)
	if err != nil {
		t.Fatalf("expected %s to be a symlink: %v", path, err)
	}
	if got != want {
		t.Fatalf("expected %s to point to %s, got %s", path, want, got)
	}
}
