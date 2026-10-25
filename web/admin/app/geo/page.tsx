'use client';

// Geo-analytics — real Leaflet/OSM map with town circles (live traveller
// activity vs live businesses, see GET /admin/regions),
// vendor pins in their category's colour and icon (ringed red when flagged
// from bad reviews), an investigation pin for checking an arbitrary area
// (a searchable list of the businesses around it), the picked business's
// details under the map, and the region opportunity leaderboard.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  AlertTriangle, CheckCircle2, Clock, ExternalLink, Facebook, Globe, Instagram, MapPin, Navigation,
  Phone, Radio, Search, ShieldAlert, Star, Tag, Ticket, Undo2, User, Users, X,
} from 'lucide-react';
import dynamic from 'next/dynamic';
import Link from 'next/link';
import { useMemo, useState } from 'react';

import {
  ApiError, PIN_TONE_COLORS, VENDOR_CATEGORIES, VENDOR_SEGMENTS, WEEKDAYS, resolveMediaUrl, timeAgo,
  type AdminVendorStats, type DayHours, type Region, type Review, type Vendor, type VendorCategory,
  type VendorProfile, type Weekday,
} from '@spotto/shared';
import Shell from '../../components/shell';
import { OTHER_PIN_COLOR, pinColor, pinIcon } from '../../components/category-pin';
import { regionScore, regionTone } from '../../components/region-tone';
import { InlineErrorBanner } from '../../components/route-status';
import type { MapFocus } from '../../components/vendor-map';
import { api } from '../../lib/api';

// Leaflet touches `window` at import time, so the map must not be part of
// the server-rendered bundle.
const VendorMap = dynamic(() => import('../../components/vendor-map'), {
  ssr: false,
  loading: () => <div className="h-full flex items-center justify-center text-muted font-semibold">Loading map…</div>,
});

export default function GeoPage() {
  const qc = useQueryClient();
  const { data: regions = [], isError: regionsError } = useQuery<Region[]>({
    queryKey: ['admin-regions'],
    queryFn: () => api.get<Region[]>('/admin/regions'),
  });
  const { data: vendors = [], isError: vendorsError } = useQuery<Vendor[]>({
    queryKey: ['admin-vendors-map'],
    queryFn: () => api.get<Vendor[]>('/admin/vendors/map'),
    refetchInterval: 15000,
  });
  // The live taxonomy (admins can add categories), over the built-in one.
  const { data: liveCategories = [] } = useQuery<VendorCategory[]>({
    queryKey: ['admin_categories'],
    queryFn: () => api.get<VendorCategory[]>('/admin/categories'),
  });
  const categories = useMemo(
    () => new Map([...VENDOR_CATEGORIES, ...liveCategories].map((c) => [c.id, c])),
    [liveCategories],
  );
  const categoryLabel = (v: Vendor) => categories.get(v.category)?.label ?? (v.category || 'Uncategorised');

  const [selectedVendorId, setSelectedVendorId] = useState<string | null>(null);
  const [investigationPin, setInvestigationPin] = useState<{ lat: number; lng: number } | null>(null);
  const [nearbyVendors, setNearbyVendors] = useState<Vendor[] | null>(null);
  const [focus, setFocus] = useState<MapFocus | null>(null);

  const selectedVendor = vendors.find((v) => v.id === selectedVendorId)
    ?? nearbyVendors?.find((v) => v.id === selectedVendorId) ?? null;

  const suspend = useMutation({
    mutationFn: (id: string) => api.post(`/admin/vendors/${id}/suspend`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin-vendors-map'] }),
  });
  const reinstate = useMutation({
    mutationFn: (id: string) => api.post(`/admin/vendors/${id}/reinstate`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin-vendors-map'] }),
  });

  const flaggedVendors = vendors.filter((v) => v.flagged);
  // Most traveller activity per live business first.
  const leaderboard = [...regions].sort((a, b) =>
    regionScore(b.travelers, b.vendors) - regionScore(a.travelers, a.vendors) || b.travelers - a.travelers);
  const hasOtherPins = vendors.some((v) => pinColor(categories.get(v.category)) === OTHER_PIN_COLOR);

  // Picking a business from a list flies the map to it; the area list, if
  // any, stays open so the admin can go through it one by one.
  const selectVendor = (v: Vendor) => {
    setSelectedVendorId(v.id);
    setFocus({ lat: v.lat, lng: v.lng });
  };

  const investigate = async (lat: number, lng: number) => {
    setSelectedVendorId(null);
    setInvestigationPin({ lat, lng });
    setNearbyVendors(null);
    const near = await api.get<Vendor[]>(`/admin/vendors/near?lat=${lat}&lng=${lng}&radiusKm=5`);
    setNearbyVendors(near);
  };

  return (
    <Shell>
      {(regionsError || vendorsError) && <InlineErrorBanner message="Couldn't load geo-analytics data — try refreshing." />}
      <div className="grid grid-cols-[1fr_360px] gap-5 items-start">
        <div className="space-y-4 min-w-0">
          <div className="panel p-1 overflow-hidden relative" style={{ height: 640 }}>
            <VendorMap
              regions={regions}
              vendors={vendors}
              categories={categories}
              selectedVendorId={selectedVendorId}
              onSelectVendor={(id) => setSelectedVendorId(id)}
              investigationPin={investigationPin}
              onInvestigate={investigate}
              focus={focus}
            />
            <MapLegend showOther={hasOtherPins} />
          </div>

          {selectedVendor && (
            <PlaceDetails
              key={selectedVendor.id}
              vendor={selectedVendor}
              category={categories.get(selectedVendor.category)}
              categoryLabel={categoryLabel(selectedVendor)}
              onClose={() => setSelectedVendorId(null)}
              onSuspend={() => suspend.mutate(selectedVendor.id)}
              onReinstate={() => reinstate.mutate(selectedVendor.id)}
              busy={suspend.isPending || reinstate.isPending}
            />
          )}
        </div>

        <div className="space-y-4">
          {investigationPin && (
            <AreaPanel
              key={`${investigationPin.lat},${investigationPin.lng}`}
              pin={investigationPin}
              vendors={nearbyVendors}
              categories={categories}
              categoryLabel={categoryLabel}
              selectedId={selectedVendorId}
              onSelect={selectVendor}
              onClose={() => { setInvestigationPin(null); setNearbyVendors(null); }}
            />
          )}

          {flaggedVendors.length > 0 && (
            <div className="panel overflow-hidden">
              <p className="px-4 py-3 border-b border-line font-bold text-[13.5px] flex items-center gap-2" style={{ color: '#e11d48' }}>
                <ShieldAlert size={15} /> Flagged vendors ({flaggedVendors.length})
              </p>
              <div className="max-h-[200px] overflow-y-auto">
                {flaggedVendors.map((v) => (
                  <button key={v.id} onClick={() => selectVendor(v)}
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
            <div className="px-4 py-3 border-b border-line">
              <p className="font-bold text-[13.5px]">Opportunity leaderboard</p>
              <p className="text-[11.5px] text-muted font-semibold mt-0.5">
                Traveller check-ins, reviews and saves in the last 30 days, per live business. Towns at the top need more businesses.
              </p>
            </div>
            <div className="max-h-[300px] overflow-y-auto">
              {leaderboard.map((r) => {
                const tone = regionTone(r.travelers, r.vendors);
                return (
                  <div key={r.id}
                    className="w-full text-left px-4 py-3 border-b border-line last:border-0 flex items-center justify-between gap-3">
                    <span className="min-w-0">
                      <span className="block font-bold text-[13.5px]">{r.name}</span>
                      <span className="block text-[11.5px] font-semibold" style={{ color: tone.ring }}>{tone.label}</span>
                    </span>
                    <span className="text-right shrink-0">
                      <span className="block font-bold text-[13px]">{r.travelers} {r.travelers === 1 ? 'action' : 'actions'}</span>
                      <span className="block text-[11.5px] text-muted font-semibold">{r.vendors} {r.vendors === 1 ? 'business' : 'businesses'}</span>
                    </span>
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

// Pin colours follow the category's group, like the traveller app's map.
function MapLegend({ showOther }: { showOther: boolean }) {
  return (
    <div className="absolute right-3 top-3 z-[1000] rounded-lg px-3 py-2 flex flex-wrap items-center gap-x-3 gap-y-1 max-w-[calc(100%-80px)]"
      style={{ background: 'var(--surface)', border: '1px solid var(--line)', boxShadow: '0 4px 14px rgba(0,0,0,0.15)' }}>
      {VENDOR_SEGMENTS.map((s) => (
        <span key={s.id} className="flex items-center gap-1.5 text-[11.5px] font-semibold">
          <span className="w-2.5 h-2.5 rounded-full" style={{ background: PIN_TONE_COLORS[s.id] }} /> {s.label}
        </span>
      ))}
      {showOther && (
        <span className="flex items-center gap-1.5 text-[11.5px] font-semibold">
          <span className="w-2.5 h-2.5 rounded-full" style={{ background: OTHER_PIN_COLOR }} /> Other
        </span>
      )}
      <span className="flex items-center gap-1.5 text-[11.5px] font-semibold">
        <span className="w-2.5 h-2.5 rounded-full" style={{ boxShadow: '0 0 0 2px #e11d48' }} /> Flagged
      </span>
    </div>
  );
}

/** A round badge in the category's pin colour with its icon — the same look as the map pin. */
function CategoryBadge({ category, size = 28 }: { category: VendorCategory | undefined; size?: number }) {
  const Icon = pinIcon(category);
  return (
    <span className="rounded-full flex items-center justify-center shrink-0"
      style={{ width: size, height: size, background: pinColor(category) }}>
      <Icon size={Math.round(size * 0.5)} color="#fff" strokeWidth={2.4} />
    </span>
  );
}

// "Investigating this area": the businesses within 5km of a clicked point,
// searchable and filterable by category, scrolling inside the panel.
function AreaPanel({
  pin,
  vendors,
  categories,
  categoryLabel,
  selectedId,
  onSelect,
  onClose,
}: {
  pin: { lat: number; lng: number };
  vendors: Vendor[] | null;
  categories: Map<string, VendorCategory>;
  categoryLabel: (v: Vendor) => string;
  selectedId: string | null;
  onSelect: (v: Vendor) => void;
  onClose: () => void;
}) {
  const [query, setQuery] = useState('');
  const [picked, setPicked] = useState<Set<string>>(new Set());

  // One chip per category in this area, most common first.
  const chips = useMemo(() => {
    const counts = new Map<string, number>();
    for (const v of vendors ?? []) counts.set(v.category, (counts.get(v.category) ?? 0) + 1);
    return [...counts.entries()].sort((a, b) => b[1] - a[1]).map(([id, count]) => ({ id, count }));
  }, [vendors]);

  const q = query.trim().toLowerCase();
  const shown = (vendors ?? []).filter((v) =>
    (picked.size === 0 || picked.has(v.category)) &&
    (!q || `${v.name} ${categoryLabel(v)} ${v.town}`.toLowerCase().includes(q)));

  const toggle = (id: string) => setPicked((prev) => {
    const next = new Set(prev);
    if (next.has(id)) next.delete(id); else next.add(id);
    return next;
  });

  return (
    <div className="panel overflow-hidden">
      <div className="p-5 pb-3 border-b border-line">
        <div className="flex items-center justify-between mb-1">
          <h2 className="font-bold text-[15px] flex items-center gap-2">
            <MapPin size={16} /> Investigating this area
          </h2>
          <button className="btn !p-1.5" onClick={onClose} aria-label="Close"><X size={14} /></button>
        </div>
        <p className="text-[12px] text-muted font-semibold mb-3">
          {pin.lat.toFixed(4)}, {pin.lng.toFixed(4)} · 5km radius
          {vendors && vendors.length > 0 && ` · ${shown.length} of ${vendors.length}`}
        </p>
        {vendors && vendors.length > 0 && (
          <>
            <div className="relative">
              <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted" />
              <input className="input !pl-8 w-full" placeholder="Search this area…" value={query}
                onChange={(e) => setQuery(e.target.value)} />
            </div>
            <div className="flex flex-wrap gap-1.5 mt-3">
              {chips.map(({ id, count }) => {
                const category = categories.get(id);
                const color = pinColor(category);
                const on = picked.has(id);
                const Icon = pinIcon(category);
                return (
                  <button key={id} onClick={() => toggle(id)}
                    className="flex items-center gap-1.5 rounded-full pl-1 pr-2.5 py-1 text-[11.5px] font-bold"
                    style={{
                      border: `1.5px solid ${on ? color : 'var(--line)'}`,
                      background: on ? `color-mix(in srgb, ${color} 18%, transparent)` : 'transparent',
                    }}>
                    <span className="w-[18px] h-[18px] rounded-full flex items-center justify-center" style={{ background: color }}>
                      <Icon size={10} color="#fff" strokeWidth={2.6} />
                    </span>
                    {category?.label ?? (id || 'Uncategorised')}
                    <span className="text-muted">{count}</span>
                  </button>
                );
              })}
              {picked.size > 0 && (
                <button onClick={() => setPicked(new Set())}
                  className="rounded-full px-2.5 py-1 text-[11.5px] font-bold text-accent">
                  Show all
                </button>
              )}
            </div>
          </>
        )}
      </div>

      <div className="max-h-[420px] overflow-y-auto p-2">
        {vendors === null && <p className="px-3 py-2 text-[13px] text-muted font-semibold">Loading…</p>}
        {vendors?.length === 0 && (
          <p className="px-3 py-2 text-[13px] text-muted font-semibold">No vendors within 5km of this point.</p>
        )}
        {vendors && vendors.length > 0 && shown.length === 0 && (
          <p className="px-3 py-2 text-[13px] text-muted font-semibold">Nothing here matches.</p>
        )}
        {shown.map((v) => (
          <button key={v.id} onClick={() => onSelect(v)}
            className="w-full text-left px-3 py-2.5 rounded-lg flex items-center gap-3 hover:bg-[color-mix(in_srgb,var(--accent)_8%,transparent)]"
            style={selectedId === v.id ? { background: 'color-mix(in srgb, var(--accent) 14%, transparent)' } : undefined}>
            <CategoryBadge category={categories.get(v.category)} />
            <span className="min-w-0 flex-1">
              <span className="block font-bold text-[13.5px] truncate">{v.name}</span>
              <span className="block text-[11.5px] text-muted font-semibold truncate">{categoryLabel(v)} · {v.town}</span>
            </span>
            {v.flagged && <ShieldAlert size={15} className="text-[#e11d48] shrink-0" />}
          </button>
        ))}
      </div>
    </div>
  );
}

/** The parts of GET /admin/vendors/{id}/place the details panel reads. */
interface PlaceInfo {
  id: string;
  cover?: string;
  coverThumb?: string;
  blurb?: string;
  tags?: string[];
  type?: string;
  hours?: Partial<Record<Weekday, DayHours>>;
  contact?: VendorProfile['contact'];
}

const JS_DAY_TO_WEEKDAY: Weekday[] = ['sunday', 'monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday'];

const BR_STATUS: Record<string, string> = {
  verified: 'Verified',
  invalid: 'Not found in registry',
  pending: 'Checking…',
  manual_review: 'Needs a manual check',
};

function webLink(value: string, base: string): string {
  return /^https?:\/\//i.test(value) ? value : `${base}${value.replace(/^@/, '')}`;
}

// The picked business, shown under the map: photo, category, about, hours,
// contact, location, activity and account details, plus moderation.
function PlaceDetails({
  vendor,
  category,
  categoryLabel,
  onClose,
  onSuspend,
  onReinstate,
  busy,
}: {
  vendor: Vendor;
  category: VendorCategory | undefined;
  categoryLabel: string;
  onClose: () => void;
  onSuspend: () => void;
  onReinstate: () => void;
  busy: boolean;
}) {
  const { data: place } = useQuery<PlaceInfo | null>({
    queryKey: ['admin-vendor-place', vendor.id],
    queryFn: async () => {
      try {
        return await api.get<PlaceInfo>(`/admin/vendors/${vendor.id}/place`);
      } catch (err) {
        if (err instanceof ApiError && err.status === 404) return null;
        throw err;
      }
    },
  });
  const { data: stats } = useQuery<AdminVendorStats>({
    queryKey: ['admin-vendor-stats', vendor.id],
    queryFn: () => api.get<AdminVendorStats>(`/admin/vendors/${vendor.id}/stats`),
  });
  const { data: badReviews = [] } = useQuery<Review[]>({
    queryKey: ['admin-vendor-reviews', vendor.id],
    queryFn: () => api.get<Review[]>(`/admin/vendors/${vendor.id}/reviews`),
    enabled: !!vendor.flagged,
  });

  // The business's own profile first, then what its map listing carries
  // (imported places keep their hours/contact/blurb there).
  const profile: VendorProfile = vendor.profile ?? {};
  const hours = profile.hours && Object.keys(profile.hours).length > 0 ? profile.hours : place?.hours;
  const contact = { ...place?.contact, ...Object.fromEntries(Object.entries(profile.contact ?? {}).filter(([, v]) => v)) };
  const blurb = profile.blurb || place?.blurb;
  const [photoFailed, setPhotoFailed] = useState(false);
  const photo = photoFailed ? undefined : place?.coverThumb || place?.cover;
  const today = JS_DAY_TO_WEEKDAY[new Date().getDay()];
  const color = pinColor(category);
  const CategoryIcon = pinIcon(category);
  const hasLocation = vendor.lat !== 0 || vendor.lng !== 0;

  return (
    <div className="panel overflow-hidden">
      <div className="flex gap-4 p-5 border-b border-line">
        {photo ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img src={resolveMediaUrl(photo)} alt="" className="w-28 h-28 rounded-xl object-cover shrink-0"
            onError={() => setPhotoFailed(true)} />
        ) : (
          <span className="w-28 h-28 rounded-xl flex items-center justify-center shrink-0"
            style={{ background: `color-mix(in srgb, ${color} 18%, transparent)` }}>
            <CategoryIcon size={36} color={color} />
          </span>
        )}
        <div className="min-w-0 flex-1">
          <div className="flex items-start justify-between gap-3">
            <h2 className="font-bold text-[18px] leading-tight">{vendor.name}</h2>
            <button className="btn !p-1.5 shrink-0" onClick={onClose} aria-label="Close"><X size={14} /></button>
          </div>
          <div className="flex flex-wrap items-center gap-2 mt-2">
            <span className="flex items-center gap-1.5 rounded-full pl-1 pr-2.5 py-1 text-[11.5px] font-bold"
              style={{ background: `color-mix(in srgb, ${color} 18%, transparent)`, color }}>
              <span className="w-[18px] h-[18px] rounded-full flex items-center justify-center" style={{ background: color }}>
                <CategoryIcon size={10} color="#fff" strokeWidth={2.6} />
              </span>
              {categoryLabel}
            </span>
            {vendor.town && (
              <span className="text-[12.5px] text-muted font-semibold flex items-center gap-1">
                <MapPin size={12} /> {vendor.town}
              </span>
            )}
          </div>
          <div className="flex flex-wrap items-center gap-2 mt-2.5">
            <Badge>{vendor.status}</Badge>
            <Badge>{vendor.tier || 'Free'}</Badge>
            {vendor.flagged && <Badge danger>{vendor.badReviewCount} bad reviews</Badge>}
          </div>
          {(place?.tags?.length ?? 0) > 0 && (
            <p className="text-[12px] text-muted font-semibold mt-2.5 truncate">{place!.tags!.join(' · ')}</p>
          )}
        </div>
      </div>

      <div className="p-5 space-y-3">
        <div className="grid grid-cols-4 gap-3">
          <Tile icon={Users} label="Check-ins" value={stats ? `${stats.checkins}` : '—'}
            sub={stats ? `${stats.checkins30d} in 30 days` : undefined} />
          <Tile icon={Star} label="Rating" value={stats && stats.reviews > 0 ? stats.avgRating.toFixed(1) : '—'}
            sub={stats ? `${stats.reviews} reviews · ${stats.badReviews} bad` : undefined} />
          <Tile icon={Ticket} label="Redemptions" value={stats ? `${stats.redeemed}` : '—'}
            sub={stats ? `${stats.redeemed30d} in 30 days` : undefined} />
          <Tile icon={Radio} label="Live offers" value={stats ? `${stats.offersLive}` : '—'}
            sub={stats ? `${stats.offersDraft} draft` : undefined} />
        </div>

        {blurb && (
          <Card title="About">
            <p className="text-[13.5px] font-medium leading-relaxed">{blurb}</p>
          </Card>
        )}

        <div className="grid grid-cols-2 gap-3">
          <Card title="Opening hours">
            {hours && Object.keys(hours).length > 0 ? (
              <div className="space-y-1">
                {WEEKDAYS.map((d) => {
                  const h = hours[d];
                  return (
                    <div key={d} className={`flex justify-between text-[13px] ${d === today ? 'font-bold' : 'font-medium'}`}>
                      <span className="capitalize">{d}{d === today && <span className="text-accent"> · today</span>}</span>
                      <span className={h && !h.closed ? '' : 'text-muted'}>
                        {h && !h.closed ? `${h.open} – ${h.close}` : 'Closed'}
                      </span>
                    </div>
                  );
                })}
              </div>
            ) : (
              <p className="text-[13px] text-muted font-semibold flex items-center gap-1.5"><Clock size={13} /> No hours set.</p>
            )}
          </Card>

          <div className="space-y-3">
            <Card title="Contact">
              <Row icon={Phone} label="Phone" value={contact.phone} href={contact.phone ? `tel:${contact.phone.replace(/\s/g, '')}` : undefined} />
              <Row icon={Globe} label="Website" value={contact.website} href={contact.website ? webLink(contact.website, 'https://') : undefined} />
              <Row icon={Instagram} label="Instagram" value={contact.instagram}
                href={contact.instagram ? webLink(contact.instagram, 'https://instagram.com/') : undefined} />
              <Row icon={Facebook} label="Facebook" value={contact.facebook}
                href={contact.facebook ? webLink(contact.facebook, 'https://facebook.com/') : undefined} />
            </Card>
            <Card title="Location">
              <Row icon={MapPin} label="Town" value={vendor.town} />
              <Row icon={Navigation} label="Pin" value={hasLocation ? `${vendor.lat.toFixed(5)}, ${vendor.lng.toFixed(5)}` : undefined}
                href={hasLocation ? `https://www.google.com/maps/search/?api=1&query=${vendor.lat},${vendor.lng}` : undefined} />
            </Card>
          </div>
        </div>

        <Card title="Account">
          <div className="grid grid-cols-2 gap-x-6 gap-y-1.5">
            <Row icon={User} label="Owner" value={vendor.owner} />
            <Row icon={Clock} label="Listed" value={vendor.submittedAt ? `${new Date(vendor.submittedAt).toLocaleDateString()} (${timeAgo(vendor.submittedAt)})` : undefined} />
            <Row icon={Tag} label="BR cert" value={vendor.brCertNumber
              ? `${vendor.brCertNumber}${BR_STATUS[vendor.brCertVerifyStatus] ? ` · ${BR_STATUS[vendor.brCertVerifyStatus]}` : ''}`
              : undefined} />
            <Row icon={Users} label="Team" value={stats ? `${stats.teamTotal} (${stats.teamManagers} mgr · ${stats.teamStaff} staff)` : undefined} />
          </div>
        </Card>

        {vendor.flagged && badReviews.length > 0 && (
          <div className="rounded-xl p-4" style={{ background: '#fde4d84d', border: '1px solid #e11d4855' }}>
            <p className="font-bold text-[13px] flex items-center gap-2" style={{ color: '#e11d48' }}>
              <ShieldAlert size={14} /> {vendor.badReviewCount} bad reviews (1-2 stars)
            </p>
            <div className="mt-2 space-y-2 max-h-[220px] overflow-y-auto">
              {badReviews.map((rv) => (
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

        <div className="flex gap-2 pt-1">
          {vendor.status !== 'suspended' ? (
            <button className="btn btn-danger flex-1 py-2.5 flex items-center justify-center gap-2" disabled={busy} onClick={onSuspend}>
              <AlertTriangle size={15} /> Suspend
            </button>
          ) : (
            <button className="btn btn-primary flex-1 py-2.5 flex items-center justify-center gap-2" disabled={busy} onClick={onReinstate}>
              <CheckCircle2 size={15} /> Reinstate
            </button>
          )}
          <Link href={`/vendors?id=${encodeURIComponent(vendor.id)}`}
            className="btn flex-1 py-2.5 flex items-center justify-center gap-2 font-bold">
            Open full profile <ExternalLink size={14} />
          </Link>
        </div>
        {vendor.status === 'suspended' && (
          <p className="text-[11.5px] text-muted font-semibold flex items-center gap-1.5">
            <Undo2 size={12} /> Suspended vendors can't log into the vendor portal until reinstated.
          </p>
        )}
      </div>
    </div>
  );
}

function Badge({ children, danger }: { children: React.ReactNode; danger?: boolean }) {
  return (
    <span className="text-[11px] font-bold uppercase rounded-full px-2.5 py-1"
      style={danger
        ? { background: '#e11d481a', border: '1px solid #e11d4855', color: '#e11d48' }
        : { background: 'var(--bg)', border: '1px solid var(--line)' }}>
      {children}
    </span>
  );
}

function Card({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="rounded-xl p-4" style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
      <p className="text-[11.5px] font-bold text-muted uppercase mb-2">{title}</p>
      <div className="space-y-1.5">{children}</div>
    </div>
  );
}

function Row({ icon: Icon, label, value, href }: { icon: typeof Phone; label: string; value?: string; href?: string }) {
  return (
    <div className="flex items-center gap-2 text-[13px] min-w-0">
      <Icon size={13} className="text-muted shrink-0" />
      <span className="text-muted font-semibold w-20 shrink-0">{label}</span>
      {value && href ? (
        <a href={href} target={href.startsWith('http') ? '_blank' : undefined} rel="noreferrer"
          className="font-medium truncate text-accent hover:underline">{value}</a>
      ) : (
        <span className="font-medium truncate">{value || '—'}</span>
      )}
    </div>
  );
}

function Tile({ icon: Icon, label, value, sub }: { icon: typeof Phone; label: string; value: string; sub?: string }) {
  return (
    <div className="rounded-xl p-3" style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
      <p className="text-[11px] font-bold text-muted uppercase flex items-center gap-1.5"><Icon size={12} /> {label}</p>
      <p className="font-bold text-[18px] mt-1">{value}</p>
      {sub && <p className="text-[11px] text-muted font-semibold">{sub}</p>}
    </div>
  );
}
