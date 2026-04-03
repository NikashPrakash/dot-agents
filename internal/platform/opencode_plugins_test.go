package platform

import (
	"path/filepath"
	"testing"
)

func TestOpenCodeCreateLinksEmitsNativePluginsFromProjectAndGlobalScopes(t *testing.T) {
	paths := newPlatformTestPaths(t)
	agentsHome := paths.agentsHome
	home := paths.home
	repo := paths.repo

	projectPluginDir := filepath.Join(agentsHome, "plugins", "proj", "runtime-plugin")
	writeTextFile(t, filepath.Join(projectPluginDir, PluginManifestName), `kind: native
name: runtime-plugin
platforms:
  - opencode
`)
	projectSource := filepath.Join(projectPluginDir, "files", "main.js")
	projectNested := filepath.Join(projectPluginDir, "files", "lib", "util.ts")
	projectOverride := filepath.Join(projectPluginDir, "platforms", "opencode", "extra.js")
	writeTextFile(t, projectSource, "export const main = () => 'project';\n")
	writeTextFile(t, projectNested, "export const util = true;\n")
	writeTextFile(t, projectOverride, "export const extra = true;\n")

	globalPluginDir := filepath.Join(agentsHome, "plugins", "global", "global-runtime")
	writeTextFile(t, filepath.Join(globalPluginDir, PluginManifestName), `kind: native
name: global-runtime
platforms:
  - opencode
`)
	globalSource := filepath.Join(globalPluginDir, "files", "index.ts")
	writeTextFile(t, globalSource, "export const global = true;\n")

	packagePluginDir := filepath.Join(agentsHome, "plugins", "proj", "package-only")
	writeTextFile(t, filepath.Join(packagePluginDir, PluginManifestName), `kind: package
name: package-only
platforms:
  - claude
`)
	writeTextFile(t, filepath.Join(packagePluginDir, "files", "ignored.js"), "export const ignored = true;\n")

	mkdirAll(t, repo)

	mustCreateLinks(t, "OpenCode", NewOpenCode(), fixtureProject, repo)

	assertSymlinkTarget(t, filepath.Join(repo, ".opencode", "plugins", "runtime-plugin", "main.js"), projectSource)
	assertSymlinkTarget(t, filepath.Join(repo, ".opencode", "plugins", "runtime-plugin", "lib", "util.ts"), projectNested)
	assertSymlinkTarget(t, filepath.Join(repo, ".opencode", "plugins", "runtime-plugin", "extra.js"), projectOverride)
	assertSymlinkTarget(t, filepath.Join(home, ".config", "opencode", "plugins", "global-runtime", "index.ts"), globalSource)
	assertNoFile(t, filepath.Join(repo, ".opencode", "plugins", "package-only", "ignored.js"))
}

func TestOpenCodeRemoveLinksRemovesManagedNativePlugins(t *testing.T) {
	paths := newPlatformTestPaths(t)
	agentsHome := paths.agentsHome
	home := paths.home
	repo := paths.repo

	projectPluginDir := filepath.Join(agentsHome, "plugins", "proj", "runtime-plugin")
	writeTextFile(t, filepath.Join(projectPluginDir, PluginManifestName), `kind: native
name: runtime-plugin
platforms:
  - opencode
`)
	projectSource := filepath.Join(projectPluginDir, "files", "main.js")
	writeTextFile(t, projectSource, "export const main = true;\n")

	globalPluginDir := filepath.Join(agentsHome, "plugins", "global", "global-runtime")
	writeTextFile(t, filepath.Join(globalPluginDir, PluginManifestName), `kind: native
name: global-runtime
platforms:
  - opencode
`)
	globalSource := filepath.Join(globalPluginDir, "files", "index.ts")
	writeTextFile(t, globalSource, "export const global = true;\n")

	mkdirAll(t, repo)

	mustCreateLinks(t, "OpenCode", NewOpenCode(), fixtureProject, repo)
	mustRemoveLinks(t, "OpenCode", NewOpenCode(), fixtureProject, repo)

	assertNoFile(t, filepath.Join(repo, ".opencode", "plugins", "runtime-plugin", "main.js"))
	assertNoFile(t, filepath.Join(home, ".config", "opencode", "plugins", "global-runtime", "index.ts"))
}
