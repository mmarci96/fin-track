import { useNavigate } from 'react-router-dom';
import { Moon, Sun, LogOut, User } from 'lucide-react';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { useTheme } from '@/components/ThemeProvider';
import { useAuth } from '@/auth/AuthContext';

export function Settings() {
  const { theme, toggleTheme } = useTheme();
  const { userId, logout } = useAuth();
  const navigate = useNavigate();

  const onLogout = () => {
    logout();
    navigate('/login', { replace: true });
  };

  return (
    <div className="space-y-4 p-4">
      <h1 className="text-xl font-semibold">Settings</h1>

      <Card>
        <CardContent className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <User className="h-5 w-5 text-muted-foreground" />
            <div>
              <p className="font-medium">Signed in</p>
              <p className="text-sm text-muted-foreground">
                User ID {userId ?? '—'}
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardContent className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            {theme === 'dark' ? (
              <Moon className="h-5 w-5 text-muted-foreground" />
            ) : (
              <Sun className="h-5 w-5 text-muted-foreground" />
            )}
            <div>
              <p className="font-medium">Appearance</p>
              <p className="text-sm text-muted-foreground capitalize">
                {theme} mode
              </p>
            </div>
          </div>
          <Button variant="outline" size="sm" onClick={toggleTheme}>
            Switch
          </Button>
        </CardContent>
      </Card>

      <Button
        variant="ghost"
        className="w-full text-destructive hover:bg-destructive/10"
        onClick={onLogout}
      >
        <LogOut className="h-4 w-4" />
        Sign out
      </Button>
    </div>
  );
}
