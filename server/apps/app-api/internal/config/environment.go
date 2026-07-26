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
