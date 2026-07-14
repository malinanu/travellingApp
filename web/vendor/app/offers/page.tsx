'use client';

// Offers — list with claim-limit donuts, status actions, and a creator
// drawer. Port of OfferCreator.jsx.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { BadgeDollarSign, Gift, Package, Percent, Plus, Trash2, X, LucideIcon } from 'lucide-react';
import { useState } from 'react';

import { Donut } from '@spotto/charts';
import type { Offer, OfferStatus } from '@spotto/shared';
import Shell from '../../components/shell';
import { api } from '../../lib/api';

const ACCENT = '#0b6f86';

const DISCOUNT_TYPES: { id: string; apiType: string; label: string; icon: LucideIcon; hint: string }[] = [
  { id: 'pct', apiType: 'Percentage', label: 'Percentage off', icon: Percent, hint: 'e.g. 20% off' },
  { id: 'amount', apiType: 'Amount', label: 'Amount off', icon: BadgeDollarSign, hint: 'e.g. Rs 500 off' },
  { id: 'freebie', apiType: 'Freebie', label: 'Freebie', icon: Gift, hint: 'e.g. free coffee' },
  { id: 'bundle', apiType: 'Bundle', label: 'Bundle', icon: Package, hint: 'e.g. 2-for-1' },
];

const STATUS_STYLE: Record<OfferStatus, string> = {
  live: 'bg-accent-tint text-accent',
  ending: 'bg-[#fdf0cd] text-[#a97b00]',
  draft: 'bg-sand text-muted',
  ended: 'bg-sand text-muted/70 line-through',
};

export default function OffersPage() {
  const qc = useQueryClient();
  const [drawer, setDrawer] = useState(false);
  const { data: offers = [] } = useQuery<Offer[]>({
    queryKey: ['offers'],
    queryFn: () => api.get<Offer[]>('/vendor/offers'),
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

  return (
    <Shell>
      <div className="flex items-center justify-between mb-5">
        <p className="text-muted font-semibold text-[14px]">
          {offers.filter((o) => o.status === 'live' || o.status === 'ending').length} live ·{' '}
          {offers.length} total
        </p>
        <button className="btn-accent px-4 py-2.5 flex items-center gap-2 text-[14px]" onClick={() => setDrawer(true)}>
          <Plus size={17} /> New offer
        </button>
      </div>

      <div className="space-y-3">
        {offers.map((o) => (
          <div key={o.id} className="card p-4 flex items-center gap-4 group">
            <Donut value={o.claimed} max={o.limit || 1} accent={ACCENT} />
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <p className="font-extrabold text-[15px] truncate">{o.title}</p>
                <span className={`text-[11px] font-extrabold rounded-full px-2 py-0.5 uppercase ${STATUS_STYLE[o.status]}`}>
                  {o.status}
                </span>
              </div>
              <p className="text-[12.5px] text-muted font-semibold">
                {o.type} · {o.value} · {o.start} → {o.end}
              </p>
              <p className="text-[12.5px] font-bold mt-0.5">
                {o.claimed}/{o.limit} claimed · <span className="text-accent">{o.redeemed} redeemed</span>
              </p>
            </div>
            <div className="flex gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
              {o.status === 'draft' && (
                <ActionBtn onClick={() => setStatus.mutate({ id: o.id, status: 'live' })}>Go live</ActionBtn>
              )}
              {(o.status === 'live' || o.status === 'ending') && (
                <ActionBtn onClick={() => setStatus.mutate({ id: o.id, status: 'ended' })}>End</ActionBtn>
              )}
              {o.status === 'ended' && (
                <ActionBtn onClick={() => setStatus.mutate({ id: o.id, status: 'live' })}>Relaunch</ActionBtn>
              )}
              <button
                aria-label="Delete offer"
                className="rounded-lg border border-line p-2 text-coral hover:bg-[#fde4d8] transition-colors"
                onClick={() => remove.mutate(o.id)}
              >
                <Trash2 size={15} />
              </button>
            </div>
          </div>
        ))}
      </div>

      {drawer && <CreatorDrawer onClose={() => setDrawer(false)} />}
    </Shell>
  );
}

function ActionBtn({ children, onClick }: { children: React.ReactNode; onClick: () => void }) {
  return (
    <button onClick={onClick}
      className="rounded-lg border border-line px-3 py-1.5 text-[12.5px] font-extrabold hover:border-accent hover:text-accent transition-colors">
      {children}
    </button>
  );
}

function CreatorDrawer({ onClose }: { onClose: () => void }) {
  const qc = useQueryClient();
  const [title, setTitle] = useState('');
  const [typeId, setTypeId] = useState('pct');
  const [value, setValue] = useState('');
  const [limit, setLimit] = useState(50);
  const [start, setStart] = useState('Today');
  const [end, setEnd] = useState('This week');
  const [goLive, setGoLive] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const create = useMutation({
    mutationFn: () =>
      api.post('/vendor/offers', {
        title,
        type: DISCOUNT_TYPES.find((t) => t.id === typeId)?.apiType ?? 'Percentage',
        value,
        status: goLive ? 'live' : 'draft',
        start,
        end,
        limit,
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['offers'] });
      onClose();
    },
    onError: (e) => setError(e instanceof Error ? e.message : 'Failed to create'),
  });

  return (
    <div className="fixed inset-0 z-50 flex justify-end bg-ink/30" onClick={onClose}>
      <div className="w-[440px] bg-cream h-full overflow-y-auto p-6" onClick={(e) => e.stopPropagation()}>
        <div className="flex items-center justify-between mb-5">
          <h2 className="text-lg font-extrabold">New offer</h2>
          <button aria-label="Close" onClick={onClose} className="rounded-lg p-2 hover:bg-sand"><X size={18} /></button>
        </div>

        <label className="block text-[13px] font-extrabold mb-1.5">Offer title</label>
        <input className="input mb-4" placeholder="Free flat white with any day pass"
          value={title} onChange={(e) => setTitle(e.target.value)} />

        <label className="block text-[13px] font-extrabold mb-1.5">Type</label>
        <div className="grid grid-cols-2 gap-2 mb-4">
          {DISCOUNT_TYPES.map((t) => {
            const Icon = t.icon;
            const on = typeId === t.id;
            return (
              <button key={t.id} type="button" onClick={() => setTypeId(t.id)}
                className={`rounded-xl border p-3 text-left transition-colors ${
                  on ? 'border-accent bg-accent-tint' : 'border-line bg-white hover:border-accent'
                }`}>
                <Icon size={17} className={on ? 'text-accent' : 'text-muted'} />
                <p className="font-extrabold text-[13px] mt-1">{t.label}</p>
                <p className="text-[11.5px] text-muted font-semibold">{t.hint}</p>
              </button>
            );
          })}
        </div>

        <label className="block text-[13px] font-extrabold mb-1.5">Value shown to travellers</label>
        <input className="input mb-4" placeholder="20% off" value={value} onChange={(e) => setValue(e.target.value)} />

        <div className="grid grid-cols-2 gap-3 mb-4">
          <div>
            <label className="block text-[13px] font-extrabold mb-1.5">Starts</label>
            <input className="input" value={start} onChange={(e) => setStart(e.target.value)} />
          </div>
          <div>
            <label className="block text-[13px] font-extrabold mb-1.5">Ends</label>
            <input className="input" value={end} onChange={(e) => setEnd(e.target.value)} />
          </div>
        </div>

        <label className="block text-[13px] font-extrabold mb-1.5">Claim limit: {limit}</label>
        <input type="range" min={10} max={500} step={10} value={limit}
          onChange={(e) => setLimit(Number(e.target.value))} className="w-full accent-[#0b6f86] mb-4" />

        <label className="flex items-center gap-2.5 mb-6 font-extrabold text-[14px]">
          <input type="checkbox" checked={goLive} onChange={(e) => setGoLive(e.target.checked)}
            className="w-4 h-4 accent-[#0b6f86]" />
          Go live immediately
        </label>

        {error && <p className="text-coral font-bold text-[13px] mb-3">{error}</p>}

        <button className="btn-accent w-full py-3" disabled={!title || create.isPending}
          onClick={() => create.mutate()}>
          {create.isPending ? 'Creating…' : goLive ? 'Publish offer' : 'Save draft'}
        </button>
      </div>
    </div>
  );
}
