package engine

import (
	"fmt"

	"github.com/awaketai/crawler/collect"
	"go.uber.org/zap"
)

// Schedule
// 1.创建调度程序，接收任务并将任务存储起来
// 2.执行调度任务，通过一定的调度算法将任务调度到合适的worker
// 3.创建指定数量的worker，完成实际任务的处理
// 4.创建数据处理协程，对爬取到的数据进行进一步处理
type Schedule struct {
	// requestCh 接收请求
	requestCh chan *collect.Request
	// workerCh 分配任务给worker
	workerCh chan *collect.Request
	reqQueue []*collect.Request
	Logger   *zap.Logger
	// priReqQueue 优先队列
	priReqQueue []*collect.Request
}

func NewSchedule() *Schedule {
	s := &Schedule{}
	requestCh := make(chan *collect.Request)
	worketCh := make(chan *collect.Request)
	s.requestCh = requestCh
	s.workerCh = worketCh

	return s
}

func (s *Schedule) Push(reqs ...*collect.Request) {
	for _, req := range reqs {
		s.requestCh <- req
	}
}

func (s *Schedule) Pull() *collect.Request {
	r := <-s.workerCh
	return r
}

func (s *Schedule) Schedule() {
	var (
		req *collect.Request
		ch  chan *collect.Request
	)
	for {
		if req == nil && len(s.priReqQueue) > 0 {
			// 先执行优先队列中的任务
			req = s.priReqQueue[0]
			s.priReqQueue = s.priReqQueue[1:]
			ch = s.workerCh
		}
		if req == nil && len(s.reqQueue) > 0 {
			req = s.reqQueue[0]
			s.reqQueue = s.reqQueue[1:]
			ch = s.workerCh
		}
		select {
		case r := <-s.requestCh:
			if r.Priority > 0 {
				s.priReqQueue = append(s.priReqQueue, r)
			} else {
				s.reqQueue = append(s.reqQueue, r)
			}
		case ch <- req:
			fmt.Println("ch received req:", req)
			req = nil
			ch = nil
		}
	}
}

// 避免重复请求
// 1.用什么数据结构存储数据才能保证快速地查找到请求的记录：哈希表
// 2.如何保证并发查找与写入时，不出现并发冲突问题：加入互扩锁
// 3.在什么条件下，我们才能确认请求是重复的，从而停止爬取
