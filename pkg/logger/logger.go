package logger

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/develoop/taxi-platform/configs"
)

func New(config configs.LoggerConfig) (*zap.Logger, error) {
	var zapConfig zap.Config
	if config.Development {
		zapConfig = zap.NewDevelopmentConfig()
	} else {
		zapConfig = zap.NewProductionConfig()
	}

	level, err := zapcore.ParseLevel(config.Level)
	if err != nil {
		return nil, fmt.Errorf("parse log level: %w", err)
	}

	zapConfig.Level = zap.NewAtomicLevelAt(level)
	zapConfig.Encoding = config.Encoding
	zapConfig.DisableStacktrace = !config.Development

	logger, err := zapConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("build logger: %w", err)
	}

	return logger, nil
}
