import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useForm, useFieldArray } from 'react-hook-form';
import { ArrowLeft, Plus, Trash2 } from 'lucide-react';
import { useCreateReceipt } from '@/api/receipts';
import { MerchantSelect } from '@/components/MerchantSelect';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent } from '@/components/ui/card';
import { MoneyInput } from '@/components/MoneyInput';
import { WarningBanner } from '@/components/WarningBanner';
import { parseMoney, displayMoney, CURRENCY_CODES } from '@/lib/format';

interface FormRow {
  name: string;
  price: string; // display string in major units
}
interface FormValues {
  total: string;
  currency: string;
  products: FormRow[];
}

/**
 * Post an expense without scanning. Mirrors the ReceiptDetail form, but creates
 * a fresh receipt against a chosen merchant and lands on the review screen.
 */
export function AddExpense() {
  const navigate = useNavigate();
  const create = useCreateReceipt();
  const [merchantId, setMerchantId] = useState<number | null>(null);
  const [formError, setFormError] = useState('');

  const { register, control, handleSubmit, watch } = useForm<FormValues>({
    defaultValues: {
      total: '',
      currency: 'HUF',
      products: [{ name: '', price: '' }],
    },
  });
  const { fields, append, remove } = useFieldArray({
    control,
    name: 'products',
  });

  const currency = watch('currency');
  const rows = watch('products');
  const computedTotal = (rows ?? []).reduce(
    (sum, r) => sum + parseMoney(r.price || '0', currency),
    0,
  );

  const onSubmit = (values: FormValues) => {
    setFormError('');
    if (!merchantId) {
      setFormError('Pick a store first.');
      return;
    }
    const products = values.products
      .filter((p) => p.name.trim() !== '')
      .map((p) => ({
        name: p.name.trim(),
        price: parseMoney(p.price, values.currency),
      }));
    if (products.length === 0) {
      setFormError('Add at least one item.');
      return;
    }

    // Leave the total blank to fall back to the items' sum.
    const enteredTotal = parseMoney(values.total || '0', values.currency);
    const total_amount =
      enteredTotal > 0
        ? enteredTotal
        : products.reduce((s, p) => s + p.price, 0);

    create.mutate(
      {
        merchant_id: merchantId,
        total_amount,
        currency: values.currency,
        products,
      },
      {
        onSuccess: (receipt) =>
          navigate(`/receipts/${receipt.id}`, { replace: true }),
      },
    );
  };

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
          <h1 className="text-lg font-semibold leading-tight">Add expense</h1>
          <p className="text-xs text-muted-foreground">Enter it manually</p>
        </div>
      </header>

      {formError && <WarningBanner tone="destructive" title={formError} />}
      {create.isError && (
        <WarningBanner
          tone="destructive"
          title="Could not save"
          messages={[(create.error as Error).message]}
        />
      )}

      <section className="space-y-2">
        <h2 className="text-sm font-medium text-muted-foreground">Store</h2>
        <MerchantSelect value={merchantId} onChange={setMerchantId} />
      </section>

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
                  onClick={() => remove(index)}
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
              . Leave the total blank to use this sum.
            </p>
          </CardContent>
        </Card>
      </section>

      <Button
        type="submit"
        size="lg"
        className="w-full"
        disabled={create.isPending}
      >
        {create.isPending ? 'Saving…' : 'Save expense'}
      </Button>
    </form>
  );
}
