import { useEffect } from 'react';
import { useParams, useNavigate, useLocation } from 'react-router-dom';
import { useForm, useFieldArray } from 'react-hook-form';
import { ArrowLeft, Plus, Trash2 } from 'lucide-react';
import {
  useReceipt,
  useUpdateReceipt,
  useDeleteReceipt,
  type Decision,
} from '@/api/receipts';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent } from '@/components/ui/card';
import { CenteredSpinner } from '@/components/ui/spinner';
import { MoneyInput } from '@/components/MoneyInput';
import { WarningBanner } from '@/components/WarningBanner';
import { formatMoney, parseMoney } from '@/lib/format';

interface FormRow {
  name: string;
  price: string; // display string in major units
}
interface FormValues {
  total: string;
  products: FormRow[];
}

// Navigation state passed from the scan flow so we can nudge review when the
// parse was only best-effort.
interface NavState {
  decision?: Decision;
  warnings?: string[];
  reconciled?: boolean;
}

export function ReceiptDetail() {
  const { id: idParam } = useParams();
  const id = Number(idParam);
  const navigate = useNavigate();
  const { state } = useLocation() as { state: NavState | null };

  const { data: receipt, isLoading, isError } = useReceipt(id);
  const update = useUpdateReceipt(id);
  const remove = useDeleteReceipt();

  const { register, control, handleSubmit, reset, watch } = useForm<FormValues>(
    { defaultValues: { total: '', products: [] } },
  );
  const {
    fields,
    append,
    remove: removeRow,
  } = useFieldArray({
    control,
    name: 'products',
  });

  // Seed the form once the receipt loads.
  useEffect(() => {
    if (!receipt) return;
    reset({
      total: formatMoney(receipt.total),
      products: receipt.products.map((p) => ({
        name: p.name,
        price: formatMoney(p.price),
      })),
    });
  }, [receipt, reset]);

  const rows = watch('products');
  const computedTotal = (rows ?? []).reduce(
    (sum, r) => sum + parseMoney(r.price || '0'),
    0,
  );
  const enteredTotal = parseMoney(watch('total') || '0');
  const mismatch = Math.abs(computedTotal - enteredTotal) > 0;

  const onSubmit = (values: FormValues) => {
    update.mutate(
      {
        total_amount: parseMoney(values.total),
        products: values.products
          .filter((p) => p.name.trim() !== '')
          .map((p) => ({ name: p.name.trim(), price: parseMoney(p.price) })),
      },
      { onSuccess: () => navigate('/') },
    );
  };

  const onDelete = () => {
    if (!confirm('Delete this receipt?')) return;
    remove.mutate(id, { onSuccess: () => navigate('/', { replace: true }) });
  };

  if (isLoading) return <CenteredSpinner />;
  if (isError || !receipt)
    return (
      <div className="p-4">
        <WarningBanner tone="destructive" title="Receipt not found" />
      </div>
    );

  const bestEffort = state?.decision === 'best_effort';
  const accepted = state?.decision === 'accepted';

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4 p-4">
      <header className="flex items-center gap-2">
        <Button
          type="button"
          variant="ghost"
          size="icon"
          onClick={() => navigate('/')}
        >
          <ArrowLeft className="h-5 w-5" />
        </Button>
        <div>
          <h1 className="text-lg font-semibold leading-tight">
            {receipt.merchant || 'Unknown merchant'}
          </h1>
          <p className="text-xs text-muted-foreground">Review &amp; edit</p>
        </div>
      </header>

      {bestEffort && (
        <WarningBanner
          tone="warning"
          title="Please double-check this one"
          messages={[
            ...(state?.warnings ?? []),
            state?.reconciled === false
              ? "The total didn't match the items — verify the prices."
              : '',
          ].filter(Boolean)}
        />
      )}
      {accepted && (
        <WarningBanner
          tone="success"
          title="Scanned and saved"
          messages={['Edit anything below if needed.']}
        />
      )}

      <section className="space-y-2">
        <h2 className="text-sm font-medium text-muted-foreground">Items</h2>
        {fields.map((field, index) => (
          <Card key={field.id}>
            <CardContent className="space-y-2">
              <Input
                placeholder="Item name"
                aria-label="Item name"
                {...register(`products.${index}.name` as const)}
              />
              <div className="flex items-center gap-2">
                <MoneyInput
                  aria-label="Item price"
                  className="flex-1"
                  {...register(`products.${index}.price` as const)}
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  onClick={() => removeRow(index)}
                  aria-label="Remove item"
                >
                  <Trash2 className="h-5 w-5 text-destructive" />
                </Button>
              </div>
            </CardContent>
          </Card>
        ))}

        <Button
          type="button"
          variant="outline"
          className="w-full"
          onClick={() => append({ name: '', price: '' })}
        >
          <Plus className="h-4 w-4" />
          Add item
        </Button>
      </section>

      <section className="space-y-2">
        <h2 className="text-sm font-medium text-muted-foreground">Total</h2>
        <Card>
          <CardContent className="space-y-2">
            <MoneyInput aria-label="Total amount" {...register('total')} />
            <p className="text-xs text-muted-foreground">
              Items add up to{' '}
              <span className="font-medium tabular-nums">
                {formatMoney(computedTotal)}
              </span>
              {mismatch && (
                <span className="ml-1 font-medium text-warning">
                  — doesn&apos;t match the total above.
                </span>
              )}
            </p>
          </CardContent>
        </Card>
      </section>

      <div className="space-y-2 pt-2">
        <Button
          type="submit"
          size="lg"
          className="w-full"
          disabled={update.isPending}
        >
          {update.isPending ? 'Saving…' : 'Save changes'}
        </Button>
        <Button
          type="button"
          variant="ghost"
          className="w-full text-destructive hover:bg-destructive/10"
          onClick={onDelete}
          disabled={remove.isPending}
        >
          <Trash2 className="h-4 w-4" />
          Delete receipt
        </Button>
      </div>
    </form>
  );
}
