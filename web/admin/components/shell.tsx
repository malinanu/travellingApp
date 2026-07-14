'use client';

// Admin command-center shell — 232px dark rail + command bar with KPI strip,
// theme toggle (light/dark CSS vars), and search stub. Port of AdminApp.jsx.

import { useQuery } from '@tanstack/react-query';
import {
  Activity, Bell, CreditCard, Headphones, Map, Moon, Radio, Search, Sun, UserCheck, LucideIcon,
} from 'lucide-react';
import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { useEffect, useState } from 'react';

import type { AdminKPIs, User } from '@spotto/shared';
import { api } from '../lib/api';

const NAV: { href: string; label: string; icon: LucideIcon; live?: boolean }[] = [
  { href: '/queue', label: 'Approval queue', icon: UserCheck },
  { href: '/content', label: 'Content monitor', icon: Radio, live: true },
  { href: '/geo', label: 'Geo-analytics', icon: Map },
  { href: '/subscriptions', label: 'Subscriptions', icon: CreditCard },
  { href: '/audio', label: 'Audio guides', icon: Headphones },
];

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
  return (
    <aside className="w-[232px] shrink-0 flex flex-col" style={{ background: '#103d3a', borderRight: '1px solid rgba(255,255,255,0.06)' }}>
      <div className="flex items-center gap-2.5 px-5 h-[58px]" style={{ borderBottom: '1px solid rgba(255,255,255,0.06)' }}>
        <svg width="26" height="26" viewBox="0 0 32 32" fill="none">
          <path d="M16 3C10.5 3 6 7.3 6 12.7 6 19.5 16 29 16 29s10-9.5 10-16.3C26 7.3 21.5 3 16 3z" fill="var(--accent)" />
          <circle cx="16" cy="12.5" r="4.4" fill="#f6b81e" />
        </svg>
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
        {NAV.map((n) => {
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

function CommandBar({ title }: { title: string }) {
  const router = useRouter();
  const { data: kpis } = useKpis();
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
      <div className="flex items-center gap-2 rounded-lg px-3 py-1.5 w-[280px]" style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
        <Search size={16} className="text-muted" />
        <input placeholder="Search vendors, content, IDs…" className="bg-transparent outline-none text-[13px] font-medium w-full" />
        <kbd className="text-[10px] font-mono rounded px-1 py-0.5" style={{ background: 'var(--surface)', border: '1px solid var(--line)', color: 'var(--muted)' }}>
          ⌘K
        </kbd>
      </div>

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
        <button aria-label="Notifications" className="w-9 h-9 rounded-lg flex items-center justify-center border border-line bg-bg text-ink-soft">
          <Bell size={16} />
        </button>
        <button
          onClick={() => router.push('/profile')}
          className="flex items-center gap-2 rounded-lg px-2 py-1.5 border border-line bg-bg"
        >
          <span className="w-7 h-7 rounded-full bg-accent text-white flex items-center justify-center text-[11px] font-bold">A</span>
          <span className="text-left leading-tight">
            <span className="block font-bold text-[12px]">Admin</span>
            <span className="block text-[10px] font-semibold text-muted">Super-admin</span>
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
