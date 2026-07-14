'use client';

// Public profile editor — hours, Wi-Fi, best times, amenities, blurb.
// PUT /vendor/business with the profile document. Port of ProfileEditor.jsx.

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Save, Wifi } from 'lucide-react';
import { useEffect, useState } from 'react';

import type { VendorProfile } from '@spotto/shared';
import Shell, { useBusiness } from '../../components/shell';
import { api } from '../../lib/api';

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

  const set = (patch: Partial<VendorProfile>) => {
    setP((prev) => ({ ...prev, ...patch }));
    setDirty(true);
  };

  const amenities = p.amenities ?? [];
  const bestTimes = p.bestTimes ?? [];

  return (
    <Shell>
      <div className="max-w-2xl space-y-4">
        <div className="card p-6">
          <h2 className="font-extrabold text-[16px] mb-4">Hours & Wi-Fi</h2>
          <div className="grid grid-cols-2 gap-3 mb-4">
            <div>
              <label className="block text-[13px] font-extrabold mb-1.5">Opens</label>
              <input className="input" type="time" value={p.hours?.open ?? '07:00'}
                onChange={(e) => set({ hours: { open: e.target.value, close: p.hours?.close ?? '21:00' } })} />
            </div>
            <div>
              <label className="block text-[13px] font-extrabold mb-1.5">Closes</label>
              <input className="input" type="time" value={p.hours?.close ?? '21:00'}
                onChange={(e) => set({ hours: { open: p.hours?.open ?? '07:00', close: e.target.value } })} />
            </div>
          </div>
          <label className="flex items-center justify-between rounded-xl bg-sand p-4">
            <span className="flex items-center gap-2.5 font-extrabold text-[14px]">
              <Wifi size={17} className="text-accent" />
              Show Wi-Fi speed ({p.wifi?.speed ?? 0} Mbps · score {p.wifi?.score ?? 0})
            </span>
            <input type="checkbox" className="w-4 h-4 accent-[#0b6f86]"
              checked={p.wifi?.enabled ?? false}
              onChange={(e) => set({ wifi: { enabled: e.target.checked, score: p.wifi?.score ?? 0, speed: p.wifi?.speed ?? 0 } })} />
          </label>
        </div>

        <div className="card p-6">
          <h2 className="font-extrabold text-[16px] mb-4">Best times to visit</h2>
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
            <label className="ml-auto flex items-center gap-2 font-extrabold text-[13.5px]">
              <input type="checkbox" className="w-4 h-4 accent-[#0b6f86]" checked={p.seaView ?? false}
                onChange={(e) => set({ seaView: e.target.checked })} />
              Sea view
            </label>
          </div>
        </div>

        <div className="card p-6">
          <h2 className="font-extrabold text-[16px] mb-4">Amenities</h2>
          <div className="flex flex-wrap gap-2">
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
          </div>
        </div>

        <div className="card p-6">
          <h2 className="font-extrabold text-[16px] mb-4">About</h2>
          <textarea className="input min-h-[110px] resize-y" value={p.blurb ?? ''}
            onChange={(e) => set({ blurb: e.target.value })} />
        </div>

        <button className="btn-accent px-6 py-3 flex items-center gap-2"
          disabled={!dirty || save.isPending} onClick={() => save.mutate()}>
          <Save size={16} /> {save.isPending ? 'Saving…' : dirty ? 'Save profile' : 'Saved'}
        </button>
      </div>
    </Shell>
  );
}
