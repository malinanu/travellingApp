'use client';

// Bookings across every venue, host and event.

import Shell from '../../components/shell';
import { BookingsTable } from '../../components/bookings-table';
import { PageHeader } from '../../lib/money-ui';

export default function AdminBookingsPage() {
  return (
    <Shell>
      <PageHeader title="Bookings" sub="Every booking across venues, hosts and events. Times are Sri Lankan time." />
      <BookingsTable />
    </Shell>
  );
}
