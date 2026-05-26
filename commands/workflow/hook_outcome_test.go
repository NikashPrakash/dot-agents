package workflow

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.yaml.in/yaml/v3"
)

// ── Schema compile/validation tests ──────────────────────────────────────────

func TestCompiledWorkflowHookOutcomeSchema(t *testing.T) {
	sch, err := compiledWorkflowHookOutcomeSchema(stdSchemaCompiler{})
	if err != nil {
		t.Fatalf("compiledWorkflowHookOutcomeSchema: %v", err)
	}
	if sch == nil {
		t.Error("expected non-nil schema")
	}
}

func newValidHookOutcomeRecord() HookOutcomeRecord {
	return HookOutcomeRecord{
		SchemaVersion:        1,
		SentinelID:           "iteration-close-r1",
		Skill:                "iteration-close",
		LifecyclePoint:       "stop",
		InterventionClass:    "remediate_at_stop",
		Result:               "remediate",
		RuleID:               "iteration-close.R1.1",
		Platform:             "claude",
		TS:                   "2026-05-26T12:00:00Z",
		ArchivedSentinelPath: ".agents/history/p/hook-sentinels/2026-05-26/iteration-close-r1.json",
		CorrelationID:        "iteration-close-r1",
	}
}

func newValidHookOutcomeSidecar() *HookOutcomeSidecar {
	return &HookOutcomeSidecar{
		SchemaVersion: 1,
		Records:       []HookOutcomeRecord{newValidHookOutcomeRecord()},
	}
}

func TestValidateHookOutcomeSidecar_Valid(t *testing.T) {
	if err := validateHookOutcomeSidecar(newValidHookOutcomeSidecar()); err != nil {
		t.Fatalf("valid sidecar rejected: %v", err)
	}
}

func TestValidateHookOutcomeSidecar_Nil(t *testing.T) {
	if err := validateHookOutcomeSidecar(nil); err == nil {
		t.Error("expected error for nil sidecar")
	}
}

func TestValidateHookOutcomeSidecar_BadSkillEnum(t *testing.T) {
	sc := newValidHookOutcomeSidecar()
	sc.Records[0].Skill = "rogue"
	if err := validateHookOutcomeSidecar(sc); err == nil {
		t.Error("expected schema rejection for unknown skill enum value")
	}
}

func TestValidateHookOutcomeSidecar_BadLifecyclePointEnum(t *testing.T) {
	sc := newValidHookOutcomeSidecar()
	sc.Records[0].LifecyclePoint = "made-up"
	if err := validateHookOutcomeSidecar(sc); err == nil {
		t.Error("expected schema rejection for unknown lifecycle_point enum value")
	}
}

func TestValidateHookOutcomeSidecar_BadInterventionClassEnum(t *testing.T) {
	sc := newValidHookOutcomeSidecar()
	sc.Records[0].InterventionClass = "wat"
	if err := validateHookOutcomeSidecar(sc); err == nil {
		t.Error("expected schema rejection for unknown intervention_class enum value")
	}
}

func TestValidateHookOutcomeSidecar_BadResultEnum(t *testing.T) {
	sc := newValidHookOutcomeSidecar()
	sc.Records[0].Result = "OK"
	if err := validateHookOutcomeSidecar(sc); err == nil {
		t.Error("expected schema rejection for unknown result enum value")
	}
}

func TestValidateHookOutcomeSidecar_BadPlatformEnum(t *testing.T) {
	sc := newValidHookOutcomeSidecar()
	sc.Records[0].Platform = "gemini"
	if err := validateHookOutcomeSidecar(sc); err == nil {
		t.Error("expected schema rejection for unknown platform enum value")
	}
}

func TestValidateHookOutcomeSidecar_BadRuleIDPattern(t *testing.T) {
	sc := newValidHookOutcomeSidecar()
	sc.Records[0].RuleID = "Has Spaces"
	if err := validateHookOutcomeSidecar(sc); err == nil {
		t.Error("expected schema rejection for rule_id pattern mismatch")
	}
}

func TestValidateHookOutcomeSidecar_BadTSFormat(t *testing.T) {
	sc := newValidHookOutcomeSidecar()
	sc.Records[0].TS = "yesterday"
	if err := validateHookOutcomeSidecar(sc); err == nil {
		t.Error("expected schema rejection for malformed ts")
	}
}

func TestValidateHookOutcomeSidecar_MissingRequiredField(t *testing.T) {
	sc := newValidHookOutcomeSidecar()
	sc.Records[0].SentinelID = ""
	if err := validateHookOutcomeSidecar(sc); err == nil {
		t.Error("expected schema rejection for empty sentinel_id (minLength 1)")
	}
}

// ── Enum / pattern guards ────────────────────────────────────────────────────

func TestValidHookOutcomeSkill(t *testing.T) {
	for _, s := range []string{
		"iteration-close", "isp", "loop-worker",
		"orchestrator-session-start", "delegation-lifecycle",
	} {
		if !validHookOutcomeSkill(s) {
			t.Errorf("expected %q valid", s)
		}
	}
	for _, s := range []string{"", "ISP", "loop_worker", "other"} {
		if validHookOutcomeSkill(s) {
			t.Errorf("expected %q invalid", s)
		}
	}
}

func TestValidHookOutcomeLifecyclePoint(t *testing.T) {
	for _, s := range []string{"pre_tool_use", "stop", "subagent_stop", "subagent_start", "pre_compact", "post_tool_use", "post_tool_use_failure"} {
		if !validHookOutcomeLifecyclePoint(s) {
			t.Errorf("expected %q valid", s)
		}
	}
	if validHookOutcomeLifecyclePoint("PreToolUse") {
		t.Error("expected camelCase variant to be invalid")
	}
}

func TestValidHookOutcomeInterventionClass(t *testing.T) {
	for _, s := range []string{"prevent_before_action", "remediate_at_stop", "continuity_advice", "observe_tool_result"} {
		if !validHookOutcomeInterventionClass(s) {
			t.Errorf("expected %q valid", s)
		}
	}
	if validHookOutcomeInterventionClass("remediate") {
		t.Error("expected bare result enum value to be invalid as intervention class")
	}
}

func TestValidHookOutcomeResult(t *testing.T) {
	for _, s := range []string{"allow", "advise", "remediate"} {
		if !validHookOutcomeResult(s) {
			t.Errorf("expected %q valid", s)
		}
	}
	if validHookOutcomeResult("ok") {
		t.Error("expected ok to be invalid")
	}
}

func TestValidHookOutcomePlatform(t *testing.T) {
	for _, s := range []string{"claude", "codex", "copilot", "cursor"} {
		if !validHookOutcomePlatform(s) {
			t.Errorf("expected %q valid", s)
		}
	}
	if validHookOutcomePlatform("Claude") {
		t.Error("expected CamelCase variant to be invalid")
	}
}

func TestValidHookOutcomeRuleID(t *testing.T) {
	for _, s := range []string{
		"iteration-close.R1.1",
		"loop-worker.R3.1",
		"orchestrator-handoff.R3.3",
		"a.b.c.d",
	} {
		if !validHookOutcomeRuleID(s) {
			t.Errorf("expected %q valid", s)
		}
	}
	for _, s := range []string{
		"",
		"NoDot",
		"Spaces In Name",
		"Uppercase.Prefix.Bad",
		".leading.dot",
		"trailing.dot.",
	} {
		if validHookOutcomeRuleID(s) {
			t.Errorf("expected %q invalid", s)
		}
	}
}

// ── buildHookOutcomeRecord tests ─────────────────────────────────────────────

func validHookOutcomeWriteInputs() hookOutcomeWriteInputs {
	return hookOutcomeWriteInputs{
		SentinelID:           "iteration-close-r1",
		Skill:                "iteration-close",
		LifecyclePoint:       "stop",
		InterventionClass:    "remediate_at_stop",
		Result:               "remediate",
		RuleID:               "iteration-close.R1.1",
		Platform:             "claude",
		CorrelationID:        "",
		ArchivedSentinelPath: "",
		TS:                   "",
	}
}

func TestBuildHookOutcomeRecord_Defaults(t *testing.T) {
	t.Cleanup(func() { hookOutcomeNow = func() time.Time { return time.Now() } })
	fixed := time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)
	hookOutcomeNow = func() time.Time { return fixed }
	rec, err := buildHookOutcomeRecord(validHookOutcomeWriteInputs())
	if err != nil {
		t.Fatalf("buildHookOutcomeRecord: %v", err)
	}
	if rec.SchemaVersion != HookOutcomeSchemaVersion {
		t.Errorf("SchemaVersion = %d, want %d", rec.SchemaVersion, HookOutcomeSchemaVersion)
	}
	if rec.CorrelationID != rec.SentinelID {
		t.Errorf("CorrelationID = %q, want default to sentinel_id %q", rec.CorrelationID, rec.SentinelID)
	}
	if rec.TS != "2026-05-26T12:00:00Z" {
		t.Errorf("TS = %q, want default to now()=2026-05-26T12:00:00Z", rec.TS)
	}
}

func TestBuildHookOutcomeRecord_MissingSentinelID(t *testing.T) {
	in := validHookOutcomeWriteInputs()
	in.SentinelID = "   "
	_, err := buildHookOutcomeRecord(in)
	if err == nil || !strings.Contains(err.Error(), "sentinel-id is required") {
		t.Errorf("expected sentinel-id error, got %v", err)
	}
}

func TestBuildHookOutcomeRecord_BadSkill(t *testing.T) {
	in := validHookOutcomeWriteInputs()
	in.Skill = "made-up"
	_, err := buildHookOutcomeRecord(in)
	if err == nil || !strings.Contains(err.Error(), "--skill must be one of") {
		t.Errorf("expected --skill enum error, got %v", err)
	}
}

func TestBuildHookOutcomeRecord_BadLifecyclePoint(t *testing.T) {
	in := validHookOutcomeWriteInputs()
	in.LifecyclePoint = "PreToolUse" // camelCase variant; schema is snake_case
	_, err := buildHookOutcomeRecord(in)
	if err == nil || !strings.Contains(err.Error(), "--lifecycle-point must be one of") {
		t.Errorf("expected --lifecycle-point enum error, got %v", err)
	}
}

func TestBuildHookOutcomeRecord_BadInterventionClass(t *testing.T) {
	in := validHookOutcomeWriteInputs()
	in.InterventionClass = "block"
	_, err := buildHookOutcomeRecord(in)
	if err == nil || !strings.Contains(err.Error(), "--intervention-class must be one of") {
		t.Errorf("expected --intervention-class enum error, got %v", err)
	}
}

func TestBuildHookOutcomeRecord_BadResult(t *testing.T) {
	in := validHookOutcomeWriteInputs()
	in.Result = "deny"
	_, err := buildHookOutcomeRecord(in)
	if err == nil || !strings.Contains(err.Error(), "--result must be one of") {
		t.Errorf("expected --result enum error, got %v", err)
	}
}

func TestBuildHookOutcomeRecord_BadRuleID(t *testing.T) {
	in := validHookOutcomeWriteInputs()
	in.RuleID = "NoSegment"
	_, err := buildHookOutcomeRecord(in)
	if err == nil || !strings.Contains(err.Error(), "--rule-id") {
		t.Errorf("expected --rule-id pattern error, got %v", err)
	}
}

func TestBuildHookOutcomeRecord_BadPlatform(t *testing.T) {
	in := validHookOutcomeWriteInputs()
	in.Platform = "Gemini"
	_, err := buildHookOutcomeRecord(in)
	if err == nil || !strings.Contains(err.Error(), "--platform must be one of") {
		t.Errorf("expected --platform enum error, got %v", err)
	}
}

func TestBuildHookOutcomeRecord_CustomTSAndCorrelation(t *testing.T) {
	in := validHookOutcomeWriteInputs()
	in.TS = "2026-05-26T13:00:00Z"
	in.CorrelationID = "merged-intent-1"
	rec, err := buildHookOutcomeRecord(in)
	if err != nil {
		t.Fatalf("buildHookOutcomeRecord: %v", err)
	}
	if rec.TS != in.TS {
		t.Errorf("TS = %q, want preserved %q", rec.TS, in.TS)
	}
	if rec.CorrelationID != in.CorrelationID {
		t.Errorf("CorrelationID = %q, want %q (caller-supplied wins)", rec.CorrelationID, in.CorrelationID)
	}
}

// ── Idempotency-key tests ────────────────────────────────────────────────────

func TestHookOutcomeIdempotencyKeyMatches_AllFour(t *testing.T) {
	a := newValidHookOutcomeRecord()
	b := newValidHookOutcomeRecord()
	b.TS = "2099-01-01T00:00:00Z" // outside the key
	b.Result = "allow"            // outside the key
	if !hookOutcomeIdempotencyKeyMatches(a, b) {
		t.Error("expected match on the four-tuple key (sentinel_id, rule_id, lifecycle_point, intervention_class)")
	}
}

func TestHookOutcomeIdempotencyKeyMatches_DiffSentinel(t *testing.T) {
	a := newValidHookOutcomeRecord()
	b := newValidHookOutcomeRecord()
	b.SentinelID = "iteration-close-r2"
	if hookOutcomeIdempotencyKeyMatches(a, b) {
		t.Error("expected no match when sentinel_id differs")
	}
}

func TestHookOutcomeIdempotencyKeyMatches_DiffRule(t *testing.T) {
	a := newValidHookOutcomeRecord()
	b := newValidHookOutcomeRecord()
	b.RuleID = "iteration-close.R1.2"
	if hookOutcomeIdempotencyKeyMatches(a, b) {
		t.Error("expected no match when rule_id differs")
	}
}

func TestHookOutcomeIdempotencyKeyMatches_DiffLifecycle(t *testing.T) {
	a := newValidHookOutcomeRecord()
	b := newValidHookOutcomeRecord()
	b.LifecyclePoint = "subagent_stop"
	if hookOutcomeIdempotencyKeyMatches(a, b) {
		t.Error("expected no match when lifecycle_point differs")
	}
}

func TestHookOutcomeIdempotencyKeyMatches_DiffIntervention(t *testing.T) {
	a := newValidHookOutcomeRecord()
	b := newValidHookOutcomeRecord()
	b.InterventionClass = "prevent_before_action"
	if hookOutcomeIdempotencyKeyMatches(a, b) {
		t.Error("expected no match when intervention_class differs")
	}
}

// ── resolveActiveIterationN tests ────────────────────────────────────────────

func TestResolveActiveIterationN_NoIterDir(t *testing.T) {
	dir := t.TempDir()
	n, active, err := resolveActiveIterationN(dir)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if active {
		t.Error("expected active=false when iter-log dir is missing")
	}
	if n != 0 {
		t.Errorf("expected n=0 when no iteration, got %d", n)
	}
}

func TestResolveActiveIterationN_EmptyIterDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".agents", "active", "iteration-log"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	_, active, err := resolveActiveIterationN(dir)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if active {
		t.Error("expected active=false when iter-log dir exists but is empty")
	}
}

func TestResolveActiveIterationN_PicksMax(t *testing.T) {
	dir := t.TempDir()
	iterDir := filepath.Join(dir, ".agents", "active", "iteration-log")
	if err := os.MkdirAll(iterDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	for _, name := range []string{"iter-1.yaml", "iter-2.yaml", "iter-3.yaml", "iter-3.score.yaml", "not-an-iter.yaml"} {
		if err := os.WriteFile(filepath.Join(iterDir, name), []byte("schema_version: 2\n"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	n, active, err := resolveActiveIterationN(dir)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !active {
		t.Fatal("expected active=true")
	}
	if n != 3 {
		t.Errorf("expected n=3 (max of iter-1, iter-2, iter-3), got %d (score sidecar must not count)", n)
	}
}

func TestResolveActiveIterationN_ReadDirError(t *testing.T) {
	t.Cleanup(func() { hookOutcomeReadDir = os.ReadDir })
	hookOutcomeReadDir = func(string) ([]os.DirEntry, error) {
		return nil, errors.New("synthetic readdir fault")
	}
	_, _, err := resolveActiveIterationN(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "read iteration-log dir") {
		t.Errorf("expected wrapped readdir error, got %v", err)
	}
}

// ── loadHookOutcomeSidecar tests ─────────────────────────────────────────────

func TestLoadHookOutcomeSidecar_MissingFileYieldsEmpty(t *testing.T) {
	sc, err := loadHookOutcomeSidecar(filepath.Join(t.TempDir(), "nope.yaml"))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if sc == nil {
		t.Fatal("expected non-nil empty sidecar")
	}
	if sc.SchemaVersion != HookOutcomeSchemaVersion {
		t.Errorf("SchemaVersion = %d, want %d", sc.SchemaVersion, HookOutcomeSchemaVersion)
	}
	if len(sc.Records) != 0 {
		t.Errorf("expected empty records slice, got %d", len(sc.Records))
	}
}

func TestLoadHookOutcomeSidecar_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "iter-1.hook-outcomes.yaml")
	orig := newValidHookOutcomeSidecar()
	body, err := yaml.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := loadHookOutcomeSidecar(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(got.Records) != 1 || got.Records[0].SentinelID != orig.Records[0].SentinelID {
		t.Errorf("round-trip records mismatch: got %+v", got.Records)
	}
}

func TestLoadHookOutcomeSidecar_BadYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "iter-1.hook-outcomes.yaml")
	if err := os.WriteFile(path, []byte("not: : valid: : yaml"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := loadHookOutcomeSidecar(path); err == nil {
		t.Error("expected parse error for malformed YAML")
	}
}

func TestLoadHookOutcomeSidecar_MissingSchemaVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "iter-1.hook-outcomes.yaml")
	if err := os.WriteFile(path, []byte("records: []\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := loadHookOutcomeSidecar(path)
	if err == nil || !strings.Contains(err.Error(), "missing schema_version") {
		t.Errorf("expected missing schema_version error, got %v", err)
	}
}

func TestLoadHookOutcomeSidecar_UnsupportedSchemaVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "iter-1.hook-outcomes.yaml")
	if err := os.WriteFile(path, []byte("schema_version: 999\nrecords: []\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := loadHookOutcomeSidecar(path)
	if err == nil || !strings.Contains(err.Error(), "unsupported schema_version") {
		t.Errorf("expected unsupported schema_version error, got %v", err)
	}
}

// ── appendHookOutcome end-to-end tests ───────────────────────────────────────

func setupProjectWithIter(t *testing.T, n int) string {
	t.Helper()
	dir := t.TempDir()
	iterDir := filepath.Join(dir, ".agents", "active", "iteration-log")
	if err := os.MkdirAll(iterDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(iterDir, fmt.Sprintf("iter-%d.yaml", n)), []byte("schema_version: 2\niteration: 1\n"), 0o644); err != nil {
		t.Fatalf("write iter file: %v", err)
	}
	return dir
}

func TestAppendHookOutcome_WritesNewSidecar(t *testing.T) {
	dir := setupProjectWithIter(t, 1)
	rec := newValidHookOutcomeRecord()
	res, err := appendHookOutcome(dir, rec)
	if err != nil {
		t.Fatalf("appendHookOutcome: %v", err)
	}
	if res.Status != "written" {
		t.Errorf("Status = %q, want written", res.Status)
	}
	if res.Iteration != 1 {
		t.Errorf("Iteration = %d, want 1", res.Iteration)
	}
	wantPath := filepath.Join(dir, ".agents", "active", "iteration-log", "iter-1.hook-outcomes.yaml")
	if res.Path != wantPath {
		t.Errorf("Path = %q, want %q", res.Path, wantPath)
	}
	body, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatalf("read sidecar: %v", err)
	}
	if !strings.Contains(string(body), "iteration-close.R1.1") {
		t.Error("sidecar missing the appended rule_id")
	}
	if !strings.HasPrefix(string(body), "# yaml-language-server:") {
		t.Error("sidecar missing yaml-language-server header")
	}
}

func TestAppendHookOutcome_NoActiveIterationIsSilentExit(t *testing.T) {
	// Empty project: no iter-log dir at all.
	dir := t.TempDir()
	res, err := appendHookOutcome(dir, newValidHookOutcomeRecord())
	if err != nil {
		t.Fatalf("appendHookOutcome: %v", err)
	}
	if res.Status != "no-active-iteration" {
		t.Errorf("Status = %q, want no-active-iteration", res.Status)
	}
	if res.Path != "" {
		t.Errorf("expected empty Path on no-active-iteration, got %q", res.Path)
	}
}

func TestAppendHookOutcome_IdempotentDuplicate(t *testing.T) {
	dir := setupProjectWithIter(t, 1)
	rec := newValidHookOutcomeRecord()
	if _, err := appendHookOutcome(dir, rec); err != nil {
		t.Fatalf("first append: %v", err)
	}
	// Second write with same idempotency-key tuple but a different ts/result:
	// must be a no-op duplicate.
	dup := rec
	dup.TS = "2099-01-01T00:00:00Z"
	dup.Result = "allow"
	res, err := appendHookOutcome(dir, dup)
	if err != nil {
		t.Fatalf("second append: %v", err)
	}
	if res.Status != "duplicate" {
		t.Errorf("Status = %q, want duplicate", res.Status)
	}
	// Sidecar must still hold exactly one record, with the ORIGINAL ts/result.
	sc, err := loadHookOutcomeSidecar(res.Path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(sc.Records) != 1 {
		t.Fatalf("expected 1 record after duplicate, got %d", len(sc.Records))
	}
	if sc.Records[0].TS == "2099-01-01T00:00:00Z" {
		t.Error("expected original ts preserved on duplicate write (append-only invariant)")
	}
	if sc.Records[0].Result != "remediate" {
		t.Errorf("expected original result preserved, got %q", sc.Records[0].Result)
	}
}

func TestAppendHookOutcome_DifferentRuleIDAppends(t *testing.T) {
	dir := setupProjectWithIter(t, 1)
	first := newValidHookOutcomeRecord()
	if _, err := appendHookOutcome(dir, first); err != nil {
		t.Fatalf("first: %v", err)
	}
	second := newValidHookOutcomeRecord()
	second.RuleID = "iteration-close.R1.2"
	res, err := appendHookOutcome(dir, second)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if res.Status != "written" {
		t.Errorf("Status = %q, want written (different rule_id = different key)", res.Status)
	}
	sc, err := loadHookOutcomeSidecar(res.Path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(sc.Records) != 2 {
		t.Errorf("expected 2 records, got %d", len(sc.Records))
	}
}

func TestAppendHookOutcome_TargetsLatestIteration(t *testing.T) {
	dir := setupProjectWithIter(t, 1)
	iterDir := filepath.Join(dir, ".agents", "active", "iteration-log")
	// Add iter-2 and iter-3 so the latest is 3.
	for _, n := range []int{2, 3} {
		if err := os.WriteFile(filepath.Join(iterDir, fmt.Sprintf("iter-%d.yaml", n)), []byte("schema_version: 2\n"), 0o644); err != nil {
			t.Fatalf("write iter-%d: %v", n, err)
		}
	}
	res, err := appendHookOutcome(dir, newValidHookOutcomeRecord())
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	if res.Iteration != 3 {
		t.Errorf("Iteration = %d, want 3 (latest)", res.Iteration)
	}
	if !strings.HasSuffix(res.Path, "iter-3.hook-outcomes.yaml") {
		t.Errorf("Path %q does not target latest iter-3 sidecar", res.Path)
	}
}

func TestAppendHookOutcome_RenameError(t *testing.T) {
	t.Cleanup(func() { hookOutcomeRename = os.Rename })
	dir := setupProjectWithIter(t, 1)
	hookOutcomeRename = func(string, string) error { return errors.New("synthetic rename fault") }
	_, err := appendHookOutcome(dir, newValidHookOutcomeRecord())
	if err == nil || !strings.Contains(err.Error(), "publish hook outcome sidecar") {
		t.Errorf("expected wrapped rename error, got %v", err)
	}
}

func TestAppendHookOutcome_ValidationRejectsDisallowedRecordShape(t *testing.T) {
	dir := setupProjectWithIter(t, 1)
	rec := newValidHookOutcomeRecord()
	rec.Skill = "rogue-skill" // schema enum rejection
	_, err := appendHookOutcome(dir, rec)
	if err == nil || !strings.Contains(err.Error(), "workflow-hook-outcome.schema.json") {
		t.Errorf("expected schema rejection wrapped by validator, got %v", err)
	}
}

// ── hookOutcomeRecordKey readback ────────────────────────────────────────────

func TestHookOutcomeRecordKey(t *testing.T) {
	rec := newValidHookOutcomeRecord()
	got := hookOutcomeRecordKey(rec)
	want := "iteration-close-r1|iteration-close.R1.1|stop|remediate_at_stop"
	if got != want {
		t.Errorf("hookOutcomeRecordKey = %q, want %q", got, want)
	}
}

// ── runHookOutcomeWrite end-to-end (CLI handler) tests ──────────────────────

func newValidHookOutcomeWriteInputs() hookOutcomeWriteInputs {
	return hookOutcomeWriteInputs{
		SentinelID:        "iteration-close-r1",
		Skill:             "iteration-close",
		LifecyclePoint:    "stop",
		InterventionClass: "remediate_at_stop",
		Result:            "remediate",
		RuleID:            "iteration-close.R1.1",
		Platform:          "claude",
		TS:                "2026-05-26T12:00:00Z",
		CorrelationID:     "iteration-close-r1",
	}
}

func withHookOutcomeProject(t *testing.T, path string) {
	t.Helper()
	prior := hookOutcomeResolveProject
	hookOutcomeResolveProject = func() (workflowProjectRef, error) {
		return workflowProjectRef{Name: "p", Path: path}, nil
	}
	t.Cleanup(func() { hookOutcomeResolveProject = prior })
}

func captureStdoutString(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	old := os.Stdout
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = old
	var buf strings.Builder
	b := make([]byte, 4096)
	for {
		n, err := r.Read(b)
		if n > 0 {
			buf.Write(b[:n])
		}
		if err != nil {
			break
		}
	}
	return buf.String()
}

func TestRunHookOutcomeWrite_WrittenText(t *testing.T) {
	dir := setupProjectWithIter(t, 3)
	withHookOutcomeProject(t, dir)
	out := captureStdoutString(t, func() {
		if err := runHookOutcomeWrite(newValidHookOutcomeWriteInputs()); err != nil {
			t.Fatalf("runHookOutcomeWrite: %v", err)
		}
	})
	if !strings.Contains(out, "wrote hook outcome") || !strings.Contains(out, "iter-3") {
		t.Errorf("text output missing written marker / iter-3: %q", out)
	}
}

func TestRunHookOutcomeWrite_WrittenJSON(t *testing.T) {
	dir := setupProjectWithIter(t, 2)
	withHookOutcomeProject(t, dir)
	prior := deps.Flags.JSON
	deps.Flags.JSON = func() bool { return true }
	t.Cleanup(func() { deps.Flags.JSON = prior })
	out := captureStdoutString(t, func() {
		if err := runHookOutcomeWrite(newValidHookOutcomeWriteInputs()); err != nil {
			t.Fatalf("runHookOutcomeWrite: %v", err)
		}
	})
	if !strings.Contains(out, `"status": "written"`) || !strings.Contains(out, `"iteration": 2`) {
		t.Errorf("JSON output missing written/iteration field: %q", out)
	}
}

func TestRunHookOutcomeWrite_DuplicateText(t *testing.T) {
	dir := setupProjectWithIter(t, 1)
	withHookOutcomeProject(t, dir)
	if err := runHookOutcomeWrite(newValidHookOutcomeWriteInputs()); err != nil {
		t.Fatalf("first write: %v", err)
	}
	out := captureStdoutString(t, func() {
		if err := runHookOutcomeWrite(newValidHookOutcomeWriteInputs()); err != nil {
			t.Fatalf("second write: %v", err)
		}
	})
	if !strings.Contains(out, "duplicate hook outcome") {
		t.Errorf("expected duplicate marker, got %q", out)
	}
}

func TestRunHookOutcomeWrite_DuplicateJSON(t *testing.T) {
	dir := setupProjectWithIter(t, 1)
	withHookOutcomeProject(t, dir)
	if err := runHookOutcomeWrite(newValidHookOutcomeWriteInputs()); err != nil {
		t.Fatalf("first: %v", err)
	}
	prior := deps.Flags.JSON
	deps.Flags.JSON = func() bool { return true }
	t.Cleanup(func() { deps.Flags.JSON = prior })
	out := captureStdoutString(t, func() {
		if err := runHookOutcomeWrite(newValidHookOutcomeWriteInputs()); err != nil {
			t.Fatalf("second: %v", err)
		}
	})
	if !strings.Contains(out, `"status": "duplicate"`) {
		t.Errorf("expected JSON duplicate status, got %q", out)
	}
}

func TestRunHookOutcomeWrite_NoActiveIterationText(t *testing.T) {
	// Empty project: no iter-log dir → stderr advisory, exit 0, no stdout body.
	dir := t.TempDir()
	withHookOutcomeProject(t, dir)
	stderr := captureStderr(t, func() {
		if err := runHookOutcomeWrite(newValidHookOutcomeWriteInputs()); err != nil {
			t.Fatalf("runHookOutcomeWrite: %v", err)
		}
	})
	if !strings.Contains(stderr, "no active iteration") {
		t.Errorf("expected no-active-iteration stderr advisory, got %q", stderr)
	}
}

func TestRunHookOutcomeWrite_NoActiveIterationJSON(t *testing.T) {
	dir := t.TempDir()
	withHookOutcomeProject(t, dir)
	prior := deps.Flags.JSON
	deps.Flags.JSON = func() bool { return true }
	t.Cleanup(func() { deps.Flags.JSON = prior })
	out := captureStdoutString(t, func() {
		if err := runHookOutcomeWrite(newValidHookOutcomeWriteInputs()); err != nil {
			t.Fatalf("runHookOutcomeWrite: %v", err)
		}
	})
	if !strings.Contains(out, `"status": "no-active-iteration"`) {
		t.Errorf("expected JSON no-active-iteration status, got %q", out)
	}
}

func TestRunHookOutcomeWrite_ResolveProjectError(t *testing.T) {
	prior := hookOutcomeResolveProject
	hookOutcomeResolveProject = func() (workflowProjectRef, error) {
		return workflowProjectRef{}, errors.New("boom")
	}
	t.Cleanup(func() { hookOutcomeResolveProject = prior })
	if err := runHookOutcomeWrite(newValidHookOutcomeWriteInputs()); err == nil {
		t.Fatal("expected error from resolve project, got nil")
	}
}

func TestRunHookOutcomeWrite_BadInputBuildError(t *testing.T) {
	dir := setupProjectWithIter(t, 1)
	withHookOutcomeProject(t, dir)
	bad := newValidHookOutcomeWriteInputs()
	bad.Skill = "not-a-real-skill"
	if err := runHookOutcomeWrite(bad); err == nil {
		t.Fatal("expected buildHookOutcomeRecord error, got nil")
	}
}

// ── hookOutcomeSidecarPath helper ────────────────────────────────────────────

func TestHookOutcomeSidecarPath(t *testing.T) {
	got := hookOutcomeSidecarPath("/proj", 7)
	want := filepath.Join("/proj", ".agents", "active", "iteration-log", "iter-7.hook-outcomes.yaml")
	if got != want {
		t.Errorf("hookOutcomeSidecarPath = %q, want %q", got, want)
	}
}
