package auth

import (
	"reflect"
	"testing"

	platformauth "skill_registry/internal/platform/auth"
)

func TestAuthorizerAliasMatchesPlatformType(t *testing.T) {
	if reflect.TypeOf(Principal{}) != reflect.TypeOf(platformauth.Principal{}) {
		t.Fatalf("expected Principal to alias platform Principal")
	}
	if reflect.TypeOf((*Authorizer)(nil)) != reflect.TypeOf((*platformauth.Authorizer)(nil)) {
		t.Fatalf("expected Authorizer to alias platform Authorizer")
	}
}
