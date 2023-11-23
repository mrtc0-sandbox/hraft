package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"

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

	nodeDataDir := filepath.Join(raftDataBaseDir, *dataDir)
	if err := os.MkdirAll(nodeDataDir, 0700); err != nil {
		log.Fatal(err)
	}

	s := store.NewStore(nodeDataDir, *raftAddr)
	if err := s.Start(*nodeID, *bootstrap); err != nil {
		log.Fatal(err)
	}

	api := api.NewStoreApiServer(*httpAddr, s)
	go func() {
		if err := api.Start(); err != nil {
			log.Fatal(err)
		}
	}()

	if *joinAddr != "" {
		if err := join(*joinAddr, *raftAddr, *nodeID); err != nil {
			log.Fatal(err)
		}
	}

	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, os.Interrupt)
	<-terminate
	slog.Info("Shutting down...")
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
