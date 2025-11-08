import { useEffect, useMemo, useState } from 'react'
import { Link, NavLink, useLocation, useParams } from 'react-router-dom'

import { useBuckets } from '../../hooks/useBuckets'
import { useBucketModal } from '../../hooks/useBucketModal'

export const AppSidebar = () => {
  const location = useLocation()
  const { bucketId } = useParams<{ bucketId: string }>()
  const { data: buckets = [] } = useBuckets()
  const { openCreateModal } = useBucketModal()
  const [isMyBucketsExpanded, setIsMyBucketsExpanded] = useState(true)
  const [expandedCredentials, setExpandedCredentials] = useState<Set<string>>(new Set())

  // Check if we're on dashboard or any bucket page
  const isOnDashboardOrBucket = location.pathname === '/dashboard' || location.pathname.startsWith('/buckets/')

  // Group buckets by credential
  const bucketsByCredential = useMemo(() => {
    const groups = new Map<string, typeof buckets>()

    buckets.forEach((bucket) => {
      const key = bucket.credentialId
      if (!groups.has(key)) {
        groups.set(key, [])
      }
      groups.get(key)!.push(bucket)
    })

    return groups
  }, [buckets])

  // Auto-expand credential section containing active bucket
  useEffect(() => {
    if (bucketId && buckets.length > 0) {
      const activeBucket = buckets.find((b) => b.id === bucketId)
      if (activeBucket) {
        setExpandedCredentials((prev) => new Set(prev).add(activeBucket.credentialId))
      }
    }
  }, [bucketId, buckets])

  const toggleCredential = (credentialId: string) => {
    setExpandedCredentials((prev) => {
      const next = new Set(prev)
      if (next.has(credentialId)) {
        next.delete(credentialId)
      } else {
        next.add(credentialId)
      }
      return next
    })
  }

  return (
    <aside className="hidden w-64 flex-col justify-between border-r border-slate-200 bg-white p-4 text-slate-700 dark:border-slate-800 dark:bg-[#111a22] dark:text-slate-200 md:flex">
      <div className="flex flex-col gap-8">
        <Link to="/dashboard" className="flex items-center gap-3 px-2 transition-opacity hover:opacity-90">
          <img
            src="/bucketbird.png"
            alt="BucketBird logo"
            className="size-16 rounded-lg bg-white/80 object-contain p-1 shadow-sm dark:bg-white/10 dark:[filter:drop-shadow(0_0_1px_rgba(255,255,255,0.1))_drop-shadow(0_0_2px_rgba(255,255,255,0.1))]"
          />
          <div className="flex flex-col">
            <span className="text-lg font-semibold text-slate-900 dark:text-white">BucketBird</span>
            <span className="text-xs font-medium text-slate-500 dark:text-slate-400">Storage console</span>
          </div>
        </Link>
        <nav className="flex flex-col gap-2">
          {/* My Buckets - Expandable */}
          <div>
            <button
              onClick={() => setIsMyBucketsExpanded(!isMyBucketsExpanded)}
              className={[
                'flex w-full items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
                isOnDashboardOrBucket
                  ? 'bg-primary/10 text-primary-strong dark:bg-white/10 dark:text-white'
                  : 'text-slate-600 hover:bg-background-muted dark:text-slate-300 dark:hover:bg-slate-800',
              ].join(' ')}
            >
              <span className="material-symbols-outlined text-xl">inventory_2</span>
              <span className="flex-1 text-left">My Buckets</span>
              <span className="material-symbols-outlined text-lg">
                {isMyBucketsExpanded ? 'expand_more' : 'chevron_right'}
              </span>
            </button>

            {/* Bucket List - Grouped by Credential */}
            {isMyBucketsExpanded && (
              <div className="ml-3 mt-1 flex flex-col gap-1">
                {buckets.length === 0 ? (
                  <div className="px-3 py-2 text-xs text-slate-500 dark:text-slate-400">No buckets yet</div>
                ) : (
                  Array.from(bucketsByCredential.entries()).map(([credentialId, credentialBuckets]) => {
                    const firstBucket = credentialBuckets[0]
                    const isCredentialExpanded = expandedCredentials.has(credentialId)

                    return (
                      <div key={credentialId} className="flex flex-col">
                        {/* Credential Header */}
                        <button
                          onClick={() => toggleCredential(credentialId)}
                          className="flex w-full items-center gap-2 rounded-lg px-3 py-1.5 text-xs font-medium text-slate-700 transition-colors hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-800"
                        >
                          <span className="material-symbols-outlined text-sm">
                            {isCredentialExpanded ? 'expand_more' : 'chevron_right'}
                          </span>
                          <span className="flex-1 truncate text-left">
                            {firstBucket.credentialName}
                          </span>
                          <span className="text-[10px] text-slate-500 dark:text-slate-500">
                            {credentialBuckets.length}
                          </span>
                        </button>

                        {/* Buckets under this credential */}
                        {isCredentialExpanded && (
                          <div className="ml-3 mt-0.5 flex flex-col gap-0.5 border-l border-slate-200 pl-2 dark:border-slate-700">
                            {credentialBuckets.map((bucket) => (
                              <NavLink
                                key={bucket.id}
                                to={`/buckets/${bucket.id}`}
                                className={({ isActive }) =>
                                  [
                                    'flex items-center gap-2 rounded-lg px-3 py-1.5 text-xs transition-colors',
                                    isActive
              ? 'bg-primary/10 font-medium text-primary-strong dark:bg-white/10 dark:text-white'
              : 'text-slate-600 hover:bg-background-muted dark:text-slate-300 dark:hover:bg-slate-800',
                                  ].join(' ')
                                }
                              >
                                <span className="material-symbols-outlined text-sm">folder</span>
                                <span className="truncate">{bucket.name}</span>
                              </NavLink>
                            ))}
                          </div>
                        )}
                      </div>
                    )
                  })
                )}
              </div>
            )}
          </div>
        </nav>
      </div>
      <div className="flex flex-col gap-4">
        <button
          onClick={openCreateModal}
          className="flex h-10 w-full items-center justify-center rounded-lg bg-primary text-sm font-semibold text-white shadow-sm transition-colors hover:bg-primary-strong dark:text-slate-900 dark:hover:text-white"
        >
          <span className="truncate">Add Bucket</span>
        </button>
        <NavLink
          to="/settings"
          className={({ isActive }) =>
            [
              'flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium',
              isActive
                ? 'bg-primary/10 text-primary-strong dark:bg-white/10 dark:text-white'
                : 'text-slate-600 hover:bg-background-muted dark:text-slate-300 dark:hover:bg-slate-800',
            ].join(' ')
          }
        >
          <span className="material-symbols-outlined text-xl">settings</span>
          <span>Settings</span>
        </NavLink>
      </div>
    </aside>
  )
}

export default AppSidebar
