import { useEffect, useState } from 'react'
import Button from '../ui/Button'

type PromptDialogProps = {
  isOpen: boolean
  onClose: () => void
  onConfirm: (value: string) => void
  title: string
  message?: string
  defaultValue?: string
  placeholder?: string
  confirmText?: string
  cancelText?: string
  allowEmpty?: boolean
}

export const PromptDialog = ({
  isOpen,
  onClose,
  onConfirm,
  title,
  message,
  defaultValue = '',
  placeholder,
  confirmText = 'OK',
  cancelText = 'Cancel',
  allowEmpty = false,
}: PromptDialogProps) => {
  const [value, setValue] = useState(defaultValue)

  useEffect(() => {
    if (isOpen) {
      setValue(defaultValue)
    }
  }, [isOpen, defaultValue])

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

  if (!isOpen) return null

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!allowEmpty && !value.trim()) {
      return
    }
    onConfirm(value.trim())
    onClose()
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div
        className="relative w-full max-w-md rounded-lg bg-white shadow-xl dark:bg-slate-900"
        onClick={(e) => e.stopPropagation()}
      >
        <form onSubmit={handleSubmit} className="flex flex-col gap-4 p-6">
          <div>
            <h3 className="text-lg font-semibold text-slate-900 dark:text-white">{title}</h3>
            {message && <p className="mt-2 text-sm text-slate-600 dark:text-slate-400">{message}</p>}
          </div>

          <input
            type="text"
            value={value}
            onChange={(e) => setValue(e.target.value)}
            placeholder={placeholder}
            autoFocus
            className="form-input w-full rounded-lg border border-slate-300 bg-white px-4 py-2.5 text-sm text-slate-900 placeholder:text-slate-400 focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20 dark:border-slate-600 dark:bg-surface-dark/80 dark:text-white dark:placeholder:text-slate-400"
          />

          <div className="flex justify-end gap-3">
            <Button variant="outline" type="button" onClick={onClose}>
              {cancelText}
            </Button>
            <Button type="submit" disabled={!allowEmpty && !value.trim()}>
              {confirmText}
            </Button>
          </div>
        </form>
      </div>
    </div>
  )
}
