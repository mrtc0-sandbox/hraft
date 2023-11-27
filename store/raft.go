package store

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb/v2"
)

const (
	defaultResponseTimeout = 10 * time.Second
)

type fsm Store

type snapshot struct {
	store map[string]string
}

type command struct {
	Op    string `json:"op"`
	Key   string `json:"key"`
	Value string `json:"value"`
}

func NewStore(dir, address string, logger hclog.Logger) *Store {
	store := &Store{
		RaftDir:     dir,
		BindAddress: address,
		Logger:      logger,

		store:           make(map[string]string),
		ResponseTimeout: defaultResponseTimeout,

		observerCh:    make(chan raft.Observation),
		observerDone:  make(chan struct{}),
		observerClose: make(chan struct{}),
	}

	store.observer = raft.NewObserver(store.observerCh, false, func(o *raft.Observation) bool {
		_, isLeaderChange := o.Data.(raft.LeaderObservation)
		_, isFailedHeartBeat := o.Data.(raft.FailedHeartbeatObservation)
		return isLeaderChange || isFailedHeartBeat
	})

	return store
}

func (s *Store) Start(id string, bootstrap bool) error {
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(id)

	// raft communication
	addr, err := net.ResolveTCPAddr("tcp", s.BindAddress)
	if err != nil {
		return err
	}

	transport, err := raft.NewTCPTransport(s.BindAddress, addr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return err
	}

	snapshots, err := raft.NewFileSnapshotStore(s.RaftDir, retainSnapshotCount, os.Stderr)
	if err != nil {
		return err
	}

	logStore, err := raftboltdb.New(raftboltdb.Options{
		Path: filepath.Join(s.RaftDir, "log.db"),
	})
	if err != nil {
		return err
	}

	stableStore, err := raftboltdb.New(raftboltdb.Options{
		Path: filepath.Join(s.RaftDir, "stable.db"),
	})
	if err != nil {
		return err
	}

	r, err := raft.NewRaft(config, (*fsm)(s), logStore, stableStore, snapshots, transport)
	if err != nil {
		return err
	}

	s.raft = r

	nodes, err := s.Nodes()
	if err != nil {
		return err
	}
	hasPeers := len(nodes) > 0

	if bootstrap || !hasPeers {
		config := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      config.LocalID,
					Address: transport.LocalAddr(),
				},
			},
		}

		s.raft.BootstrapCluster(config)
	}

	s.RaftID = string(config.LocalID)
	s.logStore = logStore
	s.stableStore = stableStore

	s.raft.RegisterObserver(s.observer)
	s.observerClose, s.observerDone = s.observe()

	return nil
}

func (s *Store) Stop() error {
	s.Logger.Info("stopping store server", "id", s.RaftID)

	close(s.observerClose)
	<-s.observerDone

	if err := s.raft.Shutdown().Error(); err != nil {
		return err
	}

	if err := s.logStore.Close(); err != nil {
		return err
	}

	if err := s.stableStore.Close(); err != nil {
		return err
	}

	return nil
}

func (s *Store) Join(nodeID, nodeAddress string) error {
	s.Logger.Info("received join request", "id", nodeID, "address", nodeAddress)

	configFuture := s.raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		return err
	}

	for _, server := range configFuture.Configuration().Servers {
		// Check if the ID or address is already joined in the cluster.
		if server.ID == raft.ServerID(nodeID) || server.Address == raft.ServerAddress(nodeAddress) {
			// If the node with the same ID and address exists, do nothing.
			if server.ID == raft.ServerID(nodeID) && server.Address == raft.ServerAddress(nodeAddress) {
				s.Logger.Info("Node already joined", "id", nodeID, "address", nodeAddress)
				return nil
			}

			// If either the ID or address exists, remove the node.
			f := s.raft.RemoveServer(server.ID, 0, 0)
			if err := f.Error(); err != nil {
				return fmt.Errorf("failed to remove server %s: %s", server.ID, err)
			}
		}
	}

	f := s.raft.AddVoter(raft.ServerID(nodeID), raft.ServerAddress(nodeAddress), 0, 0)
	if err := f.Error(); err != nil {
		return fmt.Errorf("failed to add voter %s: %s", nodeID, err)
	}

	s.Logger.Info("Node joined successfully", "id", nodeID, "address", nodeAddress)
	return nil
}

func (s *Store) Nodes() ([]raft.Server, error) {
	configFuture := s.raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		return nil, err
	}

	return configFuture.Configuration().Servers, nil
}

func (s *Store) IsLeader() bool {
	return s.raft.State() == raft.Leader
}

func (s *Store) LeadershipTransfer() error {
	f := s.raft.LeadershipTransfer()
	return f.Error()
}

func (s *Store) observe() (closeCh, doneCh chan struct{}) {
	closeCh = make(chan struct{})
	doneCh = make(chan struct{})

	go func() {
		defer close(doneCh)

		for {
			select {
			case o := <-s.observerCh:
				switch sig := o.Data.(type) {
				case raft.LeaderObservation:
					s.Logger.Info(fmt.Sprintf("leadership changed, new leader is %s(ID: %s)", sig.Leader, sig.LeaderID))
				case raft.FailedHeartbeatObservation:
					dur := time.Since(sig.LastContact)
					if dur > s.ResponseTimeout {
						s.Logger.Warn("heartbeet failed, remove node from cluster", "id", sig.PeerID, "last_contact", sig.LastContact)
						f := s.raft.RemoveServer(sig.PeerID, 0, 0)
						if f.Error() != nil {
							s.Logger.Error(fmt.Sprintf("failed to remove node %s", sig.PeerID), "error", f.Error())
						}
					}
				}
			case <-closeCh:
				return
			}
		}
	}()

	return closeCh, doneCh
}

func (f *fsm) Apply(l *raft.Log) interface{} {
	var c command
	if err := json.Unmarshal(l.Data, &c); err != nil {
		f.Logger.Error("Failed to unmarshal command", "error", err)
	}

	switch c.Op {
	case "set":
		return f.applySet(c.Key, c.Value)
	case "delete":
		return f.applyDelete(c.Key)
	default:
		f.Logger.Error("Unrecognized command op", "op", c.Op)
	}
	return nil
}

func (f *fsm) Snapshot() (raft.FSMSnapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	currentStore := make(map[string]string)
	for k, v := range f.store {
		currentStore[k] = v
	}

	return &snapshot{store: currentStore}, nil
}

func (f *fsm) Restore(rc io.ReadCloser) error {
	store := make(map[string]string)
	if err := json.NewDecoder(rc).Decode(&store); err != nil {
		f.Logger.Error("Failed to decode snapshot", "error", err)
	}

	f.store = store
	return nil
}

func (f *fsm) applySet(key, value string) interface{} {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.store[key] = value
	return nil
}

func (f *fsm) applyDelete(key string) interface{} {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.store, key)
	return nil
}

func (s *snapshot) Persist(sink raft.SnapshotSink) error {
	b, err := json.Marshal(s.store)
	if err != nil {
		sink.Cancel()
		return err
	}

	if _, err := sink.Write(b); err != nil {
		sink.Cancel()
		return err
	}

	return sink.Close()
}

func (s *snapshot) Release() {}
