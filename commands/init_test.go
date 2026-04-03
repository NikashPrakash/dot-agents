package commands

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestInitCanonicalDirsIncludesPluginsBucket(t *testing.T) {
	tmp := t.TempDir()
	agentsHome := filepath.Join(tmp, ".agents")

	dirs := initCanonicalDirs(agentsHome)
	want := filepath.Join(agentsHome, "plugins", "global")
	for _, dir := range dirs {
		if dir == want {
			return
		}
	}

	t.Fatalf("initCanonicalDirs() missing %s", want)
}

func TestInitReadmeContentMentionsPluginsBucket(t *testing.T) {
	readme := initReadmeContent()
	if !strings.Contains(readme, "`plugins/` for canonical plugin bundles") {
		t.Fatalf("initReadmeContent() missing plugins bucket description:\n%s", readme)
	}
}
