'use client';

import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  AlertCircle,
  Award,
  Calendar,
  Check,
  CheckCircle2,
  ChevronRight,
  Clock,
  ExternalLink,
  Filter,
  Gift,
  HelpCircle,
  MapPin,
  Search,
  Sparkles,
  Store,
  Tag,
  TrendingUp,
  Undo2,
  Users,
  X,
  XCircle,
} from 'lucide-react';
import Link from 'next/link';
import { useEffect, useMemo, useState } from 'react';
import { timeAgo, type QuestCompletion, type QuestSchedule, type VendorSponsorship } from '@spotto/shared';
import Shell from '../../components/shell';
import { InlineErrorBanner } from '../../components/route-status';
import { api } from '../../lib/api';

type Tab = 'queue' | 'approved' | 'scheduler' | 'completions';

export default function QuestsPage() {
  const [tab, setTab] = useState<Tab>('queue');
  const [prefilledVendor, setPrefilledVendor] = useState<string>('');
  const [prefilledSponsorship, setPrefilledSponsorship] = useState<string>('');
  const [searchFilter, setSearchFilter] = useState('');
  const qc = useQueryClient();

  const { data: pending = [], isLoading: pendingLoading } = useQuery<VendorSponsorship[]>({
    queryKey: ['admin-quests', 'pending'],
    queryFn: () => api.get<VendorSponsorship[]>('/admin/quests/sponsorships/pending'),
  });

  const { data: approved = [], isLoading: approvedLoading } = useQuery<VendorSponsorship[]>({
    queryKey: ['admin-quests', 'approved'],
    queryFn: () => api.get<VendorSponsorship[]>('/admin/quests/sponsorships/approved'),
  });

  const { data: rejected = [], isLoading: rejectedLoading } = useQuery<VendorSponsorship[]>({
    queryKey: ['admin-quests', 'rejected'],
    queryFn: () => api.get<VendorSponsorship[]>('/admin/quests/sponsorships?status=rejected'),
  });

  const { data: schedules = [], isLoading: schedulesLoading } = useQuery<QuestSchedule[]>({
    queryKey: ['admin-quests', 'schedules'],
    queryFn: () => api.get<QuestSchedule[]>('/admin/quests/schedules'),
  });

  const act = useMutation({
    mutationFn: ({ id, status }: { id: string; status: 'approved' | 'rejected' | 'pending' }) =>
      api.post(`/admin/quests/sponsorships/${id}/status`, { status }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin-quests'] });
    },
  });

  const totalCompletions = useMemo(
    () => schedules.reduce((acc, s) => acc + (s.completions || 0), 0),
    [schedules]
  );
  const totalRedemptions = useMemo(
    () => schedules.reduce((acc, s) => acc + (s.redemptions || 0), 0),
    [schedules]
  );

  const handleScheduleSponsor = (vendorId: string, sponsorshipId?: string) => {
    setPrefilledVendor(vendorId);
    setPrefilledSponsorship(sponsorshipId || '');
    setTab('scheduler');
  };

  return (
    <Shell>
      <div className="space-y-6 max-w-6xl pb-12">
        {/* Page Header */}
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div>
            <div className="flex items-center gap-2.5">
              <h1 className="text-[22px] font-bold tracking-tight text-ink">Daily Quest Sponsorships</h1>
              <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-[11px] font-bold bg-accent/10 text-accent border border-accent/20">
                <Sparkles size={11} /> Platform Quests
              </span>
            </div>
            <p className="text-[13px] text-muted font-medium mt-1">
              Review vendor reward proposals, schedule featured daily quests, and monitor traveller redemptions.
            </p>
          </div>

          {/* Quick Stats Pill Strip */}
          <div className="flex items-center gap-3">
            <div className="panel px-3.5 py-2 flex items-center gap-3">
              <div className="w-8 h-8 rounded-lg bg-amber-500/10 text-amber-500 flex items-center justify-center">
                <Clock size={16} />
              </div>
              <div>
                <p className="text-[10.5px] font-bold text-muted uppercase tracking-wider">Pending Review</p>
                <p className="text-[16px] font-bold text-ink leading-tight">{pending.length}</p>
              </div>
            </div>

            <div className="panel px-3.5 py-2 flex items-center gap-3">
              <div className="w-8 h-8 rounded-lg bg-emerald-500/10 text-emerald-500 flex items-center justify-center">
                <Award size={16} />
              </div>
              <div>
                <p className="text-[10.5px] font-bold text-muted uppercase tracking-wider">Approved Perks</p>
                <p className="text-[16px] font-bold text-ink leading-tight">{approved.length}</p>
              </div>
            </div>

            <div className="panel px-3.5 py-2 flex items-center gap-3">
              <div className="w-8 h-8 rounded-lg bg-accent/10 text-accent flex items-center justify-center">
                <Gift size={16} />
              </div>
              <div>
                <p className="text-[10.5px] font-bold text-muted uppercase tracking-wider">Redemptions</p>
                <p className="text-[16px] font-bold text-ink leading-tight">{totalRedemptions}</p>
              </div>
            </div>
          </div>
        </div>

        {/* Navigation Tabs */}
        <div className="flex items-center justify-between border-b border-line pb-1">
          <div className="flex gap-2">
            <button
              onClick={() => setTab('queue')}
              className={`relative px-4 py-2 rounded-xl text-[13.5px] font-bold transition-all flex items-center gap-2 ${
                tab === 'queue'
                  ? 'bg-accent text-white shadow-sm'
                  : 'text-ink-soft hover:text-ink hover:bg-black/5 dark:hover:bg-white/5'
              }`}
            >
              <span>Approval Queue</span>
              {pending.length > 0 && (
                <span
                  className={`inline-flex items-center justify-center text-[11px] font-bold px-1.5 py-0.2 min-w-5 h-5 rounded-full ${
                    tab === 'queue' ? 'bg-white text-accent' : 'bg-amber-500 text-white'
                  }`}
                >
                  {pending.length}
                </span>
              )}
            </button>

            <button
              onClick={() => setTab('approved')}
              className={`px-4 py-2 rounded-xl text-[13.5px] font-bold transition-all flex items-center gap-2 ${
                tab === 'approved'
                  ? 'bg-accent text-white shadow-sm'
                  : 'text-ink-soft hover:text-ink hover:bg-black/5 dark:hover:bg-white/5'
              }`}
            >
              <span>Approved Perks</span>
              <span
                className={`text-[11px] font-bold px-1.5 py-0.2 rounded-full ${
                  tab === 'approved' ? 'bg-white/20 text-white' : 'bg-black/10 dark:bg-white/10 text-muted'
                }`}
              >
                {approved.length}
              </span>
            </button>

            <button
              onClick={() => setTab('scheduler')}
              className={`px-4 py-2 rounded-xl text-[13.5px] font-bold transition-all flex items-center gap-2 ${
                tab === 'scheduler'
                  ? 'bg-accent text-white shadow-sm'
                  : 'text-ink-soft hover:text-ink hover:bg-black/5 dark:hover:bg-white/5'
              }`}
            >
              <Calendar size={15} />
              <span>Daily Scheduler</span>
            </button>

            <button
              onClick={() => setTab('completions')}
              className={`px-4 py-2 rounded-xl text-[13.5px] font-bold transition-all flex items-center gap-2 ${
                tab === 'completions'
                  ? 'bg-accent text-white shadow-sm'
                  : 'text-ink-soft hover:text-ink hover:bg-black/5 dark:hover:bg-white/5'
              }`}
            >
              <Users size={15} />
              <span>Completions & Redemptions</span>
            </button>
          </div>

          {(tab === 'queue' || tab === 'approved') && (
            <div className="relative w-64">
              <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted" />
              <input
                type="text"
                placeholder="Filter by vendor, town, reward..."
                value={searchFilter}
                onChange={(e) => setSearchFilter(e.target.value)}
                className="input pl-8 py-1.5 text-[12.5px]"
              />
              {searchFilter && (
                <button
                  onClick={() => setSearchFilter('')}
                  className="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted hover:text-ink"
                >
                  <X size={14} />
                </button>
              )}
            </div>
          )}
        </div>

        {/* Tab Content */}
        {tab === 'queue' && (
          <ApprovalQueueTab
            pending={pending}
            pendingLoading={pendingLoading}
            rejected={rejected}
            rejectedLoading={rejectedLoading}
            filter={searchFilter}
            onApprove={(id) => act.mutate({ id, status: 'approved' })}
            onReject={(id) => act.mutate({ id, status: 'rejected' })}
            onResetPending={(id) => act.mutate({ id, status: 'pending' })}
            isPending={act.isPending}
          />
        )}

        {tab === 'approved' && (
          <ApprovedSponsorsTab
            items={approved}
            isLoading={approvedLoading}
            filter={searchFilter}
            onSchedule={handleScheduleSponsor}
            onRevoke={(id) => act.mutate({ id, status: 'rejected' })}
            isPending={act.isPending}
          />
        )}

        {tab === 'scheduler' && (
          <SchedulerTab
            approved={approved}
            schedules={schedules}
            prefilledVendor={prefilledVendor}
            prefilledSponsorship={prefilledSponsorship}
            onClearPrefilled={() => {
              setPrefilledVendor('');
              setPrefilledSponsorship('');
            }}
          />
        )}

        {tab === 'completions' && <CompletionsTab />}
      </div>
    </Shell>
  );
}

// -------------------------------------------------------------
// Sub-tab 1: Approval Queue
// -------------------------------------------------------------
function ApprovalQueueTab({
  pending,
  pendingLoading,
  rejected,
  rejectedLoading,
  filter,
  onApprove,
  onReject,
  onResetPending,
  isPending,
}: {
  pending: VendorSponsorship[];
  pendingLoading: boolean;
  rejected: VendorSponsorship[];
  rejectedLoading: boolean;
  filter: string;
  onApprove: (id: string) => void;
  onReject: (id: string) => void;
  onResetPending: (id: string) => void;
  isPending: boolean;
}) {
  const [queueView, setQueueView] = useState<'pending' | 'rejected' | 'all'>('pending');

  const items = useMemo(() => {
    if (queueView === 'pending') return pending;
    if (queueView === 'rejected') return rejected;
    return [...pending, ...rejected];
  }, [queueView, pending, rejected]);

  const isLoading = queueView === 'rejected' ? rejectedLoading : pendingLoading;

  const filtered = useMemo(() => {
    if (!filter.trim()) return items;
    const q = filter.toLowerCase();
    return items.filter(
      (s) =>
        s.vendorName?.toLowerCase().includes(q) ||
        s.vendorTown?.toLowerCase().includes(q) ||
        s.vendorCategory?.toLowerCase().includes(q) ||
        s.vendorId.toLowerCase().includes(q) ||
        s.title?.toLowerCase().includes(q) ||
        s.rewardDescription.toLowerCase().includes(q)
    );
  }, [items, filter]);

  return (
    <div className="space-y-4">
      {/* Sub-filter bar */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-1.5 p-1 bg-black/5 dark:bg-white/5 rounded-xl border border-line">
          <button
            type="button"
            onClick={() => setQueueView('pending')}
            className={`px-3 py-1 rounded-lg text-[12px] font-bold transition-all flex items-center gap-1.5 ${
              queueView === 'pending'
                ? 'bg-accent text-white shadow-xs'
                : 'text-muted hover:text-ink'
            }`}
          >
            <span>Pending Review</span>
            <span
              className={`px-1.5 py-0.2 rounded-full text-[10px] font-bold ${
                queueView === 'pending' ? 'bg-white text-accent' : 'bg-amber-500/20 text-amber-500'
              }`}
            >
              {pending.length}
            </span>
          </button>

          <button
            type="button"
            onClick={() => setQueueView('rejected')}
            className={`px-3 py-1 rounded-lg text-[12px] font-bold transition-all flex items-center gap-1.5 ${
              queueView === 'rejected'
                ? 'bg-accent text-white shadow-xs'
                : 'text-muted hover:text-ink'
            }`}
          >
            <span>Rejected Proposals</span>
            <span
              className={`px-1.5 py-0.2 rounded-full text-[10px] font-bold ${
                queueView === 'rejected' ? 'bg-white text-accent' : 'bg-black/10 dark:bg-white/10 text-muted'
              }`}
            >
              {rejected.length}
            </span>
          </button>

          <button
            type="button"
            onClick={() => setQueueView('all')}
            className={`px-3 py-1 rounded-lg text-[12px] font-bold transition-all flex items-center gap-1.5 ${
              queueView === 'all'
                ? 'bg-accent text-white shadow-xs'
                : 'text-muted hover:text-ink'
            }`}
          >
            <span>All Requests</span>
            <span
              className={`px-1.5 py-0.2 rounded-full text-[10px] font-bold ${
                queueView === 'all' ? 'bg-white text-accent' : 'bg-black/10 dark:bg-white/10 text-muted'
              }`}
            >
              {pending.length + rejected.length}
            </span>
          </button>
        </div>

        <div className="text-[12px] font-bold text-muted">
          Showing {filtered.length} {queueView === 'rejected' ? 'rejected' : queueView === 'pending' ? 'pending' : ''} {filtered.length === 1 ? 'proposal' : 'proposals'}
        </div>
      </div>

      {isLoading ? (
        <div className="panel p-12 text-center">
          <Clock className="w-8 h-8 text-accent animate-spin mx-auto mb-3" />
          <p className="text-[14px] text-muted font-bold">Loading sponsorship proposals...</p>
        </div>
      ) : filtered.length === 0 ? (
        <div className="panel p-12 text-center">
          <div className="w-12 h-12 rounded-2xl bg-emerald-500/10 text-emerald-500 flex items-center justify-center mx-auto mb-3">
            <CheckCircle2 size={24} />
          </div>
          <h3 className="font-bold text-[16px] text-ink">
            {filter
              ? 'No proposals match your search'
              : queueView === 'rejected'
              ? 'No rejected proposals'
              : 'All clear! No pending proposals'}
          </h3>
          <p className="text-[13px] text-muted max-w-md mx-auto mt-1 font-medium">
            {filter
              ? 'Try clearing your search query to see all proposals.'
              : queueView === 'rejected'
              ? 'Proposals you reject will appear here for audit and reconsiderations.'
              : 'When vendors submit or update a Daily Quest reward sponsorship, it will appear here for admin review.'}
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-4">
          {filtered.map((s) => {
            const isItemRejected = s.status === 'rejected';
            const actionId = s.id || s.vendorId;

            return (
              <div
                key={s.id || s.vendorId}
                className="panel p-5 transition-all hover:border-accent/40 shadow-sm space-y-4"
              >
                <div className="flex flex-col sm:flex-row sm:items-start justify-between gap-3">
                  {/* Vendor Details */}
                  <div className="space-y-1 min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2.5">
                      <Link
                        href={`/vendors?id=${s.vendorId}`}
                        className="font-extrabold text-[16px] text-ink hover:text-accent flex items-center gap-1.5 transition-colors group"
                        title="Open vendor details in Vendors Directory"
                      >
                        <span>{s.vendorName || 'Unnamed Business'}</span>
                        <ExternalLink size={13} className="opacity-0 group-hover:opacity-100 transition-opacity text-accent" />
                      </Link>

                      {s.vendorCategory && (
                        <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-[11px] font-bold bg-accent/10 text-accent border border-accent/20">
                          <Tag size={10} /> {s.vendorCategory}
                        </span>
                      )}

                      {s.vendorTown && (
                        <span className="inline-flex items-center gap-1 text-[12px] font-semibold text-muted">
                          <MapPin size={12} className="text-accent" /> {s.vendorTown}
                        </span>
                      )}

                      <span className="font-mono text-[11px] px-2 py-0.5 rounded bg-black/5 dark:bg-white/5 text-muted border border-line">
                        {s.vendorId}
                      </span>
                    </div>

                    {s.createdAt && (
                      <p className="text-[11.5px] text-muted font-medium">
                        Submitted {timeAgo(s.createdAt)}
                      </p>
                    )}
                  </div>

                  {/* Prominent Status Pill */}
                  {isItemRejected ? (
                    <span className="inline-flex items-center gap-1 px-3 py-1 rounded-full text-[11px] font-extrabold bg-rose-500/10 text-rose-500 border border-rose-500/20 shrink-0">
                      <XCircle size={12} /> Edits Requested
                    </span>
                  ) : (
                    <span className="inline-flex items-center gap-1 px-3 py-1 rounded-full text-[11px] font-extrabold bg-amber-500/15 text-amber-600 border border-amber-500/25 shrink-0">
                      <Clock size={12} /> Pending Admin Review
                    </span>
                  )}
                </div>

                {/* Reward Description & Limit Block */}
                <div className="rounded-xl p-4 bg-black/[0.02] dark:bg-white/[0.02] border border-line flex flex-col md:flex-row md:items-center justify-between gap-4">
                  <div className="flex items-start gap-3 min-w-0 flex-1">
                    <div className="w-9 h-9 rounded-xl bg-accent/15 text-accent flex items-center justify-center shrink-0 mt-0.5">
                      <Gift size={18} />
                    </div>
                    <div className="min-w-0">
                      {s.title && (
                        <p className="font-extrabold text-[14.5px] text-ink mb-0.5 leading-snug">
                          {s.title}
                        </p>
                      )}
                      <p className="text-[10.5px] font-bold text-muted uppercase tracking-wider">
                        Offered Quest Reward
                      </p>
                      <p className="text-[14px] font-bold text-ink leading-relaxed mt-0.5">
                        {s.rewardDescription}
                      </p>
                    </div>
                  </div>

                  <div className="flex items-center gap-4 text-[12.5px] font-medium shrink-0 pt-2 md:pt-0 md:border-l md:border-line md:pl-5">
                    <div>
                      <span className="text-muted block text-[11px] font-bold uppercase">Daily Limit</span>
                      <span className="font-bold text-ink">
                        {s.rewardLimit > 0 ? `${s.rewardLimit} / day` : 'Unlimited'}
                      </span>
                    </div>

                    <div>
                      <span className="text-muted block text-[11px] font-bold uppercase">Sponsorship</span>
                      <span className={`font-bold ${s.isActive ? 'text-emerald-500' : 'text-muted'}`}>
                        {s.isActive ? 'Active' : 'Paused by vendor'}
                      </span>
                    </div>
                  </div>
                </div>

                {/* Clear, Prominent Action Strip */}
                <div className="flex items-center justify-end gap-3 pt-2 border-t border-line">
                  {isItemRejected ? (
                    <>
                      <button
                        onClick={() => onResetPending(actionId)}
                        disabled={isPending}
                        className="btn text-amber-500 hover:border-amber-500/50 hover:bg-amber-500/10 flex items-center gap-1.5 px-4 py-2 text-[13px] font-bold rounded-xl transition-all"
                        title="Restore to Pending Review status"
                      >
                        <Undo2 size={15} />
                        <span>Reset to Pending</span>
                      </button>

                      <button
                        onClick={() => onApprove(actionId)}
                        disabled={isPending}
                        className="btn bg-emerald-600 hover:bg-emerald-500 text-white border-emerald-600 flex items-center gap-2 px-5 py-2 text-[13px] shadow-sm font-extrabold rounded-xl transition-all"
                        title="Reconsider and approve proposal"
                      >
                        <CheckCircle2 size={16} />
                        <span>Approve Reward</span>
                      </button>
                    </>
                  ) : (
                    <>
                      <button
                        onClick={() => onReject(actionId)}
                        disabled={isPending}
                        className="btn border-rose-300 dark:border-rose-900/60 text-rose-600 hover:bg-rose-500/10 hover:border-rose-400 flex items-center gap-1.5 px-4 py-2 text-[13px] font-bold rounded-xl transition-all"
                        title="Reject or request edits for this perk"
                      >
                        <XCircle size={15} />
                        <span>Request Edits / Reject</span>
                      </button>

                      <button
                        onClick={() => onApprove(actionId)}
                        disabled={isPending}
                        className="btn bg-emerald-600 hover:bg-emerald-500 text-white border-emerald-600 flex items-center gap-2 px-5 py-2 text-[13px] shadow-sm font-extrabold rounded-xl transition-all"
                        title="Approve for scheduling"
                      >
                        <CheckCircle2 size={16} />
                        <span>Approve Reward</span>
                      </button>
                    </>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

// -------------------------------------------------------------
// Sub-tab 2: Approved Sponsors & Rewards
// -------------------------------------------------------------
function ApprovedSponsorsTab({
  items,
  isLoading,
  filter,
  onSchedule,
  onRevoke,
  isPending,
}: {
  items: VendorSponsorship[];
  isLoading: boolean;
  filter: string;
  onSchedule: (vendorId: string, sponsorshipId?: string) => void;
  onRevoke: (id: string) => void;
  isPending: boolean;
}) {
  const filtered = useMemo(() => {
    if (!filter.trim()) return items;
    const q = filter.toLowerCase();
    return items.filter(
      (s) =>
        s.vendorName?.toLowerCase().includes(q) ||
        s.vendorTown?.toLowerCase().includes(q) ||
        s.vendorCategory?.toLowerCase().includes(q) ||
        s.vendorId.toLowerCase().includes(q) ||
        s.title?.toLowerCase().includes(q) ||
        s.rewardDescription.toLowerCase().includes(q)
    );
  }, [items, filter]);

  if (isLoading) {
    return (
      <div className="panel p-12 text-center">
        <Clock className="w-8 h-8 text-accent animate-spin mx-auto mb-3" />
        <p className="text-[14px] text-muted font-bold">Loading approved sponsors...</p>
      </div>
    );
  }

  if (filtered.length === 0) {
    return (
      <div className="panel p-12 text-center">
        <div className="w-12 h-12 rounded-2xl bg-black/5 dark:bg-white/5 text-muted flex items-center justify-center mx-auto mb-3">
          <Award size={24} />
        </div>
        <h3 className="font-bold text-[16px] text-ink">No approved rewards found</h3>
        <p className="text-[13px] text-muted max-w-md mx-auto mt-1 font-medium">
          Once you approve vendor sponsorship requests from the Approval Queue, they will be listed here ready to be scheduled for daily quests.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between text-[12px] font-bold text-muted px-1">
        <span>{filtered.length} approved {filtered.length === 1 ? 'reward' : 'rewards'} ready to schedule</span>
        <span>Click &ldquo;Schedule Quest&rdquo; to feature this perk on a date</span>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {filtered.map((s) => (
          <div key={s.id || s.rewardDescription} className="panel p-5 flex flex-col justify-between gap-4 transition-all hover:border-accent/40">
            <div>
              <div className="flex items-start justify-between gap-2">
                <div>
                  <Link
                    href={`/vendors?id=${s.vendorId}`}
                    className="font-bold text-[15px] text-ink hover:text-accent flex items-center gap-1.5 transition-colors group"
                    title="Open vendor details in Vendors Directory"
                  >
                    <span>{s.vendorName || s.vendorId}</span>
                    <ExternalLink size={12} className="opacity-0 group-hover:opacity-100 transition-opacity text-accent" />
                  </Link>
                  <div className="flex items-center gap-2 mt-1 text-[12px] text-muted">
                    {s.vendorCategory && <span>{s.vendorCategory}</span>}
                    {s.vendorCategory && s.vendorTown && <span>·</span>}
                    {s.vendorTown && (
                      <span className="flex items-center gap-1">
                        <MapPin size={11} className="text-accent" /> {s.vendorTown}
                      </span>
                    )}
                  </div>
                </div>

                <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-[11px] font-bold bg-emerald-500/10 text-emerald-500 border border-emerald-500/20 shrink-0">
                  <Check size={11} /> Approved
                </span>
              </div>

              <div className="mt-3.5 p-3.5 rounded-xl bg-black/[0.02] dark:bg-white/[0.02] border border-line">
                {s.title && (
                  <p className="font-extrabold text-[13.5px] text-ink mb-0.5">{s.title}</p>
                )}
                <p className="text-[10.5px] font-bold text-muted uppercase tracking-wider">Perk Description</p>
                <p className="text-[13.5px] font-bold text-ink mt-0.5 leading-relaxed">{s.rewardDescription}</p>
                <div className="flex items-center gap-3 mt-2 text-[11.5px] text-muted font-medium">
                  <span>Limit: <strong className="text-ink">{s.rewardLimit > 0 ? `${s.rewardLimit}/day` : 'Unlimited'}</strong></span>
                  <span>·</span>
                  <span className={s.isActive ? 'text-emerald-500 font-semibold' : 'text-muted'}>
                    {s.isActive ? 'Active sponsorship' : 'Turned off by vendor'}
                  </span>
                </div>
              </div>
            </div>

            <div className="flex items-center justify-between pt-3 border-t border-line">
              <span className="font-mono text-[11px] text-muted">{s.vendorId}</span>
              <div className="flex items-center gap-2">
                <button
                  onClick={() => onRevoke(s.id || s.vendorId)}
                  disabled={isPending}
                  className="text-[12px] font-bold text-rose-500 hover:underline px-2 py-1"
                >
                  Revoke
                </button>
                <button
                  onClick={() => onSchedule(s.vendorId, s.id)}
                  className="btn btn-primary flex items-center gap-1.5 text-[12.5px] py-1.5 px-3.5 font-bold shadow-xs"
                >
                  <Calendar size={13} />
                  <span>Schedule Quest</span>
                </button>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

// -------------------------------------------------------------
// Sub-tab 3: Daily Scheduler
// -------------------------------------------------------------
function SchedulerTab({
  approved,
  schedules,
  prefilledVendor,
  prefilledSponsorship,
  onClearPrefilled,
}: {
  approved: VendorSponsorship[];
  schedules: QuestSchedule[];
  prefilledVendor: string;
  prefilledSponsorship: string;
  onClearPrefilled: () => void;
}) {
  const qc = useQueryClient();
  const todayStr = new Date().toISOString().slice(0, 10);
  const [selectedDate, setSelectedDate] = useState(todayStr);
  const [selectedVendor, setSelectedVendor] = useState(prefilledVendor || (approved[0]?.vendorId ?? ''));
  const [selectedSponsorshipId, setSelectedSponsorshipId] = useState(prefilledSponsorship || '');
  const [goal, setGoal] = useState(5);
  const [successNotice, setSuccessNotice] = useState<string | null>(null);

  // Group approved rewards by vendor
  const uniqueVendors = useMemo(() => {
    const map = new Map<string, { id: string; name: string; town: string; count: number }>();
    for (const a of approved) {
      if (!map.has(a.vendorId)) {
        map.set(a.vendorId, {
          id: a.vendorId,
          name: a.vendorName || a.vendorId,
          town: a.vendorTown || '',
          count: 0,
        });
      }
      map.get(a.vendorId)!.count++;
    }
    return Array.from(map.values());
  }, [approved]);

  // Synchronize prefilled vendor when prop updates or approved list loads
  useEffect(() => {
    if (prefilledVendor) {
      setSelectedVendor(prefilledVendor);
    } else if (approved.length > 0 && !approved.some((a) => a.vendorId === selectedVendor)) {
      setSelectedVendor(approved[0].vendorId);
    }
  }, [prefilledVendor, approved]);

  // Filter approved rewards for the currently selected vendor
  const vendorRewards = useMemo(() => {
    return approved.filter((a) => a.vendorId === selectedVendor);
  }, [approved, selectedVendor]);

  // Keep selected sponsorship ID in sync with the vendor's rewards
  useEffect(() => {
    if (prefilledSponsorship && vendorRewards.some((r) => r.id === prefilledSponsorship)) {
      setSelectedSponsorshipId(prefilledSponsorship);
    } else if (vendorRewards.length > 0 && !vendorRewards.some((r) => r.id === selectedSponsorshipId)) {
      setSelectedSponsorshipId(vendorRewards[0].id || '');
    } else if (vendorRewards.length === 0) {
      setSelectedSponsorshipId('');
    }
  }, [selectedVendor, prefilledSponsorship]);

  const existingScheduleForDate = useMemo(
    () => schedules.find((s) => s.date === selectedDate),
    [schedules, selectedDate]
  );

  const scheduleMutation = useMutation({
    mutationFn: (req: { date: string; vendorId: string; sponsorshipId?: string; goal: number }) =>
      api.post('/admin/quests/schedules', req),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin-quests'] });
      setSuccessNotice(`Scheduled quest for ${selectedDate} successfully!`);
      setTimeout(() => setSuccessNotice(null), 4000);
      onClearPrefilled();
    },
  });

  const deleteScheduleMutation = useMutation({
    mutationFn: (date: string) => api.del(`/admin/quests/schedules/${date}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin-quests'] });
      setSuccessNotice(`Quest schedule removed.`);
      setTimeout(() => setSuccessNotice(null), 4000);
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedDate || !selectedVendor) return;
    scheduleMutation.mutate({
      date: selectedDate,
      vendorId: selectedVendor,
      sponsorshipId: selectedSponsorshipId || undefined,
      goal: Number(goal) || 5,
    });
  };

  const selectedRewardData = useMemo(
    () => approved.find((a) => a.id === selectedSponsorshipId || (a.vendorId === selectedVendor && !selectedSponsorshipId)),
    [approved, selectedSponsorshipId, selectedVendor]
  );

  return (
    <div className="grid grid-cols-1 lg:grid-cols-12 gap-8">
      {/* Schedule Form */}
      <div className="lg:col-span-5 space-y-4">
        <div>
          <h2 className="font-bold text-[16px] text-ink">Schedule a Quest Day</h2>
          <p className="text-[12.5px] text-muted font-medium mt-0.5">
            Select an approved sponsor and reward to feature for travellers on a specific date.
          </p>
        </div>

        <form onSubmit={handleSubmit} className="panel p-5 space-y-4">
          {successNotice && (
            <div className="p-3 rounded-xl bg-emerald-500/10 border border-emerald-500/20 text-emerald-500 text-[13px] font-bold flex items-center gap-2">
              <CheckCircle2 size={16} />
              <span>{successNotice}</span>
            </div>
          )}

          <div>
            <label className="block text-[12px] font-bold text-muted mb-1.5 uppercase tracking-wider">
              Quest Date (UTC)
            </label>
            <input
              type="date"
              value={selectedDate}
              onChange={(e) => setSelectedDate(e.target.value)}
              required
              className="input text-[14px]"
            />
            <div className="flex gap-2 mt-2">
              <button
                type="button"
                onClick={() => setSelectedDate(todayStr)}
                className="text-[11px] font-bold px-2 py-0.5 rounded bg-black/5 dark:bg-white/5 text-muted hover:text-ink"
              >
                Today
              </button>
              <button
                type="button"
                onClick={() => {
                  const d = new Date();
                  d.setDate(d.getDate() + 1);
                  setSelectedDate(d.toISOString().slice(0, 10));
                }}
                className="text-[11px] font-bold px-2 py-0.5 rounded bg-black/5 dark:bg-white/5 text-muted hover:text-ink"
              >
                Tomorrow
              </button>
            </div>

            {existingScheduleForDate ? (
              <div className="p-2.5 rounded-xl bg-amber-500/10 border border-amber-500/20 text-amber-600 dark:text-amber-400 text-[12px] font-semibold flex items-start gap-2 mt-2.5">
                <AlertCircle size={15} className="shrink-0 mt-0.5 text-amber-500" />
                <span>
                  <strong>{selectedDate}</strong> is currently scheduled for{' '}
                  <strong>{existingScheduleForDate.sponsorName || existingScheduleForDate.vendorId}</strong> ({existingScheduleForDate.rewardDescription}). Submitting will reassign this quest day.
                </span>
              </div>
            ) : selectedDate ? (
              <div className="p-2 rounded-xl bg-emerald-500/10 border border-emerald-500/20 text-emerald-600 dark:text-emerald-400 text-[11.5px] font-semibold flex items-center gap-1.5 mt-2.5">
                <CheckCircle2 size={13} className="shrink-0 text-emerald-500" />
                <span><strong>{selectedDate}</strong> is open and ready to schedule.</span>
              </div>
            ) : null}
          </div>

          <div>
            <label className="block text-[12px] font-bold text-muted mb-1.5 uppercase tracking-wider">
              Featured Vendor
            </label>
            {uniqueVendors.length === 0 ? (
              <p className="text-[13px] text-rose-500 font-bold p-3 rounded-xl bg-rose-500/10 border border-rose-500/20">
                No approved sponsors available. Please approve a sponsorship from the Approval Queue first.
              </p>
            ) : (
              <select
                value={selectedVendor}
                onChange={(e) => setSelectedVendor(e.target.value)}
                required
                className="input text-[13.5px] cursor-pointer"
              >
                <option value="">-- Choose an approved vendor --</option>
                {uniqueVendors.map((v) => (
                  <option key={v.id} value={v.id}>
                    {v.name} ({v.town || 'Unknown'} · {v.count} {v.count === 1 ? 'reward' : 'rewards'})
                  </option>
                ))}
              </select>
            )}
          </div>

          {/* Reward Picker for selected vendor */}
          {vendorRewards.length > 0 && (
            <div>
              <label className="block text-[12px] font-bold text-muted mb-1.5 uppercase tracking-wider">
                Featured Quest Reward
              </label>
              <select
                value={selectedSponsorshipId}
                onChange={(e) => setSelectedSponsorshipId(e.target.value)}
                className="input text-[13.5px] cursor-pointer"
              >
                {vendorRewards.map((r) => (
                  <option key={r.id || r.rewardDescription} value={r.id || ''}>
                    {r.title ? `${r.title} — ` : ''}{r.rewardDescription} ({r.rewardLimit > 0 ? `Limit ${r.rewardLimit}` : 'Unlimited'})
                  </option>
                ))}
              </select>
            </div>
          )}

          {selectedRewardData && (
            <div className="rounded-xl p-3.5 bg-black/[0.02] dark:bg-white/[0.02] border border-line space-y-1">
              <p className="text-[10.5px] font-bold text-muted uppercase tracking-wider">Featured Reward Preview</p>
              {selectedRewardData.title && (
                <p className="text-[14px] font-black text-ink">{selectedRewardData.title}</p>
              )}
              <p className="text-[13px] font-bold text-ink leading-relaxed">{selectedRewardData.rewardDescription}</p>
              <p className="text-[11.5px] text-muted font-medium mt-1">
                Limit: {selectedRewardData.rewardLimit > 0 ? `${selectedRewardData.rewardLimit}/day` : 'Unlimited'} · Location: {selectedRewardData.vendorTown || '—'}
              </p>
            </div>
          )}

          <div>
            <label className="block text-[12px] font-bold text-muted mb-1.5 uppercase tracking-wider">
              Visit Goal (Places to check in)
            </label>
            <input
              type="number"
              min="1"
              max="20"
              value={goal}
              onChange={(e) => setGoal(parseInt(e.target.value) || 5)}
              className="input text-[14px]"
            />
            <p className="text-[11.5px] text-muted font-medium mt-1">
              Standard Daily Quest requires 5 check-ins before unlocking the vendor reward.
            </p>
          </div>

          <button
            type="submit"
            disabled={scheduleMutation.isPending || approved.length === 0}
            className="btn btn-primary w-full py-2.5 text-[14px] flex items-center justify-center gap-2 mt-2 shadow-sm font-bold"
          >
            <Calendar size={16} />
            <span>{scheduleMutation.isPending ? 'Scheduling...' : existingScheduleForDate ? 'Reassign Quest Schedule' : 'Confirm Quest Schedule'}</span>
          </button>
        </form>
      </div>

      {/* Scheduled Quests List */}
      <div className="lg:col-span-7 space-y-4">
        <div>
          <h2 className="font-bold text-[16px] text-ink">Upcoming & Past Schedules</h2>
          <p className="text-[12.5px] text-muted font-medium mt-0.5">
            Active and scheduled daily quests with live completion and redemption counts.
          </p>
        </div>

        <div className="space-y-3">
          {schedules.map((s) => {
            const [year, month, day] = s.date.split('-');
            const d = new Date(parseInt(year, 10), parseInt(month, 10) - 1, parseInt(day, 10));
            const isToday = s.date === todayStr;
            const isUpcoming = s.date >= todayStr;

            return (
              <div
                key={s.date}
                className={`panel p-4 flex items-center gap-4 transition-all hover:border-accent/40 ${
                  isToday ? 'border-accent bg-accent/[0.03]' : ''
                }`}
              >
                {/* Date Badge */}
                <div
                  className={`w-14 h-14 rounded-xl flex flex-col items-center justify-center shrink-0 border ${
                    isToday
                      ? 'bg-accent text-white border-accent'
                      : 'bg-black/5 dark:bg-white/5 border-line text-ink'
                  }`}
                >
                  <span className="text-[10px] font-bold uppercase tracking-wider">
                    {d.toLocaleString('default', { month: 'short' })}
                  </span>
                  <span className="text-[18px] font-extrabold leading-none">{d.getDate()}</span>
                </div>

                {/* Sponsor & Quest Info */}
                <div className="min-w-0 flex-1 space-y-0.5">
                  <div className="flex items-center gap-2">
                    <Link
                      href={`/vendors?id=${s.vendorId}`}
                      className="font-bold text-[14.5px] text-ink hover:text-accent flex items-center gap-1 transition-colors group truncate"
                      title="Open vendor details in Vendors Directory"
                    >
                      <span className="truncate">{s.sponsorName || s.vendorId}</span>
                      <ExternalLink size={11} className="opacity-0 group-hover:opacity-100 transition-opacity text-accent shrink-0" />
                    </Link>
                    {isToday && (
                      <span className="px-2 py-0.5 rounded-full text-[10px] font-bold bg-accent text-white">
                        TODAY
                      </span>
                    )}
                    {s.vendorTown && (
                      <span className="text-[11.5px] text-muted font-semibold flex items-center gap-0.5">
                        <MapPin size={10} className="text-accent" /> {s.vendorTown}
                      </span>
                    )}
                  </div>

                  <p className="text-[13px] text-muted font-medium truncate">
                    Reward: <span className="text-ink font-semibold">{s.rewardTitle ? `${s.rewardTitle} — ` : ''}{s.rewardDescription || 'Special Perk'}</span>
                    {' '}(Goal: {s.goal || 5} stops)
                  </p>
                </div>

                {/* Actions & Redemptions/Completions Stats */}
                <div className="text-right shrink-0 pl-2 space-y-1">
                  <div className="flex items-center justify-end gap-1.5 text-[13px] font-bold text-ink">
                    <CheckCircle2 size={13} className="text-emerald-500" />
                    <span>{s.completions}</span>
                    <span className="text-muted text-[11px] font-normal">completed</span>
                  </div>
                  <div className="flex items-center justify-end gap-1.5 text-[12px] font-semibold text-muted">
                    <Gift size={12} className="text-accent" />
                    <span className="text-ink font-bold">{s.redemptions}</span>
                    <span>claimed</span>
                  </div>
                  {isUpcoming && (
                    <button
                      type="button"
                      disabled={deleteScheduleMutation.isPending}
                      onClick={() => {
                        if (confirm(`Remove quest schedule for ${s.date}?`)) {
                          deleteScheduleMutation.mutate(s.date);
                        }
                      }}
                      className="text-[11px] font-bold text-rose-500 hover:underline pt-0.5 block ml-auto"
                    >
                      Cancel Schedule
                    </button>
                  )}
                </div>
              </div>
            );
          })}

          {schedules.length === 0 && (
            <div className="panel p-8 text-center text-muted text-[13px] font-medium">
              No quests scheduled yet. Use the form on the left to schedule the first one.
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

// -------------------------------------------------------------
// Sub-tab 4: Quest Completions & Redemptions
// -------------------------------------------------------------
function CompletionsTab() {
  const [date, setDate] = useState('');
  const { data, isLoading, isError, fetchNextPage, hasNextPage, isFetchingNextPage } = useInfiniteQuery({
    queryKey: ['admin-quests', 'completions', date],
    initialPageParam: '',
    queryFn: ({ pageParam }) => {
      const q = new URLSearchParams();
      if (date) q.set('date', date);
      if (pageParam) q.set('before', pageParam);
      return api.get<QuestCompletion[]>(`/admin/quests/completions?${q}`);
    },
    getNextPageParam: (last) => (last.length < 100 ? undefined : last[last.length - 1].completedAt),
  });

  const rows = data?.pages.flat() ?? [];
  const redeemedCount = rows.filter((r) => r.redeemedAt).length;

  return (
    <div className="space-y-4">
      {/* Filters & Summary */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <label className="flex items-center gap-2">
            <span className="text-[12px] font-bold text-muted uppercase">Quest Day:</span>
            <input
              type="date"
              value={date}
              onChange={(e) => setDate(e.target.value)}
              className="input py-1 text-[13px] w-40"
            />
          </label>
          {date && (
            <button
              onClick={() => setDate('')}
              className="text-[12px] font-bold text-accent hover:underline"
            >
              Show all days
            </button>
          )}
        </div>

        <div className="text-[12.5px] font-medium text-muted">
          Showing <span className="font-bold text-ink">{rows.length}</span> completions ·{' '}
          <span className="font-bold text-emerald-500">{redeemedCount}</span> redeemed
        </div>
      </div>

      {isError && <InlineErrorBanner message="Couldn't load quest completions — try refreshing." />}

      {/* Completions Table */}
      <div className="panel overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-[13px] min-w-[700px]">
            <thead>
              <tr className="border-b border-line text-left text-[11px] font-bold uppercase tracking-wider text-muted bg-black/[0.02] dark:bg-white/[0.02]">
                <th className="px-4 py-3">Traveller</th>
                <th className="px-4 py-3">Quest Day</th>
                <th className="px-4 py-3">Featured Sponsor</th>
                <th className="px-4 py-3">Completed At</th>
                <th className="px-4 py-3 text-right">Redemption Status</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-line">
              {rows.map((c) => (
                <tr key={`${c.travellerId}-${c.date}`} className="hover:bg-black/[0.01] dark:hover:bg-white/[0.01]">
                  <td className="px-4 py-3">
                    <p className="font-bold text-ink">{c.travellerName || 'Traveller'}</p>
                    {c.travellerEmail && <p className="text-[11.5px] text-muted font-medium">{c.travellerEmail}</p>}
                  </td>
                  <td className="px-4 py-3 font-semibold tabular-nums text-ink">{c.date}</td>
                  <td className="px-4 py-3">
                    {c.vendorId ? (
                      <Link
                        href={`/vendors?id=${c.vendorId}`}
                        className="font-bold text-ink hover:text-accent flex items-center gap-1 transition-colors group"
                        title="Open vendor details in Vendors Directory"
                      >
                        <span>{c.sponsorName}</span>
                        <ExternalLink size={11} className="opacity-0 group-hover:opacity-100 transition-opacity text-accent" />
                      </Link>
                    ) : (
                      <p className="font-bold text-ink">{c.sponsorName}</p>
                    )}
                    <p className="text-[11.5px] text-muted">{c.rewardDescription} (Goal: {c.goal})</p>
                  </td>
                  <td className="px-4 py-3 text-muted font-medium" title={new Date(c.completedAt).toLocaleString()}>
                    {timeAgo(c.completedAt)}
                  </td>
                  <td className="px-4 py-3 text-right">
                    {c.redeemedAt ? (
                      <span
                        className="inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-[11px] font-bold bg-emerald-500/10 text-emerald-500 border border-emerald-500/20"
                        title={new Date(c.redeemedAt).toLocaleString()}
                      >
                        <CheckCircle2 size={12} /> Claimed at Vendor
                      </span>
                    ) : (
                      <span className="inline-flex rounded-full px-2.5 py-0.5 text-[11px] font-semibold bg-black/5 dark:bg-white/5 text-muted border border-line">
                        Reward Unclaimed
                      </span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {isLoading && (
          <div className="p-8 text-center text-muted text-[13px] font-medium">Loading completions...</div>
        )}

        {!isLoading && rows.length === 0 && (
          <div className="p-8 text-center text-muted text-[13px] font-medium">
            {date ? 'Nobody completed the quest on this day.' : 'No quest completions recorded yet.'}
          </div>
        )}
      </div>

      {hasNextPage && (
        <button
          onClick={() => fetchNextPage()}
          disabled={isFetchingNextPage}
          className="btn w-full py-2 text-[13px] font-bold"
        >
          {isFetchingNextPage ? 'Loading more...' : 'Load older completions'}
        </button>
      )}
    </div>
  );
}
