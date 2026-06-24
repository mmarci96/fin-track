import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react';
import { USER_ID_STORAGE_KEY } from '@/lib/api';

// AUTH SEAM.
//
// PRODUCTION auth is handled entirely by the edge: the Traefik `traefikauth`
// plugin verifies the auth-service JWT cookie, strips any client-supplied
// X-User-ID, and injects the trusted one. The SPA only loads for already
// authenticated users, so it does not gate or redirect itself — FORCE_AUTH is
// false and the manual /login route is not registered in prod builds (App.tsx).
//
// DEV (npm run dev, no edge) keeps a mock identity: a user id typed into the
// dev-only /login page, sent as the X-User-ID header (see lib/api.ts). In prod
// that header is ignored (the edge overrides it), so this is harmless there.

export const FORCE_AUTH = false;

interface AuthContextValue {
  userId: string | null;
  isAuthenticated: boolean;
  login: (userId: string) => void;
  logout: () => void;
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [userId, setUserId] = useState<string | null>(() =>
    localStorage.getItem(USER_ID_STORAGE_KEY),
  );

  useEffect(() => {
    if (userId) localStorage.setItem(USER_ID_STORAGE_KEY, userId);
    else localStorage.removeItem(USER_ID_STORAGE_KEY);
  }, [userId]);

  const value = useMemo<AuthContextValue>(
    () => ({
      userId,
      isAuthenticated: userId != null,
      login: (id: string) => setUserId(id.trim()),
      logout: () => setUserId(null),
    }),
    [userId],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}
