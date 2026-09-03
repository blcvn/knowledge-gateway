import React, {
  createContext, useContext, useState, useEffect,
  useCallback, ReactNode,
} from 'react';
import { authApiClient } from '../services/auth-api.service';
import type { AuthUser } from '../types/auth';

// ─── Types ────────────────────────────────────────────────────────────────────

export type UserRole = 'admin' | 'developer' | 'viewer' | 'devops' | 'ml_engineer' | 'architect';

/** Backward-compat User shape (maps from AuthUser) */
export interface User {
  id:       string;
  email:    string;
  name:     string;
  roles:    UserRole[];
  tenantId: string;
}

interface AuthState {
  user:            User | null;
  isAuthenticated: boolean;
  isLoading:       boolean;
}

interface AuthContextType extends AuthState {
  login:      (email: string, password: string) => Promise<void>;
  logout:     () => void;
  hasRole:    (role: UserRole) => boolean;
  hasAnyRole: (roles: UserRole[]) => boolean;
}

// ─── Context ──────────────────────────────────────────────────────────────────

const AuthContext = createContext<AuthContextType | undefined>(undefined);

// ─── Helpers ──────────────────────────────────────────────────────────────────

function mapApiUserToUser(apiUser: AuthUser): User {
  return {
    id:       apiUser.id,
    email:    apiUser.email,
    name:     apiUser.name,
    roles:    [(apiUser.role as UserRole) ?? 'viewer'],
    tenantId: apiUser.tenant_id,
  };
}

// ─── Provider ─────────────────────────────────────────────────────────────────

const IDLE_TIMEOUT_MS = 30 * 60 * 1000;

export const AuthProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
  const [state, setState] = useState<AuthState>({
    user:            null,
    isAuthenticated: false,
    isLoading:       true,
  });

  // Restore session from token on mount — calls real /v1/auth/me
  useEffect(() => {
    if (!authApiClient.isAuthenticated()) {
      setState({ user: null, isAuthenticated: false, isLoading: false });
      return;
    }

    authApiClient
      .getMe()
      .then((apiUser) => {
        setState({
          user:            mapApiUserToUser(apiUser),
          isAuthenticated: true,
          isLoading:       false,
        });
      })
      .catch(() => {
        // Token invalid — clear and show login
        setState({ user: null, isAuthenticated: false, isLoading: false });
      });
  }, []);

  // Idle timeout — auto logout after 30min inactivity
  useEffect(() => {
    if (!state.isAuthenticated) return;

    let timeoutId: ReturnType<typeof setTimeout>;
    const resetTimer = () => {
      clearTimeout(timeoutId);
      timeoutId = setTimeout(() => logout(), IDLE_TIMEOUT_MS);
    };

    const events = ['mousedown', 'keydown', 'scroll', 'touchstart'];
    events.forEach((e) => window.addEventListener(e, resetTimer));
    resetTimer();

    return () => {
      clearTimeout(timeoutId);
      events.forEach((e) => window.removeEventListener(e, resetTimer));
    };
  }, [state.isAuthenticated]);

  const login = useCallback(async (email: string, password: string) => {
    const resp = await authApiClient.login({ email, password });
    setState({
      user:            mapApiUserToUser(resp.user),
      isAuthenticated: true,
      isLoading:       false,
    });
  }, []);

  const logout = useCallback(async () => {
    try {
      await authApiClient.logout();
    } finally {
      setState({ user: null, isAuthenticated: false, isLoading: false });
      window.location.replace('/login');
    }
  }, []);

  const hasRole    = useCallback((role: UserRole) => state.user?.roles.includes(role) ?? false, [state.user]);
  const hasAnyRole = useCallback((roles: UserRole[]) => roles.some((r) => state.user?.roles.includes(r)), [state.user]);

  return (
    <AuthContext.Provider value={{ ...state, login, logout, hasRole, hasAnyRole }}>
      {children}
    </AuthContext.Provider>
  );
};

// ─── Hook ─────────────────────────────────────────────────────────────────────

export function useAuth(): AuthContextType {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within an AuthProvider');
  return ctx;
}

// ─── Route Guard ──────────────────────────────────────────────────────────────

interface RouteGuardProps {
  children:      ReactNode;
  requiredRoles?: UserRole[];
  fallback?:     ReactNode;
}

export const RouteGuard: React.FC<RouteGuardProps> = ({ children, requiredRoles, fallback }) => {
  const { isAuthenticated, isLoading, hasAnyRole } = useAuth();

  if (isLoading) {
    return (
      <div className="flex-1 flex items-center justify-center bg-[#0a0a0f]">
        <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-blue-500" />
      </div>
    );
  }

  if (!isAuthenticated) {
    return fallback ?? (
      <div className="flex-1 flex items-center justify-center bg-[#0a0a0f] text-white">
        <p>Please log in to continue.</p>
      </div>
    );
  }

  if (requiredRoles && requiredRoles.length > 0 && !hasAnyRole(requiredRoles)) {
    return (
      <div className="flex-1 flex items-center justify-center bg-[#0a0a0f] text-white">
        <div className="text-center">
          <h2 className="text-xl font-bold mb-2">Access Denied</h2>
          <p className="text-zinc-400">You do not have permission to view this page.</p>
        </div>
      </div>
    );
  }

  return <>{children}</>;
};

// ─── RBAC Component Visibility ────────────────────────────────────────────────

interface RBACProps {
  roles:    UserRole[];
  children: ReactNode;
  fallback?: ReactNode;
}

export const RequireRole: React.FC<RBACProps> = ({ roles, children, fallback }) => {
  const { hasAnyRole } = useAuth();
  if (!hasAnyRole(roles)) return fallback ? <>{fallback}</> : null;
  return <>{children}</>;
};
