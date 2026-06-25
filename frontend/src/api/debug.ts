import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '@/lib/api';

// Developer-only API for the persisted "debug" uploads: the original image plus
// its OCR transcript and parser output. Backs the side-by-side viewer
// (pages/DebugReceipts). Mirrors the boundary-normalization style of
// src/api/receipts.ts.

// --- Wire format -----------------------------------------------------------

interface RawSummary {
  id: number;
  receipt_id: number | null;
  original_name: string;
  content_type: string;
  created_at: string;
  // Whether a human-approved clean transcript exists. Optional so the UI keeps
  // working against backends that don't report it yet.
  approved?: boolean;
}

interface RawMeta extends RawSummary {
  ocr_text: string;
  clean_text: string | null;
  parse: ParseResult | null;
  // Structured result of re-parsing clean_text (the ground-truth parse).
  clean_parse: ParseResult | null;
}

// Subset of the Go receipt.Result we render.
export interface ParseResult {
  merchant_name: string;
  merchant_known: boolean;
  items: { name: string; price: number }[];
  total: number;
  computed_total: number;
  reconciled: boolean;
  confidence: number;
  decision: string;
  warnings?: string[] | null;
}

// --- App types -------------------------------------------------------------

export interface DebugImageSummary {
  id: number;
  receiptId: number | null;
  originalName: string;
  contentType: string;
  createdAt: string;
  /** A human-approved clean transcript exists for this capture. */
  approved: boolean;
}

export interface DebugImageMeta extends DebugImageSummary {
  ocrText: string;
  cleanText: string | null;
  parse: ParseResult | null;
  /** Re-parse of the cleaned transcript — the structured ground-truth output. */
  cleanParse: ParseResult | null;
}

/** Source URL for the stored image; auth rides on the session cookie / dev default user. */
export function debugImageUrl(id: number): string {
  return `/api/receipt-images/${id}`;
}

const keys = {
  all: ['debug-images'] as const,
  meta: (id: number) => ['debug-images', id, 'meta'] as const,
};

export function useDebugImages() {
  return useQuery({
    queryKey: keys.all,
    queryFn: async () => {
      const body =
        await api.get<{ result: RawSummary[] | null }>('/receipt-images');
      return (body.result ?? []).map(mapSummary);
    },
  });
}

export function useDebugImageMeta(id: number | null) {
  return useQuery({
    queryKey: id ? keys.meta(id) : ['debug-images', 'none'],
    enabled: id != null && id > 0,
    queryFn: async () => {
      const raw = await api.get<RawMeta>(`/receipt-images/${id}/meta`);
      return {
        ...mapSummary(raw),
        ocrText: raw.ocr_text,
        cleanText: raw.clean_text,
        parse: raw.parse,
        cleanParse: raw.clean_parse,
      } satisfies DebugImageMeta;
    },
  });
}

export interface DebugUploadResult {
  imageId: number;
  decision: string;
}

/** Upload an image to the debug endpoint; the backend persists it for review. */
export function useUploadDebugImage() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (file: File): Promise<DebugUploadResult> => {
      const form = new FormData();
      form.append('image', file, file.name);
      const body = await api.postForm<{ image_id: number; decision: string }>(
        '/receipts/image/debug',
        form,
      );
      return { imageId: body.image_id, decision: body.decision };
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: keys.all }),
  });
}

/**
 * Save the corrected transcript. The backend re-parses it and returns the
 * structured result so the viewer can show the extracted items live.
 */
export function useSaveCleanText(id: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (cleanText: string) =>
      api.put<{ ok: boolean; clean_parse: ParseResult }>(
        `/receipt-images/${id}/clean`,
        { clean_text: cleanText },
      ),
    onSuccess: () => qc.invalidateQueries({ queryKey: keys.meta(id) }),
  });
}

/** A merchant alias the flywheel learned when a capture was approved. */
export interface LearnedAlias {
  alias: string;
  merchant: string;
}

/** Approve (or un-approve) a capture's clean transcript as ground truth. */
export function useApproveImage(id: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (approved: boolean) =>
      api.put<{ ok: boolean; approved: boolean; learned_alias?: LearnedAlias }>(
        `/receipt-images/${id}/approve`,
        { approved },
      ),
    // Refresh both the detail (badge) and the list (its ✓ marker).
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: keys.meta(id) });
      void qc.invalidateQueries({ queryKey: keys.all });
    },
  });
}

function mapSummary(r: RawSummary): DebugImageSummary {
  return {
    id: r.id,
    receiptId: r.receipt_id,
    originalName: r.original_name,
    contentType: r.content_type,
    createdAt: r.created_at,
    approved: r.approved ?? false,
  };
}
