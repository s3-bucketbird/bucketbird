import { useEffect, useState } from 'react'
import type { ChangeEvent, FormEvent } from 'react'
import { Link } from 'react-router-dom'

import Button from '../components/ui/Button'
import ConfirmDialog from '../components/ui/ConfirmDialog'
import CredentialCard from '../components/credentials/CredentialCard'
import CredentialForm from '../components/credentials/CredentialForm'
import { useCredentials } from '../hooks/useCredentials'
import { useProfile, useUpdatePassword, useUpdateProfile } from '../hooks/useProfile'
import {
  useCreateCredential,
  useUpdateCredential,
  useDeleteCredential,
  useTestCredential,
} from '../hooks/useCredentialMutations'
import { useAuth } from '../contexts/useAuth'
import { useTheme } from '../hooks/useTheme'

type FormMode = 'create' | 'edit' | null
type SelectedCredential = {
  id: string
  name: string
  provider: string
  region: string
  endpoint: string
  useSSL: boolean
} | null

type ProfileFormState = {
  firstName: string
  lastName: string
  email: string
}

const SettingsPage = () => {
  const { updateUser } = useAuth()
  const { theme, setTheme } = useTheme()
  const [formMode, setFormMode] = useState<FormMode>(null)
  const [selectedCredential, setSelectedCredential] = useState<SelectedCredential>(null)
  const [testResults, setTestResults] = useState<Record<string, { success: boolean; message: string }>>({})
  const [credentialToDelete, setCredentialToDelete] = useState<{ id: string; name: string } | null>(null)
  const [profileForm, setProfileForm] = useState<ProfileFormState>({
    firstName: '',
    lastName: '',
    email: '',
  })
  const [profileSuccess, setProfileSuccess] = useState<string | null>(null)
  const [profileSaveError, setProfileSaveError] = useState<string | null>(null)
  const [passwordForm, setPasswordForm] = useState({
    currentPassword: '',
    newPassword: '',
    confirmPassword: '',
  })
  const [passwordError, setPasswordError] = useState<string | null>(null)
  const [passwordSuccess, setPasswordSuccess] = useState<string | null>(null)

  const {
    data: profile,
    isLoading: profileLoading,
    isError: profileError,
    error: profileErrorDetail,
  } = useProfile()

  const {
    data: credentials = [],
    isLoading: credentialsLoading,
    isError: credentialsError,
    error: credentialsErrorDetail,
  } = useCredentials()

  const updateProfileMutation = useUpdateProfile()
  const updatePasswordMutation = useUpdatePassword()

  useEffect(() => {
    if (profile) {
      setProfileForm({
        firstName: profile.firstName ?? '',
        lastName: profile.lastName ?? '',
        email: profile.email ?? '',
      })
      setProfileSuccess(null)
      setProfileSaveError(null)
    }
  }, [profile])

  const createMutation = useCreateCredential()
  const updateMutation = useUpdateCredential()
  const deleteMutation = useDeleteCredential()
  const testMutation = useTestCredential()

  const handleProfileInputChange = (field: keyof ProfileFormState) => (
    event: ChangeEvent<HTMLInputElement>,
  ) => {
    setProfileForm((prev) => ({
      ...prev,
      [field]: event.target.value,
    }))
    setProfileSaveError(null)
    setProfileSuccess(null)
  }

  const handleProfileCancel = () => {
    if (profile) {
      setProfileForm({
        firstName: profile.firstName ?? '',
        lastName: profile.lastName ?? '',
        email: profile.email ?? '',
      })
    } else {
      setProfileForm({
        firstName: '',
        lastName: '',
        email: '',
      })
    }
    updateProfileMutation.reset()
    setProfileSaveError(null)
    setProfileSuccess(null)
  }

  const handleProfileSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setProfileSaveError(null)
    setProfileSuccess(null)

    try {
      const updatedProfile = await updateProfileMutation.mutateAsync({
        firstName: profileForm.firstName.trim(),
        lastName: profileForm.lastName.trim(),
        email: profileForm.email.trim(),
      })

      setProfileSuccess('Profile updated successfully.')
      updateUser({
        firstName: updatedProfile.firstName,
        lastName: updatedProfile.lastName,
        email: updatedProfile.email,
      })

      setTimeout(() => setProfileSuccess(null), 4000)
    } catch (error) {
      setProfileSaveError((error as Error).message)
    }
  }

  const handlePasswordInputChange = (field: 'currentPassword' | 'newPassword' | 'confirmPassword') =>
    (event: ChangeEvent<HTMLInputElement>) => {
      setPasswordForm((prev) => ({ ...prev, [field]: event.target.value }))
      setPasswordError(null)
      setPasswordSuccess(null)
    }

  const handlePasswordSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setPasswordError(null)
    setPasswordSuccess(null)

    if (!passwordForm.currentPassword.trim()) {
      setPasswordError('Current password is required.')
      return
    }

    if (passwordForm.newPassword !== passwordForm.confirmPassword) {
      setPasswordError('New password and confirmation do not match.')
      return
    }

    try {
      await updatePasswordMutation.mutateAsync({
        currentPassword: passwordForm.currentPassword,
        newPassword: passwordForm.newPassword,
      })

      setPasswordSuccess('Password updated successfully.')
      setPasswordForm({ currentPassword: '', newPassword: '', confirmPassword: '' })
      setTimeout(() => setPasswordSuccess(null), 4000)
    } catch (error) {
      setPasswordError((error as Error).message)
    }
  }

  const handleCreateClick = () => {
    setFormMode('create')
    setSelectedCredential(null)
  }

  const handleEditClick = (credential: typeof credentials[0]) => {
    setFormMode('edit')
    setSelectedCredential({
      id: credential.id,
      name: credential.name,
      provider: credential.provider,
      region: credential.region,
      endpoint: credential.endpoint,
      useSSL: credential.useSSL,
    })
  }

  const handleFormSubmit = async (data: {
    name: string
    provider: string
    region: string
    endpoint: string
    accessKey: string
    secretKey: string
    useSSL: boolean
  }) => {
    try {
      if (formMode === 'create') {
        await createMutation.mutateAsync(data)
      } else if (formMode === 'edit' && selectedCredential) {
        await updateMutation.mutateAsync({
          id: selectedCredential.id,
          input: data,
        })
      }
      setFormMode(null)
      setSelectedCredential(null)
    } catch (err) {
      // Error is handled by mutation error state
      console.error('Failed to save credential:', err)
    }
  }

  const handleDeleteClick = (id: string, name: string) => {
    setCredentialToDelete({ id, name })
  }

  const handleConfirmDelete = async () => {
    if (!credentialToDelete) return

    try {
      await deleteMutation.mutateAsync(credentialToDelete.id)
      // Clear test result for deleted credential
      setTestResults((prev) => {
        const newResults = { ...prev }
        delete newResults[credentialToDelete.id]
        return newResults
      })
      setCredentialToDelete(null)
    } catch (err) {
      console.error('Failed to delete credential:', err)
      // Keep dialog open on error so user can retry
    }
  }

  const handleTestClick = async (id: string) => {
    try {
      const result = await testMutation.mutateAsync(id)
      setTestResults((prev) => ({
        ...prev,
        [id]: result,
      }))
      // Clear test result after 5 seconds
      setTimeout(() => {
        setTestResults((prev) => {
          const newResults = { ...prev }
          delete newResults[id]
          return newResults
        })
      }, 5000)
    } catch (err) {
      console.error('Failed to test credential:', err)
    }
  }

  const formError =
    createMutation.error?.message || updateMutation.error?.message || null

  const themeOptions = [
    { value: 'light' as const, label: 'Light', icon: 'light_mode' },
    { value: 'dark' as const, label: 'Dark', icon: 'dark_mode' },
  ]

  return (
    <div className="flex min-h-screen bg-background-light text-slate-800 dark:bg-background-dark dark:text-slate-100">
      <aside className="hidden w-64 flex-col border-r border-slate-200 bg-white p-4 dark:border-slate-800 dark:bg-surface-dark md:flex">
        <div className="flex items-center justify-between">
          <h2 className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
            Navigation
          </h2>
          <span className="material-symbols-outlined text-base text-slate-400">settings</span>
        </div>
        <nav className="mt-4 flex flex-col gap-1">
          <Link
            to="/dashboard"
            className="flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium text-slate-600 transition-colors hover:bg-background-muted dark:text-slate-300 dark:hover:bg-surface-dark/70"
          >
            <span className="material-symbols-outlined text-xl">arrow_back</span>
            Back to dashboard
          </Link>
          <button
            type="button"
            className="flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium text-primary-strong/90 dark:text-primary"
            disabled
          >
            <span className="material-symbols-outlined text-xl">account_circle</span>
            Profile settings
          </button>
        </nav>
      </aside>

      <main className="flex-1 px-4 py-6 sm:px-8 lg:px-12 lg:py-10">
        {/* Mobile Back Button */}
        <Link
          to="/dashboard"
          className="mb-4 flex items-center gap-2 text-sm font-medium text-slate-600 transition-colors hover:text-primary dark:text-slate-300 dark:hover:text-primary md:hidden"
        >
          <span className="material-symbols-outlined text-xl">arrow_back</span>
          <span>Back to dashboard</span>
        </Link>
        <div className="mx-auto flex w-full max-w-4xl flex-col gap-10">
          {(profileLoading || credentialsLoading) && (
            <div className="rounded-lg border border-slate-200 bg-white p-6 text-sm text-slate-500 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-300">
              Loading settings…
            </div>
          )}
          {(profileError || credentialsError) && (
            <div className="rounded-lg border border-red-200 bg-red-50 p-6 text-sm text-red-600 dark:border-red-500/50 dark:bg-red-500/10 dark:text-red-200">
              {(profileErrorDetail as Error | undefined)?.message ||
                (credentialsErrorDetail as Error | undefined)?.message ||
                'Failed to load settings.'}
            </div>
          )}

          {deleteMutation.error && (
            <div className="rounded-lg border border-red-200 bg-red-50 p-6 text-sm text-red-600 dark:border-red-500/50 dark:bg-red-500/10 dark:text-red-200">
              Failed to delete credential: {deleteMutation.error.message}
            </div>
          )}

          <header className="flex flex-wrap items-center justify-between gap-4">
            <div>
              <h1 className="text-3xl font-bold text-slate-900 dark:text-white">Profile</h1>
              <p className="text-base text-slate-500 dark:text-slate-400">
                Keep your personal details and account security up to date.
              </p>
            </div>
          </header>

          <section className="rounded-xl border border-slate-200 bg-white p-6 shadow-card dark:border-slate-800 dark:bg-slate-900/60">
            <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <h2 className="text-lg font-semibold text-slate-900 dark:text-white">Appearance</h2>
                <p className="text-sm text-slate-500 dark:text-slate-400">
                  Choose between light and dark modes for the application interface.
                </p>
              </div>
              <div className="flex items-center gap-3">
                {themeOptions.map((option) => {
                  const isActive = theme === option.value
                  return (
                    <button
                      key={option.value}
                      type="button"
                      onClick={() => setTheme(option.value)}
                      aria-pressed={isActive}
                      className={`flex items-center gap-2 rounded-lg border px-4 py-2 text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50 focus-visible:ring-offset-2 focus-visible:ring-offset-white dark:focus-visible:ring-offset-slate-900 ${
                        isActive
                          ? 'border-primary bg-primary/10 text-primary-strong dark:border-primary/60 dark:bg-primary/20 dark:text-white'
                          : 'border-slate-300 text-slate-600 hover:border-primary hover:text-primary dark:border-slate-700 dark:text-slate-300 dark:hover:border-slate-500'
                      }`}
                    >
                      <span className="material-symbols-outlined text-base">{option.icon}</span>
                      <span>{option.label}</span>
                    </button>
                  )
                })}
              </div>
            </div>
            <p className="mt-3 text-xs text-slate-500 dark:text-slate-400">
              Theme preference is stored locally and switches immediately.
            </p>
          </section>

          <section className="rounded-xl border border-slate-200 bg-white p-6 shadow-card dark:border-slate-800 dark:bg-slate-900/60">
            <form className="flex flex-col gap-6" onSubmit={handleProfileSubmit}>
              <div>
                <h2 className="text-lg font-semibold text-slate-900 dark:text-white">
                  Personal information
                </h2>
                <p className="text-sm text-slate-500 dark:text-slate-400">
                  Update your personal details here.
                </p>
              </div>

              {profileSaveError && (
                <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-600 dark:border-red-500/50 dark:bg-red-500/10 dark:text-red-200">
                  {profileSaveError}
                </div>
              )}

              {profileSuccess && (
                <div className="rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-500/50 dark:bg-green-500/10 dark:text-green-200">
                  {profileSuccess}
                </div>
              )}

              <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
                <label className="flex flex-col gap-2">
                  <span className="text-sm font-medium text-slate-800 dark:text-slate-200">
                    First name
                  </span>
                  <input
                    value={profileForm.firstName}
                    onChange={handleProfileInputChange('firstName')}
                    placeholder="Alex"
                    className="form-input rounded-lg border border-slate-300 bg-background-light px-3.5 py-2.5 text-slate-900 placeholder:text-slate-400 focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20 dark:border-slate-700 dark:bg-background-dark dark:text-white dark:placeholder:text-slate-500"
                  />
                </label>
                <label className="flex flex-col gap-2">
                  <span className="text-sm font-medium text-slate-800 dark:text-slate-200">
                    Last name
                  </span>
                  <input
                    value={profileForm.lastName}
                    onChange={handleProfileInputChange('lastName')}
                    placeholder="Doe"
                    className="form-input rounded-lg border border-slate-300 bg-background-light px-3.5 py-2.5 text-slate-900 placeholder:text-slate-400 focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20 dark:border-slate-700 dark:bg-background-dark dark:text-white dark:placeholder:text-slate-500"
                  />
                </label>
              </div>

              <label className="flex flex-col gap-2">
                <span className="text-sm font-medium text-slate-800 dark:text-slate-200">
                  Email address
                </span>
                <input
                  type="email"
                  value={profileForm.email}
                  onChange={handleProfileInputChange('email')}
                  placeholder="alex.doe@example.com"
                  className="form-input rounded-lg border border-slate-300 bg-background-light px-3.5 py-2.5 text-slate-900 placeholder:text-slate-400 focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20 dark:border-slate-700 dark:bg-background-dark dark:text-white dark:placeholder:text-slate-500"
                  required
                />
              </label>

              <div className="flex justify-end gap-3">
                <Button
                  type="button"
                  variant="outline"
                  onClick={handleProfileCancel}
                  disabled={updateProfileMutation.isPending}
                  className="border-slate-300 text-slate-700 dark:border-slate-700 dark:text-white"
                >
                  Cancel
                </Button>
                <Button type="submit" disabled={updateProfileMutation.isPending}>
                  {updateProfileMutation.isPending ? 'Saving…' : 'Save changes'}
                </Button>
              </div>
            </form>
          </section>
          <section className="rounded-xl border border-slate-200 bg-white p-6 shadow-card dark:border-slate-800 dark:bg-slate-900/60">
            <form className="flex flex-col gap-6" onSubmit={handlePasswordSubmit}>
              <div>
                <h2 className="text-lg font-semibold text-slate-900 dark:text-white">Change password</h2>
                <p className="text-sm text-slate-500 dark:text-slate-400">
                  Update your password to keep your account secure.
                </p>
              </div>

              {passwordError && (
                <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-600 dark:border-red-500/50 dark:bg-red-500/10 dark:text-red-200">
                  {passwordError}
                </div>
              )}

              {passwordSuccess && (
                <div className="rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-500/50 dark:bg-green-500/10 dark:text-green-200">
                  {passwordSuccess}
                </div>
              )}

              <label className="flex flex-col gap-2">
                <span className="text-sm font-medium text-slate-800 dark:text-slate-200">
                  Current password
                </span>
                <input
                  type="password"
                  value={passwordForm.currentPassword}
                  onChange={handlePasswordInputChange('currentPassword')}
                  placeholder="Enter current password"
                  className="form-input rounded-lg border border-slate-300 bg-background-light px-3.5 py-2.5 text-slate-900 placeholder:text-slate-400 focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20 dark:border-slate-700 dark:bg-background-dark dark:text-white dark:placeholder:text-slate-500"
                  required
                />
              </label>

              <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
                <label className="flex flex-col gap-2">
                  <span className="text-sm font-medium text-slate-800 dark:text-slate-200">
                    New password
                  </span>
                  <input
                    type="password"
                    value={passwordForm.newPassword}
                    onChange={handlePasswordInputChange('newPassword')}
                    placeholder="Enter new password"
                    className="form-input rounded-lg border border-slate-300 bg-background-light px-3.5 py-2.5 text-slate-900 placeholder:text-slate-400 focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20 dark:border-slate-700 dark:bg-background-dark dark:text-white dark:placeholder:text-slate-500"
                    required
                  />
                </label>

                <label className="flex flex-col gap-2">
                  <span className="text-sm font-medium text-slate-800 dark:text-slate-200">
                    Confirm password
                  </span>
                  <input
                    type="password"
                    value={passwordForm.confirmPassword}
                    onChange={handlePasswordInputChange('confirmPassword')}
                    placeholder="Re-enter new password"
                    className="form-input rounded-lg border border-slate-300 bg-background-light px-3.5 py-2.5 text-slate-900 placeholder:text-slate-400 focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20 dark:border-slate-700 dark:bg-background-dark dark:text-white dark:placeholder:text-slate-500"
                    required
                  />
                </label>
              </div>

              <div className="flex justify-end gap-3">
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => {
                    setPasswordForm({ currentPassword: '', newPassword: '', confirmPassword: '' })
                    setPasswordError(null)
                    setPasswordSuccess(null)
                    updatePasswordMutation.reset()
                  }}
                  disabled={updatePasswordMutation.isPending}
                  className="border-slate-300 text-slate-700 dark:border-slate-700 dark:text-white"
                >
                  Cancel
                </Button>
                <Button type="submit" disabled={updatePasswordMutation.isPending}>
                  {updatePasswordMutation.isPending ? 'Updating…' : 'Update password'}
                </Button>
              </div>
            </form>
          </section>

          <section className="flex flex-col gap-6 rounded-xl border border-slate-200 bg-white p-6 shadow-card dark:border-slate-800 dark:bg-slate-900/60">
            <div className="flex flex-wrap items-center justify-between gap-4">
              <div>
                <h2 className="text-lg font-semibold text-slate-900 dark:text-white">Storage credentials</h2>
                <p className="text-sm text-slate-500 dark:text-slate-400">
                  Connect the accounts BucketBird can use to list and manage buckets.
                </p>
              </div>
              <Button onClick={handleCreateClick}>
                <span className="material-symbols-outlined text-base">add</span>
                Add new
              </Button>
            </div>

            {(!credentials || credentials.length === 0) && !credentialsLoading && (
              <div className="rounded-lg border border-slate-200 bg-slate-50 p-8 text-center dark:border-slate-700 dark:bg-slate-800/50">
                <span className="material-symbols-outlined mx-auto mb-3 block text-4xl text-slate-400">
                  storage
                </span>
                <h3 className="mb-1 text-base font-semibold text-slate-900 dark:text-white">
                  No credentials yet
                </h3>
                <p className="mb-4 text-sm text-slate-500 dark:text-slate-400">
                  Add your first storage credential to get started.
                </p>
                <Button onClick={handleCreateClick} className="mx-auto">
                  <span className="material-symbols-outlined text-base">add</span>
                  Add credential
                </Button>
              </div>
            )}

            {credentials && credentials.length > 0 && (
              <div className="grid gap-4 md:grid-cols-2">
                {credentials.map((cred) => (
                  <CredentialCard
                    key={cred.id}
                    credential={cred}
                    onEdit={() => handleEditClick(cred)}
                    onDelete={() => handleDeleteClick(cred.id, cred.name)}
                    onTest={() => handleTestClick(cred.id)}
                    isTestLoading={testMutation.isPending && testMutation.variables === cred.id}
                    testResult={testResults[cred.id] || null}
                  />
                ))}
              </div>
            )}

            {/* Delete Confirmation Dialog */}
            <ConfirmDialog
              open={credentialToDelete !== null}
              onClose={() => setCredentialToDelete(null)}
              onConfirm={handleConfirmDelete}
              title="Delete credential"
              description={
                <>
                  Are you sure you want to delete <strong>{credentialToDelete?.name}</strong>? This
                  will remove access to all buckets using this credential. This action cannot be
                  undone.
                </>
              }
              confirmText="Delete"
              cancelText="Cancel"
              variant="danger"
              isLoading={deleteMutation.isPending}
            />

            {formMode && (
              <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
                <div className="w-full max-w-2xl rounded-xl border border-slate-200 bg-white p-6 shadow-xl dark:border-slate-800 dark:bg-slate-900">
                  <CredentialForm
                    credential={selectedCredential || undefined}
                    onSubmit={handleFormSubmit}
                    onCancel={() => {
                      setFormMode(null)
                      setSelectedCredential(null)
                    }}
                    isLoading={createMutation.isPending || updateMutation.isPending}
                    error={formError}
                  />
                </div>
              </div>
            )}
          </section>
        </div>
      </main>
    </div>
  )
}

export default SettingsPage
