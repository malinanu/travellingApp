'use client';

// Services — manage what a bookable vendor (surf schools, restaurants,
// hotels, tour guides, ...) actually offers travelers a real booking
// against. Follows the Offers page's exact pattern: an always-visible
// creator form + a card grid of existing services below it. Fields switch
// by BookingServiceType (session/table/stay) — see @spotto/shared's
// BookableService.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Bed, CalendarCheck, Check, Clock, Pencil, Trash2, Users, Utensils, X, LucideIcon,
} from 'lucide-react';
import { useState } from 'react';

import type { BookableService, BookingServiceType, ServiceAvailabilityDoc } from '@spotto/shared';
import Shell, { useBusiness, useMe } from '../../components/shell';
import { api } from '../../lib/api';

const WEEKDAYS: { id: string; label: string }[] = [
  { id: 'monday', label: 'Mon' }, { id: 'tuesday', label: 'Tue' }, { id: 'wednesday', label: 'Wed' },
  { id: 'thursday', label: 'Thu' }, { id: 'friday', label: 'Fri' }, { id: 'saturday', label: 'Sat' },
  { id: 'sunday', label: 'Sun' },
];

const TYPE_INFO: Record<BookingServiceType, { label: string; icon: LucideIcon; hint: string }> = {
  session: { label: 'Session', icon: Clock, hint: 'A scheduled slot with a fixed duration — a lesson, class, or tour.' },
  table: { label: 'Table', icon: Utensils, hint: 'A short reservation for a party size — a table or seating.' },
  stay: { label: 'Stay', icon: Bed, hint: 'A date-range booking with a guest count — a room or unit.' },
};

type FormState = {
  id: string | null;
  name: string;
  description: string;
  type: BookingServiceType;
  durationMins: number;
  capacity: number;
  partySizeMin: number;
  partySizeMax: number;
  guestsPerRoom: number;
  priceLabel: string;
  activeDays: Set<string>;
  windowStart: string;
  windowEnd: string;
  active: boolean;
};

function blankForm(type: BookingServiceType): FormState {
  return {
    id: null, name: '', description: '', type, durationMins: 60, capacity: 1,
    partySizeMin: 1, partySizeMax: 4, guestsPerRoom: 2, priceLabel: '',
    activeDays: new Set(WEEKDAYS.map((d) => d.id)), windowStart: '09:00', windowEnd: '17:00', active: true,
  };
}

function formFromService(svc: BookableService): FormState {
  const days = new Set<string>();
  let windowStart = '09:00';
  let windowEnd = '17:00';
  if (svc.availability) {
    for (const [day, windows] of Object.entries(svc.availability)) {
      if (windows.length > 0) {
        days.add(day);
        windowStart = windows[0].start;
        windowEnd = windows[0].end;
      }
    }
  }
  return {
    id: svc.id, name: svc.name, description: svc.description, type: svc.type,
    durationMins: svc.durationMins || 60, capacity: svc.capacity || 1,
    partySizeMin: svc.partySizeMin || 1, partySizeMax: svc.partySizeMax || 4,
    guestsPerRoom: svc.guestsPerRoom || 2, priceLabel: svc.priceLabel,
    activeDays: svc.availability ? days : new Set(WEEKDAYS.map((d) => d.id)),
    windowStart, windowEnd, active: svc.active,
  };
}

export default function ServicesPage() {
  const qc = useQueryClient();
  const { data: me } = useMe();
  const isStaff = me?.vendorRole === 'staff';
  const { data: business } = useBusiness();
  const { data: services = [] } = useQuery<BookableService[]>({
    queryKey: ['vendor-services'],
    queryFn: () => api.get<BookableService[]>('/vendor/services'),
  });

  const [form, setForm] = useState<FormState>(() => blankForm('session'));
  const [error, setError] = useState<string | null>(null);

  const resetForm = () => { setForm(blankForm(form.type)); setError(null); };
  const startEdit = (svc: BookableService) => { setForm(formFromService(svc)); setError(null); window.scrollTo({ top: 0, behavior: 'smooth' }); };

  const save = useMutation({
    mutationFn: () => {
      const availability: ServiceAvailabilityDoc | undefined = form.type === 'stay' ? undefined : Object.fromEntries(
        [...form.activeDays].map((day) => [day, [{ start: form.windowStart, end: form.windowEnd }]]),
      );
      const body = {
        name: form.name, description: form.description, type: form.type,
        durationMins: form.type === 'stay' ? undefined : form.durationMins,
        capacity: form.capacity,
        partySizeMin: form.type === 'stay' ? undefined : form.partySizeMin,
        partySizeMax: form.type === 'stay' ? undefined : form.partySizeMax,
        guestsPerRoom: form.type === 'stay' ? form.guestsPerRoom : undefined,
        priceLabel: form.priceLabel, availability, active: form.active,
      };
      return form.id ? api.put(`/vendor/services/${form.id}`, body) : api.post('/vendor/services', body);
    },
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['vendor-services'] }); resetForm(); },
    onError: (e) => setError(e instanceof Error ? e.message : 'Failed to save'),
  });

  const toggleActive = useMutation({
    mutationFn: (svc: BookableService) => api.put(`/vendor/services/${svc.id}`, { ...svc, active: !svc.active }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['vendor-services'] }),
  });

  const remove = useMutation({
    mutationFn: (id: string) => api.del(`/vendor/services/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['vendor-services'] }),
  });

  const toggleDay = (day: string) => {
    setForm((f) => {
      const next = new Set(f.activeDays);
      if (next.has(day)) next.delete(day); else next.add(day);
      return { ...f, activeDays: next };
    });
  };

  return (
    <Shell>
      {!isStaff && (
        <div className="card p-6 mb-8">
          <div className="flex items-center justify-between mb-2">
            <label className="block text-[13px] font-extrabold">Booking type</label>
            {form.id && (
              <button type="button" onClick={resetForm} className="text-[12.5px] font-extrabold text-muted hover:text-ink">
                Cancel edit &amp; add a new service
              </button>
            )}
          </div>
          <div className="grid grid-cols-3 gap-2 mb-5">
            {(Object.keys(TYPE_INFO) as BookingServiceType[]).map((t) => {
              const info = TYPE_INFO[t];
              const Icon = info.icon;
              const on = form.type === t;
              return (
                <button key={t} type="button" disabled={!!form.id} onClick={() => setForm((f) => ({ ...f, type: t }))}
                  className={`rounded-xl border py-3 flex flex-col items-center gap-1.5 text-[12.5px] font-extrabold transition-colors disabled:opacity-60 ${
                    on ? 'border-accent bg-accent text-white' : 'border-line bg-white hover:border-accent'
                  }`}>
                  <Icon size={18} />
                  {info.label}
                </button>
              );
            })}
          </div>
          <p className="text-[12.5px] text-muted font-semibold mb-5">{TYPE_INFO[form.type].hint}</p>

          <div className="grid grid-cols-2 gap-3 mb-5">
            <div>
              <label className="block text-[13px] font-extrabold mb-1.5">Name</label>
              <input className="input" placeholder="e.g. Beginner surf lesson" value={form.name}
                onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))} />
            </div>
            <div>
              <label className="block text-[13px] font-extrabold mb-1.5">Price (display only)</label>
              <input className="input" placeholder="e.g. LKR 4,500pp" value={form.priceLabel}
                onChange={(e) => setForm((f) => ({ ...f, priceLabel: e.target.value }))} />
            </div>
          </div>

          <label className="block text-[13px] font-extrabold mb-1.5">Description</label>
          <textarea className="input mb-5" rows={2} value={form.description}
            onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))} />

          {form.type !== 'stay' ? (
            <div className="grid grid-cols-3 gap-3 mb-5">
              <div>
                <label className="block text-[13px] font-extrabold mb-1.5">Duration (mins)</label>
                <input className="input" type="number" min={5} value={form.durationMins}
                  onChange={(e) => setForm((f) => ({ ...f, durationMins: Number(e.target.value) }))} />
              </div>
              <div>
                <label className="block text-[13px] font-extrabold mb-1.5">Capacity (concurrent bookings)</label>
                <input className="input" type="number" min={1} value={form.capacity}
                  onChange={(e) => setForm((f) => ({ ...f, capacity: Number(e.target.value) }))} />
              </div>
              <div className="grid grid-cols-2 gap-1.5">
                <div>
                  <label className="block text-[13px] font-extrabold mb-1.5">Party min</label>
                  <input className="input" type="number" min={1} value={form.partySizeMin}
                    onChange={(e) => setForm((f) => ({ ...f, partySizeMin: Number(e.target.value) }))} />
                </div>
                <div>
                  <label className="block text-[13px] font-extrabold mb-1.5">Party max</label>
                  <input className="input" type="number" min={1} value={form.partySizeMax}
                    onChange={(e) => setForm((f) => ({ ...f, partySizeMax: Number(e.target.value) }))} />
                </div>
              </div>
            </div>
          ) : (
            <div className="grid grid-cols-2 gap-3 mb-5">
              <div>
                <label className="block text-[13px] font-extrabold mb-1.5">Rooms/units of this type</label>
                <input className="input" type="number" min={1} value={form.capacity}
                  onChange={(e) => setForm((f) => ({ ...f, capacity: Number(e.target.value) }))} />
              </div>
              <div>
                <label className="block text-[13px] font-extrabold mb-1.5">Guests per room</label>
                <input className="input" type="number" min={1} value={form.guestsPerRoom}
                  onChange={(e) => setForm((f) => ({ ...f, guestsPerRoom: Number(e.target.value) }))} />
              </div>
            </div>
          )}

          {form.type !== 'stay' && (
            <>
              <label className="block text-[13px] font-extrabold mb-2">Available days</label>
              <div className="flex flex-wrap gap-1.5 mb-3">
                {WEEKDAYS.map((d) => (
                  <button key={d.id} type="button" onClick={() => toggleDay(d.id)}
                    className={`rounded-full border px-3.5 py-1.5 text-[12.5px] font-extrabold transition-colors ${
                      form.activeDays.has(d.id) ? 'border-accent bg-accent text-white' : 'border-line bg-white hover:border-accent'
                    }`}>
                    {d.label}
                  </button>
                ))}
              </div>
              <div className="grid grid-cols-2 gap-3 mb-5">
                <div>
                  <label className="block text-[13px] font-extrabold mb-1.5">Window start</label>
                  <input className="input" type="time" value={form.windowStart}
                    onChange={(e) => setForm((f) => ({ ...f, windowStart: e.target.value }))} />
                </div>
                <div>
                  <label className="block text-[13px] font-extrabold mb-1.5">Window end</label>
                  <input className="input" type="time" value={form.windowEnd}
                    onChange={(e) => setForm((f) => ({ ...f, windowEnd: e.target.value }))} />
                </div>
              </div>
              <p className="text-[11.5px] text-muted font-semibold mb-5">
                Still gated by your Business Profile&rsquo;s Hours &amp; holiday closures — a day marked closed there is never bookable here either.
              </p>
            </>
          )}

          {error && <p className="text-[13px] font-bold text-coral mb-3">{error}</p>}

          <button className="btn-accent px-5 py-2.5 flex items-center gap-2" disabled={!form.name || save.isPending}
            onClick={() => save.mutate()}>
            <CalendarCheck size={16} /> {save.isPending ? 'Saving…' : form.id ? 'Save changes' : 'Add service'}
          </button>
        </div>
      )}

      <h2 className="text-lg font-extrabold mb-1">Your services</h2>
      <p className="text-[13px] text-muted font-semibold mb-4">
        {business?.name ?? 'Your business'} — what travelers can actually book.
      </p>

      <div className="grid md:grid-cols-2 xl:grid-cols-3 gap-4">
        {services.map((svc) => {
          const Icon = TYPE_INFO[svc.type]?.icon ?? CalendarCheck;
          return (
            <div key={svc.id} className={`card p-4 ${svc.active ? '' : 'opacity-60'}`}>
              <div className="flex items-start gap-3 mb-2.5">
                <span className="w-9 h-9 rounded-xl bg-accent-tint text-accent flex items-center justify-center shrink-0">
                  <Icon size={17} />
                </span>
                <div className="flex-1 min-w-0">
                  <p className="font-extrabold text-[14.5px] truncate">{svc.name}</p>
                  <p className="text-[12px] text-muted font-semibold truncate">{svc.priceLabel || 'No price set'}</p>
                </div>
                {!svc.active && (
                  <span className="text-[10.5px] font-extrabold uppercase rounded-full px-2 py-0.5 bg-sand text-muted shrink-0">Paused</span>
                )}
              </div>
              <p className="text-[12px] text-muted font-semibold flex items-center gap-1.5 mb-3">
                <Users size={12} />
                {svc.type === 'stay'
                  ? `${svc.capacity} room${svc.capacity === 1 ? '' : 's'} · up to ${svc.guestsPerRoom} guests each`
                  : `${svc.capacity} concurrent · ${svc.partySizeMin}-${svc.partySizeMax} guests · ${svc.durationMins}min`}
              </p>
              {!isStaff && (
                <div className="grid grid-cols-3 gap-1.5">
                  <button className="rounded-lg border border-line py-1.5 text-[12px] font-extrabold flex items-center justify-center gap-1 hover:border-accent hover:text-accent transition-colors" onClick={() => startEdit(svc)}>
                    <Pencil size={13} /> Edit
                  </button>
                  <button className="rounded-lg border border-line py-1.5 text-[12px] font-extrabold flex items-center justify-center gap-1 hover:border-accent hover:text-accent transition-colors" onClick={() => toggleActive.mutate(svc)}>
                    {svc.active ? <X size={13} /> : <Check size={13} />} {svc.active ? 'Pause' : 'Resume'}
                  </button>
                  <button className="rounded-lg border border-line py-1.5 text-[12px] font-extrabold text-coral flex items-center justify-center gap-1 hover:bg-[#fde4d8] transition-colors" onClick={() => remove.mutate(svc.id)}>
                    <Trash2 size={13} /> Delete
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
