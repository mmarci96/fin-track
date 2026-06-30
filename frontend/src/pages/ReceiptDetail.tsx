import { useEffect } from 'react';
import { useParams, useNavigate, useLocation } from 'react-router-dom';
import { useForm, useFieldArray, Controller } from 'react-hook-form';
import { ArrowLeft, Plus, Trash2 } from 'lucide-react';
import {
  useReceipt,
  useUpdateReceipt,
  useDeleteReceipt,
  type Decision,
} from '@/api/receipts';
import { useUpdateMerchant } from '@/api/merchants';
import { useCategories } from '@/api/categories';
import { CategoryPicker } from '@/components/CategoryPicker';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent } from '@/components/ui/card';
import { CenteredSpinner } from '@/components/ui/spinner';
import { MoneyInput } from '@/components/MoneyInput';
import { WarningBanner } from '@/components/WarningBanner';
import {
  formatMoney,
  parseMoney,
  displayMoney,
  CURRENCY_CODES,
} from '@/lib/format';

interface FormRow {
  name: string;
  price: string; // display string in major units
  categoryIds: number[];
}
interface FormValues {
  merchant: string;
  total: string;
  currency: string;
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
  const updateMerchant = useUpdateMerchant();
  const remove = useDeleteReceipt();
  const { data: categories } = useCategories();

  const { register, control, handleSubmit, reset, watch, setValue } = useForm<FormValues>(
    {
      defaultValues: { merchant: '', total: '', currency: 'HUF', products: [] },
    },
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
      merchant: receipt.merchant,
      total: formatMoney(receipt.total, receipt.currency),
      currency: receipt.currency,
      products: receipt.products.map((p) => ({
        name: p.name,
        price: formatMoney(p.price, receipt.currency),
        categoryIds: p.categories.map((c) => c.id),
      })),
    });
  }, [receipt, reset]);

  // Format/parse against the currency currently selected in the form, so
  // switching it reformats the displayed amounts live.
  const currency = watch('currency');
  const rows = watch('products');
  const computedTotal = (rows ?? []).reduce(
    (sum, r) => sum + parseMoney(r.price || '0', currency),
    0,
  );
  const enteredTotal = parseMoney(watch('total') || '0', currency);
  const mismatch = Math.abs(computedTotal - enteredTotal) > 0;

  const applyToAll = (categoryId: number) => {
    const current = watch('products');
    current.forEach((row, i) => {
      if (!row.categoryIds.includes(categoryId)) {
        setValue(`products.${i}.categoryIds`, [...row.categoryIds, categoryId]);
      }
    });
  };

  const onSubmit = async (values: FormValues) => {
    if (!receipt) return;
    // The merchant name is global reference data, renamed via its own endpoint.
    // Only call it when the name actually changed and we have a real merchant.
    const merchant = values.merchant.trim();
    if (merchant && receipt.merchantId > 0 && merchant !== receipt.merchant) {
      await updateMerchant.mutateAsync({
        id: receipt.merchantId,
        name: merchant,
      });
    }
    await update.mutateAsync({
      total_amount: parseMoney(values.total, values.currency),
      currency: values.currency,
      products: values.products
        .filter((p) => p.name.trim() !== '')
        .map((p) => ({
          name: p.name.trim(),
          price: parseMoney(p.price, values.currency),
          category_ids: p.categoryIds,
        })),
    });
    navigate('/');
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
            {watch('merchant')?.trim() || 'Unknown merchant'}
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
        <h2 className="text-sm font-medium text-muted-foreground">Store</h2>
        <Card>
          <CardContent>
            <Input
              placeholder="Merchant name"
              aria-label="Merchant name"
              {...register('merchant')}
            />
          </CardContent>
        </Card>
      </section>

      <section className="space-y-2">
        <div className="flex items-center justify-between gap-2">
          <h2 className="text-sm font-medium text-muted-foreground">Items</h2>
          {categories && categories.length > 0 && (
            <select
              aria-label="Tag all items with category"
              className="h-8 rounded-md border border-border bg-card px-2 text-xs text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              value=""
              onChange={(e) => {
                if (e.target.value) applyToAll(Number(e.target.value));
                e.currentTarget.value = '';
              }}
            >
              <option value="">Tag all items…</option>
              {categories.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </select>
          )}
        </div>
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
              <Controller
                control={control}
                name={`products.${index}.categoryIds` as const}
                render={({ field }) => (
                  <CategoryPicker
                    value={field.value ?? []}
                    onChange={field.onChange}
                  />
                )}
              />
            </CardContent>
          </Card>
        ))}

        <Button
          type="button"
          variant="outline"
          className="w-full"
          onClick={() => append({ name: '', price: '', categoryIds: [] })}
        >
          <Plus className="h-4 w-4" />
          Add item
        </Button>
      </section>

      <section className="space-y-2">
        <h2 className="text-sm font-medium text-muted-foreground">Total</h2>
        <Card>
          <CardContent className="space-y-2">
            <div className="flex items-center gap-2">
              <MoneyInput
                aria-label="Total amount"
                className="flex-1"
                {...register('total')}
              />
              <select
                aria-label="Currency"
                className="h-10 rounded-md border border-input bg-background px-2 text-sm"
                {...register('currency')}
              >
                {CURRENCY_CODES.map((code) => (
                  <option key={code} value={code}>
                    {code}
                  </option>
                ))}
              </select>
            </div>
            <p className="text-xs text-muted-foreground">
              Items add up to{' '}
              <span className="font-medium tabular-nums">
                {displayMoney(computedTotal, currency)}
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
          disabled={update.isPending || updateMerchant.isPending}
        >
          {update.isPending || updateMerchant.isPending
            ? 'Saving…'
            : 'Save changes'}
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
