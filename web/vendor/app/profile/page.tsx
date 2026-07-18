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
  Camera, Check, Clock, Coffee, FileText, ImagePlus, Laptop, Plus, Save, Star,
  Store, Sun, Sunrise, Sunset, UtensilsCrossed, Waves, Wifi, X, LucideIcon,
} from 'lucide-react';
import { useEffect, useState } from 'react';

import type { Vendor, VendorProfile } from '@spotto/shared';
import Shell, { isFoodCategory, useBusiness } from '../../components/shell';
import { api } from '../../lib/api';
import { BUSINESS_CATEGORIES, isSelectedCategory } from '../../lib/business-categories';

const TIME_SLOTS = [
  { id: 'morning', label: 'Morning', icon: Sunrise },
  { id: 'afternoon', label: 'Afternoon', icon: Sun },
  { id: 'evening', label: 'Evening', icon: Sunset },
];

const ALL_AMENITIES = [
  'Power outlets', 'Standing desks', 'Air-con loft', 'Specialty coffee', 'Quiet zone',
  'Sea-view deck', 'Board storage', 'Outdoor shower', 'Live music', 'Parking',
];

const PHOTO_SLOTS = [
  { id: 'p1', label: 'Sea View', gradient: 'linear-gradient(135deg, #2bb6c4, #0b6f86)', icon: Waves },
  { id: 'p2', label: 'Co-work loft', gradient: 'linear-gradient(135deg, #0e8c8a, #0a5f6e)', icon: Laptop },
  { id: 'p3', label: 'Coffee bar', gradient: 'linear-gradient(135deg, #c98a3c, #8a5a23)', icon: Coffee },
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

export default function ProfilePage() {
  const qc = useQueryClient();
  const { data: business } = useBusiness();
  const [p, setP] = useState<VendorProfile>({});
  const [dirty, setDirty] = useState(false);
  const [justSaved, setJustSaved] = useState(false);
  const [newAmenity, setNewAmenity] = useState('');
  const [addingAmenity, setAddingAmenity] = useState(false);

  useEffect(() => {
    if (business?.profile && !dirty) setP(business.profile);
  }, [business, dirty]);

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

  const set = (patch: Partial<VendorProfile>) => {
    setP((prev) => ({ ...prev, ...patch }));
    setDirty(true);
  };
  const setWifi = (patch: Partial<NonNullable<VendorProfile['wifi']>>) => {
    set({ wifi: { enabled: p.wifi?.enabled ?? false, score: p.wifi?.score ?? 0, speed: p.wifi?.speed ?? 0, ...patch } });
  };

  const amenities = p.amenities ?? [];
  const bestTimes = p.bestTimes ?? [];
  const remainingPresets = ALL_AMENITIES.filter((a) => !amenities.includes(a));

  const addAmenity = () => {
    const a = newAmenity.trim();
    if (!a || amenities.includes(a)) return;
    set({ amenities: [...amenities, a] });
    setNewAmenity('');
  };

  return (
    <Shell>
      <div>
        <div className="flex items-start justify-between mb-5 gap-4">
          <div>
            <h1 className="text-2xl font-extrabold">Business profile</h1>
            <p className="text-muted font-semibold mt-0.5">This is what travelers see on the Spotto map.</p>
          </div>
          <button className="btn-accent flex items-center gap-2 px-4 py-2.5 shrink-0 whitespace-nowrap"
            disabled={!dirty || save.isPending} onClick={() => save.mutate()}>
            <Save size={16} /> {save.isPending ? 'Saving…' : justSaved ? 'Saved' : 'Save changes'}
          </button>
        </div>

        <div className="grid gap-6" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(330px, 100%), 1fr))' }}>
          <div className="card p-6">
            <CardHeader icon={Store} title="Business type" sub="What kind of spot are you?" />
            <div className="grid grid-cols-3 gap-2 mb-4">
              {BUSINESS_CATEGORIES.map((t) => {
                const Icon = t.icon;
                const on = isSelectedCategory(business?.category, t);
                return (
                  <button type="button" key={t.id} disabled={updateCategory.isPending}
                    onClick={() => updateCategory.mutate(t.id)}
                    className={`rounded-xl border px-2 py-2.5 flex flex-col items-center gap-1 text-[12px] font-extrabold text-center leading-tight transition-colors disabled:opacity-60 ${
                      on ? 'border-accent bg-accent-tint text-accent' : 'border-line bg-sand hover:border-accent'
                    }`}>
                    <Icon size={17} />
                    {t.label}
                  </button>
                );
              })}
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

          <div className="card p-6">
            <CardHeader icon={Camera} title="Photos" sub="Show off your sea view & space" />
            <div className="grid grid-cols-2 gap-2.5">
              {PHOTO_SLOTS.map((slot, i) => {
                const Icon = slot.icon;
                return (
                  <div key={slot.id} className="relative rounded-xl overflow-hidden aspect-[4/3]" style={{ background: slot.gradient }}>
                    <Icon size={52} className="absolute -right-1 -bottom-2 text-white/25" />
                    <div className="absolute left-2 bottom-2 rounded-md px-2 py-0.5 text-[11px] font-extrabold text-white bg-black/30">
                      {slot.label}
                    </div>
                    {i === 0 && (
                      <div className="absolute right-2 top-2 rounded-full px-2 py-0.5 text-[10px] font-extrabold flex items-center gap-1 bg-[var(--sun)] text-[#5a3b00]">
                        <Star size={10} className="fill-current" /> Cover
                      </div>
                    )}
                  </div>
                );
              })}
              <button type="button"
                className="aspect-[4/3] rounded-xl border-2 border-dashed border-line flex flex-col items-center justify-center gap-1 text-muted hover:opacity-80 transition-opacity">
                <ImagePlus size={24} />
                <span className="text-[12px] font-bold">Add photo</span>
              </button>
            </div>
          </div>

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
            <CardHeader icon={FileText} title="About & amenities" sub="A short, honest pitch" />
            <textarea className="input min-h-[90px] resize-y mb-4" rows={3} value={p.blurb ?? ''}
              onChange={(e) => set({ blurb: e.target.value })} />

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
      </div>
    </Shell>
  );
}
