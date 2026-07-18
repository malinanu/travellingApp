'use client';

// Dashboard — greeting + KPIs, weekly traffic, live redemption feed (3s
// polling), and active offers. Port of VendorDashboard.jsx.

import { useQuery } from '@tanstack/react-query';
import {
  ArrowRight, Eye, Plus, Tag, Ticket, TrendingDown, TrendingUp, LucideIcon,
} from 'lucide-react';
import Link from 'next/link';
import { useState } from 'react';

import { Donut, TrafficChart } from '@spotto/charts';
import { timeAgo, type Claim, type DashboardBlob, type Offer } from '@spotto/shared';
import Shell, { useBusiness } from '../../components/shell';
import OfferDetailDrawer from '../../components/offer-detail-drawer';
import { api } from '../../lib/api';

const ACCENT = '#0b6f86';
const KPI_ICONS: Record<string, LucideIcon> = { Tag, Ticket, Eye };
const KPI_TONES: Record<string, { bg: string; fg: string }> = {
  teal: { bg: '#d6efec', fg: '#0b6f86' },
  coral: { bg: '#ffe5d6', fg: '#f0682f' },
  sun: { bg: '#fbeec4', fg: '#c98a00' },
};

const STYLE_DOT: Record<string, string> = {
  'Digital Nomad': '#0b6f86',
  'Surf Traveler': '#f0682f',
  Luxury: '#c98a00',
  Backpacker: '#5a5f9e',
};

function greeting() {
  const h = new Date().getHours();
  if (h < 12) return 'Good morning';
  if (h < 18) return 'Good afternoon';
  return 'Good evening';
}

function initial(name: string) {
  return name.trim().split(/\s+/)[0]?.[0]?.toUpperCase() ?? '?';
}

export default function DashboardPage() {
  const { data: business } = useBusiness();
  const { data: dash } = useQuery<DashboardBlob>({
    queryKey: ['dashboard'],
    queryFn: () => api.get<DashboardBlob>('/vendor/dashboard'),
  });
  const { data: redemptions = [] } = useQuery<Claim[]>({
    queryKey: ['redemptions'],
    queryFn: () => api.get<Claim[]>('/vendor/redemptions'),
    refetchInterval: 3000,
  });
  const { data: offers = [] } = useQuery<Offer[]>({
    queryKey: ['offers'],
    queryFn: () => api.get<Offer[]>('/vendor/offers'),
  });
  const [detailOffer, setDetailOffer] = useState<Offer | null>(null);

  const activeOffers = offers.filter((o) => o.status === 'live' || o.status === 'ending');

  return (
    <Shell>
      <div className="space-y-6">
        <div className="flex items-end justify-between gap-4">
          <div className="min-w-0">
            <h1 className="font-display font-extrabold tracking-tight text-[28px] leading-[1.15]">
              {greeting()}{business?.owner ? `, ${business.owner.split(' ')[0]}` : ''} 👋
            </h1>
            <p className="text-[14.5px] text-muted font-semibold mt-1.5">
              Here&rsquo;s how {business?.name?.split(' ')[0] ?? 'your business'} is doing in {business?.town ?? 'town'} today.
            </p>
          </div>
          <Link href="/offers"
            className="btn-accent rounded-xl px-4 py-2.5 font-extrabold text-[14px] flex items-center gap-2 active:scale-[0.98] transition-transform shrink-0 whitespace-nowrap">
            <Plus size={18} strokeWidth={2.5} /> New offer
          </Link>
        </div>

        <div className="grid gap-4" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))' }}>
          {(dash?.kpis ?? []).map((k) => {
            const Icon = KPI_ICONS[k.icon] ?? Tag;
            const tone = KPI_TONES[k.tone] ?? KPI_TONES.teal;
            const positive = (k.delta ?? 0) >= 0;
            return (
              <div key={k.id} className="card p-5">
                <div className="flex items-start justify-between">
                  <div className="w-11 h-11 rounded-xl flex items-center justify-center" style={{ background: tone.bg, color: tone.fg }}>
                    <Icon size={22} />
                  </div>
                  {k.delta != null && (
                    <div className="flex items-center gap-1 rounded-full px-2 py-1 text-[12.5px] font-extrabold"
                      style={{ background: positive ? '#e7f6ec' : '#fde8e4', color: positive ? '#16a34a' : '#dc4a2d' }}>
                      {positive ? <TrendingUp size={13} /> : <TrendingDown size={13} />}
                      {positive ? '+' : ''}{k.delta}%
                    </div>
                  )}
                </div>
                <div className="font-display font-extrabold tracking-tight text-[34px] mt-4">{k.value.toLocaleString()}</div>
                <div className="flex items-center gap-2 mt-1">
                  <span className="text-[14px] font-bold text-ink-soft">{k.label}</span>
                  <span className="text-[12.5px] text-muted font-semibold">· {k.hint}</span>
                </div>
              </div>
            );
          })}
        </div>

        <div className="grid gap-6 grid-cols-1 lg:[grid-template-columns:minmax(0,1.5fr)_minmax(0,1fr)]">
          <div className="card p-5">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="font-extrabold text-[16px]">Traffic this week</h3>
                <p className="text-[12.5px] text-muted font-semibold">Profile views · redemptions</p>
              </div>
              <div className="flex items-center gap-3 text-[12px] text-muted font-bold">
                <span className="flex items-center gap-1.5"><span className="w-2.5 h-2.5 rounded-sm bg-accent opacity-25" />Views</span>
                <span className="flex items-center gap-1.5"><span className="w-2.5 h-2.5 rounded-sm bg-accent" />Redeemed</span>
              </div>
            </div>
            <div className="mt-3">{dash?.week && <TrafficChart data={dash.week} accent={ACCENT} />}</div>
          </div>

          <div className="card p-5">
            <div className="flex items-center justify-between">
              <h3 className="font-extrabold text-[16px]">Live redemptions</h3>
              <span className="flex items-center gap-1.5 text-[12px] font-bold text-[#16a34a]">
                <span className="w-2 h-2 rounded-full bg-[#16a34a]" style={{ boxShadow: '0 0 0 3px #e7f6ec' }} /> Live
              </span>
            </div>
            <div className="mt-1">
              {redemptions.length === 0 && (
                <p className="text-[13px] text-muted font-semibold py-4">
                  No redemptions yet — validate a code and it lands here.
                </p>
              )}
              {redemptions.map((r) => {
                const dotColor = r.travellerStyle ? STYLE_DOT[r.travellerStyle] ?? '#6c8582' : null;
                return (
                  <div key={r.id} className="feed-in flex items-center gap-3 py-2.5 border-t border-line">
                    <div className="w-9 h-9 rounded-full bg-sand text-ink flex items-center justify-center text-[12px] font-extrabold shrink-0">
                      {initial(r.travellerName || 'T')}
                    </div>
                    <div className="min-w-0 flex-1">
                      <p className="font-bold text-[13.5px] truncate">
                        {r.travellerName || 'Traveller'}{r.travellerCountry ? ` (${r.travellerCountry})` : ''}
                      </p>
                      <p className="text-[12px] text-muted font-semibold truncate">{r.offerLabel || r.code}</p>
                    </div>
                    <div className="text-right shrink-0">
                      {dotColor && (
                        <div className="flex items-center gap-1 justify-end text-[11px] font-bold" style={{ color: dotColor }}>
                          <span className="w-1.5 h-1.5 rounded-full" style={{ background: dotColor }} /> {r.travellerStyle}
                        </div>
                      )}
                      <div className="text-[11.5px] text-muted font-semibold">{timeAgo(r.redeemedAt)}</div>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        </div>

        <div className="card p-5">
          <div className="flex items-center justify-between mb-1">
            <h3 className="font-extrabold text-[16px]">Active offers</h3>
            <Link href="/offers" className="text-[13px] font-extrabold text-accent flex items-center gap-1">
              Manage <ArrowRight size={15} />
            </Link>
          </div>
          {activeOffers.length === 0 && (
            <p className="text-[13px] text-muted font-semibold py-4">No live offers right now — create one to get started.</p>
          )}
          {activeOffers.map((o) => {
            const ending = o.status === 'ending';
            return (
              <div key={o.id} className="flex items-center gap-4 py-3.5 border-t border-line cursor-pointer"
                onClick={() => setDetailOffer(o)}>
                <Donut value={o.claimed} max={o.limit} accent={ending ? '#f0682f' : ACCENT} size={50} stroke={7} />
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <span className="font-bold text-[14.5px] truncate">{o.title}</span>
                    <span className="shrink-0 rounded-full px-2 py-0.5 text-[11px] font-extrabold uppercase tracking-wide"
                      style={{ background: ending ? '#fde8e4' : '#e7f6ec', color: ending ? '#dc4a2d' : '#16a34a' }}>
                      {ending ? 'Ending soon' : 'Live'}
                    </span>
                  </div>
                  <div className="text-[12.5px] text-muted font-semibold mt-0.5">
                    {o.claimed}/{o.limit} claimed · {o.redeemed} redeemed · ends {o.end}
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      </div>
      <OfferDetailDrawer offer={detailOffer} onClose={() => setDetailOffer(null)} />
    </Shell>
  );
}
