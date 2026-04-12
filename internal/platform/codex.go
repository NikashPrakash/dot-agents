package platform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/NikashPrakash/dot-agents/internal/config"
	"github.com/NikashPrakash/dot-agents/internal/links"
)

type codex struct{}

const (
	codexAgentsDir      = ".agents"
	codexDir            = ".codex"
	codexHooksJSON      = "hooks.json"
	codexAgentsMarkdown = "AGENTS.md"
)

func NewCodex() Platform { return &codex{} }

func (c *codex) ID() string          { return "codex" }
func (c *codex) DisplayName() string { return "Codex CLI" }

func (c *codex) IsInstalled() bool {
	_, err := exec.LookPath("codex")
	return err == nil
}

func (c *codex) Version() string {
	out, err := exec.Command("codex", "--version").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(strings.Split(string(out), "\n")[0])
}

func (c *codex) HasDeprecatedFormat(repoPath string) bool { return false }
func (c *codex) DeprecatedDetails(repoPath string) string { return "" }

func (c *codex) CreateLinks(project, repoPath string) error {
	agentsHome := config.AgentsHome()

	if err := c.ensureUserAgents(agentsHome); err != nil {
		return err
	}
	if err := c.ensureUserSkills(agentsHome); err != nil {
		return err
	}

	// AGENTS.md: global then project override
	globalCandidates := []string{
		filepath.Join(agentsHome, "rules", "global", "agents.md"),
		filepath.Join(agentsHome, "rules", "global", "agents.mdc"),
		filepath.Join(agentsHome, "rules", "global", "rules.md"),
		filepath.Join(agentsHome, "rules", "global", "rules.mdc"),
	}
	for _, src := range globalCandidates {
		if _, err := os.Stat(src); err == nil {
			links.Symlink(src, filepath.Join(repoPath, codexAgentsMarkdown))
			break
		}
	}
	// Project override
	for _, name := range []string{"agents.md", "agents.mdc"} {
		src := filepath.Join(agentsHome, "rules", project, name)
		if _, err := os.Stat(src); err == nil {
			links.Symlink(src, filepath.Join(repoPath, codexAgentsMarkdown))
			break
		}
	}

	// .codex/config.toml
	if err := os.MkdirAll(filepath.Join(repoPath, codexDir), 0755); err != nil {
		return err
	}
	if src := resolveScopedFile(agentsHome, "settings", project, "codex.toml"); src != "" {
		links.Symlink(src, filepath.Join(repoPath, codexDir, "config.toml"))
	}

	// Project agents → .codex/agents/*.toml
	if err := c.createAgentsLinks(project, repoPath, agentsHome); err != nil {
		return err
	}

	// Project skills → .agents/skills/
	if err := c.createSkillsLinks(project, repoPath, agentsHome); err != nil {
		return err
	}

	if err := c.createPackagePluginLinks(project, repoPath, agentsHome); err != nil {
		return err
	}

	// Project hooks → .codex/hooks.json
	if err := c.createHooksLinks(project, repoPath, agentsHome); err != nil {
		return err
	}

	return nil
}

func (c *codex) ensureUserAgents(agentsHome string) error {
	globalAgents := filepath.Join(agentsHome, "agents", "global")
	if _, err := os.Stat(globalAgents); err != nil {
		return nil
	}
	for _, homeRoot := range config.UserHomeRoots() {
		userAgentsDir := filepath.Join(homeRoot, codexDir, "agents")
		if err := os.MkdirAll(userAgentsDir, 0755); err != nil {
			continue
		}
		if err := c.writeCodexAgents(agentsHome, "global", userAgentsDir); err != nil {
			return err
		}
	}
	return nil
}

func (c *codex) ensureUserSkills(agentsHome string) error {
	for _, homeRoot := range config.UserHomeRoots() {
		userSkillsDir := filepath.Join(homeRoot, codexAgentsDir, "skills")
		if err := syncScopedDirSymlinks(agentsHome, "skills", "global", "SKILL.md", userSkillsDir); err != nil {
			return err
		}
	}
	return nil
}

func (c *codex) createAgentsLinks(project, repoPath, agentsHome string) error {
	agentsTarget := filepath.Join(repoPath, codexDir, "agents")
	if err := os.MkdirAll(agentsTarget, 0755); err != nil {
		return err
	}
	return c.writeCodexAgents(agentsHome, project, agentsTarget)
}

func (c *codex) createSkillsLinks(project, repoPath, agentsHome string) error {
	return syncScopedDirSymlinksTargets(agentsHome, "skills", project, "SKILL.md", filepath.Join(repoPath, codexAgentsDir, "skills"))
}

func (c *codex) createPackagePluginLinks(project, repoPath, agentsHome string) error {
	spec, _, err := selectedPackagePluginForPlatform(agentsHome, project, c.ID())
	if err != nil {
		return err
	}
	if spec == nil {
		return c.removeManagedPackagePlugin(repoPath, agentsHome)
	}

	skillsSrc := pluginResourcesDir(*spec, "skills")
	rootSources := existingPluginSourceRoots(pluginFilesDir(*spec), pluginPlatformDir(*spec, c.ID()))
	if len(rootSources) == 0 {
		if err := c.removeManagedPackagePlugin(repoPath, agentsHome); err != nil {
			return err
		}
	}
	if _, err := os.Stat(skillsSrc); err == nil {
		if err := syncPluginOverlayTree(filepath.Join(repoPath, "skills"), skillsSrc); err != nil {
			return err
		}
	} else {
		if err := removeManagedPluginOverlayTree(filepath.Join(repoPath, "skills"), agentsHome); err != nil {
			return err
		}
	}
	if len(rootSources) > 0 {
		if err := syncCodexPackageRootTree(repoPath, agentsHome, rootSources...); err != nil {
			return err
		}
	}
	if err := c.writeCodexPluginManifest(repoPath, *spec); err != nil {
		return err
	}
	return c.writeCodexPluginMarketplace(repoPath, *spec)
}

func (c *codex) removeManagedPackagePlugin(repoPath, agentsHome string) error {
	if err := removeManagedPluginOverlayTree(repoPath, filepath.Join(agentsHome, "plugins")); err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(repoPath, ".codex-plugin", "plugin.json")); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Remove(filepath.Join(repoPath, ".agents", "plugins", "marketplace.json")); err != nil && !os.IsNotExist(err) {
		return err
	}
	_ = removeDirIfEmpty(filepath.Join(repoPath, ".agents", "plugins"))
	_ = removeDirIfEmpty(filepath.Join(repoPath, ".agents"))
	return removeDirIfEmpty(filepath.Join(repoPath, ".codex-plugin"))
}

type codexPluginManifest struct {
	Name        string                `json:"name"`
	Version     string                `json:"version,omitempty"`
	Description string                `json:"description,omitempty"`
	Repository  string                `json:"repository,omitempty"`
	License     string                `json:"license,omitempty"`
	Keywords    []string              `json:"keywords,omitempty"`
	Skills      string                `json:"skills,omitempty"`
	Hooks       string                `json:"hooks,omitempty"`
	MCPServers  string                `json:"mcpServers,omitempty"`
	Apps        string                `json:"apps,omitempty"`
	Interface   *codexPluginInterface `json:"interface,omitempty"`
}

type codexPluginInterface struct {
	DisplayName      string `json:"displayName,omitempty"`
	ShortDescription string `json:"shortDescription,omitempty"`
	LongDescription  string `json:"longDescription,omitempty"`
	DeveloperName    string `json:"developerName,omitempty"`
}

func (c *codex) writeCodexPluginManifest(repoPath string, spec PluginSpec) error {
	manifest := codexPluginManifest{
		Name:        spec.Name,
		Version:     spec.Version,
		Description: spec.Description,
		Repository:  spec.Marketplace.Repo,
		License:     spec.License,
		Keywords:    append([]string(nil), spec.Marketplace.Tags...),
	}
	if display := strings.TrimSpace(spec.DisplayName); display != "" || strings.TrimSpace(spec.Description) != "" {
		if display == "" {
			display = spec.Name
		}
		manifest.Interface = &codexPluginInterface{
			DisplayName:      display,
			ShortDescription: spec.Description,
			LongDescription:  spec.Description,
		}
	}
	if len(spec.Authors) > 0 {
		manifest.Interface = ensureCodexPluginInterface(manifest.Interface)
		manifest.Interface.DeveloperName = strings.Join(spec.Authors, ", ")
	}

	if pathExists(filepath.Join(repoPath, "skills")) {
		manifest.Skills = "./skills/"
	}
	if pathExists(filepath.Join(repoPath, "hooks.json")) {
		manifest.Hooks = "./hooks.json"
	}
	if pathExists(filepath.Join(repoPath, ".mcp.json")) {
		manifest.MCPServers = "./.mcp.json"
	}
	if pathExists(filepath.Join(repoPath, ".app.json")) {
		manifest.Apps = "./.app.json"
	}

	data, err := marshalJSON(manifest)
	if err != nil {
		return err
	}
	manifestPath := filepath.Join(repoPath, ".codex-plugin", "plugin.json")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0755); err != nil {
		return err
	}
	return writeManagedFile(manifestPath, data)
}

func (c *codex) writeCodexPluginMarketplace(repoPath string, spec PluginSpec) error {
	if !shouldRenderPluginMarketplace(spec, c.ID()) {
		return nil
	}
	content, err := renderCodexMarketplace(spec)
	if err != nil {
		return err
	}
	return writeManagedFile(filepath.Join(repoPath, ".agents", "plugins", "marketplace.json"), content)
}

func syncCodexPackageRootTree(repoPath, agentsHome string, srcRoots ...string) error {
	desired, err := collectPluginOverlayFiles(srcRoots...)
	if err != nil {
		return err
	}
	if len(desired) == 0 {
		return nil
	}
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		return err
	}

	rels := make([]string, 0, len(desired))
	for rel := range desired {
		rels = append(rels, rel)
	}
	sort.Strings(rels)
	for _, rel := range rels {
		if err := links.Symlink(desired[rel], filepath.Join(repoPath, rel)); err != nil {
			return err
		}
	}
	return pruneCodexPackageRootTree(repoPath, agentsHome, desired)
}

func pruneCodexPackageRootTree(repoPath, agentsHome string, desired map[string]string) error {
	info, err := os.Stat(repoPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return nil
	}

	skip := map[string]bool{
		"skills":        true,
		".codex-plugin": true,
	}
	prefix := filepath.Join(agentsHome, "plugins")
	if err := filepath.WalkDir(repoPath, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			if os.IsNotExist(walkErr) {
				return nil
			}
			return walkErr
		}
		if path == repoPath {
			return nil
		}

		rel, err := filepath.Rel(repoPath, path)
		if err != nil {
			return err
		}
		first := rel
		if idx := strings.IndexRune(rel, filepath.Separator); idx >= 0 {
			first = rel[:idx]
		}
		if d.IsDir() {
			if skip[first] {
				return filepath.SkipDir
			}
			return nil
		}

		if want, ok := desired[rel]; ok && links.IsSymlinkTo(path, want) {
			return nil
		}
		if links.IsSymlinkUnder(path, prefix) {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	return pruneEmptyDirsBottomUp(repoPath)
}

func ensureCodexPluginInterface(iface *codexPluginInterface) *codexPluginInterface {
	if iface != nil {
		return iface
	}
	return &codexPluginInterface{}
}

func pathExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func (c *codex) createHooksLinks(project, repoPath, agentsHome string) error {
	if err := c.writeRepoHooks(project, repoPath, agentsHome); err != nil {
		return err
	}
	return c.writeUserHomeHooks(project, agentsHome)
}

func (c *codex) writeRepoHooks(project, repoPath, agentsHome string) error {
	repoTarget := filepath.Join(repoPath, codexDir, codexHooksJSON)
	repoBundles, err := collectCanonicalHookSpecsForPlatform(agentsHome, project, c.ID(), "global", project)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(repoPath, codexDir), 0755); err != nil {
		return err
	}
	return emitPreferredHookFile(
		repoTarget,
		renderCodexHookConfig,
		resolveHookSpec(agentsHome, []string{"hooks"}, project, "codex.json", "codex-hooks.json"),
		directSymlinkHookMode,
		removeRenderedCodexHookConfig,
		repoBundles,
	)
}

func (c *codex) writeUserHomeHooks(project, agentsHome string) error {
	globalBundles, err := collectCanonicalHookSpecsForPlatform(agentsHome, project, c.ID(), "global")
	if err != nil {
		return err
	}
	return emitPreferredHookFileToUserHomes(
		filepath.Join(codexDir, codexHooksJSON),
		renderCodexHookConfig,
		resolveHookSpec(agentsHome, []string{"hooks"}, project, "codex.json", "codex-hooks.json"),
		directSymlinkHookMode,
		removeRenderedCodexHookConfig,
		globalBundles,
	)
}

func (c *codex) RemoveLinks(project, repoPath string) error {
	agentsHome := config.AgentsHome()

	links.RemoveIfSymlinkUnder(filepath.Join(repoPath, codexAgentsMarkdown), agentsHome)
	links.RemoveIfSymlinkUnder(filepath.Join(repoPath, codexDir, "config.toml"), agentsHome)
	repoBundles, err := collectCanonicalHookSpecsForPlatform(agentsHome, project, c.ID(), "global", project)
	if err == nil && len(repoBundles) > 0 {
		_ = removeManagedRenderedHookFile(repoBundles, filepath.Join(repoPath, codexDir, codexHooksJSON), renderCodexHookConfig)
	}
	links.RemoveIfSymlinkUnder(filepath.Join(repoPath, codexDir, codexHooksJSON), agentsHome)

	_ = c.pruneManagedCodexAgentTomls(agentsHome, project, filepath.Join(repoPath, codexDir, "agents"))

	skillsDir := filepath.Join(repoPath, codexAgentsDir, "skills")
	if entries, err := os.ReadDir(skillsDir); err == nil {
		for _, e := range entries {
			links.RemoveIfSymlinkUnder(filepath.Join(skillsDir, e.Name()), agentsHome)
		}
	}

	_ = c.removeManagedPackagePlugin(repoPath, agentsHome)

	return nil
}

func (c *codex) writeCodexAgents(agentsHome, scope, dstRoot string) error {
	entries, err := listScopedResourceDirs(agentsHome, "agents", scope, "AGENT.md")
	if err != nil {
		return nil
	}
	wanted := map[string]bool{}
	for _, entry := range entries {
		wanted[entry.Name+".toml"] = true
		dst := filepath.Join(dstRoot, entry.Name+".toml")
		if err := c.writeCodexAgentToml(dst, entry.File); err != nil {
			return err
		}
	}
	if existing, err := os.ReadDir(dstRoot); err == nil {
		for _, e := range existing {
			if !strings.HasSuffix(e.Name(), ".toml") || wanted[e.Name()] {
				continue
			}
			_ = os.Remove(filepath.Join(dstRoot, e.Name()))
		}
	}
	return nil
}

func (c *codex) pruneManagedCodexAgentTomls(agentsHome, scope, dstRoot string) error {
	entries, err := listScopedResourceDirs(agentsHome, "agents", scope, "AGENT.md")
	if err != nil {
		return nil
	}
	for _, entry := range entries {
		if err := os.Remove(filepath.Join(dstRoot, entry.Name+".toml")); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (c *codex) writeCodexAgentToml(dst, agentMD string) error {
	content, err := renderCodexAgentToml(agentMD)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	if _, err := os.Lstat(dst); err == nil {
		if err := os.Remove(dst); err != nil {
			return err
		}
	}
	return os.WriteFile(dst, content, 0644)
}

func renderCodexAgentToml(agentMD string) ([]byte, error) {
	meta := readFrontmatter(agentMD)
	body, err := readAgentBody(agentMD)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(meta["name"])
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(filepath.Dir(agentMD)), string(filepath.Ext(agentMD)))
	}
	description := strings.TrimSpace(meta["description"])
	model := strings.TrimSpace(meta["model"])
	var b strings.Builder
	fmt.Fprintf(&b, "name = %s\n", strconv.Quote(name))
	fmt.Fprintf(&b, "description = %s\n", strconv.Quote(description))
	if model != "" {
		fmt.Fprintf(&b, "model = %s\n", strconv.Quote(model))
	}
	if strings.TrimSpace(body) != "" {
		b.WriteString("developer_instructions = ")
		b.WriteString(tomlMultilineString(body))
		b.WriteString("\n")
	}
	return []byte(b.String()), nil
}

func readAgentBody(agentMD string) (string, error) {
	data, err := os.ReadFile(agentMD)
	if err != nil {
		return "", err
	}
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	if !strings.HasPrefix(text, "---\n") {
		return text, nil
	}
	rest := strings.TrimPrefix(text, "---\n")
	end := strings.Index(rest, "\n---\n")
	if end == -1 {
		return text, nil
	}
	body := rest[end+len("\n---\n"):]
	body = strings.TrimLeft(body, "\n")
	return body, nil
}

func tomlMultilineString(value string) string {
	escaped := strings.ReplaceAll(value, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"""`, `\"\"\"`)
	return "\"\"\"\n" + escaped + "\n\"\"\""
}
