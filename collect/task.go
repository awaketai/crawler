package collect

import (
	"sync"
)

type Task struct {
	Visited     map[string]bool
	VisitedLock sync.Mutex
	// Rule 当前任务规则
	Rule RuleTree
	Options
}

// TaskMode 动态规则模型
type TaskMode struct {
	Options
	// Root 初始化种子节点的JS脚本
	Root string `json:"root"`
	// Rules 具体爬虫规则树
	Rules []RuleMode `json:"rule"`
}

func NewTask(opts ...Option) *Task {
	options := defaultOptions
	for _, o := range opts {
		o(&options)
	}
	return &Task{
		Visited: make(map[string]bool),
		Options: options,
	}
}
