import { AlertTriangle, CheckCircle2, XCircle } from 'lucide-react';
import { cn } from '@/lib/utils';

type Tone = 'success' | 'warning' | 'destructive';

const toneStyles: Record<Tone, string> = {
  success: 'border-success/40 bg-success/10 text-foreground',
  warning: 'border-warning/40 bg-warning/10 text-foreground',
  destructive: 'border-destructive/40 bg-destructive/10 text-foreground',
};

const toneIcon = {
  success: CheckCircle2,
  warning: AlertTriangle,
  destructive: XCircle,
};

export function WarningBanner({
  tone,
  title,
  messages,
}: {
  tone: Tone;
  title: string;
  messages?: string[] | null;
}) {
  const Icon = toneIcon[tone];
  return (
    <div className={cn('rounded-lg border p-3 text-sm', toneStyles[tone])}>
      <div className="flex items-start gap-2">
        <Icon className="mt-0.5 h-5 w-5 shrink-0" />
        <div className="space-y-1">
          <p className="font-medium">{title}</p>
          {messages && messages.length > 0 && (
            <ul className="list-disc space-y-0.5 pl-4 text-muted-foreground">
              {messages.map((m, i) => (
                <li key={i}>{m}</li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </div>
  );
}
