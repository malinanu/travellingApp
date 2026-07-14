'use client';

// Growth tools — flash sale radius, stamp card, BTS clips, sponsored boost.
// Port of GrowthTools.jsx (actions are demo-local, data from /vendor/growth).

import { useQuery } from '@tanstack/react-query';
import { Coffee, Flame, MapPin, Megaphone, Play, Sparkles, Star, Zap, LucideIcon } from 'lucide-react';
import { useState } from 'react';

import type { GrowthBlob } from '@spotto/shared';
import Shell from '../../components/shell';
import { api } from '../../lib/api';

const CLIP_ICONS: Record<string, LucideIcon> = { Coffee, Flame, Sparkles };
const TIER_ICONS: Record<string, LucideIcon> = { MapPin, Star, Megaphone };

export default function GrowthPage() {
  const { data: g } = useQuery<GrowthBlob>({
    queryKey: ['growth'],
    queryFn: () => api.get<GrowthBlob>('/vendor/growth'),
  });
  const [flashSent, setFlashSent] = useState(false);
  const [boostTier, setBoostTier] = useState('category');

  return (
    <Shell>
      <div className="grid grid-cols-2 gap-4">
        {/* Flash sale */}
        <div className="card p-6">
          <div className="flex items-center gap-2.5 mb-1">
            <span className="rounded-xl bg-[#fdf0cd] text-[#a97b00] p-2"><Zap size={18} /></span>
            <h2 className="font-extrabold text-[16px]">Flash sale blast</h2>
          </div>
          <p className="text-[13px] text-muted font-semibold mb-4">
            Ping every traveller within {g?.flash?.radiusKm ?? 3} km right now.
          </p>
          <div className="rounded-2xl bg-sand p-4 mb-4">
            <p className="text-[32px] font-extrabold leading-none">{g?.flash?.inRange ?? '—'}</p>
            <p className="text-[12.5px] text-muted font-bold mt-1">
              travellers in range · {g?.flash?.nomads ?? 0} nomads · {g?.flash?.surfers ?? 0} surfers
            </p>
          </div>
          <button
            className="btn-accent w-full py-3"
            disabled={flashSent}
            onClick={() => setFlashSent(true)}
          >
            {flashSent ? 'Blast sent ✓' : 'Send a flash sale'}
          </button>
          <p className="text-[11.5px] text-muted font-semibold mt-2 text-center">
            Last sent {Math.round((g?.flash?.lastSentMins ?? 0) / 60)} hr ago · limit 1 per 3 hr
          </p>
        </div>

        {/* Stamp card */}
        <div className="card p-6">
          <h2 className="font-extrabold text-[16px] mb-1">Digital stamp card</h2>
          <p className="text-[13px] text-muted font-semibold mb-4">
            {g?.stampCard?.stampsNeeded ?? 6} stamps → {g?.stampCard?.reward ?? 'reward'}
          </p>
          <div className="space-y-2.5 mb-4">
            {(g?.stampCard?.buckets ?? []).map((b) => (
              <div key={b.away}>
                <div className="flex justify-between text-[12.5px] font-extrabold mb-1">
                  <span>{b.label}</span>
                  <span className="text-muted">{b.count}</span>
                </div>
                <div className="h-2.5 rounded-full bg-sand overflow-hidden">
                  <div className="h-full rounded-full bg-accent"
                    style={{ width: `${Math.min(100, (b.count / 70) * 100)}%` }} />
                </div>
              </div>
            ))}
          </div>
          <div className="flex justify-between rounded-2xl bg-sand p-4 text-center">
            <div>
              <p className="text-[20px] font-extrabold">{g?.stampCard?.totalHolders ?? 0}</p>
              <p className="text-[11.5px] text-muted font-bold">card holders</p>
            </div>
            <div>
              <p className="text-[20px] font-extrabold">{g?.stampCard?.redeemedThisMonth ?? 0}</p>
              <p className="text-[11.5px] text-muted font-bold">rewards this month</p>
            </div>
          </div>
        </div>

        {/* BTS clips */}
        <div className="card p-6">
          <h2 className="font-extrabold text-[16px] mb-1">Behind-the-scenes clips</h2>
          <p className="text-[13px] text-muted font-semibold mb-4">Short loops shown on your traveller profile.</p>
          <div className="grid grid-cols-3 gap-3">
            {(g?.btsClips ?? []).map((c) => {
              const Icon = CLIP_ICONS[c.icon] ?? Sparkles;
              return (
                <div key={c.id} className="rounded-2xl overflow-hidden text-white relative aspect-[3/4] p-3 flex flex-col justify-between"
                  style={{ background: c.grad }}>
                  <Icon size={18} />
                  <div>
                    <p className="text-[12px] font-extrabold leading-tight">{c.title}</p>
                    <p className="text-[10.5px] font-bold opacity-80 mt-0.5 flex items-center gap-1">
                      <Play size={10} /> {c.views.toLocaleString()} · {c.secs}s
                    </p>
                  </div>
                </div>
              );
            })}
          </div>
        </div>

        {/* Boost */}
        <div className="card p-6">
          <h2 className="font-extrabold text-[16px] mb-1">Boost an event</h2>
          <p className="text-[13px] text-muted font-semibold mb-4">
            {g?.boost?.event?.name} · {g?.boost?.event?.when} · {g?.boost?.event?.dj}
          </p>
          <div className="space-y-2.5 mb-4">
            {(g?.boost?.tiers ?? []).map((t) => {
              const Icon = TIER_ICONS[t.icon] ?? Star;
              const on = boostTier === t.id;
              return (
                <button key={t.id} onClick={() => setBoostTier(t.id)}
                  className={`w-full rounded-xl border p-3.5 flex items-center gap-3 text-left transition-colors ${
                    on ? 'border-accent bg-accent-tint' : 'border-line bg-white hover:border-accent'
                  }`}>
                  <Icon size={18} className={on ? 'text-accent' : 'text-muted'} />
                  <span className="flex-1">
                    <span className="font-extrabold text-[13.5px] flex items-center gap-2">
                      {t.label}
                      {t.best && <span className="text-[10px] bg-sun text-ink rounded-full px-1.5 py-0.5 uppercase">Best value</span>}
                    </span>
                    <span className="block text-[12px] text-muted font-semibold">{t.reach} · {t.days} days</span>
                  </span>
                  <span className="font-extrabold">${t.price}</span>
                </button>
              );
            })}
          </div>
          <button className="btn-accent w-full py-3">Boost for ${g?.boost?.tiers?.find((t) => t.id === boostTier)?.price ?? 0}</button>
        </div>
      </div>
    </Shell>
  );
}
