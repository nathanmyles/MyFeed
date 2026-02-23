# MyFeed - P2P Social Network

A peer-to-peer social application with a Go libp2p daemon and Electron + React frontend. Users can create posts, follow friends, and fetch feeds directly from their devices without centralized servers.

## Architecture

```
┌───────────────────────────────────────────────────────┐
│                    Electron App                       │
│  ┌─────────────────────────────────────────────────┐  │
│  │              React Frontend                     │  │
│  │  • Feed Screen    • Peers Screen                │  │
│  │  • Profile Screen • Real-time WebSocket events  │  │
│  └─────────────────────────────────────────────────┘  │
│                          │                            │
│                    HTTP/WebSocket                     │
│                          ▼                            │
│  ┌─────────────────────────────────────────────────┐  │
│  │              Go Daemon (bundled)                │  │
│  │  • libp2p networking (TCP/QUIC, Noise, DHT)     │  │
│  │  • BadgerDB storage                             │  │
│  │  • Feed & Profile protocols                     │  │
│  │  • HTTP API server                              │  │
│  └─────────────────────────────────────────────────┘  │
└───────────────────────────────────────────────────────┘
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

| Endpoint       | Method    | Description                          |
|----------------|-----------|--------------------------------------|
| `/api/status`  | GET       | Daemon info, peer ID, addresses      |
| `/api/feed`    | GET       | All posts (merged, time-sorted)      |
| `/api/posts`   | POST      | Create new post                      |
| `/api/peers`   | GET       | Discovered peers with status         |
| `/api/profile` | GET/POST  | Get or update profile                |
| `/api/events`  | WebSocket | Real-time events                     |

### WebSocket Events

- `peer:discovered` - New peer found
- `peer:connected` - Peer connected
- `peer:disconnected` - Peer disconnected
- `feed:updated` - New posts available

## P2P Protocols

| Protocol ID                   | Description                             |
|-------------------------------|-----------------------------------------|
| `/socialapp/feed/1.0.0`       | Exchange posts (newline-delimited JSON) |
| `/socialapp/profile/1.0.0`    | Exchange profile (JSON)                 |

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
├── identity.key    # Persistent peer identity
├── daemon.port     # API port (auto-generated)
└── db/             # BadgerDB storage
```

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
