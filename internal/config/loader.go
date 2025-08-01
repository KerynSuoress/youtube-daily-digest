package config

import (
	"fmt"

	"youtube-summarizer/pkg/types"

	"github.com/spf13/viper"
)

// Loader handles configuration loading from files and environment
type Loader struct {
	configPath string
	envPath    string
}

// NewLoader creates a new configuration loader
func NewLoader(configPath, envPath string) *Loader {
	return &Loader{
		configPath: configPath,
		envPath:    envPath,
	}
}

// Load loads configuration from config file only (single source of truth)
func (l *Loader) Load() (*types.Config, error) {
	// Start with default configuration
	config := DefaultConfig()

	// Set up viper to read from config file only
	viper.SetConfigFile(l.configPath)
	viper.SetConfigType("yaml")

	// Read config file (required for proper operation)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found - use defaults
			// This is acceptable for testing but log a warning
		} else {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Unmarshal into our config struct
	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate the configuration
	if err := Validate(config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return config, nil
}

// Removed LoadFromEnvironment - config.yaml is the single source of truth

// SaveConfig saves configuration to the specified file (for UI integration)
func (l *Loader) SaveConfig(config *types.Config) error {
	viper.Set("app", config.App)
	viper.Set("youtube", config.YouTube)
	viper.Set("processing", config.Processing)
	viper.Set("email", config.Email)
	viper.Set("ai", config.AI)

	return viper.WriteConfigAs(l.configPath)
}

// bindEnvVars manually binds environment variables to viper keys
// Only API keys and secrets should be environment variables
// All configuration should come from config.yaml for UI compatibility
func bindEnvVars() {
	// No configuration environment variables - config.yaml is the single source of truth
	// API keys remain as environment variables since they're secrets
}
