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
      <span className="badge uppercase text-[#16803c] bg-emerald-500/10 border border-emerald-500/20">
        <span className="w-1.5 h-1.5 rounded-full bg-[#16803c]" /> Live
      </span>
    );
  }
  if (status === 'ending') {
    return <span className="badge uppercase bg-[#fdf0cd] text-[#a97b00] border border-amber-300/40">Ending soon</span>;
  }
  if (status === 'draft') {
    return (
      <span className="badge uppercase bg-sand text-muted border border-line">
        <Pencil size={9} /> Draft
      </span>
    );
  }
  return <span className="badge uppercase bg-black/5 dark:bg-white/5 text-muted border border-line">Ended</span>;
}
