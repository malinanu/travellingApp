'use client';

// Admin command-center shell — 232px dark rail + command bar with KPI
// strip, theme toggle (light/dark CSS vars), real ⌘K search, and a real
// notification bell. Port of AdminApp.jsx.

import { useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Activity, Bell, CreditCard, Flag, Headphones, History, KeyRound, Map, MapPin, Moon, Radio, Search, ShieldAlert, Shapes, Sparkles, Store, Sun, UserCheck, Users, LucideIcon,
} from 'lucide-react';
import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { useEffect, useRef, useState } from 'react';

import type { AdminKPIs, AdminNotification, AdminSearchResults, User } from '@spotto/shared';
import { api } from '../lib/api';

const NAV: { href: string; label: string; icon: LucideIcon; live?: boolean; superAdminOnly?: boolean }[] = [
  { href: '/queue', label: 'Approval queue', icon: UserCheck },
  { href: '/vendors', label: 'Vendors', icon: Store },
  { href: '/users', label: 'Users', icon: Users },
  { href: '/quests', label: 'Quests', icon: Activity },
  { href: '/content', label: 'Content monitor', icon: Radio, live: true },
  { href: '/geo', label: 'Geo-analytics', icon: Map },
  { href: '/subscriptions', label: 'Subscriptions', icon: CreditCard },
  { href: '/audio', label: 'Audio guides', icon: Headphones },
  { href: '/local-guide', label: 'Local Guide', icon: Sparkles },
  { href: '/categories', label: 'Categories', icon: Shapes },
  { href: '/access-grants', label: 'Access grants', icon: KeyRound },
  { href: '/audit-log', label: 'Audit log', icon: History, superAdminOnly: true },
];

// Reversed small-size cut of the brand symbol (brand/logo/spotto-symbol-small-reversed.svg),
// for the always-dark sidebar and login panel.
export function SpottoMark({ size = 26 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 32 32" fill="none" role="img" aria-label="Spotto">
      <path d="M5.42 17.19A10.58 10.58 0 0 1 26.58 17.19Z" fill="#F6B81E" />
      <path d="M3.3 19.05H28.7" stroke="#FBF6ED" strokeWidth="3.22" strokeLinecap="round" />
      <path d="M8.59 23.54H23.41" stroke="#0E8C8A" strokeWidth="3.22" strokeLinecap="round" />
    </svg>
  );
}

export function useMe() {
  return useQuery<User>({
    queryKey: ['admin-me'],
    queryFn: () => api.get<User>('/me'),
    enabled: typeof window !== 'undefined' && !!api.getToken(),
  });
}

function useKpis() {
  return useQuery<AdminKPIs>({
    queryKey: ['admin-kpis'],
    queryFn: () => api.get<AdminKPIs>('/admin/kpis'),
    refetchInterval: 10000,
  });
}

const NOTIF_ICONS: Record<string, LucideIcon> = { UserCheck, Flag, ShieldAlert };
// Each bucket id from Store.AdminNotifications routes to the page where an
// admin actually resolves it.
const NOTIF_TARGET: Record<string, string> = {
  'pending-vendors': '/queue',
  'flagged-content': '/content',
  'bad-review-vendors': '/geo',
};

function toneStyle(tone: string) {
  if (tone === 'sun') return { background: 'rgba(246,184,30,0.16)', color: '#c98a00' };
  if (tone === 'muted') return { background: 'rgba(255,255,255,0.06)', color: 'var(--muted)' };
  return { background: 'var(--accent-tint)', color: 'var(--accent)' };
}

function AdminNotificationBell() {
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  const qc = useQueryClient();
  const { data: notifs = [] } = useQuery<AdminNotification[]>({
    queryKey: ['admin-notifications'],
    queryFn: () => api.get<AdminNotification[]>('/admin/notifications'),
    refetchInterval: 15000,
  });
  const unread = notifs.filter((n) => n.unread).length;

  useEffect(() => {
    const onDoc = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener('mousedown', onDoc);
    return () => document.removeEventListener('mousedown', onDoc);
  }, []);

  const markAll = async () => {
    await api.post('/admin/notifications/read');
    qc.invalidateQueries({ queryKey: ['admin-notifications'] });
  };

  return (
    <div className="relative" ref={ref}>
      <button
        aria-label="Notifications"
        onClick={() => setOpen((o) => !o)}
        className="relative w-9 h-9 rounded-lg flex items-center justify-center border border-line bg-bg text-ink-soft"
      >
        <Bell size={16} />
        {unread > 0 && (
          <span className="absolute top-1.5 right-2 w-2 h-2 rounded-full" style={{ background: 'var(--accent)', boxShadow: '0 0 0 2px var(--bg)' }} />
        )}
      </button>
      {open && (
        <div className="absolute right-0 mt-2 w-[340px] rounded-xl overflow-hidden z-40"
          style={{ background: 'var(--surface)', border: '1px solid var(--line)', boxShadow: '0 24px 50px -20px rgba(0,0,0,0.35)' }}>
          <div className="flex items-center gap-2 px-4 py-3 border-b border-line">
            <span className="font-bold text-[13.5px]">Notifications</span>
            {unread > 0 && (
              <button onClick={markAll} className="ml-auto text-[12px] font-bold" style={{ color: 'var(--accent)' }}>Mark all read</button>
            )}
          </div>
          <div className="max-h-[360px] overflow-y-auto">
            {notifs.length === 0 && (
              <p className="px-4 py-6 text-[13px] text-muted font-semibold">Nothing needs attention right now.</p>
            )}
            {notifs.map((n, i) => {
              const NIcon = NOTIF_ICONS[n.icon] ?? Bell;
              return (
                <button key={n.id} onClick={() => { setOpen(false); router.push(NOTIF_TARGET[n.id] ?? '/queue'); }}
                  className="w-full flex items-start gap-3 px-4 py-3 text-left transition-colors"
                  style={{ borderTop: i ? '1px solid var(--line)' : 'none' }}>
                  <span className="w-9 h-9 rounded-lg flex items-center justify-center shrink-0" style={toneStyle(n.tone)}>
                    <NIcon size={16} />
                  </span>
                  <span className="flex-1 min-w-0">
                    <span className="block text-[13px] font-bold leading-snug">{n.title}</span>
                    <span className="block text-[12px] text-muted font-medium mt-0.5">{n.sub}</span>
                    <span className="block text-[10.5px] text-muted font-semibold mt-1">{n.time}</span>
                  </span>
                  {n.unread && <span className="w-2 h-2 rounded-full shrink-0 mt-1.5" style={{ background: 'var(--accent)' }} />}
                </button>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}

function useTheme() {
  const [theme, setTheme] = useState<'light' | 'dark'>('light');
  useEffect(() => {
    const saved = (localStorage.getItem('spotto_admin_theme') as 'light' | 'dark') || 'light';
    setTheme(saved);
  }, []);
  const toggle = () => {
    const next = theme === 'light' ? 'dark' : 'light';
    setTheme(next);
    document.documentElement.dataset.theme = next;
    localStorage.setItem('spotto_admin_theme', next);
  };
  return { theme, toggle };
}

function Rail({ pending }: { pending: number }) {
  const pathname = usePathname();
  const { data: me } = useMe();
  const visibleNav = NAV.filter((n) => !n.superAdminOnly || me?.isSuperAdmin);
  return (
    <aside className="w-[232px] shrink-0 flex flex-col" style={{ background: '#103d3a', borderRight: '1px solid rgba(255,255,255,0.06)' }}>
      <div className="flex items-center gap-2.5 px-5 h-[58px]" style={{ borderBottom: '1px solid rgba(255,255,255,0.06)' }}>
        <SpottoMark size={26} />
        <div className="leading-none">
          <p className="font-bold text-[16px] tracking-tight text-white">Spotto</p>
          <p className="text-[9.5px] font-bold uppercase tracking-[0.18em] mt-0.5" style={{ color: 'var(--accent)' }}>
            Command Center
          </p>
        </div>
      </div>

      <nav className="flex-1 px-3 py-4 space-y-1">
        <p className="px-3 pb-2 text-[10px] font-bold uppercase tracking-[0.15em]" style={{ color: 'rgba(255,255,255,0.3)' }}>
          Operations
        </p>
        {visibleNav.map((n) => {
          const on = pathname.startsWith(n.href);
          const Icon = n.icon;
          return (
            <Link
              key={n.href}
              href={n.href}
              className="w-full flex items-center gap-3 rounded-lg px-3 py-2.5 transition-all"
              style={{
                background: on ? 'rgba(255,255,255,0.07)' : 'transparent',
                color: on ? '#fff' : 'rgba(255,255,255,0.55)',
                boxShadow: on ? 'inset 2px 0 0 var(--accent)' : 'none',
              }}
            >
              <Icon size={18} strokeWidth={on ? 2.4 : 2} style={{ color: on ? 'var(--accent)' : 'inherit' }} />
              <span className="font-semibold text-[13.5px]">{n.label}</span>
              {n.href === '/queue' && pending > 0 && (
                <span className="ml-auto rounded-full px-1.5 py-0.5 text-[10px] font-bold text-white" style={{ background: 'var(--accent)' }}>
                  {pending}
                </span>
              )}
              {n.live && (
                <span className="ml-auto w-2 h-2 rounded-full" style={{ background: '#16a34a', boxShadow: '0 0 0 3px rgba(22,163,74,0.25)' }} />
              )}
            </Link>
          );
        })}
      </nav>

      <div className="p-3">
        <div className="rounded-lg p-3" style={{ background: 'rgba(255,255,255,0.04)' }}>
          <p className="flex items-center gap-2 text-[11px] font-bold" style={{ color: '#16a34a' }}>
            <Activity size={13} /> All systems operational
          </p>
          <p className="text-[10.5px] font-medium mt-1" style={{ color: 'rgba(255,255,255,0.4)' }}>
            API live · Postgres connected
          </p>
        </div>
      </div>
    </aside>
  );
}

function resultRoute(kind: 'vendor' | 'feedItem' | 'subscription', status?: string, id?: string): string {
  // Non-pending vendors used to route to /geo, which silently drops any
  // vendor without coordinates — /vendors always has a row for them.
  let route = '/subscriptions';
  if (kind === 'vendor') route = status === 'pending' ? '/queue' : '/vendors';
  else if (kind === 'feedItem') route = '/content';
  
  return id ? `${route}?id=${id}` : route;
}

function CommandSearch() {
  const router = useRouter();
  const [q, setQ] = useState('');
  const [debouncedQ, setDebouncedQ] = useState('');
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    const t = setTimeout(() => setDebouncedQ(q), 300);
    return () => clearTimeout(t);
  }, [q]);

  const enabled = debouncedQ.trim().length >= 2;
  const { data: results } = useQuery<AdminSearchResults>({
    queryKey: ['admin-search', debouncedQ],
    queryFn: () => api.get<AdminSearchResults>(`/admin/search?q=${encodeURIComponent(debouncedQ)}`),
    enabled,
  });

  useEffect(() => {
    const onDoc = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        inputRef.current?.focus();
        setOpen(true);
      }
    };
    document.addEventListener('mousedown', onDoc);
    document.addEventListener('keydown', onKey);
    return () => {
      document.removeEventListener('mousedown', onDoc);
      document.removeEventListener('keydown', onKey);
    };
  }, []);

  const go = (path: string) => {
    setOpen(false);
    router.push(path);
  };

  const showDropdown = open && enabled;
  const hasResults = !!results && (results.vendors.length + results.feedItems.length + results.subscriptions.length) > 0;

  return (
    <div className="relative" ref={ref}>
      <div className="flex items-center gap-2 rounded-lg px-3 py-1.5 w-[280px]" style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
        <Search size={16} className="text-muted" />
        <input
          ref={inputRef}
          value={q}
          onChange={(e) => setQ(e.target.value)}
          onFocus={() => setOpen(true)}
          placeholder="Search vendors, content, IDs…"
          className="bg-transparent outline-none text-[13px] font-medium w-full"
        />
        <kbd className="text-[10px] font-mono rounded px-1 py-0.5" style={{ background: 'var(--surface)', border: '1px solid var(--line)', color: 'var(--muted)' }}>
          ⌘K
        </kbd>
      </div>
      {showDropdown && (
        <div className="absolute left-0 top-full mt-2 w-[380px] rounded-xl overflow-hidden z-40"
          style={{ background: 'var(--surface)', border: '1px solid var(--line)', boxShadow: '0 24px 50px -20px rgba(0,0,0,0.35)' }}>
          {!hasResults && (
            <p className="px-4 py-5 text-[13px] text-muted font-semibold">
              {results ? 'No matches.' : 'Searching…'}
            </p>
          )}
          {results && results.vendors.length > 0 && (
            <ResultGroup label="Vendors">
              {results.vendors.map((v) => (
                <ResultRow key={v.id} title={v.name} sub={`${v.category} · ${v.town}`} onClick={() => go(resultRoute('vendor', v.status, v.id))} />
              ))}
            </ResultGroup>
          )}
          {results && results.feedItems.length > 0 && (
            <ResultGroup label="Feed items">
              {results.feedItems.map((f) => (
                <ResultRow key={f.id} title={f.vendor || `#${f.id}`} sub={f.text} onClick={() => go(resultRoute('feedItem', undefined, String(f.id)))} />
              ))}
            </ResultGroup>
          )}
          {results && results.subscriptions.length > 0 && (
            <ResultGroup label="Subscriptions">
              {results.subscriptions.map((s) => (
                <ResultRow key={s.id} title={s.vendor} sub={`${s.tier} · ${s.state}`} onClick={() => go(resultRoute('subscription', undefined, String(s.id)))} />
              ))}
            </ResultGroup>
          )}
        </div>
      )}
    </div>
  );
}

function ResultGroup({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="border-b border-line last:border-0">
      <p className="px-4 pt-2.5 pb-1 text-[10px] font-bold uppercase tracking-wide text-muted">{label}</p>
      {children}
    </div>
  );
}

function ResultRow({ title, sub, onClick }: { title: string; sub?: string; onClick: () => void }) {
  return (
    <button onClick={onClick} className="w-full text-left px-4 py-2 transition-colors hover:bg-[color:var(--bg)]">
      <span className="block text-[13px] font-bold truncate">{title}</span>
      {sub && <span className="block text-[11.5px] text-muted font-medium truncate">{sub}</span>}
    </button>
  );
}

function CommandBar({ title }: { title: string }) {
  const router = useRouter();
  const { data: kpis } = useKpis();
  const { data: me } = useMe();
  const { theme, toggle } = useTheme();

  const strip = [
    { label: 'Pending approvals', value: kpis?.pendingVendors ?? '—' },
    { label: 'Feed items', value: kpis?.feedItems ?? '—' },
    { label: 'Active subs', value: kpis?.activeSubs ?? '—' },
    { label: 'Platform MRR', value: kpis ? `$${kpis.mrr.toLocaleString()}` : '—' },
  ];

  return (
    <header className="flex items-center gap-4 px-5 h-[58px] shrink-0" style={{ background: 'var(--surface)', borderBottom: '1px solid var(--line)' }}>
      <h1 className="font-bold text-[15px] w-[180px] shrink-0">{title}</h1>
      <CommandSearch />

      <div className="flex-1 flex items-center justify-end gap-6">
        {strip.map((k) => (
          <div key={k.label} className="text-right">
            <p className="font-bold text-[14px]">{k.value}</p>
            <p className="text-[10px] font-semibold text-muted mt-0.5">{k.label}</p>
          </div>
        ))}
        <button aria-label="Toggle theme" onClick={toggle}
          className="w-9 h-9 rounded-lg flex items-center justify-center border border-line bg-bg text-ink-soft">
          {theme === 'light' ? <Moon size={16} /> : <Sun size={16} />}
        </button>
        <AdminNotificationBell />
        <button
          onClick={() => router.push('/profile')}
          className="flex items-center gap-2 rounded-lg px-2 py-1.5 border border-line bg-bg"
        >
          <span className="w-7 h-7 rounded-full bg-accent text-white flex items-center justify-center text-[11px] font-bold">A</span>
          <span className="text-left leading-tight">
            <span className="block font-bold text-[12px]">Admin</span>
            <span className="block text-[10px] font-semibold text-muted">{me?.isSuperAdmin ? 'Super-admin' : 'Admin'}</span>
          </span>
        </button>
      </div>
    </header>
  );
}

export default function Shell({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const [ready, setReady] = useState(false);
  const { data: me, isError } = useMe();
  const { data: kpis } = useKpis();

  useEffect(() => {
    if (!api.getToken()) {
      router.replace('/login');
      return;
    }
    api.onUnauthorized = () => router.replace('/login');
    setReady(true);
  }, [router]);

  useEffect(() => {
    const saved = (localStorage.getItem('spotto_admin_theme') as 'light' | 'dark') || 'light';
    document.documentElement.dataset.theme = saved;
  }, []);

  if (!ready || (!me && !isError)) {
    return <div className="min-h-screen flex items-center justify-center text-muted font-bold" style={{ background: 'var(--bg)' }}>Loading…</div>;
  }
  if (me && me.role !== 'admin') {
    return <div className="min-h-screen flex items-center justify-center font-bold">This account is not an admin.</div>;
  }

  const title = NAV.find((n) => pathname.startsWith(n.href))?.label ?? (pathname.startsWith('/profile') ? 'Profile' : '');

  return (
    <div className="min-h-screen flex" style={{ background: 'var(--bg)' }}>
      <Rail pending={kpis?.pendingVendors ?? 0} />
      <div className="flex-1 min-w-0 flex flex-col">
        <CommandBar title={title} />
        <main className="flex-1 overflow-y-auto p-6">{children}</main>
      </div>
    </div>
  );
}
