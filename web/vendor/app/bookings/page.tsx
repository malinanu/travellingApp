'use client';

// Bookings — every real booking made against this vendor's services, with
// status filters and manager+-gated actions (cancel, mark completed/no-show
// by hand for phone bookings or a code that was never scanned). Check-in
// itself happens via the existing Validate page (BOOK-XXXX codes), not here.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Ban, Bed, Calendar, CheckCircle2, Clock, Users, UserX, Utensils, LucideIcon } from 'lucide-react';
import { useState } from 'react';

import type { Booking, BookingServiceType, BookingStatus } from '@spotto/shared';
import Shell, { useMe } from '../../components/shell';
import { api } from '../../lib/api';

const TYPE_ICON: Record<BookingServiceType, LucideIcon> = { session: Clock, table: Utensils, stay: Bed };

const STATUS_TONE: Record<BookingStatus, { bg: string; fg: string }> = {
  confirmed: { bg: 'var(--accent-tint)', fg: 'var(--accent)' },
  completed: { bg: 'rgba(22,163,74,0.14)', fg: '#15803d' },
  cancelled: { bg: 'rgba(15,61,58,0.06)', fg: 'var(--muted)' },
  no_show: { bg: 'rgba(225,29,72,0.12)', fg: '#e11d48' },
};

type FilterId = 'upcoming' | 'confirmed' | 'completed' | 'cancelled' | 'no_show' | 'all';
const FILTERS: { id: FilterId; label: string }[] = [
  { id: 'upcoming', label: 'Upcoming' },
  { id: 'confirmed', label: 'Confirmed' },
  { id: 'completed', label: 'Completed' },
  { id: 'cancelled', label: 'Cancelled' },
  { id: 'no_show', label: 'No-show' },
  { id: 'all', label: 'All' },
];

function formatWhen(b: Booking) {
  const start = new Date(b.startAt);
  const end = new Date(b.endAt);
  if (b.type === 'stay') {
    return `${start.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })} → ${end.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })}`;
  }
  return start.toLocaleString(undefined, { month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit' });
}

export default function BookingsPage() {
  const qc = useQueryClient();
  const { data: me } = useMe();
  const isStaff = me?.vendorRole === 'staff';
  const { data: bookings = [] } = useQuery<Booking[]>({
    queryKey: ['vendor-bookings'],
    queryFn: () => api.get<Booking[]>('/vendor/bookings'),
  });
  const [filter, setFilter] = useState<FilterId>('upcoming');

  const setStatus = useMutation({
    mutationFn: ({ id, status }: { id: string; status: BookingStatus }) => api.patch(`/vendor/bookings/${id}/status`, { status }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['vendor-bookings'] }),
  });
  const cancel = useMutation({
    mutationFn: (id: string) => api.post(`/vendor/bookings/${id}/cancel`, {}),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['vendor-bookings'] }),
  });

  const now = Date.now();
  const visible = bookings
    .filter((b) => {
      if (filter === 'all') return true;
      if (filter === 'upcoming') return b.status === 'confirmed' && new Date(b.startAt).getTime() >= now;
      return b.status === filter;
    })
    .sort((a, b) => new Date(a.startAt).getTime() - new Date(b.startAt).getTime());

  return (
    <Shell>
      <div className="flex items-center justify-between mb-1">
        <h2 className="text-lg font-extrabold flex items-center gap-2">
          Bookings
          <span className="rounded-full bg-accent-tint text-accent text-[12.5px] font-extrabold w-6 h-6 flex items-center justify-center">
            {bookings.length}
          </span>
        </h2>
        <div className="flex gap-1.5 flex-wrap justify-end">
          {FILTERS.map((f) => (
            <button key={f.id} onClick={() => setFilter(f.id)}
              className={`rounded-full border px-3.5 py-1.5 text-[12.5px] font-extrabold transition-colors ${
                filter === f.id ? 'border-accent bg-accent text-white' : 'border-line bg-white hover:border-accent'
              }`}>
              {f.label}
            </button>
          ))}
        </div>
      </div>
      <p className="text-[13px] text-muted font-semibold mb-4">
        Check a traveler in by scanning their BOOK-XXXX code on the Validate page — actions here cover cancellations and manual status corrections.
      </p>

      <div className="space-y-2.5">
        {visible.length === 0 && (
          <p className="text-[13px] text-muted font-semibold py-8 text-center">No bookings in this view.</p>
        )}
        {visible.map((b) => {
          const Icon = TYPE_ICON[b.type] ?? Calendar;
          const tone = STATUS_TONE[b.status] ?? STATUS_TONE.confirmed;
          return (
            <div key={b.id} className="card p-4 flex items-center gap-3">
              <span className="w-10 h-10 rounded-xl bg-accent-tint text-accent flex items-center justify-center shrink-0">
                <Icon size={18} />
              </span>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 flex-wrap">
                  <p className="font-extrabold text-[14px] truncate">{b.serviceName || 'Service'}</p>
                  <span className="text-[10.5px] font-extrabold uppercase tracking-wide rounded-full px-2 py-0.5"
                    style={{ background: tone.bg, color: tone.fg }}>
                    {b.status.replace('_', ' ')}
                  </span>
                </div>
                <p className="text-[12.5px] text-muted font-semibold flex items-center gap-3 mt-0.5 flex-wrap">
                  <span className="flex items-center gap-1"><Calendar size={11} /> {formatWhen(b)}</span>
                  <span className="flex items-center gap-1"><Users size={11} /> {b.partySize}</span>
                  <span>{b.travellerName || 'Traveler'}{b.travellerCountry ? ` · ${b.travellerCountry}` : ''}</span>
                  <span className="font-mono">{b.code}</span>
                </p>
              </div>
              {!isStaff && b.status === 'confirmed' && (
                <div className="flex gap-1.5 shrink-0">
                  <button className="rounded-lg border border-line px-2.5 py-1.5 text-[12px] font-extrabold flex items-center gap-1 hover:border-accent hover:text-accent transition-colors"
                    onClick={() => setStatus.mutate({ id: b.id, status: 'completed' })} title="Mark completed">
                    <CheckCircle2 size={13} />
                  </button>
                  <button className="rounded-lg border border-line px-2.5 py-1.5 text-[12px] font-extrabold flex items-center gap-1 hover:border-accent hover:text-accent transition-colors"
                    onClick={() => setStatus.mutate({ id: b.id, status: 'no_show' })} title="Mark no-show">
                    <UserX size={13} />
                  </button>
                  <button className="rounded-lg border border-line px-2.5 py-1.5 text-[12px] font-extrabold text-coral flex items-center gap-1 hover:bg-[#fde4d8] transition-colors"
                    onClick={() => cancel.mutate(b.id)} title="Cancel">
                    <Ban size={13} />
                  </button>
                </div>
              )}
            </div>
          );
        })}
      </div>
    </Shell>
  );
}
