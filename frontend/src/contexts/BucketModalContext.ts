import { createContext } from 'react'

export type BucketModalContextType = {
  showCreateModal: boolean
  openCreateModal: () => void
  closeCreateModal: () => void
}

export const BucketModalContext = createContext<BucketModalContextType | undefined>(undefined)
