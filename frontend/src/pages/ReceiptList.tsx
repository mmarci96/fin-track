import { Link, useNavigate } from 'react-router-dom';
import { ChevronRight, Camera, ReceiptText, PenLine } from 'lucide-react';
import { useReceipts } from '@/api/receipts';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { CenteredSpinner } from '@/components/ui/spinner';
import { WarningBanner } from '@/components/WarningBanner';
import { displayMoney } from '@/lib/format';

export function ReceiptList() {
  const navigate = useNavigate();
  const { data: receipts, isLoading, isError, error } = useReceipts();

  return (
    <div className="space-y-4 p-4">
      <header className="flex items-center justify-between">
        <h1 className="text-xl font-semibold">Receipts</h1>
        <div className="flex items-center gap-2">
          <Button
            size="sm"
            variant="outline"
            onClick={() => navigate('/expenses/new')}
          >
            <PenLine className="h-4 w-4" />
            Add
          </Button>
          <Button size="sm" onClick={() => navigate('/scan')}>
            <Camera className="h-4 w-4" />
            Scan
          </Button>
        </div>
      </header>

      {isLoading && <CenteredSpinner />}

      {isError && (
        <WarningBanner
          tone="destructive"
          title="Could not load receipts"
          messages={[(error as Error).message]}
        />
      )}

      {receipts && receipts.length === 0 && (
        <Card>
          <CardContent className="flex flex-col items-center gap-3 py-12 text-center">
            <ReceiptText className="h-10 w-10 text-muted-foreground" />
            <div>
              <p className="font-medium">No receipts yet</p>
              <p className="text-sm text-muted-foreground">
                Scan your first receipt to get started.
              </p>
            </div>
            <Button onClick={() => navigate('/scan')}>
              <Camera className="h-4 w-4" />
              Scan a receipt
            </Button>
          </CardContent>
        </Card>
      )}

      {receipts && receipts.length > 0 && (
        <ul className="space-y-2">
          {receipts.map((r) => (
            <li key={r.id}>
              <Link to={`/receipts/${r.id}`}>
                <Card className="transition-colors hover:bg-muted/40">
                  <CardContent className="flex items-center justify-between">
                    <div>
                      <p className="font-medium">{r.merchant || 'Unknown'}</p>
                      <p className="text-sm text-muted-foreground">
                        {r.products.length} item
                        {r.products.length === 1 ? '' : 's'}
                      </p>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="text-lg font-semibold tabular-nums">
                        {displayMoney(r.total, r.currency)}
                      </span>
                      <ChevronRight className="h-5 w-5 text-muted-foreground" />
                    </div>
                  </CardContent>
                </Card>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
