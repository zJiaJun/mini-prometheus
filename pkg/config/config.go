package config

import (
	"fmt"
	"time"
)

type Config struct {
	Global        GlobalConfig   `yaml:"global"`
	ScrapeConfigs []ScrapeConfig `yaml:"scrape_configs"`
}

type GlobalConfig struct {
	ScrapeInterval     time.Duration     `yaml:"scrape_interval"`
	ScrapeTimeout      time.Duration     `yaml:"scrape_timeout"`
	EvaluationInterval time.Duration     `yaml:"evaluation_interval"`
	ExternalLabels     map[string]string `yaml:"external_labels"`
}
type ScrapeConfig struct {
	JobName        string         `yaml:"job_name"`
	ScrapeInterval time.Duration  `yaml:"scrape_interval"`
	ScrapeTimeout  time.Duration  `yaml:"scrape_timeout"`
	MetricsPath    string         `yaml:"metrics_path"`
	StaticConfigs  []StaticConfig `yaml:"static_configs"`
}
type StaticConfig struct {
	Targets []string          `yaml:"targets"`
	Labels  map[string]string `yaml:"labels"`
}

func NewConfig() *Config {
	return &Config{}
}

func (c *Config) Validate() error {
	if c.Global.ScrapeTimeout > c.Global.ScrapeInterval {
		return fmt.Errorf("global scrape timeout (%v) must be <= scrape interval (%v)",
			c.Global.ScrapeTimeout, c.Global.ScrapeInterval)
	}
	for i, sc := range c.ScrapeConfigs {
		if sc.JobName == "" {
			return fmt.Errorf("scrape_config[%d]: job_name is required", i)
		}

		if sc.ScrapeTimeout > sc.ScrapeInterval {
			return fmt.Errorf("job %q: scrape timeout (%v) must be <= scrape interval (%v)",
				sc.JobName, sc.ScrapeTimeout, sc.ScrapeInterval)
		}

		if len(sc.StaticConfigs) == 0 {
			return fmt.Errorf("job %q: no targets configured", sc.JobName)
		}

		for j, stc := range sc.StaticConfigs {
			if len(stc.Targets) == 0 {
				return fmt.Errorf("job %q: static_config[%d] has no targets", sc.JobName, j)
			}
		}
	}
	return nil
}
