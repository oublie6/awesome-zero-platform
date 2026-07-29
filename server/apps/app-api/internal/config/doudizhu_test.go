package config

import (
	"testing"
	"time"
)

func TestDoudizhuConfiguration(t *testing.T) {
	t.Parallel()

	valid := Config{}
	valid.Name = "main-api"
	valid.Host = "127.0.0.1"
	valid.Port = 8888
	valid.MySQL.Addr = "127.0.0.1:3306"
	valid.MySQL.Database = "awesome_zero_platform"
	valid.MySQL.User = "app_local"
	valid.MySQL.Password = "dev-only-password"
	valid.MySQL.ParseTime = true
	valid.Redis.Addr = "127.0.0.1:6379"
	valid.Authentication.Enabled = true
	valid.Authentication.AccessTokenSecret = "0123456789abcdef0123456789abcdef"
	valid.RevealKeys.Enabled = true
	valid.RevealKeys.StaticJSON = `{"version":"reveal-key-manifest-v1","keys":[]}`
	valid.Doudizhu = DoudizhuConfig{
		Enabled: true, BeaconProvider: "trusted-adapter", BeaconRound: "round-1",
		BeaconProofSecret:  "0123456789abcdef0123456789abcdef",
		ContributionKeyID:  "contribution-v1",
		ContributionKeyHex: "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f",
	}
	valid.Prepare()
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid Doudizhu config: %v", err)
	}
	if valid.Doudizhu.BiddingTimeout != 45*time.Second || valid.Doudizhu.PlayingTimeout != 60*time.Second || valid.Doudizhu.ReplayEntries != 4096 {
		t.Fatalf("prepared Doudizhu config=%#v", valid.Doudizhu)
	}

	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{name: "requires authentication", mutate: func(cfg *Config) { cfg.Authentication.Enabled = false }},
		{name: "requires reveal keys", mutate: func(cfg *Config) { cfg.RevealKeys.Enabled = false }},
		{name: "requires beacon plan", mutate: func(cfg *Config) { cfg.Doudizhu.BeaconProvider = "" }},
		{name: "requires strong proof secret", mutate: func(cfg *Config) { cfg.Doudizhu.BeaconProofSecret = "short" }},
		{name: "requires canonical contribution key", mutate: func(cfg *Config) { cfg.Doudizhu.ContributionKeyHex = "ABCDEF" }},
		{name: "requires positive replay entries", mutate: func(cfg *Config) { cfg.Doudizhu.ReplayEntries = -1 }},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			cfg := valid
			test.mutate(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestApplyEnvironmentDoudizhu(t *testing.T) {
	t.Setenv("APP_DOUDIZHU_ENABLED", "true")
	t.Setenv("APP_DOUDIZHU_BEACON_PROVIDER", "local-hmac")
	t.Setenv("APP_DOUDIZHU_BEACON_ROUND", "local-round-1")
	t.Setenv("APP_DOUDIZHU_BEACON_PROOF_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("APP_DOUDIZHU_CONTRIBUTION_KEY_ID", "local-contribution-v1")
	t.Setenv("APP_DOUDIZHU_CONTRIBUTION_KEY_HEX", "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f")
	var cfg Config
	if err := cfg.ApplyEnvironment(); err != nil {
		t.Fatal(err)
	}
	if !cfg.Doudizhu.Enabled || cfg.Doudizhu.BeaconProvider != "local-hmac" || cfg.Doudizhu.BeaconRound != "local-round-1" || cfg.Doudizhu.BeaconProofSecret == "" || cfg.Doudizhu.ContributionKeyID != "local-contribution-v1" || cfg.Doudizhu.ContributionKeyHex == "" {
		t.Fatalf("environment config=%#v", cfg.Doudizhu)
	}
}
