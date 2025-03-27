package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

// PreCommitConfig represents the structure of the .pre-commit-config.yaml file
type PreCommitConfig struct {
	Repos []Repo `yaml:"repos"`
}

// Repo represents a repository configuration
type Repo struct {
	Repo  string `yaml:"repo"`
	Hooks []Hook `yaml:"hooks"`
}

// Hook represents a hook configuration
type Hook struct {
	ID                      string   `yaml:"id"`
	Name                    string   `yaml:"name"`
	Entry                   string   `yaml:"entry"`
	Description             string   `yaml:"description"`
	Stages                  []string `yaml:"stages"`
	Args                    []string `yaml:"args"`
	Language                string   `yaml:"language"`
	PassFilenames           bool     `yaml:"pass_filenames"`
	MinimumPreCommitVersion string   `yaml:"minimum_pre_commit_version"`
}

// PreloadedConfig is the preloaded configuration
var PreloadedConfig = PreCommitConfig{
	Repos: []Repo{
		{
			Repo: "local",
			Hooks: []Hook{
				{
					ID:                      "cx-secret-detection",
					Name:                    "Cx Secret Detection",
					Entry:                   "cx",
					Description:             "Run Cx CLI secret detection",
					Stages:                  []string{"pre-commit"},
					Args:                    []string{"hooks", "pre-commit", "secrets-scan"},
					Language:                "system",
					PassFilenames:           false,
					MinimumPreCommitVersion: "3.2.0",
				},
			},
		},
	},
}

// WritePreloadedConfig writes the pre-loaded configuration to a specified file
func WritePreloadedConfig(filePath string) error {
	data, err := yaml.Marshal(PreloadedConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %v", err)
	}

	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	return nil
}
