import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '@/lib/api';

// The Go merchant model serializes as { ID, Name } and handlers wrap payloads
// in { "result": ... }. Normalize to clean camelCase app types at the boundary,
// mirroring src/api/receipts.ts.

interface RawMerchant {
  ID: number;
  Name: string;
}

export interface Merchant {
  id: number;
  name: string;
}

function mapMerchant(m: RawMerchant): Merchant {
  return { id: m.ID, name: m.Name };
}

const keys = {
  all: ['merchants'] as const,
};

export function useMerchants() {
  return useQuery({
    queryKey: keys.all,
    queryFn: async () => {
      const body = await api.get<{ result: RawMerchant[] | null }>('/merchants');
      return (body.result ?? []).map(mapMerchant);
    },
  });
}

export function useCreateMerchant() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (name: string) => {
      const body = await api.post<{ result: RawMerchant }>('/merchants', {
        name,
      });
      return mapMerchant(body.result);
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: keys.all }),
  });
}
