package collect

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

type Propety struct {
	Name   string `json:"name"`
	Url    string `json:"url"`
	Cookie string `json:"cookie"`
	// WaitTime每个请求等待多长时间
	WaitTime time.Duration `json:"wait_time"`
	Reload   bool          `json:"reload"`
	// 爬取的最大深度
	MaxDepth int `json:"max_depth"`
}

type Task struct {
	Propety
	// RootReq 第一个请求
	RootReq     *Request
	Fetcher     Fetcher
	Visited     map[string]bool
	VisitedLock sync.Mutex
	// Reload 网站是否可以重复爬取
	Reload bool
	// Rule 当前任务规则
	Rule RuleTree
}

// TaskMode 动态规则模型
type TaskMode struct {
	Propety
	// Root 初始化种子节点的JS脚本
	Root string `json:"root"`
	// Rules 具体爬虫规则树
	Rules []RuleMode `json:"rule"`
}

// Request 单个请求
type Request struct {
	unique string
	Task   *Task
	Url    string
	// Depth 当前任务的深度
	Depth     int
	Method    string
	Priority  int
	ParseFunc func([]byte, *Request) ParseResult
	RuleName  string
	TmpData *Tmp
}

type ParseResult struct {
	Requests []*Request
	Items    []any
}

func (r *Request) Check() error {
	if r.Depth > r.Task.Propety.MaxDepth {
		return errors.New("Max depth limit reached")
	}

	return nil
}

func (r *Request) Unique() string {
	block := md5.Sum([]byte(r.Url + r.Method))
	return hex.EncodeToString(block[:])
}
