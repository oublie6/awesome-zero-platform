package main

import (
	"strings"
	"testing"
)

func TestRotationBlocksAnyPersistedJournalActions(t *testing.T) {
	for _, tc := range []struct {
		name      string
		inventory protectionInventory
		allowed   bool
	}{
		{name: "no live action rows", inventory: protectionInventory{seeds: 1}, allowed: true},
		{name: "authenticated actions still need HMAC migration", inventory: protectionInventory{actions: 3, authenticated: 3}, allowed: false},
		{name: "legacy actions cannot be silently resigned", inventory: protectionInventory{actions: 1, legacy: 1}, allowed: false},
		{name: "mixed actions remain unsafe", inventory: protectionInventory{actions: 3, authenticated: 2, legacy: 1}, allowed: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := checkPartialRotationAllowed(tc.inventory)
			if tc.allowed && err != nil {
				t.Fatalf("allowed no-journal seed rewrap rejected: %v", err)
			}
			if !tc.allowed && (err == nil || !strings.Contains(err.Error(), "C09/C10")) {
				t.Fatalf("journal rotation must fail closed with clear reason: %v", err)
			}
		})
	}
}

func TestPreviousRootsRequiresExplicitOperatorKey(t *testing.T) {
	t.Setenv("APP_DOUDIZHU_CONTRIBUTION_KEY_ID", "")
	t.Setenv("APP_DOUDIZHU_CONTRIBUTION_KEY_HEX", "")
	t.Setenv("APP_DOUDIZHU_PREVIOUS_KEYS_JSON", "")
	if _, _, err := previousRoots(); err == nil {
		t.Fatal("maintenance must not manufacture a random replacement root")
	}
	t.Setenv("APP_DOUDIZHU_CONTRIBUTION_KEY_ID", "stable-operator-v1")
	t.Setenv("APP_DOUDIZHU_CONTRIBUTION_KEY_HEX", strings.Repeat("ab", 32))
	active, roots, err := previousRoots()
	if err != nil || active != "stable-operator-v1" || len(roots) != 1 {
		t.Fatalf("stable operator-managed root did not load: %q %d %v", active, len(roots), err)
	}
	if _, _, err := previousRoots(); err != nil {
		t.Fatalf("same operator root must not require rotation on repeated startup: %v", err)
	}
}
