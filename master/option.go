package master

import (
	"go-micro.dev/v4/registry"
	"go.uber.org/zap"
)

type options struct {
	logger      *zap.Logger
	registryURL string
	GRPCAddress string
	registry    registry.Registry
}

var defultOptions = options{
	logger: zap.NewNop(),
}

type Option func(*options)

func WithLogger(logger *zap.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}

func WithRegistryURL(url string) Option {
	return func(o *options) {
		o.registryURL = url
	}
}

func WithGRPCAddress(address string) Option {
	return func(o *options) {
		o.GRPCAddress = address
	}
}

func WithRegistry(reg registry.Registry) Option {
	return func(o *options) {
		o.registry = reg
	}
}
