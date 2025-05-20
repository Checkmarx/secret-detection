package pre_receive

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"os"
)

type IgnoreSecrets struct {
	IgnoreRule     []string `yaml:"ignore_rule"`
	IgnoreScore    int      `yaml:"ignore_score"`
	IgnoreSeverity string   `yaml:"ignore_severity"`
	IgnoreSecret   []string `yaml:"ignore_secret"`
}

type PreReceiveConfig struct {
	MaxCommits    int           `yaml:"max_commits"`
	ExcludePath   []string      `yaml:"exclude_path"`
	IgnoreSecrets IgnoreSecrets `yaml:"ignore_secrets"`
}

var defaultConfig = PreReceiveConfig{
	MaxCommits:  50,
	ExcludePath: []string{},
	IgnoreSecrets: IgnoreSecrets{
		IgnoreRule:     []string{},
		IgnoreScore:    0,
		IgnoreSeverity: "",
		IgnoreSecret:   []string{},
	},
}

func LoadPreReceiveConfig(path string) PreReceiveConfig {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("Warning: No config file found. Using defaults: %v\n", err)
		return defaultConfig
	}
	var cfg PreReceiveConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		fmt.Printf("Warning: Config file misconfigured. Using defaults: %v\n", err)
		return defaultConfig
	}
	return cfg
}
