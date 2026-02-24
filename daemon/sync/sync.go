package sync

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/nathanmyles/myfeed/daemon/node"
	"github.com/nathanmyles/myfeed/daemon/protocols"
	"github.com/nathanmyles/myfeed/daemon/store"
)

type Syncer struct {
	host  host.Host
	store *store.Store
}

func NewSyncer(h host.Host, s *store.Store) *Syncer {
	return &Syncer{host: h, store: s}
}

func (s *Syncer) FetchFeed(ctx context.Context, peerID peer.ID, since time.Time) ([]store.Post, error) {
	stream, err := s.host.NewStream(ctx, peerID, protocols.FeedProtocolID)
	if err != nil {
		return nil, fmt.Errorf("failed to open feed stream: %w", err)
	}
	defer stream.Close()

	req := protocols.FeedRequest{Since: since}
	if err := json.NewEncoder(stream).Encode(req); err != nil {
		return nil, fmt.Errorf("failed to send feed request: %w", err)
	}

	var posts []store.Post
	reader := bufio.NewReader(stream)
	decoder := json.NewDecoder(reader)
	for {
		var post store.Post
		if err := decoder.Decode(&post); err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, fmt.Errorf("failed to decode post: %w", err)
		}
		post.AuthorPeerID = peerID.String()
		posts = append(posts, post)
	}

	for _, post := range posts {
		sigData := fmt.Sprintf("%s|%s|%d", post.ID, post.Content, post.CreatedAt.Unix())
		verified, err := node.VerifySignature(post.AuthorPeerID, []byte(sigData), post.Signature)
		if err != nil {
			fmt.Printf("Error verifying signature for post %s: %v\n", post.ID, err)
			continue
		}
		if !verified {
			fmt.Printf("Invalid signature for post %s from %s\n", post.ID, post.AuthorPeerID)
			continue
		}

		if err := s.store.SaveRemotePost(&post); err != nil {
			fmt.Printf("Error saving remote post %s: %v\n", post.ID, err)
		}
	}

	go func() {
		if _, err := s.FetchProfile(ctx, peerID); err != nil {
			fmt.Printf("Error fetching profile for peer %s: %v\n", peerID, err)
		}
	}()

	return posts, nil
}

func (s *Syncer) FetchProfile(ctx context.Context, peerID peer.ID) (*store.Profile, error) {
	stream, err := s.host.NewStream(ctx, peerID, protocols.ProfileProtocolID)
	if err != nil {
		return nil, fmt.Errorf("failed to open profile stream: %w", err)
	}
	defer stream.Close()

	var profile store.Profile
	if err := json.NewDecoder(stream).Decode(&profile); err != nil {
		return nil, fmt.Errorf("failed to decode profile: %w", err)
	}

	profile.PeerID = peerID.String()
	if err := s.store.SaveRemoteProfileMerge(&profile); err != nil {
		return nil, fmt.Errorf("failed to save remote profile: %w", err)
	}

	return &profile, nil
}

type SyncWorker struct {
	syncer   *Syncer
	store    *store.Store
	host     host.Host
	interval time.Duration
	stopCh   chan struct{}
}

func NewSyncWorker(syncer *Syncer, store *store.Store, h host.Host, interval time.Duration) *SyncWorker {
	return &SyncWorker{
		syncer:   syncer,
		store:    store,
		host:     h,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

func (w *SyncWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	w.syncAllPeers(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.syncAllPeers(ctx)
		}
	}
}

func (w *SyncWorker) Stop() {
	close(w.stopCh)
}

func (w *SyncWorker) syncAllPeers(ctx context.Context) {
	peers := w.store.GetKnownPeers()
	for _, peerIDStr := range peers {
		peerID, err := peer.Decode(peerIDStr)
		if err != nil {
			fmt.Printf("Invalid peer ID %s: %v\n", peerIDStr, err)
			continue
		}

		if w.syncer.host.Network().Connectedness(peerID) != network.Connected {
			continue
		}

		_, err = w.syncer.FetchFeed(ctx, peerID, time.Time{})
		if err != nil {
			fmt.Printf("Error syncing feed for peer %s: %v\n", peerIDStr, err)
		}
	}
}
