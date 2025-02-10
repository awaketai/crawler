package master

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"reflect"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go-micro.dev/v4/client"
	"go-micro.dev/v4/registry"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
	"go.uber.org/zap"

	cCfg "github.com/awaketai/crawler/config"
	"github.com/awaketai/crawler/goout/common"
	"github.com/bwmarrin/snowflake"
	"github.com/golang/protobuf/ptypes/empty"
)

// Master 分配资源的时机有三种
// 1.当master成为leader时
// 2.当客户端调用Master API进行资源的增删改查时
// 3.当master监听到worker节点发生变化时
type Master struct {
	options
	ID         string
	ready      int32
	leaderID   string
	workNodes  map[string]*NodeSpec
	IDGen      *snowflake.Node
	etcdCli    *clientv3.Client
	resources  map[string]*ResourceSpec
	forwardCli common.CrawlerMasterService
	mu         sync.Mutex
}

func NewMaster(id string, opts ...Option) (*Master, error) {
	m := &Master{
		workNodes: make(map[string]*NodeSpec),
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
		// 保存master id
		m.leaderID = string(resp.Kvs[0].Value)
	}
	workerNodeChange := m.WatchWorker()
	for {
		select {
		case err := <-leaderCh:
			m.leaderID = m.ID
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
				m.leaderID = string(resp.Kvs[0].Value)
			}

		case resp := <-workerNodeChange:
			m.logger.Info("watch worker change", zap.Any("worker:", resp.regResult))
			m.updateNodes(resp.cfg)
			if err := m.loadResource(); err != nil {
				m.logger.Error("work change load resource err:", zap.Error(err))
			}
			m.reAssign()
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
	nodeCfg, err := m.getWorkerCfg()
	if err != nil {
		return err
	}
	m.updateNodes(*nodeCfg)
	if err := m.loadResource(); err != nil {
		return fmt.Errorf("loadResource failed:%w", err)
	}
	m.reAssign()
	atomic.StoreInt32(&m.ready, 1)

	return nil
}

func (m *Master) updateNodes(sconfig cCfg.ServerConfig) {
	services, err := m.registry.GetService(sconfig.Name)
	if err != nil {
		m.logger.Error("get service failed", zap.Error(err))
		// return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	nodes := make(map[string]*NodeSpec)
	if len(services) > 0 {
		for _, spec := range services[0].Nodes {
			nodes[spec.Id] = &NodeSpec{
				Node: spec,
			}
		}
	}
	added, deleted, changed := workNodeDiff(m.workNodes, nodes)
	m.logger.Sugar().Info("worker joined: ", added, ", leaved: ", deleted, ", changed: ", changed)

	m.workNodes = nodes
}

func workNodeDiff(old map[string]*NodeSpec, new map[string]*NodeSpec) ([]string, []string, []string) {
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

type NodeSpec struct {
	Node    *registry.Node
	Payload int
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

func Encode(s *ResourceSpec) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func Decode(ds []byte) (*ResourceSpec, error) {
	var s *ResourceSpec
	err := json.Unmarshal(ds, &s)
	return s, err
}

func (m *Master) addResources(rs []*ResourceSpec) {
	for _, r := range rs {
		m.addResource(r)
	}
}

func (m *Master) addResource(r *ResourceSpec) (*NodeSpec, error) {
	r.ID = m.IDGen.Generate().String()
	ns, err := m.Assign(r)
	if err != nil {
		m.logger.Error("assign resource failed", zap.Error(err))
		return nil, err
	}
	if ns == nil || ns.Node == nil {
		m.logger.Error("invalid node")
		return nil, err
	}
	r.AssignedNode = ns.Node.Id + "|" + ns.Node.Address
	r.CreationTime = time.Now().UnixNano()
	m.logger.Debug("add resource", zap.Any("specs", r))
	_, err = m.etcdCli.Put(context.Background(), getResourcePath(r.Name), Encode(r))
	if err != nil {
		m.logger.Error("put resource failed", zap.Error(err))
		return nil, err
	}
	m.resources[r.Name] = r
	ns.Payload++

	return ns, nil
}

// DeleteResource 接口调用删除任务
func (m *Master) DeleteResource(ctx context.Context, spec *common.ResourceSpec, empty *empty.Empty) error {
	if !m.IsLeader() && m.leaderID != "" && m.leaderID != m.ID {
		addr := getLeaderAddr(m.leaderID)
		_, err := m.forwardCli.DeleteResource(ctx, spec, client.WithAddress(addr))
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.resources[spec.Name]
	if !ok {
		return fmt.Errorf("no such task:%v", spec.Name)
	}
	if _, err := m.etcdCli.Delete(context.Background(), getResourcePath(spec.Name)); err != nil {
		return err
	}
	delete(m.resources, spec.Name)
	if r.AssignedNode != "" {
		nodeID, err := getNodeID(r.AssignedNode)
		if err != nil {
			return err
		}
		if ns, ok := m.workNodes[nodeID]; ok {
			ns.Payload -= 1
		}
	}

	return nil
}

func (m *Master) SetForwardCli(forwardCli common.CrawlerMasterService) {
	m.forwardCli = forwardCli
}

// AddResource 接口调用增加任务
// 如果当前节点不是leader，则获取leader的地址进行请求转发
func (m *Master) AddResource(ctx context.Context, req *common.ResourceSpec, resp *common.NodeSpec) error {
	if !m.IsLeader() && m.leaderID != "" && m.leaderID != m.ID {
		addr := getLeaderAddr(m.leaderID)
		nodeSpec, err := m.forwardCli.AddResource(ctx, req, client.WithAddress(addr))
		if err != nil {
			return err
		}
		resp.Id = nodeSpec.Id
		resp.Address = nodeSpec.Address
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	nodeSpec, err := m.addResource(&ResourceSpec{Name: req.Name})
	if err != nil {
		return err
	}
	if nodeSpec != nil {
		resp.Id = nodeSpec.Node.Id
		resp.Address = nodeSpec.Node.Address
	}

	return nil
}

func (m *Master) HandleMsg() {
	msgCh := make(chan *Message)
	select {
	case msg := <-msgCh:
		switch msg.Cmd {
		case MSG_ADD:
			m.addResources(msg.Specs)
		}
	}

}

func (m *Master) Assign(r *ResourceSpec) (*NodeSpec, error) {
	candidates := make([]*NodeSpec, 0, len(m.workNodes))
	for _, node := range m.workNodes {
		candidates = append(candidates, node)
	}
	// 找到最低的负载
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Payload < candidates[j].Payload
	})
	if len(candidates) > 0 {
		return candidates[0], nil
	}

	return nil, errors.New("no worker nodes")
}

func (m *Master) reAssign() {
	rs := make([]*ResourceSpec, 0, len(m.resources))
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.resources {
		if r.AssignedNode == "" {
			rs = append(rs, r)
			continue
		}
		id, err := getNodeID(r.AssignedNode)
		if err != nil {
			m.logger.Error("get node id failed", zap.Error(err))
			continue
		}
		if _, ok := m.workNodes[id]; !ok {
			rs = append(rs, r)
		}
	}
	m.addResources(rs)
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
	m.addResources(rs)
}

func (m *Master) loadResource() error {
	resp, err := m.etcdCli.Get(context.Background(), RESOURCE_PATH, clientv3.WithPrefix(), clientv3.WithSerializable())
	if err != nil {
		return fmt.Errorf("etcd get failed")
	}
	resources := make(map[string]*ResourceSpec)
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, kv := range resp.Kvs {
		r, err := Decode(kv.Value)
		if err == nil && r != nil {
			resources[r.Name] = r
		}
	}
	m.resources = resources
	m.logger.Info("leader load resource", zap.Int("len", len(m.resources)))

	return nil
}

func getMasterID(id string, ipv4 string, GRPCAddress string) string {
	return fmt.Sprintf("master-%s-%s%s", id, ipv4, GRPCAddress)
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

func getNodeID(assigned string) (string, error) {
	node := strings.Split(assigned, "|")
	if len(node) < 2 {
		return "", fmt.Errorf("invalid assigned node")
	}
	id := node[0]
	return id, nil
}

func getLeaderAddr(addr string) string {
	s := strings.Split(addr, "-")
	if len(s) < 3 {
		return ""
	}

	return s[2]
}
