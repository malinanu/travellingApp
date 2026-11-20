// Booking types as the vendor portal shows them.

import { Bed, Car, Clock, Compass, PartyPopper, Utensils, type LucideIcon } from 'lucide-react';

import type { BookingServiceType } from '@spotto/shared';

export const BOOKING_TYPE_INFO: Record<BookingServiceType, { label: string; icon: LucideIcon; hint: string }> = {
  table: { label: 'Table', icon: Utensils, hint: 'A reservation for a party: guests pick a time and how many are coming.' },
  session: { label: 'Session', icon: Clock, hint: 'A slot with a fixed length: a class, a treatment, a visit.' },
  experience: { label: 'Experience', icon: Compass, hint: 'Something you run: a tour, a lesson, a trip. Weekly times or set dates.' },
  event: { label: 'Event', icon: PartyPopper, hint: 'A ticketed night or day on set dates: a gig, a supper club, a festival.' },
  stay: { label: 'Stay', icon: Bed, hint: 'Nights in a room or unit: guests pick check-in and check-out.' },
  rental: { label: 'Rental', icon: Car, hint: 'Something guests hire for a while: a car, a board, a bike.' },
};

export const WEEKDAYS: { id: string; label: string }[] = [
  { id: 'monday', label: 'Mon' }, { id: 'tuesday', label: 'Tue' }, { id: 'wednesday', label: 'Wed' },
  { id: 'thursday', label: 'Thu' }, { id: 'friday', label: 'Fri' }, { id: 'saturday', label: 'Sat' },
  { id: 'sunday', label: 'Sun' },
];

/** A "datetime-local" input value read as Sri Lankan time, as ISO. */
export function colomboLocalToISO(value: string): string {
  return new Date(`${value}:00+05:30`).toISOString();
}

/** ISO instant to a "datetime-local" value in Sri Lankan time. */
export function isoToColomboLocal(iso: string): string {
  const d = new Date(new Date(iso).getTime() + 330 * 60_000);
  return d.toISOString().slice(0, 16);
}

/** Today's date in Sri Lanka, "YYYY-MM-DD". */
export function colomboToday(offsetDays = 0): string {
  const d = new Date(Date.now() + 330 * 60_000 + offsetDays * 86_400_000);
  return d.toISOString().slice(0, 10);
}
