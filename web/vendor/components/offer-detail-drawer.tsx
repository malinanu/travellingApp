'use client';

// Offer detail drawer — shown when a vendor clicks an offer card (Offers
// page or Dashboard's Active offers section). Shared by both since offer
// stats are now genuinely live (see server's syncVendorPlace / claims.offer_id):
// Claimed/Redeemed are computed from real claims, Views from real
// traveller impressions, and the activity list below is that offer's
// actual claim/redemption history.

import { useQuery } from '@tanstack/react-query';
import { Eye, Tag, Ticket, X, LucideIcon } from 'lucide-react';

import { timeAgo, type Claim, type Offer } from '@spotto/shared';
import { api } from '../lib/api';
import { StatusPill, TYPE_ICON } from '../lib/offer-ui';

function Stat({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="rounded-xl bg-sand p-3">
      <p className="text-[20px] font-extrabold leading-tight">{value}</p>
      <p className="text-[12px] text-muted font-semibold">{label}</p>
    </div>
  );
}

function initial(name: string) {
  return name.trim()[0]?.toUpperCase() ?? '?';
}

export default function OfferDetailDrawer({ offer, onClose }: { offer: Offer | null; onClose: () => void }) {
  const { data: claims = [] } = useQuery<Claim[]>({
    queryKey: ['offer-claims', offer?.id],
    queryFn: () => api.get<Claim[]>(`/vendor/offers/${offer!.id}/claims`),
    enabled: !!offer,
  });

  if (!offer) return null;
  const Icon: LucideIcon = TYPE_ICON[offer.type] ?? Tag;
  const redemptionRate = offer.claimed > 0 ? Math.round((offer.redeemed / offer.claimed) * 100) : 0;

  return (
    <div className="fixed inset-0 z-50 bg-white flex flex-col">
      <div className="flex items-start gap-3 p-5 border-b border-line">
        <span className="w-10 h-10 rounded-xl bg-accent-tint text-accent flex items-center justify-center shrink-0">
          <Icon size={19} />
        </span>
        <div className="flex-1 min-w-0">
          <p className="font-extrabold text-[16px] truncate">{offer.title}</p>
          <div className="flex items-center gap-2 mt-1 flex-wrap">
            <span className="text-[11px] font-extrabold rounded-full px-2 py-0.5 bg-sand flex items-center gap-1">
              <Tag size={10} /> {offer.value}
            </span>
            <StatusPill status={offer.status} />
          </div>
        </div>
        <button onClick={onClose} className="w-8 h-8 rounded-lg flex items-center justify-center text-muted hover:bg-sand shrink-0">
          <X size={18} />
        </button>
      </div>

      <div className="flex-1 overflow-y-auto p-5">
        <div className="max-w-2xl mx-auto">
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-2.5 mb-2.5">
            <Stat label="Total redemptions" value={offer.redeemed} />
            <Stat label="Views" value={offer.views} />
            <Stat label={offer.limit > 0 ? `Claimed of ${offer.limit}` : 'Claimed (unlimited)'} value={offer.claimed} />
            <Stat label="Redemption rate" value={`${redemptionRate}%`} />
          </div>
          <p className="text-[12px] text-muted font-semibold mb-5">{offer.start} → {offer.end}</p>

          <h3 className="font-extrabold text-[14px] mb-2.5">Recent activity</h3>
          {claims.length === 0 && (
            <p className="text-[13px] text-muted font-semibold py-4">
              No claims yet — activity will show up here as travelers claim and redeem this offer.
            </p>
          )}
          <div className="space-y-1">
            {claims.map((c) => (
              <div key={c.id} className="flex items-center gap-3 py-2.5 border-t border-line">
                <span className="w-8 h-8 rounded-full bg-sand text-ink flex items-center justify-center text-[11px] font-extrabold shrink-0">
                  {initial(c.travellerName || 'T')}
                </span>
                <div className="min-w-0 flex-1">
                  <p className="font-bold text-[13.5px] truncate">
                    {c.travellerName || 'Traveller'}{c.travellerCountry ? ` (${c.travellerCountry})` : ''}
                  </p>
                  <p className="text-[12px] text-muted font-semibold flex items-center gap-1">
                    {c.status === 'redeemed' ? <Ticket size={11} /> : <Eye size={11} />}
                    {c.status === 'redeemed' ? 'Redeemed' : 'Claimed'} {timeAgo(c.status === 'redeemed' ? c.redeemedAt : c.claimedAt)}
                  </p>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
