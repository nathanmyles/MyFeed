import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../api/client'

export function useStatus() {
  return useQuery({
    queryKey: ['status'],
    queryFn: () => api.getStatus(),
    refetchInterval: 5000
  })
}

export function useFeed() {
  return useQuery({
    queryKey: ['feed'],
    queryFn: () => api.getFeed()
  })
}

export function useCreatePost() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: (content: string) => api.createPost(content),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['feed'] })
    }
  })
}

export function usePeers() {
  return useQuery({
    queryKey: ['peers'],
    queryFn: () => api.getPeers(),
    refetchInterval: 5000
  })
}

export function useProfile() {
  return useQuery({
    queryKey: ['profile'],
    queryFn: () => api.getProfile()
  })
}

export function useRemoteProfile(peerId: string) {
  return useQuery({
    queryKey: ['profile', peerId],
    queryFn: () => api.getRemoteProfile(peerId),
    enabled: !!peerId
  })
}

export function useSyncFeed() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: () => api.syncFeed(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['feed'] })
    }
  })
}

export function useUpdateProfile() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: ({ displayName, bio }: { displayName: string; bio: string }) => 
      api.updateProfile(displayName, bio),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['profile'] })
    }
  })
}

export function useConnectPeer() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: (address: string) => api.connectPeer(address),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['peers'] })
    }
  })
}

export function useFriends() {
  return useQuery({
    queryKey: ['friends'],
    queryFn: () => api.getFriends()
  })
}

export function useSendFriendRequest() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: (peerId: string) => api.sendFriendRequest(peerId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['friends'] })
    }
  })
}

export function useApproveFriend() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: (peerId: string) => api.approveFriend(peerId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['friends'] })
    }
  })
}

export function useRemoveFriend() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: (peerId: string) => api.removeFriend(peerId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['friends'] })
    }
  })
}
