'use client';

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useEffect, useState } from 'react';

export function ReactQueryProvider({ children }: { children: React.ReactNode }) {
  const [queryClient] = useState(() => new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 1000 * 30,
        refetchOnWindowFocus: false,
        retry: 1,
      },
    },
  }));

  // Clear all caches on forced logout (refresh failure)
  useEffect(() => {
    const handler = () => {
      queryClient.clear();
    };
    window.addEventListener('pat:logout', handler);
    return () => window.removeEventListener('pat:logout', handler);
  }, [queryClient]);

  return (
    <QueryClientProvider client={queryClient}>
      {children}
    </QueryClientProvider>
  );
}
