'use client';

// Growth tools — flash sale radius, stamp card, event promotion, BTS clips.
// Port of the Growth mockup / GrowthTools.jsx. Flash sale (incl. its
// discount %) is a real, persisted action (POST /vendor/growth/flash-sale).
// "Nudge all", the event-promotion toggle, and Boost stay non-functional —
// there's no concept of individually-addressable travellers or a payment
// processor wired up yet (Boost/billing deferred elsewhere this session);
// Record/upload has no file-upload infrastructure (same call made on the
// account page's avatar).

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Coffee, Flame, MapPin, Megaphone, Plus, Play, Sparkles, Star, UserPlus, Video, Zap, LucideIcon,
} from 'lucide-react';
import { useState } from 'react';

import type { GrowthBlob } from '@spotto/shared';
import Shell from '../../components/shell';
import { api } from '../../lib/api';

const CLIP_ICONS: Record<string, LucideIcon> = { Coffee, Flame, Sparkles };
const TIER_ICONS: Record<string, LucideIcon> = { MapPin, Star, Megaphone };
const DISCOUNTS = [20, 30, 50];

export default function GrowthPage() {
  const qc = useQueryClient();
  const { data: g } = useQuery<GrowthBlob>({
    queryKey: ['growth'],
    queryFn: () => api.get<GrowthBlob>('/vendor/growth'),
  });
  const [boostTier, setBoostTier] = useState('category');
  const [discountPct, setDiscountPct] = useState(g?.flash?.discountPct ?? 20);
  const [nudged, setNudged] = useState(false);
  const [promoteOn, setPromoteOn] = useState(true);

  const sendFlashSale = useMutation({
    mutationFn: () => api.post('/vendor/growth/flash-sale', { discountPct }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['growth'] }),
  });

  const flashDisabled = sendFlashSale.isPending || (g?.flash?.lastSentMins ?? Infinity) < 180;

  return (
    <Shell>
      <div className="mb-5">
        <h1 className="text-2xl font-extrabold font-display">Growth tools</h1>
        <p className="text-muted font-semibold mt-0.5">
          Fill slow hours, reward regulars, and get discovered — in a couple of taps.
        </p>
      </div>

      <div className="space-y-4">
        {/* Flash sale */}
        <div className="card p-6">
          <div className="flex items-center gap-2.5 mb-4">
            <span className="rounded-xl bg-[#fdf0cd] text-[#a97b00] p-2"><Zap size={18} /></span>
            <div>
              <h2 className="font-extrabold text-[16px]">Flash sale trigger</h2>
              <p className="text-[13px] text-muted font-semibold">Slow hour? Blast a deal to everyone nearby — instantly.</p>
            </div>
          </div>

          <div className="grid grid-cols-[1fr_auto] gap-6 items-center">
            <div>
              <div className="flex items-center gap-2 mb-4">
                <span className="text-[13px] font-extrabold text-muted">Discount</span>
                <div className="flex gap-1.5">
                  {DISCOUNTS.map((pct) => (
                    <button key={pct} onClick={() => setDiscountPct(pct)}
                      className={`rounded-full border px-3.5 py-1.5 text-[13px] font-extrabold transition-colors ${
                        discountPct === pct ? 'border-accent bg-accent text-white' : 'border-line bg-white hover:border-accent'
                      }`}>
                      {pct}% off
                    </button>
                  ))}
                </div>
              </div>
              <div className="flex items-center gap-3">
                <span className="w-14 h-14 rounded-full border-2 border-line flex flex-col items-center justify-center text-[11px] font-extrabold shrink-0">
                  {g?.flash?.inRange ?? '—'}
                  <span className="text-[8px] text-muted font-bold">IN RANGE</span>
                </span>
                <div>
                  <p className="font-extrabold text-[14px]">{g?.flash?.inRange ?? '—'} travelers within {g?.flash?.radiusKm ?? 3} km</p>
                  <p className="text-[12.5px] text-muted font-semibold">
                    {g?.flash?.nomads ?? 0} nomads · {g?.flash?.surfers ?? 0} surfers active now
                  </p>
                  <p className="text-[12.5px] text-muted font-semibold">
                    Last flash sent {Math.round((g?.flash?.lastSentMins ?? 0) / 60)}h ago
                  </p>
                </div>
              </div>
            </div>

            <div className="text-center">
              <button
                onClick={() => !flashDisabled && sendFlashSale.mutate()}
                disabled={flashDisabled}
                className="w-32 h-32 rounded-full flex flex-col items-center justify-center text-white font-extrabold gap-1 disabled:opacity-60 transition-opacity"
                style={{ background: 'linear-gradient(135deg, #eaa200, #c97f00)' }}
              >
                <Zap size={22} />
                <span className="text-[13px]">{sendFlashSale.isPending ? 'Sending…' : 'Flash sale'}</span>
                <span className="text-[10.5px] font-bold opacity-90">{discountPct}% OFF · {g?.flash?.radiusKm ?? 3}km</span>
              </button>
              <p className="text-[11px] text-muted font-semibold mt-2">Press &amp; hold to broadcast</p>
            </div>
          </div>
          {sendFlashSale.isError && (
            <p className="text-[12.5px] font-semibold mt-3 text-center" style={{ color: '#c2666f' }}>
              {sendFlashSale.error instanceof Error ? sendFlashSale.error.message : 'Could not send — try again.'}
            </p>
          )}
        </div>

        <div className="grid md:grid-cols-2 gap-4">
          {/* Stamp card */}
          <div className="card p-6">
            <div className="flex items-center gap-2.5 mb-4">
              <span className="rounded-xl bg-accent-tint text-accent p-2"><Star size={18} /></span>
              <div>
                <h2 className="font-extrabold text-[16px]">Digital stamp cards</h2>
                <p className="text-[13px] text-muted font-semibold">
                  {g?.stampCard?.stampsNeeded ?? 6} stamps → {g?.stampCard?.reward ?? 'reward'} · {g?.stampCard?.totalHolders ?? 0} active cards
                </p>
              </div>
            </div>

            <div className="rounded-2xl bg-accent-tint p-4 flex items-center justify-between gap-3 mb-4">
              <div>
                <p className="font-extrabold text-[22px] leading-none">{g?.stampCard?.buckets?.[0]?.count ?? 0}</p>
                <p className="text-[12.5px] font-bold mt-1">travelers are ONE stamp away</p>
                <p className="text-[11.5px] text-muted font-semibold">Nudge them in and you close {g?.stampCard?.buckets?.[0]?.count ?? 0} rewards today.</p>
              </div>
              <button
                onClick={() => { setNudged(true); setTimeout(() => setNudged(false), 2000); }}
                className="rounded-xl bg-accent text-white px-3.5 py-2 font-extrabold text-[12.5px] flex items-center gap-1.5 shrink-0"
              >
                <UserPlus size={14} /> {nudged ? 'Nudged!' : 'Nudge all'}
              </button>
            </div>

            <div className="grid grid-cols-3 gap-2">
              {(g?.stampCard?.buckets ?? []).map((b) => (
                <div key={b.away} className="rounded-xl border border-line p-3 text-center">
                  <div className="flex justify-center gap-1 mb-2">
                    {Array.from({ length: 6 }).map((_, i) => (
                      <span key={i} className="w-1.5 h-1.5 rounded-full" style={{ background: i < 6 - b.away ? 'var(--accent)' : 'var(--line)' }} />
                    ))}
                  </div>
                  <p className="font-extrabold text-[18px]">{b.count}</p>
                  <p className="text-[11px] text-muted font-semibold">{b.label}</p>
                </div>
              ))}
            </div>
          </div>

          {/* Promote my event */}
          <div className="card p-6">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-2.5">
                <span className="rounded-xl bg-[#fde4d8] text-coral p-2"><Megaphone size={18} /></span>
                <div>
                  <h2 className="font-extrabold text-[16px]">Promote my event</h2>
                  <p className="text-[13px] text-muted font-semibold">
                    {g?.boost?.event?.name ?? 'Your next event'} · {g?.boost?.event?.when ?? ''}
                  </p>
                </div>
              </div>
              <label className="relative inline-flex items-center cursor-pointer shrink-0" title="Coming soon">
                <input type="checkbox" className="sr-only peer" checked={promoteOn} onChange={(e) => setPromoteOn(e.target.checked)} />
                <span className={`w-10 h-6 rounded-full relative transition-colors ${promoteOn ? 'bg-accent' : 'bg-line'}`}>
                  <span className={`absolute top-0.5 w-5 h-5 rounded-full bg-white transition-all ${promoteOn ? 'left-[22px]' : 'left-0.5'}`} />
                </span>
              </label>
            </div>

            <div className="grid grid-cols-3 gap-2 mb-4">
              {(g?.boost?.tiers ?? []).map((t) => {
                const Icon = TIER_ICONS[t.icon] ?? Star;
                const on = boostTier === t.id;
                return (
                  <button key={t.id} onClick={() => setBoostTier(t.id)}
                    className={`relative rounded-xl border p-3 text-left transition-colors ${
                      on ? 'border-accent bg-accent-tint' : 'border-line bg-white hover:border-accent'
                    }`}>
                    {t.best && (
                      <span className="absolute -top-2 right-2 text-[9px] bg-sun text-ink rounded-full px-1.5 py-0.5 uppercase font-extrabold">Best value</span>
                    )}
                    <Icon size={16} className={on ? 'text-accent' : 'text-muted'} />
                    <p className="font-extrabold text-[12px] mt-1.5 leading-tight">{t.label}</p>
                    <p className="text-[10.5px] text-muted font-semibold mt-0.5">{t.reach}</p>
                    <p className="text-[10.5px] text-muted font-semibold">{t.days} days</p>
                    <p className="font-extrabold text-[13px] mt-1">${t.price}</p>
                  </button>
                );
              })}
            </div>

            <div className="flex items-center justify-between">
              <div>
                <p className="text-[11.5px] text-muted font-semibold">Total to feature {g?.boost?.event?.name ? g.boost.event.name.split(' ')[0] : 'your event'}…</p>
                <p className="font-extrabold text-[18px]">
                  ${g?.boost?.tiers?.find((t) => t.id === boostTier)?.price ?? 0}
                  <span className="text-[12px] text-muted font-semibold"> /{g?.boost?.tiers?.find((t) => t.id === boostTier)?.days ?? 0} days</span>
                </p>
              </div>
              <button className="rounded-xl bg-accent text-white opacity-50 cursor-not-allowed px-4 py-2.5 font-extrabold text-[13.5px] flex items-center gap-2"
                disabled title="Paid boosts are coming soon">
                <Megaphone size={15} /> Boost event
              </button>
            </div>
          </div>
        </div>

        {/* BTS clips */}
        <div className="card p-6">
          <div className="flex items-center gap-2.5 mb-4">
            <span className="rounded-xl bg-accent-tint text-accent p-2"><Video size={18} /></span>
            <div>
              <h2 className="font-extrabold text-[16px]">Behind the scenes</h2>
              <p className="text-[13px] text-muted font-semibold">10-second clips of your craft — they build trust &amp; bookings.</p>
            </div>
          </div>
          <div className="grid grid-cols-4 gap-3">
            <div className="rounded-2xl border-2 border-dashed border-line aspect-[3/4] flex flex-col items-center justify-center gap-2 text-muted opacity-60 cursor-not-allowed"
              title="Recording/uploading clips is coming soon">
              <Plus size={20} />
              <p className="text-[11.5px] font-extrabold text-center px-2">Record / upload<br />10s clip</p>
            </div>
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
      </div>
    </Shell>
  );
}
