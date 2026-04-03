package commands

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/NikashPrakash/dot-agents/internal/config"
)

func TestSummarizeCanonicalBucketCountsPluginBundles(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, "plugins")

	pluginA := filepath.Join(root, "global", "alpha")
	pluginB := filepath.Join(root, "proj", "beta")
	ignored := filepath.Join(root, "proj", "scratch")
	for _, dir := range []string{pluginA, pluginB, ignored} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	for _, manifest := range []string{
		filepath.Join(pluginA, "PLUGIN.yaml"),
		filepath.Join(pluginB, "PLUGIN.yaml"),
	} {
		if err := os.WriteFile(manifest, []byte("kind: package\nname: test\nplatforms:\n  - claude\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	scopes, entries := summarizeCanonicalBucket(root, true, "PLUGIN.yaml")
	if scopes != 2 || entries != 2 {
		t.Fatalf("summarizeCanonicalBucket() = (%d, %d), want (2, 2)", scopes, entries)
	}
}

func TestCountManagedTreeEntriesRecursesIntoPluginFiles(t *testing.T) {
	tmp := t.TempDir()
	source := filepath.Join(tmp, "source.ts")
	if err := os.WriteFile(source, []byte("export {}"), 0644); err != nil {
		t.Fatal(err)
	}

	root := filepath.Join(tmp, ".opencode", "plugins", "review-toolkit", "lib")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	linkPath := filepath.Join(root, "index.ts")
	if err := os.Symlink(source, linkPath); err != nil {
		t.Fatal(err)
	}

	warn := 0
	if got := countManagedTreeEntries(filepath.Join(tmp, ".opencode", "plugins"), &warn); got != 1 || warn != 0 {
		t.Fatalf("countManagedTreeEntries() = (%d ok, %d warn), want (1, 0)", got, warn)
	}
}

func TestCountManagedTreeEntriesCountsPackagePluginRoots(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, ".claude-plugin")

	for _, rel := range []string{
		"plugin.json",
		"marketplace.json",
		filepath.Join("resources", "skills", "review", "SKILL.md"),
	} {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("data"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	warn := 0
	if got := countManagedTreeEntries(root, &warn); got != 3 || warn != 0 {
		t.Fatalf("countManagedTreeEntries(%s) = (%d ok, %d warn), want (3, 0)", root, got, warn)
	}
}

func TestPrintPackagePluginAuditsIncludePackageRoots(t *testing.T) {
	tmp := t.TempDir()
	project := filepath.Join(tmp, "proj")
	agentsHome := filepath.Join(tmp, "agents")
	if err := os.MkdirAll(project, 0755); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{
		filepath.Join(project, ".claude-plugin", "plugin.json"),
		filepath.Join(project, ".cursor-plugin", "plugin.json"),
		filepath.Join(project, ".codex-plugin", "plugin.json"),
		filepath.Join(project, ".agents", "plugins", "marketplace.json"),
		filepath.Join(project, "plugin.json"),
		filepath.Join(project, ".github", "plugin", "marketplace.json"),
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("data"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	out := captureStdout(t, func() {
		printClaudeAudit("proj", project, agentsHome)
		printCursorAudit("proj", project, agentsHome)
		printCodexAudit("proj", project, agentsHome)
		printCopilotAudit("proj", project)
	})

	for _, want := range []string{
		".claude-plugin/plugin.json",
		".cursor-plugin/plugin.json",
		".codex-plugin/plugin.json",
		".agents/plugins/marketplace.json",
		"plugin.json",
		".github/plugin/marketplace.json",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("audit output missing %q:\n%s", want, out)
		}
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	outC := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		outC <- buf.String()
	}()

	fn()

	_ = w.Close()
	os.Stdout = old
	out := <-outC
	_ = r.Close()
	return out
}

func TestReadRefreshTimestampPrefersAgentsRCMetadata(t *testing.T) {
	projectPath := t.TempDir()

	rc := &config.AgentsRC{
		Version: 1,
		Project: "proj",
		Sources: []config.Source{{Type: "local"}},
	}
	rc.SetRefreshMetadata("1.0.0", "abc123", "v1.0.0", mustParseUTCTime(t, "2026-03-31T05:18:11Z"))
	if err := rc.Save(projectPath); err != nil {
		t.Fatalf("rc.Save: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectPath, ".agents-refresh"), []byte("refreshed_at=2020-01-01T00:00:00Z\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if got := readRefreshTimestamp(projectPath); got != "2026-03-31 05:18 UTC" {
		t.Fatalf("readRefreshTimestamp() = %q, want %q", got, "2026-03-31 05:18 UTC")
	}
}

func TestReadRefreshTimestampFallsBackToLegacyMarker(t *testing.T) {
	projectPath := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectPath, ".agents-refresh"), []byte("refreshed_at=2026-03-31T07:45:00Z\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if got := readRefreshTimestamp(projectPath); got != "2026-03-31 07:45 UTC" {
		t.Fatalf("readRefreshTimestamp() = %q, want %q", got, "2026-03-31 07:45 UTC")
	}
}

func mustParseUTCTime(t *testing.T, ts string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		t.Fatalf("time.Parse(%q): %v", ts, err)
	}
	return parsed.UTC()
}
