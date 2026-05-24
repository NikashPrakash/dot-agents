package workflow

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/sys/execabs"
)

// CommitPathSet is the deterministic, scoped path set the workflow-state
// commit subcommand (wc-commit-subcommand, downstream) will stage. It is the
// pure-data output of the derivation step: same mutation set + same git
// status ⇒ same path set.
//
// Paths are repo-relative, forward-slash normalised, and sorted (lexical).
// Membership rules are enforced by deriveWorkflowCommitPaths and documented
// inline there.
type CommitPathSet struct {
	// Paths is the staged set: each entry is a repo-relative POSIX path that
	// the caller should pass to `git add --` (literal, not a pathspec).
	Paths []string
	// ExcludedSubmodule lists status entries that were dropped because they
	// resolved to a submodule pointer (gitlink, mode 160000). Surfaced for
	// the --dry-run diagnostic line in the downstream subcommand task.
	ExcludedSubmodule []string
	// ExcludedUntrackedDir lists status entries that were dropped because
	// they were pre-existing-untracked directories not in the workflow
	// mutation surface. Surfaced for the --dry-run diagnostic line.
	ExcludedUntrackedDir []string
}

// workflowMutationSurface enumerates the canonical-store path roots that the
// workflow commit derivation considers "managed" — paths under any of these
// roots are always candidates for the commit set when dirty. The list is
// deliberately small and explicit; we never reach for a blanket `.agents/**`
// scope (see spec design.md decision #3).
//
// Each entry is a repo-relative forward-slash directory prefix (no trailing
// slash). Order is irrelevant; isManagedWorkflowPath uses prefix matching.
var workflowMutationSurface = []string{
	".agents/workflow",
	".agents/history",
}

// gitStatusEntry is one parsed line from `git status --porcelain=v2 -z`.
// Only the fields the derivation cares about are populated.
type gitStatusEntry struct {
	// path is the repo-relative POSIX path (forward slashes).
	path string
	// xy is the two-letter porcelain v1 status code (e.g. " M", "??",
	// "M ", "A ", "RM"). For porcelain v2 we synthesise this from the X/Y
	// fields so downstream consumers can keep the v1 mental model.
	xy string
	// isSubmodule is true when the entry resolves to a submodule pointer
	// (gitlink). git porcelain v2 surfaces this directly in the "1"/"2"
	// record sub-fields; we propagate it so the derivation can drop the
	// entry without re-shelling to ls-files.
	isSubmodule bool
	// isUntracked is true for "?" porcelain v2 records (untracked entries).
	isUntracked bool
	// isUntrackedDir is true when an untracked entry's path ends with "/"
	// (porcelain v2 reports untracked directories with a trailing slash
	// when -uno is not used). The derivation excludes these unless every
	// path under them is in the mutation surface.
	isUntrackedDir bool
}

// deriveWorkflowCommitPaths is the pure-core path derivation function.
//
// Inputs:
//   - entries: parsed git status entries for the worktree (typically the
//     output of parseGitStatusPorcelainV2 against `git status --porcelain=v2
//     -z`). Pure — the caller injects this so tests can drive synthetic
//     fixtures without spawning git.
//   - mutationSurface: the set of repo-relative paths the workflow code
//     touched in this session (e.g. specific .agents/workflow/plans/<id>/
//     files plus session-touched state files). May be empty — in that case
//     only entries under workflowMutationSurface roots are considered.
//
// Rules (in order):
//  1. Drop any entry whose isSubmodule is true (mandatory exclusion per
//     spec design.md decision #3 — submodule pointers are never workflow
//     state).
//  2. Drop any untracked-directory entry whose path is NOT a direct member
//     of the mutation surface AND is NOT inside a managed workflow root
//     (mandatory exclusion — "pre-existing-untracked dirs").
//  3. Keep an entry iff its path is under one of the managed workflow roots
//     OR appears in the mutation surface set (intersection with dirty).
//  4. Output is sorted lexically and de-duplicated (determinism).
//
// The function never returns an error: it is total over its inputs.
func deriveWorkflowCommitPaths(entries []gitStatusEntry, mutationSurface []string) CommitPathSet {
	surface := make(map[string]struct{}, len(mutationSurface))
	for _, p := range mutationSurface {
		surface[filepath.ToSlash(p)] = struct{}{}
	}

	seen := make(map[string]struct{}, len(entries))
	var kept []string
	var droppedSubmodule []string
	var droppedUntrackedDir []string

	for _, e := range entries {
		path := filepath.ToSlash(e.path)
		// Rule 1: submodule pointers are mandatory excludes.
		if e.isSubmodule {
			droppedSubmodule = append(droppedSubmodule, path)
			continue
		}
		// Rule 2: pre-existing-untracked dirs not on the mutation surface.
		if e.isUntrackedDir {
			if _, inSurface := surface[strings.TrimSuffix(path, "/")]; !inSurface && !isManagedWorkflowPath(path) {
				droppedUntrackedDir = append(droppedUntrackedDir, path)
				continue
			}
		}
		// Rule 3: must be either under a managed workflow root or in the
		// caller-supplied mutation surface set.
		_, inSurface := surface[path]
		if !isManagedWorkflowPath(path) && !inSurface {
			continue
		}
		if _, dup := seen[path]; dup {
			continue
		}
		seen[path] = struct{}{}
		kept = append(kept, path)
	}

	// Rule 4: determinism — sort lexically. The seen map already de-duped.
	sort.Strings(kept)
	sort.Strings(droppedSubmodule)
	sort.Strings(droppedUntrackedDir)

	return CommitPathSet{
		Paths:                kept,
		ExcludedSubmodule:    droppedSubmodule,
		ExcludedUntrackedDir: droppedUntrackedDir,
	}
}

// isManagedWorkflowPath reports whether path lies under one of the managed
// workflow root prefixes (.agents/workflow/**, .agents/history/**).
//
// Match is by POSIX path prefix followed by "/" — a literal "x/workflow"
// prefix would otherwise spuriously match ".agents/workflow-foo/y", which we
// must never include. Inputs may use either separator; we normalise.
func isManagedWorkflowPath(path string) bool {
	p := filepath.ToSlash(path)
	for _, root := range workflowMutationSurface {
		if p == root || strings.HasPrefix(p, root+"/") {
			return true
		}
	}
	return false
}

// parseGitStatusPorcelainV2 decodes `git status --porcelain=v2 -z` output
// into the gitStatusEntry slice. The -z flag uses NUL byte record separators
// and disables path quoting/encoding (essential for paths with embedded
// whitespace).
//
// Porcelain v2 record formats this function handles:
//
//	"1 XY sub mH mI mW hH hI path"            — ordinary change
//	"2 XY sub mH mI mW hH hI X<score> path\tsep_orig_path"  — renamed/copied
//	"u XY sub m1 m2 m3 mW h1 h2 h3 path"      — unmerged
//	"? path"                                  — untracked
//	"! path"                                  — ignored (we skip)
//
// The "sub" sub-field is "N..." for non-submodule entries and "S..." for
// submodule pointers; that is how we set isSubmodule deterministically
// without re-shelling.
//
// The function returns a (possibly empty) slice; on malformed input it skips
// the offending record and continues — git is the source of truth and we do
// not want a single garbled byte to drop the entire set.
func parseGitStatusPorcelainV2(z []byte) []gitStatusEntry {
	records := splitNullSeparated(z)
	out := make([]gitStatusEntry, 0, len(records))
	for i := 0; i < len(records); i++ {
		r := records[i]
		if len(r) == 0 {
			continue
		}
		switch r[0] {
		case '1':
			if e, ok := parseOrdinaryChangeRecord(r); ok {
				out = append(out, e)
			}
		case '2':
			// "2 XY sub ... path"; followed by an extra NUL-separated
			// record for the original (pre-rename) path. We consume both
			// records but only stage the destination path — the original
			// is implicitly handled by the rename's R/C status.
			if e, ok := parseRenameRecord(r); ok {
				out = append(out, e)
			}
			if i+1 < len(records) {
				i++ // consume the orig-path record
			}
		case 'u':
			if e, ok := parseUnmergedRecord(r); ok {
				out = append(out, e)
			}
		case '?':
			if e, ok := parseUntrackedRecord(r); ok {
				out = append(out, e)
			}
		case '!':
			// Ignored entries are never workflow state; skip.
			continue
		default:
			// Unknown record type — defensively skip.
			continue
		}
	}
	return out
}

// splitNullSeparated splits z on the NUL byte. It tolerates a trailing NUL
// (which git produces) without emitting an empty trailing record.
func splitNullSeparated(z []byte) []string {
	s := string(z)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, "\x00")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}

// parseOrdinaryChangeRecord decodes a "1 XY sub mH mI mW hH hI path" line.
// Fields are space-separated; the path is the 9th field and may contain
// spaces (the leading 8 fields never do, so a SplitN(9) is sufficient).
func parseOrdinaryChangeRecord(line string) (gitStatusEntry, bool) {
	fields := strings.SplitN(line, " ", 9)
	if len(fields) < 9 {
		return gitStatusEntry{}, false
	}
	xy, sub, path := fields[1], fields[2], fields[8]
	return gitStatusEntry{
		path:        path,
		xy:          xy,
		isSubmodule: isSubmoduleSubField(sub),
	}, true
}

// parseRenameRecord decodes a "2 XY sub mH mI mW hH hI X<score> path" line.
// Layout matches ordinary changes plus an X<score> field at position 8 and
// the path at position 9. The original path follows in the next NUL-record
// (consumed by the caller).
func parseRenameRecord(line string) (gitStatusEntry, bool) {
	fields := strings.SplitN(line, " ", 10)
	if len(fields) < 10 {
		return gitStatusEntry{}, false
	}
	xy, sub, path := fields[1], fields[2], fields[9]
	return gitStatusEntry{
		path:        path,
		xy:          xy,
		isSubmodule: isSubmoduleSubField(sub),
	}, true
}

// parseUnmergedRecord decodes a "u XY sub m1 m2 m3 mW h1 h2 h3 path" line.
func parseUnmergedRecord(line string) (gitStatusEntry, bool) {
	fields := strings.SplitN(line, " ", 11)
	if len(fields) < 11 {
		return gitStatusEntry{}, false
	}
	xy, sub, path := fields[1], fields[2], fields[10]
	return gitStatusEntry{
		path:        path,
		xy:          xy,
		isSubmodule: isSubmoduleSubField(sub),
	}, true
}

// parseUntrackedRecord decodes a "? path" line. Untracked records have no
// status code per se; we synthesise "??" to keep the v1 mental model.
func parseUntrackedRecord(line string) (gitStatusEntry, bool) {
	if len(line) < 3 || line[1] != ' ' {
		return gitStatusEntry{}, false
	}
	path := line[2:]
	// In porcelain v2 -z output, untracked dirs are reported as a single
	// record with a trailing slash on the path.
	isDir := strings.HasSuffix(path, "/")
	return gitStatusEntry{
		path:           path,
		xy:             "??",
		isUntracked:    true,
		isUntrackedDir: isDir,
	}, true
}

// isSubmoduleSubField reports whether the "sub" field of a porcelain v2
// change record indicates a submodule pointer. Per git docs, the field is
// either "N..." (not a submodule) or "S<c><m><u>" where the letter S marks
// it as a submodule.
func isSubmoduleSubField(sub string) bool {
	return len(sub) > 0 && sub[0] == 'S'
}

// gitStatusPorcelainV2 invokes `git status --porcelain=v2 -z --untracked-files=all`
// in the given repo and returns the raw bytes. Kept thin and unexported
// because the pure derivation core takes the parsed entries — this exists
// only to bridge to production callers (the downstream wc-commit-subcommand
// task).
//
// `--untracked-files=all` is required (not optional): the default `normal`
// mode collapses an untracked directory to a single record (e.g.
// ".agents/workflow/" with a trailing slash) so a brand-new plan directory
// becomes one indistinguishable blob the derivation cannot stage at file
// granularity. With `all`, git emits one "? <path>" record per untracked
// file — exactly what the derivation needs to make per-file decisions.
//
// The pre-existing-untracked-dir exclusion (spec rule) still functions:
// dir-level entries can appear when a user has an actually-empty untracked
// directory in the worktree (those still surface as "? path/"), and the
// rule in deriveWorkflowCommitPaths handles them. We are not bypassing the
// rule — we are getting per-file visibility into directories that actually
// contain workflow state.
func gitStatusPorcelainV2(projectPath string) ([]byte, error) {
	cmd := execabs.Command("git", "-C", projectPath,
		"status", "--porcelain=v2", "-z", "--untracked-files=all")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git status --porcelain=v2 -z: %w", err)
	}
	return out, nil
}

// DeriveWorkflowCommitPathsFromRepo is the production entry point: read git
// status in projectPath, parse it, and run the pure derivation against the
// given mutation surface. Exists so the downstream subcommand task can call
// one function; the derivation logic itself lives in deriveWorkflowCommitPaths
// and is independently unit-tested.
func DeriveWorkflowCommitPathsFromRepo(projectPath string, mutationSurface []string) (CommitPathSet, error) {
	raw, err := gitStatusPorcelainV2(projectPath)
	if err != nil {
		return CommitPathSet{}, err
	}
	entries := parseGitStatusPorcelainV2(raw)
	return deriveWorkflowCommitPaths(entries, mutationSurface), nil
}
