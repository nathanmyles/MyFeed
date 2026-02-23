import { useState, useEffect } from 'react'
import { useProfile, useUpdateProfile, useStatus } from '../api/hooks'

export function ProfileScreen() {
  const { data: profile, isLoading } = useProfile()
  const { data: status } = useStatus()
  const updateProfile = useUpdateProfile()
  const [displayName, setDisplayName] = useState('')
  const [bio, setBio] = useState('')

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

  if (isLoading) return <div>Loading...</div>

  return (
    <div className="profile-screen">
      <h2>Your Profile</h2>

      <div className="peer-id-section">
        <label>Your Peer ID:</label>
        <div className="peer-id">{status?.peerId || profile?.peerId}</div>
        <small>Share this with friends so they can connect to you</small>
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
    </div>
  )
}
