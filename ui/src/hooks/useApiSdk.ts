/**
 * @deprecated Use useOrganizationSettings.ts instead — all hooks have been consolidated.
 * This file re-exports for backward compatibility with existing page components.
 */
export {
  useApiKeys,
  useRateLimits,
  useWebhooks,
  useCreateApiKey,
  useRevokeApiKey,
  useCreateWebhook,
  useDeleteWebhook,
} from './useOrganizationSettings';
