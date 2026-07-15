'use client';

// Geo-analytics — real Leaflet/OSM map with region demand/supply bubbles,
// individual vendor pins (red-flagged from bad reviews), an investigation
// pin for checking an arbitrary area, and the region opportunity leaderboard.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { AlertTriangle, CheckCircle2, MapPin, ShieldAlert, Undo2, X } from 'lucide-react';
import dynamic from 'next/dynamic';
import { useState } from 'react';

import { timeAgo, type Region, type Review, type Vendor } from '@spotto/shared';
import Shell from '../../components/shell';
import { regionTone } from '../../components/region-tone';
import { api } from '../../lib/api';

// Leaflet touches `window` at import time, so the map must not be part of
// the server-rendered bundle.
const VendorMap = dynamic(() => import('../../components/vendor-map'), {
  ssr: false,
  loading: () => <div className="h-full flex items-center justify-center text-muted font-semibold">Loading map…</div>,
});

export default function GeoPage() {
  const qc = useQueryClient();
  const { data: regions = [] } = useQuery<Region[]>({
    queryKey: ['admin-regions'],
    queryFn: () => api.get<Region[]>('/admin/regions'),
  });
  const { data: vendors = [] } = useQuery<Vendor[]>({
    queryKey: ['admin-vendors-map'],
    queryFn: () => api.get<Vendor[]>('/admin/vendors/map'),
    refetchInterval: 15000,
  });

  const [selectedVendorId, setSelectedVendorId] = useState<string | null>(null);
  const [investigationPin, setInvestigationPin] = useState<{ lat: number; lng: number } | null>(null);
  const [nearbyVendors, setNearbyVendors] = useState<Vendor[] | null>(null);

  const selectedVendor = vendors.find((v) => v.id === selectedVendorId) ?? null;
  const { data: vendorReviews = [] } = useQuery<Review[]>({
    queryKey: ['admin-vendor-reviews', selectedVendorId],
    queryFn: () => api.get<Review[]>(`/admin/vendors/${selectedVendorId}/reviews`),
    enabled: !!selectedVendorId,
  });

  const suspend = useMutation({
    mutationFn: (id: string) => api.post(`/admin/vendors/${id}/suspend`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin-vendors-map'] }),
  });
  const reinstate = useMutation({
    mutationFn: (id: string) => api.post(`/admin/vendors/${id}/reinstate`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin-vendors-map'] }),
  });

  const flaggedVendors = vendors.filter((v) => v.flagged);
  const leaderboard = [...regions]
    .map((r) => ({ ...r, gap: r.travelers - r.vendors }))
    .sort((a, b) => b.gap - a.gap);

  const selectVendor = (id: string) => {
    setInvestigationPin(null);
    setNearbyVendors(null);
    setSelectedVendorId(id);
  };

  const investigate = async (lat: number, lng: number) => {
    setSelectedVendorId(null);
    setInvestigationPin({ lat, lng });
    const near = await api.get<Vendor[]>(`/admin/vendors/near?lat=${lat}&lng=${lng}&radiusKm=5`);
    setNearbyVendors(near);
  };

  return (
    <Shell>
      <div className="grid grid-cols-[1fr_360px] gap-5 items-start">
        <div className="panel p-1 overflow-hidden" style={{ height: 640 }}>
          <VendorMap
            regions={regions}
            vendors={vendors}
            selectedVendorId={selectedVendorId}
            onSelectVendor={selectVendor}
            investigationPin={investigationPin}
            onInvestigate={investigate}
          />
        </div>

        <div className="space-y-4">
          {selectedVendor && (
            <VendorDetail
              vendor={selectedVendor}
              reviews={vendorReviews}
              onClose={() => setSelectedVendorId(null)}
              onSuspend={() => suspend.mutate(selectedVendor.id)}
              onReinstate={() => reinstate.mutate(selectedVendor.id)}
              busy={suspend.isPending || reinstate.isPending}
            />
          )}

          {investigationPin && !selectedVendor && (
            <div className="panel p-5">
              <div className="flex items-center justify-between mb-1">
                <h2 className="font-bold text-[15px] flex items-center gap-2">
                  <MapPin size={16} /> Investigating this area
                </h2>
                <button className="btn !p-1.5" onClick={() => { setInvestigationPin(null); setNearbyVendors(null); }}>
                  <X size={14} />
                </button>
              </div>
              <p className="text-[12px] text-muted font-semibold mb-3">
                {investigationPin.lat.toFixed(4)}, {investigationPin.lng.toFixed(4)} · 5km radius
              </p>
              {nearbyVendors === null && <p className="text-[13px] text-muted font-semibold">Loading…</p>}
              {nearbyVendors?.length === 0 && (
                <p className="text-[13px] text-muted font-semibold">No vendors within 5km of this point.</p>
              )}
              {nearbyVendors && nearbyVendors.length > 0 && (
                <div className="space-y-1">
                  {nearbyVendors.map((v) => (
                    <button key={v.id} onClick={() => selectVendor(v.id)}
                      className="w-full text-left px-3 py-2.5 rounded-lg hover:bg-[color-mix(in_srgb,var(--accent)_8%,transparent)] flex items-center justify-between">
                      <span>
                        <span className="block font-bold text-[13.5px]">{v.name}</span>
                        <span className="block text-[11.5px] text-muted font-semibold">{v.category} · {v.town}</span>
                      </span>
                      {v.flagged && <ShieldAlert size={15} className="text-[#e11d48]" />}
                    </button>
                  ))}
                </div>
              )}
            </div>
          )}

          {flaggedVendors.length > 0 && (
            <div className="panel overflow-hidden">
              <p className="px-4 py-3 border-b border-line font-bold text-[13.5px] flex items-center gap-2" style={{ color: '#e11d48' }}>
                <ShieldAlert size={15} /> Flagged vendors ({flaggedVendors.length})
              </p>
              <div className="max-h-[200px] overflow-y-auto">
                {flaggedVendors.map((v) => (
                  <button key={v.id} onClick={() => selectVendor(v.id)}
                    className="w-full text-left px-4 py-3 border-b border-line last:border-0 flex items-center justify-between"
                    style={{ background: selectedVendorId === v.id ? 'color-mix(in srgb, var(--accent) 10%, transparent)' : 'transparent' }}>
                    <span>
                      <span className="block font-bold text-[13.5px]">{v.name}</span>
                      <span className="block text-[11.5px] text-muted font-semibold">{v.town}</span>
                    </span>
                    <span className="font-bold text-[13px]" style={{ color: '#e11d48' }}>{v.badReviewCount} bad</span>
                  </button>
                ))}
              </div>
            </div>
          )}

          <div className="panel overflow-hidden">
            <p className="px-4 py-3 border-b border-line font-bold text-[13.5px]">Opportunity leaderboard</p>
            <div className="max-h-[300px] overflow-y-auto">
              {leaderboard.map((r) => {
                const tone = regionTone(r.gap);
                return (
                  <div key={r.id}
                    className="w-full text-left px-4 py-3 border-b border-line last:border-0 flex items-center justify-between">
                    <span>
                      <span className="block font-bold text-[13.5px]">{r.name}</span>
                      <span className="block text-[11.5px] font-semibold" style={{ color: tone.ring }}>{tone.label}</span>
                    </span>
                    <span className="font-bold text-[14px]" style={{ color: tone.ring }}>{r.gap > 0 ? '+' : ''}{r.gap}</span>
                  </div>
                );
              })}
            </div>
          </div>
        </div>
      </div>
    </Shell>
  );
}

function VendorDetail({
  vendor,
  reviews,
  onClose,
  onSuspend,
  onReinstate,
  busy,
}: {
  vendor: Vendor;
  reviews: Review[];
  onClose: () => void;
  onSuspend: () => void;
  onReinstate: () => void;
  busy: boolean;
}) {
  return (
    <div className="panel p-5">
      <div className="flex items-start justify-between mb-1">
        <div>
          <h2 className="font-bold text-[16px]">{vendor.name}</h2>
          <p className="text-[12.5px] text-muted font-semibold">{vendor.category} · {vendor.town}</p>
        </div>
        <button className="btn !p-1.5" onClick={onClose}><X size={14} /></button>
      </div>

      <div className="flex items-center gap-2 mt-3">
        <span className="text-[11px] font-bold uppercase rounded-full px-2.5 py-1"
          style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
          {vendor.status}
        </span>
        <span className="text-[11px] font-bold uppercase rounded-full px-2.5 py-1"
          style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
          {vendor.tier}
        </span>
      </div>

      {vendor.flagged && (
        <div className="rounded-xl p-4 mt-3" style={{ background: '#fde4d84d', border: '1px solid #e11d4855' }}>
          <p className="font-bold text-[13px] flex items-center gap-2" style={{ color: '#e11d48' }}>
            <ShieldAlert size={14} /> {vendor.badReviewCount} bad reviews (1-2 stars)
          </p>
          <div className="mt-2 space-y-2 max-h-[220px] overflow-y-auto">
            {reviews.map((rv) => (
              <div key={rv.id} className="text-[12.5px] font-medium">
                <span className="font-bold">{'★'.repeat(rv.rating)}{'☆'.repeat(5 - rv.rating)}</span>{' '}
                {rv.travellerName && <span className="text-muted">{rv.travellerName} · </span>}
                {rv.text}
                <span className="block text-[11px] text-muted font-semibold">{timeAgo(rv.createdAt)}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      <div className="flex gap-2 mt-4">
        {vendor.status !== 'suspended' ? (
          <button className="btn btn-danger flex-1 py-2.5 flex items-center justify-center gap-2" disabled={busy} onClick={onSuspend}>
            <AlertTriangle size={15} /> Suspend
          </button>
        ) : (
          <button className="btn btn-primary flex-1 py-2.5 flex items-center justify-center gap-2" disabled={busy} onClick={onReinstate}>
            <CheckCircle2 size={15} /> Reinstate
          </button>
        )}
      </div>
      {vendor.status === 'suspended' && (
        <p className="text-[11.5px] text-muted font-semibold mt-2 flex items-center gap-1.5">
          <Undo2 size={12} /> Suspended vendors can't log into the vendor portal until reinstated.
        </p>
      )}
    </div>
  );
}
