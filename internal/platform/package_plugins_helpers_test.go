package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPreferredPackagePluginsForPlatformPrefersProjectScope(t *testing.T) {
	agentsHome := t.TempDir()
	writePackagePluginManifestFixture(t, filepath.Join(agentsHome, "plugins", "global", "global-plugin"), ""+
		"kind: package\n"+
		"name: global-plugin\n"+
		"platforms:\n"+
		"  - claude\n")
	writePackagePluginManifestFixture(t, filepath.Join(agentsHome, "plugins", "proj", "project-plugin"), ""+
		"kind: package\n"+
		"name: project-plugin\n"+
		"platforms:\n"+
		"  - claude\n")

	specs, scope, err := preferredPackagePluginsForPlatform(agentsHome, "proj", "claude")
	if err != nil {
		t.Fatalf("preferredPackagePluginsForPlatform returned error: %v", err)
	}
	if scope != "proj" {
		t.Fatalf("scope = %q, want proj", scope)
	}
	if len(specs) != 1 || specs[0].Name != "project-plugin" {
		t.Fatalf("unexpected specs: %+v", specs)
	}
}

func TestPreferredPackagePluginsForPlatformReturnsProjectScopeInNameOrder(t *testing.T) {
	agentsHome := t.TempDir()
	writePackagePluginManifestFixture(t, filepath.Join(agentsHome, "plugins", "global", "global-plugin"), ""+
		"kind: package\n"+
		"name: global-plugin\n"+
		"platforms:\n"+
		"  - claude\n")
	writePackagePluginManifestFixture(t, filepath.Join(agentsHome, "plugins", "proj", "beta"), ""+
		"kind: package\n"+
		"name: beta\n"+
		"platforms:\n"+
		"  - claude\n")
	writePackagePluginManifestFixture(t, filepath.Join(agentsHome, "plugins", "proj", "alpha"), ""+
		"kind: package\n"+
		"name: alpha\n"+
		"platforms:\n"+
		"  - claude\n")

	specs, scope, err := preferredPackagePluginsForPlatform(agentsHome, "proj", "claude")
	if err != nil {
		t.Fatalf("preferredPackagePluginsForPlatform returned error: %v", err)
	}
	if scope != "proj" {
		t.Fatalf("scope = %q, want proj", scope)
	}
	if len(specs) != 2 {
		t.Fatalf("len(specs) = %d, want 2", len(specs))
	}
	if specs[0].Name != "alpha" || specs[1].Name != "beta" {
		t.Fatalf("spec order = [%s %s], want [alpha beta]", specs[0].Name, specs[1].Name)
	}
}

func TestSelectedPackagePluginForPlatformReturnsNilForAmbiguousProjectScope(t *testing.T) {
	agentsHome := t.TempDir()
	writePackagePluginManifestFixture(t, filepath.Join(agentsHome, "plugins", "proj", "alpha"), ""+
		"kind: package\n"+
		"name: alpha\n"+
		"platforms:\n"+
		"  - claude\n")
	writePackagePluginManifestFixture(t, filepath.Join(agentsHome, "plugins", "proj", "beta"), ""+
		"kind: package\n"+
		"name: beta\n"+
		"platforms:\n"+
		"  - claude\n")
	writePackagePluginManifestFixture(t, filepath.Join(agentsHome, "plugins", "global", "fallback"), ""+
		"kind: package\n"+
		"name: fallback\n"+
		"platforms:\n"+
		"  - claude\n")

	spec, scope, err := selectedPackagePluginForPlatform(agentsHome, "proj", "claude")
	if err != nil {
		t.Fatalf("selectedPackagePluginForPlatform returned error: %v", err)
	}
	if spec != nil {
		t.Fatalf("spec = %+v, want nil", spec)
	}
	if scope != "proj" {
		t.Fatalf("scope = %q, want proj", scope)
	}
}

func TestSyncPluginOverlayTreeOverlaysAndPrunesManagedSymlinks(t *testing.T) {
	tmp := t.TempDir()
	base := filepath.Join(tmp, "base")
	override := filepath.Join(tmp, "override")
	dst := filepath.Join(tmp, "dst")

	writeTextFile(t, filepath.Join(base, "commands", "hello.md"), "# hello\n")
	writeTextFile(t, filepath.Join(base, "agents", "reviewer", "AGENT.md"), "# reviewer\n")
	overrideMain := filepath.Join(override, "commands", "hello.md")
	writeTextFile(t, overrideMain, "# override\n")
	writeTextFile(t, filepath.Join(override, "skills", "review", "SKILL.md"), "# skill\n")

	if err := syncPluginOverlayTree(dst, base, override); err != nil {
		t.Fatalf("syncPluginOverlayTree returned error: %v", err)
	}

	assertSymlinkTarget(t, filepath.Join(dst, "commands", "hello.md"), overrideMain)
	assertSymlinkTarget(t, filepath.Join(dst, "agents", "reviewer", "AGENT.md"), filepath.Join(base, "agents", "reviewer", "AGENT.md"))
	assertSymlinkTarget(t, filepath.Join(dst, "skills", "review", "SKILL.md"), filepath.Join(override, "skills", "review", "SKILL.md"))

	if err := os.Remove(filepath.Join(base, "agents", "reviewer", "AGENT.md")); err != nil {
		t.Fatalf("Remove(base agent): %v", err)
	}
	if err := syncPluginOverlayTree(dst, base, override); err != nil {
		t.Fatalf("syncPluginOverlayTree returned error after prune: %v", err)
	}
	assertNoFile(t, filepath.Join(dst, "agents", "reviewer", "AGENT.md"))

	if err := removeManagedPluginOverlayTree(dst, base, override); err != nil {
		t.Fatalf("removeManagedPluginOverlayTree returned error: %v", err)
	}
	assertNoFile(t, dst)
}

func writePackagePluginManifestFixture(t *testing.T, pluginDir, manifest string) {
	t.Helper()
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", pluginDir, err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, PluginManifestName), []byte(manifest), 0644); err != nil {
		t.Fatalf("WriteFile(%s): %v", filepath.Join(pluginDir, PluginManifestName), err)
	}
}
