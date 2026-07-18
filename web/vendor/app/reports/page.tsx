'use client';

// Reports — weekly traffic + category trend + popular times with derived
// headline stats and an insight callout. Port of the Reports mockup.
// TrendChart/PopularTimes/TrafficChart already match their mockup panels
// closely and are reused as-is; this rebuild adds the stat-card header,
// date-range/category subtitle, and the insight box, all computed from
// the same week/trend/popular arrays already being fetched.

import { useQuery } from '@tanstack/react-query';
import { Award, Clock, Download, Eye, Sparkles, Ticket, LucideIcon } from 'lucide-react';

import { PopularTimes, TrafficChart, TrendChart } from '@spotto/charts';
import type { DashboardBlob } from '@spotto/shared';
import Shell, { RoleLockedCard, useBusiness, useMe } from '../../components/shell';
import { api } from '../../lib/api';

const ACCENT = '#0b6f86';

function hourLabel(h: number) {
  const ap = h >= 12 ? 'pm' : 'am';
  const h12 = h % 12 === 0 ? 12 : h % 12;
  return `${h12}${ap}`;
}

function dateRangeLabel() {
  const end = new Date();
  const start = new Date(end.getTime() - 6 * 86_400_000);
  const fmt = (d: Date) => d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
  return `${fmt(start)} – ${fmt(end)}`;
}

export default function ReportsPage() {
  const { data: me } = useMe();
  const isStaff = me?.vendorRole === 'staff';
  const { data: business } = useBusiness();
  const { data: r } = useQuery<DashboardBlob>({
    queryKey: ['reports'],
    queryFn: () => api.get<DashboardBlob>('/vendor/reports'),
    enabled: !isStaff,
  });

  if (isStaff) {
    return (
      <Shell>
        <div className="mb-5">
          <h1 className="text-2xl font-extrabold font-display">Weekly report</h1>
        </div>
        <RoleLockedCard what="Reports" />
      </Shell>
    );
  }

  const totalViews = (r?.week ?? []).reduce((a, d) => a + d.views, 0);
  const totalRedemptions = (r?.week ?? []).reduce((a, d) => a + d.redemptions, 0);

  const trend = r?.trend ?? [];
  const last = trend[trend.length - 1];
  const prev = trend[trend.length - 2];
  const vsCategoryPct = last && last.cat > 0 ? Math.round(((last.you - last.cat) / last.cat) * 100) : 0;
  // Approximated from the trend index's last two weeks — this app doesn't
  // track a second week of daily view/redemption totals to diff exactly.
  const weekOverWeekPct = last && prev && prev.you > 0 ? Math.round(((last.you - prev.you) / prev.you) * 100) : 0;

  const popular = r?.popular ?? [];
  const peak = popular.reduce((best, p) => (p.v > (best?.v ?? -1) ? p : best), popular[0]);
  const peakLabel = peak ? hourLabel(peak.h) : '—';

  const town = business?.town ?? 'your area';
  const category = business?.category ?? 'business';

  return (
    <Shell>
      <div className="flex items-start justify-between mb-5">
        <div>
          <h1 className="text-2xl font-extrabold font-display">Weekly report</h1>
          <p className="text-muted font-semibold mt-0.5">
            {dateRangeLabel()} · {town} {category.toLowerCase()} category
          </p>
        </div>
        <button
          className="rounded-xl border border-line bg-white px-4 py-2.5 font-extrabold text-[13.5px] flex items-center gap-2 hover:border-accent transition-colors"
          onClick={() => window.print()}
        >
          <Download size={16} /> Export PDF
        </button>
      </div>

      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-4">
        <StatCard icon={Eye} label="Profile views" value={totalViews.toLocaleString()}
          delta={weekOverWeekPct ? `${weekOverWeekPct > 0 ? '+' : ''}${weekOverWeekPct}% vs last week` : undefined} />
        <StatCard icon={Ticket} label="Redemptions" value={totalRedemptions.toLocaleString()}
          delta={weekOverWeekPct ? `${weekOverWeekPct > 0 ? '+' : ''}${weekOverWeekPct}% vs last week` : undefined} />
        <StatCard icon={Clock} label="Peak hour" value={peakLabel} delta="busiest footfall" />
        <StatCard icon={Award} label="vs category" value={`${vsCategoryPct > 0 ? '+' : ''}${vsCategoryPct}%`}
          delta={`above ${town} avg`} />
      </div>

      <div className="space-y-4">
        <div className="card p-5">
          <div className="flex items-center justify-between mb-3">
            <div>
              <h2 className="font-extrabold text-[15px]">Your deals vs category trends</h2>
              <p className="text-[12.5px] text-muted font-semibold">Redemption index · 6 weeks · {town} {category.toLowerCase()}s</p>
            </div>
            <div className="flex items-center gap-3 text-[12px] font-bold text-muted shrink-0">
              <span className="flex items-center gap-1.5"><span className="w-3 h-0.5 rounded-full bg-accent" /> You</span>
              <span className="flex items-center gap-1.5"><span className="w-3 border-t-2" style={{ borderColor: '#9bb0ad', borderStyle: 'dashed' }} /> Category avg</span>
            </div>
          </div>
          {r?.trend && <TrendChart data={r.trend} accent={ACCENT} />}
          {vsCategoryPct !== 0 && (
            <div className="rounded-xl bg-accent-tint p-3.5 flex gap-2.5 mt-3">
              <Sparkles size={16} className="text-accent shrink-0 mt-0.5" />
              <p className="text-[13px] font-semibold">
                Your flash offers are pulling <span className="font-extrabold">{Math.abs(vsCategoryPct)}% {vsCategoryPct >= 0 ? 'more' : 'fewer'} redemptions</span> than
                the average {town} {category.toLowerCase()}. Mornings ({peakLabel}) are your strongest window — consider a recurring breakfast deal.
              </p>
            </div>
          )}
        </div>
        <div className="grid grid-cols-2 gap-4">
          <div className="card p-5">
            <h2 className="font-extrabold text-[15px]">Daily traffic</h2>
            <p className="text-[12.5px] text-muted font-semibold mb-3">Views with redemption share</p>
            {r?.week && <TrafficChart data={r.week} accent={ACCENT} />}
          </div>
          <div className="card p-5">
            <h2 className="font-extrabold text-[15px]">Popular times</h2>
            <p className="text-[12.5px] text-muted font-semibold mb-3">Avg footfall by hour · peak {peakLabel}</p>
            {r?.popular && <PopularTimes data={r.popular} accent={ACCENT} />}
          </div>
        </div>
      </div>
    </Shell>
  );
}

function StatCard({ icon: Icon, label, value, delta }: { icon: LucideIcon; label: string; value: string; delta?: string }) {
  return (
    <div className="card p-4">
      <p className="text-[12px] text-muted font-extrabold flex items-center gap-1.5 mb-1.5">
        <Icon size={13} /> {label}
      </p>
      <p className="text-[22px] font-extrabold leading-tight">{value}</p>
      {delta && <p className="text-[11.5px] text-muted font-bold mt-0.5">{delta}</p>}
    </div>
  );
}
