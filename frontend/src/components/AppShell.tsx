import { NavLink, useLocation } from 'react-router-dom';
import type { ReactNode } from 'react';
import { Receipt, Camera, BarChart3, Settings } from 'lucide-react';
import { cn } from '@/lib/utils';

const tabs = [
  { to: '/', label: 'Receipts', icon: Receipt, end: true },
  { to: '/scan', label: 'Scan', icon: Camera, end: false },
  { to: '/statistics', label: 'Stats', icon: BarChart3, end: false },
  { to: '/settings', label: 'Settings', icon: Settings, end: false },
];

/**
 * Mobile-first frame: a centered, phone-width column with a fixed bottom tab
 * bar. The bar is hidden on the full-screen scan flow so the cropper owns the
 * viewport.
 */
export function AppShell({ children }: { children: ReactNode }) {
  const { pathname } = useLocation();
  const immersive = pathname === '/scan';

  return (
    <div className="mx-auto flex min-h-full w-full max-w-md flex-col md:max-w-2xl lg:max-w-3xl">
      <main className={cn('flex-1', immersive ? '' : 'pb-20')}>{children}</main>

      {!immersive && (
        <nav className="fixed inset-x-0 bottom-0 z-20 mx-auto flex w-full max-w-md items-center justify-around border-t border-border bg-card/95 backdrop-blur md:max-w-2xl lg:max-w-3xl">
          {tabs.map(({ to, label, icon: Icon, end }) => (
            <NavLink
              key={to}
              to={to}
              end={end}
              className={({ isActive }) =>
                cn(
                  'flex flex-1 flex-col items-center gap-1 py-3 text-xs font-medium transition-colors',
                  isActive
                    ? 'text-primary'
                    : 'text-muted-foreground hover:text-foreground',
                )
              }
            >
              <Icon className="h-6 w-6" />
              {label}
            </NavLink>
          ))}
        </nav>
      )}
    </div>
  );
}
