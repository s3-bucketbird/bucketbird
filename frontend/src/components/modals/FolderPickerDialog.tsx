import { useEffect, useState } from 'react'
import Button from '../ui/Button'
import { api } from '../../api/client'

type FolderPickerDialogProps = {
  isOpen: boolean
  onClose: () => void
  onConfirm: (folderPath: string) => void
  bucketId: string
  title: string
  message?: string
  confirmText?: string
  cancelText?: string
  excludeFolders?: string[]
}

type FolderItem = {
  path: string
  name: string
  depth: number
}

export const FolderPickerDialog = ({
  isOpen,
  onClose,
  onConfirm,
  bucketId,
  title,
  message,
  confirmText = 'Select',
  cancelText = 'Cancel',
  excludeFolders = [],
}: FolderPickerDialogProps) => {
  const [selectedFolder, setSelectedFolder] = useState<string>('')
  const [folders, setFolders] = useState<FolderItem[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (isOpen) {
      setSelectedFolder('')
      loadFolders()
    }
  }, [isOpen, bucketId])

  useEffect(() => {
    if (!isOpen) return

    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose()
      }
    }

    window.addEventListener('keydown', handleEscape)
    return () => window.removeEventListener('keydown', handleEscape)
  }, [isOpen, onClose])

  const loadFolders = async () => {
    setLoading(true)
    setError(null)
    try {
      const objects = await api.getBucketObjects(bucketId, '')

      // Get all folders recursively
      const allFolders = new Set<string>()

      // Add direct folders
      objects.forEach((obj) => {
        if (obj.kind === 'folder') {
          allFolders.add(obj.key)
        }
      })

      // For each folder, also load its subfolders
      const foldersToCheck = Array.from(allFolders)
      for (const folder of foldersToCheck) {
        try {
          const subObjects = await api.getBucketObjects(bucketId, folder)
          subObjects.forEach((obj) => {
            if (obj.kind === 'folder') {
              allFolders.add(obj.key)
            }
          })
        } catch (err) {
          console.error(`Error loading folder ${folder}:`, err)
        }
      }

      // Filter out excluded folders
      const validFolders = Array.from(allFolders).filter(
        (folder) => !excludeFolders.some((excluded) => folder === excluded || folder.startsWith(excluded))
      )

      // Convert to FolderItem array with depth calculation
      const folderItems: FolderItem[] = validFolders.map((path) => {
        const depth = (path.match(/\//g) || []).length - 1
        const name = path.split('/').filter(Boolean).pop() || path
        return { path, name, depth }
      })

      // Sort by path
      folderItems.sort((a, b) => a.path.localeCompare(b.path))

      setFolders(folderItems)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load folders')
    } finally {
      setLoading(false)
    }
  }

  if (!isOpen) return null

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    onConfirm(selectedFolder)
    onClose()
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div
        className="relative w-full max-w-2xl rounded-lg bg-white shadow-xl dark:bg-slate-900"
        onClick={(e) => e.stopPropagation()}
      >
        <form onSubmit={handleSubmit} className="flex flex-col gap-4 p-6">
          <div>
            <h3 className="text-lg font-semibold text-slate-900 dark:text-white">{title}</h3>
            {message && <p className="mt-2 text-sm text-slate-600 dark:text-slate-400">{message}</p>}
          </div>

          <div className="min-h-[300px] max-h-[400px] overflow-y-auto rounded-lg border border-slate-300 dark:border-slate-600">
            {loading ? (
              <div className="flex items-center justify-center h-[300px]">
                <p className="text-sm text-slate-600 dark:text-slate-400">Loading folders...</p>
              </div>
            ) : error ? (
              <div className="flex items-center justify-center h-[300px]">
                <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
              </div>
            ) : (
              <div className="divide-y divide-slate-200 dark:divide-slate-700">
                {/* Root folder option */}
                <button
                  type="button"
                  onClick={() => setSelectedFolder('')}
                  className={`w-full text-left px-4 py-3 hover:bg-slate-50 dark:hover:bg-slate-800 transition-colors ${
                    selectedFolder === ''
                      ? 'bg-primary/10 dark:bg-primary/20 text-primary font-medium'
                      : 'text-slate-900 dark:text-white'
                  }`}
                >
                  <div className="flex items-center gap-2">
                    <svg
                      className="w-5 h-5 flex-shrink-0"
                      fill="none"
                      stroke="currentColor"
                      viewBox="0 0 24 24"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6"
                      />
                    </svg>
                    <span className="text-sm">(Root)</span>
                  </div>
                </button>

                {/* Folder list */}
                {folders.length === 0 ? (
                  <div className="px-4 py-8 text-center">
                    <p className="text-sm text-slate-600 dark:text-slate-400">No folders found</p>
                  </div>
                ) : (
                  folders.map((folder) => (
                    <button
                      key={folder.path}
                      type="button"
                      onClick={() => setSelectedFolder(folder.path)}
                      className={`w-full text-left px-4 py-3 hover:bg-slate-50 dark:hover:bg-slate-800 transition-colors ${
                        selectedFolder === folder.path
                          ? 'bg-primary/10 dark:bg-primary/20 text-primary font-medium'
                          : 'text-slate-900 dark:text-white'
                      }`}
                    >
                      <div className="flex items-center gap-2">
                        <div style={{ width: `${folder.depth * 20}px` }} className="flex-shrink-0" />
                        <svg
                          className="w-5 h-5 flex-shrink-0 text-amber-500"
                          fill="none"
                          stroke="currentColor"
                          viewBox="0 0 24 24"
                        >
                          <path
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            strokeWidth={2}
                            d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"
                          />
                        </svg>
                        <span className="text-sm truncate" title={folder.path}>
                          {folder.name}
                        </span>
                      </div>
                    </button>
                  ))
                )}
              </div>
            )}
          </div>

          <div className="flex justify-between items-center">
            <p className="text-xs text-slate-600 dark:text-slate-400">
              {selectedFolder ? (
                <span>
                  Selected: <span className="font-mono">{selectedFolder}</span>
                </span>
              ) : (
                <span>Root folder selected</span>
              )}
            </p>
            <div className="flex gap-3">
              <Button variant="outline" type="button" onClick={onClose}>
                {cancelText}
              </Button>
              <Button type="submit">
                {confirmText}
              </Button>
            </div>
          </div>
        </form>
      </div>
    </div>
  )
}
