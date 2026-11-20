'use client';

// Services & experiences: everything a bookable business or local host
// offers travellers — tables, sessions, stays, rentals, experiences and
// ticketed events. Prices are in LKR; travellers pay in the app and Spotto
// takes its commission before paying the rest out. Free table bookings
// cost the traveller nothing (Spotto charges a small fee per seated guest).

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { CalendarCheck, CalendarPlus, Pause, Pencil, Play, Plus, Trash2, Users, X } from 'lucide-react';
import { useMemo, useState } from 'react';

import {
  DataTable, Drawer, EmptyState, StatusPill, allowedBookingTypes, formatLocalDateTime, formatMinor, parseMajorToMinor, useConfirm,
  type BookableService, type BookingServiceType, type CapacityUnit, type PricingMode, type Scheduling,
  type ServiceAvailabilityDoc, type ServiceAvailabilityWindow, type ServiceOccurrence,
} from '@spotto/shared';
import Shell, { RoleLockedCard, useBusiness, useMe } from '../../components/shell';
import { api } from '../../lib/api';
import { BOOKING_TYPE_INFO, WEEKDAYS, colomboLocalToISO, colomboToday } from '../../lib/booking-ui';

type FormState = {
  id: string | null;
  type: BookingServiceType;
  name: string;
  description: string;
  pricingMode: PricingMode;
  price: string; // LKR as typed
  priceLabel: string;
  capacity: number;
  capacityUnit: CapacityUnit;
  durationMins: number;
  partySizeMin: number;
  partySizeMax: number;
  guestsPerRoom: number;
  scheduling: Scheduling;
  windows: Record<string, ServiceAvailabilityWindow[]>;
  blackoutDates: string[];
  meetingPoint: string;
  freeCancelHours: number;
  lateCancelRefundPct: number;
  active: boolean;
};

const DEFAULT_WINDOWS = () => Object.fromEntries(WEEKDAYS.map((d) => [d.id, [{ start: '09:00', end: '17:00' }]]));

function blankForm(type: BookingServiceType, isHost: boolean): FormState {
  const people = type === 'experience' || type === 'event';
  return {
    id: null, type, name: '', description: '',
    pricingMode: type === 'table' ? 'free' : type === 'stay' ? 'per_night' : people ? 'per_person' : 'per_booking',
    price: '', priceLabel: '', capacity: people ? 10 : 1, capacityUnit: people ? 'guests' : 'bookings',
    durationMins: type === 'table' ? 90 : people ? 180 : 60, partySizeMin: 1, partySizeMax: people ? 6 : 4, guestsPerRoom: 2,
    scheduling: type === 'event' || (people && isHost) ? 'occurrences' : 'weekly',
    windows: DEFAULT_WINDOWS(), blackoutDates: [], meetingPoint: '', freeCancelHours: 24, lateCancelRefundPct: 0, active: true,
  };
}

function formFromService(s: BookableService): FormState {
  return {
    id: s.id, type: s.type, name: s.name, description: s.description,
    pricingMode: s.pricingMode ?? 'free', price: s.priceMinor ? String(s.priceMinor / 100) : '', priceLabel: s.priceLabel ?? '',
    capacity: s.capacity || 1, capacityUnit: s.capacityUnit ?? 'bookings', durationMins: s.durationMins || 60,
    partySizeMin: s.partySizeMin || 1, partySizeMax: s.partySizeMax || 4, guestsPerRoom: s.guestsPerRoom || 2,
    scheduling: s.scheduling ?? 'weekly',
    windows: s.availability ? Object.fromEntries(WEEKDAYS.map((d) => [d.id, s.availability?.[d.id] ?? []])) : DEFAULT_WINDOWS(),
    blackoutDates: s.blackoutDates ?? [], meetingPoint: s.meetingPoint?.note ?? '',
    freeCancelHours: s.freeCancelHours ?? 24, lateCancelRefundPct: s.lateCancelRefundPct ?? 0, active: s.active,
  };
}

function priceSummary(s: BookableService): string {
  if (!s.priceMinor || s.pricingMode === 'free') return 'Free';
  const amount = formatMinor(s.priceMinor);
  return s.pricingMode === 'per_person' ? `${amount} / person` : s.pricingMode === 'per_night' ? `${amount} / night` : `${amount} / booking`;
}

export default function ServicesPage() {
  const qc = useQueryClient();
  const { data: me } = useMe();
  const isStaff = me?.vendorRole === 'staff';
  const { data: business } = useBusiness();
  const isHost = business?.kind === 'host';
  const types = allowedBookingTypes(business?.category) as BookingServiceType[];
  const { data: services = [], isLoading } = useQuery<BookableService[]>({
    queryKey: ['vendor-services'],
    queryFn: () => api.get<BookableService[]>('/vendor/services'),
    enabled: !isStaff,
  });
  const [form, setForm] = useState<FormState | null>(null);
  const [datesFor, setDatesFor] = useState<BookableService | null>(null);
  const [confirm, confirmDialog] = useConfirm();

  const toggleActive = useMutation({
    mutationFn: (s: BookableService) => api.put(`/vendor/services/${s.id}`, { ...s, active: !s.active }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['vendor-services'] }),
  });
  const remove = useMutation({
    mutationFn: (id: string) => api.del<{ status: string }>(`/vendor/services/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['vendor-services'] }),
  });

  if (isStaff) return <Shell><RoleLockedCard what="Services" /></Shell>;

  const onDelete = async (s: BookableService) => {
    const ok = await confirm({
      title: `Delete “${s.name}”?`,
      body: 'Travellers can no longer book it. If it already has bookings, it’s archived instead so their history stays; those bookings aren’t cancelled.',
      confirmLabel: 'Delete', tone: 'danger',
    });
    if (ok) remove.mutate(s.id);
  };

  return (
    <Shell>
      {confirmDialog}
      <div className="flex flex-wrap items-end justify-between gap-3 mb-5">
        <div>
          <h2 className="font-display font-extrabold text-[24px]">{isHost ? 'Your experiences' : 'Services & experiences'}</h2>
          <p className="text-muted font-semibold text-[13.5px] max-w-2xl">
            What travellers can book with you. They pay in the Spotto app; you get paid out weekly after each booking takes place.
            {types.includes('table') && ' Free table bookings stay free for guests: Spotto charges a small fee per guest you seat.'}
          </p>
        </div>
        {types.length > 0 && (
          <button className="btn btn-accent" onClick={() => setForm(blankForm(types[0], isHost))}>
            <Plus size={16} /> New {isHost ? 'experience' : 'service'}
          </button>
        )}
      </div>

      <div className="card">
        <DataTable
          loading={isLoading}
          rows={services}
          rowKey={(s) => s.id}
          onRowClick={(s) => setForm(formFromService(s))}
          empty={
            <EmptyState
              icon={<CalendarCheck size={34} />}
              title={types.length ? 'Nothing bookable yet' : 'Your category doesn’t take bookings'}
              body={types.length
                ? 'Add what travellers can book: a table, a class, a room, a tour or an event night.'
                : 'Change your business category in Profile if you’d like to take bookings.'}
              action={types.length ? (
                <button className="btn btn-accent" onClick={() => setForm(blankForm(types[0], isHost))}>
                  <Plus size={16} /> Add your first one
                </button>
              ) : undefined}
            />
          }
          columns={[
            {
              key: 'name', header: 'Name', sortValue: (s) => s.name.toLowerCase(),
              render: (s) => {
                const Icon = BOOKING_TYPE_INFO[s.type]?.icon ?? CalendarCheck;
                return (
                  <span className="flex items-center gap-2.5">
                    <span className="w-8 h-8 rounded-lg bg-accent-tint text-accent flex items-center justify-center shrink-0"><Icon size={16} /></span>
                    <span>
                      <span className="block font-extrabold">{s.name}</span>
                      <span className="block text-[12px] text-muted font-semibold">{BOOKING_TYPE_INFO[s.type]?.label ?? s.type}</span>
                    </span>
                  </span>
                );
              },
            },
            { key: 'price', header: 'Price', render: (s) => <span className="font-bold">{priceSummary(s)}</span> },
            {
              key: 'capacity', header: 'Capacity',
              render: (s) => <span className="inline-flex items-center gap-1 font-semibold"><Users size={13} /> {s.capacity} {s.capacityUnit === 'guests' ? 'guests' : s.type === 'stay' ? 'rooms' : 'at once'}</span>,
            },
            {
              key: 'when', header: 'When',
              render: (s) => <span className="font-semibold text-muted">{s.scheduling === 'occurrences' ? 'Set dates' : 'Weekly times'}</span>,
            },
            { key: 'status', header: 'Status', render: (s) => <StatusPill status={s.active ? 'confirmed' : 'cancelled'} className="!normal-case" /> },
            {
              key: 'actions', header: '', align: 'right',
              render: (s) => (
                <span className="inline-flex gap-1" onClick={(e) => e.stopPropagation()}>
                  {s.scheduling === 'occurrences' && (
                    <button className="btn btn-secondary !px-2.5 !py-1.5 text-[12.5px]" onClick={() => setDatesFor(s)} title="Dates">
                      <CalendarPlus size={14} /> Dates
                    </button>
                  )}
                  <button className="btn btn-secondary !px-2.5 !py-1.5" onClick={() => setForm(formFromService(s))} title="Edit"><Pencil size={14} /></button>
                  <button className="btn btn-secondary !px-2.5 !py-1.5" onClick={() => toggleActive.mutate(s)} title={s.active ? 'Pause bookings' : 'Take bookings again'}>
                    {s.active ? <Pause size={14} /> : <Play size={14} />}
                  </button>
                  <button className="btn btn-secondary !px-2.5 !py-1.5 text-rose-600" onClick={() => onDelete(s)} title="Delete"><Trash2 size={14} /></button>
                </span>
              ),
            },
          ]}
        />
      </div>

      {form && (
        <ServiceEditor
          form={form}
          types={types}
          isHost={isHost}
          onChange={setForm}
          onClose={() => setForm(null)}
          onSaved={(saved) => {
            qc.invalidateQueries({ queryKey: ['vendor-services'] });
            setForm(null);
            if (saved.scheduling === 'occurrences' && !form.id) setDatesFor(saved);
          }}
        />
      )}
      {datesFor && <OccurrencesDrawer service={datesFor} onClose={() => setDatesFor(null)} />}
    </Shell>
  );
}

function Field({ label, hint, children }: { label: string; hint?: string; children: React.ReactNode }) {
  return (
    <label className="block">
      <span className="block text-[12.5px] font-extrabold mb-1">{label}</span>
      {children}
      {hint && <span className="block text-[11.5px] text-muted font-semibold mt-1">{hint}</span>}
    </label>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="mb-6">
      <h4 className="text-[11.5px] font-extrabold uppercase tracking-wide text-muted mb-2.5">{title}</h4>
      <div className="space-y-3">{children}</div>
    </section>
  );
}

function ServiceEditor({ form, types, isHost, onChange, onClose, onSaved }: {
  form: FormState;
  types: BookingServiceType[];
  isHost: boolean;
  onChange: (f: FormState) => void;
  onClose: () => void;
  onSaved: (s: BookableService) => void;
}) {
  const [error, setError] = useState<string | null>(null);
  const [newBlackout, setNewBlackout] = useState('');
  const set = <K extends keyof FormState>(k: K, v: FormState[K]) => onChange({ ...form, [k]: v });
  const isStay = form.type === 'stay';
  const isEvent = form.type === 'event';
  const people = form.type === 'experience' || form.type === 'event';
  const priceMinor = form.pricingMode === 'free' ? 0 : parseMajorToMinor(form.price);

  const save = useMutation({
    mutationFn: () => {
      if (form.pricingMode !== 'free' && !priceMinor) throw new Error('Enter a price in LKR, or make it free.');
      const availability: ServiceAvailabilityDoc | undefined = isStay || form.scheduling === 'occurrences' ? undefined
        : Object.fromEntries(Object.entries(form.windows).filter(([, w]) => w.length > 0));
      const body: Partial<BookableService> = {
        name: form.name.trim(), description: form.description.trim(), type: form.type,
        pricingMode: form.pricingMode, priceMinor: priceMinor ?? 0, currency: 'LKR', priceLabel: form.priceLabel.trim(),
        capacity: form.capacity, capacityUnit: form.capacityUnit,
        durationMins: isStay ? undefined : form.durationMins,
        partySizeMin: isStay ? undefined : form.partySizeMin, partySizeMax: isStay ? undefined : form.partySizeMax,
        guestsPerRoom: isStay ? form.guestsPerRoom : undefined,
        scheduling: form.scheduling, availability, blackoutDates: form.blackoutDates,
        meetingPoint: form.meetingPoint.trim() ? { note: form.meetingPoint.trim() } : undefined,
        freeCancelHours: form.freeCancelHours, lateCancelRefundPct: form.lateCancelRefundPct, active: form.active,
      };
      return form.id ? api.put<BookableService>(`/vendor/services/${form.id}`, body) : api.post<BookableService>('/vendor/services', body);
    },
    onSuccess: onSaved,
    onError: (e) => setError(e instanceof Error ? e.message : 'Couldn’t save'),
  });

  const setWindow = (day: string, i: number, k: 'start' | 'end', v: string) =>
    set('windows', { ...form.windows, [day]: form.windows[day].map((w, j) => (j === i ? { ...w, [k]: v } : w)) });

  return (
    <Drawer
      open
      width={560}
      onClose={onClose}
      title={form.id ? `Edit ${form.name || 'service'}` : `New ${isHost ? 'experience' : 'service'}`}
      footer={<>
        {error && <span className="mr-auto text-[12.5px] font-bold text-rose-600 self-center">{error}</span>}
        <button className="btn btn-secondary" onClick={onClose}>Cancel</button>
        <button className="btn btn-accent" disabled={!form.name.trim() || save.isPending} onClick={() => { setError(null); save.mutate(); }}>
          {save.isPending ? 'Saving…' : form.id ? 'Save changes' : 'Create'}
        </button>
      </>}
    >
      <Section title="What is it">
        {!form.id && types.length > 1 && (
          <div className="grid grid-cols-2 gap-2">
            {types.map((t) => {
              const info = BOOKING_TYPE_INFO[t];
              const Icon = info.icon;
              const on = form.type === t;
              return (
                <button key={t} type="button" onClick={() => onChange({ ...blankForm(t, isHost), name: form.name, description: form.description })}
                  className={`rounded-xl border p-3 text-left transition-colors ${on ? 'border-accent bg-accent-tint' : 'border-line hover:border-accent'}`}>
                  <span className="flex items-center gap-2 font-extrabold text-[13.5px]"><Icon size={16} className="text-accent" /> {info.label}</span>
                  <span className="block text-[11.5px] text-muted font-semibold mt-1 leading-snug">{info.hint}</span>
                </button>
              );
            })}
          </div>
        )}
        <Field label="Name">
          <input className="input" value={form.name} onChange={(e) => set('name', e.target.value)}
            placeholder={people ? 'e.g. Sunrise whale watching' : isStay ? 'e.g. Deluxe sea-view room' : 'e.g. Dinner table'} />
        </Field>
        <Field label="Description">
          <textarea className="input" rows={3} value={form.description} onChange={(e) => set('description', e.target.value)}
            placeholder="What’s included, what to bring, who it suits." />
        </Field>
        {(people || isHost) && (
          <Field label="Meeting point" hint="Where guests find you. Shown after they book.">
            <input className="input" value={form.meetingPoint} onChange={(e) => set('meetingPoint', e.target.value)} placeholder="e.g. Mirissa harbour, blue gate" />
          </Field>
        )}
      </Section>

      <Section title="Price">
        <div className="grid grid-cols-2 gap-3">
          <Field label="Charge">
            <select className="input" value={form.pricingMode} onChange={(e) => set('pricingMode', e.target.value as PricingMode)}>
              <option value="free">Free for guests</option>
              <option value="per_person">Per person</option>
              <option value="per_booking">Per booking</option>
              {isStay && <option value="per_night">Per night</option>}
            </select>
          </Field>
          {form.pricingMode !== 'free' && (
            <Field label="Price (LKR)">
              <input className="input" inputMode="decimal" value={form.price} onChange={(e) => set('price', e.target.value)} placeholder="4,500" />
            </Field>
          )}
        </div>
        <Field label="Price note (optional)" hint="Shown next to the price, e.g. “includes breakfast”.">
          <input className="input" value={form.priceLabel} onChange={(e) => set('priceLabel', e.target.value)} />
        </Field>
        {form.pricingMode !== 'free' && (
          <div className="grid grid-cols-2 gap-3">
            <Field label="Free cancellation until" hint="Hours before the start.">
              <input className="input" type="number" min={0} max={1440} value={form.freeCancelHours} onChange={(e) => set('freeCancelHours', Number(e.target.value))} />
            </Field>
            <Field label="Refund after that (%)" hint="0 means no refund for late cancellations.">
              <input className="input" type="number" min={0} max={100} value={form.lateCancelRefundPct} onChange={(e) => set('lateCancelRefundPct', Number(e.target.value))} />
            </Field>
          </div>
        )}
        {form.pricingMode === 'free' && form.type === 'table' && (
          <p className="text-[12px] text-muted font-semibold">Guests book for free. Spotto charges you a small fee for each guest you seat (scan or mark as seated); no-shows cost nothing.</p>
        )}
      </Section>

      <Section title="Capacity">
        <div className="grid grid-cols-2 gap-3">
          <Field label={form.capacityUnit === 'guests' ? 'Guests at once' : isStay ? 'Rooms of this type' : 'Bookings at once'}>
            <input className="input" type="number" min={1} value={form.capacity} onChange={(e) => set('capacity', Number(e.target.value))} />
          </Field>
          {!isStay && (
            <Field label="Count capacity in">
              <select className="input" value={form.capacityUnit} onChange={(e) => set('capacityUnit', e.target.value as CapacityUnit)}>
                <option value="bookings">Bookings (tables, cars)</option>
                <option value="guests">Guests (tours, classes)</option>
              </select>
            </Field>
          )}
          {isStay ? (
            <Field label="Guests per room">
              <input className="input" type="number" min={1} value={form.guestsPerRoom} onChange={(e) => set('guestsPerRoom', Number(e.target.value))} />
            </Field>
          ) : (
            <>
              <Field label="Smallest group">
                <input className="input" type="number" min={1} value={form.partySizeMin} onChange={(e) => set('partySizeMin', Number(e.target.value))} />
              </Field>
              <Field label="Largest group">
                <input className="input" type="number" min={1} value={form.partySizeMax} onChange={(e) => set('partySizeMax', Number(e.target.value))} />
              </Field>
              <Field label="Length (minutes)">
                <input className="input" type="number" min={5} step={5} value={form.durationMins} onChange={(e) => set('durationMins', Number(e.target.value))} />
              </Field>
            </>
          )}
        </div>
      </Section>

      {!isStay && (
        <Section title="When">
          {!isEvent && (
            <div className="flex gap-2">
              {(['weekly', 'occurrences'] as Scheduling[]).map((sc) => (
                <button key={sc} type="button" onClick={() => set('scheduling', sc)}
                  className={`flex-1 rounded-xl border px-3 py-2 text-[13px] font-extrabold ${form.scheduling === sc ? 'border-accent bg-accent-tint text-accent' : 'border-line'}`}>
                  {sc === 'weekly' ? 'Same times every week' : 'Set dates'}
                </button>
              ))}
            </div>
          )}
          {form.scheduling === 'occurrences' ? (
            <p className="text-[12.5px] text-muted font-semibold">
              {form.id ? 'Add and manage dates with the “Dates” button in the list.' : 'After you create it, add the dates guests can book.'}
            </p>
          ) : (
            <>
              <div className="space-y-1.5">
                {WEEKDAYS.map((d) => {
                  const windows = form.windows[d.id] ?? [];
                  return (
                    <div key={d.id} className="flex items-start gap-2">
                      <label className="w-16 pt-2 flex items-center gap-1.5 text-[13px] font-extrabold">
                        <input type="checkbox" checked={windows.length > 0}
                          onChange={(e) => set('windows', { ...form.windows, [d.id]: e.target.checked ? [{ start: '09:00', end: '17:00' }] : [] })} />
                        {d.label}
                      </label>
                      <div className="flex-1 space-y-1">
                        {windows.length === 0 && <p className="pt-2 text-[12.5px] text-muted font-semibold">Closed</p>}
                        {windows.map((w, i) => (
                          <div key={i} className="flex items-center gap-1.5">
                            <input className="input !py-1.5 w-28" type="time" value={w.start} onChange={(e) => setWindow(d.id, i, 'start', e.target.value)} />
                            <span className="text-muted">–</span>
                            <input className="input !py-1.5 w-28" type="time" value={w.end} onChange={(e) => setWindow(d.id, i, 'end', e.target.value)} />
                            {i > 0 && (
                              <button type="button" className="text-muted hover:text-rose-600" title="Remove"
                                onClick={() => set('windows', { ...form.windows, [d.id]: windows.filter((_, j) => j !== i) })}><X size={15} /></button>
                            )}
                            {i === windows.length - 1 && (
                              <button type="button" className="text-[12px] font-extrabold text-accent"
                                onClick={() => set('windows', { ...form.windows, [d.id]: [...windows, { start: '18:00', end: '22:00' }] })}>+ time</button>
                            )}
                          </div>
                        ))}
                      </div>
                    </div>
                  );
                })}
              </div>
              <Field label="Closed on these dates" hint="Holidays or days off; bookings can’t be made for them.">
                <div className="flex gap-2">
                  <input className="input" type="date" min={colomboToday()} value={newBlackout} onChange={(e) => setNewBlackout(e.target.value)} />
                  <button type="button" className="btn btn-secondary" disabled={!newBlackout}
                    onClick={() => { if (!form.blackoutDates.includes(newBlackout)) set('blackoutDates', [...form.blackoutDates, newBlackout].sort()); setNewBlackout(''); }}>
                    Add
                  </button>
                </div>
              </Field>
              {form.blackoutDates.length > 0 && (
                <div className="flex flex-wrap gap-1.5">
                  {form.blackoutDates.map((d) => (
                    <span key={d} className="inline-flex items-center gap-1 rounded-full bg-sand px-2.5 py-1 text-[12px] font-extrabold">
                      {d}
                      <button type="button" onClick={() => set('blackoutDates', form.blackoutDates.filter((x) => x !== d))} aria-label={`Remove ${d}`}><X size={12} /></button>
                    </span>
                  ))}
                </div>
              )}
            </>
          )}
        </Section>
      )}
    </Drawer>
  );
}

/** Dated occurrences of a service: add dates, change capacity, call a date off. */
function OccurrencesDrawer({ service, onClose }: { service: BookableService; onClose: () => void }) {
  const qc = useQueryClient();
  const [confirm, confirmDialog] = useConfirm();
  const [start, setStart] = useState('');
  const [capacity, setCapacity] = useState('');
  const [error, setError] = useState<string | null>(null);
  const key = ['vendor-occurrences', service.id];
  const { data: occurrences = [], isLoading } = useQuery<ServiceOccurrence[]>({
    queryKey: key,
    queryFn: () => api.get<ServiceOccurrence[]>(`/vendor/services/${service.id}/occurrences`),
  });
  const upcoming = useMemo(() => occurrences.filter((o) => new Date(o.endAt) > new Date()), [occurrences]);

  const add = useMutation({
    mutationFn: () => api.post(`/vendor/services/${service.id}/occurrences`, {
      startAt: colomboLocalToISO(start), capacity: capacity ? Number(capacity) : undefined,
    }),
    onSuccess: () => { setStart(''); setCapacity(''); qc.invalidateQueries({ queryKey: key }); },
    onError: (e) => setError(e instanceof Error ? e.message : 'Couldn’t add the date'),
  });
  const cancel = useMutation({
    mutationFn: (id: string) => api.post<{ bookingsCancelled: number; bookingsFailed?: number }>(`/vendor/occurrences/${id}/cancel`),
    onSuccess: () => qc.invalidateQueries({ queryKey: key }),
  });

  const onCancel = async (o: ServiceOccurrence) => {
    const booked = (o.capacity ?? service.capacity) - (o.remaining ?? 0);
    const ok = await confirm({
      title: 'Call off this date?',
      body: booked > 0
        ? `Everyone booked on ${formatLocalDateTime(o.startAt)} is cancelled and gets a full refund, and we email them.`
        : `${formatLocalDateTime(o.startAt)} will no longer be bookable.`,
      confirmLabel: 'Call it off', tone: 'danger',
    });
    if (ok) cancel.mutate(o.id);
  };

  return (
    <Drawer open onClose={onClose} title={`Dates · ${service.name}`} subtitle="Times are Sri Lankan time." width={520}>
      {confirmDialog}
      <div className="rounded-xl border border-line p-3 mb-4">
        <div className="grid grid-cols-[1fr_110px_auto] gap-2 items-end">
          <Field label="Starts">
            <input className="input" type="datetime-local" value={start} onChange={(e) => setStart(e.target.value)} />
          </Field>
          <Field label="Places">
            <input className="input" type="number" min={1} placeholder={String(service.capacity)} value={capacity} onChange={(e) => setCapacity(e.target.value)} />
          </Field>
          <button className="btn btn-accent" disabled={!start || add.isPending} onClick={() => { setError(null); add.mutate(); }}>Add</button>
        </div>
        <p className="text-[11.5px] text-muted font-semibold mt-1.5">Lasts {service.durationMins ?? 120} minutes. Leave places empty to use the usual {service.capacity}.</p>
        {error && <p className="text-[12.5px] text-rose-600 font-bold mt-1">{error}</p>}
      </div>
      <DataTable
        loading={isLoading}
        rows={upcoming}
        rowKey={(o) => o.id}
        empty={<EmptyState title="No dates yet" body="Add the first date above." />}
        columns={[
          { key: 'when', header: 'When', render: (o) => <span className="font-bold">{formatLocalDateTime(o.startAt)}</span> },
          { key: 'left', header: 'Left', render: (o) => o.status === 'scheduled' ? `${o.remaining ?? 0} / ${o.capacity ?? service.capacity}` : '—' },
          { key: 'status', header: '', render: (o) => <StatusPill status={o.status} /> },
          {
            key: 'actions', header: '', align: 'right',
            render: (o) => o.status === 'scheduled' ? (
              <button className="btn btn-secondary !px-2.5 !py-1.5 text-[12.5px] text-rose-600" onClick={() => onCancel(o)}>Call off</button>
            ) : null,
          },
        ]}
      />
    </Drawer>
  );
}
