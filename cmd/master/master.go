package master

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/awaketai/crawler/config"
	cCfg "github.com/awaketai/crawler/config"
	cLog "github.com/awaketai/crawler/log"
	leaderMaster "github.com/awaketai/crawler/master"
	"github.com/awaketai/crawler/server"
	"github.com/go-micro/plugins/v4/registry/etcd"
	"github.com/spf13/cobra"
	"go-micro.dev/v4/registry"
	"go.uber.org/zap"
)

func init() {
	MasterCmd.Flags().StringVar(
		&masterID, "id", "1", "set master id")
	MasterCmd.Flags().StringVar(
		&HTTPListenAddress, "http", ":8081", "set HTTP listen address")

	MasterCmd.Flags().StringVar(
		&GRPCListenAddress, "grpc", ":9091", "set GRPC listen address")
}

var (
	masterID          string
	HTTPListenAddress string
	GRPCListenAddress string
)

var MasterCmd = &cobra.Command{
	Use:   "master",
	Short: "run master service",
	Long:  "run master service",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		Run()
	},
}

func Run() {
	if err := useCommanConfig(); err != nil {
		panic(err)
	}
}

func userFileConfig() error {
	logger, err := cLog.TomLog()
	if err != nil {
		return err
	}
	cfg, err := config.GetCfg()
	if err != nil {
		return err
	}
	logger.Info("master start....")
	var masterCfgs = []cCfg.ServerConfig{}
	if err := cfg.Get("MasterServer").Scan(&masterCfgs); err != nil {
		logger.Error("get GRPC Server config failed-1", zap.Error(err))
		return err
	}
	for k, sconfig := range masterCfgs {
		// if err := cfg.Get("MasterServer").Scan(&sconfig); err != nil {
		// 	logger.Error("get GRPC Server config failed", zap.Error(err))
		// }
		logger.Sugar().Debugf("grpc master server config-%d,%+v", k, sconfig)
		go runMasterServer(sconfig, logger)
	}

	quitCh := make(chan os.Signal, 1)
	signal.Notify(quitCh, os.Interrupt, syscall.SIGTERM)
	<-quitCh
	return nil
}

func useCommanConfig() error {
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
		return err
	}
	// 赋值为命令行中接收到的值
	sconfig.ID = masterID
	sconfig.HTTPListenAddress = HTTPListenAddress
	sconfig.GRPCListenAddress = GRPCListenAddress
	logger.Sugar().Debugf("grpc master server config,%+v", sconfig)
	runMasterServer(sconfig, logger)
	return nil
}

func runMasterServer(sconfig cCfg.ServerConfig, logger *zap.Logger) {
	reg := etcd.NewRegistry(registry.Addrs(sconfig.RegistryAddress))

	// leader选举
	leaderMaster.NewMaster(
		sconfig.ID,
		leaderMaster.WithLogger(logger),
		leaderMaster.WithGRPCAddress(sconfig.GRPCListenAddress),
		leaderMaster.WithRegistryURL(sconfig.RegistryAddress),
		leaderMaster.WithRegistry(reg),
	)
	go server.RunHTTPServer(sconfig)

	server.RunGRPCServer(logger, sconfig, reg)
}
