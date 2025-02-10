package worker

import (
	"fmt"
	"strconv"
	"time"

	"github.com/awaketai/crawler/collect"
	"github.com/awaketai/crawler/collector"
	"github.com/awaketai/crawler/collector/sqlstorage"
	cCfg "github.com/awaketai/crawler/config"
	"github.com/awaketai/crawler/engine"
	"github.com/awaketai/crawler/extensions"
	"github.com/awaketai/crawler/limiter"
	cLog "github.com/awaketai/crawler/log"
	"github.com/awaketai/crawler/proxy"
	"github.com/awaketai/crawler/server"
	"github.com/go-micro/plugins/v4/registry/etcd"
	"github.com/spf13/cobra"
	"go-micro.dev/v4/config"
	"go-micro.dev/v4/registry"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

func init() {
	WorkerCmd.Flags().StringVar(
		&HTTPListenAddress, "http", ":3081", "set HTTP listen address")

	WorkerCmd.Flags().StringVar(
		&GRPCListenAddress, "grpc", ":4091", "set GRPC listen address")
	WorkerCmd.Flags().StringVar(
		&workerID, "id", "", "set worker id")

	WorkerCmd.Flags().StringVar(
		&podIP, "podip", "", "set worker id")
	WorkerCmd.Flags().BoolVar(
		&cluster, "cluster", true, "run mode")
}

var (
	HTTPListenAddress string
	GRPCListenAddress string
	cluster           bool
	workerID          string
	podIP             string
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
	sconfig.ID = workerID
	sconfig.HTTPListenAddress = HTTPListenAddress
	sconfig.GRPCListenAddress = GRPCListenAddress
	logger.Sugar().Debugf("grpc worker server config,%+v", sconfig)
	reg := etcd.NewRegistry(registry.Addrs(sconfig.RegistryAddress))
	go server.RunHTTPServer(sconfig)
	go runTaskEngine(sconfig,logger)
	server.RunGRPCServer(logger, sconfig, reg, nil)
	return nil
}

func runTaskEngine(sconfig cCfg.ServerConfig, logger *zap.Logger) {
	cfg, err := cCfg.GetCfg()
	if err != nil {
		panic(err)
	}
	fetcher := GetFetcher(cfg, logger)
	storage := GetStorage(cfg, logger)
	seeds, err := GetSeeds(cfg, logger, fetcher, storage)
	if err != nil {
		panic(err)
	}
	s, err := engine.NewCrawler(
		engine.WithFetcher(fetcher),
		engine.WithLogger(logger),
		engine.WithRegistryURL(sconfig.RegistryAddress),
		engine.WithWorkCount(5),
		engine.WithSeeds(seeds),
		engine.WithScheduler(engine.NewSchedule()),
		engine.WithStorage(storage),
	)
	if err != nil {
		panic(err)
	}
	if workerID == "" {
		if podIP != "" {
			ip := extensions.IDByIP(podIP)
			workerID = strconv.Itoa(int(ip))
		} else {
			workerID = fmt.Sprintf("%d", time.Now().UnixNano())
		}
	}
	id := sconfig.Name + "-" + workerID
	zap.S().Debug("worker id", workerID)
	go s.Run(id, cluster)
}

func GetSeeds(cfg config.Config, logger *zap.Logger, fetcher collect.Fetcher, storage collector.Storager) ([]*collect.Task, error) {
	var tcfg []collect.Options
	if err := cfg.Get("Tasks").Scan(&tcfg); err != nil {
		logger.Error("get tasks err", zap.Error(err))
		return nil, err
	}
	tasks := make([]*collect.Task, 0, len(tcfg))
	for _, v := range tcfg {
		t := collect.NewTask(
			collect.WithCookie(v.Cookie),
			collect.WithFetcher(fetcher),
			collect.WithLogger(logger),
			collect.WithName(v.Name),
			collect.WithReload(v.Reload),
			collect.WithStorage(storage),
			collect.WithUrl(v.Url),
		)
		if v.WaitTime > 0 {
			t.WaitTime = v.WaitTime
		}
		if v.MaxDepth > 0 {
			t.MaxDepth = v.MaxDepth
		}
		var limits []limiter.RateLimiter
		if len(v.LimitCfg) > 0 {
			for _, l := range v.LimitCfg {
				lm := rate.NewLimiter(limiter.Per(l.EventCount, time.Duration(l.EventDur)*time.Second), 1)
				limits = append(limits, lm)
			}
			multiLimiter := limiter.NewMultiLimit(limits...)
			t.Limit = multiLimiter
		}
		switch v.FetchType {
		case collect.BrowserFetchType:
			t.Fetcher = fetcher
		}
		tasks = append(tasks, t)
	}

	return tasks, nil
}

func GetProxy(cfg config.Config) (proxy.ProxyFunc, error) {
	proxyURLs := cfg.Get("fetcher", "proxy").StringSlice([]string{})
	p, err := proxy.RoundRobinProxySwitcher(proxyURLs...)
	if err != nil {
		panic("RoundRobinProxySwitcher err:" + err.Error())
	}

	return p, err
}

func GetFetcher(cfg config.Config, logger *zap.Logger) collect.Fetcher {
	p, err := GetProxy(cfg)
	if err != nil {
		panic("getProxy err:" + err.Error())
	}
	timeout := cfg.Get("fetcher", "timeout").Int(3000)
	fetcher := &collect.BrowserFetch{
		Timeout: time.Duration(timeout) * time.Millisecond,
		Logger:  logger,
		Proxy:   p,
	}

	return fetcher
}

func GetStorage(cfg config.Config, logger *zap.Logger) collector.Storager {
	dsn := cfg.Get("storage", "dsn").String("")
	if dsn == "" {
		panic("storage dsn is empty")
	}
	storage, err := sqlstorage.NewSqlStore(
		sqlstorage.WithDSN(dsn),
		sqlstorage.WithLogger(logger.Named("sqlDB")),
		sqlstorage.WithBatchCount(1),
	)
	if err != nil {
		panic("create sql storage failed:" + err.Error())
	}

	return storage
}
