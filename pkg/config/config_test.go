package config

import (
	"reflect"
	"testing"
	"time"
)

func TestConfig_Process(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected map[string]ScrapeConfig
	}{
		{
			name: "使用默认值",
			config: &Config{
				Global: GlobalConfig{},
				ScrapeConfigs: []ScrapeConfig{
					{
						JobName: "test-job",
						StaticConfigs: []StaticConfig{
							{
								Targets: []string{"localhost:9090"},
							},
						},
					},
				},
			},
			expected: map[string]ScrapeConfig{
				"test-job": {
					JobName:        "test-job",
					ScrapeInterval: DefaultScrapeInterval,
					ScrapeTimeout:  DefaultScrapeTimeout,
					MetricsPath:    DefaultMetricPath,
					StaticConfigs: []StaticConfig{
						{
							Targets: []string{"http://localhost:9090/metrics"},
							Labels:  map[string]string{},
						},
					},
				},
			},
		},
		{
			name: "使用全局配置",
			config: &Config{
				Global: GlobalConfig{
					ScrapeInterval: 30 * time.Second,
					ScrapeTimeout:  20 * time.Second,
				},
				ScrapeConfigs: []ScrapeConfig{
					{
						JobName: "test-job",
						StaticConfigs: []StaticConfig{
							{
								Targets: []string{"localhost:9090"},
							},
						},
					},
				},
			},
			expected: map[string]ScrapeConfig{
				"test-job": {
					JobName:        "test-job",
					ScrapeInterval: 30 * time.Second,
					ScrapeTimeout:  20 * time.Second,
					MetricsPath:    DefaultMetricPath,
					StaticConfigs: []StaticConfig{
						{
							Targets: []string{"http://localhost:9090/metrics"},
							Labels:  map[string]string{},
						},
					},
				},
			},
		},
		{
			name: "Job 配置覆盖全局配置",
			config: &Config{
				Global: GlobalConfig{
					ScrapeInterval: 30 * time.Second,
					ScrapeTimeout:  20 * time.Second,
				},
				ScrapeConfigs: []ScrapeConfig{
					{
						JobName:        "test-job",
						ScrapeInterval: 60 * time.Second,
						ScrapeTimeout:  50 * time.Second,
						MetricsPath:    "/custom/metrics",
						StaticConfigs: []StaticConfig{
							{
								Targets: []string{"localhost:9090"},
							},
						},
					},
				},
			},
			expected: map[string]ScrapeConfig{
				"test-job": {
					JobName:        "test-job",
					ScrapeInterval: 60 * time.Second,
					ScrapeTimeout:  50 * time.Second,
					MetricsPath:    "/custom/metrics",
					StaticConfigs: []StaticConfig{
						{
							Targets: []string{"http://localhost:9090/custom/metrics"},
							Labels:  map[string]string{},
						},
					},
				},
			},
		},
		{
			name: "合并全局标签和静态配置标签",
			config: &Config{
				Global: GlobalConfig{
					ExternalLabels: map[string]string{
						"env":     "prod",
						"cluster": "us-west-1",
					},
				},
				ScrapeConfigs: []ScrapeConfig{
					{
						JobName: "test-job",
						StaticConfigs: []StaticConfig{
							{
								Targets: []string{"localhost:9090"},
								Labels: map[string]string{
									"instance": "server1",
									"env":      "dev", // 应该覆盖全局的 env
								},
							},
						},
					},
				},
			},
			expected: map[string]ScrapeConfig{
				"test-job": {
					JobName:        "test-job",
					ScrapeInterval: DefaultScrapeInterval,
					ScrapeTimeout:  DefaultScrapeTimeout,
					MetricsPath:    DefaultMetricPath,
					StaticConfigs: []StaticConfig{
						{
							Targets: []string{"http://localhost:9090/metrics"},
							Labels: map[string]string{
								"env":      "dev", // 局部覆盖全局
								"cluster":  "us-west-1",
								"instance": "server1",
							},
						},
					},
				},
			},
		},
		{
			name: "多个 Job 配置",
			config: &Config{
				Global: GlobalConfig{
					ScrapeInterval: 30 * time.Second,
				},
				ScrapeConfigs: []ScrapeConfig{
					{
						JobName: "job1",
						StaticConfigs: []StaticConfig{
							{
								Targets: []string{"localhost:9090"},
							},
						},
					},
					{
						JobName:        "job2",
						ScrapeInterval: 60 * time.Second,
						StaticConfigs: []StaticConfig{
							{
								Targets: []string{"localhost:9091"},
							},
						},
					},
				},
			},
			expected: map[string]ScrapeConfig{
				"job1": {
					JobName:        "job1",
					ScrapeInterval: 30 * time.Second,
					ScrapeTimeout:  DefaultScrapeTimeout,
					MetricsPath:    DefaultMetricPath,
					StaticConfigs: []StaticConfig{
						{
							Targets: []string{"http://localhost:9090/metrics"},
							Labels:  map[string]string{},
						},
					},
				},
				"job2": {
					JobName:        "job2",
					ScrapeInterval: 60 * time.Second,
					ScrapeTimeout:  DefaultScrapeTimeout,
					MetricsPath:    DefaultMetricPath,
					StaticConfigs: []StaticConfig{
						{
							Targets: []string{"http://localhost:9091/metrics"},
							Labels:  map[string]string{},
						},
					},
				},
			},
		},
		{
			name: "多个静态配置和目标",
			config: &Config{
				Global: GlobalConfig{},
				ScrapeConfigs: []ScrapeConfig{
					{
						JobName: "test-job",
						StaticConfigs: []StaticConfig{
							{
								Targets: []string{"localhost:9090", "localhost:9091"},
								Labels: map[string]string{
									"group": "a",
								},
							},
							{
								Targets: []string{"localhost:9092"},
								Labels: map[string]string{
									"group": "b",
								},
							},
						},
					},
				},
			},
			expected: map[string]ScrapeConfig{
				"test-job": {
					JobName:        "test-job",
					ScrapeInterval: DefaultScrapeInterval,
					ScrapeTimeout:  DefaultScrapeTimeout,
					MetricsPath:    DefaultMetricPath,
					StaticConfigs: []StaticConfig{
						{
							Targets: []string{
								"http://localhost:9090/metrics",
								"http://localhost:9091/metrics",
							},
							Labels: map[string]string{
								"group": "a",
							},
						},
						{
							Targets: []string{"http://localhost:9092/metrics"},
							Labels: map[string]string{
								"group": "b",
							},
						},
					},
				},
			},
		},
		{
			name: "处理带有协议前缀的目标",
			config: &Config{
				Global: GlobalConfig{},
				ScrapeConfigs: []ScrapeConfig{
					{
						JobName: "test-job",
						StaticConfigs: []StaticConfig{
							{
								Targets: []string{
									"http://localhost:9090",
									"https://example.com:9090",
								},
							},
						},
					},
				},
			},
			expected: map[string]ScrapeConfig{
				"test-job": {
					JobName:        "test-job",
					ScrapeInterval: DefaultScrapeInterval,
					ScrapeTimeout:  DefaultScrapeTimeout,
					MetricsPath:    DefaultMetricPath,
					StaticConfigs: []StaticConfig{
						{
							Targets: []string{
								"http://localhost:9090/metrics",
								"https://example.com:9090/metrics",
							},
							Labels: map[string]string{},
						},
					},
				},
			},
		},
		{
			name: "自定义 metrics path",
			config: &Config{
				Global: GlobalConfig{},
				ScrapeConfigs: []ScrapeConfig{
					{
						JobName:     "test-job",
						MetricsPath: "/actuator/prometheus",
						StaticConfigs: []StaticConfig{
							{
								Targets: []string{"localhost:8080"},
							},
						},
					},
				},
			},
			expected: map[string]ScrapeConfig{
				"test-job": {
					JobName:        "test-job",
					ScrapeInterval: DefaultScrapeInterval,
					ScrapeTimeout:  DefaultScrapeTimeout,
					MetricsPath:    "/actuator/prometheus",
					StaticConfigs: []StaticConfig{
						{
							Targets: []string{"http://localhost:8080/actuator/prometheus"},
							Labels:  map[string]string{},
						},
					},
				},
			},
		},
		{
			name: "空的全局标签",
			config: &Config{
				Global: GlobalConfig{
					ExternalLabels: map[string]string{},
				},
				ScrapeConfigs: []ScrapeConfig{
					{
						JobName: "test-job",
						StaticConfigs: []StaticConfig{
							{
								Targets: []string{"localhost:9090"},
								Labels: map[string]string{
									"env": "prod",
								},
							},
						},
					},
				},
			},
			expected: map[string]ScrapeConfig{
				"test-job": {
					JobName:        "test-job",
					ScrapeInterval: DefaultScrapeInterval,
					ScrapeTimeout:  DefaultScrapeTimeout,
					MetricsPath:    DefaultMetricPath,
					StaticConfigs: []StaticConfig{
						{
							Targets: []string{"http://localhost:9090/metrics"},
							Labels: map[string]string{
								"env": "prod",
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.Process()
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Process() = %+v, want %+v", result, tt.expected)
			}
		})
	}
}

func TestConfig_buildTargetUrl(t *testing.T) {
	tests := []struct {
		name        string
		target      string
		metricsPath string
		expected    string
		expectError bool
	}{
		{
			name:        "无协议前缀的主机名",
			target:      "localhost:9090",
			metricsPath: "/metrics",
			expected:    "http://localhost:9090/metrics",
			expectError: false,
		},
		{
			name:        "带 http 协议",
			target:      "http://localhost:9090",
			metricsPath: "/metrics",
			expected:    "http://localhost:9090/metrics",
			expectError: false,
		},
		{
			name:        "带 https 协议",
			target:      "https://example.com:9090",
			metricsPath: "/metrics",
			expected:    "https://example.com:9090/metrics",
			expectError: false,
		},
		{
			name:        "自定义 metrics path",
			target:      "localhost:8080",
			metricsPath: "/actuator/prometheus",
			expected:    "http://localhost:8080/actuator/prometheus",
			expectError: false,
		},
		{
			name:        "目标已有路径",
			target:      "http://localhost:9090/app",
			metricsPath: "/metrics",
			expected:    "http://localhost:9090/app/metrics",
			expectError: false,
		},
		{
			name:        "IP 地址",
			target:      "192.168.1.100:9090",
			metricsPath: "/metrics",
			expected:    "http://192.168.1.100:9090/metrics",
			expectError: false,
		},
		{
			name:        "域名不带端口",
			target:      "example.com",
			metricsPath: "/metrics",
			expected:    "http://example.com/metrics",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{}
			result, err := c.buildTargetUrl(tt.target, tt.metricsPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("buildTargetUrl() expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("buildTargetUrl() unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("buildTargetUrl() = %v, want %v", result, tt.expected)
				}
			}
		})
	}
}
