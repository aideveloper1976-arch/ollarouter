package config

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

var envPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

// ParseFile reads and parses a YAML config file.
func ParseFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}
	return Parse(data)
}

// SaveFile writes the config back to a YAML file.
func SaveFile(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encoding config YAML: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}
	return nil
}

// Parse parses YAML config data with env var interpolation.
func Parse(data []byte) (*Config, error) {
	// Interpolate environment variables
	interpolated := interpolateEnv(string(data))

	var cfg Config
	cfg.Settings = DefaultSettings()

	if err := yaml.Unmarshal([]byte(interpolated), &cfg); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation: %w", err)
	}

	return &cfg, nil
}

// interpolateEnv replaces ${VAR} with environment variable values.
func interpolateEnv(input string) string {
	return envPattern.ReplaceAllStringFunc(input, func(match string) string {
		varName := match[2 : len(match)-1] // strip ${ and }
		defaultVal := ""
		// Check for ${VAR:-default} syntax
		if idx := findSep(varName); idx >= 0 {
			defaultVal = varName[idx+2:]
			varName = varName[:idx]
		}
		val := os.Getenv(varName)
		if val == "" {
			val = defaultVal
		}
		return val
	})
}

// findSep finds the index of ":-" in a string for default value syntax.
func findSep(s string) int {
	for i := 0; i < len(s)-1; i++ {
		if s[i] == ':' && s[i+1] == '-' {
			return i
		}
	}
	return -1
}
