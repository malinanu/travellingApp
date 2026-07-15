'use client';

// Vendor portal shell — persistent 248px sidebar + topbar with notification
// bell, ported from Prototype/design/VendorApp.jsx.

import { useQuery, useQueryClient } from '@tanstack/react-query';
import {
  BarChart3, Bell, LayoutDashboard, QrCode, Rocket, Store, Tag, Ticket,
  TrendingUp, UtensilsCrossed, Star, Clock, Wallet, LogOut, LucideIcon,
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
function isFoodCategory(category?: string) {
  const c = (category ?? '').toLowerCase();
  return FOOD_KEYWORDS.some((k) => c.includes(k));
}

const NOTIF_ICONS: Record<string, LucideIcon> = {
  Ticket, Star, Clock, TrendingUp, Wallet,
};

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
        className="relative rounded-xl bg-white border border-line p-2.5 hover:border-accent transition-colors"
      >
        <Bell size={18} />
        {unread > 0 && (
          <span className="absolute -top-1 -right-1 bg-coral text-white text-[10px] font-extrabold rounded-full min-w-[17px] h-[17px] px-1 flex items-center justify-center">
            {unread}
          </span>
        )}
      </button>
      {open && (
        <div className="absolute right-0 mt-2 w-[360px] card shadow-xl z-40 overflow-hidden">
          <div className="flex items-center justify-between px-4 py-3 border-b border-line">
            <span className="font-extrabold text-[14px]">Notifications</span>
            <button onClick={markAll} className="text-[12.5px] font-bold text-accent hover:underline">
              Mark all read
            </button>
          </div>
          <div className="max-h-[340px] overflow-y-auto">
            {notifs.length === 0 && (
              <p className="px-4 py-6 text-[13px] text-muted font-semibold">Nothing yet.</p>
            )}
            {notifs.map((n) => {
              const NIcon = NOTIF_ICONS[n.icon] ?? Bell;
              return (
                <div key={n.id} className={`flex gap-3 px-4 py-3 border-b border-line last:border-0 ${n.unread ? 'bg-accent-tint/40' : ''}`}>
                  <span className="mt-0.5 rounded-lg bg-sand p-1.5 h-fit"><NIcon size={16} /></span>
                  <div className="min-w-0">
                    <p className="text-[13.5px] font-extrabold leading-snug">{n.title}</p>
                    <p className="text-[12.5px] text-muted font-semibold">{n.sub}</p>
                    <p className="text-[11.5px] text-muted/80 font-semibold mt-0.5">{n.time}</p>
                  </div>
                </div>
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

  return (
    <div className="min-h-screen flex">
      <aside className="w-[248px] shrink-0 border-r border-line bg-white flex flex-col sticky top-0 h-screen">
        <div className="px-5 py-5 flex items-center gap-2.5">
          <span className="w-9 h-9 rounded-xl bg-accent text-white flex items-center justify-center font-extrabold">S</span>
          <div>
            <p className="font-extrabold text-[15px] leading-tight">Spotto</p>
            <p className="text-[11.5px] text-muted font-bold">Vendor Portal</p>
          </div>
        </div>
        <nav className="px-3 flex-1 space-y-1">
          {NAV.filter((n) => !n.foodOnly || isFood).map((n) => {
            const on = pathname.startsWith(n.href);
            const Icon = n.icon;
            return (
              <Link
                key={n.href}
                href={n.href}
                className={`flex items-center gap-3 rounded-xl px-3.5 py-2.5 font-extrabold text-[14.5px] transition-colors ${
                  on ? 'bg-accent text-white' : 'text-ink hover:bg-sand'
                }`}
              >
                <Icon size={20} strokeWidth={on ? 2.5 : 2} />
                {n.label}
              </Link>
            );
          })}
        </nav>
        <div className="p-3">
          <Link
            href="/account"
            className={`w-full rounded-xl p-3 flex items-center gap-3 text-left transition-colors ${
              pathname.startsWith('/account') ? 'bg-accent-tint' : 'bg-sand hover:bg-accent-tint'
            }`}
          >
            <span className="w-9 h-9 rounded-full bg-accent text-white flex items-center justify-center text-[13px] font-extrabold">
              {initials(business?.name ?? 'S')}
            </span>
            <span className="min-w-0">
              <span className="block truncate font-extrabold text-[13.5px]">{business?.name ?? '…'}</span>
              <span className="block text-[11.5px] text-muted font-bold">{business?.tier ?? ''} plan</span>
            </span>
          </Link>
        </div>
      </aside>

      <div className="flex-1 min-w-0 flex flex-col">
        <header className="sticky top-0 z-30 bg-cream/90 backdrop-blur border-b border-line px-8 py-4 flex items-center justify-between">
          <h1 className="text-[20px] font-extrabold">{title}</h1>
          <div className="flex items-center gap-3">
            <NotificationBell />
            <Link
              href="/account"
              className="rounded-xl bg-white border border-line px-3 py-2 flex items-center gap-2 font-extrabold text-[13px] hover:border-accent transition-colors"
            >
              <span className="w-6 h-6 rounded-full bg-accent text-white flex items-center justify-center text-[10.5px]">
                {initials(business?.owner ?? 'V')}
              </span>
              {business?.owner ?? ''}
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
        <h1 className="text-xl font-extrabold mb-2">
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
