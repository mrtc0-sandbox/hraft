package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/hashicorp/go-hclog"
	"github.com/mrtc0-sandbox/hraft/api"
	"github.com/mrtc0-sandbox/hraft/store"
)

var (
	httpAddr        = flag.String("haddr", ":8080", "Listen address")
	raftAddr        = flag.String("raddr", ":12000", "Raft address")
	joinAddr        = flag.String("join", "", "Join address")
	nodeID          = flag.String("id", "", "Node ID")
	bootstrap       = flag.Bool("bootstrap", false, "Bootstrap the cluster")
	dataDir         = flag.String("data", "", "Data directory")
	raftDataBaseDir = "data"
)

func main() {
	flag.Parse()
	logger := hclog.New(&hclog.LoggerOptions{
		Name:  "hraft",
		Level: hclog.LevelFromString("DEBUG"),
	})

	nodeDataDir := filepath.Join(raftDataBaseDir, *dataDir)
	if err := os.MkdirAll(nodeDataDir, 0700); err != nil {
		logger.Error("failed to create data directory", "error", err)
		os.Exit(1)
	}

	s := store.NewStore(nodeDataDir, *raftAddr, logger)
	if err := s.Start(*nodeID, *bootstrap); err != nil {
		logger.Error("failed to start store", "error", err)
		os.Exit(1)
	}

	api := api.NewStoreApiServer(*httpAddr, s)
	go func() {
		if err := api.Start(); err != nil {
			logger.Error("failed to start api server", "error", err)
			os.Exit(1)
		}
	}()

	if *joinAddr != "" {
		if err := join(*joinAddr, *raftAddr, *nodeID); err != nil {
			logger.Error("failed to join cluster", "error", err)
			os.Exit(1)
		}
	}

	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-terminate
	logger.Info("Received termination signal. Shutting down...")

	if err := s.Stop(); err != nil {
		logger.Error("failed to stop store", "error", err)
		os.Exit(1)
	}

	logger.Info("Shutdown completed")
}

func join(joinAddr, raftAddr, nodeID string) error {
	b, err := json.Marshal(map[string]string{
		"addr": raftAddr,
		"id":   nodeID,
	})

	if err != nil {
		return err
	}

	resp, err := http.Post(fmt.Sprintf("http://%s/internal/cluster/join", joinAddr), "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	return nil
}
