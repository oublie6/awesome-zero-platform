package main

import (
	"flag"
	"os"
	"strings"
	"testing"
)

func TestMaintenanceModeRejectsAllPartialRotation(t *testing.T) {
	for _, mode := range []string{"rotate", "ROTATE", "rotate-seed", "rotate-checkpoint", ""} {
		err := validateMaintenanceMode(mode)
		if err == nil {
			t.Errorf("unsafe or unknown maintenance mode %q accepted", mode)
		}
		if mode == "rotate" && !strings.Contains(err.Error(), "C09/C10") {
			t.Errorf("unsafe rotation rejection lacks migration explanation: %v", err)
		}
	}
	for _, mode := range []string{"key-audit", "prune"} {
		if err := validateMaintenanceMode(mode); err != nil {
			t.Fatalf("supported mode %s rejected: %v", mode, err)
		}
	}
}

func TestRealMaintenanceCLIRejectsRotationBeforeDatabaseOrRootAccess(t *testing.T) {
	oldArgs, oldFlags := os.Args, flag.CommandLine
	defer func() { os.Args, flag.CommandLine = oldArgs, oldFlags }()
	for _, args := range [][]string{
		{"--mode", "rotate", "--hand", "valid-hand", "--epoch", "1", "--execute"},
		{"--mode", "rotate", "--hand", "valid-hand", "--epoch", "1"},
		{"--mode", "rotate", "--hand", "valid-hand", "--epoch", "1", "--execute", "--retention-days", "30"},
	} {
		flag.CommandLine = flag.NewFlagSet("maintenance-test", flag.ContinueOnError)
		os.Args = append([]string{"doudizhu-maintenance"}, args...)
		t.Setenv("APP_MYSQL_DSN", "")
		t.Setenv("APP_DOUDIZHU_CONTRIBUTION_KEY_HEX", "")
		if err := run(); err == nil || !strings.Contains(err.Error(), "partial protection-root rotation is disabled") {
			t.Errorf("unsafe maintenance flags %v must reject before DB/root access: %v", args, err)
		}
	}
}

func TestAuditStillReadOnlyAndRequiresOnlyValidHand(t *testing.T) {
	oldArgs, oldFlags := os.Args, flag.CommandLine
	defer func() { os.Args, flag.CommandLine = oldArgs, oldFlags }()
	for _, args := range [][]string{
		{"--mode", "key-audit", "--execute"},
		{"--mode", "key-audit", "--hand", "invalid<>"},
	} {
		flag.CommandLine = flag.NewFlagSet("maintenance-test", flag.ContinueOnError)
		os.Args = append([]string{"doudizhu-maintenance"}, args...)
		if err := run(); err == nil {
			t.Fatalf("unsafe audit arguments %v accepted", args)
		}
	}
}
