package log

import (
	cfgPb "github.com/awaketai/crawler/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TomLog() (*zap.Logger, error) {
	cfg, err := cfgPb.GetCfg()
	if err != nil {
		return nil, err
	}

	// log
	logText := cfg.Get("logLevel").String("INFO")
	logLevel, err := zapcore.ParseLevel(logText)
	if err != nil {
		return nil, err
	}
	plugin := NewStdoutPlugin(logLevel)
	logger := NewLogger(plugin)
	logger.Info("log init end")

	// set zap global logger
	zap.ReplaceGlobals(logger)

	return logger, nil
}
