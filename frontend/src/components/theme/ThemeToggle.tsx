import { memo } from 'react'

import { useTheme } from '../../hooks/useTheme'

export const ThemeToggle = memo(() => {
  const { theme, toggleTheme } = useTheme()
  const isDark = theme === 'dark'

  return (
    <button
      type="button"
      onClick={toggleTheme}
      className="flex h-10 w-10 items-center justify-center rounded-full text-slate-500 transition-colors hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-800"
      aria-label="Toggle theme"
    >
      <span className="material-symbols-outlined">{isDark ? 'dark_mode' : 'light_mode'}</span>
    </button>
  )
})

ThemeToggle.displayName = 'ThemeToggle'

export default ThemeToggle
