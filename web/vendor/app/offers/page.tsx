'use client';

// Promotions Hub — unified destination for all vendor customer acquisition & loyalty tools:
// 1. Store Deals (Always-on vouchers, discounts, bundles on profile & map)
// 2. Flash Beacon (Live 3km radius push blast for slow dead hours)
// 3. Loyalty & Stamps (Digital stamp cards & repeat visitor tracking)

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  ArrowRight,
  BadgeDollarSign,
  Calendar,
  Check,
  Coffee,
  EyeOff,
  Flame,
  Gift,
  Infinity as InfinityIcon,
  Package,
  Pencil,
  Percent,
  PowerOff,
  Radio,
  Rocket,
  RotateCcw,
  Sparkles,
  Star,
  Tag,
  Trash2,
  UserPlus,
  X,
  Zap,
  LucideIcon,
} from 'lucide-react';
import { useRouter, useSearchParams } from 'next/navigation';
import { Suspense, useEffect, useRef, useState } from 'react';

import type { GrowthBlob, Offer, OfferStatus } from '@spotto/shared';
import { useAutoSave } from '@spotto/shared';
import Shell, { useBusiness, useMe } from '../../components/shell';
import OfferDetailDrawer from '../../components/offer-detail-drawer';
import { api } from '../../lib/api';
import { StatusPill, TYPE_ICON } from '../../lib/offer-ui';

const DISCOUNT_TYPES: { id: string; apiType: string; label: string; icon: LucideIcon }[] = [
  { id: 'pct', apiType: 'Percentage', label: 'Percentage off', icon: Percent },
  { id: 'amount', apiType: 'Amount', label: 'Amount off', icon: BadgeDollarSign },
  { id: 'freebie', apiType: 'Freebie', label: 'Freebie', icon: Gift },
  { id: 'bundle', apiType: 'Bundle', label: 'Bundle', icon: Package },
  { id: 'event', apiType: 'Event', label: 'Event / Party', icon: Calendar },
];

const DURATIONS = [
  { id: 'next2h', label: 'Next 2 hours' },
  { id: 'today', label: 'Today' },
  { id: 'weekend', label: 'This weekend' },
  { id: 'custom', label: 'Custom' },
] as const;
type DurationId = (typeof DURATIONS)[number]['id'];

type FilterId = 'all' | 'live' | 'draft' | 'ended';
const FILTERS: { id: FilterId; label: string }[] = [
  { id: 'all', label: 'All' },
  { id: 'live', label: 'Live' },
  { id: 'draft', label: 'Draft' },
  { id: 'ended', label: 'Ended' },
];

const FLASH_DISCOUNTS = [20, 30, 50];

function nowHHMM(offsetMin = 0) {
  const d = new Date(Date.now() + offsetMin * 60_000);
  return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`;
}

function to12h(hhmm: string) {
  const [h, m] = hhmm.split(':').map(Number);
  const ap = h >= 12 ? 'PM' : 'AM';
  const h12 = h % 12 === 0 ? 12 : h % 12;
  return `${h12}:${String(m).padStart(2, '0')} ${ap}`;
}

function valueFor(typeId: string, percent: string, amount: string, freeText: string, bundleText: string) {
  switch (typeId) {
    case 'pct':
      return `${percent || 0}% off`;
    case 'amount':
      return `Rs ${amount || 0} off`;
    case 'bundle':
      return bundleText || 'Bundle deal';
    default:
      return freeText || 'Free item';
  }
}

function parseValue(typeId: string, value: string) {
  const num = value.match(/\d+/)?.[0] ?? '';
  if (typeId === 'pct' || typeId === 'amount') return num;
  return value;
}

export default function OffersPage() {
  return (
    <Suspense fallback={null}>
      <OffersPageInner />
    </Suspense>
  );
}

function OffersPageInner() {
  const router = useRouter();
  const qc = useQueryClient();
  const searchParams = useSearchParams();
  const q = (searchParams.get('q') ?? '').trim().toLowerCase();
  const initialTab = searchParams.get('tab');
  const [activeTab, setActiveTab] = useState<'deals' | 'flash' | 'loyalty'>(
    initialTab === 'flash' || initialTab === 'loyalty' ? initialTab : 'deals'
  );

  const { data: me } = useMe();
  const isStaff = me?.vendorRole === 'staff';
  const { data: business } = useBusiness();

  // Offers Query
  const { data: offers = [] } = useQuery<Offer[]>({
    queryKey: ['offers'],
    queryFn: () => api.get<Offer[]>('/vendor/offers'),
  });

  // Growth & Flash Tools Query
  const { data: growth } = useQuery<GrowthBlob>({
    queryKey: ['growth'],
    queryFn: () => api.get<GrowthBlob>('/vendor/growth'),
    enabled: !isStaff,
  });

  const [detailOfferId, setDetailOfferId] = useState<string | null>(null);
  const detailOffer = offers.find((o) => o.id === detailOfferId) ?? null;

  const [filter, setFilter] = useState<FilterId>('all');
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editingStatus, setEditingStatus] = useState<'live' | 'draft' | null>(null);
  const [dirty, setDirty] = useState(false);
  const skipNextDirtyRef = useRef(false);

  // Form State
  const [typeId, setTypeId] = useState('pct');
  const [title, setTitle] = useState('');
  const [percent, setPercent] = useState('20');
  const [amount, setAmount] = useState('500');
  const [freeText, setFreeText] = useState('');
  const [bundleText, setBundleText] = useState('');
  const [duration, setDuration] = useState<DurationId>('next2h');
  const [startTime, setStartTime] = useState(nowHHMM());
  const [endTime, setEndTime] = useState(nowHHMM(120));
  const [limit, setLimit] = useState(50);
  const [unlimited, setUnlimited] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Flash Sale State
  const [flashDiscount, setFlashDiscount] = useState(20);
  const [nudged, setNudged] = useState(false);

  const sendFlashSale = useMutation({
    mutationFn: () => api.post('/vendor/growth/flash-sale', { discountPct: flashDiscount }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['growth'] }),
  });
  const flashDisabled = sendFlashSale.isPending || (growth?.flash?.lastSentMins ?? Infinity) < 180;

  const resetForm = () => {
    skipNextDirtyRef.current = true;
    setEditingId(null);
    setEditingStatus(null);
    setDirty(false);
    setTypeId('pct');
    setTitle('');
    setPercent('20');
    setAmount('500');
    setFreeText('');
    setBundleText('');
    setDuration('next2h');
    setStartTime(nowHHMM());
    setEndTime(nowHHMM(120));
    setLimit(50);
    setUnlimited(false);
    setError(null);
  };

  const applyDuration = (id: DurationId) => {
    setDuration(id);
    if (id === 'next2h') {
      setStartTime(nowHHMM());
      setEndTime(nowHHMM(120));
    } else if (id === 'today' || id === 'weekend') {
      setStartTime('07:00');
      setEndTime('21:00');
    }
  };

  const startEdit = (o: Offer) => {
    skipNextDirtyRef.current = true;
    const found = DISCOUNT_TYPES.find((t) => t.apiType === o.type);
    const tId = found?.id ?? 'pct';
    setEditingId(o.id);
    setEditingStatus(o.status === 'live' || o.status === 'draft' ? o.status : null);
    setDirty(false);
    setTypeId(tId);
    setTitle(o.title);
    setPercent(tId === 'pct' ? parseValue(tId, o.value) : '20');
    setAmount(tId === 'amount' ? parseValue(tId, o.value) : '500');
    setFreeText(tId === 'freebie' ? o.value : '');
    setBundleText(tId === 'bundle' ? o.value : '');
    setDuration('custom');
    setStartTime(o.start.match(/\d{1,2}:\d{2}/)?.[0] ?? '09:00');
    setEndTime(o.end.match(/\d{1,2}:\d{2}/)?.[0] ?? '17:00');
    setLimit(o.limit || 50);
    setUnlimited(o.limit === 0);
    setError(null);
    setActiveTab('deals');
    window.scrollTo({ top: 0, behavior: 'smooth' });
  };

  const dayLabel = duration === 'weekend' ? 'This weekend' : duration === 'custom' ? 'Custom' : 'Today';
  const computedValue = valueFor(typeId, percent, amount, freeText, bundleText);
  const computedLimit = unlimited ? 0 : limit;

  const save = useMutation({
    mutationFn: ({ status }: { status: 'live' | 'draft'; resetAfter: boolean }) => {
      const body = {
        title,
        type: DISCOUNT_TYPES.find((t) => t.id === typeId)?.apiType ?? 'Percentage',
        value: computedValue,
        status,
        start: `${dayLabel} ${to12h(startTime)}`,
        end: `${dayLabel} ${to12h(endTime)}`,
        limit: computedLimit,
      };
      return editingId ? api.put(`/vendor/offers/${editingId}`, body) : api.post('/vendor/offers', body);
    },
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({ queryKey: ['offers'] });
      if (vars.resetAfter) resetForm();
      else setDirty(false);
    },
    onError: (e) => setError(e instanceof Error ? e.message : 'Failed to save'),
  });

  const canAutoSaveOffer = editingId != null && editingStatus != null;
  useEffect(() => {
    if (skipNextDirtyRef.current) {
      skipNextDirtyRef.current = false;
      return;
    }
    setDirty(true);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [typeId, title, percent, amount, freeText, bundleText, duration, startTime, endTime, limit, unlimited]);

  useAutoSave({
    deps: [typeId, title, percent, amount, freeText, bundleText, duration, startTime, endTime, limit, unlimited],
    dirty: dirty && canAutoSaveOffer,
    onSave: () => {
      if (editingStatus) save.mutate({ status: editingStatus, resetAfter: false });
    },
  });

  const setStatus = useMutation({
    mutationFn: ({ id, status }: { id: string; status: OfferStatus }) =>
      api.patch(`/vendor/offers/${id}/status`, { status }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['offers'] }),
  });

  const remove = useMutation({
    mutationFn: (id: string) => api.del(`/vendor/offers/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['offers'] }),
  });

  const counts = {
    all: offers.length,
    live: offers.filter((o) => o.status === 'live' || o.status === 'ending').length,
    draft: offers.filter((o) => o.status === 'draft').length,
    ended: offers.filter((o) => o.status === 'ended').length,
  };
  const visible = offers
    .filter((o) => filter === 'all' || o.status === filter || (filter === 'live' && o.status === 'ending'))
    .filter((o) => !q || o.title.toLowerCase().includes(q) || o.value.toLowerCase().includes(q));

  const handleTabChange = (tab: 'deals' | 'flash' | 'loyalty') => {
    setActiveTab(tab);
    const url = new URL(window.location.href);
    if (tab === 'deals') url.searchParams.delete('tab');
    else url.searchParams.set('tab', tab);
    window.history.replaceState({}, '', url.toString());
  };

  return (
    <Shell>
      {/* Top Header & Consolidated 3-Pillar Tabs */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 mb-7">
        <div>
          <div className="flex items-center gap-2.5">
            <h1 className="text-2xl font-black font-display tracking-tight text-ink">Promotions Hub</h1>
            <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-[11px] font-extrabold bg-accent/10 text-accent border border-accent/20">
              <Sparkles size={11} /> Foot Traffic &amp; Loyalty
            </span>
          </div>
          <p className="text-muted font-medium text-[13.5px] mt-1 max-w-2xl">
            Acquire new guests, broadcast urgent flash beacons during dead hours, and turn first-time visitors into loyal regulars.
          </p>
        </div>

        {/* 3 Unified Operational Tabs */}
        <div className="flex items-center p-1 rounded-2xl bg-sand/80 border border-line shrink-0 self-start md:self-auto">
          <button
            type="button"
            onClick={() => handleTabChange('deals')}
            className={`flex items-center gap-1.5 px-3.5 py-2 rounded-xl text-[13px] font-extrabold transition-all ${
              activeTab === 'deals' ? 'bg-white text-ink shadow-xs' : 'text-muted hover:text-ink'
            }`}
          >
            <Tag size={14} className={activeTab === 'deals' ? 'text-accent' : ''} />
            <span>Store Deals</span>
            <span className="text-[11px] font-bold px-1.5 py-0.2 rounded-full bg-sand text-muted">
              {offers.length}
            </span>
          </button>
          <button
            type="button"
            onClick={() => handleTabChange('flash')}
            className={`flex items-center gap-1.5 px-3.5 py-2 rounded-xl text-[13px] font-extrabold transition-all ${
              activeTab === 'flash' ? 'bg-accent text-white shadow-xs' : 'text-muted hover:text-ink'
            }`}
          >
            <Zap size={14} />
            <span>Flash Beacon</span>
          </button>
          <button
            type="button"
            onClick={() => handleTabChange('loyalty')}
            className={`flex items-center gap-1.5 px-3.5 py-2 rounded-xl text-[13px] font-extrabold transition-all ${
              activeTab === 'loyalty' ? 'bg-white text-ink shadow-xs' : 'text-muted hover:text-ink'
            }`}
          >
            <Star size={14} className={activeTab === 'loyalty' ? 'text-sun' : ''} />
            <span>Loyalty Stamps</span>
          </button>
        </div>
      </div>

      {/* ───────────────────────────────────────────────────────────── */}
      {/* TAB 1: STORE DEALS (Regular always-on offers & vouchers)       */}
      {/* ───────────────────────────────────────────────────────────── */}
      {activeTab === 'deals' && (
        <div>
          {!isStaff && (
            <div className="grid lg:grid-cols-[1fr_320px] gap-5 items-start mb-8">
              <div className="card p-6">
                {/* Cross-linking hint banner to Flash Beacon */}
                <div className="flex items-center justify-between p-3 mb-5 rounded-xl bg-amber-500/10 border border-amber-500/20">
                  <div className="flex items-center gap-2">
                    <Zap size={15} className="text-amber-600 shrink-0" />
                    <span className="text-[12.5px] font-semibold text-amber-900">
                      Empty seats right now? Broadcast an instant deal to all travellers within 3km.
                    </span>
                  </div>
                  <button
                    type="button"
                    onClick={() => handleTabChange('flash')}
                    className="btn btn-secondary text-[11.5px] font-extrabold px-2.5 py-1 shrink-0 ml-2"
                  >
                    Flash Beacon →
                  </button>
                </div>

                {editingId && (
                  <div className="flex items-center justify-between p-3 mb-4 rounded-xl bg-accent-tint/60 border border-accent/20">
                    <div className="flex items-center gap-2">
                      <Pencil size={15} className="text-accent" />
                      <span className="text-[13px] font-extrabold text-accent">Editing offer: {title || 'Untitled'}</span>
                    </div>
                    <button
                      type="button"
                      onClick={resetForm}
                      className="text-[12px] font-bold px-2.5 py-1 rounded-md text-muted hover:text-ink hover:bg-sand transition-colors flex items-center gap-1"
                    >
                      <X size={13} /> Cancel edit
                    </button>
                  </div>
                )}

                <label className="block text-[13px] font-extrabold mb-2">Discount type</label>
                <div className="grid grid-cols-4 gap-2 mb-5">
                  {DISCOUNT_TYPES.map((t) => {
                    const Icon = t.icon;
                    const on = typeId === t.id;
                    return (
                      <button
                        key={t.id}
                        type="button"
                        onClick={() => setTypeId(t.id)}
                        className={`rounded-xl border py-3 flex flex-col items-center gap-1.5 text-[12.5px] font-extrabold transition-colors ${
                          on ? 'border-accent bg-accent text-white' : 'border-line bg-white hover:border-accent'
                        }`}
                      >
                        <Icon size={18} />
                        <span>{t.label}</span>
                      </button>
                    );
                  })}
                </div>

                {typeId === 'pct' && (
                  <div className="mb-5">
                    <label className="block text-[13px] font-extrabold mb-1.5">Percentage off</label>
                    <div className="flex gap-2">
                      <input
                        className="input flex-1"
                        type="number"
                        min={5}
                        max={90}
                        value={percent}
                        onChange={(e) => setPercent(e.target.value)}
                      />
                      <span className="rounded-xl border border-line bg-sand px-4 py-2.5 font-extrabold text-[13px] flex items-center">
                        % off
                      </span>
                    </div>
                  </div>
                )}

                {typeId === 'amount' && (
                  <div className="mb-5">
                    <label className="block text-[13px] font-extrabold mb-1.5">Amount discount</label>
                    <div className="flex gap-2">
                      <input
                        className="input flex-1"
                        type="number"
                        min={50}
                        step={50}
                        value={amount}
                        onChange={(e) => setAmount(e.target.value)}
                      />
                      <span className="rounded-xl border border-line bg-sand px-4 py-2.5 font-extrabold text-[13px] flex items-center">
                        Rs off
                      </span>
                    </div>
                  </div>
                )}

                {typeId === 'freebie' && (
                  <div className="mb-5">
                    <label className="block text-[13px] font-extrabold mb-1.5">Free item description</label>
                    <input
                      className="input w-full"
                      placeholder="e.g. Free iced tea with lunch"
                      value={freeText}
                      onChange={(e) => setFreeText(e.target.value)}
                    />
                  </div>
                )}

                {typeId === 'bundle' && (
                  <div className="mb-5">
                    <label className="block text-[13px] font-extrabold mb-1.5">Bundle deal description</label>
                    <input
                      className="input w-full"
                      placeholder="e.g. Burger + Beer combo for Rs 1800"
                      value={bundleText}
                      onChange={(e) => setBundleText(e.target.value)}
                    />
                  </div>
                )}

                <div className="mb-5">
                  <label className="block text-[13px] font-extrabold mb-1.5">Offer title</label>
                  <input
                    className="input w-full"
                    placeholder="e.g. 20% off after 8 PM"
                    value={title}
                    onChange={(e) => setTitle(e.target.value)}
                  />
                </div>

                <div className="mb-5">
                  <label className="block text-[13px] font-extrabold mb-1.5">Schedule / Validity</label>
                  <div className="grid grid-cols-4 gap-2">
                    {DURATIONS.map((d) => (
                      <button
                        key={d.id}
                        type="button"
                        onClick={() => applyDuration(d.id)}
                        className={`rounded-xl border py-2.5 font-extrabold text-[12.5px] transition-colors ${
                          duration === d.id ? 'border-accent bg-accent text-white' : 'border-line bg-white hover:border-accent'
                        }`}
                      >
                        {d.label}
                      </button>
                    ))}
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-3 mb-5">
                  <div>
                    <label className="block text-[13px] font-extrabold mb-1.5">Start time</label>
                    <input
                      className="input"
                      type="time"
                      value={startTime}
                      onChange={(e) => {
                        setStartTime(e.target.value);
                        setDuration('custom');
                      }}
                    />
                  </div>
                  <div>
                    <label className="block text-[13px] font-extrabold mb-1.5">End time</label>
                    <input
                      className="input"
                      type="time"
                      value={endTime}
                      onChange={(e) => {
                        setEndTime(e.target.value);
                        setDuration('custom');
                      }}
                    />
                  </div>
                </div>

                <div className="flex items-center justify-between mb-1.5">
                  <label className="text-[13px] font-extrabold">Claim limit</label>
                  <span className="text-[11.5px] text-muted font-semibold">cap how many travelers can claim</span>
                </div>
                <div className="flex gap-2">
                  <input
                    className="input flex-1"
                    type="number"
                    min={1}
                    value={limit}
                    disabled={unlimited}
                    onChange={(e) => setLimit(Number(e.target.value))}
                  />
                  <button
                    type="button"
                    onClick={() => setUnlimited((v) => !v)}
                    className={`rounded-xl border px-4 py-2.5 font-extrabold text-[13px] flex items-center gap-1.5 whitespace-nowrap transition-colors ${
                      unlimited ? 'border-accent text-accent bg-accent-tint' : 'border-line bg-white hover:border-accent'
                    }`}
                  >
                    <InfinityIcon size={15} /> Unlimited
                  </button>
                </div>

                {editingId && (
                  <button
                    type="button"
                    onClick={resetForm}
                    className="mt-4 text-[13px] font-extrabold text-muted hover:text-ink block"
                  >
                    Cancel edit &amp; start a new offer
                  </button>
                )}
              </div>

              {/* Traveler Preview & CTA */}
              <div className="card p-5 lg:sticky lg:top-5">
                <p className="text-[11px] font-extrabold uppercase text-muted mb-3 flex items-center gap-1.5">
                  <Zap size={12} /> Traveler preview
                </p>
                <div
                  className="rounded-2xl overflow-hidden mb-4"
                  style={{ background: 'linear-gradient(135deg, #0e8c8a 0%, #0b4a54 100%)' }}
                >
                  <div className="relative h-20 flex items-start p-3">
                    <span className="rounded-full bg-sun text-ink text-[11px] font-extrabold px-2.5 py-1">
                      {computedValue}
                    </span>
                    <Coffee size={22} className="absolute right-4 top-1/2 -translate-y-1/2 text-white/70" />
                  </div>
                  <div className="bg-white p-3">
                    <p className="font-extrabold text-[14px] truncate">{business?.name ?? 'Your business'}</p>
                    <p className="text-[13px] font-semibold text-muted truncate">{title || 'Offer title'}</p>
                    <p className="text-[11.5px] text-muted font-semibold mt-1">
                      {DURATIONS.find((d) => d.id === duration)?.label} ·{' '}
                      {unlimited ? 'Unlimited claims' : `${limit} claims left`}
                    </p>
                  </div>
                </div>

                {error && <p className="text-[13px] font-bold text-coral mb-3">{error}</p>}

                <button
                  className="btn btn-accent w-full py-3 flex items-center justify-center gap-2 mb-2.5"
                  disabled={!title || save.isPending}
                  onClick={() => save.mutate({ status: 'live', resetAfter: true })}
                >
                  <Rocket size={16} /> {save.isPending ? 'Publishing…' : 'Publish offer'}
                </button>
                <button
                  className="btn btn-secondary w-full py-3 font-extrabold text-[14px] flex items-center justify-center gap-2"
                  disabled={!title || save.isPending}
                  onClick={() => save.mutate({ status: 'draft', resetAfter: true })}
                >
                  <Pencil size={15} /> Save as draft
                </button>
                <p className="text-[11.5px] text-muted font-semibold text-center mt-2.5">
                  Publish goes live instantly · drafts stay private
                </p>
              </div>
            </div>
          )}

          {/* Existing Offers Grid */}
          <div className="flex items-center justify-between mb-1">
            <h2 className="text-lg font-extrabold flex items-center gap-2">
              Your active &amp; saved offers
              <span className="rounded-full bg-accent-tint text-accent text-[12.5px] font-extrabold w-6 h-6 flex items-center justify-center">
                {offers.length}
              </span>
            </h2>
            <div className="flex gap-1.5">
              {FILTERS.map((f) => (
                <button
                  key={f.id}
                  onClick={() => setFilter(f.id)}
                  className={`rounded-full border px-3.5 py-1.5 text-[12.5px] font-extrabold transition-colors ${
                    filter === f.id ? 'border-accent bg-accent text-white' : 'border-line bg-white hover:border-accent'
                  }`}
                >
                  {f.label} {counts[f.id]}
                </button>
              ))}
            </div>
          </div>
          <p className="text-[13px] text-muted font-semibold mb-4">
            Live offers show on every traveler&rsquo;s map and spots feed · drafts stay private · ended offers keep stats.
          </p>

          <div className="grid md:grid-cols-2 xl:grid-cols-3 gap-4">
            {visible.map((o) => (
              <OfferCard
                key={o.id}
                offer={o}
                onEdit={() => startEdit(o)}
                readOnly={isStaff}
                onSetStatus={(status) => setStatus.mutate({ id: o.id, status })}
                onDelete={() => remove.mutate(o.id)}
                onOpenDetail={() => setDetailOfferId(o.id)}
              />
            ))}
          </div>
        </div>
      )}

      {/* ───────────────────────────────────────────────────────────── */}
      {/* TAB 2: FLASH BEACON (Real-time 3km dead-hour push blast)      */}
      {/* ───────────────────────────────────────────────────────────── */}
      {activeTab === 'flash' && (
        <div className="space-y-6">
          <div className="card p-6">
            <div className="flex items-center gap-3 mb-5">
              <span className="w-10 h-10 rounded-2xl bg-[#fdf0cd] text-[#a97b00] flex items-center justify-center shrink-0">
                <Zap size={20} />
              </span>
              <div>
                <h2 className="font-extrabold text-[18px]">Flash Sale Broadcast</h2>
                <p className="text-[13px] text-muted font-semibold">
                  Slow afternoon or unexpected empty tables? Blast an urgent deal to everyone nearby instantly.
                </p>
              </div>
            </div>

            <div className="grid md:grid-cols-[1fr_auto] gap-8 items-center bg-sand/30 p-6 rounded-2xl border border-line">
              <div>
                <label className="block text-[13px] font-extrabold mb-2 text-ink">Choose Discount Percentage</label>
                <div className="flex gap-2 mb-6">
                  {FLASH_DISCOUNTS.map((pct) => (
                    <button
                      key={pct}
                      type="button"
                      onClick={() => setFlashDiscount(pct)}
                      className={`rounded-xl border px-5 py-2.5 text-[14px] font-extrabold transition-all ${
                        flashDiscount === pct
                          ? 'border-accent bg-accent text-white shadow-xs'
                          : 'border-line bg-white hover:border-accent text-ink'
                      }`}
                    >
                      {pct}% off
                    </button>
                  ))}
                </div>

                <div className="flex items-center gap-4">
                  <span className="w-16 h-16 rounded-2xl border-2 border-line bg-white flex flex-col items-center justify-center text-[13px] font-black text-ink shrink-0 shadow-xs">
                    {growth?.flash?.inRange ?? '—'}
                    <span className="text-[8.5px] text-muted font-extrabold tracking-wider">IN RANGE</span>
                  </span>
                  <div>
                    <p className="font-extrabold text-[15px] text-ink">
                      {growth?.flash?.inRange ?? '—'} active travelers within {growth?.flash?.radiusKm ?? 3} km
                    </p>
                    <p className="text-[13px] text-muted font-medium mt-0.5">
                      {growth?.flash?.nomads ?? 0} digital nomads · {growth?.flash?.surfers ?? 0} surfers nearby
                    </p>
                    <p className="text-[12px] text-muted font-medium mt-1">
                      Cooldown: 1 blast every 3 hours · Last sent {Math.round((growth?.flash?.lastSentMins ?? 0) / 60)}h ago
                    </p>
                  </div>
                </div>
              </div>

              <div className="flex flex-col items-center justify-center shrink-0">
                <button
                  type="button"
                  onClick={() => !flashDisabled && sendFlashSale.mutate()}
                  disabled={flashDisabled}
                  className="w-36 h-36 rounded-full flex flex-col items-center justify-center text-white font-black gap-1 disabled:opacity-50 transition-all active:scale-95 shadow-md"
                  style={{ background: 'linear-gradient(135deg, #f0682f, #c94f1c)' }}
                >
                  <Radio size={24} className="animate-pulse" />
                  <span className="text-[14px] tracking-tight">
                    {sendFlashSale.isPending ? 'Broadcasting…' : 'Broadcast Now'}
                  </span>
                  <span className="text-[11px] font-bold opacity-90">
                    {flashDiscount}% OFF · {growth?.flash?.radiusKm ?? 3}km
                  </span>
                </button>
                <p className="text-[11.5px] text-muted font-semibold mt-2.5">Instant push notification</p>
              </div>
            </div>

            {sendFlashSale.isError && (
              <p className="text-[13px] font-bold mt-4 text-center text-coral">
                {sendFlashSale.error instanceof Error ? sendFlashSale.error.message : 'Could not broadcast — please try again.'}
              </p>
            )}
            {sendFlashSale.isSuccess && (
              <p className="text-[13px] font-bold mt-4 text-center text-emerald-600 flex items-center justify-center gap-1.5">
                <Check size={16} /> Flash sale broadcasted to travelers in your area!
              </p>
            )}
          </div>

          <div className="grid md:grid-cols-3 gap-4">
            <div className="card p-5">
              <span className="w-8 h-8 rounded-lg bg-accent/15 text-accent flex items-center justify-center mb-3">
                <Radio size={16} />
              </span>
              <h3 className="font-extrabold text-[14px]">3 km Geofence Push</h3>
              <p className="text-[12.5px] text-muted font-medium mt-1 leading-relaxed">
                Reaches tourists already walking around your neighborhood who are deciding where to eat or drink right now.
              </p>
            </div>
            <div className="card p-5">
              <span className="w-8 h-8 rounded-lg bg-accent/15 text-accent flex items-center justify-center mb-3">
                <Flame size={16} />
              </span>
              <h3 className="font-extrabold text-[14px]">2-Hour High Urgency</h3>
              <p className="text-[12.5px] text-muted font-medium mt-1 leading-relaxed">
                Appears with a countdown timer at the very top of traveler maps and feeds to drive immediate foot traffic.
              </p>
            </div>
            <div className="card p-5">
              <span className="w-8 h-8 rounded-lg bg-accent/15 text-accent flex items-center justify-center mb-3">
                <Calendar size={16} />
              </span>
              <h3 className="font-extrabold text-[14px]">Anti-Spam 3h Cooldown</h3>
              <p className="text-[12.5px] text-muted font-medium mt-1 leading-relaxed">
                A built-in 3-hour cooldown maintains authenticity so travelers take flash beacons seriously when they pop up.
              </p>
            </div>
          </div>
        </div>
      )}

      {/* ───────────────────────────────────────────────────────────── */}
      {/* TAB 3: LOYALTY & STAMPS (Digital customer retention)          */}
      {/* ───────────────────────────────────────────────────────────── */}
      {activeTab === 'loyalty' && (
        <div className="space-y-6">
          <div className="card p-6">
            <div className="flex items-center gap-3 mb-5">
              <span className="w-10 h-10 rounded-2xl bg-accent-tint text-accent flex items-center justify-center shrink-0">
                <Star size={20} />
              </span>
              <div>
                <h2 className="font-extrabold text-[18px]">Digital Stamp Cards</h2>
                <p className="text-[13px] text-muted font-semibold">
                  Reward repeat visits automatically — {growth?.stampCard?.stampsNeeded ?? 6} stamps earns a {growth?.stampCard?.reward ?? 'reward'}.
                </p>
              </div>
            </div>

            <div className="rounded-2xl bg-accent-tint/60 border border-accent/20 p-5 flex flex-col sm:flex-row sm:items-center justify-between gap-4 mb-6">
              <div>
                <p className="font-extrabold text-[26px] text-ink leading-tight">
                  {growth?.stampCard?.buckets?.[0]?.count ?? 0}
                </p>
                <p className="text-[13.5px] font-extrabold text-ink mt-0.5">
                  travelers in town are just ONE stamp away from their reward
                </p>
                <p className="text-[12px] text-muted font-medium mt-0.5">
                  Send a gentle nudge notification reminding them to complete their card before leaving town.
                </p>
              </div>
              <button
                type="button"
                onClick={() => {
                  setNudged(true);
                  setTimeout(() => setNudged(false), 2500);
                }}
                disabled={nudged}
                className="btn btn-accent px-4 py-2.5 text-[13px] font-extrabold flex items-center gap-2 shrink-0 self-start sm:self-auto shadow-xs"
              >
                <UserPlus size={15} />
                <span>{nudged ? '✓ Nudge Broadcasted!' : 'Nudge 1-Stamp Travelers'}</span>
              </button>
            </div>

            <h3 className="font-extrabold text-[14px] text-ink mb-3">Active Guest Progress</h3>
            <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-6 gap-3">
              {(growth?.stampCard?.buckets ?? []).map((b) => (
                <div key={b.away} className="rounded-xl border border-line bg-white p-3.5 text-center">
                  <div className="flex justify-center gap-1 mb-2">
                    {Array.from({ length: 6 }).map((_, i) => (
                      <span
                        key={i}
                        className="w-1.5 h-1.5 rounded-full"
                        style={{ background: i < 6 - b.away ? 'var(--accent)' : 'var(--line)' }}
                      />
                    ))}
                  </div>
                  <p className="font-extrabold text-[18px] text-ink">{b.count}</p>
                  <p className="text-[11px] text-muted font-semibold mt-0.5">{b.label}</p>
                </div>
              ))}
            </div>
          </div>

          <div className="grid md:grid-cols-2 gap-4">
            <div className="card p-5">
              <h3 className="font-extrabold text-[14.5px] text-ink">How Digital Stamps Build Loyalty</h3>
              <ul className="space-y-2.5 mt-3 text-[13px] text-muted font-medium">
                <li className="flex items-start gap-2">
                  <Check size={15} className="text-accent shrink-0 mt-0.5" />
                  <span>Guests scan your counter QR on every purchase to collect a stamp.</span>
                </li>
                <li className="flex items-start gap-2">
                  <Check size={15} className="text-accent shrink-0 mt-0.5" />
                  <span>No paper cards to lose — stamps stay securely saved in their Spotto digital wallet.</span>
                </li>
                <li className="flex items-start gap-2">
                  <Check size={15} className="text-accent shrink-0 mt-0.5" />
                  <span>Spotto notifies travelers when they are close to completing a card while still in town.</span>
                </li>
              </ul>
            </div>

            <div className="card p-5 flex flex-col justify-between">
              <div>
                <h3 className="font-extrabold text-[14.5px] text-ink">Sponsor the Town's Grand Prize?</h3>
                <p className="text-[13px] text-muted font-medium mt-1 leading-relaxed">
                  Looking for town-wide spotlight beyond your regulars? Sponsor the Daily Quest to be featured as today's official destination for all travelers.
                </p>
              </div>
              <button
                type="button"
                onClick={() => router.push('/sponsorship')}
                className="btn btn-secondary text-[12.5px] font-extrabold py-2 px-3 mt-4 flex items-center justify-between"
              >
                <span>View Daily Quest Sponsorship</span>
                <ArrowRight size={14} />
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Offer Detail Drawer */}
      <OfferDetailDrawer
        offer={detailOffer}
        onClose={() => setDetailOfferId(null)}
        onSetStatus={
          isStaff
            ? undefined
            : (status) => detailOfferId && setStatus.mutate({ id: detailOfferId, status })
        }
        onDelete={
          isStaff
            ? undefined
            : () => {
                if (detailOfferId) remove.mutate(detailOfferId);
                setDetailOfferId(null);
              }
        }
      />
    </Shell>
  );
}

function OfferCard({
  offer,
  onEdit,
  onSetStatus,
  onDelete,
  onOpenDetail,
  readOnly,
}: {
  offer: Offer;
  onEdit: () => void;
  onSetStatus: (status: OfferStatus) => void;
  onDelete: () => void;
  onOpenDetail: () => void;
  readOnly?: boolean;
}) {
  const Icon = TYPE_ICON[offer.type] ?? Tag;
  const pct = offer.limit > 0 ? Math.min(100, Math.round((offer.claimed / offer.limit) * 100)) : 0;
  const barColor = offer.status === 'ending' ? '#eaa200' : offer.status === 'live' ? undefined : '#cbd5d3';

  return (
    <div className="card p-4 cursor-pointer" onClick={onOpenDetail}>
      <div className="flex items-start gap-3 mb-2.5">
        <span className="w-9 h-9 rounded-xl bg-accent-tint text-accent flex items-center justify-center shrink-0">
          <Icon size={17} />
        </span>
        <div className="flex-1 min-w-0">
          <p className="font-extrabold text-[14.5px] truncate">{offer.title}</p>
          <div className="flex items-center gap-2 mt-1 flex-wrap">
            <span className="badge bg-sand text-ink border border-line/60">
              <Tag size={10} /> {offer.value}
            </span>
            <StatusPill status={offer.status} />
          </div>
        </div>
      </div>

      {offer.status === 'draft' ? (
        <div className="rounded-xl bg-sand px-3 py-2.5 mb-3 flex items-center gap-2 text-[12.5px] font-semibold text-muted">
          <EyeOff size={14} /> Not visible to travelers yet — publish when ready.
        </div>
      ) : (
        <>
          <div className="flex items-center justify-between text-[12.5px] font-bold mb-1">
            <span>{offer.limit > 0 ? `${offer.claimed}/${offer.limit} claimed` : `${offer.claimed} claimed · unlimited`}</span>
            <span className="text-accent">{offer.redeemed} redeemed</span>
          </div>
          {offer.limit > 0 && (
            <div className="h-2 rounded-full bg-line overflow-hidden mb-2.5">
              <div className="h-full rounded-full" style={{ width: `${pct}%`, background: barColor ?? 'var(--accent)' }} />
            </div>
          )}
        </>
      )}

      <p className="text-[12px] text-muted font-semibold flex items-center gap-1.5 mb-3">
        <Calendar size={12} /> {offer.status === 'draft' ? 'Schedule set on publish' : `${offer.start} → ${offer.end}`}
      </p>

      {!readOnly && (
        <div className="grid grid-cols-3 gap-1.5" onClick={(e) => e.stopPropagation()}>
          <button className="btn btn-secondary py-1.5 text-[12px] font-extrabold" onClick={onEdit}>
            <Pencil size={13} /> Edit
          </button>
          {offer.status === 'draft' && (
            <button className="btn btn-secondary py-1.5 text-[12px] font-extrabold" onClick={() => onSetStatus('live')}>
              <Zap size={13} /> Publish
            </button>
          )}
          {(offer.status === 'live' || offer.status === 'ending') && (
            <button
              className="btn btn-secondary py-1.5 text-[12px] font-extrabold text-[#d9534f] hover:text-[#c9302c]"
              onClick={() => onSetStatus('ended')}
            >
              <PowerOff size={13} /> Close
            </button>
          )}
          {offer.status === 'ended' && (
            <button
              className="btn btn-secondary py-1.5 text-[12px] font-extrabold text-accent"
              onClick={() => onSetStatus('live')}
            >
              <RotateCcw size={13} /> Reopen
            </button>
          )}
          <button className="btn btn-danger-outline py-1.5 text-[12px] font-extrabold" onClick={onDelete}>
            <Trash2 size={13} /> Delete
          </button>
        </div>
      )}
    </div>
  );
}
