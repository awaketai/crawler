package engine

import (
	"github.com/awaketai/crawler/collect"
	"go.uber.org/zap"
)

type Option func(opt *options)

type options struct{
	WorkCount int
	Fetcher collect.Fetcher
	Logger *zap.Logger
	Seeds []*collect.Request
}

var defaultOptions = options{
	Logger: zap.NewNop(),
}

func WithLogger(logger *zap.Logger) Option {
	return func(opt *options) {
		opt.Logger = logger
	}
}

func WithFetcher(fetcher collect.Fetcher) Option {
	return func(opt *options) {
		opt.Fetcher = fetcher
	}
}

func WithWorkCount(workCount int) Option {
	return func(opt *options) {
		opt.WorkCount = workCount
	}
}

func WithSeeds(seed []*collect.Request) Option {
	return func(opt *options) {
		opt.Seeds = seed
	}
}
