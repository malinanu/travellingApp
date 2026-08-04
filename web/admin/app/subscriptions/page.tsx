'use client';

// Subscriptions — 4 summary tiles, state tabs, sticky-header table with
// hover upsell action. Port of Subscriptions.jsx.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ArrowUpCircle } from 'lucide-react';
import { useState } from 'react';

import type { Subscription } from '@spotto/shared';
import Shell from '../../components/shell';
import { InlineErrorBanner } from '../../components/route-status';
import { api } from '../../lib/api';

const STATE_TABS = ['all', 'active', 'soon', 'overdue', 'trial', 'free'] as const;
const STATE_STYLE: Record<string, string> = {
  active: '#16a34a', soon: '#d97706', overdue: '#e11d48', trial: '#0b6f86', free: '#6c8582',
};

export default function SubscriptionsPage() {
  const qc = useQueryClient();
  const [tab, setTab] = useState<(typeof STATE_TABS)[number]>('all');
  const { data: subs = [], isError, isLoading } = useQuery<Subscription[]>({
    queryKey: ['admin-subs'],
    queryFn: () => api.get<Subscription[]>('/admin/subscriptions'),
  });

  const upgrade = useMutation({
    mutationFn: (id: string) => api.post(`/admin/subscriptions/${id}/upgrade`, {}),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin-subs'] }),
  });

  const premium = subs.filter((s) => s.tier === 'Premium');
  const mrr = premium.reduce((a, s) => a + s.mrr, 0);
  const overdue = subs.filter((s) => s.state === 'overdue').length;
  const free = subs.filter((s) => s.tier === 'Free').length;

  const filtered = subs.filter((s) => tab === 'all' || s.state === tab);

  return (
    <Shell>
      {isError && <InlineErrorBanner message="Couldn't load subscriptions — try refreshing." />}
      <div className="grid grid-cols-4 gap-4 mb-5">
        <Tile label="Platform MRR" value={`$${mrr.toLocaleString()}`} />
        <Tile label="Premium vendors" value={`${premium.length}`} />
        <Tile label="Free vendors" value={`${free}`} />
        <Tile label="Overdue" value={`${overdue}`} tone={overdue > 0 ? '#e11d48' : undefined} />
      </div>

      <div className="flex gap-1.5 mb-4">
        {STATE_TABS.map((t) => (
          <button key={t} onClick={() => setTab(t)} className="btn capitalize"
            style={{
              background: tab === t ? 'var(--accent)' : 'var(--surface)',
              color: tab === t ? '#fff' : 'var(--ink)',
              borderColor: tab === t ? 'var(--accent)' : 'var(--line)',
            }}>
            {t}
          </button>
        ))}
      </div>

      <div className="panel overflow-hidden">
        <table className="w-full text-[13.5px]">
          <thead className="sticky top-0" style={{ background: 'var(--surface)' }}>
            <tr className="text-left border-b border-line">
              <Th>Vendor</Th><Th>Town</Th><Th>Tier</Th><Th>MRR</Th><Th>Renews</Th><Th>State</Th><Th />
            </tr>
          </thead>
          <tbody>
            {filtered.map((s) => (
              <tr key={s.id} className="border-b border-line last:border-0 group">
                <td className="px-4 py-3 font-bold">{s.vendor}</td>
                <td className="px-4 py-3 text-muted font-medium">{s.town}</td>
                <td className="px-4 py-3 font-semibold">{s.tier}</td>
                <td className="px-4 py-3 font-bold">{s.mrr ? `$${s.mrr}` : '—'}</td>
                <td className="px-4 py-3 text-muted font-medium">
                  {s.renewsAt ? new Date(s.renewsAt).toLocaleDateString() : s.since}
                </td>
                <td className="px-4 py-3">
                  <span className="font-bold text-[12px] uppercase" style={{ color: STATE_STYLE[s.state] }}>{s.state}</span>
                </td>
                <td className="px-4 py-3 text-right">
                  {s.tier === 'Free' && (
                    <button
                      className="btn !py-1.5 opacity-0 group-hover:opacity-100 transition-opacity flex items-center gap-1.5 ml-auto"
                      onClick={() => upgrade.mutate(s.id)}
                    >
                      <ArrowUpCircle size={13} /> Upsell
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Shell>
  );
}

function Tile({ label, value, tone }: { label: string; value: string; tone?: string }) {
  return (
    <div className="panel p-4">
      <p className="text-[11.5px] font-bold text-muted">{label}</p>
      <p className="text-[24px] font-bold mt-1" style={tone ? { color: tone } : undefined}>{value}</p>
    </div>
  );
}

function Th({ children }: { children?: React.ReactNode }) {
  return <th className="px-4 py-3 text-[11px] font-bold uppercase text-muted">{children}</th>;
}
