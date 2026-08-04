'use client';

// Audio guides CMS — library grid + new-guide form with depth multi-select
// and publish toggle. Port of AudioCMS.jsx.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Headphones, MapPin, Plus, Radio, X } from 'lucide-react';
import { useState } from 'react';

import type { AudioGuide } from '@spotto/shared';
import Shell from '../../components/shell';
import { InlineErrorBanner } from '../../components/route-status';
import { api } from '../../lib/api';

const DEPTHS = ['Quick', 'Standard', 'In-depth'] as const;
const DEPTH_INFO: Record<string, string> = {
  Quick: '3–5 min · highlights only',
  Standard: '10–15 min · essential story',
  'In-depth': '25–35 min · full history',
};

export default function AudioPage() {
  const qc = useQueryClient();
  const [drawer, setDrawer] = useState(false);
  const { data: guides = [], isError } = useQuery<AudioGuide[]>({
    queryKey: ['admin-audio'],
    queryFn: () => api.get<AudioGuide[]>('/admin/audio-guides'),
  });

  const publish = useMutation({
    mutationFn: ({ id, published }: { id: string; published: boolean }) =>
      api.patch(`/admin/audio-guides/${id}/publish`, { published }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin-audio'] }),
  });

  return (
    <Shell>
      {isError && <InlineErrorBanner message="Couldn't load audio guides — try refreshing." />}
      <div className="flex items-center justify-between mb-5">
        <p className="text-muted font-semibold text-[13.5px]">{guides.length} guides · {guides.filter((g) => g.published).length} published</p>
        <button className="btn btn-primary flex items-center gap-2" onClick={() => setDrawer(true)}>
          <Plus size={15} /> New guide
        </button>
      </div>

      <div className="grid grid-cols-3 gap-4">
        {guides.map((g) => (
          <div key={g.id} className="panel p-5">
            <div className="flex items-start justify-between mb-2">
              <span className="rounded-xl p-2" style={{ background: 'var(--bg)' }}><Headphones size={18} /></span>
              <button
                className="text-[11px] font-bold uppercase rounded-full px-2.5 py-1"
                style={{
                  background: g.published ? 'color-mix(in srgb, var(--accent) 15%, transparent)' : 'var(--bg)',
                  color: g.published ? 'var(--accent)' : 'var(--muted)',
                }}
                onClick={() => publish.mutate({ id: g.id, published: !g.published })}
              >
                {g.published ? 'Published' : 'Draft'}
              </button>
            </div>
            <h3 className="font-bold text-[15px]">{g.site}</h3>
            <p className="text-[12.5px] text-muted font-semibold flex items-center gap-1 mt-0.5">
              <MapPin size={12} /> {g.region}
            </p>
            <div className="flex flex-wrap gap-1.5 mt-3">
              {g.depths.map((d) => (
                <span key={d} className="text-[11px] font-bold rounded-full px-2 py-0.5" style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
                  {d} · {g.duration[d]}
                </span>
              ))}
            </div>
            <p className="flex items-center gap-1.5 text-[12px] text-muted font-semibold mt-3">
              <Radio size={12} /> {g.plays.toLocaleString()} plays
            </p>
          </div>
        ))}
      </div>

      {drawer && <NewGuideDrawer onClose={() => setDrawer(false)} />}
    </Shell>
  );
}

function NewGuideDrawer({ onClose }: { onClose: () => void }) {
  const qc = useQueryClient();
  const [site, setSite] = useState('');
  const [region, setRegion] = useState('');
  const [lat, setLat] = useState('');
  const [lng, setLng] = useState('');
  const [depths, setDepths] = useState<string[]>(['Quick']);

  const create = useMutation({
    mutationFn: () =>
      api.post('/admin/audio-guides', {
        site, region, lat: parseFloat(lat), lng: parseFloat(lng), lang: 'EN', published: false, plays: 0,
        depths, duration: Object.fromEntries(depths.map((d) => [d, d === 'Quick' ? '4m' : d === 'Standard' ? '12m' : '28m'])),
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin-audio'] });
      onClose();
    },
  });

  const validCoords = lat !== '' && lng !== '' && !Number.isNaN(parseFloat(lat)) && !Number.isNaN(parseFloat(lng));

  return (
    <div className="fixed inset-0 z-50 flex justify-end" style={{ background: 'rgba(0,0,0,0.3)' }} onClick={onClose}>
      <div className="w-[420px] h-full overflow-y-auto p-6" style={{ background: 'var(--bg)' }} onClick={(e) => e.stopPropagation()}>
        <div className="flex items-center justify-between mb-5">
          <h2 className="font-bold text-lg">New audio guide</h2>
          <button aria-label="Close" onClick={onClose} className="btn !p-2"><X size={16} /></button>
        </div>

        <label className="block text-[13px] font-bold mb-1.5">Site name</label>
        <input className="input mb-4" value={site} onChange={(e) => setSite(e.target.value)} />
        <label className="block text-[13px] font-bold mb-1.5">Region</label>
        <input className="input mb-4" value={region} onChange={(e) => setRegion(e.target.value)} />

        <div className="grid grid-cols-2 gap-3 mb-4">
          <div>
            <label className="block text-[13px] font-bold mb-1.5">Latitude</label>
            <input className="input" type="number" step="any" placeholder="5.9724" value={lat} onChange={(e) => setLat(e.target.value)} />
          </div>
          <div>
            <label className="block text-[13px] font-bold mb-1.5">Longitude</label>
            <input className="input" type="number" step="any" placeholder="80.4292" value={lng} onChange={(e) => setLng(e.target.value)} />
          </div>
        </div>

        <label className="block text-[13px] font-bold mb-1.5">Audio file (MP3 / WAV)</label>
        <input className="input mb-4" type="file" accept="audio/*" />

        <label className="block text-[13px] font-bold mb-1.5">Depths</label>
        <div className="space-y-2 mb-5">
          {DEPTHS.map((d) => {
            const on = depths.includes(d);
            return (
              <button key={d} type="button"
                onClick={() => setDepths(on ? depths.filter((x) => x !== d) : [...depths, d])}
                className="w-full text-left rounded-xl p-3 border"
                style={{ borderColor: on ? 'var(--accent)' : 'var(--line)', background: on ? 'color-mix(in srgb, var(--accent) 10%, transparent)' : 'var(--surface)' }}>
                <p className="font-bold text-[13.5px]">{d}</p>
                <p className="text-[12px] text-muted font-medium">{DEPTH_INFO[d]}</p>
              </button>
            );
          })}
        </div>

        <button className="btn btn-primary w-full py-3" disabled={!site || !validCoords || create.isPending} onClick={() => create.mutate()}>
          {create.isPending ? 'Creating…' : 'Create guide'}
        </button>
      </div>
    </div>
  );
}
