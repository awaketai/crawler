package engine

import (
	"github.com/awaketai/crawler/collect"
	"github.com/awaketai/crawler/parse/doubangroup"
)


func init() {
	Store.Add(doubangroup.DouBanGroupTask)
}

// Store 全局爬虫任务实例
var Store = &CrawlerStore{
	list: []*collect.Task{},
	hash: map[string]*collect.Task{},
}

type CrawlerStore struct{
	list []*collect.Task
	hash map[string]*collect.Task
}

func (c *CrawlerStore) Add(task *collect.Task){
	c.hash[task.Name] = task
	c.list = append(c.list, task)
}

