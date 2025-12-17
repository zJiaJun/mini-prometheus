package config

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

// TestNewLoader 测试 Loader 创建
func TestNewLoader(t *testing.T) {
	t.Run("使用指定路径", func(t *testing.T) {
		loader := NewLoader("custom.yaml")
		if loader.getConfigPath() != "custom.yaml" {
			t.Errorf("期望 configPath=custom.yaml, 实际=%s", loader.configPath)
		}
	})

	t.Run("使用默认路径", func(t *testing.T) {
		loader := NewLoader("")
		if loader.getConfigPath() != DefaultConfigPath {
			t.Errorf("期望 configPath=%s, 实际=%s", DefaultConfigPath, loader.configPath)
		}
	})
}

// TestLoader_Load_Valid 测试加载有效配置
func TestLoader_Load_Valid(t *testing.T) {
	tests := []struct {
		name     string
		file     string
		validate func(*testing.T, *Config)
	}{
		{
			name: "完整配置",
			file: "testdata/valid.yaml",
			validate: func(t *testing.T, c *Config) {
				// 验证 global 配置
				if c.Global.ScrapeInterval != 15*time.Second {
					t.Errorf("期望 scrape_interval=15s, 实际=%v", c.Global.ScrapeInterval)
				}
				if c.Global.ScrapeTimeout != 10*time.Second {
					t.Errorf("期望 scrape_timeout=10s, 实际=%v", c.Global.ScrapeTimeout)
				}
				if c.Global.EvaluationInterval != 60*time.Second {
					t.Errorf("期望 evaluation_interval=60s, 实际=%v", c.Global.EvaluationInterval)
				}

				// 验证 external_labels
				if c.Global.ExternalLabels["env"] != "prod" {
					t.Errorf("期望 env=prod, 实际=%s", c.Global.ExternalLabels["env"])
				}
				if c.Global.ExternalLabels["cluster"] != "test-cluster" {
					t.Errorf("期望 cluster=test-cluster, 实际=%s", c.Global.ExternalLabels["cluster"])
				}

				// 验证 scrape_configs
				if len(c.ScrapeConfigs) != 1 {
					t.Fatalf("期望 1 个 scrape_config, 实际=%d", len(c.ScrapeConfigs))
				}

				sc := c.ScrapeConfigs[0]
				if sc.JobName != "my-app" {
					t.Errorf("期望 job_name=my-app, 实际=%s", sc.JobName)
				}
				if sc.MetricsPath != "/metrics" {
					t.Errorf("期望 metrics_path=/metrics, 实际=%s", sc.MetricsPath)
				}

				// 验证 static_configs
				if len(sc.StaticConfigs) != 2 {
					t.Fatalf("期望 2 个 static_config, 实际=%d", len(sc.StaticConfigs))
				}

				// 验证第一个 static_config
				stc1 := sc.StaticConfigs[0]
				if len(stc1.Targets) != 2 {
					t.Errorf("期望 2 个 targets, 实际=%d", len(stc1.Targets))
				}
				if stc1.Targets[0] != "localhost:8080" {
					t.Errorf("期望 target=localhost:8080, 实际=%s", stc1.Targets[0])
				}
				if stc1.Labels["team"] != "backend" {
					t.Errorf("期望 team=backend, 实际=%s", stc1.Labels["team"])
				}

				// 验证第二个 static_config
				stc2 := sc.StaticConfigs[1]
				if len(stc2.Targets) != 1 {
					t.Errorf("期望 1 个 target, 实际=%d", len(stc2.Targets))
				}
				if stc2.Targets[0] != "localhost:9090" {
					t.Errorf("期望 target=localhost:9090, 实际=%s", stc2.Targets[0])
				}
			},
		},
		{
			name: "最小配置",
			file: "testdata/minimal.yaml",
			validate: func(t *testing.T, c *Config) {
				if c.Global.ScrapeInterval != 15*time.Second {
					t.Errorf("期望 scrape_interval=15s, 实际=%v", c.Global.ScrapeInterval)
				}

				if len(c.ScrapeConfigs) != 1 {
					t.Fatalf("期望 1 个 scrape_config, 实际=%d", len(c.ScrapeConfigs))
				}

				sc := c.ScrapeConfigs[0]
				if sc.JobName != "test" {
					t.Errorf("期望 job_name=test, 实际=%s", sc.JobName)
				}

				if len(sc.StaticConfigs) != 1 {
					t.Fatalf("期望 1 个 static_config, 实际=%d", len(sc.StaticConfigs))
				}

				if len(sc.StaticConfigs[0].Targets) != 1 {
					t.Errorf("期望 1 个 target, 实际=%d", len(sc.StaticConfigs[0].Targets))
				}
			},
		},
		{
			name: "默认配置文件",
			file: "testdata/default_config.yaml",
			validate: func(t *testing.T, c *Config) {
				if c.Global.ScrapeInterval != 15*time.Second {
					t.Errorf("期望 scrape_interval=15s, 实际=%v", c.Global.ScrapeInterval)
				}

				if len(c.ScrapeConfigs) != 0 {
					t.Errorf("期望 0 个 scrape_config, 实际=%d", len(c.ScrapeConfigs))
				}
			},
		},
		{
			name: "测试配置文件",
			file: "testdata/test_config.yaml",
			validate: func(t *testing.T, c *Config) {
				if len(c.ScrapeConfigs) != 1 {
					t.Fatalf("期望 1 个 scrape_config, 实际=%d", len(c.ScrapeConfigs))
				}

				sc := c.ScrapeConfigs[0]
				if sc.JobName != "default" {
					t.Errorf("期望 job_name=default, 实际=%s", sc.JobName)
				}

				if len(sc.StaticConfigs[0].Targets) != 2 {
					t.Errorf("期望 2 个 targets, 实际=%d", len(sc.StaticConfigs[0].Targets))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := NewLoader(tt.file)
			cfg, err := loader.Load()
			if err != nil {
				t.Fatalf("加载配置失败: %v", err)
			}

			if cfg == nil {
				t.Fatal("配置不应该为 nil")
			}

			tt.validate(t, cfg)
		})
	}
}

// TestLoader_Load_Errors 测试各种错误情况
func TestLoader_Load_Errors(t *testing.T) {
	tests := []struct {
		name        string
		file        string
		expectError string
	}{
		{
			name:        "文件不存在",
			file:        "testdata/nonexistent.yaml",
			expectError: "failed to read config file",
		},
		{
			name:        "无效的 YAML 语法",
			file:        "testdata/invalid_yaml.yaml",
			expectError: "failed to parse config file",
		},
		{
			name:        "缺少 job_name",
			file:        "testdata/missing_job_name.yaml",
			expectError: "job_name is required",
		},
		{
			name:        "全局 timeout 大于 interval",
			file:        "testdata/timeout_gt_interval.yaml",
			expectError: "scrape timeout",
		},
		{
			name:        "任务 timeout 大于 interval",
			file:        "testdata/job_timeout_gt_interval.yaml",
			expectError: "scrape timeout",
		},
		{
			name:        "没有配置 targets",
			file:        "testdata/no_targets.yaml",
			expectError: "no targets configured",
		},
		{
			name:        "targets 为空数组",
			file:        "testdata/empty_targets.yaml",
			expectError: "has no targets",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := NewLoader(tt.file)
			cfg, err := loader.Load()

			// 应该返回错误
			if err == nil {
				t.Errorf("期望错误，但返回了 nil")
			}

			// 验证错误信息
			if err != nil && !strings.Contains(err.Error(), tt.expectError) {
				t.Errorf("错误信息不匹配:\n期望包含: %s\n实际错误: %v", tt.expectError, err)
			}

			// 配置应该为 nil
			if cfg != nil {
				t.Errorf("错误情况下配置应该为 nil")
			}
		})
	}
}

// TestLoader_Load_ErrorWrapping 测试错误包装
func TestLoader_Load_ErrorWrapping(t *testing.T) {
	t.Run("读取错误应该包装原始错误", func(t *testing.T) {
		loader := NewLoader("testdata/nonexistent.yaml")
		_, err := loader.Load()

		if err == nil {
			t.Fatal("期望错误")
		}

		// 验证错误链
		if !errors.Is(err, os.ErrNotExist) {
			t.Error("错误应该包装 os.ErrNotExist")
		}

		// 验证错误信息包含文件路径
		if !strings.Contains(err.Error(), "testdata/nonexistent.yaml") {
			t.Errorf("错误信息应该包含文件路径，实际: %v", err)
		}
	})

	t.Run("解析错误应该包装原始错误", func(t *testing.T) {
		loader := NewLoader("testdata/invalid_yaml.yaml")
		_, err := loader.Load()

		if err == nil {
			t.Fatal("期望错误")
		}

		// 验证错误信息
		if !strings.Contains(err.Error(), "failed to parse config file") {
			t.Errorf("错误信息应该包含 parse 信息，实际: %v", err)
		}

		if !strings.Contains(err.Error(), "testdata/invalid_yaml.yaml") {
			t.Errorf("错误信息应该包含文件路径，实际: %v", err)
		}
	})

	t.Run("验证错误应该返回", func(t *testing.T) {
		loader := NewLoader("testdata/missing_job_name.yaml")
		_, err := loader.Load()

		if err == nil {
			t.Fatal("期望错误")
		}

		// 验证错误信息
		if !strings.Contains(err.Error(), "job_name is required") {
			t.Errorf("错误信息应该包含验证失败原因，实际: %v", err)
		}
	})
}

// TestLoader_Load_EdgeCases 测试边界情况
func TestLoader_Load_EdgeCases(t *testing.T) {
	t.Run("空的 external_labels", func(t *testing.T) {
		loader := NewLoader("testdata/default_config.yaml")
		cfg, err := loader.Load()
		if err != nil {
			t.Fatalf("加载配置失败: %v", err)
		}

		// external_labels 应该存在（可能为空）
		if cfg.Global.ExternalLabels == nil {
			t.Error("external_labels 不应该为 nil")
		}
	})

	t.Run("空的 scrape_configs", func(t *testing.T) {
		loader := NewLoader("testdata/default_config.yaml")
		cfg, err := loader.Load()
		if err != nil {
			t.Fatalf("加载配置失败: %v", err)
		}

		// 空的 scrape_configs 应该是有效的
		if cfg.ScrapeConfigs == nil {
			t.Error("scrape_configs 不应该为 nil")
		}

		if len(cfg.ScrapeConfigs) != 0 {
			t.Errorf("期望 0 个 scrape_config, 实际=%d", len(cfg.ScrapeConfigs))
		}
	})
}

// TestConfig_Validate 单独测试 Validate 方法
func TestConfig_Validate(t *testing.T) {
	t.Run("有效配置", func(t *testing.T) {
		cfg := &Config{
			Global: GlobalConfig{
				ScrapeInterval: 15 * time.Second,
				ScrapeTimeout:  10 * time.Second,
			},
			ScrapeConfigs: []ScrapeConfig{
				{
					JobName:        "test",
					ScrapeInterval: 15 * time.Second,
					ScrapeTimeout:  10 * time.Second,
					StaticConfigs: []StaticConfig{
						{
							Targets: []string{"localhost:9090"},
						},
					},
				},
			},
		}

		err := cfg.Validate()
		if err != nil {
			t.Errorf("有效配置不应该返回错误: %v", err)
		}
	})

	t.Run("全局 timeout > interval", func(t *testing.T) {
		cfg := &Config{
			Global: GlobalConfig{
				ScrapeInterval: 10 * time.Second,
				ScrapeTimeout:  15 * time.Second,
			},
			ScrapeConfigs: []ScrapeConfig{},
		}

		err := cfg.Validate()
		if err == nil {
			t.Error("期望验证错误")
		}
		if !strings.Contains(err.Error(), "scrape timeout") {
			t.Errorf("错误信息不正确: %v", err)
		}
	})

	t.Run("任务 timeout > interval", func(t *testing.T) {
		cfg := &Config{
			Global: GlobalConfig{
				ScrapeInterval: 15 * time.Second,
				ScrapeTimeout:  10 * time.Second,
			},
			ScrapeConfigs: []ScrapeConfig{
				{
					JobName:        "bad-job",
					ScrapeInterval: 10 * time.Second,
					ScrapeTimeout:  15 * time.Second,
					StaticConfigs: []StaticConfig{
						{Targets: []string{"localhost:9090"}},
					},
				},
			},
		}

		err := cfg.Validate()
		if err == nil {
			t.Error("期望验证错误")
		}
		if !strings.Contains(err.Error(), "bad-job") {
			t.Errorf("错误信息应该包含任务名: %v", err)
		}
	})

	t.Run("缺少 job_name", func(t *testing.T) {
		cfg := &Config{
			Global: GlobalConfig{
				ScrapeInterval: 15 * time.Second,
				ScrapeTimeout:  10 * time.Second,
			},
			ScrapeConfigs: []ScrapeConfig{
				{
					JobName: "",
					StaticConfigs: []StaticConfig{
						{Targets: []string{"localhost:9090"}},
					},
				},
			},
		}

		err := cfg.Validate()
		if err == nil {
			t.Error("期望验证错误")
		}
		if !strings.Contains(err.Error(), "job_name is required") {
			t.Errorf("错误信息不正确: %v", err)
		}
	})

	t.Run("没有 targets", func(t *testing.T) {
		cfg := &Config{
			Global: GlobalConfig{
				ScrapeInterval: 15 * time.Second,
				ScrapeTimeout:  10 * time.Second,
			},
			ScrapeConfigs: []ScrapeConfig{
				{
					JobName:       "test",
					StaticConfigs: []StaticConfig{},
				},
			},
		}

		err := cfg.Validate()
		if err == nil {
			t.Error("期望验证错误")
		}
		if !strings.Contains(err.Error(), "no targets configured") {
			t.Errorf("错误信息不正确: %v", err)
		}
	})

	t.Run("targets 数组为空", func(t *testing.T) {
		cfg := &Config{
			Global: GlobalConfig{
				ScrapeInterval: 15 * time.Second,
				ScrapeTimeout:  10 * time.Second,
			},
			ScrapeConfigs: []ScrapeConfig{
				{
					JobName: "test",
					StaticConfigs: []StaticConfig{
						{Targets: []string{}},
					},
				},
			},
		}

		err := cfg.Validate()
		if err == nil {
			t.Error("期望验证错误")
		}
		if !strings.Contains(err.Error(), "has no targets") {
			t.Errorf("错误信息不正确: %v", err)
		}
	})
}
