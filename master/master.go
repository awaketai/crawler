package master

import (
	"context"
	"encoding/json"
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
	"github.com/bwmarrin/snowflake"
)

type Master struct {
	ID        string
	ready     int32
	leaderID  string
	workNodes map[string]*registry.Node
	IDGen     *snowflake.Node
	etcdCli   *clientv3.Client
	resources map[string]*ResourceSpec
	options
}

func NewMaster(id string, opts ...Option) (*Master, error) {
	m := &Master{
		workNodes: make(map[string]*registry.Node),
		resources: make(map[string]*ResourceSpec),
	}
	options := defultOptions
	for _, o := range opts {
		o(&options)
	}
	node, err := snowflake.NewNode(1)
	if err != nil {
		return nil, err
	}
	m.IDGen = node
	ipv4, err := getLocalIP()
	if err != nil {
		return nil, err
	}
	m.options = options
	m.ID = getMasterID(id, ipv4, m.GRPCAddress)
	m.logger.Info("master id", zap.String("id", m.ID))
	endpoints := []string{m.registryURL}
	cli, err := clientv3.New(clientv3.Config{
		Endpoints: endpoints,
	})
	if err != nil {
		return nil, err
	}
	workerCfg, err := m.getWorkerCfg()
	if err != nil {
		return nil, err
	}
	m.etcdCli = cli
	m.updateNodes(*workerCfg)
	m.AddSeed()

	go m.Campaign()
	go m.HandleMsg()
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
					err = m.BecomeLeader()
					if err != nil {
						m.logger.Error("Become leader failed:%w", zap.Error(err))
					}
				}
			}
		case resp := <-leaderChange:
			if len(resp.Kvs) > 0 {
				m.logger.Info("watch leader change", zap.String("leader", string(resp.Kvs[0].Value)))
			}

		case resp := <-workerNodeChange:
			m.logger.Info("watch worker change", zap.Any("worker:", resp.regResult))
			m.updateNodes(resp.cfg)
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

type watchWorkerChParam struct {
	regResult *registry.Result
	cfg       cCfg.ServerConfig
}

func (m *Master) WatchWorker() chan watchWorkerChParam {
	sconfig, err := m.getWorkerCfg()
	if err != nil {
		m.logger.Error("get worker config err:", zap.Error(err))
		return nil
	}

	watch, err := m.registry.Watch(registry.WatchService(sconfig.Name))
	if err != nil {
		m.logger.Error("watch worker failed", zap.Error(err))
		return nil
	}
	ch := make(chan watchWorkerChParam)
	go func() {
		for {
			res, err := watch.Next()
			if err != nil {
				m.logger.Error("watch worker failed", zap.Error(err))
				continue
			}
			param := watchWorkerChParam{
				regResult: res,
				cfg:       *sconfig,
			}
			ch <- param
		}
	}()

	return ch
}

func (m *Master) BecomeLeader() error {
	if err := m.loadResource(); err != nil {
		return fmt.Errorf("loadResource failed:%w", err)
	}
	atomic.StoreInt32(&m.ready, 1)

	return nil
}

func (m *Master) updateNodes(sconfig cCfg.ServerConfig) {
	services, err := m.registry.GetService(sconfig.Name)
	if err != nil {
		m.logger.Error("get service failed", zap.Error(err))
		// return
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

func (m *Master) getWorkerCfg() (*cCfg.ServerConfig, error) {
	cfg, err := cCfg.GetCfg()
	if err != nil {
		m.logger.Error("get worker server config failed", zap.Error(err))
		return nil, err
	}
	var sconfig cCfg.ServerConfig
	if err := cfg.Get("WorkerServer").Scan(&sconfig); err != nil {
		m.logger.Error("get GRPC Server config failed", zap.Error(err))
		return nil, err
	}

	return &sconfig, nil
}

type Command int

const RESOURCE_PATH = "/resources"

const (
	MSG_ADD Command = iota
	MSG_DEL
)

type Message struct {
	Cmd   Command
	Specs []*ResourceSpec
}

type ResourceSpec struct {
	ID           string
	Name         string
	AssignedNode string
	CreationTime int64
}

func getResourcePath(name string) string {
	return fmt.Sprintf("%s/%s", RESOURCE_PATH, name)
}

func encode(s *ResourceSpec) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func decode(ds []byte) (*ResourceSpec, error) {
	var s *ResourceSpec
	err := json.Unmarshal(ds, s)
	return s, err
}

func (m *Master) AddResource(rs []*ResourceSpec) {
	for _, r := range rs {
		r.ID = m.IDGen.Generate().String()
		ns, err := m.Assign(r)
		if err != nil {
			m.logger.Error("assign resource failed", zap.Error(err))
			continue
		}
		r.AssignedNode = ns.Id + "|" + ns.Address
		r.CreationTime = time.Now().UnixNano()
		m.logger.Debug("add resource", zap.Any("specs", r))
		_, err = m.etcdCli.Put(context.Background(), getResourcePath(r.Name), encode(r))
		if err != nil {
			m.logger.Error("put resource failed", zap.Error(err))
			continue
		}
		m.resources[r.Name] = r
	}

}

func (m *Master) HandleMsg() {
	msgCh := make(chan *Message)
	select {
	case msg := <-msgCh:
		switch msg.Cmd {
		case MSG_ADD:
			m.AddResource(msg.Specs)
		}
	}

}

func (m *Master) Assign(r *ResourceSpec) (*registry.Node, error) {
	for _, n := range m.workNodes {
		return n, nil
	}

	return nil, errors.New("no worker nodes")
}

func (m *Master) AddSeed() {
	rs := make([]*ResourceSpec, 0, len(m.Seeds))
	for _, seed := range m.Seeds {
		resp, err := m.etcdCli.Get(
			context.Background(),
			getResourcePath(seed.Name),
			clientv3.WithSerializable(),
			clientv3.WithPrefix(),
		)
		if err != nil {
			m.logger.Error("etcd get resource failed", zap.Error(err))
			continue
		}
		if len(resp.Kvs) == 0 {
			r := &ResourceSpec{
				Name: seed.Name,
			}
			rs = append(rs, r)
		}
	}
	m.AddResource(rs)
}

func (m *Master) loadResource() error {
	resp, err := m.etcdCli.Get(context.Background(), RESOURCE_PATH, clientv3.WithSerializable())
	if err != nil {
		return fmt.Errorf("etcd get failed")
	}
	resources := make(map[string]*ResourceSpec)
	for _, kv := range resp.Kvs {
		r, err := decode(kv.Value)
		if err == nil && r != nil {
			resources[r.Name] = r
		}
	}
	m.logger.Info("leader init load resource", zap.Int("len", len(m.resources)))
	m.resources = resources

	return nil
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
