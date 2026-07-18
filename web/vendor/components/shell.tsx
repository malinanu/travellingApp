'use client';

// Vendor portal shell — persistent 248px sidebar + topbar with notification
// bell, ported from Prototype/design/VendorApp.jsx.

import { useQuery, useQueryClient } from '@tanstack/react-query';
import {
  BarChart3, Bell, ChevronDown, LayoutDashboard, Menu, QrCode, Rocket, Search,
  Store, Tag, Ticket, TrendingUp, UtensilsCrossed, Star, Clock, Wallet, LogOut,
  LucideIcon,
} from 'lucide-react';
import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { useEffect, useRef, useState } from 'react';

import type { User, Vendor, VendorNotification } from '@spotto/shared';
import { api } from '../lib/api';

const NAV: { href: string; label: string; icon: LucideIcon; foodOnly?: boolean }[] = [
  { href: '/dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { href: '/offers', label: 'Offers', icon: Tag },
  { href: '/menu', label: 'Menu', icon: UtensilsCrossed, foodOnly: true },
  { href: '/growth', label: 'Growth', icon: Rocket },
  { href: '/validate', label: 'Validate', icon: QrCode },
  { href: '/reports', label: 'Reports', icon: BarChart3 },
  { href: '/profile', label: 'Profile', icon: Store },
];

// Matched case-insensitively against whatever substring appears in the
// category string — vendor categories are generated in a few different
// places (registration's lowercase ids like "cafe", the admin queue's
// capitalized "Restaurant", cmd/gendata's "Cafe"/"Bakery"/"Street Food")
// and none of them are guaranteed to equal one of these exactly.
const FOOD_KEYWORDS = ['restaurant', 'cafe', 'café', 'bar', 'bakery', 'food', 'coffee'];
export function isFoodCategory(category?: string) {
  const c = (category ?? '').toLowerCase();
  return FOOD_KEYWORDS.some((k) => c.includes(k));
}

const NOTIF_ICONS: Record<string, LucideIcon> = {
  Ticket, Star, Clock, TrendingUp, Wallet,
};

function toneStyle(tone: string) {
  if (tone === 'sun') return { background: 'rgba(246,184,30,0.16)', color: '#c98a00' };
  if (tone === 'muted') return { background: 'rgba(15,61,58,0.06)', color: 'var(--muted)' };
  return { background: 'var(--accent-tint)', color: 'var(--accent)' };
}

export function SpottoMark({ size = 28, light = false }: { size?: number; light?: boolean }) {
  return (
    <svg width={size} height={size} viewBox="0 0 32 32" fill="none">
      <path d="M16 3C10.5 3 6 7.3 6 12.7 6 19.5 16 29 16 29s10-9.5 10-16.3C26 7.3 21.5 3 16 3z" fill={light ? '#fff' : 'var(--accent)'} />
      <circle cx="16" cy="12.5" r="4.4" fill="var(--sun)" />
    </svg>
  );
}

export function useMe() {
  return useQuery<User>({
    queryKey: ['me'],
    queryFn: () => api.get<User>('/me'),
    enabled: typeof window !== 'undefined' && !!api.getToken(),
  });
}

export function useBusiness() {
  return useQuery<Vendor>({
    queryKey: ['business'],
    queryFn: () => api.get<Vendor>('/vendor/business'),
    enabled: typeof window !== 'undefined' && !!api.getToken(),
  });
}

function initials(name: string) {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((w) => w[0]?.toUpperCase())
    .join('');
}

function NotificationBell() {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  const qc = useQueryClient();
  const { data: notifs = [] } = useQuery<VendorNotification[]>({
    queryKey: ['notifications'],
    queryFn: () => api.get<VendorNotification[]>('/vendor/notifications'),
    refetchInterval: 5000,
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
    await api.post('/vendor/notifications/read');
    qc.invalidateQueries({ queryKey: ['notifications'] });
  };

  return (
    <div className="relative" ref={ref}>
      <button
        aria-label="Notifications"
        onClick={() => setOpen((o) => !o)}
        className="relative w-10 h-10 rounded-xl flex items-center justify-center border transition-colors"
        style={{ background: open ? 'var(--accent-tint)' : '#fff', borderColor: open ? 'var(--accent)' : 'var(--line)', color: open ? 'var(--accent)' : 'var(--ink)' }}
      >
        <Bell size={19} />
        {unread > 0 && (
          <span className="absolute top-2 right-2.5 w-2 h-2 rounded-full bg-sun" style={{ boxShadow: '0 0 0 2px #fff' }} />
        )}
      </button>
      {open && (
        <div className="absolute right-0 mt-2 w-[360px] rounded-2xl bg-white overflow-hidden z-40"
          style={{ border: '1px solid var(--line)', boxShadow: '0 24px 50px -20px rgba(16,61,58,0.35)' }}>
          <div className="flex items-center gap-2 px-4 py-3 border-b border-line">
            <span className="font-extrabold text-[14.5px] font-display">Notifications</span>
            {unread > 0 && (
              <span className="rounded-full px-1.5 py-0.5 text-[10.5px] font-extrabold bg-sun text-[#5a3b00]">{unread} new</span>
            )}
            {unread > 0 && (
              <button onClick={markAll} className="ml-auto text-[12px] font-extrabold text-accent">Mark all read</button>
            )}
          </div>
          <div className="max-h-[380px] overflow-y-auto">
            {notifs.length === 0 && (
              <p className="px-4 py-6 text-[13px] text-muted font-semibold">Nothing yet.</p>
            )}
            {notifs.map((n, i) => {
              const NIcon = NOTIF_ICONS[n.icon] ?? Bell;
              return (
                <button key={n.id} className="w-full flex items-start gap-3 px-4 py-3 text-left transition-colors"
                  style={{ borderTop: i ? '1px solid var(--line)' : 'none', background: n.unread ? 'rgba(246,184,30,0.05)' : '#fff' }}>
                  <span className="w-9 h-9 rounded-xl flex items-center justify-center shrink-0" style={toneStyle(n.tone)}>
                    <NIcon size={16} />
                  </span>
                  <span className="flex-1 min-w-0">
                    <span className="block text-[13px] font-extrabold leading-snug">{n.title}</span>
                    <span className="block text-[12px] text-muted font-semibold mt-0.5 truncate">{n.sub}</span>
                    <span className="block text-[10.5px] text-muted/80 font-bold mt-1">{n.time}</span>
                  </span>
                  {n.unread && <span className="w-2 h-2 rounded-full bg-sun shrink-0 mt-1.5" />}
                </button>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}

/** Full-page gate: requires a token, and an approved vendor for the portal. */
export default function Shell({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const [ready, setReady] = useState(false);
  const [navOpen, setNavOpen] = useState(false);
  const [search, setSearch] = useState('');
  const { data: me, isError } = useMe();
  const { data: business } = useBusiness();

  useEffect(() => {
    if (!api.getToken()) {
      router.replace('/login');
      return;
    }
    api.onUnauthorized = () => router.replace('/login');
    setReady(true);
  }, [router]);

  if (!ready || (!me && !isError)) {
    return <div className="min-h-screen flex items-center justify-center text-muted font-bold">Loading…</div>;
  }

  if (me && me.vendorStatus && me.vendorStatus !== 'approved') {
    return <PendingGate status={me.vendorStatus} />;
  }

  const isFood = isFoodCategory(business?.category);
  const title = NAV.find((n) => pathname.startsWith(n.href))?.label ??
    (pathname.startsWith('/account') ? 'Account' : '');

  const submitSearch = (e: React.FormEvent) => {
    e.preventDefault();
    const query = search.trim();
    router.push(query ? `/offers?q=${encodeURIComponent(query)}` : '/offers');
  };

  return (
    <div className="min-h-screen flex">
      <div
        onClick={() => setNavOpen(false)}
        className="fixed inset-0 z-30 bg-[rgba(16,61,58,0.4)] transition-opacity lg:hidden"
        style={{ opacity: navOpen ? 1 : 0, pointerEvents: navOpen ? 'auto' : 'none' }}
      />
      <aside
        className="w-[248px] shrink-0 border-r border-line bg-white flex flex-col fixed lg:sticky top-0 left-0 h-screen z-40 transition-transform lg:translate-x-0"
        style={{ transform: navOpen ? 'translateX(0)' : undefined, transitionTimingFunction: 'cubic-bezier(0.22,1,0.36,1)' }}
      >
        <div className="h-16 px-5 flex items-center gap-2.5 border-b border-line shrink-0">
          <SpottoMark size={28} />
          <div className="leading-none">
            <p className="font-display font-extrabold text-[18px] tracking-tight">Spotto</p>
            <p className="text-[10.5px] font-extrabold uppercase tracking-widest mt-0.5 text-accent">Vendor</p>
          </div>
        </div>
        <nav className="px-3 py-4 flex-1 space-y-1">
          {NAV.filter((n) => !n.foodOnly || isFood).map((n) => {
            const on = pathname.startsWith(n.href);
            const Icon = n.icon;
            return (
              <Link
                key={n.href}
                href={n.href}
                onClick={() => setNavOpen(false)}
                className={`w-full flex items-center gap-3 rounded-xl px-3.5 py-2.5 transition-colors ${
                  on ? 'bg-accent-tint text-accent' : 'text-ink-soft hover:bg-sand'
                }`}
              >
                <Icon size={20} strokeWidth={on ? 2.5 : 2} />
                <span className="font-extrabold text-[14.5px]">{n.label}</span>
                {n.href === '/validate' && (
                  <span className="ml-auto text-[10px] font-extrabold rounded-full px-1.5 py-0.5 bg-accent text-white">SCAN</span>
                )}
              </Link>
            );
          })}
        </nav>
        <div className="p-3">
          <Link
            href="/account"
            className={`w-full rounded-xl p-3 flex items-center gap-3 text-left transition-colors ${
              pathname.startsWith('/account') ? 'bg-accent-tint' : 'bg-sand'
            }`}
          >
            <span className="w-10 h-10 rounded-lg bg-accent text-white flex items-center justify-center text-[14px] font-extrabold shrink-0">
              {initials(business?.name ?? 'S')}
            </span>
            <span className="min-w-0">
              <span className="block truncate font-extrabold text-[13px]">{business?.name ?? '…'}</span>
              <span className="block text-[11.5px] text-muted font-semibold truncate">{business?.tier ?? ''} plan</span>
            </span>
          </Link>
        </div>
      </aside>

      <div className="flex-1 min-w-0 flex flex-col">
        <header className="h-16 sticky top-0 z-20 bg-cream/90 backdrop-blur border-b border-line px-5 flex items-center gap-3 shrink-0">
          <button onClick={() => setNavOpen(true)} className="lg:hidden w-9 h-9 rounded-lg flex items-center justify-center text-ink shrink-0">
            <Menu size={22} />
          </button>
          <form onSubmit={submitSearch} className="hidden sm:flex items-center gap-2 rounded-xl px-3 py-2 flex-1 max-w-xs bg-white border border-line">
            <Search size={17} className="text-muted shrink-0" />
            <input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search offers, codes…"
              className="bg-transparent outline-none text-[13.5px] font-semibold w-full text-ink"
            />
          </form>
          <h1 className="flex-1 sm:hidden font-display font-extrabold text-[17px]">{title}</h1>
          <div className="ml-auto flex items-center gap-2 shrink-0">
            <NotificationBell />
            <Link
              href="/account"
              className="rounded-xl bg-white px-3 py-2 flex items-center gap-2 border transition-colors"
              style={{ borderColor: pathname.startsWith('/account') ? 'var(--accent)' : 'var(--line)' }}
            >
              <span className="w-7 h-7 rounded-lg bg-accent text-white flex items-center justify-center text-[12px] font-extrabold">
                {initials(business?.owner ?? 'V')}
              </span>
              <ChevronDown size={15} className="text-muted" />
            </Link>
          </div>
        </header>
        <main id="scroll-main" className="flex-1 px-8 py-6">
          <div className="max-w-[1180px] mx-auto">{children}</div>
        </main>
      </div>
    </div>
  );
}

function PendingGate({ status }: { status: string }) {
  const router = useRouter();
  return (
    <div className="min-h-screen flex items-center justify-center p-6">
      <div className="card max-w-md w-full p-8 text-center">
        <span className="mx-auto w-14 h-14 rounded-2xl bg-sun/20 text-sun flex items-center justify-center mb-4">
          <Clock size={28} className="text-[#a97b00]" />
        </span>
        <h1 className="text-xl font-extrabold font-display mb-2">
          {status === 'rejected' ? 'Application not approved' : 'Pending review'}
        </h1>
        <p className="text-muted font-semibold text-[14px] leading-relaxed">
          {status === 'rejected'
            ? 'Your application was not approved. Contact partners@spotto.dev for details.'
            : status === 'edits'
              ? 'Our team requested edits to your application. Check your email for details, then resubmit.'
              : 'Our team is reviewing your business. You will get access to the full portal as soon as an admin approves it — usually within a day.'}
        </p>
        <button
          onClick={() => {
            api.setToken(null);
            router.replace('/login');
          }}
          className="mt-6 inline-flex items-center gap-2 font-extrabold text-accent hover:underline"
        >
          <LogOut size={16} /> Sign out
        </button>
      </div>
    </div>
  );
}
