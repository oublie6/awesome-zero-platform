package admin

import (
	"errors"
	"testing"

	"github.com/oublie6/awesome-zero-platform/server/platform/authz"
)

func TestProtectRawRulesRequiresSuperAdministratorSafetyNet(t *testing.T) {
	valid := []authz.RawRule{
		{PType: "p", Values: []string{SuperAdminRole, "/*", ".*"}},
		{PType: "g", Values: []string{"account-1", SuperAdminRole}},
	}
	if err := protectRawRules(valid); err != nil {
		t.Fatalf("protectRawRules(valid) error = %v", err)
	}

	for name, rules := range map[string][]authz.RawRule{
		"missing wildcard": {{PType: "g", Values: []string{"account-1", SuperAdminRole}}},
		"missing member":   {{PType: "p", Values: []string{SuperAdminRole, "/*", ".*"}}},
	} {
		t.Run(name, func(t *testing.T) {
			if err := protectRawRules(rules); !errors.Is(err, ErrProtectedRole) {
				t.Fatalf("protectRawRules() error = %v, want ErrProtectedRole", err)
			}
		})
	}
}

func TestUniqueStringsNormalizesAndSorts(t *testing.T) {
	got := uniqueStrings([]string{" operator ", "viewer", "operator", ""})
	want := []string{"operator", "viewer"}
	if len(got) != len(want) {
		t.Fatalf("uniqueStrings() = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("uniqueStrings() = %#v, want %#v", got, want)
		}
	}
}
