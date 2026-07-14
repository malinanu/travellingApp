'use client';

// Reports — weekly traffic + category trend + popular times with an
// export stub. Port of Reports.jsx.

import { useQuery } from '@tanstack/react-query';
import { Download } from 'lucide-react';

import { PopularTimes, TrafficChart, TrendChart } from '@spotto/charts';
import type { DashboardBlob } from '@spotto/shared';
import Shell from '../../components/shell';
import { api } from '../../lib/api';

const ACCENT = '#0b6f86';

export default function ReportsPage() {
  const { data: r } = useQuery<DashboardBlob>({
    queryKey: ['reports'],
    queryFn: () => api.get<DashboardBlob>('/vendor/reports'),
  });

  const totalViews = (r?.week ?? []).reduce((a, d) => a + d.views, 0);
  const totalRedemptions = (r?.week ?? []).reduce((a, d) => a + d.redemptions, 0);
  const conversion = totalViews ? ((totalRedemptions / totalViews) * 100).toFixed(1) : '0';

  return (
    <Shell>
      <div className="flex items-center justify-between mb-5">
        <div className="flex gap-6">
          <Stat label="Views this week" value={totalViews.toLocaleString()} />
          <Stat label="Redemptions" value={totalRedemptions.toLocaleString()} />
          <Stat label="Conversion" value={`${conversion}%`} />
        </div>
        <button
          className="rounded-xl border border-line bg-white px-4 py-2.5 font-extrabold text-[13.5px] flex items-center gap-2 hover:border-accent transition-colors"
          onClick={() => window.print()}
        >
          <Download size={16} /> Export report
        </button>
      </div>

      <div className="space-y-4">
        <div className="card p-5">
          <h2 className="font-extrabold mb-3">Weekly traffic</h2>
          {r?.week && <TrafficChart data={r.week} accent={ACCENT} />}
        </div>
        <div className="grid grid-cols-2 gap-4">
          <div className="card p-5">
            <h2 className="font-extrabold mb-3">You vs. category</h2>
            {r?.trend && <TrendChart data={r.trend} accent={ACCENT} />}
          </div>
          <div className="card p-5">
            <h2 className="font-extrabold mb-3">Popular times</h2>
            {r?.popular && <PopularTimes data={r.popular} accent={ACCENT} />}
          </div>
        </div>
      </div>
    </Shell>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-[12px] text-muted font-extrabold">{label}</p>
      <p className="text-[22px] font-extrabold leading-tight">{value}</p>
    </div>
  );
}
