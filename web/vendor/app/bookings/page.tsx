'use client';

// Bookings: who is coming, when. A day or week calendar (Sri Lankan time)
// plus a filterable list, with each booking's details and actions in a side
// panel. Every action that changes a booking asks first; cancelling a paid
// booking refunds the guest in full.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { CalendarDays, ChevronLeft, ChevronRight, List, QrCode, Users } from 'lucide-react';
import Link from 'next/link';
import { useMemo, useState } from 'react';

import {
  DataTable, DetailList, Drawer, EmptyState, Money, StatusPill, formatLocalDateTime, formatMinor, useConfirm,
  type BookableService, type Booking,
} from '@spotto/shared';
import Shell, { useMe } from '../../components/shell';
import { api } from '../../lib/api';
import { BOOKING_TYPE_INFO, colomboToday } from '../../lib/booking-ui';

type View = 'day' | 'week' | 'list';
const LIST_FILTERS = [
  { id: 'upcoming', label: 'Upcoming' },
  { id: 'awaiting_payment', label: 'Awaiting payment' },
  { id: 'completed', label: 'Checked in' },
  { id: 'no_show', label: 'No-show' },
  { id: 'cancelled', label: 'Cancelled' },
  { id: 'all', label: 'All' },
] as const;

/** "YYYY-MM-DD" plus n days. */
function addDays(day: string, n: number): string {
  const d = new Date(`${day}T00:00:00Z`);
  d.setUTCDate(d.getUTCDate() + n);
  return d.toISOString().slice(0, 10);
}
/** Monday of the week containing day. */
function weekStart(day: string): string {
  const dow = new Date(`${day}T00:00:00Z`).getUTCDay(); // 0 = Sunday
  return addDays(day, -((dow + 6) % 7));
}
/** The Sri Lankan calendar day of an instant. */
function colomboDay(iso: string): string {
  return new Date(new Date(iso).getTime() + 330 * 60_000).toISOString().slice(0, 10);
}
function colomboTime(iso: string): string {
  return new Date(iso).toLocaleTimeString('en-GB', { timeZone: 'Asia/Colombo', hour: '2-digit', minute: '2-digit' });
}
function dayLabel(day: string, opts: Intl.DateTimeFormatOptions = { weekday: 'short', day: 'numeric', month: 'short' }) {
  return new Date(`${day}T12:00:00Z`).toLocaleDateString('en-GB', { ...opts, timeZone: 'UTC' });
}

export default function BookingsPage() {
  const { data: me } = useMe();
  const isStaff = me?.vendorRole === 'staff';
  const [view, setView] = useState<View>('week');
  const [anchor, setAnchor] = useState(colomboToday());
  const [filter, setFilter] = useState<(typeof LIST_FILTERS)[number]['id']>('upcoming');
  const [serviceId, setServiceId] = useState('');
  const [selected, setSelected] = useState<Booking | null>(null);

  const range = view === 'day' ? { from: anchor, to: anchor } : view === 'week' ? { from: weekStart(anchor), to: addDays(weekStart(anchor), 6) } : null;
  const query = range
    ? `from=${range.from}&to=${range.to}`
    : filter === 'upcoming' ? `from=${colomboToday()}&to=${colomboToday(365)}`
    : filter === 'all' ? '' : `status=${filter}`;
  const { data: bookings = [], isLoading, isError } = useQuery<Booking[]>({
    queryKey: ['vendor-bookings', query],
    queryFn: () => api.get<Booking[]>(`/vendor/bookings${query ? `?${query}` : ''}`),
    refetchInterval: 30_000,
  });
  const { data: services = [] } = useQuery<BookableService[]>({
    queryKey: ['vendor-services'],
    queryFn: () => api.get<BookableService[]>('/vendor/services'),
    enabled: !isStaff,
  });

  const visible = useMemo(() => {
    let out = serviceId ? bookings.filter((b) => b.serviceId === serviceId) : bookings;
    if (!range && filter === 'upcoming') out = out.filter((b) => b.status === 'confirmed' || b.status === 'awaiting_payment');
    // The calendar shows bookings that are on, or waiting for payment.
    if (range) out = out.filter((b) => b.status !== 'expired');
    return [...out].sort((a, b) => a.startAt.localeCompare(b.startAt));
  }, [bookings, serviceId, range, filter]);

  const step = view === 'day' ? 1 : 7;
  const days = range ? Array.from({ length: view === 'day' ? 1 : 7 }, (_, i) => addDays(range.from, i)) : [];
  const guests = visible.filter((b) => b.status === 'confirmed' || b.status === 'completed').reduce((n, b) => n + b.partySize, 0);

  return (
    <Shell>
      <div className="flex flex-wrap items-end justify-between gap-3 mb-4">
        <div>
          <h2 className="font-display font-extrabold text-[24px]">Bookings</h2>
          <p className="text-muted font-semibold text-[13.5px]">
            Check guests in by scanning their code on <Link href="/validate" className="text-accent font-extrabold">Validate</Link>. Times are Sri Lankan time.
          </p>
        </div>
        <div className="flex items-center gap-1 rounded-xl border border-line bg-white p-1">
          {([['day', 'Day', CalendarDays], ['week', 'Week', CalendarDays], ['list', 'List', List]] as const).map(([id, label, Icon]) => (
            <button key={id} onClick={() => setView(id)}
              className={`flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-[13px] font-extrabold ${view === id ? 'bg-accent text-white' : 'text-ink-soft hover:bg-sand'}`}>
              <Icon size={15} /> {label}
            </button>
          ))}
        </div>
      </div>

      <div className="card p-3 mb-4 flex flex-wrap items-center gap-2">
        {range ? (
          <>
            <button className="btn btn-secondary !px-2.5" onClick={() => setAnchor(addDays(anchor, -step))} aria-label="Previous"><ChevronLeft size={16} /></button>
            <button className="btn btn-secondary" onClick={() => setAnchor(colomboToday())}>Today</button>
            <button className="btn btn-secondary !px-2.5" onClick={() => setAnchor(addDays(anchor, step))} aria-label="Next"><ChevronRight size={16} /></button>
            <span className="font-extrabold text-[14px] ml-1">
              {view === 'day' ? dayLabel(anchor, { weekday: 'long', day: 'numeric', month: 'long' }) : `${dayLabel(range.from)} – ${dayLabel(range.to)}`}
            </span>
            <input type="date" className="input !py-1.5 !w-auto" value={anchor} onChange={(e) => e.target.value && setAnchor(e.target.value)} />
          </>
        ) : (
          <div className="flex flex-wrap gap-1.5">
            {LIST_FILTERS.map((f) => (
              <button key={f.id} onClick={() => setFilter(f.id)}
                className={`pill ${filter === f.id ? '!bg-accent !text-white' : ''}`}>{f.label}</button>
            ))}
          </div>
        )}
        {services.length > 1 && (
          <select className="input !py-1.5 !w-auto ml-auto" value={serviceId} onChange={(e) => setServiceId(e.target.value)}>
            <option value="">All services</option>
            {services.map((s) => <option key={s.id} value={s.id}>{s.name}</option>)}
          </select>
        )}
        <span className={`text-[12.5px] font-extrabold text-muted ${services.length > 1 ? '' : 'ml-auto'}`}>
          {visible.length} booking{visible.length === 1 ? '' : 's'} · {guests} guest{guests === 1 ? '' : 's'}
        </span>
      </div>

      {isError && <p className="text-rose-600 font-bold text-[13px] mb-3">Couldn’t load bookings. Refresh to try again.</p>}

      {range ? (
        <div className={`grid gap-2 ${view === 'week' ? 'grid-cols-1 md:grid-cols-7' : 'grid-cols-1'}`}>
          {days.map((day) => {
            const items = visible.filter((b) => colomboDay(b.startAt) === day);
            const isToday = day === colomboToday();
            return (
              <div key={day} className={`card p-2 min-h-[120px] ${isToday ? 'ring-2 ring-accent' : ''}`}>
                {view === 'week' && (
                  <p className={`px-1 pb-1.5 text-[12px] font-extrabold ${isToday ? 'text-accent' : 'text-muted'}`}>{dayLabel(day)}</p>
                )}
                {isLoading && <div className="h-10 rounded-lg bg-sand animate-pulse" />}
                {!isLoading && items.length === 0 && (
                  <p className="px-1 py-3 text-[12px] text-muted font-semibold">{view === 'day' ? 'No bookings this day.' : '—'}</p>
                )}
                <div className="space-y-1.5">
                  {items.map((b) => {
                    const Icon = BOOKING_TYPE_INFO[b.type]?.icon ?? CalendarDays;
                    return (
                      <button key={b.id} onClick={() => setSelected(b)}
                        className={`w-full text-left rounded-lg border px-2 py-1.5 transition-colors hover:border-accent ${
                          b.status === 'cancelled' || b.status === 'no_show' ? 'opacity-50 border-line' : 'border-line bg-white'
                        } ${b.status === 'awaiting_payment' ? 'border-dashed' : ''}`}>
                        <span className="flex items-center gap-1.5 text-[12px] font-extrabold">
                          <Icon size={12} className="text-accent shrink-0" />
                          {b.type === 'stay' ? 'Check-in' : colomboTime(b.startAt)}
                          <span className="ml-auto inline-flex items-center gap-0.5 text-muted"><Users size={11} />{b.partySize}</span>
                        </span>
                        <span className="block truncate text-[12.5px] font-bold">{b.travellerName || b.code}</span>
                        {view === 'day' && <span className="block truncate text-[11.5px] text-muted font-semibold">{b.serviceName}</span>}
                        <StatusPill status={b.status} className="mt-1 !text-[10px]" />
                      </button>
                    );
                  })}
                </div>
              </div>
            );
          })}
        </div>
      ) : (
        <div className="card">
          <DataTable
            loading={isLoading}
            rows={visible}
            rowKey={(b) => b.id}
            onRowClick={setSelected}
            initialSort={{ key: 'when', dir: filter === 'upcoming' ? 'asc' : 'desc' }}
            empty={<EmptyState icon={<CalendarDays size={32} />} title="No bookings here" body="When travellers book, they show up here and on the calendar." />}
            columns={[
              { key: 'when', header: 'When', sortValue: (b) => b.startAt, render: (b) => <span className="font-bold whitespace-nowrap">{formatLocalDateTime(b.startAt)}</span> },
              { key: 'guest', header: 'Guest', render: (b) => <span><span className="font-bold">{b.travellerName || '—'}</span>{b.travellerCountry ? <span className="text-muted"> · {b.travellerCountry}</span> : null}</span> },
              { key: 'service', header: 'Service', render: (b) => b.serviceName },
              { key: 'party', header: 'Guests', align: 'right', sortValue: (b) => b.partySize, render: (b) => b.partySize },
              { key: 'amount', header: 'Paid', align: 'right', render: (b) => (b.amountMinor ? <Money minor={b.amountMinor} /> : <span className="text-muted">Free</span>) },
              { key: 'status', header: 'Status', render: (b) => <StatusPill status={b.status} /> },
              { key: 'code', header: 'Code', render: (b) => <span className="font-mono text-[12px]">{b.code}</span> },
            ]}
          />
        </div>
      )}

      {selected && <BookingDrawer booking={selected} canManage={!isStaff} onClose={() => setSelected(null)} />}
    </Shell>
  );
}

function BookingDrawer({ booking: b, canManage, onClose }: { booking: Booking; canManage: boolean; onClose: () => void }) {
  const qc = useQueryClient();
  const [confirm, confirmDialog] = useConfirm();
  const [error, setError] = useState<string | null>(null);
  const done = () => { qc.invalidateQueries({ queryKey: ['vendor-bookings'] }); onClose(); };
  const fail = (e: unknown) => setError(e instanceof Error ? e.message : 'Something went wrong');
  const setStatus = useMutation({
    mutationFn: (status: 'completed' | 'no_show') => api.patch(`/vendor/bookings/${b.id}/status`, { status }),
    onSuccess: done, onError: fail,
  });
  const cancel = useMutation({
    mutationFn: () => api.post<{ refundMinor: number }>(`/vendor/bookings/${b.id}/cancel`, {}),
    onSuccess: done, onError: fail,
  });

  const paid = (b.amountMinor ?? 0) > 0 && (b.paymentStatus === 'paid' || b.paymentStatus === 'partially_refunded');
  const refundLeft = (b.amountMinor ?? 0) - (b.refundedMinor ?? 0);
  const isLive = b.status === 'confirmed';
  const started = new Date(b.startAt).getTime() <= Date.now();

  const ask = async (kind: 'seated' | 'no_show' | 'cancel') => {
    setError(null);
    if (kind === 'seated') {
      const fee = b.commissionMode === 'per_guest_fee' ? ` Spotto’s fee of ${formatMinor((b.feePerGuestMinor ?? 0) * b.partySize)} for ${b.partySize} guest(s) applies.` : '';
      if (await confirm({ title: 'Mark as checked in?', body: `Use this if you couldn’t scan their code.${fee}`, confirmLabel: 'Checked in' })) setStatus.mutate('completed');
    } else if (kind === 'no_show') {
      if (await confirm({ title: 'Mark as no-show?', body: 'The guest didn’t come. This can’t be undone.', confirmLabel: 'No-show', tone: 'danger' })) setStatus.mutate('no_show');
    } else {
      const body = paid
        ? `The guest gets a full refund of ${formatMinor(refundLeft)}, and we email them. Your share of this booking is taken back from your next payout.`
        : 'We email the guest to let them know.';
      if (await confirm({ title: 'Cancel this booking?', body, confirmLabel: 'Cancel booking', cancelLabel: 'Keep it', tone: 'danger' })) cancel.mutate();
    }
  };

  const busy = setStatus.isPending || cancel.isPending;
  return (
    <Drawer
      open
      onClose={onClose}
      title={b.travellerName || 'Booking'}
      subtitle={<span className="inline-flex items-center gap-2">{b.serviceName} <StatusPill status={b.status} /></span>}
      footer={canManage && (isLive || b.status === 'awaiting_payment') ? (
        <>
          {error && <span className="mr-auto text-[12.5px] font-bold text-rose-600 self-center">{error}</span>}
          <button className="btn btn-danger-outline" disabled={busy} onClick={() => ask('cancel')}>Cancel booking</button>
          {isLive && started && <button className="btn btn-secondary" disabled={busy} onClick={() => ask('no_show')}>No-show</button>}
          {isLive && <button className="btn btn-accent" disabled={busy} onClick={() => ask('seated')}>Mark checked in</button>}
        </>
      ) : undefined}
    >
      {confirmDialog}
      <DetailList rows={[
        ['When', b.type === 'stay' ? `${formatLocalDateTime(b.startAt)} → ${formatLocalDateTime(b.endAt)}` : formatLocalDateTime(b.startAt)],
        ['Guests', b.partySize],
        ['Code', <span key="c" className="font-mono">{b.code}</span>],
        ['Guest', [b.travellerName, b.travellerCountry].filter(Boolean).join(' · ') || '—'],
        ...(b.note ? [['Note', b.note] as [string, string]] : []),
        ['Payment', b.amountMinor ? <span key="p" className="inline-flex items-center gap-2"><Money minor={b.amountMinor} /> <StatusPill status={b.paymentStatus ?? 'pending'} /></span> : 'Free booking'],
        ...((b.refundedMinor ?? 0) > 0 ? [['Refunded', <Money key="r" minor={b.refundedMinor} />] as [string, React.ReactNode]] : []),
        ...(b.amountMinor ? [['Your share', <Money key="n" minor={b.vendorNetMinor} />] as [string, React.ReactNode], ['Spotto commission', <Money key="k" minor={b.commissionMinor} />] as [string, React.ReactNode]] : []),
        ...(b.commissionMode === 'per_guest_fee' ? [['Spotto fee', `${formatMinor(b.feePerGuestMinor)} per seated guest`] as [string, string]] : []),
        ...(b.checkedInAt ? [['Checked in', formatLocalDateTime(b.checkedInAt)] as [string, string]] : []),
        ...(b.cancelledAt ? [['Cancelled', `${formatLocalDateTime(b.cancelledAt)} by ${b.cancelledBy}`] as [string, string]] : []),
      ]} />
      {isLive && (
        <Link href={`/validate?code=${encodeURIComponent(b.code)}`} className="mt-4 inline-flex items-center gap-1.5 text-[13px] font-extrabold text-accent">
          <QrCode size={15} /> Check in on the Validate page
        </Link>
      )}
    </Drawer>
  );
}
