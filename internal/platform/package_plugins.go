package platform

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/NikashPrakash/dot-agents/internal/links"
)

type pluginAuthor struct {
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
	URL   string `json:"url,omitempty"`
}

func pluginFilesDir(spec PluginSpec) string {
	return filepath.Join(spec.Dir, "files")
}

func pluginResourcesDir(spec PluginSpec, component string) string {
	return filepath.Join(spec.Dir, "resources", component)
}

func pluginPlatformDir(spec PluginSpec, platformID string) string {
	return filepath.Join(spec.Dir, "platforms", platformID)
}

func pluginOverrideString(spec PluginSpec, platformID, key string) string {
	if spec.PlatformOverrides == nil {
		return ""
	}
	values, ok := spec.PlatformOverrides[platformID]
	if !ok {
		return ""
	}
	value, ok := values[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func pluginOverrideStringSlice(spec PluginSpec, platformID, key string) []string {
	if spec.PlatformOverrides == nil {
		return nil
	}
	values, ok := spec.PlatformOverrides[platformID]
	if !ok {
		return nil
	}
	raw, ok := values[key]
	if !ok {
		return nil
	}
	switch typed := raw.(type) {
	case []string:
		return sortedUniqueStrings(typed)
	case []any:
		out := make([]string, 0, len(typed))
		for _, entry := range typed {
			text, ok := entry.(string)
			if !ok {
				continue
			}
			out = append(out, text)
		}
		return sortedUniqueStrings(out)
	default:
		return nil
	}
}

func pluginAuthorFromSpec(spec PluginSpec) *pluginAuthor {
	if len(spec.Authors) == 0 {
		return nil
	}
	name := strings.TrimSpace(spec.Authors[0])
	if name == "" {
		return nil
	}
	return &pluginAuthor{Name: name}
}

func listPackagePluginsForPlatformInScope(agentsHome, scope, platformID string) ([]PluginSpec, error) {
	specs, err := ListPluginSpecs(agentsHome, scope)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	out := make([]PluginSpec, 0, len(specs))
	for _, spec := range specs {
		if spec.Kind != PluginKindPackage || !pluginSpecHasPlatform(spec, platformID) {
			continue
		}
		out = append(out, spec)
	}
	return out, nil
}

func pluginSpecHasPlatform(spec PluginSpec, platformID string) bool {
	for _, id := range spec.Platforms {
		if id == platformID {
			return true
		}
	}
	return false
}

func selectedPackagePluginForPlatform(agentsHome, project, platformID string) (*PluginSpec, string, error) {
	for _, scope := range scopedNames(project) {
		specs, err := listPackagePluginsForPlatformInScope(agentsHome, scope, platformID)
		if err != nil {
			return nil, "", err
		}
		if len(specs) == 1 {
			spec := specs[0]
			return &spec, scope, nil
		}
		if len(specs) > 1 {
			return nil, scope, nil
		}
	}
	return nil, "", nil
}

func preferredPackagePluginsForPlatform(agentsHome, project, platformID string) ([]PluginSpec, string, error) {
	for _, scope := range scopedNames(project) {
		specs, err := listPackagePluginsForPlatformInScope(agentsHome, scope, platformID)
		if err != nil {
			return nil, "", err
		}
		if len(specs) > 0 {
			return specs, scope, nil
		}
	}
	return nil, "", nil
}

func existingPluginSourceRoots(paths ...string) []string {
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			out = append(out, path)
		}
	}
	return out
}

func syncPluginOverlayTree(dstRoot string, srcRoots ...string) error {
	desired, err := collectPluginOverlayFiles(srcRoots...)
	if err != nil {
		return err
	}
	if len(desired) == 0 {
		return removeManagedPluginOverlayTree(dstRoot)
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
	return pruneStalePluginOverlayTree(dstRoot, desired)
}

func collectPluginOverlayFiles(srcRoots ...string) (map[string]string, error) {
	out := map[string]string{}
	for _, root := range srcRoots {
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

func removeManagedPluginOverlayTree(dstRoot string, srcRoots ...string) error {
	info, err := os.Stat(dstRoot)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return nil
	}

	roots := normalizeRoots(srcRoots)
	if err := filepath.WalkDir(dstRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if os.IsNotExist(walkErr) {
				return nil
			}
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if len(roots) == 0 {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return err
			}
			return nil
		}
		for _, root := range roots {
			if links.IsSymlinkUnder(path, root) {
				if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
					return err
				}
				break
			}
		}
		return nil
	}); err != nil {
		return err
	}
	return pruneEmptyDirsBottomUp(dstRoot)
}

func pruneStalePluginOverlayTree(dstRoot string, desired map[string]string) error {
	info, err := os.Stat(dstRoot)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return nil
	}

	if err := filepath.WalkDir(dstRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if os.IsNotExist(walkErr) {
				return nil
			}
			return walkErr
		}
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(dstRoot, path)
		if err != nil {
			return err
		}
		want, ok := desired[rel]
		if ok && links.IsSymlinkTo(path, want) {
			return nil
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return pruneEmptyDirsBottomUp(dstRoot)
}

func normalizeRoots(srcRoots []string) []string {
	out := make([]string, 0, len(srcRoots))
	for _, root := range srcRoots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		out = append(out, filepath.Clean(root))
	}
	return out
}

func pruneEmptyDirsBottomUp(root string) error {
	info, err := os.Stat(root)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return nil
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

func pluginDirHasFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			return true
		}
		subdir := filepath.Join(dir, entry.Name())
		if pluginDirHasFiles(subdir) {
			return true
		}
	}
	return false
}
