package engine

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/awaketai/crawler/collect"
	"github.com/awaketai/crawler/collector"
	"go.uber.org/zap"
)

type Crawler struct {
	out         chan collect.ParseResult
	Visited     map[string]bool
	VisitedLock sync.Mutex
	// failures 失败尝试队列
	failures    map[string]*collect.Request
	failureLock sync.Mutex
	options
}

func NewCrawler(opts ...Option) *Crawler {
	options := defaultOptions
	for _, opt := range opts {
		opt(&options)
	}
	c := &Crawler{
		out:         make(chan collect.ParseResult),
		Visited:     map[string]bool{},
		VisitedLock: sync.Mutex{},
		failures:    map[string]*collect.Request{},
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
	var reqs = make([]*collect.Request, 0, len(c.Seeds))
	for _, seed := range c.Seeds {
		// 获取初始化任务
		task := Store.hash[seed.Name]
		if task == nil {
			c.Logger.Error("current seed task nil", zap.String("seed_name", seed.Name))
			continue
		}
		task.Fetcher = seed.Fetcher
		task.Storage = seed.Storage
		task.Logger = c.Logger
		task.Limit = seed.Limit
		rootReqs, err := task.Rule.Root()
		if err != nil {
			c.Logger.Error("task rule root err:", zap.String("seed_name", seed.Name), zap.Error(err))
			continue
		}
		for _, req := range rootReqs {
			req.Task = task
		}
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
		if !r.Task.Reload && c.HasVisited(r) {
			c.Logger.Error("requested has visited", zap.String("url", r.Url))
			continue
		}
		c.StoreVisited(r)
		var (
			body []byte
			err  error
		)
		if r.Test && len(r.TestBody) > 0 {
			body = r.TestBody
		} else {
			c.Logger.Info("fetching", zap.String("url", r.Url))
			body, err = r.Fetch(context.Background())
			if err != nil {
				c.Logger.Error("fetch failed", zap.Error(err))
				c.SetFailure(r)
				continue
			}
		}
		strBody := string(body)
		if strings.Contains(strBody, "你访问豆瓣的方式有点像机器人程序") {
			c.Logger.Error("fetch be banned", zap.String("url", r.Url))
			c.SetFailure(r)
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
		// 获取当前任务对应的规则
		rule := r.Task.Rule.Trunk[r.RuleName]
		result, _ := rule.ParseFunc(&collect.CrawlerContext{
			Body: body,
			Req:  r,
		})
		// 新任务加入队列中
		if len(result.Requests) > 0 {
			go c.scheduler.Push(result.Requests...)
		}

		c.out <- result
	}
}

// HandleResult 如何没有数据，直接循环会导致dead lock
// 此处使用select {}规避
func (c *Crawler) HandleResult() {
	for {
		select {
		case res := <-c.out:
			for _, item := range res.Items {
				// 数据存储
				c.Logger.Info("item:", zap.Any("item", item))
				switch d := item.(type) {
				case *collector.DataCell:
					name := d.GetTaskName()
					task := Store.hash[name]
					c.Logger.Info("data cell:", zap.String("task", name), zap.Any("data", d))
					err := task.Storage.Save(d)
					if err != nil {
						c.Logger.Error("storage save failed", zap.Error(err))
					}
				}
			}
			// 防止cpu空转，避免忙等
		case <-time.After(10 * time.Second):
			fmt.Println("c.out no data")
		}
	}
}

func (c *Crawler) HasVisited(r *collect.Request) bool {
	c.VisitedLock.Lock()
	defer c.VisitedLock.Unlock()
	unique := r.Unique()
	return c.Visited[unique]
}

func (c *Crawler) StoreVisited(reqs ...*collect.Request) {
	c.VisitedLock.Lock()
	defer c.VisitedLock.Unlock()
	for _, v := range reqs {
		uniqie := v.Unique()
		c.Visited[uniqie] = true
	}
}

func (c *Crawler) SetFailure(req *collect.Request) {
	if !req.Task.Reload {
		c.VisitedLock.Lock()
		unique := req.Unique()
		delete(c.Visited, unique)
		c.VisitedLock.Unlock()
	}
	c.failureLock.Lock()
	defer c.failureLock.Unlock()
	if _, ok := c.failures[req.Unique()]; !ok {
		c.failures[req.Unique()] = req
		c.scheduler.Push(req)
	}
	// todo 失败两次，加入失败队列中

}
