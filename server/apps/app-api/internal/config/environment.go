package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func (c *Config) ApplyEnvironment() error {
	applyString("APP_HOST", &c.Host)
	applyString("APP_MYSQL_ADDR", &c.MySQL.Addr)
	applyString("APP_MYSQL_DATABASE", &c.MySQL.Database)
	applyString("APP_MYSQL_USER", &c.MySQL.User)
	applyString("APP_MYSQL_PASSWORD", &c.MySQL.Password)
	applyString("APP_REDIS_ADDR", &c.Redis.Addr)
	applyString("APP_REDIS_USERNAME", &c.Redis.Username)
	applyString("APP_REDIS_PASSWORD", &c.Redis.Password)
	applyString("APP_AUTH_ACCESS_TOKEN_SECRET", &c.Authentication.AccessTokenSecret)
	applyString("APP_ADMIN_BOOTSTRAP_TOKEN", &c.Admin.BootstrapToken)
	applyString("APP_INSTANCE_ID", &c.Authorization.Cluster.InstanceID)
	applyString("APP_REVEAL_KEYS_STATIC_JSON", &c.RevealKeys.StaticJSON)
	applyString("APP_DOUDIZHU_BEACON_PROVIDER", &c.Doudizhu.BeaconProvider)
	applyString("APP_DOUDIZHU_BEACON_ROUND", &c.Doudizhu.BeaconRound)
	applyString("APP_DOUDIZHU_BEACON_PROOF_SECRET", &c.Doudizhu.BeaconProofSecret)
	applyString("APP_DOUDIZHU_CONTRIBUTION_KEY_ID", &c.Doudizhu.ContributionKeyID)
	applyString("APP_DOUDIZHU_CONTRIBUTION_KEY_HEX", &c.Doudizhu.ContributionKeyHex)
	if raw := strings.TrimSpace(os.Getenv("APP_DOUDIZHU_ENABLED")); raw != "" {
		enabled, err := strconv.ParseBool(raw)
		if err != nil {
			return fmt.Errorf("parse APP_DOUDIZHU_ENABLED: %w", err)
		}
		c.Doudizhu.Enabled = enabled
	}

	if raw := strings.TrimSpace(os.Getenv("APP_REVEAL_KEYS_ENABLED")); raw != "" {
		enabled, err := strconv.ParseBool(raw)
		if err != nil {
			return fmt.Errorf("parse APP_REVEAL_KEYS_ENABLED: %w", err)
		}
		c.RevealKeys.Enabled = enabled
	}

	if raw := strings.TrimSpace(os.Getenv("APP_PORT")); raw != "" {
		port, err := strconv.Atoi(raw)
		if err != nil {
			return fmt.Errorf("parse APP_PORT: %w", err)
		}
		c.Port = port
	}
	return nil
}

func applyString(name string, target *string) {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		*target = value
	}
}
