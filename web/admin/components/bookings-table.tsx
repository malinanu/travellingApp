'use client';

// Bookings table and detail drawer: used by the Bookings page and the
// Vendors page's Bookings tab. Admins with bookings.manage can cancel a
// booking, which refunds a paid one in full.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { CalendarCheck, Search } from 'lucide-react';
import { useState } from 'react';

import {
  DataTable, DetailList, Drawer, EmptyState, Money, StatusPill, formatLocalDateTime, useConfirm,
  type Booking, type LedgerEntry, type PaymentRecord, type Refund,
} from '@spotto/shared';
import { hasAdminPerm, useMe } from './shell';
import { api } from '../lib/api';
import { Tabs, colomboDay } from '../lib/money-ui';

const STATUSES = [
  { id: '', label: 'All' },
  { id: 'confirmed', label: 'Confirmed' },
  { id: 'awaiting_payment', label: 'Awaiting payment' },
  { id: 'completed', label: 'Checked in' },
  { id: 'cancelled', label: 'Cancelled' },
  { id: 'no_show', label: 'No-show' },
  { id: 'expired', label: 'Expired' },
] as const;

export function BookingsTable({ vendorId }: { vendorId?: string }) {
  const [status, setStatus] = useState<(typeof STATUSES)[number]['id']>('');
  const [q, setQ] = useState('');
  const [from, setFrom] = useState(vendorId ? '' : colomboDay(-30));
  const [to, setTo] = useState(vendorId ? '' : colomboDay(60));
  const [selected, setSelected] = useState<string | null>(null);
  const params = new URLSearchParams();
  if (status) params.set('status', status);
  if (q.trim()) params.set('q', q.trim());
  if (from) params.set('from', from);
  if (to) params.set('to', to);
  const path = vendorId ? `/admin/vendors/${vendorId}/bookings` : '/admin/bookings';
  const { data = [], isLoading, isError } = useQuery<Booking[]>({
    queryKey: ['admin-bookings', path, params.toString()],
    queryFn: () => api.get<Booking[]>(`${path}?${params}`),
  });

  return (
    <>
      <div className="flex flex-wrap items-center gap-2 mb-3">
        <Tabs value={status} onChange={setStatus} options={[...STATUSES]} />
        <div className="ml-auto flex flex-wrap items-center gap-2">
          <label className="relative">
            <Search size={14} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-muted" />
            <input className="input !pl-8 !w-56" placeholder="Code, guest name or email" value={q} onChange={(e) => setQ(e.target.value)} />
          </label>
          <input type="date" className="input !w-auto" value={from} onChange={(e) => setFrom(e.target.value)} aria-label="From" />
          <input type="date" className="input !w-auto" value={to} onChange={(e) => setTo(e.target.value)} aria-label="To" />
        </div>
      </div>
      {isError && <p className="text-rose-600 text-[13px] font-semibold mb-2">Couldn’t load bookings. You may need the bookings.view permission.</p>}
      <div className="panel overflow-hidden">
        <DataTable
          loading={isLoading}
          rows={data}
          rowKey={(b) => b.id}
          onRowClick={(b) => setSelected(b.id)}
          initialSort={{ key: 'when', dir: 'desc' }}
          empty={<EmptyState icon={<CalendarCheck size={30} />} title="No bookings match" body="Try another status or date range." />}
          columns={[
            { key: 'when', header: 'When', sortValue: (b) => b.startAt, render: (b) => <span className="whitespace-nowrap font-semibold">{formatLocalDateTime(b.startAt)}</span> },
            ...(vendorId ? [] : [{ key: 'vendor', header: 'Vendor', sortValue: (b: Booking) => b.vendorName ?? '', render: (b: Booking) => b.vendorName }]),
            { key: 'service', header: 'Service', render: (b) => b.serviceName },
            { key: 'guest', header: 'Guest', render: (b) => b.travellerName || '—' },
            { key: 'party', header: 'Guests', align: 'right', render: (b) => b.partySize },
            { key: 'amount', header: 'Amount', align: 'right', sortValue: (b) => b.amountMinor ?? 0, render: (b) => (b.amountMinor ? <Money minor={b.amountMinor} /> : <span className="text-muted">Free</span>) },
            { key: 'commission', header: 'Spotto', align: 'right', sortValue: (b) => b.commissionMinor ?? 0, render: (b) => <Money minor={b.commissionMinor} /> },
            { key: 'status', header: 'Status', render: (b) => <StatusPill status={b.status} /> },
            { key: 'payment', header: 'Payment', render: (b) => <StatusPill status={b.paymentStatus ?? 'not_required'} /> },
          ]}
        />
      </div>
      {selected && <BookingDetail id={selected} onClose={() => setSelected(null)} />}
    </>
  );
}

function BookingDetail({ id, onClose }: { id: string; onClose: () => void }) {
  const qc = useQueryClient();
  const { data: me } = useMe();
  const [confirm, dialog] = useConfirm();
  const { data } = useQuery<{ booking: Booking; payments: PaymentRecord[]; refunds: Refund[]; ledger: LedgerEntry[] }>({
    queryKey: ['admin-booking', id],
    queryFn: () => api.get(`/admin/bookings/${id}`),
  });
  const cancel = useMutation({
    mutationFn: (reason: string) => api.post(`/admin/bookings/${id}/cancel`, { reason }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['admin-bookings'] }); qc.invalidateQueries({ queryKey: ['admin-booking', id] }); },
  });
  const b = data?.booking;
  const canCancel = b && hasAdminPerm(me, 'bookings.manage') && (b.status === 'confirmed' || b.status === 'awaiting_payment');

  return (
    <Drawer
      open
      onClose={onClose}
      width={560}
      title={b ? `${b.serviceName} · ${b.code}` : 'Booking'}
      subtitle={b ? <span className="inline-flex items-center gap-2">{b.vendorName} <StatusPill status={b.status} /></span> : undefined}
      footer={canCancel ? (
        <button className="btn btn-danger" disabled={cancel.isPending} onClick={async () => {
          const reason = await confirm({
            title: 'Cancel this booking?',
            body: (b.amountMinor ?? 0) > 0 ? 'The guest gets a full refund and an email. The vendor’s share is taken back.' : 'The guest gets an email.',
            reasonLabel: 'Reason (kept in the audit log)', confirmLabel: 'Cancel booking', tone: 'danger',
          });
          if (typeof reason === 'string') cancel.mutate(reason);
        }}>
          Cancel booking
        </button>
      ) : undefined}
    >
      {dialog}
      {!b ? <div className="h-24 rounded-lg bg-black/5 animate-pulse" /> : (
        <>
          <DetailList rows={[
            ['When', `${formatLocalDateTime(b.startAt)} → ${formatLocalDateTime(b.endAt)}`],
            ['Guest', `${b.travellerName || '—'}${b.travellerCountry ? ` · ${b.travellerCountry}` : ''} · party of ${b.partySize}`],
            ['Amount', b.amountMinor ? <Money key="a" minor={b.amountMinor} /> : 'Free'],
            ['Payment', <StatusPill key="p" status={b.paymentStatus ?? 'not_required'} />],
            ['Refunded', <Money key="r" minor={b.refundedMinor} />],
            ['Commission', b.commissionMode === 'per_guest_fee' ? <span key="c">Per-guest fee <Money minor={b.feePerGuestMinor} /> · charged <Money minor={b.commissionMinor} /></span>
              : b.commissionMode === 'percent' ? <span key="c">{(b.commissionRateBps ?? 0) / 100}% · <Money minor={b.commissionMinor} /></span> : 'None'],
            ['Vendor’s share', <Money key="n" minor={b.vendorNetMinor} />],
            ['Cancellation terms', `Free until ${b.freeCancelHours ?? 24}h before, then ${b.lateCancelRefundPct ?? 0}% back`],
            ...(b.cancelledAt ? [['Cancelled', `${formatLocalDateTime(b.cancelledAt)} by ${b.cancelledBy}`] as [string, string]] : []),
          ]} />
          <h4 className="mt-5 mb-1 text-[12px] font-bold uppercase text-muted">Payments</h4>
          <DataTable rows={data.payments} rowKey={(p) => p.id} empty={<p className="text-[13px] text-muted py-2">No payments.</p>}
            columns={[
              { key: 'when', header: 'Created', render: (p) => formatLocalDateTime(p.createdAt) },
              { key: 'gw', header: 'Gateway', render: (p) => `${p.gateway} ${p.gatewayPaymentId || ''}` },
              { key: 'amt', header: 'Amount', align: 'right', render: (p) => <Money minor={p.amountMinor} /> },
              { key: 'st', header: 'Status', render: (p) => <StatusPill status={p.status === 'created' ? 'pending' : p.status === 'succeeded' ? 'paid' : p.status} /> },
            ]} />
          <h4 className="mt-5 mb-1 text-[12px] font-bold uppercase text-muted">Refunds</h4>
          <DataTable rows={data.refunds} rowKey={(r) => r.id} empty={<p className="text-[13px] text-muted py-2">No refunds.</p>}
            columns={[
              { key: 'when', header: 'Created', render: (r) => formatLocalDateTime(r.createdAt) },
              { key: 'amt', header: 'Amount', align: 'right', render: (r) => <Money minor={r.amountMinor} /> },
              { key: 'st', header: 'Status', render: (r) => <StatusPill status={r.status} /> },
            ]} />
          <h4 className="mt-5 mb-1 text-[12px] font-bold uppercase text-muted">Ledger</h4>
          <DataTable rows={data.ledger} rowKey={(l) => l.id} empty={<p className="text-[13px] text-muted py-2">No ledger lines.</p>}
            columns={[
              { key: 'kind', header: 'Line', render: (l) => l.kind.replace(/_/g, ' ') },
              { key: 'amt', header: 'Amount', align: 'right', render: (l) => <Money minor={l.amountMinor} signed /> },
              { key: 'settled', header: '', render: (l) => (l.payoutId ? 'Paid out' : `Payable ${formatLocalDateTime(l.payableAt)}`) },
            ]} />
        </>
      )}
    </Drawer>
  );
}

