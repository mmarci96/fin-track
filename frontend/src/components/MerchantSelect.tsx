import { useState } from 'react';
import { useMerchants, useCreateMerchant } from '@/api/merchants';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Spinner } from '@/components/ui/spinner';

// Sentinel option value that switches the control into "add a new store" mode.
const ADD_NEW = '__add_new__';

/**
 * Controlled store picker: a native dropdown of the user's merchants plus an
 * "add new" option that reveals an inline input. Creating a store selects it.
 */
export function MerchantSelect({
  value,
  onChange,
}: {
  value: number | null;
  onChange: (id: number) => void;
}) {
  const { data: merchants, isLoading } = useMerchants();
  const create = useCreateMerchant();
  const [adding, setAdding] = useState(false);
  const [name, setName] = useState('');

  const confirmAdd = () => {
    const trimmed = name.trim();
    if (!trimmed) return;
    create.mutate(trimmed, {
      onSuccess: (m) => {
        onChange(m.id);
        setAdding(false);
        setName('');
      },
    });
  };

  if (adding) {
    return (
      <div className="flex items-center gap-2">
        <Input
          autoFocus
          placeholder="New store name"
          aria-label="New store name"
          className="flex-1"
          value={name}
          onChange={(e) => setName(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              e.preventDefault();
              confirmAdd();
            }
          }}
        />
        <Button
          type="button"
          onClick={confirmAdd}
          disabled={!name.trim() || create.isPending}
        >
          {create.isPending ? <Spinner /> : 'Add'}
        </Button>
        <Button
          type="button"
          variant="ghost"
          onClick={() => {
            setAdding(false);
            setName('');
          }}
          disabled={create.isPending}
        >
          Cancel
        </Button>
      </div>
    );
  }

  return (
    <select
      aria-label="Store"
      className="h-11 w-full rounded-md border border-border bg-card px-3 text-base text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50"
      value={value ?? ''}
      disabled={isLoading}
      onChange={(e) => {
        if (e.target.value === ADD_NEW) {
          setAdding(true);
          return;
        }
        onChange(Number(e.target.value));
      }}
    >
      <option value="" disabled>
        {isLoading ? 'Loading stores…' : 'Select a store'}
      </option>
      {(merchants ?? []).map((m) => (
        <option key={m.id} value={m.id}>
          {m.name}
        </option>
      ))}
      <option value={ADD_NEW}>+ Add new store…</option>
    </select>
  );
}
