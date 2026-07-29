'use client';

// Read-only Spots list — lets an admin directly verify what's actually in
// the `places` table (e.g. after an importspots run), independent of the
// traveller app or vendor listings.

import { useQuery } from '@tanstack/react-query';
import { MapPin, Star } from 'lucide-react';

import Shell from '../../components/shell';
import { InlineErrorBanner } from '../../components/route-status';
import { api } from '../../lib/api';

interface AdminPlace {
  id: string;
  name: string;
  address?: string;
  type?: string;
  rating?: number;
  reviews?: number;
  source?: string;
}

export default function SpotsPage() {
  const { data: spots = [], isError } = useQuery<AdminPlace[]>({
    queryKey: ['admin-places'],
    queryFn: () => api.get<AdminPlace[]>('/places'),
  });

  return (
    <Shell>
      {isError && <InlineErrorBanner message="Couldn't load spots — try refreshing." />}
      <p className="text-muted font-semibold text-[13.5px] mb-5">{spots.length} spots</p>

      <div className="grid grid-cols-3 gap-4">
        {spots.map((s) => (
          <div key={s.id} className="panel p-5">
            <div className="flex items-start justify-between mb-2">
              <span className="rounded-xl p-2" style={{ background: 'var(--bg)' }}><MapPin size={18} /></span>
              {s.source && (
                <span
                  className="text-[11px] font-bold uppercase rounded-full px-2.5 py-1"
                  style={{ background: 'var(--bg)', color: 'var(--muted)' }}
                >
                  {s.source}
                </span>
              )}
            </div>
            <h3 className="font-bold text-[15px]">{s.name}</h3>
            {s.address && <p className="text-[12.5px] text-muted font-semibold mt-0.5">{s.address}</p>}
            <div className="flex items-center gap-3 mt-3">
              {s.type && (
                <span className="text-[11px] font-bold rounded-full px-2 py-0.5" style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
                  {s.type}
                </span>
              )}
              {typeof s.rating === 'number' && (
                <p className="flex items-center gap-1 text-[12px] text-muted font-semibold">
                  <Star size={12} /> {s.rating.toFixed(1)}{s.reviews ? ` (${s.reviews})` : ''}
                </p>
              )}
            </div>
          </div>
        ))}
      </div>
    </Shell>
  );
}
