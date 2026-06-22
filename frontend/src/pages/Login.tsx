import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Wallet } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { useAuth } from '@/auth/AuthContext';

/**
 * Mock login: there is no password yet, the backend just needs a user id. The
 * form is intentionally already here so switching to real credentials later is
 * a change to this file + AuthContext only (see the AUTH SEAM note).
 */
export function Login() {
  const { login } = useAuth();
  const navigate = useNavigate();
  const [userId, setUserId] = useState('1');

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    const id = userId.trim();
    if (!id) return;
    login(id);
    navigate('/', { replace: true });
  };

  return (
    <div className="flex min-h-full flex-col justify-center px-6">
      <div className="mb-8 flex flex-col items-center gap-3 text-center">
        <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-primary text-primary-foreground">
          <Wallet className="h-8 w-8" />
        </div>
        <div>
          <h1 className="text-2xl font-semibold">fin-track</h1>
          <p className="text-sm text-muted-foreground">
            Track your spending, one receipt at a time.
          </p>
        </div>
      </div>

      <form onSubmit={submit} className="space-y-4">
        <div className="space-y-1.5">
          <label htmlFor="userId" className="text-sm font-medium">
            User ID
          </label>
          <Input
            id="userId"
            inputMode="numeric"
            value={userId}
            onChange={(e) => setUserId(e.target.value)}
            placeholder="1"
            autoFocus
          />
          <p className="text-xs text-muted-foreground">
            Authentication is mocked for now — any id identifies you.
          </p>
        </div>
        <Button type="submit" size="lg" className="w-full">
          Continue
        </Button>
      </form>
    </div>
  );
}
