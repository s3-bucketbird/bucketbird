import { resolveApiUrl } from '../lib/runtimeConfig'

const API_BASE_URL = resolveApiUrl()

const TOKEN_STORAGE_KEY = 'bucketbird_tokens'

function getAuthToken(): string | null {
	if (typeof window === 'undefined') {
		return null
	}
	try {
		const tokensStr = window.sessionStorage.getItem(TOKEN_STORAGE_KEY)
		if (!tokensStr) return null
		const tokens = JSON.parse(tokensStr) as { accessToken: string }
		return tokens.accessToken
	} catch {
		return null
	}
}

type FetchOptions = {
  method?: string
  body?: unknown
  signal?: AbortSignal
}

async function request<T>(path: string, options: FetchOptions = {}): Promise<T> {
  const url = new URL(path, API_BASE_URL)
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }

  // Add auth token if available
  const token = getAuthToken()
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const response = await fetch(url, {
    method: options.method ?? 'GET',
    headers,
    body: options.body ? JSON.stringify(options.body) : undefined,
    signal: options.signal,
    credentials: 'include',
  })

  if (!response.ok) {
    const message = await extractError(response)
    throw new Error(message)
  }

  // Handle 204 No Content responses
  if (response.status === 204) {
    return undefined as T
  }

  return (await response.json()) as T
}

async function extractError(response: Response): Promise<string> {
  try {
    const data = (await response.json()) as { error?: string }
    return data.error ?? `request failed with status ${response.status}`
  } catch {
    return `request failed with status ${response.status}`
  }
}

type BucketsResponse = {
  buckets: Array<{
    id: string
    name: string
    region: string
    description?: string | null
    credentialId: string
    credentialName: string
    credentialProvider: string
    createdAt: string
    size: string
  }>
}

type BucketObjectsResponse = {
  objects: Array<{
    key: string
    name: string
    kind: 'folder' | 'file'
    lastModified: string
    size: string
    icon: string
    iconColor: string
  }>
}

type CreateFolderResponse = {
  folder: {
    key: string
    name: string
    kind: 'folder'
    lastModified: string
    size: string
    icon: string
    iconColor: string
  }
}

type DeleteObjectsResponse = {
  result: {
    deleted: number
  }
}

type RenameObjectResponse = {
  result: {
    objectsMoved: number
  }
}

type YouTubeImportResponse = {
  result: {
    kind: 'video' | 'playlist'
    imported: number
    skipped: number
    totalBytes: number
    items: Array<{
      title: string
      key: string
      videoId: string
      sizeBytes: number
      contentType: string
    }>
    errors: Array<{
      title?: string
      videoId?: string
      error: string
    }>
  }
}

export type YouTubeImportProgress = {
  stage: string
  kind?: 'video' | 'playlist'
  index?: number
  total?: number
  imported?: number
  failed?: number
  totalBytes?: number
  videoTitle?: string
  videoId?: string
  message?: string
  error?: string
  destination?: string
  bytesRead?: number
  totalBytesExpected?: number
  percent?: number
  speedBytesPerSec?: number
  skipped?: boolean
  skippedCount?: number
}

type MetadataResponse = {
  metadata: {
    key: string
    sizeBytes: number
    size: string
    lastModified: string
    etag?: string
    contentType?: string
    storageClass?: string
  }
}

type CredentialsResponse = {
  credentials: Array<{
    id: string
    name: string
    provider: string
    region: string
    endpoint: string
    useSSL: boolean
    status: string
    logo?: string
  }>
}

type CredentialResponse = {
  credential: {
    id: string
    name: string
    provider: string
    region: string
    endpoint: string
    useSSL: boolean
    status: string
    logo?: string
  }
}

type CreateCredentialInput = {
  name: string
  provider: string
  region: string
  endpoint: string
  accessKey: string
  secretKey: string
  useSSL: boolean
  logo?: string
}

type UpdateCredentialInput = {
  name: string
  provider: string
  region: string
  endpoint: string
  accessKey: string
  secretKey: string
  useSSL: boolean
  logo?: string
}

type TestCredentialResponse = {
  result: {
    success: boolean
    message: string
  }
}

type DiscoveredBucketsResponse = {
  buckets: Array<{
    name: string
    createdAt?: string
  }>
}

type ProfileResponse = {
  profile: {
    id: string
    firstName: string
    lastName: string
    email: string
  }
}

type UpdateProfileInput = {
  firstName: string
  lastName: string
  email: string
}

type UpdatePasswordInput = {
  currentPassword: string
  newPassword: string
}

export type CreateBucketInput = {
  name: string
  region: string
  credentialId: string
  description?: string | null
}

export type CreateFolderInput = {
  name: string
  prefix?: string
}

export type RenameObjectInput = {
  sourceKey: string
  targetKey: string
}

export type ObjectMetadata = MetadataResponse['metadata']

export const api = {
  async getBuckets(signal?: AbortSignal) {
    const data = await request<BucketsResponse>('/api/v1/buckets', { signal })
    return data.buckets ?? []
  },
  async getBucketObjects(bucketId: string, prefix = '', signal?: AbortSignal) {
    const params = new URLSearchParams()
    if (prefix) {
      params.set('prefix', prefix)
    }
    const query = params.toString()
    const endpoint = `/api/v1/buckets/${bucketId}/objects${query ? `?${query}` : ''}`
    const data = await request<BucketObjectsResponse>(endpoint, {
      signal,
    })
    return data.objects ?? []
  },
  async searchBucketObjects(bucketId: string, query: string, signal?: AbortSignal) {
    const params = new URLSearchParams()
    params.set('q', query)
    const endpoint = `/api/v1/buckets/${bucketId}/objects/search?${params.toString()}`
    const data = await request<BucketObjectsResponse>(endpoint, {
      signal,
    })
    return data.objects ?? []
  },
  async getCredentials(signal?: AbortSignal) {
    const data = await request<CredentialsResponse>('/api/v1/credentials', { signal })
    return data.credentials ?? []
  },
  async getCredentialBuckets(credentialId: string, signal?: AbortSignal) {
    const data = await request<DiscoveredBucketsResponse>(`/api/v1/credentials/${credentialId}/buckets`, { signal })
    return data.buckets ?? []
  },
  async deleteBucket(bucketId: string, options?: { deleteRemote?: boolean }) {
    const params = new URLSearchParams()
    if (options?.deleteRemote) {
      params.set('deleteRemote', 'true')
    }
    const endpoint = `/api/v1/buckets/${bucketId}${params.toString() ? `?${params.toString()}` : ''}`
    await request<void>(endpoint, {
      method: 'DELETE',
    })
  },
  async createFolder(bucketId: string, input: CreateFolderInput) {
    const data = await request<CreateFolderResponse>(`/api/v1/buckets/${bucketId}/objects/folders`, {
      method: 'POST',
      body: input,
    })
    return data.folder
  },
  async deleteObjects(bucketId: string, keys: string[]) {
    const data = await request<DeleteObjectsResponse>(`/api/v1/buckets/${bucketId}/objects/delete`, {
      method: 'POST',
      body: { keys },
    })
    return data.result
  },
  async renameObject(bucketId: string, input: RenameObjectInput) {
    const data = await request<RenameObjectResponse>(`/api/v1/buckets/${bucketId}/objects/rename`, {
      method: 'POST',
      body: input,
    })
    return data.result
  },
  async importYouTubeWithProgress(
    bucketId: string,
    input: { url: string; destinationPrefix?: string },
    onProgress?: (progress: YouTubeImportProgress) => void,
  ) {
    const url = new URL(
      `/api/v1/buckets/${bucketId}/objects/import/youtube?stream=1`,
      API_BASE_URL,
    )

    const payload = {
      url: input.url,
      destinationPrefix: input.destinationPrefix ?? '',
    }

    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    }

    const token = getAuthToken()
    if (token) {
      headers['Authorization'] = `Bearer ${token}`
    }

    const response = await fetch(url, {
      method: 'POST',
      headers,
      body: JSON.stringify(payload),
      credentials: 'include',
    })

    if (!response.ok) {
      const message = await extractError(response)
      throw new Error(message)
    }

    if (!response.body) {
      throw new Error('Streaming is not supported in this browser.')
    }

    const reader = response.body.getReader()
    const decoder = new TextDecoder()
    let buffer = ''

    while (true) {
      const { value, done } = await reader.read()
      if (done) {
        break
      }

      buffer += decoder.decode(value, { stream: true })

      let newlineIndex = buffer.indexOf('\n')
      while (newlineIndex >= 0) {
        const line = buffer.slice(0, newlineIndex).trim()
        buffer = buffer.slice(newlineIndex + 1)
        if (line.length > 0) {
          const payload = JSON.parse(line) as {
            progress?: YouTubeImportProgress
            result?: YouTubeImportResponse['result']
            error?: string
          }
          if (payload.progress && onProgress) {
            onProgress(payload.progress)
          }
          if (payload.error) {
            throw new Error(payload.error)
          }
          if (payload.result) {
            return payload.result
          }
        }

        newlineIndex = buffer.indexOf('\n')
      }
    }

    buffer = buffer.trim()
    if (buffer.length > 0) {
      const payload = JSON.parse(buffer) as {
        progress?: YouTubeImportProgress
        result?: YouTubeImportResponse['result']
        error?: string
      }
      if (payload.error) {
        throw new Error(payload.error)
      }
      if (payload.result) {
        return payload.result
      }
    }

    throw new Error('Import ended unexpectedly without a result.')
  },
  async getObjectMetadata(bucketId: string, key: string, signal?: AbortSignal) {
    const params = new URLSearchParams({ key })
    const data = await request<MetadataResponse>(`/api/v1/buckets/${bucketId}/objects/metadata?${params.toString()}`, {
      signal,
    })
    return data.metadata
  },
  async presignObject(bucketId: string, input: { key: string; method: 'GET' | 'PUT'; expiresInSeconds?: number; contentType?: string }) {
    const payload: Record<string, unknown> = {
      key: input.key,
      method: input.method,
    }
    if (input.expiresInSeconds !== undefined) {
      payload.expiresInSeconds = input.expiresInSeconds
    }
    if (input.contentType) {
      payload.contentType = input.contentType
    }
    const data = await request<{ presign: { url: string; method: string } }>(
      `/api/v1/buckets/${bucketId}/objects/presign`,
      {
        method: 'POST',
        body: payload,
      },
    )
    return data.presign
  },
  async getCredential(id: string, signal?: AbortSignal) {
    const data = await request<CredentialResponse>(`/api/v1/credentials/${id}`, { signal })
    return data.credential
  },
  async createCredential(input: CreateCredentialInput) {
    const data = await request<CredentialResponse>('/api/v1/credentials', {
      method: 'POST',
      body: input,
    })
    return data.credential
  },
  async updateCredential(id: string, input: UpdateCredentialInput) {
    await request<void>(`/api/v1/credentials/${id}`, {
      method: 'PUT',
      body: input,
    })
  },
  async deleteCredential(id: string) {
    await request<void>(`/api/v1/credentials/${id}`, {
      method: 'DELETE',
    })
  },
  async testCredential(id: string) {
    const data = await request<TestCredentialResponse>(`/api/v1/credentials/${id}/test`, {
      method: 'POST',
    })
    return data.result
  },
  async createBucket(input: CreateBucketInput) {
    const data = await request<{ bucket: BucketsResponse['buckets'][number] }>('/api/v1/buckets', {
      method: 'POST',
      body: {
        name: input.name,
        region: input.region,
        credentialId: input.credentialId,
        description: input.description,
      },
    })
    return data.bucket
  },
  async recalculateBucketSize(bucketId: string) {
    const data = await request<{ bucket: BucketsResponse['buckets'][number] }>(`/api/v1/buckets/${bucketId}/recalculate-size`, {
      method: 'POST',
    })
    return data.bucket
  },
  async getProfile(signal?: AbortSignal) {
    const data = await request<ProfileResponse>('/api/v1/profile', { signal })
    return data.profile
  },
  async updateProfile(input: UpdateProfileInput) {
    const data = await request<ProfileResponse>('/api/v1/profile', {
      method: 'PUT',
      body: {
        firstName: input.firstName,
        lastName: input.lastName,
        email: input.email,
      },
    })
    return data.profile
  },
  async updatePassword(input: UpdatePasswordInput) {
    await request<void>('/api/v1/profile/password', {
      method: 'PUT',
      body: {
        currentPassword: input.currentPassword,
        newPassword: input.newPassword,
      },
    })
  },
  async downloadObject(bucketId: string, key: string, signal?: AbortSignal): Promise<Blob> {
    const params = new URLSearchParams({ key })
    const url = new URL(`/api/v1/buckets/${bucketId}/objects/download?${params.toString()}`, API_BASE_URL)

    const headers: Record<string, string> = {}
    const token = getAuthToken()
    if (token) {
      headers['Authorization'] = `Bearer ${token}`
    }

    const response = await fetch(url, {
      method: 'GET',
      headers,
      signal,
      credentials: 'include',
    })

    if (!response.ok) {
      const message = await extractError(response)
      throw new Error(message)
    }

    return await response.blob()
  },
  async uploadObject(
    bucketId: string,
    key: string,
    file: File,
    onProgress?: (percentage: number) => void,
    signal?: AbortSignal
  ): Promise<void> {
    const url = new URL(`/api/v1/buckets/${bucketId}/objects/upload`, API_BASE_URL)

    const formData = new FormData()
    formData.append('key', key)
    formData.append('file', file)

    const token = getAuthToken()

    console.log('[Upload] Starting upload:', {
      bucketId,
      key,
      fileName: file.name,
      fileSize: file.size,
      fileType: file.type,
      url: url.toString(),
    })

    return new Promise((resolve, reject) => {
      const xhr = new XMLHttpRequest()

      // Set timeout to 30 minutes for large files
      xhr.timeout = 30 * 60 * 1000

      // Handle abort signal
      if (signal) {
        signal.addEventListener('abort', () => {
          console.log('[Upload] Upload aborted by signal')
          xhr.abort()
          reject(new Error('Upload aborted'))
        })
      }

      // Track upload progress
      xhr.upload.addEventListener('progress', (e) => {
        if (e.lengthComputable && onProgress) {
          const percentage = (e.loaded / e.total) * 100
          console.log('[Upload] Progress:', {
            loaded: e.loaded,
            total: e.total,
            percentage: percentage.toFixed(2) + '%',
          })
          onProgress(percentage)
        }
      })

      xhr.upload.addEventListener('loadstart', () => {
        console.log('[Upload] Upload started')
      })

      xhr.upload.addEventListener('loadend', () => {
        console.log('[Upload] Upload finished, waiting for server response...')
      })

      // Handle completion
      xhr.addEventListener('load', () => {
        console.log('[Upload] Response received:', {
          status: xhr.status,
          statusText: xhr.statusText,
          responseText: xhr.responseText.substring(0, 500),
        })

        if (xhr.status >= 200 && xhr.status < 300) {
          console.log('[Upload] Upload successful!')
          resolve()
        } else {
          let message = `Upload failed with status ${xhr.status}`
          try {
            const data = JSON.parse(xhr.responseText) as { error?: string }
            if (data.error) {
              message = data.error
            }
          } catch {
            // Use default message
          }
          console.error('[Upload] Upload failed:', message)
          reject(new Error(message))
        }
      })

      // Handle errors
      xhr.addEventListener('error', (e) => {
        console.error('[Upload] Network error:', {
          readyState: xhr.readyState,
          status: xhr.status,
          statusText: xhr.statusText,
          responseText: xhr.responseText,
          event: e,
        })
        reject(new Error('Upload failed: Network error. Check browser console for details.'))
      })

      xhr.addEventListener('abort', () => {
        console.log('[Upload] Upload aborted')
        reject(new Error('Upload aborted'))
      })

      xhr.addEventListener('timeout', () => {
        console.error('[Upload] Upload timed out after 30 minutes')
        reject(new Error('Upload timed out. Please try uploading a smaller file.'))
      })

      xhr.addEventListener('loadstart', () => {
        console.log('[Upload] Request started')
      })

      xhr.addEventListener('loadend', () => {
        console.log('[Upload] Request ended')
      })

      // Open and send request
      xhr.open('POST', url.toString())
      if (token) {
        xhr.setRequestHeader('Authorization', `Bearer ${token}`)
      }
      xhr.withCredentials = true

      console.log('[Upload] Sending request...')
      xhr.send(formData)
    })
  },
}

export type ApiClient = typeof api
