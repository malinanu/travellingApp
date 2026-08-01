// Tiny fetch wrapper with localStorage token auth (localhost demo trade-off;
// see README for the production cookie/CSRF caveat).

export const API_URL = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080';

// The R2 bucket behind media uploads is private — the server hands back
// app-relative paths like `/media/raw/<key>?token=...` (see
// handleMediaRaw/signMediaPath server-side) rather than an absolute R2/CDN
// URL, so every <img>/<video>/<a> render site needs to resolve against our
// own API host first. Anything already absolute (http/https) passes
// through unchanged.
export function resolveMediaUrl(url: string): string {
  return url.startsWith('/') ? `${API_URL}${url}` : url;
}

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
  /**
   * remember (default true) picks localStorage (survives closing the
   * browser) vs sessionStorage (cleared on tab close) — a real "Remember
   * me" checkbox, not just cosmetic. Every existing call site that omits
   * it keeps today's always-localStorage behavior.
   */
  setToken(token: string | null, remember?: boolean): void;
  onUnauthorized?: () => void;
}

export function createApi(tokenKey: string): Api {
  const getToken = () =>
    typeof window === 'undefined' ? null : window.localStorage.getItem(tokenKey) ?? window.sessionStorage.getItem(tokenKey);

  const api: Api = {
    getToken,
    setToken(token, remember = true) {
      if (typeof window === 'undefined') return;
      window.localStorage.removeItem(tokenKey);
      window.sessionStorage.removeItem(tokenKey);
      if (!token) return;
      (remember ? window.localStorage : window.sessionStorage).setItem(tokenKey, token);
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
