'use client';

// Dashboard — KPIs, weekly traffic, popular times, trend, and the live
// redemption feed (3s polling). Port of VendorDashboard.jsx.

import { useQuery } from '@tanstack/react-query';
import { Eye, Tag, Ticket, TrendingUp, LucideIcon } from 'lucide-react';

import { PopularTimes, TrafficChart, TrendChart } from '@spotto/charts';
import { timeAgo, type Claim, type DashboardBlob } from '@spotto/shared';
import Shell from '../../components/shell';
import { api } from '../../lib/api';

const ACCENT = '#0b6f86';
const KPI_ICONS: Record<string, LucideIcon> = { Tag, Ticket, Eye };
const KPI_TONES: Record<string, string> = {
  teal: 'bg-accent-tint text-accent',
  coral: 'bg-[#fde4d8] text-coral',
  sun: 'bg-[#fdf0cd] text-[#a97b00]',
};

export default function DashboardPage() {
  const { data: dash } = useQuery<DashboardBlob>({
    queryKey: ['dashboard'],
    queryFn: () => api.get<DashboardBlob>('/vendor/dashboard'),
  });
  const { data: redemptions = [] } = useQuery<Claim[]>({
    queryKey: ['redemptions'],
    queryFn: () => api.get<Claim[]>('/vendor/redemptions'),
    refetchInterval: 3000,
  });

  return (
    <Shell>
      <div className="grid grid-cols-3 gap-4 mb-6">
        {(dash?.kpis ?? []).map((k) => {
          const Icon = KPI_ICONS[k.icon] ?? Tag;
          return (
            <div key={k.id} className="card p-5 flex items-start gap-4">
              <span className={`rounded-xl p-2.5 ${KPI_TONES[k.tone] ?? KPI_TONES.teal}`}>
                <Icon size={20} />
              </span>
              <div>
                <p className="text-[13px] text-muted font-extrabold">{k.label}</p>
                <p className="text-[26px] font-extrabold leading-tight">
                  {k.value.toLocaleString()}
                  {k.delta != null && (
                    <span className="ml-2 text-[13px] text-accent align-middle inline-flex items-center gap-0.5">
                      <TrendingUp size={13} /> +{k.delta}%
                    </span>
                  )}
                </p>
                <p className="text-[12px] text-muted font-semibold">{k.hint}</p>
              </div>
            </div>
          );
        })}
      </div>

      <div className="grid grid-cols-[1fr_360px] gap-4 items-start">
        <div className="space-y-4">
          <div className="card p-5">
            <h2 className="font-extrabold mb-1">Weekly traffic</h2>
            <p className="text-[12.5px] text-muted font-semibold mb-3">Profile views with redemptions overlaid</p>
            {dash?.week && <TrafficChart data={dash.week} accent={ACCENT} />}
          </div>
          <div className="card p-5">
            <h2 className="font-extrabold mb-1">Popular times</h2>
            <p className="text-[12.5px] text-muted font-semibold mb-3">Relative footfall by hour</p>
            {dash?.popular && <PopularTimes data={dash.popular} accent={ACCENT} />}
          </div>
          <div className="card p-5">
            <h2 className="font-extrabold mb-1">You vs. category</h2>
            <p className="text-[12.5px] text-muted font-semibold mb-3">Weekly redemptions against the Weligama café index</p>
            {dash?.trend && <TrendChart data={dash.trend} accent={ACCENT} />}
          </div>
        </div>

        <div className="card p-5">
          <div className="flex items-center justify-between mb-3">
            <h2 className="font-extrabold">Live redemptions</h2>
            <span className="flex items-center gap-1.5 text-[11.5px] font-extrabold text-accent">
              <span className="w-2 h-2 rounded-full bg-accent animate-pulse" /> LIVE
            </span>
          </div>
          <div className="space-y-1">
            {redemptions.length === 0 && (
              <p className="text-[13px] text-muted font-semibold py-4">
                No redemptions yet — validate a code and it lands here.
              </p>
            )}
            {redemptions.map((r) => (
              <div key={r.id} className="feed-in flex items-start gap-3 py-2.5 border-b border-line last:border-0">
                <span className="rounded-lg bg-accent-tint text-accent p-1.5 mt-0.5">
                  <Ticket size={15} />
                </span>
                <div className="min-w-0 flex-1">
                  <p className="text-[13.5px] font-extrabold truncate">
                    {r.travellerName || 'Traveller'}
                    {r.travellerCountry ? ` (${r.travellerCountry})` : ''}
                  </p>
                  <p className="text-[12.5px] text-muted font-semibold truncate">{r.offerLabel || r.code}</p>
                </div>
                <span className="text-[11.5px] text-muted font-bold whitespace-nowrap">{timeAgo(r.redeemedAt)}</span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </Shell>
  );
}
