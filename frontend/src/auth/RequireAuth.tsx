import { Navigate, useLocation } from 'react-router-dom';
import type { ReactNode } from 'react';
import { useAuth, FORCE_AUTH } from '@/auth/AuthContext';

/**
 * Route guard. While auth is mocked (FORCE_AUTH = false) it lets everyone
 * through — the backend falls back to a default user. Flip FORCE_AUTH to true
 * and unauthenticated visitors are bounced to /login with no other changes.
 */
export function RequireAuth({ children }: { children: ReactNode }) {
  const { isAuthenticated } = useAuth();
  const location = useLocation();

  if (FORCE_AUTH && !isAuthenticated) {
    return <Navigate to="/login" replace state={{ from: location }} />;
  }
  return <>{children}</>;
}
