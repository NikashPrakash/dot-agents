package platform

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/NikashPrakash/dot-agents/internal/config"
	"github.com/NikashPrakash/dot-agents/internal/links"
)

type claude struct{}

const (
	claudeCodeJSON          = "claude-code.json"
	claudeSettingsJSON      = "settings.json"
	claudeSettingsLocalJSON = "settings.local.json"
	claudeDir               = ".claude"
)

func NewClaude() Platform { return &claude{} }

func (c *claude) ID() string          { return "claude" }
func (c *claude) DisplayName() string { return "Claude Code" }

func (c *claude) IsInstalled() bool {
	if _, err := exec.LookPath("claude"); err == nil {
		return true
	}
	home, _ := os.UserHomeDir()
	_, err := os.Stat(filepath.Join(home, claudeDir))
	return err == nil
}

func (c *claude) Version() string {
	out, err := exec.Command("claude", "--version").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(strings.Split(string(out), "\n")[0])
}

func (c *claude) HasDeprecatedFormat(repoPath string) bool {
	_, err := os.Stat(filepath.Join(repoPath, ".claude.json"))
	return err == nil
}

func (c *claude) DeprecatedDetails(repoPath string) string {
	if c.HasDeprecatedFormat(repoPath) {
		return ".claude.json → .claude/settings.json"
	}
	return ""
}

func (c *claude) CreateLinks(project, repoPath string) error {
	agentsHome := config.AgentsHome()

	if err := c.prepareLinks(repoPath, agentsHome); err != nil {
		return err
	}

	if err := c.createRulesLinks(project, repoPath, agentsHome); err != nil {
		return err
	}
	c.linkProjectSettings(project, repoPath, agentsHome)
	c.linkProjectMCP(project, repoPath, agentsHome)

	if err := c.createAgentsLinks(project, repoPath, agentsHome); err != nil {
		return err
	}

	if err := c.createSkillsLinks(project, repoPath, agentsHome); err != nil {
		return err
	}
	return c.createPackagePluginLinks(project, repoPath, agentsHome)
}

func (c *claude) prepareLinks(repoPath, agentsHome string) error {
	if err := c.ensureUserAgents(agentsHome); err != nil {
		return err
	}
	if err := c.ensureUserRules(agentsHome); err != nil {
		return err
	}
	if err := c.ensureUserSettings(agentsHome); err != nil {
		return err
	}
	return os.MkdirAll(filepath.Join(repoPath, claudeDir, "rules"), 0755)
}

func (c *claude) linkProjectSettings(project, repoPath, agentsHome string) {
	target := filepath.Join(repoPath, claudeDir, claudeSettingsLocalJSON)
	projectBundles, err := collectCanonicalHookSpecsForPlatform(agentsHome, project, c.ID(), project)
	if err != nil {
		return
	}
	globalBundles, err := collectCanonicalHookSpecsForPlatform(agentsHome, project, c.ID(), "global")
	if err != nil {
		return
	}
	_ = emitPreferredHookFile(
		target,
		renderClaudeHookSettings,
		findClaudeSettingsHookSpec(agentsHome, project),
		directSymlinkHookMode,
		removeRenderedClaudeHookSettings,
		projectBundles,
		globalBundles,
	)
}

func (c *claude) linkProjectMCP(project, repoPath, agentsHome string) {
	if src := resolveScopedFile(agentsHome, "mcp", project, "claude.json", "mcp.json"); src != "" {
		links.Symlink(src, filepath.Join(repoPath, ".mcp.json"))
	}
}

func findClaudeSettingsHookSpec(agentsHome, scope string) *HookSpec {
	return resolveHookSpecInScope(agentsHome, []string{"hooks", "settings"}, scope, claudeCodeJSON)
}

func (c *claude) createRulesLinks(project, repoPath, agentsHome string) error {
	rulesDir := filepath.Join(repoPath, claudeDir, "rules")
	projectRulesDir := filepath.Join(agentsHome, "rules", project)

	entries, err := os.ReadDir(projectRulesDir)
	if err != nil {
		return nil // no project rules
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		ext := filepath.Ext(name)
		if ext != ".md" && ext != ".mdc" && ext != ".txt" {
			continue
		}
		stem := strings.TrimSuffix(name, ext)
		src := filepath.Join(projectRulesDir, name)
		dst := filepath.Join(rulesDir, project+"--"+stem+".md")
		links.Symlink(src, dst)
	}
	return nil
}

func (c *claude) ensureUserAgents(agentsHome string) error {
	globalAgents := filepath.Join(agentsHome, "agents", "global")
	entries, err := os.ReadDir(globalAgents)
	if err != nil {
		return nil
	}

	for _, homeRoot := range config.UserHomeRoots() {
		if err := c.ensureUserAgentsInHome(homeRoot, globalAgents, entries); err != nil {
			continue
		}
	}
	return nil
}

func (c *claude) ensureUserAgentsInHome(homeRoot, globalAgents string, entries []os.DirEntry) error {
	userAgentsDir := filepath.Join(homeRoot, claudeDir, "agents")
	if err := os.MkdirAll(userAgentsDir, 0755); err != nil {
		return err
	}
	for _, entry := range entries {
		c.linkUserAgent(globalAgents, userAgentsDir, entry)
	}
	return nil
}

func (c *claude) linkUserAgent(globalAgents, userAgentsDir string, entry os.DirEntry) {
	agentDir := filepath.Join(globalAgents, entry.Name())
	if !isClaudeAgentDir(agentDir) {
		return
	}
	target := filepath.Join(userAgentsDir, entry.Name())
	if isSymlink(target) {
		return
	}
	links.Symlink(agentDir, target)
}

func (c *claude) ensureUserRules(agentsHome string) error {
	// Priority list for source
	candidates := []string{
		filepath.Join(agentsHome, "rules", "global", "claude-code.mdc"),
		filepath.Join(agentsHome, "rules", "global", "claude-code.md"),
		filepath.Join(agentsHome, "rules", "global", "rules.mdc"),
		filepath.Join(agentsHome, "rules", "global", "rules.md"),
		filepath.Join(agentsHome, "rules", "global", "rules.txt"),
	}

	var src string
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			src = c
			break
		}
	}
	if src == "" {
		return nil
	}

	for _, homeRoot := range config.UserHomeRoots() {
		target := filepath.Join(homeRoot, claudeDir, "CLAUDE.md")
		if info, err := os.Lstat(target); err == nil && info.Mode()&os.ModeSymlink != 0 {
			continue // already a symlink
		}
		os.MkdirAll(filepath.Join(homeRoot, claudeDir), 0755)
		links.Symlink(src, target)
	}
	return nil
}

func (c *claude) ensureUserSettings(agentsHome string) error {
	globalBundles, err := collectCanonicalHookSpecsForPlatform(agentsHome, "global", c.ID(), "global")
	if err != nil {
		return err
	}
	if len(globalBundles) > 0 {
		return emitRenderedHookFileToUserHomes(globalBundles, filepath.Join(claudeDir, claudeSettingsJSON), renderClaudeHookSettings)
	}

	spec := findClaudeSettingsHookSpec(agentsHome, "global")
	if spec == nil {
		for _, homeRoot := range config.UserHomeRoots() {
			_ = removeManagedFileIf(filepath.Join(homeRoot, claudeDir, claudeSettingsJSON), isLikelyRenderedClaudeHookSettings)
		}
		return nil
	}
	for _, homeRoot := range config.UserHomeRoots() {
		target := filepath.Join(homeRoot, claudeDir, claudeSettingsJSON)
		if info, err := os.Lstat(target); err == nil && info.Mode()&os.ModeSymlink != 0 {
			continue // already a symlink, leave it
		}
		_ = emitHookSpec(spec, target, HookEmissionMode{
			Shape:     HookShapeDirect,
			Transport: HookTransportSymlink,
		})
	}
	return nil
}

func (c *claude) ensureUserSkills(agentsHome string) error {
	for _, homeRoot := range config.UserHomeRoots() {
		userSkillsDir := filepath.Join(homeRoot, claudeDir, "skills")
		if err := syncScopedDirSymlinks(agentsHome, "skills", "global", "SKILL.md", userSkillsDir); err != nil {
			return err
		}
	}
	return nil
}

func (c *claude) createAgentsLinks(project, repoPath, agentsHome string) error {
	return syncScopedDirSymlinksTargets(agentsHome, "agents", project, "AGENT.md", filepath.Join(repoPath, claudeDir, "agents"))
}

func (c *claude) createSkillsLinks(project, repoPath, agentsHome string) error {
	c.ensureUserSkills(agentsHome)
	return syncScopedDirSymlinksTargets(
		agentsHome,
		"skills",
		project,
		"SKILL.md",
		filepath.Join(repoPath, claudeDir, "skills"),
		filepath.Join(repoPath, ".agents", "skills"),
	)
}

func (c *claude) createPackagePluginLinks(project, repoPath, agentsHome string) error {
	specs, _, err := preferredPackagePluginsForPlatform(agentsHome, project, c.ID())
	if err != nil {
		return err
	}

	root := filepath.Join(repoPath, ".claude-plugin")
	if len(specs) != 1 {
		return c.removePackagePluginLinks(root, agentsHome)
	}

	spec := specs[0]
	if err := os.MkdirAll(root, 0755); err != nil {
		return err
	}
	if err := syncPluginOverlayTree(root, pluginFilesDir(spec), pluginPlatformDir(spec, c.ID())); err != nil {
		return err
	}
	if err := syncPluginOverlayTree(filepath.Join(root, "commands"), pluginResourcesDir(spec, "commands")); err != nil {
		return err
	}
	if err := syncPluginOverlayTree(filepath.Join(root, "agents"), pluginResourcesDir(spec, "agents")); err != nil {
		return err
	}
	if err := syncPluginOverlayTree(filepath.Join(root, "skills"), pluginResourcesDir(spec, "skills")); err != nil {
		return err
	}
	if err := syncPluginOverlayTree(filepath.Join(root, "hooks"), pluginResourcesDir(spec, "hooks")); err != nil {
		return err
	}
	if err := claudePluginSyncNamedFile(filepath.Join(root, ".mcp.json"), claudePluginFindSourceFile(spec, ".mcp.json")); err != nil {
		return err
	}
	if err := c.emitPackagePluginManifest(root, spec); err != nil {
		return err
	}
	return c.emitPackagePluginMarketplace(root, spec)
}

func (c *claude) emitPackagePluginManifest(root string, spec PluginSpec) error {
	content, err := renderClaudePackagePluginManifest(spec)
	if err != nil {
		return err
	}
	return writeManagedFile(filepath.Join(root, "plugin.json"), content)
}

func (c *claude) emitPackagePluginMarketplace(root string, spec PluginSpec) error {
	if !shouldRenderPluginMarketplace(spec, c.ID()) {
		return nil
	}
	content, err := renderClaudeMarketplace(spec)
	if err != nil {
		return err
	}
	return writeManagedFile(filepath.Join(root, "marketplace.json"), content)
}

func (c *claude) removePackagePluginLinks(root, agentsHome string) error {
	_ = os.Remove(filepath.Join(root, "plugin.json"))
	_ = os.Remove(filepath.Join(root, "marketplace.json"))
	prefix := filepath.Join(agentsHome, "plugins")
	_ = removeManagedPluginOverlayTree(filepath.Join(root, "commands"), prefix)
	_ = removeManagedPluginOverlayTree(filepath.Join(root, "agents"), prefix)
	_ = removeManagedPluginOverlayTree(filepath.Join(root, "skills"), prefix)
	_ = removeManagedPluginOverlayTree(filepath.Join(root, "hooks"), prefix)
	_ = os.Remove(filepath.Join(root, ".mcp.json"))
	_ = removeManagedPluginOverlayTree(root, prefix)
	_ = pruneEmptyDirsBottomUp(root)
	return nil
}

func (c *claude) RemoveLinks(project, repoPath string) error {
	agentsHome := config.AgentsHome()

	c.removeProjectRuleLinks(project, repoPath, agentsHome)
	c.removeProjectSettingsLink(project, repoPath, agentsHome)
	links.RemoveIfSymlinkUnder(filepath.Join(repoPath, ".mcp.json"), agentsHome)
	c.removeScopedDirLinks(filepath.Join(repoPath, claudeDir, "agents"), agentsHome)
	c.removeScopedDirLinks(filepath.Join(repoPath, claudeDir, "skills"), agentsHome)
	c.removeScopedDirLinks(filepath.Join(repoPath, ".agents", "skills"), agentsHome)
	_ = c.removePackagePluginLinks(filepath.Join(repoPath, ".claude-plugin"), agentsHome)
	return nil
}

func (c *claude) removeProjectRuleLinks(project, repoPath, agentsHome string) {
	rulesDir := filepath.Join(repoPath, claudeDir, "rules")
	if entries, err := os.ReadDir(rulesDir); err == nil {
		prefix := project + "--"
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), prefix) {
				linkPath := filepath.Join(rulesDir, e.Name())
				links.RemoveIfSymlinkUnder(linkPath, agentsHome)
			}
		}
	}
}

func (c *claude) removeProjectSettingsLink(project, repoPath, agentsHome string) {
	projectBundles, err := collectCanonicalHookSpecsForPlatform(agentsHome, project, c.ID(), project)
	if err == nil && len(projectBundles) > 0 {
		_ = removeManagedRenderedHookFile(projectBundles, filepath.Join(repoPath, claudeDir, claudeSettingsLocalJSON), renderClaudeHookSettings)
	} else {
		globalBundles, globalErr := collectCanonicalHookSpecsForPlatform(agentsHome, project, c.ID(), "global")
		if globalErr == nil && len(globalBundles) > 0 {
			_ = removeManagedRenderedHookFile(globalBundles, filepath.Join(repoPath, claudeDir, claudeSettingsLocalJSON), renderClaudeHookSettings)
		}
	}
	links.RemoveIfSymlinkUnder(filepath.Join(repoPath, claudeDir, claudeSettingsLocalJSON), agentsHome)
}

func (c *claude) removeScopedDirLinks(dir, agentsHome string) {
	if entries, err := os.ReadDir(dir); err == nil {
		for _, e := range entries {
			links.RemoveIfSymlinkUnder(filepath.Join(dir, e.Name()), agentsHome)
		}
	}
}

func isClaudeAgentDir(path string) bool {
	if !links.IsDirEntry(path) {
		return false
	}
	_, err := os.Stat(filepath.Join(path, "AGENT.md"))
	return err == nil
}

func isSymlink(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode()&os.ModeSymlink != 0
}

type claudePackagePluginManifest struct {
	Name        string        `json:"name"`
	Version     string        `json:"version,omitempty"`
	Description string        `json:"description,omitempty"`
	Author      *pluginAuthor `json:"author,omitempty"`
	Homepage    string        `json:"homepage,omitempty"`
	Repository  string        `json:"repository,omitempty"`
	License     string        `json:"license,omitempty"`
	Keywords    []string      `json:"keywords,omitempty"`
	Commands    string        `json:"commands,omitempty"`
	Agents      string        `json:"agents,omitempty"`
	Skills      string        `json:"skills,omitempty"`
	Hooks       string        `json:"hooks,omitempty"`
	MCPServers  string        `json:"mcpServers,omitempty"`
}

func renderClaudePackagePluginManifest(spec PluginSpec) ([]byte, error) {
	manifest := claudePackagePluginManifest{
		Name:        spec.Name,
		Version:     spec.Version,
		Description: spec.Description,
		Author:      pluginAuthorFromSpec(spec),
		Homepage:    spec.Homepage,
		Repository:  spec.Marketplace.Repo,
		License:     spec.License,
		Keywords:    append([]string(nil), spec.Marketplace.Tags...),
	}
	if pluginDirHasFiles(pluginResourcesDir(spec, "commands")) || pluginDirHasFiles(filepath.Join(pluginPlatformDir(spec, "claude"), "commands")) {
		manifest.Commands = "./commands/"
	}
	if pluginDirHasFiles(pluginResourcesDir(spec, "agents")) || pluginDirHasFiles(filepath.Join(pluginPlatformDir(spec, "claude"), "agents")) {
		manifest.Agents = "./agents/"
	}
	if pluginDirHasFiles(pluginResourcesDir(spec, "skills")) || pluginDirHasFiles(filepath.Join(pluginPlatformDir(spec, "claude"), "skills")) {
		manifest.Skills = "./skills/"
	}
	if pluginDirHasFiles(pluginResourcesDir(spec, "hooks")) || pluginDirHasFiles(filepath.Join(pluginPlatformDir(spec, "claude"), "hooks")) {
		manifest.Hooks = "./hooks/hooks.json"
	}
	if claudePluginFindSourceFile(spec, ".mcp.json", "mcp.json") != "" {
		manifest.MCPServers = "./.mcp.json"
	}
	return marshalJSON(manifest)
}

func claudePluginFindSourceFile(spec PluginSpec, names ...string) string {
	for _, dir := range []string{
		pluginResourcesDir(spec, "mcp"),
		pluginPlatformDir(spec, "claude"),
	} {
		for _, name := range names {
			src := filepath.Join(dir, name)
			if info, err := os.Stat(src); err == nil && !info.IsDir() {
				return src
			}
		}
	}
	return ""
}

func claudePluginSyncNamedFile(dst string, src string) error {
	if src == "" {
		if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	if _, err := os.Lstat(dst); err == nil {
		if err := os.Remove(dst); err != nil {
			return err
		}
	}
	return os.Symlink(src, dst)
}
