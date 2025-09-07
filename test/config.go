package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
)

// Config represents the test configuration
type Config struct {
	Variants     VariantConfig     `toml:"variants"`
	TestSettings TestSettingsConfig `toml:"test_settings"`
	Paths        PathsConfig       `toml:"paths"`
}

// VariantConfig contains the list of variants to test
type VariantConfig struct {
	Names []string `toml:"names"`
}

// TestSettingsConfig contains test execution settings
type TestSettingsConfig struct {
	MaxConcurrent  int `toml:"max_concurrent"`
	TimeoutSeconds int `toml:"timeout_seconds"`
}

// PathsConfig contains directory paths
type PathsConfig struct {
	BinDir     string `toml:"bin_dir"`
	TempDir    string `toml:"temp_dir"`
	ReportsDir string `toml:"reports_dir"`
}

// LoadConfig loads the configuration from config.toml
func LoadConfig() (*Config, error) {
	// Get the directory where the executable is located
	execDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return nil, fmt.Errorf("failed to get executable directory: %v", err)
	}

	// Look for config.toml in the same directory as the executable, or parent directory
	configPaths := []string{
		filepath.Join(execDir, "config.toml"),
		filepath.Join(filepath.Dir(execDir), "config.toml"),
		filepath.Join(execDir, "..", "config.toml"),
		"config.toml",
	}

	var configFile string
	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			configFile = path
			break
		}
	}

	if configFile == "" {
		return nil, fmt.Errorf("config.toml not found in any of these locations: %v", configPaths)
	}

	var config Config
	if _, err := toml.DecodeFile(configFile, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %v", configFile, err)
	}

	return &config, nil
}

// GetTimeout returns the configured timeout as a time.Duration
func (c *Config) GetTimeout() time.Duration {
	return time.Duration(c.TestSettings.TimeoutSeconds) * time.Second
}