package engine

import "github.com/awaketai/crawler/collect"

type Scheduler interface {
	Schedule()
	Push(...*collect.Request)
	Pull() *collect.Request
}
