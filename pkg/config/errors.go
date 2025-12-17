package config

import (
	"fmt"
)

func NewReadError(configPath string, err error) error {
	return fmt.Errorf("failed to read config file %q: %w", configPath, err)
}

func NewParseError(configPath string, err error) error {
	return fmt.Errorf("failed to parse config file %q: %w", configPath, err)
}
