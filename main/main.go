package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/awaketai/crawler/cmd"
	"github.com/awaketai/crawler/collect"
	"github.com/awaketai/crawler/collector"
	"github.com/awaketai/crawler/collector/sqlstorage"
	"github.com/awaketai/crawler/engine"
	pb "github.com/awaketai/crawler/goout/hello"
	"github.com/awaketai/crawler/limiter"
	log2 "github.com/awaketai/crawler/log"
	"github.com/awaketai/crawler/middleware"
	"github.com/awaketai/crawler/proxy"
	"github.com/awaketai/crawler/service"
	grpccli "github.com/go-micro/plugins/v4/client/grpc"
	"github.com/go-micro/plugins/v4/config/encoder/toml"
	etcdReg "github.com/go-micro/plugins/v4/registry/etcd"
	gs "github.com/go-micro/plugins/v4/server/grpc"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"
	"go-micro.dev/v4"
	"go-micro.dev/v4/config"
	"go-micro.dev/v4/config/reader"
	"go-micro.dev/v4/config/reader/json"
	"go-micro.dev/v4/config/source"
	"go-micro.dev/v4/config/source/file"
	"go-micro.dev/v4/registry"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatalf("run err:%v", err)
	}
}

func main5() {
	enc := toml.NewEncoder()
	cfg, err := config.NewConfig(
		config.WithReader(json.NewReader(reader.WithEncoder(enc))),
	)
	if err != nil {
		panic(err)
	}
	err = cfg.Load(file.NewSource(
		file.WithPath("/Users/ashertai/wwwroot/distribute_crawler/config.toml"),
		source.WithEncoder(enc),
	))
	if err != nil {
		panic(err)
	}

	logText := cfg.Get("logLevel").String("INFO")
	logLevel, err := zapcore.ParseLevel(logText)
	if err != nil {
		panic(err)
	}
	plugin := log2.NewStdoutPlugin(logLevel)
	logger := log2.NewLogger(plugin)
	logger.Info("log inited")
	zap.ReplaceGlobals(logger)
	// server config init
	var serverCfg ServerConfig
	err = cfg.Get("GRPCServer").Scan(&serverCfg)
	if err != nil {
		panic(err)
	}
	logger.Sugar().Debugf("serverCfg:%+v", serverCfg)
	multiWorkDouban(cfg, logger)

	RunGRPCServer(logger, cfg)
}

type ServerConfig struct {
	GRPCListenAddress string
	HTTPListenAddress string
	ID                string
	RegistryAddress   string
	RegisterTTL       int
	RegisterInterval  int
	Name              string
	ClientTimeOut     int
}

func multiWorkDouban(cfg config.Config, logger *zap.Logger) {
	fetcher := getFetcher(cfg, logger)
	storage := getStorage(cfg, logger)
	tasks, err := getSeeds(cfg, logger, fetcher, storage)
	if err != nil {
		panic("get seeds err:" + err.Error())
	}

	s := engine.NewCrawler(
		engine.WithFetcher(fetcher),
		engine.WithLogger(logger),
		engine.WithTasks(tasks),
		engine.WithWorkCount(5),
		engine.WithScheduler(engine.NewSchedule()),
	)
	go s.Run()
}

func getProxy(cfg config.Config) (proxy.ProxyFunc, error) {
	proxyURLs := cfg.Get("fetcher", "proxy").StringSlice([]string{})
	p, err := proxy.RoundRobinProxySwitcher(proxyURLs...)
	if err != nil {
		panic("RoundRobinProxySwitcher err:" + err.Error())
	}

	return p, err
}

func getFetcher(cfg config.Config, logger *zap.Logger) collect.Fetcher {
	p, err := getProxy(cfg)
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

func getStorage(cfg config.Config, logger *zap.Logger) collector.Storager {
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

func RunGRPCServer(logger *zap.Logger, cfg config.Config) {
	// go HandleHTTP(cfg)
	reg := etcdReg.NewRegistry(
		registry.Addrs(cfg.Get("RegistryAddress").String(":2379")),
	)
	address := cfg.Get("GRPCServer", "GRPCListenAddress").String("localhost:50051")
	name := cfg.Get("GRPCServer", "Name").String("go.micro.server.worker")
	svc := micro.NewService(
		micro.Server(gs.NewServer()),
		micro.Address(address),
		micro.Name(name),
		micro.Registry(reg),
		micro.WrapHandler(middleware.LogWrapper(logger)),
	)
	svc.Init()
	pb.RegisterGreeterHandler(svc.Server(), new(service.Greet))
	// ticker := time.NewTicker(5 * time.Second)
	// go func() {
	// 	for range ticker.C {
	// 		reqGRPC(cfg)
	// 	}
	// }()
	if err := svc.Run(); err != nil {
		fmt.Println(err)
	}
}

func HandleHTTP(cfg config.Config) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	grpcAddr := cfg.Get("GRPCServer", "GRPCListenAddress").String("localhost:50051")
	httpAddr := cfg.Get("GRPCServer", "HTTPListenAddress").String(":8080")
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	err := pb.RegisterGreeterGwFromEndpoint(ctx, mux, grpcAddr, opts)
	if err != nil {
		log.Fatalf("register http failed:%v", err)
	}

	log.Fatal(http.ListenAndServe(httpAddr, mux))
}

func generateRandomString(minLen, maxLen int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	rand.Seed(time.Now().UnixNano())
	length := rand.Intn(maxLen-minLen+1) + minLen
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func reqGRPC(cfg config.Config) {
	reg := etcdReg.NewRegistry(
		registry.Addrs(cfg.Get("RegistryAddress").String(":2379")),
	)
	svc := micro.NewService(
		micro.Registry(reg),
		micro.Client(grpccli.NewClient()),
	)
	svc.Init()

	cli := pb.NewGreeterService("go.micro.server.worker", svc.Client())
	// make grpc request
	// generate a random string

	rsp, err := cli.Hello(context.Background(), &pb.Request{
		Name: generateRandomString(3, 12),
	})
	if err != nil {
		fmt.Println("grpc req err:", err)
	}
	fmt.Println("grpc resp:", rsp.Greeting)
}

func getSeeds(cfg config.Config, logger *zap.Logger, fetcher collect.Fetcher, storage collector.Storager) ([]*collect.Task, error) {
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

func cobraTest() {
	var echoTimes int
	var cmdPrint = &cobra.Command{
		Use:   "c [string to print]",
		Short: "Print anything to the screen",
		Long:  `print is for printing anything back to the screen.`,
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Print:", strings.Join(args, " "))
		},
	}
	var cmdEcho = &cobra.Command{
		Use:   "echo [string to echo]",
		Short: "Echo anything to the screen",
		Long:  `echo is for echoing anything back.Echo works a lot like print, except it has a child command.`,
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Echo: " + strings.Join(args, " "))
		},
	}
	var cmdTimes = &cobra.Command{
		Use:   "times [string to echo]",
		Short: "Echo anything to the screen more times",
		Long:  `echo things multiple times back to the user by providinga count and a string.`,
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			for i := 0; i < echoTimes; i++ {
				fmt.Println("Echo: " + strings.Join(args, " "))
			}
		}}
	cmdTimes.Flags().IntVarP(&echoTimes, "times", "t", 1, "times echo the input")
	var rootCmd = &cobra.Command{
		Use: "app",
	}
	rootCmd.AddCommand(cmdPrint, cmdEcho)
	cmdEcho.AddCommand(cmdTimes)
	rootCmd.Execute()
}
