package platform

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/NikashPrakash/dot-agents/internal/config"
	"github.com/NikashPrakash/dot-agents/internal/links"
)

type cursor struct{}

const (
	cursorHooksFile   = "hooks.json"
	cursorJSON        = "cursor.json"
	cursorDir         = ".cursor"
	globalRulesPrefix = "global--"
)

func NewCursor() Platform { return &cursor{} }

func (c *cursor) ID() string          { return "cursor" }
func (c *cursor) DisplayName() string { return "Cursor" }

func (c *cursor) IsInstalled() bool {
	if _, err := os.Stat("/Applications/Cursor.app"); err == nil {
		return true
	}
	_, err := exec.LookPath("cursor")
	return err == nil
}

func (c *cursor) Version() string {
	// Try app version on macOS
	if _, err := os.Stat("/Applications/Cursor.app"); err == nil {
		out, err := exec.Command("defaults", "read",
			"/Applications/Cursor.app/Contents/Info.plist",
			"CFBundleShortVersionString").Output()
		if err == nil {
			appVer := strings.TrimSpace(string(out))
			if path, err := exec.LookPath("cursor"); err == nil {
				cliOut, err := exec.Command(path, "--version").Output()
				if err == nil {
					cliVer := strings.TrimSpace(strings.Split(string(cliOut), "\n")[0])
					return appVer + " (CLI: " + cliVer + ")"
				}
			}
			return appVer + " (App)"
		}
	}
	if path, err := exec.LookPath("cursor"); err == nil {
		out, err := exec.Command(path, "--version").Output()
		if err == nil {
			return strings.TrimSpace(strings.Split(string(out), "\n")[0])
		}
	}
	return ""
}

func (c *cursor) HasDeprecatedFormat(repoPath string) bool {
	_, err := os.Stat(filepath.Join(repoPath, ".cursorrules"))
	return err == nil
}

func (c *cursor) DeprecatedDetails(repoPath string) string {
	if c.HasDeprecatedFormat(repoPath) {
		return ".cursorrules → .cursor/rules/*.mdc"
	}
	return ""
}

func (c *cursor) CreateLinks(project, repoPath string) error {
	agentsHome := config.AgentsHome()

	if err := c.createRuleLinks(project, repoPath, agentsHome); err != nil {
		return err
	}
	if err := c.createSettingsLinks(project, repoPath, agentsHome); err != nil {
		return err
	}
	if err := c.createMCPLinks(project, repoPath, agentsHome); err != nil {
		return err
	}
	if err := c.createIgnoreLink(project, repoPath, agentsHome); err != nil {
		return err
	}
	if err := c.createAgentsLinks(project, repoPath, agentsHome); err != nil {
		return err
	}
	if err := c.createHooksLinks(project, repoPath, agentsHome); err != nil {
		return err
	}
	return c.createPackagePluginLinks(project, repoPath, agentsHome)
}

func (c *cursor) createRuleLinks(project, repoPath, agentsHome string) error {
	rulesDir := filepath.Join(repoPath, cursorDir, "rules")
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		return err
	}

	c.linkRuleDir(filepath.Join(agentsHome, "rules", "global"), rulesDir, globalRulesPrefix)
	c.linkRuleDir(filepath.Join(agentsHome, "rules", project), rulesDir, project+"--")
	return nil
}

func (c *cursor) linkRuleDir(sourceDir, rulesDir, prefix string) {
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		c.linkRuleEntry(entry, sourceDir, rulesDir, prefix)
	}
}

func (c *cursor) linkRuleEntry(entry os.DirEntry, sourceDir, rulesDir, prefix string) {
	if entry.IsDir() {
		return
	}
	name := entry.Name()
	if !isCursorRuleFile(name) {
		return
	}
	links.Hardlink(
		filepath.Join(sourceDir, name),
		filepath.Join(rulesDir, prefix+toMDC(name)),
	) // best-effort
}

func (c *cursor) createSettingsLinks(project, repoPath, agentsHome string) error {
	if err := os.MkdirAll(filepath.Join(repoPath, cursorDir), 0755); err != nil {
		return err
	}
	if src := resolveScopedFile(agentsHome, "settings", project, cursorJSON); src != "" {
		dst := filepath.Join(repoPath, cursorDir, "settings.json")
		links.Hardlink(src, dst) // best-effort
	}
	return nil
}

func (c *cursor) createMCPLinks(project, repoPath, agentsHome string) error {
	if err := os.MkdirAll(filepath.Join(repoPath, cursorDir), 0755); err != nil {
		return err
	}
	if src := resolveScopedFile(agentsHome, "mcp", project, cursorJSON, "mcp.json"); src != "" {
		dst := filepath.Join(repoPath, cursorDir, "mcp.json")
		links.Hardlink(src, dst) // best-effort
	}
	return nil
}

func (c *cursor) createIgnoreLink(project, repoPath, agentsHome string) error {
	if src := resolveScopedFile(agentsHome, "settings", project, "cursorignore"); src != "" {
		dst := filepath.Join(repoPath, ".cursorignore")
		links.Hardlink(src, dst) // best-effort
	}
	return nil
}

func (c *cursor) createHooksLinks(project, repoPath, agentsHome string) error {
	if err := c.writeRepoHooks(project, repoPath, agentsHome); err != nil {
		return err
	}
	return c.writeUserHomeHooks(project, agentsHome)
}

func (c *cursor) createPackagePluginLinks(project, repoPath, agentsHome string) error {
	specs, _, err := preferredPackagePluginsForPlatform(agentsHome, project, c.ID())
	if err != nil {
		return err
	}

	root := filepath.Join(repoPath, ".cursor-plugin")
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
	if err := syncPluginOverlayTree(filepath.Join(root, "rules"), pluginResourcesDir(spec, "rules")); err != nil {
		return err
	}
	if err := syncPluginOverlayTree(filepath.Join(root, "agents"), pluginResourcesDir(spec, "agents")); err != nil {
		return err
	}
	if err := syncPluginOverlayTree(filepath.Join(root, "skills"), pluginResourcesDir(spec, "skills")); err != nil {
		return err
	}
	if err := syncPluginOverlayTree(filepath.Join(root, "commands"), pluginResourcesDir(spec, "commands")); err != nil {
		return err
	}
	if err := syncPluginOverlayTree(filepath.Join(root, "hooks"), pluginResourcesDir(spec, "hooks")); err != nil {
		return err
	}
	if err := cursorPluginSyncNamedFile(filepath.Join(root, "mcp.json"), cursorPluginFindSourceFile(spec, "mcp.json")); err != nil {
		return err
	}
	if err := c.emitPackagePluginManifest(root, spec); err != nil {
		return err
	}
	return c.emitPackagePluginMarketplace(root, spec)
}

func (c *cursor) emitPackagePluginManifest(root string, spec PluginSpec) error {
	content, err := renderCursorPackagePluginManifest(spec)
	if err != nil {
		return err
	}
	return writeManagedFile(filepath.Join(root, "plugin.json"), content)
}

func (c *cursor) emitPackagePluginMarketplace(root string, spec PluginSpec) error {
	if !shouldRenderPluginMarketplace(spec, c.ID()) {
		return nil
	}
	content, err := renderCursorMarketplace(spec)
	if err != nil {
		return err
	}
	return writeManagedFile(filepath.Join(root, "marketplace.json"), content)
}

func (c *cursor) writeRepoHooks(project, repoPath, agentsHome string) error {
	repoTarget := filepath.Join(repoPath, cursorDir, cursorHooksFile)
	repoBundles, err := collectCanonicalHookSpecsForPlatform(agentsHome, project, c.ID(), "global", project)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(repoPath, cursorDir), 0755); err != nil {
		return err
	}
	return emitPreferredHookFile(
		repoTarget,
		renderCursorHookConfig,
		resolveHookSpec(agentsHome, []string{"hooks"}, project, cursorJSON),
		directHardlinkHookMode,
		removeRenderedCursorHookConfig,
		repoBundles,
	)
}

func (c *cursor) writeUserHomeHooks(project, agentsHome string) error {
	globalBundles, err := collectCanonicalHookSpecsForPlatform(agentsHome, project, c.ID(), "global")
	if err != nil {
		return err
	}
	return emitPreferredHookFileToUserHomes(
		filepath.Join(cursorDir, cursorHooksFile),
		renderCursorHookConfig,
		resolveHookSpecInScope(agentsHome, []string{"hooks"}, "global", cursorJSON),
		directHardlinkHookMode,
		removeRenderedCursorHookConfig,
		globalBundles,
	)
}

func (c *cursor) createAgentsLinks(project, repoPath, agentsHome string) error {
	return syncScopedDirSymlinksTargets(agentsHome, "agents", project, "AGENT.md", filepath.Join(repoPath, ".claude", "agents"))
}

func (c *cursor) RemoveLinks(project, repoPath string) error {
	agentsHome := config.AgentsHome()
	c.removeRuleLinks(project, repoPath, agentsHome)
	c.removeHooksLink(project, repoPath, agentsHome)
	c.removeAgentLinks(repoPath, agentsHome)
	_ = c.removePackagePluginLinks(filepath.Join(repoPath, ".cursor-plugin"), agentsHome)

	return nil
}

func (c *cursor) removeRuleLinks(project, repoPath, agentsHome string) {
	rulesDir := filepath.Join(repoPath, cursorDir, "rules")
	entries, err := os.ReadDir(rulesDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		c.removeRuleEntry(entry, rulesDir, project, agentsHome)
	}
}

func (c *cursor) removeRuleEntry(entry os.DirEntry, rulesDir, project, agentsHome string) {
	if entry.IsDir() {
		return
	}
	name := entry.Name()
	filePath := filepath.Join(rulesDir, name)

	switch {
	case strings.HasPrefix(name, globalRulesPrefix):
		removeHardlinkIfLinkedToAny(filePath, cursorRuleSources(agentsHome, "global", strings.TrimPrefix(name, globalRulesPrefix)))
	case strings.HasPrefix(name, project+"--"):
		removeHardlinkIfLinkedToAny(filePath, cursorRuleSources(agentsHome, project, strings.TrimPrefix(name, project+"--")))
	}
}

func (c *cursor) removeHooksLink(project, repoPath, agentsHome string) {
	hooksFilePath := filepath.Join(repoPath, cursorDir, cursorHooksFile)
	repoBundles, err := collectCanonicalHookSpecsForPlatform(agentsHome, project, c.ID(), "global", project)
	if err == nil && len(repoBundles) > 0 {
		_ = removeManagedRenderedHookFile(repoBundles, hooksFilePath, renderCursorHookConfig)
	}
	removeHardlinkIfLinkedToAny(hooksFilePath, []string{
		filepath.Join(agentsHome, "hooks", project, cursorJSON),
		filepath.Join(agentsHome, "hooks", "global", cursorJSON),
	})
}

func (c *cursor) removeAgentLinks(repoPath, agentsHome string) {
	agentsTarget := filepath.Join(repoPath, cursorDir, "agents")
	entries, err := os.ReadDir(agentsTarget)
	if err != nil {
		return
	}
	for _, entry := range entries {
		links.RemoveIfSymlinkUnder(filepath.Join(agentsTarget, entry.Name()), agentsHome)
	}
}

// toMDC converts .md extension to .mdc; leaves .mdc unchanged.
func toMDC(name string) string {
	if strings.HasSuffix(name, ".md") && !strings.HasSuffix(name, ".mdc") {
		return strings.TrimSuffix(name, ".md") + ".mdc"
	}
	return name
}

func isCursorRuleFile(name string) bool {
	return strings.HasSuffix(name, ".mdc") || strings.HasSuffix(name, ".md")
}

func cursorRuleSources(agentsHome, scope, name string) []string {
	return []string{
		filepath.Join(agentsHome, "rules", scope, name),
		filepath.Join(agentsHome, "rules", scope, strings.TrimSuffix(name, ".mdc")+".md"),
	}
}

func removeHardlinkIfLinkedToAny(path string, sources []string) bool {
	for _, src := range sources {
		if linked, _ := links.AreHardlinked(path, src); linked {
			_ = os.Remove(path)
			return true
		}
	}
	return false
}

type cursorPackagePluginManifest struct {
	Name        string        `json:"name"`
	Version     string        `json:"version,omitempty"`
	Description string        `json:"description,omitempty"`
	Author      *pluginAuthor `json:"author,omitempty"`
	Homepage    string        `json:"homepage,omitempty"`
	Repository  string        `json:"repository,omitempty"`
	License     string        `json:"license,omitempty"`
	Keywords    []string      `json:"keywords,omitempty"`
	Rules       string        `json:"rules,omitempty"`
	Commands    string        `json:"commands,omitempty"`
	Agents      string        `json:"agents,omitempty"`
	Skills      string        `json:"skills,omitempty"`
	Hooks       string        `json:"hooks,omitempty"`
	MCPServers  string        `json:"mcpServers,omitempty"`
}

func renderCursorPackagePluginManifest(spec PluginSpec) ([]byte, error) {
	manifest := cursorPackagePluginManifest{
		Name:        spec.Name,
		Version:     spec.Version,
		Description: spec.Description,
		Author:      pluginAuthorFromSpec(spec),
		Homepage:    spec.Homepage,
		Repository:  spec.Marketplace.Repo,
		License:     spec.License,
		Keywords:    append([]string(nil), spec.Marketplace.Tags...),
	}
	if pluginDirHasFiles(pluginResourcesDir(spec, "rules")) || pluginDirHasFiles(filepath.Join(pluginPlatformDir(spec, "cursor"), "rules")) {
		manifest.Rules = "./rules/"
	}
	if pluginDirHasFiles(pluginResourcesDir(spec, "commands")) || pluginDirHasFiles(filepath.Join(pluginPlatformDir(spec, "cursor"), "commands")) {
		manifest.Commands = "./commands/"
	}
	if pluginDirHasFiles(pluginResourcesDir(spec, "agents")) || pluginDirHasFiles(filepath.Join(pluginPlatformDir(spec, "cursor"), "agents")) {
		manifest.Agents = "./agents/"
	}
	if pluginDirHasFiles(pluginResourcesDir(spec, "skills")) || pluginDirHasFiles(filepath.Join(pluginPlatformDir(spec, "cursor"), "skills")) {
		manifest.Skills = "./skills/"
	}
	if pluginDirHasFiles(pluginResourcesDir(spec, "hooks")) || pluginDirHasFiles(filepath.Join(pluginPlatformDir(spec, "cursor"), "hooks")) {
		manifest.Hooks = "./hooks/hooks.json"
	}
	if cursorPluginFindSourceFile(spec, "mcp.json", ".mcp.json") != "" {
		manifest.MCPServers = "./mcp.json"
	}
	return marshalJSON(manifest)
}

func (c *cursor) removePackagePluginLinks(root, agentsHome string) error {
	_ = os.Remove(filepath.Join(root, "plugin.json"))
	_ = os.Remove(filepath.Join(root, "marketplace.json"))
	prefix := filepath.Join(agentsHome, "plugins")
	_ = removeManagedPluginOverlayTree(filepath.Join(root, "rules"), prefix)
	_ = removeManagedPluginOverlayTree(filepath.Join(root, "agents"), prefix)
	_ = removeManagedPluginOverlayTree(filepath.Join(root, "skills"), prefix)
	_ = removeManagedPluginOverlayTree(filepath.Join(root, "commands"), prefix)
	_ = removeManagedPluginOverlayTree(filepath.Join(root, "hooks"), prefix)
	_ = os.Remove(filepath.Join(root, "mcp.json"))
	_ = removeManagedPluginOverlayTree(root, prefix)
	_ = pruneEmptyDirsBottomUp(root)
	return nil
}

func cursorPluginFindSourceFile(spec PluginSpec, names ...string) string {
	for _, dir := range []string{
		pluginResourcesDir(spec, "mcp"),
		pluginPlatformDir(spec, "cursor"),
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

func cursorPluginSyncNamedFile(dst string, src string) error {
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
