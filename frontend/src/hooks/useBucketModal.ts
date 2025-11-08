import { useContext } from 'react'
import { BucketModalContext } from '../contexts/BucketModalContext'

export const useBucketModal = () => {
  const context = useContext(BucketModalContext)
  if (!context) {
    throw new Error('useBucketModal must be used within BucketModalProvider')
  }
  return context
}
