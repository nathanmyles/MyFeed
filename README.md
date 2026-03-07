# MyFeed - P2P Social Network

A peer-to-peer social application with a Go libp2p daemon and Electron + React frontend. Users can create posts, follow friends, and fetch feeds directly from their devices without centralized servers. Posts are cryptographically signed for authenticity, and a friend system enables relay connections across different networks.

## Architecture

```
┌──────────────────────────────────────────────────────────┐
│                    Electron App                          │
│  ┌────────────────────────────────────────────────────┐  │
│  │                React Frontend                      │  │
│  │  • Feed Screen      • Peers Screen                 │  │
│  │  • Profile Screen   • Real-time WebSocket events   │  │
│  └────────────────────────────────────────────────────┘  │
│                            │                             │
│                      HTTP/WebSocket                      │
│                            ▼                             │
│  ┌────────────────────────────────────────────────────┐  │
│  │                Go Daemon (bundled)                 │  │
│  │  • libp2p networking (TCP/QUIC, Noise, DHT, Relay) │  │
│  │  • BadgerDB storage                                │  │
│  │  • Feed & Profile protocols                        │  │
│  │  • HTTP API server                                 │  │
│  └────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────┘
                            │
                ┌───────────┴───────────┐
                ▼                       ▼
          ┌──────────┐            ┌──────────┐
          │  Peer A  │◄──libp2p──►│  Peer B  │
          └──────────┘            └──────────┘
```

## Prerequisites

- Go 1.21+
- Node.js 18+
- pnpm

## Quick Start

```bash
# Build and run the daemon
make daemon
./daemon/bin/myfeed-daemon

# In another terminal, start the UI
make ui-dev
```

## Project Structure

```
.
├── daemon/                 # Go libp2p daemon
│   ├── node/              # libp2p host setup
│   ├── store/             # BadgerDB storage
│   ├── protocols/         # Stream protocol handlers
│   ├── sync/              # Peer sync worker
│   ├── api/               # HTTP + WebSocket server
│   ├── cmd/testclient/    # CLI test tool
│   ├── main.go            # Entry point
│   └── Dockerfile         # Headless deployment
├── ui/                    # Electron + React frontend
│   ├── electron/          # Main process (daemon mgmt)
│   ├── src/
│   │   ├── api/          # API client & hooks
│   │   ├── screens/      # Feed, Peers, Profile
│   │   └── components/   # Nav, shared components
│   └── resources/daemon/ # Bundled daemon binary
├── Makefile              # Build commands
└── README.md
```

## Daemon API

| Endpoint           | Method    | Description                             |
|--------------------|-----------|-----------------------------------------|
| `/api/status`      | GET       | Daemon info, peer ID, addresses         |
| `/api/feed`        | GET       | All posts (merged, time-sorted)         |
| `/api/posts`       | POST      | Create new post                         |
| `/api/peers`       | GET       | Discovered peers with status            |
| `/api/profile`     | GET/POST  | Get or update local profile             |
| `/api/profile/:id` | GET       | Get remote profile by peer ID           |
| `/api/friends`     | GET       | List friends                            |
| `/api/friends`     | POST      | Send friend request                     |
| `/api/friends/:id` | POST      | Approve friend request (action=approve) |
| `/api/friends/:id` | DELETE    | Remove friend                           |
| `/api/sync`        | POST      | Trigger manual sync with peers          |
| `/api/connect`     | POST      | Connect to a peer by address            |
| `/api/events`      | WebSocket | Real-time events                        |

### WebSocket Events

- `peer:discovered` - New peer found
- `peer:connected` - Peer connected
- `peer:disconnected` - Peer disconnected
- `feed:updated` - New posts available
- `friend:request` - Received friend request
- `friend:approved` - Friend request approved

## P2P Protocols

| Protocol ID                        | Description                             |
|------------------------------------|-----------------------------------------|
| `/socialapp/feed/1.0.0`            | Exchange posts (newline-delimited JSON) |
| `/socialapp/profile/1.0.0`         | Exchange profile (JSON)                 |
| `/socialapp/friend-request/1.0.0`  | Friend request (JSON)                   |
| `/socialapp/friend-approved/1.0.0` | Friend approved notification (JSON)     |

## Security & Cryptography

Posts are cryptographically signed using **Ed25519** to ensure authenticity and integrity:

- **Key Management**: Each peer has an Ed25519 key pair stored in `~/.myfeed/identity.key`. The same key is used for both libp2p identity and post signing.
- **Signing**: When creating a post, it's signed using the format `ID|Content|Timestamp`.
- **Verification**: When syncing posts from remote peers, signatures are verified using the public key derived from the author's peer ID (`peer.ID.ExtractPublicKey()`). Posts with invalid signatures are rejected.
- **Transport**: All P2P communication is encrypted using the Noise protocol.

## Friend System

MyFeed implements a friend system with mutual approval:

- **Adding Friends**: Send a friend request to any discovered peer
- **Approval Required**: Friends must be approved by the recipient before the connection is established
- **Relay Access**: Only approved friends can use your peer as a relay for connectivity
- **Cross-Network**: Friend requests are sent over P2P, enabling connections across different networks
- **Manual Address Exchange**: Since there's no central server, peers manually exchange addresses (e.g., via copy-paste) to connect across networks

### Friend Flow

1. User A discovers User B via mDNS/DHT or manual address exchange
2. User A sends a friend request to User B
3. User B sees the request and approves it
4. User A receives notification that their request was approved
5. Both users are now friends and can use each other's relay

### Relay Support

Friends can use each other's libp2p relays to establish connections when direct connectivity isn't possible (e.g., behind NATs). The relay ACL ensures only friends can use this feature.

## Development

### Daemon

```bash
cd daemon
go mod tidy
go build -o bin/myfeed-daemon .
./bin/myfeed-daemon -data /path/to/data

# Run test client
./bin/testclient -peer "/ip4/127.0.0.1/tcp/62338/p2p/12D3KooW..."
```

### UI

```bash
cd ui
pnpm install
pnpm run dev     # Development
pnpm run build   # Production build
pnpm run lint    # Lint check
```

### Build All Platforms

```bash
# Cross-compile daemon
make daemon-all

# Build production app
make build
```

## Testing P2P Locally

1. Start first daemon:
```bash
./daemon/bin/myfeed-daemon -data ~/.myfeed1
```

2. Start second daemon in another terminal:
```bash
./daemon/bin/myfeed-daemon -data ~/.myfeed2
```

Both daemons will discover each other via mDNS on the local network.

3. Use the test client to fetch from a peer:
```bash
./daemon/bin/testclient -peer "/ip4/127.0.0.1/tcp/PORT/p2p/PEER_ID"
```

## Docker

```bash
cd daemon
docker build -t myfeed-daemon .
docker run -v ~/.myfeed:/data -p 0:0 myfeed-daemon
```

## Configuration

Data is stored in `~/.myfeed/` by default:

```
~/.myfeed/
├── identity.key    # Persistent peer identity (Ed25519)
├── daemon.port     # API port (auto-generated)
├── peer.id         # Your peer ID (share this with friends)
└── db/             # BadgerDB storage
```

### Connecting Across Networks

Since there's no central signaling server, connecting peers on different networks requires manual address exchange:

1. Each user shares their peer ID and listen addresses (shown in Profile screen)
2. The other user adds these addresses to their daemon:
   ```bash
   curl -X POST http://localhost:PORT/api/connect \
     -H "Content-Type: application/json" \
     -d '{"address": "/ip4/WAN_IP/tcp/4001/p2p/PEER_ID"}'
   ```
3. Once connected, friends can use each other's relay for connectivity

## Key Dependencies

**Go Daemon:**
- `github.com/libp2p/go-libp2p` - P2P networking
- `github.com/libp2p/go-libp2p-kad-dht` - DHT routing
- `github.com/dgraph-io/badger/v4` - Embedded database
- `github.com/gorilla/websocket` - WebSocket support

**Electron UI:**
- `electron-vite` - Build tooling
- `react` + `react-router-dom` - Frontend
- `@tanstack/react-query` - Data fetching

## License

MIT
