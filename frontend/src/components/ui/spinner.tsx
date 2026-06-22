import { Loader2 } from 'lucide-react';
import { cn } from '@/lib/utils';

export function Spinner({ className }: { className?: string }) {
  return <Loader2 className={cn('h-5 w-5 animate-spin', className)} />;
}

export function CenteredSpinner() {
  return (
    <div className="flex justify-center py-16 text-muted-foreground">
      <Spinner className="h-8 w-8" />
    </div>
  );
}
