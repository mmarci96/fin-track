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
// Today auth is mocked: the "identity" is just a user id that gets sent as the
// X-User-ID header (see lib/api.ts). To switch to forced authentication later:
//   1. set FORCE_AUTH = true so unauthenticated users are redirected to /login
//      (RequireAuth already enforces this),
//   2. change `login` to call POST /auth/login and store the returned token,
//   3. change lib/api.ts authHeaders() to send `Authorization: Bearer <token>`.
// No screen or feature code outside src/auth/ + lib/api.ts needs to change.

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
