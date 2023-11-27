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
	"strings"
	"syscall"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/mrtc0-sandbox/hraft/api"
	"github.com/mrtc0-sandbox/hraft/store"
)

var (
	httpAddr        = flag.String("haddr", ":8080", "Listen address")
	raftAddr        = flag.String("raddr", ":12000", "Raft address")
	joinAddrs       = flag.String("join", "", "Join address")
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

	if *joinAddrs != "" {
		if err := join(*joinAddrs, *raftAddr, *nodeID); err != nil {
			logger.Error("failed to join cluster", "error", err)
			os.Exit(1)
		}
	}

	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-terminate
	logger.Info("Received termination signal. Shutting down...")

	if s.IsLeader() {
		if err := s.LeadershipTransfer(); err != nil {
			logger.Error("failed to transfer leadership", "error", err)
		}
	}

	if err := s.Stop(); err != nil {
		logger.Error("failed to stop store", "error", err)
		os.Exit(1)
	}

	logger.Info("Shutdown completed")
}

func join(joinAddrs, raftAddr, nodeID string) error {
	// TODO: Implement retry logic

	for {
		for _, joinAddr := range strings.Split(joinAddrs, ",") {
			b, err := json.Marshal(map[string]string{
				"addr": raftAddr,
				"id":   nodeID,
			})

			if err != nil {
				return err
			}

			resp, err := http.Post(fmt.Sprintf("http://%s/internal/cluster/join", joinAddr), "application/json", bytes.NewReader(b))
			if err != nil || resp.StatusCode != http.StatusOK {
				continue
			}

			return nil
		}

		time.Sleep(1 * time.Second)
	}
}
