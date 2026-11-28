'use client';

// Earnings & payouts: what Spotto owes the business (or the business owes
// Spotto in per-guest fees), the weekly payout, bank details and a
// downloadable statement of every line.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Banknote, CalendarClock, Download, Landmark, Receipt, ShieldCheck } from 'lucide-react';
import { useState } from 'react';

import {
  DataTable, EmptyState, Money, StatusPill, downloadAuthed, formatLocalDate, formatLocalDateTime, formatMinor,
  type LedgerEntry, type PayoutAccount, type VendorEarnings,
} from '@spotto/shared';
import Shell, { RoleLockedCard, useMe } from '../../components/shell';
import { api } from '../../lib/api';
import { colomboToday } from '../../lib/booking-ui';

const KIND_LABEL: Record<LedgerEntry['kind'], string> = {
  booking_net: 'Booking paid (your share)',
  per_guest_fee: 'Per-guest fee',
  refund_reversal: 'Refund to guest (your share)',
  adjustment: 'Adjustment',
};

export default function EarningsPage() {
  const { data: me } = useMe();
  const isStaff = me?.vendorRole === 'staff';
  const isOwner = !me?.vendorRole || me.vendorRole === 'owner';
  const [from, setFrom] = useState(colomboToday(-30));
  const [to, setTo] = useState(colomboToday());
  const [downloading, setDownloading] = useState(false);

  const { data: earnings, isLoading } = useQuery<VendorEarnings>({
    queryKey: ['vendor-earnings'],
    queryFn: () => api.get<VendorEarnings>('/vendor/earnings'),
    enabled: !isStaff,
  });
  const { data: statement, isLoading: loadingStatement } = useQuery<{ entries: LedgerEntry[] }>({
    queryKey: ['vendor-statement', from, to],
    queryFn: () => api.get(`/vendor/statement?from=${from}&to=${to}`),
    enabled: !isStaff,
  });

  if (isStaff) return <Shell><RoleLockedCard what="Earnings" /></Shell>;

  const owed = earnings?.owedToSpottoMinor ?? 0;
  return (
    <Shell>
      <div className="mb-5">
        <h2 className="font-display font-extrabold text-[24px]">Earnings & payouts</h2>
        <p className="text-muted font-semibold text-[13.5px] max-w-2xl">
          Travellers pay in the Spotto app. Once a booking has taken place, your share becomes payable and is paid to your bank every Monday
          (from {formatMinor(earnings?.minPayoutMinor ?? 100000)}). Per-guest fees on free table bookings are taken off the same payout.
        </p>
      </div>

      <div className="grid sm:grid-cols-3 gap-3 mb-5">
        <Stat icon={<Banknote size={18} />} label={owed > 0 ? 'You owe Spotto' : 'Ready to pay out'} loading={isLoading}
          value={<Money minor={owed > 0 ? owed : earnings?.payableMinor} className={owed > 0 ? 'text-rose-600' : ''} />}
          hint={owed > 0 ? 'Per-guest fees or refunds are larger than what’s due to you; they come off future payouts.' : 'From bookings that have already happened.'} />
        <Stat icon={<CalendarClock size={18} />} label="Upcoming" loading={isLoading} value={<Money minor={earnings?.pendingMinor} />}
          hint="From paid bookings still to come." />
        <Stat icon={<Landmark size={18} />} label="Next payout" loading={isLoading}
          value={earnings ? formatLocalDate(earnings.nextPayoutAt) : ''}
          hint={earnings?.payoutAccount.set ? (earnings.payoutAccount.verified ? `To ${earnings.payoutAccount.bankName} ••${earnings.payoutAccount.accountLast4}` : 'Bank details waiting for Spotto to verify.') : 'Add your bank details to get paid.'} />
      </div>

      <div className="grid lg:grid-cols-[1fr_360px] gap-4 items-start">
        <div className="space-y-4">
          <div className="card">
            <div className="flex flex-wrap items-center gap-2 p-4 border-b border-line">
              <h3 className="font-extrabold text-[15px] flex items-center gap-2 mr-auto"><Receipt size={17} /> Statement</h3>
              <input type="date" className="input !py-1.5 !w-auto" value={from} max={to} onChange={(e) => e.target.value && setFrom(e.target.value)} />
              <span className="text-muted">–</span>
              <input type="date" className="input !py-1.5 !w-auto" value={to} min={from} onChange={(e) => e.target.value && setTo(e.target.value)} />
              <button className="btn btn-secondary" disabled={downloading} onClick={async () => {
                setDownloading(true);
                try { await downloadAuthed(api, `/vendor/statement?from=${from}&to=${to}&format=csv`, `spotto-statement-${from}-${to}.csv`); } finally { setDownloading(false); }
              }}>
                <Download size={15} /> CSV
              </button>
            </div>
            <DataTable
              loading={loadingStatement}
              rows={statement?.entries ?? []}
              rowKey={(e) => e.id}
              empty={<EmptyState title="Nothing in this period" body="Paid bookings, fees and refunds show up here." />}
              columns={[
                { key: 'date', header: 'Date', sortValue: (e) => e.createdAt, render: (e) => <span className="whitespace-nowrap">{formatLocalDateTime(e.createdAt)}</span> },
                { key: 'kind', header: 'What', render: (e) => <span><span className="font-bold">{KIND_LABEL[e.kind]}</span>{e.serviceName ? <span className="text-muted"> · {e.serviceName}</span> : null}</span> },
                { key: 'booking', header: 'Booking', render: (e) => <span className="font-mono text-[12px]">{e.bookingCode || '—'}</span> },
                { key: 'amount', header: 'Amount', align: 'right', sortValue: (e) => e.amountMinor, render: (e) => <Money minor={e.amountMinor} signed /> },
                { key: 'settled', header: '', render: (e) => (e.payoutId ? <StatusPill status="paid" /> : <span className="text-[12px] text-muted font-semibold">Unsettled</span>) },
              ]}
            />
          </div>

          <div className="card">
            <h3 className="font-extrabold text-[15px] p-4 border-b border-line">Payouts</h3>
            <DataTable
              loading={isLoading}
              rows={earnings?.payouts ?? []}
              rowKey={(p) => p.id}
              empty={<EmptyState title="No payouts yet" body="Your first payout is drafted on the Monday after your first booking takes place." />}
              columns={[
                { key: 'date', header: 'Created', render: (p) => formatLocalDate(p.createdAt) },
                { key: 'amount', header: 'Amount', align: 'right', render: (p) => <Money minor={p.amountMinor} /> },
                { key: 'status', header: 'Status', render: (p) => <StatusPill status={p.status} /> },
                { key: 'ref', header: 'Bank reference', render: (p) => p.bankReference || '—' },
              ]}
            />
          </div>
        </div>

        <BankDetails canEdit={isOwner} />
      </div>
    </Shell>
  );
}

function Stat({ icon, label, value, hint, loading }: { icon: React.ReactNode; label: string; value: React.ReactNode; hint: string; loading?: boolean }) {
  return (
    <div className="card p-4">
      <p className="flex items-center gap-2 text-[12.5px] font-extrabold text-muted">{icon}{label}</p>
      <div className="mt-1.5 text-[22px] font-display font-extrabold">{loading ? <span className="inline-block h-6 w-28 rounded bg-sand animate-pulse" /> : value}</div>
      <p className="mt-1 text-[12px] font-semibold text-muted">{hint}</p>
    </div>
  );
}

function BankDetails({ canEdit }: { canEdit: boolean }) {
  const qc = useQueryClient();
  const { data: account } = useQuery<PayoutAccount>({
    queryKey: ['payout-account'],
    queryFn: () => api.get<PayoutAccount>('/vendor/payout-account'),
  });
  const [editing, setEditing] = useState(false);
  const [form, setForm] = useState({ accountHolder: '', bankName: '', branch: '', accountNumber: '' });
  const [error, setError] = useState<string | null>(null);
  const save = useMutation({
    mutationFn: () => api.put('/vendor/payout-account', form),
    onSuccess: () => {
      setEditing(false);
      setForm({ accountHolder: '', bankName: '', branch: '', accountNumber: '' });
      qc.invalidateQueries({ queryKey: ['payout-account'] });
      qc.invalidateQueries({ queryKey: ['vendor-earnings'] });
    },
    onError: (e) => setError(e instanceof Error ? e.message : 'Couldn’t save'),
  });

  return (
    <div className="card p-4">
      <h3 className="font-extrabold text-[15px] flex items-center gap-2"><Landmark size={17} /> Bank details</h3>
      {account?.set && !editing ? (
        <div className="mt-3 text-[13.5px] font-semibold space-y-1">
          <p className="font-extrabold">{account.accountHolder}</p>
          <p>{account.bankName}{account.branch ? `, ${account.branch}` : ''}</p>
          <p className="font-mono">•••• {account.accountLast4}</p>
          <p className={`flex items-center gap-1.5 text-[12.5px] font-extrabold ${account.verified ? 'text-emerald-600' : 'text-amber-700'}`}>
            <ShieldCheck size={14} /> {account.verified ? 'Verified by Spotto' : 'Waiting for Spotto to verify'}
          </p>
          {canEdit && <button className="btn btn-secondary mt-2" onClick={() => setEditing(true)}>Change</button>}
        </div>
      ) : canEdit ? (
        <div className="mt-3 space-y-2.5">
          {!account?.set && <p className="text-[12.5px] text-muted font-semibold">Where should we send your money? Only the owner can change this.</p>}
          {account?.set && <p className="text-[12.5px] text-amber-700 font-semibold">Changing your bank details pauses payouts until Spotto verifies the new account.</p>}
          {(['accountHolder', 'bankName', 'branch', 'accountNumber'] as const).map((k) => (
            <label key={k} className="block">
              <span className="block text-[12px] font-extrabold mb-1">{{ accountHolder: 'Account holder', bankName: 'Bank', branch: 'Branch', accountNumber: 'Account number' }[k]}</span>
              <input className="input" value={form[k]} inputMode={k === 'accountNumber' ? 'numeric' : undefined} autoComplete="off"
                onChange={(e) => setForm({ ...form, [k]: e.target.value })} />
            </label>
          ))}
          {error && <p className="text-[12.5px] text-rose-600 font-bold">{error}</p>}
          <div className="flex gap-2 justify-end">
            {account?.set && <button className="btn btn-secondary" onClick={() => setEditing(false)}>Cancel</button>}
            <button className="btn btn-accent" disabled={save.isPending || !form.accountHolder || !form.bankName || !form.accountNumber}
              onClick={() => { setError(null); save.mutate(); }}>
              {save.isPending ? 'Saving…' : 'Save bank details'}
            </button>
          </div>
        </div>
      ) : (
        <p className="mt-3 text-[12.5px] text-muted font-semibold">The business owner adds the bank account payouts go to.</p>
      )}
    </div>
  );
}
