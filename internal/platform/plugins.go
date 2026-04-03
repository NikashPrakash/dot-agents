package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

const PluginManifestName = "PLUGIN.yaml"

type PluginKind string

const (
	PluginKindNative  PluginKind = "native"
	PluginKindPackage PluginKind = "package"
)

type PluginResources struct {
	Agents   []string `yaml:"agents,omitempty"`
	Skills   []string `yaml:"skills,omitempty"`
	Commands []string `yaml:"commands,omitempty"`
	Hooks    []string `yaml:"hooks,omitempty"`
	MCP      []string `yaml:"mcp,omitempty"`
}

type PluginMarketplace struct {
	Repo string   `yaml:"repo,omitempty"`
	Tags []string `yaml:"tags,omitempty"`
}

// PluginSpec is the canonical dot-agents plugin bundle manifest.
type PluginSpec struct {
	Kind              PluginKind                `yaml:"kind"`
	Name              string                    `yaml:"name"`
	Version           string                    `yaml:"version,omitempty"`
	DisplayName       string                    `yaml:"display_name,omitempty"`
	Description       string                    `yaml:"description,omitempty"`
	Authors           []string                  `yaml:"authors,omitempty"`
	Homepage          string                    `yaml:"homepage,omitempty"`
	License           string                    `yaml:"license,omitempty"`
	Platforms         []string                  `yaml:"platforms"`
	Resources         PluginResources           `yaml:"resources,omitempty"`
	Marketplace       PluginMarketplace         `yaml:"marketplace,omitempty"`
	Dependencies      map[string]any            `yaml:"dependencies,omitempty"`
	PlatformOverrides map[string]map[string]any `yaml:"platform_overrides,omitempty"`
	Dir               string                    `yaml:"-"`
	ManifestPath      string                    `yaml:"-"`
	Scope             string                    `yaml:"-"`
}

// LoadPluginSpec parses and validates a canonical plugin manifest from pluginDir.
func LoadPluginSpec(pluginDir string) (PluginSpec, error) {
	manifestPath := filepath.Join(pluginDir, PluginManifestName)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return PluginSpec{}, fmt.Errorf("reading plugin manifest %s: %w", manifestPath, err)
	}

	var spec PluginSpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return PluginSpec{}, fmt.Errorf("parsing plugin manifest %s: %w", manifestPath, err)
	}

	spec.Dir = pluginDir
	spec.ManifestPath = manifestPath

	if err := validatePluginSpec(spec); err != nil {
		return PluginSpec{}, fmt.Errorf("invalid plugin manifest %s: %w", manifestPath, err)
	}

	spec.Platforms = sortedUniqueStrings(spec.Platforms)
	spec.Authors = sortedUniqueStrings(spec.Authors)
	spec.Resources.Agents = sortedUniqueStrings(spec.Resources.Agents)
	spec.Resources.Skills = sortedUniqueStrings(spec.Resources.Skills)
	spec.Resources.Commands = sortedUniqueStrings(spec.Resources.Commands)
	spec.Resources.Hooks = sortedUniqueStrings(spec.Resources.Hooks)
	spec.Resources.MCP = sortedUniqueStrings(spec.Resources.MCP)
	spec.Marketplace.Tags = sortedUniqueStrings(spec.Marketplace.Tags)

	return spec, nil
}

// ListPluginSpecs returns valid canonical plugin specs for a scope.
func ListPluginSpecs(agentsHome, scope string) ([]PluginSpec, error) {
	root := filepath.Join(agentsHome, "plugins", scope)
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	specs := make([]PluginSpec, 0, len(entries))
	for _, entry := range entries {
		pluginDir := filepath.Join(root, entry.Name())
		if !isPluginDir(pluginDir) {
			continue
		}

		spec, err := LoadPluginSpec(pluginDir)
		if err != nil {
			return nil, err
		}
		spec.Scope = scope
		specs = append(specs, spec)
	}

	sort.Slice(specs, func(i, j int) bool {
		return specs[i].Name < specs[j].Name
	})
	return specs, nil
}

func validatePluginSpec(spec PluginSpec) error {
	switch spec.Kind {
	case PluginKindNative, PluginKindPackage:
	default:
		return fmt.Errorf("kind must be %q or %q", PluginKindNative, PluginKindPackage)
	}

	if strings.TrimSpace(spec.Name) == "" {
		return fmt.Errorf("name is required")
	}

	if len(spec.Platforms) == 0 {
		return fmt.Errorf("platforms must contain at least one platform id")
	}
	for _, platformID := range spec.Platforms {
		if !IsKnownID(platformID) {
			return fmt.Errorf("unknown platform %q", platformID)
		}
	}
	for platformID := range spec.PlatformOverrides {
		if !IsKnownID(platformID) {
			return fmt.Errorf("unknown platform override %q", platformID)
		}
	}

	return nil
}

func isPluginDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return false
	}
	_, err = os.Stat(filepath.Join(path, PluginManifestName))
	return err == nil
}

func sortedUniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
