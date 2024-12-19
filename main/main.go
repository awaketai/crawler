package main

import (
	"time"

	"github.com/awaketai/crawler/collect"
	"github.com/awaketai/crawler/collector"
	"github.com/awaketai/crawler/collector/sqlstorage"
	"github.com/awaketai/crawler/engine"
	log2 "github.com/awaketai/crawler/log"
	"github.com/awaketai/crawler/proxy"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	multiWorkDouban()
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
	seeds = append(seeds, &collect.Task{
		Propety: collect.Propety{
			Name: "douban_book_list",
		},
		Fetcher: f,
		Storage: storage,
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
