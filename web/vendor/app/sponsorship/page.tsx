'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { CheckCircle2, Gift, Save, Users } from 'lucide-react';
import { useEffect, useState } from 'react';
import { timeAgo, type QuestCompletion } from '@spotto/shared';
import Shell from '../../components/shell';
import { api } from '../../lib/api';

type Sponsorship = {
  vendorId: string;
  rewardDescription: string;
  rewardLimit: number;
  isActive: boolean;
};

export default function SponsorshipPage() {
  const qc = useQueryClient();
  const { data: spons, isLoading } = useQuery<Sponsorship>({
    queryKey: ['sponsorship'],
    queryFn: () => api.get<Sponsorship>('/vendor/sponsorship'),
  });

  const [desc, setDesc] = useState('');
  const [limit, setLimit] = useState(0);
  const [active, setActive] = useState(false);

  useEffect(() => {
    if (spons) {
      setDesc(spons.rewardDescription || '');
      setLimit(spons.rewardLimit || 0);
      setActive(spons.isActive || false);
    }
  }, [spons]);

  const save = useMutation({
    mutationFn: () => api.put<Sponsorship>('/vendor/sponsorship', {
      rewardDescription: desc,
      rewardLimit: limit,
      isActive: active,
    }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['sponsorship'] });
    },
  });

  return (
    <Shell>
      <div className="mb-6">
        <h1 className="text-2xl font-extrabold font-display">Daily Quest Sponsorship</h1>
        <p className="text-muted font-semibold mt-0.5">
          Opt-in to sponsor the Daily Quest. Travelers who complete the quest will be directed to your business for their reward.
        </p>
      </div>

      {isLoading ? (
        <p className="text-muted font-bold">Loading...</p>
      ) : (
        <div className="card p-6 space-y-5 max-w-xl">
          <div className="flex items-center gap-3">
            <Gift className="text-accent" />
            <h2 className="font-extrabold text-[16px]">Configure Reward</h2>
          </div>
          
          <label className="flex items-center gap-3 bg-sand p-4 rounded-xl cursor-pointer">
            <input 
              type="checkbox" 
              checked={active}
              onChange={(e) => setActive(e.target.checked)}
              className="w-5 h-5 accent-accent"
            />
            <div>
              <p className="font-extrabold text-[14px]">Active Sponsorship</p>
              <p className="text-[12px] text-muted font-semibold mt-0.5">Turn this on to appear as a sponsor in the app.</p>
            </div>
          </label>

          <div className="space-y-1.5">
            <label className="text-[13px] font-extrabold">Reward Description</label>
            <input 
              className="input w-full"
              placeholder="e.g. Free Ceylon Tea"
              value={desc}
              onChange={(e) => setDesc(e.target.value)}
            />
            <p className="text-[12px] text-muted font-semibold">What will the traveler receive upon showing you the completion code?</p>
          </div>

          <div className="space-y-1.5">
            <label className="text-[13px] font-extrabold">Daily Limit</label>
            <input 
              className="input w-full"
              type="number"
              min="0"
              value={limit}
              onChange={(e) => setLimit(parseInt(e.target.value) || 0)}
            />
            <p className="text-[12px] text-muted font-semibold">Maximum number of redemptions per day. Set to 0 for unlimited.</p>
          </div>

          <button 
            onClick={() => save.mutate()}
            disabled={save.isPending}
            className="btn-accent w-full py-3 mt-4 flex items-center justify-center gap-2"
          >
            <Save size={18} />
            {save.isPending ? 'Saving...' : 'Save Configuration'}
          </button>
          
          {save.isSuccess && <p className="text-sun font-bold text-[13px] text-center mt-3">Saved successfully!</p>}
          {save.isError && <p className="text-coral font-bold text-[13px] text-center mt-3">Failed to save.</p>}
        </div>
      )}

      <QuestCompletions />
    </Shell>
  );
}

// Travellers who finished the quest on days this business sponsored it,
// and whether they've collected their reward yet.
function QuestCompletions() {
  const { data = [], isLoading, isError } = useQuery<QuestCompletion[]>({
    queryKey: ['vendor-quest-completions'],
    queryFn: () => api.get<QuestCompletion[]>('/vendor/quest/completions?days=30'),
    refetchInterval: 60000,
  });
  const redeemed = data.filter((c) => c.redeemedAt).length;

  return (
    <div className="card p-6 max-w-3xl mt-6">
      <div className="flex items-center gap-3 mb-1">
        <Users className="text-accent" />
        <h2 className="font-extrabold text-[16px]">Quest completions</h2>
      </div>
      <p className="text-[12.5px] text-muted font-semibold mb-4">
        Last 30 days, on days you sponsored the quest
        {data.length > 0 && ` · ${data.length} completed, ${redeemed} redeemed`}
      </p>
      {isError && <p className="text-coral font-bold text-[13px]">Couldn&apos;t load quest completions. Try refreshing.</p>}
      {isLoading && <p className="text-muted font-bold text-[13px]">Loading...</p>}
      {!isLoading && !isError && data.length === 0 && (
        <p className="text-muted font-semibold text-[13px]">No one has completed one of your sponsored quests yet.</p>
      )}
      {data.length > 0 && (
        <div className="overflow-x-auto">
          <table className="w-full text-[13px] min-w-[480px]">
            <thead>
              <tr className="text-muted text-left text-[11px] uppercase tracking-wide">
                <th className="font-extrabold pb-2 pr-3">Traveller</th>
                <th className="font-extrabold pb-2 pr-3">Day</th>
                <th className="font-extrabold pb-2 pr-3">Completed</th>
                <th className="font-extrabold pb-2 text-right">Reward</th>
              </tr>
            </thead>
            <tbody>
              {data.map((c) => (
                <tr key={`${c.travellerId}-${c.date}`} className="border-t border-line">
                  <td className="py-2.5 pr-3 font-extrabold">{c.travellerName}</td>
                  <td className="py-2.5 pr-3 font-semibold tabular-nums">{c.date}</td>
                  <td className="py-2.5 pr-3 text-muted font-semibold" title={new Date(c.completedAt).toLocaleString()}>{timeAgo(c.completedAt)}</td>
                  <td className="py-2.5 text-right">
                    {c.redeemedAt ? (
                      <span className="inline-flex items-center gap-1 text-[12px] font-extrabold text-accent">
                        <CheckCircle2 size={13} /> Redeemed
                      </span>
                    ) : (
                      <span className="text-[12px] font-bold text-muted">Not yet</span>
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
