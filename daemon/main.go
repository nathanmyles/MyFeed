package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/nathanmyles/myfeed/daemon/api"
	"github.com/nathanmyles/myfeed/daemon/node"
	"github.com/nathanmyles/myfeed/daemon/protocols"
	"github.com/nathanmyles/myfeed/daemon/store"
	"github.com/nathanmyles/myfeed/daemon/sync"
)

func main() {
	dataDir := flag.String("data", "", "Data directory for the daemon")
	flag.Parse()

	if *dataDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get home directory: %v\n", err)
			os.Exit(1)
		}
		*dataDir = homeDir + "/.myfeed"
	}

	if err := os.MkdirAll(*dataDir, 0700); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create data directory: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	node, err := node.New(ctx, *dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create node: %v\n", err)
		os.Exit(1)
	}
	defer node.Close()

	fmt.Printf("Node started with peer ID: %s\n", node.Host.ID())
	fmt.Println("Listening on:")
	for _, addr := range node.GetListeningAddrs() {
		fmt.Printf("  %s\n", addr)
	}

	dbPath := *dataDir + "/db"
	store, err := store.New(dbPath, node.Host.ID().String())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create store: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	protoHandler := protocols.NewProtocolHandler(node.Host, store)
	protoHandler.Register()

	syncer := sync.NewSyncer(node.Host, store)
	syncWorker := sync.NewSyncWorker(syncer, store, node.Host, 30*time.Second)

	server, err := api.NewServer(node, store, syncer, *dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create API server: %v\n", err)
		os.Exit(1)
	}
	defer server.Close()

	fmt.Printf("API server listening on port %d\n", server.Port())

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := node.Advertise(ctx); err != nil {
					fmt.Printf("Error advertising: %v\n", err)
				}
			}
		}
	}()

	go syncWorker.Start(ctx)

	go connectToKnownPeers(ctx, node, store, syncer)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nShutting down...")
	cancel()
}

func connectToKnownPeers(ctx context.Context, n *node.Node, s *store.Store, syncer *sync.Syncer) {
	profiles := s.GetKnownPeersWithProfiles()

	for _, profile := range profiles {
		if len(profile.Addresses) == 0 {
			continue
		}
		for _, addr := range profile.Addresses {
			peerInfo, err := peer.AddrInfoFromString(addr)
			if err != nil {
				continue
			}
			if n.Host.Network().Connectedness(peerInfo.ID) == network.Connected {
				continue
			}
			fmt.Printf("Connecting to known peer: %s\n", profile.PeerID)
			if err := n.Host.Connect(ctx, *peerInfo); err != nil {
				fmt.Printf("Failed to connect to %s: %v\n", profile.PeerID, err)
				continue
			}
			fmt.Printf("Connected to known peer: %s\n", profile.PeerID)

			if syncer != nil {
				pid, _ := peer.Decode(profile.PeerID)
				syncer.FetchFeed(ctx, pid, time.Time{})
			}
			break
		}
	}
}
