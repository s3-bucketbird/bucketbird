import React, { type ChangeEvent, type DragEvent } from 'react'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useParams, useSearchParams } from 'react-router-dom'
import { useQueryClient, useQuery } from '@tanstack/react-query'

import AppShell from '../components/layout/AppShell'
import Button from '../components/ui/Button'
import { api } from '../api/client'
import { useBucketObjects } from '../hooks/useBucketObjects'
import { useBuckets } from '../hooks/useBuckets'
import { useCreateFolder, useDeleteObjects, useRenameObject } from '../hooks/useObjectActions'
import { ConfirmDialog } from '../components/modals/ConfirmDialog'
import { PromptDialog } from '../components/modals/PromptDialog'
import { AlertDialog } from '../components/modals/AlertDialog'

type BucketObject = {
  key: string
  name: string
  kind: 'folder' | 'file'
  lastModified: string
  size: string
  icon: string
  iconColor: string
}

const ensureTrailingSlash = (value: string) => {
  if (value === '') return ''
  return value.endsWith('/') ? value : `${value}/`
}

const parentPrefixOf = (key: string) => {
  const cleaned = key.endsWith('/') ? key.slice(0, -1) : key
  const lastSlash = cleaned.lastIndexOf('/')
  if (lastSlash === -1) {
    return ''
  }
  return `${cleaned.slice(0, lastSlash + 1)}`
}

const getFileExtension = (filename: string): string => {
  const lastDot = filename.lastIndexOf('.')
  return lastDot > 0 ? filename.substring(lastDot + 1).toLowerCase() : ''
}

const canPreview = (filename: string): boolean => {
  const ext = getFileExtension(filename)
  const previewableExtensions = [
    // Images
    'jpg', 'jpeg', 'png', 'gif', 'webp', 'svg', 'bmp', 'ico',
    // Videos
    'mp4', 'webm', 'ogg', 'mov',
    // Audio
    'mp3', 'wav', 'ogg', 'm4a', 'aac',
    // Text
    'txt', 'md', 'json', 'xml', 'csv', 'log', 'yml', 'yaml',
    // Code
    'js', 'jsx', 'ts', 'tsx', 'css', 'scss', 'html', 'py', 'go', 'java', 'c', 'cpp', 'h', 'sh', 'bash',
    // Documents
    'pdf'
  ]
  return previewableExtensions.includes(ext)
}

const getFileType = (filename: string): 'image' | 'video' | 'audio' | 'text' | 'pdf' | 'unknown' => {
  const ext = getFileExtension(filename)

  if (['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg', 'bmp', 'ico'].includes(ext)) return 'image'
  if (['mp4', 'webm', 'ogg', 'mov'].includes(ext)) return 'video'
  if (['mp3', 'wav', 'ogg', 'm4a', 'aac'].includes(ext)) return 'audio'
  if (['pdf'].includes(ext)) return 'pdf'
  if (['txt', 'md', 'json', 'xml', 'csv', 'log', 'yml', 'yaml', 'js', 'jsx', 'ts', 'tsx', 'css', 'scss', 'html', 'py', 'go', 'java', 'c', 'cpp', 'h', 'sh', 'bash'].includes(ext)) return 'text'

  return 'unknown'
}

const BucketPage = () => {
  const { bucketId: bucketIdParam } = useParams<{ bucketId: string }>()
  const bucketId = bucketIdParam ?? ''
  const [searchParams, setSearchParams] = useSearchParams()
  const queryClient = useQueryClient()
  const { data: buckets = [] } = useBuckets()
  const bucketMeta = buckets.find((bucket) => bucket.id === bucketId)

  // Get prefix from URL or default to empty
  const prefix = searchParams.get('prefix') ?? ''

  const [openObjectMenu, setOpenObjectMenu] = useState<string | null>(null)
  const [isDragActive, setIsDragActive] = useState(false)
  const [isUploading, setIsUploading] = useState(false)
  const [uploadProgress, setUploadProgress] = useState<Record<string, number>>({})
  const [uploadStatus, setUploadStatus] = useState<Record<string, 'uploading' | 'success' | 'error'>>({})
  const [uploadErrors, setUploadErrors] = useState<Record<string, string>>({})
  const [isUploadMinimized, setIsUploadMinimized] = useState(false)
  const [isUploadPanelExpanded, setIsUploadPanelExpanded] = useState(false)
  const [previewFile, setPreviewFile] = useState<{ key: string; name: string; url: string } | null>(null)
  const [isLoadingPreview, setIsLoadingPreview] = useState(false)
  const [isDownloading, setIsDownloading] = useState(false)
  const [downloadMessage, setDownloadMessage] = useState('Downloading file...')
  const [sortBy, setSortBy] = useState<'name' | 'modified' | 'size' | 'type'>('name')
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('asc')
  const [selectedKeys, setSelectedKeys] = useState<Set<string>>(new Set())
  const [searchQuery, setSearchQuery] = useState('')
  const fileInputRef = useRef<HTMLInputElement | null>(null)

  // Dialog states
  const [confirmDialog, setConfirmDialog] = useState<{
    isOpen: boolean
    title: string
    message: string
    onConfirm: () => void
  }>({ isOpen: false, title: '', message: '', onConfirm: () => {} })

  const [promptDialog, setPromptDialog] = useState<{
    isOpen: boolean
    title: string
    message?: string
    defaultValue?: string
    placeholder?: string
    onConfirm: (value: string) => void
  }>({ isOpen: false, title: '', onConfirm: () => {} })

  const [alertDialog, setAlertDialog] = useState<{
    isOpen: boolean
    title: string
    message: string
    variant?: 'error' | 'success' | 'info' | 'warning'
  }>({ isOpen: false, title: '', message: '' })

  // Function to update prefix and URL
  const setPrefix = useCallback((newPrefix: string) => {
    if (newPrefix === '') {
      setSearchParams({})
    } else {
      setSearchParams({ prefix: newPrefix })
    }
  }, [setSearchParams])

  const createFolderMutation = useCreateFolder(bucketId, prefix)
  const deleteObjectsMutation = useDeleteObjects(bucketId, prefix)
  const renameObjectMutation = useRenameObject(bucketId, prefix)

  const {
    data: objects = [],
    isLoading,
    isError,
    error,
    refetch,
    isFetching,
  } = useBucketObjects(bucketId, prefix)

  // Recursive search query
  const {
    data: searchResults = [],
    isLoading: isSearchLoading,
    isError: isSearchError,
    error: searchError,
  } = useQuery({
    queryKey: ['bucketSearch', bucketId, searchQuery],
    queryFn: ({ signal }) => api.searchBucketObjects(bucketId, searchQuery, signal),
    enabled: !!searchQuery && searchQuery.trim().length > 0,
  })

  // Use search results when searching, otherwise use regular objects
  const displayObjects = searchQuery ? searchResults : objects
  const displayIsLoading = searchQuery ? isSearchLoading : isLoading
  const displayIsError = searchQuery ? isSearchError : isError
  const displayError = searchQuery ? searchError : error
  const showUploadDropzone = isUploadPanelExpanded || isDragActive || isUploading

  const fileCount = useMemo(() => {
    return displayObjects.filter((obj) => obj.kind === 'file').length
  }, [displayObjects])

  // Sort objects (no client-side filtering needed anymore since backend does it)
  const sortedObjects = useMemo(() => {
    const sorted = [...displayObjects].sort((a, b) => {
      // Always keep folders before files
      if (a.kind !== b.kind) {
        return a.kind === 'folder' ? -1 : 1
      }

      let comparison = 0

      switch (sortBy) {
        case 'name':
          comparison = a.name.localeCompare(b.name, undefined, { numeric: true, sensitivity: 'base' })
          break
        case 'modified': {
          const aTime = a.lastModified && a.lastModified !== '0001-01-01T00:00:00Z' ? new Date(a.lastModified).getTime() : 0
          const bTime = b.lastModified && b.lastModified !== '0001-01-01T00:00:00Z' ? new Date(b.lastModified).getTime() : 0
          comparison = aTime - bTime
          break
        }
        case 'size': {
          // Extract numeric value from size string (e.g., "1.2 MB" -> 1200000)
          const parseSize = (sizeStr: string): number => {
            if (!sizeStr || sizeStr.trim() === '') return 0
            const match = sizeStr.match(/^([\d.]+)\s*([KMGT]?B)?/i)
            if (!match) return 0
            const value = parseFloat(match[1])
            const unit = match[2]?.toUpperCase() || 'B'
            const multipliers: Record<string, number> = { B: 1, KB: 1024, MB: 1024 ** 2, GB: 1024 ** 3, TB: 1024 ** 4 }
            return value * (multipliers[unit] || 1)
          }
          comparison = parseSize(a.size) - parseSize(b.size)
          break
        }
        case 'type': {
          const aExt = getFileExtension(a.name)
          const bExt = getFileExtension(b.name)
          comparison = aExt.localeCompare(bExt)
          break
        }
      }

      return sortDirection === 'asc' ? comparison : -comparison
    })

    return sorted
  }, [displayObjects, sortBy, sortDirection])

  const handleSort = useCallback((column: 'name' | 'modified' | 'size' | 'type') => {
    if (sortBy === column) {
      setSortDirection((prev) => (prev === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortBy(column)
      setSortDirection('asc')
    }
  }, [sortBy])

  // Checkbox selection handlers
  const handleToggleSelect = useCallback((key: string) => {
    setSelectedKeys((prev) => {
      const next = new Set(prev)
      if (next.has(key)) {
        next.delete(key)
      } else {
        next.add(key)
      }
      return next
    })
  }, [])

  const handleSelectAll = useCallback(() => {
    if (selectedKeys.size === sortedObjects.length) {
      setSelectedKeys(new Set())
    } else {
      setSelectedKeys(new Set(sortedObjects.map((obj) => obj.key)))
    }
  }, [selectedKeys.size, sortedObjects])

  const handleBulkDelete = useCallback(() => {
    if (selectedKeys.size === 0) return

    const count = selectedKeys.size
    setConfirmDialog({
      isOpen: true,
      title: 'Delete Items',
      message: `Are you sure you want to delete ${count} item${count > 1 ? 's' : ''}? This action cannot be undone.`,
      onConfirm: async () => {
        try {
          await deleteObjectsMutation.mutateAsync(Array.from(selectedKeys))
          setSelectedKeys(new Set())
        } catch (err) {
          setAlertDialog({
            isOpen: true,
            title: 'Delete Failed',
            message: (err as Error).message,
            variant: 'error',
          })
        }
      },
    })
  }, [selectedKeys, deleteObjectsMutation])

  const breadcrumbItems = useMemo(() => {
    const rootLabel = bucketMeta?.name ?? bucketId
    const items: Array<{ label: string; path: string }> = [{ label: rootLabel, path: '' }]
    if (!prefix) {
      return items
    }
    const traversable = prefix.endsWith('/') ? prefix : `${prefix}/`
    let running = ''
    for (let idx = 0; idx < traversable.length; idx += 1) {
      const char = traversable[idx]
      running += char
      if (char !== '/') {
        continue
      }
      const trimmed = running.endsWith('/') ? running.slice(0, -1) : running
      const raw = trimmed.substring(trimmed.lastIndexOf('/') + 1)
      const label = raw === '' ? '(empty)' : raw
      items.push({ label, path: running })
    }
    return items
  }, [bucketMeta?.name, bucketId, prefix])

  useEffect(() => {
    setOpenObjectMenu(null)
    setSelectedKeys(new Set())
    setSearchQuery('')
    setIsUploadPanelExpanded(false)
  }, [prefix, bucketId])

  useEffect(() => {
    if (!openObjectMenu) {
      return
    }
    const handleClickAway = (event: MouseEvent) => {
      const target = event.target as HTMLElement | null
      if (target && target.closest('[data-object-actions]')) {
        return
      }
      setOpenObjectMenu(null)
    }
    window.addEventListener('click', handleClickAway, true)
    return () => window.removeEventListener('click', handleClickAway, true)
  }, [openObjectMenu])

  const handleCreateFolder = useCallback(() => {
    setPromptDialog({
      isOpen: true,
      title: 'Create New Folder',
      placeholder: 'Enter folder name',
      onConfirm: async (name) => {
        try {
          await createFolderMutation.mutateAsync({ name, prefix })
        } catch (err) {
          setAlertDialog({
            isOpen: true,
            title: 'Create Folder Failed',
            message: (err as Error).message,
            variant: 'error',
          })
        }
      },
    })
  }, [createFolderMutation, prefix])

  const handleDeleteObject = useCallback(
    (key: string) => {
      setOpenObjectMenu(null)
      setConfirmDialog({
        isOpen: true,
        title: 'Delete Item',
        message: 'Are you sure you want to delete this item? This action cannot be undone.',
        onConfirm: async () => {
          try {
            await deleteObjectsMutation.mutateAsync([key])
          } catch (err) {
            setAlertDialog({
              isOpen: true,
              title: 'Delete Failed',
              message: (err as Error).message,
              variant: 'error',
            })
          }
        },
      })
    },
    [deleteObjectsMutation],
  )

  const handleRenameObject = useCallback(
    (obj: BucketObject) => {
      setOpenObjectMenu(null)
      setPromptDialog({
        isOpen: true,
        title: 'Rename',
        message: `Rename "${obj.name}" to:`,
        defaultValue: obj.name,
        placeholder: 'Enter new name',
        onConfirm: async (newName) => {
          if (newName === obj.name) return

          const parent = parentPrefixOf(obj.key)
          let targetKey = `${parent}${newName}`
          if (obj.kind === 'folder') {
            targetKey = ensureTrailingSlash(targetKey)
          }
          try {
            await renameObjectMutation.mutateAsync({ sourceKey: obj.key, targetKey })
          } catch (err) {
            setAlertDialog({
              isOpen: true,
              title: 'Rename Failed',
              message: (err as Error).message,
              variant: 'error',
            })
          }
        },
      })
    },
    [renameObjectMutation],
  )

  const handleDownloadObject = useCallback(
    async (key: string, name: string, isFolder: boolean) => {
      try {
        setDownloadMessage(isFolder ? 'Preparing folder download...' : 'Downloading file...')
        setIsDownloading(true)
        setOpenObjectMenu(null)

        const blob = await api.downloadObject(bucketId, key)
        const url = URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        const safeName = name && name.trim().length > 0 ? name : 'download'
        a.download = isFolder ? `${safeName}.zip` : safeName
        document.body.appendChild(a)
        a.click()
        document.body.removeChild(a)
        URL.revokeObjectURL(url)
      } catch (err) {
        setAlertDialog({
          isOpen: true,
          title: 'Download Failed',
          message: (err as Error).message,
          variant: 'error',
        })
      } finally {
        setDownloadMessage('Downloading file...')
        setIsDownloading(false)
      }
    },
    [bucketId],
  )

  const handlePreviewFile = useCallback(
    async (key: string, name: string) => {
      try {
        setIsLoadingPreview(true)
        setOpenObjectMenu(null)

        const blob = await api.downloadObject(bucketId, key)
        const url = URL.createObjectURL(blob)

        // Clean up previous object URL if it exists
        if (previewFile?.url) {
          URL.revokeObjectURL(previewFile.url)
        }

        setPreviewFile({ key, name, url })
      } catch (err) {
        setAlertDialog({
          isOpen: true,
          title: 'Preview Failed',
          message: `Failed to load preview: ${(err as Error).message}`,
          variant: 'error',
        })
        setPreviewFile(null)
      } finally {
        setIsLoadingPreview(false)
      }
    },
    [bucketId, previewFile],
  )

  // Get list of previewable files for navigation
  const previewableFiles = useMemo(() => {
    return sortedObjects.filter((obj) => obj.kind === 'file' && canPreview(obj.name))
  }, [sortedObjects])

  // Navigate to next/previous file
  const navigatePreview = useCallback(
    async (direction: 'next' | 'prev') => {
      if (!previewFile || previewableFiles.length === 0) return

      const currentIndex = previewableFiles.findIndex((f) => f.key === previewFile.key)
      if (currentIndex === -1) return

      let newIndex: number
      if (direction === 'next') {
        newIndex = (currentIndex + 1) % previewableFiles.length
      } else {
        newIndex = (currentIndex - 1 + previewableFiles.length) % previewableFiles.length
      }

      const nextFile = previewableFiles[newIndex]
      await handlePreviewFile(nextFile.key, nextFile.name)
    },
    [previewFile, previewableFiles, handlePreviewFile],
  )

  // Keyboard navigation
  useEffect(() => {
    if (!previewFile) return

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setPreviewFile(null)
      } else if (e.key === 'ArrowRight') {
        navigatePreview('next')
      } else if (e.key === 'ArrowLeft') {
        navigatePreview('prev')
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [previewFile, navigatePreview])

  // Cleanup object URLs when preview is closed or component unmounts
  useEffect(() => {
    return () => {
      if (previewFile?.url) {
        URL.revokeObjectURL(previewFile.url)
      }
    }
  }, [previewFile?.url])

  const handleViewDetails = useCallback(
    async (key: string) => {
      setOpenObjectMenu(null)
      try {
        const metadata = await api.getObjectMetadata(bucketId, key)
        setAlertDialog({
          isOpen: true,
          title: 'Object Details',
          message: `Key: ${metadata.key}\nSize: ${metadata.size} (${metadata.sizeBytes} bytes)\nLast Modified: ${metadata.lastModified}\nContent Type: ${metadata.contentType ?? 'n/a'}\nStorage Class: ${metadata.storageClass ?? 'n/a'}\nETag: ${metadata.etag ?? 'n/a'}`,
          variant: 'info',
        })
      } catch (err) {
        setAlertDialog({
          isOpen: true,
          title: 'Failed to Load Details',
          message: (err as Error).message,
          variant: 'error',
        })
      }
    },
    [bucketId],
  )

  const handleFilesUpload = useCallback(
    async (incoming: FileList | File[]) => {
      const files = Array.from(incoming)
      if (files.length === 0 || !bucketId) {
        return
      }

      setIsUploading(true)
      setUploadProgress({})
      setUploadStatus({})
      setUploadErrors({})

      // Initialize all files
      const initialProgress: Record<string, number> = {}
      const initialStatus: Record<string, 'uploading' | 'success' | 'error'> = {}
      files.forEach((file) => {
        initialProgress[file.name] = 0
        initialStatus[file.name] = 'uploading'
      })
      setUploadProgress(initialProgress)
      setUploadStatus(initialStatus)

      const CONCURRENT_UPLOADS = 3 // Upload 3 files at a time
      const uploadQueue = [...files]
      const activeUploads: Promise<void>[] = []
      let successCount = 0
      let errorCount = 0

      const uploadFile = async (file: File) => {
        const base = ensureTrailingSlash(prefix)
        const key = `${base}${file.name}`

        try {
          console.log('[Upload] Starting upload:', file.name)

          await api.uploadObject(bucketId, key, file, (percentage) => {
            setUploadProgress((prev) => ({ ...prev, [file.name]: percentage }))
          })

          console.log('[Upload] Upload successful:', file.name)
          setUploadStatus((prev) => ({ ...prev, [file.name]: 'success' }))
          setUploadProgress((prev) => ({ ...prev, [file.name]: 100 }))
          successCount++
        } catch (err) {
          console.error('[Upload] Upload failed:', file.name, err)
          setUploadStatus((prev) => ({ ...prev, [file.name]: 'error' }))
          setUploadErrors((prev) => ({ ...prev, [file.name]: (err as Error).message }))
          errorCount++
        }
      }

      // Process queue with concurrent limit
      while (uploadQueue.length > 0 || activeUploads.length > 0) {
        // Fill up to concurrent limit
        while (activeUploads.length < CONCURRENT_UPLOADS && uploadQueue.length > 0) {
          const file = uploadQueue.shift()!
          const uploadPromise = uploadFile(file).then(() => {
            // Remove from active uploads when done
            const index = activeUploads.indexOf(uploadPromise)
            if (index > -1) activeUploads.splice(index, 1)
          })
          activeUploads.push(uploadPromise)
        }

        // Wait for at least one to complete
        if (activeUploads.length > 0) {
          await Promise.race(activeUploads)
        }
      }

      console.log(`[Upload] All uploads complete. Success: ${successCount}, Errors: ${errorCount}`)

      // Refresh the file list
      await queryClient.invalidateQueries({ queryKey: ['bucketObjects', bucketId, prefix] })

      // Keep notification visible for errors, auto-hide for all success
      if (errorCount === 0) {
        setTimeout(() => {
          setUploadProgress({})
          setUploadStatus({})
          setUploadErrors({})
          setIsUploading(false)
        }, 2000)
      } else {
        // Keep visible if there were errors
        setIsUploading(false)
      }
    },
    [bucketId, prefix, queryClient],
  )

  const handleBrowseFiles = useCallback(() => {
    setIsUploadPanelExpanded(true)
    fileInputRef.current?.click()
  }, [])

  const handleFileInputChange = useCallback(
    async (event: ChangeEvent<HTMLInputElement>) => {
      if (event.target.files) {
        await handleFilesUpload(event.target.files)
        event.target.value = ''
      }
    },
    [handleFilesUpload],
  )

  const handleDragOver = useCallback((event: DragEvent<HTMLDivElement>) => {
    event.preventDefault()
    setIsDragActive(true)
  }, [])

  const handleDragLeave = useCallback((event: DragEvent<HTMLDivElement>) => {
    event.preventDefault()
    setIsDragActive(false)
  }, [])

  const handleDrop = useCallback(
    async (event: DragEvent<HTMLDivElement>) => {
      event.preventDefault()
      setIsDragActive(false)
      setIsUploadPanelExpanded(true)
      if (event.dataTransfer.files && event.dataTransfer.files.length > 0) {
        await handleFilesUpload(event.dataTransfer.files)
      }
    },
    [handleFilesUpload],
  )

  const isActionPending =
    createFolderMutation.isPending || deleteObjectsMutation.isPending || renameObjectMutation.isPending || isUploading

  return (
    <AppShell searchPlaceholder="Search files and folders..." searchValue={searchQuery} onSearchChange={setSearchQuery}>
      <input ref={fileInputRef} type="file" multiple className="hidden" onChange={handleFileInputChange} />
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-3xl font-bold text-slate-900 dark:text-white">{bucketMeta?.name ?? bucketId}</h1>
        <div className="flex flex-wrap gap-2">
          {selectedKeys.size > 0 && (
            <Button
              variant="outline"
              className="border-red-300 text-red-600 hover:bg-red-50 dark:border-red-700 dark:text-red-400 dark:hover:bg-red-900/20"
              onClick={handleBulkDelete}
              disabled={deleteObjectsMutation.isPending}
            >
              <span className="material-symbols-outlined text-xl">delete</span>
              <span className="truncate">Delete ({selectedKeys.size})</span>
            </Button>
          )}
          <Button
            variant="outline"
            className="border-slate-300 text-slate-800 hover:bg-slate-50 dark:border-slate-700 dark:text-white dark:hover:bg-slate-800"
            onClick={handleCreateFolder}
            disabled={isActionPending}
          >
            <span className="material-symbols-outlined text-xl">create_new_folder</span>
            <span className="truncate">New Folder</span>
          </Button>
          <Button onClick={handleBrowseFiles} disabled={isUploading}>
            <span className="material-symbols-outlined text-xl">upload_file</span>
            <span className="truncate">Upload Files</span>
          </Button>
        </div>
      </div>

      <p className="mt-2 text-sm text-slate-500 dark:text-slate-400">
        {displayIsError
          ? 'Unable to load files.'
          : displayIsLoading
            ? 'Loading file count…'
            : searchQuery
              ? `${fileCount} result${fileCount === 1 ? '' : 's'} found`
              : `${fileCount} file${fileCount === 1 ? '' : 's'} in this ${prefix ? 'folder' : 'bucket root'}`}
      </p>

      <nav className="mt-4 flex flex-wrap items-center gap-2 text-sm">
        {breadcrumbItems.map((item, index) => {
          const isActive = index === breadcrumbItems.length - 1
          return (
            <div key={`${item.label}-${index}`} className="flex items-center gap-2">
              <button
                type="button"
                onClick={() => !isActive && setPrefix(item.path)}
                disabled={isActive}
                className={`rounded-md px-2 py-1 text-left transition-colors ${
                  isActive
                    ? 'cursor-default bg-slate-100 text-slate-800 dark:bg-slate-800/40 dark:text-white'
                    : 'text-slate-500 hover:bg-slate-100 hover:text-primary dark:text-slate-400 dark:hover:bg-slate-800'
                }`}
              >
                {item.label}
              </button>
              {index < breadcrumbItems.length - 1 && (
                <span className="text-slate-400 dark:text-slate-500">/</span>
              )}
            </div>
          )
        })}
      </nav>

      <section
        className={`mt-4 rounded-xl border-2 border-dashed transition-all transition-colors ${
          isDragActive
            ? 'border-primary bg-primary/10'
            : 'border-slate-300 bg-slate-50 dark:border-slate-700 dark:bg-slate-800/20'
        } ${showUploadDropzone ? 'p-6' : 'p-4'}`}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
      >
        <div className="flex flex-col gap-4">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div className="flex items-start gap-3">
              <div
                className={`flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-full transition-colors ${
                  showUploadDropzone
                    ? 'bg-primary/15 text-primary dark:bg-primary/25 dark:text-white'
                    : 'bg-white text-primary dark:bg-slate-800 dark:text-white/80'
                }`}
              >
                <span className="material-symbols-outlined text-2xl">upload</span>
              </div>
              <div>
                <h3 className="text-base font-semibold text-slate-800 dark:text-white">Upload files</h3>
                <p className="text-sm text-slate-500 dark:text-slate-400">
                  {showUploadDropzone
                    ? isUploading
                      ? 'Uploading files…'
                      : 'Drag files into the dropzone below to start uploading.'
                    : 'Drag files here or expand the dropzone when you need it.'}
                </p>
              </div>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <button
                type="button"
                onClick={() => setIsUploadPanelExpanded((prev) => !prev)}
                className="flex items-center gap-1 rounded-lg border border-transparent px-3 py-2 text-sm font-medium text-slate-500 transition-colors hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50 focus-visible:ring-offset-2 focus-visible:ring-offset-white dark:text-slate-300 dark:hover:text-primary dark:focus-visible:ring-offset-slate-900"
                aria-expanded={showUploadDropzone}
              >
                <span className="material-symbols-outlined text-base">
                  {showUploadDropzone ? 'expand_less' : 'expand_more'}
                </span>
                {showUploadDropzone ? 'Hide dropzone' : 'Show dropzone'}
              </button>
              <Button
                onClick={handleBrowseFiles}
                disabled={isUploading}
                variant="outline"
                className="border-slate-300 text-slate-700 hover:border-primary hover:text-primary dark:border-slate-600 dark:text-white dark:hover:border-primary"
              >
                <span className="material-symbols-outlined text-base">upload_file</span>
                <span>Select files</span>
              </Button>
            </div>
          </div>
          {showUploadDropzone && (
            <div className="flex flex-col items-center justify-center gap-3 rounded-lg border border-dashed border-slate-300 bg-white/70 p-6 text-center dark:border-slate-700 dark:bg-slate-900/60">
              <p className="text-sm text-slate-600 dark:text-slate-300">
                {isUploading
                  ? 'Files are uploading in the background. You can keep exploring your bucket.'
                  : 'Drop files anywhere in this area to begin uploading.'}
              </p>
              {!isUploading && (
                <p className="text-xs text-slate-400 dark:text-slate-500">
                  You can also use Select files to browse your device.
                </p>
              )}
            </div>
          )}
        </div>
      </section>

      <div className="mt-8 flex-1 overflow-x-auto">
        <table className="w-full text-left">
          <thead>
            <tr className="border-b border-slate-200 text-xs uppercase text-slate-500 dark:border-slate-800 dark:text-slate-400">
              <th className="px-4 py-3">
                <input
                  type="checkbox"
                  className="form-checkbox rounded border-slate-400 bg-transparent text-primary focus:ring-primary dark:border-slate-600"
                  checked={sortedObjects.length > 0 && selectedKeys.size === sortedObjects.length}
                  onChange={handleSelectAll}
                  disabled={sortedObjects.length === 0}
                />
              </th>
              <th className="px-4 py-3 font-medium">
                <button
                  type="button"
                  onClick={() => handleSort('name')}
                  className="flex items-center gap-1 hover:text-primary transition-colors"
                >
                  Name
                  {sortBy === 'name' && (
                    <span className="material-symbols-outlined text-sm">
                      {sortDirection === 'asc' ? 'arrow_upward' : 'arrow_downward'}
                    </span>
                  )}
                </button>
              </th>
              <th className="px-4 py-3 font-medium">
                <button
                  type="button"
                  onClick={() => handleSort('modified')}
                  className="flex items-center gap-1 hover:text-primary transition-colors"
                >
                  Last Modified
                  {sortBy === 'modified' && (
                    <span className="material-symbols-outlined text-sm">
                      {sortDirection === 'asc' ? 'arrow_upward' : 'arrow_downward'}
                    </span>
                  )}
                </button>
              </th>
              <th className="px-4 py-3 font-medium">
                <button
                  type="button"
                  onClick={() => handleSort('size')}
                  className="flex items-center gap-1 hover:text-primary transition-colors"
                >
                  File Size
                  {sortBy === 'size' && (
                    <span className="material-symbols-outlined text-sm">
                      {sortDirection === 'asc' ? 'arrow_upward' : 'arrow_downward'}
                    </span>
                  )}
                </button>
              </th>
              <th className="px-4 py-3 font-medium">
                <button
                  type="button"
                  onClick={() => handleSort('type')}
                  className="flex items-center gap-1 hover:text-primary transition-colors"
                >
                  Type
                  {sortBy === 'type' && (
                    <span className="material-symbols-outlined text-sm">
                      {sortDirection === 'asc' ? 'arrow_upward' : 'arrow_downward'}
                    </span>
                  )}
                </button>
              </th>
              <th className="px-4 py-3 font-medium" />
            </tr>
          </thead>
          <tbody className="text-sm text-slate-700 dark:text-slate-300">
            {displayIsLoading && (
              <tr>
                <td colSpan={7} className="px-4 py-6 text-center text-sm text-slate-500 dark:text-slate-400">
                  {searchQuery ? 'Searching...' : 'Loading objects…'}
                </td>
              </tr>
            )}
            {displayIsError && (
              <tr>
                <td colSpan={7} className="px-4 py-6 text-center text-sm text-red-500 dark:text-red-400">
                  {(displayError as Error).message}
                  <button
                    className="ml-3 text-sm font-semibold text-primary hover:underline"
                    type="button"
                    onClick={() => refetch()}
                    disabled={isFetching}
                  >
                    Retry
                  </button>
                </td>
              </tr>
            )}
            {!displayIsLoading && !displayIsError && sortedObjects.length === 0 && searchQuery && (
              <tr>
                <td colSpan={7} className="px-4 py-6 text-center text-sm text-slate-500 dark:text-slate-400">
                  No files found matching "{searchQuery}" in this bucket
                </td>
              </tr>
            )}
            {!displayIsLoading && !displayIsError && sortedObjects.length === 0 && !searchQuery && (
              <tr>
                <td colSpan={7} className="px-4 py-6 text-center text-sm text-slate-500 dark:text-slate-400">
                  This folder is empty. Drag and drop files or use the upload actions above.
                </td>
              </tr>
            )}
            {sortedObjects.map((obj) => {
              const isFolder = obj.kind === 'folder'
              const isMenuOpen = openObjectMenu === obj.key
              const lastModifiedLabel = obj.lastModified && obj.lastModified !== '0001-01-01T00:00:00Z'
                ? new Date(obj.lastModified).toLocaleString()
                : '—'
              const sizeLabel = obj.size && obj.size.trim() !== '' ? obj.size : '—'
              const typeLabel = isFolder ? 'Folder' : (getFileExtension(obj.name).toUpperCase() || '—')

              return (
                <tr
                  key={obj.key}
                  className={`border-b border-slate-200 transition-colors dark:border-slate-800 ${
                    isFolder || canPreview(obj.name) ? 'cursor-pointer hover:bg-slate-100 dark:hover:bg-slate-800/60' : 'hover:bg-slate-50 dark:hover:bg-slate-800/50'
                  }`}
                  onClick={() => {
                    if (isFolder) {
                      setPrefix(obj.key)
                    } else if (canPreview(obj.name)) {
                      handlePreviewFile(obj.key, obj.name)
                    }
                  }}
                >
                  <td className="px-4 py-3" onClick={(e) => e.stopPropagation()}>
                    <input
                      type="checkbox"
                      className="form-checkbox rounded border-slate-400 bg-transparent text-primary focus:ring-primary dark:border-slate-600"
                      checked={selectedKeys.has(obj.key)}
                      onChange={() => handleToggleSelect(obj.key)}
                    />
                  </td>
                  <td className="px-4 py-3 font-medium">
                    <div className="flex items-center gap-3">
                      <span className={`material-symbols-outlined text-xl ${obj.iconColor}`}>{obj.icon}</span>
                      <span className={isFolder ? 'text-primary' : undefined}>{obj.name}</span>
                    </div>
                  </td>
                  <td className="px-4 py-3 text-slate-500 dark:text-slate-400">{lastModifiedLabel}</td>
                  <td className="px-4 py-3 text-slate-500 dark:text-slate-400">{sizeLabel}</td>
                  <td className="px-4 py-3 text-slate-500 dark:text-slate-400">{typeLabel}</td>
                  <td
                    className="relative px-4 py-3 text-right"
                    onClick={(event) => event.stopPropagation()}
                    data-object-actions={obj.key}
                  >
                    <button
                      type="button"
                      className={`rounded-full p-2 text-slate-500 transition-colors hover:bg-slate-200 dark:text-slate-400 dark:hover:bg-slate-700 ${
                        isMenuOpen ? 'bg-slate-200 text-primary dark:bg-slate-700/60 dark:text-primary' : ''
                      }`}
                      onClick={(event) => {
                        event.stopPropagation()
                        const nativeEvent = event.nativeEvent as MouseEvent
                        if (nativeEvent.stopImmediatePropagation) {
                          nativeEvent.stopImmediatePropagation()
                        }
                        setOpenObjectMenu((prev) => (prev === obj.key ? null : obj.key))
                      }}
                      aria-haspopup="menu"
                      aria-expanded={isMenuOpen}
                    >
                      <span className="material-symbols-outlined text-xl">more_horiz</span>
                    </button>
                    {isMenuOpen && (
                      <div
                        role="menu"
                        className="absolute right-0 top-full z-20 mt-2 w-48 rounded-lg border border-slate-200 bg-white p-1 shadow-lg dark:border-slate-700 dark:bg-slate-900"
                        onClick={(event) => event.stopPropagation()}
                      >
                        {!isFolder && canPreview(obj.name) && (
                          <button
                            className="flex w-full items-center gap-3 rounded-md px-4 py-2 text-left text-sm text-slate-700 hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-800"
                            onClick={() => handlePreviewFile(obj.key, obj.name)}
                          >
                            <span className="material-symbols-outlined text-base">visibility</span>
                            <span>Preview</span>
                          </button>
                        )}
                        <button
                          className="flex w-full items-center gap-3 rounded-md px-4 py-2 text-left text-sm text-slate-700 hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-800"
                          onClick={() => handleDownloadObject(obj.key, obj.name, isFolder)}
                        >
                          <span className="material-symbols-outlined text-base">download</span>
                          <span>Download</span>
                        </button>
                        <button
                          className="flex w-full items-center gap-3 rounded-md px-4 py-2 text-left text-sm text-slate-700 hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-800"
                          onClick={() => handleRenameObject(obj)}
                          disabled={renameObjectMutation.isPending}
                        >
                          <span className="material-symbols-outlined text-base">drive_file_rename_outline</span>
                          <span>Rename</span>
                        </button>
                        <button
                          className="flex w-full items-center gap-3 rounded-md px-4 py-2 text-left text-sm text-slate-700 hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-800"
                          onClick={() => handleViewDetails(obj.key)}
                        >
                          <span className="material-symbols-outlined text-base">info</span>
                          <span>View details</span>
                        </button>
                        <div className="my-1 border-t border-slate-200 dark:border-slate-700" />
                        <button
                          className="flex w-full items-center gap-3 rounded-md px-4 py-2 text-left text-sm text-red-500 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-500/10"
                          onClick={() => handleDeleteObject(obj.key)}
                          disabled={deleteObjectsMutation.isPending}
                        >
                          <span className="material-symbols-outlined text-base">delete</span>
                          <span>Delete</span>
                        </button>
                      </div>
                    )}
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>

      {/* Upload Progress - Fixed Corner Notification */}
      {(isUploading || Object.keys(uploadProgress).length > 0) && (
        <div className="fixed bottom-4 right-4 z-50 w-96 max-w-[calc(100vw-2rem)] rounded-lg bg-white shadow-2xl dark:bg-slate-900 border border-slate-200 dark:border-slate-700">
          {/* Header */}
          <div className="flex items-center justify-between border-b border-slate-200 bg-slate-50 px-4 py-3 dark:border-slate-700 dark:bg-slate-800/50">
            <div className="flex items-center gap-2">
              {isUploading ? (
                <span className="material-symbols-outlined text-primary animate-pulse">upload</span>
              ) : Object.values(uploadStatus).some((s) => s === 'error') ? (
                <span className="material-symbols-outlined text-red-500">error</span>
              ) : (
                <span className="material-symbols-outlined text-green-500">check_circle</span>
              )}
              <h3 className="font-semibold text-slate-900 dark:text-white">
                {isUploading
                  ? `Uploading ${Object.keys(uploadProgress).length} file${Object.keys(uploadProgress).length > 1 ? 's' : ''}`
                  : Object.values(uploadStatus).some((s) => s === 'error')
                    ? 'Upload Complete with Errors'
                    : 'Upload Complete'}
              </h3>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-sm text-slate-500 dark:text-slate-400">
                {Object.values(uploadStatus).filter((s) => s === 'success').length} /{' '}
                {Object.keys(uploadProgress).length}
              </span>
              <button
                onClick={() => setIsUploadMinimized(!isUploadMinimized)}
                className="rounded p-1 text-slate-500 hover:bg-slate-200 dark:text-slate-400 dark:hover:bg-slate-700"
                title={isUploadMinimized ? 'Expand' : 'Minimize'}
              >
                <span className="material-symbols-outlined text-lg">
                  {isUploadMinimized ? 'expand_more' : 'expand_less'}
                </span>
              </button>
              {!isUploading && (
                <button
                  onClick={() => {
                    setUploadProgress({})
                    setUploadStatus({})
                    setUploadErrors({})
                  }}
                  className="rounded p-1 text-slate-500 hover:bg-slate-200 dark:text-slate-400 dark:hover:bg-slate-700"
                  title="Close"
                >
                  <span className="material-symbols-outlined text-lg">close</span>
                </button>
              )}
            </div>
          </div>

          {/* Content - Collapsible */}
          {!isUploadMinimized && (
            <div className="max-h-96 overflow-y-auto p-4">
              <div className="space-y-3">
                {Object.entries(uploadProgress).map(([filename, percentage]) => {
                  const status = uploadStatus[filename]
                  const error = uploadErrors[filename]

                  return (
                    <div key={filename}>
                      <div className="mb-1 flex items-center justify-between gap-2 text-sm">
                        <div className="flex items-center gap-2 min-w-0 flex-1">
                          {status === 'success' && (
                            <span className="material-symbols-outlined text-green-500 text-base flex-shrink-0">
                              check_circle
                            </span>
                          )}
                          {status === 'error' && (
                            <span className="material-symbols-outlined text-red-500 text-base flex-shrink-0">
                              error
                            </span>
                          )}
                          {status === 'uploading' && (
                            <span className="material-symbols-outlined text-primary text-base flex-shrink-0 animate-pulse">
                              upload
                            </span>
                          )}
                          <span
                            className={`truncate font-medium ${
                              status === 'error'
                                ? 'text-red-600 dark:text-red-400'
                                : 'text-slate-700 dark:text-slate-300'
                            }`}
                            title={filename}
                          >
                            {filename}
                          </span>
                        </div>
                        <span className="flex-shrink-0 text-slate-500 dark:text-slate-400">
                          {status === 'success' ? (
                            'Done'
                          ) : status === 'error' ? (
                            'Failed'
                          ) : percentage === 100 ? (
                            <span className="animate-pulse">Processing...</span>
                          ) : (
                            `${Math.round(percentage)}%`
                          )}
                        </span>
                      </div>
                      {status === 'uploading' && (
                        <div className="h-1.5 overflow-hidden rounded-full bg-slate-200 dark:bg-slate-700">
                          <div
                            className={`h-full rounded-full transition-all duration-300 ${
                              percentage === 100 ? 'bg-primary/70 animate-pulse' : 'bg-primary'
                            }`}
                            style={{ width: `${percentage}%` }}
                          />
                        </div>
                      )}
                      {error && (
                        <p className="mt-1 text-xs text-red-600 dark:text-red-400" title={error}>
                          {error.length > 60 ? `${error.substring(0, 60)}...` : error}
                        </p>
                      )}
                    </div>
                  )
                })}
              </div>
              <p className="mt-3 text-xs text-slate-500 dark:text-slate-400">
                {isUploading
                  ? 'You can continue browsing while files upload in the background'
                  : Object.values(uploadStatus).some((s) => s === 'error')
                    ? 'Some files failed to upload. Check the errors above.'
                    : 'All files uploaded successfully!'}
              </p>
            </div>
          )}
        </div>
      )}

      {/* Preview Modal */}
      {previewFile && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 p-4"
          onClick={() => setPreviewFile(null)}
        >
          {/* Previous Button */}
          {previewableFiles.length > 1 && (
            <button
              onClick={(e) => {
                e.stopPropagation()
                navigatePreview('prev')
              }}
              className="absolute left-4 z-50 rounded-full bg-white/90 p-3 text-slate-900 shadow-lg transition-all hover:bg-white hover:scale-110 dark:bg-slate-800/90 dark:text-white dark:hover:bg-slate-800"
              title="Previous (←)"
            >
              <span className="material-symbols-outlined text-3xl">chevron_left</span>
            </button>
          )}

          <div
            className="relative max-h-[90vh] max-w-[90vw] overflow-auto rounded-lg bg-white dark:bg-slate-900"
            onClick={(e) => e.stopPropagation()}
          >
            {/* Header */}
            <div className="sticky top-0 z-10 flex items-center justify-between border-b border-slate-200 bg-white px-6 py-4 dark:border-slate-700 dark:bg-slate-900">
              <div className="flex items-center gap-4">
                <h2 className="text-xl font-semibold text-slate-900 dark:text-white">{previewFile.name}</h2>
                {previewableFiles.length > 1 && (
                  <span className="text-sm text-slate-500 dark:text-slate-400">
                    {previewableFiles.findIndex((f) => f.key === previewFile.key) + 1} / {previewableFiles.length}
                  </span>
                )}
              </div>
              <button
                onClick={() => setPreviewFile(null)}
                className="rounded-full p-2 text-slate-500 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-800"
                title="Close (Esc)"
              >
                <span className="material-symbols-outlined text-2xl">close</span>
              </button>
            </div>

            {/* Content */}
            <div className="p-6">
              {getFileType(previewFile.name) === 'image' && (
                <img
                  src={previewFile.url}
                  alt={previewFile.name}
                  className="max-h-[70vh] max-w-full object-contain"
                />
              )}

              {getFileType(previewFile.name) === 'video' && (
                <video
                  src={previewFile.url}
                  controls
                  className="max-h-[70vh] max-w-full"
                >
                  Your browser does not support the video tag.
                </video>
              )}

              {getFileType(previewFile.name) === 'audio' && (
                <audio src={previewFile.url} controls className="w-full">
                  Your browser does not support the audio tag.
                </audio>
              )}

              {getFileType(previewFile.name) === 'pdf' && (
                <iframe
                  src={previewFile.url}
                  className="h-[70vh] w-full min-w-[600px]"
                  title={previewFile.name}
                />
              )}

              {getFileType(previewFile.name) === 'text' && (
                <PreviewTextFile url={previewFile.url} />
              )}
            </div>
          </div>

          {/* Next Button */}
          {previewableFiles.length > 1 && (
            <button
              onClick={(e) => {
                e.stopPropagation()
                navigatePreview('next')
              }}
              className="absolute right-4 z-50 rounded-full bg-white/90 p-3 text-slate-900 shadow-lg transition-all hover:bg-white hover:scale-110 dark:bg-slate-800/90 dark:text-white dark:hover:bg-slate-800"
              title="Next (→)"
            >
              <span className="material-symbols-outlined text-3xl">chevron_right</span>
            </button>
          )}
        </div>
      )}

      {/* Loading Overlay for Preview/Download */}
      {(isLoadingPreview || isDownloading) && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
          <div className="rounded-lg bg-white p-6 shadow-2xl dark:bg-slate-900">
            <div className="flex flex-col items-center gap-4">
              <div className="h-12 w-12 animate-spin rounded-full border-4 border-slate-200 border-t-primary dark:border-slate-700 dark:border-t-primary" />
              <p className="text-sm font-medium text-slate-900 dark:text-white">
                {isLoadingPreview ? 'Loading preview...' : downloadMessage}
              </p>
            </div>
          </div>
        </div>
      )}

      {/* Custom Dialogs */}
      <ConfirmDialog
        isOpen={confirmDialog.isOpen}
        onClose={() => setConfirmDialog({ ...confirmDialog, isOpen: false })}
        onConfirm={confirmDialog.onConfirm}
        title={confirmDialog.title}
        message={confirmDialog.message}
      />

      <PromptDialog
        isOpen={promptDialog.isOpen}
        onClose={() => setPromptDialog({ ...promptDialog, isOpen: false })}
        onConfirm={promptDialog.onConfirm}
        title={promptDialog.title}
        message={promptDialog.message}
        defaultValue={promptDialog.defaultValue}
        placeholder={promptDialog.placeholder}
      />

      <AlertDialog
        isOpen={alertDialog.isOpen}
        onClose={() => setAlertDialog({ ...alertDialog, isOpen: false })}
        title={alertDialog.title}
        message={alertDialog.message}
        variant={alertDialog.variant}
      />
    </AppShell>
  )
}

// Text file preview component
const PreviewTextFile = ({ url }: { url: string }) => {
  const [content, setContent] = React.useState<string>('Loading...')
  const [error, setError] = React.useState<string | null>(null)

  React.useEffect(() => {
    fetch(url)
      .then((res) => {
        if (!res.ok) throw new Error('Failed to load file')
        return res.text()
      })
      .then(setContent)
      .catch((err) => setError(err.message))
  }, [url])

  if (error) {
    return <div className="text-red-500">Error: {error}</div>
  }

  return (
    <pre className="max-h-[70vh] overflow-auto rounded-lg bg-slate-50 p-4 text-sm dark:bg-slate-800">
      <code>{content}</code>
    </pre>
  )
}

export default BucketPage
