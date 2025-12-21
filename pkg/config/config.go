package config

import (
	"fmt"
	"net/url"
	"path"
	"strings"
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

const (
	DefaultMetricPath     = "/metrics"
	DefaultScrapeInterval = 15 * time.Second
	DefaultScrapeTimeout  = 10 * time.Second
)

func (c *Config) Process() map[string]ScrapeConfig {
	globalInterval := c.Global.ScrapeInterval
	if globalInterval == 0 {
		globalInterval = DefaultScrapeInterval
	}
	globalTimeout := c.Global.ScrapeTimeout
	if globalTimeout == 0 {
		globalTimeout = DefaultScrapeTimeout
	}
	globalLabels := c.Global.ExternalLabels
	hasGlobalLabels := globalLabels != nil && len(globalLabels) > 0
	result := make(map[string]ScrapeConfig, len(c.ScrapeConfigs))
	for i := range c.ScrapeConfigs {
		osc := c.ScrapeConfigs[i]
		sc := ScrapeConfig{
			JobName: osc.JobName,
		}
		metricsPath := osc.MetricsPath
		if metricsPath == "" {
			metricsPath = DefaultMetricPath
		}
		sc.MetricsPath = metricsPath
		if osc.ScrapeInterval == 0 {
			sc.ScrapeInterval = globalInterval
		} else {
			sc.ScrapeInterval = osc.ScrapeInterval
		}
		if osc.ScrapeTimeout == 0 {
			sc.ScrapeTimeout = globalTimeout
		} else {
			sc.ScrapeTimeout = osc.ScrapeTimeout
		}
		stcs := make([]StaticConfig, 0, len(osc.StaticConfigs))
		for j := range osc.StaticConfigs {
			ostc := osc.StaticConfigs[j]
			mergedLabels := make(map[string]string)
			if hasGlobalLabels {
				for k, v := range globalLabels {
					mergedLabels[k] = v
				}
			}
			if ostc.Labels != nil {
				for k, v := range ostc.Labels {
					mergedLabels[k] = v
				}
			}

			realTargets := make([]string, 0, len(ostc.Targets))
			for _, target := range ostc.Targets {
				if realTarget, err := c.buildTargetUrl(target, metricsPath); err == nil {
					realTargets = append(realTargets, realTarget)
				}
			}
			stc := StaticConfig{
				Targets: realTargets,
				Labels:  mergedLabels,
			}
			stcs = append(stcs, stc)
		}
		sc.StaticConfigs = stcs
		result[sc.JobName] = sc
	}
	return result
}

func (c *Config) buildTargetUrl(target, metricPath string) (string, error) {
	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		target = "http://" + target
	}
	u, err := url.Parse(target)
	if err != nil {
		return "", err
	}
	u.Path = path.Join(u.Path, metricPath)
	return u.String(), nil
}
