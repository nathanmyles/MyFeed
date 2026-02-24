import { useState } from 'react'
import { usePeers, useConnectPeer, useRemoteProfile } from '../api/hooks'

function PeerName({ peerId }: { peerId: string }) {
  const { data: profile } = useRemoteProfile(peerId)
  const displayName = profile?.displayName || 'Unknown User'
  
  return (
    <span title={peerId}>{displayName}</span>
  )
}

export function PeersScreen() {
  const { data: peers, isLoading, error, refetch } = usePeers()
  const connectPeer = useConnectPeer()
  const [address, setAddress] = useState('')
  const [error2, setError] = useState('')

  const handleConnect = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    if (!address.trim()) return
    
    try {
      await connectPeer.mutateAsync(address.trim())
      setAddress('')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to connect')
    }
  }

  if (isLoading) return <div>Loading...</div>
  if (error) return <div>Error loading peers</div>

  return (
    <div className="peers-screen">
      <div className="peers-header">
        <h2>Discovered Peers</h2>
        <button onClick={() => refetch()}>Refresh</button>
      </div>

      <form onSubmit={handleConnect} className="connect-form">
        <input
          type="text"
          value={address}
          onChange={(e) => setAddress(e.target.value)}
          placeholder="Peer address (e.g., /ip4/127.0.0.1/tcp/12345/p2p/12D3KooW...)"
        />
        <button type="submit" disabled={connectPeer.isPending || !address.trim()}>
          {connectPeer.isPending ? 'Connecting...' : 'Connect'}
        </button>
        {error2 && <div className="connect-error">{error2}</div>}
      </form>

      <div className="peers-list">
        {peers?.map((peer) => (
          <div key={peer.peerId} className="peer">
            <div className="peer-status">
              <span className={`status-dot ${peer.online ? 'online' : 'offline'}`} />
              {peer.online ? 'Online' : 'Offline'}
            </div>
            <div className="peer-name">
              <PeerName peerId={peer.peerId} />
            </div>
          </div>
        ))}
        {peers?.length === 0 && (
          <div className="no-peers">
            No peers connected. Enter a peer address above or wait for mDNS discovery on your local network.
          </div>
        )}
      </div>
    </div>
  )
}
