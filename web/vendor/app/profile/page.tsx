'use client';

// Business profile — the public listing travelers see on the map. Port of
// the "Business profile" mockup: business-type picker (drives the sidebar's
// Menu tab, see isFoodCategory in components/shell.tsx), a decorative
// coming-soon photos grid (no upload infra yet), connectivity/vibe, best
// time to visit, and a combined about + amenities card with custom-amenity
// add. Category is a top-level vendors column so it saves through its own
// PATCH endpoint immediately on click; everything else batches into the
// profile document and saves via the header's "Save changes" button.

import { useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Coffee, Image as ImageIcon, Martini, Palette, Plus, Save, Store,
  Utensils, Waves, Wifi, X, LucideIcon,
} from 'lucide-react';
import { useEffect, useState } from 'react';

import type { Vendor, VendorProfile } from '@spotto/shared';
import Shell, { isFoodCategory, useBusiness } from '../../components/shell';
import { api } from '../../lib/api';

const CATEGORIES: { id: string; label: string; icon: LucideIcon }[] = [
  { id: 'restaurant', label: 'Restaurant', icon: Utensils },
  { id: 'cafe', label: 'Café', icon: Coffee },
  { id: 'bar', label: 'Bar & Sunset', icon: Martini },
  { id: 'surf', label: 'Surf & Water', icon: Waves },
  { id: 'craft', label: 'Craft & Studio', icon: Palette },
  { id: 'shop', label: 'Shop', icon: Store },
];

function isSelectedCategory(current: string | undefined, t: { id: string; label: string }) {
  const c = (current ?? '').toLowerCase().trim();
  return c === t.id || c === t.label.toLowerCase();
}

const TIME_SLOTS = [
  { id: 'morning', label: 'Morning' },
  { id: 'afternoon', label: 'Afternoon' },
  { id: 'evening', label: 'Evening' },
];

const ALL_AMENITIES = [
  'Power outlets', 'Standing desks', 'Air-con loft', 'Specialty coffee', 'Quiet zone',
  'Sea-view deck', 'Board storage', 'Outdoor shower', 'Live music', 'Parking',
];

export default function ProfilePage() {
  const qc = useQueryClient();
  const { data: business } = useBusiness();
  const [p, setP] = useState<VendorProfile>({});
  const [dirty, setDirty] = useState(false);
  const [newAmenity, setNewAmenity] = useState('');

  useEffect(() => {
    if (business?.profile && !dirty) setP(business.profile);
  }, [business, dirty]);

  const save = useMutation({
    mutationFn: () => api.put('/vendor/business', p),
    onSuccess: () => {
      setDirty(false);
      qc.invalidateQueries({ queryKey: ['business'] });
    },
  });

  const updateCategory = useMutation({
    mutationFn: (category: string) => api.patch<Vendor>('/vendor/business/category', { category }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['business'] }),
  });

  const set = (patch: Partial<VendorProfile>) => {
    setP((prev) => ({ ...prev, ...patch }));
    setDirty(true);
  };
  const setWifi = (patch: Partial<NonNullable<VendorProfile['wifi']>>) => {
    set({ wifi: { enabled: p.wifi?.enabled ?? false, score: p.wifi?.score ?? 0, speed: p.wifi?.speed ?? 0, ...patch } });
  };

  const amenities = p.amenities ?? [];
  const bestTimes = p.bestTimes ?? [];
  const customAmenities = amenities.filter((a) => !ALL_AMENITIES.includes(a));

  const addAmenity = () => {
    const a = newAmenity.trim();
    if (!a || amenities.includes(a)) return;
    set({ amenities: [...amenities, a] });
    setNewAmenity('');
  };

  return (
    <Shell>
      <div className="max-w-2xl">
        <div className="flex items-start justify-between mb-5">
          <div>
            <h1 className="text-2xl font-extrabold">Business profile</h1>
            <p className="text-muted font-semibold mt-0.5">This is what travelers see on the Spotto map.</p>
          </div>
          <button className="btn-accent flex items-center gap-2 px-4 py-2.5"
            disabled={!dirty || save.isPending} onClick={() => save.mutate()}>
            <Save size={16} /> {save.isPending ? 'Saving…' : dirty ? 'Save changes' : 'Saved'}
          </button>
        </div>

        <div className="space-y-4">
          <div className="card p-6">
            <h2 className="font-extrabold text-[16px] mb-1">Business type</h2>
            <p className="text-[13px] text-muted font-semibold mb-4">What kind of spot is this?</p>
            <div className="grid grid-cols-3 gap-2 mb-4">
              {CATEGORIES.map((t) => {
                const Icon = t.icon;
                const on = isSelectedCategory(business?.category, t);
                return (
                  <button type="button" key={t.id} disabled={updateCategory.isPending}
                    onClick={() => updateCategory.mutate(t.id)}
                    className={`rounded-xl border px-2 py-2.5 flex flex-col items-center gap-1 text-[12px] font-extrabold transition-colors disabled:opacity-60 ${
                      on ? 'border-accent bg-accent-tint text-accent' : 'border-line bg-white hover:border-accent'
                    }`}>
                    <Icon size={17} />
                    {t.label}
                  </button>
                );
              })}
            </div>
            <p className="text-[12.5px] text-muted font-semibold rounded-xl bg-sand p-3">
              Restaurants, cafés and bars get an extra <span className="font-extrabold text-ink">Menu tab</span> for
              managing dishes and drinks.
              {isFoodCategory(business?.category) && <span className="text-accent font-extrabold"> Yours is enabled.</span>}
            </p>
          </div>

          <div className="card p-6">
            <h2 className="font-extrabold text-[16px] mb-1">Photos</h2>
            <p className="text-[13px] text-muted font-semibold mb-4">Photo uploads are coming soon — for now our team can add photos on request.</p>
            <div className="grid grid-cols-4 gap-2">
              {[0, 1, 2].map((i) => (
                <div key={i} className="relative aspect-square rounded-xl bg-sand border border-line flex items-center justify-center">
                  <ImageIcon size={20} className="text-muted" />
                  {i === 0 && (
                    <span className="absolute top-1.5 left-1.5 text-[10px] font-extrabold uppercase rounded-full px-2 py-0.5 bg-white/90 text-muted">
                      Cover
                    </span>
                  )}
                </div>
              ))}
              <button type="button" disabled
                className="aspect-square rounded-xl border border-dashed border-line flex flex-col items-center justify-center gap-1 text-muted opacity-60 cursor-not-allowed">
                <Plus size={18} />
                <span className="text-[10.5px] font-extrabold">Add photo</span>
              </button>
            </div>
          </div>

          <div className="card p-6">
            <h2 className="font-extrabold text-[16px] mb-4">Connectivity &amp; vibe</h2>
            <label className="flex items-center justify-between rounded-xl bg-sand p-4 mb-4">
              <span className="flex items-center gap-2.5 font-extrabold text-[14px]">
                <Wifi size={17} className="text-accent" /> Wi-Fi available
              </span>
              <input type="checkbox" className="w-4 h-4 accent-[#0b6f86]"
                checked={p.wifi?.enabled ?? false}
                onChange={(e) => setWifi({ enabled: e.target.checked })} />
            </label>
            <label className="block text-[13px] font-extrabold mb-1.5">
              Speed rating <span className="text-muted font-semibold">({(p.wifi?.score ?? 0).toFixed(1)}/10)</span>
            </label>
            <input type="range" min={0} max={10} step={0.1} className="w-full accent-[#0b6f86]"
              disabled={!p.wifi?.enabled} value={p.wifi?.score ?? 0}
              onChange={(e) => setWifi({ score: parseFloat(e.target.value) })} />
            <div className="flex items-center justify-between text-[11.5px] text-muted font-bold mb-4">
              <span>Patchy</span>
              <span>Blazing fast</span>
            </div>
            <label className="flex items-center justify-between rounded-xl bg-sand p-4">
              <span className="font-extrabold text-[14px]">Sea view</span>
              <input type="checkbox" className="w-4 h-4 accent-[#0b6f86]" checked={p.seaView ?? false}
                onChange={(e) => set({ seaView: e.target.checked })} />
            </label>
          </div>

          <div className="card p-6">
            <h2 className="font-extrabold text-[16px] mb-4">Best time to visit</h2>
            <div className="flex gap-2">
              {TIME_SLOTS.map((t) => {
                const on = bestTimes.includes(t.id);
                return (
                  <button key={t.id}
                    className={`rounded-xl border px-4 py-2.5 font-extrabold text-[13.5px] transition-colors ${
                      on ? 'border-accent bg-accent-tint text-accent' : 'border-line bg-white hover:border-accent'
                    }`}
                    onClick={() => set({ bestTimes: on ? bestTimes.filter((x) => x !== t.id) : [...bestTimes, t.id] })}>
                    {t.label}
                  </button>
                );
              })}
            </div>
          </div>

          <div className="card p-6">
            <h2 className="font-extrabold text-[16px] mb-1">About &amp; amenities</h2>
            <p className="text-[13px] text-muted font-semibold mb-3">A short blurb and what travelers can expect on-site.</p>
            <textarea className="input min-h-[110px] resize-y mb-4" value={p.blurb ?? ''}
              onChange={(e) => set({ blurb: e.target.value })} />

            <div className="flex flex-wrap gap-2 mb-3">
              {ALL_AMENITIES.map((a) => {
                const on = amenities.includes(a);
                return (
                  <button key={a}
                    className={`rounded-full border px-3.5 py-1.5 font-extrabold text-[12.5px] transition-colors ${
                      on ? 'border-accent bg-accent-tint text-accent' : 'border-line bg-white hover:border-accent'
                    }`}
                    onClick={() => set({ amenities: on ? amenities.filter((x) => x !== a) : [...amenities, a] })}>
                    {a}
                  </button>
                );
              })}
              {customAmenities.map((a) => (
                <span key={a} className="rounded-full border border-accent bg-accent-tint text-accent px-3.5 py-1.5 font-extrabold text-[12.5px] flex items-center gap-1.5">
                  {a}
                  <button type="button" onClick={() => set({ amenities: amenities.filter((x) => x !== a) })}>
                    <X size={12} />
                  </button>
                </span>
              ))}
            </div>

            <div className="flex items-center gap-2">
              <input className="input flex-1" placeholder="Add a custom amenity…" value={newAmenity}
                onChange={(e) => setNewAmenity(e.target.value)}
                onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); addAmenity(); } }} />
              <button type="button" onClick={addAmenity} disabled={!newAmenity.trim()}
                className="rounded-xl border border-dashed border-line px-4 py-2.5 font-extrabold text-[13px] flex items-center gap-1.5 disabled:opacity-50 hover:border-accent transition-colors">
                <Plus size={15} /> Add
              </button>
            </div>
          </div>
        </div>
      </div>
    </Shell>
  );
}
