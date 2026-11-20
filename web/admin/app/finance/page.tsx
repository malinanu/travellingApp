'use client';

// Finance: what bookings earned Spotto in a period, the vendor ledger, and
// refunds that need a person (the gateway couldn't refund, or it failed).

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Download, Plus } from 'lucide-react';
import { useState } from 'react';

import {
  DataTable, Drawer, EmptyState, Money, StatusPill, downloadAuthed, formatLocalDateTime, formatMinor, parseMajorToMinor, useConfirm,
  type FinanceSummary, type LedgerEntry, type Refund,
} from '@spotto/shared';
import Shell, { hasAdminPerm, useMe } from '../../components/shell';
import { api } from '../../lib/api';
import { KpiTile, PageHeader, Tabs, colomboDay } from '../../lib/money-ui';

type Tab = 'summary' | 'refunds' | 'ledger';

export default function FinancePage() {
  const { data: me } = useMe();
  const canPay = hasAdminPerm(me, 'finance.payouts');
  const [tab, setTab] = useState<Tab>('summary');
  const [from, setFrom] = useState(colomboDay(-30));
  const [to, setTo] = useState(colomboDay());

  return (
    <Shell>
      <PageHeader
        title="Finance"
        sub="Spotto earns a commission on paid bookings and a per-guest fee on free tables. Amounts are in LKR; times are Sri Lankan time."
        actions={tab !== 'refunds' && (
          <>
            <input type="date" className="input !w-auto" value={from} max={to} onChange={(e) => e.target.value && setFrom(e.target.value)} />
            <input type="date" className="input !w-auto" value={to} min={from} onChange={(e) => e.target.value && setTo(e.target.value)} />
          </>
        )}
      />
      <div className="mb-4">
        <Tabs value={tab} onChange={setTab} options={[{ id: 'summary', label: 'Summary' }, { id: 'refunds', label: 'Refunds' }, { id: 'ledger', label: 'Ledger' }]} />
      </div>
      {tab === 'summary' && <Summary from={from} to={to} />}
      {tab === 'refunds' && <Refunds canAct={canPay} />}
      {tab === 'ledger' && <Ledger from={from} to={to} canAdjust={canPay} />}
    </Shell>
  );
}

function Summary({ from, to }: { from: string; to: string }) {
  const { data, isLoading } = useQuery<FinanceSummary>({
    queryKey: ['finance-summary', from, to],
    queryFn: () => api.get<FinanceSummary>(`/admin/finance/summary?from=${from}&to=${to}`),
  });
  const spotto = (data?.commissionMinor ?? 0) + (data?.guestFeesMinor ?? 0);
  return (
    <>
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-3 mb-4">
        <KpiTile label="Spotto earned" value={<Money minor={spotto} />} hint={`${formatMinor(data?.commissionMinor)} commission · ${formatMinor(data?.guestFeesMinor)} guest fees`} />
        <KpiTile label="Paid by travellers" value={<Money minor={data?.grossMinor} />} hint={`${data?.paidBookings ?? 0} paid of ${data?.bookings ?? 0} bookings`} />
        <KpiTile label="Refunded" value={<Money minor={data?.refundedMinor} />} hint={`${data?.openRefunds ?? 0} refund(s) need attention`} />
        <KpiTile label="Waiting to pay out" value={<Money minor={data?.pendingPayoutMinor} />} hint={`${data?.pendingPayouts ?? 0} draft or approved payout(s)`} />
      </div>
      <div className="panel overflow-hidden">
        <DataTable
          loading={isLoading}
          rows={data?.byCategory ?? []}
          rowKey={(c) => c.category}
          initialSort={{ key: 'commission', dir: 'desc' }}
          empty={<EmptyState title="No bookings in this period" />}
          columns={[
            { key: 'category', header: 'Category', sortValue: (c) => c.category, render: (c) => <span className="font-semibold">{c.category}</span> },
            { key: 'bookings', header: 'Bookings', align: 'right', sortValue: (c) => c.bookings, render: (c) => c.bookings },
            { key: 'gross', header: 'Paid by travellers', align: 'right', sortValue: (c) => c.grossMinor, render: (c) => <Money minor={c.grossMinor} /> },
            { key: 'commission', header: 'Spotto earned', align: 'right', sortValue: (c) => c.commissionMinor, render: (c) => <Money minor={c.commissionMinor} /> },
          ]}
        />
      </div>
    </>
  );
}

function Refunds({ canAct }: { canAct: boolean }) {
  const qc = useQueryClient();
  const [status, setStatus] = useState<'' | 'manual' | 'failed' | 'requested'>('manual');
  const [confirm, dialog] = useConfirm();
  const { data = [], isLoading } = useQuery<Refund[]>({
    queryKey: ['admin-refunds', status],
    queryFn: () => api.get<Refund[]>(`/admin/refunds${status ? `?status=${status}` : ''}`),
  });
  const done = useMutation({
    mutationFn: ({ id, reference }: { id: string; reference: string }) => api.post(`/admin/refunds/${id}/mark-done`, { reference }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin-refunds'] }),
  });
  const retry = useMutation({
    mutationFn: (id: string) => api.post(`/admin/refunds/${id}/retry`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin-refunds'] }),
  });
  return (
    <>
      {dialog}
      <div className="mb-3">
        <Tabs value={status} onChange={setStatus} options={[
          { id: 'manual', label: 'Needs manual refund' }, { id: 'failed', label: 'Failed' }, { id: 'requested', label: 'In progress' }, { id: '', label: 'All' },
        ]} />
      </div>
      <div className="panel overflow-hidden">
        <DataTable
          loading={isLoading}
          rows={data}
          rowKey={(r) => r.id}
          empty={<EmptyState title="Nothing to do" body="Refunds the gateway couldn’t make show up here." />}
          columns={[
            { key: 'when', header: 'Requested', sortValue: (r) => r.createdAt, render: (r) => formatLocalDateTime(r.createdAt) },
            { key: 'booking', header: 'Booking', render: (r) => <span><span className="font-mono text-[12px]">{r.bookingCode}</span> · {r.vendorName}</span> },
            { key: 'guest', header: 'Refund to', render: (r) => <span>{r.travellerName}<span className="block text-[11.5px] text-muted">{r.travellerEmail}</span></span> },
            { key: 'amount', header: 'Amount', align: 'right', render: (r) => <Money minor={r.amountMinor} /> },
            { key: 'status', header: 'Status', render: (r) => <StatusPill status={r.status} /> },
            {
              key: 'actions', header: '', align: 'right',
              render: (r) => canAct && ['manual', 'failed', 'requested'].includes(r.status) ? (
                <span className="inline-flex gap-1.5">
                  {r.status !== 'manual' && <button className="btn" disabled={retry.isPending} onClick={() => retry.mutate(r.id)}>Retry</button>}
                  <button className="btn btn-primary" onClick={async () => {
                    const reference = await confirm({
                      title: `Mark ${formatMinor(r.amountMinor)} as refunded?`,
                      body: 'Only after you have paid it back to the guest yourself.',
                      reasonLabel: 'Bank or gateway reference', confirmLabel: 'Mark refunded',
                    });
                    if (typeof reference === 'string') done.mutate({ id: r.id, reference });
                  }}>Mark refunded</button>
                </span>
              ) : null,
            },
          ]}
        />
      </div>
    </>
  );
}

function Ledger({ from, to, canAdjust }: { from: string; to: string; canAdjust: boolean }) {
  const qc = useQueryClient();
  const [vendorId, setVendorId] = useState('');
  const [adjusting, setAdjusting] = useState(false);
  const q = `from=${from}&to=${to}${vendorId ? `&vendorId=${encodeURIComponent(vendorId)}` : ''}`;
  const { data = [], isLoading } = useQuery<LedgerEntry[]>({
    queryKey: ['admin-ledger', q],
    queryFn: () => api.get<LedgerEntry[]>(`/admin/ledger?${q}`),
  });
  return (
    <>
      <div className="flex flex-wrap items-center gap-2 mb-3">
        <input className="input !w-64" placeholder="Filter by vendor id" value={vendorId} onChange={(e) => setVendorId(e.target.value.trim())} />
        <button className="btn ml-auto" onClick={() => downloadAuthed(api, `/admin/ledger?${q}&format=csv`, `spotto-ledger-${from}-${to}.csv`)}>
          <Download size={14} /> CSV
        </button>
        {canAdjust && <button className="btn btn-primary" onClick={() => setAdjusting(true)}><Plus size={14} /> Adjustment</button>}
      </div>
      <div className="panel overflow-hidden">
        <DataTable
          loading={isLoading}
          rows={data}
          rowKey={(e) => e.id}
          empty={<EmptyState title="No ledger lines in this period" />}
          columns={[
            { key: 'when', header: 'Date', sortValue: (e) => e.createdAt, render: (e) => formatLocalDateTime(e.createdAt) },
            { key: 'vendor', header: 'Vendor', sortValue: (e) => e.vendorName ?? '', render: (e) => e.vendorName },
            { key: 'kind', header: 'Line', render: (e) => <span className="capitalize">{e.kind.replace(/_/g, ' ')}</span> },
            { key: 'booking', header: 'Booking', render: (e) => <span className="font-mono text-[12px]">{e.bookingCode || '—'}</span> },
            { key: 'memo', header: 'Memo', render: (e) => <span className="text-muted">{e.memo}</span> },
            { key: 'amount', header: 'Amount', align: 'right', sortValue: (e) => e.amountMinor, render: (e) => <Money minor={e.amountMinor} signed /> },
            { key: 'settled', header: '', render: (e) => (e.payoutId ? <StatusPill status="paid" /> : <span className="text-[12px] text-muted">Unsettled</span>) },
          ]}
        />
      </div>
      {adjusting && <AdjustmentDrawer onClose={() => setAdjusting(false)} onDone={() => { setAdjusting(false); qc.invalidateQueries({ queryKey: ['admin-ledger'] }); }} />}
    </>
  );
}

function AdjustmentDrawer({ onClose, onDone }: { onClose: () => void; onDone: () => void }) {
  const [form, setForm] = useState({ vendorId: '', amount: '', direction: 'credit' as 'credit' | 'debit', memo: '' });
  const [error, setError] = useState<string | null>(null);
  const minor = parseMajorToMinor(form.amount);
  const save = useMutation({
    mutationFn: () => api.post('/admin/ledger/adjustments', {
      vendorId: form.vendorId.trim(), amountMinor: (minor ?? 0) * (form.direction === 'credit' ? 1 : -1), memo: form.memo,
    }),
    onSuccess: onDone,
    onError: (e) => setError(e instanceof Error ? e.message : 'Couldn’t save'),
  });
  return (
    <Drawer open onClose={onClose} title="Ledger adjustment" subtitle="A manual correction, kept in the audit log."
      footer={<>
        {error && <span className="mr-auto text-[12.5px] font-semibold text-rose-600 self-center">{error}</span>}
        <button className="btn" onClick={onClose}>Cancel</button>
        <button className="btn btn-primary" disabled={!form.vendorId || !minor || !form.memo || save.isPending} onClick={() => save.mutate()}>Add adjustment</button>
      </>}>
      <div className="space-y-3">
        <label className="block"><span className="text-[12px] font-bold">Vendor id</span>
          <input className="input mt-1" value={form.vendorId} onChange={(e) => setForm({ ...form, vendorId: e.target.value })} /></label>
        <div className="grid grid-cols-2 gap-2">
          <label className="block"><span className="text-[12px] font-bold">Direction</span>
            <select className="input mt-1" value={form.direction} onChange={(e) => setForm({ ...form, direction: e.target.value as 'credit' | 'debit' })}>
              <option value="credit">Spotto owes the vendor</option>
              <option value="debit">Vendor owes Spotto</option>
            </select></label>
          <label className="block"><span className="text-[12px] font-bold">Amount (LKR)</span>
            <input className="input mt-1" inputMode="decimal" value={form.amount} onChange={(e) => setForm({ ...form, amount: e.target.value })} /></label>
        </div>
        <label className="block"><span className="text-[12px] font-bold">Memo</span>
          <textarea className="input mt-1" rows={3} value={form.memo} onChange={(e) => setForm({ ...form, memo: e.target.value })} /></label>
      </div>
    </Drawer>
  );
}
