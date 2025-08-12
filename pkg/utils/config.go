package utils

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config holds application configuration
type Config struct {
	Analyzer            AnalyzerConfig  `mapstructure:"analyzer"`
	Placement           PlacementConfig `mapstructure:"placement"`
	Logging             LoggingConfig   `mapstructure:"logging"`
	ClusterProfilesPath string          `mapstructure:"clusterProfilesPath"`
}

// AnalyzerConfig holds analyzer-specific configuration
type AnalyzerConfig struct {
	EnableAdvancedMetrics       bool          `mapstructure:"enableAdvancedMetrics"`
	NetworkLatencyThreshold     time.Duration `mapstructure:"networkLatencyThreshold"`
	ResourceEfficiencyThreshold float64       `mapstructure:"resourceEfficiencyThreshold"`
	DefaultJobTimeout           time.Duration `mapstructure:"defaultJobTimeout"`
}

// PlacementConfig holds placement-specific configuration
type PlacementConfig struct {
	MaxClusters                   int     `mapstructure:"maxClusters"`
	MinClusterUtilization         float64 `mapstructure:"minClusterUtilization"`
	MaxClusterUtilization         float64 `mapstructure:"maxClusterUtilization"`
	NetworkLatencyWeight          float64 `mapstructure:"networkLatencyWeight"`
	ResourceAffinityWeight        float64 `mapstructure:"resourceAffinityWeight"`
	CostOptimizationWeight        float64 `mapstructure:"costOptimizationWeight"`
	PerformanceOptimizationWeight float64 `mapstructure:"performanceOptimizationWeight"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// LoadConfig loads configuration from file
func LoadConfig(configPath string) (*Config, error) {
	// Set default values
	viper.SetDefault("analyzer.enableAdvancedMetrics", true)
	viper.SetDefault("analyzer.networkLatencyThreshold", "10ms")
	viper.SetDefault("analyzer.resourceEfficiencyThreshold", 0.8)
	viper.SetDefault("analyzer.defaultJobTimeout", "2h")

	viper.SetDefault("placement.maxClusters", 5)
	viper.SetDefault("placement.minClusterUtilization", 0.2)
	viper.SetDefault("placement.maxClusterUtilization", 0.8)
	viper.SetDefault("placement.networkLatencyWeight", 0.3)
	viper.SetDefault("placement.resourceAffinityWeight", 0.4)
	viper.SetDefault("placement.costOptimizationWeight", 0.2)
	viper.SetDefault("placement.performanceOptimizationWeight", 0.1)

	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "console")

	// Try to read config file
	if configPath != "" {
		viper.SetConfigFile(configPath)
		if err := viper.ReadInConfig(); err != nil {
			// Config file not found or invalid - use defaults
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return nil, fmt.Errorf("failed to read config file: %w", err)
			}
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}
