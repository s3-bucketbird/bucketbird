import { useEffect, useState, type FormEvent } from 'react'

import Button from '../ui/Button'
import { useCredentials } from '../../hooks/useCredentials'
import { useCredentialBuckets } from '../../hooks/useCredentialBuckets'
import { useCreateBucket } from '../../hooks/useBucketMutations'
import { useBucketModal } from '../../hooks/useBucketModal'

export const CreateBucketModal = () => {
  const { showCreateModal, closeCreateModal } = useBucketModal()
  const {
    data: credentials = [],
    isLoading: credentialsLoading,
    isError: credentialsError,
  } = useCredentials()
  const createBucketMutation = useCreateBucket()
  const [formError, setFormError] = useState<string | null>(null)
  const [formState, setFormState] = useState({
    name: '',
    region: 'us-east-1',
    credentialId: '',
    description: '',
  })
  const {
    data: discoveredBuckets = [],
    isLoading: discoveryLoading,
    isError: discoveryError,
  } = useCredentialBuckets(formState.credentialId || undefined)

  const handleCloseModal = () => {
    if (createBucketMutation.isPending) {
      return
    }
    closeCreateModal()
  }

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setFormError(null)

    if (!formState.name.trim()) {
      setFormError('Bucket name is required.')
      return
    }
    if (!formState.credentialId) {
      setFormError('Select a credential to connect this bucket.')
      return
    }

    try {
      await createBucketMutation.mutateAsync({
        name: formState.name.trim(),
        region: formState.region || 'us-east-1',
        credentialId: formState.credentialId,
        description: formState.description.trim() ? formState.description.trim() : undefined,
      })
      closeCreateModal()
      setFormState({
        name: '',
        region: 'us-east-1',
        credentialId: '',
        description: '',
      })
    } catch (err) {
      setFormError((err as Error).message)
    }
  }

  useEffect(() => {
    if (showCreateModal && !formState.credentialId && credentials.length > 0) {
      setFormState((prev) => ({ ...prev, credentialId: credentials[0].id }))
    }
  }, [showCreateModal, formState.credentialId, credentials])

  useEffect(() => {
    if (showCreateModal) {
      setFormError(null)
      setFormState((prev) => ({
        name: '',
        region: 'us-east-1',
        credentialId: credentials[0]?.id ?? prev.credentialId ?? '',
        description: '',
      }))
    }
  }, [showCreateModal, credentials])

  const credentialsUnavailable = !credentialsLoading && !credentialsError && credentials.length === 0
  const isSubmitDisabled = createBucketMutation.isPending || credentialsUnavailable

  if (!showCreateModal) {
    return null
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div className="w-full max-w-xl rounded-xl border border-slate-200 bg-white p-6 shadow-2xl dark:border-slate-800 dark:bg-slate-900">
        <header className="mb-4 flex items-start justify-between">
          <div>
            <h2 className="text-xl font-semibold text-slate-900 dark:text-white">Create bucket</h2>
            <p className="text-sm text-slate-500 dark:text-slate-400">
              Buckets are created with the selected credential and region.
            </p>
          </div>
          <button
            type="button"
            onClick={handleCloseModal}
            className="rounded-md p-1 text-slate-500 transition-colors hover:bg-slate-100 hover:text-slate-700 dark:text-slate-400 dark:hover:bg-slate-800"
            aria-label="Close"
          >
            <span className="material-symbols-outlined">close</span>
          </button>
        </header>

        <form onSubmit={handleSubmit} className="flex flex-col gap-5">
          {formError && (
            <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-600 dark:border-red-500/50 dark:bg-red-500/10 dark:text-red-200">
              {formError}
            </div>
          )}

          <label className="flex flex-col gap-2">
            <span className="text-sm font-medium text-slate-800 dark:text-slate-200">
              Bucket name <span className="text-red-500">*</span>
            </span>
            <input
              type="text"
              value={formState.name}
              onChange={(event) => setFormState((prev) => ({ ...prev, name: event.target.value }))}
              placeholder="e.g., media-assets-prod"
              required
              className="form-input rounded-lg border border-slate-300 bg-background-light px-3.5 py-2.5 text-slate-900 focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20 dark:border-slate-700 dark:bg-background-dark dark:text-white"
            />
            {formState.credentialId && (
              <div className="mt-2 rounded-lg border border-slate-200 bg-slate-50 p-3 text-xs text-slate-600 dark:border-slate-700 dark:bg-slate-800/60 dark:text-slate-300">
                {discoveryLoading && <p>Scanning existing buckets for this credential…</p>}
                {discoveryError && (
                  <p>
                    Unable to discover buckets right now. You can still enter a bucket name manually.
                  </p>
                )}
                {!discoveryLoading && !discoveryError && (
                  <>
                    {discoveredBuckets.length > 0 ? (
                      <>
                        <p className="mb-2 font-medium text-slate-700 dark:text-slate-200">
                          Choose from discovered buckets or type a new name:
                        </p>
                        <div className="flex flex-wrap gap-2">
                          {discoveredBuckets.slice(0, 8).map((bucket) => {
                            const isSelected = formState.name.trim() === bucket.name
                            return (
                              <button
                                type="button"
                                key={bucket.name}
                                onClick={() =>
                                  setFormState((prev) => ({
                                    ...prev,
                                    name: bucket.name,
                                  }))
                                }
                                className={[
                                  'rounded-full border px-3 py-1 text-xs font-medium transition',
                                  isSelected
                                    ? 'border-primary bg-primary/10 text-primary dark:border-primary/70 dark:bg-primary/20'
                                    : 'border-slate-300 text-slate-600 hover:border-primary hover:bg-primary/10 hover:text-primary dark:border-slate-600 dark:text-slate-200',
                                ].join(' ')}
                              >
                                {bucket.name}
                              </button>
                            )
                          })}
                        </div>
                      </>
                    ) : (
                      <p>No existing buckets were found for this credential. Enter a name to create one.</p>
                    )}
                  </>
                )}
              </div>
            )}
          </label>

          <div className="grid grid-cols-1 gap-5 md:grid-cols-2">
            <label className="flex flex-col gap-2">
              <span className="text-sm font-medium text-slate-800 dark:text-slate-200">
                Region <span className="text-red-500">*</span>
              </span>
              <input
                type="text"
                value={formState.region}
                onChange={(event) => setFormState((prev) => ({ ...prev, region: event.target.value }))}
                placeholder="us-east-1"
                className="form-input rounded-lg border border-slate-300 bg-background-light px-3.5 py-2.5 text-slate-900 focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20 dark:border-slate-700 dark:bg-background-dark dark:text-white"
              />
            </label>

            <label className="flex flex-col gap-2">
              <span className="text-sm font-medium text-slate-800 dark:text-slate-200">
                Credential <span className="text-red-500">*</span>
              </span>
              <select
                value={formState.credentialId}
                onChange={(event) => setFormState((prev) => ({ ...prev, credentialId: event.target.value }))}
                className="form-select rounded-lg border border-slate-300 bg-background-light px-3.5 py-2.5 text-slate-900 focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20 dark:border-slate-700 dark:bg-background-dark dark:text-white"
                disabled={credentialsLoading}
              >
                <option value="" disabled>
                  {credentialsLoading ? 'Loading credentials…' : 'Select credential'}
                </option>
                {credentials.map((credential) => (
                  <option key={credential.id} value={credential.id}>
                    {credential.name} · {credential.provider}
                  </option>
                ))}
              </select>
              <div className="mt-2 flex flex-wrap items-center gap-2 text-xs">
                {credentialsUnavailable && (
                  <p className="text-xs text-slate-500 dark:text-slate-400">
                    Add a credential from Settings before creating a bucket.
                  </p>
                )}
                {credentialsError && (
                  <p className="text-xs text-red-500 dark:text-red-400">
                    Unable to load credentials. Please try again from Settings.
                  </p>
                )}
                <button
                  type="button"
                  onClick={() => window.open('/settings', '_blank', 'noopener,noreferrer')}
                  className="ml-auto inline-flex items-center gap-1 rounded-full border border-slate-200 px-2.5 py-1 text-xs font-medium text-primary transition hover:border-primary hover:bg-primary/10 dark:border-slate-700 dark:text-primary dark:hover:bg-primary/10"
                >
                  <span className="material-symbols-outlined text-sm">open_in_new</span>
                  Manage credentials
                </button>
              </div>
            </label>
          </div>

          <label className="flex flex-col gap-2">
            <span className="text-sm font-medium text-slate-800 dark:text-slate-200">Description</span>
            <textarea
              value={formState.description}
              onChange={(event) => setFormState((prev) => ({ ...prev, description: event.target.value }))}
              placeholder="Optional description for this bucket"
              rows={3}
              className="form-textarea rounded-lg border border-slate-300 bg-background-light px-3.5 py-2.5 text-sm text-slate-900 focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20 dark:border-slate-700 dark:bg-background-dark dark:text-white"
            />
          </label>

          <div className="flex justify-end gap-3">
            <Button
              type="button"
              variant="outline"
              onClick={handleCloseModal}
              disabled={createBucketMutation.isPending}
              className="border-slate-300 text-slate-700 dark:border-slate-700 dark:text-white"
            >
              Cancel
            </Button>
            <Button type="submit" disabled={isSubmitDisabled}>
              {createBucketMutation.isPending ? 'Creating…' : 'Create bucket'}
            </Button>
          </div>
        </form>
      </div>
    </div>
  )
}

export default CreateBucketModal
