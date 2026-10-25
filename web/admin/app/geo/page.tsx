'use client';

// Geo-analytics — one map of Sri Lanka with switchable heat-map layers:
// where travellers are (rough ~1 km areas, never under 3 per area), what
// they do there, where businesses sign up, and gaps and demand (activity per
// business, places travellers want that aren't on Spotto, upcoming trips).
// One filter bar narrows every layer and compares the period with the one
// before it; the filters live in the URL so a view can be shared. Business
// details stay on the Vendors page — pins here only link to it.

import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { ArrowDownRight, ArrowUpRight, ExternalLink, MapPin, X } from 'lucide-react';
import dynamic from 'next/dynamic';
import { usePathname, useSearchParams } from 'next/navigation';
import { Suspense, useMemo, type ReactNode } from 'react';

import {
  PIN_TONE_COLORS, VENDOR_CATEGORIES, VENDOR_SEGMENTS, timeAgo,
  type AdminGeoArea, type AdminGeoFilters, type AdminGeoHeat, type AdminGeoTown, type Vendor, type VendorCategory,
} from '@spotto/shared';
import Shell from '../../components/shell';
import { OTHER_PIN_COLOR, pinColor } from '../../components/category-pin';
import type { TownCircle } from '../../components/geo-map';
import { regionScore, regionTone } from '../../components/region-tone';
import { InlineErrorBanner } from '../../components/route-status';
import { api } from '../../lib/api';

// Leaflet touches `window` at import time, so the map must not be part of
// the server-rendered bundle.
const GeoMap = dynamic(() => import('../../components/geo-map'), {
  ssr: false,
  loading: () => <div className="h-full flex items-center justify-center text-muted font-semibold">Loading map…</div>,
});

type Layer = 'travellers' | 'activity' | 'businesses' | 'gaps';
type Gap = 'opportunity' | 'wanted' | 'trips';

const LAYERS: { id: Layer; label: string }[] = [
  { id: 'travellers', label: 'Travellers' },
  { id: 'activity', label: 'Activity' },
  { id: 'businesses', label: 'Business growth' },
  { id: 'gaps', label: 'Gaps & demand' },
];

const GAPS: { id: Gap; label: string }[] = [
  { id: 'opportunity', label: 'Opportunity' },
  { id: 'wanted', label: 'Wanted places' },
  { id: 'trips', label: 'Upcoming trips' },
];

const PERIODS = [7, 30, 90];

const ACTIVITY_LABELS: Record<string, string> = {
  checkin: 'Check-ins',
  vibe: 'Vibe check-ins',
  review: 'Reviews',
  save: 'Saves',
  redemption: 'Deal redemptions',
  booking: 'Bookings',
};

const STYLE_LABELS: Record<string, string> = {
  nomad: 'Digital nomad',
  surf: 'Surf',
  luxury: 'Luxury',
  backpacker: 'Backpacker',
};

const STATUSES = [
  { id: '', label: 'Any status' },
  { id: 'approved', label: 'Live' },
  { id: 'pending', label: 'Pending' },
  { id: 'edits', label: 'Edits requested' },
  { id: 'rejected', label: 'Rejected' },
  { id: 'suspended', label: 'Suspended' },
];

// What each view measures, for the summary and the heat colours.
const VIEWS: Record<string, { title: string; unit: string; about: string; gradient: Record<number, string> }> = {
  travellers: {
    title: 'Where travellers are',
    unit: 'travellers',
    about: 'Different app users seen in each area (rough ~1 km location, from travellers with usage analytics on). Areas with fewer than 3 are hidden.',
    gradient: { 0.2: '#38bdf8', 0.5: '#6366f1', 0.75: '#a855f7', 1: '#e11d48' },
  },
  activity: {
    title: 'Traveller activity',
    unit: 'actions',
    about: 'Check-ins, vibe check-ins, reviews, saves, deal redemptions and bookings at places.',
    gradient: { 0.2: '#fde68a', 0.5: '#f59e0b', 0.8: '#ea580c', 1: '#b91c1c' },
  },
  businesses: {
    title: 'Business growth',
    unit: 'sign-ups',
    about: 'New business sign-ups at their map pin. Imported listings are left out unless included.',
    gradient: { 0.2: '#99f6e4', 0.5: '#14b8a6', 0.8: '#0f766e', 1: '#134e4a' },
  },
  opportunity: {
    title: 'Opportunity',
    unit: 'actions',
    about: 'Traveller activity per live business by town. Towns at the top need more businesses.',
    gradient: {},
  },
  wanted: {
    title: 'Wanted places',
    unit: 'saves',
    about: "Places travellers saved from reels that aren't on Spotto yet — leads for sign-ups.",
    gradient: { 0.2: '#c4b5fd', 0.5: '#8b5cf6', 0.8: '#6d28d9', 1: '#4c1d95' },
  },
  trips: {
    title: 'Upcoming trips',
    unit: 'planned stops',
    about: "Stops in travellers' trips over the coming days — where they're heading next.",
    gradient: { 0.2: '#bbf7d0', 0.5: '#22c55e', 0.8: '#15803d', 1: '#14532d' },
  },
};

const AREA_RADIUS_KM = 5;

export default function GeoPage() {
  return (
    <Shell>
      <Suspense>
        <GeoAnalytics />
      </Suspense>
    </Shell>
  );
}

/** The page's filters, read from and written to the URL. */
function useGeoFilters() {
  const params = useSearchParams();
  const pathname = usePathname();
  const layer = (LAYERS.some((l) => l.id === params.get('layer')) ? params.get('layer') : 'travellers') as Layer;
  const gap = (GAPS.some((g) => g.id === params.get('gap')) ? params.get('gap') : 'opportunity') as Gap;
  const days = PERIODS.includes(Number(params.get('days'))) ? Number(params.get('days')) : 30;
  const filters = {
    layer,
    gap,
    days,
    country: params.get('country') ?? '',
    style: params.get('style') ?? '',
    category: params.get('category') ?? '',
    types: (params.get('types') ?? '').split(',').filter((t) => t in ACTIVITY_LABELS),
    status: params.get('status') ?? '',
    imported: params.get('imported') === '1',
    pins: params.get('pins') === '1',
    rank: params.get('rank') === 'least' ? 'least' as const : 'most' as const,
    area: params.get('at')?.split(',').map(Number).filter((n) => !Number.isNaN(n)) ?? [],
  };
  const set = (patch: Record<string, string | null>) => {
    const next = new URLSearchParams(params.toString());
    for (const [k, v] of Object.entries(patch)) {
      if (v === null || v === '') next.delete(k);
      else next.set(k, v);
    }
    // Native history, which Next keeps useSearchParams in sync with: no
    // server round-trip per filter click, and unlike router.replace it also
    // works when the page was opened with a query string already.
    window.history.replaceState(null, '', `${pathname}?${next.toString()}`);
  };
  return { filters, set };
}

function GeoAnalytics() {
  const { filters: f, set } = useGeoFilters();
  const apiLayer = f.layer === 'gaps' ? f.gap : f.layer;
  const view = VIEWS[apiLayer];

  // Which filters mean something for the current view.
  const uses = {
    traveller: apiLayer !== 'businesses',
    category: apiLayer === 'activity' || apiLayer === 'opportunity' || apiLayer === 'businesses',
    types: apiLayer === 'activity' || apiLayer === 'opportunity',
    status: apiLayer === 'businesses',
  };

  const query = new URLSearchParams({ layer: apiLayer, days: String(f.days) });
  if (uses.traveller && f.country) query.set('country', f.country);
  if (uses.traveller && f.style) query.set('style', f.style);
  if (uses.category && f.category) query.set('category', f.category);
  if (uses.types && f.types.length) query.set('types', f.types.join(','));
  if (uses.status && f.status) query.set('status', f.status);
  if (uses.status && f.imported) query.set('imported', '1');
  const qs = query.toString();

  const { data: heat, isError: heatError, isFetching } = useQuery<AdminGeoHeat>({
    queryKey: ['admin-geo-heat', qs],
    queryFn: () => api.get<AdminGeoHeat>(`/admin/geo/heat?${qs}`),
    placeholderData: keepPreviousData,
  });
  const { data: options } = useQuery<AdminGeoFilters>({
    queryKey: ['admin-geo-filters'],
    queryFn: () => api.get<AdminGeoFilters>('/admin/geo/filters'),
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
  const { data: vendors } = useQuery<Vendor[]>({
    queryKey: ['admin-vendors-map'],
    queryFn: () => api.get<Vendor[]>('/admin/vendors/map'),
    enabled: f.pins,
  });
  const pins = f.pins ? (vendors ?? []).filter((v) => !f.category || v.category === f.category) : null;

  const area = f.area.length === 2 ? { lat: f.area[0], lng: f.area[1], radiusKm: AREA_RADIUS_KM } : null;
  const areaQuery = new URLSearchParams(qs);
  areaQuery.delete('layer');
  const { data: areaStats } = useQuery<AdminGeoArea>({
    queryKey: ['admin-geo-area', area?.lat, area?.lng, areaQuery.toString()],
    queryFn: () => api.get<AdminGeoArea>(`/admin/geo/area?lat=${area!.lat}&lng=${area!.lng}&radiusKm=${AREA_RADIUS_KM}&${areaQuery}`),
    enabled: !!area,
  });

  const towns: TownCircle[] | null = useMemo(() => {
    if (apiLayer !== 'opportunity' || !heat) return null;
    const maxValue = Math.max(1, ...heat.regions.map((r) => r.value));
    return heat.regions.filter((r) => r.value > 0 || r.businesses > 0).map((r) => {
      const tone = regionTone(r.value, r.businesses);
      return {
        id: r.id, name: r.name, lat: r.lat, lng: r.lng, size: r.value / maxValue, color: tone.ring, label: tone.label,
        detail: `${r.value} traveller ${r.value === 1 ? 'action' : 'actions'} · ${r.businesses} live ${r.businesses === 1 ? 'business' : 'businesses'}`,
      };
    });
  }, [apiLayer, heat]);

  return (
    <>
      {heatError && <InlineErrorBanner message="Couldn't load geo-analytics data — try refreshing." />}

      <FilterBar f={f} set={set} uses={uses} options={options} categories={categories} />

      <div className="grid grid-cols-[1fr_360px] gap-5 items-start mt-4">
        <div className="panel p-1 overflow-hidden relative" style={{ height: 640 }}>
          <GeoMap
            heat={apiLayer === 'opportunity' ? null : heat?.points ?? []}
            gradient={view.gradient}
            towns={towns}
            vendors={pins}
            categories={categories}
            area={area}
            onMapClick={(lat, lng) => set({ at: `${lat.toFixed(4)},${lng.toFixed(4)}` })}
          />
          <MapLegend view={apiLayer} showPins={f.pins} pins={pins} categories={categories} />
          {isFetching && (
            <span className="absolute left-14 top-3 z-[1000] rounded-lg px-2.5 py-1 text-[11.5px] font-bold"
              style={{ background: 'var(--surface)', border: '1px solid var(--line)' }}>Updating…</span>
          )}
        </div>

        <div className="space-y-4">
          <Summary heat={heat} view={apiLayer} days={f.days} />
          {area && (
            <AreaCard area={area} stats={areaStats} days={f.days} onClose={() => set({ at: null })} />
          )}
          {apiLayer === 'wanted' && heat?.wanted && <WantedList places={heat.wanted} />}
          {heat && (
            <TownRanking heat={heat} view={apiLayer} rank={f.rank} onRank={(rank) => set({ rank: rank === 'most' ? null : rank })} />
          )}
        </div>
      </div>
    </>
  );
}

function FilterBar({
  f,
  set,
  uses,
  options,
  categories,
}: {
  f: ReturnType<typeof useGeoFilters>['filters'];
  set: ReturnType<typeof useGeoFilters>['set'];
  uses: { traveller: boolean; category: boolean; types: boolean; status: boolean };
  options?: AdminGeoFilters;
  categories: Map<string, VendorCategory>;
}) {
  const bySegment = VENDOR_SEGMENTS.map((s) => ({
    ...s,
    items: [...categories.values()].filter((c) => c.segment === s.id).sort((a, b) => a.label.localeCompare(b.label)),
  })).filter((s) => s.items.length > 0);

  return (
    <div className="panel p-4 space-y-3">
      <div className="flex flex-wrap items-center gap-2">
        <Segmented
          items={LAYERS}
          value={f.layer}
          onChange={(layer) => set({ layer: layer === 'travellers' ? null : layer })}
        />
        {f.layer === 'gaps' && (
          <Segmented small items={GAPS} value={f.gap} onChange={(gap) => set({ gap: gap === 'opportunity' ? null : gap })} />
        )}
        <span className="flex-1" />
        <Segmented
          small
          items={PERIODS.map((d) => ({ id: String(d), label: f.layer === 'gaps' && f.gap === 'trips' ? `Next ${d} days` : `${d} days` }))}
          value={String(f.days)}
          onChange={(d) => set({ days: d === '30' ? null : d })}
        />
      </div>

      <div className="flex flex-wrap items-center gap-2">
        {uses.traveller && (
          <>
            <select className="input !w-auto" value={f.country} onChange={(e) => set({ country: e.target.value })}>
              <option value="">Any home country</option>
              {options?.countries.map((c) => <option key={c.value} value={c.value}>{c.value} ({c.count})</option>)}
            </select>
            {(options?.styles ?? []).map((s) => (
              <Chip key={s.value} on={f.style === s.value} onClick={() => set({ style: f.style === s.value ? null : s.value })}>
                {STYLE_LABELS[s.value] ?? s.value}
              </Chip>
            ))}
          </>
        )}
        {uses.category && (
          <select className="input !w-auto" value={f.category} onChange={(e) => set({ category: e.target.value })}>
            <option value="">All categories</option>
            {bySegment.map((s) => (
              <optgroup key={s.id} label={s.label}>
                {s.items.map((c) => <option key={c.id} value={c.id}>{c.label}</option>)}
              </optgroup>
            ))}
          </select>
        )}
        {uses.types && Object.entries(ACTIVITY_LABELS).map(([id, label]) => {
          const on = f.types.includes(id);
          return (
            <Chip key={id} on={on} onClick={() => {
              const next = on ? f.types.filter((t) => t !== id) : [...f.types, id];
              set({ types: next.join(',') });
            }}>{label}</Chip>
          );
        })}
        {uses.status && (
          <>
            <select className="input !w-auto" value={f.status} onChange={(e) => set({ status: e.target.value })}>
              {STATUSES.map((s) => <option key={s.id} value={s.id}>{s.label}</option>)}
            </select>
            <Chip on={f.imported} onClick={() => set({ imported: f.imported ? null : '1' })}>Include imported listings</Chip>
          </>
        )}
        <span className="flex-1" />
        <Chip on={f.pins} onClick={() => set({ pins: f.pins ? null : '1' })}>
          <MapPin size={12} /> Business pins
        </Chip>
      </div>
    </div>
  );
}

function Segmented<T extends string>({
  items,
  value,
  onChange,
  small,
}: {
  items: { id: T; label: string }[];
  value: T;
  onChange: (id: T) => void;
  small?: boolean;
}) {
  return (
    <div className="inline-flex rounded-xl p-1 gap-1" style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
      {items.map((it) => (
        <button key={it.id} onClick={() => onChange(it.id)}
          className={`rounded-lg font-bold ${small ? 'px-2.5 py-1 text-[12px]' : 'px-3.5 py-1.5 text-[13px]'}`}
          style={value === it.id ? { background: 'var(--accent)', color: '#fff' } : { color: 'var(--muted)' }}>
          {it.label}
        </button>
      ))}
    </div>
  );
}

function Chip({ on, onClick, children }: { on: boolean; onClick: () => void; children: ReactNode }) {
  return (
    <button onClick={onClick}
      className="flex items-center gap-1.5 rounded-full px-3 py-1.5 text-[12px] font-bold"
      style={on
        ? { background: 'color-mix(in srgb, var(--accent) 16%, transparent)', border: '1.5px solid var(--accent)', color: 'var(--accent)' }
        : { border: '1.5px solid var(--line)', color: 'var(--muted)' }}>
      {children}
    </button>
  );
}

function MapLegend({
  view,
  showPins,
  pins,
  categories,
}: {
  view: string;
  showPins: boolean;
  pins: Vendor[] | null;
  categories: Map<string, VendorCategory>;
}) {
  const v = VIEWS[view];
  const stops = Object.entries(v.gradient).sort(([a], [b]) => Number(a) - Number(b)).map(([, c]) => c);
  const hasOther = !!pins?.some((p) => pinColor(categories.get(p.category)) === OTHER_PIN_COLOR);
  return (
    <div className="absolute right-3 top-3 z-[1000] rounded-lg px-3 py-2 space-y-1.5 max-w-[calc(100%-80px)]"
      style={{ background: 'var(--surface)', border: '1px solid var(--line)', boxShadow: '0 4px 14px rgba(0,0,0,0.15)' }}>
      {view === 'opportunity' ? (
        <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
          {[regionTone(100, 1), regionTone(6, 1), regionTone(2, 1), regionTone(1, 10), regionTone(0, 1)].map((t) => (
            <span key={t.label} className="flex items-center gap-1.5 text-[11.5px] font-semibold">
              <span className="w-2.5 h-2.5 rounded-full" style={{ background: t.ring }} /> {t.label}
            </span>
          ))}
        </div>
      ) : (
        <div className="flex items-center gap-2 text-[11.5px] font-semibold">
          <span className="text-muted">Fewer</span>
          <span className="h-2.5 w-28 rounded-full" style={{ background: `linear-gradient(90deg, ${stops.join(', ')})` }} />
          <span className="text-muted">More {v.unit}</span>
        </div>
      )}
      {showPins && (
        <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
          {VENDOR_SEGMENTS.map((s) => (
            <span key={s.id} className="flex items-center gap-1.5 text-[11.5px] font-semibold">
              <span className="w-2.5 h-2.5 rounded-full" style={{ background: PIN_TONE_COLORS[s.id] }} /> {s.label}
            </span>
          ))}
          {hasOther && (
            <span className="flex items-center gap-1.5 text-[11.5px] font-semibold">
              <span className="w-2.5 h-2.5 rounded-full" style={{ background: OTHER_PIN_COLOR }} /> Other
            </span>
          )}
          <span className="flex items-center gap-1.5 text-[11.5px] font-semibold">
            <span className="w-2.5 h-2.5 rounded-full" style={{ boxShadow: '0 0 0 2px #e11d48' }} /> Flagged
          </span>
        </div>
      )}
    </div>
  );
}

/** "+12%" / "−5%" / "new" against the previous period. */
function Change({ value, previous }: { value: number; previous: number }) {
  if (previous === 0 && value === 0) return <span className="text-muted">—</span>;
  if (previous === 0) return <span className="font-bold" style={{ color: '#16a34a' }}>new</span>;
  const pct = Math.round(((value - previous) / previous) * 100);
  if (pct === 0) return <span className="text-muted font-bold">0%</span>;
  const up = pct > 0;
  const Icon = up ? ArrowUpRight : ArrowDownRight;
  return (
    <span className="inline-flex items-center font-bold" style={{ color: up ? '#16a34a' : '#e11d48' }}>
      <Icon size={13} />{up ? '+' : ''}{pct}%
    </span>
  );
}

function Summary({ heat, view, days }: { heat?: AdminGeoHeat; view: string; days: number }) {
  const v = VIEWS[view];
  const breakdown = Object.entries(heat?.breakdown ?? {}).filter(([, n]) => n > 0).sort((a, b) => b[1] - a[1]);
  return (
    <div className="panel p-5">
      <p className="font-bold text-[15px]">{v.title}</p>
      <p className="text-[12px] text-muted font-semibold mt-0.5">{v.about}</p>
      <div className="flex items-end gap-3 mt-3">
        <span className="font-display font-extrabold text-[30px] leading-none">{heat ? heat.total.toLocaleString() : '—'}</span>
        <span className="text-[12.5px] text-muted font-semibold pb-1">
          {v.unit} · {view === 'trips' ? `next ${days} days` : `last ${days} days`}
        </span>
      </div>
      {heat && view !== 'trips' && (
        <p className="text-[12.5px] font-semibold mt-1.5">
          <Change value={heat.total} previous={heat.previousTotal} />{' '}
          <span className="text-muted">vs the {days} days before ({heat.previousTotal.toLocaleString()})</span>
        </p>
      )}
      {breakdown.length > 0 && (
        <div className="flex flex-wrap gap-1.5 mt-3">
          {breakdown.map(([k, n]) => (
            <span key={k} className="rounded-full px-2.5 py-1 text-[11.5px] font-bold"
              style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
              {ACTIVITY_LABELS[k] ?? k} · {n}
            </span>
          ))}
        </div>
      )}
    </div>
  );
}

function TownRanking({
  heat,
  view,
  rank,
  onRank,
}: {
  heat: AdminGeoHeat;
  view: string;
  rank: 'most' | 'least';
  onRank: (rank: 'most' | 'least') => void;
}) {
  const opportunity = view === 'opportunity';
  const score = (r: AdminGeoTown) => (opportunity ? regionScore(r.value, r.businesses) : r.value);
  const towns = [...heat.regions]
    // Towns with no businesses and no activity say nothing about opportunity.
    .filter((r) => !opportunity || r.value > 0 || r.businesses > 0)
    .sort((a, b) => (rank === 'most' ? score(b) - score(a) : score(a) - score(b)) || a.name.localeCompare(b.name));
  return (
    <div className="panel overflow-hidden">
      <div className="px-4 py-3 border-b border-line flex items-center justify-between gap-2">
        <div>
          <p className="font-bold text-[13.5px]">Towns</p>
          <p className="text-[11.5px] text-muted font-semibold">
            {opportunity ? 'Activity per live business' : `${VIEWS[view].unit}, with change`}
          </p>
        </div>
        <Segmented small items={[{ id: 'most', label: 'Most' }, { id: 'least', label: 'Least' }]} value={rank} onChange={onRank} />
      </div>
      <div className="max-h-[360px] overflow-y-auto">
        {towns.length === 0 && <p className="px-4 py-3 text-[13px] text-muted font-semibold">No towns yet.</p>}
        {towns.map((r) => {
          const tone = regionTone(r.value, r.businesses);
          return (
            <div key={r.id} className="px-4 py-2.5 border-b border-line last:border-0 flex items-center justify-between gap-3">
              <span className="min-w-0">
                <span className="block font-bold text-[13.5px] truncate">{r.name}</span>
                <span className="block text-[11.5px] font-semibold" style={opportunity ? { color: tone.ring } : undefined}>
                  {opportunity
                    ? tone.label
                    : <span className="text-muted">{r.businesses} live {r.businesses === 1 ? 'business' : 'businesses'}</span>}
                </span>
              </span>
              <span className="text-right shrink-0">
                <span className="block font-bold text-[13px]">
                  {r.hidden ? '<3' : opportunity ? `${r.value} / ${r.businesses}` : r.value.toLocaleString()}
                </span>
                {!opportunity && view !== 'trips' && !r.hidden && (
                  <span className="block text-[11.5px]"><Change value={r.value} previous={r.previous} /></span>
                )}
              </span>
            </div>
          );
        })}
      </div>
    </div>
  );
}

function WantedList({ places }: { places: NonNullable<AdminGeoHeat['wanted']> }) {
  return (
    <div className="panel overflow-hidden">
      <div className="px-4 py-3 border-b border-line">
        <p className="font-bold text-[13.5px]">Most wanted, not on Spotto</p>
        <p className="text-[11.5px] text-muted font-semibold">Saved from reels by travellers — sign-up leads.</p>
      </div>
      <div className="max-h-[300px] overflow-y-auto">
        {places.length === 0 && <p className="px-4 py-3 text-[13px] text-muted font-semibold">Nothing saved in this period.</p>}
        {places.map((p) => (
          <div key={p.id} className="px-4 py-2.5 border-b border-line last:border-0 flex items-center justify-between gap-3">
            <span className="min-w-0">
              <span className="block font-bold text-[13.5px] truncate">{p.name}</span>
              <span className="block text-[11.5px] text-muted font-semibold truncate">
                {[p.city, p.kind].filter(Boolean).join(' · ')} · saved {timeAgo(p.lastSaved)}
              </span>
            </span>
            <span className="flex items-center gap-2 shrink-0">
              <span className="font-bold text-[13px]">{p.savers}</span>
              {p.mapsUrl && (
                <a href={p.mapsUrl} target="_blank" rel="noreferrer" className="text-accent" aria-label="Open in Google Maps">
                  <ExternalLink size={14} />
                </a>
              )}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

function AreaCard({
  area,
  stats,
  days,
  onClose,
}: {
  area: { lat: number; lng: number; radiusKm: number };
  stats?: AdminGeoArea;
  days: number;
  onClose: () => void;
}) {
  const activityTotal = Object.values(stats?.activity ?? {}).reduce((a, b) => a + b, 0);
  const rows: [string, ReactNode][] = stats ? [
    ['Travellers', stats.travellersHidden ? '<3' : stats.travellers],
    ['Traveller actions', activityTotal],
    ['Live businesses', stats.liveBusinesses],
    ['New sign-ups', stats.newBusinesses],
    ['Wanted places', stats.wantedPlaces],
    [`Trip stops, next ${days} days`, stats.upcomingStops],
  ] : [];
  return (
    <div className="panel p-5">
      <div className="flex items-center justify-between mb-1">
        <h2 className="font-bold text-[15px] flex items-center gap-2"><MapPin size={16} /> This area</h2>
        <button className="btn !p-1.5" onClick={onClose} aria-label="Close"><X size={14} /></button>
      </div>
      <p className="text-[12px] text-muted font-semibold mb-3">
        {area.lat.toFixed(4)}, {area.lng.toFixed(4)} · {area.radiusKm} km around · last {days} days
      </p>
      {!stats && <p className="text-[13px] text-muted font-semibold">Loading…</p>}
      <div className="space-y-1.5">
        {rows.map(([label, value]) => (
          <div key={label} className="flex justify-between text-[13px]">
            <span className="text-muted font-semibold">{label}</span>
            <span className="font-bold">{value}</span>
          </div>
        ))}
      </div>
      {activityTotal > 0 && (
        <div className="flex flex-wrap gap-1.5 mt-3">
          {Object.entries(stats!.activity).filter(([, n]) => n > 0).map(([k, n]) => (
            <span key={k} className="rounded-full px-2.5 py-1 text-[11px] font-bold"
              style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
              {ACTIVITY_LABELS[k] ?? k} · {n}
            </span>
          ))}
        </div>
      )}
    </div>
  );
}
