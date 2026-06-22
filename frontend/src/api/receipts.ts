import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '@/lib/api';

// --- Wire format -----------------------------------------------------------
// The Go models are serialized with default (PascalCase) field names and the
// handlers wrap payloads in { "result": ... }. We normalize to clean camelCase
// app types right here at the boundary so no UI code deals with the wire shape.

interface RawProduct {
  ID: number;
  Name: string;
  Price: number;
}
interface RawReceipt {
  ID: number;
  UserID: number;
  MerchantID: number;
  Merchant: { ID: number; Name: string };
  Products: RawProduct[] | null;
  TotalAmount: number;
}

// --- App types -------------------------------------------------------------

export interface Product {
  id: number;
  name: string;
  price: number; // integer minor units
}
export interface Receipt {
  id: number;
  merchantId: number;
  merchant: string;
  total: number; // integer minor units
  products: Product[];
}

export type Decision = 'accepted' | 'best_effort' | 'reject';

export interface UploadResult {
  receipt: Receipt;
  decision: Decision;
  warnings: string[];
  detected: { total: number; reconciled: boolean; merchantKnown: boolean };
}

function mapReceipt(r: RawReceipt): Receipt {
  return {
    id: r.ID,
    merchantId: r.MerchantID,
    merchant: r.Merchant?.Name ?? '',
    total: r.TotalAmount,
    products: (r.Products ?? []).map((p) => ({
      id: p.ID,
      name: p.Name,
      price: p.Price,
    })),
  };
}

// --- Request payloads (these DO use snake_case json tags on the backend) ----

export interface ReceiptUpdateInput {
  total_amount: number;
  products: { name: string; price: number }[];
}

const keys = {
  all: ['receipts'] as const,
  detail: (id: number) => ['receipts', id] as const,
};

export function useReceipts() {
  return useQuery({
    queryKey: keys.all,
    queryFn: async () => {
      const body = await api.get<{ result: RawReceipt[] | null }>('/receipts');
      return (body.result ?? []).map(mapReceipt);
    },
  });
}

export function useReceipt(id: number) {
  return useQuery({
    queryKey: keys.detail(id),
    queryFn: async () => {
      const body = await api.get<{ result: RawReceipt }>(`/receipts/${id}`);
      return mapReceipt(body.result);
    },
    enabled: Number.isFinite(id) && id > 0,
  });
}

export function useUploadImage() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (blob: Blob): Promise<UploadResult> => {
      const form = new FormData();
      form.append('image', blob, 'receipt.jpg');
      const body = await api.postForm<{
        result: RawReceipt;
        decision: Decision;
        warnings: string[] | null;
        detected: {
          total: number;
          reconciled: boolean;
          merchant_known: boolean;
        };
      }>('/receipts/image', form);
      return {
        receipt: mapReceipt(body.result),
        decision: body.decision,
        warnings: body.warnings ?? [],
        detected: {
          total: body.detected.total,
          reconciled: body.detected.reconciled,
          merchantKnown: body.detected.merchant_known,
        },
      };
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: keys.all }),
  });
}

export function useUpdateReceipt(id: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: ReceiptUpdateInput) => {
      const body = await api.put<{ result: RawReceipt }>(
        `/receipts/${id}`,
        input,
      );
      return mapReceipt(body.result);
    },
    onSuccess: (updated) => {
      qc.setQueryData(keys.detail(id), updated);
      qc.invalidateQueries({ queryKey: keys.all });
    },
  });
}

export function useDeleteReceipt() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => api.del<void>(`/receipts/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: keys.all }),
  });
}
