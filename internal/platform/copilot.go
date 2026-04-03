package platform

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/NikashPrakash/dot-agents/internal/config"
	"github.com/NikashPrakash/dot-agents/internal/links"
)

type copilot struct{}

const (
	copilotMCPJSON           = "mcp.json"
	copilotClaudeDir         = ".claude"
	copilotSettingsLocalJSON = "settings.local.json"
	copilotInstructionsMD    = "copilot-instructions.md"
	copilotGitHubDir         = ".github"
	copilotVSCodeDir         = ".vscode"
)

func NewCopilot() Platform { return &copilot{} }

func (c *copilot) ID() string          { return "copilot" }
func (c *copilot) DisplayName() string { return "GitHub Copilot" }

func (c *copilot) IsInstalled() bool {
	home, _ := os.UserHomeDir()
	for _, dir := range []string{
		filepath.Join(home, copilotVSCodeDir, "extensions"),
		filepath.Join(home, ".vscode-insiders", "extensions"),
		filepath.Join(home, ".vscode-server", "extensions"),
	} {
		if entries, err := os.ReadDir(dir); err == nil {
			for _, e := range entries {
				if e.IsDir() && strings.Contains(e.Name(), "copilot") {
					return true
				}
			}
		}
	}
	_, err := exec.LookPath("copilot")
	return err == nil
}

func (c *copilot) Version() string {
	home, _ := os.UserHomeDir()
	for _, dir := range []string{
		filepath.Join(home, copilotVSCodeDir, "extensions"),
		filepath.Join(home, ".vscode-insiders", "extensions"),
	} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() && strings.Contains(name, "copilot") {
				// Name format: publisher.extension-version
				parts := strings.Split(name, "-")
				if len(parts) >= 2 {
					return parts[len(parts)-1] + " (Extension)"
				}
			}
		}
	}
	out, err := exec.Command("copilot", "--version").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(strings.Split(string(out), "\n")[0])
}

func (c *copilot) HasDeprecatedFormat(repoPath string) bool { return false }
func (c *copilot) DeprecatedDetails(repoPath string) string { return "" }

func (c *copilot) CreateLinks(project, repoPath string) error {
	agentsHome := config.AgentsHome()

	// .github/copilot-instructions.md
	if err := c.createInstructionsLink(project, repoPath, agentsHome); err != nil {
		return err
	}

	// .agents/skills/
	if err := c.createSkillsLinks(project, repoPath, agentsHome); err != nil {
		return err
	}

	// .github/agents/{name}.agent.md
	if err := c.createAgentsLinks(project, repoPath, agentsHome); err != nil {
		return err
	}

	// .vscode/mcp.json
	if err := c.createMCPLinks(project, repoPath, agentsHome); err != nil {
		return err
	}

	// .claude/settings.local.json (hooks compat)
	if err := c.createClaudeCompatLinks(project, repoPath, agentsHome); err != nil {
		return err
	}

	// .github/hooks/{name}.json
	if err := c.createProjectHookFiles(project, repoPath, agentsHome); err != nil {
		return err
	}

	if err := c.createPackagePluginLinks(project, repoPath, agentsHome); err != nil {
		return err
	}

	return nil
}

func (c *copilot) resolveInstructionsSrc(project, agentsHome string) string {
	// Priority order per bash implementation
	candidates := []string{
		filepath.Join(agentsHome, "rules", project, copilotInstructionsMD),
		filepath.Join(agentsHome, "rules", "global", copilotInstructionsMD),
	}
	for _, f := range candidates {
		if _, err := os.Stat(f); err == nil {
			return f
		}
	}
	// Fallback: rules.(md|mdc|txt)
	for _, scope := range []string{project, "global"} {
		for _, ext := range []string{"md", "mdc", "txt"} {
			f := filepath.Join(agentsHome, "rules", scope, "rules."+ext)
			if _, err := os.Stat(f); err == nil {
				return f
			}
		}
	}
	return ""
}

func (c *copilot) createInstructionsLink(project, repoPath, agentsHome string) error {
	src := c.resolveInstructionsSrc(project, agentsHome)
	if src == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Join(repoPath, copilotGitHubDir), 0755); err != nil {
		return err
	}
	links.Symlink(src, filepath.Join(repoPath, copilotGitHubDir, copilotInstructionsMD))
	return nil
}

func (c *copilot) createSkillsLinks(project, repoPath, agentsHome string) error {
	return syncScopedDirSymlinksTargets(agentsHome, "skills", project, "SKILL.md", filepath.Join(repoPath, ".agents", "skills"))
}

func (c *copilot) createAgentsLinks(project, repoPath, agentsHome string) error {
	agentsTarget := filepath.Join(repoPath, copilotGitHubDir, "agents")
	if err := os.MkdirAll(agentsTarget, 0755); err != nil {
		return err
	}
	projectAgents := filepath.Join(agentsHome, "agents", project)
	entries, err := os.ReadDir(projectAgents)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		agentDir := filepath.Join(projectAgents, e.Name())
		if !links.IsDirEntry(agentDir) {
			continue
		}
		agentMD := filepath.Join(agentDir, "AGENT.md")
		if _, err := os.Stat(agentMD); err != nil {
			continue
		}
		target := filepath.Join(agentsTarget, e.Name()+".agent.md")
		if _, err := os.Lstat(target); err == nil {
			continue
		}
		links.Symlink(agentMD, target)
	}
	return nil
}

func (c *copilot) createMCPLinks(project, repoPath, agentsHome string) error {
	if src := resolveScopedFile(agentsHome, "mcp", project, "copilot.json", copilotMCPJSON); src != "" {
		if err := os.MkdirAll(filepath.Join(repoPath, copilotVSCodeDir), 0755); err != nil {
			return err
		}
		links.Symlink(src, filepath.Join(repoPath, copilotVSCodeDir, copilotMCPJSON))
	}
	return nil
}

func (c *copilot) createClaudeCompatLinks(project, repoPath, agentsHome string) error {
	target := filepath.Join(repoPath, copilotClaudeDir, copilotSettingsLocalJSON)
	projectBundles, err := collectCanonicalHookSpecsForPlatform(agentsHome, project, c.ID(), project)
	if err != nil {
		return err
	}
	globalBundles, err := collectCanonicalHookSpecsForPlatform(agentsHome, project, c.ID(), "global")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(repoPath, copilotClaudeDir), 0755); err != nil {
		return err
	}
	return emitPreferredHookFile(
		target,
		renderClaudeHookSettings,
		resolveHookSpec(agentsHome, []string{"hooks", "settings"}, project, "claude-code.json"),
		directSymlinkHookMode,
		removeRenderedClaudeHookSettings,
		projectBundles,
		globalBundles,
	)
}

func (c *copilot) createProjectHookFiles(project, repoPath, agentsHome string) error {
	hooksDir := filepath.Join(repoPath, copilotGitHubDir, "hooks")
	canonicalSpecs, err := collectCanonicalHookSpecsForPlatform(agentsHome, project, c.ID(), "global", project)
	if err != nil {
		return err
	}
	if len(canonicalSpecs) > 0 {
		return c.emitCanonicalProjectHookFiles(canonicalSpecs, hooksDir)
	}

	return c.emitLegacyProjectHookFiles(agentsHome, project, hooksDir)
}

func (c *copilot) emitCanonicalProjectHookFiles(specs []HookSpec, hooksDir string) error {
	if err := emitRenderedHookFanout(specs, hooksDir, renderCopilotHookFile); err != nil {
		return err
	}
	wanted, err := renderedCopilotHookNames(specs)
	if err != nil {
		return err
	}
	return pruneManagedRenderedFanoutExtras(hooksDir, wanted, isLikelyRenderedCopilotHookFile)
}

func (c *copilot) emitLegacyProjectHookFiles(agentsHome, project, hooksDir string) error {
	specs, err := listHookSpecs(agentsHome, project)
	if err != nil {
		return pruneManagedRenderedFanoutExtras(hooksDir, map[string]bool{}, isLikelyRenderedCopilotHookFile)
	}
	if err := emitHookFanout(specs, hooksDir, HookEmissionMode{
		Shape:     HookShapeRenderFanout,
		Transport: HookTransportSymlink,
	}, func(spec HookSpec) (string, bool) {
		if spec.Name == "cursor" || spec.Name == "claude-code" {
			return "", false
		}
		return spec.Name + ".json", true
	}); err != nil {
		return err
	}
	wanted := legacyCopilotHookNames(specs)
	return pruneManagedRenderedFanoutExtras(hooksDir, wanted, isLikelyRenderedCopilotHookFile)
}

func renderedCopilotHookNames(specs []HookSpec) (map[string]bool, error) {
	wanted := map[string]bool{}
	for _, spec := range specs {
		name, _, ok, renderErr := renderCopilotHookFile(spec)
		if renderErr != nil {
			return nil, renderErr
		}
		if ok {
			wanted[name] = true
		}
	}
	return wanted, nil
}

func legacyCopilotHookNames(specs []HookSpec) map[string]bool {
	wanted := map[string]bool{}
	for _, spec := range specs {
		if spec.Name == "cursor" || spec.Name == "claude-code" {
			continue
		}
		wanted[spec.Name+".json"] = true
	}
	return wanted
}

func (c *copilot) RemoveLinks(project, repoPath string) error {
	agentsHome := config.AgentsHome()

	c.removeTopLevelLinks(repoPath, agentsHome)
	c.removeClaudeCompatSettings(project, repoPath, agentsHome)
	c.removeSkillsLinks(repoPath, agentsHome)
	c.removeAgentLinks(repoPath, agentsHome)
	c.removeHookLinks(project, repoPath, agentsHome)
	c.removePackagePluginLinks(project, repoPath, agentsHome)
	return nil
}

func (c *copilot) removeTopLevelLinks(repoPath, agentsHome string) {
	links.RemoveIfSymlinkUnder(filepath.Join(repoPath, copilotGitHubDir, copilotInstructionsMD), agentsHome)
	links.RemoveIfSymlinkUnder(filepath.Join(repoPath, copilotVSCodeDir, copilotMCPJSON), agentsHome)
}

func (c *copilot) removeClaudeCompatSettings(project, repoPath, agentsHome string) {
	projectBundles, err := collectCanonicalHookSpecsForPlatform(agentsHome, project, c.ID(), project)
	if err == nil && len(projectBundles) > 0 {
		_ = removeManagedRenderedHookFile(projectBundles, filepath.Join(repoPath, copilotClaudeDir, copilotSettingsLocalJSON), renderClaudeHookSettings)
	} else {
		globalBundles, globalErr := collectCanonicalHookSpecsForPlatform(agentsHome, project, c.ID(), "global")
		if globalErr == nil && len(globalBundles) > 0 {
			_ = removeManagedRenderedHookFile(globalBundles, filepath.Join(repoPath, copilotClaudeDir, copilotSettingsLocalJSON), renderClaudeHookSettings)
		}
	}
	links.RemoveIfSymlinkUnder(filepath.Join(repoPath, copilotClaudeDir, copilotSettingsLocalJSON), agentsHome)
}

func (c *copilot) removeSkillsLinks(repoPath, agentsHome string) {
	skillsDir := filepath.Join(repoPath, ".agents", "skills")
	if entries, err := os.ReadDir(skillsDir); err == nil {
		for _, e := range entries {
			links.RemoveIfSymlinkUnder(filepath.Join(skillsDir, e.Name()), agentsHome)
		}
	}
}

func (c *copilot) removeAgentLinks(repoPath, agentsHome string) {
	agentsDir := filepath.Join(repoPath, copilotGitHubDir, "agents")
	if entries, err := os.ReadDir(agentsDir); err == nil {
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".agent.md") {
				links.RemoveIfSymlinkUnder(filepath.Join(agentsDir, e.Name()), agentsHome)
			}
		}
	}
}

func (c *copilot) removeHookLinks(project, repoPath, agentsHome string) {
	hooksDir := filepath.Join(repoPath, copilotGitHubDir, "hooks")
	canonicalSpecs, err := collectCanonicalHookSpecsForPlatform(agentsHome, project, c.ID(), "global", project)
	if err == nil && len(canonicalSpecs) > 0 {
		_ = removeManagedRenderedHookFanout(canonicalSpecs, hooksDir, renderCopilotHookFile)
	}
	if entries, err := os.ReadDir(hooksDir); err == nil {
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".json") {
				links.RemoveIfSymlinkUnder(filepath.Join(hooksDir, e.Name()), agentsHome)
			}
		}
	}
}

func (c *copilot) createPackagePluginLinks(project, repoPath, agentsHome string) error {
	spec, err := c.selectPackagePluginSpec(agentsHome, project)
	if err != nil {
		return err
	}
	if spec == nil {
		if err := c.removePackagePluginOutputs(repoPath, agentsHome); err != nil {
			return err
		}
		_ = os.Remove(filepath.Join(repoPath, "plugin.json"))
		_ = os.Remove(filepath.Join(repoPath, ".github", "plugin", "marketplace.json"))
		return nil
	}

	if err := c.removePackagePluginOutputs(repoPath, agentsHome); err != nil {
		return err
	}

	if err := c.syncPackagePluginTree(filepath.Join(repoPath, "agents"), filepath.Join(spec.Dir, "resources", "agents")); err != nil {
		return err
	}
	if err := c.syncPackagePluginTree(filepath.Join(repoPath, "skills"), filepath.Join(spec.Dir, "resources", "skills")); err != nil {
		return err
	}
	if err := c.syncPackagePluginTree(filepath.Join(repoPath, "commands"), filepath.Join(spec.Dir, "resources", "commands")); err != nil {
		return err
	}
	if err := c.syncPackagePluginTree(repoPath, filepath.Join(spec.Dir, "files"), filepath.Join(spec.Dir, "platforms", c.ID())); err != nil {
		return err
	}

	manifest, err := c.renderPackagePluginManifest(repoPath, spec)
	if err != nil {
		return err
	}
	if err := writeManagedFile(filepath.Join(repoPath, "plugin.json"), manifest); err != nil {
		return err
	}
	return c.writePackagePluginMarketplace(repoPath, spec)
}

func (c *copilot) removePackagePluginLinks(project, repoPath, agentsHome string) {
	spec, err := c.selectPackagePluginSpec(agentsHome, project)
	if err != nil {
		return
	}
	if spec == nil {
		_ = c.removePackagePluginOutputs(repoPath, agentsHome)
		_ = os.Remove(filepath.Join(repoPath, "plugin.json"))
		_ = os.Remove(filepath.Join(repoPath, ".github", "plugin", "marketplace.json"))
		return
	}

	manifest, err := c.renderPackagePluginManifest(repoPath, spec)
	_ = c.removePackagePluginOutputs(repoPath, agentsHome)
	if err == nil {
		_ = removeManagedFile(filepath.Join(repoPath, "plugin.json"), manifest)
	} else {
		_ = os.Remove(filepath.Join(repoPath, "plugin.json"))
	}
	if marketplace, marketplaceErr := renderCopilotMarketplace(*spec); marketplaceErr == nil {
		_ = removeManagedFile(filepath.Join(repoPath, ".github", "plugin", "marketplace.json"), marketplace)
	} else {
		_ = os.Remove(filepath.Join(repoPath, ".github", "plugin", "marketplace.json"))
	}
}

func (c *copilot) selectPackagePluginSpec(agentsHome, project string) (*PluginSpec, error) {
	projectSpecs, err := c.packagePluginsForScope(agentsHome, project)
	if err != nil {
		return nil, err
	}
	switch len(projectSpecs) {
	case 1:
		return &projectSpecs[0], nil
	case 0:
	default:
		return nil, nil
	}

	globalSpecs, err := c.packagePluginsForScope(agentsHome, "global")
	if err != nil {
		return nil, err
	}
	switch len(globalSpecs) {
	case 1:
		return &globalSpecs[0], nil
	default:
		return nil, nil
	}
}

func (c *copilot) packagePluginsForScope(agentsHome, scope string) ([]PluginSpec, error) {
	specs, err := ListPluginSpecs(agentsHome, scope)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]PluginSpec, 0, len(specs))
	for _, spec := range specs {
		if spec.Kind != PluginKindPackage || !copilotPluginSpecHasPlatform(spec, c.ID()) {
			continue
		}
		out = append(out, spec)
	}
	return out, nil
}

func copilotPluginSpecHasPlatform(spec PluginSpec, platformID string) bool {
	for _, id := range spec.Platforms {
		if id == platformID {
			return true
		}
	}
	return false
}

func (c *copilot) syncPackagePluginTree(dstRoot string, srcRoots ...string) error {
	desired, err := c.collectPackagePluginFiles(srcRoots...)
	if err != nil {
		return err
	}
	if len(desired) == 0 {
		return nil
	}
	if err := os.MkdirAll(dstRoot, 0755); err != nil {
		return err
	}
	rels := make([]string, 0, len(desired))
	for rel := range desired {
		rels = append(rels, rel)
	}
	sort.Strings(rels)
	for _, rel := range rels {
		if err := links.Symlink(desired[rel], filepath.Join(dstRoot, rel)); err != nil {
			return err
		}
	}
	return nil
}

func (c *copilot) collectPackagePluginFiles(srcRoots ...string) (map[string]string, error) {
	out := map[string]string{}
	for _, root := range srcRoots {
		if root == "" {
			continue
		}
		info, err := os.Stat(root)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("plugin source root %s is not a directory", root)
		}
		if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			out[rel] = path
			return nil
		}); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (c *copilot) removePackagePluginOutputs(repoPath, agentsHome string) error {
	if err := c.removeSymlinksUnderPrefix(repoPath, filepath.Join(agentsHome, "plugins")); err != nil {
		return err
	}
	for _, path := range []string{
		filepath.Join(repoPath, "agents"),
		filepath.Join(repoPath, "skills"),
		filepath.Join(repoPath, "commands"),
	} {
		_ = c.removeEmptyDirsBottomUp(path)
	}
	return nil
}

func (c *copilot) removeSymlinksUnderPrefix(root, prefix string) error {
	info, err := os.Stat(root)
	if os.IsNotExist(err) || err != nil || !info.IsDir() {
		return err
	}
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if os.IsNotExist(walkErr) {
				return nil
			}
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if links.IsSymlinkUnder(path, prefix) {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
		return nil
	})
}

func (c *copilot) removeEmptyDirsBottomUp(root string) error {
	info, err := os.Stat(root)
	if os.IsNotExist(err) || err != nil || !info.IsDir() {
		return err
	}
	dirs := []string{}
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if os.IsNotExist(walkErr) {
				return nil
			}
			return walkErr
		}
		if d.IsDir() {
			dirs = append(dirs, path)
		}
		return nil
	}); err != nil {
		return err
	}
	sort.Slice(dirs, func(i, j int) bool {
		return len(dirs[i]) > len(dirs[j])
	})
	for _, dir := range dirs {
		if dir == root {
			continue
		}
		if err := removeDirIfEmpty(dir); err != nil {
			return err
		}
	}
	return removeDirIfEmpty(root)
}

func (c *copilot) renderPackagePluginManifest(repoPath string, spec *PluginSpec) ([]byte, error) {
	manifest := copilotPluginManifest{
		Name:        spec.Name,
		Version:     spec.Version,
		Description: spec.Description,
		Homepage:    spec.Homepage,
		License:     spec.License,
	}
	if len(spec.Authors) > 0 {
		manifest.Author = &copilotPluginAuthor{Name: spec.Authors[0]}
	}
	if spec.Marketplace.Repo != "" {
		manifest.Repository = spec.Marketplace.Repo
	}
	if len(spec.Marketplace.Tags) > 0 {
		manifest.Keywords = append([]string{}, spec.Marketplace.Tags...)
	}
	if hasManagedContent(filepath.Join(repoPath, "agents")) {
		manifest.Agents = "./agents/"
	}
	if hasManagedContent(filepath.Join(repoPath, "skills")) {
		manifest.Skills = "./skills/"
	}
	if hasManagedContent(filepath.Join(repoPath, "commands")) {
		manifest.Commands = "./commands/"
	}
	if fileExists(filepath.Join(repoPath, "hooks.json")) {
		manifest.Hooks = "./hooks.json"
	}
	if fileExists(filepath.Join(repoPath, ".mcp.json")) {
		manifest.MCPServers = "./.mcp.json"
	}
	return marshalJSON(manifest)
}

func (c *copilot) writePackagePluginMarketplace(repoPath string, spec *PluginSpec) error {
	if !shouldRenderPluginMarketplace(*spec, c.ID()) {
		return nil
	}
	content, err := renderCopilotMarketplace(*spec)
	if err != nil {
		return err
	}
	return writeManagedFile(filepath.Join(repoPath, ".github", "plugin", "marketplace.json"), content)
}

type copilotPluginManifest struct {
	Name        string               `json:"name"`
	Version     string               `json:"version,omitempty"`
	Description string               `json:"description,omitempty"`
	Author      *copilotPluginAuthor `json:"author,omitempty"`
	Homepage    string               `json:"homepage,omitempty"`
	Repository  string               `json:"repository,omitempty"`
	License     string               `json:"license,omitempty"`
	Keywords    []string             `json:"keywords,omitempty"`
	Agents      string               `json:"agents,omitempty"`
	Skills      string               `json:"skills,omitempty"`
	Commands    string               `json:"commands,omitempty"`
	Hooks       string               `json:"hooks,omitempty"`
	MCPServers  string               `json:"mcpServers,omitempty"`
}

type copilotPluginAuthor struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
	URL   string `json:"url,omitempty"`
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func hasManagedContent(path string) bool {
	entries, err := os.ReadDir(path)
	return err == nil && len(entries) > 0
}
