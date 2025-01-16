package master

import (
	"github.com/awaketai/crawler/config"
	cCfg "github.com/awaketai/crawler/config"
	cLog "github.com/awaketai/crawler/log"
	"github.com/awaketai/crawler/server"
	"go.uber.org/zap"
)

func Run() error {
	logger, err := cLog.TomLog()
	if err != nil {
		return err
	}
	cfg, err := config.GetCfg()
	if err != nil {
		return err
	}
	logger.Info("master start....")
	var sconfig cCfg.ServerConfig
	if err := cfg.Get("MasterServer").Scan(&sconfig); err != nil {
		logger.Error("get GRPC Server config failed", zap.Error(err))
	}
	logger.Sugar().Debugf("grpc master server config,%+v", sconfig)
	go server.RunHTTPServer(sconfig)

	server.RunGRPCServer(logger, sconfig)
	return nil
}
