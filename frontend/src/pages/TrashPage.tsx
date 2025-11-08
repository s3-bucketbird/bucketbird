import AppShell from '../components/layout/AppShell'

const TrashPage = () => {
  return (
    <AppShell searchPlaceholder="Search trash...">
      <div className="flex flex-1 flex-col items-center justify-center rounded-xl border border-dashed border-slate-300 bg-white py-20 text-center text-slate-500 dark:border-slate-700 dark:bg-slate-900/40 dark:text-slate-300">
        <span className="material-symbols-outlined mb-4 text-4xl text-primary">delete</span>
        <h2 className="text-2xl font-semibold text-slate-800 dark:text-white">Trash</h2>
        <p className="mt-2 max-w-sm text-sm text-slate-500 dark:text-slate-400">
          Deleted items will live here until they are purged according to retention policies.
        </p>
      </div>
    </AppShell>
  )
}

export default TrashPage
