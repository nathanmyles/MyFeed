package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	"github.com/nathanmyles/myfeed/daemon/protocols"
	"github.com/nathanmyles/myfeed/daemon/store"
)

func main() {
	peerAddr := flag.String("peer", "", "Peer multiaddress to connect to")
	since := flag.Duration("since", 24*time.Hour, "Fetch posts since this duration ago")
	flag.Parse()

	if *peerAddr == "" {
		fmt.Fprintln(os.Stderr, "Usage: testclient -peer <multiaddr>")
		os.Exit(1)
	}

	ctx := context.Background()

	h, err := libp2p.New(
		libp2p.Security(noise.ID, noise.New),
		libp2p.DefaultTransports,
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create host: %v\n", err)
		os.Exit(1)
	}
	defer h.Close()

	peerInfo, err := peer.AddrInfoFromString(*peerAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid peer address: %v\n", err)
		os.Exit(1)
	}

	if err := h.Connect(ctx, *peerInfo); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to peer: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Connected to peer: %s\n", peerInfo.ID)

	stream, err := h.NewStream(ctx, peerInfo.ID, protocols.FeedProtocolID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open feed stream: %v\n", err)
		os.Exit(1)
	}
	defer stream.Close()

	req := protocols.FeedRequest{Since: time.Now().Add(-*since)}
	if err := json.NewEncoder(stream).Encode(req); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to send request: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Posts received:")
	reader := bufio.NewReader(stream)
	decoder := json.NewDecoder(reader)
	for {
		var post store.Post
		if err := decoder.Decode(&post); err != nil {
			break
		}
		fmt.Printf("- [%s] %s\n", post.CreatedAt.Format(time.RFC822), post.Content)
	}

	profileStream, err := h.NewStream(ctx, peerInfo.ID, protocols.ProfileProtocolID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open profile stream: %v\n", err)
		os.Exit(1)
	}
	defer profileStream.Close()

	var profile store.Profile
	if err := json.NewDecoder(profileStream).Decode(&profile); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to decode profile: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nProfile:\n")
	fmt.Printf("  Peer ID: %s\n", profile.PeerID)
	fmt.Printf("  Display Name: %s\n", profile.DisplayName)
	fmt.Printf("  Bio: %s\n", profile.Bio)
}
