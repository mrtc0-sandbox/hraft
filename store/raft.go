package store

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb/v2"
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

func NewStore(dir, address string) *Store {
	return &Store{
		RaftDir:     dir,
		BindAddress: address,
		store:       make(map[string]string),
	}
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

	if bootstrap {
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

	return nil
}

func (s *Store) Join(nodeID, nodeAddress string) error {
	slog.Info("received join request", "id", nodeID, "address", nodeAddress)

	configFuture := s.raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		return err
	}

	for _, server := range configFuture.Configuration().Servers {
		// Check if the ID or address is already joined in the cluster.
		if server.ID == raft.ServerID(nodeID) || server.Address == raft.ServerAddress(nodeAddress) {
			// If the node with the same ID and address exists, do nothing.
			if server.ID == raft.ServerID(nodeID) && server.Address == raft.ServerAddress(nodeAddress) {
				slog.Info("Node already joined", "id", nodeID, "address", nodeAddress)
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

	slog.Info("Node joined successfully", "id", nodeID, "address", nodeAddress)
	return nil
}

func (f *fsm) Apply(l *raft.Log) interface{} {
	var c command
	if err := json.Unmarshal(l.Data, &c); err != nil {
		slog.Error("Failed to unmarshal command", "error", err)
	}

	switch c.Op {
	case "set":
		return f.applySet(c.Key, c.Value)
	case "delete":
		return f.applyDelete(c.Key)
	default:
		slog.Error("Unrecognized command op", "op", c.Op)
	}
	fmt.Println("ddddddddddd")
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
		slog.Error("Failed to decode snapshot", "error", err)
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
