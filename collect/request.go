package collect

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"math/rand"
	"time"
)

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
	TmpData   *Tmp
	// TestBody 测试用
	TestBody []byte
	Test     bool
}

type ParseResult struct {
	Requests []*Request
	Items    []any
}

func (r *Request) Check() error {
	if r.Depth > r.Task.MaxDepth {
		return errors.New("max depth limit reached")
	}
	if r.Task.Closed {
		return errors.New("task has closed")
	}

	return nil
}

func (r *Request) Unique() string {
	block := md5.Sum([]byte(r.Url + r.Method))
	return hex.EncodeToString(block[:])
}

func (r *Request) Fetch(ctx context.Context) ([]byte, error) {
	if err := r.Task.Limit.Wait(ctx); err != nil {
		return nil, err
	}
	// 随机休眠
	sleepTime := rand.Int63n(r.Task.WaitTime * 1000)
	time.Sleep(time.Duration(sleepTime) * time.Millisecond)

	return r.Task.Fetcher.Get(r)
}
