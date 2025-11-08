import AppShell from '../components/layout/AppShell'

const RecentPage = () => {
  return (
    <AppShell searchPlaceholder="Search recent files...">
      <div className="flex flex-1 flex-col items-center justify-center rounded-xl border border-dashed border-slate-300 bg-white py-20 text-center text-slate-500 dark:border-slate-700 dark:bg-slate-900/40 dark:text-slate-300">
        <span className="material-symbols-outlined mb-4 text-4xl text-primary">schedule</span>
        <h2 className="text-2xl font-semibold text-slate-800 dark:text-white">Recent activity</h2>
        <p className="mt-2 max-w-sm text-sm text-slate-500 dark:text-slate-400">
          Your most recently accessed files will surface here once activity tracking is wired up.
        </p>
      </div>
    </AppShell>
  )
}

export default RecentPage
