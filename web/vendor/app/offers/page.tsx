'use client';

// Offers — always-visible creator (with a live traveler preview) plus a
// filterable card grid of existing offers. Port of the Offers mockup.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  BadgeDollarSign, Calendar, Check, Coffee, EyeOff, Gift, Infinity as InfinityIcon,
  Package, Pencil, Percent, Rocket, Tag, Trash2, Zap, LucideIcon,
} from 'lucide-react';
import { useState } from 'react';

import type { Offer, OfferStatus } from '@spotto/shared';
import Shell, { useBusiness } from '../../components/shell';
import { api } from '../../lib/api';

const DISCOUNT_TYPES: { id: string; apiType: string; label: string; icon: LucideIcon }[] = [
  { id: 'pct', apiType: 'Percentage', label: 'Percentage off', icon: Percent },
  { id: 'amount', apiType: 'Amount', label: 'Amount off', icon: BadgeDollarSign },
  { id: 'freebie', apiType: 'Freebie', label: 'Freebie', icon: Gift },
  { id: 'bundle', apiType: 'Bundle', label: 'Bundle', icon: Package },
];

const TYPE_ICON: Record<string, LucideIcon> = {
  Percentage: Percent,
  Amount: BadgeDollarSign,
  Freebie: Gift,
  Bundle: Package,
};

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

// Best-effort reverse-parse for editing an existing offer's value string
// back into the type-specific field — offers created before this redesign
// (or via cmd/seed's fixed demo data) won't always parse cleanly, so this
// always falls back to something reasonable rather than throwing.
function parseValue(typeId: string, value: string) {
  const num = value.match(/\d+/)?.[0] ?? '';
  if (typeId === 'pct' || typeId === 'amount') return num;
  return value;
}

export default function OffersPage() {
  const qc = useQueryClient();
  const { data: business } = useBusiness();
  const { data: offers = [] } = useQuery<Offer[]>({
    queryKey: ['offers'],
    queryFn: () => api.get<Offer[]>('/vendor/offers'),
  });

  const [filter, setFilter] = useState<FilterId>('all');
  const [editingId, setEditingId] = useState<string | null>(null);

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

  const resetForm = () => {
    setEditingId(null);
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
    const found = DISCOUNT_TYPES.find((t) => t.apiType === o.type);
    const tId = found?.id ?? 'pct';
    setEditingId(o.id);
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
    window.scrollTo({ top: 0, behavior: 'smooth' });
  };

  const dayLabel = duration === 'weekend' ? 'This weekend' : duration === 'custom' ? 'Custom' : 'Today';
  const computedValue = valueFor(typeId, percent, amount, freeText, bundleText);
  const computedLimit = unlimited ? 0 : limit;

  const save = useMutation({
    mutationFn: (status: 'live' | 'draft') => {
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
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['offers'] });
      resetForm();
    },
    onError: (e) => setError(e instanceof Error ? e.message : 'Failed to save'),
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
  const visible = offers.filter((o) => filter === 'all' || o.status === filter || (filter === 'live' && o.status === 'ending'));

  return (
    <Shell>
      <div className="grid lg:grid-cols-[1fr_320px] gap-5 items-start mb-8">
        <div className="card p-6">
          <label className="block text-[13px] font-extrabold mb-2">Discount type</label>
          <div className="grid grid-cols-4 gap-2 mb-5">
            {DISCOUNT_TYPES.map((t) => {
              const Icon = t.icon;
              const on = typeId === t.id;
              return (
                <button key={t.id} type="button" onClick={() => setTypeId(t.id)}
                  className={`rounded-xl border py-3 flex flex-col items-center gap-1.5 text-[12.5px] font-extrabold transition-colors ${
                    on ? 'border-accent bg-accent text-white' : 'border-line bg-white hover:border-accent'
                  }`}>
                  <Icon size={18} />
                  {t.label}
                </button>
              );
            })}
          </div>

          <div className="grid grid-cols-[1fr_auto] gap-3 mb-5 items-end">
            <div>
              <div className="flex items-center justify-between mb-1.5">
                <label className="text-[13px] font-extrabold">Offer title</label>
                <span className="text-[11.5px] text-muted font-semibold">what travelers see</span>
              </div>
              <input className="input" placeholder="e.g. 20% off coffee" value={title} onChange={(e) => setTitle(e.target.value)} />
            </div>
            <div className="w-28">
              {typeId === 'pct' && (
                <>
                  <label className="block text-[13px] font-extrabold mb-1.5">Percent</label>
                  <div className="relative">
                    <input className="input pr-7" type="number" min={0} max={100} value={percent} onChange={(e) => setPercent(e.target.value)} />
                    <span className="absolute right-3 top-1/2 -translate-y-1/2 text-muted font-bold text-[13px]">%</span>
                  </div>
                </>
              )}
              {typeId === 'amount' && (
                <>
                  <label className="block text-[13px] font-extrabold mb-1.5">Amount (Rs)</label>
                  <input className="input" type="number" min={0} value={amount} onChange={(e) => setAmount(e.target.value)} />
                </>
              )}
              {typeId === 'freebie' && (
                <>
                  <label className="block text-[13px] font-extrabold mb-1.5">What&rsquo;s free</label>
                  <input className="input" value={freeText} onChange={(e) => setFreeText(e.target.value)} />
                </>
              )}
              {typeId === 'bundle' && (
                <>
                  <label className="block text-[13px] font-extrabold mb-1.5">Bundle terms</label>
                  <input className="input" value={bundleText} onChange={(e) => setBundleText(e.target.value)} />
                </>
              )}
            </div>
          </div>

          <label className="block text-[13px] font-extrabold mb-2">Duration</label>
          <div className="flex flex-wrap gap-2 mb-5">
            {DURATIONS.map((d) => (
              <button key={d.id} type="button" onClick={() => applyDuration(d.id)}
                className={`rounded-full border px-4 py-2 text-[13px] font-extrabold transition-colors ${
                  duration === d.id ? 'border-accent text-accent bg-accent-tint' : 'border-line bg-white hover:border-accent'
                }`}>
                {d.label}
              </button>
            ))}
          </div>

          <div className="grid grid-cols-2 gap-3 mb-5">
            <div>
              <label className="block text-[13px] font-extrabold mb-1.5">Start time</label>
              <input className="input" type="time" value={startTime} onChange={(e) => { setStartTime(e.target.value); setDuration('custom'); }} />
            </div>
            <div>
              <label className="block text-[13px] font-extrabold mb-1.5">End time</label>
              <input className="input" type="time" value={endTime} onChange={(e) => { setEndTime(e.target.value); setDuration('custom'); }} />
            </div>
          </div>

          <div className="flex items-center justify-between mb-1.5">
            <label className="text-[13px] font-extrabold">Claim limit</label>
            <span className="text-[11.5px] text-muted font-semibold">cap how many travelers can claim</span>
          </div>
          <div className="flex gap-2">
            <input className="input flex-1" type="number" min={1} value={limit} disabled={unlimited}
              onChange={(e) => setLimit(Number(e.target.value))} />
            <button type="button" onClick={() => setUnlimited((v) => !v)}
              className={`rounded-xl border px-4 py-2.5 font-extrabold text-[13px] flex items-center gap-1.5 whitespace-nowrap transition-colors ${
                unlimited ? 'border-accent text-accent bg-accent-tint' : 'border-line bg-white hover:border-accent'
              }`}>
              <InfinityIcon size={15} /> Unlimited
            </button>
          </div>

          {editingId && (
            <button type="button" onClick={resetForm} className="mt-4 text-[13px] font-extrabold text-muted hover:text-ink">
              Cancel edit &amp; start a new offer
            </button>
          )}
        </div>

        <div className="card p-5 lg:sticky lg:top-5">
          <p className="text-[11px] font-extrabold uppercase text-muted mb-3 flex items-center gap-1.5">
            <Zap size={12} /> Traveler preview
          </p>
          <div className="rounded-2xl overflow-hidden mb-4" style={{ background: 'linear-gradient(135deg, #0e8c8a 0%, #0b4a54 100%)' }}>
            <div className="relative h-20 flex items-start p-3">
              <span className="rounded-full bg-sun text-ink text-[11px] font-extrabold px-2.5 py-1">{computedValue}</span>
              <Coffee size={22} className="absolute right-4 top-1/2 -translate-y-1/2 text-white/70" />
            </div>
            <div className="bg-white p-3">
              <p className="font-extrabold text-[14px] truncate">{business?.name ?? 'Your business'}</p>
              <p className="text-[13px] font-semibold text-muted truncate">{title || 'Offer title'}</p>
              <p className="text-[11.5px] text-muted font-semibold mt-1">
                {DURATIONS.find((d) => d.id === duration)?.label} · {unlimited ? 'Unlimited claims' : `${limit} claims left`}
              </p>
            </div>
          </div>

          {error && <p className="text-[13px] font-bold text-coral mb-3">{error}</p>}

          <button className="btn-accent w-full py-3 flex items-center justify-center gap-2 mb-2.5"
            disabled={!title || save.isPending} onClick={() => save.mutate('live')}>
            <Rocket size={16} /> {save.isPending ? 'Publishing…' : 'Publish offer'}
          </button>
          <button className="rounded-xl border border-line bg-white w-full py-3 font-extrabold text-[14px] flex items-center justify-center gap-2"
            disabled={!title || save.isPending} onClick={() => save.mutate('draft')}>
            <Pencil size={15} /> Save as draft
          </button>
          <p className="text-[11.5px] text-muted font-semibold text-center mt-2.5">
            Publish goes live instantly · drafts stay private
          </p>
        </div>
      </div>

      <div className="flex items-center justify-between mb-1">
        <h2 className="text-lg font-extrabold flex items-center gap-2">
          Your offers
          <span className="rounded-full bg-accent-tint text-accent text-[12.5px] font-extrabold w-6 h-6 flex items-center justify-center">
            {offers.length}
          </span>
        </h2>
        <div className="flex gap-1.5">
          {FILTERS.map((f) => (
            <button key={f.id} onClick={() => setFilter(f.id)}
              className={`rounded-full border px-3.5 py-1.5 text-[12.5px] font-extrabold transition-colors ${
                filter === f.id ? 'border-accent bg-accent text-white' : 'border-line bg-white hover:border-accent'
              }`}>
              {f.label} {counts[f.id]}
            </button>
          ))}
        </div>
      </div>
      <p className="text-[13px] text-muted font-semibold mb-4">
        Live offers show on every traveler&rsquo;s map · drafts stay private · ended offers keep their stats.
      </p>

      <div className="grid md:grid-cols-2 xl:grid-cols-3 gap-4">
        {visible.map((o) => (
          <OfferCard key={o.id} offer={o} onEdit={() => startEdit(o)}
            onSetStatus={(status) => setStatus.mutate({ id: o.id, status })}
            onDelete={() => remove.mutate(o.id)} />
        ))}
      </div>
    </Shell>
  );
}

function OfferCard({
  offer, onEdit, onSetStatus, onDelete,
}: {
  offer: Offer;
  onEdit: () => void;
  onSetStatus: (status: OfferStatus) => void;
  onDelete: () => void;
}) {
  const Icon = TYPE_ICON[offer.type] ?? Tag;
  const pct = offer.limit > 0 ? Math.min(100, Math.round((offer.claimed / offer.limit) * 100)) : 0;
  const barColor = offer.status === 'ending' ? '#eaa200' : offer.status === 'live' ? undefined : '#cbd5d3';

  return (
    <div className="card p-4">
      <div className="flex items-start gap-3 mb-2.5">
        <span className="w-9 h-9 rounded-xl bg-accent-tint text-accent flex items-center justify-center shrink-0">
          <Icon size={17} />
        </span>
        <div className="flex-1 min-w-0">
          <p className="font-extrabold text-[14.5px] truncate">{offer.title}</p>
          <div className="flex items-center gap-2 mt-1 flex-wrap">
            <span className="text-[11px] font-extrabold rounded-full px-2 py-0.5 bg-sand flex items-center gap-1">
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

      <div className="grid grid-cols-3 gap-1.5">
        <button className="rounded-lg border border-line py-1.5 text-[12px] font-extrabold flex items-center justify-center gap-1 hover:border-accent hover:text-accent transition-colors" onClick={onEdit}>
          <Pencil size={13} /> Edit
        </button>
        {offer.status === 'draft' && (
          <button className="rounded-lg border border-line py-1.5 text-[12px] font-extrabold flex items-center justify-center gap-1 hover:border-accent hover:text-accent transition-colors" onClick={() => onSetStatus('live')}>
            <Zap size={13} /> Publish
          </button>
        )}
        {(offer.status === 'live' || offer.status === 'ending') && (
          <button className="rounded-lg border border-line py-1.5 text-[12px] font-extrabold flex items-center justify-center gap-1 hover:border-accent hover:text-accent transition-colors" onClick={() => onSetStatus('ended')}>
            <Check size={13} /> End
          </button>
        )}
        {offer.status === 'ended' && (
          <button className="rounded-lg border border-line py-1.5 text-[12px] font-extrabold flex items-center justify-center gap-1 hover:border-accent hover:text-accent transition-colors" onClick={() => onSetStatus('live')}>
            <Zap size={13} /> Relaunch
          </button>
        )}
        <button className="rounded-lg border border-line py-1.5 text-[12px] font-extrabold text-coral flex items-center justify-center gap-1 hover:bg-[#fde4d8] transition-colors" onClick={onDelete}>
          <Trash2 size={13} /> Delete
        </button>
      </div>
    </div>
  );
}

function StatusPill({ status }: { status: OfferStatus }) {
  if (status === 'live') {
    return (
      <span className="text-[11px] font-extrabold uppercase flex items-center gap-1 text-[#16803c]">
        <span className="w-1.5 h-1.5 rounded-full bg-[#16803c]" /> Live
      </span>
    );
  }
  if (status === 'ending') {
    return <span className="text-[11px] font-extrabold uppercase rounded-full px-2 py-0.5 bg-[#fdf0cd] text-[#a97b00]">Ending soon</span>;
  }
  if (status === 'draft') {
    return (
      <span className="text-[11px] font-extrabold uppercase rounded-full px-2 py-0.5 bg-sand text-muted flex items-center gap-1">
        <Pencil size={9} /> Draft
      </span>
    );
  }
  return <span className="text-[11px] font-extrabold uppercase text-muted">Ended</span>;
}
