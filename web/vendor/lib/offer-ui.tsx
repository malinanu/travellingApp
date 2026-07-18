// Shared bits between the Offers page and the offer detail drawer
// (web/vendor/components/offer-detail-drawer.tsx) — kept here rather than
// exported from app/offers/page.tsx to avoid a circular import (the page
// renders the drawer, so the drawer can't import back from the page).
import { BadgeDollarSign, Gift, Package, Pencil, Percent, LucideIcon } from 'lucide-react';

import type { OfferStatus } from '@spotto/shared';

export const TYPE_ICON: Record<string, LucideIcon> = {
  Percentage: Percent,
  Amount: BadgeDollarSign,
  Freebie: Gift,
  Bundle: Package,
};

export function StatusPill({ status }: { status: OfferStatus }) {
  if (status === 'live') {
    return (
      <span className="text-[11px] font-extrabold uppercase flex items-center gap-1 text-[#16803c]">
        <span className="w-1.5 h-1.5 rounded-full bg-[#16803c]" /> Live
      </span>
    );
  }
  if (status === 'ending') {
    return <span className="text-[11px] font-extrabold uppercase rounded-full px-2 py-0.5 bg-[#fdf0cd] text-[#a97b00]">Ending soon</span>;
  }
  if (status === 'draft') {
    return (
      <span className="text-[11px] font-extrabold uppercase rounded-full px-2 py-0.5 bg-sand text-muted flex items-center gap-1">
        <Pencil size={9} /> Draft
      </span>
    );
  }
  return <span className="text-[11px] font-extrabold uppercase text-muted">Ended</span>;
}
