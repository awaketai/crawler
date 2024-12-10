package engine

import (
	"sync"

	"github.com/awaketai/crawler/collect"
	"go.uber.org/zap"
)

type Crawler struct {
	out chan collect.ParseResult
	Visited map[string]bool
	VisitedLock sync.Mutex
	// failures 失败尝试队列
	failures map[string]*collect.Request
	failureLock sync.Mutex
	options
}

func NewCrawler(opts ...Option) *Crawler {
	options := defaultOptions
	for _, opt := range opts {
		opt(&options)
	}
	c := &Crawler{
		out: make(chan collect.ParseResult),
		Visited: map[string]bool{},
		VisitedLock: sync.Mutex{},
	}
	c.options = options
	
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
		// 获取初始化任务
		task := Store.hash[seed.Name]
		rootReqs := task.Rule.Root()
		reqs = append(reqs, rootReqs...)
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
		// 检测是否已访问过当前请求
		if c.HasVisited(r) {
			c.Logger.Error("requested has visited", zap.String("url",r.Url))
			continue
		}
		c.StoreVisited(r)
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

func (c *Crawler) HasVisited(r *collect.Request) bool {
	c.VisitedLock.Lock()
	defer c.VisitedLock.Unlock()
	unique := r.Unique()
	return c.Visited[unique]
}

func (c *Crawler) StoreVisited(reqs ...*collect.Request){
	c.VisitedLock.Lock()
	defer c.VisitedLock.Unlock()
	for _, v := range reqs {
		uniqie := v.Unique()
		c.Visited[uniqie] = true
	}
}

func (c *Crawler) SetFailure(req *collect.Request){
	if !req.Task.Reload {
		c.VisitedLock.Lock()
		unique := req.Unique()
		delete(c.Visited,unique)
		c.VisitedLock.Unlock()
	}
	c.failureLock.Lock()
	defer c.failureLock.Unlock()
	if _,ok := c.failures[req.Unique()]; !ok {
		c.failures[req.Unique()] = req
		c.scheduler.Push(req)
	}
	// todo 失败两次，加入失败队列中

}
