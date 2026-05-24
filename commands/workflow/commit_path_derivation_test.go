package workflow

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"golang.org/x/sys/execabs"
)

// ── Pure-core determinism / scoping tests ────────────────────────────────────
//
// These exercise deriveWorkflowCommitPaths against hand-crafted
// gitStatusEntry slices so the rules are isolated from git's behaviour.

func TestDeriveWorkflowCommitPaths_OnlyManagedRootsAndSurface(t *testing.T) {
	entries := []gitStatusEntry{
		{path: ".agents/workflow/plans/p1/TASKS.yaml", xy: " M"},
		{path: ".agents/history/p0/PLAN.yaml", xy: " M"},
		{path: "commands/workflow/commit_path_derivation.go", xy: " M"},
		{path: "README.md", xy: " M"},
		{path: ".agents/lessons/foo/LESSON.md", xy: " M"},
	}
	surface := []string{"commands/workflow/commit_path_derivation.go"}

	got := deriveWorkflowCommitPaths(entries, surface)

	want := []string{
		".agents/history/p0/PLAN.yaml",
		".agents/workflow/plans/p1/TASKS.yaml",
		"commands/workflow/commit_path_derivation.go",
	}
	if !reflect.DeepEqual(got.Paths, want) {
		t.Fatalf("paths mismatch:\n got=%v\nwant=%v", got.Paths, want)
	}
	if len(got.ExcludedSubmodule) != 0 {
		t.Fatalf("expected no submodule excludes, got %v", got.ExcludedSubmodule)
	}
	if len(got.ExcludedUntrackedDir) != 0 {
		t.Fatalf("expected no untracked-dir excludes, got %v", got.ExcludedUntrackedDir)
	}
}

func TestDeriveWorkflowCommitPaths_DropsSubmodulePointers(t *testing.T) {
	entries := []gitStatusEntry{
		{path: "vendor/some-submodule", xy: " M", isSubmodule: true},
		{path: ".agents/workflow/plans/p1/TASKS.yaml", xy: " M"},
		// A submodule pointer inside the managed roots is still excluded —
		// mandatory rule, irrespective of location.
		{path: ".agents/workflow/external-mod", xy: " M", isSubmodule: true},
	}

	got := deriveWorkflowCommitPaths(entries, nil)

	want := []string{".agents/workflow/plans/p1/TASKS.yaml"}
	if !reflect.DeepEqual(got.Paths, want) {
		t.Fatalf("paths mismatch:\n got=%v\nwant=%v", got.Paths, want)
	}
	wantExcl := []string{".agents/workflow/external-mod", "vendor/some-submodule"}
	if !reflect.DeepEqual(got.ExcludedSubmodule, wantExcl) {
		t.Fatalf("submodule excludes mismatch:\n got=%v\nwant=%v", got.ExcludedSubmodule, wantExcl)
	}
}

func TestDeriveWorkflowCommitPaths_DropsPreExistingUntrackedDirs(t *testing.T) {
	entries := []gitStatusEntry{
		// Pre-existing untracked dir outside the surface — excluded.
		{path: "tmp/", xy: "??", isUntracked: true, isUntrackedDir: true},
		// Untracked dir inside .agents/workflow → KEPT (it's a managed
		// root, the new plan/<id>/ scaffold case).
		{path: ".agents/workflow/plans/p2/", xy: "??", isUntracked: true, isUntrackedDir: true},
		// Untracked file in a managed root → KEPT.
		{path: ".agents/history/p1/impl-results.md", xy: "??", isUntracked: true},
		// Untracked file NOT in a managed root and NOT in surface → DROPPED
		// (not by the dir rule but by the "not on surface" rule).
		{path: "scratch.log", xy: "??", isUntracked: true},
	}

	got := deriveWorkflowCommitPaths(entries, nil)

	want := []string{
		".agents/history/p1/impl-results.md",
		".agents/workflow/plans/p2/",
	}
	if !reflect.DeepEqual(got.Paths, want) {
		t.Fatalf("paths mismatch:\n got=%v\nwant=%v", got.Paths, want)
	}
	wantExcl := []string{"tmp/"}
	if !reflect.DeepEqual(got.ExcludedUntrackedDir, wantExcl) {
		t.Fatalf("untracked-dir excludes mismatch:\n got=%v\nwant=%v",
			got.ExcludedUntrackedDir, wantExcl)
	}
}

func TestDeriveWorkflowCommitPaths_Deterministic_SameInputSameOutput(t *testing.T) {
	entries := []gitStatusEntry{
		{path: ".agents/workflow/plans/z/TASKS.yaml", xy: " M"},
		{path: ".agents/workflow/plans/a/TASKS.yaml", xy: " M"},
		{path: ".agents/history/m/PLAN.yaml", xy: " M"},
		// Duplicate — must be de-duped.
		{path: ".agents/workflow/plans/a/TASKS.yaml", xy: "M "},
	}
	got1 := deriveWorkflowCommitPaths(entries, nil)
	got2 := deriveWorkflowCommitPaths(entries, nil)
	if !reflect.DeepEqual(got1, got2) {
		t.Fatalf("non-deterministic: %v vs %v", got1, got2)
	}
	want := []string{
		".agents/history/m/PLAN.yaml",
		".agents/workflow/plans/a/TASKS.yaml",
		".agents/workflow/plans/z/TASKS.yaml",
	}
	if !reflect.DeepEqual(got1.Paths, want) {
		t.Fatalf("not sorted/deduped:\n got=%v\nwant=%v", got1.Paths, want)
	}
}

func TestDeriveWorkflowCommitPaths_PrefixGuardNeverMatchesSiblings(t *testing.T) {
	entries := []gitStatusEntry{
		// `.agents/workflow-foo` must NEVER be treated as inside
		// `.agents/workflow` — that would be the classic prefix-bug.
		{path: ".agents/workflow-foo/bar.txt", xy: " M"},
		// Same for `.agents/historyx`.
		{path: ".agents/historyx/y.md", xy: " M"},
		// Sanity: a real managed-root entry IS kept.
		{path: ".agents/workflow/plans/p/TASKS.yaml", xy: " M"},
	}
	got := deriveWorkflowCommitPaths(entries, nil)
	want := []string{".agents/workflow/plans/p/TASKS.yaml"}
	if !reflect.DeepEqual(got.Paths, want) {
		t.Fatalf("prefix-guard regressed:\n got=%v\nwant=%v", got.Paths, want)
	}
}

func TestDeriveWorkflowCommitPaths_EmptyEntries(t *testing.T) {
	got := deriveWorkflowCommitPaths(nil, nil)
	if len(got.Paths) != 0 {
		t.Fatalf("expected empty paths, got %v", got.Paths)
	}
}

// ── Porcelain v2 parser tests ────────────────────────────────────────────────

func TestParseGitStatusPorcelainV2_OrdinaryAndUntracked(t *testing.T) {
	// Build a NUL-separated v2 stream with one ordinary change and one
	// untracked file. Field layout:
	//   "1 XY sub mH mI mW hH hI path"
	z := []byte(
		"1 .M N... 100644 100644 100644 abc abc .agents/workflow/plans/p/TASKS.yaml\x00" +
			"? scratch.log\x00",
	)
	got := parseGitStatusPorcelainV2(z)
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d: %#v", len(got), got)
	}
	if got[0].path != ".agents/workflow/plans/p/TASKS.yaml" || got[0].xy != ".M" || got[0].isSubmodule {
		t.Fatalf("ordinary entry parsed wrong: %#v", got[0])
	}
	if got[1].path != "scratch.log" || !got[1].isUntracked || got[1].isUntrackedDir {
		t.Fatalf("untracked entry parsed wrong: %#v", got[1])
	}
}

func TestParseGitStatusPorcelainV2_SubmoduleSubFieldDetected(t *testing.T) {
	// "S..." sub field marks a submodule pointer per git docs.
	z := []byte("1 .M S.MU 160000 160000 160000 abc abc vendor/sub\x00")
	got := parseGitStatusPorcelainV2(z)
	if len(got) != 1 || !got[0].isSubmodule {
		t.Fatalf("expected submodule flagged, got %#v", got)
	}
}

func TestParseGitStatusPorcelainV2_UntrackedDirHasTrailingSlash(t *testing.T) {
	z := []byte("? tmp/\x00")
	got := parseGitStatusPorcelainV2(z)
	if len(got) != 1 || !got[0].isUntrackedDir {
		t.Fatalf("expected untracked-dir flag, got %#v", got)
	}
}

func TestParseGitStatusPorcelainV2_RenameSkipsOrigPathRecord(t *testing.T) {
	// "2 XY sub mH mI mW hH hI Xscore path\x00orig_path\x00"
	z := []byte(
		"2 R. N... 100644 100644 100644 abc def R100 new.txt\x00old.txt\x00",
	)
	got := parseGitStatusPorcelainV2(z)
	if len(got) != 1 {
		t.Fatalf("rename should produce one entry (dest only), got %d: %#v", len(got), got)
	}
	if got[0].path != "new.txt" {
		t.Fatalf("rename dest path wrong: %#v", got[0])
	}
}

func TestParseGitStatusPorcelainV2_IgnoresMalformedAndIgnored(t *testing.T) {
	z := []byte(
		"! ignored.txt\x00" + // ignored entry — skipped
			"1 .M\x00" + // malformed ordinary — skipped
			"? real.txt\x00",
	)
	got := parseGitStatusPorcelainV2(z)
	if len(got) != 1 || got[0].path != "real.txt" {
		t.Fatalf("expected only real.txt, got %#v", got)
	}
}

// ── End-to-end fixture tests: dirty worktree + submodule pointer ─────────────
//
// These build a real git repo (the dirty worktree fixture) and a second
// repo registered as a submodule, then prove DeriveWorkflowCommitPathsFromRepo
// returns the spec-mandated set against real git output.

func TestDeriveWorkflowCommitPathsFromRepo_DirtyWorktreeFixture(t *testing.T) {
	repo := initWorkflowTestRepo(t)
	// Add managed-root dirty content + noise outside the surface.
	mustWrite(t, filepath.Join(repo, ".agents/workflow/plans/p1/TASKS.yaml"),
		"schema_version: 1\nplan_id: p1\ntasks: []\n")
	mustWrite(t, filepath.Join(repo, ".agents/history/p0/impl-results.md"),
		"# done\n")
	// Pre-existing untracked file inside a dir outside surface: with
	// --untracked-files=all git emits the file path, not the dir. The
	// derivation must drop it because it is neither on the surface nor in
	// a managed root.
	mustMkdirAll(t, filepath.Join(repo, "tmp/scratch"))
	mustWrite(t, filepath.Join(repo, "tmp/scratch/log.txt"), "noise\n")
	// Pre-existing untracked file at the repo root, not on surface — also
	// excluded by the surface intersection rule.
	mustWrite(t, filepath.Join(repo, "OTHER.md"), "noise\n")

	// Mutation surface: only the two managed-root files we just touched.
	// (Note initWorkflowTestRepo also dirtied README.md — that must NOT
	// appear in the path set because it is not on the surface and not
	// under a managed root.)
	surface := []string{
		".agents/workflow/plans/p1/TASKS.yaml",
		".agents/history/p0/impl-results.md",
	}

	got, err := DeriveWorkflowCommitPathsFromRepo(repo, surface)
	if err != nil {
		t.Fatalf("DeriveWorkflowCommitPathsFromRepo: %v", err)
	}

	want := []string{
		".agents/history/p0/impl-results.md",
		".agents/workflow/plans/p1/TASKS.yaml",
	}
	if !reflect.DeepEqual(got.Paths, want) {
		t.Fatalf("dirty-worktree path set mismatch:\n got=%v\nwant=%v", got.Paths, want)
	}
	// README.md was dirtied by initWorkflowTestRepo but is not on the
	// surface and not under a managed root → must be absent.
	for _, p := range got.Paths {
		if p == "README.md" {
			t.Fatalf("README.md leaked into staged set: %v", got.Paths)
		}
	}
	// Sanity: tmp/scratch/log.txt and OTHER.md are real untracked files
	// outside the managed roots and outside the surface — they MUST NOT
	// leak into the staged set (the surface intersection rule).
	for _, p := range got.Paths {
		if p == "tmp/scratch/log.txt" || p == "OTHER.md" {
			t.Fatalf("out-of-surface untracked path leaked: %v", got.Paths)
		}
	}
}

// TestDeriveWorkflowCommitPathsFromRepo_UntrackedDirEnumeration drives the
// "pre-existing-untracked dir" exclusion rule against a real git status that
// actually emits a directory record. With `--untracked-files=normal` (the git
// default when callers shell to status without the all flag), git collapses
// untracked dirs to a single trailing-slash record — and our derivation must
// drop them. This is the exact spec scenario.
func TestDeriveWorkflowCommitPathsFromRepo_UntrackedDirEnumeration(t *testing.T) {
	repo := initWorkflowTestRepo(t)
	// A brand-new untracked directory with content. With -unormal git
	// would report "tmp/"; with our production -uall it reports each file
	// individually. Either way, the derivation must NOT stage anything
	// from outside the managed roots.
	mustMkdirAll(t, filepath.Join(repo, "tmp/sub"))
	mustWrite(t, filepath.Join(repo, "tmp/sub/x.txt"), "x\n")

	// Empty mutation surface — nothing is allowed except managed roots.
	got, err := DeriveWorkflowCommitPathsFromRepo(repo, nil)
	if err != nil {
		t.Fatalf("DeriveWorkflowCommitPathsFromRepo: %v", err)
	}

	for _, p := range got.Paths {
		if strings.HasPrefix(p, "tmp/") || p == "tmp/" {
			t.Fatalf("tmp/ contents leaked into staged set: %v", got.Paths)
		}
		// README.md is dirtied by initWorkflowTestRepo; also must not leak.
		if p == "README.md" {
			t.Fatalf("README.md leaked into staged set: %v", got.Paths)
		}
	}
}

func TestDeriveWorkflowCommitPathsFromRepo_SubmodulePointerFixture(t *testing.T) {
	// Outer repo with managed-root content + an inner repo registered as a
	// submodule. The submodule pointer line MUST be excluded — this is the
	// mandatory rule the spec calls out by name.
	outer := initWorkflowTestRepo(t)
	inner := t.TempDir()
	gitInRepo(t, inner, "init", "-q")
	gitInRepo(t, inner, "config", "user.email", "test@example.com")
	gitInRepo(t, inner, "config", "user.name", "Test")
	mustWrite(t, filepath.Join(inner, "README.md"), "inner\n")
	gitInRepo(t, inner, "add", "README.md")
	gitInRepoEnv(t, inner, []string{
		"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com",
	}, "commit", "-q", "-m", "inner")

	// Register inner as a submodule of outer at path "subm". Newer git
	// versions refuse local-path submodule adds without the protocol
	// override — set it explicitly so the test is portable.
	gitInRepoEnv(t, outer, nil,
		"-c", "protocol.file.allow=always",
		"submodule", "add", "-q", inner, "subm")

	// Now also touch a managed-root file so we have a real positive entry.
	mustWrite(t, filepath.Join(outer, ".agents/workflow/plans/p1/TASKS.yaml"),
		"schema_version: 1\nplan_id: p1\ntasks: []\n")

	surface := []string{".agents/workflow/plans/p1/TASKS.yaml"}

	got, err := DeriveWorkflowCommitPathsFromRepo(outer, surface)
	if err != nil {
		t.Fatalf("DeriveWorkflowCommitPathsFromRepo: %v", err)
	}

	// Submodule path "subm" must NOT appear in Paths.
	for _, p := range got.Paths {
		if p == "subm" || strings.HasPrefix(p, "subm/") {
			t.Fatalf("submodule pointer leaked into staged set: %v", got.Paths)
		}
	}
	// And it must appear in the diagnostic excludes — proves the
	// porcelain v2 sub-field detection actually fired.
	if !containsString(got.ExcludedSubmodule, "subm") {
		t.Fatalf("expected 'subm' in submodule excludes, got %v",
			got.ExcludedSubmodule)
	}
	// And the managed-root file IS present.
	if !containsString(got.Paths, ".agents/workflow/plans/p1/TASKS.yaml") {
		t.Fatalf("managed-root file missing from staged set: %v", got.Paths)
	}
}

// ── Local test helpers (file-scoped to keep testutil_test.go untouched) ─────

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
}

func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

// gitInRepo runs git with -C in a test repo. Mirrors initWorkflowTestRepoWithCommit's
// execabs+env pattern but factored for reuse with explicit env control.
func gitInRepo(t *testing.T, repo string, args ...string) {
	t.Helper()
	gitInRepoEnv(t, repo, nil, args...)
}

func gitInRepoEnv(t *testing.T, repo string, extraEnv []string, args ...string) {
	t.Helper()
	cmd := execabs.Command("git", append([]string{"-C", repo}, args...)...)
	cmd.Env = append(os.Environ(), extraEnv...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
	}
}
