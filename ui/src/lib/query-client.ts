/**
 * React Query global configuration for Enterprise-grade data fetching.
 *
 * Features:
 * - Smart caching with configurable staleTime / gcTime
 * - Automatic retry with exponential backoff for transient errors
 * - Global error handler via MutationCache / QueryCache
 * - Optimistic Update utilities
 */

import { QueryClient } from '@tanstack/react-query';
import { logger } from './logger';

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5 * 60 * 1000,     // 5 minutes — data is fresh for this period
      gcTime: 30 * 60 * 1000,        // 30 minutes — garbage collect unused cache
      retry: 2,                       // Retry failed queries up to 2 times
      retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 10000),
      refetchOnWindowFocus: false,    // Prevent refetch on window focus (dev-friendly)
    },
    mutations: {
      retry: 1,
    },
  },
});

// --- Global Error Handlers ---

queryClient.getQueryCache().subscribe((event) => {
  if (event.type === 'updated' && event.query.state.status === 'error') {
    const error = event.query.state.error;
    logger.error('React Query Error:', error);
    // Here you'd trigger a global toast notification
  }
});

queryClient.getMutationCache().subscribe((event) => {
  if (event.type === 'updated' && event.mutation?.state.status === 'error') {
    const error = event.mutation.state.error;
    logger.error('React Mutation Error:', error);
  }
});

// --- Optimistic Update Helpers ---

/**
 * Creates standard optimistic update callbacks for React Query mutations.
 *
 * Usage:
 * ```ts
 * const mutation = useMutation({
 *   mutationFn: updateItem,
 *   ...createOptimisticUpdate<Item[]>(['items'], (old, newItem) =>
 *     old.map(i => i.id === newItem.id ? newItem : i)
 *   ),
 * });
 * ```
 */
export function createOptimisticUpdate<TData>(
  queryKey: unknown[],
  updater: (oldData: TData, variables: any) => TData
) {
  return {
    onMutate: async (variables: any) => {
      await queryClient.cancelQueries({ queryKey });
      const previousData = queryClient.getQueryData<TData>(queryKey);
      if (previousData !== undefined) {
        queryClient.setQueryData<TData>(queryKey, (old) =>
          old !== undefined ? updater(old, variables) : old
        );
      }
      return { previousData };
    },
    onError: (_err: unknown, _vars: unknown, context: { previousData?: TData } | undefined) => {
      if (context?.previousData !== undefined) {
        queryClient.setQueryData(queryKey, context.previousData);
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey });
    },
  };
}
