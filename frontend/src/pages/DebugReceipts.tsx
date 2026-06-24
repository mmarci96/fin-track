import { useEffect, useState } from 'react';
import { Button } from '@/components/ui/button';
import {
  useDebugImages,
  useDebugImageMeta,
  useSaveCleanText,
  debugImageUrl,
  type ParseResult,
} from '@/api/debug';

/**
 * Developer-only tool (dev builds only — see App.tsx). Lists the persisted debug
 * uploads and shows the original image next to its OCR transcript and parser
 * output, so mistakes are easy to spot. A "clean transcript" editor captures the
 * corrected ground truth that feeds the recognition flywheel.
 *
 * Laptop-oriented layout by design: these are for developers, not end users.
 */
export function DebugReceipts() {
  const { data: images, isLoading } = useDebugImages();
  const [selected, setSelected] = useState<number | null>(null);

  // Auto-select the newest capture once the list loads.
  useEffect(() => {
    if (selected == null && images && images.length > 0) {
      setSelected(images[0].id);
    }
  }, [images, selected]);

  return (
    <div className="px-4 py-4">
      <h1 className="mb-4 text-xl font-semibold">
        Debug — image ⟷ transcript
      </h1>

      {isLoading && <p className="text-muted-foreground">Loading…</p>}
      {!isLoading && (!images || images.length === 0) && (
        <p className="text-muted-foreground">
          No debug uploads yet. POST an image to{' '}
          <code className="rounded bg-muted px-1">/api/receipts/image/debug</code>{' '}
          (e.g. <code className="rounded bg-muted px-1">make -C backend test-debug-img</code>).
        </p>
      )}

      {images && images.length > 0 && (
        <div className="grid grid-cols-[16rem_1fr] gap-4">
          {/* Capture list */}
          <aside className="max-h-[80vh] space-y-1 overflow-auto border-r border-border pr-2">
            {images.map((img) => (
              <button
                key={img.id}
                onClick={() => setSelected(img.id)}
                className={
                  'block w-full truncate rounded px-2 py-1.5 text-left text-sm ' +
                  (selected === img.id
                    ? 'bg-primary text-primary-foreground'
                    : 'hover:bg-muted')
                }
              >
                <span className="font-medium">#{img.id}</span>{' '}
                {img.originalName || '(unnamed)'}
                <span className="block text-xs opacity-70">
                  {new Date(img.createdAt).toLocaleString()}
                  {img.receiptId ? ` · receipt ${img.receiptId}` : ' · no receipt'}
                </span>
              </button>
            ))}
          </aside>

          {/* Detail */}
          <section>
            {selected != null ? (
              <DebugDetail id={selected} />
            ) : (
              <p className="text-muted-foreground">Select a capture.</p>
            )}
          </section>
        </div>
      )}
    </div>
  );
}

function DebugDetail({ id }: { id: number }) {
  const { data: meta, isLoading } = useDebugImageMeta(id);
  const save = useSaveCleanText(id);
  const [clean, setClean] = useState('');

  // Seed the editor from the stored clean text, falling back to the raw OCR so
  // correcting is edit-in-place rather than retyping.
  useEffect(() => {
    if (meta) setClean(meta.cleanText ?? meta.ocrText ?? '');
  }, [meta]);

  if (isLoading || !meta) return <p className="text-muted-foreground">Loading…</p>;

  return (
    <div className="grid grid-cols-2 gap-4">
      {/* Left: the image */}
      <div className="overflow-auto rounded border border-border bg-muted/30 p-2">
        <img
          src={debugImageUrl(id)}
          alt={meta.originalName}
          className="mx-auto max-h-[78vh] w-auto object-contain"
        />
      </div>

      {/* Right: parse summary, transcript, clean-text editor */}
      <div className="space-y-4 overflow-auto">
        {meta.parse && <ParseSummary parse={meta.parse} />}

        <div>
          <h3 className="mb-1 text-sm font-semibold">Raw OCR transcript</h3>
          <pre className="max-h-48 overflow-auto whitespace-pre-wrap rounded bg-muted p-2 text-xs">
            {meta.ocrText || '(empty)'}
          </pre>
        </div>

        <div>
          <h3 className="mb-1 text-sm font-semibold">
            Clean transcript (ground truth)
          </h3>
          <textarea
            value={clean}
            onChange={(e) => setClean(e.target.value)}
            rows={10}
            className="w-full rounded border border-border bg-background p-2 font-mono text-xs"
          />
          <div className="mt-2 flex items-center gap-2">
            <Button
              size="sm"
              onClick={() => save.mutate(clean)}
              disabled={save.isPending}
            >
              {save.isPending ? 'Saving…' : 'Save clean transcript'}
            </Button>
            {save.isSuccess && (
              <span className="text-xs text-muted-foreground">Saved.</span>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function ParseSummary({ parse }: { parse: ParseResult }) {
  return (
    <div className="rounded border border-border p-3 text-sm">
      <div className="mb-2 flex flex-wrap items-center gap-2">
        <Badge>{parse.decision}</Badge>
        <span className="text-muted-foreground">
          {parse.merchant_name || 'UNKNOWN'}
          {parse.merchant_known ? ' ✓' : ' (unknown)'}
        </span>
        <span className="text-muted-foreground">
          total {parse.total} / sum {parse.computed_total}
          {parse.reconciled ? ' · reconciled' : ' · mismatch'}
        </span>
        <span className="text-muted-foreground">
          conf {parse.confidence.toFixed(2)}
        </span>
      </div>
      {parse.warnings && parse.warnings.length > 0 && (
        <p className="mb-2 text-xs text-amber-600">
          {parse.warnings.join(', ')}
        </p>
      )}
      <table className="w-full text-xs">
        <tbody>
          {parse.items.map((it, i) => (
            <tr key={i} className="border-t border-border/50">
              <td className="py-0.5 pr-2">{it.name}</td>
              <td className="py-0.5 text-right tabular-nums">{it.price}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function Badge({ children }: { children: React.ReactNode }) {
  return (
    <span className="rounded bg-secondary px-2 py-0.5 text-xs font-medium text-secondary-foreground">
      {children}
    </span>
  );
}
