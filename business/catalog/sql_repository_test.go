package catalog

import (
	"context"
	"database/sql"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestSQLRepositoryListSkillsAppliesSortAndPageInStore(t *testing.T) {
	repo := newSQLRepositoryForTest(t, SampleData())

	page, err := repo.ListSkills(context.Background(), SkillQuery{
		Sort:     "package_name_asc",
		PageSize: 1,
		Cursor:   1,
	})
	if err != nil {
		t.Fatalf("list skills: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("expected exactly one item, got %d", len(page.Items))
	}
	if page.Items[0].PackageName != "@platform.skill/pdf-skill" {
		t.Fatalf("expected second lexicographic package, got %q", page.Items[0].PackageName)
	}
	if page.PageInfo.HasNextPage {
		t.Fatalf("expected second page to be terminal, got page info %+v", page.PageInfo)
	}
}

func TestSQLRepositoryListGovernanceChangesPagesInStore(t *testing.T) {
	repo := newSQLRepositoryForTest(t, SampleData())

	_, err := repo.ApplyVersionGovernanceChange(context.Background(), "@platform.skill/pdf-skill", "1.2.0", VersionGovernanceChangeInput{
		Action: GovernanceActionDeprecate,
		Reason: "legacy",
	}, "operator")
	if err != nil {
		t.Fatalf("seed governance change: %v", err)
	}

	page, err := repo.ListGovernanceChanges(context.Background(), GovernanceQuery{
		PackageName: "@platform.skill/pdf-skill",
		PageSize:    1,
		Cursor:      1,
	})
	if err != nil {
		t.Fatalf("list governance changes: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("expected one governance item, got %d", len(page.Items))
	}
	if page.Items[0].ID != "gov_01hxyank" {
		t.Fatalf("expected paged governance record gov_01hxyank, got %q", page.Items[0].ID)
	}
	if page.Items[0].Version == nil || *page.Items[0].Version != "1.3.0" {
		t.Fatalf("expected paged governance version 1.3.0, got %#v", page.Items[0].Version)
	}
	if page.PageInfo.HasNextPage {
		t.Fatalf("expected second page to be terminal, got page info %+v", page.PageInfo)
	}
}

func TestSQLRepositoryListVersionsPagesWithVersionAwareOrderInStore(t *testing.T) {
	publishedAt := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	data := SampleCatalogData{
		Skills: []skillRecord{
			{
				Detail: SkillDetail{
					SkillSummary: SkillSummary{
						PackageName:         "@platform.skill/version-order",
						DisplayName:         "Version Order",
						Description:         "Covers version paging order",
						SourceType:          SourceTypeImported,
						Status:              SkillStatusActive,
						LatestVersion:       stringPtr("1.10.0"),
						LatestVersionStatus: stringPtr(string(VersionStatusPublished)),
						UpdatedAt:           publishedAt,
					},
					CreatedAt: publishedAt.Add(-time.Hour),
				},
				Versions: []SkillVersionDetail{
					newMaterializedVersion("https://npm.internal.example.com", "@platform.skill/version-order", "1.9.0", VersionStatusPublished, InstallPolicyResolvable, publishedAt, "Version Order", "Covers version paging order", []string{"commands"}),
					newMaterializedVersion("https://npm.internal.example.com", "@platform.skill/version-order", "1.10.0", VersionStatusPublished, InstallPolicyResolvable, publishedAt, "Version Order", "Covers version paging order", []string{"commands"}),
					newMaterializedVersion("https://npm.internal.example.com", "@platform.skill/version-order", "1.8.0", VersionStatusPublished, InstallPolicyResolvable, publishedAt.Add(-time.Hour), "Version Order", "Covers version paging order", []string{"commands"}),
				},
			},
		},
	}
	repo := newSQLRepositoryForTest(t, data)

	firstPage, err := repo.ListVersions(context.Background(), VersionQuery{
		PackageName: "@platform.skill/version-order",
		PageSize:    1,
	})
	if err != nil {
		t.Fatalf("list first versions page: %v", err)
	}
	if len(firstPage.Items) != 1 {
		t.Fatalf("expected one first-page item, got %d", len(firstPage.Items))
	}
	if firstPage.Items[0].Version != "1.10.0" {
		t.Fatalf("expected semver-newer tie to sort first, got %q", firstPage.Items[0].Version)
	}
	if !firstPage.PageInfo.HasNextPage || firstPage.PageInfo.NextCursor == nil || *firstPage.PageInfo.NextCursor != "cursor_1" {
		t.Fatalf("expected first page to expose next cursor, got %+v", firstPage.PageInfo)
	}

	secondPage, err := repo.ListVersions(context.Background(), VersionQuery{
		PackageName: "@platform.skill/version-order",
		PageSize:    1,
		Cursor:      1,
	})
	if err != nil {
		t.Fatalf("list second versions page: %v", err)
	}
	if len(secondPage.Items) != 1 {
		t.Fatalf("expected one second-page item, got %d", len(secondPage.Items))
	}
	if secondPage.Items[0].Version != "1.9.0" {
		t.Fatalf("expected next semver tie on second page, got %q", secondPage.Items[0].Version)
	}
}

func TestSQLStateStoreListSkillsSQLAppliesSortLimitOffset(t *testing.T) {
	store := &SQLStateStore{dialect: "sqlite"}
	query := SkillQuery{
		Sort:     "package_name_asc",
		PageSize: 1,
		Cursor:   1,
	}

	sortKey, err := normalizedSkillsSort(query.Sort, query.Query)
	if err != nil {
		t.Fatalf("normalize sort: %v", err)
	}

	sqlText := store.listSkillsSQL(query, sortKey)
	if !strings.Contains(sqlText, "ORDER BY package_name ASC") {
		t.Fatalf("expected package sort in SQL, got %q", sqlText)
	}
	if !strings.Contains(sqlText, "LIMIT ?") {
		t.Fatalf("expected SQL limit clause, got %q", sqlText)
	}
	if !strings.Contains(sqlText, "OFFSET ?") {
		t.Fatalf("expected SQL offset clause, got %q", sqlText)
	}

	args := store.listSkillsArgs(query, sortKey)
	want := []any{2, 1}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("expected limit/offset args %v, got %v", want, args)
	}
}

func TestSQLStateStoreListSkillsSQLAppliesAllowedPackageFilters(t *testing.T) {
	store := &SQLStateStore{dialect: "sqlite"}
	query := SkillQuery{
		AllowedPackages: []string{
			"@platform.skill/pdf-skill",
			"@platform.skill/code*",
		},
		PageSize: 10,
	}

	sortKey, err := normalizedSkillsSort(query.Sort, query.Query)
	if err != nil {
		t.Fatalf("normalize sort: %v", err)
	}

	sqlText := store.listSkillsSQL(query, sortKey)
	if !strings.Contains(sqlText, "package_name = ?") {
		t.Fatalf("expected exact package filter in SQL, got %q", sqlText)
	}
	if !strings.Contains(sqlText, "package_name LIKE ?") {
		t.Fatalf("expected prefix package filter in SQL, got %q", sqlText)
	}

	args := store.listSkillsFilterArgs(query)
	want := []any{"@platform.skill/pdf-skill", "@platform.skill/code%"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("expected allowed package args %v, got %v", want, args)
	}
}

func TestSQLStateStoreListCuratedSkillsSQLAppliesLimitOffsetAndAllowedPackages(t *testing.T) {
	store := &SQLStateStore{dialect: "sqlite"}
	query := CuratedSkillQuery{
		AllowedPackages: []string{"@platform.skill/code*"},
		PageSize:        1,
		Cursor:          2,
	}

	sqlText := store.listCuratedSkillsSQL(query)
	if !strings.Contains(sqlText, "ORDER BY updated_at DESC") {
		t.Fatalf("expected curated ordering in SQL, got %q", sqlText)
	}
	if !strings.Contains(sqlText, "LIMIT ?") || !strings.Contains(sqlText, "OFFSET ?") {
		t.Fatalf("expected curated SQL pagination clauses, got %q", sqlText)
	}
	if !strings.Contains(sqlText, "package_name LIKE ?") {
		t.Fatalf("expected curated allowed-package filter, got %q", sqlText)
	}

	args := store.listCuratedSkillsArgs(query)
	want := []any{"@platform.skill/code%", 2, 2}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("expected curated args %v, got %v", want, args)
	}
}

func TestSQLStateStoreListGovernanceSQLAppliesLimitOffset(t *testing.T) {
	store := &SQLStateStore{dialect: "sqlite"}
	query := GovernanceQuery{
		PackageName: "@platform.skill/pdf-skill",
		PageSize:    1,
		Cursor:      1,
	}

	sqlText := store.listGovernanceSQL(query)
	if !strings.Contains(sqlText, "ORDER BY changed_at DESC, id ASC") {
		t.Fatalf("expected governance ordering in SQL, got %q", sqlText)
	}
	if !strings.Contains(sqlText, "LIMIT ?") {
		t.Fatalf("expected SQL limit clause, got %q", sqlText)
	}
	if !strings.Contains(sqlText, "OFFSET ?") {
		t.Fatalf("expected SQL offset clause, got %q", sqlText)
	}

	args := store.listGovernanceArgs(query)
	want := []any{"@platform.skill/pdf-skill", 2, 1}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("expected governance args %v, got %v", want, args)
	}
}

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

func TestSQLRepositoryWithDBUsesInjectedHandle(t *testing.T) {
	ctx := context.Background()
	databaseURL := "file:" + filepath.Join(t.TempDir(), "shared.db")
	driverName := sqlDriverNameForURL(databaseURL)
	db := openCatalogTestDB(t, databaseURL)
	defer db.Close()

	repo, err := NewSQLRepositoryWithDB(ctx, db, driverName, SampleData(), SQLRepositoryOptions{
		ArtifactSource: NewStaticArtifactSource(SampleData().Artifacts),
	})
	if err != nil {
		t.Fatalf("create repository with shared db: %v", err)
	}

	store, ok := repo.reads.store.(*SQLStateStore)
	if !ok {
		t.Fatalf("expected SQLStateStore, got %T", repo.reads.store)
	}
	if store.db != db {
		t.Fatal("expected repository to reuse injected *sql.DB")
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

func TestSQLRepositoryApplyVersionGovernanceChangePreservesExistingGovernanceRows(t *testing.T) {
	ctx := context.Background()
	databaseURL := "file:" + filepath.Join(t.TempDir(), "catalog.db")

	repo, err := NewSQLRepository(ctx, databaseURL, SampleData(), SQLRepositoryOptions{
		ArtifactSource: NewStaticArtifactSource(SampleData().Artifacts),
	})
	if err != nil {
		t.Fatalf("create repository: %v", err)
	}

	db := openCatalogTestDB(t, databaseURL)
	defer db.Close()

	installGovernanceAuditTriggers(t, ctx, db)

	if _, err := repo.ApplyVersionGovernanceChange(ctx, "@platform.skill/pdf-skill", "1.2.0", VersionGovernanceChangeInput{
		Action: GovernanceActionDeprecate,
		Reason: "legacy",
	}, "operator"); err != nil {
		t.Fatalf("apply governance change: %v", err)
	}

	var deletedExistingCount int
	if err := db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM governance_audit
WHERE id = ? AND op = 'delete'`, "gov_01hxyank").Scan(&deletedExistingCount); err != nil {
		t.Fatalf("count deleted existing governance rows: %v", err)
	}
	if deletedExistingCount != 0 {
		t.Fatalf("expected existing governance rows to be preserved, got %d delete operations", deletedExistingCount)
	}

	var updatedExistingCount int
	if err := db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM governance_audit
WHERE id = ? AND op = 'update'`, "gov_01hxyank").Scan(&updatedExistingCount); err != nil {
		t.Fatalf("count updated existing governance rows: %v", err)
	}
	if updatedExistingCount != 0 {
		t.Fatalf("expected existing governance rows to avoid no-op updates, got %d update operations", updatedExistingCount)
	}

	var insertedPackageCount int
	if err := db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM governance_audit
WHERE package_name = ? AND op = 'insert'`, "@platform.skill/pdf-skill").Scan(&insertedPackageCount); err != nil {
		t.Fatalf("count inserted governance rows: %v", err)
	}
	if insertedPackageCount != 1 {
		t.Fatalf("expected exactly 1 inserted governance row for the appended change, got %d", insertedPackageCount)
	}

	var count int
	if err := db.QueryRowContext(ctx, bindSQL(sqlDriverNameForURL(databaseURL), `
SELECT COUNT(*)
FROM governance_records
WHERE package_name = ?`), "@platform.skill/pdf-skill").Scan(&count); err != nil {
		t.Fatalf("count governance rows: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 governance rows for package after append-only change, got %d", count)
	}
}

func TestSQLStateStorePutAggregatePrunesRemovedGovernanceRows(t *testing.T) {
	ctx := context.Background()
	databaseURL := "file:" + filepath.Join(t.TempDir(), "catalog.db")

	db := openCatalogTestDB(t, databaseURL)
	defer db.Close()

	store := NewSQLStateStore(db, sqlDriverNameForURL(databaseURL))
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate store: %v", err)
	}

	data := SampleData()
	var aggregate AgentSkillAggregate
	for _, item := range aggregatesFromSampleData(data) {
		if item.Skill.Detail.PackageName == "@platform.skill/pdf-skill" {
			aggregate = item
			break
		}
	}
	if aggregate.Skill.Detail.PackageName == "" {
		t.Fatal("expected sample aggregate for @platform.skill/pdf-skill")
	}
	if err := store.PutAggregate(ctx, aggregate, 0); err != nil {
		t.Fatalf("put aggregate: %v", err)
	}

	loaded, err := store.GetAggregate(ctx, aggregate.Skill.Detail.PackageName)
	if err != nil {
		t.Fatalf("get aggregate: %v", err)
	}
	loaded.Governance = nil

	if err := store.PutAggregate(ctx, loaded, loaded.Revision); err != nil {
		t.Fatalf("update aggregate: %v", err)
	}

	var remainingCount int
	if err := db.QueryRowContext(ctx, bindSQL(sqlDriverNameForURL(databaseURL), `
SELECT COUNT(*)
FROM governance_records
WHERE package_name = ?`), aggregate.Skill.Detail.PackageName).Scan(&remainingCount); err != nil {
		t.Fatalf("count remaining governance rows: %v", err)
	}
	if remainingCount != 0 {
		t.Fatalf("expected removed governance rows to be pruned, got %d remaining", remainingCount)
	}

	var removedIDCount int
	if err := db.QueryRowContext(ctx, bindSQL(sqlDriverNameForURL(databaseURL), `
SELECT COUNT(*)
FROM governance_records
WHERE id = ?`), "gov_01hxyank").Scan(&removedIDCount); err != nil {
		t.Fatalf("count removed governance id: %v", err)
	}
	if removedIDCount != 0 {
		t.Fatalf("expected removed governance id to be absent, got %d rows", removedIDCount)
	}
}

func TestSQLStateStoreSyncsGovernanceRowsOnPostgres(t *testing.T) {
	ctx := context.Background()
	databaseURL := startTempPostgres(t)

	db := openCatalogTestDB(t, databaseURL)
	defer db.Close()

	store := NewSQLStateStore(db, sqlDriverNameForURL(databaseURL))
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate store: %v", err)
	}

	data := SampleData()
	var aggregate AgentSkillAggregate
	for _, item := range aggregatesFromSampleData(data) {
		if item.Skill.Detail.PackageName == "@platform.skill/pdf-skill" {
			aggregate = item
			break
		}
	}
	if aggregate.Skill.Detail.PackageName == "" {
		t.Fatal("expected sample aggregate for @platform.skill/pdf-skill")
	}
	if err := store.PutAggregate(ctx, aggregate, 0); err != nil {
		t.Fatalf("put aggregate: %v", err)
	}

	loaded, err := store.GetAggregate(ctx, aggregate.Skill.Detail.PackageName)
	if err != nil {
		t.Fatalf("get aggregate: %v", err)
	}
	loaded.Governance = nil

	if err := store.PutAggregate(ctx, loaded, loaded.Revision); err != nil {
		t.Fatalf("prune governance rows: %v", err)
	}

	var remainingCount int
	if err := db.QueryRowContext(ctx, bindSQL(sqlDriverNameForURL(databaseURL), `
SELECT COUNT(*)
FROM governance_records
WHERE package_name = ?`), aggregate.Skill.Detail.PackageName).Scan(&remainingCount); err != nil {
		t.Fatalf("count remaining governance rows: %v", err)
	}
	if remainingCount != 0 {
		t.Fatalf("expected governance rows to be pruned on postgres, got %d remaining", remainingCount)
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

func installGovernanceAuditTriggers(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()

	for _, statement := range []string{
		`CREATE TABLE governance_audit (
			id TEXT NOT NULL,
			package_name TEXT NOT NULL,
			op TEXT NOT NULL
		)`,
		`CREATE TRIGGER governance_audit_insert
		AFTER INSERT ON governance_records
		BEGIN
			INSERT INTO governance_audit (id, package_name, op) VALUES (NEW.id, NEW.package_name, 'insert');
		END`,
		`CREATE TRIGGER governance_audit_update
		AFTER UPDATE ON governance_records
		BEGIN
			INSERT INTO governance_audit (id, package_name, op) VALUES (NEW.id, NEW.package_name, 'update');
		END`,
		`CREATE TRIGGER governance_audit_delete
		AFTER DELETE ON governance_records
		BEGIN
			INSERT INTO governance_audit (id, package_name, op) VALUES (OLD.id, OLD.package_name, 'delete');
		END`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatalf("install governance audit trigger: %v", err)
		}
	}
}

func newSQLRepositoryForTest(t *testing.T, data SampleCatalogData) *CatalogRepository {
	t.Helper()

	repo, err := NewSQLRepository(context.Background(), "file:"+filepath.Join(t.TempDir(), "catalog.db"), data, SQLRepositoryOptions{
		ArtifactSource: NewStaticArtifactSource(data.Artifacts),
	})
	if err != nil {
		t.Fatalf("create repository: %v", err)
	}
	return repo
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
