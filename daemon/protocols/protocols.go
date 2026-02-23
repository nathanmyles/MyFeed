package protocols

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/nathanmyles/myfeed/daemon/store"
)

const (
	FeedProtocolID    = "/socialapp/feed/1.0.0"
	ProfileProtocolID = "/socialapp/profile/1.0.0"
)

type FeedRequest struct {
	Since time.Time `json:"since"`
}

type ProtocolHandler struct {
	host  host.Host
	store *store.Store
}

func NewProtocolHandler(h host.Host, s *store.Store) *ProtocolHandler {
	return &ProtocolHandler{host: h, store: s}
}

func (p *ProtocolHandler) Register() {
	p.host.SetStreamHandler(FeedProtocolID, p.handleFeedStream)
	p.host.SetStreamHandler(ProfileProtocolID, p.handleProfileStream)
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
