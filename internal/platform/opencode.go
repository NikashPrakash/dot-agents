package platform

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/NikashPrakash/dot-agents/internal/config"
	"github.com/NikashPrakash/dot-agents/internal/links"
)

type opencode struct{}

const opencodeJSON = "opencode.json"
const (
	opencodeProjectPluginsDir = ".opencode/plugins"
	opencodeGlobalPluginsDir  = ".config/opencode/plugins"
)

func NewOpenCode() Platform { return &opencode{} }

func (o *opencode) ID() string          { return "opencode" }
func (o *opencode) DisplayName() string { return "OpenCode" }

func (o *opencode) IsInstalled() bool {
	_, err := exec.LookPath("opencode")
	return err == nil
}

func (o *opencode) Version() string {
	out, err := exec.Command("opencode", "--version").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(strings.Split(string(out), "\n")[0])
}

func (o *opencode) HasDeprecatedFormat(repoPath string) bool { return false }
func (o *opencode) DeprecatedDetails(repoPath string) string { return "" }

func (o *opencode) CreateLinks(project, repoPath string) error {
	agentsHome := config.AgentsHome()

	if err := o.ensureUserAgents(agentsHome); err != nil {
		return err
	}

	// opencode.json config
	if src := resolveScopedFile(agentsHome, "settings", project, opencodeJSON); src != "" {
		links.Symlink(src, filepath.Join(repoPath, opencodeJSON))
	}

	// .opencode/agent/ definitions from canonical agents/{scope}/{name}/AGENT.md
	agentDir := filepath.Join(repoPath, ".opencode", "agent")
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		return err
	}

	if err := syncScopedFileSymlinks(agentsHome, "agents", project, "AGENT.md", agentDir, ".md"); err != nil {
		return err
	}

	// Project skills → .agents/skills/
	if err := o.createSkillsLinks(project, repoPath, agentsHome); err != nil {
		return err
	}
	if err := o.createPluginLinks(project, repoPath, agentsHome); err != nil {
		return err
	}

	return nil
}

func (o *opencode) ensureUserAgents(agentsHome string) error {
	for _, homeRoot := range config.UserHomeRoots() {
		userAgentsDir := filepath.Join(homeRoot, ".opencode", "agent")
		if err := syncScopedFileSymlinks(agentsHome, "agents", "global", "AGENT.md", userAgentsDir, ".md"); err != nil {
			return err
		}
	}
	return nil
}

func (o *opencode) createSkillsLinks(project, repoPath, agentsHome string) error {
	return syncScopedDirSymlinksTargets(agentsHome, "skills", project, "SKILL.md", filepath.Join(repoPath, ".agents", "skills"))
}

func (o *opencode) createPluginLinks(project, repoPath, agentsHome string) error {
	projectPlugins, err := o.loadNativePlugins(agentsHome, project)
	if err != nil {
		return err
	}
	if err := o.emitPlugins(projectPlugins, filepath.Join(repoPath, opencodeProjectPluginsDir)); err != nil {
		return err
	}

	globalPlugins, err := o.loadNativePlugins(agentsHome, "global")
	if err != nil {
		return err
	}
	for _, homeRoot := range config.UserHomeRoots() {
		dstRoot := filepath.Join(homeRoot, opencodeGlobalPluginsDir)
		if err := o.emitPlugins(globalPlugins, dstRoot); err != nil {
			return err
		}
	}
	return nil
}

func (o *opencode) loadNativePlugins(agentsHome, scope string) ([]PluginSpec, error) {
	if _, err := os.Stat(filepath.Join(agentsHome, "plugins", scope)); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	specs, err := ListPluginSpecs(agentsHome, scope)
	if err != nil {
		return nil, err
	}

	filtered := make([]PluginSpec, 0, len(specs))
	for _, spec := range specs {
		if spec.Kind != PluginKindNative {
			continue
		}
		if !pluginSpecHasPlatform(spec, o.ID()) {
			continue
		}
		filtered = append(filtered, spec)
	}
	return filtered, nil
}

func (o *opencode) emitPlugins(specs []PluginSpec, dstRoot string) error {
	if len(specs) == 0 {
		return nil
	}
	if err := os.MkdirAll(dstRoot, 0755); err != nil {
		return err
	}
	for _, spec := range specs {
		if err := o.emitPlugin(spec, dstRoot); err != nil {
			return err
		}
	}
	return nil
}

func (o *opencode) emitPlugin(spec PluginSpec, dstRoot string) error {
	pluginRoot := filepath.Join(dstRoot, spec.Name)
	if err := os.MkdirAll(pluginRoot, 0755); err != nil {
		return err
	}
	if err := syncPluginTree(filepath.Join(spec.Dir, "files"), pluginRoot); err != nil {
		return err
	}
	if err := syncPluginTree(filepath.Join(spec.Dir, "platforms", o.ID()), pluginRoot); err != nil {
		return err
	}
	return nil
}

func syncPluginTree(srcRoot, dstRoot string) error {
	entries, err := os.ReadDir(srcRoot)
	if err != nil {
		return nil
	}
	for _, entry := range entries {
		srcPath := filepath.Join(srcRoot, entry.Name())
		dstPath := filepath.Join(dstRoot, entry.Name())
		if entry.IsDir() {
			if err := syncPluginTree(srcPath, dstPath); err != nil {
				return err
			}
			continue
		}
		if err := links.Symlink(srcPath, dstPath); err != nil {
			return err
		}
	}
	return nil
}

func (o *opencode) RemoveLinks(project, repoPath string) error {
	agentsHome := config.AgentsHome()

	links.RemoveIfSymlinkUnder(filepath.Join(repoPath, opencodeJSON), agentsHome)

	agentDir := filepath.Join(repoPath, ".opencode", "agent")
	if entries, err := os.ReadDir(agentDir); err == nil {
		for _, e := range entries {
			links.RemoveIfSymlinkUnder(filepath.Join(agentDir, e.Name()), agentsHome)
		}
	}

	_ = removePluginLinks(filepath.Join(repoPath, opencodeProjectPluginsDir), agentsHome)
	for _, homeRoot := range config.UserHomeRoots() {
		_ = removePluginLinks(filepath.Join(homeRoot, opencodeGlobalPluginsDir), agentsHome)
	}

	skillsDir := filepath.Join(repoPath, ".agents", "skills")
	if entries, err := os.ReadDir(skillsDir); err == nil {
		for _, e := range entries {
			links.RemoveIfSymlinkUnder(filepath.Join(skillsDir, e.Name()), agentsHome)
		}
	}

	return nil
}

func removePluginLinks(root, agentsHome string) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	for _, entry := range entries {
		path := filepath.Join(root, entry.Name())
		if entry.IsDir() {
			if err := removePluginLinks(path, agentsHome); err != nil {
				return err
			}
			continue
		}
		_ = links.RemoveIfSymlinkUnder(path, agentsHome)
	}
	return nil
}
