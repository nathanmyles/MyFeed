import { useState } from 'react'
import { usePeers, useConnectPeer, useRemoteProfile, useSendFriendRequest, useFriends } from '../api/hooks'

function PeerName({ peerId }: { peerId: string }) {
  const { data: profile } = useRemoteProfile(peerId)
  const displayName = profile?.displayName || 'Unknown User'
  
  return (
    <span title={peerId}>{displayName}</span>
  )
}

function PeerActions({ peerId, friendStatus, onFriendRequestSent }: { peerId: string, friendStatus: 'none' | 'pending' | 'friend', onFriendRequestSent: () => void }) {
  const sendFriendRequest = useSendFriendRequest()

  const handleAddFriend = async () => {
    try {
      await sendFriendRequest.mutateAsync(peerId)
      onFriendRequestSent()
    } catch (err) {
      console.error('Failed to send friend request:', err)
    }
  }

  if (friendStatus === 'friend') {
    return <span className="friend-status">Friend</span>
  }

  if (friendStatus === 'pending') {
    return <span className="friend-status pending">Pending</span>
  }

  return (
    <button 
      onClick={handleAddFriend}
      disabled={sendFriendRequest.isPending}
      className="add-friend-btn"
    >
      {sendFriendRequest.isPending ? '...' : 'Add Friend'}
    </button>
  )
}

export function PeersScreen() {
  const { data: peers, isLoading, error, refetch } = usePeers()
  const { data: friendsData } = useFriends()
  const connectPeer = useConnectPeer()
  const [address, setAddress] = useState('')
  const [error2, setError] = useState('')
  const [friendRequestSent, setFriendRequestSent] = useState('')

  const getFriendStatus = (peerId: string): 'none' | 'pending' | 'friend' => {
    if (friendsData?.friends?.some(f => f.peerId === peerId)) {
      return 'friend'
    }
    if (friendsData?.pendingRequests?.some(r => r.peerId === peerId)) {
      return 'pending'
    }
    return 'none'
  }

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

      <div className="add-friend-section">
        <h3>Add Friend</h3>
        <p>Enter a friend's address from their Profile page to connect</p>
        <form onSubmit={handleConnect} className="connect-form">
          <input
            type="text"
            value={address}
            onChange={(e) => setAddress(e.target.value)}
            placeholder="/ip4/1.2.3.4/tcp/12345/p2p/12D3KooW..."
          />
          <button type="submit" disabled={connectPeer.isPending || !address.trim()}>
            {connectPeer.isPending ? 'Connecting...' : 'Add Friend'}
          </button>
        </form>
        {error2 && <div className="connect-error">{error2}</div>}
      </div>

      <div className="peers-list">
        {peers?.map((peer) => (
          <div key={peer.peerId} className="peer">
            <div className="peer-info">
              <div className="peer-status">
                <span className={`status-dot ${peer.online ? 'online' : 'offline'}`} />
                {peer.online ? 'Online' : 'Offline'}
              </div>
              <div className="peer-name">
                <PeerName peerId={peer.peerId} />
              </div>
            </div>
            <PeerActions 
              peerId={peer.peerId}
              friendStatus={getFriendStatus(peer.peerId)}
              onFriendRequestSent={() => setFriendRequestSent(peer.peerId)} 
            />
          </div>
        ))}
        {friendRequestSent && (
          <div className="friend-request-sent">
            Friend request sent!
          </div>
        )}
        {peers?.length === 0 && (
          <div className="no-peers">
            No peers connected. Enter a peer address above or wait for mDNS discovery on your local network.
          </div>
        )}
      </div>
    </div>
  )
}
