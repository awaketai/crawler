package engine

import (
	"github.com/awaketai/crawler/collect"
	"github.com/awaketai/crawler/parse/doubangroup"
	"github.com/robertkrimen/otto"
)

func init() {
	Store.Add(doubangroup.DouBanGroupTask)
	Store.AddJSTask(doubangroup.DouBanGroupJSTask)
}

// Store 全局爬虫任务实例
var Store = &CrawlerStore{
	list: []*collect.Task{},
	hash: map[string]*collect.Task{},
}

type CrawlerStore struct {
	list []*collect.Task
	hash map[string]*collect.Task
}

func (c *CrawlerStore) Add(task *collect.Task) {
	c.hash[task.Name] = task
	c.list = append(c.list, task)
}

func (c *CrawlerStore) AddJSTask(m *collect.TaskMode) {
	task := &collect.Task{
		Propety: m.Propety,
	}

	task.Rule.Root = func() ([]*collect.Request, error) {
		vm := otto.New()
		vm.Set("AddJSReqs", AddJSReqs)
		v, err := vm.Eval(m.Root)
		if err != nil {
			return nil, err
		}
		e, err := v.Export()
		if err != nil {
			return nil, err
		}
		return e.([]*collect.Request), nil
	}

	for _, r := range m.Rules {
		parseFunc := func(parse string) func(ctx *collect.CrawlerContext) (collect.ParseResult, error) {
			return func(ctx *collect.CrawlerContext) (collect.ParseResult, error) {
				vm := otto.New()
				vm.Set("ctx", ctx)
				v, err := vm.Eval(parse)
				if err != nil {
					return collect.ParseResult{}, err
				}
				e, err := v.Export()
				if err != nil {
					return collect.ParseResult{}, err
				}
				if e == nil {
					return collect.ParseResult{}, err
				}
				return e.(collect.ParseResult), err
			}
		}(r.ParseFunc)
		if task.Rule.Trunk == nil {
			task.Rule.Trunk = make(map[string]*collect.Rule, 0)
		}
		task.Rule.Trunk[r.Name] = &collect.Rule{
			ParseFunc: parseFunc,
		}
	}

	c.hash[task.Name] = task
	c.list = append(c.list, task)

}

// AddJSReqs  动态规则添加请求
func AddJSReqs(jreqs []map[string]any) []*collect.Request {
	reqs := make([]*collect.Request, 0, len(jreqs))
	for _, v := range jreqs {
		req := &collect.Request{}
		u, ok := v["Url"].(string)
		if !ok {
			return nil
		}
		req.Url = u
		req.RuleName, _ = v["RuleName"].(string)
		req.Method, _ = v["Method"].(string)
		req.Priority, _ = v["Priority"].(int)
		reqs = append(reqs, req)
	}

	return reqs
}

func AddJSReq(jreq map[string]any) []*collect.Request {
	reqs := make([]*collect.Request, 0)
	req := &collect.Request{}
	u, ok := jreq["Url"].(string)
	if !ok {
		return nil
	}
	req.Url = u
	req.RuleName, _ = jreq["RuleName"].(string)
	req.Method, _ = jreq["Method"].(string)
	req.Priority, _ = jreq["Priority"].(int)
	reqs = append(reqs, req)
	return reqs
}
