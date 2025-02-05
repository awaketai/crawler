package server

import (
	"time"

	cCfg "github.com/awaketai/crawler/config"
	"github.com/awaketai/crawler/goout/common"
	pb "github.com/awaketai/crawler/goout/common"
	"github.com/awaketai/crawler/master"
	"github.com/awaketai/crawler/middleware"
	"github.com/awaketai/crawler/service"
	"github.com/go-micro/plugins/v4/server/grpc"
	"go-micro.dev/v4"
	"go-micro.dev/v4/client"
	"go-micro.dev/v4/registry"
	"go-micro.dev/v4/server"
	"go.uber.org/zap"
)

func RunGRPCServer(logger *zap.Logger, cfg cCfg.ServerConfig, reg registry.Registry, masterSrv *master.Master) {
	svc := micro.NewService(
		micro.Server(grpc.NewServer(
			server.Id(cfg.ID),
		)),
		micro.Address(cfg.GRPCListenAddress),
		micro.Registry(reg),
		micro.RegisterTTL(time.Duration(cfg.RegisterTTL)*time.Second),
		micro.RegisterInterval(time.Duration(cfg.RegisterInterval)*time.Second),
		micro.WrapHandler(middleware.LogWrapper(logger)),
		micro.Name(cfg.Name),
	)

	// 设置micro 客户端默认超时时间为10秒钟
	if err := svc.Client().Init(client.RequestTimeout(time.Duration(cfg.ClientTimeOut) * time.Second)); err != nil {
		logger.Sugar().Error("micro client init error. ", zap.String("error:", err.Error()))

		return
	}

	svc.Init()
	// 注册处理函数
	if cfg.IsMaster {
		err := common.RegisterCrawlerMasterHandler(svc.Server(), masterSrv)
		if err != nil {
			logger.Fatal("register master failed")
		}
	}
	if err := pb.RegisterGreeterHandler(svc.Server(), new(service.Greet)); err != nil {
		logger.Fatal("register handler failed")
	}

	if err := svc.Run(); err != nil {
		logger.Fatal("grpc server stop")
	}
}
