package platform

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadPluginSpecParsesCanonicalManifest(t *testing.T) {
	tmp := t.TempDir()
	pluginDir := filepath.Join(tmp, "review-toolkit")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatal(err)
	}

	manifest := `kind: package
name: review-toolkit
version: 0.1.0
display_name: Review Toolkit
description: Review helpers for code workflows.
authors:
  - Nikash Prakash
platforms:
  - codex
  - claude
resources:
  agents:
    - reviewer
  skills:
    - review-pr
  commands:
    - pr-summary
  hooks:
    - lint-on-edit
  mcp:
    - github
marketplace:
  repo: https://github.com/example/review-toolkit
  tags:
    - review
    - git
platform_overrides:
  codex:
    app_id: review-toolkit
`
	if err := os.WriteFile(filepath.Join(pluginDir, PluginManifestName), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}

	spec, err := LoadPluginSpec(pluginDir)
	if err != nil {
		t.Fatalf("LoadPluginSpec failed: %v", err)
	}

	if spec.Kind != PluginKindPackage {
		t.Fatalf("Kind = %q, want %q", spec.Kind, PluginKindPackage)
	}
	if spec.Name != "review-toolkit" {
		t.Fatalf("Name = %q, want review-toolkit", spec.Name)
	}
	if !reflect.DeepEqual(spec.Platforms, []string{"claude", "codex"}) {
		t.Fatalf("Platforms = %v, want [claude codex]", spec.Platforms)
	}
	if !reflect.DeepEqual(spec.Resources.Commands, []string{"pr-summary"}) {
		t.Fatalf("Commands = %v, want [pr-summary]", spec.Resources.Commands)
	}
	if got := spec.PlatformOverrides["codex"]["app_id"]; got != "review-toolkit" {
		t.Fatalf("codex app_id = %v, want review-toolkit", got)
	}
}

func TestLoadPluginSpecRejectsUnknownPlatform(t *testing.T) {
	tmp := t.TempDir()
	pluginDir := filepath.Join(tmp, "bad-plugin")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, PluginManifestName), []byte("kind: package\nname: bad-plugin\nplatforms:\n  - unknown\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadPluginSpec(pluginDir); err == nil {
		t.Fatal("expected invalid platform error")
	}
}

func TestListPluginSpecsReturnsSortedScopedBundles(t *testing.T) {
	tmp := t.TempDir()
	agentsHome := filepath.Join(tmp, ".agents")
	scopeDir := filepath.Join(agentsHome, "plugins", "global")
	for name := range map[string]struct{}{
		"beta":  {},
		"alpha": {},
	} {
		pluginDir := filepath.Join(scopeDir, name)
		if err := os.MkdirAll(pluginDir, 0755); err != nil {
			t.Fatal(err)
		}
		manifest := "kind: package\nname: " + name + "\nplatforms:\n  - claude\n"
		if err := os.WriteFile(filepath.Join(pluginDir, PluginManifestName), []byte(manifest), 0644); err != nil {
			t.Fatal(err)
		}
	}

	ignoredDir := filepath.Join(scopeDir, "scratch")
	if err := os.MkdirAll(ignoredDir, 0755); err != nil {
		t.Fatal(err)
	}

	specs, err := ListPluginSpecs(agentsHome, "global")
	if err != nil {
		t.Fatalf("ListPluginSpecs failed: %v", err)
	}

	if len(specs) != 2 {
		t.Fatalf("len(specs) = %d, want 2", len(specs))
	}
	if specs[0].Name != "alpha" || specs[1].Name != "beta" {
		t.Fatalf("sorted names = [%s %s], want [alpha beta]", specs[0].Name, specs[1].Name)
	}
	if specs[0].Scope != "global" || specs[1].Scope != "global" {
		t.Fatalf("scopes = [%s %s], want [global global]", specs[0].Scope, specs[1].Scope)
	}
}
