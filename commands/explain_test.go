package commands

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestPrintPluginsExplanationReflectsMarketplaceSupport(t *testing.T) {
	output := captureExplainOutput(t, printPluginsExplanation)

	for _, want := range []string{
		"Package emitters exist for Claude, Cursor, Codex, and Copilot.",
		"OpenCode native plugin trees emit to",
		".claude-plugin/marketplace.json",
		".cursor-plugin/marketplace.json",
		".agents/plugins/marketplace.json",
		".github/plugin/marketplace.json",
		"status",
		"doctor",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("plugin explanation missing %q:\n%s", want, output)
		}
	}
}

func captureExplainOutput(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe failed: %v", err)
	}

	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = original

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("ReadFrom failed: %v", err)
	}
	return buf.String()
}
