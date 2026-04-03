package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCollectBrokenLinksIncludesPackagePluginRoots(t *testing.T) {
	project := t.TempDir()
	agentsHome := t.TempDir()

	healthyFile := filepath.Join(project, ".claude-plugin", "README.md")
	if err := os.MkdirAll(filepath.Dir(healthyFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(healthyFile, []byte("ok"), 0644); err != nil {
		t.Fatal(err)
	}

	expectedBroken := []string{
		filepath.Join(".claude-plugin", "plugin.json"),
		filepath.Join(".cursor-plugin", "plugin.json"),
		filepath.Join(".codex-plugin", "plugin.json"),
		filepath.Join(".agents", "plugins", "marketplace.json"),
		"plugin.json",
		filepath.Join(".github", "plugin", "marketplace.json"),
	}

	for _, rel := range expectedBroken {
		path := filepath.Join(project, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(t.TempDir(), "missing-target")
		if err := os.Symlink(target, path); err != nil {
			t.Fatal(err)
		}
	}

	broken := collectBrokenLinks("proj", project, agentsHome)
	got := map[string]bool{}
	for _, bl := range broken {
		got[bl.linkPath] = true
	}
	for _, want := range expectedBroken {
		if !got[want] {
			t.Fatalf("collectBrokenLinks() missing %q in %#v", want, broken)
		}
	}

	ok, brokenCount := countProjectLinks("proj", project, agentsHome)
	if ok == 0 {
		t.Fatalf("countProjectLinks() ok count = 0, want > 0")
	}
	if brokenCount != len(expectedBroken) {
		t.Fatalf("countProjectLinks() broken count = %d, want %d", brokenCount, len(expectedBroken))
	}
}
