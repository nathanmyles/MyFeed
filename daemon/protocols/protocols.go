package protocols

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/nathanmyles/myfeed/daemon/store"
)

const (
	FeedProtocolID           = "/socialapp/feed/1.0.0"
	ProfileProtocolID        = "/socialapp/profile/1.0.0"
	FriendRequestProtocolID  = "/socialapp/friend-request/1.0.0"
	FriendApprovedProtocolID = "/socialapp/friend-approved/1.0.0"
)

type FeedRequest struct {
	Since time.Time `json:"since"`
}

type FriendRequestMessage struct {
	PeerID    string `json:"peerId"`
	Timestamp int64  `json:"timestamp"`
}

type FriendApprovedMessage struct {
	PeerID string `json:"peerId"`
}

type ProtocolHandler struct {
	host             host.Host
	store            *store.Store
	onRequest        func(peerID string)
	onFriendApproved func(peerID string)
}

func NewProtocolHandler(h host.Host, s *store.Store) *ProtocolHandler {
	return &ProtocolHandler{host: h, store: s}
}

func (p *ProtocolHandler) SetFriendRequestCallback(fn func(peerID string)) {
	p.onRequest = fn
}

func (p *ProtocolHandler) SetFriendApprovedCallback(fn func(peerID string)) {
	p.onFriendApproved = fn
}

func (p *ProtocolHandler) Register() {
	p.host.SetStreamHandler(FeedProtocolID, p.handleFeedStream)
	p.host.SetStreamHandler(ProfileProtocolID, p.handleProfileStream)
	p.host.SetStreamHandler(FriendRequestProtocolID, p.handleFriendRequestStream)
	p.host.SetStreamHandler(FriendApprovedProtocolID, p.handleFriendApprovedStream)
}

func (p *ProtocolHandler) SendFriendRequest(ctx context.Context, peerID peer.ID) error {
	stream, err := p.host.NewStream(ctx, peerID, FriendRequestProtocolID)
	if err != nil {
		return fmt.Errorf("failed to open friend request stream: %w", err)
	}
	defer stream.Close()

	msg := FriendRequestMessage{
		PeerID:    p.host.ID().String(),
		Timestamp: time.Now().Unix(),
	}

	if err := json.NewEncoder(stream).Encode(msg); err != nil {
		return fmt.Errorf("failed to encode friend request: %w", err)
	}

	return nil
}

func (p *ProtocolHandler) SendFriendApproved(ctx context.Context, peerID peer.ID) error {
	stream, err := p.host.NewStream(ctx, peerID, FriendApprovedProtocolID)
	if err != nil {
		return fmt.Errorf("failed to open friend approved stream: %w", err)
	}
	defer stream.Close()

	msg := FriendApprovedMessage{
		PeerID: p.host.ID().String(),
	}

	if err := json.NewEncoder(stream).Encode(msg); err != nil {
		return fmt.Errorf("failed to encode friend approved: %w", err)
	}

	return nil
}

func (p *ProtocolHandler) handleFeedStream(s network.Stream) {
	defer s.Close()

	var req FeedRequest
	decoder := json.NewDecoder(s)
	if err := decoder.Decode(&req); err != nil {
		if err != io.EOF {
			fmt.Printf("Error decoding feed request: %v\n", err)
		}
		return
	}

	posts, err := p.store.GetLocalPosts(req.Since)
	if err != nil {
		fmt.Printf("Error getting local posts: %v\n", err)
		return
	}

	writer := bufio.NewWriter(s)
	encoder := json.NewEncoder(writer)
	for _, post := range posts {
		if err := encoder.Encode(post); err != nil {
			fmt.Printf("Error encoding post: %v\n", err)
			return
		}
		if err := writer.Flush(); err != nil {
			fmt.Printf("Error flushing writer: %v\n", err)
			return
		}
	}
}

func (p *ProtocolHandler) handleProfileStream(s network.Stream) {
	defer s.Close()

	profile, err := p.store.GetProfile()
	if err != nil {
		fmt.Printf("Error getting profile: %v\n", err)
		return
	}

	encoder := json.NewEncoder(s)
	if err := encoder.Encode(profile); err != nil {
		fmt.Printf("Error encoding profile: %v\n", err)
	}
}

func (p *ProtocolHandler) handleFriendRequestStream(s network.Stream) {
	defer s.Close()

	var msg FriendRequestMessage
	if err := json.NewDecoder(s).Decode(&msg); err != nil {
		fmt.Printf("Error decoding friend request: %v\n", err)
		return
	}

	friend := &store.Friend{
		PeerID:    msg.PeerID,
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	if err := p.store.SaveFriend(friend); err != nil {
		fmt.Printf("Error saving friend request: %v\n", err)
		return
	}

	if p.onRequest != nil {
		p.onRequest(msg.PeerID)
	}
}

func (p *ProtocolHandler) handleFriendApprovedStream(s network.Stream) {
	defer s.Close()

	var msg FriendApprovedMessage
	if err := json.NewDecoder(s).Decode(&msg); err != nil {
		fmt.Printf("Error decoding friend approved: %v\n", err)
		return
	}

	friend := &store.Friend{
		PeerID:    msg.PeerID,
		Status:    "approved",
		CreatedAt: time.Now(),
	}
	if err := p.store.SaveFriend(friend); err != nil {
		fmt.Printf("Error saving friend approved: %v\n", err)
		return
	}

	if p.onFriendApproved != nil {
		p.onFriendApproved(msg.PeerID)
	}
}
