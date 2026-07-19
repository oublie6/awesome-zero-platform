package config

import (
	"fmt"
	"strings"
)

func (c Config) Validate() error {
	if strings.TrimSpace(c.Name) == "" {
		return fmt.Errorf("name must not be empty")
	}

	if strings.TrimSpace(c.Host) == "" {
		return fmt.Errorf("host must not be empty")
	}

	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}

	return nil
}
