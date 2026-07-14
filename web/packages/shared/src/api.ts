// Tiny fetch wrapper with localStorage token auth (localhost demo trade-off;
// see README for the production cookie/CSRF caveat).

export const API_URL = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080';

export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

export interface Api {
  get<T>(path: string): Promise<T>;
  post<T>(path: string, body?: unknown): Promise<T>;
  put<T>(path: string, body?: unknown): Promise<T>;
  patch<T>(path: string, body?: unknown): Promise<T>;
  del<T>(path: string): Promise<T>;
  getToken(): string | null;
  setToken(token: string | null): void;
  onUnauthorized?: () => void;
}

export function createApi(tokenKey: string): Api {
  const getToken = () =>
    typeof window === 'undefined' ? null : window.localStorage.getItem(tokenKey);

  const api: Api = {
    getToken,
    setToken(token) {
      if (typeof window === 'undefined') return;
      if (token) window.localStorage.setItem(tokenKey, token);
      else window.localStorage.removeItem(tokenKey);
    },
    get: (path) => request('GET', path),
    post: (path, body) => request('POST', path, body),
    put: (path, body) => request('PUT', path, body),
    patch: (path, body) => request('PATCH', path, body),
    del: (path) => request('DELETE', path),
  };

  async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
    const headers: Record<string, string> = {};
    if (body !== undefined) headers['Content-Type'] = 'application/json';
    const token = getToken();
    if (token) headers.Authorization = `Bearer ${token}`;

    const res = await fetch(`${API_URL}${path}`, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
    });
    if (res.status === 401) {
      api.setToken(null);
      api.onUnauthorized?.();
    }
    if (!res.ok) {
      let message = `Request failed (${res.status})`;
      try {
        const data = await res.json();
        if (data && typeof data.error === 'string') message = data.error;
      } catch {
        /* keep fallback */
      }
      throw new ApiError(res.status, message);
    }
    return (await res.json()) as T;
  }

  return api;
}
