'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Users, Store, Ticket, Star, Loader2, ArrowRight, AlertTriangle } from 'lucide-react';
import { useEffect, useState } from 'react';
import { Money, type AdminPhoneCodeSettings, type FinanceSummary, type FxRatesResponse } from '@spotto/shared';
import Shell, { hasAdminPerm, useMe } from '../components/shell';
import { api } from '../lib/api';
import Link from 'next/link';

interface PlatformAnalytics {
  totalUsers: number;
  totalVendors: number;
  totalClaims: number;
  totalReviews: number;
}

/** Bookings and money over the last 30 days, for admins with finance.view. */
function MoneyStrip() {
  const { data: me } = useMe();
  const allowed = hasAdminPerm(me, 'finance.view');
  const { data } = useQuery<FinanceSummary>({
    queryKey: ['dashboard-finance'],
    queryFn: () => api.get<FinanceSummary>('/admin/finance/summary'),
    enabled: allowed,
    refetchInterval: 60000,
  });
  if (!allowed) return null;
  const tiles = [
    { label: 'Bookings (30 days)', value: data?.bookings ?? '—', href: '/bookings' },
    { label: 'Paid by travellers', value: <Money minor={data?.grossMinor} />, href: '/finance' },
    { label: 'Spotto earned', value: <Money minor={(data?.commissionMinor ?? 0) + (data?.guestFeesMinor ?? 0)} />, href: '/finance' },
    { label: 'Refunds needing a person', value: data?.openRefunds ?? '—', href: '/finance' },
    ...(data?.chargebacks ? [{ label: 'Chargebacks', value: data.chargebacks, href: '/finance?tab=issues' }] : []),
    { label: 'Payouts waiting', value: data ? <span>{data.pendingPayouts} · <Money minor={data.pendingPayoutMinor} /></span> : '—', href: '/payouts' },
  ];
  return (
    <>
    <FxRateWarning />
    <div className="grid grid-cols-2 lg:grid-cols-5 gap-3 mb-6">
      {tiles.map((t) => (
        <Link key={t.label} href={t.href} className="panel p-4 hover:border-[color:var(--accent)] transition-colors">
          <p className="text-[11.5px] font-bold uppercase tracking-wide text-[var(--muted)]">{t.label}</p>
          <p className="mt-1 text-[20px] font-bold text-[color:var(--ink)] tabular-nums">{t.value}</p>
        </Link>
      ))}
    </div>
    </>
  );
}

/** Warns when a gateway that charges another currency has no usable rate
 *  (it then stops taking payments) or its rate expires within a day. */
function FxRateWarning() {
  const { data } = useQuery<FxRatesResponse>({
    queryKey: ['fx-rates'],
    queryFn: () => api.get<FxRatesResponse>('/admin/finance/fx-rates'),
    refetchInterval: 300000,
  });
  if (!data) return null;
  const now = Date.now();
  const maxAgeMs = data.maxAgeHours * 3600_000;
  const problems: string[] = [];
  for (const g of data.gateways) {
    if (g.chargeCurrency === 'LKR') continue;
    const rate = data.rates.find((r) => r.currency === g.chargeCurrency);
    if (!rate) problems.push(`${g.displayName} has no ${g.chargeCurrency} rate, so it isn't taking payments.`);
    else if (rate.stale) problems.push(`${g.displayName}'s ${g.chargeCurrency} rate is too old, so it isn't taking payments.`);
    else if (now - new Date(rate.setAt).getTime() > maxAgeMs - 24 * 3600_000) {
      problems.push(`${g.displayName}'s ${g.chargeCurrency} rate expires within a day.`);
    }
  }
  if (problems.length === 0) return null;
  return (
    <Link href="/finance?tab=fx" className="mb-4 flex items-start gap-3 rounded-xl border border-amber-500/40 bg-amber-500/10 p-3 text-[13px] text-amber-900">
      <AlertTriangle className="h-4 w-4 mt-0.5 shrink-0" />
      <span>{problems.join(' ')} Set a new exchange rate in Finance.</span>
    </Link>
  );
}

export default function DashboardPage() {
  const { data, isLoading, isError } = useQuery<PlatformAnalytics>({
    queryKey: ['adminAnalytics'],
    queryFn: () => api.get<PlatformAnalytics>('/admin/analytics'),
    refetchInterval: 30000,
  });

  if (isLoading) {
    return (
      <Shell>
        <div className="flex h-full items-center justify-center">
          <Loader2 className="h-8 w-8 animate-spin text-[var(--muted)]" />
        </div>
      </Shell>
    );
  }

  if (isError || !data) {
    return (
      <Shell>
        <div className="p-8 text-[var(--accent)] font-medium">Failed to load analytics</div>
      </Shell>
    );
  }

  const kpis = [
    { label: 'Total Travellers', value: data.totalUsers, icon: Users, color: 'text-blue-400', bg: 'bg-blue-400/10' },
    { label: 'Registered Vendors', value: data.totalVendors, icon: Store, color: 'text-emerald-400', bg: 'bg-emerald-400/10' },
    { label: 'Offers Claimed', value: data.totalClaims, icon: Ticket, color: 'text-purple-400', bg: 'bg-purple-400/10' },
    { label: 'Reviews Written', value: data.totalReviews, icon: Star, color: 'text-amber-400', bg: 'bg-amber-400/10' },
  ];

  return (
    <Shell>
      <div className="p-6 md:p-8 max-w-7xl mx-auto w-full">
        <header className="mb-10">
          <h1 className="text-3xl font-bold tracking-tight text-[color:var(--ink)] mb-2">Spotto HQ</h1>
          <p className="text-[var(--muted)] text-sm md:text-base max-w-2xl">
            Real-time platform metrics and top-level activity overview.
          </p>
        </header>

        <MoneyStrip />
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 md:gap-6">
          {kpis.map((kpi, i) => {
            const Icon = kpi.icon;
            return (
              <div key={i} className="relative group overflow-hidden rounded-2xl panel p-6 hover:border-[color:var(--accent)] transition-colors">
                <div className="flex items-center justify-between mb-4">
                  <div className={`p-3 rounded-xl ${kpi.bg}`}>
                    <Icon className={`h-6 w-6 ${kpi.color}`} />
                  </div>
                </div>
                <div className="flex flex-col gap-1">
                  <span className="text-4xl font-bold text-[color:var(--ink)] tracking-tight">
                    {kpi.value.toLocaleString()}
                  </span>
                  <span className="text-sm font-medium text-[var(--muted)]">
                    {kpi.label}
                  </span>
                </div>
                
                {/* Subtle gradient glow effect */}
                <div className={`absolute -inset-x-4 -bottom-4 h-12 bg-gradient-to-t ${kpi.bg.replace('/10', '/0')} to-transparent opacity-0 group-hover:opacity-100 transition-opacity blur-xl`} />
              </div>
            );
          })}
        </div>

        <div className="mt-12 grid grid-cols-1 md:grid-cols-2 gap-6">
            {/* Quick Actions / Shortcuts */}
            <div className="rounded-2xl panel p-6 flex flex-col">
              <h2 className="text-lg font-semibold text-[color:var(--ink)] mb-4">Quick Actions</h2>
              <div className="flex flex-col gap-2 flex-grow">
                <Link href="/queue" className="flex items-center justify-between p-4 rounded-xl bg-black/[0.04] hover:bg-black/[0.08] transition-colors group">
                  <span className="text-sm font-medium text-[color:var(--ink)]">Review Pending Vendors</span>
                  <ArrowRight className="h-4 w-4 text-[var(--muted)] group-hover:text-[color:var(--ink)] transition-colors" />
                </Link>
                <Link href="/access-grants" className="flex items-center justify-between p-4 rounded-xl bg-black/[0.04] hover:bg-black/[0.08] transition-colors group">
                  <span className="text-sm font-medium text-[color:var(--ink)]">Manage Profile Grants</span>
                  <ArrowRight className="h-4 w-4 text-[var(--muted)] group-hover:text-[color:var(--ink)] transition-colors" />
                </Link>
                <Link href="/local-guide" className="flex items-center justify-between p-4 rounded-xl bg-black/[0.04] hover:bg-black/[0.08] transition-colors group">
                  <span className="text-sm font-medium text-[color:var(--ink)]">Train Local Guide AI</span>
                  <ArrowRight className="h-4 w-4 text-[var(--muted)] group-hover:text-[color:var(--ink)] transition-colors" />
                </Link>
              </div>
            </div>

            {/* Platform Security */}
            <SecuritySettingsPanel />

            <PhoneCodeLimitsPanel />
        </div>

      </div>
    </Shell>
  );
}

function SecuritySettingsPanel() {
  const { data, refetch } = useQuery<{ rateLimitRPM: number }>({
    queryKey: ['adminSettings'],
    queryFn: () => api.get<{ rateLimitRPM: number }>('/admin/settings'),
  });

  const updateSettings = useMutation({
    mutationFn: (rpm: number) => api.put('/admin/settings', { rateLimitRPM: rpm }),
    onSuccess: () => refetch(),
  });

  if (!data) return (
    <div className="rounded-2xl panel p-6 flex items-center justify-center">
      <Loader2 className="h-6 w-6 animate-spin text-[var(--muted)]" />
    </div>
  );

  return (
    <div className="rounded-2xl panel p-6 flex flex-col">
      <div className="flex items-center justify-between mb-2">
        <h2 className="text-lg font-semibold text-[color:var(--ink)]">Platform Security</h2>
        {updateSettings.isPending && <Loader2 className="h-4 w-4 animate-spin text-[var(--muted)]" />}
      </div>
      <p className="text-sm text-[var(--muted)] mb-6">
        Adjust the global API rate limit (requests per minute per IP address) in real-time to defend against traffic spikes or abuse.
      </p>

      <div className="flex-grow flex flex-col justify-center">
        <div className="flex justify-between items-end mb-3">
          <label className="text-sm font-medium text-[var(--accent)]">Global API Rate Limit</label>
          <span className="text-xl font-bold text-[color:var(--ink)] tracking-tight">{data.rateLimitRPM} <span className="text-sm font-normal text-[var(--muted)]">RPM</span></span>
        </div>
        <div className="flex items-center gap-4">
          <input 
            type="range" 
            min="10" 
            max="1000" 
            step="10"
            value={data.rateLimitRPM}
            onChange={(e) => updateSettings.mutate(parseInt(e.target.value, 10))}
            className="w-full accent-emerald-500 bg-black/10 h-2 rounded-lg appearance-none cursor-pointer"
          />
        </div>
        <div className="flex justify-between text-xs text-[var(--muted)] mt-2 font-medium uppercase tracking-wider">
          <span>Strict (10)</span>
          <span>Permissive (1000)</span>
        </div>
      </div>
    </div>
  );
}

// Daily limits on phone-verification codes: each one is a paid SMS or
// WhatsApp message, so a flood of fake sign-ups can't run up the bill.
// Sri Lankan numbers (text.lk) and foreign ones (Twilio — dearer, and where
// SMS fraud aims) have separate limits. Super-admins change them; other
// admins see the usage.
function PhoneCodeLimitsPanel() {
  const qc = useQueryClient();
  const { data: me } = useMe();
  const { data } = useQuery<AdminPhoneCodeSettings>({
    queryKey: ['admin-phone-codes'],
    queryFn: () => api.get<AdminPhoneCodeSettings>('/admin/settings/phone-codes'),
    refetchInterval: 60000,
  });
  const [lk, setLk] = useState('');
  const [other, setOther] = useState('');
  useEffect(() => {
    if (data) {
      setLk(String(data.limitSriLanka));
      setOther(String(data.limitOther));
    }
  }, [data?.limitSriLanka, data?.limitOther]); // eslint-disable-line react-hooks/exhaustive-deps

  const save = useMutation({
    mutationFn: () =>
      api.put<AdminPhoneCodeSettings>('/admin/settings/phone-codes', { limitSriLanka: Number(lk), limitOther: Number(other) }),
    onSuccess: (saved) => {
      qc.setQueryData(['admin-phone-codes'], saved);
      qc.invalidateQueries({ queryKey: ['admin-notifications'] });
    },
  });

  if (!data) return (
    <div className="md:col-span-2 rounded-2xl panel p-6 flex items-center justify-center">
      <Loader2 className="h-6 w-6 animate-spin text-[var(--muted)]" />
    </div>
  );

  const valid = (v: string) => /^\d+$/.test(v) && Number(v) >= 1 && Number(v) <= 100000;
  const changed = lk !== String(data.limitSriLanka) || other !== String(data.limitOther);
  const canEdit = !!me?.isSuperAdmin;

  return (
    <div className="md:col-span-2 rounded-2xl panel p-6">
      <h2 className="text-lg font-semibold text-[color:var(--ink)] mb-2">Phone verification codes</h2>
      <p className="text-sm text-[var(--muted)] mb-6">
        Each code is a paid SMS or WhatsApp message. When a daily limit is reached, new codes pause until the
        24-hour window clears; people can still use the app. You&apos;ll get a bell alert at 80%.
      </p>
      <form className="grid grid-cols-1 md:grid-cols-2 gap-6"
        onSubmit={(e) => { e.preventDefault(); if (canEdit && changed && valid(lk) && valid(other)) save.mutate(); }}>
        <UsageRow label="Sri Lankan numbers" via="SMS by text.lk, or WhatsApp" sent={data.sentSriLanka} limit={data.limitSriLanka}
          value={lk} onChange={setLk} canEdit={canEdit} invalid={!valid(lk)} />
        <UsageRow label="Foreign numbers" via="SMS by Twilio, or WhatsApp" sent={data.sentOther} limit={data.limitOther}
          value={other} onChange={setOther} canEdit={canEdit} invalid={!valid(other)} />
        <div className="md:col-span-2 flex items-center justify-end gap-3">
          {save.isError && <span className="text-sm font-medium text-rose-400 mr-auto">{(save.error as Error).message}</span>}
          {save.isSuccess && !changed && <span className="text-sm font-medium text-emerald-400 mr-auto">Saved.</span>}
          {canEdit ? (
            <button type="submit" disabled={!changed || !valid(lk) || !valid(other) || save.isPending}
              className="rounded-xl px-5 py-2.5 text-sm font-semibold text-white bg-emerald-600 hover:bg-emerald-500 disabled:opacity-40 disabled:cursor-not-allowed">
              {save.isPending ? 'Saving…' : 'Save limits'}
            </button>
          ) : (
            <span className="text-xs text-[var(--muted)] font-medium">Only super-admins can change these limits.</span>
          )}
        </div>
      </form>
    </div>
  );
}

function UsageRow({ label, via, sent, limit, value, onChange, canEdit, invalid }: {
  label: string;
  via: string;
  sent: number;
  limit: number;
  value: string;
  onChange: (v: string) => void;
  canEdit: boolean;
  invalid: boolean;
}) {
  const pct = limit > 0 ? sent / limit : 0;
  const color = pct >= 1 ? '#e11d48' : pct >= 0.8 ? '#d97706' : '#10b981';
  return (
    <div>
      <div className="flex items-end justify-between mb-2">
        <span className="text-sm font-medium text-[var(--accent)]">
          {label} <span className="text-[var(--muted)] font-normal">· {via}</span>
        </span>
        <span className="text-sm text-[color:var(--ink)] font-semibold">
          {sent.toLocaleString()} <span className="text-[var(--muted)] font-normal">of {limit.toLocaleString()} in 24h</span>
        </span>
      </div>
      <div className="h-2 rounded-full bg-black/10 overflow-hidden">
        <div className="h-full rounded-full" style={{ width: `${Math.min(100, pct * 100)}%`, background: color }} />
      </div>
      {canEdit && (
        <label className="flex items-center gap-2 mt-3 text-xs text-[var(--muted)] font-medium">
          Daily limit
          <input type="number" min={1} max={100000} value={value} onChange={(e) => onChange(e.target.value)}
            className="w-28 rounded-lg bg-transparent border px-2.5 py-1.5 text-sm text-[color:var(--ink)] outline-none"
            style={{ borderColor: invalid ? '#e11d48' : 'rgba(255,255,255,0.1)' }} />
        </label>
      )}
    </div>
  );
}
