package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/NikashPrakash/dot-agents/internal/platform"
	"go.yaml.in/yaml/v3"
)

const (
	canonicalImportProject = "proj"
	promptLogJSON          = "prompt-log.json"
	yamlUnmarshalFailedFmt = "yaml.Unmarshal failed: %v\n%s"
)

func TestMapGlobalRelToDest(t *testing.T) {
	cases := []struct {
		rel  string
		want string
	}{
		{".claude/settings.json", "settings/global/claude-code.json"},
		{".cursor/settings.json", "settings/global/cursor.json"},
		{".cursor/mcp.json", "mcp/global/mcp.json"},
		{".claude/CLAUDE.md", "rules/global/agents.md"},
		{".codex/config.toml", "settings/global/codex.toml"},
		{".codex/hooks.json", "hooks/global/codex.json"},
		{".cursor/hooks.json", "hooks/global/cursor.json"},
		{".unknown", ""},
	}

	for _, c := range cases {
		got := mapGlobalRelToDest(c.rel)
		if got != c.want {
			t.Fatalf("mapGlobalRelToDest(%q)=%q, want %q", c.rel, got, c.want)
		}
	}
}

func TestMapResourceRelToDestHooks(t *testing.T) {
	project := "my-project"
	cases := []struct {
		rel  string
		want string
	}{
		{relCursorHooksJSON, agentsHooksPrefix + project + "/cursor.json"},
		{relCodexHooksJSON, agentsHooksPrefix + project + "/codex.json"},
		{".github/hooks/pre-tool.json", agentsHooksPrefix + project + "/pre-tool/HOOK.yaml"},
		{".github/hooks/post-save.json", agentsHooksPrefix + project + "/post-save/HOOK.yaml"},
	}

	for _, c := range cases {
		got := mapResourceRelToDest(project, c.rel)
		if got != c.want {
			t.Fatalf("mapResourceRelToDest(%q, %q)=%q, want %q", project, c.rel, got, c.want)
		}
	}
}

func TestCanonicalHookBundleContentFromCopilotFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, promptLogJSON)
	if err := os.WriteFile(src, []byte(`{
  "version": 1,
  "hooks": {
    "userPromptSubmitted": [
      {
        "type": "command",
        "bash": "./prompt-log.sh",
        "timeoutSec": 5
      }
    ]
  }
}
`), 0644); err != nil {
		t.Fatal(err)
	}

	content, err := canonicalHookBundleContentFromCopilotFile(src, "prompt-log")
	if err != nil {
		t.Fatalf("canonicalHookBundleContentFromCopilotFile failed: %v", err)
	}

	var manifest map[string]any
	if err := yaml.Unmarshal(content, &manifest); err != nil {
		t.Fatalf(yamlUnmarshalFailedFmt, err, string(content))
	}

	if got := manifest["name"]; got != "prompt-log" {
		t.Fatalf("name = %#v, want prompt-log", got)
	}
	if got := manifest["when"]; got != "user_prompt_submit" {
		t.Fatalf("when = %#v, want user_prompt_submit", got)
	}
	run, ok := manifest["run"].(map[string]any)
	if !ok {
		t.Fatalf("run missing from manifest: %#v", manifest)
	}
	if got := run["command"]; got != "./prompt-log.sh" {
		t.Fatalf("run.command = %#v, want ./prompt-log.sh", got)
	}
	if got := run["timeout_ms"]; got != 5000 {
		t.Fatalf("run.timeout_ms = %#v, want 5000", got)
	}
}

func TestCanonicalImportOutputsFromCursorHooksJSON(t *testing.T) {
	outputs, ok := canonicalImportFromJSON(t, relCursorHooksJSON, `{
  "version": 1,
  "hooks": {
    "preToolUse": [
      {
        "command": "./guard.sh",
        "matcher": "Bash",
        "timeout": 7
      }
    ]
  }
}
	`)
	assertSingleCanonicalOutput(t, outputs, ok, "hooks/proj/pre-tool-use-guard/HOOK.yaml")

	manifest := mustUnmarshalYAMLMap(t, outputs[0].content)
	if got := manifest["when"]; got != "pre_tool_use" {
		t.Fatalf("when = %#v, want pre_tool_use", got)
	}
	run := manifest["run"].(map[string]any)
	if got := run["command"]; got != "./guard.sh" {
		t.Fatalf("run.command = %#v, want ./guard.sh", got)
	}
	if got := run["timeout_ms"]; got != 7000 {
		t.Fatalf("run.timeout_ms = %#v, want 7000", got)
	}
}

func TestCanonicalImportOutputsFromCodexHooksJSON(t *testing.T) {
	outputs, ok := canonicalImportFromJSON(t, relCodexHooksJSON, `{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "*",
        "hooks": [
          {
            "type": "command",
            "command": "./banner.sh"
          }
        ]
      }
    ]
  }
}
	`)
	assertSingleCanonicalOutput(t, outputs, ok, "hooks/proj/session-start-banner/HOOK.yaml")
}

func TestCanonicalImportOutputsFromClaudeCompatSettings(t *testing.T) {
	outputs, ok := canonicalImportFromJSON(t, relClaudeSettingsLocal, `{
  "$schema": "https://json.schemastore.org/claude-code-settings.json",
  "hooks": {
    "SessionStart": [
      {
        "matcher": "Write|Edit",
        "hooks": [
          {
            "type": "command",
            "command": "./banner.sh"
          }
        ]
      }
    ]
  }
}
		`)
	assertSingleCanonicalOutput(t, outputs, ok, "hooks/proj/session-start-banner/HOOK.yaml")

	manifest := mustUnmarshalYAMLMap(t, outputs[0].content)
	if got := manifest["when"]; got != "session_start" {
		t.Fatalf("when = %#v, want session_start", got)
	}
	if manifest["enabled_on"] == nil {
		t.Fatalf("expected enabled_on in manifest")
	}
}

func TestCanonicalImportOutputsFromOpenCodePluginFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, relOpenCodePluginsDir, "review-toolkit", "index.ts")
	if err := os.MkdirAll(filepath.Dir(src), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("export default {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	outputs, ok := canonicalImportFromPath(t, dir, src)
	if !ok || len(outputs) != 2 {
		t.Fatalf("expected two canonical outputs, ok=%v len=%d", ok, len(outputs))
	}
	if got, want := outputs[0].destRel, "plugins/proj/review-toolkit/PLUGIN.yaml"; got != want {
		t.Fatalf("manifest destRel = %q, want %q", got, want)
	}
	if got, want := outputs[1].destRel, "plugins/proj/review-toolkit/files/index.ts"; got != want {
		t.Fatalf("file destRel = %q, want %q", got, want)
	}

	var manifest map[string]any
	if err := yaml.Unmarshal(outputs[0].content, &manifest); err != nil {
		t.Fatalf(yamlUnmarshalFailedFmt, err, string(outputs[0].content))
	}
	if got := manifest["kind"]; got != "native" {
		t.Fatalf("kind = %#v, want native", got)
	}
	if got := manifest["name"]; got != "review-toolkit" {
		t.Fatalf("name = %#v, want review-toolkit", got)
	}
}

func TestCanonicalImportOutputsFromClaudePackagePluginManifestAndResources(t *testing.T) {
	sourceRoot := t.TempDir()
	pluginDir := filepath.Join(sourceRoot, relClaudePluginDir[:len(relClaudePluginDir)-1])
	writePackagePluginFixture(t, filepath.Join(pluginDir, "plugin.json"), `{
  "name": "claude-review",
  "version": "1.2.3",
  "description": "Claude review toolkit",
  "homepage": "https://example.com/claude-review",
  "repository": "https://github.com/example/claude-review",
  "license": "MIT",
  "keywords": ["review", "claude"]
}
`)
	writePackagePluginFixture(t, filepath.Join(pluginDir, "commands", "run.md"), "# run\n")
	writePackagePluginFixture(t, filepath.Join(pluginDir, ".mcp.json"), "{\"mcp\":true}\n")
	writePackagePluginFixture(t, filepath.Join(pluginDir, "README.md"), "claude overlay\n")

	manifestOutputs, ok := canonicalImportFromPath(t, sourceRoot, filepath.Join(pluginDir, "plugin.json"))
	if !ok || len(manifestOutputs) != 2 {
		t.Fatalf("expected two canonical outputs, ok=%v len=%d", ok, len(manifestOutputs))
	}
	assertDestRel(t, manifestOutputs[0], "plugins/proj/claude-review/PLUGIN.yaml")
	assertDestRel(t, manifestOutputs[1], "plugins/proj/claude-review/platforms/claude/plugin.json")

	manifest := mustUnmarshalYAMLMap(t, manifestOutputs[0].content)
	if got := manifest["kind"]; got != "package" {
		t.Fatalf("kind = %#v, want package", got)
	}
	if got := manifest["name"]; got != "claude-review" {
		t.Fatalf("name = %#v, want claude-review", got)
	}
	if got := manifest["platforms"]; got == nil {
		t.Fatalf("expected platforms in canonical manifest")
	}

	componentOutputs, ok := canonicalImportFromPath(t, sourceRoot, filepath.Join(pluginDir, "commands", "run.md"))
	if !ok || len(componentOutputs) != 1 {
		t.Fatalf("expected one canonical component output, ok=%v len=%d", ok, len(componentOutputs))
	}
	assertDestRel(t, componentOutputs[0], "plugins/proj/claude-review/resources/commands/run.md")

	overlayOutputs, ok := canonicalImportFromPath(t, sourceRoot, filepath.Join(pluginDir, "README.md"))
	if !ok || len(overlayOutputs) != 1 {
		t.Fatalf("expected one canonical overlay output, ok=%v len=%d", ok, len(overlayOutputs))
	}
	assertDestRel(t, overlayOutputs[0], "plugins/proj/claude-review/platforms/claude/README.md")
}

func TestCanonicalImportOutputsFromCursorPackagePluginManifestAndResources(t *testing.T) {
	sourceRoot := t.TempDir()
	pluginDir := filepath.Join(sourceRoot, relCursorPluginDir[:len(relCursorPluginDir)-1])
	writePackagePluginFixture(t, filepath.Join(pluginDir, "plugin.json"), `{
  "name": "cursor-review",
  "version": "0.4.0",
  "description": "Cursor review toolkit",
  "homepage": "https://example.com/cursor-review",
  "repository": "https://github.com/example/cursor-review",
  "license": "Apache-2.0",
  "keywords": ["review", "cursor"]
}
`)
	writePackagePluginFixture(t, filepath.Join(pluginDir, "rules", "global.mdc"), "---\ndescription: global\n---\n")
	writePackagePluginFixture(t, filepath.Join(pluginDir, "mcp.json"), "{\"mcp\":true}\n")
	writePackagePluginFixture(t, filepath.Join(pluginDir, "README.md"), "cursor overlay\n")

	manifestOutputs, ok := canonicalImportFromPath(t, sourceRoot, filepath.Join(pluginDir, "plugin.json"))
	if !ok || len(manifestOutputs) != 2 {
		t.Fatalf("expected two canonical outputs, ok=%v len=%d", ok, len(manifestOutputs))
	}
	assertDestRel(t, manifestOutputs[0], "plugins/proj/cursor-review/PLUGIN.yaml")
	assertDestRel(t, manifestOutputs[1], "plugins/proj/cursor-review/platforms/cursor/plugin.json")

	componentOutputs, ok := canonicalImportFromPath(t, sourceRoot, filepath.Join(pluginDir, "rules", "global.mdc"))
	if !ok || len(componentOutputs) != 1 {
		t.Fatalf("expected one canonical component output, ok=%v len=%d", ok, len(componentOutputs))
	}
	assertDestRel(t, componentOutputs[0], "plugins/proj/cursor-review/resources/rules/global.mdc")

	overlayOutputs, ok := canonicalImportFromPath(t, sourceRoot, filepath.Join(pluginDir, "README.md"))
	if !ok || len(overlayOutputs) != 1 {
		t.Fatalf("expected one canonical overlay output, ok=%v len=%d", ok, len(overlayOutputs))
	}
	assertDestRel(t, overlayOutputs[0], "plugins/proj/cursor-review/platforms/cursor/README.md")
}

func TestCanonicalImportOutputsFromCodexAndCopilotPackagePluginTrees(t *testing.T) {
	sourceRoot := t.TempDir()
	codexDir := filepath.Join(sourceRoot, relCodexPluginDir[:len(relCodexPluginDir)-1])
	writePackagePluginFixture(t, filepath.Join(codexDir, "plugin.json"), `{
  "name": "codex-review",
  "version": "2.0.0",
  "description": "Codex review toolkit",
  "repository": "https://github.com/example/codex-review",
  "license": "MIT",
  "keywords": ["review", "codex"]
}
`)
	writePackagePluginFixture(t, filepath.Join(codexDir, "skills", "review", "SKILL.md"), "# skill\n")
	writePackagePluginFixture(t, filepath.Join(sourceRoot, relCodexPluginMarket), `{
  "name": "codex-review-marketplace",
  "plugins": [
    {
      "name": "codex-review",
      "source": {
        "source": "local",
        "path": "."
      }
    }
  ]
}
`)

	codexManifestOutputs, ok := canonicalImportFromPath(t, sourceRoot, filepath.Join(codexDir, "plugin.json"))
	if !ok || len(codexManifestOutputs) != 2 {
		t.Fatalf("expected two canonical codex outputs, ok=%v len=%d", ok, len(codexManifestOutputs))
	}
	assertDestRel(t, codexManifestOutputs[0], "plugins/proj/codex-review/PLUGIN.yaml")
	assertDestRel(t, codexManifestOutputs[1], "plugins/proj/codex-review/platforms/codex/plugin.json")

	codexComponentOutputs, ok := canonicalImportFromPath(t, sourceRoot, filepath.Join(codexDir, "skills", "review", "SKILL.md"))
	if !ok || len(codexComponentOutputs) != 1 {
		t.Fatalf("expected one canonical codex component output, ok=%v len=%d", ok, len(codexComponentOutputs))
	}
	assertDestRel(t, codexComponentOutputs[0], "plugins/proj/codex-review/resources/skills/review/SKILL.md")

	codexMarketplaceOutputs, ok := canonicalImportFromPath(t, sourceRoot, filepath.Join(sourceRoot, relCodexPluginMarket))
	if !ok || len(codexMarketplaceOutputs) != 1 {
		t.Fatalf("expected one canonical codex marketplace output, ok=%v len=%d", ok, len(codexMarketplaceOutputs))
	}
	assertDestRel(t, codexMarketplaceOutputs[0], "plugins/proj/codex-review/platforms/codex/marketplace.json")

	copilotDir := sourceRoot
	writePackagePluginFixture(t, filepath.Join(copilotDir, "plugin.json"), `{
  "name": "copilot-review",
  "version": "3.1.0",
  "description": "Copilot review toolkit",
  "repository": "https://github.com/example/copilot-review",
  "license": "MIT",
  "keywords": ["review", "copilot"]
}
`)
	writePackagePluginFixture(t, filepath.Join(copilotDir, "agents", "reviewer", "AGENT.md"), "# reviewer\n")
	writePackagePluginFixture(t, filepath.Join(copilotDir, ".github", "plugin", "marketplace.json"), `{
  "name": "copilot-review-copilot-marketplace",
  "plugins": [
    {
      "name": "copilot-review",
      "source": "."
    }
  ]
}
`)

	copilotManifestOutputs, ok := canonicalImportFromPath(t, sourceRoot, filepath.Join(copilotDir, "plugin.json"))
	if !ok || len(copilotManifestOutputs) != 2 {
		t.Fatalf("expected two canonical copilot outputs, ok=%v len=%d", ok, len(copilotManifestOutputs))
	}
	assertDestRel(t, copilotManifestOutputs[0], "plugins/proj/copilot-review/PLUGIN.yaml")
	assertDestRel(t, copilotManifestOutputs[1], "plugins/proj/copilot-review/platforms/copilot/plugin.json")

	copilotComponentOutputs, ok := canonicalImportFromPath(t, sourceRoot, filepath.Join(copilotDir, "agents", "reviewer", "AGENT.md"))
	if !ok || len(copilotComponentOutputs) != 1 {
		t.Fatalf("expected one canonical copilot component output, ok=%v len=%d", ok, len(copilotComponentOutputs))
	}
	assertDestRel(t, copilotComponentOutputs[0], "plugins/proj/copilot-review/resources/agents/reviewer/AGENT.md")

	copilotMarketplaceOutputs, ok := canonicalImportFromPath(t, sourceRoot, filepath.Join(copilotDir, ".github", "plugin", "marketplace.json"))
	if !ok || len(copilotMarketplaceOutputs) != 1 {
		t.Fatalf("expected one canonical copilot marketplace output, ok=%v len=%d", ok, len(copilotMarketplaceOutputs))
	}
	assertDestRel(t, copilotMarketplaceOutputs[0], "plugins/proj/copilot-review/platforms/copilot/marketplace.json")
}

func TestCanonicalImportOutputsFromManifestDeclaredCodexAndCopilotDirectPaths(t *testing.T) {
	sourceRoot := t.TempDir()

	writePackagePluginFixture(t, filepath.Join(sourceRoot, ".codex-plugin", "plugin.json"), `{
  "name": "codex-review",
  "skills": "./dev/skills/",
  "hooks": "./dev/runtime/codex-hooks.json",
  "mcpServers": "./config/codex-mcp.json",
  "apps": "./config/codex-apps.json"
}
`)
	writePackagePluginFixture(t, filepath.Join(sourceRoot, "dev", "skills", "review", "SKILL.md"), "# skill\n")
	writePackagePluginFixture(t, filepath.Join(sourceRoot, "dev", "runtime", "codex-hooks.json"), "{\"hooks\":[]}\n")
	writePackagePluginFixture(t, filepath.Join(sourceRoot, "config", "codex-mcp.json"), "{\"mcp\":true}\n")
	writePackagePluginFixture(t, filepath.Join(sourceRoot, "config", "codex-apps.json"), "{\"apps\":[]}\n")

	codexSkillOutputs, ok := canonicalImportFromPath(t, sourceRoot, filepath.Join(sourceRoot, "dev", "skills", "review", "SKILL.md"))
	if !ok || len(codexSkillOutputs) != 1 {
		t.Fatalf("expected one canonical codex direct-path output, ok=%v len=%d", ok, len(codexSkillOutputs))
	}
	assertDestRel(t, codexSkillOutputs[0], "plugins/proj/codex-review/resources/skills/review/SKILL.md")

	codexHookOutputs, ok := canonicalImportFromPath(t, sourceRoot, filepath.Join(sourceRoot, "dev", "runtime", "codex-hooks.json"))
	if !ok || len(codexHookOutputs) != 1 {
		t.Fatalf("expected one canonical codex hook output, ok=%v len=%d", ok, len(codexHookOutputs))
	}
	assertDestRel(t, codexHookOutputs[0], "plugins/proj/codex-review/platforms/codex/hooks.json")

	codexMCPOutputs, ok := canonicalImportFromPath(t, sourceRoot, filepath.Join(sourceRoot, "config", "codex-mcp.json"))
	if !ok || len(codexMCPOutputs) != 1 {
		t.Fatalf("expected one canonical codex mcp output, ok=%v len=%d", ok, len(codexMCPOutputs))
	}
	assertDestRel(t, codexMCPOutputs[0], "plugins/proj/codex-review/platforms/codex/.mcp.json")

	codexAppsOutputs, ok := canonicalImportFromPath(t, sourceRoot, filepath.Join(sourceRoot, "config", "codex-apps.json"))
	if !ok || len(codexAppsOutputs) != 1 {
		t.Fatalf("expected one canonical codex apps output, ok=%v len=%d", ok, len(codexAppsOutputs))
	}
	assertDestRel(t, codexAppsOutputs[0], "plugins/proj/codex-review/platforms/codex/.app.json")

	writePackagePluginFixture(t, filepath.Join(sourceRoot, "plugin.json"), `{
  "name": "copilot-review",
  "agents": "./copilot/agents/",
  "skills": "./copilot/skills/",
  "commands": "./copilot/commands/",
  "hooks": "./copilot/runtime/hooks.json",
  "mcpServers": "./copilot/config/mcp.json"
}
`)
	writePackagePluginFixture(t, filepath.Join(sourceRoot, "copilot", "agents", "reviewer", "AGENT.md"), "# reviewer\n")
	writePackagePluginFixture(t, filepath.Join(sourceRoot, "copilot", "skills", "review", "SKILL.md"), "# skill\n")
	writePackagePluginFixture(t, filepath.Join(sourceRoot, "copilot", "commands", "summary.md"), "# summary\n")
	writePackagePluginFixture(t, filepath.Join(sourceRoot, "copilot", "runtime", "hooks.json"), "{\"hooks\":[]}\n")
	writePackagePluginFixture(t, filepath.Join(sourceRoot, "copilot", "config", "mcp.json"), "{\"mcp\":true}\n")

	copilotAgentOutputs, ok := canonicalImportFromPath(t, sourceRoot, filepath.Join(sourceRoot, "copilot", "agents", "reviewer", "AGENT.md"))
	if !ok || len(copilotAgentOutputs) != 1 {
		t.Fatalf("expected one canonical copilot agent output, ok=%v len=%d", ok, len(copilotAgentOutputs))
	}
	assertDestRel(t, copilotAgentOutputs[0], "plugins/proj/copilot-review/resources/agents/reviewer/AGENT.md")

	copilotSkillOutputs, ok := canonicalImportFromPath(t, sourceRoot, filepath.Join(sourceRoot, "copilot", "skills", "review", "SKILL.md"))
	if !ok || len(copilotSkillOutputs) != 1 {
		t.Fatalf("expected one canonical copilot skill output, ok=%v len=%d", ok, len(copilotSkillOutputs))
	}
	assertDestRel(t, copilotSkillOutputs[0], "plugins/proj/copilot-review/resources/skills/review/SKILL.md")

	copilotCommandOutputs, ok := canonicalImportFromPath(t, sourceRoot, filepath.Join(sourceRoot, "copilot", "commands", "summary.md"))
	if !ok || len(copilotCommandOutputs) != 1 {
		t.Fatalf("expected one canonical copilot command output, ok=%v len=%d", ok, len(copilotCommandOutputs))
	}
	assertDestRel(t, copilotCommandOutputs[0], "plugins/proj/copilot-review/resources/commands/summary.md")

	copilotHookOutputs, ok := canonicalImportFromPath(t, sourceRoot, filepath.Join(sourceRoot, "copilot", "runtime", "hooks.json"))
	if !ok || len(copilotHookOutputs) != 1 {
		t.Fatalf("expected one canonical copilot hook output, ok=%v len=%d", ok, len(copilotHookOutputs))
	}
	assertDestRel(t, copilotHookOutputs[0], "plugins/proj/copilot-review/platforms/copilot/hooks.json")

	copilotMCPOutputs, ok := canonicalImportFromPath(t, sourceRoot, filepath.Join(sourceRoot, "copilot", "config", "mcp.json"))
	if !ok || len(copilotMCPOutputs) != 1 {
		t.Fatalf("expected one canonical copilot mcp output, ok=%v len=%d", ok, len(copilotMCPOutputs))
	}
	assertDestRel(t, copilotMCPOutputs[0], "plugins/proj/copilot-review/platforms/copilot/.mcp.json")
}

func TestCanonicalImportOutputsRejectsIncompletePackagePluginManifests(t *testing.T) {
	cases := []struct {
		name      string
		relPath   string
		manifest  string
		wantOK    bool
		wantCount int
	}{
		{
			name:      "claude root manifest without name",
			relPath:   filepath.Join(relClaudePluginDir[:len(relClaudePluginDir)-1], "plugin.json"),
			manifest:  `{"description":"missing name"}`,
			wantOK:    false,
			wantCount: 0,
		},
		{
			name:      "cursor root manifest without name",
			relPath:   filepath.Join(relCursorPluginDir[:len(relCursorPluginDir)-1], "plugin.json"),
			manifest:  `{"keywords":["cursor"]}`,
			wantOK:    false,
			wantCount: 0,
		},
		{
			name:      "codex root manifest without name",
			relPath:   filepath.Join(relCodexPluginDir[:len(relCodexPluginDir)-1], "plugin.json"),
			manifest:  `{"license":"MIT"}`,
			wantOK:    false,
			wantCount: 0,
		},
		{
			name:      "copilot marketplace without name",
			relPath:   filepath.Join(".github", "plugin", "marketplace.json"),
			manifest:  `{"plugins":[{"source":"."}]}`,
			wantOK:    false,
			wantCount: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sourceRoot := t.TempDir()
			sourcePath := filepath.Join(sourceRoot, tc.relPath)
			writePackagePluginFixture(t, sourcePath, tc.manifest)

			outputs, ok := canonicalImportFromPath(t, sourceRoot, sourcePath)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if len(outputs) != tc.wantCount {
				t.Fatalf("len(outputs) = %d, want %d", len(outputs), tc.wantCount)
			}
		})
	}
}

func TestGatherProjectCandidatesIncludesPackagePluginOverlayFiles(t *testing.T) {
	projectPath := t.TempDir()
	pluginDir := filepath.Join(projectPath, relClaudePluginDir[:len(relClaudePluginDir)-1])
	writePackagePluginFixture(t, filepath.Join(pluginDir, "plugin.json"), `{
  "name": "claude-review"
}
`)
	writePackagePluginFixture(t, filepath.Join(pluginDir, "README.md"), "claude overlay\n")

	candidates := gatherProjectCandidates("proj", projectPath)
	got := map[string]bool{}
	for _, candidate := range candidates {
		rel, err := filepath.Rel(projectPath, candidate.sourcePath)
		if err != nil {
			t.Fatalf("filepath.Rel failed: %v", err)
		}
		got[filepath.ToSlash(rel)] = true
	}

	for _, want := range []string{
		".claude-plugin/plugin.json",
		".claude-plugin/README.md",
	} {
		if !got[want] {
			t.Fatalf("gatherProjectCandidates missing %q in %#v", want, got)
		}
	}
}

func TestGatherProjectCandidatesIncludesManifestDeclaredPackageDirectPaths(t *testing.T) {
	projectPath := t.TempDir()
	writePackagePluginFixture(t, filepath.Join(projectPath, "plugin.json"), `{
  "name": "copilot-review",
  "agents": "./copilot/agents/",
  "commands": "./copilot/commands/",
  "hooks": "./copilot/runtime/hooks.json"
}
`)
	writePackagePluginFixture(t, filepath.Join(projectPath, "copilot", "agents", "reviewer", "AGENT.md"), "# reviewer\n")
	writePackagePluginFixture(t, filepath.Join(projectPath, "copilot", "commands", "summary.md"), "# summary\n")
	writePackagePluginFixture(t, filepath.Join(projectPath, "copilot", "runtime", "hooks.json"), "{\"hooks\":[]}\n")
	writePackagePluginFixture(t, filepath.Join(projectPath, "notes.txt"), "ambiguous root overlay\n")

	writePackagePluginFixture(t, filepath.Join(projectPath, ".codex-plugin", "plugin.json"), `{
  "name": "codex-review",
  "skills": "./dev/skills/",
  "apps": "./config/codex-apps.json"
}
`)
	writePackagePluginFixture(t, filepath.Join(projectPath, "dev", "skills", "review", "SKILL.md"), "# skill\n")
	writePackagePluginFixture(t, filepath.Join(projectPath, "config", "codex-apps.json"), "{\"apps\":[]}\n")

	candidates := gatherProjectCandidates("proj", projectPath)
	got := map[string]bool{}
	for _, candidate := range candidates {
		rel, err := filepath.Rel(projectPath, candidate.sourcePath)
		if err != nil {
			t.Fatalf("filepath.Rel failed: %v", err)
		}
		got[filepath.ToSlash(rel)] = true
	}

	for _, want := range []string{
		"copilot/agents/reviewer/AGENT.md",
		"copilot/commands/summary.md",
		"copilot/runtime/hooks.json",
		"dev/skills/review/SKILL.md",
		"config/codex-apps.json",
	} {
		if !got[want] {
			t.Fatalf("gatherProjectCandidates missing %q in %#v", want, got)
		}
	}
	if got["notes.txt"] {
		t.Fatalf("gatherProjectCandidates should not include ambiguous repo-root overlay in %#v", got)
	}
}

func TestCanonicalImportOutputsLeavesAmbiguousRepoRootPackageOverlaysDeferred(t *testing.T) {
	sourceRoot := t.TempDir()

	writePackagePluginFixture(t, filepath.Join(sourceRoot, ".codex-plugin", "plugin.json"), `{
  "name": "codex-review",
  "skills": "./skills/"
}
`)
	writePackagePluginFixture(t, filepath.Join(sourceRoot, "notes.txt"), "ambiguous codex root overlay\n")

	outputs, ok := canonicalImportFromPath(t, sourceRoot, filepath.Join(sourceRoot, "notes.txt"))
	if ok || len(outputs) != 0 {
		t.Fatalf("expected ambiguous codex repo-root overlay to stay deferred, ok=%v len=%d", ok, len(outputs))
	}

	writePackagePluginFixture(t, filepath.Join(sourceRoot, "plugin.json"), `{
  "name": "copilot-review",
  "agents": "./agents/"
}
`)
	writePackagePluginFixture(t, filepath.Join(sourceRoot, "plugin-extra.txt"), "ambiguous copilot root overlay\n")

	outputs, ok = canonicalImportFromPath(t, sourceRoot, filepath.Join(sourceRoot, "plugin-extra.txt"))
	if ok || len(outputs) != 0 {
		t.Fatalf("expected ambiguous copilot repo-root overlay to stay deferred, ok=%v len=%d", ok, len(outputs))
	}
}

func TestCanonicalImportOutputsAssignsDistinctNamesForGenericCommandsUsingMatchers(t *testing.T) {
	outputs, ok := canonicalImportFromJSON(t, relCursorHooksJSON, `{
  "version": 1,
  "hooks": {
    "preToolUse": [
      {
        "command": "./run.sh",
        "matcher": "Write|Edit"
      },
      {
        "command": "./run.sh",
        "matcher": "Bash"
      }
    ]
  }
}
	`)
	assertTwoCanonicalOutputs(t, outputs, ok,
		"hooks/proj/pre-tool-use-write-edit-run/HOOK.yaml",
		"hooks/proj/pre-tool-use-bash-run/HOOK.yaml",
	)
}

func TestCanonicalImportOutputsAppendsStableSuffixForDuplicateIdentity(t *testing.T) {
	outputs, ok := canonicalImportFromJSON(t, relCodexHooksJSON, `{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Write",
        "hooks": [
          {
            "type": "command",
            "command": "./guard.sh"
          }
        ]
      },
      {
        "matcher": "Write",
        "hooks": [
          {
            "type": "command",
            "command": "./guard.sh"
          }
        ]
      }
    ]
  }
}
	`)
	assertTwoCanonicalOutputs(t, outputs, ok,
		"hooks/proj/pre-tool-use-guard/HOOK.yaml",
		"hooks/proj/pre-tool-use-guard-2/HOOK.yaml",
	)
}

func TestCanonicalImportOutputsSplitsMultipleActionsIntoDistinctCanonicalHooks(t *testing.T) {
	outputs, ok := canonicalImportFromJSON(t, relClaudeSettingsLocal, `{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Write|Edit",
        "hooks": [
          {
            "type": "command",
            "command": "./lint.sh"
          },
          {
            "type": "command",
            "command": "./format.sh"
          }
        ]
      }
    ]
  }
}
	`)
	assertTwoCanonicalOutputs(t, outputs, ok,
		"hooks/proj/pre-tool-use-lint/HOOK.yaml",
		"hooks/proj/pre-tool-use-format/HOOK.yaml",
	)
}

func TestCanonicalImportOutputsCanonicalizesMultiActionCopilotFanoutUsingFilenameHint(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, relGitHubHooksDir, promptLogJSON)
	if err := os.MkdirAll(filepath.Dir(src), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte(`{
  "version": 1,
  "hooks": {
    "userPromptSubmitted": [
      {
        "type": "command",
        "bash": "./prompt-log.sh"
      },
      {
        "type": "command",
        "bash": "./second.sh"
      }
    ]
  }
}
`), 0644); err != nil {
		t.Fatal(err)
	}

	outputs, ok := canonicalImportFromPath(t, dir, src)
	assertTwoCanonicalOutputs(t, outputs, ok,
		"hooks/proj/prompt-log/HOOK.yaml",
		"hooks/proj/prompt-log-second/HOOK.yaml",
	)
}

func TestCanonicalImportOutputsFallsBackToLegacyWhenCopilotEventIsUnknown(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, relGitHubHooksDir, promptLogJSON)
	if err := os.MkdirAll(filepath.Dir(src), 0755); err != nil {
		t.Fatal(err)
	}
	raw := []byte(`{
  "version": 1,
  "hooks": {
    "unknownEvent": [
      {
        "type": "command",
        "bash": "./prompt-log.sh"
      }
    ]
  }
}
`)
	if err := os.WriteFile(src, raw, 0644); err != nil {
		t.Fatal(err)
	}

	outputs, ok := canonicalImportFromPath(t, dir, src)
	if !ok || len(outputs) != 1 {
		t.Fatalf("expected one fallback output, ok=%v len=%d", ok, len(outputs))
	}
	if got, want := outputs[0].destRel, "hooks/proj/prompt-log.json"; got != want {
		t.Fatalf("destRel = %q, want %q", got, want)
	}
	if string(outputs[0].content) != string(raw) {
		t.Fatalf("expected raw legacy fallback content to be preserved")
	}
}

func TestCanonicalImportOutputsUsesMatcherHintForGenericCommandName(t *testing.T) {
	outputs, ok := canonicalImportFromJSON(t, relCursorHooksJSON, `{
  "version": 1,
  "hooks": {
    "preToolUse": [
      {
        "command": "./run.sh",
        "matcher": "Bash"
      }
    ]
  }
}
	`)
	assertSingleCanonicalOutput(t, outputs, ok, "hooks/proj/pre-tool-use-bash-run/HOOK.yaml")
}

func TestCanonicalImportOutputsPreservesRawMatcherOverrideWhenNormalized(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, relClaudeSettingsLocal)
	if err := os.MkdirAll(filepath.Dir(src), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte(`{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Write | Edit",
        "hooks": [
          {
            "type": "command",
            "command": "./guard.sh"
          }
        ]
      }
    ]
  }
}
`), 0644); err != nil {
		t.Fatal(err)
	}

	outputs, ok := canonicalImportFromPath(t, dir, src)
	if !ok || len(outputs) != 1 {
		t.Fatalf("expected one canonical output, ok=%v len=%d", ok, len(outputs))
	}

	manifest := mustUnmarshalYAMLMap(t, outputs[0].content)
	match, ok := manifest["match"].(map[string]any)
	if !ok {
		t.Fatalf("expected match section in manifest: %#v", manifest)
	}
	tools, ok := match["tools"].([]any)
	if !ok || len(tools) != 2 || tools[0] != "Write" || tools[1] != "Edit" {
		t.Fatalf("match.tools = %#v, want [Write Edit]", match["tools"])
	}
	if got := match["expression"]; got != "Write | Edit" {
		t.Fatalf("match.expression = %#v, want %q", got, "Write | Edit")
	}
}

func TestRestoreFromResourcesCountedCanonicalizesGitHubHooks(t *testing.T) {
	tmp := t.TempDir()
	agentsHome := filepath.Join(tmp, ".agents")
	t.Setenv("AGENTS_HOME", agentsHome)

	resourceFile := filepath.Join(agentsHome, "resources", "proj", ".github", "hooks", "pre-tool.json")
	if err := os.MkdirAll(filepath.Dir(resourceFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(resourceFile, []byte(`{
  "version": 1,
  "hooks": {
    "preToolUse": [
      {
        "type": "command",
        "bash": "./guard.sh"
      }
    ]
  }
}
`), 0644); err != nil {
		t.Fatal(err)
	}

	restored := restoreFromResourcesCounted("proj", filepath.Join(tmp, "repo"))
	if restored != 1 {
		t.Fatalf("restoreFromResourcesCounted restored %d files, want 1", restored)
	}

	dest := filepath.Join(agentsHome, "hooks", "proj", "pre-tool", "HOOK.yaml")
	content, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("expected canonical hook bundle at %s: %v", dest, err)
	}
	var manifest map[string]any
	if err := yaml.Unmarshal(content, &manifest); err != nil {
		t.Fatalf(yamlUnmarshalFailedFmt, err, string(content))
	}
	if got := manifest["name"]; got != "pre-tool" {
		t.Fatalf("name = %#v, want pre-tool", got)
	}
	if got := manifest["when"]; got != "pre_tool_use" {
		t.Fatalf("when = %#v, want pre_tool_use", got)
	}
}

func TestRestoreFromResourcesCountedCanonicalizesCursorHooks(t *testing.T) {
	tmp := t.TempDir()
	agentsHome := filepath.Join(tmp, ".agents")
	t.Setenv("AGENTS_HOME", agentsHome)

	resourceFile := filepath.Join(agentsHome, "resources", "proj", relCursorHooksJSON)
	if err := os.MkdirAll(filepath.Dir(resourceFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(resourceFile, []byte(`{
  "version": 1,
  "hooks": {
    "preToolUse": [
      {
        "command": "./guard.sh",
        "matcher": "Bash"
      }
    ]
  }
}
`), 0644); err != nil {
		t.Fatal(err)
	}

	restored := restoreFromResourcesCounted("proj", filepath.Join(tmp, "repo"))
	if restored != 1 {
		t.Fatalf("restoreFromResourcesCounted restored %d files, want 1", restored)
	}

	dest := filepath.Join(agentsHome, "hooks", "proj", "pre-tool-use-guard", "HOOK.yaml")
	if _, err := os.Stat(dest); err != nil {
		t.Fatalf("expected canonical hook bundle at %s: %v", dest, err)
	}
}

func TestRestoreFromResourcesCountedCanonicalizesOpenCodePluginFiles(t *testing.T) {
	tmp := t.TempDir()
	agentsHome := filepath.Join(tmp, ".agents")
	t.Setenv("AGENTS_HOME", agentsHome)

	resourceFile := filepath.Join(agentsHome, "resources", "proj", relOpenCodePluginsDir, "review-toolkit", "index.ts")
	if err := os.MkdirAll(filepath.Dir(resourceFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(resourceFile, []byte("export default {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	restored := restoreFromResourcesCounted("proj", filepath.Join(tmp, "repo"))
	if restored != 2 {
		t.Fatalf("restoreFromResourcesCounted restored %d files, want 2", restored)
	}

	manifestPath := filepath.Join(agentsHome, "plugins", "proj", "review-toolkit", platform.PluginManifestName)
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf("expected plugin manifest at %s: %v", manifestPath, err)
	}
	filePath := filepath.Join(agentsHome, "plugins", "proj", "review-toolkit", "files", "index.ts")
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("expected canonical plugin file at %s: %v", filePath, err)
	}
}

func TestFilesDifferent(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.txt")
	b := filepath.Join(dir, "b.txt")
	c := filepath.Join(dir, "c.txt")
	if err := os.WriteFile(a, []byte("same"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("same"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(c, []byte("different"), 0644); err != nil {
		t.Fatal(err)
	}

	same, err := filesDifferent(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if same {
		t.Fatalf("expected equal files")
	}

	diff, err := filesDifferent(a, c)
	if err != nil {
		t.Fatal(err)
	}
	if !diff {
		t.Fatalf("expected different files")
	}
}

func canonicalImportFromJSON(t *testing.T, relPath, content string) ([]importOutput, bool) {
	t.Helper()
	sourceRoot := t.TempDir()
	sourcePath := filepath.Join(sourceRoot, relPath)
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourcePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return canonicalImportFromPath(t, sourceRoot, sourcePath)
}

func canonicalImportFromPath(t *testing.T, sourceRoot, sourcePath string) ([]importOutput, bool) {
	t.Helper()
	outputs, ok, err := canonicalImportOutputs(importCandidate{
		project:    canonicalImportProject,
		sourceRoot: sourceRoot,
		sourcePath: sourcePath,
	})
	if err != nil {
		t.Fatalf("canonicalImportOutputs failed: %v", err)
	}
	return outputs, ok
}

func writePackagePluginFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func assertDestRel(t *testing.T, output importOutput, want string) {
	t.Helper()
	if output.destRel != want {
		t.Fatalf("destRel = %q, want %q", output.destRel, want)
	}
}

func assertSingleCanonicalOutput(t *testing.T, outputs []importOutput, ok bool, wantDest string) {
	t.Helper()
	if !ok || len(outputs) != 1 {
		t.Fatalf("expected one canonical output, ok=%v len=%d", ok, len(outputs))
	}
	if got := outputs[0].destRel; got != wantDest {
		t.Fatalf("destRel = %q, want %q", got, wantDest)
	}
}

func assertTwoCanonicalOutputs(t *testing.T, outputs []importOutput, ok bool, wantFirst, wantSecond string) {
	t.Helper()
	if !ok || len(outputs) != 2 {
		t.Fatalf("expected two canonical outputs, ok=%v len=%d", ok, len(outputs))
	}
	if got := outputs[0].destRel; got != wantFirst {
		t.Fatalf("first destRel = %q, want %q", got, wantFirst)
	}
	if got := outputs[1].destRel; got != wantSecond {
		t.Fatalf("second destRel = %q, want %q", got, wantSecond)
	}
}

func mustUnmarshalYAMLMap(t *testing.T, content []byte) map[string]any {
	t.Helper()
	var manifest map[string]any
	if err := yaml.Unmarshal(content, &manifest); err != nil {
		t.Fatalf(yamlUnmarshalFailedFmt, err, string(content))
	}
	return manifest
}
