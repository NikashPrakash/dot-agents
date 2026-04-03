package platform

import "strings"

func shouldRenderPluginMarketplace(spec PluginSpec, platformID string) bool {
	if strings.TrimSpace(spec.Marketplace.Repo) != "" || len(spec.Marketplace.Tags) > 0 {
		return true
	}
	return strings.TrimSpace(pluginOverrideString(spec, platformID, "marketplace_name")) != ""
}

func renderClaudeMarketplace(spec PluginSpec) ([]byte, error) {
	entry := buildGenericMarketplacePluginEntry(spec, ".", "claude")
	payload := genericMarketplace{
		Name:    defaultMarketplaceName(spec, "claude"),
		Owner:   pluginAuthorFromSpec(spec),
		Plugins: []genericMarketplacePluginEntry{entry},
	}
	if desc := strings.TrimSpace(spec.Description); desc != "" || strings.TrimSpace(spec.Version) != "" {
		payload.Metadata = &genericMarketplaceMetadata{
			Description: strings.TrimSpace(spec.Description),
			Version:     strings.TrimSpace(spec.Version),
		}
	}
	return marshalJSON(payload)
}

func renderCursorMarketplace(spec PluginSpec) ([]byte, error) {
	entry := buildGenericMarketplacePluginEntry(spec, ".", "cursor")
	payload := genericMarketplace{
		Name:    defaultMarketplaceName(spec, "cursor"),
		Owner:   pluginAuthorFromSpec(spec),
		Plugins: []genericMarketplacePluginEntry{entry},
	}
	if desc := strings.TrimSpace(spec.Description); desc != "" || strings.TrimSpace(spec.Version) != "" {
		payload.Metadata = &genericMarketplaceMetadata{
			Description: strings.TrimSpace(spec.Description),
			Version:     strings.TrimSpace(spec.Version),
		}
	}
	return marshalJSON(payload)
}

func renderCopilotMarketplace(spec PluginSpec) ([]byte, error) {
	entry := buildGenericMarketplacePluginEntry(spec, ".", "copilot")
	payload := genericMarketplace{
		Name:    defaultMarketplaceName(spec, "copilot"),
		Owner:   pluginAuthorFromSpec(spec),
		Plugins: []genericMarketplacePluginEntry{entry},
	}
	if desc := strings.TrimSpace(spec.Description); desc != "" || strings.TrimSpace(spec.Version) != "" {
		payload.Metadata = &genericMarketplaceMetadata{
			Description: strings.TrimSpace(spec.Description),
			Version:     strings.TrimSpace(spec.Version),
		}
	}
	return marshalJSON(payload)
}

func renderCodexMarketplace(spec PluginSpec) ([]byte, error) {
	category := pluginOverrideString(spec, "codex", "category")
	if category == "" {
		category = "Productivity"
	}
	payload := codexMarketplace{
		Name: defaultMarketplaceName(spec, "codex"),
		Plugins: []codexMarketplacePluginEntry{
			{
				Name: spec.Name,
				Source: codexMarketplaceSource{
					Source: "local",
					Path:   ".",
				},
				Policy: codexMarketplacePolicy{
					Installation:   "AVAILABLE",
					Authentication: "ON_INSTALL",
				},
				Category: category,
			},
		},
	}
	if displayName := strings.TrimSpace(spec.DisplayName); displayName != "" {
		payload.Interface = &codexMarketplaceInterface{DisplayName: displayName}
	}
	return marshalJSON(payload)
}

type genericMarketplace struct {
	Name     string                          `json:"name"`
	Owner    *pluginAuthor                   `json:"owner,omitempty"`
	Metadata *genericMarketplaceMetadata     `json:"metadata,omitempty"`
	Plugins  []genericMarketplacePluginEntry `json:"plugins"`
}

type genericMarketplaceMetadata struct {
	Description string `json:"description,omitempty"`
	Version     string `json:"version,omitempty"`
	PluginRoot  string `json:"pluginRoot,omitempty"`
}

type genericMarketplacePluginEntry struct {
	Name        string        `json:"name"`
	Source      string        `json:"source"`
	Description string        `json:"description,omitempty"`
	Version     string        `json:"version,omitempty"`
	Author      *pluginAuthor `json:"author,omitempty"`
	Homepage    string        `json:"homepage,omitempty"`
	Repository  string        `json:"repository,omitempty"`
	License     string        `json:"license,omitempty"`
	Keywords    []string      `json:"keywords,omitempty"`
	Category    string        `json:"category,omitempty"`
	Tags        []string      `json:"tags,omitempty"`
}

type codexMarketplace struct {
	Name      string                        `json:"name"`
	Interface *codexMarketplaceInterface    `json:"interface,omitempty"`
	Plugins   []codexMarketplacePluginEntry `json:"plugins"`
}

type codexMarketplaceInterface struct {
	DisplayName string `json:"displayName,omitempty"`
}

type codexMarketplacePluginEntry struct {
	Name     string                 `json:"name"`
	Source   codexMarketplaceSource `json:"source"`
	Policy   codexMarketplacePolicy `json:"policy"`
	Category string                 `json:"category"`
}

type codexMarketplaceSource struct {
	Source string `json:"source"`
	Path   string `json:"path"`
}

type codexMarketplacePolicy struct {
	Installation   string   `json:"installation"`
	Authentication string   `json:"authentication"`
	Products       []string `json:"products,omitempty"`
}

func buildGenericMarketplacePluginEntry(spec PluginSpec, source, platformID string) genericMarketplacePluginEntry {
	entry := genericMarketplacePluginEntry{
		Name:        spec.Name,
		Source:      source,
		Description: strings.TrimSpace(spec.Description),
		Version:     strings.TrimSpace(spec.Version),
		Author:      pluginAuthorFromSpec(spec),
		Homepage:    strings.TrimSpace(spec.Homepage),
		Repository:  strings.TrimSpace(spec.Marketplace.Repo),
		License:     strings.TrimSpace(spec.License),
		Keywords:    spec.Marketplace.Tags,
		Tags:        spec.Marketplace.Tags,
	}
	if category := pluginOverrideString(spec, platformID, "category"); category != "" {
		entry.Category = category
	}
	return entry
}

func defaultMarketplaceName(spec PluginSpec, platformID string) string {
	if name := pluginOverrideString(spec, platformID, "marketplace_name"); name != "" {
		return name
	}
	return spec.Name + "-" + platformID + "-marketplace"
}
