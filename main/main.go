package main

import (
	"time"

	"github.com/awaketai/crawler/collect"
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
	seeds = append(seeds, &collect.Task{
		Propety: collect.Propety{
			Name: "js_find_douban_sun_room",
		},
		Fetcher: f,
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
