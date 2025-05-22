package pre_receive

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"os"
	"strings"
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
			return PreReceiveConfig{}, fmt.Errorf("configuration file at %s is misconfigured", configPath)
		}
	}
	return PreReceiveConfig{
		ExcludePath:   cfg.ExcludePath,
		IgnoreSecrets: cfg.IgnoreSecrets,
	}, nil
}

func configExcludesToGitExcludes(patterns []string) []string {
	var specs []string
	for _, pattern := range patterns {
		// Trim spaces and surrounding quotes
		p := strings.Trim(strings.TrimSpace(pattern), `"`)
		if p == "" {
			continue
		}
		// Normalize Windows backslashes to forward slashes
		p = strings.ReplaceAll(p, `\`, "/")
		// Strip any leading slashes
		p = strings.TrimLeft(p, "/")
		// Wrap in Git negative pathspec
		specs = append(specs, fmt.Sprintf(`:(exclude)%s`, p))
	}
	return specs
}
