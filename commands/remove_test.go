package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveProjectDirsIncludesPluginsBucket(t *testing.T) {
	tmp := t.TempDir()
	agentsHome := filepath.Join(tmp, ".agents")
	t.Setenv("AGENTS_HOME", agentsHome)

	pluginDir := filepath.Join(agentsHome, "plugins", "proj", "sample")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "PLUGIN.yaml"), []byte("kind: package\nname: sample\nplatforms:\n  - claude\n"), 0644); err != nil {
		t.Fatal(err)
	}

	removeProjectDirs("proj")

	if _, err := os.Stat(filepath.Join(agentsHome, "plugins", "proj")); !os.IsNotExist(err) {
		t.Fatalf("expected plugin project dir to be removed, got %v", err)
	}
}
