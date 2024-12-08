package engine

import (
	"github.com/awaketai/crawler/collect"
	"go.uber.org/zap"
)

// ScheduleEngine
// 1.创建调度程序，接收任务并将任务存储起来
// 2.执行调度任务，通过一定的调度算法将任务调度到合适的worker
// 3.创建指定数量的worker，完成实际任务的处理
// 4.创建数据处理协程，对爬取到的数据进行进一步处理
type ScheduleEngine struct {
	// requestCh 接收请求
	requestCh chan *collect.Request
	// workerCh 分配任务给worker
	workerCh chan *collect.Request
	// out 处理爬取后的数据
	out   chan collect.ParseResult
	options
}

func NewScheduleEngine(opts ...Option) *ScheduleEngine {
	options := defaultOptions
	for _,opt := range opts {
		opt(&options)
	}
	s := &ScheduleEngine{}
	s.options = options

	return s
}

func (s *ScheduleEngine) Run() {
	requestCh := make(chan *collect.Request)
	workerCh := make(chan *collect.Request)
	out := make(chan collect.ParseResult)
	s.requestCh = requestCh
	s.workerCh = workerCh
	s.out = out
	go s.Schedule()
	for i := 0; i < s.WorkCount; i++ {
		go s.CreateWork()
	}
	s.HandleResult()
}

func (s *ScheduleEngine) Schedule() {
	var reqQueue = s.Seeds
	go func() {
		for {
			var (
				req *collect.Request
				ch  chan *collect.Request
			)
			if len(reqQueue) > 0 {
				req = reqQueue[0]
				reqQueue = reqQueue[1:]
				ch = s.workerCh
			}
			select {
			case r := <-s.requestCh:
				reqQueue = append(reqQueue, r)
			case ch <- req:
			}

		}
	}()
}

func (s *ScheduleEngine) CreateWork() {
	for {
		r := <-s.workerCh
		body, err := s.Fetcher.Get(r)
		if err != nil {
			s.Logger.Error("fet err", zap.String("url", r.Url), zap.Error(err))
			continue
		}
		result := r.ParseFunc(body, r)
		s.out <- result
	}
}

func (s *ScheduleEngine) HandleResult() {
	for result := range s.out {
		for _, req := range result.Requests {
			s.requestCh <- req
		}
		for _, item := range result.Items {
			// 数据存储
			s.Logger.Sugar().Info("get result:", item)
		}
	}
}
