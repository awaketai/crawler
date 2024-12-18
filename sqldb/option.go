package sqldb

import "go.uber.org/zap"

type options struct {
	logger *zap.Logger
	dsn    string
}

var defaultOptions = options{
	logger: zap.NewNop(),
}

type Option func(opts *options)

func WithLogger(logger *zap.Logger) Option {
	return func(opts *options) {
		opts.logger = logger
	}
}

func WithDSN(dsn string) Option {
	return func(opts *options) {
		opts.dsn = dsn
	}
}
