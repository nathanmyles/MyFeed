export interface Post {
  id: string
  authorPeerId: string
  content: string
  createdAt: string
  attachments?: string[]
}

export interface Profile {
  peerId: string
  displayName: string
  bio: string
  avatarHash?: string
}

export interface Peer {
  peerId: string
  online: boolean
  address?: string
}

export interface Status {
  version: string
  peerId: string
  addresses: string[]
  connectedPeers: number
}

export interface Event {
  type: 'peer:discovered' | 'peer:connected' | 'peer:disconnected' | 'feed:updated'
  data?: unknown
}

class ApiClient {
  private baseUrl: string = ''
  private ws: WebSocket | null = null
  private listeners: ((event: Event) => void)[] = []

  async getPort(): Promise<number> {
    const port = await window.electron.readPortFile()
    if (!port) {
      throw new Error('Daemon port file not found. Is the daemon running?')
    }
    this.baseUrl = `http://127.0.0.1:${port}`
    return port
  }

  async ensurePort() {
    if (!this.baseUrl) {
      await this.getPort()
    }
  }

  async getStatus(): Promise<Status> {
    await this.ensurePort()
    const response = await fetch(`${this.baseUrl}/api/status`)
    return response.json()
  }

  async getFeed(): Promise<Post[]> {
    await this.ensurePort()
    const response = await fetch(`${this.baseUrl}/api/feed`)
    return response.json()
  }

  async createPost(content: string): Promise<Post> {
    await this.ensurePort()
    const response = await fetch(`${this.baseUrl}/api/posts`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ content })
    })
    return response.json()
  }

  async getPeers(): Promise<Peer[]> {
    await this.ensurePort()
    const response = await fetch(`${this.baseUrl}/api/peers`)
    return response.json()
  }

  async getProfile(): Promise<Profile> {
    await this.ensurePort()
    const response = await fetch(`${this.baseUrl}/api/profile`)
    return response.json()
  }

  async updateProfile(displayName: string, bio: string): Promise<Profile> {
    await this.ensurePort()
    const response = await fetch(`${this.baseUrl}/api/profile`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ displayName, bio })
    })
    return response.json()
  }

  async connectPeer(address: string): Promise<Peer> {
    await this.ensurePort()
    const response = await fetch(`${this.baseUrl}/api/connect`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ address })
    })
    return response.json()
  }

  connectWebSocket(onEvent: (event: Event) => void) {
    this.listeners.push(onEvent)
    
    if (this.ws) return

    const connect = async () => {
      const port = await this.getPort()
      const wsUrl = `ws://127.0.0.1:${port}/api/events`
      
      this.ws = new WebSocket(wsUrl)
      
      this.ws.onopen = () => {
        console.log('WebSocket connected')
      }
      
      this.ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data) as Event
          this.listeners.forEach(listener => listener(data))
        } catch (err) {
          console.error('Failed to parse WebSocket message:', err)
        }
      }
      
      this.ws.onclose = () => {
        console.log('WebSocket disconnected, reconnecting...')
        this.ws = null
        setTimeout(connect, 3000)
      }
      
      this.ws.onerror = (err) => {
        console.error('WebSocket error:', err)
      }
    }
    
    connect()
  }

  disconnectWebSocket() {
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
    this.listeners = []
  }
}

export const api = new ApiClient()
