import type { ReactNode } from 'react'
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'

import ThemeToggle from '../theme/ThemeToggle'
import { useTheme } from '../../hooks/useTheme'
import { useAuth } from '../../contexts/useAuth'

type TopBarProps = {
  searchPlaceholder?: string
  searchValue?: string
  onSearchChange?: (value: string) => void
  rightSection?: ReactNode
}

const DefaultRightSection = () => {
  const [showDropdown, setShowDropdown] = useState(false)
  const { user, logout } = useAuth()
  const navigate = useNavigate()

  const handleLogout = async () => {
    await logout()
    navigate('/login')
  }

  return (
    <>
      <ThemeToggle />
      <button
        onClick={() => navigate('/settings')}
        className="flex h-10 w-10 items-center justify-center rounded-full text-slate-500 transition-colors hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-800"
      >
        <span className="material-symbols-outlined">settings</span>
      </button>

      <div className="relative">
        <button
          onClick={() => setShowDropdown(!showDropdown)}
          className="flex h-10 w-10 items-center justify-center rounded-full transition-all hover:ring-2 hover:ring-primary/30"
        >
          <div
            className="size-10 rounded-full bg-primary/20 bg-cover bg-center bg-no-repeat flex items-center justify-center text-primary font-semibold"
            aria-hidden="true"
          >
            {user?.firstName?.[0] || user?.email?.[0]?.toUpperCase() || 'U'}
          </div>
        </button>

        {showDropdown && (
          <>
            <div
              className="fixed inset-0 z-10"
              onClick={() => setShowDropdown(false)}
            />
            <div className="absolute right-0 top-full z-20 mt-2 w-56 rounded-lg border border-slate-200 bg-white py-2 shadow-lg dark:border-slate-700 dark:bg-slate-900">
              <div className="px-4 py-2 border-b border-slate-200 dark:border-slate-700">
                <p className="text-sm font-medium text-slate-900 dark:text-white">
                  {user?.firstName} {user?.lastName}
                </p>
                <p className="text-xs text-slate-500 dark:text-slate-400">{user?.email}</p>
              </div>
              <button
                onClick={() => {
                  setShowDropdown(false)
                  navigate('/settings')
                }}
                className="flex w-full items-center gap-3 px-4 py-2 text-left text-sm text-slate-700 hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-800"
              >
                <span className="material-symbols-outlined text-lg">settings</span>
                <span>Settings</span>
              </button>
              <button
                onClick={handleLogout}
                className="flex w-full items-center gap-3 px-4 py-2 text-left text-sm text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/20"
              >
                <span className="material-symbols-outlined text-lg">logout</span>
                <span>Logout</span>
              </button>
            </div>
          </>
        )}
      </div>
    </>
  )
}

const MobileActions = () => {
  const [isOpen, setIsOpen] = useState(false)
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const { theme, toggleTheme } = useTheme()

  const handleLogout = async () => {
    await logout()
    navigate('/login', { replace: true })
  }

  return (
    <div className="relative md:hidden" data-mobile-actions>
      <button
        type="button"
        onClick={() => setIsOpen((prev) => !prev)}
        className="flex h-10 w-10 items-center justify-center rounded-full text-slate-500 transition-colors hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-800"
        aria-label="Open menu"
      >
        <span className="material-symbols-outlined text-2xl">more_vert</span>
      </button>

      {isOpen && (
        <>
          <div className="fixed inset-0 z-10" onClick={() => setIsOpen(false)} />
          <div className="absolute right-0 top-full z-20 mt-2 w-56 rounded-lg border border-slate-200 bg-white py-2 shadow-lg dark:border-slate-700 dark:bg-slate-900">
            <div className="px-4 py-2 border-b border-slate-200 dark:border-slate-700">
              <p className="text-sm font-medium text-slate-900 dark:text-white">
                {user?.firstName} {user?.lastName}
              </p>
              <p className="text-xs text-slate-500 dark:text-slate-400">{user?.email}</p>
            </div>

            <button
              type="button"
              className="flex w-full items-center gap-3 px-4 py-2 text-left text-sm text-slate-700 transition-colors hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-800"
              onClick={() => {
                toggleTheme()
                setIsOpen(false)
              }}
            >
              <span className="material-symbols-outlined text-base">{theme === 'dark' ? 'light_mode' : 'dark_mode'}</span>
              <span>{theme === 'dark' ? 'Light mode' : 'Dark mode'}</span>
            </button>

            <button
              type="button"
              className="flex w-full items-center gap-3 px-4 py-2 text-left text-sm text-slate-700 transition-colors hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-800"
              onClick={() => {
                setIsOpen(false)
                navigate('/settings')
              }}
            >
              <span className="material-symbols-outlined text-base">settings</span>
              <span>Settings</span>
            </button>

            <button
              type="button"
              className="flex w-full items-center gap-3 px-4 py-2 text-left text-sm text-red-600 transition-colors hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/20"
              onClick={async () => {
                setIsOpen(false)
                await handleLogout()
              }}
            >
              <span className="material-symbols-outlined text-base">logout</span>
              <span>Logout</span>
            </button>
          </div>
        </>
      )}
    </div>
  )
}

export const TopBar = ({
  searchPlaceholder = 'Search…',
  searchValue = '',
  onSearchChange,
  rightSection,
}: TopBarProps) => {
  const navigate = useNavigate()
  const renderedRightSection = rightSection ?? <DefaultRightSection />

  return (
    <header className="flex h-16 flex-shrink-0 items-center justify-between border-b border-slate-200 bg-white px-4 text-slate-700 shadow-sm dark:border-slate-800 dark:bg-[#111a22] dark:text-slate-200 md:px-8">
      {/* Mobile Logo */}
      <button
        onClick={() => navigate('/dashboard')}
        className="flex items-center md:hidden"
        aria-label="Go to dashboard"
      >
        <img
          src="/bucketbird.png"
          alt="BucketBird"
          className="size-10 rounded-lg bg-white/80 object-contain p-1 shadow-sm dark:bg-white/10"
        />
      </button>

      <div className="flex flex-1 md:max-w-lg">
        <label className="flex w-full flex-col">
          <div className="flex h-10 w-full items-stretch rounded-lg bg-transparent">
            <div className="flex items-center justify-center rounded-l-lg pl-3 text-slate-500 dark:text-slate-400">
              <span className="material-symbols-outlined text-xl">search</span>
            </div>
            <input
              className="form-input flex w-full flex-1 resize-none overflow-hidden rounded-lg border-none bg-transparent p-2 text-base leading-normal text-slate-900 placeholder:text-slate-500 focus:outline-none focus:ring-0 dark:text-white dark:placeholder:text-slate-400"
              placeholder={searchPlaceholder}
              value={searchValue}
              onChange={(e) => onSearchChange?.(e.target.value)}
            />
          </div>
        </label>
      </div>
      <div className="ml-2 hidden items-center gap-4 md:flex">{renderedRightSection}</div>
      <MobileActions />
    </header>
  )
}

export default TopBar
