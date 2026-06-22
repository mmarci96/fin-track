// Thin fetch wrapper around the Go API. It owns two responsibilities:
//  1. attach the auth identity to every request (today: X-User-ID header),
//  2. unwrap the JSON envelope and surface typed errors.
//
// This is the ONLY module that talks to fetch. Swapping the mock X-User-ID auth
// for real bearer-token auth later is a change confined here + AuthContext.

export const USER_ID_STORAGE_KEY = 'ft_user_id';

const BASE_URL = '/api';

export class ApiError extends Error {
  status: number;
  code?: string;
  payload?: unknown;

  constructor(
    status: number,
    message: string,
    code?: string,
    payload?: unknown,
  ) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
    this.payload = payload;
  }
}

function authHeaders(): Record<string, string> {
  const userId = localStorage.getItem(USER_ID_STORAGE_KEY);
  return userId ? { 'X-User-ID': userId } : {};
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(BASE_URL + path, {
    ...init,
    headers: { ...authHeaders(), ...(init?.headers ?? {}) },
  });

  const isJson = res.headers.get('content-type')?.includes('application/json');
  const body = isJson ? await res.json() : null;

  if (!res.ok) {
    const message =
      (body && (body.error as string)) || res.statusText || 'request failed';
    throw new ApiError(res.status, message, body?.code, body);
  }

  return body as T;
}

export const api = {
  get: <T>(path: string) => request<T>(path),

  post: <T>(path: string, data: unknown) =>
    request<T>(path, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    }),

  put: <T>(path: string, data: unknown) =>
    request<T>(path, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    }),

  del: <T>(path: string) => request<T>(path, { method: 'DELETE' }),

  /** Multipart upload — do not set Content-Type, the browser adds the boundary. */
  postForm: <T>(path: string, form: FormData) =>
    request<T>(path, { method: 'POST', body: form }),
};
