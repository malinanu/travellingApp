'use client';

// Vendor directory — every registered vendor regardless of status or
// whether it has map coordinates, which the approval queue (pending only)
// and the geo map (coords only) each miss. Selecting a vendor opens the
// monitor pane: live activity stats plus the same admin actions the queue
// and geo pages expose.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  AlertTriangle, CheckCircle2, Clock, CreditCard, Edit3, Eye, FileText, Film, Globe, History,
  ImagePlus, Loader2, MapPin, Pencil, Phone, Power, Radio, Save, ShieldAlert, ShieldCheck, Star, Store, Tag,
  Ticket, Trash2, Users, Wifi, XCircle,
} from 'lucide-react';
import { useEffect, useMemo, useRef, useState } from 'react';

import {
  categoriesBySegment, VENDOR_SEGMENTS,
  resolveMediaUrl, timeAgo, uploadMedia, useAutoSave, validateFile, type AdminAuditLogEntry, type AdminVendorProfileGrant,
  type AdminVendorStats, type AdminVendorSummary, type AdminVendorTeam, type Claim, type Media,
  type MediaRole, type Offer, type Review, type Vendor, type VendorProfile, type VendorStatus, type VibeAggregate,
} from '@spotto/shared';
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

  // Network-wide roll-up for the aggregate band above the directory.
  const { data: summary } = useQuery<AdminVendorSummary>({
    queryKey: ['admin-vendor-summary'],
    queryFn: () => api.get<AdminVendorSummary>('/admin/vendors/summary'),
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
      qc.invalidateQueries({ queryKey: ['admin-vendor-summary'] });
      qc.invalidateQueries({ queryKey: ['admin-kpis'] });
    },
  });

  return (
    <Shell>
      {isError && <InlineErrorBanner message="Couldn't load the vendor directory — try refreshing." />}

      <div className="mb-4">
        <p className="text-[11.5px] font-bold text-muted uppercase tracking-wide mb-2">Vendor stats · network-wide</p>
        <div className="grid grid-cols-2 sm:grid-cols-4 xl:grid-cols-7 gap-3">
          <StatTile icon={Store} label="Vendors"
            value={summary ? `${summary.total}` : '—'}
            sub={summary ? `${summary.pending} pending · ${summary.approved} live · ${summary.suspended} susp.` : undefined} />
          <StatTile icon={Star} label="Avg rating"
            value={summary && summary.reviews > 0 ? summary.avgRating.toFixed(1) : '—'}
            sub={summary ? `${summary.reviews} reviews · ${summary.badReviews} bad` : undefined} />
          <StatTile icon={ShieldAlert} label="Flagged"
            value={summary ? `${summary.flagged}` : '—'}
            sub={summary ? 'vendors over threshold' : undefined}
            warn={!!summary && summary.flagged > 0} />
          <StatTile icon={Ticket} label="Redemptions"
            value={summary ? `${summary.redeemed}` : '—'}
            sub={summary ? `${summary.redeemed30d} in 30d` : undefined} />
          <StatTile icon={Users} label="Check-ins"
            value={summary ? `${summary.checkins}` : '—'}
            sub={summary ? `${summary.checkins30d} in 30d` : undefined} />
          <StatTile icon={Radio} label="Offers"
            value={summary ? `${summary.offersLive}` : '—'}
            sub={summary ? `live · ${summary.offersDraft} draft · ${summary.offers} total` : undefined} />
          <StatTile icon={CreditCard} label="Subscriptions"
            value={summary ? `${summary.activeSubs}` : '—'}
            sub={summary ? `$${summary.mrr}/mo MRR` : undefined} />
        </div>
      </div>

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
            {VENDOR_SEGMENTS.map((segment) => (
              <optgroup key={segment.id} label={segment.label}>
                {categoriesBySegment(segment.id).map((c) => (
                  <option key={c.id} value={c.id}>{c.label}</option>
                ))}
              </optgroup>
            ))}
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

type DetailTab = 'overview' | 'reviews' | 'offers' | 'activity' | 'team' | 'history';
const DETAIL_TABS: { id: DetailTab; label: string }[] = [
  { id: 'overview', label: 'Overview' },
  { id: 'reviews', label: 'Reviews' },
  { id: 'offers', label: 'Offers' },
  { id: 'activity', label: 'Activity' },
  { id: 'team', label: 'Team' },
  { id: 'history', label: 'History' },
];

function VendorMonitor({
  vendor,
  busy,
  onAct,
}: {
  vendor: Vendor;
  busy: boolean;
  onAct: (action: string) => void;
}) {
  const qc = useQueryClient();
  const [tab, setTab] = useState<DetailTab>('overview');
  const [editing, setEditing] = useState(false);
  const [takedownOpen, setTakedownOpen] = useState(false);

  // Reset transient view state when the selected vendor changes.
  useEffect(() => {
    setTab('overview');
    setEditing(false);
    setTakedownOpen(false);
  }, [vendor.id]);

  const { data: stats } = useQuery<AdminVendorStats>({
    queryKey: ['admin-vendor-stats', vendor.id],
    queryFn: () => api.get<AdminVendorStats>(`/admin/vendors/${vendor.id}/stats`),
  });

  const invalidateVendor = () => {
    qc.invalidateQueries({ queryKey: ['admin-vendors'] });
    qc.invalidateQueries({ queryKey: ['admin-vendor-summary'] });
    qc.invalidateQueries({ queryKey: ['admin-vendor-stats', vendor.id] });
  };

  const takedown = useMutation({
    mutationFn: (reason: string) => api.post(`/admin/vendors/${vendor.id}/takedown`, { reason }),
    onSuccess: () => {
      invalidateVendor();
      setTakedownOpen(false);
    },
  });

  const hasCoords = vendor.lat !== 0 || vendor.lng !== 0;
  const canTakedown = vendor.status !== 'suspended' && vendor.status !== 'rejected';

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

      {/* Admin controls: edit + emergency takedown, always available. */}
      <div className="flex items-center gap-2 mt-3">
        <button className="btn !py-1.5 flex items-center gap-1.5 text-[12.5px]"
          onClick={() => setEditing((e) => !e)}>
          <Pencil size={13} /> {editing ? 'Close editor' : 'Edit details'}
        </button>
        {canTakedown && (
          <button className="btn btn-danger !py-1.5 flex items-center gap-1.5 text-[12.5px]"
            onClick={() => setTakedownOpen(true)}>
            <Power size={13} /> Take offline
          </button>
        )}
      </div>

      {stats?.flagged && (
        <div className="rounded-xl p-4 mt-4" style={{ background: '#fde4d84d', border: '1px solid #e11d4855' }}>
          <p className="font-bold text-[13px] flex items-center gap-2" style={{ color: '#e11d48' }}>
            <ShieldAlert size={14} /> {stats.badReviews} bad reviews (1-2 stars) — flagged
          </p>
        </div>
      )}

      {editing ? (
        <EditForm vendor={vendor} onDone={() => setEditing(false)} onSaved={invalidateVendor} />
      ) : (
        <>
          <div className="flex items-center gap-1 mt-5 border-b border-line overflow-x-auto">
            {DETAIL_TABS.map((t) => (
              <button key={t.id} onClick={() => setTab(t.id)}
                className="px-3 py-2 text-[13px] font-bold whitespace-nowrap border-b-2 -mb-px transition-colors"
                style={{
                  borderColor: tab === t.id ? 'var(--accent)' : 'transparent',
                  color: tab === t.id ? 'var(--ink)' : 'var(--muted)',
                }}>
                {t.label}
              </button>
            ))}
          </div>

          <div className="mt-5">
            {tab === 'overview' && <OverviewTab vendor={vendor} stats={stats} />}
            {tab === 'reviews' && <ReviewsTab vendorId={vendor.id} />}
            {tab === 'offers' && <OffersTab vendorId={vendor.id} />}
            {tab === 'activity' && <ActivityTab vendorId={vendor.id} stats={stats} />}
            {tab === 'team' && <TeamTab vendorId={vendor.id} />}
            {tab === 'history' && <HistoryTab vendorId={vendor.id} />}
          </div>
        </>
      )}

      <div className="flex gap-3 mt-6">
        <a
          href={`${process.env.NEXT_PUBLIC_VENDOR_URL}/dashboard?impersonate=${vendor.id}`}
          target="_blank"
          rel="noreferrer"
          className="btn py-2.5 flex items-center justify-center gap-2 text-accent font-extrabold"
        >
          <Eye size={16} /> View as vendor
        </a>
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
          Suspended vendors can&apos;t log into the vendor portal until reinstated, and their listing is hidden from travellers.
        </p>
      )}

      {takedownOpen && (
        <TakedownModal
          busy={takedown.isPending}
          error={takedown.isError ? 'Takedown failed — try again.' : null}
          onCancel={() => setTakedownOpen(false)}
          onConfirm={(reason) => takedown.mutate(reason)}
        />
      )}
    </div>
  );
}

function OverviewTab({ vendor, stats }: { vendor: Vendor; stats?: AdminVendorStats }) {
  const p: VendorProfile = vendor.profile ?? {};
  const contact = p.contact ?? {};
  const menu = vendor.menu;
  const activePreset = menu?.presets?.find((x) => x.id === menu.activePresetId) ?? menu?.presets?.[0];
  const menuItems = (activePreset?.sections ?? []).reduce((a, s) => a + s.items.length, 0);
  return (
    <>
      <div className="grid grid-cols-4 gap-3">
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
        <StatTile icon={Tag} label="Priority / assets"
          value={vendor.priority || 'normal'}
          sub={`${vendor.docs ? 'docs ✓' : 'no docs'} · ${vendor.photos} photos${
            vendor.brCertVerifyStatus === 'verified' ? ' · BR ✓' :
            vendor.brCertVerifyStatus === 'invalid' ? ' · BR ✗' : ''}`} />
      </div>

      <div className="grid grid-cols-2 gap-3 mt-4">
        <InfoCard title="Contact">
          <InfoRow icon={Phone} label="Phone" value={contact.phone} />
          <InfoRow icon={Globe} label="Website" value={contact.website} />
          <InfoRow icon={Users} label="Instagram" value={contact.instagram} />
          <InfoRow icon={Users} label="Facebook" value={contact.facebook} />
        </InfoCard>
        <InfoCard title="Amenities & vibe">
          <InfoRow icon={Wifi} label="Wi-Fi"
            value={p.wifi?.enabled ? `score ${p.wifi.score}` : 'not offered'} />
          <InfoRow icon={Star} label="Sea view" value={p.seaView ? 'yes' : 'no'} />
          <InfoRow icon={Clock} label="Best times" value={(p.bestTimes ?? []).join(', ')} />
          <InfoRow icon={Tag} label="Amenities" value={(p.amenities ?? []).join(', ')} />
        </InfoCard>
      </div>

      {p.blurb && (
        <div className="rounded-xl p-4 mt-3" style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
          <p className="text-[11.5px] font-bold text-muted uppercase mb-1">Blurb</p>
          <p className="text-[13.5px] font-medium leading-relaxed">{p.blurb}</p>
        </div>
      )}

      <div className="rounded-xl p-4 mt-3" style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
        <p className="text-[11.5px] font-bold text-muted uppercase mb-1">Active menu</p>
        <p className="text-[13.5px] font-semibold">
          {menu?.presets?.length
            ? `${activePreset?.name ?? 'Menu'} · ${activePreset?.sections.length ?? 0} sections · ${menuItems} items`
            : 'No menu set.'}
        </p>
      </div>

      <VendorMediaPanel vendor={vendor} />

      {vendor.flags && vendor.flags.length > 0 && (
        <div className="rounded-xl p-4 mt-3" style={{ background: '#fdf0cd4d', border: '1px solid #f6b81e55' }}>
          <p className="font-bold text-[13px]" style={{ color: '#a97b00' }}>Flags</p>
          <ul className="mt-1.5 space-y-1">
            {vendor.flags.map((f) => <li key={f} className="text-[13px] font-medium">{f}</li>)}
          </ul>
        </div>
      )}

      {vendor.note && (
        <div className="rounded-xl p-4 mt-3" style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
          <p className="text-[11.5px] font-bold text-muted uppercase mb-1">Internal note</p>
          <p className="text-[13.5px] font-medium leading-relaxed">{vendor.note}</p>
        </div>
      )}
    </>
  );
}

function ReviewsTab({ vendorId }: { vendorId: string }) {
  const qc = useQueryClient();
  const { data: reviews = [], isError, isLoading } = useQuery<Review[]>({
    queryKey: ['admin-vendor-reviews', vendorId],
    queryFn: () => api.get<Review[]>(`/admin/vendors/${vendorId}/reviews?all=1`),
  });

  const deleteReview = useMutation({
    mutationFn: (id: string) => api.del(`/admin/reviews/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin-vendor-reviews', vendorId] }),
  });

  if (isError) return <InlineErrorBanner message="Couldn't load reviews." />;
  if (isLoading) return <TabEmpty label="Loading reviews…" />;
  if (reviews.length === 0) return <TabEmpty label="No reviews yet." />;

  const dist = [5, 4, 3, 2, 1].map((r) => ({ r, n: reviews.filter((x) => x.rating === r).length }));
  const max = Math.max(1, ...dist.map((d) => d.n));
  return (
    <>
      <div className="rounded-xl p-4 mb-4" style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
        {dist.map((d) => (
          <div key={d.r} className="flex items-center gap-2 mb-1 last:mb-0">
            <span className="text-[12px] font-bold w-8 flex items-center gap-0.5">{d.r}<Star size={11} /></span>
            <div className="flex-1 h-2 rounded-full" style={{ background: 'var(--line)' }}>
              <div className="h-2 rounded-full" style={{ width: `${(d.n / max) * 100}%`, background: 'var(--accent)' }} />
            </div>
            <span className="text-[12px] font-semibold text-muted w-8 text-right">{d.n}</span>
          </div>
        ))}
      </div>
      <div className="space-y-3">
        {reviews.map((rv) => (
          <div key={rv.id} className="rounded-xl p-3.5" style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
            <div className="flex items-center justify-between">
              <p className="font-bold text-[13px]">{rv.travellerName || 'Traveller'}</p>
              <div className="flex items-center gap-3">
                <span className="flex items-center gap-0.5 text-[12.5px] font-bold"
                  style={{ color: rv.rating <= 2 ? '#e11d48' : '#a97b00' }}>
                  {rv.rating}<Star size={12} />
                </span>
                <button
                  onClick={() => deleteReview.mutate(rv.id)}
                  disabled={deleteReview.isPending}
                  className="p-1 hover:text-coral transition-colors text-muted"
                  aria-label="Delete review"
                >
                  <Trash2 size={14} />
                </button>
              </div>
            </div>
            {rv.text && <p className="text-[13px] font-medium mt-1 leading-relaxed">{rv.text}</p>}
            <p className="text-[11px] text-muted font-semibold mt-1">{timeAgo(rv.createdAt)}</p>
          </div>
        ))}
      </div>
    </>
  );
}

function OffersTab({ vendorId }: { vendorId: string }) {
  const { data: offers = [], isError, isLoading } = useQuery<Offer[]>({
    queryKey: ['admin-vendor-offers', vendorId],
    queryFn: () => api.get<Offer[]>(`/admin/vendors/${vendorId}/offers`),
  });
  if (isError) return <InlineErrorBanner message="Couldn't load offers." />;
  if (isLoading) return <TabEmpty label="Loading offers…" />;
  if (offers.length === 0) return <TabEmpty label="No offers created yet." />;
  return (
    <div className="panel overflow-hidden">
      <table className="w-full text-[13px]">
        <thead style={{ background: 'var(--bg)' }}>
          <tr className="text-left border-b border-line text-[11px] uppercase text-muted">
            <th className="px-3 py-2">Offer</th><th className="px-3 py-2">Status</th>
            <th className="px-3 py-2 text-right">Claimed</th><th className="px-3 py-2 text-right">Redeemed</th>
            <th className="px-3 py-2 text-right">Views</th>
          </tr>
        </thead>
        <tbody>
          {offers.map((o) => (
            <tr key={o.id} className="border-b border-line last:border-0">
              <td className="px-3 py-2.5">
                <p className="font-bold">{o.title}</p>
                <p className="text-[11.5px] text-muted font-semibold">{o.type} · {o.value}</p>
              </td>
              <td className="px-3 py-2.5"><span className="text-[11.5px] font-bold uppercase">{o.status}</span></td>
              <td className="px-3 py-2.5 text-right font-semibold">{o.claimed}</td>
              <td className="px-3 py-2.5 text-right font-semibold">{o.redeemed}</td>
              <td className="px-3 py-2.5 text-right font-semibold flex items-center justify-end gap-1"><Eye size={12} />{o.views}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function ActivityTab({ vendorId, stats }: { vendorId: string; stats?: AdminVendorStats }) {
  const qc = useQueryClient();
  const { data: claims = [], isError, isLoading } = useQuery<Claim[]>({
    queryKey: ['admin-vendor-redemptions', vendorId],
    queryFn: () => api.get<Claim[]>(`/admin/vendors/${vendorId}/redemptions`),
  });

  const voidClaim = useMutation({
    mutationFn: (id: string) => api.post(`/admin/claims/${id}/void`, {}),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin-vendor-redemptions', vendorId] }),
  });

  return (
    <>
      <div className="grid grid-cols-3 gap-3 mb-4">
        <StatTile icon={Ticket} label="Redeemed (all)" value={stats ? `${stats.redeemed}` : '—'} />
        <StatTile icon={Ticket} label="Redeemed (30d)" value={stats ? `${stats.redeemed30d}` : '—'} />
        <StatTile icon={Users} label="Check-ins (30d)" value={stats ? `${stats.checkins30d}` : '—'} />
      </div>
      {isError && <InlineErrorBanner message="Couldn't load recent redemptions." />}
      {isLoading && <TabEmpty label="Loading activity…" />}
      {!isLoading && claims.length === 0 && <TabEmpty label="No redemptions yet." />}
      <div className="space-y-2">
        {claims.map((c) => (
          <div key={c.id} className="flex items-center justify-between rounded-xl px-3.5 py-2.5"
            style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
            <div>
              <p className="font-bold text-[13px]">{c.travellerName || 'Traveller'}
                {c.travellerCountry ? <span className="text-muted font-semibold"> · {c.travellerCountry}</span> : null}</p>
              <p className="text-[11.5px] text-muted font-semibold">{c.offerLabel || c.code}</p>
            </div>
            <div className="flex items-center gap-3">
              <p className="text-[11.5px] text-muted font-semibold">{c.redeemedAt ? timeAgo(c.redeemedAt) : ''}</p>
              <button
                onClick={() => voidClaim.mutate(c.id)}
                disabled={voidClaim.isPending}
                className="text-[11px] font-extrabold text-coral border border-coral/30 hover:bg-coral/10 rounded-lg px-2 py-1 transition-colors"
              >
                Void
              </button>
            </div>
          </div>
        ))}
      </div>
    </>
  );
}

function TeamTab({ vendorId }: { vendorId: string }) {
  const { data, isError, isLoading } = useQuery<AdminVendorTeam>({
    queryKey: ['admin-vendor-team', vendorId],
    queryFn: () => api.get<AdminVendorTeam>(`/admin/vendors/${vendorId}/team`),
  });
  if (isError) return <InlineErrorBanner message="Couldn't load the team." />;
  if (isLoading || !data) return <TabEmpty label="Loading team…" />;
  return (
    <div className="space-y-4">
      <div>
        <p className="text-[11.5px] font-bold text-muted uppercase mb-2">Members ({data.members.length})</p>
        <div className="space-y-2">
          {data.members.map((m) => (
            <div key={m.id} className="flex items-center justify-between rounded-xl px-3.5 py-2.5"
              style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
              <div>
                <p className="font-bold text-[13px]">{m.name || m.email}</p>
                <p className="text-[11.5px] text-muted font-semibold">{m.email}</p>
              </div>
              <span className="text-[11px] font-bold uppercase rounded-full px-2 py-0.5"
                style={{ background: 'var(--surface)', border: '1px solid var(--line)' }}>{m.vendorRole || '—'}</span>
            </div>
          ))}
          {data.members.length === 0 && <TabEmpty label="No members." />}
        </div>
      </div>
      {data.invites.length > 0 && (
        <div>
          <p className="text-[11.5px] font-bold text-muted uppercase mb-2">Pending invites ({data.invites.length})</p>
          <div className="space-y-2">
            {data.invites.map((iv) => (
              <div key={iv.id} className="flex items-center justify-between rounded-xl px-3.5 py-2.5"
                style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
                <p className="font-semibold text-[13px]">{iv.email || iv.code} <span className="text-muted">· {iv.role}</span></p>
                <p className="text-[11.5px] text-muted font-semibold">expires {timeAgo(iv.expiresAt)}</p>
              </div>
            ))}
          </div>
        </div>
      )}
      {data.grants.length > 0 && (
        <div>
          <p className="text-[11.5px] font-bold text-muted uppercase mb-2">Access grants ({data.grants.length})</p>
          <div className="space-y-2">
            {data.grants.map((g: AdminVendorProfileGrant) => (
              <div key={g.id} className="flex items-center justify-between rounded-xl px-3.5 py-2.5"
                style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
                <p className="font-semibold text-[13px]">{g.granteeName} <span className="text-muted">· by {g.ownerName}</span></p>
                <span className="text-[11px] font-bold uppercase" style={{ color: '#16a34a' }}>{g.status}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

function HistoryTab({ vendorId }: { vendorId: string }) {
  const { data: entries = [], isError, isLoading } = useQuery<AdminAuditLogEntry[]>({
    queryKey: ['admin-vendor-audit', vendorId],
    queryFn: () => api.get<AdminAuditLogEntry[]>(`/admin/vendors/${vendorId}/audit`),
  });
  if (isError) return <InlineErrorBanner message="Couldn't load history." />;
  if (isLoading) return <TabEmpty label="Loading history…" />;
  if (entries.length === 0) return <TabEmpty label="No admin actions on this vendor yet." />;
  return (
    <div className="space-y-2">
      {entries.map((e) => (
        <div key={e.id} className="flex items-start gap-3 rounded-xl px-3.5 py-2.5"
          style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
          <History size={14} className="mt-0.5 shrink-0 text-muted" />
          <div className="min-w-0 flex-1">
            <p className="font-bold text-[13px]">{e.action}</p>
            <p className="text-[11.5px] text-muted font-semibold">
              {e.actorName || e.actorEmail} · {timeAgo(e.createdAt)}
              {e.metadata ? ` · ${JSON.stringify(e.metadata)}` : ''}
            </p>
          </div>
        </div>
      ))}
    </div>
  );
}

function EditForm({ vendor, onDone, onSaved }: { vendor: Vendor; onDone: () => void; onSaved: () => void }) {
  // Lazy per-vendor detail, same convention as VendorMonitor's stats query —
  // not attached to the bulk vendor list, so opening the edit form is the
  // only thing that fetches it.
  const { data: vibe } = useQuery<VibeAggregate>({
    queryKey: ['admin-vendor-vibe', vendor.id],
    queryFn: () => api.get<VibeAggregate>(`/admin/vendors/${vendor.id}/vibe`),
  });
  const p: VendorProfile = vendor.profile ?? {};
  const c = p.contact ?? {};
  const [f, setF] = useState({
    name: vendor.name, category: vendor.category, town: vendor.town, country: vendor.country,
    priority: vendor.priority || 'normal', note: vendor.note ?? '', flags: (vendor.flags ?? []).join(', '),
    blurb: p.blurb ?? '', amenities: (p.amenities ?? []).join(', '), seaView: !!p.seaView,
    wifiEnabled: !!p.wifi?.enabled, wifiScore: p.wifi?.score ?? 0,
    phone: c.phone ?? '', website: c.website ?? '', instagram: c.instagram ?? '', facebook: c.facebook ?? '',
  });
  const [dirty, setDirty] = useState(false);
  const set = (k: keyof typeof f, v: string | number | boolean) => {
    setF((prev) => ({ ...prev, [k]: v }));
    setDirty(true);
  };

  const save = useMutation({
    mutationFn: (_vars: { closeAfter: boolean }) => {
      const list = (s: string) => s.split(',').map((x) => x.trim()).filter(Boolean);
      const profile: VendorProfile = {
        ...p,
        blurb: f.blurb,
        seaView: f.seaView,
        amenities: list(f.amenities),
        contact: { phone: f.phone, website: f.website, instagram: f.instagram, facebook: f.facebook },
        wifi: f.wifiEnabled
          ? { enabled: true, score: Number(f.wifiScore) || 0, speed: p.wifi?.speed ?? 0 }
          : { enabled: false, score: p.wifi?.score ?? 0, speed: p.wifi?.speed ?? 0 },
      };
      return api.patch(`/admin/vendors/${vendor.id}`, {
        name: f.name, category: f.category, town: f.town, country: f.country,
        priority: f.priority, note: f.note, flags: list(f.flags), profile,
      });
    },
    onSuccess: (_data, vars) => {
      setDirty(false);
      onSaved();
      // Only the explicit "Save changes" button closes the form back to
      // view mode — a background auto-save mid-edit must leave it open, or
      // the form would collapse out from under whoever's still editing it.
      if (vars.closeAfter) onDone();
    },
  });

  // Auto-save: debounced while typing, plus an immediate flush() wired to
  // onBlur and the manual "Save changes" button (kept as a fallback).
  const autoSave = useAutoSave({
    deps: [f],
    dirty,
    onSave: () => save.mutate({ closeAfter: false }),
  });

  const input = 'rounded-lg px-3 py-2 text-[13px] font-medium outline-none w-full';
  const inputStyle = { background: 'var(--bg)', border: '1px solid var(--line)' } as const;

  return (
    <div className="mt-5 rounded-xl p-4" style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
      <p className="text-[11.5px] font-bold text-muted uppercase mb-3">Admin edit — applies immediately, even for suspended vendors</p>
      <div className="grid grid-cols-2 gap-3">
        <Field label="Name">
          <input className={input} style={inputStyle} value={f.name} onChange={(e) => set('name', e.target.value)} onBlur={autoSave.flush} />
          {!f.name.trim() && <span className="text-[11px] font-extrabold text-coral mt-1 block">Name is required</span>}
        </Field>
        <Field label="Category">
          <select className={input} style={inputStyle} value={f.category} onChange={(e) => { set('category', e.target.value); autoSave.flush(); }}>
            {VENDOR_SEGMENTS.map((segment) => (
              <optgroup key={segment.id} label={segment.label}>
                {categoriesBySegment(segment.id).map((c) => (
                  <option key={c.id} value={c.id}>{c.label}</option>
                ))}
              </optgroup>
            ))}
          </select>
          <span className="text-[11px] font-semibold text-muted mt-1 block">
            {vibe && vibe.sampleSize > 0
              ? `Live vibe: ${vibe.productive.toFixed(1)} productive · ${vibe.loud.toFixed(1)} loud · ${vibe.crowded.toFixed(1)} crowded (${vibe.sampleSize} traveler${vibe.sampleSize === 1 ? '' : 's'})`
              : 'Live vibe: no check-ins yet'}
          </span>
        </Field>
        <Field label="Town">
          <input className={input} style={inputStyle} value={f.town} onChange={(e) => set('town', e.target.value)} onBlur={autoSave.flush} />
          {!f.town.trim() && <span className="text-[11px] font-extrabold text-coral mt-1 block">Town is required</span>}
        </Field>
        <Field label="Country"><input className={input} style={inputStyle} value={f.country} onChange={(e) => set('country', e.target.value)} onBlur={autoSave.flush} /></Field>
        <Field label="Priority">
          <select className={input} style={inputStyle} value={f.priority} onChange={(e) => set('priority', e.target.value)}>
            <option value="high">high</option><option value="normal">normal</option><option value="low">low</option>
          </select>
        </Field>
        <Field label="Flags (comma-separated)"><input className={input} style={inputStyle} value={f.flags} onChange={(e) => set('flags', e.target.value)} onBlur={autoSave.flush} /></Field>
        <Field label="Phone"><input className={input} style={inputStyle} value={f.phone} onChange={(e) => set('phone', e.target.value)} onBlur={autoSave.flush} /></Field>
        <Field label="Website"><input className={input} style={inputStyle} value={f.website} onChange={(e) => set('website', e.target.value)} onBlur={autoSave.flush} /></Field>
        <Field label="Instagram"><input className={input} style={inputStyle} value={f.instagram} onChange={(e) => set('instagram', e.target.value)} onBlur={autoSave.flush} /></Field>
        <Field label="Facebook"><input className={input} style={inputStyle} value={f.facebook} onChange={(e) => set('facebook', e.target.value)} onBlur={autoSave.flush} /></Field>
        <Field label="Amenities (comma-separated)"><input className={input} style={inputStyle} value={f.amenities} onChange={(e) => set('amenities', e.target.value)} onBlur={autoSave.flush} /></Field>
        <Field label="Wi-Fi score (if offered)"><input className={input} style={inputStyle} inputMode="numeric" value={f.wifiScore} onChange={(e) => set('wifiScore', Number(e.target.value) || 0)} onBlur={autoSave.flush} /></Field>
      </div>
      <div className="flex items-center gap-4 mt-3">
        <label className="flex items-center gap-2 text-[13px] font-semibold">
          <input type="checkbox" checked={f.wifiEnabled} onChange={(e) => set('wifiEnabled', e.target.checked)} /> Wi-Fi offered
        </label>
        <label className="flex items-center gap-2 text-[13px] font-semibold">
          <input type="checkbox" checked={f.seaView} onChange={(e) => set('seaView', e.target.checked)} /> Sea view
        </label>
      </div>
      <Field label="Blurb" className="mt-3">
        <textarea className={input} style={inputStyle} rows={2} value={f.blurb} onChange={(e) => set('blurb', e.target.value)} onBlur={autoSave.flush} />
      </Field>
      <Field label="Internal note" className="mt-3">
        <textarea className={input} style={inputStyle} rows={2} value={f.note} onChange={(e) => set('note', e.target.value)} onBlur={autoSave.flush} />
      </Field>
      <p className="text-[11.5px] text-muted font-semibold mt-2">
        Opening hours and the full menu are edited from the vendor portal.
      </p>
      {save.isError && <p className="text-[13px] font-bold text-coral mt-2">Save failed — try again.</p>}
      <div className="flex justify-end gap-2 mt-4">
        <button className="btn !py-2" onClick={onDone}>Cancel</button>
        <button className="btn btn-primary !py-2 flex items-center gap-2" disabled={save.isPending} onClick={() => save.mutate({ closeAfter: true })}>
          <Save size={15} /> {save.isPending ? 'Saving…' : 'Save changes'}
        </button>
      </div>
    </div>
  );
}

function TakedownModal({
  busy, error, onCancel, onConfirm,
}: {
  busy: boolean; error: string | null; onCancel: () => void; onConfirm: (reason: string) => void;
}) {
  const [reason, setReason] = useState('');
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4" style={{ background: 'rgba(0,0,0,0.45)' }}>
      <div className="panel p-6 w-full max-w-md">
        <p className="text-lg font-bold flex items-center gap-2" style={{ color: '#e11d48' }}>
          <Power size={18} /> Take vendor offline
        </p>
        <p className="text-[13px] text-muted font-medium mt-1">
          This suspends the vendor and immediately removes their listing from the traveller app. A reason is required and recorded in the audit log.
        </p>
        <textarea
          autoFocus rows={3}
          className="rounded-lg px-3 py-2 text-[13px] font-medium outline-none w-full mt-3"
          style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}
          placeholder="Reason (e.g. reported for fraudulent offers)"
          value={reason} onChange={(e) => setReason(e.target.value)}
        />
        {error && <p className="text-[13px] font-bold text-coral mt-2">{error}</p>}
        <div className="flex justify-end gap-2 mt-4">
          <button className="btn !py-2" onClick={onCancel}>Cancel</button>
          <button className="btn btn-danger !py-2 flex items-center gap-2"
            disabled={busy || !reason.trim()} onClick={() => onConfirm(reason.trim())}>
            <Power size={15} /> {busy ? 'Taking offline…' : 'Take offline'}
          </button>
        </div>
      </div>
    </div>
  );
}

function InfoCard({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="rounded-xl p-4" style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
      <p className="text-[11.5px] font-bold text-muted uppercase mb-2">{title}</p>
      <div className="space-y-1.5">{children}</div>
    </div>
  );
}

function InfoRow({ icon: Icon, label, value }: { icon: typeof Phone; label: string; value?: string }) {
  return (
    <div className="flex items-center gap-2 text-[13px]">
      <Icon size={13} className="text-muted shrink-0" />
      <span className="text-muted font-semibold w-20 shrink-0">{label}</span>
      <span className="font-medium truncate">{value || '—'}</span>
    </div>
  );
}

function Field({ label, children, className }: { label: string; children: React.ReactNode; className?: string }) {
  return (
    <label className={`block ${className ?? ''}`}>
      <span className="text-[11px] font-bold text-muted uppercase block mb-1">{label}</span>
      {children}
    </label>
  );
}

function TabEmpty({ label }: { label: string }) {
  return <p className="px-4 py-8 text-center text-muted font-semibold text-[13px]">{label}</p>;
}

// VendorMediaPanel lets an admin view, add, and remove a vendor's uploaded
// media (cover / gallery / promo video / verification docs) — the override
// path, so it works regardless of vendor status.
function VendorMediaPanel({ vendor }: { vendor: Vendor }) {
  const vendorId = vendor.id;
  const qc = useQueryClient();
  const { data: media = [] } = useQuery<Media[]>({
    queryKey: ['admin-vendor-media', vendorId],
    queryFn: () => api.get<Media[]>(`/admin/vendors/${vendorId}/media`),
  });
  const [busy, setBusy] = useState<MediaRole | null>(null);
  const [error, setError] = useState<string | null>(null);

  const cover = media.find((m) => m.role === 'cover');
  const gallery = media.filter((m) => m.role === 'gallery');
  const promo = media.find((m) => m.role === 'promo');
  const docs = media.filter((m) => m.role === 'doc');
  const brCert = media.find((m) => m.role === 'br_cert');

  const invalidate = () => qc.invalidateQueries({ queryKey: ['admin-vendor-media', vendorId] });
  const add = async (file: File | undefined, role: MediaRole, kind: 'image' | 'video' | 'doc') => {
    if (!file) return;
    const invalid = validateFile(file, kind);
    if (invalid) { setError(invalid); return; }
    setError(null);
    setBusy(role);
    try {
      await uploadMedia(api, file, { ownerType: 'vendor', ownerId: vendorId, role });
      invalidate();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Upload failed.');
    } finally {
      setBusy(null);
    }
  };
  const remove = useMutation({
    mutationFn: (id: string) => api.del(`/media/${id}`),
    onSuccess: invalidate,
  });

  return (
    <div className="rounded-xl p-4 mt-3" style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
      <p className="text-[11.5px] font-bold text-muted uppercase mb-2">Media</p>
      {error && <p className="text-[12.5px] font-bold text-coral mb-2">{error}</p>}

      {vendor.brCertNumber && (
        <div className="mb-3 rounded-lg p-2.5 text-[12.5px]" style={{ background: 'var(--sand, #faf7f2)' }}>
          <p className="font-extrabold">BR cert: {vendor.brCertNumber} — {vendor.brCertVerifyStatus || 'not yet checked'}</p>
          {vendor.brCertCompanyName && <p className="text-muted font-semibold">eROC name: {vendor.brCertCompanyName}</p>}
          {vendor.brCertMessage && <p className="text-coral font-semibold">{vendor.brCertMessage}</p>}
        </div>
      )}

      <div className="flex flex-wrap gap-2 items-start">
        {cover && <MediaThumb m={cover} label="Cover" onDelete={() => remove.mutate(cover.id)} />}
        {gallery.map((m) => <MediaThumb key={m.id} m={m} onDelete={() => remove.mutate(m.id)} />)}
        {promo && <MediaThumb m={promo} label="Promo" video onDelete={() => remove.mutate(promo.id)} />}
        {brCert && <MediaThumb m={brCert} label="BR cert" doc onDelete={() => remove.mutate(brCert.id)} />}
        {docs.map((m) => <MediaThumb key={m.id} m={m} label="Doc" doc onDelete={() => remove.mutate(m.id)} />)}
      </div>
      <div className="flex flex-wrap gap-2 mt-3">
        <AdminUploadBtn label="Cover" icon={ImagePlus} busy={busy === 'cover'} accept="image/*" onPick={(f) => add(f, 'cover', 'image')} />
        <AdminUploadBtn label="Photo" icon={ImagePlus} busy={busy === 'gallery'} accept="image/*" onPick={(f) => add(f, 'gallery', 'image')} />
        <AdminUploadBtn label="Video" icon={Film} busy={busy === 'promo'} accept="video/*" onPick={(f) => add(f, 'promo', 'video')} />
        <AdminUploadBtn label="BR cert" icon={ShieldCheck} busy={busy === 'br_cert'} accept="application/pdf,image/*" onPick={(f) => add(f, 'br_cert', 'doc')} />
        <AdminUploadBtn label="Doc" icon={FileText} busy={busy === 'doc'} accept="application/pdf,image/*" onPick={(f) => add(f, 'doc', 'doc')} />
      </div>
    </div>
  );
}

function MediaThumb({ m, label, video, doc, onDelete }: { m: Media; label?: string; video?: boolean; doc?: boolean; onDelete: () => void }) {
  return (
    <div className="relative w-16 h-16 rounded-lg overflow-hidden group" style={{ border: '1px solid var(--line)' }}>
      {video ? (
        <video src={resolveMediaUrl(m.url)} className="w-full h-full object-cover" />
      ) : doc ? (
        <a href={resolveMediaUrl(m.url)} target="_blank" rel="noreferrer" className="w-full h-full flex items-center justify-center" style={{ background: 'var(--surface)' }}>
          <FileText size={22} className="text-muted" />
        </a>
      ) : (
        // eslint-disable-next-line @next/next/no-img-element
        <img src={resolveMediaUrl(m.thumbUrl || m.url)} alt="" className="w-full h-full object-cover" />
      )}
      {label && <span className="absolute left-1 bottom-1 text-[9px] font-bold px-1 rounded bg-black/50 text-white">{label}</span>}
      <button onClick={onDelete} aria-label="Delete"
        className="absolute right-0.5 top-0.5 rounded-full p-0.5 bg-black/50 text-white opacity-0 group-hover:opacity-100 transition-opacity">
        <Trash2 size={11} />
      </button>
    </div>
  );
}

function AdminUploadBtn({ label, icon: Icon, busy, accept, onPick }: { label: string; icon: typeof ImagePlus; busy: boolean; accept: string; onPick: (f?: File) => void }) {
  const ref = useRef<HTMLInputElement>(null);
  return (
    <button type="button" disabled={busy} onClick={() => ref.current?.click()}
      className="btn !py-1.5 flex items-center gap-1.5 text-[12px]">
      {busy ? <Loader2 size={13} className="animate-spin" /> : <Icon size={13} />} {label}
      <input ref={ref} type="file" accept={accept} className="hidden"
        onChange={(e) => { onPick(e.target.files?.[0]); e.currentTarget.value = ''; }} />
    </button>
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
