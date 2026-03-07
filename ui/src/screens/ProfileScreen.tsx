import { useState, useEffect } from 'react'
import { useProfile, useUpdateProfile, useStatus, useFriends, useSendFriendRequest, useApproveFriend, useRemoveFriend } from '../api/hooks'

function FriendRequest({ friend, onApprove, onReject }: { friend: any, onApprove: () => void, onReject: () => void }) {
  return (
    <div className="friend-request">
      <span className="friend-peer-id">{friend.peerId}</span>
      <div className="friend-actions">
        <button onClick={onApprove} className="approve-btn">Approve</button>
        <button onClick={onReject} className="reject-btn">Reject</button>
      </div>
    </div>
  )
}

function FriendList({ friends, onRemove }: { friends: any[], onRemove: (peerId: string) => void }) {
  return (
    <div className="friends-list">
      {friends.map(friend => (
        <div key={friend.peerId} className="friend-item">
          <span className="friend-peer-id" title={friend.peerId}>{friend.peerId.slice(0, 12)}...</span>
          <button onClick={() => onRemove(friend.peerId)} className="remove-btn">Remove</button>
        </div>
      ))}
    </div>
  )
}

export function ProfileScreen() {
  const { data: profile, isLoading } = useProfile()
  const { data: status } = useStatus()
  const { data: friendsData } = useFriends()
  const updateProfile = useUpdateProfile()
  const sendFriendRequest = useSendFriendRequest()
  const approveFriend = useApproveFriend()
  const removeFriend = useRemoveFriend()
  const [displayName, setDisplayName] = useState('')
  const [bio, setBio] = useState('')
  const [friendPeerId, setFriendPeerId] = useState('')

  useEffect(() => {
    if (profile) {
      setDisplayName(profile.displayName || '')
      setBio(profile.bio || '')
    }
  }, [profile])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    await updateProfile.mutateAsync({ displayName, bio })
  }

  const handleSendFriendRequest = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!friendPeerId.trim()) return
    try {
      await sendFriendRequest.mutateAsync(friendPeerId.trim())
      setFriendPeerId('')
    } catch (err) {
      console.error('Failed to send friend request:', err)
    }
  }

  const handleApprove = async (peerId: string) => {
    await approveFriend.mutateAsync(peerId)
  }

  const handleReject = async (peerId: string) => {
    await removeFriend.mutateAsync(peerId)
  }

  const handleRemove = async (peerId: string) => {
    await removeFriend.mutateAsync(peerId)
  }

  if (isLoading) return <div>Loading...</div>

  return (
    <div className="profile-screen">
      <h2>Your Profile</h2>

      <div className="peer-id-section">
        <label>Your Peer ID:</label>
        <div className="peer-id">{status?.peerId || profile?.peerId}</div>
      </div>

      <div className="peer-address-section">
        <label>Your Addresses:</label>
        {status?.addresses?.map((addr, i) => (
          <code key={i} className="peer-address">{addr}</code>
        ))}
        <small>Share a local address (192.168.x.x) with friends on the same network, or public address for remote connections</small>
      </div>

      <form onSubmit={handleSubmit} className="profile-form">
        <div className="form-group">
          <label htmlFor="displayName">Display Name</label>
          <input
            id="displayName"
            type="text"
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
            placeholder="Your name"
          />
        </div>

        <div className="form-group">
          <label htmlFor="bio">Bio</label>
          <textarea
            id="bio"
            value={bio}
            onChange={(e) => setBio(e.target.value)}
            placeholder="Tell us about yourself"
            rows={4}
          />
        </div>

        <button type="submit" disabled={updateProfile.isPending}>
          {updateProfile.isPending ? 'Saving...' : 'Save Profile'}
        </button>
        {updateProfile.isSuccess && <span className="success">Saved!</span>}
      </form>

      <div className="friends-section">
        <h3>Friends</h3>
        
        <div className="add-friend-form">
          <input
            type="text"
            value={friendPeerId}
            onChange={(e) => setFriendPeerId(e.target.value)}
            placeholder="Enter friend's Peer ID to send request"
          />
          <button 
            type="button" 
            onClick={handleSendFriendRequest}
            disabled={sendFriendRequest.isPending || !friendPeerId.trim()}
          >
            {sendFriendRequest.isPending ? 'Sending...' : 'Add Friend'}
          </button>
        </div>

        {friendsData?.pendingRequests && friendsData.pendingRequests.length > 0 && (
          <div className="pending-requests">
            <h4>Pending Requests</h4>
            {friendsData.pendingRequests.map(req => (
              <FriendRequest 
                key={req.peerId} 
                friend={req} 
                onApprove={() => handleApprove(req.peerId)}
                onReject={() => handleReject(req.peerId)}
              />
            ))}
          </div>
        )}

        {friendsData?.friends && friendsData.friends.length > 0 && (
          <div className="friends-list-section">
            <h4>Your Friends</h4>
            <FriendList friends={friendsData.friends} onRemove={handleRemove} />
          </div>
        )}
      </div>
    </div>
  )
}
