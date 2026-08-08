'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Gift, Save } from 'lucide-react';
import { useEffect, useState } from 'react';
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
    </Shell>
  );
}
