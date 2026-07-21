'use client';

// Vendor directory — every registered vendor regardless of status or
// whether it has map coordinates, which the approval queue (pending only)
// and the geo map (coords only) each miss. Selecting a vendor opens the
// monitor pane: live activity stats plus the same admin actions the queue
// and geo pages expose.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  AlertTriangle, CheckCircle2, CreditCard, Edit3, MapPin,
  Radio, ShieldAlert, Star, Ticket, Users, XCircle,
} from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';

import { timeAgo, type AdminVendorStats, type Vendor, type VendorStatus } from '@spotto/shared';
import Shell from '../../components/shell';
import { InlineErrorBanner } from '../../components/route-status';
import { api } from '../../lib/api';

const STATUS_ORDER: VendorStatus[] = ['pending', 'approved', 'edits', 'suspended', 'rejected'];

const STATUS_TONE: Record<VendorStatus, { bg: string; fg: string }> = {
  pending: { bg: 'rgba(246,184,30,0.16)', fg: '#a97b00' },
  approved: { bg: 'rgba(22,163,74,0.14)', fg: '#15803d' },
  edits: { bg: 'rgba(37,99,235,0.14)', fg: '#1d4ed8' },
  suspended: { bg: 'rgba(225,29,72,0.12)', fg: '#e11d48' },
  rejected: { bg: 'rgba(255,255,255,0.06)', fg: 'var(--muted)' },
};

function StatusBadge({ status }: { status: VendorStatus }) {
  const tone = STATUS_TONE[status] ?? STATUS_TONE.rejected;
  return (
    <span className="text-[10.5px] font-bold uppercase tracking-wide rounded-full px-2 py-0.5 shrink-0"
      style={{ background: tone.bg, color: tone.fg }}>
      {status}
    </span>
  );
}

export default function VendorsPage() {
  const qc = useQueryClient();
  // No status param — this is the only admin surface that lists them all.
  const { data: vendors = [], isError } = useQuery<Vendor[]>({
    queryKey: ['admin-vendors', 'all'],
    queryFn: () => api.get<Vendor[]>('/admin/vendors'),
  });

  const [query, setQuery] = useState('');
  const [status, setStatus] = useState<VendorStatus | 'all'>('all');
  const [category, setCategory] = useState('all');
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const counts = useMemo(() => {
    const c: Record<string, number> = {};
    for (const v of vendors) c[v.status] = (c[v.status] ?? 0) + 1;
    return c;
  }, [vendors]);

  const categories = useMemo(
    () => [...new Set(vendors.map((v) => v.category).filter(Boolean))].sort(),
    [vendors],
  );

  const visible = useMemo(() => {
    const q = query.trim().toLowerCase();
    return vendors.filter((v) => {
      if (status !== 'all' && v.status !== status) return false;
      if (category !== 'all' && v.category !== category) return false;
      if (!q) return true;
      return `${v.name} ${v.town} ${v.owner}`.toLowerCase().includes(q);
    });
  }, [vendors, query, status, category]);

  // Keep the selection valid as filters narrow the list.
  useEffect(() => {
    if (selectedId && !visible.some((v) => v.id === selectedId)) setSelectedId(null);
  }, [visible, selectedId]);

  const selected = visible.find((v) => v.id === selectedId) ?? null;

  const act = useMutation({
    mutationFn: ({ id, action }: { id: string; action: string }) =>
      api.post(`/admin/vendors/${id}/${action}`),
    // Refreshes this list, the approval queue, and the KPI strip's pending count.
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin-vendors'] });
      qc.invalidateQueries({ queryKey: ['admin-kpis'] });
    },
  });

  return (
    <Shell>
      {isError && <InlineErrorBanner message="Couldn't load the vendor directory — try refreshing." />}

      <div className="panel px-4 py-3 mb-4 flex items-center gap-4 flex-wrap">
        <p className="font-bold text-[13.5px]">
          {vendors.length} registered
          {visible.length !== vendors.length && (
            <span className="text-muted font-semibold"> · {visible.length} shown</span>
          )}
        </p>
        <div className="flex items-center gap-1.5">
          {STATUS_ORDER.filter((s) => counts[s]).map((s) => (
            <button key={s} onClick={() => setStatus(status === s ? 'all' : s)}
              className="text-[11px] font-bold rounded-full px-2.5 py-1 transition-opacity"
              style={{ ...{ background: STATUS_TONE[s].bg, color: STATUS_TONE[s].fg }, opacity: status === 'all' || status === s ? 1 : 0.4 }}>
              {counts[s]} {s}
            </button>
          ))}
        </div>

        <div className="flex items-center gap-2 ml-auto">
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search name, town, owner…"
            className="rounded-lg px-3 py-1.5 text-[13px] font-medium outline-none w-[240px]"
            style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}
          />
          <select value={status} onChange={(e) => setStatus(e.target.value as VendorStatus | 'all')}
            className="rounded-lg px-2.5 py-1.5 text-[13px] font-semibold outline-none"
            style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
            <option value="all">All statuses</option>
            {STATUS_ORDER.map((s) => <option key={s} value={s}>{s}</option>)}
          </select>
          <select value={category} onChange={(e) => setCategory(e.target.value)}
            className="rounded-lg px-2.5 py-1.5 text-[13px] font-semibold outline-none"
            style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
            <option value="all">All categories</option>
            {categories.map((c) => <option key={c} value={c}>{c}</option>)}
          </select>
        </div>
      </div>

      <div className="grid grid-cols-[340px_1fr] gap-5 items-start">
        <div className="panel overflow-hidden">
          <div className="max-h-[calc(100vh-220px)] overflow-y-auto">
            {visible.map((v) => (
              <button
                key={v.id}
                onClick={() => setSelectedId(v.id)}
                className="w-full text-left px-4 py-3 border-b border-line last:border-0 transition-colors"
                style={{ background: selected?.id === v.id ? 'color-mix(in srgb, var(--accent) 10%, transparent)' : 'transparent' }}
              >
                <div className="flex items-center justify-between gap-2">
                  <p className="font-bold text-[13.5px] truncate">{v.name}</p>
                  <StatusBadge status={v.status} />
                </div>
                <p className="text-[12px] text-muted font-medium truncate">
                  {v.category || 'uncategorised'} · {v.town || 'no town'}
                </p>
                <p className="text-[11px] text-muted font-semibold mt-0.5">Joined {timeAgo(v.submittedAt)}</p>
              </button>
            ))}
            {visible.length === 0 && (
              <p className="px-4 py-8 text-center text-muted font-semibold text-[13px]">
                {vendors.length === 0 ? 'No vendors registered yet.' : 'No vendors match these filters.'}
              </p>
            )}
          </div>
        </div>

        {selected ? (
          <VendorMonitor
            vendor={selected}
            busy={act.isPending}
            onAct={(action) => act.mutate({ id: selected.id, action })}
          />
        ) : (
          <div className="panel p-10 text-center text-muted font-semibold">
            Select a vendor to see its stats.
          </div>
        )}
      </div>
    </Shell>
  );
}

function VendorMonitor({
  vendor,
  busy,
  onAct,
}: {
  vendor: Vendor;
  busy: boolean;
  onAct: (action: string) => void;
}) {
  const { data: stats, isError } = useQuery<AdminVendorStats>({
    queryKey: ['admin-vendor-stats', vendor.id],
    queryFn: () => api.get<AdminVendorStats>(`/admin/vendors/${vendor.id}/stats`),
  });

  const hasCoords = vendor.lat !== 0 || vendor.lng !== 0;

  return (
    <div className="panel p-6">
      <div className="flex items-start justify-between gap-4 mb-1">
        <div>
          <h2 className="text-xl font-bold">{vendor.name}</h2>
          <p className="text-muted font-medium text-[13.5px] mt-0.5">
            {vendor.category || 'uncategorised'} · {vendor.town || 'no town'} · owner {vendor.owner || '—'}
          </p>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          <StatusBadge status={vendor.status} />
          <span className="text-[10.5px] font-bold uppercase rounded-full px-2 py-0.5"
            style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
            {vendor.tier}
          </span>
        </div>
      </div>

      <p className="flex items-center gap-1.5 text-[12.5px] font-semibold mt-2"
        style={{ color: hasCoords ? 'var(--muted)' : '#d97706' }}>
        <MapPin size={13} />
        {hasCoords
          ? `${vendor.lat.toFixed(4)}, ${vendor.lng.toFixed(4)}${vendor.regionId ? ` · region ${vendor.regionId}` : ''}`
          : 'No location set — this vendor does not appear on the geo map.'}
      </p>

      {stats?.flagged && (
        <div className="rounded-xl p-4 mt-4" style={{ background: '#fde4d84d', border: '1px solid #e11d4855' }}>
          <p className="font-bold text-[13px] flex items-center gap-2" style={{ color: '#e11d48' }}>
            <ShieldAlert size={14} /> {stats.badReviews} bad reviews (1-2 stars) — flagged
          </p>
        </div>
      )}

      {isError && <div className="mt-4"><InlineErrorBanner message="Couldn't load this vendor's stats." /></div>}

      <div className="grid grid-cols-4 gap-3 mt-5">
        <StatTile icon={Ticket} label="Redemptions"
          value={stats ? `${stats.redeemed}` : '—'}
          sub={stats ? `${stats.redeemed30d} in 30d · ${stats.claimed} claimed` : undefined} />
        <StatTile icon={Users} label="Check-ins"
          value={stats ? `${stats.checkins}` : '—'}
          sub={stats ? `${stats.checkins30d} in 30d` : undefined} />
        <StatTile icon={Star} label="Rating"
          value={stats && stats.reviews > 0 ? stats.avgRating.toFixed(1) : '—'}
          sub={stats ? `${stats.reviews} reviews · ${stats.badReviews} bad` : undefined} />
        <StatTile icon={Radio} label="Offers"
          value={stats ? `${stats.offersLive}` : '—'}
          sub={stats ? `live · ${stats.offersDraft} draft · ${stats.offers} total` : undefined} />
        <StatTile icon={CreditCard} label="Subscription"
          value={stats?.subTier ?? vendor.tier}
          sub={stats?.subState ? `${stats.subState}${stats.subMrr ? ` · $${stats.subMrr}/mo` : ''}` : 'no subscription'} />
        <StatTile icon={Users} label="Team"
          value={stats ? `${stats.teamTotal}` : '—'}
          sub={stats ? `${stats.teamManagers} mgr · ${stats.teamStaff} staff` : undefined} />
        <StatTile icon={Users} label="Pending invites"
          value={stats ? `${stats.pendingInvites}` : '—'}
          sub={stats?.subRenewsAt ? `renews ${timeAgo(stats.subRenewsAt)}` : undefined} />
      </div>

      {vendor.flags && vendor.flags.length > 0 && (
        <div className="rounded-xl p-4 mt-5" style={{ background: '#fdf0cd4d', border: '1px solid #f6b81e55' }}>
          <p className="font-bold text-[13px]" style={{ color: '#a97b00' }}>Flags</p>
          <ul className="mt-1.5 space-y-1">
            {vendor.flags.map((f) => <li key={f} className="text-[13px] font-medium">{f}</li>)}
          </ul>
        </div>
      )}

      {vendor.note && <p className="text-[14px] font-medium leading-relaxed mt-5">{vendor.note}</p>}

      <div className="flex gap-3 mt-6">
        {vendor.status === 'pending' || vendor.status === 'edits' ? (
          <>
            <button className="btn btn-primary flex-1 py-2.5 flex items-center justify-center gap-2"
              disabled={busy} onClick={() => onAct('approve')}>
              <CheckCircle2 size={16} /> Approve
            </button>
            <button className="btn flex-1 py-2.5 flex items-center justify-center gap-2"
              disabled={busy} onClick={() => onAct('request-edits')}>
              <Edit3 size={16} /> Request edits
            </button>
            <button className="btn btn-danger flex-1 py-2.5 flex items-center justify-center gap-2"
              disabled={busy} onClick={() => onAct('reject')}>
              <XCircle size={16} /> Reject
            </button>
          </>
        ) : vendor.status === 'suspended' ? (
          <button className="btn btn-primary flex-1 py-2.5 flex items-center justify-center gap-2"
            disabled={busy} onClick={() => onAct('reinstate')}>
            <CheckCircle2 size={16} /> Reinstate
          </button>
        ) : (
          <button className="btn btn-danger flex-1 py-2.5 flex items-center justify-center gap-2"
            disabled={busy} onClick={() => onAct('suspend')}>
            <AlertTriangle size={16} /> Suspend
          </button>
        )}
      </div>
      {vendor.status === 'suspended' && (
        <p className="text-[11.5px] text-muted font-semibold mt-2">
          Suspended vendors can&apos;t log into the vendor portal until reinstated.
        </p>
      )}
    </div>
  );
}

function StatTile({
  icon: Icon,
  label,
  value,
  sub,
  warn,
}: {
  icon: typeof Ticket;
  label: string;
  value: string;
  sub?: string;
  warn?: boolean;
}) {
  return (
    <div className="rounded-xl p-3.5" style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
      <p className="flex items-center gap-1.5 text-[11.5px] font-bold text-muted"><Icon size={13} /> {label}</p>
      <p className="text-[18px] font-bold mt-1" style={{ color: warn ? '#e11d48' : 'var(--ink)' }}>{value}</p>
      {sub && <p className="text-[11px] text-muted font-semibold mt-0.5">{sub}</p>}
    </div>
  );
}
