package collect

import (
	"github.com/awaketai/crawler/collector"
	"github.com/awaketai/crawler/limiter"
	"go.uber.org/zap"
)

type Options struct {
	Name     string              `json:"name"`      // 任务名称，保证唯一
	Url      string              `json:"url"`       // 任务url
	Cookie   string              `json:"cookie"`    // 任务cookie
	WaitTime int64               `json:"wait_time"` // 任务等待时间
	Reload   bool                `json:"reload"`    // 任务是否可以重复爬取
	MaxDepth int                 `json:"max_depth"` // 任务最大深度
	Fetcher  Fetcher           // 任务的Fetcher
	Storage  collector.Storager  // 任务的Storager
	Logger   *zap.Logger         // 任务的Logger
	Limit    limiter.RateLimiter // 任务的RateLimiter
	LimitCfg []LimitConfig `json:"Limits"`
	FetchType FetchType
}

type LimitConfig struct {
	EventCount int
	EventDur   int // seconds
	Bucket     int
}

var defaultOptions = Options{
	WaitTime: 5,
	Reload:   false,
	MaxDepth: 5,
	Logger:   zap.NewNop(),
}

type Option func(options *Options)

func WithLogger(logger *zap.Logger) Option {
	return func(options *Options) {
		options.Logger = logger
	}
}

func WithName(name string) Option {
	return func(options *Options) {
		options.Name = name
	}
}

func WithUrl(url string) Option {
	return func(options *Options) {
		options.Url = url
	}
}

func WithCookie(cookie string) Option {
	return func(options *Options) {
		options.Cookie = cookie
	}
}

func WithWaitTime(waitTime int64) Option {
	return func(options *Options) {
		options.WaitTime = waitTime
	}
}

func WithReload(reload bool) Option {
	return func(options *Options) {
		options.Reload = reload
	}
}

func WithMaxDepth(maxDepth int) Option {
	return func(options *Options) {
		options.MaxDepth = maxDepth
	}
}

func WithFetcher(fetcher Fetcher) Option {
	return func(options *Options) {
		options.Fetcher = fetcher
	}
}

func WithStorage(storage collector.Storager) Option {
	return func(options *Options) {
		options.Storage = storage
	}
}

func WithLimit(limit limiter.RateLimiter) Option {
	return func(options *Options) {
		options.Limit = limit
	}
}
