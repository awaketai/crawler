package collect

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

type Task struct{
	Name string 
	Url string
	Cookie string
	// WaitTime 第个请求等待多长时间
	WaitTime time.Duration
	// MaxDepth 标识爬取的最大深度
	MaxDepth int
	// RootReq 第一个请求
	RootReq *Request
	Fetcher Fetcher
	Visited map[string]bool
	VisitedLock sync.Mutex
	// Reload 网站是否可以重复爬取
	Reload bool 
	// Rule 当前任务规则
	Rule RuleTree
}

// Request 单个请求
type Request struct{
	unique string
	Task *Task
	Url string
	// Depth 当前任务的深度
	Depth int
	Method string
	Priority int
	ParseFunc func([]byte,*Request)  ParseResult
	RuleName string
}

type ParseResult struct{
	Requests []*Request
	Items []any
}

func (r *Request) Check() error {
	if r.Depth > r.Task.MaxDepth {
		return errors.New("Max depth limit reached")
	}

	return nil
}

func (r *Request) Unique()string {
	block := md5.Sum([]byte(r.Url + r.Method))
	return hex.EncodeToString(block[:])
}