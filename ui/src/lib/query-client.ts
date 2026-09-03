/**
 * React Query global configuration for Enterprise-grade data fetching.
 *
 * Features:
 * - Smart retry: skip 401/403 auth errors (no infinite loop)
 * - Smart caching with configurable staleTime / gcTime
 * - Global error handler via MutationCache / QueryCache
 * - Optimistic Update utilities
 */

import { QueryClient } from '@tanstack/react-query';
import { AppError }    from './api-client';
import { logger }      from './logger';

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,            // 30s default — modules override per need
      gcTime:    30 * 60 * 1000,   // 30 minutes garbage collect
      retry: (failureCount, error) => {
        // Never retry auth errors — they go through refresh interceptor instead
        if (error instanceof AppError && error.status === 401) return false;
        if (error instanceof AppError && error.status === 403) return false;
        return failureCount < 2;
      },
      refetchOnWindowFocus: false,
    },
    mutations: {
      retry: false,
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
