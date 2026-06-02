package catalog

import (
	"context"
	"database/sql"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSQLRepositoryPersistsReleaseRecordsAcrossInstances(t *testing.T) {
	ctx := context.Background()
	databaseURL := "file:" + filepath.Join(t.TempDir(), "catalog.db")

	repo, err := NewSQLRepository(ctx, databaseURL, SampleData(), SQLRepositoryOptions{
		ArtifactSource: NewStaticArtifactSource(SampleData().Artifacts),
	})
	if err != nil {
		t.Fatalf("create repository: %v", err)
	}

	result, err := repo.CreateReleaseRecord(ctx, "@platform.skill/agent-notes", "0.1.0", ReleaseRecordInput{})
	if err != nil {
		t.Fatalf("create release record: %v", err)
	}
	if !result.Created {
		t.Fatalf("expected created release record, got %+v", result)
	}

	reloaded, err := NewSQLRepository(ctx, databaseURL, SampleData(), SQLRepositoryOptions{
		ArtifactSource: NewStaticArtifactSource(SampleData().Artifacts),
	})
	if err != nil {
		t.Fatalf("reopen repository: %v", err)
	}

	skill, err := reloaded.GetSkill(ctx, "@platform.skill/agent-notes")
	if err != nil {
		t.Fatalf("load persisted skill: %v", err)
	}
	if skill.PackageName != "@platform.skill/agent-notes" {
		t.Fatalf("unexpected persisted package: %s", skill.PackageName)
	}

	db := openCatalogTestDB(t, databaseURL)
	defer db.Close()

	var skillCount int
	if err := db.QueryRowContext(ctx, bindSQL(sqlDriverNameForURL(databaseURL), `SELECT COUNT(*) FROM skills WHERE package_name = ?`), "@platform.skill/agent-notes").Scan(&skillCount); err != nil {
		t.Fatalf("count persisted skills: %v", err)
	}
	if skillCount != 1 {
		t.Fatalf("expected 1 persisted skill row, got %d", skillCount)
	}
	var versionCount int
	if err := db.QueryRowContext(ctx, bindSQL(sqlDriverNameForURL(databaseURL), `SELECT COUNT(*) FROM skill_versions WHERE package_name = ? AND version = ?`), "@platform.skill/agent-notes", "0.1.0").Scan(&versionCount); err != nil {
		t.Fatalf("count persisted versions: %v", err)
	}
	if versionCount != 1 {
		t.Fatalf("expected 1 persisted version row, got %d", versionCount)
	}
}

func TestSQLRepositoryPersistsGovernanceRows(t *testing.T) {
	ctx := context.Background()
	databaseURL := "file:" + filepath.Join(t.TempDir(), "catalog.db")

	repo, err := NewSQLRepository(ctx, databaseURL, SampleData(), SQLRepositoryOptions{
		ArtifactSource: NewStaticArtifactSource(SampleData().Artifacts),
	})
	if err != nil {
		t.Fatalf("create repository: %v", err)
	}

	_, err = repo.ApplyVersionGovernanceChange(ctx, "@platform.skill/pdf-skill", "1.2.0", VersionGovernanceChangeInput{
		Action:                    GovernanceActionYank,
		Reason:                    "security issue",
		ReplacementRecommendation: stringPtr("1.1.0"),
	}, "user_123")
	if err != nil {
		t.Fatalf("apply governance change: %v", err)
	}

	db := openCatalogTestDB(t, databaseURL)
	defer db.Close()

	var (
		targetType string
		version    sql.NullString
		action     string
	)
	if err := db.QueryRowContext(ctx, bindSQL(sqlDriverNameForURL(databaseURL), `
SELECT target_type, version, action
FROM governance_records
WHERE package_name = ? AND changed_by = ?
ORDER BY changed_at DESC
LIMIT 1`), "@platform.skill/pdf-skill", "user_123").Scan(&targetType, &version, &action); err != nil {
		t.Fatalf("load governance row: %v", err)
	}
	if targetType != string(GovernanceTargetVersion) || !version.Valid || version.String != "1.2.0" || action != string(GovernanceActionYank) {
		t.Fatalf("unexpected governance row: type=%s version=%v action=%s", targetType, version, action)
	}
}

func TestSQLStateStorePersistsAggregateRevision(t *testing.T) {
	ctx := context.Background()
	databaseURL := "file:" + filepath.Join(t.TempDir(), "catalog.db")

	db := openCatalogTestDB(t, databaseURL)
	defer db.Close()

	store := NewSQLStateStore(db, sqlDriverNameForURL(databaseURL))
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate store: %v", err)
	}

	aggregate := aggregatesFromSampleData(SampleData())[0]
	aggregate.Revision = 0

	if err := store.PutAggregate(ctx, aggregate, 0); err != nil {
		t.Fatalf("put aggregate: %v", err)
	}
	loaded, err := store.GetAggregate(ctx, aggregate.Skill.Detail.PackageName)
	if err != nil {
		t.Fatalf("get aggregate: %v", err)
	}
	if loaded.Revision == 0 {
		t.Fatalf("expected persisted revision, got %+v", loaded)
	}

	loaded.Skill.Detail.Description = "updated description"
	if err := store.PutAggregate(ctx, loaded, loaded.Revision); err != nil {
		t.Fatalf("update aggregate: %v", err)
	}
	if err := store.PutAggregate(ctx, loaded, loaded.Revision); err != ErrAggregateRevisionConflict {
		t.Fatalf("expected revision conflict, got %v", err)
	}
}

func TestOpenSQLRepositoryDriverSelection(t *testing.T) {
	if driver := sqlDriverNameForURL("file:/tmp/catalog.db"); driver != "sqlite" {
		t.Fatalf("expected sqlite driver, got %s", driver)
	}
	if driver := sqlDriverNameForURL("postgres://user:pass@localhost:5432/catalog"); driver != "pgx" {
		t.Fatalf("expected pgx driver, got %s", driver)
	}
}

func TestSQLRepositoryCanSkipSeeding(t *testing.T) {
	ctx := context.Background()
	databaseURL := "file:" + filepath.Join(t.TempDir(), "catalog.db")

	repo, err := NewSQLRepository(ctx, databaseURL, SampleData(), SQLRepositoryOptions{
		ArtifactSource: NewStaticArtifactSource(SampleData().Artifacts),
		DisableSeed:    true,
	})
	if err != nil {
		t.Fatalf("create repository: %v", err)
	}

	page, err := repo.ListSkills(ctx, SkillQuery{PageSize: 10})
	if err != nil {
		t.Fatalf("list skills: %v", err)
	}
	if len(page.Items) != 0 {
		t.Fatalf("expected empty repository when seed is disabled, got %d skills", len(page.Items))
	}
}

func TestSQLRepositoryPersistsReleaseRecordsOnPostgres(t *testing.T) {
	ctx := context.Background()
	databaseURL := startTempPostgres(t)

	repo, err := NewSQLRepository(ctx, databaseURL, SampleData(), SQLRepositoryOptions{
		ArtifactSource: NewStaticArtifactSource(SampleData().Artifacts),
	})
	if err != nil {
		t.Fatalf("create postgres repository: %v", err)
	}

	result, err := repo.CreateReleaseRecord(ctx, "@platform.skill/agent-notes", "0.1.0", ReleaseRecordInput{})
	if err != nil {
		t.Fatalf("create postgres release record: %v", err)
	}
	if !result.Created {
		t.Fatalf("expected created release record, got %+v", result)
	}

	if _, err := NewSQLRepository(ctx, databaseURL, SampleData(), SQLRepositoryOptions{
		ArtifactSource: NewStaticArtifactSource(SampleData().Artifacts),
	}); err != nil {
		t.Fatalf("reopen postgres repository: %v", err)
	}

	db := openCatalogTestDB(t, databaseURL)
	defer db.Close()

	var count int
	query := bindSQL(sqlDriverNameForURL(databaseURL), `SELECT COUNT(*) FROM skills WHERE package_name = ?`)
	if err := db.QueryRowContext(ctx, query, "@platform.skill/agent-notes").Scan(&count); err != nil {
		t.Fatalf("count postgres persisted skills: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 persisted postgres skill row, got %d", count)
	}
}

func TestTempPostgresUnavailableReason(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		output string
		log    string
		want   bool
	}{
		{
			name:   "matches direct command output",
			output: "FATAL: could not create shared memory segment: Operation not permitted",
			want:   true,
		},
		{
			name:   "matches postgres log when pg_ctl output is generic",
			output: "waiting for server to start.... stopped waiting\npg_ctl: could not start server\nExamine the log output.\n",
			log:    "2026-06-02 13:41:37.690 CST [72585] FATAL:  could not create shared memory segment: Operation not permitted",
			want:   true,
		},
		{
			name:   "ignores unrelated startup errors",
			output: "pg_ctl: could not start server",
			log:    "FATAL: database files are incompatible with server",
			want:   false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tempPostgresUnavailableReason(tt.output, tt.log); got != tt.want {
				t.Fatalf("tempPostgresUnavailableReason(%q, %q) = %t, want %t", tt.output, tt.log, got, tt.want)
			}
		})
	}
}

func openCatalogTestDB(t *testing.T, databaseURL string) *sql.DB {
	t.Helper()

	db, err := sql.Open(sqlDriverNameForURL(databaseURL), databaseURL)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	return db
}

func startTempPostgres(t *testing.T) string {
	t.Helper()

	if databaseURL := os.Getenv("SKILL_REGISTRY_PG_TEST_URL"); databaseURL != "" {
		return databaseURL
	}

	initdbPath, err := exec.LookPath("initdb")
	if err != nil {
		t.Skip("initdb not available")
	}
	pgCtlPath, err := exec.LookPath("pg_ctl")
	if err != nil {
		t.Skip("pg_ctl not available")
	}
	psqlPath, err := exec.LookPath("psql")
	if err != nil {
		t.Skip("psql not available")
	}

	root := t.TempDir()
	dataDir := filepath.Join(root, "pgdata")
	socketDir := filepath.Join(root, "socket")
	logPath := filepath.Join(root, "postgres.log")
	if err := os.MkdirAll(socketDir, 0o755); err != nil {
		t.Fatalf("mkdir postgres socket dir: %v", err)
	}

	run := func(name string, args ...string) {
		t.Helper()
		command := exec.Command(name, args...)
		command.Env = append(os.Environ(), "LC_ALL=C")
		output, err := command.CombinedOutput()
		if err != nil {
			outputText := string(output)
			logText, readErr := os.ReadFile(logPath)
			if readErr != nil && !os.IsNotExist(readErr) {
				t.Fatalf("read postgres log %s: %v", logPath, readErr)
			}
			if tempPostgresUnavailableReason(outputText, string(logText)) {
				t.Skipf("postgres temporary cluster is not available in this environment: %s", outputText)
			}
			t.Fatalf("%s %v failed: %v output=%s", name, args, err, string(output))
		}
	}

	run(
		initdbPath,
		"-D", dataDir,
		"-A", "trust",
		"-U", "postgres",
		"--no-instructions",
		"-c", "dynamic_shared_memory_type=posix",
		"-c", "shared_memory_type=mmap",
	)
	run(pgCtlPath, "-D", dataDir, "-l", logPath, "-o", "-k "+socketDir+" -c listen_addresses='' -c dynamic_shared_memory_type=posix -c shared_memory_type=mmap", "-w", "start")
	t.Cleanup(func() {
		run(pgCtlPath, "-D", dataDir, "-m", "fast", "stop")
	})
	run(psqlPath, "-h", socketDir, "-U", "postgres", "-d", "postgres", "-c", "CREATE DATABASE skill_registry_test")

	return "postgresql://postgres@/skill_registry_test?host=" + url.QueryEscape(socketDir) + "&sslmode=disable"
}

func tempPostgresUnavailableReason(outputText string, logText string) bool {
	combined := outputText + "\n" + logText
	return strings.Contains(combined, "could not create shared memory segment") ||
		strings.Contains(combined, "Operation not permitted")
}
