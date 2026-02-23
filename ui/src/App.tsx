import { useEffect } from 'react'
import { Routes, Route } from 'react-router-dom'
import { Nav } from './components/Nav'
import { FeedScreen } from './screens/FeedScreen'
import { PeersScreen } from './screens/PeersScreen'
import { ProfileScreen } from './screens/ProfileScreen'
import { api } from './api/client'
import { useQueryClient } from '@tanstack/react-query'

function App() {
  const queryClient = useQueryClient()

  useEffect(() => {
    api.connectWebSocket((event) => {
      if (event.type === 'feed:updated') {
        queryClient.invalidateQueries({ queryKey: ['feed'] })
      } else if (event.type === 'peer:discovered' || event.type === 'peer:connected' || event.type === 'peer:disconnected') {
        queryClient.invalidateQueries({ queryKey: ['peers'] })
      }
    })

    return () => {
      api.disconnectWebSocket()
    }
  }, [queryClient])

  return (
    <div className="app">
      <Nav />
      <main className="main-content">
        <Routes>
          <Route path="/" element={<FeedScreen />} />
          <Route path="/peers" element={<PeersScreen />} />
          <Route path="/profile" element={<ProfileScreen />} />
        </Routes>
      </main>
    </div>
  )
}

export default App
