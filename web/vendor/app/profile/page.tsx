'use client';

// Business profile — the public listing travelers see on the map. Port of
// the "Business profile" mockup: business-type picker (drives the sidebar's
// Menu tab, see isFoodCategory in components/shell.tsx), a decorative
// coming-soon photos grid (no upload infra yet), connectivity/vibe, best
// time to visit, and a combined about + amenities card with custom-amenity
// add. Category is a top-level vendors column so it saves through its own
// PATCH endpoint immediately on click; everything else batches into the
// profile document and saves via the header's "Save changes" button.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Check, Clock, Facebook, FileText, Globe, Instagram,
  KeyRound, Laptop, Lock, MapPin, Monitor, Phone, Plus, Save, ShieldCheck, Store, Sun,
  Sunrise, Sunset, UtensilsCrossed, Users, Volume2, Waves, Wifi, X, LucideIcon,
} from 'lucide-react';
import { useEffect, useState } from 'react';

import type { DayHours, UpcomingHoliday, Vendor, VendorProfile, VendorProfileGrant, Weekday } from '@spotto/shared';
import { useAutoSave, WEEKDAYS } from '@spotto/shared';
import AddressSearchField from '../../components/address-search-field';
import LocationPicker from '../../components/location-picker-auto';
import Shell, { isFoodCategory, useBusiness, useMe } from '../../components/shell';
import VendorPhotos from '../../components/vendor-photos';
import VendorDocs from '../../components/vendor-docs';
import VendorBRCertificate from '../../components/vendor-br-certificate';
import { api } from '../../lib/api';
import { VENDOR_SEGMENTS, categoriesBySegment, categoryById, isSelectedCategory, categoryIconComponent } from '../../lib/business-categories';

const VIBE_DIMENSIONS: { id: 'productive' | 'loud' | 'crowded'; label: string; icon: LucideIcon }[] = [
  { id: 'productive', label: 'Work-friendly', icon: Laptop },
  { id: 'loud', label: 'Lively', icon: Volume2 },
  { id: 'crowded', label: 'Crowded', icon: Users },
];

const TIME_SLOTS = [
  { id: 'morning', label: 'Morning', icon: Sunrise },
  { id: 'afternoon', label: 'Afternoon', icon: Sun },
  { id: 'evening', label: 'Evening', icon: Sunset },
];

const ALL_AMENITIES = [
  'Power outlets', 'Standing desks', 'Air-con loft', 'Specialty coffee', 'Quiet zone',
  'Sea-view deck', 'Board storage', 'Outdoor shower', 'Live music', 'Parking',
];

function CardHeader({ icon: Icon, title, sub }: { icon: LucideIcon; title: string; sub: string }) {
  return (
    <div className="flex items-center gap-2.5 mb-4">
      <span className="w-9 h-9 rounded-xl bg-accent-tint text-accent flex items-center justify-center shrink-0">
        <Icon size={18} />
      </span>
      <div>
        <h2 className="font-extrabold text-[15.5px]">{title}</h2>
        <p className="text-[12.5px] text-muted font-semibold">{sub}</p>
      </div>
    </div>
  );
}

// useCountdown returns a "12m" / "45s" style label for a future ISO
// timestamp, ticking every second, or null once it's passed.
function useCountdown(target?: string) {
  const [label, setLabel] = useState<string | null>(null);
  useEffect(() => {
    if (!target) { setLabel(null); return; }
    const tick = () => {
      const ms = new Date(target).getTime() - Date.now();
      if (ms <= 0) { setLabel(null); return; }
      const mins = Math.floor(ms / 60000);
      const secs = Math.floor((ms % 60000) / 1000);
      setLabel(mins > 0 ? `${mins}m ${secs}s` : `${secs}s`);
    };
    tick();
    const id = setInterval(tick, 1000);
    return () => clearInterval(id);
  }, [target]);
  return label;
}

// UpcomingHolidaysCard surfaces public holidays within the notice window
// that don't yet have an open/closed decision — a lighter action than the
// rest of Business Profile (see requireBusinessProfileAccess's bypass in
// vendor.go), so it renders for owner and manager alike regardless of
// whether a temporary edit grant is active.
function UpcomingHolidaysCard() {
  const qc = useQueryClient();
  const { data: upcoming = [] } = useQuery<UpcomingHoliday[]>({
    queryKey: ['upcoming-holidays'],
    queryFn: () => api.get<UpcomingHoliday[]>('/vendor/holidays/upcoming'),
  });
  const decide = useMutation({
    mutationFn: (vars: { date: string; name: string; closed: boolean }) => api.post('/vendor/holidays/decision', vars),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['upcoming-holidays'] }),
  });

  if (upcoming.length === 0) return null;

  return (
    <div className="space-y-2 mb-5">
      {upcoming.map((h) => {
        const days = Math.ceil((new Date(h.date).getTime() - Date.now()) / 86_400_000);
        const dateLabel = new Date(h.date).toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
        return (
          <div key={h.date} className="flex items-center justify-between gap-3 rounded-xl px-4 py-3"
            style={{ background: 'rgba(246,184,30,0.12)', border: '1px solid rgba(246,184,30,0.4)' }}>
            <div className="flex items-center gap-2.5 min-w-0">
              <span className="w-9 h-9 rounded-xl flex items-center justify-center shrink-0" style={{ background: 'rgba(246,184,30,0.2)', color: '#a97b00' }}>
                <Clock size={17} />
              </span>
              <p className="text-[13px] font-semibold min-w-0">
                <span className="font-extrabold">{h.name}</span> is {days <= 0 ? 'today' : `in ${days} day${days === 1 ? '' : 's'}`} ({dateLabel}) — will you be open?
              </p>
            </div>
            <div className="flex gap-1.5 shrink-0">
              <button className="rounded-lg border border-line bg-white px-3 py-1.5 font-extrabold text-[12.5px] hover:border-accent hover:text-accent transition-colors"
                disabled={decide.isPending} onClick={() => decide.mutate({ date: h.date, name: h.name, closed: false })}>
                Open
              </button>
              <button className="rounded-lg border border-line bg-white px-3 py-1.5 font-extrabold text-[12.5px] hover:border-coral hover:text-coral transition-colors"
                disabled={decide.isPending} onClick={() => decide.mutate({ date: h.date, name: h.name, closed: true })}>
                Closed
              </button>
            </div>
          </div>
        );
      })}
    </div>
  );
}

function LockedProfileCard({ grantId }: { grantId?: string }) {
  const qc = useQueryClient();
  const [pin, setPin] = useState('');
  const redeem = useMutation({
    mutationFn: () => api.post(`/vendor/profile-grants/${grantId}/redeem`, { pin }),
    onSuccess: () => {
      setPin('');
      qc.invalidateQueries({ queryKey: ['profile-grant-mine'] });
    },
  });

  return (
    <div className="card p-8 max-w-md mx-auto text-center">
      <span className="mx-auto w-14 h-14 rounded-2xl bg-accent-tint text-accent flex items-center justify-center mb-4">
        <Lock size={26} />
      </span>
      <h2 className="font-extrabold text-[16px] font-display mb-1.5">Business Profile is locked</h2>
      {grantId ? (
        <>
          <p className="text-muted font-semibold text-[13.5px] mb-4">
            An owner sent you a temporary access PIN. Enter it below to unlock editing.
          </p>
          <div className="flex items-center gap-2">
            <input className="input text-center tracking-[0.3em] font-extrabold" maxLength={6}
              placeholder="000000" value={pin} onChange={(e) => setPin(e.target.value.replace(/\D/g, ''))} />
            <button className="btn-accent px-4 py-2.5 shrink-0" disabled={pin.length !== 6 || redeem.isPending}
              onClick={() => redeem.mutate()}>
              <KeyRound size={16} />
            </button>
          </div>
          {redeem.isError && (
            <p className="text-[12.5px] font-bold mt-2" style={{ color: 'var(--coral)' }}>Invalid or expired PIN.</p>
          )}
        </>
      ) : (
        <p className="text-muted font-semibold text-[13.5px]">
          Only managers can be granted temporary edit access — ask an owner to send you a PIN from the Account page.
        </p>
      )}
    </div>
  );
}

export default function ProfilePage() {
  const qc = useQueryClient();
  const { data: me } = useMe();
  const { data: business } = useBusiness();
  const { data: grant } = useQuery<VendorProfileGrant | null>({
    queryKey: ['profile-grant-mine'],
    queryFn: () => api.get<VendorProfileGrant | null>('/vendor/profile-grants/mine'),
    enabled: me?.vendorRole === 'manager',
    refetchInterval: 15000,
  });
  const [p, setP] = useState<VendorProfile>({});
  const [dirty, setDirty] = useState(false);
  const [justSaved, setJustSaved] = useState(false);
  const [newAmenity, setNewAmenity] = useState('');
  const [addingAmenity, setAddingAmenity] = useState(false);
  const [activeSegment, setActiveSegment] = useState<string | null>(null);

  const [identity, setIdentity] = useState({ name: '', town: '', lat: 0, lng: 0 });
  const [identityDirty, setIdentityDirty] = useState(false);
  const [identitySaved, setIdentitySaved] = useState(false);
  // Bumped whenever the Address search resolves a new location, so the map
  // pans/jumps there — see LocationPickerAuto's recenterToken prop.
  const [pinSeq, setPinSeq] = useState(0);

  useEffect(() => {
    if (business?.profile && !dirty) setP(business.profile);
  }, [business, dirty]);

  useEffect(() => {
    if (business && !identityDirty) {
      setIdentity({ name: business.name, town: business.town, lat: business.lat, lng: business.lng });
    }
  }, [business, identityDirty]);

  const save = useMutation({
    mutationFn: () => api.put('/vendor/business', p),
    onSuccess: () => {
      setDirty(false);
      setJustSaved(true);
      setTimeout(() => setJustSaved(false), 2400);
      qc.invalidateQueries({ queryKey: ['business'] });
    },
  });

  const updateCategory = useMutation({
    mutationFn: (category: string) => api.patch<Vendor>('/vendor/business/category', { category }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['business'] }),
  });

  const saveIdentity = useMutation({
    mutationFn: () => api.patch<Vendor>('/vendor/business/identity', identity),
    onSuccess: () => {
      setIdentityDirty(false);
      setIdentitySaved(true);
      setTimeout(() => setIdentitySaved(false), 2400);
      qc.invalidateQueries({ queryKey: ['business'] });
    },
  });
  const setIdentityField = (patch: Partial<typeof identity>) => {
    setIdentity((prev) => ({ ...prev, ...patch }));
    setIdentityDirty(true);
  };

  const set = (patch: Partial<VendorProfile>) => {
    setP((prev) => ({ ...prev, ...patch }));
    setDirty(true);
  };
  const setWifi = (patch: Partial<NonNullable<VendorProfile['wifi']>>) => {
    set({ wifi: { enabled: p.wifi?.enabled ?? false, score: p.wifi?.score ?? 0, speed: p.wifi?.speed ?? 0, ...patch } });
  };
  const setDayHours = (day: Weekday, patch: Partial<DayHours>) => {
    const current = p.hours?.[day] ?? { open: '09:00', close: '17:00' };
    set({ hours: { ...p.hours, [day]: { ...current, ...patch } } });
  };
  const setContact = (patch: Partial<NonNullable<VendorProfile['contact']>>) => {
    set({ contact: { ...p.contact, ...patch } });
  };

  // Auto-save: debounced while typing, plus an immediate flush() wired to
  // onBlur and to the manual buttons below (kept as a fallback). See
  // useAutoSave's own doc comment for why scheduling from an effect avoids
  // the stale-state race the vendor menu page hit with a naive
  // save-immediately-after-setState approach.
  const identityAutoSave = useAutoSave({
    deps: [identity],
    dirty: identityDirty,
    onSave: () => saveIdentity.mutate(),
  });
  const profileAutoSave = useAutoSave({
    deps: [p],
    dirty,
    onSave: () => save.mutate(),
  });

  const amenities = p.amenities ?? [];
  const bestTimes = p.bestTimes ?? [];
  const remainingPresets = ALL_AMENITIES.filter((a) => !amenities.includes(a));

  const addAmenity = () => {
    const a = newAmenity.trim();
    if (!a || amenities.includes(a)) return;
    set({ amenities: [...amenities, a] });
    setNewAmenity('');
  };

  const isOwner = !me?.vendorRole || me.vendorRole === 'owner';
  const isStaff = me?.vendorRole === 'staff';
  const accessActive = !!grant?.accessExpiresAt && new Date(grant.accessExpiresAt).getTime() > Date.now();
  const pendingGrantId = grant && !grant.redeemedAt ? grant.id : undefined;
  const hasAccess = isOwner || accessActive;
  const accessCountdown = useCountdown(accessActive ? grant?.accessExpiresAt : undefined);

  return (
    <Shell>
      <div>
        <div className="flex items-start justify-between mb-5 gap-4">
          <div>
            <h1 className="text-2xl font-extrabold font-display">Business profile</h1>
            <p className="text-muted font-semibold mt-0.5">This is what travelers see on the Spotto map.</p>
          </div>
          {hasAccess && (
            <button className="btn-accent flex items-center gap-2 px-4 py-2.5 shrink-0 whitespace-nowrap"
              disabled={!dirty || save.isPending} onClick={profileAutoSave.flush}>
              <Save size={16} /> {save.isPending ? 'Saving…' : justSaved ? 'Saved' : 'Save changes'}
            </button>
          )}
        </div>

        {!isOwner && accessActive && accessCountdown && (
          <div className="flex items-center gap-2 rounded-xl bg-accent-tint text-accent px-4 py-2.5 mb-5 font-extrabold text-[13px]">
            <ShieldCheck size={16} /> Temporary edit access expires in {accessCountdown}
          </div>
        )}

        {!isStaff && <UpcomingHolidaysCard />}

        {isStaff && <LockedProfileCard />}
        {!isOwner && !isStaff && !hasAccess && <LockedProfileCard grantId={pendingGrantId} />}

        {hasAccess && (
        <div className="grid gap-6" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(330px, 100%), 1fr))' }}>
          <div className="card p-6">
            <div className="flex items-center justify-between mb-4">
              <CardHeader icon={MapPin} title="Business identity & location" sub="Your name, address, and map pin" />
              <button className="btn-accent flex items-center gap-2 px-3.5 py-2 text-[13px] shrink-0"
                disabled={!identityDirty || saveIdentity.isPending} onClick={identityAutoSave.flush}>
                <Save size={14} /> {saveIdentity.isPending ? 'Saving…' : identitySaved ? 'Saved' : 'Save'}
              </button>
            </div>
            <div className="grid sm:grid-cols-2 gap-3 mb-3">
              <label className="block">
                <span className="block text-[12.5px] font-extrabold mb-1.5 text-muted">Business name</span>
                <input className="input" value={identity.name} onChange={(e) => setIdentityField({ name: e.target.value })}
                  onBlur={identityAutoSave.flush} />
              </label>
              <AddressSearchField
                variant="plain"
                value={identity.town}
                onChange={(town) => setIdentityField({ town })}
                onLocationSelect={(lat, lng) => { setIdentityField({ lat, lng }); setPinSeq((s) => s + 1); }}
                onPlaceDetails={({ phone, website, category, hours }) => {
                  if (phone && !p.contact?.phone) setContact({ phone });
                  if (website && !p.contact?.website) setContact({ website });
                  if (hours && !p.hours) set({ hours });
                  if (category && !business?.category) updateCategory.mutate(category);
                }}
                onBlur={identityAutoSave.flush}
                placeholder="Search for your address…"
              />
            </div>
            <span className="block text-[12.5px] font-extrabold mb-1.5 text-muted">Pin your exact location</span>
            <p className="text-[12px] text-muted font-semibold mb-2">
              Click anywhere on the map to move your pin — this is where travelers get directions to.
            </p>
            <div className="h-[220px] rounded-xl overflow-hidden border border-line">
              {business ? (
                <LocationPicker lat={identity.lat} lng={identity.lng} recenterToken={pinSeq}
                  onChange={(lat, lng) => setIdentityField({ lat, lng })} />
              ) : (
                <div className="h-full flex items-center justify-center text-muted font-semibold text-[13px]">Loading map…</div>
              )}
            </div>
          </div>

          <div className="card p-6">
            <CardHeader icon={Store} title="Business type" sub="What kind of spot are you?" />
            <div className="mb-4">
              <div className="flex overflow-x-auto gap-2 pb-2 mb-3 -mx-2 px-2" style={{ scrollbarWidth: 'none' }}>
                {VENDOR_SEGMENTS.map((segment) => {
                  const currentSegmentId = activeSegment ?? (business?.category ? categoryById(business.category)?.segment : VENDOR_SEGMENTS[0].id) ?? VENDOR_SEGMENTS[0].id;
                  const isActive = currentSegmentId === segment.id;
                  return (
                    <button type="button" key={segment.id}
                      onClick={() => setActiveSegment(segment.id)}
                      className={`whitespace-nowrap rounded-full px-3.5 py-1.5 font-extrabold text-[12px] transition-colors border shrink-0 ${
                        isActive ? 'border-accent bg-accent text-white' : 'border-line bg-sand text-muted hover:border-accent hover:text-accent'
                      }`}>
                      {segment.label}
                    </button>
                  );
                })}
              </div>

              <div className="grid grid-cols-3 gap-2">
                {(() => {
                  const currentSegmentId = activeSegment ?? (business?.category ? categoryById(business.category)?.segment : VENDOR_SEGMENTS[0].id) ?? VENDOR_SEGMENTS[0].id;
                  // @ts-ignore - in case categoriesBySegment requires a specific enum type, but string is fine
                  return categoriesBySegment(currentSegmentId).map((t) => {
                    const Icon = categoryIconComponent(t);
                    const on = isSelectedCategory(business?.category, t);
                    return (
                      <button type="button" key={t.id} disabled={updateCategory.isPending}
                        onClick={() => updateCategory.mutate(t.id)}
                        className={`rounded-xl border px-2 py-2.5 flex flex-col items-center justify-center gap-1 text-[12px] font-extrabold text-center leading-tight transition-colors disabled:opacity-60 ${
                          on ? 'border-accent bg-accent-tint text-accent' : 'border-line bg-sand hover:border-accent'
                        }`}>
                        <Icon size={17} className="mb-0.5" />
                        {t.label}
                      </button>
                    );
                  });
                })()}
              </div>
            </div>
            <div className="flex items-start gap-2 rounded-xl bg-sand p-3">
              <UtensilsCrossed size={15} className="text-accent shrink-0 mt-0.5" />
              <p className="text-[12.5px] text-muted font-semibold">
                Restaurants, cafés and bars get a <span className="font-extrabold text-ink">Menu</span> tab — upload
                dishes with photos and prices.
                {isFoodCategory(business?.category) && <span className="text-accent font-extrabold"> Yours is enabled.</span>}
              </p>
            </div>
          </div>

          {business?.id && <VendorBRCertificate vendorId={business.id} />}
          {business?.id && <VendorPhotos vendorId={business.id} />}
          {business?.id && <VendorDocs vendorId={business.id} />}

          <div className="card p-6">
            <CardHeader icon={Wifi} title="Connectivity & vibe" sub="What digital nomads check first" />
            <div className="flex items-center justify-between py-2.5">
              <div>
                <div className="font-bold text-[14px] text-ink">Wi-Fi available</div>
                <div className="text-[12.5px] text-muted font-semibold">Show a Wi-Fi badge on your pin</div>
              </div>
              <label className="relative inline-flex items-center cursor-pointer shrink-0">
                <input type="checkbox" className="sr-only peer" checked={p.wifi?.enabled ?? false}
                  onChange={(e) => setWifi({ enabled: e.target.checked })} />
                <span className={`w-10 h-6 rounded-full relative transition-colors ${p.wifi?.enabled ? 'bg-accent' : 'bg-line'}`}>
                  <span className={`absolute top-0.5 w-5 h-5 rounded-full bg-white transition-all ${p.wifi?.enabled ? 'left-[22px]' : 'left-0.5'}`} />
                </span>
              </label>
            </div>
            {p.wifi?.enabled && (
              <div className="py-2.5 border-t border-line">
                <div className="flex items-center justify-between mb-2">
                  <span className="font-bold text-[14px] text-ink">Speed rating</span>
                  <span className="font-extrabold text-[15px] text-accent">{(p.wifi?.score ?? 0).toFixed(1)} / 10</span>
                </div>
                <input type="range" min={0} max={10} step={0.1} className="w-full accent-[#0b6f86]"
                  value={p.wifi?.score ?? 0}
                  onChange={(e) => setWifi({ score: parseFloat(e.target.value) })} />
                <div className="flex items-center justify-between text-[11.5px] text-muted font-semibold mt-1">
                  <span>Patchy</span>
                  <span>Blazing fast</span>
                </div>
              </div>
            )}
            <div className="flex items-center justify-between py-2.5 border-t border-line">
              <div>
                <div className="font-bold text-[14px] text-ink flex items-center gap-1.5">
                  <Waves size={15} className="text-accent" /> Sea view
                </div>
                <div className="text-[12.5px] text-muted font-semibold">Tag your spot as ocean-facing</div>
              </div>
              <label className="relative inline-flex items-center cursor-pointer shrink-0">
                <input type="checkbox" className="sr-only peer" checked={p.seaView ?? false}
                  onChange={(e) => set({ seaView: e.target.checked })} />
                <span className={`w-10 h-6 rounded-full relative transition-colors ${p.seaView ? 'bg-accent' : 'bg-line'}`}>
                  <span className={`absolute top-0.5 w-5 h-5 rounded-full bg-white transition-all ${p.seaView ? 'left-[22px]' : 'left-0.5'}`} />
                </span>
              </label>
            </div>
          </div>

          <div className="card p-6">
            <CardHeader icon={Monitor} title="Live vibe" sub="What real travelers are saying right now — read-only" />
            {business?.vibe && business.vibe.sampleSize > 0 ? (
              <div className="grid grid-cols-3 gap-2.5">
                {VIBE_DIMENSIONS
                  .filter((d) => (categoryById(business.category)?.relevantVibeDimensions ?? ['productive', 'loud', 'crowded']).includes(d.id))
                  .map((d) => {
                    const Icon = d.icon;
                    return (
                      <div key={d.id} className="rounded-xl border border-line bg-sand py-3.5 flex flex-col items-center gap-1.5">
                        <Icon size={18} className="text-accent" />
                        <span className="font-extrabold text-[15px] text-ink">{business.vibe![d.id].toFixed(1)}</span>
                        <span className="text-[11px] font-bold text-muted">{d.label}</span>
                      </div>
                    );
                  })}
              </div>
            ) : (
              <p className="text-[12.5px] text-muted font-semibold">
                No live vibe check-ins yet — travelers rate this from your business page once they've visited.
              </p>
            )}
            {business?.vibe && business.vibe.sampleSize > 0 && (
              <p className="text-[11px] text-muted font-semibold mt-2.5">
                From {business.vibe.sampleSize} traveler{business.vibe.sampleSize === 1 ? '' : 's'} in the last 30 days
              </p>
            )}
          </div>

          <div className="card p-6">
            <CardHeader icon={Clock} title="Best time to visit" sub="Helps Spotto rank you at the right hours" />
            <div className="grid grid-cols-3 gap-2.5">
              {TIME_SLOTS.map((t) => {
                const Icon = t.icon;
                const on = bestTimes.includes(t.id);
                return (
                  <button key={t.id}
                    className={`rounded-xl border py-3.5 flex flex-col items-center gap-1.5 font-extrabold text-[12.5px] transition-colors ${
                      on ? 'border-accent bg-accent text-white' : 'border-line bg-white text-ink hover:border-accent'
                    }`}
                    onClick={() => set({ bestTimes: on ? bestTimes.filter((x) => x !== t.id) : [...bestTimes, t.id] })}>
                    <Icon size={22} />
                    {t.label}
                  </button>
                );
              })}
            </div>
            <p className="text-[12px] text-muted font-semibold mt-3">
              Selected windows get a small boost in map ranking during those hours.
            </p>
          </div>

          <div className="card p-6">
            <div className="flex items-center justify-between mb-4">
              <CardHeader icon={Clock} title="Hours" sub="When travelers can expect you to be open, day by day" />
              <button type="button" onClick={() => {
                const monday = p.hours?.monday ?? { open: '09:00', close: '17:00' };
                set({ hours: Object.fromEntries(WEEKDAYS.map((d) => [d, { ...monday }])) as VendorProfile['hours'] });
              }} className="text-[12px] font-extrabold text-accent shrink-0 whitespace-nowrap">
                Same every day
              </button>
            </div>
            <div className="space-y-2">
              {WEEKDAYS.map((day) => {
                const d: DayHours = p.hours?.[day] ?? { open: '09:00', close: '17:00', closed: true };
                return (
                  <div key={day} className="grid grid-cols-[34px_1fr_1fr_auto] gap-1.5 items-center">
                    <span className="text-[13px] font-extrabold capitalize">{day.slice(0, 3)}</span>
                    <input type="time" className="input min-w-0 !px-2 !text-[12.5px]" disabled={d.closed} value={d.open}
                      onChange={(e) => setDayHours(day, { open: e.target.value })} onBlur={profileAutoSave.flush} />
                    <input type="time" className="input min-w-0 !px-2 !text-[12.5px]" disabled={d.closed} value={d.close}
                      onChange={(e) => setDayHours(day, { close: e.target.value })} onBlur={profileAutoSave.flush} />
                    <label className="relative inline-flex items-center cursor-pointer shrink-0" title="Closed">
                      <input type="checkbox" className="sr-only peer" checked={!d.closed}
                        onChange={(e) => setDayHours(day, { closed: !e.target.checked })} />
                      <span className={`w-9 h-5 rounded-full relative transition-colors ${!d.closed ? 'bg-accent' : 'bg-line'}`}>
                        <span className={`absolute top-0.5 w-4 h-4 rounded-full bg-white transition-all ${!d.closed ? 'left-[18px]' : 'left-0.5'}`} />
                      </span>
                    </label>
                  </div>
                );
              })}
            </div>
            <p className="text-[12px] text-muted font-semibold mt-3">
              Shown to travelers as your open/closed status on the map.
            </p>
          </div>

          <div className="card p-6">
            <CardHeader icon={Phone} title="Contact & links" sub="How travelers can reach or follow you" />
            <div className="space-y-3">
              <label className="block">
                <span className="block text-[12.5px] font-extrabold mb-1.5 text-muted">Phone</span>
                <div className="relative">
                  <Phone size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted" />
                  <input className="input pl-9" placeholder="+94 …" value={p.contact?.phone ?? ''}
                    onChange={(e) => setContact({ phone: e.target.value })} onBlur={profileAutoSave.flush} />
                </div>
              </label>
              <label className="block">
                <span className="block text-[12.5px] font-extrabold mb-1.5 text-muted">Website</span>
                <div className="relative">
                  <Globe size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted" />
                  <input className="input pl-9" placeholder="https://…" value={p.contact?.website ?? ''}
                    onChange={(e) => setContact({ website: e.target.value })} onBlur={profileAutoSave.flush} />
                </div>
              </label>
              <div className="grid grid-cols-2 gap-3">
                <label className="block">
                  <span className="block text-[12.5px] font-extrabold mb-1.5 text-muted">Instagram</span>
                  <div className="relative">
                    <Instagram size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted" />
                    <input className="input pl-9" placeholder="@yourbusiness" value={p.contact?.instagram ?? ''}
                      onChange={(e) => setContact({ instagram: e.target.value })} onBlur={profileAutoSave.flush} />
                  </div>
                </label>
                <label className="block">
                  <span className="block text-[12.5px] font-extrabold mb-1.5 text-muted">Facebook</span>
                  <div className="relative">
                    <Facebook size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted" />
                    <input className="input pl-9" placeholder="facebook.com/…" value={p.contact?.facebook ?? ''}
                      onChange={(e) => setContact({ facebook: e.target.value })} onBlur={profileAutoSave.flush} />
                  </div>
                </label>
              </div>
            </div>
          </div>

          <div className="card p-6">
            <CardHeader icon={FileText} title="About & amenities" sub="A short, honest pitch" />
            <textarea className="input min-h-[90px] resize-y mb-4" rows={3} value={p.blurb ?? ''}
              onChange={(e) => set({ blurb: e.target.value })} onBlur={profileAutoSave.flush} />

            <div className="flex flex-wrap gap-2 mb-3">
              {amenities.map((a) => (
                <span key={a} className="rounded-full bg-sand text-ink px-3.5 py-1.5 font-bold text-[12.5px] flex items-center gap-1.5">
                  <Check size={13} className="text-accent" /> {a}
                  <button type="button" onClick={() => set({ amenities: amenities.filter((x) => x !== a) })}>
                    <X size={12} />
                  </button>
                </span>
              ))}
              <button type="button" onClick={() => setAddingAmenity((v) => !v)}
                className="rounded-full border border-dashed border-accent text-accent px-3.5 py-1.5 font-extrabold text-[12.5px] flex items-center gap-1">
                <Plus size={13} /> Add
              </button>
            </div>

            {addingAmenity && (
              <div className="rounded-xl bg-sand p-3 space-y-3">
                {remainingPresets.length > 0 && (
                  <div className="flex flex-wrap gap-2">
                    {remainingPresets.map((a) => (
                      <button key={a} type="button" onClick={() => set({ amenities: [...amenities, a] })}
                        className="rounded-full border border-line bg-white px-3.5 py-1.5 font-extrabold text-[12.5px] hover:border-accent transition-colors">
                        {a}
                      </button>
                    ))}
                  </div>
                )}
                <div className="flex items-center gap-2">
                  <input className="input flex-1" placeholder="Add a custom amenity…" value={newAmenity}
                    onChange={(e) => setNewAmenity(e.target.value)}
                    onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); addAmenity(); } }} />
                  <button type="button" onClick={addAmenity} disabled={!newAmenity.trim()}
                    className="rounded-xl border border-dashed border-line px-4 py-2.5 font-extrabold text-[13px] flex items-center gap-1.5 disabled:opacity-50 hover:border-accent transition-colors shrink-0">
                    <Plus size={15} /> Add
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>
        )}
      </div>
    </Shell>
  );
}
