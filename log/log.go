package log

import (
	"io"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Plugin = zapcore.Core

func NewLogger (plugin zapcore.Core,options ...zap.Option) *zap.Logger {

	return zap.New(plugin,append(DefaultOption(),options...)...)
}

func NewPlugin(writer zapcore.WriteSyncer,enabler zapcore.LevelEnabler) Plugin {

	return zapcore.NewCore(DefaultEncoder(),writer,enabler)
}

func NewStdoutPlugin(enabler zapcore.LevelEnabler) Plugin {

	return NewPlugin(zapcore.Lock(zapcore.AddSync(os.Stdout)),enabler)
}

func NewStderrPlugin(enabler zapcore.LevelEnabler) Plugin {

	return NewPlugin(zapcore.Lock(zapcore.AddSync(os.Stderr)),enabler)
}

func NewFilePlugin(filePath string,enalber zapcore.LevelEnabler) (Plugin,io.Closer) {
	var writer = DefaultLumberjackLogger()
	writer.Filename = filePath
	
	return NewPlugin(zapcore.AddSync(writer),enalber),writer
}
