package api

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/nathanmyles/myfeed/daemon/node"
	"github.com/nathanmyles/myfeed/daemon/store"
	syncer "github.com/nathanmyles/myfeed/daemon/sync"
)

type Server struct {
	host      host.Host
	node      *node.Node
	store     *store.Store
	syncer    *syncer.Syncer
	port      int
	portFile  string
	server    *http.Server
	wsClients map[*websocket.Conn]bool
	wsMutex   sync.Mutex
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func NewServer(n *node.Node, s *store.Store, syn *syncer.Syncer, dataDir string) (*Server, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port
	portFile := filepath.Join(dataDir, "daemon.port")

	if err := os.WriteFile(portFile, []byte(fmt.Sprintf("%d", port)), 0644); err != nil {
		listener.Close()
		return nil, fmt.Errorf("failed to write port file: %w", err)
	}

	srv := &Server{
		host:      n.Host,
		node:      n,
		store:     s,
		syncer:    syn,
		port:      port,
		portFile:  portFile,
		wsClients: make(map[*websocket.Conn]bool),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", srv.handleStatus)
	mux.HandleFunc("/api/feed", srv.handleFeed)
	mux.HandleFunc("/api/posts", srv.handlePosts)
	mux.HandleFunc("/api/peers", srv.handlePeers)
	mux.HandleFunc("/api/connect", srv.handleConnect)
	mux.HandleFunc("/api/profile", srv.handleProfile)
	mux.HandleFunc("/api/profile/", srv.handleRemoteProfile)
	mux.HandleFunc("/api/sync", srv.handleSync)
	mux.HandleFunc("/api/events", srv.handleEvents)

	srv.server = &http.Server{
		Handler: srv.corsMiddleware(mux),
	}

	go srv.server.Serve(listener)

	return srv, nil
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) jsonError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func (s *Server) jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		s.jsonError(w, "Method not allowed", 405)
		return
	}

	connectedPeers := len(s.host.Network().Conns())

	s.jsonResponse(w, map[string]interface{}{
		"version":        "0.1.0",
		"peerId":         s.host.ID().String(),
		"addresses":      s.getListeningAddrs(),
		"connectedPeers": connectedPeers,
	})
}

func (s *Server) handleFeed(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		s.jsonError(w, "Method not allowed", 405)
		return
	}

	posts, err := s.store.GetAllPosts()
	if err != nil {
		s.jsonError(w, "Failed to get posts", 500)
		return
	}

	sort.Slice(posts, func(i, j int) bool {
		return posts[i].CreatedAt.After(posts[j].CreatedAt)
	})

	s.jsonResponse(w, posts)
}

func (s *Server) handlePosts(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		s.jsonError(w, "Method not allowed", 405)
		return
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "Invalid request body", 400)
		return
	}

	if req.Content == "" {
		s.jsonError(w, "Content is required", 400)
		return
	}

	post := &store.Post{
		Content: req.Content,
	}

	if err := s.store.SavePost(post); err != nil {
		s.jsonError(w, "Failed to save post", 500)
		return
	}

	sigData := fmt.Sprintf("%s|%s|%d", post.ID, post.Content, post.CreatedAt.Unix())
	signature, err := s.node.Sign([]byte(sigData))
	if err != nil {
		s.jsonError(w, "Failed to sign post", 500)
		return
	}
	post.Signature = signature

	if err := s.store.UpdatePostSignature(post.ID, post.Signature); err != nil {
		s.jsonError(w, "Failed to update signature", 500)
		return
	}

	s.BroadcastEvent("feed:updated", nil)
	s.jsonResponse(w, post)
}

func (s *Server) handlePeers(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		s.jsonError(w, "Method not allowed", 405)
		return
	}

	var peers []map[string]interface{} = []map[string]interface{}{}
	seenPeers := make(map[peer.ID]bool)

	for _, conn := range s.host.Network().Conns() {
		p := conn.RemotePeer()
		if seenPeers[p] {
			continue
		}
		seenPeers[p] = true
		peers = append(peers, map[string]interface{}{
			"peerId":  p.String(),
			"online":  s.host.Network().Connectedness(p) == network.Connected,
			"address": conn.RemoteMultiaddr().String(),
		})
	}

	knownPeers := s.store.GetKnownPeers()
	for _, peerIDStr := range knownPeers {
		p, err := peer.Decode(peerIDStr)
		if err != nil || seenPeers[p] {
			continue
		}
		peers = append(peers, map[string]interface{}{
			"peerId": peerIDStr,
			"online": s.host.Network().Connectedness(p) == network.Connected,
		})
	}

	s.jsonResponse(w, peers)
}

func (s *Server) handleConnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		s.jsonError(w, "Method not allowed", 405)
		return
	}

	var req struct {
		Address string `json:"address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "Invalid request body", 400)
		return
	}

	if req.Address == "" {
		s.jsonError(w, "Address is required", 400)
		return
	}

	peerInfo, err := peer.AddrInfoFromString(req.Address)
	if err != nil {
		s.jsonError(w, fmt.Sprintf("Invalid peer address: %v", err), 400)
		return
	}

	if err := s.host.Connect(r.Context(), *peerInfo); err != nil {
		s.jsonError(w, fmt.Sprintf("Failed to connect: %v", err), 500)
		return
	}

	profile := &store.Profile{
		PeerID:    peerInfo.ID.String(),
		Addresses: []string{req.Address},
	}
	s.store.SaveRemoteProfile(profile)

	s.BroadcastEvent("peer:connected", map[string]string{"peerId": peerInfo.ID.String()})

	s.jsonResponse(w, map[string]interface{}{
		"peerId":  peerInfo.ID.String(),
		"online":  true,
		"address": req.Address,
	})
}

func (s *Server) handleSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		s.jsonError(w, "Method not allowed", 405)
		return
	}

	if s.syncer == nil {
		s.jsonError(w, "Syncer not available", 500)
		return
	}

	peers := s.store.GetKnownPeers()
	ctx := r.Context()
	synced := 0
	for _, peerIDStr := range peers {
		peerID, err := peer.Decode(peerIDStr)
		if err != nil {
			continue
		}
		if s.host.Network().Connectedness(peerID) != network.Connected {
			continue
		}
		if _, err := s.syncer.FetchFeed(ctx, peerID, time.Time{}); err != nil {
			fmt.Printf("Error syncing feed from %s: %v\n", peerIDStr, err)
			continue
		}
		synced++
	}

	s.jsonResponse(w, map[string]interface{}{"syncedPeers": synced})
}

func (s *Server) handleProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		profile, err := s.store.GetProfile()
		if err != nil {
			s.jsonError(w, "Failed to get profile", 500)
			return
		}
		s.jsonResponse(w, profile)
		return
	}

	if r.Method == "POST" {
		var req struct {
			DisplayName string `json:"displayName"`
			Bio         string `json:"bio"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.jsonError(w, "Invalid request body", 400)
			return
		}

		profile, err := s.store.GetProfile()
		if err != nil {
			s.jsonError(w, "Failed to get profile", 500)
			return
		}

		profile.DisplayName = req.DisplayName
		profile.Bio = req.Bio

		if err := s.store.SaveProfile(profile); err != nil {
			s.jsonError(w, "Failed to save profile", 500)
			return
		}

		s.jsonResponse(w, profile)
		return
	}

	s.jsonError(w, "Method not allowed", 405)
}

func (s *Server) handleRemoteProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		s.jsonError(w, "Method not allowed", 405)
		return
	}

	peerID := r.URL.Path[len("/api/profile/"):]
	if peerID == "" {
		s.jsonError(w, "Peer ID is required", 400)
		return
	}

	profile, err := s.store.GetRemoteProfile(peerID)
	if err != nil {
		s.jsonError(w, "Failed to get profile", 500)
		return
	}

	s.jsonResponse(w, profile)
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	s.wsMutex.Lock()
	s.wsClients[conn] = true
	s.wsMutex.Unlock()

	defer func() {
		s.wsMutex.Lock()
		delete(s.wsClients, conn)
		s.wsMutex.Unlock()
		conn.Close()
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (s *Server) BroadcastEvent(eventType string, data interface{}) {
	msg := map[string]interface{}{
		"type": eventType,
		"data": data,
	}

	s.wsMutex.Lock()
	defer s.wsMutex.Unlock()

	for conn := range s.wsClients {
		err := conn.WriteJSON(msg)
		if err != nil {
			conn.Close()
			delete(s.wsClients, conn)
		}
	}
}

func (s *Server) getListeningAddrs() []string {
	var addrs []string
	for _, addr := range s.host.Addrs() {
		fullAddr := fmt.Sprintf("%s/p2p/%s", addr, s.host.ID())
		addrs = append(addrs, fullAddr)
	}
	return addrs
}

func (s *Server) Port() int {
	return s.port
}

func (s *Server) Close() error {
	os.Remove(s.portFile)
	return s.server.Close()
}
