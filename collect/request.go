package collect

import (
	"errors"
	"sync"
	"time"
)

// Request 单个请求
type Request struct{
	Task *Task
	Url string
	// Depth 当前任务的深度
	Depth int
	ParseFunc func([]byte,*Request)  ParseResult
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

type Task struct{
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
}

