package utils

import (
    "fmt"
    "strings"

    "go.uber.org/zap"
    "go.uber.org/zap"
)

// NewLogger creates a new structured logger
func NewLogger(level string) (*zap.Logger, error) {
    var config zap.Config

    switch strings.ToLower(level) {
    case "debug":
        config = zap.NewDevelopmentConfig()
        config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
    case "info":
        config = zap.NewProductionConfig()
        config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
    case "warn":
        config = zap.NewProductionConfig()
        config.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
    case "error":
        config = zap.NewProductionConfig()
        config.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
    default:
        return nil, fmt.Errorf("invalid log level: %s", level)
    }

    // Configure output format
    config.Encoding = "console"
    config.EncoderConfig.TimeKey = "time"
    config.EncoderConfig.LevelKey = "level"
    config.EncoderConfig.MessageKey = "message"
    config.EncoderConfig.EncodeTime = zap.ISO8601TimeEncoder
    config.EncoderConfig.EncodeLevel = zap.CapitalColorLevelEncoder

    logger, err := config.Build()
    if err != nil {
        return nil, fmt.Errorf("failed to build logger: %w", err)
    }

    return logger, nil
}