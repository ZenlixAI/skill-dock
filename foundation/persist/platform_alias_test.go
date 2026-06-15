package persist

import (
	"reflect"
	"testing"

	platformpersist "skill_registry/internal/platform/persistence"
)

func TestStoreSetAliasMatchesPlatformType(t *testing.T) {
	if reflect.TypeOf(SQLStoreSet{}) != reflect.TypeOf(platformpersist.SQLStoreSet{}) {
		t.Fatalf("expected SQLStoreSet to alias platform SQLStoreSet")
	}
	if reflect.TypeOf((*SQLAuditStore)(nil)) != reflect.TypeOf((*platformpersist.SQLAuditStore)(nil)) {
		t.Fatalf("expected SQLAuditStore to alias platform SQLAuditStore")
	}
	if reflect.TypeOf((*SQLWorkspaceStore)(nil)) != reflect.TypeOf((*platformpersist.SQLWorkspaceStore)(nil)) {
		t.Fatalf("expected SQLWorkspaceStore to alias platform SQLWorkspaceStore")
	}
}
