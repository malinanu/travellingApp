'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  AlertCircle,
  Award,
  Calendar,
  Check,
  CheckCircle2,
  Clock,
  Edit2,
  ExternalLink,
  Gift,
  HelpCircle,
  Info,
  MapPin,
  Pause,
  Play,
  Plus,
  Power,
  RotateCcw,
  Save,
  Sparkles,
  Trash2,
  Users,
  X,
  XCircle,
} from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import { timeAgo, type QuestCompletion, type QuestSchedule, type VendorSponsorship } from '@spotto/shared';
import Shell from '../../components/shell';
import { api } from '../../lib/api';

export default function SponsorshipPage() {
  const qc = useQueryClient();

  const { data: sponsorships = [], isLoading: sponsorshipsLoading } = useQuery<VendorSponsorship[]>({
    queryKey: ['vendor-sponsorships'],
    queryFn: async () => {
      try {
        const res = await api.get<VendorSponsorship[]>('/vendor/sponsorships');
        if (Array.isArray(res)) return res;
      } catch (err) {
        // Fallback to legacy single-endpoint if running older server instance
      }
      const single = await api.get<VendorSponsorship>('/vendor/sponsorship');
      return single && single.status !== 'none' ? [single] : [];
    },
  });

  const { data: schedules = [], isLoading: schedulesLoading } = useQuery<QuestSchedule[]>({
    queryKey: ['vendor-quest-schedules'],
    queryFn: () => api.get<QuestSchedule[]>('/vendor/quest/schedules'),
  });

  const [modalOpen, setModalOpen] = useState(false);
  const [editingItem, setEditingItem] = useState<VendorSponsorship | null>(null);
  const [formTitle, setFormTitle] = useState('');
  const [formDesc, setFormDesc] = useState('');
  const [formLimit, setFormLimit] = useState(0);
  const [formActive, setFormActive] = useState(true);
  const [statusMsg, setStatusMsg] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  const openCreateModal = () => {
    setEditingItem(null);
    setFormTitle('');
    setFormDesc('');
    setFormLimit(0);
    setFormActive(true);
    setModalOpen(true);
  };

  const openEditModal = (s: VendorSponsorship) => {
    setEditingItem(s);
    setFormTitle(s.title || '');
    setFormDesc(s.rewardDescription || '');
    setFormLimit(s.rewardLimit || 0);
    setFormActive(s.isActive ?? true);
    setModalOpen(true);
  };

  const saveMutation = useMutation({
    mutationFn: async () => {
      if (editingItem && editingItem.id) {
        return api.put<VendorSponsorship>(`/vendor/sponsorships/${editingItem.id}`, {
          title: formTitle,
          rewardDescription: formDesc,
          rewardLimit: formLimit,
          isActive: formActive,
        });
      } else if (editingItem) {
        return api.put<VendorSponsorship>('/vendor/sponsorship', {
          title: formTitle,
          rewardDescription: formDesc,
          rewardLimit: formLimit,
          isActive: formActive,
        });
      } else {
        return api.post<VendorSponsorship>('/vendor/sponsorships', {
          title: formTitle,
          rewardDescription: formDesc,
          rewardLimit: formLimit,
          isActive: formActive,
        });
      }
    },
    onSuccess: (data) => {
      qc.invalidateQueries({ queryKey: ['vendor-sponsorships'] });
      qc.invalidateQueries({ queryKey: ['sponsorship'] });
      setModalOpen(false);
      setStatusMsg({
        type: 'success',
        text: editingItem
          ? 'Reward updated successfully.'
          : 'New quest reward proposed and submitted for admin review!',
      });
      setTimeout(() => setStatusMsg(null), 5000);
    },
    onError: () => {
      setStatusMsg({ type: 'error', text: 'Failed to save quest reward. Please check your inputs.' });
    },
  });

  const toggleActiveMutation = useMutation({
    mutationFn: (item: VendorSponsorship) => {
      const nextActive = !item.isActive;
      if (item.id) {
        return api.put<VendorSponsorship>(`/vendor/sponsorships/${item.id}`, {
          title: item.title || '',
          rewardDescription: item.rewardDescription,
          rewardLimit: item.rewardLimit,
          isActive: nextActive,
        });
      }
      return api.put<VendorSponsorship>('/vendor/sponsorship', {
        rewardDescription: item.rewardDescription,
        rewardLimit: item.rewardLimit,
        isActive: nextActive,
      });
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['vendor-sponsorships'] });
      qc.invalidateQueries({ queryKey: ['sponsorship'] });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.del(`/vendor/sponsorships/${id}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['vendor-sponsorships'] });
      qc.invalidateQueries({ queryKey: ['sponsorship'] });
      setStatusMsg({ type: 'success', text: 'Quest reward removed.' });
      setTimeout(() => setStatusMsg(null), 4000);
    },
  });

  // Calculate upcoming scheduled date
  const todayStr = new Date().toISOString().slice(0, 10);
  const upcomingSchedule = useMemo(() => {
    return schedules
      .filter((s) => s.date >= todayStr)
      .sort((a, b) => a.date.localeCompare(b.date))[0];
  }, [schedules, todayStr]);

  const approvedCount = sponsorships.filter((s) => s.status === 'approved').length;
  const pendingCount = sponsorships.filter((s) => s.status === 'pending').length;
  const totalRedemptions = schedules.reduce((acc, s) => acc + (s.redemptions || 0), 0);

  return (
    <Shell>
      <div className="max-w-5xl space-y-7 pb-16">
        {/* Top Header */}
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div>
            <div className="flex items-center gap-2.5">
              <h1 className="text-2xl font-black font-display tracking-tight text-ink">Daily Quest Sponsorship</h1>
              <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-[11px] font-extrabold bg-accent/10 text-accent border border-accent/20">
                <Sparkles size={11} /> Official Town Spotlight
              </span>
            </div>
            <p className="text-muted font-medium text-[13.5px] mt-1 max-w-2xl">
              The Official Town Grand Prize · Sponsor the featured reward for travellers completing local quests in your region. On your scheduled day, every active adventurer in town is guided to your counter.
            </p>
          </div>

          <button
            onClick={openCreateModal}
            className="btn btn-accent px-4 py-2.5 rounded-xl flex items-center justify-center gap-2 shadow-sm font-extrabold text-[13.5px] shrink-0 self-start md:self-auto"
          >
            <Plus size={16} />
            <span>Propose New Reward</span>
          </button>
        </div>

        {/* Global Toast Alert */}
        {statusMsg && (
          <div
            className={`p-3.5 rounded-xl text-[13px] font-bold flex items-center gap-2.5 border transition-all ${
              statusMsg.type === 'success'
                ? 'bg-emerald-500/10 border-emerald-500/20 text-emerald-800'
                : 'bg-rose-500/10 border-rose-500/20 text-rose-800'
            }`}
          >
            {statusMsg.type === 'success' ? <CheckCircle2 size={16} /> : <AlertCircle size={16} />}
            <span>{statusMsg.text}</span>
          </div>
        )}

        {/* Quick Stats Pill Strip */}
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3.5">
          <div className="card p-4 flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-accent/15 text-accent flex items-center justify-center shrink-0">
              <Gift size={18} />
            </div>
            <div>
              <p className="text-[11px] font-bold text-muted uppercase tracking-wider">Total Perks</p>
              <p className="text-[18px] font-black text-ink leading-tight">{sponsorships.length}</p>
            </div>
          </div>

          <div className="card p-4 flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-emerald-500/15 text-emerald-600 flex items-center justify-center shrink-0">
              <CheckCircle2 size={18} />
            </div>
            <div>
              <p className="text-[11px] font-bold text-muted uppercase tracking-wider">Approved</p>
              <p className="text-[18px] font-black text-emerald-600 leading-tight">{approvedCount}</p>
            </div>
          </div>

          <div className="card p-4 flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-amber-500/15 text-amber-600 flex items-center justify-center shrink-0">
              <Clock size={18} />
            </div>
            <div>
              <p className="text-[11px] font-bold text-muted uppercase tracking-wider">In Review</p>
              <p className="text-[18px] font-black text-amber-600 leading-tight">{pendingCount}</p>
            </div>
          </div>

          <div className="card p-4 flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-purple-500/15 text-purple-600 flex items-center justify-center shrink-0">
              <Users size={18} />
            </div>
            <div>
              <p className="text-[11px] font-bold text-muted uppercase tracking-wider">Redemptions</p>
              <p className="text-[18px] font-black text-purple-600 leading-tight">{totalRedemptions}</p>
            </div>
          </div>
        </div>

        {/* How It Works Explainer Banner */}
        <div className="card p-5 bg-sand/30 border border-line">
          <div className="flex items-center gap-2 mb-3">
            <Sparkles size={16} className="text-accent" />
            <h3 className="font-extrabold text-[14.5px] text-ink">How Daily Quest Sponsorship Works</h3>
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-4 gap-3">
            <div className="bg-white p-3.5 rounded-xl border border-line/70">
              <span className="w-6 h-6 rounded-full bg-accent/15 text-accent text-[12px] font-black flex items-center justify-center mb-2">1</span>
              <p className="font-extrabold text-[13px] text-ink">Propose Your Perk</p>
              <p className="text-[11.5px] text-muted font-medium mt-0.5 leading-snug">
                Offer a small complimentary gift (e.g. Free Pot of Tea, Drink, Snack, or Dessert).
              </p>
            </div>

            <div className="bg-white p-3.5 rounded-xl border border-line/70">
              <span className="w-6 h-6 rounded-full bg-accent/15 text-accent text-[12px] font-black flex items-center justify-center mb-2">2</span>
              <p className="font-extrabold text-[13px] text-ink">Spotto Approval</p>
              <p className="text-[11.5px] text-muted font-medium mt-0.5 leading-snug">
                Admins review and approve your proposed perk so it can be scheduled for upcoming dates.
              </p>
            </div>

            <div className="bg-white p-3.5 rounded-xl border border-line/70">
              <span className="w-6 h-6 rounded-full bg-accent/15 text-accent text-[12px] font-black flex items-center justify-center mb-2">3</span>
              <p className="font-extrabold text-[13px] text-ink">Featured Day</p>
              <p className="text-[11.5px] text-muted font-medium mt-0.5 leading-snug">
                Your venue becomes the featured destination for all active travelers exploring that day.
              </p>
            </div>

            <div className="bg-white p-3.5 rounded-xl border border-line/70">
              <span className="w-6 h-6 rounded-full bg-accent/15 text-accent text-[12px] font-black flex items-center justify-center mb-2">4</span>
              <p className="font-extrabold text-[13px] text-ink">Scan & Welcome</p>
              <p className="text-[11.5px] text-muted font-medium mt-0.5 leading-snug">
                Travelers show their SPOTTO code at your counter. Scan or enter it to verify and greet them.
              </p>
            </div>
          </div>
        </div>

        {/* Quest Rewards Section Header */}
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-[17px] font-extrabold text-ink">Your Quest Reward Offerings</h2>
              <p className="text-[12.5px] text-muted font-medium">
                Manage the rewards you provide to travelers. You can offer multiple rewards, and admins can feature any approved reward.
              </p>
            </div>
            <button
              onClick={openCreateModal}
              className="btn btn-primary text-[12.5px] py-1.5 px-3 flex items-center gap-1.5 font-bold"
            >
              <Plus size={14} />
              <span>Add Reward</span>
            </button>
          </div>

          {sponsorshipsLoading ? (
            <div className="card p-12 text-center text-muted font-bold">
              <Clock className="w-7 h-7 text-accent animate-spin mx-auto mb-2" />
              <span>Loading quest rewards...</span>
            </div>
          ) : sponsorships.length === 0 ? (
            <div className="card p-10 text-center space-y-3 bg-sand/20 border-dashed">
              <div className="w-12 h-12 rounded-2xl bg-accent/10 text-accent flex items-center justify-center mx-auto">
                <Gift size={24} />
              </div>
              <div className="max-w-md mx-auto space-y-1">
                <h3 className="font-extrabold text-[15px] text-ink">No Quest Rewards Yet</h3>
                <p className="text-[12.5px] text-muted font-medium">
                  Partner with Spotto by offering a complimentary perk for travelers. Submit your first reward to get started!
                </p>
              </div>
              <button
                onClick={openCreateModal}
                className="btn btn-accent px-4 py-2 text-[13px] font-extrabold inline-flex items-center gap-2 shadow-sm"
              >
                <Plus size={15} />
                <span>Propose Your First Reward</span>
              </button>
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {sponsorships.map((s) => {
                const isPending = s.status === 'pending';
                const isApproved = s.status === 'approved';
                const isRejected = s.status === 'rejected';
                const isPaused = !s.isActive;

                // Check if this specific reward is scheduled
                const isScheduledUpcoming = schedules.some(
                  (sch) => (sch.sponsorshipId === s.id || (!sch.sponsorshipId && s.rewardDescription === sch.rewardDescription)) && sch.date >= todayStr
                );

                return (
                  <div
                    key={s.id || s.rewardDescription}
                    className="card p-5 flex flex-col justify-between gap-4 transition-all hover:border-accent/40 shadow-sm relative overflow-hidden"
                  >
                    <div className="space-y-3">
                      {/* Top Bar: Title & Status Badge */}
                      <div className="flex items-start justify-between gap-2.5">
                        <div className="flex items-center gap-2.5 min-w-0">
                          <div className="w-9 h-9 rounded-xl bg-accent/15 text-accent flex items-center justify-center shrink-0">
                            <Gift size={18} />
                          </div>
                          <div className="min-w-0">
                            <h3 className="font-extrabold text-[15px] text-ink truncate leading-tight">
                              {s.title || 'Daily Quest Perk'}
                            </h3>
                            <p className="text-[11px] text-muted font-semibold mt-0.5">
                              {s.createdAt ? `Created ${timeAgo(s.createdAt)}` : 'Active proposal'}
                            </p>
                          </div>
                        </div>

                        {/* Status Badges (Strictly Informational Labels) */}
                        {isScheduledUpcoming ? (
                          <span className="badge bg-accent text-white shadow-xs shrink-0">
                            <Calendar size={11} /> Scheduled
                          </span>
                        ) : isApproved ? (
                          <span className="badge bg-emerald-500/15 text-emerald-700 border border-emerald-500/20 shrink-0">
                            <CheckCircle2 size={11} /> Approved
                          </span>
                        ) : isPending ? (
                          <span className="badge bg-amber-500/15 text-amber-600 border border-amber-500/20 shrink-0">
                            <Clock size={11} /> Pending Review
                          </span>
                        ) : isRejected ? (
                          <span className="badge bg-coral/15 text-coral border border-coral/20 shrink-0">
                            <XCircle size={11} /> Edits Requested
                          </span>
                        ) : null}
                      </div>

                      {/* Reward Description Block */}
                      <div className="p-3.5 rounded-xl bg-sand/30 border border-line">
                        <p className="text-[11px] font-extrabold text-muted uppercase tracking-wider mb-1">
                          Offered Quest Reward
                        </p>
                        <p className="text-[13.5px] font-bold text-ink leading-relaxed">
                          {s.rewardDescription}
                        </p>
                      </div>

                      {/* Badges strip: Daily limit & Active state indicator */}
                      <div className="flex items-center gap-2.5 text-[12px] font-semibold text-muted flex-wrap">
                        <span className="badge bg-sand/70 text-ink border border-line/60">
                          Limit: <strong className="font-extrabold">{s.rewardLimit > 0 ? `${s.rewardLimit}/day` : 'Unlimited'}</strong>
                        </span>

                        <span className={`badge ${s.isActive ? 'bg-emerald-500/10 text-emerald-700 border border-emerald-500/20' : 'bg-sand text-muted border border-line'}`}>
                          <span className={`w-1.5 h-1.5 rounded-full ${s.isActive ? 'bg-emerald-500' : 'bg-gray-400'}`} />
                          <span>{s.isActive ? 'Active for scheduling' : 'Paused by you'}</span>
                        </span>
                      </div>

                      {/* Rejection notice if edits requested */}
                      {isRejected && (
                        <div className="p-2.5 rounded-xl bg-coral/10 border border-coral/20 text-coral text-[12px] font-semibold flex items-start gap-2">
                          <AlertCircle size={14} className="shrink-0 mt-0.5" />
                          <span>Admins requested adjustments. Click &ldquo;Edit&rdquo; to modify the perk description.</span>
                        </div>
                      )}
                    </div>

                    {/* Bottom Action Bar — Unmistakable Buttons */}
                    <div className="flex items-center justify-between pt-3 border-t border-line mt-1">
                      {/* Explicit Interactive Control for Active/Pause State */}
                      <button
                        type="button"
                        onClick={() => toggleActiveMutation.mutate(s)}
                        disabled={toggleActiveMutation.isPending}
                        className={`btn btn-secondary text-[12px] py-1.5 px-3 font-extrabold ${
                          s.isActive
                            ? 'text-amber-800 hover:text-amber-900 border-amber-300 hover:border-amber-400 bg-amber-50/50'
                            : 'text-emerald-700 hover:text-emerald-800 border-emerald-300 hover:border-emerald-400 bg-emerald-50/50'
                        }`}
                        title={s.isActive ? 'Pause reward from scheduling' : 'Activate reward for scheduling'}
                      >
                        {s.isActive ? (
                          <>
                            <Pause size={13} className="text-amber-600" />
                            <span>Pause Reward</span>
                          </>
                        ) : (
                          <>
                            <Play size={13} className="text-emerald-600" />
                            <span>Activate Reward</span>
                          </>
                        )}
                      </button>

                      <div className="flex items-center gap-2">
                        <button
                          type="button"
                          onClick={() => openEditModal(s)}
                          className="btn btn-secondary text-[12px] py-1.5 px-3 font-bold"
                          title="Edit reward details"
                        >
                          <Edit2 size={13} />
                          <span>Edit</span>
                        </button>

                        {s.id && (
                          <button
                            type="button"
                            onClick={() => {
                              if (confirm('Are you sure you want to delete this quest reward proposal?')) {
                                deleteMutation.mutate(s.id!);
                              }
                            }}
                            disabled={deleteMutation.isPending}
                            className="btn btn-danger-outline text-[12px] py-1.5 px-3 font-bold"
                            title="Delete this reward proposal"
                          >
                            <Trash2 size={13} />
                            <span>Delete</span>
                          </button>
                        )}
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>

        {/* Quest Schedules & History */}
        <div className="card p-6 space-y-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="w-9 h-9 rounded-xl bg-accent/15 text-accent flex items-center justify-center">
                <Calendar size={18} />
              </div>
              <div>
                <h2 className="font-extrabold text-[16px] text-ink">Sponsorship Schedule & History</h2>
                <p className="text-[12.5px] text-muted font-semibold">
                  Dates your business has been chosen as the featured Daily Quest destination
                </p>
              </div>
            </div>

            <span className="text-[12px] font-extrabold text-muted">
              {schedules.length} {schedules.length === 1 ? 'date' : 'dates'} scheduled
            </span>
          </div>

          {schedulesLoading ? (
            <p className="text-muted font-bold text-[13px] py-4">Loading quest schedules...</p>
          ) : schedules.length === 0 ? (
            <div className="p-6 rounded-xl bg-sand/30 border border-line text-center space-y-1">
              <p className="font-extrabold text-[14px] text-ink">No scheduled dates yet</p>
              <p className="text-[12.5px] text-muted font-semibold max-w-md mx-auto">
                Once an offered reward is approved, Spotto Command Center admins will schedule dates for your business. They will appear right here.
              </p>
            </div>
          ) : (
            <div className="divide-y divide-line border border-line rounded-xl overflow-hidden">
              {schedules.map((s) => {
                const isToday = s.date === todayStr;
                const isUpcoming = s.date >= todayStr;
                const [y, m, d] = s.date.split('-');
                const dateObj = new Date(parseInt(y, 10), parseInt(m, 10) - 1, parseInt(d, 10));
                return (
                  <div
                    key={s.date}
                    className={`p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3 bg-white hover:bg-sand/20 transition-colors ${
                      isToday ? 'bg-accent/[0.04] border-l-4 border-l-accent' : ''
                    }`}
                  >
                    <div className="flex items-center gap-3.5">
                      <div
                        className={`w-12 h-12 rounded-xl flex flex-col items-center justify-center shrink-0 border ${
                          isToday
                            ? 'bg-accent text-white border-accent shadow-sm'
                            : isUpcoming
                            ? 'bg-accent/10 border-accent/20 text-accent'
                            : 'bg-sand border-line text-ink'
                        }`}
                      >
                        <span className="text-[10px] font-extrabold uppercase">
                          {dateObj.toLocaleString('default', { month: 'short' })}
                        </span>
                        <span className="text-[16px] font-black leading-none">{dateObj.getDate()}</span>
                      </div>

                      <div>
                        <div className="flex items-center gap-2">
                          <span className="font-extrabold text-[14px] text-ink">{s.date}</span>
                          {isToday ? (
                            <span className="badge bg-accent text-white shadow-xs text-[10px] font-black">
                              TODAY
                            </span>
                          ) : isUpcoming ? (
                            <span className="badge bg-accent/15 text-accent border border-accent/20">
                              Upcoming
                            </span>
                          ) : (
                            <span className="badge bg-sand text-muted border border-line">
                              Completed Day
                            </span>
                          )}
                        </div>
                        <p className="text-[12.5px] text-muted font-semibold mt-0.5">
                          Featured Perk: <span className="text-ink font-bold">{s.rewardTitle ? `${s.rewardTitle} — ` : ''}{s.rewardDescription || 'Quest Reward'}</span> (Goal: {s.goal} places)
                        </p>
                      </div>
                    </div>

                    <div className="flex items-center gap-6 sm:text-right shrink-0 border-t sm:border-t-0 pt-2 sm:pt-0 border-line">
                      <div>
                        <p className="text-[10.5px] font-extrabold text-muted uppercase">Finished Quests</p>
                        <p className="text-[14px] font-extrabold text-ink">{s.completions}</p>
                      </div>
                      <div>
                        <p className="text-[10.5px] font-extrabold text-muted uppercase">Redeemed</p>
                        <p className="text-[14px] font-extrabold text-emerald-600">{s.redemptions}</p>
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>

        {/* Live Completions Feed */}
        <QuestCompletions />

        {/* Create / Edit Reward Modal */}
        {modalOpen && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-ink/40 backdrop-blur-xs animate-in fade-in duration-150">
            <div className="bg-white border border-line rounded-2xl w-full max-w-lg shadow-2xl overflow-hidden animate-in zoom-in-95 duration-150">
              <div className="p-5 bg-sand/20 border-b border-line flex items-center justify-between">
                <div className="flex items-center gap-2.5">
                  <div className="w-8 h-8 rounded-lg bg-accent/15 text-accent flex items-center justify-center">
                    <Gift size={16} />
                  </div>
                  <h3 className="font-extrabold text-[16px] text-ink">
                    {editingItem ? 'Edit Quest Reward' : 'Propose New Quest Reward'}
                  </h3>
                </div>
                <button
                  type="button"
                  onClick={() => setModalOpen(false)}
                  className="p-1.5 rounded-lg text-muted hover:text-ink hover:bg-sand transition-colors"
                  title="Close modal"
                >
                  <X size={18} />
                </button>
              </div>

              <div className="p-6 space-y-4.5">
                {/* Reward Short Title */}
                <div className="space-y-1.5">
                  <div className="flex items-center justify-between">
                    <label className="text-[12.5px] font-extrabold text-ink">Reward Title (Optional)</label>
                    <span className="text-[11px] text-muted font-bold">e.g. Free Pot of Tea & Cake</span>
                  </div>
                  <input
                    className="input w-full text-[13.5px]"
                    placeholder="e.g. Pot of Ceylon Tea & Homemade Cake"
                    value={formTitle}
                    onChange={(e) => setFormTitle(e.target.value)}
                  />
                </div>

                {/* Reward Description */}
                <div className="space-y-1.5">
                  <div className="flex items-center justify-between">
                    <label className="text-[12.5px] font-extrabold text-ink">
                      Reward Description <span className="text-coral">*</span>
                    </label>
                    <span className="text-[11px] text-muted font-bold">Clear complimentary perk</span>
                  </div>
                  <textarea
                    rows={3}
                    className="input w-full text-[13.5px] py-2 leading-relaxed resize-none"
                    placeholder="e.g. Free pot of premium Ceylon black tea and a slice of chocolate cake upon presenting verified quest completion code."
                    value={formDesc}
                    onChange={(e) => setFormDesc(e.target.value)}
                  />
                  <p className="text-[11.5px] text-muted font-medium">
                    What will completed travellers receive when visiting your counter?
                  </p>
                </div>

                {/* Daily Redemption Limit */}
                <div className="space-y-1.5">
                  <div className="flex items-center justify-between">
                    <label className="text-[12.5px] font-extrabold text-ink">Daily Redemption Limit</label>
                    <span className="text-[11px] text-muted font-bold">0 = Unlimited</span>
                  </div>
                  <input
                    type="number"
                    min="0"
                    className="input w-full text-[13.5px]"
                    value={formLimit}
                    onChange={(e) => setFormLimit(parseInt(e.target.value, 10) || 0)}
                  />
                  <p className="text-[11.5px] text-muted font-medium">
                    Maximum number of rewards you wish to redeem per scheduled day.
                  </p>
                </div>

                {/* Active for Scheduling Switch */}
                <label className="flex items-center gap-3 p-3.5 rounded-xl bg-sand/40 border border-line/80 cursor-pointer hover:bg-sand/60 transition-colors">
                  <input
                    type="checkbox"
                    checked={formActive}
                    onChange={(e) => setFormActive(e.target.checked)}
                    className="w-4 h-4 accent-accent cursor-pointer rounded"
                  />
                  <div>
                    <p className="text-[13px] font-extrabold text-ink">Active for Daily Quest Scheduling</p>
                    <p className="text-[11.5px] text-muted font-medium">
                      Keep enabled so Spotto Command Center admins can schedule this reward on featured days.
                    </p>
                  </div>
                </label>

                {/* Review Notice */}
                <div className="p-3 rounded-xl bg-amber-500/10 border border-amber-500/25 text-amber-800 text-[12px] font-semibold flex items-start gap-2">
                  <Clock size={14} className="shrink-0 mt-0.5 text-amber-600" />
                  <span>
                    New proposals or changes to reward description are reviewed by the Spotto team to ensure guest safety and quality standards before going live.
                  </span>
                </div>
              </div>

              {/* Modal Footer */}
              <div className="p-4 bg-sand/30 border-t border-line flex items-center justify-end gap-2.5">
                <button
                  type="button"
                  onClick={() => setModalOpen(false)}
                  className="btn btn-secondary text-[13px] px-4 py-2 font-bold"
                >
                  Cancel
                </button>
                <button
                  type="button"
                  onClick={() => saveMutation.mutate()}
                  disabled={saveMutation.isPending || !formDesc.trim()}
                  className="btn btn-accent text-[13px] px-5 py-2 rounded-xl flex items-center gap-2 font-extrabold shadow-sm"
                >
                  <Save size={15} />
                  <span>{saveMutation.isPending ? 'Saving...' : editingItem ? 'Save Changes' : 'Submit Proposal'}</span>
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </Shell>
  );
}

// -------------------------------------------------------------
// Live completions list for this vendor
// -------------------------------------------------------------
function QuestCompletions() {
  const { data = [], isLoading, isError } = useQuery<QuestCompletion[]>({
    queryKey: ['vendor-quest-completions'],
    queryFn: () => api.get<QuestCompletion[]>('/vendor/quest/completions?days=30'),
    refetchInterval: 60000,
  });
  const redeemed = data.filter((c) => c.redeemedAt).length;

  return (
    <div className="card p-6 space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="w-9 h-9 rounded-xl bg-accent/15 text-accent flex items-center justify-center">
            <Users size={18} />
          </div>
          <div>
            <h2 className="font-extrabold text-[16px] text-ink">Recent Quest Redemptions</h2>
            <p className="text-[12.5px] text-muted font-semibold">
              Travelers who completed the Daily Quest on your sponsored days in the last 30 days
            </p>
          </div>
        </div>

        {data.length > 0 && (
          <div className="text-[12px] font-extrabold text-muted">
            <span className="text-ink">{data.length}</span> finished ·{' '}
            <span className="text-emerald-600">{redeemed}</span> claimed
          </div>
        )}
      </div>

      {isError && (
        <div className="p-3 rounded-xl bg-coral/10 border border-coral/20 text-coral text-[13px] font-bold">
          Couldn&apos;t load quest completions. Try refreshing.
        </div>
      )}

      {isLoading && <p className="text-muted font-bold text-[13px] py-4">Loading completions...</p>}

      {!isLoading && !isError && data.length === 0 && (
        <div className="p-6 rounded-xl bg-sand/30 border border-line text-center space-y-1">
          <p className="font-extrabold text-[14px] text-ink">No completions recorded yet</p>
          <p className="text-[12.5px] text-muted font-semibold">
            When travelers finish their quest and visit your venue on scheduled days, they will be listed here.
          </p>
        </div>
      )}

      {data.length > 0 && (
        <div className="overflow-x-auto border border-line rounded-xl">
          <table className="w-full text-[13px] min-w-[500px]">
            <thead>
              <tr className="text-muted text-left text-[11px] font-extrabold uppercase tracking-wide bg-sand/40 border-b border-line">
                <th className="py-2.5 px-4">Traveller</th>
                <th className="py-2.5 px-4">Quest Date</th>
                <th className="py-2.5 px-4">Completed</th>
                <th className="py-2.5 px-4 text-right">Counter Redemption</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-line">
              {data.map((c) => (
                <tr key={`${c.travellerId}-${c.date}`} className="hover:bg-sand/20 transition-colors">
                  <td className="py-3 px-4 font-extrabold text-ink">{c.travellerName}</td>
                  <td className="py-3 px-4 font-semibold tabular-nums text-muted">{c.date}</td>
                  <td className="py-3 px-4 text-muted font-semibold" title={new Date(c.completedAt).toLocaleString()}>
                    {timeAgo(c.completedAt)}
                  </td>
                  <td className="py-3 px-4 text-right">
                    {c.redeemedAt ? (
                      <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-[11px] font-extrabold bg-emerald-500/15 text-emerald-700">
                        <CheckCircle2 size={12} /> Claimed
                      </span>
                    ) : (
                      <span className="inline-flex px-2 py-0.5 rounded-full text-[11px] font-bold bg-sand text-muted">
                        Not Yet Redeemed
                      </span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
