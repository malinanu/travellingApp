// Tiny fetch wrapper with localStorage token auth (localhost demo trade-off;
// see README for the production cookie/CSRF caveat).

export const API_URL = process.env.NEXT_PUBLIC_API_URL || '';

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
   * Stores the access token, plus the session's refresh token when the
   * server sent one (sign-in responses do; password change/2FA disable keep
   * the existing one). remember picks localStorage (survives closing the
   * browser) vs sessionStorage (cleared on tab close) — a real "Remember
   * me" checkbox, not just cosmetic. Omitted, it keeps wherever the session
   * already lives (localStorage for a new one). null clears both tokens.
   */
  setToken(token: string | null, remember?: boolean, refreshToken?: string): void;
  /**
   * Signs this browser out: ends the server session (best effort) and
   * clears the stored tokens. Pass refreshToken for a session that was
   * started but never stored (e.g. a sign-in rejected for the wrong role).
   */
  logout(refreshToken?: string): void;
  onUnauthorized?: () => void;
}

type RefreshResult = 'refreshed' | 'rejected' | 'failed';

/** client is sent as X-Spotto-Client, so the server applies the web
 * session limits and can label the session for admins. */
export function createApi(tokenKey: string, client: 'admin' | 'vendor' = 'vendor'): Api {
  const refreshKey = `${tokenKey}_refresh`;
  const read = (key: string) =>
    typeof window === 'undefined' ? null : window.localStorage.getItem(key) ?? window.sessionStorage.getItem(key);
  const getToken = () => read(tokenKey);
  const getRefresh = () => read(refreshKey);
  const clientHeaders = { 'X-Spotto-Client': client };

  const api: Api = {
    getToken,
    setToken(token, remember, refreshToken) {
      if (typeof window === 'undefined') return;
      const refresh = refreshToken ?? (token ? getRefresh() : null);
      const keep = remember ?? window.sessionStorage.getItem(tokenKey) === null;
      for (const store of [window.localStorage, window.sessionStorage]) {
        store.removeItem(tokenKey);
        store.removeItem(refreshKey);
      }
      if (!token) return;
      const store = keep ? window.localStorage : window.sessionStorage;
      store.setItem(tokenKey, token);
      if (refresh) store.setItem(refreshKey, refresh);
    },
    logout(refreshToken) {
      const refresh = refreshToken ?? getRefresh();
      if (refresh) {
        // keepalive: the request still goes out if the page navigates away.
        fetch(`${API_URL}/auth/logout`, {
          method: 'POST',
          keepalive: true,
          headers: { 'Content-Type': 'application/json', ...clientHeaders },
          body: JSON.stringify({ refreshToken: refresh }),
        }).catch(() => {});
      }
      if (!refreshToken) api.setToken(null);
    },
    get: (path) => request('GET', path),
    post: (path, body) => request('POST', path, body),
    put: (path, body) => request('PUT', path, body),
    patch: (path, body) => request('PATCH', path, body),
    del: (path) => request('DELETE', path),
  };

  // Single-flight: requests that hit 401 together share one refresh.
  let refreshing: Promise<RefreshResult> | null = null;

  async function refreshSession(sentToken: string | null): Promise<RefreshResult> {
    // Another request (or tab) already refreshed since this one was sent.
    const current = getToken();
    if (sentToken && current && current !== sentToken) return 'refreshed';
    refreshing ??= doRefresh().finally(() => { refreshing = null; });
    return refreshing;
  }

  async function doRefresh(): Promise<RefreshResult> {
    const used = getRefresh();
    if (!used) return 'rejected';
    let res: Response;
    try {
      res = await fetch(`${API_URL}/auth/refresh`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...clientHeaders },
        body: JSON.stringify({ refreshToken: used }),
      });
    } catch {
      return 'failed';
    }
    if (res.ok) {
      const data = (await res.json()) as { token: string; refreshToken: string };
      api.setToken(data.token, undefined, data.refreshToken);
      return 'refreshed';
    }
    // Another tab sharing this storage rotated the token first.
    if (getRefresh() !== used && getToken()) return 'refreshed';
    return res.status === 401 || res.status === 400 ? 'rejected' : 'failed';
  }

  // A token from before server sessions existed has no refresh token; trade
  // it for a session once, so this deploy doesn't sign anyone out.
  let upgradeChecked = false;
  async function upgradeLegacySession() {
    upgradeChecked = true;
    const token = getToken();
    if (!token || getRefresh()) return;
    try {
      const res = await fetch(`${API_URL}/auth/session/upgrade`, {
        method: 'POST',
        headers: { ...clientHeaders, Authorization: `Bearer ${token}` },
      });
      if (!res.ok) return;
      const data = (await res.json()) as { token: string; refreshToken: string };
      api.setToken(data.token, undefined, data.refreshToken);
    } catch {
      /* offline: try again on the next page load */
    }
  }

  async function request<T>(method: string, path: string, body?: unknown, retried = false): Promise<T> {
    if (!upgradeChecked && !path.startsWith('/auth/')) await upgradeLegacySession();
    const headers: Record<string, string> = { ...clientHeaders };
    if (body !== undefined) headers['Content-Type'] = 'application/json';
    const token = getToken();
    if (token) headers.Authorization = `Bearer ${token}`;

    let res: Response;
    try {
      res = await fetch(`${API_URL}${path}`, {
        method,
        headers,
        body: body === undefined ? undefined : JSON.stringify(body),
      });
    } catch {
      // fetch() rejects with a bare "TypeError: Failed to fetch" when no
      // response arrives at all (offline, server down, or a CORS refusal);
      // say that in words instead of showing the browser's string.
      throw new ApiError(0, 'Can’t reach Spotto right now. Check your connection and try again.');
    }
    if (res.status === 401 && !retried && !path.startsWith('/auth/')) {
      const result = await refreshSession(token);
      if (result === 'refreshed') return request<T>(method, path, body, true);
      // Couldn't reach the server to refresh: keep the session and fail
      // this request like any other network error.
      if (result === 'failed') throw new ApiError(res.status, 'Your session could not be refreshed. Check your connection and try again.');
    }
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
