import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'

import AppShell from '../components/layout/AppShell'
import Button from '../components/ui/Button'
import { useBuckets } from '../hooks/useBuckets'
import { useDeleteBucket, useRecalculateBucketSize } from '../hooks/useBucketMutations'
import { useBucketModal } from '../hooks/useBucketModal'
import { ConfirmDialog } from '../components/modals/ConfirmDialog'
import { AlertDialog } from '../components/modals/AlertDialog'

const VIEW_MODES = [
  { key: 'list', icon: 'list', label: 'List view' },
  { key: 'grid', icon: 'grid_view', label: 'Grid view' },
] as const

type ViewMode = (typeof VIEW_MODES)[number]['key']

const DashboardPage = () => {
  const {
    data: buckets = [],
    isLoading,
    isError,
    error,
    refetch,
    isFetching,
  } = useBuckets()
  const navigate = useNavigate()
  const deleteBucketMutation = useDeleteBucket()
  const recalculateSizeMutation = useRecalculateBucketSize()
  const { openCreateModal } = useBucketModal()
  const [viewMode, setViewMode] = useState<ViewMode>('grid')
  const [openActionsFor, setOpenActionsFor] = useState<string | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [confirmDialog, setConfirmDialog] = useState<{
    isOpen: boolean
    bucketId: string | null
    bucketName: string
    deleteRemote: boolean
  }>({ isOpen: false, bucketId: null, bucketName: '', deleteRemote: false })
  const [alertDialog, setAlertDialog] = useState<{
    isOpen: boolean
    title: string
    message: string
    variant?: 'error' | 'success' | 'info' | 'warning'
  }>({ isOpen: false, title: '', message: '' })
  type BucketEntity = (typeof buckets)[number]

  // Filter buckets based on search query
  const filteredBuckets = searchQuery
    ? buckets.filter((bucket) =>
        bucket.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
        bucket.credentialName.toLowerCase().includes(searchQuery.toLowerCase()) ||
        bucket.region.toLowerCase().includes(searchQuery.toLowerCase())
      )
    : buckets

  useEffect(() => {
    if (!openActionsFor) {
      return
    }
    const handleClickAway = (event: MouseEvent) => {
      const target = event.target as HTMLElement | null
      if (target && target.closest('[data-bucket-actions]')) {
        return
      }
      setOpenActionsFor(null)
    }
    window.addEventListener('click', handleClickAway, true)
    return () => window.removeEventListener('click', handleClickAway, true)
  }, [openActionsFor])

  const handleRowClick = (bucketId: string) => {
    setOpenActionsFor(null)
    navigate(`/buckets/${bucketId}`)
  }

  const handleDeleteBucket = (bucketId: string, bucketName: string) => {
    setConfirmDialog({ isOpen: true, bucketId, bucketName, deleteRemote: false })
  }

  const confirmDeleteBucket = async (bucketId: string | null, deleteRemote: boolean) => {
    if (!bucketId) return
    try {
      await deleteBucketMutation.mutateAsync({ bucketId, deleteRemote })
      setAlertDialog({
        isOpen: true,
        title: 'Bucket deleted',
        message: deleteRemote
          ? 'The bucket and its data have been removed from your connected storage.'
          : 'The bucket entry has been removed from your workspace.',
        variant: 'success',
      })
    } catch (err) {
      setAlertDialog({
        isOpen: true,
        title: 'Delete failed',
        message: (err as Error).message,
        variant: 'error',
      })
    } finally {
      setConfirmDialog({ isOpen: false, bucketId: null, bucketName: '', deleteRemote: false })
      setOpenActionsFor(null)
    }
  }

  const handleRecalculateSize = async (bucketId: string) => {
    try {
      await recalculateSizeMutation.mutateAsync(bucketId)
      setAlertDialog({
        isOpen: true,
        title: 'Size recalculated',
        message: 'The bucket size has been updated successfully.',
        variant: 'success',
      })
    } catch (err) {
      setAlertDialog({
        isOpen: true,
        title: 'Recalculation failed',
        message: (err as Error).message,
        variant: 'error',
      })
    } finally {
      setOpenActionsFor(null)
    }
  }

  function renderActionsMenu(bucket: BucketEntity) {
    return (
      <div
        role="menu"
        className="absolute right-0 top-full z-20 mt-2 w-48 rounded-lg border border-slate-200 bg-white py-1 shadow-lg dark:border-slate-700 dark:bg-slate-900"
        onClick={(event) => event.stopPropagation()}
      >
        <button
          type="button"
          className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm text-slate-700 transition-colors hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-800"
          onClick={() => {
            setOpenActionsFor(null)
            handleRowClick(bucket.id)
          }}
        >
          <span className="material-symbols-outlined text-base">folder_open</span>
          View bucket
        </button>
        <button
          type="button"
          className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm text-slate-700 transition-colors hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-50 dark:text-slate-300 dark:hover:bg-slate-800"
          disabled={recalculateSizeMutation.isPending && openActionsFor === bucket.id}
          onClick={() => handleRecalculateSize(bucket.id)}
        >
          <span className="material-symbols-outlined text-base">refresh</span>
          {recalculateSizeMutation.isPending && openActionsFor === bucket.id ? 'Calculating...' : 'Recalculate size'}
        </button>
        <div className="my-1 border-t border-slate-200 dark:border-slate-700" />
        <button
          type="button"
          className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm text-red-500 transition-colors hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-500/10"
          disabled={deleteBucketMutation.isPending && openActionsFor === bucket.id}
          onClick={() => handleDeleteBucket(bucket.id, bucket.name)}
        >
          <span className="material-symbols-outlined text-base">delete</span>
          Delete bucket
        </button>
      </div>
    )
  }

  const renderListView = () => (
    <div className="overflow-x-auto pb-32">
      <div className="inline-block min-w-full align-middle">
        <div className="relative overflow-visible rounded-lg border border-slate-200 dark:border-slate-800">
          <table className="min-w-full divide-y divide-slate-200 dark:divide-slate-800">
            <thead className="bg-slate-50 dark:bg-slate-800/50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-slate-400">
                  Bucket Name
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-slate-400">
                  Region
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-slate-400">
                  Credential
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-slate-400">
                  Creation Date
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-slate-400">
                  Size
                </th>
                <th scope="col" className="relative px-6 py-3">
                  <span className="sr-only">Actions</span>
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-200 bg-white dark:divide-slate-800 dark:bg-background-dark">
              {filteredBuckets.map((bucket) => {
                const isMenuOpen = openActionsFor === bucket.id
                return (
                  <tr
                    key={bucket.id}
                    className="cursor-pointer hover:bg-slate-50 dark:hover:bg-slate-800/50"
                    onClick={() => handleRowClick(bucket.id)}
                  >
                    <td className="whitespace-nowrap px-6 py-4 text-sm font-medium text-slate-900 dark:text-white">
                      {bucket.name}
                    </td>
                    <td className="whitespace-nowrap px-6 py-4 text-sm text-slate-500 dark:text-slate-400">{bucket.region}</td>
                    <td className="whitespace-nowrap px-6 py-4 text-sm text-slate-500 dark:text-slate-400">
                      <div className="flex flex-col">
                        <span className="font-medium text-slate-700 dark:text-slate-200">{bucket.credentialName}</span>
                        <span className="text-xs text-slate-500 dark:text-slate-400">{bucket.credentialProvider}</span>
                      </div>
                    </td>
                    <td className="whitespace-nowrap px-6 py-4 text-sm text-slate-500 dark:text-slate-400">
                      {new Date(bucket.createdAt).toLocaleString()}
                    </td>
                    <td className="whitespace-nowrap px-6 py-4 text-sm text-slate-500 dark:text-slate-400">{bucket.size}</td>
                    <td className="relative whitespace-nowrap px-6 py-4 text-right text-sm font-medium" data-bucket-actions={bucket.id}>
                      <button
                        type="button"
                        onClick={(event) => {
                          event.stopPropagation()
                          const nativeEvent = event.nativeEvent as MouseEvent
                          if (nativeEvent.stopImmediatePropagation) {
                            nativeEvent.stopImmediatePropagation()
                          }
                          setOpenActionsFor((prev) => (prev === bucket.id ? null : bucket.id))
                        }}
                        className={`rounded-full p-1 text-slate-500 transition-colors hover:bg-slate-100 hover:text-primary dark:text-slate-400 dark:hover:bg-slate-800 ${
                          isMenuOpen ? 'bg-slate-100 text-primary dark:bg-slate-800/60' : ''
                        }`}
                        aria-haspopup="menu"
                        aria-expanded={isMenuOpen}
                      >
                        <span className="material-symbols-outlined">more_horiz</span>
                        <span className="sr-only">Bucket actions</span>
                      </button>
                      {isMenuOpen && renderActionsMenu(bucket)}
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )

  const renderGridView = () => (
    <div className="grid gap-4 pb-32 sm:grid-cols-2 xl:grid-cols-3">
      {filteredBuckets.map((bucket) => {
        const isMenuOpen = openActionsFor === bucket.id
        return (
          <article
            key={bucket.id}
            onClick={() => handleRowClick(bucket.id)}
            className="group relative flex cursor-pointer flex-col rounded-xl border border-slate-200 bg-white p-5 shadow-sm transition hover:border-primary/40 hover:shadow-md dark:border-slate-700 dark:bg-surface-dark"
          >
            <header className="flex items-start justify-between gap-3">
              <div>
                <h3 className="text-base font-semibold text-slate-900 dark:text-white">{bucket.name}</h3>
                <p className="text-xs text-slate-500 dark:text-slate-400">
                  Created {new Date(bucket.createdAt).toLocaleDateString()}
                </p>
              </div>
              <div className="relative" data-bucket-actions={bucket.id}>
                <button
                  type="button"
                  onClick={(event) => {
                    event.stopPropagation()
                    const nativeEvent = event.nativeEvent as MouseEvent
                    if (nativeEvent.stopImmediatePropagation) {
                      nativeEvent.stopImmediatePropagation()
                    }
                    setOpenActionsFor((prev) => (prev === bucket.id ? null : bucket.id))
                  }}
                  className={`rounded-full p-1.5 text-slate-500 transition-colors hover:bg-slate-100 hover:text-primary dark:text-slate-300 dark:hover:bg-slate-800 ${
                    isMenuOpen ? 'bg-slate-100 text-primary dark:bg-slate-800/60' : ''
                  }`}
                  aria-haspopup="menu"
                  aria-expanded={isMenuOpen}
                >
                  <span className="material-symbols-outlined text-base">more_horiz</span>
                </button>
                {isMenuOpen && renderActionsMenu(bucket)}
              </div>
            </header>

            <dl className="mt-4 grid grid-cols-2 gap-3 text-sm text-slate-600 dark:text-slate-300">
              <div>
                <dt className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-400">Region</dt>
                <dd className="mt-1 font-medium text-slate-800 dark:text-white">{bucket.region}</dd>
              </div>
              <div>
                <dt className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-400">Credential</dt>
                <dd className="mt-1 font-medium text-slate-800 dark:text-white">{bucket.credentialName}</dd>
                <span className="text-xs text-slate-500 dark:text-slate-400">{bucket.credentialProvider}</span>
              </div>
              <div>
                <dt className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-400">Created</dt>
                <dd className="mt-1 text-sm text-slate-600 dark:text-slate-300">
                  {new Date(bucket.createdAt).toLocaleString()}
                </dd>
              </div>
              <div>
                <dt className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-400">Size</dt>
                <dd className="mt-1 text-sm text-slate-600 dark:text-slate-300">{bucket.size}</dd>
              </div>
            </dl>
          </article>
        )
      })}
    </div>
  )

  const statusContent = () => {
    if (isLoading) {
      return (
        <div className="rounded-lg border border-slate-200 bg-white p-6 text-center text-sm text-slate-500 dark:border-slate-800 dark:bg-surface-dark dark:text-slate-300">
          Loading buckets…
        </div>
      )
    }

    if (isError) {
      return (
        <div className="rounded-lg border border-red-200 bg-red-50 p-6 text-center text-sm text-red-600 dark:border-red-500/50 dark:bg-red-500/10 dark:text-red-200">
          {(error as Error).message}
          <button
            className="ml-3 text-sm font-semibold text-primary hover:underline"
            type="button"
            onClick={() => refetch()}
            disabled={isFetching}
          >
            Retry
          </button>
        </div>
      )
    }

    if (filteredBuckets.length === 0 && searchQuery) {
      return (
        <div className="rounded-lg border border-dashed border-slate-300 bg-white p-10 text-center text-sm text-slate-500 dark:border-slate-700 dark:bg-surface-dark dark:text-slate-300">
          No buckets found matching "{searchQuery}"
        </div>
      )
    }

    if (buckets.length === 0) {
      return (
        <div className="rounded-lg border border-dashed border-slate-300 bg-white p-10 text-center text-sm text-slate-500 dark:border-slate-700 dark:bg-surface-dark dark:text-slate-300">
          No buckets yet. Connect your first storage account to get started.
          <div className="mt-4 flex justify-center">
            <Button variant="primary" onClick={openCreateModal}>
              Add your first bucket
            </Button>
          </div>
        </div>
      )
    }

    return viewMode === 'grid' ? renderGridView() : renderListView()
  }

  return (
    <AppShell searchPlaceholder="Search buckets..." searchValue={searchQuery} onSearchChange={setSearchQuery}>
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="flex flex-col gap-1">
          <h1 className="text-3xl font-black leading-tight tracking-tight text-slate-900 dark:text-white">
            My Buckets
          </h1>
          <p className="text-base text-slate-600 dark:text-slate-400">
            Organize and explore your object storage with BucketBird.
          </p>
        </div>
        <Button className="min-w-[120px]" variant="primary" onClick={openCreateModal}>
          Add Bucket
        </Button>
      </div>

      <div className="mt-6 flex flex-wrap items-center justify-end gap-2 border-b border-slate-200 pb-2 dark:border-slate-800">
        <div className="flex gap-1">
          {VIEW_MODES.map((mode) => {
            const isActive = viewMode === mode.key
            return (
              <button
                key={mode.key}
                type="button"
                className={`flex h-9 w-9 items-center justify-center rounded-lg transition-colors ${
                  isActive
                    ? 'bg-primary/15 text-primary-strong dark:bg-white/10 dark:text-white'
                    : 'text-slate-500 hover:bg-background-muted dark:text-slate-400 dark:hover:bg-slate-800'
                }`}
                aria-label={mode.label}
                aria-pressed={isActive}
                onClick={() => setViewMode(mode.key)}
              >
                <span className="material-symbols-outlined text-xl">{mode.icon}</span>
              </button>
            )
          })}
        </div>
      </div>

      <div className="mt-4 flex-1">{statusContent()}</div>

      <ConfirmDialog
        isOpen={confirmDialog.isOpen}
        onClose={() => setConfirmDialog({ isOpen: false, bucketId: null, bucketName: '', deleteRemote: false })}
        onConfirm={() => confirmDeleteBucket(confirmDialog.bucketId, confirmDialog.deleteRemote)}
        title="Delete bucket"
        message={
          <div className="space-y-4 text-left">
            <p>
              Are you sure you want to delete <strong>{confirmDialog.bucketName}</strong>? This action cannot be
              undone.
            </p>
            <label className="flex items-start gap-2 text-sm text-slate-600 dark:text-slate-300">
              <input
                type="checkbox"
                className="mt-1 h-4 w-4 rounded border-slate-300 text-primary focus:ring-primary"
                checked={confirmDialog.deleteRemote}
                onChange={(event) =>
                  setConfirmDialog((prev) => ({ ...prev, deleteRemote: event.target.checked }))
                }
              />
              <span>
                Also delete the bucket and its objects from the connected storage provider.
                <span className="mt-1 block text-xs text-slate-500 dark:text-slate-400">
                  This may take a few moments for large buckets.
                </span>
              </span>
            </label>
          </div>
        }
        confirmText={deleteBucketMutation.isPending ? 'Deleting...' : 'Delete bucket'}
        isLoading={deleteBucketMutation.isPending}
      />

      <AlertDialog
        isOpen={alertDialog.isOpen}
        onClose={() => setAlertDialog({ isOpen: false, title: '', message: '' })}
        title={alertDialog.title}
        message={alertDialog.message}
        variant={alertDialog.variant}
      />
    </AppShell>
  )
}

export default DashboardPage
