package config

import "os"
import "gopkg.in/yaml.v3"

type Loader struct {
	configPath string
}

const DefaultConfigPath = "config.yaml"

func NewLoader(configPath string) *Loader {
	if configPath == "" {
		configPath = DefaultConfigPath
	}
	return &Loader{configPath: configPath}
}

func (l *Loader) Load() (*Config, error) {
	c, err := os.ReadFile(l.configPath)
	if err != nil {
		return nil, NewReadError(l.configPath, err)
	}
	cfg := NewConfig()
	err = yaml.Unmarshal(c, cfg)
	if err != nil {
		return nil, NewParseError(l.configPath, err)
	}
	err = cfg.Validate()
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func (l *Loader) getConfigPath() string {
	return l.configPath
}
