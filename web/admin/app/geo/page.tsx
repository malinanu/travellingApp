'use client';

// Geo-analytics — demand/supply map + region detail + opportunity
// leaderboard (gap thresholds: >=28 critical, >=12 under-served, <=-10
// saturated). Port of GeoAnalytics.jsx.

import { useQuery } from '@tanstack/react-query';
import { useState } from 'react';

import type { Region } from '@spotto/shared';
import Shell from '../../components/shell';
import SriLankaMap, { regionTone } from '../../components/sri-lanka-map';
import { api } from '../../lib/api';

export default function GeoPage() {
  const { data: regions = [] } = useQuery<Region[]>({
    queryKey: ['admin-regions'],
    queryFn: () => api.get<Region[]>('/admin/regions'),
  });
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const selected = regions.find((r) => r.id === selectedId) ?? null;

  const leaderboard = [...regions]
    .map((r) => ({ ...r, gap: r.travelers - r.vendors }))
    .sort((a, b) => b.gap - a.gap);

  return (
    <Shell>
      <div className="grid grid-cols-[1fr_360px] gap-5 items-start">
        <div className="panel p-4" style={{ height: 640 }}>
          <SriLankaMap regions={regions} selected={selectedId} onSelect={setSelectedId} />
        </div>

        <div className="space-y-4">
          {selected && (
            <div className="panel p-5">
              <h2 className="font-bold text-[16px]">{selected.name}</h2>
              <p className="text-[12.5px] font-bold mt-1" style={{ color: regionTone(selected.travelers - selected.vendors).ring }}>
                {regionTone(selected.travelers - selected.vendors).label}
              </p>
              <div className="grid grid-cols-2 gap-3 mt-3">
                <div className="rounded-xl p-3" style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
                  <p className="text-[11px] font-bold text-muted">Travelers</p>
                  <p className="text-[20px] font-bold">{selected.travelers}</p>
                </div>
                <div className="rounded-xl p-3" style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
                  <p className="text-[11px] font-bold text-muted">Vendors</p>
                  <p className="text-[20px] font-bold">{selected.vendors}</p>
                </div>
              </div>
            </div>
          )}

          <div className="panel overflow-hidden">
            <p className="px-4 py-3 border-b border-line font-bold text-[13.5px]">Opportunity leaderboard</p>
            <div className="max-h-[440px] overflow-y-auto">
              {leaderboard.map((r) => {
                const tone = regionTone(r.gap);
                return (
                  <button key={r.id} onClick={() => setSelectedId(r.id)}
                    className="w-full text-left px-4 py-3 border-b border-line last:border-0 flex items-center justify-between"
                    style={{ background: selectedId === r.id ? 'color-mix(in srgb, var(--accent) 10%, transparent)' : 'transparent' }}>
                    <span>
                      <span className="block font-bold text-[13.5px]">{r.name}</span>
                      <span className="block text-[11.5px] font-semibold" style={{ color: tone.ring }}>{tone.label}</span>
                    </span>
                    <span className="font-bold text-[14px]" style={{ color: tone.ring }}>{r.gap > 0 ? '+' : ''}{r.gap}</span>
                  </button>
                );
              })}
            </div>
          </div>
        </div>
      </div>
    </Shell>
  );
}
