package store

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/raft"
)

const (
	retainSnapshotCount = 2
	raftTimeout         = 10 * time.Second
)

type Store struct {
	mu sync.Mutex

	RaftDir     string
	BindAddress string

	store map[string]string
	raft  *raft.Raft
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
