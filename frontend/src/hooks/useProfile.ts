import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { api } from '../api/client'

export const useProfile = () => {
  return useQuery({
    queryKey: ['profile'],
    queryFn: ({ signal }) => api.getProfile(signal),
  })
}

export const useUpdateProfile = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (input: Parameters<typeof api.updateProfile>[0]) => api.updateProfile(input),
    onSuccess: (profile) => {
      queryClient.setQueryData(['profile'], profile)
    },
  })
}

export const useUpdatePassword = () => {
  return useMutation({
    mutationFn: (input: Parameters<typeof api.updatePassword>[0]) => api.updatePassword(input),
  })
}

export const useYouTubeCookie = () => {
  return useQuery({
    queryKey: ['youtube-cookie'],
    queryFn: ({ signal }) => api.getYouTubeCookie(signal),
  })
}

export const useUpdateYouTubeCookie = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (input: Parameters<typeof api.updateYouTubeCookie>[0]) => api.updateYouTubeCookie(input),
    onSuccess: (_, variables) => {
      queryClient.setQueryData(['youtube-cookie'], variables.cookie)
    },
  })
}

export default useProfile
