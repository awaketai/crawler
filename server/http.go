package server

import (
	"context"
	"net/http"

	cCfg "github.com/awaketai/crawler/config"
	pb "github.com/awaketai/crawler/goout/common"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"go.uber.org/zap"
	grpc2 "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func RunHTTPServer(cfg cCfg.ServerConfig) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	defer cancel()

	mux := runtime.NewServeMux()
	opts := []grpc2.DialOption{
		grpc2.WithTransportCredentials(insecure.NewCredentials()),
	}
	//
	if cfg.IsMaster {
		if err := pb.RegisterCrawlerMasterGwFromEndpoint(ctx, mux, cfg.GRPCListenAddress, opts); err != nil {
			zap.L().Fatal("Register crawler http server endpoint failed")
		}
	}

	if err := pb.RegisterGreeterGwFromEndpoint(ctx, mux, cfg.GRPCListenAddress, opts); err != nil {
		zap.L().Fatal("Register hello http server endpoint failed")
	}
	zap.S().Debugf("start master http server listening on %v proxy to grpc server;%v", cfg.HTTPListenAddress, cfg.GRPCListenAddress)
	if err := http.ListenAndServe(cfg.HTTPListenAddress, mux); err != nil {
		zap.L().Fatal("http listenAndServe failed")
	}
}
