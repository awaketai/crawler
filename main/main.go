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

func task2() {
	plugin := log2.NewStdoutPlugin(zapcore.InfoLevel)
	logger := log2.NewLogger(plugin)
	var seeds = make([]*collect.Task, 0, 1000)
	cookie := `viewed="27043167_25863515_10746113_2243615_36667173_1007305_1091086"; __utma=30149280.1138703939.1688435343.1733118222.1733122303.10; ll="108288"; bid=p4zwdHrVY7w; __utmz=30149280.1729597487.8.2.utmcsr=ruanyifeng.com|utmccn=(referral)|utmcmd=referral|utmcct=/; _pk_id.100001.8cb4=18c04f5fb62d2e52.1733118221.; __utmc=30149280; dbcl2="285159894:dMkA02qtf50"; ck=tQmt; push_noty_num=0; push_doumail_num=0; __utmv=30149280.28515; __yadk_uid=3D5K4bndWlX7TLf8CjyAjVV5aB26MFa8; loc-last-index-location-id="108288"; _vwo_uuid_v2=DA5C0F35C5141ECEE7520D43DF2106264|8d200da2a9f789409ca0ce01e00d2789; frodotk_db="4a184671f7672f9cde48d355e6358ed4"; _pk_ses.100001.8cb4=1; __utmb=30149280.26.9.1733123639802; __utmt=1`
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
			Name:   "find_douban_sun_room",
			Cookie: cookie,
		},

		Fetcher: f,
	})
	s := engine.NewCrawler(
		engine.WithFetcher(f),
		engine.WithLogger(logger),
		engine.WithSeeds(seeds),
		engine.WithWorkCount(2),
		engine.WithScheduler(engine.NewSchedule()),
	)
	s.Run()
}
