import { X } from 'lucide-react';
import { useCategories } from '@/api/categories';

// Sentinel value for the "add" dropdown's placeholder option.
const NONE = '';

/**
 * Per-item category editor: shows the assigned categories as removable chips
 * plus a dropdown of the remaining global categories to add more. Controlled —
 * `value` is the list of assigned category ids.
 */
export function CategoryPicker({
  value,
  onChange,
}: {
  value: number[];
  onChange: (ids: number[]) => void;
}) {
  const { data: categories, isLoading } = useCategories();

  const selected = (categories ?? []).filter((c) => value.includes(c.id));
  const available = (categories ?? []).filter((c) => !value.includes(c.id));

  const add = (id: number) => {
    if (!value.includes(id)) onChange([...value, id]);
  };
  const removeId = (id: number) => onChange(value.filter((v) => v !== id));

  return (
    <div className="space-y-2">
      {selected.length > 0 && (
        <div className="flex flex-wrap gap-1.5">
          {selected.map((c) => (
            <span
              key={c.id}
              className="inline-flex items-center gap-1 rounded-full bg-primary/10 px-2.5 py-1 text-xs font-medium text-primary"
            >
              {c.name}
              <button
                type="button"
                onClick={() => removeId(c.id)}
                aria-label={`Remove ${c.name}`}
                className="text-muted-foreground hover:text-foreground"
              >
                <X className="h-3 w-3" />
              </button>
            </span>
          ))}
        </div>
      )}

      <select
        aria-label="Add category"
        className="h-9 w-full rounded-md border border-border bg-card px-2 text-sm text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50"
        value={NONE}
        disabled={isLoading || available.length === 0}
        onChange={(e) => {
          if (e.target.value === NONE) return;
          add(Number(e.target.value));
        }}
      >
        <option value={NONE}>
          {isLoading
            ? 'Loading categories…'
            : available.length === 0
              ? 'All categories added'
              : '+ Add category…'}
        </option>
        {available.map((c) => (
          <option key={c.id} value={c.id}>
            {c.name}
          </option>
        ))}
      </select>
    </div>
  );
}
