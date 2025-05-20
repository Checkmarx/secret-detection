package pre_receive

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"os"
)

type IgnoreSecrets struct {
	IgnoreRule   []string `yaml:"ignore_rule"`
	IgnoreSecret []string `yaml:"ignore_result_id"`
}

type PreReceiveConfig struct {
	ExcludePath   []string      `yaml:"exclude_path"`
	IgnoreSecrets IgnoreSecrets `yaml:"ignore_secrets"`
}

func loadScanConfig(configPath string) (PreReceiveConfig, error) {
	var cfg PreReceiveConfig
	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return PreReceiveConfig{}, fmt.Errorf("could not find config file at %s", configPath)
		}

		if err = yaml.Unmarshal(data, &cfg); err != nil {
			return PreReceiveConfig{}, fmt.Errorf("configuration file is misconfigured")
		}
	}
	return PreReceiveConfig{
		ExcludePath:   cfg.ExcludePath,
		IgnoreSecrets: cfg.IgnoreSecrets,
	}, nil
}
