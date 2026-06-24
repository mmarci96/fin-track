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
}

interface RawMeta extends RawSummary {
  ocr_text: string;
  clean_text: string | null;
  parse: ParseResult | null;
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
}

export interface DebugImageMeta extends DebugImageSummary {
  ocrText: string;
  cleanText: string | null;
  parse: ParseResult | null;
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
      } satisfies DebugImageMeta;
    },
  });
}

export function useSaveCleanText(id: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (cleanText: string) =>
      api.put<{ ok: boolean }>(`/receipt-images/${id}/clean`, {
        clean_text: cleanText,
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: keys.meta(id) }),
  });
}

function mapSummary(r: RawSummary): DebugImageSummary {
  return {
    id: r.id,
    receiptId: r.receipt_id,
    originalName: r.original_name,
    contentType: r.content_type,
    createdAt: r.created_at,
  };
}
