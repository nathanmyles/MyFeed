import { useState } from 'react'
import { useFeed, useCreatePost, useSyncFeed, useStatus, useProfile, useRemoteProfile } from '../api/hooks'

function PostAuthor({ peerId }: { peerId: string }) {
  const { data: status } = useStatus()
  const isLocal = status?.peerId === peerId
  const { data: profile } = isLocal ? useProfile() : useRemoteProfile(peerId)
  const displayName = profile?.displayName || 'Unknown User'
  
  return (
    <span className="post-author" title={"Peer ID: " + peerId}>
      {displayName}
    </span>
  )
}

export function FeedScreen() {
  const { data: posts, isLoading, error } = useFeed()
  const createPost = useCreatePost()
  const syncFeed = useSyncFeed()
  const [content, setContent] = useState('')

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!content.trim()) return
    await createPost.mutateAsync(content.trim())
    setContent('')
  }

  const handleSync = async () => {
    await syncFeed.mutateAsync()
  }

  if (isLoading) return <div>Loading...</div>
  if (error) return <div>Error loading feed</div>

  return (
    <div className="feed-screen">
      <div className="feed-header">
        <button onClick={handleSync} disabled={syncFeed.isPending}>
          {syncFeed.isPending ? 'Syncing...' : 'Sync'}
        </button>
      </div>
      <form onSubmit={handleSubmit} className="post-form">
        <textarea
          value={content}
          onChange={(e) => setContent(e.target.value)}
          placeholder="What's happening?"
          rows={3}
        />
        <button type="submit" disabled={createPost.isPending || !content.trim()}>
          {createPost.isPending ? 'Posting...' : 'Post'}
        </button>
      </form>

      <div className="posts-list">
        {posts?.map((post) => (
          <div key={post.id} className="post">
            <div className="post-header">
              <PostAuthor peerId={post.authorPeerId} />
              <span className="post-time">
                {new Date(post.createdAt).toLocaleString()}
              </span>
            </div>
            <div className="post-content">{post.content}</div>
          </div>
        ))}
        {posts?.length === 0 && <div className="no-posts">No posts yet</div>}
      </div>
    </div>
  )
}
