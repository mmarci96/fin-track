import { useEffect, useRef, useState } from 'react';
import { Link } from 'react-router-dom';
import { ChevronLeft, CheckCircle2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { api } from '@/lib/api';
import {
  useDebugImages,
  useDebugImageMeta,
  useSaveCleanText,
  useApproveImage,
  useUploadDebugImage,
  type ParseResult,
} from '@/api/debug';

/**
 * Developer-only tool (dev builds only — see App.tsx). A full-screen, desktop
 * workbench for cleaning OCR transcripts: a capture list on the left, then three
 * full-height columns side by side — the original image, the raw OCR transcript,
 * and the editable clean transcript (ground truth). The transcript panes scroll
 * horizontally and never wrap, so the end of every line stays visible while
 * correcting.
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
    <div className="flex h-full flex-col bg-background">
      {/* Header chrome — this page owns the viewport, so it carries its own nav. */}
      <header className="flex items-center gap-3 border-b border-border px-4 py-2">
        <Link
          to="/"
          className="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
        >
          <ChevronLeft className="h-4 w-4" />
          Back
        </Link>
        <h1 className="text-sm font-semibold">Debug — image ⟷ transcript</h1>
        <div className="ml-auto">
          <Uploader onUploaded={setSelected} />
        </div>
      </header>

      <div className="flex min-h-0 flex-1">
        {/* Capture list */}
        <aside className="w-60 shrink-0 space-y-1 overflow-y-auto border-r border-border p-2">
          {isLoading && <p className="px-2 text-muted-foreground">Loading…</p>}
          {!isLoading && (!images || images.length === 0) && (
            <p className="px-2 py-4 text-sm text-muted-foreground">
              No debug uploads yet — drop receipt images using the button above.
            </p>
          )}
          {images?.map((img) => (
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
              <span className="flex items-center gap-1">
                {img.approved && (
                  <CheckCircle2
                    className={
                      'h-3.5 w-3.5 shrink-0 ' +
                      (selected === img.id
                        ? 'text-primary-foreground'
                        : 'text-success')
                    }
                  />
                )}
                <span className="font-medium">#{img.id}</span>{' '}
                <span className="truncate">
                  {img.originalName || '(unnamed)'}
                </span>
              </span>
              <span className="block text-xs opacity-70">
                {new Date(img.createdAt).toLocaleString()}
                {img.receiptId
                  ? ` · receipt ${img.receiptId}`
                  : ' · no receipt'}
              </span>
            </button>
          ))}
        </aside>

        {/* Detail */}
        <section className="min-w-0 flex-1">
          {selected != null ? (
            <DebugDetail key={selected} id={selected} />
          ) : (
            <p className="p-4 text-muted-foreground">Select a capture.</p>
          )}
        </section>
      </div>
    </div>
  );
}

function DebugDetail({ id }: { id: number }) {
  const { data: meta, isLoading } = useDebugImageMeta(id);
  const save = useSaveCleanText(id);
  const approve = useApproveImage(id);
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

  if (isLoading || !meta)
    return <p className="p-4 text-muted-foreground">Loading…</p>;

  // Live structured result of the cleaned transcript: the freshest Save wins,
  // otherwise the parse stored on the server (component remounts per capture, so
  // a previous capture's Save never leaks in).
  const cleanParse = save.data?.clean_parse ?? meta.cleanParse;
  const learnedAlias = approve.data?.learned_alias;

  // The "current" summary prefers the cleaned re-parse (recounted from the
  // corrections) and falls back to the image parse, so the prominent total/sum
  // reflects edits as soon as they're saved.
  const currentParse = cleanParse ?? meta.parse;

  return (
    <div className="flex h-full flex-col">
      {currentParse && (
        <ParseSummary
          parse={currentParse}
          approved={meta.approved}
          source={cleanParse ? 'clean' : 'image'}
        />
      )}

      {learnedAlias && (
        <div className="border-b border-success/30 bg-success/10 px-4 py-1 text-xs text-success">
          Learned merchant alias:{' '}
          <span className="font-mono">{learnedAlias.alias}</span> →{' '}
          {learnedAlias.merchant}
        </div>
      )}

      {/* Three full-height columns side by side. */}
      <div className="grid min-h-0 flex-1 grid-cols-3 divide-x divide-border">
        {/* Image */}
        <Column title="Image">
          <div className="h-full overflow-auto bg-muted/30 p-2">
            {imgUrl ? (
              <img
                src={imgUrl}
                alt={meta.originalName}
                className="mx-auto w-auto object-contain"
              />
            ) : (
              <p className="p-4 text-sm text-muted-foreground">
                Loading image…
              </p>
            )}
          </div>
        </Column>

        {/* Raw OCR — no wrapping, scroll to see the end of every line. */}
        <Column title="Raw OCR transcript">
          <pre className="h-full overflow-auto whitespace-pre bg-muted/40 p-3 font-mono text-xs leading-relaxed">
            {meta.ocrText || '(empty)'}
          </pre>
        </Column>

        {/* Clean transcript editor — ground truth. */}
        <Column
          title="Clean transcript (ground truth)"
          action={
            <div className="flex items-center gap-2">
              {save.isSuccess && (
                <span className="text-xs text-success">Saved</span>
              )}
              {save.isError && (
                <span className="text-xs text-destructive">Save failed</span>
              )}
              {approve.isError && (
                <span className="text-xs text-destructive">Approve failed</span>
              )}
              <Button
                size="sm"
                variant="secondary"
                onClick={() => save.mutate(clean)}
                disabled={save.isPending}
              >
                {save.isPending ? 'Saving…' : 'Save'}
              </Button>
              {meta.approved ? (
                // Already approved — click to un-approve.
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => approve.mutate(false)}
                  disabled={approve.isPending}
                  className="border-success/40 text-success hover:bg-success/10"
                  title="Approved as ground truth — click to un-approve"
                >
                  <CheckCircle2 className="h-4 w-4" />
                  {approve.isPending ? '…' : 'Approved'}
                </Button>
              ) : (
                <Button
                  size="sm"
                  onClick={() => approve.mutate(true)}
                  disabled={approve.isPending}
                  title="Approve this clean transcript as ground truth"
                >
                  {approve.isPending ? '…' : 'Approve'}
                </Button>
              )}
            </div>
          }
        >
          <textarea
            value={clean}
            onChange={(e) => setClean(e.target.value)}
            wrap="off"
            spellCheck={false}
            className="h-full w-full resize-none overflow-auto whitespace-pre bg-background p-3 font-mono text-xs leading-relaxed outline-none"
          />
        </Column>
      </div>
    </div>
  );
}

/** A titled, full-height pane with an optional header action. */
function Column({
  title,
  action,
  children,
}: {
  title: string;
  action?: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <div className="flex min-h-0 min-w-0 flex-col">
      <div className="flex items-center gap-2 border-b border-border bg-card px-3 py-1.5">
        <h3 className="text-xs font-semibold text-muted-foreground">{title}</h3>
        {action && <div className="ml-auto">{action}</div>}
      </div>
      <div className="min-h-0 flex-1">{children}</div>
    </div>
  );
}

function ParseSummary({
  parse,
  approved,
  source,
}: {
  parse: ParseResult;
  approved: boolean;
  source: 'image' | 'clean';
}) {
  const [open, setOpen] = useState(false);
  return (
    <div className="border-b border-border bg-card text-sm">
      <button
        onClick={() => setOpen((o) => !o)}
        className="flex w-full flex-wrap items-center gap-2 px-4 py-2 text-left"
      >
        {approved && (
          <span className="flex items-center gap-1 rounded bg-success/15 px-2 py-0.5 text-xs font-medium text-success">
            <CheckCircle2 className="h-3.5 w-3.5" /> Approved
          </span>
        )}
        {/* Make it unambiguous whether these numbers are recounted from the
            corrected transcript or still the raw image parse. */}
        <span
          className={
            'rounded px-2 py-0.5 text-xs font-medium ' +
            (source === 'clean'
              ? 'bg-success/15 text-success'
              : 'bg-muted text-muted-foreground')
          }
          title={
            source === 'clean'
              ? 'Recounted from the corrected transcript'
              : 'Read from the image — save a correction to recount'
          }
        >
          {source === 'clean' ? 'corrected' : 'from image'}
        </span>
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
        <span className="ml-auto text-xs text-muted-foreground">
          {open ? 'hide items ▲' : `${parse.items.length} items ▼`}
        </span>
      </button>
      {open && (
        <div className="px-4 pb-3">
          {parse.warnings && parse.warnings.length > 0 && (
            <p className="mb-2 text-xs text-warning">
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
      )}
    </div>
  );
}

function Uploader({ onUploaded }: { onUploaded: (id: number) => void }) {
  const upload = useUploadDebugImage();
  const inputRef = useRef<HTMLInputElement>(null);
  const [status, setStatus] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

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
    <div className="flex items-center gap-2">
      {status && (
        <span className="text-xs text-muted-foreground">{status}</span>
      )}
      {error && <span className="text-xs text-destructive">{error}</span>}
      <Button
        size="sm"
        variant="secondary"
        onClick={() => inputRef.current?.click()}
        disabled={upload.isPending}
      >
        {upload.isPending ? 'Uploading…' : 'Upload images'}
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
