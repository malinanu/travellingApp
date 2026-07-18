'use client';

// Access grants — cross-vendor list of temporary Business Profile PIN
// grants an owner has issued to a manager, with admin extend/shorten/
// revoke actions. Port of the subscriptions page's flat-list +
// row-action-mutation convention (no per-vendor detail page exists yet).

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { MinusCircle, PlusCircle, XCircle } from 'lucide-react';
import { useState } from 'react';

import type { AdminVendorProfileGrant } from '@spotto/shared';
import Shell from '../../components/shell';
import { InlineErrorBanner } from '../../components/route-status';
import { api } from '../../lib/api';

const STATE_TABS = ['all', 'active', 'pending', 'expired', 'revoked'] as const;
const STATE_STYLE: Record<string, string> = {
  active: '#16a34a', pending: '#d97706', expired: '#6c8582', revoked: '#e11d48',
};

function expiryLabel(g: AdminVendorProfileGrant) {
  const iso = g.status === 'pending' ? g.pinExpiresAt : g.accessExpiresAt;
  if (!iso) return '—';
  return new Date(iso).toLocaleString();
}

export default function AccessGrantsPage() {
  const qc = useQueryClient();
  const [tab, setTab] = useState<(typeof STATE_TABS)[number]>('all');
  const { data: grants = [], isError } = useQuery<AdminVendorProfileGrant[]>({
    queryKey: ['admin-vendor-grants', tab],
    queryFn: () => api.get<AdminVendorProfileGrant[]>(`/admin/vendor-grants${tab === 'all' ? '' : `?status=${tab}`}`),
  });

  const invalidate = () => qc.invalidateQueries({ queryKey: ['admin-vendor-grants'] });
  const extend = useMutation({
    mutationFn: ({ id, minutes }: { id: string; minutes: number }) => api.post(`/admin/vendor-grants/${id}/extend`, { minutes }),
    onSuccess: invalidate,
  });
  const revoke = useMutation({
    mutationFn: (id: string) => api.post(`/admin/vendor-grants/${id}/revoke`, {}),
    onSuccess: invalidate,
  });

  const active = grants.filter((g) => g.status === 'active').length;
  const pending = grants.filter((g) => g.status === 'pending').length;

  return (
    <Shell>
      {isError && <InlineErrorBanner message="Couldn't load access grants — try refreshing." />}
      <div className="grid grid-cols-2 gap-4 mb-5">
        <Tile label="Active grants" value={`${active}`} tone="#16a34a" />
        <Tile label="Pending PINs" value={`${pending}`} tone="#d97706" />
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
              <Th>Vendor</Th><Th>Manager</Th><Th>Granted by</Th><Th>Status</Th><Th>Expires</Th><Th />
            </tr>
          </thead>
          <tbody>
            {grants.map((g) => {
              const actionable = g.status === 'active' || g.status === 'pending';
              return (
                <tr key={g.id} className="border-b border-line last:border-0 group">
                  <td className="px-4 py-3 font-bold">{g.vendorName}</td>
                  <td className="px-4 py-3 text-muted font-medium">{g.granteeName}</td>
                  <td className="px-4 py-3 text-muted font-medium">{g.ownerName}</td>
                  <td className="px-4 py-3">
                    <span className="font-bold text-[12px] uppercase" style={{ color: STATE_STYLE[g.status] }}>{g.status}</span>
                  </td>
                  <td className="px-4 py-3 text-muted font-medium">{expiryLabel(g)}</td>
                  <td className="px-4 py-3 text-right">
                    {actionable && (
                      <div className="flex items-center justify-end gap-1.5 opacity-0 group-hover:opacity-100 transition-opacity">
                        <button className="btn !py-1.5 flex items-center gap-1" title="Extend 15 minutes"
                          onClick={() => extend.mutate({ id: g.id, minutes: 15 })}>
                          <PlusCircle size={13} /> 15m
                        </button>
                        <button className="btn !py-1.5 flex items-center gap-1" title="Shorten 15 minutes"
                          onClick={() => extend.mutate({ id: g.id, minutes: -15 })}>
                          <MinusCircle size={13} /> 15m
                        </button>
                        <button className="btn !py-1.5 flex items-center gap-1" style={{ color: '#e11d48' }} title="Revoke"
                          onClick={() => revoke.mutate(g.id)}>
                          <XCircle size={13} /> Revoke
                        </button>
                      </div>
                    )}
                  </td>
                </tr>
              );
            })}
            {grants.length === 0 && (
              <tr><td colSpan={6} className="px-4 py-8 text-center text-muted font-semibold">No grants yet.</td></tr>
            )}
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
