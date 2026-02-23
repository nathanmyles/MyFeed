import { useState } from 'react'
import { useFeed, useCreatePost } from '../api/hooks'

export function FeedScreen() {
  const { data: posts, isLoading, error } = useFeed()
  const createPost = useCreatePost()
  const [content, setContent] = useState('')

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!content.trim()) return
    await createPost.mutateAsync(content.trim())
    setContent('')
  }

  if (isLoading) return <div>Loading...</div>
  if (error) return <div>Error loading feed</div>

  return (
    <div className="feed-screen">
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
              <span className="post-author">{post.authorPeerId.slice(0, 8)}...</span>
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
