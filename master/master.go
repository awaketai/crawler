package master

import (
	"context"
	"errors"
	"fmt"
	"net"
	"reflect"
	"sync/atomic"
	"time"

	"go-micro.dev/v4/registry"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
	"go.uber.org/zap"

	cCfg "github.com/awaketai/crawler/config"
)

type Master struct {
	ID        string
	ready     int32
	leaderID  string
	workNodes map[string]*registry.Node
	options
}

func NewMaster(id string, opts ...Option) (*Master, error) {
	m := &Master{
		workNodes: make(map[string]*registry.Node),
	}
	options := defultOptions
	for _, o := range opts {
		o(&options)
	}

	ipv4, err := getLocalIP()
	if err != nil {
		return nil, err
	}
	m.options = options
	m.ID = getMasterID(id, ipv4, m.GRPCAddress)
	m.logger.Info("master id", zap.String("id", m.ID))
	go m.Campaign()
	return m, nil
}

func (m *Master) IsLeader() bool {
	return atomic.LoadInt32(&m.ready) != 0
}

func (m *Master) Campaign() error {
	endpoints := []string{m.registryURL}
	cli, err := clientv3.New(clientv3.Config{
		Endpoints: endpoints,
	})
	if err != nil {
		return err
	}
	s, err := concurrency.NewSession(cli, concurrency.WithTTL(5))
	if err != nil {
		return err
	}
	defer s.Close()
	// 创建一个新的etcd选举election
	// 第二个参数是所有master都在抢占的key，抢占到该key的master即将变为leader
	e := concurrency.NewElection(s, "/resources/election")
	leaderCh := make(chan error)
	go m.elect(e, leaderCh)
	leaderChange := e.Observe(context.Background())
	select {
	case resp := <-leaderChange:
		m.logger.Info("leader change", zap.String("leader", string(resp.Kvs[0].Value)))
	}
	workerNodeChange := m.WatchWorker()
	for {
		select {
		case err := <-leaderCh:
			if err != nil {
				m.logger.Error("leader campaign failed", zap.Error(err))
				go m.elect(e, leaderCh)

			} else {
				m.logger.Info("master change to leader")
				m.leaderID = m.ID
				if !m.IsLeader() {
					m.BecomeLeader()
				}
			}
		case resp := <-leaderChange:
			if len(resp.Kvs) > 0 {
				m.logger.Info("watch leader change", zap.String("leader", string(resp.Kvs[0].Value)))
			}

		case resp := <-workerNodeChange:
			m.logger.Info("watch worker change", zap.Any("worker:", resp))
		case <-time.After(20 * time.Second):
			rsp, err := e.Leader(context.Background())
			if err != nil {
				m.logger.Error("get leader failed", zap.Error(err))
				if errors.Is(err, concurrency.ErrElectionNoLeader) {
					go m.elect(e, leaderCh)
				}
			}
			if rsp != nil && len(rsp.Kvs) > 0 {
				m.logger.Debug("get leader", zap.String("value", string(rsp.Kvs[0].Value)))
				if m.IsLeader() && m.ID != string(rsp.Kvs[0].Value) {
					// 不再是leader
					atomic.StoreInt32(&m.ready, 0)
				}
			}
		}
	}
}

func (m *Master) elect(e *concurrency.Election, ch chan error) {
	// 堵塞直到选举成功
	err := e.Campaign(context.Background(), m.ID)
	ch <- err
}

func (m *Master) WatchWorker() chan *registry.Result {
	cfg, err := cCfg.GetCfg()
	if err != nil {
		fmt.Println("---:", err)
		return nil
	}
	var sconfig cCfg.ServerConfig
	if err := cfg.Get("WorkerServer").Scan(&sconfig); err != nil {
		m.logger.Error("get GRPC Server config failed", zap.Error(err))
		return nil
	}

	watch, err := m.registry.Watch(registry.WatchService(sconfig.Name))
	if err != nil {
		m.logger.Error("watch worker failed", zap.Error(err))
		return nil
	}
	ch := make(chan *registry.Result)
	go func() {
		for {
			res, err := watch.Next()
			if err != nil {
				m.logger.Error("watch worker failed", zap.Error(err))
				continue
			}
			ch <- res
		}
	}()

	return ch
}

func (m *Master) BecomeLeader() {
	atomic.StoreInt32(&m.ready, 1)
}

func (m *Master) updateNodes(sconfig cCfg.ServerConfig) {
	services, err := m.registry.GetService(sconfig.Name)
	if err != nil {
		m.logger.Error("get service failed", zap.Error(err))
		return
	}
	nodes := make(map[string]*registry.Node)
	if len(services) > 0 {
		for _, spec := range services[0].Nodes {
			nodes[spec.Id] = spec
		}
	}
	added, deleted, changed := workNodeDiff(m.workNodes, nodes)
	m.logger.Sugar().Info("worker joined: ", added, ", leaved: ", deleted, ", changed: ", changed)

	m.workNodes = nodes
}

func workNodeDiff(old map[string]*registry.Node, new map[string]*registry.Node) ([]string, []string, []string) {
	added := make([]string, 0)
	deleted := make([]string, 0)
	changed := make([]string, 0)
	for k, v := range new {
		if ov, ok := old[k]; ok {
			if !reflect.DeepEqual(v, ov) {
				changed = append(changed, k)
			}

		} else {
			added = append(added, k)
		}
	}
	for k := range old {
		if _, ok := new[k]; !ok {
			deleted = append(deleted, k)
		}
	}

	return added, deleted, changed
}

func getMasterID(id string, ipv4 string, GRPCAddress string) string {
	return fmt.Sprintf("master-%s-%s-%s", id, ipv4, GRPCAddress)
}

func getLocalIP() (string, error) {
	var (
		addrs []net.Addr
		err   error
	)

	// 获取所有网卡
	if addrs, err = net.InterfaceAddrs(); err != nil {
		return "", err
	}
	// 取第一个非lo的网卡ip
	for _, addr := range addrs {
		if ipNet, isIpNet := addr.(*net.IPNet); isIpNet && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String(), nil
			}
		}
	}

	return "", fmt.Errorf("获取本地ip失败")
}
