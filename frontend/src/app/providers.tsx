import type { ReactNode } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

import { ThemeProvider } from '../components/theme/ThemeProvider'
import { AuthProvider } from '../contexts/AuthContext'
import { BucketModalProvider } from '../contexts/BucketModalProvider'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 30,
      refetchOnWindowFocus: false,
    },
  },
})

type AppProvidersProps = {
  children: ReactNode
}

export const AppProviders = ({ children }: AppProvidersProps) => {
  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <BucketModalProvider>
          <ThemeProvider>{children}</ThemeProvider>
        </BucketModalProvider>
      </AuthProvider>
    </QueryClientProvider>
  )
}

export default AppProviders
