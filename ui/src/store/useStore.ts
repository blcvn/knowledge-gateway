/**
 * Zustand Store — VNP Memory Console
 *
 * TASK-UI-004: Thêm tenant_id + avatar_url vào UserProfile interface.
 * Thêm setUser action để Login flow cập nhật store với đầy đủ fields.
 */

import { create } from 'zustand';

// ─── Types ────────────────────────────────────────────────────────────────────

export interface UserProfile {
  id:          string;
  name:        string;
  email:       string;
  role:        string;
  tenant_id:   string;   // Required for multi-tenant API calls (X-Tenant-ID header)
  avatar_url?: string;   // Optional avatar URL from backend
  /** @deprecated use avatar_url */
  avatar?:     string;
}

interface AppState {
  theme:            'dark' | 'light';
  sidebarOpen:      boolean;
  selectedTenantId: string | null;
  isAuthenticated:  boolean;
  token:            string | null;
  user:             UserProfile | null;

  // Actions
  setTheme:            (theme: 'dark' | 'light') => void;
  toggleSidebar:       () => void;
  setSelectedTenantId: (id: string) => void;
  /** @deprecated use setUser(user) — setAuth remains for backward compat */
  setAuth:             (token: string, user: UserProfile) => void;
  /**
   * TASK-UI-004: Preferred action for login flow.
   * Sets user + persists access_token reads from localStorage.
   */
  setUser:             (user: UserProfile) => void;
  logout:              () => void;
}

// ─── Store ────────────────────────────────────────────────────────────────────

export const useStore = create<AppState>((set) => ({
  theme:            'dark',
  sidebarOpen:      true,
  selectedTenantId: null,
  isAuthenticated:  false,
  token:            null,
  user:             null,

  setTheme:            (theme)  => set({ theme }),
  toggleSidebar:       ()       => set((s) => ({ sidebarOpen: !s.sidebarOpen })),
  setSelectedTenantId: (id)     => set({ selectedTenantId: id }),

  // Backward compat: Login.tsx calls setAuth(token, user)
  setAuth: (token, user) =>
    set({
      isAuthenticated:  true,
      token,
      user,
      // Also sync tenant to selectedTenantId for quick access
      selectedTenantId: user.tenant_id ?? null,
    }),

  // Preferred: Login.tsx should migrate to setUser(user)
  // Token is already in localStorage (set by authService.login)
  setUser: (user) =>
    set({
      isAuthenticated:  true,
      token:            localStorage.getItem('access_token'),
      user,
      selectedTenantId: user.tenant_id ?? null,
    }),

  logout: () =>
    set({
      isAuthenticated:  false,
      token:            null,
      user:             null,
      selectedTenantId: null,
    }),
}));
