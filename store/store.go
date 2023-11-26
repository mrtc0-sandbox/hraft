package store

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb/v2"
)

const (
	retainSnapshotCount = 2
	raftTimeout         = 10 * time.Second
)

type Store struct {
	mu sync.Mutex

	RaftID      string
	RaftDir     string
	BindAddress string
	Logger      hclog.Logger

	store map[string]string
	raft  *raft.Raft

	logStore    *raftboltdb.BoltStore
	stableStore *raftboltdb.BoltStore

	// ResponseTimeout is the duration after which a non-reachable node is removed
	ResponseTimeout time.Duration

	observer   *raft.Observer
	observerCh chan raft.Observation
	// Channel to notify that the observer has terminated
	observerDone chan struct{}
	// Channel for notifying the observer to terminate
	// To terminate the observer, close this channel.
	observerClose chan struct{}
}

func (s *Store) Get(key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.store[key], nil
}

func (s *Store) Set(key, value string) error {
	if s.raft.State() != raft.Leader {
		return fmt.Errorf("not leader")
	}

	c := &command{
		Op:    "set",
		Key:   key,
		Value: value,
	}
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}

	f := s.raft.Apply(b, raftTimeout)
	return f.Error()
}

func (s *Store) Delete(key string) error {
	if s.raft.State() != raft.Leader {
		return fmt.Errorf("not leader")
	}

	c := &command{
		Op:  "delete",
		Key: key,
	}
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}

	f := s.raft.Apply(b, raftTimeout)
	return f.Error()
}
