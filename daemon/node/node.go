package node

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/routing"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	"github.com/multiformats/go-multiaddr"
)

const (
	mdnsServiceName = "myfeed-social"
)

type discoveryNotifee struct {
	h   host.Host
	ctx context.Context
}

func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	n.h.Connect(n.ctx, pi)
}

type Node struct {
	Host    host.Host
	DHT     *dht.IpfsDHT
	mdnsSvc mdns.Service
	privKey crypto.PrivKey
}

func loadOrGenerateKey(keyPath string) (crypto.PrivKey, error) {
	data, err := os.ReadFile(keyPath)
	if err == nil {
		keyBytes, err := hex.DecodeString(string(data))
		if err != nil {
			return nil, err
		}
		return crypto.UnmarshalPrivateKey(keyBytes)
	}

	priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, err
	}

	keyBytes, err := crypto.MarshalPrivateKey(priv)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(keyPath), 0700); err != nil {
		return nil, err
	}

	if err := os.WriteFile(keyPath, []byte(hex.EncodeToString(keyBytes)), 0600); err != nil {
		return nil, err
	}

	return priv, nil
}

func New(ctx context.Context, dataDir string) (*Node, error) {
	keyPath := filepath.Join(dataDir, "identity.key")
	priv, err := loadOrGenerateKey(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load/generate identity: %w", err)
	}

	var kadDHT *dht.IpfsDHT

	h, err := libp2p.New(
		libp2p.Identity(priv),
		libp2p.ListenAddrStrings(
			"/ip4/0.0.0.0/tcp/0",
			"/ip4/0.0.0.0/udp/0/quic-v1",
		),
		libp2p.Security(noise.ID, noise.New),
		libp2p.DefaultTransports,
		libp2p.NATPortMap(),
		libp2p.EnableHolePunching(),
		libp2p.EnableNATService(),
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			var err error
			kadDHT, err = dht.New(ctx, h)
			return kadDHT, err
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create host: %w", err)
	}

	mdnsSvc := mdns.NewMdnsService(h, mdnsServiceName, &discoveryNotifee{h: h, ctx: ctx})

	return &Node{
		Host:    h,
		DHT:     kadDHT,
		mdnsSvc: mdnsSvc,
		privKey: priv,
	}, nil
}

func (n *Node) Close() error {
	n.mdnsSvc.Close()
	n.DHT.Close()
	return n.Host.Close()
}

func (n *Node) Advertise(ctx context.Context) error {
	return nil
}

func (n *Node) DiscoverPeers(ctx context.Context) ([]peer.AddrInfo, error) {
	var peers []peer.AddrInfo
	for _, p := range n.DHT.RoutingTable().ListPeers() {
		if p == n.Host.ID() {
			continue
		}
		peers = append(peers, peer.AddrInfo{ID: p})
	}
	return peers, nil
}

func (n *Node) GetListeningAddrs() []string {
	var addrs []string
	for _, addr := range n.Host.Addrs() {
		fullAddr := fmt.Sprintf("%s/p2p/%s", addr, n.Host.ID())
		addrs = append(addrs, fullAddr)
	}
	return addrs
}

func (n *Node) GetObservedAddrs() []string {
	var addrs []string
	for _, conn := range n.Host.Network().Conns() {
		ma := conn.RemoteMultiaddr()
		ip, _ := ma.ValueForProtocol(multiaddr.P_IP4)
		tcpPort, _ := ma.ValueForProtocol(multiaddr.P_TCP)
		if ip != "" && tcpPort != "" {
			observedAddr := fmt.Sprintf("/ip4/%s/tcp/%s/p2p/%s", ip, tcpPort, conn.RemotePeer())
			addrs = append(addrs, observedAddr)
		}
	}
	return addrs
}

func (n *Node) GetConnectedPeers() []peer.ID {
	var peers []peer.ID
	for _, conn := range n.Host.Network().Conns() {
		peers = append(peers, conn.RemotePeer())
	}
	return peers
}

func (n *Node) IsConnected(p peer.ID) bool {
	return n.Host.Network().Connectedness(p) == network.Connected
}

func (n *Node) Reachability() network.Reachability {
	return network.ReachabilityUnknown
}

func (n *Node) WaitForDHTConnection(ctx context.Context) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			for _, p := range n.DHT.RoutingTable().ListPeers() {
				if n.Host.Network().Connectedness(p) == network.Connected {
					return nil
				}
			}
		}
	}
}

func (n *Node) Sign(data []byte) (string, error) {
	sig, err := n.privKey.Sign(data)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(sig), nil
}

func (n *Node) GetPublicKey() crypto.PubKey {
	return n.privKey.GetPublic()
}

func VerifySignature(peerID string, data []byte, signatureHex string) (bool, error) {
	pid, err := peer.Decode(peerID)
	if err != nil {
		return false, err
	}

	pubKey, err := pid.ExtractPublicKey()
	if err != nil {
		return false, err
	}

	sigBytes, err := hex.DecodeString(signatureHex)
	if err != nil {
		return false, err
	}

	verified, err := pubKey.Verify(data, sigBytes)
	if err != nil {
		return false, err
	}

	return verified, nil
}
