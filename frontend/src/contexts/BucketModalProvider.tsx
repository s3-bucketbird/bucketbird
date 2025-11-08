import { useState, type ReactNode } from 'react'

import { BucketModalContext } from './BucketModalContext'

export const BucketModalProvider = ({ children }: { children: ReactNode }) => {
  const [showCreateModal, setShowCreateModal] = useState(false)

  const openCreateModal = () => setShowCreateModal(true)
  const closeCreateModal = () => setShowCreateModal(false)

  return (
    <BucketModalContext.Provider value={{ showCreateModal, openCreateModal, closeCreateModal }}>
      {children}
    </BucketModalContext.Provider>
  )
}
