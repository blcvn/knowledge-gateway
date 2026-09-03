/**
 * Profiles Hooks — real API, no mock
 * TASK-API-008: buffer status poll 30s, query key factory, useProfileConfig
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { profileService }   from '../services/profile.service';
import type { ProfileConfig } from '../types/profile';

// ─── Query Key Factory ────────────────────────────────────────────────────────

const keys = {
  all:     () => ['profiles'] as const,
  list:    () => [...keys.all(), 'list'] as const,
  detail:  (id: string) => [...keys.all(), id] as const,
  buffers: (id: string) => [...keys.all(), id, 'buffers'] as const,
  context: (id: string) => [...keys.all(), id, 'context'] as const,
  events:  (id: string) => [...keys.all(), id, 'events'] as const,
  config:  () => [...keys.all(), 'config'] as const,
};

// ─── Query Hooks ──────────────────────────────────────────────────────────────

/** GET /v1/console/profiles — list all user profiles */
export function useProfileList() {
  return useQuery({
    queryKey:  keys.list(),
    queryFn:   () => profileService.listProfiles(),
    staleTime: 5 * 60_000,
  });
}

/** GET /v1/console/profiles/{user_id} */
export function useProfileDetail(userId: string) {
  return useQuery({
    queryKey: keys.detail(userId),
    queryFn:  () => profileService.getProfile(userId),
    enabled:  !!userId,
  });
}

/**
 * GET /v1/console/profiles/{user_id}/buffers
 * Poll 30s — buffer fill changes with ingestion rate
 */
export function useBufferStatus(userId: string) {
  return useQuery({
    queryKey:        keys.buffers(userId),
    queryFn:         () => profileService.getBuffers(userId),
    enabled:         !!userId,
    refetchInterval: 30_000,
  });
}

/** GET /v1/console/profiles/{user_id}/context — assembled LLM prompt context */
export function useContextAssembly(userId: string) {
  return useQuery({
    queryKey: keys.context(userId),
    queryFn:  () => profileService.getContext(userId),
    enabled:  !!userId,
  });
}

/** GET /v1/console/profiles/{user_id}/events */
export function useUserEvents(userId: string) {
  return useQuery({
    queryKey: keys.events(userId),
    queryFn:  () => profileService.getEvents(userId),
    enabled:  !!userId,
  });
}

/** GET /v1/console/profiles/config */
export function useProfileConfig() {
  return useQuery({
    queryKey: keys.config(),
    queryFn:  () => profileService.getProfileConfig(),
  });
}

// ─── Mutation Hooks ───────────────────────────────────────────────────────────

/** PUT /v1/console/profiles/config */
export function useUpdateProfileConfig() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (config: Partial<ProfileConfig>) => profileService.updateProfileConfig(config),
    onSuccess:  () => qc.invalidateQueries({ queryKey: keys.config() }),
  });
}
