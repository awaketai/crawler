package engine

import (
	"github.com/awaketai/crawler/collect"
	"go.uber.org/zap"
)

type Crawler struct {
	out chan collect.ParseResult
	options
}

func NewCrawler(opts ...Option) *Crawler {
	options := defaultOptions
	for _, opt := range opts {
		opt(&options)
	}
	out := make(chan collect.ParseResult)
	c := &Crawler{}
	c.options = options
	c.out = out

	return c
}

func (c *Crawler) Run() {
	go c.Schedule()
	for i := 0; i < c.WorkCount; i++ {
		go c.CreateWork()
	}
	c.HandleResult()
}

func (c *Crawler) Schedule() {
	var reqs []*collect.Request
	for _, seed := range c.Seeds {
		seed.RootReq.Task = seed
		seed.RootReq.Url = seed.Url
		reqs = append(reqs, seed.RootReq)
	}

	go c.scheduler.Schedule()
	go c.scheduler.Push(reqs...)
}

func (c *Crawler) CreateWork() {
	for {
		r := c.scheduler.Pull()
		if err := r.Check(); err != nil {
			c.Logger.Error("check failed", zap.Error(err))
			continue
		}
		body, err := r.Task.Fetcher.Get(r)
		if err != nil {
			c.Logger.Error("fetch failed", zap.Error(err))
			continue
		}
		if len(body) < 6000 {
			c.Logger.Error("fetch body too short",
				zap.Int("length", len(body)),
				zap.String("url", r.Url),
				zap.String("body", string(body)),
			)
			continue
		}
		result := r.ParseFunc(body, r)
		if len(result.Requests) > 0 {
			go c.scheduler.Push(result.Requests...)
		}

		c.out <- result
	}
}

func (c *Crawler) HandleResult() {
	for res := range c.out {
		for _, item := range res.Items {
			// to do store
			c.Logger.Sugar().Info("get res:", item)
		}
	}

}
