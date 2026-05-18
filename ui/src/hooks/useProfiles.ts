import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { profileService } from '../services/profile.service';
import { profileMock } from '../mock/profile.mock';
import { API_CONFIG } from '../config/api.config';

const useMock = API_CONFIG.useMockData;

/** Lists all user profiles via GET /v1/console/profiles */
export function useProfileList() {
  return useQuery({
    queryKey: ['profiles'],
    queryFn: useMock
      ? () => Promise.resolve(profileMock.users)
      : () => profileService.listProfiles(),
    staleTime: 5 * 60 * 1000,
  });
}

export function useProfileDetail(userId: string) {
  return useQuery({
    queryKey: ['profiles', userId],
    queryFn: useMock
      ? () => Promise.resolve(profileMock.userDetail)
      : () => profileService.getProfile(userId),
    enabled: !!userId,
  });
}

export function useBufferStatus(userId: string) {
  return useQuery({
    queryKey: ['profiles', userId, 'buffer'],
    queryFn: useMock
      ? () => Promise.resolve(profileMock.buffers[0])
      : () => profileService.getBuffers(userId),
    enabled: !!userId,
  });
}

export function useUserEvents(userId: string) {
  return useQuery({
    queryKey: ['profiles', userId, 'events'],
    queryFn: useMock
      ? () => Promise.resolve(profileMock.events)
      : () => profileService.getEvents(userId),
    enabled: !!userId,
  });
}

export function useContextAssembly(userId: string) {
  return useQuery({
    queryKey: ['profiles', userId, 'context'],
    queryFn: useMock
      ? () => Promise.resolve(profileMock.context)
      : () => profileService.getContext(userId),
    enabled: !!userId,
  });
}

export function useUpdateProfileConfig() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: profileService.updateProfileConfig,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['profiles', 'config'] }),
  });
}
