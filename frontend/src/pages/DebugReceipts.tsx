import { useEffect, useRef, useState } from 'react';
import { Button } from '@/components/ui/button';
import { api } from '@/lib/api';
import {
  useDebugImages,
  useDebugImageMeta,
  useSaveCleanText,
  useUploadDebugImage,
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

      <Uploader onUploaded={setSelected} />

      {isLoading && <p className="text-muted-foreground">Loading…</p>}
      {!isLoading && (!images || images.length === 0) && (
        <p className="text-muted-foreground">
          No debug uploads yet — drop some receipt images above to get started.
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
  const [imgUrl, setImgUrl] = useState<string | null>(null);

  // Seed the editor from the stored clean text, falling back to the raw OCR so
  // correcting is edit-in-place rather than retyping.
  useEffect(() => {
    if (meta) setClean(meta.cleanText ?? meta.ocrText ?? '');
  }, [meta]);

  // Load the image as an authenticated blob (a plain <img> can't send the auth
  // header). Revoke the object URL on change/unmount to avoid leaks.
  useEffect(() => {
    let url: string | null = null;
    let cancelled = false;
    api
      .getBlob(`/receipt-images/${id}`)
      .then((blob) => {
        if (cancelled) return;
        url = URL.createObjectURL(blob);
        setImgUrl(url);
      })
      .catch(() => !cancelled && setImgUrl(null));
    return () => {
      cancelled = true;
      if (url) URL.revokeObjectURL(url);
      setImgUrl(null);
    };
  }, [id]);

  if (isLoading || !meta) return <p className="text-muted-foreground">Loading…</p>;

  return (
    <div className="grid grid-cols-2 gap-4">
      {/* Left: the image */}
      <div className="overflow-auto rounded border border-border bg-muted/30 p-2">
        {imgUrl ? (
          <img
            src={imgUrl}
            alt={meta.originalName}
            className="mx-auto max-h-[78vh] w-auto object-contain"
          />
        ) : (
          <p className="p-4 text-sm text-muted-foreground">Loading image…</p>
        )}
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

function Uploader({ onUploaded }: { onUploaded: (id: number) => void }) {
  const upload = useUploadDebugImage();
  const inputRef = useRef<HTMLInputElement>(null);
  const [status, setStatus] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [dragOver, setDragOver] = useState(false);

  async function handleFiles(files: FileList | null) {
    const list = Array.from(files ?? []).filter((f) =>
      f.type.startsWith('image/'),
    );
    if (list.length === 0) return;
    setError(null);
    let lastId = 0;
    for (let i = 0; i < list.length; i++) {
      setStatus(`Uploading ${i + 1}/${list.length}: ${list[i].name}…`);
      try {
        // Sequential: OCR is heavy, no point hammering it in parallel.
        const r = await upload.mutateAsync(list[i]);
        lastId = r.imageId;
      } catch (e) {
        setError(`Failed on ${list[i].name}: ${(e as Error).message}`);
      }
    }
    setStatus(null);
    if (lastId) onUploaded(lastId);
  }

  return (
    <div
      onDragOver={(e) => {
        e.preventDefault();
        setDragOver(true);
      }}
      onDragLeave={() => setDragOver(false)}
      onDrop={(e) => {
        e.preventDefault();
        setDragOver(false);
        void handleFiles(e.dataTransfer.files);
      }}
      className={
        'mb-4 flex flex-col items-center gap-2 rounded border-2 border-dashed p-4 text-sm transition-colors ' +
        (dragOver ? 'border-primary bg-primary/5' : 'border-border')
      }
    >
      <p className="text-muted-foreground">
        Drop receipt images here, or
      </p>
      <Button
        size="sm"
        variant="secondary"
        onClick={() => inputRef.current?.click()}
        disabled={upload.isPending}
      >
        {upload.isPending ? 'Uploading…' : 'Choose images'}
      </Button>
      <input
        ref={inputRef}
        type="file"
        accept="image/*"
        multiple
        className="hidden"
        onChange={(e) => {
          void handleFiles(e.target.files);
          e.target.value = ''; // allow re-selecting the same file
        }}
      />
      {status && <p className="text-muted-foreground">{status}</p>}
      {error && <p className="text-destructive">{error}</p>}
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
