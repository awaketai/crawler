package middleware

import (
	"context"

	"go-micro.dev/v4/server"
	"go.uber.org/zap"
)

func LogWrapper(log *zap.Logger) server.HandlerWrapper {
	return func(hf server.HandlerFunc) server.HandlerFunc {
		return func(ctx context.Context, req server.Request, rsp interface{}) error {
			log.Info("receive request",
				zap.String("method", req.Method()),
				zap.String("service", req.Service()),
				zap.Reflect("request params", req.Body()),
			)
			err := hf(ctx, req, rsp)
			return err
		}
	}
}
