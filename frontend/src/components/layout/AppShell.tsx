import type { ReactNode } from 'react'

import { cn } from '../../lib/cn'
import { useAuth } from '../../contexts/useAuth'
import AppSidebar from './AppSidebar'
import TopBar from './TopBar'
import CreateBucketModal from '../buckets/CreateBucketModal'

type AppShellProps = {
  children: ReactNode
  searchPlaceholder?: string
  searchValue?: string
  onSearchChange?: (value: string) => void
  topRight?: ReactNode
  contentClassName?: string
}

export const AppShell = ({
  children,
  searchPlaceholder,
  searchValue,
  onSearchChange,
  topRight,
  contentClassName,
}: AppShellProps) => {
  const { user } = useAuth()
  const isDemo = user?.isReadonly || false

  return (
    <>
      <div className="flex min-h-screen w-full bg-gradient-to-br from-background-light via-white to-background-muted text-slate-900 dark:bg-background-dark dark:text-white">
        <AppSidebar />
        <div className="flex flex-1 flex-col overflow-hidden">
          <TopBar searchPlaceholder={searchPlaceholder} searchValue={searchValue} onSearchChange={onSearchChange} rightSection={topRight} />
          {isDemo && (
            <div className="flex items-center justify-center gap-3 bg-gradient-to-r from-amber-500 to-orange-500 px-4 py-3 text-white shadow-md">
              <span className="material-symbols-outlined text-xl">info</span>
              <p className="text-sm font-medium">
                You're viewing BucketBird in demo mode. All write operations are disabled.
              </p>
            </div>
          )}
          <main
            className={cn(
              'flex flex-1 flex-col bg-white/90 px-4 py-6 text-slate-800 shadow-inner dark:bg-surface-dark dark:text-slate-100 md:px-8',
              contentClassName,
            )}
          >
            {children}
          </main>
        </div>
      </div>
      <CreateBucketModal />
    </>
  )
}

export default AppShell
