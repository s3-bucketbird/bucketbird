import { useMutation, useQueryClient } from '@tanstack/react-query'

import { api, type BulkObjectOperationItem, type CopyObjectInput, type CreateFolderInput, type RenameObjectInput } from '../api/client'

export const useCreateFolder = (bucketId: string, prefix: string) => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (input: CreateFolderInput) => api.createFolder(bucketId, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['bucketObjects', bucketId, prefix] })
    },
  })
}

export const useDeleteObjects = (bucketId: string, prefix: string) => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (keys: string[]) => api.deleteObjects(bucketId, keys),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['bucketObjects', bucketId, prefix] })
    },
  })
}

export const useRenameObject = (bucketId: string, prefix: string) => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (input: RenameObjectInput) => api.renameObject(bucketId, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['bucketObjects', bucketId, prefix] })
    },
  })
}

export const useCopyObject = (bucketId: string, prefix: string) => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (input: CopyObjectInput) => api.copyObject(bucketId, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['bucketObjects', bucketId, prefix] })
    },
  })
}

export const useMoveObjects = (bucketId: string, prefix: string) => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (items: BulkObjectOperationItem[]) => api.moveObjects(bucketId, items),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['bucketObjects', bucketId, prefix] })
    },
  })
}

export const useCopyObjectsBulk = (bucketId: string, prefix: string) => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (items: BulkObjectOperationItem[]) => api.copyObjectsBulk(bucketId, items),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['bucketObjects', bucketId, prefix] })
    },
  })
}
