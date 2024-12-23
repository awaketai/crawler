package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/awaketai/crawler/collect"
	"github.com/awaketai/crawler/collector"
	"github.com/awaketai/crawler/collector/sqlstorage"
	"github.com/awaketai/crawler/engine"
	pb "github.com/awaketai/crawler/goout/hello"
	"github.com/awaketai/crawler/limiter"
	log2 "github.com/awaketai/crawler/log"
	"github.com/awaketai/crawler/proxy"
	"github.com/awaketai/crawler/service"
	grpccli "github.com/go-micro/plugins/v4/client/grpc"
	etcdReg "github.com/go-micro/plugins/v4/registry/etcd"
	gs "github.com/go-micro/plugins/v4/server/grpc"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"go-micro.dev/v4"
	"go-micro.dev/v4/registry"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
)

func main() {
	// multiWorkDouban()
	HelloGTPC()
}

func multiWorkDouban() {
	plugin := log2.NewStdoutPlugin(zapcore.InfoLevel)
	logger := log2.NewLogger(plugin)
	var seeds = make([]*collect.Task, 0, 1000)
	proxyURLs := []string{"http://127.0.0.1:4780"}
	p, err := proxy.RoundRobinProxySwitcher(proxyURLs...)
	if err != nil {
		logger.Error("RoundRobinProxySwitcher err:", zap.Error(err))
	}
	var f collect.Fetcher = &collect.BrowserFetch{
		Timeout: 10 * time.Second,
		Logger:  logger,
		Proxy:   p,
	}
	var storage collector.Storager
	storage, err = sqlstorage.NewSqlStore(
		sqlstorage.WithDSN("root:admin123@tcp(127.0.0.1:3306)/test?charset=utf8"),
		sqlstorage.WithLogger(logger.Named("sqlDB")),
		sqlstorage.WithBatchCount(1),
	)
	if err != nil {
		logger.Error("create sql storage failed", zap.Error(err))
		return
	}
	// 限速
	// 2秒钟一个
	secondLimit := rate.NewLimiter(limiter.Per(1, 2*time.Second), 1)
	// 60秒20个
	minuteLimie := rate.NewLimiter(limiter.Per(20, 1*time.Minute), 20)
	multiLimiter := limiter.NewMultiLimit(secondLimit, minuteLimie)
	seeds = append(seeds, &collect.Task{
		Propety: collect.Propety{
			Name: "douban_book_list",
		},
		Fetcher: f,
		Storage: storage,
		Limit:   multiLimiter,
	})

	s := engine.NewCrawler(
		engine.WithFetcher(f),
		engine.WithLogger(logger),
		engine.WithSeeds(seeds),
		engine.WithWorkCount(5),
		engine.WithScheduler(engine.NewSchedule()),
	)
	s.Run()
}

func HelloGTPC() {
	go HandleHTTP()
	reg := etcdReg.NewRegistry(
		registry.Addrs(":2379"),
	)
	svc := micro.NewService(
		micro.Server(gs.NewServer()),
		micro.Address("localhost:50051"),
		micro.Name("go.micro.server.worker"),
		micro.Registry(reg),
	)
	svc.Init()
	pb.RegisterGreeterHandler(svc.Server(), new(service.Greet))
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for range ticker.C {
			reqGRPC()
		}
	}()
	if err := svc.Run(); err != nil {
		fmt.Println(err)
	}
}

func HandleHTTP() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	err := pb.RegisterGreeterGwFromEndpoint(ctx, mux, "localhost:50051", opts)
	if err != nil {
		log.Fatalf("register http failed:%v", err)
	}

	log.Fatal(http.ListenAndServe(":8080", mux))
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

func reqGRPC() {
	reg := etcdReg.NewRegistry(
		registry.Addrs(":2379"),
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
