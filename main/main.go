package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/awaketai/crawler/cmd"
	"github.com/awaketai/crawler/cmd/worker"
	"github.com/awaketai/crawler/engine"
	pb "github.com/awaketai/crawler/goout/common"
	log2 "github.com/awaketai/crawler/log"
	"github.com/awaketai/crawler/middleware"
	"github.com/awaketai/crawler/service"
	grpccli "github.com/go-micro/plugins/v4/client/grpc"
	"github.com/go-micro/plugins/v4/config/encoder/toml"
	etcdReg "github.com/go-micro/plugins/v4/registry/etcd"
	gs "github.com/go-micro/plugins/v4/server/grpc"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"go-micro.dev/v4"
	"go-micro.dev/v4/config"
	"go-micro.dev/v4/config/reader"
	"go-micro.dev/v4/config/reader/json"
	"go-micro.dev/v4/config/source"
	"go-micro.dev/v4/config/source/file"
	"go-micro.dev/v4/registry"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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
	fetcher := worker.GetFetcher(cfg, logger)
	storage := worker.GetStorage(cfg, logger)
	tasks, err := worker.GetSeeds(cfg, logger, fetcher, storage)
	if err != nil {
		panic("get seeds err:" + err.Error())
	}

	s, err := engine.NewCrawler(
		engine.WithFetcher(fetcher),
		engine.WithLogger(logger),
		engine.WithTasks(tasks),
		engine.WithWorkCount(5),
		engine.WithScheduler(engine.NewSchedule()),
	)
	if err != nil {
		log.Fatal(err)
	}
	go s.Run("1", true)
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
