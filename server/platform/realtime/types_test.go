package realtime

import (
	"testing"
	"time"
)

func TestConfigPrepareAndValidate(t *testing.T) {
	config := Config{Enabled: true}
	config.Prepare()
	if err := config.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if config.Path != "/ws" {
		t.Fatalf("Path = %q, want /ws", config.Path)
	}
	if config.SendQueueSize <= 0 || config.MaxConnections <= 0 || config.MaxConnectionsPerAccount <= 0 {
		t.Fatalf("Prepare() did not apply positive limits: %+v", config)
	}
}

func TestConfigRejectsUnsafeOrInconsistentValues(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{name: "relative path", mutate: func(config *Config) { config.Path = "ws" }},
		{name: "wildcard origin", mutate: func(config *Config) { config.AllowedOrigins = []string{"*"} }},
		{name: "invalid origin", mutate: func(config *Config) { config.AllowedOrigins = []string{"example.com"} }},
		{name: "per account exceeds total", mutate: func(config *Config) { config.MaxConnections = 2; config.MaxConnectionsPerAccount = 3 }},
		{name: "unbounded queue", mutate: func(config *Config) { config.SendQueueSize = -1 }},
		{name: "tiny message", mutate: func(config *Config) { config.MaxMessageBytes = 512 }},
		{name: "heartbeat ordering", mutate: func(config *Config) { config.PingInterval = time.Minute; config.PongTimeout = time.Minute }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := Config{Enabled: true}
			config.Prepare()
			test.mutate(&config)
			if err := config.Validate(); err == nil {
				t.Fatal("Validate() error = nil, want rejection")
			}
		})
	}
}

func TestValidateTopic(t *testing.T) {
	for _, topic := range []string{"game:table-42", "account.123", "room_test", " space "} {
		if err := validateTopic(topic); err != nil {
			t.Fatalf("validateTopic(%q) error = %v", topic, err)
		}
	}
	for _, topic := range []string{"", "bad topic", "a/b", "*", string(make([]byte, 129))} {
		if err := validateTopic(topic); err == nil {
			t.Fatalf("validateTopic(%q) error = nil, want rejection", topic)
		}
	}
}
