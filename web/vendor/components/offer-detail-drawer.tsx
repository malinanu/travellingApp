'use client';

// Offer detail — full-screen view of a single offer's real performance,
// shown when a vendor clicks an offer card (Offers page or Dashboard's
// Active offers section). Every number here is genuine: Claimed/Redeemed
// are computed from real claims, Views from real traveller impressions
// (see server's syncVendorPlace / claims.offer_id), and the activity feed
// is that offer's actual claim/redemption history — not placeholders.

import { useQuery } from '@tanstack/react-query';
import {
  ArrowLeft, Calendar, Check, Eye, Percent, Ticket, Trash2, Zap, LucideIcon,
} from 'lucide-react';

import { Donut } from '@spotto/charts';
import { timeAgo, type Claim, type Offer, type OfferStatus } from '@spotto/shared';
import { api } from '../lib/api';
import { StatusPill, TYPE_ICON } from '../lib/offer-ui';

const STYLE_DOT: Record<string, string> = {
  'Digital Nomad': '#0b6f86',
  'Surf Traveler': '#f0682f',
  Luxury: '#c98a00',
  Backpacker: '#5a5f9e',
};

const TONE: Record<'teal' | 'sun' | 'green', { bg: string; fg: string }> = {
  teal: { bg: '#d6efec', fg: '#0b6f86' },
  sun: { bg: '#fbeec4', fg: '#c98a00' },
  green: { bg: '#e7f6ec', fg: '#16a34a' },
};

function StatTile({ icon: Icon, tone, label, value }: {
  icon: LucideIcon; tone: 'teal' | 'sun' | 'green'; label: string; value: string | number;
}) {
  const t = TONE[tone];
  return (
    <div className="card p-4 flex items-center gap-3">
      <span className="w-10 h-10 rounded-xl flex items-center justify-center shrink-0" style={{ background: t.bg, color: t.fg }}>
        <Icon size={18} />
      </span>
      <div className="min-w-0">
        <p className="text-[22px] font-extrabold leading-tight" style={{ fontVariantNumeric: 'tabular-nums' }}>{value}</p>
        <p className="text-[12px] text-muted font-semibold leading-snug">{label}</p>
      </div>
    </div>
  );
}

function initial(name: string) {
  return name.trim()[0]?.toUpperCase() ?? '?';
}

export default function OfferDetailDrawer({
  offer, onClose, onSetStatus, onDelete,
}: {
  offer: Offer | null;
  onClose: () => void;
  onSetStatus?: (status: OfferStatus) => void;
  onDelete?: () => void;
}) {
  const { data: claims = [] } = useQuery<Claim[]>({
    queryKey: ['offer-claims', offer?.id],
    queryFn: () => api.get<Claim[]>(`/vendor/offers/${offer!.id}/claims`),
    enabled: !!offer,
  });

  if (!offer) return null;
  const Icon: LucideIcon = TYPE_ICON[offer.type] ?? Percent;
  const redemptionRate = offer.claimed > 0 ? Math.round((offer.redeemed / offer.claimed) * 100) : 0;
  const accent = offer.status === 'ending' ? '#f0682f' : '#0b6f86';

  return (
    <div className="fixed inset-0 z-50 bg-cream flex flex-col">
      <header className="bg-white border-b border-line shrink-0">
        <div className="max-w-5xl mx-auto px-6 py-3 flex items-center justify-between">
          <button onClick={onClose}
            className="flex items-center gap-1.5 text-[13px] font-extrabold text-muted hover:text-ink transition-colors -ml-2 px-2 py-1.5 rounded-lg hover:bg-sand">
            <ArrowLeft size={16} /> Back to offers
          </button>
          {(onSetStatus || onDelete) && (
            <div className="flex items-center gap-2">
              {onSetStatus && offer.status === 'draft' && (
                <button onClick={() => onSetStatus('live')}
                  className="rounded-lg border border-line px-3 py-1.5 text-[12.5px] font-extrabold flex items-center gap-1.5 hover:border-accent hover:text-accent transition-colors">
                  <Zap size={13} /> Publish
                </button>
              )}
              {onSetStatus && (offer.status === 'live' || offer.status === 'ending') && (
                <button onClick={() => onSetStatus('ended')}
                  className="rounded-lg border border-line px-3 py-1.5 text-[12.5px] font-extrabold flex items-center gap-1.5 hover:border-accent hover:text-accent transition-colors">
                  <Check size={13} /> End offer
                </button>
              )}
              {onSetStatus && offer.status === 'ended' && (
                <button onClick={() => onSetStatus('live')}
                  className="rounded-lg border border-line px-3 py-1.5 text-[12.5px] font-extrabold flex items-center gap-1.5 hover:border-accent hover:text-accent transition-colors">
                  <Zap size={13} /> Relaunch
                </button>
              )}
              {onDelete && (
                <button onClick={onDelete}
                  className="rounded-lg border border-line px-3 py-1.5 text-[12.5px] font-extrabold text-coral flex items-center gap-1.5 hover:bg-[#fde4d8] hover:border-coral/40 transition-colors">
                  <Trash2 size={13} /> Delete
                </button>
              )}
            </div>
          )}
        </div>
      </header>

      <div className="flex-1 overflow-y-auto">
        <div className="max-w-5xl mx-auto px-6 py-8">
          <div className="flex items-start gap-4 mb-8">
            <span className="w-12 h-12 rounded-2xl bg-accent-tint text-accent flex items-center justify-center shrink-0">
              <Icon size={22} />
            </span>
            <div className="min-w-0">
              <h1 className="font-display font-extrabold text-[26px] leading-tight">{offer.title}</h1>
              <div className="flex items-center gap-2 mt-1.5 flex-wrap">
                <span className="text-[12px] font-extrabold rounded-full px-2.5 py-1 bg-sand">{offer.value}</span>
                <StatusPill status={offer.status} />
              </div>
            </div>
          </div>

          <div className="grid lg:grid-cols-[minmax(0,1fr)_minmax(0,1.4fr)] gap-5 mb-8">
            <div className="card p-6 flex items-center gap-6">
              <div className="relative shrink-0">
                <Donut value={offer.claimed} max={offer.limit || 1} accent={accent} size={128} stroke={12} />
                <div className="absolute inset-0 flex flex-col items-center justify-center">
                  <span className="text-[26px] font-extrabold leading-none" style={{ fontVariantNumeric: 'tabular-nums' }}>{offer.claimed}</span>
                  <span className="text-[11px] text-muted font-bold mt-0.5">claimed</span>
                </div>
              </div>
              <div className="min-w-0">
                <p className="font-extrabold text-[15px]">
                  {offer.limit > 0 ? `of ${offer.limit} available` : 'Unlimited claims'}
                </p>
                <p className="text-[12.5px] text-muted font-semibold mt-1 leading-relaxed">
                  {offer.limit > 0
                    ? `${Math.max(offer.limit - offer.claimed, 0)} spot${offer.limit - offer.claimed === 1 ? '' : 's'} left before this offer is fully claimed.`
                    : 'This offer has no claim cap — every eligible traveler can claim it.'}
                </p>
                <p className="text-[12.5px] text-muted font-semibold mt-3 flex items-center gap-1.5">
                  <Calendar size={12} />
                  {offer.status === 'draft' ? 'Schedule set on publish' : `${offer.start} → ${offer.end}`}
                </p>
              </div>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
              <StatTile icon={Ticket} tone="teal" label="Total redemptions" value={offer.redeemed} />
              <StatTile icon={Eye} tone="sun" label="Views" value={offer.views} />
              <StatTile icon={Check} tone="green" label="Redemption rate" value={`${redemptionRate}%`} />
            </div>
          </div>

          <div className="card p-6">
            <div className="flex items-center justify-between mb-4">
              <h2 className="font-extrabold text-[15px]">Recent activity</h2>
              {claims.length > 0 && <span className="text-[12px] text-muted font-semibold">{claims.length} total</span>}
            </div>

            {claims.length === 0 ? (
              <div className="flex flex-col items-center text-center py-12">
                <span className="w-11 h-11 rounded-full bg-sand text-muted flex items-center justify-center mb-3">
                  <Ticket size={19} />
                </span>
                <p className="font-bold text-[14px]">No claims yet</p>
                <p className="text-[12.5px] text-muted font-semibold mt-1 max-w-xs">
                  Activity shows up here the moment a traveler claims or redeems this offer.
                </p>
              </div>
            ) : (
              <div>
                {claims.map((c, i) => {
                  const dotColor = c.travellerStyle ? STYLE_DOT[c.travellerStyle] ?? '#6c8582' : null;
                  const redeemed = c.status === 'redeemed';
                  return (
                    <div key={c.id} className={`flex items-center gap-3 py-3 ${i ? 'border-t border-line' : ''}`}>
                      <span className="w-9 h-9 rounded-full bg-sand text-ink flex items-center justify-center text-[12px] font-extrabold shrink-0">
                        {initial(c.travellerName || 'T')}
                      </span>
                      <div className="min-w-0 flex-1">
                        <p className="font-bold text-[13.5px] truncate">
                          {c.travellerName || 'Traveller'}{c.travellerCountry ? ` (${c.travellerCountry})` : ''}
                        </p>
                        {dotColor && (
                          <p className="text-[11px] font-bold mt-0.5 flex items-center gap-1" style={{ color: dotColor }}>
                            <span className="w-1.5 h-1.5 rounded-full" style={{ background: dotColor }} /> {c.travellerStyle}
                          </p>
                        )}
                      </div>
                      <div className="text-right shrink-0">
                        <span className={`text-[11px] font-extrabold uppercase rounded-full px-2 py-0.5 ${
                          redeemed ? 'bg-[#e7f6ec] text-[#16803c]' : 'bg-sand text-muted'
                        }`}>
                          {redeemed ? 'Redeemed' : 'Claimed'}
                        </span>
                        <p className="text-[11px] text-muted font-semibold mt-1">
                          {timeAgo(redeemed ? c.redeemedAt : c.claimedAt)}
                        </p>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
