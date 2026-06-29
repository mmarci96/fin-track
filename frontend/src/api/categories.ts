import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api';

// Categories are global reference data. The Go model serializes as { ID, Name }
// and handlers wrap payloads in { "result": ... }; normalize to camelCase here,
// mirroring src/api/merchants.ts.

interface RawCategory {
  ID: number;
  Name: string;
}

export interface Category {
  id: number;
  name: string;
}

function mapCategory(c: RawCategory): Category {
  return { id: c.ID, name: c.Name };
}

const keys = {
  all: ['categories'] as const,
};

export function useCategories() {
  return useQuery({
    queryKey: keys.all,
    queryFn: async () => {
      const body = await api.get<{ result: RawCategory[] | null }>(
        '/categories',
      );
      return (body.result ?? []).map(mapCategory);
    },
  });
}
