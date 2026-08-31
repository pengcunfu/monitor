package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config 应用配置，由 YAML 文件覆盖默认值。
type Config struct {
	Server    Server    `yaml:"server"`
	Collector Collector `yaml:"collector"`
}

// Server HTTP 服务配置。
type Server struct {
	Addr       string `yaml:"addr"`
	DBPath     string `yaml:"db_path"`
	JWTSecret  string `yaml:"jwt_secret"`
	JWTExpireH int    `yaml:"jwt_expire_h"`
}

// Collector 采集周期配置（可被数据库 settings 覆盖）。
type Collector struct {
	IntervalSec        int `yaml:"interval_sec"`
	ProcessIntervalSec int `yaml:"process_interval_sec"`
	ServiceIntervalSec int `yaml:"service_interval_sec"`
	ProcessTopN        int `yaml:"process_top_n"`
}

// Default 返回默认配置。
func Default() *Config {
	return &Config{
		Server: Server{
			Addr:       ":8080",
			DBPath:     "monitor.db",
			JWTSecret:  "please-change-me",
			JWTExpireH: 72,
		},
		Collector: Collector{
			IntervalSec:        10,
			ProcessIntervalSec: 30,
			ServiceIntervalSec: 30,
			ProcessTopN:        20,
		},
	}
}

// Load 加载配置文件：path 为空时使用默认配置；非空时文件必须存在。
func Load(path string) (*Config, error) {
	cfg := Default()
	if path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("读取配置文件 %s 失败: %w", path, err)
		}
		if err := yaml.Unmarshal(b, cfg); err != nil {
			return nil, fmt.Errorf("解析配置文件 %s 失败: %w", path, err)
		}
	}
	return cfg, nil
}
