package platform

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/NikashPrakash/dot-agents/internal/links"
)

func TestCopilotCreateLinksEmitsPackagePluginManifestAndTrees(t *testing.T) {
	paths := newPlatformTestPaths(t)
	agentsHome := paths.agentsHome
	repo := paths.repo

	pluginDir := filepath.Join(agentsHome, "plugins", "proj", "review-toolkit")
	writeTextFile(t, filepath.Join(pluginDir, PluginManifestName), `kind: package
name: review-toolkit
version: 1.2.3
description: Review helpers for Copilot CLI
authors:
  - Nikash Prakash
homepage: https://example.com/review-toolkit
license: MIT
platforms:
  - copilot
marketplace:
  repo: https://github.com/example/review-toolkit
  tags:
    - review
    - copilot
`)

	writeTextFile(t, filepath.Join(pluginDir, "resources", "agents", "reviewer", "AGENT.md"), "# reviewer\n")
	writeTextFile(t, filepath.Join(pluginDir, "resources", "skills", "review", "SKILL.md"), "---\nname: review\n---\n")
	writeTextFile(t, filepath.Join(pluginDir, "resources", "commands", "summary.md"), "# base command\n")
	writeTextFile(t, filepath.Join(pluginDir, "files", "plugin-data.txt"), "base data\n")
	writeTextFile(t, filepath.Join(pluginDir, "platforms", "copilot", "agents", "reviewer", "AGENT.md"), "# override reviewer\n")
	writeTextFile(t, filepath.Join(pluginDir, "platforms", "copilot", "commands", "summary.md"), "# override command\n")
	writeTextFile(t, filepath.Join(pluginDir, "platforms", "copilot", "plugin-extra.txt"), "extra overlay\n")
	writeTextFile(t, filepath.Join(pluginDir, "platforms", "copilot", "hooks.json"), "{\"hooks\":[]}\n")
	writeTextFile(t, filepath.Join(pluginDir, "platforms", "copilot", ".mcp.json"), "{\"mcp\":true}\n")
	mkdirAll(t, repo)

	mustCreateLinks(t, "Copilot", NewCopilot(), fixtureProject, repo)

	manifest := readJSONFile(t, filepath.Join(repo, "plugin.json"))
	marketplace := readJSONFile(t, filepath.Join(repo, ".github", "plugin", "marketplace.json"))
	assertJSONPathEquals(t, manifest, "name", "review-toolkit")
	assertJSONPathEquals(t, manifest, "version", "1.2.3")
	assertJSONPathEquals(t, manifest, "description", "Review helpers for Copilot CLI")
	assertJSONPathEquals(t, manifest, "homepage", "https://example.com/review-toolkit")
	assertJSONPathEquals(t, manifest, "repository", "https://github.com/example/review-toolkit")
	assertJSONPathEquals(t, manifest, "license", "MIT")
	assertJSONPathEquals(t, manifest, "author.name", "Nikash Prakash")
	assertJSONPathEquals(t, manifest, "keywords.0", "copilot")
	assertJSONPathEquals(t, manifest, "keywords.1", "review")
	assertJSONPathEquals(t, manifest, "agents", "./agents/")
	assertJSONPathEquals(t, manifest, "skills", "./skills/")
	assertJSONPathEquals(t, manifest, "commands", "./commands/")
	assertJSONPathEquals(t, manifest, "hooks", "./hooks.json")
	assertJSONPathEquals(t, manifest, "mcpServers", "./.mcp.json")
	assertJSONPathEquals(t, marketplace, "name", "review-toolkit-copilot-marketplace")
	assertJSONPathEquals(t, marketplace, "plugins.0.name", "review-toolkit")
	assertJSONPathEquals(t, marketplace, "plugins.0.source", ".")

	assertSymlinkTarget(t, filepath.Join(repo, "agents", "reviewer", "AGENT.md"), filepath.Join(pluginDir, "platforms", "copilot", "agents", "reviewer", "AGENT.md"))
	assertSymlinkTarget(t, filepath.Join(repo, "skills", "review", "SKILL.md"), filepath.Join(pluginDir, "resources", "skills", "review", "SKILL.md"))
	assertSymlinkTarget(t, filepath.Join(repo, "commands", "summary.md"), filepath.Join(pluginDir, "platforms", "copilot", "commands", "summary.md"))
	assertSymlinkTarget(t, filepath.Join(repo, "plugin-data.txt"), filepath.Join(pluginDir, "files", "plugin-data.txt"))
	assertSymlinkTarget(t, filepath.Join(repo, "plugin-extra.txt"), filepath.Join(pluginDir, "platforms", "copilot", "plugin-extra.txt"))
	assertSymlinkTarget(t, filepath.Join(repo, "hooks.json"), filepath.Join(pluginDir, "platforms", "copilot", "hooks.json"))
	assertSymlinkTarget(t, filepath.Join(repo, ".mcp.json"), filepath.Join(pluginDir, "platforms", "copilot", ".mcp.json"))
}

func TestCopilotRemoveLinksRemovesPackagePluginOutputs(t *testing.T) {
	paths := newPlatformTestPaths(t)
	agentsHome := paths.agentsHome
	repo := paths.repo

	pluginDir := filepath.Join(agentsHome, "plugins", "proj", "review-toolkit")
	writeTextFile(t, filepath.Join(pluginDir, PluginManifestName), `kind: package
name: review-toolkit
platforms:
  - copilot
`)
	writeTextFile(t, filepath.Join(pluginDir, "resources", "skills", "review", "SKILL.md"), "---\nname: review\n---\n")
	writeTextFile(t, filepath.Join(pluginDir, "platforms", "copilot", "hooks.json"), "{\"hooks\":[]}\n")
	mkdirAll(t, repo)

	mustCreateLinks(t, "Copilot", NewCopilot(), fixtureProject, repo)
	mustRemoveLinks(t, "Copilot", NewCopilot(), fixtureProject, repo)

	assertNoFile(t, filepath.Join(repo, "plugin.json"))
	assertNoFile(t, filepath.Join(repo, ".github", "plugin", "marketplace.json"))
	assertNoFile(t, filepath.Join(repo, "skills", "review", "SKILL.md"))
	assertNoFile(t, filepath.Join(repo, "hooks.json"))
}

func TestCopilotCreateLinksSkipsAmbiguousPackagePluginBundlesAndCleansOutputs(t *testing.T) {
	paths := newPlatformTestPaths(t)
	agentsHome := paths.agentsHome
	repo := paths.repo

	first := filepath.Join(agentsHome, "plugins", "proj", "alpha")
	second := filepath.Join(agentsHome, "plugins", "proj", "beta")
	writeTextFile(t, filepath.Join(first, PluginManifestName), `kind: package
name: alpha
platforms:
  - copilot
`)
	writeTextFile(t, filepath.Join(second, PluginManifestName), `kind: package
name: beta
platforms:
  - copilot
`)
	staleSource := filepath.Join(first, "files", "legacy.txt")
	writeTextFile(t, staleSource, "legacy\n")
	mkdirAll(t, repo)
	if err := links.Symlink(staleSource, filepath.Join(repo, "legacy.txt")); err != nil {
		t.Fatalf("seed stale legacy symlink: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "plugin.json"), []byte("{\"name\":\"stale\"}\n"), 0644); err != nil {
		t.Fatalf("seed stale plugin.json: %v", err)
	}

	mustCreateLinks(t, "Copilot", NewCopilot(), fixtureProject, repo)

	assertNoFile(t, filepath.Join(repo, "plugin.json"))
	assertNoFile(t, filepath.Join(repo, "legacy.txt"))
}
