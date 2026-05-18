import { create } from 'zustand';

interface AppState {
  theme: 'dark' | 'light';
  sidebarOpen: boolean;
  selectedTenantId: string | null;
  setTheme: (theme: 'dark' | 'light') => void;
  toggleSidebar: () => void;
  setSelectedTenantId: (id: string) => void;
}

export const useStore = create<AppState>((set) => ({
  theme: 'dark',
  sidebarOpen: true,
  selectedTenantId: null,
  setTheme: (theme) => set({ theme }),
  toggleSidebar: () => set((state) => ({ sidebarOpen: !state.sidebarOpen })),
  setSelectedTenantId: (id) => set({ selectedTenantId: id }),
}));
