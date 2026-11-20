'use client';

// Payouts: every Monday the server drafts one per vendor whose payable
// balance reaches the minimum and whose bank details are verified. Finance
// approves them, exports the bank file, transfers, then marks each paid
// with its reference. A failed payout releases its lines for next time.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Download, Play, ShieldCheck } from 'lucide-react';
import { useState } from 'react';

import {
  DataTable, DetailList, Drawer, EmptyState, Money, StatusPill, downloadAuthed, formatLocalDate, formatLocalDateTime, formatMinor, useConfirm,
  type LedgerEntry, type Payout, type PayoutAccount, type PayoutStatus,
} from '@spotto/shared';
import Shell, { hasAdminPerm, useMe } from '../../components/shell';
import { api } from '../../lib/api';
import { PageHeader, Tabs } from '../../lib/money-ui';

type RunResult = { created: Payout[]; noAccount: string[]; belowMinimum: string[] };

export default function PayoutsPage() {
  const qc = useQueryClient();
  const { data: me } = useMe();
  const canPay = hasAdminPerm(me, 'finance.payouts');
  const [status, setStatus] = useState<PayoutStatus | ''>('draft');
  const [selected, setSelected] = useState<Payout | null>(null);
  const [lastRun, setLastRun] = useState<RunResult | null>(null);
  const [confirm, dialog] = useConfirm();
  const { data = [], isLoading } = useQuery<Payout[]>({
    queryKey: ['admin-payouts', status],
    queryFn: () => api.get<Payout[]>(`/admin/payouts${status ? `?status=${status}` : ''}`),
  });
  const run = useMutation({
    mutationFn: () => api.post<RunResult>('/admin/payouts/run'),
    onSuccess: (r) => { setLastRun(r); qc.invalidateQueries({ queryKey: ['admin-payouts'] }); },
  });

  return (
    <Shell>
      {dialog}
      <PageHeader
        title="Payouts"
        sub="Drafted every Monday 03:00 (Sri Lanka) for vendors with at least LKR 1,000 payable and verified bank details. Approve, export the bank file, transfer, then mark paid."
        actions={canPay && (
          <>
            <button className="btn" onClick={() => downloadAuthed(api, '/admin/payouts-export.csv?status=approved', 'spotto-payouts-approved.csv')}>
              <Download size={14} /> Bank file (approved)
            </button>
            <button className="btn btn-primary" disabled={run.isPending} onClick={async () => {
              if (await confirm({ title: 'Draft payouts now?', body: 'Does what Monday’s run does, for everything payable right now.', confirmLabel: 'Draft payouts' })) run.mutate();
            }}>
              <Play size={14} /> Run now
            </button>
          </>
        )}
      />
      {lastRun && (
        <div className="panel p-3 mb-4 text-[13px] font-medium">
          Drafted {lastRun.created.length} payout(s).
          {lastRun.noAccount.length > 0 && <> {lastRun.noAccount.length} vendor(s) have money due but no verified bank details: {lastRun.noAccount.join(', ')}.</>}
          {lastRun.belowMinimum.length > 0 && <> {lastRun.belowMinimum.length} below the minimum or owing Spotto.</>}
        </div>
      )}
      <div className="mb-3">
        <Tabs value={status} onChange={setStatus} options={[
          { id: 'draft', label: 'Draft' }, { id: 'approved', label: 'Approved' }, { id: 'paid', label: 'Paid' },
          { id: 'failed', label: 'Failed' }, { id: '', label: 'All' },
        ]} />
      </div>
      <div className="panel overflow-hidden">
        <DataTable
          loading={isLoading}
          rows={data}
          rowKey={(p) => p.id}
          onRowClick={setSelected}
          empty={<EmptyState title="No payouts here" />}
          columns={[
            { key: 'created', header: 'Drafted', sortValue: (p) => p.createdAt, render: (p) => formatLocalDate(p.createdAt) },
            { key: 'vendor', header: 'Vendor', sortValue: (p) => p.vendorName, render: (p) => <span className="font-semibold">{p.vendorName}</span> },
            { key: 'bank', header: 'Bank', render: (p) => (p.bankName ? `${p.bankName} ••${p.accountLast4}` : '—') },
            { key: 'lines', header: 'Lines', align: 'right', render: (p) => p.entryCount },
            { key: 'amount', header: 'Amount', align: 'right', sortValue: (p) => p.amountMinor, render: (p) => <Money minor={p.amountMinor} /> },
            { key: 'status', header: 'Status', render: (p) => <StatusPill status={p.status} /> },
          ]}
        />
      </div>
      {selected && <PayoutDrawer payout={selected} canPay={canPay} onClose={() => setSelected(null)} />}
    </Shell>
  );
}

function PayoutDrawer({ payout, canPay, onClose }: { payout: Payout; canPay: boolean; onClose: () => void }) {
  const qc = useQueryClient();
  const [confirm, dialog] = useConfirm();
  const [error, setError] = useState<string | null>(null);
  const { data } = useQuery<{ payout: Payout; entries: LedgerEntry[] }>({
    queryKey: ['admin-payout', payout.id],
    queryFn: () => api.get(`/admin/payouts/${payout.id}`),
  });
  const { data: account, refetch: loadAccount, isFetching: loadingAccount } = useQuery<PayoutAccount>({
    queryKey: ['admin-payout-account', payout.vendorId],
    queryFn: () => api.get<PayoutAccount>(`/admin/vendors/${payout.vendorId}/payout-account`),
    enabled: false,
  });
  const p = data?.payout ?? payout;
  const act = useMutation({
    mutationFn: ({ action, body }: { action: string; body?: object }) => api.post(`/admin/payouts/${p.id}/${action}`, body ?? {}),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['admin-payouts'] }); onClose(); },
    onError: (e) => setError(e instanceof Error ? e.message : 'Couldn’t do that'),
  });
  const verify = useMutation({
    mutationFn: () => api.post(`/admin/vendors/${p.vendorId}/payout-account/verify`),
    onSuccess: () => loadAccount(),
  });

  return (
    <Drawer open onClose={onClose} width={560} title={`${p.vendorName} · ${formatMinor(p.amountMinor)}`} subtitle={<StatusPill status={p.status} />}
      footer={canPay && (p.status === 'draft' || p.status === 'approved') ? (
        <>
          {error && <span className="mr-auto text-[12.5px] font-semibold text-rose-600 self-center">{error}</span>}
          <button className="btn btn-danger-outline" onClick={async () => {
            const note = await confirm({ title: 'Mark this payout failed?', body: 'Its lines go back to the vendor’s balance for the next run.', reasonLabel: 'What happened', confirmLabel: 'Mark failed', tone: 'danger' });
            if (typeof note === 'string') act.mutate({ action: 'fail', body: { note } });
          }}>Failed</button>
          {p.status === 'draft' && (
            <button className="btn btn-primary" onClick={async () => {
              if (await confirm({ title: `Approve ${formatMinor(p.amountMinor)} to ${p.vendorName}?`, body: 'It goes into the bank file for transfer.', confirmLabel: 'Approve' })) act.mutate({ action: 'approve' });
            }}>Approve</button>
          )}
          {p.status === 'approved' && (
            <button className="btn btn-primary" onClick={async () => {
              const ref = await confirm({ title: 'Mark as paid?', body: `Only once ${formatMinor(p.amountMinor)} has left the bank.`, reasonLabel: 'Bank transfer reference', confirmLabel: 'Mark paid' });
              if (typeof ref === 'string') act.mutate({ action: 'mark-paid', body: { bankReference: ref } });
            }}>Mark paid</button>
          )}
        </>
      ) : undefined}>
      {dialog}
      <DetailList rows={[
        ['Drafted', formatLocalDateTime(p.createdAt)],
        ['Covers lines up to', formatLocalDateTime(p.periodEnd)],
        ...(p.approvedAt ? [['Approved', `${formatLocalDateTime(p.approvedAt)} by ${p.approvedBy}`] as [string, string]] : []),
        ...(p.paidAt ? [['Paid', `${formatLocalDateTime(p.paidAt)} · ref ${p.bankReference}`] as [string, string]] : []),
        ...(p.note ? [['Note', p.note] as [string, string]] : []),
        ['Bank', p.bankName ? `${p.accountHolder} · ${p.bankName}${p.branch ? `, ${p.branch}` : ''} · ••${p.accountLast4}` : 'No bank details'],
      ]} />
      {canPay && (
        <div className="mt-3 rounded-xl border border-line p-3 text-[13px]">
          {account ? (
            <div className="flex flex-wrap items-center gap-2">
              <span className="font-mono">{account.accountNumber || 'Unavailable (no key)'}</span>
              {account.verifiedAt ? (
                <span className="inline-flex items-center gap-1 text-emerald-600 font-semibold"><ShieldCheck size={14} /> Verified</span>
              ) : (
                <button className="btn" disabled={verify.isPending} onClick={async () => {
                  if (await confirm({ title: 'Mark these bank details verified?', body: 'Only after checking them against the vendor’s documents.', confirmLabel: 'Verified' })) verify.mutate();
                }}>Mark verified</button>
              )}
            </div>
          ) : (
            <button className="btn" disabled={loadingAccount} onClick={() => loadAccount()}>Show full account number (logged)</button>
          )}
        </div>
      )}
      <h4 className="mt-5 mb-1 text-[12px] font-bold uppercase text-muted">Lines</h4>
      <DataTable rows={data?.entries ?? []} rowKey={(e) => e.id} empty={<p className="text-[13px] text-muted py-2">No lines.</p>}
        columns={[
          { key: 'when', header: 'Date', render: (e) => formatLocalDate(e.createdAt) },
          { key: 'kind', header: 'Line', render: (e) => <span className="capitalize">{e.kind.replace(/_/g, ' ')}</span> },
          { key: 'booking', header: 'Booking', render: (e) => <span className="font-mono text-[12px]">{e.bookingCode || '—'}</span> },
          { key: 'amt', header: 'Amount', align: 'right', render: (e) => <Money minor={e.amountMinor} signed /> },
        ]} />
    </Drawer>
  );
}
