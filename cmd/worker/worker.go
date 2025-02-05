package worker

import (
	cCfg "github.com/awaketai/crawler/config"
	cLog "github.com/awaketai/crawler/log"
	"github.com/awaketai/crawler/server"
	"github.com/go-micro/plugins/v4/registry/etcd"
	"github.com/spf13/cobra"
	"go-micro.dev/v4/registry"
	"go.uber.org/zap"
)

func init() {
	WorkerCmd.Flags().StringVar(
		&masterID, "id", "1", "set worker id")
	WorkerCmd.Flags().StringVar(
		&HTTPListenAddress, "http", ":3081", "set HTTP listen address")

	WorkerCmd.Flags().StringVar(
		&GRPCListenAddress, "grpc", ":4091", "set GRPC listen address")
}

var (
	masterID          string
	HTTPListenAddress string
	GRPCListenAddress string
)

var WorkerCmd = &cobra.Command{
	Use:   "worker",
	Short: "run worker service",
	Long:  "run woker service",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		Run()
	},
}

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
	// 赋值为命令行中接收到的值
	sconfig.ID = masterID
	sconfig.HTTPListenAddress = HTTPListenAddress
	sconfig.GRPCListenAddress = GRPCListenAddress
	logger.Sugar().Debugf("grpc worker server config,%+v", sconfig)
	reg := etcd.NewRegistry(registry.Addrs(sconfig.RegistryAddress))
	go server.RunHTTPServer(sconfig)

	server.RunGRPCServer(logger, sconfig, reg)
	return nil
}
