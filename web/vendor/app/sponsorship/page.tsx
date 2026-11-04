'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  AlertCircle,
  Award,
  Calendar,
  CheckCircle2,
  Clock,
  ExternalLink,
  Gift,
  HelpCircle,
  Info,
  MapPin,
  Save,
  Sparkles,
  Users,
  XCircle,
} from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import { timeAgo, type QuestCompletion, type QuestSchedule, type VendorSponsorship } from '@spotto/shared';
import Shell from '../../components/shell';
import { api } from '../../lib/api';

export default function SponsorshipPage() {
  const qc = useQueryClient();

  const { data: spons, isLoading } = useQuery<VendorSponsorship>({
    queryKey: ['sponsorship'],
    queryFn: () => api.get<VendorSponsorship>('/vendor/sponsorship'),
  });

  const { data: schedules = [], isLoading: schedulesLoading } = useQuery<QuestSchedule[]>({
    queryKey: ['vendor-quest-schedules'],
    queryFn: () => api.get<QuestSchedule[]>('/vendor/quest/schedules'),
  });

  const [desc, setDesc] = useState('');
  const [limit, setLimit] = useState(0);
  const [active, setActive] = useState(false);
  const [saveSuccessMsg, setSaveSuccessMsg] = useState<string | null>(null);

  useEffect(() => {
    if (spons) {
      setDesc(spons.rewardDescription || '');
      setLimit(spons.rewardLimit || 0);
      setActive(spons.isActive ?? false);
    }
  }, [spons]);

  const save = useMutation({
    mutationFn: () =>
      api.put<VendorSponsorship>('/vendor/sponsorship', {
        rewardDescription: desc,
        rewardLimit: limit,
        isActive: active,
      }),
    onSuccess: (data) => {
      qc.invalidateQueries({ queryKey: ['sponsorship'] });
      setSaveSuccessMsg(
        data.status === 'pending'
          ? 'Reward saved! Proposal submitted for admin review.'
          : 'Sponsorship configuration updated.'
      );
      setTimeout(() => setSaveSuccessMsg(null), 5000);
    },
  });

  // Calculate upcoming scheduled date
  const todayStr = new Date().toISOString().slice(0, 10);
  const upcomingSchedule = useMemo(() => {
    return schedules
      .filter((s) => s.date >= todayStr)
      .sort((a, b) => a.date.localeCompare(b.date))[0];
  }, [schedules, todayStr]);

  const isPending = spons?.status === 'pending';
  const isApproved = spons?.status === 'approved';
  const isRejected = spons?.status === 'rejected';
  const isInactive = !active && spons?.status !== 'none';

  return (
    <Shell>
      <div className="max-w-4xl space-y-6 pb-16">
        {/* Page Header */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <div className="flex items-center gap-2.5">
              <h1 className="text-2xl font-extrabold font-display text-ink">Daily Quest Sponsorship</h1>
              {/* Dynamic Status Badge (Mutually Exclusive) */}
              {!active && spons?.status !== 'none' ? (
                <span className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-[12px] font-bold bg-sand text-muted border border-line">
                  Sponsorship Paused (Inactive)
                </span>
              ) : isPending ? (
                <span className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-[12px] font-extrabold bg-amber-500/15 text-amber-600 border border-amber-500/20">
                  <Clock size={13} /> Pending Admin Review
                </span>
              ) : isApproved && upcomingSchedule ? (
                <span className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-[12px] font-extrabold bg-accent text-white shadow-sm">
                  <Calendar size={13} /> Scheduled for {upcomingSchedule.date}
                </span>
              ) : isApproved && !upcomingSchedule ? (
                <span className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-[12px] font-extrabold bg-emerald-500/15 text-emerald-700 border border-emerald-500/20">
                  <CheckCircle2 size={13} /> Approved & Ready for Scheduling
                </span>
              ) : isRejected ? (
                <span className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-[12px] font-extrabold bg-coral/15 text-coral border border-coral/20">
                  <XCircle size={13} /> Edits Requested
                </span>
              ) : null}
            </div>
            <p className="text-muted font-semibold text-[13.5px] mt-1 max-w-2xl">
              Partner with Spotto to sponsor the Daily Quest. When travelers complete today&apos;s exploration challenge, they are directed to your venue to collect their reward.
            </p>
          </div>
        </div>

        {/* How it Works / Lifecycle Explanation Banner */}
        <div className="card p-5 bg-sand/40 border border-line">
          <div className="flex items-center gap-2 mb-3">
            <Sparkles size={16} className="text-accent" />
            <h3 className="font-extrabold text-[14.5px] text-ink">How Daily Quest Sponsorship Works</h3>
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-4 gap-3">
            <div className="bg-white p-3.5 rounded-xl border border-line/70">
              <span className="w-6 h-6 rounded-full bg-accent/15 text-accent text-[12px] font-black flex items-center justify-center mb-2">1</span>
              <p className="font-extrabold text-[13px] text-ink">Set Your Perk</p>
              <p className="text-[11.5px] text-muted font-semibold mt-0.5 leading-snug">
                Propose a small complimentary gift (e.g. Free Tea, Snack, or Dessert).
              </p>
            </div>

            <div className="bg-white p-3.5 rounded-xl border border-line/70">
              <span className="w-6 h-6 rounded-full bg-accent/15 text-accent text-[12px] font-black flex items-center justify-center mb-2">2</span>
              <p className="font-extrabold text-[13px] text-ink">Admin Review</p>
              <p className="text-[11.5px] text-muted font-semibold mt-0.5 leading-snug">
                Spotto Command Center reviews and approves your proposed perk.
              </p>
            </div>

            <div className="bg-white p-3.5 rounded-xl border border-line/70">
              <span className="w-6 h-6 rounded-full bg-accent/15 text-accent text-[12px] font-black flex items-center justify-center mb-2">3</span>
              <p className="font-extrabold text-[13px] text-ink">Featured Date</p>
              <p className="text-[11.5px] text-muted font-semibold mt-0.5 leading-snug">
                Your venue becomes the featured destination for all active travelers that day.
              </p>
            </div>

            <div className="bg-white p-3.5 rounded-xl border border-line/70">
              <span className="w-6 h-6 rounded-full bg-accent/15 text-accent text-[12px] font-black flex items-center justify-center mb-2">4</span>
              <p className="font-extrabold text-[13px] text-ink">Redeem & Greet</p>
              <p className="text-[11.5px] text-muted font-semibold mt-0.5 leading-snug">
                Travelers show their in-app code. You scan or confirm it to welcome them.
              </p>
            </div>
          </div>
        </div>

        {/* Configuration Form Card */}
        {isLoading ? (
          <div className="card p-8 text-center text-muted font-bold">Loading sponsorship details...</div>
        ) : (
          <div className="card p-6 space-y-5">
            <div className="flex items-center justify-between border-b border-line pb-4">
              <div className="flex items-center gap-3">
                <div className="w-9 h-9 rounded-xl bg-accent/15 text-accent flex items-center justify-center">
                  <Gift size={18} />
                </div>
                <div>
                  <h2 className="font-extrabold text-[16px] text-ink">Configure Sponsorship Reward</h2>
                  <p className="text-[12.5px] text-muted font-semibold">Customize what completed quest travellers will receive</p>
                </div>
              </div>

              {/* Status Explanation Pill */}
              {isPending && (
                <div className="flex items-center gap-1.5 text-[12px] text-amber-600 font-extrabold bg-amber-500/10 px-3 py-1 rounded-full border border-amber-500/20">
                  <Clock size={13} /> Changes will be reviewed by admin
                </div>
              )}
            </div>

            {/* Active Switch */}
            <label className="flex items-center gap-3 bg-sand p-4 rounded-xl cursor-pointer transition-colors hover:bg-sand/80">
              <input
                type="checkbox"
                checked={active}
                onChange={(e) => setActive(e.target.checked)}
                className="w-5 h-5 accent-accent cursor-pointer"
              />
              <div className="flex-1">
                <p className="font-extrabold text-[14px] text-ink">Active Quest Sponsorship</p>
                <p className="text-[12px] text-muted font-semibold mt-0.5">
                  Keep this turned on so Spotto admins can feature your venue on daily quest schedules.
                </p>
              </div>
            </label>

            {/* Status Information Callout */}
            {isRejected && (
              <div className="p-4 rounded-xl bg-coral/10 border border-coral/20 text-coral text-[13px] font-semibold space-y-1">
                <div className="flex items-center gap-2 font-extrabold">
                  <AlertCircle size={16} />
                  <span>Sponsorship Proposal Needs Revision</span>
                </div>
                <p className="text-[12px] leading-relaxed">
                  Your previous reward was not approved. Please adjust the reward description below to a clear, complimentary perk and save to submit for re-approval.
                </p>
              </div>
            )}

            {isApproved && (
              <div className="p-4 rounded-xl bg-emerald-500/10 border border-emerald-500/20 text-emerald-800 text-[13px] font-semibold space-y-1">
                <div className="flex items-center gap-2 font-extrabold text-emerald-700">
                  <CheckCircle2 size={16} />
                  <span>Your Sponsorship is Approved</span>
                </div>
                <p className="text-[12px] text-emerald-700/90 leading-relaxed">
                  Your current perk is approved and ready. Changing the description will submit it for admin re-approval.
                </p>
              </div>
            )}

            {/* Reward Description */}
            <div className="space-y-1.5">
              <div className="flex items-center justify-between">
                <label className="text-[13px] font-extrabold text-ink">Reward Description</label>
                <span className="text-[11.5px] font-bold text-muted">Example: Free Ceylon Tea & Cake</span>
              </div>
              <input
                className="input w-full"
                placeholder="e.g. Free Pot of Ceylon Tea & Cake"
                value={desc}
                onChange={(e) => setDesc(e.target.value)}
              />
              <p className="text-[12px] text-muted font-semibold">
                What will the traveler receive upon presenting their verified completion code at your counter?
              </p>
            </div>

            {/* Daily Limit */}
            <div className="space-y-1.5">
              <div className="flex items-center justify-between">
                <label className="text-[13px] font-extrabold text-ink">Daily Redemption Limit</label>
                <span className="text-[11.5px] font-bold text-muted">0 = Unlimited</span>
              </div>
              <input
                className="input w-full"
                type="number"
                min="0"
                value={limit}
                onChange={(e) => setLimit(parseInt(e.target.value, 10) || 0)}
              />
              <p className="text-[12px] text-muted font-semibold">
                Maximum number of quest rewards you wish to redeem per scheduled day. Set to 0 if you have no limit.
              </p>
            </div>

            {/* Submit Button */}
            <button
              onClick={() => save.mutate()}
              disabled={save.isPending || !desc.trim()}
              className="btn-accent w-full py-3 mt-4 flex items-center justify-center gap-2 shadow-sm font-extrabold text-[14px]"
            >
              <Save size={18} />
              {save.isPending ? 'Saving Proposal...' : 'Save & Submit Sponsorship'}
            </button>

            {saveSuccessMsg && (
              <div className="p-3 rounded-xl bg-emerald-500/10 border border-emerald-500/20 text-emerald-700 text-center text-[13px] font-bold">
                {saveSuccessMsg}
              </div>
            )}
            {save.isError && (
              <div className="p-3 rounded-xl bg-coral/10 border border-coral/20 text-coral text-center text-[13px] font-bold">
                Failed to save. Please verify all fields and try again.
              </div>
            )}
          </div>
        )}

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
                Once your sponsorship proposal is approved, Spotto Command Center admins will schedule dates for your business. They will appear right here.
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
                            <span className="px-2 py-0.5 rounded-full text-[10px] font-black bg-accent text-white shadow-sm">
                              TODAY
                            </span>
                          ) : isUpcoming ? (
                            <span className="px-2 py-0.5 rounded-full text-[10.5px] font-extrabold bg-accent/15 text-accent">
                              Upcoming
                            </span>
                          ) : (
                            <span className="px-2 py-0.5 rounded-full text-[10.5px] font-extrabold bg-sand text-muted">
                              Completed Day
                            </span>
                          )}
                        </div>
                        <p className="text-[12.5px] text-muted font-semibold mt-0.5">
                          Reward: <span className="text-ink font-bold">{s.rewardDescription || desc}</span> (Goal: {s.goal} places)
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
