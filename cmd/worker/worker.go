package worker

import (
	cCfg "github.com/awaketai/crawler/config"
	cLog "github.com/awaketai/crawler/log"
	"github.com/awaketai/crawler/server"
	"github.com/go-micro/plugins/v4/registry/etcd"
	"go-micro.dev/v4/registry"
	"go.uber.org/zap"
)

func Run() error {
	logger, err := cLog.TomLog()
	if err != nil {
		return err
	}
	cfg, err := cCfg.GetCfg()
	if err != nil {
		return err
	}
	logger.Info("worker start....")
	var sconfig cCfg.ServerConfig
	if err := cfg.Get("WorkerServer").Scan(&sconfig); err != nil {
		logger.Error("get GRPC Server config failed", zap.Error(err))
		return err
	}
	logger.Sugar().Debugf("grpc worker server config,%+v", sconfig)
	reg := etcd.NewRegistry(registry.Addrs(sconfig.RegistryAddress))
	go server.RunHTTPServer(sconfig)

	server.RunGRPCServer(logger, sconfig, reg)
	return nil
}
