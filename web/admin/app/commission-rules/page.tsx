'use client';

// Commission rules: Spotto's cut. The most specific rule in effect wins —
// vendor, then category, then booking type, then the default. Percent
// rules apply to paid bookings; per-guest fees to free ones, charged when
// the guest is checked in. Rules are never edited: a new rate is a new
// rule from a date, so past bookings keep the rate they were made under.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Plus } from 'lucide-react';
import { useMemo, useState } from 'react';

import {
  DataTable, Drawer, EmptyState, VENDOR_CATEGORIES, formatLocalDate, formatMinor, parseMajorToMinor, useConfirm,
  type CommissionRule, type CommissionScope,
} from '@spotto/shared';
import Shell, { hasAdminPerm, useMe } from '../../components/shell';
import { api } from '../../lib/api';
import { PageHeader } from '../../lib/money-ui';

const SCOPE_LABEL: Record<CommissionScope, string> = { default: 'Default', booking_type: 'Booking type', category: 'Category', vendor: 'Vendor' };
const BOOKING_TYPES = ['table', 'session', 'experience', 'event', 'stay', 'rental'];

export default function CommissionRulesPage() {
  const qc = useQueryClient();
  const { data: me } = useMe();
  const canEdit = hasAdminPerm(me, 'finance.commission');
  const [adding, setAdding] = useState(false);
  const [confirm, dialog] = useConfirm();
  const { data = [], isLoading } = useQuery<CommissionRule[]>({
    queryKey: ['commission-rules'],
    queryFn: () => api.get<CommissionRule[]>('/admin/commission-rules'),
  });
  const now = Date.now();
  // The rule in effect for each scope+key+mode is the newest one already active.
  const inEffect = useMemo(() => {
    const seen = new Set<string>();
    const ids = new Set<string>();
    for (const r of [...data].sort((a, b) => b.activeFrom.localeCompare(a.activeFrom))) {
      const k = `${r.scope}|${r.scopeKey}|${r.mode}`;
      if (new Date(r.activeFrom).getTime() <= now && !seen.has(k)) { seen.add(k); ids.add(r.id); }
    }
    return ids;
  }, [data, now]);
  const remove = useMutation({
    mutationFn: (id: string) => api.del(`/admin/commission-rules/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['commission-rules'] }),
  });

  return (
    <Shell>
      {dialog}
      <PageHeader
        title="Commission rules"
        sub="The most specific rule wins: vendor, then category, then booking type, then default. Percent applies to paid bookings; per-guest fees to free bookings, charged at check-in. To change a rate, add a new rule; past bookings keep their rate."
        actions={canEdit && <button className="btn btn-primary" onClick={() => setAdding(true)}><Plus size={14} /> Add rule</button>}
      />
      <div className="panel overflow-hidden">
        <DataTable
          loading={isLoading}
          rows={data}
          rowKey={(r) => r.id}
          empty={<EmptyState title="No rules" body="Without a rule Spotto earns nothing on bookings." />}
          columns={[
            { key: 'scope', header: 'Applies to', render: (r) => <span><span className="font-semibold">{SCOPE_LABEL[r.scope]}</span>{r.scopeKey ? <span className="text-muted"> · {r.scopeKey}</span> : null}</span> },
            { key: 'rate', header: 'Spotto takes', render: (r) => (r.mode === 'percent' ? `${r.rateBps / 100}% of paid bookings` : `${formatMinor(r.feePerGuestMinor)} per seated guest (free bookings)`) },
            { key: 'from', header: 'From', sortValue: (r) => r.activeFrom, render: (r) => formatLocalDate(r.activeFrom) },
            {
              key: 'state', header: '',
              render: (r) => new Date(r.activeFrom).getTime() > now
                ? <span className="text-[12px] font-semibold text-amber-700">Starts later</span>
                : inEffect.has(r.id) ? <span className="text-[12px] font-semibold text-emerald-600">In effect</span>
                : <span className="text-[12px] text-muted">Superseded</span>,
            },
            { key: 'note', header: 'Note', render: (r) => <span className="text-muted">{r.note}</span> },
            {
              key: 'actions', header: '', align: 'right',
              render: (r) => canEdit && new Date(r.activeFrom).getTime() > now ? (
                <button className="btn" onClick={async () => {
                  if (await confirm({ title: 'Delete this scheduled rule?', confirmLabel: 'Delete', tone: 'danger' })) remove.mutate(r.id);
                }}>Delete</button>
              ) : null,
            },
          ]}
        />
      </div>
      {adding && <AddRule onClose={() => setAdding(false)} onDone={() => { setAdding(false); qc.invalidateQueries({ queryKey: ['commission-rules'] }); }} />}
    </Shell>
  );
}

function AddRule({ onClose, onDone }: { onClose: () => void; onDone: () => void }) {
  const [form, setForm] = useState({ scope: 'category' as CommissionScope, scopeKey: '', mode: 'percent' as 'percent' | 'per_guest_fee', rate: '10', fee: '100', activeFrom: '', note: '' });
  const [error, setError] = useState<string | null>(null);
  const save = useMutation({
    mutationFn: () => {
      const fee = parseMajorToMinor(form.fee);
      return api.post('/admin/commission-rules', {
        scope: form.scope, scopeKey: form.scope === 'default' ? '' : form.scopeKey, mode: form.mode,
        rateBps: form.mode === 'percent' ? Math.round(Number(form.rate) * 100) : 0,
        feePerGuestMinor: form.mode === 'per_guest_fee' ? fee ?? 0 : 0,
        activeFrom: form.activeFrom || undefined, note: form.note,
      });
    },
    onSuccess: onDone,
    onError: (e) => setError(e instanceof Error ? e.message : 'Couldn’t save'),
  });
  const set = (k: keyof typeof form, v: string) => setForm({ ...form, [k]: v });
  return (
    <Drawer open onClose={onClose} title="Add commission rule"
      footer={<>
        {error && <span className="mr-auto text-[12.5px] font-semibold text-rose-600 self-center">{error}</span>}
        <button className="btn" onClick={onClose}>Cancel</button>
        <button className="btn btn-primary" disabled={save.isPending || (form.scope !== 'default' && !form.scopeKey)} onClick={() => { setError(null); save.mutate(); }}>Add rule</button>
      </>}>
      <div className="space-y-3">
        <label className="block"><span className="text-[12px] font-bold">Applies to</span>
          <select className="input mt-1" value={form.scope} onChange={(e) => setForm({ ...form, scope: e.target.value as CommissionScope, scopeKey: '' })}>
            {(Object.keys(SCOPE_LABEL) as CommissionScope[]).map((s) => <option key={s} value={s}>{SCOPE_LABEL[s]}</option>)}
          </select></label>
        {form.scope === 'category' && (
          <label className="block"><span className="text-[12px] font-bold">Category</span>
            <select className="input mt-1" value={form.scopeKey} onChange={(e) => set('scopeKey', e.target.value)}>
              <option value="">Choose…</option>
              {VENDOR_CATEGORIES.filter((c) => c.bookingType).map((c) => <option key={c.id} value={c.id}>{c.label}</option>)}
            </select></label>
        )}
        {form.scope === 'booking_type' && (
          <label className="block"><span className="text-[12px] font-bold">Booking type</span>
            <select className="input mt-1" value={form.scopeKey} onChange={(e) => set('scopeKey', e.target.value)}>
              <option value="">Choose…</option>
              {BOOKING_TYPES.map((t) => <option key={t} value={t}>{t}</option>)}
            </select></label>
        )}
        {form.scope === 'vendor' && (
          <label className="block"><span className="text-[12px] font-bold">Vendor id</span>
            <input className="input mt-1" value={form.scopeKey} onChange={(e) => set('scopeKey', e.target.value.trim())} placeholder="ven_…" /></label>
        )}
        <label className="block"><span className="text-[12px] font-bold">Kind</span>
          <select className="input mt-1" value={form.mode} onChange={(e) => set('mode', e.target.value)}>
            <option value="percent">Percent of paid bookings</option>
            <option value="per_guest_fee">Fee per seated guest (free bookings)</option>
          </select></label>
        {form.mode === 'percent' ? (
          <label className="block"><span className="text-[12px] font-bold">Rate (%)</span>
            <input className="input mt-1" type="number" min={0} max={50} step={0.25} value={form.rate} onChange={(e) => set('rate', e.target.value)} /></label>
        ) : (
          <label className="block"><span className="text-[12px] font-bold">Fee per guest (LKR)</span>
            <input className="input mt-1" inputMode="decimal" value={form.fee} onChange={(e) => set('fee', e.target.value)} /></label>
        )}
        <label className="block"><span className="text-[12px] font-bold">Starts on (empty = now)</span>
          <input className="input mt-1" type="date" value={form.activeFrom} onChange={(e) => set('activeFrom', e.target.value)} /></label>
        <label className="block"><span className="text-[12px] font-bold">Note</span>
          <input className="input mt-1" value={form.note} onChange={(e) => set('note', e.target.value)} placeholder="e.g. launch rate for Colombo cafés" /></label>
      </div>
    </Drawer>
  );
}
